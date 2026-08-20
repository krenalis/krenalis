// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package krenalistester

import (
	"encoding/json"
	"testing"
)

// TestFilterJSON verifies the recursive filter JSON representation, including
// encoding a nil FilterCondition.Values slice as an empty array.
func TestFilterJSON(t *testing.T) {

	filter := Filter{
		Operator: OpAnd,
		Rules: []FilterRule{
			&FilterCondition{Property: "x", Operator: OpExists},
			&Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "y", Operator: OpIs, Values: []string{"a"}},
					&FilterCondition{Property: "z", Operator: OpIs, Values: []string{"b"}},
				},
			},
		},
	}

	got, err := json.Marshal(filter)
	if err != nil {
		t.Fatalf("cannot marshal filter: %v", err)
	}
	expected := `{"operator":"and","rules":[{"property":"x","operator":"exists","values":[]},` +
		`{"operator":"or","rules":[{"property":"y","operator":"is","values":["a"]},` +
		`{"property":"z","operator":"is","values":["b"]}]}]}`
	if string(got) != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}

}
