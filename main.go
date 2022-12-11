//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"chichi/admin"
	"chichi/apis"

	"gopkg.in/gcfg.v1"
)

type Settings struct {
	Main struct {
		PrintESBuildWarningsOnStderr bool
	}
	PostgreSQL apis.PostgreSQLConfig
	ClickHouse apis.ClickHouseConfig
}

func main() {

	// Configure the logger.
	logFile, err := os.OpenFile("error.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		p, err := filepath.Abs("error.log")
		if err != nil {
			p = "error.log"
		}
		log.Fatalf("cannot open %q: %s", p, err)
	}
	defer logFile.Close()
	log.SetOutput(io.MultiWriter(logFile, os.Stderr))
	log.SetFlags(log.Ldate | log.Ltime)

	// Read the configuration file.
	var settings Settings
	err = gcfg.ReadFileInto(&settings, "app.ini")
	if err != nil {
		p, err := filepath.Abs("app.ini")
		if err != nil {
			p = "app.ini"
		}
		log.Fatalf("cannot open %q: %s", p, err)
	}

	apis, err := apis.New(&apis.Config{
		PostgreSQL: settings.PostgreSQL,
		ClickHouse: settings.ClickHouse,
	})
	if err != nil {
		log.Fatalf("[error] %s", err)
	}
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
