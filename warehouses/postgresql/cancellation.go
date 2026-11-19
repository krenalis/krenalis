// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// retryClosedConnectionCancellation makes bounded best-effort cancellation
// attempts while the pool destroys an already closed connection. It is called
// only by BeforeClose, after the caller has relinquished the connection.
// CleanupDone stops retries; it does not acknowledge server-side termination.
func retryClosedConnectionCancellation(conn *pgx.Conn) {
	target := conn.PgConn()
	if !target.IsClosed() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	delay := 50 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	for {
		select {
		case <-target.CleanupDone():
			return
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		// Cleanup may have finished at the same time as the timer.
		select {
		case <-target.CleanupDone():
			return
		case <-ctx.Done():
			return
		default:
			// The connection still needs bounded cleanup attempts.
		}
		attempt, stop := context.WithTimeout(ctx, time.Second)
		// Delivery failures are best effort too: retry until cleanup or the
		// overall budget, without changing the caller's original error.
		_ = target.CancelRequest(attempt)
		stop()
		delay = min(2*delay, time.Second)
		timer.Reset(delay)
	}
}
