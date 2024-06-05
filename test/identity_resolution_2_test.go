//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

// TestIdentityResolution2 tests the behavior of Identity Resolution for arrays
// and primary connections.
func TestIdentityResolution2(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t, chichitester.DoNotPopulateUserSchema)
	defer c.Stop()

	// Add properties to the user schema.
	schema := types.Object([]types.Property{
		{Name: "email", Type: types.Text().WithCharLen(300), Nullable: true},
		{Name: "name", Type: types.Text(), Nullable: true},
		{Name: "phone_numbers", Type: types.Array(types.Text()), Nullable: true},
		{Name: "total_orders", Type: types.Int(32), Nullable: true},
	})
	c.ChangeUserSchema(schema, nil, nil)

	// Set the email as the only identifier, as the 3 identities, imported from
	// the 3 connections, will all be put together in a single user as they
	// share the same email.
	c.SetWorkspaceIdentifiers([]string{"email"})

	storage := chichitester.NewTempStorage(t)

	sourceA := c.AddSourceFilesystem(storage.Root())
	sourceB := c.AddSourceFilesystem(storage.Root())
	sourceC := c.AddSourceFilesystem(storage.Root())

	// Create three JSON files in storage, one for each connection. Each file
	// will contain a single user, which will be the only identity imported for
	// each connection.

	writeUser := func(filename string, user map[string]any) {
		content, err := json.Marshal([]any{user})
		if err != nil {
			t.Fatal(err)
		}
		absPath := filepath.Join(storage.Root(), filename)
		err = os.WriteFile(absPath, content, 0755)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("file %q written", absPath)
	}

	writeUser("A.json", map[string]any{
		"email":            "a@b",
		"name":             "John",
		"phone_numbers":    []string{"+11 111"},
		"total_orders":     10,
		"last_change_time": "2000-01-01 12:00:00",
	})

	writeUser("B.json", map[string]any{
		"email":            "a@b",
		"name":             nil,
		"phone_numbers":    []string{"+22 222", "+33 333"},
		"total_orders":     20,
		"last_change_time": "2000-01-02 12:00:00",
	})

	writeUser("C.json", map[string]any{
		"email":            "a@b",
		"name":             nil,
		"phone_numbers":    nil,
		"total_orders":     21,
		"last_change_time": "2000-01-03 12:00:00",
	})

	// Create and execute the actions.

	mapping := map[string]string{
		"email":         "email",
		"name":          "name",
		"phone_numbers": "phone_numbers",
		"total_orders":  "total_orders",
	}
	addJSONAction := func(source int, filename string) int {
		return c.AddAction(source, "Users", chichitester.ActionToSet{
			Name: "Action",
			Path: filename,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.JSON(), Nullable: true},
				{Name: "name", Type: types.JSON(), Nullable: true},
				{Name: "phone_numbers", Type: types.JSON(), Nullable: true},
				{Name: "total_orders", Type: types.JSON(), Nullable: true},
				{Name: "last_change_time", Type: types.JSON(), Nullable: true},
			}),
			OutSchema: schema,
			Transformation: chichitester.Transformation{
				Mapping: mapping,
			},
			IdentityProperty:       "email",
			LastChangeTimeProperty: "last_change_time",
			LastChangeTimeFormat:   "DateTime",
			Connector:              "JSON",
			UIValues:               []byte("{}"),
		})
	}

	actionA := addJSONAction(sourceA, "A.json")
	actionB := addJSONAction(sourceB, "B.json")
	actionC := addJSONAction(sourceC, "C.json")

	c.ExecuteAction(sourceA, actionA, false)
	c.ExecuteAction(sourceB, actionB, false)
	c.ExecuteAction(sourceC, actionC, false)
	c.WaitActionsToFinish(sourceA)
	c.WaitActionsToFinish(sourceB)
	c.WaitActionsToFinish(sourceC)

	// Explicitly run the Identity Resolution, even if it has been executed at
	// the end of the import action executions.
	c.RunIdentityResolution()

	// Check that there is only one user, and that its properties have been
	// merged correctly.

	users, _, count := c.Users(schema.PropertiesNames(), "", 0, 100)
	if count != 1 {
		t.Fatalf("expected just 1 user (which is the merge of the 3 identities), got %d instead", count)
	}
	user := users[0]
	expectedUser := map[string]any{
		"email":         "a@b",
		"name":          "John",
		"phone_numbers": []any{"+11 111", "+22 222", "+33 333"},
		"total_orders":  json.Number("21"),
	}
	if !reflect.DeepEqual(user, expectedUser) {
		t.Fatalf("expected user %#v, got %#v", expectedUser, user)
	}

	// Change the primary sources, making the "total_orders" property have
	// connection B as its primary source. This should change the value for that
	// property, as that value would no longer be taken from the incoming
	// identity from C (which was the most up-to-date one) but instead would be
	// taken from the incoming identity from B, which has a value of 20 instead
	// of 21.

	primarySources := map[string]int{
		"total_orders": sourceB,
	}
	c.ChangeUserSchema(schema, primarySources, nil)

	c.RunIdentityResolution()

	users, _, count = c.Users(schema.PropertiesNames(), "", 0, 100)
	if count != 1 {
		t.Fatalf("expected just 1 user (which is the merge of the 3 identities), got %d instead", count)
	}
	user = users[0]
	expectedUser = map[string]any{
		"email":         "a@b",
		"name":          "John",
		"phone_numbers": []any{"+11 111", "+22 222", "+33 333"},
		"total_orders":  json.Number("20"),
	}
	if !reflect.DeepEqual(user, expectedUser) {
		t.Fatalf("expected user %#v, got %#v", expectedUser, user)
	}

	storage.Remove()
}
