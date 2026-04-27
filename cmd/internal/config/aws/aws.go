// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package aws loads configuration settings from AWS Systems Manager Parameter
// Store.
//
// The package reads supported parameters recursively under a common path
// prefix, for example "/prod", in the configured AWS region. Each supported
// setting is mapped from a Parameter Store path such as:
//
//	/prod/http/host
//	/prod/http/port
//	/prod/db/schema
//
// Only Krenalis's parameter paths are recognized.
//
// Database connection credentials can also be loaded from the optional Secrets
// Manager reference exposed by Parameter Store at:
//
//	/aws/reference/secretsmanager/<prefix>/db
//
// The referenced secret must contain the standard AWS JSON structure for a
// PostgreSQL database secret. It populates DB_HOST, DB_PORT, DB_DATABASE,
// DB_USERNAME, and DB_PASSWORD.
//
// Database credentials must be configured either through /db/... Parameter
// Store entries or through the /db Secrets Manager reference; mixing the two
// sources is rejected.
package aws

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/cmd/internal/config"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type Store struct {
	client      ssmClient
	prefix      string
	secretNames []string
}

type ssmClient interface {
	GetParametersByPath(context.Context, *ssm.GetParametersByPathInput, ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
	GetParameters(context.Context, *ssm.GetParametersInput, ...func(*ssm.Options)) (*ssm.GetParametersOutput, error)
}

func normalizePrefix(prefix string) string {
	return strings.TrimRight(prefix, "/")
}

func New(ctx context.Context, options string) (*Store, error) {
	region, prefix, err := validateOptions(options)
	if err != nil {
		return nil, fmt.Errorf("config/aws: %s", err)
	}
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("config/aws: %s", err)
	}
	prefix = normalizePrefix(prefix)
	return &Store{
		client:      ssm.NewFromConfig(cfg),
		prefix:      prefix,
		secretNames: []string{"/aws/reference/secretsmanager" + prefix + "/db"},
	}, nil
}

func (s *Store) Load(ctx context.Context) (config.Config, error) {
	values, err := s.loadParametersByPath(ctx)
	if err != nil {
		return nil, err
	}
	err = s.loadSecretsManagerReferences(ctx, values)
	if err != nil {
		return nil, err
	}
	return &Config{values: values}, nil
}

var truePtr = new(true)

// loadParametersByPath loads Parameter Store values under the configured
// prefix.
func (s *Store) loadParametersByPath(ctx context.Context) (map[string]string, error) {
	values := map[string]string{}
	input := &ssm.GetParametersByPathInput{
		Path:           aws.String(s.prefix),
		Recursive:      truePtr,
		WithDecryption: truePtr,
	}
	pageCount := 1
	for {
		out, err := s.client.GetParametersByPath(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("config/aws: %s", err)
		}
		for _, p := range out.Parameters {
			name, value, err := s.normalizeParameter(p)
			if err != nil {
				return nil, err
			}
			values[name] = value
		}
		if out.NextToken == nil || *out.NextToken == "" {
			break
		}
		if pageCount == 10 {
			return nil, fmt.Errorf("config/aws: too many parameters under prefix %q", s.prefix)
		}
		input.NextToken = out.NextToken
		pageCount++
	}
	return values, nil
}

var dbSecretNames = []string{"DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD"}

