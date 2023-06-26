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
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/errors"
	_events "chichi/apis/events"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/warehouses"
	"chichi/apis/warehouses/clickhouse"
	"chichi/apis/warehouses/postgresql"
	_connector "chichi/connector"
	"chichi/connector/types"

	"github.com/jxskiss/base62"
	"golang.org/x/exp/slices"
)

const (
	maxEventsListenedTo = 1000 // maximum number of processed events listened to.
)

var (
	AlreadyConnected     errors.Code = "AlreadyConnected"
	ConnectionFailed     errors.Code = "ConnectionFailed"
	InvalidSchemaTable   errors.Code = "InvalidSchemaTable"
	InvalidSettings      errors.Code = "InvalidSettings"
	NoWarehouse          errors.Code = "NoWarehouse"
	NotConnected         errors.Code = "NotConnected"
	OrderNotExists       errors.Code = "OrderNotExists"
	OrderTypeNotSortable errors.Code = "OrderTypeNotSortable"
	PropertyNotExists    errors.Code = "PropertyNotExists"
	RepeatedPropertyName errors.Code = "RepeatedPropertyName"
	ServerNotExists      errors.Code = "ServerNotExists"
	SourceNotExists      errors.Code = "SourceNotExists"
	StreamNotExists      errors.Code = "StreamNotExists"
	TooManyListeners     errors.Code = "TooManyListeners"
	WarehouseFailed      errors.Code = "WarehouseFailed"
)

// ConnectionOptions values are passed to the AddConnection method with options
// relative to the connection.
type ConnectionOptions struct {

	// Name is the name of the connection. It cannot be longer than 100 runes.
	// If empty, the connection name will be the name of its connector.
	Name string

	// Enable reports whether the connection is enabled or disabled when added.
	Enabled bool

	// Storage is the storage of a file connection. It must be 0 if the
	// connection is not a file or has no storage.
	Storage int

	// Compression is the compression for file connections. It must be
	// NoCompression if there is no storage.
	Compression Compression

	// WebsiteHost is the host, in the form "host:port", of a website
	// connection. It must be empty if the connection is not a website. It
	// cannot be longer than 261 runes.
	WebsiteHost string

	// OAuth is an OAuth token returned by OAuthToken. It must be empty if the
	// connector does not support OAuth.
	OAuth string
}

