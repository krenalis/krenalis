// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"database/sql"
	"time"

	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/warehouses"
)

type warehouseOp string

const (
	alterProfileSchema warehouseOp = "AlterProfileSchema"
	identityResolution warehouseOp = "IdentityResolution"
)

type opStatus struct {
	canBeStarted     bool
	alreadyCompleted bool
	executionError   *warehouses.OperationError
}

// connection is implemented by Snowflake database handles and transactions.
type connection interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// executeOperation starts an operation, identified by an ID.
//
// The returned status indicates whether the operation can be started or
// describes an operation that has already completed.
func (warehouse *Snowflake) executeOperation(ctx context.Context, opID string, opType warehouseOp) (status *opStatus, err error) {

	bo := backoff.New(200)
	bo.SetCap(500 * time.Millisecond)
	for bo.Next(ctx) {
		err = warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
			status, err = warehouse.readOperationStatus(ctx, tx, opID)
			if err != nil {
				return err
			}
			if status != nil {
				return nil
			}
			// No existing operation was found, so this one can be started.
			_, err = tx.ExecContext(ctx,
				`INSERT INTO "KRENALIS_SYSTEM_OPERATIONS" ("ID", "OPERATION_TYPE") VALUES (?, ?)`,
				opID, opType)
			if err != nil {
				return snowflake(err)
			}
			status = &opStatus{canBeStarted: true}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if status.canBeStarted || status.alreadyCompleted {
			return status, nil
		}
		// The operation is still running, so wait before checking again.
	}

	return nil, ctx.Err()
}

// readOperationStatus reads the persisted status of an operation. It returns
// nil when the operation does not exist.
func (warehouse *Snowflake) readOperationStatus(ctx context.Context, conn connection, opID string) (*opStatus, error) {

	// LEFT counts characters, not bytes. It bounds the error returned by the query
	// before the driver receives it; NewPersistedOperationError enforces the exact
	// byte limit.
	const operationErrorReadLimitCharacters = warehouses.MaxOperationErrorBytes + 1

	var completedAt *time.Time
	var opError string
	err := conn.QueryRowContext(ctx, `SELECT "COMPLETED_AT", LEFT("ERROR", ?)`+
		` FROM "KRENALIS_SYSTEM_OPERATIONS" WHERE "ID" = ? LIMIT 1`,
		operationErrorReadLimitCharacters, opID).
		Scan(&completedAt, &opError)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, snowflake(err)
	}
	status := &opStatus{}
	if completedAt == nil {
		return status, nil
	}
	status.alreadyCompleted = true
	if opError != "" {
		status.executionError = warehouses.NewPersistedOperationError(opError)
	}

	return status, nil
}

// setOperationAsCompleted sets the given operation as completed. opError is the
// error returned by the operation; nil indicates that it completed
// successfully. If the operation has already been set as completed, this
// method does nothing.
func (warehouse *Snowflake) setOperationAsCompleted(ctx context.Context, conn connection, opID string, opError *warehouses.OperationError) error {

	var opErrorStr string
	if opError != nil {
		opErrorStr = opError.Error()
	}
	_, err := conn.ExecContext(ctx, `UPDATE "KRENALIS_SYSTEM_OPERATIONS" SET "COMPLETED_AT" = ?, "ERROR" = ?`+
		` WHERE "ID" = ? AND "COMPLETED_AT" IS NULL`, time.Now().UTC(), opErrorStr, opID)

	return snowflake(err)
}
