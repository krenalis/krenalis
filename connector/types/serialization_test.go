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
			Expected: `{"name":"a","label":"","description":"","placeholder":null,` +
				`"type":{"name":"Text"},"nullable":false}`,
		},
		{
			Property: Property{Name: "a", Label: "a label", Type: Text()},
			Expected: `{"name":"a","label":"a label","description":"","placeholder":null,"type":{"name":"Text"},"nullable":false}`,
		},
		{
			Property: Property{Name: "a", Label: "a label",
				Description: "some description", Type: Text()},
			Expected: `{"name":"a","label":"a label","description":"some description",` +
				`"placeholder":null,"type":{"name":"Text"},"nullable":false}`,
		},
		{
			Property: Property{
				Name: "a", Label: "a label", Description: "some description",
				Placeholder: "<placeholder>", Type: Text()},
			Expected: `{"name":"a","label":"a label","description":"some description",` +
				`"placeholder":"<placeholder>","type":{"name":"Text"},"nullable":false}`,
		},
		{
			Property: Property{
				Name: "a", Label: "a label", Description: "some description",
				Placeholder: "<placeholder>", Role: DestinationRole, Type: Text()},
			Expected: `{"name":"a","label":"a label","description":"some description",` +
				`"placeholder":"<placeholder>","type":{"name":"Text"},"nullable":false}`,
		},
		{
			Property: Property{
				Name: "a", Label: "a label", Description: "some description",
				Placeholder: "<placeholder>", Role: DestinationRole, Type: Text(),
				Required: true},
			Expected: `{"name":"a","label":"a label","description":"some description",` +
				`"placeholder":"<placeholder>","type":{"name":"Text"},"required":true,` +
				`"nullable":false}`,
		},
		{
			Property: Property{
				Name: "a", Label: "a label", Description: "some description",
				Placeholder: "<placeholder>", Role: DestinationRole, Type: Text(),
				Required: true, Nullable: true},
			Expected: `{"name":"a","label":"a label","description":"some description",` +
				`"placeholder":"<placeholder>","type":{"name":"Text"},"required":true,` +
				`"nullable":true}`,
		},
		{
			Property: Property{
				Name: "a", Label: "a label", Description: "some description",
				Placeholder: "<placeholder>", Role: DestinationRole, Type: Text(),
				Required: true, Nullable: true, Flat: true},
			Expected: `{"name":"a","label":"a label","description":"some description",` +
				`"placeholder":"<placeholder>","type":{"name":"Text"},"required":true,` +
				`"nullable":true,"flat":true}`,
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
			JSON: `{"name":"a","label":"","description":"","placeholder":null,` +
				`"type":{"name":"Text"},"nullable":false}`,
			Property: Property{Name: "a", Type: Text()},
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
		JSON     string
		Property Property
	}{
		{
			`{"name":"Apple","label":"","description":"","placeholder":null,"type":{"name":"Text"},"nullable":false}`,
			Property{Name: "Apple", Type: Text()},
		},
		{
			`{"name":"Apple","label":"A label","description":"Some description...","placeholder":null,"type":{"name":"Text"},"nullable":false}`,
			Property{Name: "Apple", Label: "A label", Description: "Some description...", Type: Text()},
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			var p Property
			err := json.Unmarshal([]byte(test.JSON), &p)
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
			if test.JSON != string(got) {
				t.Fatalf("expected %q, got %q", test.JSON, string(got))
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
			Data: `{"name":"JSON","charLen":10000}`,
			Type: JSON().WithCharLen(10000),
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
			Data: `{"name":"Array","itemType":{"name":"Text"}}`,
			Type: Array(Text()),
		}, {
			Data: `{"name":"Array","itemType":{"name":"Int","bitSize":32}}`,
			Type: Array(Int(32)),
		}, {
			Data: `{"name":"Array","minItems":2,"maxItems":8,"uniqueItems":true,"itemType":{"name":"Decimal"}}`,
			Type: Array(Decimal(0, 0)).WithMinItems(2).WithMaxItems(8).WithUnique(),
		}, {
			Data: `{"name":"Object","properties":[{"name":"email","label":"","description":"","placeholder":null,"type":{"name":"Text"},"nullable":false},{"name":"size","label":"","description":"","placeholder":null,"type":{"name":"Decimal"},"nullable":false}]}`,
			Type: Object([]Property{{Name: "email", Type: Text()}, {Name: "size", Type: Decimal(0, 0)}}),
		}, {
			Data: `{"name":"Object","properties":[{"name":"email","label":"","description":"","placeholder":null,"type":{"name":"Text"},"nullable":true}]}`,
			Type: Object([]Property{{Name: "email", Type: Text(), Nullable: true}}),
		}, {
			Data: `{"name":"Object","properties":[{"name":"birthday","label":"","description":"","placeholder":null,"type":{"name":"Date"},"nullable":false}]}`,
			Type: Object([]Property{{Name: "birthday", Type: Date()}}),
		}, {
			Data: `{"name":"Object","properties":[{"name":"birthday","label":"","description":"","placeholder":"mm/dd/yyyy","type":{"name":"Date"},"nullable":false}]}`,
			Type: Object([]Property{{Name: "birthday", Placeholder: "mm/dd/yyyy", Type: Date()}}),
		}, {
			Data: `{"name":"Object","properties":[{"name":"values","label":"","description":"","placeholder":{"a":"1","b":"2"},"type":{"name":"Map","valueType":{"name":"Int","bitSize":32}},"nullable":false}]}`,
			Type: Object([]Property{{Name: "values", Placeholder: map[string]string{"a": "1", "b": "2"}, Type: Map(Int(32))}}),
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
