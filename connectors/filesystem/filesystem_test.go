//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package filesystem

import (
	"testing"

	"chichi/connector"
)

func TestPathConvert(t *testing.T) {
	c := &connection{settings: &settings{Root: "/"}}
	tests := []connector.CompletePathTest{
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
	err := connector.TestCompletePath(c, tests)
	if err != nil {
		t.Errorf("Filesystem connector: %s", err)
	}
	c = &connection{settings: &settings{Root: "/root"}}
	tests = []connector.CompletePathTest{
		{Name: "a", Expected: "/root/a"},
		{Name: "/a", Expected: "/root/a"},
	}
	err = connector.TestCompletePath(c, tests)
	if err != nil {
		t.Errorf("Filesystem connector: %s", err)
	}
}
