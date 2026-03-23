// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/types"
)

// pgTypeInfo holds information about a PostgreSQL type, as read from the
// PostgreSQL information tables (as 'information_schema.columns' and
// 'information_schema.attributes').
type pgTypeInfo struct {
	table      string
	column     string
	dataType   string
	udtName    string
	charLength *string
	precision  *string
	radix      *string
	scale      *string
}

// columns returns the columns of the table in schema.
//
// If the table does not exist, this method returns an error.
func (ps *PostgreSQL) columns(ctx context.Context, schema, table string) ([]connectors.Column, error) {

	if err := ps.openDB(ctx); err != nil {
		return nil, err
	}
	tx, err := ps.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Read the enums.
	enums := map[string]types.Type{}
	{
		query := "SELECT pg_type.typname, pg_enum.enumlabel FROM pg_type JOIN pg_enum ON pg_enum.enumtypid = pg_type.oid"
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		rawEnums := map[string][]string{}
		for rows.Next() {
			var typName, enumLabel string
			if err = rows.Scan(&typName, &enumLabel); err != nil {
				return nil, err
			}
			if typName == "" {
				return nil, errors.New("invalid empty enum name")
			}
			if !utf8.ValidString(enumLabel) {
				return nil, fmt.Errorf("not-valid UTF-8 encoded enum label for type %q", typName)
			}
			rawEnums[typName] = append(rawEnums[typName], enumLabel)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		for name, values := range rawEnums {
			enums[name] = types.String().WithValues(values...)
		}
	}

	// Read the "attTypMods".
	// They are necessary to build the Meergo type of certain columns with
	// specific PostgreSQL types.
	attTypMods := map[string]map[string]*int{}
	{
		query := "SELECT c.relname, a.attname, a.atttypmod\n" +
			"FROM pg_attribute AS a\n" +
			"INNER JOIN pg_class AS c ON a.attrelid = c.oid\n" +
			"INNER JOIN pg_namespace AS n ON c.relnamespace = n.oid\n" +
			"WHERE n.nspname = '" + schema + "' AND a.atttypmod <> -1"
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var relname, attname string
			var atttypmod int
			err := rows.Scan(&relname, &attname, &atttypmod)
			if err != nil {
				return nil, err
			}
			if attTypMods[relname] == nil {
				attTypMods[relname] = map[string]*int{attname: &atttypmod}
			} else {
				attTypMods[relname][attname] = &atttypmod
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}

	// Read the columns.
	var columns []connectors.Column
	{
		query := "SELECT c.table_name, c.column_name, c.is_nullable, c.data_type, c.udt_name, c.character_maximum_length," +
			" c.numeric_precision, c.numeric_precision_radix, c.numeric_scale, c.is_updatable\n" +
			"FROM information_schema.columns c\n" +
			"INNER JOIN information_schema.tables t ON c.table_name = t.table_name AND c.table_schema = t.table_schema\n" +
			"WHERE t.table_schema = '" + schema + "' AND t.table_type = 'BASE TABLE' AND" +
			" ( t.table_name IN ('" + table + "'))\n" +
			"ORDER BY c.table_name, c.ordinal_position"
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var row pgTypeInfo
			var tableName, columnName, dataType, udtName, isNullable, isUpdatable *string
			if err = rows.Scan(&tableName, &columnName, &isNullable, &dataType,
				&udtName, &row.charLength, &row.precision, &row.radix, &row.scale, &isUpdatable); err != nil {
				return nil, err
			}
			if tableName == nil {
				return nil, errors.New("database has returned NULL as table name")
			}
			row.table = *tableName
			if columnName == nil {
				return nil, errors.New("database has returned NULL as column name")
			}
			row.column = *columnName
			if isNullable == nil {
				return nil, errors.New("database has returned NULL as nullability of column")
			}
			if dataType == nil {
				return nil, errors.New("database has returned NULL as column data type")
			}
			row.dataType = *dataType
			if udtName == nil {
				return nil, errors.New("database has returned NULL as column udt name")
			}
			row.udtName = *udtName
			if isUpdatable == nil {
				return nil, errors.New("database has returned NULL as updatability of column")
			}
			typ, issue, err := columnType(row, enums, attTypMods)
			if err != nil {
				return nil, fmt.Errorf("database has returned an invalid type: %s", err)
			}
			var column connectors.Column
			if typ.Valid() {
				column.Name = row.column
				column.Type = typ
				column.Nullable = *isNullable == "YES"
				column.Writable = *isUpdatable == "YES"
			}
			column.Issue = issue
			columns = append(columns, column)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if columns == nil {
			return nil, fmt.Errorf("table %q does not exist", table)
		}
	}

	return columns, nil
}

// columnType returns the types.Type corresponding to the PostgreSQL type
// described in column.
//
// enums is a mapping of the available enum types.
//
// attTypMods holds a type-specific data attributes read from the table
// 'pg_attribute.atttypmod'. The first key is the table name (or the composite
// type name, which is stored in PostgreSQL as a table), while the second key is
// the column name (or composite type field name). It represents, for example,
// the maximum length of a varchar column or the maximum length of the text of
// an array element); may not contain a key if the column type has no associated
// type-specific data.
func columnType(column pgTypeInfo, enums map[string]types.Type, attTypMods map[string]map[string]*int) (types.Type, string, error) {
	var t types.Type
	switch column.dataType {
	case "smallint":
		t = types.Int(16)
	case "integer":
		t = types.Int(32)
	case "bigint":
		t = types.Int(64)
	case "numeric":
		// Parse precision radix.
		if column.radix == nil {
			return types.Type{}, "", errors.New("numeric_precision_radix value is NULL")
		}
		rdx, _ := strconv.Atoi(*column.radix)
		if rdx != 2 && rdx != 10 {
			return types.Type{}, "", fmt.Errorf("numeric_precision_radix value %q is not valid", *column.radix)
		}
		// Parse precision.
		if column.precision == nil {
			return types.Type{}, "", errors.New("numeric_precision value is NULL")
		}
		p, err := strconv.ParseInt(*column.precision, rdx, 64)
		if err != nil || p < 1 {
			return types.Type{}, "", fmt.Errorf("numeric_precision value %q is not valid", *column.precision)
		}
		// Parse scale.
		if column.scale == nil {
			return types.Type{}, "", errors.New("numeric_scale value is NULL")
		}
		s, err := strconv.ParseInt(*column.scale, rdx, 64)
		if err != nil || s < 0 || s > p {
			return types.Type{}, "", fmt.Errorf("numeric_scale value %q is not valid", *column.scale)
		}
		if p > types.MaxDecimalPrecision {
			issue := fmt.Sprintf("Column %q has a precision of %d, which exceeds the maximum supported precision of %d.", column.column, p, types.MaxDecimalPrecision)
			return types.Type{}, issue, nil
		}
		if s > types.MaxDecimalScale {
			issue := fmt.Sprintf("Column %q has a scale of %d, which exceeds the maximum supported scale of %d.", column.column, s, types.MaxDecimalScale)
			return types.Type{}, issue, nil
		}
		t = types.Decimal(int(p), int(s))
	case "real":
		t = types.Float(32)
	case "double precision":
		t = types.Float(64)
	case "character varying", "character":
		if column.charLength != nil {
			chars, err := strconv.Atoi(*column.charLength)
			if err != nil {
				return types.Type{}, "", fmt.Errorf("character_maximum_length value %q is not valid", *column.precision)
			}
			if chars < 1 || chars > types.MaxStringLen {
				issue := fmt.Sprintf("Column %q has a character length of %d, which exceeds the maximum allowed length of %d", column.column, chars, types.MaxStringLen)
				return types.Type{}, issue, nil
			}
			t = types.String().WithMaxLength(chars)
		} else {
			t = types.String()
		}
	case "text":
		t = types.String()
	case "timestamp without time zone", "timestamp with time zone":
		t = types.DateTime()
	case "date":
		t = types.Date()
	case "time without time zone":
		t = types.Time()
	case "boolean":
		t = types.Boolean()
	case "inet":
		t = types.IP()
	case "uuid":
		t = types.UUID()
	case "json", "jsonb":
		t = types.JSON()
	case "ARRAY":
		// From https://www.postgresql.org/docs/current/arrays.html:
		//
		// “[...] However, the current implementation ignores any supplied array
		// size limits, i.e., the behavior is the same as for arrays of
		// unspecified length.”
		//
		// so there's no way to limit the min/max number of array elements.
		var et types.Type
		switch column.udtName {
		case "_bool":
			et = types.Boolean()
		case "_date":
			et = types.Date()
		case "_float4":
			et = types.Float(32)
		case "_float8":
			et = types.Float(64)
		case "_inet":
			et = types.IP()
		case "_int2":
			et = types.Int(16)
		case "_int4":
			et = types.Int(32)
		case "_int8":
			et = types.Int(64)
		case "_json", "_jsonb":
			et = types.JSON()
		case "_text":
			et = types.String()
		case "_time":
			et = types.Time()
		case "_timestamp":
			et = types.DateTime()
		case "_uuid":
			et = types.UUID()
		case "_bpchar", "_varchar":
			attTypMod := attTypMods[column.table][column.column]
			if attTypMod != nil {
				length := *attTypMod - 4 // See the function "_pg_char_max_length".
				if length < 1 {
					return types.Type{}, "", fmt.Errorf("atttypmod value %d is not valid", *attTypMod)
				}
				et = types.String().WithMaxLength(length)
			} else {
				et = types.String()
			}
		}
		if !et.Valid() {
			issue := fmt.Sprintf("Column %q is an array, but its element type %q is not supported.", column.column, column.udtName)
			return types.Type{}, issue, nil
		}
		t = types.Array(et)
	case "USER-DEFINED":
		// Check if the user-defined type is an enum.
		var ok bool
		t, ok = enums[column.udtName]
		if !ok {
			issue := fmt.Sprintf("Column %q has a composite type, which is not supported.", column.column)
			return types.Type{}, issue, nil
		}
	}
	if !t.Valid() {
		issue := fmt.Sprintf("Column %q has an unsupported type %q.", column.column, column.dataType)
		return types.Type{}, issue, nil
	}
	return t, "", nil
}
