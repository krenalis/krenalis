// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	jsonv1 "encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

type eventDiscoveryTestWarehouse struct {
	warehouses.Warehouse
	page    warehouses.EventSourceDiscoveryPage
	sample  warehouses.EventFieldSample
	err     error
	wait    bool
	cancel  context.CancelFunc
	queries int
}

// DiscoverEventSources supplies a test page or waits for cancellation.
func (warehouse *eventDiscoveryTestWarehouse) DiscoverEventSources(ctx context.Context, request warehouses.EventSourceDiscoveryRequest) (warehouses.EventSourceDiscoveryPage, error) {
	warehouse.queries++
	if warehouse.cancel != nil {
		warehouse.cancel()
	}
	if warehouse.wait {
		<-ctx.Done()
		return warehouses.EventSourceDiscoveryPage{}, ctx.Err()
	}
	return warehouse.page, warehouse.err
}

// SampleEventFields supplies a bounded test response or waits for cancellation.
func (warehouse *eventDiscoveryTestWarehouse) SampleEventFields(ctx context.Context, request warehouses.EventFieldSampleRequest) (warehouses.EventFieldSample, error) {
	warehouse.queries++
	if warehouse.cancel != nil {
		warehouse.cancel()
	}
	if warehouse.wait {
		<-ctx.Done()
		return warehouses.EventFieldSample{}, ctx.Err()
	}
	return warehouse.sample, warehouse.err
}

// TestEventDiscoveryFailure discards successful prefixes on errors or late
// cancel.
func TestEventDiscoveryFailure(t *testing.T) {

	for _, name := range []string{"sources", "fields"} {
		for _, failure := range []string{"warehouse", "late cancellation"} {
			t.Run(name+"/"+failure, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				backend := &eventDiscoveryTestWarehouse{
					page:   warehouses.EventSourceDiscoveryPage{Sources: []warehouses.ObservedEventSource{{Type: "page"}}},
					sample: warehouses.EventFieldSample{Rows: []warehouses.EventFieldSampleRow{{}}},
				}
				if failure == "warehouse" {
					backend.err = errors.New("warehouse failure")
				} else {
					backend.cancel = cancel
				}
				store := &Store{mc: newModeCoordinator(state.Normal)}
				store.wh.Store(wrapWarehouse(backend))
				var err error
				partial := false
				if name == "sources" {
					result, readErr := store.ObservedEventSources(ctx, warehouses.EventSourceDiscoveryRequest{
						Window: eventDiscoveryTestSampleRequest().Window, Limit: 100,
					})
					err, partial = readErr, result.Sources != nil
				} else {
					result, readErr := store.ObservedEventFields(ctx, eventDiscoveryTestSampleRequest())
					err, partial = readErr, result.Fields != nil || result.Roots != nil
				}
				if err != nil {
					_, ok := errors.AsType[*WarehouseError](err)
					if !ok || partial || backend.queries != 1 {
						t.Fatalf("expected one failed read without partial data, got %v, partial=%t", err, partial)
					}
					return
				}
				t.Fatal("expected failed discovery, got success")
			})
		}
	}

}

// TestEventDiscoveryFieldLimits checks stable path admission and bounded
// output.
func TestEventDiscoveryFieldLimits(t *testing.T) {

	request := eventDiscoveryTestSampleRequest()
	keys := make([]string, 300)
	for i := range keys {
		keys[i] = fmt.Sprintf(`"p%03d":%d`, i, i)
	}
	first := warehouses.EventFieldSampleRow{Properties: warehouses.EventJSONSample{
		Present: true, Value: []byte("{" + strings.Join(keys, ",") + "}"),
	}}
	second := first
	reversed := slices.Clone(keys)
	slices.Reverse(reversed)
	second.Properties.Value = []byte("{" + strings.Join(reversed, ",") + "}")
	result, err := inspectEventFields(request, warehouses.EventFieldSample{
		Rows: []warehouses.EventFieldSampleRow{first, second},
	})
	if err != nil {
		t.Fatalf("expected capped paths, got %v", err)
	}
	if len(result.Fields) != 256 || !result.Sample.FieldsTruncated {
		t.Fatalf("expected 256 paths and truncation, got %+v", result.Sample)
	}
	for i, field := range result.Fields {
		if field.Property != fmt.Sprintf("properties.p%03d", i) || field.Present != 2 {
			t.Fatalf("expected stable path with full-sample count, got %+v", field)
		}
	}

	for _, document := range []string{
		strings.Repeat(`{"x":`, 10) + "0" + strings.Repeat("}", 10),
		`{"` + strings.Repeat("a", 30000) + `":1,"` + strings.Repeat("b", 30000) + `":2}`,
		`{"` + strings.Repeat("<", 10000) + `":1,"` + strings.Repeat(">", 10000) + `":2}`,
	} {
		row := warehouses.EventFieldSampleRow{Properties: warehouses.EventJSONSample{Present: true, Value: []byte(document)}}
		result, err := inspectEventFields(request, warehouses.EventFieldSample{Rows: []warehouses.EventFieldSampleRow{row}})
		if err != nil {
			t.Fatalf("expected limited field inspection, got %v", err)
		}
		encoded, err := jsonv1.Marshal(result)
		if err != nil {
			t.Fatalf("expected encoded field inspection, got %v", err)
		}
		if !result.Sample.FieldsTruncated || len(encoded) > eventDiscoveryResultBytes {
			t.Fatalf("expected path truncation within response budget, got %d bytes, %+v", len(encoded), result.Sample)
		}
	}

}

