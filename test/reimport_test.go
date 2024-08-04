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

func TestReimport(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// First of all, create a Dummy connection.
	dummy := c.AddDummy("Dummy", meergotester.Source)

	// Add an action that imports users from Dummy, that imports:
	//
	// - the email
	// - the first name
	//
	dummyAction := c.AddAction(dummy, "Users", meergotester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "firstName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: meergotester.Transformation{
			Mapping: map[string]string{
				"email":      "email",
				"first_name": "firstName",
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
	users, _, count := c.Users([]string{"email", "first_name", "last_name"}, "email", false, 0, 2)
	if count != expectedCount {
		t.Fatalf("expected a total of %d users, got %d", expectedCount, count)
	}
	assertEq("first  user email", "abenois2@example.com", users[0].Properties["email"])
	assertEq("first  user first name", "Ariela", users[0].Properties["first_name"])
	assertEq("first  user last name", nil, users[0].Properties["last_name"])
	assertEq("second user email", "bdroghan5@example.com", users[1].Properties["email"])
	assertEq("second user first name", "Bryon", users[1].Properties["first_name"])
	assertEq("second user last name", nil, users[1].Properties["last_name"])

	// Change an action that imports users from Dummy, that imports:
	//
	// - the email
	// - the last name (instead of the first name)
	//
	c.SetAction(dummy, dummyAction, meergotester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "lastName", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: meergotester.Transformation{
			Mapping: map[string]string{
				"email":     "email",
				"last_name": "lastName",
			},
		},
	})

	// Import again the users from dummy.
	c.ExecuteAction(dummy, dummyAction, true) // reimport = true
	c.WaitActionsToFinish(dummy)

	// Check the users again.
	//
	// This time the first name must be nil, while the last name should have a value.
	// TODO: The previous statement will only be true after issue #767 is resolved.
	users, _, count = c.Users([]string{"email", "first_name", "last_name"}, "email", false, 0, 2)
	if count != expectedCount {
		t.Fatalf("expected a total of %d users, got %d", expectedCount, count)
	}
	assertEq("first  user email", "abenois2@example.com", users[0].Properties["email"])
	//assertEq("first  user first name", nil, users[0].Properties["first_name"])    // <- now is nil (see issue https://github.com/meergo/meergo/issues/767)
	assertEq("first  user last name", "Benois", users[0].Properties["last_name"]) // <- now has a value
	assertEq("second user email", "bdroghan5@example.com", users[1].Properties["email"])
	//assertEq("second user first name", nil, users[1].Properties["first_name"])     // <- now is nil (see issue https://github.com/meergo/meergo/issues/767)
	assertEq("second user last name", "Droghan", users[1].Properties["last_name"]) // <- now has a value

}
