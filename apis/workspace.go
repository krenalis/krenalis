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
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/apis/types"
	"chichi/apis/warehouses"
	"chichi/apis/warehouses/clickhouse"
	"chichi/apis/warehouses/postgresql"
	_connector "chichi/connector"

	"golang.org/x/exp/slices"
)

var (
	AlreadyConnected     errors.Code = "AlreadyConnected"
	ConnectionFailed     errors.Code = "ConnectionFailed"
	InvalidSchemaTable   errors.Code = "InvalidSchemaTable"
	InvalidSettings      errors.Code = "InvalidSettings"
	NoWarehouse          errors.Code = "NoWarehouse"
	NotConnected         errors.Code = "NotConnected"
	OrderNotExist        errors.Code = "OrderNotExist"
	OrderTypeNotSortable errors.Code = "OrderTypeNotSortable"
	PropertyNotExist     errors.Code = "PropertyNotExist"
	RepeatedPropertyName errors.Code = "RepeatedPropertyName"
	WarehouseFailed      errors.Code = "WarehouseFailed"
)

// AddConnection adds a connection given its role, connector, name, and options
// related to the connector and returns its identifier. name cannot be empty
// and cannot be longer than 120 runes.
//
// If the connector, storage or stream does not exist, it returns an
// errors.UnprocessableError error with code ConnectorNotExist, StorageNotExist
// and StreamNotExist respectively.
func (this *Workspace) AddConnection(role ConnectionRole, connector int, name string, opts ConnectionOptions) (int, error) {

	if role != SourceRole && role != DestinationRole {
		return 0, errors.BadRequest("role %q is not valid", role)
	}
	if connector < 1 || connector > maxInt32 {
		return 0, errors.BadRequest("connector identifier %d is not valid", connector)
	}
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return 0, errors.BadRequest("name %q is not valid", name)
	}
	if opts.Storage < 0 || opts.Storage > maxInt32 {
		return 0, errors.BadRequest("storage identifier %d is not valid", opts.Storage)
	}
	if opts.Stream < 0 || opts.Stream > maxInt32 {
		return 0, errors.BadRequest("stream identifier %d is not valid", opts.Stream)
	}
	if opts.OAuth != nil {
		if opts.OAuth.AccessToken == "" {
			return 0, errors.BadRequest("OAuth access token is empty")
		}
		if opts.OAuth.RefreshToken == "" {
			return 0, errors.BadRequest("OAuth refresh token is empty")
		}
	}

	n := state.AddConnectionNotification{
		Workspace: this.workspace.ID,
		Name:      name,
		Role:      state.ConnectionRole(role),
		Connector: connector,
	}
	c, ok := this.state.Connector(connector)
	if !ok {
		return 0, errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", connector)
	}

	// Validate the storage.
	if opts.Storage > 0 {
		if c.Type != state.FileType {
			return 0, errors.BadRequest("connector %d cannot have a storage, it's a %s",
				c.ID, strings.ToLower(c.Type.String()))
		}
		s, ok := this.workspace.Connection(opts.Storage)
		if !ok {
			return 0, errors.Unprocessable(StorageNotExist, "storage %d does not exist", opts.Storage)
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
	}

	// Validate the stream.
	if opts.Stream > 0 {
		if c.Type == state.MobileType || c.Type == state.ServerType || c.Type == state.WebsiteType {
			return 0, errors.BadRequest("connector %d cannot have a stream, it's a %s",
				c.ID, strings.ToLower(c.Type.String()))
		}
		s, ok := this.workspace.Connection(opts.Stream)
		if !ok {
			return 0, errors.Unprocessable(StreamNotExist, "stream %d does not exist", opts.Stream)
		}
		if s.Connector().Type != state.StreamType {
			return 0, errors.BadRequest("connection %d is not a stream", opts.Stream)
		}
		if ConnectionRole(s.Role) != role {
			if role == SourceRole {
				return 0, errors.BadRequest("stream %d is not a source", opts.Stream)
			}
			return 0, errors.BadRequest("stream %d is not a destination", opts.Stream)
		}
		n.Stream = opts.Stream
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
			if port, _ := strconv.Atoi(p); port <= 0 || port > 65535 {
				return 0, errors.BadRequest("website host %q is not valid", opts.WebsiteHost)
			}
		}
		n.WebsiteHost = opts.WebsiteHost
	}

	// Validate OAuth.
	if (opts.OAuth == nil) != (c.OAuth == nil) {
		if opts.OAuth == nil {
			return 0, errors.BadRequest("OAuth is required by connector %d", connector)
		}
		return 0, errors.BadRequest("connector %d does not support OAuth", connector)
	}

	// Set the resource. It can be an existent resource or a resource to be created.
	if opts.OAuth != nil {
		connection, err := _connector.RegisteredApp(c.Name).Open(context.Background(), &_connector.AppConfig{
			Role:         _connector.Role(role),
			ClientSecret: c.OAuth.ClientSecret,
			AccessToken:  opts.OAuth.AccessToken,
		})
		if err != nil {
			return 0, err
		}
		code, err := connection.Resource()
		if err != nil {
			return 0, err
		}
		n.Resource.Code = code
		resource, _ := this.workspace.ResourceByCode(code)
		if resource != nil {
			n.Resource.ID = resource.ID
		}
		if resource == nil || opts.OAuth.AccessToken != resource.AccessToken ||
			opts.OAuth.RefreshToken != resource.RefreshToken ||
			opts.OAuth.ExpiresIn != resource.ExpiresIn {
			n.Resource.AccessToken = opts.OAuth.AccessToken
			n.Resource.RefreshToken = opts.OAuth.RefreshToken
			n.Resource.ExpiresIn = opts.OAuth.ExpiresIn
		}
	}

	// Generate a connection identifier.
	var err error
	n.ID, err = generateConnectionID()
	if err != nil {
		return 0, err
	}

	// Generate a server key.
	var binaryKey []byte
	if c.Type == state.ServerType {
		n.Key, err = generateServerKey()
		if err != nil {
			return 0, err
		}
		binaryKey, err = decodeServerKey(n.Key)
		if err != nil {
			return 0, err
		}
	}

	err = this.db.Transaction(func(tx *postgres.Tx) error {
		if n.Resource.Code != "" {
			if n.Resource.ID == 0 {
				// Insert a new resource.
				err = tx.QueryRow("INSERT INTO resources (workspace, connector, code, access_token,"+
					" refresh_token, expires_in) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
					n.Workspace, connector, n.Resource.Code, n.Resource.AccessToken, n.Resource.RefreshToken, n.Resource.ExpiresIn).
					Scan(&n.Resource.ID)
			} else if n.Resource.AccessToken != "" {
				// Update the current resource.
				_, err = tx.Exec("UPDATE resources "+
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
		_, err = tx.Exec("INSERT INTO connections (id, workspace, name, type, role, connector, storage, stream,"+
			" resource, website_host) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, 0), NULLIF($8, 0), $9, $10)",
			n.ID, n.Workspace, n.Name, c.Type, n.Role, n.Connector, n.Storage, n.Stream, n.Resource.ID, n.WebsiteHost)
		if err != nil {
			if err != nil {
				if postgres.IsForeignKeyViolation(err) {
					switch postgres.ErrConstraintName(err) {
					case "connections_workspace_fkey":
						err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
					case "connections_connector_fkey":
						err = errors.Unprocessable(ConnectorNotExist, "connector %d does not exist", n.Connector)
					case "connections_storage_fkey":
						err = errors.Unprocessable(StorageNotExist, "storage %d does not exist", n.Storage)
					case "connections_stream_fkey":
						err = errors.Unprocessable(StreamNotExist, "stream %d does not exist", n.Stream)
					}
				}
			}
			return err
		}
		if binaryKey != nil {
			// Insert the server key.
			_, err = tx.Exec("INSERT INTO connections_keys (connection, value, creation_time) VALUES ($1, $2, $3)",
				n.ID, binaryKey, time.Now().UTC())
			if err != nil {
				return err
			}
		}
		return tx.Notify(n)
	})
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// Connection returns the connection with identifier id of the workspace ws.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
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
		db:          this.db,
		connection:  c,
		ID:          c.ID,
		Name:        c.Name,
		Type:        ConnectorType(conn.Type),
		Role:        ConnectionRole(c.Role),
		HasSettings: conn.HasSettings,
		LogoURL:     conn.LogoURL,
		Enabled:     c.Enabled,
		UsersQuery:  c.UsersQuery,
	}
	for _, t := range c.Mappings() {
		connection.Mappings = append(connection.Mappings, &MappingInfo{
			ID:         t.ID,
			In:         t.In,
			SourceCode: t.SourceCode,
			Out:        t.Out,
		})
	}
	if s, ok := c.Storage(); ok {
		connection.Storage = s.ID
	}
	if s, ok := c.Stream(); ok {
		connection.Stream = s.ID
	}
	if conn.OAuth != nil {
		connection.OAuthURL = conn.OAuth.URL
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
			db:          this.db,
			connection:  c,
			ID:          c.ID,
			Name:        c.Name,
			Type:        ConnectorType(conn.Type),
			Role:        ConnectionRole(c.Role),
			HasSettings: conn.HasSettings,
			LogoURL:     conn.LogoURL,
			Enabled:     c.Enabled,
			UsersQuery:  c.UsersQuery,
		}
		for _, t := range c.Mappings() {
			connection.Mappings = append(connection.Mappings, &MappingInfo{
				ID:         t.ID,
				In:         t.In,
				SourceCode: t.SourceCode,
				Out:        t.Out,
			})
		}
		if s, ok := c.Storage(); ok {
			connection.Storage = s.ID
		}
		if s, ok := c.Stream(); ok {
			connection.Stream = s.ID
		}
		if conn.OAuth != nil {
			connection.OAuthURL = conn.OAuth.URL
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
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE workspaces SET warehouse_type = $1, warehouse_settings = $2 WHERE id = $3"+
			" AND warehouse_type IS NULL",
			n.Type, string(n.Settings), n.Workspace)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			err = tx.QueryVoid("SELECT FROM workspaces WHERE id = $1", n.Workspace)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return err
			}
			return errors.Unprocessable(AlreadyConnected, "workspace %d is already connected to a data warehouse", ws.ID)
		}
		return tx.Notify(n)
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
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		var typ *state.WarehouseType
		err := tx.QueryRow("SELECT warehouse_type FROM workspaces WHERE id = $1", n.Workspace).Scan(&typ)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.NotFound("workspace %d does not exist", n.Workspace)
			}
			return err
		}
		if typ == nil {
			return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", n.Workspace)
		}
		_, err = tx.Exec("UPDATE workspaces SET warehouse_type = NULL, warehouse_settings = '', schemas = '' WHERE id = $1", n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(n)
	})
	return err
}

