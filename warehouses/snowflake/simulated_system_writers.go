// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/warehouses"
)

// Snowflake does not enforce ordinary-table primary keys. A singleton created
// atomically with the table serializes fence insertion, advancement and writes.
// All guarded DML executes in one server-side block: client suspension or a lost
// reply cannot resume a mutation outside a server transaction that has expired.
const simulatedSystemWriterTransaction = `DECLARE
	writer_generation BIGINT;
	BEGIN
	BEGIN TRANSACTION;
	UPDATE KRENALIS_SIMULATED_SYSTEM_WRITERS SET SERIALIZATION = NOT SERIALIZATION WHERE SIMULATED_SYSTEM = '';
	IF (SQLROWCOUNT <> 1) THEN ROLLBACK; RETURN FALSE; END IF;
	`

// DeleteSimulatedSystemRecords deletes records under the exact warehouse fence.
func (warehouse *Snowflake) DeleteSimulatedSystemRecords(ctx context.Context, system string, generation int64) error {

	if system == "" || generation < 0 || generation%2 != 1 {
		return fmt.Errorf("invalid simulated system writer authority")
	}
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return snowflake(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var mutation strings.Builder
	mutation.WriteString(`DELETE FROM KRENALIS_SIMULATED_SYSTEM_RECORDS WHERE SIMULATED_SYSTEM = `)
	quoteString(&mutation, system)

	return mergeSimulatedSystemWriter(ctx, conn, simulatedSystemWriter{system, generation}, mutation.String())
}

// FenceSimulatedSystem advances a persistent warehouse fence without reversing it.
func (warehouse *Snowflake) FenceSimulatedSystem(ctx context.Context, system string, generation int64) error {

	if system == "" || generation < 0 {
		return fmt.Errorf("invalid simulated system writer authority")
	}
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return snowflake(err)
	}
	var body strings.Builder
	body.WriteString(simulatedSystemWriterTransaction)
	body.WriteString(`MERGE INTO KRENALIS_SIMULATED_SYSTEM_WRITERS D USING (SELECT `)
	quoteString(&body, system)
	body.WriteString(` AS SIMULATED_SYSTEM, `)
	body.WriteString(strconv.FormatInt(generation, 10))
	body.WriteString(` AS GENERATION) S ON D.SIMULATED_SYSTEM = S.SIMULATED_SYSTEM
		WHEN MATCHED THEN UPDATE SET D.GENERATION = GREATEST(D.GENERATION, S.GENERATION)
		WHEN NOT MATCHED THEN INSERT (SIMULATED_SYSTEM, GENERATION, SERIALIZATION)
		VALUES (S.SIMULATED_SYSTEM, S.GENERATION, FALSE);
		COMMIT;
		RETURN TRUE;
		EXCEPTION WHEN OTHER THEN ROLLBACK; RAISE;
		END;`)

	return executeSimulatedSystemWriter(ctx, db, body.String())
}

// MergeSimulatedSystemRecords merges records under the exact warehouse fence.
func (warehouse *Snowflake) MergeSimulatedSystemRecords(ctx context.Context, system string, generation int64, table warehouses.Table, rows [][]any) error {

	if system == "" || generation < 0 || generation%2 != 0 || table.Name != "krenalis_simulated_system_records" {
		return fmt.Errorf("invalid simulated system writer authority")
	}
	for _, row := range rows {
		if len(row) != 4 || row[0] != system {
			return fmt.Errorf("invalid simulated system writer records")
		}
	}
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return snowflake(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	return mergeWithSimulatedSystemWriter(ctx, conn, table, rows, nil, &simulatedSystemWriter{system, generation})
}

type simulatedSystemWriter struct {
	system     string
	generation int64
}

type simulatedSystemWriterQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// executeSimulatedSystemWriter executes the complete transaction server-side.
func executeSimulatedSystemWriter(ctx context.Context, db simulatedSystemWriterQueryer, body string) error {

	var query strings.Builder
	query.WriteString("EXECUTE IMMEDIATE ")
	quoteString(&query, body)
	var permitted bool
	if err := db.QueryRowContext(ctx, query.String()).Scan(&permitted); err != nil {
		return snowflake(err)
	}
	if !permitted {
		return fmt.Errorf("simulated system writer authority has been revoked")
	}

	return nil
}

// mergeSimulatedSystemWriter holds the warehouse lock until mutation commits.
func mergeSimulatedSystemWriter(ctx context.Context, conn *sql.Conn, writer simulatedSystemWriter, mutation string) error {

	var body strings.Builder
	body.WriteString(simulatedSystemWriterTransaction)
	body.WriteString(`SELECT MAX(GENERATION) INTO :writer_generation FROM KRENALIS_SIMULATED_SYSTEM_WRITERS WHERE SIMULATED_SYSTEM = `)
	quoteString(&body, writer.system)
	body.WriteString(`;
		IF (writer_generation IS NULL OR writer_generation <> `)
	body.WriteString(strconv.FormatInt(writer.generation, 10))
	body.WriteString(`) THEN ROLLBACK; RETURN FALSE; END IF;
		`)
	body.WriteString(mutation)
	body.WriteString(`;
		COMMIT;
		RETURN TRUE;
		EXCEPTION WHEN OTHER THEN ROLLBACK; RAISE;
		END;`)

	return executeSimulatedSystemWriter(ctx, conn, body.String())
}
