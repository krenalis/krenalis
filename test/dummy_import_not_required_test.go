//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestDummyImportNotRequired(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.Start()
	defer c.Stop()

	// Import users from Dummy.
	dummySrc := c.CreateDummy("Dummy (source)", meergotester.Source)
	importUsersID := c.CreateAction(dummySrc, "User", meergotester.ActionToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "favourite_movie", Type: types.Text(), ReadOptional: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
			{Name: "favorite_movie", Type: types.Object([]types.Property{
				{Name: "title", Type: types.Text(), ReadOptional: true},
			}), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"favorite_movie.title": "favourite_movie",
				"email":                "email",
			},
		},
	})
	exec := c.ExecuteAction(importUsersID)
	c.WaitForExecutionsCompletion(dummySrc, exec)

	// Test that the "favorite_movie.title" property, which has been imported
	// from a not required property in Dummy, has been imported just for some
	// users.
	users, _, total := c.Users([]string{"email", "favorite_movie"}, "email", false, 0, 100)
	if total != 10 {
		t.Fatalf("expected 10 users, got %d instead", total)
	}
	expectedUserTraits := []map[string]any{
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
		got := users[i].Traits
		expected := expectedUserTraits[i]
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("user n. %d: expected traits %#v, got %#v", i+1, expected, got)
		}
	}

}
