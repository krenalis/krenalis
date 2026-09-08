// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"testing"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestQueryOrdering verifies that an empty OrderBy is omitted and descending order is applied to every ordered column.
func TestQueryOrdering(t *testing.T) {

	columns := []warehouses.Column{{Name: "_kpid", Type: types.UUID()}}
	for _, tc := range []struct {
		name      string
		orderBy   []warehouses.Column
		orderDesc bool
		want      string
	}{
		{
			name:    "empty",
			orderBy: []warehouses.Column{},
			want:    `SELECT "_KPID" FROM "PROFILES"`,
		},
		{
			name: "descending",
			orderBy: []warehouses.Column{
				{Name: "_updated_at", Type: types.DateTime()},
				{Name: "_kpid", Type: types.UUID()},
			},
			orderDesc: true,
			want:      `SELECT "_KPID" FROM "PROFILES" ORDER BY "_UPDATED_AT" DESC, "_KPID" DESC`,
		},
	} {

		t.Run(tc.name, func(t *testing.T) {

			db, queries := newCheckReadOnlyTestDB(t, []checkReadOnlyQuery{{
				match: `SELECT`,
				cols:  []string{"_KPID"},
			}})
			defer db.Close()

			rows, _, err := (&Snowflake{db: db}).Query(t.Context(), warehouses.RowQuery{
				Table:     "profiles",
				Columns:   columns,
				OrderBy:   tc.orderBy,
				OrderDesc: tc.orderDesc,
			}, false)
			if err != nil {
				t.Fatal(err)
			}
			err = rows.Close()
			if err != nil {
				t.Fatal(err)
			}
			if len(*queries) != 1 {
				t.Fatalf("expected 1 query, got %d: %q", len(*queries), *queries)
			}
			if (*queries)[0] != tc.want {
				t.Fatalf("expected query %q, got %q", tc.want, (*queries)[0])
			}

		})

	}

}
