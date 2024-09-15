//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestImportWithTransformation(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Create a Dummy (source) connection.
	dummyID := c.AddDummy("Dummy (source)", meergotester.Source)

	c.ChangeIdentityResolutionSettings(true, []string{"email"})

	// Add an action with a transformation function which imports users, then
	// execute it.
	importUsersID := c.AddAction(dummyID, "Users", meergotester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "firstName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "gender", Type: types.Text(), ReadOptional: true},
		}),
		Transformation: meergotester.Transformation{
			Function: &meergotester.TransformationFunction{
				Source: `
def transform(user: dict) -> dict:
	if user["firstName"] == "Jerad":
		gender = "male"
	else:
		gender = "female"
	return {
		"email": user["email"],
		"first_name": user["firstName"],
		"gender": gender,
	}`,
				Language:      "Python",
				InProperties:  []string{"firstName", "email"},
				OutProperties: []string{"email", "first_name", "gender"},
			},
		},
	})
	exec := c.ExecuteAction(dummyID, importUsersID, true)
	c.WaitForExecutionsCompletion(dummyID, exec)

	// Retrieve the users.
	const expectedTotalCount = 10
	users, _, count := c.Users([]string{"email", "first_name", "gender"}, "email", false, 0, expectedTotalCount)

	// Validate the users count.
	if count != expectedTotalCount {
		t.Fatalf("expected \"count\" to be %d, got %d", expectedTotalCount, count)
	}

	// Validate the count of the returned users.
	usersLen := len(users)
	if expectedTotalCount != usersLen {
		t.Fatalf("expected %d users, got %d", expectedTotalCount, usersLen)
	}

	// Validate the users.
	expectedProperties := []map[string]any{
		{"email": "abenois2@example.com", "first_name": "Ariela", "gender": "female"},
		{"email": "bdroghan5@example.com", "first_name": "Bryon", "gender": "female"},
		{"email": "ctroy7@example.com", "first_name": "Codie", "gender": "female"},
		{"email": "cveschambes3@example.com", "first_name": "Conroy", "gender": "female"},
		{"email": "gclother1@example.com", "first_name": "Glyn", "gender": "female"},
		{"email": "jdebrett9@example.com", "first_name": "Jerad", "gender": "male"},
		{"email": "jsharpin8@example.com", "first_name": "Janifer", "gender": "female"},
		{"email": "kbuessen0@example.com", "first_name": "Kinsley", "gender": "female"},
		{"email": "kdericut4@example.com", "first_name": "Kingsly", "gender": "female"},
		{"email": "kfellon6@example.com", "first_name": "Katine", "gender": "female"},
	}
	if len(expectedProperties) != len(users) {
		t.Fatalf("expected %d users, got %d", len(expectedProperties), len(users))
	}
	for i, user := range users {
		expected := expectedProperties[i]
		if !reflect.DeepEqual(expected, user.Properties) {
			t.Fatalf("expected %#v, got %#v", expected, user)
		}
	}

}
