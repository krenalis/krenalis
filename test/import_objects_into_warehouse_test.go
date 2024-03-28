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
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "ios", Type: types.Object([]types.Property{
				{Name: "id", Type: types.Text(), Nullable: true},
				{Name: "idfa", Type: types.Text(), Nullable: true},
			})}, // TODO(Gianluca): see https://github.com/open2b/chichi/issues/527 for nullability of 'ios'.
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
				Language: "Python",
			},
		},
	})
	c.ExecuteAction(dummy, importUsersID, true)
	c.WaitActionsToFinish(dummy)

	// Check if the users have been imported - and then returned - correctly.

	users, _, count := c.Users([]string{"email", "ios"}, "", 0, 1)

	// Validate the users count.
	const expectedTotalCount = 10
	if count != expectedTotalCount {
		t.Fatalf("expected \"count\" to be %d, got %d", expectedTotalCount, count)
	}

	// Validate the users.
	expectedUsers := []map[string]any{
		{"email": "kbuessen0@example.com",
			"ios": map[string]any{
				"id":        "kbuessen0@example.com-id",
				"idfa":      "kbuessen0@example.com-idfa",
				"pushToken": nil},
		},
	}
	if len(expectedUsers) != len(users) {
		t.Fatalf("expecting %d users, got %d", len(expectedUsers), len(users))
	}
	for i, user := range users {
		expected := expectedUsers[i]
		if !reflect.DeepEqual(expected, user) {
			t.Fatalf("expecting %#v, got %#v", expected, user)
		}
	}

}
