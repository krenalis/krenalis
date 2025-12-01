// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo/tools/decimal"

	"golang.org/x/text/unicode/norm"
)

// InvalidPropertyNameError is returned by ObjectOf when a property name is not
// valid.
type InvalidPropertyNameError struct {
	Index int
	Name  string
}

func (err InvalidPropertyNameError) Error() string {
	return fmt.Sprintf("property name %q is not valid", err.Name)
}

// RepeatedPropertyNameError is returned by ObjectOf when a property name is
// repeated.
type RepeatedPropertyNameError struct {
	Index1, Index2 int
	Name           string
}

func (err RepeatedPropertyNameError) Error() string {
	return fmt.Sprintf("property name %q is repeated", err.Name)
}

var (
	bitSize     = [...]int{8, 16, 24, 32, 64}
	minInt      = [...]int64{MinInt8, MinInt16, MinInt24, MinInt32, MinInt64}
	maxInt      = [...]int64{MaxInt8, MaxInt16, MaxInt24, MaxInt32, MaxInt64}
	maxUnsigned = [...]uint64{MaxUint8, MaxUint16, MaxUint24, MaxUint32, MaxUint64}
)

const (
	MaxDecimalPrecision = 76             // Maximum precision for a decimal type
	MaxDecimalScale     = 37             // Maximum scale for a decimal type
	MaxElements         = math.MaxInt32  // Maximum number of elements of an array type
	MaxStringLen        = math.MaxUint32 // Maximum length in bytes and characters for a string type
	MaxYear             = 9999           // Maximum year for datetime, date and year types
	MinYear             = 1              // Minimum year for datetime, date and year types

	MaxInt16  = math.MaxInt16
	MaxInt24  = 1<<23 - 1
	MaxInt32  = math.MaxInt32
	MaxInt64  = math.MaxInt64
	MaxInt8   = math.MaxInt8
	MaxUint16 = math.MaxUint16
	MaxUint24 = 1<<24 - 1
	MaxUint32 = math.MaxUint32
	MaxUint64 = math.MaxUint64
	MaxUint8  = math.MaxUint8
	MinInt16  = math.MinInt16
	MinInt24  = -1 << 23
	MinInt32  = math.MinInt32
	MinInt64  = math.MinInt64
	MinInt8   = math.MinInt8
)

type Kind int8

const (
	InvalidKind Kind = iota
	StringKind
	BooleanKind
	IntKind
	FloatKind
	DecimalKind
	DateTimeKind
	DateKind
	TimeKind
	YearKind
	UUIDKind
	JSONKind
	IPKind
	ArrayKind
	ObjectKind
	MapKind
)

var kindName = []string{
	"string",
	"boolean",
	"int",
	"float",
	"decimal",
	"datetime",
	"date",
	"time",
	"year",
	"uuid",
	"json",
	"ip",
	"array",
	"object",
	"map",
}

// String returns the name of k.
func (k Kind) String() string {
	if !k.Valid() {
		return "Invalid"
	}
	return kindName[k-1]
}

// Valid reports whether k is a valid kind.
func (k Kind) Valid() bool {
	return 1 <= k && int(k) <= len(kindName)
}

// KindByName returns a kind by its name. The second return parameter reports
// whether a kind with the given name exists.
func KindByName(name string) (Kind, bool) {
	for i, n := range kindName {
		if n == name {
			return Kind(i + 1), true
		}
	}
	return InvalidKind, false
}

// Property represents an object property.
type Property struct {
	Name           string
	Prefilled      string
	Type           Type
	CreateRequired bool
	UpdateRequired bool
	ReadOptional   bool
	Nullable       bool
	Description    string
}

var _ interface {
	json.Marshaler
	json.Unmarshaler
} = (*Property)(nil)

// Type represents a type.
type Type struct {
	kind Kind

	size int8 // size for int and float: 0 (8 bits), 1 (16 bits), 2 (24 bits), 3 (32 bits), and 4 (64 bits)

	generic  bool // generic reports whether it is a generic type.
	unsigned bool // unsigned reports whether the integer type is unsigned.
	unique   bool // unique reports whether the elements of an array must be unique.
	real     bool // real reports whether NaN, +Inf and -Inf are not allowed for float.

	// p represents
	//   - minimum value for int with 8, 16, 24, and 32 bits; for unsigned, p is converted to uint32
	//   - precision for decimal
	//   - length in bytes, as uint32(p), for string
	//   - minimum length for array
	p int32

	// s represents
	//   - maximum value for int with 8, 16, 24, and 32 bits; for unsigned, p is converted to uint32
	//   - scale for decimal
	//   - length in characters, as uint32(s), for string
	//   - maximum length for array
	s int32

	// vl can contain one of
	//   - []string with the values for string
	//   - *regexp.Regexp value for string
	//   - intRange value for int with 64 bits
	//   - floatRange value for float
	//   - decimalRange value for decimal
	//   - Properties{properties, names} for object
	//   - Type of the elements for array
	//   - Type of the value for map
	vl any

	_ []int // make Type non-comparable.
}

