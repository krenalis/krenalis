//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package datastore

import (
	"reflect"
	"testing"
)

func TestIDsSerializationDeserialization(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"", []int{}},
		{"20", []int{20}},
		{"10,10", []int{10, 10}},
		{"1,2,20", []int{1, 2, 20}},
		{"10,15,30", []int{10, 15, 30}},
		{"5,25,50,100", []int{5, 25, 50, 100}},
		{"32895,25,50,100", []int{32895, 25, 50, 100}},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := deserializeIDs(test.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if !reflect.DeepEqual(test.expected, got) {
				t.Fatalf("expecting %v, got %v", test.expected, got)
			}
			serialized := serializeIDs(got)
			if test.input != serialized {
				t.Fatalf("expecting %q, got %q", test.input, serialized)
			}
		})
	}
}
