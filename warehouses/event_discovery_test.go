// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"strings"
	"testing"
	"time"
)

// TestEventDiscoveryValidation checks the public warehouse request limits.
func TestEventDiscoveryValidation(t *testing.T) {

	window := EventDiscoveryWindow{Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		End: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}
	for _, invalid := range []EventDiscoveryWindow{
		{}, {Start: window.End, End: window.Start},
		{Start: window.Start, End: window.End.Add(time.Nanosecond)},
		{Start: window.Start, End: window.Start.Add(366 * 24 * time.Hour)},
		{Start: time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC), End: window.End},
	} {
		err := invalid.Validate()
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid window rejection, got success for %+v", invalid)
	}
	for _, name := range []string{"", "Purchase", strings.Repeat("é", 512)} {
		for _, typ := range []string{"track", "page", "screen"} {
			source := ObservedEventSource{Type: typ, RecordedName: &name}
			err := source.Validate()
			if err != nil {
				t.Fatalf("expected exact observed source, got %v", err)
			}
		}
	}
	for _, name := range []string{strings.Repeat("é", 513), "a\x00b", "\xff"} {
		err := (ObservedEventSource{Type: "page", RecordedName: &name}).Validate()
		if err != nil {
			continue
		}
		t.Fatal("expected invalid observed name rejection, got success")
	}
	for _, limit := range []int{-1, 0, MaxEventDiscoverySources + 1} {
		err := (EventSourceDiscoveryRequest{Window: window, Limit: limit}).Validate()
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid source limit, got success for %d", limit)
	}
	for _, limit := range []int{-1, 0, MaxEventDiscoveryRows + 1} {
		err := (EventFieldSampleRequest{Window: window, Source: ObservedEventSource{Type: "page"}, Limit: limit}).Validate()
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid sample limit, got success for %d", limit)
	}
	err := (EventSourceDiscoveryRequest{Window: window, Limit: 500, NameContains: "%_\\"}).Validate()
	if err != nil {
		t.Fatalf("expected literal name search, got %v", err)
	}
	for _, query := range []EventSourceDiscoveryRequest{
		{Window: window, Limit: 1, Type: "identify"},
		{Window: window, Limit: 1, NameContains: strings.Repeat("x", 257)},
		{Window: window, Limit: 1, NameContains: "\xff"},
		{Window: window, Limit: 1, NameContains: "\x00"},
	} {
		err := query.Validate()
		if err != nil {
			continue
		}
		t.Fatal("expected invalid source query rejection, got success")
	}

}
