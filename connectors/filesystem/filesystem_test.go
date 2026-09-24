//go:build !windows

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package filesystem

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestInvalidName(t *testing.T) {

	// Mutex access to 'root' is not necessary as it is essential that these
	// tests are run non-concurrently.
	root = t.TempDir()
	err := os.Mkdir(filepath.Join(root, "a"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(root, "b"), []byte("b"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	fs := &FileSystem{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{})}}

	for _, name := range []string{"/", ".", "./b", "a/", "a/./b", "a/../b", "a//b"} {
		t.Run(name, func(t *testing.T) {
			t.Run("Reader", func(t *testing.T) {
				_, _, err := fs.Reader(t.Context(), name)
				if err != nil {
					if _, ok := errors.AsType[*connectors.InvalidPathError](err); !ok {
						t.Fatalf("expected *connectors.InvalidPathError, got %T (%s)", err, err)
					}
					return
				}
				t.Fatal("expected *connectors.InvalidPathError, got nil")
			})
			t.Run("Write", func(t *testing.T) {
				err := fs.Write(t.Context(), strings.NewReader("data"), name, "text/plain")
				if err != nil {
					if _, ok := errors.AsType[*connectors.InvalidPathError](err); !ok {
						t.Fatalf("expected *connectors.InvalidPathError, got %T (%s)", err, err)
					}
					return
				}
				t.Fatal("expected *connectors.InvalidPathError, got nil")
			})
		})
	}

	for _, dir := range []string{root, filepath.Join(root, "a")} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if e.Name() != "a" && e.Name() != "b" {
				t.Fatalf("expected no files created in %s, got %s", dir, e.Name())
			}
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "b"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "b" {
		t.Fatalf("expected %q, got %q", "b", data)
	}

}

func TestSymlinkOutsideRoot(t *testing.T) {

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
	err = os.Symlink(filepath.Join(outside, "secret"), filepath.Join(root, "file"))
	if err != nil {
		t.Fatal(err)
	}
	err = os.Symlink(outside, filepath.Join(root, "dir"))
	if err != nil {
		t.Fatal(err)
	}

	fs := &FileSystem{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{})}}

	t.Run("Reader", func(t *testing.T) {
		_, _, err := fs.Reader(t.Context(), "file")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
	})

	t.Run("Write", func(t *testing.T) {
		err := fs.Write(t.Context(), strings.NewReader("data"), "dir/file", "text/plain")
		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		for _, name := range []string{"file", "file.tmp"} {
			_, err := os.Lstat(filepath.Join(outside, name))
			if !os.IsNotExist(err) {
				t.Fatalf("expected %s to not exist outside the root, got error %v", name, err)
			}
		}
	})

	t.Run("Inside root", func(t *testing.T) {
		err := fs.Write(t.Context(), strings.NewReader("data"), "/a.csv", "text/csv")
		if err != nil {
			t.Fatalf("expected no error, got %s", err)
		}
		r, _, err := fs.Reader(t.Context(), "a.csv")
		if err != nil {
			t.Fatalf("expected no error, got %s", err)
		}
		defer r.Close()
		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("expected no error, got %s", err)
		}
		if string(data) != "data" {
			t.Fatalf("expected %q, got %q", "data", data)
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
