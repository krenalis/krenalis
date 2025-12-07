// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/tools/errors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"

	"github.com/relvacode/iso8601"
)

type workspace struct {
	*apisServer
}

// AlterProfileSchema alters the profile schema of a workspace.
func (workspace workspace) AlterProfileSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Schema         types.Type     `json:"schema"`
		PrimarySources map[string]int `json:"primarySources"`
		RePaths        map[string]any `json:"rePaths"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.AlterProfileSchema(r.Context(), body.Schema, body.PrimarySources, body.RePaths)
	return nil, err
}

// AuthToken returns an authorization token, given an authorization code and
// the redirection URI used to obtain that code, that can be used to add a new
// connection to the workspace for the specified connector.
func (workspace workspace) AuthToken(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	query := r.URL.Query()
	connector := query.Get("connector")
	redirectURI := query.Get("redirectURI")
	authCode := query.Get("authCode")
	return ws.AuthToken(r.Context(), connector, redirectURI, authCode)
}

// Connection returns a connection of the current workspace.
func (workspace workspace) Connection(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	id, ok := parseID(r.PathValue("id")) // ID of the connection
	if !ok {
		return nil, errors.BadRequest("connection identifier %q is not valid", r.PathValue("id"))
	}
	return ws.Connection(r.Context(), id)
}

// Connections returns the connections of the current workspace.
func (workspace workspace) Connections(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return map[string]any{"connections": ws.Connections()}, nil
}

// CreateConnection creates a connection for a workspace.
func (workspace workspace) CreateConnection(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		core.ConnectionToAdd
		AuthToken string `json:"authToken"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if body.Settings != nil && body.Settings.IsNull() {
		body.Settings = nil
	}
	return ws.CreateConnection(r.Context(), body.ConnectionToAdd, body.AuthToken)
}

