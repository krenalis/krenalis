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

// TestTransformation2 tests that the transformation functions are behaving
// correctly, especially with respect to InPaths and OutPaths.
func TestTransformation2(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	// Create a Dummy (source) connection.
	dummy := c.CreateDummy("Dummy (source)", meergotester.Source)

	// Create a pipeline with a transformation function which imports users, then run it.
	pipeline := c.CreatePipeline(dummy, "User", meergotester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "street", Type: types.String(), Nullable: true},
			}), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Function: &meergotester.TransformationFunction{
				Language: "Python",
				Source: `
def transform(user: dict) -> dict:
	assert "@" in user["email"]
	assert isinstance(user["address"], dict)
	assert "street" in user["address"]
	assert "postal_code" not in user["address"]
	assert len(user["address"]) == 1
	return {
		"email": user["email"],
	}`,
				InPaths:  []string{"email", "address.street"},
				OutPaths: []string{"email"},
			},
		},
	})
	run := c.RunPipeline(pipeline)
	c.WaitRunsCompletion(dummy, run)

	// Retrieve the profiles.
	const expectedTotal = 10
	profiles, _, total := c.Profiles([]string{"email"}, "email", false, 0, expectedTotal)

	// Validate the profiles total.
	if total != expectedTotal {
		t.Fatalf("expected \"total\" to be %d, got %d", expectedTotal, total)
	}

	// Validate the total of the returned profiles.
	profilesLen := len(profiles)
	if expectedTotal != profilesLen {
		t.Fatalf("expected %d profiles, got %d", expectedTotal, profilesLen)
	}

	// Validate the profiles.
	expectedProperties := []map[string]any{
		{"email": "abenois2@example.com"},
		{"email": "bdroghan5@example.com"},
		{"email": "ctroy7@example.com"},
		{"email": "cveschambes3@example.com"},
		{"email": "gclother1@example.com"},
		{"email": "jdebrett9@example.com"},
		{"email": "jsharpin8@example.com"},
		{"email": "kbuessen0@example.com"},
		{"email": "kdericut4@example.com"},
		{"email": "kfellon6@example.com"},
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
