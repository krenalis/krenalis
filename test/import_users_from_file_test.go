// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func TestImportUsersFromFile(t *testing.T) {

	// Determine the storage directory and assert that such directory exists.
	storageDir, err := filepath.Abs("testdata/storage")
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

	// Create the File System connection.
	fsID := c.CreateSourceFileSystem()

	c.UpdateIdentityResolution(true, []string{"email"})

	// Create a pipeline for the CSV for importing the users.
	importUsersPipelineID := c.CreatePipeline(fsID, "User", meergotester.PipelineToSet{
		Name:    "Import users from CSV on File System",
		Enabled: true,
		Path:    "users.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.String()},
			{Name: "name", Type: types.String()},
			{Name: "email", Type: types.String()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "first_name", Type: types.String().WithMaxLength(300), ReadOptional: true},
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"first_name": "name",
				"email":      "email",
			},
		},
		IdentityColumn: "identity",
		Format:         "csv",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
			"separator":      ",",
			"hasColumnNames": true,
		}),
	})

	// Run the pipeline that imports users.
	run := c.RunPipeline(importUsersPipelineID)

	// Wait for the import to finish.
	c.WaitRunsCompletion(fsID, run)

	// Retrieve the profiles and test them.
	const (
		expectedTotal       = 2
		expectedProfilesLen = 2
	)
	profiles, _, total := c.Profiles([]string{"email"}, "", false, 0, 100)
	profilesLen := len(profiles)
	if profilesLen != expectedProfilesLen {
		t.Fatalf("expected %d profiles, got %d", expectedProfilesLen, profilesLen)
	}
	if total != expectedTotal {
		t.Fatalf("expected \"total\" to be %d, got %d", expectedTotal, total)
	}

	// Retrieve the identities and test them.
	identities, total := c.ConnectionIdentities(fsID, 0, 100)
	if total != 2 {
		t.Fatalf("expected 2 identities, got %d", total)
	}
	for _, identity := range identities {
		if identity.Connection != fsID {
			t.Fatalf("expected connection %d, got %d", fsID, identity.Connection)
		}
		if identity.Pipeline != importUsersPipelineID {
			t.Fatalf("expected pipeline %d, got %d", importUsersPipelineID, identity.Pipeline)
		}
		if len(identity.AnonymousIDs) != 0 {
			t.Fatalf("expected zero anonymous ID for the identity, got %v", identity.AnonymousIDs)
		}
	}

}