// CreateEventListener creates an event listener for a workspace that listens to
// events.
func (workspace workspace) CreateEventListener(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Connection *int         `json:"connection"`
		Size       *int         `json:"size"`
		Filter     *core.Filter `json:"filter"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	connection := 0
	if body.Connection != nil {
		connection = *body.Connection
	}
	size := 10
	if body.Size != nil {
		size = *body.Size
	}
	id, err := ws.CreateEventListener(connection, size, body.Filter)
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id}, nil
}

// Delete deletes the current workspace with all its connections.
func (workspace workspace) Delete(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	err = ws.Delete(r.Context())
	return nil, err
}

// DeleteEventListener deletes an event listener of a workspace.
func (workspace workspace) DeleteEventListener(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	ws.DeleteEventListener(r.PathValue("id"))
	return nil, nil
}

// Events returns the events.
func (workspace workspace) Events(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	// Read and parse the parameters from the query string.
	q := r.URL.Query()
	properties := splitQueryParameters(q["properties"])
	var filter *core.Filter
	if f := q.Get("filter"); f != "" {
		err := json.Unmarshal([]byte(f), &filter)
		if err != nil {
			return nil, errors.BadRequest("invalid filter")
		}
		if filter == nil {
			return nil, errors.BadRequest("filter cannot be null")
		}
	}
	order := q.Get("order")
	orderDesc := q.Get("orderDesc") == "true"
	var first int
	if f := q.Get("first"); f != "" {
		first, err = strconv.Atoi(f)
		if err != nil {
			return nil, errors.BadRequest("invalid first")
		}
	}
	var limit int
	if l := q.Get("limit"); l != "" {
		limit, err = strconv.Atoi(l)
		if err != nil {
			return nil, errors.BadRequest("invalid limit")
		}
	}

	evts, err := ws.Events(r.Context(), properties, filter, order, orderDesc, first, limit)
	if err != nil {
		return nil, err
	}
	events, _ := types.Marshal(evts, types.Array(core.EventSchema()))

	return map[string]any{"events": events}, nil
}

// Identities returns the identities of a profile, and an estimate of their
// total number without applying first and limit.
func (workspace workspace) Identities(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	mpid := r.PathValue("mpid")
	var first = 0
	var limit = 100
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
	identities, total, err := ws.Identities(r.Context(), mpid, first, limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"identities": identities,
		"total":      total,
	}, nil
}

// ProfilePropertiesSuitableAsIdentifiers returns the properties of the profile
// schema that can be used as identifiers in the Identity Resolution.
func (workspace workspace) ProfilePropertiesSuitableAsIdentifiers(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, ws, _, err := workspace.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errMissingWorkspace
	}
	return ws.ProfilePropertiesSuitableAsIdentifiers(), nil
}

// IngestEvents ingests a batch of events.
func (workspace workspace) IngestEvents(w http.ResponseWriter, r *http.Request) (any, error) {
	// Removes the headers that were set earlier, as ServeEvents handles the response fully.
	w.Header().Del("Cache-Control")
	w.Header().Del("Pragma")
	w.Header().Del("Expires")
	workspace.core.ServeEvents(w, r)
	return nil, nil
}

// LatestIdentityResolution returns information about the latest Identity
// Resolution of a workspace.
func (workspace workspace) LatestIdentityResolution(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	startTime, endTime, err := ws.LatestIdentityResolution()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"startTime": startTime,
		"endTime":   endTime,
	}, nil
}

// LatestAlterProfileSchema returns information about the latest altering of the
// profile schema of a workspace.
func (workspace workspace) LatestAlterProfileSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	startTime, endTime, alterError, err := ws.LatestAlterProfileSchema()
	if err != nil {
		return nil, err
	}
	res := map[string]any{
		"startTime": startTime,
		"endTime":   endTime,
	}
	if alterError != "" {
		res["error"] = alterError
	} else {
		res["error"] = nil
	}
	return res, nil
}

// ListenedEvents returns the events listen to by a specified listener and the
// number of omitted events.
func (workspace workspace) ListenedEvents(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	listenedEvents, omitted, err := ws.ListenedEvents(r.PathValue("id"))
	if err != nil {
		return nil, err
	}
	events := make([]json.Value, len(listenedEvents))
	for i, event := range listenedEvents {
		events[i] = event
	}
	return map[string]any{
		"events":  events,
		"omitted": omitted,
	}, nil
}

// Execution returns the execution of a pipeline in the current workspace.
func (workspace workspace) Execution(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	id, ok := parseID(r.PathValue("id")) // ID of the execution.
	if !ok {
		return nil, errors.BadRequest("identifier %q is not a valid execution identifier", r.PathValue("id"))
	}
	return ws.Execution(r.Context(), id)
}

// Executions returns the executions of the pipelines of the current workspace.
func (workspace workspace) Executions(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	executions, err := ws.Executions(r.Context())
	if err != nil {
		return nil, err
	}
	return map[string]any{"executions": executions}, nil
}

// IdentityResolutionSettings returns the identity resolution settings of the
// workspace.
func (workspace workspace) IdentityResolutionSettings(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	runOnBatchImport, identifiers := ws.IdentityResolutionSettings()
	return map[string]any{
		"runOnBatchImport": runOnBatchImport,
		"identifiers":      identifiers,
	}, nil
}

// Pipeline returns a pipeline of a connection.
func (workspace workspace) Pipeline(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	id, ok := parseID(r.PathValue("id")) // ID of the pipeline.
	if !ok {
		return nil, errors.BadRequest("identifier %q is not a valid pipeline identifier", r.PathValue("id"))
	}
	return ws.Pipeline(id)
}

// PipelineErrors returns the pipeline errors of the workspace.
func (workspace workspace) PipelineErrors(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	// Parse start.
	s := r.PathValue("start")
	start, err := iso8601.ParseString(s)
	if err != nil {
		return nil, errors.BadRequest("start is not valid")
	}

	// Parse end.
	e := r.PathValue("end")
	end, err := iso8601.ParseString(e)
	if err != nil {
		return nil, errors.BadRequest("end is not valid")
	}

	q := r.URL.Query()

	// Parse pipelines.
	var pipelines []int
	if ids := splitQueryParameters(q["pipelines"]); len(ids) > 0 {
		pipelines = make([]int, len(ids))
		for i, id := range ids {
			var ok bool
			pipelines[i], ok = parseID(id)
			if !ok {
				return nil, errors.BadRequest("a pipeline is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least a pipeline must be provided")
	}

	// Parse step.
	var step *core.PipelineStep
	if s, ok := q["step"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'step' parameter is allowed")
		}
		st, err := core.ParsePipelineStep(s[0])
		if err != nil {
			return nil, errors.BadRequest("'step' parameter is not valid")
		}
		step = &st
	}

	// Parse first and limit.
	first, limit := 0, 100
	if s, ok := q["first"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'first' parameter is allowed")
		}
		first, err = strconv.Atoi(s[0])
		if err != nil {
			return nil, errors.BadRequest("'first' parameter is not valid")
		}
	}
	if s, ok := q["limit"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'limit' parameters is allowed")
		}
		limit, err = strconv.Atoi(s[0])
		if err != nil {
			return nil, errors.BadRequest("'limit' parameter is not valid")
		}
	}

	errs, err := ws.PipelineErrors(r.Context(), start, end, pipelines, step, first, limit)
	if err != nil {
		return nil, err
	}

	return map[string][]core.PipelineError{"errors": errs}, nil
}

// PipelineMetricsPerDate returns metrics aggregated by day for a time interval
// between specified start and end dates.
func (workspace workspace) PipelineMetricsPerDate(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	// Parse start.
	s := r.PathValue("start")
	start, err := iso8601.ParseString(s)
	if err != nil {
		return nil, errors.BadRequest("start is not valid")
	}

	// Parse end.
	e := r.PathValue("end")
	end, err := iso8601.ParseString(e)
	if err != nil {
		return nil, errors.BadRequest("end is not valid")
	}

	q := r.URL.Query()

	// Parse pipelines.
	var pipelines []int
	if ids := splitQueryParameters(q["pipelines"]); len(ids) > 0 {
		pipelines = make([]int, len(ids))
		for i, id := range ids {
			var ok bool
			pipelines[i], ok = parseID(id)
			if !ok {
				return nil, errors.BadRequest("a pipeline is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least a pipeline must be provided")
	}

	metrics, err := ws.PipelineMetricsPerDate(r.Context(), start, end, pipelines)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  metrics.Start,
		"end":    metrics.End,
		"passed": metrics.Passed,
		"failed": metrics.Failed}, nil
}

// PipelineMetricsPerDay returns the pipeline metrics for a specified number of
// days.
func (workspace workspace) PipelineMetricsPerDay(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	// Parse days.
	d := r.PathValue("days")
	days, err := strconv.Atoi(d)
	if err != nil {
		return nil, errors.BadRequest("days is not valid")
	}

	q := r.URL.Query()

	// Parse pipelines.
	var pipelines []int
	if ids := splitQueryParameters(q["pipelines"]); len(ids) > 0 {
		pipelines = make([]int, len(ids))
		for i, id := range ids {
			var ok bool
			pipelines[i], ok = parseID(id)
			if !ok {
				return nil, errors.BadRequest("an 'pipeline' parameter is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least a pipeline must be provided")
	}

	metrics, err := ws.PipelineMetricsPerTimeUnit(r.Context(), days, core.Day, pipelines)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  metrics.Start,
		"end":    metrics.End,
		"passed": metrics.Passed,
		"failed": metrics.Failed}, nil
}

// PipelineMetricsPerHour returns the pipeline metrics for a specified number of
// hours.
func (workspace workspace) PipelineMetricsPerHour(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	// Parse hours.
	h := r.PathValue("hours")
	hours, err := strconv.Atoi(h)
	if err != nil {
		return nil, errors.BadRequest("hours is not valid")
	}

	q := r.URL.Query()

	// Parse pipelines.
	var pipelines []int
	if ids := splitQueryParameters(q["pipelines"]); len(ids) > 0 {
		pipelines = make([]int, len(ids))
		for i, id := range ids {
			var ok bool
			pipelines[i], ok = parseID(id)
			if !ok {
				return nil, errors.BadRequest("a pipeline is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least a pipeline must be provided")
	}

	metrics, err := ws.PipelineMetricsPerTimeUnit(r.Context(), hours, core.Hour, pipelines)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  metrics.Start,
		"end":    metrics.End,
		"passed": metrics.Passed,
		"failed": metrics.Failed}, nil
}

// PipelineMetricsPerMinute returns the pipeline metrics for a specified number of
// minutes.
func (workspace workspace) PipelineMetricsPerMinute(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	// Parse minutes.
	m := r.PathValue("minutes")
	minutes, err := strconv.Atoi(m)
	if err != nil {
		return nil, errors.BadRequest("minutes is not valid")
	}

	q := r.URL.Query()

	// Parse pipelines.
	var pipelines []int
	if ids := splitQueryParameters(q["pipelines"]); len(ids) > 0 {
		pipelines = make([]int, len(ids))
		for i, id := range ids {
			var ok bool
			pipelines[i], ok = parseID(id)
			if !ok {
				return nil, errors.BadRequest("a pipeline is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least a pipeline must be provided")
	}

	metrics, err := ws.PipelineMetricsPerTimeUnit(r.Context(), minutes, core.Minute, pipelines)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  metrics.Start,
		"end":    metrics.End,
		"passed": metrics.Passed,
		"failed": metrics.Failed}, nil
}

// PreviewAlterProfileSchema provides a preview of an alter profile schema
// operation by returning the queries that would be executed on the warehouse to
// perform a given alter schema.
func (workspace workspace) PreviewAlterProfileSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Schema  types.Type     `json:"schema"`
		RePaths map[string]any `json:"rePaths"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	queries, err := ws.PreviewAlterProfileSchema(r.Context(), body.Schema, body.RePaths)
	if err != nil {
		return nil, err
	}
	return map[string]any{"queries": queries}, nil
}

