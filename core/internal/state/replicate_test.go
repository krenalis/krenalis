// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	stateTestWorkspaceID        = "6NpT4zB8QaR2"
	stateTestOrganizationID     = "K8mR3vP2xQ6a"
	stateTestConnectionID       = "V2qH7mA4rN9x"
	stateTestPipelineID         = "B9xK3mQ7vA2r"
	stateTestMemberID           = "F4pN8zR2hV6q"
	stateTestConsentPurposeID   = "D7hV4xK9mP2a"
	stateTestBarrierPurposeID   = "5Qm8R2vJ9aK3"
	stateTestUnappliedPurposeID = "A7v4N9q2K6mP"
)

// cancelingInt8Codec cancels the test lifecycle during real row conversion.
type cancelingInt8Codec struct {
	pgtype.Int8Codec
	cancel context.CancelFunc
}

// PlanScan wraps the driver's normal conversion without replacing query I/O.
func (codec cancelingInt8Codec) PlanScan(m *pgtype.Map, oid uint32, format int16, target any) pgtype.ScanPlan {
	plan := codec.Int8Codec.PlanScan(m, oid, format, target)
	if plan == nil {
		return nil
	}
	return cancelingScanPlan{ScanPlan: plan, cancel: codec.cancel}
}

type cancelingScanPlan struct {
	pgtype.ScanPlan
	cancel context.CancelFunc
}

// Scan cancels after converting a value and preserves the driver's result.
func (plan cancelingScanPlan) Scan(src []byte, target any) error {
	err := plan.ScanPlan.Scan(src, target)
	plan.cancel()
	return err
}

// TestApplyElections preserves election dispatch and heartbeat semantics while
// keeping both event types outside version publication and acknowledgements.
func TestApplyElections(t *testing.T) {

	state := &State{mu: &sync.Mutex{}, changing: &sync.RWMutex{}}
	state.version.current = 123
	ack := make(chan struct{}, 1)
	state.notifications.acks.Store(0, ack)
	calls := 0
	state.listeners = []any{func(ElectLeader) { calls++ }}
	for range 2 {
		err := state.applyNotification(notification{0, "ElectLeader", `{"Number":7,"Leader":"leader"}`}, nil)
		if err != nil {
			t.Fatalf("expected election application, got %v", err)
		}
	}

	previous := time.Unix(1, 0)
	state.election.lastSeen = previous
	err := state.applyNotification(notification{0, "SeeLeader", `{"Election":6}`}, nil)
	if err != nil {
		t.Fatalf("expected outdated heartbeat handling, got %v", err)
	}

	if state.election.lastSeen != previous {
		t.Fatalf("expected outdated heartbeat ignored, got %v", state.election.lastSeen)
	}

	err = state.applyNotification(notification{0, "SeeLeader", `{"Election":7}`}, nil)
	if err != nil {
		t.Fatalf("expected current heartbeat handling, got %v", err)
	}

	if state.election.lastSeen == previous || calls != 1 || state.Version() != 123 || len(ack) != 0 {
		t.Fatalf("expected current heartbeat and one election dispatch without version/ack, got election=%+v calls=%d version=%d acks=%d", state.election, calls, state.Version(), len(ack))
	}

}

// TestApplyUnknownEvent checks terminal classification before publication,
// including that the condition variable does not wake a waiting observer.
func TestApplyUnknownEvent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		state := &State{changing: &sync.RWMutex{}}
		state.version.current = 123
		state.version.next = sync.Cond{L: &state.version.RWMutex}
		ack := make(chan struct{}, 1)
		state.notifications.acks.Store(124, ack)
		woke := false
		go func() { state.version.Lock(); state.version.next.Wait(); woke = true; state.version.Unlock() }()
		synctest.Wait()
		err := state.applyNotification(notification{124, "Unknown", `{}`}, nil)
		func() {
			if err != nil {
				if _, terminal := errors.AsType[*replicationError](err); !terminal {
					t.Fatalf("expected terminal unknown event, got %v", err)
				}
				return
			}
			t.Fatal("expected terminal unknown event, got success")
		}()
		synctest.Wait()
		if woke || state.Version() != 123 || len(ack) != 0 {
			t.Fatalf("expected no Broadcast/version/ack, got woke=%t version=%d acks=%d", woke, state.Version(), len(ack))
		}

		state.version.Lock()
		state.version.next.Broadcast()
		state.version.Unlock()
		synctest.Wait()

	})
}

// TestNotificationCoalescence documents PostgreSQL's coalescence of identical
// fragments in one transaction, and that an invalid reconstruction resets.
func TestNotificationCoalescence(t *testing.T) {

	database, key := replicationDatabase(t)
	_, session := replicationState(t, database, key)
	defer session.close()
	_, err := database.Exec(t.Context(), "BEGIN; NOTIFY krenalis, 'same*'; NOTIFY krenalis, 'same*'; NOTIFY krenalis, 'end@124'; COMMIT")
	if err != nil {
		t.Fatalf("expected committed repeated fragments, got %v", err)
	}

	var frames []string
	for range 2 {
		raw, err := session.conn.Underlying().WaitForNotification(t.Context())
		if err != nil {
			t.Fatalf("expected received fragment, got %v", err)
		}
		frames = append(frames, raw.Payload)
	}

	if strings.Join(frames, ",") != "same*,end@124" {
		t.Fatalf("expected identical fragments coalesced by PostgreSQL, got %v", frames)
	}

	n, err := session.reconstruct(t.Context(), key, frames[0])
	if err != nil {
		t.Fatalf("expected incomplete fragment, got %v", err)
	}

	if n != nil {
		t.Fatalf("expected no application of partial message, got %v", n)
	}

	n, err = session.reconstruct(t.Context(), key, frames[1])
	if err != nil {
		if n != nil || session.fragments.Len() != 0 {
			t.Fatalf("expected rejected message and reset buffer, got event=%v length=%d", n, session.fragments.Len())
		}
		return
	}

	t.Fatal("expected failed reconstruction, got success")

}

