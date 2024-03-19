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
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/connectors"
	"chichi/apis/datastore"
	"chichi/apis/datastore/expr"
	"chichi/apis/datastore/warehouses"
	"chichi/apis/datastore/warehouses/clickhouse"
	"chichi/apis/datastore/warehouses/diffschemas"
	"chichi/apis/datastore/warehouses/postgresql"
	"chichi/apis/datastore/warehouses/snowflake"
	"chichi/apis/encoding"
	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/connector/types"

	"github.com/jxskiss/base62"
	"golang.org/x/exp/maps"
)

const (
	maxEventsListenedTo = 1000 // maximum number of processed events listened to.
)

var (
	AlreadyConnected     errors.Code = "AlreadyConnected"
	CurrentlyConnected   errors.Code = "CurrentlyConnected"
	DataWarehouseFailed  errors.Code = "DataWarehouseFailed"
	InvalidSchemaChange  errors.Code = "InvalidSchemaChange"
	InvalidSettings      errors.Code = "InvalidSettings"
	InvalidWarehouseType errors.Code = "InvalidWarehouseType"
	NoWarehouse          errors.Code = "NoWarehouse"
	NotConnected         errors.Code = "NotConnected"
	OrderNotExist        errors.Code = "OrderNotExist"
	OrderTypeNotSortable errors.Code = "OrderTypeNotSortable"
	PropertyNotExist     errors.Code = "PropertyNotExist"
	TooManyListeners     errors.Code = "TooManyListeners"
)

// AddConnection adds a new connection. oAuthToken is an OAuth token returned by
// the OAuthToken method and must be empty if the connector does not support
// OAuth authentication.
//
// It returns an errors.UnprocessableError error with code
//   - ConnectorNotExist, if the connector does not exist.
//   - InvalidSettings, if the settings are not valid.
func (this *Workspace) AddConnection(ctx context.Context, connection ConnectionToAdd, oAuthToken string) (int, error) {

	this.apis.mustBeOpen()

	if connection.Role != Source && connection.Role != Destination {
		return 0, errors.BadRequest("role %d is not valid", int(connection.Role))
	}
	if connection.Connector < 1 || connection.Connector > maxInt32 {
		return 0, errors.BadRequest("connector identifier %d is not valid", connection.Connector)
	}
	if utf8.RuneCountInString(connection.Name) > 100 {
		return 0, errors.BadRequest("name %q is not valid", connection.Name)
	}

	if s := connection.Strategy; s != nil {
		if !isValidStrategy(*s) {
			return 0, errors.BadRequest("strategy %q is not valid", *s)
		}
		if connection.Role == Destination {
			return 0, errors.BadRequest("destination connections cannot have a strategy")
		}
	}

	c, ok := this.apis.state.Connector(connection.Connector)
	if !ok {
		return 0, errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", connection.Connector)
	}
	if c.Type == state.FileType {
		return 0, errors.BadRequest("cannot add a connection with type File")
	}

	n := state.AddConnection{
		Workspace:   this.workspace.ID,
		Name:        connection.Name,
		Role:        state.Role(connection.Role),
		Enabled:     connection.Enabled,
		Connector:   connection.Connector,
		Strategy:    (*state.Strategy)(connection.Strategy),
		WebsiteHost: connection.WebsiteHost,
		BusinessID:  connection.BusinessID,
	}

	// Validate the strategy.
	if connection.Role == Source {
		switch c.Type {
		case state.MobileType, state.WebsiteType:
			if connection.Strategy == nil {
				return 0, errors.BadRequest("%s connections must have a strategy", strings.ToLower(c.Type.String()))
			}
		default:
			if connection.Strategy != nil {
				return 0, errors.BadRequest("%s connections cannot have a strategy", strings.ToLower(c.Type.String()))
			}
		}
	}

	// Validate the website host.
	if n.WebsiteHost != "" {
		if c.Type != state.WebsiteType {
			return 0, errors.BadRequest("connector %d cannot have a website host, it's a %s",
				c.ID, strings.ToLower(c.Type.String()))
		}
		if h, p, found := strings.Cut(n.WebsiteHost, ":"); h == "" || len(n.WebsiteHost) > 255 {
			return 0, errors.BadRequest("website host %q is not valid", n.WebsiteHost)
		} else if found {
			if port, _ := strconv.Atoi(p); port < 1 || port > 65535 {
				return 0, errors.BadRequest("website host %q is not valid", n.WebsiteHost)
			}
		}
	}

	// Validate the Business ID.
	err := validateBusinessID(c.Type, n.Role, n.BusinessID)
	if err != nil {
		return 0, err
	}

	// Validate OAuth.
	if (oAuthToken == "") != (c.OAuth == nil) {
		if oAuthToken == "" {
			return 0, errors.BadRequest("OAuth is required by connector %d", n.Connector)
		}
		return 0, errors.BadRequest("connector %d does not support OAuth", n.Connector)
	}

	// Set the resource. It can be an existing resource or a resource that needs to be created.
	if oAuthToken != "" {
		data, err := base62.DecodeString(oAuthToken)
		if err != nil {
			return 0, errors.BadRequest("OAuth is not valid")
		}
		var resource authorizedResource
		err = json.Unmarshal(data, &resource)
		if err != nil {
			return 0, errors.BadRequest("OAuth is not valid")
		}
		if resource.Workspace != this.workspace.ID || resource.Connector != c.ID {
			return 0, errors.BadRequest("OAuth is not valid")
		}
		n.Resource.Code = resource.Code
		r, ok := this.workspace.ResourceByCode(resource.Code)
		if ok {
			n.Resource.ID = r.ID
		}
		if !ok || resource.AccessToken != r.AccessToken || resource.RefreshToken != r.RefreshToken ||
			resource.ExpiresIn != r.ExpiresIn {
			n.Resource.AccessToken = resource.AccessToken
			n.Resource.RefreshToken = resource.RefreshToken
			n.Resource.ExpiresIn = resource.ExpiresIn
		}
	}

	// Validate the settings.
	if c.HasSettings {
		settings := connection.Settings
		if settings == nil {
			settings = json.RawMessage("{}")
		}
		var clientSecret string
		if c.OAuth != nil {
			clientSecret = c.OAuth.ClientSecret
		}
		conf := &connectors.ConnectorConfig{
			Role:         n.Role,
			Resource:     n.Resource.Code,
			ClientSecret: clientSecret,
			AccessToken:  n.Resource.AccessToken,
			Region:       state.PrivacyRegion(this.PrivacyRegion),
		}
		var err error
		n.Settings, err = this.apis.connectors.ValidateSettings(ctx, c, conf, settings)
		if err != nil {
			if err != connectors.ErrNoUserInterface {
				return 0, errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
			}
			if connection.Settings != nil {
				return 0, errors.BadRequest("settings cannot be provided because %s connector %s does not have a UI",
					strings.ToLower(connection.Role.String()), c.Name)
			}
		} else if connection.Settings == nil {
			return 0, errors.BadRequest("settings must be provided because %s connector %s has a UI",
				strings.ToLower(connection.Role.String()), c.Name)
		}
	}

	// Generate the identifier.
	n.ID, err = generateRandomID()
	if err != nil {
		return 0, err
	}

	// Generate a write key.
	switch c.Type {
	case state.MobileType, state.ServerType, state.WebsiteType:
		n.Key, err = generateWriteKey()
		if err != nil {
			return 0, err
		}
	}

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		if n.Resource.Code != "" {
			if n.Resource.ID == 0 {
				// Insert a new resource.
				err = tx.QueryRow(ctx, "INSERT INTO resources (workspace, connector, code, access_token,"+
					" refresh_token, expires_in) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
					n.Workspace, n.Connector, n.Resource.Code, n.Resource.AccessToken, n.Resource.RefreshToken, n.Resource.ExpiresIn).
					Scan(&n.Resource.ID)
			} else if n.Resource.AccessToken != "" {
				// Update the current resource.
				_, err = tx.Exec(ctx, "UPDATE resources "+
					"SET access_token = $1, refresh_token = $2, expires_in = $3 WHERE id = $4",
					n.Resource.AccessToken, n.Resource.RefreshToken, n.Resource.ExpiresIn, n.Resource.ID)
			}
			if err != nil {
				if postgres.IsForeignKeyViolation(err) {
					switch postgres.ErrConstraintName(err) {
					case "resources_workspace_fkey":
						err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
					case "resources_connector_fkey":
						err = errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", n.Connector)
					}
				}
				return err
			}
		}
		// Insert the connection.
		_, err = tx.Exec(ctx, "INSERT INTO connections "+
			"(id, workspace, name, type, role, enabled, connector,"+
			" resource, strategy, website_host, business_id, settings)"+
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)",
			n.ID, n.Workspace, n.Name, c.Type, n.Role, n.Enabled, n.Connector,
			n.Resource.ID, n.Strategy, n.WebsiteHost, n.BusinessID, string(n.Settings))
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				switch postgres.ErrConstraintName(err) {
				case "connections_workspace_fkey":
					err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
				case "connections_connector_fkey":
					err = errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", n.Connector)
				}
			}
			return err
		}
		if n.Key != "" {
			// Insert the server key.
			_, err = tx.Exec(ctx, "INSERT INTO connections_keys (connection, value, creation_time) VALUES ($1, $2, $3)",
				n.ID, n.Key, time.Now().UTC())
			if err != nil {
				return err
			}
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// AddEventListener adds an event listener to the workspace that listens to
// collected events and returns its identifier.
//
// size specifies the maximum number of observed events to be returned by a
// subsequent call to the ListenedEvents method, and must be in range [1, 1000].
//
// source represents the identifier of a source, whether it's a mobile, server,
// or website connection. If source is non-zero, only events originating from
// this source will be observed.
//
// onlyValid determines whether only valid events should be observed.
//
// It returns an errors.UnprocessableError error with code:
//   - ConnectionNotExist, if the source connection does not exist.
//   - TooManyListeners, if there are already too many listeners.
func (this *Workspace) AddEventListener(ctx context.Context, size, source int, onlyValid bool) (string, error) {

	this.apis.mustBeOpen()

	if size < 1 || size > maxEventsListenedTo {
		return "", errors.BadRequest("size %d is not valid", size)
	}
	if source < 0 || source > maxInt32 {
		return "", errors.BadRequest("source identifier %d is not valid", source)
	}

	if source > 0 {
		var typ state.ConnectorType
		var role state.Role
		err := this.apis.db.QueryRow(ctx, "SELECT type, role FROM connections\n"+
			"WHERE id = $1 AND workspace = $2", source, this.workspace.ID).Scan(&typ, &role)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", errors.Unprocessable(ConnectionNotExist, "connection %d does not exist", source)
			}
			return "", err
		}
		switch typ {
		case state.MobileType, state.ServerType, state.WebsiteType:
		default:
			return "", errors.BadRequest("connection %d is not a mobile, server or website", source)
		}
		if role != state.Source {
			return "", errors.BadRequest("connection %d is not a source", source)
		}
	}

	id, err := this.apis.events.Observer().AddListener(size, source, onlyValid)
	if err != nil {
		if err == events.ErrTooManyListeners {
			err = errors.Unprocessable(TooManyListeners, "there are already %d listeners", events.MaxEventListeners)
		}
		return "", err
	}

	return id, nil
}

