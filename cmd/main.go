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
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/state"

	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
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

	// Set slog to write to stderr and to the 'error.log' file.
	fileLogger := slog.New(slog.NewTextHandler(io.MultiWriter(logFile, os.Stderr), nil))
	slog.SetDefault(fileLogger)

	// Read environment variables from the '.env' file, if exists.
	// It is important to call Overload instead of Load because we want any
	// environment variables already set to be overwritten.
	err = godotenv.Overload()
	if err != nil {
		if _, ok := err.(*fs.PathError); !ok {
			slog.Error("cmd: error occurred while loading .env file", "err", err)
			os.Exit(1)
		}
	}

	// Read the settings from the environment variables.
	settings, err := settingsFromEnv()
	if err != nil {
		slog.Error("cmd: error occurred while reading settings from environment variables", "err", err)
		os.Exit(1)
	}

	// Clear all the Meergo environment variables, that are the environment
	// variables that start with "MEERGO_".
	for _, v := range os.Environ() {
		if key, _, ok := strings.Cut(v, "="); ok && strings.HasPrefix(key, "MEERGO_") {
			err := os.Unsetenv(key)
			if err != nil {
				slog.Error("cmd: cannot unset environment variable %q: %s", key, err)
				os.Exit(1)
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
			slog.Error("cmd: cannot init Sentry", "err", err)
			os.Exit(1)
		}
		defer sentry.Flush(2 * time.Second)
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
		slog.Error("cmd: error occurred running server", "err", err)
		os.Exit(1)
	}
}

