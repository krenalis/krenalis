//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package postgresql

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/connector/types"

	"github.com/jackc/pgx/v5"
)

// NOTE. This file must be kept in sync with 'apis/datastore/warehouses/postgresql/tables.go'.

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

type tableSchema struct {
	name    string
	columns []types.Property
}

// tablesSchemas returns the schemas for the existing tables in the schema
// specified in tableNames.
// Therefore, if none of the tables indicated in tableNames exists, this
// function returns an empty slice.
// tableNames must always contain at least one table name.
func tablesSchemas(ctx context.Context, tx pgx.Tx, schema string, tableNames []string) ([]*tableSchema, error) {

	if len(tableNames) == 0 {
		return nil, errors.New("tableNames cannot be empty")
	}

	var table *tableSchema
	var tables []*tableSchema

	// Read the available enums.
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
	enums := map[string]types.Type{}
	for name, values := range rawEnums {
		enums[name] = types.Text().WithValues(values...)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Read the 'atttypmod' attribute of column types, where relevant.
	query = "SELECT c.relname, a.attname, a.atttypmod\n" +
		"FROM pg_attribute AS a\n" +
		"INNER JOIN pg_class AS c ON a.attrelid = c.oid\n" +
		"INNER JOIN pg_namespace AS n ON c.relnamespace = n.oid\n" +
		"WHERE n.nspname = '" + schema + "' AND a.atttypmod <> -1"
	rows, err = tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	attTypMods := map[string]map[string]*int{}
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

	// Instantiate a resolver for the composite types.
	ctResolver, err := initCompositeTypeResolver(ctx, tx, enums, attTypMods)
	if err != nil {
		return nil, err
	}

	// Read columns.
	var tablesNamesStr strings.Builder
	for i, name := range tableNames {
		if i > 0 {
			tablesNamesStr.WriteByte(',')
		}
		tablesNamesStr.WriteByte('\'')
		tablesNamesStr.WriteString(name)
		tablesNamesStr.WriteByte('\'')
	}
	query = "SELECT c.table_name, c.column_name, c.is_nullable, c.data_type, c.udt_name, c.character_maximum_length," +
		" c.numeric_precision, c.numeric_precision_radix, c.numeric_scale, c.is_updatable," +
		" col_description(c.table_name::regclass::oid, c.ordinal_position)\n" +
		"FROM information_schema.columns c\n" +
		"INNER JOIN information_schema.tables t ON c.table_name = t.table_name AND c.table_schema = t.table_schema\n" +
		"WHERE t.table_schema = '" + schema + "' AND t.table_type = 'BASE TABLE' AND" +
		" ( t.table_name IN (" + tablesNamesStr.String() + "))\n" +
		"ORDER BY c.table_name, c.ordinal_position"

	rows, err = tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var row pgTypeInfo
		var tableName, columnName, dataType, udtName, isNullable, isUpdatable, description *string
		if err = rows.Scan(&tableName, &columnName, &isNullable, &dataType,
			&udtName, &row.charLength, &row.precision, &row.radix, &row.scale, &isUpdatable, &description); err != nil {
			return nil, err
		}
		if tableName == nil {
			return nil, errors.New("database has returned NULL as table name")
		}
		row.table = *tableName
		if columnName == nil {
			return nil, errors.New("database has returned NULL as column name")
		}
		if strings.HasPrefix(*columnName, "__") && strings.HasSuffix(*columnName, "__") { // used internally by Chichi.
			continue
		}
		if !types.IsValidPropertyName(*columnName) {
			return nil, fmt.Errorf("column name %q is not supported", *columnName)
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
		var role types.Role
		if *isUpdatable != "YES" {
			role = types.SourceRole
		}
		column := types.Property{
			Name:     row.column,
			Role:     role,
			Nullable: *isNullable == "YES",
		}
		column.Type, err = columnType(row, enums, ctResolver, attTypMods)
		if err != nil {
			return nil, fmt.Errorf("database has returned an invalid type: %s", err)
		}
		if !column.Type.Valid() {
			return nil, fmt.Errorf("type of column %s.%s is not supported", row.table, column.Name)
		}
		if description != nil {
			column.Description = *description
		}
		if table == nil || row.table != table.name {
			table = &tableSchema{name: row.table}
			tables = append(tables, table)
		}
		table.columns = append(table.columns, column)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tables, nil
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
//
// resolver is a compositeTypeResolver which resolves composite types defined in
// the PostgreSQL schema.
//
// It returns an invalid type if typ is not supported. It returns an error if an
// argument is not valid.
func columnType(column pgTypeInfo, enums map[string]types.Type, resolver compositeTypeResolver, attTypMods map[string]map[string]*int) (types.Type, error) {
	var t types.Type
	switch column.dataType {
	case "smallint":
		t = types.Int16()
	case "integer":
		t = types.Int()
	case "bigint":
		t = types.Int64()
	case "numeric":
		// Parse precision radix.
		if column.radix == nil {
			return types.Type{}, errors.New("numeric_precision_radix value is NULL")
		}
		rdx, _ := strconv.Atoi(*column.radix)
		if rdx != 2 && rdx != 10 {
			return types.Type{}, fmt.Errorf("numeric_precision_radix value %q is not valid", *column.radix)
		}
		// Parse precision.
		if column.precision == nil {
			return types.Type{}, errors.New("numeric_precision value is NULL")
		}
		p, err := strconv.ParseInt(*column.precision, rdx, 64)
		if err != nil || p < 1 {
			return types.Type{}, fmt.Errorf("numeric_precision value %q is not valid", *column.precision)
		}
		// Parse scale.
		if column.scale == nil {
			return types.Type{}, errors.New("numeric_scale value is NULL")
		}
		s, err := strconv.ParseInt(*column.scale, rdx, 64)
		if err != nil || s < 0 || s > p {
			return types.Type{}, fmt.Errorf("numeric_scale value %q is not valid", *column.scale)
		}
		t = types.Decimal(int(p), int(s))
	case "real":
		t = types.Float32()
	case "double precision":
		t = types.Float()
	case "character varying", "character":
		if column.charLength != nil {
			chars, _ := strconv.Atoi(*column.charLength)
			if chars < 1 || chars > types.MaxTextLen {
				return types.Type{}, fmt.Errorf("character_maximum_length value %q is not valid", *column.charLength)
			}
			t = types.Text().WithCharLen(chars)
		} else {
			t = types.Text()
		}
	case "text":
		t = types.Text()
	case "timestamp without time zone", "timestamp with time zone":
		t = types.DateTime().WithLayout("2006-01-02 15:04:05.999999")
	case "date":
		t = types.Date().WithLayout(time.DateOnly)
	case "time without time zone":
		t = types.Time()
	case "boolean":
		t = types.Boolean()
	case "inet":
		t = types.Inet()
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
			et = types.Float32()
		case "_float8":
			et = types.Float()
		case "_inet":
			et = types.Inet()
		case "_int2":
			et = types.Int16()
		case "_int4":
			et = types.Int()
		case "_int8":
			et = types.Int64()
		case "_json", "_jsonb":
			et = types.JSON()
		case "_text":
			et = types.Text()
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
					return types.Type{}, fmt.Errorf("atttypmod value %d is not valid", *attTypMod)
				}
				et = types.Text().WithCharLen(length)
			} else {
				et = types.Text()
			}
		}
		if et.Valid() {
			t = types.Array(et)
		}
	case "USER-DEFINED":
		// Check if the user-defined type is an enum.
		if typ, ok := enums[column.udtName]; ok {
			t = typ
		} else {
			var err error
			t, err = resolver(column.udtName)
			if err != nil {
				return types.Type{}, err
			}
		}
	}
	return t, nil
}

