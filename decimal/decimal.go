//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

// Package decimal provides a decimal floating-point number type with arbitrary
// precision. The [Decimal] type represents a decimal number, with a zero value
// equivalent to 0:
//
//	var x Decimal  // x is the decimal 0
//	y := Decimal{} // y is the decimal 0
//
// [Decimal] values can be created using factory functions:
//
//	New(31, 5)                // creates the decimal 0.00031
//	Int(-23, 2, 0)            // creates the decimal -23
//	Uint(540, 3, 0)           // creates the decimal 540
//	Float64(690.366, 5, 3)    // creates the decimal 690.366
//	Parse("737.012e2", 10, 3) // creates the decimal 73701.2
//
// If the decimal’s mantissa fits into a uint64, no heap allocation occurs.
// Precision defines the total number of significant digits, while scale
// specifies the number of digits after the decimal point. Precision must be in
// the range [1, MaxPrecision], and scale must be within [MinScale, MaxScale].
//
// The [Int], [Uint], and [Parse] functions ensure the decimal fits within the
// specified precision and scale, returning an error otherwise. The [New] function
// accepts a mantissa and scale to create the decimal. [Float64] rounds the given
// float to the specified scale and returns an error if it exceeds the provided
// precision.
//
// Each factory function, except [New], has a corresponding Must variant. These
// Must functions do not impose any additional precision and scale limits beyond
// the absolute maximum and minimum values indicated earlier.
//
// When using the [Parse] function, the original string representation of the
// decimal is retained and returned by the [Decimal.String] method, avoiding
// unnecessary string allocations.
//
// Once created, [Decimal] values are immutable, allowing their methods to be
// safely called concurrently by multiple goroutines.
package decimal

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"strings"

	"github.com/ericlagergren/decimal"
)

const (
	MaxPrecision = 999_999_999_999_999_999
	MinPrecision = -MaxPrecision
	MaxScale     = 999_999_999_999_999_999
	MinScale     = -MaxScale
)

var ErrSyntax = errors.New("syntax error")
var ErrOutOfRange = errors.New("out of range")

// Decimal represents a decimal number.
// The zero value for a Decimal represents the value 0.
type Decimal struct {
	s string
	b decimal.Big
}

// Append appends x to buf and returns the extended buffer.
func (x Decimal) Append(buf []byte) []byte {
	s := x.s
	if s == "" {
		s = x.b.String() // TODO(marco): Optimize by avoiding string allocations.
	}
	return append(buf, s...)
}

// Cmp compares x and y and returns:
//
//   - -1 if x < y;
//   - 0  if x == y;
//   - +1 if x > y.
func (x Decimal) Cmp(y Decimal) int {
	return x.b.Cmp(&y.b)
}

// Equal reports whether x is equal to y.
func (x Decimal) Equal(y Decimal) bool {
	return x.b.Cmp(&y.b) == 0
}

// Float64 returns the float64 value closest to x and a bool indicating whether
// the float64 represents exactly x.
func (x Decimal) Float64() (float64, bool) {
	return x.b.Float64()
}

// Format implements the fmt.Formatter interface.
// See https://pkg.go.dev/github.com/ericlagergren/decimal#Big.Format.
func (x Decimal) Format(s fmt.State, c rune) {
	x.b.Format(s, c)
}

// Greater reports whether x is greater than y.
func (x Decimal) Greater(y Decimal) bool {
	return x.b.Cmp(&y.b) > 0
}

// GreaterEqual reports whether x is greater than or equal to y.
func (x Decimal) GreaterEqual(y Decimal) bool {
	return x.b.Cmp(&y.b) >= 0
}

// Int64 converts x to its int64 representation.
// It returns 0 and [ErrOutOfRange] if x cannot be represented as an int64.
func (x Decimal) Int64() (int64, error) {
	if !x.b.IsInt() {
		return 0, ErrOutOfRange
	}
	i, ok := x.b.Int64()
	if !ok {
		return 0, ErrOutOfRange
	}
	return i, nil
}

// Less reports whether x is less than y.
func (x Decimal) Less(y Decimal) bool {
	return x.b.Cmp(&y.b) < 0
}

// LessEqual reports whether x is less than or equal to y.
func (x Decimal) LessEqual(y Decimal) bool {
	return x.b.Cmp(&y.b) <= 0
}

// MarshalJSON returns x as the JSON encoding of x.
func (x Decimal) MarshalJSON() ([]byte, error) {
	return x.b.MarshalText()
}

