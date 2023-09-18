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
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"chichi/server"

	"gopkg.in/yaml.v2"
)

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
	settingsFile := "config.yaml"
	if len(os.Args) > 1 {
		settingsFile = os.Args[1]
	}
	settingsFileContent, err := os.ReadFile(settingsFile)
	if err != nil {
		log.Fatalf("cannot read configuration file %q: %s", settingsFile, err)
	}
	var settings server.Settings
	err = yaml.UnmarshalStrict(settingsFileContent, &settings)
	if err != nil {
		log.Fatalf("cannot parse configuration file %q: %s", settingsFile, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	err = server.Run(ctx, &settings)
	if err != nil {
		log.Printf("[error] %s", err)
	}

}
