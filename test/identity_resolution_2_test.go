// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/meergotester"
)

// TestIdentityResolution2 tests the behavior of Identity Resolution for arrays
// and primary connections.
func TestIdentityResolution2(t *testing.T) {

	storage := meergotester.NewTempStorage(t)

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.PopulateUserSchema(false)
	c.SetFilesystemRoot(storage.Root())
	c.Start()
	defer c.Stop()

	// Add properties to the user schema.
	schema := types.Object([]types.Property{
		{Name: "email", Type: types.Text().WithCharLen(254), ReadOptional: true},
		{Name: "name", Type: types.Text(), ReadOptional: true},
		{Name: "phone_numbers", Type: types.Array(types.Text()), ReadOptional: true},
		{Name: "total_orders", Type: types.Int(32), ReadOptional: true},
	})
	c.AlterUserSchema(schema, nil, nil)

	// Set the email as the only identifier, as the 3 identities, imported from
	// the 3 connections, will all be put together in a single user as they
	// share the same email.
	//
	// Also disable the automatic execution of the Identity Resolution, which
	// will be explicitly executed later.
	c.UpdateIdentityResolution(false, []string{"email"})

	sourceA := c.CreateSourceFilesystem()
	sourceB := c.CreateSourceFilesystem()
	sourceC := c.CreateSourceFilesystem()

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

	properties := map[string]bool{
		"email":            true,
		"name":             true,
		"phone_numbers":    true,
		"total_orders":     true,
		"last_change_time": true,
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

	addJSONAction := func(source int, filename string, properties map[string]bool) int {
		return c.CreateAction(source, "User", meergotester.ActionToSet{
			Name:    "Action",
			Enabled: true,
			Path:    filename,
			InSchema: types.Object([]types.Property{
				{Name: "email", Type: types.JSON()},
				{Name: "name", Type: types.JSON()},
				{Name: "phone_numbers", Type: types.JSON()},
				{Name: "total_orders", Type: types.JSON()},
				{Name: "last_change_time", Type: types.JSON()},
			}),
			OutSchema: schema,
			Transformation: &meergotester.Transformation{
				// This transformation functions returns the user without the
				// properties that are "null".
				Function: &meergotester.TransformationFunction{
					Language: "Python",
					Source: strings.Join([]string{
						`def transform(user: dict) -> dict:`,
						`    return {k: v for k, v in user.items() if v is not None}`,
					}, "\n"),
					InPaths:  []string{"email", "name", "phone_numbers", "total_orders"},
					OutPaths: []string{"email", "name", "phone_numbers", "total_orders"},
				},
			},
			IdentityColumn:       "email",
			LastChangeTimeColumn: "last_change_time",
			LastChangeTimeFormat: "%Y-%m-%d %H:%M:%S",
			Format:               "json",
			FormatSettings:       meergotester.SettingsProperties(properties),
		})
	}

	actionA := addJSONAction(sourceA, "A.json", properties)
	actionB := addJSONAction(sourceB, "B.json", properties)
	actionC := addJSONAction(sourceC, "C.json", properties)

	exec1 := c.ExecuteAction(actionA)
	exec2 := c.ExecuteAction(actionB)
	exec3 := c.ExecuteAction(actionC)
	c.WaitForExecutionsCompletion(sourceA, exec1)
	c.WaitForExecutionsCompletion(sourceB, exec2)
	c.WaitForExecutionsCompletion(sourceC, exec3)

	// Resolve the identities.
	c.RunIdentityResolution()

	// Test that the execution of the Identity Resolution has ended and that its
	// duration has a reasonable value.
	startTime, endTime := c.LatestIdentityResolution()
	if startTime == nil {
		t.Fatalf("startTime should be a valid timestamp, got nil")
	}
	if endTime == nil {
		t.Fatalf("endTime should be a valid timestamp, got nil")
	}
	duration := endTime.Sub(*startTime)
	if duration == 0 {
		t.Fatalf("expected a positive Identity Resolution duration, got zero")
	}
	if duration > 1*time.Hour {
		t.Fatalf("expected an Identity Resolution duration less than 1 hour, got a duration of %v", duration)
	}

	// Check that there is only one user, and that its properties have been
	// merged correctly.

	users, _, total := c.Users(schema.Properties().Names(), "", false, 0, 100)
	if total != 1 {
		t.Fatalf("expected just 1 user (which is the merge of the 3 identities), got %d instead", total)
	}
	user := users[0]
	if user.SourcesLastUpdate.IsZero() {
		t.Fatalf("expected a valid value for 'sourcesLastUpdate', got zero instead")
	}
	expected := map[string]any{
		"email":         "a@b",
		"name":          "John",
		"phone_numbers": []any{"+11 111", "+22 222", "+33 333"},
		"total_orders":  json.Number("21"),
	}
	if !reflect.DeepEqual(user.Traits, expected) {
		t.Fatalf("expected user traits %#v, got %#v", expected, user)
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
	c.AlterUserSchema(schema, primarySources, nil)

	c.RunIdentityResolution()

	users, _, total = c.Users(schema.Properties().Names(), "", false, 0, 100)
	if total != 1 {
		t.Fatalf("expected just 1 user (which is the merge of the 3 identities), got %d instead", total)
	}
	user = users[0]
	if user.SourcesLastUpdate.IsZero() {
		t.Fatalf("expected a valid value for 'sourcesLastUpdate', got zero instead")
	}
	expected = map[string]any{
		"email":         "a@b",
		"name":          "John",
		"phone_numbers": []any{"+11 111", "+22 222", "+33 333"},
		"total_orders":  json.Number("20"),
	}
	if !reflect.DeepEqual(user.Traits, expected) {
		t.Fatalf("expected user traits %#v, got %#v", expected, user)
	}

	storage.Remove()
}
