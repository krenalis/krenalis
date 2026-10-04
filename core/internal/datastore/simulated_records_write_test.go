// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

func TestWriteSimulatedSystemRecords(t *testing.T) {
	previous := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	warehouse := &simulatedRecordsWriteWarehouse{records: map[string]simulatedRecordsWriteRow{
		"existing": {updatedAt: previous, data: json.Value(`{"customer_id":"existing"}`)},
	}}
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(warehouses.Warehouse(warehouse))

	result, err := store.WriteSimulatedSystemRecords(simulatedWriterContext(t, "account-a", 2), "account-a",
		[]SimulatedSystemRecordWrite{{ID: "created", Data: json.Value(`{"customer_id":"created"}`)}},
		[]SimulatedSystemRecordWrite{{ID: "existing", Data: json.Value(`{"customer_id":"existing","changed":true}`)}},
	)
	if err != nil {
		t.Fatalf("expected simulated-system writes, got %v", err)
	}
	if len(result.CollidingCreates) != 0 || len(result.MissingUpdates) != 0 {
		t.Fatalf("expected no identity write failures, got %+v", result)
	}
	created := warehouse.records["created"]
	updated := warehouse.records["existing"]
	if !created.updatedAt.After(previous) || !updated.updatedAt.After(previous) {
		t.Fatalf("expected destination timestamps after previous maximum, got %v and %v", created.updatedAt, updated.updatedAt)
	}
	if !created.updatedAt.Equal(updated.updatedAt) {
		t.Fatalf("expected one batch timestamp, got %v and %v", created.updatedAt, updated.updatedAt)
	}
	if created.updatedAt.Location() != time.UTC || created.updatedAt.Nanosecond()%1000 != 0 {
		t.Fatalf("expected UTC microsecond timestamp, got %v", created.updatedAt)
	}

	result, err = store.WriteSimulatedSystemRecords(simulatedWriterContext(t, "account-a", 2), "account-a",
		[]SimulatedSystemRecordWrite{{ID: "existing", Data: json.Value(`{"customer_id":"overwrite"}`)}},
		[]SimulatedSystemRecordWrite{{ID: "missing", Data: json.Value(`{"customer_id":"missing"}`)}},
	)
	if err != nil {
		t.Fatalf("expected identity failures without a warehouse error, got %v", err)
	}
	if !slices.Equal(result.CollidingCreates, []string{"existing"}) || !slices.Equal(result.MissingUpdates, []string{"missing"}) {
		t.Fatalf("expected collision and missing-update results, got %+v", result)
	}
	if string(warehouse.records["existing"].data) != `{"customer_id":"existing","changed":true}` {
		t.Fatalf("expected colliding create to leave the existing row unchanged, got %s", warehouse.records["existing"].data)
	}
}

