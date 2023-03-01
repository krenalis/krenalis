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

// columnType returns the types.Type corresponding to the PostgreSQL type typ
// stored in the information_schema.columns column.
//
// udtName is the name of the column data type, which is relevant in case of
// user-defined types and arrays.
//
// attTypMod, when non-nil, is a type-specific data attribute read from
// pg_attribute.atttypmod (represents, for example, the maximum length of a
// varchar column or the maximum length of the text of an array element); is nil
// if the column type has no associated type-specific data.
//
// enums is a mapping of available enum types.
//
// It returns an invalid type if typ is not supported. It returns an error if an
// argument is not valid.
func columnType(typ, udtName string, charLength, precision, radix, scale *string, attTypMod *int, enums map[string]types.Type) (types.Type, error) {
	var t types.Type
	switch typ {
	case "smallint":
		t = types.Int16()
	case "integer":
		t = types.Int()
	case "bigint":
		t = types.Int64()
	case "numeric":
		// Parse precision radix.
		if radix == nil {
			return types.Type{}, errors.New("numeric_precision_radix value is NULL")
		}
		rdx, _ := strconv.Atoi(*radix)
		if rdx != 2 && rdx != 10 {
			return types.Type{}, fmt.Errorf("numeric_precision_radix value %q is not valid", *radix)
		}
		// Parse precision.
		if precision == nil {
			return types.Type{}, errors.New("numeric_precision value is NULL")
		}
		p, err := strconv.ParseInt(*precision, rdx, 64)
		if err != nil || p < 1 {
			return types.Type{}, fmt.Errorf("numeric_precision value %q is not valid", *precision)
		}
		// Parse scale.
		if scale == nil {
			return types.Type{}, errors.New("numeric_scale value is NULL")
		}
		s, err := strconv.ParseInt(*scale, rdx, 64)
		if err != nil || s < 0 || s > p {
			return types.Type{}, fmt.Errorf("numeric_scale value %q is not valid", *scale)
		}
		t = types.Decimal(int(p), int(s))
	case "real":
		t = types.Float32()
	case "double precision":
		t = types.Float()
	case "character varying", "character":
		if charLength != nil {
			chars, _ := strconv.Atoi(*charLength)
			if chars < 1 {
				return types.Type{}, fmt.Errorf("character_maximum_length value %q is not valid", *charLength)
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
		switch udtName {
		case "_bool":
			t = types.Array(types.Boolean())
		case "_int4":
			t = types.Array(types.Int())
		case "_varchar":
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
		if typ, ok := enums[udtName]; ok {
			t = typ
		}
	}
	return t, nil
}
