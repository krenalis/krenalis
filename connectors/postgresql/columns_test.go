// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"fmt"
	"testing"

	"github.com/meergo/meergo/tools/types"
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
		{`smallint`, "", types.Int(16), nil, nil, nil, nil},
		{`integer`, "", types.Int(32), nil, nil, nil, nil},
		{`bigint`, "", types.Int(64), nil, nil, nil, nil},
		{`numeric`, "", types.Decimal(10, 3), nil, new("10"), new("10"), new("3")},
		{`real`, "", types.Float(32), nil, nil, nil, nil},
		{`double precision`, "", types.Float(64), nil, nil, nil, nil},
		{`character varying`, "", types.String().WithMaxLength(20), new("20"), nil, nil, nil},
		{`character`, "", types.String().WithMaxLength(8), new("8"), nil, nil, nil},
		{`text`, "", types.String(), nil, nil, nil, nil},
		{`timestamp without time zone`, "", types.DateTime(), nil, nil, nil, nil},
		{`timestamp with time zone`, "", types.DateTime(), nil, nil, nil, nil},
		{`date`, "", types.Date(), nil, nil, nil, nil},
		{`time without time zone`, "", types.Time(), nil, nil, nil, nil},
		{`boolean`, "", types.Boolean(), nil, nil, nil, nil},
		{`uuid`, "", types.UUID(), nil, nil, nil, nil},
		{`json`, "", types.JSON(), nil, nil, nil, nil},
		{`jsonb`, "", types.JSON(), nil, nil, nil, nil},
		{`ARRAY`, `_int4`, types.Array(types.Int(32)), nil, nil, nil, nil},
		{`ARRAY`, `_varchar`, types.Array(types.String()), nil, nil, nil, nil},
		{`ARRAY`, `_bool`, types.Array(types.Boolean()), nil, nil, nil, nil},
	}

	for _, test := range tests {
		row := pgTypeInfo{
			table:      "test_table",
			column:     "test_column",
			dataType:   test.s,
			udtName:    test.udtName,
			charLength: test.charLength,
			precision:  test.precision,
			radix:      test.radix,
			scale:      test.scale,
		}
		got, _, err := columnType(row, nil, nil)
		if err != nil {
			t.Error(err)
		}
		if got.Valid() != test.t.Valid() {
			if test.t.Valid() {
				t.Errorf("%s: expected a valid type, got an invalid type", test.s)
			}
			t.Errorf("%s: expected an invalid type, got a valid type: %#v", test.s, got)
		}
		if !types.Equal(got, test.t) {
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

	for i, test := range tests {
		t.Run(test, func(t *testing.T) {
			row := pgTypeInfo{
				dataType: test,
				column:   fmt.Sprintf("c%d", i),
			}
			got, issue, err := columnType(row, nil, nil)
			if err != nil {
				t.Fatalf("expected no error, got error '%v'", err)
			}
			expected := fmt.Sprintf("Column %q has an unsupported type %q.", row.column, row.dataType)
			if expected != issue {
				t.Fatalf("expected issue %q, got issue %q", expected, issue)
			}
			if got.Valid() {
				t.Fatalf("expected an invalid type, got %s", got)
			}
		})
	}
}

// pointer returns a pointer to s.
//
//go:fix inline
func pointer(s string) *string {
	return new(s)
}
