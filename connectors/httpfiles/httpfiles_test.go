//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package httpfiles

import (
	"testing"

	"github.com/meergo/meergo"
)

func TestAbsolutePath(t *testing.T) {
	httpFiles := &HTTPFiles{settings: &innerSettings{Host: "example.com", Port: 443}}
	httpFiles2 := &HTTPFiles{settings: &innerSettings{Host: "example.com", Port: 8080}}
	tests := []meergo.AbsolutePathTest{
		{Name: "/a", Expected: "https://example.com/a"},
		{Name: "a", Expected: "https://example.com/a"},
		{Name: "/a/b", Expected: "https://example.com/a/b"},
		{Name: "/a/b?", Expected: "https://example.com/a/b"},
		{Name: "/a/b?x=y", Expected: "https://example.com/a/b?x=y"},
		{Name: "a/b?x=y", Expected: "https://example.com/a/b?x=y"},
		{Name: "/%5z"},
		{Name: "%5z"},
		{Name: "/\x00"},
		{Name: "/a/b?x=y#"},
		{Name: "/a", Expected: "https://example.com:8080/a", Storage: httpFiles2},
	}
	err := meergo.TestAbsolutePath(httpFiles, tests)
	if err != nil {
		t.Errorf("HTTP Files connector: %s", err)
	}
}
