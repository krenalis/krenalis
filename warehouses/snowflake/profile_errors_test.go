// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"errors"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestProfileSchemaFailures checks database/sql cursor failures before a schema
// mismatch without network access or credentials.
func TestProfileSchemaFailures(t *testing.T) {

	const profile = "00000000-0000-0000-0000-000000000001"
	population := warehouses.ProfileSource{Table: "profiles", Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}
	filter := warehouses.Filter{Root: &warehouses.RecordFilter{
		Expr: warehouses.NewBaseExpr(population.Key, warehouses.OpIs, profile),
	}}
	failure := errors.New("synthetic cursor close failure")
	for _, operation := range []struct {
		name string
		call func(context.Context, *Snowflake) (bool, error)
	}{
		{"membership", func(ctx context.Context, warehouse *Snowflake) (bool, error) {
			request := warehouses.ProfileOutcomeRequest{Population: population, Limit: 1}
			result, err := warehouse.ProfileOutcomes(ctx, filter, request)
			return result != nil, err
		}},
		{"explanation", func(ctx context.Context, warehouse *Snowflake) (bool, error) {
			result, err := warehouse.ExplainProfileFilter(ctx, filter, population, profile)
			return result != nil, err
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			for _, test := range []struct {
				name     string
				closeErr error
			}{
				{"close failure", failure},
				{"clean mismatch", nil},
			} {
				t.Run(test.name, func(t *testing.T) {
					db, queries := newCheckReadOnlyTestDB(t, []checkReadOnlyQuery{
						{cols: []string{"UNEXPECTED"}, closeErr: test.closeErr},
					})
					t.Cleanup(func() {
						err := db.Close()
						if err != nil {
							t.Errorf("expected test database cleanup, got %v", err)
						}
					})
					partial, err := operation.call(t.Context(), &Snowflake{db: db})
					if err != nil {
						if partial || len(*queries) != 1 || db.Stats().InUse != 0 {
							t.Fatalf("expected one closed query without partial data, got partial=%t, queries=%d, in_use=%d",
								partial, len(*queries), db.Stats().InUse)
						}
						if test.closeErr != nil {
							if !errors.Is(err, test.closeErr) {
								t.Fatalf("expected original cursor failure, got %v", err)
							}
							return
						}
						if err.Error() != "warehouses/snowflake: profile outcome schema mismatch" {
							t.Fatalf("expected schema mismatch without a cursor failure, got %v", err)
						}
						return
					}
					t.Fatal("expected cursor or schema failure, got success")
				})
			}
		})
	}

}
