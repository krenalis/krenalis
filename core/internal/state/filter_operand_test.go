// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestDecodeFilterIntegerWidths keeps 64-bit thresholds at every integer width.
func TestDecodeFilterIntegerWidths(t *testing.T) {
	for _, size := range []int{8, 16, 24, 32, 64} {
		for _, unsigned := range []bool{false, true} {
			typ := types.Int(size)
			raw := json.Value("9223372036854775807")
			var want any = int(math.MaxInt64)
			if unsigned {
				typ, raw, want = typ.Unsigned(), json.Value("18446744073709551615"), uint(math.MaxUint64)
			}
			t.Run(strconv.Itoa(size)+"/unsigned="+strconv.FormatBool(unsigned), func(t *testing.T) {
				got, err := DecodeFilterOperand(raw, typ, OpIs)
				if err != nil {
					t.Fatalf("expected a 64-bit threshold for %s, got %v", typ, err)
				}
				if got != want {
					t.Fatalf("expected %#v, got %#v", want, got)
				}
				for _, invalid := range []json.Value{json.Value(`"100"`), json.Value("100.0"), json.Value("1e2")} {
					_, err := DecodeFilterOperand(invalid, typ, OpIs)
					if err != nil {
						continue
					}
					t.Fatalf("expected invalid integer operand %s, got nil error", invalid)
				}
			})
		}
	}
}