// TestReplicationCanceledBeforeApply leaves even an already-buffered event
// unapplied when the lifecycle has been canceled.
func TestReplicationCanceledBeforeApply(t *testing.T) {

	database, key := replicationDatabase(t)
	state, session := replicationState(t, database, key)
	sendReplicationEvent(t, database, key, 124, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestConsentPurposeID})
	_, err := session.conn.Exec(t.Context(), "SELECT 1")
	if err != nil {
		t.Fatalf("expected notification buffering query, got %v", err)
	}

	state.close.cancel()
	terminal := state.runNotifications(state.close.ctx, session, nil)
	if terminal != nil || state.Version() != 123 || len(state.workspaces[stateTestWorkspaceID].consentPurposes) != 0 {
		t.Fatalf("expected cancellation before application, got terminal=%v version=%d", terminal, state.Version())
	}

}

// TestReplicationCancellationDuringApply verifies that cancellation does not
// release replay rows or the dedicated connection until a blocked listener
// completes.
func TestReplicationCancellationDuringApply(t *testing.T) {

	database, key := replicationDatabase(t)
	state, session := replicationState(t, database, key)
	ownedConn := session.conn
	cleanupDone := ownedConn.Underlying().PgConn().CleanupDone()
	entered, release := make(chan struct{}), make(chan struct{})
	releaseDispatch := sync.OnceFunc(func() { close(release) })
	defer releaseDispatch()
	state.listeners = []any{func(AddConsentPurpose) { close(entered); <-release }}
	ack := make(chan struct{}, 1)
	state.notifications.acks.Store(124, ack)
	logReplicationEvent(t, database, 124, "replayed")
	startReplication(t, state, session)
	// The future notification forces replay of version 124. Its listener blocks
	// during dispatch while the session still owns the replay rows and connection.
	sendReplicationEvent(t, database, key, 125, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestBarrierPurposeID})
	select {
	case <-entered:
	case <-state.close.ctx.Done():
		t.Fatalf("expected started dispatch, got %v", state.close.ctx.Err())
	}

	state.close.cancel()
	completed := make(chan struct{})
	go func() { state.close.Wait(); close(completed) }()
	waitReplicationBlocked(t, "dispatchNotification[", completed)
	if session.conn != ownedConn {
		t.Fatal("expected retained replay connection during dispatch, got different session connection")
	}
	if session.rows == nil {
		t.Fatal("expected retained replay rows during dispatch, got nil rows")
	}
	if ownedConn.Underlying().IsClosed() {
		t.Fatal("expected open replay connection during dispatch, got closed connection")
	}
	if !ownedConn.Underlying().PgConn().IsBusy() {
		t.Fatal("expected busy replay connection during dispatch, got idle connection")
	}
	if got := database.PoolStats().AcquiredConns(); got != 1 {
		t.Fatalf("expected 1 acquired connection during dispatch, got %d", got)
	}

	select {
	case <-cleanupDone:
		t.Fatal("expected pending connection cleanup during blocked listener, got completed cleanup")
	default:
	}

	select {
	case <-completed:
		t.Fatal("expected pending worker cleanup during blocked listener, got completed worker")
	default:
	}

	if got := state.Version(); got != 123 {
		t.Fatalf("expected version 123 during dispatch, got %d", got)
	}

	releaseDispatch()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case <-completed:
	case <-ctx.Done():
		t.Fatalf("expected cleanup after listener completion, got %v", ctx.Err())
	}

	if session.conn != nil {
		t.Fatal("expected released replay connection after listener completion, got retained connection")
	}
	if session.rows != nil {
		t.Fatal("expected released replay rows after listener completion, got retained rows")
	}

	select {
	case <-cleanupDone:
	default:
		t.Fatal("expected completed physical cleanup after listener completion, got pending cleanup")
	}

	checkBootstrapConnectionsReleased(t, database, 4)
	if got := state.Version(); got != 124 {
		t.Fatalf("expected version 124 after dispatch, got %d", got)
	}
	if got := len(ack); got != 1 {
		t.Fatalf("expected 1 ack after dispatch, got %d", got)
	}
	if state.workspaces[stateTestWorkspaceID].consentPurposes[replicationConsentPurposeID(124)] == nil {
		t.Fatal("expected applied replayed consent purpose, got missing purpose")
	}

}

