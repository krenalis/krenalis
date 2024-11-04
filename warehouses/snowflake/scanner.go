//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package snowflake

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// scanner implements the meergo.Rows interface to read and normalize the rows
// read from Snowflake.
type scanner struct {
	columns []meergo.Column
	rows    *sql.Rows
	values  []any
	dest    []any
	index   int
}

// newScanner returns a new scanner.
func newScanner(columns []meergo.Column, rows *sql.Rows) *scanner {
	s := &scanner{
		columns: columns,
		rows:    rows,
	}
	s.values = make([]any, len(columns))
	for i := range len(s.columns) {
		s.values[i] = scanValue{s}
	}
	return s
}

func (s *scanner) Close() error {
	return s.rows.Close()
}

func (s *scanner) Err() error {
	return s.rows.Err()
}

func (s *scanner) Next() bool {
	return s.rows.Next()
}

// Scan copies the columns from the current row into dest. This differs from the
// Rows.Scan method in the sql package, which copies values into the locations
// pointed to by dest.
func (s *scanner) Scan(dest ...any) error {
	s.dest = dest
	err := s.rows.Scan(s.values...)
	s.dest = nil
	return err
}

// normalize normalizes the value v read from Snowflake.
func (s *scanner) normalize(name string, typ types.Type, v any) (any, error) {

	// TODO(Gianluca): the implementation of this 'normalize' method is obsolete
	// and must be rewritten.
	//
	// See the issue https://github.com/meergo/meergo/issues/1121.

	switch typ.Kind() {
	case types.BooleanKind:
		if _, ok := v.(bool); ok {
			return v, nil
		}
	case types.IntKind:
		switch v := v.(type) {
		case int:
			return meergo.ValidateInt(name, typ, v)
		case int64:
			return meergo.ValidateInt(name, typ, int(v))
		case string:
			if v, err := strconv.ParseInt(v, 10, 64); err == nil {
				return meergo.ValidateInt(name, typ, int(v))
			}
		}
	case types.UintKind:
		switch v := v.(type) {
		case int:
			if v >= 0 {
				return meergo.ValidateUint(name, typ, uint(v))
			}
		case int64:
			if v >= 0 {
				return meergo.ValidateUint(name, typ, uint(v))
			}
		case string:
			if v, err := strconv.ParseUint(v, 10, 64); err == nil {
				return meergo.ValidateUint(name, typ, uint(v))
			}
		}
	case types.FloatKind:
		if v, ok := v.(float64); ok {
			return meergo.ValidateFloat(name, typ, v)
		}
	case types.DecimalKind:
		if v, ok := v.(string); ok {
			return meergo.ValidateDecimalString(name, typ, v)
		}
	case types.DateTimeKind:
		if v, ok := v.(time.Time); ok {
			return meergo.ValidateDateTime(name, v)
		}
	case types.DateKind:
		if v, ok := v.(time.Time); ok {
			return meergo.ValidateDate(name, v)
		}
	case types.TimeKind:
		if v, ok := v.(time.Time); ok {
			return meergo.ValidateTime(v)
		}
	case types.UUIDKind:
		if v, ok := v.(string); ok {
			return meergo.ValidateUUID(name, v)
		}
	case types.JSONKind:
		return meergo.ValidateJSON(name, v)
	case types.TextKind:
		if v, ok := v.(string); ok {
			return meergo.ValidateText(name, typ, v)
		}
	case types.ArrayKind:
		// TODO(Gianluca): the code that handles the arrays has been modified
		// just to allow for the development and testing of identity resolution,
		// and therefore it needs to be reviewed and rewritten.
		//
		// See https://github.com/meergo/meergo/issues/1121.
		v, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Array type", v, name)
		}
		if v == "" {
			return nil, fmt.Errorf("data warehouse returned an empty string for column %s which is an Array type", name)
		}
		ev := json.Value(v)
		if !json.Valid(ev) {
			return nil, fmt.Errorf("data warehouse returned a string with invalid JSON for column %s", name)
		}
		if !ev.IsArray() {
			return nil, fmt.Errorf("data warehouse returned a JSON %s for column %s which is an Array type", ev.Kind(), name)
		}
		min := typ.MinElements()
		max := typ.MaxElements()
		arr := []any{}
		for i, elem := range ev.Elements() {
			if i == max {
				return nil, fmt.Errorf("data warehouse returned an array with more than %d elements for column %s", max, name)
			}
			var elemAny any
			err := elem.Unmarshal(&elemAny)
			if err != nil {
				return nil, fmt.Errorf("cannot unmarshal array element: %s", err)
			}
			arr = append(arr, elemAny)
		}
		if len(arr) < min {
			return nil, fmt.Errorf("data warehouse returned an array with less than %d elements for column %s", min, name)
		}
		if len(arr) == 0 {
			return nil, nil // return the untyped nil.
		}
		return arr, nil
	case types.MapKind:
		// The driver returns the value as a JSON object.
		v, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Map type", v, name)
		}
		if v == "" {
			return nil, fmt.Errorf("data warehouse returned an empty string for column %s which is an Array type", name)
		}
		// Snowflake only supports JSON as the item type.
		if typ.Elem().Kind() != types.JSONKind {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Map type", v, name)
		}
		ev := json.Value(v)
		if !json.Valid(ev) {
			return nil, fmt.Errorf("data warehouse returned a string with invalid JSON for column %s", name)
		}
		if !ev.IsObject() {
			return nil, fmt.Errorf("data warehouse returned a JSON %s for column %s which is a Map type", ev.Kind(), name)
		}
		m := map[string]any{}
		for k, v := range ev.Properties() {
			m[k] = v
		}
		return m, nil
	}
	return nil, fmt.Errorf("Snowflake has returned an unsupported type %T for column %s", v, name)
}

// scanValue implements the sql.Scanner interface to read the values.
type scanValue struct {
	s *scanner
}

func (sv scanValue) Scan(v any) error {
	c := sv.s.columns[sv.s.index]
	var err error
	if v != nil {
		v, err = sv.s.normalize(c.Name, c.Type, v)
	} else if !c.Nullable {
		return fmt.Errorf("column %s is non-nullable, but Snowflake returned a NULL value", c.Name)
	}
	if err != nil {
		sv.s.index = 0
		return err
	}
	sv.s.dest[sv.s.index] = v
	sv.s.index = (sv.s.index + 1) % len(sv.s.columns)
	return nil
}