var _ interface {
	json.Marshaler
	json.Unmarshaler
} = (*Type)(nil)

// Parameter returns a type parameter with the specified name. name must follow
// the syntax rules of a property and must not conflict with the name of a kind.
func Parameter(name string) Type {
	if !IsValidPropertyName(name) {
		panic("parameter name is not valid")
	}
	if slices.Contains(kindName, name) {
		panic("parameter name cannot be the same as a kind")
	}
	return Type{kind: InvalidKind, generic: true, vl: name}
}

// String returns a string type.
func String() Type {
	return Type{kind: StringKind}
}

// Boolean returns the boolean type.
func Boolean() Type {
	return Type{kind: BooleanKind}
}

// Int returns the int type with the provided bit size. It panics if size is not
// 8, 16, 24, 32 or 64.
func Int(size int) Type {
	t := Type{kind: IntKind}
	switch size {
	case 8:
		t.p = MinInt8
		t.s = MaxInt8
	case 16:
		t.size = 1
		t.p = MinInt16
		t.s = MaxInt16
	case 24:
		t.size = 2
		t.p = MinInt24
		t.s = MaxInt24
	case 32:
		t.size = 3
		t.p = MinInt32
		t.s = MaxInt32
	case 64:
		t.size = 4
	default:
		panic("bit size is not valid")
	}
	return t
}

// Float returns the float type with the provided bit size. It panics if size is
// not 32 or 64.
func Float(size int) Type {
	t := Type{kind: FloatKind}
	switch size {
	case 32:
		t.size = 3
	case 64:
		t.size = 4
	default:
		panic("bit size is not valid")
	}
	return t
}

// Decimal returns the decimal type with the given precision and scale.
// Panics if precision is not in range [1,MaxDecimalPrecision] or if scale is
// not in range [0,MaxDecimalScale] or if it is greater that precision.
func Decimal(precision, scale int) Type {
	if precision < 1 || precision > MaxDecimalPrecision {
		panic("invalid decimal precision")
	}
	if scale < 0 || scale > MaxDecimalScale || scale > precision {
		panic("invalid decimal scale")
	}
	min, max := decimal.Range(precision, scale)
	vl := decimalRange{min, max}
	return Type{kind: DecimalKind, p: int32(precision), s: int32(scale), vl: vl}
}

// DateTime returns the datetime type.
func DateTime() Type {
	return Type{kind: DateTimeKind}
}

// Date returns the date type.
func Date() Type {
	return Type{kind: DateKind}
}

// Time returns the time type.
func Time() Type {
	return Type{kind: TimeKind}
}

// Year returns the year type.
func Year() Type {
	return Type{kind: YearKind}
}

// UUID returns the uuid type.
func UUID() Type {
	return Type{kind: UUIDKind}
}

// JSON returns the json type.
func JSON() Type {
	return Type{kind: JSONKind}
}

// IP returns the ip type.
func IP() Type {
	return Type{kind: IPKind}
}

// Array returns an array type with elements of type t.
func Array(t Type) Type {
	return Type{kind: ArrayKind, s: MaxElements, vl: t}
}

// Object returns an object type with the given properties.
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

// Map returns a map type with value type t.
func Map(t Type) Type {
	return Type{kind: MapKind, vl: t}
}

