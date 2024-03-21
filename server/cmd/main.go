//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"chichi/server"

	"gopkg.in/yaml.v3"
)

func main() {

	// Configure the logger.
	logFile, err := os.OpenFile("error.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		p, err := filepath.Abs("error.log")
		if err != nil {
			p = "error.log"
		}
		slog.Error("cannot open log file", "path", p, "err", err)
		os.Exit(1)
	}
	defer logFile.Close()
	logger := slog.New(slog.NewTextHandler(io.MultiWriter(logFile, os.Stderr), nil))
	slog.SetDefault(logger)

	// Read the configuration file.
	settingsFile := "config.yaml"
	if len(os.Args) > 1 {
		settingsFile = os.Args[1]
	}
	settingsFileContent, err := os.ReadFile(settingsFile)
	if err != nil {
		slog.Error("cannot read configuration file", "path", settingsFile, "err", err)
		os.Exit(1)
	}
	var settings server.Settings
	dec := yaml.NewDecoder(bytes.NewReader(settingsFileContent))
	dec.KnownFields(true)
	err = dec.Decode(&settings)
	if err != nil {
		slog.Error("cannot parse configuration file", "path", settingsFile, "err", err)
		os.Exit(1)
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
		slog.Error("error occurred running server", "err", err)
		os.Exit(1)
	}

}
