// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import "github.com/krenalis/krenalis/tools/errors"

// SemanticKind identifies the meaning associated with a property value.
type SemanticKind int8

const (
	InvalidSemanticKind     SemanticKind = iota // does not identify a semantic
	EmailSemanticKind                           // email address
	PhoneSemanticKind                           // phone number
	URLSemanticKind                             // web URL
	CountrySemanticKind                         // country
	DateTimeSemanticKind                        // date and time represented as formatted text
	MoneySemanticKind                           // monetary amount
	PercentageSemanticKind                      // percentage
	MeasurementSemanticKind                     // numeric measurement
	DurationSemanticKind                        // duration
)

// semanticKindName contains the JSON name of each valid semantic kind.
var semanticKindName = []string{
	"email",
	"phone",
	"url",
	"country",
	"datetime",
	"money",
	"percentage",
	"measurement",
	"duration",
}

// SemanticKindByName returns a semantic kind by its name. The second return
// parameter reports whether a semantic kind with the given name exists.
func SemanticKindByName(name string) (SemanticKind, bool) {
	for i, n := range semanticKindName {
		if n == name {
			return SemanticKind(i + 1), true
		}
	}
	return InvalidSemanticKind, false
}

// String returns the name of k.
func (k SemanticKind) String() string {
	if !k.Valid() {
		return "Invalid"
	}
	return semanticKindName[k-1]
}

// Valid reports whether k is a valid semantic kind.
func (k SemanticKind) Valid() bool {
	return 1 <= k && int(k) <= len(semanticKindName)
}

// Semantic describes what a property value represents and, when necessary,
// how its representation is interpreted. For an array property, it describes
// each element; for a map property, it describes each value, not its keys.
// For nested arrays and maps, this applies recursively until a type that is
// neither an array nor a map is reached.
//
// Semantic values are immutable.
type Semantic interface {

	// Kind returns the semantic kind.
	Kind() SemanticKind

	// semantic prevents implementations outside this package.
	semantic()
}

// EmailSemantic describes an email address.
type EmailSemantic interface {
	Semantic

	// email distinguishes email semantics from other stateless semantics.
	email()
}

// emailSemantic implements EmailSemantic.
type emailSemantic struct{}

// emailSemanticInstance is the shared email semantic.
var emailSemanticInstance = &emailSemantic{}

// Email returns the semantic for an email address.
func Email() EmailSemantic {
	return emailSemanticInstance
}

// Kind returns the email semantic kind.
func (*emailSemantic) Kind() SemanticKind {
	return EmailSemanticKind
}

// email implements EmailSemantic.
func (*emailSemantic) email() {}

// semantic implements Semantic.
func (*emailSemantic) semantic() {}

// PhoneSemantic describes a phone number.
type PhoneSemantic interface {
	Semantic

	// phone distinguishes phone semantics from other stateless semantics.
	phone()
}

// phoneSemantic implements PhoneSemantic.
type phoneSemantic struct{}

// phoneSemanticInstance is the shared phone semantic.
var phoneSemanticInstance = &phoneSemantic{}

// Phone returns the semantic for a phone number.
func Phone() PhoneSemantic {
	return phoneSemanticInstance
}

// Kind returns the phone semantic kind.
func (*phoneSemantic) Kind() SemanticKind {
	return PhoneSemanticKind
}

// phone implements PhoneSemantic.
func (*phoneSemantic) phone() {}

// semantic implements Semantic.
func (*phoneSemantic) semantic() {}

// URLSemantic describes a URL.
type URLSemantic interface {
	Semantic

	// url distinguishes URL semantics from other stateless semantics.
	url()
}

// urlSemantic implements URLSemantic.
type urlSemantic struct{}

// urlSemanticInstance is the shared URL semantic.
var urlSemanticInstance = &urlSemantic{}

// URL returns the semantic for a URL.
func URL() URLSemantic {
	return urlSemanticInstance
}

// Kind returns the URL semantic kind.
func (*urlSemantic) Kind() SemanticKind {
	return URLSemanticKind
}

// semantic implements Semantic.
func (*urlSemantic) semantic() {}

// url implements URLSemantic.
func (*urlSemantic) url() {}

// CountryFormat identifies how a country value is represented.
type CountryFormat int8

const (
	InvalidCountryFormat CountryFormat = iota // does not identify a country format
	ISO3166Alpha2                             // two-letter ISO 3166-1 alpha-2 code
	ISO3166Alpha3                             // three-letter ISO 3166-1 alpha-3 code
)

// countryFormatName contains the JSON name of each valid country format.
var countryFormatName = []string{
	"iso_3166_1_alpha_2",
	"iso_3166_1_alpha_3",
}

