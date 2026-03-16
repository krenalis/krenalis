# Users: RecordFetcher / RecordUpserter

## RecordSchema

Signature:

```go
RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error)
```

Notes:

- `target` is currently `connectors.TargetUser`.
- `role` can be `connectors.Source` or `connectors.Destination`.
- It is valid to return different schemas per role.
- It is also valid to return the same static schema for both roles when source and destination differ only by role-dependent flags (`ReadOptional`, `CreateRequired`, `UpdateRequired`).
- `ReadOptional` describes the read path, not the fact that a property may be used for destination matching.
- If the API distinguishes readable fields from writable ones, the destination schema must exclude the non-writable fields even if they appear in read responses or metadata listings.
- If you do keep separate source and destination schemas in code, keep them role-coherent for readability: avoid `ReadOptional: true` in a destination-only schema, and avoid `CreateRequired` / `UpdateRequired` in a source-only schema unless there is a specific, documented reason.
- Concretely: if you build a schema inside a `role == connectors.Source` branch, do not use destination-only flags unless there is a specific, documented reason. If you build a schema inside a `role == connectors.Destination` branch, do not use source/read-only flags unless there is a specific, documented reason.
- Some connectors build schemas dynamically (e.g. HubSpot/Mailchimp fetch fields from API).

### Static vs dynamic schema (standard rule)

- Default to a **static schema** that covers the stable core fields needed for the implemented capability (minimal-but-useful).
- Implement a **dynamic schema** only when:
  - the application's user schema is genuinely tenant-specific or user-configurable (custom fields), AND
  - the API provides a documented metadata endpoint to list fields, AND
  - using a static schema would make the connector misleading or severely incomplete.
- If you implement a dynamic schema:
  - cache results in-memory (per connector instance) with a reasonable TTL to avoid repeated metadata calls
  - for source/read schemas, treat missing/unknown fields conservatively as `ReadOptional` unless the API guarantees presence
  - when deciding whether a dynamic field belongs in the destination schema, explicitly check whether the API marks it as writable or read-only
  - common signals include `readOnly`, `readonly`, `writable`, `mutable`, `editable`, `updatable`, `createOnly`, `outputOnly`, `calculated`, `calculatedValue`, `derived`, `computed`, or documentation notes like "response only", "cannot be modified", or "managed by the system"
  - if the API exposes one broad metadata listing but the write endpoints accept only a subset of fields, treat the write endpoint contract as authoritative for the destination schema
  - keep the dynamic surface bounded: do not expose every obscure field if the API has hundreds; prioritize the common + user-defined fields.

## Records (reading users)

Signature:

```go
Records(ctx context.Context, target connectors.Targets, updatedAt time.Time, cursor string, schema types.Type) ([]connectors.Record, string, error)
```

Contract:

- If `updatedAt` is non-zero, return only records updated at/after updatedAt (microsecond precision).
- Pagination:
  - `cursor` is empty on first call.
  - return `nextCursor` string for the next page.
  - when there are no more records, return `io.EOF` (possibly with final batch).
- It is allowed to return duplicate IDs; Meergo deduplicates upstream.
- Each returned record must have:
  - `ID` non-empty UTF-8, and it MUST be the application's User ID (Meergo terminology): the canonical unique user identifier (typically the primary key assigned by the application, generated server-side when the user/contact is created)
    - Do NOT set `Record.ID` to a natural key like email/phone/ext_id, even if the vendor API accepts them as alternate identifiers when addressing a user.
    - Put those alternate identifiers in `Attributes` and use Meergo pipeline matching to map the matching property value to the application's User ID for updates. (In Meergo core this app User ID is stored as `DestinationProfile.ExternalID`.)
  - `Attributes` map containing the properties defined in `schema` (unless `ReadOptional`), with values compatible with the corresponding property types
    - If a non-optional requested property is missing from the upstream API response, do not silently omit it: either mark the property as `ReadOptional` in your `RecordSchema` (preferred if the API may omit it), or set `record.Err` for that record.
    - Extra attributes not explicitly requested are allowed; Meergo will ignore unknown keys. For efficiency, avoid fetching/processing extra fields if the API lets you request only needed fields; balance readability vs performance and prefer returning only the requested subset unless the cost is negligible or the API cannot filter fields.
    - Do not use type-incompatible placeholders (e.g. `""` for `DateTime`); if a value is missing, omit the key (preferred) or set it to `nil` only if the property is `Nullable`.
  - `UpdatedAt` MUST be set to the application's actual "last updated" timestamp (non-zero; year >= 1900). Any time zone is allowed; Meergo normalizes to UTC and truncates to microseconds.
  - `Err` optionally set to mark a per-record read error (if Err != nil, only ID is significant)
    - Keep `record.Err` messages fixed: do not include input-specific values or upstream field contents in the message text.

Use `schema` to request/return only needed fields (important for performance).

### Destination matching depends on Records() (important)

For destination application exports, Meergo can synchronize destination users by calling `Records(...)` with a schema pruned to the pipeline's "out matching property". This is a read/input pipeline schema concern. If you reuse one shared schema for both roles, Meergo will apply the correct role semantics to the role-dependent flags. If you keep a separate destination schema, omit `ReadOptional: true` unless there is a specific, documented reason to keep it. The fact that destination matching uses `Records()` does not change this readability rule. For this to work reliably:

- `Record.ID` must be the application's User ID (unique user identifier).
- `Record.Attributes` must include the requested out matching property path (it may be nil, but it must be present when requested unless it is read-optional).

Do not substitute the matching property value into `Record.ID`.

### Time fields (standard rule)