// RepairWarehouse repairs the database objects needed by Meergo on a
// workspace's data warehouse.
func (workspace workspace) RepairWarehouse(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	err = ws.RepairWarehouse(r.Context())
	return nil, err
}

// ServeUI serves the user interface for a connector.
func (workspace workspace) ServeUI(w http.ResponseWriter, r *http.Request) (any, error) {
	_, ws, _, err := workspace.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errMissingWorkspace
	}
	var body struct {
		Connector string     `json:"connector"`
		Event     string     `json:"event"`
		Settings  json.Value `json:"settings"`
		Role      string     `json:"role"`
		AuthToken string     `json:"authToken"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if strings.HasSuffix(r.URL.Path, "/ui") {
		body.Event = "load"
		body.Settings = nil
	}
	var role core.Role
	switch body.Role {
	case "Source":
		role = core.Source
	case "Destination":
		role = core.Destination
	default:
		return nil, errors.BadRequest("unexpected connection role '%s'", body.Role)
	}
	return ws.ServeUI(r.Context(), body.Event, body.Settings, body.Connector, role, body.AuthToken)
}

// StartIdentityResolution starts an Identity Resolution operation that resolves
// the identities of the workspace.
func (workspace workspace) StartIdentityResolution(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	err = ws.StartIdentityResolution(r.Context())
	return nil, err
}

// TestWarehouseUpdate tests a warehouse update.
func (workspace workspace) TestWarehouseUpdate(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Settings    json.Value `json:"settings"`
		MCPSettings json.Value `json:"mcpSettings"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if body.MCPSettings != nil && body.MCPSettings.IsNull() {
		body.MCPSettings = nil
	}
	err = ws.TestWarehouseUpdate(r.Context(), body.Settings, body.MCPSettings)
	return nil, err
}

