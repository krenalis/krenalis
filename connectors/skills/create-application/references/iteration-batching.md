# Iteration and batching

Treat iterator consumption and request rollback as one transaction. A batch is correct only when every yielded item is either included, discarded with an error, or postponed unchanged.

## Follow iterator state

Call exactly one of `All`/`Same`/`SameUser`/`First` on a given `Records` or `Events` value. A yielded item is consumed when produced. Breaking after an accepted item leaves later, unyielded items for the next call.

Use `Peek` to inspect the next item without consuming or modifying it. The tested runtime permits an initial `Peek` for records as well as events, although the public `Records.Peek` comment is narrower.

Call `Postpone` only during an iterator callback and only before modifying the current record attributes or event type values. It keeps that item unread for a later framework call. Production event iteration rejects postponing the first event; production record iteration currently permits the first record despite a contradictory comment. Ensure every successful call makes progress.

Use `Discard(err)` for a deterministic item-local failure discovered before delivery. Do not both discard and return the same item error.

## Use `BodyBuffer` safely

Acquire request bodies from `ApplicationEnv.HTTPClient.GetBodyBuffer` and always `defer bb.Close()`. `NewRequest` finalizes the buffer, sets replay support through `GetBody`, and makes all operations except `Close` invalid.

Understand these exact rollback semantics:

- `Len` counts flushed plus unflushed plain bytes; with `Gzip`, it is not the compressed wire length.
- `Flush` commits completed plain content. With `Gzip`, it compresses that content immediately.
- `Truncate(n)` keeps the first `n` **unflushed** bytes and discards the rest. Its argument is not a total-body length.

After flushing an accepted prefix and leaving no unflushed bytes, write one candidate chunk. On overflow, `Truncate(0)` removes that candidate while preserving the flushed prefix. Never pass an earlier total `Len()` checkpoint to `Truncate` after `Flush`; that can panic or retain the wrong bytes. If no content has ever been flushed, an absolute checkpoint is also the unflushed length and may be used safely.

Reserve or measure closing delimiters and fixed envelope overhead before accepting the last item. Maintain commas and framing as part of each rollback-safe candidate chunk. On encoding failure, restore the same unflushed checkpoint before deciding whether to discard or fail the request.

## Apply limits without losing items

For an item-count limit, stop after the last accepted item without yielding the next one. For a byte limit that can only be known after encoding:

1. preflight the first item with `Peek` when necessary;
2. encode the current item into the unflushed candidate region;
3. if it fits, flush/commit it and continue;
4. if it does not fit, roll it back and call `Postpone` while it is still unchanged;
5. if a first event cannot fit alone, discard it with a safe validation error or fail the call according to provider semantics—never postpone it.

If the provider's limit applies to compressed bytes, `BodyBuffer.Len` is insufficient for `Gzip`; use a format/encoding strategy that can enforce the documented wire limit. Do not guess whether a published limit is compressed or uncompressed.

Do not partially send a logical item. Test exact-limit, one-byte-over, envelope-only, oversized-first-item, encode-error, mixed-operation, and repeated-call cases.