// TestReplicationFastPath checks direct delivery and duplicate suppression
// while any replay query would fail because the log is unavailable.
func TestReplicationFastPath(t *testing.T) {

	database, key := replicationDatabase(t)
	state, session := replicationState(t, database, key)
	pid := session.conn.Underlying().PgConn().PID()
	_, err := database.Exec(t.Context(), "DROP TABLE notifications")
	if err != nil {
		t.Fatalf("expected unavailable replay log, got %v", err)
	}

	calls := 0
	state.listeners = []any{func(AddConsentPurpose) { calls++ }}
	ack := make(chan struct{}, 1)
	state.notifications.acks.Store(124, ack)
	startReplication(t, state, session)
	sendReplicationEvent(t, database, key, 124, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestConsentPurposeID, Name: "original"})
	waitReplicationVersion(t, state, 124)
	select {
	case <-ack:
	case <-state.close.ctx.Done():
		t.Fatalf("expected first acknowledgement, got %v", state.close.ctx.Err())
	}

	duplicateAck := make(chan struct{}, 1)
	state.notifications.acks.Store(124, duplicateAck)
	sendReplicationEvent(t, database, key, 124, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestConsentPurposeID, Name: "duplicate"})
	sendReplicationEvent(t, database, key, 125, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestBarrierPurposeID})
	waitReplicationVersion(t, state, 125)
	var query string
	err = database.QueryRow(t.Context(), "SELECT query FROM pg_stat_activity WHERE pid = $1", pid).Scan(&query)
	if err != nil {
		t.Fatalf("expected listener query history, got %v", err)
	}

	if query != "LISTEN krenalis" {
		t.Fatalf("expected delivery without SQL after LISTEN, got %q", query)
	}

	state.close.cancel()
	state.close.Wait()
	if calls != 2 || len(duplicateAck) != 0 || state.workspaces[stateTestWorkspaceID].consentPurposes[stateTestConsentPurposeID].Name != "original" {
		t.Fatalf("expected two direct applications and no duplicate mutation/dispatch/ack or replay, got calls=%d acks=%d", calls, len(duplicateAck))
	}

}

// TestReplicationGapAndElections checks that replay includes the gap-signalling
// event and preserves intact election messages queued during the query.
func TestReplicationGapAndElections(t *testing.T) {

	database, key := replicationDatabase(t)
	state, session := replicationState(t, database, key)
	logReplicationEvent(t, database, 124, "first")
	logReplicationEvent(t, database, 125, "from-log")
	entered, release := make(chan struct{}), make(chan struct{})
	releaseDispatch := sync.OnceFunc(func() { close(release) })
	defer releaseDispatch()
	var seen []string
	state.listeners = []any{func(e AddConsentPurpose) {
		seen = append(seen, e.Name)
		if e.Name == "first" {
			close(entered)
			<-release
		}
	}}
	electionAck := make(chan struct{}, 1)
	state.notifications.acks.Store(0, electionAck)
	startReplication(t, state, session)
	sendReplicationEvent(t, database, key, 125, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestUnappliedPurposeID, Name: "must-not-apply"})
	select {
	case <-entered:
	case <-state.close.ctx.Done():
		t.Fatalf("expected replay dispatch, got %v", state.close.ctx.Err())
	}

	sendReplicationEvent(t, database, key, 0, ElectLeader{Number: 3, Leader: "leader"})
	sendReplicationEvent(t, database, key, 0, SeeLeader{Election: 3})
	releaseDispatch()
	sendReplicationEvent(t, database, key, 126, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestBarrierPurposeID, Name: "barrier"})
	waitReplicationVersion(t, state, 126)
	state.close.cancel()
	state.close.Wait()
	if strings.Join(seen, ",") != "first,from-log,barrier" || state.election.number != 3 || state.election.leader != "leader" || len(electionAck) != 0 {
		t.Fatalf("expected replay once and queued version-zero elections, got %v election=%+v acks=%d", seen, state.election, len(electionAck))
	}

}

// TestReplicationPendingRows checks that a terminal decision and panic are not
// held behind pgx draining a later row blocked on a real PostgreSQL lock.
func TestReplicationPendingRows(t *testing.T) {
	for _, mode := range []string{"panic", "gap", "unknown"} {
		t.Run(mode, func(t *testing.T) {

			database, key := replicationDatabase(t)
			state, session := replicationState(t, database, key)
			blocker, err := database.Conn(t.Context())
			if err != nil {
				t.Fatalf("expected lock owner, got %v", err)
			}

			defer closeNotificationConnection(blocker)
			_, err = blocker.Exec(t.Context(), "SELECT pg_advisory_lock(42)")
			if err != nil {
				t.Fatalf("expected held row barrier, got %v", err)
			}

			firstVersion := 124
			if mode == "gap" {
				firstVersion = 125
			}

			eventName := "AddConsentPurpose"
			if mode == "unknown" {
				eventName = "Unknown"
			}

			_, err = database.Exec(t.Context(), `
    INSERT INTO notifications VALUES ($1, $2, jsonb_build_array(repeat('x', 32768))), ($1 + 1, 'AddConsentPurpose', '{}'::jsonb);
   `, firstVersion, eventName)
			if err != nil {
				t.Fatalf("expected pending result fixture, got %v", err)
			}

			_, err = database.Exec(t.Context(), fmt.Sprintf(`
    ALTER TABLE notifications RENAME TO pending_log;
    CREATE FUNCTION pending_payload(v bigint, p jsonb) RETURNS jsonb LANGUAGE plpgsql STABLE AS $$
    BEGIN IF v > %d THEN PERFORM pg_advisory_xact_lock(42); END IF; RETURN p; END $$;
    CREATE VIEW notifications AS SELECT version, name, pending_payload(version, payload) AS payload FROM pending_log;
   `, firstVersion))
			if err != nil {
				t.Fatalf("expected blocked result view, got %v", err)
			}

			_, err = session.conn.Exec(t.Context(), "SET enable_seqscan = off; SET enable_sort = off")
			if err != nil {
				t.Fatalf("expected streaming index scan, got %v", err)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			if mode == "panic" {
				func() {
					defer func() {
						if got := recover(); got != "invalid notification payload AddConsentPurpose (version 124)" {
							t.Errorf("expected original trusted panic, got %v", got)
						}
					}()
					defer session.close()
					_ = state.replay(ctx, session, 0, nil)
					t.Error("expected synchronous trusted panic, got normal return")
				}()
			} else {

				sendReplicationEvent(t, database, key, 130, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestConsentPurposeID})
				terminal := state.runNotifications(ctx, session, nil)
				if terminal == nil || session.rows == nil || !session.conn.Underlying().PgConn().IsBusy() {
					t.Fatalf("expected terminal observed before cleanup of pending query, got terminal=%v rows=%v", terminal, session.rows)
				}

				if state.Version() != 123 {
					t.Fatalf("expected unchanged cursor before terminal, got %d", state.Version())
				}

				session.close()

			}

			err = ctx.Err()
			if err != nil {
				t.Fatalf("expected decision and cleanup independent of guard timeout or row unlock, got %v", err)
			}

			checkBootstrapConnectionsReleased(t, database, 3)

		})
	}
}

