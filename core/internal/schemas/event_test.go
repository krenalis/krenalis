// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package schemas

import (
	"testing"
)

func Test_Schema(t *testing.T) {
	const expected = 19
	if n := Event.Properties().Len(); n != expected {
		t.Fatalf("expected %d properties, got %d", expected, n)
	}
}
