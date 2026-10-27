// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// CompiledQueryDialect returns the physical adapters for generated queries.
func (warehouse *PostgreSQL) CompiledQueryDialect() warehouses.QueryDialect {
	return warehouses.QueryDialect{QuoteIdentifier: quoteIdent, Record: compiledRecord,
		KnownText: compiledKnownText, Parameter: compiledParameter, CountVerified: true}
}

// QueryCompiled executes generated SQL using the normalized scanner.
func (warehouse *PostgreSQL) QueryCompiled(ctx context.Context, query warehouses.CompiledQuery) (warehouses.Rows, error) {
	values, err := query.Values()
	if err != nil {
		return nil, err
	}
	pool, _, err := warehouse.connectionPool(ctx, false)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, query.SQL, values...)
	if err != nil {
		return nil, err
	}
	if len(rows.FieldDescriptions()) != len(query.Columns) {
		rows.Close()
		return nil, fmt.Errorf("warehouses/postgresql: compiled result schema mismatch")
	}
	return newScanner(query.Columns, rows), nil
}

// compiledKnownText extracts only a JSON String, preserving unknown business
// values as SQL NULL. It does not change the Record comparison operators.
func compiledKnownText(column warehouses.Column, keys []string) (string, error) {
	if !warehouses.IsValidIdentifier(column.Name) || !column.Type.IsJSON() || len(keys) > 32 {
		return "", fmt.Errorf("warehouses/postgresql: invalid governed text binding")
	}
	target := quoteIdent(column.Name)
	for _, key := range keys {
		if len(key) > 1024 || strings.ContainsRune(key, 0) {
			return "", fmt.Errorf("warehouses/postgresql: invalid governed text key")
		}
		var literal strings.Builder
		quoteString(&literal, key)
		target = "(" + target + " -> " + literal.String() + ")"
	}
	return "CASE WHEN jsonb_typeof(" + target + ")='string' THEN " + target + " #>> '{}' END", nil
}

func compiledParameter(index int, typ types.Type) (string, error) {
	if index < 1 || index > 1000 {
		return "", fmt.Errorf("warehouses/postgresql: invalid compiled parameter index")
	}
	p := "$" + strconv.Itoa(index)
	switch typ.Kind() {
	case types.StringKind:
		return p, nil
	case types.DateTimeKind:
		return "CAST(" + p + " AS timestamp(6))", nil
	default:
		return "", fmt.Errorf("warehouses/postgresql: unsupported compiled parameter type")
	}
}

// compiledRecord keeps malformed internal prepared expressions from bringing
// down the caller; the original renderer still owns every Record operation.
func compiledRecord(expr warehouses.Expr) (text string, err error) {
	defer func() {
		if recover() != nil {
			text, err = "", fmt.Errorf("warehouses/postgresql: malformed prepared Record expression")
		}
	}()
	if expr == nil {
		return "", fmt.Errorf("warehouses/postgresql: missing prepared Record expression")
	}
	var b strings.Builder
	err = renderExpr(&b, expr)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}
