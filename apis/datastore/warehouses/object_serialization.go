//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package warehouses

import (
	"github.com/open2b/chichi/types"
)

// PropertiesToColumns returns the columns of properties.
func PropertiesToColumns(properties []types.Property) []types.Property {
	columns := make([]types.Property, 0, len(properties))
	for _, p := range properties {
		if p.Type.Kind() == types.ObjectKind {
			for _, column := range PropertiesToColumns(types.Properties(p.Type)) {
				column.Name = p.Name + "_" + column.Name
				columns = append(columns, column)
			}
			continue
		}
		columns = append(columns, types.Property{
			Name:     p.Name,
			Type:     p.Type,
			Nullable: p.Nullable,
		})
	}
	return columns
}

// DeserializeRowAsMap deserializes a row returned by a data warehouse as map.
// It returns the deserialized row and the remaining row values to read.
func DeserializeRowAsMap(properties []types.Property, row []any) (map[string]any, []any) {
	values := make(map[string]any, len(properties))
	for _, p := range properties {
		if p.Type.Kind() == types.ObjectKind {
			values[p.Name], row = DeserializeRowAsMap(types.Properties(p.Type), row)
			continue
		}
		values[p.Name] = row[0]
		row = row[1:]
	}
	return values, row
}

// SerializeRow serializes a row to be passed to a data warehouse by flattening
// fields based on the provided schema.
func SerializeRow(row map[string]any, schema types.Type) {
	serialize(row, schema)
}

// serialize serializes v with type t.
func serialize(v any, t types.Type) {
	if v == nil {
		return
	}
	switch t.Kind() {
	case types.ObjectKind:
		v := v.(map[string]any)
		for _, p := range t.Properties() {
			value, ok := v[p.Name]
			if !ok {
				continue
			}
			if p.Type.Kind() == types.ObjectKind {
				delete(v, p.Name)
				flattenInto(v, value.(map[string]any), p.Name, p.Type)
				continue
			}
			serialize(value, p.Type)
			continue
		}
	case types.ArrayKind:
		itemType := t.Elem()
		for _, value := range v.([]any) {
			serialize(value, itemType)
		}
	case types.MapKind:
		valueType := t.Elem()
		for _, value := range v.(map[string]any) {
			serialize(value, valueType)
		}
	}
}

// flattenInto flattens the properties of obj with type t into dst with names
// prefixed by prefix.
func flattenInto(dst, obj map[string]any, prefix string, t types.Type) {
	for name, value := range obj {
		p, _ := t.Property(name)
		if p.Type.Kind() == types.ObjectKind {
			flattenInto(dst, value.(map[string]any), prefix+"_"+name, p.Type)
			continue
		}
		serialize(value, p.Type)
		dst[prefix+"_"+name] = value
	}
}
