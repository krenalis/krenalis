// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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

// connection is implemented by PostgreSQL pools and transactions.
type connection interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// executeOperation starts an operation, identified by an ID.
//
// The returned status indicates whether the operation can be started or
// describes an operation that has already completed.
func (warehouse *PostgreSQL) executeOperation(ctx context.Context, opID string, opType warehouseOp) (status *opStatus, err error) {

	bo := backoff.New(200)
	bo.SetCap(500 * time.Millisecond)
	for bo.Next(ctx) {
		err = warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
			_, err = tx.Exec(ctx, `LOCK "krenalis_system_operations"`)
			if err != nil {
				return err
			}
			status, err = warehouse.readOperationStatus(ctx, tx, opID)
			if err != nil {
				return err
			}
			if status != nil {
				return nil
			}
			// No existing operation was found, so this one can be started.
			_, err = tx.Exec(ctx,
				`INSERT INTO "krenalis_system_operations" ("id", "operation_type") VALUES ($1, $2)`,
				opID, opType)
			if err != nil {
				return err
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
func (warehouse *PostgreSQL) readOperationStatus(ctx context.Context, conn connection, opID string) (*opStatus, error) {

	var completedAt *time.Time
	var opError string
	err := conn.QueryRow(ctx, `SELECT "completed_at", LEFT("error", $2)`+
		` FROM "krenalis_system_operations" WHERE "id" = $1 LIMIT 1`,
		opID, warehouses.MaxOperationErrorBytes+1).
		Scan(&completedAt, &opError)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
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
func (warehouse *PostgreSQL) setOperationAsCompleted(ctx context.Context, conn connection, opID string, opError *warehouses.OperationError) error {

	var opErrorStr string
	if opError != nil {
		opErrorStr = opError.Error()
	}
	_, err := conn.Exec(ctx, `UPDATE "krenalis_system_operations" SET "completed_at" = $1, "error" = $2`+
		` WHERE "id" = $3 AND "completed_at" IS NULL`, time.Now().UTC(), opErrorStr, opID)

	return err
}
