// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/krenalis/krenalis/warehouses"
)

type cancellationTestDialer struct {
	mu           sync.Mutex
	attempts     int // protected by mu
	failures     int
	discardFirst bool
}

func (d *cancellationTestDialer) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.attempts
}

// wrap injects cancellation failures without altering SQL or query results.
func (d *cancellationTestDialer) wrap(dial warehouses.DialFunc) warehouses.DialFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		return &cancellationTestConn{Conn: conn, dialer: d, first: true}, nil
	}
}

type cancellationTestConn struct {
	net.Conn
	dialer    *cancellationTestDialer
	first     bool
	discarded bool
}

// Read returns server data or EOF for a discarded cancellation request.
func (c *cancellationTestConn) Read(data []byte) (int, error) {
	if c.discarded {
		return 0, io.EOF
	}
	return c.Conn.Read(data)
}

// Write drops or rejects selected cancellation requests before delivery.
func (c *cancellationTestConn) Write(data []byte) (int, error) {
	if c.first {
		c.first = false
		// The disposable server has no TLS. Recognize only cancellation's
		// startup packet, without reading or logging its PID and secret key.
		if len(data) >= 16 && binary.BigEndian.Uint32(data[4:8]) == 80877102 {
			c.dialer.mu.Lock()
			c.dialer.attempts++
			attempt := c.dialer.attempts
			c.dialer.mu.Unlock()
			if attempt <= c.dialer.failures {
				return 0, errors.New("simulated cancellation delivery failure")
			}
			if attempt == 1 && c.dialer.discardFirst {
				c.discarded = true
				err := c.Conn.Close()
				if err != nil {
					return 0, err
				}
				return len(data), nil
			}
		}
	}
	return c.Conn.Write(data)
}

