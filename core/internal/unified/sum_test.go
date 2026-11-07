// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"math/big"
	"testing"

	"github.com/krenalis/krenalis/tools/decimal"
)

// TestSumThreshold compares normalized tests against exact rational arithmetic.
func TestSumThreshold(t *testing.T) {
	for _, literal := range []string{"0", "-0", "1e-37", "-1e-37", "30.00001", "-30.00001",
		"30.00000", "-30", "1e75", "-1e75", "99999999999999.9999", "1000.0000", "5000.0000",
		"-2000.0000", "999.9999", "1000.0001", "1e33", "-1e33"} {
		for _, op := range []Comparison{Less, LessOrEqual, Greater, GreaterOrEqual} {
			test := SumTest{Operator: op, Value: decimal.MustParse(literal)}
			threshold, atLeast, err := test.threshold()
			if err != nil {
				t.Fatalf("expected normalized threshold, got %v", err)
			}
			precision, scale := threshold.PrecisionScale()
			if precision > 38 || scale > 4 {
				t.Fatalf("expected exact backend-representable threshold, got %s", threshold)
			}
			for _, value := range []string{"-100000000000000000000000000000000", "-30.0001", "-30", "-0.0001", "0",
				"0.0001", "29.9999", "30", "30.0001", "999.9999", "1000", "1500", "5000",
				"-2000", "100000000000000000000000000000000"} {
				a, _ := new(big.Rat).SetString(value)
				b, _ := new(big.Rat).SetString(literal)
				comparison := a.Cmp(b)
				want := false
				switch op {
				case Less:
					want = comparison < 0
				case LessOrEqual:
					want = comparison <= 0
				case Greater:
					want = comparison > 0
				case GreaterOrEqual:
					want = comparison >= 0
				default:
					t.Fatalf("expected ordered comparison, got %v", op)
				}
				n := decimal.MustParse(value)
				got := n.LessEqual(threshold)
				if atLeast {
					got = n.GreaterEqual(threshold)
				}
				if got != want {
					t.Fatalf("expected %s op%d %s = %v, got %v at normalized %s", value, op, literal, want, got, threshold)
				}
			}
		}
	}
}
