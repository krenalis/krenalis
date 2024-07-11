//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"net/http"
	"strconv"

	"github.com/meergo/meergo/apis"
	"github.com/meergo/meergo/apis/errors"

	"github.com/google/uuid"
)

type user struct {
	*apisServer
}

// Events returns the events of a user, ordered from the most recent to the
// oldest.
func (user user) Events(_ http.ResponseWriter, r *http.Request) (any, error) {
	u, err := user.user(r)
	if err != nil {
		return nil, err
	}
	events, err := u.Events(r.Context(), 10)
	if err != nil {
		return nil, err
	}
	return map[string]any{"events": rawJSON(events)}, nil
}

// Identities returns the user identities of a user, and an estimate of their
// count without applying first and limit.
func (user user) Identities(_ http.ResponseWriter, r *http.Request) (any, error) {
	u, err := user.user(r)
	if err != nil {
		return nil, err
	}
	var first = 0
	var limit = 1000
	query := r.URL.Query()
	if v, ok := query["first"]; ok {
		first, err = strconv.Atoi(v[0])
		if err != nil {
			return nil, errors.BadRequest("first is not valid")
		}
	}
	if v, ok := query["limit"]; ok {
		limit, err = strconv.Atoi(v[0])
		if err != nil {
			return nil, errors.BadRequest("limit is not valid")
		}
	}
	identities, count, err := u.Identities(r.Context(), first, limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"identities": identities,
		"count":      count,
	}, nil
}

// Traits returns the traits of a user.
func (user user) Traits(_ http.ResponseWriter, r *http.Request) (any, error) {
	u, err := user.user(r)
	if err != nil {
		return nil, err
	}
	traits, err := u.Traits(r.Context())
	if err != nil {
		return nil, err
	}
	return map[string]any{"traits": rawJSON(traits)}, nil
}

func (user user) user(r *http.Request) (*apis.User, error) {
	ws, err := workspace{user.apisServer}.workspace(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("user")
	id, ok := parseUUID(v)
	if !ok {
		return nil, errors.NotFound("")
	}
	return ws.User(id)
}

// parseUUID parses an UUID in the form:
//
//	xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
//
// in case insensitive manner. If the UUID is not in that form, uuid.UUID{} and
// false are returned.
func parseUUID(s string) (uuid.UUID, bool) {
	if len(s) != 36 {
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, false
	}
	return id, true
}