func TestWriteSimulatedSystemRecordsPostgreSQL(t *testing.T) {
	settings := simulatedRecordsWarehouseSettings(t, "PostgreSQL")
	warehouse := warehouses.Registered("PostgreSQL").New(newTestSettingsLoader(settings), nil)
	t.Cleanup(func() {
		if err := warehouse.Close(); err != nil {
			t.Errorf("expected PostgreSQL warehouse closure, got %v", err)
		}
	})
	err := warehouse.Initialize(t.Context(), warehouses.Development, nil)
	if err != nil {
		t.Fatalf("expected PostgreSQL warehouse initialization, got %v", err)
	}
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(warehouse)
	err = store.FenceSimulatedSystem(t.Context(), "account-a", 2)
	if err != nil {
		t.Fatalf("expected PostgreSQL writer admission, got %v", err)
	}
	creates := []SimulatedSystemRecordWrite{
		{ID: "record-a", Data: json.Value(`{"customer_id":"record-a","first_name":"Ada"}`)},
		{ID: "record-b", Data: json.Value(`{"customer_id":"record-b","first_name":"Grace"}`)},
	}
	result, err := store.WriteSimulatedSystemRecords(simulatedWriterContext(t, "account-a", 2), "account-a", creates, nil)
	if err != nil {
		t.Fatalf("expected PostgreSQL destination create, got %v", err)
	}
	if len(result.CollidingCreates) != 0 || len(result.MissingUpdates) != 0 {
		t.Fatalf("expected successful PostgreSQL creates, got %+v", result)
	}
	first, err := store.SimulatedSystemRecords(t.Context(), "account-a", SimulatedRecordQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected PostgreSQL created records, got %v", err)
	}
	if len(first.Records) != 2 || first.Records[0].UpdatedAt != first.Records[1].UpdatedAt {
		t.Fatalf("expected one persisted batch timestamp for two creates, got %+v", first.Records)
	}
	firstUpdatedAt := simulatedRecordWithID(t, first.Records, "record-a").UpdatedAt
	assertSimulatedRecordTimestamp(t, firstUpdatedAt)

	result, err = store.WriteSimulatedSystemRecords(simulatedWriterContext(t, "account-a", 2), "account-a",
		[]SimulatedSystemRecordWrite{{ID: "record-a", Data: json.Value(`{"customer_id":"overwrite"}`)}}, nil)
	if err != nil {
		t.Fatalf("expected PostgreSQL create collision result, got %v", err)
	}
	if !slices.Equal(result.CollidingCreates, []string{"record-a"}) {
		t.Fatalf("expected existing ID collision, got %+v", result)
	}
	unchanged, err := store.SimulatedSystemRecords(t.Context(), "account-a", SimulatedRecordQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected PostgreSQL records after collision, got %v", err)
	}
	var unchangedDocument map[string]any
	err = json.Unmarshal(simulatedRecordWithID(t, unchanged.Records, "record-a").Attributes, &unchangedDocument)
	if err != nil {
		t.Fatalf("expected existing PostgreSQL document after collision, got %v", err)
	}
	if unchangedDocument["customer_id"] != "record-a" || unchangedDocument["first_name"] != "Ada" {
		t.Fatalf("expected collision to leave the existing row unchanged, got %v", unchangedDocument)
	}
	result, err = store.WriteSimulatedSystemRecords(simulatedWriterContext(t, "account-a", 2), "account-a", nil,
		[]SimulatedSystemRecordWrite{{ID: "missing", Data: json.Value(`{"customer_id":"missing"}`)}})
	if err != nil {
		t.Fatalf("expected PostgreSQL missing-update result, got %v", err)
	}
	if !slices.Equal(result.MissingUpdates, []string{"missing"}) {
		t.Fatalf("expected missing update to be rejected, got %+v", result)
	}
	missing, err := store.SimulatedSystemRecordsByID(t.Context(), "account-a", []string{"missing"})
	if err != nil {
		t.Fatalf("expected PostgreSQL missing-record lookup, got %v", err)
	}
	if _, ok := missing["missing"]; ok {
		t.Fatalf("expected missing update ID to remain absent, got %v", missing)
	}

	result, err = store.WriteSimulatedSystemRecords(simulatedWriterContext(t, "account-a", 2), "account-a", nil,
		[]SimulatedSystemRecordWrite{{ID: "record-a", Data: json.Value(`{"customer_id":"record-a","first_name":"Ada Updated"}`)}})
	if err != nil || len(result.MissingUpdates) != 0 {
		t.Fatalf("expected PostgreSQL existing-record update, got result=%+v error=%v", result, err)
	}
	updated, err := store.SimulatedSystemRecords(t.Context(), "account-a", SimulatedRecordQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected PostgreSQL updated records, got %v", err)
	}
	secondUpdatedAt := simulatedRecordWithID(t, updated.Records, "record-a").UpdatedAt
	if !secondUpdatedAt.After(firstUpdatedAt) {
		t.Fatalf("expected persisted update timestamp after %v, got %v", firstUpdatedAt, secondUpdatedAt)
	}
	assertSimulatedRecordTimestamp(t, secondUpdatedAt)

	freshStore := &Store{mc: newModeCoordinator(state.Normal)}
	freshStore.wh.Store(warehouse)
	result, err = freshStore.WriteSimulatedSystemRecords(simulatedWriterContext(t, "account-a", 2), "account-a", nil,
		[]SimulatedSystemRecordWrite{{ID: "record-a", Data: json.Value(`{"customer_id":"record-a","first_name":"Ada Reloaded"}`)}})
	if err != nil || len(result.MissingUpdates) != 0 {
		t.Fatalf("expected update through fresh datastore Store, got result=%+v error=%v", result, err)
	}
	final, err := freshStore.SimulatedSystemRecords(t.Context(), "account-a", SimulatedRecordQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected PostgreSQL records after fresh Store update, got %v", err)
	}
	if len(final.Records) != 2 {
		t.Fatalf("expected missing update not to insert and successful update to persist, got %+v", final.Records)
	}
	var finalDocument map[string]any
	finalRecord := simulatedRecordWithID(t, final.Records, "record-a")
	err = json.Unmarshal(finalRecord.Attributes, &finalDocument)
	if err != nil {
		t.Fatalf("expected PostgreSQL document after fresh Store update, got %v", err)
	}
	if finalDocument["customer_id"] != "record-a" || finalDocument["first_name"] != "Ada Reloaded" {
		t.Fatalf("expected successful update to persist without inserting the missing ID, got %v", finalDocument)
	}
	thirdUpdatedAt := finalRecord.UpdatedAt
	if !thirdUpdatedAt.After(secondUpdatedAt) {
		t.Fatalf("expected fresh Store timestamp after persisted maximum %v, got %v", secondUpdatedAt, thirdUpdatedAt)
	}
	assertSimulatedRecordTimestamp(t, thirdUpdatedAt)
}

