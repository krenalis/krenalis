// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/errors"

	"github.com/krenalis/analytics-go"
)

// keepNotifications owns the runtime session until the worker stops.
func (state *State) keepNotifications(session *notificationSession) {

	defer state.close.Done()
	var client analytics.Client
	if state.sendStats {
		client, _ = analytics.NewWithConfig("eEC2uyWaJ1XmFNEq0dkH0a872GzZChUV", analytics.Config{
			Endpoint: "https://telemetry.krenalis.com/v1/events",
			Logger:   discardLogger{}, // comment this line to debug sending of analytics data.
		})
		defer func() {
			err := client.Close()
			if err != nil {
				slog.Error("error while closing analytics.Client", "error", err)
			}
		}()
	}

	defer session.close()

	err := state.runNotifications(state.close.ctx, session, client)
	if err != nil {
		// Step 6 will decide process termination here, before cleanup or logging.
		slog.Error("core/state: notification replication stopped", "error", err)
	}

}

// replay applies rows progressively from the State cursor. The session owns
// active rows, so terminal errors reach its owner before cleanup and panics
// unwind to the same owner without draining the query first.
func (state *State) replay(ctx context.Context, session *notificationSession, targetVersion int, client analytics.Client) error {

	const query = "SELECT version, name, payload FROM notifications WHERE version > $1 ORDER BY version"
	rows, err := session.conn.Query(ctx, query, state.Version())
	if err != nil {
		return err
	}

	session.rows = rows
	for ctx.Err() == nil && rows.Next() {

		var n notification
		err = rows.Scan(&n.Version, &n.Name, &n.Payload)
		if err != nil {
			break
		}

		// Reading the row may have completed after lifecycle cancellation.
		err = ctx.Err()
		if err != nil {
			break
		}

		expectedVersion := state.Version() + 1
		if n.Version != expectedVersion {
			return &replicationError{
				message: fmt.Sprintf("expected notification version %d, got %d", expectedVersion, n.Version),
			}
		}

		err = state.applyNotification(n, client)
		if err != nil {
			return err
		}

	}

	closeErr := rows.Close()
	session.rows = nil
	if err != nil {
		return err
	}

	if closeErr != nil {
		return closeErr
	}

	err = ctx.Err()
	if err != nil {
		return err
	}

	if state.Version() < targetVersion {
		return &replicationError{message: fmt.Sprintf("notification log ended before version %d", targetVersion)}
	}

	return nil
}

// runNotifications serializes reception, application and recovery on one
// session. A terminal result returns before the owner's session cleanup.
func (state *State) runNotifications(ctx context.Context, session *notificationSession, client analytics.Client) *replicationError {

	const replayInterval = 5 * time.Second
	nextReplayAt := time.Now().Add(replayInterval)
	bo := backoff.New(10)
	bo.SetCap(5 * time.Second)
	replayNeeded := false
	targetVersion := 0
	var err error

	for {

		if err != nil {

			if terminal, ok := errors.AsType[*replicationError](err); ok {
				return terminal
			}

			if ctx.Err() != nil {
				return nil
			}

			if session.conn != nil && session.conn.Underlying().IsClosed() {
				session.close()
			}

			slog.Warn("core/state: notification recovery failed; retrying", "error", err)
			if !bo.Next(ctx) {
				return nil
			}

			err = nil
			replayNeeded = true

		}

		if ctx.Err() != nil {
			return nil
		}

		if session.conn == nil {
			session.conn, err = state.notifications.connect(ctx)
			if err != nil {
				continue
			}
			replayNeeded = true
		}

		if replayNeeded || !time.Now().Before(nextReplayAt) {

			err = state.replay(ctx, session, targetVersion, client)
			if err != nil {
				continue
			}

			nextReplayAt = time.Now().Add(replayInterval)
			replayNeeded = false
			targetVersion = 0
			bo.Reset()

		}

		if ctx.Err() != nil {
			return nil
		}

		waitCtx, cancel := context.WithDeadline(ctx, nextReplayAt)
		raw, waitErr := session.conn.Underlying().WaitForNotification(waitCtx)
		cancel()
		if waitErr != nil {
			replayNeeded = true
			if errors.Is(waitErr, context.DeadlineExceeded) && ctx.Err() == nil {
				continue
			}
			err = waitErr
			continue
		}

		if raw.Channel != "krenalis" {
			continue
		}

		var n *notification
		n, err = session.reconstruct(ctx, state.notifications.key, raw.Payload)
		if err != nil {
			// Reconcile persisted events only; lost elections cannot be replayed.
			replayNeeded = true
			continue
		}

		if n == nil {
			continue
		}

		if ctx.Err() != nil {
			return nil
		}

		if n.Version == 0 {
			err = state.applyNotification(*n, client)
			continue
		}

		version := state.Version()
		if n.Version <= version {
			continue
		}

		if n.Version == version+1 {
			err = state.applyNotification(*n, client)
			continue
		}

		targetVersion = n.Version
		replayNeeded = true

	}

}

// replicationError identifies a confirmed gap or unknown trusted event.
type replicationError struct {
	message string
}

// Error returns a diagnostic containing only protocol metadata.
func (err *replicationError) Error() string {
	return "core/state: " + err.message
}
