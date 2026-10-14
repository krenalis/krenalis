// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"errors"
	"fmt"

	"github.com/krenalis/krenalis/tools/validation"
)

// Semantics identifies what a type's values represent.
type Semantics int8

const (
	NoSemantics          Semantics = iota // no semantics
	CountrySemantics                      // country
	DurationSemantics                     // duration
	EmailSemantics                        // email address
	MeasurementSemantics                  // numeric measurement
	MoneySemantics                        // monetary amount
	PercentageSemantics                   // percentage
	PhoneSemantics                        // phone number
	URLSemantics                          // web URL
)

// semanticsName contains the JSON names of all semantics except NoSemantics.
var semanticsName = []string{
	"country",
	"duration",
	"email",
	"measurement",
	"money",
	"percentage",
	"phone",
	"url",
}

// SemanticsByName returns the semantics identified by name. It returns
// NoSemantics, false if name does not identify any semantics. NoSemantics
// represents the absence of semantics and cannot be looked up by name.
func SemanticsByName(name string) (Semantics, bool) {
	for i, n := range semanticsName {
		if n == name {
			return Semantics(i + 1), true
		}
	}
	return NoSemantics, false
}

// String returns the name of s.
func (s Semantics) String() string {
	if s == NoSemantics {
		return "none"
	}
	if s < 1 || int(s) > len(semanticsName) {
		return fmt.Sprintf("%%!Semantics(%d)", int(s))
	}
	return semanticsName[s-1]
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
	return 0, false
}

// String returns the name of f.
func (f CountryFormat) String() string {
	if f != ISO3166Alpha2 && f != ISO3166Alpha3 {
		return fmt.Sprintf("%%!CountryFormat(%d)", int(f))
	}
	return countryFormatName[f-2]
}

// DurationUnit identifies the unit used to represent a duration.
type DurationUnit int8

const (
	Millisecond DurationUnit = iota + 1 // millisecond
	Second                              // second
	Minute                              // minute
	Hour                                // hour
	Day                                 // day
	Week                                // week
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
	return DurationUnit(0), false
}

// String returns the name of u.
func (u DurationUnit) String() string {
	if u < 1 || int(u) > len(durationUnitName) {
		return fmt.Sprintf("%%!DurationUnit(%d)", int(u))
	}
	return durationUnitName[u-1]
}

// UnitOfMeasure identifies a unit of measure.
type UnitOfMeasure int8

const (
	Gram       UnitOfMeasure = iota + 1 // gram
	Kilogram                            // kilogram
	Millimeter                          // millimeter
	Centimeter                          // centimeter
	Meter                               // meter
	Kilometer                           // kilometer
	Milliliter                          // milliliter
	Liter                               // liter
	Byte                                // byte
	Kilobyte                            // kilobyte
	Megabyte                            // megabyte
	Gigabyte                            // gigabyte
	Celsius                             // degree Celsius
	Fahrenheit                          // degree Fahrenheit
	Ounce                               // ounce
	Pound                               // pound
	Inch                                // inch
	Foot                                // foot
	Yard                                // yard
	Mile                                // mile
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
	return 0, false
}

// String returns the name of u.
func (u UnitOfMeasure) String() string {
	if u < 1 || int(u) > len(unitOfMeasureName) {
		return fmt.Sprintf("%%!UnitOfMeasure(%d)", int(u))
	}
	return unitOfMeasureName[u-1]
}

