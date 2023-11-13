//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"
	"golang.org/x/text/unicode/norm"
)

// PathNotExistError is returned by PropertyByPath when the path does not exist.
type PathNotExistError struct {
	Path Path
}

func (err PathNotExistError) Error() string {
	return fmt.Sprintf("property path %q does not exist", err.Path)
}

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

const (
	MaxDecimalPrecision = 76             // Maximum precision for a Decimal type
	MaxDecimalScale     = 38             // Maximum scale for a Decimal type
	MaxItems            = math.MaxInt32  // Maximum number of items of an Array type
	MaxTextLen          = math.MaxUint32 // Maximum length in bytes and characters for a Text type
	MaxYear             = 9999           // Maximum year for DataTime, Date and Year types
	MinYear             = 1              // Minimum year for DataTime, Date and Year types

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
)

type PhysicalType int8

const (
	PtInvalid PhysicalType = iota
	PtBoolean
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
	PtInet
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
	"Inet",
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

// Placeholder represents a property placeholder. It can hold nil or a string
// value. In the case of maps, it can also hold a non-nil map[string]string
// value where the keys correspond to the keys of the map.
type Placeholder any

// Property represents an object property.
type Property struct {
	Name        string
	Label       string
	Description string
	Placeholder Placeholder
	Role        Role
	Type        Type
	Required    bool
	Nullable    bool
	Flat        bool
}

// Path represents a property path.
type Path []string

// String returns the string representation of p.
func (p Path) String() string {
	return strings.Join(p, ".")
}

// Equals reports whether path is equal to p.
func (p Path) Equals(path Path) bool {
	if len(p) != len(path) {
		return false
	}
	for i, name := range p {
		if name != path[i] {
			return false
		}
	}
	return true
}

// Type represents a type.
type Type struct {
	pt PhysicalType

	unique bool // unique reports whether the items of an Array must be unique.
	real   bool // real reports whether NaN, +Inf and -Inf are allowed for Float and Float32 types.
	flat   bool // flat reports whether contains, at any level, a flat property.

	// p represents
	//   - minimum value for Int, Int8, Int16 and Int24
	//   - minimum value, as uint32(p), for UInt, UInt8, UInt16 and UInt24
	//   - precision for Decimal
	//   - length in bytes, as uint32(p), for Text
	//   - minimum length for Array
	p int32

	// s represents
	//   - maximum value for Int, Int8, Int16 and Int24
	//   - maximum value, as uint32(s), for UInt, UInt8, UInt16 and UInt24
	//   - scale for Decimal
	//   - length in characters, as uint32(s), for Text
	//   - maximum length for Array
	s int32

	// vl can contain one of
	//   - intRange value for Int64
	//   - uintRange value for UInt64
	//   - floatRange value for Float and Float32
	//   - decimalRange value for Decimal
	//   - *regexp.Regexp value for Text
	//   - []string with the values for Text
	//   - []Property for Object
	//   - Type of the item for Array
	//   - Type of the value for Map
	vl any

	_ []int // make Type non-comparable.
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

// Inet returns the Inet type.
func Inet() Type {
	return Type{pt: PtInet}
}

// Text returns a Text type.
func Text() Type {
	return Type{pt: PtText}
}

// Array returns an Array type with items of type t.
func Array(t Type) Type {
	return Type{pt: PtArray, flat: t.flat, s: MaxItems, vl: t}
}

// Object returns an Object type with the given properties.
// Panics if properties is empty, or if a property name is empty or repeated,
// or if a property string field is not UTF-8 encoded or if a property type is
// not valid.
func Object(properties []Property) Type {
	t, err := ObjectOf(properties)
	if err != nil {
		panic(err)
	}
	return t
}

// Map returns a Map type with value type t.
func Map(t Type) Type {
	return Type{pt: PtMap, flat: t.flat, vl: t}
}

// ObjectOf is like Object but returns an error instead of panicking if any.
func ObjectOf(properties []Property) (Type, error) {
	if len(properties) == 0 {
		return Type{}, errors.New("no property in type")
	}
	exists := make(map[string]struct{}, len(properties))
	ps := make([]Property, len(properties))
	flat := false
	for i, property := range properties {
		if property.Name == "" {
			return Type{}, errors.New("property name is empty")
		}
		if !IsValidPropertyName(property.Name) {
			return Type{}, errors.New("invalid property name")
		}
		if _, ok := exists[property.Name]; ok {
			return Type{}, fmt.Errorf("property %s name is repeated", property.Name)
		}
		exists[property.Name] = struct{}{}
		label, err := normalizedUTF8(property.Label)
		if err != nil {
			return Type{}, err
		}
		description, err := normalizedUTF8(property.Description)
		if err != nil {
			return Type{}, err
		}
		if property.Role < BothRole || property.Role > DestinationRole {
			return Type{}, errors.New("invalid property role")
		}
		if !property.Type.Valid() {
			return Type{}, errors.New("invalid property type")
		}
		placeholder, err := clonePlaceholder(property.Placeholder, property.Type)
		if err != nil {
			return Type{}, err
		}
		if property.Flat && property.Type.pt != PtObject {
			return Type{}, errors.New("invalid property flat")
		}
		ps[i] = Property{
			Name:        property.Name,
			Label:       label,
			Description: description,
			Placeholder: placeholder,
			Role:        property.Role,
			Type:        property.Type,
			Required:    property.Required,
			Nullable:    property.Nullable,
			Flat:        property.Flat,
		}
		flat = flat || property.Flat || property.Type.flat
	}
	return Type{pt: PtObject, flat: flat, vl: ps}, nil
}

// IsValidPropertyName reports whether name is a valid property name.
// A property name must:
//   - start with [A-Za-z_]
//   - subsequently contain only [A-Za-z0-9_]
func IsValidPropertyName(name string) bool {
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

// IsValidPropertyPath reports whether path is a valid property path.
// A property path is formed by property names separated by periods.
func IsValidPropertyPath(path string) bool {
	for path != "" {
		i := strings.IndexByte(path, '.')
		if i == -1 {
			i = len(path)
		}
		if !IsValidPropertyName(path[:i]) {
			return false
		}
		if i == len(path) {
			return true
		}
		path = path[i+1:]
	}
	return false
}

// ErrPathInvalid is the error returned by ParsePropertyPath is the path is not
// valid.
var ErrPathInvalid = errors.New("property path is not valid")

// ParsePropertyPath parses a property path and returns its representation as
// a Path. It returns the ErrPathInvalid error if the path is not valid.
func ParsePropertyPath(path string) (Path, error) {
	pp := strings.Split(path, ".")
	if len(pp) == 0 {
		return nil, ErrPathInvalid
	}
	for _, p := range pp {
		if !IsValidPropertyPath(p) {
			return nil, ErrPathInvalid
		}
	}
	return pp, nil
}

// AsRole returns an object type with the properties of typ but that are
// compatible with role. It returns typ if all properties are compatible and
// an invalid type if there are no compatible properties.
//
// Panics if typ or role are not valid, or typ is not an object type or role is
// Both.
func (t Type) AsRole(role Role) Type {
	if !t.Valid() {
		panic("type is not valid")
	}
	if t.pt != PtObject {
		panic("cannot return type as role for non-Object type")
	}
	if role < BothRole || role > DestinationRole {
		panic("role is not valid")
	}
	if role == BothRole {
		return t
	}
	last := 0
	var roleProperties []Property
	properties := t.vl.([]Property)
	for i, p := range properties {
		if p.Role == BothRole || p.Role == role {
			continue
		}
		if last < i {
			roleProperties = append(roleProperties, properties[last:i]...)
		}
		last = i + 1
	}
	if last == 0 {
		return t
	}
	if last < len(properties) {
		roleProperties = append(roleProperties, properties[last:]...)
	}
	if roleProperties == nil {
		return Type{}
	}
	return Type{pt: PtObject, vl: roleProperties}
}

// AsReal returns t but as a real number. As a real number, t does not allow
// NaN, +Inf and -Inf values. t must be a Float or a Float32 type. t cannot be
// already real and cannot have a range.
// It panics if previous restrictions are not met.
func (t Type) AsReal() Type {
	if t.pt != PtFloat && t.pt != PtFloat32 {
		panic("type is not a Float or Float32 type")
	}
	if t.real {
		panic("type is already real")
	}
	if _, ok := t.vl.(floatRange); ok {
		panic("type has a range")
	}
	t.real = true
	return t
}

// IsReal reports whether t is real.
// Panics if t is not a Float or a Float32 type.
func (t Type) IsReal() bool {
	if t.pt != PtFloat && t.pt != PtFloat32 {
		panic("type is not a Float or Float32 type")
	}
	return t.real
}

// Valid indicates if t is valid.
func (t Type) Valid() bool {
	return t.pt != PtInvalid
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
				s += strconv.Itoa(int(t.s)) + " chars"
			}
			s += ")"
		}
	}
	return s
}

