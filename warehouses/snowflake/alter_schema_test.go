// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"fmt"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// Test_typeToSnowflakeType verifies the mapping between Krenalis types and the
// Snowflake type strings returned by typeToSnowflakeType.
func Test_typeToSnowflakeType(t *testing.T) {

	tests := []struct {
		typ      types.Type
		expected string
	}{
		// string.
		{types.String().AsPhone(), "VARCHAR(16)"},
		{types.String().AsCountry(types.ISO3166Alpha2), "VARCHAR(2)"},

		// array.
		{types.Array(types.String().AsPhone()), "ARRAY"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test.typ), func(t *testing.T) {
			gotType := typeToSnowflakeType(test.typ)
			if test.expected != gotType {
				t.Fatalf("expected %q to be returned, got %q instead", test.expected, gotType)
			}
		})
	}

}
