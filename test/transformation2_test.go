//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

// TestTransformation2 tests that the transformation functions are behaving
// correctly, especially with respect to InPaths and OutPaths.
func TestTransformation2(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Create a Dummy (source) connection.
	dummy := c.CreateDummy("Dummy (source)", meergotester.Source)

	// Create an action with a transformation function which imports users, then
	// execute it.
	action := c.CreateAction(dummy, "Users", meergotester.ActionToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "street", Type: types.Text()},
				{Name: "postal_code", Type: types.Text()}, // not used, but check is done only at first level.
				{Name: "city", Type: types.Text()},        // not used, but check is done only at first level.
			})},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: meergotester.Transformation{
			Function: &meergotester.TransformationFunction{
				Source: `
def transform(user: dict) -> dict:
	assert "@" in user["email"]
	assert isinstance(user["address"], dict)
	assert "street" in user["address"]
	assert "postal_code" not in user["address"]
	assert len(user["address"]) == 1
	return {
		"email": user["email"],
	}`,
				Language: "Python",
				InPaths:  []string{"email", "address.street"},
				OutPaths: []string{"email"},
			},
		},
	})
	exec := c.ExecuteAction(action, true)
	c.WaitForExecutionsCompletion(dummy, exec)

	// Retrieve the users.
	const expectedTotal = 10
	users, _, total := c.Users([]string{"email"}, "email", false, 0, expectedTotal)

	// Validate the users total.
	if total != expectedTotal {
		t.Fatalf("expected \"total\" to be %d, got %d", expectedTotal, total)
	}

	// Validate the total of the returned users.
	usersLen := len(users)
	if expectedTotal != usersLen {
		t.Fatalf("expected %d users, got %d", expectedTotal, usersLen)
	}

	// Validate the users.
	expectedProperties := []map[string]any{
		{"email": "abenois2@example.com"},
		{"email": "bdroghan5@example.com"},
		{"email": "ctroy7@example.com"},
		{"email": "cveschambes3@example.com"},
		{"email": "gclother1@example.com"},
		{"email": "jdebrett9@example.com"},
		{"email": "jsharpin8@example.com"},
		{"email": "kbuessen0@example.com"},
		{"email": "kdericut4@example.com"},
		{"email": "kfellon6@example.com"},
	}
	if len(expectedProperties) != len(users) {
		t.Fatalf("expected %d users, got %d", len(expectedProperties), len(users))
	}
	for i, user := range users {
		expected := expectedProperties[i]
		if !reflect.DeepEqual(expected, user.Traits) {
			t.Fatalf("expected %#v, got %#v", expected, user)
		}
	}

}
