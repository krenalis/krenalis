//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2023 Open2b
//

package transformations_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"chichi/apis/transformations"
)

func TestRun(t *testing.T) {
	type test struct {
		code     string
		input    []any
		expected []any
	}
	cases := []test{
		{
			code:     "def f(a): return a",
			input:    []any{42},
			expected: []any{42.0},
		},
		{
			code:     "def f(a, b): return a + b, a - b",
			input:    []any{100, 20},
			expected: []any{120.0, 80.0},
		},
	}
	for _, cas := range cases {
		t.Run("", func(t *testing.T) {
			pool := transformations.NewPool()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			got, err := pool.Run(ctx, cas.code, cas.input)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(cas.expected, got) {
				t.Fatalf("expecting %#v, got %#v", cas.expected, got)
			}
		})
	}
}
