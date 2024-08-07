//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"reflect"
	"strings"
	"testing"
)

func Test_readPropertyFrom(t *testing.T) {
	cases := []struct {
		m          map[string]any
		prop       string
		expectedV  any
		expectedOk bool
	}{
		{
			m:          map[string]any{},
			prop:       "email",
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"email": "hello@example.com"},
			prop:       "email",
			expectedV:  "hello@example.com",
			expectedOk: true,
		},
		{
			m:          map[string]any{"traits": map[string]any{"email": "world@example.com"}},
			prop:       "traits.email",
			expectedV:  "world@example.com",
			expectedOk: true,
		},
		{
			m:          map[string]any{"traits": nil},
			prop:       "traits.email",
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"traits": 42},
			prop:       "traits.email",
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"traits": map[string]any{"email": "world@example.com"}},
			prop:       "traits.name",
			expectedV:  nil,
			expectedOk: false,
		},
	}
	for _, cas := range cases {
		t.Run("", func(t *testing.T) {
			gotV, gotOk := readPropertyFrom(cas.m, cas.prop)
			if !reflect.DeepEqual(gotV, cas.expectedV) {
				t.Fatalf("expected %#v, got %#v", cas.expectedV, gotV)
			}
			if gotOk != cas.expectedOk {
				t.Fatalf("expected ok = %t, got %t", cas.expectedOk, gotOk)
			}
		})
	}
}

// readPropertyFrom reads the property with the given path from m, returning its
// value (if found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
func readPropertyFrom(m map[string]any, path string) (any, bool) {
	var name string
	for {
		name, path, _ = strings.Cut(path, ".")
		v, ok := m[name]
		if !ok {
			return nil, false
		}
		if path == "" {
			return v, true
		}
		m, ok = v.(map[string]any)
		if !ok {
			return nil, false
		}
	}
}
