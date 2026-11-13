// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Event discovery limits bound the transport before inspecting warehouse JSON.
const (
	MaxEventDiscoveryNameBytes   = 1024
	MaxEventDiscoverySourceBytes = 96 * 1024
	MaxEventDiscoverySources     = 500
	MaxEventDiscoveryRows        = 1000
	MaxEventDiscoveryRootBytes   = 64 * 1024
	MaxEventDiscoverySampleBytes = 8 * 1024 * 1024
)

// EventDiscoveryWindow is an absolute half-open event timestamp interval.
type EventDiscoveryWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Validate checks the interval's representation and maximum elapsed length.
func (window EventDiscoveryWindow) Validate() error {
	if window.Start.UTC().Year() < 1 || window.End.UTC().Year() > 9999 ||
		window.Start.Nanosecond()%1000 != 0 || window.End.Nanosecond()%1000 != 0 ||
		!window.Start.Before(window.End) || window.End.Sub(window.Start) > 365*24*time.Hour {
		return fmt.Errorf("warehouses: invalid event discovery window")
	}
	return nil
}

// ObservedEventSource identifies an exact recorded type and nullable name.
// RecordedName is event for track and name for page/screen, never a Count
// source.
type ObservedEventSource struct {
	Type         string  `json:"type"`
	RecordedName *string `json:"recordedName"`
}

// Validate checks a source that will be supplied as warehouse parameters.
func (source ObservedEventSource) Validate() error {
	if source.Type != "track" && source.Type != "page" && source.Type != "screen" {
		return fmt.Errorf("warehouses: invalid observed event type")
	}
	if name := source.RecordedName; name != nil &&
		(len(*name) > MaxEventDiscoveryNameBytes || !utf8.ValidString(*name) || strings.ContainsRune(*name, 0)) {
		return fmt.Errorf("warehouses: invalid observed event name")
	}
	return nil
}

// EventSourceDiscoveryRequest bounds a distinct-source read.
type EventSourceDiscoveryRequest struct {
	Window       EventDiscoveryWindow
	Type         string
	NameContains string
	Limit        int
}

// Validate checks the resolved request before any warehouse operation.
func (request EventSourceDiscoveryRequest) Validate() error {
	err := request.Window.Validate()
	if err != nil {
		return err
	}
	if request.Type != "" && request.Type != "track" && request.Type != "page" && request.Type != "screen" ||
		len(request.NameContains) > 256 || !utf8.ValidString(request.NameContains) ||
		strings.ContainsRune(request.NameContains, 0) || request.Limit < 1 || request.Limit > MaxEventDiscoverySources {
		return fmt.Errorf("warehouses: invalid event source discovery request")
	}
	return nil
}

// EventSourceDiscoveryPage contains a bounded prefix, not a global source
// count.
// OmittedLongNames counts only oversized names in the inspected prefix.
type EventSourceDiscoveryPage struct {
	Sources          []ObservedEventSource
	OmittedLongNames int
	Truncated        bool
}

// EventFieldSampleRequest selects a bounded recent sample from one exact
// source.
type EventFieldSampleRequest struct {
	Window EventDiscoveryWindow
	Source ObservedEventSource
	Limit  int
}

// Validate checks the resolved request before any warehouse operation.
func (request EventFieldSampleRequest) Validate() error {
	err := request.Window.Validate()
	if err != nil {
		return err
	}
	err = request.Source.Validate()
	if err != nil {
		return err
	}
	if request.Limit < 1 || request.Limit > MaxEventDiscoveryRows {
		return fmt.Errorf("warehouses: invalid event field sample limit")
	}
	return nil
}

// EventJSONSample distinguishes absent, readable and oversized JSON roots.
// Value contains bounded untrusted bytes; callers must validate JSON before
// use.
type EventJSONSample struct {
	Value       []byte
	Present     bool
	Uninspected bool
}

// EventFieldSampleRow contains only the two JSON roots, without identifiers.
type EventFieldSampleRow struct {
	Traits     EventJSONSample
	Properties EventJSONSample
}

// EventFieldSample is a recent stored-row prefix bounded by rows and bytes.
// HasMoreRows also reports a prefix shortened by the cumulative byte budget.
type EventFieldSample struct {
	Rows        []EventFieldSampleRow
	HasMoreRows bool
}

// EventDiscoveryWarehouse provides bounded metadata reads on recorded events.
// It is independent of the prepared profile-filter verification gate.
type EventDiscoveryWarehouse interface {
	DiscoverEventSources(context.Context, EventSourceDiscoveryRequest) (EventSourceDiscoveryPage, error)
	SampleEventFields(context.Context, EventFieldSampleRequest) (EventFieldSample, error)
}
