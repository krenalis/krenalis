//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/testimages"
	"github.com/meergo/meergo/types"

	_ "github.com/meergo/meergo/warehouses" // for registering warehouses.

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// This file contains tests on Identity Resolution. These tests are executed on
// the registered data warehouses, provided that the environment variables:
//
//      MEERGO_TEST_PATH_WAREHOUSE_<warehouse-name>
//
// are set for the corresponding data warehouse and point to JSON files
// containing the warehouse settings.
//
// WARNING: the warehouses must be empty, as the tests will initialize them.

var columns = []meergo.Column{
	{Name: "email", Type: types.Text(), Nullable: true},
	{Name: "first_name", Type: types.Text(), Nullable: true},
	{Name: "last_name", Type: types.Text(), Nullable: true},
	{Name: "notes", Type: types.Array(types.Text()), Nullable: true},
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
	anonymousIds []string
	properties   map[string]any
}

// TestWarehousesIdentityResolution tests the Identity Resolution.
func TestWarehousesIdentityResolution(t *testing.T) {

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
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
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
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 100,
					action:     2,
					id:         "c@d",
					properties: map[string]any{"email": "c@d", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				{"email": "c@d", "first_name": nil, "last_name": nil, "notes": nil},
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
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 100,
					action:     2,
					id:         "abcd",
					properties: map[string]any{"email": "c@d", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "c@d", "first_name": nil, "last_name": nil, "notes": nil},
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
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 200,
					action:     2,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
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
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 200,
					action:     2,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
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
					properties: map[string]any{"email": nil, "first_name": nil, "last_name": "Manzarek", "notes": nil},
				},
				{
					connection: 200,
					action:     2,
					id:         "c@d",
					properties: map[string]any{"email": nil, "first_name": nil, "last_name": "Manzarek", "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": nil, "first_name": nil, "last_name": "Manzarek", "notes": nil},
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
					properties: map[string]any{"email": nil, "first_name": nil, "last_name": "Manzarek", "notes": nil},
				},
				{
					connection: 200,
					action:     2,
					id:         "c@d",
					properties: map[string]any{"email": nil, "first_name": "Ray", "last_name": "Manzarek", "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": nil, "first_name": "Ray", "last_name": "Manzarek", "notes": nil},
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
					properties: map[string]any{"email": "a@b", "first_name": "a", "last_name": nil, "notes": nil},
				},
				{
					connection: 200,
					action:     2,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": "b", "last_name": nil, "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": "b", "last_name": nil, "notes": nil},
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
					properties:  map[string]any{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
				},
				{
					connection:  100,
					action:      2,
					isAnonymous: true,
					id:          "46d59a94-6032-46d9-87f2-f006af0156f0",
					properties:  map[string]any{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
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
					properties:  map[string]any{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
				},
				{
					connection: 100,
					action:     2,
					id:         "46d59a94-6032-46d9-87f2-f006af0156f0",
					properties: map[string]any{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
				{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
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
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
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
					properties: map[string]any{"email": "a@b", "first_name": "FIRST_100", "last_name": "LAST_100", "notes": nil},
				},
				{
					connection: 200,
					action:     2,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": "FIRST_200", "last_name": "LAST_200", "notes": nil},
				},
				{
					connection: 300,
					action:     3,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": "FIRST_300", "last_name": "LAST_300", "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": "FIRST_200", "last_name": "LAST_200", "notes": nil},
			},
		},
		{
			name:        "Array - just one identity",
			identifiers: []string{},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA"}},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA"}},
			},
		},
		{
			name:        "Array - merging three identities",
			identifiers: []string{"email"},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA"}},
				},
				{
					connection: 200,
					action:     2,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"ZZZ"}},
				},
				{
					connection: 200,
					action:     2,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA", "BBB", "ZZZ"}},
			},
		},
		{
			name:        "Array - merging four identities, two by two (with duplicated values)",
			identifiers: []string{"email"},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "A",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA"}},
				},
				{
					connection: 200,
					action:     2,
					id:         "B",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"ZZZ"}},
				},
				{
					connection: 300,
					action:     3,
					id:         "C",
					properties: map[string]any{"email": "c@d", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
				},
				{
					connection: 400,
					action:     4,
					id:         "D",
					properties: map[string]any{"email": "c@d", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA", "ZZZ"}},
				{"email": "c@d", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
			},
		},
		{
			name:        "Array - handling of null values (1)",
			identifiers: []string{"email"},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA"}},
				},
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"ZZZ"}},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA", "ZZZ"}},
			},
		},
		{
			name:        "Array - handling of null values (2)",
			identifiers: []string{"email"},
			identities: []identity{
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
				},
				{
					connection: 100,
					action:     1,
					id:         "a@b",
					properties: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedUsers: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
			},
		},
	}

	// Run the tests on every warehouse type.
	warehouseTypes := meergo.WarehouseDrivers()
	if len(warehouseTypes) == 0 {
		t.Fatal("there are no warehouse drivers. Missing warehouse drivers import in test file?")
	}
	for _, warehouseType := range warehouseTypes {
		t.Run(warehouseType.Name, func(t *testing.T) {
			var settings []byte
			switch warehouseType.Name {
			case "PostgreSQL":
				const (
					database = "test_meergo"
					username = "test_meergo"
					password = "test_meergo"
				)
				ctx := context.Background()
				postgresContainer, err := postgres.Run(ctx,
					testimages.PostgreSQL,
					postgres.WithDatabase(database),
					postgres.WithUsername(username),
					postgres.WithPassword(password),
					testcontainers.WithWaitStrategy(
						wait.ForLog("database system is ready to accept connections").
							WithOccurrence(2).
							WithStartupTimeout(60*time.Second)),
				)
				defer func() {
					if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
						t.Error(err)
					}
				}()
				if err != nil {
					t.Fatal(err)
				}
				host, err := postgresContainer.Host(ctx)
				if err != nil {
					t.Fatal(err)
				}
				port, err := postgresContainer.MappedPort(ctx, "5432/tcp")
				if err != nil {
					t.Fatal(err)
				}

				settings, err = json.Marshal(map[string]any{
					"Host":     host,
					"Port":     port.Int(),
					"Username": username,
					"Password": password,
					"Database": database,
					"Schema":   "public",
				})
				if err != nil {
					t.Fatal(err)
				}
			case "Snowflake":
				// Read the warehouse settings, if the env variable is set,
				// otherwise skip this warehouse.
				settingsEnvKey := fmt.Sprintf("MEERGO_TEST_PATH_WAREHOUSE_%s", strings.ToUpper(warehouseType.Name))
				settingsFile, ok := os.LookupEnv(settingsEnvKey)
				if !ok {
					t.Skipf("the %s environment variable is not present", settingsEnvKey)
				}
				// Read the JSON file with the warehouse settings.
				var err error
				settings, err = os.ReadFile(settingsFile)
				if err != nil {
					t.Fatalf("cannot open the path %q specified in the %s environment variable: %s", settingsFile, settingsEnvKey, err)
				}
			default:
				panic(fmt.Sprintf("unsupported data warehouse %q", warehouseType.Name))
			}

			// Open the warehouse.
			wh, err := warehouseType.New(&meergo.WarehouseConfig{
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
			err = wh.Initialize(ctx, columns)
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
							"__anonymous_ids__":    toSliceAny(user.anonymousIds),
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
							OrderBy: []meergo.Column{columnByName["email"]},
						}
						r, _, err := wh.Query(ctx, query, true)
						if err != nil {
							t.Fatal(err)
						}
						defer r.Close()
						row := make([]any, len(columns))
						for r.Next() {
							err := r.Scan(row...)
							if err != nil {
								t.Fatal(err)
							}
							user := make(map[string]any, len(columns))
							for i, c := range columns {
								user[c.Name] = row[i]
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
					// The returned users are sorted solely by email, as it is
					// only possible to sort users by one property. Therefore,
					// in the case of users with the same email but with
					// different values for first_name, etc..., the tests may
					// randomly fail based on how Meergo returned them. For this
					// reason, here the users are sorted based on all their Text
					// properties, in ascending order.
					slices.SortFunc(gotUsers, func(u1, u2 map[string]any) int {
						for _, c := range columns {
							if c.Type.Kind() == types.TextKind {
								v1, _ := u1[c.Name].(string)
								v2, _ := u2[c.Name].(string)
								if v1 != v2 {
									return cmp.Compare(v1, v2)
								}
							}
						}
						return 0
					})
					if !reflect.DeepEqual(test.expectedUsers, gotUsers) {
						t.Fatalf("\nexpected users:\n\t%v\ngot:\n\t%v", test.expectedUsers, gotUsers)
					}
				})
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
