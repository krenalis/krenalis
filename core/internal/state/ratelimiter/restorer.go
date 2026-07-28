// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"context"
	"sync"
	"time"
)

const (
	// maxRestoreBatchSize is the maximum number of subjects restored by one
	// PostgreSQL call.
	maxRestoreBatchSize = 64
	// maxRestoreWorkers limits the number of concurrent PostgreSQL calls during
	// shutdown.
	maxRestoreWorkers = 4
	// maxRestoredUnits is the largest restoration accepted for one subject. It
	// matches the maximum event-bucket capacity allowed by the schema.
	maxRestoredUnits = 100_000
)

// defaultRestorationTimeout limits the duration of one asynchronous PostgreSQL
// restoration.
var defaultRestorationTimeout = 5 * time.Second

// restorationQueue collects capacity that no longer fits in local buckets and
// aggregates it by subject until the background restorer returns it to
// PostgreSQL.
type restorationQueue struct {
	sync.Mutex
	pending map[subjectKey]int
	wake    chan struct{}
}

// newRestorationQueue creates an empty queue with a wake-up channel that
// coalesces repeated notifications.
func newRestorationQueue() restorationQueue {
	return restorationQueue{
		pending: make(map[subjectKey]int),
		wake:    make(chan struct{}, 1),
	}
}

// add adds capacity to the queue and notifies the background restorer.
func (queue *restorationQueue) add(key subjectKey, units int) {
	queue.Lock()
	if queue.pending == nil {
		queue.pending = make(map[subjectKey]int)
	}
	current := queue.pending[key]
	if units >= maxRestoredUnits-current {
		queue.pending[key] = maxRestoredUnits
	} else {
		queue.pending[key] = current + units
	}
	wake := queue.wake
	queue.Unlock()
	if wake != nil {
		select {
		case wake <- struct{}{}:
		default:
		}
	}
}

// mergeInto merges queued capacity into capacity collected from local buckets.
// It does not remove entries from the queue.
func (queue *restorationQueue) mergeInto(available map[subjectKey]int) {
	queue.Lock()
	defer queue.Unlock()
	for key, units := range queue.pending {
		available[key] = min(maxRestoredUnits, available[key]+units)
	}
}

// takeBatch takes one batch and removes it from the queue. Each removed batch
// is attempted at most once because a PostgreSQL error may leave its commit
// status ambiguous.
func (queue *restorationQueue) takeBatch() []unusedCapacity {
	queue.Lock()
	defer queue.Unlock()
	batch := make([]unusedCapacity, 0, min(maxRestoreBatchSize, len(queue.pending)))
	for key, units := range queue.pending {
		batch = append(batch, unusedCapacity{SubjectKind: key.kind, SubjectID: key.id, Units: units})
		delete(queue.pending, key)
		if len(batch) == maxRestoreBatchSize {
			break
		}
	}
	return batch
}

// restoreBatchFunc restores unused capacity for one batch of subjects.
type restoreBatchFunc func(context.Context, []unusedCapacity) error

// unusedCapacity identifies unused node-local capacity for one subject.
type unusedCapacity struct {
	SubjectKind SubjectKind `json:"subject_kind"`
	SubjectID   string      `json:"subject_id"`
	Units       int         `json:"units"`
}
