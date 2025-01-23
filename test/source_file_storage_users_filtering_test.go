//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"path/filepath"
	"testing"

	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestSourceFileStorageUsersFiltering(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	storageDir, err := filepath.Abs("testdata/source_file_storage_users_filtering")
	if err != nil {
		t.Fatal(err)
	}
	fs1 := c.CreateSourceFilesystem(storageDir)

	action1 := c.CreateAction(fs1, "Users", meergotester.ActionToSet{
		Name:    "CSV",
		Enabled: true,
		Path:    "users.csv",
		Filter: &meergotester.Filter{
			Logical: meergotester.OpAnd,
			Conditions: []meergotester.FilterCondition{
				{
					Property: "email",
					Operator: meergotester.OpIsNot,
					Values:   []string{"et@example.com"},
				},
			},
		},
		InSchema: types.Object([]types.Property{
			{Name: "CSV_id", Type: types.Text()},
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
		IdentityProperty: "CSV_id",
		Format:           "CSV",
		FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})

	exec1 := c.ExecuteAction(action1, true)

	c.WaitForExecutionsCompletionAllowFailed(fs1, exec1)

	_, _, total := c.Users([]string{"email"}, "", false, 0, 100)

	// The CSV file contains 10 users, but one of them was filtered out, so
	// there must be 9.
	const expectedTotal = 9
	if expectedTotal != total {
		t.Fatalf("expected %d users, got %d", expectedTotal, total)
	}
	t.Logf("the APIs successfully returned %d users", total)
}
