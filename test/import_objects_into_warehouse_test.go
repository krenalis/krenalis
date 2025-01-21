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

func TestImportObjectsIntoWarehouse(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	dummy := c.CreateDummy("Dummy (source)", meergotester.Source)
	importUsersID := c.CreateAction(dummy, "Users", meergotester.ActionToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "ios", Type: types.Object([]types.Property{
				{Name: "id", Type: types.Text(), ReadOptional: true},
				{Name: "idfa", Type: types.Text(), ReadOptional: true},
			}), ReadOptional: true},
		}),
		Transformation: meergotester.Transformation{
			Function: &meergotester.TransformationFunction{
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
				InPaths:  []string{"email"},
				OutPaths: []string{"email", "ios"},
			},
		},
	})
	exec := c.ExecuteAction(importUsersID, true)
	c.WaitForExecutionsCompletion(dummy, exec)

	// Check if the users have been imported - and then returned - correctly.

	users, _, total := c.Users([]string{"email", "ios"}, "email", false, 0, 1)

	// Validate the users total.
	const expectedTotal = 10
	if total != expectedTotal {
		t.Fatalf("expected \"total\" to be %d, got %d", expectedTotal, total)
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
		t.Fatalf("expected %d users, got %d", len(expectedProperties), len(users))
	}
	for i, user := range users {
		expected := expectedProperties[i]
		if !reflect.DeepEqual(expected, user.Traits) {
			t.Fatalf("expected %#v, got %#v", expected, user)
		}
	}

}
