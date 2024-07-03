//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func Test_UserIdentities(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	c.SetWorkspaceIdentifiers([]string{"email"})

	storageDir, err := filepath.Abs("testdata/user_identities_test")
	if err != nil {
		t.Fatal(err)
	}
	fs1 := c.AddSourceFilesystem(storageDir)
	fs2 := c.AddSourceFilesystem(storageDir)

	action1 := c.AddAction(fs1, "Users", chichitester.ActionToSet{
		Name: "CSV 1",
		Path: "users1.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.Text()},
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		IdentityProperty: "identity",
		Connector:        "CSV",
		UIValues: chichitester.JSONEncodeUIValues(map[string]any{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})

	action2 := c.AddAction(fs2, "Users", chichitester.ActionToSet{
		Name: "CSV 2",
		Path: "users2.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.Text()},
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		IdentityProperty: "identity",
		Connector:        "CSV",
		UIValues: chichitester.JSONEncodeUIValues(map[string]any{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})

	c.ExecuteAction(fs1, action1, true)
	c.ExecuteAction(fs2, action2, true)

	c.WaitActionsToFinish(fs1)
	c.WaitActionsToFinish(fs2)

	users, _, count := c.Users([]string{"email"}, "", 0, 100)

	const expectedCount = 4
	if expectedCount != count {
		t.Fatalf("expecting %d users, got %d", expectedCount, count)
	}
	t.Logf("the APIs successfully returned %d users", count)

	var totalIdentities int

	for _, user := range users {

		identities, count := c.UserIdentities(user.ID, 0, 1000)

		if count != 1 && count != 2 {
			t.Fatalf("expecting 'count' to be 1 or 2, got %d", count)
		}

		for _, identity := range identities {

			if anonIds := identity.AnonymousIds; anonIds != nil {
				t.Fatalf("identity should have a nil 'AnonymousIds', got %v", anonIds)
			}

			t.Logf(
				"the APIs returned an identity for user with GID %s that has"+
					" action = %d, identity ID = %v and last change time = %q",
				user.ID, identity.Action, identity.ID, identity.LastChangeTime)

			var idPrefix string
			switch identity.Action {
			case action1:
				idPrefix = "users1_"
			case action2:
				idPrefix = "users2_"
			default:
				t.Fatalf("unexpected action %d", identity.Action)
			}

			// Check the identity ID label.
			if !strings.HasPrefix(identity.ID, idPrefix) {
				t.Fatalf("unexpected identity ID %q, it should have prefix %q", identity.ID, idPrefix)
			}

			totalIdentities++
		}
	}

	const expectedTotalIdentities = 6
	if expectedTotalIdentities != totalIdentities {
		t.Fatalf("expecting a total of %d identities, got %d", expectedTotalIdentities, totalIdentities)
	}
	t.Logf("there is a total of %d identities", totalIdentities)

	// Additional test: test that a call to '/identities' for an user which does not exist
	// returns a NotFound error.
	{
		url := "/api/workspaces/1/users/7682c2a8-d85d-458b-9bd8-dc57cc12575a/identities"
		err := c.Call("GET", url, nil, nil)
		if err == nil {
			t.Fatalf("expecting error, got nothing")
		}
		errorMsg := err.Error()
		const expectedErr = `unexpected HTTP status code 404: {"error":{"code":"NotFound","message":"user 7682c2a8-d85d-458b-9bd8-dc57cc12575a does not exist"}}`
		if expectedErr != errorMsg {
			t.Fatalf("expected error %q, got %q", expectedErr, errorMsg)
		}
	}

}