// Neg returns the negation of x.
func (x Decimal) Neg() Decimal {
	sign := x.b.Sign()
	if sign == 0 {
		return Decimal{}
	}
	y := Decimal{}
	y.b.Copy(&x.b)
	y.b.SetSignbit(sign > 0)
	if sign < 0 && x.s != "" {
		y.s = x.s[1:]
	}
	return y
}

// Sign returns:
//
//	-1 if x <  0
//	 0 if x == 0
//	+1 if x >  0
func (x Decimal) Sign() int {
	return x.b.Sign()
}

// String returns the string representation of x.
//
// Use fmt.Sprintf(f, x) to get a specific format. String does not guarantee
// a specific format for the returned value but does not allocate a new string
// if x was created using the [Parse] function.
func (x Decimal) String() string {
	if x.s != "" {
		return x.s
	}
	if x.b.Sign() == 0 {
		return "0"
	}
	return fmt.Sprintf("%s", &x.b)
}

// Uint64 converts x to its uint64 representation.
// It returns 0 and [ErrOutOfRange] if x cannot be represented as an uint64.
func (x Decimal) Uint64() (uint64, error) {
	if !x.b.IsInt() {
		return 0, ErrOutOfRange
	}
	i, ok := x.b.Uint64()
	if !ok {
		return 0, ErrOutOfRange
	}
	return i, nil
}

// Value implements the driver.Valuer interface.
//
// If x fits in an int64, it returns an int64; otherwise, it returns a string
// formatted as -dddd.dd.
func (x Decimal) Value() (driver.Value, error) {
	if x.b.IsInt() {
		if i, ok := x.b.Int64(); ok {
			return i, nil
		}
	}
	return fmt.Sprintf("%f", &x.b), nil
}

var zero = []byte("0")

// WriteTo implements the io.WriteTo interface.
func (x Decimal) WriteTo(w io.Writer) (int64, error) {
	if x.s != "" {
		if w, ok := w.(io.StringWriter); ok {
			n, err := w.WriteString(x.s)
			return int64(n), err
		}
	}
	if x.b.Sign() == 0 {
		n, err := w.Write(zero)
		return int64(n), err
	}
	state := fmtState{w: w}
	x.b.Format(&state, 's') // TODO(marco): Optimize and test to ensure that WriteTo does not allocate.
	return state.result()
}

// Float64 returns the decimal represented by f, rounded down scale; where
// precision > 0 and scale <= precision. If the resulting decimal exceeds
// precision, or f is NaN, +Inf, -Inf, it returns 0 and [ErrOutOfRange].
func Float64(f float64, precision, scale int) (Decimal, error) {
	if err := validPrecisionScale(precision, scale); err != nil {
		return Decimal{}, err
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return Decimal{}, ErrOutOfRange
	}
	if f == 0 {
		return Decimal{}, nil
	}
	x := Decimal{}
	x.b.SetFloat64(f)
	s := x.b.Precision() - x.b.Scale() + scale
	x.b.Round(s)
	if x.b.Precision() > precision {
		return Decimal{}, ErrOutOfRange
	}
	return x, nil
}

// Int returns the decimal represented by i. It returns [ErrOutOfRange] if the
// decimal exceed the specified precision or scale, where precision > 0
// and scale <= precision.
func Int(i, precision, scale int) (Decimal, error) {
	if err := validPrecisionScale(precision, scale); err != nil {
		return Decimal{}, err
	}
	if i == 0 {
		return Decimal{}, nil
	}
	n := Decimal{}
	n.b.SetMantScale(int64(i), 0)
	if n.b.Precision() > precision-scale {
		return Decimal{}, ErrOutOfRange
	}
	return n, nil
}

// MustInt returns the decimal represented by i.
func MustInt(i int) Decimal {
	x := Decimal{}
	x.b.SetMantScale(int64(i), 0)
	return x
}

// MustParse parses n, which must contain a text representation of a decimal
// number. It panics if n is not syntactically correct or if the number is out
// of range for the [Decimal] type. See the [Parse] function for valid formats.
func MustParse[T ~string | ~[]byte](n T) Decimal {
	x, err := Parse(n, 0, 0)
	if err != nil {
		panic(err.Error())
	}
	return x
}

// MustUint returns the decimal represented by i.
func MustUint(i uint) Decimal {
	x := Decimal{}
	x.b.SetUint64(uint64(i))
	return x
}

