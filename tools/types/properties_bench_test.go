// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types

import (
	"strconv"
	"strings"
	"testing"
)

// BenchmarkPropertiesWalkObjectsEarlyStop measures time and allocations when
// iteration is stopped after visiting a parent with many children and a
// short or long name.
func BenchmarkPropertiesWalkObjectsEarlyStop(b *testing.B) {

	children := make([]Property, 1024)
	for i := range children {
		children[i] = Property{Name: "child_" + strconv.Itoa(i), Type: String()}
	}

	for _, tc := range []struct {
		name       string
		parentName string
	}{
		{"short_name", "parent"},
		{"long_name", strings.Repeat("p", 4096)},
	} {
		b.Run(tc.name, func(b *testing.B) {

			properties := Object([]Property{{Name: tc.parentName, Type: Object(children)}}).Properties()

			var got string
			b.ReportAllocs()
			for b.Loop() {
				for path := range properties.WalkObjects() {
					got = path
					break
				}
			}
			if got != tc.parentName {
				b.Fatalf("expected path %q, got %q", tc.parentName, got)
			}

		})
	}

}
