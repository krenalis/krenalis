// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
)

// Test_CountryFormat tests country format names and lookup.
func Test_CountryFormat(t *testing.T) {
	tests := []struct {
		format CountryFormat
		name   string
	}{
		{ISO3166Alpha2, "alpha-2"},
		{ISO3166Alpha3, "alpha-3"},
	}
	for _, test := range tests {
		if got := test.format.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		got, ok := CountryFormatByName(test.name)
		if !ok || got != test.format {
			t.Errorf("expected format %d and true, got %d and %t", test.format, got, ok)
		}
	}
	if CountryFormat(0).String() != "%!CountryFormat(0)" {
		t.Fatalf("expected invalid country format name %q, got %q", "%!CountryFormat(0)", CountryFormat(0).String())
	}
	if got, ok := CountryFormatByName("ISO3166Alpha2"); ok || got != CountryFormat(0) {
		t.Fatalf("expected invalid format and false, got %d and %t", got, ok)
	}
}

// Test_DurationUnit tests duration unit names, lookup, and validation.
func Test_DurationUnit(t *testing.T) {
	tests := []struct {
		unit DurationUnit
		name string
	}{
		{Millisecond, "millisecond"},
		{Second, "second"},
		{Minute, "minute"},
		{Hour, "hour"},
		{Day, "day"},
		{Week, "week"},
	}
	for _, test := range tests {
		if test.unit < 1 || int(test.unit) > len(durationUnitName) {
			t.Errorf("expected valid duration unit, got %d", test.unit)
		}
		if got := test.unit.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		got, ok := DurationUnitByName(test.name)
		if !ok || got != test.unit {
			t.Errorf("expected unit %d and true, got %d and %t", test.unit, got, ok)
		}
	}
	if (1 <= DurationUnit(0) && int(DurationUnit(0)) <= len(durationUnitName)) ||
		(1 <= DurationUnit(-1) && int(DurationUnit(-1)) <= len(durationUnitName)) ||
		(1 <= DurationUnit(127) && int(DurationUnit(127)) <= len(durationUnitName)) ||
		DurationUnit(0).String() != "%!DurationUnit(0)" {
		t.Fatalf("expected invalid duration unit name %q, got %q", "%!DurationUnit(0)", DurationUnit(0).String())
	}
	if got, ok := DurationUnitByName("seconds"); ok || got != DurationUnit(0) {
		t.Fatalf("expected invalid unit and false, got %d and %t", got, ok)
	}
}

// Test_Semantics tests names and lookup for semantics.
func Test_Semantics(t *testing.T) {
	tests := []struct {
		semantics Semantics
		name      string
	}{
		{NoSemantics, "none"},
		{EmailSemantics, "email"},
		{PhoneSemantics, "phone"},
		{URLSemantics, "url"},
		{CountrySemantics, "country"},
		{MoneySemantics, "money"},
		{PercentageSemantics, "percentage"},
		{MeasurementSemantics, "measurement"},
		{DurationSemantics, "duration"},
	}
	for _, test := range tests {
		if got := test.semantics.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		if test.semantics != NoSemantics {
			got, ok := SemanticsByName(test.name)
			if !ok || got != test.semantics {
				t.Errorf("expected semantics %d and true, got %d and %t", test.semantics, got, ok)
			}
		}
	}
	for _, name := range []string{"none", "Email"} {
		if got, ok := SemanticsByName(name); ok || got != NoSemantics {
			t.Fatalf("expected no semantics and false for %q, got %d and %t", name, got, ok)
		}
	}
}

