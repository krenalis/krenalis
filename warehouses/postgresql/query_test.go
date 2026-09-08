// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// Test_appendJoins renders join clauses and checks for errors on invalid join
// conditions.
func Test_appendJoins(t *testing.T) {
	join := warehouses.Join{
		Type:  warehouses.InnerJoin,
		Table: "t2",
		Condition: warehouses.NewBaseExpr(
			warehouses.Column{Name: "id", Type: types.Int(32)},
			warehouses.OpIs,
			warehouses.Column{Name: "fk", Type: types.Int(32)},
		),
	}
	var b strings.Builder
	if err := appendJoins(&b, []warehouses.Join{join}); err != nil {
		t.Fatal(err)
	}
	expected := " JOIN \"t2\" ON \"id\" = \"fk\""
	if b.String() != expected {
		t.Fatalf("expected %q, got %q", expected, b.String())
	}

	bad := warehouses.Join{
		Type:      warehouses.InnerJoin,
		Table:     "t3",
		Condition: warehouses.NewBaseExpr(warehouses.Column{Name: "bad name", Type: types.Int(32)}, warehouses.OpIs, 1),
	}
	b.Reset()
	if err := appendJoins(&b, []warehouses.Join{bad}); err == nil {
		t.Fatal("expected error for bad join condition")
	}
}

// Test_renderCountQuery verifies counts with optional joins and filters.
func Test_renderCountQuery(t *testing.T) {

	id := warehouses.Column{Name: "id", Type: types.Int(32)}
	joins := []warehouses.Join{{
		Type:      warehouses.InnerJoin,
		Table:     "identities",
		Condition: warehouses.NewBaseExpr(id, warehouses.OpIs, warehouses.Column{Name: "fk", Type: types.Int(32)}),
	}}
	where := warehouses.NewBaseExpr(id, warehouses.OpIs, 1)

	for _, tc := range []struct {
		name  string
		joins []warehouses.Join
		where warehouses.Expr
		want  string
	}{
		{name: "all rows", want: `SELECT COUNT(*) FROM "profiles"`},
		{name: "filtered rows", where: where, want: `SELECT COUNT(*) FROM "profiles" WHERE "id" = 1`},
		{
			name:  "joined rows",
			joins: joins,
			want:  `SELECT COUNT(*) FROM "profiles" JOIN "identities" ON "id" = "fk"`,
		},
		{
			name:  "filtered joined rows",
			joins: joins,
			where: where,
			want:  `SELECT COUNT(*) FROM "profiles" JOIN "identities" ON "id" = "fk" WHERE "id" = 1`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			statement, err := renderCountQuery("profiles", tc.joins, tc.where)
			if err != nil {
				t.Fatal(err)
			}
			if statement != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, statement)
			}
		})
	}

}
