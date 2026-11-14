// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

// EventDiscoveryWindow is an absolute half-open event timestamp interval.
type EventDiscoveryWindow = warehouses.EventDiscoveryWindow

// ObservedEventSource identifies a stored type and exact nullable name.
type ObservedEventSource = warehouses.ObservedEventSource

// EventSourceDiscoveryQuery selects stored names, without interpreting them.
// A nil Window uses the last 30 elapsed days; a nil Limit defaults to 100.
type EventSourceDiscoveryQuery struct {
	Window       *EventDiscoveryWindow
	Type         string
	NameContains string
	Limit        *int
}

// EventSourceDiscoveryResult is a bounded prefix of distinct stored sources.
type EventSourceDiscoveryResult = datastore.EventSourceDiscoveryResult

// EventFieldDiscoveryQuery selects a recent sample from an exact stored source.
// A nil Window uses the last 30 elapsed days; a nil SampleLimit defaults to
// 200.
type EventFieldDiscoveryQuery struct {
	Window      *EventDiscoveryWindow
	Source      ObservedEventSource
	SampleLimit *int
}

// EventFieldDiscoveryResult describes sampled JSON, not a guaranteed schema.
type EventFieldDiscoveryResult = datastore.EventFieldDiscoveryResult

// EventDiscoverySample describes the independent row and path limits.
type EventDiscoverySample = datastore.EventDiscoverySample

// ObservedEventRoot distinguishes present roots from unread oversized values.
type ObservedEventRoot = datastore.ObservedEventRoot

// ObservedEventField counts observed types and presence in inspected roots.
type ObservedEventField = datastore.ObservedEventField

// ObservedEventFields describes a bounded recent sample's JSON fields.
// It returns errors.BadRequestError for invalid arguments,
// errors.UnavailableError on warehouse failure, and errors.UnprocessableError
// with code MaintenanceMode while warehouse maintenance prevents inspection.
func (this *Workspace) ObservedEventFields(ctx context.Context, query EventFieldDiscoveryQuery) (EventFieldDiscoveryResult, error) {

	this.core.mustBeOpen()

	window, err := resolveEventDiscoveryWindow(query.Window)
	if err != nil {
		return EventFieldDiscoveryResult{}, errors.BadRequest("event discovery window is not valid: %s", err)
	}
	limit := 200
	if query.SampleLimit != nil {
		limit = *query.SampleLimit
	}
	request := warehouses.EventFieldSampleRequest{Window: window, Source: query.Source, Limit: limit}
	err = request.Validate()
	if err != nil {
		return EventFieldDiscoveryResult{}, errors.BadRequest("event field query is not valid: %s", err)
	}
	result, err := this.store.ObservedEventFields(ctx, request)
	if err != nil {
		if errors.Is(err, datastore.ErrMaintenanceMode) {
			return EventFieldDiscoveryResult{}, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if _, ok := errors.AsType[*datastore.WarehouseError](err); ok {
			return EventFieldDiscoveryResult{}, errors.Unavailable("event field inspection failed")
		}
		return EventFieldDiscoveryResult{}, errors.New("core: event field inspection failed")
	}

	return result, nil
}

// ObservedEventSources lists stored names in a window, shortened visibly by
// row and byte limits. It returns errors.BadRequestError for invalid arguments,
// errors.UnavailableError on warehouse failure, and errors.UnprocessableError
// with code MaintenanceMode while warehouse maintenance prevents inspection.
func (this *Workspace) ObservedEventSources(ctx context.Context, query EventSourceDiscoveryQuery) (EventSourceDiscoveryResult, error) {

	this.core.mustBeOpen()

	window, err := resolveEventDiscoveryWindow(query.Window)
	if err != nil {
		return EventSourceDiscoveryResult{}, errors.BadRequest("event discovery window is not valid: %s", err)
	}
	limit := 100
	if query.Limit != nil {
		limit = *query.Limit
	}
	request := warehouses.EventSourceDiscoveryRequest{Window: window, Type: query.Type,
		NameContains: query.NameContains, Limit: limit}
	err = request.Validate()
	if err != nil {
		return EventSourceDiscoveryResult{}, errors.BadRequest("event source query is not valid: %s", err)
	}
	result, err := this.store.ObservedEventSources(ctx, request)
	if err != nil {
		if errors.Is(err, datastore.ErrMaintenanceMode) {
			return EventSourceDiscoveryResult{}, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if _, ok := errors.AsType[*datastore.WarehouseError](err); ok {
			return EventSourceDiscoveryResult{}, errors.Unavailable("event source discovery failed")
		}
		return EventSourceDiscoveryResult{}, errors.New("core: event source discovery failed")
	}

	return result, nil
}

// resolveEventDiscoveryWindow fixes one request's elapsed window in UTC.
func resolveEventDiscoveryWindow(window *EventDiscoveryWindow) (warehouses.EventDiscoveryWindow, error) {
	at := time.Now().UTC().Truncate(time.Microsecond)
	resolved := warehouses.EventDiscoveryWindow{Start: at.Add(-30 * 24 * time.Hour), End: at}
	if window != nil {
		resolved = warehouses.EventDiscoveryWindow{Start: window.Start.UTC(), End: window.End.UTC()}
	}
	err := resolved.Validate()
	if err != nil {
		return warehouses.EventDiscoveryWindow{}, err
	}
	if resolved.End.After(at) {
		return warehouses.EventDiscoveryWindow{}, errors.New("end cannot be in the future")
	}
	return resolved, nil
}
