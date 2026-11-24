// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/krenalis/krenalis/warehouses"
)

const cancellationTestDSN = "host=127.0.0.1 port=5432 user=test dbname=test sslmode=disable sslcertmode=disable" +
	" sslrootcert='' target_session_attrs=any require_auth=none channel_binding=disable"

// cancellationTestDialer simulates delayed, failed or dropped cancellation
// requests and counts cancellation attempts.
type cancellationTestDialer struct {
	mu           sync.Mutex
	attempts     int // protected by mu
	failures     int
	discardFirst bool
	blockFirst   func()
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

// Write delays, drops or rejects selected cancellation requests.
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
			if attempt == 1 && c.dialer.blockFirst != nil {
				c.dialer.blockFirst()
			}
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

// TestConfigureCancellation checks that the configured hooks receive the
// established primary connection.
func TestConfigureCancellation(t *testing.T) {

	config, err := pgx.ParseConfig(cancellationTestDSN)
	if err != nil {
		t.Fatalf("expected connection config, got %v", err)
	}

	synctest.Test(t, func(t *testing.T) {

		var transport net.Conn
		config.AfterNetConnect = func(_ context.Context, _ *pgconn.Config, conn net.Conn) (net.Conn, error) {
			transport = conn
			return conn, nil
		}
		var established *pgconn.PgConn
		config.AfterConnect = func(_ context.Context, conn *pgconn.PgConn) error {
			established = conn
			return nil
		}

		conn, finishCleanup := newCancellationTestConnection(t, config, nil, 4)
		defer finishCleanup()
		if established != conn.PgConn() {
			t.Fatalf("expected AfterConnect to receive %p, got %p", conn.PgConn(), established)
		}
		if transport != established.Conn() {
			t.Fatalf("expected AfterNetConnect to receive %p, got %p", established.Conn(), transport)
		}

	})

}

// TestConfigureCancellationConnectTimeout checks the error from a timeout
// while pgx is still negotiating TLS on the primary connection.
func TestConfigureCancellationConnectTimeout(t *testing.T) {

	config, err := pgx.ParseConfig("host=127.0.0.1 port=5432 user=test dbname=test" +
		" sslmode=require sslnegotiation=postgres sslcertmode=disable sslrootcert=''")
	if err != nil {
		t.Fatalf("expected connection config, got %v", err)
	}

	synctest.Test(t, func(t *testing.T) {

		client, server := net.Pipe()
		defer client.Close()
		defer server.Close()
		config.DialFunc = func(context.Context, string, string) (net.Conn, error) { return client, nil }
		configureCancellation(config)
		go func() {
			// Consume the SSL request and leave the response pending.
			_, err := io.ReadFull(server, make([]byte, 8))
			if err != nil {
				t.Errorf("expected SSL request, got %v", err)
			}
		}()

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := pgx.ConnectConfig(ctx, config)
		if err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("expected connection deadline exceeded, got %v", err)
			}
			return
		}
		t.Fatal("expected connection deadline exceeded, got success")

	})

}

