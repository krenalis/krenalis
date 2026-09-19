// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"fmt"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/validation"
)

// Semantic identifies what a type's values represent.
type Semantic int8

const (
	NoSemantic          Semantic = iota // no semantic
	CountrySemantic                     // country
	DurationSemantic                    // duration
	EmailSemantic                       // email address
	MeasurementSemantic                 // numeric measurement
	MoneySemantic                       // monetary amount
	PercentageSemantic                  // percentage
	PhoneSemantic                       // phone number
	URLSemantic                         // web URL
)

// semanticName contains the JSON names of all semantics except NoSemantic.
var semanticName = []string{
	"country",
	"duration",
	"email",
	"measurement",
	"money",
	"percentage",
	"phone",
	"url",
}

// SemanticByName returns the semantic identified by name. It returns
// NoSemantic, false if name does not identify a semantic. NoSemantic represents
// the absence of a semantic and cannot be looked up by name.
func SemanticByName(name string) (Semantic, bool) {
	for i, n := range semanticName {
		if n == name {
			return Semantic(i + 1), true
		}
	}
	return NoSemantic, false
}

// String returns the name of s.
func (s Semantic) String() string {
	if s == NoSemantic {
		return "none"
	}
	return semanticName[s-1]
}

// CountryFormat identifies how a country value is represented.
type CountryFormat int8

const (
	ISO3166Alpha2 CountryFormat = iota + 2 // two-letter ISO 3166-1 alpha-2 code
	ISO3166Alpha3                          // three-letter ISO 3166-1 alpha-3 code
)

// countryFormatName contains the JSON names of all valid country formats.
var countryFormatName = [...]string{
	"alpha-2",
	"alpha-3",
}

// CountryFormatByName returns the country format with the given name.
// The second return parameter reports whether a country format with the given
// name exists.
func CountryFormatByName(name string) (CountryFormat, bool) {
	for i, n := range countryFormatName {
		if n == name {
			return CountryFormat(i + 2), true
		}
	}
	return CountryFormat(0), false
}

// String returns the name of f.
func (f CountryFormat) String() string {
	if f != ISO3166Alpha2 && f != ISO3166Alpha3 {
		return "Invalid"
	}
	return countryFormatName[f-2]
}

// DurationUnit identifies the unit used to represent a duration.
type DurationUnit int8

const (
	InvalidDurationUnit DurationUnit = iota // does not identify a duration unit
	Millisecond                             // millisecond
	Second                                  // second
	Minute                                  // minute
	Hour                                    // hour
	Day                                     // day
	Week                                    // week
)

// durationUnitName contains the JSON names of all valid duration units.
var durationUnitName = [...]string{
	"millisecond",
	"second",
	"minute",
	"hour",
	"day",
	"week",
}

// DurationUnitByName returns the duration unit with the given name. The second
// return parameter reports whether a duration unit with the given name exists.
func DurationUnitByName(name string) (DurationUnit, bool) {
	for i, n := range durationUnitName {
		if n == name {
			return DurationUnit(i + 1), true
		}
	}
	return InvalidDurationUnit, false
}

// String returns the name of u.
func (u DurationUnit) String() string {
	if u < 1 || int(u) > len(durationUnitName) {
		return "Invalid"
	}
	return durationUnitName[u-1]
}

// UnitOfMeasure identifies a unit of measure.
type UnitOfMeasure int8

const (
	InvalidUnitOfMeasure UnitOfMeasure = iota // does not identify a unit of measure
	Gram                                      // gram
	Kilogram                                  // kilogram
	Millimeter                                // millimeter
	Centimeter                                // centimeter
	Meter                                     // meter
	Kilometer                                 // kilometer
	Milliliter                                // milliliter
	Liter                                     // liter
	Byte                                      // byte
	Kilobyte                                  // kilobyte
	Megabyte                                  // megabyte
	Gigabyte                                  // gigabyte
	Celsius                                   // degree Celsius
	Fahrenheit                                // degree Fahrenheit
	Ounce                                     // ounce
	Pound                                     // pound
	Inch                                      // inch
	Foot                                      // foot
	Yard                                      // yard
	Mile                                      // mile
)

