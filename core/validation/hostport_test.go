//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package validation

import (
	"strings"
	"testing"
)

// TestValidateHost verifies that ValidateHost correctly accepts valid hosts
// (IPv4, IPv6, ASCII hostnames, and IDNs) and rejects invalid ones,
// including inputs with ports or invalid characters.
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
			name:       "valid IP with zone index",
			host:       "fe80::1ff:fe23:4567:890a%eth0",
			wantErr:    true,
			wantErrMsg: "host cannot contain a zone",
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
			name:       "hostname with invalid port",
			host:       "example.com:abc",
			wantErr:    true,
			wantErrMsg: "host is not valid",
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
			err := ValidateHost(tc.host)
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

// TestValidatePort verifies that ValidatePort correctly accepts and rejects
// ports.
func TestValidatePort(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		wantError bool
	}{
		{"lower bound valid", 1, false},
		{"upper bound valid", 65535, false},
		{"zero invalid", 0, true},
		{"negative invalid", -10, true},
		{"above range invalid", 65536, true},
		{"middle valid", 8080, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port)
			if tt.wantError && err == nil {
				t.Errorf("expected error, got nil for port %d", tt.port)
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected nil, got error %v for port %d", err, tt.port)
			}
		})
	}
}

// TestValidatePortString verifies that ValidatePortString correctly parses and
// validates string inputs as TCP ports, returning the correct integer or an
// error.
func TestValidatePortString(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPort  int
		wantError bool
	}{
		{"valid lower bound", "1", 1, false},
		{"valid upper bound", "65535", 65535, false},
		{"valid middle", "8080", 8080, false},
		{"empty string", "", 0, true},
		{"non-numeric", "abc", 0, true},
		{"negative-like string", "-1", 0, true},
		{"zero invalid", "0", 0, true},
		{"too large", "70000", 0, true},
		{"leading zeros valid", "00080", 80, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePortString(tt.input)
			if tt.wantError && err == nil {
				t.Errorf("expected error, got nil for input %q", tt.input)
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected nil, got error %v for input %q", err, tt.input)
			}
			if got != tt.wantPort {
				t.Errorf("expected %d, got %d for input %q", tt.wantPort, got, tt.input)
			}
		})
	}
}