// TestReplicationPeriodic checks lost tail delivery, continuous heartbeat
// traffic and fragment preservation across a periodic replay on the same PID.
func TestReplicationPeriodic(t *testing.T) {
	for _, mode := range []string{"quiet", "heartbeats", "partial-fragment"} {
		t.Run(mode, func(t *testing.T) {

			database, key := replicationDatabase(t)
			state, session := replicationState(t, database, key)
			pid := session.conn.Underlying().PgConn().PID()
			logReplicationEvent(t, database, 124, "lost-tail")
			var finalFragment string
			if mode == "partial-fragment" {
				b, err := appendEncodeNotification(t.Context(), nil, key, "AddConsentPurpose", AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestConsentPurposeID, Name: "complete"})
				if err != nil {
					t.Fatalf("expected encoded notification, got %v", err)
				}
				sendReplicationFragment(t, database, string(b[:len(b)/2])+"*")
				finalFragment = string(b[len(b)/2:]) + "@125"
			}

			var heartbeatDone chan error
			heartbeatCount := 0
			heartbeatCtx, stopHeartbeats := context.WithCancel(t.Context())
			if mode == "heartbeats" {
				sendReplicationEvent(t, database, key, 0, ElectLeader{Number: 1, Leader: "leader"})
				heartbeatDone = make(chan error, 1)
				go func() {

					producer := &State{db: database}
					producer.notifications.notifier = &notifier{db: database, key: key}
					for heartbeatCtx.Err() == nil {
						err := producer.Transaction(t.Context(), func(tx *db.Tx) (any, error) { return SeeLeader{Election: 1}, nil })
						if err != nil {
							heartbeatDone <- err
							return
						}
						heartbeatCount++
					}

					heartbeatDone <- nil

				}()
			}

			defer func() {
				stopHeartbeats()
				if heartbeatDone != nil {
					<-heartbeatDone
				}
			}()
			startReplication(t, state, session)
			waitReplicationVersion(t, state, 124)
			stopHeartbeats()
			if heartbeatDone != nil {
				err := <-heartbeatDone
				heartbeatDone = nil
				if err != nil {
					t.Fatalf("expected continuous heartbeat production until periodic replay, got %v", err)
				}
				if heartbeatCount < 2 {
					t.Fatalf("expected repeated heartbeat traffic, got %d messages", heartbeatCount)
				}
			}

			if finalFragment != "" {
				sendReplicationFragment(t, database, finalFragment)
			} else {
				sendReplicationEvent(t, database, key, 125, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestBarrierPurposeID})
			}

			waitReplicationVersion(t, state, 125)
			var samePID bool
			err := database.QueryRow(t.Context(), "SELECT EXISTS (SELECT FROM pg_stat_activity WHERE pid = $1 AND query LIKE 'SELECT version, name, payload FROM notifications%')", pid).Scan(&samePID)
			if err != nil {
				t.Fatalf("expected replay session lookup, got %v", err)
			}

			if !samePID {
				t.Fatal("expected replay and continued reception on original PID, got replacement")
			}

		})
	}
}

// TestReplicationReconnect checks the actual recovery loop, including effective
// LISTEN before replay, recovery without new notifications and lifecycle exit.
func TestReplicationReconnect(t *testing.T) {

	database, key := replicationDatabase(t)
	state, session := replicationState(t, database, key)
	oldPID := session.conn.Underlying().PgConn().PID()
	cleanupDone := session.conn.Underlying().PgConn().CleanupDone()
	logReplicationEvent(t, database, 124, "missed")
	_, err := database.Exec(t.Context(), `
  ALTER TABLE notifications RENAME TO reconnect_log;
  CREATE FUNCTION listening_payload(p jsonb) RETURNS jsonb LANGUAGE plpgsql STABLE AS $$
  BEGIN
   IF NOT EXISTS (SELECT FROM pg_listening_channels() WHERE pg_listening_channels = 'krenalis') THEN
    RAISE EXCEPTION 'replay before effective LISTEN';
   END IF;
   RETURN p;
  END $$;
  CREATE VIEW notifications AS SELECT version, name, listening_payload(payload) AS payload FROM reconnect_log;
 `)
	if err != nil {
		t.Fatalf("expected reconnect LISTEN guard, got %v", err)
	}

	_, err = database.Exec(t.Context(), "SELECT pg_terminate_backend($1)", oldPID)
	if err != nil {
		t.Fatalf("expected terminated old backend, got %v", err)
	}

	startReplication(t, state, session)
	waitReplicationVersion(t, state, 124)
	select {
	case <-cleanupDone:
	default:
		t.Fatal("expected old physical cleanup before recovered delivery, got pending cleanup")
	}

	state.close.cancel()
	state.close.Wait()
	checkBootstrapConnectionsReleased(t, database, 4)

}

