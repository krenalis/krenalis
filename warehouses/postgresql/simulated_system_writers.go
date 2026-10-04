// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/krenalis/krenalis/warehouses"
)

// DeleteSimulatedSystemRecords deletes records under the exact warehouse fence.
func (warehouse *PostgreSQL) DeleteSimulatedSystemRecords(ctx context.Context, system string, generation int64) error {

	if system == "" || generation < 0 || generation%2 != 1 {
		return fmt.Errorf("invalid simulated system writer authority")
	}
	pool, _, err := warehouse.connectionPool(ctx, false)
	if err != nil {
		return err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	return mergeSimulatedSystemWriter(ctx, conn, simulatedSystemWriter{system, generation},
		`DELETE FROM krenalis_simulated_system_records WHERE simulated_system = $1`, system)
}

// FenceSimulatedSystem advances a persistent warehouse fence without reversing it.
func (warehouse *PostgreSQL) FenceSimulatedSystem(ctx context.Context, system string, generation int64) error {

	if system == "" || generation < 0 {
		return fmt.Errorf("invalid simulated system writer authority")
	}
	pool, _, err := warehouse.connectionPool(ctx, false)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `INSERT INTO krenalis_simulated_system_writers (simulated_system, generation)
		VALUES ($1, $2) ON CONFLICT (simulated_system) DO UPDATE
		SET generation = GREATEST(krenalis_simulated_system_writers.generation, EXCLUDED.generation)`, system, generation)

	return err
}

// MergeSimulatedSystemRecords merges records under the exact warehouse fence.
func (warehouse *PostgreSQL) MergeSimulatedSystemRecords(ctx context.Context, system string, generation int64, table warehouses.Table, rows [][]any) error {

	if system == "" || generation < 0 || generation%2 != 0 || table.Name != "krenalis_simulated_system_records" {
		return fmt.Errorf("invalid simulated system writer authority")
	}
	for _, row := range rows {
		if len(row) != 4 || row[0] != system {
			return fmt.Errorf("invalid simulated system writer records")
		}
	}
	pool, _, err := warehouse.connectionPool(ctx, false)
	if err != nil {
		return err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	return mergeWithSimulatedSystemWriter(ctx, conn, table, rows, nil, &simulatedSystemWriter{system, generation})
}

type simulatedSystemWriter struct {
	system     string
	generation int64
}

// mergeSimulatedSystemWriter holds the fence row lock until mutation commits.
func mergeSimulatedSystemWriter(ctx context.Context, conn *pgxpool.Conn, writer simulatedSystemWriter, mutation string, args ...any) error {

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var generation int64
	if err := tx.QueryRow(ctx, `SELECT generation FROM krenalis_simulated_system_writers
		WHERE simulated_system = $1 FOR UPDATE`, writer.system).Scan(&generation); err != nil {
		return err
	}
	if generation != writer.generation {
		return fmt.Errorf("simulated system writer authority has been revoked")
	}

	_, err = tx.Exec(ctx, mutation, args...)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
