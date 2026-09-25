//go:build !windows

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package filesystem

import (
	"errors"
	"io"
	fsPkg "io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
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
			{Name: ""},
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

func TestSymlinks(t *testing.T) {

	// Mutex access to 'root' is not necessary as it is essential that these
	// tests are run non-concurrently.
	dir := t.TempDir()
	root = filepath.Join(dir, "root")
	outside := filepath.Join(dir, "outside")
	for _, d := range []string{root, outside} {
		err := os.Mkdir(d, 0755)
		if err != nil {
			t.Fatal(err)
		}
	}
	err := os.WriteFile(filepath.Join(outside, "secret"), []byte("secret"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(root, "a.csv"), []byte("data"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	links := map[string]string{
		"file":             filepath.Join(outside, "secret"),
		"relative-outside": filepath.Join("..", "outside", "secret"),
		"dir":              outside,
		"absolute-inside":  filepath.Join(root, "a.csv"),
		"relative-inside":  "a.csv",
	}
	for name, target := range links {
		err := os.Symlink(target, filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
	}

	fs := &FileSystem{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{})}}

	t.Run("Reader", func(t *testing.T) {
		for _, name := range []string{"file", "relative-outside", "dir/secret", "absolute-inside"} {
			r, _, err := fs.Reader(t.Context(), name)
			if err != nil {
				pErr, ok := errors.AsType[*fsPkg.PathError](err)
				if !ok {
					t.Errorf("%q: expected *fs.PathError, got %T (%s)", name, err, err)
					continue
				}
				if pErr.Path != filepath.Join(root, name) {
					t.Errorf("%q: expected path %q, got %q", name, filepath.Join(root, name), pErr.Path)
				}
				continue
			}
			_ = r.Close()
			t.Errorf("%q: expected *fs.PathError, got nil", name)
		}
	})

	t.Run("Write", func(t *testing.T) {
		err := fs.Write(t.Context(), strings.NewReader("data"), "dir/file", "text/plain")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		for _, name := range []string{"file", "file.tmp"} {
			_, err := os.Lstat(filepath.Join(outside, name))
			if err != nil {
				if !errors.Is(err, fsPkg.ErrNotExist) {
					t.Fatalf("expected %s to not exist outside the root, got error %s", name, err)
				}
				continue
			}
			t.Fatalf("expected %s to not exist outside the root, got it", name)
		}
	})

	t.Run("Inside root", func(t *testing.T) {
		err := fs.Write(t.Context(), strings.NewReader("data"), "/b.csv", "text/csv")
		if err != nil {
			t.Fatalf("expected no error, got %s", err)
		}
		for _, name := range []string{"b.csv", "relative-inside"} {
			r, _, err := fs.Reader(t.Context(), name)
			if err != nil {
				t.Fatalf("%q: expected no error, got %s", name, err)
			}
			data, err := io.ReadAll(r)
			_ = r.Close()
			if err != nil {
				t.Fatalf("%q: expected no error, got %s", name, err)
			}
			if string(data) != "data" {
				t.Fatalf("%q: expected %q, got %q", name, "data", data)
			}
		}
	})

}