// TestReplicationReconnectCancellation ensures lifecycle cancellation cannot
// turn an unusable session into a replacement acquisition.
func TestReplicationReconnectCancellation(t *testing.T) {

	database, key := replicationDatabase(t)
	state, session := replicationState(t, database, key)
	cleanupDone := session.conn.Underlying().PgConn().CleanupDone()
	err := session.conn.Underlying().Close(t.Context())
	if err != nil {
		t.Fatalf("expected closed old session, got %v", err)
	}

	state.close.cancel()
	acquired := database.PoolStats().AcquireCount()
	terminal := state.runNotifications(state.close.ctx, session, nil)
	session.close()
	if terminal != nil || database.PoolStats().AcquireCount() != acquired {
		t.Fatalf("expected cancellation without replacement, got terminal=%v acquisitions=%d", terminal, database.PoolStats().AcquireCount()-acquired)
	}

	select {
	case <-cleanupDone:
	default:
		t.Fatal("expected physically closed abandoned session, got pending cleanup")
	}

}

// TestReplicationReconstructionFailure checks buffer reset and reconciliation
// without applying corrupt data or discarding subsequent intact elections.
func TestReplicationReconstructionFailure(t *testing.T) {
	database, key := replicationDatabase(t)
	for _, failure := range []string{"base64", "decrypt", "gzip"} {
		t.Run(failure, func(t *testing.T) {

			_, err := database.Exec(t.Context(), "TRUNCATE notifications")
			if err != nil {
				t.Fatalf("expected empty fixture log, got %v", err)
			}

			state, session := replicationState(t, database, key)
			logReplicationEvent(t, database, 124, "persisted")
			switch failure {
			case "base64":
				sendReplicationFragment(t, database, "invalid!*")
				sendReplicationFragment(t, database, "final@124")
			case "decrypt":
				sendReplicationFragment(t, database, base64.RawStdEncoding.EncodeToString(make([]byte, 64))+"@124")
			case "gzip":
				encrypted, err := key.Encrypt(t.Context(), []byte("not gzip data"))
				if err != nil {
					t.Fatalf("expected encrypted invalid gzip, got %v", err)
				}
				sendReplicationFragment(t, database, base64.RawStdEncoding.EncodeToString(encrypted)+"@124")
			}

			sendReplicationEvent(t, database, key, 0, ElectLeader{Number: 7, Leader: "after-corruption"})
			sendReplicationEvent(t, database, key, 125, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestBarrierPurposeID})
			calls := 0
			state.listeners = []any{func(AddConsentPurpose) { calls++ }}
			startReplication(t, state, session)
			waitReplicationVersion(t, state, 125)
			state.close.cancel()
			state.close.Wait()
			if calls != 2 || state.election.number != 7 || state.election.leader != "after-corruption" {
				t.Fatalf("expected persisted reconciliation and intact election, got calls=%d election=%+v", calls, state.election)
			}

		})
	}
}

