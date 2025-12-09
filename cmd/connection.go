// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http"
	"strconv"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/tools/errors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

type connection struct {
	*apisServer
}

// APIUsers returns the users of an API connection.
func (connection connection) APIUsers(_ http.ResponseWriter, r *http.Request) (any, error) {

	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}

	// Read and parse the parameters from the query string.
	q := r.URL.Query()
	var schema types.Type
	if s := q.Get("schema"); s != "" {
		err := json.Unmarshal([]byte(s), &schema)
		if err != nil {
			return nil, errors.BadRequest("invalid schema: %s", err)
		}
	}
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
	cursor := q.Get("cursor")

	users, cursor, err := c.APIUsers(r.Context(), schema, filter, cursor)
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
	path, err := c.AbsolutePath(r.Context(), r.URL.Query().Get("path"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"path": path}, nil
}

// CreatePipeline creates a pipeline.
func (connection connection) CreatePipeline(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace{connection.apisServer}.workspace(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Connection int         `json:"connection"`
		Target     core.Target `json:"target"`
		EventType  string      `json:"eventType"`
		core.PipelineToSet
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if body.Target == 0 {
		return nil, errors.BadRequest("target is required")
	}
	if body.FormatSettings != nil && body.FormatSettings.IsNull() {
		body.FormatSettings = nil
	}
	c, err := ws.Connection(r.Context(), body.Connection)
	if err != nil {
		return nil, err
	}
	return c.CreatePipeline(r.Context(), body.Target, body.EventType, body.PipelineToSet)
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
	rows, schema, issues, err := c.ExecQuery(r.Context(), body.Query, body.Limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"rows": rows, "schema": schema, "issues": issues}, nil
}

// File returns the records and schema of the file located at the specified path
// within a connection.
func (connection connection) File(_ http.ResponseWriter, r *http.Request) (any, error) {

	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}

	// Read and parse the parameters from the query string.
	q := r.URL.Query()
	path := q.Get("path")
	format := q.Get("format")
	sheet := q.Get("sheet")
	var compression core.Compression
	switch c := q.Get("compression"); c {
	case "":
		compression = core.NoCompression
	case "Zip":
		compression = core.ZipCompression
	case "Gzip":
		compression = core.GzipCompression
	case "Snappy":
		compression = core.SnappyCompression
	default:
		return nil, errors.BadRequest("invalid compression %q", c)
	}
	var formatSettings json.Value
	if sett := q.Get("formatSettings"); sett != "" {
		if err := json.Validate([]byte(sett)); err != nil {
			return nil, errors.BadRequest("invalid formatSettings: %s", err)
		}
		formatSettings = json.Value(sett)
		if formatSettings.IsNull() {
			formatSettings = nil
		}
	}
	var limit int
	if l := q.Get("limit"); l != "" {
		limit, err = strconv.Atoi(l)
		if err != nil {
			return nil, errors.BadRequest("invalid limit")
		}
	}

	records, schema, issues, err := c.File(r.Context(), path, format, sheet, compression, formatSettings, limit)
	if err != nil {
		return nil, err
	}

	return map[string]any{"records": records, "schema": schema, "issues": issues}, nil
}

// Identities returns the identities of a connection.
func (connection connection) Identities(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
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
	identities, total, err := c.Identities(r.Context(), first, limit)
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
	dst, ok := parseID(r.PathValue("dst"))
	if !ok {
		return nil, errors.BadRequest("dst is not a valid connection identifier")
	}
	err = src.LinkConnection(r.Context(), dst)
	return nil, err
}

// PipelineSchemas returns the pipeline schema of a target.
//
// TODO(Gianluca): this method is deprecated. See the issue
// https://github.com/meergo/meergo/issues/1266.
func (connection connection) PipelineSchemas(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, ws, _, err := connection.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errMissingWorkspace
	}
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		return nil, errors.BadRequest("connection identifier %q is not valid", r.PathValue("id"))
	}
	c, err := ws.Connection(r.Context(), id)
	if err != nil {
		return nil, err
	}
	target, typ, err := connection.target(r)
	if err != nil {
		return nil, err
	}
	return c.PipelineSchemas(r.Context(), target, typ)
}

