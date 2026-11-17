// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"sync"
	"testing"
)

// TestMigrateEmpty verifies that Migrate fails on an empty database.
func TestMigrateEmpty(t *testing.T) {
	database := newTestDatabase(t)
	err := Migrate(t.Context(), database)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestMigrateConcurrent verifies that concurrent calls to Migrate on the same
// initialized database succeed.
func TestMigrateConcurrent(t *testing.T) {
	database := newInitializedTestDatabase(t)
	const instances = 4
	errs := make([]error, instances)
	var wg sync.WaitGroup
	for i := range instances {
		wg.Go(func() {
			errs[i] = Migrate(t.Context(), database)
		})
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("expected no error from instance %d, got %v", i, err)
		}
	}
}
