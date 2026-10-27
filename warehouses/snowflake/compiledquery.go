// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"fmt"
	"strings"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// CompiledQueryDialect returns the physical adapters for generated queries.
func (warehouse *Snowflake) CompiledQueryDialect() warehouses.QueryDialect {
	return warehouses.QueryDialect{QuoteIdentifier: quoteIdent, Record: compiledRecord,
		KnownText: compiledKnownText, Parameter: compiledParameter, CountVerified: false}
}

// QueryCompiled executes generated SQL using the normalized scanner.
func (warehouse *Snowflake) QueryCompiled(ctx context.Context, query warehouses.CompiledQuery) (warehouses.Rows, error) {
	values, err := query.Values()
	if err != nil {
		return nil, err
	}
	db, err := warehouse.openDB(ctx)
	if err != nil {
		return nil, snowflake(err)
	}
	rows, err := db.QueryContext(ctx, query.SQL, values...)
	if err != nil {
		return nil, snowflake(err)
	}
	columns, err := rows.Columns()
	if err != nil {
		_ = rows.Close()
		return nil, err
	}
	if len(columns) != len(query.Columns) {
		_ = rows.Close()
		return nil, fmt.Errorf("warehouses/snowflake: compiled result schema mismatch")
	}
	return newScanner(query.Columns, rows), nil
}

// compiledKnownText extracts only a JSON String, preserving unknown business
// values as SQL NULL. It does not change the Record comparison operators.
func compiledKnownText(column warehouses.Column, keys []string) (string, error) {
	if !warehouses.IsValidIdentifier(column.Name) || !column.Type.IsJSON() || len(keys) > 32 {
		return "", fmt.Errorf("warehouses/snowflake: invalid governed text binding")
	}
	target := quoteIdent(column.Name)
	for _, key := range keys {
		if len(key) > 1024 || strings.ContainsRune(key, 0) {
			return "", fmt.Errorf("warehouses/snowflake: invalid governed text key")
		}
		var literal strings.Builder
		quoteString(&literal, key)
		target = "GET(" + target + ", " + literal.String() + ")"
	}
	return "CASE WHEN IS_VARCHAR(" + target + ") THEN AS_VARCHAR(" + target + ") END", nil
}

func compiledParameter(index int, typ types.Type) (string, error) {
	if index < 1 || index > 1000 {
		return "", fmt.Errorf("warehouses/snowflake: invalid compiled parameter index")
	}
	p := "?"
	switch typ.Kind() {
	case types.StringKind:
		return p, nil
	case types.DateTimeKind:
		return "CAST(" + p + " AS TIMESTAMP_NTZ(6))", nil
	default:
		return "", fmt.Errorf("warehouses/snowflake: unsupported compiled parameter type")
	}
}

// compiledRecord keeps malformed internal prepared expressions from bringing
// down the caller; the original renderer still owns every Record operation.
func compiledRecord(expr warehouses.Expr) (text string, err error) {
	defer func() {
		if recover() != nil {
			text, err = "", fmt.Errorf("warehouses/snowflake: malformed prepared Record expression")
		}
	}()
	if expr == nil {
		return "", fmt.Errorf("warehouses/snowflake: missing prepared Record expression")
	}
	var b strings.Builder
	err = renderExpr(&b, expr)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}