// loadSecretsManagerReferences loads supported Secrets Manager references into
// values.
func (s *Store) loadSecretsManagerReferences(ctx context.Context, values map[string]string) error {
	out, err := s.client.GetParameters(ctx, &ssm.GetParametersInput{
		Names:          s.secretNames,
		WithDecryption: truePtr,
	})
	if err != nil {
		return fmt.Errorf("config/aws: %s", err)
	}
	if len(out.Parameters) > len(s.secretNames) {
		return errors.New("config/aws: AWS has returned more secret parameters than expected")
	}
	for _, p := range out.Parameters {
		if p.Name == nil {
			return errors.New("config/aws: AWS has returned a nil parameter name")
		}
		if !slices.Contains(s.secretNames, *p.Name) {
			return fmt.Errorf("config/aws: AWS has returned an unexpected secret parameter %q", *p.Name)
		}
		if p.Value == nil {
			return errors.New("config/aws: AWS has returned a nil parameter value")
		}
		if strings.HasSuffix(*p.Name, "/db") {
			// Reject mixed database parameters.
			for _, name := range dbSecretNames {
				if _, ok := values[name]; ok {
					return fmt.Errorf("config/aws: both the '%s/db' Secrets Manager secret and '%s/db/...' parameters are configured; only one database configuration source is allowed", s.prefix, s.prefix)
				}
			}
			err := unmarshalDatabaseSecret(values, *p.Value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// normalizeParameter maps a Parameter Store entry to its internal config key.
func (s *Store) normalizeParameter(p types.Parameter) (string, string, error) {
	if p.Name == nil {
		return "", "", errors.New("config/aws: AWS has returned a nil parameter name")
	}
	if p.Value == nil {
		return "", "", errors.New("config/aws: AWS has returned a nil parameter value")
	}
	name := *p.Name
	if s.prefix != "" {
		var ok bool
		name, ok = strings.CutPrefix(name, s.prefix)
		if !ok {
			return "", "", fmt.Errorf("config/aws: AWS has returned an unexpected parameter name %q", *p.Name)
		}
	}
	envName, ok := parameters[name]
	if !ok {
		return "", "", fmt.Errorf("config/aws: AWS has returned an unexpected parameter name %q", *p.Name)
	}
	return envName, *p.Value, nil
}

// unmarshalDatabaseSecret decodes a database secret into config values.
func unmarshalDatabaseSecret(values map[string]string, secret string) error {
	var db struct {
		Engine   string `json:"engine"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		DBName   string `json:"dbname"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal([]byte(secret), &db); err != nil {
		return fmt.Errorf("config/aws: invalid '/db' secret JSON: %s", err)
	}
	if db.Engine != "postgres" {
		return fmt.Errorf("config/aws: invalid '/db' secret engine %q", db.Engine)
	}
	values["DB_HOST"] = db.Host
	if db.Port != 0 {
		values["DB_PORT"] = strconv.Itoa(db.Port)
	}
	values["DB_DATABASE"] = db.DBName
	values["DB_USERNAME"] = db.Username
	values["DB_PASSWORD"] = db.Password
	return nil
}

type Config struct {
	values map[string]string
}

func (c *Config) Get(name string) string {
	name, found := strings.CutPrefix(name, "KRENALIS_")
	if !found {
		return ""
	}
	return c.values[name]
}

func (c *Config) Lookup(name string) (string, bool) {
	name, found := strings.CutPrefix(name, "KRENALIS_")
	if !found {
		return "", false
	}
	v, ok := c.values[name]
	return v, ok
}

// validateOptions validates and splits AWS store options.
func validateOptions(options string) (string, string, error) {
	region, prefix, found := strings.Cut(options, ":")
	if !found {
		return "", "", errors.New("options must be in the form '<region>:<prefix>'")
	}
	if err := validateRegion(region); err != nil {
		return "", "", err
	}
	if err := validatePrefix(prefix); err != nil {
		return "", "", err
	}
	return region, prefix, nil
}

func validatePrefix(prefix string) error {
	if prefix == "" {
		return errors.New("prefix must not be empty")
	}
	if prefix == "/" {
		return errors.New("prefix must not be '/'")
	}
	prefix = normalizePrefix(prefix)
	if !strings.HasPrefix(prefix, "/") {
		return errors.New("prefix must start with '/'")
	}
	if strings.Contains(prefix, "//") {
		return errors.New("prefix must not contain empty path elements")
	}
	for _, r := range prefix {
		switch {
		case r == '/':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_', r == '.', r == '-':
		default:
			return fmt.Errorf("prefix contains invalid character %q", r)
		}
	}
	return nil
}

func validateRegion(region string) error {
	if region == "" {
		return errors.New("region must not be empty")
	}
	for _, r := range region {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return errors.New("region must be an AWS region code such as 'us-east-1'")
		}
	}
	return nil
}

var parameters = map[string]string{
	"/db/database":                            "DB_DATABASE",
	"/db/host":                                "DB_HOST",
	"/db/max-connections":                     "DB_MAX_CONNECTIONS",
	"/db/password":                            "DB_PASSWORD",
	"/db/port":                                "DB_PORT",
	"/db/schema":                              "DB_SCHEMA",
	"/db/username":                            "DB_USERNAME",
	"/external-assets-urls":                   "EXTERNAL_ASSETS_URLS",
	"/http/external-event-url":                "HTTP_EXTERNAL_EVENT_URL",
	"/http/external-url":                      "HTTP_EXTERNAL_URL",
	"/http/host":                              "HTTP_HOST",
	"/http/idle-timeout":                      "HTTP_IDLE_TIMEOUT",
	"/http/port":                              "HTTP_PORT",
	"/http/read-header-timeout":               "HTTP_READ_HEADER_TIMEOUT",
	"/http/read-timeout":                      "HTTP_READ_TIMEOUT",
	"/http/tls/cert-file":                     "HTTP_TLS_CERT_FILE",
	"/http/tls/dns-names":                     "HTTP_TLS_DNS_NAMES",
	"/http/tls/enabled":                       "HTTP_TLS_ENABLED",
	"/http/tls/key-file":                      "HTTP_TLS_KEY_FILE",
	"/http/write-timeout":                     "HTTP_WRITE_TIMEOUT",
	"/invite-members-via-email":               "INVITE_MEMBERS_VIA_EMAIL",
	"/javascript-sdk-url":                     "JAVASCRIPT_SDK_URL",
	"/kms":                                    "KMS",
	"/max-queued-events-per-destination":      "MAX_QUEUED_EVENTS_PER_DESTINATION",
	"/maxmind-db-path":                        "MAXMIND_DB_PATH",
	"/member-email-from":                      "MEMBER_EMAIL_FROM",
	"/nats/compression":                       "NATS_COMPRESSION",
	"/nats/nkey":                              "NATS_NKEY",
	"/nats/password":                          "NATS_PASSWORD",
	"/nats/replicas":                          "NATS_REPLICAS",
	"/nats/storage":                           "NATS_STORAGE",
	"/nats/token":                             "NATS_TOKEN",
	"/nats/url":                               "NATS_URL",
	"/nats/user":                              "NATS_USER",
	"/organizations-api-key":                  "ORGANIZATIONS_API_KEY",
	"/potential-connectors-url":               "POTENTIAL_CONNECTORS_URL",
	"/prometheus-metrics-enabled":             "PROMETHEUS_METRICS_ENABLED",
	"/smtp/host":                              "SMTP_HOST",
	"/smtp/password":                          "SMTP_PASSWORD",
	"/smtp/port":                              "SMTP_PORT",
	"/smtp/username":                          "SMTP_USERNAME",
	"/telemetry-level":                        "TELEMETRY_LEVEL",
	"/termination-delay":                      "TERMINATION_DELAY",
	"/transformers/aws-lambda/nodejs/layer":   "TRANSFORMERS_AWS_LAMBDA_NODEJS_LAYER",
	"/transformers/aws-lambda/nodejs/runtime": "TRANSFORMERS_AWS_LAMBDA_NODEJS_RUNTIME",
	"/transformers/aws-lambda/python/layer":   "TRANSFORMERS_AWS_LAMBDA_PYTHON_LAYER",
	"/transformers/aws-lambda/python/runtime": "TRANSFORMERS_AWS_LAMBDA_PYTHON_RUNTIME",
	"/transformers/aws-lambda/role":           "TRANSFORMERS_AWS_LAMBDA_ROLE",
	"/transformers/local/doas-user":           "TRANSFORMERS_LOCAL_DOAS_USER",
	"/transformers/local/functions-dir":       "TRANSFORMERS_LOCAL_FUNCTIONS_DIR",
	"/transformers/local/nodejs/executable":   "TRANSFORMERS_LOCAL_NODEJS_EXECUTABLE",
	"/transformers/local/python/executable":   "TRANSFORMERS_LOCAL_PYTHON_EXECUTABLE",
	"/transformers/local/sudo-user":           "TRANSFORMERS_LOCAL_SUDO_USER",
	"/transformers/provider":                  "TRANSFORMERS_PROVIDER",
}
