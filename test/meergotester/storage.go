// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package meergotester

import (
	"os"
	"testing"
)

// TempStorage represents a temporary storage for the tests.
type TempStorage struct {
	t    *testing.T
	root string
}

// NewTempStorage returns a new temporary storage that can be used in tests.
//
// Use the Remove method to remove the storage when the test ends. If not called
// (eg. the test failed) the storage is kept; this can be useful to debug the
// test.
func NewTempStorage(t *testing.T) *TempStorage {
	root, err := os.MkdirTemp("", "meergo-test-storage")
	if err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if !stat.IsDir() {
		t.Fatalf("%q is not a dir", root)
	}
	t.Logf("created temporary directory for the storage: %q", root)
	return &TempStorage{
		t:    t,
		root: root,
	}
}

// Root returns the root dir of the temporary storage.
func (tmp *TempStorage) Root() string {
	return tmp.root
}

// Remove removes the temporary storage root directory and its contents.
func (tmp *TempStorage) Remove() {
	err := os.RemoveAll(tmp.root)
	if err != nil {
		tmp.t.Logf("cannot remove the temporary directory used by the storage: %s", err)
	}
}
