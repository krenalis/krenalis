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
	"github.com/krenalis/krenalis/warehouses/internal/profilefilter"
)

// profileFilterVerified gates prepared filters pending real-database tests.
const profileFilterVerified = true

// ProfileFilterVerified reports backend verification of prepared filters.
func (warehouse *PostgreSQL) ProfileFilterVerified() bool {
	return profileFilterVerified
}

// ProfileOutcomes returns a complete Profiles page and exact ternary
// totals from one statement. Source, execution, cancellation, scan and Close
// failures return no partial result. It returns
// warehouses.ErrProfileProjectionTooWide or warehouses.ErrProfileResultTooLarge
// when the requested projection or page exceeds its budget.
func (warehouse *PostgreSQL) ProfileOutcomes(ctx context.Context, filter warehouses.Filter, request warehouses.ProfileOutcomeRequest) (*warehouses.ProfileOutcomes, error) {
	err := warehouses.ValidateProfileFilter(filter, request)
	if err != nil {
		return nil, err
	}
	if !profileFilterVerified {
		return nil, fmt.Errorf("warehouses/postgresql: prepared Profiles filtering has not been verified")
	}
	query, err := profilefilter.Build(filter, request, profileFilterDialect())
	if err != nil {
		return nil, err
	}
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
		return nil, fmt.Errorf("warehouses/postgresql: profile outcome schema mismatch")
	}
	scanner := newScanner(query.Columns, rows)
	scanner.maxBytes = 8 * warehouses.MaxProfilePageBytes
	return profilefilter.Read(ctx, scanner, request)
}

// profileExactDecimal preserves exact readings and returns NULL for unknown
// or incompatible values. It never coerces JSON strings or rounds a reading.
func profileExactDecimal(reading warehouses.DecimalReading) (string, error) {
	err := reading.Validate()
	if err != nil {
		return "", err
	}
	target := quoteIdent(reading.Column.Name)
	for _, key := range reading.Keys {
		var literal strings.Builder
		quoteString(&literal, key)
		target = "(" + target + " -> " + literal.String() + ")"
	}
	number := "(" + target + " #>> '{}')::numeric"
	return "CASE WHEN jsonb_typeof(" + target + ")='number' THEN CASE WHEN " + number +
		" BETWEEN -99999999999999.9999 AND 99999999999999.9999 AND trunc(" + number + ",4)=" + number +
		" THEN CAST(" + number + " AS DECIMAL(18,4)) END END", nil
}

func profileFilterDialect() profilefilter.Dialect {
	return profilefilter.Dialect{ValueSize: profileValueSize, QuoteIdentifier: quoteIdent, Record: profileRecord,
		KnownText: profileKnownText, ExactDecimal: profileExactDecimal, Parameter: profileParameter}
}

// profileKnownText extracts only a JSON String, preserving unknown business
// values as SQL NULL. It does not change the Record comparison operators.
func profileKnownText(column warehouses.Column, keys []string) (string, error) {
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

func profileParameter(index int, typ types.Type) (string, error) {
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

// profileRecord keeps malformed internal prepared expressions from bringing
// down the caller; the original renderer still owns every Record operation.
func profileRecord(expr warehouses.Expr) (text string, err error) {
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

func profileValueSize(expression string) string {
	return "OCTET_LENGTH(CAST(" + expression + " AS text))"
}
