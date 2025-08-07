//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package filters

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
)

// Test_Applies exercises the Applies function with various operators and
// values.
func Test_Applies(t *testing.T) {

	var n670 = decimal.MustInt(670)
	var v670 = state.JSONConditionValue{String: "670", Number: &n670}

	var n812 = decimal.MustInt(812)
	var v812 = state.JSONConditionValue{String: "812", Number: &n812}

	var vFoo = state.JSONConditionValue{String: "foo"}
	var vBoo = state.JSONConditionValue{String: "boo"}

	tests := []struct {
		op        state.WhereOperator
		v         any
		v0        any
		v1        any
		notExists bool
		expected  bool
	}{
		// OpIs.
		{op: state.OpIs, v0: 5, expected: false, notExists: true},
		{op: state.OpIs, v: 5, v0: 5, expected: true},
		{op: state.OpIs, v: 5, v0: 7, expected: false},
		{op: state.OpIs, v: uint(12), v0: uint(12), expected: true},
		{op: state.OpIs, v: 12.3829401183652, v0: 12.3829401183652, expected: true},
		{op: state.OpIs, v: float64(float32(-16.09275)), v0: float64(float32(-16.09275)), expected: true},
		{op: state.OpIs, v: float64(float32(-16.09277)), v0: float64(float32(-16.09275)), expected: false},
		{op: state.OpIs, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("947126405.18361204926705329"), expected: true},
		{op: state.OpIs, v: decimal.MustParse("947126405.18361204926705328"), v0: decimal.MustParse("947126405.18361204926705329"), expected: false},
		{op: state.OpIs, v: "foo", v0: "boo", expected: false},
		{op: state.OpIs, v: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: true},
		{op: state.OpIs, v: time.Date(2024, 9, 11, 9, 58, 15, 704152446, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},
		{op: state.OpIs, v: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC), v0: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC), expected: true},
		{op: state.OpIs, v: time.Date(1970, 1, 1, 9, 58, 15, 704152446, time.UTC), v0: time.Date(1970, 1, 1, 9, 58, 15, 704152446, time.UTC), expected: true},
		{op: state.OpIs, v: nil, v0: 5, expected: false},
		{op: state.OpIs, v: json.Value("670"), v0: v670, expected: true},
		{op: state.OpIs, v: json.Value("670.0"), v0: v670, expected: true},
		{op: state.OpIs, v: json.Value(`"670"`), v0: v670, expected: true},
		{op: state.OpIs, v: json.Value("true"), v0: v670, expected: false},
		{op: state.OpIs, v: json.Value(`"foo"`), v0: v670, expected: false},
		{op: state.OpIs, v: json.Value("920"), v0: v670, expected: false},
		{op: state.OpIs, v: json.Value("null"), v0: v670, expected: false},

		// OpIsNot.
		{op: state.OpIsNot, v0: 5, expected: true, notExists: true},
		{op: state.OpIsNot, v: 5, v0: 5, expected: false},
		{op: state.OpIsNot, v: 5, v0: 7, expected: true},
		{op: state.OpIsNot, v: uint(12), v0: uint(12), expected: false},
		{op: state.OpIsNot, v: 12.3829401183652, v0: 12.3829401183652, expected: false},
		{op: state.OpIsNot, v: float64(float32(-16.09275)), v0: float64(float32(-16.09275)), expected: false},
		{op: state.OpIsNot, v: float64(float32(-16.09277)), v0: float64(float32(-16.09275)), expected: true},
		{op: state.OpIsNot, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("947126405.18361204926705329"), expected: false},
		{op: state.OpIsNot, v: decimal.MustParse("947126405.18361204926705328"), v0: decimal.MustParse("947126405.18361204926705329"), expected: true},
		{op: state.OpIsNot, v: "foo", v0: "boo", expected: true},
		{op: state.OpIsNot, v: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},
		{op: state.OpIsNot, v: time.Date(2024, 9, 11, 9, 58, 15, 704152446, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: true},
		{op: state.OpIsNot, v: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC), v0: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC), expected: false},
		{op: state.OpIsNot, v: time.Date(1970, 1, 1, 9, 58, 15, 704152446, time.UTC), v0: time.Date(1970, 1, 1, 9, 58, 15, 704152446, time.UTC), expected: false},
		{op: state.OpIsNot, v: nil, v0: 5, expected: true},
		{op: state.OpIsNot, v: json.Value("670"), v0: v670, expected: false},
		{op: state.OpIsNot, v: json.Value("670.0"), v0: v670, expected: false},
		{op: state.OpIsNot, v: json.Value(`"670"`), v0: v670, expected: false},
		{op: state.OpIsNot, v: json.Value("true"), v0: v670, expected: true},
		{op: state.OpIsNot, v: json.Value(`"foo"`), v0: v670, expected: true},
		{op: state.OpIsNot, v: json.Value("920"), v0: v670, expected: true},
		{op: state.OpIsNot, v: json.Value("null"), v0: v670, expected: true},

		// OpIsLessThan.
		{op: state.OpIsLessThan, v0: 359, expected: false, notExists: true},
		{op: state.OpIsLessThan, v: 201, v0: 359, expected: true},
		{op: state.OpIsLessThan, v: 10, v0: 10, expected: false},
		{op: state.OpIsLessThan, v: -89, v0: -302, expected: false},
		{op: state.OpIsLessThan, v: uint(5), v0: uint(7), expected: true},
		{op: state.OpIsLessThan, v: uint(93), v0: uint(12), expected: false},
		{op: state.OpIsLessThan, v: 1.5829371949264, v0: 1.5829371949265, expected: true},
		{op: state.OpIsLessThan, v: 1.5829371949264, v0: 1.5829371949264, expected: false},
		{op: state.OpIsLessThan, v: float64(float32(7.983)), v0: float64(float32(7.984)), expected: true},
		{op: state.OpIsLessThan, v: float64(float32(7.984)), v0: float64(float32(7.983)), expected: false},
		{op: state.OpIsLessThan, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("1204471285.038153"), expected: true},
		{op: state.OpIsLessThan, v: nil, v0: 5, expected: false},
		{op: state.OpIsLessThan, v: json.Value("315"), v0: v670, expected: true},
		{op: state.OpIsLessThan, v: json.Value("315.0"), v0: v670, expected: true},
		{op: state.OpIsLessThan, v: json.Value(`"315"`), v0: v670, expected: true},
		{op: state.OpIsLessThan, v: json.Value("true"), v0: v670, expected: false},
		{op: state.OpIsLessThan, v: json.Value(`"foo"`), v0: v670, expected: false},
		{op: state.OpIsLessThan, v: json.Value("920"), v0: v670, expected: false},
		{op: state.OpIsLessThan, v: json.Value("null"), v0: v670, expected: false},

		// OpIsLessThanOrEqualTo.
		{op: state.OpIsLessThanOrEqualTo, v0: 359, expected: false, notExists: true},
		{op: state.OpIsLessThanOrEqualTo, v: 359, v0: 359, expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: 8, v0: 10, expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: -89, v0: -904, expected: false},
		{op: state.OpIsLessThanOrEqualTo, v: uint(5), v0: uint(5), expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: uint(93), v0: uint(12), expected: false},
		{op: state.OpIsLessThanOrEqualTo, v: 1.5829371949264, v0: 1.5829371949265, expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: 1.5829371949264, v0: 1.5829371949264, expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: 1.5829371949264, v0: 1.5829371949263, expected: false},
		{op: state.OpIsLessThanOrEqualTo, v: float64(float32(7.983)), v0: float64(float32(7.984)), expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: float64(float32(7.983)), v0: float64(float32(7.983)), expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: float64(float32(7.984)), v0: float64(float32(7.983)), expected: false},
		{op: state.OpIsLessThanOrEqualTo, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("1204471285.038153"), expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("947126405.18361204926705329"), expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: decimal.MustParse("1204471285.038153"), v0: decimal.MustParse("947126405.18361204926705329"), expected: false},
		{op: state.OpIsLessThanOrEqualTo, v: nil, v0: 5, expected: false},
		{op: state.OpIsLessThanOrEqualTo, v: json.Value("315"), v0: v670, expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: json.Value("315.0"), v0: v670, expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: json.Value(`"315"`), v0: v670, expected: true},
		{op: state.OpIsLessThanOrEqualTo, v: json.Value(`"671"`), v0: v670, expected: false},
		{op: state.OpIsLessThanOrEqualTo, v: json.Value("true"), v0: v670, expected: false},
		{op: state.OpIsLessThanOrEqualTo, v: json.Value(`"foo"`), v0: v670, expected: false},
		{op: state.OpIsLessThanOrEqualTo, v: json.Value("920"), v0: v670, expected: false},
		{op: state.OpIsLessThanOrEqualTo, v: json.Value("null"), v0: v670, expected: false},

		// OpIsGreaterThan.
		{op: state.OpIsGreaterThan, v0: 359, expected: false, notExists: true},
		{op: state.OpIsGreaterThan, v: 360, v0: 359, expected: true},
		{op: state.OpIsGreaterThan, v: 12, v0: 10, expected: true},
		{op: state.OpIsGreaterThan, v: -89, v0: -904, expected: true},
		{op: state.OpIsGreaterThan, v: uint(6), v0: uint(5), expected: true},
		{op: state.OpIsGreaterThan, v: uint(12), v0: uint(12), expected: false},
		{op: state.OpIsGreaterThan, v: 1.5829371949266, v0: 1.5829371949265, expected: true},
		{op: state.OpIsGreaterThan, v: 1.5829371949264, v0: 1.5829371949264, expected: false},
		{op: state.OpIsGreaterThan, v: 1.5829371949264, v0: 1.5829371949265, expected: false},
		{op: state.OpIsGreaterThan, v: float64(float32(7.984)), v0: float64(float32(7.983)), expected: true},
		{op: state.OpIsGreaterThan, v: float64(float32(7.983)), v0: float64(float32(7.983)), expected: false},
		{op: state.OpIsGreaterThan, v: float64(float32(7.982)), v0: float64(float32(7.983)), expected: false},
		{op: state.OpIsGreaterThan, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("1204471285.038153"), expected: false},
		{op: state.OpIsGreaterThan, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("947126405.18361204926705329"), expected: false},
		{op: state.OpIsGreaterThan, v: decimal.MustParse("1204471285.038153"), v0: decimal.MustParse("947126405.18361204926705329"), expected: true},
		{op: state.OpIsGreaterThan, v: nil, v0: 5, expected: false},
		{op: state.OpIsGreaterThan, v: json.Value("315.0"), v0: v670, expected: false},
		{op: state.OpIsGreaterThan, v: json.Value("810"), v0: v670, expected: true},
		{op: state.OpIsGreaterThan, v: json.Value("810.0"), v0: v670, expected: true},
		{op: state.OpIsGreaterThan, v: json.Value(`"810"`), v0: v670, expected: true},
		{op: state.OpIsGreaterThan, v: json.Value(`"670"`), v0: v670, expected: false},
		{op: state.OpIsGreaterThan, v: json.Value("true"), v0: v670, expected: false},
		{op: state.OpIsGreaterThan, v: json.Value(`"foo"`), v0: v670, expected: true},
		{op: state.OpIsGreaterThan, v: json.Value(`""`), v0: v670, expected: false},
		{op: state.OpIsGreaterThan, v: json.Value("100"), v0: v670, expected: false},
		{op: state.OpIsGreaterThan, v: json.Value("null"), v0: v670, expected: false},

		// OpIsGreaterThanOrEqualTo.
		{op: state.OpIsGreaterThanOrEqualTo, v0: 359, expected: false, notExists: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: 360, v0: 359, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: 359, v0: 359, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: 12, v0: 10, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: -89, v0: -904, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: uint(6), v0: uint(5), expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: uint(6), v0: uint(6), expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: uint(11), v0: uint(12), expected: false},
		{op: state.OpIsGreaterThanOrEqualTo, v: 1.5829371949266, v0: 1.5829371949265, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: 1.5829371949264, v0: 1.5829371949264, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: 1.5829371949264, v0: 1.5829371949265, expected: false},
		{op: state.OpIsGreaterThanOrEqualTo, v: float64(float32(7.984)), v0: float64(float32(7.983)), expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: float64(float32(7.983)), v0: float64(float32(7.983)), expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: float64(float32(7.982)), v0: float64(float32(7.983)), expected: false},
		{op: state.OpIsGreaterThanOrEqualTo, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("1204471285.038153"), expected: false},
		{op: state.OpIsGreaterThanOrEqualTo, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("947126405.18361204926705329"), expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: decimal.MustParse("1204471285.038153"), v0: decimal.MustParse("947126405.18361204926705329"), expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: nil, v0: 5, expected: false},
		{op: state.OpIsGreaterThanOrEqualTo, v: json.Value("315.0"), v0: v670, expected: false},
		{op: state.OpIsGreaterThanOrEqualTo, v: json.Value("810"), v0: v670, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: json.Value("810.0"), v0: v670, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: json.Value(`"810"`), v0: v670, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: json.Value(`"670"`), v0: v670, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: json.Value("true"), v0: v670, expected: false},
		{op: state.OpIsGreaterThanOrEqualTo, v: json.Value(`"foo"`), v0: v670, expected: true},
		{op: state.OpIsGreaterThanOrEqualTo, v: json.Value(`""`), v0: v670, expected: false},
		{op: state.OpIsGreaterThanOrEqualTo, v: json.Value("100"), v0: v670, expected: false},
		{op: state.OpIsGreaterThanOrEqualTo, v: json.Value("null"), v0: v670, expected: false},

		// OpIsBetween.
		{op: state.OpIsBetween, v0: 350, v1: 412, expected: false, notExists: true},
		{op: state.OpIsBetween, v: 100, v0: 359, v1: 412, expected: false},
		{op: state.OpIsBetween, v: 359, v0: 359, v1: 412, expected: true},
		{op: state.OpIsBetween, v: 405, v0: 359, v1: 412, expected: true},
		{op: state.OpIsBetween, v: 412, v0: 359, v1: 412, expected: true},
		{op: state.OpIsBetween, v: 500, v0: 359, v1: 412, expected: false},
		{op: state.OpIsBetween, v: uint(3), v0: uint(5), v1: uint(10), expected: false},
		{op: state.OpIsBetween, v: uint(6), v0: uint(5), v1: uint(10), expected: true},
		{op: state.OpIsBetween, v: uint(5), v0: uint(5), v1: uint(10), expected: true},
		{op: state.OpIsBetween, v: uint(10), v0: uint(5), v1: uint(10), expected: true},
		{op: state.OpIsBetween, v: uint(11), v0: uint(5), v1: uint(10), expected: false},
		{op: state.OpIsBetween, v: 1.1382645273452, v0: 1.5829371949265, v1: 2.0938546724332, expected: false},
		{op: state.OpIsBetween, v: 1.5829371949265, v0: 1.5829371949265, v1: 2.0938546724332, expected: true},
		{op: state.OpIsBetween, v: 1.7737061300485, v0: 1.5829371949265, v1: 2.0938546724332, expected: true},
		{op: state.OpIsBetween, v: 2.0938546724332, v0: 1.5829371949265, v1: 2.0938546724332, expected: true},
		{op: state.OpIsBetween, v: 2.5032846299106, v0: 1.5829371949265, v1: 2.0938546724332, expected: false},
		{op: state.OpIsBetween, v: float64(float32(6.018)), v0: float64(float32(7.983)), v1: float64(float32(9.579)), expected: false},
		{op: state.OpIsBetween, v: float64(float32(7.983)), v0: float64(float32(7.983)), v1: float64(float32(9.579)), expected: true},
		{op: state.OpIsBetween, v: float64(float32(8.125)), v0: float64(float32(7.983)), v1: float64(float32(9.579)), expected: true},
		{op: state.OpIsBetween, v: float64(float32(9.579)), v0: float64(float32(7.983)), v1: float64(float32(9.579)), expected: true},
		{op: state.OpIsBetween, v: float64(float32(12.662)), v0: float64(float32(7.983)), v1: float64(float32(9.579)), expected: false},
		{op: state.OpIsBetween, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("1204471285.038153"), v1: decimal.MustParse("3084136838.720635"), expected: false},
		{op: state.OpIsBetween, v: decimal.MustParse("1204471285.038153"), v0: decimal.MustParse("1204471285.038153"), v1: decimal.MustParse("3084136838.720635"), expected: true},
		{op: state.OpIsBetween, v: decimal.MustParse("2726608135.048165"), v0: decimal.MustParse("1204471285.038153"), v1: decimal.MustParse("3084136838.720635"), expected: true},
		{op: state.OpIsBetween, v: decimal.MustParse("3084136838.720635"), v0: decimal.MustParse("1204471285.038153"), v1: decimal.MustParse("3084136838.720635"), expected: true},
		{op: state.OpIsBetween, v: decimal.MustParse("8539500341.8264811"), v0: decimal.MustParse("947126405.18361204926705329"), v1: decimal.MustParse("3084136838.720635"), expected: false},
		{op: state.OpIsBetween, v: nil, v0: 5, v1: 8, expected: false},
		{op: state.OpIsBetween, v: json.Value("511.0"), v0: v670, v1: v812, expected: false},
		{op: state.OpIsBetween, v: json.Value("775.0"), v0: v670, v1: v812, expected: true},
		{op: state.OpIsBetween, v: json.Value("913.0"), v0: v670, v1: v812, expected: false},
		{op: state.OpIsBetween, v: json.Value("670"), v0: v670, v1: v812, expected: true},
		{op: state.OpIsBetween, v: json.Value("810"), v0: v670, v1: v812, expected: true},
		{op: state.OpIsBetween, v: json.Value("812.0"), v0: v670, v1: v812, expected: true},
		{op: state.OpIsBetween, v: json.Value(`"850"`), v0: v670, v1: v812, expected: false},
		{op: state.OpIsBetween, v: json.Value(`"670"`), v0: v670, v1: v812, expected: true},
		{op: state.OpIsBetween, v: json.Value("true"), v0: v670, v1: v812, expected: false},
		{op: state.OpIsBetween, v: json.Value(`"foo"`), v0: v670, v1: v812, expected: false},
		{op: state.OpIsBetween, v: json.Value(`""`), v0: v670, v1: v812, expected: false},
		{op: state.OpIsBetween, v: json.Value("null"), v0: v670, v1: v812, expected: false},

		// OpIsNotBetween.
		{op: state.OpIsNotBetween, v0: 350, v1: 412, expected: true, notExists: true},
		{op: state.OpIsNotBetween, v: 100, v0: 359, v1: 412, expected: true},
		{op: state.OpIsNotBetween, v: 359, v0: 359, v1: 412, expected: false},
		{op: state.OpIsNotBetween, v: 405, v0: 359, v1: 412, expected: false},
		{op: state.OpIsNotBetween, v: 412, v0: 359, v1: 412, expected: false},
		{op: state.OpIsNotBetween, v: 500, v0: 359, v1: 412, expected: true},
		{op: state.OpIsNotBetween, v: uint(3), v0: uint(5), v1: uint(10), expected: true},
		{op: state.OpIsNotBetween, v: uint(6), v0: uint(5), v1: uint(10), expected: false},
		{op: state.OpIsNotBetween, v: uint(5), v0: uint(5), v1: uint(10), expected: false},
		{op: state.OpIsNotBetween, v: uint(10), v0: uint(5), v1: uint(10), expected: false},
		{op: state.OpIsNotBetween, v: uint(11), v0: uint(5), v1: uint(10), expected: true},
		{op: state.OpIsNotBetween, v: 1.1382645273452, v0: 1.5829371949265, v1: 2.0938546724332, expected: true},
		{op: state.OpIsNotBetween, v: 1.5829371949265, v0: 1.5829371949265, v1: 2.0938546724332, expected: false},
		{op: state.OpIsNotBetween, v: 1.7737061300485, v0: 1.5829371949265, v1: 2.0938546724332, expected: false},
		{op: state.OpIsNotBetween, v: 2.0938546724332, v0: 1.5829371949265, v1: 2.0938546724332, expected: false},
		{op: state.OpIsNotBetween, v: 2.5032846299106, v0: 1.5829371949265, v1: 2.0938546724332, expected: true},
		{op: state.OpIsNotBetween, v: float64(float32(6.018)), v0: float64(float32(7.983)), v1: float64(float32(9.579)), expected: true},
		{op: state.OpIsNotBetween, v: float64(float32(7.983)), v0: float64(float32(7.983)), v1: float64(float32(9.579)), expected: false},
		{op: state.OpIsNotBetween, v: float64(float32(8.125)), v0: float64(float32(7.983)), v1: float64(float32(9.579)), expected: false},
		{op: state.OpIsNotBetween, v: float64(float32(9.579)), v0: float64(float32(7.983)), v1: float64(float32(9.579)), expected: false},
		{op: state.OpIsNotBetween, v: float64(float32(12.662)), v0: float64(float32(7.983)), v1: float64(float32(9.579)), expected: true},
		{op: state.OpIsNotBetween, v: decimal.MustParse("947126405.18361204926705329"), v0: decimal.MustParse("1204471285.038153"), v1: decimal.MustParse("3084136838.720635"), expected: true},
		{op: state.OpIsNotBetween, v: decimal.MustParse("1204471285.038153"), v0: decimal.MustParse("1204471285.038153"), v1: decimal.MustParse("3084136838.720635"), expected: false},
		{op: state.OpIsNotBetween, v: decimal.MustParse("2726608135.048165"), v0: decimal.MustParse("1204471285.038153"), v1: decimal.MustParse("3084136838.720635"), expected: false},
		{op: state.OpIsNotBetween, v: decimal.MustParse("3084136838.720635"), v0: decimal.MustParse("1204471285.038153"), v1: decimal.MustParse("3084136838.720635"), expected: false},
		{op: state.OpIsNotBetween, v: decimal.MustParse("8539500341.8264811"), v0: decimal.MustParse("947126405.18361204926705329"), v1: decimal.MustParse("3084136838.720635"), expected: true},
		{op: state.OpIsNotBetween, v: nil, v0: 5, v1: 8, expected: true},
		{op: state.OpIsNotBetween, v: json.Value("511.0"), v0: v670, v1: v812, expected: true},
		{op: state.OpIsNotBetween, v: json.Value("670.0"), v0: v670, v1: v812, expected: false},
		{op: state.OpIsNotBetween, v: json.Value("775.0"), v0: v670, v1: v812, expected: false},
		{op: state.OpIsNotBetween, v: json.Value("812.0"), v0: v670, v1: v812, expected: false},
		{op: state.OpIsNotBetween, v: json.Value("913.0"), v0: v670, v1: v812, expected: true},
		{op: state.OpIsNotBetween, v: json.Value(`"670"`), v0: v670, v1: v812, expected: false},
		{op: state.OpIsNotBetween, v: json.Value("812.0"), v0: v670, v1: v812, expected: false},
		{op: state.OpIsNotBetween, v: json.Value(`"850"`), v0: v670, v1: v812, expected: true},
		{op: state.OpIsNotBetween, v: json.Value(`"670"`), v0: v670, v1: v812, expected: false},
		{op: state.OpIsNotBetween, v: json.Value("true"), v0: v670, v1: v812, expected: true},
		{op: state.OpIsNotBetween, v: json.Value(`"foo"`), v0: v670, v1: v812, expected: true},
		{op: state.OpIsNotBetween, v: json.Value(`""`), v0: v670, v1: v812, expected: true},
		{op: state.OpIsNotBetween, v: json.Value("null"), v0: v670, v1: v812, expected: true},

		// OpContains.
		{op: state.OpContains, v0: "boo", expected: false, notExists: true},
		{op: state.OpContains, v: "foo boo", v0: "boo", expected: true},
		{op: state.OpContains, v: "foo", v0: "boo", expected: false},
		{op: state.OpContains, v: "", v0: "boo", expected: false},
		{op: state.OpContains, v: "", v0: "", expected: true},
		{op: state.OpContains, v: "foo", v0: "", expected: true},
		{op: state.OpContains, v: nil, v0: "foo", expected: false},
		{op: state.OpContains, v: []any{5, 12, 7}, v0: 7, expected: true},
		{op: state.OpContains, v: []any{5, 12, 7}, v0: 8, expected: false},
		{op: state.OpContains, v: []any{5, 12, 7}, v0: 7, expected: true},
		{op: state.OpContains, v: []any{"foo", "boo"}, v0: "foo", expected: true},
		{op: state.OpContains, v: []any{"foo", "boo"}, v0: "moo", expected: false},
		{op: state.OpContains, v: []any{decimal.MustInt(1), decimal.MustInt(2)}, v0: decimal.MustInt(2), expected: true},
		{op: state.OpContains, v: []any{decimal.MustInt(1)}, v0: decimal.MustInt(2), expected: false},
		{op: state.OpContains, v: []any{time.Unix(0, 0), time.Unix(10, 0)}, v0: time.Unix(0, 0), expected: true},
		{op: state.OpContains, v: []any{time.Unix(0, 0)}, v0: time.Unix(5, 0), expected: false},
		{op: state.OpContains, v: json.Value(`"foo"`), v0: vFoo, expected: true},
		{op: state.OpContains, v: json.Value(`"boo"`), v0: vFoo, expected: false},
		{op: state.OpContains, v: json.Value(`""`), v0: vFoo, expected: false},
		{op: state.OpContains, v: json.Value(`[6.05,23,812.0,-100.93]`), v0: v812, expected: true},
		{op: state.OpContains, v: json.Value(`[6.05,23,815.0,-100.93]`), v0: v812, expected: false},
		{op: state.OpContains, v: json.Value("5.0"), v0: vFoo, expected: false},
		{op: state.OpContains, v: json.Value("true"), v0: vFoo, expected: false},
		{op: state.OpContains, v: json.Value("null"), v0: vFoo, expected: false},

		// OpDoesNotContain.
		{op: state.OpDoesNotContain, v0: "boo", expected: true, notExists: true},
		{op: state.OpDoesNotContain, v: "foo boo", v0: "boo", expected: false},
		{op: state.OpDoesNotContain, v: "foo", v0: "boo", expected: true},
		{op: state.OpDoesNotContain, v: "", v0: "boo", expected: true},
		{op: state.OpDoesNotContain, v: "", v0: "", expected: false},
		{op: state.OpDoesNotContain, v: "foo", v0: "", expected: false},
		{op: state.OpDoesNotContain, v: nil, v0: "boo", expected: true},
		{op: state.OpDoesNotContain, v: []any{5, 12, 7}, v0: 7, expected: false},
		{op: state.OpDoesNotContain, v: []any{5, 12, 7}, v0: 8, expected: true},
		{op: state.OpDoesNotContain, v: []any{5, 12, 7}, v0: 7, expected: false},
		{op: state.OpDoesNotContain, v: []any{"foo", "boo"}, v0: "foo", expected: false},
		{op: state.OpDoesNotContain, v: []any{"foo", "boo"}, v0: "moo", expected: true},
		{op: state.OpDoesNotContain, v: []any{decimal.MustInt(1), decimal.MustInt(2)}, v0: decimal.MustInt(2), expected: false},
		{op: state.OpDoesNotContain, v: []any{decimal.MustInt(1)}, v0: decimal.MustInt(2), expected: true},
		{op: state.OpDoesNotContain, v: []any{time.Unix(0, 0), time.Unix(10, 0)}, v0: time.Unix(0, 0), expected: false},
		{op: state.OpDoesNotContain, v: []any{time.Unix(0, 0)}, v0: time.Unix(5, 0), expected: true},
		{op: state.OpDoesNotContain, v: json.Value(`"foo"`), v0: vFoo, expected: false},
		{op: state.OpDoesNotContain, v: json.Value(`"boo"`), v0: vFoo, expected: true},
		{op: state.OpDoesNotContain, v: json.Value(`""`), v0: vFoo, expected: true},
		{op: state.OpDoesNotContain, v: json.Value(`[6.05,23,812.0,-100.93]`), v0: v812, expected: false},
		{op: state.OpDoesNotContain, v: json.Value(`[6.05,23,815.0,-100.93]`), v0: v812, expected: true},
		{op: state.OpDoesNotContain, v: json.Value("5.0"), v0: vFoo, expected: true},
		{op: state.OpDoesNotContain, v: json.Value("true"), v0: vFoo, expected: true},
		{op: state.OpDoesNotContain, v: json.Value("null"), v0: vFoo, expected: true},

		// OpIsOneOf.
		{op: state.OpIsOneOf, v0: "foo", expected: false, notExists: true},
		{op: state.OpIsOneOf, v: "foo", v0: "boo", v1: "foo", expected: true},
		{op: state.OpIsOneOf, v: "foo", v0: "foo", v1: "boo", expected: true},
		{op: state.OpIsOneOf, v: "foo", v0: "boo", v1: "goo", expected: false},
		{op: state.OpIsOneOf, v: 5, v0: 5, v1: 3, expected: true},
		{op: state.OpIsOneOf, v: 5, v0: 3, v1: 7, expected: false},
		{op: state.OpIsOneOf, v: 1.2, v0: 1.1, v1: 1.2, expected: true},
		{op: state.OpIsOneOf, v: 1.2, v0: 1.1, v1: 1.3, expected: false},
		{op: state.OpIsOneOf, v: n670, v0: n812, v1: n670, expected: true},
		{op: state.OpIsOneOf, v: n670, v0: n812, expected: false},
		{op: state.OpIsOneOf, v: nil, v0: "foo", v1: "boo", expected: false},
		{op: state.OpIsOneOf, v: json.Value(`"foo"`), v0: vFoo, v1: vBoo, expected: true},
		{op: state.OpIsOneOf, v: json.Value(`"foo"`), v0: vBoo, v1: vFoo, expected: true},
		{op: state.OpIsOneOf, v: json.Value(`"goo"`), v0: vBoo, v1: vFoo, expected: false},
		{op: state.OpIsOneOf, v: json.Value("670.0"), v0: v812, v1: v670, expected: true},
		{op: state.OpIsOneOf, v: json.Value("null"), v0: vFoo, v1: vBoo, expected: false},

		// OpIsNotOneOf.
		{op: state.OpIsNotOneOf, v0: "foo", expected: true, notExists: true},
		{op: state.OpIsNotOneOf, v: "foo", v0: "boo", v1: "foo", expected: false},
		{op: state.OpIsNotOneOf, v: "foo", v0: "foo", v1: "boo", expected: false},
		{op: state.OpIsNotOneOf, v: "foo", v0: "boo", v1: "goo", expected: true},
		{op: state.OpIsNotOneOf, v: 5, v0: 5, v1: 3, expected: false},
		{op: state.OpIsNotOneOf, v: 5, v0: 3, v1: 7, expected: true},
		{op: state.OpIsNotOneOf, v: 1.2, v0: 1.1, v1: 1.2, expected: false},
		{op: state.OpIsNotOneOf, v: 1.2, v0: 1.1, v1: 1.3, expected: true},
		{op: state.OpIsNotOneOf, v: n670, v0: n812, v1: n670, expected: false},
		{op: state.OpIsNotOneOf, v: n670, v0: n812, expected: true},
		{op: state.OpIsNotOneOf, v: nil, v0: "foo", v1: "boo", expected: true},
		{op: state.OpIsNotOneOf, v: json.Value(`"foo"`), v0: vFoo, v1: vBoo, expected: false},
		{op: state.OpIsNotOneOf, v: json.Value(`"foo"`), v0: vBoo, v1: vFoo, expected: false},
		{op: state.OpIsNotOneOf, v: json.Value(`"goo"`), v0: vBoo, v1: vFoo, expected: true},
		{op: state.OpIsNotOneOf, v: json.Value("670.0"), v0: v812, v1: v670, expected: false},
		{op: state.OpIsNotOneOf, v: json.Value("670.0"), v0: v812, v1: v670, expected: false},
		{op: state.OpIsNotOneOf, v: json.Value("null"), v0: vFoo, v1: vBoo, expected: true},

		// OpStartsWith.
		{op: state.OpStartsWith, v0: "foo", expected: false, notExists: true},
		{op: state.OpStartsWith, v: "foo boo", v0: "foo", expected: true},
		{op: state.OpStartsWith, v: "foo", v0: "boo", expected: false},
		{op: state.OpStartsWith, v: "", v0: "boo", expected: false},
		{op: state.OpStartsWith, v: "boo", v0: "", expected: true},
		{op: state.OpStartsWith, v: nil, v0: "foo", expected: false},
		{op: state.OpStartsWith, v: json.Value(`"foo"`), v0: vFoo, expected: true},
		{op: state.OpStartsWith, v: json.Value(`"boo"`), v0: vFoo, expected: false},
		{op: state.OpStartsWith, v: json.Value(`"foo boo"`), v0: vFoo, expected: true},
		{op: state.OpStartsWith, v: json.Value(`""`), v0: vFoo, expected: false},
		{op: state.OpStartsWith, v: json.Value("5.0"), v0: vFoo, expected: false},
		{op: state.OpStartsWith, v: json.Value("true"), v0: vFoo, expected: false},
		{op: state.OpStartsWith, v: json.Value("null"), v0: vFoo, expected: false},

		// OpEndsWith.
		{op: state.OpEndsWith, v0: "boo", expected: false, notExists: true},
		{op: state.OpEndsWith, v: "foo boo", v0: "boo", expected: true},
		{op: state.OpEndsWith, v: "boo", v0: "boo", expected: true},
		{op: state.OpEndsWith, v: "boo foo", v0: "boo", expected: false},
		{op: state.OpEndsWith, v: "boo", v0: "", expected: true},
		{op: state.OpEndsWith, v: nil, v0: "foo", expected: false},
		{op: state.OpEndsWith, v: json.Value(`"boo"`), v0: vFoo, expected: false},
		{op: state.OpEndsWith, v: json.Value(`"boo foo"`), v0: vFoo, expected: true},
		{op: state.OpEndsWith, v: json.Value(`"foo boo"`), v0: vFoo, expected: false},
		{op: state.OpEndsWith, v: json.Value(`""`), v0: vFoo, expected: false},
		{op: state.OpEndsWith, v: json.Value("5.0"), v0: vFoo, expected: false},
		{op: state.OpEndsWith, v: json.Value("true"), v0: vFoo, expected: false},
		{op: state.OpEndsWith, v: json.Value("nil"), v0: vFoo, expected: false},

		// OpIsBefore.
		{op: state.OpIsBefore, v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false, notExists: true},
		{op: state.OpIsBefore, v: time.Date(2024, 9, 10, 8, 23, 44, 395612580, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: true},
		{op: state.OpIsBefore, v: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},
		{op: state.OpIsBefore, v: time.Date(2024, 9, 12, 14, 5, 23, 539572515, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},
		{op: state.OpIsBefore, v: time.Date(2024, 9, 8, 0, 0, 0, 0, time.UTC), v0: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC), expected: true},
		{op: state.OpIsBefore, v: time.Date(1970, 1, 1, 16, 51, 6, 471190035, time.UTC), v0: time.Date(1970, 1, 1, 14, 13, 45, 632886017, time.UTC), expected: false},
		{op: state.OpIsBefore, v: nil, v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},

		// OpIsOnOrBefore.
		{op: state.OpIsOnOrBefore, v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false, notExists: true},
		{op: state.OpIsOnOrBefore, v: time.Date(2024, 9, 10, 8, 23, 44, 395612580, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: true},
		{op: state.OpIsOnOrBefore, v: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: true},
		{op: state.OpIsOnOrBefore, v: time.Date(2024, 9, 12, 14, 5, 23, 539572515, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},
		{op: state.OpIsOnOrBefore, v: time.Date(2024, 9, 8, 0, 0, 0, 0, time.UTC), v0: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC), expected: true},
		{op: state.OpIsOnOrBefore, v: time.Date(1970, 1, 1, 16, 51, 6, 471190035, time.UTC), v0: time.Date(1970, 1, 1, 14, 13, 45, 632886017, time.UTC), expected: false},
		{op: state.OpIsOnOrBefore, v: nil, v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},

		// OpIsAfter.
		{op: state.OpIsAfter, v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false, notExists: true},
		{op: state.OpIsAfter, v: time.Date(2024, 9, 10, 8, 23, 44, 395612580, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},
		{op: state.OpIsAfter, v: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},
		{op: state.OpIsAfter, v: time.Date(2024, 9, 12, 14, 5, 23, 539572515, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: true},
		{op: state.OpIsAfter, v: time.Date(2024, 9, 8, 0, 0, 0, 0, time.UTC), v0: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC), expected: false},
		{op: state.OpIsAfter, v: time.Date(1970, 1, 1, 16, 51, 6, 471190035, time.UTC), v0: time.Date(1970, 1, 1, 14, 13, 45, 632886017, time.UTC), expected: true},
		{op: state.OpIsAfter, v: nil, v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},

		// OpIsOnOrAfter.
		{op: state.OpIsOnOrAfter, v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false, notExists: true},
		{op: state.OpIsOnOrAfter, v: time.Date(2024, 9, 10, 8, 23, 44, 395612580, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},
		{op: state.OpIsOnOrAfter, v: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: true},
		{op: state.OpIsOnOrAfter, v: time.Date(2024, 9, 12, 14, 5, 23, 539572515, time.UTC), v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: true},
		{op: state.OpIsOnOrAfter, v: time.Date(2024, 9, 8, 0, 0, 0, 0, time.UTC), v0: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC), expected: false},
		{op: state.OpIsOnOrAfter, v: time.Date(1970, 1, 1, 16, 51, 6, 471190035, time.UTC), v0: time.Date(1970, 1, 1, 14, 13, 45, 632886017, time.UTC), expected: true},
		{op: state.OpIsOnOrAfter, v: nil, v0: time.Date(2024, 9, 10, 9, 58, 15, 704152446, time.UTC), expected: false},

		// OpIsTrue.
		{op: state.OpIsTrue, expected: false, notExists: true},
		{op: state.OpIsTrue, v: true, expected: true},
		{op: state.OpIsTrue, v: false, expected: false},
		{op: state.OpIsTrue, v: json.Value(`true`), expected: true},
		{op: state.OpIsTrue, v: json.Value(`false`), expected: false},
		{op: state.OpIsTrue, v: 5, expected: false},
		{op: state.OpIsTrue, v: nil, expected: false},

		// OpIsFalse.
		{op: state.OpIsFalse, expected: false, notExists: true},
		{op: state.OpIsFalse, v: true, expected: false},
		{op: state.OpIsFalse, v: false, expected: true},
		{op: state.OpIsFalse, v: json.Value(`true`), expected: false},
		{op: state.OpIsFalse, v: json.Value(`false`), expected: true},
		{op: state.OpIsFalse, v: 5, expected: false},
		{op: state.OpIsFalse, v: nil, expected: false},

		// OpIsNull.
		{op: state.OpIsNull, expected: false, notExists: true},
		{op: state.OpIsNull, v: nil, expected: true},
		{op: state.OpIsNull, v: 5, expected: false},
		{op: state.OpIsNull, v: json.Value(`true`), expected: false},
		{op: state.OpIsNull, v: json.Value(`null`), expected: true},

		// OpIsNotNull.
		{op: state.OpIsNotNull, expected: true, notExists: true},
		{op: state.OpIsNotNull, v: nil, expected: false},
		{op: state.OpIsNotNull, v: 5, expected: true},
		{op: state.OpIsNotNull, v: json.Value(`true`), expected: true},
		{op: state.OpIsNotNull, v: json.Value(`null`), expected: false},

		// OpExists.
		{op: state.OpExists, expected: false, notExists: true},
		{op: state.OpExists, v: json.Value(`5`), expected: true},

		// OpDoesNotExist.
		{op: state.OpDoesNotExist, expected: true, notExists: true},
		{op: state.OpDoesNotExist, v: json.Value(`"foo"`), expected: false},
	}

	for _, test := range tests {
		name := fmt.Sprintf("%#v %s (%#v, %#v)", test.v, test.op, test.v0, test.v1)
		t.Run(name, func(t *testing.T) {
			filter := &state.Where{
				Logical: state.OpAnd,
				Conditions: []state.WhereCondition{
					{Property: []string{"v"}, Operator: test.op},
				},
			}
			if test.v0 != nil {
				if test.v1 != nil {
					filter.Conditions[0].Values = []any{test.v0, test.v1}
				} else {
					filter.Conditions[0].Values = []any{test.v0}
				}
			}
			properties := map[string]any{"v": test.v}
			if test.notExists {
				delete(properties, "v")
			}
			got := Applies(filter, properties)
			if test.expected != got {
				t.Fatalf("expected %t, got %t", test.expected, got)
			}
		})
	}

	t.Run("nil filter", func(t *testing.T) {
		if !Applies(nil, map[string]any{"v": 5}) {
			t.Fatal("expected true, got false")
		}
	})

	t.Run("logical and", func(t *testing.T) {
		filter := &state.Where{
			Logical: state.OpAnd,
			Conditions: []state.WhereCondition{
				{Property: []string{"a"}, Operator: state.OpIs, Values: []any{5}},
				{Property: []string{"b"}, Operator: state.OpContains, Values: []any{"boo"}},
			},
		}
		if !Applies(filter, map[string]any{"a": 5, "b": "foo boo"}) {
			t.Fatal("expected true, got false")
		}
		if Applies(filter, map[string]any{"a": 5, "b": "foo"}) {
			t.Fatal("expected false, got true")
		}
	})

	t.Run("logical or", func(t *testing.T) {
		filter := &state.Where{
			Logical: state.OpOr,
			Conditions: []state.WhereCondition{
				{Property: []string{"a"}, Operator: state.OpIs, Values: []any{5}},
				{Property: []string{"b"}, Operator: state.OpContains, Values: []any{"boo"}},
			},
		}
		if !Applies(filter, map[string]any{"a": 5, "b": "foo boo"}) {
			t.Fatal("expected true, got false")
		}
		if !Applies(filter, map[string]any{"a": 5, "b": "foo"}) {
			t.Fatal("expected true, got false")
		}
		if Applies(filter, map[string]any{"a": 3, "b": "foo"}) {
			t.Fatal("expected false, got true")
		}
	})

}

// Test_AppliesWithNestedPath ensures Applies handles filters with property
// paths.
func Test_AppliesWithNestedPath(t *testing.T) {
	filter := &state.Where{
		Logical: state.OpAnd,
		Conditions: []state.WhereCondition{
			{Property: []string{"n1", "b"}, Operator: state.OpIs, Values: []any{5}},
			{Property: []string{"n2", "b"}, Operator: state.OpIs, Values: []any{func() state.JSONConditionValue {
				d := decimal.MustInt(5)
				return state.JSONConditionValue{Number: &d, String: "5"}
			}()}},
		},
	}

	props := map[string]any{
		"n1": map[string]any{"b": 5},
		"n2": json.Value(`{"b":5}`),
	}

	if !Applies(filter, props) {
		t.Fatal("expected true, got false")
	}

	delete(props, "n1")
	if Applies(filter, props) {
		t.Fatal("expected false when missing n1")
	}
}

// Test_readPropertyFrom verifies retrieving nested properties from maps and
// JSON.
func Test_readPropertyFrom(t *testing.T) {
	jsonObj := json.Value(`{"b":{"c":4}}`)
	m := map[string]any{
		"a": 5,
		"nested": map[string]any{
			"b": map[string]any{"c": "foo"},
		},
		"json": jsonObj,
	}

	cases := []struct {
		path     []string
		expected any
		ok       bool
	}{
		{[]string{"a"}, 5, true},
		{[]string{"nested", "b", "c"}, "foo", true},
		{[]string{"nested", "x"}, nil, false},
		{[]string{"a", "b"}, nil, false},
		{[]string{"json", "b", "c"}, json.Value("4"), true},
		{[]string{"json", "x"}, nil, false},
	}

	for _, cas := range cases {
		got, ok := readPropertyFrom(m, cas.path)
		if ok != cas.ok || !reflect.DeepEqual(got, cas.expected) {
			t.Fatalf("%v: expected (%v,%v) got (%v,%v)", cas.path, cas.expected, cas.ok, got, ok)
		}
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic with empty path")
		}
	}()
	readPropertyFrom(m, []string{})
}
