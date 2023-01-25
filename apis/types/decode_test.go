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
			Data:  `null`,
			Type:  Text().WithNull(),
			Value: nil,
		},
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
			Data:  `-12`,
			Type:  Int8().WithIntRange(-12, 5),
			Value: -12,
		},
		{
			Data:  `127`,
			Type:  Int8(),
			Value: 127,
		},
		{
			Data:  `-128`,
			Type:  Int8(),
			Value: -128,
		},
		{
			Data:  `255`,
			Type:  UInt8(),
			Value: uint(255),
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
			Data:  `"11:39:24"`,
			Type:  Time(),
			Value: "11:39:24",
		},
		{
			Data:  `"11:39:24.623901"`,
			Type:  Time(),
			Value: "11:39:24.623901",
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
			Data:  `"192.0.2.235"`,
			Type:  Inet(),
			Value: `192.0.2.235`,
		},
		{
			Data:  `"::FFFF:192.0.2.235"`,
			Type:  Inet(),
			Value: `::FFFF:192.0.2.235`,
		},
		{
			Data:  `"2001:db8::8a2e:370:7334"`,
			Type:  Inet(),
			Value: `2001:db8::8a2e:370:7334`,
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
			Data: `{"first_name":"John Smith","values":[3, 8, 1],"billing_address":{"city":"Venice","country":"IT"},"birthday":null}`,
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
					Name:    "address",
					Aliases: []string{"billing_address"},
					Label:   "address",
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
				{
					Name:  "birthday",
					Label: "Birthday",
					Type:  Date("2006-01-02").WithNull(),
				},
			}),
			Value: map[string]any{
				"first_name": "John Smith",
				"values":     []any{3, 8, 1},
				"address": map[string]any{
					"city":    "Venice",
					"country": "IT",
				},
				"birthday": nil,
			},
		},
		{
			Data:  `{"first":1,"second":2}`,
			Type:  Map(Int()),
			Value: map[string]any{"first": 1, "second": 2},
		},
	}

	for _, test := range tests {
		dec := json.NewDecoder(strings.NewReader(test.Data))
		dec.UseNumber()
		got, err := decodeByType(dec, nil, test.Type, false)
		if err != nil {
			t.Errorf("cannot decode '%s': %s", test.Data, err)
			continue
		}
		if !reflect.DeepEqual(got, test.Value) {
			t.Errorf("non equals")
		}
	}

}
