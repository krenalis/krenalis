// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	// Import warehouse platforms for TestWarehousesIdentityResolution.
	_ "github.com/krenalis/krenalis/warehouses/postgresql"
	_ "github.com/krenalis/krenalis/warehouses/snowflake"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// This file contains tests on Identity Resolution. These tests are executed on
// the registered data warehouses, provided that the environment variables:
//
//      KRENALIS_TEST_PATH_WAREHOUSE_<warehouse-name>
//
// are set for the corresponding data warehouse and point to JSON files
// containing the warehouse settings.
//
// WARNING: the warehouses must be empty, as the tests will initialize them.

var columns = []warehouses.Column{
	{Name: "email", Type: types.String(), Nullable: true},
	{Name: "first_name", Type: types.String(), Nullable: true},
	{Name: "last_name", Type: types.String(), Nullable: true},
	{Name: "notes", Type: types.Array(types.String()), Nullable: true},
}

var columnByName map[string]warehouses.Column

func init() {
	columnByName = make(map[string]warehouses.Column, len(columns))
	for _, c := range columns {
		columnByName[c.Name] = c
	}
}

type identity struct {
	connection   int // a multiple of 100, from 100 to 900 (included)
	pipeline     int // can be 1, 2 ... 9
	id           string
	isAnonymous  bool
	anonymousIDs []string
	attributes   map[string]any
}

