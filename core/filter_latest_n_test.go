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
	"github.com/krenalis/krenalis/warehouses"
)

const latestNConditionJSON = `{"latestN":{"base":{"source":{"kind":"recordedEvents",` +
	`"name":"Order Completed","grain":"registered-message"},"period":{"kind":"fullRelevantHistory",` +
	`"history":{"definition":"registered-queryable-events","meaningRevision":"1"}}},` +
	`"size":3,"predicate":{"operator":"and","rules":[` +
	`{"property":"properties.order_id","operator":"starts with","values":["WEB-"]}]},` +
	`"quantifier":{"kind":"all"}}}`

// TestExplainLatestNSelection observes set ambiguity, not permutations of ties.
func TestExplainLatestNSelection(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		candidates    int64
		absent        bool
		kind, absence string
	}{
		{0, true, "absent", "certain"}, {2, true, "ambiguous", "possible"},
		{2, false, "unique", "impossible"}, {3, false, "ambiguous", "impossible"},
	} {
		index := 0
		result, err := explainProfileRule(&warehouses.LatestOccurrences{Size: 2, End: at},
			unified.Rule{LatestN: &unified.LatestN{Size: 2}}, []warehouses.FilterNodeOutcome{{Truth: warehouses.FilterUnknown,
				Experience: &warehouses.OccurrenceSelection{Candidates: test.candidates, AbsencePossible: test.absent}}},
			&index, at)
		if err != nil {
			t.Fatalf("expected block explanation, got %v", err)
		}
		if result.Kind != "latestN" || result.LatestN == nil || result.LatestN.Selection.Kind != test.kind ||
			result.LatestN.Selection.Absence != test.absence || result.LatestN.Selection.Candidates != test.candidates ||
			result.LatestN.Size != 2 || result.LatestN.Quantifier.Kind != "all" || index != 1 {
			t.Fatalf("expected %s set with %s absence, got %+v", test.kind, test.absence, result.LatestN)
		}
	}
}

