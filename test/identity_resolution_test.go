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
// updated) every time an user is imported, then it's loaded into Chichi by
// running the import action on the JSON.
func TestIdentityResolution(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Create a storage where the the JSON files (containing the incoming users)
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

	allProps := []string{"dummy_id", "Email"}
	identifiers := []string{"dummy_id", "Email"}

	// Generate and add an action to the JSON for importing the users.
	inSchemaProps := make([]types.Property, len(allProps))
	outSchemaProps := make([]types.Property, len(allProps))
	mapping := map[string]string{}
	for i, p := range allProps {
		inSchemaProps[i] = types.Property{Name: p, Type: types.Text()}
		outSchemaProps[i] = types.Property{Name: p, Type: types.Text()}
		mapping[p] = p
	}
	importUsersActionID := c.AddAction(jsonID, map[string]any{
		"Target": "Users",
		"Action": map[string]any{
			"Name":        "Import users from JSON on Filesystem",
			"Path":        "users.json",
			"InSchema":    types.Object(inSchemaProps),
			"OutSchema":   types.Object(outSchemaProps),
			"Identifiers": identifiers,
			"Mapping":     mapping,
		},
	})

	// Define a function "expectUsers" which checks if the expected users match
	// with the users on the data warehouse.
	expectUsers := func(expected []map[string]string) {

		// Retrieve the users from the APIs and convert their format.
		rawUsers := c.Users(allProps, 0, 1000)["users"].([]any)
		gotUsers := make([]map[string]string, len(rawUsers))
		for i := range rawUsers {
			u := map[string]string{}
			for j, p := range allProps {
				v := rawUsers[i].([]any)[j].(string)
				if v != "" {
					u[p] = v
				}
			}
			gotUsers[i] = u
		}

		// Check if the users are equal to the expected or not.
		if !reflect.DeepEqual(expected, gotUsers) {
			t.Fatalf("expecting: %v, got: %v", expected, gotUsers)
		}
		t.Logf("users: %v", gotUsers)
	}

	// Define a function "importUser" which imports the user into the data
	// warehouse.
	importUser := func(props map[string]string) {

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
		c.ExecuteAction(jsonID, importUsersActionID, true)
		c.WaitActionsToFinish(jsonID)

	}

	// -------------------------------------------------------------------------

	// Add the tests on the identity resolution here.

	expectUsers([]map[string]string{})

	importUser(map[string]string{"Email": "a@b"})
	expectUsers([]map[string]string{
		{"Email": "a@b"},
	})

	importUser(map[string]string{"Email": "c@d"})
	expectUsers([]map[string]string{
		{"Email": "a@b"},
		{"Email": "c@d"},
	})

	importUser(map[string]string{"dummy_id": "AAA", "Email": "a@b"})
	expectUsers([]map[string]string{
		{"Email": "a@b"},
		{"Email": "c@d"},
		{"dummy_id": "AAA", "Email": "a@b"},
	})

	importUser(map[string]string{"dummy_id": "AAA", "Email": "e@f"})
	expectUsers([]map[string]string{
		{"Email": "a@b"},
		{"Email": "c@d"},
		{"dummy_id": "AAA", "Email": "e@f"},
	})

	// TODO(Gianluca): see the issue
	// https://github.com/open2b/chichi/issues/254.
	//
	// importUser(map[string]string{"dummy_id": "AAA"})
	// expectUsers([]map[string]string{
	// 	{"Email": "a@b"},
	// 	{"Email": "c@d"},
	// 	{"dummy_id": "AAA", "Email": "e@f"},
	// })

	// -------------------------------------------------------------------------

	// The test completed successfully, so the temporary directory for the
	// storage can be removed.
	removeTempDirectory = true

}
