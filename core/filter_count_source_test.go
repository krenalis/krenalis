// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestFilterCountSourceDecoding rejects explicit null and empty selectors.
func TestFilterCountSourceDecoding(t *testing.T) {
	for _, member := range []string{
		`"name":null`, `"name":""`, `"name":42`, `"name":{}`, `"name":[]`,
		`"type":null`, `"type":""`, `"type":42`, `"type":false`, `"type":[]`,
		`"type":"page","type":"screen"`, `"extra":true`,
	} {
		t.Run(member, func(t *testing.T) {
			var source FilterCountSource
			err := json.Unmarshal([]byte(`{"kind":"recordedEvents","grain":"registered-message",`+member+`}`), &source)
			if err == nil {
				t.Fatal("expected malformed source rejected, got success")
			}
		})
	}
}

// TestFilterCountSourceInvalidBeforeUnsupported protects all source consumers.
func TestFilterCountSourceInvalidBeforeUnsupported(t *testing.T) {
	for _, typ := range []string{"page", "screen", "identify"} {
		for _, kind := range []string{"currency", "sum"} {
			for _, reversed := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/reversed=%v", typ, kind, reversed), func(t *testing.T) {
					invalid := currencyCountConditionJSON
					if kind == "sum" {
						invalid = sumConditionJSON
					}
					invalid = strings.Replace(invalid, `"kind":"recordedEvents"`,
						`"kind":"recordedEvents","type":"`+typ+`"`, 1)
					unavailable := strings.Replace(countConditionJSON,
						`"values":[0]`, `"values":[18446744073709551616]`, 1)
					first, second := invalid, unavailable
					if reversed {
						first, second = second, first
					}
					var filter Filter
					err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+first+`,`+second+`]}`), &filter)
					if err != nil {
						t.Fatalf("expected structurally valid filter, got %v", err)
					}
					nodes := 0
					schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
					_, err = prepareBehavioralRule(&filter, schema, 0, &nodes)
					if err != nil {
						if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); unsupported {
							t.Fatalf("expected invalid non-track monetary source before capability, got %v", err)
						}
						return
					}
					t.Fatal("expected invalid non-track monetary source, got success")
				})
			}
		}
	}
}

// TestFilterCountSources checks strict source fields and their resolved types.
func TestFilterCountSources(t *testing.T) {
	for _, test := range []struct {
		name, source, typ, event string
		invalid                  bool
	}{
		{"legacy track", `{"kind":"recordedEvents","name":"Purchase","grain":"registered-message"}`, "track", "Purchase", false},
		{"explicit track", `{"kind":"recordedEvents","name":"Purchase","grain":"registered-message","type":"track"}`, "track", "Purchase", false},
		{"named page", `{"kind":"recordedEvents","name":"Pricing","grain":"registered-message","type":"page"}`, "page", "Pricing", false},
		{"all page", `{"kind":"recordedEvents","grain":"registered-message","type":"page"}`, "page", "", false},
		{"named screen", `{"kind":"recordedEvents","name":"Home","grain":"registered-message","type":"screen"}`, "screen", "Home", false},
		{"all screen", `{"kind":"recordedEvents","grain":"registered-message","type":"screen"}`, "screen", "", false},
		{"track without name", `{"kind":"recordedEvents","grain":"registered-message","type":"track"}`, "", "", true},
		{"implicit track without name", `{"kind":"recordedEvents","grain":"registered-message"}`, "", "", true},
		{"unsupported type", `{"kind":"recordedEvents","name":"Pricing","grain":"registered-message","type":"identify"}`, "", "", true},
		{"type case", `{"kind":"recordedEvents","name":"Pricing","grain":"registered-message","type":"Page"}`, "", "", true},
		{"name NUL", `{"kind":"recordedEvents","name":"\u0000","grain":"registered-message","type":"page"}`, "", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := `{"count":{"source":` + test.source +
				`,"period":{"kind":"elapsedLookback","duration":{"value":7,"unit":"day"}}},` +
				`"operator":"is","values":[0]}`
			var condition FilterCountCondition
			err := json.Unmarshal([]byte(input), &condition)
			if err != nil {
				t.Fatalf("expected source syntax decoded, got %v", err)
			}
			before, err := json.Marshal(&condition)
			if err != nil {
				t.Fatalf("expected source encoding, got %v", err)
			}
			if string(before) != input {
				t.Fatalf("expected unchanged source document %s, got %s", input, before)
			}
			prepared, err := prepareFilterCount(&condition, nil)
			if err != nil {
				if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); unsupported || !test.invalid {
					t.Fatalf("expected invalid=%v, got %v", test.invalid, err)
				}
				return
			}
			if test.invalid {
				t.Fatal("expected invalid source rejection, got success")
			}
			if prepared.Count.Scope.Type != test.typ || prepared.Count.Scope.Event != test.event {
				t.Fatalf("expected resolved %s/%s, got %+v", test.typ, test.event, prepared.Count.Scope)
			}
			after, err := json.Marshal(&condition)
			if err != nil {
				t.Fatalf("expected source encoding after preparation, got %v", err)
			}
			if !bytes.Equal(before, after) {
				t.Fatalf("expected unchanged public source %s, got %s", before, after)
			}
		})
	}
}