// ObjectOf is like Object but returns an error instead of panicking if any.
// It returns an InvalidPropertyNameError error if a property name is not
// valid, and a RepeatedPropertyNameError error if a property name is repeated.
func ObjectOf(properties []Property) (Type, error) {
	if len(properties) == 0 {
		return Type{}, errors.New("no property in type")
	}
	var generic bool
	pn := Properties{
		properties: make([]Property, len(properties)),
		names:      make(map[string]int, len(properties)),
	}
	for i, property := range properties {
		if property.Name == "" {
			return Type{}, errors.New("property name is empty")
		}
		if !IsValidPropertyName(property.Name) {
			return Type{}, InvalidPropertyNameError{i, property.Name}
		}
		if j, ok := pn.names[property.Name]; ok {
			return Type{}, RepeatedPropertyNameError{j, i, property.Name}
		}
		pn.names[property.Name] = i
		prefilled, err := normalizedUTF8(property.Prefilled)
		if err != nil {
			return Type{}, err
		}
		if property.Type.Generic() {
			generic = true
		} else if !property.Type.Valid() {
			return Type{}, errors.New("invalid property type")
		}
		description, err := normalizedUTF8(property.Description)
		if err != nil {
			return Type{}, err
		}
		pn.properties[i] = Property{
			Name:           property.Name,
			Prefilled:      prefilled,
			Type:           property.Type,
			CreateRequired: property.CreateRequired,
			UpdateRequired: property.UpdateRequired,
			ReadOptional:   property.ReadOptional,
			Nullable:       property.Nullable,
			Description:    description,
		}
	}
	return Type{kind: ObjectKind, generic: generic, vl: pn}, nil
}

