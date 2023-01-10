//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package postgresql

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"chichi/apis/types"
)

// columnType returns the types.Type corresponding to the PostgreSQL type typ
// stored in the information_schema.columns column. It returns an invalid type
// if typ is not supported. It returned an error if an argument is not valid.
func columnType(typ string, isNullable, charLength, precision, radix, scale sql.NullString) (types.Type, error) {
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
		if !radix.Valid {
			return types.Type{}, errors.New("numeric_precision_radix value is NULL")
		}
		rdx, _ := strconv.Atoi(radix.String)
		if rdx != 2 && rdx != 10 {
			return types.Type{}, fmt.Errorf("numeric_precision_radix value %q is not valid", radix.String)
		}
		// Parse precision.
		if !precision.Valid {
			return types.Type{}, errors.New("numeric_precision value is NULL")
		}
		p, err := strconv.ParseInt(precision.String, rdx, 64)
		if err != nil || p < 1 {
			return types.Type{}, fmt.Errorf("numeric_precision value %q is not valid", precision.String)
		}
		// Parse scale.
		if !scale.Valid {
			return types.Type{}, errors.New("numeric_scale value is NULL")
		}
		s, err := strconv.ParseInt(scale.String, rdx, 64)
		if err != nil || s < 0 || s > p {
			return types.Type{}, fmt.Errorf("numeric_scale value %q is not valid", scale.String)
		}
		t = types.Decimal(int(p), int(s))
	case "real":
		t = types.Float32()
	case "double precision":
		t = types.Float()
	case "character varying", "character":
		if charLength.Valid {
			chars, _ := strconv.Atoi(charLength.String)
			if chars < 1 {
				return types.Type{}, fmt.Errorf("character_maximum_length value %q is not valid", charLength.String)
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
		t = types.Date("2006-01-02")
	case "time without time zone", "time with time zone":
		t = types.Time("15:04:05")
	case "boolean":
		t = types.Boolean()
	case "uuid":
		t = types.UUID()
	case "json", "jsonb":
		t = types.JSON()
	}
	if t.Valid() && isNullable.String == "YES" {
		t = t.WithNull()
	}
	return t, nil
}
