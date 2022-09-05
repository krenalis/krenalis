// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2018 Open2b
//

package decimal

import (
	"errors"
	"math/big"
	"strconv"
	"strings"

	"chichi/pkg/code.google.com/p/godec/dec"

	"github.com/open2b/scriggo/native"
)

type Dec struct {
	d *dec.Dec
}

const Accuracy dec.Scale = 28

var zero = &Dec{d: dec.NewDecInt64(0)}

var down = dec.RoundDown
var halfDown = dec.RoundHalfDown
var halfEven = dec.RoundHalfEven
var halfUp = dec.RoundHalfUp
var up = dec.RoundUp

func Int(i int) *Dec {
	return &Dec{d: dec.NewDecInt64(int64(i))}
}

func Int64(i int64) *Dec {
	return &Dec{d: dec.NewDecInt64(i)}
}

func Float64(f float64, accuracy int) *Dec {
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
	return &Dec{d: num}
}

func String(s string) *Dec {
	// rispetto a Open2b::Decimal, non vuole spazi all'inizio e alla fine
	var d, ok = str(s)
	if !ok {
		panic("Value '" + s + "' is not a valid decimal")
	}
	return d
}

func ParseString(s string) (*Dec, bool) {
	return str(strings.TrimSpace(s))
}

func str(s string) (*Dec, bool) {
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
		return &Dec{d: d.SetScale(d.Scale() - exp)}, true
	}
	return &Dec{d: d}, true
}

func Zero() *Dec {
	return zero
}

func (a *Dec) ComparedTo(b *Dec) int {
	return a.d.Cmp(b.d)
}

func (a *Dec) Divided(b *Dec) *Dec {
	if zero.d.Cmp(b.d) == 0 {
		panic("Division by zero")
	}
	return &Dec{d: new(dec.Dec).Quo(a.d, b.d, Accuracy, halfEven)}
}

func (a *Dec) IsNegative() bool {
	return a.d.Cmp(zero.d) < 0
}

func (a *Dec) IsPositive() bool {
	return a.d.Cmp(zero.d) > 0
}

func (a *Dec) IsZero() bool {
	return a.d.Cmp(zero.d) == 0
}

func (a *Dec) JS() native.JS {
	return native.JS(a.String())
}

func (a *Dec) JSON() native.JSON {
	return native.JSON(a.String())
}

func (a *Dec) Minus(b *Dec) *Dec {
	return &Dec{d: new(dec.Dec).Sub(a.d, b.d)}
}

func (a *Dec) Module(b *Dec) *Dec {
	if zero.d.Cmp(b.d) == 0 {
		panic("Division by zero")
	}
	if a.d.Cmp(b.d) <= 0 {
		return a
	}
	q := new(dec.Dec).Quo(a.d, b.d, dec.Scale(0), down)
	return &Dec{d: q.Mul(q, b.d).Neg(q).Add(q, a.d)}
}

func (a *Dec) Multiplied(b *Dec) *Dec {
	var d = new(dec.Dec).Mul(a.d, b.d)
	return &Dec{d: d.Round(d, Accuracy, halfEven)}
}

func (a *Dec) Opposite() *Dec {
	return &Dec{d: new(dec.Dec).Neg(a.d)}
}

func (a *Dec) Plus(b *Dec) *Dec {
	return &Dec{d: new(dec.Dec).Add(a.d, b.d)}
}

func (a *Dec) Rounded(decimals int, mode string) *Dec {
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
	return &Dec{d: new(dec.Dec).Round(a.d, dec.Scale(decimals), mod)}
}

// String ritorna la rappresentazione di a come stringa con sole le cifre decimali significative.
func (a *Dec) String() string {
	s := a.d.String()
	if i := strings.Index(s, "."); i != -1 {
		s = strings.TrimRightFunc(s, func(r rune) bool { return r == '0' })
		if s == "" {
			s = "0"
		} else {
			s = strings.TrimSuffix(s, ".")
		}
	}
	return s
}

// StringN ritorna la rappresentazione del troncamento di a ad n cifre decimali.
// La stringa ritornata avrà n cifre decimali. Deve essere n >= 0.
func (a *Dec) StringN(n int) string {
	if n < 0 {
		panic("decimal: n can not be negative")
	}
	s := a.d.String()
	if i := strings.Index(s, "."); i != -1 {
		t := len(s) - i - 1
		if n == 0 {
			s = s[:i]
		} else if t < n {
			s += strings.Repeat("0", n-t)
		} else if t > n {
			s = s[:i+n+1]
		}
	} else if n > 0 {
		s += "." + strings.Repeat("0", n)
	}
	return s
}

// Digits ritorna il numero di cifre dopo la virgola
func (a Dec) Digits() int {
	var s = a.d.Scale()
	if s <= 0 {
		return 0
	}
	return int(s)
}

// MarshalText implements encoding.TextMarshaler interface
func (a *Dec) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

// MarshalJSON implements json.Marshaler interface
func (a *Dec) MarshalJSON() ([]byte, error) {
	return []byte(a.String()), nil
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
func (a *Dec) Value() (any, error) {
	return a.d.String(), nil
}
