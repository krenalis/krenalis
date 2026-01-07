// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"crypto/ed25519"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/streams/nats"
	"github.com/meergo/meergo/tools/validation"
)

// parseEnvSettings parses the settings from the process environment variables.
//
// It does not alter the environment variables.
func parseEnvSettings() (*Settings, error) {

	envVars, err := connectors.GetEnvVars()
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

	if url, err := parseEnvURL("MEERGO_JAVASCRIPT_SDK_URL", 0); err != nil {
		return nil, fmt.Errorf("MEERGO_JAVASCRIPT_SDK_URL must be a valid URL: %s", err)
	} else if url == "" {
		settings.JavaScriptSDKURL = "https://cdn.meergo.com/meergo.min.js"
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

	settings.ExternalAssetsURLs = []string{}
	if assetsURLs := envVars.Get("MEERGO_EXTERNAL_ASSETS_URLS"); assetsURLs != "" {
		for url := range strings.SplitSeq(assetsURLs, ",") {
			url = strings.TrimSpace(url) // there may be spaces around commas.
			url, err := validation.ParseURL(url, validation.NoQuery)
			if err != nil {
				return nil, fmt.Errorf("invalid URL specified in environment variable MEERGO_EXTERNAL_ASSETS_URLS: %s", err)
			}
			if url != "" {
				settings.ExternalAssetsURLs = append(settings.ExternalAssetsURLs, url)
			}
		}
	}
	if len(settings.ExternalAssetsURLs) == 0 {
		settings.ExternalAssetsURLs = append(settings.ExternalAssetsURLs, "https://assets.meergo.com/")
	}

	switch potentialsURL := envVars.Get("MEERGO_POTENTIAL_CONNECTORS_URL"); potentialsURL {
	case "":
		settings.PotentialConnectorsURL = "https://assets.meergo.com/admin/connectors/potentials.json"
	case "none":
		settings.PotentialConnectorsURL = ""
	default:
		settings.PotentialConnectorsURL, err = validation.ParseURL(potentialsURL, 0)
		if err != nil {
			return nil, fmt.Errorf("invalid value specified for environment variable MEERGO_POTENTIAL_CONNECTORS_URL, which is neither empty, the string \"none\" nor a valid URL (%s)", err)
		}
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

	if externalURL, err := parseEnvURL("MEERGO_HTTP_EXTERNAL_URL", validation.NoPath|validation.NoQuery); err != nil {
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

	if eventURL, err := parseEnvURL("MEERGO_HTTP_EXTERNAL_EVENT_URL", validation.NoQuery); err != nil {
		return nil, err
	} else if eventURL == "" {
		settings.HTTP.ExternalEventURL = settings.HTTP.ExternalURL + "v1/events"
	} else {
		settings.HTTP.ExternalEventURL = eventURL
	}

	if settings.HTTP.ReadHeaderTimeout, err = parseEnvHTTPDuration("MEERGO_HTTP_READ_HEADER_TIMEOUT", 2*time.Second); err != nil {
		return nil, err
	}

	if settings.HTTP.ReadTimeout, err = parseEnvHTTPDuration("MEERGO_HTTP_READ_TIMEOUT", 5*time.Second); err != nil {
		return nil, err
	}

	if settings.HTTP.WriteTimeout, err = parseEnvHTTPDuration("MEERGO_HTTP_WRITE_TIMEOUT", 30*time.Second); err != nil {
		return nil, err
	}

	if settings.HTTP.IdleTimeout, err = parseEnvHTTPDuration("MEERGO_HTTP_IDLE_TIMEOUT", 120*time.Second); err != nil {
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

	if username, ok := envVars.Lookup("MEERGO_DB_USERNAME"); !ok {
		return nil, fmt.Errorf("environment variable MEERGO_DB_USERNAME is missing")
	} else if username == "" {
		return nil, fmt.Errorf("MEERGO_DB_USERNAME cannot be empty")
	} else if len(username) > 63 {
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

	if s := envVars.Get("MEERGO_NATS_URL"); s == "" {
		settings.NATS.Servers = []string{"nats://127.0.0.1:4222"}
	} else {
		var hasWS bool
		var hasNonWS bool
		settings.NATS.Servers = []string{}
		for entry := range strings.SplitSeq(s, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			if strings.Contains(entry, "://") {
				u, err := url.Parse(entry)
				if err != nil {
					return nil, fmt.Errorf("MEERGO_NATS_URL contains an invalid URL: %q", entry)
				}
				switch u.Scheme {
				case "nats", "tls":
					hasNonWS = true
				case "ws", "wss":
					hasWS = true
				default:
					return nil, fmt.Errorf("MEERGO_NATS_URL scheme %s is not allowed. Allowed schemes are nats, tls, ws, and wss", u.Scheme)
				}
			} else {
				if _, err := url.Parse("nats://" + entry); err != nil {
					return nil, fmt.Errorf("MEERGO_NATS_URL contains an invalid URL: %q", entry)
				}
				hasNonWS = true
			}
			if hasWS && hasNonWS {
				return nil, fmt.Errorf("MEERGO_NATS_URL contains both websocket and non-websocket URLs")
			}
			settings.NATS.Servers = append(settings.NATS.Servers, entry)
		}
		if len(settings.NATS.Servers) == 0 {
			return nil, fmt.Errorf("MEERGO_NATS_URL does not contain URLs")
		}
	}
	settings.NATS.User = envVars.Get("MEERGO_NATS_USER")
	if pw := envVars.Get("MEERGO_NATS_PASSWORD"); pw != "" {
		if settings.NATS.User == "" {
			return nil, fmt.Errorf("MEERGO_NATS_USER must be set if MEERGO_NATS_PASSWORD is provided")
		}
		settings.NATS.Password = pw
	}
	settings.NATS.Token = envVars.Get("MEERGO_NATS_TOKEN")
	if nkey := envVars.Get("MEERGO_NATS_NKEY"); nkey != "" {
		prefix, seed, err := nats.DecodeSeed([]byte(nkey))
		if err != nil || prefix != nats.PrefixByteUser || len(seed) != ed25519.SeedSize {
			return nil, fmt.Errorf("MEERGO_NATS_NKEY value is not a user NKey")
		}
		settings.NATS.NKey = ed25519.NewKeyFromSeed(seed)
	}
	switch storage := envVars.Get("MEERGO_NATS_STORAGE"); strings.ToLower(storage) {
	case "", "file":
		settings.NATS.Storage = nats.FileStorage
	case "memory":
		settings.NATS.Storage = nats.MemoryStorage
	default:
		return nil, fmt.Errorf("MEERGO_NATS_STORAGE value %q is not supported; expected file or memory", storage)
	}
	switch replicas := envVars.Get("MEERGO_NATS_REPLICAS"); replicas {
	case "", "1":
		settings.NATS.Replicas = 1
	case "2", "3", "4", "5":
		settings.NATS.Replicas = int(replicas[0] - '0')
	default:
		return nil, fmt.Errorf("MEERGO_NATS_REPLICAS value %q is not supported; expected 1, 2, 3, 4, or 5", replicas)
	}
	switch compression := envVars.Get("MEERGO_NATS_COMPRESSION"); strings.ToLower(compression) {
	case "":
		settings.NATS.Compression = nats.NoCompression
	case "s2":
		settings.NATS.Compression = nats.S2Compression
	default:
		return nil, fmt.Errorf("MEERGO_NATS_COMPRESSION value %q is not supported; expected s2", compression)
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

	settings.Transformers.Lambda.AccessKeyID = envVars.Get("MEERGO_TRANSFORMERS_AWS_LAMBDA_ACCESS_KEY_ID")
	settings.Transformers.Lambda.SecretAccessKey = envVars.Get("MEERGO_TRANSFORMERS_AWS_LAMBDA_SECRET_ACCESS_KEY")
	settings.Transformers.Lambda.Region = envVars.Get("MEERGO_TRANSFORMERS_AWS_LAMBDA_REGION")
	settings.Transformers.Lambda.Role = envVars.Get("MEERGO_TRANSFORMERS_AWS_LAMBDA_ROLE")
	settings.Transformers.Lambda.NodeJS.Runtime = envVars.Get("MEERGO_TRANSFORMERS_AWS_LAMBDA_NODEJS_RUNTIME")
	settings.Transformers.Lambda.NodeJS.Layer = envVars.Get("MEERGO_TRANSFORMERS_AWS_LAMBDA_NODEJS_LAYER")
	settings.Transformers.Lambda.Python.Runtime = envVars.Get("MEERGO_TRANSFORMERS_AWS_LAMBDA_PYTHON_RUNTIME")
	settings.Transformers.Lambda.Python.Layer = envVars.Get("MEERGO_TRANSFORMERS_AWS_LAMBDA_PYTHON_LAYER")

	settings.Transformers.Local.NodeJSExecutable = envVars.Get("MEERGO_TRANSFORMERS_LOCAL_NODEJS_EXECUTABLE")
	settings.Transformers.Local.PythonExecutable = envVars.Get("MEERGO_TRANSFORMERS_LOCAL_PYTHON_EXECUTABLE")
	settings.Transformers.Local.FunctionsDir = envVars.Get("MEERGO_TRANSFORMERS_LOCAL_FUNCTIONS_DIR")
	if (settings.Transformers.Local.NodeJSExecutable != "" || settings.Transformers.Local.PythonExecutable != "") &&
		(settings.Transformers.Lambda.NodeJS.Runtime != "" || settings.Transformers.Lambda.Python.Runtime != "") {
		return nil, fmt.Errorf("invalid configuration: cannot set both Lambda and local transformers")
	}
	settings.Transformers.Local.SudoUser = envVars.Get("MEERGO_TRANSFORMERS_LOCAL_SUDO_USER")
	settings.Transformers.Local.DoasUser = envVars.Get("MEERGO_TRANSFORMERS_LOCAL_DOAS_USER")
	if settings.Transformers.Local.SudoUser != "" && settings.Transformers.Local.DoasUser != "" {
		return nil, fmt.Errorf("cannot specify a value for both MEERGO_TRANSFORMERS_LOCAL_SUDO_USER" +
			" and MEERGO_TRANSFORMERS_LOCAL_DOAS_USER: you must specify one of the two, or neither")
	}

	settings.MetricsEnabled, err = boolEnvVar(envVars.Get("MEERGO_METRICS_ENABLED"), false)
	if err != nil {
		return nil, fmt.Errorf("MEERGO_METRICS_ENABLED must be a boolean: %s", err)
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

// parseEnvHTTPDuration parses the value of an HTTP configuration setting into a
// time.Duration.
func parseEnvHTTPDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	envVars, err := connectors.GetEnvVars()
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

// parseEnvURL parses the value of a configuration setting into a normalized
// URL. If the input string is empty, it returns an empty string.
func parseEnvURL(key string, flags validation.URLValidationFlag) (string, error) {
	envVars, err := connectors.GetEnvVars()
	if err != nil {
		return "", err
	}
	s := envVars.Get(key)
	u, err := validation.ParseURL(s, flags)
	if err != nil {
		return "", errInvalidURL{key, err.Error()}
	}
	return u, nil
}

// resolveFilePath resolves a file configuration setting to its absolute path,
// returning an error if it does not exist or is not a regular file.
func resolveFilePath(key string) (string, error) {
	envVars, err := connectors.GetEnvVars()
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
