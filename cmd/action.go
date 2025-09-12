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
	"github.com/meergo/meergo/core/json"
)

type action struct {
	*apisServer
}

// Delete deletes an action.
func (action action) Delete(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
	if err != nil {
		return nil, err
	}
	err = a.Delete(r.Context())
	return nil, err
}

// Execute executes an action.
func (action action) Execute(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Incremental *bool `json:"incremental"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	id, err := a.Execute(r.Context(), body.Incremental)
	if err != nil {
		return nil, err
	}
	return map[string]int{"id": id}, nil
}

// ServeUI serves the UI of an action.
func (action action) ServeUI(w http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
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
	return a.ServeUI(r.Context(), body.Event, body.FormatSettings)
}

// SetSchedulePeriod sets the schedule period of an action.
func (action action) SetSchedulePeriod(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Period *core.SchedulePeriod `json:"period"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = a.SetSchedulePeriod(r.Context(), body.Period)
	return nil, err
}

// SetStatus sets the status of an action.
func (action action) SetStatus(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
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

// Update updates an action.
func (action action) Update(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
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

func (action action) id(r *http.Request) (*core.Action, error) {
	ws, err := workspace{action.apisServer}.workspace(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("id")
	if v[0] == '+' {
		return nil, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return nil, errors.NotFound("")
	}
	return ws.Action(id)
}
