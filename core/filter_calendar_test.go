// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const calendarReferenceJSON = `{"definition":"gregorian-monday/Europe/Rome","meaningRevision":"1"}`
const calendarConditionJSON = `{"count":{"source":{"kind":"recordedEvents","name":"Order Completed",` +
	`"grain":"registered-message"},"period":{"kind":"lastCompleteCalendarPeriods","unit":"month","count":1}},` +
	`"operator":"is greater than or equal to","values":[1]}`

func TestFilterCalendarInvalidBeforeUnsupported(t *testing.T) {
	var root Filter
	err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+
		strings.Replace(calendarConditionJSON, `"count":1`, `"count":18446744073709551616`, 1)+
		`],"defaultCalendar":`+calendarReferenceJSON+`}`), &root)
	if err != nil {
		t.Fatalf("expected valid-language calendar syntax, got %v", err)
	}
	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	nodes := 0
	_, err = prepareUnifiedRule(&root, schema, 0, &nodes)
	if err != nil {
		if _, ok := errors.AsType[*unified.UnsupportedError](err); !ok {
			t.Fatalf("expected a genuinely unavailable sibling, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected unavailable calendar count, got success")
	}
	for _, reverse := range []bool{false, true} {
		invalid := &FilterCountCondition{Count: FilterCount{
			Source: FilterCountSource{Kind: "recordedEvents", Name: "Order Completed", Grain: "registered-message"},
			Period: FilterPeriod{Kind: "lastCompleteCalendarPeriods", Unit: "year", Count: json.Value(`1`)},
		}, Operator: OpIs, Values: filterValues(0)}
		rules := []FilterRule{root.Rules[0], invalid}
		if reverse {
			rules[0], rules[1] = rules[1], rules[0]
		}
		filter := &Filter{Operator: OpAnd, Rules: rules, DefaultCalendar: root.DefaultCalendar}
		nodes = 0
		_, err := prepareUnifiedRule(filter, schema, 0, &nodes)
		if err != nil {
			if _, ok := errors.AsType[*unified.UnsupportedError](err); ok {
				t.Fatalf("expected invalid calendar before unavailable sibling, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid calendar parameters, got success")
	}
}

func TestFilterCalendarRoundTrip(t *testing.T) {

	input := []byte(`{"operator":"and","rules":[{"operator":"or","rules":[` +
		calendarConditionJSON + `]}],"defaultCalendar":` + calendarReferenceJSON + `}`)
	var filter Filter
	err := json.Unmarshal(input, &filter)
	if err != nil {
		t.Fatalf("expected calendar filter decoding, got %v", err)
	}
	before, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected calendar filter encoding, got %v", err)
	}
	if !bytes.Equal(before, input) {
		t.Fatalf("expected relative calendar round-trip %s, got %s", input, before)
	}
	clear(input)
	nodes := 0
	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	root, err := prepareUnifiedRule(&filter, schema, 0, &nodes)
	if err != nil {
		t.Fatalf("expected inherited calendar preparation, got %v", err)
	}
	scope := root.Children[0].Children[0].Count.Scope
	if scope.Calendar == nil || scope.Calendar.Unit != "month" || scope.Calendar.Count != 1 ||
		!scope.Start.IsZero() || !scope.End.IsZero() || nodes != 3 {
		t.Fatalf("expected unresolved relative calendar at unchanged depth, got %+v and %d nodes", scope, nodes)
	}
	err = unified.Validate(&root, time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
	if err != nil {
		t.Fatalf("expected admitted calendar filter, got %v", err)
	}
	after, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected calendar encoding after preparation, got %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("expected unchanged public filter %s, got %s", before, after)
	}
	record := &Filter{Operator: OpAnd, DefaultCalendar: filter.DefaultCalendar,
		Rules: []FilterRule{&FilterCondition{Property: "country", Operator: OpExists}}}
	_, _, err = prepareFilter(record, schema)
	if err != nil {
		return // Record-only paths must not silently discard calendar metadata.
	}
	t.Fatal("expected Record-only calendar rejection, got success")

}

func TestFilterCalendarStructure(t *testing.T) {
	for _, test := range []struct{ name, input string }{
		{"null calendar", `{"operator":"and","rules":[],"defaultCalendar":null}`},
		{"null definition", `{"operator":"and","rules":[],"defaultCalendar":{"definition":null}}`},
		{"unknown member", `{"operator":"and","rules":[],"defaultCalendar":{"timezone":"UTC"}}`},
		{"duplicate calendar", `{"operator":"and","rules":[],"defaultCalendar":{},"defaultCalendar":{}}`},
		{"duplicate definition", `{"operator":"and","rules":[],"defaultCalendar":{"definition":"a","definition":"b"}}`},
		{"duplicate period count", `{"operator":"and","rules":[` + strings.Replace(calendarConditionJSON, `"count":1`, `"count":1,"count":2`, 1) + `]}`},
		{"nested calendar", `{"operator":"and","rules":[{"operator":"and","rules":[],"defaultCalendar":{}}]}`},
		{"Record condition calendar", `{"operator":"and","rules":[{"property":"x","operator":"exists","defaultCalendar":{}}]}`},
		{"Count condition calendar", `{"operator":"and","rules":[` + strings.TrimSuffix(calendarConditionJSON, `}`) + `,"defaultCalendar":{}}]}`},
		{"occurrence calendar", `{"operator":"and","rules":[{"count":{"source":{},"period":{},"where":{"operator":"and","rules":[],"defaultCalendar":{}}},"operator":"is","values":[0]}]}`},
		{"null unit", `{"operator":"and","rules":[` + strings.Replace(calendarConditionJSON, `"unit":"month"`, `"unit":null`, 1) + `]}`},
		{"foreign empty unit", `{"kind":"elapsedLookback","duration":{"value":1,"unit":"day"},"unit":""}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var target any = &Filter{}
			if test.name == "foreign empty unit" {
				target = &FilterPeriod{}
			}
			err := json.Unmarshal([]byte(test.input), target)
			if err != nil {
				return
			}
			t.Fatal("expected invalid calendar shape, got success")
		})
	}
}

func TestFilterCalendarValidation(t *testing.T) {
	for _, test := range []struct {
		name, from, to string
		unsupported    bool
	}{
		{"zero count", `"count":1`, `"count":0`, false},
		{"negative count", `"count":1`, `"count":-1`, false},
		{"string count", `"count":1`, `"count":"1"`, false},
		{"fractional count", `"count":1`, `"count":1.5`, false},
		{"exponent count", `"count":1`, `"count":1e0`, false},
		{"missing count", `,"count":1`, ``, false},
		{"unknown unit", `"unit":"month"`, `"unit":"year"`, false},
		{"null count", `"count":1`, `"count":null`, false},
		{"missing calendar", `,"defaultCalendar":` + calendarReferenceJSON, ``, false},
		{"local timezone", `Europe/Rome`, `Local`, false},
		{"unknown revision", `"meaningRevision":"1"`, `"meaningRevision":"2"`, false},
		{"missing definition", `"definition":"gregorian-monday/Europe/Rome",`, ``, false},
		{"missing revision", `,"meaningRevision":"1"`, ``, false},
		{"huge count", `"count":1`, `"count":18446744073709551616`, true},
		{"count outside timestamps", `"count":1`, `"count":4000000`, true},
		{"wrong temporal variant", `"unit":"month"`, `"start":"2026-01-01T00:00:00Z","unit":"month"`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := `{"operator":"and","rules":[` + calendarConditionJSON +
				`],"defaultCalendar":` + calendarReferenceJSON + `}`
			input = strings.Replace(input, test.from, test.to, 1)
			var filter Filter
			err := json.Unmarshal([]byte(input), &filter)
			if err == nil {
				nodes := 0
				var root unified.Rule
				root, err = prepareUnifiedRule(&filter,
					types.Object([]types.Property{{Name: "country", Type: types.String()}}), 0, &nodes)
				if err == nil {
					err = unified.Validate(&root, time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
				}
			}
			if err != nil {
				_, unsupported := errors.AsType[*unified.UnsupportedError](err)
				if unsupported != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			t.Fatal("expected rejected calendar parameters, got success")
		})
	}
}