// PhysicalType returns the physical type of t.
// Returns PtInvalid if t is not valid.
func (t Type) PhysicalType() PhysicalType {
	return t.pt
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
	if t.pt != PtUInt64 {
		return uint64(uint32(t.p)), uint64(uint32(t.s))
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
	if t.real {
		if t.pt == PtFloat32 {
			return -math.MaxFloat32, math.MaxFloat32
		}
		return -math.MaxFloat64, math.MaxFloat64
	}
	return math.Inf(-1), math.Inf(1)
}

// WithFloatRange returns t but with values in [min,max]. t must be a Float or
// Float32 type. min cannot be greater than max. min and max cannot be NaN, and
// if r is real they cannot be ±Inf.
// It panics if previous restrictions are not met.
func (t Type) WithFloatRange(min, max float64) Type {
	if t.pt != PtFloat && t.pt != PtFloat32 {
		panic("type is not a Float or Float32 type")
	}
	if math.IsNaN(min) || math.IsNaN(max) {
		panic("min and max cannot be NaN")
	}
	if t.real && (math.IsInf(min, 0) || math.IsInf(max, 0)) {
		panic("min and max cannot be ±Inf")
	}
	if math.IsInf(min, -1) && math.IsInf(max, 1) {
		return t
	}
	if max < min {
		panic("max cannot be less than min")
	}
	var minS, maxS string
	if t.pt == PtFloat32 {
		min, max = float64(float32(min)), float64(float32(max))
		if !math.IsInf(min, -1) {
			minS = decimal.NewFromFloat32(float32(min)).String()
		}
		if !math.IsInf(max, 1) {
			maxS = decimal.NewFromFloat32(float32(max)).String()
		}
	} else {
		if !math.IsInf(min, -1) {
			minS = decimal.NewFromFloat(min).String()
		}
		if !math.IsInf(max, 1) {
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

// ByteLen returns the maximum length in bytes of a Text type and true.
// If t has no maximum length in bytes, it returns 0 and false.
// Panics if t is not a Text type.
func (t Type) ByteLen() (int, bool) {
	if t.pt != PtText {
		panic("cannot get byte length of a non-Text type")
	}
	return int(uint32(t.p)), t.p != 0
}

// WithByteLen returns t with a maximum length of l of a Text type. l must be in
// range [1, MaxTextLen].
// Panics if t is not a Text type, or if l is not in range, or if t has already
// a byte length, or if t already has values.
func (t Type) WithByteLen(l int) Type {
	if t.pt != PtText {
		panic("cannot set byte length of a non-Text type")
	}
	if t.s > 0 {
		panic("repeated length in bytes")
	}
	if l < 1 || MaxTextLen < l {
		panic("invalid text length")
	}
	if _, ok := t.vl.([]string); ok {
		panic("t already has values")
	}
	t.p = int32(uint32(l))
	return t
}

// CharLen returns the maximum length in characters of JSON and Text types and
// true. If t has no maximum length in characters, it returns 0 and false.
// Panics if t is not a JSON or Text type.
func (t Type) CharLen() (int, bool) {
	if t.pt != PtJSON && t.pt != PtText {
		panic("cannot get character length of non-JSON and non-Text types")
	}
	return int(uint32(t.s)), t.s != 0
}

// WithCharLen returns t with a maximum length of l of a JSON and Text type. l
// must be in range [1, MaxTextLen].
// Panics if t is not a JSON or Text type, or if l is not in range, or if t has
// already a char length, or if t already has values.
func (t Type) WithCharLen(l int) Type {
	if t.pt != PtJSON && t.pt != PtText {
		panic("cannot set character length of non-JSON and non-Text types")
	}
	if t.s > 0 {
		panic("repeated length in characters")
	}
	if l < 1 || MaxTextLen < l {
		panic("invalid text length")
	}
	if _, ok := t.vl.([]string); ok {
		panic("t already has values")
	}
	t.s = int32(uint32(l))
	return t
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
// Panics if t is not a Text type, or t has already a regular expression or has
// values.
func (t Type) WithRegexp(re *regexp.Regexp) Type {
	if t.pt != PtText {
		panic("cannot set regular expression for a non-Text type")
	}
	switch t.vl.(type) {
	case []string:
		panic("cannot set regular expression when t has values")
	case *regexp.Regexp:
		panic("t already has a regular expression")
	}
	t.vl = re
	return t
}

// Values returns the values of t. Returns nil if t has no values.
// Panics if t is not a Text type.
func (t Type) Values() []string {
	if t.pt != PtText {
		panic("cannot get values for a non-Text type")
	}
	if vl, ok := t.vl.([]string); ok {
		values := make([]string, len(vl))
		copy(values, vl)
		return values
	}
	return nil
}

// WithValues returns t but restricted to some values. t must be a Text type.
// It panics if t is not of Text type, if the values is empty or contains an
// invalid UTF-8 string, or if t already has values, a regular expression, or
// if it is already restricted by byte or character length.
func (t Type) WithValues(values ...string) Type {
	if t.pt != PtText {
		panic("cannot set values for a non-Text type")
	}
	if len(values) == 0 {
		panic("values is empty")
	}
	switch t.vl.(type) {
	case []string:
		panic("t already has values")
	case *regexp.Regexp:
		panic("t already has a regular expression")
	}
	if t.p != 0 {
		panic("t already has a maximum byte length")
	}
	if t.s != 0 {
		panic("t already has a maximum character length")
	}
	vl := make([]string, len(values))
	for i, s := range values {
		v, err := normalizedUTF8(s)
		if err != nil {
			panic(err)
		}
		vl[i] = v
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

// HasFlatProperties reports whether t contains flat properties.
func (t Type) HasFlatProperties() bool {
	return t.flat
}

// Unflatten returns t but with all properties as not flat.
// It returns t if t has no flat properties.
func (t Type) Unflatten() Type {
	if !t.flat {
		return t
	}
	switch t.pt {
	case PtObject:
		pp := t.Properties()
		for i := 0; i < len(pp); i++ {
			pp[i].Type = pp[i].Type.Unflatten()
			pp[i].Flat = false
		}
		t.vl = pp
	case PtArray, PtMap:
		t.vl = t.Elem().Unflatten()
	}
	t.flat = false
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

// PropertyByPath returns the property with the given path, or a
// PathNotExistError error if the path does not exist.
// Panics if t is not an Object type, path is empty or a property name within it
// is not a valid property name.
func (t Type) PropertyByPath(path Path) (Property, error) {
	if t.pt != PtObject {
		panic("cannot get the properties of a non-Object type")
	}
	if len(path) == 0 {
		panic("path must contain at least one name")
	}
	last := len(path) - 1
	var i int
	var name string
	for i, name = range path {
		if t.pt != PtObject {
			break
		}
		for _, prop := range t.vl.([]Property) {
			if prop.Name != name {
				continue
			}
			if i == last {
				return prop, nil
			}
			t = prop.Type
			break
		}
	}
	// Not found.
	for _, p := range path[i:] {
		if !IsValidPropertyName(p) {
			panic("invalid property path")
		}
	}
	return Property{}, PathNotExistError{path[:i+1]}
}

// Property returns the property with the given name and a boolean value
// indicating if the property exists.
// Panics if t is not an Object type or name is not a valid property name.
func (t Type) Property(name string) (Property, bool) {
	if t.pt != PtObject {
		panic("cannot get the properties of a non-Object type")
	}
	for _, p := range t.vl.([]Property) {
		if p.Name == name {
			return p, true
		}
	}
	if !IsValidPropertyName(name) {
		panic("invalid property name")
	}
	return Property{}, false
}

// Properties returns the properties of the Object type t.
// Panics if t is not an Object type.
func (t Type) Properties() []Property {
	if t.pt != PtObject {
		panic("cannot get the properties of a non-Object type")
	}
	return slices.Clone(t.vl.([]Property))
}

// PropertiesNames returns the names of the properties of the Object t.
// Panics if t is not an Object type.
func (t Type) PropertiesNames() []string {
	if t.pt != PtObject {
		panic("cannot get the properties names of a non-Object type")
	}
	properties := t.vl.([]Property)
	names := make([]string, len(properties))
	for i, p := range properties {
		names[i] = p.Name
	}
	return names
}

// Elem returns a type's element type.
// Panics if t is not an Array or Map type.
func (t Type) Elem() Type {
	if t.pt != PtArray && t.pt != PtMap {
		panic("cannot get the element type for a non-Array and non-Map type")
	}
	return t.vl.(Type)
}

// EqualTo reports whether t is equals to t2.
func (t Type) EqualTo(t2 Type) bool {
	almostEqual := t.pt == t2.pt && t.unique == t2.unique && t.real == t2.real && t.flat == t2.flat && t.p == t2.p
	if !almostEqual {
		return false
	}
	if t.vl == nil && t2.vl == nil {
		return true
	}
	if (t.vl == nil) != (t2.vl == nil) {
		return false
	}
	switch vl1 := t.vl.(type) {
	case Type:
		return vl1.EqualTo(t2.vl.(Type))
	case intRange, uintRange, floatRange, decimalRange, string:
		return vl1 == t2.vl
	case []Property:
		vl2 := t2.vl.([]Property)
		if len(vl1) != len(vl2) {
			return false
		}
		for i, p1 := range vl1 {
			p2 := (vl2)[i]
			if p1.Name != p2.Name ||
				p1.Label != p2.Label ||
				p1.Description != p2.Description ||
				!equalPlaceholder(p1.Placeholder, p2.Placeholder) ||
				p1.Role != p2.Role ||
				p1.Required != p2.Required ||
				p1.Nullable != p2.Nullable ||
				p1.Flat != p2.Flat ||
				!p1.Type.EqualTo(p2.Type) {
				return false
			}
		}
		return true
	case []string:
		vl2, ok := t2.vl.([]string)
		return ok && slices.Equal(vl1, vl2)
	case *regexp.Regexp:
		vl2, ok := t2.vl.(*regexp.Regexp)
		return ok && vl1.String() == vl2.String()
	}
	panic("unreachable code")
}

// clonePlaceholder clones and validates a placeholder for a property of type t
// and returns the cloned placeholder or an error if it is not valid.
func clonePlaceholder(placeholder any, t Type) (Placeholder, error) {
	switch placeholder := placeholder.(type) {
	case nil:
		return nil, nil
	case string:
		return normalizedUTF8(placeholder)
	case map[string]string:
		if t.PhysicalType() != PtMap {
			return nil, fmt.Errorf("invalid placeholder type")
		}
		if placeholder == nil {
			return nil, fmt.Errorf("invalid placeholder value")
		}
		var err error
		m := make(map[string]string, len(placeholder))
		for key, value := range placeholder {
			key, err = normalizedUTF8(key)
			if err != nil {
				return nil, err
			}
			value, err = normalizedUTF8(value)
			if err != nil {
				return nil, err
			}
			m[key] = value
		}
		return m, nil
	}
	return nil, fmt.Errorf("invalid placeholder type")
}

// normalizedUTF8 returns s as a normalized UTF-8 encoded string.
// Returns an error if s is not a valid UTF-8 encoded string.
func normalizedUTF8(s string) (string, error) {
	if !utf8.ValidString(s) {
		return "", errors.New("invalid UTF-8 encoding")
	}
	return norm.NFC.String(s), nil
}

// equalPlaceholder reports whether ph and ph2 are equal.
func equalPlaceholder(ph, ph2 Placeholder) bool {
	if ph, ok := ph.(map[string]string); ok {
		ph2, ok := ph2.(map[string]string)
		return ok && maps.Equal(ph, ph2)
	}
	return ph == ph2
}
