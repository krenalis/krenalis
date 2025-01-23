//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo/test/analytics-go"
	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

func Test_ImportFromManyConnections(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	ctx := context.Background()

	// Import users from Dummy.
	t.Log("importing from Dummy...")
	var dummy, dummyAction int
	{

		dummy = c.CreateDummy("Dummy", meergotester.Source)
		dummyAction = c.CreateAction(dummy, "Users", meergotester.ActionToSet{
			Name:    "Import users from Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "firstName", Type: types.Text()},
				{Name: "lastName", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "first_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "last_name", Type: types.Text().WithCharLen(300), ReadOptional: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"email":      "email",
					"first_name": "firstName",
					"last_name":  "lastName",
				},
			},
		})
		exec := c.ExecuteAction(dummyAction, true)
		c.WaitForExecutionsCompletion(dummy, exec)
	}

	// Ensure that there are 10 users.
	_, _, total := c.Users([]string{"email"}, "", false, 0, 1000)
	if total != 10 {
		t.Fatalf("expected 10 users, got %d", total)
	}

	// Imports users from CSV.
	var fs, csvAction int
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
		fs = c.CreateSourceFilesystem(storageDir)
		csvAction = c.CreateAction(fs, "Users", meergotester.ActionToSet{
			Name:    "Import users from CSV on Filesystem",
			Enabled: true,
			Path:    "users_genders.csv",
			InSchema: types.Object([]types.Property{
				{Name: "csv_id", Type: types.Text()},
				{Name: "email", Type: types.Text()},
				{Name: "gender", Type: types.Text()},
				{Name: "timestamp", Type: types.Text()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
				{Name: "gender", Type: types.Text(), ReadOptional: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"email":  "email",
					"gender": "gender",
				},
			},
			IdentityProperty:       "csv_id",
			LastChangeTimeProperty: "timestamp",
			LastChangeTimeFormat:   "%Y-%m-%d %H:%M:%S",
			Format:                 "CSV",
			FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
				"Comma":          ",",
				"HasColumnNames": true,
			}),
		})
		exec := c.ExecuteAction(csvAction, true)
		c.WaitForExecutionsCompletion(fs, exec)
	}

	// Ensure that there are 13 users (10 from Dummy + 3 from CSV).
	_, _, total = c.Users([]string{"email"}, "", false, 0, 1000)
	if total != 13 {
		t.Fatalf("expected 13 users, got %d", total)
	}

	// Import users and events from a JavaScript connection.
	var javaScript, javascriptUsersAction int
	t.Log("importing users and events...")
	{
		// Create a JavaScript connection with two actions (one for importing
		// events, one for importing user identities) and retrieve its key.
		var javaScriptKey string
		{
			javaScript = c.CreateJavaScriptSource("JavaScript (source)", "example.com", nil)
			keys := c.EventWriteKeys(javaScript)
			if len(keys) != 1 {
				t.Fatalf("expected one key, got %d keys", len(keys))
			}
			javaScriptKey = keys[0]
			c.CreateAction(javaScript, "Events", meergotester.ActionToSet{
				Name:    "JavaScript",
				Enabled: true,
			})
			javascriptUsersAction = c.CreateAction(javaScript, "Users", meergotester.ActionToSet{
				Name:     "JavaScript",
				Enabled:  true,
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text().WithCharLen(300), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
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
		time.Sleep(time.Second)
		c.StartIdentityResolution()
		c.WaitEventsStoredIntoWarehouse(ctx, 1)
	}

	// Ensure that there are 14 users (10 from Dummy + 3 from CSV + 1 from event).
	_, _, total = c.Users([]string{"email"}, "", false, 0, 1000)
	if total != 14 {
		t.Fatalf("expected 14 users, got %d", total)
	}

	// Set the "email" as identifier and run the Identity Resolution.
	c.UpdateIdentityResolution(true, []string{"email"})
	c.StartIdentityResolution()

	// Ensure that there are 10 users.
	users, _, total := c.Users([]string{"email"}, "", false, 0, 1000)
	if total != 10 {
		t.Fatalf("expected 10 users, got %d", total)
	}

	// Retrieve the GID of "kbuessen0@example.com".
	var kBuessenGid uuid.UUID
	for _, user := range users {
		if user.Traits["email"] == "kbuessen0@example.com" {
			kBuessenGid = user.ID
			break
		}
	}
	if kBuessenGid == (uuid.UUID{}) {
		t.Fatalf("user with email %q not found", "kbuessen0@example.com")
	}

	// Ensure that "kbuessen0@example.com" has one event associated.
	events := c.UserEvents(kBuessenGid, []string{"timestamp"})
	if len(events) != 1 {
		t.Fatalf("expected %q to have one event associated, got %d", "kbuessen0@example.com", len(events))
	}

	// Validate the identities.
	identities, total := c.UserIdentities(kBuessenGid, 0, 1000)
	if total != 3 {
		t.Fatalf("expected user %s to have 3 identities associated, got %d", kBuessenGid, total)
	}
	assertEqualIdentity := func(got, expected meergotester.UserIdentity) {
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected identity %#v, got %#v", expected, got)
		}
	}
	{
		dummyIdentity := identities[0]
		dummyIdentity.LastChangeTime = time.Time{}
		assertEqualIdentity(dummyIdentity, meergotester.UserIdentity{
			Connection:     dummy, // TODO(Gianluca): remove when the Connection field is removed from the UserIdentity.
			Action:         dummyAction,
			ID:             "dummy1",
			AnonymousIds:   nil,
			LastChangeTime: time.Time{},
		})
	}
	t.Log("identity imported from Dummy is ok")
	{
		csvIdentity := identities[1]
		assertEqualIdentity(csvIdentity, meergotester.UserIdentity{
			Connection:     fs, // TODO(Gianluca): remove when the Connection field is removed from the UserIdentity.
			Action:         csvAction,
			ID:             "1",
			AnonymousIds:   nil,
			LastChangeTime: time.Date(2001, 2, 2, 3, 4, 5, 0, time.UTC),
		})
	}
	t.Log("identity imported from CSV is ok")
	{
		eventIdentity := identities[2]
		eventIdentity.LastChangeTime = time.Time{}
		assertEqualIdentity(eventIdentity, meergotester.UserIdentity{
			Connection:     javaScript, // TODO(Gianluca): remove when the Connection field is removed from the UserIdentity.
			Action:         javascriptUsersAction,
			ID:             "f4ca124298",
			AnonymousIds:   []string{"5ce0fd49-199a-47e7-b0c8-498f5144f0ee"},
			LastChangeTime: time.Time{},
		})
	}
	t.Log("identity imported from JavaScript is ok")

}