// Attributes returns the attributes of a profile, given its MPID.
func (workspace workspace) Attributes(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	mpid := r.PathValue("mpid")
	attributes, err := ws.Attributes(r.Context(), mpid)
	if err != nil {
		return nil, err
	}
	return map[string]any{"attributes": attributes}, nil
}

// Update updates the name and the displayed properties of a workspace.
func (workspace workspace) Update(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name          string             `json:"name"`
		UIPreferences core.UIPreferences `json:"uiPreferences"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.Update(r.Context(), body.Name, body.UIPreferences)
	return nil, err
}

// UpdateIdentityResolutionSettings updates the identity resolution settings of
// the workspace.
func (workspace workspace) UpdateIdentityResolutionSettings(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		RunOnBatchImport bool     `json:"runOnBatchImport"`
		Identifiers      []string `json:"identifiers"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.UpdateIdentityResolutionSettings(r.Context(), body.RunOnBatchImport, body.Identifiers)
	return nil, err
}

// UpdateWarehouse updates the warehouse of a workspace.
func (workspace workspace) UpdateWarehouse(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Settings                     json.Value         `json:"settings"`
		MCPSettings                  json.Value         `json:"mcpSettings"`
		Mode                         core.WarehouseMode `json:"mode"`
		CancelIncompatibleOperations bool               `json:"cancelIncompatibleOperations"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if body.MCPSettings != nil && body.MCPSettings.IsNull() {
		body.MCPSettings = nil
	}
	err = ws.UpdateWarehouse(r.Context(), body.Mode, body.Settings, body.MCPSettings, body.CancelIncompatibleOperations)
	return nil, err
}

// UpdateWarehouseMode updates the mode of the data warehouse for a workspace.
func (workspace workspace) UpdateWarehouseMode(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Mode                         core.WarehouseMode `json:"mode"`
		CancelIncompatibleOperations bool               `json:"cancelIncompatibleOperations"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.UpdateWarehouseMode(r.Context(), body.Mode, body.CancelIncompatibleOperations)
	return nil, err
}

