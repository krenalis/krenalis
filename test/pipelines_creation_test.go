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

func TestPipelinesCreation(t *testing.T) {

	// IMPORTANT: these tests were written before making the pipeline validation
	// function testable. These tests also cover the API call part and the HTTP
	// layer, so they are not removed for these reasons. However, unless there
	// is a particular motivation, instead of adding tests here it is better to
	// add them on the pipeline validation function, which is faster to test and
	// easier to debug.

	// Determine the storage.
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

	// Create some connections that will be used by the pipelines.
	srcFsID := c.CreateSourceFileSystem()
	dstFsID := c.CreateDestinationFilesystem()
	javaScriptConnection := c.CreateJavaScriptSource("JavaScript (source)", nil)
	postgreSQLConnection := c.CreateSourcePostgreSQL()

	tests := []struct {
		conn     int
		pipeline meergotester.PipelineToSet
		err      string
	}{
		{
			conn: srcFsID,
			pipeline: meergotester.PipelineToSet{
				Name: "Import users from a CSV on File System",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "identity", Type: types.String()},
					{Name: "Email", Type: types.String()},
					{Name: "timestamp", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
					{Name: "timestamp", Type: types.DateTime(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email":     "Email",
						"timestamp": "timestamp",
					},
				},
				IdentityColumn:  "identity",
				UpdatedAtColumn: "timestamp",
				UpdatedAtFormat: "%Y-%m-%d %H:%M:%S",
				Format:          "csv",
				FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
					"separator":      ",",
					"hasColumnNames": true,
				}),
			},
		},
		{
			conn: srcFsID,
			pipeline: meergotester.PipelineToSet{
				Name: "Import users from CSV on File System",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "identity", Type: types.String()},
					{Name: "_email", Type: types.String()},
					{Name: "timestamp", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "_email", Type: types.String()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"_email":    "_email",
						"timestamp": "timestamp",
					},
				},
				IdentityColumn:  "identity",
				UpdatedAtColumn: "timestamp",
				UpdatedAtFormat: "%Y-%m-%d %H:%M:%S",
				Format:          "csv",
				FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
					"separator":      ",",
					"hasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"output pipeline schema property \"_email\" is a meta property"}}`,
		},
		{
			conn: dstFsID,
			pipeline: meergotester.PipelineToSet{
				Name: "Export profiles to a CSV on File System",
				Path: "profiles.csv",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
					{Name: "timestamp", Type: types.DateTime(), ReadOptional: true},
				}),
				Format:  "csv",
				OrderBy: "email",
				FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
					"separator":      ",",
					"hasColumnNames": true,
				}),
			},
		},
		{
			conn: srcFsID,
			pipeline: meergotester.PipelineToSet{
				Name: "Import users from CSV on File System",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "identity", Type: types.String()},
					{Name: "email", Type: types.String()},
					{Name: "timestamp", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
					{Name: "timestamp", Type: types.DateTime(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email":     "email",
						"timestamp": "timestamp",
					},
				},
				IdentityColumn:  "identity",
				UpdatedAtColumn: "timestamp",
				Format:          "csv",
				FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
					"separator":      ",",
					"hasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"update time format is required"}}`,
		},
		{
			conn: srcFsID,
			pipeline: meergotester.PipelineToSet{
				Name: "Import users from CSV on File System",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "timestamp", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
					{Name: "timestamp", Type: types.DateTime(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email":     "email",
						"timestamp": "timestamp",
					},
				},
				UpdatedAtColumn: "timestamp",
				UpdatedAtFormat: "%Y-%m-%d %H:%M:%S",
				Format:          "csv",
				FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
					"separator":      ",",
					"hasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"identity column is mandatory"}}`,
		},
		{
			conn: srcFsID,
			pipeline: meergotester.PipelineToSet{
				Name: "Import users from CSV on File System",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "timestamp", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email":     "email",
						"timestamp": "timestamp",
					},
				},
				IdentityColumn: "- - invalid - -",
				Format:         "csv",
				FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
					"separator":      ",",
					"hasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"identity column is not a valid property name"}}`,
		},
		{
			conn: srcFsID,
			pipeline: meergotester.PipelineToSet{
				Name: "Import users from CSV on File System",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityColumn:  "email",
				UpdatedAtColumn: "timestamp",
				Format:          "csv",
				FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
					"separator":      ",",
					"hasColumnNames": true,
				}),
			},
		},
		{
			conn: srcFsID,
			pipeline: meergotester.PipelineToSet{
				Name:            "Import users from CSV on File System",
				Path:            "users.csv",
				IdentityColumn:  "email",
				UpdatedAtColumn: "timestamp",
				UpdatedAtFormat: "%Y-%m-%d %H:%M:%S",
				Format:          "csv",
				FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
					"separator":      ",",
					"hasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"input schema must be valid"}}`,
		},
		{
			conn: srcFsID,
			pipeline: meergotester.PipelineToSet{
				Name: "Import users from CSV on File System",
				Path: "users.csv",
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityColumn:  "email",
				UpdatedAtColumn: "timestamp",
				UpdatedAtFormat: "%Y-%m-%d %H:%M:%S",
				Format:          "csv",
				FormatSettings: meergotester.JSONEncodeSettings(map[string]any{
					"separator":      ",",
					"hasColumnNames": true,
				}),
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"pipeline cannot specify an update time format"}}`,
		},
		{
			conn: postgreSQLConnection,
			pipeline: meergotester.PipelineToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "email" FROM "my_table"`,
				InSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityColumn:  "my_id_column",
				UpdatedAtColumn: "timestamp",
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"identity column \"my_id_column\" not found within input schema"}}`,
		},
		{
			conn: postgreSQLConnection,
			pipeline: meergotester.PipelineToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "id", "email" FROM "my_table"`,
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email", Type: types.String()},
					{Name: "timestamp", Type: types.DateTime()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityColumn:  "id",
				UpdatedAtColumn: "timestamp",
			},
		},
		{
			conn: postgreSQLConnection,
			pipeline: meergotester.PipelineToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "id", "email", "timestamp" FROM "my_table"`,
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email", Type: types.String()},
					{Name: "timestamp", Type: types.String()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityColumn:  "id",
				UpdatedAtColumn: "timestamp",
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"update time format is required"}}`,
		},
		{
			conn: postgreSQLConnection,
			pipeline: meergotester.PipelineToSet{
				Name:  "Import users from PostgreSQL",
				Query: `SELECT "id", "email", "my_last_change_time" FROM "my_table"`,
				InSchema: types.Object([]types.Property{
					{Name: "id", Type: types.Int(32)},
					{Name: "email", Type: types.String()},
					{Name: "my_last_change_time", Type: types.DateTime()},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email": "email",
					},
				},
				IdentityColumn:  "id",
				UpdatedAtColumn: "my_last_change_time",
			},
		},
		{
			conn: javaScriptConnection,
			pipeline: meergotester.PipelineToSet{
				Name:     "Import identities from events",
				Enabled:  true,
				Filter:   meergotester.DefaultFilterUserFromEvents,
				InSchema: types.Type{},
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email": "traits.email",
					},
				},
			},
		},
		{
			conn: javaScriptConnection,
			pipeline: meergotester.PipelineToSet{
				Name:    "Import identities from events",
				Enabled: true,
				Filter:  meergotester.DefaultFilterUserFromEvents,
				InSchema: types.Object([]types.Property{
					{Name: "traits", Type: types.Object([]types.Property{
						{Name: "email", Type: types.String()},
					})},
				}),
				OutSchema: types.Object([]types.Property{
					{Name: "email", Type: types.String(), ReadOptional: true},
				}),
				Transformation: &meergotester.Transformation{
					Mapping: map[string]string{
						"email": "traits.email",
					},
				},
			},
			err: `unexpected HTTP status code 400: {"error":{"code":"BadRequest","message":"input schema must be invalid for pipelines that import identities from events"}}`,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			_, err := c.CreatePipelineErr(test.conn, "User", test.pipeline)
			switch {
			case test.err == "" && err == nil:
				// Ok.
			case test.err == "" && err != nil:
				t.Fatalf("expected no errors, got err: %q", err)
			case test.err != "" && err == nil:
				t.Fatalf("expected error %q, got no errors", test.err)
			case test.err != "" && err != nil:
				if test.err == err.Error() {
					// Ok.
				} else {
					t.Fatalf("expected error %q, got: %q", test.err, err)
				}
			}
		})
	}

}