// AddConnection adds a connection given its role, connector, settings, and
// options and returns its identifier.
//
// It returns an errors.UnprocessableError error with code
//   - ConnectorNotExists, if the connector does not exist.
//   - InvalidSettings, if the settings are not valid.
//   - StorageNotExists, if the storage does not exist.
func (this *Workspace) AddConnection(role ConnectionRole, connector int, settings []byte, opts ConnectionOptions) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.BadRequest("role %q is not valid", role)
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.BadRequest("connector identifier %d is not valid", connector)
	}
	if utf8.RuneCountInString(opts.Name) > 100 {
		return 0, errors.BadRequest("name %q is not valid", opts.Name)
	}
	if opts.Storage < 0 || opts.Storage > maxInt32 {
		return 0, errors.BadRequest("storage identifier %d is not valid", opts.Storage)
	}
	switch opts.Compression {
	case NoCompression, ZipCompression, GzipCompression, SnappyCompression:
	default:
		return 0, errors.BadRequest("compression %q is not valid", opts.Compression)
	}
	if opts.Storage == 0 && opts.Compression != NoCompression {
		return 0, errors.BadRequest("compression requires a storage")
	}

	c, ok := this.state.Connector(connector)
	if !ok {
		return 0, errors.Unprocessable(ConnectorNotExists, "connector %d does not exist", connector)
	}

	n := state.AddConnectionNotification{
		Workspace: this.workspace.ID,
		Name:      opts.Name,
		Role:      state.ConnectionRole(role),
		Enabled:   opts.Enabled,
		Connector: connector,
	}
	if opts.Name == "" {
		n.Name = c.Name
	}

	// Validate the storage.
	if opts.Storage > 0 {
		if c.Type != state.FileType {
			return 0, errors.BadRequest("connector %d cannot have a storage, it's a %s",
				c.ID, strings.ToLower(c.Type.String()))
		}
		s, ok := this.workspace.Connection(opts.Storage)
		if !ok {
			return 0, errors.Unprocessable(StorageNotExists, "storage %d does not exist", opts.Storage)
		}
		if s.Connector().Type != state.StorageType {
			return 0, errors.BadRequest("connection %d is not a storage", opts.Storage)
		}
		if ConnectionRole(s.Role) != role {
			if role == SourceRole {
				return 0, errors.BadRequest("storage %d is not a source", opts.Storage)
			}
			return 0, errors.BadRequest("storage %d is not a destination", opts.Storage)
		}
		n.Storage = opts.Storage
		n.Compression = state.Compression(opts.Compression)
	}

	// Validate the website host.
	if opts.WebsiteHost != "" {
		if c.Type != state.WebsiteType {
			return 0, errors.BadRequest("connector %d cannot have a website host, it's a %s",
				c.ID, strings.ToLower(c.Type.String()))
		}
		if h, p, found := strings.Cut(opts.WebsiteHost, ":"); h == "" || len(opts.WebsiteHost) > 255 {
			return 0, errors.BadRequest("website host %q is not valid", opts.WebsiteHost)
		} else if found {
			if port, _ := strconv.Atoi(p); port < 1 || port > 65535 {
				return 0, errors.BadRequest("website host %q is not valid", opts.WebsiteHost)
			}
		}
		n.WebsiteHost = opts.WebsiteHost
	}

	// Validate OAuth.
	if (opts.OAuth == "") != (c.OAuth == nil) {
		if opts.OAuth == "" {
			return 0, errors.BadRequest("OAuth is required by connector %d", connector)
		}
		return 0, errors.BadRequest("connector %d does not support OAuth", connector)
	}

	// Set the resource. It can be an existing resource or a resource that needs to be created.
	if opts.OAuth != "" {
		data, err := base62.DecodeString(opts.OAuth)
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

	ctx := context.Background()

	// Validate the settings.
	if c.HasSettings {
		var clientSecret string
		if c.OAuth != nil {
			clientSecret = c.OAuth.ClientSecret
		}
		connector := &Connector{connector: c, http: this.http}
		connectionUI, err := connector.openUI(ctx, role, n.Resource.Code, clientSecret, n.Resource.AccessToken)
		if err != nil {
			return 0, err
		}
		if connectionUI == nil {
			return 0, errors.BadRequest("connector %d does not have a UI", c.ID)
		}
		n.Settings, err = connectionUI.ValidateSettings(settings)
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

	// Generate a server key.
	if c.Type == state.ServerType {
		n.Key, err = generateServerKey()
		if err != nil {
			return 0, err
		}
	}

	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		if n.Resource.Code != "" {
			if n.Resource.ID == 0 {
				// Insert a new resource.
				err = tx.QueryRow(ctx, "INSERT INTO resources (workspace, connector, code, access_token,"+
					" refresh_token, expires_in) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
					n.Workspace, connector, n.Resource.Code, n.Resource.AccessToken, n.Resource.RefreshToken, n.Resource.ExpiresIn).
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
						err = errors.Unprocessable(WorkspaceNotExists, "workspace %d does not exist", n.Workspace)
					case "resources_connector_fkey":
						err = errors.Unprocessable(ConnectorNotExists, "connector %d does not exist", n.Connector)
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
			if err != nil {
				if postgres.IsForeignKeyViolation(err) {
					switch postgres.ErrConstraintName(err) {
					case "connections_workspace_fkey":
						err = errors.Unprocessable(WorkspaceNotExists, "workspace %d does not exist", n.Workspace)
					case "connections_connector_fkey":
						err = errors.Unprocessable(ConnectorNotExists, "connector %d does not exist", n.Connector)
					case "connections_storage_fkey":
						err = errors.Unprocessable(StorageNotExists, "storage %d does not exist", n.Storage)
					}
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

// AddEventListener adds a listener that listen to processed events
//
//   - occurred on the mobile or website connection source, if source is not
//     zero,
//   - sent by the server connection server, if server is not zero,
//   - received from the stream connection stream, if stream is not zero,
//
// and returns its identifier. size is the maximum number of events to return
// for each call to the Events method, it must be in [1,1000].
//
// If the source, server, or stream does not exist, it returns an
// errors.UnprocessableError error with code SourceNotExists, ServerNotExists,
// and StreamNotExists respectively.
// If there are already too many listeners, it returns an
// errors.UnprocessableError error with code TooManyListeners.
func (this *Workspace) AddEventListener(size, source, server, stream int) (string, error) {

	if size < 1 || size > maxEventsListenedTo {
		return "", errors.BadRequest("size %d is not valid", size)
	}
	if source < 0 || source > maxInt32 {
		return "", errors.BadRequest("source identifier %d is not valid", source)
	}
	if server < 0 || server > maxInt32 {
		return "", errors.BadRequest("server identifier %d is not valid", server)
	}
	if stream < 0 || stream > maxInt32 {
		return "", errors.BadRequest("stream identifier %d is not valid", stream)
	}

	if source > 0 || server > 0 || stream > 0 {

		var sourceExist, serverExist, streamExist bool
		err := this.db.QueryScan(context.Background(), "SELECT id, type , role FROM connections\n"+
			"WHERE id IN ($1, $2, $3) AND workspace = $4", source, server, stream, this.workspace.ID,
			func(rows *postgres.Rows) error {
				var id int
				var typ state.ConnectorType
				var role state.ConnectionRole
				for rows.Next() {
					if err := rows.Scan(&id, &typ, &role); err != nil {
						return err
					}
					switch id {
					case source:
						if typ != state.MobileType && typ != state.WebsiteType {
							return errors.BadRequest("connection %d is not a mobile or website", source)
						}
						sourceExist = true
					case server:
						if typ != state.ServerType {
							return errors.BadRequest("connection %d is not a server", server)
						}
						serverExist = true
					case stream:
						if typ != state.StreamType {
							return errors.BadRequest("connection %d is not a stream", stream)
						}
						streamExist = true
					}
					if role != state.SourceRole {
						return errors.BadRequest("connection %d is not a source", id)
					}
				}
				return nil
			})
		if err != nil {
			return "", err
		}
		if source > 0 && !sourceExist {
			return "", errors.Unprocessable(SourceNotExists, "source %d does not exist", source)
		}
		if server > 0 && !serverExist {
			return "", errors.Unprocessable(ServerNotExists, "server %d does not exist", server)
		}
		if stream > 0 && !streamExist {
			return "", errors.Unprocessable(StreamNotExists, "stream %d does not exist", stream)
		}

	}

	id, err := this.eventObserver.AddListener(size, source, server, stream)
	if err != nil {
		if err == _events.ErrTooManyListeners {
			err = errors.Unprocessable(TooManyListeners, "there are already %d listeners", _events.MaxEventListeners)
		}
		return "", err
	}

	return id, nil
}

// Delete deletes the workspace with all its connections.
//
// It returns an errors.NotFound error if the workspace does not exist anymore.
func (this *Workspace) Delete() error {
	n := state.DeleteWorkspaceNotification{
		ID: this.workspace.ID,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "DELETE FROM workspaces WHERE id = $1", n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("workspace %d does not exist", n.ID)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// Connection returns the connection with identifier id of the workspace ws.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
// It returns an errors.UnprocessableError error with code
//
//   - FetchSchemaFailed, if an error occurred fetching the schema.
func (this *Workspace) Connection(id int) (*Connection, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("connection identifier %d is not valid", id)
	}
	c, ok := this.workspace.Connection(id)
	if !ok {
		return nil, errors.NotFound("connection %d does not exist", id)
	}
	conn := c.Connector()

	connection := Connection{
		db:           this.db,
		connection:   c,
		http:         this.http,
		ID:           c.ID,
		Name:         c.Name,
		Type:         ConnectorType(conn.Type),
		Role:         ConnectionRole(c.Role),
		Connector:    conn.ID,
		HasSettings:  conn.HasSettings,
		Enabled:      c.Enabled,
		ActionsCount: len(c.Actions()),
		Health:       Health(c.Health),
	}
	// Set the storage.
	if s, ok := c.Storage(); ok {
		connection.Storage = s.ID
	}
	// Set the action types.
	ts, err := (&Connection{db: this.db, connection: c, http: this.http}).actionTypes()
	if err != nil {
		return nil, err
	}
	connection.ActionTypes = &ts
	// Set the actions.
	actions := c.Actions()
	a := make([]Action, len(actions))
	connection.Actions = &a
	for i, a := range actions {
		(*connection.Actions)[i].fromState(this.db, this.http, a)
	}
	return &connection, nil
}

// Connections returns the connections of the workspace.
func (this *Workspace) Connections() []*Connection {
	connections := this.workspace.Connections()
	infos := make([]*Connection, len(connections))
	for i, c := range connections {
		conn := c.Connector()
		connection := Connection{
			db:           this.db,
			connection:   c,
			http:         this.http,
			ID:           c.ID,
			Name:         c.Name,
			Type:         ConnectorType(conn.Type),
			Role:         ConnectionRole(c.Role),
			Connector:    conn.ID,
			HasSettings:  conn.HasSettings,
			Enabled:      c.Enabled,
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

// ConnectWarehouse connects a data warehouse, with the given settings, to the
// workspace. It also creates the tables in the connected data warehouse.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore, and it returns an errors.UnprocessableError error with code
//   - AlreadyConnected, if the workspace is already connected to a data
//     warehouse.
//   - InvalidSettings, if the settings are not valid.
//   - ConnectionFailed, if the connection fails.
func (this *Workspace) ConnectWarehouse(typ WarehouseType, settings []byte) error {
	ws := this.workspace
	if ws.Warehouse != nil {
		return errors.Unprocessable(AlreadyConnected, "workspace %d is already connected to a data warehouse", ws.ID)
	}
	warehouse, err := openWarehouse(typ, settings)
	if err != nil {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
	}
	err = warehouse.Ping(context.Background())
	if err != nil {
		return errors.Unprocessable(ConnectionFailed, "cannot connect to the data warehouse: %w", err)
	}
	n := state.SetWarehouseSettingsNotification{
		Workspace: ws.ID,
		Type:      state.WarehouseType(typ),
		Settings:  warehouse.Settings(),
	}
	ctx := context.Background()
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_type = $1, warehouse_settings = $2 WHERE id = $3"+
			" AND warehouse_type IS NULL",
			n.Type, string(n.Settings), n.Workspace)
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

// DisconnectWarehouse disconnects the data warehouse of the workspace.
//
// If the workspace does not exist anymore, it returns an errors.NotFoundError
// error. If the workspace is not connected to a data warehouse, it returns an
// errors.UnprocessableError error with code NotConnected.
func (this *Workspace) DisconnectWarehouse() error {
	ws := this.workspace
	if ws.Warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.ID)
	}
	n := state.SetWarehouseSettingsNotification{
		Workspace: ws.ID,
		Settings:  nil,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
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

// InitWarehouse initializes the data warehouse of the workspace by creating
// the supporting tables.
//
// If the workspace does not exist, it returns an errors.NotFoundError error.
// It returns an errors.UnprocessableError error with code NotConnected, if the
// workspace is not connected to a data warehouse.
func (this *Workspace) InitWarehouse() error {
	ws := this.workspace
	if ws.Warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.ID)
	}
	return ws.Warehouse.Init(context.Background())
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
// connection for the specified connector.
//
// It returns an errors.NotFound error if the workspace does not exist anymore.
// It returns an errors.UnprocessableError error with code ConnectorNotExists if
// the connector does not exist.
func (this *Workspace) OAuthToken(authorizationCode, redirectURI string, connector int) (string, error) {

	if authorizationCode == "" {
		return "", errors.BadRequest("authorization code is empty")
	}
	if connector < 1 || connector > maxInt32 {
		return "", errors.BadRequest("connector identifier %d is not valid", connector)
	}

	c, ok := this.state.Connector(connector)
	if !ok {
		return "", errors.Unprocessable(ConnectorNotExists, "connector %d does not exist", connector)
	}
	if c.OAuth == nil {
		return "", errors.BadRequest("connector %d does not support OAuth", connector)
	}

	accessToken, refreshToken, expiresIn, err := this.http.GrantAuthorization(context.Background(), c.OAuth, authorizationCode, redirectURI)
	if err != nil {
		return "", err
	}

	connection, err := _connector.RegisteredApp(c.Name).Open(context.Background(), &_connector.AppConfig{
		HTTPClient: this.http.Client(c.OAuth.ClientSecret, accessToken),
		Region:     _connector.PrivacyRegion(this.workspace.PrivacyRegion),
	})
	if err != nil {
		return "", err
	}
	code, err := connection.Resource()
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

// ListenedEvents returns the events listen to by the specified listener and
// the number of discarded events.
//
// It returns an errors.NotFoundError error, if the listener does not exist.
func (this *Workspace) ListenedEvents(listener string) ([]json.RawMessage, int, error) {
	events, discarded, err := this.eventObserver.Events(listener)
	if err != nil {
		if err == _events.ErrEventListenerNotFound {
			return nil, 0, errors.NotFound("event listener %q does not exist", listener)
		}
		return nil, 0, err
	}
	return events, discarded, nil
}

// ReloadSchemas reloads the schemas of the workspace.
//
// It returns an errors.NotFoundError error, if the workspace does not exist,
// and it returns an errors.UnprocessableError error with code
//   - NotConnected, if the workspace is not connected to a data warehouse.
//   - WarehouseFailed, if the connection to the data warehouse failed.
//   - InvalidSchemaTable, if a table of a schema is not valid.
func (this *Workspace) ReloadSchemas() error {
	ws := this.workspace
	if ws.Warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.ID)
	}
	tables, err := ws.Warehouse.Tables(context.Background())
	if err != nil {
		if err, ok := err.(*warehouses.Error); ok {
			return errors.Unprocessable(WarehouseFailed, "data warehouse has returned an error: %w", err.Err)
		}
		return err
	}
	n := state.SetWorkspaceSchemasNotification{
		Workspace: ws.ID,
		Schemas:   map[string]*types.Type{},
	}
	for _, table := range tables {
		// Check that the 'users' and the 'groups' tables, when exists, contain
		// the 'id' and the 'timestamp' columns.
		if table.Name == "users" || table.Name == "groups" {
			// Check the 'id' column.
			idIndex := slices.IndexFunc(table.Columns, func(c types.Property) bool {
				return c.Name == "id"
			})
			if idIndex == -1 {
				return errors.Unprocessable(InvalidSchemaTable, "'%s' table has no 'id' column", table.Name)
			}
			if c := table.Columns[idIndex]; c.Type.PhysicalType() != types.PtInt {
				return errors.Unprocessable(InvalidSchemaTable, "column '%s.id' does not have type Int", table.Name)
			} else if c.Nullable {
				return errors.Unprocessable(InvalidSchemaTable, "column '%s.id' must not be nullable", table.Name)
			}
			// Check the 'creation_time' and 'timestamp' columns.
			for _, column := range []string{"creation_time", "timestamp"} {
				colIndex := slices.IndexFunc(table.Columns, func(c types.Property) bool {
					return c.Name == column
				})
				if colIndex == -1 {
					return errors.Unprocessable(InvalidSchemaTable, "'%s' table has no '%s' column", column, table.Name)
				}
				if c := table.Columns[colIndex]; c.Type.PhysicalType() != types.PtDateTime {
					return errors.Unprocessable(InvalidSchemaTable, "column '%s.%s' does not have type DateTime", table.Name, column)
				} else if c.Nullable {
					return errors.Unprocessable(InvalidSchemaTable, "column '%s.%s' must not be nullable", table.Name, column)
				}
			}
		}
		if table.Name == "events" {
			// The schema of the "events" table is hardcoded in the file
			// "apis/events/schema.go".
			continue
		}
		properties, err := warehouses.ColumnsToProperties(table.Columns)
		if err, ok := err.(warehouses.RepeatedPropertyNameError); ok {
			return errors.Unprocessable(RepeatedPropertyName,
				"column %s.%s results in a repeated property named %s", table.Name, err.Column, err.Property)
		}
		schema := types.Object(properties)
		n.Schemas[table.Name] = &schema
	}
	newRawSchemas, err := json.Marshal(n.Schemas)
	if err != nil {
		return fmt.Errorf("cannot marshal data warehouse schema for workspace %d: %s", ws.ID, err)
	}
	ctx := context.Background()
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
		if bytes.Equal(newRawSchemas, oldRawSchemas) {
			return nil
		}
		_, err = tx.Exec(ctx, "UPDATE workspaces SET schemas = $1 WHERE id = $2", newRawSchemas, n.Workspace)
		if err != nil {
			return err
		}
		if len(oldRawSchemas) > 0 {
			var oldSchemas map[string]*types.Type
			err = json.Unmarshal(oldRawSchemas, &oldSchemas)
			if err != nil {
				return fmt.Errorf("cannot parse schemas of workspace %d: %s", n.Workspace, err)
			}
			for name, t := range n.Schemas {
				if t2, ok := oldSchemas[name]; ok && t.EqualTo(*t2) {
					n.Schemas[name] = nil
				}
			}
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// RemoveEventListener removes the given event listener. It does nothing if the
// listener does not exist.
func (this *Workspace) RemoveEventListener(listener string) {
	this.eventObserver.RemoveListener(listener)
}

// Rename renames the workspace with the given new name.
// name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the workspace does not exist
// anymore.
func (this *Workspace) Rename(name string) error {
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return errors.BadRequest("name %q is not valid", name)
	}
	if name == this.workspace.Name {
		return nil
	}
	n := state.RenameWorkspaceNotification{
		Workspace: this.workspace.ID,
		Name:      name,
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
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
	ws := this.workspace
	schema, ok := ws.Schemas[name]
	if !ok {
		return types.Type{}
	}
	return schema.Unflatten()
}

// SetPrivacyRegion sets the privacy region of the workspace.
func (this *Workspace) SetPrivacyRegion(region PrivacyRegion) error {
	switch region {
	case PrivacyRegionNotSpecified,
		PrivacyRegionEurope:
	default:
		return errors.BadRequest("invalid privacy region %q", string(region))
	}
	ws := this.workspace
	n := state.SetWorkspacePrivacyRegion{
		Workspace:     ws.ID,
		PrivacyRegion: state.PrivacyRegion(region),
	}
	ctx := context.Background()
	err := this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE workspaces SET privacy_region = $1 WHERE id = $2",
			n.PrivacyRegion, n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// SetWarehouseSettings sets the settings of the workspace's data warehouse.
//
// It returns an errors.NotFoundError error, if the workspace does not exist,
// and it returns an errors.UnprocessableError error with code
//   - NotConnected, if the workspace is not connected to a data warehouse.
//   - InvalidSettings, if the settings are not valid.
//   - ConnectionFailed, if the connection fails.
func (this *Workspace) SetWarehouseSettings(typ WarehouseType, settings []byte) error {
	ws := this.workspace
	if ws.Warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.ID)
	}
	if typ != typeOfWarehouse(ws.Warehouse) {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", fmt.Errorf(
			"workspace %d is connected to a %s data warehouse, but settings are for a %s data warehouse",
			ws.ID, typeOfWarehouse(ws.Warehouse), typ))
	}
	warehouse, err := openWarehouse(typ, settings)
	if err != nil {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
	}
	err = warehouse.Ping(context.Background())
	if err != nil {
		return errors.Unprocessable(ConnectionFailed, "cannot connect to the data warehouse: %w", err)
	}
	n := state.SetWarehouseSettingsNotification{
		Workspace: ws.ID,
		Type:      state.WarehouseType(typ),
		Settings:  warehouse.Settings(),
	}
	ctx := context.Background()
	err = this.db.Transaction(ctx, func(tx *postgres.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_settings = $1 WHERE id = $2 AND warehouse_type = $3",
			string(n.Settings), n.Workspace, n.Type)
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
			return errors.Unprocessable(NoWarehouse, "workspace %d is not connected to a PostgreSQL data warehouse", ws.ID)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// User returns the user with identifier id. If the user does not exist, the
// error is deferred until methods of *User are called.
func (this *Workspace) User(id int) (*User, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("user identifier %d is not valid", id)
	}
	return &User{
		workspace: this.workspace,
		id:        id,
	}, nil
}

// Users returns the user schema and the users of the workspace. It returns
// the users in range [first,first+limit] with first >= 0 and 0 < limit <= 1000
// and only the given properties. properties cannot be empty.
//
// order is the property by which to sort the returned users and cannot have
// type JSON, Array, Object, or Map.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore.
// It returns an errors.UnprocessableError error with code
//
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - OrderNotExist, if order does not exist in schema.
//   - OrderTypeNotSortable, if the type of the order property is not sortable.
//   - PropertyNotExist, if a property does not exist.
//   - WarehouseFailed, if the data warehouse failed.
func (this *Workspace) Users(properties []string, order string, first, limit int) (types.Type, [][]any, error) {

	ws := this.workspace

	// Verify that the workspace has a data warehouse.
	if ws.Warehouse == nil {
		return types.Type{}, nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Read the schema.
	var schemaProperties []types.Property
	if typ, ok := ws.Schemas["users"]; ok {
		schemaProperties = typ.Properties()
	}
	propertyByName := map[string]types.Property{}
	for _, p := range schemaProperties {
		propertyByName[p.Name] = p
	}

	// Validate the arguments.
	if len(properties) == 0 {
		return types.Type{}, nil, errors.BadRequest("properties is empty")
	}
	for _, name := range properties {
		if _, ok := propertyByName[name]; !ok {
			if name == "" {
				return types.Type{}, nil, errors.BadRequest("a property name is empty")
			}
			if !types.IsValidPropertyName(name) {
				return types.Type{}, nil, errors.BadRequest("property name %q is not valid", name)
			}
			return types.Type{}, nil, errors.Unprocessable(PropertyNotExists, "property name %s does not exist", name)
		}
	}
	var orderProperty types.Property
	if order != "" {
		if !types.IsValidPropertyName(order) {
			return types.Type{}, nil, errors.BadRequest("order %q is not a valid property name", order)
		}
		orderProperty, ok := propertyByName[order]
		if !ok {
			return types.Type{}, nil, errors.Unprocessable(OrderNotExists, "order %s does not exist in schema", order)
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
	columns := warehouses.PropertiesToColumns(requestedProperties)
	rows, err := ws.Warehouse.Select(context.Background(), "users", columns, nil, orderProperty, first, limit)
	if err != nil {
		if err2, ok := err.(*warehouses.Error); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("[error] cannot get users from the data warehouse of the workspace %d: %s", ws.ID, err)
			err = errors.Unprocessable(WarehouseFailed, "warehouse connection is failed: %w", err2.Err)
		}
		return types.Type{}, nil, err
	}

	users := make([][]any, len(rows))
	for i, row := range rows {
		users[i] = warehouses.DeserializeRowAsSlice(requestedProperties, row)
	}

	return schema.Unflatten(), users, err
}

// openWarehouse opens a data warehouse with the given type and settings.
// It returns an error if typ or settings are not valid.
func openWarehouse(typ WarehouseType, settings []byte) (warehouses.Warehouse, error) {
	switch typ {
	case BigQuery, Redshift, Snowflake:
		return nil, fmt.Errorf("warehouse type %s is not yet supported", typ)
	case PostgreSQL:
		return postgresql.Open(settings)
	case ClickHouse:
		return clickhouse.Open(settings)
	}
	return nil, fmt.Errorf("warehouse type %d is not valid", typ)
}

// typeOfWarehouse returns the type of the given data warehouse.
func typeOfWarehouse(warehouse warehouses.Warehouse) WarehouseType {
	switch warehouse.(type) {
	case *clickhouse.ClickHouse:
		return ClickHouse
	case *postgresql.PostgreSQL:
		return PostgreSQL
	}
	panic("unknown Warehouse")
}

// WarehouseType represents a data warehouse type.
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
	panic("invalid warehouse type")
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
