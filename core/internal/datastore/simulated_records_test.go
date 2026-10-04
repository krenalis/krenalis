// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"os"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"

	// Import warehouse platforms for TestSimulatedSystemRecordsWarehouses.
	_ "github.com/krenalis/krenalis/warehouses/postgresql"
	_ "github.com/krenalis/krenalis/warehouses/snowflake"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type emptySimulatedRecordsRows struct{}

// Close closes the empty rows.
func (*emptySimulatedRecordsRows) Close() error { return nil }

// Err reports successful iteration.
func (*emptySimulatedRecordsRows) Err() error { return nil }

// Next reports that no row is available.
func (*emptySimulatedRecordsRows) Next() bool { return false }

// Scan does nothing because no row is available.
func (*emptySimulatedRecordsRows) Scan(...any) error { return nil }

type emptySimulatedRecordsWarehouse struct {
	warehouses.Warehouse
	query warehouses.RowQuery
}

// Query records the query and returns no rows.
func (warehouse *emptySimulatedRecordsWarehouse) Query(_ context.Context, query warehouses.RowQuery, _ bool) (warehouses.Rows, int, error) {
	warehouse.query = query
	return &emptySimulatedRecordsRows{}, 0, nil
}

// TestParseSimulatedRecordsCursor verifies symmetric cursor serialization.
func TestParseSimulatedRecordsCursor(t *testing.T) {

	zero, err := ParseSimulatedRecordsCursor("")
	if err != nil || zero != (SimulatedRecordsCursor{}) || zero.String() != "" {
		t.Fatalf("expected a zero cursor, got cursor=%+v string=%q error=%v", zero, zero.String(), err)
	}

	updatedAt := time.Date(2026, 9, 22, 10, 0, 34, 567890000, time.UTC)
	cursor := SimulatedRecordsCursor{UpdatedAt: updatedAt, ID: "sr_v2_source_abcd1234"}
	encoded := cursor.String()
	parsed, err := ParseSimulatedRecordsCursor(encoded)
	if err != nil {
		t.Fatalf("expected cursor parsing, got %v", err)
	}
	if parsed.ID != cursor.ID || !parsed.UpdatedAt.Equal(cursor.UpdatedAt) || parsed.UpdatedAt.Location() != time.UTC ||
		parsed.UpdatedAt.Nanosecond()%1000 != 0 {
		t.Fatalf("expected exact UTC microsecond cursor round-trip, got %+v", parsed)
	}

	malformed := []string{
		"missing-separator", ":record", "1:", "value:record", "1:record:extra", "9223372036854775807:record",
	}
	for _, value := range malformed {
		if _, err := ParseSimulatedRecordsCursor(value); err == nil {
			t.Fatalf("expected malformed cursor %q to be rejected", value)
		}
	}

}

// TestSimulatedSystemRecordsArguments verifies argument bounds.
func TestSimulatedSystemRecordsArguments(t *testing.T) {

	for _, query := range []SimulatedRecordQuery{{Limit: -1}, {}, {Limit: simulatedRecordMaxPageSize + 1}} {
		_, err := (&Store{}).SimulatedSystemRecords(t.Context(), "account-a", query)
		if err == nil {
			t.Fatalf("expected invalid limit %d to be rejected", query.Limit)
		}
	}

	_, err := (&Store{}).SimulatedSystemRecords(t.Context(), "", SimulatedRecordQuery{Limit: 1})
	if err == nil {
		t.Fatal("expected an empty account to be rejected")
	}
	_, err = (&Store{}).SimulatedSystemRecords(t.Context(), "account-a", SimulatedRecordQuery{
		After: SimulatedRecordsCursor{ID: "record-a"}, Limit: 1,
	})
	if err == nil {
		t.Fatal("expected a partial cursor to be rejected")
	}

	warehouse := &emptySimulatedRecordsWarehouse{}
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(warehouses.Warehouse(warehouse))
	page, err := store.SimulatedSystemRecords(t.Context(), "account-a",
		SimulatedRecordQuery{Limit: simulatedRecordMaxPageSize})
	if err != nil {
		t.Fatalf("expected maximum bounded limit, got %v", err)
	}
	if len(page.Records) != 0 || page.HasMore || page.Next != (SimulatedRecordsCursor{}) {
		t.Fatalf("expected an empty final page, got %+v", page)
	}
	if warehouse.query.Limit != simulatedRecordMaxPageSize+1 {
		t.Fatalf("expected look-ahead limit %d, got %d", simulatedRecordMaxPageSize+1, warehouse.query.Limit)
	}

}

