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

const occurredInConditionJSON = `{"occurredIn":{"kind":"elapsedLookback",` +
	`"duration":{"value":30,"unit":"day"}}}`
const experienceRecordConditionJSON = `{"property":"properties.order_id","operator":"starts with","values":["WEB-"]}`

// TestFilterOccurredInBudget counts temporal leaves in the body budget.
func TestFilterOccurredInBudget(t *testing.T) {
	var condition FilterExperienceCondition
	err := json.Unmarshal([]byte(strings.Replace(experienceConditionJSON,
		experienceRecordConditionJSON, occurredInConditionJSON, 1)), &condition)
	if err != nil {
		t.Fatalf("expected temporal experience decoded, got %v", err)
	}
	leaf := condition.Experience.Verify.Rules[0]
	for _, test := range []struct{ size, depth int }{{100, 1}, {101, 1}, {1, 2}, {1, 3}} {
		body := &Filter{Operator: OpAnd, Rules: make([]FilterRule, test.size)}
		for i := range body.Rules {
			body.Rules[i] = leaf
		}
		for depth := 1; depth < test.depth; depth++ {
			body = &Filter{Operator: OpOr, Rules: []FilterRule{body}}
		}
		condition.Experience.Verify = body
		_, err := prepareFilterExperience(&condition, nil)
		invalid := test.size == 101 || test.depth == 3
		if err != nil {
			if !invalid {
				t.Fatalf("expected size=%d depth=%d accepted, got %v", test.size, test.depth, err)
			}
			continue
		}
		if invalid {
			t.Fatalf("expected size=%d depth=%d rejected, got success", test.size, test.depth)
		}
	}
}

// TestFilterOccurredInCodec preserves the public period and inherited calendar.
func TestFilterOccurredInCodec(t *testing.T) {
	input := strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, occurredInConditionJSON, 1)
	for _, calendar := range []bool{false, true} {
		root := `{"operator":"and","rules":[` + input + `]}`
		if calendar {
			root = strings.Replace(root, `"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}`,
				`"kind":"lastCompleteCalendarPeriods","unit":"day","count":1`, 1)
			root = strings.TrimSuffix(root, "}") + `,"defaultCalendar":` + calendarReferenceJSON + `}`
		}
		var filter Filter
		err := json.Unmarshal([]byte(root), &filter)
		if err != nil {
			t.Fatalf("expected temporal filter decoded, got %v", err)
		}
		before, err := json.Marshal(&filter)
		if err != nil {
			t.Fatalf("expected temporal filter encoded, got %v", err)
		}
		nodes := 0
		prepared, err := prepareUnifiedRule(&filter,
			types.Object([]types.Property{{Name: "country", Type: types.String()}}), 0, &nodes)
		if err != nil {
			t.Fatalf("expected temporal filter prepared, got %v", err)
		}
		period := prepared.Children[0].Experience.Verify.Children[0].OccurredIn
		if period == nil || calendar != (period.Calendar != nil) {
			t.Fatalf("expected inherited calendar=%v, got %+v", calendar, period)
		}
		after, err := json.Marshal(&filter)
		if err != nil {
			t.Fatalf("expected temporal filter reencoded, got %v", err)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("expected unchanged public period %s, got %s", before, after)
		}
	}
}

// TestFilterOccurredInExplanation retains leaf positions and exact bounds.
func TestFilterOccurredInExplanation(t *testing.T) {
	input := strings.Replace(experienceConditionJSON, experienceRecordConditionJSON,
		occurredInConditionJSON+`,{"operator":"or","rules":[`+occurredInConditionJSON+`]}`, 1)
	var condition FilterExperienceCondition
	err := json.Unmarshal([]byte(input), &condition)
	if err != nil {
		t.Fatalf("expected mixed verify decoded, got %v", err)
	}
	rule, err := prepareFilterExperience(&condition, nil)
	if err != nil {
		t.Fatalf("expected mixed verify prepared, got %v", err)
	}
	at := time.Date(2026, 3, 30, 10, 0, 0, 123456000, time.UTC)
	periods, err := explainOccurrencePeriods(rule.Experience.Verify, at, nil)
	if err != nil {
		t.Fatalf("expected resolved explanation periods, got %v", err)
	}
	encoded, err := json.Marshal(periods)
	if err != nil {
		t.Fatalf("expected explanation periods encoded, got %v", err)
	}
	if len(periods) != 2 || !strings.Contains(string(encoded), `"path":[0]`) ||
		!strings.Contains(string(encoded), `"path":[1,0]`) {
		t.Fatalf("expected distinct ordered leaf paths, got %s", encoded)
	}
	for _, period := range periods {
		if !period.Start.Equal(at.Add(-30*24*time.Hour)) || !period.End.Equal(at) {
			t.Fatalf("expected exact 30-day window ending at %s, got %+v", at, period)
		}
	}
}

