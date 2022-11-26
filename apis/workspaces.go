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

	"chichi/apis/types"
	"chichi/pkg/open2b/sql"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type WorkspaceAPI struct {
	workspace       int
	api             *AccountAPI
	myDB            *sql.DB
	chDB            chDriver.Conn
	Connections     *Connections
	Transformations *Transformations
}

// Schema returns the schema with the given name. name can be "user", "group"
// or "event". If the schema with the given name does not exist, it returns an
// invalid schema.
func (ws *WorkspaceAPI) Schema(name string) (types.Schema, error) {
	var column string
	switch name {
	case "user":
		column = "userSchema"
	case "group":
		column = "groupSchema"
	case "event":
		column = "eventSchema"
	default:
		return types.Schema{}, fmt.Errorf("invalid schema name %q", name)
	}
	var rawSchema string
	err := ws.myDB.QueryRow("SELECT `"+column+"` FROM `workspaces` WHERE `id` = ?", ws.workspace).Scan(&rawSchema)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.Schema{}, errors.New("workspace does not exist anymore")
		}
		return types.Schema{}, err
	}
	if len(rawSchema) == 0 {
		return types.Schema{}, nil
	}
	schema, err := types.ParseSchema(rawSchema, nil)
	if err != nil {
		return types.Schema{}, fmt.Errorf("cannot unmarshal schema of workspace %d: %s", ws.workspace, err)
	}
	return schema, nil
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
