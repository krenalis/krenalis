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

	if n := types.NumProperties(Schema); n != 19 {
		t.Fatalf("expected 18 properties, got %d", n)
	}

}
