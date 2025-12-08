// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/tools/types"
)

func Test_Identities(t *testing.T) {

	// Determine the storage directory.
	storageDir, err := filepath.Abs("testdata/identities_test")
	if err != nil {
		t.Fatal(err)
	}

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.NewMeergoInstance(t)
	c.SetFileSystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	c.UpdateIdentityResolution(true, []string{"email"})

	fs1 := c.CreateSourceFileSystem()
	fs2 := c.CreateSourceFileSystem()

	pipeline1 := c.CreatePipeline(fs1, "User", meergotester.PipelineToSet{
		Name:    "CSV 1",
		Enabled: true,
		Path:    "users1.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.String()},
			{Name: "email", Type: types.String()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		IdentityColumn: "identity",
		Format:         "csv",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
			"Separator":      ",",
			"HasColumnNames": true,
		}),
	})

	pipeline2 := c.CreatePipeline(fs2, "User", meergotester.PipelineToSet{
		Name:    "CSV 2",
		Enabled: true,
		Path:    "users2.csv",
		InSchema: types.Object([]types.Property{
			{Name: "identity", Type: types.String()},
			{Name: "email", Type: types.String()},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
		IdentityColumn: "identity",
		Format:         "csv",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
			"Separator":      ",",
			"HasColumnNames": true,
		}),
	})

	run1 := c.RunPipeline(pipeline1)
	run2 := c.RunPipeline(pipeline2)

	c.WaitRunsCompletion(fs1, run1)
	c.WaitRunsCompletion(fs2, run2)

	profiles, _, total := c.Profiles([]string{"email"}, "", false, 0, 100)

	const expectedTotal = 4
	if expectedTotal != total {
		t.Fatalf("expected %d profiles, got %d", expectedTotal, total)
	}
	t.Logf("the APIs successfully returned %d profiles", total)

	var totalIdentities int

	for _, profile := range profiles {

		identities, total := c.Identities(profile.MPID, 0, 1000)

		if total != 1 && total != 2 {
			t.Fatalf("expected 'total' to be 1 or 2, got %d", total)
		}

		for _, identity := range identities {

			if anonIds := identity.AnonymousIDs; anonIds != nil {
				t.Fatalf("identity should have a nil 'AnonymousIDs', got %v", anonIds)
			}

			t.Logf(
				"the APIs returned an identity for profile with MPID %s that has"+
					" pipeline = %d, user ID = %v and updated at = %q",
				profile.MPID, identity.Pipeline, identity.UserID, identity.UpdatedAt)

			var idPrefix string
			switch identity.Pipeline {
			case pipeline1:
				idPrefix = "profiles1_"
			case pipeline2:
				idPrefix = "profiles2_"
			default:
				t.Fatalf("unexpected pipeline %d", identity.Pipeline)
			}

			// Check the identity ID label.
			if !strings.HasPrefix(identity.UserID, idPrefix) {
				t.Fatalf("unexpected user ID %q, it should have prefix %q", identity.UserID, idPrefix)
			}

			totalIdentities++
		}
	}

	const expectedTotalIdentities = 6
	if expectedTotalIdentities != totalIdentities {
		t.Fatalf("expected a total of %d identities, got %d", expectedTotalIdentities, totalIdentities)
	}
	t.Logf("there is a total of %d identities", totalIdentities)

	// Additional test: test that a call to '/identities' for a profile which does not exist
	// returns an empty slice.
	{
		var res struct {
			Identities []any `json:"identities"`
			Total      int   `json:"total"`
		}
		err := c.Call("GET", "/v1/profiles/7682c2a8-d85d-458b-9bd8-dc57cc12575a/identities", nil, &res)
		if err != nil {
			t.Fatalf("expected no identities, got error: %q", err)
		}
		if res.Identities == nil {
			t.Fatal("expected an empty slice for identities, got nil instead")
		}
		if n := len(res.Identities); n > 0 {
			t.Fatalf("expected no identities, got %d", n)
		}
		if res.Total != 0 {
			t.Fatalf("expected total of 0, got %d", res.Total)
		}
	}

}
