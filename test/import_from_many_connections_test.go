// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"

	"github.com/google/uuid"
	"github.com/meergo/analytics-go"
)

func Test_ImportFromManyConnections(t *testing.T) {

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

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.SetFileSystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	ctx := context.Background()

	// Import users from Dummy.
	t.Log("importing from Dummy...")
	var dummy, dummyPipeline int
	{

		dummy = c.CreateDummy("Dummy", meergotester.Source)
		dummyPipeline = c.CreatePipeline(dummy, "User", meergotester.PipelineToSet{
			Name:    "Import users from Dummy",
			Enabled: true,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String(), Nullable: true},
				{Name: "firstName", Type: types.String(), Nullable: true},
				{Name: "lastName", Type: types.String(), Nullable: true},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "last_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"email":      "email",
					"first_name": "firstName",
					"last_name":  "lastName",
				},
			},
		})
		run := c.RunPipeline(dummyPipeline)
		c.WaitRunsCompletion(dummy, run)
	}

	// Ensure that there are 10 profiles.
	_, _, total := c.Profiles([]string{"email"}, "", false, 0, 1000)
	if total != 10 {
		t.Fatalf("expected 10 profiles, got %d", total)
	}

	// Imports users from CSV.
	var fs, csvPipeline int
	t.Log("importing from CSV file...")
	{
		fs = c.CreateSourceFileSystem()
		csvPipeline = c.CreatePipeline(fs, "User", meergotester.PipelineToSet{
			Name:    "Import users from CSV on File System",
			Enabled: true,
			Path:    "users_genders.csv",
			InSchema: types.Object([]types.Property{
				{Name: "csv_id", Type: types.String()},
				{Name: "email", Type: types.String()},
				{Name: "gender", Type: types.String()},
				{Name: "timestamp", Type: types.String()},
			}),
			OutSchema: types.Object([]types.Property{
				{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
				{Name: "gender", Type: types.String(), ReadOptional: true},
			}),
			Transformation: &meergotester.Transformation{
				Mapping: map[string]string{
					"email":  "email",
					"gender": "gender",
				},
			},
			IdentityColumn:       "csv_id",
			LastChangeTimeColumn: "timestamp",
			LastChangeTimeFormat: "%Y-%m-%d %H:%M:%S",
			Format:               "csv",
			FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
				"Separator":      ",",
				"HasColumnNames": true,
			}),
		})
		run := c.RunPipeline(csvPipeline)
		c.WaitRunsCompletion(fs, run)
	}

	// Ensure that there are 13 profiles (10 from Dummy + 3 from CSV).
	_, _, total = c.Profiles([]string{"email"}, "", false, 0, 1000)
	if total != 13 {
		t.Fatalf("expected 13 profiles, got %d", total)
	}

	// Import users and events from a JavaScript connection.
	var javaScript, javascriptUsersPipeline int
	t.Log("importing users and events...")
	{
		// Create a JavaScript connection with two pipelines (one for importing
		// events, one for importing identities) and retrieve its key.
		var javaScriptKey string
		{
			javaScript = c.CreateJavaScriptSource("JavaScript (source)", nil)
			keys := c.EventWriteKeys(javaScript)
			if len(keys) != 1 {
				t.Fatalf("expected one key, got %d keys", len(keys))
			}
			javaScriptKey = keys[0]
			c.CreatePipeline(javaScript, "Event", meergotester.PipelineToSet{
				Name:    "JavaScript",
				Enabled: true,
			})
			javascriptUsersPipeline = c.CreatePipeline(javaScript, "User", meergotester.PipelineToSet{
				Name:     "JavaScript",
				Enabled:  true,
				Filter:   meergotester.DefaultFilterUserFromEvents,
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email": "traits.email",
					},
				},
			})
		}

		// Send an identity event. More than importing an event, this should create an identity.
		c.SendEvent(javaScriptKey, analytics.Identify{
			UserId:      "f4ca124298",
			AnonymousId: "5ce0fd49-199a-47e7-b0c8-498f5144f0ee",
			Traits: map[string]interface{}{
				"email": "kbuessen0@example.com",
			},
		})
		time.Sleep(time.Second)
		c.RunIdentityResolution()
		c.WaitEventsStoredIntoWarehouse(ctx, 1)
	}

	// Ensure that there are 14 profiles (10 from Dummy + 3 from CSV + 1 from event).
	_, _, total = c.Profiles([]string{"email"}, "", false, 0, 1000)
	if total != 14 {
		t.Fatalf("expected 14 profiles, got %d", total)
	}

	// Set the "email" as identifier and run the Identity Resolution.
	c.UpdateIdentityResolution(true, []string{"email"})
	c.RunIdentityResolution()

	// Ensure that there are 10 profiles.
	profiles, _, total := c.Profiles([]string{"email"}, "", false, 0, 1000)
	if total != 10 {
		t.Fatalf("expected 10 users, got %d", total)
	}

	// Retrieve the MPID of "kbuessen0@example.com".
	var kBuessenMPID uuid.UUID
	for _, profile := range profiles {
		if profile.Attributes["email"] == "kbuessen0@example.com" {
			kBuessenMPID = profile.MPID
			break
		}
	}
	if kBuessenMPID == (uuid.UUID{}) {
		t.Fatalf("profile with email %q not found", "kbuessen0@example.com")
	}

	// Ensure that "kbuessen0@example.com" has one event associated.
	events := c.ProfileEvents(kBuessenMPID, []string{"timestamp"})
	if len(events) != 1 {
		t.Fatalf("expected %q to have one event associated, got %d", "kbuessen0@example.com", len(events))
	}

	// Validate the identities.
	identities, total := c.Identities(kBuessenMPID, 0, 1000)
	if total != 3 {
		t.Fatalf("expected profile %s to have 3 identities associated, got %d", kBuessenMPID, total)
	}
	assertEqualIdentity := func(got, expected meergotester.Identity) {
		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("expected identity %#v, got %#v", expected, got)
		}
	}
	{
		eventIdentity := getIdentityByConnection(t, identities, javaScript)
		eventIdentity.UpdatedAt = time.Time{}
		assertEqualIdentity(eventIdentity, meergotester.Identity{
			UserID:       "f4ca124298",
			AnonymousIDs: []string{"5ce0fd49-199a-47e7-b0c8-498f5144f0ee"},
			UpdatedAt:    time.Time{},
			Connection:   javaScript,
			Pipeline:     javascriptUsersPipeline,
		})
	}
	t.Log("identity imported from JavaScript is ok")
	{
		csvIdentity := getIdentityByConnection(t, identities, fs)
		assertEqualIdentity(csvIdentity, meergotester.Identity{
			UserID:       "1",
			AnonymousIDs: nil,
			UpdatedAt:    time.Date(2001, 2, 2, 3, 4, 5, 0, time.UTC),
			Connection:   fs,
			Pipeline:     csvPipeline,
		})
	}
	t.Log("identity imported from CSV is ok")
	{
		dummyIdentity := getIdentityByConnection(t, identities, dummy)
		dummyIdentity.UpdatedAt = time.Time{}
		assertEqualIdentity(dummyIdentity, meergotester.Identity{
			UserID:       "dummy1",
			AnonymousIDs: nil,
			UpdatedAt:    time.Time{},
			Connection:   dummy,
			Pipeline:     dummyPipeline,
		})
	}
	t.Log("identity imported from Dummy is ok")

}

func getIdentityByConnection(t *testing.T, identities []meergotester.Identity, connection int) meergotester.Identity {
	for _, identity := range identities {
		if identity.Connection == connection {
			return identity
		}
	}
	t.Fatalf("identity with connection %d not found among provided identities", connection)
	return meergotester.Identity{}
}
