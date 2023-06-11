//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connector

import "fmt"

// CompletePathTest is a test for StorageConnection.CompletePath.
type CompletePathTest struct {
	Name       string            // path name.
	Expected   string            // expected complete path.
	Connection StorageConnection // connection to use, if not nil.
}

// TestCompletePath tests StorageConnection.CompletePath of connection executing
// the given tests. It returns an error if a test fails.
func TestCompletePath(connection StorageConnection, tests []CompletePathTest) error {
	for _, test := range tests {
		c := connection
		if test.Connection != nil {
			c = test.Connection
		}
		got, err := c.CompletePath(test.Name)
		if err != nil {
			_, ok := err.(InvalidPathError)
			if !ok {
				if test.Expected == "" {
					return fmt.Errorf("%q: expecting an InvalidPathError, got error %#v", test.Name, err)
				}
				return fmt.Errorf("%q: expecting no errors, got %s", test.Name, err)
			}
			if test.Expected != "" {
				return fmt.Errorf("%q: expecting no errors, got error %qs", test.Name, err)
			}
			continue
		}
		if test.Expected == "" {
			return fmt.Errorf("%q: expecting error, got no errors", test.Name)
		}
		if got != test.Expected {
			return fmt.Errorf("%q: expecting complete path %q, got %q", test.Name, test.Expected, got)
		}
	}
	return nil
}
