// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package http

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/json"
)

func TestAbsolutePath(t *testing.T) {
	ht := &HTTP{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{Host: "example.com", Port: 443})}}
	ht2 := &HTTP{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{Host: "example.com", Port: 8080})}}
	tests := []testconnector.AbsolutePathTest{
		{Name: "/a", Expected: "https://example.com/a"},
		{Name: "a", Expected: "https://example.com/a"},
		{Name: "/a/b", Expected: "https://example.com/a/b"},
		{Name: "/a/b?", Expected: "https://example.com/a/b"},
		{Name: "/a/b?x=y", Expected: "https://example.com/a/b?x=y"},
		{Name: "a/b?x=y", Expected: "https://example.com/a/b?x=y"},
		{Name: "/%5z"},
		{Name: "%5z"},
		{Name: "/\x00"},
		{Name: "/a/b?x=y#"},
		{Name: "/a", Expected: "https://example.com:8080/a", Storage: ht2},
	}
	err := testconnector.TestAbsolutePath(ht, tests)
	if err != nil {
		t.Errorf("HTTP Files connector: %s", err)
	}
}

// TestReader verifies that Reader performs a GET request and returns the
// response body with the Last-Modified timestamp.
func TestReader(t *testing.T) {

	const responseBody = "id,email\n1,a@example.com\n"

	lastModified := time.Date(2026, time.March, 14, 12, 30, 0, 0, time.UTC)
	requests := make(chan recordedRequest, 1)

	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests <- recordedRequest{
			method:     r.Method,
			requestURI: r.URL.RequestURI(),
		}
		w.Header().Set("Last-Modified", lastModified.Format(time.RFC1123))
		_, _ = io.WriteString(w, responseBody)
	})
	defer server.Close()

	storage, err := testconnector.NewStorage[*HTTP]("http-get", storageSettings(t, server.URL, nil))
	if err != nil {
		t.Fatalf("cannot create HTTP GET storage: %s", err)
	}

	r, ts, err := storage.Reader(context.Background(), "users.csv?batch=1")
	if err != nil {
		t.Fatalf("Reader returned error: %s", err)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("cannot read response body: %s", err)
	}
	request := <-requests
	if got, want := request.method, "GET"; got != want {
		t.Errorf("method = %s, want %s", got, want)
	}
	if got, want := request.requestURI, "/users.csv?batch=1"; got != want {
		t.Errorf("request URI = %q, want %q", got, want)
	}
	if got, want := string(data), responseBody; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if !ts.Equal(lastModified) {
		t.Errorf("timestamp = %s, want %s", ts, lastModified)
	}

}

// TestWrite verifies that Write performs a POST request with the configured
// headers, content type, and request body.
func TestWrite(t *testing.T) {

	const requestBody = "id,email\n1,a@example.com\n"

	requests := make(chan recordedRequest, 1)
	server := newTLSServer(t, func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		requests <- recordedRequest{
			body:        string(data),
			contentType: r.Header.Get("Content-Type"),
			err:         err,
			method:      r.Method,
			requestURI:  r.URL.RequestURI(),
			token:       r.Header.Get("Authorization"),
		}
		w.WriteHeader(201)
	})
	defer server.Close()

	settings := storageSettings(t, server.URL, []connectors.KV{{Key: "Authorization", Value: "Bearer token"}})
	storage, err := testconnector.NewStorage[*HTTP]("http-post", settings)
	if err != nil {
		t.Fatalf("cannot create HTTP POST storage: %s", err)
	}

	err = storage.Write(context.Background(), strings.NewReader(requestBody), "exports/users.csv", "text/csv")
	if err != nil {
		t.Fatalf("Write returned error: %s", err)
	}
	request := <-requests
	if request.err != nil {
		t.Fatalf("cannot read request body: %s", request.err)
	}
	if got, want := request.method, "POST"; got != want {
		t.Errorf("method = %s, want %s", got, want)
	}
	if got, want := request.requestURI, "/exports/users.csv"; got != want {
		t.Errorf("request URI = %q, want %q", got, want)
	}
	if got, want := request.contentType, "text/csv"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if got, want := request.token, "Bearer token"; got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
	if got, want := request.body, requestBody; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}

}

// newTLSServer returns a new HTTPS server for use in tests.
func newTLSServer(t *testing.T, h http.HandlerFunc) *httptest.Server {
	s := httptest.NewTLSServer(h)
	// The connector always builds HTTPS URLs and uses the package-level
	// transport, so tests install the TLS transport trusted by httptest.
	oldTransport := transport
	testTransport, ok := s.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("test server client transport is %T, want *http.Transport", s.Client().Transport)
	}
	transport = testTransport
	t.Cleanup(func() {
		transport = oldTransport
	})
	return s
}

// recordedRequest stores the request data observed by a test HTTP server.
type recordedRequest struct {
	body        string
	contentType string
	err         error
	method      string
	requestURI  string
	token       string
}

// storageSettings returns HTTP connector settings that target rawURL.
func storageSettings(t *testing.T, rawURL string, headers []connectors.KV) innerSettings {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("cannot parse test server URL: %s", err)
	}
	host, p, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("cannot split test server host %q: %s", u.Host, err)
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		t.Fatalf("cannot parse port %q: %s", p, err)
	}
	return innerSettings{Host: host, Port: port, Headers: headers}
}

type testSettingsStore struct {
	settings json.Value
}

func newTestSettingsStore(t *testing.T, settings any) *testSettingsStore {
	t.Helper()

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("cannot marshal test settings: %s", err)
	}
	return &testSettingsStore{settings: data}
}

func (s *testSettingsStore) Load(ctx context.Context, dst any) error {
	return json.Unmarshal(s.settings, dst)
}

func (s *testSettingsStore) Store(ctx context.Context, src any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	s.settings = data
	return nil
}
