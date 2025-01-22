//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package meergo

import (
	"context"
	"fmt"
)

// CompletePathTest is a test for FileStorage.CompletePath.
type CompletePathTest struct {
	Name     string // path name.
	Expected string // expected complete path.
	Storage  any    // storage to use, if not nil.
}

// TestCompletePath tests FileStorage.CompletePath of the connector executing the
// given tests. It returns an error if a test fails.
func TestCompletePath(storage any, tests []CompletePathTest) error {
	ctx := context.Background()
	for _, test := range tests {
		s := storage
		if test.Storage != nil {
			s = test.Storage
		}
		got, err := s.(interface {
			CompletePath(ctx context.Context, name string) (string, error)
		}).CompletePath(ctx, test.Name)
		if err != nil {
			_, ok := err.(*InvalidPathError)
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
			return fmt.Errorf("%q: expected complete path %q, got %q", test.Name, test.Expected, got)
		}
	}
	return nil
}
