// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"testing"

	"github.com/krenalis/krenalis/tools/decimal"
)

func TestParseErrors(t *testing.T) {
	tests := []struct {
		data string
		err  string
	}{
		{"null", "invalid type syntax"},
		{"[]", "invalid type syntax"},
		{"{\"kind\":\"string\"}{", "invalid token { after top-level value"},
		{"{\"bitSize\":8}", "missing 'kind' key"},
		{"{\"kind\":\"custom\"}", `unknown type kind "custom"`},
		{"{\"kind\":\"array\",\"elementType\":{\"kind\":\"T\"}}", `unknown type kind "T"`},
		{"{\"kind\":\"map\",\"elementType\":{\"kind\":\"T\"}}", `unknown type kind "T"`},
		{"{\"kind\":\"int\",\"bitSize\":8,\"bitSize\":16}", "repeated 'bitSize' key"},
		{"{\"kind\":\"string\",\"pattern\":\"a\",\"values\":[\"b\"]}", "values cannot be provided if pattern is provided"},
		{"{\"kind\":\"string\",\"pattern\":\"a\",\"maxBytes\":1}", "max bytes cannot be provided if pattern is provided"},
		{"{\"kind\":\"string\",\"maxBytes\":1,\"pattern\":\"a\"}", "pattern cannot be provided if max bytes is provided"},
		{"{\"kind\":\"string\",\"pattern\":\"a\",\"maxLength\":1}", "max length cannot be provided if pattern is provided"},
		{"{\"kind\":\"string\",\"maxLength\":1,\"pattern\":\"a\"}", "pattern cannot be provided if max length is provided"},
		{"{\"kind\":\"string\",\"values\":[\"a\"],\"maxBytes\":1}", "max bytes cannot be provided if values are provided"},
		{"{\"kind\":\"string\",\"maxBytes\":1,\"values\":[\"a\"]}", "values cannot be provided if max bytes is provided"},
		{"{\"kind\":\"string\",\"values\":[\"a\"],\"maxLength\":1}", "max length cannot be provided if values are provided"},
		{"{\"kind\":\"string\",\"maxLength\":1,\"values\":[\"a\"]}", "values cannot be provided if max length is provided"},
		{"{\"kind\":\"float\",\"bitSize\":32,\"minimum\":1e400}", "invalid value for minimum"},
		{"{\"kind\":\"float\",\"bitSize\":32,\"maximum\":1e400}", "invalid value for maximum"},
		{"{\"kind\":\"float\",\"bitSize\":64,\"minimum\":1e400}", "invalid value for minimum"},
		{"{\"kind\":\"float\",\"bitSize\":64,\"maximum\":1e400}", "invalid value for maximum"},
		{"{\"kind\":\"float\",\"bitSize\":32,\"minimum\":1,\"maximum\":0}", "maximum cannot be less than minimum"},
		{"{\"kind\":\"float\",\"bitSize\":32,\"minimum\":1.00000001,\"maximum\":1}", "maximum cannot be less than minimum"},
		{"{\"kind\":\"float\",\"bitSize\":32,\"minimum\":1e100,\"maximum\":1e99}", "maximum cannot be less than minimum"},
		{"{\"kind\":\"float\",\"bitSize\":32,\"minimum\":-1e99,\"maximum\":-1e100}", "maximum cannot be less than minimum"},
		{"{\"kind\":\"float\",\"bitSize\":64,\"minimum\":1,\"maximum\":0}", "maximum cannot be less than minimum"},
	}
	for _, tc := range tests {
		_, err := Parse(tc.data)
		if err == nil || err.Error() != tc.err {
			t.Fatalf("%s: expected %q, got %q", tc.data, tc.err, err)
		}
	}
}

