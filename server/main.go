//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {

	// Configure the logger.
	logFile, err := os.OpenFile("error.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)

	// Read the configuration file.
	settings, err := parseINIFile()
	if err != nil {
		log.Printf("[error] cannot read configuration file: %s", err)
		return
	}

	// Open a connection to the database.
	db, err := sql.Open("mysql", settings.DB.Username+":"+settings.DB.Password+"@"+settings.DB.Address+"/"+settings.DB.Database+
		"?clientFoundRows=true&charset=utf8mb4,utf8&parseTime=true&allowOldPasswords=true")
	if err != nil {
		log.Printf("[error] cannot connect to the database: %v", err)
		return
	}
	defer db.Close()

	// Serve the request as CGI.
	err = cgi.Serve(newServer(db))
	if err != nil {
		log.Fatal(err)
	}

}

type Server struct {
	db *sql.DB
}

func newServer(db *sql.DB) *Server {
	return &Server{db: db}
}

// ServeHTTP serves the POST requests.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rpath := r.URL.Path
	rpath = strings.TrimSuffix(rpath, "/")
	switch {
	case strings.HasSuffix(rpath, "/log-event"):
		var event *Event
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		err = server.logEvent(event)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("[error] cannot log event: %s", err)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
}

// logEvent logs the given event on the database.
func (server *Server) logEvent(e *Event) error {
	query := "INSERT INTO `events`\n" +
		"(`timestamp`, `language`, `browser`, `url`, `referrer`, `target`, `event`)\n" +
		"VALUES\n" +
		"(?, ?, ?, ?, ?, ?, ?)"
	_, err := server.db.Exec(query, e.Timestamp, e.Language, e.Browser, e.URL, e.Referrer, e.Target, e.Event)
	return err
}

type Event struct {
	Timestamp time.Time
	Language  string // "it-IT"
	Browser   string
	URL       string // "https://example.com"
	Referrer  string // "https://example.com"
	Target    string // "https://example.com"
	Event     string // "visit", "click", ...
}
