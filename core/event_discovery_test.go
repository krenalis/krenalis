// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
)

// TestEventDiscoveryFacadeValidation checks defaults and API argument limits.
func TestEventDiscoveryFacadeValidation(t *testing.T) {

	workspace := &Workspace{core: &Core{}}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC().Truncate(time.Microsecond)
	valid := &EventDiscoveryWindow{Start: start, End: start.Add(24 * time.Hour)}
	for _, window := range []*EventDiscoveryWindow{
		{}, {Start: start, End: start.Add(time.Nanosecond)},
		{Start: now, End: now.Add(time.Hour)},
		{Start: start, End: start.Add(366 * 24 * time.Hour)},
	} {
		_, err := workspace.ObservedEventSources(t.Context(), EventSourceDiscoveryQuery{Window: window})
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
				t.Fatalf("expected invalid window BadRequest, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid window rejection, got success")
	}
	for _, limit := range []int{-1, 0, 501} {
		_, err := workspace.ObservedEventSources(t.Context(), EventSourceDiscoveryQuery{Window: valid, Limit: &limit})
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
				t.Fatalf("expected invalid limit BadRequest, got %v", err)
			}
			continue
		}
		t.Fatalf("expected invalid source limit rejection, got success for %d", limit)
	}
	for _, query := range []EventFieldDiscoveryQuery{
		{Window: valid}, {Window: valid, Source: ObservedEventSource{Type: "identify"}},
		{Window: valid, Source: ObservedEventSource{Type: "page"}, SampleLimit: new(0)},
		{Window: valid, Source: ObservedEventSource{Type: "page"}, SampleLimit: new(1001)},
	} {
		_, err := workspace.ObservedEventFields(t.Context(), query)
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
				t.Fatalf("expected invalid sample BadRequest, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid sample rejection, got success")
	}
	for _, query := range []EventSourceDiscoveryQuery{
		{Window: valid, Type: "identify"}, {Window: valid, NameContains: strings.Repeat("x", 257)},
	} {
		_, err := workspace.ObservedEventSources(t.Context(), query)
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
				t.Fatalf("expected invalid source BadRequest, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid source rejection, got success")
	}
	window, err := resolveEventDiscoveryWindow(nil)
	if err != nil {
		t.Fatalf("expected default window, got %v", err)
	}
	if window.End.Sub(window.Start) != 30*24*time.Hour || window.End.Location() != time.UTC ||
		window.End.Nanosecond()%1000 != 0 || window.End.After(time.Now()) {
		t.Fatalf("expected last 30 elapsed days in UTC with microseconds, got %+v", window)
	}

}