// New returns a [Decimal] value with the given mantissa and scale.
//
// If scale is positive, the decimal will represent a number less than 1
// (shifting the decimal point to the left). If scale is negative, the decimal
// will represent a number greater than 1 (shifting the decimal point to the
// right by the absolute value of the scale). For example, New(5, scale) is
//
//	   0.005 if scale == 3
//	   5     if scale == 0
//	5000     if scale == -3
//
// It panics if scale is not valid.
func New(mantissa int64, scale int) Decimal {
	if scale < MinScale || scale > MaxScale {
		panic(ErrOutOfRange.Error())
	}
	x := Decimal{}
	x.b.SetMantScale(mantissa, scale)
	return x
}

// Parse parses n, which must contain a text representation of a decimal number
// that does not exceed the provided precision or scale, where precision > 0
// and scale <= precision.
//
// If n is not syntactically correct, it returns the [ErrSyntax] error, and if it
// is out of range, it returns the [ErrOutOfRange] error.
//
// As a special case, if precision is 0, the precision and scale are ignored,
// and no limits are enforced on the number of significant digits or the scale
// of the number.
//
// The following formats are valid:
//
//	0
//	0.
//	.5
//	123
//	-123
//	123.
//	+123.00
//	123.4560
//	0.00
//	0.4560
//	123e+4
//	123.4560E0
//	123.4560e7
//	-123.4560E+7
//	123.4560e-7
func Parse[T ~string | ~[]byte](n T, precision, scale int) (Decimal, error) {
	switch {
	case
		precision != 0 && (precision < MinPrecision || precision > MaxPrecision),
		scale < MinScale || scale > MaxScale,
		scale > precision:
		return Decimal{}, ErrOutOfRange
	}
	if len(n) == 0 {
		return Decimal{}, ErrSyntax
	}
	str := n
	switch c := n[0]; c {
	case '+':
		str = str[1:]
		fallthrough
	case '-':
		n = n[1:]
	}
	dot := 0 // dot position relative to n[0]; it can be negative, and is 0 if there is no dot.
	switch n[0] {
	case '0':
		if len(n) == 1 {
			return Decimal{}, nil
		}
		switch n[1] {
		case '.':
			dot = -1
			n = n[2:]
		case 'e', 'E':
			n = n[1:]
		default:
			return Decimal{}, ErrSyntax
		}
	case '.':
		dot = -1
		n = n[1:]
	}
	zeros := 0
	mantissa := mantissa{}
	var i int
	for i = 0; i < len(n); i++ {
		c := n[i]
		if c == '0' {
			zeros++
			continue
		}
		if '1' <= c && c <= '9' || c == '.' && dot == 0 {
			if zeros > 0 {
				if mantissa.digits() > 0 {
					for range zeros {
						mantissa.add('0')
					}
				}
				zeros = 0
			}
			if c == '.' {
				dot = i
			} else {
				mantissa.add(c)
			}
			continue
		}
		if c == 'e' || c == 'E' {
			break
		}
		return Decimal{}, ErrSyntax
	}
	var s int // number of significant digits after the decimal point.
	if dot == 0 {
		s = -zeros
	} else {
		s = max(0, i-(dot+1)-zeros)
	}
	if i < len(n) {
		// Parse the exponent.
		start := i + len(str) - len(n)
		i += 1
		if i == len(n) {
			return Decimal{}, ErrSyntax
		}
		neg := false
		switch n[i] {
		case '-':
			neg = true
			fallthrough
		case '+':
			i++
			if i == len(n) {
				return Decimal{}, ErrSyntax
			}
		}
		prev := s
		factor := 1
		for j := len(n) - 1; j >= i; j-- {
			c := n[j]
			if c < '0' || c > '9' {
				return Decimal{}, ErrSyntax
			}
			if c == '0' {
				factor *= 10
				continue
			}
			d := int(c-'0') * factor
			if d < 0 {
				return Decimal{}, ErrOutOfRange
			}
			s2 := s
			if neg {
				s += d
				if s < s2 {
					return Decimal{}, ErrOutOfRange
				}
			} else {
				s -= d
				if s > s2 {
					return Decimal{}, ErrOutOfRange
				}
			}
			factor *= 10
		}
		if s < MinScale || s > MaxScale {
			return Decimal{}, ErrOutOfRange
		}
		if s == prev {
			str = str[:start]
		} else {
			zeros = 0
		}
	}
	if mantissa.isZero() {
		return Decimal{}, nil
	}
	if precision > 0 {
		p := max(0, mantissa.digits()-s) + scale
		if p > precision || s > scale {
			return Decimal{}, ErrOutOfRange
		}
	}
	x := Decimal{}
	mantissa.set(&x.b, s)
	if str[0] == '-' {
		x.b.SetSignbit(true)
	}
	if reflect.TypeFor[T]().Kind() == reflect.String {
		s := string(str)
		if s[len(s)-zeros-1] == '0' {
			zeros--
		}
		if zeros > 0 {
			s = s[:len(s)-zeros]
		}
		x.s = strings.TrimSuffix(s, ".")

	}
	return x, nil
}

