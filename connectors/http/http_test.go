// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package http

import (
	"context"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/json"
)

func TestAbsolutePath(t *testing.T) {
	http := &HTTP{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{Host: "example.com", Port: 443})}}
	http2 := &HTTP{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{Host: "example.com", Port: 8080})}}
	tests := []testconnector.AbsolutePathTest{
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
	err := testconnector.TestAbsolutePath(http, tests)
	if err != nil {
		t.Errorf("HTTP Files connector: %s", err)
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