// TestSimulatedSystemRecordsWarehouses verifies reads on every warehouse.
func TestSimulatedSystemRecordsWarehouses(t *testing.T) {

	for _, platform := range []string{"PostgreSQL", "Snowflake"} {
		t.Run(platform, func(t *testing.T) {

			settings := simulatedRecordsWarehouseSettings(t, platform)
			warehouse := warehouses.Registered(platform).New(newTestSettingsLoader(settings), nil)
			t.Cleanup(func() {
				if err := warehouse.Close(); err != nil {
					t.Errorf("expected warehouse closure, got %v", err)
				}
			})
			err := warehouse.Initialize(t.Context(), warehouses.Development, nil)
			if err != nil {
				t.Fatalf("expected warehouse initialization, got %v", err)
			}

			base := time.Date(2026, 9, 22, 9, 10, 11, 120000000, time.UTC)
			since := base.Add(time.Second)
			rows := [][]any{
				{"account-a", "record-old", base, json.Value(`{"source_record_id":"record-old","name":"Older"}`)},
				{"account-a", "record-b", since, json.Value(`{"source_record_id":"record-b","name":"Beta"}`)},
				{"account-a", "record-a", since,
					json.Value(`{"customer_id":"record-a","first_name":"Ada","address":{"country":"IT"},"photo":"/photos/catalog/256/face-000001.webp"}`)},
				{"account-a", "record-c", since.Add(time.Microsecond),
					json.Value(`{"source_record_id":"record-c","name":"Microsecond"}`)},
				{"account-a", "record-d", since.Add(2 * time.Microsecond),
					json.Value(`{"source_record_id":"record-d","name":"Later"}`)},
				{"account-b", "record-a", since, json.Value(`{"source_record_id":"record-a","name":"Other account"}`)},
			}
			err = warehouse.Merge(t.Context(), simulatedSystemRecordsTable, rows, nil)
			if err != nil {
				t.Fatalf("expected simulated-record seed, got %v", err)
			}

			store := &Store{mc: newModeCoordinator(state.Normal)}
			store.wh.Store(warehouse)
			testSimulatedSystemRecords(t, store, since)

		})
	}

}

func simulatedRecordsWarehouseSettings(t *testing.T, platform string) json.Value {

	t.Helper()
	switch platform {
	case "PostgreSQL":
		const (
			database = "test_simulated_records"
			username = "test_simulated_records"
			password = "test_simulated_records"
		)
		ctx := context.Background()
		container, err := postgres.Run(ctx,
			testimages.PostgreSQL,
			postgres.WithDatabase(database),
			postgres.WithUsername(username),
			postgres.WithPassword(password),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second)),
		)
		if err != nil {
			t.Fatalf("expected PostgreSQL test container, got %v", err)
		}
		t.Cleanup(func() {
			if err := testcontainers.TerminateContainer(container); err != nil {
				t.Errorf("expected PostgreSQL test container termination, got %v", err)
			}
		})
		host, err := container.Host(ctx)
		if err != nil {
			t.Fatalf("expected PostgreSQL test host, got %v", err)
		}
		port, err := container.MappedPort(ctx, "5432/tcp")
		if err != nil {
			t.Fatalf("expected PostgreSQL test port, got %v", err)
		}
		settings, err := json.Marshal(map[string]any{
			"host": host, "port": port.Num(), "username": username, "password": password,
			"database": database, "schema": "public",
		})
		if err != nil {
			t.Fatalf("expected PostgreSQL settings, got %v", err)
		}
		return settings
	case "Snowflake":
		if os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
			t.Skip("Snowflake credentials are unavailable")
		}
		testEnv, err := snowflaketester.CreateTestEnvironment()
		if err != nil {
			t.Fatalf("expected Snowflake test environment, got %v", err)
		}
		t.Cleanup(func() {
			if err := testEnv.Teardown(); err != nil {
				t.Errorf("expected Snowflake test environment teardown, got %v", err)
			}
		})
		return testEnv.Settings().JSON()
	default:
		panic("unsupported warehouse platform")
	}

}

