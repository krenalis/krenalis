//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"fmt"
	"math"
	"regexp"
	"strconv"

	"github.com/shopspring/decimal"
)

// one is the decimal.Decimal 1.
var one = decimal.New(1, 0)

var (
	minInt  = [...]int64{MinInt, MinInt8, MinInt16, MinInt24, MinInt64}
	maxInt  = [...]int64{MaxInt, MaxInt8, MaxInt16, MaxInt24, MaxInt64}
	maxUInt = [...]uint64{MaxUInt, MaxUInt8, MaxUInt16, MaxUInt24, MaxUInt64}
)

var (
	MaxDecimal = decimal.New(1, MaxDecimalPrecision).Sub(one)
	MinDecimal = MaxDecimal.Neg()
)

// Time layouts.
const (
	Nanoseconds  = "ns"
	Microseconds = "us"
	Milliseconds = "ms"
	Seconds      = "s"
)

const (
	MaxArrayLen         = MaxInt24 // Maximum length of an Array type
	MaxDecimalPrecision = 76       // Maximum precision for a Decimal type
	MaxDecimalScale     = 38       // Maximum scale for a Decimal type
	MaxTextLen          = MaxInt   // Maximum length in bytes and characters for a Text type
	MaxYear             = 9999     // Maximum year for DataTime, Date and Year types
	MinYear             = 1        // Minimum year for DataTime, Date and Year types

	MaxInt    = math.MaxInt32
	MaxInt16  = math.MaxInt16
	MaxInt24  = 1<<23 - 1
	MaxInt64  = math.MaxInt64
	MaxInt8   = math.MaxInt8
	MaxUInt   = math.MaxUint
	MaxUInt16 = math.MaxUint16
	MaxUInt24 = 1<<24 - 1
	MaxUInt64 = math.MaxUint64
	MaxUInt8  = math.MaxUint8
	MinInt    = math.MinInt32
	MinInt16  = math.MinInt16
	MinInt24  = -1 << 23
	MinInt64  = math.MinInt64
	MinInt8   = math.MinInt8

	MaxFloat   = math.MaxFloat64
	MaxFloat32 = math.MaxFloat32
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
	PtArray
	PtObject
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
	"Array",
	"Object",
}

// String returns the name of pt. Panics if pt is not a physical type.
func (pt PhysicalType) String() string {
	if !pt.Valid() {
		panic("invalid physical type")
	}
	return physicalName[pt-1]
}

// Valid reports whether pt is a valid physical type.
func (pt PhysicalType) Valid() bool {
	return 1 <= pt && int(pt) <= len(physicalName)
}

// PhysicalTypeByName returns a physical type by its name. The second return
// parameter reports whether a physical type with the given name exists.
func PhysicalTypeByName(name string) (PhysicalType, bool) {
	for i, n := range physicalName {
		if n == name {
			return PhysicalType(i + 1), true
		}
	}
	return 0, false
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
	if !lt.Valid() {
		panic("invalid logical type")
	}
	return logicalName[lt-1]
}

// Valid reports whether lt is a valid logical type.
func (lt LogicalType) Valid() bool {
	return 1 <= lt && int(lt) <= len(logicalName)
}

// LogicalTypeByName returns a logical type by its name. The second return
// parameter reports whether a logical type with the given name exists.
func LogicalTypeByName(name string) (LogicalType, bool) {
	for i, n := range logicalName {
		if n == name {
			return LogicalType(i + 1), true
		}
	}
	return 0, false
}

// Role represents the role of a property.
type Role int

const (
	BothRole        Role = iota // both
	SourceRole                  // source
	DestinationRole             // destination
)

// String returns the string representation of role.
// It panics if role is not a valid Role value.
func (role Role) String() string {
	switch role {
	case BothRole:
		return "Both"
	case SourceRole:
		return "Source"
	case DestinationRole:
		return "Destination"
	}
	panic("invalid role")
}

// Property represents an object property.
type Property struct {
	Name        string
	Label       string
	Description string
	Role        Role
	Type        Type
}

