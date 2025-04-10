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
	"strings"
	"time"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/relvacode/iso8601"
)

type workspace struct {
	*apisServer
}

// Action returns an action of a connection.
func (workspace workspace) Action(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
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

// ActionErrors returns the action errors of the workspace.
func (workspace workspace) ActionErrors(_ http.ResponseWriter, r *http.Request) (any, error) {

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

	// Parse actions.
	var actions []int
	if ids, ok := q["actions"]; ok {
		actions = make([]int, len(ids))
		for i, id := range ids {
			actions[i], err = strconv.Atoi(id)
			if err != nil {
				return nil, errors.BadRequest("an action is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least an action must be provided")
	}

	// Parse step.
	var step *core.ActionStep
	if s, ok := q["step"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'step' parameter is allowed")
		}
		st, err := core.ParseActionStep(s[0])
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

	errs, err := ws.ActionErrors(r.Context(), start, end, actions, step, first, limit)
	if err != nil {
		return nil, err
	}

	return map[string][]core.ActionError{"errors": errs}, nil
}

// ActionMetricsPerDate returns metrics aggregated by day for a time interval
// between specified start and end dates.
func (workspace workspace) ActionMetricsPerDate(_ http.ResponseWriter, r *http.Request) (any, error) {

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

	// Parse actions.
	var actions []int
	if ids, ok := q["actions"]; ok {
		actions = make([]int, len(ids))
		for i, id := range ids {
			actions[i], err = strconv.Atoi(id)
			if err != nil {
				return nil, errors.BadRequest("an action is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least an action must be provided")
	}

	metrics, err := ws.ActionMetricsPerDate(r.Context(), start, end, actions)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  metrics.Start,
		"end":    metrics.End,
		"passed": metrics.Passed,
		"failed": metrics.Failed}, nil
}

// ActionMetricsPerDay returns the action metrics for a specified number of
// days.
func (workspace workspace) ActionMetricsPerDay(_ http.ResponseWriter, r *http.Request) (any, error) {

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

	// Parse actions.
	var actions []int
	if ids, ok := q["actions"]; ok {
		actions = make([]int, len(ids))
		for i, id := range ids {
			actions[i], err = strconv.Atoi(id)
			if err != nil {
				return nil, errors.BadRequest("an 'action' parameter is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least an action must be provided")
	}

	metrics, err := ws.ActionMetricsPerTimeUnit(r.Context(), days, core.Day, actions)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  metrics.Start,
		"end":    metrics.End,
		"passed": metrics.Passed,
		"failed": metrics.Failed}, nil
}

// ActionMetricsPerHour returns the action metrics for a specified number of
// hours.
func (workspace workspace) ActionMetricsPerHour(_ http.ResponseWriter, r *http.Request) (any, error) {

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

	// Parse actions.
	var actions []int
	if ids, ok := q["actions"]; ok {
		actions = make([]int, len(ids))
		for i, id := range ids {
			actions[i], err = strconv.Atoi(id)
			if err != nil {
				return nil, errors.BadRequest("an action is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least an action must be provided")
	}

	metrics, err := ws.ActionMetricsPerTimeUnit(r.Context(), hours, core.Hour, actions)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  metrics.Start,
		"end":    metrics.End,
		"passed": metrics.Passed,
		"failed": metrics.Failed}, nil
}

// ActionMetricsPerMinute returns the action metrics for a specified number of
// minutes.
func (workspace workspace) ActionMetricsPerMinute(_ http.ResponseWriter, r *http.Request) (any, error) {

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

	// Parse actions.
	var actions []int
	if ids, ok := q["actions"]; ok {
		actions = make([]int, len(ids))
		for i, id := range ids {
			actions[i], err = strconv.Atoi(id)
			if err != nil {
				return nil, errors.BadRequest("an action is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least an action must be provided")
	}

	metrics, err := ws.ActionMetricsPerTimeUnit(r.Context(), minutes, core.Minute, actions)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  metrics.Start,
		"end":    metrics.End,
		"passed": metrics.Passed,
		"failed": metrics.Failed}, nil
}

// Connection returns a connection of a workspace.
func (workspace workspace) Connection(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
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
	return ws.Connection(r.Context(), id)
}

// Connections returns the connections of a workspace.
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
		Size   *int         `json:"size"`
		Filter *core.Filter `json:"filter"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	var size = 10
	if body.Size != nil {
		size = *body.Size
	}
	id, err := ws.CreateEventListener(size, body.Filter)
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id}, nil
}

// Delete deletes a workspace with all its connections.
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
	properties := q["properties"]
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
	events, _ := types.Marshal(evts, types.Array(events.Schema))

	return map[string]any{"events": events}, nil
}

// Identities returns the user identities of a user, and an estimate of their
// total number without applying first and limit.
func (workspace workspace) Identities(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	userID := r.PathValue("id")
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
	identities, total, err := ws.Identities(r.Context(), userID, first, limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"identities": identities,
		"total":      total,
	}, nil
}

// UserPropertiesSuitableAsIdentifiers returns the properties of the "users"
// schema that can be used as identifiers in the Identity Resolution.
func (workspace workspace) UserPropertiesSuitableAsIdentifiers(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return ws.UserPropertiesSuitableAsIdentifiers(), nil
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
	startTime, endTime, err := ws.LatestIdentityResolution(r.Context())
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"startTime": startTime,
		"endTime":   endTime,
	}, nil
}

// LatestUserSchemaUpdate returns information about the latest update of the
// user schema of a workspace.
func (workspace workspace) LatestUserSchemaUpdate(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	startTime, endTime, updateError, err := ws.LatestUserSchemaUpdate(r.Context())
	if err != nil {
		return nil, err
	}
	res := map[string]any{
		"startTime": startTime,
		"endTime":   endTime,
	}
	if updateError != "" {
		res["error"] = updateError
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

// Execution returns the execution of an action in a workspace.
func (workspace workspace) Execution(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("id")
	if v[0] == '+' {
		return nil, errors.BadRequest("identifier %q is not a valid execution identifier", v)
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return nil, errors.BadRequest("identifier %q is not a valid execution identifier", v)
	}
	return ws.Execution(r.Context(), id)
}

// Executions returns the executions of the actions of a workspace.
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

// PreviewUserSchemaUpdate previews a user schema update and returns the queries
// that would be executed.
func (workspace workspace) PreviewUserSchemaUpdate(_ http.ResponseWriter, r *http.Request) (any, error) {
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
	queries, err := ws.PreviewUserSchemaUpdate(r.Context(), body.Schema, body.RePaths)
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
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
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
		Settings json.Value `json:"settings"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.TestWarehouseUpdate(r.Context(), body.Settings)
	return nil, err
}

// Traits returns the traits of a user.
func (workspace workspace) Traits(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	userID := r.PathValue("id")
	traits, err := ws.Traits(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"traits": traits}, nil
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

// UpdateUserSchema updates the user schema of a workspace.
func (workspace workspace) UpdateUserSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
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
	err = ws.UpdateUserSchema(r.Context(), body.Schema, body.PrimarySources, body.RePaths)
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
		Mode                         core.WarehouseMode `json:"mode"`
		CancelIncompatibleOperations bool               `json:"cancelIncompatibleOperations"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.UpdateWarehouse(r.Context(), body.Mode, body.Settings, body.CancelIncompatibleOperations)
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

// UserEvents returns the events of a user.
func (workspace workspace) UserEvents(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	// Parse the user ID.
	id := r.PathValue("id")
	if _, ok := types.ParseUUID(id); !ok {
		return nil, errors.BadRequest("value %q is not a valid user identifier", id)
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
	properties, ok := q["properties"]
	if !ok {
		return nil, errors.BadRequest("no properties were provided to return")
	}

	filter := &core.Filter{
		Logical: core.OpAnd,
		Conditions: []core.FilterCondition{
			{
				Property: "user",
				Operator: core.OpIs,
				Values:   []string{id},
			},
		},
	}

	evts, err := ws.Events(r.Context(), properties, filter, "timestamp", true, 0, limit)
	if err != nil {
		return nil, err
	}

	events, _ := types.Marshal(evts, types.Array(events.Schema))

	return map[string]any{"events": events}, nil
}

// UserSchema returns the user schema of a workspace.
func (workspace workspace) UserSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return ws.UserSchema, nil
}

// Users returns the users, the user schema of a workspace, and an estimate of
// their total number without applying first and limit.
func (workspace workspace) Users(w http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	// Read and parse the parameters from the query string.
	q := r.URL.Query()
	properties := q["properties"]
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

	users, schema, total, err := ws.Users(r.Context(), properties, filter, order, orderDesc, first, limit)
	if err != nil {
		return nil, err
	}
	w.Header().Set("Content-Type", "application/json")
	b := newBodyWriter(w)
	b.writeString(`{"users":[`)
	for i, user := range users {
		if i > 0 {
			b.writeByte(',')
		}
		b.writeString(`{"id":"`)
		b.writeString(user.ID)
		b.writeString(`","sourcesLastUpdate":"`)
		buf := b.availableBuffer()
		b.write(user.LastChangeTime.AppendFormat(buf, time.RFC3339Nano))
		b.writeString(`","traits":`)
		s, _ := types.Marshal(user.Traits, schema)
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

// Warehouse returns the type and settings of the data warehouse for a
// workspace.
func (workspace workspace) Warehouse(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	name, settings := ws.Warehouse()
	return map[string]any{"name": name, "settings": settings}, nil
}

// workspace returns the workspace.
func (workspace workspace) workspace(r *http.Request) (*core.Workspace, error) {
	_, ws, err := workspace.credentials(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errors.Forbidden("access to the workspace is not allowed")
	}
	return ws, nil
}
