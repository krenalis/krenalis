// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

type recordFetcherFunc func(context.Context, connectors.Targets, time.Time, string, types.Type) ([]connectors.Record, string, error)

func (recordFetcherFunc) RecordSchema(context.Context, connectors.Targets, connectors.Role) (types.Type, error) {
	panic("unexpected call to RecordSchema")
}

func (f recordFetcherFunc) Records(ctx context.Context, target connectors.Targets, updatedAt time.Time, cursor string, schema types.Type) ([]connectors.Record, string, error) {
	return f(ctx, target, updatedAt, cursor, schema)
}

// TestAppRecordsPreservesConnectorRecordError verifies that an application
// record preserves the connector error and does not read its update time or
// attributes when the record has an error.
func TestAppRecordsPreservesConnectorRecordError(t *testing.T) {

	recordErr := errors.New("record cannot be read")
	schema := types.Object([]types.Property{
		{Name: "email", Type: types.String()},
	})
	records := &appRecords{
		schema:    schema,
		appSchema: schema,
		updatedAt: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		connector: "test",
		inner: recordFetcherFunc(func(context.Context, connectors.Targets, time.Time, string, types.Type) ([]connectors.Record, string, error) {
			return []connectors.Record{{
				ID:  "user-1",
				Err: recordErr,
			}}, "", io.EOF
		}),
	}

	var got []Record
	for record := range records.All(t.Context()) {
		got = append(got, record)
	}
	if err := records.Err(); err != nil {
		t.Fatalf("expected no iterator error, got %s", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one record, got %d", len(got))
	}
	record := got[0]
	if record.ID != "user-1" {
		t.Fatalf("expected record ID %q, got %q", "user-1", record.ID)
	}
	if !errors.Is(record.Err, recordErr) {
		t.Fatalf("expected record error %q, got %v", recordErr, record.Err)
	}
	if !record.UpdatedAt.IsZero() {
		t.Fatalf("expected zero update time, got %s", record.UpdatedAt)
	}
	if record.Attributes != nil {
		t.Fatalf("expected nil attributes, got %#v", record.Attributes)
	}
	if !records.Last() {
		t.Fatal("expected the errored record to be the last record")
	}
}

func Test_sameValue(t *testing.T) {

	object := types.Object([]types.Property{
		{Name: "foo", Type: types.String()},
		{Name: "boo", Type: types.Array(types.Boolean())},
	})

	tests := []struct {
		t        types.Type
		v, v2    any
		expected bool
	}{
		{t: types.String(), v: nil, v2: nil, expected: true},
		{t: types.String(), v: nil, v2: 5, expected: false},
		{t: types.Int(32), v: 4, v2: 4, expected: true},
		{t: types.Int(32), v: 4, v2: nil, expected: false},
		{t: types.Float(64), v: 12.9037, v2: 12.9037, expected: true},
		{t: types.JSON(), v: nil, v2: nil, expected: true},
		{t: types.JSON(), v: json.Value(`null`), v2: json.Value(`null`), expected: true},
		{t: types.JSON(), v: json.Value(`{"a":3,"b":[1,2]}`), v2: json.Value(`{"a":3,"b":[1,2]}`), expected: true},
		{t: types.JSON(), v: json.Value(`{"a":3,"b":[1,2]}`), v2: json.Value(`{"a":3,"c":true}`), expected: false},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: []any{"a", "b"}, expected: true},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: []any{"b", "a"}, expected: false},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: []any{}, expected: false},
		{t: types.Array(types.String()), v: []any{"a", "b"}, v2: nil, expected: false},
		{t: object, v: map[string]any{}, v2: nil, expected: false},
		{t: object, v: map[string]any{}, v2: map[string]any{}, expected: true},
		{t: object, v: map[string]any{"foo": "a", "boo": []any{true, false, true}}, v2: map[string]any{"foo": "a", "boo": []any{true, false, true}}, expected: true},
		{t: object, v: map[string]any{"foo": "a", "boo": []any{true, false, true}}, v2: map[string]any{"foo": "a", "boo": []any{true, true, true}}, expected: false},
		{t: object, v: map[string]any{"foo": "a", "boo": []any{true, false, true}}, v2: nil, expected: false},
		{t: types.Map(types.Int(32)), v: map[string]any{"a": 5, "b": 0}, v2: map[string]any{"a": 5, "b": 0}, expected: true},
		{t: types.Map(types.Int(32)), v: map[string]any{"a": 5, "b": 0}, v2: map[string]any{"b": 0, "a": 5}, expected: true},
		{t: types.Map(types.Int(32)), v: map[string]any{"a": 5, "b": 0}, v2: map[string]any{"b": 0, "a": 3}, expected: false},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := sameValue(test.t, test.v, test.v2)
			if test.expected != got {
				t.Fatalf("expected %t, got %t", test.expected, got)
			}
		})
	}

}
