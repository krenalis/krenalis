//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package snowflake

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/meergo/meergo"
	"github.com/meergo/meergo/backoff"
)

type warehouseOp string

const (
	alterUserColumns   warehouseOp = "AlterUserColumns"
	identityResolution warehouseOp = "IdentityResolution"
)

type opStatus struct {
	canBeStarted     bool
	alreadyCompleted bool
	// executionError is significant only if 'alreadyCompleted' is true.
	// If executionError is not nil, it has type meergo.OperationError.
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
			err = tx.QueryRow(`SELECT "COMPLETED_AT", "ERROR" FROM "_OPERATIONS" WHERE "ID" = ?`, opID).Scan(&completedAt, &opError)
			if err != nil {
				if err != pgx.ErrNoRows {
					// Generic database error.
					return err
				}
				// ErrNoRows, so the operation can be started.
				_, err = tx.Exec(`INSERT INTO "_OPERATIONS" ("ID", "OPERATION_TYPE") VALUES (?, ?)`, opID, opType)
				if err != nil {
					return err
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
			return &opStatus{alreadyCompleted: true, executionError: meergo.NewOperationError(errors.New(opError))}, nil
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
		return err
	}
	return nil
}
