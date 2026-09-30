// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"context"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/connectors"
)

// echoAbsolutePathStorage is a file storage connector whose AbsolutePath
// method returns the name it receives.
type echoAbsolutePathStorage struct{}

func (echoAbsolutePathStorage) AbsolutePath(_ context.Context, name string) (string, error) {
	return name, nil
}

// TestFileStorageAbsolutePath verifies that AbsolutePath rejects a name longer
// than 1024 runes after placeholder replacement.
func TestFileStorageAbsolutePath(t *testing.T) {

	storage := &FileStorage{connector: "test", inner: echoAbsolutePathStorage{}}
	replacer := func(name string) (string, bool) {
		return "0123456789", name == "ten"
	}

	tests := []struct {
		name     string
		expected string // expected absolute path; empty if an *InvalidPathError is expected.
	}{
		{name: strings.Repeat("a", 1014) + "${ten}", expected: strings.Repeat("a", 1014) + "0123456789"},
		{name: strings.Repeat("à", 1014) + "${ten}", expected: strings.Repeat("à", 1014) + "0123456789"},
		{name: strings.Repeat("a", 1015) + "${ten}"},
		{name: strings.Repeat("à", 1015) + "${ten}"},
	}
	for _, test := range tests {
		got, err := storage.AbsolutePath(t.Context(), test.name, replacer)
		if err != nil {
			if test.expected != "" {
				t.Fatalf("expected no error, got %q", err)
			}
			if _, ok := err.(*connectors.InvalidPathError); !ok {
				t.Fatalf("expected *connectors.InvalidPathError, got %T", err)
			}
			continue
		}
		if test.expected == "" {
			t.Fatalf("expected *connectors.InvalidPathError, got no error")
		}
		if got != test.expected {
			t.Fatalf("expected %q, got %q", test.expected, got)
		}
	}

}
