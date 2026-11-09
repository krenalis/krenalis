// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
)

// TestOccurrenceRecordRoundTrip preserves prepared types, groups and order.
func TestOccurrenceRecordRoundTrip(t *testing.T) {
	where := &state.Where{Operator: state.OpOr, Rules: []state.WhereRule{
		&state.WhereCondition{Property: []string{"properties", "a.b"}, Operator: state.OpIsNotOneOf,
			Values: []any{json.Value(`100`), json.Value(`"100"`), json.Value(`null`)}},
		&state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"int"}, Operator: state.OpIsNot,
				Values: []any{int64(-9223372036854775808)}},
			&state.WhereCondition{Property: []string{"uint"}, Operator: state.OpIsNot,
				Values: []any{uint64(18446744073709551615)}},
			&state.WhereCondition{Property: []string{"decimal"}, Operator: state.OpIsLessThan,
				Values: []any{decimal.MustParse("9007199254740993.0001")}},
			&state.WhereCondition{Property: []string{"timestamp"}, Operator: state.OpIsBefore,
				Values: []any{time.Date(2026, 1, 1, 0, 0, 0, 123456000, time.UTC)}},
			&state.WhereCondition{Property: []string{"text"}, Operator: state.OpDoesNotContain, Values: []any{"x"}},
			&state.WhereCondition{Property: []string{"missing"}, Operator: state.OpDoesNotExist},
		}},
	}}
	for _, input := range []*state.Where{nil, where} {
		predicate, err := RecordPredicate(input)
		if err != nil {
			t.Fatalf("expected prepared Record conversion, got %v", err)
		}
		var restored *state.Where
		if predicate != nil {
			var ok bool
			restored, ok = predicate.recordWhere()
			if !ok {
				t.Fatal("expected an entirely Record predicate, got another predicate")
			}
		}
		if !reflect.DeepEqual(input, restored) {
			t.Fatalf("expected identical prepared Record round-trip %+v, got %+v", input, restored)
		}
	}
}

// TestOccurrenceWhereRejectsTime protects the directly constructed Count model.
func TestOccurrenceWhereRejectsTime(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	period := Scope{Elapsed: &ElapsedPeriod{Older: time.Hour}}
	for _, body := range []*OccurrencePredicate{
		{OccurredIn: &period},
		{Operator: state.OpOr, Children: []OccurrencePredicate{{OccurredIn: &period}}},
	} {
		count := &Count{Scope: Scope{Event: "Order Completed", Grain: "registered-message",
			Start: at.Add(-time.Hour), End: at}, Where: body}
		err := Validate(&Rule{Count: count}, at, Page{Limit: 1})
		if err != nil {
			if err.Error() != "occurredIn requires experience verify" {
				t.Fatalf("expected occurredIn context rejection, got %v", err)
			}
			continue
		}
		t.Fatal("expected occurredIn rejected in Count where, got success")
	}
}
