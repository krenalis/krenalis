// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package db

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof handlers
	"runtime/pprof"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// traceAcquiring controls whether connection acquire/release tracing and the
// custom "dbconns" pprof profile are active.
//
// When set to true, every call to pgxpool.Acquire and pgxpool.Release is
// traced: log messages are emitted with pool statistics, and acquired
// connections are added to the "dbconns" pprof profile until they are released.
// Active holds are therefore visible in the profile and can be examined with
// standard pprof tools:
//
//	go tool pprof http://localhost:6060/debug/pprof/dbconns
//
// This makes it possible to identify connections that are held for a long time
// and to correlate them with backend PIDs, helping diagnose pool exhaustion and
// long-running queries.
//
// When set to false, all tracer methods return immediately without logging or
// updating profiles, effectively disabling any overhead introduced by tracing.
// Because this is a compile-time constant, leaving it false allows the compiler
// to optimize away the tracing code entirely.
const traceAcquiring = false

func init() {
	if traceAcquiring {
		// Start an HTTP server that exposes the default pprof endpoints on localhost:6060.
		go func() {
			// ListenAndServe returns only on error; log it to avoid silent failures.
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				log.Printf("pprof server stopped: %v", err)
			}
		}()
	}
}

type ctxKey string

// t0Key is the context key used to store the timestamp when a pool acquire starts.
const t0Key ctxKey = "pgxpoolAcquireStart"

var (
	// dbConnProfile is a custom pprof profile that tracks currently held DB connections.
	// Each acquired connection is added on acquire and removed on release.
	dbConnProfile = pprof.NewProfile("dbconns")

	// holdMu guards holdsByConn.
	holdMu sync.Mutex

	// holdsByConn maps the underlying *pgx.Conn to the active hold record
	// so we can remove the correct profile entry on release.
	holdsByConn = make(map[*pgx.Conn]*connHold)
)

// connHold describes a single connection being held by the application.
type connHold struct {
	conn    *pgx.Conn
	pid     uint32
	started time.Time
}

// String returns a human-friendly description of how long the connection
// has been held along with its backend PID.
func (h *connHold) String() string {
	return fmt.Sprintf("pid=%d held_for=%s", h.pid, time.Since(h.started).Round(time.Millisecond))
}

// acquireTracer implements both pgx.QueryTracer (no-op) and the pool hooks
// pgxpool.AcquireTracer and pgxpool.ReleaseTracer. Assign an instance of this
// to cfg.ConnConfig.Tracer so the pool can discover the extra interfaces.
type acquireTracer struct{}

// TraceQueryStart implements pgx.QueryTracer. This is a no-op because
// we only care about connection acquire/release in this tracer.
func (acquireTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return ctx
}

// TraceQueryEnd implements pgx.QueryTracer. This is a no-op.
func (acquireTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {}

// TraceAcquireStart implements pgxpool.AcquireTracer. It records the start time
// in the context so we can compute how long the acquire took.
func (acquireTracer) TraceAcquireStart(ctx context.Context, pool *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	log.Print("[pgxpool] try acquire")
	return context.WithValue(ctx, t0Key, time.Now())
}

// TraceAcquireEnd implements pgxpool.AcquireTracer. It logs the wait time and
// pool stats; if a connection was acquired, it adds an entry to the custom
// "dbconns" pprof profile so you can inspect long-held connections.
func (acquireTracer) TraceAcquireEnd(ctx context.Context, pool *pgxpool.Pool, data pgxpool.TraceAcquireEndData) {
	wait := time.Duration(0)
	if t0, ok := ctx.Value(t0Key).(time.Time); ok {
		wait = time.Since(t0)
	}

	if data.Err != nil {
		log.Printf("[pgxpool] acquire FAILED after %s: %v", wait, data.Err)
		return
	}

	var pid uint32
	if data.Conn != nil && data.Conn.PgConn() != nil {
		pid = data.Conn.PgConn().PID()
	}

	// Take a snapshot of pool statistics for logging.
	s := pool.Stat()
	log.Printf("[pgxpool] acquired pid=%d waited=%s acquired=%d idle=%d total=%d",
		pid, wait, s.AcquiredConns(), s.IdleConns(), s.TotalConns())

	// Track the hold in our map and pprof profile so we can see “live” holds.
	if data.Conn != nil {
		h := &connHold{
			conn:    data.Conn,
			pid:     pid,
			started: time.Now(),
		}
		holdMu.Lock()
		holdsByConn[data.Conn] = h
		holdMu.Unlock()

		// Add to the custom pprof profile with a weight of 1.
		dbConnProfile.Add(h, 1)
	}
}

// TraceRelease implements pgxpool.ReleaseTracer. It logs pool stats and removes
// the connection from the custom pprof profile to reflect that it is no longer held.
func (acquireTracer) TraceRelease(pool *pgxpool.Pool, data pgxpool.TraceReleaseData) {
	var pid uint32
	if data.Conn != nil && data.Conn.PgConn() != nil {
		pid = data.Conn.PgConn().PID()
	}

	s := pool.Stat()
	log.Printf("[pgxpool] release pid=%d acquired=%d idle=%d total=%d",
		pid, s.AcquiredConns(), s.IdleConns(), s.TotalConns())

	// Remove the hold (if any) and drop it from the pprof profile.
	if data.Conn != nil {
		holdMu.Lock()
		h := holdsByConn[data.Conn]
		delete(holdsByConn, data.Conn)
		holdMu.Unlock()
		if h != nil {
			dbConnProfile.Remove(h)
		}
	}
}
