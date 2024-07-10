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

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestImportObjectsIntoWarehouse(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	dummy := c.AddDummy("Dummy (source)", chichitester.Source)
	importUsersID := c.AddAction(dummy, "Users", chichitester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "ios", Type: types.Object([]types.Property{
				{Name: "id", Type: types.Text()},
				{Name: "idfa", Type: types.Text()},
			})},
		}),
		Transformation: chichitester.Transformation{
			Function: &chichitester.TransformationFunction{
				Source: `
def transform(user: dict) -> dict:
	email = user["email"]
	return {
		"email": email,
		"ios": {
			"id": email + "-id",
			"idfa": email + "-idfa",
		}
	}`,
				Language:      "Python",
				InProperties:  []string{"email"},
				OutProperties: []string{"email", "ios"},
			},
		},
	})
	c.ExecuteAction(dummy, importUsersID, true)
	c.WaitActionsToFinish(dummy)

	// Check if the users have been imported - and then returned - correctly.

	users, _, count := c.Users([]string{"email", "ios"}, "email", false, 0, 1)

	// Validate the users count.
	const expectedTotalCount = 10
	if count != expectedTotalCount {
		t.Fatalf("expected \"count\" to be %d, got %d", expectedTotalCount, count)
	}

	// Validate the users.
	expectedProperties := []map[string]any{
		{
			"email": "abenois2@example.com",
			"ios": map[string]any{
				"id":   "abenois2@example.com-id",
				"idfa": "abenois2@example.com-idfa",
				// push_token is not set, so should not be returned by the APIs.
			},
		},
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
