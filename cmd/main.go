// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/meergo/meergo/core"

	"github.com/getsentry/sentry-go"
)

//go:embed static
var static embed.FS

// Main is the function that executes Meergo. It is designed to be used in
// executable packages that run Meergo's code, and should be utilized in the
// following form:
//
//	func main() {
//	    cmd.Main(assets)
//	}
func Main(assets fs.FS) {

	var help bool
	var initDBIfEmpty bool
	var initDockerMember bool
	flag.BoolVar(&help, "help", false, "print the help for meergo and exit")
	flag.BoolVar(&initDBIfEmpty, "init-db-if-empty", false, "initialize the PostgreSQL database, if it is empty")
	flag.BoolVar(&initDockerMember, "init-docker-member", false,
		"when initializing the PostgreSQL database, also initialize the Docker member;"+
			" this flag is primarily intended for automated scenarios involving Docker and testing purposes")
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}
	if initDockerMember && !initDBIfEmpty {
		flag.Usage()
		fatal(1, "the -init-docker-member flag can be provided only when the -init-db-if-empty flag is provided")
	}

	if !devMode && assets != nil {
		assets, _ = fs.Sub(assets, "admin/assets")
		_, err := fs.Stat(assets, "index.html.br")
		if err != nil {
			fatal(1, `file "admin/assets/index.html.br" not found in assets (did you forget to generate and embed them?)`)
		}
	}

	// Parse the settings from the environment variables.
	settings, err := parseEnvSettings()
	if err != nil {
		fatal(1, err.Error())
	}

	// Unset the Meergo environment variables, except for those intended for
	// connectors, which can be read by them at any time.
	//
	// This minimizes the possibility that any point in the code can read the
	// configuration passed from the environment.
	for _, v := range os.Environ() {
		if key, _, ok := strings.Cut(v, "="); ok {
			isMeergoVar := strings.HasPrefix(key, "MEERGO_")
			isMeergoConnectorVar := strings.HasPrefix(key, "MEERGO_CONNECTOR_")
			if isMeergoVar && !isMeergoConnectorVar {
				// os.Unsetenv can only fail on Windows if the key is not UTF-8
				// encoded. But since Meergo only supports UTF-8 keys, and this
				// is a rare edge case, failing to unset such a variable
				// shouldn't prevent Meergo from starting.
				_ = os.Unsetenv(key)
			}
		}
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

	err = Run(ctx, settings, assets, initDBIfEmpty, initDockerMember)
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