// TestEventDiscoveryFields distinguishes observed types, absence and unread
// roots.
func TestEventDiscoveryFields(t *testing.T) {

	request := eventDiscoveryTestSampleRequest()
	root := func(value string) warehouses.EventJSONSample {
		return warehouses.EventJSONSample{Present: true, Value: []byte(value)}
	}
	document := `{"amount":10,"coupon":null,"nested":{"x":true},` +
		`"arr":[{"hidden":1}],"user.rating":3,"":false}`
	sample := warehouses.EventFieldSample{Rows: []warehouses.EventFieldSampleRow{
		{Traits: root(`{"age":30}`), Properties: root(document)},
		{Properties: warehouses.EventJSONSample{Present: true, Uninspected: true}},
		{Traits: root(`"tag"`), Properties: root(`{"nested":null,"amount":"10"}`)},
	}}
	result, err := inspectEventFields(request, sample)
	if err != nil {
		t.Fatalf("expected observed fields, got %v", err)
	}
	if result.Sample.Rows != 3 || result.Sample.FieldsTruncated ||
		result.Roots["properties"].Present != 3 || result.Roots["properties"].Uninspected != 1 ||
		result.Roots["properties"].Kinds["object"] != 2 || result.Roots["traits"].Kinds["string"] != 1 {
		t.Fatalf("expected separate inspected/uninspected roots, got %+v", result)
	}
	want := map[string]struct {
		present  int
		property string
		kinds    map[string]int
	}{
		`["traits","age"]`:             {1, "traits.age", map[string]int{"number": 1}},
		`["properties",""]`:            {1, "", map[string]int{"boolean": 1}},
		`["properties","amount"]`:      {2, "properties.amount", map[string]int{"number": 1, "string": 1}},
		`["properties","arr"]`:         {1, "properties.arr", map[string]int{"array": 1}},
		`["properties","coupon"]`:      {1, "properties.coupon", map[string]int{"null": 1}},
		`["properties","nested"]`:      {2, "properties.nested", map[string]int{"object": 1, "null": 1}},
		`["properties","nested","x"]`:  {1, "properties.nested.x", map[string]int{"boolean": 1}},
		`["properties","user.rating"]`: {1, "", map[string]int{"number": 1}},
	}
	if len(result.Fields) != len(want) {
		t.Fatalf("expected %d paths without array traversal, got %+v", len(want), result.Fields)
	}
	for _, field := range result.Fields {
		path, err := json.Marshal(field.Path)
		if err != nil {
			t.Fatalf("expected JSON path, got %v", err)
		}
		expected, ok := want[string(path)]
		if !ok || field.Present != expected.present || field.Property != expected.property ||
			!reflect.DeepEqual(field.Kinds, expected.kinds) {
			t.Fatalf("expected path %s as %+v, got %+v", path, expected, field)
		}
	}

	t.Run("late path after unread root", func(t *testing.T) {
		sample := warehouses.EventFieldSample{Rows: []warehouses.EventFieldSampleRow{
			{Properties: warehouses.EventJSONSample{Present: true, Uninspected: true}}, {Properties: root(`{"late":1}`)},
		}}
		result, err := inspectEventFields(request, sample)
		if err != nil {
			t.Fatalf("expected late observed path, got %v", err)
		}
		if len(result.Fields) != 1 || result.Sample.Rows-result.Roots["properties"].Uninspected-
			result.Fields[0].Present != 0 || result.Sample.FieldsTruncated {
			t.Fatalf("expected no inferred absence and no path truncation, got %+v", result)
		}
	})

	t.Run("empty sample", func(t *testing.T) {
		result, err := inspectEventFields(request, warehouses.EventFieldSample{})
		if err != nil {
			t.Fatalf("expected empty sample, got %v", err)
		}
		if result.Sample.Rows != 0 || result.Fields == nil || len(result.Fields) != 0 ||
			len(result.Roots) != 2 || result.Roots["traits"].Present != 0 {
			t.Fatalf("expected empty arrays and zero root counters, got %+v", result)
		}
	})

}