// Type represents a type.
type Type struct {
	pt PhysicalType
	lt LogicalType

	// u reports whether the items of an Array must be unique.
	unique bool

	// p represents
	//   - precision of a Decimal type
	//   - length in bytes of a Text type
	//   - minimum length of an Array type
	p int32

	// s represents
	//   - scale of a Decimal type
	//   - length in characters of a Text type
	//   - maximum length of an Array type
	s int32

	// vl can contain one of
	//   - intRange value for Int, Int8, Int16, Int24 and Int64
	//   - uintRange value for UInt, UInt8, UInt16, UInt24 and UInt64
	//   - floatRange value for Float and Float32
	//   - decimalRange value for Decimal
	//   - string value representing a layout for DateTime, Date and Time
	//   - *regexp.Regexp value for Text
	//   - []string with the enum values for Text
	//   - []Property for Object
	//   - Type of the items for Array
	vl any

	// custom type. Empty for non-custom types.
	custom string
}

// Array returns an Array type with items of type t.
func Array(t Type) Type {
	return Type{pt: PtArray, s: MaxArrayLen, vl: t}
}

// Object returns an Object type with the given properties.
// Panics if properties is empty, or if a property name is empty or repeated.
func Object(properties []Property) Type {
	if len(properties) == 0 {
		panic("no property in object")
	}
	pr := make([]Property, len(properties))
	for i, property := range properties {
		if property.Name == "" {
			panic("empty property name")
		}
		for _, p := range pr[:i] {
			if property.Name == p.Name {
				panic("property name is repeated")
			}
		}
		pr[i] = property
	}
	return Type{pt: PtObject, vl: pr}
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
// Panics if precision is not in range [1,MaxDecimalPrecision] or if scale is
// not in range [0,MaxDecimalScale] or if it is greater that precision.
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
	return Type{pt: PtDecimal, p: int32(precision), s: int32(scale)}
}

// DateTime returns the DateTime type with the given layout.
func DateTime(layout string) Type {
	return Type{pt: PtDateTime, vl: layout}
}

// Date returns the Date type with the given layout.
func Date(layout string) Type {
	return Type{pt: PtDate, vl: layout}
}

// Time returns the Time type with the given layout.
func Time(layout string) Type {
	return Type{pt: PtTime, vl: layout}
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
			if l <= 0 || l > MaxTextLen {
				panic("invalid text length")
			}
			t.p = int32(l)
		case Chars:
			if t.s > 0 {
				panic("repeated length in characters")
			}
			if l <= 0 || l > MaxTextLen {
				panic("invalid text length")
			}
			t.s = int32(l)
		}
	}
	return t
}

// AsCustom returns t as a custom type called name.
// Panics if t is not valid, or t is already a custom type or name is empty.
func (t Type) AsCustom(name string) Type {
	if !t.Valid() {
		panic("type is not valid")
	}
	if t.custom != "" {
		panic("type is already a custom type")
	}
	if name == "" {
		panic("custom type name is empty")
	}
	t.custom = name
	return t
}

// TimeLayout returns the time layout of DateTime, Date and Time types.
// Panics if t is not a DateTime, Date or Time type.
func (t Type) TimeLayout() string {
	if t.pt != PtDateTime && t.pt != PtTime {
		panic("cannot get time layout of a non-time type")
	}
	return t.vl.(string)
}

// WithLogicalType returns the type t but with the logical type lt.
// Panics if lt is not a logical type.
func (t Type) WithLogicalType(lt LogicalType) Type {
	if !lt.Valid() {
		panic("invalid logical type")
	}
	t.lt = lt
	return t
}

// PhysicalType returns the physical type of t.
func (t Type) PhysicalType() PhysicalType {
	return t.pt
}

// Types used to represent number ranges.
type intRange struct{ min, max int64 }
type uintRange struct{ min, max uint64 }
type floatRange struct {
	min, max   float64
	minS, maxS string
}
type decimalRange struct{ min, max decimal.Decimal }

