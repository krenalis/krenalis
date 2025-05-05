//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/meergo/meergo/core/state"

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

	// Clear all the environment variables.
	// This, in addition to cleanliness, prevents the content of such variables
	// (especially those with sensitive data) from being accessible to all the
	// code, third-party packages and any started processes.
	os.Clearenv()

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

	settings.EncryptionKey = os.Getenv("MEERGO_ENCRYPTION_KEY")
	if termDelay := os.Getenv("MEERGO_TERMINATION_DELAY"); termDelay != "" {
		settings.TerminationDelay, err = time.ParseDuration(termDelay)
		if err != nil {
			return nil, fmt.Errorf("invalid duration value specified for MEERGO_TERMINATION_DELAY: %s", err)
		}
	}
	settings.JavaScriptSDKURL = os.Getenv("MEERGO_JAVASCRIPT_SDK_URL")

	// HTTP.
	settings.HTTP.Host = os.Getenv("MEERGO_HTTP_HOST")
	if httpPort := os.Getenv("MEERGO_HTTP_PORT"); httpPort != "" {
		settings.HTTP.Port, err = strconv.Atoi(httpPort)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value specified for MEERGO_HTTP_PORT: %s", err)
		}
	}
	settings.HTTP.TLS.Enabled = os.Getenv("MEERGO_HTTP_TLS_ENABLED") == "true"
	settings.HTTP.TLS.CertFile = os.Getenv("MEERGO_HTTP_TLS_CERT_FILE")
	settings.HTTP.TLS.KeyFile = os.Getenv("MEERGO_HTTP_TLS_KEY_FILE")
	settings.HTTP.ExternalURL = os.Getenv("MEERGO_HTTP_EXTERNAL_URL")
	settings.HTTP.EventURL = os.Getenv("MEERGO_HTTP_EVENT_URL")

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
		settings.OAuth["MailChimp"] = &state.ConnectorOAuth{
			ClientID:     clientID,
			ClientSecret: os.Getenv("MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET"),
		}
	}

	// Telemetry.
	settings.Telemetry.Enable = os.Getenv("MEERGO_TELEMETRY_ENABLE") == "true"

	return settings, nil
}
