// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"context"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/tools/types"
)

// CompiledQuery is an application-generated statement with a normalized result
// schema. SQL must never come from a filter payload or an external request.
type CompiledQuery struct {
	SQL       string
	Arguments []QueryArgument
	Columns   []Column
}

// Values validates the transport envelope and returns driver arguments.
// DateTime arguments preserve microseconds as UTC timestamp-without-zone text.
func (q CompiledQuery) Values() ([]any, error) {
	if len(q.SQL) == 0 || len(q.SQL) > 1<<20 || len(q.Columns) == 0 || len(q.Columns) > 100 || len(q.Arguments) > 1000 {
		return nil, fmt.Errorf("warehouses: invalid compiled query envelope")
	}
	for _, column := range q.Columns {
		if !IsValidIdentifier(column.Name) || !column.Type.Valid() {
			return nil, fmt.Errorf("warehouses: invalid compiled result column")
		}
	}
	values := make([]any, len(q.Arguments))
	for i, arg := range q.Arguments {
		switch arg.Type.Kind() {
		case types.StringKind:
			value, ok := arg.Value.(string)
			if !ok || len(value) > 4096 {
				return nil, fmt.Errorf("warehouses: invalid compiled text argument")
			}
			values[i] = value
		case types.DateTimeKind:
			value, ok := arg.Value.(time.Time)
			value = value.UTC()
			if !ok || value.IsZero() || value.Year() < 1 || value.Year() > 9999 || value.Nanosecond()%1000 != 0 {
				return nil, fmt.Errorf("warehouses: invalid compiled timestamp argument")
			}
			values[i] = value.UTC().Format("2006-01-02 15:04:05.000000")
		default:
			return nil, fmt.Errorf("warehouses: unsupported compiled argument type")
		}
	}
	return values, nil
}

// CompiledQueryWarehouse provides the internal generated-query path without
// changing RowQuery, Expr or the public filtering format.
type CompiledQueryWarehouse interface {
	CompiledQueryDialect() QueryDialect
	QueryCompiled(context.Context, CompiledQuery) (Rows, error)
}

// QueryArgument is a typed value bound by the compiled-query transport.
type QueryArgument struct {
	Type  types.Type
	Value any
}

// QueryDialect contains only physical rendering operations. Its callbacks are
// used with resolved identifiers and prepared Record expressions.
type QueryDialect struct {
	QuoteIdentifier func(string) string
	Record          func(Expr) (string, error)
	KnownText       func(Column, []string) (string, error)
	Parameter       func(int, types.Type) (string, error)

	// CountVerified gates only the internal Count subset until this backend
	// has passed its real-database tests.
	CountVerified bool
}
