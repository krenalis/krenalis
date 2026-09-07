// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestSuitableAsIdentifier verifies which types are suitable as identifiers.
func TestSuitableAsIdentifier(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{name: "string", typ: types.String(), want: true},
		{name: "boolean", typ: types.Boolean(), want: false},
		{name: "int16", typ: types.Int(16), want: true},
		{name: "int32", typ: types.Int(32), want: true},
		{name: "int64", typ: types.Int(64), want: true},
		{name: "uint8", typ: types.Int(8).Unsigned(), want: true},
		{name: "uint32", typ: types.Int(32).Unsigned(), want: true},
		{name: "float32", typ: types.Float(32), want: false},
		{name: "float64", typ: types.Float(64), want: false},
		{name: "integer decimal", typ: types.Decimal(10, 0), want: true},
		{name: "fractional decimal", typ: types.Decimal(10, 3), want: false},
		{name: "fractional decimal without integer digits", typ: types.Decimal(3, 3), want: false},
		{name: "datetime", typ: types.DateTime(), want: false},
		{name: "date", typ: types.Date(), want: false},
		{name: "time", typ: types.Time(), want: false},
		{name: "year", typ: types.Year(), want: false},
		{name: "UUID", typ: types.UUID(), want: true},
		{name: "IP", typ: types.IP(), want: true},
		{name: "string array", typ: types.Array(types.String()), want: false},
		{name: "float array", typ: types.Array(types.Float(32)), want: false},
		{name: "decimal array", typ: types.Array(types.Decimal(10, 0)), want: false},
		{name: "nested array", typ: types.Array(types.Array(types.String())), want: false},
		{
			name: "object",
			typ:  types.Object([]types.Property{{Name: "a", Type: types.String()}}),
			want: false,
		},
		{name: "map", typ: types.Map(types.String()), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := suitableAsIdentifier(test.typ)
			if got != test.want {
				t.Errorf("suitableAsIdentifier(%v) = %t, want %t", test.typ, got, test.want)
			}
		})
	}
}

// TestWorkspaceEncodeJSON verifies that json.Encode preserves the distinction
// between nil and empty Workspace collection fields.
func TestWorkspaceEncodeJSON(t *testing.T) {
	tests := []struct {
		name               string
		workspace          Workspace
		wantPrimarySources string
		wantIdentifiers    string
	}{
		{
			name:               "nil collections",
			workspace:          Workspace{},
			wantPrimarySources: "null",
			wantIdentifiers:    "null",
		},
		{
			name: "empty collections",
			workspace: Workspace{
				PrimarySources: map[string]string{},
				Identifiers:    []string{},
			},
			wantPrimarySources: "{}",
			wantIdentifiers:    "[]",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var encoded bytes.Buffer
			err := json.Encode(&encoded, test.workspace)
			if err != nil {
				t.Fatalf("json.Encode() error = %v", err)
			}
			value := json.Value(encoded.Bytes())
			primarySources, ok := value.Get([]string{"primarySources"})
			if !ok {
				t.Fatal("json.Encode() omitted PrimarySources")
			}
			if string(primarySources) != test.wantPrimarySources {
				t.Fatalf("json.Encode() PrimarySources = %q, want %q", primarySources, test.wantPrimarySources)
			}
			identifiers, ok := value.Get([]string{"identifiers"})
			if !ok {
				t.Fatal("json.Encode() omitted Identifiers")
			}
			if string(identifiers) != test.wantIdentifiers {
				t.Fatalf("json.Encode() Identifiers = %q, want %q", identifiers, test.wantIdentifiers)
			}
		})
	}
}

// TestWorkspaceMarshalJSON verifies that json.Marshal encodes nil Workspace
// collection fields using their empty JSON representations.
func TestWorkspaceMarshalJSON(t *testing.T) {

	value, err := json.Marshal(Workspace{})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	tests := []struct {
		name string
		want string
	}{
		{name: "primarySources", want: "{}"},
		{name: "identifiers", want: "[]"},
	}

	for _, test := range tests {
		got, ok := value.Get([]string{test.name})
		if !ok {
			t.Fatalf("json.Marshal() omitted %s", test.name)
		}
		if string(got) != test.want {
			t.Fatalf("json.Marshal() %s = %q, want %q", test.name, got, test.want)
		}
	}

}

func Test_validateUIPreferences(t *testing.T) {
	tests := []struct {
		name  string
		prefs UIPreferences
		err   string
	}{
		{
			name: "Nothing is set",
			prefs: UIPreferences{
				Profile: struct {
					Image     string "json:\"image\""
					FirstName string "json:\"firstName\""
					LastName  string "json:\"lastName\""
					Extra     string "json:\"extra\""
				}{},
			},
		},
		{
			name: "Valid property paths",
			prefs: UIPreferences{
				Profile: struct {
					Image     string "json:\"image\""
					FirstName string "json:\"firstName\""
					LastName  string "json:\"lastName\""
					Extra     string "json:\"extra\""
				}{
					Image:     "additional_data.image",
					FirstName: "first_name",
					LastName:  "last_name",
					Extra:     "email",
				},
			},
		},
		{
			name: "Last name has an invalid property path",
			prefs: UIPreferences{
				Profile: struct {
					Image     string "json:\"image\""
					FirstName string "json:\"firstName\""
					LastName  string "json:\"lastName\""
					Extra     string "json:\"extra\""
				}{
					Image:     "additional_data.image",
					FirstName: "first_name",
					LastName:  "last name", // space instead of _
					Extra:     "email",
				},
			},
			err: "invalid profile 'lastName' \"last name\"",
		},
		{
			name: "Extra is too long",
			prefs: UIPreferences{
				Profile: struct {
					Image     string "json:\"image\""
					FirstName string "json:\"firstName\""
					LastName  string "json:\"lastName\""
					Extra     string "json:\"extra\""
				}{
					Image:     "additional_data.image",
					FirstName: "first_name",
					LastName:  "last_name",
					Extra:     strings.Repeat("x", 1025),
				},
			},
			err: "invalid profile 'extra' \"" + strings.Repeat("x", 1025) + "\"",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := validateUIPreferences(test.prefs)
			var gotStr string
			if got != nil {
				gotStr = got.Error()
			}
			if gotStr != test.err {
				t.Fatalf("expected error %q, got %q", test.err, gotStr)
			}
		})
	}
}
