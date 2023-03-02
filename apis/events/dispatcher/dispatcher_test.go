//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package dispatcher_test

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"testing"
	"time"

	"chichi/apis/events/dispatcher"
	"chichi/apis/events/pipe"
)

const debug = false

var r *rand.Rand

func init() {
	r = rand.New(rand.NewSource(100))
}

// TestDispatcher tests the dispatcher.
func TestDispatcher(t *testing.T) {
	events := make(chan *pipe.Event, 1)
	done := make(chan *pipe.Event, 1)
	out := dispatcher.Dispatch(pipe.Channel{Events: events, Done: done})
	newSenderPool(context.Background(), 10, out)
	end := make(chan struct{})
	go func() {
		for event := range done {
			if debug {
				log.Printf("done received for event %d", event.ID)
			}
		}
		end <- struct{}{}
	}()
	start := time.Now()
	for i := 0; i < 100; i++ {
		anonymousId := string(byte(r.Intn(26) + 'a'))
		event := &pipe.Event{
			ID:          i,
			AnonymousId: anonymousId,
			Connection:  1,
			Endpoint:    1,
		}
		events <- event
	}
	close(events)
	_ = <-end
	log.Printf("time: %s", time.Now().Sub(start))
}

type senderPool struct {
	ctx  context.Context
	size int
	in   pipe.Channel
}

func newSenderPool(ctx context.Context, size int, in pipe.Channel) *senderPool {
	for i := 0; i < size; i++ {
		go worker(ctx, in)
	}
	return &senderPool{ctx, size, in}
}

func (p *senderPool) SetSize(size int) {
	if delta := size - p.size; delta > 0 {
		for i := 0; i < delta; i++ {
			go worker(p.ctx, p.in)
		}
	} else if delta < 0 {
		go func() {
			for i := 0; i > delta; i-- {
				//p.in <- nil
			}
		}()
	}
}

func worker(ctx context.Context, in pipe.Channel) {
	for {
		event := <-in.Events
		if event == nil {
			return
		}
		// do the request
		t := time.Duration(r.ExpFloat64() * float64(100*time.Millisecond))
		if t < 20*time.Millisecond {
			t = 20 * time.Millisecond
		} else if t > 2*time.Second {
			t = 2 * time.Second
		}
		timer := time.NewTimer(t)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
		n := r.Intn(100)
		if n < 1 {
			event.Err = dispatcher.ErrDestinationDown
		}
		if n < 5 {
			event.Err = errors.New("send error")
		}
		in.Done <- event
	}
}