func TestPropertySerialization(t *testing.T) {
	tests := []struct {
		Property Property
		Expected string
		Err      string
	}{
		{
			Property: Property{},
			Err:      "missing property name",
		},
		{
			Property: Property{Name: "Qwerty"},
			Err:      "missing property type",
		},
		{
			Property: Property{Name: "a", Type: Parameter("custom")},
			Expected: `{"name":"a","type":{"kind":"custom"}}`,
		},
		{
			Property: Property{Name: "a", Type: String()},
			Expected: `{"name":"a","type":{"kind":"string"}}`,
		},
		{
			Property: Property{Name: "a", Type: String()},
			Expected: `{"name":"a","type":{"kind":"string"}}`,
		},
		{
			Property: Property{Name: "a", Type: String(), Description: "some description"},
			Expected: `{"name":"a","type":{"kind":"string"},"description":"some description"}`,
		},
		{
			Property: Property{Name: "first_name", Type: String(), DisplayName: "First name", Description: "some description"},
			Expected: `{"name":"first_name","type":{"kind":"string"},` +
				`"displayName":"First name","description":"some description"}`,
		},
		{
			Property: Property{Name: "a", Prefilled: "<prefilled>", Type: String(), Description: "some description"},
			Expected: `{"name":"a","prefilled":"<prefilled>",` +
				`"type":{"kind":"string"},"description":"some description"}`,
		},
		{
			Property: Property{Name: "a", Prefilled: "<prefilled>", Type: String(), Description: "some description"},
			Expected: `{"name":"a","prefilled":"<prefilled>",` +
				`"type":{"kind":"string"},"description":"some description"}`,
		},
		{
			Property: Property{Name: "a", Prefilled: "<prefilled>", Type: String(), CreateRequired: true, UpdateRequired: true, Description: "some description"},
			Expected: `{"name":"a","prefilled":"<prefilled>",` +
				`"type":{"kind":"string"},"createRequired":true,"updateRequired":true,"description":"some description"}`,
		},
		{
			Property: Property{Name: "a", Prefilled: "<prefilled>", Type: String(), CreateRequired: true, Nullable: true, Description: "some description"},
			Expected: `{"name":"a","prefilled":"<prefilled>",` +
				`"type":{"kind":"string"},"createRequired":true,` +
				`"nullable":true,"description":"some description"}`,
		},
		{
			Property: Property{Name: "a", Prefilled: "<prefilled>", Type: String(), UpdateRequired: true, ReadOptional: true, Nullable: true, Description: "some description"},
			Expected: `{"name":"a","prefilled":"<prefilled>",` +
				`"type":{"kind":"string"},"updateRequired":true,"readOptional":true,` +
				`"nullable":true,"description":"some description"}`,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, err := test.Property.MarshalJSON()
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if test.Err != gotErr {
				t.Fatalf("expected error %q, got %q", test.Err, gotErr)
			}
			if test.Expected != string(got) {
				t.Fatalf("expected %q, got %q", test.Expected, string(got))
			}
		})
	}
}

