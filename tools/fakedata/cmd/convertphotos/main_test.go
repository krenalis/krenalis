// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package main

import (
	"path/filepath"
	"testing"
)

// TestRunMissingLocalSelectionSeed rejects an omitted or empty local seed.
func TestRunMissingLocalSelectionSeed(t *testing.T) {
	directory := t.TempDir()
	err := run(t.Context(), filepath.Join(directory, "snapshot"), "", "",
		filepath.Join(directory, "base.json"), filepath.Join(directory, "socioeconomic.json"),
		filepath.Join(directory, "output"))
	if err != nil {
		if err.Error() != "-selection-seed is required with -local" {
			t.Fatalf("expected missing selection seed rejected before reading inputs, got %v", err)
		}
		return
	}
	t.Fatal("expected missing selection seed rejected, got success")
}
