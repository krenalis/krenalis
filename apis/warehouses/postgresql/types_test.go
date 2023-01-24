//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"testing"

	"chichi/apis/types"
)

func TestTypes(t *testing.T) {

	tests := []struct {
		s          string
		udtName    string
		t          types.Type
		isNullable *string
		charLength *string
		precision  *string
		radix      *string
		scale      *string
	}{
		{`smallint`, "", types.Int16(), pointer("NO"), nil, nil, nil, nil},
		{`integer`, "", types.Int(), pointer("NO"), nil, nil, nil, nil},
		{`bigint`, "", types.Int64(), pointer("NO"), nil, nil, nil, nil},
		{`numeric`, "", types.Decimal(10, 3), pointer("NO"), nil, pointer("10"), pointer("10"), pointer("3")},
		{`real`, "", types.Float32(), pointer("NO"), nil, nil, nil, nil},
		{`double precision`, "", types.Float().WithNull(), pointer("YES"), nil, nil, nil, nil},
		{`character varying`, "", types.Text(types.Chars(20)), pointer("NO"), pointer("20"), nil, nil, nil},
		{`character`, "", types.Text(types.Chars(8)), pointer("NO"), pointer("8"), nil, nil, nil},
		{`text`, "", types.Text(), pointer("NO"), nil, nil, nil, nil},
		{`timestamp without time zone`, "", types.DateTime("2006-01-02 15:04:05.999999"), pointer("NO"), nil, nil, nil, nil},
		{`timestamp with time zone`, "", types.DateTime("2006-01-02 15:04:05.999999"), pointer("NO"), nil, nil, nil, nil},
		{`date`, "", types.Date("2006-01-02"), pointer("NO"), nil, nil, nil, nil},
		{`time without time zone`, "", types.Time("15:04:05"), pointer("NO"), nil, nil, nil, nil},
		{`time with time zone`, "", types.Time("15:04:05"), pointer("NO"), nil, nil, nil, nil},
		{`boolean`, "", types.Boolean().WithNull(), pointer("YES"), nil, nil, nil, nil},
		{`uuid`, "", types.UUID(), pointer("NO"), nil, nil, nil, nil},
		{`json`, "", types.JSON(), pointer("NO"), nil, nil, nil, nil},
		{`jsonb`, "", types.JSON(), pointer("NO"), nil, nil, nil, nil},
		{`ARRAY`, `_int4`, types.Array(types.Int()), pointer("NO"), nil, nil, nil, nil},
		{`ARRAY`, `_varchar`, types.Array(types.Text()), pointer("NO"), nil, nil, nil, nil},
		{`ARRAY`, `_bool`, types.Array(types.Boolean()), pointer("NO"), nil, nil, nil, nil},
	}

	for _, test := range tests {
		got, err := columnType(test.s, test.udtName, test.isNullable, test.charLength, test.precision, test.radix, test.scale, nil)
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
		got, err := columnType(test, "", pointer("NO"), nil, nil, nil, nil, nil)
		if err != nil {
			t.Error(err)
		}
		if got.Valid() {
			t.Errorf("%s: expecting an invalid type, got a valid type: %#v", test, got)
		}
	}
}

// pointer returns a pointer to s.
func pointer(s string) *string {
	return &s
}