// TestFilterLatestNCodec checks the main-tree discriminator and closed objects.
func TestFilterLatestNCodec(t *testing.T) {

	var filter Filter
	err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+latestNConditionJSON+`]}`), &filter)
	if err != nil {
		t.Fatalf("expected latestN fixture, got %v", err)
	}
	if !filter.RequiresUnified() || filter.HasCount() {
		t.Fatal("expected Unified without Count, got wrong detection")
	}
	encoded, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected latestN round-trip, got %v", err)
	}
	var restored Filter
	err = json.Unmarshal(encoded, &restored)
	if err != nil {
		t.Fatalf("expected round-trip decoding, got %v", err)
	}
	again, err := json.Marshal(&restored)
	if err != nil {
		t.Fatalf("expected round-trip encoding, got %v", err)
	}
	if !bytes.Equal(encoded, again) {
		t.Fatalf("expected unchanged round-trip %s, got %s", encoded, again)
	}

	t.Run("owned size", func(t *testing.T) {
		data := []byte(latestNConditionJSON[len(`{"latestN":`) : len(latestNConditionJSON)-1])
		var latest FilterLatestN
		err := latest.UnmarshalJSON(data)
		if err != nil {
			t.Fatalf("expected decoded subject, got %v", err)
		}
		clear(data)
		if string(latest.Size) != "3" {
			t.Fatalf("expected retained size 3, got %q", latest.Size)
		}
	})

	_, _, err = prepareFilter(&filter, types.Object([]types.Property{{Name: "x", Type: types.String()}}))
	if err == nil {
		t.Fatal("expected Record-only rejection, got success")
	}

	for _, test := range []struct{ name, old, value string }{
		{"null rule", `"latestN":{`, `"latestN":null,"extra":{`},
		{"missing size", `"size":3,`, ``}, {"null size", `"size":3`, `"size":null`},
		{"unknown field", `"size":3`, `"size":3,"other":true`},
		{"null predicate", `"predicate":{"operator":"and","rules":[{"property":"properties.order_id","operator":"starts with","values":["WEB-"]}]}`, `"predicate":null`},
		{"occurredIn", `{"property":"properties.order_id","operator":"starts with","values":["WEB-"]}`, `{"occurredIn":{"kind":"elapsedLookback","duration":{"value":1,"unit":"day"}}}`},
		{"other subject", `"latestN":`, `"count":{},"latestN":`},
		{"outer operator", `"latestN":`, `"operator":"is","latestN":`},
		{"null base", `"base":{`, `"base":null,"extra":{`},
		{"null quantifier", `{"kind":"all"}`, `null`},
		{"unknown quantifier", `{"kind":"all"}`, `{"kind":"none"}`},
		{"null kind", `{"kind":"all"}`, `{"kind":null}`},
		{"unknown quantifier field", `{"kind":"all"}`, `{"kind":"all","extra":1}`},
		{"all operator", `{"kind":"all"}`, `{"kind":"all","operator":"is"}`},
		{"some values", `{"kind":"all"}`, `{"kind":"some","values":[]}`},
		{"cardinality missing values", `{"kind":"all"}`, `{"kind":"cardinality","operator":"is"}`},
		{"cardinality null values", `{"kind":"all"}`, `{"kind":"cardinality","operator":"is","values":null}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var condition FilterLatestNCondition
			err := json.Unmarshal([]byte(strings.Replace(latestNConditionJSON, test.old, test.value, 1)), &condition)
			if err == nil {
				t.Fatal("expected strict decoding error, got success")
			}
		})
	}

}

// TestFilterLatestNQuantifiers classifies incompatibility before availability.
func TestFilterLatestNQuantifiers(t *testing.T) {
	for _, test := range []struct {
		name, quantifier string
		invalid          bool
	}{
		{"all", `{"kind":"all"}`, false}, {"some", `{"kind":"some"}`, false},
		{"exactly size", `{"kind":"cardinality","operator":"is","values":[3]}`, false},
		{"at least size", `{"kind":"cardinality","operator":"is greater than or equal to","values":[3]}`, false},
		{"range at size", `{"kind":"cardinality","operator":"is between","values":[3,3]}`, false},
		{"range beyond size", `{"kind":"cardinality","operator":"is between","values":[3,5]}`, false},
		{"range point", `{"kind":"cardinality","operator":"is between","values":[2,2]}`, false},
		{"range zero", `{"kind":"cardinality","operator":"is between","values":[0,0]}`, false},
		{"exactly above size", `{"kind":"cardinality","operator":"is","values":[4]}`, true},
		{"range point above size", `{"kind":"cardinality","operator":"is between","values":[4,4]}`, true},
		{"exactly impossible", `{"kind":"cardinality","operator":"is","values":[5]}`, true},
		{"at least impossible", `{"kind":"cardinality","operator":"is greater than or equal to","values":[5]}`, true},
		{"at most redundant", `{"kind":"cardinality","operator":"is less than or equal to","values":[5]}`, false},
		{"range redundant", `{"kind":"cardinality","operator":"is between","values":[0,5]}`, false},
		{"range impossible", `{"kind":"cardinality","operator":"is between","values":[4,5]}`, true},
		{"range inverted", `{"kind":"cardinality","operator":"is between","values":[2,1]}`, true},
		{"zero", `{"kind":"cardinality","operator":"is greater than or equal to","values":[0]}`, false},
		{"negative zero", `{"kind":"cardinality","operator":"is","values":[-0]}`, false},
		{"one", `{"kind":"cardinality","operator":"is","values":[1]}`, false},
		{"huge impossible", `{"kind":"cardinality","operator":"is","values":[1000000000000000000000000000000]}`, true},
		{"huge redundant", `{"kind":"cardinality","operator":"is less than or equal to","values":[1000000000000000000000000000000]}`, false},
		{"fraction", `{"kind":"cardinality","operator":"is","values":[1.0]}`, true},
		{"exponent", `{"kind":"cardinality","operator":"is","values":[1e0]}`, true},
		{"negative", `{"kind":"cardinality","operator":"is","values":[-1]}`, true},
		{"string", `{"kind":"cardinality","operator":"is","values":["1"]}`, true},
		{"different", `{"kind":"cardinality","operator":"is not","values":[1]}`, true},
		{"greater", `{"kind":"cardinality","operator":"is greater than","values":[1]}`, true},
		{"less", `{"kind":"cardinality","operator":"is less than","values":[1]}`, true},
		{"too many", `{"kind":"cardinality","operator":"is","values":[1,2]}`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var condition FilterLatestNCondition
			err := json.Unmarshal([]byte(strings.Replace(latestNConditionJSON,
				`{"kind":"all"}`, test.quantifier, 1)), &condition)
			if err != nil {
				t.Fatalf("expected structurally decoded quantifier, got %v", err)
			}
			_, err = prepareFilterLatestN(&condition, nil)
			if err != nil {
				_, unsupported := errors.AsType[*unified.UnsupportedError](err)
				if unsupported == test.invalid || test.name == "all" {
					t.Fatalf("expected invalid=%v, got %v", test.invalid, err)
				}
				return
			}
			if test.name != "all" {
				t.Fatal("expected invalid or unavailable quantifier, got success")
			}
		})
	}
}

// TestFilterLatestNValidation checks budgets and error precedence.
func TestFilterLatestNValidation(t *testing.T) {

	var condition FilterLatestNCondition
	err := json.Unmarshal([]byte(latestNConditionJSON), &condition)
	if err != nil {
		t.Fatalf("expected latestN fixture, got %v", err)
	}
	for _, test := range []struct {
		size    string
		invalid bool
	}{
		{"1", false}, {"10", false}, {"0", true}, {"11", true},
		{"-1", true}, {"1.0", true}, {"1e0", true}, {`"1"`, true},
	} {
		t.Run("size="+test.size, func(t *testing.T) {
			var modified FilterLatestNCondition
			err := json.Unmarshal([]byte(strings.Replace(latestNConditionJSON, `"size":3`,
				`"size":`+test.size, 1)), &modified)
			if err != nil {
				t.Fatalf("expected structurally decoded size %s, got %v", test.size, err)
			}
			_, err = prepareFilterLatestN(&modified, nil)
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported || !test.invalid {
					t.Fatalf("expected invalid=%v for size %s, got %v", test.invalid, test.size, err)
				}
				return
			}
			if test.invalid {
				t.Fatalf("expected invalid size %s, got success", test.size)
			}
		})
	}
	for _, local := range []bool{true, false} {
		for _, test := range []struct{ size, depth int }{{100, 1}, {101, 1}, {1, 2}, {1, 3}} {
			body := &Filter{Operator: OpAnd, Rules: make([]FilterRule, test.size)}
			for i := range body.Rules {
				body.Rules[i] = condition.LatestN.Predicate.Rules[0]
			}
			for depth := 1; depth < test.depth; depth++ {
				body = &Filter{Operator: OpAnd, Rules: []FilterRule{body}}
			}
			modified := condition
			if local {
				modified.LatestN.Predicate = body
			} else {
				modified.LatestN.Base.Where = body
			}
			_, err := prepareFilterLatestN(&modified, nil)
			invalid := test.size == 101 || test.depth > 2
			if (err != nil) != invalid {
				t.Fatalf("expected invalid=%v for size %d depth %d, got %v", invalid, test.size, test.depth, err)
			}
		}
	}

	unsupported := condition
	unsupported.LatestN.Quantifier.Kind = "some"
	invalid := condition
	invalid.LatestN.Size = json.Value("0")
	for _, rules := range [][]FilterRule{{&unsupported, &invalid}, {&invalid, &unsupported}} {
		nodes := 0
		_, err := prepareUnifiedRule(&Filter{Operator: OpAnd, Rules: rules},
			types.Object([]types.Property{{Name: "x", Type: types.String()}}), 0, &nodes)
		if err != nil {
			if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
				t.Fatalf("expected invalid sibling before unavailable, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid sibling, got success")
	}
	unsupported.LatestN.Base.Source.Name = ""
	_, err = prepareFilterLatestN(&unsupported, nil)
	if err != nil {
		if _, unavailable := errors.AsType[*unified.UnsupportedError](err); unavailable {
			t.Fatalf("expected invalid base before quantifier availability, got %v", err)
		}
		return
	}
	t.Fatal("expected invalid base, got success")

}
