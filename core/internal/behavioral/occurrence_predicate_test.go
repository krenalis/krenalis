// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package behavioral

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
)

// TestOccurrencePredicateValidation protects the internal typed entry point.
func TestOccurrencePredicateValidation(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	base := Scope{Event: "Order Completed", Grain: "registered-message", Start: at.Add(-24 * time.Hour), End: at}
	period := Scope{Elapsed: &ElapsedPeriod{Older: time.Hour}}
	history, err := ResolveHistory("registered-queryable-events", "1")
	if err != nil {
		t.Fatalf("expected recorded history resolved, got %v", err)
	}
	unsupported := OccurrencePredicate{OccurredIn: &Scope{History: history}}
	invalid := OccurrencePredicate{Record: &state.WhereCondition{}}
	for _, test := range []struct {
		name        string
		body        *OccurrencePredicate
		unsupported bool
	}{
		{"nil", nil, false},
		{"empty", &OccurrencePredicate{}, false},
		{"mixed", &OccurrencePredicate{OccurredIn: &period, Record: &state.WhereCondition{}}, false},
		{"invalid group", &OccurrencePredicate{Operator: -1, Children: []OccurrencePredicate{{OccurredIn: &period}}}, false},
		{"invalid period", &OccurrencePredicate{OccurredIn: &Scope{Start: at, End: at.Add(-time.Hour)}}, false},
		{"full history", &unsupported, true},
		{"unsupported before invalid", &OccurrencePredicate{Children: []OccurrencePredicate{unsupported, invalid}}, false},
		{"invalid before unsupported", &OccurrencePredicate{Children: []OccurrencePredicate{invalid, unsupported}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := &Rule{Experience: &Experience{Scope: base, Verify: test.body}}
			err := Validate(root, at, Page{Limit: 1})
			if err != nil {
				_, unsupported := errors.AsType[*UnsupportedError](err)
				if unsupported != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			t.Fatal("expected invalid or unsupported verification, got success")
		})
	}
	for _, test := range []struct{ depth, size int }{{1, 100}, {1, 101}, {2, 1}, {3, 1}} {
		body := &OccurrencePredicate{Children: make([]OccurrencePredicate, test.size)}
		for i := range body.Children {
			body.Children[i] = OccurrencePredicate{OccurredIn: &period}
		}
		for depth := 1; depth < test.depth; depth++ {
			body = &OccurrencePredicate{Children: []OccurrencePredicate{*body}}
		}
		err := Validate(&Rule{Experience: &Experience{Scope: base, Verify: body}}, at, Page{Limit: 1})
		invalid := test.depth == 3 || test.size == 101
		if err != nil {
			if !invalid {
				t.Fatalf("expected depth=%d size=%d accepted, got %v", test.depth, test.size, err)
			}
			continue
		}
		if invalid {
			t.Fatalf("expected depth=%d size=%d rejected, got success", test.depth, test.size)
		}
	}
	invalidBase := base
	invalidBase.Start, invalidBase.End = base.End, base.Start
	err = Validate(&Rule{Experience: &Experience{Scope: invalidBase, Verify: &unsupported}}, at, Page{Limit: 1})
	if err != nil {
		if _, unavailable := errors.AsType[*UnsupportedError](err); unavailable {
			t.Fatalf("expected invalid base before unsupported verify, got %v", err)
		}
		return
	}
	t.Fatal("expected invalid base, got success")
}

func recordCountWhere(t *testing.T, where *state.Where) *OccurrencePredicate {
	t.Helper()
	predicate, err := RecordPredicate(where)
	if err != nil {
		t.Fatalf("expected prepared Record predicate, got %v", err)
	}
	return predicate
}
