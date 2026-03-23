// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"context"
	"errors"
	"expvar"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/cmd/mcp"
	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/prometheus"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const telemetryLevelErrors = core.TelemetryLevelErrors
const telemetryLevelAll = core.TelemetryLevelAll

type Settings struct {
	TerminationDelay       time.Duration
	JavaScriptSDKURL       string
	SentryTelemetryLevel   core.TelemetryLevel
	ExternalAssetsURLs     []string // always non nil, can be empty.
	PotentialConnectorsURL string   // must be a valid URL or empty string (which means: do not load the JSON file).
	InviteMembersViaEmail  bool
	HTTP                   struct {
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
	NATS            core.NATSConfig
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
	PrometheusMetricsEnabled      bool
	OAuthCredentials              map[string]*core.OAuthCredentials
	MaxQueuedEventsPerDestination int
}

type LambdaConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Role            string
	NodeJS          struct {
		Runtime string
		Layer   string
	}
	Python struct {
		Runtime string
		Layer   string
	}
}

type LocalConfig struct {
	NodeJSExecutable string
	PythonExecutable string
	FunctionsDir     string
	SudoUser         string
	DoasUser         string
}

// Run runs the server.
// Cancel ctx to terminate the execution. If ctx is canceled, Run does not
// return any error.
// initDBIfEmpty controls whether the PostgreSQL database should be initialized
// in case it is empty; if initDockerMember is true in addition to
// initDBIfEmpty, a member specific for Docker scenarios is initialized.
func Run(ctx context.Context, settings *Settings, assetsFS fs.FS, initDBIfEmpty, initDockerMember bool) error {

	config := core.Config{
		DB:                            settings.DB,
		NATS:                          settings.NATS,
		MaxMindDBPath:                 settings.MaxMindDBPath,
		MemberEmailFrom:               settings.MemberEmailFrom,
		SMTP:                          settings.SMTP,
		OAuthCredentials:              maps.Clone(settings.OAuthCredentials),
		SentryTelemetryLevel:          settings.SentryTelemetryLevel,
		MaxQueuedEventsPerDestination: settings.MaxQueuedEventsPerDestination,
	}
	config.DatabaseInitialization.InitIfEmpty = initDBIfEmpty
	config.DatabaseInitialization.InitDockerMember = initDockerMember

	// Choose the transformation function provider setting.
	if settings.Transformers.Lambda.NodeJS.Runtime != "" || settings.Transformers.Lambda.Python.Runtime != "" {
		config.FunctionProvider = core.LambdaConfig(settings.Transformers.Lambda)
	}
	if settings.Transformers.Local.NodeJSExecutable != "" || settings.Transformers.Local.PythonExecutable != "" {
		config.FunctionProvider = core.LocalConfig(settings.Transformers.Local)
	}

	core, err := core.New(ctx, &config)
	if err != nil {
		return err
	}
	defer core.Close(ctx)

	// Destroy the NATS private key.
	for i := range config.NATS.NKey {
		config.NATS.NKey[i] = 0
	}

	sentryErrorTunnel := newSentryErrorTunnel()
	defer sentryErrorTunnel.Close()

	runsOnHTTPS := settings.HTTP.TLS.Enabled || strings.HasPrefix(settings.HTTP.ExternalURL, "https://")
	apisServer := newAPIsServer(core, runsOnHTTPS, settings.JavaScriptSDKURL,
		settings.HTTP.ExternalURL, settings.HTTP.ExternalEventURL, settings.ExternalAssetsURLs,
		settings.PotentialConnectorsURL, settings.InviteMembersViaEmail,
		settings.SentryTelemetryLevel, sentryErrorTunnel)

	admin, err := newAdmin(assetsFS)
	if err != nil {
		return err
	}
	defer admin.Close()

	// Instantiate a new MCP (Model Context Protocol) server.
	mcpServer := mcp.NewMCPServer(core)
	defer func() {
		err := mcpServer.Close(context.Background())
		if err != nil {
			slog.Warn("an error occurred closing the  MCP server", "error", err)
		}
	}()

	// Instantiate the Prometheus metrics handler.
	var prometheusMetricsHandler http.Handler
	if settings.PrometheusMetricsEnabled {
		prometheusMetricsHandler = promhttp.Handler()
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Handle panics.
		defer func() {
			if r := recover(); r != nil {

				// Log the panic (and the stack trace) using slog.Error.
				slog.Error("cmd: a panic occurred, Meergo will exit with status code 1", "reason", r, "stacktrace", string(debug.Stack()))

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
			if r.Method == "GET" && strings.Contains(r.Header.Get("Accept"), "text/html") {
				err := serveMCPServerHTMLIndex(w)
				if err != nil {
					slog.Error("failed to serve the MCP server's HTML index page", "error", err)
				}
				return
			}
			mcpServer.ServeHTTP(w, r)
			return
		}

		switch {
		case strings.HasPrefix(r.URL.Path, "/v1/"):
			apisServer.ServeHTTP(w, r)
			return
		//case strings.HasPrefix(r.URL.Path, "/webhook/"): TODO(marco): implement webhooks
		//	core.ServeWebhook(w, r)
		//	return
		case r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/"):
			admin.ServeHTTP(w, r)
			return
		case r.URL.Path == "/metrics":
			if settings.PrometheusMetricsEnabled {
				prometheusMetricsHandler.ServeHTTP(w, r)
				return
			}
		case prometheus.Enabled && strings.HasPrefix(r.URL.Path, "/debug/vars"):
			expvar.Handler().ServeHTTP(w, r)
			return
		default:
		}

		http.NotFound(w, r)

	})

	c := http.NewCrossOriginProtection()
	c.AddInsecureBypassPattern("POST /v1/events")
	origin := strings.TrimSuffix(settings.HTTP.ExternalURL, "/")
	err = c.AddTrustedOrigin(origin)
	if err != nil {
		return fmt.Errorf("unexpected error calling CrossOriginProtection.AddTrustedOrigin with %q", origin)
	}

	addr := net.JoinHostPort(settings.HTTP.Host, strconv.Itoa(settings.HTTP.Port))

	httpServer := http.Server{
		Addr:              addr,
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

	// Log a human-readable overview of all externally exposed server endpoints.
	prometheusMetricsLine := ""
	if settings.PrometheusMetricsEnabled {
		prometheusMetricsLine = fmt.Sprintf("├─ Prometheus metrics:  %s\n", settings.HTTP.ExternalURL+"metrics")
	}
	msg := fmt.Sprintf(
		"The Meergo server has been started at %s\n"+
			"%s"+
			"├─ MCP server: %s\n"+
			"├─ REST API:   %s\n"+
			"└─ Event ingestion endpoint: %s\n\n"+
			" > Admin console: %s\n\n",
		addr,
		prometheusMetricsLine,
		settings.HTTP.ExternalURL+"mcp",
		settings.HTTP.ExternalURL+"v1/",
		settings.HTTP.ExternalEventURL,
		settings.HTTP.ExternalURL+"admin",
	)
	slog.Info(msg)

	select {
	case <-ctx.Done():
		if delay := settings.TerminationDelay; delay == 0 {
			slog.Info("cmd: received termination signal, shutting down")
		} else {
			slog.Info("cmd: received termination signal; waiting for before proceeding", "delay", delay)
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

// serveMCPServerHTMLIndex returns the MCP server HTML index page.
func serveMCPServerHTMLIndex(w http.ResponseWriter) error {
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive, nosnippet, notranslate, noimageindex")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fi, err := static.Open("static/mcp_index.html")
	if err != nil {
		return errors.New("embedded file 'static/mcp_index.html' not found in executable")
	}
	_, _ = io.Copy(w, fi)
	_ = fi.Close()
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
