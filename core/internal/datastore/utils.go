// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"strings"

	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/warehouses"
)

// IsMetaProperty reports whether the given property name refers to a property
// considered a meta property by a data warehouse.
func IsMetaProperty(name string) bool {
	return len(name) >= 5 && strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__")
}

// appendColumnsFromProperties appends to columns the columns corresponding to
// the provided properties paths, based on the mapping defined in
// columnByProperty, and returns the extended slice.
func appendColumnsFromProperties(columns []warehouses.Column, properties []string, columnByProperty map[string]warehouses.Column) []warehouses.Column {
	for _, path := range properties {
		column, ok := columnByProperty[path]
		if ok {
			columns = append(columns, column)
			continue
		}
		n := len(path)
		for property, column := range columnByProperty {
			// Append columns of properties whose path starts with `path` followed by a "."
			if len(property) > n && property[n] == '.' && property[:n] == path {
				columns = append(columns, column)
			}
		}
	}
	return columns
}

// An unflatRowFunc function unflats a row read from the data warehouse into a
// map[string]any value.
type unflatRowFunc func(row []any) map[string]any

// columnsFromProperties returns the columns corresponding to the provided
// properties paths, based on the mapping defined in columnByProperty that can
// be used in a query to a data warehouse. Additionally, it returns a function
// that can be used to transform a row retrieved from a data warehouse into its
// map representation. omitNil, when set to true, specifies that properties with
// a nil value should be omitted from each record during transformation using
// unflatRowFunc.
func columnsFromProperties(properties []string, columnByProperty map[string]warehouses.Column, omitNil bool) ([]warehouses.Column, unflatRowFunc) {
	pk := &propertyKey{}
	columns := make([]warehouses.Column, 0, len(properties))
	for _, path := range properties {
		column, ok := columnByProperty[path]
		if ok {
			pk.add(path, len(columns))
			columns = append(columns, column)
			continue
		}
		n := len(path)
		for property, column := range columnByProperty {
			// Append columns of properties whose path starts with `path` followed by a "."
			if len(property) > n && property[n] == '.' && property[:n] == path {
				pk.add(property, len(columns))
				columns = append(columns, column)
			}
		}
	}
	unflat := func(row []any) map[string]any {
		return unflatRow(pk, row, omitNil)
	}
	return columns, unflat
}

type propertyKey struct {
	name       string
	index      int
	properties []propertyKey
}

// add adds a property path with a given index.
func (pk *propertyKey) add(path string, index int) {
	var ok bool
	var name string
Path:
	for {
		name, path, ok = strings.Cut(path, ".")
		if !ok {
			pk.properties = append(pk.properties, propertyKey{name: name, index: index})
			return
		}
		for i := 0; i < len(pk.properties); i++ {
			if pk.properties[i].name == name {
				pk = &pk.properties[i]
				continue Path
			}
		}
		p := propertyKey{name: name, properties: []propertyKey{}}
		pk.properties = append(pk.properties, p)
		pk = &pk.properties[len(pk.properties)-1]
	}
}

// identityColumnByProperty returns a mapping from identity properties to their
// corresponding columns.
//
// This mapping is derived from the profile's property-to-column mapping,
// substituting meta properties with the meta properties of identity.
func identityColumnByProperty(userColumnByProperty map[string]warehouses.Column) map[string]warehouses.Column {
	columns := map[string]warehouses.Column{
		"__pk__":               {Name: "__pk__", Type: types.Int(32)},
		"__action__":           {Name: "__action__", Type: types.Int(32)},
		"__is_anonymous__":     {Name: "__is_anonymous__", Type: types.Boolean()},
		"__identity_id__":      {Name: "__identity_id__", Type: types.Text()},
		"__connection__":       {Name: "__connection__", Type: types.Int(32)},
		"__anonymous_ids__":    {Name: "__anonymous_ids__", Type: types.Array(types.Text()), Nullable: true},
		"__last_change_time__": {Name: "__last_change_time__", Type: types.DateTime()},
		"__mpid__":             {Name: "__mpid__", Type: types.UUID(), Nullable: true},
	}
	for property, column := range userColumnByProperty {
		if !IsMetaProperty(property) {
			columns[property] = column
		}
	}
	return columns
}

func unflatRow(pk *propertyKey, row []any, omitNil bool) map[string]any {
	v := unflatRowRec(pk, row, omitNil)
	if v == nil {
		return map[string]any{}
	}
	return v.(map[string]any)
}

func unflatRowRec(pk *propertyKey, row []any, omitNil bool) any {
	var x any
	var v map[string]any
	for _, p := range pk.properties {
		if p.properties == nil {
			x = row[p.index]
		} else {
			x = unflatRowRec(&p, row, omitNil)
		}
		if omitNil && x == nil {
			continue
		}
		if v == nil {
			v = map[string]any{p.name: x}
		} else {
			v[p.name] = x
		}
	}
	if v == nil {
		return nil
	}
	return v
}

// profileColumnByProperty returns a mapping from properties of the profile
// schema to their respective columns. It assumes that for a property path like
// "a.b.c", the corresponding column is named "a_b_c".
func profileColumnByProperty(schema types.Type) map[string]warehouses.Column {
	columnByProperty := map[string]warehouses.Column{}
	for path, p := range schema.Properties().WalkAll() {
		if p.Type.Kind() == types.ObjectKind {
			continue
		}
		name := strings.ReplaceAll(path, ".", "_")
		columnByProperty[path] = warehouses.Column{
			Name: name,
			Type: p.Type,
			// Profile schema properties are always non-nullable, while profile
			// columns are always nullable.
			Nullable: true,
		}
	}
	return columnByProperty
}
