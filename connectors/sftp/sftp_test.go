//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package sftp

import (
	"testing"

	"github.com/meergo/meergo/core/testconnector"
)

func TestPathConvert(t *testing.T) {
	sf := &SFTP{settings: &innerSettings{Host: "example.com", Port: 22}}
	tests := []testconnector.AbsolutePathTest{
		{Name: "/a", Expected: "sftp://example.com:22/a"},
		{Name: "/a/b", Expected: "sftp://example.com:22/a/b"},
		{Name: "a", Expected: "sftp://example.com:22/a"},
		{Name: "/\x00", Expected: "sftp://example.com:22/%00"},
	}
	err := testconnector.TestAbsolutePath(sf, tests)
	if err != nil {
		t.Errorf("SFTP connector: %s", err)
	}
}
