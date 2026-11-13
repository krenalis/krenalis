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
	"github.com/krenalis/krenalis/warehouses/internal/profilefilter"
)

// profileFilterVerified gates prepared filters pending real-database tests.
const profileFilterVerified = false

// ProfileFilterVerified reports backend verification of prepared filters.
func (warehouse *Snowflake) ProfileFilterVerified() bool {
	return profileFilterVerified
}

// ProfileOutcomes returns a complete Profiles page and exact ternary
// totals from one statement. Source, execution, cancellation, scan and Close
// failures return no partial result. It returns
// warehouses.ErrProfileProjectionTooWide or warehouses.ErrProfileResultTooLarge
// when the requested projection or page exceeds its budget.
func (warehouse *Snowflake) ProfileOutcomes(ctx context.Context, filter warehouses.Filter, request warehouses.ProfileOutcomeRequest) (*warehouses.ProfileOutcomes, error) {
	err := warehouses.ValidateProfileFilter(filter, request)
	if err != nil {
		return nil, err
	}
	if !profileFilterVerified {
		return nil, fmt.Errorf("warehouses/snowflake: prepared Profiles filtering has not been verified")
	}
	query, err := profilefilter.Build(filter, request, profileFilterDialect())
	if err != nil {
		return nil, err
	}
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
		err = rows.Err()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("warehouses/snowflake: profile outcome schema mismatch")
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
		target = "GET(" + target + ", " + literal.String() + ")"
	}
	number := "TRY_TO_DECIMAL(TO_VARCHAR(" + target + "),18,4)"
	double := "AS_DOUBLE(" + target + ")"
	inRange := "ABS(" + double + ")<100000000000000"
	scaled := "(IFF(" + inRange + "," + double + ",0)*16)"
	// A binary value is on the scale-four grid exactly when it is a multiple
	// of 1/16. Bound every cast independently of CASE evaluation order.
	return "CASE WHEN TYPEOF(" + target + ")='DOUBLE' THEN CASE WHEN " + inRange +
		" AND " + scaled + "=TRUNC(" + scaled + ") THEN CAST(CAST(" + scaled +
		" AS NUMBER(16,0))*0.0625 AS NUMBER(18,4)) END WHEN IS_DECIMAL(" + target + ") THEN CASE WHEN " +
		target + "=TO_VARIANT(" + number + ") THEN " + number + " END END", nil
}

// profileExactText fixes the collation of a native source identity.
func profileExactText(column warehouses.Column) (string, error) {
	if !warehouses.IsValidIdentifier(column.Name) || !column.Type.IsString() {
		return "", fmt.Errorf("warehouses/snowflake: invalid exact text binding")
	}
	return "COLLATE(" + quoteIdent(column.Name) + ", 'utf8')", nil
}

func profileFilterDialect() profilefilter.Dialect {
	return profilefilter.Dialect{ValueSize: profileValueSize, QuoteIdentifier: quoteIdent, Record: profileRecord,
		ExactText: profileExactText, KnownText: profileKnownText, ExactDecimal: profileExactDecimal,
		Parameter: profileParameter}
}

// profileKnownText extracts only a JSON String, preserving unknown business
// values as SQL NULL. It does not change the Record comparison operators.
func profileKnownText(column warehouses.Column, keys []string) (string, error) {
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

func profileParameter(index int, typ types.Type) (string, error) {
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

// profileRecord keeps malformed internal prepared expressions from bringing
// down the caller; the original renderer still owns every Record operation.
func profileRecord(expr warehouses.Expr) (text string, err error) {
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

func profileValueSize(expression string) string {
	return "OCTET_LENGTH(TO_VARCHAR(" + expression + "))"
}
