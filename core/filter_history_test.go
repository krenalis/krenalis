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

const historyPeriodJSON = `{"kind":"fullRelevantHistory","history":` +
	`{"definition":"registered-queryable-events","meaningRevision":"1"}}`

// TestFilterHistoryRoundTrip keeps the explicit descriptor in the document.
func TestFilterHistoryRoundTrip(t *testing.T) {

	input := []byte(`{"operator":"and","rules":[` + strings.Replace(countConditionJSON,
		`{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}`, historyPeriodJSON, 1) + `]}`)
	var filter Filter
	err := json.Unmarshal(input, &filter)
	if err != nil {
		t.Fatalf("expected history filter decoding, got %v", err)
	}
	before, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected history filter encoding, got %v", err)
	}
	if !bytes.Equal(before, input) {
		t.Fatalf("expected explicit history reference round-trip %s, got %s", input, before)
	}
	clear(input)

	nodes := 0
	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	root, err := prepareUnifiedRule(&filter, schema, 0, &nodes)
	if err != nil {
		t.Fatalf("expected resolved history reference, got %v", err)
	}
	scope := root.Children[0].Count.Scope
	if scope.History == nil || !scope.Start.IsZero() || !scope.End.IsZero() || nodes != 2 {
		t.Fatalf("expected history descriptor without invented bounds at two nodes, got %+v and %d", scope, nodes)
	}
	err = unified.Validate(&root, time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
	if err != nil {
		t.Fatalf("expected admitted history scope, got %v", err)
	}
	after, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected encoding after preparation, got %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("expected unchanged filter %s, got %s", before, after)
	}

	_, _, err = prepareFilter(&filter, schema)
	if err != nil {
		return // Record-only contexts must still reject Count, including history.
	}
	t.Fatal("expected Record-only rejection, got success")

}

// TestFilterHistoryValidation rejects unresolved and mixed temporal forms.
func TestFilterHistoryValidation(t *testing.T) {
	for _, test := range []struct{ name, period string }{
		{"missing history", `{"kind":"fullRelevantHistory"}`},
		{"null history", `{"kind":"fullRelevantHistory","history":null}`},
		{"history string", `{"kind":"fullRelevantHistory","history":"registered-queryable-events"}`},
		{"empty history", `{"kind":"fullRelevantHistory","history":{}}`},
		{"unknown definition", strings.Replace(historyPeriodJSON, `registered-queryable-events`, `all-real-events`, 1)},
		{"unknown revision", strings.Replace(historyPeriodJSON, `"1"`, `"2"`, 1)},
		{"missing definition", strings.Replace(historyPeriodJSON, `"definition":"registered-queryable-events",`, ``, 1)},
		{"missing revision", strings.Replace(historyPeriodJSON, `,"meaningRevision":"1"`, ``, 1)},
		{"null definition", strings.Replace(historyPeriodJSON, `"registered-queryable-events"`, `null`, 1)},
		{"numeric revision", strings.Replace(historyPeriodJSON, `"1"`, `1`, 1)},
		{"oversized definition", strings.Replace(historyPeriodJSON, `registered-queryable-events`, strings.Repeat("x", 261), 1)},
		{"unknown member", strings.Replace(historyPeriodJSON, `"history":{`, `"history":{"x":0,`, 1)},
		{"duplicate definition", strings.Replace(historyPeriodJSON, `"history":{`, `"history":{"definition":"other",`, 1)},
		{"duplicate history", strings.Replace(historyPeriodJSON, `"history":`, `"history":{},"history":`, 1)},
		{"explicit start", strings.Replace(historyPeriodJSON, `"history":`, `"start":"2026-01-01T00:00:00Z","history":`, 1)},
		{"explicit end", strings.Replace(historyPeriodJSON, `"history":`, `"end":"2026-01-01T00:00:00Z","history":`, 1)},
		{"duration", strings.Replace(historyPeriodJSON, `"history":`, `"duration":{"value":1,"unit":"day"},"history":`, 1)},
		{"older", strings.Replace(historyPeriodJSON, `"history":`, `"older":{"value":1,"unit":"day"},"history":`, 1)},
		{"newer", strings.Replace(historyPeriodJSON, `"history":`, `"newer":{"value":0,"unit":"day"},"history":`, 1)},
		{"calendar count", strings.Replace(historyPeriodJSON, `"history":`, `"count":1,"history":`, 1)},
		{"empty calendar unit", strings.Replace(historyPeriodJSON, `"history":`, `"unit":"","history":`, 1)},
		{"history on elapsed", strings.Replace(historyPeriodJSON, `"fullRelevantHistory"`, `"elapsedLookback","duration":{"value":1,"unit":"day"}`, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			condition := countFilterCondition()
			err := json.Unmarshal([]byte(test.period), &condition.Count.Period)
			if err == nil {
				_, err = prepareFilterCount(condition, nil)
			}
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
					t.Fatalf("expected invalid history configuration, got %v", err)
				}
				return
			}
			t.Fatal("expected invalid history configuration, got success")
		})
	}
}
