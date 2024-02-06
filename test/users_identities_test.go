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

	"chichi/connector/types"
	"chichi/test/chichitester"
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
	fs := c.AddSourceFilesystem(storageDir)

	csv1 := c.AddSourceCSV(fs)
	csv2 := c.AddSourceCSV(fs)

	action1 := c.AddAction(csv1, "Users", chichitester.ActionToSet{
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
	})

	action2 := c.AddAction(csv2, "Users", chichitester.ActionToSet{
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
	})

	c.ExecuteAction(csv1, action1, false)
	c.ExecuteAction(csv2, action2, false)

	c.WaitActionsToFinish(csv1)
	c.WaitActionsToFinish(csv2)

	usersResponse := c.Users([]string{"Id"}, "", 0, 100)

	count, _ := usersResponse["count"].(json.Number).Int64()
	const expectedCount = 4
	if expectedCount != count {
		t.Fatalf("expecting %d users, got %d", expectedCount, count)
	}
	t.Logf("the APIs successfully returned %d users", count)

	users := usersResponse["users"].([]any)

	var totalIdentities int

	for _, user := range users {

		id, _ := user.(map[string]any)["Id"].(json.Number).Int64()

		identities, count := c.UserIdentities(int(id), 0, 1000)

		if count != 1 && count != 2 {
			t.Fatalf("expecting 'count' to be 1 or 2, got %d", count)
		}

		for _, identity := range identities {

			connectionInt64, _ := identity["Connection"].(json.Number).Int64()
			connection := int(connectionInt64)

			externalID := identity["ExternalId"].(map[string]any)

			timestamp := identity["Timestamp"].(string)

			if _, ok := identity["AnonymousIds"]; !ok {
				t.Fatalf("identity should have an 'AnonymousIds' key")
			}
			if anonIds := identity["AnonymousIds"]; anonIds != nil {
				t.Fatalf("identity should have a nil 'AnonymousIds', got %v", anonIds)
			}

			t.Logf(
				"the APIs returned an identity for user with GID %d that has"+
					" connection = %d, external ID = %q and timestamp = %q",
				id, connection, externalID, timestamp)

			var externalIDPrefix string
			switch connection {
			case csv1:
				externalIDPrefix = "users1_"
			case csv2:
				externalIDPrefix = "users2_"
			default:
				t.Fatalf("unexpected connection %d", connection)
			}

			// Check the External ID label.
			const expectedExternalIDLabel = "ID"
			if expectedExternalIDLabel != externalID["Label"].(string) {
				t.Fatalf("expected External ID label %q, got %q", expectedExternalIDLabel, externalID["Label"].(string))
			}

			if !strings.HasPrefix(externalID["Value"].(string), externalIDPrefix) {
				t.Fatalf("unexpected external ID %q, it should have prefix %q", externalID, externalIDPrefix)
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
		_, err := c.Call("POST", url, req)
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
