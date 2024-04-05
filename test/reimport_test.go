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

func TestReimport(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// First of all, create a Dummy connection.
	dummy := c.AddDummy("Dummy", chichitester.Source)

	// Add an action that imports users from Dummy, that imports:
	//
	// - the email
	// - the firstName
	//
	dummyAction := c.AddAction(dummy, "Users", chichitester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "firstName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "firstName", Type: types.Text(), Nullable: true},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email":     "email",
				"firstName": "firstName",
			},
		},
	})

	// Import the users from dummy.
	c.ExecuteAction(dummy, dummyAction, true)
	c.WaitActionsToFinish(dummy)

	// Check the users.
	assertEq := func(msg string, expected, got any) {
		if !reflect.DeepEqual(expected, got) {
			t.Fatalf("%s: expected value %#v, got %#v", msg, expected, got)
		}
		t.Logf("%s: value %#v matches the expected value", msg, expected)
	}
	const expectedCount = 10
	users, _, count := c.Users([]string{"email", "firstName", "lastName"}, "email", 0, 2)
	if count != expectedCount {
		t.Fatalf("expecting a total of %d users, got %d", expectedCount, count)
	}
	assertEq("first  user email", "kbuessen0@example.com", users[0]["email"])
	assertEq("first  user firstName", "Kinsley", users[0]["firstName"])
	assertEq("first  user last name", nil, users[0]["lastName"])
	assertEq("second user email", "jdebrett9@example.com", users[1]["email"])
	assertEq("second user firstName", "Jerad", users[1]["firstName"])
	assertEq("second user last name", nil, users[1]["lastName"])

	// Change an action that imports users from Dummy, that imports:
	//
	// - the email
	// - the lastName (instead of the firstName)
	//
	c.SetAction(dummy, dummyAction, chichitester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "lastName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "lastName", Type: types.Text(), Nullable: true},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email":    "email",
				"lastName": "lastName",
			},
		},
	})

	// Import again the users from dummy.
	c.ExecuteAction(dummy, dummyAction, true) // reimport = true
	c.WaitActionsToFinish(dummy)

	// Check the users again.
	//
	// This time the firstName must be nil (as it the users have been deleted),
	// while the lastName should have a value.
	users, _, count = c.Users([]string{"email", "firstName", "lastName"}, "email", 0, 2)
	if count != expectedCount {
		t.Fatalf("expecting a total of %d users, got %d", expectedCount, count)
	}
	assertEq("first  user email", "kbuessen0@example.com", users[0]["email"])
	assertEq("first  user firstName", nil, users[0]["firstName"])     // <- now is nil
	assertEq("first  user lastName", "Buessen", users[0]["lastName"]) // <- now has a value
	assertEq("second user email", "jdebrett9@example.com", users[1]["email"])
	assertEq("second user firstName", nil, users[1]["firstName"])     // <- now is nil
	assertEq("second user lastName", "Debrett", users[1]["lastName"]) // <- now has a value

}
