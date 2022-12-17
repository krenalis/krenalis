//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chichi/apis/postgres"
	"chichi/apis/types"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/shopspring/decimal"
)

type Workspaces struct {
	*Account
	state workspacesState
}

type workspacesState struct {
	sync.Mutex
	ids map[int]*Workspace
}

var errWorkspaceNotFound = errors.New("workspace does not exist")

// get returns the workspace with identifier id.
// Returns the ErrWorkspaceNotFound error if the workspace does not exist.
func (this *Workspaces) get(id int) (*Workspace, error) {
	this.state.Lock()
	w, ok := this.state.ids[id]
	this.state.Unlock()
	if ok {
		return w, nil
	}
	return nil, errWorkspaceNotFound
}

// newWorkspaces returns a new *Workspaces value.
func newWorkspaces(account *Account) *Workspaces {
	return &Workspaces{Account: account, state: workspacesState{ids: map[int]*Workspace{}}}
}

// Workspace represents a workspace.
type Workspace struct {
	db              *postgres.DB
	chDB            chDriver.Conn
	Connections     *Connections
	EventTypes      *EventTypes
	EventDataTypes  *EventDataTypes
	EventListeners  *EventListeners
	Transformations *Transformations
	id              int
	account         *Account
	schema          struct {
		user  types.Schema
		group types.Schema
		event types.Schema
	}
	schemaSources struct {
		user  string
		group string
		event string
	}
}

// A WorkspaceInfo describes a workspace as returned by Get and List.
type WorkspaceInfo struct {
	ID int

	// Schema and SchemaSources are only returned by the Get method.
	Schema struct {
		User  types.Schema
		Group types.Schema
		Event types.Schema
	}
	SchemaSources struct {
		User  string
		Group string
		Event string
	}
}

// An ErrWorkspaceNotFound error is returned by Get if the workspace does not
// exist.
var ErrWorkspaceNotFound = errors.New("workspace does not exist")

// Get returns a WorkspaceInfo describing the workspace with identifier id.
// Returns the ErrWorkspaceNotFound error if the workspace does not exist.
func (this *Workspaces) Get(id int) (*WorkspaceInfo, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.New("invalid workspace identifier")
	}
	ws, err := this.get(id)
	if err != nil {
		return nil, ErrWorkspaceNotFound
	}
	info := WorkspaceInfo{ID: ws.id}
	info.Schema.User = ws.schema.user
	info.Schema.Group = ws.schema.group
	info.Schema.Event = ws.schema.event
	info.SchemaSources.User = ws.schemaSources.user
	info.SchemaSources.Group = ws.schemaSources.group
	info.SchemaSources.Event = ws.schemaSources.event
	return &info, nil
}

// As returns the workspace with identifier id.
// Returns an error if the workspace does not exist.
func (this *Workspaces) As(id int) (*Workspace, error) {
	return this.get(id)
}

// An InvalidSchema error indicates that a schema is not valid.
type InvalidSchema struct {
	Err error
}

func (err *InvalidSchema) Error() string {
	return fmt.Sprintf("schema is not valid: %s", err.Err.Error())
}

// Info returns a WorkspaceInfo describing the workspace.
func (ws *Workspace) Info() *WorkspaceInfo {
	info := WorkspaceInfo{ID: ws.id}
	info.Schema.User = ws.schema.user
	info.Schema.Group = ws.schema.group
	info.Schema.Event = ws.schema.event
	info.SchemaSources.User = ws.schemaSources.user
	info.SchemaSources.Group = ws.schemaSources.group
	info.SchemaSources.Event = ws.schemaSources.event
	return &info
}

// SetUserSchema sets the user schema. schema must be valid and cannot be
// longer than 2^24-1 runes. If schema is not valid, it returns an
// InvalidSchema error.
func (ws *Workspace) SetUserSchema(schema string) error {
	return ws.setSchema("user", schema)
}

// SetGroupSchema sets the group schema. schema must be valid and cannot be
// longer than 2^24-1 runes. If schema is not valid, it returns an
// // InvalidSchema error.
func (ws *Workspace) SetGroupSchema(schema string) error {
	return ws.setSchema("group", schema)
}

