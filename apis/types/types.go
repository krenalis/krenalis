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

type PhysicalType int8

const (
	PtBoolean PhysicalType = 1 + iota
	PtInt
	PtInt8
	PtInt16
	PtInt24
	PtInt64
	PtUInt
	PtUInt8
	PtUInt16
	PtUInt24
	PtUInt64
	PtFloat
	PtFloat32
	PtDecimal
	PtDateTime
	PtDate
	PtTime
	PtYear
	PtUUID
	PtJSON
	PtText
)

var physicalName = []string{
	"Boolean",
	"Int",
	"Int8",
	"Int16",
	"Int24",
	"Int64",
	"UInt",
	"UInt8",
	"UInt16",
	"UInt24",
	"UInt64",
	"Float",
	"Float32",
	"Decimal",
	"DateTime",
	"Date",
	"Time",
	"Year",
	"UUID",
	"JSON",
	"Text",
}

// String returns the name of pt. Panics if pt is not a physical type.
func (pt PhysicalType) String() string {
	if pt < 1 || int(pt) > len(physicalName) {
		panic("invalid physical type")
	}
	return physicalName[pt-1]
}

// LogicalType represents a logical type.
type LogicalType int8

const (
	LtPersonFirstName LogicalType = 1 + iota
	LtPersonMiddleName
	LtPersonLastName
	LtPersonFullName
	LtPersonEmail
	LtPersonLanguage
	LtPersonTimeZone
	LtPersonBirthDate

	LtLocationAddress
	LtLocationStreet1
	LtLocationStreet2
	LtLocationStreet3
	LtLocationCity
	LtLocationPostalCode
	LtLocationStateProv
	LtLocationCountry
)

var logicalName = []string{
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

// String returns the name of lt. Panics if lt is not a logical type.
func (lt LogicalType) String() string {
	if lt < 1 || int(lt) > len(logicalName) {
		panic("invalid logical type")
	}
	return logicalName[lt-1]
}

// Type represents a type.
type Type struct {
	pt PhysicalType
	lt LogicalType
	p  int // precision of a Decimal type or length in bytes of a Text type.
	s  int // scale of a Decimal type or length in characters of a Text type.
}

// Boolean returns the Boolean type.
func Boolean() Type {
	return Type{pt: PtBoolean}
}

// Int returns the Int type.
func Int() Type {
	return Type{pt: PtInt}
}

// Int8 returns the Int8 type.
func Int8() Type {
	return Type{pt: PtInt8}
}

// Int16 returns the Int16 type.
func Int16() Type {
	return Type{pt: PtInt16}
}

// Int24 returns the Int24 type.
func Int24() Type {
	return Type{pt: PtInt24}
}

// Int64 returns the Int64 type.
func Int64() Type {
	return Type{pt: PtInt64}
}

// UInt returns the UInt type.
func UInt() Type {
	return Type{pt: PtUInt}
}

// UInt8 returns the UInt8 type.
func UInt8() Type {
	return Type{pt: PtUInt8}
}

// UInt16 returns the UInt16 type.
func UInt16() Type {
	return Type{pt: PtUInt16}
}

// UInt24 returns the UInt24 type.
func UInt24() Type {
	return Type{pt: PtUInt24}
}

// UInt64 returns the UInt64 type.
func UInt64() Type {
	return Type{pt: PtUInt64}
}

// Float returns the Float type.
func Float() Type {
	return Type{pt: PtFloat}
}

// Float32 returns the Float32 type.
func Float32() Type {
	return Type{pt: PtFloat32}
}

// Decimal returns the Decimal type with the given precision and scale.
// Panics if precision is not in range [1,MaxDecimalPrecision] and if scale is
// not in range [0,MaxDecimalScale] and if it is greater that precision.
// As a special case if both precision and scale are zero, the type has no
// precision and scale.
func Decimal(precision, scale int) Type {
	if precision == 0 && scale == 0 {
		return Type{pt: PtDecimal}
	}
	if precision < 1 || precision > MaxDecimalPrecision {
		panic("invalid decimal precision")
	}
	if scale < 0 || scale > MaxDecimalScale || scale > precision {
		panic("invalid decimal scale")
	}
	return Type{pt: PtDecimal, p: precision, s: scale}
}

// DateTime returns the DateTime type.
func DateTime() Type {
	return Type{pt: PtDateTime}
}

// Date returns the Date type.
func Date() Type {
	return Type{pt: PtDate}
}

// Time returns the Time type.
func Time() Type {
	return Type{pt: PtTime}
}

// Year returns the Year type.
func Year() Type {
	return Type{pt: PtYear}
}

// UUID returns the UUID type.
func UUID() Type {
	return Type{pt: PtUUID}
}

// JSON returns the JSON type.
func JSON() Type {
	return Type{pt: PtJSON}
}

// Text returns the Text type with the given lengths.
// Panics if a length is not greater than zero and panics if there is more than
// one length in bytes or more than one length in characters.
func Text(lengths ...Length) Type {
	t := Type{pt: PtText}
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
	return []byte(`"` + t.PhysicalType().String() + `"`), nil
}

// WithLogicalType returns the type t but with the logical type lt.
// Panics if lt is not a logical type.
func (t Type) WithLogicalType(lt LogicalType) Type {
	if lt < 1 || int(lt) > len(logicalName) {
		panic("invalid logical type")
	}
	t.lt = lt
	return t
}

// PhysicalType returns the physical type of t.
func (t Type) PhysicalType() PhysicalType {
	return t.pt
}

// LogicalType returns the logical type of t and true.
// If t has no logical type, it returns false.
func (t Type) LogicalType() (LogicalType, bool) {
	return t.lt, t.lt > 0
}

// ByteLen returns the maximum length in bytes of a Text type and true.
// If t has no maximum length in bytes, it returns 0 and false.
// Panics if t is not a Text type.
func (t Type) ByteLen() (int, bool) {
	if t.pt != PtText {
		panic("cannot get byte length of a non-text type")
	}
	return t.p, t.p > 0
}

// CharLen returns the maximum length in characters of a Text type and true.
// If t has no maximum length in characters, it returns 0 and false.
// Panics if t is not a Text type.
func (t Type) CharLen() (int, bool) {
	if t.pt != PtText {
		panic("cannot get character length of a non-text type")
	}
	return t.s, t.s > 0
}

// Precision returns the precision of a Decimal type.
// Panics if t is not a Decimal type.
func (t Type) Precision() int {
	if t.pt != PtDecimal {
		panic("cannot get precision of a non-decimal type")
	}
	return t.p
}

// Scale returns the scale of a Decimal type.
// Panics if t is not a Decimal type.
func (t Type) Scale() int {
	if t.pt != PtDecimal {
		panic("cannot get scale of a non-decimal type")
	}
	return t.s
}

// String returns a string representation of t.
// Panics if t is not a valid type.
func (t Type) String() string {
	s := t.pt.String()
	switch t.pt {
	case PtDecimal:
		if t.p > 0 {
			s += "(" + strconv.Itoa(t.p) + "," + strconv.Itoa(t.s) + ")"
		}
	case PtText:
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
	if t.lt > 0 {
		s += " [" + t.lt.String() + "]"
	}
	return s
}

// Valid indicates if t is valid.
func (t Type) Valid() bool {
	return t.pt != 0
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
