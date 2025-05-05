//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"expvar"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/metrics"
	"github.com/meergo/meergo/telemetry"
)

type Settings struct {
	EncryptionKey    string
	TerminationDelay time.Duration
	JavaScriptSDKURL string
	HTTP             struct {
		Host string
		Port int
		TLS  struct {
			Enabled  bool
			CertFile string
			KeyFile  string
		}
		ExternalURL string
		EventURL    string
	}
	DB   core.DBConfig
	SMTP struct {
		Host     string
		Port     int
		Username string
		Password string
	}
	Transformations struct {
		Lambda LambdaConfig
		Local  LocalConfig
	}
	OAuth     map[string]*state.ConnectorOAuth
	Telemetry struct {
		Enable bool
	}
}

type LambdaConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Role            string
	Node            struct {
		Runtime string
		Layer   string
	}
	Python struct {
		Runtime string
		Layer   string
	}
}

type LocalConfig struct {
	NodeExecutable   string
	PythonExecutable string
	FunctionsDir     string
}

// Run runs the server.
// Cancel ctx to terminate the execution. If ctx is cancelled, Run does not
// return any error.
func Run(ctx context.Context, settings *Settings, assetsFS fs.FS) error {

	if settings.Telemetry.Enable {
		err := telemetry.Init(ctx)
		if err != nil {
			return err
		}
	}

	// Determine the address, the external URL, the JavaScript SDK URL and the
	// event URL.
	addr := settings.HTTP.Host + ":" + strconv.Itoa(settings.HTTP.Port)
	if addr == ":" {
		addr = "127.0.0.1:9090"
	}
	externalURL := settings.HTTP.ExternalURL
	if externalURL == "" {
		protocol := "http"
		if settings.HTTP.TLS.Enabled {
			protocol = "https"
		}
		externalURL = fmt.Sprintf("%s://%s", protocol, addr)
	}

	config := core.Config{
		DB:   settings.DB,
		SMTP: settings.SMTP,
	}

	// Choose the transformation function provider setting.
	if settings.Transformations.Lambda.Node.Runtime != "" || settings.Transformations.Lambda.Python.Runtime != "" {
		config.FunctionProvider = core.LambdaConfig(settings.Transformations.Lambda)
	}
	if settings.Transformations.Local.NodeExecutable != "" || settings.Transformations.Local.PythonExecutable != "" {
		if config.FunctionProvider != nil {
			return errors.New("cannot specify both the Lambda and the local transformation provider in the configuration (hint: check the environment variables passed to Meergo)")
		}
		config.FunctionProvider = core.LocalConfig(settings.Transformations.Local)
	}

	// Validate the settings of the connectors.
	if settings.OAuth != nil {
		for name, setting := range settings.OAuth {
			if (setting.ClientID == "") == (setting.ClientSecret == "") {
				continue
			}
			if setting.ClientID == "" {
				return fmt.Errorf("OAuth clientID value for connector %q cannot be empty (hint: check the environment variables passed to Meergo)", name)
			}
			return fmt.Errorf("OAuth clientSecret value for connector %q cannot be empty (hint: check the environment variables passed to Meergo)", name)
		}
		config.ConnectorsOAuth = maps.Clone(settings.OAuth)
	}

	// Decode the encryption key.
	if settings.EncryptionKey == "" {
		return errors.New("encryption key is missing from the configuration file")
	}
	if padding := len(settings.EncryptionKey) % 4; padding > 0 {
		settings.EncryptionKey += strings.Repeat("=", 4-padding)
	}
	var err error
	config.EncryptionKey, err = base64.StdEncoding.DecodeString(settings.EncryptionKey)
	if err != nil {
		return errors.New("encryption key in the configuration file is not in Base64 format")
	}
	if len(config.EncryptionKey) != 64 {
		return fmt.Errorf("encryption key in the configuration file is not 64 bytes long, but %d", len(config.EncryptionKey))
	}

	core, err := core.New(&config)
	if err != nil {
		return err
	}
	defer core.Close()

	javaScriptSDKURL := settings.JavaScriptSDKURL
	if javaScriptSDKURL == "" {
		javaScriptSDKURL = strings.TrimRight(externalURL, "/") + "/javascript-sdk/dist/meergo.min.js"
	}
	eventURL := settings.HTTP.EventURL
	if eventURL == "" {
		eventURL = strings.TrimRight(externalURL, "/") + "/api/v1/events"
	}

	apisServer := newAPIsServer(core, config.EncryptionKey, settings.HTTP.TLS.Enabled, javaScriptSDKURL, eventURL)

	assets, err := newAssets(assetsFS)
	if err != nil {
		return err
	}
	defer assets.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Handle panics.
		defer func() {
			if r := recover(); r != nil {
				panicsFilename, err := filepath.Abs("panics.log")
				if err != nil {
					slog.Error("cmd: cannot get absolute filepath of 'panics.log'", "err", err)
					return
				}
				slog.Error("cmd: a panic occurred, Meergo will exit with status code 1. See the file 'panics.log' for the panic details", "panic reason", r, "panics.log filename", panicsFilename)
				f, err := os.OpenFile(panicsFilename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
				if err != nil {
					slog.Error("cmd: cannot open panic file", "err", err)
					return
				}
				defer f.Close()
				_, err = fmt.Fprintf(f, "\n----- %s -----\nPanic reason: %v\nStack trace:\n%s",
					time.Now().Format("2006-01-02 15:04:05.000"), r, debug.Stack())
				if err != nil {
					slog.Error("cmd: cannot write on panic file", "err", err)
					return
				}
				os.Exit(1)
			}
		}()

		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/"):
			apisServer.ServeHTTP(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/webhook/"):
			core.ServeWebhook(w, r)
			return
		case r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/") || strings.HasPrefix(r.URL.Path, "/javascript-sdk/"):
			assets.ServeHTTP(w, r)
			return
		case metrics.Enabled && strings.HasPrefix(r.URL.Path, "/debug/vars"):
			expvar.Handler().ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
			return
		}

	})

	httpServer := http.Server{
		Addr:     addr,
		Handler:  handler,
		ErrorLog: log.New(&httpLogger{}, "", 0),
	}
	var certPem, keyPem string
	if settings.HTTP.TLS.Enabled {
		certPem, err = filepath.Abs(settings.HTTP.TLS.CertFile)
		if err != nil {
			return err
		}
		keyPem, err = filepath.Abs(settings.HTTP.TLS.KeyFile)
		if err != nil {
			return err
		}
	}

	exited := make(chan error)
	go func() {
		if settings.HTTP.TLS.Enabled {
			exited <- httpServer.ListenAndServeTLS(certPem, keyPem)
		} else {
			exited <- httpServer.ListenAndServe()
		}
	}()

	// Print a message with the external URL.
	_, _ = fmt.Fprintf(os.Stderr, "The Meergo admin is now available at: %s\n", strings.TrimRight(externalURL, "/")+"/admin")

	select {
	case <-ctx.Done():
		if delay := settings.TerminationDelay; delay == 0 {
			slog.Info("cmd: received termination signal, shutting down")
		} else {
			slog.Info(fmt.Sprintf("cmd: received termination signal. Waiting for %s before proceeding...", delay))
			time.Sleep(delay)
			slog.Info("cmd: initiating shutdown")
		}
		err = httpServer.Shutdown(context.Background())
		if err != nil {
			return err
		}
		err = <-exited
	case err = <-exited:
	}
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// httpLogger is the HTTP server's logger that filters out unwanted messages.
type httpLogger struct{}

var tlsHandshakeMsg = []byte("http: TLS handshake error from ")

func (f *httpLogger) Write(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return
	}
	if bytes.HasPrefix(p, tlsHandshakeMsg) {
		return
	}
	if p[len(p)-1] == '\n' {
		p = p[:len(p)-1]
	}
	slog.Info(string(p))
	return
}
