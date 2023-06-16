//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"testing"

	"github.com/shopspring/decimal"
)

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
			Data: `{"name":"Text","enum":["a","b"]}`,
			Type: Text().WithEnum([]string{"a", "b"}),
		}, {
			Data: `{"name":"Text","charLen":10000}`,
			Type: Text().WithCharLen(10000),
		}, {
			Data: `{"name":"JSON","charLen":10000}`,
			Type: JSON().WithCharLen(10000),
		}, {
			Data: `{"name":"Int8","minimum":10}`,
			Type: Int8().WithIntRange(10, MaxInt8),
		}, {
			Data: `{"name":"Float","minimum":-3.9936173,"maximum":8.00002312}`,
			Type: Float().WithFloatRange(-3.9936173, 8.00002312),
		}, {
			Data: `{"name":"Float32","minimum":3.99,"maximum":5.31}`,
			Type: Float32().WithFloatRange(3.99, 5.31),
		}, {
			Data: `{"name":"Float","real":true}`,
			Type: Float().AsReal(),
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
			Data: `{"name":"DateTime","layout":"2006-01-02T15:04"}`,
			Type: DateTime().WithLayout("2006-01-02T15:04"),
		}, {
			Data: `{"name":"Date","layout":"2006-01-02"}`,
			Type: Date().WithLayout("2006-01-02"),
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
			Data: `{"name":"Array","itemType":{"name":"Int"}}`,
			Type: Array(Int()),
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
			Data: `{"name":"Object","properties":[{"name":"values","label":"","description":"","placeholder":{"a":"1","b":"2"},"type":{"name":"Map","valueType":{"name":"Int"}},"nullable":false}]}`,
			Type: Object([]Property{{Name: "values", Placeholder: map[string]string{"a": "1", "b": "2"}, Type: Map(Int())}}),
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
