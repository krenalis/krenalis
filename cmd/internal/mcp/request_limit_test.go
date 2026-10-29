// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/krenalis/krenalis/tools/errors"

	"github.com/mark3labs/mcp-go/mcp"
)

const initializeRequest = `{"jsonrpc":"2.0","id":"size","method":"initialize","params":` +
	`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"size-test","version":"1"}}}`

type trackedRequestBody struct {
	reader io.Reader
	read   int
	calls  int
	closed bool
}

// Close records closure of the request body.
func (b *trackedRequestBody) Close() error {
	b.closed = true
	return nil
}

// Read counts bytes consumed from the underlying reader.
func (b *trackedRequestBody) Read(p []byte) (int, error) {
	b.calls++
	n, err := b.reader.Read(p)
	b.read += n
	return n, err
}

func TestMCPRequestBodyLimit(t *testing.T) {
	s := NewMCPServer(nil)
	defer s.Close(t.Context())
	server := httptest.NewServer(s)
	defer server.Close()
	for _, chunked := range []bool{false, true} {
		for _, test := range []struct {
			name string
			size int
			code int
		}{
			{"small", len(initializeRequest), http.StatusOK},
			{"below", maxRequestBytes - 1, http.StatusOK},
			{"at", maxRequestBytes, http.StatusOK},
			{"above", maxRequestBytes + 1, http.StatusRequestEntityTooLarge},
		} {
			name := "known/" + test.name
			if chunked {
				name = "chunked/" + test.name
			}
			t.Run(name, func(t *testing.T) {

				// A valid prefix followed by spaces detects silent truncation.
				body := initializeRequest + strings.Repeat(" ", test.size-len(initializeRequest))
				req, err := http.NewRequestWithContext(t.Context(), "POST", server.URL, strings.NewReader(body))
				if err != nil {
					t.Fatalf("expected request, got %v", err)
				}
				req.Header.Set("Authorization", "Bearer mcp_size_test")
				req.Header.Set("Content-Type", "application/json")
				if chunked {
					req.ContentLength = -1
				}

				resp, err := server.Client().Do(req)
				if err != nil {
					t.Fatalf("expected response, got %v", err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != test.code {
					t.Fatalf("expected HTTP %d, got %d", test.code, resp.StatusCode)
				}
				if resp.StatusCode == http.StatusOK {
					var message struct {
						Result *struct {
							ProtocolVersion string `json:"protocolVersion"`
						} `json:"result"`
						Error any `json:"error"`
					}
					err = json.NewDecoder(resp.Body).Decode(&message)
					if err != nil {
						t.Fatalf("expected initialize result, got %v", err)
					}
					if message.Result == nil || message.Result.ProtocolVersion == "" || message.Error != nil {
						t.Fatalf("expected successful SDK initialization, got %+v", message)
					}
				}

			})
		}
	}
}

func TestMCPRequestBodyLimitReads(t *testing.T) {
	s := NewMCPServer(nil)
	defer s.Close(t.Context())
	for _, test := range []struct {
		name   string
		length int64
		read   int
	}{
		{"declared oversize", maxRequestBytes + 1, 0},
		{"unknown length", -1, maxRequestBytes + 1},
		{"underdeclared length", 1, maxRequestBytes + 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := &trackedRequestBody{reader: strings.NewReader(strings.Repeat("x", maxRequestBytes+64))}
			req := httptest.NewRequest("POST", "/mcp", body).WithContext(t.Context())
			req.ContentLength = test.length
			req.Header.Set("Authorization", "Bearer mcp_size_test")
			response := httptest.NewRecorder()
			s.ServeHTTP(response, req)
			if response.Code != http.StatusRequestEntityTooLarge || body.read != test.read {
				t.Fatalf("expected HTTP 413 and %d bytes read, got %d and %d", test.read, response.Code, body.read)
			}
			if test.read == 0 && body.calls != 0 {
				t.Fatalf("expected no reads for declared oversize, got %d", body.calls)
			}
			if test.read > 0 && !body.closed {
				t.Fatal("expected consumed request body to be closed, got open")
			}
		})
	}
}

func TestMCPRequestBodyReadError(t *testing.T) {
	s := NewMCPServer(nil)
	defer s.Close(t.Context())
	body := &trackedRequestBody{reader: iotest.ErrReader(errors.New("private reader error"))}
	req := httptest.NewRequest("POST", "/mcp", body).WithContext(t.Context())
	req.Header.Set("Authorization", "Bearer mcp_size_test")
	response := httptest.NewRecorder()
	s.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest || response.Body.String() != "cannot read MCP request body\n" ||
		!body.closed {
		t.Fatalf("expected opaque HTTP 400 and closed body, got %d, %q, %v",
			response.Code, response.Body.String(), body.closed)
	}
}

func TestMCPRequestBodyOtherHandling(t *testing.T) {
	s := NewMCPServer(nil)
	defer s.Close(t.Context())
	for _, method := range []string{"GET", "DELETE", "HEAD"} {
		t.Run(method, func(t *testing.T) {
			body := &trackedRequestBody{reader: iotest.ErrReader(errors.New("body must not be read"))}
			// End a possible GET event stream without timers or goroutines.
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			req := httptest.NewRequest(method, "/mcp", body).WithContext(ctx)
			req.ContentLength = maxRequestBytes + 1
			req.Header.Set("Authorization", "Bearer mcp_size_test")
			response := httptest.NewRecorder()
			s.ServeHTTP(response, req)
			if body.calls != 0 || response.Code == http.StatusRequestEntityTooLarge {
				t.Fatalf("expected unchanged non-POST handling without reads, got %d calls and HTTP %d",
					body.calls, response.Code)
			}
		})
	}
	t.Run("malformed JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/mcp", strings.NewReader("{")).WithContext(t.Context())
		req.Header.Set("Authorization", "Bearer mcp_size_test")
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		s.ServeHTTP(response, req)
		var message struct {
			Error struct {
				Code int `json:"code"`
			} `json:"error"`
		}
		err := json.Unmarshal(response.Body.Bytes(), &message)
		if err != nil {
			t.Fatalf("expected SDK JSON-RPC error, got %v", err)
		}
		if message.Error.Code != mcp.PARSE_ERROR {
			t.Fatalf("expected SDK parse error %d, got %d", mcp.PARSE_ERROR, message.Error.Code)
		}
	})
}
