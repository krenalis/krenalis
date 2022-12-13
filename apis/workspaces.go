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
	"strconv"
	"strings"
	"sync"
	"time"

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
// Returns the errWorkspaceNotFound error if the workspace does not exist.
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
	EventListeners  *EventListeners
	Transformations *Transformations
	mu              sync.Mutex // for userSchema, groupSchema and eventSchema fields.
	id              int
	account         *Account
	userSchema      string
	groupSchema     string
	eventSchema     string
}

// As returns the workspace with identifier id.
// Returns an error is the workspace does not exist.
func (this *Workspaces) As(id int) (*Workspace, error) {
	return this.get(id)
}

// Schema returns the schema with the given name. name can be "user", "group"
// or "event". If the schema with the given name does not exist, it returns an
// empty string.
func (ws *Workspace) Schema(name string) (string, error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	switch name {
	case "user":
		return ws.userSchema, nil
	case "group":
		return ws.groupSchema, nil
	case "event":
		return ws.eventSchema, nil
	default:
		return "", fmt.Errorf("invalid schema name %q", name)
	}
}

// An InvalidSchemaSyntaxError error indicates that a schema has an invalid
// syntax.
type InvalidSchemaSyntaxError struct {
	Err error
}

func (err *InvalidSchemaSyntaxError) Error() string {
	return fmt.Sprintf("schema is not valid: %s", err.Err.Error())
}

// SetSchema sets the schema with the given name. name can be "user", "group"
// or "event". If the schema has a syntax error, it returns an
// InvalidSchemaSyntaxError error.
func (ws *Workspace) SetSchema(name, schema string) error {
	var column string
	switch name {
	case "user":
		column = "user_schema"
	case "group":
		column = "group_schema"
	case "event":
		column = "event_schema"
	default:
		return fmt.Errorf("invalid schema name %q", name)
	}
	_, err := types.ParseSchema(strings.NewReader(schema), nil)
	if err != nil {
		return &InvalidSchemaSyntaxError{err}
	}
	_, err = ws.db.Exec("UPDATE workspaces SET "+column+" = $1 WHERE id = $2", schema, ws.id)
	if err != nil {
		return err
	}
	ws.mu.Lock()
	switch name {
	case "user":
		ws.userSchema = schema
	case "group":
		ws.groupSchema = schema
	case "event":
		ws.eventSchema = schema
	}
	ws.mu.Unlock()
	return nil
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
	ws.mu.Lock()
	rawSchema := ws.userSchema
	ws.mu.Unlock()
	schema, err := types.ParseSchema(strings.NewReader(rawSchema), nil)
	if err != nil {
		return types.Schema{}, nil, err
	}
	schemaProperties := schema.Properties()
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
	err = ws.db.QueryScan(query.String(), func(rows *postgres.Rows) error {
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
	schema, err = types.SchemaOf(returnedProperties)
	if err != nil {
		return types.Schema{}, nil, fmt.Errorf("cannot create a new schema from the user schema: %s", err)
	}

	return schema, users, err
}
