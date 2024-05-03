//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package warehouses

import (
	"errors"
	"fmt"
	"strings"

	"github.com/open2b/chichi/types"
)

// ColumnsToProperties returns the type properties of columns.
// Consecutive columns with a common prefix are grouped into a single object
// property. It could change the columns slice and the column names.
//
// Grouping columns can result in properties with the same name. In this case,
// it returns a RepeatedPropertyNameError error.
//
// TODO(Gianluca): this code will probably be rewritten or removed when we
// implement the changes related to the schemas/properties/columns discussed in
// the issue https://github.com/open2b/chichi/issues/708.
func ColumnsToProperties(columns []types.Property) ([]types.Property, error) {
	var properties []types.Property
	for i := 0; i < len(columns); i++ {
		c := columns[i]
		name := c.Name
		var property types.Property
		// group the columns with the same prefix.
		if prefix, n := columnsCommonPrefix(columns[i:]); prefix != "" {
			group := columns[i : i+n]
			i += n - 1
			for j := 0; j < n; j++ {
				column := group[j]
				group[j].Name = strings.TrimPrefix(column.Name, prefix)
			}
			if n == 0 {
				continue
			}
			props, err := ColumnsToProperties(group[:n])
			if err != nil {
				return nil, err
			}
			if strings.HasPrefix(prefix, "__") && strings.HasSuffix(prefix, "__") {
				property = types.Property{
					Name: prefix,
					Type: types.Object(props),
				}
			} else {
				property = types.Property{
					Name: strings.TrimSuffix(prefix, "_"),
					Type: types.Object(props),
				}
			}
		} else {
			property = c
			property.Name = c.Name
		}
		for _, p := range properties {
			if p.Name == property.Name {
				return nil, Errorf("column %s results in a repeated property named %s", name, p.Name)
			}
		}
		properties = append(properties, property)
	}
	return properties, nil
}

// columnsCommonPrefix returns the common prefix between the first column in
// columns and the successive consecutive columns. A common prefix, if exists,
// ends with an underscore character ('_').
//
// If a common prefix exists, it returns the prefix, and the number of
// consecutive columns having the common prefix, starting from the first
// column, otherwise it returns an empty string and zero.
//
// See TestColumnsCommonPrefix for some examples.
func columnsCommonPrefix(columns []types.Property) (string, int) {
	first := columns[0].Name
	if strings.HasPrefix(first, "__") && strings.HasSuffix(first, "__") {
		return "", 0
	}
	var prefix string
	var n = len(columns)
Columns:
	for i := 0; i < len(first)-1; i++ {
		c := first[i]
		for k := 1; k < n; k++ {
			name := columns[k].Name
			if i < len(name)-1 && name[i] == c {
				// continue with the next column.
				if i > 0 && c == '_' {
					prefix = first[:i+1]
				}
				continue
			}
			if prefix == "" {
				// continue only with the previous columns.
				n = k
				continue Columns
			}
			// break and return the prefix.
			break Columns
		}
	}
	if prefix == "" {
		n = 0
	}
	return prefix, n
}

// PropertiesToColumns returns the columns of properties.
func PropertiesToColumns(properties []types.Property) []types.Property {
	columns := make([]types.Property, 0, len(properties))
	for _, p := range properties {
		if p.Type.Kind() == types.ObjectKind {
			for _, column := range PropertiesToColumns(p.Type.Properties()) {
				column.Name = PropertyNameToColumnName(p.Name) + "_" + column.Name
				columns = append(columns, column)
			}
			continue
		}
		columns = append(columns, types.Property{
			Name:     PropertyNameToColumnName(p.Name),
			Type:     p.Type,
			Nullable: p.Nullable,
		})
	}
	return columns
}

// PropertyNameToColumnName returns the given property name as column name.
//
// TODO(Gianluca): this code will probably be rewritten or removed when we
// implement the changes related to the schemas/properties/columns discussed in
// the issue https://github.com/open2b/chichi/issues/708.
func PropertyNameToColumnName(name string) string {
	return name
}

// PropertyPathToColumn returns the column for the property path in schema.
func PropertyPathToColumn(schema types.Type, path string) (column types.Property, err error) {
	typ := schema
	var name strings.Builder
	parts := strings.Split(path, ".")
	for i, part := range parts {
		if typ.Kind() != types.ObjectKind {
			return types.Property{}, errors.New("path refers to a non-object type")
		}
		prop, ok := typ.Property(part)
		if !ok {
			return types.Property{}, fmt.Errorf("property %q does not exist", part)
		}
		typ = prop.Type
		if i == 0 {
			name.WriteString(PropertyNameToColumnName(prop.Name))
		} else {
			name.WriteByte('_')
			name.WriteString(prop.Name)
		}
	}
	property := types.Property{
		Name: name.String(),
		Type: typ,
	}
	return property, nil
}

// DeserializeRowAsMap deserializes a row returned by a data warehouse as map.
// It returns the deserialized row and the remaining row values to read.
func DeserializeRowAsMap(properties []types.Property, row []any) (map[string]any, []any) {
	values := make(map[string]any, len(properties))
	for _, p := range properties {
		if p.Type.Kind() == types.ObjectKind {
			values[p.Name], row = DeserializeRowAsMap(p.Type.Properties(), row)
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
				flattenInto(v, value.(map[string]any), PropertyNameToColumnName(p.Name), p.Type)
				continue
			}
			serialize(value, p.Type)
			if name := PropertyNameToColumnName(p.Name); name != p.Name {
				v[name] = v[p.Name]
				delete(v, p.Name)
			}
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
			flattenInto(dst, value.(map[string]any), prefix+"_"+PropertyNameToColumnName(name), p.Type)
			continue
		}
		serialize(value, p.Type)
		dst[prefix+"_"+PropertyNameToColumnName(name)] = value
	}
}
