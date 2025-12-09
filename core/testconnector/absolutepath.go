// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package testconnector

import (
	"context"
	"fmt"

	"github.com/meergo/meergo/connectors"
)

// AbsolutePathTest is a test for FileStorage.AbsolutePath.
type AbsolutePathTest struct {
	Name     string // path name.
	Expected string // expected absolute path.
	Storage  any    // storage to use, if not nil.
}

// TestAbsolutePath tests FileStorage.AbsolutePath of the connector executing the
// given tests. It returns an error if a test fails.
func TestAbsolutePath(storage any, tests []AbsolutePathTest) error {
	ctx := context.Background()
	for _, test := range tests {
		s := storage
		if test.Storage != nil {
			s = test.Storage
		}
		got, err := s.(interface {
			AbsolutePath(ctx context.Context, name string) (string, error)
		}).AbsolutePath(ctx, test.Name)
		if err != nil {
			_, ok := err.(*connectors.InvalidPathError)
			if !ok {
				if test.Expected == "" {
					return fmt.Errorf("%q: expected an *InvalidPathError, got error %#v", test.Name, err)
				}
				return fmt.Errorf("%q: expected no errors, got %s", test.Name, err)
			}
			if test.Expected != "" {
				return fmt.Errorf("%q: expected no errors, got error %qs", test.Name, err)
			}
			continue
		}
		if test.Expected == "" {
			return fmt.Errorf("%q: expected error, got no errors", test.Name)
		}
		if got != test.Expected {
			return fmt.Errorf("%q: expected absolute path %q, got %q", test.Name, test.Expected, got)
		}
	}
	return nil
}
