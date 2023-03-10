//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"context"
	"errors"

	"chichi/apis/postgres"
	"chichi/apis/types"
)

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
func initCompositeTypeResolver(ctx context.Context, tx *postgres.Tx, enums map[string]types.Type, attTypMods map[string]map[string]*int) (compositeTypeResolver, error) {

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
			return nil, errors.New("data warehouse has returned NULL as composite type udt name")
		}
		pgType.table = *udtName
		if attributeName == nil {
			return nil, errors.New("data warehouse has returned NULL as composite type attribute name")
		}
		pgType.column = *attributeName
		if dataType == nil {
			return nil, errors.New("data warehouse has returned NULL as composite type data type")
		}
		pgType.dataType = *dataType
		if attributeUdtName == nil {
			return nil, errors.New("data warehouse has returned NULL as composite type attribute udt name")
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
		if rows, ok := pgTypesOf[name]; ok {
			props := []types.Property{}
			for _, row := range rows {
				typ, err := columnType(row, enums, resolve, attTypMods)
				if err != nil {
					return types.Type{}, err
				}
				prop := types.Property{
					Name: row.column,
					Type: typ,
					// Composite type fields are always nullable.
					// From the PostgreSQL doc (https://www.postgresql.org/docs/current/rowtypes.html):
					//
					//    «[...] no constraints (such as NOT NULL) can presently be included»
					//
					Nullable: true,
				}
				props = append(props, prop)
			}
			t := types.Object(props)
			// TODO(Gianluca): should we call 'AsCustom(name)' here?
			typeOf[name] = t
			return t, nil
		}
		return types.Type{}, nil
	}

	return resolve, nil
}
