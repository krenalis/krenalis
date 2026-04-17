// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cipher

import (
	"bytes"
	"testing"
	"time"
)

// TestCachePutGetDone verifies inserting, retrieving, and releasing a cached
// key.
func TestCachePutGetDone(t *testing.T) {
	t.Parallel()

	c := &cache{keys: make(map[string]*clearKey)}
	encryptedKey := []byte("key")
	value := []byte("plaintext-key")

	putKey := c.Put(encryptedKey, value)
	if putKey.cache != c {
		t.Fatalf("expected cache %p, got %p", c, putKey.cache)
	}
	if putKey.uses != 1 {
		t.Fatalf("expected uses 1, got %d", putKey.uses)
	}

	gotKey, ok := c.Get(encryptedKey)
	if !ok {
		t.Fatal("expected cached key, got cache miss")
	}
	if gotKey != putKey {
		t.Fatalf("expected key %p, got %p", putKey, gotKey)
	}
	if gotKey.uses != 2 {
		t.Fatalf("expected uses 2, got %d", gotKey.uses)
	}

	gotKey.Done()
	if putKey.uses != 1 {
		t.Fatalf("expected uses 1, got %d", putKey.uses)
	}

	putKey.Done()
	if putKey.uses != 0 {
		t.Fatalf("expected uses 0, got %d", putKey.uses)
	}

}

// TestCacheExists verifies Exists reports presence and value size.
func TestCacheExists(t *testing.T) {
	t.Parallel()

	c := &cache{keys: make(map[string]*clearKey)}
	encryptedKey := []byte("key")

	size, ok := c.Exists(encryptedKey)
	if ok {
		t.Fatalf("expected cache miss, got hit with size %d", size)
	}
	if size != 0 {
		t.Fatalf("expected size 0, got %d", size)
	}

	value := []byte("plaintext-key")
	cachedKey := c.Put(encryptedKey, value)
	cachedKey.Done()
	uses := cachedKey.uses

	size, ok = c.Exists(encryptedKey)
	if !ok {
		t.Fatal("expected cached key, got cache miss")
	}
	if size != len(cachedKey.Value) {
		t.Fatalf("expected size %d, got %d", len(cachedKey.Value), size)
	}
	if cachedKey.uses != uses {
		t.Fatalf("expected uses %d, got %d", uses, cachedKey.uses)
	}
}

// TestCachePutExistingKeyReusesEntryAndClearsInput verifies that Put reuses an
// existing entry.
func TestCachePutExistingKeyReusesEntryAndClearsInput(t *testing.T) {
	t.Parallel()

	c := &cache{keys: make(map[string]*clearKey)}
	encryptedKey := []byte("key")

	originalValue := []byte("original")
	originalKey := c.Put(encryptedKey, originalValue)
	originalKey.Done()

	originalLastAccessed := originalKey.lastAccessed
	originalEvictAfter := originalKey.evictAfter
	replacementValue := []byte("replacement")

	reusedKey := c.Put(encryptedKey, replacementValue)
	if reusedKey != originalKey {
		t.Fatalf("expected key %p, got %p", originalKey, reusedKey)
	}
	if !bytes.Equal(reusedKey.Value, originalValue) {
		t.Fatalf("expected value %q, got %q", originalValue, reusedKey.Value)
	}
	if reusedKey.uses != 1 {
		t.Fatalf("expected uses 1, got %d", reusedKey.uses)
	}
	if reusedKey.lastAccessed.Before(originalLastAccessed) {
		t.Fatalf("expected lastAccessed at or after %v, got %v", originalLastAccessed, reusedKey.lastAccessed)
	}
	if reusedKey.evictAfter.Before(originalEvictAfter) {
		t.Fatalf("expected evictAfter at or after %v, got %v", originalEvictAfter, reusedKey.evictAfter)
	}
	if !bytes.Equal(replacementValue, make([]byte, len(replacementValue))) {
		t.Fatalf("expected cleared replacement value, got %v", replacementValue)
	}

}

// TestCachePruneRemovesOnlyExpiredUnusedKeys verifies that prune removes only
// expired keys with no active uses.
func TestCachePruneRemovesOnlyExpiredUnusedKeys(t *testing.T) {
	t.Parallel()

	now := time.Now()
	expiredUnusedValue := []byte("expired-unused")
	expiredInUseValue := []byte("expired-in-use")
	freshUnusedValue := []byte("fresh-unused")

	c := &cache{
		keys: map[string]*clearKey{
			"expired-unused": {
				Value:      expiredUnusedValue,
				evictAfter: now.Add(-time.Second),
			},
			"expired-in-use": {
				Value:      expiredInUseValue,
				evictAfter: now.Add(-time.Second),
				uses:       1,
			},
			"fresh-unused": {
				Value:      freshUnusedValue,
				evictAfter: now.Add(time.Second),
			},
		},
	}

	pruned := c.prune(now)
	if !pruned {
		t.Fatal("expected prune to remove at least one key, got no removals")
	}
	if _, ok := c.keys["expired-unused"]; ok {
		t.Fatal("expected expired unused key to be removed, got present key")
	}
	if _, ok := c.keys["expired-in-use"]; !ok {
		t.Fatal("expected expired in-use key to remain, got missing key")
	}
	if _, ok := c.keys["fresh-unused"]; !ok {
		t.Fatal("expected fresh unused key to remain, got missing key")
	}
	if !bytes.Equal(expiredUnusedValue, make([]byte, len(expiredUnusedValue))) {
		t.Fatalf("expected cleared expired value, got %v", expiredUnusedValue)
	}

}

