//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/types"
	"chichi/apis/warehouses"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var (
	NoSchema             errors.Code = "NoSchema"
	NoWarehouse          errors.Code = "NoWarehouse"
	NotConnected         errors.Code = "NotConnected"
	ConnectionFailed     errors.Code = "ConnectionFailed"
	OrderNotExist        errors.Code = "OrderNotExist"
	OrderTypeNotSortable errors.Code = "OrderTypeNotSortable"
	PropertyNotExist     errors.Code = "PropertyNotExist"
	AlreadyConnected     errors.Code = "AlreadyConnected"
	WarehouseFailed      errors.Code = "WarehouseFailed"
	InvalidSettings      errors.Code = "InvalidSettings"
)

type Workspaces struct {
	*Account
	state *workspacesState
}

// newWorkspaces returns a new *Workspaces value.
func newWorkspaces(account *Account, state *workspacesState) *Workspaces {
	return &Workspaces{Account: account, state: state}
}

// Workspace represents a workspace.
type Workspace struct {
	db             *postgres.DB
	chDB           chDriver.Conn
	warehouse      warehouses.Warehouse
	Connections    *Connections
	EventTypes     *EventTypes
	DataTypes      *DataTypes
	EventListeners *EventListeners
	id             int
	account        *Account
	resources      *resourcesState
}

// A WorkspaceInfo describes a workspace as returned by Get and List.
type WorkspaceInfo struct {
	ID int
}

// WarehouseSettings is the interface implemented by data warehouse settings.
type WarehouseSettings interface {
	warehouseType() warehouses.Type
}

// PostgreSQLSettings are the settings used to connect to a PostgreSQL data
// warehouse. It implements the WarehouseSettings interface.
type PostgreSQLSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Schema   string
}

func (s *PostgreSQLSettings) warehouseType() warehouses.Type {
	return warehouses.PostgreSQL
}

// ConnectWarehouse connects a data warehouse, with the given settings, to the
// workspace. It also creates the tables in the connected data warehouse.
//
// It returns an errors.NotFoundError error, if the workspace does not exist,
// and it returns an errors.UnprocessableError error with code
//   - NoSchema, if the workspace does not have the user schema.
//   - AlreadyConnected, if the workspace is already connected to a data
//     warehouse.
//   - InvalidSettings, if the settings are not valid.
//   - ConnectionFailed, if the connection fails.
func (ws *Workspace) ConnectWarehouse(settings WarehouseSettings) error {
	userType, ok := ws.DataTypes.state.Get("user")
	if !ok {
		return errors.Unprocessable(NoSchema, "workspace %d does not have the user schema", ws.id)
	}
	if ws.warehouse != nil {
		return errors.Unprocessable(AlreadyConnected, "workspace %d is already connected to a data warehouse", ws.id)
	}
	s := settings.(*PostgreSQLSettings)
	ps := &warehouses.PostgreSQLSettings{
		Host:     s.Host,
		Port:     s.Port,
		Username: s.Username,
		Password: s.Password,
		Database: s.Database,
		Schema:   s.Schema,
	}
	warehouse := warehouses.OpenPostgres(ps)
	err := warehouse.Validate()
	if err != nil {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
	}
	err = warehouse.Ping(context.Background())
	if err != nil {
		return errors.Unprocessable(ConnectionFailed, "cannot connect to the data warehouse: %w", err)
	}
	rawSettings, err := json.Marshal(s)
	if err != nil {
		return err
	}
	n := setWorkspaceWarehouseNotification{
		Workspace: ws.id,
		Warehouse: &notifiedWarehouse{
			Type:     warehouses.PostgreSQL,
			Settings: rawSettings,
		},
	}
	err = ws.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE workspaces SET warehouse_type = $1, warehouse_settings = $2 WHERE id = $3"+
			" AND warehouse_type IS NULL",
			n.Warehouse.Type, string(n.Warehouse.Settings), n.Workspace)
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
			return errors.Unprocessable(AlreadyConnected, "workspace %d is already connected to a data warehouse", ws.id)
		}
		err = warehouse.CreateTables(context.Background(), userType.typ)
		if err != nil {
			return err
		}
		return tx.Notify(n)
	})
	return err
}

// DisconnectWarehouse disconnects the data warehouse. If deleteTables is true,
// it also deletes the tables from the data warehouse.
//
// If the workspace does not exist, it returns an errors.NotFoundError error.
// If the workspace is not connected to a data warehouse, it returns an
// errors.UnprocessableError error with code NotConnected.
func (ws *Workspace) DisconnectWarehouse(deleteTables bool) error {
	if ws.warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.id)
	}
	n := setWorkspaceWarehouseNotification{
		Workspace: ws.id,
		Warehouse: nil,
	}
	err := ws.db.Transaction(func(tx *postgres.Tx) error {
		var typ *warehouses.Type
		var settings []byte
		err := tx.QueryRow("SELECT warehouse_type, warehouse_settings FROM workspaces WHERE id = $1",
			n.Workspace).Scan(&typ, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return errors.NotFound("workspace %d does not exist", n.Workspace)
			}
			return err
		}
		if typ == nil {
			return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", n.Workspace)
		}
		if deleteTables {
			var s warehouses.PostgreSQLSettings
			err := json.Unmarshal(settings, &s)
			if err != nil {
				return err
			}
			warehouse := warehouses.OpenPostgres(&s)
			var userSchema types.Type
			if dataType, ok := ws.DataTypes.state.Get("user"); ok {
				userSchema = dataType.typ
			}
			// TODO(marco): consider whether there is a better solution than removing the tables at this time.
			err = warehouse.DropTables(context.Background(), userSchema)
			if err != nil {
				return err
			}
		}
		_, err = tx.Exec("UPDATE workspaces SET warehouse_type = NULL, warehouse_settings = '' WHERE id = $1", n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(n)
	})
	return err
}

