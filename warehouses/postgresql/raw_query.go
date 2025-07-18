//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package postgresql

import "context"

// RawQuery executes a query and returns the results as [][]any.
// TODO(Gianluca): for a spec about returned values, see https://github.com/meergo/meergo/issues/1666.
func (warehouse *PostgreSQL) RawQuery(ctx context.Context, query string) ([][]any, error) {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result [][]any
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