// ChangeUsersSchema changes the "users" schema to schema.
//
// rePaths is a mapping containing the renamed property paths, where the key is
// the new property path and its value is the old property path. In case of new
// properties created with the same name of already existent properties, the
// value must be the untyped nil. rePaths cannot contain keys with the same path
// as their value. Any property path which does not refer to changed properties
// is ignored.
//
// It returns an errors.UnprocessableError error with code:
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - InvalidSchemaChange, if the schema change is invalid.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) ChangeUsersSchema(ctx context.Context, schema types.Type, rePaths map[string]any) error {
	this.apis.mustBeOpen()
	if !schema.Valid() {
		return errors.BadRequest("schema must be valid")
	}
	if schema.Kind() != types.ObjectKind {
		return errors.BadRequest("expected schema with kind Object, got %s", schema.Kind())
	}
	if this.store == nil {
		return errors.Unprocessable(NoWarehouse, "workspace %d does not have a data store", this.workspace.ID)
	}
	if err := validateRePaths(rePaths); err != nil {
		return errors.BadRequest("invalid rePaths: %s", err)
	}
	current := removeMetaProperties(this.workspace.UsersSchema)
	schema = removeMetaProperties(schema)
	operations, err := diffschemas.Diff(current, schema, rePaths, "")
	if err != nil {
		return errors.Unprocessable(InvalidSchemaChange, "cannot change the schema as specified: %s", err)
	}
	if len(operations) == 0 {
		return nil
	}

	// Add the "Id" meta property.
	// TODO(Gianluca): see https://github.com/open2b/chichi/issues/573.
	schema = types.Object(append([]types.Property{
		{Name: "Id", Type: types.Int(32)},
	}, schema.Properties()...))

	// Update the database and send the notification.
	n := state.SetWorkspaceUsersSchema{
		Workspace:   this.ID,
		UsersSchema: schema,
	}
	usersSchemaJSON, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE workspaces SET users_schema = $1 WHERE id = $2",
			usersSchemaJSON, n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return err
	}

	// Alter the schema on the data warehouse.
	err = this.store.AlterSchema(ctx, operations)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return errors.Unprocessable(DataWarehouseFailed, "data warehouse has returned an error: %w", err.Err)
		}
		if err, ok := err.(datastore.UnsupportedAlterSchemaErr); ok {
			return errors.Unprocessable(InvalidSchemaChange, "cannot apply the schema change: %s", err)
		}
		return err
	}

	return nil
}