// Get returns a WorkspaceInfo describing the workspace with identifier id.
// If the workspace does not exist, it returns an errors.NotFoundError error.
func (this *Workspaces) Get(id int) (*WorkspaceInfo, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("workspace identifier %d is not valid", id)
	}
	ws, ok := this.state.Get(id)
	if !ok {
		return nil, errors.NotFound("workspace %d does not exist", id)
	}
	info := WorkspaceInfo{ID: ws.id}
	return &info, nil
}

// As returns the workspace with identifier id.
// Returns an error if the workspace does not exist.
func (this *Workspaces) As(id int) (*Workspace, error) {
	ws, ok := this.state.Get(id)
	if !ok {
		return nil, errors.NotFound("workspace %d does not exist", id)
	}
	return ws, nil
}

// Info returns a WorkspaceInfo describing the workspace.
func (ws *Workspace) Info() *WorkspaceInfo {
	info := WorkspaceInfo{ID: ws.id}
	return &info
}

// SetWarehouse sets the settings used to connect to the data warehouse.
//
// It returns an errors.NotFoundError error, if the workspace does not exist,
// and it returns an errors.UnprocessableError error with code
//   - NotConnected, if the workspace is not connected to a data warehouse.
//   - InvalidSettings, if the settings are not valid.
//   - ConnectionFailed, if the connection fails.
func (ws *Workspace) SetWarehouse(settings WarehouseSettings) error {
	if ws.warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.id)
	}
	if typ := ws.warehouse.Type(); typ != warehouses.PostgreSQL {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", fmt.Errorf(
			"workspace %d is connected to a %s data warehouse, but settings are for a %s data warehouse",
			ws.id, typ, warehouses.PostgreSQL))
	}
	s := settings.(*PostgreSQLSettings)
	ps := &warehouses.PostgreSQLSettings{
		Host:     s.Host,
		Port:     s.Port,
		Username: s.Username,
		Password: s.Password,
		Database: s.Database,
		Schema:   s.Schema,
	}
	warehouse := warehouses.OpenPostgres(ps)
	err := warehouse.Validate()
	if err != nil {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
	}
	err = warehouse.Ping(context.Background())
	if err != nil {
		return errors.Unprocessable(ConnectionFailed, "cannot connect to the data warehouse: %w", err)
	}
	rawSettings, err := json.Marshal(s)
	if err != nil {
		return err
	}
	n := setWorkspaceWarehouseNotification{
		Workspace: ws.id,
		Warehouse: &notifiedWarehouse{
			Type:     warehouses.PostgreSQL,
			Settings: rawSettings,
		},
	}
	err = ws.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE workspaces SET warehouse_settings = $1 WHERE id = $2 AND warehouse_type = $3",
			string(n.Warehouse.Settings), n.Workspace, n.Warehouse.Type)
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
			return errors.Unprocessable(NoWarehouse, "workspace %d is not connected to a PostgreSQL data warehouse", ws.id)
		}
		return tx.Notify(n)
	})
	return err
}

// Users returns the user schema and the users, with only given properties, in
// range [first,first+limit] with first >= 0 and 0 < limit <= 1000. properties
// cannot be empty.
//
// If a property does not exist, it returns an errors.UnprocessableError error
// with code PropertyNotExist.
// If the warehouse failed, it returns an errors.UnprocessableError error with
// code WarehouseFailed.
func (ws *Workspace) Users(properties []string, order string, first, limit int) (types.Type, [][]any, error) {

	// Verify that the workspace has a data warehouse.
	if ws.warehouse == nil {
		return types.Type{}, nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.id)
	}

	// Read the schema.
	var schemaProperties []types.ObjectProperty
	if dataType, ok := ws.DataTypes.state.Get("user"); ok {
		schemaProperties = dataType.typ.Properties()
	}
	propertyByName := map[string]types.ObjectProperty{}
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
	var orderProperty types.ObjectProperty
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
	queryProperties := make([]types.ObjectProperty, len(properties))
	for i, name := range properties {
		queryProperties[i] = propertyByName[name]
	}
	schema := types.Object(queryProperties)

	users, err := ws.warehouse.Users(context.Background(), schema, orderProperty, first, limit)
	if err != nil {
		if _, ok := err.(*warehouses.Error); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("cannot get users from the data warehouse of the workspace %d: %s", ws.id, err)
			err = errors.Unprocessable(WarehouseFailed, "warehouse connection is failed")
		}
		return types.Type{}, nil, err
	}

	return schema, users, err
}

// List returns a list of WorkspaceInfo describing all workspaces.
func (this *Workspaces) List() []*WorkspaceInfo {
	workspaces := this.state.List()
	infos := make([]*WorkspaceInfo, len(workspaces))
	for i, c := range workspaces {
		info := WorkspaceInfo{
			ID: c.id,
		}
		infos[i] = &info
	}
	sort.Slice(infos, func(i, j int) bool {
		a, b := infos[i], infos[j]
		return a.ID < b.ID
	})
	return infos
}
