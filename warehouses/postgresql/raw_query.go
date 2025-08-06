//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package postgresql

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/meergo/meergo"
)

// RawQuery executes a query and returns the results and the number of columns
// in each row.
func (warehouse *PostgreSQL) RawQuery(ctx context.Context, query string) (meergo.Rows, int, error) {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	columnCount := len(rows.FieldDescriptions())
	return rawQueryRows{pgxRows: rows}, columnCount, nil
}

// rawQueryRows is a minimal wrapper for 'pgx.Rows' to change the signature of
// the 'Close' method (which must return an error) and make it compatible with
// 'meergo.Rows'.
type rawQueryRows struct {
	pgxRows pgx.Rows
}

func (r rawQueryRows) Close() error {
	r.pgxRows.Close()
	return nil
}

func (r rawQueryRows) Err() error {
	return r.pgxRows.Err()
}

func (r rawQueryRows) Next() bool {
	return r.pgxRows.Next()
}

func (r rawQueryRows) Scan(dest ...any) error {
	return r.pgxRows.Scan(dest...)
}
