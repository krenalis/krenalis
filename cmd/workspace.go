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
	"strings"
	"time"

	"github.com/meergo/meergo/apis"
	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/filters"
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
	if ids, ok := q["action"]; ok {
		actions = make([]int, len(ids))
		for i, id := range ids {
			actions[i], err = strconv.Atoi(id)
			if err != nil {
				return nil, errors.BadRequest("an 'action' parameter is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least an 'action' parameter must be provided")
	}

	// Parse step.
	var step *apis.ActionStep
	if s, ok := q["step"]; ok {
		if len(s) > 1 {
			return nil, errors.BadRequest("only one 'step' parameter is allowed")
		}
		st, err := apis.ParseActionStep(s[0])
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

	return map[string][]apis.ActionError{"errors": errs}, nil
}

// ActionStatsPerDate returns statistics aggregated by day for a time interval
// between specified start and end dates.
func (workspace workspace) ActionStatsPerDate(_ http.ResponseWriter, r *http.Request) (any, error) {

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
	if ids, ok := q["action"]; ok {
		actions = make([]int, len(ids))
		for i, id := range ids {
			actions[i], err = strconv.Atoi(id)
			if err != nil {
				return nil, errors.BadRequest("an 'action' parameter is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least an 'action' parameter must be provided")
	}

	stats, err := ws.ActionStatsPerDate(r.Context(), start, end, actions)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  stats.Start,
		"end":    stats.End,
		"passed": stats.Passed,
		"failed": stats.Failed}, nil
}

// ActionStatsPerDay returns the action statistics for a specified number of
// days.
func (workspace workspace) ActionStatsPerDay(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}

	q := r.URL.Query()

	// Parse days.
	days := 48
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
	if ids, ok := q["action"]; ok {
		actions = make([]int, len(ids))
		for i, id := range ids {
			actions[i], err = strconv.Atoi(id)
			if err != nil {
				return nil, errors.BadRequest("an 'action' parameter is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least an 'action' parameter must be provided")
	}

	stats, err := ws.ActionStatsPerTimeUnit(r.Context(), days, apis.Day, actions)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  stats.Start,
		"end":    stats.End,
		"passed": stats.Passed,
		"failed": stats.Failed}, nil
}

// ActionStatsPerHour returns the action statistics for a specified number of
// hours.
func (workspace workspace) ActionStatsPerHour(_ http.ResponseWriter, r *http.Request) (any, error) {

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
	if ids, ok := q["action"]; ok {
		actions = make([]int, len(ids))
		for i, id := range ids {
			actions[i], err = strconv.Atoi(id)
			if err != nil {
				return nil, errors.BadRequest("an 'action' parameter is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least an 'action' parameter must be provided")
	}

	stats, err := ws.ActionStatsPerTimeUnit(r.Context(), hours, apis.Hour, actions)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  stats.Start,
		"end":    stats.End,
		"passed": stats.Passed,
		"failed": stats.Failed}, nil
}

// ActionStatsPerMinute returns the action statistics for a specified number of
// minutes.
func (workspace workspace) ActionStatsPerMinute(_ http.ResponseWriter, r *http.Request) (any, error) {

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
	if ids, ok := q["action"]; ok {
		actions = make([]int, len(ids))
		for i, id := range ids {
			actions[i], err = strconv.Atoi(id)
			if err != nil {
				return nil, errors.BadRequest("an 'action' parameter is not valid")
			}
		}
	} else {
		return nil, errors.BadRequest("at least an 'action' parameter must be provided")
	}

	stats, err := ws.ActionStatsPerTimeUnit(r.Context(), minutes, apis.Minute, actions)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"start":  stats.Start,
		"end":    stats.End,
		"passed": stats.Passed,
		"failed": stats.Failed}, nil
}

// AddConnection adds a new connection.
func (workspace workspace) AddConnection(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Connection apis.ConnectionToAdd
		OAuthToken string
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	return ws.AddConnection(r.Context(), body.Connection, body.OAuthToken)
}

// AddEventListener adds an event listener to a workspace that listens to
// collected or enriched events.
func (workspace workspace) AddEventListener(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Enriched      bool
		Size          *int
		Sources       []int
		OnlyValid     bool
		HasUserTraits bool
		Filter        *filters.Filter
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	var size = 10
	if body.Size != nil {
		size = *body.Size
	}
	var id string
	if body.Enriched {
		id, err = ws.AddEnrichedEventListener(size, body.Sources, body.HasUserTraits, body.Filter)
	} else {
		id, err = ws.AddCollectedEventListener(size, body.Sources, body.OnlyValid)
	}
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id}, nil
}

// ChangeUserSchema changes the user schema of a workspace.
func (workspace workspace) ChangeUserSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Schema         types.Type
		PrimarySources map[string]int
		RePaths        map[string]any
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.ChangeUserSchema(r.Context(), body.Schema, body.PrimarySources, body.RePaths)
	return nil, err
}

// ChangeUserSchemaQueries returns the queries that would be executed changing
// the "users" schema of a workspace.
func (workspace workspace) ChangeUserSchemaQueries(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Schema  types.Type
		RePaths map[string]any
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	queries, err := ws.ChangeUserSchemaQueries(r.Context(), body.Schema, body.RePaths)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Queries": queries}, nil
}

// ChangeWarehouseMode changes the mode of the data warehouse for a workspace.
func (workspace workspace) ChangeWarehouseMode(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Mode apis.WarehouseMode
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.ChangeWarehouseMode(r.Context(), body.Mode)
	return nil, err
}

// ChangeWarehouseSettings changes the settings of the data warehouse for a
// workspace.
func (workspace workspace) ChangeWarehouseSettings(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Type     apis.WarehouseType
		Settings rawJSON
		Mode     apis.WarehouseMode
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.ChangeWarehouseSettings(r.Context(), body.Type, body.Mode, body.Settings)
	return nil, err
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

// ConnectWarehouse connects a workspace to a data warehouse.
func (workspace workspace) ConnectWarehouse(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Type     apis.WarehouseType
		Mode     apis.WarehouseMode
		Settings rawJSON
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.ConnectWarehouse(r.Context(), body.Type, body.Mode, body.Settings)
	return nil, err
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

// DisconnectWarehouse disconnects a workspace from its data warehouse.
func (workspace workspace) DisconnectWarehouse(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	err = ws.DisconnectWarehouse(r.Context())
	return nil, err
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

// InitWarehouse initializes a data warehouse of the workspace by creating the
// supporting tables.
func (workspace workspace) InitWarehouse(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	err = ws.InitWarehouse(r.Context())
	return nil, err
}

// ListenedEvents returns the events listen to by a specified listener and the
// number of discarded events.
func (workspace workspace) ListenedEvents(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	events, discarded, err := ws.ListenedEvents(r.PathValue("listener"))
	if err != nil {
		return nil, err
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
	body := struct {
		OAuthCode   string
		RedirectURI string
		Connector   string
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	return ws.OAuthToken(r.Context(), body.OAuthCode, body.RedirectURI, body.Connector)
}

// PingWarehouse pings a warehouse with given settings, verifying that the
// settings are valid and a connection can be established.
func (workspace workspace) PingWarehouse(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Type     apis.WarehouseType
		Settings rawJSON
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.PingWarehouse(r.Context(), body.Type, body.Settings)
	return nil, err
}

// RemoveEventListener removes an event listener from a workspace.
func (workspace workspace) RemoveEventListener(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	ws.RemoveEventListener(r.PathValue("listener"))
	return nil, nil
}

// RunIdentityResolution runs the Identity Resolution on a workspace.
func (workspace workspace) RunIdentityResolution(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	err = ws.RunIdentityResolution(r.Context())
	return nil, err
}

// ServeUI serves the user interface for a connector.
func (workspace workspace) ServeUI(w http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Connector  string
		Event      string
		Values     rawJSON
		Role       string
		OAuthToken string
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if strings.HasSuffix(r.URL.Path, "/ui") {
		body.Event = "load"
		body.Values = nil
	}
	var role apis.Role
	switch body.Role {
	case "Source":
		role = apis.Source
	case "Destination":
		role = apis.Destination
	default:
		return nil, errors.BadRequest("unexpected connection role '%s'", body.Role)
	}
	ui, err := ws.ServeUI(r.Context(), body.Event, body.Values, body.Connector, role, body.OAuthToken)
	if err != nil {
		return nil, err
	}
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(ui)
	return nil, nil
}

// Set sets the name, the privacy region and the displayed properties of a
// workspace.
func (workspace workspace) Set(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name                string
		PrivacyRegion       apis.PrivacyRegion
		DisplayedProperties apis.DisplayedProperties
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.Set(r.Context(), body.Name, body.PrivacyRegion, body.DisplayedProperties)
	return nil, err
}

// ChangeIdentityResolutionSettings changes the settings of the Identity
// Resolution of the workspace.
func (workspace workspace) ChangeIdentityResolutionSettings(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		RunOnBatchImport bool
		Identifiers      []string
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.ChangeIdentityResolutionSettings(r.Context(), body.RunOnBatchImport, body.Identifiers)
	return nil, err
}

// Users returns the users, the user schema of a workspace, and an estimate of
// their count without applying first and limit.
func (workspace workspace) Users(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Properties []string
		Filter     *filters.Filter
		Order      string
		OrderDesc  bool
		First      int
		Limit      int
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	users, schema, count, err := ws.Users(r.Context(), body.Properties, body.Filter, body.Order, body.OrderDesc, body.First, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"users":  rawJSON(users),
		"schema": schema,
		"count":  count,
	}, nil
}

// IdentityResolutionExecution returns information about the execution of the
// Identity Resolution of the workspace.
func (workspace workspace) IdentityResolutionExecution(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	startTime, endTime, err := ws.IdentityResolutionExecution(r.Context())
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"startTime": startTime,
		"endTime":   endTime,
	}, nil
}

// UserSchema returns the user schema of a workspace.
func (workspace workspace) UserSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return ws.UserSchema, nil
}

// WarehouseSettings returns the type and settings of the data warehouse for a
// workspace.
func (workspace workspace) WarehouseSettings(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	typ, settings, err := ws.WarehouseSettings()
	if err != nil {
		return nil, err
	}
	return map[string]any{"type": typ, "settings": rawJSON(settings)}, nil
}

func (workspace workspace) workspace(r *http.Request) (*apis.Workspace, error) {
	v := r.PathValue("workspace")
	if v[0] == '+' {
		return nil, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return nil, errors.NotFound("")
	}
	_, organization, err := workspace.credentials(r)
	if err != nil {
		return nil, err
	}
	return organization.Workspace(id)
}
