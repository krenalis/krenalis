//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b

package cmd

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/validation"
)

// parseSettings parses the settings from the process environment variables.
//
// It does not alter the environment variables.
func parseSettings() (*Settings, error) {

	envVars, err := meergo.GetEnvVars()
	if err != nil {
		return nil, err
	}

	settings := &Settings{}

	if delay := envVars.Get("MEERGO_TERMINATION_DELAY"); delay != "" {
		delay, err := time.ParseDuration(delay)
		if err != nil {
			return nil, fmt.Errorf("invalid duration value specified for MEERGO_TERMINATION_DELAY: %s", err)
		}
		settings.TerminationDelay = delay
	}

	if url, err := parseURL("MEERGO_JAVASCRIPT_SDK_URL", 0); err != nil {
		return nil, fmt.Errorf("MEERGO_JAVASCRIPT_SDK_URL must be a valid URL: %s", err)
	} else if url == "" {
		settings.JavaScriptSDKURL = "https://cdn.jsdelivr.net/npm/@meergo/javascript-sdk/dist/meergo.min.js"
	} else {
		settings.JavaScriptSDKURL = url
	}

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
		return nil, fmt.Errorf("invalid MEERGO_TELEMETRY_LEVEL: want one of none, errors, stats, or all")
	}

	if host := envVars.Get("MEERGO_HTTP_HOST"); host == "" {
		settings.HTTP.Host = "127.0.0.1"
	} else if err := validation.ValidateHost(host); err != nil {
		return nil, fmt.Errorf("MEERGO_HTTP_HOST must be a valid host: %s", err)
	} else {
		settings.HTTP.Host = host
	}

	if port := envVars.Get("MEERGO_HTTP_PORT"); port == "" {
		settings.HTTP.Port = 2022
	} else if port, err := validation.ValidatePortString(port); err != nil {
		return nil, fmt.Errorf("MEERGO_HTTP_PORT must be a valid port: %s", err)
	} else {
		settings.HTTP.Port = port
	}

	if tls, err := boolEnvVar(envVars.Get("MEERGO_HTTP_TLS_ENABLED"), false); err != nil {
		return nil, fmt.Errorf("MEERGO_HTTP_TLS_ENABLED must be a boolean: %s", err)
	} else if tls {
		certFile, err := resolveFilePath("MEERGO_HTTP_TLS_CERT_FILE")
		if err != nil {
			return nil, err
		}
		if certFile == "" {
			return nil, fmt.Errorf("MEERGO_HTTP_TLS_CERT_FILE must be set when MEERGO_HTTP_TLS_ENABLED is true")
		}
		keyFile, err := resolveFilePath("MEERGO_HTTP_TLS_KEY_FILE")
		if err != nil {
			return nil, err
		}
		if keyFile == "" {
			return nil, fmt.Errorf("MEERGO_HTTP_TLS_KEY_FILE must be set when MEERGO_HTTP_TLS_ENABLED is true")
		}
		settings.HTTP.TLS.Enabled = true
		settings.HTTP.TLS.CertFile = certFile
		settings.HTTP.TLS.KeyFile = keyFile
	} else {
		if certFile := envVars.Get("MEERGO_HTTP_TLS_CERT_FILE"); certFile != "" {
			return nil, fmt.Errorf("MEERGO_HTTP_TLS_CERT_FILE must not be set when MEERGO_HTTP_TLS_ENABLED is false")
		}
		if keyFile := envVars.Get("MEERGO_HTTP_TLS_KEY_FILE"); keyFile != "" {
			return nil, fmt.Errorf("MEERGO_HTTP_TLS_KEY_FILE must not be set when MEERGO_HTTP_TLS_ENABLED is false")
		}
	}

	if externalURL, err := parseURL("MEERGO_HTTP_EXTERNAL_URL", noPath|noQuery); err != nil {
		return nil, err
	} else if externalURL == "" {
		protocol := "http"
		if settings.HTTP.TLS.Enabled {
			protocol = "https"
		}
		var addr string
		if port := settings.HTTP.Port; protocol == "http" && port == 80 || protocol == "https" && port == 443 {
			addr = settings.HTTP.Host
		} else {
			addr = net.JoinHostPort(settings.HTTP.Host, strconv.Itoa(settings.HTTP.Port))
		}
		settings.HTTP.ExternalURL = fmt.Sprintf("%s://%s/", protocol, addr)
	} else {
		settings.HTTP.ExternalURL = externalURL
	}

	if eventURL, err := parseURL("MEERGO_HTTP_EXTERNAL_EVENT_URL", noQuery); err != nil {
		return nil, err
	} else if eventURL == "" {
		settings.HTTP.ExternalEventURL = settings.HTTP.ExternalURL + "api/v1/events"
	} else {
		settings.HTTP.ExternalEventURL = eventURL
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

	if host := envVars.Get("MEERGO_DB_HOST"); host == "" {
		settings.DB.Host = "127.0.0.1"
	} else if err := validation.ValidateHost(host); err != nil {
		return nil, fmt.Errorf("MEERGO_DB_HOST must be a valid host: %s", err)
	} else {
		settings.DB.Host = host
	}

	if port := envVars.Get("MEERGO_DB_PORT"); port == "" {
		settings.DB.Port = 5432
	} else if port, err := validation.ValidatePortString(port); err != nil {
		return nil, fmt.Errorf("MEERGO_DB_PORT must be a valid port: %s", err)
	} else {
		settings.DB.Port = port
	}

	if username := envVars.Get("MEERGO_DB_USERNAME"); len(username) < 1 || len(username) > 63 {
		return nil, fmt.Errorf("invalid MEERGO_DB_USERNAME: length must be 1..63 bytes")
	} else {
		settings.DB.Username = username
	}

	settings.DB.Password = envVars.Get("MEERGO_DB_PASSWORD")
	if n := utf8.RuneCountInString(settings.DB.Password); n < 1 || n > 100 {
		return nil, fmt.Errorf("invalid MEERGO_DB_PASSWORD: length must be 1..100 characters")
	}

	settings.DB.Database = envVars.Get("MEERGO_DB_DATABASE")
	if n := len(settings.DB.Database); n < 1 || n > 63 {
		return nil, fmt.Errorf("invalid MEERGO_DB_DATABASE: length must be 1..63 bytes")
	}

	if schema := envVars.Get("MEERGO_DB_SCHEMA"); schema == "" {
		settings.DB.Schema = "public"
	} else if n := len(schema); n < 1 || n > 63 {
		return nil, fmt.Errorf("invalid MEERGO_DB_SCHEMA: length must be 1..63 bytes")
	} else {
		settings.DB.Schema = schema
	}

	if c := envVars.Get("MEERGO_DB_MAX_CONNECTIONS"); c == "" {
		settings.DB.MaxConnections = 8
	} else {
		maxConn, err := strconv.ParseInt(c, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("MEERGO_DB_MAX_CONNECTIONS must be an integer")
		}
		if maxConn < 2 {
			return nil, fmt.Errorf("MEERGO_DB_MAX_CONNECTIONS must be >= 2, got %d", maxConn)
		}
		settings.DB.MaxConnections = int32(maxConn)
	}

	settings.MemberEmailVerificationRequired, err = boolEnvVar(envVars.Get("MEERGO_MEMBER_EMAIL_VERIFICATION_REQUIRED"), true)
	if err != nil {
		return nil, fmt.Errorf("MEERGO_MEMBER_EMAIL_VERIFICATION_REQUIRED must be a boolean: %s", err)
	}
	settings.MemberEmailFrom = envVars.Get("MEERGO_MEMBER_EMAIL_FROM")

	if host := envVars.Get("MEERGO_SMTP_HOST"); host != "" {
		if err := validation.ValidateHost(host); err != nil {
			return nil, fmt.Errorf("MEERGO_SMTP_HOST must be a valid host: %s", err)
		}
		p := envVars.Get("MEERGO_SMTP_PORT")
		if p == "" {
			return nil, fmt.Errorf("MEERGO_SMTP_PORT is required if MEERGO_SMTP_HOST is set")
		}
		port, err := validation.ValidatePortString(p)
		if err != nil {
			return nil, fmt.Errorf("MEERGO_SMTP_PORT must be a valid port: %s", err)
		}
		settings.SMTP.Host = host
		settings.SMTP.Port = port
		settings.SMTP.Username = envVars.Get("MEERGO_SMTP_USERNAME")
		settings.SMTP.Password = envVars.Get("MEERGO_SMTP_PASSWORD")
	}

	if path, err := resolveFilePath("MEERGO_MAXMIND_DB_PATH"); err != nil {
		return nil, err
	} else if path != "" {
		settings.MaxMindDBPath = path
	}

	settings.Transformers.Lambda.AccessKeyID = envVars.Get("MEERGO_TRANSFORMERS_LAMBDA_ACCESS_KEY_ID")
	settings.Transformers.Lambda.SecretAccessKey = envVars.Get("MEERGO_TRANSFORMERS_LAMBDA_SECRET_ACCESS_KEY")
	settings.Transformers.Lambda.Region = envVars.Get("MEERGO_TRANSFORMERS_LAMBDA_REGION")
	settings.Transformers.Lambda.Role = envVars.Get("MEERGO_TRANSFORMERS_LAMBDA_ROLE")
	settings.Transformers.Lambda.Node.Runtime = envVars.Get("MEERGO_TRANSFORMERS_LAMBDA_NODE_RUNTIME")
	settings.Transformers.Lambda.Node.Layer = envVars.Get("MEERGO_TRANSFORMERS_LAMBDA_NODE_LAYER")
	settings.Transformers.Lambda.Python.Runtime = envVars.Get("MEERGO_TRANSFORMERS_LAMBDA_PYTHON_RUNTIME")
	settings.Transformers.Lambda.Python.Layer = envVars.Get("MEERGO_TRANSFORMERS_LAMBDA_PYTHON_LAYER")

	settings.Transformers.Local.NodeExecutable = envVars.Get("MEERGO_TRANSFORMERS_LOCAL_NODE_EXECUTABLE")
	settings.Transformers.Local.PythonExecutable = envVars.Get("MEERGO_TRANSFORMERS_LOCAL_PYTHON_EXECUTABLE")
	settings.Transformers.Local.FunctionsDir = envVars.Get("MEERGO_TRANSFORMERS_LOCAL_FUNCTIONS_DIR")
	if (settings.Transformers.Local.NodeExecutable != "" || settings.Transformers.Local.PythonExecutable != "") &&
		(settings.Transformers.Lambda.Node.Runtime != "" || settings.Transformers.Lambda.Python.Runtime != "") {
		return nil, fmt.Errorf("invalid configuration: cannot set both Lambda and local transformers")
	}

	if id := envVars.Get("MEERGO_OAUTH_HUBSPOT_CLIENT_ID"); id != "" {
		secret := envVars.Get("MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET is required when MEERGO_OAUTH_HUBSPOT_CLIENT_ID is set")
		}
		settings.OAuthCredentials = map[string]*core.OAuthCredentials{
			"hubspot": {
				ClientID:     id,
				ClientSecret: secret,
			}}
	} else if secret := envVars.Get("MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET"); secret != "" {
		return nil, fmt.Errorf("MEERGO_OAUTH_HUBSPOT_CLIENT_ID is required when MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET is set")
	}

	if id := envVars.Get("MEERGO_OAUTH_MAILCHIMP_CLIENT_ID"); id != "" {
		secret := envVars.Get("MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET is required when MEERGO_OAUTH_MAILCHIMP_CLIENT_ID is set")
		}
		if settings.OAuthCredentials == nil {
			settings.OAuthCredentials = make(map[string]*core.OAuthCredentials)
		}
		settings.OAuthCredentials["mailchimp"] = &core.OAuthCredentials{
			ClientID:     id,
			ClientSecret: secret,
		}
	} else if secret := envVars.Get("MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET"); secret != "" {
		return nil, fmt.Errorf("MEERGO_OAUTH_MAILCHIMP_CLIENT_ID is required when MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET is set")
	}

	return settings, nil
}

