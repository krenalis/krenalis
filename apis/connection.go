//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/transformations"
	"chichi/apis/types"
	_connector "chichi/connector"
	"chichi/connector/ui"

	"github.com/jxskiss/base62"
)

const (
	maxKeysPerServer = 20 // maximum number of keys per server.
	maxInt32         = math.MaxInt32
	rawSchemaMaxSize = 16_777_215 // maximum size in runes of the 'schema' column of the 'connections' table.
	queryMaxSize     = 16_777_215 // maximum size in runes of a connection query.
)

var (
	AlreadyHasMappings          errors.Code = "AlreadyHasMappings"
	AlreadyHasTransformation    errors.Code = "AlreadyHasTransformation"
	ConnectorNotExist           errors.Code = "ConnectorNotExist"
	EventNotExist               errors.Code = "EventNotExist"
	InvalidRefreshToken         errors.Code = "InvalidRefreshToken"
	KeyNotExist                 errors.Code = "KeyNotExist"
	NoStorage                   errors.Code = "NoStorage"
	NoTransformationNorMappings errors.Code = "NoMappings"
	QueryExecutionFailed        errors.Code = "QueryExecutionFailed"
	StorageNotExist             errors.Code = "StorageNotExist"
	TooManyKeys                 errors.Code = "TooManyKeys"
	UniqueKey                   errors.Code = "UniqueKey"
	WorkspaceNotExist           errors.Code = "WorkspaceNotExist"
)

// Connection represents a connection.
type Connection struct {
	db             *postgres.DB
	connection     *state.Connection
	ID             int
	Name           string
	Type           ConnectorType
	Role           ConnectionRole
	Storage        int    // zero if the connection is not a file or does not have a storage.
	OAuthURL       string // empty if the connection does not use OAuth.
	HasSettings    bool
	LogoURL        string
	Enabled        bool
	UsersQuery     string          // only for databases.
	Transformation *Transformation // nil if connection has no transformation.
	Mappings       []*Mapping
	Health         ConnectionHealth
}

// Transformation represents the transformation of a connection.
type Transformation struct {
	In           types.Type
	Out          types.Type
	PythonSource string
}

// Delete deletes the connection.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
func (this *Connection) Delete() error {
	n := state.DeleteConnectionNotification{
		ID: this.connection.ID,
	}
	connector := this.connection.Connector()
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "DELETE FROM connections WHERE id = $1", n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("connection %d does not exist", n.ID)
		}
		if connector.OAuth != nil {
			// Delete the resource of the deleted connection if it has no other connections.
			_, err := tx.Exec(ctx, "DELETE FROM resources AS r WHERE NOT EXISTS (\n"+
				"\tSELECT FROM connections AS c\n"+
				"\tWHERE r.id = c.resource AND c.id <> $1 AND c.resource IS NULL\n)", n.ID)
			if err != nil {
				return err
			}
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// GenerateKey generates a new key for the connection. The connection must be a
// source server connection.
//
// If the server does not exist, it returns an errors.NotFoundError error.
// If the server has already too many keys, it returns an
// errors.UnprocessableError error with code TooManyKeys.
func (this *Connection) GenerateKey() (string, error) {
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.ServerType {
		return "", errors.NotFound("connection %d is not a server", c.ID)
	}
	if c.Role != state.SourceRole {
		return "", errors.NotFound("server %d is not a source", c.ID)
	}
	value, err := generateServerKey()
	if err != nil {
		return "", err
	}
	n := state.AddConnectionKeyNotification{
		Connection:   c.ID,
		Value:        value,
		CreationTime: time.Now().UTC(),
	}
	ctx := context.Background()
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM connections_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == maxKeysPerServer {
			return errors.Unprocessable(TooManyKeys, "server %d has already %d types", n.Connection, maxKeysPerServer)
		}
		_, err = tx.Exec(ctx, "INSERT INTO connections_keys (connection, value, creation_time) VALUES ($1, $2, $3)",
			n.Connection, n.Value, n.CreationTime)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "connections_keys_connection_fkey" {
					err = errors.NotFound("connection %d does not exist", n.Connection)
				}
			}
			return err
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return "", err
	}

	return value, nil
}

// Keys returns the keys of the source server with identifier id.
//
// If the server does not exist, it returns an errors.NotFoundError error.
func (this *Connection) Keys() ([]string, error) {
	c := this.connection
	if c.Connector().Type != state.ServerType {
		return nil, errors.NotFound("connection %d is not a server", c.ID)
	}
	if c.Role != state.SourceRole {
		return nil, errors.NotFound("server %d is not a source", c.ID)
	}
	keys := make([]string, len(c.Keys))
	for i, key := range c.Keys {
		keys[i] = encodeServerKey([]byte(key))
	}
	return keys, nil
}

// An Import describes a connection import as returned by Imports.
type Import struct {
	ID        int
	StartTime time.Time
	EndTime   *time.Time
	Error     string
}

