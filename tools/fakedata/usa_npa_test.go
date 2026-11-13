// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestUSANPACompletionInvalid rejects unsafe or unidentified NPA inputs.
func TestUSANPACompletionInvalid(t *testing.T) {

	encoded, err := os.ReadFile("testdata/published/" + usaReleaseDirectory + "npas.json")
	if err != nil {
		t.Fatalf("expected published NPA fixture, got %v", err)
	}
	cases := []struct {
		name   string
		change func(*usaNPADocument)
	}{
		{"source", func(d *usaNPADocument) { d.SourceSHA256 = strings.Repeat("0", 64) }},
		{"history", func(d *usaNPADocument) { d.HistoricalAsOf = "2026-10-16" }},
		{"status", func(d *usaNPADocument) { d.Status = "APPROVED" }},
		{"non-geographic", func(d *usaNPADocument) { d.Entries[0].NPA = "800" }},
		{"territory", func(d *usaNPADocument) { d.Entries[0].StateUSPS = "PR" }},
		{"future", func(d *usaNPADocument) { d.Entries[0].InServiceDate = "2026-01-02" }},
		{"invalid-date", func(d *usaNPADocument) { d.Entries[0].InServiceDate = "2025-02-30" }},
		{"missing", func(d *usaNPADocument) { d.Entries = d.Entries[:367] }},
		{"duplicate", func(d *usaNPADocument) { d.Entries[1] = d.Entries[0] }},
		{"wrong-state", func(d *usaNPADocument) { d.Entries[0].StateUSPS = "CA" }},
		{"wrong-date", func(d *usaNPADocument) { d.Entries[0].InServiceDate = "1948-01-01" }},
		{"679", func(d *usaNPADocument) { d.Entries[0].NPA = "679" }},
		{"bound", func(d *usaNPADocument) {
			for len(d.Entries) <= 400 {
				d.Entries = append(d.Entries, d.Entries[0])
			}
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var document usaNPADocument
			err := json.Unmarshal(encoded, &document)
			if err != nil {
				t.Fatalf("expected NPA fixture, got %v", err)
			}
			test.change(&document)
			input, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("expected mutated NPA document, got %v", err)
			}
			_, err = decodeUSANPACandidate(t.Context(), bytes.NewReader(input))
			if err != nil {
				return
			}
			t.Fatal("expected rejected NPA mutation, got nil")
		})
	}

}
