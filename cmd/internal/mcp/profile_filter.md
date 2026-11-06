# Querying profiles with Record and Count filters

Use `query-profiles` for questions about profiles and segments. It runs the same
`core.Filter` and `Workspace.QueryProfiles` path as `GET /v1/profiles`.
Read `profile-schema` and, for Count qualification, `event-schema`. Use the schema
property `path`, not its physical warehouse column name. Fetch only the profile
properties needed to answer the user.

Use `list-event-definitions` to discover events by ID, technical name, display
name or description. The optional `query` is a literal substring, not semantic
search; omit it to compare the full catalog. Then use `get-event-definition`
with an exact catalog ID to inspect the native schemas and property meanings.
Both tools require a workspace MCP key and rate capacity, but not MCP warehouse
settings. Their JSON is returned as text and structured content.

Catalog IDs select metadata, not filter sources. For example, retrieve
`commerce.order_completed`, read its `event` value `Order Completed`, then use
that exact value as Count `source.name`. To find profiles with at least two
orders in the last thirty elapsed days, use:

```json
{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}},"operator":"is greater than or equal to","values":[2]}]}
```

This counts recorded messages, not deduplicated business orders. Catalog entries
do not prove which events were received or that historical data follows the
current schema. A page/screen definition with an empty event name does not make
it a valid Count name; do not invent a track event for it. The legacy merge
definition is historical, not an enabled ingestion or query capability.

The native schemas provide descriptions, units and nested structure. Conditions
on traits/properties still use the raw JSON operators described below, not
native date, currency or other governed semantics inferred from the catalog.
Declared array elements do not enable traversal or same-element predicates.
Sums, business-order deduplication and other unsupported constructions must be
reported as unsupported, without SQL fallback.

## Connection and access

MCP warehouse settings must be configured to enable profile queries. This tool
executes through Krenalis's **main warehouse connection**, not the separate MCP
credentials. Those credentials and their table, column and other database access
restrictions apply to `query-data-warehouse` only. They do not restrict which
profile or event properties `query-profiles` can read or use in conditions.

On Snowflake, `query-profiles` uses the main virtual warehouse, even if a different
virtual warehouse is configured for MCP. Record queries remain available; Count
is currently unavailable on Snowflake. Do not work around that gate using SQL.

## Arguments and results

`filter` is **JSON text in a String**, exactly as in the HTTP API query parameter.
This transports numeric literals without rounding or exponent conversion by the
MCP SDK. Do not send an object or SQL. Omit `filter` to list all profiles. An empty
String, JSON `null`, unknown arguments and explicit null arguments are invalid.

Other arguments: optional `properties` (top-level property names; omission requests
all), `order` (a sortable profile property), `orderDesc` (default false), `first`
(default 0), `limit` (default 20, range 1–1000), and `at` (Count only).
Arguments are limited to 500 KiB. The full HTTP POST, including its JSON-RPC
envelope, is limited to 1 MiB; oversize is HTTP 413 before the tool runs.
The complete result JSON is limited to 128 KiB.
An oversized result is an error: reduce `limit` or request fewer `properties`.
Rows are never silently dropped to fit that budget.

The returned text is JSON with `profiles`, `schema` and exact matching `total`.
With Count, it also contains `evaluation: {at, match, noMatch, unknown}`. Totals
refer to the entire profile population before pagination, from the same
evaluation as the page. The page contains only MATCH profiles. UNKNOWN is not
NO_MATCH and is not an execution error. An empty page still returns totals.

`at` is a nonfuture RFC3339 instant, exactly representable at microsecond
precision. Omission fixes T once. Reuse the returned `evaluation.at` for subsequent
pages to anchor relative periods. It does not freeze the data, identity mappings
or profile values between requests. Count sorting uses NULLS LAST and KPID as a
deterministic tie-break; Record-only ordering retains the existing API behavior.

Invalid filters and unavailable capabilities return tool errors with actionable
messages/codes. Technical failures return an opaque operation error, never
invented totals, partial success, NO_MATCH or UNKNOWN. `ProfileResultTooLarge`
means reduce the page/projection; `ProfileProjectionTooWide` means request fewer
properties. `FilterCapabilityNotImplemented` and `UnifiedFilteringUnavailable`
must be reported to the user, not replaced with another query language.

The MCP adapter also uses `InvalidArguments` for invalid requests/filters and
`MCPNotConfigured` when MCP warehouse settings have not been configured. These
are MCP diagnostic prefixes, not additional core error codes.

## Filter syntax and supported subset

The root is `{"operator":"and"|"or","rules":[...]}`, with optional
`defaultCalendar` as described below. Rules are Record
conditions, Count conditions or nested AND/OR groups. At most four main group
levels including the root, and 100 descendants excluding the root, are allowed.

A Record condition is `{"property":"path","operator":"...","values":[...]}`.
It retains the existing schema-aware operators, payloads and types. Values are
JSON literals, not SQL and not an expression language. Numeric operands must be
written with their exact digits, never rounded through a floating-point value.

The Record operators are: `is`, `is not`, `is less than`, `is less than or equal to`,
`is greater than`, `is greater than or equal to`, `is between`, `is not between`,
`contains`, `does not contain`, `is one of`, `is not one of`, `starts with`,
`ends with`, `is before`, `is on or before`, `is after`, `is on or after`, `is true`,
`is false`, `is empty`, `is not empty`, `is null`, `is not null`, `exists`,
`does not exist`. The property's type determines which operators and operands
are admissible. Data-state operators have no comparison operands.

