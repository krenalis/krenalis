// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const countConditionJSON = `{"count":{"source":{"kind":"recordedEvents","name":"Purchase",` +
	`"grain":"registered-message"},"period":{"kind":"elapsedLookback",` +
	`"duration":{"value":30,"unit":"day"}}},"operator":"is","values":[0]}`

// TestFilterCountDecoding enforces an exclusive rule shape and bounded Record
// bodies independently of member order, without selecting a backend.
func TestFilterCountDecoding(t *testing.T) {

	for _, test := range []struct{ name, condition string }{
		{"null", `{"count":null,"operator":"is","values":[0]}`},
		{"root discriminator", `{"count":{},"property":"country","operator":"is","values":[0]}`},
		{"group discriminator", `{"count":{},"operator":"and","rules":[],"values":[0]}`},
		{"missing operator", strings.Replace(countConditionJSON, `,"operator":"is"`, ``, 1)},
		{"missing values", strings.Replace(countConditionJSON, `,"values":[0]`, ``, 1)},
		{"unknown subject member", strings.Replace(countConditionJSON, `"source":`, `"unexpected":0,"source":`, 1)},
		{"unknown source member", strings.Replace(countConditionJSON, `"kind":"recordedEvents"`, `"kind":"recordedEvents","x":0`, 1)},
		{"unknown period member", strings.Replace(countConditionJSON, `"kind":"elapsedLookback"`, `"kind":"elapsedLookback","x":0`, 1)},
		{"unknown duration member", strings.Replace(countConditionJSON, `"unit":"day"`, `"unit":"day","x":0`, 1)},
		{"duplicate count", strings.Replace(countConditionJSON, `"count":`, `"count":{},"count":`, 1)},
		{"duplicate source", strings.Replace(countConditionJSON, `"source":`, `"source":{},"source":`, 1)},
		{"duplicate duration", strings.Replace(countConditionJSON, `"unit":"day"`, `"unit":"day","unit":"hour"`, 1)},
		{"null source", strings.Replace(countConditionJSON, `{"kind":"recordedEvents","name":"Purchase","grain":"registered-message"}`, `null`, 1)},
		{"null duration", strings.Replace(countConditionJSON, `{"value":30,"unit":"day"}`, `null`, 1)},
		{"missing source", `{"count":{"period":{"kind":"elapsedLookback"}},"operator":"is","values":[0]}`},
		{"missing period", `{"count":{"source":{}},"operator":"is","values":[0]}`},
		{"Count inside where", `{"count":{"source":{},"period":{},"where":{"operator":"and","rules":[` + countConditionJSON + `]}},"operator":"is","values":[0]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var filter Filter
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+test.condition+`]}`), &filter)
			if err == nil {
				t.Fatal("expected malformed Count document rejection, got success")
			}
		})
	}

	var root Filter
	err := json.Unmarshal([]byte(countConditionJSON), &root)
	if err == nil {
		t.Fatal("expected a root group requirement, got success")
	}

	var leaf FilterCountCondition
	err = json.Unmarshal([]byte(`{"values":[0],"operator":"is","count":{"period":`+
		`{"duration":{"unit":"day","value":30},"kind":"elapsedLookback"},`+
		`"source":{"grain":"registered-message","name":"Purchase","kind":"recordedEvents"}}}`), &leaf)
	if err != nil {
		t.Fatalf("expected field-order-independent Count decoding, got %v", err)
	}

}

// TestFilterCountEncoding keeps syntax distinct from semantic admission.
func TestFilterCountEncoding(t *testing.T) {

	for _, rule := range []FilterRule{(*FilterCountCondition)(nil), nil} {
		_, err := json.Marshal(&Filter{Operator: OpAnd, Rules: []FilterRule{rule}})
		if err == nil {
			t.Fatal("expected nil rule encoding rejection, got success")
		}
	}

	condition := countFilterCondition()
	condition.Operator = "future operator"
	condition.Values = []json.Value{json.Value(`9223372036854775808`)}
	data, err := json.Marshal(condition)
	if err != nil {
		t.Fatalf("expected syntax encoding without capability validation, got %v", err)
	}
	var decoded FilterCountCondition
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("expected syntax round-trip, got %v", err)
	}
	if decoded.Operator != condition.Operator || string(decoded.Values[0]) != string(condition.Values[0]) {
		t.Fatalf("expected exact operator and literal round-trip, got %+v", decoded)
	}

}

// TestFilterCountInstantOffsets preserves the admitted offsets and instants.
func TestFilterCountInstantOffsets(t *testing.T) {
	for _, test := range []struct {
		offset string
		shift  time.Duration
	}{
		{"Z", 0}, {"+01:00", time.Hour}, {"-00:00", 0},
	} {
		t.Run(test.offset, func(t *testing.T) {
			data := fmt.Sprintf(`{"kind":"absoluteHalfOpen",`+
				`"start":"2026-01-01T00:00:00.000001%s","end":"2026-01-02T00:00:00.000001%s"}`, test.offset, test.offset)
			var period FilterPeriod
			err := json.Unmarshal([]byte(data), &period)
			if err != nil {
				t.Fatalf("expected admitted offset %s, got %v", test.offset, err)
			}
			encoded, err := json.Marshal(&period)
			if err != nil {
				t.Fatalf("expected period encoding, got %v", err)
			}
			var restored FilterPeriod
			err = json.Unmarshal(encoded, &restored)
			if err != nil {
				t.Fatalf("expected restored period, got %v", err)
			}
			start := time.Date(2026, 1, 1, 0, 0, 0, 1000, time.UTC).Add(-test.shift)
			end := start.Add(24 * time.Hour)
			if !period.Start.Equal(start) || !period.End.Equal(end) ||
				!restored.Start.Equal(start) || !restored.End.Equal(end) {
				t.Fatalf("expected instants [%s,%s) before and after encoding, got %s and %s",
					start, end, data, encoded)
			}
		})
	}
}

// TestFilterCountInstants rejects excess precision before time.Time decoding.
func TestFilterCountInstants(t *testing.T) {
	for _, test := range []struct {
		fraction string
		nanos    int
		valid    bool
	}{
		{".1234567891", 0, false}, {".0000000001", 0, false}, {".0000010009", 0, false},
		{".0000000000000000000001", 0, false}, {".0000000000", 0, false},
		{".123456789", 123456789, true}, {".000001000", 1000, true}, {".000001", 1000, true},
	} {
		for _, bound := range []string{"start", "end"} {
			t.Run(bound+test.fraction, func(t *testing.T) {
				start, end := "2026-01-01T00:00:00Z", "2026-01-02T00:00:00Z"
				if bound == "start" {
					start = strings.TrimSuffix(start, "Z") + test.fraction + "Z"
				} else {
					end = strings.TrimSuffix(end, "Z") + test.fraction + "Z"
				}
				var period FilterPeriod
				err := json.Unmarshal([]byte(fmt.Sprintf(`{"kind":"absoluteHalfOpen","start":%q,"end":%q}`,
					start, end)), &period)
				if err != nil {
					if test.valid {
						t.Fatalf("expected representable timestamp, got %v", err)
					}
					return
				}
				if !test.valid {
					t.Fatal("expected excess fractional digits rejected before truncation, got success")
				}
				instant := period.Start
				if bound == "end" {
					instant = period.End
				}
				if instant.Nanosecond() != test.nanos {
					t.Fatalf("expected %d nanoseconds, got %d", test.nanos, instant.Nanosecond())
				}
				count := countFilterCondition()
				count.Count.Period = period
				rule, err := prepareFilterCount(count)
				if err != nil {
					t.Fatalf("expected exact Count preparation, got %v", err)
				}
				err = unified.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
				if err != nil {
					if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported || test.nanos%1000 == 0 {
						t.Fatalf("expected only submicrosecond precision unsupported, got %v", err)
					}
					return
				}
				if test.nanos%1000 != 0 {
					t.Fatal("expected unsupported submicrosecond precision, got success")
				}
			})
		}
	}
}

// TestFilterCountInvalidBeforeUnsupported checks the complete document even
// when another rule or another field exceeds an execution limit.
func TestFilterCountInvalidBeforeUnsupported(t *testing.T) {

	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	for _, test := range []struct {
		name   string
		change func(*FilterCountCondition)
	}{
		{"unit", func(c *FilterCountCondition) {
			c.Count.Period.Duration.Unit = "fortnight"
			c.Count.Period.Duration.Value = json.Value(`18446744073709551616`)
		}},
		{"between above int64", func(c *FilterCountCondition) { c.Operator = OpIsBetween; c.Values = filterValues(uint64(1<<63), 1) }},
		{"between above uint64", func(c *FilterCountCondition) {
			c.Operator = OpIsBetween
			c.Values = []json.Value{json.Value(`18446744073709551616`), json.Value(`1`)}
		}},
		{"invalid second operand", func(c *FilterCountCondition) {
			c.Operator = OpIsBetween
			c.Values = []json.Value{json.Value(`18446744073709551616`), json.Value(`-1`)}
		}},
		{"period", func(c *FilterCountCondition) { c.Count.Period.Kind = "calendarMonth" }},
		{"source", func(c *FilterCountCondition) { c.Count.Source.Kind = "SQL" }},
		{"event", func(c *FilterCountCondition) { c.Count.Source.Name = "" }},
		{"grain", func(c *FilterCountCondition) { c.Count.Source.Grain = "" }},
		{"unknown grain", func(c *FilterCountCondition) { c.Count.Source.Grain = "business-order" }},
		{"minimum", func(c *FilterCountCondition) {
			c.Values = filterValues(0)
			c.Count.Minimum = json.Value(`18446744073709551616`)
		}},
		{"where", func(c *FilterCountCondition) {
			c.Count.Where = &Filter{Operator: OpAnd, Rules: []FilterRule{
				&FilterCondition{Property: "absent", Operator: OpExists}}}
		}},
		{"where with unavailable period", func(c *FilterCountCondition) {
			c.Values = filterValues(0)
			c.Count.Period.Duration.Value = json.Value(`1000000000000000000000000000000`)
			c.Count.Where = &Filter{Operator: OpAnd, Rules: []FilterRule{
				&FilterCondition{Property: "absent", Operator: OpExists}}}
		}},
		{"deep where", func(c *FilterCountCondition) {
			c.Count.Where = &Filter{Operator: OpAnd, Rules: []FilterRule{
				&FilterCondition{Property: "event", Operator: OpIs, Values: filterValues("Purchase")}}}
			for range 2 {
				c.Count.Where = &Filter{Operator: OpAnd, Rules: []FilterRule{c.Count.Where}}
			}
		}},
		{"ago band ordering", func(c *FilterCountCondition) {
			c.Count.Period = FilterPeriod{Kind: "elapsedAgoBand",
				Older: &FilterDuration{Value: json.Value(`18446744073709551616`), Unit: "day"},
				Newer: &FilterDuration{Value: json.Value(`18446744073709551616`), Unit: "week"}}
		}},
		{"absolute ordering", func(c *FilterCountCondition) {
			end := time.Date(2026, 1, 1, 0, 0, 0, 1, time.UTC)
			c.Count.Period = FilterPeriod{Kind: "absoluteHalfOpen", Start: &end, End: &end}
		}},
		{"zero absolute start", func(c *FilterCountCondition) {
			start := time.Time{}
			end := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
			c.Count.Period = FilterPeriod{Kind: "absoluteHalfOpen", Start: &start, End: &end}
		}},
		{"absolute start outside UTC range", func(c *FilterCountCondition) {
			start := time.Date(1, 1, 1, 0, 30, 0, 0, time.FixedZone("", 3600))
			end := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
			c.Count.Period = FilterPeriod{Kind: "absoluteHalfOpen", Start: &start, End: &end}
		}},
	} {
		for _, reversed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reversed=%v", test.name, reversed), func(t *testing.T) {
				invalid := countFilterCondition()
				invalid.Values = []json.Value{json.Value(`18446744073709551616`)}
				test.change(invalid)
				unavailable := countFilterCondition()
				unavailable.Values = []json.Value{json.Value(`18446744073709551616`)}
				rules := []FilterRule{unavailable, &Filter{Operator: OpOr, Rules: []FilterRule{invalid}}}
				if reversed {
					rules[0], rules[1] = rules[1], rules[0]
				}
				nodes := 0
				rule, err := prepareUnifiedRule(&Filter{Operator: OpAnd, Rules: rules}, schema, 0, &nodes)
				if err == nil {
					err = unified.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
				}
				if err != nil {
					if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
						t.Fatalf("expected invalidity before unavailable capability, got %v", err)
					}
					return
				}
				t.Fatal("expected invalid Count document, got success")
			})
		}
	}

	for _, reversed := range []bool{false, true} {
		unavailable := countFilterCondition()
		unavailable.Values = []json.Value{json.Value(`18446744073709551616`)}
		rules := []FilterRule{unavailable, &FilterCondition{Property: "absent", Operator: OpExists}}
		if reversed {
			rules[0], rules[1] = rules[1], rules[0]
		}
		nodes := 0
		_, err := prepareUnifiedRule(&Filter{Operator: OpAnd, Rules: rules}, schema, 0, &nodes)
		if err != nil {
			if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
				t.Fatalf("expected invalid Record sibling in either order, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid Record sibling rejection, got success")
	}

}

// TestFilterCountIntegerArithmetic compares linear decimal operations with an
// independent big-integer oracle, including operands much larger than int64.
func TestFilterCountIntegerArithmetic(t *testing.T) {

	random := rand.New(rand.NewPCG(41, 89))
	for _, digits := range []int{1, 2, 18, 19, 20, 31, 100, 1000, 65536} {
		text := make([]byte, digits)
		for i := range text {
			text[i] = byte(random.IntN(10)) + '0'
		}
		text[0] = '9'
		n, err := parseFilterCountInteger(json.Value(text))
		if err != nil {
			t.Fatalf("expected exact natural, got %v", err)
		}
		oracle, ok := new(big.Int).SetString(string(text), 10)
		if !ok {
			t.Fatal("expected independent integer oracle, got invalid fixture")
		}
		for _, unit := range []uint64{
			1, 1000, 1000000, 1000000000, 60000000000, 3600000000000, 86400000000000, 604800000000000,
		} {
			want := new(big.Int).Mul(oracle, new(big.Int).SetUint64(unit)).String()
			if got := n.scaled(unit); string(got) != want {
				t.Fatalf("expected exact product for %d digits and unit %d, got unequal value", digits, unit)
			}
		}
		for _, other := range []string{
			"0", "1", "9223372036854775807", "18446744073709551616", string(text), string(text) + "0",
		} {
			otherOracle, _ := new(big.Int).SetString(other, 10)
			if got, want := n.compare(filterCountInteger(other)), oracle.Cmp(otherOracle); got != want {
				t.Fatalf("expected comparison %d for %d digits, got %d", want, digits, got)
			}
		}
	}

	// The size of a valid literal is not itself a semantic invalidity. Retain
	// validation precedence over unsupported execution ranges.
	for _, invalid := range []bool{false, true} {
		c := countFilterCondition()
		c.Operator = OpIsBetween
		large := strings.Repeat("9", 100000)
		c.Values = []json.Value{json.Value(large), json.Value(large)}
		if invalid {
			c.Values[1] = json.Value("1")
		}
		_, err := prepareFilterCount(c)
		if err != nil {
			_, unsupported := errors.AsType[*unified.UnsupportedError](err)
			if unsupported == invalid {
				t.Fatalf("expected invalid=%t before execution range, got %v", invalid, err)
			}
		}
		if err == nil {
			t.Fatal("expected range or validation error, got success")
		}
	}

}

// TestFilterCountLimits keeps main and occurrence-body budgets independent.
func TestFilterCountLimits(t *testing.T) {
	for _, test := range []struct {
		main, body, nodes int
		valid             bool
	}{
		{4, 2, 99, true}, {4, 1, 100, true}, {5, 2, 99, false}, {4, 3, 1, false}, {1, 1, 101, false},
	} {
		t.Run(fmt.Sprintf("%d/%d/%d", test.main, test.body, test.nodes), func(t *testing.T) {
			count := countFilterCondition()
			body := &Filter{Operator: OpAnd, Rules: make([]FilterRule, test.nodes)}
			for i := range body.Rules {
				body.Rules[i] = &FilterCondition{Property: "properties.currency",
					Operator: OpIs, Values: filterValues("EUR")}
			}
			for level := 1; level < test.body; level++ {
				body = &Filter{Operator: OpAnd, Rules: []FilterRule{body}}
			}
			count.Count.Where = body
			root := &Filter{Operator: OpAnd, Rules: []FilterRule{count}}
			for level := 1; level < test.main; level++ {
				root = &Filter{Operator: OpAnd, Rules: []FilterRule{root}}
			}
			data, err := json.Marshal(root)
			if err != nil {
				t.Fatalf("expected syntax encoding, got %v", err)
			}
			var restored Filter
			err = json.Unmarshal(data, &restored)
			if err != nil {
				if test.valid {
					t.Fatalf("expected independent main/body budgets, got %v", err)
				}
				return
			}
			if !test.valid {
				t.Fatal("expected exceeded budget rejection, got success")
			}
		})
	}
}

// TestFilterCountNulls distinguishes missing configuration from JSON literals.
func TestFilterCountNulls(t *testing.T) {

	type testCase struct {
		name, condition string
		valid           bool
	}
	cases := []testCase{
		{"optional fields omitted", countConditionJSON, true},
		{"where null", strings.Replace(countConditionJSON, `"source":`, `"where":null,"source":`, 1), true},
		{"minimum null", strings.Replace(countConditionJSON, `"source":`, `"minimum":null,"source":`, 1), false},
		{"Count null literal", strings.Replace(countConditionJSON, `"values":[0]`, `"values":[null]`, 1), false},
		{"foreign temporal null", strings.Replace(countConditionJSON, `"kind":"elapsedLookback"`, `"kind":"elapsedLookback","older":null`, 1), false},
		{"Record membership null literal", strings.Replace(countConditionJSON, `"source":`, `"where":{"operator":"and","rules":[{"property":"properties.currency","operator":"is one of","values":[null]}]},"source":`, 1), true},
		{"Record comparison null literal", strings.Replace(countConditionJSON, `"source":`, `"where":{"operator":"and","rules":[{"property":"properties.currency","operator":"is","values":[null]}]},"source":`, 1), false},
	}
	for _, path := range [][]string{
		{"count"}, {"operator"}, {"values"}, {"count", "source"}, {"count", "period"},
		{"count", "source", "kind"}, {"count", "source", "name"}, {"count", "source", "grain"},
		{"count", "period", "kind"}, {"count", "period", "duration"},
		{"count", "period", "duration", "value"}, {"count", "period", "duration", "unit"},
	} {
		value, ok := json.Value(countConditionJSON).Get(path)
		if !ok {
			t.Fatalf("expected required member %v in the fixture, got none", path)
		}
		key := `"` + path[len(path)-1] + `":`
		member := key + string(value)
		omitted := strings.Replace(countConditionJSON, member+",", "", 1)
		if omitted == countConditionJSON {
			omitted = strings.Replace(countConditionJSON, ","+member, "", 1)
		}
		cases = append(cases,
			testCase{strings.Join(path, ".") + " omitted", omitted, false},
			testCase{strings.Join(path, ".") + " null", strings.Replace(countConditionJSON, member, key+"null", 1), false})
	}

	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if !json.Valid([]byte(test.condition)) {
				t.Fatalf("expected valid fixture JSON, got %s", test.condition)
			}
			var filter Filter
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+test.condition+`]}`), &filter)
			var prepared unified.Rule
			if err == nil {
				nodes := 0
				prepared, err = prepareUnifiedRule(&filter, schema, 0, &nodes)
			}
			if err == nil {
				err = unified.Validate(&prepared, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
			}
			if err != nil {
				if test.valid {
					t.Fatalf("expected admitted null/omission semantics, got %v", err)
				}
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
					t.Fatalf("expected invalid configuration, got %v", err)
				}
				return
			}
			if !test.valid {
				t.Fatal("expected invalid null or missing member, got success")
			}
			if prepared.Children[0].Count.Minimum != 0 {
				t.Fatalf("expected no extra sample minimum, got %d", prepared.Children[0].Count.Minimum)
			}
			encoded, err := json.Marshal(&filter)
			if err != nil {
				t.Fatalf("expected admitted filter encoding, got %v", err)
			}
			count := filter.Rules[0].(*FilterCountCondition)
			if count.Count.Where == nil {
				if strings.Contains(string(encoded), `"where"`) {
					t.Fatalf("expected omitted unfiltered where, got %s", encoded)
				}
			}
			if strings.Contains(test.condition, `"values":[null]`) &&
				!strings.Contains(string(encoded), `"values":[null]`) {
				t.Fatalf("expected preserved Record JSON null literal, got %s", encoded)
			}
		})
	}

}

// TestFilterCountPeriodRanges checks both sides of each duration boundary.
func TestFilterCountPeriodRanges(t *testing.T) {
	for _, test := range []struct {
		unit  string
		value string
		valid bool
	}{
		{"day", "106751", true}, {"day", "106752", false},
		{"week", "15250", true}, {"week", "15251", false},
		{"microsecond", "9223372036854775", true}, {"microsecond", "9223372036854776", false},
	} {
		t.Run(test.unit+"/"+test.value, func(t *testing.T) {
			count := countFilterCondition()
			count.Count.Period.Duration = &FilterDuration{Value: json.Value(test.value), Unit: test.unit}
			rule, err := prepareFilterCount(count)
			if err == nil {
				err = unified.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
			}
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported || test.valid {
					t.Fatalf("expected valid=%v at duration boundary, got %v", test.valid, err)
				}
				return
			}
			if !test.valid {
				t.Fatal("expected duration range limit, got success")
			}
		})
	}
}

// TestFilterCountPeriods checks exact fixed units, conflicting forms and
// unavailable duration ranges without truncation or Float conversion.
func TestFilterCountPeriods(t *testing.T) {
	for _, test := range []struct {
		unit string
		want time.Duration
	}{
		{"microsecond", time.Microsecond}, {"millisecond", time.Millisecond}, {"second", time.Second},
		{"minute", time.Minute}, {"hour", time.Hour}, {"day", 24 * time.Hour}, {"week", 7 * 24 * time.Hour},
	} {
		condition := countFilterCondition()
		condition.Count.Period.Duration = &FilterDuration{Value: json.Value(`1`), Unit: test.unit}
		rule, err := prepareFilterCount(condition)
		if err != nil {
			t.Fatalf("expected fixed unit %s, got %v", test.unit, err)
		}
		if rule.Count.Scope.Elapsed.Older != test.want {
			t.Fatalf("expected elapsed duration %s, got %s", test.want, rule.Count.Scope.Elapsed.Older)
		}
	}
	for _, test := range []struct {
		name        string
		change      func(*FilterCountCondition)
		unsupported bool
	}{
		{"zero duration", func(c *FilterCountCondition) { c.Count.Period.Duration.Value = json.Value(`0`) }, false},
		{"negative duration", func(c *FilterCountCondition) { c.Count.Period.Duration.Value = json.Value(`-1`) }, false},
		{"months", func(c *FilterCountCondition) { c.Count.Period.Duration.Unit = "month" }, false},
		{"years", func(c *FilterCountCondition) { c.Count.Period.Duration.Unit = "year" }, false},
		{"duration overflow", func(c *FilterCountCondition) { c.Count.Period.Duration.Value = json.Value(`106752`) }, true},
		{"missing grain", func(c *FilterCountCondition) { c.Count.Source.Grain = "" }, false},
		{"other grain", func(c *FilterCountCondition) { c.Count.Source.Grain = "business-order" }, false},
		{"unknown source", func(c *FilterCountCondition) { c.Count.Source.Kind = "SQL" }, false},
		{"missing instant", func(c *FilterCountCondition) { c.Count.Period = FilterPeriod{Kind: "absoluteHalfOpen"} }, false},
		{"conflicting fields", func(c *FilterCountCondition) { c.Count.Period.Older = &FilterDuration{} }, false},
		{"empty band", func(c *FilterCountCondition) {
			c.Count.Period = FilterPeriod{Kind: "elapsedAgoBand",
				Older: &FilterDuration{Value: json.Value(`1`), Unit: "day"},
				Newer: &FilterDuration{Value: json.Value(`24`), Unit: "hour"}}
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			condition := countFilterCondition()
			test.change(condition)
			rule, err := prepareFilterCount(condition)
			if err == nil {
				err = unified.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
			}
			if err != nil {
				_, unsupported := errors.AsType[*unified.UnsupportedError](err)
				if unsupported != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			t.Fatal("expected rejected period, got success")
		})
	}
}

// TestFilterCountRoundTrip preserves mixed groups, operands and relative time.
func TestFilterCountRoundTrip(t *testing.T) {

	data := `{"operator":"and","rules":[{"property":"country","operator":"is","values":["IT"]},` +
		countConditionJSON + `]}`
	input := []byte(data)
	var filter Filter
	err := json.Unmarshal(input, &filter)
	if err != nil {
		t.Fatalf("expected mixed filter decoding, got %v", err)
	}
	clear(input)
	encoded, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected mixed filter encoding, got %v", err)
	}
	if string(encoded) != data {
		t.Fatalf("expected exact public round-trip %s, got %s", data, encoded)
	}

	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	nodes := 0
	rule, err := prepareUnifiedRule(&filter, schema, 0, &nodes)
	if err != nil {
		t.Fatalf("expected mixed preparation, got %v", err)
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	err = unified.Validate(&rule, at, unified.Page{Limit: 1})
	if err != nil {
		t.Fatalf("expected admitted mixed filter, got %v", err)
	}
	if len(rule.Children) != 2 || rule.Children[1].Count.Scope.Elapsed.Older != 30*24*time.Hour ||
		!rule.Children[1].Count.Scope.Start.IsZero() || nodes != 3 {
		t.Fatalf("expected a Record and unresolved relative Count, got %+v", rule)
	}

	after, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected unchanged input encoding, got %v", err)
	}
	if !bytes.Equal(encoded, after) {
		t.Fatalf("expected unchanged input %s, got %s", encoded, after)
	}

	for _, recordSchema := range []types.Type{schema, schemas.Event} {
		_, _, err = prepareFilter(&Filter{Operator: OpAnd, Rules: []FilterRule{filter.Rules[1]}}, recordSchema)
		if err == nil {
			t.Fatal("expected Count rejection in Record filtering, got success")
		}
	}

}

// TestPrepareFilterCount checks the closed comparisons and exact integer input.
func TestPrepareFilterCount(t *testing.T) {

	for _, op := range operators {
		condition := countFilterCondition()
		condition.Operator = op
		if op == OpIsBetween {
			condition.Values = filterValues(0, 2)
		}
		_, err := prepareFilterCount(condition)
		allowed := op == OpIs || op == OpIsNot || op == OpIsLessThan || op == OpIsLessThanOrEqualTo ||
			op == OpIsGreaterThan || op == OpIsGreaterThanOrEqualTo || op == OpIsBetween
		if err != nil {
			if allowed {
				t.Fatalf("expected Count operator %q, got %v", op, err)
			}
			continue
		}
		if !allowed {
			t.Fatalf("expected Count operator %q rejected, got success", op)
		}
	}

	for _, test := range []struct {
		value       string
		unsupported bool
	}{
		{`-1`, false}, {`1.0`, false}, {`1e1`, false}, {`"1"`, false}, {`null`, false}, {`true`, false},
		{`9223372036854775808`, true}, {`18446744073709551616`, true},
	} {
		t.Run(test.value, func(t *testing.T) {
			condition := countFilterCondition()
			condition.Values = []json.Value{json.Value(test.value)}
			rule, err := prepareFilterCount(condition)
			if err == nil {
				err = unified.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
			}
			if err != nil {
				_, unsupported := errors.AsType[*unified.UnsupportedError](err)
				if unsupported != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			t.Fatal("expected rejected Count operand, got success")
		})
	}

	for _, literal := range []string{`-0`, `9007199254740993`, `9223372036854775807`} {
		condition := countFilterCondition()
		condition.Values = []json.Value{json.Value(literal)}
		rule, err := prepareFilterCount(condition)
		if err != nil {
			t.Fatalf("expected exact Count literal %s, got %v", literal, err)
		}
		if rule.Count.Test.Value > 0 && fmt.Sprint(rule.Count.Test.Value) != literal {
			t.Fatalf("expected exact integer %s, got %d", literal, rule.Count.Test.Value)
		}
	}

	for _, change := range []func(*FilterCountCondition){
		func(c *FilterCountCondition) { c.Values = nil },
		func(c *FilterCountCondition) { c.Values = filterValues(0, 1) },
		func(c *FilterCountCondition) { c.Operator = OpIsBetween },
		func(c *FilterCountCondition) { c.Count.Minimum = json.Value(`0`) },
		func(c *FilterCountCondition) { c.Count.Source.Grain = "" },
		func(c *FilterCountCondition) { c.Count.Period.Duration.Unit = "month" },
		func(c *FilterCountCondition) { c.Count.Period.Older = &FilterDuration{} },
	} {
		condition := countFilterCondition()
		change(condition)
		_, err := prepareFilterCount(condition)
		if err == nil {
			t.Fatal("expected invalid Count rejection, got success")
		}
	}

}

func countFilterCondition() *FilterCountCondition {
	return &FilterCountCondition{Count: FilterCount{
		Source: FilterCountSource{Kind: "recordedEvents", Name: "Purchase", Grain: "registered-message"},
		Period: FilterPeriod{Kind: "elapsedLookback", Duration: &FilterDuration{Value: json.Value(`31`), Unit: "day"}},
	}, Operator: OpIs, Values: filterValues(0)}
}
