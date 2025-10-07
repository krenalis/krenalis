//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/meergo/meergo/core"

	"github.com/getsentry/sentry-go"
)

// Main is the function that executes Meergo. It is designed to be used in
// executable packages that run Meergo's code, and should be utilized in the
// following form:
//
//	func main() {
//	    cmd.Main(assets)
//	}
func Main(assets fs.FS) {

	var help bool
	flag.BoolVar(&help, "help", false, "print the help for meergo and exit")
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}

	if assets != nil {
		var err error
		assets, err = fs.Sub(assets, "meergo-assets")
		if err != nil {
			fatal(1, `directory "meergo-assets" not found in assets (did you forget to generate and embed them?)`)
		}
	}

	// Configure the logger.
	logFile, err := os.OpenFile("error.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		p, err2 := filepath.Abs("error.log")
		if err2 != nil {
			p = "error.log"
		}
		fatalf(1, "cannot open log file %q: %s", p, err)
	}
	defer logFile.Close()

	// Set slog to write to stderr and to the 'error.log' file.
	fileLogger := slog.New(slog.NewTextHandler(io.MultiWriter(logFile, os.Stderr), nil))
	slog.SetDefault(fileLogger)

	// Parse the settings from the environment variables.
	//
	// It is crucial NOT to delete the "MEERGO_" environment variables, because
	// a connector may access them even after initialization.
	settings, err := parseEnvSettings()
	if err != nil {
		fatal(1, err.Error())
	}

	// Configure Sentry, if necessary.
	if settings.SentryTelemetryLevel == core.TelemetryLevelErrors ||
		settings.SentryTelemetryLevel == core.TelemetryLevelAll {
		// Configure Sentry.
		err = sentry.Init(sentry.ClientOptions{
			Dsn:              "https://83b8a272533bd2db6b535547c6517d0e@o4509282180136960.ingest.de.sentry.io/4509282208514128",
			Debug:            false, // set to "true" to get information about telemetry sent to Sentry.
			AttachStacktrace: true,
			SendDefaultPII:   false, // TODO: is it okay to set it to false? See https://github.com/meergo/meergo/issues/1517.
			Integrations: func(integrations []sentry.Integration) []sentry.Integration {
				// The list of integrations loaded by the Sentry SDK by default
				// is available here: https://github.com/getsentry/sentry-go/blob/master/integrations.go.
				var filteredIntegrations []sentry.Integration
				for _, integration := range integrations {
					if integration.Name() == "ContextifyFrames" {
						continue
					}
					filteredIntegrations = append(filteredIntegrations, integration)
				}
				return filteredIntegrations
			},
		})
		if err != nil {
			// Failing to initialize Sentry shouldn't stop Meergo from starting.
			slog.Warn("meergo: failed to init Sentry", "err", err)
		} else {
			defer sentry.Flush(2 * time.Second)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	err = Run(ctx, settings, assets)
	if err != nil {
		slog.Error("meergo: error occurred running server", "err", err)
		fatal(1, err.Error())
	}
}

// fatal writes the message (if not empty) to stderr and exits with the given
// code.
func fatal(code int, msg string) {
	if msg != "" {
		fprint := fmt.Fprintln
		if strings.HasSuffix(msg, "\n") {
			fprint = fmt.Fprint
		}
		_, _ = fprint(os.Stderr, "error: "+msg)
	}
	os.Exit(code)
}

// fatalf formats according to a format specifier and writes (if not empty) to
// stderr, and exits with the given code.
func fatalf(code int, format string, a ...any) {
	fatal(code, fmt.Sprintf(format, a...))
}
