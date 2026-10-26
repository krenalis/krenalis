// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestFilterDateTimeOffsetRoundTrip preserves the direct and persisted
// policies.
func TestFilterDateTimeOffsetRoundTrip(t *testing.T) {
	schema := types.Object([]types.Property{{Name: "when", Type: types.DateTime()}})
	filter := &Filter{Operator: OpAnd, Rules: []FilterRule{
		&FilterCondition{Property: "when", Operator: OpIsAfter, Values: filterValues("2026-10-24T12:00:00.123456789+02:00")},
	}}
	where := mustPrepareFilter(t, filter, schema)
	value := where.Rules[0].(*state.WhereCondition).Values[0].(time.Time)
	if _, offset := value.Zone(); offset != 7200 {
		t.Fatalf("expected direct offset 7200, got %d", offset)
	}
	back := convertWhereToFilter(where, schema)
	got := back.Rules[0].(*FilterCondition).Values[0].String()
	if got != "2026-10-24T12:00:00.123456789+02:00" {
		t.Fatalf("expected preserved offset, got %s", got)
	}
	stored := state.MarshalWhere(where, schema)
	if !strings.Contains(string(stored), "2026-10-24T10:00:00.123456789Z") {
		t.Fatalf("expected existing persisted UTC normalization, got %s", stored)
	}
}

// TestPrepareFilterBodyFieldOrder checks schema-independent body decoding.
func TestPrepareFilterBodyFieldOrder(t *testing.T) {
	schema := `{"kind":"object","properties":[{"name":"account","type":{"kind":"int","bitSize":64}}]}`
	filter := `{"operator":"and","rules":[{"property":"account","operator":"is","values":[9223372036854775807]}]}`
	for _, body := range []string{
		`{"filter":` + filter + `,"inSchema":` + schema + `}`,
		`{"inSchema":` + schema + `,"filter":` + filter + `}`,
	} {
		var pipeline PipelineToSet
		err := json.Unmarshal([]byte(body), &pipeline)
		if err != nil {
			t.Fatalf("expected schema-independent body decoding, got %v", err)
		}
		where, _, err := prepareFilter(pipeline.Filter, pipeline.InSchema)
		if err != nil {
			t.Fatalf("expected filter preparation, got %v", err)
		}
		got := where.Rules[0].(*state.WhereCondition).Values[0]
		if got != int(9223372036854775807) {
			t.Fatalf("expected exact signed maximum, got %v", got)
		}
	}
}

// TestPrepareFilterMissingSchemaProperty preserves the missing-path error type.
func TestPrepareFilterMissingSchemaProperty(t *testing.T) {
	schema := types.Object([]types.Property{{Name: "value", Type: types.JSON()}})
	filter := &Filter{Operator: OpAnd, Rules: []FilterRule{
		&FilterCondition{Property: "absent", Operator: OpIs, Values: filterValues("x")},
	}}
	_, _, err := prepareFilter(filter, schema)
	if err != nil {
		if missing, ok := errors.AsType[types.PathNotExistError](err); ok && missing.Path == "absent" {
			return
		}
		t.Fatalf("expected PathNotExistError, got %v", err)
	}
	t.Fatal("expected missing schema property error, got nil")
}

// TestPrepareFilterOwnsJSONOperands checks ownership without mutating the
// filter.
func TestPrepareFilterOwnsJSONOperands(t *testing.T) {
	schema := types.Object([]types.Property{{Name: "value", Type: types.JSON()}})
	raw := json.Value(`"EUR"`)
	filter := &Filter{Operator: OpAnd, Rules: []FilterRule{
		&FilterCondition{Property: "value", Operator: OpIs, Values: []json.Value{raw}},
	}}
	before, err := json.Marshal(filter)
	if err != nil {
		t.Fatalf("expected valid input encoding, got %v", err)
	}
	where, _, err := prepareFilter(filter, schema)
	if err != nil {
		t.Fatalf("expected prepared filter, got %v", err)
	}
	after, err := json.Marshal(filter)
	if err != nil {
		t.Fatalf("expected unchanged input encoding, got %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("expected unchanged input %s, got %s", before, after)
	}
	copy(raw, `"USD"`)
	got := where.Rules[0].(*state.WhereCondition).Values[0].(json.Value)
	if string(got) != `"EUR"` {
		t.Fatalf("expected owned operand EUR, got %s", got)
	}
}

// TestPrepareTypedJSONFilter checks the approved JSON operand restrictions.
func TestPrepareTypedJSONFilter(t *testing.T) {
	schema := types.Object([]types.Property{
		{Name: "value", Type: types.JSON(), ReadOptional: true},
		{Name: "text", Type: types.String()},
	})
	for _, test := range []struct {
		name     string
		property string
		op       FilterOperator
		values   []json.Value
		invalid  bool
	}{
		{"mixed membership", "value", OpIsOneOf, filterValues(json.Value("100"), "100", true, false, nil, ""), false},
		{"mixed negative membership", "value", OpIsNotOneOf, filterValues(json.Value("100"), false, nil), false},
		{"null membership", "value", OpIsOneOf, filterValues(nil), false},
		{"null equality", "value", OpIs, filterValues(nil), true},
		{"boolean equality", "value", OpIs, filterValues(true), true},
		{"boolean contains", "value", OpContains, filterValues(false), true},
		{"object membership", "value", OpIsOneOf, filterValues(json.Value("{}")), true},
		{"array membership", "value", OpIsOneOf, filterValues(json.Value("[]")), true},
		{"String interval", "value", OpIsBetween, filterValues("100", "9"), false},
		{"Number interval", "value", OpIsBetween, filterValues(json.Value("1e2"), json.Value("100.0")), false},
		{"mixed interval", "value", OpIsBetween, filterValues("100", json.Value("100")), true},
		{"mixed negative interval", "value", OpIsNotBetween, filterValues(json.Value("100"), "100"), true},
		{"Number prefix", "value", OpStartsWith, filterValues(json.Value("100")), true},
		{"empty JSON String", "value", OpIs, filterValues(""), false},
		{"empty ordinary String", "text", OpIs, filterValues(""), true},
		{"unary absent", "value", OpIsNull, nil, false},
		{"unary empty slice", "value", OpIsNull, []json.Value{}, true},
		{"unary Null operand", "value", OpIsNull, filterValues(nil), true},
		{"raw nil", "value", OpIs, []json.Value{nil}, true},
		{"raw invalid", "value", OpIs, []json.Value{json.Value("1e")}, true},
		{"raw multiple values", "value", OpIs, []json.Value{json.Value("1 2")}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			filter := &Filter{Operator: OpAnd, Rules: []FilterRule{
				&FilterCondition{Property: test.property, Operator: test.op, Values: test.values},
			}}
			where, paths, err := prepareFilter(filter, schema)
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected valid filter, got %v", err)
				}
				return
			}
			if test.invalid {
				t.Fatalf("expected invalid filter, got %#v", where)
			}
			if !reflect.DeepEqual(paths, []string{test.property}) {
				t.Fatalf("expected path %s, got %v", test.property, paths)
			}
		})
	}
}

