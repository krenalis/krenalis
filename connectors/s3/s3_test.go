// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package s3

import (
	"context"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/json"
)

func TestPathConvert(t *testing.T) {
	s3 := &S3{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{Bucket: "my-example-bucket"})}}
	tests := []testconnector.AbsolutePathTest{
		{Name: "a", Expected: "s3://my-example-bucket/a"},
		{Name: "a/b", Expected: "s3://my-example-bucket/a/b"},
		{Name: "/a", Expected: "s3://my-example-bucket/a"},
		{Name: "\x00", Expected: "s3://my-example-bucket/\x00"},
		{Name: strings.Repeat("x", 1024), Expected: "s3://my-example-bucket/" + strings.Repeat("x", 1024)},
		{Name: "/" + strings.Repeat("x", 1023), Expected: "s3://my-example-bucket/" + strings.Repeat("x", 1023)},
		{Name: strings.Repeat("x", 1025)},
	}
	err := testconnector.TestAbsolutePath(s3, tests)
	if err != nil {
		t.Errorf("S3 connector: %s", err)
	}
}

type testSettingsStore struct {
	settings json.Value
}

func newTestSettingsStore(t *testing.T, settings any) *testSettingsStore {
	t.Helper()

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("cannot marshal test settings: %s", err)
	}
	return &testSettingsStore{settings: data}
}

func (s *testSettingsStore) Load(ctx context.Context, dst any) error {
	return json.Unmarshal(s.settings, dst)
}

func (s *testSettingsStore) Store(ctx context.Context, src any) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	s.settings = data
	return nil
}