// TestRetryClosedConnectionCancellation verifies cancellation retries
// through the connection pool against a real PostgreSQL server.
func TestRetryClosedConnectionCancellation(t *testing.T) {

	original, originalPool := newTestPostgreSQLWarehouse(t)
	_, err := originalPool.Exec(t.Context(), "CREATE TABLE cancellation_writes (id int PRIMARY KEY)")
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
		{"blocked_first_request", 0, false, "blocked"},
		{"transient_errors", 3, false, "query"},
		{"healthy_connection", 0, false, "healthy"},
		{"streaming", 0, true, "streaming"},
		{"transaction", 0, true, "transaction"},
		{"ordinary_cancellation", 0, false, "query"},
		{"unavailable_channel", 1000, false, "unavailable"},
	} {
		t.Run(c.name, func(t *testing.T) {

			dialer := &cancellationTestDialer{failures: c.failures, discardFirst: c.discardFirst}
			var firstPending chan struct{}
			var releaseFirst context.CancelFunc
			if c.kind == "blocked" {
				firstPending = make(chan struct{})
				var hold context.Context
				hold, releaseFirst = context.WithCancel(t.Context())
				defer releaseFirst()
				dialer.blockFirst = func() {
					close(firstPending)
					<-hold.Done()
				}
			}
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
				_, err := originalPool.Exec(cleanupCtx, "SELECT pg_cancel_backend($1)", pid)
				if err != nil {
					t.Errorf("expected fixture cancellation, got %v", err)
				}
				if finished != nil {
					// The worker owns rollback and release after it starts.
					select {
					case <-finished:
						return
					case <-cleanupCtx.Done():
						t.Error("expected interrupted query worker to finish, got timeout")
						return
					}
				}
				if tx != nil {
					_ = tx.Rollback(cleanupCtx)
				}
				acquired.Release()
			})

			if c.kind == "healthy" {
				var value int
				row := acquired.QueryRow(ctx, "SELECT 42")
				err := row.Scan(&value)
				if err != nil {
					t.Fatalf("expected healthy query, got %v", err)
				}
				if value != 42 {
					t.Fatalf("expected value 42, got %d", value)
				}
				cancel()
				acquired.Release()
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
				tx, err = acquired.Begin(t.Context())
				if err != nil {
					t.Fatalf("expected transaction, got %v", err)
				}
				_, err = tx.Exec(t.Context(), "INSERT INTO cancellation_writes VALUES (1)")
				if err != nil {
					t.Fatalf("expected uncommitted write, got %v", err)
				}
			}
			done := make(chan error, 1)
			if c.kind == "streaming" {
				rows, err := acquired.Query(ctx, `SELECT i,repeat('x',8192),
pg_sleep(CASE WHEN i<3 THEN 0 ELSE 30 END) FROM generate_series(1,3) i`)
				if err != nil {
					t.Fatalf("expected streaming query, got %v", err)
				}
				if !rows.Next() {
					rows.Close()
					t.Fatalf("expected first streamed row, got %v", rows.Err())
				}
				finished = make(chan struct{})
				go func() {
					defer close(finished)
					count := 1
					for rows.Next() {
						count++
						if count >= 3 {
							rows.Close()
							acquired.Release()
							done <- errors.New("warehouses/postgresql: delayed row completed before cancellation")
							return
						}
					}
					rows.Close()
					err := rows.Err()
					acquired.Release()
					done <- err
				}()
			} else {
				finished = make(chan struct{})
				go func() {
					defer close(finished)
					if tx != nil {
						_, err := tx.Exec(ctx, "SELECT pg_sleep(30)")
						_ = tx.Rollback(t.Context())
						acquired.Release()
						done <- err
						return
					}
					_, err := acquired.Exec(ctx, "SELECT pg_sleep(30)")
					if firstPending != nil {
						// Let pgx reach its first write before pool release can start a retry.
						select {
						case <-firstPending:
						case <-t.Context().Done():
						}
					}
					acquired.Release()
					done <- err
				}()
			}
			waitForCancellationQuery(t, originalPool, pid, true)
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
			case <-time.After(3 * time.Second):
				t.Fatal("expected prompt caller return, got timeout")
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
				waitForCancellationQuery(t, originalPool, pid, true)
				var sent bool
				row := originalPool.QueryRow(waitCtx, "SELECT pg_cancel_backend($1)", pid)
				err := row.Scan(&sent)
				if err != nil {
					t.Fatalf("expected independent fixture cancellation, got %v", err)
				}
				if !sent {
					t.Fatal("expected fixture cancellation delivered, got false")
				}

			}

			waitForCancellationQuery(t, originalPool, pid, false)
			if releaseFirst != nil {
				if count := dialer.count(); count < 2 {
					t.Fatalf("expected a retry while the first request was pending, got %d attempts", count)
				}
				releaseFirst()
			}
			select {
			case <-cleanup:
			case <-time.After(3 * time.Second):
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
				checkCtx, stop := context.WithTimeout(t.Context(), 3*time.Second)
				defer stop()
				_, err := originalPool.Exec(checkCtx, "INSERT INTO cancellation_writes VALUES (1)")
				if err != nil {
					t.Fatalf("expected to reuse the primary key after rollback, got %v", err)
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
			if count := dialer.count(); count < minimum {
				t.Fatalf("expected at least %d cancellation attempts, got %d", minimum, count)
			}
			t.Logf("attempts=%d total_elapsed_ms=%d", dialer.count(), time.Since(canceled).Milliseconds())

		})
	}

}

