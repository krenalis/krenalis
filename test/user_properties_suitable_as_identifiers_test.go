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

func TestUserPropertiesSuitableAsIdentifiers(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Retrieve the user properties that are suitable as identifiers and ensure
	// it has the correct number of properties.
	schema := c.UserPropertiesSuitableAsIdentifiers()
	const expectedLen = 5
	if n := types.NumProperties(schema); expectedLen != n {
		t.Fatalf("expected %d properties suitable as identifiers, got %d", expectedLen, n)
	}

}
