// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"testing"
)

// Test_CountryFormat tests country format names, lookup, and validation.
func Test_CountryFormat(t *testing.T) {

	tests := []struct {
		format CountryFormat
		name   string
	}{
		{ISO3166Alpha2, "iso_3166_1_alpha_2"},
		{ISO3166Alpha3, "iso_3166_1_alpha_3"},
	}
	for _, test := range tests {
		if !test.format.Valid() {
			t.Errorf("%d is not valid", test.format)
		}
		if got := test.format.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		got, ok := CountryFormatByName(test.name)
		if !ok || got != test.format {
			t.Errorf("expected format %d and true, got %d and %t", test.format, got, ok)
		}
	}
	if InvalidCountryFormat.Valid() || CountryFormat(-1).Valid() || CountryFormat(127).Valid() ||
		InvalidCountryFormat.String() != "Invalid" {
		t.Fatal("invalid country format is valid or has an unexpected name")
	}
	if got, ok := CountryFormatByName("ISO3166Alpha2"); ok || got != InvalidCountryFormat {
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
		if !test.unit.Valid() {
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
	if InvalidDurationUnit.Valid() || DurationUnit(-1).Valid() || DurationUnit(127).Valid() ||
		InvalidDurationUnit.String() != "Invalid" {
		t.Fatal("invalid duration unit is valid or has an unexpected name")
	}
	if got, ok := DurationUnitByName("seconds"); ok || got != InvalidDurationUnit {
		t.Fatalf("expected invalid unit and false, got %d and %t", got, ok)
	}

}

// Test_PercentageFormat tests percentage format names, lookup, and validation.
func Test_PercentageFormat(t *testing.T) {

	tests := []struct {
		format PercentageFormat
		name   string
	}{
		{FractionPercentage, "fraction"},
		{WholePercentage, "whole"},
	}
	for _, test := range tests {
		if !test.format.Valid() {
			t.Errorf("%d is not valid", test.format)
		}
		if got := test.format.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		got, ok := PercentageFormatByName(test.name)
		if !ok || got != test.format {
			t.Errorf("expected format %d and true, got %d and %t", test.format, got, ok)
		}
	}
	if InvalidPercentageFormat.Valid() || PercentageFormat(-1).Valid() || PercentageFormat(127).Valid() ||
		InvalidPercentageFormat.String() != "Invalid" {
		t.Fatal("invalid percentage format is valid or has an unexpected name")
	}
	if got, ok := PercentageFormatByName("percent"); ok || got != InvalidPercentageFormat {
		t.Fatalf("expected invalid format and false, got %d and %t", got, ok)
	}

}

// Test_SemanticCompatibility tests semantic compatibility with property types.
func Test_SemanticCompatibility(t *testing.T) {

	valid := []struct {
		name     string
		type_    Type
		semantic Semantic
	}{
		{"no semantic on generic type", Parameter("T"), nil},
		{"email", String(), Email()},
		{"phone", String(), Phone()},
		{"URL", String(), URL()},
		{"country", String(), Country(ISO3166Alpha2)},
		{"formatted datetime", String(), FormattedDateTime("dd/MM/yyyy HH:mm:ss")},
		{"money int", Int(32), Money()},
		{"money unsigned int", Int(32).Unsigned(), Money()},
		{"money decimal", Decimal(10, 2), Money()},
		{"money real float", Float(64).Real(), Money()},
		{"percentage int", Int(32), Percentage(WholePercentage)},
		{"percentage decimal", Decimal(10, 2), Percentage(FractionPercentage)},
		{"percentage real float", Float(32).Real(), Percentage(FractionPercentage)},
		{"measurement int", Int(64), Measurement()},
		{"measurement decimal", Decimal(10, 2), Measurement().WithUnitOfMeasure(Kilogram)},
		{"measurement real float", Float(64).Real(), Measurement()},
		{"duration int", Int(32), Duration(Second)},
		{"duration decimal", Decimal(10, 3), Duration(Millisecond)},
		{"duration real float", Float(32).Real(), Duration(Hour)},
		{"array email", Array(String()), Email()},
		{"map country", Map(String()), Country(ISO3166Alpha3)},
		{"array money", Array(Int(32)), Money()},
		{"map percentage", Map(Int(32)), Percentage(WholePercentage)},
		{"nested array and map measurement", Array(Map(Decimal(10, 2))), Measurement().WithUnitOfMeasure(Kilogram)},
		{"nested map and array duration", Map(Array(Float(32).Real())), Duration(Second)},
	}
	for _, test := range valid {
		t.Run("valid "+test.name, func(t *testing.T) {

			_, err := ObjectOf([]Property{{Name: "value", Type: test.type_, Semantic: test.semantic}})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

		})
	}

	invalid := []struct {
		name     string
		type_    Type
		semantic Semantic
	}{
		{"boolean country", Boolean(), Country(ISO3166Alpha2)},
		{"object email", Object([]Property{{Name: "value", Type: String()}}), Email()},
		{"JSON measurement", JSON(), Measurement()},
		{"generic country", Parameter("T"), Country(ISO3166Alpha2)},
		{"array boolean country", Array(Boolean()), Country(ISO3166Alpha2)},
		{"map object email", Map(Object([]Property{{Name: "value", Type: String()}})), Email()},
		{"array JSON measurement", Array(JSON()), Measurement()},
		{"map generic country", Map(Parameter("T")), Country(ISO3166Alpha2)},
		{"nested array and map ordinary float money", Array(Map(Float(32))), Money()},
		{"ordinary float money", Float(32), Money()},
		{"ordinary float percentage", Float(64), Percentage(FractionPercentage)},
		{"ordinary float measurement", Float(32), Measurement()},
		{"ordinary float duration", Float(64), Duration(Second)},
		{"datetime type and formatted datetime", DateTime(), FormattedDateTime("2006-01-02")},
		{"string money", String(), Money()},
	}
	for _, test := range invalid {
		t.Run("invalid "+test.name, func(t *testing.T) {

			_, err := ObjectOf([]Property{{Name: "value", Type: test.type_, Semantic: test.semantic}})
			if err == nil {
				t.Fatal("expected an error")
			}

		})
	}

	p := Property{Name: "value", Type: Boolean(), Semantic: Email()}
	if _, err := p.MarshalJSON(); err == nil {
		t.Fatal("expected Property.MarshalJSON to reject an incompatible semantic")
	}

}

// Test_SemanticConstructorPanics tests that semantic constructors reject
// invalid arguments.
func Test_SemanticConstructorPanics(t *testing.T) {

	tests := []struct {
		name string
		f    func()
	}{
		{"invalid country format", func() { Country(InvalidCountryFormat) }},
		{"negative country format", func() { Country(CountryFormat(-1)) }},
		{"invalid percentage format", func() { Percentage(InvalidPercentageFormat) }},
		{"negative percentage format", func() { Percentage(PercentageFormat(-1)) }},
		{"invalid duration unit", func() { Duration(InvalidDurationUnit) }},
		{"negative duration unit", func() { Duration(DurationUnit(-1)) }},
		{"empty datetime format", func() { FormattedDateTime("") }},
		{"invalid UTF-8 datetime format", func() { FormattedDateTime(string([]byte{0xff})) }},
		{"NUL in datetime format", func() { FormattedDateTime("yyyy\x00MM") }},
		{"empty currency", func() { Money().WithCurrency("") }},
		{"short currency", func() { Money().WithCurrency("US") }},
		{"long currency", func() { Money().WithCurrency("USDD") }},
		{"lowercase currency", func() { Money().WithCurrency("usd") }},
		{"non-letter currency", func() { Money().WithCurrency("U1D") }},
		{"invalid unit of measure", func() { Measurement().WithUnitOfMeasure(InvalidUnitOfMeasure) }},
		{"negative unit of measure", func() { Measurement().WithUnitOfMeasure(UnitOfMeasure(-1)) }},
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

// Test_SemanticConstructors tests semantic constructors, kinds, and options.
func Test_SemanticConstructors(t *testing.T) {

	tests := []struct {
		semantic Semantic
		kind     SemanticKind
	}{
		{Email(), EmailSemanticKind},
		{Phone(), PhoneSemanticKind},
		{URL(), URLSemanticKind},
		{Country(ISO3166Alpha2), CountrySemanticKind},
		{FormattedDateTime("dd/MM/yyyy"), DateTimeSemanticKind},
		{Money(), MoneySemanticKind},
		{Percentage(FractionPercentage), PercentageSemanticKind},
		{Measurement(), MeasurementSemanticKind},
		{Duration(Second), DurationSemanticKind},
	}
	for _, test := range tests {
		if got := test.semantic.Kind(); got != test.kind {
			t.Errorf("expected kind %v, got %v", test.kind, got)
		}
	}

	if got := Country(ISO3166Alpha3).Format(); got != ISO3166Alpha3 {
		t.Errorf("expected country format %v, got %v", ISO3166Alpha3, got)
	}
	if got := FormattedDateTime("Cafe\u0301").Format(); got != "Caf\u00e9" {
		t.Errorf("expected normalized datetime format %q, got %q", "Caf\u00e9", got)
	}
	if got := Percentage(WholePercentage).Format(); got != WholePercentage {
		t.Errorf("expected percentage format %v, got %v", WholePercentage, got)
	}
	if got := Duration(Week).Unit(); got != Week {
		t.Errorf("expected duration unit %v, got %v", Week, got)
	}
	if currency, ok := Money().Currency(); ok || currency != "" {
		t.Errorf("expected no currency, got %q and %t", currency, ok)
	}
	if unit, ok := Measurement().UnitOfMeasure(); ok || unit != InvalidUnitOfMeasure {
		t.Errorf("expected no unit, got %v and %t", unit, ok)
	}

}

// Test_SemanticEquality tests semantic equality, including semantic-specific
// options.
func Test_SemanticEquality(t *testing.T) {

	tests := []struct {
		name     string
		type_    Type
		semantic Semantic
		other    Semantic
		equal    bool
	}{
		{"both missing", String(), nil, nil, true},
		{"equal email", String(), Email(), Email(), true},
		{"equal phone", String(), Phone(), Phone(), true},
		{"equal URL", String(), URL(), URL(), true},
		{"missing and present", String(), nil, Email(), false},
		{"different kinds", String(), Email(), Phone(), false},
		{"equal country format", String(), Country(ISO3166Alpha2), Country(ISO3166Alpha2), true},
		{"different country format", String(), Country(ISO3166Alpha2), Country(ISO3166Alpha3), false},
		{
			"equal datetime format",
			String(),
			FormattedDateTime("yyyy-MM-dd"),
			FormattedDateTime("yyyy-MM-dd"),
			true,
		},
		{
			"different datetime format",
			String(),
			FormattedDateTime("yyyy-MM-dd"),
			FormattedDateTime("dd/MM/yyyy"),
			false,
		},
		{
			"equal currency",
			Decimal(10, 2),
			Money().WithCurrency("USD"),
			Money().WithCurrency("USD"),
			true,
		},
		{
			"different currency",
			Decimal(10, 2),
			Money().WithCurrency("USD"),
			Money().WithCurrency("EUR"),
			false,
		},
		{"equal money without currency", Decimal(10, 2), Money(), Money(), true},
		{"missing and present currency", Decimal(10, 2), Money(), Money().WithCurrency("USD"), false},
		{
			"equal percentage format",
			Decimal(10, 2),
			Percentage(FractionPercentage),
			Percentage(FractionPercentage),
			true,
		},
		{
			"different percentage format",
			Decimal(10, 2),
			Percentage(FractionPercentage),
			Percentage(WholePercentage),
			false,
		},
		{
			"equal unit of measure",
			Decimal(10, 2),
			Measurement().WithUnitOfMeasure(Kilogram),
			Measurement().WithUnitOfMeasure(Kilogram),
			true,
		},
		{
			"different unit of measure",
			Decimal(10, 2),
			Measurement().WithUnitOfMeasure(Kilogram),
			Measurement().WithUnitOfMeasure(Meter),
			false,
		},
		{"equal measurement without unit", Decimal(10, 2), Measurement(), Measurement(), true},
		{
			"missing and present unit of measure",
			Decimal(10, 2),
			Measurement(),
			Measurement().WithUnitOfMeasure(Kilogram),
			false,
		},
		{"equal duration unit", Decimal(10, 2), Duration(Second), Duration(Second), true},
		{"different duration unit", Decimal(10, 2), Duration(Second), Duration(Hour), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			t1 := Object([]Property{{Name: "value", Type: test.type_, Semantic: test.semantic}})
			t2 := Object([]Property{{Name: "value", Type: test.type_, Semantic: test.other}})
			if got := Equal(t1, t2); got != test.equal {
				t.Fatalf("expected equality %t, got %t", test.equal, got)
			}
			if got := Equal(t2, t1); got != test.equal {
				t.Fatalf("expected reverse equality %t, got %t", test.equal, got)
			}

		})
	}

}

// Test_SemanticCopyOnWrite tests that semantic configuration methods do not
// mutate their receivers.
func Test_SemanticCopyOnWrite(t *testing.T) {

	money := Money()
	usd := money.WithCurrency("USD")
	if currency, ok := money.Currency(); ok || currency != "" {
		t.Fatalf("Money was mutated: got %q and %t", currency, ok)
	}
	if currency, ok := usd.Currency(); !ok || currency != "USD" {
		t.Fatalf("expected USD, got %q and %t", currency, ok)
	}
	eur := usd.WithCurrency("EUR")
	if currency, _ := usd.Currency(); currency != "USD" {
		t.Fatalf("WithCurrency mutated its receiver: got %q", currency)
	}
	if currency, _ := eur.Currency(); currency != "EUR" {
		t.Fatalf("expected EUR, got %q", currency)
	}
	measurement := Measurement()
	weight := measurement.WithUnitOfMeasure(Kilogram)
	if unit, ok := measurement.UnitOfMeasure(); ok || unit != InvalidUnitOfMeasure {
		t.Fatalf("Measurement was mutated: got %v and %t", unit, ok)
	}
	if unit, ok := weight.UnitOfMeasure(); !ok || unit != Kilogram {
		t.Fatalf("expected kilogram, got %v and %t", unit, ok)
	}
	length := weight.WithUnitOfMeasure(Meter)
	if unit, _ := weight.UnitOfMeasure(); unit != Kilogram {
		t.Fatalf("WithUnitOfMeasure mutated its receiver: got %v", unit)
	}
	if unit, _ := length.UnitOfMeasure(); unit != Meter {
		t.Fatalf("expected meter, got %v", unit)
	}

}

// Test_SemanticJSONErrors tests rejection of invalid semantic JSON representations.
func Test_SemanticJSONErrors(t *testing.T) {

	stringSemantic := `{"name":"value","type":{"kind":"string"},"semantic":`
	intSemantic := `{"name":"value","type":{"kind":"int","bitSize":32},"semantic":`

	tests := []struct {
		name string
		data string
		err  string
	}{
		{"null", stringSemantic + `null}`, "invalid semantic syntax"},
		{"array", stringSemantic + `[]}`, "invalid semantic syntax"},
		{
			"repeated semantic",
			stringSemantic + `{"kind":"email"},"semantic":{"kind":"email"}}`,
			"repeated 'semantic' key",
		},
		{"missing kind", stringSemantic + `{}}`, "missing 'kind' key"},
		{
			"repeated kind",
			stringSemantic + `{"kind":"email","kind":"phone"}}`,
			"repeated 'kind' key",
		},
		{"unknown kind", stringSemantic + `{"kind":"unknown"}}`, `invalid semantic kind "unknown"`},
		{"unknown option", stringSemantic + `{"kind":"email","option":true}}`, `unknown semantic key "option"`},
		{"kind wrong type", stringSemantic + `{"kind":1}}`, "invalid semantic kind"},
		{"format wrong type", stringSemantic + `{"kind":"country","format":1}}`, "invalid semantic format"},
		{"currency wrong type", intSemantic + `{"kind":"money","currency":1}}`, "invalid semantic currency"},
		{"unit wrong type", intSemantic + `{"kind":"duration","unit":1}}`, "invalid semantic unit"},
		{
			"repeated format",
			stringSemantic +
				`{"kind":"country","format":"iso_3166_1_alpha_2","format":"iso_3166_1_alpha_3"}}`,
			"repeated 'format' key",
		},
		{
			"repeated currency",
			intSemantic + `{"kind":"money","currency":"USD","currency":"EUR"}}`,
			"repeated 'currency' key",
		},
		{
			"repeated unit",
			intSemantic + `{"kind":"duration","unit":"second","unit":"hour"}}`,
			"repeated 'unit' key",
		},
		{"missing country format", stringSemantic + `{"kind":"country"}}`, "missing country format"},
		{"missing datetime format", stringSemantic + `{"kind":"datetime"}}`, "missing datetime format"},
		{"missing percentage format", intSemantic + `{"kind":"percentage"}}`, "missing percentage format"},
		{"missing duration unit", intSemantic + `{"kind":"duration"}}`, "missing duration unit"},
		{
			"invalid country format",
			stringSemantic + `{"kind":"country","format":"iso_2"}}`,
			`invalid country format "iso_2"`,
		},
		{
			"empty country format",
			stringSemantic + `{"kind":"country","format":""}}`,
			`invalid country format ""`,
		},
		{
			"empty datetime format",
			stringSemantic + `{"kind":"datetime","format":""}}`,
			"datetime format is empty",
		},
		{
			"NUL in datetime format",
			stringSemantic + `{"kind":"datetime","format":"yyyy\u0000MM"}}`,
			"contains NUL byte",
		},
		{"invalid currency", intSemantic + `{"kind":"money","currency":"usd"}}`, `invalid currency code "usd"`},
		{"empty currency", intSemantic + `{"kind":"money","currency":""}}`, `invalid currency code ""`},
		{
			"invalid percentage format",
			intSemantic + `{"kind":"percentage","format":"percent"}}`,
			`invalid percentage format "percent"`,
		},
		{
			"invalid measurement unit",
			intSemantic + `{"kind":"measurement","unit":"stone"}}`,
			`invalid unit of measure "stone"`,
		},
		{
			"empty measurement unit",
			intSemantic + `{"kind":"measurement","unit":""}}`,
			`invalid unit of measure ""`,
		},
		{
			"invalid duration unit",
			intSemantic + `{"kind":"duration","unit":"months"}}`,
			`invalid duration unit "months"`,
		},
		{
			"empty duration unit",
			intSemantic + `{"kind":"duration","unit":""}}`,
			`invalid duration unit ""`,
		},
		{
			"unexpected email option",
			stringSemantic + `{"kind":"email","format":"RFC 5322"}}`,
			"unexpected option for email semantic",
		},
		{
			"unexpected money option",
			intSemantic + `{"kind":"money","unit":"cent"}}`,
			"unexpected option for money semantic",
		},
		{
			"unexpected measurement option",
			intSemantic + `{"kind":"measurement","currency":"USD"}}`,
			"unexpected option for measurement semantic",
		},
		{
			"unexpected duration option",
			intSemantic + `{"kind":"duration","unit":"second","format":"whole"}}`,
			"unexpected option for duration semantic",
		},
		{
			"incompatible type",
			`{"name":"value","type":{"kind":"boolean"},"semantic":` +
				`{"kind":"country","format":"iso_3166_1_alpha_2"}}`,
			"country semantic requires string type",
		},
		{
			"generic type",
			`{"name":"value","type":{"kind":"T"},"semantic":` +
				`{"kind":"country","format":"iso_3166_1_alpha_2"}}`,
			"semantic cannot be used with a generic type",
		},
		{
			"ordinary float",
			`{"name":"value","type":{"kind":"float","bitSize":64},"semantic":{"kind":"money"}}`,
			"money semantic requires an int, decimal, or real float type",
		},
		{
			"array with incompatible element type",
			`{"name":"value","type":{"kind":"array","elementType":{"kind":"boolean"}},` +
				`"semantic":{"kind":"email"}}`,
			"email semantic requires string type",
		},
		{
			"map with ordinary float value",
			`{"name":"value","type":{"kind":"map","elementType":{"kind":"float","bitSize":64}},` +
				`"semantic":{"kind":"money"}}`,
			"money semantic requires an int, decimal, or real float type",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			var p Property
			err := json.Unmarshal([]byte(test.data), &p)
			if err == nil {
				t.Fatal("expected an error")
			}
			if err.Error() != test.err {
				t.Fatalf("expected error %q, got %q", test.err, err)
			}

		})
	}

}

// Test_SemanticJSONRoundTrip tests canonical JSON encoding and round-trip
// decoding of semantics.
func Test_SemanticJSONRoundTrip(t *testing.T) {

	tests := []struct {
		name     string
		property Property
		data     string
	}{
		{
			"email",
			Property{Name: "email", Type: String(), Semantic: Email()},
			`{"name":"email","type":{"kind":"string"},"semantic":{"kind":"email"},"description":""}`,
		},
		{
			"phone",
			Property{Name: "phone", Type: String(), Semantic: Phone()},
			`{"name":"phone","type":{"kind":"string"},"semantic":{"kind":"phone"},"description":""}`,
		},
		{
			"URL",
			Property{Name: "url", Type: String(), Semantic: URL()},
			`{"name":"url","type":{"kind":"string"},"semantic":{"kind":"url"},"description":""}`,
		},
		{
			"country alpha-2",
			Property{Name: "country", Type: String(), Semantic: Country(ISO3166Alpha2)},
			`{"name":"country","type":{"kind":"string"},` +
				`"semantic":{"kind":"country","format":"iso_3166_1_alpha_2"},"description":""}`,
		},
		{
			"formatted datetime",
			Property{Name: "updated_at", Type: String(), Semantic: FormattedDateTime("dd/MM/yyyy HH:mm:ss")},
			`{"name":"updated_at","type":{"kind":"string"},` +
				`"semantic":{"kind":"datetime","format":"dd/MM/yyyy HH:mm:ss"},"description":""}`,
		},
		{
			"money without currency",
			Property{Name: "amount", Type: Decimal(10, 2), Semantic: Money()},
			`{"name":"amount","type":{"kind":"decimal","precision":10,"scale":2},` +
				`"semantic":{"kind":"money"},"description":""}`,
		},
		{
			"money EUR",
			Property{Name: "amount", Type: Decimal(10, 2), Semantic: Money().WithCurrency("EUR")},
			`{"name":"amount","type":{"kind":"decimal","precision":10,"scale":2},` +
				`"semantic":{"kind":"money","currency":"EUR"},"description":""}`,
		},
		{
			"percentage fraction",
			Property{Name: "ratio", Type: Float(64).Real(), Semantic: Percentage(FractionPercentage)},
			`{"name":"ratio","type":{"kind":"float","bitSize":64,"real":true},` +
				`"semantic":{"kind":"percentage","format":"fraction"},"description":""}`,
		},
		{
			"measurement without unit",
			Property{Name: "weight", Type: Decimal(10, 2), Semantic: Measurement()},
			`{"name":"weight","type":{"kind":"decimal","precision":10,"scale":2},` +
				`"semantic":{"kind":"measurement"},"description":""}`,
		},
		{
			"measurement kilogram",
			Property{
				Name: "weight", Type: Decimal(10, 2), Semantic: Measurement().WithUnitOfMeasure(Kilogram),
			},
			`{"name":"weight","type":{"kind":"decimal","precision":10,"scale":2},` +
				`"semantic":{"kind":"measurement","unit":"kg"},"description":""}`,
		},
		{
			"duration millisecond",
			Property{Name: "elapsed", Type: Int(64), Semantic: Duration(Millisecond)},
			`{"name":"elapsed","type":{"kind":"int","bitSize":64},` +
				`"semantic":{"kind":"duration","unit":"millisecond"},"description":""}`,
		},
		{
			"array email",
			Property{Name: "recipients", Type: Array(String()), Semantic: Email()},
			`{"name":"recipients","type":{"kind":"array","elementType":{"kind":"string"}},` +
				`"semantic":{"kind":"email"},"description":""}`,
		},
		{
			"nested array and map money",
			Property{
				Name: "amounts", Type: Array(Map(Decimal(10, 2))), Semantic: Money().WithCurrency("EUR"),
			},
			`{"name":"amounts","type":{"kind":"array","elementType":{"kind":"map",` +
				`"elementType":{"kind":"decimal","precision":10,"scale":2}}},` +
				`"semantic":{"kind":"money","currency":"EUR"},"description":""}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			got, err := json.Marshal(test.property)
			if err != nil {
				t.Fatalf("cannot marshal property: %v", err)
			}
			if string(got) != test.data {
				t.Fatalf("expected %q, got %q", test.data, got)
			}

			var property Property
			if err := json.Unmarshal(got, &property); err != nil {
				t.Fatalf("cannot unmarshal property: %v", err)
			}
			if err := sameProperty(test.property, property); err != nil {
				t.Fatal(err)
			}

		})
	}

}

// Test_SemanticJSONWithoutSemantic tests JSON handling when a property has no
// semantic.
func Test_SemanticJSONWithoutSemantic(t *testing.T) {

	data := []byte(`{"name":"value","type":{"kind":"string"},"description":""}`)
	p := Property{Name: "old", Type: String(), Semantic: Email()}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("cannot unmarshal property: %v", err)
	}
	if p.Semantic != nil {
		t.Fatalf("expected no semantic, got %#v", p.Semantic)
	}
	got, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("cannot marshal property: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("expected %q, got %q", data, got)
	}

}

// Test_SemanticJSONWithReorderedKeysAndNestedProperty tests reordered JSON keys
// and semantics on nested properties.
func Test_SemanticJSONWithReorderedKeysAndNestedProperty(t *testing.T) {

	propertyData := []byte(`{"semantic":{"format":"iso_3166_1_alpha_3","kind":"country"},` +
		`"type":{"kind":"string"},"name":"country"}`)
	var p Property
	if err := json.Unmarshal(propertyData, &p); err != nil {
		t.Fatalf("cannot unmarshal property with reordered keys: %v", err)
	}
	if err := sameProperty(Property{Name: "country", Type: String(), Semantic: Country(ISO3166Alpha3)}, p); err != nil {
		t.Fatal(err)
	}

	typeData := `{"kind":"object","properties":[{"name":"profile","type":{"kind":"object","properties":[` +
		`{"name":"country","type":{"kind":"string"},"semantic":{"kind":"country",` +
		`"format":"iso_3166_1_alpha_2"},"description":""}]},"description":""}]}`
	want := Object([]Property{{
		Name: "profile",
		Type: Object([]Property{{Name: "country", Type: String(), Semantic: Country(ISO3166Alpha2)}}),
	}})
	got, err := Parse(typeData)
	if err != nil {
		t.Fatalf("cannot parse nested semantic: %v", err)
	}
	if !Equal(want, got) {
		t.Fatal("parsed type does not preserve the nested semantic")
	}
	marshaled, err := got.MarshalJSON()
	if err != nil {
		t.Fatalf("cannot marshal nested semantic: %v", err)
	}
	if string(marshaled) != typeData {
		t.Fatalf("expected %q, got %q", typeData, marshaled)
	}

}

// Test_PropertyReadsShareSemanticInstances tests that property reads preserve
// shared semantic instances.
func Test_PropertyReadsShareSemanticInstances(t *testing.T) {

	country := Country(ISO3166Alpha2)
	email := Email()
	weight := Measurement().WithUnitOfMeasure(Kilogram)
	eventTime := FormattedDateTime("yyyy-MM-dd HH:mm:ss")
	balance := Money().WithCurrency("EUR")
	schema := Object([]Property{
		{Name: "country", Type: String(), Semantic: country},
		{Name: "profile", Type: Object([]Property{
			{Name: "email", Type: String(), Semantic: email, ReadOptional: true},
			{Name: "weight", Type: Decimal(10, 2), Semantic: weight},
		})},
		{Name: "events", Type: Array(Object([]Property{
			{Name: "created_at", Type: String(), Semantic: eventTime},
		}))},
		{Name: "balances", Type: Map(Object([]Property{
			{Name: "amount", Type: Decimal(10, 2), Semantic: balance},
		}))},
	})
	properties := schema.Properties()

	p, ok := properties.ByName("country")
	if !ok || p.Semantic != country {
		t.Fatal("ByName did not preserve the semantic instance")
	}
	p.Semantic = Email()
	p, ok = properties.ByName("country")
	if !ok || p.Semantic != country {
		t.Fatal("modifying a returned property changed the stored semantic")
	}
	p, err := properties.ByPath("profile.email")
	if err != nil || p.Semantic != email {
		t.Fatal("ByPath did not preserve the semantic instance")
	}
	p, err = properties.ByPathSlice([]string{"profile", "weight"})
	if err != nil || p.Semantic != weight {
		t.Fatal("ByPathSlice did not preserve the semantic instance")
	}
	if got := properties.Slice()[0].Semantic; got != country {
		t.Fatal("Slice did not preserve the semantic instance")
	}
	foundCountry := false
	for _, p := range properties.All() {
		if p.Name == "country" && p.Semantic != country {
			t.Fatal("All did not preserve the semantic instance")
		}
		foundCountry = foundCountry || p.Name == "country"
	}
	if !foundCountry {
		t.Fatal("All did not return country")
	}
	foundEmail, foundEventTime, foundBalance := false, false, false
	for path, p := range properties.WalkAll() {
		switch path {
		case "profile.email":
			foundEmail = true
			if p.Semantic != email {
				t.Fatal("WalkAll did not preserve the semantic instance in an object")
			}
		case "events.created_at":
			foundEventTime = true
			if p.Semantic != eventTime {
				t.Fatal("WalkAll did not preserve the semantic instance in an array element")
			}
		case "balances.amount":
			foundBalance = true
			if p.Semantic != balance {
				t.Fatal("WalkAll did not preserve the semantic instance in a map value")
			}
		}
	}
	if !foundEmail || !foundEventTime || !foundBalance {
		t.Fatalf(
			"WalkAll did not return all nested semantic properties: %t, %t, %t",
			foundEmail, foundEventTime, foundBalance,
		)
	}
	foundWeight := false
	for path, p := range properties.WalkObjects() {
		if path == "profile.weight" {
			foundWeight = true
			if p.Semantic != weight {
				t.Fatal("WalkObjects did not preserve the semantic instance")
			}
		}
	}
	if !foundWeight {
		t.Fatal("WalkObjects did not return profile.weight")
	}

}

// Test_SchemaTransformationsPreserveSemantics tests that schema transformations
// preserve property semantics.
func Test_SchemaTransformationsPreserveSemantics(t *testing.T) {

	country := Country(ISO3166Alpha2)
	email := Email()
	weight := Measurement().WithUnitOfMeasure(Kilogram)
	schema := Object([]Property{
		{Name: "country", Type: String(), Semantic: country},
		{Name: "profile", Type: Object([]Property{
			{Name: "email", Type: String(), Semantic: email, ReadOptional: true},
			{Name: "weight", Type: Decimal(10, 2), Semantic: weight},
		})},
	})

	filtered := Filter(schema, func(p Property) bool { return p.Name == "country" })
	if p, ok := filtered.Properties().ByName("country"); !ok || !equalSemantics(p.Semantic, country) {
		t.Fatal("Filter did not preserve the semantic")
	}
	pruned := Prune(schema, func(path string) bool { return path == "profile.email" })
	if p, err := pruned.Properties().ByPath("profile.email"); err != nil || !equalSemantics(p.Semantic, email) {
		t.Fatal("Prune did not preserve the semantic")
	}
	prunedAtPath, err := PruneAtPath(schema, "profile.weight")
	if err != nil {
		t.Fatalf("PruneAtPath returned an error: %v", err)
	}
	if p, err := prunedAtPath.Properties().ByPath("profile.weight"); err != nil || !equalSemantics(p.Semantic, weight) {
		t.Fatal("PruneAtPath did not preserve the semantic")
	}
	asDestination := AsRole(schema, Destination)
	if p, err := asDestination.Properties().ByPath("profile.email"); err != nil || !equalSemantics(p.Semantic, email) {
		t.Fatal("AsRole did not preserve the semantic")
	}

}

// Test_SemanticKind tests semantic kind names, lookup, and validation.
func Test_SemanticKind(t *testing.T) {

	tests := []struct {
		kind SemanticKind
		name string
	}{
		{EmailSemanticKind, "email"},
		{PhoneSemanticKind, "phone"},
		{URLSemanticKind, "url"},
		{CountrySemanticKind, "country"},
		{DateTimeSemanticKind, "datetime"},
		{MoneySemanticKind, "money"},
		{PercentageSemanticKind, "percentage"},
		{MeasurementSemanticKind, "measurement"},
		{DurationSemanticKind, "duration"},
	}
	for _, test := range tests {
		if !test.kind.Valid() {
			t.Errorf("%d is not valid", test.kind)
		}
		if got := test.kind.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		got, ok := SemanticKindByName(test.name)
		if !ok || got != test.kind {
			t.Errorf("expected kind %d and true, got %d and %t", test.kind, got, ok)
		}
	}
	if InvalidSemanticKind.Valid() || SemanticKind(-1).Valid() || SemanticKind(127).Valid() ||
		InvalidSemanticKind.String() != "Invalid" {
		t.Fatal("invalid semantic kind is valid or has an unexpected name")
	}
	if got, ok := SemanticKindByName("Email"); ok || got != InvalidSemanticKind {
		t.Fatalf("expected invalid kind and false, got %d and %t", got, ok)
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
	}
	for _, test := range tests {
		if !test.unit.Valid() {
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
	if InvalidUnitOfMeasure.Valid() || UnitOfMeasure(-1).Valid() || UnitOfMeasure(127).Valid() ||
		InvalidUnitOfMeasure.String() != "Invalid" {
		t.Fatal("invalid unit of measure is valid or has an unexpected name")
	}
	if got, ok := UnitOfMeasureByName("kilogram"); ok || got != InvalidUnitOfMeasure {
		t.Fatalf("expected invalid unit and false, got %d and %t", got, ok)
	}

}
