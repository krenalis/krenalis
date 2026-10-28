// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestProfileScanBudget bounds raw values before normalization across the
// entire cursor, including JSON, arrays and maps read from the warehouse.
func TestProfileScanBudget(t *testing.T) {

	for _, typ := range []types.Type{
		types.String(), types.JSON(), types.Array(types.String()), types.Map(types.String()),
	} {
		s := &scanner{columns: []warehouses.Column{{Name: "value", Type: typ}}, maxBytes: 16, dest: make([]any, 1)}
		err := (scanValue{s}).Scan(strings.Repeat("x", 17))
		if err == nil || s.dest[0] != nil {
			t.Fatalf("expected bounded %s before normalization, got %v, %v", typ, s.dest[0], err)
		}
	}

	for _, typ := range []types.Type{types.UUID(), types.Decimal(38, 10)} {
		s := &scanner{columns: []warehouses.Column{{Name: "value", Type: typ}}, maxBytes: 1024, dest: make([]any, 1)}
		err := (scanValue{s}).Scan("warehouse-controlled-value")
		if err != nil {
			if strings.Contains(err.Error(), "warehouse-controlled-value") {
				t.Fatalf("expected sanitized normalization error, got %v", err)
			}
		}
		if err == nil {
			t.Fatal("expected invalid scalar error, got success")
		}
	}

	for _, limit := range []int{0, 8} {
		s := &scanner{columns: []warehouses.Column{{Name: "value", Type: types.String()}},
			maxBytes: limit, dest: make([]any, 1)}
		for i := range 3 {
			err := (scanValue{s}).Scan("four")
			if err != nil {
				if limit == 0 || i != 2 {
					t.Fatalf("expected only the bounded third value to fail, got %v", err)
				}
			}
			if limit > 0 && i == 2 && err == nil {
				t.Fatal("expected cumulative cursor budget failure, got success")
			}
		}
	}

}
