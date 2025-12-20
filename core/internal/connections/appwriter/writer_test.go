// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package appwriter

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/state"
)

func Test_Writer(t *testing.T) {

	tests := []struct {
		num    int     // number of records to process
		seed   int64   // seed value to pseudo-randomize the Upsert method
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
			w := New("test", state.TargetUser, app.Upsert, app.ack)

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

type ack struct {
	ids []string
	err error
}

type application struct {
	t    *testing.T
	rng  *rand.Rand
	mu   sync.Mutex
	n    int
	acks []ack
}

func newApplication(t *testing.T, seed int64) *application {
	return &application{t: t, rng: rand.New(rand.NewSource(seed))}
}
func (app *application) validateRecord(r connectors.Record) {
	if r.Attributes == nil {
		app.t.Fatal("Upsert: expected attributes, got nil")
	}
	if r.Attributes["id"] != r.ID {
		app.t.Fatalf("Upsert: expected attributes[\"id\"] == %q, got %q", r.Attributes["id"], r.ID)
	}
}

func (app *application) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records) error {

	// Test Peek.
	if app.rng.Int()%8 == 0 {
		record, _ := records.Peek()
		app.validateRecord(record)
		if app.rng.Int()%4 == 0 {
			record, ok := records.Peek()
			if !ok {
				return nil
			}
			app.validateRecord(record)
		}
	}

	// Test First.
	if app.rng.Int()%5 == 0 {
		app.validateRecord(records.First())
		time.Sleep(time.Duration(app.rng.Int()%10) * time.Nanosecond)
		return nil
	}

	var seq iter.Seq[connectors.Record]
	if app.rng.Int()%3 == 0 {
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
		if n > 0 && app.rng.Int()%3 == 0 {
			records.Postpone()
		} else if app.rng.Int()%16 == 0 {
			records.Discard(errors.New("event is invalid"))
		}
		if n == app.rng.Int()/2 {
			break
		}
		n++
	}

	time.Sleep(time.Duration(app.rng.Int()%10) * time.Microsecond)

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