// Test_SemanticsInvalidStrings tests string representations of invalid enum
// values.
func Test_SemanticsInvalidStrings(t *testing.T) {
	tests := []struct {
		value fmt.Stringer
		name  string
	}{
		{Semantics(-128), "%!Semantics(-128)"},
		{Semantics(-1), "%!Semantics(-1)"},
		{Semantics(9), "%!Semantics(9)"},
		{Semantics(20), "%!Semantics(20)"},
		{Semantics(127), "%!Semantics(127)"},
		{CountryFormat(-128), "%!CountryFormat(-128)"},
		{CountryFormat(-1), "%!CountryFormat(-1)"},
		{CountryFormat(0), "%!CountryFormat(0)"},
		{CountryFormat(1), "%!CountryFormat(1)"},
		{CountryFormat(4), "%!CountryFormat(4)"},
		{CountryFormat(127), "%!CountryFormat(127)"},
		{DurationUnit(-128), "%!DurationUnit(-128)"},
		{DurationUnit(-1), "%!DurationUnit(-1)"},
		{DurationUnit(0), "%!DurationUnit(0)"},
		{DurationUnit(7), "%!DurationUnit(7)"},
		{DurationUnit(127), "%!DurationUnit(127)"},
		{UnitOfMeasure(-128), "%!UnitOfMeasure(-128)"},
		{UnitOfMeasure(-1), "%!UnitOfMeasure(-1)"},
		{UnitOfMeasure(0), "%!UnitOfMeasure(0)"},
		{UnitOfMeasure(21), "%!UnitOfMeasure(21)"},
		{UnitOfMeasure(127), "%!UnitOfMeasure(127)"},
	}
	for _, test := range tests {
		if got := test.value.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
	}
}

// Test_TypeSemanticsConfiguration tests semantics configuration and options.
func Test_TypeSemanticsConfiguration(t *testing.T) {
	tests := []struct {
		name      string
		type_     Type
		semantics Semantics
	}{
		{"email", String().AsEmail(), EmailSemantics},
		{"phone", String().AsPhone(), PhoneSemantics},
		{"URL", String().AsURL(), URLSemantics},
		{"country", String().AsCountry(ISO3166Alpha2), CountrySemantics},
		{"money decimal", Decimal(10, 2).AsMoney(), MoneySemantics},
		{"percentage", Decimal(10, 2).AsPercentage(), PercentageSemantics},
		{"measurement int", Int(64).AsMeasurement(Kilogram), MeasurementSemantics},
		{"measurement decimal", Decimal(10, 2).AsMeasurement(Kilogram), MeasurementSemantics},
		{"measurement real float", Float(64).Real().AsMeasurement(Kilogram), MeasurementSemantics},
		{"duration int", Int(32).AsDuration(Second), DurationSemantics},
		{"duration decimal", Decimal(10, 3).AsDuration(Millisecond), DurationSemantics},
		{"duration real float", Float(32).Real().AsDuration(Hour), DurationSemantics},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.type_.Semantics(); got != test.semantics {
				t.Fatalf("expected semantics %s, got %s", test.semantics, got)
			}
		})
	}
	country := String().AsCountry(ISO3166Alpha3)
	if got := country.CountryFormat(); got != ISO3166Alpha3 {
		t.Fatalf("expected country format %s, got %s", ISO3166Alpha3, got)
	}
	phone := String().AsPhone()
	constraintTests := []Type{
		country,
		phone,
		String().AsCountry(ISO3166Alpha2),
	}
	for _, typ := range constraintTests {
		if maxBytes, ok := typ.MaxBytes(); ok || maxBytes != 0 {
			t.Fatalf("expected no maxBytes constraint, got %d and %t", maxBytes, ok)
		}
		if maxLength, ok := typ.MaxLength(); ok || maxLength != 0 {
			t.Fatalf("expected no maxLength constraint, got %d and %t", maxLength, ok)
		}
	}
	money := Decimal(10, 2).AsMoney()
	if currency, ok := money.Currency(); ok || currency != "" {
		t.Fatalf("expected no currency, got %q and %t", currency, ok)
	}
	money = money.WithCurrency("EUR")
	if currency, ok := money.Currency(); !ok || currency != "EUR" {
		t.Fatalf("expected EUR and true, got %q and %t", currency, ok)
	}
	if got := Decimal(10, 2).AsMeasurement(Kilogram).UnitOfMeasure(); got != Kilogram {
		t.Fatalf("expected kilogram, got %s", got)
	}
	if got := Int(64).AsDuration(Week).DurationUnit(); got != Week {
		t.Fatalf("expected week, got %s", got)
	}
}

