//go:build !windows

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package filesystem

import (
	"context"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/json"
)

func TestPathConvert(t *testing.T) {

	t.Run("Root is '/'", func(t *testing.T) {
		// Mutex access to 'root' is not necessary as it is essential that these
		// tests are run non-concurrently.
		root = "/"
		fs := &FileSystem{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{})}}
		tests := []testconnector.AbsolutePathTest{
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
		}
		err := testconnector.TestAbsolutePath(fs, tests)
		if err != nil {
			t.Errorf("File System connector: %s", err)
		}
	})

	t.Run("Root is '/root'", func(t *testing.T) {
		// Mutex access to 'root' is not necessary as it is essential that these
		// tests are run non-concurrently.
		root = "/root"
		fs := &FileSystem{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{})}}
		tests := []testconnector.AbsolutePathTest{
			{Name: "a", Expected: "/root/a"},
			{Name: "/a", Expected: "/root/a"},
		}
		err := testconnector.TestAbsolutePath(fs, tests)
		if err != nil {
			t.Errorf("File System connector: %s", err)
		}
	})

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
