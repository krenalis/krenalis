// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

// API rate limiter design
//
// Purpose and quota model
//
// The rate limiter gives each workspace independent budgets for API operations
// and ingestion, and gives each organization a separate budget for nonspecific
// API operations.
//
// API operations fall into three categories:
//
//   - workspace operations, which act on one specific workspace;
//   - ingestion operations, which write events to one specific workspace;
//   - nonspecific operations, which do not belong to a workspace-specific API
//     category and are not scoped to a workspace.
//
// Exactly one budget applies to each request. A workspace request does not also
// consume the organization's nonspecific budget. An ingestion request does not
// consume either normal API budget. An unscoped nonspecific request does not
// consume capacity assigned to any workspace. A rate-limited pipeline metrics
// request uses the workspace budget when authentication scopes it to one.
//
// Budgets belong to organizations and workspaces, not to individual API keys.
//
// Architecture overview
//
// PostgreSQL stores authoritative rate-limit capacity and is the only shared
// coordination mechanism between application nodes. Each node acquires chunks
// of capacity, called leases, from PostgreSQL and consumes them from process
// memory.
//
// Public consumption never accesses PostgreSQL directly. Requests normally
// complete using local capacity. When local capacity is insufficient, a request
// may wait for one refill that has already been successfully published to the
// refill queue, subject to a bounded admission budget and a finite deadline.
//
// The public consumption methods on core.Organization and core.Workspace accept
// a context so caller cancellation and deadlines propagate to the limiter.
// After decoding an ingestion request, the collector consumes capacity for its
// event count through the same context-aware path on the corresponding
// Workspace instance.
//
// Normal API operations support costs from 1 through 100. Ingestion operations
// use their event count as the cost, up to a maximum of 20,000. Invalid costs
// return ErrInvalidAPICost. Requests that cannot be served immediately or
// admitted as waiters return ErrAPICapacityExceeded. Caller cancellation and
// caller deadlines preserve the corresponding context error.
//
// Each Organization instance owns one nonspecific bucket. Each Workspace owns
// separate API and ingestion buckets. The shared rateLimiter owns the refill
// queue, batcher, PostgreSQL lease acquisition, global backoff, metrics, and
// shutdown lifecycle.
//
// Local consumption and refill generations
//
// Each bucket has at most one current refill. Every refill has:
//
//   - a unique generation identity;
//   - an immutable lease request for PostgreSQL;
//   - a FIFO list of admitted waiters.
//
// Generation identity prevents a late completion from an older refill from
// changing a newer refill or resolving its waiters.
//
// A refill moves through three states:
//
//   1. publishing: the immutable lease request has been prepared while holding
//      the bucket mutex, but publication to the refill queue has not yet been
//      confirmed;
//   2. active: publication succeeded and the refill is queued or in progress;
//   3. finished: the refill completed, failed, was canceled, or was discarded.
//
// Publishing to the refill queue never happens while holding the bucket mutex.
// After a successful non-blocking publication, the publisher reacquires the
// mutex and atomically activates the refill and decides whether to admit the
// request that created it.
//
// The batcher must not process a published refill until this activation
// decision is complete. Requests racing with the publishing phase may be
// rejected.
//
// If local capacity is sufficient, the request cost is deducted immediately.
// A low remaining balance may also prepare and publish a proactive refill.
// Failure to publish that proactive refill does not change the successful
// consumption decision.
//
// If local capacity is insufficient:
//
//   - an existing active refill may admit the request as a waiter;
//   - otherwise, a new refill is prepared and the request may be admitted only
//     after publication succeeds;
//   - active backoff, shutdown, a disabled bucket, a full refill queue, a race
//     with the publishing phase, or exhausted admission capacity causes the
//     request to be rejected immediately.
//
// An admitted request waits only for the refill generation to which it belongs.
// It is never transferred to a later refill.
//
// Waiting ends when:
//
//   - the refill finishes;
//   - the caller context is canceled or reaches its deadline;
//   - the limiter shuts down;
//   - the limiter's fixed maximum wait duration expires.
//
// Resolution is ordered by the bucket state transition. If caller cancellation
// removes the waiter before capacity is assigned, the caller receives the
// context error. If refill completion assigns capacity first, authorization is
// final.
//
// If the waiter is still pending and the caller context is already canceled
// when shutdown or the internal wait timeout is handled, the caller context
// error takes precedence.
//
// Admission budget
//
// The lease request is calculated and frozen before publication. Each request
// asks for no more than the bucket's configured lease size. A negative local
// balance may make the amount missing from the local target larger than one
// lease, but the request remains capped at one lease.
//
// Waiter admission is bounded by the actual frozen request, not by the
// configured lease size in isolation:
//
//   reservable = requested units - max(0, -local available)
//
// The total cost of current waiters plus the candidate waiter must not exceed
// reservable.
//
// This guarantees that a full grant can repay existing overdraft and serve all
// admitted waiters, even if positive local capacity is consumed while
// PostgreSQL is processing the request.
//
// Positive local capacity is deliberately excluded from the calculation because
// it remains available to unrelated local requests and may be consumed before
// the grant arrives.
//
// A partial or zero grant may still cause some or all admitted waiters to be
// rejected.
//
// A request that cannot be served completely from local capacity consumes none
// of that capacity before waiting. Positive residual capacity remains available
// to smaller operations that can complete immediately.
//
// Canceling a waiter removes it from the FIFO list and subtracts its cost from
// the pending total. The released admission capacity may then be used by a
// later waiter.
//
// If all waiters leave while the refill remains active, overdraft may become
// available again.
//
// Applying leases and serving waiters
//
// PostgreSQL may grant all, part, or none of a lease request. A valid grant is
// added to the current local balance rather than replacing it, because local
// traffic may continue while the database query is running.
//
// Applying a grant and allocating capacity to waiters is one atomic operation
// for the bucket:
//
//   1. verify that the result belongs to the current active generation;
//   2. add the granted units to local capacity;
//   3. inspect waiters in FIFO order;
//   4. deduct each authorized request cost before recording success;
//   5. at the first waiter that cannot be served, reject that waiter and every
//      waiter after it;
//   6. make every waiter decision final and detach the generation;
//   7. notify the waiters.
//
// Waiters do not wake merely to compete for the bucket again. Capacity assigned
// to an authorized waiter has already been deducted, so a concurrent request
// cannot consume it.
//
// The intentional FIFO head-of-line policy prevents smaller requests from
// bypassing an older, more expensive request.
//
// Cancellation and refill completion are serialized through the bucket state.
// If cancellation resolves the waiter first, the waiter is removed and consumes
// no later grant. If completion assigns capacity first, authorization is final.
// Every waiter is resolved exactly once.
//
// Capacity and PostgreSQL safety
//
// The batcher sends organization, workspace, and ingestion lease requests in
// the same call to acquire_api_rate_limit_leases.
//
// PostgreSQL reads the current limits from the authoritative domain tables,
// locks each relevant bucket, calculates the capacity currently available for
// refill, and subtracts the granted amount before returning it to the node.
//
// This preserves the main distributed safety property:
//
//   a process crash may lose unused leased capacity, but it cannot create
//   additional capacity.
//
// Unused leases are not returned during shutdown. Safe return would require
// persistent lease identities and fencing. A best-effort return could race with
// local consumption and credit the same capacity twice.
//
// Batching and validation
//
// One batcher goroutine collects refill generations for a short interval, up to
// the configured batch size. PostgreSQL processes bucket rows in deterministic
// subject order to reduce deadlock risk between application nodes.
//
// Lease acquisition has its own finite deadline, independent of waiter
// deadlines, so a stuck query cannot occupy the single batcher indefinitely.
//
// The complete database response is validated before any capacity is applied.
// Every requested subject must have exactly one matching result.
//
// Each granted amount must be:
//
//   - non-negative;
//   - no larger than the requested amount;
//   - no larger than the available capacity reported by PostgreSQL.
//
// Unexpected, duplicate, invalid, or incomplete results fail the complete local
// batch. A failed batch applies no capacity, rejects every associated waiter,
// and activates global backoff.
//
// Targets and refill thresholds
//
// Each local target is capped by both the standard lease size and the configured
// burst capacity.
//
// A refill is normally prepared before the bucket becomes empty when:
//
//   - local capacity falls below its scaled threshold;
//   - an operation leaves no more capacity than the cost it just consumed;
//   - an operation finds insufficient local capacity.
//
// Targets and thresholds are deliberately simple. They are not adaptive to
// traffic rate or database latency.
//
// Limited overdraft
//
// Cost-1 operations may temporarily make an active bucket negative while its
// refill has no waiters.
//
// Overdraft is allowed only when:
//
//   - refills are open;
//   - no global backoff is active;
//   - the bucket is active;
//   - the configured per-node negative limit would not be exceeded.
//
// Ingestion retains its one-time initial cost-1 overdraft immediately before
// publishing the cold bucket's first refill. That operation remains authorized
// even if publication subsequently fails. The one-time allowance is not
// restored.
//
// Once the first waiter is admitted, no new overdraft is permitted until all
// waiters leave or the refill finishes.
//
// Any negative balance that already exists is subtracted from the admission
// budget. A grant naturally repays that balance before its remaining capacity
// is assigned to waiters.
//
// For one subject, each node may exceed global capacity by at most
// apiRateLimitOverdraftLimit cost-1 operations. The approximate distributed
// upper bound remains:
//
//   number of nodes * apiRateLimitOverdraftLimit
//
// Global refill backoff
//
// Lease-acquisition errors and invalid batch responses activate a short global
// backoff. The failed batch receives no capacity, and all of its waiters are
// rejected.
//
// Cancellation caused by rateLimiter.Close does not activate backoff.
//
// During backoff:
//
//   - positive local capacity remains usable;
//   - no new refill is published;
//   - no waiter is admitted;
//   - no overdraft is allowed.
//
// Refill generations already buffered in the queue are discarded without
// PostgreSQL access when the batcher reaches them. Their waiters are resolved
// as part of that discard.
//
// The first request arriving after the backoff deadline may publish a new
// refill. No retry timer or additional goroutine is required.
//
// Removal and shutdown
//
// Removing an organization or workspace disables its buckets, clears their
// local capacity, detaches the current refill generation, and rejects every
// waiter.
//
// Pointers to queued generations remain memory-safe. Later requests and late
// refill results are ignored because the bucket is disabled and the generation
// identity no longer matches.
//
// Closing the limiter:
//
//   - prevents new refills;
//   - prevents new waiters and overdraft;
//   - cancels in-progress database work;
//   - stops the batcher;
//   - directly unblocks pending waiters, including those belonging to entries
//     left in the undrained refill queue;
//   - abandons remaining local capacity without returning it to PostgreSQL.
//
// Important invariants
//
// Future changes should preserve these properties unless they deliberately
// redefine the limiter's guarantees:
//
//   1. Public consumption never accesses PostgreSQL directly.
//   2. Exactly one workspace, ingestion, or nonspecific budget applies to each
//      request.
//   3. Each subject has one local bucket per process.
//   4. The bucket mutex protects all mutations to bucket, refill-generation,
//      and waiter state. Final waiter results are safely published to readers
//      after the mutex is released.
//   5. No bucket mutex is held during refill-queue or database I/O.
//   6. At most one refill generation is publishing, queued, or in progress per
//      bucket.
//   7. A waiter belongs to one active refill generation and never moves to
//      another.
//   8. Pending waiter cost never exceeds requested units minus local overdraft
//      debt.
//   9. New overdraft is disabled while any waiter exists.
//  10. PostgreSQL subtracts capacity before returning a lease.
//  11. Granted capacity is added to the current local value.
//  12. Capacity assigned to a waiter is deducted atomically before the waiter
//      is notified.
//  13. Errors and invalid results never add capacity.
//  14. Every refill reaches a final state that resolves all waiters still
//      associated with it.
//  15. Cancellation and completion resolve every waiter exactly once.
//  16. Disabled buckets cannot consume capacity, request refills, or apply late
//      results.
//  17. Generation identity prevents stale results from affecting newer work.
//  18. Every waiter has a finite maximum wait, even when the caller provides no
//      deadline.
//  19. Shutdown unblocks waiters even when the refill queue is not drained.
//  20. Leases are not returned without persistent identity and fencing.
//
// Possible future work
//
// Production observations may justify adapting:
//
//   - lease size;
//   - refill thresholds;
//   - batching delay;
//   - waiter maximum duration;
//   - lease-acquisition deadline;
//   - global backoff duration.
//
// Fairer lease distribution, persistent lease identities, multiple batcher
// workers, or explicit coordination between nodes would materially increase
// concurrency complexity and should be introduced only in response to a
// demonstrated need.
