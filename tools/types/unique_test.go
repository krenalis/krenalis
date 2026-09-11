// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
)

// TestFirstDuplicate checks that FirstDuplicate returns the first duplicate
// index and applies the documented equality rules.
func TestFirstDuplicate(t *testing.T) {

	instant := time.Date(2026, 9, 8, 12, 30, 0, 125000000, time.UTC)
	zone := time.FixedZone("offset", 3600)
	withMonotonic := time.Now()
	tests := []struct {
		name    string
		element Type
		values  []any
		want    int
	}{
		{"empty", String(), nil, -1},
		{"singleton", Boolean(), []any{true}, -1},
		{"first duplicate", Int(32), []any{1, 2, 2, 1}, 2},
		{"duplicate before invalid representation", String(), []any{"a", "a", 1}, 1},
		{"unsigned", Int(64).Unsigned(), []any{uint(1), uint(2), uint(1)}, 2},
		{"booleans", Boolean(), []any{true, false, true}, 2},
		{"string case", String(), []any{"Foo", "foo"}, -1},
		{"strings", String(), []any{"a", "b", "a"}, 2},
		{"years", Year(), []any{2025, 2026, 2025}, 2},
		{"NaN representations", Float(64), []any{math.NaN(), math.Float64frombits(0xfff0000000000001)}, 1},
		{"one NaN", Float(64), []any{0.0, math.NaN(), math.Inf(1), math.Inf(-1)}, -1},
		{"signed zero", Float(64), []any{0.0, math.Copysign(0, -1)}, 1},
		{"equivalent decimals", Decimal(6, 2), []any{decimal.New(15, 1), decimal.MustParse("1.50")}, 1},
		{"decimal signed zero", Decimal(6, 2), []any{decimal.Decimal{}, decimal.MustParse("-0.00")}, 1},
		{
			"equivalent large decimals", Decimal(76, 2),
			[]any{decimal.MustParse("1e25"), decimal.MustParse("10000000000000000000000000.00")}, 1,
		},
		{"datetime time zones", DateTime(), []any{instant, instant.In(zone)}, 1},
		{"datetime monotonic clock", DateTime(), []any{withMonotonic, withMonotonic.Round(0)}, 1},
		{"date ignores time", Date(), []any{instant, instant.Add(time.Hour)}, 1},
		{
			"date ignores location", Date(),
			[]any{time.Date(2026, 9, 8, 1, 2, 3, 0, time.UTC), time.Date(2026, 9, 8, 20, 30, 0, 0, zone)}, 1,
		},
		{"time ignores date", Time(), []any{instant, instant.AddDate(0, 1, 0)}, 1},
		{
			"time ignores location", Time(),
			[]any{instant, time.Date(2030, 1, 1, 12, 30, 0, 125000000, zone)}, 1,
		},
		{"time nanoseconds", Time(), []any{instant, instant.Add(time.Nanosecond)}, -1},
		{"date strings", Date(), []any{"2026-09-08", "2026-10-08", "2026-09-08"}, 2},
		{
			"datetime strings verbatim", DateTime(),
			[]any{"2026-09-08T12:30:00Z", "2026-09-08T13:30:00+01:00"}, -1,
		},
		{"timestamps", DateTime(), []any{int64(42), int64(43), int64(42)}, 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := FirstDuplicate(test.values, test.element)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got index %d, want %d", got, test.want)
			}
		})
	}

}

