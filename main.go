//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	_ "time/tzdata" // workaround for clickhouse-go issue #162

	"chichi/admin"
	"chichi/apis"
	"chichi/pkg/open2b/sql"

	"github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {

	// Configure the logger.
	logFile, err := os.OpenFile("error.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(io.MultiWriter(logFile, os.Stderr))
	log.SetFlags(log.Ldate | log.Ltime)

	// Read the configuration file.
	settings, err := parseINIFile()
	if err != nil {
		log.Printf("[error] cannot read configuration file: %s", err)
		return
	}
	// Open a connection to the PostgreSQL database.
	db, err := sql.Open(&settings.PostgreSQL)
	if err != nil {
		log.Fatalf("[error] cannot connect to the database: %s", err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatalf("[error] cannot ping PostgreSQL server: %s", err)
	}
	log.Printf("[info] successfully connected to the PostgreSQL server")

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
		log.Printf("[warning] cannot ping ClickHouse server: %s", err)
	} else {
		log.Printf("[info] successfully connected to the ClickHouse server")
	}

	apis := apis.New(db, clickHouseConn)
	admin := admin.New(apis)

	http.HandleFunc("/admin/", admin.ServeHTTP)
	http.HandleFunc("/api/", apis.ServeHTTP)
	http.HandleFunc("/webhook/", apis.ServeWebhook)
	http.Handle("/trace-events-script/", http.FileServer(http.Dir(".")))
	err = http.ListenAndServeTLS(":9090", "cert.pem", "key.pem", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
