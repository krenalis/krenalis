//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"gopkg.in/yaml.v3"
)

// Main is the function that executes Meergo. It is designed to be used in
// executable packages that run Meergo's code, and should be utilized in the
// following form:
//
//	func main() {
//	    cmd.Main(assets)
//	}
func Main(assets fs.FS) {

	if assets != nil {
		var err error
		assets, err = fs.Sub(assets, "meergo-assets")
		if err != nil {
			panic("meergo: there is no directory 'meergo-assets' in assets")
		}
	}

	// Configure the logger.
	logFile, err := os.OpenFile("error.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		p, err := filepath.Abs("error.log")
		if err != nil {
			p = "error.log"
		}
		slog.Error("cmd: cannot open log file", "path", p, "err", err)
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
		slog.Error("cmd: cannot read configuration file", "path", settingsFile, "err", err)
		os.Exit(1)
	}
	var settings Settings
	dec := yaml.NewDecoder(bytes.NewReader(settingsFileContent))
	dec.KnownFields(true)
	err = dec.Decode(&settings)
	if err != nil {
		slog.Error("cmd: cannot parse configuration file", "path", settingsFile, "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	err = Run(ctx, &settings, assets)
	if err != nil {
		slog.Error("cmd: error occurred running server", "err", err)
		os.Exit(1)
	}
}
