// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"fmt"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// TestScannerArrayBounds checks that array sizes and scan limits are validated
// before decoding elements.
func TestScannerArrayBounds(t *testing.T) {

	s := &scanner{}
	typ := types.Array(types.Boolean())
	// An array of n true values takes 5*n+1 JSON bytes and 16*n result bytes.
	const maxCount = (maxArrayScanBytes - 1) / (5 + 16)
	payload := "[" + strings.Repeat("true,", maxCount-1) + "true]"
	t.Run("largest bool array within scan limit", func(t *testing.T) {
		values, err := s.scanArray("value", typ, payload)
		if err != nil {
			t.Fatalf("expected array within scan limit, got %v", err)
		}
		if len(values) != maxCount {
			t.Fatalf("expected %d elements, got %d", maxCount, len(values))
		}
		if got := values[maxCount-1]; got != true {
			t.Fatalf("expected last element to be true, got %#v", got)
		}
	})

	for _, test := range []struct{ name, data string }{
		{"smallest bool array above scan limit", payload[:len(payload)-1] + ",true]"},
		{"oversized JSON payload", strings.Repeat(" ", maxArrayScanBytes) + "[]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := s.scanArray("value", typ, test.data)
			if err != nil {
				want := fmt.Sprintf("array for column value exceeds the %d-byte scan limit", maxArrayScanBytes)
				if err.Error() != want {
					t.Fatalf("expected %q, got %v", want, err)
				}
				return
			}
			t.Fatal("expected scan limit error, got nil")
		})
	}

}
