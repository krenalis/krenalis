// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package ratelimiter implements process-local rate limiting using capacity
// leases acquired from PostgreSQL.
//
// A Bucket holds capacity for one subject, identified by a SubjectKind and an
// identifier. The single platform subject uses "platform" as its canonical
// identifier. A Limiter batches refill requests and obtains capacity from
// PostgreSQL. The local bucket logic does not interpret subject kinds; the
// PostgreSQL lease store validates the kinds supported by Krenalis.
//
// # Krenalis quota model
//
// Krenalis has one global budget for requests to the platform management API.
// It also gives each organization a separate organization-level budget for
// normal requests and each workspace independent budgets for normal requests
// and event ingestion. Every rate-limited request is subject to exactly one of
// these budgets. Organization and workspace budgets belong to those subjects,
// not to API keys. The global platform budget belongs to the single platform
// subject.
//
// A request to the platform management API consumes the global platform budget.
// A normal request without a workspace consumes the authenticated
// organization's organization-level request budget. A normal request associated
// with a workspace consumes that workspace's request budget. Event ingestion
// consumes the workspace's event budget. A workspace can be selected through a
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
// Consume returns a CapacityExceededError when the requested capacity is
// unavailable. It returns ErrLimiterUnavailable when a temporary condition
// makes capacity availability impossible to determine. This includes
// acquisition timeouts, active operational backoff, saturation of the refill
// queue, and insufficient remaining admission budget in the current refill
// generation. Invalid or incomplete acquisition responses are internal failures
// and are returned without being converted to CapacityExceededError or
// ErrLimiterUnavailable.
//
// A CapacityExceededError carries an advisory retry duration. The duration is
// calculated from the capacity remaining in the local bucket, the authoritative
// PostgreSQL state, and the fractional refill remainder. For waiters in the
// rejected FIFO suffix that can eventually be served, retry durations are
// calculated cumulatively. The retry duration for a later waiter therefore
// includes the capacity required by earlier waiters. A request larger than the
// reported bucket capacity receives no positive retry duration.
//
// The retry duration is not a reservation. It is the earliest FIFO schedule
// calculated from the state observed when the lease completes. Concurrent
// consumption, refills already in progress, configuration changes, and the
// actual retry time may require the caller to wait longer.
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
// It is not necessarily the bucket's lease size. When an operation triggers a
// refill, the admission budget includes the units required by that operation
// and, if the lease size permits, up to 10 additional units of baseline
// headroom for other waiters. Positive local capacity is excluded because
// unrelated local requests may consume it before the lease arrives. A partial
// grant serves only the first consecutive FIFO waiters that can be satisfied.
// This deliberately prevents an operation requiring fewer units from being
// served before an older operation requiring more. Cancellation removes a
// pending waiter and returns its reserved units to the admission budget for
// later waiters.
//
// If a request cannot be admitted because the refill generation has
// insufficient remaining admission budget, no authoritative lease result is
// available. This is local saturation, not evidence that the authoritative
// quota is exhausted. Consume therefore returns ErrLimiterUnavailable rather
// than CapacityExceededError.
//
// A caller may Restore units that it consumed but ultimately did not use.
// Restoration fills the local bucket up to its target. Capacity that no longer
// fits locally, for example because an in-flight refill has filled the bucket
// to its target, is queued for best-effort restoration to PostgreSQL. Restore
// neither cancels refills nor wakes waiters.
//
// PostgreSQL restorations are aggregated by subject and processed
// asynchronously in bounded batches. A failed batch is not retried because an
// interrupted database operation can have an ambiguous commit status; retrying
// it could restore the same capacity twice. A failure may therefore cause
// capacity to be lost, which is the conservative outcome, but it cannot create
// capacity. Restore validates and queues the operation synchronously, so a nil
// error does not guarantee that the asynchronous database operation will
// succeed.
//
// Pending restoration units are aggregated per subject and capped at the
// maximum authoritative bucket capacity. Units above this limit are discarded
// instead of being retained as deferred restoration credit. Otherwise, the
// queued excess could be restored after earlier capacity has been consumed,
// allowing unused leases to accumulate outside the token bucket and act as
// additional stored capacity. Discarding the excess may conservatively lose
// capacity, but it cannot create additional capacity.
//
// Each local target is capped by both the lease size and the capacity reported
// by PostgreSQL. A bucket without a recent positive grant uses a baseline target
// of at most 10 units. If the next refill is needed within five seconds of a
// positive grant, the target doubles, up to the lease size. If more than five
// seconds have passed, or if the previous refill granted no capacity, the next
// refill uses a target of at most 10 units. An operation that triggers a refill
// raises the target to include both the units it requires and, where possible,
// the baseline admission headroom.
//
// Lowering the target does not discard capacity already held locally. If the
// existing local capacity is sufficient for the lower target, the limiter lets
// operations drain that capacity before acquiring another lease. There is no
// idle timer, so a bucket that receives no further operations retains its
// unused local capacity. A refill is normally considered when an operation
// cannot be served, when local capacity falls below a threshold calculated from
// the current target, or when an operation leaves no more capacity than it
// consumed. No lease is acquired if the existing local capacity already
// satisfies the lower adaptive target.
//
// If the reported capacity causes the local target to decrease, any capacity
// already held above the new target is revoked and discarded. It is not
// returned to PostgreSQL because it was leased under the previous, higher
// target. In this case, only capacity from the new grant that does not fit
// within the new target is restored asynchronously.
//
// # PostgreSQL safety and batching
//
// PostgreSQL stores authoritative capacity and is the only shared coordination
// mechanism between application nodes. The limiter batches lease requests for
// platform, organization, workspace, and event-ingestion buckets. PostgreSQL
// locks the corresponding rows and subtracts granted capacity before returning
// it. A process crash can lose an unused lease but cannot create additional
// capacity.
//
// Leased capacity is deducted from PostgreSQL before it is consumed locally.
// Because PostgreSQL may refill while earlier leases remain unused, the
// configured maximum capacity bounds the authoritative bucket and each local
// target, but does not strictly bound the total number of operations performed
// within a short time interval.
//
// The refiller collects generations for a short interval, up to a fixed batch
// size. Lease acquisition has its own finite deadline, independent of waiter
// deadlines, so a stuck query cannot block the single refiller indefinitely.
// A response returned without an acquisition error is still validated even if
// the acquisition context has been canceled or its deadline has expired. If the
// response is complete and valid, it is applied.
//
// The complete acquisition response is validated before any capacity is
// applied. Every requested subject must have exactly one matching result.
// In an otherwise valid response, a missing subject has all numeric result
// fields set to zero and causes only its refill generation to fail with
// ErrLimiterUnavailable. For existing subjects, grants must be non-negative and
// must not exceed either the requested amount or the reported capacity.
// Reported capacity must be positive. Authoritative available capacity must be
// non-negative, and the sum of authoritative available capacity and granted
// capacity must not exceed the reported capacity. The refill rate must be
// positive, and the fractional refill remainder must be in the range
// [0, 60,000,000), which matches the number of microseconds per minute used as
// the denominator in the refill calculation. An invalid or incomplete response
// causes all requests in the local batch to fail.
//
// Acquisition errors and invalid responses start a short global backoff that
// stores the corresponding operational or internal error. A call made while
// backoff is active may still use positive local capacity, but it neither
// publishes a new refill nor admits a waiter. If the call cannot use local
// capacity, it returns the stored error. If backoff is still active when the
// refiller processes a queued generation, the refiller rejects that generation
// with the stored error. Shutdown cancellation does not start backoff.
//
// # Bucket lifetime and shutdown
//
// Buckets are not closed. An owner stops using a bucket when its organization
// or workspace is no longer canonical. A queued refill, an acquisition in
// progress, or an admitted waiter may temporarily keep the bucket reachable.
// After those references are released, Go collects it normally. Unused capacity
// from buckets for deleted organizations and workspaces does not need to be
// restored because their authoritative PostgreSQL rows are removed by cascading
// deletion.
//
// Limiter.Close has a strict lifecycle precondition. Before calling it, the
// caller must stop all use of the limiter and every bucket created by it. No
// exported Limiter or Bucket method may be in progress. The caller must keep
// buckets for existing subjects reachable until Close returns. Close cancels
// lease acquisition, stops background work, and discards queued refills. It
// waits for all limiter operations, including asynchronous restorations, to
// stop even if the context passed to Close expires or is canceled.
//
// After background work stops, Close makes one best-effort restoration attempt.
// It includes unused local capacity from buckets that are still reachable and
// queued excess capacity that has not yet been submitted to PostgreSQL. The
// restoration uses the caller's context. It is not retried after an error or
// cancellation, and it ignores subjects that have been deleted from PostgreSQL.
// If the context is already canceled, Close does not start the restoration. If
// cancellation occurs during restoration, Close waits for the restoration
// operation to stop. A process crash or cancellation during shutdown can
// therefore still cause unused capacity to be lost.
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
//   - Granted capacity is added to the current local value and capped at the
//     newly reported local target. Any excess from the new grant is returned
//     asynchronously.
//   - Capacity assigned to a waiter is deducted before notification.
//   - Retry scheduling accounts for remaining local and authoritative capacity.
//   - Feasible waiters in a rejected FIFO suffix receive cumulative retry
//     durations; requests above the reported capacity receive no retry hint.
//   - Errors and invalid responses never add capacity.
//   - Generation identity prevents stale results from affecting newer work.
//   - Every asynchronous or shutdown restoration batch is attempted at most
//     once and a failed or interrupted restoration is not retried.
package ratelimiter