// TestDecodeFilterOperand checks operand kinds, parsers, and numeric limits.
func TestDecodeFilterOperand(t *testing.T) {
	for _, test := range []struct {
		name    string
		typ     types.Type
		op      WhereOperator
		raw     json.Value
		want    any
		invalid bool
	}{
		{"string", types.String(), OpIs, json.Value(`"100"`), "100", false},
		{"int8 threshold beyond column range", types.Int(8), OpIs, json.Value("1000"), 1000, false},
		{"signed minimum", types.Int(64), OpIs, json.Value("-9223372036854775808"), int(math.MinInt64), false},
		{"signed maximum", types.Int(64), OpIs, json.Value("9223372036854775807"), int(math.MaxInt64), false},
		{"unsigned maximum", types.Int(64).Unsigned(), OpIs, json.Value("18446744073709551615"), uint(math.MaxUint64), false},
		{"unsafe JS integer", types.Int(64), OpIs, json.Value("9007199254740993"), int(9007199254740993), false},
		{"signed overflow", types.Int(64), OpIs, json.Value("9223372036854775808"), nil, true},
		{"signed underflow", types.Int(64), OpIs, json.Value("-9223372036854775809"), nil, true},
		{"unsigned overflow", types.Int(64).Unsigned(), OpIs, json.Value("18446744073709551616"), nil, true},
		{"unsigned negative", types.Int(8).Unsigned(), OpIs, json.Value("-1"), nil, true},
		{"integer String", types.Int(64), OpIs, json.Value(`"100"`), nil, true},
		{"integer fraction", types.Int(64), OpIs, json.Value("100.0"), nil, true},
		{"integer exponent", types.Int(64), OpIs, json.Value("1e2"), nil, true},
		{"year", types.Year(), OpIs, json.Value("2026"), 2026, false},
		{"year fraction", types.Year(), OpIs, json.Value("2026.0"), nil, true},
		{"year exponent", types.Year(), OpIs, json.Value("2e3"), nil, true},
		{"year zero", types.Year(), OpIs, json.Value("0"), nil, true},
		{"float beyond decimal range", types.Float(64), OpIs, json.Value("1e100"), 1e100, false},
		{"float32 overflow", types.Float(32), OpIs, json.Value("1e100"), nil, true},
		{"float special String", types.Float(64), OpIs, json.Value(`"NaN"`), nil, true},
		{"decimal beyond P S", types.Decimal(5, 2), OpIs, json.Value("1000.00"), decimal.MustParse("1000"), false},
		{"decimal precision", types.Decimal(5, 2), OpIs, json.Value(strings.Repeat("9", 76)), decimal.MustParse(strings.Repeat("9", 76)), false},
		{"decimal scale", types.Decimal(5, 2), OpIs, json.Value("1e-37"), decimal.MustParse("1e-37"), false},
		{"decimal excessive precision", types.Decimal(5, 2), OpIs, json.Value("1e76"), nil, true},
		{"decimal excessive scale", types.Decimal(5, 2), OpIs, json.Value("1e-38"), nil, true},
		{"decimal String", types.Decimal(5, 2), OpIs, json.Value(`"100"`), nil, true},
		{"JSON Number", types.JSON(), OpIs, json.Value("100.0"), json.Value("100"), false},
		{"JSON exponent", types.JSON(), OpIs, json.Value("1e2"), json.Value("100"), false},
		{"JSON extreme zero", types.JSON(), OpIs, json.Value("-0.00e-20000000000000000000"), json.Value("0"), false},
		{"JSON String", types.JSON(), OpIs, json.Value(`"100"`), json.Value(`"100"`), false},
		{"JSON empty String", types.JSON(), OpContains, json.Value(`""`), json.Value(`""`), false},
		{"JSON Number precision", types.JSON(), OpIs, json.Value("1e76"), nil, true},
		{"JSON Number scale", types.JSON(), OpIs, json.Value("1e-38"), nil, true},
		{"JSON Boolean member", types.JSON(), OpIsOneOf, json.Value("false"), json.Value("false"), false},
		{"JSON Null member", types.JSON(), OpIsNotOneOf, json.Value("null"), json.Value("null"), false},
		{"JSON Boolean equality", types.JSON(), OpIs, json.Value("true"), nil, true},
		{"JSON Null contains", types.JSON(), OpContains, json.Value("null"), nil, true},
		{"JSON Number prefix", types.JSON(), OpStartsWith, json.Value("100"), nil, true},
		{"JSON object", types.JSON(), OpIsOneOf, json.Value("{}"), nil, true},
		{"JSON array", types.JSON(), OpIsOneOf, json.Value("[]"), nil, true},
		{"array JSON Boolean", types.Array(types.JSON()), OpContains, json.Value("false"), nil, true},
		{"unary operand", types.JSON(), OpIsNull, json.Value("null"), nil, true},
		{"nil raw", types.JSON(), OpIs, nil, nil, true},
		{"empty raw", types.JSON(), OpIs, json.Value{}, nil, true},
		{"invalid raw", types.JSON(), OpIs, json.Value("1e"), nil, true},
		{"multiple documents", types.JSON(), OpIs, json.Value("1 2"), nil, true},
		{"non JSON number", types.JSON(), OpIs, json.Value("+100"), nil, true},
		{"UUID", types.UUID(), OpIs, json.Value(`"38D065AB-CA46-4812-A83C-A9712E09C153"`), "38d065ab-ca46-4812-a83c-a9712e09c153", false},
		{"IP", types.IP(), OpIs, json.Value(`"::ffff:192.168.1.1"`), "192.168.1.1", false},
		{"datetime offset", types.DateTime(), OpIs, json.Value(`"2026-10-24T12:00:00.123456789+02:00"`), time.Date(2026, 10, 24, 12, 0, 0, 123456789, time.FixedZone("", 7200)), false},
		{"date", types.Date(), OpIs, json.Value(`"2026-10-24"`), time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), false},
		{"time", types.Time(), OpIs, json.Value(`"12:03:04.123456789"`), time.Date(1970, 1, 1, 12, 3, 4, 123456789, time.UTC), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := DecodeFilterOperand(test.raw, test.typ, test.op)
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected valid operand, got %v", err)
				}
				return
			}
			if test.invalid {
				t.Fatalf("expected invalid operand, got %#v", got)
			}
			if expected, ok := test.want.(decimal.Decimal); ok {
				if actual, ok := got.(decimal.Decimal); !ok || !actual.Equal(expected) {
					t.Fatalf("expected %s, got %v", expected, got)
				}
				return
			}
			if expected, ok := test.want.(json.Value); ok && expected.IsNumber() {
				actual, ok := got.(json.Value)
				if !ok || !actual.IsNumber() {
					t.Fatalf("expected JSON Number, got %#v", got)
				}
				number, err := actual.Decimal(0, 0)
				if err != nil {
					t.Fatalf("expected exact numeric operand, got %v", err)
				}
				bound, _ := expected.Decimal(0, 0)
				if !number.Equal(bound) {
					t.Fatalf("expected %s, got %s", expected, actual)
				}
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expected %#v, got %#v", test.want, got)
			}
		})
	}
}

