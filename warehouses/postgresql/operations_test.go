// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

// Test_withDelayedCancellation checks that the returned context outlives its
// parent by the delay and is canceled by its cancel function.
func Test_withDelayedCancellation(t *testing.T) {

	t.Run("outlives its parent by the delay", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			const delay = 10 * time.Second
			parent, cancelParent := context.WithCancel(t.Context())
			ctx, cancel := withDelayedCancellation(parent, delay)
			defer cancel()
			cancelParent()
			time.Sleep(delay - time.Nanosecond)
			synctest.Wait()
			if err := ctx.Err(); err != nil {
				t.Fatalf("expected nil error before the delay ends, got %v", err)
			}
			time.Sleep(time.Nanosecond)
			synctest.Wait()
			if err := ctx.Err(); err != context.Canceled {
				t.Fatalf("expected %v after the delay ends, got %v", context.Canceled, err)
			}
		})
	})

	t.Run("is canceled by its cancel function", func(t *testing.T) {
		ctx, cancel := withDelayedCancellation(t.Context(), 10*time.Second)
		cancel()
		if err := ctx.Err(); err != context.Canceled {
			t.Fatalf("expected %v, got %v", context.Canceled, err)
		}
	})

}