// WithIntRange returns t but with values in [min,max]. t must be an Int, Int8,
// Int16, Int24 or Int64 type. min cannot be greater than max. min and max must
// be within the range of values of t.
// It panics it previous restrictions are not met.
func (t Type) WithIntRange(min, max int64) Type {
	if t.pt < PtInt || t.pt > PtInt64 {
		panic("type is not an Int, Int8, Int16, Int24 or Int64 type")
	}
	Min, Max := minInt[t.pt-PtInt], maxInt[t.pt-PtInt]
	if min == Min && max == Max {
		return t
	}
	if min < Min || min > Max {
		panic(fmt.Sprintf("min is not in range [%d,%d]", Min, Max))
	}
	if max < min {
		panic("max cannot be less than min")
	}
	if max > Max {
		panic(fmt.Sprintf("max cannot be greater than %d", Max))
	}
	t.vl = intRange{min, max}
	return t
}

// IntRange returns the minimum and maximum value for t. t must be ab Int,
// Int8, Int16, Int24 or Int64 type, otherwise it panics.
func (t Type) IntRange() (min, max int64) {
	if t.pt < PtInt || t.pt > PtInt64 {
		panic("type is not an Int, Int8, Int16, Int24 or Int64 type")
	}
	if i, ok := t.vl.(intRange); ok {
		return i.min, i.max
	}
	return minInt[t.pt-PtInt], maxInt[t.pt-PtInt]
}

// WithUintRange returns t but with values in [min,max]. t must be a UInt,
// UInt8, UInt16, UInt24 or UInt64 type. min cannot be greater than max.
// min and max must be within the range of values of t.
// It panics it previous restrictions are not met.
func (t Type) WithUintRange(min, max uint64) Type {
	if t.pt < PtUInt || t.pt > PtUInt64 {
		panic("type is not a UInt, UInt8, Int16, UInt24 or UInt64 type")
	}
	Max := maxUInt[t.pt-PtInt]
	if min == 0 && max == Max {
		return t
	}
	if min > Max {
		panic(fmt.Sprintf("min is not in range [0,%d]", Max))
	}
	if max < min {
		panic("max cannot be less than min")
	}
	if max > Max {
		panic(fmt.Sprintf("max cannot be greater than %d", Max))
	}
	t.vl = uintRange{min, max}
	return t
}

// UIntRange returns the minimum and maximum value for t. t must be an UInt,
// UInt8, UInt16, UInt24 or UInt64, otherwise it panics.
func (t Type) UIntRange() (min, max uint64) {
	if t.pt < PtUInt || t.pt > PtUInt64 {
		panic("type is not an UInt, UInt8, Int16, UInt24 or UInt64 type")
	}
	if i, ok := t.vl.(uintRange); ok {
		return i.min, i.max
	}
	return 0, maxUInt[t.pt-PtInt]
}

// WithFloatRange returns t but with values in [min,max]. t must be a Float or
// Float32 type. min cannot be greater than max.min and max cannot be NaN and
// ±Inf. It panics if previous restrictions are not met.
func (t Type) WithFloatRange(min, max float64) Type {
	if t.pt != PtFloat && t.pt != PtFloat32 {
		panic("type is not a Float or Float32 type")
	}
	if math.IsNaN(min) || math.IsNaN(max) {
		panic("min and max cannot be NaN")
	}
	if math.IsInf(min, 0) || math.IsInf(max, 0) {
		panic("min and max cannot be Inf")
	}
	Max := MaxFloat
	if t.pt == PtFloat32 {
		Max = MaxFloat32
	}
	if min == -Max && max == Max {
		return t
	}
	if min < -Max || min > Max {
		panic(fmt.Sprintf("min is not in range [%f,%f]", -Max, Max))
	}
	if max < min {
		panic("max cannot be less than min")
	}
	if max > Max {
		panic(fmt.Sprintf("max cannot be greater than %f", Max))
	}
	var minS, maxS string
	if t.pt == PtFloat32 {
		min, max = float64(float32(min)), float64(float32(max))
		if min != -MaxFloat32 {
			minS = decimal.NewFromFloat32(float32(min)).String()
		}
		if max != MaxFloat32 {
			maxS = decimal.NewFromFloat32(float32(max)).String()
		}
	} else {
		if min != -MaxFloat {
			minS = decimal.NewFromFloat(min).String()
		}
		if max != MaxFloat {
			maxS = decimal.NewFromFloat(max).String()
		}
	}
	t.vl = floatRange{min: min, max: max, minS: minS, maxS: maxS}
	return t
}

