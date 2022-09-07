//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"chichi/pkg/open2b/sql"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/open2b/nuts/ipset"
	"github.com/oschwald/geoip2-golang"
)

const (
	maxEventsQueueLength = 10000
	flushQueueTimeout    = 1 // Interval (in seconds) to flush the queue.
	geoLite2Path         = "GeoLite2-City.mmdb"
	testGEOIP            = "79.9.108.176"
)

type Server struct {
	settings         *Settings
	mySQLDB          *sql.DB
	clickHouseConn   chDriver.Conn
	clickHouseCtx    context.Context
	eventsQueue      []*Event
	eventsQueueMutex sync.Mutex
}

func newServer(settings *Settings, mySQLDB *sql.DB, clickHouseConn chDriver.Conn, clickHouseCtx context.Context) *Server {
	s := &Server{settings: settings, mySQLDB: mySQLDB, clickHouseConn: clickHouseConn, clickHouseCtx: clickHouseCtx}
	s.timeoutFlusher()
	return s
}

// serveLogEvent receives an event via HTTP and enqueues it.
func (server *Server) serveLogEvent(w http.ResponseWriter, r *http.Request) {
	var event *Event
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Check property existence.
	if event.Property == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	row := server.mySQLDB.QueryRow("SELECT `customer` FROM `properties` WHERE `id` = ?", event.Property)
	var customer string
	err = row.Scan(&customer)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] queriyng `properties`: %s", err)
		return
	}

	// Get the request IP.
	var requestIP net.IP
	{
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		requestIP = net.ParseIP(host)
		if requestIP == nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if requestIP.String() == "127.0.0.1" {
			requestIP = net.ParseIP(testGEOIP)
		}
	}

	// Check if the request IP is internal for the customer.
	var internalIPs string
	err = server.mySQLDB.QueryRow("SELECT `internalIPs` FROM `customers` WHERE `id` = ?", customer).Scan(&internalIPs)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] queriyng `customers`: %s", err)
		return
	}
	if internalIPs != "" {
		set := ipset.New()
		for _, ip := range strings.Fields(internalIPs) {
			set.Add(ip)
		}
		if set.Has(requestIP.String()) {
			return
		}
	}

	// Check if the event is from a domain enabled for the property.
	url, err := url.Parse(event.URL)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	hasDomain := map[string]bool{}
	{
		rows, err := server.mySQLDB.Query("SELECT `name` FROM `domains` WHERE `property` = ?", event.Property)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("[error] queriyng `domains`: %s", err)
			return
		}
		defer rows.Close()
		var domain string
		for rows.Next() {
			var err = rows.Scan(&domain)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				log.Printf("[error] cannot scan `domains`: %s", err)
				return
			}
			hasDomain[domain] = true
		}
		if err = rows.Err(); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("[error] cannot scan `domains`: %s", err)
			return
		}
	}

	if len(hasDomain) > 0 && !hasDomain[url.Host] {
		fmt.Printf("unexpected domain %q for property %q, discarding", url.Host, event.Property)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	geoDB, err := geoip2.Open(geoLite2Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("cannot read the %s database: %s", geoLite2Path, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if !errors.Is(err, fs.ErrNotExist) {
		defer geoDB.Close()
		city, err := geoDB.City(requestIP)
		if err != nil {
			fmt.Printf("cannot read the city from the %s database: %s", geoLite2Path, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		event.Country = city.Country.IsoCode
		event.City = city.City.Names["en"]
		geoDB.Close()
	}

	server.eventsQueueMutex.Lock()
	server.eventsQueue = append(server.eventsQueue, event)
	var toFlush []*Event
	if len(server.eventsQueue) == maxEventsQueueLength {
		toFlush = server.eventsQueue
		server.eventsQueue = nil
	}
	server.eventsQueueMutex.Unlock()
	if toFlush != nil {
		go server.flushEvents(toFlush)
	}
}

// timeoutFlusher launches a goroutine that flushes the events queue every
// flushQueueTimeout seconds
func (server *Server) timeoutFlusher() {
	ticker := time.NewTicker(flushQueueTimeout * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				server.eventsQueueMutex.Lock()
				toFlush := server.eventsQueue
				server.eventsQueue = nil
				server.eventsQueueMutex.Unlock()
				go server.flushEvents(toFlush)
			}
		}
	}()
}

// flushEvents writes a batch of events to ClickHouse.
func (server *Server) flushEvents(events []*Event) {
	if len(events) == 0 {
		return
	}
	log.Printf("[info] flushing %d events", len(events))
RETRY:
	for {
		batch, err := server.clickHouseConn.PrepareBatch(server.clickHouseCtx, "INSERT INTO `events`\n"+
			"(`property`, `timestamp`, `language`, `browser`, `url`, `referrer`, `target`, `event`, `text`, `title`, "+
			"`user`, `country`, `city`)")
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		for _, event := range events {
			err := batch.Append(event.Property, event.Timestamp, event.Language, event.Browser, event.URL,
				event.Referrer, event.Target, event.Event, event.Text, event.Title, event.User, event.Country, event.City)
			if err != nil {
				log.Printf("[error] cannot log events: %s", err)
				time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
				continue RETRY
			}
		}
		err = batch.Send()
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		return
	}
}
