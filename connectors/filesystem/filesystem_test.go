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

	"chichi"
)

func TestPathConvert(t *testing.T) {
	fs := &Filesystem{settings: &settings{Root: "/"}}
	fs2 := &Filesystem{settings: &settings{Root: "/root"}}
	tests := []chichi.CompletePathTest{
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
		{Name: "a", Expected: "/root/a", Storage: fs2},
		{Name: "/a", Expected: "/root/a", Storage: fs2},
	}
	err := chichi.TestCompletePath(fs, tests)
	if err != nil {
		t.Errorf("Filesystem connector: %s", err)
	}
}