// ChangeUsersSchemaQueries returns the queries that would be executed changing
// the "users" schema to schema.
//
// rePaths is a mapping containing the renamed property paths, where the key is
// the new property path and its value is the old property path. In case of new
// properties created with the same name of already existent properties, the
// value must be the untyped nil. rePaths cannot contain keys with the same path
// as their value.
//
// It returns an errors.UnprocessableError error with code:
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - InvalidSchemaChange, if the schema change is invalid.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) ChangeUsersSchemaQueries(ctx context.Context, schema types.Type, rePaths map[string]any) ([]string, error) {
	this.apis.mustBeOpen()
	if !schema.Valid() {
		return nil, errors.BadRequest("schema must be valid")
	}
	if schema.Kind() != types.ObjectKind {
		return nil, errors.BadRequest("expected schema with kind Object, got %s", schema.Kind())
	}
	if err := validateRePaths(rePaths); err != nil {
		return nil, errors.BadRequest("invalid rePaths: %s", err)
	}
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data store", this.workspace.ID)
	}
	users := this.workspace.UsersSchema
	users = removeMetaProperties(users)
	schema = removeMetaProperties(schema)
	operations, err := diffschemas.Diff(users, schema, rePaths, "")
	if err != nil {
		return nil, errors.Unprocessable(InvalidSchemaChange, "cannot change the schema as specified: %s", err)
	}
	if len(operations) == 0 {
		return []string{}, nil
	}
	queries, err := this.store.AlterSchemaQueries(ctx, operations)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return nil, errors.Unprocessable(DataWarehouseFailed, "data warehouse has returned an error: %w", err.Err)
		}
		if err, ok := err.(datastore.UnsupportedAlterSchemaErr); ok {
			return nil, errors.Unprocessable(InvalidSchemaChange, "cannot get the queries for the schema change: %s", err)
		}
		return nil, err
	}
	return queries, nil
}

// ChangeWarehouseSettings changes the settings of the data warehouse for the
// workspace.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore, and it returns an errors.UnprocessableError error with code
//
//   - NotConnected, if the workspace is not connected to a data warehouse.
//   - InvalidWarehouseType, if the workspace is connected to a data warehouse
//     of a different type,
//   - InvalidSettings, if the settings are not valid.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) ChangeWarehouseSettings(ctx context.Context, typ WarehouseType, settings []byte) error {
	this.apis.mustBeOpen()

	ws := this.workspace
	if this.store == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.ID)
	}
	if ws.Warehouse.Type != state.WarehouseType(typ) {
		return errors.Unprocessable(InvalidWarehouseType, "workspace %d is connected with a %s data warehouse, not %s", ws.ID, ws.Warehouse.Type, typ)
	}

	settings, err := this.organization.apis.datastore.NormalizeWarehouseSettings(ws.Warehouse.Type, settings)
	if err != nil {
		if err, ok := err.(*datastore.SettingsError); ok {
			return errors.Unprocessable(InvalidSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return err
	}

	err = this.organization.apis.datastore.PingWarehouse(ctx, ws.Warehouse.Type, settings)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return errors.Unprocessable(DataWarehouseFailed, "cannot connect to the data warehouse: %w", err.Err)
		}
		return err
	}

	n := state.SetWarehouse{
		Workspace: ws.ID,
		Warehouse: &state.Warehouse{
			Type:     ws.Warehouse.Type,
			Settings: settings,
		},
	}

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_settings = $1 WHERE id = $2 AND warehouse_type = $3",
			string(n.Warehouse.Settings), n.Workspace, n.Warehouse.Type)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			var warehouseType *state.WarehouseType
			err = tx.QueryRow(ctx, "SELECT warehouse_type FROM workspaces WHERE id = $1", n.Workspace).Scan(&warehouseType)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return err
			}
			return errors.Unprocessable(InvalidWarehouseType, "workspace %d is connected with a %s data warehouse, not %s",
				ws.ID, *warehouseType, n.Warehouse.Type)
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// Connection returns the connection with identifier id of the workspace.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// It returns an errors.UnprocessableError error with code
//
//   - FetchSchemaFailed, if an error occurred fetching the schema.
//   - NoWarehouse, if the workspace does not have a data warehouse.
func (this *Workspace) Connection(ctx context.Context, id int) (*Connection, error) {
	this.apis.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("connection identifier %d is not valid", id)
	}
	c, ok := this.workspace.Connection(id)
	if !ok {
		return nil, errors.NotFound("connection %d does not exist", id)
	}
	conn := c.Connector()
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", this.workspace.ID)
	}

	connection := Connection{
		apis:         this.apis,
		store:        this.store,
		connection:   c,
		ID:           c.ID,
		Name:         c.Name,
		Type:         ConnectorType(conn.Type),
		Role:         Role(c.Role),
		Enabled:      c.Enabled,
		Connector:    conn.ID,
		Strategy:     (*Strategy)(c.Strategy),
		WebsiteHost:  c.WebsiteHost,
		BusinessID:   c.BusinessID,
		HasSettings:  conn.HasSettings,
		ActionsCount: len(c.Actions()),
		Health:       Health(c.Health),
	}

	// Set the action types.
	ts, err := connection.actionTypes(ctx)
	if err != nil {
		return nil, err
	}
	connection.ActionTypes = &ts
	// Set the actions.
	actions := c.Actions()
	a := make([]Action, len(actions))
	connection.Actions = &a
	for i, a := range actions {
		(*connection.Actions)[i].fromState(this.apis, this.store, a)
	}
	return &connection, nil
}

