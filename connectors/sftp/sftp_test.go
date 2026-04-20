// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package sftp

import (
	"context"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/json"
)

func TestPathConvert(t *testing.T) {
	sf := &SFTP{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{Host: "example.com", Port: 22})}}
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