- `Record.UpdatedAt`:
  - MUST be set to the application's actual "last updated" timestamp for every returned record.
  - Any time zone is allowed; Meergo converts to UTC and truncates to microseconds. You do not need to call `.UTC()` yourself (unless you specifically need a UTC timestamp for some API formatting).
  - If missing or unparseable, set `record.Err` (or fail the page) and do NOT use `time.Time{}` as a placeholder.
- Time-typed properties inside `Record.Attributes`:
  - if missing: omit the key when the schema is `ReadOptional` (preferred), otherwise set `record.Err`
  - if present but invalid/unparseable: treat like missing (omit if optional, else `record.Err`)
  - prefer returning the upstream string/bytes and using `ApplicationSpec.TimeLayouts` when the API uses a non-ISO format, instead of parsing these attributes manually in the connector

### Pagination and page size (performance)

When the application's list endpoint supports a `limit` / `pageSize` parameter with a documented maximum, default to the **maximum allowed page size** to minimize round trips and speed up imports.

- If the docs say `limit` is allowed in `[min,max]`, choose `max` unless there is a strong reason not to (e.g. the API becomes unreliable at max, payloads become too large, or rate limiting penalizes large pages).
- Encode the chosen page size as a constant (e.g. `recordsPageLimit`) and document in code/comments which API limit it matches.
- In the "Connector Design" summary, explicitly state the chosen page size and cite the vendor's documented max/default.

When using `updatedAt` in API queries:

- `updatedAt` is already in `time.UTC` (location is UTC) and truncated to microseconds by Meergo before being passed to your connector, so you do not need to call `updatedAt.UTC()` before formatting it.
- If the API requires a string, you can format it directly (e.g. `updatedAt.Format(time.RFC3339Nano)` or an app-specific layout).

Meergo truncates `Record.UpdatedAt` to microseconds internally; connectors may also truncate to microseconds for clarity and to avoid surprising diffs.

## Upsert (writing users)

Signature:

```go
Upsert(ctx context.Context, target connectors.Targets, records connectors.Records, schema types.Type) error
```

The **records sequence is non-empty**, but you do NOT need to process all items in a single call. You must:

- consume at least one record each call (use `First()` or iterate)
- batch intelligently based on API constraints
- handle invalid inputs with `records.Discard(err)` or return `connectors.RecordsError` for per-record failures when the API provides them
- use `records.Postpone()` to handle body size limits / request limits, so the record will be retried in a later call

The attributes of each record conform to the schema passed as an argument to `Upsert`. The provided schema is a subset of the connector's destination schema, so it may not include all properties. If the destination schema is dynamic, the schema passed to `Upsert` is a subset of the most recent destination schema retrieved by Meergo.

### Record.ID semantics (create vs update)

- A record to be created has an **empty** `Record.ID` (this is a Meergo contract). In code, prefer `record.IsCreate()`.
- A record to be updated has a **non-empty** `Record.ID` containing the application's User ID (unique user identifier). In code, prefer `record.IsUpdate()`.

### Avoid validating Record.ID shape in Upsert (important)

When `Upsert` receives a record with `record.IsUpdate() == true`, `record.ID` is the application's User ID that Meergo already knows (it comes from the destination application's previously read users, i.e. from your connector's `Records(...)` results, and is then reused for exports).

Therefore:

- Do not validate its format (for example, do not require it to be an integer) and do not reject records based on ID shape.
- If your API client strongly prefers a typed ID (for example `int64`), you may parse `record.ID` for convenience; if parsing fails, treat it as an internal connector inconsistency/bug (not a user-data validation error).
- Use it to address the user in the API (typically `url.PathEscape(record.ID)`), and rely on the API to reject unknown/non-existent IDs if needed.

Even if a vendor API supports updating a user by email/ext_id/phone (or an `identifierType` parameter), keep `Record.ID` reserved for the application's unique ID. If the vendor requires a non-ID identifier for update, treat it as an attribute/matching property and resolve the unique ID first (typically via the destination users sync + matching).

### Request body building: stream JSON (performance)

When building the HTTP request body for `Upsert`, avoid allocating intermediate "payload maps" (for example `map[string]any{"id": ..., "attributes": ...}`) and then marshaling them.

Prefer streaming JSON into `BodyBuffer`:

```go
bb := c.env.HTTPClient.GetBodyBuffer(connectors.Gzip) // or NoEncoding
defer bb.Close()

bb.WriteByte('{')
// If the API needs the user ID in the body, you can encode it here; often the ID is only used in the URL path.
if record.IsUpdate() {
	_ = bb.EncodeKeyValue("id", record.ID)
}
_ = bb.EncodeKeyValue("email", record.Attributes["email"]) // example field
// ...encode the rest of the fields directly...
bb.WriteByte('}')

req, err := bb.NewRequest(ctx, http.MethodPost, url)
```

If you intentionally allocate an intermediate structure for the request body, add a short comment explaining why streaming is not feasible for that API.

### Batching default (when the API supports both single and batch)

- Default to **batch** requests to reduce round trips **when** the API documents a true batch/multi-item endpoint **and** you can implement it correctly (limits + error mapping). In practice, prefer **batch** as long as the API documents:
  - a max batch size and/or a body size limit you can enforce, AND
  - how validation errors are returned (ideally per-item, otherwise you may be forced to mark the whole batch as failed).
- Prefer **single-item** requests when:
  - the API's batch endpoint does not provide actionable per-item errors and a single invalid record would fail the whole batch, OR
  - the batch semantics differ (async jobs) and complicate correct error reporting.
- If the API requires homogeneous batches (all create or all update), use `records.Peek()` + `records.Same()` to batch the right operation without consuming incompatible records.
