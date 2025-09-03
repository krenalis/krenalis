//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package core

import (
	"strings"
	"testing"

	"github.com/meergo/meergo/types"
)

func Test_suitableAsIdentifier(t *testing.T) {
	tests := []struct {
		t        types.Type
		expected bool
	}{
		{types.Text(), true},
		{types.Boolean(), false},
		{types.Int(16), true},
		{types.Int(32), true},
		{types.Int(64), true},
		{types.Uint(8), true},
		{types.Uint(32), true},
		{types.Float(32), false},
		{types.Float(64), false},
		{types.Decimal(10, 0), true},
		{types.Decimal(10, 3), false},
		{types.Decimal(3, 3), false},
		{types.DateTime(), false},
		{types.Date(), false},
		{types.Time(), false},
		{types.Year(), false},
		{types.UUID(), true},
		{types.Inet(), true},
		{types.Array(types.Text()), false},
		{types.Array(types.Float(32)), false},
		{types.Array(types.Decimal(10, 0)), false},
		{types.Array(types.Array(types.Text())), false},
		{types.Object([]types.Property{{Name: "a", Type: types.Text()}}), false},
		{types.Map(types.Text()), false},
	}
	for _, test := range tests {
		got := suitableAsIdentifier(test.t)
		if got != test.expected {
			t.Errorf("type %v: expected %t, got %t", test.t, test.expected, got)
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
				UserProfile: struct {
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
				UserProfile: struct {
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
				UserProfile: struct {
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
			err: "invalid user profile 'lastName' \"last name\"",
		},
		{
			name: "Extra is too long",
			prefs: UIPreferences{
				UserProfile: struct {
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
			err: "invalid user profile 'extra' \"" + strings.Repeat("x", 1025) + "\"",
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
