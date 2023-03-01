//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"testing"
	"time"

	"chichi/apis/types"
)

func TestTypes(t *testing.T) {

	tests := []struct {
		s          string
		udtName    string
		t          types.Type
		charLength *string
		precision  *string
		radix      *string
		scale      *string
	}{
		{`smallint`, "", types.Int16(), nil, nil, nil, nil},
		{`integer`, "", types.Int(), nil, nil, nil, nil},
		{`bigint`, "", types.Int64(), nil, nil, nil, nil},
		{`numeric`, "", types.Decimal(10, 3), nil, pointer("10"), pointer("10"), pointer("3")},
		{`real`, "", types.Float32(), nil, nil, nil, nil},
		{`double precision`, "", types.Float(), nil, nil, nil, nil},
		{`character varying`, "", types.Text(types.Chars(20)), pointer("20"), nil, nil, nil},
		{`character`, "", types.Text(types.Chars(8)), pointer("8"), nil, nil, nil},
		{`text`, "", types.Text(), nil, nil, nil, nil},
		{`timestamp without time zone`, "", types.DateTime("2006-01-02 15:04:05.999999"), nil, nil, nil, nil},
		{`timestamp with time zone`, "", types.DateTime("2006-01-02 15:04:05.999999"), nil, nil, nil, nil},
		{`date`, "", types.Date(time.DateOnly), nil, nil, nil, nil},
		{`time without time zone`, "", types.Time(), nil, nil, nil, nil},
		{`boolean`, "", types.Boolean(), nil, nil, nil, nil},
		{`uuid`, "", types.UUID(), nil, nil, nil, nil},
		{`json`, "", types.JSON(), nil, nil, nil, nil},
		{`jsonb`, "", types.JSON(), nil, nil, nil, nil},
		{`ARRAY`, `_int4`, types.Array(types.Int()), nil, nil, nil, nil},
		{`ARRAY`, `_varchar`, types.Array(types.Text()), nil, nil, nil, nil},
		{`ARRAY`, `_bool`, types.Array(types.Boolean()), nil, nil, nil, nil},
	}

	for _, test := range tests {
		got, err := columnType(test.s, test.udtName, test.charLength, test.precision, test.radix, test.scale, nil, nil)
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
		got, err := columnType(test, "", nil, nil, nil, nil, nil, nil)
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
