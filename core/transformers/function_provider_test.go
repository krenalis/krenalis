//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

import "testing"

func Test_ValidFunctionName(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
	}{
		{"", false},
		{"abc", true},
		{"a_bc5.py", false},
		{"_a-b", true},
		{"-a", false},
		{"ABC", true},
		{"a", true},
		{" abc.js", false},
		{"abc.js ", false},
		{"ab c.py ", false},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			ok := ValidFunctionName(test.name)
			if test.ok != ok {
				t.Fatalf("ValidFunctionName(%q): expected ok = %t, got %t", test.name, test.ok, ok)
			}
		})
	}
}
