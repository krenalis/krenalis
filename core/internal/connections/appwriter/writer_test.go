// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package appwriter

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math"
	"math/rand/v2"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/types"
)

func Test_Writer(t *testing.T) {

	tests := []struct {
		num    int     // number of records to process
		seed   uint64  // seed value to pseudo-randomize the Upsert method
		create float32 // percentage of records to create, in the range [0,1]
	}{
		{num: 0, seed: 0, create: 1},
		{num: 1, seed: 25, create: 1},
		{num: 1, seed: 92, create: 0},
		{num: minBatchSize / 2, seed: 63, create: 0.16},
		{num: minBatchSize * 3, seed: 47, create: 0.75},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%d/%d/%f", test.num, test.seed, test.create), func(t *testing.T) {

			app := newApplication(t, test.seed)
			w := New("test", state.TargetUser, app.Upsert, types.Type{}, app.ack)

			ctx := context.Background()

			ids := map[string]int{}
			for i := range test.num {
				var id string
				mod := 1
				if test.create > 0 {
					mod = int(math.Ceil(1 / float64(test.create)))
				}
				if i%mod != 0 {
					id = strconv.Itoa(i)
				}
				ids[id]++
				attributes := map[string]any{
					"id": id,
				}
				if !w.Write(ctx, id, attributes) {
					t.Fatal("Write: expected true, got false")
				}
			}

			var n int
			for {
				time.Sleep(10 * time.Millisecond)
				app.mu.Lock()
				n = app.n
				app.mu.Unlock()
				if n == test.num {
					break
				}
			}
			if n != test.num {
				t.Fatalf("expected %d IDs, got %d", test.num, n)
			}

			app.mu.Lock()
			defer app.mu.Unlock()

			for i, ack := range app.acks {
				for _, id := range ack.ids {
					ids[id]--
					if id != "" && ids[id] < 0 {
						t.Fatalf("ack %d/%d: ID %q has already been received", i+1, test.num, id)
					}
				}
			}
			if ids[""] != 0 {
				t.Fatalf("missing %d created", ids[""])
			}

			if err := w.Close(ctx); err != nil {
				t.Fatalf("Close: expected no error, got error %q", err)
			}

		})
	}

}

func TestWriterReceivesCreateAndUpdateIDs(t *testing.T) {

	tests := []struct {
		name  string
		id    string
		value string
	}{
		{name: "new without transformation", value: "new without transformation"},
		{name: "existing without transformation", id: "destination-1", value: "existing without transformation"},
		{name: "new with transformation", value: "new with transformation"},
		{name: "existing with transformation", id: "destination-2", value: "existing with transformation"},
	}

	var got []connectors.Record
	w := New("test", state.TargetUser, func(_ context.Context, _ connectors.Targets, records connectors.Records, _ types.Type) error {
		for record := range records.All() {
			got = append(got, record)
		}
		return nil
	}, types.Type{}, func(_ []string, _ error) {})
	for i, test := range tests {
		attributes := map[string]any{"value": test.value}
		if !w.Write(context.Background(), test.id, attributes) {
			t.Fatalf("case %d: expected Write to succeed, got false", i)
		}
	}
	if err := w.Close(context.Background()); err != nil {
		t.Fatalf("expected Close to succeed, got %v", err)
	}
	if len(got) != len(tests) {
		t.Fatalf("expected %d records, got %d", len(tests), len(got))
	}
	for i, test := range tests {
		if got[i].ID != test.id {
			t.Errorf("case %d (%s): expected ID %q, got %q", i, test.name, test.id, got[i].ID)
		}
		if got[i].Attributes["value"] != test.value {
			t.Errorf("case %d (%s): expected value %q, got %q", i, test.name, test.value, got[i].Attributes["value"])
		}
	}

}

