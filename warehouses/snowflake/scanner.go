// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

// scanner implements the warehouses.Rows interface to read and normalize the rows
// read from Snowflake.
type scanner struct {
	columns []warehouses.Column
	rows    *sql.Rows
	values  []any
	dest    []any
	index   int
}

// newScanner returns a new scanner.
func newScanner(columns []warehouses.Column, rows *sql.Rows) *scanner {
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
	return snowflake(err)
}

// normalize normalizes the value v read from Snowflake.
func (s *scanner) normalize(name string, typ types.Type, v any) (any, error) {
	switch typ.Kind() {
	case types.StringKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateString(name, typ, v)
		}
	case types.BooleanKind:
		if _, ok := v.(bool); ok {
			return v, nil
		}
	case types.IntKind:
		switch v := v.(type) {
		case int:
			return warehouses.ValidateInt(name, typ, v)
		case int64:
			return warehouses.ValidateInt(name, typ, int(v))
		case string:
			if v, err := strconv.ParseInt(v, 10, 64); err == nil {
				return warehouses.ValidateInt(name, typ, int(v))
			}
		}
	case types.UintKind:
		switch v := v.(type) {
		case int:
			if v >= 0 {
				return warehouses.ValidateUint(name, typ, uint(v))
			}
		case int64:
			if v >= 0 {
				return warehouses.ValidateUint(name, typ, uint(v))
			}
		case string:
			if v, err := strconv.ParseUint(v, 10, 64); err == nil {
				return warehouses.ValidateUint(name, typ, uint(v))
			}
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
	case types.YearKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateYearString(name, v)
		}
	case types.UUIDKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateUUID(name, v)
		}
	case types.JSONKind:
		return warehouses.ValidateJSON(name, v)
	case types.InetKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateInet(name, v)
		}
	case types.ArrayKind:
		if v, ok := v.(string); ok {
			r := strings.NewReader(v)
			if v, err := types.Decode[[]any](r, typ); err == nil {
				return v, nil
			}
		}
	case types.MapKind:
		if v, ok := v.(string); ok {
			r := strings.NewReader(v)
			if v, err := types.Decode[map[string]any](r, typ); err == nil {
				return v, nil
			}
		}
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
