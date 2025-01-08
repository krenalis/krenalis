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

// ActionErrors returns the action errors of the workspace.
func (workspace workspace) ActionErrors(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	q := r.URL.Query()

	// Parse start.
	var start time.Time
	if s, ok := q["start"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'start' parameter is allowed")
		}
		start, err = iso8601.ParseString(s[0])
		if err != nil {
			return nil, errors.BadRequest("'start' parameter is not a valid time")
		}
		start = start.UTC()
	} else {
		return nil, errors.BadRequest("'start' parameter is missing")
	}

	// Parse end.
	var end time.Time
	if s, ok := q["end"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'end' parameter is allowed")
		}
		end, err = iso8601.ParseString(s[0])
		if err != nil {
			return nil, errors.BadRequest("'end' parameter is not a valid time")
		}
		end = end.UTC()
	} else {
		end = time.Now().UTC()
	}

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

	q := r.URL.Query()

	// Parse start.
	var start time.Time
	if s, ok := q["start"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'start' parameter is allowed")
		}
		start, err = iso8601.ParseString(s[0])
		if err != nil {
			return nil, errors.BadRequest("'start' parameter is not valid")
		}
	}

	// Parse end.
	var end time.Time
	if s, ok := q["end"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'end' parameter is allowed")
		}
		end, err = iso8601.ParseString(s[0])
		if err != nil {
			return nil, errors.BadRequest("'end' parameter is not valid")
		}
	}

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

	q := r.URL.Query()

	// Parse days.
	days := 30
	if s, ok := q["days"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'days' parameter is allowed")
		}
		days, err = strconv.Atoi(s[0])
		if err != nil {
			return nil, errors.BadRequest("'days' parameter is not valid")
		}
	}

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

	q := r.URL.Query()

	// Parse hours.
	hours := 48
	if s, ok := q["hours"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'hours' parameter is allowed")
		}
		hours, err = strconv.Atoi(s[0])
		if err != nil {
			return nil, errors.BadRequest("'hours' parameter is not valid")
		}
	}

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

	q := r.URL.Query()

	// Parse minutes.
	minutes := 60
	if s, ok := q["minutes"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'minutes' parameter is allowed")
		}
		minutes, err = strconv.Atoi(s[0])
		if err != nil {
			return nil, errors.BadRequest("'minutes' parameter is not valid")
		}
	}

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
	v := r.PathValue("connection")
	if v[0] == '+' {
		return nil, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return nil, errors.NotFound("")
	}
	return ws.Connection(id)
}

// Connections returns the connections of a workspace.
func (workspace workspace) Connections(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return ws.Connections(), nil
}

// CreateConnection creates a connection for a workspace.
func (workspace workspace) CreateConnection(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Connection core.ConnectionToAdd `json:"connection"`
		OAuthToken string               `json:"oauthToken"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	return ws.CreateConnection(r.Context(), body.Connection, body.OAuthToken)
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
	ws.DeleteEventListener(r.PathValue("listener"))
	return nil, nil
}

// Events returns the events.
func (workspace workspace) Events(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Properties []string     `json:"properties"`
		Filter     *core.Filter `json:"filter"`
		Order      string       `json:"order"`
		OrderDesc  bool         `json:"orderDesc"`
		First      int          `json:"first"`
		Limit      int          `json:"limit"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	evts, err := ws.Events(r.Context(), body.Properties, body.Filter, body.Order, body.OrderDesc, body.First, body.Limit)
	if err != nil {
		return nil, err
	}
	events, _ := types.Marshal(evts, types.Array(events.Schema))
	return map[string]any{"events": events}, nil
}

// Identities returns the user identities of a user, and an estimate of their
// count without applying first and limit.
func (workspace workspace) Identities(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	user := r.PathValue("user")
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
	identities, count, err := ws.Identities(r.Context(), user, first, limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"identities": identities,
		"count":      count,
	}, nil
}