// AsCountry returns t with country semantics using format. It panics if t is
// not a string type, if t already has semantics or any string constraints,
// or if format is invalid.
func (t Type) AsCountry(format CountryFormat) Type {
	t, err := t.withSemantics(CountrySemantics, format)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsDuration returns t with duration semantics using unit. It panics unless
// t is an int, decimal, or real float type without semantics and unit is
// valid.
func (t Type) AsDuration(unit DurationUnit) Type {
	t, err := t.withSemantics(DurationSemantics, unit)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsEmail returns t with email semantics. It panics unless t is a string
// type without semantics.
func (t Type) AsEmail() Type {
	t, err := t.withSemantics(EmailSemantics, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsMeasurement returns t with measurement semantics using unit. It panics
// unless t is an int, decimal, or real float type without semantics and unit
// is valid.
func (t Type) AsMeasurement(unit UnitOfMeasure) Type {
	t, err := t.withSemantics(MeasurementSemantics, unit)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsMoney returns t with money semantics. It panics unless t is a decimal
// type without semantics.
func (t Type) AsMoney() Type {
	t, err := t.withSemantics(MoneySemantics, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsPercentage returns t with percentage semantics. It panics unless t is a
// decimal type without semantics.
func (t Type) AsPercentage() Type {
	t, err := t.withSemantics(PercentageSemantics, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsPhone returns t with phone semantics. It panics if t is not a string
// type or if t already has semantics or any string constraints.
func (t Type) AsPhone() Type {
	t, err := t.withSemantics(PhoneSemantics, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// AsURL returns t with URL semantics. It panics unless t is a string type
// without semantics.
func (t Type) AsURL() Type {
	t, err := t.withSemantics(URLSemantics, nil)
	if err != nil {
		panic(err.Error())
	}
	return t
}

// CountryFormat returns the country format of t. It panics unless t has
// country semantics.
func (t Type) CountryFormat() CountryFormat {
	if t.semantics != CountrySemantics {
		panic("type does not have country semantics")
	}
	return t.semanticsOption.(CountryFormat)
}

// Currency returns the currency code of t and whether one is set. It panics
// unless t has money semantics.
func (t Type) Currency() (string, bool) {
	if t.semantics != MoneySemantics {
		panic("type does not have money semantics")
	}
	currency, _ := t.semanticsOption.(string)
	return currency, currency != ""
}

// DurationUnit returns the duration unit of t. It panics unless t has
// duration semantics.
func (t Type) DurationUnit() DurationUnit {
	if t.semantics != DurationSemantics {
		panic("type does not have duration semantics")
	}
	return t.semanticsOption.(DurationUnit)
}

// Semantics returns the semantics of t, or NoSemantics if t has no semantics.
func (t Type) Semantics() Semantics {
	return t.semantics
}

// UnitOfMeasure returns the unit of measure of t. It panics unless t has
// measurement semantics.
func (t Type) UnitOfMeasure() UnitOfMeasure {
	if t.semantics != MeasurementSemantics {
		panic("type does not have measurement semantics")
	}
	return t.semanticsOption.(UnitOfMeasure)
}

// WithCurrency returns t with the given ISO 4217 currency code. It panics
// unless t has money semantics and currency is valid.
func (t Type) WithCurrency(currency string) Type {
	if t.semantics != MoneySemantics {
		panic("type does not have money semantics")
	}
	if !validation.IsValidCurrencyCode(currency) {
		panic("invalid currency code")
	}
	t.semanticsOption = currency
	return t
}

// withSemantics returns t with the given semantics and option, or an error if
// they are invalid or incompatible with t.
func (t Type) withSemantics(semantics Semantics, option any) (Type, error) {

	if semantics == NoSemantics {
		return Type{}, errors.New("cannot apply NoSemantics to a type")
	}
	if t.semantics != NoSemantics {
		return Type{}, errors.New("type already has semantics")
	}

	switch semantics {
	case EmailSemantics:
		if t.kind != StringKind {
			return Type{}, errors.New("email semantics require a string type")
		}
		if option != nil {
			return Type{}, errors.New("email semantics do not accept an option")
		}
	case PhoneSemantics:
		if t.kind != StringKind {
			return Type{}, errors.New("phone semantics require a string type")
		}
		if option != nil {
			return Type{}, errors.New("phone semantics do not accept an option")
		}
	case URLSemantics:
		if t.kind != StringKind {
			return Type{}, errors.New("URL semantics require a string type")
		}
		if option != nil {
			return Type{}, errors.New("URL semantics do not accept an option")
		}
	case CountrySemantics:
		if t.kind != StringKind {
			return Type{}, errors.New("country semantics require a string type")
		}
		format, ok := option.(CountryFormat)
		if !ok || format != ISO3166Alpha2 && format != ISO3166Alpha3 {
			return Type{}, errors.New("invalid country format")
		}
	case MoneySemantics:
		if t.kind != DecimalKind {
			return Type{}, errors.New("money semantics require a decimal type")
		}
		if option != nil {
			return Type{}, errors.New("money semantics do not accept an option")
		}
	case PercentageSemantics:
		if t.kind != DecimalKind {
			return Type{}, errors.New("percentage semantics require a decimal type")
		}
		if option != nil {
			return Type{}, errors.New("percentage semantics do not accept an option")
		}
	case MeasurementSemantics:
		if !semanticsNumericType(t) {
			return Type{}, errors.New("measurement semantics require an int, decimal, or real float type")
		}
		unit, ok := option.(UnitOfMeasure)
		if !ok || unit < 1 || int(unit) > len(unitOfMeasureName) {
			return Type{}, errors.New("invalid unit of measure")
		}
	case DurationSemantics:
		if !semanticsNumericType(t) {
			return Type{}, errors.New("duration semantics require an int, decimal, or real float type")
		}
		unit, ok := option.(DurationUnit)
		if !ok || unit < 1 || int(unit) > len(durationUnitName) {
			return Type{}, errors.New("invalid duration unit")
		}
	}
	if (semantics == CountrySemantics || semantics == PhoneSemantics) && (t.p != 0 || t.s != 0 || t.vl != nil) {
		return Type{}, fmt.Errorf("%s semantics cannot be combined with other string constraints", semantics)
	}

	t.semantics = semantics
	t.semanticsOption = option

	return t, nil
}

// semanticsNumericType reports whether t is an int, decimal, or real float
// type.
func semanticsNumericType(t Type) bool {
	switch t.kind {
	case IntKind, DecimalKind:
		return true
	case FloatKind:
		return t.real
	default:
		return false
	}
}