var one = big.NewInt(1)
var base = big.NewInt(10)

// Range returns the minimum and maximum decimal values based on the provided
// precision and scale, where precision > 0 and scale <= precision.
// If either precision or scale are outside these ranges, it panics.
func Range(precision, scale int) (Decimal, Decimal) {
	if err := validPrecisionScale(precision, scale); err != nil {
		panic(err.Error())
	}
	y := Decimal{}
	if precision < 20 {
		m := uint64(math.Pow(10, float64(precision))) - 1
		y.b.SetUint64(m)
		y.b.SetScale(scale)
	} else {
		m := new(big.Int).Exp(base, big.NewInt(int64(precision)), nil)
		m.Sub(m, one)
		y.b.SetBigMantScale(m, scale)
	}
	x := Decimal{}
	x.b.Copy(&y.b)
	x.b.SetSignbit(true)
	return x, y
}

// Uint returns the decimal represented by i, and true. It returns zero and
// false, if the decimal exceed the specified precision or scale, where
// precision > 0 and scale <= precision.
func Uint(i uint, precision, scale int) (Decimal, error) {
	if err := validPrecisionScale(precision, scale); err != nil {
		return Decimal{}, err
	}
	if i == 0 {
		return Decimal{}, nil
	}
	n := Decimal{}
	n.b.SetUint64(uint64(i))
	if n.b.Precision() > precision-scale {
		return Decimal{}, ErrOutOfRange
	}
	return n, nil
}

// mantissa represents the mantissa of a decimal.
// It is primarily used in the Parse function.
type mantissa struct {
	B big.Int
	I uint64
	v big.Int // value to add
	f big.Int // factor, is 10 when B is not 0
	d int     // number of decimal digits
}

// add adds a digit to the mantissa.
func (m *mantissa) add(c uint8) {
	m.d++
	d := uint64(c - '0')
	if m.B.Sign() == 0 {
		if m.I <= (math.MaxUint64-d)/10 {
			m.I = m.I*10 + d
			return
		}
		m.B.SetUint64(m.I)
		m.f.SetInt64(10)
		m.I = 0
	}
	m.B.Mul(&m.B, &m.f)
	if d == 0 {
		return
	}
	m.v.SetUint64(d)
	m.B.Add(&m.B, &m.v)
}

// digits returns the number of digit of the mantissa.
func (m *mantissa) digits() int {
	return m.d
}

// isZero reports whether the mantissa is zero.
func (m *mantissa) isZero() bool {
	return m.I == 0 && m.B.Sign() == 0
}

// set sets big with mantissa m and scale s.
func (m *mantissa) set(big *decimal.Big, s int) {
	if m.B.Sign() == 0 {
		big.SetUint64(m.I)
		big.SetScale(s)
		return
	}
	big.SetBigMantScale(&m.B, s)
}

// fmtState implements the fmt.State interface.
type fmtState struct {
	w   io.Writer
	n   int64
	err error
}

func (s *fmtState) result() (n int64, err error) {
	return s.n, s.err
}

func (s *fmtState) Write(b []byte) (n int, err error) {
	if s.err != nil {
		return 0, err
	}
	n, err = s.w.Write(b)
	s.n += int64(n)
	s.err = err
	return
}

func (s *fmtState) Width() (int, bool) {
	return 0, false
}

func (s *fmtState) Precision() (int, bool) {
	return 0, false
}

func (s *fmtState) Flag(int) bool {
	return false
}

// validPrecisionScale reports whether precision and scale are valid.
func validPrecisionScale(precision, scale int) error {
	if precision < MinPrecision || precision > MaxPrecision {
		return ErrOutOfRange
	}
	if scale < MinScale || scale > MaxScale {
		return ErrOutOfRange
	}
	if precision < scale {
		return ErrOutOfRange
	}
	return nil
}
