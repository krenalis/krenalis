//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

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

	"github.com/shopspring/decimal"
	"golang.org/x/text/unicode/norm"
)

// Seq2 represents a sequence of K and V values.
type Seq2[K, V any] func(yield func(K, V) bool)

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

// PathNotExistError is returned by PropertyByPath when the path does not exist.
type PathNotExistError struct {
	Path string
}

func (err PathNotExistError) Error() string {
	return fmt.Sprintf("property path %q does not exist", err.Path)
}

// one is the decimal.Decimal 1.
var one = decimal.New(1, 0)

var (
	bitSize = [...]int{8, 16, 24, 32, 64}
	minInt  = [...]int64{MinInt8, MinInt16, MinInt24, MinInt32, MinInt64}
	maxInt  = [...]int64{MaxInt8, MaxInt16, MaxInt24, MaxInt32, MaxInt64}
	maxUint = [...]uint64{MaxUint8, MaxUint16, MaxUint24, MaxUint32, MaxUint64}
)

var (
	MaxDecimal = decimal.New(1, MaxDecimalPrecision).Sub(one)
	MinDecimal = MaxDecimal.Neg()
)

const (
	MaxDecimalPrecision = 76             // Maximum precision for a Decimal type
	MaxDecimalScale     = 37             // Maximum scale for a Decimal type
	MaxItems            = math.MaxInt32  // Maximum number of items of an Array type
	MaxTextLen          = math.MaxUint32 // Maximum length in bytes and characters for a Text type
	MaxYear             = 9999           // Maximum year for DataTime, Date and Year types
	MinYear             = 1              // Minimum year for DataTime, Date and Year types

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
	BooleanKind
	IntKind
	UintKind
	FloatKind
	DecimalKind
	DateTimeKind
	DateKind
	TimeKind
	YearKind
	UUIDKind
	JSONKind
	InetKind
	TextKind
	ArrayKind
	ObjectKind
	MapKind
)

