# Querying profiles with Record, Count and Sum filters

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
Business-order deduplication and other unsupported constructions must be
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
  Omission means no guard; explicit null is invalid. Evaluate the guard and
  comparison separately, then combine their results with Kleene AND. For
  example, count in [1,3], minimum 3 and `is not 3` gives UNKNOWN, even though
  no count satisfies both requirements jointly.
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

For all retained, queryable recorded history before T, use:

```json
{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}}
```

This is a `period`, with no start, end, duration, calendar unit or count fields.
The history reference is explicit; no implicit default or alias is resolved.
Only this definition and immutable meaning revision 1 are currently available.
It has no lower date and excludes events exactly at T. It needs no calendar.
"Never purchased" means Count is 0 in this recorded history, not a statement
about all real-world purchases. Retention and deletion can change this result;
the filter neither reconstructs deleted events nor certifies external coverage.
For "purchased previously but not in the last 30 days", AND an existence Count
over this history with an absence Count over an elapsedLookback of 30 days.

No relevant recorded events means Count zero. The history is the events recorded
and queryable in Krenalis, not a certification of events in external systems.
Unknown decisive local values can yield UNKNOWN; independent qualification
uncertainty is supported. Shared governed facts, ambiguous identity/timestamps,
business deduplication, AvailableHistory and other calendar/personal-anchor constructors,
other reductions, shares and E/K/P/T are not available through this tool.

## Monetary Sum

Sum adds the totals reported by recorded `Order Completed` messages in one
explicit native currency. It does not calculate total customer spend across
currencies, convert money, certify payment/settlement, or deduplicate business
orders. Other known currencies are excluded. Distinct message identities
contribute separately even when they refer to the same order.

```json
{"operator":"and","rules":[{"sum":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}},"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is greater than or equal to","values":[30]}]}
```

This is the only currently resolved monetary field reference. Do not invent
other references by concatenating catalog IDs and property names. It fixes
`properties.total`, exact Decimal(18,4) reading and the associated
`properties.currency` field. The source must be `recordedEvents`, exact name
`Order Completed`, grain `registered-message`; incompatible sources are invalid.
All supported Count periods work, including complete calendar periods with the
same root calendar requirement and fullRelevantHistory with its explicit reference.

`nativeCurrency` is mandatory: three uppercase ASCII letters. Reading is the
same as governed Count currency: missing, JSON null and non-String are unknown;
any known String is compared verbatim. `"eur"` is different from `"EUR"`.
There is no implicit USD default. The existing raw JSON Record currency tests
retain their distinct semantics.

Use the four ordered operators (`is less than`, `is less than or equal to`,
`is greater than`, `is greater than or equal to`) with one signed JSON Number.
Thresholds are exact, up to precision 76 and scale 37, independently of the
field's four decimal places. They never pass through a float. Input literal
spelling is bounded to 4096 bytes. Equality, inequality and intervals are valid
language capabilities not yet implemented for Sum.

Optional positive integer `minimum` counts the events qualified by the source,
period, `where` and native currency, including events with an unknown amount.
Certainly excluded events do not count. Omission means no extra sample guard;
null and zero are invalid. Values beyond signed 64-bit range are unsupported.
Evaluate the sample guard and sum comparison separately, then combine their
results with Kleene AND, as for Count. Do not condition the sum bounds on the
minimum. For example, one certain EUR order of 50 and two orders of 60 with
unknown currency have `certain:1`, `uncertain:2` and bounds [50,170]. At
`minimum:3` and `is less than or equal to 100`, the result is UNKNOWN: each
requirement is individually uncertain, even though no joint choice satisfies
both. A certainly unreachable minimum gives NO_MATCH even with unknown amounts.

Optional `where` qualifies the individual events before adding their amounts.
It uses the existing Record operators, at most two group levels and 100
descendant nodes, as in Count. For example, merge these fields into `sum` to
restrict the total to orders with coupon PROMO:

```json
{"where":{"operator":"and","rules":[{"property":"properties.coupon","operator":"is","values":["PROMO"]}]}}
```

This first implementation temporarily excludes paths that overlap the monetary
reading: `properties` itself, `properties.total`, `properties.currency`, and any
descendants of the latter two. The restriction applies to every operator and
every nested AND/OR branch, with exact case-sensitive key segments.
`properties.total_discount` and `properties.Total` are different paths and remain
admissible. A valid filter using an excluded path returns
`FilterCapabilityNotImplemented`, not UNKNOWN or an invalid-filter diagnosis.
These are temporary execution limits, not restrictions of the full language.
Do not replace `sum.where` or sample requirements with separate profile
conditions or SQL: those do not select the same contributing events.

