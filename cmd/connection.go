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

// ActionSchemas returns the action schema of a target.
//
// TODO(Gianluca): this method is deprecated. See the issue
// https://github.com/meergo/meergo/issues/1266.
func (connection connection) ActionSchemas(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	target, typ, err := connection.target(r)
	if err != nil {
		return nil, err
	}
	return c.ActionSchemas(r.Context(), target, typ)
}

// ActionTypes returns the action types of a connection.
//
// TODO(Gianluca): this method is deprecated. See the issue
// https://github.com/meergo/meergo/issues/1265.
func (connection connection) ActionTypes(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	return c.ActionTypes(r.Context())
}

// AppUsers returns the users of an app connection.
func (connection connection) AppUsers(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
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

// AbsolutePath returns the absolute path of a storage path.
func (connection connection) AbsolutePath(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	path, err := c.AbsolutePath(r.Context(), r.PathValue("path"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"path": path}, nil
}

// CreateAction creates an action.
func (connection connection) CreateAction(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace{connection.apisServer}.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Connection int         `json:"connection"`
		Target     core.Target `json:"target"`
		EventType  string      `json:"eventType"`
		core.ActionToSet
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if body.FormatSettings != nil && body.FormatSettings.IsNull() {
		body.FormatSettings = nil
	}
	c, err := ws.Connection(r.Context(), body.Connection)
	if err != nil {
		return nil, err
	}
	return c.CreateAction(r.Context(), body.Target, body.EventType, body.ActionToSet)
}

// CreateEventWriteKey creates a new event write key for a connection.
func (connection connection) CreateEventWriteKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	return c.CreateEventWriteKey(r.Context())
}

// Delete deletes a connection.
func (connection connection) Delete(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	err = c.Delete(r.Context())
	return nil, err
}

// DeleteEventWriteKey deletes an event write key of a connection.
func (connection connection) DeleteEventWriteKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	err = c.DeleteEventWriteKey(r.Context(), r.PathValue("key"))
	return nil, err
}

// ExecQuery executes a query on a database connection.
func (connection connection) ExecQuery(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
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

// File returns the records and schema of the file located at the specified path
// within a connection.
func (connection connection) File(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	path := r.PathValue("path")
	var body struct {
		Format         string           `json:"format"`
		Sheet          string           `json:"sheet"`
		Compression    core.Compression `json:"compression"`
		FormatSettings json.Value       `json:"formatSettings"`
		Limit          int              `json:"limit"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	records, schema, err := c.File(r.Context(), path, body.Format, body.Sheet, body.Compression, body.FormatSettings, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"records": records, "schema": schema}, nil
}

// Identities returns the user identities of a connection.
func (connection connection) Identities(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
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
	identities, total, err := c.Identities(r.Context(), body.First, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"identities": identities,
		"total":      total,
	}, nil
}

// LinkConnection links a connection to another connection and vice versa.
func (connection connection) LinkConnection(_ http.ResponseWriter, r *http.Request) (any, error) {
	src, err := connection.src(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("dst")
	if v[0] == '+' {
		return nil, errors.BadRequest("dst is not a valid connection identifier")
	}
	dst, _ := strconv.Atoi(v)
	if dst <= 0 {
		return nil, errors.BadRequest("dst is not a valid connection identifier")
	}
	err = src.LinkConnection(r.Context(), dst)
	return nil, err
}

// PreviewSendEvent previews sending an event.
func (connection connection) PreviewSendEvent(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Type           string                  `json:"type"`
		Event          json.Value              `json:"event"`
		Transformation core.DataTransformation `json:"transformation"`
		OutSchema      types.Type              `json:"outSchema"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	preview, err := c.PreviewSendEvent(r.Context(), body.Type, body.Event, body.Transformation, body.OutSchema)
	if err != nil {
		return nil, err
	}
	return map[string]any{"preview": string(preview)}, nil
}

// ServeUI serves the user interface for a connection.
func (connection connection) ServeUI(w http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
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

// Sheets returns the sheets of a file at the given path.
func (connection connection) Sheets(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	path := r.PathValue("path")
	var body struct {
		Format         string           `json:"format"`
		Compression    core.Compression `json:"compression"`
		FormatSettings json.Value       `json:"formatSettings"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	sheets, err := c.Sheets(r.Context(), path, body.Format, body.Compression, body.FormatSettings)
	if err != nil {
		return nil, err
	}
	return map[string]any{"sheets": sheets}, nil
}

// TableSchema returns the schema of a table of a database connection.
func (connection connection) TableSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	return c.TableSchema(r.Context(), r.PathValue("name"))
}

// UnlinkConnection unlink a connection from another connection and vice versa.
func (connection connection) UnlinkConnection(_ http.ResponseWriter, r *http.Request) (any, error) {
	src, err := connection.src(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("dst")
	if v[0] == '+' {
		return nil, errors.BadRequest("dst is not a valid connection identifier")
	}
	dst, _ := strconv.Atoi(v)
	if dst <= 0 {
		return nil, errors.BadRequest("dst is not a valid connection identifier")
	}
	err = src.UnlinkConnection(r.Context(), dst)
	return nil, err
}

// Update updates a connection.
func (connection connection) Update(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		core.ConnectionToSet
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = c.Update(r.Context(), body.ConnectionToSet)
	return nil, err
}

// AppEventSchema returns the schema of the provided event type of the
// connection.
func (connection connection) AppEventSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	schema, err := c.AppEventSchema(r.Context(), r.PathValue("type"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"schema": schema}, nil
}

// AppUserSchemas returns the user schemas for an app connection.
func (connection connection) AppUserSchemas(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	src, dst, err := c.AppUserSchemas(r.Context())
	if err != nil {
		return nil, err
	}
	return map[string]any{"schemas": map[string]any{"source": src, "destination": dst}}, nil
}

// EventWriteKeys returns the event write keys of a connection.
func (connection connection) EventWriteKeys(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	return c.EventWriteKeys()
}

func (connection connection) id(r *http.Request) (*core.Connection, error) {
	ws, err := workspace{connection.apisServer}.workspace(r)
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

func (connection connection) src(r *http.Request) (*core.Connection, error) {
	ws, err := workspace{connection.apisServer}.workspace(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("src")
	if v[0] == '+' {
		return nil, errors.BadRequest("src is not a valid connection identifier")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return nil, errors.BadRequest("src is not a valid connection identifier")
	}
	return ws.Connection(r.Context(), id)
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
		return core.Events, r.PathValue("type"), nil
	}
	return 0, "", errors.NotFound("")
}