// Test_TypeSemanticsConfigurationPanics tests invalid semantics configuration.
func Test_TypeSemanticsConfigurationPanics(t *testing.T) {
	tests := []struct {
		name string
		f    func()
	}{
		{"email on boolean", func() { Boolean().AsEmail() }},
		{"phone on boolean", func() { Boolean().AsPhone() }},
		{"URL on boolean", func() { Boolean().AsURL() }},
		{"country on boolean", func() { Boolean().AsCountry(ISO3166Alpha2) }},
		{"country invalid format", func() { String().AsCountry(CountryFormat(0)) }},
		{"country with max bytes", func() { String().WithMaxBytes(2).AsCountry(ISO3166Alpha2) }},
		{"country with max length", func() { String().WithMaxLength(2).AsCountry(ISO3166Alpha2) }},
		{"country with pattern", func() { String().WithPattern(regexp.MustCompile(".")).AsCountry(ISO3166Alpha2) }},
		{"country with values", func() { String().WithValues("IT").AsCountry(ISO3166Alpha2) }},
		{"phone with max bytes", func() { String().WithMaxBytes(16).AsPhone() }},
		{"phone with max length", func() { String().WithMaxLength(16).AsPhone() }},
		{"phone with pattern", func() { String().WithPattern(regexp.MustCompile(".")).AsPhone() }},
		{"phone with values", func() { String().WithValues("+390000000000").AsPhone() }},
		{"money on string", func() { String().AsMoney() }},
		{"money on int", func() { Int(32).AsMoney() }},
		{"money on unsigned int", func() { Int(32).Unsigned().AsMoney() }},
		{"money on ordinary float", func() { Float(64).AsMoney() }},
		{"money on real float", func() { Float(64).Real().AsMoney() }},
		{"percentage on int", func() { Int(32).AsPercentage() }},
		{"measurement on JSON", func() { JSON().AsMeasurement(Kilogram) }},
		{"measurement invalid unit", func() { Int(64).AsMeasurement(UnitOfMeasure(0)) }},
		{"duration on ordinary float", func() { Float(32).AsDuration(Second) }},
		{"duration invalid unit", func() { Int(64).AsDuration(DurationUnit(0)) }},
		{"duplicate semantics", func() { String().AsEmail().AsPhone() }},
		{"invalid currency", func() { Decimal(10, 2).AsMoney().WithCurrency("usd") }},
		{"currency without money", func() { Decimal(10, 2).WithCurrency("EUR") }},
		{"country format without country", func() { String().CountryFormat() }},
		{"duration unit without duration", func() { Int(64).DurationUnit() }},
		{"unit of measure without measurement", func() { Decimal(10, 2).UnitOfMeasure() }},
		{"max bytes after country", func() { String().AsCountry(ISO3166Alpha2).WithMaxBytes(2) }},
		{"max length after country", func() { String().AsCountry(ISO3166Alpha2).WithMaxLength(2) }},
		{"pattern after country", func() { String().AsCountry(ISO3166Alpha2).WithPattern(regexp.MustCompile(".")) }},
		{"values after country", func() { String().AsCountry(ISO3166Alpha2).WithValues("IT") }},
		{"max bytes after phone", func() { String().AsPhone().WithMaxBytes(16) }},
		{"max length after phone", func() { String().AsPhone().WithMaxLength(16) }},
		{"pattern after phone", func() { String().AsPhone().WithPattern(regexp.MustCompile(".")) }},
		{"values after phone", func() { String().AsPhone().WithValues("+390000000000") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic, got nil")
				}
			}()
			test.f()
		})
	}
}

