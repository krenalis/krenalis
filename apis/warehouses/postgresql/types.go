//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package postgresql

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"chichi/apis/types"
)

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
// ctResolver is a types.Resolver which resolves composite types.
//
// It returns an invalid type if typ is not supported. It returns an error if an
// argument is not valid.
func columnType(column pgTypeInfo, enums map[string]types.Type, ctResolver types.Resolver, attTypMods map[string]map[string]*int) (types.Type, error) {
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
			if chars < 1 {
				return types.Type{}, fmt.Errorf("character_maximum_length value %q is not valid", *column.charLength)
			}
			t = types.Text(types.Chars(chars))
		} else {
			t = types.Text()
		}
	case "text":
		t = types.Text()
	case "timestamp without time zone", "timestamp with time zone":
		t = types.DateTime("2006-01-02 15:04:05.999999")
	case "date":
		t = types.Date(time.DateOnly)
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
		switch column.udtName {
		case "_bool":
			t = types.Array(types.Boolean())
		case "_int4":
			t = types.Array(types.Int())
		case "_varchar":
			attTypMod := attTypMods[column.table][column.column]
			if attTypMod != nil {
				length := *attTypMod - 4 // See the function "_pg_char_max_length".
				if length < 1 {
					return types.Type{}, fmt.Errorf("atttypmod value %d is not valid", *attTypMod)
				}
				t = types.Text(types.Chars(length))
			} else {
				t = types.Text()
			}
			t = types.Array(t)
		}
	case "USER-DEFINED":
		// Check if the user-defined type is an enum.
		if typ, ok := enums[column.udtName]; ok {
			t = typ
		} else {
			var err error
			t, err = ctResolver(column.udtName)
			if err != nil {
				return types.Type{}, err
			}
		}
	}
	return t, nil
}
