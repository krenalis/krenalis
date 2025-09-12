//go:build !windows

//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package filesystem

import (
	"testing"

	"github.com/meergo/meergo/core/testconnector"
)

func TestPathConvert(t *testing.T) {

	t.Run("Root is '/'", func(t *testing.T) {
		// Mutex access to 'root' is not necessary as it is essential that these
		// tests are run non-concurrently.
		root = "/"
		fs := &Filesystem{settings: &innerSettings{}}
		tests := []testconnector.AbsolutePathTest{
			{Name: "a", Expected: "/a"},
			{Name: "a.e", Expected: "/a.e"},
			{Name: "a/b.e", Expected: "/a/b.e"},
			{Name: "/a", Expected: "/a"},
			{Name: "/a/b", Expected: "/a/b"},
			{Name: "/\x00", Expected: "/\x00"},
			{Name: "/"},
			{Name: "a/./b"},
			{Name: "a/.."},
			{Name: "../a"},
			{Name: "a/"},
		}
		err := testconnector.TestAbsolutePath(fs, tests)
		if err != nil {
			t.Errorf("Filesystem connector: %s", err)
		}
	})

	t.Run("Root is '/root'", func(t *testing.T) {
		// Mutex access to 'root' is not necessary as it is essential that these
		// tests are run non-concurrently.
		root = "/root"
		fs := &Filesystem{settings: &innerSettings{}}
		tests := []testconnector.AbsolutePathTest{
			{Name: "a", Expected: "/root/a"},
			{Name: "/a", Expected: "/root/a"},
		}
		err := testconnector.TestAbsolutePath(fs, tests)
		if err != nil {
			t.Errorf("Filesystem connector: %s", err)
		}
	})

}
