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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/types"
	"chichi/apis/warehouses"
	"chichi/apis/warehouses/clickhouse"
	"chichi/apis/warehouses/postgresql"
)

var (
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
	warehouse      warehouses.Warehouse
	schema         map[string]*types.Type
	Connections    *Connections
	EventListeners *EventListeners
	id             int
	account        *Account
	resources      *resourcesState
}

// A WorkspaceInfo describes a workspace as returned by Get and List.
type WorkspaceInfo struct {
	ID     int
	Schema map[string]types.Type
}

// ConnectWarehouse connects a data warehouse, with the given settings, to the
// workspace. It also creates the tables in the connected data warehouse.
//
// It returns an errors.NotFoundError error, if the workspace does not exist,
// and it returns an errors.UnprocessableError error with code
//   - AlreadyConnected, if the workspace is already connected to a data
//     warehouse.
//   - InvalidSettings, if the settings are not valid.
//   - ConnectionFailed, if the connection fails.
func (ws *Workspace) ConnectWarehouse(typ WarehouseType, settings []byte) error {
	if ws.warehouse != nil {
		return errors.Unprocessable(AlreadyConnected, "workspace %d is already connected to a data warehouse", ws.id)
	}
	warehouse, err := openWarehouse(typ, settings)
	if err != nil {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
	}
	err = warehouse.Ping(context.Background())
	if err != nil {
		return errors.Unprocessable(ConnectionFailed, "cannot connect to the data warehouse: %w", err)
	}
	n := setWorkspaceWarehouseNotification{
		Workspace: ws.id,
		Warehouse: &notifiedWarehouse{
			Type:     typ,
			Settings: warehouse.Settings(),
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
		return tx.Notify(n)
	})
	return err
}

// DisconnectWarehouse disconnects the data warehouse.
//
// If the workspace does not exist, it returns an errors.NotFoundError error.
// If the workspace is not connected to a data warehouse, it returns an
// errors.UnprocessableError error with code NotConnected.
func (ws *Workspace) DisconnectWarehouse() error {
	if ws.warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.id)
	}
	n := setWorkspaceWarehouseNotification{
		Workspace: ws.id,
		Warehouse: nil,
	}
	err := ws.db.Transaction(func(tx *postgres.Tx) error {
		var typ *WarehouseType
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
		_, err = tx.Exec("UPDATE workspaces SET warehouse_type = NULL, warehouse_settings = '', schema = '' WHERE id = $1", n.Workspace)
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

// InitWarehouse initializes the connected data warehouse by creating the
// supporting tables.
//
// It returns an errors.UnprocessableError error with code NotConnected, if the
// workspace is not connected to a data warehouse.
func (ws *Workspace) InitWarehouse() error {
	if ws.warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.id)
	}
	return ws.warehouse.Init(context.Background())
}

// Schema returns the schema with the given name.
func (ws *Workspace) Schema(name string) (types.Type, bool) {
	schema, ok := ws.schema[name]
	if ok {
		return *schema, true
	}
	return types.Type{}, false
}

