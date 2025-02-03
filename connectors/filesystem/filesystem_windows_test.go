//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package filesystem

import (
	"testing"

	"github.com/meergo/meergo"
)

func TestPathConvert(t *testing.T) {
	fs := &Filesystem{settings: &innerSettings{Root: "C:\\"}}
	fs2 := &Filesystem{settings: &innerSettings{Root: "C:\\root"}}
	tests := []meergo.AbsolutePathTest{
		{Name: "a", Expected: "C:\\a"},
		{Name: "a.e", Expected: "C:\\a.e"},
		{Name: "a/b.e", Expected: "C:\\a\\b.e"},
		{Name: "a/b.e", Expected: "C:\\a\\b.e"},
		{Name: "/a", Expected: "C:\\a"},
		{Name: "/a/b", Expected: "C:\\a\\b"},
		{Name: "/\x00", Expected: "C:\\\x00"},
		{Name: "/"},
		{Name: "a/./b"},
		{Name: "a/.."},
		{Name: "../a"},
		{Name: "a/"},
		{Name: "a", Expected: "C:\\root\\a", Storage: fs2},
		{Name: "/a", Expected: "C:\\root\\a", Storage: fs2},
		{Name: "a/b", Expected: "C:\\a\\b"},
		{Name: "/a/b", Expected: "C:\\a\\b"},
	}
	err := meergo.TestAbsolutePath(fs, tests)
	if err != nil {
		t.Errorf("Filesystem connector: %s", err)
	}
}
