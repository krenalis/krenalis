//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/meergo/meergo/decimal"
)

func TestParseErrors(t *testing.T) {
	tests := []struct {
		data string
		err  string
	}{
		{"null", "invalid type syntax"},
		{"[]", "invalid type syntax"},
		{"{\"kind\":\"text\"}{", "invalid token { after top-level value"},
		{"{\"bitSize\":8}", "missing 'kind' key"},
		{"{\"kind\":\"int\",\"bitSize\":8,\"bitSize\":16}", "repeated 'bitSize' key"},
		{"{\"kind\":\"text\",\"regexp\":\"a\",\"values\":[\"b\"]}", "values cannot be provided if regular expression is provided"},
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
			Property: Property{Name: "a", Type: Text()},
			Expected: `{"name":"a","type":{"kind":"text"},"description":""}`,
		},
		{
			Property: Property{Name: "a", Type: Text()},
			Expected: `{"name":"a","type":{"kind":"text"},"description":""}`,
		},
		{
			Property: Property{Name: "a", Type: Text(), Description: "some description"},
			Expected: `{"name":"a","type":{"kind":"text"},"description":"some description"}`,
		},
		{
			Property: Property{Name: "a", Prefilled: "<prefilled>", Type: Text(), Description: "some description"},
			Expected: `{"name":"a","prefilled":"<prefilled>",` +
				`"type":{"kind":"text"},"description":"some description"}`,
		},
		{
			Property: Property{Name: "a", Prefilled: "<prefilled>", Type: Text(), Description: "some description"},
			Expected: `{"name":"a","prefilled":"<prefilled>",` +
				`"type":{"kind":"text"},"description":"some description"}`,
		},
		{
			Property: Property{Name: "a", Prefilled: "<prefilled>", Type: Text(), CreateRequired: true, UpdateRequired: true, Description: "some description"},
			Expected: `{"name":"a","prefilled":"<prefilled>",` +
				`"type":{"kind":"text"},"createRequired":true,"updateRequired":true,"description":"some description"}`,
		},
		{
			Property: Property{Name: "a", Prefilled: "<prefilled>", Type: Text(), CreateRequired: true, Nullable: true, Description: "some description"},
			Expected: `{"name":"a","prefilled":"<prefilled>",` +
				`"type":{"kind":"text"},"createRequired":true,` +
				`"nullable":true,"description":"some description"}`,
		},
		{
			Property: Property{Name: "a", Prefilled: "<prefilled>", Type: Text(), UpdateRequired: true, ReadOptional: true, Nullable: true, Description: "some description"},
			Expected: `{"name":"a","prefilled":"<prefilled>",` +
				`"type":{"kind":"text"},"updateRequired":true,"readOptional":true,` +
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
			JSON:     `{"name":"a","description":"","type":{"kind":"text"}}`,
			Property: Property{Name: "a", Type: Text()},
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
			`{"name":"Apple","type":{"kind":"text"},"description":""}`,
			Property{Name: "Apple", Type: Text()},
			`{"name":"Apple","type":{"kind":"text"},"description":""}`,
		},
		{
			`{"name":"Apple","type":{"kind":"text","values":["g","c"]},"description":"Some description..."}`,
			Property{Name: "Apple", Type: Text().WithValues("g", "c"), Description: "Some description..."},
			`{"name":"Apple","type":{"kind":"text","values":["g","c"]},"description":"Some description..."}`,
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
			Data: `{"kind":"text"}`,
			Type: Text(),
		}, {
			Data: `{"kind":"text","charLen":10}`,
			Type: Text().WithCharLen(10),
		}, {
			Data: `{"kind":"text","byteLen":24}`,
			Type: Text().WithByteLen(24),
		}, {
			Data: `{"kind":"text","byteLen":80,"charLen":100}`,
			Type: Text().WithByteLen(80).WithCharLen(100),
		}, {
			Data: `{"kind":"text","values":["a","b"]}`,
			Type: Text().WithValues("a", "b"),
		}, {
			Data: `{"kind":"text","charLen":10000}`,
			Type: Text().WithCharLen(10000),
		}, {
			Data: `{"kind":"text","regexp":"\\d+$"}`,
			Type: Text().WithRegexp(regexp.MustCompile(`\d+$`)),
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
			Data: `{"kind":"text","values":["b","a","c"]}`,
			Type: Text().WithValues("b", "a", "c"),
		}, {
			Data: `{"kind":"array","elementType":{"kind":"text"}}`,
			Type: Array(Text()),
		}, {
			Data: `{"kind":"array","elementType":{"kind":"int","bitSize":32}}`,
			Type: Array(Int(32)),
		}, {
			Data: `{"kind":"array","minElements":2,"maxElements":8,"uniqueElements":true,"elementType":{"kind":"decimal","precision":1}}`,
			Type: Array(Decimal(1, 0)).WithMinElements(2).WithMaxElements(8).WithUnique(),
		}, {
			Data: `{"kind":"object","properties":[{"name":"email","type":{"kind":"text"},"description":""},{"name":"size","type":{"kind":"decimal","precision":1},"description":""}]}`,
			Type: Object([]Property{{Name: "email", Type: Text()}, {Name: "size", Type: Decimal(1, 0)}}),
		}, {
			Data: `{"kind":"object","properties":[{"name":"email","type":{"kind":"text"},"nullable":true,"description":""}]}`,
			Type: Object([]Property{{Name: "email", Type: Text(), Nullable: true}}),
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
