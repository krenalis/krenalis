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
	"expvar"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/cmd/mcp"
	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/metrics"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const telemetryLevelErrors = core.TelemetryLevelErrors
const telemetryLevelAll = core.TelemetryLevelAll

type Settings struct {
	TerminationDelay                time.Duration
	JavaScriptSDKURL                string
	SentryTelemetryLevel            core.TelemetryLevel
	ExternalAssetsURLs              []string // always non nil, can be empty.
	MemberEmailVerificationRequired bool
	HTTP                            struct {
		Host string
		Port int
		TLS  struct {
			Enabled  bool
			CertFile string
			KeyFile  string
		}
		ExternalURL       string
		ExternalEventURL  string
		ReadHeaderTimeout time.Duration
		ReadTimeout       time.Duration
		WriteTimeout      time.Duration
		IdleTimeout       time.Duration
	}
	DB              core.DBConfig
	MaxMindDBPath   string
	MemberEmailFrom string
	SMTP            struct {
		Host     string
		Port     int
		Username string
		Password string
	}
	Transformers struct {
		Lambda LambdaConfig
		Local  LocalConfig
	}
	OAuthCredentials map[string]*core.OAuthCredentials
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
	SudoUser         string
}

// Run runs the server.
// Cancel ctx to terminate the execution. If ctx is cancelled, Run does not
// return any error.
func Run(ctx context.Context, settings *Settings, assetsFS fs.FS) error {

	config := core.Config{
		DB:                   settings.DB,
		MaxMindDBPath:        settings.MaxMindDBPath,
		MemberEmailFrom:      settings.MemberEmailFrom,
		SMTP:                 settings.SMTP,
		OAuthCredentials:     maps.Clone(settings.OAuthCredentials),
		SentryTelemetryLevel: settings.SentryTelemetryLevel,
	}

	// Choose the transformation function provider setting.
	if settings.Transformers.Lambda.Node.Runtime != "" || settings.Transformers.Lambda.Python.Runtime != "" {
		config.FunctionProvider = core.LambdaConfig(settings.Transformers.Lambda)
	}
	if settings.Transformers.Local.NodeExecutable != "" || settings.Transformers.Local.PythonExecutable != "" {
		config.FunctionProvider = core.LocalConfig(settings.Transformers.Local)
	}

	core, err := core.New(&config)
	if err != nil {
		return err
	}
	defer core.Close()

	sentryErrorTunnel := newSentryErrorTunnel()
	defer sentryErrorTunnel.Close()

	runsOnHTTPS := settings.HTTP.TLS.Enabled || strings.HasPrefix(settings.HTTP.ExternalURL, "https://")
	apisServer := newAPIsServer(core, runsOnHTTPS, settings.JavaScriptSDKURL, settings.HTTP.ExternalURL, settings.HTTP.ExternalEventURL, settings.ExternalAssetsURLs,
		settings.MemberEmailVerificationRequired, settings.SentryTelemetryLevel, sentryErrorTunnel)

	assets, err := newAssets(assetsFS)
	if err != nil {
		return err
	}
	defer assets.Close()

	// Instantiate a new MCP (Model Context Protocol) server.
	mcpServer := mcp.NewMCPServer(core)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Handle panics.
		defer func() {
			if r := recover(); r != nil {

				// Log the panic to panics.log.
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

				// Send the panic to Sentry.
				if settings.SentryTelemetryLevel == telemetryLevelErrors || settings.SentryTelemetryLevel == telemetryLevelAll {
					sentry.CurrentHub().Recover(r)
					flushCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					sentry.FlushWithContext(flushCtx)
				}

				os.Exit(1)

			}
		}()

		// Serve the requests for the MCP (Model Context Protocol) server.
		if r.URL.Path == "/mcp" {
			mcpServer.ServeHTTP(w, r)
			return
		}

		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/"):
			apisServer.ServeHTTP(w, r)
			return
		//case strings.HasPrefix(r.URL.Path, "/webhook/"): TODO(marco): implement webhooks
		//	core.ServeWebhook(w, r)
		//	return
		case r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/") || strings.HasPrefix(r.URL.Path, "/javascript-sdk/"):
			assets.ServeHTTP(w, r)
			return
		case r.URL.Path == "/metrics":
			promhttp.Handler().ServeHTTP(w, r)
			return
		case metrics.Enabled && strings.HasPrefix(r.URL.Path, "/debug/vars"):
			expvar.Handler().ServeHTTP(w, r)
			return
		default:
			http.NotFound(w, r)
			return
		}

	})

	c := http.NewCrossOriginProtection()
	c.AddInsecureBypassPattern("POST /api/v1/events")
	origin := strings.TrimSuffix(settings.HTTP.ExternalURL, "/")
	err = c.AddTrustedOrigin(origin)
	if err != nil {
		return fmt.Errorf("unexpected error calling CrossOriginProtection.AddTrustedOrigin with %q", origin)
	}

	httpServer := http.Server{
		Addr:              net.JoinHostPort(settings.HTTP.Host, strconv.Itoa(settings.HTTP.Port)),
		Handler:           c.Handler(handler),
		ErrorLog:          log.New(&httpLogger{}, "", 0),
		ReadHeaderTimeout: settings.HTTP.ReadHeaderTimeout,
		ReadTimeout:       settings.HTTP.ReadTimeout,
		WriteTimeout:      settings.HTTP.WriteTimeout,
		IdleTimeout:       settings.HTTP.IdleTimeout,
	}

	exited := make(chan error)
	go func() {
		if settings.HTTP.TLS.Enabled {
			exited <- httpServer.ListenAndServeTLS(settings.HTTP.TLS.CertFile, settings.HTTP.TLS.KeyFile)
		} else {
			exited <- httpServer.ListenAndServe()
		}
	}()

	// Print a message with the external URL.
	_, _ = fmt.Fprintf(os.Stderr, "The Meergo Admin console is now available at: %s\n", settings.HTTP.ExternalURL+"admin")

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
