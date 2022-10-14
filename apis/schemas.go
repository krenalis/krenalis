//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"unicode/utf8"

	"chichi/pkg/open2b/sql"
)

type Schemas struct {
	*WorkspaceAPI
}

type Schema string

// Get gets the schema with the given name. name can be "user", "group" o
// "event". If the schema with the given name does not exist, this method
// returns an empty schema.
func (this *Schemas) Get(name string) (Schema, error) {
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
	row, err := this.myDB.Table("Schemas").Get(sql.Where{"workspace": this.workspace}, []any{column})
	if err != nil {
		return "", err
	}
	schema, _ := row[column].(string)
	return Schema(schema), nil
}

// Update updates the schema with the given name. name can be "user", "group"
// or "event".
func (this *Schemas) Update(name, schema string) error {
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
	_, err := this.myDB.Table("Schemas").Update(sql.Set{column: schema}, sql.Where{"account": this.api.account})
	return err
}

// UserProperties returns the name of the properties of the user schema.
//
// TODO(Gianluca): return properties with the same ordering of the schema,
// instead of sorting them alphabetically.
func (this *Schemas) UserProperties() ([]string, error) {
	schema, err := this.Schemas.Get("user")
	if err != nil {
		return nil, err
	}
	var v struct {
		Properties map[string]any
	}
	err = json.Unmarshal([]byte(schema), &v)
	if err != nil {
		return nil, err
	}
	props := make([]string, 0, len(v.Properties))
	for name := range v.Properties {
		props = append(props, name)
	}
	sort.Strings(props)
	return props, nil
}