// TestReplicationReplay checks progressive application and distinguishes a
// failed read from internal gaps and a missing target after successful EOF.
func TestReplicationReplay(t *testing.T) {
	database, key := replicationDatabase(t)
	t.Run("partial-error-and-retry", func(t *testing.T) {

		state, session := replicationState(t, database, key)
		defer session.close()
		logReplicationEvent(t, database, 124, strings.Repeat("x", 32768))
		logReplicationEvent(t, database, 125, "second")
		_, err := database.Exec(t.Context(), `
   ALTER TABLE notifications RENAME TO replay_log;
   CREATE FUNCTION replay_payload(v bigint, p jsonb) RETURNS jsonb LANGUAGE plpgsql STABLE AS $$
   BEGIN IF v = 125 THEN RAISE EXCEPTION 'test interrupted read'; END IF; RETURN p; END $$;
   CREATE VIEW notifications AS SELECT version, name, replay_payload(version, payload) AS payload FROM replay_log;
  `)
		if err != nil {
			t.Fatalf("expected failing replay fixture, got %v", err)
		}

		_, err = session.conn.Exec(t.Context(), "SET enable_seqscan = off; SET enable_sort = off")
		if err != nil {
			t.Fatalf("expected ordered index scan, got %v", err)
		}

		calls := 0
		state.listeners = []any{func(AddConsentPurpose) { calls++ }}
		firstAck := make(chan struct{}, 1)
		state.notifications.acks.Store(124, firstAck)
		err = state.replay(t.Context(), session, 130, nil)
		func() {
			if err != nil {
				if _, terminal := errors.AsType[*replicationError](err); terminal {
					t.Fatalf("expected recoverable read error despite missing target, got %v", err)
				}
				return
			}
			t.Fatal("expected interrupted query, got success")
		}()
		if state.Version() != 124 || calls != 1 || len(firstAck) != 1 {
			t.Fatalf("expected applied prefix 124 acknowledged once, got version=%d calls=%d acks=%d", state.Version(), calls, len(firstAck))
		}

		_, err = database.Exec(t.Context(), `CREATE OR REPLACE FUNCTION replay_payload(v bigint, p jsonb) RETURNS jsonb LANGUAGE plpgsql STABLE AS $$ BEGIN RETURN p; END $$`)
		if err != nil {
			t.Fatalf("expected repaired test query, got %v", err)
		}

		duplicateAck, secondAck := make(chan struct{}, 1), make(chan struct{}, 1)
		state.notifications.acks.Store(124, duplicateAck)
		state.notifications.acks.Store(125, secondAck)
		err = state.replay(t.Context(), session, 125, nil)
		if err != nil {
			t.Fatalf("expected retry from applied 124, got %v", err)
		}

		if state.Version() != 125 || calls != 2 || len(duplicateAck) != 0 || len(secondAck) != 1 {
			t.Fatalf("expected version 125 without duplicate application or acknowledgement, got version=%d calls=%d duplicate=%d second=%d", state.Version(), calls, len(duplicateAck), len(secondAck))
		}

		err = state.replay(t.Context(), session, 130, nil)
		func() {
			if err != nil {
				if _, terminal := errors.AsType[*replicationError](err); !terminal || session.rows != nil {
					t.Fatalf("expected missing target only after closed successful EOF, got %v rows=%v", err, session.rows)
				}
				return
			}
			t.Fatal("expected terminal missing target, got success")
		}()
		_, err = database.Exec(t.Context(), "DROP VIEW notifications; ALTER TABLE replay_log RENAME TO notifications; TRUNCATE notifications")
		if err != nil {
			t.Fatalf("expected restored fixture, got %v", err)
		}

	})
	t.Run("cancel-after-prefix", func(t *testing.T) {

		state, session := replicationState(t, database, key)
		logReplicationEvent(t, database, 124, "first")
		logReplicationEvent(t, database, 125, "second")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		state.listeners = []any{func(AddConsentPurpose) { cancel() }}
		err := state.replay(ctx, session, 125, nil)
		func() {
			if err != nil {
				if _, terminal := errors.AsType[*replicationError](err); terminal {
					t.Fatalf("expected cancellation without evidence of missing target, got %v", err)
				}
				return
			}
			t.Fatal("expected canceled replay, got success")
		}()
		if state.Version() != 124 {
			t.Fatalf("expected completed prefix despite cancellation, got %d", state.Version())
		}

		session.close()
		_, err = database.Exec(t.Context(), "TRUNCATE notifications")
		if err != nil {
			t.Fatalf("expected restored fixture, got %v", err)
		}

	})
	t.Run("cancel-during-row-conversion", func(t *testing.T) {

		state, session := replicationState(t, database, key)
		defer session.close()
		logReplicationEvent(t, database, 124, "canceled")
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		session.conn.Underlying().TypeMap().RegisterType(&pgtype.Type{
			Name: "int8", OID: pgtype.Int8OID, Codec: cancelingInt8Codec{cancel: cancel},
		})
		ack := make(chan struct{}, 1)
		state.notifications.acks.Store(124, ack)
		err := state.replay(ctx, session, 124, nil)
		func() {
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("expected lifecycle cancellation, got %v", err)
				}
				return
			}
			t.Fatal("expected cancellation before application, got success")
		}()
		if state.Version() != 123 || len(ack) != 0 || len(state.workspaces[stateTestWorkspaceID].consentPurposes) != 0 {
			t.Fatalf("expected no application or acknowledgement after cancellation, got version=%d acks=%d", state.Version(), len(ack))
		}

		_, err = database.Exec(t.Context(), "TRUNCATE notifications")
		if err != nil {
			t.Fatalf("expected restored fixture, got %v", err)
		}

	})
	t.Run("internal-gap", func(t *testing.T) {
		state, session := replicationState(t, database, key)
		defer session.close()
		logReplicationEvent(t, database, 125, "gap")
		err := state.replay(t.Context(), session, 0, nil)
		func() {
			if err != nil {
				if _, terminal := errors.AsType[*replicationError](err); !terminal {
					t.Fatalf("expected terminal gap, got %v", err)
				}
				return
			}
			t.Fatal("expected terminal gap, got success")
		}()
		if state.Version() != 123 || session.rows == nil {
			t.Fatalf("expected unchanged cursor and rows owned until decision, got version=%d rows=%v", state.Version(), session.rows)
		}
	})
}

// TestReplicationReplaySnapshot leaves a notification committed during replay
// for the following loop, without using it as a target of the older snapshot.
func TestReplicationReplaySnapshot(t *testing.T) {

	database, key := replicationDatabase(t)
	state, session := replicationState(t, database, key)
	logReplicationEvent(t, database, 124, "snapshot")
	var seen []string
	state.listeners = []any{func(e AddConsentPurpose) {
		seen = append(seen, e.Name)
		if e.Name == "snapshot" {
			logReplicationEvent(t, database, 125, "after-snapshot")
			sendReplicationEvent(t, database, key, 125, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: replicationConsentPurposeID(125), Name: "after-snapshot"})
		}
	}}
	err := state.replay(t.Context(), session, 0, nil)
	if err != nil {
		t.Fatalf("expected successful replay of the original snapshot, got %v", err)
	}

	if state.Version() != 124 {
		t.Fatalf("expected snapshot ending at version 124, got %d", state.Version())
	}

	startReplication(t, state, session)
	waitReplicationVersion(t, state, 125)
	state.close.cancel()
	state.close.Wait()
	if strings.Join(seen, ",") != "snapshot,after-snapshot" {
		t.Fatalf("expected each event applied once across snapshots, got %v", seen)
	}

}

