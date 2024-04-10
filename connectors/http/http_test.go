//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package http

import (
	"testing"

	"github.com/open2b/chichi"
)

func TestCompletePath(t *testing.T) {
	http := &HTTP{settings: &Settings{Host: "example.com", Port: 443}}
	http2 := &HTTP{settings: &Settings{Host: "example.com", Port: 8080}}
	tests := []chichi.CompletePathTest{
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
		{Name: "/a", Expected: "https://example.com:8080/a", Storage: http2},
	}
	err := chichi.TestCompletePath(http, tests)
	if err != nil {
		t.Errorf("HTTP connector: %s", err)
	}
}