// TestFilterOccurredInInvalidBeforeUnsupported checks both sibling orders.
func TestFilterOccurredInInvalidBeforeUnsupported(t *testing.T) {
	full := `{"occurredIn":{"kind":"fullRelevantHistory","history":` +
		`{"definition":"registered-queryable-events","meaningRevision":"1"}}}`
	for _, invalid := range []string{
		`{"property":"unknown","operator":"is","values":[1]}`,
		`{"property":"properties.x","operator":"bad","values":[1]}`,
		`{"occurredIn":{"kind":"elapsedLookback","duration":{"value":0,"unit":"day"}}}`,
		`{"occurredIn":{"kind":"lastCompleteCalendarPeriods","unit":"day","count":1}}`,
		strings.Replace(occurredInConditionJSON, `"kind":"elapsedLookback"`,
			`"history":{"definition":"registered-queryable-events","meaningRevision":"1"},"kind":"elapsedLookback"`, 1),
	} {
		for _, rules := range []string{full + "," + invalid, invalid + "," + full} {
			for _, separate := range []bool{false, true} {
				input := strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, rules, 1)
				if separate {
					input = strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, full, 1) + "," +
						strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, invalid, 1)
					if strings.HasPrefix(rules, invalid) {
						input = strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, invalid, 1) + "," +
							strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, full, 1)
					}
				}
				var filter Filter
				err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+input+`]}`), &filter)
				if err != nil {
					t.Fatalf("expected decoded validity fixture, got %v", err)
				}
				nodes := 0
				_, err = prepareUnifiedRule(&filter,
					types.Object([]types.Property{{Name: "country", Type: types.String()}}), 0, &nodes)
				if err != nil {
					if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
						t.Fatalf("expected invalidity before missing capability, got %v", err)
					}
					continue
				}
				t.Fatal("expected invalid filter, got success")
			}
		}
	}
	var condition FilterExperienceCondition
	err := json.Unmarshal([]byte(strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, full, 1)),
		&condition)
	if err != nil {
		t.Fatalf("expected full history decoded, got %v", err)
	}
	rule, err := prepareFilterExperience(&condition, nil)
	if err != nil {
		t.Fatalf("expected valid history prepared for semantic validation, got %v", err)
	}
	err = unified.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
	if err != nil {
		if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported {
			t.Fatalf("expected valid but unimplemented period, got %v", err)
		}
		return
	}
	t.Fatal("expected missing capability, got success")
}

// TestFilterOccurredInOwner rejects the leaf outside selected verification.
func TestFilterOccurredInOwner(t *testing.T) {
	for _, input := range []string{
		`{"operator":"and","rules":[` + occurredInConditionJSON + `]}`,
		strings.Replace(experienceConditionJSON, `"base":{`,
			`"base":{"where":{"operator":"and","rules":[`+occurredInConditionJSON+`]},`, 1),
		strings.Replace(countConditionJSON, `"count":{`,
			`"count":{"where":{"operator":"and","rules":[`+occurredInConditionJSON+`]},`, 1),
		strings.Replace(sumConditionJSON, `"sum":{`,
			`"sum":{"where":{"operator":"and","rules":[`+occurredInConditionJSON+`]},`, 1),
		strings.Replace(averageConditionJSON, `"average":{`,
			`"average":{"where":{"operator":"and","rules":[`+occurredInConditionJSON+`]},`, 1),
		strings.Replace(experienceConditionJSON, experienceRecordConditionJSON,
			`{"occurredIn":null}`, 1),
		strings.Replace(experienceConditionJSON, experienceRecordConditionJSON,
			strings.TrimSuffix(occurredInConditionJSON, "}")+`,"operator":"is","values":[1]}`, 1),
	} {
		if !strings.HasPrefix(input, `{"operator":`) {
			input = `{"operator":"and","rules":[` + input + `]}`
		}
		var filter Filter
		err := json.Unmarshal([]byte(input), &filter)
		if err == nil {
			t.Fatalf("expected invalid owner or leaf rejected, got %s", input)
		}
	}
	leaf := &FilterOccurredInCondition{OccurredIn: FilterPeriod{Kind: "elapsedLookback",
		Duration: &FilterDuration{Value: json.Value(`30`), Unit: "day"}}}
	filter := &Filter{Operator: OpAnd, Rules: []FilterRule{leaf}}
	_, _, err := prepareFilter(filter, types.Object([]types.Property{{Name: "country", Type: types.String()}}))
	if err == nil {
		t.Fatal("expected temporal leaf rejected in Record pipeline, got success")
	}
	nodes := 0
	_, err = prepareUnifiedRule(filter, types.Object([]types.Property{{Name: "country", Type: types.String()}}), 0, &nodes)
	if err == nil {
		t.Fatal("expected temporal leaf rejected on Profiles root, got success")
	}
}
