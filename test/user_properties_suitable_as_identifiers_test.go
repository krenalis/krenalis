// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/meergo/meergo/test/meergotester"
)

func TestUserPropertiesSuitableAsIdentifiers(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	// Retrieve the user properties that are suitable as identifiers and ensure
	// it has the correct number of properties.
	schema := c.UserPropertiesSuitableAsIdentifiers()
	const expectedLen = 5
	if n := schema.Properties().Len(); expectedLen != n {
		t.Fatalf("expected %d properties suitable as identifiers, got %d", expectedLen, n)
	}

}