// TestTypedFilterConditionNilUnmarshal rejects a nil receiver without
// panicking.
func TestTypedFilterConditionNilUnmarshal(t *testing.T) {
	var condition *FilterCondition
	err := condition.UnmarshalJSON([]byte(`{"property":"value","operator":"is","values":[100]}`))
	if err != nil {
		return
	}
	t.Fatal("expected nil receiver error, got nil")
}

// TestTypedFilterIntegerRoundTrip preserves exact integer Numbers through core.
func TestTypedFilterIntegerRoundTrip(t *testing.T) {
	for _, test := range []struct {
		typ  types.Type
		raw  string
		want any
	}{
		{types.Int(64), "-9223372036854775808", int(-9223372036854775808)},
		{types.Int(64), "9223372036854775807", int(9223372036854775807)},
		{types.Int(64).Unsigned(), "18446744073709551615", uint(18446744073709551615)},
		{types.Int(8), "9007199254740993", int(9007199254740993)},
	} {
		t.Run(test.raw, func(t *testing.T) {
			data := []byte(`{"operator":"and","rules":[{"property":"account","operator":"is",` +
				`"values":[` + test.raw + `]}]}`)
			var filter Filter
			err := json.Unmarshal(data, &filter)
			if err != nil {
				t.Fatalf("expected exact Number decoding, got %v", err)
			}
			schema := types.Object([]types.Property{{Name: "account", Type: test.typ}})
			where := mustPrepareFilter(t, &filter, schema)
			if got := where.Rules[0].(*state.WhereCondition).Values[0]; got != test.want {
				t.Fatalf("expected %#v, got %#v", test.want, got)
			}
			got, err := json.Marshal(convertWhereToFilter(where, schema))
			if err != nil {
				t.Fatalf("expected integer filter encoding, got %v", err)
			}
			if !bytes.Equal(got, data) {
				t.Fatalf("expected exact Number round trip %s, got %s", data, got)
			}
		})
	}
}

// TestTypedFilterJSONRoundTrip preserves kinds on repeated public decoding.
func TestTypedFilterJSONRoundTrip(t *testing.T) {
	data := []byte(`{"operator":"and","rules":[{"property":"value","operator":"is one of",` +
		`"values":[100,100.0,1e2,"100",true,false,null,""]}]}`)
	var filter Filter
	for range 2 {
		err := json.Unmarshal(data, &filter)
		if err != nil {
			t.Fatalf("expected typed filter decode, got %v", err)
		}
		got, err := json.Marshal(&filter)
		if err != nil {
			t.Fatalf("expected typed filter encoding, got %v", err)
		}
		if !bytes.Equal(got, data) {
			t.Fatalf("expected %s, got %s", data, got)
		}
	}
	var condition FilterCondition
	for _, text := range []string{
		`{"property":"value","operator":"is one of","values":[100,"100",false,null]}`,
		`{"property":"value","operator":"is one of","values":[true,"",1e2]}`,
	} {
		err := json.Unmarshal([]byte(text), &condition)
		if err != nil {
			t.Fatalf("expected replacement of standalone condition, got %v", err)
		}
		got, err := json.Marshal(&condition)
		if err != nil {
			t.Fatalf("expected standalone condition encoding, got %v", err)
		}
		if string(got) != text {
			t.Fatalf("expected %s, got %s", text, got)
		}
	}
}
