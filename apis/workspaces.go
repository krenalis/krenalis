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
	AlreadyConnected     errors.Code = "AlreadyConnected"
	ConnectionFailed     errors.Code = "ConnectionFailed"
	InvalidSettings      errors.Code = "InvalidSettings"
	NoWarehouse          errors.Code = "NoWarehouse"
	NotConnected         errors.Code = "NotConnected"
	OrderNotExist        errors.Code = "OrderNotExist"
	OrderTypeNotSortable errors.Code = "OrderTypeNotSortable"
	PropertyNotExist     errors.Code = "PropertyNotExist"
	RepeatedPropertyName errors.Code = "RepeatedPropertyName"
	WarehouseFailed      errors.Code = "WarehouseFailed"
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
	schemas        map[string]*types.Type
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
		_, err = tx.Exec("UPDATE workspaces SET warehouse_type = NULL, warehouse_settings = '', schemas = '' WHERE id = $1", n.Workspace)
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
	schema, ok := ws.schemas[name]
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
	n := setWorkspaceSchemasNotification{
		Workspace: ws.id,
		Schemas:   map[string]*types.Type{},
	}
	for _, table := range tables {
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
		return fmt.Errorf("cannot marshal data warehouse schema for workspace %d: %s", ws.id, err)
	}
	err = ws.db.Transaction(func(tx *postgres.Tx) error {
		var typ *WarehouseType
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
	if typ, ok := ws.schemas["users"]; ok {
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