// TestWarehousesIdentityResolution tests the Identity Resolution.
func TestWarehousesIdentityResolution(t *testing.T) {

	tests := []struct {
		name             string
		identifiers      []string
		primarySources   map[string]int
		identities       []identity
		expectedProfiles []map[string]any
	}{
		{
			name:        "One identity, no identifiers",
			identifiers: []string{},
			identities: []identity{
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
			},
		},
		{
			name:        "Two identities from the same connection (different ID), no identifiers",
			identifiers: []string{},
			identities: []identity{
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 100,
					pipeline:   2,
					id:         "c@d",
					attributes: map[string]any{"email": "c@d", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
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
					pipeline:   1,
					id:         "abcd",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 100,
					pipeline:   2,
					id:         "abcd",
					attributes: map[string]any{"email": "c@d", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": "c@d", "first_name": nil, "last_name": nil, "notes": nil},
			},
		},
		{
			name:        "Two identities from two different connections, no identifiers",
			identifiers: []string{},
			identities: []identity{
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 200,
					pipeline:   2,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
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
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 200,
					pipeline:   2,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
			},
		},
		{
			name:        "Two identities from two different connections, matching for an identifier with second-level priority, previous identifiers are both nil",
			identifiers: []string{"email", "last_name"},
			identities: []identity{
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": nil, "first_name": nil, "last_name": "Manzarek", "notes": nil},
				},
				{
					connection: 200,
					pipeline:   2,
					id:         "c@d",
					attributes: map[string]any{"email": nil, "first_name": nil, "last_name": "Manzarek", "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": nil, "first_name": nil, "last_name": "Manzarek", "notes": nil},
			},
		},
		{
			name:        "Two identities from two different connections, matching for an identifier with second-level priority, previous identifiers are nil and not nil",
			identifiers: []string{"email", "last_name"},
			identities: []identity{
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": nil, "first_name": nil, "last_name": "Manzarek", "notes": nil},
				},
				{
					connection: 200,
					pipeline:   2,
					id:         "c@d",
					attributes: map[string]any{"email": nil, "first_name": "Ray", "last_name": "Manzarek", "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": nil, "first_name": "Ray", "last_name": "Manzarek", "notes": nil},
			},
		},
		{
			name:        "Two identities from two different connections, two identifiers that merge them",
			identifiers: []string{"email", "first_name"},
			identities: []identity{
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": "a", "last_name": nil, "notes": nil},
				},
				{
					connection: 200,
					pipeline:   2,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": "b", "last_name": nil, "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": "a@b", "first_name": "b", "last_name": nil, "notes": nil},
			},
		},
		{
			name:        "Merging two anonymous identities from the same connection",
			identifiers: []string{},
			identities: []identity{
				{
					connection:  100,
					pipeline:    1,
					isAnonymous: true,
					id:          "46d59a94-6032-46d9-87f2-f006af0156f0",
					attributes:  map[string]any{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
				},
				{
					connection:  100,
					pipeline:    2,
					isAnonymous: true,
					id:          "46d59a94-6032-46d9-87f2-f006af0156f0",
					attributes:  map[string]any{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
			},
		},
		{
			name:        "Two identities not merged as one is anonymous and one is not",
			identifiers: []string{},
			identities: []identity{
				{
					connection:  100,
					pipeline:    1,
					isAnonymous: true,
					id:          "46d59a94-6032-46d9-87f2-f006af0156f0",
					attributes:  map[string]any{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
				},
				{
					connection: 100,
					pipeline:   2,
					id:         "46d59a94-6032-46d9-87f2-f006af0156f0",
					attributes: map[string]any{"email": nil, "first_name": "Luke", "last_name": nil, "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
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
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
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
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": "FIRST_100", "last_name": "LAST_100", "notes": nil},
				},
				{
					connection: 200,
					pipeline:   2,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": "FIRST_200", "last_name": "LAST_200", "notes": nil},
				},
				{
					connection: 300,
					pipeline:   3,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": "FIRST_300", "last_name": "LAST_300", "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": "a@b", "first_name": "FIRST_200", "last_name": "LAST_200", "notes": nil},
			},
		},
		{
			name:        "Array - just one identity",
			identifiers: []string{},
			identities: []identity{
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA"}},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA"}},
			},
		},
		{
			name:        "Array - merging three identities",
			identifiers: []string{"email"},
			identities: []identity{
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA"}},
				},
				{
					connection: 200,
					pipeline:   2,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"ZZZ"}},
				},
				{
					connection: 200,
					pipeline:   2,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA", "BBB", "ZZZ"}},
			},
		},
		{
			name:        "Array - merging four identities, two by two (with duplicated values)",
			identifiers: []string{"email"},
			identities: []identity{
				{
					connection: 100,
					pipeline:   1,
					id:         "A",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA"}},
				},
				{
					connection: 200,
					pipeline:   2,
					id:         "B",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"ZZZ"}},
				},
				{
					connection: 300,
					pipeline:   3,
					id:         "C",
					attributes: map[string]any{"email": "c@d", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
				},
				{
					connection: 400,
					pipeline:   4,
					id:         "D",
					attributes: map[string]any{"email": "c@d", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
				},
			},
			expectedProfiles: []map[string]any{
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
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA"}},
				},
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"ZZZ"}},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"AAA", "ZZZ"}},
			},
		},
		{
			name:        "Array - handling of null values (2)",
			identifiers: []string{"email"},
			identities: []identity{
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
				},
				{
					connection: 100,
					pipeline:   1,
					id:         "a@b",
					attributes: map[string]any{"email": "a@b", "first_name": nil, "last_name": nil, "notes": nil},
				},
			},
			expectedProfiles: []map[string]any{
				{"email": "a@b", "first_name": nil, "last_name": nil, "notes": []any{"BBB"}},
			},
		},
	}

	// Run the tests on PostgreSQL and Snowflake warehouse platforms.
	platforms := warehouses.Platforms()
	if len(platforms) == 0 {
		t.Fatal("there are no warehouse platform. Missing warehouse platforms import in test file?")
	}
	for _, platform := range platforms {
		t.Run(platform.Name, func(t *testing.T) {
			var settings json.Value
			switch platform.Name {
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
					"host":     host,
					"port":     port.Int(),
					"username": username,
					"password": password,
					"database": database,
					"schema":   "public",
				})
				if err != nil {
					t.Fatal(err)
				}
			case "Snowflake":
				// Read the warehouse settings, if the env variable is set,
				// otherwise skip this warehouse.
				settingsFile, ok := os.LookupEnv("KRENALIS_TEST_PATH_WAREHOUSE_SNOWFLAKE")
				if !ok {
					t.Skipf("the KRENALIS_TEST_PATH_WAREHOUSE_SNOWFLAKE environment variable is not present")
				}
				// Read the JSON file with the warehouse settings.
				var err error
				settings, err = os.ReadFile(settingsFile)
				if err != nil {
					t.Fatalf("cannot open the path %q specified in the KRENALIS_TEST_PATH_WAREHOUSE_SNOWFLAKE environment variable: %s", settingsFile, err)
				}
			default:
				panic(fmt.Sprintf("unsupported data warehouse %q", platform.Name))
			}

			// Open the warehouse.
			wh, err := platform.New(&warehouses.Config{
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

					// Truncate the existing identities.
					//
					// TODO(Gianluca): how should the platforms expose the table names? We
					// have an issue where we discuss this (https://github.com/krenalis/krenalis/issues/928).
					err = wh.Truncate(ctx, "meergo_identities")
					if err != nil {
						t.Fatal(err)
					}

					// Merge the test's identities on the warehouse.
					var rows []map[string]any
					validatePrimarySources(t, test.primarySources)
					for _, profile := range test.identities {
						validateIdentity(t, profile)
						// Sleep for 1 millisecond to ensure that
						// timestamps are generated incrementally. This
						// is not necessary on Linux, where timestamps
						// have a nanosecond precision, but is required
						// on Windows, where the timestamps precision is
						// lower and may happen that two timestamps of
						// two different identities have the same value,
						// making the test fail.
						time.Sleep(1 * time.Millisecond)
						row := map[string]any{
							"_pipeline":      profile.pipeline,
							"_is_anonymous":  profile.isAnonymous,
							"_identity_id":   profile.id,
							"_connection":    profile.connection,
							"_anonymous_ids": toSliceAny(profile.anonymousIDs),
							"_updated_at":    time.Now().UTC(),
							"_run":           1,
						}
						maps.Copy(row, profile.attributes)
						rows = append(rows, row)
					}
					err = wh.MergeIdentities(ctx, mergeColumns, rows)
					if err != nil {
						t.Fatal(err)
					}

					// Resolve the identities.
					var identifiers []warehouses.Column
					for _, id := range test.identifiers {
						identifiers = append(identifiers, columnByName[id])
					}
					opID, err := uuid.NewUUID()
					if err != nil {
						t.Fatal(err)
					}
					// Call ResolveIdentities several times, just to do a
					// minimal idempotency test.
					for range 5 {
						err = wh.ResolveIdentities(ctx, opID.String(), identifiers, columns, test.primarySources)
						if err != nil {
							t.Fatal(err)
						}
					}

					// Read the profiles from the warehouse and check that they match with
					// the expected ones.
					var gotProfiles []map[string]any
					{
						query := warehouses.RowQuery{
							Columns: columns,
							Table:   "profiles",
							OrderBy: []warehouses.Column{columnByName["email"]},
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
							profile := make(map[string]any, len(columns))
							for i, c := range columns {
								profile[c.Name] = row[i]
							}
							gotProfiles = append(gotProfiles, profile)
						}
						if err := r.Err(); err != nil {
							t.Fatal(err)
						}
						err = r.Close()
						if err != nil {
							t.Fatal(err)
						}
					}
					// The returned profiles are sorted solely by email, as it is
					// only possible to sort profiles by one property. Therefore,
					// in the case of profiles with the same email but with
					// different values for first_name, etc..., the tests may
					// randomly fail based on how Krenalis returned them. For this
					// reason, here the profiles are sorted based on all their string
					// properties, in ascending order.
					slices.SortFunc(gotProfiles, func(u1, u2 map[string]any) int {
						for _, c := range columns {
							if c.Type.Kind() == types.StringKind {
								v1, _ := u1[c.Name].(string)
								v2, _ := u2[c.Name].(string)
								if v1 != v2 {
									return cmp.Compare(v1, v2)
								}
							}
						}
						return 0
					})
					if !reflect.DeepEqual(test.expectedProfiles, gotProfiles) {
						t.Fatalf("\nexpected profiles:\n\t%v\ngot:\n\t%v", test.expectedProfiles, gotProfiles)
					}
				})
			}
		})
	}

}

