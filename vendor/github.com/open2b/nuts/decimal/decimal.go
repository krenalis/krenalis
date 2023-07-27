// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2018 Open2b
//

package decimal

import (
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/open2b/nuts/pkg/code.google.com/p/godec/dec"
)

type Dec struct {
	d *dec.Dec
}

const Accuracy dec.Scale = 28

var zero = Dec{d: dec.NewDecInt64(0)}

var down = dec.RoundDown
var halfDown = dec.RoundHalfDown
var halfEven = dec.RoundHalfEven
var halfUp = dec.RoundHalfUp
var up = dec.RoundUp

func Int(i int) Dec {
	return Dec{d: dec.NewDecInt64(int64(i))}
}

func Int64(i int64) Dec {
	return Dec{d: dec.NewDecInt64(i)}
}

func Float64(f float64, accuracy int) Dec {
	if accuracy < 0 {
		accuracy = int(Accuracy)
	}
	var rat = new(big.Rat).SetFloat64(f)
	var num = dec.NewDec(rat.Num(), 0)
	var denom = dec.NewDec(rat.Denom(), 0)
	num = num.Quo(num, denom, dec.Scale(accuracy), halfEven)
	if num == nil {
		panic("decimal.Float64 failed")
	}
	return Dec{d: num}
}

func String(s string) Dec {
	// rispetto a Open2b::Decimal, non vuole spazi all'inizio e alla fine
	var d, ok = str(s)
	if !ok {
		panic("Value '" + s + "' is not a valid decimal")
	}
	return d
}

func ParseString(s string) (Dec, bool) {
	return str(strings.TrimSpace(s))
}

func str(s string) (Dec, bool) {
	var exp dec.Scale
	if i := strings.IndexAny(s, "eE"); i != -1 && i != len(s)-1 {
		if e, err := strconv.ParseInt(s[i+1:], 10, 32); err == nil {
			s = s[:i]
			exp = dec.Scale(e)
		}
	}
	var d, ok = new(dec.Dec).SetString(s)
	if !ok {
		return zero, false
	}
	if exp != 0 {
		return Dec{d: d.SetScale(d.Scale() - exp)}, true
	}
	return Dec{d: d}, true
}

func Zero() Dec {
	return zero
}

func (a Dec) ComparedTo(b Dec) int {
	return a.d.Cmp(b.d)
}

func (a Dec) Divided(b Dec) Dec {
	if zero.d.Cmp(b.d) == 0 {
		panic("Division by zero")
	}
	return Dec{d: new(dec.Dec).Quo(a.d, b.d, Accuracy, halfEven)}
}

func (a Dec) IsNegative() bool {
	return a.d.Cmp(zero.d) < 0
}

func (a Dec) IsPositive() bool {
	return a.d.Cmp(zero.d) > 0
}

func (a Dec) IsZero() bool {
	return a.d.Cmp(zero.d) == 0
}

func (a Dec) Minus(b Dec) Dec {
	return Dec{d: new(dec.Dec).Sub(a.d, b.d)}
}

func (a Dec) Module(b Dec) Dec {
	if zero.d.Cmp(b.d) == 0 {
		panic("Division by zero")
	}
	if a.d.Cmp(b.d) <= 0 {
		return a
	}
	q := new(dec.Dec).Quo(a.d, b.d, dec.Scale(0), down)
	return Dec{d: q.Mul(q, b.d).Neg(q).Add(q, a.d)}
}

func (a Dec) Multiplied(b Dec) Dec {
	var d = new(dec.Dec).Mul(a.d, b.d)
	return Dec{d: d.Round(d, Accuracy, halfEven)}
}

func (a Dec) Opposite() Dec {
	return Dec{d: new(dec.Dec).Neg(a.d)}
}

func (a Dec) Plus(b Dec) Dec {
	return Dec{d: new(dec.Dec).Add(a.d, b.d)}
}

func (a Dec) Rounded(decimals int, mode string) Dec {
	var mod dec.Rounder
	switch mode {
	case "Down":
		mod = down
	case "HalfDown":
		mod = halfDown
	case "HalfEven":
		mod = halfEven
	case "HalfUp":
		mod = halfUp
	case "Up":
		mod = up
	default:
		panic("invalid rounding mode")
	}
	return Dec{d: new(dec.Dec).Round(a.d, dec.Scale(decimals), mod)}
}

func (a Dec) String() string {
	return a.d.String()
}

// Digits ritorna il numero di cifre dopo la virgola
func (a Dec) Digits() int {
	var s = a.d.Scale()
	if s <= 0 {
		return 0
	}
	return int(s)
}

// Precision ritorna il numero totale di cifre
func (a Dec) Precision() int {
	if a.IsZero() {
		return 0
	}
	return int(float64(a.d.Unscaled().BitLen())/math.Log2(10)) + 1
}

// func (a Dec) ToAccuracy(Accuracy int) Dec {
// 	return Dec{}
// }

// MarshalText implements encoding.TextMarshaler interface
func (a Dec) MarshalText() ([]byte, error) {
	return []byte(a.d.String()), nil
}

// MarshalJSON implements json.Marshaler interface
func (a Dec) MarshalJSON() ([]byte, error) {
	return []byte(a.d.String()), nil
}

// UnmarshalJSON implements json.Unmarshaler interface
func (a *Dec) UnmarshalJSON(v []byte) error {
	if a.d != nil {
		return errors.New("Already instantiated decimal value")
	}
	if d, ok := str(string(v)); ok {
		a.d = d.d
	} else {
		return errors.New("Value '" + string(v) + "' is not a valid decimal")
	}
	return nil
}

// Value implements driver.Valuer interface
func (a Dec) Value() (interface{}, error) {
	return a.d.String(), nil
}

// Scan implements driver.Scanner interface
func (a *Dec) Scan(src interface{}) error {
	var b Dec
	var ok = true
	switch s := src.(type) {
	case string:
		b, ok = ParseString(s)
	case int:
		b = Int(s)
	case int64:
		b = Int64(s)
	case float64:
		b = Float64(s, int(Accuracy))
	case []byte:
		b, ok = ParseString(string(s))
	default:
		return errors.New("Incompatible type for decimal.Dec")
	}
	if !ok {
		return errors.New("Failed to parse decimal.Dec")
	}
	*a = b
	return nil
}