// TestEventDiscoveryInvalidSamples rejects malformed or unsafe warehouse
// samples.
func TestEventDiscoveryInvalidSamples(t *testing.T) {
	request := eventDiscoveryTestSampleRequest()
	for _, root := range []warehouses.EventJSONSample{
		{Value: []byte(`{}`)}, {Uninspected: true}, {Present: true},
		{Present: true, Value: []byte(`{`)}, {Present: true, Value: []byte{'"', 0xff, '"'}},
		{Present: true, Value: []byte(strings.Repeat("x", 65537))},
		{Present: true, Uninspected: true, Value: []byte(`{}`)},
	} {
		_, err := inspectEventFields(request, warehouses.EventFieldSample{
			Rows: []warehouses.EventFieldSampleRow{{Properties: root}},
		})
		if err != nil {
			if strings.Contains(err.Error(), string(root.Value)) && len(root.Value) > 2 {
				t.Fatalf("expected sanitized error, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid sample rejection, got success")
	}
}

// TestEventDiscoveryStore checks operation modes and encoded source budgets.
func TestEventDiscoveryStore(t *testing.T) {

	request := eventDiscoveryTestSampleRequest()
	for _, mode := range []state.WarehouseMode{state.Normal, state.Inspection, state.Maintenance} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			backend := &eventDiscoveryTestWarehouse{}
			store := &Store{mc: newModeCoordinator(mode)}
			store.wh.Store(wrapWarehouse(backend))
			_, err := store.ObservedEventFields(t.Context(), request)
			if err != nil {
				if mode != state.Maintenance || !errors.Is(err, ErrMaintenanceMode) || backend.queries != 0 {
					t.Fatalf("expected maintenance rejection before query, got %v, %d queries", err, backend.queries)
				}
				return
			}
			if mode == state.Maintenance || backend.queries != 1 {
				t.Fatalf("expected one admitted query, got %d in %v", backend.queries, mode)
			}
		})
	}
	for _, fill := range []string{"\x01", "<", "\u2028"} {
		backend := &eventDiscoveryTestWarehouse{page: warehouses.EventSourceDiscoveryPage{}}
		for i := range 96 {
			name := fmt.Sprintf("%03d", i) + strings.Repeat(fill, 1021/len(fill))
			backend.page.Sources = append(backend.page.Sources,
				warehouses.ObservedEventSource{Type: "page", RecordedName: &name})
		}
		store := &Store{mc: newModeCoordinator(state.Normal)}
		store.wh.Store(wrapWarehouse(backend))
		page, err := store.ObservedEventSources(t.Context(), warehouses.EventSourceDiscoveryRequest{
			Window: request.Window, Limit: 500,
		})
		if err != nil {
			t.Fatalf("expected encoded source prefix, got %v", err)
		}
		encoded, err := jsonv1.Marshal(page)
		if err != nil {
			t.Fatalf("expected encoded source result, got %v", err)
		}
		if !page.Truncated || len(page.Sources) >= 96 || len(encoded) > eventDiscoveryResultBytes {
			t.Fatalf("expected escaped-name truncation within 128 KiB, got %d sources and %d bytes",
				len(page.Sources), len(encoded))
		}
	}

}

// TestEventDiscoveryTimeout checks the datastore deadline with virtual time.
func TestEventDiscoveryTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		backend := &eventDiscoveryTestWarehouse{wait: true}
		store := &Store{mc: newModeCoordinator(state.Normal)}
		store.wh.Store(wrapWarehouse(backend))
		start := time.Now()
		_, err := store.ObservedEventFields(t.Context(), eventDiscoveryTestSampleRequest())
		if err != nil {
			unavailable, ok := errors.AsType[*WarehouseError](err)
			if !ok || !errors.Is(unavailable.Err, context.DeadlineExceeded) || time.Since(start) != 30*time.Second {
				t.Fatalf("expected 30-second datastore deadline, got %v after %v", err, time.Since(start))
			}
			return
		}
		t.Fatal("expected deadline failure, got success")
	})
}

func eventDiscoveryTestSampleRequest() warehouses.EventFieldSampleRequest {
	return warehouses.EventFieldSampleRequest{Window: warehouses.EventDiscoveryWindow{
		Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}, Source: warehouses.ObservedEventSource{Type: "track"}, Limit: 1000}
}
