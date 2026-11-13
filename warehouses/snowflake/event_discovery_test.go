// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

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
		"ORDER BY budgeted.rn", "COLLATE("} {
		if !strings.Contains(query, required) {
			t.Fatalf("expected source SQL fragment %q, got %s", required, query)
		}
	}
	if strings.Contains(query, name) || strings.Contains(query, "COUNT(*)") || len(args) != 4 || args[3] != name {
		t.Fatalf("expected one parameterized source statement without total query, got %s, %#v", query, args)
	}
	if !strings.Contains(query, "POSITION(COLLATE(p.needle,'utf8'),COLLATE(recorded_name,'utf8'))") {
		t.Fatalf("expected literal collated POSITION search, got %s", query)
	}
	sample, values := eventDiscoverySampleQuery(warehouses.EventFieldSampleRequest{
		Window: window, Source: warehouses.ObservedEventSource{Type: "page", RecordedName: &name}, Limit: 10,
	})
	for _, required := range []string{"LIMIT 11", "rn<=10", "charged<=8388608", "traits_size<=65536",
		"properties_size<=65536", "ORDER BY sample_time DESC,sample_id DESC", "ORDER BY budgeted.rn",
		`TO_JSON("PROPERTIES")`,
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
	if strings.Count(query, "?") != 4 || strings.Count(sample, "?") != 4 ||
		args[0] != "2026-01-01 00:00:00.123456" {
		t.Fatalf("expected four parameters and exact UTC microseconds, got %#v", args)
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
		var source any
		if raw != nil {
			source = raw
		}
		err := target.Scan(source)
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
	for _, raw := range []any{"", "12345678", "123456789", 12} {
		target := eventDiscoveryText{maxBytes: 8}
		err := target.Scan(raw)
		if err != nil {
			if raw == "" || raw == "12345678" || target.value != nil {
				t.Fatalf("expected invalid or oversized raw value before copy, got %v", err)
			}
			continue
		}
		if raw != "" && raw != "12345678" {
			t.Fatalf("expected invalid raw value rejection, got %v", raw)
		}
	}

}

// TestEventDiscoveryTextBudget rejects a cursor overflow before copying it.
func TestEventDiscoveryTextBudget(t *testing.T) {
	remaining := 3
	first := eventDiscoveryText{maxBytes: 8, remaining: &remaining}
	second := eventDiscoveryText{maxBytes: 8, remaining: &remaining}
	err := first.Scan("12")
	if err != nil {
		t.Fatalf("expected first bounded text, got %v", err)
	}
	err = second.Scan("34")
	if err != nil {
		if remaining != 1 || second.value != nil || string(first.value) != "12" {
			t.Fatalf("expected rejected second copy with budget unchanged, got remaining=%d value=%v", remaining, second.value)
		}
		return
	}
	t.Fatal("expected cumulative budget rejection, got success")
}
