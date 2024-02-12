//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"chichi/connector/types"
	"chichi/test/chichitester"

	"github.com/segmentio/analytics-go/v3"
)

func Test_ImportFromManyConnections(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	ctx := context.Background()

	// Import users from Dummy.
	t.Log("importing from Dummy...")
	var dummy int
	{

		dummy = c.AddDummyWithBusinessID("Dummy", chichitester.Source, chichitester.BusinessID{Name: "email", Label: "Dummy email"})
		dummyAction := c.AddAction(dummy, "Users", chichitester.ActionToSet{
			Name: "Import users from Dummy",
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "firstName", Type: types.Text()},
				{Name: "lastName", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "firstName", Type: types.Text(), Nullable: true},
				{Name: "lastName", Type: types.Text(), Nullable: true},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email":     "email",
					"firstName": "firstName",
					"lastName":  "lastName",
				},
			},
		})
		c.ExecuteAction(dummy, dummyAction, true)
		c.WaitActionsToFinish(dummy)
	}

	// Ensure that there are 10 users.
	_, _, count := c.Users([]string{"email"}, "", 0, 1000)
	if count != 10 {
		t.Fatalf("expected 10 users, got %d", count)
	}

	// Imports users from CSV.
	var csv int
	t.Log("importing from CSV file...")
	{
		// Determine the storage directory and assert that such directory exists.
		storageDir, err := filepath.Abs("testdata/import_from_many_connections_test")
		if err != nil {
			t.Fatal(err)
		}
		stat, err := os.Stat(storageDir)
		if err != nil {
			t.Fatal(err)
		}
		if !stat.IsDir() {
			t.Fatalf("%q is not a dir", storageDir)
		}
		fs := c.AddSourceFilesystem(storageDir)
		csv = c.AddSourceCSVWithBusinessID(fs, chichitester.BusinessID{Name: "email", Label: "CSV email"})
		csvAction := c.AddAction(csv, "Users", chichitester.ActionToSet{
			Name: "Import users from CSV on Filesystem",
			Path: "users_genders.csv",
			InSchema: types.Object([]types.Property{
				{Name: "csv_id", Type: types.Text()},
				{Name: "email", Type: types.Text()},
				{Name: "gender", Type: types.Text()},
				{Name: "timestamp", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Nullable: true},
				{Name: "gender", Type: types.Text().WithValues("male", "female", "other"), Nullable: true},
			}),
			Transformation: chichitester.Transformation{
				Mapping: map[string]string{
					"email":  "email",
					"gender": "gender",
				},
			},
			IdentityColumn:  "csv_id",
			TimestampColumn: "timestamp",
			TimestampFormat: "'%Y-%m-%d %H:%M:%S'",
		})
		c.ExecuteAction(csv, csvAction, true)
		c.WaitActionsToFinish(csv)
	}

	// Ensure that there are 13 users (10 from Dummy + 3 from CSV).
	_, _, count = c.Users([]string{"email"}, "", 0, 1000)
	if count != 13 {
		t.Fatalf("expected 13 users, got %d", count)
	}

	// Import users and events from a JavaScript connection.
	var javaScript int
	t.Log("importing users and events...")
	{
		// Add a JavaScript connection with two actions (one for importing
		// events, one for importing user traits) and retrieve its key.
		var javaScriptKey string
		{
			javaScript = c.AddJavaScriptSourceWithBusinessID("JavaScript (source)", "example.com", chichitester.BusinessID{Name: "email", Label: "JavaScript email"})
			keys := c.ConnectionKeys(javaScript)
			if len(keys) != 1 {
				t.Fatalf("expecting one key, got %d keys", len(keys))
			}
			javaScriptKey = keys[0]
			c.AddAction(javaScript, "Events", chichitester.ActionToSet{
				Name:    "JavaScript",
				Enabled: true,
			})
			c.AddAction(javaScript, "Users", chichitester.ActionToSet{
				Name:     "JavaScript",
				Enabled:  true,
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text(), Nullable: true},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "traits.email",
					},
				},
			})
		}

		// Send an identity event. More than importing an event, this should create
		// a user identity.
		c.SendEvent(javaScriptKey, analytics.Identify{
			UserId:      "f4ca124298",
			AnonymousId: "5ce0fd49-199a-47e7-b0c8-498f5144f0ee",
			Traits: map[string]interface{}{
				"email": "kbuessen0@example.com",
			},
		})
		c.WaitEventsStoredIntoWarehouse(ctx, 1)
	}

	// Ensure that there are 14 users (10 from Dummy + 3 from CSV + 1 from event).
	_, _, count = c.Users([]string{"email"}, "", 0, 1000)
	if count != 14 {
		t.Fatalf("expected 14 users, got %d", count)
	}

	// Set the "email" as identifier and run the Workspace Identity Resolution.
	c.SetWorkspaceIdentifiers([]string{"email"})
	c.RunWorkspaceIdentityResolution()

	// Ensure that there are 10 users.
	users, _, count := c.Users([]string{"Id", "email"}, "", 0, 1000)
	if count != 10 {
		t.Fatalf("expected 10 users, got %d", count)
	}

	// Retrieve the GID of "kbuessen0@example.com".
	var kBuessenGid int
	for _, user := range users {
		if user["email"] == "kbuessen0@example.com" {
			gid, err := user["Id"].(json.Number).Int64()
			if err != nil {
				t.Fatal(err)
			}
			kBuessenGid = int(gid)
			break
		}
	}
	if kBuessenGid == 0 {
		t.Fatalf("user with email %q not found", "kbuessen0@example.com")
	}

	// Ensure that "kbuessen0@example.com" has one event associated.
	events := c.UserEvents(kBuessenGid)
	if len(events) != 1 {
		t.Fatalf("expecting %q to have one event associated, got %d", "kbuessen0@example.com", len(events))
	}

	// Validate the identities.
	identities, count := c.UserIdentities(kBuessenGid, 0, 1000)
	if count != 3 {
		t.Fatalf("expecting user %d to have 3 identities associated, got %d", kBuessenGid, count)
	}
	assertEqualIdentity := func(got, expected chichitester.UserIdentity) {
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expecting identity %#v, got %#v", expected, got)
		}
	}
	{
		dummyIdentity := identities[0]
		dummyIdentity.Timestamp = time.Time{}
		assertEqualIdentity(dummyIdentity, chichitester.UserIdentity{
			Connection:   dummy,
			ExternalId:   chichitester.LabelValue{Label: "Dummy Unique ID", Value: "dummy1"},
			BusinessId:   chichitester.LabelValue{Label: "Dummy email", Value: "kbuessen0@example.com"},
			AnonymousIds: nil,
			Timestamp:    time.Time{},
		})
	}
	t.Log("identity imported from Dummy is ok")
	{
		csvIdentity := identities[1]
		assertEqualIdentity(csvIdentity, chichitester.UserIdentity{
			Connection: csv,
			ExternalId: chichitester.LabelValue{
				Label: "ID",
				Value: "1",
			},
			BusinessId:   chichitester.LabelValue{Label: "CSV email", Value: "kbuessen0@example.com"},
			AnonymousIds: nil,
			Timestamp:    time.Date(2001, 2, 2, 3, 4, 5, 0, time.UTC),
		})
	}
	t.Log("identity imported from CSV is ok")
	{
		eventIdentity := identities[2]
		eventIdentity.Timestamp = time.Time{}
		assertEqualIdentity(eventIdentity, chichitester.UserIdentity{
			Connection:   javaScript,
			ExternalId:   chichitester.LabelValue{Label: "User ID", Value: "f4ca124298"},
			BusinessId:   chichitester.LabelValue{Label: "JavaScript email", Value: "kbuessen0@example.com"},
			AnonymousIds: []string{"5ce0fd49-199a-47e7-b0c8-498f5144f0ee"},
			Timestamp:    time.Time{},
		})
	}
	t.Log("identity imported from JavaScript is ok")

}
