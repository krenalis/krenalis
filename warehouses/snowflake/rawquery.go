// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"

	"github.com/meergo/meergo/warehouses"
)

// RawQuery executes a query and returns the results and the number of columns
// in each row.
func (warehouse *Snowflake) RawQuery(ctx context.Context, query string) (warehouses.Rows, int, error) {
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