// TestWhereJSONSemanticEquality checks scalar equality without coercion.
func TestWhereJSONSemanticEquality(t *testing.T) {
	for _, test := range []struct {
		left, right string
		equal       bool
	}{
		{"100", "100.0", true}, {"100.0", "1e2", true}, {"0", "-0.0", true},
		{`"100"`, "100", false}, {`"a"`, `"\u0061"`, true},
		{"true", "true", true}, {"false", "false", true}, {"true", "false", false},
		{"false", "0", false}, {"false", `"false"`, false}, {"null", "null", true},
		{"null", `"null"`, false}, {"null", "false", false},
	} {
		t.Run(test.left+"/"+test.right, func(t *testing.T) {
			left := &WhereCondition{Values: []any{json.Value(test.left)}}
			right := &WhereCondition{Values: []any{json.Value(test.right)}}
			if got := left.Equal(right); got != test.equal {
				t.Fatalf("expected equal %t, got %t", test.equal, got)
			}
		})
	}
}

// TestWhereTypedJSONBRoundTrip checks PostgreSQL JSONB normalization of
// operands.
func TestWhereTypedJSONBRoundTrip(t *testing.T) {
	database, _ := replicationDatabase(t)
	schema := types.Object([]types.Property{
		{Name: "value", Type: types.JSON()},
		{Name: "account", Type: types.Int(64).Unsigned()},
	})
	var values []any
	for _, raw := range []json.Value{
		json.Value("0e20000000000000000000"), json.Value(strings.Repeat("9", 76)),
		json.Value("1e-37"), json.Value(`"100"`), json.Value(`""`),
		json.Value("true"), json.Value("false"), json.Value("null"),
	} {
		operand, err := DecodeFilterOperand(raw, types.JSON(), OpIsOneOf)
		if err != nil {
			t.Fatalf("expected representable operand %s, got %v", raw, err)
		}
		values = append(values, operand)
	}
	where := &Where{Operator: OpAnd, Rules: []WhereRule{
		&WhereCondition{Property: []string{"value"}, Operator: OpIsOneOf, Values: values},
		&WhereCondition{Property: []string{"account"}, Operator: OpIs, Values: []any{uint(math.MaxUint64)}},
	}}
	var stored []byte
	err := database.QueryRow(t.Context(), "SELECT $1::jsonb", string(MarshalWhere(where, schema))).Scan(&stored)
	if err != nil {
		t.Fatalf("expected representable JSONB filter, got %v", err)
	}
	loaded, err := unmarshalWhere(stored, schema)
	if err != nil {
		t.Fatalf("expected typed JSONB filter decoding, got %v", err)
	}
	if !where.Equal(loaded) {
		t.Fatalf("expected equivalent JSONB filter, got %s", stored)
	}
}

