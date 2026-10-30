// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"encoding/base64"
	"sync"
	"testing"

	"github.com/krenalis/krenalis/tools/kms"
)

// TestInitIfEmptyConcurrent verifies that concurrent calls to InitIfEmpty on
// the same empty database succeed and initialize it only once.
func TestInitIfEmptyConcurrent(t *testing.T) {
	database := newTestDatabase(t)
	key, err := kms.New(t.Context(), "key:"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	const instances = 4
	errs := make([]error, instances)
	var wg sync.WaitGroup
	for i := range instances {
		wg.Go(func() {
			errs[i] = InitIfEmpty(t.Context(), database, key, false)
		})
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("expected no error from instance %d, got %v", i, err)
		}
	}
	var organizations int
	err = database.QueryRow(t.Context(), "SELECT COUNT(*) FROM organizations").Scan(&organizations)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if organizations != 1 {
		t.Fatalf("expected 1 organization, got %d", organizations)
	}
}
