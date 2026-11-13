// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouseexpr

import (
	"fmt"
	"math"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// SQLFloatCases returns native Float cases with explicit SQL outcomes.
// PostgreSQL and Snowflake order NaN above all other values, unlike Go.
// Operands are finite; non-finite values occur only in stored data.
func SQLFloatCases() []ValueCase {
	var cases []ValueCase
	for _, bits := range []int{32, 64} {
		typ := types.Float(bits)
		maximum := math.MaxFloat64
		if bits == 32 {
			maximum = math.MaxFloat32
		}
		for _, c := range []ValueCase{
			{"zero", typ, warehouses.OpIs, []any{0.0}, math.Copysign(0, -1), true},
			{"negative zero", typ, warehouses.OpIsNot, []any{0.0}, math.Copysign(0, -1), false},
			{"finite order", typ, warehouses.OpIsGreaterThan, []any{1.0}, 1.25, true},
			{"finite interval", typ, warehouses.OpIsBetween, []any{1.0, 1.25}, 1.25, true},
			{"positive infinity", typ, warehouses.OpIsGreaterThan, []any{maximum}, math.Inf(1), true},
			{"negative infinity", typ, warehouses.OpIsLessThan, []any{-maximum}, math.Inf(-1), true},
			{"infinity interval", typ, warehouses.OpIsBetween, []any{-maximum, maximum}, math.Inf(1), false},
			{"NaN equal", typ, warehouses.OpIs, []any{0.0}, math.NaN(), false},
			{"NaN not equal", typ, warehouses.OpIsNot, []any{0.0}, math.NaN(), true},
			{"NaN less", typ, warehouses.OpIsLessThan, []any{0.0}, math.NaN(), false},
			{"NaN less equal", typ, warehouses.OpIsLessThanOrEqualTo, []any{0.0}, math.NaN(), false},
			{"NaN greater", typ, warehouses.OpIsGreaterThan, []any{maximum}, math.NaN(), true},
			{"NaN greater equal", typ, warehouses.OpIsGreaterThanOrEqualTo, []any{0.0}, math.NaN(), true},
			{"NaN between", typ, warehouses.OpIsBetween, []any{-maximum, maximum}, math.NaN(), false},
			{"NaN not between", typ, warehouses.OpIsNotBetween, []any{-maximum, maximum}, math.NaN(), true},
		} {
			c.Name = fmt.Sprintf("float%d/%s", bits, c.Name)
			cases = append(cases, c)
		}
	}
	cases = append(cases,
		ValueCase{"float64/adjacent", types.Float(64), warehouses.OpIsGreaterThan,
			[]any{100.0}, math.Nextafter(100, math.Inf(1)), true},
		ValueCase{"float64/adjacent interval", types.Float(64), warehouses.OpIsBetween,
			[]any{100.0, 100.0}, math.Nextafter(100, math.Inf(1)), false},
		ValueCase{"array NaN", types.Array(types.Float(64)), warehouses.OpContains,
			[]any{0.0}, []any{math.NaN()}, false},
		ValueCase{"array NaN negative", types.Array(types.Float(64)), warehouses.OpDoesNotContain,
			[]any{0.0}, []any{math.NaN()}, true},
	)

	return cases
}
