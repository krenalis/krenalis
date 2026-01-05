// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package parquet

import (
	"fmt"
	"testing"
	"time"
)

func Test_convertInt96(t *testing.T) {
	tests := []struct {
		src         any
		expected    time.Time
		expectedErr string
	}{
		{
			src:         42,
			expectedErr: `expected byte array, got value type "int"`,
		},
		{
			src:         []byte{},
			expectedErr: `expected byte array, got value type "[]uint8"`,
		},
		{
			src:         [0]byte{},
			expectedErr: `unexpected byte array length 0`,
		},
		{
			src:         [8]byte{},
			expectedErr: `unexpected byte array length 8`,
		},
		{
			src:         [100]byte{},
			expectedErr: `unexpected byte array length 100`,
		},
		// These values have been generated in Python with 'pyarrow.parquet'
		// module using the 'use_deprecated_int96_timestamps' option, then read
		// with the 'parquet-tool' tool (https://github.com/fraugster/parquet-go/tree/master/cmd/parquet-tool).
		{
			src:      [...]byte{0, 0, 0, 0, 0, 0, 0, 0, 140, 61, 37, 0},
			expected: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			src:      [...]byte{0, 0, 0, 0, 0, 0, 0, 0, 31, 60, 37, 0},
			expected: time.Date(1969, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			src:      [...]byte{0, 0, 0, 0, 0, 0, 0, 0, 176, 175, 37, 0},
			expected: time.Date(2050, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			src:      [...]byte{0, 112, 193, 104, 30, 11, 0, 0, 167, 232, 37, 0},
			expected: time.Date(2089, 12, 5, 3, 23, 45, 234_432_000, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprint(test.src), func(t *testing.T) {
			got, gotErr := convertInt96(test.src)
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if gotErrStr != test.expectedErr {
				t.Fatalf("expected error `%s`, got `%s`", test.expectedErr, gotErrStr)
			}
			if !got.Equal(test.expected) {
				t.Fatalf("expected %q, got %q", test.expected, got)
			}
		})
	}
}
