// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/cmd/internal/config"
	"github.com/krenalis/krenalis/cmd/internal/config/aws"
	"github.com/krenalis/krenalis/cmd/internal/config/env"
	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/core/natsopts"
	"github.com/krenalis/krenalis/tools/validation"
)

type Config struct {
	KMS                    string
	OrganizationsAPIKey    string
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
			DNSNames []string
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
	OAuthCredentials              map[string]*core.OAuthCredentials // always empty (no connector currently uses OAuth).
	MaxQueuedEventsPerDestination int
}

type LambdaConfig struct {
	Role   string
	NodeJS struct {
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

// loadConfig parses the settings from the process environment variables.
//
// It does not alter the environment variables.
func loadConfig(ctx context.Context, source string) (*Config, error) {

	backend, path, found := strings.Cut(source, ":")
	if !found {
		return nil, errors.New("cmd: config source is invalid")
	}
	var err error
	var store config.Store
	switch backend {
	case "env":
		store = env.New()
	case "aws":
		store, err = aws.New(ctx, path)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("cmd: unsupported conf backend")
	}

	conf, err := store.Load(ctx)
	if err != nil {
		return nil, err
	}

	settings := &Config{}

	if kms, ok := conf.Lookup("KRENALIS_KMS"); ok {
		backend, options, found := strings.Cut(kms, ":")
		if !found {
			return nil, errors.New("KRENALIS_KMS must be in the form 'key:<base64>' or 'aws:<region>:<key-id>'")
		}
		switch backend {
		case "key":
			decodedValue, err := base64.RawStdEncoding.DecodeString(strings.TrimSuffix(options, "="))
			if err != nil {
				return nil, errors.New("KRENALIS_KMS key value is not valid base64")
			}
			if n := len(decodedValue); n != 32 {
				clear(decodedValue)
				return nil, fmt.Errorf("KRENALIS_KMS key value decodes to %d bytes, expected 32", n)
			}
		case "aws":
			if options == "" {
				return nil, errors.New("KRENALIS_KMS aws value is empty")
			}
		default:
			return nil, errors.New("KRENALIS_KMS must be in the form 'key:<base64>' or 'aws:<region>:<key-id>'")
		}
		settings.KMS = kms
	} else {
		return nil, errors.New("KRENALIS_KMS is not set")
	}

	if orgAPIKey, ok := conf.Lookup("KRENALIS_ORGANIZATIONS_API_KEY"); ok {
		apiKey, ok := strings.CutPrefix(orgAPIKey, "org_")
		if !ok {
			return nil, errors.New("KRENALIS_ORGANIZATIONS_API_KEY must start with 'org_'")
		}
		if utf8.RuneCountInString(apiKey) != 43 {
			return nil, fmt.Errorf("KRENALIS_ORGANIZATIONS_API_KEY has an invalid length (expected 'org_' + 43 alphanumeric characters)")
		}
		for _, c := range apiKey {
			switch {
			case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
			default:
				return nil, fmt.Errorf("invalid format of KRENALIS_ORGANIZATIONS_API_KEY, unexpected character %q", c)
			}
		}
		settings.OrganizationsAPIKey = orgAPIKey
	}

	if delay := conf.Get("KRENALIS_TERMINATION_DELAY"); delay != "" {
		delay, err := time.ParseDuration(delay)
		if err != nil {
			return nil, fmt.Errorf("invalid duration value specified for KRENALIS_TERMINATION_DELAY: %s", err)
		}
		settings.TerminationDelay = delay
	}

	if url, err := parseEnvURL(conf, "KRENALIS_JAVASCRIPT_SDK_URL", 0); err != nil {
		return nil, fmt.Errorf("KRENALIS_JAVASCRIPT_SDK_URL must be a valid URL: %s", err)
	} else if url == "" {
		settings.JavaScriptSDKURL = "https://cdn.krenalis.com/krenalis.min.js"
	} else {
		settings.JavaScriptSDKURL = url
	}

	switch conf.Get("KRENALIS_TELEMETRY_LEVEL") {
	case "none":
		settings.SentryTelemetryLevel = core.TelemetryLevelNone
	case "errors":
		settings.SentryTelemetryLevel = core.TelemetryLevelErrors
	case "stats":
		settings.SentryTelemetryLevel = core.TelemetryLevelStats
	case "", "all":
		settings.SentryTelemetryLevel = core.TelemetryLevelAll
	default:
		return nil, fmt.Errorf("invalid KRENALIS_TELEMETRY_LEVEL: want one of none, errors, stats, or all")
	}

	settings.ExternalAssetsURLs = []string{}
	if assetsURLs := conf.Get("KRENALIS_EXTERNAL_ASSETS_URLS"); assetsURLs != "" {
		for url := range strings.SplitSeq(assetsURLs, ",") {
			url = strings.TrimSpace(url) // there may be spaces around commas.
			url, err := validation.ParseURL(url, validation.NoQuery)
			if err != nil {
				return nil, fmt.Errorf("invalid URL specified in environment variable KRENALIS_EXTERNAL_ASSETS_URLS: %s", err)
			}
			if url != "" {
				settings.ExternalAssetsURLs = append(settings.ExternalAssetsURLs, url)
			}
		}
	}
	if len(settings.ExternalAssetsURLs) == 0 {
		settings.ExternalAssetsURLs = append(settings.ExternalAssetsURLs, "https://assets.krenalis.com/")
	}

	switch potentialsURL := conf.Get("KRENALIS_POTENTIAL_CONNECTORS_URL"); potentialsURL {
	case "":
		settings.PotentialConnectorsURL = "https://assets.krenalis.com/admin/connectors/potentials.json"
	case "none":
		settings.PotentialConnectorsURL = ""
	default:
		settings.PotentialConnectorsURL, err = validation.ParseURL(potentialsURL, 0)
		if err != nil {
			return nil, fmt.Errorf("invalid value specified for environment variable KRENALIS_POTENTIAL_CONNECTORS_URL, which is neither empty, the string \"none\" nor a valid URL (%s)", err)
		}
	}

	if host := conf.Get("KRENALIS_HTTP_HOST"); host == "" {
		settings.HTTP.Host = "127.0.0.1"
	} else if err := validation.ValidateHost(host); err != nil {
		return nil, fmt.Errorf("KRENALIS_HTTP_HOST must be a valid host: %s", err)
	} else {
		settings.HTTP.Host = host
	}

	if port := conf.Get("KRENALIS_HTTP_PORT"); port == "" {
		settings.HTTP.Port = 2022
	} else if port, err := validation.ValidatePortString(port); err != nil {
		return nil, fmt.Errorf("KRENALIS_HTTP_PORT must be a valid port: %s", err)
	} else {
		settings.HTTP.Port = port
	}

	if tls, err := boolEnvVar(conf.Get("KRENALIS_HTTP_TLS_ENABLED"), false); err != nil {
		return nil, fmt.Errorf("KRENALIS_HTTP_TLS_ENABLED must be a boolean: %s", err)
	} else if tls {
		certFile, err := resolveFilePath(conf, "KRENALIS_HTTP_TLS_CERT_FILE")
		if err != nil {
			return nil, err
		}
		if certFile == "" {
			return nil, fmt.Errorf("KRENALIS_HTTP_TLS_CERT_FILE must be set when KRENALIS_HTTP_TLS_ENABLED is true")
		}
		keyFile, err := resolveFilePath(conf, "KRENALIS_HTTP_TLS_KEY_FILE")
		if err != nil {
			return nil, err
		}
		if keyFile == "" {
			return nil, fmt.Errorf("KRENALIS_HTTP_TLS_KEY_FILE must be set when KRENALIS_HTTP_TLS_ENABLED is true")
		}
		settings.HTTP.TLS.Enabled = true
		settings.HTTP.TLS.CertFile = certFile
		settings.HTTP.TLS.KeyFile = keyFile
		// Set settings.HTTP.TLS.DNSNames.
		if names := conf.Get("KRENALIS_HTTP_TLS_DNS_NAMES"); names != "" {
			if dnsNames := strings.Split(names, ","); len(dnsNames) > 0 {
				for i, name := range dnsNames {
					name = strings.TrimSpace(name) // there may be spaces around commas.
					if err := validation.ValidateHost(name); err != nil {
						return nil, fmt.Errorf("KRENALIS_HTTP_TLS_DNS_NAMES contains an invalid DNS name: %s", name)
					}
					dnsNames[i] = strings.ToLower(name)
				}
				slices.Sort(dnsNames)
				settings.HTTP.TLS.DNSNames = slices.Compact(dnsNames)
			}
		}
	} else {
		if certFile := conf.Get("KRENALIS_HTTP_TLS_CERT_FILE"); certFile != "" {
			return nil, fmt.Errorf("KRENALIS_HTTP_TLS_CERT_FILE must not be set when KRENALIS_HTTP_TLS_ENABLED is false")
		}
		if keyFile := conf.Get("KRENALIS_HTTP_TLS_KEY_FILE"); keyFile != "" {
			return nil, fmt.Errorf("KRENALIS_HTTP_TLS_KEY_FILE must not be set when KRENALIS_HTTP_TLS_ENABLED is false")
		}
		if names := conf.Get("KRENALIS_HTTP_TLS_DNS_NAMES"); names != "" {
			return nil, fmt.Errorf("KRENALIS_HTTP_TLS_DNS_NAMES must not be set when KRENALIS_HTTP_TLS_ENABLED is false")
		}
	}

	if externalURL, err := parseEnvURL(conf, "KRENALIS_HTTP_EXTERNAL_URL", validation.NoPath|validation.NoQuery); err != nil {
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

	if eventURL, err := parseEnvURL(conf, "KRENALIS_HTTP_EXTERNAL_EVENT_URL", validation.NoQuery); err != nil {
		return nil, err
	} else if eventURL == "" {
		settings.HTTP.ExternalEventURL = settings.HTTP.ExternalURL + "v1/events"
	} else {
		settings.HTTP.ExternalEventURL = eventURL
	}

	// Set settings.HTTP.TLS.DNSNames from ExternalURL and ExternalEventURL.
	if settings.HTTP.TLS.Enabled && settings.HTTP.TLS.DNSNames == nil {
		dnsNames := make([]string, 0, 2)
		if name, ok := httpsHost(settings.HTTP.ExternalURL); ok {
			dnsNames = append(dnsNames, name)
		}
		if name, ok := httpsHost(settings.HTTP.ExternalEventURL); ok && (len(dnsNames) == 0 || name != dnsNames[0]) {
			dnsNames = append(dnsNames, name)
		}
		settings.HTTP.TLS.DNSNames = dnsNames
	}

	if settings.HTTP.ReadHeaderTimeout, err = parseEnvHTTPDuration(conf, "KRENALIS_HTTP_READ_HEADER_TIMEOUT", 2*time.Second); err != nil {
		return nil, err
	}

	if settings.HTTP.ReadTimeout, err = parseEnvHTTPDuration(conf, "KRENALIS_HTTP_READ_TIMEOUT", 5*time.Second); err != nil {
		return nil, err
	}

	if settings.HTTP.WriteTimeout, err = parseEnvHTTPDuration(conf, "KRENALIS_HTTP_WRITE_TIMEOUT", 30*time.Second); err != nil {
		return nil, err
	}

	if settings.HTTP.IdleTimeout, err = parseEnvHTTPDuration(conf, "KRENALIS_HTTP_IDLE_TIMEOUT", 120*time.Second); err != nil {
		return nil, err
	}

	if host := conf.Get("KRENALIS_DB_HOST"); host == "" {
		settings.DB.Host = "127.0.0.1"
	} else if err := validation.ValidateHost(host); err != nil {
		return nil, fmt.Errorf("KRENALIS_DB_HOST must be a valid host: %s", err)
	} else {
		settings.DB.Host = host
	}

	if port := conf.Get("KRENALIS_DB_PORT"); port == "" {
		settings.DB.Port = 5432
	} else if port, err := validation.ValidatePortString(port); err != nil {
		return nil, fmt.Errorf("KRENALIS_DB_PORT must be a valid port: %s", err)
	} else {
		settings.DB.Port = port
	}

	if username, ok := conf.Lookup("KRENALIS_DB_USERNAME"); !ok {
		return nil, fmt.Errorf("environment variable KRENALIS_DB_USERNAME is missing")
	} else if username == "" {
		return nil, fmt.Errorf("KRENALIS_DB_USERNAME cannot be empty")
	} else if len(username) > 63 {
		return nil, fmt.Errorf("invalid KRENALIS_DB_USERNAME: length must be 1..63 bytes")
	} else {
		settings.DB.Username = username
	}

	settings.DB.Password = conf.Get("KRENALIS_DB_PASSWORD")
	if n := utf8.RuneCountInString(settings.DB.Password); n > 100 {
		return nil, fmt.Errorf("invalid KRENALIS_DB_PASSWORD: length must be a maximum of 100 characters")
	}

	settings.DB.Database = conf.Get("KRENALIS_DB_DATABASE")
	if n := len(settings.DB.Database); n < 1 || n > 63 {
		return nil, fmt.Errorf("invalid KRENALIS_DB_DATABASE: length must be 1..63 bytes")
	}

	if schema := conf.Get("KRENALIS_DB_SCHEMA"); schema == "" {
		settings.DB.Schema = "public"
	} else if n := len(schema); n < 1 || n > 63 {
		return nil, fmt.Errorf("invalid KRENALIS_DB_SCHEMA: length must be 1..63 bytes")
	} else {
		settings.DB.Schema = schema
	}

	if c := conf.Get("KRENALIS_DB_MAX_CONNECTIONS"); c == "" {
		settings.DB.MaxConnections = 8
	} else {
		maxConn, err := strconv.ParseInt(c, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("KRENALIS_DB_MAX_CONNECTIONS must be an integer")
		}
		if maxConn < 2 {
			return nil, fmt.Errorf("KRENALIS_DB_MAX_CONNECTIONS must be >= 2, got %d", maxConn)
		}
		settings.DB.MaxConnections = int32(maxConn)
	}

	if s := conf.Get("KRENALIS_NATS_URL"); s == "" {
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
					return nil, fmt.Errorf("KRENALIS_NATS_URL contains an invalid URL: %q", entry)
				}
				switch u.Scheme {
				case "nats", "tls":
					hasNonWS = true
				case "ws", "wss":
					hasWS = true
				default:
					return nil, fmt.Errorf("KRENALIS_NATS_URL scheme %s is not allowed. Allowed schemes are nats, tls, ws, and wss", u.Scheme)
				}
			} else {
				if _, err := url.Parse("nats://" + entry); err != nil {
					return nil, fmt.Errorf("KRENALIS_NATS_URL contains an invalid URL: %q", entry)
				}
				hasNonWS = true
			}
			if hasWS && hasNonWS {
				return nil, fmt.Errorf("KRENALIS_NATS_URL contains both websocket and non-websocket URLs")
			}
			settings.NATS.Servers = append(settings.NATS.Servers, entry)
		}
		if len(settings.NATS.Servers) == 0 {
			return nil, fmt.Errorf("KRENALIS_NATS_URL does not contain URLs")
		}
	}
	settings.NATS.User = conf.Get("KRENALIS_NATS_USER")
	if pw := conf.Get("KRENALIS_NATS_PASSWORD"); pw != "" {
		if settings.NATS.User == "" {
			return nil, fmt.Errorf("KRENALIS_NATS_USER must be set if KRENALIS_NATS_PASSWORD is provided")
		}
		settings.NATS.Password = pw
	}
	settings.NATS.Token = conf.Get("KRENALIS_NATS_TOKEN")
	if nkey := conf.Get("KRENALIS_NATS_NKEY"); nkey != "" {
		prefix, seed, err := natsopts.DecodeSeed([]byte(nkey))
		if err != nil || prefix != natsopts.PrefixByteUser || len(seed) != ed25519.SeedSize {
			return nil, fmt.Errorf("KRENALIS_NATS_NKEY value is not a user NKey")
		}
		settings.NATS.NKey = ed25519.NewKeyFromSeed(seed)
	}
	switch storage := conf.Get("KRENALIS_NATS_STORAGE"); strings.ToLower(storage) {
	case "", "file":
		settings.NATS.Storage = natsopts.FileStorage
	case "memory":
		settings.NATS.Storage = natsopts.MemoryStorage
	default:
		return nil, fmt.Errorf("KRENALIS_NATS_STORAGE value %q is not supported; expected file or memory", storage)
	}
	switch replicas := conf.Get("KRENALIS_NATS_REPLICAS"); replicas {
	case "", "1":
		settings.NATS.Replicas = 1
	case "2", "3", "4", "5":
		settings.NATS.Replicas = int(replicas[0] - '0')
	default:
		return nil, fmt.Errorf("KRENALIS_NATS_REPLICAS value %q is not supported; expected 1, 2, 3, 4, or 5", replicas)
	}
	switch compression := conf.Get("KRENALIS_NATS_COMPRESSION"); strings.ToLower(compression) {
	case "":
		settings.NATS.Compression = natsopts.NoCompression
	case "s2":
		settings.NATS.Compression = natsopts.S2Compression
	default:
		return nil, fmt.Errorf("KRENALIS_NATS_COMPRESSION value %q is not supported; expected s2", compression)
	}
	if settings.NATS.Compression != natsopts.NoCompression && settings.NATS.Storage != natsopts.FileStorage {
		return nil, errors.New("KRENALIS_NATS_COMPRESSION can be set only when using file storage")
	}

	settings.InviteMembersViaEmail, err = boolEnvVar(conf.Get("KRENALIS_INVITE_MEMBERS_VIA_EMAIL"), false)
	if err != nil {
		return nil, fmt.Errorf("KRENALIS_INVITE_MEMBERS_VIA_EMAIL must be a boolean: %s", err)
	}
	settings.MemberEmailFrom = conf.Get("KRENALIS_MEMBER_EMAIL_FROM")

	if host := conf.Get("KRENALIS_SMTP_HOST"); host != "" {
		if err := validation.ValidateHost(host); err != nil {
			return nil, fmt.Errorf("KRENALIS_SMTP_HOST must be a valid host: %s", err)
		}
		p := conf.Get("KRENALIS_SMTP_PORT")
		if p == "" {
			return nil, fmt.Errorf("KRENALIS_SMTP_PORT is required if KRENALIS_SMTP_HOST is set")
		}
		port, err := validation.ValidatePortString(p)
		if err != nil {
			return nil, fmt.Errorf("KRENALIS_SMTP_PORT must be a valid port: %s", err)
		}
		settings.SMTP.Host = host
		settings.SMTP.Port = port
		settings.SMTP.Username = conf.Get("KRENALIS_SMTP_USERNAME")
		settings.SMTP.Password = conf.Get("KRENALIS_SMTP_PASSWORD")
	}

	if path, err := resolveFilePath(conf, "KRENALIS_MAXMIND_DB_PATH"); err != nil {
		return nil, err
	} else if path != "" {
		settings.MaxMindDBPath = path
	}

	switch strings.ToLower(conf.Get("KRENALIS_TRANSFORMERS_PROVIDER")) {
	case "":
	case "local":
		settings.Transformers.Local.NodeJSExecutable = conf.Get("KRENALIS_TRANSFORMERS_LOCAL_NODEJS_EXECUTABLE")
		settings.Transformers.Local.PythonExecutable = conf.Get("KRENALIS_TRANSFORMERS_LOCAL_PYTHON_EXECUTABLE")
		settings.Transformers.Local.FunctionsDir = conf.Get("KRENALIS_TRANSFORMERS_LOCAL_FUNCTIONS_DIR")
		settings.Transformers.Local.SudoUser = conf.Get("KRENALIS_TRANSFORMERS_LOCAL_SUDO_USER")
		settings.Transformers.Local.DoasUser = conf.Get("KRENALIS_TRANSFORMERS_LOCAL_DOAS_USER")
	case "aws-lambda":
		settings.Transformers.Lambda.Role = conf.Get("KRENALIS_TRANSFORMERS_AWS_LAMBDA_ROLE")
		settings.Transformers.Lambda.NodeJS.Runtime = conf.Get("KRENALIS_TRANSFORMERS_AWS_LAMBDA_NODEJS_RUNTIME")
		settings.Transformers.Lambda.NodeJS.Layer = conf.Get("KRENALIS_TRANSFORMERS_AWS_LAMBDA_NODEJS_LAYER")
		settings.Transformers.Lambda.Python.Runtime = conf.Get("KRENALIS_TRANSFORMERS_AWS_LAMBDA_PYTHON_RUNTIME")
		settings.Transformers.Lambda.Python.Layer = conf.Get("KRENALIS_TRANSFORMERS_AWS_LAMBDA_PYTHON_LAYER")
	default:
		return nil, fmt.Errorf("invalid KRENALIS_TRANSFORMERS_PROVIDER: want one of local or aws-lambda")
	}
	if conf.Get("KRENALIS_TRANSFORMERS_LOCAL_SUDO_USER") != "" && conf.Get("KRENALIS_TRANSFORMERS_LOCAL_DOAS_USER") != "" {
		return nil, fmt.Errorf("cannot specify a value for both KRENALIS_TRANSFORMERS_LOCAL_SUDO_USER" +
			" and KRENALIS_TRANSFORMERS_LOCAL_DOAS_USER: you must specify one of the two, or neither")
	}

	settings.PrometheusMetricsEnabled, err = boolEnvVar(conf.Get("KRENALIS_PROMETHEUS_METRICS_ENABLED"), false)
	if err != nil {
		return nil, fmt.Errorf("KRENALIS_PROMETHEUS_METRICS_ENABLED must be a boolean: %s", err)
	}

	if e := conf.Get("KRENALIS_MAX_QUEUED_EVENTS_PER_DESTINATION"); e == "" {
		settings.MaxQueuedEventsPerDestination = 50_000
	} else {
		maxEvents, err := strconv.ParseInt(e, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("KRENALIS_MAX_QUEUED_EVENTS_PER_DESTINATION must be an integer")
		}
		if maxEvents < 1 {
			return nil, fmt.Errorf("KRENALIS_MAX_QUEUED_EVENTS_PER_DESTINATION must be >= 1, got %d", maxEvents)
		}
		settings.MaxQueuedEventsPerDestination = int(maxEvents)
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

// httpsHost extracts the host from an HTTPS URL and reports whether it
// succeeded.
func httpsHost(u string) (string, bool) {
	const prefix = "https://"
	if !strings.HasPrefix(u, prefix) {
		return "", false
	}
	s := u[len(prefix):]
	// Trim path, query, and fragment.
	if i := strings.IndexAny(s, "/?#"); i != -1 {
		s = s[:i]
	}
	// Handle bracketed IPv6 addresses with an optional port.
	if strings.HasPrefix(s, "[") {
		if i := strings.Index(s, "]"); i != -1 {
			return s[1:i], true // inside brackets
		}
		return "", false // malformed
	}
	// Strip the port from hostnames and IPv4 addresses.
	if i := strings.LastIndex(s, ":"); i != -1 {
		return s[:i], true
	}
	return s, true
}

// parseEnvHTTPDuration parses the value of an HTTP configuration setting into a
// time.Duration.
func parseEnvHTTPDuration(conf config.Config, key string, defaultValue time.Duration) (time.Duration, error) {
	s := conf.Get(key)
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
func parseEnvURL(conf config.Config, key string, flags validation.URLValidationFlag) (string, error) {
	s := conf.Get(key)
	u, err := validation.ParseURL(s, flags)
	if err != nil {
		return "", errInvalidURL{key, err.Error()}
	}
	return u, nil
}

// resolveFilePath resolves a file configuration setting to its absolute path,
// returning an error if it does not exist or is not a regular file.
func resolveFilePath(conf config.Config, key string) (string, error) {
	s := conf.Get(key)
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
