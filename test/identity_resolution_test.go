//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

// TestIdentityResolution tests the identity resolution by importing users and
// retrieving the users from the APIs.
//
// This works by importing users through a JSON file, which is created (or
// updated) every time a user is imported, then it's loaded into Meergo by
// running the import action on the JSON.
func TestIdentityResolution(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Create a storage where the JSON files (containing the incoming users)
	// will be created.
	storageDir, err := os.MkdirTemp("", "meergo-test-identity-resolution")
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

	properties := map[string]bool{
		"dummyId":      true,
		"email":        true,
		"phoneNumbers": true,
	}

	inSchemaProps := []types.Property{
		{Name: "dummyId", Type: types.JSON()},
		{Name: "email", Type: types.JSON()},
		{Name: "phoneNumbers", Type: types.JSON()},
	}
	outSchemaProps := []types.Property{
		{Name: "dummy_id", Type: types.Text(), ReadOptional: true},
		{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		{Name: "phone_numbers", Type: types.Array(types.Text().WithCharLen(300)), ReadOptional: true},
	}

	c.ChangeIdentityResolutionSettings(true, []string{"dummy_id", "email"})

	// Generate and add an action to the JSON for importing the users.
	mapping := map[string]string{
		"dummy_id":      "dummyId",
		"email":         "email",
		"phone_numbers": "phoneNumbers",
	}

	// Add the action A.
	actionA := c.AddAction(fsID, "Users", meergotester.ActionToSet{
		Name:      "Action A",
		Path:      "users.json",
		InSchema:  types.Object(inSchemaProps),
		OutSchema: types.Object(outSchemaProps),
		Transformation: meergotester.Transformation{
			Mapping: mapping,
		},
		IdentityProperty: "dummyId",
		Connector:        "JSON",
		UIValues:         meergotester.UIJSONProperties(properties),
	})

	// Add the action B.
	actionB := c.AddAction(fsID, "Users", meergotester.ActionToSet{
		Name:      "Action B",
		Path:      "users.json",
		InSchema:  types.Object(inSchemaProps),
		OutSchema: types.Object(outSchemaProps),
		Transformation: meergotester.Transformation{
			Mapping: mapping,
		},
		IdentityProperty: "dummyId",
		Connector:        "JSON",
		UIValues:         meergotester.UIJSONProperties(properties),
	})

	// Define a function "expectUsers" which checks if the expected user
	// properties match with the users on the data warehouse.
	expectUsers := func(expectedUsers []map[string]any) {

		// Retrieve the users from the APIs.
		users, _, _ := c.Users([]string{"dummy_id", "email", "phone_numbers"}, "dummy_id", false, 0, 1000)

		// Check if the users are equal to the expected or not.
		if len(expectedUsers) != len(users) {
			t.Fatalf("\nexpected: %d users\ngot %d", len(expectedUsers), len(users))
		}
		for i, user := range users {
			if !reflect.DeepEqual(expectedUsers[i], user.Properties) {
				t.Fatalf("\nexpected at index %d: %#v\ngot:                %s%#v", i, expectedUsers[i], strings.Repeat(" ", len(strconv.Itoa(i))), user.Properties)
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
			t.Fatalf("cannot write the incoming user to the JSON file: %s", err)
		}

		// Import the users in the JSON.
		exec := c.ExecuteAction(fsID, action, false)
		c.WaitForExecutionsCompletion(fsID, exec)

	}

	// -------------------------------------------------------------------------

	// Add the tests on the identity resolution here.

	expectUsers([]map[string]any{})

	expectUsers([]map[string]any{})
	importUser(actionA, map[string]any{"dummyId": "AAA", "email": "", "phoneNumbers": []any{}})
	expectUsers([]map[string]any{
		{"dummy_id": "AAA", "email": "", "phone_numbers": []any{}},
	})

	importUser(actionA, map[string]any{"dummyId": "AAA", "email": "", "phoneNumbers": []any{"333"}})
	expectUsers([]map[string]any{
		{"dummy_id": "AAA", "email": "", "phone_numbers": []any{"333"}},
	})

	importUser(actionA, map[string]any{"dummyId": "BBB", "email": "", "phoneNumbers": []any{"333"}})
	expectUsers([]map[string]any{
		{"dummy_id": "BBB", "email": "", "phone_numbers": []any{"333"}},
	})

	importUser(actionB, map[string]any{"dummyId": "AAA", "email": "a@b", "phoneNumbers": []any{"333"}})
	expectUsers([]map[string]any{
		{"dummy_id": "AAA", "email": "a@b", "phone_numbers": []any{"333"}},
		{"dummy_id": "BBB", "email": "", "phone_numbers": []any{"333"}},
	})

	importUser(actionA, map[string]any{"dummyId": "AAA", "email": "a@b", "phoneNumbers": []any{"444"}})
	expectUsers([]map[string]any{
		{"dummy_id": "AAA", "email": "a@b", "phone_numbers": []any{"333", "444"}},
	})

	// -------------------------------------------------------------------------

	// The test completed successfully, so the temporary directory for the
	// storage can be removed.
	removeTempDirectory = true

}