// AsReal returns t but as a real number. As a real number, t does not allow
// NaN, +Inf and -Inf values. t must be a float type. t cannot be already real
// and cannot have a range. It panics if previous restrictions are not met.
func (t Type) AsReal() Type {
	if t.kind != FloatKind {
		panic("type is not a float type")
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

// IsReal reports whether t is real. Panics if t is not a float type.
func (t Type) IsReal() bool {
	if t.kind != FloatKind {
		panic("type is not a float type")
	}
	return t.real
}

// Valid indicates if t is valid.
func (t Type) Valid() bool {
	return t.kind != InvalidKind
}

// Generic reports whether t is generic.
func (t Type) Generic() bool {
	return t.generic
}

// KindName returns the kind name of t. t must be valid or a generic type.
func (t Type) KindName() string {
	if t.kind == InvalidKind {
		if t.generic {
			return t.vl.(string)
		}
		panic("type is the invalid type")
	}
	return t.kind.String()
}

// String returns the string representation of t. If t is a type parameter, it
// returns its name; otherwise, for invalid types, it returns an empty string.
func (t Type) String() string {
	s := t.kind.String()
	switch t.kind {
	case IntKind, FloatKind:
		s += "(" + strconv.Itoa(bitSize[t.size]) + ")"
	case DecimalKind:
		if t.p > 0 {
			s += "(" + strconv.Itoa(int(t.p)) + "," + strconv.Itoa(int(t.s)) + ")"
		}
	case ArrayKind, MapKind:
		s += "(" + t.Elem().String() + ")"
	case InvalidKind:
		s, _ = t.vl.(string)
	}
	return s
}

// Kind returns the kind of t.
// Returns PtInvalid if t is not valid.
func (t Type) Kind() Kind {
	return t.kind
}

// BitSize returns the bit size of t as 8, 16, 24, 32 or 64. t must be an int or
// float type, otherwise it panics.
func (t Type) BitSize() int {
	if t.kind != IntKind && t.kind != FloatKind {
		panic("type is not an int or float type")
	}
	return bitSize[t.size]
}

// Unsigned returns t as an unsigned int with the maximum range of values based
// in its bit size. It must be an int type and should not be already unsigned
// otherwise it panics.
func (t Type) Unsigned() Type {
	if t.kind != IntKind {
		panic("type is not an int type")
	}
	if t.unsigned {
		panic("type is already unsigned")
	}
	t.p = 0
	switch bitSize[t.size] {
	case 8:
		t.s = int32(MaxUint8)
	case 16:
		t.s = int32(MaxUint16)
	case 24:
		t.s = int32(MaxUint24)
	case 32:
		s := MaxUint32
		t.s = int32(s)
	case 64:
		t.s = 0
	}
	t.vl = nil
	t.unsigned = true
	return t
}

// IsUnsigned reports whether t is unsigned. Panics if t is not an int type.
func (t Type) IsUnsigned() bool {
	if t.kind != IntKind {
		panic("type is not an int type")
	}
	return t.unsigned
}

type intRange struct{ min, max int64 }

// IntRange returns the minimum and maximum value for t. t must be an int type,
// otherwise it panics.
func (t Type) IntRange() (min, max int64) {
	if t.kind != IntKind {
		panic("type is not an int type")
	}
	if t.unsigned {
		panic("cannot get int range for an unsigned int")
	}
	if t.size < 4 {
		// 8, 16, 24, and 32 bits.
		return int64(t.p), int64(t.s)
	}
	// 64 bits.
	if i, ok := t.vl.(intRange); ok {
		return i.min, i.max
	}
	return MinInt64, MaxInt64
}

// WithIntRange returns t but with values in [min,max]. t must be an int type.
// min cannot be greater than max. min and max must be within the range of
// values of t. It panics it previous restrictions are not met.
func (t Type) WithIntRange(min, max int64) Type {
	if t.kind != IntKind {
		panic("type is not an int type")
	}
	if t.unsigned {
		panic("cannot set int range for an unsigned int")
	}
	Min, Max := minInt[t.size], maxInt[t.size]
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
	if t.size < 4 {
		// 8, 16, 24, and 32 bits.
		t.p, t.s = int32(min), int32(max)
	} else {
		// 64 bits.
		t.vl = intRange{min, max}
	}
	return t
}

// UnsignedRange returns the minimum and maximum value for t. t must be an
// unsigned int type, otherwise it panics.
func (t Type) UnsignedRange() (min, max uint64) {
	if t.kind != IntKind {
		panic("type is not an int type")
	}
	if !t.unsigned {
		panic("type is not an unsigned int type")
	}
	if t.size < 4 {
		// 8, 16, 24, and 32 bits.
		return uint64(uint32(t.p)), uint64(uint32(t.s))
	}
	// 64 bits.
	if i, ok := t.vl.(intRange); ok {
		return uint64(i.min), uint64(i.max)
	}
	return 0, MaxUint64
}

// WithUnsignedRange returns t but with values in [min,max]. t must be an
// unsigned int type. min cannot be greater than max. min and max must be within
// the range of values of t. It panics it previous restrictions are not met.
func (t Type) WithUnsignedRange(min, max uint64) Type {
	if t.kind != IntKind {
		panic("type is not an int type")
	}
	if !t.unsigned {
		panic("type is not an unsigned int type")
	}
	Max := maxUnsigned[t.size]
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
	if t.size < 4 {
		// 8, 16, 24, and 32 bits.
		t.p, t.s = int32(min), int32(max)
	} else {
		// 64 bits.
		t.vl = intRange{int64(min), int64(max)}
	}
	return t
}

type floatRange struct {
	min, max   float64
	minS, maxS string
}

// FloatRange returns the minimum and maximum value of t. t must be a float
// type, otherwise it panics.
func (t Type) FloatRange() (min, max float64) {
	if t.kind != FloatKind {
		panic("type is not a float type")
	}
	if f, ok := t.vl.(floatRange); ok {
		return f.min, f.max
	}
	if t.real {
		if t.size == 3 {
			// 32 bits.
			return -math.MaxFloat32, math.MaxFloat32
		}
		// 64 bits.
		return -math.MaxFloat64, math.MaxFloat64
	}
	return math.Inf(-1), math.Inf(1)
}

// WithFloatRange returns t but with values in [min,max]. t must be a float
// type. min cannot be greater than max. min and max cannot be NaN, and if r is
// real they cannot be ±Inf. It panics if previous restrictions are not met.
func (t Type) WithFloatRange(min, max float64) Type {
	if t.kind != FloatKind {
		panic("type is not a float type")
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
	if t.size == 3 {
		// 32 bits.
		min, max = float64(float32(min)), float64(float32(max))
		if !math.IsInf(min, -1) {
			minS = strconv.FormatFloat(min, 'f', -1, 32)
		}
		if !math.IsInf(max, 1) {
			maxS = strconv.FormatFloat(max, 'f', -1, 32)
		}
	} else {
		// 64 bits.
		if !math.IsInf(min, -1) {
			minS = strconv.FormatFloat(min, 'f', -1, 64)
		}
		if !math.IsInf(max, 1) {
			maxS = strconv.FormatFloat(max, 'f', -1, 64)
		}
	}
	t.vl = floatRange{min: min, max: max, minS: minS, maxS: maxS}
	return t
}

type decimalRange struct{ min, max decimal.Decimal }

// DecimalRange returns the minimum and maximum value for t. t must be a
// decimal type, otherwise it panics.
func (t Type) DecimalRange() (min, max decimal.Decimal) {
	if t.kind != DecimalKind {
		panic("type is not a decimal type")
	}
	dr := t.vl.(decimalRange)
	return dr.min, dr.max
}

// WithDecimalRange returns t with values constrained to the range [min, max].
// t must be of type decimal, min must be less than or equal to max, and both
// min and max must fit within the precision and scale of t; otherwise, it
// panics.
func (t Type) WithDecimalRange(min, max decimal.Decimal) Type {
	if t.kind != DecimalKind {
		panic("type is not a decimal type")
	}
	if max.Less(min) {
		panic("max cannot be less than min")
	}
	dr := t.vl.(decimalRange)
	minOverflow, maxOverflow := overflow(min, dr.min, dr.max), overflow(max, dr.min, dr.max)
	if minOverflow || maxOverflow {
		typeMin, typeMax := decimal.Range(t.Precision(), t.Scale())
		if overflow(min, typeMin, typeMax) {
			panic(fmt.Sprintf("min must be in range [%s,%s]", typeMin, typeMax))
		}
		if overflow(max, typeMin, typeMax) {
			panic(fmt.Sprintf("max must be in range [%s,%s]", typeMin, typeMax))
		}
	}
	t.vl = decimalRange{min, max}
	return t
}

// Precision returns the precision of a decimal type.
// Panics if t is not a decimal type.
func (t Type) Precision() int {
	if t.kind != DecimalKind {
		panic("cannot get precision of a non-decimal type")
	}
	return int(t.p)
}

// Scale returns the scale of a decimal type.
// Panics if t is not a decimal type.
func (t Type) Scale() int {
	if t.kind != DecimalKind {
		panic("cannot get scale of a non-decimal type")
	}
	return int(t.s)
}

// MaxBytes returns the maximum number of bytes allowed for the string type t,
// along with true. If t has no maximum length, it returns 0 and false.
// It panics if t is not a string type.
func (t Type) MaxBytes() (int, bool) {
	if t.kind != StringKind {
		panic("cannot get max bytes of a non-string type")
	}
	return int(uint32(t.p)), t.p != 0
}

// WithMaxBytes returns t configured with a maximum of n bytes. n must be in the
// range [1, MaxStringLen].
// It panics if t is not a string type, if t already specifies a maximum number
// of bytes, if t already has values, or if n is out of range.
func (t Type) WithMaxBytes(n int) Type {
	if t.kind != StringKind {
		panic("cannot set max byte length of a non-string type")
	}
	if t.p > 0 {
		panic("max bytes already specified")
	}
	if n < 1 || MaxStringLen < n {
		panic("invalid max bytes")
	}
	if _, ok := t.vl.([]string); ok {
		panic("t already has values")
	}
	t.p = int32(uint32(n))
	return t
}

// MaxLength returns the maximum length in characters of a string type and true.
// If t has no maximum length in characters, it returns 0 and false. Panics if t
// is not a string type.
func (t Type) MaxLength() (int, bool) {
	if t.kind != StringKind {
		panic("cannot get max length of non-string types")
	}
	return int(uint32(t.s)), t.s != 0
}

// WithMaxLength returns t with a maximum length of l of a string type. l must
// be in range [1, MaxStringLen]. Panics if t is not a string type, or if l is
// not in range, or if t has already a char length, or if t already has values.
func (t Type) WithMaxLength(l int) Type {
	if t.kind != StringKind {
		panic("cannot set max length of non-string types")
	}
	if t.s > 0 {
		panic("repeated length in characters")
	}
	if l < 1 || MaxStringLen < l {
		panic("invalid string length")
	}
	if _, ok := t.vl.([]string); ok {
		panic("t already has values")
	}
	t.s = int32(uint32(l))
	return t
}

// Pattern returns the pattern of t. If t has no pattern, it returns nil.
// Panics if t is not a string type.
func (t Type) Pattern() *regexp.Regexp {
	if t.kind != StringKind {
		panic("cannot return pattern for a non-string type")
	}
	re, _ := t.vl.(*regexp.Regexp)
	return re
}

// WithPattern returns t with the pattern p.
// Panics if t is not a string type, or t has already apattern or has values.
func (t Type) WithPattern(p *regexp.Regexp) Type {
	if t.kind != StringKind {
		panic("cannot set pattern for a non-string type")
	}
	switch t.vl.(type) {
	case []string:
		panic("cannot set pattern when t has values")
	case *regexp.Regexp:
		panic("t already has a pattern")
	}
	t.vl = p
	return t
}

// Values returns the values of t. Returns nil if t has no values. Panics if t
// is not a string type.
func (t Type) Values() []string {
	if t.kind != StringKind {
		panic("cannot get values for a non-string type")
	}
	if vl, ok := t.vl.([]string); ok {
		values := make([]string, len(vl))
		copy(values, vl)
		return values
	}
	return nil
}

// WithValues returns t but restricted to some values. t must be a string type.
// It panics if t is not of string type, if the values is empty or contains an
// invalid UTF-8 string, or if t already has values, a regular expression, or
// if it is already restricted by byte or character length.
func (t Type) WithValues(values ...string) Type {
	if t.kind != StringKind {
		panic("cannot set values for a non-string type")
	}
	if len(values) == 0 {
		panic("values is empty")
	}
	switch t.vl.(type) {
	case []string:
		panic("t already has values")
	case *regexp.Regexp:
		panic("t already has a pattern")
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

// MinElements returns the minimum number of elements of t. t must be an array,
// otherwise it panics.
func (t Type) MinElements() int {
	if t.kind != ArrayKind {
		panic("cannot get the minimum number of elements of a non-array type")
	}
	return int(t.p)
}

// WithMinElements returns t but with the minimum number of elements sets to
// min. t must be an array. Panics if t is not an array type or min is not in
// [0,max] where max is the maximum number of elements of t.
func (t Type) WithMinElements(min int) Type {
	if t.kind != ArrayKind {
		panic("cannot set the minimum number of elements for a non-array type")
	}
	if min < 0 || min > int(t.s) {
		panic(fmt.Sprintf("minimum number of elements not in [0,%d]", t.s))
	}
	t.p = int32(min)
	return t
}

// MaxElements returns the maximum number of elements of t. t must be an array,
// otherwise it panics.
func (t Type) MaxElements() int {
	if t.kind != ArrayKind {
		panic("cannot get the maximum number of elements of a non-array type")
	}
	return int(t.s)
}

// WithMaxElements returns t but with the maximum number of elements sets to
// max. t must be an array. Panics if t is not an array type or max is not in
// range [min,MaxElements] where min is the minimum number of elements of t.
func (t Type) WithMaxElements(max int) Type {
	if t.kind != ArrayKind {
		panic("cannot set the maximum number of elements for a non-array type")
	}
	if max < int(t.p) || max > MaxElements {
		panic(fmt.Sprintf("maximum number of elements not in [%d,%d]", t.p, MaxElements))
	}
	t.s = int32(max)
	return t
}

// Unique reports whether the elements of t are unique.
// Panics if t is not an array.
func (t Type) Unique() bool {
	if t.kind != ArrayKind {
		panic("cannot get unique of a non-array type")
	}
	return t.unique
}

// WithUnique returns the type t but with unique elements. t must be an array
// and its element type cannot be json, array, map, or object.
// Panics if t is not an array or the element type is json, array, map, or
// object.
func (t Type) WithUnique() Type {
	if t.kind != ArrayKind {
		panic("cannot set unique of a non-array type")
	}
	if k := t.vl.(Type).kind; k == JSONKind || k == ArrayKind || k == MapKind || k == ObjectKind {
		panic("cannot set unique for an array with elements of type array, map, or object")
	}
	t.unique = true
	return t
}

// Properties returns the properties of t.
// It panics if t is not an object.
func (t Type) Properties() Properties {
	if t.kind != ObjectKind {
		panic("cannot get properties of a non-object type")
	}
	return t.vl.(Properties)
}

// Elem returns a type's element type.
// Panics if t is not an array or map type.
func (t Type) Elem() Type {
	if t.kind != ArrayKind && t.kind != MapKind {
		panic("cannot get the element type for a non-array and non-map type")
	}
	return t.vl.(Type)
}

// normalizedUTF8 returns s as a normalized UTF-8 encoded string. Returns an
// error if s is not a valid UTF-8 encoded string or contains a NUL byte.
func normalizedUTF8(s string) (string, error) {
	if s == "" {
		return s, nil
	}
	if !utf8.ValidString(s) {
		return "", errors.New("invalid UTF-8 encoding")
	}
	if strings.Contains(s, "\x00") {
		return "", errors.New("contains NUL byte")
	}
	return norm.NFC.String(s), nil
}

// overflow reports whether v < min or v > max.
func overflow(v, min, max decimal.Decimal) bool {
	return v.Less(min) || v.GreaterEqual(max)
}
