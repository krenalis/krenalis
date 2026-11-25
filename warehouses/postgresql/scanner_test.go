// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"slices"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/jackc/pgx/v5/pgtype"
)

// TestScannerArrayBounds checks that array bounds are validated before decoding
// elements.
func TestScannerArrayBounds(t *testing.T) {

	tm := pgtype.NewMap()
	encodeArray := func(t *testing.T, elements any) []byte {
		t.Helper()
		data, err := tm.Encode(pgtype.JSONArrayOID, pgtype.BinaryFormatCode, elements, nil)
		if err != nil {
			t.Fatalf("expected binary array, got %v", err)
		}
		return data
	}
	withInt32 := func(src []byte, offset int, value int32) []byte {
		t.Helper()
		if offset < 0 || offset > len(src)-4 {
			t.Fatalf("expected four bytes at offset %d, got a %d-byte fixture", offset, len(src))
		}
		data := slices.Clone(src)
		binary.BigEndian.PutUint32(data[offset:], uint32(value))
		return data
	}
	// The 12-byte header is followed by an 8-byte dimension (count and lower bound),
	// so the first element's length starts at offset 20.
	const (
		dimensionCountOffset = 0
		elementCountOffset   = 12
		elementLengthOffset  = 20
	)
	data := encodeArray(t, []string{`1`})
	empty := withInt32(data[:elementCountOffset], dimensionCountOffset, 0)
	excessCount := withInt32(data, elementCountOffset, 2)
	zeroLengthElement := withInt32(data, elementLengthOffset, 0)
	matrix := encodeArray(t, pgtype.Array[string]{
		Elements: []string{`1`},
		Dims:     []pgtype.ArrayDimension{{Length: 1, LowerBound: 1}, {Length: 1, LowerBound: 1}},
		Valid:    true,
	})

	rangeError := func(count, min, max int) error {
		return fmt.Errorf("data warehouse returned an array with %d elements for column value, "+
			"which is not within the expected range of [%d, %d]", count, min, max)
	}

	tests := []struct {
		name string
		data []byte
		min  int
		max  int
		want []any
		err  error
	}{
		{name: "empty", data: empty, max: 0, want: []any{}},
		{name: "at both bounds", data: data, min: 1, max: 1, want: []any{json.Value(`1`)}},
		{name: "within bounds", data: data, min: 0, max: 2, want: []any{json.Value(`1`)}},
		{name: "truncated array header", data: data[:elementCountOffset-1], max: 2, err: errInvalidData},
		{name: "truncated dimension", data: data[:elementLengthOffset-1], max: 2, err: errInvalidData},
		{name: "two dimensions", data: matrix, max: 2, err: errors.New("type is not supported")},
		{name: "negative dimensions", data: withInt32(empty, dimensionCountOffset, -1), max: 2, err: errInvalidData},
		{name: "negative count", data: withInt32(data, elementCountOffset, -1), max: 2, err: errInvalidData},
		{name: "count exceeds available bytes", data: excessCount, max: 2, err: errInvalidData},
		{name: "zero-length JSON element", data: zeroLengthElement[:elementLengthOffset+4], max: 2, err: errInvalidData},
		{name: "zero-length JSON element with trailing data", data: zeroLengthElement, max: 2, err: errInvalidData},
		{name: "element length exceeds available bytes", data: withInt32(data, elementLengthOffset, 2), max: 2, err: errInvalidData},
		{name: "above maximum", data: zeroLengthElement, max: 0, err: rangeError(1, 0, 0)},
		{name: "below minimum", data: zeroLengthElement, min: 2, max: 2, err: rangeError(1, 2, 2)},
		{name: "empty below minimum", data: empty, min: 2, max: 2, err: rangeError(0, 2, 2)},
		{name: "one element below minimum", data: data, min: 2, max: 2, err: rangeError(1, 2, 2)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			typ := types.Array(types.JSON()).WithMaxElements(test.max).WithMinElements(test.min)
			s := newScanner([]warehouses.Column{{Name: "value", Type: typ}}, nil)
			s.dest = make([]any, 1)
			err := tm.Scan(pgtype.JSONArrayOID, pgtype.BinaryFormatCode, test.data, s.values[0])
			checkArrayScanError(t, err, test.err)
			if test.err == nil && !reflect.DeepEqual(s.dest[0], test.want) {
				t.Fatalf("expected %#v, got %#v", test.want, s.dest[0])
			}
		})
	}

	s := &scanner{}
	typ := types.Array(types.JSON()).WithMaxElements(2)
	t.Run("count exceeds available bytes without allocating", func(t *testing.T) {
		var src any = excessCount
		var err error
		allocs := testing.AllocsPerRun(100, func() {
			_, err = s.scanArray("value", typ, src)
		})
		checkArrayScanError(t, err, errInvalidData)
		if allocs != 0 {
			t.Fatalf("expected 0 allocations, got %v", allocs)
		}
	})

	t.Run("above maximum without allocating the result slice", func(t *testing.T) {
		const count = 4096
		const runs = 100
		var src any = encodeArray(t, slices.Repeat([]string{`1`}, count))
		var err error
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		for range runs {
			_, err = s.scanArray("value", typ, src)
		}
		runtime.ReadMemStats(&after)
		checkArrayScanError(t, err, rangeError(count, 0, 2))
		// The 16 KiB limit leaves room to format the error. The result slice alone
		// would require at least 32 KiB.
		const maxBytesPerScan = 16 << 10
		bytesPerScan := (after.TotalAlloc - before.TotalAlloc) / runs
		if bytesPerScan > maxBytesPerScan {
			t.Fatalf("expected at most %d bytes per scan, got %d", maxBytesPerScan, bytesPerScan)
		}
	})

}

// TestScannerStringArrayEmptyElements checks that the scanner
// correctly handles empty strings in binary PostgreSQL arrays.
func TestScannerStringArrayEmptyElements(t *testing.T) {

	tm := pgtype.NewMap()
	typ := types.Array(types.String())
	tests := []struct {
		name string
		want []any
	}{
		{"empty_array", []any{}},
		{"empty_element", []any{""}},
		{"trailing_empty", []any{"a", ""}},
		{"leading_empty", []any{"", "ab"}},
		{"middle_empty", []any{"a", "", "ab"}},
		{"all_empty", []any{"", ""}},
	}

	for _, oid := range []uint32{pgtype.TextArrayOID, pgtype.VarcharArrayOID, pgtype.BPCharArrayOID} {
		for _, test := range tests {
			t.Run(fmt.Sprintf("%d/%s", oid, test.name), func(t *testing.T) {
				data, err := tm.Encode(oid, pgtype.BinaryFormatCode, test.want, nil)
				if err != nil {
					t.Fatalf("expected encoding to succeed, got %v", err)
				}
				s := &scanner{}
				got, err := s.normalize("value", typ, data)
				if err != nil {
					t.Fatalf("expected normalization to succeed, got %v", err)
				}
				if !reflect.DeepEqual(got, test.want) {
					t.Errorf("expected %#v, got %#v", test.want, got)
				}
			})
		}
	}

}

// checkArrayScanError matches errInvalidData by identity and other errors by
// their message.
func checkArrayScanError(t *testing.T, err, want error) {
	t.Helper()
	if err != nil {
		if err == want {
			return
		}
		if want != nil && want != errInvalidData && err.Error() == want.Error() {
			return
		}
		t.Fatalf("expected %v, got %v", want, err)
	}
	if want != nil {
		t.Fatalf("expected %v, got nil", want)
	}
}
