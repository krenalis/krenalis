// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// TestElectAsLeader verifies that election numbers beyond the 32-bit limit
// are stored in PostgreSQL and sent through NOTIFY, and that stale attempts
// and attempts to exceed the 64-bit limit are rejected.
func TestElectAsLeader(t *testing.T) {

	database, key := replicationDatabase(t)
	_, err := database.Exec(t.Context(), `
		CREATE TABLE election (number bigint PRIMARY KEY, leader text NOT NULL, date timestamp NOT NULL);
		INSERT INTO election VALUES (1, 'previous', '2023-01-01');
	`)
	if err != nil {
		t.Fatalf("expected election fixture, got %v", err)
	}

	for _, test := range []struct {
		name     string
		current  int
		previous int
		expected int
		ended    bool
		limit    bool
	}{
		{name: "past-int32", current: math.MaxInt32, previous: math.MaxInt32, expected: math.MaxInt32 + 1},
		{name: "above-int32", current: math.MaxInt32 + 1, previous: math.MaxInt32 + 1, expected: math.MaxInt32 + 2},
		{name: "maximum", current: math.MaxInt64 - 1, previous: math.MaxInt64 - 1, expected: math.MaxInt64},
		{name: "ended", current: math.MaxInt32 + 1, previous: math.MaxInt32, expected: math.MaxInt32 + 1, ended: true},
		{name: "stale-at-maximum", current: math.MaxInt64, previous: math.MaxInt64 - 1, expected: math.MaxInt64, ended: true},
		{name: "exhausted", current: math.MaxInt64, previous: math.MaxInt64, expected: math.MaxInt64, limit: true},
	} {
		t.Run(test.name, func(t *testing.T) {

			_, err := database.Exec(t.Context(), "UPDATE election SET number = $1, leader = 'previous'", test.current)
			if err != nil {
				t.Fatalf("expected initial election %d, got %v", test.current, err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			state := &State{id: "candidate", db: database, notifications: &notifier{db: database, key: key}}
			state.close.ctx = ctx
			conn, err := state.notifications.connect(ctx)
			if err != nil {
				t.Fatalf("expected election observer, got %v", err)
			}
			defer closeNotificationConnection(conn)

			err = state.electAsLeader(test.previous)
			if err != nil {
				if !test.ended && !test.limit {
					t.Fatalf("expected successful election, got %v", err)
				}
				if test.ended && !errors.Is(err, errEndedElection) {
					t.Fatalf("expected ended election error, got %v", err)
				}
				if test.limit {
					const expected = "election number limit reached"
					if err.Error() != expected {
						t.Fatalf("expected %q, got %v", expected, err)
					}
				}
			}
			if (test.ended || test.limit) && err == nil {
				t.Fatal("expected election error, got nil")
			}

			var number, count int
			var leader string
			err = database.QueryRow(ctx, "SELECT number, leader, count(*) OVER () FROM election").Scan(&number, &leader, &count)
			if err != nil {
				t.Fatalf("expected persisted election, got %v", err)
			}
			if count != 1 {
				t.Fatalf("expected one election row, got %d", count)
			}
			if number != test.expected {
				t.Fatalf("expected election number %d, got %d", test.expected, number)
			}
			if test.ended || test.limit {
				if leader != "previous" {
					t.Fatalf("expected unchanged leader %q, got %q", "previous", leader)
				}
				// A committed marker follows any notification from the rejected attempt.
				const barrier = "rejected-election-barrier"
				_, err = database.Exec(ctx, "SELECT pg_notify('krenalis', $1)", barrier)
				if err != nil {
					t.Fatalf("expected committed notification barrier, got %v", err)
				}
				raw, err := conn.Underlying().WaitForNotification(ctx)
				if err != nil {
					t.Fatalf("expected notification barrier, got %v", err)
				}
				if raw.Payload != barrier {
					t.Fatalf("expected notification barrier %q first, got %q", barrier, raw.Payload)
				}
				return
			}
			if leader != state.id {
				t.Fatalf("expected leader %q, got %q", state.id, leader)
			}

			raw, err := conn.Underlying().WaitForNotification(ctx)
			if err != nil {
				t.Fatalf("expected election NOTIFY, got %v", err)
			}
			session := &notificationSession{}
			n, err := session.reconstruct(ctx, key, raw.Payload)
			if err != nil {
				t.Fatalf("expected reconstructed election, got %v", err)
			}
			if n == nil || n.Version != 0 || n.Name != "ElectLeader" {
				t.Fatalf("expected unversioned ElectLeader, got %#v", n)
			}
			var event ElectLeader
			err = json.Unmarshal([]byte(n.Payload), &event)
			if err != nil {
				t.Fatalf("expected decoded election, got %v", err)
			}
			if event.Number != test.expected || event.Leader != state.id {
				t.Fatalf("expected election %d with leader %q, got %+v", test.expected, state.id, event)
			}

		})
	}

}

// TestKeepElectionsAdvancesPastMaxInt32 verifies that the election worker
// advances the election number beyond math.MaxInt32 and the replication
// worker applies the new election.
func TestKeepElectionsAdvancesPastMaxInt32(t *testing.T) {

	database, key := replicationDatabase(t)
	_, err := database.Exec(t.Context(), `
		CREATE TABLE election (number bigint PRIMARY KEY, leader text NOT NULL, date timestamp NOT NULL);
		INSERT INTO election VALUES (2147483647, 'previous', '2023-01-01');
	`)
	if err != nil {
		t.Fatalf("expected election fixture, got %v", err)
	}
	state, session := replicationState(t, database, key)
	state.id = "candidate"
	state.election = election{number: math.MaxInt32, leader: "previous"}
	applied := make(chan ElectLeader, 1)
	state.listeners = []any{func(e ElectLeader) {
		select {
		case applied <- e:
		case <-state.close.ctx.Done():
		}
	}}
	startReplication(t, state, session)
	state.close.Add(1)
	go state.keepElections()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	select {
	case e := <-applied:
		if e.Number != math.MaxInt32+1 || e.Leader != state.id {
			t.Fatalf("expected election %d with leader %q, got %+v", math.MaxInt32+1, state.id, e)
		}
	case <-ctx.Done():
		t.Fatalf("expected applied election beyond MaxInt32, got %v", ctx.Err())
	}
	state.close.cancel()
	state.close.Wait()

	var number, count int
	var leader string
	err = database.QueryRow(t.Context(), "SELECT number, leader, count(*) OVER () FROM election").Scan(&number, &leader, &count)
	if err != nil {
		t.Fatalf("expected persisted election, got %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one election row, got %d", count)
	}
	if number != math.MaxInt32+1 || leader != state.id {
		t.Fatalf("expected persisted election %d with leader %q, got %d with leader %q", math.MaxInt32+1, state.id, number, leader)
	}
	if got := state.election.number; got != math.MaxInt32+1 {
		t.Fatalf("expected applied election number %d, got %d", math.MaxInt32+1, got)
	}
	if !state.IsLeader() {
		t.Fatal("expected this node to be leader, got follower")
	}

}

// TestKeepElectionsHeartbeats verifies that heartbeat sequences advance across
// election acknowledgement, replays with older sequences are ignored, and
// recreating the sender starts a new sequence in a new election.
func TestKeepElectionsHeartbeats(t *testing.T) {

	database, key := replicationDatabase(t)
	_, err := database.Exec(t.Context(), `
		CREATE TABLE election (number bigint PRIMARY KEY, leader text NOT NULL, date timestamp NOT NULL);
		INSERT INTO election VALUES (7, 'previous', '2023-01-01');
	`)
	if err != nil {
		t.Fatalf("expected election fixture, got %v", err)
	}

	// New assigns a new node ID on every restart.
	for _, id := range []string{"first", "restarted"} {
		t.Run(id, func(t *testing.T) {

			state, session := replicationState(t, database, key)
			state.id = id
			err := database.QueryRow(t.Context(), "SELECT number, leader FROM election").Scan(&state.election.number, &state.election.leader)
			if err != nil {
				t.Fatalf("expected loaded election, got %v", err)
			}
			expectedElection := state.election.number + 1
			startedAt := time.Now().UnixMilli()
			state.close.Add(1)
			go state.keepElections()
			defer func() { state.close.cancel(); state.close.Wait() }()

			// Delay applying the election until the candidate has sent heartbeats.
			raw, err := session.conn.Underlying().WaitForNotification(state.close.ctx)
			if err != nil {
				t.Fatalf("expected election notification, got %v", err)
			}
			elected, err := session.reconstruct(t.Context(), key, raw.Payload)
			if err != nil {
				t.Fatalf("expected reconstructed election, got %v", err)
			}
			if elected == nil || elected.Name != "ElectLeader" || elected.Version != 0 {
				t.Fatalf("expected unversioned ElectLeader notification, got %+v", elected)
			}
			var number int
			var leader, electionVersion string
			err = database.QueryRow(t.Context(), "SELECT number, leader, xmin::text FROM election").Scan(&number, &leader, &electionVersion)
			if err != nil {
				t.Fatalf("expected persisted election, got %v", err)
			}
			if number != expectedElection || leader != id {
				t.Fatalf("expected persisted election %d with leader %q, got %d with leader %q", expectedElection, id, number, leader)
			}

			var replayPayload string
			var replayTimestamp, acknowledgedAt int64
			var heartbeatsAfterAck int
			for sequence := uint64(1); ; sequence++ {

				raw, err := session.conn.Underlying().WaitForNotification(state.close.ctx)
				if err != nil {
					t.Fatalf("expected heartbeat %d, got %v", sequence, err)
				}
				n, err := session.reconstruct(t.Context(), key, raw.Payload)
				if err != nil {
					t.Fatalf("expected reconstructed heartbeat, got %v", err)
				}
				if n == nil || n.Name != "SeeLeader" || n.Version != 0 {
					t.Fatalf("expected unversioned SeeLeader, got %+v", n)
				}
				var event SeeLeader
				err = json.Unmarshal([]byte(n.Payload), &event)
				if err != nil {
					t.Fatalf("expected decoded heartbeat, got %v", err)
				}
				if event.Election != expectedElection || event.Sequence != sequence {
					t.Fatalf("expected election %d and sequence %d, got %+v", expectedElection, sequence, event)
				}
				receivedAt := time.Now().UnixMilli()
				if event.Timestamp < startedAt || event.Timestamp > receivedAt {
					t.Fatalf("expected send timestamp between %d and %d, got %d", startedAt, receivedAt, event.Timestamp)
				}
				if sequence == 1 {
					continue
				}
				if sequence == 2 {
					state.electLeader(*elected)
					acknowledgedAt = time.Now().UnixMilli()
				}

				state.seeLeader(*n)
				state.mu.Lock()
				expected := state.election
				state.mu.Unlock()
				if expected.number != expectedElection || expected.leader != id || expected.sequence != sequence {
					t.Fatalf("expected acknowledged election and heartbeat, got %+v", expected)
				}

				// One send may have been in progress at acknowledgement. Wait for
				// two later sends to exercise the confirmed leader's loop as well.
				if sequence > 2 && event.Timestamp > acknowledgedAt {
					heartbeatsAfterAck++
					if heartbeatsAfterAck == 2 {
						break
					}
				}
				replayPayload = raw.Payload
				replayTimestamp = event.Timestamp

			}

			state.mu.Lock()
			expected := state.election
			state.mu.Unlock()
			// Replay the preceding heartbeat's ciphertext without using the key.
			sendReplicationFragment(t, database, replayPayload)
			for {
				replayed, err := session.conn.Underlying().WaitForNotification(state.close.ctx)
				if err != nil {
					t.Fatalf("expected replayed heartbeat, got %v", err)
				}
				// The sender may have queued more heartbeats before the replay.
				if replayed.Payload != replayPayload {
					continue
				}
				n, err := session.reconstruct(t.Context(), key, replayed.Payload)
				if err != nil {
					t.Fatalf("expected authenticated replay, got %v", err)
				}
				if n == nil || n.Name != "SeeLeader" || n.Version != 0 {
					t.Fatalf("expected complete unversioned heartbeat, got %+v", n)
				}
				state.seeLeader(*n)
				// The sequence check must reject the replay while it is still fresh.
				age := time.Since(time.UnixMilli(replayTimestamp))
				if age < -leaderClockSkew || age >= grantedLeaderInterval+leaderClockSkew {
					t.Fatalf("expected replay within heartbeat validity window, got age %v", age)
				}
				state.mu.Lock()
				got := state.election
				state.mu.Unlock()
				if got != expected {
					t.Fatalf("expected replay to leave election %+v unchanged, got %+v", expected, got)
				}
				break
			}

			var gotVersion string
			err = database.QueryRow(t.Context(), "SELECT xmin::text FROM election").Scan(&gotVersion)
			if err != nil {
				t.Fatalf("expected election row version, got %v", err)
			}
			if gotVersion != electionVersion {
				t.Fatalf("expected heartbeats to leave election row version %s unchanged, got %s", electionVersion, gotVersion)
			}
			var count int
			err = database.QueryRow(t.Context(), "SELECT count(*) FROM notifications").Scan(&count)
			if err != nil {
				t.Fatalf("expected notification log query, got %v", err)
			}
			if count != 0 {
				t.Fatalf("expected heartbeats outside the notification log, got %d rows", count)
			}

		})
	}

}
