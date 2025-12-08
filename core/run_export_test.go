// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import "testing"

func TestGetAttribute(t *testing.T) {
	cases := []struct {
		name       string
		attributes map[string]any
		path       string
		expected   any
		ok         bool
	}{
		{
			name: "flat path",
			attributes: map[string]any{
				"email": "user@example.com",
			},
			path:     "email",
			expected: "user@example.com",
			ok:       true,
		},
		{
			name: "nested path",
			attributes: map[string]any{
				"profile": map[string]any{
					"address": map[string]any{
						"city": "Rome",
					},
				},
			},
			path:     "profile.address.city",
			expected: "Rome",
			ok:       true,
		},
		{
			name: "nested path",
			attributes: map[string]any{
				"address": map[string]any{
					"country": "IT",
				},
			},
			path: "address.city",
			ok:   false,
		},
	}
	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			got, ok := getAttribute(test.attributes, test.path)
			if got != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
			if ok != test.ok {
				t.Fatalf("expected ok %t, got ok %t", test.ok, ok)
			}
		})
	}
}

func TestSetAttribute(t *testing.T) {

	t.Run("email", func(t *testing.T) {
		attributes := map[string]any{}
		setAttribute(attributes, "email", "user@example.com")
		got, ok := attributes["email"]
		if !ok {
			t.Fatal("expected top-level property to be set")
		}
		if got != "user@example.com" {
			t.Fatalf("expected %v, got %v", "user@example.com", got)
		}
	})

	t.Run("profile.address.city", func(t *testing.T) {
		attributes := map[string]any{
			"profile": map[string]any{
				"address": map[string]any{
					"street": "Via Veneto 143",
				},
			},
		}
		setAttribute(attributes, "profile.address.city", "Rome")
		profile, ok := attributes["profile"].(map[string]any)
		if !ok {
			t.Fatal("expected 'profile' map to exist")
		}
		address, ok := profile["address"].(map[string]any)
		if !ok {
			t.Fatal("expected 'address' map to exist")
		}
		got, ok := address["city"]
		if !ok {
			t.Fatal("expected 'city' property to be set")
		}
		if got != "Rome" {
			t.Fatalf("expected %v, got %v", "Rome", got)
		}
	})

	t.Run("profile.name", func(t *testing.T) {
		attributes := map[string]any{
			"profile": map[string]any{
				"address": map[string]any{
					"street": "Via Veneto 143",
				},
			},
		}
		setAttribute(attributes, "profile.name", "Marcello")
		profile, ok := attributes["profile"].(map[string]any)
		if !ok {
			t.Fatal("expected 'profile' map to exist")
		}
		got, ok := profile["name"]
		if !ok {
			t.Fatal("expected 'name' property to be set")
		}
		if got != "Marcello" {
			t.Fatalf("expected %v, got %v", "Marcello", got)
		}
	})

	t.Run("games.tetris.score", func(t *testing.T) {
		attributes := map[string]any{}
		setAttribute(attributes, "games.tetris.score", 704)
		games, ok := attributes["games"].(map[string]any)
		if !ok {
			t.Fatal("expected 'games' map to exist")
		}
		tetris, ok := games["tetris"].(map[string]any)
		if !ok {
			t.Fatal("expected 'name' map to exist")
		}
		got, ok := tetris["score"]
		if !ok {
			t.Fatal("expected 'score' property to be set")
		}
		if got != 704 {
			t.Fatalf("expected %v, got %v", "Marcello", got)
		}
	})

}