// Connections returns the connections of the workspace.
func (this *Workspace) Connections() []*Connection {
	this.apis.mustBeOpen()
	connections := this.workspace.Connections()
	infos := make([]*Connection, len(connections))
	for i, c := range connections {
		conn := c.Connector()
		connection := Connection{
			apis:         this.apis,
			store:        this.store,
			connection:   c,
			ID:           c.ID,
			Name:         c.Name,
			Type:         ConnectorType(conn.Type),
			Role:         Role(c.Role),
			Enabled:      c.Enabled,
			Connector:    conn.ID,
			Strategy:     (*Strategy)(c.Strategy),
			WebsiteHost:  c.WebsiteHost,
			BusinessID:   c.BusinessID,
			HasSettings:  conn.HasSettings,
			ActionsCount: len(c.Actions()),
			Health:       Health(c.Health),
		}
		infos[i] = &connection
	}
	sort.Slice(infos, func(i, j int) bool {
		a, b := infos[i], infos[j]
		return a.Name < b.Name || a.Name == b.Name && a.ID == b.ID
	})
	return infos
}

// ConnectWarehouse connects the workspace to a data warehouse, with the given
// settings. It also creates the tables in the connected data warehouse.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore, and it returns an errors.UnprocessableError error with code
//   - AlreadyConnected, if the workspace is already connected to a data
//     warehouse.
//   - InvalidSettings, if the settings are not valid.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) ConnectWarehouse(ctx context.Context, typ WarehouseType, settings []byte) error {
	this.apis.mustBeOpen()

	ws := this.workspace
	if ws.Warehouse != nil {
		return errors.Unprocessable(AlreadyConnected, "workspace %d is already connected to a data warehouse", ws.ID)
	}

	settings, err := this.organization.apis.datastore.NormalizeWarehouseSettings(state.WarehouseType(typ), settings)
	if err != nil {
		if err, ok := err.(*datastore.SettingsError); ok {
			return errors.Unprocessable(InvalidSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return err
	}

	n := state.SetWarehouse{
		Workspace: ws.ID,
		Warehouse: &state.Warehouse{
			Type:     state.WarehouseType(typ),
			Settings: settings,
		},
	}

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_type = $1, warehouse_settings = $2"+
			"  WHERE id = $3 AND warehouse_type IS NULL",
			n.Warehouse.Type, string(n.Warehouse.Settings), n.Workspace)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			err = tx.QueryVoid(ctx, "SELECT FROM workspaces WHERE id = $1", n.Workspace)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return err
			}
			return errors.Unprocessable(AlreadyConnected, "workspace %d is already connected to a data warehouse", ws.ID)
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// Delete deletes the workspace with all its connections.
//
// If the workspace does not exist anymore, it returns an errors.NotFound error.
// If the workspace is currently connected to a data warehouse, it returns an
// errors.UnprocessableError error with code CurrentlyConnected.
func (this *Workspace) Delete(ctx context.Context) error {
	this.apis.mustBeOpen()
	ws := this.workspace
	if ws.Warehouse != nil {
		return errors.Unprocessable(CurrentlyConnected, "workspace %d is currently connected to %s data warehouse", ws.ID, ws.Warehouse.Type)
	}
	n := state.DeleteWorkspace{
		ID: this.workspace.ID,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "DELETE FROM workspaces WHERE id = $1 AND warehouse_type IS NULL", n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			var warehouseType state.WarehouseType
			err := tx.QueryRow(ctx, "SELECT warehouse_type FROM workspaces WHERE id = $1", n.ID).Scan(&warehouseType)
			if err != nil {
				if err == sql.ErrNoRows {
					return errors.NotFound("workspace %d does not exist", n.ID)
				}
				return err
			}
			return errors.Unprocessable(CurrentlyConnected, "workspace %d is currently connected to %s data warehouse", ws.ID, warehouseType)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// DisconnectWarehouse disconnects the workspace from the data warehouse.
//
// If the workspace does not exist anymore, it returns an errors.NotFoundError
// error. If the workspace is not connected to a data warehouse, it returns an
// errors.UnprocessableError error with code NotConnected.
func (this *Workspace) DisconnectWarehouse(ctx context.Context) error {
	this.apis.mustBeOpen()
	ws := this.workspace
	if ws.Warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.ID)
	}
	n := state.SetWarehouse{
		Workspace: ws.ID,
		Warehouse: nil,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		var typ *state.WarehouseType
		err := tx.QueryRow(ctx, "SELECT warehouse_type FROM workspaces WHERE id = $1", n.Workspace).Scan(&typ)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.NotFound("workspace %d does not exist", n.Workspace)
			}
			return err
		}
		if typ == nil {
			return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", n.Workspace)
		}
		_, err = tx.Exec(ctx, "UPDATE workspaces SET warehouse_type = NULL, warehouse_settings = '' WHERE id = $1", n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// IdentifiersSchema returns the properties of the "users" schema that can be
// used as identifiers in the Workspace Identity Resolution.
// If none of the properties can be an identifier, this method returns the
// invalid schema.
func (this *Workspace) IdentifiersSchema(ctx context.Context) (types.Type, error) {
	this.apis.mustBeOpen()
	var properties []types.Property
	for _, p := range this.workspace.UsersSchema.Properties() {
		if isMetaProperty(p.Name) {
			continue
		}
		if datastore.CanBeIdentifier(p.Type) {
			properties = append(properties, p)
		}
	}
	if len(properties) == 0 {
		return types.Type{}, nil
	}
	return types.Object(properties), nil
}

// InitWarehouse initializes the data warehouse of the workspace by creating the
// supporting tables.
//
// It returns an errors.UnprocessableError error with code:
//
//   - NotConnected, if the workspace is not connected to a data warehouse
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) InitWarehouse(ctx context.Context) error {
	this.apis.mustBeOpen()
	if this.store == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a warehouse", this.workspace.ID)
	}
	err := this.store.InitWarehouse(ctx)
	if err, ok := err.(*datastore.DataWarehouseError); ok {
		return errors.Unprocessable(DataWarehouseFailed, "data warehouse failed: %s", err.Err)
	}
	return err
}

// RunIdentityResolution runs the Workspace Identity Resolution on the
// workspace.
//
// It returns an errors.UnprocessableError error with code:
//
//   - NotConnected, if the workspace is not connected to a data warehouse
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) RunIdentityResolution(ctx context.Context) error {
	this.apis.mustBeOpen()
	if this.store == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a warehouse", this.workspace.ID)
	}
	slog.Info("running Workspace Identity Resolution", "workspace", this.workspace.ID)
	err := this.store.RunWorkspaceIdentityResolution(ctx)
	if err != nil {
		return err
	}
	slog.Info("execution of Workspace Identity Resolution is completed", "workspace", this.workspace.ID)
	return nil
}

