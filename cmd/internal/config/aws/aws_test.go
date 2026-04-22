// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package aws

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func TestConfigGetAndLookup(t *testing.T) {
	t.Parallel()

	c := &Config{
		values: map[string]string{
			"DB_HOST": "db.example.internal",
		},
	}

	if got := c.Get("KRENALIS_DB_HOST"); got != "db.example.internal" {
		t.Fatalf("Get(KRENALIS_DB_HOST) = %q, want %q", got, "db.example.internal")
	}

	if got := c.Get("DB_HOST"); got != "" {
		t.Fatalf("Get(DB_HOST) = %q, want empty string", got)
	}

	if got := c.Get("KRENALIS_DB_PORT"); got != "" {
		t.Fatalf("Get(KRENALIS_DB_PORT) = %q, want empty string", got)
	}

	if got, ok := c.Lookup("KRENALIS_DB_HOST"); !ok || got != "db.example.internal" {
		t.Fatalf("Lookup(KRENALIS_DB_HOST) = (%q, %t), want (%q, true)", got, ok, "db.example.internal")
	}

	if got, ok := c.Lookup("DB_HOST"); ok || got != "" {
		t.Fatalf("Lookup(DB_HOST) = (%q, %t), want (\"\", false)", got, ok)
	}

	if got, ok := c.Lookup("KRENALIS_DB_PORT"); ok || got != "" {
		t.Fatalf("Lookup(KRENALIS_DB_PORT) = (%q, %t), want (\"\", false)", got, ok)
	}
}

func TestNewNormalizesPrefix(t *testing.T) {
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	store, err := New(context.Background(), "/prod/")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if store.prefix != "/prod" {
		t.Fatalf("store.prefix = %q, want %q", store.prefix, "/prod")
	}

	store, err = New(context.Background(), "/prod//")
	if err != nil {
		t.Fatalf("New() with multiple trailing slashes error = %v", err)
	}
	if store.prefix != "/prod" {
		t.Fatalf("store.prefix with multiple trailing slashes = %q, want %q", store.prefix, "/prod")
	}
}