// TestRetryClosedConnectionCancellationLimits verifies retry timeouts,
// the overall retry deadline, and cleanup using simulated time and
// in-memory connections.
func TestRetryClosedConnectionCancellationLimits(t *testing.T) {

	config, err := pgx.ParseConfig(cancellationTestDSN)
	if err != nil {
		t.Fatalf("expected connection config, got %v", err)
	}

	for _, c := range []struct {
		name         string
		blockAt      string
		cleanupAfter time.Duration
		wantAttempts int
	}{
		{"dial_errors", "", 0, 13},
		{"blocked_dial", "dial", 0, 7},
		{"blocked_write", "write", 0, 7},
		{"blocked_read", "read", 0, 7},
		{"cleanup_before_retry", "", 25 * time.Millisecond, 0},
		{"cleanup_during_backoff", "", 75 * time.Millisecond, 1},
		{"cleanup_during_dial", "dial", 75 * time.Millisecond, 1},
		{"cleanup_during_write", "write", 75 * time.Millisecond, 1},
		{"cleanup_during_read", "read", 75 * time.Millisecond, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {

				var attempts []time.Time
				var servers []net.Conn
				config := config.Copy()
				config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
					attempts = append(attempts, time.Now())
					deadline, ok := ctx.Deadline()
					if limit := time.Until(deadline); !ok || limit <= 0 || limit > time.Second {
						t.Errorf("expected cancellation deadline within one second, got %v", limit)
					}
					switch c.blockAt {
					case "":
						return nil, errors.New("simulated cancellation dial failure")
					case "dial":
						<-ctx.Done()
						return nil, ctx.Err()
					case "read", "write":
						client, server := net.Pipe()
						t.Cleanup(func() { _ = server.Close() })
						servers = append(servers, server)
						if c.blockAt == "read" {
							go func() {
								_, err := io.ReadFull(server, make([]byte, 16))
								if err != nil {
									t.Errorf("expected cancellation packet, got %v", err)
								}
							}()
						}
						return client, nil
					default:
						return nil, errors.New("warehouses/postgresql: unexpected blocking stage " + c.blockAt)
					}
				}
				conn, finishCleanup := newCancellationTestConnection(t, config, nil, 4)
				limit := 10 * time.Second
				if c.cleanupAfter > 0 {
					limit = c.cleanupAfter
					time.AfterFunc(limit, finishCleanup)
				}
				started := time.Now()
				retryClosedConnectionCancellation(conn)
				elapsed := time.Since(started)
				if elapsed != limit {
					t.Fatalf("expected retry duration %v, got %v", limit, elapsed)
				}
				if len(attempts) != c.wantAttempts {
					t.Fatalf("expected %d cancellation attempts, got %d", c.wantAttempts, len(attempts))
				}
				if len(attempts) > 0 && attempts[0].Sub(started) != 50*time.Millisecond {
					t.Fatalf("expected first retry after 50ms, got %v", attempts[0].Sub(started))
				}
				wantDelay := 100 * time.Millisecond
				for i := 1; i < len(attempts); i++ {
					delay := attempts[i].Sub(attempts[i-1])
					if c.blockAt != "" {
						delay -= time.Second
					}
					if delay != wantDelay {
						t.Fatalf("expected delay %v before retry %d, got %v", wantDelay, i+1, delay)
					}
					wantDelay = min(2*wantDelay, time.Second)
				}

				count := len(attempts)
				time.Sleep(2 * time.Second)
				synctest.Wait()
				if len(attempts) != count {
					t.Fatalf("expected no attempts after retries stopped, got %d", len(attempts)-count)
				}
				for _, server := range servers {
					n, err := server.Read(make([]byte, 16))
					if err != nil {
						if !errors.Is(err, io.EOF) {
							t.Fatalf("expected closed cancellation connection, got %v", err)
						}
						continue
					}
					t.Fatalf("expected closed cancellation connection, got %d bytes", n)
				}
				finishCleanup()
				synctest.Wait()
				select {
				case <-conn.PgConn().CleanupDone():
				default:
					t.Fatal("expected connection cleanup, got pending cleanup")
				}

			})
		})
	}

}

