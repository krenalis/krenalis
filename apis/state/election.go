//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"chichi/apis/postgres"
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

	// Check if the CHICHI_DEBUG_ELECTION variable is set.
	if v := os.Getenv("CHICHI_DEBUG_ELECTION"); v == "true" {
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

	ctx := context.Background()
	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

	debugf := func(format string, a ...any) {
		if !debugElection {
			return
		}
		_, _ = fmt.Fprintf(os.Stderr, format, a...)
	}

	// leader is called when the node is the leader.
	leader := func(election election) {
		debugf("-- %d Leader\n", election.number)
		for {
			// Send the see leader notification.
			err := state.db.Transaction(ctx, func(tx *postgres.Tx) error {
				return tx.Notify(ctx, SeeLeaderNotification{Election: election.number})
			})
			if err == nil {
				break
			}
			debugf("\t%s\n", err)
			log.Printf("[warning] cannot send a see leader notification: %s", err)
			time.Sleep(100 * time.Millisecond)
		}
		time.Sleep(leaderInterval)
	}

	// follower is called when the node is a follower.
	follower := func(election election) {
		debugf("-- %d Follower\n", election.number)
		now := time.Now()
		deadline := election.lastSeen.Add(grantedLeaderInterval)
		if deadline.After(now) {
			time.Sleep(deadline.Sub(now))
			return
		}
		d := time.Duration(randSource.Intn(int(electionRandomInterval)))
		debugf("\t%s until election\n", d)
		time.Sleep(d)
		election.number++
		state.mu.Lock()
		number := state.election.number
		state.mu.Unlock()
		if election.number == number {
			debugf("\telection already ended\n")
			return
		}
		debugf("\ttry election %d: ", election.number)
		err := state.electAsLeader(election.number)
		if err == nil {
			debugf("elected!\n")
			time.Sleep(leaderInterval)
			// Await notification of the elected leader.
			for {
				state.mu.Lock()
				ack := election.number <= state.election.number
				state.mu.Unlock()
				if ack {
					return
				}
				leader(election)
			}
		}
		if err == errEndedElection {
			debugf("number was ended\n")
			return
		}
		debugf("\t%s\n", err)
		log.Printf("[warning] cannot send leader number notification: %s", err)
		return
	}

	for {
		state.mu.Lock()
		election := state.election
		state.mu.Unlock()
		if election.leader == state.id {
			leader(election)
		} else {
			follower(election)
		}
	}

}

// errEndedElection indicates that an election is ended.
var errEndedElection = errors.New("ended election")

// electAsLeader attempts to elect the current node as leader in the given
// election. It returns an errEndedElection error if the given election is
// ended.
func (state *State) electAsLeader(election int) error {
	n := ElectLeaderNotification{Leader: state.id, Number: election}
	ctx := context.Background()
	err := state.db.Transaction(ctx, func(tx *postgres.Tx) error {
		var t bool
		err := tx.QueryRow(ctx, "UPDATE election\n"+
			"SET number = $1, leader = $2, date = NOW()::timestamp\n"+
			"WHERE number = $3 RETURNING true", n.Number, n.Leader, n.Number-1).Scan(&t)
		if err != nil {
			if err == sql.ErrNoRows {
				err = errEndedElection
			}
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
}