// TestWhereTypedPersistence preserves operand kinds and the persisted UTC
// policy.
func TestWhereTypedPersistence(t *testing.T) {
	schema := types.Object([]types.Property{
		{Name: "signed", Type: types.Int(64)},
		{Name: "unsigned", Type: types.Int(64).Unsigned()},
		{Name: "value", Type: types.JSON()},
		{Name: "when", Type: types.DateTime()},
		{Name: "ratio", Type: types.Float(64)},
		{Name: "amount", Type: types.Decimal(5, 2)},
	})
	where := &Where{Operator: OpAnd, Rules: []WhereRule{
		&WhereCondition{Property: []string{"signed"}, Operator: OpIs, Values: []any{int(math.MinInt64)}},
		&WhereCondition{Property: []string{"unsigned"}, Operator: OpIs, Values: []any{uint(math.MaxUint64)}},
		&WhereCondition{Property: []string{"value"}, Operator: OpIsOneOf, Values: []any{
			json.Value("1e2"), json.Value(`"100"`), json.Value("true"), json.Value("false"), json.Value("null"), json.Value(`""`),
		}},
		&WhereCondition{Property: []string{"when"}, Operator: OpIs, Values: []any{
			time.Date(2026, 10, 24, 12, 0, 0, 123456789, time.FixedZone("", 7200)),
		}},
		&WhereCondition{Property: []string{"ratio"}, Operator: OpIs, Values: []any{1e100}},
		&WhereCondition{Property: []string{"amount"}, Operator: OpIsLessThan, Values: []any{decimal.MustParse("1000.00")}},
	}}
	encoded := MarshalWhere(where, schema)
	for _, fragment := range []string{
		`"values":[-9223372036854775808]`, `"values":[18446744073709551615]`,
		`"values":[1e2,"100",true,false,null,""]`, `2026-10-24T10:00:00.123456789Z`,
	} {
		if !strings.Contains(string(encoded), fragment) {
			t.Fatalf("expected encoded fragment %s, got %s", fragment, encoded)
		}
	}
	loaded, err := unmarshalWhere(encoded, schema)
	if err != nil {
		t.Fatalf("expected typed persisted filter, got %v", err)
	}
	if got := loaded.Rules[0].(*WhereCondition).Values[0]; got != int(math.MinInt64) {
		t.Fatalf("expected exact signed minimum, got %v", got)
	}
	if got := loaded.Rules[1].(*WhereCondition).Values[0]; got != uint(math.MaxUint64) {
		t.Fatalf("expected exact unsigned maximum, got %v", got)
	}
	values := loaded.Rules[2].(*WhereCondition).Values
	for i, kind := range []json.Kind{json.Number, json.String, json.True, json.False, json.Null, json.String} {
		if got := values[i].(json.Value).Kind(); got != kind {
			t.Fatalf("expected kind %v at %d, got %v", kind, i, got)
		}
	}
	instant := loaded.Rules[3].(*WhereCondition).Values[0].(time.Time)
	if _, offset := instant.Zone(); offset != 0 || instant.Hour() != 10 {
		t.Fatalf("expected persisted UTC 10:00, got %v", instant)
	}
	again, err := unmarshalWhere(MarshalWhere(loaded, schema), schema)
	if err != nil {
		t.Fatalf("expected second persisted round trip, got %v", err)
	}
	if !loaded.Equal(again) {
		t.Fatal("expected equivalent persisted filters, got unequal values")
	}
}

// TestWhereTypedPersistenceValidation rejects invalid typed persisted operands.
func TestWhereTypedPersistenceValidation(t *testing.T) {
	schema := types.Object([]types.Property{
		{Name: "value", Type: types.JSON()},
		{Name: "account", Type: types.Int(64)},
	})
	for _, condition := range []string{
		`{"property":["value"],"operator":"IsNull","values":[null]}`,
		`{"property":["value"],"operator":"IsNull","values":[]}`,
		`{"property":["value"],"operator":"Is","values":[null]}`,
		`{"property":["value"],"operator":"IsBetween","values":[100,"100"]}`,
		`{"property":["value"],"operator":"IsOneOf","values":[{}]}`,
		`{"property":["account"],"operator":"Is","values":["9223372036854775807"]}`,
		`{"property":["account"],"operator":"Is","values":[100.0]}`,
		`{"property":["account"],"operator":"Is","values":[1e2]}`,
	} {
		t.Run(condition, func(t *testing.T) {
			_, err := unmarshalWhere([]byte(`{"operator":"And","rules":[`+condition+`]}`), schema)
			if err != nil {
				return
			}
			t.Fatal("expected invalid persisted operand, got nil error")
		})
	}
}
