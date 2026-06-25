// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

func Test_RemoveUsersWhenDeletingConnections(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	k.UpdateIdentityResolution(false, nil)

	// Create two Dummy connections for importing users.
	dummy1 := k.CreateDummy("Dummy 1", krenalistester.Source)
	dummy2 := k.CreateDummy("Dummy 2", krenalistester.Source)

	// Create two identical pipelines for two different connections.
	pipelineParams := krenalistester.PipelineToSet{
		Enabled: true,
		Name:    "Import users from Dummy",
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

	// Import from both pipelines, then run the identity resolution.
	run1 := k.RunPipeline(pipeline1)
	run2 := k.RunPipeline(pipeline2)
	k.WaitRunsCompletion(dummy1, run1)
	k.WaitRunsCompletion(dummy2, run2)
	k.RunIdentityResolution()

	// Now there should be total of 20 profiles.
	_, _, total := k.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 20 {
		t.Fatalf("expected 20 profiles, got %d", total)
	}

	// Delete one Dummy, wait for the identities to be purged, resolve
	// identities, and ensure that only 10 profiles remain.
	k.DeleteConnection(dummy1)
	time.Sleep(1 * time.Second)
	k.RunIdentityResolution()
	_, _, total = k.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 10 {
		t.Fatalf("expected 10 profiles, got %d", total)
	}

	// Delete also the other Dummy connection; now the total number of profiles
	// should be zero.
	k.DeleteConnection(dummy2)
	time.Sleep(1 * time.Second)
	k.RunIdentityResolution()
	_, _, total = k.Profiles([]string{"email"}, "", false, 0, 100)
	if total != 0 {
		t.Fatalf("expected no profiles, got %d", total)
	}

}
