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
	dummyID := c.AddDummy("Dummy (source)", connector.SourceRole)

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
				{Name: "Email", Type: types.Text()},
				{Name: "FirstName", Type: types.Text()},
				{Name: "Gender", Type: types.Text().WithEnum([]string{"male", "female", "other"})},
			}),
			"Identifiers": []string{"Email"},
			"Mapping": map[string]string{
				"Email": "email",
			},
			"Transformation": map[string]any{
				"Source": `
def transform(user: dict) -> dict:
	if user["first_name"] == "Jerad":
		gender = "male"
	else:
		gender = "female"
	return {
		"FirstName": user["first_name"],
		"Gender": gender,
	}`,
				"Language": "Python",
			},
		},
	})
	c.ExecuteAction(dummyID, importUsersID, true)
	c.WaitActionsToFinish(dummyID)

	// Retrieve the users.
	ret := c.Users([]string{"Email", "FirstName", "Gender"}, 0, 2)

	// Validate the total count of the users.
	totalCount := int(ret["count"].(float64))
	const expectedTotalCount = 10
	if expectedTotalCount != totalCount {
		t.Fatalf("expecting a total of %d users, got %d", expectedTotalCount, totalCount)
	}

	// Validate the users.
	users := ret["users"].([]any)
	expectedUsers := [][]any{
		{"kbuessen0@example.com", "Kinsley", "female"},
		{"jdebrett9@example.com", "Jerad", "male"},
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
