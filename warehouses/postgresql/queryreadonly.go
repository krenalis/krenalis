// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"sync"

	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/postgresql/internal/readonlysql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// QueryReadOnly executes a read-only query and returns the results and the
// number of columns in each row.
//
// Safety depends on deployment assumptions in addition to SQL validation:
//   - The workspace warehouse user must have only read-only access.
//   - The workspace warehouse schema must not expose user-defined operators.
//   - The PostgreSQL standard_conforming_strings session setting must be on
//     (this is the default).
func (warehouse *PostgreSQL) QueryReadOnly(ctx context.Context, query string) (warehouses.Rows, int, error) {
	// Security is layered:
	// 1. QueryReadOnly rejects queries outside a supported read-only subset.
	// 2. The query runs inside a PostgreSQL read-only transaction.
	// 3. The PostgreSQL role is expected to have read-only privileges.
	// 4. The workspace warehouse schema is expected not to expose user-defined
	//    operators.

	if err := readonlysql.ValidateReadOnly(query); err != nil {
		return nil, 0, err
	}
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return nil, 0, err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, 0, err
	}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		conn.Release()
		return nil, 0, err
	}
	rows, err := tx.Query(ctx, query)
	if err != nil {
		_ = tx.Rollback(context.Background())
		conn.Release()
		return nil, 0, err
	}
	columnCount := len(rows.FieldDescriptions())

	return &queryReadOnlyRows{pgxRows: rows, tx: tx, conn: conn}, columnCount, nil
}

// queryReadOnlyRows is a minimal wrapper for pgx.Rows to change the signature
// of the Close method (which must return an error) and make it compatible with
// warehouses.Rows.
type queryReadOnlyRows struct {
	pgxRows    pgx.Rows
	tx         pgx.Tx
	conn       *pgxpool.Conn
	closeOnce  sync.Once
	closeError error
}

func (r *queryReadOnlyRows) Close() error {
	r.closeOnce.Do(func() {
		r.pgxRows.Close()
		if err := r.tx.Rollback(context.Background()); err != nil && err != pgx.ErrTxClosed {
			r.closeError = err
		}
		r.conn.Release()
	})
	return r.closeError
}

func (r *queryReadOnlyRows) Err() error {
	return r.pgxRows.Err()
}

func (r *queryReadOnlyRows) Next() bool {
	return r.pgxRows.Next()
}

func (r *queryReadOnlyRows) Scan(dest ...any) error {
	return r.pgxRows.Scan(dest...)
}