// ObservedEvent represents an observed event.
type ObservedEvent struct {

	// Source, if not zero, it is the source mobile, server or website
	// connection for which the event was sent.
	Source int

	// Header is the message header. It is nil if a validation error occurred
	// processing the entire message.
	Header *ObservedEventHeader

	// Data contains the data, encoded in JSON, of a single event in the message,
	// if Header is not nil, or the data of the entire message, if Header is nil.
	Data []byte

	// Err, if not empty, is a validation error occurred processing the message.
	// It refers to a single event, if Header is not nil, or to the entire message
	// if Header is nil.
	Err string
}

type ObservedEventHeader struct {
	ReceivedAt time.Time   `json:"receivedAt"`
	RemoteAddr string      `json:"remoteAddr"`
	Method     string      `json:"method"`
	Proto      string      `json:"proto"`
	URL        string      `json:"url"`
	Headers    http.Header `json:"headers"`
}

// ListenedEvents returns the events listen to by the specified listener and
// the number of discarded events.
//
// It returns an errors.NotFoundError error, if the listener does not exist.
func (this *Workspace) ListenedEvents(listener string) ([]ObservedEvent, int, error) {
	this.apis.mustBeOpen()
	observedEvents, discarded, err := this.apis.events.Observer().Events(listener)
	if err != nil {
		if err == events.ErrEventListenerNotFound {
			return nil, 0, errors.NotFound("event listener %q does not exist", listener)
		}
		return nil, 0, err
	}
	evs := make([]ObservedEvent, len(observedEvents))
	for i := range len(evs) {
		ov := observedEvents[i]
		var header *ObservedEventHeader
		if ov.Header != nil {
			header = &ObservedEventHeader{
				ReceivedAt: ov.Header.ReceivedAt,
				RemoteAddr: ov.Header.RemoteAddr,
				Method:     ov.Header.Method,
				Proto:      ov.Header.Proto,
				URL:        ov.Header.URL,
				Headers:    maps.Clone(ov.Header.Headers),
			}
		}
		evs[i] = ObservedEvent{
			Source: ov.Source,
			Header: header,
			Data:   slices.Clone(ov.Data),
			Err:    ov.Err,
		}
	}
	return evs, discarded, nil
}

// authorizedResource represents an authorized resource that can be used to
// create a new connection.
type authorizedResource struct {
	Workspace    int
	Connector    int
	Code         string
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Time
}

// OAuthToken returns an OAuth token, given an OAuth authorization code and the
// redirection URI used to obtain that code, that can be used to add a new
// connection to the workspace for the specified connector.
//
// It returns an errors.NotFound error if the workspace does not exist anymore.
// It returns an errors.UnprocessableError error with code ConnectorNotExist if
// the connector does not exist.
func (this *Workspace) OAuthToken(ctx context.Context, code, redirectionURI string, connector int) (string, error) {

	this.apis.mustBeOpen()

	if code == "" {
		return "", errors.BadRequest("authorization code is empty")
	}
	if connector < 1 || connector > maxInt32 {
		return "", errors.BadRequest("connector identifier %d is not valid", connector)
	}

	c, ok := this.apis.state.Connector(connector)
	if !ok {
		return "", errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", connector)
	}
	if c.OAuth == nil {
		return "", errors.BadRequest("connector %d does not support OAuth", connector)
	}

	region := state.PrivacyRegion(this.PrivacyRegion)
	auth, err := this.apis.connectors.GrantAuthorization(ctx, c, code, redirectionURI, region)
	if err != nil {
		return "", err
	}

	resource, err := json.Marshal(authorizedResource{
		Workspace:    this.workspace.ID,
		Connector:    connector,
		Code:         auth.ResourceCode,
		AccessToken:  auth.AccessToken,
		RefreshToken: auth.RefreshToken,
		ExpiresIn:    auth.ExpiresIn,
	})
	if err != nil {
		return "", err
	}

	// TODO(marco): Encrypt the token.

	return base62.EncodeToString(resource), nil
}

// RemoveEventListener removes the given event listener from the workspace. It
// does nothing if the listener does not exist.
func (this *Workspace) RemoveEventListener(listener string) {
	this.apis.mustBeOpen()
	this.apis.events.Observer().RemoveListener(listener)
}

