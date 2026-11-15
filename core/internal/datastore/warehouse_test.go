// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

type capabilityWarehouse struct {
	errorWarehouse
	verified bool
}

// DiscoverEventSources implements [warehouses.EventDiscoveryWarehouse].
func (w *capabilityWarehouse) DiscoverEventSources(ctx context.Context, request warehouses.EventSourceDiscoveryRequest) (warehouses.EventSourceDiscoveryPage, error) {
	return warehouses.EventSourceDiscoveryPage{Sources: []warehouses.ObservedEventSource{{Type: "track"}}}, w.err
}

// ExplainProfileFilter supplies a fixture explanation.
func (w *capabilityWarehouse) ExplainProfileFilter(ctx context.Context, filter warehouses.Filter, source warehouses.ProfileSource, profile string) ([]warehouses.FilterNodeOutcome, error) {
	return []warehouses.FilterNodeOutcome{{Truth: warehouses.FilterMatch}}, w.err
}

// ProfileFilterVerified implements [warehouses.ProfileFilterWarehouse].
func (w *capabilityWarehouse) ProfileFilterVerified() bool {
	return w.verified
}

// ProfileOutcomes implements [warehouses.ProfileFilterWarehouse].
func (w *capabilityWarehouse) ProfileOutcomes(ctx context.Context, filter warehouses.Filter, request warehouses.ProfileOutcomeRequest) (*warehouses.ProfileOutcomes, error) {
	return &warehouses.ProfileOutcomes{Match: 3}, w.err
}

// SampleEventFields implements [warehouses.EventDiscoveryWarehouse].
func (w *capabilityWarehouse) SampleEventFields(ctx context.Context, request warehouses.EventFieldSampleRequest) (warehouses.EventFieldSample, error) {
	return warehouses.EventFieldSample{Rows: []warehouses.EventFieldSampleRow{{}}}, w.err
}

type errorWarehouse struct {
	warehouses.Warehouse
	rows warehouses.Rows
	err  error
}

// AlterProfileSchema implements [warehouses.Warehouse].
func (w *errorWarehouse) AlterProfileSchema(ctx context.Context, opID string, columns []warehouses.Column, operations []warehouses.AlterOperation) error {
	return w.err
}

// Query implements [warehouses.Warehouse].
func (w *errorWarehouse) Query(ctx context.Context, query warehouses.RowQuery, withTotal bool) (warehouses.Rows, int, error) {
	return w.rows, 2, w.err
}

type errorRows struct {
	values   []any
	next     bool
	closed   bool
	scanErr  error
	rowsErr  error
	closeErr error
}

// Close implements [warehouses.Rows].
func (r *errorRows) Close() error {
	r.closed = true
	return r.closeErr
}

// Err implements [warehouses.Rows].
func (r *errorRows) Err() error {
	return r.rowsErr
}

// Next implements [warehouses.Rows].
func (r *errorRows) Next() bool {
	next := r.next
	r.next = false
	return next
}

// Scan implements [warehouses.Rows].
func (r *errorRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	copy(dest, r.values)
	return nil
}

// TestWarehouseCapabilities preserves optional results, gates and errors.
func TestWarehouseCapabilities(t *testing.T) {
	for _, cause := range []error{nil, errors.New("capability failed")} {
		t.Run(fmt.Sprint(cause), func(t *testing.T) {
			inner := &capabilityWarehouse{errorWarehouse: errorWarehouse{err: cause}, verified: true}
			dw := &warehouse{inner: inner}
			if !dw.ProfileFilterVerified() {
				t.Fatal("expected verified provider, got closed gate")
			}
			inner.verified = false
			if dw.ProfileFilterVerified() {
				t.Fatal("expected unverified provider, got open gate")
			}
			result, err := dw.ProfileOutcomes(t.Context(), warehouses.Filter{}, warehouses.ProfileOutcomeRequest{})
			assertWarehouseError(t, err, cause)
			if result == nil || result.Match != 3 {
				t.Fatalf("expected forwarded profile result, got %+v", result)
			}
			nodes, err := dw.ExplainProfileFilter(t.Context(), warehouses.Filter{}, warehouses.ProfileSource{}, "profile")
			assertWarehouseError(t, err, cause)
			if len(nodes) != 1 || nodes[0].Truth != warehouses.FilterMatch {
				t.Fatalf("expected forwarded explanation, got %+v", nodes)
			}
			page, err := dw.DiscoverEventSources(t.Context(), warehouses.EventSourceDiscoveryRequest{})
			assertWarehouseError(t, err, cause)
			if len(page.Sources) != 1 || page.Sources[0].Type != "track" {
				t.Fatalf("expected forwarded event sources, got %+v", page)
			}
			sample, err := dw.SampleEventFields(t.Context(), warehouses.EventFieldSampleRequest{})
			assertWarehouseError(t, err, cause)
			if len(sample.Rows) != 1 {
				t.Fatalf("expected forwarded event sample, got %+v", sample)
			}
		})
	}
}

