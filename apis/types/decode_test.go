//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestDecode(t *testing.T) {

	tests := []struct {
		Data    string
		Type    Type
		Resolve Resolver
		Value   any
	}{
		{
			Data:  `1`,
			Type:  Int(),
			Value: 1,
		},
		{
			Data:  `5`,
			Type:  Int8().WithIntRange(3, 6),
			Value: 5,
		},
		{
			Data:  `3.14`,
			Type:  Float(),
			Value: 3.14,
		},
		{
			Data:  `3.14`,
			Type:  Decimal(3, 2),
			Value: decimal.RequireFromString("3.14"),
		},
		{
			Data:  `1669113414031`,
			Type:  DateTime("ms"),
			Value: time.UnixMilli(1669113414031).UTC(),
		},
		{
			Data:  `"2022-11-22T11:51:49+01:00"`,
			Type:  DateTime(time.RFC3339),
			Value: time.Date(2022, 11, 22, 10, 51, 49, 0, time.UTC),
		},
		{
			Data:  `"2022-11-22"`,
			Type:  Date("2006-01-02"),
			Value: time.Date(2022, 11, 22, 0, 0, 0, 0, time.UTC),
		},
		{
			Data:  `18945218`,
			Type:  Time("ms"),
			Value: 18945218,
		},
		{
			Data:  `2022`,
			Type:  Year(),
			Value: 2022,
		},
		{
			Data:  `"f18c2024-beaf-4c7f-b4e1-0dc2d6468b6a"`,
			Type:  UUID(),
			Value: "f18c2024-beaf-4c7f-b4e1-0dc2d6468b6a",
		},
		{
			Data:  `"{\"name\":\"John\"}"`,
			Type:  JSON(),
			Value: `{"name":"John"}`,
		},
		{
			Data:  `"abc"`,
			Type:  Text(Chars(5)),
			Value: "abc",
		},
		{
			Data:  `[]`,
			Type:  Array(Text()),
			Value: []any{},
		},
		{
			Data:  `[3,8,11,2]`,
			Type:  Array(Int()),
			Value: []any{3, 8, 11, 2},
		},
		{
			Data: `{"first_name":"John Smith","values":[3, 8, 1],"address":{"city":"Venice","country":"IT"}}`,
			Type: Object([]Property{
				{
					Name:        "first_name",
					Label:       "First name",
					Description: "The first name of a customer",
					Type:        Text(),
				},
				{
					Name:  "values",
					Label: "Values",
					Type:  Array(Int()),
				},
				{
					Name:  "address",
					Label: "address",
					Type: Object([]Property{
						{
							Name:        "city",
							Label:       "City",
							Description: "The city of the address",
							Type:        Text(),
						},
						{
							Name:  "country",
							Label: "Country",
							Type:  Text(),
						},
					}),
				},
			}),
			Value: map[string]any{
				"first_name": "John Smith",
				"values":     []any{3, 8, 1},
				"address": map[string]any{
					"city":    "Venice",
					"country": "IT",
				},
			},
		},
	}

	for _, test := range tests {
		dec := json.NewDecoder(strings.NewReader(test.Data))
		dec.UseNumber()
		got, err := decode(dec, nil, test.Type, false)
		if err != nil {
			t.Errorf("cannot decode '%s': %s", test.Data, err)
			continue
		}
		if !reflect.DeepEqual(got, test.Value) {
			t.Errorf("non equals")
		}
	}

}
