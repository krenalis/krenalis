// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"encoding/json"
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
	if CountryFormat(0).String() != "Invalid" {
		t.Fatal("invalid country format has an unexpected name")
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
			t.Errorf("%d is not valid", test.unit)
		}
		if got := test.unit.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		got, ok := DurationUnitByName(test.name)
		if !ok || got != test.unit {
			t.Errorf("expected unit %d and true, got %d and %t", test.unit, got, ok)
		}
	}
	if (1 <= InvalidDurationUnit && int(InvalidDurationUnit) <= len(durationUnitName)) ||
		(1 <= DurationUnit(-1) && int(DurationUnit(-1)) <= len(durationUnitName)) ||
		(1 <= DurationUnit(127) && int(DurationUnit(127)) <= len(durationUnitName)) ||
		InvalidDurationUnit.String() != "Invalid" {
		t.Fatal("invalid duration unit is valid or has an unexpected name")
	}
	if got, ok := DurationUnitByName("seconds"); ok || got != InvalidDurationUnit {
		t.Fatalf("expected invalid unit and false, got %d and %t", got, ok)
	}
}

// Test_Semantic tests semantic names and lookup.
func Test_Semantic(t *testing.T) {
	tests := []struct {
		semantic Semantic
		name     string
	}{
		{NoSemantic, "none"},
		{EmailSemantic, "email"},
		{PhoneSemantic, "phone"},
		{URLSemantic, "url"},
		{CountrySemantic, "country"},
		{MoneySemantic, "money"},
		{PercentageSemantic, "percentage"},
		{MeasurementSemantic, "measurement"},
		{DurationSemantic, "duration"},
	}
	for _, test := range tests {
		if got := test.semantic.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		if test.semantic != NoSemantic {
			got, ok := SemanticByName(test.name)
			if !ok || got != test.semantic {
				t.Errorf("expected semantic %d and true, got %d and %t", test.semantic, got, ok)
			}
		}
	}
	for _, name := range []string{"none", "Email"} {
		if got, ok := SemanticByName(name); ok || got != NoSemantic {
			t.Fatalf("expected no semantic and false for %q, got %d and %t", name, got, ok)
		}
	}
}

// Test_TypeSemanticConfiguration tests semantic configuration and options.
func Test_TypeSemanticConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		type_    Type
		semantic Semantic
	}{
		{"email", String().AsEmail(), EmailSemantic},
		{"phone", String().AsPhone(), PhoneSemantic},
		{"URL", String().AsURL(), URLSemantic},
		{"country", String().AsCountry(ISO3166Alpha2), CountrySemantic},
		{"money decimal", Decimal(10, 2).AsMoney(), MoneySemantic},
		{"percentage", Decimal(10, 2).AsPercentage(), PercentageSemantic},
		{"measurement int", Int(64).AsMeasurement(Kilogram), MeasurementSemantic},
		{"measurement decimal", Decimal(10, 2).AsMeasurement(Kilogram), MeasurementSemantic},
		{"measurement real float", Float(64).Real().AsMeasurement(Kilogram), MeasurementSemantic},
		{"duration int", Int(32).AsDuration(Second), DurationSemantic},
		{"duration decimal", Decimal(10, 3).AsDuration(Millisecond), DurationSemantic},
		{"duration real float", Float(32).Real().AsDuration(Hour), DurationSemantic},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.type_.Semantic(); got != test.semantic {
				t.Fatalf("expected semantic %s, got %s", test.semantic, got)
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

// Test_TypeSemanticConfigurationPanics tests invalid semantic configuration.
func Test_TypeSemanticConfigurationPanics(t *testing.T) {
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
		{"measurement invalid unit", func() { Int(64).AsMeasurement(InvalidUnitOfMeasure) }},
		{"duration on ordinary float", func() { Float(32).AsDuration(Second) }},
		{"duration invalid unit", func() { Int(64).AsDuration(InvalidDurationUnit) }},
		{"second semantic", func() { String().AsEmail().AsPhone() }},
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
					t.Fatal("expected a panic")
				}
			}()
			test.f()
		})
	}
}

// Test_TypeSemanticCopyOnWrite tests that semantic methods do not mutate their
// receivers.
func Test_TypeSemanticCopyOnWrite(t *testing.T) {
	decimalType := Decimal(10, 2)
	money := decimalType.AsMoney()
	euros := money.WithCurrency("EUR")
	if decimalType.Semantic() != NoSemantic {
		t.Fatal("AsMoney mutated its receiver")
	}
	if currency, ok := money.Currency(); ok || currency != "" {
		t.Fatalf("WithCurrency mutated its receiver: got %q and %t", currency, ok)
	}
	if currency, ok := euros.Currency(); !ok || currency != "EUR" {
		t.Fatalf("expected EUR and true, got %q and %t", currency, ok)
	}
}

