//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package schemas

import (
	"testing"

	"github.com/meergo/meergo/core/types"
)

func Test_Schema(t *testing.T) {
	const expected = 20
	if n := types.NumProperties(Event); n != expected {
		t.Fatalf("expected %d properties, got %d", expected, n)
	}
}
