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
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
	"golang.org/x/text/unicode/norm"
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
	MaxDecimalPrecision = 76       // Maximum precision for a Decimal type
	MaxDecimalScale     = 38       // Maximum scale for a Decimal type
	MaxItems            = MaxInt24 // Maximum number of items of an Array type
	MaxTextLen          = MaxInt   // Maximum length in bytes and characters for a Text type
	MaxYear             = 9999     // Maximum year for DataTime, Date and Year types
	MinYear             = 1        // Minimum year for DataTime, Date and Year types

	MaxInt    = math.MaxInt32
	MaxInt16  = math.MaxInt16
	MaxInt24  = 1<<23 - 1
	MaxInt64  = math.MaxInt64
	MaxInt8   = math.MaxInt8
	MaxUInt   = math.MaxUint32
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
	PtMap
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
	"Map",
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

// ObjectProperty represents an object property.
type ObjectProperty struct {
	Name        string
	Aliases     []string
	Label       string
	Description string
	Type        Type
}

// Type represents a type.
type Type struct {
	pt PhysicalType
	lt LogicalType

	null   bool // null reports whether null values are allowed.
	unique bool // unique reports whether the items of an Array must be unique.

	// p represents
	//   - minimum value of Int, Int8, In16 and Int24 types
	//   - minimum value, as uint32(p), of UInt, UInt8, UIn16 and UInt24 types
	//   - precision of a Decimal type
	//   - length in bytes of a Text type
	//   - minimum length of an Array type
	p int32

	// s represents
	//   - maximum value for Int, Int8, In16 and Int24
	//   - maximum value, as uint32(s), for UInt, UInt8, UIn16 and UInt24
	//   - scale for Decimal
	//   - length in characters for Text
	//   - maximum length for Array
	s int32

	// vl can contain one of
	//   - intRange value for Int64
	//   - uintRange value for UInt64
	//   - floatRange value for Float and Float32
	//   - decimalRange value for Decimal
	//   - string value representing a layout for DateTime, Date and Time
	//   - *regexp.Regexp value for Text
	//   - []string with the enum values for Text
	//   - []ObjectProperty for Object
	//   - Type of the item for Array
	//   - Type of the value for Map
	vl any

	// custom type. Empty for non-custom types.
	custom string
}

// Boolean returns the Boolean type.
func Boolean() Type {
	return Type{pt: PtBoolean}
}

// Int returns the Int type.
func Int() Type {
	return Type{pt: PtInt, p: MinInt, s: MaxInt}
}

// Int8 returns the Int8 type.
func Int8() Type {
	return Type{pt: PtInt8, p: MinInt8, s: MaxInt8}
}

// Int16 returns the Int16 type.
func Int16() Type {
	return Type{pt: PtInt16, p: MinInt16, s: MaxInt16}
}

// Int24 returns the Int24 type.
func Int24() Type {
	return Type{pt: PtInt24, p: MinInt24, s: MaxInt24}
}

// Int64 returns the Int64 type.
func Int64() Type {
	return Type{pt: PtInt64}
}

// UInt returns the UInt type.
func UInt() Type {
	s := MaxUInt
	return Type{pt: PtUInt, s: int32(s)}
}

// UInt8 returns the UInt8 type.
func UInt8() Type {
	return Type{pt: PtUInt8, s: int32(MaxUInt8)}
}

// UInt16 returns the UInt16 type.
func UInt16() Type {
	return Type{pt: PtUInt16, s: int32(MaxUInt16)}
}

