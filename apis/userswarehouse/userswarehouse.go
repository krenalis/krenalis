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
	"reflect"
	"slices"
	"sort"
	"time"

	"chichi/apis/datastore"
	"chichi/apis/errors"
	"chichi/apis/state"
	"chichi/connector/types"
	"chichi/telemetry"

	"golang.org/x/exp/maps"
)

// SetUser sets the user U into the data warehouse by resolving its identity.
func SetUser(ctx context.Context, store *datastore.Store, A *state.Action, U map[string]any) error {

	connection := A.Connection()
	ws := connection.Workspace()
	anonIdents := ws.AnonymousIdentifiers.Priority

	// Determine which properties are arrays.
	schema, ok := ws.Schemas["users"]
	if !ok {
		return errors.New("users schema not found")
	}
	isArray := map[string]bool{}
	for _, p := range schema.Properties() {
		if p.Type.PhysicalType() == types.PtArray {
			isArray[p.Name] = true
		}
	}

	// Retrieve the users that has at least one property value in common with
	// the identifiers of U.
	identsOfA := actionIdents(A, anonIdents)
	identsValues := make(map[string]any, len(identsOfA))
	for _, p := range identsOfA {
		if v, ok := U[p]; ok {
			identsValues[p] = v
		}
	}
	matching, err := store.UsersWithCommonValue(ctx, identsValues, isArray)
	if err != nil {
		return fmt.Errorf("cannot retrieve users with common values: %s", err)
	}

	// Add the user U to the users. Maybe it will be merged with an existent
	// user in a following step of the algorithm.
	gid, err := createGR(ctx, ws, store, U, isArray)
	if err != nil {
		return err
	}
	log.Printf("[info] user %d created", gid)

	// Name 'current' the newly created user.
	current := datastore.IRUser{
		ID:         gid,
		Properties: U,
	}

	// Keep in 'matching' only the users who match with U for the current action
	// A.
	matching = slices.DeleteFunc(matching, func(u datastore.IRUser) bool {
		return match(current, u, A, true, anonIdents, isArray) != matchSame
	})

	// Sort the matching users by ascending GID.
	sort.Slice(matching, func(i, j int) bool {
		return matching[i].ID < matching[j].ID
	})

	// For every user 'u' in 'matching':
matchingUsersLoop:
	for i, u := range matching {

		// If 'current' does not match with 'u' for the current action A, then
		// continue to the next user. This check is skipped for the first user,
		// as it has been already performed.
		if i > 0 && match(current, u, A, true, anonIdents, isArray) != matchSame {
			continue
		}

		// For every action 'a' != A.
		for _, a := range connection.Actions() {
			if a == A {
				continue
			}
			// If the match states that 'current' and 'u' are different users,
			// then continue to the next user.
			if match(current, u, a, false, anonIdents, isArray) == matchDifferent {
				continue matchingUsersLoop
			}
		}

		// Merge 'current' into 'u'.
		err := merge(ctx, ws, store, current, u, isArray)
		if err != nil {
			return fmt.Errorf("cannot merge users: %s", err)
		}
		log.Printf("[info] user %d merged into %d", current.ID, u.ID)

		// Assume 'u' as 'current'.
		current = u
	}

	return nil
}

// actionIdents returns the identifiers for the action a, including the
// anonymous identifiers of the workspace, which should be passed in anonIdents.
func actionIdents(a *state.Action, anonIdents []string) []string {
	ids := make([]string, 0, len(a.Identifiers)+len(anonIdents))
	ids = append(ids, a.Identifiers...)
	ids = append(ids, anonIdents...)
	if len(ids) == 0 {
		panic("BUG: cannot resolve identities as there are no identifiers")
	}
	return ids
}

// createGR creates a new Golden Record for the user U and returns its GID.
// isArray indicates which properties have type array.
func createGR(ctx context.Context, ws *state.Workspace, store *datastore.Store, U map[string]any, isArray map[string]bool) (int, error) {

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
	id, err := store.CreateUser(ctx, user, isArray)

	return id, err
}

