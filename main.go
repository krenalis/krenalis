//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
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

	// Run the server.
	server := newServer(db)
	http.HandleFunc("/admin/src/", server.serveWithESBuild)
	http.HandleFunc("/log-event", server.serveLogEvent)
	http.HandleFunc("/run-query", server.serveRunQuery)
	http.Handle("/", http.FileServer(http.Dir("./")))
	err = http.ListenAndServeTLS(":9090", "cert.pem", "key.pem", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

type Event struct {
	Browser   string
	Event     string // "visit", "click", ...
	Language  string // "it-IT"
	Referrer  string // "https://example.com"
	Session   string
	Target    string // "https://example.com"
	Text      string // "Add to cart"
	Timestamp time.Time
	Title     string // "Product X"
	URL       string // "https://example.com"
}
