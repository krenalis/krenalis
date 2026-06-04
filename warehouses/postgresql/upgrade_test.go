// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"reflect"
	"testing"
)

func TestPostgreSQLMigrateBase58ColumnQuery(t *testing.T) {
	query, args := postgresqlMigrateBase58ColumnQuery("krenalis_events", "connection_id", map[int]string{
		100: "4fr7Kp9ZaQ2m",
		3:   "8QaT3mN7KxP5",
	})
	const wantQuery = `WITH "id_map" ("old_id", "new_id") AS (VALUES ($1::integer, $2::text),($3::integer, $4::text))
UPDATE "krenalis_events" SET "connection_id_base58" = "id_map"."new_id"
FROM "id_map"
WHERE "connection_id" = "id_map"."old_id"`
	if query != wantQuery {
		t.Fatalf("unexpected query:\n%s", query)
	}
	wantArgs := []any{3, "8QaT3mN7KxP5", 100, "4fr7Kp9ZaQ2m"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected args: %#v", args)
	}
}
