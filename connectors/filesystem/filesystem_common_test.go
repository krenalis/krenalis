// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package filesystem

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
)

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

	names := []string{"/", ".", "./b", "a/", "a/./b", "a/../b", "a//b"}

	t.Run("Reader", func(t *testing.T) {
		for _, name := range names {
			_, _, err := fs.Reader(t.Context(), name)
			if err != nil {
				if _, ok := errors.AsType[*connectors.InvalidPathError](err); !ok {
					t.Errorf("%q: expected *connectors.InvalidPathError, got %T (%s)", name, err, err)
				}
				continue
			}
			t.Errorf("%q: expected *connectors.InvalidPathError, got nil", name)
		}
	})

	t.Run("Write", func(t *testing.T) {
		for _, name := range names {
			err := fs.Write(t.Context(), strings.NewReader("data"), name, "text/plain")
			if err != nil {
				if _, ok := errors.AsType[*connectors.InvalidPathError](err); !ok {
					t.Errorf("%q: expected *connectors.InvalidPathError, got %T (%s)", name, err, err)
				}
				continue
			}
			t.Errorf("%q: expected *connectors.InvalidPathError, got nil", name)
		}
	})

	for dir, expected := range map[string][]string{root: {"a", "b"}, filepath.Join(root, "a"): nil} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		if !slices.Equal(names, expected) {
			t.Fatalf("expected entries %q in %s, got %q", expected, dir, names)
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

func TestRewritePathError(t *testing.T) {

	// Mutex access to 'root' and 'displayedRoot' is not necessary as it is
	// essential that these tests are run non-concurrently.
	root = t.TempDir()
	shown := t.TempDir()
	t.Cleanup(func() { displayedRoot = "" })

	for _, test := range []struct{ name, displayed string }{{"Without displayed root", ""}, {"With displayed root", shown}} {
		displayedRoot = test.displayed
		rootToShow := root
		if test.displayed != "" {
			rootToShow = test.displayed
		}
		t.Run(test.name, func(t *testing.T) {
			t.Run("PathError", func(t *testing.T) {
				for _, path := range []string{"a/b", filepath.Join(root, "a/b")} {
					err := rewritePathError(&fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist})
					pErr, ok := err.(*fs.PathError)
					if !ok {
						t.Fatalf("expected *fs.PathError, got %T", err)
					}
					if expected := filepath.Join(rootToShow, "a/b"); pErr.Path != expected {
						t.Fatalf("expected path %q, got %q", expected, pErr.Path)
					}
				}
			})
			t.Run("LinkError", func(t *testing.T) {
				err := rewritePathError(&os.LinkError{Op: "renameat", Old: "a.tmp", New: "a", Err: fs.ErrExist})
				lErr, ok := err.(*os.LinkError)
				if !ok {
					t.Fatalf("expected *os.LinkError, got %T", err)
				}
				if expected := filepath.Join(rootToShow, "a.tmp"); lErr.Old != expected {
					t.Fatalf("expected old path %q, got %q", expected, lErr.Old)
				}
				if expected := filepath.Join(rootToShow, "a"); lErr.New != expected {
					t.Fatalf("expected new path %q, got %q", expected, lErr.New)
				}
			})
		})
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