func TestPropertyDeserialization(t *testing.T) {
	tests := []struct {
		JSON     string
		Property Property
		Err      string
	}{
		{
			JSON: `{}`,
			Err:  "missing property name",
		},
		{
			JSON: `{"name":"a"}`,
			Err:  "missing property type",
		},
		{
			JSON: `{"Name":"a"}`,
			Err:  "unknown property 'Name'",
		},
		{
			JSON: `2`,
			Err:  "invalid property syntax",
		},
		{
			JSON: `[]`,
			Err:  "invalid property syntax",
		},
		{
			JSON: ``,
			Err:  "unexpected end of JSON input",
		},
		{
			JSON: `[`,
			Err:  "unexpected end of JSON input",
		},
		{
			JSON:     `{"name":"a","description":"","type":{"kind":"string"}}`,
			Property: Property{Name: "a", Type: String()},
		},
		{
			JSON:     `{"name":"a","description":"","type":{"kind":"int","bitSize":32}}`,
			Property: Property{Name: "a", Type: Int(32)},
		},
		{
			JSON:     `{"name":"first_name","displayName":"First name","description":"","type":{"kind":"string"}}`,
			Property: Property{Name: "first_name", Type: String(), DisplayName: "First name"},
		},
		{
			JSON: `{"name":"a","displayName":false,"type":{"kind":"string"}}`,
			Err:  "unexpected value for property display name",
		},
		{
			JSON: `{"name":"a","displayName":"A","displayName":"B","type":{"kind":"string"}}`,
			Err:  "repeated 'displayName' key",
		},
		{
			JSON: `{{`,
			Err:  "invalid character '{' looking for beginning of object key string",
		},
		{
			JSON: `{"name":"a","type":{"kind":"custom"}}`,
			Err:  `unknown type kind "custom"`,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			var p Property
			err := json.Unmarshal([]byte(test.JSON), &p)
			var gotErr string
			if err != nil {
				gotErr = err.Error()
			}
			if test.Err != gotErr {
				t.Fatalf("expected error %q, got %q", test.Err, gotErr)
			}
			if err := sameProperty(test.Property, p); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPropertySerializationDeserialization(t *testing.T) {
	tests := []struct {
		InJSON   string
		Property Property
		OutJSON  string
	}{
		{
			`{"name":"Apple","type":{"kind":"string"},"description":""}`,
			Property{Name: "Apple", Type: String()},
			`{"name":"Apple","type":{"kind":"string"}}`,
		},
		{
			`{"name":"Apple","type":{"kind":"string","values":["g","c"]},"description":"Some description..."}`,
			Property{Name: "Apple", Type: String().WithValues("g", "c"), Description: "Some description..."},
			`{"name":"Apple","type":{"kind":"string","values":["g","c"]},"description":"Some description..."}`,
		},
		{
			`{"name":"first_name","type":{"kind":"string"},"displayName":"First name","description":"Given name"}`,
			Property{Name: "first_name", Type: String(), DisplayName: "First name", Description: "Given name"},
			`{"name":"first_name","type":{"kind":"string"},"displayName":"First name","description":"Given name"}`,
		},
		{
			`{"name":"amount","type":{"kind":"decimal","currency":"EUR","semantics":"money",` +
				`"precision":10,"scale":2},"description":"Amount"}`,
			Property{
				Name: "amount", Type: Decimal(10, 2).AsMoney().WithCurrency("EUR"), Description: "Amount",
			},
			`{"name":"amount","type":{"kind":"decimal","semantics":"money","currency":"EUR",` +
				`"precision":10,"scale":2},"description":"Amount"}`,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			var p Property
			err := json.Unmarshal([]byte(test.InJSON), &p)
			if err != nil {
				t.Fatalf("cannot unmarshal property: %s", err)
			}
			if err := sameProperty(test.Property, p); err != nil {
				t.Fatal(err)
			}
			got, err := p.MarshalJSON()
			if err != nil {
				t.Fatalf("cannot marshal property: %s", err)
			}
			if test.OutJSON != string(got) {
				t.Fatalf("expected %q, got %q", test.OutJSON, string(got))
			}
		})
	}
}

func TestTypeSerialization(t *testing.T) {

	type testCase struct {
		Data      string
		ParseData string
		Type      Type
	}
	tests := []testCase{
		{
			Data: `{"kind":"string"}`,
			Type: String(),
		}, {
			Data: `{"kind":"string","maxLength":10}`,
			Type: String().WithMaxLength(10),
		}, {
			Data: `{"kind":"string","maxBytes":24}`,
			Type: String().WithMaxBytes(24),
		}, {
			Data: `{"kind":"string","maxBytes":80,"maxLength":100}`,
			Type: String().WithMaxBytes(80).WithMaxLength(100),
		}, {
			Data: `{"kind":"string","values":["a","b"]}`,
			Type: String().WithValues("a", "b"),
		}, {
			Data: `{"kind":"string","maxLength":10000}`,
			Type: String().WithMaxLength(10000),
		}, {
			Data: `{"kind":"string","pattern":"\\d+$"}`,
			Type: String().WithPattern(regexp.MustCompile(`\d+$`)),
		}, {
			Data: `{"kind":"int","bitSize":8,"minimum":10}`,
			Type: Int(8).WithIntRange(10, MaxInt8),
		}, {
			Data:      `{"kind":"int","bitSize":8}`,
			ParseData: `{"kind":"int","bitSize":8,"minimum":-128}`,
			Type:      Int(8),
		}, {
			Data:      `{"kind":"int","bitSize":16,"maximum":-1}`,
			ParseData: `{"kind":"int","bitSize":16,"minimum":-32768,"maximum":-1}`,
			Type:      Int(16).WithIntRange(MinInt16, -1),
		}, {
			Data:      `{"kind":"int","bitSize":24}`,
			ParseData: `{"kind":"int","bitSize":24,"maximum":8388607}`,
			Type:      Int(24),
		}, {
			Data:      `{"kind":"int","bitSize":8,"unsigned":true,"minimum":1}`,
			ParseData: `{"kind":"int","bitSize":8,"unsigned":true,"minimum":1,"maximum":255}`,
			Type:      Int(8).Unsigned().WithUnsignedRange(1, MaxUint8),
		}, {
			Data:      `{"kind":"int","bitSize":32,"unsigned":true}`,
			ParseData: `{"kind":"int","bitSize":32,"unsigned":true,"maximum":4294967295}`,
			Type:      Int(32).Unsigned(),
		}, {
			Data: `{"kind":"int","bitSize":32,"unsigned":true,"minimum":20,"maximum":30}`,
			Type: Int(32).Unsigned().WithUnsignedRange(20, 30),
		}, {
			Data: `{"kind":"int","bitSize":32,"unsigned":true,"minimum":3000000000,"maximum":4000000000}`,
			Type: Int(32).Unsigned().WithUnsignedRange(3000000000, 4000000000),
		}, {
			Data:      `{"kind":"int","semantics":"year","bitSize":16}`,
			ParseData: `{"kind":"int","bitSize":16,"semantics":"year","minimum":1,"maximum":9999}`,
			Type:      Int(16).AsYear(),
		}, {
			Data:      `{"kind":"int","semantics":"year","bitSize":64,"unsigned":true}`,
			ParseData: `{"kind":"int","bitSize":64,"unsigned":true,"semantics":"year","minimum":1,"maximum":9999}`,
			Type:      Int(64).Unsigned().AsYear(),
		}, {
			Data: `{"kind":"float","bitSize":64,"minimum":-3.9936173,"maximum":8.00002312}`,
			Type: Float(64).WithFloatRange(-3.9936173, 8.00002312),
		}, {
			Data: `{"kind":"float","bitSize":32,"minimum":3.99,"maximum":5.31}`,
			Type: Float(32).WithFloatRange(3.99, 5.31),
		}, {
			Data:      `{"kind":"float","bitSize":32,"minimum":0,"maximum":0}`,
			ParseData: `{"kind":"float","bitSize":32,"minimum":-0,"maximum":-0}`,
			Type:      Float(32).WithFloatRange(0, 0),
		}, {
			Data:      `{"kind":"float","bitSize":64,"minimum":0,"maximum":0}`,
			ParseData: `{"kind":"float","bitSize":64,"minimum":-0,"maximum":-0}`,
			Type:      Float(64).WithFloatRange(0, 0),
		}, {
			Data:      `{"kind":"float","bitSize":32,"minimum":0,"maximum":0}`,
			ParseData: `{"kind":"float","bitSize":32,"minimum":-1e-100,"maximum":-1e-100}`,
			Type:      Float(32).WithFloatRange(0, 0),
		}, {
			Data:      `{"kind":"float","bitSize":32,"minimum":1,"maximum":1}`,
			ParseData: `{"kind":"float","bitSize":32,"minimum":1,"maximum":1.00000001}`,
			Type:      Float(32).WithFloatRange(1, 1.00000001),
		}, {
			Data:      `{"kind":"float","bitSize":32,"minimum":1.0000001,"maximum":2}`,
			ParseData: `{"kind":"float","bitSize":32,"minimum":1.000000059604644775390626,"maximum":2}`,
			Type:      Float(32).WithFloatRange(float64(math.Nextafter32(1, 2)), 2),
		}, {
			Data:      `{"kind":"float","bitSize":64,"minimum":1.0000000596046448,"maximum":2}`,
			ParseData: `{"kind":"float","bitSize":64,"minimum":1.000000059604644775390626,"maximum":2}`,
			Type:      Float(64).WithFloatRange(1.0000000596046448, 2),
		}, {
			Data: `{"kind":"decimal","precision":1}`,
			Type: Decimal(1, 0),
		}, {
			Data: `{"kind":"decimal","minimum":-3.9936173,"maximum":8.00002312,"precision":9,"scale":8}`,
			Type: Decimal(9, 8).WithDecimalRange(
				decimal.MustParse("-3.9936173"),
				decimal.MustParse("8.00002312"),
			),
		}, {
			Data: `{"kind":"decimal","precision":10}`,
			Type: Decimal(10, 0),
		}, {
			Data: `{"kind":"decimal","precision":10,"scale":8}`,
			Type: Decimal(10, 8),
		}, {
			Data: `{"kind":"datetime"}`,
			Type: DateTime(),
		}, {
			Data: `{"kind":"date"}`,
			Type: Date(),
		}, {
			Data: `{"kind":"string","values":["b","a","c"]}`,
			Type: String().WithValues("b", "a", "c"),
		}, {
			Data: `{"kind":"array","elementType":{"kind":"string"}}`,
			Type: Array(String()),
		}, {
			Data: `{"kind":"array","elementType":{"kind":"int","bitSize":32}}`,
			Type: Array(Int(32)),
		}, {
			Data: `{"kind":"array","minElements":2,"maxElements":8,"uniqueElements":true,"elementType":{"kind":"decimal","precision":1}}`,
			Type: Array(Decimal(1, 0)).WithMinElements(2).WithMaxElements(8).WithUnique(),
		}, {
			Data: `{"kind":"object","properties":[{"name":"email","type":{"kind":"string"}},{"name":"size","type":{"kind":"decimal","precision":1}}]}`,
			Type: Object([]Property{{Name: "email", Type: String()}, {Name: "size", Type: Decimal(1, 0)}}),
		}, {
			Data: `{"kind":"object","properties":[{"name":"email","type":{"kind":"string"},"nullable":true}]}`,
			Type: Object([]Property{{Name: "email", Type: String(), Nullable: true}}),
		}, {
			Data: `{"kind":"object","properties":[{"name":"first_name","type":{"kind":"string"},"displayName":"First name"}]}`,
			Type: Object([]Property{{Name: "first_name", Type: String(), DisplayName: "First name"}}),
		}, {
			Data: `{"kind":"object","properties":[{"name":"birthday","type":{"kind":"date"}}]}`,
			Type: Object([]Property{{Name: "birthday", Type: Date()}}),
		}, {
			Data: `{"kind":"object","properties":[{"name":"birthday","prefilled":"mm/dd/yyyy","type":{"kind":"date"}}]}`,
			Type: Object([]Property{{Name: "birthday", Prefilled: "mm/dd/yyyy", Type: Date()}}),
		},
	}

	for _, size := range []int{32, 64} {
		limit := math.MaxFloat64
		if size == 32 {
			limit = math.MaxFloat32
		}
		for _, test := range []struct {
			min, max float64
			bounds   string
		}{
			{-1, 1, `,"minimum":-1,"maximum":1`},
			{0, 0, `,"minimum":0,"maximum":0`},
			{0, math.Inf(1), `,"minimum":0`},
			{math.Inf(-1), 0, `,"maximum":0`},
			{math.Inf(-1), math.Inf(1), ""},
			{0, limit, `,"minimum":0`},
			{-limit, 0, `,"maximum":0`},
			{-limit, limit, ""},
			{0, math.MaxFloat64, `,"minimum":0`},
			{-math.MaxFloat64, 0, `,"maximum":0`},
			{-math.MaxFloat64, math.MaxFloat64, ""},
			{math.MaxFloat64, math.MaxFloat64, `,"minimum":` + strconv.FormatFloat(limit, 'f', -1, size)},
			{-math.MaxFloat64, -math.MaxFloat64, `,"maximum":` + strconv.FormatFloat(-limit, 'f', -1, size)},
		} {
			var parseBounds string
			if !math.IsInf(test.min, -1) {
				parseBounds += fmt.Sprintf(`,"minimum":%g`, test.min)
			}
			if !math.IsInf(test.max, 1) {
				parseBounds += fmt.Sprintf(`,"maximum":%g`, test.max)
			}
			bounded := Float(size).WithFloatRange(test.min, test.max)
			data, err := bounded.MarshalJSON()
			if err != nil {
				t.Fatalf("expected valid JSON, got %v", err)
			}
			tests = append(tests,
				// Check normalization and the round trip for non-real floats.
				testCase{Data: string(data), Type: bounded},
				testCase{
					Data:      string(data),
					ParseData: fmt.Sprintf(`{"kind":"float","bitSize":%d%s}`, size, parseBounds),
					Type:      bounded,
				},
				testCase{
					Data:      fmt.Sprintf(`{"kind":"float","bitSize":%d,"real":true%s}`, size, test.bounds),
					ParseData: fmt.Sprintf(`{"kind":"float","bitSize":%d,"real":true%s}`, size, parseBounds),
					Type:      Float(size).Real().WithFloatRange(test.min, test.max),
				},
			)
		}
	}

	for _, test := range tests {
		data := test.Data
		if test.ParseData != "" {
			data = test.ParseData
		}
		got, err := Parse(data)
		if err != nil {
			t.Errorf("cannot unmarshal type %q: %s", data, err)
			continue
		}
		if err = sameType(test.Type, got); err != nil {
			t.Errorf("%s: %s", data, err)
			continue
		}
		b, err := test.Type.MarshalJSON()
		if err != nil {
			t.Errorf("%s: %s", test.Data, err)
			continue
		}
		if data := string(b); test.Data != data {
			t.Errorf("\nexpected\t%s\ngot\t\t\t%s", test.Data, data)
		}
	}

}

func TestGenericTypeSerialization(t *testing.T) {
	tests := []struct {
		Type Type
		Data string
	}{
		{
			Type: Parameter("T"),
			Data: `{"kind":"T"}`,
		}, {
			Type: Array(Parameter("T")),
			Data: `{"kind":"array","elementType":{"kind":"T"}}`,
		}, {
			Type: Map(Parameter("T")),
			Data: `{"kind":"map","elementType":{"kind":"T"}}`,
		}, {
			Type: Object([]Property{{Name: "items", Type: Array(Parameter("T"))}}),
			Data: `{"kind":"object","properties":[{"name":"items","type":{"kind":"array","elementType":{"kind":"T"}}}]}`,
		},
	}
	for _, test := range tests {
		b, err := test.Type.MarshalJSON()
		if err != nil {
			t.Errorf("%s: %s", test.Data, err)
			continue
		}
		if data := string(b); test.Data != data {
			t.Errorf("\nexpected\t%s\ngot\t\t\t%s", test.Data, data)
		}
	}
}
