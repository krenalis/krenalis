// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
)

// TestPipelinesWaitStore checks that WaitStore waits for the requested tick.
func TestPipelinesWaitStore(t *testing.T) {

	tests := []struct {
		name       string
		storedTick int
	}{
		{name: "exact_tick", storedTick: 2},
		{name: "later_tick", storedTick: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {

				p := newStubPipelines()
				p.tick = 2
				result := make(chan error, 1)
				go func() { result <- p.WaitStore(t.Context()) }()
				synctest.Wait()

				p.Lock()
				p.stored.Broadcast()
				p.Unlock()
				synctest.Wait()
				select {
				case err := <-result:
					t.Fatalf("expected waiter blocked after wakeup without storage, got %v", err)
				default:
				}

				p.Lock()
				p.stored.tick = 1
				p.stored.Broadcast()
				p.Unlock()
				synctest.Wait()
				select {
				case err := <-result:
					t.Fatalf("expected waiter blocked after partial storage, got %v", err)
				default:
				}

				p.Lock()
				p.tick = test.storedTick + 1
				p.stored.tick = test.storedTick
				p.stored.Broadcast()
				p.Unlock()
				err := <-result
				if err != nil {
					t.Fatalf("expected completed wait after stored tick %d, got %v", test.storedTick, err)
				}

			})
		})
	}

}

// TestPipelinesWaitStoreDeadline checks that waiting stops when the
// context deadline expires.
func TestPipelinesWaitStoreDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		p := newStubPipelines()
		p.tick = 1
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		start := time.Now()
		err := p.WaitStore(ctx)
		if err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("expected context.DeadlineExceeded, got %v", err)
			}
			if elapsed := time.Since(start); elapsed != time.Second {
				t.Fatalf("expected wait of %s, got %s", time.Second, elapsed)
			}
			return
		}

		t.Fatal("expected context.DeadlineExceeded, got nil")

	})
}
