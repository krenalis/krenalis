//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package s3

import (
	"strings"
	"testing"

	"github.com/meergo/meergo"
)

func TestPathConvert(t *testing.T) {
	s3 := &S3{settings: &innerSettings{Bucket: "my-example-bucket"}}
	tests := []meergo.AbsolutePathTest{
		{Name: "a", Expected: "s3://my-example-bucket/a"},
		{Name: "a/b", Expected: "s3://my-example-bucket/a/b"},
		{Name: "/a", Expected: "s3://my-example-bucket/a"},
		{Name: "\x00", Expected: "s3://my-example-bucket/\x00"},
		{Name: strings.Repeat("x", 1024), Expected: "s3://my-example-bucket/" + strings.Repeat("x", 1024)},
		{Name: "/" + strings.Repeat("x", 1023), Expected: "s3://my-example-bucket/" + strings.Repeat("x", 1023)},
		{Name: strings.Repeat("x", 1025)},
	}
	err := meergo.TestAbsolutePath(s3, tests)
	if err != nil {
		t.Errorf("S3 connector: %s", err)
	}
}