// UInt24 returns the UInt24 type.
func UInt24() Type {
	return Type{pt: PtUInt24, s: int32(MaxUInt24)}
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
// It panics if layout is not a valid UTF-8-encoded string.
func DateTime(layout string) Type {
	return Type{pt: PtDateTime, vl: normalizedUTF8(layout)}
}

// Date returns the Date type with the given layout.
// It panics if layout is not a valid UTF-8-encoded string.
func Date(layout string) Type {
	return Type{pt: PtDate, vl: normalizedUTF8(layout)}
}

// Time returns the Time type with the given layout.
// It panics if layout is not a valid UTF-8-encoded string.
func Time(layout string) Type {
	return Type{pt: PtTime, vl: normalizedUTF8(layout)}
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

// Array returns an Array type with items of type t.
func Array(t Type) Type {
	return Type{pt: PtArray, s: MaxItems, vl: t}
}

// Object returns an Object type with the given properties.
// Panics if properties is empty, or if a property name pr alias is empty or
// repeated, or if a property string field is not UTF-8 encoded or if a
// property type is not valid.
func Object(properties []ObjectProperty) Type {
	if len(properties) == 0 {
		panic("no property in object")
	}
	exists := make(map[string]struct{}, len(properties))
	ps := make([]ObjectProperty, len(properties))
	for i, property := range properties {
		if property.Name == "" {
			panic("property name is empty")
		}
		if !IsValidPropertyName(property.Name) {
			panic("invalid property name")
		}
		if _, ok := exists[property.Name]; ok {
			panic("property name is repeated")
		}
		exists[property.Name] = struct{}{}
		var aliases []string
		if len(property.Aliases) > 0 {
			aliases = make([]string, len(property.Aliases))
			for i, alias := range property.Aliases {
				if alias == "" {
					panic("property alias is empty")
				}
				if !IsValidPropertyName(alias) {
					panic("invalid property alias")
				}
				if _, ok := exists[alias]; ok {
					panic("property alias already named")
				}
				aliases[i] = alias
				exists[alias] = struct{}{}
			}
			sort.Strings(aliases)
		}
		if !property.Type.Valid() {
			panic("invalid property type")
		}
		ps[i] = ObjectProperty{
			Name:        property.Name,
			Aliases:     aliases,
			Label:       normalizedUTF8(property.Label),
			Description: normalizedUTF8(property.Description),
			Type:        property.Type,
		}
	}
	return Type{pt: PtObject, vl: ps}
}

// Map returns a Map type with value type t.
func Map(t Type) Type {
	return Type{pt: PtMap, vl: t}
}

// Valid indicates if t is valid.
func (t Type) Valid() bool {
	return t.pt != 0
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

// PhysicalType returns the physical type of t.
func (t Type) PhysicalType() PhysicalType {
	return t.pt
}

// LogicalType returns the logical type of t and true.
// If t has no logical type, it returns false.
func (t Type) LogicalType() (LogicalType, bool) {
	return t.lt, t.lt > 0
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

// Null reports whether null values are allowed.
// Panics if t is not a valid type.
func (t Type) Null() bool {
	if !t.Valid() {
		panic("type is not valid")
	}
	return t.null
}

// WithNull returns the type t but with null values allowed.
// Panics if t is not a valid type.
func (t Type) WithNull() Type {
	if !t.Valid() {
		panic("type is not valid")
	}
	t.null = true
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

// AsCustom returns t as a custom type called name.
// Panics if t is not valid, or t is already a custom type, or name is not
// valid custom name.
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
	if !IsValidCustomTypeName(name) {
		panic("custom type name is not valid")
	}
	t.custom = name
	return t
}

type intRange struct{ min, max int64 }

// IntRange returns the minimum and maximum value for t. t must be ab Int,
// Int8, Int16, Int24 or Int64 type, otherwise it panics.
func (t Type) IntRange() (min, max int64) {
	if t.pt < PtInt || t.pt > PtInt64 {
		panic("type is not an Int, Int8, Int16, Int24 or Int64 type")
	}
	if t.pt != PtInt64 {
		return int64(t.p), int64(t.s)
	}
	if i, ok := t.vl.(intRange); ok {
		return i.min, i.max
	}
	return MinInt64, MaxInt64
}

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
	if t.pt == PtInt64 {
		t.vl = intRange{min, max}
	} else {
		t.p, t.s = int32(min), int32(max)
	}
	return t
}

type uintRange struct{ min, max uint64 }

// UIntRange returns the minimum and maximum value for t. t must be an UInt,
// UInt8, UInt16, UInt24 or UInt64, otherwise it panics.
func (t Type) UIntRange() (min, max uint64) {
	if t.pt < PtUInt || t.pt > PtUInt64 {
		panic("type is not an UInt, UInt8, Int16, UInt24 or UInt64 type")
	}
	if t.pt != PtInt64 {
		return uint64(t.p), uint64(t.s)
	}
	if i, ok := t.vl.(uintRange); ok {
		return i.min, i.max
	}
	return 0, MaxUInt64
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
	if t.pt == PtInt64 {
		t.vl = uintRange{min, max}
	} else {
		t.p, t.s = int32(min), int32(max)
	}
	return t
}

type floatRange struct {
	min, max   float64
	minS, maxS string
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

type decimalRange struct{ min, max decimal.Decimal }

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

// Layout returns the layout of DateTime, Date and Time types.
// Panics if t is not a DateTime, Date or Time type.
func (t Type) Layout() string {
	if t.pt != PtDateTime && t.pt != PtDate && t.pt != PtTime {
		panic("cannot get layout of a non-time type")
	}
	return t.vl.(string)
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
// Panics if t is not a Text type, or t has already a regular expression or
// has enum.
func (t Type) WithRegexp(re *regexp.Regexp) Type {
	if t.pt != PtText {
		panic("cannot set regular expression for a non-Text type")
	}
	switch t.vl.(type) {
	case []string:
		panic("cannot set regular expression when t has an enum")
	case *regexp.Regexp:
		panic("t already has a regular expression")
	}
	t.vl = re
	return t
}

// ItemType returns the type of the item of an Array type.
// Panics if t is not an Array type.
func (t Type) ItemType() Type {
	if t.pt != PtArray {
		panic("cannot get the item type of a non-Array type")
	}
	return t.vl.(Type)
}

// Enum returns the enum values of t. Returns nil if t has no enum.
// Panics if t is not a Text type.
func (t Type) Enum() []string {
	if t.pt != PtText {
		panic("cannot get enum for a non-Text type")
	}
	if vl, ok := t.vl.([]string); ok {
		enum := make([]string, len(vl))
		copy(enum, vl)
		return enum
	}
	return nil
}

// WithEnum returns t but with an enum. t must be a Text type.
// Panics if t is not a Text type, or enum is empty or contains an invalid
// UTF-8 string, or t already has an enum or a regular expression.
func (t Type) WithEnum(enum []string) Type {
	if t.pt != PtText {
		panic("cannot set enum for a non-Text type")
	}
	if len(enum) == 0 {
		panic("enum is empty")
	}
	switch t.vl.(type) {
	case []string:
		panic("t already has an enum")
	case *regexp.Regexp:
		panic("cannot set enum when t has a regular expression")
	}
	vl := make([]string, len(enum))
	for i, s := range enum {
		vl[i] = normalizedUTF8(s)
	}
	t.vl = vl
	return t
}

// MinItems returns the minimum number of items of t. t must be an Array,
// otherwise it panics.
func (t Type) MinItems() int {
	if t.pt != PtArray {
		panic("cannot get the minimum number of items of a non-Array type")
	}
	return int(t.p)
}

// WithMinItems returns t but with the minimum number of items sets to min.
// t must be an Array. Panics if t is not an Array type or min is not in
// [0,max] where max is the maximum number of items of t.
func (t Type) WithMinItems(min int) Type {
	if t.pt != PtArray {
		panic("cannot set the minimum number of items for a non-Array type")
	}
	if min < 0 || min > int(t.s) {
		panic(fmt.Sprintf("minimum number of items not in [0,%d]", t.s))
	}
	t.p = int32(min)
	return t
}

// MaxItems returns the maximum number of items of t. t must be an Array,
// otherwise it panics.
func (t Type) MaxItems() int {
	if t.pt != PtArray {
		panic("cannot get the maximum number of items of a non-Array type")
	}
	return int(t.s)
}

// WithMaxItems returns t but with the maximum number of items sets to max.
// t must be an Array. Panics if t is not an Array type or max is not in
// [min,MaxItems] where min is the minimum number of items of t.
func (t Type) WithMaxItems(max int) Type {
	if t.pt != PtArray {
		panic("cannot set the maximum number of items for a non-Array type")
	}
	if max < int(t.p) || max > MaxItems {
		panic(fmt.Sprintf("maximum number of items not in [%d,%d]", t.p, MaxItems))
	}
	t.s = int32(max)
	return t
}

// Unique reports whether the items of t are unique.
// Panics if t is not an Array.
func (t Type) Unique() bool {
	if t.pt != PtArray {
		panic("cannot get unique of a non-Array type")
	}
	return t.unique
}

// WithUnique returns the type t but with unique items. t must be an Array and
// its item type cannot be Array or Object.
// Panics if t is not an Array or the item type is Array or Object.
func (t Type) WithUnique() Type {
	if t.pt != PtArray {
		panic("cannot set unique of a non-Array type")
	}
	if pt := t.vl.(Type).pt; pt == PtArray || pt == PtObject {
		panic("cannot set unique for an Array with items of type Array or Object")
	}
	t.unique = true
	return t
}

// Properties returns the properties of the Object type t.
// Panics if t is not an Object type.
func (t Type) Properties() []ObjectProperty {
	if t.pt != PtObject {
		panic("cannot get the properties of a non-Object type")
	}
	properties := slices.Clone(t.vl.([]ObjectProperty))
	for _, property := range properties {
		property.Aliases = slices.Clone(property.Aliases)
	}
	return properties
}

// PropertiesNames returns the names of the properties of the Object t.
// Panics if t is not an Object type.
func (t Type) PropertiesNames() []string {
	if t.pt != PtObject {
		panic("cannot get the properties names of a non-Object type")
	}
	properties := t.vl.([]ObjectProperty)
	names := make([]string, len(properties))
	for i, p := range properties {
		names[i] = p.Name
	}
	return names
}

// ValueType returns the type of the value of a Map type.
// Panics if t is not a Map type.
func (t Type) ValueType() Type {
	if t.pt != PtMap {
		panic("cannot get the value type of a non-Map type")
	}
	return t.vl.(Type)
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

// normalizedUTF8 returns s as a normalized UTF-8 encoded string.
// Panics if s is not a valid UTF-8 encoded string.
func normalizedUTF8(s string) string {
	if !utf8.ValidString(s) {
		panic("invalid UTF-8 encoding")
	}
	return norm.NFC.String(s)
}

// IsValidCustomTypeName reports whether name is a valid custom type name.
// A custom type name must:
//   - start with [A-Za-z_]
//   - subsequently contain only [A-Za-z0-9_]
func IsValidCustomTypeName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !('a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return false
		}
	}
	return true
}
