// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package krenalistester

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
)

// TestStartServerCanceledWhileWaitingForAssets checks cancellation before the
// server goroutine is launched.
func TestStartServerCanceledWhileWaitingForAssets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		k := &Krenalis{assetsGenerated: make(chan struct{})}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		var err error
		go func() {
			err = k.startServer(ctx, func() { t.Error("expected canceled startup, got server launch") })
		}()
		synctest.Wait()
		cancel()
		synctest.Wait()
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context.Canceled, got %v", err)
			}
			if k.krenalisRunning != nil {
				t.Fatal("expected no server exit channel, got an initialized channel")
			}
			return
		}
		t.Fatal("expected context.Canceled, got nil")
	})
}

// TestStartServerWaitsForAssets checks that the server starts only after asset
// generation completes.
func TestStartServerWaitsForAssets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		k := &Krenalis{assetsGenerated: make(chan struct{})}
		started := false
		var err error
		go func() {
			err = k.startServer(t.Context(), func() { started = true })
		}()
		synctest.Wait()
		if started {
			t.Fatal("expected server to wait for assets, got early start")
		}
		close(k.assetsGenerated)
		synctest.Wait()
		if err != nil {
			t.Fatalf("expected successful startup, got %v", err)
		}
		if !started {
			t.Fatal("expected server to start after asset generation, got no start")
		}
	})
}