// unitOfMeasureName contains the JSON names of all valid units of measure.
var unitOfMeasureName = []string{
	"g",
	"kg",
	"mm",
	"cm",
	"m",
	"km",
	"mL",
	"L",
	"B",
	"kB",
	"MB",
	"GB",
	"°C",
	"°F",
	"oz",
	"lb",
	"in",
	"ft",
	"yd",
	"mi",
}

// UnitOfMeasureByName returns the unit of measure with the given name. The
// second return parameter reports whether a unit with the given name exists.
func UnitOfMeasureByName(name string) (UnitOfMeasure, bool) {
	for i, n := range unitOfMeasureName {
		if n == name {
			return UnitOfMeasure(i + 1), true
		}
	}
	return InvalidUnitOfMeasure, false
}

// String returns the name of u.
func (u UnitOfMeasure) String() string {
	if u < 1 || int(u) > len(unitOfMeasureName) {
		return "Invalid"
	}
	return unitOfMeasureName[u-1]
}

// AsCountry returns t with the country semantic using format. It panics if t is
// not a string type, if t already has a semantic or other string constraints,
// or if format is invalid.
func (t Type) AsCountry(format CountryFormat) Type {
	t, err := t.withSemantic(CountrySemantic, format)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsDuration returns t with the duration semantic using unit. It panics unless
// t is an int, decimal, or real float type without a semantic and unit is
// valid.
func (t Type) AsDuration(unit DurationUnit) Type {
	t, err := t.withSemantic(DurationSemantic, unit)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsEmail returns t with the email semantic. It panics unless t is a string
// type without a semantic.
func (t Type) AsEmail() Type {
	t, err := t.withSemantic(EmailSemantic, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsMeasurement returns t with the measurement semantic using unit. It panics
// unless t is an int, decimal, or real float type without a semantic and unit
// is valid.
func (t Type) AsMeasurement(unit UnitOfMeasure) Type {
	t, err := t.withSemantic(MeasurementSemantic, unit)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsMoney returns t with the money semantic. It panics unless t is a decimal
// type without a semantic.
func (t Type) AsMoney() Type {
	t, err := t.withSemantic(MoneySemantic, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsPercentage returns t with the percentage semantic. It panics unless t is a
// decimal type without a semantic.
func (t Type) AsPercentage() Type {
	t, err := t.withSemantic(PercentageSemantic, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsPhone returns t with the phone semantic. It panics if t is not a string
// type or if t already has a semantic or other string constraints.
func (t Type) AsPhone() Type {
	t, err := t.withSemantic(PhoneSemantic, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsURL returns t with the URL semantic. It panics unless t is a string type
// without a semantic.
func (t Type) AsURL() Type {
	t, err := t.withSemantic(URLSemantic, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// CountryFormat returns the country format of t. It panics unless t has the
// country semantic.
func (t Type) CountryFormat() CountryFormat {
	if t.semantic != CountrySemantic {
		panic("type does not have country semantic")
	}
	return t.semanticOption.(CountryFormat)
}

// Currency returns the currency code of t and whether one is set. It panics
// unless t has the money semantic.
func (t Type) Currency() (string, bool) {
	if t.semantic != MoneySemantic {
		panic("type does not have money semantic")
	}
	currency, _ := t.semanticOption.(string)
	return currency, currency != ""
}

// DurationUnit returns the duration unit of t. It panics unless t has the
// duration semantic.
func (t Type) DurationUnit() DurationUnit {
	if t.semantic != DurationSemantic {
		panic("type does not have duration semantic")
	}
	return t.semanticOption.(DurationUnit)
}

// Semantic returns the semantic of t, or NoSemantic if t has no semantic.
func (t Type) Semantic() Semantic {
	return t.semantic
}

// UnitOfMeasure returns the unit of measure of t. It panics unless t has the
// measurement semantic.
func (t Type) UnitOfMeasure() UnitOfMeasure {
	if t.semantic != MeasurementSemantic {
		panic("type does not have measurement semantic")
	}
	return t.semanticOption.(UnitOfMeasure)
}

// WithCurrency returns t with the given ISO 4217 currency code. It panics
// unless t has the money semantic and currency is valid.
func (t Type) WithCurrency(currency string) Type {
	if t.semantic != MoneySemantic {
		panic("type does not have money semantic")
	}
	if !validation.IsValidCurrencyCode(currency) {
		panic("invalid currency code")
	}
	t.semanticOption = currency
	return t
}

// withSemantic returns t with the given semantic and option, or an error if
// they are invalid or incompatible with t.
func (t Type) withSemantic(semantic Semantic, option any) (Type, error) {

	if semantic == NoSemantic {
		return Type{}, errors.New("semantic does not specialize type")
	}
	if t.semantic != NoSemantic {
		return Type{}, errors.New("type already has a semantic")
	}

	switch semantic {
	case EmailSemantic:
		if t.kind != StringKind {
			return Type{}, errors.New("email semantic requires string type")
		}
		if option != nil {
			return Type{}, errors.New("email semantic does not accept an option")
		}
	case PhoneSemantic:
		if t.kind != StringKind {
			return Type{}, errors.New("phone semantic requires string type")
		}
		if option != nil {
			return Type{}, errors.New("phone semantic does not accept an option")
		}
	case URLSemantic:
		if t.kind != StringKind {
			return Type{}, errors.New("URL semantic requires string type")
		}
		if option != nil {
			return Type{}, errors.New("URL semantic does not accept an option")
		}
	case CountrySemantic:
		if t.kind != StringKind {
			return Type{}, errors.New("country semantic requires string type")
		}
		format, ok := option.(CountryFormat)
		if !ok || format != ISO3166Alpha2 && format != ISO3166Alpha3 {
			return Type{}, errors.New("invalid country format")
		}
	case MoneySemantic:
		if t.kind != DecimalKind {
			return Type{}, errors.New("money semantic requires decimal type")
		}
		if option != nil {
			return Type{}, errors.New("money semantic does not accept an option")
		}
	case PercentageSemantic:
		if t.kind != DecimalKind {
			return Type{}, errors.New("percentage semantic requires decimal type")
		}
		if option != nil {
			return Type{}, errors.New("percentage semantic does not accept an option")
		}
	case MeasurementSemantic:
		if !semanticNumericType(t) {
			return Type{}, errors.New("measurement semantic requires an int, decimal, or real float type")
		}
		unit, ok := option.(UnitOfMeasure)
		if !ok || unit < 1 || int(unit) > len(unitOfMeasureName) {
			return Type{}, errors.New("invalid unit of measure")
		}
	case DurationSemantic:
		if !semanticNumericType(t) {
			return Type{}, errors.New("duration semantic requires an int, decimal, or real float type")
		}
		unit, ok := option.(DurationUnit)
		if !ok || unit < 1 || int(unit) > len(durationUnitName) {
			return Type{}, errors.New("invalid duration unit")
		}
	}
	if (semantic == CountrySemantic || semantic == PhoneSemantic) && (t.p != 0 || t.s != 0 || t.vl != nil) {
		return Type{}, fmt.Errorf("%s semantic cannot be combined with other string constraints", semantic)
	}

	t.semantic = semantic
	t.semanticOption = option

	return t, nil
}

// semanticNumericType reports whether t is an int, decimal, or real float type.
func semanticNumericType(t Type) bool {
	switch t.kind {
	case IntKind, DecimalKind:
		return true
	case FloatKind:
		return t.real
	default:
		return false
	}
}
