//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestIdentifiersSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Retrieve the identifiers schema and ensure it has the correct number of
	// properties.
	schema := c.IdentifiersSchema()
	const expectedLen = 5
	if n := types.NumProperties(schema); expectedLen != n {
		t.Fatalf("expected %d properties in the identifiers schema, got %d", expectedLen, n)
	}

}
