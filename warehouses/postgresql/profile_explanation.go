// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"fmt"

	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/internal/profilefilter"
)

// ExplainProfileFilter returns every closed node result for one profile.
// It returns nil for a missing profile and no partial explanation on error.
func (warehouse *PostgreSQL) ExplainProfileFilter(ctx context.Context, filter warehouses.Filter, population warehouses.ProfileSource, profile string) ([]warehouses.FilterNodeOutcome, error) {
	query, err := profilefilter.BuildExplanation(filter, population, profile, profileFilterDialect())
	if err != nil {
		return nil, err
	}
	if !profileFilterVerified {
		return nil, fmt.Errorf("warehouses/postgresql: prepared Profiles filtering has not been verified")
	}
	values, err := query.Values()
	if err != nil {
		return nil, err
	}
	pool, _, err := warehouse.connectionPool(ctx, false)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, query.SQL, values...)
	if err != nil {
		return nil, err
	}
	if len(rows.FieldDescriptions()) != len(query.Columns) {
		rows.Close()
		err = rows.Err()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("warehouses/postgresql: profile outcome schema mismatch")
	}
	scanner := newScanner(query.Columns, rows)
	scanner.maxBytes = 64 * 1024
	return query.ReadExplanation(ctx, scanner, profile)
}
