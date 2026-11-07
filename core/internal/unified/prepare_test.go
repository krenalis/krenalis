// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"math"
	"testing"
)

// TestAcceptedCountRanges checks decisions against an independent enumeration.
// Static invalid-parameter admission is separately tested by TestValidate.
func TestAcceptedCountRanges(t *testing.T) {
	check := func(test CountTest, lo, hi int64) {
		ranges := acceptedCountRanges(test)
		got := uint8(0)
		intersects := false
		for _, r := range ranges {
			if lo >= r.Lower && (r.Upper == nil || hi <= *r.Upper) {
				got = 1
				break
			}
			intersects = intersects || hi >= r.Lower && (r.Upper == nil || lo <= *r.Upper)
		}
		if got != 1 && intersects {
			got = 2
		}
		yes, no := false, false
		// Only small grids are enumerated; edge tests below use singleton bounds.
		for n := lo; ; n++ {
			ok := false
			switch test.Operator {
			case Equal:
				ok = uint64(n) == test.Value
			case Different:
				ok = uint64(n) != test.Value
			case Less:
				ok = uint64(n) < test.Value
			case LessOrEqual:
				ok = uint64(n) <= test.Value
			case Greater:
				ok = uint64(n) > test.Value
			case GreaterOrEqual:
				ok = uint64(n) >= test.Value
			case Between:
				ok = uint64(n) >= test.Value && uint64(n) <= test.Upper
			default:
				t.Fatalf("expected known comparison, got %d", test.Operator)
			}
			yes = yes || ok
			no = no || !ok
			if n == hi {
				break
			}
		}
		want := uint8(0)
		if yes {
			want = 1
		}
		if yes && no {
			want = 2
		}
		if got != want {
			t.Fatalf("expected %d for %+v [%d,%d], got %d", want, test, lo, hi, got)
		}
	}
	for _, op := range []Comparison{Equal, Different, Less, LessOrEqual, Greater, GreaterOrEqual, Between} {
		for v := uint64(0); v <= 6; v++ {
			for lo := int64(0); lo <= 8; lo++ {
				for hi := lo; hi <= 8; hi++ {
					check(CountTest{Operator: op, Value: v, Upper: v + 2}, lo, hi)
				}
			}
		}
		for _, v := range []uint64{math.MaxInt64 - 2, math.MaxInt64 - 1, math.MaxInt64} {
			if op == Greater && v == math.MaxInt64 {
				continue
			}
			for _, n := range []int64{math.MaxInt64 - 2, math.MaxInt64 - 1, math.MaxInt64} {
				check(CountTest{Operator: op, Value: v, Upper: math.MaxInt64}, n, n)
			}
		}
	}
	// A pointer distinguishes an explicit finite upper limit from no upper limit.
	finite := acceptedCountRanges(CountTest{Operator: LessOrEqual, Value: math.MaxInt64})
	infinite := acceptedCountRanges(CountTest{Operator: GreaterOrEqual})
	if finite[0].Upper == nil || infinite[0].Upper != nil {
		t.Fatal("expected finite maximum distinct from unbounded, got same representation")
	}
}
