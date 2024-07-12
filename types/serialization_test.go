//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

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
			Expected: `{"name":"a","label":"","type":{"name":"Text"},"note":""}`,
		},
		{
			Property: Property{Name: "a", Label: "a label", Type: Text()},
			Expected: `{"name":"a","label":"a label","type":{"name":"Text"},"note":""}`,
		},
		{
			Property: Property{Name: "a", Label: "a label",
				Type: Text(), Note: "some note"},
			Expected: `{"name":"a","label":"a label","type":{"name":"Text"},"note":"some note"}`,
		},
		{
			Property: Property{
				Name: "a", Label: "a label", Placeholder: "<placeholder>",
				Type: Text(), Note: "some note"},
			Expected: `{"name":"a","label":"a label","placeholder":"<placeholder>",` +
				`"type":{"name":"Text"},"note":"some note"}`,
		},
		{
			Property: Property{
				Name: "a", Label: "a label", Placeholder: "<placeholder>", Role: DestinationRole,
				Type: Text(), Note: "some note"},
			Expected: `{"name":"a","label":"a label","placeholder":"<placeholder>",` +
				`"type":{"name":"Text"},"note":"some note"}`,
		},
		{
			Property: Property{
				Name: "a", Label: "a label", Placeholder: "<placeholder>", Role: DestinationRole,
				Type: Text(), Required: true, Note: "some note"},
			Expected: `{"name":"a","label":"a label","placeholder":"<placeholder>",` +
				`"type":{"name":"Text"},"required":true,"note":"some note"}`,
		},
		{
			Property: Property{
				Name: "a", Label: "a label", Placeholder: "<placeholder>", Role: DestinationRole,
				Type: Text(), Required: true, Nullable: true, Note: "some note"},
			Expected: `{"name":"a","label":"a label","placeholder":"<placeholder>",` +
				`"type":{"name":"Text"},"required":true,` +
				`"nullable":true,"note":"some note"}`,
		},
		{
			Property: Property{
				Name: "a", Label: "a label", Placeholder: "<placeholder>", Role: DestinationRole,
				Type: Text(), Required: true, Nullable: true, Note: "some note"},
			Expected: `{"name":"a","label":"a label","placeholder":"<placeholder>",` +
				`"type":{"name":"Text"},"required":true,` +
				`"nullable":true,"note":"some note"}`,
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
				t.Fatalf("expecting error %q, got %q", test.Err, gotErr)
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
			Err:  "json: missing property name",
		},
		{
			JSON: `{"name":"a"}`,
			Err:  "json: missing property type",
		},
		{
			JSON: `{"Name":"a"}`,
			Err:  "json: unknown property 'Name'",
		},
		{
			JSON: `2`,
			Err:  "json: invalid property syntax",
		},
		{
			JSON: `[]`,
			Err:  "json: invalid property syntax",
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
			JSON:     `{"name":"a","label":"","note":"","type":{"name":"Text"}}`,
			Property: Property{Name: "a", Type: Text()},
		},
		{
			JSON:     `{"name":"a","label":"","note":"","type":{"name":"Int","bitSize":32}}`,
			Property: Property{Name: "a", Type: Int(32)},
		},
		{
			JSON: `{{`,
			Err:  "invalid character '{' looking for beginning of object key string",
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
				t.Fatalf("expecting error %q, got %q", test.Err, gotErr)
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
			`{"name":"Apple","label":"","type":{"name":"Text"},"note":""}`,
			Property{Name: "Apple", Type: Text()},
			`{"name":"Apple","label":"","type":{"name":"Text"},"note":""}`,
		},
		{
			`{"name":"Apple","label":"A label","type":{"name":"Text","values":["g","c"]},"note":"Some note..."}`,
			Property{Name: "Apple", Label: "A label", Type: Text().WithValues("c", "g"), Note: "Some note..."},
			`{"name":"Apple","label":"A label","type":{"name":"Text","values":["c","g"]},"note":"Some note..."}`,
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
			Data: `{"name":"Text"}`,
			Type: Text(),
		}, {
			Data: `{"name":"Text","charLen":10}`,
			Type: Text().WithCharLen(10),
		}, {
			Data: `{"name":"Text","byteLen":24}`,
			Type: Text().WithByteLen(24),
		}, {
			Data: `{"name":"Text","byteLen":80,"charLen":100}`,
			Type: Text().WithByteLen(80).WithCharLen(100),
		}, {
			Data: `{"name":"Text","values":["a","b"]}`,
			Type: Text().WithValues("a", "b"),
		}, {
			Data: `{"name":"Text","charLen":10000}`,
			Type: Text().WithCharLen(10000),
		}, {
			Data: `{"name":"Int","bitSize":8,"minimum":10}`,
			Type: Int(8).WithIntRange(10, MaxInt8),
		}, {
			Data: `{"name":"Float","bitSize":64,"minimum":-3.9936173,"maximum":8.00002312}`,
			Type: Float(64).WithFloatRange(-3.9936173, 8.00002312),
		}, {
			Data: `{"name":"Float","bitSize":32,"minimum":3.99,"maximum":5.31}`,
			Type: Float(32).WithFloatRange(3.99, 5.31),
		}, {
			Data: `{"name":"Float","bitSize":64,"real":true}`,
			Type: Float(64).AsReal(),
		}, {
			Data: `{"name":"Decimal"}`,
			Type: Decimal(0, 0),
		}, {
			Data: `{"name":"Decimal","minimum":-3.9936173,"maximum":8.00002312}`,
			Type: Decimal(0, 0).WithDecimalRange(
				decimal.RequireFromString("-3.9936173"),
				decimal.RequireFromString("8.00002312"),
			),
		}, {
			Data: `{"name":"Decimal","precision":10}`,
			Type: Decimal(10, 0),
		}, {
			Data: `{"name":"Decimal","precision":10,"scale":8}`,
			Type: Decimal(10, 8),
		}, {
			Data: `{"name":"DateTime"}`,
			Type: DateTime(),
		}, {
			Data: `{"name":"Date"}`,
			Type: Date(),
		}, {
			Data: `{"name":"Text","values":["a","b","c"]}`,
			Type: Text().WithValues("b", "a", "c"),
		}, {
			Data: `{"name":"Array","elementType":{"name":"Text"}}`,
			Type: Array(Text()),
		}, {
			Data: `{"name":"Array","elementType":{"name":"Int","bitSize":32}}`,
			Type: Array(Int(32)),
		}, {
			Data: `{"name":"Array","minElements":2,"maxElements":8,"uniqueElements":true,"elementType":{"name":"Decimal"}}`,
			Type: Array(Decimal(0, 0)).WithMinElements(2).WithMaxElements(8).WithUnique(),
		}, {
			Data: `{"name":"Object","properties":[{"name":"email","label":"","type":{"name":"Text"},"note":""},{"name":"size","label":"","type":{"name":"Decimal"},"note":""}]}`,
			Type: Object([]Property{{Name: "email", Type: Text()}, {Name: "size", Type: Decimal(0, 0)}}),
		}, {
			Data: `{"name":"Object","properties":[{"name":"email","label":"","type":{"name":"Text"},"nullable":true,"note":""}]}`,
			Type: Object([]Property{{Name: "email", Type: Text(), Nullable: true}}),
		}, {
			Data: `{"name":"Object","properties":[{"name":"birthday","label":"","type":{"name":"Date"},"note":""}]}`,
			Type: Object([]Property{{Name: "birthday", Type: Date()}}),
		}, {
			Data: `{"name":"Object","properties":[{"name":"birthday","label":"","placeholder":"mm/dd/yyyy","type":{"name":"Date"},"note":""}]}`,
			Type: Object([]Property{{Name: "birthday", Placeholder: "mm/dd/yyyy", Type: Date()}}),
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
			t.Errorf("\nexpecting\t%s\ngot\t\t\t%s", test.Data, data)
		}
	}

}