// CountryFormatByName returns a country format by its name. The second return
// parameter reports whether a country format with the given name exists.
func CountryFormatByName(name string) (CountryFormat, bool) {
	for i, n := range countryFormatName {
		if n == name {
			return CountryFormat(i + 1), true
		}
	}
	return InvalidCountryFormat, false
}

// String returns the name of f.
func (f CountryFormat) String() string {
	if !f.Valid() {
		return "Invalid"
	}
	return countryFormatName[f-1]
}

// Valid reports whether f is a valid country format.
func (f CountryFormat) Valid() bool {
	return 1 <= f && int(f) <= len(countryFormatName)
}

// CountrySemantic describes a country value and its representation.
type CountrySemantic interface {
	Semantic

	// Format returns the format used to represent the country.
	Format() CountryFormat

	// country distinguishes country semantics from other semantics.
	country()
}

// countrySemantic implements CountrySemantic.
type countrySemantic struct {
	format CountryFormat
}

// countrySemantics contains the shared country semantics indexed by format.
var countrySemantics = [...]countrySemantic{
	{},
	{format: ISO3166Alpha2},
	{format: ISO3166Alpha3},
}

// Country returns the semantic for a country represented using format.
// It panics if format is invalid.
func Country(format CountryFormat) CountrySemantic {
	if !format.Valid() {
		panic("invalid country format")
	}
	return &countrySemantics[format]
}

// Format returns the format used to represent the country.
func (s *countrySemantic) Format() CountryFormat {
	return s.format
}

// Kind returns the country semantic kind.
func (*countrySemantic) Kind() SemanticKind {
	return CountrySemanticKind
}

// country implements CountrySemantic.
func (*countrySemantic) country() {}

// semantic implements Semantic.
func (*countrySemantic) semantic() {}

// DateTimeSemantic describes a date and time represented as formatted text.
type DateTimeSemantic interface {
	Semantic

	// Format returns the format used to represent the date and time.
	Format() string

	// dateTime distinguishes formatted date and time semantics from other semantics.
	dateTime()
}

// dateTimeSemantic implements DateTimeSemantic.
type dateTimeSemantic struct {
	format string
}

// FormattedDateTime returns the semantic for a date and time represented using
// format. It panics if format is empty, is not valid UTF-8, or contains a NUL
// byte.
func FormattedDateTime(format string) DateTimeSemantic {
	s, err := newFormattedDateTime(format)
	if err != nil {
		panic(err)
	}
	return s
}

// newFormattedDateTime returns a validated formatted date and time semantic.
func newFormattedDateTime(format string) (*dateTimeSemantic, error) {
	format, err := normalizedUTF8(format)
	if err != nil {
		return nil, err
	}
	if format == "" {
		return nil, errors.New("datetime format is empty")
	}
	return &dateTimeSemantic{format: format}, nil
}

// Format returns the format used to represent the date and time.
func (s *dateTimeSemantic) Format() string {
	return s.format
}

// Kind returns the formatted date and time semantic kind.
func (*dateTimeSemantic) Kind() SemanticKind {
	return DateTimeSemanticKind
}

// dateTime implements DateTimeSemantic.
func (*dateTimeSemantic) dateTime() {}

// semantic implements Semantic.
func (*dateTimeSemantic) semantic() {}

// MoneySemantic describes a monetary amount.
type MoneySemantic interface {
	Semantic

	// Currency returns the currency code and whether one is set.
	Currency() (string, bool)

	// WithCurrency returns a copy of the semantic with the specified currency
	// code. It panics unless currency consists of three uppercase ASCII letters.
	WithCurrency(currency string) MoneySemantic

	// money distinguishes money semantics from other semantics.
	money()
}

// moneySemantic implements MoneySemantic.
type moneySemantic struct {
	currency string
}

// moneySemanticInstance is the shared money semantic without a currency.
var moneySemanticInstance = &moneySemantic{}

// Money returns the semantic for a monetary amount without a fixed currency.
func Money() MoneySemantic {
	return moneySemanticInstance
}

// Currency returns the currency code and whether one is set.
func (s *moneySemantic) Currency() (string, bool) {
	return s.currency, s.currency != ""
}

// Kind returns the money semantic kind.
func (*moneySemantic) Kind() SemanticKind {
	return MoneySemanticKind
}

// WithCurrency returns a copy of s with the specified currency code.
// It panics unless currency consists of three uppercase ASCII letters.
func (s *moneySemantic) WithCurrency(currency string) MoneySemantic {

	if !validCurrencyCode(currency) {
		panic("invalid currency code")
	}
	if s.currency == currency {
		return s
	}

	c := *s
	c.currency = currency

	return &c
}

// money implements MoneySemantic.
func (*moneySemantic) money() {}

// semantic implements Semantic.
func (*moneySemantic) semantic() {}

