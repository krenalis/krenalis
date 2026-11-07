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
		{YearSemantics, "year"},
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
		{Semantics(10), "%!Semantics(10)"},
		{Semantics(20), "%!Semantics(20)"},
		{Semantics(127), "%!Semantics(127)"},
		{CountryFormat(-128), "%!CountryFormat(-128)"},
		{CountryFormat(-1), "%!CountryFormat(-1)"},
		{CountryFormat(0), "%!CountryFormat(0)"},
		{CountryFormat(1), "%!CountryFormat(1)"},
		{CountryFormat(4), "%!CountryFormat(4)"},
		{CountryFormat(127), "%!CountryFormat(127)"},
		{Unit(-128), "%!Unit(-128)"},
		{Unit(-1), "%!Unit(-1)"},
		{Unit(0), "%!Unit(0)"},
		{Unit(29), "%!Unit(29)"},
		{Unit(127), "%!Unit(127)"},
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
		{"year int", Int(16).AsYear(), YearSemantics},
		{"year unsigned int", Int(16).Unsigned().AsYear(), YearSemantics},
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
	if got := Decimal(10, 2).AsMeasurement(Kilogram).Unit(); got != Kilogram {
		t.Fatalf("expected kilogram, got %s", got)
	}
	if got := Int(64).AsDuration(Week).Unit(); got != Week {
		t.Fatalf("expected week, got %s", got)
	}
}

