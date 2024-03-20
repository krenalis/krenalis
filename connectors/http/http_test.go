//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package http

import (
	"testing"

	"chichi/connector"
)

func TestCompletePath(t *testing.T) {
	c := &connection{settings: &settings{Host: "example.com", Port: 443}}
	c2 := &connection{settings: &settings{Host: "example.com", Port: 8080}}
	tests := []connector.CompletePathTest{
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
		{Name: "/a", Expected: "https://example.com:8080/a", Connection: c2},
	}
	err := connector.TestCompletePath(c, tests)
	if err != nil {
		t.Errorf("HTTP connector: %s", err)
	}
}
