// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package filesystem

import (
	"testing"

	"github.com/krenalis/krenalis/core/testconnector"
)

func TestPathConvert(t *testing.T) {

	t.Run("Root is 'C:\\'", func(t *testing.T) {
		// Mutex access to 'root' is not necessary as it is essential that these
		// tests are run non-concurrently.
		root = "C:\\"
		fs := &FileSystem{settings: &innerSettings{}}
		tests := []testconnector.AbsolutePathTest{
			{Name: "a", Expected: "C:\\a"},
			{Name: "a.e", Expected: "C:\\a.e"},
			{Name: "a/b.e", Expected: "C:\\a\\b.e"},
			{Name: "/a", Expected: "C:\\a"},
			{Name: "/a/b", Expected: "C:\\a\\b"},
			{Name: "/\x00", Expected: "C:\\\x00"},
			{Name: "/"},
			{Name: "a/./b"},
			{Name: "a/.."},
			{Name: "../a"},
			{Name: "a/"},
		}
		err := testconnector.TestAbsolutePath(fs, tests)
		if err != nil {
			t.Errorf("File System connector: %s", err)
		}
	})

	t.Run("Root is 'C:\\root'", func(t *testing.T) {
		// Mutex access to 'root' is not necessary as it is essential that these
		// tests are run non-concurrently.
		root = "C:\\root"
		fs := &FileSystem{settings: &innerSettings{}}
		tests := []testconnector.AbsolutePathTest{
			{Name: "a", Expected: "C:\\root\\a"},
			{Name: "/a", Expected: "C:\\root\\a"},
		}
		err := testconnector.TestAbsolutePath(fs, tests)
		if err != nil {
			t.Errorf("File System connector: %s", err)
		}
	})

}
