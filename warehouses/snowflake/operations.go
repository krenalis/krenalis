// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/meergo/meergo/core/backoff"
	"github.com/meergo/meergo/warehouses"
)

type warehouseOp string

const (
	alterProfileSchema warehouseOp = "AlterProfileSchema"
	identityResolution warehouseOp = "IdentityResolution"
)

type opStatus struct {
	canBeStarted     bool
	alreadyCompleted bool
	// executionError is significant only if 'alreadyCompleted' is true.
	// If executionError is not nil, it has type warehouses.OperationError.
	executionError error
}

// executeOperation starts an operation, identified by an ID.
//
// The returned status indicates whether the operation can be started, or
// returns the status of a current executing or previous execution.
func (warehouse *Snowflake) executeOperation(ctx context.Context, opID string, opType warehouseOp) (status *opStatus, err error) {
	var completedAt *time.Time
	var opError string
	bo := backoff.New(200)
	bo.SetCap(500 * time.Millisecond)
	for bo.Next(ctx) {
		err := warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
			var readID *string
			rows, err := tx.Query(`SELECT "ID", "COMPLETED_AT", "ERROR" FROM "_OPERATIONS" WHERE "ID" = ?`, opID)
			if err != nil {
				return snowflake(err)
			}
			defer rows.Close()
			for rows.Next() {
				err := rows.Scan(&readID, &completedAt, &opError)
				if err != nil {
					return snowflake(err)
				}
			}
			if err := rows.Err(); err != nil {
				return snowflake(err)
			}
			if readID == nil {
				// No rows in DB, so the operation can be started.
				_, err = tx.Exec(`INSERT INTO "_OPERATIONS" ("ID", "OPERATION_TYPE") VALUES (?, ?)`, opID, opType)
				if err != nil {
					return snowflake(err)
				}
				status = &opStatus{canBeStarted: true}
				return nil
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if status != nil {
			return status, nil
		}
		// Operation is still running, so wait 500ms then try again to check if
		// it has completed.
		if completedAt == nil {
			continue
		}
		// Operation is completed with an error.
		if opError != "" {
			return &opStatus{alreadyCompleted: true, executionError: warehouses.NewOperationError(errors.New(opError))}, nil
		}
		// Operations is completed without errors.
		return &opStatus{alreadyCompleted: true}, nil
	}
	return nil, ctx.Err()
}

// setOperationAsCompleted sets the given operation as completed. opError is the
// possible error in the execution of the operation, which will be stored in the
// database; nil means operation ended successfully.
// If an operation has already been set as completed, this method does
// nothing.
func (warehouse *Snowflake) setOperationAsCompleted(ctx context.Context, opID string, opError error) error {
	db := warehouse.openDB()
	var opErrorStr string
	if opError != nil {
		opErrorStr = opError.Error()
	}
	_, err := db.ExecContext(ctx, `UPDATE "_OPERATIONS" SET "COMPLETED_AT" = ?, "ERROR" = ?`+
		` WHERE "ID" = ? AND "COMPLETED_AT" IS NULL`, time.Now().UTC(), opErrorStr, opID)
	if err != nil {
		return snowflake(err)
	}
	return nil
}
