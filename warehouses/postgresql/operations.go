//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/meergo/meergo"

	"github.com/jackc/pgx/v5"
)

// warehouseOperation represents an operation on the data warehouse.
type warehouseOperation string

const (
	alterUserColumns   warehouseOperation = "AlterUserColumns"
	identityResolution warehouseOperation = "IdentityResolution"
)

// startOperation starts an operation on the data warehouse, returning the ID of
// that operation.
//
// It is then the caller's responsibility to call the 'endOperation' method to
// mark the operation as completed.
//
// In the case that an AlterSchema operation is already in progress, the error
// ErrWarehouseAlterInProgress is returned; if an IdentityResolution operation
// is already in progress, the error ErrWarehouseIdentityResolutionInProgress
// is returned.
func (warehouse *PostgreSQL) startOperation(ctx context.Context, operation warehouseOperation) (int, error) {
	var opID int
	err := warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `LOCK TABLE "_operations"`)
		if err != nil {
			return err
		}
		var runningOp *warehouseOperation
		err = tx.QueryRow(ctx, `SELECT "operation" FROM "_operations" `+
			`WHERE "start_time" IS NOT NULL AND "end_time" IS NULL ORDER BY "id" DESC LIMIT 1`).Scan(&runningOp)
		if err != nil && err != pgx.ErrNoRows {
			return err
		}
		if runningOp != nil {
			switch *runningOp {
			case alterUserColumns:
				return meergo.ErrWarehouseAlterInProgress
			case identityResolution:
				return meergo.ErrWarehouseIdentityResolutionInProgress
			default:
				return fmt.Errorf("unexpected operation %q", *runningOp)
			}
		}
		err = tx.QueryRow(ctx, `INSERT INTO "_operations" ("operation", "start_time", "end_time") `+
			`VALUES ($1, (clock_timestamp() at time zone 'utc')::timestamp, NULL) RETURNING "id"`, operation).Scan(&opID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return opID, nil
}

// endOperation marks the operation with the given ID as completed, setting its
// endTime to the provided value. If the operation had already been completed
// previously, the call to this method is a no-op.
func (warehouse *PostgreSQL) endOperation(ctx context.Context, opID int, endTime time.Time) error {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `UPDATE "_operations" SET "end_time" = $1 WHERE "id" = $2 AND "end_time" IS NULL`, endTime, opID)
	return err
}