// Rename renames the workspace with the given new name.
// name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the workspace does not exist
// anymore.
func (this *Workspace) Rename(ctx context.Context, name string) error {
	this.apis.mustBeOpen()
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return errors.BadRequest("name %q is not valid", name)
	}
	if name == this.workspace.Name {
		return nil
	}
	n := state.RenameWorkspace{
		Workspace: this.workspace.ID,
		Name:      name,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET name = $1 WHERE id = $2", n.Name, n.Workspace)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("workspace %d does not exist", n.Workspace)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// ServeUI serves the user interface for the given connector, with the given
// role. event is the event and values contains the form values in JSON format.
// oAuth is the OAuth token returned by the (*Workspace).OAuth method, it is
// required if the connector requires OAuth.
//
// It returns an errors.UnprocessableError error with code:
// - ConnectorNotExist, if the connector does not exist.
// - EventNotExist, if the event does not exist.
func (this *Workspace) ServeUI(ctx context.Context, event string, values []byte, connector int, role Role, oAuth string) ([]byte, error) {

	this.apis.mustBeOpen()

	if connector < 1 || connector > maxInt32 {
		return nil, errors.BadRequest("connector identifier %d is not valid", connector)
	}
	c, ok := this.apis.state.Connector(connector)
	if !ok {
		return nil, errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", connector)
	}

	if (oAuth == "") != (c.OAuth == nil) {
		if oAuth == "" {
			return nil, errors.BadRequest("OAuth is required by connector %d", c.ID)
		}
		return nil, errors.BadRequest("connector %d does not support OAuth", c.ID)
	}

	// Decode oAuth.
	var r authorizedResource
	if oAuth != "" {
		data, err := base62.DecodeString(oAuth)
		if err != nil {
			return nil, errors.BadRequest("oAuth is not valid")
		}
		err = json.Unmarshal(data, &r)
		if err != nil {
			return nil, errors.BadRequest("oAuth is not valid")
		}
	}

	var clientSecret string
	if oAuth != "" {
		clientSecret = c.OAuth.ClientSecret
	}
	conf := &connectors.ConnectorConfig{
		Role:         state.Role(role),
		Resource:     r.Code,
		ClientSecret: clientSecret,
		AccessToken:  r.AccessToken,
		Region:       this.workspace.PrivacyRegion,
	}
	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	b, err := this.apis.connectors.ServeConnectorUI(ctx, c, conf, event, values)
	if err != nil {
		switch err {
		case connectors.ErrNoUserInterface:
			err = errors.BadRequest("connector %d does not have a UI", c.ID)
		case connectors.ErrEventNotExist:
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s connector", event, c.Name)
		}
		return nil, err
	}

	return b, nil
}

// SetIdentifiers sets the identifiers of the workspace.
func (this *Workspace) SetIdentifiers(ctx context.Context, identifiers []string) error {

	this.apis.mustBeOpen()

	// Validate the identifiers.
	for i, id := range identifiers {
		if !types.IsValidPropertyPath(id) {
			return errors.BadRequest("identifier %q is not a valid property path", id)
		}
		name := strings.Split(id, ".")[0]
		if isMetaProperty(name) {
			return errors.BadRequest("meta properties cannot be used as identifiers")
		}
		if slices.Contains(identifiers[i+1:], id) {
			return errors.BadRequest("identifier %s is repeated", id)
		}
	}

	// Update the database and send the notification.
	if identifiers == nil {
		identifiers = []string{}
	}
	ws := this.workspace
	n := state.SetWorkspaceIdentifiers{
		Workspace:   ws.ID,
		Identifiers: identifiers,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE workspaces SET identifiers = $1 WHERE id = $2",
			n.Identifiers, n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// Set sets the name, the privacy region and the displayed properties of the
// workspace. name must be between 1 and 100 runes long. displayedProperties
// must contain valid displayed property names. A valid displayed property name
// is an empty string, or alternatively a valid property name between 1 and 100
// runes long.
func (this *Workspace) Set(ctx context.Context, name string, region PrivacyRegion, displayedProperties DisplayedProperties) error {
	this.apis.mustBeOpen()
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return errors.BadRequest("name %q is not valid", name)
	}
	switch region {
	case PrivacyRegionNotSpecified,
		PrivacyRegionEurope:
	default:
		return errors.BadRequest("invalid privacy region %q", string(region))
	}
	if !isValidDisplayedPropertyName(displayedProperties.Image) {
		return errors.BadRequest("invalid displayed image %q", displayedProperties.Image)
	}
	if !isValidDisplayedPropertyName(displayedProperties.FirstName) {
		return errors.BadRequest("invalid displayed first name %q", displayedProperties.FirstName)
	}
	if !isValidDisplayedPropertyName(displayedProperties.LastName) {
		return errors.BadRequest("invalid displayed last name %q", displayedProperties.LastName)
	}
	if !isValidDisplayedPropertyName(displayedProperties.Information) {
		return errors.BadRequest("invalid displayed information %q", displayedProperties.Information)
	}
	ws := this.workspace
	n := state.SetWorkspace{
		Workspace:           ws.ID,
		Name:                name,
		PrivacyRegion:       state.PrivacyRegion(region),
		DisplayedProperties: state.DisplayedProperties(displayedProperties),
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE workspaces SET name = $1, privacy_region = $2, displayed_image = $3, "+
			"displayed_first_name = $4, displayed_last_name = $5, displayed_information = $6 "+
			"WHERE id = $7",
			n.Name, n.PrivacyRegion, n.DisplayedProperties.Image, n.DisplayedProperties.FirstName,
			n.DisplayedProperties.LastName, n.DisplayedProperties.Information, n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// SetWarehouseSettings sets the settings of the workspace's data store.
//
// It returns an errors.NotFoundError error, if the workspace does not exist,
// and it returns an errors.UnprocessableError error with code
//   - NotConnected, if the workspace is not connected to a data store.
//   - InvalidSettings, if the settings are not valid.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) SetWarehouseSettings(ctx context.Context, typ WarehouseType, settings []byte) error {
	this.apis.mustBeOpen()
	ws := this.workspace
	if ws.Warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data store", ws.ID)
	}
	if state.WarehouseType(typ) != ws.Warehouse.Type {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", fmt.Errorf(
			"workspace %d is connected to a %s data store, but settings are for a %s data store",
			ws.ID, ws.Warehouse.Type, typ))
	}
	warehouse, err := openWarehouse(typ, settings)
	if err != nil {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
	}
	err = warehouse.Ping(ctx)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return errors.Unprocessable(DataWarehouseFailed, "cannot connect to the data warehouse: %w", err)
		}
		return err
	}
	n := state.SetWarehouse{
		Workspace: ws.ID,
		Warehouse: &state.Warehouse{
			Type:     state.WarehouseType(typ),
			Settings: warehouse.Settings(),
		},
	}
	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_settings = $1 WHERE id = $2 AND warehouse_type = $3",
			string(n.Warehouse.Settings), n.Workspace, n.Warehouse.Type)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			err = tx.QueryVoid(ctx, "SELECT FROM workspaces WHERE id = $1", n.Workspace)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return err
			}
			return errors.Unprocessable(NoWarehouse, "workspace %d is not connected to a PostgreSQL data store", ws.ID)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// PingWarehouse pings the data warehouse with the given settings, verifying
// that the settings are valid and a connection can be established.
//
// It returns an errors.UnprocessableError error with code
//   - InvalidSettings, if the settings are not valid.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) PingWarehouse(ctx context.Context, typ WarehouseType, settings []byte) error {
	this.apis.mustBeOpen()
	err := this.organization.apis.datastore.PingWarehouse(ctx, state.WarehouseType(typ), settings)
	switch err := err.(type) {
	case *datastore.SettingsError:
		return errors.Unprocessable(InvalidSettings, "data warehouse settings are not valid: %w", err.Err)
	case *datastore.DataWarehouseError:
		return errors.Unprocessable(DataWarehouseFailed, "cannot connect to the data warehouse: %w", err.Err)
	}
	return err
}

// User returns the user with identifier id of the workspace. If the user does
// not exist, the error is deferred until methods of *User are called.
func (this *Workspace) User(id int) (*User, error) {
	this.apis.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("user identifier %d is not valid", id)
	}
	return &User{
		apis:      this.apis,
		workspace: this.workspace,
		store:     this.store,
		id:        id,
	}, nil
}

