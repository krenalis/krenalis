// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"

	"github.com/krenalis/krenalis/warehouses"
)

// RawQuery executes a query and returns the results and the number of columns
// in each row.
func (warehouse *Snowflake) RawQuery(ctx context.Context, query string) (warehouses.Rows, int, error) {
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return nil, 0, snowflake(err)
	}
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
