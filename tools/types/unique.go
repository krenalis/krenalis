// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
)

const uniqueMapThreshold = 20

// FirstDuplicate returns the index of the first element equal to an earlier
// element, or -1 if there are no duplicates. It does not modify values.
// The element type must be a non-generic scalar permitted by [Type.WithUnique].
// Elements must use the corresponding Go representation. Temporal elements
// may instead all be strings or, for datetimes, int64 timestamps. Strings are
// compared verbatim, without parsing.
//
// Decimals are compared numerically, datetimes by instant, dates by calendar
// date, and times by clock time including nanoseconds. The locations and
// monotonic clock metadata of time.Time values are ignored. All NaN values
// compare equal, as do positive and negative zero.
//
// Go representations and decimal scales are validated as values are scanned.
// Slices with fewer than 20 elements use direct comparisons; slices with 20
// or more elements use a map with memory proportional to their length.
func FirstDuplicate(values []any, elementType Type) (int, error) {

	if !elementType.Valid() || elementType.Generic() {
		return -1, fmt.Errorf("element type must be a non-generic scalar")
	}
	kind := elementType.Kind()
	var (
		representation reflect.Type
		scale          int
	)
	switch kind {
	case StringKind, UUIDKind, IPKind:
		representation = reflect.TypeFor[string]()
	case BooleanKind:
		representation = reflect.TypeFor[bool]()
	case IntKind:
		representation = reflect.TypeFor[int]()
		if elementType.IsUnsigned() {
			representation = reflect.TypeFor[uint]()
		}
	case FloatKind:
		representation = reflect.TypeFor[float64]()
	case DecimalKind:
		representation = reflect.TypeFor[decimal.Decimal]()
		scale = elementType.Scale()
	case YearKind:
		representation = reflect.TypeFor[int]()
	case DateTimeKind, DateKind, TimeKind:
		representation = reflect.TypeFor[time.Time]()
		if len(values) > 0 {
			switch values[0].(type) {
			case string:
				representation = reflect.TypeFor[string]()
			case int64:
				if kind == DateTimeKind {
					representation = reflect.TypeFor[int64]()
				}
			}
		}
	default:
		return -1, fmt.Errorf("element type %s does not support uniqueness", elementType)
	}

	var keys [uniqueMapThreshold]any
	var seen map[any]struct{}
	if len(values) >= uniqueMapThreshold {
		seen = make(map[any]struct{}, len(values))
	}
	for i, value := range values {

		if reflect.TypeOf(value) != representation {
			return -1, fmt.Errorf("element %d has Go type %T, expected %s", i, value, representation)
		}
		key := value
		switch value := value.(type) {
		case decimal.Decimal:
			if value.Sign() == 0 {
				key = ""
				break
			}
			binary, err := value.Binary(scale)
			if err != nil {
				return -1, fmt.Errorf("element %d cannot be represented at decimal scale %d", i, scale)
			}
			// Remove redundant sign extension so equivalent coefficients have the same key.
			for len(binary) > 1 &&
				((binary[0] == 0 && binary[1]&0x80 == 0) ||
					(binary[0] == 0xff && binary[1]&0x80 != 0)) {
				binary = binary[1:]
			}
			key = string(binary)
		case time.Time:
			switch kind {
			case DateTimeKind:
				key = value.UTC()
			case DateKind:
				year, month, day := value.Date()
				key = time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
			case TimeKind:
				hour, minute, second := value.Clock()
				key = time.Date(1970, 1, 1, hour, minute, second, value.Nanosecond(), time.UTC)
			}
		case float64:
			// NaN != NaN, so use nil as the shared key for all NaNs.
			if math.IsNaN(value) {
				key = nil
			}
		}

		if seen != nil {
			if _, exists := seen[key]; exists {
				return i, nil
			}
			seen[key] = struct{}{}
		} else {
			if slices.Contains(keys[:i], key) {
				return i, nil
			}
			keys[i] = key
		}

	}

	return -1, nil
}
