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
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
)

func TestInvalidName(t *testing.T) {

	fs := &FileSystem{env: &connectors.FileStorageEnv{Settings: newTestSettingsStore(t, innerSettings{})}}

	const dotsMessage = "path name cannot contain “.” or “..” as a directory or file name"
	const slashesMessage = "path name cannot contain consecutive slashes"

	tests := []struct {
		name    string
		message string
	}{
		{"/", "path name cannot refer to the root directory"},
		{".", dotsMessage},
		{"..", dotsMessage},
		{"./b", dotsMessage},
		{"a/", "path name cannot end with a slash"},
		{"a/./b", dotsMessage},
		{"a/../b", dotsMessage},
		{"a//b", slashesMessage},
		{"//b", slashesMessage},
		{"a/\xff", "path name is not valid UTF-8"},
	}

	for _, test := range tests {
		_, err := fs.AbsolutePath(t.Context(), test.name)
		if err != nil {
			if _, ok := errors.AsType[*connectors.InvalidPathError](err); !ok {
				t.Errorf("%q: expected *connectors.InvalidPathError, got %T (%s)", test.name, err, err)
				continue
			}
			if err.Error() != test.message {
				t.Errorf("%q: expected message %q, got %q", test.name, test.message, err.Error())
			}
			continue
		}
		t.Errorf("%q: expected *connectors.InvalidPathError, got nil", test.name)
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
			t.Run("Path outside the root", func(t *testing.T) {
				for _, path := range []string{filepath.Join(filepath.Dir(root), "input/source"), root + "x/a"} {
					err := rewritePathError(&fs.PathError{Op: "read", Path: path, Err: fs.ErrClosed})
					pErr, ok := err.(*fs.PathError)
					if !ok {
						t.Fatalf("expected *fs.PathError, got %T", err)
					}
					if pErr.Path != path {
						t.Fatalf("expected path %q, got %q", path, pErr.Path)
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
