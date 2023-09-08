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

	allProps := []string{"dummy_id", "Email", "PhoneNumbers"}
	identifiers := []string{"dummy_id", "Email", "PhoneNumbers"}
	inSchemaProps := []types.Property{
		{Name: "dummy_id", Type: types.JSON()},
		{Name: "Email", Type: types.JSON()},
		{Name: "PhoneNumbers", Type: types.JSON()},
	}
	outSchemaProps := []types.Property{
		{Name: "dummy_id", Type: types.Text()},
		{Name: "Email", Type: types.Text()},
		{Name: "PhoneNumbers", Type: types.Array(types.Text())},
	}

	// Generate and add an action to the JSON for importing the users.
	mapping := map[string]string{}
	for _, p := range allProps {
		mapping[p] = p
	}

	// Add the action A.
	actionA := c.AddAction(jsonID, map[string]any{
		"Target": "Users",
		"Action": map[string]any{
			"Name":        "Action A",
			"Path":        "users.json",
			"InSchema":    types.Object(inSchemaProps),
			"OutSchema":   types.Object(outSchemaProps),
			"Identifiers": identifiers,
			"Mapping":     mapping,
		},
	})

	// Add the action B.
	actionB := c.AddAction(jsonID, map[string]any{
		"Target": "Users",
		"Action": map[string]any{
			"Name":        "Action B",
			"Path":        "users.json",
			"InSchema":    types.Object(inSchemaProps),
			"OutSchema":   types.Object(outSchemaProps),
			"Identifiers": identifiers,
			"Mapping":     mapping,
		},
	})

	// Define a function "expectUsers" which checks if the expected users match
	// with the users on the data warehouse.
	expectUsers := func(expected []map[string]any) {

		// Retrieve the users from the APIs and convert their format.
		rawUsers := c.Users(allProps, 0, 1000)["users"].([]any)
		gotUsers := make([]map[string]any, len(rawUsers))
		for i := range rawUsers {
			u := map[string]any{}
			for j, p := range allProps {
				u[p] = rawUsers[i].([]any)[j]
			}
			gotUsers[i] = u
		}

		// Check if the users are equal to the expected or not.
		if !reflect.DeepEqual(expected, gotUsers) {
			t.Fatalf("\nexpected: %#v\ngot:      %#v", expected, gotUsers)
		}
		t.Logf("users: %v", gotUsers)
	}

	// Define a function "importUser" which imports the user into the data
	// warehouse.
	importUser := func(action int, props map[string]any) {

		// Create a JSON file with the user.
		t.Logf("importing user %v", props)
		content, err := json.Marshal([]any{props})
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
	importUser(actionA, map[string]any{"dummy_id": "AAA", "Email": "", "PhoneNumbers": []any{}})
	expectUsers([]map[string]any{
		{"dummy_id": "AAA", "Email": "", "PhoneNumbers": []any{}},
	})

	importUser(actionA, map[string]any{"dummy_id": "AAA", "Email": "", "PhoneNumbers": []any{"333"}})
	expectUsers([]map[string]any{
		{"dummy_id": "AAA", "Email": "", "PhoneNumbers": []any{"333"}},
	})

	importUser(actionA, map[string]any{"dummy_id": "BBB", "Email": "", "PhoneNumbers": []any{"333"}})
	expectUsers([]map[string]any{
		{"dummy_id": "AAA", "Email": "", "PhoneNumbers": []any{"333"}},
		{"dummy_id": "BBB", "Email": "", "PhoneNumbers": []any{"333"}},
	})

	importUser(actionB, map[string]any{"dummy_id": "BBB", "Email": "a@b", "PhoneNumbers": []any{"333"}})
	expectUsers([]map[string]any{
		{"dummy_id": "AAA", "Email": "", "PhoneNumbers": []any{"333"}},
		{"dummy_id": "BBB", "Email": "a@b", "PhoneNumbers": []any{"333"}},
	})

	importUser(actionB, map[string]any{"dummy_id": "BBB", "Email": "a@b", "PhoneNumbers": []any{"444"}})
	expectUsers([]map[string]any{
		{"dummy_id": "AAA", "Email": "", "PhoneNumbers": []any{"333"}},
		{"dummy_id": "BBB", "Email": "a@b", "PhoneNumbers": []any{"444", "333"}},
	})

	// -------------------------------------------------------------------------

	// The test completed successfully, so the temporary directory for the
	// storage can be removed.
	removeTempDirectory = true

}