// Imports returns a list of Import describing all imports of the connection.
// The connection must be a source app, database, or file connection.
func (this *Connection) Imports() ([]*Import, error) {
	c := this.connection
	connector := c.Connector()
	switch connector.Type {
	case state.AppType, state.DatabaseType, state.FileType, state.StreamType:
	default:
		return nil, errors.BadRequest("connection %d cannot have imports, it's a %s connection",
			c.ID, strings.ToLower(connector.Type.String()))
	}
	if c.Role == state.DestinationRole {
		return nil, errors.BadRequest("connection %d cannot have imports, it's a destination", c.ID)
	}
	imports := []*Import{}
	err := this.db.QueryScan(context.Background(),
		"SELECT id, start_time, end_time, error\n"+
			"FROM connections_imports\n"+
			"WHERE connection = $1\n"+
			"ORDER BY id DESC", c.ID, func(rows *postgres.Rows) error {
			var err error
			for rows.Next() {
				var imp Import
				if err = rows.Scan(&imp.ID, &imp.StartTime, &imp.EndTime, &imp.Error); err != nil {
					return err
				}
				imports = append(imports, &imp)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return imports, nil
}

// RevokeKey revokes the given key of the source server with identifier id. key
// cannot be empty and cannot be the unique key of the server.
//
// If the key does not exist, it returns an errors.NotFoundError error.
// If the key is the unique key of the server, it returns an
// errors.UnprocessableError error with code UniqueKey.
func (this *Connection) RevokeKey(key string) error {
	if key == "" {
		return errors.BadRequest("key is empty")
	}
	if !isServerKey(key) {
		return errors.BadRequest("key %q is malformed", key)
	}
	c := this.connection
	connector := c.Connector()
	if connector.Type != state.ServerType {
		return errors.BadRequest("connection %d is not a server", c.ID)
	}
	if c.Role != state.SourceRole {
		return errors.BadRequest("server %d is not a source", c.ID)
	}
	n := state.RevokeConnectionKeyNotification{
		Connection: c.ID,
		Value:      key,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		var count int
		err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM connections_keys WHERE connection = $1", n.Connection).Scan(&count)
		if err != nil {
			return err
		}
		if count == 1 {
			return errors.Unprocessable(UniqueKey, "key cannot be revoked because it's the unique key of the server")
		}
		result, err := tx.Exec(ctx, "DELETE FROM connections_keys WHERE connection = $1 AND value = $2", n.Connection, n.Value)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.Unprocessable(KeyNotExist, "key %q does not exist", key)
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// Schema returns the schema of the connection. The connection must be an app,
// database, or file connection. If the connection does not have a schema, it
// returns an invalid schema.
func (this *Connection) Schema() (types.Type, error) {
	c := this.connection
	switch c.Connector().Type {
	case state.StorageType:
		return types.Type{}, errors.BadRequest("connection %d has no properties, it's a storage", c.ID)
	case state.StreamType:
		return types.Type{}, errors.BadRequest("connection %d has no properties, it's a stream", c.ID)
	}
	return c.Schema, nil
}

// SetStatus sets the status of the connection.
func (this *Connection) SetStatus(enabled bool) error {
	if enabled == this.Enabled {
		return nil
	}
	n := state.SetConnectionStatusNotification{
		Connection: this.connection.ID,
		Enabled:    enabled,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections SET enabled = $1 WHERE id = $2 AND enabled <> $1", n.Enabled, n.Connection)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// Action returns the action with identifier id of the connection, which must be
// a destination of type app.
// It returns an errors.NotFound error if the action does not exist.
func (this *Connection) Action(id int) (*Action, error) {
	c := this.connection
	if c.Role != state.DestinationRole {
		return nil, errors.BadRequest("connection is not a destination")
	}
	connector := c.Connector()
	if connector.Type != state.AppType {
		return nil, errors.BadRequest("connection type is not app")
	}
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid action identifier", id)
	}
	a, ok := this.connection.Action(id)
	if !ok {
		return nil, errors.NotFound("action %d does not exist", id)
	}
	action := Action{
		db:             this.db,
		action:         a,
		connection:     this,
		ID:             a.ID,
		Connection:     this.connection.ID,
		ActionType:     a.ActionType,
		Name:           a.Name,
		Enabled:        a.Enabled,
		Endpoint:       a.Endpoint,
		Mapping:        a.Mapping,
		Transformation: (*Transformation)(a.Transformation),
	}
	action.Filter.Logical = a.Filter.Logical
	action.Filter.Conditions = make([]ActionFilterCondition, len(a.Filter.Conditions))
	for i := range action.Filter.Conditions {
		action.Filter.Conditions[i] = ActionFilterCondition(a.Filter.Conditions[i])
	}
	return &action, nil
}

// Actions returns the actions of the connection.
func (this *Connection) Actions() ([]*Action, error) {
	c := this.connection
	if c.Role != state.DestinationRole {
		return nil, errors.BadRequest("connection is not a destination")
	}
	connector := c.Connector()
	if connector.Type != state.AppType {
		return nil, errors.BadRequest("connection type is not app")
	}
	as := this.connection.Actions()
	actions := make([]*Action, len(as))
	for i, a := range as {
		action := Action{
			db:             this.db,
			action:         a,
			connection:     this,
			ID:             a.ID,
			Connection:     this.connection.ID,
			ActionType:     a.ActionType,
			Name:           a.Name,
			Enabled:        a.Enabled,
			Endpoint:       a.Endpoint,
			Mapping:        a.Mapping,
			Transformation: (*Transformation)(a.Transformation),
		}
		action.Filter.Logical = a.Filter.Logical
		action.Filter.Conditions = make([]ActionFilterCondition, len(a.Filter.Conditions))
		for i := range action.Filter.Conditions {
			action.Filter.Conditions[i] = ActionFilterCondition(a.Filter.Conditions[i])
		}
		actions[i] = &action
	}
	return actions, nil
}

// ActionTypes returns the action types of the connection, which must be a
// destination of type app.
func (this *Connection) ActionTypes() ([]*ActionType, error) {
	c := this.connection
	if c.Role != state.DestinationRole {
		return nil, errors.BadRequest("connection is not a destination")
	}
	if c.Connector().Type != state.AppType {
		return nil, errors.BadRequest("connection type is not app")
	}
	return this.actionTypes(), nil
}

// actionTypes returns the action types for this connection.
func (this *Connection) actionTypes() []*ActionType {
	connector := this.connection.Connector()
	app := _connector.RegisteredApp(connector.Name)
	stateActionTypes := this.connection.ActionTypes()
	actionTypes := make([]*ActionType, len(stateActionTypes))
	for i, at := range stateActionTypes {
		endpoints := map[int]string{}
		for _, e := range at.Endpoints {
			endpoints[e] = app.Endpoints[e]
		}
		actionType := ActionType{
			actionType:  at,
			ID:          at.ID,
			Name:        at.Name,
			Description: at.Description,
			Endpoints:   endpoints,
			Schema:      at.Schema,
		}
		actionTypes[i] = &actionType
	}
	return actionTypes
}

// Column represents a column of a database connection.
type Column struct {
	Name string
	Type types.Type
}

// Query executes the given query on the connection and returns the resulting
// columns and rows. The connection must be a source database connection.
//
// query must be UTF-8 encoded, it cannot be longer than 16,777,215 runes and
// must contain the ':limit' placeholder between '[[' and ']]'. limit must be
// between 1 and 100.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// If the execution of the query fails, it returns an errors.UnprocessableError
// with code QueryExecutionFailed.
func (this *Connection) Query(query string, limit int) ([]Column, [][]string, error) {

	if !utf8.ValidString(query) {
		return nil, nil, errors.BadRequest("query is not UTF-8 encoded")
	}
	if utf8.RuneCountInString(query) > queryMaxSize {
		return nil, nil, errors.BadRequest("query is longer than 16,777,215 runes")
	}
	if limit < 1 || limit > 100 {
		return nil, nil, errors.BadRequest("limit %d is not valid", limit)
	}

	c := this.connection
	connector := c.Connector()
	if connector.Type != state.DatabaseType {
		return nil, nil, errors.BadRequest("connection %d is not a database", c.ID)
	}
	if c.Role != state.SourceRole {
		return nil, nil, errors.BadRequest("database %d is not a source", c.ID)
	}

	const cRole = _connector.SourceRole

	// Execute the query.
	var err error
	query, err = compileConnectionQuery(query, limit)
	if err != nil {
		return nil, nil, err
	}
	fh := this.newFirehose(context.Background())
	connection, err := _connector.RegisteredDatabase(connector.Name).Open(fh.ctx, &_connector.DatabaseConfig{
		Role:     cRole,
		Settings: c.Settings,
		Firehose: fh,
	})
	if err != nil {
		return nil, nil, err
	}
	rawColumns, rawRows, err := connection.Query(query)
	if err != nil {
		if err, ok := err.(*_connector.DatabaseQueryError); ok {
			return nil, nil, errors.Unprocessable(QueryExecutionFailed, "query execution of connection %d failed: %w", c.ID, err)
		}
		return nil, nil, err
	}

	// Fill the columns.
	columns := make([]Column, len(rawColumns))
	for i, c := range rawColumns {
		columns[i].Name = c.Name
		columns[i].Type = c.Type
	}

	// Fill the rows.
	var rows [][]string
	values := make([]any, len(columns))
	for i := range values {
		var value string
		values[i] = &value
	}
	for rawRows.Next() {
		if err := rawRows.Scan(values...); err != nil {
			return nil, nil, err
		}
		row := make([]string, len(rawColumns))
		for i, v := range values {
			row[i] = *(v.(*string))
		}
		rows = append(rows, row)
	}
	err = rawRows.Close()
	if err != nil {
		return nil, nil, err
	}
	if rows == nil {
		rows = [][]string{}
	}

	return columns, rows, nil
}

// Rename renames the connection with the given new name.
// name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
func (this *Connection) Rename(name string) error {
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return errors.BadRequest("name %q is not valid", name)
	}
	if name == this.connection.Name {
		return nil
	}
	n := state.RenameConnectionNotification{
		Connection: this.connection.ID,
		Name:       name,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections SET name = $1 WHERE id = $2", n.Name, n.Connection)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("connection %d does not exist", n.Connection)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// ServeUI serves the user interface for the connection. event is the event and
// values contains the form values in JSON format.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// If the event does not exist, it returns an errors.UnprocessableError error
// with code EventNotExist.
func (this *Connection) ServeUI(event string, values []byte) ([]byte, error) {

	c := this.connection
	cRole := _connector.Role(c.Role)
	connector := c.Connector()

	var err error
	var connection any

	switch connector.Type {
	case state.AppType:

		var clientSecret, resourceCode, accessToken string
		if r, ok := c.Resource(); ok {
			clientSecret = connector.OAuth.ClientSecret
			resourceCode = r.Code
			var err error
			accessToken, err = freshAccessToken(this.db, r)
			if err != nil {
				return nil, fmt.Errorf("cannot retrive the OAuth access token: %s", err)
			}
		}

		fh := this.newFirehose(context.Background())
		connection, err = _connector.RegisteredApp(connector.Name).Open(fh.ctx, &_connector.AppConfig{
			Role:         cRole,
			Settings:     c.Settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})

	default:

		fh := this.newFirehose(context.Background())

		switch connector.Type {
		case state.DatabaseType:
			connection, err = _connector.RegisteredDatabase(connector.Name).Open(fh.ctx, &_connector.DatabaseConfig{
				Role:     cRole,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.FileType:
			connection, err = _connector.RegisteredFile(connector.Name).Open(fh.ctx, &_connector.FileConfig{
				Role:     cRole,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.MobileType:
			connection, err = _connector.RegisteredMobile(connector.Name).Open(fh.ctx, &_connector.MobileConfig{
				Role:     cRole,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.ServerType:
			connection, err = _connector.RegisteredServer(connector.Name).Open(fh.ctx, &_connector.ServerConfig{
				Role:     cRole,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.StorageType:
			connection, err = _connector.RegisteredStorage(connector.Name).Open(fh.ctx, &_connector.StorageConfig{
				Role:     cRole,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.StreamType:
			connection, err = _connector.RegisteredStream(connector.Name).Open(fh.ctx, &_connector.StreamConfig{
				Role:     cRole,
				Settings: c.Settings,
				Firehose: fh,
			})
		case state.WebsiteType:
			connection, err = _connector.RegisteredWebsite(connector.Name).Open(fh.ctx, &_connector.WebsiteConfig{
				Role:     cRole,
				Settings: c.Settings,
				Firehose: fh,
			})
		}

	}
	if err != nil {
		return nil, err
	}
	connectionUI, ok := connection.(_connector.UI)
	if !ok {
		return nil, errors.BadRequest("connector %d does not have a UI", c.ID)
	}

	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	form, alert, err := connectionUI.ServeUI(event, values)
	if err != nil {
		if err == ui.ErrEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s connector",
				event, connector.Name)
		}
		return nil, err
	}

	return marshalUIFormAlert(form, alert, ui.Role(c.Role))
}

// AddAction adds action to connection, returning the identifier of the added
// action. The connection must be a destination of type app.
//
// The action name must be a non-empty valid UTF-8 encoded string and cannot be
// longer than 60 runes. The action must have a mapping associated or a
// function, and cannot have both.
//
// The action endpoint must be the identifier of one the endpoints supported by
// the action type.
//
// If it has a mapping, the names of the properties in which the values are
// mapped must be present in the action type schema.
//
// If it has a transformation, such transformation should have at least one
// input and one output property, its source should be a valid Python source,
// and the names of the properties in the output schema must be present in the
// action type schema.
// TODO(Gianluca): specify how this transformation function should be written,
// depending on the use on the events dispatcher.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
func (this *Connection) AddAction(action ActionToSet) (int, error) {
	c := this.connection
	if c.Role != state.DestinationRole {
		return 0, errors.BadRequest("connection is not a destination")
	}
	connector := c.Connector()
	if connector.Type != state.AppType {
		return 0, errors.BadRequest("connection type is not app")
	}
	actionTypes, err := this.ActionTypes()
	if err != nil {
		return 0, err
	}
	err = validateAction(action, actionTypes)
	if err != nil {
		return 0, errors.BadRequest(err.Error())
	}
	n := state.AddConnectionActionNotification{
		Connection:     c.ID,
		ActionType:     action.ActionType,
		Name:           action.Name,
		Enabled:        action.Enabled,
		Endpoint:       action.Endpoint,
		Mapping:        action.Mapping,
		Transformation: (*state.Transformation)(action.Transformation),
	}
	n.Filter.Logical = action.Filter.Logical
	n.Filter.Conditions = make([]state.ActionFilterConditionNotification, len(action.Filter.Conditions))
	for i := range n.Filter.Conditions {
		n.Filter.Conditions[i] = (state.ActionFilterConditionNotification)(action.Filter.Conditions[i])
	}
	ctx := context.Background()
	var filter, mapping, tIn, tOut, tSource []byte
	filter, err = json.Marshal(action.Filter)
	if err != nil {
		return 0, err
	}
	if action.Mapping != nil {
		mapping, err = json.Marshal(action.Mapping)
		if err != nil {
			return 0, err
		}
	}
	if t := action.Transformation; t != nil {
		tIn, err = json.Marshal(t.In)
		if err != nil {
			return 0, err
		}
		tOut, err = json.Marshal(t.Out)
		if err != nil {
			return 0, err
		}
		tSource = []byte(t.PythonSource)
	}
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		query := "INSERT INTO actions (connection, action_type, name, enabled,\n" +
			"endpoint, filter, mapping, transformation.in_types,\n" +
			"transformation.out_types, transformation.python_source)\n" +
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)\n" +
			"RETURNING id"
		err := tx.QueryRow(ctx, query, n.Connection, n.ActionType, n.Name,
			n.Enabled, n.Endpoint, string(filter), string(mapping), string(tIn),
			string(tOut), string(tSource)).
			Scan(&n.ID)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) && postgres.ErrConstraintName(err) == "connections_connection_fkey" {
				err = errors.Unprocessable(ConnectorNotExist, "connection %d does not exist", n.Connection)
			}
			return err
		}
		return tx.Notify(ctx, n)
	})
	return n.ID, err
}

// SetTransformation sets the transformation of the connection. The connection
// must be an app, database or file connection. Calling this method with a nil
// transformation removes the transformation associated to the connection.
//
// The transformation, when not nil, should have at least one input and one
// output property, and its source should be a valid Python source declaring a
// 'transform' function which takes and returns a Python dictionary as
// parameter.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
// It returns an errors.UnprocessableError error with code AlreadyHasMappings if
// the connection has one or more mappings associated to it.
func (this *Connection) SetTransformation(transformation *Transformation) error {

	id := this.connection.ID

	// Validate the connection type.
	switch typ := this.connection.Connector().Type; typ {
	case state.AppType, state.DatabaseType, state.FileType:
		// Ok.
	default:
		return errors.BadRequest("connection %d has no type app, database or file", id)
	}

	// Remove the transformation, if it should be removed.
	if transformation == nil {
		n := state.SetConnectionTransformationNotification{
			Connection:     id,
			Transformation: nil,
		}
		ctx := context.Background()
		err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
			query := "UPDATE connections SET\n" +
				"transformation.in_types = '',\n" +
				"transformation.out_types = '',\n" +
				"transformation.python_source = ''\n" +
				"WHERE id = $1"
			result, err := tx.Exec(ctx, query, n.Connection)
			if err != nil {
				return err
			}
			if result.RowsAffected() == 0 {
				return errors.NotFound("connection %d does not exist", n.Connection)
			}
			return tx.Notify(ctx, n)
		})
		return err
	}

	// Validate the transformation.
	err := validateTransformation(transformation)
	if err != nil {
		return errors.BadRequest(err.Error())
	}

	n := state.SetConnectionTransformationNotification{
		Connection:     id,
		Transformation: (*state.Transformation)(transformation),
	}

	ctx := context.Background()

	// Prepare the input/output types to be written on the database.
	inTypes, err := n.Transformation.In.MarshalJSON()
	if err != nil {
		return err
	}
	outTypes, err := n.Transformation.Out.MarshalJSON()
	if err != nil {
		return err
	}

	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {

		// Ensure that the connection does not already have associated mappings.
		err := tx.QueryVoid(ctx,
			"SELECT FROM connections_mappings WHERE connection = $1",
			n.Connection)
		if err == nil {
			return errors.Unprocessable(AlreadyHasMappings,
				"connection %d already has mappings associated", n.Connection)
		}
		if err != sql.ErrNoRows {
			return err
		}

		// Write the transformation of the connection.
		query := "UPDATE connections SET\n" +
			"transformation.in_types = $1,\n" +
			"transformation.out_types = $2,\n" +
			"transformation.python_source = $3\n" +
			"WHERE id = $4"
		result, err := tx.Exec(ctx, query, inTypes, outTypes,
			n.Transformation.PythonSource, n.Connection)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("connection %d does not exist", n.Connection)
		}

		return tx.Notify(ctx, n)

	})

	return err

}

// SetMappings sets the mappings of the connection. The connection must be an
// app, database or file connection.
//
// InProperties and OutProperties of a mapping should contain valid property
// names and are both not-empty, and one of PredefinedFunc and MappingCustomFunc
// are provided (or none, for "one to one" mappings).
//
// "One-to-one" mappings must have one input and one output properties.
//
// For mappings with non-nil PredefinedFunc, the pointed value must be the ID of
// a predefined mapping function and the count of the input/output properties
// must match the count of the input/output parameters of the referred
// predefined function.
//
// For mappings with non-nil MappingCustomFunc, the pointed value must be a
// descriptor of a custom transformation function, with a valid Python source
// code, and the count of the input/output properties must match the count of
// the input/output types of the referred custom function.
//
// It returns an errors.NotFoundError error if the connection does not exist
// anymore.
// It returns an errors.UnprocessableError error with code
// AlreadyHasTransformation if one or more mappings are added when the
// connection already has a transformation associated to it.
func (this *Connection) SetMappings(mappings []*Mapping) error {

	// Validate the connection type.
	switch typ := this.connection.Connector().Type; typ {
	case state.AppType, state.DatabaseType, state.FileType:
		// Ok.
	default:
		return errors.BadRequest("cannot set mappings on a connection of type %q", typ)
	}

	// Validate the mappings.
	for _, m := range mappings {

		if m.PredefinedFunc != nil && m.CustomFunc != nil {
			return errors.BadRequest("mapping cannot have both predefined function and custom function")
		}

		// Validate the property names.
		for _, p := range m.InProperties {
			if !types.IsValidPropertyName(p) {
				return errors.BadRequest("input property name %q is not valid", p)
			}
		}
		for _, p := range m.OutProperties {
			if !types.IsValidPropertyName(p) {
				return errors.BadRequest("output property name %q is not valid", p)
			}
		}

		// Validate a "one-to-one" mapping.
		if m.PredefinedFunc == nil && m.CustomFunc == nil {
			if len(m.InProperties) != 1 {
				return errors.BadRequest("one-to-one mapping should have one input property, got %d", len(m.InProperties))
			}
			if len(m.OutProperties) != 1 {
				return errors.BadRequest("one-to-one mapping should have one output property, got %d", len(m.OutProperties))
			}
			continue
		}

		// Validate a mapping with predefined function.
		if m.PredefinedFunc != nil {
			funcID := *m.PredefinedFunc
			f, ok := predefinedFuncDefinitionByID(state.PredefinedFunc(funcID))
			if !ok {
				return errors.BadRequest("predefined function with ID %d does not exist", funcID)
			}
			fInProps, fOutProps := f.In.Properties(), f.Out.Properties()
			if len(fInProps) != len(m.InProperties) {
				return errors.BadRequest("predefined function expects %d input properties, got %d", len(fInProps), len(m.InProperties))
			}
			if len(fOutProps) != len(m.OutProperties) {
				return errors.BadRequest("predefined function expects %d output properties, got %d", len(fOutProps), len(m.OutProperties))
			}
			continue
		}

		// Validate a mapping with custom transformation function.
		if m.CustomFunc.Source == "" {
			return errors.BadRequest("custom function cannot have empty source code")
		}
		if len(m.CustomFunc.InTypes) == 0 {
			return errors.BadRequest("custom function should have at least one input type")
		}
		if len(m.CustomFunc.OutTypes) == 0 {
			return errors.BadRequest("custom function should have at least one output type")
		}
		// TODO(Gianluca): consider validating the Python source code.
		if len(m.CustomFunc.InTypes) != len(m.InProperties) {
			return errors.BadRequest("custom function expects %d input properties, got %d", len(m.CustomFunc.InTypes), len(m.InProperties))
		}
		if len(m.CustomFunc.OutTypes) != len(m.OutProperties) {
			return errors.BadRequest("custom function expects %d output properties, got %d", len(m.CustomFunc.OutTypes), len(m.OutProperties))
		}
		inSchema, err := typesToSchema(m.CustomFunc.InTypes, m.InProperties)
		if err != nil || !inSchema.Valid() {
			return errors.BadRequest("invalid input types for custom function: %s", err)
		}
		outSchema, err := typesToSchema(m.CustomFunc.OutTypes, m.OutProperties)
		if err != nil || !outSchema.Valid() {
			return errors.BadRequest("invalid output types for custom function: %s", err)
		}

	}

	n := state.SetConnectionMappingsNotification{Connection: this.connection.ID}

	// Prepare the mappings for the notification and marshal the input types
	// into JSON.
	n.Mappings = make([]*state.Mapping, len(mappings))
	customFuncInTypes := make([][]byte, len(mappings))
	customFuncOutTypes := make([][]byte, len(mappings))
	for i, m := range mappings {
		n.Mappings[i] = &state.Mapping{
			InProperties:   m.InProperties,
			OutProperties:  m.OutProperties,
			PredefinedFunc: (*state.PredefinedFunc)(m.PredefinedFunc),
			CustomFunc:     (*state.MappingCustomFunc)(m.CustomFunc),
		}
		if m.CustomFunc != nil {
			var err error
			customFuncInTypes[i], err = json.Marshal(m.CustomFunc.InTypes)
			if err != nil {
				return err
			}
			customFuncOutTypes[i], err = json.Marshal(m.CustomFunc.OutTypes)
			if err != nil {
				return err
			}
		}
	}

	ctx := context.Background()

	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {

		// If adding one or more mappings, ensure that the connection does not
		// already have an associated transformation.
		if len(n.Mappings) > 0 {
			err := tx.QueryVoid(ctx, "SELECT FROM connections WHERE id = $1 AND (transformation).in_types <> ''", n.Connection)
			if err == nil {
				return errors.Unprocessable(AlreadyHasTransformation, "connection already has a transformation associated")
			}
			if err != sql.ErrNoRows {
				return err
			}
		}

		_, err := tx.Exec(ctx, "DELETE FROM connections_mappings WHERE connection = $1", n.Connection)
		if err != nil {
			return err
		}

		stmt, err := tx.Prepare(ctx, "INSERT INTO connections_mappings\n"+
			"(connection, position, in_properties, out_properties, predefined_func,\n"+
			"custom_func.in_types, custom_func.out_types, custom_func.source)\n"+
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8)")
		if err != nil {
			return err
		}
		for i, m := range n.Mappings {
			position := i + 1
			var customFunc struct {
				in_types  []byte
				out_types []byte
				source    string
			}
			if m.CustomFunc != nil {
				customFunc.in_types = customFuncInTypes[i]
				customFunc.out_types = customFuncOutTypes[i]
				customFunc.source = m.CustomFunc.Source
			}
			_, err := stmt.Exec(ctx, n.Connection, position, m.InProperties, m.OutProperties,
				m.PredefinedFunc, customFunc.in_types, customFunc.out_types, customFunc.source)
			if err != nil {
				if postgres.IsForeignKeyViolation(err) {
					if postgres.ErrConstraintName(err) == "connections_mappings_connection_fkey" {
						err = errors.NotFound("connection %d does not exist", n.Connection)
					}
				}
				return err
			}

		}
		err = stmt.Close()
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// SetStorage sets the storage of the connection. The connection must be a file
// connection. storage is the storage connection. The connection and the
// storage must have the same role. As a special case, the current storage of
// the file, if there is one, is removed if the storage argument is 0.
//
// If the connection does not exist anymore, it returns an errors.NotFoundError
// error.
// If the storage does not exist, it returns an errors.UnprocessableError error
// with code StorageNotExist.
func (this *Connection) SetStorage(storage int) error {

	if storage < 0 || storage > maxInt32 {
		return errors.BadRequest("storage identifier %d is not valid", storage)
	}

	c := this.connection
	if c.Connector().Type != state.FileType {
		return errors.BadRequest("file is not a file connector")
	}
	var s *state.Connection
	if storage > 0 {
		var ok bool
		s, ok = c.Workspace().Connection(storage)
		if !ok {
			return errors.Unprocessable(StorageNotExist, "storage %d does not exist", storage)
		}
		if s.Connector().Type != state.StorageType {
			return errors.BadRequest("connection %d is not a storage", storage)
		}
		if s.Role != c.Role {
			if c.Role == state.SourceRole {
				return errors.BadRequest("storage %d is not a source", storage)
			}
			return errors.BadRequest("storage %d is not a destination", storage)
		}
	}

	n := state.SetConnectionStorageNotification{
		Connection: c.ID,
		Storage:    storage,
	}

	ctx := context.Background()

	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections SET storage = NULLIF($1, 0) WHERE id = $2", n.Storage, n.Connection)
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				if postgres.ErrConstraintName(err) == "connections_storage_fkey" {
					err = errors.Unprocessable(StorageNotExist, "storage %d does not exist", storage)
				}
			}
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("connection %d does not exist", n.Connection)
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// SetUsersQuery sets the users query of connection. The connection must be a
// database source connection. query must be UTF-8 encoded, it cannot be longer
// than 16,777,215 runes and must contain the ':limit' placeholder.
//
// If the connection does not exist anymore, it returns an errors.NotFoundError
// error.
func (this *Connection) SetUsersQuery(query string) error {

	if !utf8.ValidString(query) {
		return errors.BadRequest("query is not UTF-8 encoded")
	}
	if utf8.RuneCountInString(query) > queryMaxSize {
		return errors.BadRequest("query is longer than %d", queryMaxSize)
	}
	if !strings.Contains(query, ":limit") {
		return errors.BadRequest("query does not contain the ':limit' placeholder")
	}

	c := this.connection
	if c.Connector().Type != state.DatabaseType {
		return errors.BadRequest("connection %d is not a database", c.ID)
	}
	if c.Role != state.SourceRole {
		return errors.BadRequest("database %d is not a source", c.ID)
	}

	n := state.SetConnectionUserQueryNotification{
		Connection: c.ID,
		Query:      query,
	}

	ctx := context.Background()

	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE connections\nSET users_query = $1 WHERE id = $2", query, c.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("connection %d does not exist", c.ID)
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// ConnectionsStats represents the statistics on a connection for the last 24
// hours.
type ConnectionsStats struct {
	UsersIn [24]int // ingested users per hour
}

// Stats returns statistics on the connection for the last 24 hours.
//
// It returns an errors.Notfound error if the connection does not exist
// anymore.
func (this *Connection) Stats() (*ConnectionsStats, error) {
	now := time.Now().UTC()
	toSlot := statsTimeSlot(now)
	fromSlot := toSlot - 23
	stats := &ConnectionsStats{
		UsersIn: [24]int{},
	}
	query := "SELECT time_slot, users_in\nFROM connections_stats\nWHERE connection = $1 AND time_slot BETWEEN $2 AND $3"
	err := this.db.QueryScan(context.Background(), query, this.connection.ID, fromSlot, toSlot, func(rows *postgres.Rows) error {
		var err error
		var slot, usersIn int
		for rows.Next() {
			if err = rows.Scan(&slot, &usersIn); err != nil {
				return err
			}
			stats.UsersIn[slot-fromSlot] = usersIn
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// reloadActionTypes reloads the action types for the destination connection of
// type app.
func (this *Connection) reloadActionTypes() error {

	c := this.connection
	connector := c.Connector()
	cRole := _connector.Role(c.Role)

	// Retrieve the resource secrets, if necessary.
	var clientSecret, resourceCode, accessToken string
	if r, ok := c.Resource(); ok {
		clientSecret = connector.OAuth.ClientSecret
		resourceCode = r.Code
		var err error
		accessToken, err = freshAccessToken(this.db, r)
		if err != nil {
			return err
		}
	}

	// Retrieve the app.
	app := _connector.RegisteredApp(connector.Name)

	// Instantiate a firehose.
	fh := this.newFirehose(context.Background())
	connection, err := app.Open(fh.ctx, &_connector.AppConfig{
		Role:         cRole,
		Settings:     c.Settings,
		Firehose:     fh,
		ClientSecret: clientSecret,
		Resource:     resourceCode,
		AccessToken:  accessToken,
	})
	if err != nil {
		return err
	}

	// Retrieve the action types from the connection.
	actionTypes, err := connection.ActionTypes()
	if err != nil {
		return err
	}

	n := state.SetConnectionActionTypesNotification{
		Connection:  c.ID,
		ActionTypes: make([]state.ActionTypeNotification, len(actionTypes)),
	}
	for i, at := range actionTypes {
		n.ActionTypes[i] = state.ActionTypeNotification{
			ID:          at.ID,
			Name:        at.Name,
			Description: at.Description,
			Endpoints:   at.Endpoints,
			Schema:      at.Schema,
		}
	}

	ctx := context.Background()

	rawActionTypes, err := json.Marshal(actionTypes)
	if err != nil {
		return err
	}

	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err = tx.Exec(ctx, "UPDATE connections SET \"action_types\" = $1 WHERE id = $2", rawActionTypes, n.Connection)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})

	return nil
}

// newFirehose returns a new Firehose.
func (this *Connection) newFirehose(ctx context.Context) *firehose {
	return this.newFirehoseForConnection(ctx, this.connection)
}

// newFirehose returns a new Firehose for the connection c.
func (this *Connection) newFirehoseForConnection(ctx context.Context, c *state.Connection) *firehose {
	var resource int
	if r, ok := c.Resource(); ok {
		resource = r.ID
	}
	fh := &firehose{
		db:         this.db,
		connection: c,
		resource:   resource,
	}
	fh.ctx, fh.cancel = context.WithCancel(ctx)
	return fh
}

// readGRUsers reads the Golden Record users with the given IDs.
func (this *Connection) readGRUsers(ids []int) ([]map[string]any, error) {
	return nil, nil // TODO(Gianluca): implement.
}

var errRecordStop = errors.New("stop record")

// reloadSchema reloads the schema of the connection. The connection must be a
// source app, database or file connection.
func (this *Connection) reloadSchema() error {

	c := this.connection
	connector := c.Connector()
	switch connector.Type {
	case state.AppType, state.DatabaseType:
	case state.FileType:
		if _, ok := c.Storage(); !ok {
			return errors.New("file connection has not storage")
		}
	default:
		return fmt.Errorf("cannot import properties from a %s connection",
			strings.ToLower(connector.Type.String()))
	}
	if c.Role == state.DestinationRole {
		return errors.New("cannot import from a destination")
	}

	cRole := _connector.Role(c.Role)

	var schema types.Type

	switch connector.Type {
	case state.AppType:

		var clientSecret, resourceCode, accessToken string
		if r, ok := c.Resource(); ok {
			clientSecret = connector.OAuth.ClientSecret
			resourceCode = r.Code
			var err error
			accessToken, err = freshAccessToken(this.db, r)
			if err != nil {
				return err
			}
		}

		fh := this.newFirehose(context.Background())
		connection, err := _connector.RegisteredApp(connector.Name).Open(fh.ctx, &_connector.AppConfig{
			Role:         cRole,
			Settings:     c.Settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			return err
		}
		schema, _, err = connection.Schemas()
		if err != nil {
			return err
		}
		if !schema.Valid() {
			return fmt.Errorf("connection %d returned an invalid schema", c.ID)
		}
		schema = schema.AsRole(types.Role(c.Role))
		if !schema.Valid() {
			return errors.New("connection has returned a schema without source properties")
		}

	case state.DatabaseType:

		usersQuery, err := compileConnectionQuery(c.UsersQuery, 0)
		if err != nil {
			return err
		}
		fh := this.newFirehose(context.Background())
		connection, err := _connector.RegisteredDatabase(connector.Name).Open(fh.ctx, &_connector.DatabaseConfig{
			Role:     cRole,
			Settings: c.Settings,
			Firehose: fh,
		})
		if err != nil {
			return err
		}
		columns, rows, err := connection.Query(usersQuery)
		if err != nil {
			return err
		}
		err = rows.Close()
		if err != nil {
			return err
		}
		properties := make([]types.Property, len(columns))
		for i, col := range columns {
			properties[i].Name = col.Name
			properties[i].Type = col.Type
		}
		schema = types.Object(properties)

	case state.FileType:

		var ctx = context.Background()

		// Get the file reader.
		var files *fileReader
		{
			s, ok := c.Storage()
			if !ok {
				return errors.New("file connection has not storage")
			}
			fh := this.newFirehose(ctx)
			ctx = fh.ctx
			connection, err := _connector.RegisteredStorage(s.Connector().Name).Open(ctx, &_connector.StorageConfig{
				Role:     cRole,
				Settings: c.Settings,
				Firehose: fh,
			})
			if err != nil {
				return err
			}
			files = newFileReader(connection)
		}

		// Connect to the file connector and read only the columns.
		fh := this.newFirehose(ctx)
		file, err := _connector.RegisteredFile(connector.Name).Open(fh.ctx, &_connector.FileConfig{
			Role:     cRole,
			Settings: c.Settings,
			Firehose: fh,
		})
		if err != nil {
			return err
		}

		// Read only the columns.
		records := fh.newRecordWriter(identityColumn, timestampColumn, true)
		err = file.Read(files, records)
		if err != nil && err != errRecordStop {
			return err
		}
		properties := make([]types.Property, len(records.columns))
		for i, col := range records.columns {
			properties[i].Name = col.Name
			properties[i].Type = col.Type
		}
		schema = types.Object(properties)

	}

	// Update the schema.
	rawSchema, err := schema.MarshalJSON()
	if err != nil {
		return fmt.Errorf("cannot marshal schema of connection %d: %s", c.ID, err)
	}
	if utf8.RuneCount(rawSchema) > rawSchemaMaxSize {
		return fmt.Errorf("cannot marshal schema of the connection %d: data is too large", c.ID)
	}

	n := state.SetConnectionUserSchemaNotification{
		Connection: c.ID,
		Schema:     schema,
	}

	ctx := context.Background()

	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err = tx.Exec(ctx, "UPDATE connections SET \"schema\" = $1 WHERE id = $2", rawSchema, n.Connection)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// userSchema returns the user schema and the paths of the mapped properties of
// the connection.
func (this *Connection) userSchema() (types.Type, []_connector.PropertyPath, error) {

	c := this.connection

	// Collect the paths of the properties used in transformation or mappings.
	var paths []_connector.PropertyPath
	if t := c.Transformation(); t != nil {
		for _, name := range t.In.PropertiesNames() {
			paths = append(paths, []string{name})
		}
	}
	for _, m := range c.Mappings() {
		for _, in := range m.InProperties {
			paths = append(paths, []string{in})
		}
	}

	// Create a schema with only the properties mapped.
	mapped := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		mapped[p[0]] = struct{}{}
	}
	mappedProperties := make([]types.Property, 0, len(paths))
	for _, property := range c.Schema.Properties() {
		if _, ok := mapped[property.Name]; ok {
			mappedProperties = append(mappedProperties, property)
		}
	}
	schema := c.Schema
	if mappedProperties != nil {
		schema = types.Object(mappedProperties)
	}

	return schema, paths, nil
}

const noQueryLimit = -1

// compileConnectionQuery compiles the given query and returns it. If limit is
// noQueryLimit removes the ':limit' placeholder (along with '[[' and ']]');
// otherwise, replaces the placeholders with limit.
func compileConnectionQuery(query string, limit int) (string, error) {
	p := strings.Index(query, ":limit")
	if p == -1 {
		return "", errors.BadRequest("query does not contain the ':limit' placeholder")
	}
	s1 := strings.Index(query[:p], "[[")
	if s1 == -1 {
		return "", errors.BadRequest("query does not contain '[['")
	}
	n := len(":limit")
	s2 := strings.Index(query[p+n:], "]]")
	if s2 == -1 {
		return "", errors.BadRequest("query does not contain ']]'")
	}
	s2 += p + n + 2
	if limit == noQueryLimit {
		return query[:s1] + query[s2:], nil
	}
	return query[:s1] + strings.ReplaceAll(query[s1+2:s2-2], ":limit", strconv.Itoa(limit)) + query[s2:], nil
}

// fileReader implements the connector.FileReader interface.
type fileReader struct {
	s _connector.StorageConnection
}

// newFileReader returns a new file reader for the given storage.
func newFileReader(storage _connector.StorageConnection) *fileReader {
	return &fileReader{s: storage}
}

// Reader returns a ReadCloser from which to read the file at the given
// path and its last update time.
// It is the caller's responsibility to close the returned reader.
func (files *fileReader) Reader(path string) (io.ReadCloser, time.Time, error) {
	return files.s.Reader(path)
}

// isServerKey reports whether key can be a server key.
func isServerKey(key string) bool {
	if len(key) != 32 {
		return false
	}
	_, err := base62.DecodeString(key)
	return err == nil
}

// encodeServerKey encodes a binary server key to its base62 form and returns
// it.
func encodeServerKey(key []byte) string {
	return base62.EncodeToString(key)[0:32]
}

// generateServerKey generates a server key in its base62 form.
func generateServerKey() (string, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return "", errors.New("cannot generate a server key")
	}
	return base62.EncodeToString(key)[0:32], nil
}

// marshalUIFormAlert marshals form with given role and alert in JSON format.
// form and alert can be nil or not, independently of each other.
func marshalUIFormAlert(form *ui.Form, alert *ui.Alert, role ui.Role) ([]byte, error) {

	if form == nil && alert == nil {
		return []byte("null"), nil
	}

	var b bytes.Buffer
	enc := json.NewEncoder(&b)

	b.WriteString("{")

	// Serialize the form, if present.
	if form != nil {

		// Makes the keys of form.Values to have the same case as the Name field of the components.
		values := map[string]any{}
		if len(form.Values) > 0 {
			err := json.Unmarshal(form.Values, &values)
			if err != nil {
				return nil, err
			}
		}

		comma := false
		b.WriteString(`"Form":{"Fields":[`)
		for _, field := range form.Fields {
			ok, err := marshalUIComponent(&b, field, role, values, comma)
			if err != nil {
				return nil, err
			}
			if ok {
				comma = true
			}
		}
		b.WriteString(`],"Actions":`)
		err := enc.Encode(form.Actions)
		if err != nil {
			return nil, err
		}
		if len(form.Values) > 0 {
			b.WriteString(`,"Values":`)
			err = json.NewEncoder(&b).Encode(values)
			if err != nil {
				return nil, err
			}
		}
		b.WriteString("}")

	}

	// Serialize the alert, if present.
	if alert != nil {
		if form != nil {
			b.WriteString(",")
		}
		b.WriteString(`"Alert":{"Message":`)
		err := enc.Encode(alert.Message)
		if err != nil {
			return nil, err
		}
		b.WriteString(`,"Variant":"`)
		b.WriteString(alert.Variant.String())
		b.WriteString(`"`)
		b.WriteString("}")
	}

	b.WriteString(`}`)

	return b.Bytes(), nil
}

// adjustValuesCase adjusts the case of keys of values.
func adjustValuesCase(key string, values map[string]any) {
	var found struct {
		key   string
		value any
	}
	for k, v := range values {
		if strings.EqualFold(k, key) {
			found.key = k
			found.value = v
			break
		}
	}
	if found.key == "" {
		return
	}
	delete(values, found.key)
	values[key] = found.value
}

// marshalUIComponent marshals component with the given role in JSON format. If
// comma is true, it prepends a comma. Returns whether it has been marhalled.
func marshalUIComponent(b *bytes.Buffer, component ui.Component, role ui.Role, values map[string]any, comma bool) (bool, error) {
	rv := reflect.ValueOf(component).Elem()
	rt := rv.Type()
	if role != ui.BothRole {
		if r := ui.Role(rv.FieldByName("Role").Int()); r != ui.BothRole && r != role {
			return false, nil
		}
	}
	if comma {
		b.WriteString(`,`)
	}
	b.WriteString(`{"ComponentType":"`)
	b.WriteString(rt.Name())
	b.WriteString(`"`)
	for j := 0; j < rt.NumField(); j++ {
		name := rt.Field(j).Name
		if name == "Role" {
			continue
		}
		field := rv.Field(j)
		if name == "Name" && values != nil {
			adjustValuesCase(field.String(), values)
		}
		b.WriteString(`,"`)
		b.WriteString(name)
		b.WriteString(`":`)
		var err error
		switch field := field.Interface().(type) {
		case ui.Component:
			_, err = marshalUIComponent(b, field, role, values, false)
		case []ui.FieldSet:
			b.WriteByte('[')
			comma = false
			for _, set := range field {
				var ok bool
				ok, err = marshalUIFieldSet(b, set, role, values, comma)
				if ok {
					comma = true
				}
			}
			b.WriteByte(']')
		default:
			err = json.NewEncoder(b).Encode(field)
		}
		if err != nil {
			return false, err
		}
	}
	b.WriteString(`}`)
	return true, nil
}

// marshalUIFieldSet marshals fieldSet with the given role in JSON format. If
// comma is true, it prepends a comma. Returns whether it has been marhalled.
func marshalUIFieldSet(b *bytes.Buffer, fieldSet ui.FieldSet, role ui.Role, values map[string]any, comma bool) (bool, error) {
	if role != ui.BothRole {
		if fieldSet.Role != ui.BothRole && fieldSet.Role != role {
			return false, nil
		}
	}
	name := fieldSet.Name
	if values != nil {
		adjustValuesCase(name, values)
	}
	if comma {
		b.WriteByte(',')
	}
	b.WriteString(`{"Name":`)
	_ = json.NewEncoder(b).Encode(name)
	b.WriteString(`,"Label":`)
	_ = json.NewEncoder(b).Encode(fieldSet.Label)
	b.WriteString(`,"Fields":[`)
	comma = false
	for _, c := range fieldSet.Fields {
		var valuesOfSet map[string]any
		switch vs := values[name].(type) {
		case nil:
		case map[string]any:
			valuesOfSet = vs
		default:
			return false, fmt.Errorf("expected a map[string]any value for field set %s, got %T", name, values[name])
		}
		ok, err := marshalUIComponent(b, c, role, valuesOfSet, comma)
		if err != nil {
			return false, err
		}
		if ok {
			comma = true
		}
	}
	b.WriteString(`]}`)
	return true, nil
}

// abbreviate abbreviates s to almost n runes. If s is longer than n runes,
// the abbreviated string terminates with "...".
func abbreviate(s string, n int) string {
	const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace
	s = strings.TrimRight(s, spaces)
	if len(s) <= n {
		return s
	}
	if n < 3 {
		return ""
	}
	p := 0
	n2 := 0
	for i := range s {
		switch p {
		case n - 2:
			n2 = i
		case n:
			break
		}
		p++
	}
	if p < n {
		return s
	}
	if p = strings.LastIndexAny(s[:n2], spaces); p > 0 {
		s = strings.TrimRight(s[:p], spaces)
	} else {
		s = ""
	}
	if l := len(s) - 1; l >= 0 && (s[l] == '.' || s[l] == ',') {
		s = s[:l]
	}
	return s + "..."
}

// exportUser returns a user to export (with the given ID) applying the given
// mappings to the properties.
//
// TODO(Gianluca): note that this code has never been tested and misses some
// validation parts, as the export procedure is still work in progress.
//
// TODO(Gianluca): when the export implementation will be discussed and
// refactored, add support for output transformations.
func exportUser(id string, properties map[string]any, mappings []*state.Mapping) (_connector.User, error) {

	user := _connector.User{
		ID:         id,
		Properties: map[string]any{},
	}

	pool := transformations.NewPool()

	for _, m := range mappings {

		// Ensure that the input properties exist.
		for _, p := range m.InProperties {
			if _, ok := properties[p]; !ok {
				return _connector.User{}, exportError{fmt.Errorf("property %q not found", p)}
			}
		}

		if m.PredefinedFunc == nil && m.CustomFunc == nil {

			// "One to one" mapping.
			user.Properties[m.OutProperties[0]] = properties[m.InProperties[0]]

		} else if m.PredefinedFunc != nil {

			// Predefined transformation.
			f, _ := predefinedFuncDefinitionByID(*m.PredefinedFunc)
			in := make([]any, len(m.InProperties))
			// TODO(Gianluca): this code that makes the validation can be
			// simplified by changing the APIs of the 'types' package.
			values := map[string]any{}
			for i, p := range m.InProperties {
				values[p] = properties[p]
				in[i] = properties[m.InProperties[i]]
			}
			j, _ := json.Marshal(values)
			_, err := types.Decode(bytes.NewReader(j), f.In)
			if err != nil {
				return _connector.User{}, exportError{err}
			}
			out := callPredefinedFunc(f, in)
			for i, outName := range m.OutProperties {
				user.Properties[outName] = out[i]
			}

		} else {

			// Mapping with a custom transformation function.
			in := make([]any, len(m.InProperties))
			for i := range in {
				in[i] = properties[m.InProperties[i]]
			}
			out, err := pool.Run(context.Background(), m.CustomFunc.Source, in)
			if err != nil {
				return _connector.User{}, exportError{fmt.Errorf("error while calling transformation function of mapping: %s", err)}
			}
			for i, name := range m.OutProperties {
				user.Properties[name] = out[i]
			}

		}
	}

	return user, nil

}

// ConnectionHealth is an indicator of the current state of a connection.
type ConnectionHealth int

const (
	Healthy ConnectionHealth = iota
	NoRecentData
	RecentError
	AccessDenied
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if health is not a valid ConnectionHealth value.
func (health ConnectionHealth) MarshalJSON() ([]byte, error) {
	return []byte(`"` + health.String() + `"`), nil
}

// String returns the string representation of health.
// It panics if health is not a valid ConnectionHealth value.
func (health ConnectionHealth) String() string {
	switch health {
	case Healthy:
		return "Healthy"
	case NoRecentData:
		return "NoRecentData"
	case RecentError:
		return "RecentError"
	case AccessDenied:
		return "AccessDenied"
	}
	panic("invalid connection health")
}

// ConnectionRole represents a connection role.
type ConnectionRole int

const (
	SourceRole      ConnectionRole = iota + 1 // source
	DestinationRole                           // destination
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if role is not a valid ConnectionRole value.
func (role ConnectionRole) MarshalJSON() ([]byte, error) {
	return []byte(`"` + role.String() + `"`), nil
}

// String returns the string representation of role.
// It panics if role is not a valid ConnectionRole value.
func (role ConnectionRole) String() string {
	switch role {
	case SourceRole:
		return "Source"
	case DestinationRole:
		return "Destination"
	}
	panic("invalid connection role")
}

var null = []byte("null")

// UnmarshalJSON implements the json.Unmarshaler interface.
func (role *ConnectionRole) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a apis.ConnectionRole value: %s", err)
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.ConnectionRole value", v)
	}
	var r ConnectionRole
	switch s {
	case "Source":
		r = SourceRole
	case "Destination":
		r = DestinationRole
	default:
		return fmt.Errorf("invalid apis.ConnectionRole: %s", s)
	}
	*role = r
	return nil
}

// validateTransformation validates the given not-nil transformation, returning
// nil if the transformation is valid or an error with an error message
// explaining why the transformation is invalid.
func validateTransformation(t *Transformation) error {
	if t == nil {
		panic("t is nil")
	}
	if !t.In.Valid() || t.In.PhysicalType() != types.PtObject {
		return errors.New("input schema is invalid")
	}
	if len(t.In.Properties()) == 0 {
		return errors.New("input schema does not have properties")
	}
	if !t.Out.Valid() || t.Out.PhysicalType() != types.PtObject {
		return errors.New("output schema is invalid")
	}
	if len(t.Out.Properties()) == 0 {
		return errors.New("output schema does not have properties")
	}
	// TODO(Gianluca): do a proper validation of the Python source code.
	if !strings.Contains(t.PythonSource, "def transform") {
		return errors.New("Python source code does not contain 'transform' function")
	}
	return nil
}
