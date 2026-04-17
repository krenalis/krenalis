// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cipher

import (
	"sync"
	"time"
)

var (
	ttl            = 10 * time.Second
	pruneFrequency = 5 * time.Second
	maxCachedKeys  = 1024
)

// cache stores plaintext data keys in memory for a limited time.
// It is safe for concurrent use by multiple goroutines.
type cache struct {
	mu    sync.Mutex
	keys  map[string]*clearKey
	timer *time.Timer
	done  chan struct{}
}

// clearKey is a plaintext data key, optionally tracked by a cache.
type clearKey struct {
	cache        *cache    // it is nil if the key is not in cache.
	Value        []byte    // immutable Value
	lastAccessed time.Time // access must be protected by cache.mu
	evictAfter   time.Time // access must be protected by cache.mu
	uses         int       // access must be protected by cache.mu
}

// Done marks the key as no longer in use.
// It must be called once for each Get or Put that returned this key.
func (ck *clearKey) Done() {
	// If the key is not associated with a cache,
	// clear it immediately and return.
	if ck.cache == nil {
		clear(ck.Value)
		return
	}
	ck.cache.mu.Lock()
	ck.uses--
	overflow := ck.uses < 0
	ck.cache.mu.Unlock()
	if overflow {
		panic("cipher: clearKey use count underflow")
	}
}

// newCache returns a new cache instance.
func newCache() *cache {
	c := &cache{
		keys:  make(map[string]*clearKey),
		timer: time.NewTimer(pruneFrequency),
		done:  make(chan struct{}),
	}
	go func() {
		for {
			select {
			case <-c.timer.C:
				now := time.Now()
				c.mu.Lock()
				c.prune(now)
				c.mu.Unlock()
				c.timer.Reset(pruneFrequency)
			case <-c.done:
				close(c.done)
				return
			}
		}
	}()
	return c
}

// Close stops background pruning and clears all stored keys.
//
// Close must not be called concurrently with any other cache method, and it
// must be called at most once. The cache must not be used after Close.
func (c *cache) Close() {
	// Terminate the prune goroutine.
	c.done <- struct{}{}
	select {
	case <-c.done:
	}
	// Clear the cached keys.
	for _, clearKey := range c.keys {
		clear(clearKey.Value[:])
	}
	clear(c.keys)
}

// Exists reports whether encryptedKey is present in the cache.
// If so, it returns the size of the associated value
func (c *cache) Exists(encryptedKey []byte) (int, bool) {
	c.mu.Lock()
	ck, ok := c.keys[string(encryptedKey)]
	c.mu.Unlock()
	if ok {
		return len(ck.Value), true
	}
	return 0, false
}

// Get returns the key associated with encryptedKey.
// The returned boolean reports whether the key was found.
// A successful lookup extends the entry's retention window.
//
// The caller must call Done on the returned key when no longer needed.
func (c *cache) Get(encryptedKey []byte) (*clearKey, bool) {
	now := time.Now()
	evictAfter := now.Add(ttl)
	c.mu.Lock()
	defer c.mu.Unlock()
	ck, ok := c.keys[string(encryptedKey)]
	if !ok {
		return nil, false
	}
	ck.lastAccessed = now
	ck.evictAfter = evictAfter
	ck.uses++
	if ck.uses < 0 {
		panic("cipher: key usage limit exceeded")
	}
	return ck, true
}

// Put associates value with encryptedKey and extends the key's retention
// window.
//
// Ownership of value is transferred to the cache. The caller must not read or
// modify value after calling Put, and should use the returned key instead.
//
// The caller must call Done on the returned key when no longer needed.
func (c *cache) Put(encryptedKey, value []byte) *clearKey {
	now := time.Now()
	evictAfter := now.Add(ttl)
	c.mu.Lock()
	defer c.mu.Unlock()
	if ck, ok := c.keys[string(encryptedKey)]; ok {
		ck.lastAccessed = now
		ck.evictAfter = evictAfter
		ck.uses++
		if ck.uses < 0 {
			panic("cipher: key usage limit exceeded")
		}
		// Zeroes value unless it shares the same underlying array as ck.Value.
		if &value[0] != &ck.Value[0] {
			clear(value)
		}
		return ck
	}
	if len(c.keys) < maxCachedKeys || c.evictOne(now) {
		ck := &clearKey{
			cache:        c,
			Value:        value,
			lastAccessed: now,
			evictAfter:   evictAfter,
			uses:         1,
		}
		c.keys[string(encryptedKey)] = ck
		return ck
	}
	return &clearKey{Value: value}
}

// evictOne removes a single evictable entry to free space. It prefers expired
// entries; otherwise, it removes the least recently accessed entry with no
// active uses. It reports whether an entry was removed.
func (c *cache) evictOne(now time.Time) bool {
	var victimKey string
	var victim *clearKey
	for key, ck := range c.keys {
		if ck.uses > 0 {
			continue
		}
		if !ck.evictAfter.After(now) {
			victimKey = key
			victim = ck
			break
		}
		if victim == nil || ck.lastAccessed.Before(victim.lastAccessed) {
			victimKey = key
			victim = ck
		}
	}
	if victim == nil {
		return false
	}
	clear(victim.Value)
	delete(c.keys, victimKey)
	return true
}

// prune removes expired entries with no active uses.
// It must be called with c.mu held.
// It is invoked automatically and is not intended to be called by callers.
func (c *cache) prune(now time.Time) bool {
	pruned := false
	for key, ck := range c.keys {
		if ck.uses == 0 && !ck.evictAfter.After(now) {
			clear(ck.Value)
			delete(c.keys, key)
			pruned = true
		}
	}
	return pruned
}
