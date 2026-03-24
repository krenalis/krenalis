// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

func TestImportWithTransformation(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.Start()
	defer c.Stop()

	// Create a Dummy (source) connection.
	dummyID := c.CreateDummy("Dummy (source)", krenalistester.Source)

	c.UpdateIdentityResolution(false, []string{"email"})

	// Create a pipeline with a transformation function which imports users, then run it.
	importUsersID := c.CreatePipeline(dummyID, "User", krenalistester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
			{Name: "firstName", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "gender", Type: types.String(), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Function: &krenalistester.TransformationFunction{
				Language: "Python",
				Source: `
def transform(user: dict) -> dict:
	if user["firstName"] == "Jerad":
		gender = "male"
	else:
		gender = "female"
	return {
		"email": user["email"],
		"first_name": user["firstName"],
		"gender": gender,
	}`,
				InPaths:  []string{"firstName", "email"},
				OutPaths: []string{"email", "first_name", "gender"},
			},
		},
	})
	run := c.RunPipeline(importUsersID)
	c.WaitRunsCompletion(dummyID, run)

	c.RunIdentityResolution()

	// Retrieve the profiles.
	const expectedTotal = 10
	profiles, _, total := c.Profiles([]string{"email", "first_name", "gender"}, "email", false, 0, expectedTotal)

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
	expectedProfiles := []map[string]any{
		{"email": "abenois2@example.com", "first_name": "Ariela", "gender": "female"},
		{"email": "bdroghan5@example.com", "first_name": "Bryon", "gender": "female"},
		{"email": "ctroy7@example.com", "first_name": "Codie", "gender": "female"},
		{"email": "cveschambes3@example.com", "first_name": "Conroy", "gender": "female"},
		{"email": "gclother1@example.com", "first_name": "Glyn", "gender": "female"},
		{"email": "jdebrett9@example.com", "first_name": "Jerad", "gender": "male"},
		{"email": "jsharpin8@example.com", "first_name": "Janifer", "gender": "female"},
		{"email": "kbuessen0@example.com", "first_name": "Kinsley", "gender": "female"},
		{"email": "kdericut4@example.com", "first_name": "Kingsly", "gender": "female"},
		{"email": "kfellon6@example.com", "first_name": "Katine", "gender": "female"},
	}
	if len(expectedProfiles) != len(profiles) {
		t.Fatalf("expected %d profiles, got %d", len(expectedProfiles), len(profiles))
	}
	for i, profile := range profiles {
		expected := expectedProfiles[i]
		if !reflect.DeepEqual(expected, profile.Attributes) {
			t.Fatalf("expected %#v, got %#v", expected, profile)
		}
	}

}
