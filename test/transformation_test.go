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

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestImportWithTransformation(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Create a Dummy (source) connection.
	dummyID := c.AddDummy("Dummy (source)", chichitester.Source)

	c.SetWorkspaceIdentifiers([]string{"email"})

	// Add an action with a transformation function which imports users, then
	// execute it.
	importUsersID := c.AddAction(dummyID, "Users", chichitester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "firstName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "first_name", Type: types.Text()},
			{Name: "gender", Type: types.Text().WithValues("male", "female", "other")},
		}),
		Transformation: chichitester.Transformation{
			Function: &chichitester.TransformationFunction{
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
	c.ExecuteAction(dummyID, importUsersID, true)
	c.WaitActionsToFinish(dummyID)

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
		t.Fatalf("expecting %d users, got %d", expectedTotalCount, usersLen)
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
		t.Fatalf("expecting %d users, got %d", len(expectedProperties), len(users))
	}
	for i, user := range users {
		expected := expectedProperties[i]
		if !reflect.DeepEqual(expected, user.Properties) {
			t.Fatalf("expecting %#v, got %#v", expected, user)
		}
	}

}