func TestWriterUpsertSuccess(t *testing.T) {

	var gotIDs []string
	var gotAcks []ack
	w := New("test", state.TargetUser, func(_ context.Context, _ connectors.Targets, records connectors.Records, _ types.Type) error {
		for record := range records.All() {
			gotIDs = append(gotIDs, record.ID)
		}
		return nil
	}, types.Type{}, func(ids []string, err error) {
		gotAcks = append(gotAcks, ack{ids: ids, err: err})
	})
	if !w.Write(t.Context(), "first", map[string]any{"value": 1}) {
		t.Fatal("expected Write to succeed, got false")
	}
	if !w.Write(t.Context(), "second", map[string]any{"value": 2}) {
		t.Fatal("expected Write to succeed, got false")
	}
	if err := w.Close(t.Context()); err != nil {
		t.Fatalf("expected Close to succeed, got %v", err)
	}
	if len(gotIDs) != 2 {
		t.Fatalf("expected 2 consumed records, got %d", len(gotIDs))
	}
	if gotIDs[0] != "first" || gotIDs[1] != "second" {
		t.Fatalf("expected consumed IDs [first second], got %v", gotIDs)
	}
	if len(gotAcks) != 1 {
		t.Fatalf("expected 1 acknowledgement, got %d", len(gotAcks))
	}
	if gotAcks[0].err != nil {
		t.Fatalf("expected successful acknowledgement, got %v", gotAcks[0].err)
	}

}

func TestWriterUpsertRecordsError(t *testing.T) {
	firstErr := errors.New("first record failed")
	thirdErr := errors.New("third record failed")
	var gotAcks []ack
	w := New("test", state.TargetUser, func(_ context.Context, _ connectors.Targets, records connectors.Records, _ types.Type) error {
		for range records.All() {
		}
		return connectors.RecordsError{0: firstErr, 2: thirdErr}
	}, types.Type{}, func(ids []string, err error) {
		gotAcks = append(gotAcks, ack{ids: ids, err: err})
	})
	for i := range 3 {
		if !w.Write(t.Context(), fmt.Sprintf("record-%d", i), map[string]any{"value": i}) {
			t.Fatalf("record %d: expected Write to succeed, got false", i)
		}
	}
	if err := w.Close(t.Context()); err != nil {
		t.Fatalf("expected Close to succeed, got %v", err)
	}
	if len(gotAcks) != 3 {
		t.Fatalf("expected 3 acknowledgements, got %d", len(gotAcks))
	}
	got := map[string][]string{}
	for _, gotAck := range gotAcks {
		key := "success"
		if gotAck.err != nil {
			key = gotAck.err.Error()
		}
		got[key] = append(got[key], gotAck.ids...)
	}
	if len(got[firstErr.Error()]) != 1 || got[firstErr.Error()][0] != "record-0" {
		t.Fatalf("expected first error for [record-0], got %v", got[firstErr.Error()])
	}
	if len(got[thirdErr.Error()]) != 1 || got[thirdErr.Error()][0] != "record-2" {
		t.Fatalf("expected third error for [record-2], got %v", got[thirdErr.Error()])
	}
	if len(got["success"]) != 1 || got["success"][0] != "record-1" {
		t.Fatalf("expected successful record [record-1], got %v", got["success"])
	}
}

func TestWriterUpsertGeneralErrorAfterFullConsumption(t *testing.T) {
	upsertErr := errors.New("upsert failed after full consumption")
	var gotAcks []ack
	w := New("test", state.TargetUser, func(_ context.Context, _ connectors.Targets, records connectors.Records, _ types.Type) error {
		for range records.All() {
		}
		return upsertErr
	}, types.Type{}, func(ids []string, err error) {
		gotAcks = append(gotAcks, ack{ids: ids, err: err})
	})
	for i := range 2 {
		if !w.Write(t.Context(), fmt.Sprintf("record-%d", i), map[string]any{"value": i}) {
			t.Fatalf("record %d: expected Write to succeed, got false", i)
		}
	}
	err := w.Close(t.Context())
	if !errors.Is(err, upsertErr) {
		t.Fatalf("expected Close error %v, got %v", upsertErr, err)
	}
	if len(gotAcks) != 1 || !errors.Is(gotAcks[0].err, upsertErr) {
		t.Fatalf("expected one failed acknowledgement, got %v", gotAcks)
	}
}

