//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"reflect"
	"testing"
)

func Test_readPropertyFrom(t *testing.T) {
	cases := []struct {
		m          map[string]any
		prop       []string
		expectedV  any
		expectedOk bool
	}{
		{
			m:          map[string]any{},
			prop:       []string{"email"},
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"email": "hello@example.com"},
			prop:       []string{"email"},
			expectedV:  "hello@example.com",
			expectedOk: true,
		},
		{
			m:          map[string]any{"traits": map[string]any{"email": "world@example.com"}},
			prop:       []string{"traits", "email"},
			expectedV:  "world@example.com",
			expectedOk: true,
		},
		{
			m:          map[string]any{"traits": nil},
			prop:       []string{"traits", "email"},
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"traits": 42},
			prop:       []string{"traits", "email"},
			expectedV:  nil,
			expectedOk: false,
		},
		{
			m:          map[string]any{"traits": map[string]any{"email": "world@example.com"}},
			prop:       []string{"traits", "name"},
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

func Test_writePropertyTo(t *testing.T) {
	cases := []struct {
		m         map[string]any
		prop      []string
		v         any
		expectedM map[string]any
	}{
		{
			m:         map[string]any{},
			prop:      []string{"Email"},
			v:         "test@example.com",
			expectedM: map[string]any{"Email": "test@example.com"},
		},
		{
			m:         map[string]any{},
			prop:      []string{"User", "Email"},
			v:         "test@example.com",
			expectedM: map[string]any{"User": map[string]any{"Email": "test@example.com"}},
		},
		{
			m:         map[string]any{"User": nil},
			prop:      []string{"User", "Email"},
			v:         "test@example.com",
			expectedM: map[string]any{"User": nil},
		},
		{
			m:         map[string]any{"User": 42},
			prop:      []string{"User", "Email"},
			v:         "test@example.com",
			expectedM: map[string]any{"User": 42},
		},
	}
	for _, cas := range cases {
		t.Run("", func(t *testing.T) {
			writePropertyTo(cas.m, cas.prop, cas.v)
			if !reflect.DeepEqual(cas.m, cas.expectedM) {
				t.Fatalf("expecting %#v, got %#v", cas.expectedM, cas.m)
			}
		})
	}
}
