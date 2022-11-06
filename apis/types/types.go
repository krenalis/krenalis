//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"strconv"
)

const (
	MaxDecimalPrecision = 76 // Maximum precision for a Decimal type
	MaxDecimalScale     = 38 // Maximum scale for a Decimal type
)

type BaseType int8

const (
	TBoolean BaseType = 1 + iota
	TInt
	TTinyInt
	TSmallInt
	TMediumInt
	TBigInt
	TUnsignedInt
	TUnsignedTinyInt
	TUnsignedSmallInt
	TUnsignedMediumInt
	TUnsignedBigInt
	TReal
	TDouble
	TDecimal
	TDateTime
	TDate
	TTime
	TYear
	TUUID
	TJSON
	TText
)

var baseName = []string{
	"Boolean",
	"Int",
	"TinyInt",
	"SmallInt",
	"MediumInt",
	"BigInt",
	"UnsignedInt",
	"UnsignedTinyInt",
	"UnsignedSmallInt",
	"UnsignedMediumInt",
	"UnsignedBigInt",
	"Real",
	"Double",
	"Decimal",
	"DateTime",
	"Date",
	"Time",
	"Year",
	"UUID",
	"JSON",
	"Text",
}

// String returns the name of b. Panics if b is not a base type.
func (b BaseType) String() string {
	if b < 1 || int(b) > len(baseName) {
		panic("invalid base type")
	}
	return baseName[b-1]
}

// Kind represents a kind.
type Kind int8

const (
	PersonFirstName Kind = 1 + iota
	PersonMiddleName
	PersonLastName
	PersonFullName
	PersonEmail
	PersonLanguage
	PersonTimeZone
	PersonBirthDate

	LocationAddress
	LocationStreet1
	LocationStreet2
	LocationStreet3
	LocationCity
	LocationPostalCode
	LocationStateProv
	LocationCountry
)

var kindName = []string{
	// Person
	"Person.FirstName",
	"Person.MiddleName",
	"Person.LastName",
	"Person.FullName",
	"Person.Email",
	"Person.Language",
	"Person.TimeZone",
	"Person.BirthDate",
	// Location
	"Location.Address",
	"Location.Street1",
	"Location.Street2",
	"Location.Street3",
	"Location.City",
	"Location.PostalCode",
	"Location.StateProv",
	"Location.Country",
}

// String returns the name of k. Panics if k is not a kind.
func (k Kind) String() string {
	if k < 1 || int(k) > len(kindName) {
		panic("invalid kind")
	}
	return kindName[k-1]
}

// Type represents a type.
type Type struct {
	b BaseType
	k Kind
	p int // precision of a Decimal type or length in bytes of a Text type.
	s int // scale of a Decimal type or length in characters of a Text type.
}

// Boolean returns the Boolean type.
func Boolean() Type {
	return Type{b: TBoolean}
}

// Int returns the Int type.
func Int() Type {
	return Type{b: TInt}
}

// TinyInt returns the TinyInt type.
func TinyInt() Type {
	return Type{b: TTinyInt}
}

// SmallInt returns the SmallInt type.
func SmallInt() Type {
	return Type{b: TSmallInt}
}

// MediumInt returns the MediumInt type.
func MediumInt() Type {
	return Type{b: TMediumInt}
}

// BigInt returns the BigInt type.
func BigInt() Type {
	return Type{b: TBigInt}
}

// UnsignedInt returns the UnsignedInt type.
func UnsignedInt() Type {
	return Type{b: TUnsignedInt}
}

// UnsignedTinyInt returns the UnsignedTinyInt type.
func UnsignedTinyInt() Type {
	return Type{b: TUnsignedTinyInt}
}

// UnsignedSmallInt returns the UnsignedSmallInt type.
func UnsignedSmallInt() Type {
	return Type{b: TUnsignedSmallInt}
}

// UnsignedMediumInt returns the UnsignedMediumInt type.
func UnsignedMediumInt() Type {
	return Type{b: TUnsignedMediumInt}
}

// UnsignedBigInt returns the UnsignedBigInt type.
func UnsignedBigInt() Type {
	return Type{b: TUnsignedBigInt}
}

// Real returns the Real type.
func Real() Type {
	return Type{b: TReal}
}

// Double returns the Double type.
func Double() Type {
	return Type{b: TDouble}
}