// TestReplicationUnknownRuntime checks that the loop returns the terminal
// without replay, replacement acquisition or waiting for cancellation.
func TestReplicationUnknownRuntime(t *testing.T) {

	database, key := replicationDatabase(t)
	state, session := replicationState(t, database, key)
	payload, err := appendEncodeNotification(t.Context(), nil, key, "Unknown", struct{}{})
	if err != nil {
		t.Fatalf("expected encoded unknown event, got %v", err)
	}

	sendReplicationFragment(t, database, string(payload)+"@124")
	acquired := database.PoolStats().AcquireCount()
	terminal := state.runNotifications(state.close.ctx, session, nil)
	if terminal == nil || state.close.ctx.Err() != nil || state.Version() != 123 || database.PoolStats().AcquireCount() != acquired {
		t.Fatalf("expected terminal stop without version advancement, cancellation or replacement acquisition, got terminal=%v context=%v version=%d acquisitions=%d", terminal, state.close.ctx.Err(), state.Version(), database.PoolStats().AcquireCount()-acquired)
	}

}

// TestReplicationWaitDeadline verifies real pgx deadline recovery, replay and
// subsequent reception on the original PostgreSQL backend.
func TestReplicationWaitDeadline(t *testing.T) {

	database, key := replicationDatabase(t)
	state, session := replicationState(t, database, key)
	defer session.close()
	var before int
	err := session.conn.QueryRow(t.Context(), "SELECT pg_backend_pid() FROM pg_listening_channels() WHERE pg_listening_channels = 'krenalis'").Scan(&before)
	if err != nil {
		t.Fatalf("expected effective LISTEN and PID, got %v", err)
	}

	waitCtx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	_, err = session.conn.Underlying().WaitForNotification(waitCtx)
	func() {
		if err != nil {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("expected intentional deadline, got %v", err)
			}
			return
		}
		t.Fatal("expected deadline, got notification")
	}()
	logReplicationEvent(t, database, 124, "deadline-replay")
	err = state.replay(t.Context(), session, 124, nil)
	if err != nil {
		t.Fatalf("expected replay with valid lifecycle context, got %v", err)
	}

	sendReplicationEvent(t, database, key, 0, SeeLeader{Election: 2})
	raw, err := session.conn.Underlying().WaitForNotification(t.Context())
	if err != nil {
		t.Fatalf("expected notification after deadline, got %v", err)
	}

	if raw.Channel != "krenalis" {
		t.Fatalf("expected krenalis channel, got %q", raw.Channel)
	}

	var after int
	err = session.conn.QueryRow(t.Context(), "SELECT pg_backend_pid()").Scan(&after)
	if err != nil {
		t.Fatalf("expected final PID query, got %v", err)
	}

	if after != before {
		t.Fatalf("expected unchanged PID %d, got %d", before, after)
	}

}

// TestReplicationWaitsForCleanupDone verifies that replication waits for the
// old connection to finish cleanup before acquiring a replacement.
func TestReplicationWaitsForCleanupDone(t *testing.T) {

	database, key := replicationDatabase(t)
	state, session := replicationState(t, database, key)
	proxy, cancelEntered, releaseCancel := replicationCleanupProxy(t, session.conn.Underlying().Config())
	defer releaseCancel()
	session.close()
	state.notifications.db = proxy
	conn, err := state.notifications.connect(t.Context())
	if err != nil {
		t.Fatalf("expected proxied listening session, got %v", err)
	}
	session.conn = conn

	oldPgConn := conn.Underlying().PgConn()
	oldPID := oldPgConn.PID()
	cleanupDone := oldPgConn.CleanupDone()
	calls := 0
	state.listeners = []any{func(AddConsentPurpose) {
		select {
		case <-cleanupDone:
		default:
			t.Error("expected old physical cleanup before recovered dispatch, got pending cleanup")
		}
		if pid := session.conn.Underlying().PgConn().PID(); pid == oldPID {
			t.Errorf("expected replacement PID different from %d, got %d", oldPID, pid)
		}
		calls++
	}}

	// Closing the socket causes WaitForNotification to enter asyncClose.
	err = oldPgConn.Conn().Close()
	if err != nil {
		t.Fatalf("expected interrupted listener socket, got %v", err)
	}

	acquired := proxy.PoolStats().AcquireCount()
	startReplication(t, state, session)
	completed := make(chan struct{})
	go func() { state.close.Wait(); close(completed) }()
	select {
	case <-cancelEntered:
	case <-state.close.ctx.Done():
		t.Fatalf("expected async cleanup at cancel barrier, got %v", state.close.ctx.Err())
	}

	waitReplicationBlocked(t, "closeNotificationConnection(", completed)
	select {
	case <-cleanupDone:
		t.Fatal("expected pending CleanupDone at cancel barrier, got completed cleanup")
	default:
	}

	if got := proxy.PoolStats().AcquireCount(); got != acquired {
		t.Fatalf("expected acquire count %d during pending cleanup, got %d", acquired, got)
	}

	logReplicationEvent(t, database, 124, "reconnected")
	releaseCancel()
	waitReplicationVersion(t, state, 124)
	sendReplicationEvent(t, database, key, 125, AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: stateTestBarrierPurposeID})
	waitReplicationVersion(t, state, 125)
	state.close.cancel()
	state.close.Wait()
	if calls != 2 {
		t.Fatalf("expected one replay and one barrier dispatch, got %d", calls)
	}

}

func logReplicationEvent(t *testing.T, database *db.DB, version int, name string) {

	t.Helper()
	payload, err := json.Marshal(AddConsentPurpose{Workspace: stateTestWorkspaceID, ID: replicationConsentPurposeID(version), Name: name})
	if err != nil {
		t.Fatalf("expected encoded log event, got %v", err)
	}

	_, err = database.Exec(t.Context(), "INSERT INTO notifications VALUES ($1, 'AddConsentPurpose', $2)", version, payload)
	if err != nil {
		t.Fatalf("expected persisted event %d, got %v", version, err)
	}

}