// Test_TypeSemanticsCopyOnWrite tests that methods configuring semantics do not
// mutate their receivers.
func Test_TypeSemanticsCopyOnWrite(t *testing.T) {
	decimalType := Decimal(10, 2)
	money := decimalType.AsMoney()
	euros := money.WithCurrency("EUR")
	if decimalType.Semantics() != NoSemantics {
		t.Fatalf("expected receiver semantics %s, got %s", NoSemantics, decimalType.Semantics())
	}
	if currency, ok := money.Currency(); ok || currency != "" {
		t.Fatalf("expected no currency and false, got %q and %t", currency, ok)
	}
	if currency, ok := euros.Currency(); !ok || currency != "EUR" {
		t.Fatalf("expected EUR and true, got %q and %t", currency, ok)
	}
}

// TestEqualSemantics tests semantics equality while ignoring kind and unrelated
// type properties.
func TestEqualSemantics(t *testing.T) {

	tests := []struct {
		name      string
		t1        Type
		t2        Type
		wantEqual bool
	}{
		{"without semantics", String(), Int(64), true},
		{"equal phone", String().AsPhone(), String().AsPhone(), true},
		{"missing and present", String(), String().AsPhone(), false},
		{"different semantics", String().AsEmail(), String().AsPhone(), false},
		{"equal country", String().AsCountry(ISO3166Alpha2), String().AsCountry(ISO3166Alpha2), true},
		{"different country format", String().AsCountry(ISO3166Alpha2), String().AsCountry(ISO3166Alpha3), false},
		{"equal duration with different constraints", Int(32).AsDuration(Second), Int(64).AsDuration(Second), true},
		{"different duration unit", Int(64).AsDuration(Second), Int(64).AsDuration(Minute), false},
		{"equal measurement", Decimal(10, 2).AsMeasurement(Kilogram), Int(64).AsMeasurement(Kilogram), true},
		{"different measurement unit", Decimal(10, 2).AsMeasurement(Kilogram), Int(64).AsMeasurement(Gram), false},
		{"money without currency", Decimal(10, 2).AsMoney(), Decimal(18, 4).AsMoney(), true},
		{
			"equal money currency", Decimal(10, 2).AsMoney().WithCurrency("EUR"),
			Decimal(18, 4).AsMoney().WithCurrency("EUR"), true,
		},
		{
			"different money currency", Decimal(10, 2).AsMoney().WithCurrency("EUR"),
			Decimal(10, 2).AsMoney().WithCurrency("USD"), false,
		},
		{"missing money currency", Decimal(10, 2).AsMoney(), Decimal(10, 2).AsMoney().WithCurrency("EUR"), false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := EqualSemantics(test.t1, test.t2); got != test.wantEqual {
				t.Fatalf("expected equality %t, got %t", test.wantEqual, got)
			}
			if got := EqualSemantics(test.t2, test.t1); got != test.wantEqual {
				t.Fatalf("expected reverse equality %t, got %t", test.wantEqual, got)
			}
		})
	}

}

// Test_TypeSemanticsEquality tests equality with semantics values and options.
func Test_TypeSemanticsEquality(t *testing.T) {
	tests := []struct {
		name  string
		t1    Type
		t2    Type
		equal bool
	}{
		{"without semantics", String(), String(), true},
		{"equal email", String().AsEmail(), String().AsEmail(), true},
		{"missing and present", String(), String().AsEmail(), false},
		{"different semantics", String().AsEmail(), String().AsPhone(), false},
		{"equal country", String().AsCountry(ISO3166Alpha2), String().AsCountry(ISO3166Alpha2), true},
		{"different country format", String().AsCountry(ISO3166Alpha2), String().AsCountry(ISO3166Alpha3), false},
		{
			"equal currency", Decimal(10, 2).AsMoney().WithCurrency("EUR"),
			Decimal(10, 2).AsMoney().WithCurrency("EUR"), true,
		},
		{
			"different currency", Decimal(10, 2).AsMoney().WithCurrency("EUR"),
			Decimal(10, 2).AsMoney().WithCurrency("USD"), false,
		},
		{"equal unit", Decimal(10, 2).AsMeasurement(Kilogram), Decimal(10, 2).AsMeasurement(Kilogram), true},
		{"different unit", Decimal(10, 2).AsMeasurement(Kilogram), Decimal(10, 2).AsMeasurement(Gram), false},
		{"nested semantics", Array(Map(String().AsEmail())), Array(Map(String().AsEmail())), true},
		{"different nested semantics", Array(Map(String().AsEmail())), Array(Map(String().AsPhone())), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Equal(test.t1, test.t2); got != test.equal {
				t.Fatalf("expected equality %t, got %t", test.equal, got)
			}
			if got := Equal(test.t2, test.t1); got != test.equal {
				t.Fatalf("expected reverse equality %t, got %t", test.equal, got)
			}
		})
	}
}

