//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"chichi/pkg/open2b/sql"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type WorkspaceAPI struct {
	workspace       int
	api             *AccountAPI
	myDB            *sql.DB
	chDB            chDriver.Conn
	DataSources     *DataSources
	Transformations *Transformations
}

type Schema string

// Schema returns the schema with the given name. name can be "user", "group"
// or "event". If the schema with the given name does not exist, it returns an
// empty schema.
func (ws *WorkspaceAPI) Schema(name string) (Schema, error) {
	var column string
	switch name {
	case "user":
		column = "userSchema"
	case "group":
		column = "groupSchema"
	case "event":
		column = "eventSchema"
	default:
		return "", fmt.Errorf("invalid schema name %q", name)
	}
	row, err := ws.myDB.Table("Workspaces").Get(sql.Where{"id": ws.workspace}, []any{column})
	if err != nil {
		return "", err
	}
	schema, _ := row[column].(string)
	return Schema(schema), nil
}

// SetSchema sets the schema with the given name. name can be "user", "group"
// or "event".
func (ws *WorkspaceAPI) SetSchema(name, schema string) error {
	var column string
	switch name {
	case "user":
		column = "userSchema"
	case "group":
		column = "groupSchema"
	case "event":
		column = "eventSchema"
	default:
		return fmt.Errorf("invalid schema name %q", name)
	}
	if !utf8.ValidString(schema) {
		return errors.New("invalid schema")
	}
	_, err := ws.myDB.Table("Workspaces").Update(sql.Set{column: schema}, sql.Where{"workspace": ws.workspace})
	return err
}
