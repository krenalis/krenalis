// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package appwriter

import (
	"iter"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/connectors"
)

func newPeekTestIterator() *iterator {
	w := &Writer{
		connector: "test",
		available: 2,
		records: []record{
			{id: "first", attributes: map[string]any{"name": "First"}},
			{id: "second", attributes: map[string]any{"name": "Second"}},
		},
	}
	w.close.completed.L = &w.mu
	it := newIterator(w)
	w.iterator = it
	return it
}

// Test_iterator_Peek verifies Peek's behavior before, during, and after a
// record iteration.
func Test_iterator_Peek(t *testing.T) {

	tests := []struct {
		name string
		seq  func(*iterator) iter.Seq[connectors.Record]
	}{
		{name: "All", seq: (*iterator).All},
		{name: "Same", seq: (*iterator).Same},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			it := newPeekTestIterator()
			first, ok := it.Peek()
			if !ok || first.ID != "first" {
				t.Fatalf("Peek before iteration: expected record ID %q and true, got %q and %t", "first", first.ID, ok)
			}
			if got, ok := it.Peek(); !ok || got.ID != first.ID {
				t.Fatalf("repeated Peek before iteration: expected record ID %q and true, got %q and %t", first.ID, got.ID, ok)
			}

			want := []string{"first", "second"}
			yielded := 0
			for record := range test.seq(it) {
				if yielded >= len(want) {
					t.Fatalf("expected %d records, got more", len(want))
				}
				if record.ID != want[yielded] {
					t.Fatalf("record %d: expected ID %q, got %q", yielded, want[yielded], record.ID)
				}
				if yielded+1 < len(want) {
					if got, ok := it.Peek(); !ok || got.ID != want[yielded+1] {
						t.Fatalf("Peek during iteration: expected record ID %q and true, got %q and %t", want[yielded+1], got.ID, ok)
					}
				} else if got, ok := it.Peek(); ok || got.ID != "" || got.Attributes != nil || !got.UpdatedAt.IsZero() || got.Err != nil {
					t.Fatalf("Peek at the end of the iteration: expected a zero record and false, got %#v and %t", got, ok)
				}
				yielded++
			}
			if yielded != len(want) {
				t.Fatalf("expected %d records, got %d", len(want), yielded)
			}

			defer func() {
				r := recover()
				msg, ok := r.(string)
				if !ok || !strings.Contains(msg, "Records.Peek outside of an iteration") {
					t.Errorf("Peek after iteration: unexpected panic %v", r)
				}
			}()
			it.Peek()
		})
	}

}

// Test_iterator_PeekAfterFirst verifies that Peek panics after First is called.
func Test_iterator_PeekAfterFirst(t *testing.T) {
	it := newPeekTestIterator()
	first, ok := it.Peek()
	if !ok {
		t.Fatal("Peek before First: expected a record")
	}
	if got := it.First(); got.ID != first.ID {
		t.Fatalf("First: expected record ID %q, got %q", first.ID, got.ID)
	}

	defer func() {
		r := recover()
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "Records.Peek outside of an iteration") {
			t.Errorf("Peek after First: unexpected panic %v", r)
		}
	}()
	it.Peek()
}