// Test_TypeSemanticsJSONErrors tests rejection of invalid type semantics in
// JSON.
func Test_TypeSemanticsJSONErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
		err  string
	}{
		{"semantics with wrong type", `{"kind":"string","semantics":1}`, "invalid semantics"},
		{"repeated semantics", `{"kind":"string","semantics":"email","semantics":"phone"}`, "repeated 'semantics' key"},
		{"no semantics specified", `{"kind":"string","semantics":"none"}`, `invalid semantics "none"`},
		{"unknown semantics", `{"kind":"string","semantics":"unknown"}`, `invalid semantics "unknown"`},
		{
			"format without semantics", `{"kind":"string","format":"alpha-2"}`,
			"unexpected 'format' key without semantics",
		},
		{
			"currency without semantics", `{"kind":"string","currency":"EUR"}`,
			"unexpected 'currency' key without semantics",
		},
		{
			"unit without semantics", `{"kind":"string","unit":"kg"}`,
			"unexpected 'unit' key without semantics",
		},
		{"missing country format", `{"kind":"string","semantics":"country"}`, "missing country format"},
		{
			"invalid country format", `{"kind":"string","semantics":"country","format":"alpha-4"}`,
			`invalid country format "alpha-4"`,
		},
		{
			"country on boolean", `{"kind":"boolean","semantics":"country","format":"alpha-2"}`,
			"country semantics require a string type",
		},
		{
			"country with max bytes",
			`{"kind":"string","semantics":"country","format":"alpha-2","maxBytes":2}`,
			"country semantics cannot be combined with other string constraints",
		},
		{
			"country with max length",
			`{"kind":"string","semantics":"country","format":"alpha-2","maxLength":2}`,
			"country semantics cannot be combined with other string constraints",
		},
		{
			"country with pattern",
			`{"kind":"string","semantics":"country","format":"alpha-2","pattern":".."}`,
			"country semantics cannot be combined with other string constraints",
		},
		{
			"country with values",
			`{"kind":"string","semantics":"country","format":"alpha-2","values":["IT"]}`,
			"country semantics cannot be combined with other string constraints",
		},
		{
			"phone with max bytes", `{"kind":"string","semantics":"phone","maxBytes":16}`,
			"phone semantics cannot be combined with other string constraints",
		},
		{
			"phone with max length", `{"kind":"string","semantics":"phone","maxLength":16}`,
			"phone semantics cannot be combined with other string constraints",
		},
		{
			"phone with pattern", `{"kind":"string","semantics":"phone","pattern":".+"}`,
			"phone semantics cannot be combined with other string constraints",
		},
		{
			"phone with values", `{"kind":"string","semantics":"phone","values":["+390000000000"]}`,
			"phone semantics cannot be combined with other string constraints",
		},
		{
			"invalid currency", `{"kind":"decimal","precision":10,"scale":2,"semantics":"money","currency":"usd"}`,
			`invalid currency code "usd"`,
		},
		{
			"money on int", `{"kind":"int","bitSize":64,"semantics":"money"}`,
			"money semantics require a decimal type",
		},
		{
			"money on real float", `{"kind":"float","bitSize":64,"real":true,"semantics":"money"}`,
			"money semantics require a decimal type",
		},
		{
			"missing measurement unit", `{"kind":"decimal","precision":10,"scale":2,"semantics":"measurement"}`,
			"missing measurement unit",
		},
		{
			"invalid measurement unit",
			`{"kind":"decimal","precision":10,"scale":2,"semantics":"measurement","unit":"stone"}`,
			`invalid unit of measure "stone"`,
		},
		{"missing duration unit", `{"kind":"int","bitSize":64,"semantics":"duration"}`, "missing duration unit"},
		{
			"invalid duration unit", `{"kind":"int","bitSize":64,"semantics":"duration","unit":"month"}`,
			`invalid duration unit "month"`,
		},
		{
			"unexpected email currency", `{"kind":"string","semantics":"email","currency":"EUR"}`,
			"unexpected 'currency' key for email semantics",
		},
		{
			"unexpected phone format", `{"kind":"string","semantics":"phone","format":"alpha-2"}`,
			"unexpected 'format' key for phone semantics",
		},
		{
			"unexpected URL format", `{"kind":"string","semantics":"url","format":"absolute"}`,
			"unexpected 'format' key for URL semantics",
		},
		{
			"unexpected URL currency", `{"kind":"string","semantics":"url","currency":"EUR"}`,
			"unexpected 'currency' key for URL semantics",
		},
		{
			"unexpected URL unit", `{"kind":"string","semantics":"url","unit":"m"}`,
			"unexpected 'unit' key for URL semantics",
		},
		{
			"unexpected country currency without format",
			`{"kind":"string","semantics":"country","currency":"EUR"}`,
			"unexpected 'currency' key for country semantics",
		},
		{
			"unexpected country unit",
			`{"kind":"string","semantics":"country","format":"alpha-2","unit":"m"}`,
			"unexpected 'unit' key for country semantics",
		},
		{
			"unexpected money format",
			`{"kind":"decimal","precision":10,"scale":2,"semantics":"money","format":"integer"}`,
			"unexpected 'format' key for money semantics",
		},
		{
			"unexpected money unit",
			`{"kind":"decimal","precision":10,"scale":2,"semantics":"money","unit":"cent"}`,
			"unexpected 'unit' key for money semantics",
		},
		{
			"unexpected percentage unit",
			`{"kind":"decimal","precision":10,"scale":2,"semantics":"percentage","unit":"percent"}`,
			"unexpected 'unit' key for percentage semantics",
		},
		{
			"unexpected measurement format without unit",
			`{"kind":"decimal","precision":10,"scale":2,"semantics":"measurement","format":"metric"}`,
			"unexpected 'format' key for measurement semantics",
		},
		{
			"unexpected measurement currency",
			`{"kind":"decimal","precision":10,"scale":2,"semantics":"measurement","unit":"kg","currency":"EUR"}`,
			"unexpected 'currency' key for measurement semantics",
		},
		{
			"unexpected duration format without unit", `{"kind":"int","bitSize":64,"semantics":"duration","format":"integer"}`,
			"unexpected 'format' key for duration semantics",
		},
		{
			"unexpected duration currency", `{"kind":"int","bitSize":64,"semantics":"duration","unit":"second","currency":"EUR"}`,
			"unexpected 'currency' key for duration semantics",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var type_ Type
			err := json.Unmarshal([]byte(test.data), &type_)
			if err != nil {
				if err.Error() != test.err {
					t.Fatalf("expected error %q, got %q", test.err, err)
				}
				return
			}
			t.Fatal("expected error, got nil")
		})
	}
}

