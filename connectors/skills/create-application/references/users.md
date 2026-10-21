# User import and upsert

Implement `RecordFetcher` for source users and `RecordUpserter` for destination users. Treat a provider that cannot satisfy the capability matrix as a framework mismatch, not an invitation to add a stub.

## Import users with `Records`

Honor the method contract:

- make `RecordSchema` handle `TargetUser` with both valid role values, and make `Records` accept `TargetUser`;
- treat `cursor` as an opaque continuation value and return the next value unchanged in meaning;
- when `updatedAt` is non-zero, return records created or modified at or after it, with microsecond precision;
- return attributes for all requested non-`ReadOptional` properties; extra attributes are allowed;
- return each provider record's stable, non-empty ID and real update time;
- expect the runtime to normalize update times to UTC microseconds and reject values before 1900, more than five minutes in the future, or earlier than a non-zero requested `updatedAt`;
- return the final non-empty page together with `io.EOF`, or `io.EOF` alone when no records remain;
- never return an empty page with a nil error: the runtime rejects it.

`Record.ID` and `Record.UpdatedAt` are framework metadata. Do not automatically duplicate them in `Record.Attributes` or the user schema. Expose a provider ID or update-time property as a mappable attribute only when the product use case explicitly needs it and the source/destination semantics remain accurate.

Choose page size from provider limits, response cost, and test evidence. Maximum page size is not a framework requirement.

If the provider lacks a trustworthy incremental filter or update timestamp, surface the limitation. Do not fabricate `UpdatedAt`, silently change inclusive semantics, or claim incremental sync while scanning data without documenting the cost.

The public record contract permits `Record.Err` when a stable ID is known, but the current application-record runtime does not propagate a connector-provided value. Until that mismatch is fixed, fail the page or request the framework change required by the desired item-error semantics; do not assume `Record.Err` will reach the caller.

## Consume `Records` correctly in `Upsert`

Choose one iterator method:

- `First` for a single-record endpoint;
- `Same` when create and update require different endpoints or payloads;
- `All` when one endpoint accepts both operations.

A yielded record is consumed unless it is postponed. `Discard(err)` records a local per-record failure. `Postpone()` keeps the current record for a later call and is valid only during `All` or `Same`, before its attributes are modified. The tested runtime permits postponing the first record, despite a contradictory public comment; avoid an endless no-progress loop.

Return `connectors.RecordsError` with zero-based indices for consumed records whose provider result failed. Indices absent from the map are successes. A different non-nil error applies to all records consumed by that call, so reserve it for request-level failure.

Map provider results deterministically. Preserve request order with indices or a documented correlation key; never assume an asynchronous acknowledgement means every record succeeded. For partial validation failures known before sending, call `Discard` and continue only when doing so preserves provider batching rules.

Do not mutate `Record.Attributes` before a possible `Postpone`. If payload construction requires mutation, clone the relevant map or decide consumption first.

## Test source and destination semantics

Cover first, middle, final, and empty pagination; opaque cursors; inclusive `updatedAt`; timestamp precision; missing IDs; `io.EOF`; provider item errors; selected-schema projection; create/update separation; partial `RecordsError`; request-level errors; batch item and byte boundaries; postpone/discard behavior; and no-progress prevention.
