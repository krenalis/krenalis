// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"testing"
)

func TestDumpPreviewEventRequest(t *testing.T) {

	var gzBuf bytes.Buffer
	gz := gzip.NewWriter(&gzBuf)
	if _, err := gz.Write([]byte(`{"a":1}`)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	tests := []struct {
		name            string
		body            []byte
		contentType     string
		contentEncoding string
		want            string
	}{
		{
			name:        "json",
			body:        []byte(`{"a":1}`),
			contentType: "application/json",
			want: "POST https://example.test/preview\n" +
				"Content-Type: application/json\r\n" +
				"\n" +
				"{\n  \"a\": 1\n}",
		},
		{
			name:        "ndjson",
			body:        []byte("{\"a\":1}\n{\"b\":2}"),
			contentType: "application/x-ndjson",
			want: "POST https://example.test/preview\n" +
				"Content-Type: application/x-ndjson\r\n" +
				"\n" +
				"{\n    \"a\": 1\n}\n{\n    \"b\": 2\n}",
		},
		{
			name:            "gzip-json",
			body:            gzBuf.Bytes(),
			contentType:     "application/json",
			contentEncoding: "gzip",
			want: "POST https://example.test/preview\n" +
				"Content-Encoding: gzip\r\n" +
				"Content-Type: application/json\r\n" +
				"\n" +
				"{\n  \"a\": 1\n}",
		},
		{
			name:        "binary",
			body:        []byte{0xff, 0x00, 0xfe},
			contentType: "text/plain",
			want: "POST https://example.test/preview\n" +
				"Content-Type: text/plain\r\n" +
				"\n" +
				"[A binary body of 3 bytes]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "https://example.test/preview", bytes.NewReader(test.body))
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if test.contentType != "" {
				req.Header.Set("Content-Type", test.contentType)
			}
			if test.contentEncoding != "" {
				req.Header.Set("Content-Encoding", test.contentEncoding)
			}
			out, err := dumpPreviewEventRequest(req)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if string(out) != test.want {
				t.Fatalf("expected %q, got %q", test.want, string(out))
			}
		})
	}

}
