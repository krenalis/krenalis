//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/evanw/esbuild/pkg/api"
)

const maxEventsQueueLength = 10000
const flushQueueTimeout = 1 // Interval (in seconds) to flush the queue.

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
			"(`timestamp`, `language`, `browser`, `url`, `referrer`, `target`, `event`, `text`, `title`, `session`)")
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		for _, event := range events {
			err := batch.Append(event.Timestamp, event.Language, event.Browser, event.URL, event.Referrer, event.Target, event.Event, event.Text, event.Title, event.Session)
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

func (server *Server) serveRunQuery(w http.ResponseWriter, r *http.Request) {
	var query string
	err := json.NewDecoder(r.Body).Decode(&query)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	result, err := server.runQuery(query)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot log event: %s", err)
		return
	}
	_ = json.NewEncoder(w).Encode(result)
}

// runQuery runs the given query and returns its results as a [][]any.
func (server *Server) runQuery(query string) ([][]any, error) {
	rows, err := server.clickHouseConn.Query(server.clickHouseCtx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columnTypes := rows.ColumnTypes()
	columnsLen := len(columnTypes)
	result := [][]any{}
	for rows.Next() {
		sqlRow := make([]any, columnsLen)
		for j, column := range columnTypes {
			switch column.DatabaseTypeName() {
			case "DateTime":
				var value time.Time
				sqlRow[j] = &value
			case "String":
				var value string
				sqlRow[j] = &value
			case "UInt8":
				var value uint8
				sqlRow[j] = &value
			case "UInt64":
				var value uint64
				sqlRow[j] = &value
			default:
				panic(fmt.Sprintf("BUG: handling of database type %q not implemented", column.DatabaseTypeName()))
			}
		}
		err := rows.Scan(sqlRow...)
		if err != nil {
			return nil, err
		}
		row := make([]any, len(sqlRow))
		for i, pr := range sqlRow {
			switch v := pr.(type) {
			case interface{ Value() (driver.Value, error) }:
				value, err := v.Value()
				if err != nil {
					panic(err)
				}
				row[i] = value
			case *time.Time:
				row[i] = (*v).String()
			case *string:
				row[i] = *v
			case *uint8:
				row[i] = *v
			case *uint64:
				row[i] = *v
			default:
				panic("unexpected")
			}
		}
		result = append(result, row)
	}
	return result, nil
}

func (server *Server) serveWithESBuild(w http.ResponseWriter, r *http.Request) {
	file, err := filepath.Abs("admin/src/index.js")
	if err != nil {
		panic(err)
	}
	result := api.Build(api.BuildOptions{
		Bundle:            true,
		EntryPoints:       []string{file},
		Format:            api.FormatESModule,
		JSXMode:           api.JSXModeAutomatic,
		Loader:            map[string]api.Loader{".js": api.LoaderJSX},
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Outdir:            "out",
		Target:            api.ES2018,
		TreeShaking:       api.TreeShakingTrue,
		Write:             false,
	})

	// Handle errors and warnings.
	if result.Errors != nil {
		for _, msg := range result.Errors {
			log.Printf("[error] ESBuild error: %v", msg)
		}
		log.Printf("[error] errors while executing ESbuild, cannot serve %q", r.URL.Path)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if result.Warnings != nil {
		for _, msg := range result.Warnings {
			log.Printf("[warning] ESBuild warning: %v", msg)
			if server.settings.Main.PrintESBuildWarningsOnStderr {
				fmt.Fprintf(os.Stderr, "[warning] ESBuild warning: %v", msg)
			}
		}
	}

	base := path.Base(r.URL.Path)
	for _, out := range result.OutputFiles {
		if strings.HasSuffix(out.Path, base) {
			switch filepath.Ext(base) {
			case ".js":
				w.Header().Add("Content-Type", "text/javascript")
			case ".css":
				w.Header().Add("Content-Type", "text/css")
			default:
				log.Printf("[error] cannot determine Content-Type for %q", base)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			w.Write(out.Contents)
			return
		}
	}
	http.NotFound(w, r)
}
