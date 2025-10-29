// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package meergo

import (
	"testing"
)

func Test_SuggestPropertyName(t *testing.T) {

	tests := []struct {
		s        string
		expected string
	}{
		{"", ""},
		{"_", "_"},
		{"a", "a"},
		{"first_name", "first_name"},
		{"_age", "_age"},
		{"Data di Nascita", "Data_di_Nascita"},
		{"Prénom", "Prenom"},
		{"Dirección de Correo", "Direccion_de_Correo"},
		{"Ørganisation", ""},
		{"姓", ""},
		{"amount €", "amount_"},
		{" first  name", "first_name"},
		{"last name  ", "last_name"},
		{"date-of-birth", "date_of_birth"},
		{"$$", ""},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			got, ok := SuggestPropertyName(test.s)
			if !ok {
				if test.expected != "" {
					t.Fatal("expected true, got false")
				}
				return
			}
			if test.expected == "" {
				t.Fatal("expected false, got true")
			}
			if test.expected != got {
				t.Fatalf("expected %q, got %q", test.expected, got)
			}
		})
	}

}