var kindName = []string{
	"Boolean",
	"Int",
	"Uint",
	"Float",
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

// String returns the name of k. Panics if k is not a kind.
func (k Kind) String() string {
	if !k.Valid() {
		panic("invalid kind")
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
	Placeholder string
	Role        Role
	Type        Type
	Required    bool
	Nullable    bool
	Note        string
}

var _ interface {
	json.Marshaler
	json.Unmarshaler
} = (*Property)(nil)

// Type represents a type.
type Type struct {
	kind Kind

	size int8 // size for Int, Uint and Float: 0 (8 bits), 1 (16 bits), 2 (24 bits), 3 (32 bits), and 4 (64 bits)

	unique bool // unique reports whether the items of an Array must be unique.
	real   bool // real reports whether NaN, +Inf and -Inf are not allowed for Float.

	// p represents
	//   - minimum value for Int with 8, 16, 24, and 32 bits
	//   - minimum value, as uint32(p), for Uint with 8, 16, 24, and 32 bits
	//   - precision for Decimal
	//   - length in bytes, as uint32(p), for Text
	//   - minimum length for Array
	p int32

	// s represents
	//   - maximum value for Int with 8, 16, 24, and 32 bits
	//   - maximum value, as uint32(s), for Uint with 8, 16, 24, and 32 bits
	//   - scale for Decimal
	//   - length in characters, as uint32(s), for Text
	//   - maximum length for Array
	s int32

	// vl can contain one of
	//   - intRange value for Int with 64 bits
	//   - uintRange value for Uint with 64 bits
	//   - floatRange value for Float
	//   - decimalRange value for Decimal
	//   - *regexp.Regexp value for Text
	//   - []string with the values for Text
	//   - []Property for Object
	//   - Type of the item for Array
	//   - Type of the value for Map
	vl any

	_ []int // make Type non-comparable.
}

var _ interface {
	json.Marshaler
	json.Unmarshaler
} = (*Type)(nil)

// Boolean returns the Boolean type.
func Boolean() Type {
	return Type{kind: BooleanKind}
}

// Int returns the Int type with the provided bit size. It panics if size is not
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

// Uint returns the Uint type with the provided bit size. It panics if size is
// not 8, 16, 24, 32 or 64.
func Uint(size int) Type {
	t := Type{kind: UintKind}
	switch size {
	case 8:
		t.s = int32(MaxUint8)
	case 16:
		t.size = 1
		t.s = int32(MaxUint16)
	case 24:
		t.size = 2
		t.s = int32(MaxUint24)
	case 32:
		t.size = 3
		s := MaxUint32
		t.s = int32(s)
	case 64:
		t.size = 4
	default:
		panic("bit size is not valid")
	}
	return t
}

// Float returns the Float type with the provided bit size. It panics if size is
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

// Decimal returns the Decimal type with the given precision and scale.
// Panics if precision is not in range [1,MaxDecimalPrecision] or if scale is
// not in range [0,MaxDecimalScale] or if it is greater that precision.
// As a special case if both precision and scale are zero, the type has no
// precision and scale.
func Decimal(precision, scale int) Type {
	if precision == 0 && scale == 0 {
		return Type{kind: DecimalKind}
	}
	if precision < 1 || precision > MaxDecimalPrecision {
		panic("invalid decimal precision")
	}
	if scale < 0 || scale > MaxDecimalScale || scale > precision {
		panic("invalid decimal scale")
	}
	return Type{kind: DecimalKind, p: int32(precision), s: int32(scale)}
}

// DateTime returns the DateTime type.
func DateTime() Type {
	return Type{kind: DateTimeKind}
}

// Date returns the Date type.
func Date() Type {
	return Type{kind: DateKind}
}

// Time returns the Time type.
func Time() Type {
	return Type{kind: TimeKind}
}

// Year returns the Year type.
func Year() Type {
	return Type{kind: YearKind}
}

// UUID returns the UUID type.
func UUID() Type {
	return Type{kind: UUIDKind}
}

// JSON returns the JSON type.
func JSON() Type {
	return Type{kind: JSONKind}
}

// Inet returns the Inet type.
func Inet() Type {
	return Type{kind: InetKind}
}

// Text returns a Text type.
func Text() Type {
	return Type{kind: TextKind}
}

// Array returns an Array type with items of type t.
func Array(t Type) Type {
	return Type{kind: ArrayKind, s: MaxItems, vl: t}
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
	return Type{kind: MapKind, vl: t}
}

// ObjectOf is like Object but returns an error instead of panicking if any.
// It returns an InvalidPropertyNameError error if a property name is not
// valid, and a RepeatedPropertyNameError error if a property name is repeated.
func ObjectOf(properties []Property) (Type, error) {
	if len(properties) == 0 {
		return Type{}, errors.New("no property in type")
	}
	indexOf := make(map[string]int, len(properties))
	ps := make([]Property, len(properties))
	for i, property := range properties {
		if property.Name == "" {
			return Type{}, errors.New("property name is empty")
		}
		if !IsValidPropertyName(property.Name) {
			return Type{}, InvalidPropertyNameError{i, property.Name}
		}
		if j, ok := indexOf[property.Name]; ok {
			return Type{}, RepeatedPropertyNameError{j, i, property.Name}
		}
		indexOf[property.Name] = i
		label, err := normalizedUTF8(property.Label)
		if err != nil {
			return Type{}, err
		}
		placeholder, err := normalizedUTF8(property.Placeholder)
		if err != nil {
			return Type{}, err
		}
		if property.Role < BothRole || property.Role > DestinationRole {
			return Type{}, errors.New("invalid property role")
		}
		if !property.Type.Valid() {
			return Type{}, errors.New("invalid property type")
		}
		note, err := normalizedUTF8(property.Note)
		if err != nil {
			return Type{}, err
		}
		ps[i] = Property{
			Name:        property.Name,
			Label:       label,
			Placeholder: placeholder,
			Role:        property.Role,
			Type:        property.Type,
			Required:    property.Required,
			Nullable:    property.Nullable,
			Note:        note,
		}
	}
	return Type{kind: ObjectKind, vl: ps}, nil
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
	if t.kind != ObjectKind {
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
	return Type{kind: ObjectKind, vl: roleProperties}
}

// AsReal returns t but as a real number. As a real number, t does not allow
// NaN, +Inf and -Inf values. t must be a Float type. t cannot be already real
// and cannot have a range. It panics if previous restrictions are not met.
func (t Type) AsReal() Type {
	if t.kind != FloatKind {
		panic("type is not a Float type")
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

// IsReal reports whether t is real. Panics if t is not a Float type.
func (t Type) IsReal() bool {
	if t.kind != FloatKind {
		panic("type is not a Float type")
	}
	return t.real
}

// Valid indicates if t is valid.
func (t Type) Valid() bool {
	return t.kind != InvalidKind
}

// String returns a string representation of t.
// Panics if t is not a valid type.
func (t Type) String() string {
	s := t.kind.String()
	switch t.kind {
	case IntKind, UintKind, FloatKind:
		s += "(" + strconv.Itoa(bitSize[t.size]) + ")"
	case DecimalKind:
		if t.p > 0 {
			s += "(" + strconv.Itoa(int(t.p)) + "," + strconv.Itoa(int(t.s)) + ")"
		}
	case TextKind:
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

// Kind returns the kind of t.
// Returns PtInvalid if t is not valid.
func (t Type) Kind() Kind {
	return t.kind
}

// BitSize returns the bit size of t as 8, 16, 24, 32 or 64. t must be an Int,
// Uint or Float type, otherwise it panics.
func (t Type) BitSize() int {
	if t.kind != IntKind && t.kind != UintKind && t.kind != FloatKind {
		panic("type is not an Int, Uint or Float type")
	}
	return bitSize[t.size]
}

type intRange struct{ min, max int64 }

// IntRange returns the minimum and maximum value for t. t must be an Int type,
// otherwise it panics.
func (t Type) IntRange() (min, max int64) {
	if t.kind != IntKind {
		panic("type is not an Int type")
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

// WithIntRange returns t but with values in [min,max]. t must be an Int type.
// min cannot be greater than max. min and max must be within the range of
// values of t. It panics it previous restrictions are not met.
func (t Type) WithIntRange(min, max int64) Type {
	if t.kind != IntKind {
		panic("type is not an Int type")
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

type uintRange struct{ min, max uint64 }

// UintRange returns the minimum and maximum value for t. t must be an Uint
// type, otherwise it panics.
func (t Type) UintRange() (min, max uint64) {
	if t.kind != UintKind {
		panic("type is not an Uint type")
	}
	if t.size < 4 {
		// 8, 16, 24, and 32 bits.
		return uint64(uint32(t.p)), uint64(uint32(t.s))
	}
	// 64 bits.
	if i, ok := t.vl.(uintRange); ok {
		return i.min, i.max
	}
	return 0, MaxUint64
}

// WithUintRange returns t but with values in [min,max]. t must be an Uint type.
// min cannot be greater than max. min and max must be within the range of
// values of t. It panics it previous restrictions are not met.
func (t Type) WithUintRange(min, max uint64) Type {
	if t.kind != UintKind {
		panic("type is not an Uint type")
	}
	Max := maxUint[t.size]
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
		t.vl = uintRange{min, max}
	}
	return t
}

type floatRange struct {
	min, max   float64
	minS, maxS string
}

// FloatRange returns the minimum and maximum value of t. t must be a Float
// type, otherwise it panics.
func (t Type) FloatRange() (min, max float64) {
	if t.kind != FloatKind {
		panic("type is not a Float type")
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

// WithFloatRange returns t but with values in [min,max]. t must be a Float
// type. min cannot be greater than max. min and max cannot be NaN, and if r is
// real they cannot be ±Inf. It panics if previous restrictions are not met.
func (t Type) WithFloatRange(min, max float64) Type {
	if t.kind != FloatKind {
		panic("type is not a Float type")
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
			minS = decimal.NewFromFloat32(float32(min)).String()
		}
		if !math.IsInf(max, 1) {
			maxS = decimal.NewFromFloat32(float32(max)).String()
		}
	} else {
		// 64 bits.
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
	if t.kind != DecimalKind {
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
	if t.kind != DecimalKind {
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
	if t.kind != DecimalKind {
		panic("cannot get precision of a non-Decimal type")
	}
	return int(t.p)
}

// Scale returns the scale of a Decimal type.
// Panics if t is not a Decimal type.
func (t Type) Scale() int {
	if t.kind != DecimalKind {
		panic("cannot get scale of a non-Decimal type")
	}
	return int(t.s)
}

// ByteLen returns the maximum length in bytes of a Text type and true.
// If t has no maximum length in bytes, it returns 0 and false.
// Panics if t is not a Text type.
func (t Type) ByteLen() (int, bool) {
	if t.kind != TextKind {
		panic("cannot get byte length of a non-Text type")
	}
	return int(uint32(t.p)), t.p != 0
}

// WithByteLen returns t with a maximum length of l of a Text type. l must be in
// range [1, MaxTextLen].
// Panics if t is not a Text type, or if l is not in range, or if t has already
// a byte length, or if t already has values.
func (t Type) WithByteLen(l int) Type {
	if t.kind != TextKind {
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

// CharLen returns the maximum length in characters of a Text type and true. If
// t has no maximum length in characters, it returns 0 and false. Panics if t is
// not a Text type.
func (t Type) CharLen() (int, bool) {
	if t.kind != TextKind {
		panic("cannot get character length of non-Text types")
	}
	return int(uint32(t.s)), t.s != 0
}

// WithCharLen returns t with a maximum length of l of a Text type. l must be in
// range [1, MaxTextLen]. Panics if t is not a Text type, or if l is not in
// range, or if t has already a char length, or if t already has values.
func (t Type) WithCharLen(l int) Type {
	if t.kind != TextKind {
		panic("cannot set character length of non-Text types")
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
	if t.kind != TextKind {
		panic("cannot return regular expression for a non-Text type")
	}
	re, _ := t.vl.(*regexp.Regexp)
	return re
}

// WithRegexp returns t with the regular expression re.
// Panics if t is not a Text type, or t has already a regular expression or has
// values.
func (t Type) WithRegexp(re *regexp.Regexp) Type {
	if t.kind != TextKind {
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
	if t.kind != TextKind {
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
	if t.kind != TextKind {
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
	if t.kind != ArrayKind {
		panic("cannot get the minimum number of items of a non-Array type")
	}
	return int(t.p)
}

// WithMinItems returns t but with the minimum number of items sets to min.
// t must be an Array. Panics if t is not an Array type or min is not in
// [0,max] where max is the maximum number of items of t.
func (t Type) WithMinItems(min int) Type {
	if t.kind != ArrayKind {
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
	if t.kind != ArrayKind {
		panic("cannot get the maximum number of items of a non-Array type")
	}
	return int(t.s)
}

// WithMaxItems returns t but with the maximum number of items sets to max.
// t must be an Array. Panics if t is not an Array type or max is not in
// [min,MaxItems] where min is the minimum number of items of t.
func (t Type) WithMaxItems(max int) Type {
	if t.kind != ArrayKind {
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
	if t.kind != ArrayKind {
		panic("cannot get unique of a non-Array type")
	}
	return t.unique
}

// WithUnique returns the type t but with unique items. t must be an Array and
// its item type cannot be JSON, Array, Map, or Object.
// Panics if t is not an Array or the item type is JSON, Array, Map, or Object.
func (t Type) WithUnique() Type {
	if t.kind != ArrayKind {
		panic("cannot set unique of a non-Array type")
	}
	if k := t.vl.(Type).kind; k == JSONKind || k == ArrayKind || k == MapKind || k == ObjectKind {
		panic("cannot set unique for an Array with items of type Array, Map, or Object")
	}
	t.unique = true
	return t
}

// PropertyByPath returns the property with the given path, or a
// PathNotExistError error if the property does not exist.
// Panics if t is not an Object type or path is not a valid path.
func (t Type) PropertyByPath(path string) (Property, error) {
	if t.kind != ObjectKind {
		panic("cannot get the properties of a non-Object type")
	}
	name, rest := "", path
Rest:
	for {
		name, rest, _ = strings.Cut(rest, ".")
		if t.kind != ObjectKind {
			break
		}
		properties := t.vl.([]Property)
		for j := 0; j < len(properties); j++ {
			if properties[j].Name != name {
				continue
			}
			if rest == "" {
				return properties[j], nil
			}
			t = properties[j].Type
			continue Rest
		}
		break
	}
	if !IsValidPropertyPath(path) {
		panic("invalid property path")
	}
	path = strings.TrimSuffix(strings.TrimSuffix(path, rest), ".")
	return Property{}, PathNotExistError{path}
}

// Property returns the property with the given name and a boolean value
// indicating if the property exists.
// Panics if t is not an Object type or name is not a valid property name.
func (t Type) Property(name string) (Property, bool) {
	if t.kind != ObjectKind {
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

// Properties returns an iterator over the properties of t.
// It panics if t is not an Object.
func (t Type) Properties() Seq2[int, Property] {
	if t.kind != ObjectKind {
		panic("cannot iterate over a non-Object type")
	}
	return func(yield func(i int, property Property) bool) {
		pp := t.vl.([]Property)
		for i := 0; i < len(pp); i++ {
			if !yield(i, pp[i]) {
				return
			}
		}
	}
}

// NumProperties returns the count of properties in t.
// Panics if t is not an Object type.
func (t Type) NumProperties() int {
	if t.kind != ObjectKind {
		panic("cannot get the properties of a non-Object type")
	}
	return len(t.vl.([]Property))
}

// Elem returns a type's element type.
// Panics if t is not an Array or Map type.
func (t Type) Elem() Type {
	if t.kind != ArrayKind && t.kind != MapKind {
		panic("cannot get the element type for a non-Array and non-Map type")
	}
	return t.vl.(Type)
}

// EqualTo reports whether t is equals to t2.
func (t Type) EqualTo(t2 Type) bool {
	almostEqual := t.kind == t2.kind && t.size == t2.size && t.unique == t2.unique && t.real == t2.real && t.p == t2.p && t.s == t2.s
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
				p1.Placeholder != p2.Placeholder ||
				p1.Role != p2.Role ||
				p1.Required != p2.Required ||
				p1.Nullable != p2.Nullable ||
				p1.Note != p2.Note ||
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

// Properties returns the properties of the Object type t.
// Panics if t is not an Object type.
func Properties(t Type) []Property {
	if t.kind != ObjectKind {
		panic("cannot get the properties of a non-Object type")
	}
	return slices.Clone(t.vl.([]Property))
}

// PropertyNames returns the names of the properties of the Object t.
// Panics if t is not an Object type.
func PropertyNames(t Type) []string {
	if t.kind != ObjectKind {
		panic("cannot get the property names of a non-Object type")
	}
	pp := t.vl.([]Property)
	names := make([]string, len(pp))
	for i := 0; i < len(pp); i++ {
		names[i] = pp[i].Name
	}
	return names
}

// SubsetFunc returns a subset of the object t, including only the properties
// for which f returns true, maintaining their original order in type t.
// If f returns false for all properties, it returns an invalid schema.
// It panics if t is not an object, or f is nil
func SubsetFunc(t Type, f func(p Property) bool) Type {
	if t.kind != ObjectKind {
		panic("cannot get a subset of a non-Object type")
	}
	var ps []Property
	pp := t.vl.([]Property)
	for i := 0; i < len(pp); i++ {
		if f(pp[i]) {
			ps = append(ps, pp[i])
		}
	}
	if ps == nil {
		return Type{}
	}
	return Type{kind: ObjectKind, vl: ps}
}

// Walk returns an iterator over all the properties in t in a depth-first order.
//
// For example:
//
//	Walk(func(path string, p Property) bool {
//	    fmt.Printf("%s: %s\n", path, p.Type.Kind)
//	    return true
//	})
//
// If a property "x" has type Array(Object) or Map(Object) and the object has
// the property "y", its path is "x.y".
//
// It panics if t is not an Object.
func Walk(t Type) Seq2[string, Property] {
	if t.kind != ObjectKind {
		panic("cannot iterate over a non-Object type")
	}
	return func(yield func(path string, property Property) bool) {
		type entry struct {
			base string
			prop *Property
		}
		properties := t.vl.([]Property)
		n := len(properties)
		pp := make([]entry, n)
		for i := 0; i < n; i++ {
			pp[i].prop = &properties[n-1-i]
		}
		for len(pp) > 0 {
			var e entry
			n := len(pp)
			e, pp = pp[n-1], pp[:n-1]
			t := e.prop.Type
			for t.kind == MapKind || t.kind == ArrayKind {
				t = t.Elem()
			}
			if t.kind == ObjectKind {
				properties := t.vl.([]Property)
				for i := len(properties) - 1; i >= 0; i-- {
					pp = append(pp, entry{base: e.base + e.prop.Name + ".", prop: &properties[i]})
				}
			}
			if !yield(e.base+e.prop.Name, *e.prop) {
				return
			}
		}
	}
}

// normalizedUTF8 returns s as a normalized UTF-8 encoded string.
// Returns an error if s is not a valid UTF-8 encoded string.
func normalizedUTF8(s string) (string, error) {
	if !utf8.ValidString(s) {
		return "", errors.New("invalid UTF-8 encoding")
	}
	return norm.NFC.String(s), nil
}