// IdentifiersSchema returns the properties of the "users" schema that can be
// used as identifiers in the Identity Resolution.
func (workspace workspace) IdentifiersSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return ws.IdentifiersSchema(), nil
}

// LastIdentityResolution returns information about the last Identity
// Resolution of a workspace.
func (workspace workspace) LastIdentityResolution(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	startTime, endTime, err := ws.LastIdentityResolution(r.Context())
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"startTime": startTime,
		"endTime":   endTime,
	}, nil
}

// ListenedEvents returns the events listen to by a specified listener and the
// number of discarded events.
func (workspace workspace) ListenedEvents(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	listenedEvents, discarded, err := ws.ListenedEvents(r.PathValue("listener"))
	if err != nil {
		return nil, err
	}
	events := make([]json.Value, len(listenedEvents))
	for i, event := range listenedEvents {
		events[i] = event
	}
	return map[string]any{
		"events":    events,
		"discarded": discarded,
	}, nil
}

// OAuthToken returns an OAuth token, given an OAuth authorization code and the
// redirection URI used to obtain that code, that can be used to add a new
// connection to the workspace for the specified connector.
func (workspace workspace) OAuthToken(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		OAuthCode   string `json:"oauthCode"`
		RedirectURI string `json:"redirectURI"`
		Connector   string `json:"connector"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	return ws.OAuthToken(r.Context(), body.OAuthCode, body.RedirectURI, body.Connector)
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
		Connector  string     `json:"connector"`
		Event      string     `json:"event"`
		Settings   json.Value `json:"settings"`
		Role       string     `json:"role"`
		OAuthToken string     `json:"oauthToken"`
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
	ui, err := ws.ServeUI(r.Context(), body.Event, body.Settings, body.Connector, role, body.OAuthToken)
	if err != nil {
		return nil, err
	}
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(ui)
	return nil, nil
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
	user := r.PathValue("user")
	traits, err := ws.Traits(r.Context(), user)
	if err != nil {
		return nil, err
	}
	return map[string]any{"traits": traits}, nil
}

// Update updates the name, the privacy region and the displayed properties of a
// workspace.
func (workspace workspace) Update(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name                string                   `json:"name"`
		PrivacyRegion       core.PrivacyRegion       `json:"privacyRegion"`
		DisplayedProperties core.DisplayedProperties `json:"displayedProperties"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.Update(r.Context(), body.Name, body.PrivacyRegion, body.DisplayedProperties)
	return nil, err
}

// UpdateIdentityResolution updates the identity resolution of the workspace.
func (workspace workspace) UpdateIdentityResolution(_ http.ResponseWriter, r *http.Request) (any, error) {
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
	err = ws.UpdateIdentityResolution(r.Context(), body.RunOnBatchImport, body.Identifiers)
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

// UserSchema returns the user schema of a workspace.
func (workspace workspace) UserSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return ws.UserSchema, nil
}

// Users returns the users, the user schema of a workspace, and an estimate of
// their count without applying first and limit.
func (workspace workspace) Users(w http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Properties []string     `json:"properties"`
		Filter     *core.Filter `json:"filter"`
		Order      string       `json:"order"`
		OrderDesc  bool         `json:"orderDesc"`
		First      int          `json:"first"`
		Limit      int          `json:"limit"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	users, schema, count, err := ws.Users(r.Context(), body.Properties, body.Filter, body.Order, body.OrderDesc, body.First, body.Limit)
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
		b.writeString(`","lastChangeTime":"`)
		buf := b.availableBuffer()
		b.write(user.LastChangeTime.AppendFormat(buf, time.RFC3339Nano))
		b.writeString(`","properties":`)
		s, _ := types.Marshal(user.Properties, schema)
		b.write(s)
		b.writeByte('}')
	}
	b.writeString(`],"schema":`)
	buf, _ := schema.MarshalJSON()
	b.write(buf)
	b.writeString(`,"count":`)
	buf = b.availableBuffer()
	b.write(strconv.AppendInt(buf, int64(count), 10))
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
