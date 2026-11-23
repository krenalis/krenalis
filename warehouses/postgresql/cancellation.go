// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// cancellationConn closes the raw transport when its dialing context ends,
// unless the connection becomes the primary PostgreSQL connection.
type cancellationConn struct {
	net.Conn
	stop func()
}

func newCancellationConn(ctx context.Context, conn net.Conn) *cancellationConn {
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	// OnceFunc drops stop after its first call, releasing the dialing context
	// while TLS keeps the wrapper alive.
	return &cancellationConn{Conn: conn, stop: sync.OnceFunc(func() { stop() })}
}

// Close closes the transport and releases its context cancellation callback.
func (c *cancellationConn) Close() error {
	defer c.stop()
	return c.Conn.Close()
}

// configureCancellation bounds temporary cancellation connections by their
// dialing context. The primary connection is detached from that context.
// config must be unused and not already configured.
func configureCancellation(config *pgx.ConnConfig) {

	dial := config.DialFunc
	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		return newCancellationConn(ctx, conn), nil
	}

	afterNetConnect := config.AfterNetConnect
	config.AfterNetConnect = func(ctx context.Context, pgConfig *pgconn.Config, conn net.Conn) (net.Conn, error) {
		raw := conn
		tlsConn, isTLS := conn.(*tls.Conn)
		if isTLS {
			raw = tlsConn.NetConn()
		}
		guarded, ok := raw.(*cancellationConn)
		if !ok {
			// pgx closes the returned connection when this hook fails.
			return conn, fmt.Errorf("warehouses/postgresql: unexpected connection type %T", raw)
		}
		guarded.stop()
		if !isTLS {
			conn = guarded.Conn
		}
		if afterNetConnect != nil {
			return afterNetConnect(ctx, pgConfig, conn)
		}
		return conn, nil
	}

}

// retryClosedConnectionCancellation makes bounded best-effort cancellation
// attempts while the pool destroys an already closed connection, for example
// after a context cancellation. As the pool's BeforeClose hook, it runs after
// the caller has released the connection, without delaying the caller.
//
// Retries have a 10-second overall budget and a one-second limit per attempt.
// The first attempt starts after 50 milliseconds, with increasing delays capped
// at one second. The loop stops when it observes completion of the connection's
// local cleanup; an attempt in progress can continue until its deadline.
// The pool closes healthy connections normally, without additional cancellation
// attempts.
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
