// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package krenalistester

import (
	"testing"
	"testing/synctest"
)

// TestStartServerWaitsForAssets checks that the server starts only after asset
// generation completes.
func TestStartServerWaitsForAssets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		k := &Krenalis{assetsGenerated: make(chan struct{})}
		started := false
		go k.startServer(func() { started = true })
		synctest.Wait()
		if started {
			t.Fatal("expected server to wait for assets, got early start")
		}
		close(k.assetsGenerated)
		synctest.Wait()
		if !started {
			t.Fatal("expected server to start after asset generation, got no start")
		}
	})
}