Amounts are known only when JSON Numbers exactly representable as Decimal(18,4).
Missing, null, other kinds and non-representable readings are unknown, never zero
or rounded. Empty sums are zero. With 20 EUR and an unknown amount in USD, the
EUR sum is exactly 20: certainly excluded amounts do not affect the result.
Amounts have no certified non-negative sign: 35 EUR and an unknown EUR amount
can yield UNKNOWN at >=30. With 35 EUR plus a known 20 of unknown currency, the
reachable extrema are 35 and 55, so >=30 is MATCH. Conflicting copies, execution
overflow and exhausted resources fail the complete operation, never yield UNKNOWN.

Sum explanations add `sum` with the same `start`, `end`, `unboundedStart`, and
`history` names and omission rules as Count, plus `field`, `nativeCurrency` and
`bounds:{lower,upper}`. They always include `certain` and `uncertain`, even
without a minimum: the latter counts additional potentially qualified events,
not the total possible count (`certain + uncertain`). Unknown amounts still
count. Finite monetary numbers are tight extrema under independent
per-event qualification outcomes; values between them need not be reachable.
For example, 20 EUR plus 20 of unknown currency has bounds 20 and 40, with only
those two totals possible.

Record `where` keeps Kleene logic without solving constraints between leaves
over the same missing field. For example, with a missing coupon, both
`coupon is PROMO AND coupon is not PROMO` and the corresponding OR yield UNKNOWN.
The occurrence can therefore remain optional in this qualification model, even
when joint reasoning about possible coupon values would determine the predicate.
Reachability refers to those ternary qualification outcomes, not to all possible
assignments of raw fields. This distinction also applies to Count.

A null bound means no finite bound in that direction, never zero. References
identify the reading actually resolved by core; bounds do not certify external
history or explain every individual event. An explanation does not make
unsupported forms available.
Exact JSON Numbers can use exponent notation (1000 can be serialized as `1E+3`)
and omit trailing fractional zeroes; compare their numeric value, not their spelling.

Raw JSON and governed business properties differ. With `properties.currency is
"EUR"`, a present JSON null is FALSE and a missing path is UNKNOWN; Number 100
is different from String "100". Negative comparisons do not make missing values
match. Count NativeCurrency bindings remain internal; do not infer one from raw JSON.
Sum resolves its currency reading through the explicit monetary field reference.

Event names and arbitrary JSON contents are workspace-specific. Ask the user or
use `query-data-warehouse` to discover them before constructing a filter. SQL may
also serve other exploratory analysis. It must not be used to emulate a segment
or complete a Unified feature that `query-profiles` cannot execute.

## Examples

To understand a known profile's result, call `explain-profile-filter` with its
`kpid`, the same JSON-text `filter`, and optionally the listing's `evaluation.at`
as `at`. It returns `{kpid,at,rule}`. Each rule has `kind` (`and`, `or`, `record`,
`count`, `sum`), `result` (`MATCH`, `NO_MATCH`, `UNKNOWN`), and ordered child `rules`
for groups. Count adds `{start,end,certain,uncertain}`: the reachable count is
every integer in `[certain,certain+uncertain]` for this supported subset.
For fullRelevantHistory, `start` is omitted and `unboundedStart:true` is present;
`end` is T. `history:{definition,meaningRevision}` identifies the resolved
recorded-history definition. There is no inferred beginning of the customer's
real relationship.

For example, 1 certain and 1 uncertain occurrence means Count >= 2 is UNKNOWN,
but Count >= 1 is MATCH. An OR can MATCH while a child is UNKNOWN; retain that
distinction. These are recorded occurrences, not deduplicated business orders.
Do not infer the missing field from these counters: the tool does not explain
individual qualification operands or return event payloads. It does not make
unsupported capabilities available and cannot reconstruct an earlier data view.
The explanation is scoped to that KPID. It can succeed when query-profiles fails
because of source-contract faults confined to other profiles. That success means
a complete evaluation of the target, not recovery of the failed population query.
Do not assemble successful explanations into a partial replacement for a failed
segment listing.

The tool uses the same MCP enablement and main connection as query-profiles.
It returns `ProfileNotFound` for a missing KPID; technical errors return no
partial explanation. Explanations, including Record-only ones, require the
verified Unified backend and are currently unavailable on Snowflake.

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

No Purchase in the entire retained, queryable recorded history:

```json
{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents","name":"Purchase","grain":"registered-message"},"period":{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}}},"operator":"is","values":[0]}]}
```

Between two and five Purchases in an absolute period:

```json
{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents","name":"Purchase","grain":"registered-message"},"period":{"kind":"absoluteHalfOpen","start":"2026-01-01T00:00:00Z","end":"2026-02-01T00:00:00Z"}},"operator":"is between","values":[2,5]}]}
```

Italian profiles with no Purchase whose **raw JSON** currency equals String EUR:

```json
{"operator":"and","rules":[{"property":"country","operator":"is","values":["IT"]},{"count":{"source":{"kind":"recordedEvents","name":"Purchase","grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}},"where":{"operator":"and","rules":[{"property":"properties.currency","operator":"is","values":["EUR"]}]}},"operator":"is","values":[0]}]}
```
