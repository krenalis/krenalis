// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestStatementValues checks argument bounds and exact UTC transport.
func TestStatementValues(t *testing.T) {
	instant := time.Date(2026, 1, 1, 12, 30, 0, 123456000, time.FixedZone("offset", 7200))
	q := Statement{SQL: "SELECT 1", Columns: []warehouses.Column{{Name: "value", Type: types.Int(64)}},
		Arguments: []Argument{{Type: types.DateTime(), Value: instant}, {Type: types.String(), Value: "Purchase'"}}}
	values, err := q.Values()
	if err != nil {
		t.Fatalf("expected typed arguments, got %v", err)
	}
	if values[0] != "2026-01-01 10:30:00.123456" || values[1] != "Purchase'" {
		t.Fatalf("expected exact UTC microseconds and literal text, got %v", values)
	}
	for _, arg := range []Argument{{Type: types.DateTime(), Value: instant.Add(time.Nanosecond)},
		{Type: types.Int(64), Value: 1}, {Type: types.DateTime(), Value: "2026-01-01"}, {Type: types.String(), Value: 1}} {
		q.Arguments = []Argument{arg}
		_, err = q.Values()
		if err == nil {
			t.Fatal("expected invalid argument rejection, got success")
		}
	}
}
