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

func TestDummyImportNotRequired(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	// Import users from Dummy.
	dummySrc := c.CreateDummy("Dummy (source)", krenalistester.Source)
	importUsersID := c.CreatePipeline(dummySrc, "User", krenalistester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
			{Name: "favourite_movie", Type: types.String(), ReadOptional: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "favorite_movie", Type: types.Object([]types.Property{
				{Name: "title", Type: types.String(), ReadOptional: true},
			}), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"favorite_movie.title": "favourite_movie",
				"email":                "email",
			},
		},
	})
	run := c.RunPipeline(importUsersID)
	c.WaitRunsCompletion(dummySrc, run)

	// Test that the "favorite_movie.title" property, which has been imported
	// from a not required property in Dummy, has been imported just for some
	// profiles.
	profiles, _, total := c.Profiles([]string{"email", "favorite_movie"}, "email", false, 0, 100)
	if total != 10 {
		t.Fatalf("expected 10 profiles, got %d instead", total)
	}
	expectedAttributes := []map[string]any{
		{"email": "abenois2@example.com", "favorite_movie": map[string]any{"title": "Eclipse Protocol"}},
		{"email": "bdroghan5@example.com"},
		{"email": "ctroy7@example.com", "favorite_movie": map[string]any{"title": "Phantom Avenue"}},
		{"email": "cveschambes3@example.com", "favorite_movie": map[string]any{"title": "Forgotten Kingdoms"}},
		{"email": "gclother1@example.com"},
		{"email": "jdebrett9@example.com"},
		{"email": "jsharpin8@example.com"},
		{"email": "kbuessen0@example.com", "favorite_movie": map[string]any{"title": "Eclipse Protocol"}},
		{"email": "kdericut4@example.com", "favorite_movie": map[string]any{"title": "The Quantum Conspiracy"}},
		{"email": "kfellon6@example.com"},
	}
	for i := range 10 {
		got := profiles[i].Attributes
		expected := expectedAttributes[i]
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("profile n. %d: expected attributes %#v, got %#v", i+1, expected, got)
		}
	}

}
