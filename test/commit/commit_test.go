// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestFindPackages tests that findPackages returns, sorted, the directories of
// the root module that contain Go files, skipping vendor directories and other
// modules.
func TestFindPackages(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"go.mod",
		"main.go",
		"core/core.go",
		"core/internal/db/db.go",
		"core/internal/db/db_test.go",
		"docs/README.md",
		"vendor/github.com/example/lib/lib.go",
		"local/go.mod",
		"local/local.go",
		"local/sub/sub.go",
	}
	for _, name := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(path, nil, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}
	expected := []string{
		dir,
		filepath.Join(dir, "core"),
		filepath.Join(dir, "core", "internal", "db"),
	}
	got := findPackages(dir)
	if !slices.Equal(got, expected) {
		t.Fatalf("expected packages %q, got %q", expected, got)
	}
}
