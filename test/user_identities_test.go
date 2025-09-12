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

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

func Test_UserIdentities(t *testing.T) {

	// Determine the storage directory.
	storageDir, err := filepath.Abs("testdata/user_identities_test")
	if err != nil {
		t.Fatal(err)
	}

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.SetFilesystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	c.UpdateIdentityResolution(true, []string{"email"})

	fs1 := c.CreateSourceFilesystem()
	fs2 := c.CreateSourceFilesystem()

	action1 := c.CreateAction(fs1, "User", meergotester.ActionToSet{
		Name:    "CSV 1",
		Enabled: true,
		Path:    "users1.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.Text()},
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		IdentityColumn: "identity",
		Format:         "CSV",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
			"Separator":      ",",
			"HasColumnNames": true,
		}),
	})

	action2 := c.CreateAction(fs2, "User", meergotester.ActionToSet{
		Name:    "CSV 2",
		Enabled: true,
		Path:    "users2.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.Text()},
			{Name: "email", Type: types.Text()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		IdentityColumn: "identity",
		Format:         "CSV",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
			"Separator":      ",",
			"HasColumnNames": true,
		}),
	})

	exec1 := c.ExecuteAction(action1)
	exec2 := c.ExecuteAction(action2)

	c.WaitForExecutionsCompletion(fs1, exec1)
	c.WaitForExecutionsCompletion(fs2, exec2)

	users, _, total := c.Users([]string{"email"}, "", false, 0, 100)

	const expectedTotal = 4
	if expectedTotal != total {
		t.Fatalf("expected %d users, got %d", expectedTotal, total)
	}
	t.Logf("the APIs successfully returned %d users", total)

	var totalIdentities int

	for _, user := range users {

		identities, total := c.UserIdentities(user.ID, 0, 1000)

		if total != 1 && total != 2 {
			t.Fatalf("expected 'total' to be 1 or 2, got %d", total)
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
		t.Fatalf("expected a total of %d identities, got %d", expectedTotalIdentities, totalIdentities)
	}
	t.Logf("there is a total of %d identities", totalIdentities)

	// Additional test: test that a call to '/identities' for an user which does not exist
	// returns a NotFound error.
	{
		err := c.Call("GET", "/api/v1/users/7682c2a8-d85d-458b-9bd8-dc57cc12575a/identities", nil, nil)
		if err == nil {
			t.Fatalf("expected error, got nothing")
		}
		errorMsg := err.Error()
		const expectedErr = `unexpected HTTP status code 404: {"error":{"code":"NotFound","message":"user \"7682c2a8-d85d-458b-9bd8-dc57cc12575a\" does not exist"}}`
		if expectedErr != errorMsg {
			t.Fatalf("expected error %q, got %q", expectedErr, errorMsg)
		}
	}

}
