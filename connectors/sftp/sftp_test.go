//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package sftp

import (
	"testing"

	"chichi"
)

func TestPathConvert(t *testing.T) {
	c := &connection{settings: &settings{Host: "example.com", Port: 22}}
	tests := []chichi.CompletePathTest{
		{Name: "/a", Expected: "sftp://example.com:22/a"},
		{Name: "/a/b", Expected: "sftp://example.com:22/a/b"},
		{Name: "a", Expected: "sftp://example.com:22/a"},
		{Name: "/\x00", Expected: "sftp://example.com:22/%00"},
	}
	err := chichi.TestCompletePath(c, tests)
	if err != nil {
		t.Errorf("SFTP connector: %s", err)
	}
}
