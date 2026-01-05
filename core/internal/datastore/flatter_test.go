// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

func Test_Flatter(t *testing.T) {

	schema := types.Object([]types.Property{
		{
			Name: "name",
			Type: types.String(),
		},
		{
			Name: "address",
			Type: types.Object([]types.Property{
				{
					Name: "street",
					Type: types.String(),
				},
				{
					Name: "city",
					Type: types.String(),
				},
			}),
		},
		{
			Name: "score",
			Type: types.Object([]types.Property{
				{
					Name: "a",
					Type: types.Object([]types.Property{
						{
							Name: "b",
							Type: types.Int(32),
						},
						{
							Name: "c",
							Type: types.Int(32),
						},
					}),
				},
			}),
		},
		{
			Name: "age",
			Type: types.Int(8),
		},
	})

	columnByProperty := map[string]warehouses.Column{
		"name": {
			Name: "name",
			Type: types.String(),
		},
		"address.street": {
			Name: "address_street",
			Type: types.String(),
		},
		"address.city": {
			Name: "address_city",
			Type: types.String(),
		},
		"score.a.b": {
			Name: "score_a_b",
			Type: types.Int(32),
		},
		"score.a.c": {
			Name: "score_a_c",
			Type: types.Int(32),
		},
		"age": {
			Name: "age",
			Type: types.Int(8),
		},
	}

	tests := []struct {
		attributes map[string]any
		expected   map[string]any
	}{
		{
			attributes: map[string]any{
				"name": "Bob",
				"address": map[string]any{
					"street": "Via Alberata 5",
					"city":   "Rome",
				},
				"score": map[string]any{
					"a": map[string]any{
						"b": 5,
						"c": 8,
					},
				},
				"age": 32,
			},
			expected: map[string]any{
				"name":           "Bob",
				"address_street": "Via Alberata 5",
				"address_city":   "Rome",
				"score_a_b":      5,
				"score_a_c":      8,
				"age":            32,
			},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			flatter := newFlatter(schema, columnByProperty)
			flatter.flat(test.attributes)
			if !reflect.DeepEqual(test.attributes, test.expected) {
				t.Fatalf("expected %v, got %v", test.expected, test.attributes)
			}
		})
	}

}
