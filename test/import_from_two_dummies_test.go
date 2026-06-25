// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

func TestImportFromTwoDummies(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	// Create two Dummy connections for importing users.
	dummy1 := k.CreateDummy("Dummy 1", krenalistester.Source)
	dummy2 := k.CreateDummy("Dummy 2", krenalistester.Source)

	// Create two identical pipelines for two different connections.
	pipelineParams := krenalistester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
			{Name: "firstName", Type: types.String(), Nullable: true},
			{Name: "lastName", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"email":      "email",
				"first_name": "firstName",
				"last_name":  "lastName",
			},
		},
	}
	pipeline1 := k.CreatePipeline(dummy1, "User", pipelineParams)
	pipeline2 := k.CreatePipeline(dummy2, "User", pipelineParams)

	// Import from both pipelines - and implicitly trigger the identity resolution
	// process.
	run1 := k.RunPipeline(pipeline1)
	run2 := k.RunPipeline(pipeline2)
	k.WaitRunsCompletion(dummy1, run1)
	k.WaitRunsCompletion(dummy2, run2)

	// Ensure that the connection have the correct identities associated.
	{
		identities, total := k.ConnectionIdentities(dummy1, 0, 100)
		if total != 10 {
			t.Fatalf("expected total 10, got %d", total)
		}
		for _, identity := range identities {
			if identity.Pipeline != pipeline1 {
				t.Fatalf("expected pipeline %s, got %s, ", pipeline1, identity.Pipeline)
			}
		}
		identities, total = k.ConnectionIdentities(dummy2, 0, 100)
		if total != 10 {
			t.Fatalf("expected total 10, got %d", total)
		}
		for _, identity := range identities {
			if identity.Pipeline != pipeline2 {
				t.Fatalf("expected pipeline %s, got %s", pipeline2, identity.Pipeline)
			}
		}
	}

	// Since the profiles have been imported from two different connections without
	// any identity resolution identifier configured, there should be a total of
	// 20 profiles, even if they have the same properties.
	profiles, _, total := k.Profiles([]string{"email", "first_name", "last_name"}, "", false, 0, 100)
	expectedTotal := 20
	if expectedTotal != total {
		t.Fatalf("expected total %d, got %d", expectedTotal, total)
	}

	// Every profile now should have just one identity associated.
	totalProfiles := 0
	for _, profile := range profiles {
		_, total := k.Identities(profile.KPID, 0, 100)
		const expectedTotal = 1
		if expectedTotal != total {
			t.Fatalf("expected %d identities for profile %s, got %d", total, profile.KPID, total)
		}
		totalProfiles++
	}
	if expectedTotal != totalProfiles { // ensure that the number of profiles matches with the returned 'total' value.
		t.Fatalf("expected %d profiles returned, got %d", expectedTotal, totalProfiles)
	}

	// Update the workspace identifiers and run the Identity Resolution.
	k.UpdateIdentityResolution(true, []string{"email"})
	k.RunIdentityResolutionAndWait()

	// Now the profiles should be merged, resulting in a total of 10 profiles.
	profiles, _, total = k.Profiles([]string{"email", "first_name", "last_name"}, "", false, 0, 100)
	expectedTotal = 10
	if expectedTotal != total {
		t.Fatalf("expected total %d, got %d", expectedTotal, total)
	}

	// Every profile now should have two identities associated.
	totalProfiles = 0
	for _, profile := range profiles {
		_, total := k.Identities(profile.KPID, 0, 100)
		const expectedTotal = 2
		if expectedTotal != total {
			t.Fatalf("expected %d identities for profile %s, got %d", total, profile.KPID, total)
		}
		totalProfiles++
	}
	if expectedTotal != totalProfiles { // ensure that the total number of profiles matches with the returned 'total' value.
		t.Fatalf("expected %d profiles returned, got %d", expectedTotal, totalProfiles)
	}

}