// FloatRange returns the minimum and maximum value of t. t must be a Float or
// Float32 type, otherwise it panics.
func (t Type) FloatRange() (min, max float64) {
	if t.pt != PtFloat && t.pt != PtFloat32 {
		panic("type is not a Float or Float32 type")
	}
	if f, ok := t.vl.(floatRange); ok {
		return f.min, f.max
	}
	if t.pt == PtFloat32 {
		return -MaxFloat32, MaxFloat32
	}
	return -MaxFloat, MaxFloat
}

// WithDecimalRange returns t but with values in [min,max]. t must be a Decimal
// type, otherwise it panics.
func (t Type) WithDecimalRange(min, max decimal.Decimal) Type {
	if t.pt != PtDecimal {
		panic("type is not a Decimal type")
	}
	Max := MaxDecimal
	if t.p != 0 || t.s != 0 {
		Max = decimal.New(1, t.p).Sub(one)
		if t.s != 0 {
			Max = Max.Shift(-t.s)
		}
	}
	Min := Max.Neg()
	if min.Equal(Min) && max.Equal(Max) {
		return t
	}
	if min.LessThan(Min) || min.GreaterThan(Max) {
		panic(fmt.Sprintf("min must be in range [%s,%s]", Min, Max))
	}
	if max.LessThan(min) || max.GreaterThan(Max) {
		panic(fmt.Sprintf("max must be in range [%s,%s]", min, Max))
	}
	t.vl = decimalRange{min, max}
	return t
}

// DecimalRange returns the minimum and maximum value for t. t must be a
// Decimal type, otherwise it panics.
func (t Type) DecimalRange() (min, max decimal.Decimal) {
	if t.pt != PtDecimal {
		panic("type is not a Decimal type")
	}
	if d, ok := t.vl.(decimalRange); ok {
		return d.min, d.max
	}
	return MinDecimal, MaxDecimal
}

// ItemType returns the type of the item of an Array type.
// Panics if t is not an Array type.
func (t Type) ItemType() Type {
	if t.pt != PtArray {
		panic("cannot get the item type of a non-Array type")
	}
	return t.vl.(Type)
}

// WithEnum returns t but with a fixed set of values. t must be a Text type.
// Panics if t is not a Text type, if values is empty, or if t has a regular
// expression.
func (t Type) WithEnum(values []string) Type {
	if t.pt != PtText {
		panic("cannot set enum for a non-Text type")
	}
	if len(values) == 0 {
		panic("enum is empty")
	}
	if _, ok := t.vl.(*regexp.Regexp); ok {
		panic("cannot set enum when there is a regular expression")
	}
	vl := make([]string, len(values))
	copy(vl, values)
	t.vl = vl
	return t
}

// Enum returns the enum values of t. Returns nil if t has no enum.
// Panics if t is not a Text type.
func (t Type) Enum() []string {
	if t.pt != PtText {
		panic("cannot get enum for a non-Text type")
	}
	if vl, ok := t.vl.([]string); ok {
		values := make([]string, len(vl))
		copy(values, vl)
		return values
	}
	return nil
}

// WithLen returns the type t but with length in [min,max]. t must be an Array.
// Panics if t is not an Array type, or min is not in [0,MaxArrayLen], or max
// is not in [min,MaxArrayLen].
func (t Type) WithLen(min, max int) Type {
	if t.pt != PtArray {
		panic("cannot set the length of a non-Array type")
	}
	if min < 0 || min > MaxArrayLen {
		panic("invalid minimum length")
	}
	if max < min || max > MaxArrayLen {
		panic("invalid maximum length")
	}
	t.p = int32(min)
	t.s = int32(max)
	return t
}