// boolEnvVar parses the value read from an environment variable as a boolean,
// returning either the value read (if valid) or an error.
func boolEnvVar(v string, defaultValue bool) (bool, error) {
	switch v {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "":
		return defaultValue, nil
	default:
		return false, fmt.Errorf("value %q is not a valid boolean value (expected true, false or empty string)", v)
	}
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

// parseURL parses the value of a configuration setting into a normalized URL.
// If the input string is empty, it returns an empty string.
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
	if err := validation.ValidateHost(u.Hostname()); err != nil {
		return "", errInvalidURL{key, err.Error()}
	}
	port := u.Port()
	if port != "" {
		if _, err := validation.ValidatePortString(port); err != nil {
			return "", errInvalidURL{key, err.Error()}
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

// resolveFilePath resolves a file configuration setting to its absolute path,
// returning an error if it does not exist or is not a regular file.
func resolveFilePath(key string) (string, error) {
	envVars, err := meergo.GetEnvVars()
	if err != nil {
		return "", err
	}
	s := envVars.Get(key)
	if s == "" {
		return "", nil
	}
	path, err := filepath.Abs(s)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path from %s: %s", key, err)
	}
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s points to a non-existent file: %q", key, path)
		}
		return "", fmt.Errorf("failed to stat file from %s %q: %s", key, path, err)
	}
	if st.IsDir() {
		return "", fmt.Errorf("%s points to a directory, not a regular file: %q", key, path)
	}
	if !st.Mode().IsRegular() {
		return "", fmt.Errorf("%s does not point to a regular file: %q", key, path)
	}
	return path, nil
}
