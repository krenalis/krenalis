# Iteration and batching (Records and Events)

## Core rule: consuming is processing

Krenalis considers a record/event processed as soon as it is **read** from the sequence, unless you `Postpone()` or `Discard(err)` it.

## You must choose ONE iteration method per call

Records:

- choose exactly one of: `records.First()`, `records.All()`, `records.Same()`
- do not call another after consuming (Krenalis will panic)

Events:

- choose exactly one of: `events.First()`, `events.All()`, `events.SameUser()`
- do not call another after consuming (Krenalis will panic)

## Peek

`Peek()` reads the next element without consuming it.

Important:

- for `records`, `Peek()` can only be called during an active iteration created by `records.All()` or `records.Same()`
- for `events`, `Peek()` can be called before consuming the sequence to choose the request shape, and can also be called during an active iteration
- after `First()` or after the sequence has otherwise been consumed, `Peek()` is no longer valid

Use it to decide:

- create vs update for users (`record.IsCreate()` implies creation; `record.IsUpdate()` implies update of that existing application user)
- which endpoint/method/batch strategy to use for events

## Discard

Use `Discard(err)` **only inside a for-range iteration** to permanently drop the current item:

- validation failures
- non-retryable constraints

Never pass nil error. Do not call Discard after Postpone for the same element.

## Postpone

Use `Postpone()` **only inside iteration** to "unconsume" the current element so Krenalis will retry it in a later call:

- body size exceeded
- batch size exceeded
- API request limit reached

For Events: never Postpone the first event (Krenalis will panic).

Only call `Postpone()` if you have **not** modified the current record's `Attributes` (for records) or the current event's `Type.Values` (for events). Postponing modified items is not allowed.

## Canonical batching pattern

```go
n := 0
for rec := range records.All() {
    checkpoint := bb.Len()
    // write rec into request body...
    if bb.Len() > limit {
        bb.Truncate(checkpoint)
        records.Postpone()
        break
    }
    if err := bb.Flush(); err != nil { return err }
    n++
    if n == maxPerRequest { break }
}
```

The same pattern applies to events (`events.All()` / `events.SameUser()`).
