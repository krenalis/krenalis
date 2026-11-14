// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"slices"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// SimulatedSystemRecordWrite is one complete document to create or update.
type SimulatedSystemRecordWrite struct {
	ID   string
	Data json.Value
}

// SimulatedSystemRecordWriteResult reports writes rejected by record identity.
type SimulatedSystemRecordWriteResult struct {
	MissingUpdates   []string
	CollidingCreates []string
}

// SimulatedSystemRecordsByID returns complete documents for the requested IDs.
func (store *Store) SimulatedSystemRecordsByID(ctx context.Context, account string, ids []string) (map[string]json.Value, error) {
	store.mustBeOpen()
	if account == "" {
		return nil, errors.New("simulated system is missing")
	}
	if len(ids) == 0 {
		return map[string]json.Value{}, nil
	}
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return nil, err
	}
	defer done()

	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = id
	}
	where := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
		warehouses.NewBaseExpr(simulatedSystemRecordsTable.Columns[0], warehouses.OpIs, account),
		warehouses.NewBaseExpr(simulatedSystemRecordsTable.Columns[1], warehouses.OpIsOneOf, values...),
	})
	rows, _, err := store.warehouse().Query(ctx, warehouses.RowQuery{
		Columns: []warehouses.Column{simulatedSystemRecordsTable.Columns[1], simulatedSystemRecordsTable.Columns[3]},
		Table:   simulatedSystemRecordsTable.Name,
		Where:   where,
		Limit:   len(ids),
	}, false)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]json.Value, len(ids))
	row := make([]any, 2)
	for rows.Next() {
		if err := rows.Scan(row...); err != nil {
			return nil, err
		}
		id, idOK := row[0].(string)
		data, dataOK := row[1].(json.Value)
		if !idOK || id == "" || !dataOK {
			return nil, wrapWarehouseError(errors.New("warehouse returned an invalid simulated record"))
		}
		result[id] = slices.Clone(data)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return result, nil
}

// WriteSimulatedSystemRecords writes complete destination documents.
func (store *Store) WriteSimulatedSystemRecords(ctx context.Context, account string, creates, updates []SimulatedSystemRecordWrite) (SimulatedSystemRecordWriteResult, error) {
	store.mustBeOpen()
	if account == "" {
		return SimulatedSystemRecordWriteResult{}, errors.New("simulated system is missing")
	}
	writer, ok := ctx.Value(simulatedSystemWriterKey{}).(simulatedSystemWriter)
	if !ok || writer.system != account || writer.generation == 0 || writer.generation%2 != 0 {
		return SimulatedSystemRecordWriteResult{}, errors.New("simulated system writer authority is missing")
	}
	if len(creates) == 0 && len(updates) == 0 {
		return SimulatedSystemRecordWriteResult{}, nil
	}

	ids := make([]string, 0, len(creates)+len(updates))
	for _, record := range creates {
		ids = append(ids, record.ID)
	}
	for _, record := range updates {
		ids = append(ids, record.ID)
	}
	existing, err := store.SimulatedSystemRecordsByID(ctx, account, ids)
	if err != nil {
		return SimulatedSystemRecordWriteResult{}, err
	}

	result := SimulatedSystemRecordWriteResult{}
	rows := make([][]any, 0, len(creates)+len(updates))
	seenCreates := map[string]struct{}{}
	seenUpdates := map[string]struct{}{}
	for _, record := range creates {
		if _, ok := existing[record.ID]; ok {
			result.CollidingCreates = append(result.CollidingCreates, record.ID)
			continue
		}
		if _, ok := seenCreates[record.ID]; ok {
			result.CollidingCreates = append(result.CollidingCreates, record.ID)
			continue
		}
		seenCreates[record.ID] = struct{}{}
		rows = append(rows, []any{account, record.ID, nil, record.Data})
	}
	for _, record := range updates {
		if _, ok := existing[record.ID]; !ok {
			result.MissingUpdates = append(result.MissingUpdates, record.ID)
			continue
		}
		if _, ok := seenUpdates[record.ID]; ok {
			continue
		}
		seenUpdates[record.ID] = struct{}{}
		rows = append(rows, []any{account, record.ID, nil, record.Data})
	}
	if len(rows) == 0 {
		return result, nil
	}

	ctx, done, err := store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return SimulatedSystemRecordWriteResult{}, err
	}
	defer done()
	previousMaximum, err := store.simulatedSystemRecordsMaximumUpdatedAt(ctx, account)
	if err != nil {
		return SimulatedSystemRecordWriteResult{}, err
	}
	updatedAt := time.Now().UTC().Truncate(time.Microsecond)
	if !previousMaximum.IsZero() && !updatedAt.After(previousMaximum) {
		updatedAt = previousMaximum.Add(time.Microsecond)
	}
	for _, row := range rows {
		row[2] = updatedAt
	}
	warehouse, ok := store.warehouse().(simulatedSystemWarehouse)
	if !ok {
		return SimulatedSystemRecordWriteResult{}, wrapWarehouseError(errors.New("warehouse does not support simulated system writer fencing"))
	}
	err = warehouse.MergeSimulatedSystemRecords(ctx, account, writer.generation, simulatedSystemRecordsTable, rows)
	if err != nil {
		return SimulatedSystemRecordWriteResult{}, err
	}
	return result, nil
}

func (store *Store) simulatedSystemRecordsMaximumUpdatedAt(ctx context.Context, account string) (time.Time, error) {
	rows, _, err := store.warehouse().Query(ctx, warehouses.RowQuery{
		Columns:   []warehouses.Column{simulatedSystemRecordsTable.Columns[2]},
		Table:     simulatedSystemRecordsTable.Name,
		Where:     warehouses.NewBaseExpr(simulatedSystemRecordsTable.Columns[0], warehouses.OpIs, account),
		OrderBy:   []warehouses.Column{simulatedSystemRecordsTable.Columns[2]},
		OrderDesc: true,
		Limit:     1,
	}, false)
	if err != nil {
		return time.Time{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return time.Time{}, err
		}
		if err := rows.Close(); err != nil {
			return time.Time{}, err
		}
		return time.Time{}, nil
	}
	row := make([]any, 1)
	if err := rows.Scan(row...); err != nil {
		return time.Time{}, err
	}
	if err := rows.Close(); err != nil {
		return time.Time{}, err
	}
	updatedAt, ok := row[0].(time.Time)
	if !ok {
		return time.Time{}, wrapWarehouseError(errors.New("warehouse returned an invalid simulated-record timestamp"))
	}
	return updatedAt.UTC().Truncate(time.Microsecond), nil
}
