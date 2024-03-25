//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package filesystem

import (
	"testing"

	"chichi"
)

func TestPathConvert(t *testing.T) {
	c := &Filesystem{settings: &settings{Root: "C:\\"}}
	c2 := &Filesystem{settings: &settings{Root: "C:\\root"}}
	tests := []chichi.CompletePathTest{
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
		{Name: "a", Expected: "C:\\root\\a", Storage: c2},
		{Name: "/a", Expected: "C:\\root\\a", Storage: c2},
		{Name: "a/b", Expected: "C:\\a\\b"},
		{Name: "/a/b", Expected: "C:\\a\\b"},
	}
	err := chichi.TestCompletePath(c, tests)
	if err != nil {
		t.Errorf("Filesystem connector: %s", err)
	}
}