func assertSimulatedRecordTimestamp(t *testing.T, value time.Time) {
	t.Helper()
	if value.Location() != time.UTC || value.Nanosecond()%1000 != 0 {
		t.Fatalf("expected UTC microsecond simulated-record timestamp, got %v", value)
	}
}

func simulatedRecordWithID(t *testing.T, records []SimulatedRecord, id string) SimulatedRecord {
	t.Helper()
	for _, record := range records {
		if record.ID == id {
			return record
		}
	}
	t.Fatalf("expected simulated record %q, got %+v", id, records)
	return SimulatedRecord{}
}

type simulatedRecordsWriteWarehouse struct {
	warehouses.Warehouse
	records map[string]simulatedRecordsWriteRow
}

func (warehouse *simulatedRecordsWriteWarehouse) DeleteSimulatedSystemRecords(context.Context, string, int64) error {
	return nil
}

func (warehouse *simulatedRecordsWriteWarehouse) FenceSimulatedSystem(context.Context, string, int64) error {
	return nil
}

type simulatedRecordsWriteRow struct {
	updatedAt time.Time
	data      json.Value
}

func (warehouse *simulatedRecordsWriteWarehouse) Merge(_ context.Context, _ warehouses.Table, rows [][]any, _ []any) error {
	for _, row := range rows {
		warehouse.records[row[1].(string)] = simulatedRecordsWriteRow{updatedAt: row[2].(time.Time), data: slices.Clone(row[3].(json.Value))}
	}
	return nil
}

func (warehouse *simulatedRecordsWriteWarehouse) MergeSimulatedSystemRecords(_ context.Context, _ string, _ int64, _ warehouses.Table, rows [][]any) error {
	for _, row := range rows {
		warehouse.records[row[1].(string)] = simulatedRecordsWriteRow{updatedAt: row[2].(time.Time), data: slices.Clone(row[3].(json.Value))}
	}
	return nil
}

func (warehouse *simulatedRecordsWriteWarehouse) Query(_ context.Context, query warehouses.RowQuery, _ bool) (warehouses.Rows, int, error) {
	rows := make([][]any, 0, len(warehouse.records))
	if len(query.Columns) == 2 {
		ids := map[string]struct{}{}
		where := query.Where.(*warehouses.MultiExpr)
		idWhere := where.Operands[1].(*warehouses.BaseExpr)
		for _, value := range idWhere.Values {
			ids[value.(string)] = struct{}{}
		}
		for id, record := range warehouse.records {
			if _, ok := ids[id]; ok {
				rows = append(rows, []any{id, record.data})
			}
		}
	} else {
		for _, record := range warehouse.records {
			rows = append(rows, []any{record.updatedAt})
		}
	}
	if query.OrderDesc && len(rows) > 1 {
		if rows[0][0].(time.Time).Before(rows[1][0].(time.Time)) {
			rows[0], rows[1] = rows[1], rows[0]
		}
	}
	if query.Limit > 0 && len(rows) > query.Limit {
		rows = rows[:query.Limit]
	}
	return &simulatedRecordsWriteRows{rows: rows}, len(rows), nil
}

type simulatedRecordsWriteRows struct {
	rows  [][]any
	index int
}

func (rows *simulatedRecordsWriteRows) Close() error { return nil }

func (rows *simulatedRecordsWriteRows) Err() error { return nil }

func (rows *simulatedRecordsWriteRows) Next() bool {
	return rows.index < len(rows.rows)
}

func (rows *simulatedRecordsWriteRows) Scan(dest ...any) error {
	for i := range dest {
		dest[i] = rows.rows[rows.index][i]
	}
	rows.index++
	return nil
}

func simulatedWriterContext(t *testing.T, system string, generation int64) context.Context {
	t.Helper()
	ctx, err := WithSimulatedSystemWriter(t.Context(), system, generation)
	if err != nil {
		t.Fatalf("expected simulated writer context, got %v", err)
	}
	return ctx
}
