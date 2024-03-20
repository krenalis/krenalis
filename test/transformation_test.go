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

	"chichi/connector/types"
	"chichi/test/chichitester"
)

func TestImportWithTransformation(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Create a Dummy (source) connection.
	dummyID := c.AddDummy("Dummy (source)", chichitester.Source, "")

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
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "firstName", Type: types.Text(), Nullable: true},
			{Name: "gender", Type: types.Text().WithValues("male", "female", "other"), Nullable: true},
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
		"firstName": user["firstName"],
		"gender": gender,
	}`,
				Language: "Python",
			},
		},
	})
	c.ExecuteAction(dummyID, importUsersID, true)
	c.WaitActionsToFinish(dummyID)

	// Retrieve the users.
	const expectedTotalCount = 10
	users, _, count := c.Users([]string{"email", "firstName", "gender"}, "", 0, expectedTotalCount)

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
	expectedUsers := []map[string]any{
		{"email": "kbuessen0@example.com", "firstName": "Kinsley", "gender": "female"},
		{"email": "jdebrett9@example.com", "firstName": "Jerad", "gender": "male"},
		{"email": "emoakes2r@example.com", "firstName": "Edyth", "gender": "female"},
		{"email": "lwhitesonrr@example.com", "firstName": "Leann", "gender": "female"},
		{"email": "sattestone2s@example.com", "firstName": "Susanne", "gender": "female"},
		{"email": "aquittonden2t@example.com", "firstName": "Aimil", "gender": "female"},
		{"email": "tbrayson2u@example.com", "firstName": "Teodora", "gender": "female"},
		{"email": "csifflett2v@example.com", "firstName": "Cristiano", "gender": "female"},
		{"email": "mpordal2w@example.com", "firstName": "Mona", "gender": "female"},
		{"email": "aniece2x@example.com", "firstName": "Ashil", "gender": "female"},
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