// TestRetryClosedConnectionCancellation checks the pool hook on a real server.
func TestRetryClosedConnectionCancellation(t *testing.T) {

	original, originalPool := newTestPostgreSQLWarehouse(t)
	observer, err := pgx.Connect(t.Context(), originalPool.Config().ConnConfig.ConnString())
	if err != nil {
		t.Fatalf("expected observer connection, got %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := observer.Close(ctx)
		if err != nil {
			t.Errorf("expected observer cleanup, got %v", err)
		}
	})
	_, err = observer.Exec(t.Context(), "CREATE TABLE cancellation_writes (id int PRIMARY KEY)")
	if err != nil {
		t.Fatalf("expected transaction fixture, got %v", err)
	}

	for _, c := range []struct {
		name         string
		failures     int
		discardFirst bool
		kind         string
	}{
		{"lost_first_request", 0, true, "query"},
		{"transient_errors", 3, false, "query"},
		{"healthy_connection", 0, false, "healthy"},
		{"streaming", 0, true, "streaming"},
		{"transaction", 0, true, "transaction"},
		{"ordinary_cancellation", 0, false, "query"},
		{"unavailable_channel", 1000, false, "unavailable"},
	} {
		t.Run(c.name, func(t *testing.T) {

			dialer := &cancellationTestDialer{failures: c.failures, discardFirst: c.discardFirst}
			warehouse := New(original.settings, dialer.wrap)
			t.Cleanup(func() {
				err := warehouse.Close()
				if err != nil {
					t.Errorf("expected warehouse cleanup, got %v", err)
				}
			})
			pool, _, err := warehouse.connectionPool(t.Context(), false)
			if err != nil {
				t.Fatalf("expected warehouse pool, got %v", err)
			}
			acquired, err := pool.Acquire(t.Context())
			if err != nil {
				t.Fatalf("expected warehouse connection, got %v", err)
			}
			pid := acquired.Conn().PgConn().PID()
			cleanup := acquired.Conn().PgConn().CleanupDone()
			acquired.Release()
			ctx, cancel := context.WithTimeout(t.Context(), 35*time.Second)
			defer cancel()
			var tx pgx.Tx
			var finished chan struct{}
			t.Cleanup(func() {
				// Use an independent context to stop the fixture even if an
				// assertion fails or the canceled query's context has expired.
				cancel()
				cleanupCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
				defer stop()
				_, err := observer.Exec(cleanupCtx, "SELECT pg_cancel_backend($1)", pid)
				if err != nil {
					t.Errorf("expected fixture cancellation, got %v", err)
				}
				if finished != nil {
					select {
					case <-finished:
					case <-cleanupCtx.Done():
						t.Error("expected interrupted query worker to finish, got timeout")
						return
					}
				}
				if tx != nil {
					_ = tx.Rollback(cleanupCtx)
				}
			})

			if c.kind == "healthy" {
				var value int
				row := pool.QueryRow(ctx, "SELECT 42")
				err := row.Scan(&value)
				if err != nil {
					t.Fatalf("expected healthy query, got %v", err)
				}
				if value != 42 {
					t.Fatalf("expected value 42, got %d", value)
				}
				cancel()
				err = warehouse.Close()
				if err != nil {
					t.Fatalf("expected healthy warehouse close, got %v", err)
				}
				if count := dialer.count(); count != 0 {
					t.Fatalf("expected no cancellation attempts on healthy close, got %d", count)
				}
				return
			}

			if c.kind == "transaction" {
				tx, err = pool.Begin(t.Context())
				if err != nil {
					t.Fatalf("expected transaction, got %v", err)
				}
				_, err = tx.Exec(t.Context(), "INSERT INTO cancellation_writes VALUES (1)")
				if err != nil {
					t.Fatalf("expected uncommitted write, got %v", err)
				}
			}
			done := make(chan error, 1)
			finished = make(chan struct{})
			if c.kind == "streaming" {
				rows, err := pool.Query(ctx, `SELECT i,repeat('x',8192),
pg_sleep(CASE WHEN i<3 THEN 0 ELSE 30 END) FROM generate_series(1,3) i`)
				if err != nil {
					close(finished)
					t.Fatalf("expected streaming query, got %v", err)
				}
				if !rows.Next() {
					rows.Close()
					close(finished)
					t.Fatalf("expected first streamed row, got %v", rows.Err())
				}
				go func() {
					defer close(finished)
					count := 1
					for rows.Next() {
						count++
						if count >= 3 {
							rows.Close()
							done <- errors.New("warehouses/postgresql: delayed row completed before cancellation")
							return
						}
					}
					rows.Close()
					done <- rows.Err()
				}()
			} else {
				go func() {
					defer close(finished)
					if tx != nil {
						_, err := tx.Exec(ctx, "SELECT pg_sleep(30)")
						done <- err
						return
					}
					_, err := pool.Exec(ctx, "SELECT pg_sleep(30)")
					done <- err
				}()
			}
			waitForCancellationQuery(t, observer, pid, true)
			canceled := time.Now()
			cancel()
			select {
			case err := <-done:
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						t.Fatalf("expected context cancellation, got %v", err)
					}
					break
				}
				t.Fatal("expected canceled operation, got success")
			case <-time.After(time.Second):
				t.Fatal("expected prompt caller return, got timeout")
			}
			if tx != nil {
				// Rollback releases the pooled connection even when reporting
				// that cancellation has already closed it.
				_ = tx.Rollback(t.Context())
			}
			if c.kind == "unavailable" {

				// Observe a failed retry while every cancellation channel is
				// unavailable, without waiting out the full ten-second budget.
				waitCtx, stop := context.WithTimeout(t.Context(), 3*time.Second)
				defer stop()
				ticker := time.NewTicker(10 * time.Millisecond)
				defer ticker.Stop()
				for dialer.count() < 2 {
					select {
					case <-waitCtx.Done():
						t.Fatal("expected retry on unavailable channel, got timeout")
					case <-ticker.C:
					}
				}

				// The query still runs: stopping it here is fixture cleanup,
				// not evidence that the failed requests canceled the query.
				waitForCancellationQuery(t, observer, pid, true)
				var sent bool
				row := observer.QueryRow(waitCtx, "SELECT pg_cancel_backend($1)", pid)
				err := row.Scan(&sent)
				if err != nil {
					t.Fatalf("expected independent fixture cancellation, got %v", err)
				}
				if !sent {
					t.Fatal("expected fixture cancellation delivered, got false")
				}

			}
			waitForCancellationQuery(t, observer, pid, false)
			select {
			case <-cleanup:
			case <-time.After(2 * time.Second):
				t.Fatal("expected discarded connection cleanup, got timeout")
			}
			var replacement uint32
			row := pool.QueryRow(t.Context(), "SELECT pg_backend_pid()")
			err = row.Scan(&replacement)
			if err != nil {
				t.Fatalf("expected successful follow-up query, got %v", err)
			}
			if replacement == pid {
				t.Fatal("expected replacement connection, got discarded backend reused")
			}
			if tx != nil {
				var count int
				row := observer.QueryRow(t.Context(), "SELECT count(*) FROM cancellation_writes")
				err := row.Scan(&count)
				if err != nil {
					t.Fatalf("expected transaction state, got %v", err)
				}
				if count != 0 {
					t.Fatalf("expected uncommitted write rolled back, got %d rows", count)
				}
			}
			err = warehouse.Close()
			if err != nil {
				t.Fatalf("expected warehouse close, got %v", err)
			}
			minimum := c.failures + 1
			if c.discardFirst || c.kind == "unavailable" {
				minimum = 2
			}
			if count := dialer.count(); count < minimum || count > 16 {
				t.Fatalf("expected %d to 16 cancellation attempts, got %d", minimum, count)
			}
			t.Logf("attempts=%d recovery_ms=%d", dialer.count(), time.Since(canceled).Milliseconds())

		})
	}

}

// waitForCancellationQuery observes execution independently of query context.
func waitForCancellationQuery(t *testing.T, observer *pgx.Conn, pid uint32, sleeping bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var active, sleeps int
		queryCtx, stop := context.WithTimeout(t.Context(), time.Second)
		row := observer.QueryRow(queryCtx, `SELECT count(*) FILTER (WHERE state='active'),
count(*) FILTER (WHERE wait_event='PgSleep') FROM pg_stat_activity WHERE pid=$1`, pid)
		err := row.Scan(&active, &sleeps)
		stop()
		if err != nil {
			t.Fatalf("expected target activity, got %v", err)
		}
		if active < 0 || active > 1 || sleeps < 0 || sleeps > active {
			t.Fatalf("expected bounded target activity, got %d/%d", active, sleeps)
		}
		if sleeping && sleeps == 1 || !sleeping && active == 0 {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("expected target sleeping=%t, got active=%d sleeping=%d", sleeping, active, sleeps)
		case <-ticker.C:
		}
	}
}
