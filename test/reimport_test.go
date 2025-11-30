// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func TestReimport(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	c.UpdateIdentityResolution(false, nil)

	// First of all, create a Dummy connection.
	dummy := c.CreateDummy("Dummy", meergotester.Source)

	// Create a pipeline that imports users from Dummy, that imports:
	//
	// - the email
	// - the first name
	//
	dummyPipeline := c.CreatePipeline(dummy, "User", meergotester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
			{Name: "firstName", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithCharLen(300), ReadOptional: true},
			{Name: "first_name", Type: types.String().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email":      "email",
				"first_name": "firstName",
			},
		},
	})

	// Import the users from dummy.
	exec := c.ExecutePipeline(dummyPipeline)
	c.WaitForExecutionsCompletion(dummy, exec)

	// Run the Identity Resolution.
	c.RunIdentityResolution()

	// Check the profiles.
	assertEq := func(msg string, expected, got any) {
		if !reflect.DeepEqual(expected, got) {
			t.Fatalf("%s: expected value %#v, got %#v", msg, expected, got)
		}
		t.Logf("%s: value %#v matches the expected value", msg, expected)
	}
	const expectedTotal = 10
	profiles, _, total := c.Profiles([]string{"email", "first_name", "last_name"}, "email", false, 0, 2)
	if total != expectedTotal {
		t.Fatalf("expected a total of %d profiles, got %d", expectedTotal, total)
	}
	assertEq("first  user email", "abenois2@example.com", profiles[0].Attributes["email"])
	assertEq("first  user first name", "Ariela", profiles[0].Attributes["first_name"])
	assertEq("first  user last name", nil, profiles[0].Attributes["last_name"])
	assertEq("second user email", "bdroghan5@example.com", profiles[1].Attributes["email"])
	assertEq("second user first name", "Bryon", profiles[1].Attributes["first_name"])
	assertEq("second user last name", nil, profiles[1].Attributes["last_name"])

	// Update the pipeline that imports users from Dummy, that imports:
	//
	// - the email
	// - the last name (instead of the first name)
	//
	c.UpdatePipeline(dummyPipeline, meergotester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
			{Name: "lastName", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithCharLen(300), ReadOptional: true},
			{Name: "last_name", Type: types.String().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email":     "email",
				"last_name": "lastName",
			},
		},
	})

	// Import again the users from Dummy.
	exec = c.ExecutePipeline(dummyPipeline)
	c.WaitForExecutionsCompletion(dummy, exec)

	// Run the Identity Resolution.
	c.RunIdentityResolution()

	// Check the profiles again.
	//
	// This time the first name must be nil, while the last name should have a value.
	profiles, _, total = c.Profiles([]string{"email", "first_name", "last_name"}, "email", false, 0, 2)
	if total != expectedTotal {
		t.Fatalf("expected a total of %d profiles, got %d", expectedTotal, total)
	}
	assertEq("first  user email", "abenois2@example.com", profiles[0].Attributes["email"])
	assertEq("first  user first name", nil, profiles[0].Attributes["first_name"])    // <- now is nil
	assertEq("first  user last name", "Benois", profiles[0].Attributes["last_name"]) // <- now has a value
	assertEq("second user email", "bdroghan5@example.com", profiles[1].Attributes["email"])
	assertEq("second user first name", nil, profiles[1].Attributes["first_name"])     // <- now is nil
	assertEq("second user last name", "Droghan", profiles[1].Attributes["last_name"]) // <- now has a value

}
