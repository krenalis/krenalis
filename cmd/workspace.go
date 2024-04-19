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

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/types"
)

type workspace struct {
	*apisServer
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
// collected events.
func (workspace workspace) AddEventListener(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Size      *int
		Source    int
		OnlyValid bool
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	var size = 10
	if body.Size != nil {
		size = *body.Size
	}
	id, err := ws.AddEventListener(r.Context(), size, body.Source, body.OnlyValid)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id}, nil
}

// ChangeUsersSchema changes the "users" schema of a workspace.
func (workspace workspace) ChangeUsersSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
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
	err = ws.ChangeUsersSchema(r.Context(), body.Schema, body.RePaths)
	return nil, err
}

// ChangeUsersSchemaQueries returns the queries that would be executed changing
// the "users" schema of a workspace.
func (workspace workspace) ChangeUsersSchemaQueries(_ http.ResponseWriter, r *http.Request) (any, error) {
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
	queries, err := ws.ChangeUsersSchemaQueries(r.Context(), body.Schema, body.RePaths)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Queries": queries}, nil
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
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.ChangeWarehouseSettings(r.Context(), body.Type, body.Settings)
	return nil, err
}

// Connection returns a connection of a workspace.
func (workspace workspace) Connection(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("connection")
	if v == "" || v[0] == '+' {
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
		Settings rawJSON
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.ConnectWarehouse(r.Context(), body.Type, body.Settings)
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
// used as identifiers in the Workspace Identity Resolution.
func (workspace workspace) IdentifiersSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return ws.IdentifiersSchema(r.Context())
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
		Connector   int
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

// PrivacyRegion returns the privacy region of a workspace.
func (workspace workspace) PrivacyRegion(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return ws.PrivacyRegion, nil
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

// RunIdentityResolution runs the Workspace Identity Resolution on a workspace.
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
		Connector  int
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

// SetIdentifiers sets the identifiers of the workspace.
func (workspace workspace) SetIdentifiers(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Identifiers []string
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = ws.SetIdentifiers(r.Context(), body.Identifiers)
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
		Filter     *apis.Filter
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

// UsersSchema returns the users schema of a workspace.
func (workspace workspace) UsersSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.workspace(r)
	if err != nil {
		return nil, err
	}
	return ws.UsersSchema, nil
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
