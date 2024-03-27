//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func Test_UsersIdentities(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	c.SetWorkspaceIdentifiers([]string{"email"})

	storageDir, err := filepath.Abs("testdata/users_identities_test")
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
			{Name: "email", Type: types.Text(), Nullable: true},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		IdentityColumn: "identity",
		BusinessID:     "email",
		Connector:      chichitester.CSVConnector,
		Settings: chichitester.JSONEncodeSettings(map[string]any{
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
			{Name: "email", Type: types.Text(), Nullable: true},
		}),
		Transformation: chichitester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		IdentityColumn: "identity",
		BusinessID:     "email",
		Connector:      chichitester.CSVConnector,
		Settings: chichitester.JSONEncodeSettings(map[string]any{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})

	c.ExecuteAction(fs1, action1, false)
	c.ExecuteAction(fs2, action2, false)

	c.WaitActionsToFinish(fs1)
	c.WaitActionsToFinish(fs2)

	users, _, count := c.Users([]string{"Id"}, "", 0, 100)

	const expectedCount = 4
	if expectedCount != count {
		t.Fatalf("expecting %d users, got %d", expectedCount, count)
	}
	t.Logf("the APIs successfully returned %d users", count)

	var totalIdentities int

	for _, user := range users {

		id, _ := user["Id"].(json.Number).Int64()

		identities, count := c.UserIdentities(int(id), 0, 1000)

		if count != 1 && count != 2 {
			t.Fatalf("expecting 'count' to be 1 or 2, got %d", count)
		}

		for _, identity := range identities {

			if anonIds := identity.AnonymousIds; anonIds != nil {
				t.Fatalf("identity should have a nil 'AnonymousIds', got %v", anonIds)
			}

			t.Logf(
				"the APIs returned an identity for user with GID %d that has"+
					" connection = %d, external ID = %v and updated_at timestamp = %q",
				id, identity.Connection, identity.ExternalId, identity.UpdatedAt)

			var externalIDPrefix string
			switch identity.Connection {
			case fs1:
				externalIDPrefix = "users1_"
			case fs2:
				externalIDPrefix = "users2_"
			default:
				t.Fatalf("unexpected connection %d", identity.Connection)
			}

			// Check the External ID label.
			extID := identity.ExternalId
			const expectedExternalIDLabel = "ID"
			if expectedExternalIDLabel != extID.Label {
				t.Fatalf("expected External ID label %q, got %q", expectedExternalIDLabel, extID.Label)
			}
			if !strings.HasPrefix(extID.Value, externalIDPrefix) {
				t.Fatalf("unexpected external ID %q, it should have prefix %q", extID, externalIDPrefix)
			}

			// Check the Business ID.
			if !strings.Contains(identity.BusinessId, "@") {
				t.Fatalf("expecting Business ID value with a '@', got %q", identity.BusinessId)
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
		url := "/api/workspaces/1/users/12345/identities"
		req := map[string]any{
			"First": 0,
			"Limit": 100,
		}
		err := c.Call("POST", url, req, nil)
		if err == nil {
			t.Fatalf("expecting error, got nothing")
		}
		errorMsg := err.Error()
		const expectedErr = `unexpected HTTP status code 404: {"error":{"code":"NotFound","message":"user 12345 does not exist"}}`
		if expectedErr != errorMsg {
			t.Fatalf("expected error %q, got %q", expectedErr, errorMsg)
		}
	}

}
