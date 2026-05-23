// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"reflect"
	"strings"
	"testing"
)

// TestPostgreSQLMigrateIntColumnQuery checks stable query generation.
func TestPostgreSQLMigrateIntColumnQuery(t *testing.T) {
	query, args := postgresqlMigrateIntColumnQuery(`"krenalis_events"`, `"connection_id"`, map[int]int{
		20: 2,
		10: 1,
	})
	if !strings.Contains(query, `UPDATE "krenalis_events" SET "connection_id" = -"id_map"."new_id"`) {
		t.Fatalf("query does not update through negative IDs:\n%s", query)
	}
	if !reflect.DeepEqual(args, []any{10, 1, 20, 2}) {
		t.Fatalf("unexpected args: %#v", args)
	}
}
