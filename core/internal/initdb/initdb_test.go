// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"encoding/base64"
	"sync"
	"testing"

	"github.com/krenalis/krenalis/tools/kms"
)

// barrierKms is a kms.Kms that, on its first call to
// GenerateDataKeyWithoutPlaintext, waits until every other barrierKms sharing
// the same arrived wait group has reached it.
type barrierKms struct {
	kms.Kms
	once    sync.Once
	arrived *sync.WaitGroup
}

// GenerateDataKeyWithoutPlaintext waits for the other instances on its first
// call, then calls the method of the wrapped Kms.
func (k *barrierKms) GenerateDataKeyWithoutPlaintext(ctx context.Context, keyLen int) ([]byte, error) {
	k.once.Do(func() {
		k.arrived.Done()
		k.arrived.Wait()
	})
	return k.Kms.GenerateDataKeyWithoutPlaintext(ctx, keyLen)
}

// TestInitIfEmptyConcurrent verifies that concurrent calls to InitIfEmpty on
// the same empty database, all past its initial emptiness check, succeed and
// initialize it only once.
func TestInitIfEmptyConcurrent(t *testing.T) {
	database := newTestDatabase(t)
	key, err := kms.New(t.Context(), "key:"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	const instances = 4
	if n := database.PoolStats().MaxConns(); n < instances {
		t.Fatalf("expected at least %d pool connections, got %d", instances, n)
	}
	errs := make([]error, instances)
	var arrived sync.WaitGroup
	arrived.Add(instances)
	var wg sync.WaitGroup
	for i := range instances {
		wg.Go(func() {
			errs[i] = InitIfEmpty(t.Context(), database, &barrierKms{Kms: key, arrived: &arrived}, false)
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