// TestWarehouseErrors verifies that wrapping adds exactly one *WarehouseError
// and preserves the original cause, including *warehouses.OperationError.
func TestWarehouseErrors(t *testing.T) {
	tests := []struct {
		name  string
		cause error
	}{
		{name: "success"},
		{name: "generic", cause: errors.New("warehouse failed")},
		{name: "settings", cause: warehouses.SettingsErrorf("invalid settings")},
		{name: "operation", cause: warehouses.NewOperationError(errors.New("operation failed"))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dw := &warehouse{inner: &errorWarehouse{err: test.cause}}
			err := dw.AlterProfileSchema(t.Context(), "operation", nil, nil)
			assertWarehouseError(t, err, test.cause)
			if err != nil {
				if want, ok := test.cause.(*warehouses.OperationError); ok {
					got, ok := errors.AsType[*warehouses.OperationError](err)
					if !ok || got != want {
						t.Fatalf("expected original OperationError, got %v", got)
					}
				}
			}
		})
	}
}

// TestWarehouseMissingCapabilities fails without calling an absent backend.
func TestWarehouseMissingCapabilities(t *testing.T) {
	dw := &warehouse{inner: &errorWarehouse{}}
	if dw.ProfileFilterVerified() {
		t.Fatal("expected closed gate without profile capability, got open gate")
	}
	tests := []struct {
		name string
		call func() error
	}{
		{"profiles", func() error {
			_, err := dw.ProfileOutcomes(t.Context(), warehouses.Filter{}, warehouses.ProfileOutcomeRequest{})
			return err
		}},
		{"explanation", func() error {
			_, err := dw.ExplainProfileFilter(t.Context(), warehouses.Filter{}, warehouses.ProfileSource{}, "profile")
			return err
		}},
		{"sources", func() error {
			_, err := dw.DiscoverEventSources(t.Context(), warehouses.EventSourceDiscoveryRequest{})
			return err
		}},
		{"sample", func() error {
			_, err := dw.SampleEventFields(t.Context(), warehouses.EventFieldSampleRequest{})
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			if err != nil {
				return
			}
			t.Fatal("expected missing capability error, got success")
		})
	}
}

// TestWarehouseOptionalCapabilities exposes only capabilities the provider has.
func TestWarehouseOptionalCapabilities(t *testing.T) {
	base := &errorWarehouse{}
	capable := &capabilityWarehouse{verified: true}
	tests := []struct {
		name         string
		inner        warehouses.Warehouse
		profiles     bool
		explanations bool
		discovery    bool
	}{
		{"none", base, false, false, false},
		{"profiles", struct {
			warehouses.Warehouse
			warehouses.ProfileFilterWarehouse
		}{base, capable}, true, false, false},
		{"explanations", struct {
			warehouses.Warehouse
			warehouses.ProfileFilterExplanationWarehouse
		}{base, capable}, true, true, false},
		{"discovery", struct {
			warehouses.Warehouse
			warehouses.EventDiscoveryWarehouse
		}{base, capable}, false, false, true},
		{"profiles and discovery", struct {
			warehouses.Warehouse
			warehouses.ProfileFilterWarehouse
			warehouses.EventDiscoveryWarehouse
		}{base, capable, capable}, true, false, true},
		{"all", capable, true, true, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dw := wrapWarehouse(test.inner)
			filter, profiles := dw.(warehouses.ProfileFilterWarehouse)
			_, explanations := dw.(warehouses.ProfileFilterExplanationWarehouse)
			_, discovery := dw.(warehouses.EventDiscoveryWarehouse)
			if profiles != test.profiles || explanations != test.explanations || discovery != test.discovery {
				t.Fatalf("expected capabilities %t/%t/%t, got %t/%t/%t", test.profiles, test.explanations,
					test.discovery, profiles, explanations, discovery)
			}
			if profiles && !filter.ProfileFilterVerified() {
				t.Fatal("expected forwarded verification gate, got closed gate")
			}
		})
	}
}

