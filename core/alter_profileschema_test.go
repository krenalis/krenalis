// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/krenalis/krenalis/connectors/postgresql"
	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/datastore/diffschemas"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/kms"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestAlterProfileSchemaFinalization verifies that repeated finalization
// preserves the committed state and repeated cleanup does not repeat remote
// effects.
func TestAlterProfileSchemaFinalization(t *testing.T) {

	if testing.Short() {
		t.Skip("requires PostgreSQL")
	}
	container, err := postgres.Run(t.Context(), testimages.PostgreSQL,
		postgres.WithDatabase("krenalis"), postgres.WithUsername("krenalis"), postgres.WithPassword("krenalis"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(time.Minute)))
	if err != nil {
		t.Fatalf("expected PostgreSQL container, got %v", err)
	}
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(container)
		if err != nil {
			t.Errorf("expected container termination, got %v", err)
		}
	})
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("expected container host, got %v", err)
	}
	port, err := container.MappedPort(t.Context(), "5432/tcp")
	if err != nil {
		t.Fatalf("expected container port, got %v", err)
	}
	options := db.Options{Host: host, Port: int(port.Num()), Username: "krenalis", Password: "krenalis", Database: "krenalis"}
	admin, err := db.Open(&options)
	if err != nil {
		t.Fatalf("expected database pool, got %v", err)
	}
	t.Cleanup(admin.Close)

	t.Run("duplicate", func(t *testing.T) {

		const name = "duplicate"
		_, err := admin.Exec(t.Context(), "CREATE DATABASE "+name)
		if err != nil {
			t.Fatalf("expected test database, got %v", err)
		}
		opts := options
		opts.Database = name
		database, err := db.Open(&opts)
		if err != nil {
			t.Fatalf("expected test pool, got %v", err)
		}
		t.Cleanup(database.Close)
		manager, err := kms.New(t.Context(), "key:"+base64.StdEncoding.EncodeToString(make([]byte, 32)))
		if err != nil {
			t.Fatalf("expected KMS, got %v", err)
		}
		err = initdb.InitIfEmpty(t.Context(), database, manager, false)
		if err != nil {
			t.Fatalf("expected initialized schema, got %v", err)
		}
		const workspace = "111111111111"
		const source = "222222222222"
		const pipeline = "333333333333"
		const opID = "00000000-0000-4000-8000-000000000001"
		schema := types.Object([]types.Property{{Name: "email", Type: types.String(), ReadOptional: true}})
		raw, err := schema.MarshalJSON()
		if err != nil {
			t.Fatalf("expected encoded schema, got %v", err)
		}
		remote := &finalizationWarehouse{}
		platform := fmt.Sprintf("FinalizationTest%d", finalizationWarehouseNumber.Add(1))
		warehouses.Register(warehouses.Platform{Name: platform}, func(warehouses.SettingsLoader, warehouses.DialWith) *finalizationWarehouse {
			return remote
		})
		// Only primary sources change, so execution needs no warehouse DDL.
		_, err = database.Exec(t.Context(), `INSERT INTO workspaces
				(id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
				kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key,
				profile_schema, identifiers, alter_profile_schema_id, alter_profile_schema_schema,
				alter_profile_schema_primary_sources, alter_profile_schema_operations)
				SELECT $1, id, 'test', $5, 'Normal', '', '', '', $2, ARRAY['email'], $3, $2,
				jsonb_build_object('email', $4::text), '[]' FROM organizations`, workspace, raw, opID, source, platform)
		if err != nil {
			t.Fatalf("expected pending schema operation, got %v", err)
		}
		_, err = database.Exec(t.Context(), `INSERT INTO connections
				(id, workspace, connector, role, kms_encrypted_settings_key)
				VALUES ($1, $2, 'postgresql', 'Source', '')`, source, workspace)
		if err != nil {
			t.Fatalf("expected source connection, got %v", err)
		}
		_, err = database.Exec(t.Context(), `INSERT INTO pipelines
				(id, connection, target, event_type, ordering_group, delivery_endpoint, transformation_language,
				matching_in, matching_out, update_on_duplicates, table_key, properties_to_unset)
				VALUES ($1, $2, 'User', '', '', '', 'JavaScript', '', '', false, '', ARRAY['email'])`, pipeline, source)
		if err != nil {
			t.Fatalf("expected cleanup pipeline, got %v", err)
		}
		_, err = database.Exec(t.Context(), "UPDATE workspaces SET pipelines_to_purge = ARRAY[$1] WHERE id = $2", pipeline, workspace)
		if err != nil {
			t.Fatalf("expected pending purge, got %v", err)
		}
		st, err := state.New(t.Context(), database, manager, nil, false)
		if err != nil {
			t.Fatalf("expected initialized State, got %v", err)
		}
		t.Cleanup(func() { st.Close(context.Background()) })
		ds, err := datastore.New(st, nil)
		if err != nil {
			t.Fatalf("expected datastore, got %v", err)
		}
		t.Cleanup(ds.Close)
		core := &Core{state: st, datastore: ds}
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		defer cancel()
		remote.cancelOnRepeat = cancel
		core.close.ctx = ctx
		core.executeAlterProfileSchema(workspace, opID, schema, map[string]string{"email": source}, nil)
		var completed time.Time
		err = database.QueryRow(ctx, "SELECT alter_profile_schema_end_time FROM workspaces WHERE id = $1", workspace).
			Scan(&completed)
		if err != nil {
			t.Fatalf("expected committed finalization, got %v", err)
		}
		// A duplicate must not overwrite schema or primary sources. Change only
		// schema metadata to keep this check independent of warehouse DDL.
		duplicateSchema := types.Object([]types.Property{
			{Name: "email", Type: types.String(), ReadOptional: true, Description: "Unexpected description"},
		})
		duplicateRaw, err := duplicateSchema.MarshalJSON()
		if err != nil {
			t.Fatalf("expected encoded duplicate schema, got %v", err)
		}
		if bytes.Equal(raw, duplicateRaw) {
			t.Fatal("expected duplicate schema to differ from committed schema, got identical JSON")
		}
		core.executeAlterProfileSchema(workspace, opID, duplicateSchema, map[string]string{}, nil)
		var schemaValid, identifiersValid, operationCleared, completionUnchanged bool
		if err := database.QueryRow(ctx, `SELECT profile_schema = $2, identifiers = ARRAY['email'],
				alter_profile_schema_id IS NULL AND alter_profile_schema_schema = 'null'
					AND alter_profile_schema_primary_sources = 'null' AND alter_profile_schema_operations = 'null',
				alter_profile_schema_end_time = $3
				FROM workspaces WHERE id = $1`, workspace, raw, completed).
			Scan(&schemaValid, &identifiersValid, &operationCleared, &completionUnchanged); err != nil {
			t.Fatalf("expected finalization readback, got %v", err)
		}
		if !schemaValid {
			t.Error("expected preserved profile schema, got changed schema")
		}
		if !identifiersValid {
			t.Error("expected preserved identifiers, got changed identifiers")
		}
		if !operationCleared {
			t.Error("expected cleared operation ID, schema, primary sources and operations, got pending fields")
		}
		if !completionUnchanged {
			t.Error("expected preserved completion time, got changed completion time")
		}

		var sourceCount, matchingSources int
		if err := database.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE s.source = $2 AND s.path = 'email')
				FROM primary_sources s JOIN connections c ON c.id = s.source WHERE c.workspace = $1`, workspace, source).
			Scan(&sourceCount, &matchingSources); err != nil {
			t.Fatalf("expected primary sources readback, got %v", err)
		}
		if sourceCount != 1 || matchingSources != 1 {
			t.Errorf("expected exactly one primary source matching %s/email, got %d sources and %d matches",
				source, sourceCount, matchingSources)
		}
		var completionEvents int
		if err := database.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE name = 'EndAlterProfileSchema'").
			Scan(&completionEvents); err != nil {
			t.Fatalf("expected completion events readback, got %v", err)
		}
		if completionEvents != 1 {
			t.Errorf("expected one completion event, got %d", completionEvents)
		}

		t.Run("cleaner_does_not_repeat_remote_effects", func(t *testing.T) {

			cleaner := &pipelineCleaner{core: core}
			cleaner.close.ctx = ctx
			cleaner.purgeWorkspace(workspace)
			cleaner.unsetIdentityProperties(pipeline)
			if remote.deletes != 2 || remote.unsets != 1 {
				t.Fatalf("expected two table deletes and one unset, got %d, %d", remote.deletes, remote.unsets)
			}
			var purged, propertiesUnset bool
			if err := database.QueryRow(ctx, `SELECT
						(SELECT cardinality(pipelines_to_purge) = 0 FROM workspaces WHERE id = $1),
						(SELECT cardinality(properties_to_unset) = 0 FROM pipelines WHERE id = $2)`,
				workspace, pipeline).Scan(&purged, &propertiesUnset); err != nil {
				t.Fatalf("expected persisted cleanup, got %v", err)
			}
			if !purged {
				t.Fatal("expected no pipelines pending purge, got pending pipelines")
			}
			if !propertiesUnset {
				t.Fatal("expected no properties pending removal, got pending properties")
			}

			// Repeating completed cleanup must leave the remote counters unchanged.
			cleaner.purgeWorkspace(workspace)
			cleaner.unsetIdentityProperties(pipeline)
			if remote.deletes != 2 || remote.unsets != 1 {
				t.Fatalf("expected repeated cleanup to leave two deletes and one unset, got %d deletes and %d unsets",
					remote.deletes, remote.unsets)
			}
			if err := ctx.Err(); err != nil {
				t.Fatalf("expected active cleanup context, got %v", err)
			}

		})

	})

}

func Test_checkAllowedTypesProfileSchema(t *testing.T) {

	tests := []struct {
		name   string
		schema types.Type
		err    string
	}{
		{
			name: "No errors",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
			}),
		},
		{
			name: "Nullable object",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true, Nullable: true},
			}),
			err: "profile schema properties cannot be nullable",
		},
		{
			name: "Array with object item",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Object([]types.Property{
					{Name: "a", Type: types.Int(32), ReadOptional: true},
					{Name: "b", Type: types.String(), ReadOptional: true},
				})), ReadOptional: true},
			}),
			err: `profile schema properties cannot have type array(object)`,
		},
		{
			name: "Property with a prefilled value",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), Prefilled: "1234"},
				}), ReadOptional: true},
			}),
			err: "profile schema properties cannot have a prefilled value",
		},
		{
			name: "Meta properties",
			schema: types.Object([]types.Property{
				{Name: "_id", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
			}),
			err: "profile schema cannot have meta properties",
		},
		{
			name: "Array with unique elements",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Int(32)).WithUnique(), ReadOptional: true},
			}),
			err: "profile schema properties with type array cannot specify unique elements",
		},
		{
			name: "Arrays which specify a minimum number of elements",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Int(32)).WithMinElements(1), ReadOptional: true},
			}),
			err: "profile schema properties with type array cannot specify minimum elements count",
		},
		{
			name: "Arrays which specify a maximum number of elements",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Int(32)).WithMaxElements(types.MaxElements - 1), ReadOptional: true},
			}),
			err: "profile schema properties with type array cannot specify maximum elements count",
		},
		{
			name: "Array with string with values item",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "data", Type: types.Array(types.String().WithValues("a", "b")), ReadOptional: true},
			}),
			err: "profile schema properties of type array(string) cannot specify values for their element type",
		},
		{
			name: "Map with object item",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "data", Type: types.Map(types.Object([]types.Property{
					{Name: "a", Type: types.String(), ReadOptional: true},
				})), ReadOptional: true},
			}),
			err: "profile schema properties cannot have type map(object)",
		},
		{
			name: "Map with string with pattern item",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "data", Type: types.Map(
					types.String().WithPattern(regexp.MustCompile(`^a+$`))), ReadOptional: true},
			}),
			err: "profile schema properties of type map(string) cannot specify a pattern for their element type",
		},
		{
			name: "String with values",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String().WithValues("a", "b", "c"), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
			}),
			err: "profile schema properties with type string cannot specify values",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotErr := checkAllowedPropertyProfileSchema(test.schema)
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if gotErrStr != test.err {
				t.Fatalf("expected error %q, got %q", test.err, gotErrStr)
			}
		})
	}

}

// Test_profileSchemaChangeRequiresWarehouseDDL tests which profile schema
// changes require DDL on the data warehouse.
func Test_profileSchemaChangeRequiresWarehouseDDL(t *testing.T) {

	property := func(name string) types.Property {
		return types.Property{Name: name, Type: types.String(), ReadOptional: true}
	}

	tests := []struct {
		name      string
		oldSchema types.Type
		newSchema types.Type
		rePaths   map[string]any
		expected  bool
	}{
		{
			name:      "Identical schemas",
			oldSchema: types.Object([]types.Property{property("a"), property("b")}),
			newSchema: types.Object([]types.Property{property("a"), property("b")}),
		},
		{
			name: "Description changed",
			oldSchema: types.Object([]types.Property{
				property("a"),
			}),
			newSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), ReadOptional: true, Description: "New description"},
			}),
		},
		{
			name:      "Property added",
			oldSchema: types.Object([]types.Property{property("a")}),
			newSchema: types.Object([]types.Property{property("a"), property("b")}),
			expected:  true,
		},
		{
			name:      "Property removed",
			oldSchema: types.Object([]types.Property{property("a"), property("b")}),
			newSchema: types.Object([]types.Property{property("a")}),
			expected:  true,
		},
		{
			name:      "Property renamed",
			oldSchema: types.Object([]types.Property{property("a")}),
			newSchema: types.Object([]types.Property{property("b")}),
			rePaths:   map[string]any{"b": "a"},
			expected:  true,
		},
		{
			name:      "Top-level properties reordered",
			oldSchema: types.Object([]types.Property{property("a"), property("b")}),
			newSchema: types.Object([]types.Property{property("b"), property("a")}),
			expected:  true,
		},
		{
			name: "Nested properties reordered",
			oldSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{property("a"), property("b")}), ReadOptional: true},
			}),
			newSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{property("b"), property("a")}), ReadOptional: true},
			}),
			expected: true,
		},
		{
			name:      "Explicit property recreation",
			oldSchema: types.Object([]types.Property{property("a")}),
			newSchema: types.Object([]types.Property{property("a")}),
			rePaths:   map[string]any{"a": nil},
			expected:  true,
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			// Keep the fixtures within the profile schema domain. Invalid
			// properties, including nullable ones, belong to validation tests.
			if err := checkAllowedPropertyProfileSchema(test.oldSchema); err != nil {
				t.Fatalf("old schema contains a property not allowed in a profile schema: %s", err)
			}
			if err := checkAllowedPropertyProfileSchema(test.newSchema); err != nil {
				t.Fatalf("new schema contains a property not allowed in a profile schema: %s", err)
			}

			operations, err := diffschemas.Diff(test.oldSchema, test.newSchema, test.rePaths, "")
			if err != nil {
				t.Fatal(err)
			}
			actual := profileSchemaChangeRequiresWarehouseDDL(test.oldSchema, test.newSchema, operations)
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}

		})

	}

}

func Test_validatePrimarySources(t *testing.T) {

	const validPrimarySourceID = "7B3mN9qK2xA4"

	schema := types.Object([]types.Property{
		{Name: "first_name", Type: types.String(), ReadOptional: true},
		{Name: "address", Type: types.Object([]types.Property{
			{Name: "street", Type: types.String(), ReadOptional: true},
		}), ReadOptional: true},
		{Name: "phone_numbers", Type: types.Array(types.String()), ReadOptional: true},
	})

	tests := []struct {
		primarySources map[string]string
		expectedErr    string
	}{
		{
			primarySources: nil,
		},
		{
			primarySources: map[string]string{},
		},
		{
			primarySources: map[string]string{"first_name": validPrimarySourceID},
		},
		{
			primarySources: map[string]string{"first_name": ""},
			expectedErr:    `primary source identifier "" is not valid`,
		},
		{
			primarySources: map[string]string{"first_name": "7B3mN9qK2xA"},
			expectedErr:    `primary source identifier "7B3mN9qK2xA" is not valid`,
		},
		{
			primarySources: map[string]string{"first_name": "7B3mN9qK2x0"},
			expectedErr:    `primary source identifier "7B3mN9qK2x0" is not valid`,
		},
		{
			primarySources: map[string]string{"address.street": validPrimarySourceID},
		},
		{
			primarySources: map[string]string{"first_name": validPrimarySourceID, "not_a_prop": "9zQ4Tn7B3mS6"},
			expectedErr:    "property path \"not_a_prop\" does not exist",
		},
		{
			primarySources: map[string]string{"address": validPrimarySourceID},
			expectedErr:    "primary sources cannot be specified for object properties",
		}, {
			primarySources: map[string]string{"phone_numbers": validPrimarySourceID},
			expectedErr:    "primary sources cannot be specified for array(string) properties",
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			gotErr := validatePrimarySources(schema, test.primarySources)
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if gotErrStr != test.expectedErr {
				t.Fatalf("expected error %q, got %q instead", test.expectedErr, gotErrStr)
			}
		})
	}

}

var finalizationWarehouseNumber atomic.Uint64

// finalizationWarehouse counts remote cleanup effects; other methods are
// deliberately unavailable because these tests execute no warehouse DDL.
type finalizationWarehouse struct {
	warehouses.Warehouse
	deletes        int
	unsets         int
	cancelOnRepeat context.CancelFunc
}

func (w *finalizationWarehouse) Close() error {
	return nil
}

func (w *finalizationWarehouse) Delete(ctx context.Context, table string, where warehouses.Expr) error {
	w.deletes++
	if w.deletes > 2 {
		w.cancelOnRepeat()
		return context.Canceled
	}
	return nil
}

func (w *finalizationWarehouse) UnsetIdentityColumns(ctx context.Context, pipeline string, columns []warehouses.Column) error {
	w.unsets++
	if w.unsets > 1 {
		w.cancelOnRepeat()
		return context.Canceled
	}
	return nil
}