func (this *Workspace) EventListeners() *events.Listeners {
	return events.NewListeners(this.db, this.eventProcessor, this.workspace)
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
		// Check that the 'users' table contains the 'id' column.
		if table.Name == "users" {
			i := slices.IndexFunc(table.Columns, func(c *warehouses.Column) bool {
				return c.Name == "id"
			})
			if i == -1 {
				return errors.Unprocessable(InvalidSchemaTable, "'users' table has no 'id' column")
			}
			if c := table.Columns[i]; c.Type.PhysicalType() != types.PtInt {
				return errors.Unprocessable(InvalidSchemaTable, "column 'users.id' does not have type Int")
			} else if c.Type.Null() {
				return errors.Unprocessable(InvalidSchemaTable, "column 'users.id' must not be nullable")
			}
			table.Columns = slices.Delete(table.Columns, i, i+1)
		}
		if table.Name == "events" {
			continue // TODO(marco): skip the events table, for now
		}
		properties, err := propertiesOfColumns(table.Columns)
		if err, ok := err.(repeatedPropertyNameError); ok {
			return errors.Unprocessable(RepeatedPropertyName,
				"column %s.%s results in a repeated property named %s", table.Name, err.column, err.property)
		}
		schema := types.Object(properties).AsCustom(table.Name)
		n.Schemas[table.Name] = &schema
	}
	newRawSchemas, err := json.Marshal(n.Schemas)
	if err != nil {
		return fmt.Errorf("cannot marshal data warehouse schema for workspace %d: %s", ws.ID, err)
	}
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		var typ *state.WarehouseType
		var oldRawSchemas []byte
		err := tx.QueryRow("SELECT warehouse_type, schemas FROM workspaces WHERE id = $1", n.Workspace).Scan(&typ, &oldRawSchemas)
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
		_, err = tx.Exec("UPDATE workspaces SET schemas = $1 WHERE id = $2", newRawSchemas, n.Workspace)
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
		return tx.Notify(n)
	})
	return err
}