// TestCacheEvictOnePrefersExpiredOtherwiseLRU verifies eviction order for
// expired and least recently used keys.
func TestCacheEvictOnePrefersExpiredOtherwiseLRU(t *testing.T) {
	t.Parallel()

	t.Run("prefers expired key", func(t *testing.T) {
		now := time.Now()
		expiredValue := []byte("expired")
		freshValue := []byte("fresh")
		c := &cache{
			keys: map[string]*clearKey{
				"expired": {
					Value:        expiredValue,
					lastAccessed: now.Add(-time.Second),
					evictAfter:   now.Add(-time.Millisecond),
				},
				"fresh": {
					Value:        freshValue,
					lastAccessed: now.Add(-time.Hour),
					evictAfter:   now.Add(time.Hour),
				},
			},
		}

		evicted := c.evictOne(now)
		if !evicted {
			t.Fatal("expected eviction, got no eviction")
		}
		if _, ok := c.keys["expired"]; ok {
			t.Fatal("expected expired key to be evicted, got present key")
		}
		if _, ok := c.keys["fresh"]; !ok {
			t.Fatal("expected fresh key to remain, got missing key")
		}
		if !bytes.Equal(expiredValue, make([]byte, len(expiredValue))) {
			t.Fatalf("expected cleared expired value, got %v", expiredValue)
		}
	})

	t.Run("evicts least recently used key", func(t *testing.T) {
		now := time.Now()
		lruValue := []byte("lru")
		newerValue := []byte("newer")
		c := &cache{
			keys: map[string]*clearKey{
				"lru": {
					Value:        lruValue,
					lastAccessed: now.Add(-2 * time.Hour),
					evictAfter:   now.Add(time.Hour),
				},
				"newer": {
					Value:        newerValue,
					lastAccessed: now.Add(-time.Hour),
					evictAfter:   now.Add(time.Hour),
				},
			},
		}

		evicted := c.evictOne(now)
		if !evicted {
			t.Fatal("expected eviction, got no eviction")
		}
		if _, ok := c.keys["lru"]; ok {
			t.Fatal("expected least recently used key to be evicted, got present key")
		}
		if _, ok := c.keys["newer"]; !ok {
			t.Fatal("expected newer key to remain, got missing key")
		}
		if !bytes.Equal(lruValue, make([]byte, len(lruValue))) {
			t.Fatalf("expected cleared least recently used value, got %v", lruValue)
		}
	})
}

// TestCachePutWhenFullAndNothingEvictableReturnsUncachedKey verifies Put
// fallback when the cache is full.
func TestCachePutWhenFullAndNothingEvictableReturnsUncachedKey(t *testing.T) {
	originalMaxCachedKeys := maxCachedKeys
	maxCachedKeys = 1
	t.Cleanup(func() {
		maxCachedKeys = originalMaxCachedKeys
	})

	existingValue := []byte("existing")
	c := &cache{
		keys: map[string]*clearKey{
			"existing": {
				cache:        nil,
				Value:        existingValue,
				lastAccessed: time.Now(),
				evictAfter:   time.Now().Add(time.Hour),
				uses:         1,
			},
		},
	}
	c.keys["existing"].cache = c

	newValue := []byte("new")
	gotKey := c.Put([]byte("new"), newValue)
	if gotKey.cache != nil {
		t.Fatalf("expected uncached key, got cache %p", gotKey.cache)
	}
	if len(c.keys) != 1 {
		t.Fatalf("expected cache size 1, got %d", len(c.keys))
	}
	if _, ok := c.keys["new"]; ok {
		t.Fatal("expected new key to be absent from cache, got present key")
	}

	gotKey.Done()
	if !bytes.Equal(newValue, make([]byte, len(newValue))) {
		t.Fatalf("expected cleared uncached value, got %v", newValue)
	}
}

// TestCacheCloseClearsStoredKeys verifies that Close clears all cached values.
func TestCacheCloseClearsStoredKeys(t *testing.T) {
	t.Parallel()

	firstValue := []byte("first")
	secondValue := []byte("second")
	c := newCache()
	c.mu.Lock()
	c.keys["first"] = &clearKey{cache: c, Value: firstValue}
	c.keys["second"] = &clearKey{cache: c, Value: secondValue}
	c.mu.Unlock()

	c.Close()

	if !bytes.Equal(firstValue, make([]byte, len(firstValue))) {
		t.Fatalf("expected cleared first value, got %v", firstValue)
	}
	if !bytes.Equal(secondValue, make([]byte, len(secondValue))) {
		t.Fatalf("expected cleared second value, got %v", secondValue)
	}
	c.mu.Lock()
	keysLen := len(c.keys)
	c.mu.Unlock()
	if keysLen != 0 {
		t.Fatalf("expected cache size 0, got %d", keysLen)
	}

}
