//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/apis/errors"
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
	return map[string]any{"events": json.RawMessage(events)}, nil
}

// Identities returns the users identities of a user, and an estimate of their
// count without applying first and limit.
func (user user) Identities(_ http.ResponseWriter, r *http.Request) (any, error) {
	u, err := user.user(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		First int
		Limit int
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	identities, count, err := u.Identities(r.Context(), body.First, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"identities": json.RawMessage(identities),
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
	return map[string]any{"traits": json.RawMessage(traits)}, nil
}

func (user user) user(r *http.Request) (*apis.User, error) {
	ws, err := workspace{user.apisServer}.workspace(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("user")
	if v[0] == '+' {
		return nil, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return nil, errors.NotFound("")
	}
	return ws.User(id)
}
