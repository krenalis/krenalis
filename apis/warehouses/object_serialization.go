//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package warehouses

import (
	"fmt"
	"strings"

	"chichi/connector/types"
)

// A RepeatedPropertyNameError value is returned from ColumnsToProperties when
// grouped columns result in a repeated property name.
type RepeatedPropertyNameError struct {
	Column, Property string
}

func (err RepeatedPropertyNameError) Error() string {
	return fmt.Sprintf("column %s results in a repeated property named %s", err.Column, err.Property)
}

// ColumnsToProperties returns the type properties of columns.
// Consecutive columns with a common prefix are grouped into a single object
// property. It could change the columns slice and the column names.
//
// Columns starting with an underscore ('_'), are grouped as if the underscore
// were not present but are not returned as properties.
//
// Grouping columns can result in properties with the same name. In this case,
// it returns a RepeatedPropertyNameError error.
func ColumnsToProperties(columns []types.Property) ([]types.Property, error) {
	var properties []types.Property
	for i := 0; i < len(columns); i++ {
		c := columns[i]
		var property types.Property
		// group the columns with the same prefix.
		if prefix, n := columnsCommonPrefix(columns[i:]); prefix != "" {
			group := columns[i : i+n]
			i += n - 1
			for j := 0; j < n; j++ {
				column := group[j]
				// remove from the group the columns with an underscore prefix.
				if column.Name[0] == '_' {
					copy(group[j:], group[j+1:])
					j--
					n--
					continue
				}
				// remove the prefix from the column names.
				group[j].Name = strings.TrimPrefix(column.Name, prefix)
			}
			if n == 0 {
				continue
			}
			props, err := ColumnsToProperties(group[:n])
			if err != nil {
				return nil, err
			}
			property = types.Property{
				Name: strings.TrimSuffix(prefix, "_"),
				Type: types.Object(props),
				Flat: true,
			}
		} else {
			if c.Name[0] == '_' {
				continue
			}
			property = c
		}
		for _, p := range properties {
			if p.Name == property.Name {
				return nil, RepeatedPropertyNameError{c.Name, p.Name}
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
	if first[0] == '_' {
		first = first[1:]
	}
	var prefix string
	var n = len(columns)
Columns:
	for i := 0; i < len(first)-1; i++ {
		c := first[i]
		for k := 1; k < n; k++ {
			name := columns[k].Name
			if name[0] == '_' {
				name = name[1:]
			}
			if i < len(name)-1 && name[i] == c {
				// continue with the next column.
				if c == '_' {
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
		if p.Flat {
			for _, column := range PropertiesToColumns(p.Type.Properties()) {
				column.Name = p.Name + "_" + column.Name
				columns = append(columns, column)
			}
			continue
		}
		columns = append(columns, types.Property{Name: p.Name, Type: p.Type, Nullable: p.Nullable})
	}
	return columns
}

// DeserializeRowAsSlice deserializes a row returned by a data warehouse as
// slice.
func DeserializeRowAsSlice(properties []types.Property, row []any) []any {
	values := make([]any, len(properties))
	for i, p := range properties {
		if p.Flat {
			values[i], row = DeserializeRowAsMap(p.Type.Properties(), row)
			continue
		}
		values[i] = row[0]
		row = row[1:]
	}
	return values
}

// DeserializeRowAsMap deserializes a row returned by a data warehouse as map.
// It returns the deserialized row and the remaining row values to read.
func DeserializeRowAsMap(properties []types.Property, row []any) (map[string]any, []any) {
	values := make(map[string]any, len(properties))
	for _, p := range properties {
		if p.Flat {
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
	switch t.PhysicalType() {
	case types.PtObject:
		v := v.(map[string]any)
		for _, p := range t.Properties() {
			value, ok := v[p.Name]
			if !ok {
				continue
			}
			if p.Flat {
				delete(v, p.Name)
				flattenInto(v, value.(map[string]any), p.Name, p.Type)
				continue
			}
			serialize(value, p.Type)
			continue
		}
	case types.PtArray:
		itemType := t.ItemType()
		for _, value := range v.([]any) {
			serialize(value, itemType)
		}
	case types.PtMap:
		valueType := t.ValueType()
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
		if p.Flat {
			flattenInto(dst, value.(map[string]any), prefix+"_"+name, p.Type)
			continue
		}
		serialize(value, p.Type)
		dst[prefix+"_"+name] = value
	}
}
