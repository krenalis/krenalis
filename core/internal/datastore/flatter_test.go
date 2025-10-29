// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/types"
)

func Test_Flatter(t *testing.T) {

	schema := types.Object([]types.Property{
		{
			Name: "name",
			Type: types.Text(),
		},
		{
			Name: "address",
			Type: types.Object([]types.Property{
				{
					Name: "street",
					Type: types.Text(),
				},
				{
					Name: "city",
					Type: types.Text(),
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

	columnByProperty := map[string]meergo.Column{
		"name": {
			Name: "name",
			Type: types.Text(),
		},
		"address.street": {
			Name: "address_street",
			Type: types.Text(),
		},
		"address.city": {
			Name: "address_city",
			Type: types.Text(),
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
		properties map[string]any
		expected   map[string]any
	}{
		{
			properties: map[string]any{
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
			flatter.flat(test.properties)
			if !reflect.DeepEqual(test.properties, test.expected) {
				t.Fatalf("expected %v, got %v", test.expected, test.properties)
			}
		})
	}

}
