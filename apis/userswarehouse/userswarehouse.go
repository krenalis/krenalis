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
		err := mergeUsers(ctx, ws, store, current, u, isArray)
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

// deduplicate deduplicate the elements in s, which must be a slice.
func deduplicate(s any) any {
	rv := reflect.ValueOf(s)
	unique := make(map[any]bool, rv.Len())
	out := reflect.MakeSlice(rv.Type(), 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		e := rv.Index(i).Interface()
		if !unique[e] {
			out = reflect.Append(out, reflect.ValueOf(e))
			unique[e] = true
		}
	}
	return out.Interface()
}

// elems returns the elements of the given slice as a []any.
func elems(slice any) []any {
	rv := reflect.ValueOf(slice)
	es := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		es[i] = rv.Index(i).Interface()
	}
	return es
}

// concatSlices returns a concatenation of the slices a and b.
func concatSlices(a, b any) any {
	aRv := reflect.ValueOf(a)
	bRv := reflect.ValueOf(b)
	l := aRv.Len() + bRv.Len()
	s := reflect.MakeSlice(aRv.Type(), l, l)
	reflect.Copy(s, aRv)
	reflect.Copy(s.Slice(aRv.Len(), l), bRv)
	return s.Interface()
}

// intersection returns the intersection between a and b.
// The elements in the returned slice are ordered as they appear in a.
func intersection[T comparable](a, b []T) []T {
	out := []T{}
	for _, v := range a {
		for _, w := range b {
			if w == v {
				out = append(out, v)
				break
			}
		}
	}
	return out
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
			if inters := intersection(elems(v1), elems(v2)); len(inters) > 0 {
				return matchSame
			} else {
				return matchDifferent
			}
		} else {
			if v1 == v2 {
				return matchSame
			}
			return matchDifferent
		}
	}
	return matchDontKnow
}

// mergeUsers merges the user 'current' into 'u'.
// This function deletes the 'current' user.
// isArray indicates which properties have type array.
func mergeUsers(ctx context.Context, ws *state.Workspace, store *datastore.Store, current, u datastore.IRUser, isArray map[string]bool) error {

	telemetry.IncrementCounter(ctx, "mergeUsers", 1)

	// Serialize the row according to the schema.
	schema, ok := ws.Schemas["users"]
	if !ok {
		return errors.New("users schema not found")
	}
	datastore.SerializeRow(current.Properties, *schema)

	// TODO(Gianluca): should the user be normalized before being written on the
	// data store?

	// Update the user 'u' on the datastore.
	props := make(map[string]any, len(current.Properties))
	for p := range current.Properties {
		if isArray[p] {
			// TODO(Gianluca): support for "overwrite" mode: see https://github.com/open2b/chichi/issues/262.
			props[p] = deduplicate(concatSlices(current.Properties[p], u.Properties[p]))
		} else {
			props[p] = current.Properties[p]
		}
	}
	err := store.UpdateUser(ctx, u, props, isArray)
	if err != nil {
		return fmt.Errorf("cannot update user on the store: %s", err)
	}

	// Delete the 'current' user on the datastore.
	err = store.DeleteUser(ctx, current.ID, isArray)
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
		return reflect.ValueOf(v).Len() == 0
	}
	return v == "" || v == 0
}
