//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package schemas

import (
	"testing"
)

func Test_Schema(t *testing.T) {
	const expected = 20
	if n := Event.Properties().Count(); n != expected {
		t.Fatalf("expected %d properties, got %d", expected, n)
	}
}
