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
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/types"
	"chichi/apis/warehouses"
	"chichi/connector/ui"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var (
	NoWarehouse          errors.Code = "NoWarehouse"
	OrderNotExist        errors.Code = "OrderNotExist"
	OrderTypeNotSortable errors.Code = "OrderTypeNotSortable"
	PropertyNotExist     errors.Code = "PropertyNotExist"
	WarehouseFailed      errors.Code = "WarehouseFailed"
	WarehouseTypeInvalid errors.Code = "WarehouseTypeInvalid"
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
	EventDataTypes *EventDataTypes
	EventListeners *EventListeners
	id             int
	account        *Account
	schema         struct {
		user  types.Schema
		group types.Schema
	}
	schemaSources struct {
		user  string
		group string
	}
	resources *resourcesState
}

// A WorkspaceInfo describes a workspace as returned by Get and List.
type WorkspaceInfo struct {
	ID int

	// Schema and SchemaSources are only returned by the Get method.
	Schema struct {
		User  types.Schema
		Group types.Schema
	}
	SchemaSources struct {
		User  string
		Group string
	}
}

// DisconnectWarehouse disconnects the warehouse of the workspace with
// identifier id. A disconnected warehouse is no longer used. If the workspace
// does not have a warehouse, it does nothing. To connect a warehouse, use
// ServerWarehouseUI.
//
// If the workspace does not exist, it returns an errors.NotFoundError error.
func (this *Workspaces) DisconnectWarehouse(id int) error {
	if id < 1 || id > math.MaxInt32 {
		return errors.BadRequest("workspace identifier %d is not valid", id)
	}
	n := disconnectWorkspaceWarehouseNotification{
		Workspace: this.id,
	}
	err := this.db.Transaction(func(tx *postgres.Tx) error {
		result, err := tx.Exec("UPDATE workspaces\nSET warehouse_type = NULL, warehouse_settings = ''\n"+
			"WHERE id = $1 AND warehouse_type IS NOT NULL", n.Workspace)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			err = tx.QueryVoid("SELECT FROM workspaces WHERE id = $1", id)
			if err == sql.ErrNoRows {
				err = errors.NotFound("workspace %d does not exist", id)
			}
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
	info.Schema.User = ws.schema.user
	info.Schema.Group = ws.schema.group
	info.SchemaSources.User = ws.schemaSources.user
	info.SchemaSources.Group = ws.schemaSources.group
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
	info.Schema.User = ws.schema.user
	info.Schema.Group = ws.schema.group
	info.SchemaSources.User = ws.schemaSources.user
	info.SchemaSources.Group = ws.schemaSources.group
	return &info
}

// SetUserSchema sets the user schema. schema cannot be longer than 16,777,215
// runes.
//
// If the workspace does not exist, it returns an errors.NotFoundError error.
// If schema is not valid, it returns an errors.UnprocessableError error with
// code InvalidSchema.
func (ws *Workspace) SetUserSchema(schema string) error {
	return ws.setSchema("user", schema)
}

// SetGroupSchema sets the group schema. schema cannot be longer than
// 16,777,215 runes.
//
// If the workspace does not exist, it returns an errors.NotFoundError error.
// If schema is not valid, it returns an errors.UnprocessableError error with
// code InvalidSchema.
func (ws *Workspace) SetGroupSchema(schema string) error {
	return ws.setSchema("group", schema)
}

// setSchema is called by SetUserSchema and SetGroupSchema to set a schema.
func (ws *Workspace) setSchema(name string, schema string) error {
	if utf8.RuneCountInString(schema) > rawSchemaMaxSize {
		return fmt.Errorf("schema is longer that 16,777,215 runes")
	}
	_, err := types.ParseSchema(strings.NewReader(schema), nil)
	if err != nil {
		return errors.Unprocessable(InvalidSchema, "schema is not valid: %w", err)
	}
	var n any
	var table string
	switch name {
	case "user":
		n = setWorkspaceUserSchemaNotification{
			Workspace: ws.id,
			Schema:    schema,
		}
		table = "user_schema"
	case "group":
		n = setWorkspaceGroupSchemaNotification{
			Workspace: ws.id,
			Schema:    schema,
		}
		table = "group_schema"
	}
	err = ws.db.Transaction(func(tx *postgres.Tx) error {
		_, err = tx.Exec("UPDATE workspaces SET "+table+" = $1 WHERE id = $2", schema, ws.id)
		if err != nil {
			if err == sql.ErrNoRows {
				err = errors.NotFound("workspace %d does not exist", ws.id)
			}
			return err
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
func (ws *Workspace) Users(properties []string, order string, first, limit int) (types.Schema, [][]any, error) {

	// Verify that the workspace has a data warehouse.
	if ws.warehouse == nil {
		return types.Schema{}, nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.id)
	}

	// Read the schema.
	schemaProperties := ws.schema.user.Properties()
	propertyByName := map[string]types.Property{}
	for _, p := range schemaProperties {
		propertyByName[p.Name] = p
	}

	// Validate the arguments.
	if len(properties) == 0 {
		return types.Schema{}, nil, errors.BadRequest("properties is empty")
	}
	for _, name := range properties {
		if _, ok := propertyByName[name]; !ok {
			if name == "" {
				return types.Schema{}, nil, errors.BadRequest("a property name is empty")
			}
			if !types.IsValidPropertyName(name) {
				return types.Schema{}, nil, errors.BadRequest("property name %q is not valid", name)
			}
			return types.Schema{}, nil, errors.Unprocessable(PropertyNotExist, "property name %s does not exist", name)
		}
	}
	var orderProperty types.Property
	if order != "" {
		if !types.IsValidPropertyName(order) {
			return types.Schema{}, nil, errors.BadRequest("order %q is not a valid property name", order)
		}
		var ok bool
		orderProperty, ok = ws.schema.user.Property(order)
		if !ok {
			return types.Schema{}, nil, errors.Unprocessable(OrderNotExist, "order %s does not exist in schema", order)
		}
		switch orderProperty.Type.PhysicalType() {
		case types.PtJSON, types.PtArray, types.PtObject, types.PtMap:
			return types.Schema{}, nil, errors.Unprocessable(OrderTypeNotSortable,
				"cannot sort by %s: property has type %s", order, orderProperty.Type)
		}
	}
	if first < 0 || first > maxInt32 {
		return types.Schema{}, nil, errors.BadRequest("first %d in not valid", first)
	}
	if limit < 1 || limit > 1000 {
		return types.Schema{}, nil, errors.BadRequest("limit %d is not valid", limit)
	}

	// Create the schema to return, with only the required properties.
	queryProperties := make([]types.Property, len(properties))
	for i, name := range properties {
		queryProperties[i] = propertyByName[name]
	}
	schema, err := types.SchemaOf(queryProperties)
	if err != nil {
		return types.Schema{}, nil, fmt.Errorf("cannot create a new schema from the user schema: %s", err)
	}

	users, err := ws.warehouse.Users(context.Background(), schema, orderProperty, first, limit)
	if err != nil {
		if _, ok := err.(*warehouses.Error); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("cannot get users from the data warehouse of the workspace %d: %s", ws.id, err)
			err = errors.Unprocessable(WarehouseFailed, "warehouse connection is failed")
		}
		return types.Schema{}, nil, err
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

// ServeWarehouseUI serves the warehouse UI of the workspace with identifier
// id where typ is the type of the warehouse. event is the event and values
// contains the form values in JSON format.
//
// If the workspace does not exist, it returns an errors.NotFoundError error.
// If the warehouse of the workspace does not have type typ, it returns an
// errors.UnprocessableError with code WarehouseTypeInvalid.
// If the event does not exist, it returns an errors.UnprocessableError error
// with code EventNotExist.
func (this *Workspaces) ServeWarehouseUI(id int, typ warehouses.Type, event string, values []byte) ([]byte, error) {

	if id < 1 || id > math.MaxInt32 {
		return nil, errors.BadRequest("workspace identifier %d is not valid", id)
	}
	if !warehouses.IsValidType(typ) {
		return nil, errors.BadRequest("warehouse type %d is not valid", typ)
	}

	ws, ok := this.state.Get(id)
	if !ok {
		return nil, errors.NotFound("workspace %d does not exist", id)
	}
	warehouse := ws.warehouse
	if warehouse == nil {
		var err error
		warehouse, err = warehouses.Open(typ, nil)
		if err != nil {
			return nil, err
		}
	}

	form, alert, settings, err := warehouse.ServeUI(context.Background(), event, values)
	if err != nil {
		if err == ui.ErrEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s warehouse", event, typ)
		}
		return nil, err
	}

	response, err := marshalUIFormAlert(form, alert, ui.BothRole)
	if err != nil {
		return nil, err
	}

	// Update the settings.
	if settings != nil && !bytes.Equal(settings, values) {
		n := setWorkspaceWarehouseNotification{
			Workspace: this.id,
			Warehouse: &notifiedWarehouse{
				Type:     typ,
				Settings: settings,
			},
		}
		err := this.db.Transaction(func(tx *postgres.Tx) error {
			result, err := tx.Exec("UPDATE workspaces SET warehouse_type = $1, warehouse_settings = $2 WHERE id = $3"+
				" AND ( warehouse_type = $1 OR warehouse_type IS NULL )",
				n.Warehouse.Type, n.Warehouse.Settings, n.Workspace)
			if err != nil {
				return err
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected == 0 {
				err = tx.QueryVoid("SELECT FROM workspaces WHERE id = $1", id)
				if err != nil {
					if err == sql.ErrNoRows {
						err = errors.NotFound("workspace %d does not exist", id)
					}
					return err
				}
				return errors.Unprocessable(WarehouseTypeInvalid, "warehouse type of workspace %d is not %s", id, typ)
			}
			return tx.Notify(n)
		})
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}
