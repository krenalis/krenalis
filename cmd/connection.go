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

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/types"
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

// AddAction adds an action.
func (connection connection) AddAction(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Target    apis.Target
		EventType string
		Action    apis.ActionToSet
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	return c.AddAction(r.Context(), body.Target, body.EventType, body.Action)
}

// AddEventConnection adds an event connection to a connection and vice versa.
func (connection connection) AddEventConnection(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	c2, err := connection.connection2(r)
	if err != nil {
		return nil, err
	}
	err = c.AddEventConnection(r.Context(), c2)
	return nil, err
}

// AppUsers returns the users of an app connection.
func (connection connection) AppUsers(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Schema types.Type
		Cursor string
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	users, cursor, err := c.AppUsers(r.Context(), body.Schema, body.Cursor)
	if err != nil {
		return nil, err
	}
	return map[string]any{"users": json.RawMessage(users), "cursor": cursor}, nil
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
		Query string
		Limit int
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	rows, schema, err := c.ExecQuery(r.Context(), body.Query, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"Rows": json.RawMessage(rows), "Schema": schema}, nil
}

// Executions returns the executions of the actions of a connection.
func (connection connection) Executions(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	return c.Executions(r.Context())
}

// Identities returns the users identities of a connection.
func (connection connection) Identities(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		First int
		Limit int
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	identities, count, err := c.Identities(r.Context(), body.First, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"identities": json.RawMessage(identities),
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

// PreviewSendEvent previews sending an event.
func (connection connection) PreviewSendEvent(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		EventType      string
		Event          *apis.ObservedEvent
		Transformation apis.Transformation
		OutSchema      types.Type
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
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
		FileConnector int
		Path          string
		Sheet         string
		Compression   apis.Compression
		Settings      json.RawMessage
		Limit         int
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	records, schema, err := c.Records(r.Context(), body.FileConnector, body.Path, body.Sheet, body.Compression, body.Settings, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"records": json.RawMessage(records), "schema": schema}, nil
}

// RemoveEventConnection removes an event connection from a connection and vice
// versa.
func (connection connection) RemoveEventConnection(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	c2, err := connection.connection2(r)
	if err != nil {
		return nil, err
	}
	err = c.RemoveEventConnection(r.Context(), c2)
	return nil, err
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
		Event  string
		Values json.RawMessage
	}
	if r.Method == "GET" {
		body.Event = "load"
	} else {
		err = json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			return nil, err
		}
	}
	form, err := c.ServeUI(r.Context(), body.Event, body.Values)
	if err != nil {
		return nil, err
	}
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(form)
	return nil, nil
}

// Set sets a connection.
func (connection connection) Set(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Connection apis.ConnectionToSet
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
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
		FileConnector int
		Path          string
		Compression   apis.Compression
		Settings      json.RawMessage
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	sheets, err := c.Sheets(r.Context(), body.FileConnector, body.Path, body.Settings, body.Compression)
	if err != nil {
		return nil, err
	}
	return map[string]any{"sheets": sheets}, nil
}

// Stats returns statistics on a connection for the last 24 hours.
func (connection connection) Stats(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	return c.Stats(r.Context())
}

// TableSchema returns the schema of a table of a database connection.
func (connection connection) TableSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.connection(r)
	if err != nil {
		return nil, err
	}
	return c.TableSchema(r.Context(), r.PathValue("table"))
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

func (connection connection) connection(r *http.Request) (*apis.Connection, error) {
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
	return ws.Connection(r.Context(), id)
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

func (connection connection) target(r *http.Request) (apis.Target, string, error) {
	v := r.PathValue("target")
	switch v {
	case "Users":
		return apis.Users, "", nil
	case "Groups":
		return apis.Groups, "", nil
	case "Events":
		return apis.Events, "", nil
	case "":
		return apis.Events, r.PathValue("eventType"), nil
	}
	return 0, "", errors.NotFound("")
}
