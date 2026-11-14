// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	jsonv1 "encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

const (
	eventDiscoveryResultBytes = 128 * 1024
	eventDiscoveryEntryBytes  = eventDiscoveryResultBytes - 32*1024
	eventDiscoveryMaxFields   = 256
	eventDiscoveryMaxDepth    = 8
)

// EventSourceDiscoveryResult describes stored names in a resolved window.
// Truncated includes row, byte and omitted-long-name limits.
type EventSourceDiscoveryResult struct {
	Window           warehouses.EventDiscoveryWindow  `json:"window"`
	Sources          []warehouses.ObservedEventSource `json:"sources"`
	Limit            int                              `json:"limit"`
	Truncated        bool                             `json:"truncated"`
	OmittedLongNames int                              `json:"omittedLongNames"`
}

// EventFieldDiscoveryResult describes sampled JSON, not a guaranteed schema.
type EventFieldDiscoveryResult struct {
	Source warehouses.ObservedEventSource  `json:"source"`
	Window warehouses.EventDiscoveryWindow `json:"window"`
	Sample EventDiscoverySample            `json:"sample"`
	Roots  map[string]ObservedEventRoot    `json:"roots"`
	Fields []ObservedEventField            `json:"fields"`
}

// EventDiscoverySample records the sampled row count and independent limits.
// FieldsTruncated concerns paths only; root.Uninspected reports unread JSON.
type EventDiscoverySample struct {
	Rows            int  `json:"rows"`
	Limit           int  `json:"limit"`
	HasMoreRows     bool `json:"hasMoreRows"`
	FieldsTruncated bool `json:"fieldsTruncated"`
}

// ObservedEventRoot counts present roots, including oversized unread values.
// The kind counts plus Uninspected equal Present.
type ObservedEventRoot struct {
	Present     int            `json:"present"`
	Uninspected int            `json:"uninspected"`
	Kinds       map[string]int `json:"kinds"`
}

// ObservedEventField counts known presences in inspected roots only.
// Known absences are sample.Rows minus root.Uninspected minus Present.
// Property is omitted when Path cannot be represented by the filter grammar.
type ObservedEventField struct {
	Path     []string       `json:"path"`
	Property string         `json:"property,omitzero"`
	Present  int            `json:"present"`
	Kinds    map[string]int `json:"kinds"`
}

// ObservedEventFields inspects a bounded stored-row prefix in normal or
// inspection mode. Technical errors return no partial result.
func (store *Store) ObservedEventFields(ctx context.Context, request warehouses.EventFieldSampleRequest) (EventFieldDiscoveryResult, error) {

	store.mustBeOpen()
	err := request.Validate()
	if err != nil {
		return EventFieldDiscoveryResult{}, err
	}
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return EventFieldDiscoveryResult{}, err
	}
	defer done()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	backend, ok := store.warehouse().(warehouses.EventDiscoveryWarehouse)
	if !ok {
		return EventFieldDiscoveryResult{}, errors.New("core/internal/datastore: event discovery backend missing")
	}

	sample, err := backend.SampleEventFields(ctx, request)
	if err != nil {
		return EventFieldDiscoveryResult{}, err
	}
	result, err := inspectEventFields(request, sample)
	if err != nil {
		return EventFieldDiscoveryResult{}, err
	}
	err = ctx.Err()
	if err != nil {
		return EventFieldDiscoveryResult{}, wrapWarehouseError(err)
	}

	return result, nil
}

// ObservedEventSources reads distinct recorded names in normal or inspection
// mode. Limits visibly shorten the prefix; technical errors discard it.
func (store *Store) ObservedEventSources(ctx context.Context, request warehouses.EventSourceDiscoveryRequest) (EventSourceDiscoveryResult, error) {

	store.mustBeOpen()
	err := request.Validate()
	if err != nil {
		return EventSourceDiscoveryResult{}, err
	}
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return EventSourceDiscoveryResult{}, err
	}
	defer done()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	backend, ok := store.warehouse().(warehouses.EventDiscoveryWarehouse)
	if !ok {
		return EventSourceDiscoveryResult{}, errors.New("core/internal/datastore: event discovery backend missing")
	}
	page, err := backend.DiscoverEventSources(ctx, request)
	if err != nil {
		return EventSourceDiscoveryResult{}, err
	}
	if len(page.Sources) > request.Limit || page.OmittedLongNames < 0 ||
		page.OmittedLongNames > request.Limit-len(page.Sources) {
		return EventSourceDiscoveryResult{}, wrapWarehouseError(
			errors.New("core/internal/datastore: invalid event source page"))
	}
	result := EventSourceDiscoveryResult{Window: request.Window, Sources: []warehouses.ObservedEventSource{},
		Limit: request.Limit, Truncated: page.Truncated || page.OmittedLongNames > 0,
		OmittedLongNames: page.OmittedLongNames}
	bytes := 0
	for _, source := range page.Sources {
		if source.Type != "track" && source.Type != "page" && source.Type != "screen" ||
			source.RecordedName != nil && (len(*source.RecordedName) > warehouses.MaxEventDiscoveryNameBytes ||
				!utf8.ValidString(*source.RecordedName)) {
			return EventSourceDiscoveryResult{}, wrapWarehouseError(
				errors.New("core/internal/datastore: invalid observed event source"))
		}
		// Measure the escaping used by the MCP SDK's encoder.
		encoded, err := jsonv1.Marshal(source)
		if err != nil {
			return EventSourceDiscoveryResult{}, errors.New("core/internal/datastore: cannot encode observed event source")
		}
		if len(encoded)+1 > eventDiscoveryEntryBytes-bytes {
			result.Truncated = true
			break
		}
		bytes += len(encoded) + 1
		result.Sources = append(result.Sources, source)
	}

	err = ctx.Err()
	if err != nil {
		return EventSourceDiscoveryResult{}, wrapWarehouseError(err)
	}

	return result, nil
}