// WithUnique returns the type t but with unique items. t must be an Array and
// the item type cannot be Array and Object.
// Panics if t is not an Array or the item type is Array or Object.
func (t Type) WithUnique(on bool) Type {
	if t.pt != PtArray {
		panic("cannot set unique of a non-Array type")
	}
	if pt := t.vl.(Type).pt; pt == PtArray || pt == PtObject {
		panic("cannot set unique for items of type Array and Object")
	}
	t.unique = on
	return t
}

// Custom returns the custom name of t. If t is not a custom type it returns an empty
// string. Panics if t is not a valid type.
func (t Type) Custom() string {
	if !t.Valid() {
		panic("type is not valid")
	}
	return t.custom
}

// Properties returns an iterator to iterate over the properties of an Object
// type. Panics if t is not an Object type.
func (t Type) Properties() *Properties {
	if t.pt != PtObject {
		panic("cannot get the properties of a non-Array type")
	}
	return &Properties{t.vl.([]Property)}
}

// LogicalType returns the logical type of t and true.
// If t has no logical type, it returns false.
func (t Type) LogicalType() (LogicalType, bool) {
	return t.lt, t.lt > 0
}

// Regexp returns the regular expression of t. If t has no regular expression,
// it returns nil. Panics if t is not a Text type.
func (t Type) Regexp() *regexp.Regexp {
	if t.pt != PtText {
		panic("cannot return regular expression for a non-Text type")
	}
	re, _ := t.vl.(*regexp.Regexp)
	return re
}

// WithRegexp returns t with the regular expression re.
// Panics if t is not a Text type or if t has any values.
func (t Type) WithRegexp(re *regexp.Regexp) Type {
	if t.pt != PtText {
		panic("cannot set regular expression for a non-Text type")
	}
	if _, ok := t.vl.([]string); ok {
		panic("cannot set regular expression if there is enum")
	}
	t.vl = re
	return t
}

// ByteLen returns the maximum length in bytes of a Text type and true.
// If t has no maximum length in bytes, it returns 0 and false.
// Panics if t is not a Text type.
func (t Type) ByteLen() (int, bool) {
	if t.pt != PtText {
		panic("cannot get byte length of a non-Text type")
	}
	return int(t.p), t.p > 0
}

// CharLen returns the maximum length in characters of a Text type and true.
// If t has no maximum length in characters, it returns 0 and false.
// Panics if t is not a Text type.
func (t Type) CharLen() (int, bool) {
	if t.pt != PtText {
		panic("cannot get character length of a non-Text type")
	}
	return int(t.s), t.s > 0
}

// Precision returns the precision of a Decimal type.
// Panics if t is not a Decimal type.
func (t Type) Precision() int {
	if t.pt != PtDecimal {
		panic("cannot get precision of a non-Decimal type")
	}
	return int(t.p)
}

// Scale returns the scale of a Decimal type.
// Panics if t is not a Decimal type.
func (t Type) Scale() int {
	if t.pt != PtDecimal {
		panic("cannot get scale of a non-Decimal type")
	}
	return int(t.s)
}

// String returns a string representation of t.
// Panics if t is not a valid type.
func (t Type) String() string {
	s := t.pt.String()
	switch t.pt {
	case PtDecimal:
		if t.p > 0 {
			s += "(" + strconv.Itoa(int(t.p)) + "," + strconv.Itoa(int(t.s)) + ")"
		}
	case PtText:
		if t.p > 0 || t.s > 0 {
			s += "("
			if t.p > 0 {
				s += strconv.Itoa(int(t.p)) + " bytes"
			}
			if t.s > 0 {
				if t.p > 0 {
					s += ","
				}
				s += strconv.Itoa(int(t.p)) + " chars"
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

// Properties is an iterator to iterate over the properties of an object.
type Properties struct {
	pr []Property
}

// Next returns the next property of the iterator.
// If there are no more properties, it returns Property{} and false.
func (si *Properties) Next() (Property, bool) {
	if si.pr == nil {
		panic("next on a ended iterator")
	}
	if len(si.pr) == 0 {
		si.pr = nil
		return Property{}, false
	}
	p := si.pr[0]
	si.pr = si.pr[1:]
	return p, true
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
