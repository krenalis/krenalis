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
	"io"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/datastore"
	"chichi/apis/datastore/expr"
	"chichi/apis/datastore/warehouses"
	"chichi/apis/datastore/warehouses/clickhouse"
	"chichi/apis/datastore/warehouses/postgresql"
	"chichi/apis/datastore/warehouses/snowflake"
	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/mappings/mapexp"
	"chichi/apis/postgres"
	"chichi/apis/state"
	_connector "chichi/connector"
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
//   - StorageNotExist, if the storage does not exist.
func (this *Workspace) AddConnection(ctx context.Context, connection ConnectionToAdd, oAuthToken string) (int, error) {

	this.apis.mustBeOpen()

	if connection.Role != SourceRole && connection.Role != DestinationRole {
		return 0, errors.BadRequest("role %d is not valid", int(connection.Role))
	}
	if connection.Connector < 1 || connection.Connector > maxInt32 {
		return 0, errors.BadRequest("connector identifier %d is not valid", connection.Connector)
	}
	if utf8.RuneCountInString(connection.Name) > 100 {
		return 0, errors.BadRequest("name %q is not valid", connection.Name)
	}
	if connection.Storage < 0 || connection.Storage > maxInt32 {
		return 0, errors.BadRequest("storage identifier %d is not valid", connection.Storage)
	}
	switch connection.Compression {
	case NoCompression, ZipCompression, GzipCompression, SnappyCompression:
	default:
		return 0, errors.BadRequest("compression %q is not valid", connection.Compression)
	}
	if connection.Storage == 0 && connection.Compression != NoCompression {
		return 0, errors.BadRequest("compression requires a storage")
	}

	c, ok := this.apis.state.Connector(connection.Connector)
	if !ok {
		return 0, errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", connection.Connector)
	}

	n := state.AddConnection{
		Workspace:   this.workspace.ID,
		Name:        connection.Name,
		Role:        state.ConnectionRole(connection.Role),
		Enabled:     connection.Enabled,
		Connector:   connection.Connector,
		Storage:     connection.Storage,
		Compression: state.Compression(connection.Compression),
		WebsiteHost: connection.WebsiteHost,
	}
	if n.Name == "" {
		n.Name = c.Name
	}

	// Validate the storage.
	if n.Storage > 0 {
		if c.Type != state.FileType {
			return 0, errors.BadRequest("connector %d cannot have a storage, it's a %s",
				c.ID, strings.ToLower(c.Type.String()))
		}
		s, ok := this.workspace.Connection(n.Storage)
		if !ok {
			return 0, errors.Unprocessable(StorageNotExist, "storage %d does not exist", n.Storage)
		}
		if s.Connector().Type != state.StorageType {
			return 0, errors.BadRequest("connection %d is not a storage", n.Storage)
		}
		if ConnectionRole(s.Role) != connection.Role {
			if connection.Role == SourceRole {
				return 0, errors.BadRequest("storage %d is not a source", n.Storage)
			}
			return 0, errors.BadRequest("storage %d is not a destination", n.Storage)
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
		var clientSecret string
		if c.OAuth != nil {
			clientSecret = c.OAuth.ClientSecret
		}
		connector := &Connector{apis: this.apis, connector: c}
		connectionUI, err := connector.openUI(connection.Role, n.Resource.Code, clientSecret, n.Resource.AccessToken)
		if err != nil {
			return 0, err
		}
		if connectionUI == nil {
			return 0, errors.BadRequest("connector %d does not have a UI", c.ID)
		}
		n.Settings, err = connectionUI.ValidateSettings(ctx, connection.Settings)
		if c, ok := connectionUI.(io.Closer); ok {
			_ = c.Close()
		}
		if err != nil {
			return 0, errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
		}
		if !utf8.Valid(n.Settings) {
			return 0, errors.New("settings is not valid UTF-8")
		}
		if utf8.RuneCount(n.Settings) > maxSettingsLen {
			return 0, fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
		}
	}

	// Generate the identifier.
	var err error
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

	err = this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
			"(id, workspace, name, type, role, enabled, connector, storage, compression, resource, website_host, settings)"+
			" VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, 0), $9, $10, $11, $12)", n.ID, n.Workspace, n.Name, c.Type,
			n.Role, n.Enabled, n.Connector, n.Storage, n.Compression, n.Resource.ID, n.WebsiteHost, string(n.Settings))
		if err != nil {
			if postgres.IsForeignKeyViolation(err) {
				switch postgres.ErrConstraintName(err) {
				case "connections_workspace_fkey":
					err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
				case "connections_connector_fkey":
					err = errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", n.Connector)
				case "connections_storage_fkey":
					err = errors.Unprocessable(StorageNotExist, "storage %d does not exist", n.Storage)
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
		var role state.ConnectionRole
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
		if role != state.SourceRole {
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

	settings, err := this.account.apis.datastore.NormalizeWarehouseSettings(ws.Warehouse.Type, settings)
	if err != nil {
		if err, ok := err.(*datastore.SettingsError); ok {
			return errors.Unprocessable(InvalidSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return err
	}

	err = this.account.apis.datastore.PingWarehouse(ctx, ws.Warehouse.Type, settings)
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

	err = this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
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

	connection := Connection{
		apis:         this.apis,
		store:        this.store,
		connection:   c,
		ID:           c.ID,
		Name:         c.Name,
		Type:         ConnectorType(conn.Type),
		Role:         ConnectionRole(c.Role),
		Enabled:      c.Enabled,
		Connector:    conn.ID,
		Compression:  Compression(c.Compression),
		WebsiteHost:  c.WebsiteHost,
		HasSettings:  conn.HasSettings,
		ActionsCount: len(c.Actions()),
		Health:       Health(c.Health),
	}
	// Set the storage.
	if s, ok := c.Storage(); ok {
		connection.Storage = s.ID
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
			Role:         ConnectionRole(c.Role),
			Enabled:      c.Enabled,
			Connector:    conn.ID,
			Compression:  Compression(c.Compression),
			WebsiteHost:  c.WebsiteHost,
			HasSettings:  conn.HasSettings,
			ActionsCount: len(c.Actions()),
			Health:       Health(c.Health),
		}
		if s, ok := c.Storage(); ok {
			connection.Storage = s.ID
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

	settings, err := this.account.apis.datastore.NormalizeWarehouseSettings(state.WarehouseType(typ), settings)
	if err != nil {
		if err, ok := err.(*datastore.SettingsError); ok {
			return errors.Unprocessable(InvalidSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return err
	}

	schemas, err := this.account.apis.datastore.WarehouseSchemas(ctx, state.WarehouseType(typ), settings)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return errors.Unprocessable(DataWarehouseFailed, "an error occurred with the data warehouse: %w", err.Err)
		}
		return err
	}

	n := state.SetWarehouse{
		Workspace: ws.ID,
		Warehouse: &state.Warehouse{
			Type:     state.WarehouseType(typ),
			Settings: settings,
		},
		Schemas: map[string]*types.Type{},
	}
	for _, table := range []string{"users", "users_identities", "groups", "groups_identities", "events"} {
		schema, ok := schemas[table]
		if !ok {
			return errors.Unprocessable(DataWarehouseFailed, "table %q does not exist in the data warehouse", table)
		}
		if err = validateSchema(table, schema); err != nil {
			return errors.Unprocessable(DataWarehouseFailed, "%s", err)
		}
		n.Schemas[table] = &schema
	}

	rawSchemas, err := json.Marshal(n.Schemas)
	if err != nil {
		return fmt.Errorf("cannot marshal schemas for workspace %d: %s", this.workspace.ID, err)
	}

	err = this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_type = $1, warehouse_settings = $2, schemas = $3"+
			"  WHERE id = $4 AND warehouse_type IS NULL",
			n.Warehouse.Type, string(n.Warehouse.Settings), rawSchemas, n.Workspace)
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
// If the workspace is currently connected to a data warehouse, it returns a
// UnprocessableError error with code CurrentlyConnected.
func (this *Workspace) Delete(ctx context.Context) error {
	this.apis.mustBeOpen()
	ws := this.workspace
	if ws.Warehouse != nil {
		return errors.Unprocessable(CurrentlyConnected, "workspace %d is currently connected to %s data warehouse", ws.ID, ws.Warehouse.Type)
	}
	n := state.DeleteWorkspace{
		ID: this.workspace.ID,
	}
	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
		_, err = tx.Exec(ctx, "UPDATE workspaces SET warehouse_type = NULL, warehouse_settings = '', schemas = '' WHERE id = $1", n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
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
	for i := 0; i < len(evs); i++ {
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
// redirect URL used to obtain that code, that can be used to add a new
// connection to the workspace for the specified connector.
//
// It returns an errors.NotFound error if the workspace does not exist anymore.
// It returns an errors.UnprocessableError error with code ConnectorNotExist if
// the connector does not exist.
func (this *Workspace) OAuthToken(ctx context.Context, authorizationCode, redirectURI string, connector int) (string, error) {

	this.apis.mustBeOpen()

	if authorizationCode == "" {
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

	accessToken, refreshToken, expiresIn, err := this.apis.http.GrantAuthorization(ctx, c.OAuth, authorizationCode, redirectURI)
	if err != nil {
		return "", err
	}

	connection, err := _connector.RegisteredApp(c.Name).Open(&_connector.AppConfig{
		HTTPClient: this.apis.http.Client(c.OAuth.ClientSecret, accessToken),
		Region:     _connector.PrivacyRegion(this.workspace.PrivacyRegion),
	})
	if err != nil {
		return "", err
	}
	code, err := connection.Resource(ctx)
	if err != nil {
		return "", err
	}

	resource, err := json.Marshal(authorizedResource{
		Workspace:    this.workspace.ID,
		Connector:    connector,
		Code:         code,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	})

	// TODO(marco): Encrypt the token.

	return base62.EncodeToString(resource), nil
}

// ReloadSchemas reloads the users, groups and events schemas of the workspace.
//
// It returns an errors.NotFoundError error, if the workspace does not exist,
// and it returns an errors.UnprocessableError error with code
//   - NotConnected, if the workspace is not connected to a data warehouse.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) ReloadSchemas(ctx context.Context) error {

	this.apis.mustBeOpen()

	if this.store == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", this.workspace.ID)
	}

	schemas, err := this.store.Schemas(ctx)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return errors.Unprocessable(DataWarehouseFailed, "data warehouse has returned an error: %w", err.Err)
		}
		return err
	}

	n := state.SetWorkspaceSchemas{
		Workspace: this.workspace.ID,
		Schemas:   map[string]*types.Type{},
	}
	for _, table := range []string{"users", "users_identities", "groups", "groups_identities", "events"} {
		schema, ok := schemas[table]
		if !ok {
			return errors.Unprocessable(DataWarehouseFailed, "table %q does not exist in the data warehouse", table)
		}
		if err = validateSchema(table, schema); err != nil {
			return errors.Unprocessable(DataWarehouseFailed, "%s", err)
		}
		n.Schemas[table] = &schema
	}

	rawSchemas, err := json.Marshal(n.Schemas)
	if err != nil {
		return fmt.Errorf("cannot marshal schemas for workspace %d: %s", this.workspace.ID, err)
	}

	err = this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		var typ *state.WarehouseType
		var oldRawSchemas []byte
		err := tx.QueryRow(ctx, "SELECT warehouse_type, schemas FROM workspaces WHERE id = $1", n.Workspace).Scan(&typ, &oldRawSchemas)
		if err != nil {
			if err == sql.ErrNoRows {
				err = errors.NotFound("workspace %d does not exist", n.Workspace)
			}
			return err
		}
		if typ == nil {
			return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", n.Workspace)
		}
		if bytes.Equal(rawSchemas, oldRawSchemas) {
			return nil
		}
		_, err = tx.Exec(ctx, "UPDATE workspaces SET schemas = $1 WHERE id = $2", rawSchemas, n.Workspace)
		if err != nil {
			return err
		}
		if len(oldRawSchemas) > 0 {
			var oldSchemas map[string]*types.Type
			err = json.Unmarshal(oldRawSchemas, &oldSchemas)
			if err != nil {
				return fmt.Errorf("cannot parse schemas of workspace %d: %s", n.Workspace, err)
			}
			for name, schema := range n.Schemas {
				if oldSchema, ok := oldSchemas[name]; ok && schema.EqualTo(*oldSchema) {
					n.Schemas[name] = nil
				}
			}
		}
		return tx.Notify(ctx, n)
	})

	return err
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
	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
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

// Schema returns the schema, with the given name, of the workspace. If the
// schema does not exist, it returns an invalid schema.
func (this *Workspace) Schema(name string) types.Type {
	this.apis.mustBeOpen()
	ws := this.workspace
	schema, ok := ws.Schemas[name]
	if !ok {
		return types.Type{}
	}
	return schema.Unflatten()
}

// SetIdentifiers sets the identifiers and the anonymous identifiers of the
// workspace.
func (this *Workspace) SetIdentifiers(ctx context.Context, identifiers []string, anonIdentifiers AnonymousIdentifiers) error {

	this.apis.mustBeOpen()

	// Validate the identifiers.
	for i, id := range identifiers {
		if !types.IsValidPropertyPath(id) {
			return errors.BadRequest("identifier %q is not a valid property path", id)
		}
		if slices.Contains(identifiers[i+1:], id) {
			return errors.BadRequest("identifier %s is repeated", id)
		}
	}

	// Validate the anonymous identifiers.
	for i, id := range anonIdentifiers.Priority {
		if !types.IsValidPropertyPath(id) {
			return errors.BadRequest("anonymous identifier %q is not a valid property path", id)
		}
		if slices.Contains(anonIdentifiers.Priority[i+1:], id) {
			return errors.BadRequest("anonymous identifier %s is repeated", id)
		}
		expr, ok := anonIdentifiers.Mapping[id]
		if !ok {
			return errors.BadRequest("anonymous identifier %s does not have a mapped expression", id)
		}
		_, err := mapexp.Compile(expr, events.Schema, types.JSON(), true)
		if err != nil {
			return errors.BadRequest("expression of anonymous identifier %s is not valid: %w", id, err)
		}
	}
	if len(anonIdentifiers.Priority) != len(anonIdentifiers.Mapping) {
		for _, id := range anonIdentifiers.Priority {
			delete(anonIdentifiers.Mapping, id)
		}
		keys := maps.Keys(anonIdentifiers.Mapping)
		slices.Sort(keys)
		return errors.BadRequest("anonymous identifier %q does not exist in mapping", keys[0])
	}

	// Update the database and send the notification.
	if identifiers == nil {
		identifiers = []string{}
	}
	if anonIdentifiers.Mapping == nil {
		anonIdentifiers.Mapping = map[string]string{}
	}
	if anonIdentifiers.Priority == nil {
		anonIdentifiers.Priority = []string{}
	}
	ws := this.workspace
	n := state.SetWorkspaceIdentifiers{
		Workspace:            ws.ID,
		Identifiers:          identifiers,
		AnonymousIdentifiers: state.AnonymousIdentifiers(anonIdentifiers),
	}
	mapping, err := json.Marshal(anonIdentifiers.Mapping)
	if err != nil {
		return err
	}
	err = this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE workspaces\n"+
			"SET identifiers = $1, anonymous_identifiers_priority = $2, anonymous_identifiers_mapping = $3\n"+
			"WHERE id = $4", n.Identifiers, n.AnonymousIdentifiers.Priority, mapping, n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// Set sets the name and the privacy region of the workspace. name must be
// between 1 and 100 runes long.
func (this *Workspace) Set(ctx context.Context, name string, region PrivacyRegion) error {
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
	ws := this.workspace
	n := state.SetWorkspace{
		Workspace:     ws.ID,
		Name:          name,
		PrivacyRegion: state.PrivacyRegion(region),
	}
	err := this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE workspaces SET name = $1, privacy_region = $2 WHERE id = $3",
			n.Name, n.PrivacyRegion, n.Workspace)
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
	err = this.apis.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
	err := this.account.apis.datastore.PingWarehouse(ctx, state.WarehouseType(typ), settings)
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

// Users returns the user schema and the users of the workspace. It returns
// the users that satisfies the filter, if not nil, and in range
// [first,first+limit] with first >= 0 and 0 < limit <= 1000 and only the given
// properties. properties cannot be empty.
//
// order is the property by which to sort the returned users and cannot have
// type JSON, Array, Object, or Map.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore.
// It returns an errors.UnprocessableError error with code
//
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - NoUsersSchema, if the data warehouse does not have users schema.
//   - NoWarehouse, if the workspace does not have a data store.
//   - OrderNotExist, if order does not exist in schema.
//   - OrderTypeNotSortable, if the type of the order property is not sortable.
//   - PropertyNotExist, if a property does not exist.
func (this *Workspace) Users(ctx context.Context, properties []string, filter *Filter, order string, first, limit int) (types.Type, [][]any, error) {

	this.apis.mustBeOpen()

	ws := this.workspace

	// Verify that the workspace has a data store.
	if this.store == nil {
		return types.Type{}, nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data store", ws.ID)
	}

	// Read the schema.
	usersSchema, ok := ws.Schemas["users"]
	if !ok {
		return types.Type{}, nil, errors.Unprocessable(NoUsersSchema, "workspace %d does not have users schema", ws.ID)
	}

	// Validate the arguments.
	if len(properties) == 0 {
		return types.Type{}, nil, errors.BadRequest("properties is empty")
	}
	propertyByName := map[string]types.Property{}
	for _, p := range usersSchema.Properties() {
		propertyByName[p.Name] = p
	}
	for _, name := range properties {
		if _, ok := propertyByName[name]; !ok {
			if name == "" {
				return types.Type{}, nil, errors.BadRequest("a property name is empty")
			}
			if !types.IsValidPropertyName(name) {
				return types.Type{}, nil, errors.BadRequest("property name %q is not valid", name)
			}
			return types.Type{}, nil, errors.Unprocessable(PropertyNotExist, "property name %s does not exist", name)
		}
	}
	var where expr.Expr
	if filter != nil {
		_, err := validateFilter(filter, *usersSchema)
		if err != nil {
			if err, ok := err.(types.PathNotExistError); ok {
				return types.Type{}, nil, errors.Unprocessable(PropertyNotExist, "filter's property %s does not exist", err.Path)
			}
			return types.Type{}, nil, errors.BadRequest("filter is not valid: %w", err)
		}
		where, _ = convertFilterToExpr(filter, *usersSchema)
	}
	var orderProperty types.Property
	if order != "" {
		if !types.IsValidPropertyName(order) {
			return types.Type{}, nil, errors.BadRequest("order %q is not a valid property name", order)
		}
		orderProperty, ok := propertyByName[order]
		if !ok {
			return types.Type{}, nil, errors.Unprocessable(OrderNotExist, "order %s does not exist in schema", order)
		}
		switch orderProperty.Type.PhysicalType() {
		case types.PtJSON, types.PtArray, types.PtObject, types.PtMap:
			return types.Type{}, nil, errors.Unprocessable(OrderTypeNotSortable,
				"cannot sort by %s: property has type %s", order, orderProperty.Type)
		}
	}
	if first < 0 || first > maxInt32 {
		return types.Type{}, nil, errors.BadRequest("first %d in not valid", first)
	}
	if limit < 1 || limit > 1000 {
		return types.Type{}, nil, errors.BadRequest("limit %d is not valid", limit)
	}

	// Create the schema to return, with only the requested properties.
	requestedProperties := make([]types.Property, len(properties))
	for i, name := range properties {
		requestedProperties[i] = propertyByName[name]
	}
	schema := types.Object(requestedProperties)

	// Read the users.
	users, err := this.store.UsersSlice(ctx, requestedProperties, where, orderProperty, first, limit)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			slog.Error("cannot get users from the data store", "workspace", ws.ID, "err", err)
			return types.Type{}, nil, errors.Unprocessable(DataWarehouseFailed, "store connection is failed: %w", err.Err)
		}
		return types.Type{}, nil, err
	}

	return schema.Unflatten(), users, err
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
	Role ConnectionRole

	// Enable reports whether the connection is enabled or disabled when added.
	Enabled bool

	// Connector is the identifier of the connector.
	Connector int

	// Storage is the identifier of the storage of a file connection.
	// It must be 0 if the connection is not a file or has no storage.
	Storage int

	// Compression is the compression for file connections. It must be
	// NoCompression if there is no storage.
	Compression Compression

	// WebsiteHost is the host, in the form "host:port", of a website
	// connection. It must be empty if the connection is not a website. It
	// cannot be longer than 261 runes.
	WebsiteHost string

	// Settings represents the settings.
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

// validateSchema validates the schema of a data warehouse table.
func validateSchema(table string, schema types.Type) error {
	properties := schema.Properties()
	if table == "users" || table == "groups" {
		idIndex := slices.IndexFunc(properties, func(p types.Property) bool {
			return p.Name == "id"
		})
		if idIndex == -1 {
			return fmt.Errorf("'%s' schema has no 'id' property", table)
		}
		if p := properties[idIndex]; p.Type.PhysicalType() != types.PtInt && p.Type.PhysicalType() != types.PtDecimal {
			return fmt.Errorf("property '%s.id' does not have types Int or Decimal", table)
		} else if p.Nullable {
			return fmt.Errorf("property '%s.id' cannot be nullable", table)
		}
	}
	return nil
}
