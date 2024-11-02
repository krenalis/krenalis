//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	jsonstd "encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"
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

	stats, err := ws.ActionStatsPerTimeUnit(r.Context(), days, core.Day, actions)
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

	stats, err := ws.ActionStatsPerTimeUnit(r.Context(), hours, core.Hour, actions)
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

	stats, err := ws.ActionStatsPerTimeUnit(r.Context(), minutes, core.Minute, actions)
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
		Connection core.ConnectionToAdd
		OAuthToken string
	}{}
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
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
		Filter        *core.Filter
	}
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
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

// CanChangeWarehouseSettings determines if it is possible to change the
// warehouse settings of a workspace.
func (workspace workspace) CanChangeWarehouseSettings(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Settings rawJSON
	}{}
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.CanChangeWarehouseSettings(r.Context(), body.Settings)
	return nil, err
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
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
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
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
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
		Mode                         core.WarehouseMode
		CancelIncompatibleOperations bool
	}{}
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.ChangeWarehouseMode(r.Context(), body.Mode, body.CancelIncompatibleOperations)
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
		Settings                     rawJSON
		Mode                         core.WarehouseMode
		CancelIncompatibleOperations bool
	}{}
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.ChangeWarehouseSettings(r.Context(), body.Mode, body.Settings,
		body.CancelIncompatibleOperations)
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

// Delete deletes a workspace with all its connections.
func (workspace workspace) Delete(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	err = ws.Delete(r.Context())
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
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	return ws.OAuthToken(r.Context(), body.OAuthCode, body.RedirectURI, body.Connector)
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

// ResolveIdentities resolves the identities of a workspace.
func (workspace workspace) ResolveIdentities(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	err = ws.ResolveIdentities(r.Context())
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
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if strings.HasSuffix(r.URL.Path, "/ui") {
		body.Event = "load"
		body.Values = nil
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
	ui, err := ws.ServeUI(r.Context(), body.Event, json.Value(body.Values), body.Connector, role, body.OAuthToken)
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
		PrivacyRegion       core.PrivacyRegion
		DisplayedProperties core.DisplayedProperties
	}
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
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
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
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
		Filter     *core.Filter
		Order      string
		OrderDesc  bool
		First      int
		Limit      int
	}
	err = jsonstd.NewDecoder(r.Body).Decode(&body)
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
	name, settings := ws.WarehouseSettings()
	return map[string]any{"name": name, "settings": rawJSON(settings)}, nil
}

func (workspace workspace) workspace(r *http.Request) (*core.Workspace, error) {
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
