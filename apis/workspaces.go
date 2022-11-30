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
// empty string.
func (ws *WorkspaceAPI) Schema(name string) (string, error) {
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
	var schema string
	err := ws.myDB.QueryRow("SELECT `"+column+"` FROM `workspaces` WHERE `id` = ?", ws.workspace).Scan(&schema)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("workspace does not exist anymore")
		}
		return "", err
	}
	return schema, nil
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
	_, err := types.ParseSchema(schema, nil)
	if err != nil {
		return &InvalidSchemaSyntaxError{err}
	}
	_, err = ws.myDB.Table("Workspaces").Update(sql.Set{column: schema}, sql.Where{"id": ws.workspace})
	return err
}
