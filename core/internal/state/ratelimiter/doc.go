// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package ratelimiter implements process-local rate limiting using capacity
// leases acquired from PostgreSQL.
//
// A Bucket holds capacity for one subject, identified by a SubjectKind and an
// identifier. A Limiter batches refill requests and obtains capacity from
// PostgreSQL. The local bucket logic does not interpret subject kinds; the
// PostgreSQL lease store validates the kinds supported by Krenalis.
//
// # Krenalis quota model
//
// Krenalis gives each workspace independent budgets for normal requests and
// event ingestion. It also gives each organization a separate
// organization-level budget for normal requests. Every rate-limited request is
// subject to exactly one of these budgets. Budgets belong to organizations and
// workspaces, not to API keys.
//
// Event ingestion consumes the workspace's event budget. A normal request
// associated with a workspace consumes that workspace's request budget. A
// normal request without a workspace consumes the authenticated organization's
// organization-level request budget. A workspace can be selected through a
// workspace-scoped endpoint, a workspace-bound API key, or the
// "Krenalis-Workspace" header. Organization-only endpoints reject requests
// associated with a workspace instead of charging the organization budget.
//
// # Local consumption and refills
//
// Consume never accesses PostgreSQL on the caller's goroutine. If sufficient
// local capacity is available, it deducts the requested units immediately. If
// local capacity is insufficient, Consume may admit the request to the current
// refill and wait only for that refill. A request that cannot be admitted
// returns an ErrLimiterUnavailable error without consuming any remaining local
// capacity.
//
// Units are internal values selected or derived by caller code. They must be
// between 1 and the bucket's configured maximum. Consume and Restore return an
// error for an invalid value. Caller cancellation and deadlines do not prevent
// local consumption or the publication of a refill. If cancellation wins the
// race with refill completion, the corresponding context error is returned
// unchanged. Every waiter also has a fixed maximum wait duration. If Consume
// stops waiting because that duration expires, it returns an
// ErrLimiterUnavailable error.
//
// Consume returns an ErrCapacityExceeded error only after a successful
// acquisition confirms that the requested capacity is unavailable. It returns
// an ErrLimiterUnavailable error when a temporary condition prevents the
// limiter from determining whether capacity is available. This includes
// acquisition timeouts, active operational backoff, and local queue saturation.
// Invalid or incomplete acquisition responses are internal failures and are
// returned without being converted to ErrCapacityExceeded or
// ErrLimiterUnavailable.
//
// A bucket has at most one refill generation. Its immutable lease request and
// any waiter admitted for the operation that triggered the refill are stored
// while holding the bucket mutex. This happens before the generation is
// published to the limiter queue. As a result, every published generation is
// already fully initialized and does not require a separate activation phase.
// If publication fails, the generation and all its waiters are rejected with
// an ErrLimiterUnavailable error.
//
// Waiters are admitted in FIFO order, up to the number of units requested by
// the generation. While holding the bucket mutex, the limiter applies the
// lease, assigns capacity to the first consecutive FIFO waiters that can be
// served, and detaches the refill generation. Capacity assigned to an
// authorized waiter is deducted before that waiter is notified.
//
// The admission budget is the number of units requested when the refill starts.
// It is not the bucket's lease size. Positive local capacity is excluded
// because unrelated local requests may consume it before the lease arrives.
// A partial grant serves only the first consecutive FIFO waiters that can be
// satisfied. This deliberately prevents operations requiring fewer units from
// being served before an older operation requiring more. Cancellation removes
// a pending waiter and returns its reserved units to the admission budget for
// later waiters.
//
// A caller may Restore units that it consumed but ultimately did not use.
// Restoration affects only local capacity and cannot raise it above the local
// target. It neither cancels refills nor wakes waiters.
//
// Each local target is capped by both the lease size and the capacity reported
// by PostgreSQL. A refill is normally prepared when an operation cannot be
// served, when local capacity falls below a threshold calculated from the local
// target, or when an operation leaves no more capacity than it consumed.
// Targets and thresholds are intentionally simple and do not adapt to traffic
// rate or acquisition latency.
//
// # PostgreSQL safety and batching
//
// PostgreSQL stores authoritative capacity and is the only shared coordination
// mechanism between application nodes. The limiter batches organization,
// workspace, and event requests. PostgreSQL locks the corresponding rows and
// subtracts granted capacity before returning it. A process crash can lose an
// unused lease but cannot create additional capacity.
//
// The batcher collects generations for a short interval, up to a fixed batch
// size. Lease acquisition has its own finite deadline, independent of waiter
// deadlines, so a stuck query cannot block the single batcher indefinitely.
//
// The complete acquisition response is validated before any capacity is
// applied. Every requested subject must have exactly one matching result.
// Grants must be non-negative, must not exceed either the request or the
// reported capacity, and the reported capacity must be positive. An invalid or
// incomplete response causes all requests in the local batch to fail.
//
// Acquisition errors and invalid responses start a short global backoff that
// stores the corresponding operational or internal error. A call made while
// backoff is active may still use positive local capacity, but it neither
// publishes a new refill nor admits a waiter. If the call cannot use local
// capacity, it returns the stored error. If backoff is still active when the
// batcher processes a queued generation, it rejects that generation with the
// stored error. Shutdown cancellation does not start backoff.
//
// # Bucket lifetime and shutdown
//
// Buckets are not closed. An owner stops using a bucket when its organization
// or workspace is no longer canonical. A queued refill, an acquisition in
// progress, or an admitted waiter may temporarily keep the bucket reachable.
// After those references are released, Go collects it normally. Buckets for
// deleted organizations and workspaces do not need their unused capacity
// restored because PostgreSQL removes their authoritative rows by cascade.
//
// Limiter.Close has a strict lifecycle precondition. Before calling it, the
// caller must stop all use of the limiter and every bucket created by it. No
// exported Limiter or Bucket method may be in progress. The caller keeps
// buckets for existing subjects reachable until Close returns. Close cancels
// lease acquisition, stops the batcher, and discards queued refills. It waits
// until the batcher stops or the context passed to Close ends. If the context
// ends first, the batcher remains in progress and no capacity is restored.
//
// After the batcher stops, Close makes one best-effort attempt to restore unused
// local capacity from reachable buckets. The restoration uses the caller's
// context, is not retried after an error or cancellation, and ignores subjects
// deleted from PostgreSQL. A process crash or an interrupted shutdown can
// therefore still lose unused capacity.
//
// # Important invariants
//
// Future changes should preserve these properties unless they deliberately
// redefine the limiter's guarantees:
//
//   - Consume never accesses PostgreSQL on the caller's goroutine.
//   - Each Bucket has at most one local refill generation.
//   - A batch contains at most one refill for each subject.
//   - The bucket mutex protects capacity, refill, and waiter state.
//   - No bucket mutex is held during queue publication or lease acquisition.
//   - Local available capacity never becomes negative.
//   - Pending waiter units never exceed the units requested when the refill
//     starts.
//   - A waiter belongs to one refill generation and is resolved exactly once.
//   - Granted capacity is added to the current local value, capped at the newly
//     reported local target.
//   - Capacity assigned to a waiter is deducted before notification.
//   - Errors and invalid responses never add capacity.
//   - Generation identity prevents stale results from affecting newer work.
//   - Shutdown restores unused local capacity at most once and does not retry a
//     failed or interrupted restoration.
package ratelimiter
