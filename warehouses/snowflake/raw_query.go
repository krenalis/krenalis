//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package snowflake

import (
	"context"

	"github.com/meergo/meergo"
)

// RawQuery executes a query and returns the results and the number of columns
// in each row.
func (warehouse *Snowflake) RawQuery(ctx context.Context, query string) (meergo.Rows, int, error) {
	// TODO(Gianluca): this should be tested on a Snowflake warehouse. See
	// https://github.com/meergo/meergo/issues/1665.
	db := warehouse.openDB()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, 0, snowflake(err)
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, err
	}
	return rows, len(columns), nil
}
