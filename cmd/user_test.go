//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"testing"

	"github.com/google/uuid"
)

func Test_parseUUID(t *testing.T) {
	tests := []struct {
		s    string
		uuid uuid.UUID
		ok   bool
	}{

		// Supported UUID formats.
		{"2a9b8326-aadb-416a-adc1-71761f3ff4b9", uuid.MustParse("2a9b8326-aadb-416a-adc1-71761f3ff4b9"), true},
		{"2a9b8326-aadb-416a-ADC1-71761F3FF4B9", uuid.MustParse("2a9b8326-aadb-416a-adc1-71761f3ff4b9"), true},
		{"2A9B8326-AADB-416A-ADC1-71761F3FF4B9", uuid.MustParse("2a9b8326-aadb-416a-adc1-71761f3ff4b9"), true},

		// Unsupported UUID formats.
		{"60af802184814c8389153f9055d57e6c", uuid.UUID{}, false},
		{"60AF802184814C8389153F9055D57E6C", uuid.UUID{}, false},
		{"{60af8021-8481-4c83-8915-3f9055d57e6c}", uuid.UUID{}, false},
		{"{60AF8021-8481-4C83-8915-3F9055D57E6C}", uuid.UUID{}, false},
		{"urn:uuid:60af8021-8481-4c83-8915-3f9055d57e6c", uuid.UUID{}, false},
		{"urn:uuid:60AF8021-8481-4C83-8915-3F9055D57E6C", uuid.UUID{}, false},

		// Strings that do not represent UUIDs.
		{"", uuid.UUID{}, false},
		{"12345", uuid.UUID{}, false},
		{"abcdef0123456789", uuid.UUID{}, false},
		{"2a9b8326-aadb-416a-adc1-71761f3ff4b92a9b8326-aadb-416a-adc1-71761f3ff4b9", uuid.UUID{}, false},
	}
	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			gotUUID, gotOk := parseUUID(test.s)
			if gotUUID != test.uuid {
				t.Fatalf("expected UUID %s, got %s", test.uuid, gotUUID)
			}
			if gotOk != test.ok {
				t.Fatalf("expected ok = %t, got %t", test.ok, gotOk)
			}
		})
	}
}