// Test_TypeSemanticsConfigurationPanics tests invalid semantics configuration.
func Test_TypeSemanticsConfigurationPanics(t *testing.T) {
	tests := []struct {
		name string
		f    func()
	}{
		{"year on string", func() { String().AsYear() }},
		{"year on float", func() { Float(64).Real().AsYear() }},
		{"year already has semantics", func() { Int(16).AsYear().AsYear() }},
		{"year outside signed range", func() { Int(16).WithIntRange(-10, 0).AsYear() }},
		{"year outside unsigned range", func() { Int(16).Unsigned().WithUnsignedRange(10000, 20000).AsYear() }},
		{"year signed lower bound", func() { Int(16).AsYear().WithIntRange(0, 9999) }},
		{"year signed upper bound", func() { Int(16).AsYear().WithIntRange(1, 10000) }},
		{"year unsigned lower bound", func() { Int(16).Unsigned().AsYear().WithUnsignedRange(0, 9999) }},
		{"year unsigned upper bound", func() { Int(16).Unsigned().AsYear().WithUnsignedRange(1, 10000) }},
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
		{"measurement invalid unit", func() { Int(64).AsMeasurement(Unit(0)) }},
		{"duration on ordinary float", func() { Float(32).AsDuration(Second) }},
		{"duration invalid unit", func() { Int(64).AsDuration(Unit(0)) }},
		{"duration measurement-only unit", func() { Int(64).AsDuration(Kilogram) }},
		{"duplicate semantics", func() { String().AsEmail().AsPhone() }},
		{"invalid currency", func() { Decimal(10, 2).AsMoney().WithCurrency("usd") }},
		{"currency without money", func() { Decimal(10, 2).WithCurrency("EUR") }},
		{"country format without country", func() { String().CountryFormat() }},
		{"unit without measurement or duration", func() { Decimal(10, 2).Unit() }},
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

// Test_EqualSemantics tests semantics equality while ignoring kind and
// unrelated type properties.
func Test_EqualSemantics(t *testing.T) {

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
		{"measurement and duration with same unit", Int(64).AsMeasurement(Second), Int(64).AsDuration(Second), false},
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
		{
			"year lower bound clamped", Int(16).WithIntRange(-100, 2100).AsYear(),
			Int(16).AsYear().WithIntRange(1, 2100), true,
		},
		{
			"year upper bound clamped", Int(64).WithIntRange(1900, 20000).AsYear(),
			Int(64).AsYear().WithIntRange(1900, 9999), true,
		},
		{
			"unsigned year bounds clamped", Int(64).Unsigned().WithUnsignedRange(0, 20000).AsYear(),
			Int(64).Unsigned().AsYear(), true,
		},
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
		{"removed year kind", `{"kind":"year"}`, `unknown type kind "year"`},
		{"year on string", `{"kind":"string","semantics":"year"}`, "year semantics require an int type"},
		{
			"year outside range", `{"kind":"int","bitSize":16,"semantics":"year","minimum":10000}`,
			"year range must be within [1,9999]",
		},
		{
			"year lower bound", `{"kind":"int","bitSize":16,"semantics":"year","minimum":0}`,
			"year range must be within [1,9999]",
		},
		{
			"year upper bound", `{"kind":"int","bitSize":16,"semantics":"year","maximum":20000}`,
			"year range must be within [1,9999]",
		},
		{
			"unsigned year lower bound", `{"kind":"int","bitSize":16,"semantics":"year","unsigned":true,"minimum":0}`,
			"year range must be within [1,9999]",
		},
		{
			"unsigned year upper bound", `{"kind":"int","bitSize":16,"semantics":"year","unsigned":true,"maximum":20000}`,
			"year range must be within [1,9999]",
		},
		{
			"year with unit", `{"kind":"int","bitSize":16,"semantics":"year","unit":"year"}`,
			"unexpected 'unit' key for year semantics",
		},
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
			"invalid duration unit", `{"kind":"int","bitSize":64,"semantics":"duration","unit":"fortnight"}`,
			`invalid duration unit "fortnight"`,
		},
		{
			"measurement-only duration unit", `{"kind":"int","bitSize":64,"semantics":"duration","unit":"kg"}`,
			`invalid duration unit "kg"`,
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
		{
			"year", Int(16).AsYear(),
			`{"kind":"int","semantics":"year","bitSize":16}`,
		},
		{
			"unsigned year", Int(64).Unsigned().AsYear(),
			`{"kind":"int","semantics":"year","bitSize":64,"unsigned":true}`,
		},
		{
			"unsigned 32-bit year", Int(32).Unsigned().AsYear(),
			`{"kind":"int","semantics":"year","bitSize":32,"unsigned":true}`,
		},
		{
			"year minimum", Int(16).AsYear().WithIntRange(1900, 9999),
			`{"kind":"int","semantics":"year","bitSize":16,"minimum":1900}`,
		},
		{
			"year maximum", Int(64).AsYear().WithIntRange(1, 2100),
			`{"kind":"int","semantics":"year","bitSize":64,"maximum":2100}`,
		},
		{
			"unsigned year minimum", Int(64).Unsigned().AsYear().WithUnsignedRange(1900, 9999),
			`{"kind":"int","semantics":"year","bitSize":64,"unsigned":true,"minimum":1900}`,
		},
		{
			"unsigned year maximum", Int(32).Unsigned().AsYear().WithUnsignedRange(1, 2100),
			`{"kind":"int","semantics":"year","bitSize":32,"unsigned":true,"maximum":2100}`,
		},
		{
			"year range", Int(32).WithIntRange(1900, 2100).AsYear(),
			`{"kind":"int","semantics":"year","bitSize":32,"minimum":1900,"maximum":2100}`,
		},
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
			"measurement time unit", Float(64).Real().AsMeasurement(Second),
			`{"kind":"float","semantics":"measurement","unit":"second","bitSize":64,"real":true}`,
		},
		{
			"duration", Int(64).AsDuration(Millisecond),
			`{"kind":"int","semantics":"duration","unit":"millisecond","bitSize":64}`,
		},
		{
			"duration year", Float(64).Real().AsDuration(Year),
			`{"kind":"float","semantics":"duration","unit":"year","bitSize":64,"real":true}`,
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

// Test_TypeSemanticsJSONYearBounds checks normalization of year bounds.
func Test_TypeSemanticsJSONYearBounds(t *testing.T) {
	tests := []struct {
		data string
		want Type
	}{
		{`{"kind":"int","bitSize":16,"semantics":"year"}`, Int(16).AsYear()},
		{`{"kind":"int","bitSize":16,"semantics":"year","minimum":1,"maximum":9999}`, Int(16).AsYear()},
		{`{"kind":"int","bitSize":16,"semantics":"year","minimum":1900}`, Int(16).AsYear().WithIntRange(1900, 9999)},
		{`{"kind":"int","bitSize":16,"semantics":"year","maximum":2100}`, Int(16).AsYear().WithIntRange(1, 2100)},
		{`{"kind":"int","bitSize":8,"semantics":"year"}`, Int(8).AsYear()},
		{`{"kind":"int","bitSize":8,"unsigned":true,"semantics":"year"}`, Int(8).Unsigned().AsYear()},
		{`{"kind":"int","bitSize":64,"unsigned":true,"semantics":"year"}`, Int(64).Unsigned().AsYear()},
		{`{"kind":"int","bitSize":64,"semantics":"year","minimum":1,"maximum":9999}`, Int(64).AsYear()},
		{`{"kind":"int","bitSize":64,"unsigned":true,"semantics":"year","minimum":1,"maximum":9999}`, Int(64).Unsigned().AsYear()},
	}
	for _, test := range tests {
		t.Run(test.data, func(t *testing.T) {
			var got Type
			err := json.Unmarshal([]byte(test.data), &got)
			if err != nil {
				t.Fatalf("expected valid year bounds, got %v", err)
			}
			if !Equal(got, test.want) {
				t.Fatalf("expected %s, got %s", test.want, got)
			}
		})
	}
}

// Test_Unit tests unit names, lookup, and usability.
func Test_Unit(t *testing.T) {
	tests := []struct {
		unit     Unit
		name     string
		duration bool
	}{
		{Gram, "g", false},
		{Kilogram, "kg", false},
		{Millimeter, "mm", false},
		{Centimeter, "cm", false},
		{Meter, "m", false},
		{Kilometer, "km", false},
		{Milliliter, "mL", false},
		{Liter, "L", false},
		{Byte, "B", false},
		{Kilobyte, "kB", false},
		{Megabyte, "MB", false},
		{Gigabyte, "GB", false},
		{Celsius, "°C", false},
		{Fahrenheit, "°F", false},
		{Ounce, "oz", false},
		{Pound, "lb", false},
		{Inch, "in", false},
		{Foot, "ft", false},
		{Yard, "yd", false},
		{Mile, "mi", false},
		{Millisecond, "millisecond", true},
		{Second, "second", true},
		{Minute, "minute", true},
		{Hour, "hour", true},
		{Day, "day", true},
		{Week, "week", true},
		{Month, "month", true},
		{Year, "year", true},
	}
	for _, test := range tests {
		if got := test.unit.String(); got != test.name {
			t.Errorf("expected name %q, got %q", test.name, got)
		}
		got, ok := UnitByName(test.name)
		if !ok || got != test.unit {
			t.Errorf("expected unit %d and true, got %d and %t", test.unit, got, ok)
		}
		if !test.unit.IsMeasurementUnit() {
			t.Errorf("expected IsMeasurementUnit of %s to be true, got false", test.name)
		}
		if got := test.unit.IsDurationUnit(); got != test.duration {
			t.Errorf("expected IsDurationUnit of %s to be %t, got %t", test.name, test.duration, got)
		}
	}
	if got, ok := UnitByName("kilogram"); ok || got != Unit(0) {
		t.Fatalf("expected invalid unit and false, got %d and %t", got, ok)
	}
	for _, u := range []Unit{-1, 0, 29} {
		if m, d := u.IsMeasurementUnit(), u.IsDurationUnit(); m || d {
			t.Errorf("expected IsMeasurementUnit and IsDurationUnit of %d to be false, got %t and %t", u, m, d)
		}
	}
}