func testSimulatedSystemRecords(t *testing.T, store *Store, since time.Time) {

	t.Helper()

	full, err := store.SimulatedSystemRecords(t.Context(), "account-a", SimulatedRecordQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected full simulated-record read, got %v", err)
	}
	assertSimulatedRecordPage(t, full,
		[]string{"record-old", "record-a", "record-b", "record-c", "record-d"}, false, SimulatedRecordsCursor{})
	if !full.Records[3].UpdatedAt.After(full.Records[2].UpdatedAt) ||
		full.Records[3].UpdatedAt.Sub(full.Records[2].UpdatedAt) != time.Microsecond {
		t.Fatalf("expected records one microsecond apart, got %v and %v",
			full.Records[2].UpdatedAt, full.Records[3].UpdatedAt)
	}
	for _, record := range full.Records {
		if record.UpdatedAt.Location() != time.UTC || record.UpdatedAt.Nanosecond()%1000 != 0 {
			t.Fatalf("expected UTC microsecond timestamp, got %v", record.UpdatedAt)
		}
	}

	var attributes map[string]any
	err = json.Unmarshal(full.Records[1].Attributes, &attributes)
	if err != nil {
		t.Fatalf("expected complete JSON document, got %v", err)
	}
	expectedAttributes := map[string]any{
		"customer_id": "record-a", "first_name": "Ada", "address": map[string]any{"country": "IT"},
		"photo": "/photos/catalog/256/face-000001.webp",
	}
	if !reflect.DeepEqual(attributes, expectedAttributes) {
		t.Fatalf("expected complete JSON document %v, got %v", expectedAttributes, attributes)
	}
	if _, ok := attributes["simulated_system"]; ok {
		t.Fatalf("expected no simulated_system bookkeeping, got %v", attributes)
	}
	if _, ok := attributes["updated_at"]; ok {
		t.Fatalf("expected no updated_at bookkeeping, got %v", attributes)
	}

	inclusive, err := store.SimulatedSystemRecords(t.Context(), "account-a", SimulatedRecordQuery{
		Since: since.In(time.FixedZone("test", 3600)).Add(999 * time.Nanosecond), Limit: 10,
	})
	if err != nil {
		t.Fatalf("expected inclusive Since read, got %v", err)
	}
	assertSimulatedRecordPage(t, inclusive,
		[]string{"record-a", "record-b", "record-c", "record-d"}, false, SimulatedRecordsCursor{})

	var got []string
	query := SimulatedRecordQuery{Limit: 2}
	for {
		page, err := store.SimulatedSystemRecords(t.Context(), "account-a", query)
		if err != nil {
			t.Fatalf("expected paginated simulated-record read, got %v", err)
		}
		for _, record := range page.Records {
			got = append(got, record.ID)
		}
		if !page.HasMore {
			if page.Next != (SimulatedRecordsCursor{}) {
				t.Fatalf("expected zero final cursor, got %+v", page.Next)
			}
			break
		}
		if len(page.Records) == 0 {
			t.Fatal("expected records on a non-final page")
		}
		last := page.Records[len(page.Records)-1]
		if page.Next.ID != last.ID || !page.Next.UpdatedAt.Equal(last.UpdatedAt) {
			t.Fatalf("expected last record cursor %+v, got %+v", last, page.Next)
		}
		query.After = page.Next
	}
	expectedIDs := []string{"record-old", "record-a", "record-b", "record-c", "record-d"}
	if !slices.Equal(got, expectedIDs) {
		t.Fatalf("expected exactly-once stable pagination %v, got %v", expectedIDs, got)
	}

	combined, err := store.SimulatedSystemRecords(t.Context(), "account-a", SimulatedRecordQuery{
		Since: since, After: SimulatedRecordsCursor{UpdatedAt: since, ID: "record-a"}, Limit: 10,
	})
	if err != nil {
		t.Fatalf("expected combined Since and After read, got %v", err)
	}
	assertSimulatedRecordPage(t, combined, []string{"record-b", "record-c", "record-d"}, false, SimulatedRecordsCursor{})

	first, err := store.SimulatedSystemRecords(t.Context(), "account-a", SimulatedRecordQuery{Limit: 2})
	if err != nil {
		t.Fatalf("expected non-final page, got %v", err)
	}
	assertSimulatedRecordPage(t, first, []string{"record-old", "record-a"}, true,
		SimulatedRecordsCursor{UpdatedAt: since, ID: "record-a"})

	empty, err := store.SimulatedSystemRecords(t.Context(), "account-a", SimulatedRecordQuery{
		Since: since.Add(3 * time.Microsecond), Limit: 10,
	})
	if err != nil {
		t.Fatalf("expected empty simulated-record page, got %v", err)
	}
	assertSimulatedRecordPage(t, empty, nil, false, SimulatedRecordsCursor{})

	isolated, err := store.SimulatedSystemRecords(t.Context(), "account-b", SimulatedRecordQuery{Limit: 10})
	if err != nil {
		t.Fatalf("expected isolated account read, got %v", err)
	}
	assertSimulatedRecordPage(t, isolated, []string{"record-a"}, false, SimulatedRecordsCursor{})
	var isolatedAttributes map[string]any
	err = json.Unmarshal(isolated.Records[0].Attributes, &isolatedAttributes)
	if err != nil {
		t.Fatalf("expected isolated account JSON, got %v", err)
	}
	if isolatedAttributes["name"] != "Other account" {
		t.Fatalf("expected account-b document, got %v", isolatedAttributes)
	}

}

func assertSimulatedRecordPage(t *testing.T, page SimulatedRecordPage, ids []string, hasMore bool, next SimulatedRecordsCursor) {

	t.Helper()
	got := make([]string, len(page.Records))
	for i, record := range page.Records {
		got[i] = record.ID
	}
	if !slices.Equal(got, ids) {
		t.Fatalf("expected record IDs %v, got %v", ids, got)
	}
	if page.HasMore != hasMore {
		t.Fatalf("expected HasMore %t, got %t", hasMore, page.HasMore)
	}
	if page.Next.ID != next.ID || !page.Next.UpdatedAt.Equal(next.UpdatedAt) {
		t.Fatalf("expected next cursor %+v, got %+v", next, page.Next)
	}

}
