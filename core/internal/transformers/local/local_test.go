// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package local

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func Test_versionFromFilename(t *testing.T) {
	const (
		js = "js"
		py = "py"
	)
	tests := []struct {
		ext      string
		name     string
		filename string
		version  int
		ok       bool
	}{
		{py, "", "", 0, false},
		{py, "12345", "", 0, false},
		{py, "", ".v10.py", 10, true},
		{js, "", ".v10.js", 10, true},
		{js, "", ".v10.py", 0, false},
		{py, "", ".v10.js", 0, false},
		{py, "12345", ".py", 0, false},
		{py, "789", "12345.v10.py", 0, false},
		{py, "12345", "12345.v10.py", 10, true},
		{py, "12345", "12345_v10.py", 0, false},
		{py, "12345", "12345_z10.py", 0, false},
		{py, "pipeline", "pipeline-12345.v1.py", 0, false},
		{py, "pipeline", "pipeline-12345.vA.py", 0, false},
		{py, "pipeline", "pipeline-12345.v1.txt", 0, false},
		{py, "pipeline", "pipeline-12345.v10.py", 0, false},
		{py, "pipeline", "pipeline-12345.v1042.py", 0, false},
		{py, "pipeline-12345", "pipeline-12345.v1.py", 1, true},
		{py, "pipeline-12345", "pipeline-12345.vA.py", 0, false},
		{py, "pipeline-12345", "pipeline-12345.v1.txt", 0, false},
		{py, "pipeline-12345", "pipeline-12345.v10.py", 10, true},
		{py, "pipeline-12345", "pipeline-12345.v1042.js", 0, false},
		{py, "pipeline-12345", "pipeline-12345.v1042.py", 1042, true},
		{js, "pipeline-12345", "pipeline-12345.v1042.js", 1042, true},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			gotV, gotOk := versionFromFilename(test.filename, test.name, test.ext)
			if test.ok != gotOk {
				t.Fatalf("filenameToVersion(%q, %q, %q): expected ok = %t, got %t", test.name, test.filename, test.ext, test.ok, gotOk)
			}
			if test.version != gotV {
				t.Fatalf("filenameToVersion(%q, %q, %q): expected version = %d, got %d", test.name, test.filename, test.ext, test.version, gotV)
			}
		})
	}
}

// TestDiscardBoundary tests the discardBoundary function.
func TestDiscardBoundary(t *testing.T) {
	t.Parallel()

	const boundary = "123e4567-e89b-12d3-a456-426614174000"
	const defaultJSON = `{"records":[{"value":42}]}`
	marker := "----" + boundary

	buildPayload := func(noise, json string) []byte {
		var buf bytes.Buffer
		buf.Grow(len(boundary) + len(noise) + len(marker) + len(json))
		buf.WriteString(boundary)
		buf.WriteString(noise)
		buf.WriteString(marker)
		buf.WriteString(json)
		return buf.Bytes()
	}

	isEOFError := func(err error) bool {
		return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
	}

	largeNoise := strings.Repeat("log line with utf8 marker\n", 256)
	noiseWithHyphens := "leading----" + boundary[:18] + "--tail\n"
	noiseWithPartialMarker := "partial " + marker[:len(marker)-1]

	tests := []struct {
		name     string
		input    []byte
		wantJSON string
		errDesc  string
		errCheck func(error) bool
	}{
		{
			name:     "no stdout noise",
			input:    buildPayload("", defaultJSON),
			wantJSON: defaultJSON,
		},
		{
			name:     "with regular stdout noise",
			input:    buildPayload("console.log('hello');\n", defaultJSON),
			wantJSON: defaultJSON,
		},
		{
			name:     "noise with hyphen fragments",
			input:    buildPayload(noiseWithHyphens, defaultJSON),
			wantJSON: defaultJSON,
		},
		{
			name:     "noise containing partial marker",
			input:    buildPayload(noiseWithPartialMarker+"\n", defaultJSON),
			wantJSON: defaultJSON,
		},
		{
			name:     "large stdout noise",
			input:    buildPayload(largeNoise, defaultJSON),
			wantJSON: defaultJSON,
		},
		{
			name:     "missing closing marker",
			input:    append([]byte(boundary), []byte("spurious output")...),
			errDesc:  "boundary marker not found",
			errCheck: isEOFError,
		},
		{
			name: "closing marker truncated boundary",
			input: func() []byte {
				payload := buildPayload("stdout noise\n", "")
				return payload[:len(payload)-10]
			}(),
			errDesc:  "boundary marker truncated",
			errCheck: isEOFError,
		},
		{
			name:     "boundary prefix truncated",
			input:    []byte(boundary[:20]),
			errDesc:  "initial boundary truncated",
			errCheck: func(err error) bool { return errors.Is(err, io.ErrUnexpectedEOF) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reader := bytes.NewReader(tc.input)
			err := discardBoundary(reader)
			if tc.errCheck == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				remaining, readErr := io.ReadAll(reader)
				if readErr != nil {
					t.Fatalf("expected to read remaining payload, got %v", readErr)
				}
				if string(remaining) != tc.wantJSON {
					t.Fatalf("expected remaining payload %q, got %q", tc.wantJSON, string(remaining))
				}
				return
			}

			if err == nil {
				t.Fatalf("expected %s, got nil", tc.errDesc)
			}
			if !tc.errCheck(err) {
				t.Fatalf("expected %s, got %v", tc.errDesc, err)
			}
		})
	}
}
