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
	var (
		srcCSVConnection     int
		dstCSVConnection     int
		postgreSQLConnection int
		websiteConnection    int
	)
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
		srcFsID := c.AddSourceFilesystem(storageDir)
		srcCSVConnection = c.AddSourceCSV(srcFsID)
		dstFsID := c.AddDestinationFilesystem(storageDir)
		dstCSVConnection = c.AddDestinationCSV(dstFsID)
		websiteConnection = c.AddWebsiteSource("Website (source)", "example.com")
	}
	{
		// PostgreSQL connection.
		postgreSQLConnection = c.AddSourcePostgreSQL()
	}

	tests := []struct {
		conn   int
		action chichitester.ActionToSet
		err    string
	}{
		{
			conn: srcCSVConnection,
			action: chichitester.ActionToSet{
				Name: "Import users from CSV on Filesystem",
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
				IdentityColumn:  "identity",
				TimestampColumn: "timestamp",
				TimestampFormat: "'%Y-%m-%d %H:%M:%S'",
			},
		},
		{
			conn: srcCSVConnection,
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
				IdentityColumn:  "identity",
				TimestampColumn: "timestamp",
				TimestampFormat: "'%Y-%m-%d %H:%M:%S'",
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"output schema cannot contain meta properties"}}`,
		},
		{
			conn: dstCSVConnection,
			action: chichitester.ActionToSet{
				Name: "Export users to a CSV on Filesystem",
				Path: "users.csv",
				OutSchema: types.Object([]types.Property{
					{Name: "Email", Type: types.Text()}, // allowed because this is a destination connection.
					{Name: "timestamp", Type: types.DateTime()},
				}),
			},
		},
		{
			conn: srcCSVConnection,
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
				IdentityColumn:  "identity",
				TimestampColumn: "timestamp",
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"timestamp format is required"}}`,
		},
		{
			conn: srcCSVConnection,
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
				TimestampColumn: "timestamp",
				TimestampFormat: "'%Y-%m-%d %H:%M:%S'",
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"column name for the identity is mandatory"}}`,
		},
		{
			conn: srcCSVConnection,
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
				IdentityColumn: "- - invalid - -",
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"column name for the identity has not a valid property name"}}`,
		},
		{
			conn: srcCSVConnection,
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
				IdentityColumn:  "email",
				TimestampColumn: "timestamp",
			},
		},
		{
			conn: srcCSVConnection,
			action: chichitester.ActionToSet{
				Name:            "Import users from CSV on Filesystem",
				Path:            "users.csv",
				IdentityColumn:  "email",
				TimestampColumn: "timestamp",
				TimestampFormat: "2006-01-02 15:04:05",
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"input schema must be valid"}}`,
		},
		{
			conn: srcCSVConnection,
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
				IdentityColumn:  "email",
				TimestampColumn: "timestamp",
				TimestampFormat: "'%Y-%m-%d %H:%M:%S'",
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"action cannot specify a timestamp format"}}`,
		},
		{
			conn: postgreSQLConnection,
			action: chichitester.ActionToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "email" FROM "my_table"`,
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"identity column \"id\" not found within input schema"}}`,
		},
		{
			conn: postgreSQLConnection,
			action: chichitester.ActionToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "id", "email" FROM "my_table"`,
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email", Type: types.Text()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.Text()},
				}),
				Transformation: chichitester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
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
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"timestamp column \"timestamp\" has kind Text instead of DateTime"}}`,
		},
		{
			conn: postgreSQLConnection,
			action: chichitester.ActionToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "id", "email", "timestamp" FROM "my_table"`,
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
			},
		},
		{
			conn: websiteConnection,
			action: chichitester.ActionToSet{
				Name:     "Import user traits from events",
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
			conn: websiteConnection,
			action: chichitester.ActionToSet{
				Name:    "Import user traits from events",
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
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"input schema must be invalid for actions that import user traits from events"}}`,
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
