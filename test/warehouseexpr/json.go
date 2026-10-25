// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package warehouseexpr provides shared JSON fixtures for warehouse tests.
package warehouseexpr

import (
	"fmt"
	"slices"
	"strings"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// JSONCase specifies a predicate and the indices of the matching JSON fixtures.
type JSONCase struct {
	Name     string
	Operator warehouses.Operator
	Values   []any
	Matches  []int
}

// JSONCases returns JSON fixtures and manually specified positive matches.
// Negative matches are complements, as defined by the filter evaluator.
// A nil fixture represents an absent property, distinct from JSON null.
func JSONCases() ([]json.Value, []JSONCase) {
	values := []json.Value{
		nil, json.Value("null"), json.Value("true"), json.Value("false"),
		json.Value("\"\""), json.Value("\"foo\""), json.Value("\"foobar\""), json.Value("\"barfoo\""),
		json.Value("\"Foo\""), json.Value("\"5\""), json.Value("\"5.0\""),
		json.Value("0"), json.Value("-0.0"), json.Value("5"), json.Value("5.00"), json.Value("-5"),
		json.Value("50"), json.Value("0.5"), json.Value("1e3"), json.Value("9007199254740993"),
		json.Value("{}"), json.Value("{\"x\":null}"), json.Value("[]"), json.Value("[5]"),
		json.Value("[\"5\"]"), json.Value("[null,true,false,5,\"foo\",{},[]]"), json.Value("[[5]]"),
		json.Value("\"true\""), json.Value("\"null\""), json.Value("\"é\""), json.Value("\"😀\""),
		json.Value("9007199254740992"), json.Value("1000"),
		json.Value("0.1234567890123456789012345678901234567"),
		json.Value("0.1234567890123456789012345678901234568"),
		json.Value("-9007199254740993"),
		json.Value(strings.Repeat("9", 37)), json.Value(strings.Repeat("9", 38)),
		json.Value("12.345678901234567890123456789012345678"),
		json.Value("-12.345678901234567890123456789012345678"),
		json.Value("0." + strings.Repeat("0", 36) + "1"),
	}
	positiveNeighbor := "9007199254740993." + strings.Repeat("0", 36) + "1"
	negativeNeighbor := "-" + positiveNeighbor
	positiveFractionNeighbor := "12.3456789012345678901234567890123456781"
	negativeFractionNeighbor := "-" + positiveFractionNeighbor
	cases := []JSONCase{
		{"is number", warehouses.OpIs, []any{json.Value("5")}, []int{13, 14}},
		{"is decimal notation", warehouses.OpIs, []any{json.Value("5.0")}, []int{13, 14}},
		{"is exponent notation", warehouses.OpIs, []any{json.Value("5e0")}, []int{13, 14}},
		{"is signed text", warehouses.OpIs, []any{json.Value(`"+5"`)}, []int{}},
		{"is string", warehouses.OpIs, []any{json.Value("\"foo\"")}, []int{5}},
		{"is empty string", warehouses.OpIs, []any{json.Value("\"\"")}, []int{4}},
		{"is true text", warehouses.OpIs, []any{json.Value("\"true\"")}, []int{27}},
		{"is null text", warehouses.OpIs, []any{json.Value("\"null\"")}, []int{28}},
		{"large integer", warehouses.OpIs, []any{json.Value("9007199254740993")}, []int{19}},
		{"large integer neighbor", warehouses.OpIs, []any{json.Value("9007199254740992")}, []int{31}},
		{"large integer ordering", warehouses.OpIsGreaterThan, []any{json.Value("9007199254740992")},
			[]int{19, 36, 37}},
		{"zero", warehouses.OpIs, []any{json.Value("0")}, []int{11, 12}},
		{"negative zero", warehouses.OpIs, []any{json.Value("-0.0")}, []int{11, 12}},
		{"thousand", warehouses.OpIs, []any{json.Value("1000")}, []int{18, 32}},
		{"thousand exponent", warehouses.OpIs, []any{json.Value("1e3")}, []int{18, 32}},
		{"precise decimal", warehouses.OpIs, []any{json.Value("0.1234567890123456789012345678901234567")}, []int{33}},
		{"precise decimal neighbor", warehouses.OpIs,
			[]any{json.Value("0.1234567890123456789012345678901234568")}, []int{34}},
		{"precise decimal ordering", warehouses.OpIsLessThan, []any{json.Value("0.1234567890123456789012345678901234568")},
			[]int{11, 12, 15, 33, 35, 39, 40}},
		{"less than", warehouses.OpIsLessThan, []any{json.Value("5")}, []int{11, 12, 15, 17, 33, 34, 35, 39, 40}},
		{"less than or equal", warehouses.OpIsLessThanOrEqualTo, []any{json.Value("5")},
			[]int{11, 12, 13, 14, 15, 17, 33, 34, 35, 39, 40}},
		{"greater than", warehouses.OpIsGreaterThan, []any{json.Value("5")},
			[]int{16, 18, 19, 31, 32, 36, 37, 38}},
		{"greater than or equal", warehouses.OpIsGreaterThanOrEqualTo, []any{json.Value("5")},
			[]int{13, 14, 16, 18, 19, 31, 32, 36, 37, 38}},
		{"between inclusive", warehouses.OpIsBetween, []any{json.Value("0.5"), json.Value("5")}, []int{13, 14, 17}},
		{"between reversed", warehouses.OpIsBetween, []any{json.Value("5"), json.Value("0.5")}, []int{}},
		{"between strings", warehouses.OpIsBetween, []any{json.Value("\"5\""), json.Value("\"foo\"")}, []int{5, 7, 8, 9, 10}},
		{"contains number", warehouses.OpContains, []any{json.Value("5")}, []int{23, 25}},
		{"contains text", warehouses.OpContains, []any{json.Value("\"foo\"")}, []int{5, 6, 7, 25}},
		{"contains boolean text", warehouses.OpContains, []any{json.Value("\"true\"")}, []int{27}},
		{"contains empty text", warehouses.OpContains, []any{json.Value(`""`)}, []int{4, 5, 6, 7, 8, 9, 10, 27, 28, 29, 30}},
		{"one of", warehouses.OpIsOneOf, []any{json.Value("5"), json.Value(`"foo"`), json.Value(`"null"`)}, []int{5, 13, 14, 28}},
		{"one of duplicate", warehouses.OpIsOneOf, []any{json.Value("5"), json.Value("5e0")}, []int{13, 14}},
		{"starts with", warehouses.OpStartsWith, []any{json.Value("\"foo\"")}, []int{5, 6}},
		{"ends with", warehouses.OpEndsWith, []any{json.Value("\"foo\"")}, []int{5, 7}},
		{"is true", warehouses.OpIsTrue, nil, []int{2}},
		{"is false", warehouses.OpIsFalse, nil, []int{3}},
		{"empty", warehouses.OpIsEmpty, nil, []int{0, 1, 4, 20, 22}},
		{"null", warehouses.OpIsNull, nil, []int{1}},
		{"exists", warehouses.OpExists, nil,
			[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24,
				25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}},
		{"37 digit integer", warehouses.OpIs, []any{json.Value(strings.Repeat("9", 37))},
			[]int{36}},
		{"38 digit integer", warehouses.OpIs, []any{json.Value(strings.Repeat("9", 38))},
			[]int{37}},
		{"39 digit equality", warehouses.OpIs, []any{json.Value(positiveFractionNeighbor)},
			[]int{}},
		{"negative 39 digit equality", warehouses.OpIs, []any{json.Value(negativeFractionNeighbor)},
			[]int{}},
		{"53 digit equality", warehouses.OpIs, []any{json.Value(positiveNeighbor)},
			[]int{}},
		{"negative 53 digit equality", warehouses.OpIs, []any{json.Value(negativeNeighbor)},
			[]int{}},
		{"positive integer neighbor/less than", warehouses.OpIsLessThan, []any{json.Value(positiveNeighbor)},
			[]int{11, 12, 13, 14, 15, 16, 17, 18, 19, 31, 32, 33, 34, 35, 38, 39, 40}},
		{"positive integer neighbor/less than or equal", warehouses.OpIsLessThanOrEqualTo,
			[]any{json.Value(positiveNeighbor)},
			[]int{11, 12, 13, 14, 15, 16, 17, 18, 19, 31, 32, 33, 34, 35, 38, 39, 40}},
		{"positive integer neighbor/greater than", warehouses.OpIsGreaterThan, []any{json.Value(positiveNeighbor)},
			[]int{36, 37}},
		{"positive integer neighbor/greater than or equal", warehouses.OpIsGreaterThanOrEqualTo,
			[]any{json.Value(positiveNeighbor)}, []int{36, 37}},
		{"negative integer neighbor/less than", warehouses.OpIsLessThan, []any{json.Value(negativeNeighbor)},
			[]int{}},
		{"negative integer neighbor/less than or equal", warehouses.OpIsLessThanOrEqualTo,
			[]any{json.Value(negativeNeighbor)},
			[]int{}},
		{"negative integer neighbor/greater than", warehouses.OpIsGreaterThan, []any{json.Value(negativeNeighbor)},
			[]int{11, 12, 13, 14, 15, 16, 17, 18, 19, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}},
		{"negative integer neighbor/greater than or equal", warehouses.OpIsGreaterThanOrEqualTo,
			[]any{json.Value(negativeNeighbor)},
			[]int{11, 12, 13, 14, 15, 16, 17, 18, 19, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}},
		{"positive fraction neighbor/less than", warehouses.OpIsLessThan, []any{json.Value(positiveFractionNeighbor)},
			[]int{11, 12, 13, 14, 15, 17, 33, 34, 35, 38, 39, 40}},
		{"positive fraction neighbor/less than or equal", warehouses.OpIsLessThanOrEqualTo,
			[]any{json.Value(positiveFractionNeighbor)}, []int{11, 12, 13, 14, 15, 17, 33, 34, 35, 38, 39, 40}},
		{"positive fraction neighbor/greater than", warehouses.OpIsGreaterThan, []any{json.Value(positiveFractionNeighbor)},
			[]int{16, 18, 19, 31, 32, 36, 37}},
		{"positive fraction neighbor/greater than or equal", warehouses.OpIsGreaterThanOrEqualTo,
			[]any{json.Value(positiveFractionNeighbor)}, []int{16, 18, 19, 31, 32, 36, 37}},
		{"negative fraction neighbor/less than", warehouses.OpIsLessThan, []any{json.Value(negativeFractionNeighbor)},
			[]int{35}},
		{"negative fraction neighbor/less than or equal", warehouses.OpIsLessThanOrEqualTo,
			[]any{json.Value(negativeFractionNeighbor)}, []int{35}},
		{"negative fraction neighbor/greater than", warehouses.OpIsGreaterThan, []any{json.Value(negativeFractionNeighbor)},
			[]int{11, 12, 13, 14, 15, 16, 17, 18, 19, 31, 32, 33, 34, 36, 37, 38, 39, 40}},
		{"negative fraction neighbor/greater than or equal", warehouses.OpIsGreaterThanOrEqualTo,
			[]any{json.Value(negativeFractionNeighbor)},
			[]int{11, 12, 13, 14, 15, 16, 17, 18, 19, 31, 32, 33, 34, 36, 37, 38, 39, 40}},
		{"between precise integer neighbors", warehouses.OpIsBetween, []any{
			json.Value(positiveNeighbor), json.Value(strings.Repeat("9", 38)),
		}, []int{36, 37}},
		{"between precise negative neighbors", warehouses.OpIsBetween, []any{
			json.Value(negativeNeighbor), json.Value("-9007199254740993"),
		}, []int{35}},
		{"membership with inexact and exact operands", warehouses.OpIsOneOf, []any{
			json.Value(positiveNeighbor), json.Value("9007199254740993"), json.Value("\"foo\""),
		}, []int{5, 19}},
		{"membership with only inexact operands", warehouses.OpIsOneOf, []any{
			json.Value(positiveNeighbor), json.Value(negativeNeighbor),
			json.Value(positiveFractionNeighbor), json.Value(negativeFractionNeighbor),
		}, []int{}},
		{"76 digit equality", warehouses.OpIs, []any{json.Value(strings.Repeat("9", 76))},
			[]int{}},
		{"maximum scale equality", warehouses.OpIs, []any{json.Value("1e-37")},
			[]int{40}},
		{"is numeric string", warehouses.OpIs, []any{json.Value("\"5\"")}, []int{9}},
		{"contains numeric string", warehouses.OpContains, []any{json.Value("\"5\"")}, []int{9, 10, 24}},
		{"membership mixed kinds", warehouses.OpIsOneOf, []any{
			json.Value("5"), json.Value("\"5\""), json.Value("true"), json.Value("false"), json.Value("null"),
		}, []int{1, 2, 3, 9, 13, 14}},
		{"membership false", warehouses.OpIsOneOf, []any{json.Value("false")}, []int{3}},
		{"membership null", warehouses.OpIsOneOf, []any{json.Value("null")}, []int{1}},
		{"membership zero", warehouses.OpIsOneOf, []any{json.Value("0")}, []int{11, 12}},
	}
	negative := map[warehouses.Operator]warehouses.Operator{
		warehouses.OpIs: warehouses.OpIsNot, warehouses.OpIsBetween: warehouses.OpIsNotBetween,
		warehouses.OpContains: warehouses.OpDoesNotContain, warehouses.OpIsOneOf: warehouses.OpIsNotOneOf,
		warehouses.OpIsEmpty: warehouses.OpIsNotEmpty, warehouses.OpIsNull: warehouses.OpIsNotNull,
		warehouses.OpExists: warehouses.OpDoesNotExist,
	}
	for _, c := range cases {
		op, ok := negative[c.Operator]
		if !ok {
			continue
		}
		var matches []int
		for i := range values {
			if !slices.Contains(c.Matches, i) {
				matches = append(matches, i)
			}
		}
		cases = append(cases, JSONCase{Name: "negative/" + c.Name, Operator: op, Values: c.Values, Matches: matches})
	}
	return values, cases
}

// Number encodes a decimal operand without changing its value or checking
// filter limits, so fixtures can also exercise invalid numeric thresholds.
func Number(value decimal.Decimal) json.Value {
	encoded, err := value.MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("test/warehouseexpr: cannot encode decimal operand: %w", err))
	}
	return json.Value(encoded)
}