func TestWriterUpsertGeneralErrorAfterPartialConsumption(t *testing.T) {
	upsertErr := errors.New("upsert failed after partial consumption")
	var calls int
	var consumed []string
	var gotAcks []ack
	w := New("test", state.TargetUser, func(_ context.Context, _ connectors.Targets, records connectors.Records, _ types.Type) error {
		calls++
		count := 0
		for record := range records.All() {
			consumed = append(consumed, record.ID)
			count++
			if count == 2 {
				break
			}
		}
		return upsertErr
	}, types.Type{}, func(ids []string, err error) {
		gotAcks = append(gotAcks, ack{ids: ids, err: err})
	})
	for i := range 4 {
		if !w.Write(t.Context(), fmt.Sprintf("record-%d", i), map[string]any{"value": i}) {
			t.Fatalf("record %d: expected Write to succeed, got false", i)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	err := w.Close(ctx)
	if !errors.Is(err, upsertErr) {
		t.Fatalf("expected Close error %v, got %v", upsertErr, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 Upsert call, got %d", calls)
	}
	if len(consumed) != 2 || consumed[0] != "record-0" || consumed[1] != "record-1" {
		t.Fatalf("expected consumed IDs [record-0 record-1], got %v", consumed)
	}
	if len(gotAcks) != 1 || !errors.Is(gotAcks[0].err, upsertErr) || len(gotAcks[0].ids) != 4 {
		t.Fatalf("expected all 4 records to be acknowledged with the Upsert error, got %v", gotAcks)
	}
}

func TestWriterUpsertGeneralErrorBeforeConsumption(t *testing.T) {
	upsertErr := errors.New("upsert failed before consumption")
	var calls int
	var gotAcks []ack
	w := New("test", state.TargetUser, func(_ context.Context, _ connectors.Targets, _ connectors.Records, _ types.Type) error {
		calls++
		return upsertErr
	}, types.Type{}, func(ids []string, err error) {
		gotAcks = append(gotAcks, ack{ids: ids, err: err})
	})
	for i := range 2 {
		if !w.Write(t.Context(), fmt.Sprintf("record-%d", i), map[string]any{"value": i}) {
			t.Fatalf("record %d: expected Write to succeed, got false", i)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	err := w.Close(ctx)
	if !errors.Is(err, upsertErr) {
		t.Fatalf("expected Close error %v, got %v", upsertErr, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 Upsert call, got %d", calls)
	}
	if len(gotAcks) != 1 || !errors.Is(gotAcks[0].err, upsertErr) || len(gotAcks[0].ids) != 2 {
		t.Fatalf("expected both records to be acknowledged with the Upsert error, got %v", gotAcks)
	}
}

type ack struct {
	ids []string
	err error
}

type application struct {
	t    *testing.T
	rng  *rand.Rand // protected by mu
	mu   sync.Mutex
	n    int
	acks []ack
}

func newApplication(t *testing.T, seed uint64) *application {
	return &application{t: t, rng: rand.New(rand.NewPCG(seed, ^seed))}
}
func (app *application) validateRecord(r connectors.Record) {
	if r.Attributes == nil {
		app.t.Fatal("Upsert: expected attributes, got nil")
	}
	if r.Attributes["id"] != r.ID {
		app.t.Fatalf("Upsert: expected attributes[\"id\"] == %q, got %q", r.Attributes["id"], r.ID)
	}
}

func (app *application) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records, schema types.Type) error {

	app.mu.Lock()
	rng := rand.New(rand.NewPCG(app.rng.Uint64(), app.rng.Uint64()))
	app.mu.Unlock()

	// Test Peek.
	if rng.Int()%8 == 0 {
		record, _ := records.Peek()
		app.validateRecord(record)
		if rng.Int()%4 == 0 {
			record, ok := records.Peek()
			if !ok {
				return nil
			}
			app.validateRecord(record)
		}
	}

	// Test First.
	if rng.Int()%5 == 0 {
		app.validateRecord(records.First())
		time.Sleep(time.Duration(rng.Int()%10) * time.Nanosecond)
		return nil
	}

	var seq iter.Seq[connectors.Record]
	if rng.Int()%3 == 0 {
		seq = records.Same()
	} else {
		seq = records.All()
	}

	n := 0
	for r := range seq {
		app.validateRecord(r)
		if n%4 == 0 {
			if p, ok := records.Peek(); ok {
				app.validateRecord(p)
			}
		}
		if n > 0 && rng.Int()%3 == 0 {
			records.Postpone()
		} else if rng.Int()%16 == 0 {
			records.Discard(errors.New("event is invalid"))
		}
		if n == rng.IntN(minBatchSize) {
			break
		}
		n++
	}

	time.Sleep(time.Duration(rng.Int()%10) * time.Microsecond)

	return nil
}

func (app *application) ack(ids []string, err error) {
	if len(ids) == 0 {
		app.t.Fatalf("ack: expected at least one id, got none")
	}
	app.mu.Lock()
	if app.acks == nil {
		app.acks = []ack{}
	}
	app.acks = append(app.acks, ack{ids: ids, err: err})
	app.n += len(ids)
	app.mu.Unlock()
}