// Test_TypeSemanticEquality tests equality with semantic kinds and options.
func Test_TypeSemanticEquality(t *testing.T) {
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
		{"nested semantic", Array(Map(String().AsEmail())), Array(Map(String().AsEmail())), true},
		{"different nested semantic", Array(Map(String().AsEmail())), Array(Map(String().AsPhone())), false},
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

// Test_TypeSemanticJSONErrors tests rejection of invalid semantic type JSON.
func Test_TypeSemanticJSONErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
		err  string
	}{
		{"semantic wrong type", `{"kind":"string","semantic":1}`, "invalid semantic"},
		{"repeated semantic", `{"kind":"string","semantic":"email","semantic":"phone"}`, "repeated 'semantic' key"},
		{"no semantic specified", `{"kind":"string","semantic":"none"}`, `invalid semantic "none"`},
		{"unknown semantic", `{"kind":"string","semantic":"unknown"}`, `invalid semantic "unknown"`},
		{
			"format without semantic", `{"kind":"string","format":"alpha-2"}`,
			"unexpected 'format' key without semantic",
		},
		{
			"currency without semantic", `{"kind":"string","currency":"EUR"}`,
			"unexpected 'currency' key without semantic",
		},
		{
			"unit without semantic", `{"kind":"string","unit":"kg"}`,
			"unexpected 'unit' key without semantic",
		},
		{"missing country format", `{"kind":"string","semantic":"country"}`, "missing country format"},
		{
			"invalid country format", `{"kind":"string","semantic":"country","format":"alpha-4"}`,
			`invalid country format "alpha-4"`,
		},
		{
			"country on boolean", `{"kind":"boolean","semantic":"country","format":"alpha-2"}`,
			"country semantic requires string type",
		},
		{
			"country with max bytes",
			`{"kind":"string","semantic":"country","format":"alpha-2","maxBytes":2}`,
			"country semantic cannot be combined with other string constraints",
		},
		{
			"country with max length",
			`{"kind":"string","semantic":"country","format":"alpha-2","maxLength":2}`,
			"country semantic cannot be combined with other string constraints",
		},
		{
			"country with pattern",
			`{"kind":"string","semantic":"country","format":"alpha-2","pattern":".."}`,
			"country semantic cannot be combined with other string constraints",
		},
		{
			"country with values",
			`{"kind":"string","semantic":"country","format":"alpha-2","values":["IT"]}`,
			"country semantic cannot be combined with other string constraints",
		},
		{
			"phone with max bytes", `{"kind":"string","semantic":"phone","maxBytes":16}`,
			"phone semantic cannot be combined with other string constraints",
		},
		{
			"phone with max length", `{"kind":"string","semantic":"phone","maxLength":16}`,
			"phone semantic cannot be combined with other string constraints",
		},
		{
			"phone with pattern", `{"kind":"string","semantic":"phone","pattern":".+"}`,
			"phone semantic cannot be combined with other string constraints",
		},
		{
			"phone with values", `{"kind":"string","semantic":"phone","values":["+390000000000"]}`,
			"phone semantic cannot be combined with other string constraints",
		},
		{
			"invalid currency", `{"kind":"decimal","precision":10,"scale":2,"semantic":"money","currency":"usd"}`,
			`invalid currency code "usd"`,
		},
		{
			"money on int", `{"kind":"int","bitSize":64,"semantic":"money"}`,
			"money semantic requires decimal type",
		},
		{
			"money on real float", `{"kind":"float","bitSize":64,"real":true,"semantic":"money"}`,
			"money semantic requires decimal type",
		},
		{
			"missing measurement unit", `{"kind":"decimal","precision":10,"scale":2,"semantic":"measurement"}`,
			"missing measurement unit",
		},
		{
			"invalid measurement unit",
			`{"kind":"decimal","precision":10,"scale":2,"semantic":"measurement","unit":"stone"}`,
			`invalid unit of measure "stone"`,
		},
		{"missing duration unit", `{"kind":"int","bitSize":64,"semantic":"duration"}`, "missing duration unit"},
		{
			"invalid duration unit", `{"kind":"int","bitSize":64,"semantic":"duration","unit":"month"}`,
			`invalid duration unit "month"`,
		},
		{
			"unexpected email currency", `{"kind":"string","semantic":"email","currency":"EUR"}`,
			"unexpected 'currency' key for email semantic",
		},
		{
			"unexpected phone format", `{"kind":"string","semantic":"phone","format":"alpha-2"}`,
			"unexpected 'format' key for phone semantic",
		},
		{
			"unexpected URL format", `{"kind":"string","semantic":"url","format":"absolute"}`,
			"unexpected 'format' key for URL semantic",
		},
		{
			"unexpected URL currency", `{"kind":"string","semantic":"url","currency":"EUR"}`,
			"unexpected 'currency' key for URL semantic",
		},
		{
			"unexpected URL unit", `{"kind":"string","semantic":"url","unit":"m"}`,
			"unexpected 'unit' key for URL semantic",
		},
		{
			"unexpected country currency without format",
			`{"kind":"string","semantic":"country","currency":"EUR"}`,
			"unexpected 'currency' key for country semantic",
		},
		{
			"unexpected country unit",
			`{"kind":"string","semantic":"country","format":"alpha-2","unit":"m"}`,
			"unexpected 'unit' key for country semantic",
		},
		{
			"unexpected money format",
			`{"kind":"decimal","precision":10,"scale":2,"semantic":"money","format":"integer"}`,
			"unexpected 'format' key for money semantic",
		},
		{
			"unexpected money unit",
			`{"kind":"decimal","precision":10,"scale":2,"semantic":"money","unit":"cent"}`,
			"unexpected 'unit' key for money semantic",
		},
		{
			"unexpected percentage unit",
			`{"kind":"decimal","precision":10,"scale":2,"semantic":"percentage","unit":"percent"}`,
			"unexpected 'unit' key for percentage semantic",
		},
		{
			"unexpected measurement format without unit",
			`{"kind":"decimal","precision":10,"scale":2,"semantic":"measurement","format":"metric"}`,
			"unexpected 'format' key for measurement semantic",
		},
		{
			"unexpected measurement currency",
			`{"kind":"decimal","precision":10,"scale":2,"semantic":"measurement","unit":"kg","currency":"EUR"}`,
			"unexpected 'currency' key for measurement semantic",
		},
		{
			"unexpected duration format without unit", `{"kind":"int","bitSize":64,"semantic":"duration","format":"integer"}`,
			"unexpected 'format' key for duration semantic",
		},
		{
			"unexpected duration currency", `{"kind":"int","bitSize":64,"semantic":"duration","unit":"second","currency":"EUR"}`,
			"unexpected 'currency' key for duration semantic",
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
			t.Fatal("expected an error")
		})
	}
}