// Schema returns the schema, of the workspace with identifier id. If the
// schema does not exist, it returns an invalid schema.
func (this *Workspace) Schema(name string) types.Type {
	ws := this.workspace
	schema, ok := ws.Schemas[name]
	if !ok {
		return types.Type{}
	}
	return *schema
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
	err = this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE workspaces SET warehouse_settings = $1 WHERE id = $2 AND warehouse_type = $3",
			string(n.Settings), n.Workspace, n.Type)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			err = tx.QueryVoid("SELECT FROM workspaces WHERE id = $1", n.Workspace)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return err
			}
			return errors.Unprocessable(NoWarehouse, "workspace %d is not connected to a PostgreSQL data warehouse", ws.ID)
		}
		return tx.Notify(n)
	})
	return err
}

// Users returns the user schema and the users of the workspace. It returns
// the users in range [first,first+limit] with first >= 0 and 0 < limit <= 1000
// and only the given properties. properties cannot be empty.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore.
// If a property does not exist, it returns an errors.UnprocessableError error
// with code PropertyNotExist.
// If the warehouse failed, it returns an errors.UnprocessableError error with
// code WarehouseFailed.
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
			return types.Type{}, nil, errors.Unprocessable(PropertyNotExist, "property name %s does not exist", name)
		}
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

	// Create the schema to return, with only the required properties.
	queryProperties := make([]types.Property, len(properties))
	for i, name := range properties {
		queryProperties[i] = propertyByName[name]
	}
	schema := types.Object(queryProperties)

	users, err := ws.Warehouse.Users(context.Background(), schema, orderProperty, first, limit)
	if err != nil {
		if _, ok := err.(*warehouses.Error); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("cannot get users from the data warehouse of the workspace %d: %s", ws.ID, err)
			err = errors.Unprocessable(WarehouseFailed, "warehouse connection is failed")
		}
		return types.Type{}, nil, err
	}

	return schema, users, err
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