// deduplicate deduplicates the elements in s, returning the new slice.
// If s is nil, s is returned.
func deduplicate(s []any) []any {
	if s == nil {
		return s
	}
	ss := make([]any, 0, len(s))
	for _, elem := range s {
		add := true
		for _, elem2 := range ss {
			if equal(elem, elem2) {
				add = false
				break
			}
		}
		if add {
			ss = append(ss, elem)
		}
	}
	return ss
}

// equal reports whether a and b are two equal values, in perspective of the
// identity resolution.
//
// TODO(Gianluca): use a more efficient and correct way to check for equality.
// See the issue https://github.com/open2b/chichi/issues/272.
func equal(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

// haveCommonValue reports whether a and b have at least one common value.
func haveCommonValue(a, b []any) bool {
	for _, elem := range a {
		for _, elem2 := range b {
			if equal(elem, elem2) {
				return true
			}
		}
	}
	return false
}

type matchResult int8

const (
	matchSame matchResult = iota + 1
	matchDifferent
	matchDontKnow
)

// match performs a matching on u1 and u2. forImportingAction should be true
// when matching for the importing action, and false for every other action.
// isArray indicates which properties have type array.
func match(u1, u2 datastore.IRUser, action *state.Action, forImportingAction bool, anonIdents []string, isArray map[string]bool) matchResult {
	for _, p := range actionIdents(action, anonIdents) {
		v1 := u1.Properties[p]
		v2 := u2.Properties[p]
		v1IsZero, v2IsZero := zero(v1, isArray[p]), zero(v2, isArray[p])
		if v1IsZero && v2IsZero && !forImportingAction {
			return matchDontKnow
		}
		if v1IsZero || v2IsZero {
			continue
		}
		if isArray[p] {
			v1, v2 := v1.([]any), v2.([]any)
			if haveCommonValue(v1, v2) {
				return matchSame
			}
			return matchDifferent
		} else {
			if equal(v1, v2) {
				return matchSame
			}
			return matchDifferent
		}
	}
	return matchDontKnow
}

// merge merges the user 'a' into 'b'.
// This function deletes the user 'a' after updating 'b'.
// isArray indicates which properties have type array.
func merge(ctx context.Context, ws *state.Workspace, store *datastore.Store, a, b datastore.IRUser, isArray map[string]bool) error {

	telemetry.IncrementCounter(ctx, "merge", 1)

	// Serialize the row according to the schema.
	schema, ok := ws.Schemas["users"]
	if !ok {
		return errors.New("users schema not found")
	}
	datastore.SerializeRow(a.Properties, *schema)

	// TODO(Gianluca): should the user be normalized before being written on the
	// data store?

	// Update the user 'b' on the datastore.
	props := make(map[string]any, len(a.Properties))
	for p := range a.Properties {
		if isArray[p] {
			// TODO(Gianluca): support for "overwrite" mode: see https://github.com/open2b/chichi/issues/262.
			array, _ := a.Properties[p].([]any)
			if props, ok := b.Properties[p].([]any); ok {
				array = append(array, props...)
			}
			props[p] = deduplicate(array)
		} else {
			props[p] = a.Properties[p]
		}
	}
	err := store.UpdateUser(ctx, b, props, isArray)
	if err != nil {
		return fmt.Errorf("cannot update user on the store: %s", err)
	}

	// Delete the user 'a' on the datastore.
	err = store.DeleteUser(ctx, a.ID, isArray)
	if err != nil {
		return fmt.Errorf("cannot delete user on the store: %s", err)
	}

	return nil
}

func zero(v any, isArray bool) bool {
	// Keep in sync with the function 'zero' in 'apis/datastore'.
	if v == nil {
		return true
	}
	if isArray {
		return len(v.([]any)) == 0
	}
	return v == "" || v == 0
}