// TestRetryClosedConnectionCancellationSecretKey checks which backend keys can
// be used for a retry.
func TestRetryClosedConnectionCancellationSecretKey(t *testing.T) {

	config, err := pgx.ParseConfig(cancellationTestDSN)
	if err != nil {
		t.Fatalf("expected connection config, got %v", err)
	}
	config.MaxProtocolVersion = "3.2"

	for _, c := range []struct {
		name         string
		keyLength    int
		wantAttempts int
	}{
		{"missing", 0, 0},
		{"maximum", 256, 1},
		{"oversized", 257, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {

				config := config.Copy()
				attempts := 0
				config.DialFunc = func(context.Context, string, string) (net.Conn, error) {
					attempts++
					return nil, errors.New("simulated cancellation dial failure")
				}
				conn, finishCleanup := newCancellationTestConnection(t, config, nil, c.keyLength)

				time.AfterFunc(75*time.Millisecond, finishCleanup)
				retryClosedConnectionCancellation(conn)
				if attempts != c.wantAttempts {
					t.Fatalf("expected %d cancellation attempts, got %d", c.wantAttempts, attempts)
				}

			})
		})
	}

}

// TestRetryClosedConnectionCancellationTLS checks that blocked TLS shutdown
// respects cancellation request timeouts and the overall retry budget.
func TestRetryClosedConnectionCancellationTLS(t *testing.T) {

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("expected certificate key, got %v", err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		t.Fatalf("expected certificate, got %v", err)
	}
	certificate := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	config, err := pgx.ParseConfig(cancellationTestDSN)
	if err != nil {
		t.Fatalf("expected connection config, got %v", err)
	}

	for _, c := range []struct {
		name            string
		negotiation     string
		shouldRetry     bool
		sendCloseNotify bool
		limit           time.Duration
		wantRequests    int
	}{
		{"request_timeout", "direct", false, false, time.Second, 1},
		{"retry_budget", "direct", true, false, 10 * time.Second, 7},
		{"request_timeout_postgres", "postgres", false, false, time.Second, 1},
		{"retry_budget_postgres", "postgres", true, false, 10 * time.Second, 7},
		{"blocked_close_notify", "direct", true, true, 10 * time.Second, 7},
		{"blocked_close_notify_postgres", "postgres", true, true, 10 * time.Second, 7},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {

				serverTLS := &tls.Config{
					Certificates:           []tls.Certificate{certificate},
					SessionTicketsDisabled: true,
				}
				config := config.Copy()
				config.SSLNegotiation = c.negotiation
				config.TLSConfig = &tls.Config{
					InsecureSkipVerify: true,
				}
				var servers []net.Conn
				var received atomic.Int32
				config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
					client, server := net.Pipe()
					t.Cleanup(func() { _ = server.Close() })
					servers = append(servers, server)
					go func() {
						if c.negotiation == "postgres" {
							backend := pgproto3.NewBackend(server, server)
							msg, err := backend.ReceiveStartupMessage()
							if err != nil {
								t.Errorf("expected SSL request, got %v", err)
								return
							}
							if _, ok := msg.(*pgproto3.SSLRequest); !ok {
								t.Errorf("expected SSL request, got %T", msg)
								return
							}
							_, err = server.Write([]byte{'S'})
							if err != nil {
								t.Errorf("expected SSL acceptance, got %v", err)
								return
							}
						}
						peer := tls.Server(server, serverTLS)
						_, err := io.ReadFull(peer, make([]byte, 16))
						if err != nil {
							t.Errorf("expected encrypted cancellation packet, got %v", err)
							return
						}
						if !peer.ConnectionState().HandshakeComplete {
							t.Error("expected completed TLS handshake, got incomplete handshake")
							return
						}
						if c.sendCloseNotify {
							// End the client's read, then leave its close_notify write blocked.
							err = peer.CloseWrite()
							if err != nil {
								t.Errorf("expected TLS close notification, got %v", err)
								return
							}
						}
						received.Add(1)
					}()
					return client, nil
				}
				conn, finishCleanup := newCancellationTestConnection(t, config, serverTLS, 4)
				started := time.Now()
				if c.shouldRetry {
					retryClosedConnectionCancellation(conn)
				} else {
					ctx, cancel := context.WithTimeout(t.Context(), c.limit)
					err := conn.PgConn().CancelRequest(ctx)
					cancel()
					if err != nil {
						t.Fatalf("expected delivered cancellation request, got %v", err)
					}
				}
				elapsed := time.Since(started)
				synctest.Wait()
				if elapsed != c.limit {
					t.Fatalf("expected cancellation duration %v, got %v", c.limit, elapsed)
				}
				if count := received.Load(); count != int32(c.wantRequests) {
					t.Fatalf("expected %d encrypted cancellation requests, got %d", c.wantRequests, count)
				}
				for _, server := range servers {
					n, err := server.Read(make([]byte, 1))
					if err != nil {
						if !errors.Is(err, io.EOF) {
							t.Fatalf("expected closed cancellation transport, got %v", err)
						}
						continue
					}
					t.Fatalf("expected closed cancellation transport, got %d bytes", n)
				}

				time.Sleep(2 * time.Second)
				synctest.Wait()
				if count := received.Load(); count != int32(c.wantRequests) {
					t.Fatalf("expected no requests after cancellation stopped, got %d", count-int32(c.wantRequests))
				}
				finishCleanup()
				synctest.Wait()
				select {
				case <-conn.PgConn().CleanupDone():
				default:
					t.Fatal("expected connection cleanup, got pending cleanup")
				}

			})
		})
	}

}