// ProfileEvents returns the events of a profile.
func (workspace workspace) ProfileEvents(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	// Parse the MPID.
	mpid := r.PathValue("mpid")
	if _, ok := types.ParseUUID(mpid); !ok {
		return nil, errors.BadRequest("value %q is not a valid MPID", mpid)
	}

	q := r.URL.Query()

	// Parse limit.
	limit := 100
	if v, ok := q["limit"]; ok {
		limit, err = strconv.Atoi(v[0])
		if err != nil {
			return nil, errors.BadRequest("limit is not valid")
		}
	}

	// Parse the properties.
	properties := splitQueryParameters(q["properties"])

	filter := &core.Filter{
		Logical: core.OpAnd,
		Conditions: []core.FilterCondition{
			{
				Property: "mpid",
				Operator: core.OpIs,
				Values:   []string{mpid},
			},
		},
	}

	evts, err := ws.Events(r.Context(), properties, filter, "timestamp", true, 0, limit)
	if err != nil {
		return nil, err
	}

	events, _ := types.Marshal(evts, types.Array(core.EventSchema()))

	return map[string]any{"events": events}, nil
}

// ProfileSchema returns the profile schema of a workspace.
func (workspace workspace) ProfileSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return ws.ProfileSchema, nil
}

// Profiles returns the profiles, the profile schema of a workspace, and an
// estimate of their total number without applying first and limit.
func (workspace workspace) Profiles(w http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	// Read and parse the parameters from the query string.
	q := r.URL.Query()
	properties := splitQueryParameters(q["properties"])
	var filter *core.Filter
	if f := q.Get("filter"); f != "" {
		err := json.Unmarshal([]byte(f), &filter)
		if err != nil {
			return nil, errors.BadRequest("invalid filter")
		}
		if filter == nil {
			return nil, errors.BadRequest("filter cannot be null")
		}
	}
	order := q.Get("order")
	orderDesc := q.Get("orderDesc") == "true"
	var first int
	if f := q.Get("first"); f != "" {
		first, err = strconv.Atoi(f)
		if err != nil {
			return nil, errors.BadRequest("invalid first")
		}
	}
	var limit int
	if l := q.Get("limit"); l != "" {
		limit, err = strconv.Atoi(l)
		if err != nil {
			return nil, errors.BadRequest("invalid limit")
		}
	} else {
		limit = 100
	}

	profiles, schema, total, err := ws.Profiles(r.Context(), properties, filter, order, orderDesc, first, limit)
	if err != nil {
		return nil, err
	}
	w.Header().Set("Content-Type", "application/json")
	b := newBodyWriter(w)
	b.writeString(`{"profiles":[`)
	for i, profile := range profiles {
		if i > 0 {
			b.writeByte(',')
		}
		b.writeString(`{"mpid":"`)
		b.writeString(profile.MPID)
		b.writeString(`","updatedAt":"`)
		buf := b.availableBuffer()
		b.write(profile.UpdatedAt.AppendFormat(buf, time.RFC3339Nano))
		b.writeString(`","attributes":`)
		s, _ := types.Marshal(profile.Attributes, schema)
		b.write(s)
		b.writeByte('}')
	}
	b.writeString(`],"schema":`)
	buf, _ := schema.MarshalJSON()
	b.write(buf)
	b.writeString(`,"total":`)
	buf = b.availableBuffer()
	b.write(strconv.AppendInt(buf, int64(total), 10))
	b.writeByte('}')
	b.flush()

	return nil, nil
}

// Warehouse returns the platform, settings and MCP settings of the data
// warehouse for a workspace.
func (workspace workspace) Warehouse(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	platform, settings, mcpSettings := ws.Warehouse()
	return map[string]any{
		"platform":    platform,
		"settings":    settings,
		"mcpSettings": mcpSettings,
	}, nil
}

// workspace returns the current workspace.
func (workspace workspace) workspace(r *http.Request) (*core.Workspace, error) {
	_, ws, err := workspace.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errMissingWorkspace
	}
	return ws, nil
}
