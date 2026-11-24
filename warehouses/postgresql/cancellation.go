// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const automaticCancelKey = "warehouses/postgresql.automaticCancel"

type cancellationRetryKey struct{}

// cancellationConn closes the transport when its context ends, even if TLS
// shutdown extends the write deadline set by pgx.
type cancellationConn struct {
	net.Conn
	stop func() bool
}

// Close closes the transport and releases its context cancellation callback.
func (c *cancellationConn) Close() error {
	defer c.stop()
	return c.Conn.Close()
}

// configureCancellation bounds the cancellation dials of one pool connection.
// The pool must pass a fresh config for each connection it constructs.
func configureCancellation(config *pgx.ConnConfig) {

	// pgx uses the same dialer for startup (including fallbacks) and later
	// cancellation requests. Leave all startup transports under pgx's control.
	var connected, automaticCancel atomic.Bool
	afterConnect := config.AfterConnect
	config.AfterConnect = func(ctx context.Context, conn *pgconn.PgConn) error {
		conn.CustomData()[automaticCancelKey] = &automaticCancel
		connected.Store(true)
		if afterConnect != nil {
			return afterConnect(ctx, conn)
		}
		return nil
	}

	dial := config.DialFunc
	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		if !connected.Load() {
			return dial(ctx, network, address)
		}
		if ctx.Value(cancellationRetryKey{}) == nil {
			automaticCancel.Store(true)
		}
		conn, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
		return &cancellationConn{Conn: conn, stop: stop}, nil
	}

}

// retryClosedConnectionCancellation makes bounded best-effort cancellation
// attempts while the pool destroys an already closed connection, for example
// after a context cancellation. The pool runs BeforeClose in its destructor,
// so releasing the connection does not wait for retries.
//
// Retries have a 10-second overall budget and a one-second limit per attempt.
// While local cleanup is pending, the first retry starts after 50 milliseconds
// and subsequent delays double up to one second. If cleanup finishes after
// pgx attempted to cancel and no pool retry succeeded, retry immediately and
// continue with backoff until a retry succeeds or the same budget expires.
// pgx does not expose the outcome of its automatic request.
//
// CleanupDone does not confirm that the server stopped the query. If the
// cancellation channel is unavailable, the server may continue executing.
// The outcome of a canceled autocommit write or COMMIT can remain unknown:
// the transaction may have committed before the cancellation took effect.
func retryClosedConnectionCancellation(conn *pgx.Conn) {

	target := conn.PgConn()
	if !target.IsClosed() {
		return
	}
	// PostgreSQL cancellation keys are 4 to 256 bytes. pgx copies the
	// server-supplied key into a fresh buffer for each request.
	if n := len(target.SecretKey()); n < 4 || n > 256 {
		return
	}

	retryCtx := context.WithValue(context.Background(), cancellationRetryKey{}, true)
	budget, endBudget := context.WithTimeout(retryCtx, 10*time.Second)
	defer endBudget()
	cleanupCtx, cancel := context.WithCancel(budget)
	defer cancel()
	go func() {
		select {
		case <-target.CleanupDone():
			cancel()
		case <-cleanupCtx.Done():
		}
	}()

	delay := 50 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	ctx := cleanupCtx
	delivered := false
	for {
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
		if budget.Err() != nil {
			return
		}
		// Cleanup may have finished at the same time as the timer.
		select {
		case <-target.CleanupDone():
			automaticCancel := target.CustomData()[automaticCancelKey].(*atomic.Bool)
			if delivered || !automaticCancel.Load() {
				return
			}
			ctx = budget
		default:
			// Local cleanup can still interrupt this attempt.
		}
		attempt, stop := context.WithTimeout(ctx, time.Second)
		// pgx ignores read errors while waiting for the server to close the
		// cancellation socket, including when our context interrupts the read.
		err := target.CancelRequest(attempt)
		if err == nil && attempt.Err() == nil {
			delivered = true
		}
		stop()
		if delivered && ctx == budget {
			return
		}
		delay = min(2*delay, time.Second)
		timer.Reset(delay)
	}

}
