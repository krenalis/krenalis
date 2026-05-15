// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

type pipeline struct {
	*apisServer
}

// Delete deletes a pipeline.
func (pipeline pipeline) Delete(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateForbiddenBody(r); err != nil {
		return nil, err
	}
	p, err := pipeline.id(r)
	if err != nil {
		return nil, err
	}
	err = p.Delete(r.Context())
	return nil, err
}

// Run runs a pipeline.
func (pipeline pipeline) Run(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	p, err := pipeline.id(r)
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
	id, err := p.Run(r.Context(), body.Incremental)
	if err != nil {
		return nil, err
	}
	return map[string]int{"id": id}, nil
}

// ServeUI serves the UI of a pipeline.
func (pipeline pipeline) ServeUI(w http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	_, ws, _, err := pipeline.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errMissingWorkspace
	}
	id, ok := parseID(r.PathValue("id")) // ID of the pipeline
	if !ok {
		return nil, errors.BadRequest("identifier %q is not a valid pipeline identifier", r.PathValue("id"))
	}
	p, err := ws.Pipeline(id)
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
	return p.ServeUI(r.Context(), body.Event, body.FormatSettings)
}

// SetSchedulePeriod sets the schedule period of a pipeline.
func (pipeline pipeline) SetSchedulePeriod(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	p, err := pipeline.id(r)
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
	err = p.SetSchedulePeriod(r.Context(), body.Period)
	return nil, err
}

// SetStatus sets the status of a pipeline.
func (pipeline pipeline) SetStatus(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	p, err := pipeline.id(r)
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
	err = p.SetStatus(r.Context(), body.Enabled)
	return nil, err
}

// Update updates a pipeline.
func (pipeline pipeline) Update(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	p, err := pipeline.id(r)
	if err != nil {
		return nil, err
	}
	var body core.PipelineToSet
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = p.Update(r.Context(), body)
	return nil, err
}

// id authenticates the request and returns the pipeline identified by the 'id'
// path parameter.
func (pipeline pipeline) id(r *http.Request) (*core.Pipeline, error) {
	ws, err := workspace{pipeline.apisServer}.workspace(r)
	if err != nil {
		return nil, err
	}
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		return nil, errors.BadRequest("identifier %q is not a valid pipeline identifier", r.PathValue("id"))
	}
	return ws.Pipeline(id)
}
