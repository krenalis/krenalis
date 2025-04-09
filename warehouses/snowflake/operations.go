//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package snowflake

import (
	"context"
	"database/sql"
	"time"
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
func (warehouse *Snowflake) startOperation(ctx context.Context, operation warehouseOperation) (int, error) {
	var opID int
	err := warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
		// TODO(Gianluca): find a way to implement lock mechanism, if necessary.
		// _, err := tx.Exec(`LOCK TABLE "_OPERATIONS"`)
		// if err != nil {
		// 	return meergo.Error(err)
		// }
		var runningOp *warehouseOperation
		err := tx.QueryRow(`SELECT "OPERATION" FROM "_OPERATIONS" ` +
			`WHERE "START_TIME" IS NOT NULL AND "END_TIME" IS NULL ORDER BY "ID" DESC LIMIT 1`).Scan(&runningOp)
		if err != nil && err != sql.ErrNoRows {
			return snowflake(err)
		}
		_ = runningOp // TODO: this will be removed. See https://github.com/meergo/meergo/issues/1475.
		_, err = tx.Exec(`INSERT INTO "_OPERATIONS" ("OPERATION", "START_TIME", "END_TIME") `+
			`VALUES (?, SYSDATE(), NULL)`, operation)
		if err != nil {
			return snowflake(err)
		}
		// TODO(Gianluca): this should be reviewed. It is just a workaround, as
		// Snowflake does not support the "INSERT ... RETURNING" syntax.
		err = tx.QueryRow(`SELECT MAX("ID") FROM "_OPERATIONS"`).Scan(&opID)
		if err != nil {
			return snowflake(err)
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
func (warehouse *Snowflake) endOperation(ctx context.Context, opID int, endTime time.Time) error {
	db := warehouse.openDB()
	_, err := db.ExecContext(ctx, `UPDATE "_OPERATIONS" SET "END_TIME" = ? WHERE "ID" = ? AND "END_TIME" IS NULL`, endTime, opID)
	return snowflake(err)
}
