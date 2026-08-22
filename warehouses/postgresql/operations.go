// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

type warehouseOp string

const (
	alterProfileSchema warehouseOp = "AlterProfileSchema"
	identityResolution warehouseOp = "IdentityResolution"
)

type opStatus struct {
	canBeStarted             bool
	alreadyCompleted         bool
	identityResolutionCounts *warehouses.IdentityResolutionCounts
	executionError           *warehouses.OperationError
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

	resultColumnMigrationDone := false
	bo := backoff.New(200)
	bo.SetCap(500 * time.Millisecond)
	for bo.Next(ctx) {
		for {
			err = warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
				_, err = tx.Exec(ctx, `LOCK "krenalis_system_operations"`)
				if err != nil {
					return err
				}
				status, err = warehouse.readOperationStatus(ctx, tx, opID, opType)
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
				pgErr, ok := errors.AsType[*pgconn.PgError](err)
				if !ok || pgErr.Code != "42703" || resultColumnMigrationDone {
					return nil, err
				}
				// Add the 'result' column if it is not already present.
				pool, _, connectionErr := warehouse.connectionPool(ctx, false)
				if connectionErr != nil {
					return nil, connectionErr
				}
				_, err = pool.Exec(ctx,
					`ALTER TABLE "krenalis_system_operations" ADD COLUMN IF NOT EXISTS "result" jsonb`)
				if err != nil {
					return nil, err
				}
				resultColumnMigrationDone = true
				continue
			}
			break
		}
		if status.canBeStarted || status.alreadyCompleted {
			return status, nil
		}
		// The operation is still running, so wait before checking again.
	}

	return nil, ctx.Err()
}

// maxOperationResultBytes is the largest supported persisted operation result.
const maxOperationResultBytes = 4 << 10

// readOperationStatus reads the persisted status of an operation. It returns
// nil when the operation does not exist and validates any persisted Identity
// Resolution result.
func (warehouse *PostgreSQL) readOperationStatus(ctx context.Context, conn connection, opID string, opType warehouseOp) (*opStatus, error) {

	var completedAt *time.Time
	var opError string
	var result []byte
	err := conn.QueryRow(ctx, `SELECT "completed_at", LEFT("result"::text, $2), LEFT("error", $3)`+
		` FROM "krenalis_system_operations" WHERE "id" = $1 LIMIT 1`,
		opID, maxOperationResultBytes+1, warehouses.MaxOperationErrorBytes+1).
		Scan(&completedAt, &result, &opError)
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
		return status, nil
	}
	if opType != identityResolution {
		return status, nil
	}
	// Previous Krenalis versions did not persist Identity Resolution results.
	if result == nil {
		status.identityResolutionCounts = &warehouses.IdentityResolutionCounts{}
		return status, nil
	}
	if len(result) > maxOperationResultBytes {
		return nil, fmt.Errorf("Identity Resolution result for operation %s exceeds the supported size", opID)
	}
	var irResult struct {
		Counts *warehouses.IdentityResolutionCounts `json:"counts"`
	}
	if err := json.Unmarshal(result, &irResult); err != nil {
		return nil, fmt.Errorf("invalid Identity Resolution result for operation %s: %w", opID, err)
	}
	if irResult.Counts != nil {
		if err := warehouses.ValidateIdentityResolutionCounts(irResult.Counts); err != nil {
			return nil, fmt.Errorf("invalid Identity Resolution result for operation %s: %w", opID, err)
		}
	}
	status.identityResolutionCounts = irResult.Counts

	return status, nil
}

// setOperationAsCompleted sets the given operation as completed. result is the
// optional structured result and is persisted only for Identity Resolution.
// opError is the error returned by the operation; nil indicates that it
// completed successfully. If the operation has already been set as completed,
// this method does nothing.
func (warehouse *PostgreSQL) setOperationAsCompleted(
	ctx context.Context, conn connection, opID string, result json.Value, opError *warehouses.OperationError) error {

	var opErrorStr string
	if opError != nil {
		opErrorStr = opError.Error()
		result = nil
	}
	_, err := conn.Exec(ctx, `UPDATE "krenalis_system_operations"`+
		` SET "completed_at" = $1,`+
		` "result" = CASE WHEN "operation_type" = 'IdentityResolution' THEN $2::jsonb ELSE "result" END,`+
		` "error" = $3`+
		` WHERE "id" = $4 AND "completed_at" IS NULL`, time.Now().UTC(), []byte(result), opErrorStr, opID)

	return err
}
