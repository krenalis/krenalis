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

	"github.com/open2b/chichi/test/chichitester"
	"github.com/open2b/chichi/types"
)

func TestActionsCreation(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := chichitester.InitAndLaunch(t)
	defer c.Stop()

	// Create some connections that will be used by the actions.
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
	srcFsID := c.AddSourceFilesystem(storageDir)
	dstFsID := c.AddDestinationFilesystem(storageDir)
	javaScriptConnection := c.AddJavaScriptSource("JavaScript (source)", "example.com", nil)
	postgreSQLConnection := c.AddSourcePostgreSQL()
	dummyExportConnection := c.AddDummy("Dummy (destination)", chichitester.Destination)

	tests := []struct {
		conn   int
		action chichitester.ActionToSet
		err    string
	}{
		{
			conn: srcFsID,
			action: chichitester.ActionToSet{
				Name: "Import users from a CSV on Filesystem",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "identity", Type: types.Text()},
					{Name: "Email", Type: types.Text()},
					{Name: "timestamp", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email":     "Email",
						"timestamp": "timestamp",
					},
				},
				IdentityProperty: "identity",
				UpdatedAtColumn:  "timestamp",
				UpdatedAtFormat:  "'%Y-%m-%d %H:%M:%S'",

				Connector: chichitester.CSVConnector,
				UIValues: chichitester.JSONEncodeUIValues(map[string]any{
					"Comma":          ",",
					"HasColumnNames": true,
				}),
			},
		},
		{
			conn: srcFsID,
			action: chichitester.ActionToSet{
				Name: "Import users from CSV on Filesystem",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "identity", Type: types.Text()},
					{Name: "Email", Type: types.Text()},
					{Name: "timestamp", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "Email", Type: types.Text()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"Email":     "Email",
						"timestamp": "timestamp",
					},
				},
				IdentityProperty: "identity",
				UpdatedAtColumn:  "timestamp",
				UpdatedAtFormat:  "'%Y-%m-%d %H:%M:%S'",
				Connector:        chichitester.CSVConnector,
				UIValues: chichitester.JSONEncodeUIValues(map[string]any{
					"Comma":          ",",
					"HasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"output schema cannot contain meta properties"}}`,
		},
		{
			conn: dstFsID,
			action: chichitester.ActionToSet{
				Name: "Export users to a CSV on Filesystem",
				Path: "users.csv",
				OutSchema: types.Object([]types.Property{
					{Name: "Email", Type: types.Text()}, // allowed because this is a destination connection.
					{Name: "timestamp", Type: types.DateTime()},
				}),
				Connector: chichitester.CSVConnector,
				UIValues: chichitester.JSONEncodeUIValues(map[string]any{
					"Comma":          ",",
					"HasColumnNames": true,
				}),
			},
		},
		{
			conn: srcFsID,
			action: chichitester.ActionToSet{
				Name: "Import users from CSV on Filesystem",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "identity", Type: types.Text()},
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email":     "email",
						"timestamp": "timestamp",
					},
				},
				IdentityProperty: "identity",
				UpdatedAtColumn:  "timestamp",
				Connector:        chichitester.CSVConnector,
				UIValues: chichitester.JSONEncodeUIValues(map[string]any{
					"Comma":          ",",
					"HasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"'updated at' format is required"}}`,
		},
		{
			conn: srcFsID,
			action: chichitester.ActionToSet{
				Name: "Import users from CSV on Filesystem",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email":     "email",
						"timestamp": "timestamp",
					},
				},
				UpdatedAtColumn: "timestamp",
				UpdatedAtFormat: "'%Y-%m-%d %H:%M:%S'",
				Connector:       chichitester.CSVConnector,
				UIValues: chichitester.JSONEncodeUIValues(map[string]any{
					"Comma":          ",",
					"HasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"identity property is mandatory"}}`,
		},
		{
			conn: srcFsID,
			action: chichitester.ActionToSet{
				Name: "Import users from CSV on Filesystem",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email":     "email",
						"timestamp": "timestamp",
					},
				},
				IdentityProperty: "- - invalid - -",
				Connector:        chichitester.CSVConnector,
				UIValues: chichitester.JSONEncodeUIValues(map[string]any{
					"Comma":          ",",
					"HasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"identity property is not a valid property name"}}`,
		},
		{
			conn: srcFsID,
			action: chichitester.ActionToSet{
				Name: "Import users from CSV on Filesystem",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityProperty: "email",
				UpdatedAtColumn:  "timestamp",
				Connector:        chichitester.CSVConnector,
				UIValues: chichitester.JSONEncodeUIValues(map[string]any{
					"Comma":          ",",
					"HasColumnNames": true,
				}),
			},
		},
		{
			conn: srcFsID,
			action: chichitester.ActionToSet{
				Name:             "Import users from CSV on Filesystem",
				Path:             "users.csv",
				IdentityProperty: "email",
				UpdatedAtColumn:  "timestamp",
				UpdatedAtFormat:  "'%Y-%m-%d %H:%M:%S'",
				Connector:        chichitester.CSVConnector,
				UIValues: chichitester.JSONEncodeUIValues(map[string]any{
					"Comma":          ",",
					"HasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"input schema must be valid"}}`,
		},
		{
			conn: srcFsID,
			action: chichitester.ActionToSet{
				Name: "Import users from CSV on Filesystem",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityProperty: "email",
				UpdatedAtColumn:  "timestamp",
				UpdatedAtFormat:  "'%Y-%m-%d %H:%M:%S'",
				Connector:        chichitester.CSVConnector,
				UIValues: chichitester.JSONEncodeUIValues(map[string]any{
					"Comma":          ",",
					"HasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"action cannot specify an 'updated at' format"}}`,
		},
		{
			conn: postgreSQLConnection,
			action: chichitester.ActionToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "email" FROM "my_table"`,
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityProperty: "my_id_column",
				UpdatedAtColumn:  "timestamp",
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"identity property \"my_id_column\" not found within input schema"}}`,
		},
		{
			conn: postgreSQLConnection,
			action: chichitester.ActionToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "id", "email" FROM "my_table"`,
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityProperty: "id",
				UpdatedAtColumn:  "timestamp",
			},
		},
		{
			conn: postgreSQLConnection,
			action: chichitester.ActionToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "id", "email", "timestamp" FROM "my_table"`,
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email", Type: types.Text()},
					{Name: "timestamp", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityProperty: "id",
				UpdatedAtColumn:  "timestamp",
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"'updated at' format is required"}}`,
		},
		{
			conn: postgreSQLConnection,
			action: chichitester.ActionToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "id", "email", "my_updated_at" FROM "my_table"`,
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email", Type: types.Text()},
					{Name: "my_updated_at", Type: types.DateTime()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityProperty: "id",
				UpdatedAtColumn:  "my_updated_at",
			},
		},
		{
			conn: javaScriptConnection,
			action: chichitester.ActionToSet{
				Name:     "Import users identities from events",
				Enabled:  true,
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "traits.email",
					},
				},
			},
		},
		{
			conn: javaScriptConnection,
			action: chichitester.ActionToSet{
				Name:    "Import users identities from events",
				Enabled: true,
				InSchema: types.Object([]types.Property{
					{Name: "traits", Type: types.Object([]types.Property{
						{Name: "email", Type: types.Text()},
					})},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "traits.email",
					},
				},
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"input schema must be invalid for actions that import users identities from events"}}`,
		},
		{
			conn: dummyExportConnection,
			action: chichitester.ActionToSet{
				Name: "Export users to Dummy",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text(), Nullable: true},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				ExportMode: chichitester.ExportModeCreateOrUpdate,
				MatchingProperties: &chichitester.MatchingProperties{
					Internal: "email",
					External: types.Property{
						Name: "email",
						Type: types.Text(),
					},
				},
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"export on duplicated users setting cannot be nil"}}`,
		},
		{
			conn: javaScriptConnection,
			action: chichitester.ActionToSet{
				Name:     "Import users identities from events",
				Enabled:  true,
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "traits.email",
					},
				},
				ExportOnDuplicatedUsers: &[]bool{false}[0],
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"export on duplicated users setting must be nil"}}`,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			_, err := c.AddActionErr(test.conn, "Users", test.action)
			switch {
			case test.err == "" && err == nil:
				// Ok.
			case test.err == "" && err != nil:
				t.Fatalf("expecting no errors, got err: %q", err)
			case test.err != "" && err == nil:
				t.Fatalf("expecting error %q, got no errors", test.err)
			case test.err != "" && err != nil:
				if test.err == err.Error() {
					// Ok.
				} else {
					t.Fatalf("expecting error %q, got: %q", test.err, err)
				}
			}
		})
	}

}