// TestWarehouseRows verifies that queries preserve row values and
// counts and wrap errors from queries, scanning, iteration, and closing.
func TestWarehouseRows(t *testing.T) {

	innerRows := &errorRows{values: []any{"value", 7}, next: true}
	inner := &errorWarehouse{rows: innerRows}
	dw := &warehouse{inner: inner}
	query := func() (warehouses.Rows, int, error) {
		return dw.Query(t.Context(), warehouses.RowQuery{}, true)
	}
	rows, count, err := query()
	if err != nil {
		t.Fatalf("expected successful query, got %v", err)
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
	if rows == nil || !rows.Next() {
		t.Fatal("expected a row, got none")
	}
	values := make([]any, 2)
	err = rows.Scan(values...)
	if err != nil {
		t.Fatalf("expected successful scan, got %v", err)
	}
	if values[0] != "value" || values[1] != 7 {
		t.Fatalf("expected [value 7], got %v", values)
	}
	if rows.Next() {
		t.Fatal("expected end of rows, got another row")
	}
	assertWarehouseError(t, rows.Err(), nil)

	innerRows.rowsErr = errors.New("iteration failed")
	assertWarehouseError(t, rows.Err(), innerRows.rowsErr)
	innerRows.closeErr = errors.New("close failed")
	err = rows.Close()
	assertWarehouseError(t, err, innerRows.closeErr)
	if !innerRows.closed {
		t.Fatal("expected closed rows, got open rows")
	}

	scanRows := &errorRows{next: true, scanErr: errors.New("scan failed")}
	inner.rows = scanRows
	rows, _, err = query()
	if err != nil {
		t.Fatalf("expected successful query, got %v", err)
	}
	if rows == nil || !rows.Next() {
		t.Fatal("expected a row, got none")
	}
	defer rows.Close()
	err = rows.Scan(values...)
	assertWarehouseError(t, err, scanRows.scanErr)

	inner.rows = nil
	inner.err = errors.New("query failed")
	rows, _, err = query()
	assertWarehouseError(t, err, inner.err)
	if rows != nil {
		t.Fatalf("expected nil rows on query failure, got %T", rows)
	}

}

// assertWarehouseError checks that err is nil if cause is nil, and otherwise is
// a *WarehouseError wrapping cause directly.
func assertWarehouseError(t *testing.T, err, cause error) {
	t.Helper()
	if err != nil {
		if cause == nil {
			t.Fatalf("expected no error, got %v", err)
		}
		wrapped, ok := err.(*WarehouseError)
		if !ok || wrapped == nil {
			t.Fatalf("expected non-nil *WarehouseError, got %#v", err)
		}
		if wrapped.Err != cause {
			t.Fatalf("expected original cause %v, got %v", cause, wrapped.Err)
		}
		unwrapped := errors.Unwrap(err)
		if unwrapped != cause {
			t.Fatalf("expected Unwrap() to return %v, got %v", cause, unwrapped)
		}
		if !errors.Is(err, cause) {
			t.Fatal("expected errors.Is(err, cause) to return true, got false")
		}
		return
	}
	if cause != nil {
		t.Fatalf("expected error wrapping %v, got nil", cause)
	}
}