// Decimal returns the Decimal type with the given precision and scale.
// Panics if precision is not in range [1,MaxDecimalPrecision] and if scale is
// not in range [0,MaxDecimalScale] and if it is greater that precision.
// As a special case if both precision and scale are zero, the type has no
// precision and scale.
func Decimal(precision, scale int) Type {
	if precision == 0 && scale == 0 {
		return Type{b: TDecimal}
	}
	if precision < 1 || precision > MaxDecimalPrecision {
		panic("invalid decimal precision")
	}
	if scale < 0 || scale > MaxDecimalScale || scale > precision {
		panic("invalid decimal scale")
	}
	return Type{b: TDecimal, p: precision, s: scale}
}

// DateTime returns the DateTime type.
func DateTime() Type {
	return Type{b: TDateTime}
}

// Date returns the Date type.
func Date() Type {
	return Type{b: TDate}
}

// Time returns the Time type.
func Time() Type {
	return Type{b: TTime}
}

// Year returns the Year type.
func Year() Type {
	return Type{b: TYear}
}

// UUID returns the UUID type.
func UUID() Type {
	return Type{b: TUUID}
}

// JSON returns the JSON type.
func JSON() Type {
	return Type{b: TJSON}
}

// Text returns the Text type with the given lengths.
// Panics if a length is not greater than zero and panics if there is more than
// one length in bytes or more than one length in characters.
func Text(lengths ...Length) Type {
	t := Type{b: TText}
	for _, length := range lengths {
		switch l := length.(type) {
		case Bytes:
			if t.p > 0 {
				panic("repeated length in bytes")
			}
			if l <= 0 {
				panic("invalid text length")
			}
			t.p = int(l)
		case Chars:
			if t.s > 0 {
				panic("repeated length in characters")
			}
			if l <= 0 {
				panic("invalid text length")
			}
			t.s = int(l)
		}
	}
	return t
}

// MarshalJSON marshals t into JSON.
func (t Type) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.Base().String() + `"`), nil
}

// WithKind returns the type t but with kind k.
// Panics if k is not a kind.
func (t Type) WithKind(k Kind) Type {
	if k < 1 || int(k) > len(kindName) {
		panic("invalid kind")
	}
	t.k = k
	return t
}

// Base returns the base type of t.
func (t Type) Base() BaseType {
	return t.b
}

// Kind returns the kind of t and true.
// If t has not a kind, it returns false.
func (t Type) Kind() (Kind, bool) {
	return t.k, t.k > 0
}

// ByteLen returns the maximum length in bytes of a Text type.
// Returns 0 if there is no limit in bytes.
// Panics if t is not a Text type.
func (t Type) ByteLen() int {
	if t.b != TText {
		panic("cannot get byte length of a non-text type")
	}
	return t.p
}

// CharLen returns the maximum length in characters of a Text type.
// Returns 0 if there is no limit in characters.
// Panics if t is not a Text type.
func (t Type) CharLen() int {
	if t.b != TText {
		panic("cannot get character length of a non-text type")
	}
	return t.s
}

// Precision returns the precision of a Decimal type.
// Panics if t is not a Decimal type.
func (t Type) Precision() int {
	if t.b != TDecimal {
		panic("cannot get precision of a non-decimal type")
	}
	return t.p
}

// Scale returns the scale of a Decimal type.
// Panics if t is not a Decimal type.
func (t Type) Scale() int {
	if t.b != TDecimal {
		panic("cannot get scale of a non-decimal type")
	}
	return t.s
}

// String returns a string representation of t.
// Panics if t is not a valid type.
func (t Type) String() string {
	s := t.b.String()
	switch t.b {
	case TDecimal:
		if t.p > 0 {
			s += "(" + strconv.Itoa(t.p) + "," + strconv.Itoa(t.s) + ")"
		}
	case TText:
		if t.p > 0 || t.s > 0 {
			s += "("
			if t.p > 0 {
				s += strconv.Itoa(t.p) + " bytes"
			}
			if t.s > 0 {
				if t.p > 0 {
					s += ","
				}
				s += strconv.Itoa(t.s) + " chars"
			}
			s += ")"
		}
	}
	if t.k > 0 {
		s += " [" + t.k.String() + "]"
	}
	return s
}

// Length represents a Text length.
type Length interface {
	length() int
}

// Chars represents a length in characters.
type Chars int

func (l Chars) length() int { return int(l) }

// Bytes represents a length in bytes.
type Bytes int

func (l Bytes) length() int { return int(l) }
