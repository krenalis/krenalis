// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/meergo/meergo/tools/decimal"
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
		{"{\"kind\":\"int\",\"bitSize\":8,\"bitSize\":16}", "repeated 'bitSize' key"},
		{"{\"kind\":\"string\",\"regexp\":\"a\",\"values\":[\"b\"]}", "values cannot be provided if regular expression is provided"},
	}
	for _, tc := range tests {
		_, err := Parse(tc.data)
		if err == nil || err.Error() != tc.err {
			t.Fatalf("%s: expected %q, got %v", tc.data, tc.err, err)
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
			Property: Property{Name: "a", Type: String()},
			Expected: `{"name":"a","type":{"kind":"string"},"description":""}`,
		},
		{
			Property: Property{Name: "a", Type: String()},
			Expected: `{"name":"a","type":{"kind":"string"},"description":""}`,
		},
		{
			Property: Property{Name: "a", Type: String(), Description: "some description"},
			Expected: `{"name":"a","type":{"kind":"string"},"description":"some description"}`,
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
			JSON: `{{`,
			Err:  "invalid character '{' looking for beginning of object key string",
		},
		{
			JSON:     `{"name":"a","type":{"kind":"custom"}}`,
			Property: Property{Name: "a", Type: Parameter("custom")},
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
			`{"name":"Apple","type":{"kind":"string"},"description":""}`,
		},
		{
			`{"name":"Apple","type":{"kind":"string","values":["g","c"]},"description":"Some description..."}`,
			Property{Name: "Apple", Type: String().WithValues("g", "c"), Description: "Some description..."},
			`{"name":"Apple","type":{"kind":"string","values":["g","c"]},"description":"Some description..."}`,
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

	tests := []struct {
		Data string
		Type Type
	}{
		{
			Data: `{"kind":"string"}`,
			Type: String(),
		}, {
			Data: `{"kind":"string","maxLength":10}`,
			Type: String().WithMaxLength(10),
		}, {
			Data: `{"kind":"string","maxByteLength":24}`,
			Type: String().WithMaxByteLength(24),
		}, {
			Data: `{"kind":"string","maxByteLength":80,"maxLength":100}`,
			Type: String().WithMaxByteLength(80).WithMaxLength(100),
		}, {
			Data: `{"kind":"string","values":["a","b"]}`,
			Type: String().WithValues("a", "b"),
		}, {
			Data: `{"kind":"string","maxLength":10000}`,
			Type: String().WithMaxLength(10000),
		}, {
			Data: `{"kind":"string","regexp":"\\d+$"}`,
			Type: String().WithRegexp(regexp.MustCompile(`\d+$`)),
		}, {
			Data: `{"kind":"int","bitSize":8,"minimum":10}`,
			Type: Int(8).WithIntRange(10, MaxInt8),
		}, {
			Data: `{"kind":"float","bitSize":64,"minimum":-3.9936173,"maximum":8.00002312}`,
			Type: Float(64).WithFloatRange(-3.9936173, 8.00002312),
		}, {
			Data: `{"kind":"float","bitSize":32,"minimum":3.99,"maximum":5.31}`,
			Type: Float(32).WithFloatRange(3.99, 5.31),
		}, {
			Data: `{"kind":"float","bitSize":64,"real":true}`,
			Type: Float(64).AsReal(),
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
			Data: `{"kind":"object","properties":[{"name":"email","type":{"kind":"string"},"description":""},{"name":"size","type":{"kind":"decimal","precision":1},"description":""}]}`,
			Type: Object([]Property{{Name: "email", Type: String()}, {Name: "size", Type: Decimal(1, 0)}}),
		}, {
			Data: `{"kind":"object","properties":[{"name":"email","type":{"kind":"string"},"nullable":true,"description":""}]}`,
			Type: Object([]Property{{Name: "email", Type: String(), Nullable: true}}),
		}, {
			Data: `{"kind":"object","properties":[{"name":"birthday","type":{"kind":"date"},"description":""}]}`,
			Type: Object([]Property{{Name: "birthday", Type: Date()}}),
		}, {
			Data: `{"kind":"object","properties":[{"name":"birthday","prefilled":"mm/dd/yyyy","type":{"kind":"date"},"description":""}]}`,
			Type: Object([]Property{{Name: "birthday", Prefilled: "mm/dd/yyyy", Type: Date()}}),
		},
	}
	for _, test := range tests {
		got, err := Parse(test.Data)
		if err != nil {
			t.Errorf("cannot unmarshal type %q: %s", test.Data, err)
			continue
		}
		if err = sameType(test.Type, got); err != nil {
			t.Errorf("%s: %s", test.Data, err)
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
