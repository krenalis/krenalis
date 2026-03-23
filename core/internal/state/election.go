// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"os"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/backoff"
)

var debugElection = false

const (
	leaderInterval         = 2 * time.Second
	grantedLeaderInterval  = leaderInterval + leaderInterval/2
	electionRandomInterval = leaderInterval / 5
)

// keepElections keeps leader elections.
// It is called in its own goroutine after the state is loaded.
func (state *State) keepElections() {

	defer state.close.Done()
	state.election.lastSeen = time.Now()

	// Check if the MEERGO_DEBUG_ELECTION variable is set.
	if v := os.Getenv("MEERGO_DEBUG_ELECTION"); v == "true" {
		debugElection = true
	}

	// |--------------|·······|~~~~~~~|
	//
	// --- leader send a notification at the end of this interval.
	// ··· followers grant additional time to the leader.
	// ~~~ followers try, in a random time, to become the leader.
	//
	// |--------------|          leaderInterval
	// |--------------|·······|  grantedLeaderInterval
	// |~~~~~~~|                 electionRandomInterval

	debugf := func(format string, a ...any) {
		if !debugElection {
			return
		}
		_, _ = fmt.Fprintf(os.Stderr, format, a...)
	}

	// leader is called when the node is the leader.
	// If state.ctx is canceled, leader returns its error.
	leader := func(election election) error {
		debugf("-- %d Leader\n", election.number)
		bo := backoff.New(100)
		bo.SetCap(leaderInterval)
		var failed bool
		for bo.Next(state.close.ctx) {
			// Send the see leader notification.
			err := state.Transaction(state.close.ctx, func(tx *db.Tx) (any, error) {
				return SeeLeader{Election: election.number}, nil
			})
			if err == nil {
				break
			}
			debugf("\t%s\n", err)
			if !failed {
				slog.Warn("failed to send a see leader notification; retrying", "error", err)
				failed = true
			}
		}
		if err := state.close.ctx.Err(); err != nil {
			return err
		}
		if failed {
			slog.Info("see leader notification successfully resent")
		}
		return state.sleep(leaderInterval)
	}

	// follower is called when the node is a follower.
	// If state.ctx is canceled, follower returns its error.
	follower := func(election election) error {
		debugf("-- %d Follower\n", election.number)
		now := time.Now()
		deadline := election.lastSeen.Add(grantedLeaderInterval)
		if deadline.After(now) {
			return state.sleep(deadline.Sub(now))
		}
		d := time.Duration(rand.IntN(int(electionRandomInterval)))
		debugf("\t%s until election\n", d)
		if err := state.sleep(d); err != nil {
			return err
		}
		election.number++
		if int32(election.number) < 0 {
			election.number = 1
		}
		state.mu.Lock()
		number := state.election.number
		state.mu.Unlock()
		if election.number == number {
			debugf("\telection already ended\n")
			return nil
		}
		debugf("\ttry election %d: ", election.number)
		err := state.electAsLeader(election.number)
		if err == nil {
			debugf("elected!\n")
			if err := state.sleep(leaderInterval); err != nil {
				return err
			}
			// Await notification of the elected leader.
			for {
				state.mu.Lock()
				ack := election.number <= state.election.number
				state.mu.Unlock()
				if ack {
					return nil
				}
				err := leader(election)
				if err != nil {
					return err
				}
			}
		}
		if err == errEndedElection {
			debugf("election was ended\n")
			return nil
		}
		if err == context.Canceled {
			<-state.close.ctx.Done()
			return err
		}
		debugf("\t%s\n", err)
		slog.Warn("core/state: cannot send leader election notification", "error", err)
		return nil
	}

	var err error

	for err == nil {
		state.mu.Lock()
		election := state.election
		state.mu.Unlock()
		if election.leader == state.id {
			err = leader(election)
		} else {
			err = follower(election)
		}
	}

}

// errEndedElection indicates that an election is ended.
var errEndedElection = errors.New("ended election")

// electAsLeader attempts to elect the current node as leader in the given
// election. It returns an errEndedElection error if the given election is
// ended.
func (state *State) electAsLeader(election int) error {
	n := ElectLeader{Leader: state.id, Number: election}
	var prevElection = election - 1
	if prevElection == 0 {
		prevElection = math.MaxInt32
	}
	ctx := state.close.ctx
	err := state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		var t bool
		err := tx.QueryRow(ctx, "UPDATE election\n"+
			"SET number = $1, leader = $2, date = NOW()::timestamp\n"+
			"WHERE number = $3 RETURNING true", n.Number, n.Leader, prevElection).Scan(&t)
		if err != nil {
			if err == sql.ErrNoRows {
				err = errEndedElection
			}
			return nil, err
		}
		return n, nil
	})
	return err
}

// sleep pauses the current goroutine for at least the duration d, unless the
// state.ctx context is canceled, in that case it returns immediately with the
// context error.
func (state *State) sleep(d time.Duration) error {
	select {
	case <-state.close.ctx.Done():
		return state.close.ctx.Err()
	case <-time.After(d):
		return nil
	}
}
