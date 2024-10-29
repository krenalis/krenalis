//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"context"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"
	"github.com/meergo/meergo/warehouses/postgresql"
)

// The variable MEERGO_TEST_PATH_WAREHOUSE_POSTGRESQL must point to a JSON file
// containing the settings of the PostgreSQL warehouse that will be used in this
// test.
//
// WARNING: the warehouse must be empty, as the test will initialize it.

const settingsEnvKey = "MEERGO_TEST_PATH_WAREHOUSE_POSTGRESQL"

var columns = []meergo.Column{
	{Name: "email", Type: types.Text(), Nullable: true},
	{Name: "first_name", Type: types.Text(), Nullable: true},
	{Name: "last_name", Type: types.Text(), Nullable: true},
}

var columnByName map[string]meergo.Column

func init() {
	columnByName = make(map[string]meergo.Column, len(columns))
	for _, c := range columns {
		columnByName[c.Name] = c
	}
}

type identity struct {
	connection   int // a multiple of 100, from 100 to 900 (included)
	action       int // can be 1, 2 ... 9
	id           string
	isAnonymous  bool
	anonymousIDs []string
	properties   map[string]any
}

// TestWarehousesIdentityResolution tests the Identity Resolution. If the
// variable MEERGO_TEST_PATH_WAREHOUSE_POSTGRESQL is not set, the test is
// skipped.
func TestWarehousesIdentityResolution(t *testing.T) {

	settingsFile, ok := os.LookupEnv(settingsEnvKey)
	if !ok {
		t.Skipf("the %s environment variable is not present", settingsEnvKey)
	}

	tests := []struct {
		name           string
		identifiers    []string
		primarySources map[string]int
		identities     []identity
		expectedUsers  []map[string]any
	}{
		{
			name:        "One identity, no identifiers",
			identifiers: []string{},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil},
			},
		},
		{
			name:        "Two identities from the same connection (different ID), no identifiers",
			identifiers: []string{},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil},
				},
				{
					connection: 100,
					action:     2,
					id:         "c@d",
					properties: map[string]any{"email": "c@d", "first_name": nil, "last_name": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil},
				{"email": "c@d", "first_name": nil, "last_name": nil},
			},
		},
		{
			name:        "Two identities from the same connection (same ID), no identifiers",
			identifiers: []string{},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "abcd",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil},
				},
				{
					connection: 100,
					action:     2,
					id:         "abcd",
					properties: map[string]any{"email": "c@d", "first_name": nil, "last_name": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "c@d", "first_name": nil, "last_name": nil},
			},
		},
		{
			name:        "Two identities from two different connections, no identifiers",
			identifiers: []string{},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil},
				},
				{
					connection: 200,
					action:     2,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil},
				{"email": "a@b", "first_name": nil, "last_name": nil},
			},
		},
		{
			name:        "Two identities from two different connections, one identifier that merges them (first-level priority)",
			identifiers: []string{"email"},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil},
				},
				{
					connection: 200,
					action:     2,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil},
			},
		},
		{
			name:        "Two identities from two different connections, matching for an identifier with second-level priority, previous identifiers are both nil",
			identifiers: []string{"email", "last_name"},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": nil, "first_name": nil, "last_name": "Manzarek"},
				},
				{
					connection: 200,
					action:     2,
					id:         "c@d",
					properties: map[string]any{"email": nil, "first_name": nil, "last_name": "Manzarek"},
				},
			},
			expectedUsers: []map[string]any{
				{"email": nil, "first_name": nil, "last_name": "Manzarek"},
			},
		},
		{
			name:        "Two identities from two different connections, matching for an identifier with second-level priority, previous identifiers are nil and not nil",
			identifiers: []string{"email", "last_name"},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": nil, "first_name": nil, "last_name": "Manzarek"},
				},
				{
					connection: 200,
					action:     2,
					id:         "c@d",
					properties: map[string]any{"email": nil, "first_name": "Ray", "last_name": "Manzarek"},
				},
			},
			expectedUsers: []map[string]any{
				{"email": nil, "first_name": "Ray", "last_name": "Manzarek"},
			},
		},
		{
			name:        "Two identities from two different connections, two identifiers that merge them",
			identifiers: []string{"email", "first_name"},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": "a", "last_name": nil},
				},
				{
					connection: 200,
					action:     2,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": "b", "last_name": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": "b", "last_name": nil},
			},
		},
		{
			name:        "Merging two anonymous identities from the same connection",
			identifiers: []string{},
			identities: []identity{
				{
					connection:  100,
					action:      1,
					isAnonymous: true,
					id:          "46d59a94-6032-46d9-87f2-f006af0156f0",
					properties:  map[string]any{"email": nil, "first_name": "Luke", "last_name": nil},
				},
				{
					connection:  100,
					action:      2,
					isAnonymous: true,
					id:          "46d59a94-6032-46d9-87f2-f006af0156f0",
					properties:  map[string]any{"email": nil, "first_name": "Luke", "last_name": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": nil, "first_name": "Luke", "last_name": nil},
			},
		},
		{
			name:        "Two identities not merged as one is anonymous and one is not",
			identifiers: []string{},
			identities: []identity{
				{
					connection:  100,
					action:      1,
					isAnonymous: true,
					id:          "46d59a94-6032-46d9-87f2-f006af0156f0",
					properties:  map[string]any{"email": nil, "first_name": "Luke", "last_name": nil},
				},
				{
					connection: 100,
					action:     2,
					id:         "46d59a94-6032-46d9-87f2-f006af0156f0",
					properties: map[string]any{"email": nil, "first_name": "Luke", "last_name": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": nil, "first_name": "Luke", "last_name": nil},
				{"email": nil, "first_name": "Luke", "last_name": nil},
			},
		},
		{
			name:        "Primary source specified for one property, just one identity",
			identifiers: []string{},
			primarySources: map[string]int{
				"email": 100,
			},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil},
			},
		},
		{
			name: "Primary sources, three identities",
			identifiers: []string{
				"email",
			},
			primarySources: map[string]int{
				"first_name": 200,
				"last_name":  200,
			},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": "FIRST_100", "last_name": "LAST_100"},
				},
				{
					connection: 200,
					action:     2,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": "FIRST_200", "last_name": "LAST_200"},
				},
				{
					connection: 300,
					action:     3,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": "FIRST_300", "last_name": "LAST_300"},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": "FIRST_200", "last_name": "LAST_200"},
			},
		},
	}

	// Read the JSON file with the warehouse settings.
	settings, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("cannot open the path %q specified in the %s environment variable: %s", settingsFile, settingsEnvKey, err)
	}

	// Open the PostgreSQL warehouse.
	wh, err := postgresql.New(&meergo.WarehouseConfig{
		Settings: settings,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Determine if the warehouse can be initialized (returning an error
	// otherwise), then initialize it.
	err = wh.CanInitialize(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = wh.Initialize(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Create the necessary columns on the warehouse.
	var ops []meergo.AlterSchemaOperation
	for _, c := range columns {
		if c.Name == "email" {
			// TODO(Gianluca): the "email" column is omitted as it is already
			// part of the default schema. This can be reviewed in relation to
			// issue https://github.com/meergo/meergo/issues/1075.
			continue
		}
		ops = append(ops, meergo.AlterSchemaOperation{Operation: meergo.OperationAddColumn, Column: c.Name, Type: c.Type})
	}
	err = wh.AlterSchema(ctx, columns, ops)
	if err != nil {
		t.Fatal(err)
	}

	mergeColumns := identitiesMergeColumns(columnByName)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// Truncate the existing user identities.
			//
			// TODO(Gianluca): how should the drivers expose the table names? We
			// have an issue where we discuss this (https://github.com/meergo/meergo/issues/928).
			err = wh.Truncate(ctx, "_user_identities")
			if err != nil {
				t.Fatal(err)
			}

			// Merge the test's user identities on the warehouse.
			var rows []map[string]any
			validatePrimarySources(t, test.primarySources)
			for _, user := range test.identities {
				validateIdentity(t, user)
				row := map[string]any{
					"__action__":           user.action,
					"__is_anonymous__":     user.isAnonymous,
					"__identity_id__":      user.id,
					"__connection__":       user.connection,
					"__anonymous_ids__":    toSliceAny(user.anonymousIDs),
					"__last_change_time__": time.Now().UTC(),
					"__execution__":        1,
				}
				for k, v := range user.properties {
					row[k] = v
				}
				rows = append(rows, row)
			}
			err = wh.MergeIdentities(ctx, mergeColumns, rows)
			if err != nil {
				t.Fatal(err)
			}

			// Resolve the identities.
			var identifiers []meergo.Column
			for _, id := range test.identifiers {
				identifiers = append(identifiers, columnByName[id])
			}
			err = wh.ResolveIdentities(ctx, identifiers, columns, test.primarySources)
			if err != nil {
				t.Fatal(err)
			}

			// Read the users from the warehouse and check that they match with
			// the expected ones.
			var gotUsers []map[string]any
			{
				query := meergo.RowQuery{
					Columns: columns,
					Table:   "users",
					OrderBy: columnByName["email"],
				}
				r, _, err := wh.Query(ctx, query, true)
				if err != nil {
					t.Fatal(err)
				}
				defer r.Close()
				for r.Next() {
					var email, firstName, lastName any
					err := r.Scan(&email, &firstName, &lastName)
					if err != nil {
						t.Fatal(err)
					}
					user := make(map[string]any, 3)
					user["email"], err = wh.Normalize("email", columnByName["email"].Type, email, true)
					if err != nil {
						t.Fatal(err)
					}
					user["first_name"], err = wh.Normalize("first_name", columnByName["first_name"].Type, firstName, true)
					if err != nil {
						t.Fatal(err)
					}
					user["last_name"], err = wh.Normalize("last_name", columnByName["last_name"].Type, lastName, true)
					if err != nil {
						t.Fatal(err)
					}
					gotUsers = append(gotUsers, user)
				}
				if err := r.Err(); err != nil {
					t.Fatal(err)
				}
				err = r.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(test.expectedUsers, gotUsers) {
				t.Fatalf("\nexpected users:\n\t%v\ngot:\n\t%v", test.expectedUsers, gotUsers)
			}
		})
	}

}

func toSliceAny[T any](s []T) []any {
	sa := make([]any, len(s))
	for i := range s {
		sa[i] = s[i]
	}
	return sa
}

func validateIdentity(t *testing.T, id identity) {
	fatal := func(format string, a ...any) {
		t.Fatalf("the test is invalid because an identity is not defined correctly: %s", fmt.Sprintf(format, a...))
	}
	// connection.
	if id.connection < 100 || id.connection > 900 {
		fatal("connection ID must be a multiple of 100 in range [100, 900]")
	}
	if id.connection%100 != 0 {
		fatal("connection ID must be a multiple of 100 in range [100, 900]")
	}
	// action.
	if id.action < 1 || id.action > 9 {
		fatal("action ID must be in range [1, 9]")
	}
	// id.
	if id.id == "" {
		fatal("identity ID cannot be empty")
	}
	// properties.
	columnsNames := slices.Collect(maps.Keys(columnByName))
	propNames := slices.Collect(maps.Keys(id.properties))
	slices.Sort(columnsNames)
	slices.Sort(propNames)
	if !reflect.DeepEqual(propNames, columnsNames) {
		fatal("expected values for properties %v, got values for %v instead", columnsNames, propNames)
	}
}

func validatePrimarySources(t *testing.T, primarySources map[string]int) {
	for c := range primarySources {
		if _, ok := columnByName[c]; !ok {
			t.Fatalf("the test is invalid because the primary sources refer to a column %q which does not exist in the tests", c)
		}
	}
}
