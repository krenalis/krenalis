//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package http

import (
	"strings"
	"testing"

	"github.com/meergo/meergo/core/testconnector"
)

func TestAbsolutePath(t *testing.T) {
	http := &HTTP{settings: &innerSettings{Host: "example.com", Port: 443}}
	http2 := &HTTP{settings: &innerSettings{Host: "example.com", Port: 8080}}
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
		{Name: "/a", Expected: "https://example.com:8080/a", Storage: http2},
	}
	err := testconnector.TestAbsolutePath(http, tests)
	if err != nil {
		t.Errorf("HTTP Files connector: %s", err)
	}
}

func TestValidateHost(t *testing.T) {

	// helper: generate a hostname longer than 253 bytes
	makeTooLong := func() string {
		// 17 labels of 15 chars + 16 dots = 17*15 + 16 = 271 bytes
		label := strings.Repeat("a", 15)
		parts := make([]string, 17)
		for i := range parts {
			parts[i] = label
		}
		return strings.Join(parts, ".")
	}

	tests := []struct {
		name       string
		host       string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "valid IPv4",
			host:    "192.168.0.1",
			wantErr: false,
		},
		{
			name:    "valid IPv6",
			host:    "::1",
			wantErr: false,
		},
		{
			name:    "valid ASCII hostname",
			host:    "example.com",
			wantErr: false,
		},
		{
			name:    "valid IDN requiring punycode",
			host:    "bücher.de", // become xn--bcher-kva.de
			wantErr: false,
		},
		{
			name:       "hostname with port (forbidden)",
			host:       "example.com:80",
			wantErr:    true,
			wantErrMsg: "host cannot include a port",
		},
		{
			name:       "IPv6 with port (forbidden)",
			host:       "[2001:db8::1]:443",
			wantErr:    true,
			wantErrMsg: "host cannot include a port",
		},
		{
			name:       "empty string",
			host:       "",
			wantErr:    true,
			wantErrMsg: "host length in bytes must be in range [1,253]",
		},
		{
			name:       "space inside hostname",
			host:       "exa mple.com",
			wantErr:    true,
			wantErrMsg: "host is not valid",
		},
		{
			name:       "URL scheme not allowed",
			host:       "http://example.com",
			wantErr:    true,
			wantErrMsg: "host is not valid",
		},
		{
			name:       "too long hostname",
			host:       makeTooLong(),
			wantErr:    true,
			wantErrMsg: "host length in bytes must be in range [1,253]",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateHost(tc.host)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for host %q, got nil", tc.host)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error for host %q, got %v", tc.host, err)
			}
			if tc.wantErr && tc.wantErrMsg != "" && err != nil {
				if err.Error() != tc.wantErrMsg {
					t.Fatalf("expected %q, got %q", tc.wantErrMsg, err)
				}
			}
		})
	}

}
