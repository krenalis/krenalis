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

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/json"
)

type action struct {
	*apisServer
}

// Delete deletes an action.
func (action action) Delete(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.action(r)
	if err != nil {
		return nil, err
	}
	err = a.Delete(r.Context())
	return nil, err
}

// ServeUI serves the UI of an action.
func (action action) ServeUI(w http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.action(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Event          string     `json:"event"`
		FormatSettings json.Value `json:"formatSettings"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	ui, err := a.ServeUI(r.Context(), body.Event, body.FormatSettings)
	if err != nil {
		return nil, err
	}
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(ui)
	return nil, nil
}

// Update updates an action.
func (action action) Update(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.action(r)
	if err != nil {
		return nil, err
	}
	var body core.ActionToSet
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = a.Update(r.Context(), body)
	return nil, err
}

// SetStatus sets the status of an action.
func (action action) SetStatus(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.action(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = a.SetStatus(r.Context(), body.Enabled)
	return nil, err
}

// SetSchedulePeriod sets the schedule period of an action.
func (action action) SetSchedulePeriod(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.action(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		SchedulePeriod core.SchedulePeriod `json:"schedulePeriod"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = a.SetSchedulePeriod(r.Context(), body.SchedulePeriod)
	return nil, err
}

// Execute executes an action.
func (action action) Execute(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.action(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Reload bool `json:"reload"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	return a.Execute(r.Context(), body.Reload)
}

func (action action) action(r *http.Request) (*core.Action, error) {
	connection, err := connection{action.apisServer}.connection(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("action")
	if v[0] == '+' {
		return nil, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return nil, errors.NotFound("")
	}
	return connection.Action(r.Context(), id)
}
