//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"chichi/connector/types"
	"chichi/test/chichitester"
)

// TestIdentityResolution tests the identity resolution by importing users and
// retrieving the users from the APIs.
//
// This works by importing users through a JSON file, which is created (or
// updated) every time a user is imported, then it's loaded into Chichi by
// running the import action on the JSON.
func TestIdentityResolution(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Create a storage where the JSON files (containing the incoming users)
	// will be created.
	storageDir, err := os.MkdirTemp("", "chichi-test-identity-resolution")
	if err != nil {
		t.Fatal(err)
	}
	removeTempDirectory := false
	defer func() {
		if removeTempDirectory {
			err := os.RemoveAll(storageDir)
			if err != nil {
				t.Logf("cannot remove the temporary directory used by the storage: %s", err)
			}
		} else {
			t.Logf("the temporary directory for the storage %q has been kept for troubleshooting the test", storageDir)
		}
	}()

	jsonFilename := "users.json"
	jsonAbsPath := filepath.Join(storageDir, jsonFilename)

	// Create the Filesystem connection.
	fsID := c.AddSourceFilesystem(storageDir)

	// Create the JSON connection.
	jsonID := c.AddSourceJSON(fsID)

	allProps := []string{"dummyId", "email", "phoneNumbers"}
	identifiers := []string{"dummyId", "email"}
	inSchemaProps := []types.Property{
		{Name: "dummyId", Type: types.JSON()},
		{Name: "email", Type: types.JSON()},
		{Name: "phoneNumbers", Type: types.JSON()},
	}
	outSchemaProps := []types.Property{
		{Name: "dummyId", Type: types.Text()},
		{Name: "email", Type: types.Text()},
		{Name: "phoneNumbers", Type: types.Array(types.Text())},
	}

	c.SetWorkspaceIdentifiers(identifiers)

	// Generate and add an action to the JSON for importing the users.
	mapping := map[string]string{}
	for _, p := range allProps {
		mapping[p] = p
	}

	// Add the action A.
	actionA := c.AddAction(jsonID, "Users", chichitester.ActionToSet{
		Name:      "Action A",
		Path:      "users.json",
		InSchema:  types.Object(inSchemaProps),
		OutSchema: types.Object(outSchemaProps),
		Transformation: chichitester.Transformation{
			Mapping: mapping,
		},
		IdentityColumn: "dummyId",
	})

	// Add the action B.
	actionB := c.AddAction(jsonID, "Users", chichitester.ActionToSet{
		Name:      "Action B",
		Path:      "users.json",
		InSchema:  types.Object(inSchemaProps),
		OutSchema: types.Object(outSchemaProps),
		Transformation: chichitester.Transformation{
			Mapping: mapping,
		},
		IdentityColumn: "dummyId",
	})

	// Define a function "expectUsers" which checks if the expected users match
	// with the users on the data warehouse.
	expectUsers := func(expected []map[string]any) {

		// Retrieve the users from the APIs and convert their format.
		users := c.Users(allProps, "", 0, 1000)["users"].([]any)

		// Check if the users are equal to the expected or not.
		if len(expected) != len(users) {
			t.Fatalf("\nexpected: %d users\ngot %d", len(expected), len(users))
		}
		for i, user := range users {
			if !reflect.DeepEqual(expected[i], user) {
				t.Fatalf("\nexpected at index %d: %#v\ngot:                %s%#v", i, expected, strings.Repeat(" ", len(strconv.Itoa(i))), users)
			}
		}
		t.Logf("users: %v", users)
	}

	// Define a function "importUser" which imports the user into the data
	// warehouse.
	importUser := func(action int, props map[string]any) {

		// Create a JSON file with the user.
		t.Logf("importing user %v", props)
		var s struct {
			DummyID      *string `json:"dummyId,omitempty"`
			Email        *string `json:"email,omitempty"`
			PhoneNumbers *[]any  `json:"phoneNumbers,omitempty"`
		}
		if dummyID, ok := props["dummyId"].(string); ok {
			s.DummyID = &dummyID
		}
		if email, ok := props["email"].(string); ok {
			s.Email = &email
		}
		if phoneNumbers, ok := props["phoneNumbers"].([]any); ok {
			s.PhoneNumbers = &phoneNumbers
		}
		content, err := json.Marshal([]any{s})
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(jsonAbsPath, content, 0755)
		if err != nil {
			log.Fatalf("cannot write the incoming user to the JSON file: %s", err)
		}

		// Import the users in the JSON.
		c.ExecuteAction(jsonID, action, true)
		c.WaitActionsToFinish(jsonID)

	}

	// -------------------------------------------------------------------------

	// Add the tests on the identity resolution here.

	expectUsers([]map[string]any{})

	expectUsers([]map[string]any{})
	importUser(actionA, map[string]any{"dummyId": "AAA", "email": "", "phoneNumbers": []any{}})
	expectUsers([]map[string]any{
		{"dummyId": "AAA", "email": "", "phoneNumbers": []any{}},
	})

	importUser(actionA, map[string]any{"dummyId": "AAA", "email": "", "phoneNumbers": []any{"333"}})
	expectUsers([]map[string]any{
		{"dummyId": "AAA", "email": "", "phoneNumbers": []any{"333"}},
	})

	importUser(actionA, map[string]any{"dummyId": "BBB", "email": "", "phoneNumbers": []any{"333"}})
	expectUsers([]map[string]any{
		{"dummyId": "AAA", "email": "", "phoneNumbers": []any{"333"}},
		{"dummyId": "BBB", "email": "", "phoneNumbers": []any{"333"}},
	})

	importUser(actionB, map[string]any{"dummyId": "BBB", "email": "a@b", "phoneNumbers": []any{"333"}})
	expectUsers([]map[string]any{
		{"dummyId": "AAA", "email": "", "phoneNumbers": []any{"333"}},
		{"dummyId": "BBB", "email": "a@b", "phoneNumbers": []any{"333"}},
	})

	importUser(actionB, map[string]any{"dummyId": "BBB", "email": "a@b", "phoneNumbers": []any{"444"}})
	expectUsers([]map[string]any{
		{"dummyId": "AAA", "email": "", "phoneNumbers": []any{"333"}},
		{"dummyId": "BBB", "email": "a@b", "phoneNumbers": []any{"444"}},
	})

	// -------------------------------------------------------------------------

	// The test completed successfully, so the temporary directory for the
	// storage can be removed.
	removeTempDirectory = true

}
