// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"context"
	"fmt"
	"math"
	"testing"

	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestNumericPostgreSQL probes physical types and actual SQL decisions at int64
// boundaries. This is a PostgreSQL probe, not Snowflake execution evidence.
func TestNumericPostgreSQL(t *testing.T) {

	if testing.Short() {
		t.Skip("requires PostgreSQL")
	}
	container, err := postgres.Run(t.Context(), testimages.PostgreSQL, postgres.WithDatabase("krenalis"),
		postgres.WithUsername("krenalis"), postgres.WithPassword("krenalis"), postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatalf("expected PostgreSQL, got %v", err)
	}
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(container)
		if err != nil {
			t.Errorf("expected container cleanup, got %v", err)
		}
	})
	dsn, err := container.ConnectionString(t.Context(), "sslmode=disable")
	if err != nil {
		t.Fatalf("expected PostgreSQL URL, got %v", err)
	}
	conn, err := pgx.Connect(t.Context(), dsn)
	if err != nil {
		t.Fatalf("expected PostgreSQL connection, got %v", err)
	}
	t.Cleanup(func() {
		err := conn.Close(context.Background())
		if err != nil {
			t.Errorf("expected connection cleanup, got %v", err)
		}
	})
	rows, err := conn.Query(t.Context(), `SELECT
 COALESCE(SUM(CASE WHEN flag=TRUE THEN 1 ELSE 0 END),0),
 COALESCE(SUM(CASE WHEN flag=FALSE THEN 0 ELSE 1 END),0)
 FROM (VALUES (TRUE),(FALSE),(NULL::boolean)) input(flag)`)
	if err != nil {
		t.Fatalf("expected aggregate probe, got %v", err)
	}
	fields := rows.FieldDescriptions()
	if len(fields) != 2 || fields[0].DataTypeOID != pgtype.Int8OID || fields[1].DataTypeOID != pgtype.Int8OID {
		rows.Close()
		t.Fatalf("expected two bigint result columns, got %v", fields)
	}
	if !rows.Next() {
		rows.Close()
		t.Fatal("expected aggregate row, got none")
	}
	var lo, hi int64
	err = rows.Scan(&lo, &hi)
	if err != nil {
		rows.Close()
		t.Fatalf("expected exact aggregate values, got %v", err)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("expected complete probe, got %v", err)
	}
	if lo != 1 || hi != 2 {
		t.Fatalf("expected bounds 1/2, got %d/%d", lo, hi)
	}
	const m = int64(math.MaxInt64)
	maxValue, previous := m, m-1
	ranges := [][]warehouses.CountRange{{{Upper: &previous}}, {{Lower: m}}, {{Upper: &maxValue}}}
	pairs := [][2]int64{{m - 1, m - 1}, {m - 1, m}, {m, m}}
	want := [][]uint8{{1, 2, 0}, {0, 2, 1}, {1, 1, 1}}
	for i, accepted := range ranges {
		for j, pair := range pairs {
			sql := "SELECT " + decision(accepted, "lo",
				"hi") + fmt.Sprintf(" FROM (VALUES (%d::bigint,%d::bigint)) input(lo,hi)", pair[0], pair[1])
			var result *bool
			err := conn.QueryRow(t.Context(), sql).Scan(&result)
			if err != nil {
				t.Fatalf("expected exact numeric decision, got %v", err)
			}
			got := uint8(2)
			if result != nil {
				got = 0
				if *result {
					got = 1
				}
			}
			if got != want[i][j] {
				t.Fatalf("expected truth %d for ranges %d/pair %v, got %d", want[i][j], i, pair, got)
			}
		}
	}

	// Guard and terminal close separately, even when their conjunction has no
	// satisfying count. Probe actual SQL NULL behavior and signed-64-bit edges.
	two := int64(2)
	differentThree := []warehouses.CountRange{{Upper: &two}, {Lower: 4}}
	for _, test := range []struct {
		lo, hi, minimum int64
		want            uint8
	}{
		{1, 3, 3, 2},
		{1, 2, 3, 0},
		{3, 3, 3, 0},
		{4, 4, 3, 1},
		{2, 3, 2, 2},
		{1, 3, 0, 2},
		{m - 1, m, m, 2},
		{m, m, m, 1},
	} {
		sql := "SELECT " + withSampleMinimum(test.minimum, "lo", "hi", decision(differentThree, "lo", "hi")) +
			fmt.Sprintf(" FROM (VALUES (%d::bigint,%d::bigint)) input(lo,hi)", test.lo, test.hi)
		var result *bool
		err := conn.QueryRow(t.Context(), sql).Scan(&result)
		if err != nil {
			t.Fatalf("expected guarded numeric decision, got %v", err)
		}
		got := uint8(2)
		if result != nil {
			got = 0
			if *result {
				got = 1
			}
		}
		if got != test.want {
			t.Fatalf("expected truth %d for %+v, got %d", test.want, test, got)
		}
	}

}