// TestFirstDuplicateMap checks that map-based lookups use the documented
// equality and validation rules.
func TestFirstDuplicateMap(t *testing.T) {

	instant := time.Date(2026, 9, 8, 12, 30, 0, 125000000, time.UTC)
	size := uniqueMapThreshold + 1
	tests := []struct {
		name      string
		element   Type
		fill      func(int) any
		tail      []any
		wantIndex int
		wantError bool
	}{
		{
			"NaNs", Float(64), func(i int) any { return float64(i) },
			[]any{math.NaN(), math.Float64frombits(0xfff0000000000001)}, size - 1, false,
		},
		{
			"decimals", Decimal(6, 2), func(i int) any { return decimal.New(int64(i+100), 0) },
			[]any{decimal.New(15, 1), decimal.MustParse("1.50")}, size - 1, false,
		},
		{
			"datetimes", DateTime(), func(i int) any { return instant.AddDate(0, 0, i+1) },
			[]any{instant, instant.In(time.FixedZone("offset", 3600))}, size - 1, false,
		},
		{
			"dates", Date(), func(i int) any { return instant.AddDate(0, 0, i+1) },
			[]any{instant, instant.Add(time.Hour)}, size - 1, false,
		},
		{
			"times", Time(), func(i int) any { return instant.Add(time.Duration(i+1) * time.Second) },
			[]any{instant, instant.AddDate(0, 1, 0)}, size - 1, false,
		},
		{
			"date strings", Date(), func(i int) any { return fmt.Sprint(i) },
			[]any{"2026-09-08", "2026-10-08", "2026-09-08"}, size - 1, false,
		},
		{
			"timestamps", DateTime(), func(i int) any { return int64(i + 100) },
			[]any{int64(42), int64(43), int64(42)}, size - 1, false,
		},
		{
			"invalid representation", String(), func(i int) any { return fmt.Sprint(i) },
			[]any{1}, -1, true,
		},
		{
			"unrepresentable decimal scale", Decimal(6, 2), func(i int) any { return decimal.New(int64(i+1), 0) },
			[]any{decimal.MustParse("1.234")}, -1, true,
		},
		{
			"mixed temporal representations", DateTime(), func(i int) any { return instant.AddDate(0, 0, i) },
			[]any{"2026-09-08T00:00:00Z"}, -1, true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			values := make([]any, size-len(test.tail), size)
			for i := range values {
				values[i] = test.fill(i)
			}
			values = append(values, test.tail...)
			index, err := FirstDuplicate(values, test.element)
			if test.wantError {
				if err == nil {
					t.Fatalf("got index %d, want an error", index)
				}
				if index != -1 {
					t.Fatalf("got index %d with error %v, want -1", index, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if index != test.wantIndex {
				t.Fatalf("got index %d, want %d", index, test.wantIndex)
			}

		})
	}

}

// TestFirstDuplicateInvalidArguments checks that invalid types and
// representations return errors rather than panicking.
func TestFirstDuplicateInvalidArguments(t *testing.T) {

	tests := []struct {
		name    string
		element Type
		values  []any
	}{
		{"invalid type", Type{}, nil},
		{"generic type", Parameter("T"), nil},
		{"JSON type", JSON(), nil},
		{"array type", Array(String()), nil},
		{"map type", Map(String()), nil},
		{"object type", Object([]Property{{Name: "x", Type: String()}}), nil},
		{"nil element", String(), []any{nil}},
		{"wrong integer representation", Int(64), []any{int64(1)}},
		{"wrong unsigned representation", Int(64).Unsigned(), []any{1}},
		{"wrong float representation", Float(32), []any{float32(1)}},
		{"slice representation", String(), []any{[]any{1}}},
		{"invalid representation before duplicate", String(), []any{"a", 1, "a"}},
		{"mixed temporal representations", DateTime(), []any{time.Time{}, "2026-09-08T00:00:00Z"}},
		{"date timestamp", Date(), []any{int64(1)}},
		{"unrepresentable decimal scale", Decimal(6, 2), []any{decimal.MustParse("1.234")}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			index, err := FirstDuplicate(test.values, test.element)
			if err != nil {
				if index != -1 {
					t.Fatalf("got index %d with error %v, want -1", index, err)
				}
				return
			}
			t.Fatalf("got index %d, want an argument error", index)
		})
	}

}

// TestFirstDuplicateSmallAllocations checks that a 19-element slice of
// integers requires no allocations.
func TestFirstDuplicateSmallAllocations(t *testing.T) {

	values := make([]any, uniqueMapThreshold-1)
	for i := range values {
		values[i] = i
	}
	element := Int(32)
	allocs := testing.AllocsPerRun(100, func() {
		index, err := FirstDuplicate(values, element)
		if err != nil {
			t.Fatal(err)
		}
		if index != -1 {
			t.Fatalf("got duplicate index %d for distinct values", index)
		}
	})
	if allocs != 0 {
		t.Fatalf("got %g allocations, want zero", allocs)
	}

}

// TestFirstDuplicateThreshold checks duplicate detection below, at, and above
// the map threshold for several element representations.
func TestFirstDuplicateThreshold(t *testing.T) {

	tests := []struct {
		name    string
		element Type
		fill    func(int) any
	}{
		{"int", Int(32), func(i int) any { return i }},
		{"datetime string", DateTime(), func(i int) any { return fmt.Sprint(i) }},
		{"datetime timestamp", DateTime(), func(i int) any { return int64(i) }},
		{
			"decimal", Decimal(76, 2),
			func(i int) any {
				return decimal.MustParse(fmt.Sprint(i+1) + strings.Repeat("9", 20) + ".25")
			},
		},
	}

	for _, size := range []int{uniqueMapThreshold - 1, uniqueMapThreshold, uniqueMapThreshold + 1} {
		for _, test := range tests {
			t.Run(fmt.Sprintf("%s/%d", test.name, size), func(t *testing.T) {

				values := make([]any, size)
				for i := range values {
					values[i] = test.fill(i)
				}

				index, err := FirstDuplicate(values, test.element)
				if err != nil {
					t.Fatal(err)
				}
				if index != -1 {
					t.Fatalf("got duplicate index %d for distinct values", index)
				}
				values[1], values[2] = values[0], values[size-1]
				index, err = FirstDuplicate(values, test.element)
				if err != nil {
					t.Fatal(err)
				}
				if index != 1 {
					t.Fatalf("got index %d, want the first repeated position 1", index)
				}

			})
		}
	}

}

// TestFirstDuplicateUniqueKinds checks that FirstDuplicate and WithUnique
// support the same element kinds.
func TestFirstDuplicateUniqueKinds(t *testing.T) {

	for kind := StringKind; kind.Valid(); kind++ {

		elementType := Type{kind: kind}
		withUnique := true
		func() {
			defer func() {
				if recover() != nil {
					withUnique = false
				}
			}()
			_ = Array(elementType).WithUnique()
		}()

		_, err := FirstDuplicate(nil, elementType)
		if (err == nil) != withUnique {
			t.Fatalf("kind %s: WithUnique support is %t, FirstDuplicate returned %v", kind, withUnique, err)
		}

	}

}
