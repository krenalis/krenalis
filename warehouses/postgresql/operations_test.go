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

// Test_finalizationContext checks that the finalization context outlives its
// parent by finalizationGracePeriod and is canceled by its cancel function.
func Test_finalizationContext(t *testing.T) {

	t.Run("outlives its parent by the grace period", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			parent, cancelParent := context.WithCancel(t.Context())
			ctx, cancel := finalizationContext(parent)
			defer cancel()
			cancelParent()
			time.Sleep(finalizationGracePeriod - time.Nanosecond)
			synctest.Wait()
			if err := ctx.Err(); err != nil {
				t.Fatalf("expected nil error before the grace period ends, got %v", err)
			}
			time.Sleep(time.Nanosecond)
			synctest.Wait()
			if err := ctx.Err(); err != context.Canceled {
				t.Fatalf("expected %v after the grace period ends, got %v", context.Canceled, err)
			}
		})
	})

	t.Run("is canceled by its cancel function", func(t *testing.T) {
		ctx, cancel := finalizationContext(t.Context())
		cancel()
		if err := ctx.Err(); err != context.Canceled {
			t.Fatalf("expected %v, got %v", context.Canceled, err)
		}
	})

}