// replicationCleanupProxy forwards PostgreSQL traffic but holds cancellation
// requests behind a channel, keeping pgconn's real CleanupDone pending.
func replicationCleanupProxy(t *testing.T, config *pgx.ConnConfig) (*db.DB, <-chan struct{}, func()) {

	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("expected cleanup proxy listener, got %v", err)
	}

	entered, release := make(chan struct{}), make(chan struct{})
	releaseCancel := sync.OnceFunc(func() { close(release) })
	enterCancel := sync.OnceFunc(func() { close(entered) })
	var workers sync.WaitGroup
	workers.Go(func() {
		for {

			conn, err := listener.Accept()
			if err != nil {
				return
			}

			workers.Go(func() {

				defer conn.Close()
				var header [8]byte
				_, err := io.ReadFull(conn, header[:])
				if err != nil {
					return
				}

				if binary.BigEndian.Uint32(header[4:]) == 80877102 {
					enterCancel()
					<-release
				}

				upstream, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))))
				if err != nil {
					return
				}

				defer upstream.Close()
				_, err = upstream.Write(header[:])
				if err != nil {
					return
				}

				done := make(chan struct{})
				go func() {
					defer close(done)
					_, _ = io.Copy(conn, upstream)
					_ = conn.Close()
					_ = upstream.Close()
				}()
				_, _ = io.Copy(upstream, conn)
				_ = upstream.Close()
				<-done

			})

		}
	})
	t.Cleanup(func() { releaseCancel(); _ = listener.Close(); workers.Wait() })
	proxy, err := db.Open(&db.Options{
		Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port,
		Username: config.User, Password: config.Password, Database: config.Database, MaxConnections: 4,
	})
	if err != nil {
		t.Fatalf("expected cleanup proxy pool, got %v", err)
	}

	t.Cleanup(proxy.Close)

	return proxy, entered, releaseCancel
}

func replicationConsentPurposeID(version int) string {
	return fmt.Sprintf("A7v4N9q2K%03d", version)
}

// replicationState constructs application state and an already-listening
// session; startReplication explicitly transfers ownership to the worker.
func replicationState(t *testing.T, database *db.DB, key *cipher.Key) (*State, *notificationSession) {

	t.Helper()
	org := &Organization{mu: &sync.Mutex{}, ID: stateTestOrganizationID}
	ws := &Workspace{mu: &sync.Mutex{}, ID: stateTestWorkspaceID, organization: org, consentPurposes: map[string]*ConsentPurpose{}}
	state := &State{db: database, mu: &sync.Mutex{}, changing: &sync.RWMutex{}, workspaces: map[string]*Workspace{ws.ID: ws}}
	state.version.current = 123
	state.version.next = sync.Cond{L: &state.version.RWMutex}
	state.close.ctx, state.close.cancel = context.WithTimeout(t.Context(), 15*time.Second)
	t.Cleanup(state.close.cancel)
	state.notifications.notifier = &notifier{db: database, key: key}
	conn, err := state.notifications.connect(t.Context())
	if err != nil {
		t.Fatalf("expected listening session, got %v", err)
	}

	session := &notificationSession{conn: conn}
	t.Cleanup(session.close)

	return state, session
}

func sendReplicationEvent(t *testing.T, database *db.DB, key *cipher.Key, version int, event any) {

	t.Helper()
	// The dedicated producer test exercises Notify's actual fragmentation.
	// Here individual frames intentionally control ordering and fault injection.
	name := fmt.Sprintf("%T", event)
	name = strings.TrimPrefix(name, "state.")
	b, err := appendEncodeNotification(t.Context(), nil, key, name, event)
	if err != nil {
		t.Fatalf("expected encoded event, got %v", err)
	}

	payload := string(b)
	if version > 0 {
		payload += fmt.Sprintf("@%d", version)
	}

	sendReplicationFragment(t, database, payload)

}

func sendReplicationFragment(t *testing.T, database *db.DB, payload string) {
	t.Helper()
	_, err := database.Exec(t.Context(), "SELECT pg_notify('krenalis', $1)", payload)
	if err != nil {
		t.Fatalf("expected PostgreSQL notification, got %v", err)
	}
}

func startReplication(t *testing.T, state *State, session *notificationSession) {
	t.Helper()
	state.close.Add(1)
	go state.keepNotifications(session)
	t.Cleanup(func() { state.close.cancel(); state.close.Wait() })
}

// waitReplicationBlocked observes the worker at a channel barrier, rather than
// inferring quiescence from elapsed time; real I/O precludes a synctest bubble.
func waitReplicationBlocked(t *testing.T, function string, completed <-chan struct{}) {

	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	stack := make([]byte, 1<<20)
	for {

		for _, goroutine := range strings.Split(string(stack[:runtime.Stack(stack, true)]), "\n\n") {
			if strings.Contains(goroutine, "[chan receive]") && strings.Contains(goroutine, function) && strings.Contains(goroutine, "(*State).keepNotifications(") {
				return
			}
		}

		select {
		case <-completed:
			t.Fatalf("expected worker blocked in %s, got completed worker", function)
		case <-ctx.Done():
			t.Fatalf("expected worker blocked in %s, got %v", function, ctx.Err())
		default:
			runtime.Gosched()
		}

	}

}

func waitReplicationVersion(t *testing.T, state *State, version int) {
	t.Helper()
	err := state.WaitVersion(state.close.ctx, version)
	if err != nil {
		t.Fatalf("expected applied version %d, got %v (current %d)", version, err, state.Version())
	}
}
