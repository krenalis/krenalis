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
	"log"
	"net/http"
	"os"
	"time"
	_ "time/tzdata" // workaround for clickhouse-go issue #162

	"github.com/ClickHouse/clickhouse-go/v2"
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
	// Open a connection to the MySQL database.
	mySQLDB, err := sql.Open("mysql", settings.MySQL.Username+":"+settings.MySQL.Password+"@"+settings.MySQL.Address+"/"+settings.MySQL.Database+
		"?clientFoundRows=true&charset=utf8mb4,utf8&parseTime=true&allowOldPasswords=true")
	if err != nil {
		log.Fatalf("[error] cannot connect to the database: %s", err)
	}
	defer mySQLDB.Close()
	err = mySQLDB.Ping()
	if err != nil {
		log.Fatalf("[error] cannot ping MySQL server: %s", err)
	}
	log.Printf("[info] successfully connected to the MySQL server")

	// Open a connection to the ClickHouse database.
	clickHouseConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{settings.ClickHouse.Address},
		Auth: clickhouse.Auth{
			Database: settings.ClickHouse.Database,
			Username: settings.ClickHouse.Username,
			Password: settings.ClickHouse.Password,
		},
	})
	if err != nil {
		log.Fatalf("[error] cannot connect to the database: %s", err)
	}
	clickHouseCtx := context.Background()
	err = clickHouseConn.Ping(clickHouseCtx)
	if err != nil {
		log.Fatalf("[error] cannot ping ClickHouse server: %s", err)
	}
	log.Printf("[info] successfully connected to the ClickHouse server")

	// Run the server.
	server := newServer(mySQLDB, clickHouseConn, clickHouseCtx)
	http.HandleFunc("/admin/src/", server.serveWithESBuild)
	http.HandleFunc("/api/update-results", server.serveUpdateResults)
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