// A repeatedPropertyNameError value is returned from propertiesOfColumns when
// grouped columns result in a repeated property name.
type repeatedPropertyNameError struct {
	column, property string
}

func (err repeatedPropertyNameError) Error() string {
	return fmt.Sprintf("column %s results in a repeated property named %s", err.column, err.property)
}

// propertiesOfColumns returns the type properties of columns.
// Consecutive columns with a common prefix are grouped into a single object
// property. It could change the columns slice and the column names.
//
// Columns starting with an underscore ('_'), are grouped as if the underscore
// were not present but are not returned as properties.
//
// Grouping columns can result in properties with the same name. In this case,
// it returns a repeatedPropertyNameError error.
func propertiesOfColumns(columns []*warehouses.Column) ([]types.Property, error) {
	var properties []types.Property
	for i := 0; i < len(columns); i++ {
		c := columns[i]
		var property types.Property
		// group the columns with the same prefix.
		if prefix, n := columnsCommonPrefix(columns[i:]); prefix != "" {
			group := columns[i : i+n]
			i += n - 1
			for j := 0; j < n; j++ {
				column := group[j]
				// remove from the group the columns with an underscore prefix.
				if column.Name[0] == '_' {
					copy(group[j:], group[j+1:])
					j--
					n--
					continue
				}
				// remove the prefix from the column names.
				column.Name = strings.TrimPrefix(column.Name, prefix)
			}
			if n == 0 {
				continue
			}
			props, err := propertiesOfColumns(group[:n])
			if err != nil {
				return nil, err
			}
			property = types.Property{
				Name: strings.TrimSuffix(prefix, "_"),
				Type: types.Object(props),
			}
		} else {
			if c.Name[0] == '_' {
				continue
			}
			property = types.Property{
				Name:        c.Name,
				Description: c.Description,
				Type:        c.Type,
			}
			if !c.IsUpdatable {
				property.Role = types.SourceRole
			}
		}
		for _, p := range properties {
			if p.Name == property.Name {
				return nil, repeatedPropertyNameError{c.Name, p.Name}
			}
		}
		properties = append(properties, property)
	}
	return properties, nil
}

// columnsCommonPrefix returns the common prefix between the first column in
// columns and the successive consecutive columns. A common prefix, if exists,
// ends with an underscore character ('_').
//
// If a common prefix exists, it returns the prefix, and the number of
// consecutive columns having the common prefix, starting from the first
// column, otherwise it returns an empty string and zero.
//
// See TestColumnsCommonPrefix for some examples.
func columnsCommonPrefix(columns []*warehouses.Column) (string, int) {
	first := columns[0].Name
	if first[0] == '_' {
		first = first[1:]
	}
	var prefix string
	var n = len(columns)
Columns:
	for i := 0; i < len(first)-1; i++ {
		c := first[i]
		for k := 1; k < n; k++ {
			name := columns[k].Name
			if name[0] == '_' {
				name = name[1:]
			}
			if i < len(name)-1 && name[i] == c {
				// continue with the next column.
				if c == '_' {
					prefix = first[:i+1]
				}
				continue
			}
			if prefix == "" {
				// continue only with the previous columns.
				n = k
				continue Columns
			}
			// break and return the prefix.
			break Columns
		}
	}
	if prefix == "" {
		n = 0
	}
	return prefix, n
}