// Users returns the users, the user schema of the workspace, and an estimate of
// their count without applying first and limit. It returns the users that
// satisfies the filter, if not nil, and in range [first,first+limit] with first
// >= 0 and 0 < limit <= 1000 and only the given properties. properties cannot
// be empty.
//
// order is the property by which to sort the returned users and cannot have
// type JSON, Array, Object, or Map; it defaults to the "Id" property.
//
// orderDesc control whether the returned users should be ordered in descending
// order instead of ascending, which is the default.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore.
// It returns an errors.UnprocessableError error with code
//
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - NoWarehouse, if the workspace does not have a data store.
//   - OrderNotExist, if order does not exist in schema.
//   - OrderTypeNotSortable, if the type of the order property is not sortable.
//   - PropertyNotExist, if a property does not exist.
func (this *Workspace) Users(ctx context.Context, properties []string, filter *Filter, order string, orderDesc bool, first, limit int) ([]byte, types.Type, int, error) {

	this.apis.mustBeOpen()

	ws := this.workspace

	// Verify that the workspace has a data store.
	if this.store == nil {
		return nil, types.Type{}, 0, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data store", ws.ID)
	}

	// Validate the arguments.
	if len(properties) == 0 {
		return nil, types.Type{}, 0, errors.BadRequest("properties is empty")
	}
	propertyByName := map[string]types.Property{}
	for _, p := range ws.UsersSchema.Properties() {
		propertyByName[p.Name] = p
	}
	for _, name := range properties {
		if _, ok := propertyByName[name]; !ok {
			if name == "" {
				return nil, types.Type{}, 0, errors.BadRequest("a property name is empty")
			}
			if !types.IsValidPropertyName(name) {
				return nil, types.Type{}, 0, errors.BadRequest("property name %q is not valid", name)
			}
			return nil, types.Type{}, 0, errors.Unprocessable(PropertyNotExist, "property name %s does not exist", name)
		}
	}
	var where expr.Expr
	if filter != nil {
		_, err := validateFilter(filter, ws.UsersSchema)
		if err != nil {
			if err, ok := err.(types.PathNotExistError); ok {
				return nil, types.Type{}, 0, errors.Unprocessable(PropertyNotExist, "filter's property %s does not exist", err.Path)
			}
			return nil, types.Type{}, 0, errors.BadRequest("filter is not valid: %w", err)
		}
		where, _ = convertFilterToExpr(filter, ws.UsersSchema)
	}
	var orderProperty types.Property
	if order != "" {
		if !types.IsValidPropertyName(order) {
			return nil, types.Type{}, 0, errors.BadRequest("order %q is not a valid property name", order)
		}
		orderProperty, ok := propertyByName[order]
		if !ok {
			return nil, types.Type{}, 0, errors.Unprocessable(OrderNotExist, "order %s does not exist in schema", order)
		}
		switch orderProperty.Type.Kind() {
		case types.JSONKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return nil, types.Type{}, 0, errors.Unprocessable(OrderTypeNotSortable,
				"cannot sort by %s: property has type %s", order, orderProperty.Type)
		}
	} else {
		orderProperty = types.Property{Name: "Id", Type: types.Int(32)}
	}
	if first < 0 || first > maxInt32 {
		return nil, types.Type{}, 0, errors.BadRequest("first %d in not valid", first)
	}
	if limit < 1 || limit > 1000 {
		return nil, types.Type{}, 0, errors.BadRequest("limit %d is not valid", limit)
	}

	// Read the users.
	propsPaths := []types.Path{}
	for _, p := range properties {
		propsPaths = append(propsPaths, types.Path{p})
	}
	records, count, err := this.store.Users(ctx, datastore.UsersQuery{
		Schema:     ws.UsersSchema,
		Properties: propsPaths,
		Where:      where,
		OrderBy:    orderProperty,
		OrderDesc:  orderDesc,
		First:      first,
		Limit:      limit,
	})
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			slog.Error("cannot get users from the data store", "workspace", ws.ID, "err", err)
			return nil, types.Type{}, 0, errors.Unprocessable(DataWarehouseFailed, "store connection is failed: %w", err.Err)
		}
		return nil, types.Type{}, 0, err
	}
	users := []map[string]any{}
	err = records.For(func(user warehouses.Record) error {
		if user.Err != nil {
			return err
		}
		users = append(users, user.Properties)
		return nil
	})
	if err != nil {
		return nil, types.Type{}, 0, err
	}
	if err = records.Err(); err != nil {
		return nil, types.Type{}, 0, err
	}

	// Since the count is an estimate, being counted separately from the actual
	// number of users returned, ensure to not return a value lower than the
	// actually returned number of users.
	count = max(len(users), count)

	// Create the schema to return, with only the requested properties.
	requestedProperties := make([]types.Property, len(properties))
	for i, name := range properties {
		requestedProperties[i] = propertyByName[name]
	}
	schema := types.Object(requestedProperties)
	marshaledUsers, err := encoding.MarshalSlice(schema, users)
	if err != nil {
		return nil, types.Type{}, 0, err
	}

	return marshaledUsers, schema, count, nil
}

// WarehouseSettings returns the type and settings of the data warehouse for the
// workspace.
//
// If the workspace is not connected to a data warehouse, it returns an
// errors.UnprocessableError error with code NotConnected.
func (this *Workspace) WarehouseSettings() (WarehouseType, []byte, error) {
	this.apis.mustBeOpen()
	ws := this.workspace
	if ws.Warehouse == nil {
		return 0, nil, errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.ID)
	}
	return WarehouseType(ws.Warehouse.Type), slices.Clone(ws.Warehouse.Settings), nil
}

// ConnectionToAdd represents a connection to add to a workspace.
type ConnectionToAdd struct {

	// Name is the name of the connection. It cannot be longer than 100 runes.
	// If empty, the connection name will be the name of its connector.
	Name string

	// Role is the role.
	Role Role

	// Enable reports whether the connection is enabled or disabled when added.
	Enabled bool

	// Connector is the identifier of the connector.
	Connector int

	// Strategy is the strategy that determines how to merge anonymous and
	// non-anonymous users. It must be nil for destination connections and
	// non-event source connections.
	Strategy *Strategy

	// WebsiteHost is the host, in the form "host:port", of a website
	// connection. It must be empty if the connection is not a website. It
	// cannot be longer than 261 runes.
	WebsiteHost string

	// BusinessID is the Business ID property or column (depending on the type of the
	// connection) for source connections that import users. May be the empty string to
	// indicate to not import the Business ID.
	BusinessID string

	// Settings represents the settings. It must be nil if the connection does
	// not have settings.
	Settings json.RawMessage
}

// openWarehouse opens a data store with the given type and settings.
// It returns an error if typ or settings are not valid.
func openWarehouse(typ WarehouseType, settings []byte) (warehouses.Warehouse, error) {
	switch typ {
	case BigQuery, Redshift:
		return nil, fmt.Errorf("store type %s is not yet supported", typ)
	case ClickHouse:
		return clickhouse.Open(settings)
	case PostgreSQL:
		return postgresql.Open(settings)
	case Snowflake:
		return snowflake.Open(settings)
	}
	return nil, fmt.Errorf("store type %d is not valid", typ)
}

// WarehouseType represents a data store type.
type WarehouseType int

