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
	"os/signal"
	"path/filepath"
	"syscall"

	"chichi/admin"
	"chichi/apis"

	"gopkg.in/gcfg.v1"
)

type Settings struct {
	Main struct {
		Host                         string
		PrintESBuildWarningsOnStderr bool
	}
	PostgreSQL apis.PostgreSQLConfig
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
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Read the configuration file.
	settingsFile := "app.ini"
	if len(os.Args) > 1 {
		settingsFile = os.Args[1]
	}
	var settings Settings
	err = gcfg.ReadFileInto(&settings, settingsFile)
	if err != nil {
		p, err2 := filepath.Abs(settingsFile)
		if err2 != nil {
			p = settingsFile
		}
		log.Fatalf("cannot open %q: %s", p, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	apis, err := apis.New(ctx, &apis.Config{
		PostgreSQL: settings.PostgreSQL,
	})
	if err != nil {
		if err == context.Canceled {
			select {
			case <-ctx.Done():
				os.Exit(0)
			}
		}
		log.Fatalf("[error] %s", err)
	}
	admin := admin.New(apis)

	http.HandleFunc("/admin/", admin.ServeHTTP)
	http.HandleFunc("/api/", apis.ServeHTTP)
	http.HandleFunc("/webhook/", apis.ServeWebhook)
	http.Handle("/trace-events-script/", http.FileServer(http.Dir(".")))

	addr := settings.Main.Host
	if addr == "" {
		addr = "127.0.0.1:9090"
	}
	httpServer := http.Server{
		Addr: addr,
	}
	go func() {
		<-ctx.Done()
		err := httpServer.Shutdown(context.Background())
		if err != nil {
			log.Printf("[error] shutting down HTTP server: %s", err)
		}
	}()
	err = httpServer.ListenAndServeTLS("cert.pem", "key.pem")
	if err != nil && err != http.ErrServerClosed {
		log.Fatal("ListenAndServe: ", err)
	}
}
