//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"database/sql"
	"testing"

	"chichi/apis/types"
)

func TestTypes(t *testing.T) {

	tests := []struct {
		s          string
		t          types.Type
		isNullable sql.NullString
		charLength sql.NullString
		precision  sql.NullString
		radix      sql.NullString
		scale      sql.NullString
	}{
		{`smallint`, types.Int16(), invalid, invalid, invalid, invalid, invalid},
		{`integer`, types.Int(), invalid, invalid, invalid, invalid, invalid},
		{`bigint`, types.Int64(), invalid, invalid, invalid, invalid, invalid},
		{`numeric`, types.Decimal(10, 3), invalid, invalid, valid("10"), valid("10"), valid("3")},
		{`real`, types.Float32(), invalid, invalid, invalid, invalid, invalid},
		{`double precision`, types.Float().WithNull(), valid("YES"), invalid, invalid, invalid, invalid},
		{`character varying`, types.Text(types.Chars(20)), invalid, valid("20"), invalid, invalid, invalid},
		{`character`, types.Text(types.Chars(8)), invalid, valid("8"), invalid, invalid, invalid},
		{`text`, types.Text(), invalid, invalid, invalid, invalid, invalid},
		{`timestamp without time zone`, types.DateTime("2006-01-02 15:04:05.999999"), invalid, invalid, invalid, invalid, invalid},
		{`timestamp with time zone`, types.DateTime("2006-01-02 15:04:05.999999"), invalid, invalid, invalid, invalid, invalid},
		{`date`, types.Date("2006-01-02"), invalid, invalid, invalid, invalid, invalid},
		{`time without time zone`, types.Time("15:04:05"), invalid, invalid, invalid, invalid, invalid},
		{`time with time zone`, types.Time("15:04:05"), invalid, invalid, invalid, invalid, invalid},
		{`boolean`, types.Boolean().WithNull(), valid("YES"), invalid, invalid, invalid, invalid},
		{`uuid`, types.UUID(), invalid, invalid, invalid, invalid, invalid},
		{`json`, types.JSON(), invalid, invalid, invalid, invalid, invalid},
		{`jsonb`, types.JSON(), invalid, invalid, invalid, invalid, invalid},
	}

	for _, test := range tests {
		got, err := columnType(test.s, test.isNullable, test.charLength, test.precision, test.radix, test.scale)
		if err != nil {
			t.Error(err)
		}
		if got.Valid() != test.t.Valid() {
			if test.t.Valid() {
				t.Errorf("%s: expecting a valid type, got an invalid type", test.s)
			}
			t.Errorf("%s: expecting an invalid type, got a valid type: %#v", test.s, got)
		}
		if !got.EqualTo(test.t) {
			t.Errorf("%s: unexpected type: %#v", test.s, got)
		}
	}
}

func TestUnsupportedTypes(t *testing.T) {

	tests := []string{
		// Unsupported types.
		`money`, `bytea`, `point`, `line`, `lseg`, `box`, `path`, `polygon`, `circle`,

		// Invalid types.
		``, ` `, "\x00", "\xFF", `a`,
	}

	for _, test := range tests {
		got, err := columnType(test, invalid, invalid, invalid, invalid, invalid)
		if err != nil {
			t.Error(err)
		}
		if got.Valid() {
			t.Errorf("%s: expecting an invalid type, got a valid type: %#v", test, got)
		}
	}
}

// valid returns a valid sql.NullString with value s.
func valid(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

// invalid represents an invalid sql.NullString.
var invalid sql.NullString
