//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package userswarehouse

import (
	"context"
	"log"
	"slices"
	"sort"
	"time"

	"chichi/apis/datastore"
	"chichi/apis/errors"
	"chichi/apis/state"
	"chichi/telemetry"

	"golang.org/x/exp/maps"
)

// SetUser sets the user U into the data warehouse by resolving its identity.
func SetUser(ctx context.Context, store *datastore.Store, action *state.Action, U map[string]any) error {

	connection := action.Connection()
	ws := connection.Workspace()

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
	candidates, err := store.MatchingUsers(ctx, U)
	if err != nil {
		return err
	}
	found := []datastore.IRUser{}
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
			matching := filterCandidatesByProperty(candidates, property, v)
			if isAnonProp[property] {
				candidates = matching
				if len(candidates) == 1 {
					break identifiersLoop
				}
			} else {
				found = append(found, matching...)
				// Collect the users which are anonymous for every action, then
				// use them as candidates.
				anonymousUsers := []datastore.IRUser{}
			anonUsersLoop:
				for _, user := range candidates {
					for property, value := range user.Identifiers {
						if isNotZeroValue(value) && slices.Contains(allActionsIdents, property) {
							continue anonUsersLoop
						}
					}
					anonymousUsers = append(anonymousUsers, user)
				}
				candidates = anonymousUsers
				skipToFirstAnonProp = true
			}
		} else {
			candidates = filterCandidatesByProperty(candidates, property, nil)
			if len(candidates) == 0 {
				break identifiersLoop
			}
		}
	}

	// Add users from candidates to found, if not already present.
	for _, user := range found {
		add := true
		for _, c := range candidates {
			if c.ID == user.ID {
				add = false
				break
			}
			if add {
				candidates = append(candidates, user)
			}
		}
	}

	switch len(found) {
	case 0:
		gid, err := createGR(ctx, ws, store, U)
		if err != nil {
			return err
		}
		log.Printf("[info] created a new Golden Record with GID %d", gid)
	case 1:
		// Merge U into V, if possible, otherwise add to U to the users.
		V := found[0]
		if canMerge(U, V.Identifiers, action.Identifiers, otherActionsIdents) {
			err := updateGR(ctx, ws, store, V, U)
			if err != nil {
				return err
			}
			log.Printf("[info] updated the Golden Record with GID %d", found[0].ID)
		} else {
			gid, err := createGR(ctx, ws, store, U)
			if err != nil {
				return err
			}
			log.Printf("[info] created a new Golden Record with GID %d", gid)
		}
	default:

		// Define target as the user within found with the lower GID.
		sort.Slice(found, func(i, j int) bool {
			return found[i].ID < found[j].ID
		})
		target := found[0]

		// Merge every user in found (starting with the ones with lower GID and
		// excluding target) into target, if merge is possible, otherwise do
		// nothing.
		merged := []int{}
		for _, user := range found[1:] {
			if canMerge(user.Identifiers, target.Identifiers, action.Identifiers, otherActionsIdents) {
				err := updateGR(ctx, ws, store, target, user.Identifiers)
				if err != nil {
					return err
				}
				log.Printf("[info] updated the Golden Record with GID %d", target.ID)
				merged = append(merged, user.ID)
			}
		}

		// Merge U into target, if merge is possible, otherwise add U to the
		// users.
		if canMerge(U, target.Identifiers, action.Identifiers, otherActionsIdents) {
			err := updateGR(ctx, ws, store, target, U)
			if err != nil {
				return err
			}
			log.Printf("[info] updated the Golden Record with GID %d", target.ID)
		} else {
			gid, err := createGR(ctx, ws, store, U)
			if err != nil {
				return err
			}
			log.Printf("[info] created a new Golden Record with GID %d", gid)
		}

		// Delete every merged user.
		for _, user := range merged {
			telemetry.IncrementCounter(ctx, "deleteGR", 1)
			err := store.DeleteUser(ctx, user)
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
func createGR(ctx context.Context, ws *state.Workspace, store *datastore.Store, U map[string]any) (int, error) {
	telemetry.IncrementCounter(ctx, "createGR", 1)

	// Serialize the row.
	schema, ok := ws.Schemas["users"]
	if !ok {
		return 0, errors.New("users schema not found")
	}

	datastore.SerializeRow(U, *schema)

	// TODO(Gianluca): should the user be normalized before being written on the
	// data store?

	user := maps.Clone(U)
	user["timestamp"] = time.Now().UTC()
	id, err := store.CreateUser(ctx, user)

	return id, err
}

// filterCandidatesByProperty returns the candidates with the given property
// value. As a special case, passing a nil value make this function return
// candidates with no values for that property.
func filterCandidatesByProperty(candidates []datastore.IRUser, prop string, value any) []datastore.IRUser {
	filtered := make([]datastore.IRUser, 0, len(candidates))
	if value == nil {
		for _, c := range candidates {
			if _, ok := c.Identifiers[prop]; !ok {
				filtered = append(filtered, c)
			}
		}
	} else {
		for _, c := range candidates {
			if c.Identifiers[prop] == value {
				filtered = append(filtered, c)
			}
		}
	}
	return filtered
}

// updateGR updates the Golden Record with the given identifier. Only the
// properties in users will be updated.
func updateGR(ctx context.Context, ws *state.Workspace, store *datastore.Store, target datastore.IRUser, user map[string]any) error {
	telemetry.IncrementCounter(ctx, "updateGR", 1)
	schema, ok := ws.Schemas["users"]
	if !ok {
		return errors.New("users schema not found")
	}
	datastore.SerializeRow(user, *schema)
	// TODO(Gianluca): should the user be normalized before being written on the
	// data store?
	err := store.UpdateUser(ctx, target, user)
	return err
}
