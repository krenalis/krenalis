//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package events

import (
	"testing"

	"github.com/meergo/meergo/types"
)

func Test_Schema(t *testing.T) {
	const expected = 20
	if n := types.NumProperties(Schema); n != expected {
		t.Fatalf("expected %d properties, got %d", expected, n)
	}
}