// compositeTypeResolver represents a function which resolves composite types
// defined in the PostgreSQL schema by taking their name and returning the
// types.Type corresponding to the composite type definition. If name does not
// correspond to any composite type, returns types.Type{}, nil.
type compositeTypeResolver func(name string) (types.Type, error)

// initCompositeTypeResolver initializes and returns a resolver for the
// composite types, retrieving information from the transaction tx.
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
func initCompositeTypeResolver(ctx context.Context, tx pgx.Tx, enums map[string]types.Type, attTypMods map[string]map[string]*int) (compositeTypeResolver, error) {

	// Read from the database the information about composite types.

	// TODO(Gianluca): note that this query is executed here instead of being
	// executed on-demand within the 'resolve' function (declared below) because
	// such function is called inside a 'rows.Next()' loop, causing the pgx
	// package to return a 'conn busy' error. Investigate if it is related to
	// some of the issues of the pgx package about the 'conn busy' error:
	// https://github.com/jackc/pgx/issues?q=is%3Aissue+is%3Aopen+%22conn+busy%22

	pgTypesOf := map[string][]pgTypeInfo{}
	query := "SELECT udt_name, attribute_name, data_type, attribute_udt_name,\n" +
		"character_maximum_length, numeric_precision_radix, numeric_scale\n" +
		"FROM information_schema.attributes ORDER BY ordinal_position"
	rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var udtName, attributeName, dataType, attributeUdtName *string
		var pgType pgTypeInfo
		err := rows.Scan(&udtName, &attributeName, &dataType, &attributeUdtName,
			&pgType.charLength, &pgType.precision, &pgType.scale)
		if err != nil {
			return nil, err
		}
		if udtName == nil {
			return nil, errors.New("database has returned NULL as composite type udt name")
		}
		pgType.table = *udtName
		if attributeName == nil {
			return nil, errors.New("database has returned NULL as composite type attribute name")
		}
		pgType.column = *attributeName
		if dataType == nil {
			return nil, errors.New("database has returned NULL as composite type data type")
		}
		pgType.dataType = *dataType
		if attributeUdtName == nil {
			return nil, errors.New("database has returned NULL as composite type attribute udt name")
		}
		pgType.udtName = *attributeUdtName
		pgTypesOf[pgType.table] = append(pgTypesOf[pgType.table], pgType)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	typeOf := map[string]types.Type{}

	var resolve compositeTypeResolver
	resolve = func(name string) (types.Type, error) {
		if typ, ok := typeOf[name]; ok {
			return typ, nil
		}
		rows, ok := pgTypesOf[name]
		if !ok {
			return types.Type{}, nil
		}
		properties := make([]types.Property, len(rows))
		for i, row := range rows {
			typ, err := columnType(row, enums, resolve, attTypMods)
			if err != nil {
				return types.Type{}, err
			}
			if !typ.Valid() {
				return types.Type{}, fmt.Errorf("composite type %q includes field %q with an unsupported type", row.table, row.column)
			}
			properties[i] = types.Property{
				Name: row.column,
				Type: typ,
				// Composite type fields are always nullable.
				// From the PostgreSQL doc (https://www.postgresql.org/docs/current/rowtypes.html):
				//
				//    «[...] no constraints (such as NOT NULL) can presently be included»
				//
				Nullable: true,
			}
		}
		t := types.Object(properties)
		typeOf[name] = t
		return t, nil
	}

	return resolve, nil
}