// validCurrencyCode reports whether currency consists of three uppercase ASCII letters.
func validCurrencyCode(currency string) bool {
	if len(currency) != 3 {
		return false
	}
	for i := range currency {
		if currency[i] < 'A' || currency[i] > 'Z' {
			return false
		}
	}
	return true
}

// PercentageFormat identifies how a percentage is represented numerically.
type PercentageFormat int8

const (
	InvalidPercentageFormat PercentageFormat = iota // does not identify a percentage format

	// FractionPercentage means that 0.15 represents 15%.
	FractionPercentage

	// WholePercentage means that 15 represents 15%.
	WholePercentage
)

// percentageFormatName contains the JSON name of each valid percentage format.
var percentageFormatName = []string{
	"fraction",
	"whole",
}

// PercentageFormatByName returns a percentage format by its name. The second
// return parameter reports whether a percentage format with the given name
// exists.
func PercentageFormatByName(name string) (PercentageFormat, bool) {
	for i, n := range percentageFormatName {
		if n == name {
			return PercentageFormat(i + 1), true
		}
	}
	return InvalidPercentageFormat, false
}

// String returns the name of f.
func (f PercentageFormat) String() string {
	if !f.Valid() {
		return "Invalid"
	}
	return percentageFormatName[f-1]
}

// Valid reports whether f is a valid percentage format.
func (f PercentageFormat) Valid() bool {
	return 1 <= f && int(f) <= len(percentageFormatName)
}

// PercentageSemantic describes a percentage and its numeric representation.
type PercentageSemantic interface {
	Semantic

	// Format returns the percentage format.
	Format() PercentageFormat

	// percentage distinguishes percentage semantics from other semantics.
	percentage()
}

// percentageSemantic implements PercentageSemantic.
type percentageSemantic struct {
	format PercentageFormat
}

// percentageSemantics contains the shared percentage semantics indexed by format.
var percentageSemantics = [...]percentageSemantic{
	{},
	{format: FractionPercentage},
	{format: WholePercentage},
}

// Percentage returns the semantic for a percentage represented using format.
// It panics if format is invalid.
func Percentage(format PercentageFormat) PercentageSemantic {
	if !format.Valid() {
		panic("invalid percentage format")
	}
	return &percentageSemantics[format]
}

// Format returns the percentage format.
func (s *percentageSemantic) Format() PercentageFormat {
	return s.format
}

// Kind returns the percentage semantic kind.
func (*percentageSemantic) Kind() SemanticKind {
	return PercentageSemanticKind
}

// percentage implements PercentageSemantic.
func (*percentageSemantic) percentage() {}

// semantic implements Semantic.
func (*percentageSemantic) semantic() {}

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
)

// unitOfMeasureName contains the JSON name of each valid unit of measure.
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
}

// UnitOfMeasureByName returns a unit of measure by its name. The second return
// parameter reports whether a unit with the given name exists.
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
	if !u.Valid() {
		return "Invalid"
	}
	return unitOfMeasureName[u-1]
}

// Valid reports whether u is a valid unit of measure.
func (u UnitOfMeasure) Valid() bool {
	return 1 <= u && int(u) <= len(unitOfMeasureName)
}

// MeasurementSemantic describes a numeric value that may have a unit of measure.
type MeasurementSemantic interface {
	Semantic

	// UnitOfMeasure returns the unit of measure and whether one is set.
	UnitOfMeasure() (UnitOfMeasure, bool)

	// WithUnitOfMeasure returns a copy of the semantic with the specified unit
	// of measure. It panics if unit is invalid.
	WithUnitOfMeasure(unit UnitOfMeasure) MeasurementSemantic

	// measurement distinguishes measurement semantics from other semantics.
	measurement()
}

// measurementSemantic implements MeasurementSemantic.
type measurementSemantic struct {
	unit UnitOfMeasure
}

// measurementSemanticInstance is the shared measurement semantic without a unit.
var measurementSemanticInstance = &measurementSemantic{}

// Measurement returns the semantic for a numeric value without a unit of measure.
func Measurement() MeasurementSemantic {
	return measurementSemanticInstance
}

// Kind returns the measurement semantic kind.
func (*measurementSemantic) Kind() SemanticKind {
	return MeasurementSemanticKind
}

// UnitOfMeasure returns the unit of measure and whether one is set.
func (s *measurementSemantic) UnitOfMeasure() (UnitOfMeasure, bool) {
	return s.unit, s.unit != InvalidUnitOfMeasure
}

// WithUnitOfMeasure returns a copy of s with the specified unit of measure.
// It panics if unit is invalid.
func (s *measurementSemantic) WithUnitOfMeasure(unit UnitOfMeasure) MeasurementSemantic {

	if !unit.Valid() {
		panic("invalid unit of measure")
	}
	if s.unit == unit {
		return s
	}

	c := *s
	c.unit = unit

	return &c
}

// measurement implements MeasurementSemantic.
func (*measurementSemantic) measurement() {}

