//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package userswarehouse

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"chichi/apis/errors"
	"chichi/apis/index"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/warehouses"
	"chichi/telemetry"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slices"
)

// SetUser sets the user U into the data warehouse by resolving its identity.
func SetUser(ctx context.Context, redis *redis.Client, connection *state.Connection, action *state.Action, U map[string]any) error {

	ws := connection.Workspace()

	// Open the index on Redis.
	index := index.Open(redis)

	// Instantiate a sorted set of identifiers (including anonymous identifiers)
	// for this action.
	P := make([]string, 0, len(action.Identifiers)+len(ws.AnonymousIdentifiers.Priority))
	P = append(P, action.Identifiers...)
	P = append(P, ws.AnonymousIdentifiers.Priority...)
	if len(P) == 0 {
		return errors.New("BUG: cannot resolve identities as there are no identifiers")
	}

	isAnonProp := make(map[string]bool, len(ws.AnonymousIdentifiers.Priority))
	for _, prop := range ws.AnonymousIdentifiers.Priority {
		isAnonProp[prop] = true
	}

	// Collect the non-anonymous identifiers for other actions.
	var otherActionsIdents []string
	for _, a := range connection.Actions() {
		if a.ID == action.ID {
			continue
		}
		idents := a.Identifiers
		for _, ident := range idents {
			if slices.Contains(otherActionsIdents, ident) {
				continue
			}
			otherActionsIdents = append(otherActionsIdents, ident)
		}
	}

	// Collect the non-anonymous identifiers for every action.
	var allActionsIdents []string
	allActionsIdents = append(allActionsIdents, otherActionsIdents...)
	for _, id := range action.Identifiers {
		if slices.Contains(allActionsIdents, id) {
			continue
		}
		allActionsIdents = append(allActionsIdents, id)
	}

	// Resolve the identity of the user.
	skipToFirstAnonProp := false
	var candidates []int // nil means every user
	found := []int{}
identifiersLoop:
	for _, property := range P {
		if skipToFirstAnonProp {
			if isAnonProp[property] {
				skipToFirstAnonProp = false
			} else {
				continue
			}
		}
		v, ok := U[property]
		if ok && isNotZeroValue(v) {
			matching, err := index.UsersByPropertyValue(ctx, candidates, property, v)
			if err != nil {
				return err
			}
			if isAnonProp[property] {
				candidates = matching
				if len(candidates) == 1 {
					break identifiersLoop
				}
			} else {
				found = append(found, matching...)
				// Collect the users which are anonymous for every action, then
				// use them as candidates.
				anonymousUsers := []int{}
			anonUsersLoop:
				for _, userGID := range candidates {
					user, err := index.GetUser(ctx, userGID)
					if err != nil {
						return err
					}
					for property, value := range user {
						if isNotZeroValue(value) && slices.Contains(allActionsIdents, property) {
							continue anonUsersLoop
						}
					}
					anonymousUsers = append(anonymousUsers, userGID)
				}
				candidates = anonymousUsers
				skipToFirstAnonProp = true
			}
		} else {
			var err error
			candidates, err = index.UsersWithNoPropertyValue(ctx, candidates, property)
			if err != nil {
				return err
			}
			if len(candidates) == 0 {
				break identifiersLoop
			}
		}
	}

	// Add users from candidates to found.
	for _, user := range found {
		if !slices.Contains(candidates, user) {
			candidates = append(candidates, user)
		}
	}

	switch len(found) {
	case 0:
		gid, err := createGR(ctx, ws, index, U)
		if err != nil {
			return err
		}
		log.Printf("[info] created a new Golden Record with GID %d", gid)
	case 1:
		// Merge U into V, if possible, otherwise add to U to the users.
		V, err := index.GetUser(ctx, found[0])
		if err != nil {
			return err
		}
		if canMerge(U, V, action.Identifiers, otherActionsIdents) {
			err := updateGR(ctx, ws, index, found[0], U)
			if err != nil {
				return err
			}
			log.Printf("[info] updated the Golden Record with GID %d", found[0])
		} else {
			gid, err := createGR(ctx, ws, index, U)
			if err != nil {
				return err
			}
			log.Printf("[info] created a new Golden Record with GID %d", gid)
		}
	default:

		sort.Ints(found)

		// Define target as the user within found with the lower GID.
		targetGID := found[0]
		target, err := index.GetUser(ctx, targetGID)
		if err != nil {
			return err
		}

		// Merge every user in found (starting with the ones with lower GID and
		// excluding target) into target, if merge is possible, otherwise do
		// nothing.
		merged := []int{}
		for _, userGID := range found[1:] {
			user, err := index.GetUser(ctx, userGID)
			if err != nil {
				return err
			}
			if canMerge(user, target, action.Identifiers, otherActionsIdents) {
				err := updateGR(ctx, ws, index, targetGID, user)
				if err != nil {
					return err
				}
				log.Printf("[info] updated the Golden Record with GID %d", targetGID)
				merged = append(merged, userGID)
			}
		}

		// Merge U into target, if merge is possible, otherwise add U to the
		// users.
		if canMerge(U, target, action.Identifiers, otherActionsIdents) {
			err := updateGR(ctx, ws, index, targetGID, U)
			if err != nil {
				return err
			}
			log.Printf("[info] updated the Golden Record with GID %d", targetGID)
		} else {
			gid, err := createGR(ctx, ws, index, U)
			if err != nil {
				return err
			}
			log.Printf("[info] created a new Golden Record with GID %d", gid)
		}

		// Delete every merged user.
		for _, user := range merged {
			err := deleteGR(ctx, ws, index, user)
			if err != nil {
				return err
			}
			log.Printf("[info] user with GID %d deleted", user)
		}

	}

	return nil
}