// Test_TypeSemanticsJSONRoundTrip tests canonical JSON encoding and round-trip
// decoding of type semantics.
func Test_TypeSemanticsJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		type_ Type
		data  string
	}{
		{"no semantics", String(), `{"kind":"string"}`},
		{"email", String().AsEmail(), `{"kind":"string","semantics":"email"}`},
		{"phone", String().AsPhone(), `{"kind":"string","semantics":"phone"}`},
		{"URL", String().AsURL(), `{"kind":"string","semantics":"url"}`},
		{
			"country alpha-2", String().AsCountry(ISO3166Alpha2),
			`{"kind":"string","semantics":"country","format":"alpha-2"}`,
		},
		{
			"country alpha-3", String().AsCountry(ISO3166Alpha3),
			`{"kind":"string","semantics":"country","format":"alpha-3"}`,
		},
		{"money", Decimal(10, 2).AsMoney(), `{"kind":"decimal","semantics":"money","precision":10,"scale":2}`},
		{
			"money EUR", Decimal(10, 2).AsMoney().WithCurrency("EUR"),
			`{"kind":"decimal","semantics":"money","currency":"EUR","precision":10,"scale":2}`,
		},
		{"percentage", Decimal(18, 4).AsPercentage(), `{"kind":"decimal","semantics":"percentage","precision":18,"scale":4}`},
		{
			"measurement", Decimal(10, 2).AsMeasurement(Kilogram),
			`{"kind":"decimal","semantics":"measurement","unit":"kg","precision":10,"scale":2}`,
		},
		{
			"duration", Int(64).AsDuration(Millisecond),
			`{"kind":"int","semantics":"duration","unit":"millisecond","bitSize":64}`,
		},
		{"array email", Array(String().AsEmail()), `{"kind":"array","elementType":{"kind":"string","semantics":"email"}}`},
		{
			"nested money", Array(Map(Decimal(10, 2).AsMoney().WithCurrency("EUR"))),
			`{"kind":"array","elementType":{"kind":"map","elementType":` +
				`{"kind":"decimal","semantics":"money","currency":"EUR","precision":10,"scale":2}}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := json.Marshal(test.type_)
			if err != nil {
				t.Fatalf("expected marshal success, got error %v", err)
			}
			if string(got) != test.data {
				t.Fatalf("expected %q, got %q", test.data, got)
			}
			var type_ Type
			err = json.Unmarshal(got, &type_)
			if err != nil {
				t.Fatalf("expected unmarshal success, got error %v", err)
			}
			if !Equal(test.type_, type_) {
				t.Fatalf("expected round trip type %v, got %v", test.type_, type_)
			}
		})
	}
}

// Test_UnitOfMeasure tests unit-of-measure names, lookup, and validation.
func Test_UnitOfMeasure(t *testing.T) {
	tests := []struct {
		unit UnitOfMeasure
		name string
	}{
		{Gram, "g"},
		{Kilogram, "kg"},
		{Millimeter, "mm"},
		{Centimeter, "cm"},
		{Meter, "m"},
		{Kilometer, "km"},
		{Milliliter, "mL"},
		{Liter, "L"},
		{Byte, "B"},
		{Kilobyte, "kB"},
		{Megabyte, "MB"},
		{Gigabyte, "GB"},
		{Celsius, "°C"},
		{Fahrenheit, "°F"},
		{Ounce, "oz"},
		{Pound, "lb"},
		{Inch, "in"},
		{Foot, "ft"},
		{Yard, "yd"},
		{Mile, "mi"},
	}
	for _, test := range tests {
		if test.unit < 1 || int(test.unit) > len(unitOfMeasureName) {
			t.Errorf("expected valid unit of measure, got %d", test.unit)
		}
		if got := test.unit.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		got, ok := UnitOfMeasureByName(test.name)
		if !ok || got != test.unit {
			t.Errorf("expected unit %d and true, got %d and %t", test.unit, got, ok)
		}
	}
	if (1 <= UnitOfMeasure(0) && int(UnitOfMeasure(0)) <= len(unitOfMeasureName)) ||
		(1 <= UnitOfMeasure(-1) && int(UnitOfMeasure(-1)) <= len(unitOfMeasureName)) ||
		(1 <= UnitOfMeasure(127) && int(UnitOfMeasure(127)) <= len(unitOfMeasureName)) ||
		UnitOfMeasure(0).String() != "%!UnitOfMeasure(0)" {
		t.Fatalf("expected invalid unit of measure name %q, got %q", "%!UnitOfMeasure(0)", UnitOfMeasure(0).String())
	}
	if got, ok := UnitOfMeasureByName("kilogram"); ok || got != UnitOfMeasure(0) {
		t.Fatalf("expected invalid unit and false, got %d and %t", got, ok)
	}
}
