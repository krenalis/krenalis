//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"testing"

	"chichi/test/chichitester"
)

func TestIdentifiersSchema(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Retrieve the identifiers schema and ensure it has the correct number of
	// properties.
	schema := c.IdentifiersSchema()
	properties := schema.Properties()
	const expectedLen = 7
	if expectedLen != len(properties) {
		t.Fatalf("expected %d properties in the identifiers schema, got %d", expectedLen, len(properties))
	}

}
