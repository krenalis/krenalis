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
	"fmt"
	"time"

	"github.com/meergo/meergo"
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
// ErrAlterInProgress is returned; if an IdentityResolution operation is
// already in progress, the error ErrIdentityResolutionInProgress is returned.
//
// If a database error occurs, a *DataWarehouseError is returned.
func (warehouse *Snowflake) startOperation(ctx context.Context, operation warehouseOperation) (int, error) {
	db, err := warehouse.connection()
	if err != nil {
		return 0, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	err = warehouse.fixOperationsTable(ctx)
	if err != nil {
		return 0, err
	}
	var opID int

	err = warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
		// TODO(Gianluca): find a way to implement lock mechanism, if necessary.
		// _, err := tx.Exec(`LOCK TABLE "_operations"`)
		// if err != nil {
		// 	return meergo.Error(err)
		// }
		var runningOp *warehouseOperation
		err = tx.QueryRow(`SELECT "operation" FROM "_operations" ` +
			`WHERE "start_time" IS NOT NULL AND "end_time" IS NULL ORDER BY "id" DESC LIMIT 1`).Scan(&runningOp)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if runningOp != nil {
			switch *runningOp {
			case alterUserColumns:
				return meergo.ErrAlterInProgress
			case identityResolution:
				return meergo.ErrIdentityResolutionInProgress
			default:
				return fmt.Errorf("unexpected operation %q", *runningOp)
			}
		}
		_, err = tx.Exec(`INSERT INTO "_operations" ("operation", "start_time", "end_time") `+
			`VALUES (?, SYSDATE(), NULL)`, operation)
		if err != nil {
			return err
		}
		// TODO(Gianluca): this should be reviewed. It is just a workaround, as
		// Snowflake does not support the "INSERT ... RETURNING" syntax.
		err = tx.QueryRow(`SELECT MAX("id") FROM "_operations"`).Scan(&opID)
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
//
// If a database error occurs, a *DataWarehouseError is returned.
func (warehouse *Snowflake) endOperation(ctx context.Context, opID int, endTime time.Time) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	_, err = conn.ExecContext(ctx, `UPDATE "_operations" SET "end_time" = ? WHERE "id" = ? AND "end_time" IS NULL`, endTime, opID)
	if err != nil {
		return err
	}
	return nil
}

// fixOperationsTable fixes the '_operations' table.
//
// Note that, currently, calling this method is a no-op. See the issue
// https://github.com/meergo/meergo/issues/1046.
func (warehouse *Snowflake) fixOperationsTable(ctx context.Context) error {

	// TODO(Gianluca): this code has been commented as did not work as expected.
	//
	// See https://github.com/meergo/meergo/issues/1046.

	// db, err := warehouse.connection()
	// if err != nil {
	// 	return err
	// }
	// query := `UPDATE _operations
	// 	SET
	// 		end_time = (clock_timestamp() at time zone 'utc')::timestamp
	// 	WHERE
	// 		end_time IS NULL AND operation = 'IdentityResolution'
	// 			AND
	// 		NOT EXISTS (
	// 			SELECT pid
	// 			FROM pg_stat_activity
	// 			WHERE
	// 				datname = ` + quoteIdent(warehouse.settings.Database) + `
	// 					AND
	// 				query = 'CALL resolve_identities()'
	// 		)`
	// _, err = db.Query(ctx, query)
	// if err != nil {
	// 	return meergo.Error(err)
	// }
	return nil
}