// Test_TypeSemanticJSONRoundTrip tests canonical JSON encoding and round-trip
// decoding of type semantics.
func Test_TypeSemanticJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		type_ Type
		data  string
	}{
		{"no semantic", String(), `{"kind":"string"}`},
		{"email", String().AsEmail(), `{"kind":"string","semantic":"email"}`},
		{"phone", String().AsPhone(), `{"kind":"string","semantic":"phone"}`},
		{"URL", String().AsURL(), `{"kind":"string","semantic":"url"}`},
		{
			"country alpha-2", String().AsCountry(ISO3166Alpha2),
			`{"kind":"string","semantic":"country","format":"alpha-2"}`,
		},
		{
			"country alpha-3", String().AsCountry(ISO3166Alpha3),
			`{"kind":"string","semantic":"country","format":"alpha-3"}`,
		},
		{"money", Decimal(10, 2).AsMoney(), `{"kind":"decimal","semantic":"money","precision":10,"scale":2}`},
		{
			"money EUR", Decimal(10, 2).AsMoney().WithCurrency("EUR"),
			`{"kind":"decimal","semantic":"money","currency":"EUR","precision":10,"scale":2}`,
		},
		{"percentage", Decimal(18, 4).AsPercentage(), `{"kind":"decimal","semantic":"percentage","precision":18,"scale":4}`},
		{
			"measurement", Decimal(10, 2).AsMeasurement(Kilogram),
			`{"kind":"decimal","semantic":"measurement","unit":"kg","precision":10,"scale":2}`,
		},
		{
			"duration", Int(64).AsDuration(Millisecond),
			`{"kind":"int","semantic":"duration","unit":"millisecond","bitSize":64}`,
		},
		{"array email", Array(String().AsEmail()), `{"kind":"array","elementType":{"kind":"string","semantic":"email"}}`},
		{
			"nested money", Array(Map(Decimal(10, 2).AsMoney().WithCurrency("EUR"))),
			`{"kind":"array","elementType":{"kind":"map","elementType":` +
				`{"kind":"decimal","semantic":"money","currency":"EUR","precision":10,"scale":2}}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := json.Marshal(test.type_)
			if err != nil {
				t.Fatalf("cannot marshal type: %v", err)
			}
			if string(got) != test.data {
				t.Fatalf("expected %q, got %q", test.data, got)
			}
			var type_ Type
			err = json.Unmarshal(got, &type_)
			if err != nil {
				t.Fatalf("cannot unmarshal type: %v", err)
			}
			if !Equal(test.type_, type_) {
				t.Fatal("round trip changed type")
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
			t.Errorf("%d is not valid", test.unit)
		}
		if got := test.unit.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		got, ok := UnitOfMeasureByName(test.name)
		if !ok || got != test.unit {
			t.Errorf("expected unit %d and true, got %d and %t", test.unit, got, ok)
		}
	}
	if (1 <= InvalidUnitOfMeasure && int(InvalidUnitOfMeasure) <= len(unitOfMeasureName)) ||
		(1 <= UnitOfMeasure(-1) && int(UnitOfMeasure(-1)) <= len(unitOfMeasureName)) ||
		(1 <= UnitOfMeasure(127) && int(UnitOfMeasure(127)) <= len(unitOfMeasureName)) ||
		InvalidUnitOfMeasure.String() != "Invalid" {
		t.Fatal("invalid unit of measure is valid or has an unexpected name")
	}
	if got, ok := UnitOfMeasureByName("kilogram"); ok || got != InvalidUnitOfMeasure {
		t.Fatalf("expected invalid unit and false, got %d and %t", got, ok)
	}
}
