//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package mappings_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/apis/transformers/mappings"
	"github.com/meergo/meergo/types"
)

func Test_RelevantCases(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.JSON()},
	})

	tests := []struct {
		name       string
		properties map[string]any
		expr       string
		dt         types.Type
		nullable   bool
		layouts    *state.TimeLayouts
		value      any
		err        error
	}{
		{
			name: `Mapping a JSON null value, from the JSON connector, to a property of the user schema, that property will have no value`,
			expr: `a`, dt: types.Text(), value: mappings.Void, properties: map[string]any{"a": json.RawMessage("null")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expression, err := mappings.Compile(test.expr, schema, test.dt, test.nullable, test.layouts)
			if err != nil {
				t.Fatalf("unexpected compilation error %s", err)
			}
			v, err := expression.Eval(test.properties, mappings.None)
			if err != nil {
				if test.err == nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if err.Error() != test.err.Error() {
					t.Fatalf("expected error %q, got error %q", test.err.Error(), err.Error())
				}
				return
			}
			if test.err != nil {
				t.Fatalf("expected error %q, got no error", test.err)
			}
			if !reflect.DeepEqual(test.value, v) {
				if test.value == mappings.Void {
					t.Fatalf("expected void, got value %#v (type %T)", v, v)
				}
				if v == mappings.Void {
					t.Fatalf("expected value %#v (type %T), got void", test.value, test.value)
				}
				t.Fatalf("expected value %#v (type %T), got value %#v (type %T)", test.value, test.value, v, v)
			}
		})
	}
}
