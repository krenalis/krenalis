//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package snowflake

import "context"

// RawQuery executes a query and returns the results as [][]any.
// TODO(Gianluca): for a spec about returned values, see https://github.com/meergo/meergo/issues/1666.
func (warehouse *Snowflake) RawQuery(ctx context.Context, query string) ([][]any, error) {
	// TODO(Gianluca): this should be tested on a Snowflake warehouse. See
	// https://github.com/meergo/meergo/issues/1665.
	db := warehouse.openDB()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, snowflake(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, snowflake(err)
	}
	var result [][]any
	for rows.Next() {
		row := make([]any, len(columns))
		for i := range row {
			var v any
			row[i] = &v
		}
		err := rows.Scan(row...)
		if err != nil {
			return nil, snowflake(err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, snowflake(err)
	}
	return result, nil
}
