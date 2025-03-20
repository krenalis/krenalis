//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package appwriter

import (
	"context"
	"fmt"
	"iter"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/types"
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
		{num: maxAvailable / 2, seed: 63, create: 0.16},
		{num: maxAvailable * 3, seed: 47, create: 0.75},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%d/%d/%f", test.num, test.seed, test.create), func(t *testing.T) {

			app := newApp(t, test.seed)
			w := New(app.ack, state.Users, app, "test")

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
				properties := map[string]any{
					"id": id,
				}
				if !w.Write(ctx, id, properties) {
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
				if ack.err != nil {
					t.Fatalf("ack %d/%d: expected no error, got %#v", i+1, test.num, ack.err)
				}
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

type app struct {
	t    *testing.T
	rng  *rand.Rand
	mu   sync.Mutex
	n    int
	acks []ack
}

func newApp(t *testing.T, seed int64) *app {
	return &app{t: t, rng: rand.New(rand.NewSource(seed))}
}

func (app *app) Records(ctx context.Context, target meergo.Targets, schema types.Type, lastChangeTime time.Time, ids, properties []string, cursor string) ([]meergo.Record, string, error) {
	return nil, "", nil
}

func (app *app) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {
	return types.Type{}, nil
}

func (app *app) validateRecord(r meergo.Record) {
	if r.Properties == nil {
		app.t.Fatal("Upsert: expected properties, got nil")
	}
	if r.Properties["id"] != r.ID {
		app.t.Fatalf("Upsert: expected properties[\"id\"] == %q, got %q", r.Properties["id"], r.ID)
	}
}

func (app *app) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {

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

	var seq iter.Seq2[int, meergo.Record]
	if app.rng.Int()%3 == 0 {
		seq = records.Same()
	} else {
		seq = records.All()
	}

	for i, r := range seq {
		app.validateRecord(r)
		if i%4 == 0 {
			if p, ok := records.Peek(); ok {
				app.validateRecord(p)
			}
		}
		if app.rng.Int()%3 == 0 {
			records.Skip()
		}
		if i == app.rng.Int()/2 {
			break
		}
	}

	time.Sleep(time.Duration(app.rng.Int()%10) * time.Microsecond)

	return nil
}

func (app *app) ack(ids []string, err error) {
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
