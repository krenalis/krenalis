// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/krenalis/krenalis/warehouses"
)

// TestEventDiscoveryQueries checks SQL limits and literal bound selectors.
func TestEventDiscoveryQueries(t *testing.T) {

	window := warehouses.EventDiscoveryWindow{Start: time.Date(2026, 1, 1, 0, 0, 0, 123456000, time.UTC),
		End: time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)}
	name := "x%_\\' ? DROP TABLE events"
	query, args := eventDiscoverySourcesQuery(warehouses.EventSourceDiscoveryRequest{
		Window: window, Type: "page", NameContains: name, Limit: 500,
	})
	for _, required := range []string{"SELECT DISTINCT", "LIMIT 501", "NULLS FIRST",
		"ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW", "charged<=98304", "name_size<=1024",
		"ORDER BY budgeted.rn", `COLLATE "C"`} {
		if !strings.Contains(query, required) {
			t.Fatalf("expected source SQL fragment %q, got %s", required, query)
		}
	}
	if strings.Contains(query, name) || strings.Contains(query, "COUNT(*)") || len(args) != 4 || args[3] != name {
		t.Fatalf("expected one parameterized source statement without total query, got %s, %#v", query, args)
	}
	if !strings.Contains(query, "strpos(recorded_name, p.needle)") {
		t.Fatalf("expected literal strpos search, got %s", query)
	}
	sample, values := eventDiscoverySampleQuery(warehouses.EventFieldSampleRequest{
		Window: window, Source: warehouses.ObservedEventSource{Type: "page", RecordedName: &name}, Limit: 10,
	})
	for _, required := range []string{"LIMIT 11", "rn<=10", "charged<=8388608", "traits_size<=65536",
		"properties_size<=65536", "ORDER BY sample_time DESC,sample_id DESC", "ORDER BY budgeted.rn",
		`CAST("properties" AS text)`,
	} {
		if !strings.Contains(sample, required) {
			t.Fatalf("expected sample SQL fragment %q, got %s", required, sample)
		}
	}
	if len(values) != 4 || values[3] != name || strings.Contains(sample, name) {
		t.Fatalf("expected exact parameterized name, got %#v and %s", values, sample)
	}
	_, values = eventDiscoverySampleQuery(warehouses.EventFieldSampleRequest{
		Window: window, Source: warehouses.ObservedEventSource{Type: "page"}, Limit: 10,
	})
	if values[3] != nil {
		t.Fatalf("expected explicit SQL NULL selector, got %#v", values[3])
	}
	if args[0] != window.Start || args[1] != window.End {
		t.Fatalf("expected exact microsecond timestamps, got %#v", args)
	}

}

// TestEventDiscoveryRoot checks the payload and metadata framing.
func TestEventDiscoveryRoot(t *testing.T) {
	for _, test := range []struct {
		size, value                   []byte
		present, uninspected, invalid bool
	}{
		{nil, nil, false, false, false},
		{[]byte("4"), []byte("null"), true, false, false},
		{[]byte("65537"), nil, true, true, false},
		{nil, []byte("null"), false, false, true},
		{[]byte("2"), []byte("null"), false, false, true},
		{[]byte("4"), nil, false, false, true},
		{[]byte("65537"), []byte("x"), false, false, true},
		{[]byte("-1"), nil, false, false, true},
		{[]byte("18446744073709551616"), nil, false, false, true},
		{[]byte("1"), []byte{0xff}, false, false, true},
	} {
		root, err := eventDiscoveryRoot(test.size, test.value)
		if err != nil {
			if !test.invalid {
				t.Fatalf("expected bounded sample root, got %v", err)
			}
			continue
		}
		if test.invalid || root.Present != test.present || root.Uninspected != test.uninspected {
			t.Fatalf("expected present=%v uninspected=%v invalid=%v, got %+v",
				test.present, test.uninspected, test.invalid, root)
		}
	}
}

// TestEventDiscoveryText rejects unsafe input before retaining warehouse text.
func TestEventDiscoveryText(t *testing.T) {

	for _, raw := range [][]byte{nil, {}, []byte("12345678"), []byte("123456789")} {
		target := eventDiscoveryText{maxBytes: 8}
		plan := (pgtype.TextCodec{}).PlanScan(pgtype.NewMap(), pgtype.TextOID, pgtype.TextFormatCode, &target)
		if plan == nil {
			t.Fatal("expected raw text scan plan, got nil")
		}
		err := plan.Scan(raw, &target)
		if err != nil {
			if len(raw) <= 8 || target.value != nil {
				t.Fatalf("expected rejection before copy, got %v and %v", err, target.value)
			}
			continue
		}
		if len(raw) > 8 || !reflect.DeepEqual(target.value, raw) {
			t.Fatalf("expected exact bounded text, got %v", target.value)
		}
		if len(raw) > 0 {
			original := bytes.Clone(target.value)
			raw[0] = 'x'
			if !bytes.Equal(target.value, original) {
				t.Fatalf("expected copy independent of the next driver row, got %v", target.value)
			}
		}
	}
	target := eventDiscoveryText{maxBytes: 8}
	plan := (pgtype.TextCodec{}).PlanScan(pgtype.NewMap(), pgtype.TextOID, pgtype.BinaryFormatCode, &target)
	err := plan.Scan([]byte("binary"), &target)
	if err != nil {
		t.Fatalf("expected binary text through BytesScanner, got %v", err)
	}

}

// TestEventDiscoveryTextBudget rejects a cursor overflow before copying it.
func TestEventDiscoveryTextBudget(t *testing.T) {
	remaining := 3
	first := eventDiscoveryText{maxBytes: 8, remaining: &remaining}
	second := eventDiscoveryText{maxBytes: 8, remaining: &remaining}
	err := first.ScanBytes([]byte("12"))
	if err != nil {
		t.Fatalf("expected first bounded text, got %v", err)
	}
	err = second.ScanBytes([]byte("34"))
	if err != nil {
		if remaining != 1 || second.value != nil || string(first.value) != "12" {
			t.Fatalf("expected rejected second copy with budget unchanged, got remaining=%d value=%v", remaining, second.value)
		}
		return
	}
	t.Fatal("expected cumulative budget rejection, got success")
}