// canMerge reports whether the user V can be merged into the user W.
// identifiers contains the not-anonymous identifiers for this action, while
// otherIdentifiers contains the not-anonymous identifiers for other actions.
//
// TODO(Gianluca): this method can be extend to handle cases when a value can be
// appended to a slice without losing information.
func canMerge(U, W map[string]any, identifiers, otherIdentifiers []string) bool {
	for property := range U {
		if isNotZeroValue(W[property]) &&
			!slices.Contains(identifiers, property) &&
			slices.Contains(otherIdentifiers, property) {
			return false
		}
	}
	return true
}

// isZeroValue reports whether v is not a zero value.
func isNotZeroValue(v any) bool {
	return v != "" && v != 0 && v != nil
}

// createGR creates a new Golden Record for the user U and returns its GID.
func createGR(ctx context.Context, ws *state.Workspace, index *index.Index, U map[string]any) (int, error) {
	telemetry.IncrementCounter(ctx, "createGR", 1)
	// Create an empty Golden Record.
	var gid int
	err := ws.Warehouse.QueryRow(ctx, "INSERT INTO users DEFAULT VALUES RETURNING id").Scan(&gid)
	if err != nil {
		return 0, err
	}
	// Update the Golden Record.
	err = updateGR(ctx, ws, index, gid, U)
	if err != nil {
		return 0, err
	}
	return gid, nil
}

// updateGR updates the Golden Record with the given GID using the properties of
// U.
func updateGR(ctx context.Context, ws *state.Workspace, index *index.Index, gid int, U map[string]any) error {

	telemetry.IncrementCounter(ctx, "updateGR", 1)

	// Serialize the row.
	schema, ok := ws.Schemas["users"]
	if !ok {
		return errors.New("users schema not found")
	}

	warehouses.SerializeRow(U, *schema)

	// TODO(Gianluca): should the user be normalized before being written on the
	// data warehouse?

	// TODO(Gianluca): replace this query -- as well as the other queries in
	// this file -- with calls to specific methods of the data warehouse, when
	// these will be implemented.
	query := &strings.Builder{}
	query.WriteString("UPDATE users SET\n")
	var values []any
	i := 1
	for prop, value := range U {
		if i > 1 {
			query.WriteString(", ")
		}
		query.WriteString(postgres.QuoteIdent(prop))
		query.WriteString(" = $")
		query.WriteString(strconv.Itoa(i))
		values = append(values, value)
		i++
	}
	query.WriteString(`, "timestamp" = now()`)
	query.WriteString("\nWHERE id = $")
	query.WriteString(strconv.Itoa(i))
	values = append(values, gid)
	res, err := ws.Warehouse.Exec(ctx, query.String(), values...)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("BUG: one row should be affected, got %d", affected)
	}

	err = index.SetUser(ctx, gid, U)
	if err != nil {
		return err
	}

	return nil
}

// deleteGR deletes the Golden Record with the given GID.
func deleteGR(ctx context.Context, ws *state.Workspace, index *index.Index, gid int) error {
	telemetry.IncrementCounter(ctx, "deleteGR", 1)
	// Remove the Golden Record from the data warehouse.
	_, err := ws.Warehouse.Query(ctx, "DELETE FROM `users` WHERE `id` = ?", gid)
	if err != nil {
		return err
	}
	// Delete the user's data from the Redis index.
	err = index.DeleteUser(ctx, gid)
	return err
}