// PipelineTypes returns the pipeline types of a connection.
//
// TODO(Gianluca): this method is deprecated. See the issue
// https://github.com/meergo/meergo/issues/1265.
func (connection connection) PipelineTypes(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, ws, _, err := connection.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errMissingWorkspace
	}
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		return nil, errors.BadRequest("connection identifier %q is not valid", r.PathValue("id"))
	}
	c, err := ws.Connection(r.Context(), id)
	if err != nil {
		return nil, err
	}
	return c.PipelineTypes(r.Context())
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
	_, ws, _, err := connection.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errMissingWorkspace
	}
	id, ok := parseID(r.PathValue("id")) // ID of the connection
	if !ok {
		return nil, errors.BadRequest("connection identifier %q is not valid", r.PathValue("id"))
	}
	c, err := ws.Connection(r.Context(), id)
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
	return c.ServeUI(r.Context(), body.Event, body.Settings)
}

// Sheets returns the sheets of a file at the given path.
func (connection connection) Sheets(_ http.ResponseWriter, r *http.Request) (any, error) {

	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}

	// Read and parse the parameters from the query string.
	q := r.URL.Query()
	path := q.Get("path")
	format := q.Get("format")
	var compression core.Compression
	switch c := q.Get("compression"); c {
	case "":
		compression = core.NoCompression
	case "Zip":
		compression = core.ZipCompression
	case "Gzip":
		compression = core.GzipCompression
	case "Snappy":
		compression = core.SnappyCompression
	default:
		return nil, errors.BadRequest("invalid compression")
	}
	var formatSettings json.Value
	if sett := q.Get("formatSettings"); sett != "" {
		if err := json.Validate([]byte(sett)); err != nil {
			return nil, errors.BadRequest("invalid 'formatSettings': %s", err)
		}
		formatSettings = json.Value(sett)
		if formatSettings.IsNull() {
			formatSettings = nil
		}
	}

	sheets, err := c.Sheets(r.Context(), path, format, compression, formatSettings)
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
	schema, issues, err := c.TableSchema(r.Context(), r.URL.Query().Get("name"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"schema": schema, "issues": issues}, nil
}

// UnlinkConnection unlink a connection from another connection and vice versa.
func (connection connection) UnlinkConnection(_ http.ResponseWriter, r *http.Request) (any, error) {
	src, err := connection.src(r)
	if err != nil {
		return nil, err
	}
	dst, ok := parseID(r.PathValue("dst"))
	if !ok {
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
	schema, err := c.APIEventSchema(r.Context(), r.URL.Query().Get("type"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"schema": schema}, nil
}

// APIUserSchemas returns the user schemas for an API connection.
func (connection connection) APIUserSchemas(_ http.ResponseWriter, r *http.Request) (any, error) {
	c, err := connection.id(r)
	if err != nil {
		return nil, err
	}
	src, dst, err := c.APIUserSchemas(r.Context())
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

// id authenticates the request and returns the connection identified by the
// 'id' path parameter.
func (connection connection) id(r *http.Request) (*core.Connection, error) {
	ws, err := workspace{connection.apisServer}.workspace(r)
	if err != nil {
		return nil, err
	}
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		return nil, errors.BadRequest("connection identifier %q is not valid", r.PathValue("id"))
	}
	return ws.Connection(r.Context(), id)
}

// src authenticates the request and returns the connection identified by the
// 'src' path parameter.
func (connection connection) src(r *http.Request) (*core.Connection, error) {
	ws, err := workspace{connection.apisServer}.workspace(r)
	if err != nil {
		return nil, err
	}
	src, ok := parseID(r.PathValue("src"))
	if !ok {
		return nil, errors.BadRequest("connection identifier %q is not valid", r.PathValue("src"))
	}
	return ws.Connection(r.Context(), src)
}

// target returns a Target and, if applicable, the Event type.
// If the 'target' path parameter is present, it returns the corresponding
// Target and an empty type. Otherwise, it returns the Event Target and the type
// read from the query string.
//
// It does not authenticate the request.
func (connection connection) target(r *http.Request) (target core.Target, eventType string, err error) {
	v := r.PathValue("target")
	switch v {
	case "User":
		return core.TargetUser, "", nil
	case "Group":
		return core.TargetGroup, "", nil
	case "Event":
		return core.TargetEvent, "", nil
	case "":
		return core.TargetEvent, r.URL.Query().Get("type"), nil
	}
	return 0, "", errors.NotFound("")
}
