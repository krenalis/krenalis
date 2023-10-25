//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"chichi/connector/types"
	"chichi/test/chichitester"
)

func TestActionsCreation(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Create some connections that will be used by the actions.
	var csvConnection int
	{
		// CSV connection.
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
		fsID := c.AddSourceFilesystem(storageDir)
		csvConnection = c.AddSourceCSV(fsID)
	}

	tests := []struct {
		conn   int
		action map[string]any
		err    string
	}{
		{
			conn: csvConnection,
			action: map[string]any{
				"Target": "Users",
				"Action": map[string]any{
					"Name": "Import users from CSV on Filesystem",
					"Path": "users.csv",
					"InSchema": types.Object([]types.Property{
						{Name: "email", Type: types.Text()},
						{Name: "timestamp", Type: types.Text()},
					}),
					"OutSchema": types.Object([]types.Property{
						{Name: "Email", Type: types.Text()},
						{Name: "timestamp", Type: types.DateTime().WithLayout(time.DateTime)},
					}),
					"Mapping": map[string]string{
						"Email":     "email",
						"timestamp": "timestamp",
					},
					"IdentityProperty":  "identity",
					"TimestampProperty": "timestamp",
					"TimestampFormat":   "2006-01-02 15:04:05",
				},
			},
		},
		{
			conn: csvConnection,
			action: map[string]any{
				"Target": "Users",
				"Action": map[string]any{
					"Name": "Import users from CSV on Filesystem",
					"Path": "users.csv",
					"InSchema": types.Object([]types.Property{
						{Name: "email", Type: types.Text()},
						{Name: "timestamp", Type: types.Text()},
					}),
					"OutSchema": types.Object([]types.Property{
						{Name: "Email", Type: types.Text()},
						{Name: "timestamp", Type: types.DateTime().WithLayout(time.DateTime)},
					}),
					"Mapping": map[string]string{
						"Email":     "email",
						"timestamp": "timestamp",
					},
					"IdentityProperty":  "identity",
					"TimestampProperty": "timestamp",
				},
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"timestamp format is mandatory when a timestamp property is provided"}}`,
		},
		{
			conn: csvConnection,
			action: map[string]any{
				"Target": "Users",
				"Action": map[string]any{
					"Name": "Import users from CSV on Filesystem",
					"Path": "users.csv",
					"InSchema": types.Object([]types.Property{
						{Name: "email", Type: types.Text()},
						{Name: "timestamp", Type: types.Text()},
					}),
					"OutSchema": types.Object([]types.Property{
						{Name: "Email", Type: types.Text()},
						{Name: "timestamp", Type: types.DateTime().WithLayout(time.DateTime)},
					}),
					"Mapping": map[string]string{
						"Email":     "email",
						"timestamp": "timestamp",
					},
					"TimestampProperty": "timestamp",
					"TimestampFormat":   "2006-01-02 15:04:05",
				},
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"property for the identity is mandatory"}}`,
		},
		{
			conn: csvConnection,
			action: map[string]any{
				"Target": "Users",
				"Action": map[string]any{
					"Name": "Import users from CSV on Filesystem",
					"Path": "users.csv",
					"InSchema": types.Object([]types.Property{
						{Name: "email", Type: types.Text()},
						{Name: "timestamp", Type: types.Text()},
					}),
					"OutSchema": types.Object([]types.Property{
						{Name: "Email", Type: types.Text()},
						{Name: "timestamp", Type: types.DateTime().WithLayout(time.DateTime)},
					}),
					"Mapping": map[string]string{
						"Email":     "email",
						"timestamp": "timestamp",
					},
					"IdentityProperty": "- - invalid - -",
				},
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"property for the identity has not a valid property name"}}`,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			_, err := c.AddActionErr(test.conn, test.action)
			switch {
			case test.err == "" && err == nil:
				// Ok.
			case test.err == "" && err != nil:
				t.Fatalf("expecting no errors, got err: %s", err)
			case test.err != "" && err == nil:
				t.Fatalf("expecting error %q, got no errors", test.err)
			case test.err != "" && err != nil:
				if test.err == err.Error() {
					// Ok.
				} else {
					t.Fatalf("expecting error %q, got: %s", test.err, err)
				}
			}
		})
	}

}