// settingsFromEnv determines the Meergo settings from the process environment
// variables.
//
// This function does not alter the environment variables.
func settingsFromEnv() (*Settings, error) {

	settings := &Settings{}
	var err error

	if termDelay := os.Getenv("MEERGO_TERMINATION_DELAY"); termDelay != "" {
		settings.TerminationDelay, err = time.ParseDuration(termDelay)
		if err != nil {
			return nil, fmt.Errorf("invalid duration value specified for MEERGO_TERMINATION_DELAY: %s", err)
		}
	}
	settings.JavaScriptSDKURL = os.Getenv("MEERGO_JAVASCRIPT_SDK_URL")

	// Telemetry level.
	switch os.Getenv("MEERGO_TELEMETRY_LEVEL") {
	case "none":
		settings.SentryTelemetryLevel = core.TelemetryLevelNone
	case "errors":
		settings.SentryTelemetryLevel = core.TelemetryLevelErrors
	case "stats":
		settings.SentryTelemetryLevel = core.TelemetryLevelStats
	case "", "all":
		settings.SentryTelemetryLevel = core.TelemetryLevelAll
	default:
		return nil, fmt.Errorf("invalid telemetry level specified for MEERGO_TERMINATION_DELAY," +
			" expecting one of: \"none\", \"errors\", \"stats\", \"all\" or \"\" (which means \"all\")")
	}

	// HTTP.
	settings.HTTP.Host = os.Getenv("MEERGO_HTTP_HOST")
	if httpPort := os.Getenv("MEERGO_HTTP_PORT"); httpPort != "" {
		settings.HTTP.Port, err = strconv.Atoi(httpPort)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value specified for MEERGO_HTTP_PORT: %s", err)
		}
	}
	settings.HTTP.TLS.Enabled, err = boolEnvVar(os.Getenv("MEERGO_HTTP_TLS_ENABLED"))
	if err != nil {
		return nil, fmt.Errorf("invalid value specified for MEERGO_HTTP_TLS_ENABLED: %s", err)
	}
	settings.HTTP.TLS.CertFile = os.Getenv("MEERGO_HTTP_TLS_CERT_FILE")
	settings.HTTP.TLS.KeyFile = os.Getenv("MEERGO_HTTP_TLS_KEY_FILE")
	settings.HTTP.ExternalURL = os.Getenv("MEERGO_HTTP_EXTERNAL_URL")
	settings.HTTP.EventURL = os.Getenv("MEERGO_HTTP_EVENT_URL")
	if settings.HTTP.ReadHeaderTimeout, err = parseHTTPDuration("MEERGO_HTTP_READ_HEADER_TIMEOUT", 2*time.Second); err != nil {
		return nil, err
	}
	if settings.HTTP.ReadTimeout, err = parseHTTPDuration("MEERGO_HTTP_READ_TIMEOUT", 5*time.Second); err != nil {
		return nil, err
	}
	if settings.HTTP.WriteTimeout, err = parseHTTPDuration("MEERGO_HTTP_WRITE_TIMEOUT", 10*time.Second); err != nil {
		return nil, err
	}
	if settings.HTTP.IdleTimeout, err = parseHTTPDuration("MEERGO_HTTP_IDLE_TIMEOUT", 120*time.Second); err != nil {
		return nil, err
	}

	// DB.
	settings.DB.Host = os.Getenv("MEERGO_DB_HOST")
	if dbPort := os.Getenv("MEERGO_DB_PORT"); dbPort != "" {
		settings.DB.Port, err = strconv.Atoi(dbPort)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value specified for MEERGO_DB_PORT: %s", err)
		}
	}
	settings.DB.Username = os.Getenv("MEERGO_DB_USERNAME")
	settings.DB.Password = os.Getenv("MEERGO_DB_PASSWORD")
	settings.DB.Database = os.Getenv("MEERGO_DB_DATABASE")
	settings.DB.Schema = os.Getenv("MEERGO_DB_SCHEMA")
	if maxConn := os.Getenv("MEERGO_DB_MAX_CONNECTIONS"); maxConn != "" {
		settings.DB.MaxConnections, err = strconv.Atoi(os.Getenv("MEERGO_DB_MAX_CONNECTIONS"))
		if err != nil {
			return nil, fmt.Errorf("invalid integer value specified for MEERGO_DB_MAX_CONNECTIONS: %s", err)
		}
		if settings.DB.MaxConnections <= 0 || settings.DB.MaxConnections > math.MaxInt32 {
			return nil, fmt.Errorf("invalid integer value specified for MEERGO_DB_MAX_CONNECTIONS: %s", err)
		}
	}

	// Member emails.
	settings.SkipMemberEmailVerification, err = boolEnvVar(os.Getenv("MEERGO_SKIP_MEMBER_EMAIL_VERIFICATION"))
	if err != nil {
		return nil, fmt.Errorf("invalid value specified for MEERGO_SKIP_MEMBER_EMAIL_VERIFICATION: %s", err)
	}
	settings.MemberEmailFrom = os.Getenv("MEERGO_MEMBER_EMAIL_FROM")

	// SMTP.
	settings.SMTP.Host = os.Getenv("MEERGO_SMTP_HOST")
	if smtpPort := os.Getenv("MEERGO_SMTP_PORT"); smtpPort != "" {
		settings.SMTP.Port, err = strconv.Atoi(os.Getenv("MEERGO_SMTP_PORT"))
		if err != nil {
			return nil, fmt.Errorf("invalid integer value specified for MEERGO_SMTP_PORT: %s", err)
		}
	}
	settings.SMTP.Username = os.Getenv("MEERGO_SMTP_USERNAME")
	settings.SMTP.Password = os.Getenv("MEERGO_SMTP_PASSWORD")

	// MaxMind DB Path.
	settings.MaxMindDBPath = os.Getenv("MEERGO_MAXMIND_DB_PATH")

	// Transformations - Lambda.
	settings.Transformations.Lambda.AccessKeyID = os.Getenv("MEERGO_TRANSFORMATIONS_LAMBDA_ACCESS_KEY_ID")
	settings.Transformations.Lambda.SecretAccessKey = os.Getenv("MEERGO_TRANSFORMATIONS_LAMBDA_SECRET_ACCESS_KEY")
	settings.Transformations.Lambda.Region = os.Getenv("MEERGO_TRANSFORMATIONS_LAMBDA_REGION")
	settings.Transformations.Lambda.Role = os.Getenv("MEERGO_TRANSFORMATIONS_LAMBDA_ROLE")
	settings.Transformations.Lambda.Node.Runtime = os.Getenv("MEERGO_TRANSFORMATIONS_LAMBDA_NODE_RUNTIME")
	settings.Transformations.Lambda.Node.Layer = os.Getenv("MEERGO_TRANSFORMATIONS_LAMBDA_NODE_LAYER")
	settings.Transformations.Lambda.Python.Runtime = os.Getenv("MEERGO_TRANSFORMATIONS_LAMBDA_PYTHON_RUNTIME")
	settings.Transformations.Lambda.Python.Layer = os.Getenv("MEERGO_TRANSFORMATIONS_LAMBDA_PYTHON_LAYER")

	// Transformations - Local.
	settings.Transformations.Local.NodeExecutable = os.Getenv("MEERGO_TRANSFORMATIONS_LOCAL_NODE_EXECUTABLE")
	settings.Transformations.Local.PythonExecutable = os.Getenv("MEERGO_TRANSFORMATIONS_LOCAL_PYTHON_EXECUTABLE")
	settings.Transformations.Local.FunctionsDir = os.Getenv("MEERGO_TRANSFORMATIONS_LOCAL_FUNCTIONS_DIR")

	// OAuth.
	if clientID := os.Getenv("MEERGO_OAUTH_HUBSPOT_CLIENT_ID"); clientID != "" {
		if settings.OAuth == nil {
			settings.OAuth = make(map[string]*state.ConnectorOAuth)
		}
		settings.OAuth["HubSpot"] = &state.ConnectorOAuth{
			ClientID:     clientID,
			ClientSecret: os.Getenv("MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET"),
		}
	}
	if clientID := os.Getenv("MEERGO_OAUTH_MAILCHIMP_CLIENT_ID"); clientID != "" {
		if settings.OAuth == nil {
			settings.OAuth = make(map[string]*state.ConnectorOAuth)
		}
		settings.OAuth["Mailchimp"] = &state.ConnectorOAuth{
			ClientID:     clientID,
			ClientSecret: os.Getenv("MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET"),
		}
	}

	return settings, nil
}

// boolEnvVar parses the value read from an environment variable as a boolean,
// returning either the value read (if valid) or an error.
func boolEnvVar(v string) (bool, error) {
	switch v {
	case "true":
		return true, nil
	case "false", "":
		return false, nil
	default:
		return false, fmt.Errorf("value %q is not a valid boolean value (expected \"true\", \"false\" or empty string)", v)
	}
}

// parseHTTPDuration parses the value of an HTTP configuration setting into a
// time.Duration.
func parseHTTPDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	s := os.Getenv(key)
	if s == "" {
		return defaultValue, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid value specified for %s: %s", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("invalid value specified for %s: it must be greater than 0", key)
	}
	return d, nil
}
