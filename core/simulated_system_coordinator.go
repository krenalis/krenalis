// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"sync"
	"time"
)

// simulatedSystemCoordinator owns the executor for the current leadership.
// process runs one pass and reports whether pending work needs a retry.
type simulatedSystemCoordinator struct {
	ctx     context.Context
	process func(context.Context) bool
	mu      sync.Mutex
	closed  bool               // protected by mu
	leader  bool               // protected by mu
	cancel  context.CancelFunc // protected by mu
	wake    chan struct{}      // protected by mu
	wg      sync.WaitGroup
}

func newSimulatedSystemCoordinator(ctx context.Context, process func(context.Context) bool) *simulatedSystemCoordinator {
	return &simulatedSystemCoordinator{ctx: ctx, process: process}
}

func (c *simulatedSystemCoordinator) close() {
	c.mu.Lock()
	c.closed = true
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Unlock()
	c.wg.Wait()
}

func (c *simulatedSystemCoordinator) run(ctx context.Context, wake <-chan struct{}) {

	defer c.wg.Done()
	defer func() {
		c.mu.Lock()
		c.cancel()
		c.cancel = nil
		c.wake = nil
		c.start()
		c.mu.Unlock()
	}()

	timer := time.NewTimer(5 * time.Second)
	timer.Stop()
	defer timer.Stop()
	var retry <-chan time.Time
	for {
		if ctx.Err() != nil {
			return
		}
		if c.process(ctx) {
			if retry == nil {
				timer.Reset(5 * time.Second)
				retry = timer.C
			}
		} else {
			timer.Stop()
			retry = nil
		}
		select {
		case <-ctx.Done():
			return
		case <-wake:
		case <-retry:
			retry = nil
		}
	}

}

// setLeader records the latest election without waiting for executor exit.
// Callers must serialize elections with initial leadership reconciliation.
func (c *simulatedSystemCoordinator) setLeader(leader bool) {
	c.mu.Lock()
	c.leader = leader
	if !leader && c.cancel != nil {
		c.cancel()
	}
	c.start()
	c.mu.Unlock()
}

// start starts an executor if the previous one has exited. c.mu must be held.
func (c *simulatedSystemCoordinator) start() {
	if c.closed || !c.leader || c.cancel != nil || c.ctx.Err() != nil {
		return
	}
	ctx, cancel := context.WithCancel(c.ctx)
	c.cancel = cancel
	c.wake = make(chan struct{}, 1)
	c.wg.Add(1)
	go c.run(ctx, c.wake)
}

func (c *simulatedSystemCoordinator) wakeUp() {
	c.mu.Lock()
	if !c.closed && c.leader && c.wake != nil {
		select {
		case c.wake <- struct{}{}:
		default:
			// A pending signal already covers the current state.
		}
	}
	c.mu.Unlock()
}
