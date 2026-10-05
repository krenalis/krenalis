// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package externalassets

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/andybalholm/brotli"
)

// TestAssetURL keeps relative paths inside the configured namespace.
func TestAssetURL(t *testing.T) {

	for _, base := range []string{
		"https://example.test/assets", "https://example.test/assets/",
		"https://example.test/a/b", "https://example.test/a/b/",
	} {
		got, err := assetURL(base, "synthetic-world/photos/v1/catalog.json")
		if err != nil {
			t.Fatalf("expected joined URL, got %v", err)
		}
		want := strings.TrimSuffix(base, "/") + "/synthetic-world/photos/v1/catalog.json"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	}
	invalidPaths := []string{
		"", "/data", "../data", "x/../data", "https://other.test/data", "//other.test/data",
		"x?y", "x#y", "%2e%2e/data", "%2fdata", ".", "..", "a/./b", "a//b", "a/",
		"a\\..\\data", "a/%5c../data", "%252e%252e/data", "a/%2f../data",
	}
	for _, path := range invalidPaths {
		_, err := assetURL("https://example.test/", path)
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid relative path %q to fail, got success", path)
	}
	for _, origins := range [][]string{nil, {""}, {"malformed"}, {"https://example.test/?query"}} {
		_, err := NewResolver(origins)
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid origins %v to fail, got success", origins)
	}

}

// TestResolverActiveCancellation checks cancellation while reading a response.
func TestResolverActiveCancellation(t *testing.T) {

	entered := make(chan struct{})
	closed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		close(entered)
		<-r.Context().Done()
		close(closed)
	}))
	defer server.Close()
	resolver, err := NewResolver([]string{server.URL})
	if err != nil {
		t.Fatalf("expected resolver, got %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		finished <- resolver.Resolve(ctx, "data", 16, func(r io.Reader) error {
			_, err := io.Copy(io.Discard, r)
			return err
		})
	}()
	<-entered
	cancel()
	err = <-finished
	if err != nil {
		if !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected canceled active request, got %v", err)
		}
		<-closed
		return
	}
	t.Fatal("expected canceled active request to fail, got success")

}

// TestResolverEncodingsAndLimits checks the bound after decompression.
func TestResolverEncodingsAndLimits(t *testing.T) {

	for _, encoding := range []string{"", "identity", "gzip", "br"} {
		for _, size := range []int{7, 8, 9} {
			t.Run(fmt.Sprintf("%s/%d", encoding, size), func(t *testing.T) {

				data := bytes.Repeat([]byte{'x'}, size)
				encoded := encodeTestAsset(t, encoding, data)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Accept-Encoding") != "gzip, br" {
						t.Errorf("expected explicit encoding negotiation, got %q", r.Header.Get("Accept-Encoding"))
					}
					w.Header().Set("Content-Encoding", encoding)
					_, _ = w.Write(encoded)
				}))
				defer server.Close()
				resolver, err := NewResolver([]string{server.URL})
				if err != nil {
					t.Fatalf("expected resolver, got %v", err)
				}
				var got []byte
				err = resolver.Resolve(t.Context(), "data", 8, func(r io.Reader) error {
					var err error
					got, err = io.ReadAll(r)
					return err
				})
				if err != nil {
					if size <= 8 || !strings.Contains(err.Error(), "exceeds 8 bytes") {
						t.Fatalf("expected oversized response only, got %v", err)
					}
					return
				}
				if size > 8 || !bytes.Equal(got, data) {
					t.Fatalf("expected bounded decoded data, got %q", got)
				}

			})
		}
	}

}

