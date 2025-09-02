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
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/state"

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

	// Read the settings from the environment variables.
	//
	// It is crucial NOT to delete the "MEERGO_" environment variables, because
	// a connector may access them even after initialization.
	settings, err := settingsFromEnv()
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

// settingsFromEnv determines the Meergo settings from the process environment
// variables.
//
// This function does not alter the environment variables.
func settingsFromEnv() (*Settings, error) {

	envVars, err := meergo.GetEnvVars()
	if err != nil {
		return nil, err
	}

	settings := &Settings{}

	if termDelay := envVars.Get("MEERGO_TERMINATION_DELAY"); termDelay != "" {
		settings.TerminationDelay, err = time.ParseDuration(termDelay)
		if err != nil {
			return nil, fmt.Errorf("invalid duration value specified for MEERGO_TERMINATION_DELAY: %s", err)
		}
	}
	settings.JavaScriptSDKURL, err = parseURL("MEERGO_JAVASCRIPT_SDK_URL", 0)
	if err != nil {
		return nil, err
	}

	// Telemetry level.
	switch envVars.Get("MEERGO_TELEMETRY_LEVEL") {
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
	settings.HTTP.Host = envVars.Get("MEERGO_HTTP_HOST")
	if httpPort := envVars.Get("MEERGO_HTTP_PORT"); httpPort != "" {
		settings.HTTP.Port, err = strconv.Atoi(httpPort)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value specified for MEERGO_HTTP_PORT: %s", err)
		}
	}
	settings.HTTP.TLS.Enabled, err = boolEnvVar(envVars.Get("MEERGO_HTTP_TLS_ENABLED"))
	if err != nil {
		return nil, fmt.Errorf("invalid value specified for MEERGO_HTTP_TLS_ENABLED: %s", err)
	}
	settings.HTTP.TLS.CertFile = envVars.Get("MEERGO_HTTP_TLS_CERT_FILE")
	settings.HTTP.TLS.KeyFile = envVars.Get("MEERGO_HTTP_TLS_KEY_FILE")
	settings.HTTP.ExternalURL, err = parseURL("MEERGO_HTTP_EXTERNAL_URL", noPath|noQuery)
	if err != nil {
		return nil, err
	}
	settings.HTTP.ExternalEventURL, err = parseURL("MEERGO_HTTP_EXTERNAL_EVENT_URL", noQuery)
	if err != nil {
		return nil, err
	}
	if settings.HTTP.ReadHeaderTimeout, err = parseHTTPDuration("MEERGO_HTTP_READ_HEADER_TIMEOUT", 2*time.Second); err != nil {
		return nil, err
	}
	if settings.HTTP.ReadTimeout, err = parseHTTPDuration("MEERGO_HTTP_READ_TIMEOUT", 5*time.Second); err != nil {
		return nil, err
	}
	if settings.HTTP.WriteTimeout, err = parseHTTPDuration("MEERGO_HTTP_WRITE_TIMEOUT", 30*time.Second); err != nil {
		return nil, err
	}
	if settings.HTTP.IdleTimeout, err = parseHTTPDuration("MEERGO_HTTP_IDLE_TIMEOUT", 120*time.Second); err != nil {
		return nil, err
	}

	// DB.
	settings.DB.Host = envVars.Get("MEERGO_DB_HOST")
	if dbPort := envVars.Get("MEERGO_DB_PORT"); dbPort != "" {
		settings.DB.Port, err = strconv.Atoi(dbPort)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value specified for MEERGO_DB_PORT: %s", err)
		}
	}
	settings.DB.Username = envVars.Get("MEERGO_DB_USERNAME")
	settings.DB.Password = envVars.Get("MEERGO_DB_PASSWORD")
	settings.DB.Database = envVars.Get("MEERGO_DB_DATABASE")
	settings.DB.Schema = envVars.Get("MEERGO_DB_SCHEMA")
	if c := envVars.Get("MEERGO_DB_MAX_CONNECTIONS"); c == "" {
		settings.DB.MaxConnections = 8
	} else {
		maxConn, err := strconv.ParseInt(c, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value specified for MEERGO_DB_MAX_CONNECTIONS: %s", err)
		}
		if maxConn < 2 {
			return nil, fmt.Errorf("invalid MEERGO_DB_MAX_CONNECTIONS: %d (minimum is 2)", maxConn)
		}
		settings.DB.MaxConnections = int32(maxConn)
	}

	// Member emails.
	settings.SkipMemberEmailVerification, err = boolEnvVar(envVars.Get("MEERGO_SKIP_MEMBER_EMAIL_VERIFICATION"))
	if err != nil {
		return nil, fmt.Errorf("invalid value specified for MEERGO_SKIP_MEMBER_EMAIL_VERIFICATION: %s", err)
	}
	settings.MemberEmailFrom = envVars.Get("MEERGO_MEMBER_EMAIL_FROM")

	// SMTP.
	settings.SMTP.Host = envVars.Get("MEERGO_SMTP_HOST")
	if smtpPort := envVars.Get("MEERGO_SMTP_PORT"); smtpPort != "" {
		settings.SMTP.Port, err = strconv.Atoi(envVars.Get("MEERGO_SMTP_PORT"))
		if err != nil {
			return nil, fmt.Errorf("invalid integer value specified for MEERGO_SMTP_PORT: %s", err)
		}
	}
	settings.SMTP.Username = envVars.Get("MEERGO_SMTP_USERNAME")
	settings.SMTP.Password = envVars.Get("MEERGO_SMTP_PASSWORD")

	// MaxMind DB Path.
	settings.MaxMindDBPath = envVars.Get("MEERGO_MAXMIND_DB_PATH")

	// Transformations - Lambda.
	settings.Transformations.Lambda.AccessKeyID = envVars.Get("MEERGO_TRANSFORMATIONS_LAMBDA_ACCESS_KEY_ID")
	settings.Transformations.Lambda.SecretAccessKey = envVars.Get("MEERGO_TRANSFORMATIONS_LAMBDA_SECRET_ACCESS_KEY")
	settings.Transformations.Lambda.Region = envVars.Get("MEERGO_TRANSFORMATIONS_LAMBDA_REGION")
	settings.Transformations.Lambda.Role = envVars.Get("MEERGO_TRANSFORMATIONS_LAMBDA_ROLE")
	settings.Transformations.Lambda.Node.Runtime = envVars.Get("MEERGO_TRANSFORMATIONS_LAMBDA_NODE_RUNTIME")
	settings.Transformations.Lambda.Node.Layer = envVars.Get("MEERGO_TRANSFORMATIONS_LAMBDA_NODE_LAYER")
	settings.Transformations.Lambda.Python.Runtime = envVars.Get("MEERGO_TRANSFORMATIONS_LAMBDA_PYTHON_RUNTIME")
	settings.Transformations.Lambda.Python.Layer = envVars.Get("MEERGO_TRANSFORMATIONS_LAMBDA_PYTHON_LAYER")

	// Transformations - Local.
	settings.Transformations.Local.NodeExecutable = envVars.Get("MEERGO_TRANSFORMATIONS_LOCAL_NODE_EXECUTABLE")
	settings.Transformations.Local.PythonExecutable = envVars.Get("MEERGO_TRANSFORMATIONS_LOCAL_PYTHON_EXECUTABLE")
	settings.Transformations.Local.FunctionsDir = envVars.Get("MEERGO_TRANSFORMATIONS_LOCAL_FUNCTIONS_DIR")

	// OAuth.
	if clientID := envVars.Get("MEERGO_OAUTH_HUBSPOT_CLIENT_ID"); clientID != "" {
		if settings.OAuth == nil {
			settings.OAuth = make(map[string]*state.ConnectorOAuth)
		}
		settings.OAuth["HubSpot"] = &state.ConnectorOAuth{
			ClientID:     clientID,
			ClientSecret: envVars.Get("MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET"),
		}
	}
	if clientID := envVars.Get("MEERGO_OAUTH_MAILCHIMP_CLIENT_ID"); clientID != "" {
		if settings.OAuth == nil {
			settings.OAuth = make(map[string]*state.ConnectorOAuth)
		}
		settings.OAuth["Mailchimp"] = &state.ConnectorOAuth{
			ClientID:     clientID,
			ClientSecret: envVars.Get("MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET"),
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

// parseHTTPDuration parses the value of an HTTP configuration setting into a
// time.Duration.
func parseHTTPDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	envVars, err := meergo.GetEnvVars()
	if err != nil {
		return 0, err
	}
	s := envVars.Get(key)
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

type errInvalidURL struct {
	key string
	msg string
}

func (err errInvalidURL) Error() string {
	return fmt.Sprintf("invalid URL specified for %s: %s", err.key, err.msg)
}

type urlValidationFlag int

const (
	noPath urlValidationFlag = 1 << iota
	noQuery
)

func hasURLValidationFlag(f, flag urlValidationFlag) bool {
	return f&flag != 0
}

// parseURL parses the value of an configuration setting into a normalized URL.
func parseURL(key string, flags urlValidationFlag) (string, error) {
	envVars, err := meergo.GetEnvVars()
	if err != nil {
		return "", err
	}
	s := envVars.Get(key)
	if s == "" {
		return "", nil
	}
	if s[0] == ' ' {
		return "", errInvalidURL{key, `it starts with a space`}
	}
	if s[len(s)-1] == ' ' {
		return "", errInvalidURL{key, `it ends with a space`}
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", errInvalidURL{key, err.Error()}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errInvalidURL{key, `scheme must be "http" or "https"`}
	}
	if u.User != nil {
		return "", errInvalidURL{key, "user and password cannot be specified"}
	}
	if u.Host == "" {
		return "", errInvalidURL{key, "host must be specified"}
	}
	port := u.Port()
	if port != "" {
		if p, _ := strconv.Atoi(port); p < 1 || p > 65535 {
			return "", errInvalidURL{key, "port must be in range [1,65535]"}
		}
	}
	if hasURLValidationFlag(flags, noPath) {
		if u.Path != "" && u.Path != "/" {
			return "", errInvalidURL{key, `path must be "/"`}
		}
	}
	if hasURLValidationFlag(flags, noQuery) {
		if u.RawQuery != "" || u.ForceQuery {
			return "", errInvalidURL{key, "query cannot be specified"}
		}
	}
	if strings.IndexByte(s, '#') != -1 {
		return "", errInvalidURL{key, "fragment cannot be specified"}
	}
	var normalized bool
	if port != "" && port[0] == '0' {
		port = strings.TrimLeft(port, "0")
		u.Host = net.JoinHostPort(u.Hostname(), port)
		normalized = true
	}
	if u.Scheme == "http" && port == "80" || u.Scheme == "https" && port == "443" {
		i := strings.LastIndex(u.Host, ":")
		u.Host = u.Host[:i]
		normalized = true
	}
	if u.Path == "" {
		u.Path = "/"
		normalized = true
	}
	if normalized {
		s = u.String()
	}
	return s, nil
}
