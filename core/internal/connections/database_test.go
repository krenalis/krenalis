// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"testing"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

type sqlLiteralDatabase struct {
	databaseConnection
	literal string
	err     error
}

// SQLLiteral returns the configured literal and error.
func (database sqlLiteralDatabase) SQLLiteral(any, types.Type) (string, error) {
	return database.literal, database.err
}

// TestDatabaseUpdatedAtPlaceholder checks literal values and connector error
// handling.
func TestDatabaseUpdatedAtPlaceholder(t *testing.T) {
	tests := []struct {
		name        string
		literal     string
		err         error
		wantLiteral string
	}{
		{name: "literal", literal: "NULL", wantLiteral: "NULL"},
		{name: "connector error", literal: "partial", err: errors.New("literal conversion failed"), wantLiteral: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &Database{inner: sqlLiteralDatabase{literal: test.literal, err: test.err}}
			got, err := database.UpdatedAtPlaceholder(nil)
			if got != test.wantLiteral {
				t.Errorf("expected literal %q, got %q", test.wantLiteral, got)
			}
			if err != nil {
				if test.err == nil {
					t.Fatalf("expected no error, got %v", err)
				}
				unavailable, ok := errors.AsType[*UnavailableError](err)
				if !ok {
					t.Fatalf("expected an UnavailableError, got %T: %v", err, err)
				}
				if unavailable.Err != test.err {
					t.Errorf("expected connector error %v, got %v", test.err, unavailable.Err)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("expected connector error %v, got no error", test.err)
			}
		})
	}
}