// setSchema is called by SetUserSchema and SetGroupSchema to set a schema.
func (ws *Workspace) setSchema(name string, schema string) error {
	if utf8.RuneCountInString(schema) > rawSchemaMaxSize {
		return fmt.Errorf("schema is too longer")
	}
	_, err := types.ParseSchema(strings.NewReader(schema), nil)
	if err != nil {
		return &InvalidSchema{err}
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
			return err
		}
		return tx.Notify(n)
	})
	return err
}

// A PropertyNotFoundError is returned by the (*Workspace).Users method if a
// property does not exist.
type PropertyNotFoundError struct {
	Name string
}

func (err *PropertyNotFoundError) Error() string {
	return fmt.Sprintf("property %q does not exist", err.Name)
}

// Users returns the user schema and the users, with only given properties, in
// range [first,first+limit] with first >= 0 and 0 < limit <= 1000.
//
// If a property does not exist, it returns a PropertyNotFoundError error.
func (ws *Workspace) Users(properties []string, first, limit int) (types.Schema, [][]any, error) {

	if len(properties) == 0 {
		return types.Schema{}, nil, errors.New("properties cannot be empty")
	}
	if first < 0 || first > maxInt32 {
		return types.Schema{}, nil, errors.New("invalid first")
	}
	if limit < 1 || limit > 1000 {
		return types.Schema{}, nil, errors.New("invalid limit")
	}

	// Read the schema.
	schemaProperties := ws.schema.user.Properties()
	propertyByName := map[string]types.Property{}
	for _, p := range schemaProperties {
		propertyByName[p.Name] = p
	}

	// Build the query.
	var query bytes.Buffer
	query.WriteString("SELECT ")
	for i, name := range properties {
		if _, ok := propertyByName[name]; !ok {
			return types.Schema{}, nil, &PropertyNotFoundError{name}
		}
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteString(postgres.QuoteIdent(name))
	}
	query.WriteString(" FROM warehouse_users LIMIT ")
	query.WriteString(strconv.Itoa(limit))
	if first > 0 {
		query.WriteString(" OFFSET ")
		query.WriteString(strconv.Itoa(first))
	}

	// Execute the query.
	var users [][]any
	err := ws.db.QueryScan(query.String(), func(rows *postgres.Rows) error {
		var err error
		for rows.Next() {
			user := make([]any, len(properties))
			for i := range user {
				name := properties[i]
				typ := propertyByName[name].Type
				switch typ.PhysicalType() {
				case types.PtBoolean:
					var v bool
					user[i] = &v
				case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
					var v int
					user[i] = &v
				case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
					var v uint
					user[i] = &v
				case types.PtFloat, types.PtFloat32:
					var v float64
					user[i] = &v
				case types.PtDecimal:
					var v decimal.Decimal
					user[i] = &v
				case types.PtDateTime, types.PtDate:
					var v time.Time
					user[i] = &v
				case types.PtTime, types.PtYear:
					var v int
					user[i] = &v
				case types.PtUUID, types.PtJSON, types.PtText, types.PtArray, types.PtObject, types.PtMap:
					var v string
					user[i] = &v
				}
			}
			if err = rows.Scan(user...); err != nil {
				return err
			}
			users = append(users, user)
		}
		return nil
	})
	if err != nil {
		return types.Schema{}, nil, err
	}
	if users == nil {
		users = [][]any{}
	}

	// Create the schema to return, with only the required properties.
	returnedProperties := make([]types.Property, len(properties))
	for i, name := range properties {
		returnedProperties[i] = propertyByName[name]
	}
	schema, err := types.SchemaOf(returnedProperties)
	if err != nil {
		return types.Schema{}, nil, fmt.Errorf("cannot create a new schema from the user schema: %s", err)
	}

	return schema, users, err
}

// list returns all the workspaces.
func (this *Workspaces) list() []*Workspace {
	this.state.Lock()
	workspaces := make([]*Workspace, len(this.state.ids))
	i := 0
	for _, c := range this.state.ids {
		workspaces[i] = c
		i++
	}
	this.state.Unlock()
	return workspaces
}

// List returns a list of WorkspaceInfo describing all workspaces.
func (this *Workspaces) List() []*WorkspaceInfo {
	workspaces := this.list()
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
