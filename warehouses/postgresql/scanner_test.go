// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"encoding/binary"
	"errors"
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/tools/types"

	"github.com/jackc/pgx/v5/pgtype"
)

// TestScanArrayInvalidElementLength keeps negative element lengths invalid.
func TestScanArrayInvalidElementLength(t *testing.T) {
	for _, length := range []int32{-1, -2} {
		data, err := pgtype.NewMap().Encode(pgtype.TextArrayOID, pgtype.BinaryFormatCode, []string{"x"}, nil)
		if err != nil {
			t.Fatalf("expected binary array fixture, got %v", err)
		}
		binary.BigEndian.PutUint32(data[20:], uint32(length))
		s := &scanner{}
		_, err = s.normalize("value", types.Array(types.String()), data)
		if err != nil {
			if !errors.Is(err, errInvalidData) {
				t.Fatalf("expected invalid-data error for length %d, got %v", length, err)
			}
		}
		if err == nil {
			t.Fatalf("expected invalid-data error for length %d, got success", length)
		}
	}
}

// TestScanArrayRecord bounds binary lengths without a Unified byte budget and
// preserves the existing treatment of ordinary arrays and SQL NULL elements.
func TestScanArrayRecord(t *testing.T) {
	first, second, third := "first", "second", "third"
	for _, test := range []struct {
		name     string
		elements []*string
		declared int32
		want     []any
		invalid  bool
	}{
		{"empty", []*string{}, 0, []any{}, false},
		{"multiple elements", []*string{&first, &second, &third}, 0, []any{"first", "second", "third"}, false},
		{"SQL NULL element remains rejected", []*string{nil}, 0, nil, true},
		{"SQL NULL between values remains rejected", []*string{&first, nil, &third}, 0, nil, true},
		{"negative length", []*string{&first}, -1, nil, true},
		{"length beyond available bytes", []*string{&first}, 128, nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := pgtype.NewMap().Encode(pgtype.TextArrayOID, pgtype.BinaryFormatCode, test.elements, nil)
			if err != nil {
				t.Fatalf("expected binary array fixture, got %v", err)
			}
			if test.declared != 0 {
				binary.BigEndian.PutUint32(data[12:], uint32(test.declared))
			}
			defer func() {
				if value := recover(); value != nil {
					t.Fatalf("expected array validation without panic, got %v", value)
				}
			}()
			s := &scanner{}
			got, err := s.normalize("value", types.Array(types.String()), data)
			if err != nil {
				if !test.invalid || !errors.Is(err, errInvalidData) {
					t.Fatalf("expected invalid-data error for invalid=%t, got %v", test.invalid, err)
				}
			}
			if test.invalid {
				if err == nil {
					t.Fatal("expected invalid-data error, got success")
				}
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("expected decoded array %v, got %v", test.want, got)
			}
		})
	}
}

// TestScanStringArray preserves empty String elements and their positions.
func TestScanStringArray(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []any
	}{
		{"empty array", []any{}},
		{"nonempty elements", []any{"first", "second"}},
		{"empty element", []any{""}},
		{"empty first element", []any{"", "x"}},
		{"empty last element", []any{"x", ""}},
		{"only empty elements", []any{"", ""}},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, oid := range []uint32{pgtype.TextArrayOID, pgtype.BPCharArrayOID, pgtype.VarcharArrayOID} {
				data, err := pgtype.NewMap().Encode(oid, pgtype.BinaryFormatCode, test.values, nil)
				if err != nil {
					t.Fatalf("expected binary String array for OID %d, got %v", oid, err)
				}
				s := &scanner{}
				got, err := s.normalize("value", types.Array(types.String()), data)
				if err != nil {
					t.Fatalf("expected String array for OID %d, got %v", oid, err)
				}
				if !reflect.DeepEqual(got, test.values) {
					t.Fatalf("expected String array %#v for OID %d, got %#v", test.values, oid, got)
				}
			}
		})
	}
}
