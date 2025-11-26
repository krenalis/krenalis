// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

// TestIdentityResolution tests the identity resolution by importing users and
// retrieving the users from the APIs.
//
// This works by importing users through a JSON file, which is created (or
// updated) every time a user is imported, then it's loaded into Meergo by
// running the import pipeline on the JSON.
func TestIdentityResolution(t *testing.T) {

	storage := meergotester.NewTempStorage(t)

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.SetFileSystemRoot(storage.Root())
	c.Start()
	defer c.Stop()

	// Create the File System connection.
	fsID := c.CreateSourceFileSystem()

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

	c.UpdateIdentityResolution(false, []string{"dummy_id", "email"})

	// Create a pipeline for the JSON for importing the users.
	mapping := map[string]string{
		"dummy_id":      "dummyId",
		"email":         "email",
		"phone_numbers": "phoneNumbers",
	}

	// Create the pipeline A.
	pipelineA := c.CreatePipeline(fsID, "User", meergotester.PipelineToSet{
		Name:      "Pipeline A",
		Enabled:   true,
		Path:      "users.json",
		InSchema:  types.Object(inSchemaProps),
		OutSchema: types.Object(outSchemaProps),
		Transformation: &meergotester.Transformation{
			Mapping: mapping,
		},
		IdentityColumn: "dummyId",
		Format:         "json",
		FormatSettings: meergotester.SettingsProperties(properties),
	})

	// Create the pipeline B.
	pipelineB := c.CreatePipeline(fsID, "User", meergotester.PipelineToSet{
		Name:      "Pipeline B",
		Enabled:   true,
		Path:      "users.json",
		InSchema:  types.Object(inSchemaProps),
		OutSchema: types.Object(outSchemaProps),
		Transformation: &meergotester.Transformation{
			Mapping: mapping,
		},
		IdentityColumn: "dummyId",
		Format:         "json",
		FormatSettings: meergotester.SettingsProperties(properties),
	})

	// Define a function "expectProfiles" which checks if the expected profile
	// properties match with the profiles on the data warehouse.
	expectProfiles := func(expectedProfiles []map[string]any) {

		// Retrieve the profiles from the APIs.
		profiles, _, _ := c.Profiles([]string{"dummy_id", "email", "phone_numbers"}, "dummy_id", false, 0, 1000)

		// Check if the users are equal to the expected or not.
		if len(expectedProfiles) != len(profiles) {
			t.Fatalf("\nexpected: %d users\ngot %d", len(expectedProfiles), len(profiles))
		}
		for i, user := range profiles {
			if !reflect.DeepEqual(expectedProfiles[i], user.Attributes) {
				t.Fatalf("\nexpected at index %d: %#v\ngot:                %s%#v", i, expectedProfiles[i], strings.Repeat(" ", len(strconv.Itoa(i))), user.Attributes)
			}
		}
		t.Logf("users: %v", profiles)
	}

	// Define a function "importUser" which imports the user into the data
	// warehouse.
	importUser := func(pipeline int, props map[string]any) {

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
		jsonFilename := "users.json"
		jsonAbsPath := filepath.Join(storage.Root(), jsonFilename)
		err = os.WriteFile(jsonAbsPath, content, 0755)
		if err != nil {
			t.Fatalf("cannot write the incoming user to the JSON file: %s", err)
		}

		// Import the users in the JSON.
		exec := c.ExecutePipeline(pipeline)
		c.WaitForExecutionsCompletion(fsID, exec)
		c.RunIdentityResolution()

	}

	// Add the tests on the identity resolution here.

	expectProfiles([]map[string]any{})

	expectProfiles([]map[string]any{})
	importUser(pipelineA, map[string]any{"dummyId": "AAA", "email": "", "phoneNumbers": []any{}})
	expectProfiles([]map[string]any{
		{"dummy_id": "AAA", "email": "", "phone_numbers": []any{}},
	})

	importUser(pipelineA, map[string]any{"dummyId": "AAA", "email": "", "phoneNumbers": []any{"333"}})
	expectProfiles([]map[string]any{
		{"dummy_id": "AAA", "email": "", "phone_numbers": []any{"333"}},
	})

	importUser(pipelineA, map[string]any{"dummyId": "BBB", "email": "", "phoneNumbers": []any{"333"}})
	expectProfiles([]map[string]any{
		{"dummy_id": "BBB", "email": "", "phone_numbers": []any{"333"}},
	})

	importUser(pipelineB, map[string]any{"dummyId": "AAA", "email": "a@b", "phoneNumbers": []any{"333"}})
	expectProfiles([]map[string]any{
		{"dummy_id": "AAA", "email": "a@b", "phone_numbers": []any{"333"}},
		{"dummy_id": "BBB", "email": "", "phone_numbers": []any{"333"}},
	})

	importUser(pipelineA, map[string]any{"dummyId": "AAA", "email": "a@b", "phoneNumbers": []any{"444"}})
	expectProfiles([]map[string]any{
		{"dummy_id": "AAA", "email": "a@b", "phone_numbers": []any{"333", "444"}},
	})

	// The test completed successfully, so the storage can be removed.
	storage.Remove()

}
