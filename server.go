//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/binary"
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
	"unicode/utf8"

	"chichi/pkg/open2b/sql"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/mssola/user_agent"
	"github.com/open2b/nuts"
	"github.com/open2b/nuts/culture"
	"github.com/open2b/nuts/ipset"
	"github.com/oschwald/geoip2-golang"
	"golang.org/x/text/unicode/norm"
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
	err := json.NewDecoder(norm.NFC.Reader(r.Body)).Decode(&event)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Validate the property and verify it exists.
	if !isValidPropertyID(event.Property) {
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

	// Get the user or create it if it does not exist.
	var user uint64
	err = server.mySQLDB.QueryRow("SELECT `id` FROM `users` WHERE `property` = ? AND `device` = ?", event.Property, event.Device).Scan(&user)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] queriyng `users`: %s", err)
		return
	}
	if err == sql.ErrNoRows {
		err = server.mySQLDB.QueryRow("SELECT `user` FROM `devices` WHERE `property` = ? AND `id` = ?", event.Property, event.Device).Scan(&user)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("[error] queriyng `devices`: %s", err)
			return
		}
		if err == sql.ErrNoRows {
			user, err = makeUserID()
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				log.Printf("[error] cannot generate a random user id: %s", err)
				return
			}
			_, err = server.mySQLDB.Exec("INSERT INTO `users` SET `property` = ?, `id` = ?, `device` = ?", event.Property, user, event.Device)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				log.Printf("[error] cannot add a new user: %s", err)
				return
			}
		}
	}
	event.User = user

	// Validate the event.
	locale := culture.Locale(event.Language)
	if locale == nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	event.Language = locale.LanguageCode()
	if _, err := url.Parse(event.URL); err != nil || utf8.RuneCountInString(event.URL) > 2048 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if _, err := url.Parse(event.Referrer); err != nil || utf8.RuneCountInString(event.Referrer) > 2048 {
		event.Referrer = ""
	}
	if _, err := url.Parse(event.Target); err != nil || utf8.RuneCountInString(event.Target) > 2048 {
		event.Target = ""
	}
	if event.Event != "visit" && event.Event != "click" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(event.Text) > 120 {
		event.Text = nuts.Truncate(event.Text, 120)
		return
	}
	if utf8.RuneCountInString(event.Title) > 120 {
		event.Title = nuts.Truncate(event.Title, 120)
		return
	}
	if _, err := base64.StdEncoding.DecodeString(event.Device); err != nil || len(event.Device) != 28 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
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
		if ip := requestIP.String(); ip == "127.0.0.1" || ip == "::1" {
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

	// Enrich the event with country and city.
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

	// Enrich the event with user agent info.
	ua := user_agent.New(r.Header.Get("User-Agent"))
	osInfo := ua.OSInfo()
	switch osInfo.Name {
	case "Android", "Windows", "iOS", "MacOS", "Linux", "ChromeOS":
		event.OSName = osInfo.Name
	default:
		event.OSName = "Other"
	}
	if utf8.RuneCountInString(event.OSVersion) <= 10 {
		event.OSVersion = osInfo.Version
	}
	if name, version := ua.Browser(); utf8.RuneCountInString(name) <= 20 {
		event.BrowserName = name
		if utf8.RuneCountInString(version) <= 20 {
			event.BrowserVersion = version
		}
	} else {
		event.BrowserName = "Other"
	}
	event.DeviceType = "desktop"
	if ua.Mobile() {
		if h := strings.ToLower(r.Header.Get("User-Agent")); strings.Contains(h, "ipad") || strings.Contains(h, "tablet") {
			event.DeviceType = "tablet"
		} else {
			event.DeviceType = "mobile"
		}
	}

	// Enrich the event with the domain, path and query string.
	event.Domain = url.Host
	event.Path = strings.TrimLeft(strings.TrimRight(url.Path, "/"), "/")
	event.QueryString = url.RawQuery

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
			"(`property`, `timestamp`, `language`, `osName`, `osVersion`, `browserName`, `browserVersion`, `deviceType`, "+
			"`referrer`, `target`, `event`, `text`, `domain`, `path`, `queryString`, `title`, `user`, `country`, `city`)")
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		for _, event := range events {
			err := batch.Append(event.Property, event.Timestamp, event.Language, event.OSName, event.OSVersion,
				event.BrowserName, event.BrowserVersion, event.DeviceType,
				event.Referrer, event.Target, event.Event, event.Text, event.Domain,
				event.Path, event.QueryString, event.Title, event.User, event.Country, event.City)
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

// isValidPropertyID reports whether id is a valid property identifier.
func isValidPropertyID(id string) bool {
	if len(id) != 10 || id == "0000000000" {
		return false
	}
	for i := 0; i < 10; i++ {
		var c = id[i]
		if c < '0' || ('9' < c && c < 'A') || 'Z' < c {
			return false
		}
	}
	return true
}

// makeUserID returns a new random user identifier.
func makeUserID() (uint64, error) {
	b := make([]byte, 8)
	_, err := crand.Read(b)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}
