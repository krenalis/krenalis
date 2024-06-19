//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package events

import (
	"testing"

	"github.com/open2b/chichi/types"
)

func Test_Schema(t *testing.T) {

	if n := types.NumProperties(Schema); n != 15 {
		t.Fatalf("expecting 15 properties, got %d", n)
	}

}
