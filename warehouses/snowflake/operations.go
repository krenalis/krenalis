// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/snowflakedb/gosnowflake/v2"
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

// connection is implemented by Snowflake database handles and transactions.
type connection interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// invalidIdentifierErrorNumber is Snowflake's invalid identifier error number.
const invalidIdentifierErrorNumber = 904

// executeOperation starts an operation, identified by an ID.
//
// The returned status indicates whether the operation can be started or
// describes an operation that has already completed.
func (warehouse *Snowflake) executeOperation(ctx context.Context, opID string, opType warehouseOp) (status *opStatus, err error) {

	resultColumnMigrationDone := false
	bo := backoff.New(200)
	bo.SetCap(500 * time.Millisecond)
	for bo.Next(ctx) {
		for {
			err = warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
				status, err = warehouse.readOperationStatus(ctx, tx, opID, opType)
				if err != nil {
					return err
				}
				if status != nil {
					return nil
				}
				// No existing operation was found, so this one can be started.
				_, err = tx.Exec(
					`INSERT INTO "KRENALIS_SYSTEM_OPERATIONS" ("ID", "OPERATION_TYPE") VALUES (?, ?)`,
					opID, opType)
				if err != nil {
					return snowflake(err)
				}
				status = &opStatus{canBeStarted: true}
				return nil
			})
			if err != nil {
				snowflakeErr, ok := errors.AsType[*gosnowflake.SnowflakeError](err)
				if !ok || snowflakeErr.Number != invalidIdentifierErrorNumber || resultColumnMigrationDone {
					return nil, err
				}
				// Add the 'result' column if it is not already present.
				db, connectionErr := warehouse.openDB(ctx)
				if connectionErr != nil {
					return nil, snowflake(connectionErr)
				}
				_, err = db.ExecContext(ctx,
					`ALTER TABLE "KRENALIS_SYSTEM_OPERATIONS" ADD COLUMN IF NOT EXISTS "RESULT" VARIANT`)
				if err != nil {
					return nil, snowflake(err)
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
func (warehouse *Snowflake) readOperationStatus(ctx context.Context, conn connection, opID string, opType warehouseOp) (*opStatus, error) {

	var completedAt *time.Time
	var opError string
	var result []byte
	err := conn.QueryRowContext(ctx, `SELECT "COMPLETED_AT", LEFT(TO_JSON("RESULT"), ?), LEFT("ERROR", ?)`+
		` FROM "KRENALIS_SYSTEM_OPERATIONS" WHERE "ID" = ? LIMIT 1`,
		maxOperationResultBytes+1, warehouses.MaxOperationErrorBytes+1, opID).
		Scan(&completedAt, &result, &opError)
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
func (warehouse *Snowflake) setOperationAsCompleted(ctx context.Context, conn connection, opID string, result json.Value, opError *warehouses.OperationError) error {

	var opErrorStr string
	if opError != nil {
		opErrorStr = opError.Error()
		result = nil
	}
	_, err := conn.ExecContext(ctx, `UPDATE "KRENALIS_SYSTEM_OPERATIONS"`+
		` SET "COMPLETED_AT" = ?,`+
		` "RESULT" = CASE WHEN "OPERATION_TYPE" = 'IdentityResolution'`+
		` THEN PARSE_JSON(NULLIF(?, '')) ELSE "RESULT" END,`+
		` "ERROR" = ?`+
		` WHERE "ID" = ? AND "COMPLETED_AT" IS NULL`, time.Now().UTC(), string(result), opErrorStr, opID)

	return snowflake(err)
}
