//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package core

import "testing"

func TestGetPropertyValue(t *testing.T) {
	cases := []struct {
		name       string
		properties map[string]any
		path       string
		expected   any
		ok         bool
	}{
		{
			name: "flat path",
			properties: map[string]any{
				"email": "user@example.com",
			},
			path:     "email",
			expected: "user@example.com",
			ok:       true,
		},
		{
			name: "nested path",
			properties: map[string]any{
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
			properties: map[string]any{
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
			got, ok := getPropertyValue(test.properties, test.path)
			if got != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
			if ok != test.ok {
				t.Fatalf("expected ok %t, got ok %t", test.ok, ok)
			}
		})
	}
}

func TestSetPropertyValue(t *testing.T) {

	t.Run("email", func(t *testing.T) {
		properties := map[string]any{}
		setPropertyValue(properties, "email", "user@example.com")
		got, ok := properties["email"]
		if !ok {
			t.Fatal("expected top-level property to be set")
		}
		if got != "user@example.com" {
			t.Fatalf("expected %v, got %v", "user@example.com", got)
		}
	})

	t.Run("profile.address.city", func(t *testing.T) {
		properties := map[string]any{
			"profile": map[string]any{
				"address": map[string]any{
					"street": "Via Veneto 143",
				},
			},
		}
		setPropertyValue(properties, "profile.address.city", "Rome")
		profile, ok := properties["profile"].(map[string]any)
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
		properties := map[string]any{
			"profile": map[string]any{
				"address": map[string]any{
					"street": "Via Veneto 143",
				},
			},
		}
		setPropertyValue(properties, "profile.name", "Marcello")
		profile, ok := properties["profile"].(map[string]any)
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
		properties := map[string]any{}
		setPropertyValue(properties, "games.tetris.score", 704)
		games, ok := properties["games"].(map[string]any)
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