// TestValidatePrefix verifies prefix validation rules.
func TestValidatePrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prefix string
		want   string
		}{
			{name: "valid", prefix: "/krenalis/prod"},
			{name: "valid trailing slash", prefix: "/krenalis/prod/"},
			{name: "valid multiple trailing slashes", prefix: "/krenalis/prod//"},
			{name: "empty", prefix: "", want: "prefix must not be empty"},
		{name: "missing slash", prefix: "krenalis/prod", want: "prefix must start with '/'"},
		{name: "root only", prefix: "/", want: "prefix must not be '/'"},
		{name: "empty path element", prefix: "/krenalis//prod", want: "prefix must not contain empty path elements"},
		{name: "invalid character", prefix: "/krenalis/prod?", want: `prefix contains invalid character '?'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidatePrefix(tt.prefix)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("ValidatePrefix(%q) error = %v, want nil", tt.prefix, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidatePrefix(%q) error = nil, want %q", tt.prefix, tt.want)
			}
			if err.Error() != tt.want {
				t.Fatalf("ValidatePrefix(%q) error = %q, want %q", tt.prefix, err.Error(), tt.want)
			}
		})
	}
}

type fakeSSMClient struct {
	byPathInputs  []*ssm.GetParametersByPathInput
	byPathOutputs []*ssm.GetParametersByPathOutput
	byPathErr     error

	getInputs  []*ssm.GetParametersInput
	getOutputs []*ssm.GetParametersOutput
	getErr     error
}

func (f *fakeSSMClient) GetParametersByPath(_ context.Context, input *ssm.GetParametersByPathInput, _ ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	f.byPathInputs = append(f.byPathInputs, &ssm.GetParametersByPathInput{
		Path:           input.Path,
		Recursive:      input.Recursive,
		WithDecryption: input.WithDecryption,
		NextToken:      input.NextToken,
	})
	if f.byPathErr != nil {
		return nil, f.byPathErr
	}
	if len(f.byPathOutputs) == 0 {
		return &ssm.GetParametersByPathOutput{}, nil
	}
	out := f.byPathOutputs[0]
	f.byPathOutputs = f.byPathOutputs[1:]
	return out, nil
}

func (f *fakeSSMClient) GetParameters(_ context.Context, input *ssm.GetParametersInput, _ ...func(*ssm.Options)) (*ssm.GetParametersOutput, error) {
	f.getInputs = append(f.getInputs, &ssm.GetParametersInput{
		Names:          append([]string(nil), input.Names...),
		WithDecryption: input.WithDecryption,
	})
	if f.getErr != nil {
		return nil, f.getErr
	}
	if len(f.getOutputs) == 0 {
		return &ssm.GetParametersOutput{}, nil
	}
	out := f.getOutputs[0]
	f.getOutputs = f.getOutputs[1:]
	return out, nil
}

func TestStoreLoadPagesRequestsAndLoadsDatabaseSecret(t *testing.T) {
	t.Parallel()

	client := &fakeSSMClient{
		byPathOutputs: []*ssm.GetParametersByPathOutput{
			{
				Parameters: []types.Parameter{
					{Name: stringPtr("/prod/db/schema"), Value: stringPtr("public")},
					{Name: stringPtr("/prod/db/max-connections"), Value: stringPtr("16")},
					{Name: stringPtr("/prod/http/host"), Value: stringPtr("0.0.0.0")},
					{Name: stringPtr("/prod/http/port"), Value: stringPtr("2022")},
					{Name: stringPtr("/prod/http/external-url"), Value: stringPtr("https://example.com/")},
				},
				NextToken: stringPtr("page-2"),
			},
			{
				Parameters: []types.Parameter{
					{Name: stringPtr("/prod/kms"), Value: stringPtr("aws:test-key")},
				},
			},
		},
		getOutputs: []*ssm.GetParametersOutput{
			{
				Parameters: []types.Parameter{
					{
						Name:  stringPtr("/aws/reference/secretsmanager/prod/db"),
						Value: stringPtr(`{"host":"db.internal","port":5432,"dbname":"krenalis","username":"app","password":"secret"}`),
					},
				},
			},
		},
	}

	store := &Store{
		client:      client,
		prefix:      "/prod",
		secretNames: []string{"/aws/reference/secretsmanager/prod/db"},
	}

	conf, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := len(client.byPathInputs); got != 2 {
		t.Fatalf("len(client.byPathInputs) = %d, want 2", got)
	}
	if got := len(client.getInputs); got != 1 {
		t.Fatalf("len(client.getInputs) = %d, want 1", got)
	}

	if client.byPathInputs[0].Path == nil || *client.byPathInputs[0].Path != "/prod" {
		t.Fatalf("first request path = %v, want %q", client.byPathInputs[0].Path, "/prod")
	}
	if client.byPathInputs[0].Recursive == nil || !*client.byPathInputs[0].Recursive {
		t.Fatal("first request did not enable recursive lookup")
	}
	if client.byPathInputs[0].WithDecryption == nil || !*client.byPathInputs[0].WithDecryption {
		t.Fatal("first request did not enable decryption")
	}
	if client.byPathInputs[0].NextToken != nil {
		t.Fatalf("first request next token = %v, want nil", client.byPathInputs[0].NextToken)
	}

	if client.byPathInputs[1].NextToken == nil || *client.byPathInputs[1].NextToken != "page-2" {
		t.Fatalf("second request next token = %v, want %q", client.byPathInputs[1].NextToken, "page-2")
	}

	if got := client.getInputs[0].Names; len(got) != 1 || got[0] != "/aws/reference/secretsmanager/prod/db" {
		t.Fatalf("GetParameters names = %v, want [%q]", got, "/aws/reference/secretsmanager/prod/db")
	}
	if client.getInputs[0].WithDecryption == nil || !*client.getInputs[0].WithDecryption {
		t.Fatal("GetParameters did not enable decryption")
	}

	if got := conf.Get("KRENALIS_DB_HOST"); got != "db.internal" {
		t.Fatalf("Get(KRENALIS_DB_HOST) = %q, want %q", got, "db.internal")
	}
	if got := conf.Get("KRENALIS_DB_PORT"); got != "5432" {
		t.Fatalf("Get(KRENALIS_DB_PORT) = %q, want %q", got, "5432")
	}
	if got := conf.Get("KRENALIS_DB_DATABASE"); got != "krenalis" {
		t.Fatalf("Get(KRENALIS_DB_DATABASE) = %q, want %q", got, "krenalis")
	}
	if got := conf.Get("KRENALIS_DB_USERNAME"); got != "app" {
		t.Fatalf("Get(KRENALIS_DB_USERNAME) = %q, want %q", got, "app")
	}
	if got := conf.Get("KRENALIS_DB_PASSWORD"); got != "secret" {
		t.Fatalf("Get(KRENALIS_DB_PASSWORD) = %q, want %q", got, "secret")
	}
	if got := conf.Get("KRENALIS_DB_SCHEMA"); got != "public" {
		t.Fatalf("Get(KRENALIS_DB_SCHEMA) = %q, want %q", got, "public")
	}
	if got := conf.Get("KRENALIS_HTTP_EXTERNAL_URL"); got != "https://example.com/" {
		t.Fatalf("Get(KRENALIS_HTTP_EXTERNAL_URL) = %q, want %q", got, "https://example.com/")
	}
	if got := conf.Get("KRENALIS_KMS"); got != "aws:test-key" {
		t.Fatalf("Get(KRENALIS_KMS) = %q, want %q", got, "aws:test-key")
	}
}

func TestStoreLoadReturnsErrorOnUnexpectedParameterName(t *testing.T) {
	t.Parallel()

	store := &Store{
		client: &fakeSSMClient{
			byPathOutputs: []*ssm.GetParametersByPathOutput{
				{
					Parameters: []types.Parameter{
						{Name: stringPtr("/prod/unexpected"), Value: stringPtr("value")},
					},
				},
			},
		},
		prefix:      "/prod",
		secretNames: []string{"/aws/reference/secretsmanager/prod/db"},
	}

	_, err := store.Load(context.Background())
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}

	want := `config/aws: AWS has returned an unexpected parameter name "/prod/unexpected"`
	if err.Error() != want {
		t.Fatalf("Load() error = %q, want %q", err.Error(), want)
	}
}

func TestStoreLoadReturnsErrorOnNilNameOrValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		out  *ssm.GetParametersByPathOutput
		want string
	}{
		{
			name: "nil name",
			out: &ssm.GetParametersByPathOutput{
				Parameters: []types.Parameter{
					{Value: stringPtr("value")},
				},
			},
			want: "config/aws: AWS has returned a nil parameter name",
		},
		{
			name: "nil value",
			out: &ssm.GetParametersByPathOutput{
				Parameters: []types.Parameter{
					{Name: stringPtr("/db/host")},
				},
			},
			want: "config/aws: AWS has returned a nil parameter value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &Store{
				client: &fakeSSMClient{byPathOutputs: []*ssm.GetParametersByPathOutput{tt.out}},
			}

			_, err := store.Load(context.Background())
			if err == nil {
				t.Fatal("Load() error = nil, want non-nil")
			}
			if err.Error() != tt.want {
				t.Fatalf("Load() error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestStoreLoadWrapsClientErrors(t *testing.T) {
	t.Parallel()

	store := &Store{
		client: &fakeSSMClient{byPathErr: errors.New("boom")},
	}

	_, err := store.Load(context.Background())
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}

	if got := err.Error(); got != "config/aws: boom" {
		t.Fatalf("Load() error = %q, want %q", got, "config/aws: boom")
	}
}

func TestStoreLoadReturnsErrorWhenDatabaseConfigIsMixed(t *testing.T) {
	t.Parallel()

	store := &Store{
		client: &fakeSSMClient{
			byPathOutputs: []*ssm.GetParametersByPathOutput{
				{
					Parameters: []types.Parameter{
						{Name: stringPtr("/prod/db/host"), Value: stringPtr("db.internal")},
					},
				},
			},
			getOutputs: []*ssm.GetParametersOutput{
				{
					Parameters: []types.Parameter{
						{
							Name:  stringPtr("/aws/reference/secretsmanager/prod/db"),
							Value: stringPtr(`{"host":"db.internal","port":5432,"dbname":"krenalis","username":"app","password":"secret"}`),
						},
					},
				},
			},
		},
		prefix:      "/prod",
		secretNames: []string{"/aws/reference/secretsmanager/prod/db"},
	}

	_, err := store.Load(context.Background())
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}

	want := "config/aws: both the /db Secrets Manager reference and /db/... parameters are configured; only one database configuration source is allowed"
	if err.Error() != want {
		t.Fatalf("Load() error = %q, want %q", err.Error(), want)
	}
}

func TestStoreLoadReturnsErrorOnInvalidDatabaseSecretJSON(t *testing.T) {
	t.Parallel()

	store := &Store{
		client: &fakeSSMClient{
			getOutputs: []*ssm.GetParametersOutput{
				{
					Parameters: []types.Parameter{
						{
							Name:  stringPtr("/aws/reference/secretsmanager/prod/db"),
							Value: stringPtr(`{`),
						},
					},
				},
			},
		},
		prefix:      "/prod",
		secretNames: []string{"/aws/reference/secretsmanager/prod/db"},
	}

	_, err := store.Load(context.Background())
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}

	const wantPrefix = "config/aws: invalid /db secret JSON: "
	if got := err.Error(); !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("Load() error = %q, want prefix %q", got, wantPrefix)
	}
}

func stringPtr(s string) *string {
	return &s
}
