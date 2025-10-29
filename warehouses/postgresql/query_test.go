// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"strings"
	"testing"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/types"
)

// Test_appendJoins renders join clauses and checks for errors on invalid join
// conditions.
func Test_appendJoins(t *testing.T) {
	join := meergo.Join{
		Type:  meergo.InnerJoin,
		Table: "t2",
		Condition: meergo.NewBaseExpr(
			meergo.Column{Name: "id", Type: types.Int(32)},
			meergo.OpIs,
			meergo.Column{Name: "fk", Type: types.Int(32)},
		),
	}
	var b strings.Builder
	if err := appendJoins(&b, []meergo.Join{join}); err != nil {
		t.Fatal(err)
	}
	expected := " JOIN \"t2\" ON \"id\" = \"fk\""
	if b.String() != expected {
		t.Fatalf("expected %q, got %q", expected, b.String())
	}

	bad := meergo.Join{
		Type:      meergo.InnerJoin,
		Table:     "t3",
		Condition: meergo.NewBaseExpr(meergo.Column{Name: "bad name", Type: types.Int(32)}, meergo.OpIs, 1),
	}
	b.Reset()
	if err := appendJoins(&b, []meergo.Join{bad}); err == nil {
		t.Fatal("expected error for bad join condition")
	}
}