// semantic implements Semantic.
func (*measurementSemantic) semantic() {}

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

// durationUnitName contains the JSON name of each valid duration unit.
var durationUnitName = []string{
	"millisecond",
	"second",
	"minute",
	"hour",
	"day",
	"week",
}

// DurationUnitByName returns a duration unit by its name. The second return
// parameter reports whether a duration unit with the given name exists.
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
	if !u.Valid() {
		return "Invalid"
	}
	return durationUnitName[u-1]
}

// Valid reports whether u is a valid duration unit.
func (u DurationUnit) Valid() bool {
	return 1 <= u && int(u) <= len(durationUnitName)
}

// DurationSemantic describes a duration.
type DurationSemantic interface {
	Semantic

	// Unit returns the unit used to represent the duration.
	Unit() DurationUnit

	// duration distinguishes duration semantics from other semantics.
	duration()
}

// durationSemantic implements DurationSemantic.
type durationSemantic struct {
	unit DurationUnit
}

// durationSemantics contains the shared duration semantics indexed by unit.
var durationSemantics = [...]durationSemantic{
	{},
	{unit: Millisecond},
	{unit: Second},
	{unit: Minute},
	{unit: Hour},
	{unit: Day},
	{unit: Week},
}

// Duration returns the semantic for a duration expressed in unit.
// It panics if unit is invalid.
func Duration(unit DurationUnit) DurationSemantic {
	if !unit.Valid() {
		panic("invalid duration unit")
	}
	return &durationSemantics[unit]
}

// Kind returns the duration semantic kind.
func (*durationSemantic) Kind() SemanticKind {
	return DurationSemanticKind
}

// Unit returns the unit used to represent the duration.
func (s *durationSemantic) Unit() DurationUnit {
	return s.unit
}

// duration implements DurationSemantic.
func (*durationSemantic) duration() {}

// semantic implements Semantic.
func (*durationSemantic) semantic() {}

// equalSemantics reports whether s1 and s2 have the same kind and options.
func equalSemantics(s1, s2 Semantic) bool {

	if s1 == nil || s2 == nil {
		return s1 == nil && s2 == nil
	}
	if s1.Kind() != s2.Kind() {
		return false
	}

	switch s1 := s1.(type) {
	case *emailSemantic, *phoneSemantic, *urlSemantic:
		return true
	case *countrySemantic:
		s2, ok := s2.(*countrySemantic)
		return ok && s1.format == s2.format
	case *dateTimeSemantic:
		s2, ok := s2.(*dateTimeSemantic)
		return ok && s1.format == s2.format
	case *moneySemantic:
		s2, ok := s2.(*moneySemantic)
		return ok && s1.currency == s2.currency
	case *percentageSemantic:
		s2, ok := s2.(*percentageSemantic)
		return ok && s1.format == s2.format
	case *measurementSemantic:
		s2, ok := s2.(*measurementSemantic)
		return ok && s1.unit == s2.unit
	case *durationSemantic:
		s2, ok := s2.(*durationSemantic)
		return ok && s1.unit == s2.unit
	default:
		panic("invalid semantic")
	}

}

// semanticNumericType reports whether t has a numeric domain that excludes non-real values.
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

// validateSemanticCompatibility verifies that s can be used with t.
func validateSemanticCompatibility(s Semantic, t Type) error {

	if s == nil {
		return nil
	}
	for t.kind == ArrayKind || t.kind == MapKind {
		t = t.Elem()
	}
	if t.Generic() {
		return errors.New("semantic cannot be used with a generic type")
	}

	switch s.Kind() {
	case EmailSemanticKind:
		if t.kind != StringKind {
			return errors.New("email semantic requires string type")
		}
	case PhoneSemanticKind:
		if t.kind != StringKind {
			return errors.New("phone semantic requires string type")
		}
	case URLSemanticKind:
		if t.kind != StringKind {
			return errors.New("URL semantic requires string type")
		}
	case CountrySemanticKind:
		if t.kind != StringKind {
			return errors.New("country semantic requires string type")
		}
	case DateTimeSemanticKind:
		if t.kind != StringKind {
			return errors.New("datetime semantic requires string type")
		}
	case MoneySemanticKind:
		if !semanticNumericType(t) {
			return errors.New("money semantic requires an int, decimal, or real float type")
		}
	case PercentageSemanticKind:
		if !semanticNumericType(t) {
			return errors.New("percentage semantic requires an int, decimal, or real float type")
		}
	case MeasurementSemanticKind:
		if !semanticNumericType(t) {
			return errors.New("measurement semantic requires an int, decimal, or real float type")
		}
	case DurationSemanticKind:
		if !semanticNumericType(t) {
			return errors.New("duration semantic requires an int, decimal, or real float type")
		}
	default:
		return errors.New("invalid semantic")
	}

	return nil
}