// TestResolverFailures checks diagnostics when every origin fails.
func TestResolverFailures(t *testing.T) {

	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "unsupported")
	}))
	defer second.Close()
	resolver, err := NewResolver([]string{first.URL, second.URL})
	if err != nil {
		t.Fatalf("expected resolver, got %v", err)
	}
	err = resolver.Resolve(t.Context(), "catalog.json", 16, func(r io.Reader) error { return nil })
	if err == nil {
		t.Fatal("expected all-origin failure, got success")
	}
	if err != nil {
		for _, diagnostic := range []string{first.URL, second.URL, "404", "unsupported", "catalog.json"} {
			if !strings.Contains(err.Error(), diagnostic) {
				t.Fatalf("expected diagnostic %q, got %v", diagnostic, err)
			}
		}
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = resolver.Resolve(ctx, "data", 16, func(r io.Reader) error { return nil })
	if err != nil {
		if !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected request cancellation, got %v", err)
		}
		return
	}
	t.Fatal("expected canceled resolution to fail, got success")

}

// TestResolverFallback includes consumer validation in origin selection.
func TestResolverFallback(t *testing.T) {

	failures := []string{
		"success", "404", "500", "204", "network", "invalid JSON", "invalid data",
		"gzip", "br", "unsupported", "limit", "trailer",
	}
	for _, failure := range failures {
		t.Run(failure, func(t *testing.T) {

			var order []int
			var mu sync.Mutex
			oversized := encodeTestAsset(t, "gzip", []byte(`{"value":1}`+strings.Repeat(" ", 64)))
			corrupt := encodeTestAsset(t, "gzip", []byte(`{"value":1}`))
			corrupt[len(corrupt)-1] ^= 0xff
			first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				mu.Lock()
				order = append(order, 0)
				mu.Unlock()
				switch failure {
				case "success":
					_, _ = io.WriteString(w, `{"value":1}`)
				case "404":
					w.WriteHeader(404)
				case "500":
					w.WriteHeader(500)
				case "204":
					w.WriteHeader(204)
				case "invalid JSON":
					_, _ = io.WriteString(w, "{")
				case "invalid data":
					_, _ = io.WriteString(w, `{"value":0}`)
				case "gzip", "br":
					w.Header().Set("Content-Encoding", failure)
					_, _ = w.Write([]byte{0xff, 0xff, 0xff})
				case "unsupported":
					w.Header().Set("Content-Encoding", "deflate")
				case "limit":
					w.Header().Set("Content-Encoding", "gzip")
					_, _ = w.Write(oversized)
				case "trailer":
					w.Header().Set("Content-Encoding", "gzip")
					_, _ = w.Write(corrupt)
				}

			}))
			defer first.Close()
			if failure == "network" {
				first.Close()
			}
			second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				order = append(order, 1)
				mu.Unlock()
				_, _ = io.WriteString(w, `{"value":1}`)
			}))
			defer second.Close()
			third := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				order = append(order, 2)
				mu.Unlock()
				w.WriteHeader(500)
			}))
			defer third.Close()
			origins := []string{first.URL, second.URL, third.URL}
			resolver, err := NewResolver(origins)
			if err != nil {
				t.Fatalf("expected resolver, got %v", err)
			}
			origins[0] = third.URL // Constructor must retain its own ordered copy.
			err = resolver.Resolve(t.Context(), "data", 32, func(r io.Reader) error {
				var document struct{ Value int }
				err := json.NewDecoder(r).Decode(&document)
				if err != nil {
					return err
				}
				if document.Value != 1 {
					return fmt.Errorf("invalid value")
				}
				return nil
			})
			if err != nil {
				t.Fatalf("expected successful fallback, got %v", err)
			}
			want := []int{0, 1}
			if failure == "network" {
				want = []int{1}
			}
			if failure == "success" {
				want = []int{0}
			}
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(order, want) {
				t.Fatalf("expected origin order %v, got %v", want, order)
			}

		})
	}

}

func encodeTestAsset(t *testing.T, encoding string, data []byte) []byte {

	t.Helper()
	var buffer bytes.Buffer
	var writer io.WriteCloser
	switch encoding {
	case "", "identity":
		return data
	case "gzip":
		writer = gzip.NewWriter(&buffer)
	case "br":
		writer = brotli.NewWriter(&buffer)
	}
	_, err := writer.Write(data)
	if err != nil {
		t.Fatalf("expected encoded fixture, got %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("expected encoder close, got %v", err)
	}

	return buffer.Bytes()
}