func eventJSONKind(value json.Value) string {
	switch value.Kind() {
	case json.True, json.False:
		return "boolean"
	case json.Null:
		return "null"
	case json.Number:
		return "number"
	case json.String:
		return "string"
	case json.Object:
		return "object"
	case json.Array:
		return "array"
	default:
		panic(fmt.Sprintf("core/internal/datastore: unexpected observed JSON kind %q", value.Kind()))
	}
}

// inspectEventFields admits paths in stable depth-first order on their first
// observation. Earlier inspected roots therefore lacked a newly admitted path;
// unread roots remain unknown through the separate Uninspected counter.
func inspectEventFields(request warehouses.EventFieldSampleRequest, sample warehouses.EventFieldSample) (EventFieldDiscoveryResult, error) {

	result := EventFieldDiscoveryResult{Source: request.Source, Window: request.Window,
		Sample: EventDiscoverySample{Rows: len(sample.Rows), Limit: request.Limit, HasMoreRows: sample.HasMoreRows},
		Roots:  map[string]ObservedEventRoot{}, Fields: []ObservedEventField{}}
	if len(sample.Rows) > request.Limit {
		return EventFieldDiscoveryResult{}, wrapWarehouseError(
			errors.New("core/internal/datastore: event sample exceeds row limit"))
	}
	indices := map[string]int{}
	fieldBytes, sampleBytes := 0, 0
	for _, name := range []string{"traits", "properties"} {
		result.Roots[name] = ObservedEventRoot{Kinds: map[string]int{}}
	}

	for _, row := range sample.Rows {
		for _, name := range []string{"traits", "properties"} {
			root := row.Traits
			if name == "properties" {
				root = row.Properties
			}
			if !root.Present && (root.Uninspected || root.Value != nil) ||
				root.Uninspected && root.Value != nil || len(root.Value) > warehouses.MaxEventDiscoveryRootBytes ||
				len(root.Value) > warehouses.MaxEventDiscoverySampleBytes-sampleBytes {
				return EventFieldDiscoveryResult{}, wrapWarehouseError(
					errors.New("core/internal/datastore: invalid event sample root"))
			}
			sampleBytes += len(root.Value)
			if !root.Present {
				continue
			}
			observed := result.Roots[name]
			observed.Present++
			if root.Uninspected {
				observed.Uninspected++
				result.Roots[name] = observed
				continue
			}
			if !json.Valid(root.Value) {
				return EventFieldDiscoveryResult{}, wrapWarehouseError(
					errors.New("core/internal/datastore: invalid observed event JSON"))
			}
			value := json.Value(root.Value)
			observed.Kinds[eventJSONKind(value)]++
			result.Roots[name] = observed
			var visit func(json.Value, []string) error
			visit = func(value json.Value, path []string) error {

				if !value.IsObject() {
					return nil
				}
				if len(path) > eventDiscoveryMaxDepth {
					for range value.Properties() {
						result.Sample.FieldsTruncated = true
						break
					}
					return nil
				}

				keys := []string{}
				values := map[string]json.Value{}
				for key, child := range value.Properties() {
					keys = append(keys, key)
					values[key] = slices.Clone(child)
				}
				slices.Sort(keys)
				for _, key := range keys {
					child := values[key]
					childPath := append(slices.Clone(path), key)
					encoded, err := json.Marshal(childPath)
					if err != nil {
						return errors.New("core/internal/datastore: cannot encode observed event path")
					}
					index, exists := indices[string(encoded)]
					if !exists {
						if len(result.Fields) == eventDiscoveryMaxFields {
							result.Sample.FieldsTruncated = true
							continue
						}
						field := ObservedEventField{Path: childPath, Kinds: map[string]int{}}
						property := strings.Join(childPath, ".")
						valid := true
						for _, segment := range childPath {
							valid = valid && types.IsValidPropertyName(segment)
						}
						if valid {
							field.Property = property
						}
						// Reserve all six kinds and the largest permitted counters.
						field.Present = warehouses.MaxEventDiscoveryRows
						for _, kind := range []string{"null", "boolean", "number", "string", "object", "array"} {
							field.Kinds[kind] = warehouses.MaxEventDiscoveryRows
						}
						bound, err := jsonv1.Marshal(field)
						if err != nil {
							return errors.New("core/internal/datastore: cannot encode observed event field")
						}
						if len(bound)+1 > eventDiscoveryEntryBytes-fieldBytes {
							result.Sample.FieldsTruncated = true
							continue
						}
						fieldBytes += len(bound) + 1
						field.Present, field.Kinds = 0, map[string]int{}
						index = len(result.Fields)
						indices[string(encoded)] = index
						result.Fields = append(result.Fields, field)
					}
					field := &result.Fields[index]
					field.Present++
					field.Kinds[eventJSONKind(child)]++
					err = visit(child, childPath)
					if err != nil {
						return err
					}
				}

				return nil
			}
			err := visit(value, []string{name})
			if err != nil {
				return EventFieldDiscoveryResult{}, err
			}
		}
	}

	return result, nil
}
