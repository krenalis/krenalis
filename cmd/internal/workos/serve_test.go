// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package workos

import (
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/errors"
)

// TestOnboardRejectsInvalidAdminEmail verifies that Onboard rejects an invalid
// admin email address before creating any organization. Its WorkOS has no
// Core, so a rejection that does not happen first fails the test.
func TestOnboardRejectsInvalidAdminEmail(t *testing.T) {

	wo := &WorkOS{}

	tests := []struct {
		name  string
		email string
	}{
		{name: "empty", email: ""},
		{name: "only spaces", email: "   "},
		{name: "without at sign", email: "admin.example.com"},
		{name: "without domain", email: "admin@"},
		{name: "with a domain without dot", email: "admin@example"},
		{name: "longer than 255 runes", email: strings.Repeat("a", 250) + "@example.com"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := wo.Onboard(t.Context(), "Acme", test.email)
			if err != nil {
				if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
					t.Fatalf("expected *errors.BadRequestError, got %T", err)
				}
				return
			}
			t.Fatal("expected error, got nil")
		})
	}

}
