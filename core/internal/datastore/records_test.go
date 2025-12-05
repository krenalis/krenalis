// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/test/testimages"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"

	// Import PostgreSQL warehouse platform for Test_Records.
	_ "github.com/meergo/meergo/warehouses/postgresql"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDatabase = "meergo"
	testUser     = "meergo"
	testPassword = "meergo"
)

func Test_Records(t *testing.T) {

	// Run the PostgreSQL container.
	ctx := context.Background()
	postgresContainer, err := postgres.Run(ctx,
		testimages.PostgreSQL,
		postgres.WithDatabase(testDatabase),
		postgres.WithUsername(testUser),
		postgres.WithPassword(testPassword),
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
	testHost, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	testPort, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	settings, err := json.Marshal(map[string]any{
		"Host":     testHost,
		"Port":     testPort.Int(),
		"Username": testUser,
		"Password": testPassword,
		"Database": testDatabase,
		"Schema":   "public",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Open the data warehouse.
	wh, err := warehouses.Registered("PostgreSQL").New(&warehouses.Config{
		Settings: settings,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wh.Close()

	profilesTable := warehouses.Table{
		Name: "meergo_profiles_0",
		Columns: []warehouses.Column{
			{Name: "__mpid__", Type: types.UUID()},
			{Name: "__last_change_time__", Type: types.DateTime()},
			{Name: "id", Type: types.String()},
			{Name: "other_id", Type: types.String()},
			{Name: "name", Type: types.String()},
			{Name: "age", Type: types.Int(8)},
		},
		Keys: []string{"__mpid__"},
	}

	destinationsUsersTable := warehouses.Table{
		Name: "meergo_destination_profiles",
		Columns: []warehouses.Column{
			{Name: "__pipeline__", Type: types.Int(32)},
			{Name: "__external_id__", Type: types.String()},
			{Name: "__out_matching_value__", Type: types.String()},
		},
		Keys: []string{"__pipeline__", "__external_id__"},
	}

	err = wh.Initialize(ctx, profilesTable.Columns[2:])
	if err != nil {
		t.Fatalf("cannot initialize the warehouse: %s", err)
	}

	now := time.Now().UTC()

	initUsers := [][]any{
		{"e5a5c059-bc78-4c9c-b4d1-e9fb187562b1", now, "1", "1", "Jake Thompson", 43},
		{"943a0a39-fd0b-4f7b-a113-59046fb8a511", now, "2", "2", "Emily Davis", 58},
		{"2a3654ca-a387-49c3-8eb8-8420ab8a7532", now, "3", "3", "Michael Carter", 31},
		{"243abf79-cbc3-4c6e-8739-e1406f2f6b51", now, "2", "2", "Sophia Harris", 19},
		{"445ab9fa-5689-4870-bc39-2d01c2a71b00", now, "6", "6", "Emily Johnson", 25},
		{"ce8f366d-7144-4ec0-96e7-d0dc35597c02", now, "7", "7", "James Williams", 77},
		{"a415976f-279e-4653-ab6a-64ea7f74e174", now, "7", "7", "Daniel Brown", 12},
	}
	err = wh.Merge(ctx, profilesTable, initUsers, nil)
	if err != nil {
		t.Fatalf("cannot merge profiles: %s", err)
	}

	const pipelineID = 623

	initDestinations := [][]any{
		{85, "Ex1", "1"},
		{85, "Ex2", "2"},
		{pipelineID, "Ex1", "1"},
		{pipelineID, "Ex2", "2"},
		{pipelineID, "Ex3", "2"},
		{pipelineID, "Ex4", "3"},
		{pipelineID, "Ex5", "3"},
		{pipelineID, "Ex6", "5"},
		{719, "034", "a"},
		{719, "089", "b"},
	}
	err = wh.Merge(ctx, destinationsUsersTable, initDestinations, nil)
	if err != nil {
		t.Fatalf("cannot merge profiles: %s", err)
	}

	tests := []struct {
		mode               state.ExportMode
		updateOnDuplicates bool
		expected           []Record
	}{
		{
			mode:               state.CreateOnly,
			updateOnDuplicates: false,
			expected: []Record{
				{ID: "445ab9fa-5689-4870-bc39-2d01c2a71b00", Attributes: map[string]any{"age": 25, "id": "6", "other": map[string]any{"id": "6"}, "name": "Emily Johnson"}},
				{ID: "a415976f-279e-4653-ab6a-64ea7f74e174", Attributes: map[string]any{"age": 12, "id": "7", "other": map[string]any{"id": "7"}, "name": "Daniel Brown"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "ce8f366d-7144-4ec0-96e7-d0dc35597c02", Attributes: map[string]any{"age": 77, "id": "7", "other": map[string]any{"id": "7"}, "name": "James Williams"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
			},
		},
		{
			mode:               state.CreateOnly,
			updateOnDuplicates: true,
			expected: []Record{
				{ID: "445ab9fa-5689-4870-bc39-2d01c2a71b00", Attributes: map[string]any{"age": 25, "id": "6", "other": map[string]any{"id": "6"}, "name": "Emily Johnson"}},
				{ID: "a415976f-279e-4653-ab6a-64ea7f74e174", Attributes: map[string]any{"age": 12, "id": "7", "other": map[string]any{"id": "7"}, "name": "Daniel Brown"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "ce8f366d-7144-4ec0-96e7-d0dc35597c02", Attributes: map[string]any{"age": 77, "id": "7", "other": map[string]any{"id": "7"}, "name": "James Williams"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
			},
		},
		{
			mode:               state.UpdateOnly,
			updateOnDuplicates: false,
			expected: []Record{
				{ID: "e5a5c059-bc78-4c9c-b4d1-e9fb187562b1", ExternalID: "Ex1", Attributes: map[string]any{"age": 43, "id": "1", "other": map[string]any{"id": "1"}, "name": "Jake Thompson"}},
				{ID: "243abf79-cbc3-4c6e-8739-e1406f2f6b51", ExternalID: "Ex2", Attributes: map[string]any{"age": 19, "id": "2", "other": map[string]any{"id": "2"}, "name": "Sophia Harris"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "943a0a39-fd0b-4f7b-a113-59046fb8a511", ExternalID: "Ex2", Attributes: map[string]any{"age": 58, "id": "2", "other": map[string]any{"id": "2"}, "name": "Emily Davis"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex4", Attributes: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}, Err: errors.New("duplicates found for the matching property id in the app users")},
			},
		},
		{
			mode:               state.UpdateOnly,
			updateOnDuplicates: true,
			expected: []Record{
				{ID: "e5a5c059-bc78-4c9c-b4d1-e9fb187562b1", ExternalID: "Ex1", Attributes: map[string]any{"age": 43, "id": "1", "other": map[string]any{"id": "1"}, "name": "Jake Thompson"}},
				{ID: "243abf79-cbc3-4c6e-8739-e1406f2f6b51", ExternalID: "Ex2", Attributes: map[string]any{"age": 19, "id": "2", "other": map[string]any{"id": "2"}, "name": "Sophia Harris"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "943a0a39-fd0b-4f7b-a113-59046fb8a511", ExternalID: "Ex2", Attributes: map[string]any{"age": 58, "id": "2", "other": map[string]any{"id": "2"}, "name": "Emily Davis"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex4", Attributes: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex5", Attributes: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}},
			},
		},
		{
			mode:               state.CreateOrUpdate,
			updateOnDuplicates: false,
			expected: []Record{
				{ID: "e5a5c059-bc78-4c9c-b4d1-e9fb187562b1", ExternalID: "Ex1", Attributes: map[string]any{"age": 43, "id": "1", "other": map[string]any{"id": "1"}, "name": "Jake Thompson"}},
				{ID: "243abf79-cbc3-4c6e-8739-e1406f2f6b51", ExternalID: "Ex2", Attributes: map[string]any{"age": 19, "id": "2", "other": map[string]any{"id": "2"}, "name": "Sophia Harris"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "943a0a39-fd0b-4f7b-a113-59046fb8a511", ExternalID: "Ex2", Attributes: map[string]any{"age": 58, "id": "2", "other": map[string]any{"id": "2"}, "name": "Emily Davis"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex4", Attributes: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}, Err: errors.New("duplicates found for the matching property id in the app users")},
				{ID: "445ab9fa-5689-4870-bc39-2d01c2a71b00", ExternalID: "", Attributes: map[string]any{"age": 25, "id": "6", "other": map[string]any{"id": "6"}, "name": "Emily Johnson"}},
				{ID: "a415976f-279e-4653-ab6a-64ea7f74e174", Attributes: map[string]any{"age": 12, "id": "7", "other": map[string]any{"id": "7"}, "name": "Daniel Brown"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "ce8f366d-7144-4ec0-96e7-d0dc35597c02", Attributes: map[string]any{"age": 77, "id": "7", "other": map[string]any{"id": "7"}, "name": "James Williams"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
			},
		},
		{
			mode:               state.CreateOrUpdate,
			updateOnDuplicates: true,
			expected: []Record{
				{ID: "e5a5c059-bc78-4c9c-b4d1-e9fb187562b1", ExternalID: "Ex1", Attributes: map[string]any{"age": 43, "id": "1", "other": map[string]any{"id": "1"}, "name": "Jake Thompson"}},
				{ID: "243abf79-cbc3-4c6e-8739-e1406f2f6b51", ExternalID: "Ex2", Attributes: map[string]any{"age": 19, "id": "2", "other": map[string]any{"id": "2"}, "name": "Sophia Harris"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "943a0a39-fd0b-4f7b-a113-59046fb8a511", ExternalID: "Ex2", Attributes: map[string]any{"age": 58, "id": "2", "other": map[string]any{"id": "2"}, "name": "Emily Davis"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex4", Attributes: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex5", Attributes: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}},
				{ID: "445ab9fa-5689-4870-bc39-2d01c2a71b00", ExternalID: "", Attributes: map[string]any{"age": 25, "id": "6", "other": map[string]any{"id": "6"}, "name": "Emily Johnson"}},
				{ID: "a415976f-279e-4653-ab6a-64ea7f74e174", Attributes: map[string]any{"age": 12, "id": "7", "other": map[string]any{"id": "7"}, "name": "Daniel Brown"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
				{ID: "ce8f366d-7144-4ec0-96e7-d0dc35597c02", Attributes: map[string]any{"age": 77, "id": "7", "other": map[string]any{"id": "7"}, "name": "James Williams"}, Err: errors.New("profile has the same «id» (the matching property) as other profiles selected for export")},
			},
		},
	}

	profileColumnByProperty := map[string]warehouses.Column{
		"__mpid__": {Name: "__mpid__", Type: types.UUID()},
		"id":       {Name: "id", Type: types.String()},
		"other.id": {Name: "other_id", Type: types.String()},
		"name":     {Name: "name", Type: types.String()},
		"age":      {Name: "age", Type: types.Int(8)},
	}

	for _, test := range tests {

		name := fmt.Sprintf("(%s,%t)", test.mode, test.updateOnDuplicates)
		t.Run(name, func(t *testing.T) {

			query := Query{
				table:      "profiles",
				Properties: []string{"id", "other.id", "name", "age"},
			}

			for _, inProperty := range []string{"id", "other.id"} {

				matching := &Matching{
					Pipeline:           pipelineID,
					InProperty:         inProperty,
					ExportMode:         test.mode,
					UpdateOnDuplicates: test.updateOnDuplicates,
				}

				r, err := records(ctx, wh, query, "__mpid__", profileColumnByProperty, true, matching)
				if err != nil {
					t.Fatalf("cannot read records: %s", err)
				}

				var got []Record
				for profile := range r.All(ctx) {
					got = append(got, profile)
				}
				if r.err != nil {
					t.Fatalf("cannot scan records: %s", err)
				}

				if len(test.expected) != len(got) {
					t.Fatalf("expected %d records, got %d", len(test.expected), len(got))
				}
				for i, e := range test.expected {
					g := got[i]
					if g.Err != nil && inProperty != "id" {
						g.Err = errors.New(strings.ReplaceAll(g.Err.Error(), inProperty, "id"))
					}
					if reflect.DeepEqual(e, g) {
						continue
					}
					if e.ID != g.ID {
						t.Fatalf("record %d, inProperty %q: expected ID %q, got %q", i, inProperty, e.ID, g.ID)
					}
					if e.ExternalID != g.ExternalID {
						t.Fatalf("record %d, inProperty %q: expected ExternalID %q, got %q", i, inProperty, e.ExternalID, g.ExternalID)
					}
					if e.Err != nil && g.Err == nil {
						t.Fatalf("record %d, inProperty %q: expected error %q, got no error", i, inProperty, e.Err)
					}
					if e.Err == nil && g.Err != nil {
						t.Fatalf("record %d, inProperty %q: expected no error, got error %q", i, inProperty, g.Err)
					}
					if e.Err != nil && g.Err != nil && e.Err.Error() != g.Err.Error() {
						t.Fatalf("record %d, inProperty %q: expected error %q, got error %q", i, inProperty, e.Err, g.Err)
					}
					t.Fatalf("record %d, inProperty %q: expected attributes %#v, got %#v", i, inProperty, e.Attributes, g.Attributes)
				}

			}
		})

	}

}
