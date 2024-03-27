//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"reflect"
	"testing"

	"github.com/open2b/chichi/types"
)

func Test_readPropertyFrom(t *testing.T) {
	cases := []struct {
		m          map[string]any
		prop       types.Path
		expectedV  any
		expectedOk bool
	}{
		{
			m:          map[string]any{},
			prop:       types.Path{"email"},
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"email": "hello@example.com"},
			prop:       types.Path{"email"},
			expectedV:  "hello@example.com",
			expectedOk: true,
		},
		{
			m:          map[string]any{"traits": map[string]any{"email": "world@example.com"}},
			prop:       types.Path{"traits", "email"},
			expectedV:  "world@example.com",
			expectedOk: true,
		},
		{
			m:          map[string]any{"traits": nil},
			prop:       types.Path{"traits", "email"},
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"traits": 42},
			prop:       types.Path{"traits", "email"},
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"traits": map[string]any{"email": "world@example.com"}},
			prop:       types.Path{"traits", "name"},
			expectedV:  nil,
			expectedOk: false,
		},
	}
	for _, cas := range cases {
		t.Run("", func(t *testing.T) {
			gotV, gotOk := readPropertyFrom(cas.m, cas.prop)
			if !reflect.DeepEqual(gotV, cas.expectedV) {
				t.Fatalf("expecting %#v, got %#v", cas.expectedV, gotV)
			}
			if gotOk != cas.expectedOk {
				t.Fatalf("expecting ok = %t, got %t", cas.expectedOk, gotOk)
			}
		})
	}
}
