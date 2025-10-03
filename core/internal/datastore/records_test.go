//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package datastore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/testimages"

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
	wh, err := meergo.RegisteredWarehouseDriver("PostgreSQL").New(&meergo.WarehouseConfig{
		Settings: settings,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wh.Close()

	usersTable := meergo.Table{
		Name: "_users_0",
		Columns: []meergo.Column{
			{Name: "__id__", Type: types.UUID()},
			{Name: "__last_change_time__", Type: types.DateTime()},
			{Name: "id", Type: types.Text()},
			{Name: "other_id", Type: types.Text()},
			{Name: "name", Type: types.Text()},
			{Name: "age", Type: types.Int(8)},
		},
		Keys: []string{"__id__"},
	}

	destinationsUsersTable := meergo.Table{
		Name: "_destinations_users",
		Columns: []meergo.Column{
			{Name: "__action__", Type: types.Int(32)},
			{Name: "__external_id__", Type: types.Text()},
			{Name: "__out_matching_value__", Type: types.Text()},
		},
		Keys: []string{"__action__", "__external_id__"},
	}

	err = wh.Initialize(ctx, usersTable.Columns[2:])
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
	err = wh.Merge(ctx, usersTable, initUsers, nil)
	if err != nil {
		t.Fatalf("cannot merge users: %s", err)
	}

	const actionID = 623

	initDestinations := [][]any{
		{85, "Ex1", "1"},
		{85, "Ex2", "2"},
		{actionID, "Ex1", "1"},
		{actionID, "Ex2", "2"},
		{actionID, "Ex3", "2"},
		{actionID, "Ex4", "3"},
		{actionID, "Ex5", "3"},
		{actionID, "Ex6", "5"},
		{719, "034", "a"},
		{719, "089", "b"},
	}
	err = wh.Merge(ctx, destinationsUsersTable, initDestinations, nil)
	if err != nil {
		t.Fatalf("cannot merge users: %s", err)
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
				{ID: "445ab9fa-5689-4870-bc39-2d01c2a71b00", Properties: map[string]any{"age": 25, "id": "6", "other": map[string]any{"id": "6"}, "name": "Emily Johnson"}},
				{ID: "a415976f-279e-4653-ab6a-64ea7f74e174", Properties: map[string]any{"age": 12, "id": "7", "other": map[string]any{"id": "7"}, "name": "Daniel Brown"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "ce8f366d-7144-4ec0-96e7-d0dc35597c02", Properties: map[string]any{"age": 77, "id": "7", "other": map[string]any{"id": "7"}, "name": "James Williams"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
			},
		},
		{
			mode:               state.CreateOnly,
			updateOnDuplicates: true,
			expected: []Record{
				{ID: "445ab9fa-5689-4870-bc39-2d01c2a71b00", Properties: map[string]any{"age": 25, "id": "6", "other": map[string]any{"id": "6"}, "name": "Emily Johnson"}},
				{ID: "a415976f-279e-4653-ab6a-64ea7f74e174", Properties: map[string]any{"age": 12, "id": "7", "other": map[string]any{"id": "7"}, "name": "Daniel Brown"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "ce8f366d-7144-4ec0-96e7-d0dc35597c02", Properties: map[string]any{"age": 77, "id": "7", "other": map[string]any{"id": "7"}, "name": "James Williams"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
			},
		},
		{
			mode:               state.UpdateOnly,
			updateOnDuplicates: false,
			expected: []Record{
				{ID: "e5a5c059-bc78-4c9c-b4d1-e9fb187562b1", ExternalID: "Ex1", Properties: map[string]any{"age": 43, "id": "1", "other": map[string]any{"id": "1"}, "name": "Jake Thompson"}},
				{ID: "243abf79-cbc3-4c6e-8739-e1406f2f6b51", ExternalID: "Ex2", Properties: map[string]any{"age": 19, "id": "2", "other": map[string]any{"id": "2"}, "name": "Sophia Harris"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "943a0a39-fd0b-4f7b-a113-59046fb8a511", ExternalID: "Ex2", Properties: map[string]any{"age": 58, "id": "2", "other": map[string]any{"id": "2"}, "name": "Emily Davis"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex4", Properties: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}, Err: errors.New("duplicates found for the matching property id in the app users")},
			},
		},
		{
			mode:               state.UpdateOnly,
			updateOnDuplicates: true,
			expected: []Record{
				{ID: "e5a5c059-bc78-4c9c-b4d1-e9fb187562b1", ExternalID: "Ex1", Properties: map[string]any{"age": 43, "id": "1", "other": map[string]any{"id": "1"}, "name": "Jake Thompson"}},
				{ID: "243abf79-cbc3-4c6e-8739-e1406f2f6b51", ExternalID: "Ex2", Properties: map[string]any{"age": 19, "id": "2", "other": map[string]any{"id": "2"}, "name": "Sophia Harris"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "943a0a39-fd0b-4f7b-a113-59046fb8a511", ExternalID: "Ex2", Properties: map[string]any{"age": 58, "id": "2", "other": map[string]any{"id": "2"}, "name": "Emily Davis"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex4", Properties: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex5", Properties: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}},
			},
		},
		{
			mode:               state.CreateOrUpdate,
			updateOnDuplicates: false,
			expected: []Record{
				{ID: "e5a5c059-bc78-4c9c-b4d1-e9fb187562b1", ExternalID: "Ex1", Properties: map[string]any{"age": 43, "id": "1", "other": map[string]any{"id": "1"}, "name": "Jake Thompson"}},
				{ID: "243abf79-cbc3-4c6e-8739-e1406f2f6b51", ExternalID: "Ex2", Properties: map[string]any{"age": 19, "id": "2", "other": map[string]any{"id": "2"}, "name": "Sophia Harris"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "943a0a39-fd0b-4f7b-a113-59046fb8a511", ExternalID: "Ex2", Properties: map[string]any{"age": 58, "id": "2", "other": map[string]any{"id": "2"}, "name": "Emily Davis"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex4", Properties: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}, Err: errors.New("duplicates found for the matching property id in the app users")},
				{ID: "445ab9fa-5689-4870-bc39-2d01c2a71b00", ExternalID: "", Properties: map[string]any{"age": 25, "id": "6", "other": map[string]any{"id": "6"}, "name": "Emily Johnson"}},
				{ID: "a415976f-279e-4653-ab6a-64ea7f74e174", Properties: map[string]any{"age": 12, "id": "7", "other": map[string]any{"id": "7"}, "name": "Daniel Brown"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "ce8f366d-7144-4ec0-96e7-d0dc35597c02", Properties: map[string]any{"age": 77, "id": "7", "other": map[string]any{"id": "7"}, "name": "James Williams"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
			},
		},
		{
			mode:               state.CreateOrUpdate,
			updateOnDuplicates: true,
			expected: []Record{
				{ID: "e5a5c059-bc78-4c9c-b4d1-e9fb187562b1", ExternalID: "Ex1", Properties: map[string]any{"age": 43, "id": "1", "other": map[string]any{"id": "1"}, "name": "Jake Thompson"}},
				{ID: "243abf79-cbc3-4c6e-8739-e1406f2f6b51", ExternalID: "Ex2", Properties: map[string]any{"age": 19, "id": "2", "other": map[string]any{"id": "2"}, "name": "Sophia Harris"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "943a0a39-fd0b-4f7b-a113-59046fb8a511", ExternalID: "Ex2", Properties: map[string]any{"age": 58, "id": "2", "other": map[string]any{"id": "2"}, "name": "Emily Davis"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex4", Properties: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}},
				{ID: "2a3654ca-a387-49c3-8eb8-8420ab8a7532", ExternalID: "Ex5", Properties: map[string]any{"age": 31, "id": "3", "other": map[string]any{"id": "3"}, "name": "Michael Carter"}},
				{ID: "445ab9fa-5689-4870-bc39-2d01c2a71b00", ExternalID: "", Properties: map[string]any{"age": 25, "id": "6", "other": map[string]any{"id": "6"}, "name": "Emily Johnson"}},
				{ID: "a415976f-279e-4653-ab6a-64ea7f74e174", Properties: map[string]any{"age": 12, "id": "7", "other": map[string]any{"id": "7"}, "name": "Daniel Brown"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
				{ID: "ce8f366d-7144-4ec0-96e7-d0dc35597c02", Properties: map[string]any{"age": 77, "id": "7", "other": map[string]any{"id": "7"}, "name": "James Williams"}, Err: errors.New("user has the same «id» (the matching property) as other users selected for export")},
			},
		},
	}

	userColumnByProperty := map[string]meergo.Column{
		"__id__":   {Name: "__id__", Type: types.UUID()},
		"id":       {Name: "id", Type: types.Text()},
		"other.id": {Name: "other_id", Type: types.Text()},
		"name":     {Name: "name", Type: types.Text()},
		"age":      {Name: "age", Type: types.Int(8)},
	}

	for _, test := range tests {

		name := fmt.Sprintf("(%s,%t)", test.mode, test.updateOnDuplicates)
		t.Run(name, func(t *testing.T) {

			query := Query{
				table:      "users",
				Properties: []string{"id", "other.id", "name", "age"},
			}

			for _, inProperty := range []string{"id", "other.id"} {

				matching := &Matching{
					Action:             actionID,
					InProperty:         inProperty,
					ExportMode:         test.mode,
					UpdateOnDuplicates: test.updateOnDuplicates,
				}

				r, err := records(ctx, wh, query, "__id__", userColumnByProperty, true, matching)
				if err != nil {
					t.Fatalf("cannot read records: %s", err)
				}

				var got []Record
				for user := range r.All(ctx) {
					got = append(got, user)
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
					t.Fatalf("record %d, inProperty %q: expected properties %#v, got %#v", i, inProperty, e.Properties, g.Properties)
				}

			}
		})

	}

}