Count is `{"count":{"source":...,"period":...,"where":...,"minimum":...},
"operator":"...","values":[...]}`. It supports only:

- Source `{"kind":"recordedEvents","name":"EXACT EVENT NAME",
  "grain":"registered-message"}`. `name` matches the raw Event `event` value.
  Grain is mandatory: Count counts registered
  occurrence identities, not business orders or identical payloads.
- Comparisons `is`, `is not`, the four numeric ordered operators, and inclusive
  `is between`. One nonnegative integer JSON operand, or two for between. No
  Strings, decimals or exponent notation. `Count is 0` expresses absence;
  `Count is greater than or equal to 1` expresses existence. Execution currently
  supports signed 64-bit natural values; larger valid integers are unavailable
  capabilities, not a reason to round the literal.
- Optional positive integer `minimum` is a sample guard, separate from values.
  Omission means no guard; explicit null is invalid.
- Optional `where` is a Record filter on Event properties with at most two group
  levels and 100 descendants. Omission/null means no extra qualification. Count
  cannot be nested inside it. Main Record conditions refer to profile properties.
- Period `{"kind":"absoluteHalfOpen","start":"RFC3339","end":"RFC3339"}`
  with start < end <= T; or `{"kind":"elapsedLookback","duration":D}`; or
  `{"kind":"elapsedAgoBand","older":D1,"newer":D2}` with older > newer >= 0.
  A duration is `{"value":INTEGER,"unit":"UNIT"}`. Units: microsecond,
  millisecond, second, minute, hour, day (24 hours), week (7 × 24 hours).
  Periods are [start,end); events exactly at end/T are excluded. Elapsed units
  do not mean calendar days, weeks or months.

For the last complete calendar days, weeks or months, use
`{"kind":"lastCompleteCalendarPeriods","unit":"month","count":1}` as
`period`. Units are `day`, `week`, `month`; count is a positive JSON integer.
The current period is excluded. This counts occurrences across the complete
interval, not the number of active periods. Weeks are Monday through Sunday.

The main root must also contain, for example,
`"defaultCalendar":{"definition":"gregorian-monday/Europe/Rome","meaningRevision":"1"}`.
Use an explicit IANA timezone (or UTC), with exact spelling and case, after
`gregorian-monday/`; only revision 1 is resolved. Host-local names such as
`posixrules` are rejected. There is no implicit workspace or server-local calendar.
The reference can remain on a Profiles filter without Count; it is validated but
does not change Record results or add evaluation metadata. It is not allowed on
nested groups, occurrence `where` or in contexts restricted to Record filtering.
Use the timezone agreed with the user; ask if none has been established. Do not
infer it from the user's language, locale, server or event context. The calendar
does not reinterpret absolute instants, elapsed durations or raw JSON values.
It is optional for existing absolute/elapsed Count filters. Per-condition and
profile-timezone overrides are unavailable. Keep the same filter reference and
reuse `evaluation.at` when paging. This does not freeze data or the tzdb version.
Ambiguous/nonexistent midnight boundaries and out-of-range calendar windows
return capability errors, never UNKNOWN or approximate elapsed windows.
Havana's midnight on 2026-11-01 is ambiguous: any window starting or ending there
is unavailable, including October and November 2026 as single complete months.

No relevant recorded events means Count zero. The history is the events recorded
and queryable in Krenalis, not a certification of events in external systems.
Unknown decisive local values can yield UNKNOWN; independent qualification
uncertainty is supported. Shared governed facts, ambiguous identity/timestamps,
business deduplication, other calendar/history/personal-anchor constructors, sums,
shares and E/K/P/T are not available through this tool.

Raw JSON and governed business properties differ. With `properties.currency is
"EUR"`, a present JSON null is FALSE and a missing path is UNKNOWN; Number 100
is different from String "100". Negative comparisons do not make missing values
match. NativeCurrency bindings are internal only: do not infer or invent one.

Event names and arbitrary JSON contents are workspace-specific. Ask the user or
use `query-data-warehouse` to discover them before constructing a filter. SQL may
also serve other exploratory analysis. It must not be used to emulate a segment
or complete a Unified feature that `query-profiles` cannot execute.

## Examples

The following are filter documents; pass each as the text of `filter`. Names and
paths are examples: verify them against the actual workspace.

Profile property:

```json
{"operator":"and","rules":[{"property":"country","operator":"is","values":["IT"]}]}
```

No recorded Purchase in the last 30 elapsed days:

```json
{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents","name":"Purchase","grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}},"operator":"is","values":[0]}]}
```

Between two and five Purchases in an absolute period:

```json
{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents","name":"Purchase","grain":"registered-message"},"period":{"kind":"absoluteHalfOpen","start":"2026-01-01T00:00:00Z","end":"2026-02-01T00:00:00Z"}},"operator":"is between","values":[2,5]}]}
```

Italian profiles with no Purchase whose **raw JSON** currency equals String EUR:

```json
{"operator":"and","rules":[{"property":"country","operator":"is","values":["IT"]},{"count":{"source":{"kind":"recordedEvents","name":"Purchase","grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}},"where":{"operator":"and","rules":[{"property":"properties.currency","operator":"is","values":["EUR"]}]}},"operator":"is","values":[0]}]}
```
