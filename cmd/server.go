// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
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

	"github.com/krenalis/krenalis/cmd/internal/mcp"
	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/prometheus"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const telemetryLevelErrors = core.TelemetryLevelErrors
const telemetryLevelAll = core.TelemetryLevelAll

// Run runs the server.
// Cancel ctx to terminate the execution. If ctx is canceled, Run does not
// return any error.
// initDBIfEmpty controls whether the PostgreSQL database should be initialized
// in case it is empty; if initDockerMember is true in addition to
// initDBIfEmpty, a member specific for Docker scenarios is initialized.
func Run(ctx context.Context, config *Config, assetsFS fs.FS, initDBIfEmpty, initDockerMember bool) error {

	conf := core.Config{
		KMS:                           config.KMS,
		OrganizationsAPIKey:           config.OrganizationsAPIKey,
		DB:                            config.DB,
		NATS:                          config.NATS,
		MaxMindDBPath:                 config.MaxMindDBPath,
		MemberEmailFrom:               config.MemberEmailFrom,
		SMTP:                          config.SMTP,
		OAuthCredentials:              maps.Clone(config.OAuthCredentials),
		SentryTelemetryLevel:          config.SentryTelemetryLevel,
		MaxQueuedEventsPerDestination: config.MaxQueuedEventsPerDestination,
	}
	conf.DatabaseInitialization.InitIfEmpty = initDBIfEmpty
	conf.DatabaseInitialization.InitDockerMember = initDockerMember

	// Choose the transformation function provider setting.
	if config.Transformers.Lambda.NodeJS.Runtime != "" || config.Transformers.Lambda.Python.Runtime != "" {
		conf.FunctionProvider = core.LambdaConfig(config.Transformers.Lambda)
	}
	if config.Transformers.Local.NodeJSExecutable != "" || config.Transformers.Local.PythonExecutable != "" {
		conf.FunctionProvider = core.LocalConfig(config.Transformers.Local)
	}

	core, err := core.New(ctx, &conf)
	if err != nil {
		return err
	}
	defer core.Close(ctx)

	// Destroy the NATS private key.
	for i := range conf.NATS.NKey {
		conf.NATS.NKey[i] = 0
	}

	sentryErrorTunnel := newSentryErrorTunnel()
	defer sentryErrorTunnel.Close()

	runsOnHTTPS := config.HTTP.TLS.Enabled || strings.HasPrefix(config.HTTP.ExternalURL, "https://")
	apisServer := newAPIsServer(core, runsOnHTTPS, config.JavaScriptSDKURL,
		config.HTTP.ExternalURL, config.HTTP.ExternalEventURL, config.ExternalAssetsURLs,
		config.PotentialConnectorsURL, config.InviteMembersViaEmail, config.OrganizationsAPIKey,
		config.SentryTelemetryLevel, sentryErrorTunnel)

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
	if config.PrometheusMetricsEnabled {
		prometheusMetricsHandler = promhttp.Handler()
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Handle panics.
		defer func() {
			if r := recover(); r != nil {

				// Log the panic (and the stack trace) using slog.Error.
				slog.Error("cmd: a panic occurred, Krenalis will exit with status code 1", "reason", r, "stacktrace", string(debug.Stack()))

				// Send the panic to Sentry.
				if config.SentryTelemetryLevel == telemetryLevelErrors || config.SentryTelemetryLevel == telemetryLevelAll {
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
			if config.PrometheusMetricsEnabled {
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
	origin := strings.TrimSuffix(config.HTTP.ExternalURL, "/")
	err = c.AddTrustedOrigin(origin)
	if err != nil {
		return fmt.Errorf("unexpected error calling CrossOriginProtection.AddTrustedOrigin with %q", origin)
	}

	addr := net.JoinHostPort(config.HTTP.Host, strconv.Itoa(config.HTTP.Port))

	httpServer := http.Server{
		Addr:              addr,
		Handler:           c.Handler(handler),
		ErrorLog:          log.New(&httpLogger{}, "", 0),
		ReadHeaderTimeout: config.HTTP.ReadHeaderTimeout,
		ReadTimeout:       config.HTTP.ReadTimeout,
		WriteTimeout:      config.HTTP.WriteTimeout,
		IdleTimeout:       config.HTTP.IdleTimeout,
	}

	var cert tls.Certificate

	if config.HTTP.TLS.Enabled {
		cert, err = tls.LoadX509KeyPair(config.HTTP.TLS.CertFile, config.HTTP.TLS.KeyFile)
		if err != nil {
			return err
		}
		httpServer.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}
	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return err
	}

	exited := make(chan error)
	go func() {
		if config.HTTP.TLS.Enabled {
			exited <- httpServer.ServeTLS(ln, "", "")
		} else {
			exited <- httpServer.Serve(ln)
		}
	}()

	// Log a human-readable overview of all externally exposed server endpoints.
	prometheusMetricsLine := ""
	if config.PrometheusMetricsEnabled {
		prometheusMetricsLine = fmt.Sprintf("├─ Prometheus metrics:  %s\n", config.HTTP.ExternalURL+"metrics")
	}
	msg := fmt.Sprintf(
		"The Krenalis server has been started at %s\n"+
			"%s"+
			"├─ MCP server: %s\n"+
			"├─ REST API:   %s\n"+
			"└─ Event ingestion endpoint: %s\n\n"+
			" > Admin console: %s\n\n",
		addr,
		prometheusMetricsLine,
		config.HTTP.ExternalURL+"mcp",
		config.HTTP.ExternalURL+"v1/",
		config.HTTP.ExternalEventURL,
		config.HTTP.ExternalURL+"admin",
	)
	slog.Info(msg)

	// Warn if the TLS certificate may not be accepted by clients.
	for _, name := range config.HTTP.TLS.DNSNames {
		err := verifyCertificate(cert, name, nil)
		if err != nil {
			slog.Warn(fmt.Sprintf("%s; clients are likely to reject TLS connections", err))
		}
	}

	select {
	case <-ctx.Done():
		if delay := config.TerminationDelay; delay == 0 {
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

// verifyCertificate checks the server TLS certificate against the provided
// roots and reports issues that could cause clients to reject the connection.
// If roots is nil, the system roots are used.
func verifyCertificate(cert tls.Certificate, dnsName string, roots *x509.CertPool) error {

	// Note: with GODEBUG=x509keypairleaf=0, cert.Leaf may be nil.
	if cert.Leaf == nil {
		return nil
	}

	var err error

	if roots == nil {
		roots, err = x509.SystemCertPool()
		if err != nil {
			return fmt.Errorf("unable to load system certificate pool: %w", err)
		}
	}

	intermediates := x509.NewCertPool()
	for _, der := range cert.Certificate[1:] {
		c, err := x509.ParseCertificate(der)
		if err != nil {
			return fmt.Errorf("failed to parse intermediate certificate: %w", err)
		}
		intermediates.AddCert(c)
	}

	opts := x509.VerifyOptions{
		DNSName:       dnsName,
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	_, err = cert.Leaf.Verify(opts)
	if err != nil {
		var unknownAuthErr x509.UnknownAuthorityError
		var hostnameErr x509.HostnameError
		var invalidErr x509.CertificateInvalidError

		switch {
		case errors.As(err, &hostnameErr):
			return fmt.Errorf("server TLS certificate is not valid for the hostname %q", dnsName)
		case errors.As(err, &unknownAuthErr):
			return fmt.Errorf("server TLS certificate is not trusted by system CA")
		case errors.As(err, &invalidErr):
			if invalidErr.Reason == x509.Expired {
				return fmt.Errorf("server TLS certificate has expired")
			}
			return fmt.Errorf("server TLS certificate is not valid")
		default:
			return fmt.Errorf("server TLS certificate verification failed: %w", err)
		}
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
