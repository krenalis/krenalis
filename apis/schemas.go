//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"fmt"

	"chichi/pkg/open2b/sql"
)

type Schemas struct {
	*APIs
}

type Schema string

// Get gets the schema with the given name, relative to the given account. name
// can be "user", "group" or "event".
// If the schema with the given name for the account does not exist, this method
// returns an empty schema.
func (schemas *Schemas) Get(account int, name string) (Schema, error) {
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
	row, err := schemas.APIs.myDB.Table("Schemas").Get(sql.Where{"account": account}, []any{column})
	if err != nil {
		return "", err
	}
	schema, _ := row[column].(string)
	return Schema(schema), nil
}

// Update updates the schema with the given name, relative to the given account.
// name can be "user", "group" or "event".
func (schemas *Schemas) Update(account int, name, schema string) error {
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
	_, err := schemas.APIs.myDB.Table("Schemas").Update(sql.Set{column: schema}, sql.Where{"account": account})
	if err != nil {
		return err
	}
	return nil
}