// SetWarehouse sets the settings used to connect to the data warehouse.
//
// It returns an errors.NotFoundError error, if the workspace does not exist,
// and it returns an errors.UnprocessableError error with code
//   - NotConnected, if the workspace is not connected to a data warehouse.
//   - InvalidSettings, if the settings are not valid.
//   - ConnectionFailed, if the connection fails.
func (ws *Workspace) SetWarehouse(typ WarehouseType, settings []byte) error {
	if ws.warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.id)
	}
	if typ != typeOfWarehouse(ws.warehouse) {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", fmt.Errorf(
			"workspace %d is connected to a %s data warehouse, but settings are for a %s data warehouse",
			ws.id, typeOfWarehouse(ws.warehouse), typ))
	}
	warehouse, err := openWarehouse(typ, settings)
	if err != nil {
		return errors.Unprocessable(InvalidSettings, "settings are not valid: %w", err)
	}
	err = warehouse.Ping(context.Background())
	if err != nil {
		return errors.Unprocessable(ConnectionFailed, "cannot connect to the data warehouse: %w", err)
	}
	n := setWorkspaceWarehouseNotification{
		Workspace: ws.id,
		Warehouse: &notifiedWarehouse{
			Type:     typ,
			Settings: warehouse.Settings(),
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

// ReloadSchema reloads the schema of the workspace.
//
// It returns an errors.NotFoundError error, if the workspace does not exist,
// and it returns an errors.UnprocessableError error with code
//   - NotConnected, if the workspace is not connected to a data warehouse.
//   - WarehouseFailed, if the connection to the data warehouse failed.
func (ws *Workspace) ReloadSchema() error {
	if ws.warehouse == nil {
		return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", ws.id)
	}
	tables, err := ws.warehouse.Tables(context.Background())
	if err != nil {
		if err, ok := err.(*warehouses.Error); ok {
			return errors.Unprocessable(WarehouseFailed, "data warehouse has returned an error: %w", err.Err)
		}
		return err
	}
	n := setWorkspaceSchemaNotification{
		Workspace: ws.id,
		Schema:    map[string]*types.Type{},
	}
	for _, table := range tables {
		name := table.Name
		if name != "users" && name != "groups" && name != "events" &&
			(name == "users_" || !strings.HasPrefix(name, "users_")) &&
			(name == "groups_" || !strings.HasPrefix(name, "groups_")) &&
			(name == "events_" || !strings.HasPrefix(name, "events_")) {
			continue
		}
		var properties []types.Property
		for _, c := range table.Columns {
			property := types.Property{
				Name:        c.Name,
				Description: c.Description,
				Role:        types.BothRole,
				Nullable:    c.IsNullable,
				Type:        c.Type,
			}
			if !c.IsUpdatable {
				property.Role = types.SourceRole
			}
			properties = append(properties, property)
		}
		typ := types.Object(properties).AsCustom(name)
		n.Schema[name] = &typ
	}
	newRawSchema, err := json.Marshal(n.Schema)
	if err != nil {
		return fmt.Errorf("cannot marshal data warehouse schema for workspace %d: %s", ws.id, err)
	}
	err = ws.db.Transaction(func(tx *postgres.Tx) error {
		var typ *WarehouseType
		var oldRawSchema []byte
		err := tx.QueryRow("SELECT warehouse_type, schema FROM workspaces WHERE id = $1", n.Workspace).Scan(&typ, &oldRawSchema)
		if err != nil {
			if err == sql.ErrNoRows {
				err = errors.NotFound("workspace %d does not exist", n.Workspace)
			}
			return err
		}
		if typ == nil {
			return errors.Unprocessable(NotConnected, "workspace %d is not connected to a data warehouse", n.Workspace)
		}
		if bytes.Equal(newRawSchema, oldRawSchema) {
			return nil
		}
		_, err = tx.Exec("UPDATE workspaces SET schema = $1 WHERE id = $2", newRawSchema, n.Workspace)
		if err != nil {
			return err
		}
		if len(oldRawSchema) > 0 {
			var oldSchema map[string]*types.Type
			err = json.Unmarshal(oldRawSchema, &oldSchema)
			if err != nil {
				return fmt.Errorf("cannot parse schema of workspace %d: %s", n.Workspace, err)
			}
			for name, t := range n.Schema {
				if t2, ok := oldSchema[name]; ok && t.EqualTo(*t2) {
					n.Schema[name] = nil
				}
			}
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
	var schemaProperties []types.Property
	if typ, ok := ws.schema["users"]; ok {
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

// openWarehouse opens a data warehouse with the given type and settings.
// It returns an error if typ or settings are not valid.
func openWarehouse(typ WarehouseType, settings []byte) (warehouses.Warehouse, error) {
	switch typ {
	case BigQuery, Redshift, Snowflake:
		return nil, fmt.Errorf("warehouse type %s is not yet supported", typ)
	case PostgreSQL:
		return postgresql.OpenPostgres(settings)
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

// Scan implements the sql.Scanner interface.
func (typ *WarehouseType) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an WarehouseType value", src)
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

// String returns the string representation of typ.
// It panics if typ is not a valid WarehouseType value.
func (typ WarehouseType) String() string {
	s, err := typ.Value()
	if err != nil {
		panic("invalid warehouse type")
	}
	return s.(string)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (typ *WarehouseType) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var s any
	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a WarehouseType value: %s", err)
	}
	return typ.Scan(s)
}

// Value implements driver.Valuer interface.
// It returns an error if typ is not a valid WarehouseType.
func (typ WarehouseType) Value() (driver.Value, error) {
	switch typ {
	case BigQuery:
		return "BigQuery", nil
	case ClickHouse:
		return "ClickHouse", nil
	case PostgreSQL:
		return "PostgreSQL", nil
	case Redshift:
		return "Redshift", nil
	case Snowflake:
		return "Snowflake", nil
	}
	return nil, fmt.Errorf("not a valid WarehouseType: %d", typ)
}
