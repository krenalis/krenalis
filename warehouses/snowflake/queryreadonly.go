// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"

	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/snowflake/internal/readonlysql"
)

// QueryReadOnly executes a query accepted as read-only and returns the results
// and the number of columns in each row.
//
// Safety depends on deployment assumptions in addition to SQL validation:
//   - The MCP credentials must have passed CheckReadOnlyAccess before use.
//   - The active Snowflake role hierarchy must expose only the expected
//     read-only surface.
//
// Snowflake role privileges are the database-enforced runtime protection.
func (warehouse *Snowflake) QueryReadOnly(ctx context.Context, query string) (warehouses.Rows, int, error) {
	if err := readonlysql.ValidateReadOnly(query); err != nil {
		return nil, 0, err
	}
	return warehouse.RawQuery(ctx, query)
}
