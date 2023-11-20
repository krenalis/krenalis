//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package snowflake

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"chichi/apis/datastore/warehouses"
	"chichi/connector/types"
)

// scanValue implements the sql.Scanner interface to read the database values.
type scanValue struct {
	property    types.Property
	rows        *[][]any
	columnIndex int
	columnCount int
}

// newScanValues returns a slice containing scan values to be used to scan rows.
func newScanValues(properties []types.Property, rows *[][]any) []any {
	values := make([]any, len(properties))
	for i, p := range properties {
		values[i] = scanValue{
			property:    p,
			rows:        rows,
			columnIndex: i,
			columnCount: len(properties),
		}
	}
	return values
}

func (sv scanValue) Scan(src any) error {
	p := sv.property
	value, err := normalize(p.Name, p.Type, src, p.Nullable)
	if err != nil {
		return err
	}
	var row []any
	if sv.columnIndex == 0 {
		row = make([]any, sv.columnCount)
		*sv.rows = append(*sv.rows, row)
	} else {
		row = (*sv.rows)[len(*sv.rows)-1]
	}
	row[sv.columnIndex] = value
	return nil
}

// normalize normalizes a value returned by Snowflake and returns its normalized
// form. If the value is not valid it returns an error.
func normalize(name string, typ types.Type, v any, nullable bool) (any, error) {
	if v == nil {
		if !nullable {
			return nil, fmt.Errorf("column %s is non-nullable, but Snowflake returned a NULL value", name)
		}
		return nil, nil
	}
	switch typ.Kind() {
	case types.BooleanKind:
		if _, ok := v.(bool); ok {
			return v, nil
		}
	case types.FloatKind:
		if v, ok := v.(float64); ok {
			return warehouses.ValidateFloat(name, typ, v)
		}
	case types.DecimalKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateDecimalString(name, typ, v)
		}
	case types.DateTimeKind:
		if v, ok := v.(time.Time); ok {
			return warehouses.ValidateDateTime(name, v)
		}
	case types.DateKind:
		if v, ok := v.(time.Time); ok {
			return warehouses.ValidateDate(name, v)
		}
	case types.TimeKind:
		if v, ok := v.(time.Time); ok {
			return warehouses.ValidateTime(v)
		}
	case types.JSONKind:
		return warehouses.ValidateJSON(name, v)
	case types.TextKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateText(name, typ, v)
		}
	case types.ArrayKind:
		// The driver returns the value as a JSON array.
		v, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Array type", v, name)
		}
		if v == "" || v[0] != '[' {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Array type", v, name)
		}
		// Snowflake only supports JSON as the item type.
		if typ.Elem().Kind() != types.JSONKind {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Array type", v, name)
		}
		dec := json.NewDecoder(strings.NewReader(v))
		dec.UseNumber()
		var a any
		err := dec.Decode(&a)
		if err != nil {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Array type", v, name)
		}
		return a, nil
	case types.MapKind:
		// The driver returns the value as a JSON object.
		v, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Map type", v, name)
		}
		if v == "" || v[0] != '{' {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Map type", v, name)
		}
		// Snowflake only supports JSON as the item type.
		if typ.Elem().Kind() == types.JSONKind {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Map type", v, name)
		}
		dec := json.NewDecoder(strings.NewReader(v))
		dec.UseNumber()
		var m any
		err := dec.Decode(&m)
		if err != nil {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Map type", v, name)
		}
		return m, nil
	}
	return fmt.Errorf("Snowflake has returned an unsopported type %T for column %s", v, name), nil
}
