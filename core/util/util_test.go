//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package util

import "testing"

func Test_UUIDFromBytes(t *testing.T) {
	tests := []struct {
		bytes       []byte
		expectedStr string
		expectedOk  bool
	}{
		{
			bytes:       []byte{},
			expectedStr: "",
			expectedOk:  false,
		},
		{
			bytes:       nil,
			expectedStr: "",
			expectedOk:  false,
		},
		{
			bytes:       []byte{100},
			expectedStr: "",
			expectedOk:  false,
		},
		{
			bytes:       []byte{211, 124, 89, 213, 136, 127, 68, 248, 143, 250, 126, 36, 49, 79, 71, 62},
			expectedStr: "d37c59d5-887f-44f8-8ffa-7e24314f473e",
			expectedOk:  true,
		},
		{
			bytes:       []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expectedStr: "00000000-0000-0000-0000-000000000000",
			expectedOk:  true,
		},
		{
			bytes:       []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // 15 bytes instead of 16
			expectedStr: "",
			expectedOk:  false,
		},
		{
			bytes:       []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, // 17 bytes instead of 16
			expectedStr: "",
			expectedOk:  false,
		},
		{
			bytes:       []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			expectedStr: "ffffffff-ffff-ffff-ffff-ffffffffffff",
			expectedOk:  true,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			gotStr, gotOk := UUIDFromBytes(test.bytes)
			if gotStr != test.expectedStr {
				t.Fatalf("expected %q, got %q", test.expectedStr, gotStr)
			}
			if gotOk != test.expectedOk {
				t.Fatalf("expected ok = %t, got ok = %t", test.expectedOk, gotOk)
			}
		})
	}
}
