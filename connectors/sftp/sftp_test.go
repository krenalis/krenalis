// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package sftp

import (
	"testing"

	"github.com/krenalis/krenalis/core/testconnector"
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