const (
	BigQuery WarehouseType = iota + 1
	ClickHouse
	PostgreSQL
	Redshift
	Snowflake
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if typ is not a valid WarehouseType value.
func (typ WarehouseType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + typ.String() + `"`), nil
}

// String returns the string representation of typ.
// It panics if typ is not a valid WarehouseType value.
func (typ WarehouseType) String() string {
	switch typ {
	case BigQuery:
		return "BigQuery"
	case ClickHouse:
		return "ClickHouse"
	case PostgreSQL:
		return "PostgreSQL"
	case Redshift:
		return "Redshift"
	case Snowflake:
		return "Snowflake"
	}
	panic("invalid store type")
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (typ *WarehouseType) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a WarehouseType value: %s", err)
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an WarehouseType value", v)
	}
	var t WarehouseType
	switch s {
	case "BigQuery":
		t = BigQuery
	case "ClickHouse":
		t = ClickHouse
	case "PostgreSQL":
		t = PostgreSQL
	case "Redshift":
		t = Redshift
	case "Snowflake":
		t = Snowflake
	default:
		return fmt.Errorf("invalid WarehouseType: %s", s)
	}
	*typ = t
	return nil
}

// isValidDisplayedPropertyName reports whether property is a valid displayed
// property name. A valid displayed property name is an empty string, or
// alternatively a valid property name between 1 and 100 runes long.
func isValidDisplayedPropertyName(property string) bool {
	if property != "" && (utf8.RuneCountInString(property) > 100 || !types.IsValidPropertyName(property)) {
		return false
	}
	return true
}

type labelValue struct {
	Label string
	Value string
}
type identity struct {
	Connection   int
	ExternalId   labelValue // zero struct for identities imported from anonymous events.
	BusinessId   string     // empty string for identities with no Business ID.
	AnonymousIds []string   // nil for identities not imported from events.
	UpdatedAt    time.Time
}

// userIdentities returns the users identities matching the "where" expression,
// and an estimate of their count without applying first and limit.
//
// It returns the user identities in range [first,first+limit] with first >= 0
// and 0 < limit <= 1000.
//
// If there are no identities, a nil slice is returned.
//
// It returns an errors.UnprocessableError error with code DataWarehouseFailed,
// if an error occurred with the data warehouse.
func (this *Workspace) userIdentities(ctx context.Context, where expr.Expr, first, limit int) ([]identity, int, error) {

	// Retrieve the identities from the data warehouse.
	schema := types.Object([]types.Property{
		{Name: "Connection", Type: types.Int(32)},
		{Name: "ExternalId", Type: types.Text()},
		{Name: "UpdatedAt", Type: types.DateTime()},
		{Name: "Gid", Type: types.Int(32)},
		{Name: "AnonymousIds", Type: types.Array(types.Text()), Nullable: true},
		{Name: "BusinessId", Type: types.Text().WithCharLen(40)},
	})
	records, count, err := this.store.UserIdentities(ctx, datastore.UsersIdentitiesQuery{
		Properties: []types.Path{{"Connection"}, {"ExternalId"}, {"AnonymousIds"},
			{"UpdatedAt"}, {"BusinessId"}},
		Where:   where,
		OrderBy: types.Property{Name: "IdentityId", Type: types.Int(32)},
		Schema:  schema,
		First:   first,
		Limit:   limit,
	})
	if err != nil {
		return nil, 0, err
	}

	// Create the identities from the records returned by the warehouse.
	var identities []identity
	err = records.For(func(record warehouses.Record) error {
		if record.Err != nil {
			return err
		}

		// Retrieve the connection.
		connID := record.Properties["Connection"].(int)
		conn, ok := this.apis.state.Connection(connID)
		if !ok {
			// The connection for this user identity no longer exists, so skip
			// this identity.
			return nil
		}

		// Determine the value for the external ID, which may be the empty
		// string for identities incoming from anonymous events.
		extIDValue := record.Properties["ExternalId"].(string)

		// Determine the label for the External ID, except for the case of
		// "anonymous identities", which are identities imported from anonymous
		// events. In that case, both the External ID value and label must be
		// empty.
		var extIDLabel string
		if extIDValue != "" {
			c := conn.Connector()
			switch c.Type {
			case state.AppType:
				extIDLabel = c.ExternalIDLabel
				if extIDLabel == "" {
					extIDLabel = "ID"
				}
			case state.DatabaseType, state.StorageType:
				extIDLabel = "ID"
			case state.MobileType, state.ServerType, state.WebsiteType:
				extIDLabel = "User ID"
			default:
				return fmt.Errorf("unexpected connector type %v", c.Type)
			}
		}

		// Determine the anonymous IDs.
		var anonIDs []string
		if ids, ok := record.Properties["AnonymousIds"].([]any); ok {
			anonIDs = make([]string, len(ids))
			for i := range ids {
				anonIDs[i] = ids[i].(string)
			}
		}

		// Determine the "updated_at" timestamp.
		updatedAt := record.Properties["UpdatedAt"].(time.Time)

		// Determine the Business ID.
		businessID := record.Properties["BusinessId"].(string)

		identities = append(identities, identity{
			Connection: connID,
			ExternalId: labelValue{
				Label: extIDLabel,
				Value: extIDValue,
			},
			BusinessId:   businessID,
			AnonymousIds: anonIDs,
			UpdatedAt:    updatedAt,
		})

		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	if err = records.Err(); err != nil {
		return nil, 0, err
	}

	// Since the count is an estimate, being counted separately from the actual
	// number of identities returned, ensure to not return a value lower than
	// the actually returned number of identities.
	count = max(len(identities), count)

	return identities, count, nil
}

func validateRePaths(rePaths map[string]any) error {
	for new, old := range rePaths {
		if !types.IsValidPropertyPath(new) {
			return fmt.Errorf("invalid property path: %q", new)
		}
		switch old := old.(type) {
		case string:
			if !types.IsValidPropertyPath(old) {
				return fmt.Errorf("invalid property path: %q", new)
			}
			if new == old {
				return fmt.Errorf("rePath key cannot match with its value")
			}
			if strings.Contains(old, ".") {
				oldParts := strings.Split(old, ".")
				oldPrefix := oldParts[:len(oldParts)-1]
				newParts := strings.Split(new, ".")
				newPrefix := newParts[:len(newParts)-1]
				if !slices.Equal(oldPrefix, newPrefix) {
					return fmt.Errorf("rePath contains a renamed property whose path is different")
				}
			}
		case nil:
			// Ok.
		default:
			return fmt.Errorf("unexpected value of type %T", old)
		}
	}
	return nil
}
