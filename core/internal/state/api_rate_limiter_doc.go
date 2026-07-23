// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

// API rate limiter design
//
// Purpose and quota model
//
// The rate limiter gives each workspace independent budgets for API operations
// and ingestion, and each organization a nonspecific API budget.
//
// API operations fall into three categories:
//
//   - workspace operations, which act on one specific workspace.
//   - ingestion operations, which write events for one specific workspace.
//   - nonspecific operations, which are not in a specific API category;
//
// A workspace operation consumes capacity only from that workspace's own API
// budget. An ingestion operation consumes capacity only from that workspace's
// ingestion budget. A nonspecific operation consumes capacity from the
// organization's nonspecific API budget.
//
// Exactly one budget applies to each request. A workspace request does not also
// consume the organization's nonspecific budget, ingestion does not consume
// either of the other budgets, and a nonspecific request does not consume
// capacity assigned to any workspace.
//
// Each workspace has an independent quota. Traffic directed to one workspace
// therefore does not affect the quota of another workspace. Workspace API and
// ingestion quotas are also separate from the organization's nonspecific
// quota and from one another.
//
// Rate-limit budgets belong to organizations and workspaces, not to individual
// API keys.
//
// Architecture overview
//
// The rate limiter coordinates API capacity across multiple application nodes
// while keeping request handling local to each process.
//
// PostgreSQL is the only shared coordination mechanism between application
// nodes. API requests never access PostgreSQL directly. Instead, each node
// acquires chunks of capacity, called leases, from authoritative PostgreSQL
// buckets and consumes those leases from memory.
//
// The design favors a fast, non-blocking request path over exact coordination
// between nodes on every request.
//
// Public API and bucket ownership
//
// The public core.Organization and core.Workspace types expose normal API
// rate limiting through ConsumeRateLimitCapacity. The collector consumes
// ingestion capacity directly from the corresponding Workspace instance in
// this package after authenticating an ingestion request.
//
// Each API operation has a cost that determines how much rate-limit capacity it
// consumes. Normal API operations support costs from 1 through 100. Ingestion
// operations use their event count as their cost. Invalid costs return
// ErrInvalidAPICost. Requests without enough available capacity return
// ErrAPICapacityExceeded.
//
// Each Organization instance stores one local rateLimitBucket. Each Workspace
// instance stores separate API and ingestion buckets. The shared rateLimiter,
// in turn, owns the refill queue, batcher, PostgreSQL lease acquisition, and
// shutdown lifecycle.
//
// Local consumption
//
// ConsumeRateLimitCapacity validates the request cost, and
// ConsumeIngestionRateLimitCapacity consumes the request's event count, from
// the relevant subject's local bucket. A subject is a workspace, ingestion for
// a workspace, or nonspecific.
//
// The request path:
//
//   1. locks only the relevant bucket;
//   2. consumes locally available capacity when possible;
//   3. schedules an asynchronous refill when capacity is low or insufficient;
//   4. returns without waiting for PostgreSQL.
//
// A newly created Organization or Workspace instance starts with an empty local
// bucket, even if capacity is available in its PostgreSQL bucket. Its first
// request may therefore be rejected while the initial refill is scheduled.
// This is intentional: request handling remains local and non-blocking.
// A newly created ingestion bucket is the exception: it allows the normal
// small overdraft while its initial refill is queued, so a client can send its
// first event without retrying solely because the process has no local lease.
//
// A bucket can have at most one refill queued or in progress. refillQueued
// remains set while that refill is waiting in the queue or being processed.
//
// The flag is updated while holding the bucket mutex. It is cleared when the
// refill completes, when enqueueing fails, or when the collected batch fails.
//
// One exception occurs during shutdown. When State.Close is called, the batch
// currently being collected or processed is completed without applying any
// capacity that was not confirmed by PostgreSQL. The buffered queue is not
// drained, so buckets that remain in the queue may keep refillQueued set.
//
// This is safe because shutdown prevents further consumption and database I/O.
//
// No bucket mutex is held while publishing to the queue or performing database
// I/O.
//
// Capacity and leases
//
// PostgreSQL stores the authoritative capacity for every organization and
// workspace. Rate-limit configuration is read from the organization and
// workspace domain tables when a lease is acquired.
//
// The batcher sends organization, workspace, and ingestion requests in the
// same batch to the PostgreSQL function acquire_api_rate_limit_leases.
// PostgreSQL processes each subject independently and may grant the full
// requested amount, a partial amount, or zero units.
//
// During acquisition, PostgreSQL locks each relevant bucket, calculates newly
// available capacity, subtracts the granted lease, and only then returns the
// granted units to the process.
//
// Each batch entry requests no more than its bucket's lease size. Ingestion
// buckets use a larger lease than normal API buckets to accommodate event
// batches. The lease-size bound remains true even when the local bucket is
// negative and the amount needed to reach its target is larger than one
// standard lease.
//
// The local bucket adds the granted amount to its current value rather than
// replacing it. Requests may continue consuming local capacity while the
// database query is running.
//
// This gives the design its main safety property:
//
//   a process crash may lose unused leased capacity, but it cannot create
//   additional capacity.
//
// Unused leases are not returned during shutdown. Returning them safely would
// require persistent lease identities and fencing. A best-effort return could
// race with local consumption and credit the same capacity twice.
//
// Batching
//
// A single batcher goroutine collects pending refills for a short interval, up
// to the configured batch size, and sends them to PostgreSQL in one call.
//
// Using one worker keeps the concurrency model simple and prevents concurrent
// lease acquisitions from the same process. PostgreSQL also processes bucket
// rows in deterministic subject order to reduce the risk of deadlocks between
// different nodes.
//
// The complete database response is validated before any capacity is applied
// locally. Every requested subject must have exactly one matching result, and
// the granted amount must be non-negative, no larger than requested, and no
// larger than the capacity reported by PostgreSQL.
//
// Unexpected, duplicate, invalid, or incomplete results fail the entire local
// batch, add no capacity, and activate the same global backoff used for
// database errors.
//
// Targets and refill thresholds
//
// Each local bucket has a target capped by both the standard lease size and the
// configured burst capacity.
//
// A refill is normally scheduled before the bucket becomes empty, when local
// capacity falls below the refill threshold or an operation leaves no more
// capacity than the cost it just consumed. It is also scheduled when a request
// finds insufficient capacity.
//
// The threshold scales with the local target, so buckets with a small target do
// not request a new lease after almost every small operation. Comparing the
// remaining capacity with the operation cost makes the same policy responsive
// to larger and variable costs, such as ingestion batch sizes.
//
// The target and threshold are intentionally simple and are not currently
// adaptive to traffic rate or database latency.
//
// Limited overdraft
//
// Operations with cost 1 may temporarily make the local bucket negative while
// a refill is already pending. This temporary negative balance is called
// overdraft.
//
// These operations are served entirely from memory, allowing a small amount of
// traffic to continue while the refill is pending. Overdraft is not available
// for larger costs.
//
// It is allowed only when:
//
//   - the cost is exactly 1;
//   - refillQueued is already true;
//   - refills are currently allowed: the limiter is open and no global refill
//     backoff is active;
//   - the bucket is active;
//   - the configured negative limit would not be exceeded.
//
// The request that schedules the first refill cannot use overdraft.
// If enqueueing fails, refillQueued is cleared, so overdraft cannot be used
// unless a refill was actually queued.
//
// A granted lease is added to the current local value and naturally repays any
// overdraft. A partial or zero grant may leave the bucket negative.
//
// This makes the limiter intentionally approximate. For one subject, each node
// may exceed global capacity by at most apiRateLimitOverdraftLimit cost-1
// operations. The approximate distributed upper bound is therefore:
//
//   number of nodes * apiRateLimitOverdraftLimit
//
// Global refill backoff
//
// Without a backoff, a PostgreSQL outage could produce a tight retry loop:
//
//   refill fails -> refillQueued is cleared -> requests enqueue again ->
//   another refill fails.
//
// After an acquisition error, or after an invalid batch response, the shared
// rateLimiter activates a short global backoff. The backoff is global because
// database access and the batcher are shared, and a PostgreSQL failure will
// normally affect every subject.
//
// Cancellation caused by rateLimiter.Close does not activate the backoff.
//
// During the backoff:
//
//   - already leased positive capacity remains usable;
//   - no new refill is queued;
//   - refillQueued is not newly set;
//   - no new overdraft is allowed;
//   - requests without enough local capacity are rejected.
//
// No retry timer or additional goroutine is needed. The first request arriving
// after the deadline can schedule a new refill.
//
// Buckets that were already queued when the backoff began are completed without
// accessing PostgreSQL. Clearing their pending marker prevents them from using
// overdraft when no refill will actually be attempted.
//
// Removal and shutdown
//
// When an organization or workspace is removed from State, all of its buckets
// are disabled. A disabled bucket rejects consumption and does not request new
// leases.
//
// A bucket pointer already present in the queue remains memory-safe because Go
// keeps the referenced object alive. If it is processed later, its pending
// refill is discarded and no granted capacity is applied to it.
//
// Closing the rateLimiter:
//
//   - prevents new refills;
//   - cancels the internal database context;
//   - stops the batcher;
//   - safely completes or cancels the batch currently being handled;
//   - does not necessarily drain or clear entries still buffered in the queue;
//   - abandons any remaining process-local capacity without returning it to
//     PostgreSQL.
//
// Important invariants
//
// Future changes should preserve these properties unless they deliberately
// redefine the limiter's guarantees:
//
//   1. Public consumption never waits for or accesses PostgreSQL.
//   2. Exactly one workspace, ingestion, or nonspecific budget applies to each
//      request.
//   3. Each subject has one local bucket per process.
//   4. The bucket mutex protects all mutable bucket state.
//   5. No bucket mutex is held during queue or database I/O.
//   6. At most one refill is queued or in progress per bucket.
//   7. PostgreSQL subtracts capacity before returning a lease.
//   8. Granted capacity is added to the current local value.
//   9. Errors and invalid results never add capacity.
//  10. Local overdraft never exceeds the configured per-node limit.
//  11. Overdraft is limited to cost-1 operations with a pending refill and no
//      active global refill backoff.
//  12. Shutdown prevents new refills and database I/O, but does not guarantee
//      that refillQueued is cleared for entries left buffered in the queue.
//  13. Disabled buckets cannot schedule new refills.
//  14. Leases are not returned without persistent identity and fencing.
//  15. Adding nodes cannot create unbounded capacity; it only increases the
//      documented overdraft bound by the per-node limit.
//
// Possible future work
//
// Later revisions may tune lease size, refill threshold, batch delay, or
// backoff based on production observations.
//
// Selective prefetch or prewarming for frequently used subjects could reduce
// the initial retry required while an empty local bucket acquires its first
// lease. It would need to balance that improvement against database work and
// capacity held unused by an idle node.
//
// A request whose cost exceeds the configured burst capacity cannot be served
// atomically. Supporting such requests would require splitting them into
// independently limited operations or redefining the burst guarantee.
//
// Adaptive lease sizing, fairer lease allocation across nodes, and persistent
// lease identities could improve capacity distribution and reclaim unused
// capacity when necessary.
//
// More detailed queued/in-flight states, multiple batch workers, lease return,
// or explicit node coordination would materially increase the complexity of the
// concurrency model and should be introduced only when there is a demonstrated
// need.
