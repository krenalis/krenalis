// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

func TestImportObjectsIntoWarehouse(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	dummy := c.CreateDummy("Dummy (source)", meergotester.Source)
	importUsersID := c.CreatePipeline(dummy, "User", meergotester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "ios", Type: types.Object([]types.Property{
				{Name: "id", Type: types.Text(), ReadOptional: true},
				{Name: "idfa", Type: types.Text(), ReadOptional: true},
			}), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Function: &meergotester.TransformationFunction{
				Language: "Python",
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
				InPaths:  []string{"email"},
				OutPaths: []string{"email", "ios"},
			},
		},
	})
	exec := c.ExecutePipeline(importUsersID)
	c.WaitForExecutionsCompletion(dummy, exec)

	// Check if the profiles have been imported - and then returned - correctly.

	profiles, _, total := c.Profiles([]string{"email", "ios"}, "email", false, 0, 1)

	// Validate the profiles total.
	const expectedTotal = 10
	if total != expectedTotal {
		t.Fatalf("expected \"total\" to be %d, got %d", expectedTotal, total)
	}

	// Validate the profiles.
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
	if len(expectedProperties) != len(profiles) {
		t.Fatalf("expected %d profiles, got %d", len(expectedProperties), len(profiles))
	}
	for i, profile := range profiles {
		expected := expectedProperties[i]
		if !reflect.DeepEqual(expected, profile.Attributes) {
			t.Fatalf("expected %#v, got %#v", expected, profile)
		}
	}

}
