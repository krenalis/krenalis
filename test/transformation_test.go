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

	"chichi/apis"
	"chichi/connector"
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
	dummyID := c.AddDummy("Dummy (source)", connector.Source)

	c.SetWorkspaceIdentifiers([]string{"email"}, apis.AnonymousIdentifiers{})

	// Add an action with a transformation function which imports users, then
	// execute it.
	importUsersID := c.AddAction(dummyID, map[string]any{
		"Target": "Users",
		"Action": map[string]any{
			"Name": "Import users from Dummy",
			"InSchema": types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "first_name", Type: types.Text()},
			}),
			"OutSchema": types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "first_name", Type: types.Text()},
				{Name: "gender", Type: types.Text().WithValues("male", "female", "other")},
			}),
			"Transformation": map[string]any{
				"Function": map[string]any{
					"Source": `
def transform(user: dict) -> dict:
	if user["first_name"] == "Jerad":
		gender = "male"
	else:
		gender = "female"
	return {
		"email": user["email"],
		"first_name": user["first_name"],
		"gender": gender,
	}`,
					"Language": "Python",
				},
			},
		},
	})
	c.ExecuteAction(dummyID, importUsersID, true)
	c.WaitActionsToFinish(dummyID)

	// Retrieve the users.
	const expectedTotalCount = 10
	ret := c.Users([]string{"email", "first_name", "gender"}, "", 0, expectedTotalCount)

	// Validate the total count of the users.
	totalCount := len(ret["users"].([]any))
	if expectedTotalCount != totalCount {
		t.Fatalf("expecting a total of %d users, got %d", expectedTotalCount, totalCount)
	}

	// Validate the users.
	users := ret["users"].([]any)
	expectedUsers := []map[string]any{
		{"email": "kbuessen0@example.com", "first_name": "Kinsley", "gender": "female"},
		{"email": "jdebrett9@example.com", "first_name": "Jerad", "gender": "male"},
		{"email": "emoakes2r@example.com", "first_name": "Edyth", "gender": "female"},
		{"email": "lwhitesonrr@example.com", "first_name": "Leann", "gender": "female"},
		{"email": "sattestone2s@example.com", "first_name": "Susanne", "gender": "female"},
		{"email": "aquittonden2t@example.com", "first_name": "Aimil", "gender": "female"},
		{"email": "tbrayson2u@example.com", "first_name": "Teodora", "gender": "female"},
		{"email": "csifflett2v@example.com", "first_name": "Cristiano", "gender": "female"},
		{"email": "mpordal2w@example.com", "first_name": "Mona", "gender": "female"},
		{"email": "aniece2x@example.com", "first_name": "Ashil", "gender": "female"},
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
