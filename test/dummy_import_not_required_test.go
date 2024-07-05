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

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestDummyImportNotRequired(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Import users from Dummy.
	dummySrc := c.AddDummy("Dummy (source)", chichitester.Source)
	importUsersID := c.AddAction(dummySrc, "Users", chichitester.ActionToSet{
		Name: "Import users from Dummy",
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), Required: true},
			{Name: "favourite_movie", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "favorite_movie", Type: types.Object([]types.Property{
				{Name: "title", Type: types.Text()},
			})},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"favorite_movie.title": "favourite_movie",
				"email":                "email",
			},
		},
	})
	c.ExecuteAction(dummySrc, importUsersID, true)
	c.WaitActionsToFinish(dummySrc)

	// Test that the "favorite_movie.title" property, which has been imported
	// from a not required property in Dummy, has been imported just for some
	// users.
	users, _, count := c.Users([]string{"email", "favorite_movie"}, "email", 0, 100)
	if count != 10 {
		t.Fatalf("expected 10 users, got %d instead", count)
	}
	expectedUserProps := []map[string]any{
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
		gotProps := users[i].Properties
		expectedProps := expectedUserProps[i]
		if !reflect.DeepEqual(gotProps, expectedProps) {
			t.Fatalf("user n. %d: expected properties %#v, got %#v", i+1, expectedProps, gotProps)
		}
	}

}
