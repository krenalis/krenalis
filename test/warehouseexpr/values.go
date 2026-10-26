// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouseexpr

import (
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// ValueCase specifies a typed value predicate and its result on a present value.
// Every predicate returns SQL NULL on absence, or false in the Go evaluator.
type ValueCase struct {
	Name     string
	Type     types.Type
	Operator warehouses.Operator
	Values   []any
	Value    any
	Matches  bool
}

// ValueCases returns ordinary value predicates, including all 20 operators.
func ValueCases() []ValueCase {
	day := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	instant := day.Add(10 * time.Hour)
	offset := instant.In(time.FixedZone("", 7200))
	clock := time.Date(1970, 1, 1, 10, 30, 0, 123456000, time.UTC)
	cases := []ValueCase{
		{"is", types.Int(32), warehouses.OpIs, []any{100}, 100, true},
		{"is not", types.Int(32), warehouses.OpIsNot, []any{100}, 101, true},
		{"less", types.Int(32), warehouses.OpIsLessThan, []any{100}, 99, true},
		{"less or equal", types.Int(32), warehouses.OpIsLessThanOrEqualTo, []any{100}, 100, true},
		{"greater", types.Int(32), warehouses.OpIsGreaterThan, []any{100}, 101, true},
		{"greater or equal", types.Int(32), warehouses.OpIsGreaterThanOrEqualTo, []any{100}, 100, true},
		{"between", types.Int(32), warehouses.OpIsBetween, []any{100, 200}, 100, true},
		{"not between", types.Int(32), warehouses.OpIsNotBetween, []any{100, 200}, 99, true},
		{"contains", types.String(), warehouses.OpContains, []any{"foo"}, "foobar", true},
		{"does not contain", types.String(), warehouses.OpDoesNotContain, []any{"foo"}, "bar", true},
		{"one of", types.Int(32), warehouses.OpIsOneOf, []any{100, 200}, 100, true},
		{"not one of", types.Int(32), warehouses.OpIsNotOneOf, []any{100, 200}, 101, true},
		{"prefix", types.String(), warehouses.OpStartsWith, []any{"foo"}, "foobar", true},
		{"suffix", types.String(), warehouses.OpEndsWith, []any{"foo"}, "barfoo", true},
		{"before", types.Date(), warehouses.OpIsBefore, []any{day}, day.AddDate(0, 0, -1), true},
		{"on or before", types.Date(), warehouses.OpIsOnOrBefore, []any{day}, day, true},
		{"after", types.Date(), warehouses.OpIsAfter, []any{day}, day.AddDate(0, 0, 1), true},
		{"on or after", types.Date(), warehouses.OpIsOnOrAfter, []any{day}, day, true},
		{"true", types.Boolean(), warehouses.OpIsTrue, nil, true, true},
		{"false", types.Boolean(), warehouses.OpIsFalse, nil, false, true},
		{"datetime offset equality", types.DateTime(), warehouses.OpIs, []any{offset}, instant, true},
		{"datetime offset order", types.DateTime(), warehouses.OpIsAfter, []any{offset}, instant.Add(time.Minute), true},
		{"datetime offset interval", types.DateTime(), warehouses.OpIsBetween,
			[]any{offset, offset.Add(time.Hour)}, instant.Add(time.Minute), true},
		{"temporal reversed lower", types.Date(), warehouses.OpIsBetween, []any{day, day.AddDate(0, 0, -1)}, day, false},
		{"temporal reversed upper", types.Date(), warehouses.OpIsBetween, []any{day.AddDate(0, 0, 1), day}, day, false},
		{"temporal reversed negative", types.Date(), warehouses.OpIsNotBetween, []any{day, day.AddDate(0, 0, -1)}, day, true},
		{"temporal singleton", types.Date(), warehouses.OpIsBetween, []any{day, day}, day, true},
		{"temporal singleton negative", types.Date(), warehouses.OpIsNotBetween, []any{day, day}, day, false},
		{"array string", types.Array(types.String()), warehouses.OpContains, []any{"foo"}, []any{"foo", "bar"}, true},
		{"array int", types.Array(types.Int(32)), warehouses.OpContains, []any{100}, []any{100, 200}, true},
		{"array int64", types.Array(types.Int(64)), warehouses.OpContains,
			[]any{9223372036854775807}, []any{9223372036854775807}, true},
		{"array uint64", types.Array(types.Int(64).Unsigned()), warehouses.OpContains,
			[]any{uint(18446744073709551615)}, []any{uint(18446744073709551615)}, true},
		{"array float", types.Array(types.Float(64)), warehouses.OpContains, []any{1e100}, []any{1e100}, true},
		{"array float32", types.Array(types.Float(32)), warehouses.OpContains,
			[]any{float64(float32(1.2345678))}, []any{float64(float32(1.2345678))}, true},
		{"array decimal", types.Array(types.Decimal(18, 4)), warehouses.OpContains,
			[]any{decimal.MustParse("1.25")}, []any{decimal.MustParse("1.25")}, true},
		{"array datetime", types.Array(types.DateTime()), warehouses.OpContains, []any{offset}, []any{instant}, true},
		{"array datetime fractional", types.Array(types.DateTime()), warehouses.OpContains,
			[]any{offset.Add(time.Microsecond)}, []any{instant.Add(time.Microsecond)}, true},
		{"array date", types.Array(types.Date()), warehouses.OpContains, []any{day}, []any{day}, true},
		{"array time", types.Array(types.Time()), warehouses.OpContains, []any{clock}, []any{clock}, true},
		{"array year", types.Array(types.Year()), warehouses.OpContains, []any{2026}, []any{2026}, true},
		{"empty array", types.Array(types.String()), warehouses.OpContains, []any{"foo"}, []any{}, false},
	}
	for _, c := range cases {
		if c.Operator == warehouses.OpContains && c.Type.IsArray() {
			c.Name = "negative/" + c.Name
			c.Operator = warehouses.OpDoesNotContain
			c.Matches = !c.Matches
			cases = append(cases, c)
		}
	}
	return cases
}