// newCancellationTestConnection returns a closed connection whose cleanup
// remains pending until the returned function is called.
// It must be called inside a synctest bubble.
// A non-nil serverTLS enables TLS. A zero keyLength omits BackendKeyData.
func newCancellationTestConnection(t *testing.T, config *pgx.ConnConfig, serverTLS *tls.Config, keyLength int) (*pgx.Conn, context.CancelFunc) {

	t.Helper()
	client, server := net.Pipe()
	cleanupCtx, finishCleanup := context.WithCancel(t.Context())
	serverDone := make(chan struct{})
	t.Cleanup(func() {
		finishCleanup()
		_ = client.Close()
		<-serverDone
	})
	go func() {

		defer close(serverDone)
		defer server.Close()
		if serverTLS != nil {
			if config.SSLNegotiation == "postgres" {
				backend := pgproto3.NewBackend(server, server)
				msg, err := backend.ReceiveStartupMessage()
				if err != nil {
					t.Errorf("expected SSL request, got %v", err)
					return
				}
				if _, ok := msg.(*pgproto3.SSLRequest); !ok {
					t.Errorf("expected SSL request, got %T", msg)
					return
				}
				_, err = server.Write([]byte{'S'})
				if err != nil {
					t.Errorf("expected SSL acceptance, got %v", err)
					return
				}
			}
			server = tls.Server(server, serverTLS)
		}
		backend := pgproto3.NewBackend(server, server)
		_, err := backend.ReceiveStartupMessage()
		if err != nil {
			t.Errorf("expected startup packet, got %v", err)
			return
		}
		backend.Send(&pgproto3.AuthenticationOk{})
		if keyLength > 0 {
			backend.Send(&pgproto3.BackendKeyData{ProcessID: 1, SecretKey: make([]byte, keyLength)})
		}
		backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
		err = backend.Flush()
		if err != nil {
			t.Errorf("expected startup response, got %v", err)
			return
		}

		// Leave Terminate unread to hold CleanupDone open until the test allows it.
		<-cleanupCtx.Done()
		msg, err := backend.Receive()
		if err != nil {
			t.Errorf("expected termination packet, got %v", err)
			return
		}
		if _, ok := msg.(*pgproto3.Terminate); !ok {
			t.Errorf("expected termination packet, got %T", msg)
		}
		if serverTLS != nil {
			_, err := server.Read(make([]byte, 1))
			if err != nil {
				if !errors.Is(err, io.EOF) {
					t.Errorf("expected TLS connection close, got %v", err)
				}
				return
			}
			t.Error("expected TLS connection close, got application data")
		}

	}()

	dial := config.DialFunc
	initial := true
	config.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		if initial {
			initial = false
			return client, nil
		}
		return dial(ctx, network, address)
	}
	// Match the pool: pgx cancels its dialing context once startup completes.
	config.ConnectTimeout = time.Second
	configureCancellation(config)
	conn, err := pgx.ConnectConfig(t.Context(), config)
	if err != nil {
		t.Fatalf("expected in-memory connection, got %v", err)
	}

	// Keep fixture shutdown alive through the test's cleanup callbacks.
	closeCtx, stopClose := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(stopClose)
	closeDone := make(chan error, 1)
	go func() { closeDone <- conn.Close(closeCtx) }()
	t.Cleanup(func() {
		finishCleanup()
		synctest.Wait()
		err := <-closeDone
		if err != nil {
			t.Errorf("expected connection close, got %v", err)
		}
	})
	synctest.Wait()
	if !conn.IsClosed() {
		t.Fatal("expected closed connection, got open connection")
	}

	return conn, finishCleanup
}

// waitForCancellationQuery observes execution independently of query context.
func waitForCancellationQuery(t *testing.T, observer *pgxpool.Pool, pid uint32, sleeping bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	lastActive, lastSleeps := -1, -1
	for {
		var active, sleeps int
		row := observer.QueryRow(ctx, `SELECT count(*) FILTER (WHERE state='active'),
count(*) FILTER (WHERE wait_event='PgSleep') FROM pg_stat_activity WHERE pid=$1`, pid)
		err := row.Scan(&active, &sleeps)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) && lastActive >= 0 {
				t.Fatalf("expected sleeping=%t, got last active=%d sleeping=%d", sleeping, lastActive, lastSleeps)
			}
			t.Fatalf("expected target activity, got %v", err)
		}
		if active < 0 || active > 1 || sleeps < 0 || sleeps > 1 {
			t.Fatalf("expected bounded target activity, got %d/%d", active, sleeps)
		}
		lastActive, lastSleeps = active, sleeps
		// PostgreSQL samples state and wait_event independently.
		if sleeping && active == 1 && sleeps == 1 || !sleeping && active == 0 && sleeps == 0 {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("expected sleeping=%t, got last active=%d sleeping=%d", sleeping, active, sleeps)
		case <-ticker.C:
		}
	}
}
