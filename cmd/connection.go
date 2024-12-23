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
	"github.com/meergo/meergo/types"
)

type connection struct {
	*apisServer
}

// Action returns an action of a connection.
func (connection connection) Action(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	id, err := connection.action(r)
	if err != nil {
		return nil, err
	}
	return c.Action(r.Context(), id)
}

// ActionSchemas returns the action schema of a target.
func (connection connection) ActionSchemas(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	target, eventType, err := connection.target(r)
	if err != nil {
		return nil, err
	}
	return c.ActionSchemas(r.Context(), target, eventType)
}

// ActionTypes returns the action types of a connection.
func (connection connection) ActionTypes(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	return c.ActionTypes(r.Context())
}

// AddAction adds an action.
func (connection connection) AddAction(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Target    core.Target      `json:"target"`
		EventType string           `json:"eventType"`
		Action    core.ActionToSet `json:"action"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	return c.AddAction(r.Context(), body.Target, body.EventType, body.Action)
}

// AppUsers returns the users of an app connection.
func (connection connection) AppUsers(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Schema types.Type `json:"schema"`
		Cursor string     `json:"cursor"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	users, cursor, err := c.AppUsers(r.Context(), body.Schema, body.Cursor)
	if err != nil {
		return nil, err
	}
	return map[string]any{"users": users, "cursor": cursor}, nil
}

// CompletePath returns the complete path of a storage path.
func (connection connection) CompletePath(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	path, err := c.CompletePath(r.Context(), r.PathValue("path"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"path": path}, nil
}

// Delete deletes a connection.
func (connection connection) Delete(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	err = c.Delete(r.Context())
	return nil, err
}

// ExecQuery executes a query on a database connection.
func (connection connection) ExecQuery(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	rows, schema, err := c.ExecQuery(r.Context(), body.Query, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"rows": rows, "schema": schema}, nil
}

// Executions returns the executions of the actions of a connection.
func (connection connection) Executions(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	return c.Executions(r.Context())
}

// Identities returns the user identities of a connection.
func (connection connection) Identities(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		First int `json:"first"`
		Limit int `json:"limit"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	identities, count, err := c.Identities(r.Context(), body.First, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"identities": identities,
		"count":      count,
	}, nil
}

// Keys returns the write keys of a connection.
func (connection connection) Keys(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	return c.Keys()
}

// GenerateKey generates a new write key for a connection.
func (connection connection) GenerateKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	return c.GenerateKey(r.Context())
}

// LinkConnection links a connection to another connection and vice versa.
func (connection connection) LinkConnection(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	c2, err := connection.connection2(r)
	if err != nil {
		return nil, err
	}
	err = c.LinkConnection(r.Context(), c2)
	return nil, err
}

// PreviewSendEvent previews sending an event.
func (connection connection) PreviewSendEvent(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		EventType      string                  `json:"eventType"`
		Event          json.Value              `json:"event"`
		Transformation core.DataTransformation `json:"transformation"`
		OutSchema      types.Type              `json:"outSchema"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	preview, err := c.PreviewSendEvent(r.Context(), body.EventType, body.Event, body.Transformation, body.OutSchema)
	if err != nil {
		return nil, err
	}
	return map[string]any{"preview": string(preview)}, nil
}

// Records returns the records and the schema of the file with a given path for
// a connection.
func (connection connection) Records(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Format         string           `json:"format"`
		Path           string           `json:"path"`
		Sheet          string           `json:"sheet"`
		Compression    core.Compression `json:"compression"`
		FormatSettings json.Value       `json:"formatSettings"`
		Limit          int              `json:"limit"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	records, schema, err := c.Records(r.Context(), body.Format, body.Path, body.Sheet, body.Compression, body.FormatSettings, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"records": records, "schema": schema}, nil
}

// RevokeKey revokes a write key of a connection.
func (connection connection) RevokeKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	err = c.RevokeKey(r.Context(), r.PathValue("key"))
	return nil, err
}

// ServeUI serves the user interface for a connection.
func (connection connection) ServeUI(w http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Event    string     `json:"event"`
		Settings json.Value `json:"settings"`
	}
	if r.Method == "GET" {
		body.Event = "load"
	} else {
		err = json.Decode(r.Body, &body)
		if err != nil {
			return nil, errors.BadRequest("%s", err)
		}
	}
	ui, err := c.ServeUI(r.Context(), body.Event, body.Settings)
	if err != nil {
		return nil, err
	}
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(ui)
	return nil, nil
}

// Set sets a connection.
func (connection connection) Set(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Connection core.ConnectionToSet `json:"connection"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = c.Set(r.Context(), body.Connection)
	return nil, err
}

// Sheets returns the sheets of a file at the given path.
func (connection connection) Sheets(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Format         string           `json:"format"`
		Path           string           `json:"path"`
		Compression    core.Compression `json:"compression"`
		FormatSettings json.Value       `json:"formatSettings"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	sheets, err := c.Sheets(r.Context(), body.Format, body.Path, body.FormatSettings, body.Compression)
	if err != nil {
		return nil, err
	}
	return map[string]any{"sheets": sheets}, nil
}

// TableSchema returns the schema of a table of a database connection.
func (connection connection) TableSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	return c.TableSchema(r.Context(), r.PathValue("table"))
}

// UnlinkConnection unlink a connection from another connection and vice versa.
func (connection connection) UnlinkConnection(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	c2, err := connection.connection2(r)
	if err != nil {
		return nil, err
	}
	err = c.UnlinkConnection(r.Context(), c2)
	return nil, err
}

func (connection connection) action(r *http.Request) (int, error) {
	v := r.PathValue("action")
	if v[0] == '+' {
		return 0, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return 0, errors.NotFound("")
	}
	return id, nil
}

func (connection connection) connection(r *http.Request) (*core.Connection, error) {
	ws, err := workspace{connection.apisServer}.workspace(r)
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

func (connection connection) connection2(r *http.Request) (int, error) {
	v := r.PathValue("connection2")
	if v[0] == '+' {
		return 0, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return 0, errors.NotFound("")
	}
	return id, nil
}

func (connection connection) target(r *http.Request) (core.Target, string, error) {
	v := r.PathValue("target")
	switch v {
	case "Users":
		return core.Users, "", nil
	case "Groups":
		return core.Groups, "", nil
	case "Events":
		return core.Events, "", nil
	case "":
		return core.Events, r.PathValue("eventType"), nil
	}
	return 0, "", errors.NotFound("")
}
