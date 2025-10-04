//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package meergo

import (
	"fmt"
	"testing"
)

func TestValidateConnectorCode(t *testing.T) {
	valid := []string{"a", "abc", "abc-123", "0", "-", "-a", "a-", "a-b-c", "z9-", "12345", "alpha-0-omega", "postgresql", "http-get"}
	invalid := map[string]string{
		"":     "code is missing for a connector of type App",
		"ABC":  "connector code ABC is not valid; valid codes contain only [a-z0-9-]",
		"a_b":  "connector code a_b is not valid; valid codes contain only [a-z0-9-]",
		"a b":  "connector code a b is not valid; valid codes contain only [a-z0-9-]",
		"a.b":  "connector code a.b is not valid; valid codes contain only [a-z0-9-]",
		"a/b":  "connector code a/b is not valid; valid codes contain only [a-z0-9-]",
		"café": "connector code café is not valid; valid codes contain only [a-z0-9-]",
		"ç":    "connector code ç is not valid; valid codes contain only [a-z0-9-]",
		"🙂":    "connector code 🙂 is not valid; valid codes contain only [a-z0-9-]",
	}

	// Valid.
	for _, code := range valid {
		code := code
		t.Run(fmt.Sprintf("valid_%q", code), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("expected no panic for %q, got %v", code, r)
				}
			}()
			validateConnectorCode("App", code)
		})
	}

	// Invalid.
	for code, expected := range invalid {
		code, expected := code, expected
		t.Run(fmt.Sprintf("invalid_%q", code), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic %q, got none", expected)
				} else if r != expected {
					t.Fatalf("expected %q, got %q", expected, r)
				}
			}()
			validateConnectorCode("App", code)
		})
	}
}