// identitiesMergeColumns returns the columns to be used during the identities
// merge operation, both when importing in batch.
func identitiesMergeColumns(iwColumns map[string]warehouses.Column) []warehouses.Column {
	columns := make([]warehouses.Column, 7+len(iwColumns))
	columns[0] = warehouses.Column{Name: "_pipeline", Type: types.Int(32)}
	columns[1] = warehouses.Column{Name: "_is_anonymous", Type: types.Boolean()}
	columns[2] = warehouses.Column{Name: "_identity_id", Type: types.String()}
	columns[3] = warehouses.Column{Name: "_connection", Type: types.Int(32)}
	columns[4] = warehouses.Column{Name: "_anonymous_ids", Type: types.Array(types.String()), Nullable: true}
	columns[5] = warehouses.Column{Name: "_updated_at", Type: types.DateTime()}
	columns[6] = warehouses.Column{Name: "_run", Type: types.Int(32), Nullable: true}
	i := 7
	for _, column := range iwColumns {
		columns[i] = column
		i++
	}
	slices.SortFunc(columns[7:], func(a, b warehouses.Column) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return columns
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
	// pipeline.
	if id.pipeline < 1 || id.pipeline > 9 {
		fatal("pipeline ID must be in range [1, 9]")
	}
	// id.
	if id.id == "" {
		fatal("identity ID cannot be empty")
	}
	// properties.
	columns := slices.Collect(maps.Keys(columnByName))
	properties := slices.Collect(maps.Keys(id.attributes))
	slices.Sort(columns)
	slices.Sort(properties)
	if !reflect.DeepEqual(properties, columns) {
		fatal("expected values for properties %v, got values for %v instead", columns, properties)
	}
}

func validatePrimarySources(t *testing.T, primarySources map[string]int) {
	for c := range primarySources {
		if _, ok := columnByName[c]; !ok {
			t.Fatalf("the test is invalid because the primary sources refer to a column %q which does not exist in the tests", c)
		}
	}
}
