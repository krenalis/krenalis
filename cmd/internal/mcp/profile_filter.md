# Querying profiles with Record, Count, Sum, Average and Experience filters

Use `query-profiles` for questions about profiles and segments. It runs the same
`core.Filter` and `Workspace.QueryProfiles` path as `GET /v1/profiles`.
Read `profile-schema` and, for event qualification, `event-schema`. Use the schema
property `path`, not its physical warehouse column name. Fetch only the profile
properties needed to answer the user.

Use `list-event-definitions` to discover events by ID, technical name, display
name or description. The optional `query` is a literal substring, not semantic
search; omit it to compare the full catalog. Then use `get-event-definition`
with an exact catalog ID to inspect the native schemas and property meanings.
Both tools require a workspace MCP key and rate capacity.
Their JSON is returned as text and structured content.

Catalog IDs select metadata, not filter sources. For example, retrieve
`commerce.order_completed`, read its `event` value `Order Completed`, then use
that exact value as Count `source.name`. To find profiles with at least two
orders in the last thirty elapsed days, use:

```json
{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}},"operator":"is greater than or equal to","values":[2]}]}
```

This counts recorded messages, not deduplicated business orders. Catalog entries
do not prove which events were received or that historical data follows the
current schema. For page/screen, use `source.type` with an optional recorded
`source.name`, never the generic catalog ID/display name. Do not invent a track
event to represent a page or screen. The legacy merge
definition is historical, not an enabled ingestion or query capability.

The native schemas provide descriptions, units and nested structure. Conditions
on traits/properties still use the raw JSON operators described below, not
native date, currency or other governed semantics inferred from the catalog.
Declared array elements do not enable traversal or same-element predicates.
Business-order deduplication and other unsupported constructions must be
reported as unsupported, without SQL fallback.

## Recorded event discovery

Use `list-observed-event-sources` to find recorded track/page/screen names,
including custom events outside the catalog. Use `inspect-event-fields` for
observed paths and JSON types in a recent sample of one exact source. Both
use the workspace MCP key, rate capacity and main warehouse connection.

```json
{"type":"track","nameContains":"Purchase","start":"2026-01-01T00:00:00Z","end":"2026-02-01T00:00:00Z","limit":100}
```

`nameContains` is a case-sensitive literal substring of the stored technical
name. `%`, `_` and backslash are ordinary characters. Omit it or use an empty
string to impose no name restriction. Results are ordered literally by type
and name, with SQL NULL names first; they are not ranked by recency or frequency.
`recordedName` reads `event` for track and `name` for page/screen.

The source list defaults to 100 entries, at most 500. `truncated` reports limits
on entries, transported bytes or encoded output. `omittedLongNames` counts only
names longer than 1024 UTF-8 bytes in the inspected prefix, not all such names
in the workspace. A truncated result cannot establish that another name is
absent; narrow the type, literal search or interval. A large `limit` does not
bound the warehouse scan cost.

Copy the observed selector into the inspection tool, preserving null and empty:

```json
{"source":{"type":"track","recordedName":"Custom Purchase"},"start":"2026-01-01T00:00:00Z","end":"2026-02-01T00:00:00Z","sampleLimit":200}
```

This selector is different from Count's source. A nonempty usable name maps
to Count `source.name`, with its documented kind/type/grain. `recordedName:null`
and `recordedName:""` inspect those exact stored cases; neither means all
pages/screens. Do not omit Count's name to imitate either selector. No
`countSource` template is supplied. Discovery and Count compare technical source
names literally, independently of column collation.

Both tools use the event timestamp interval `[start,end)`. Supply both RFC3339
bounds or omit both for the last 30 elapsed days. Bounds have microsecond
precision, span at most 365 elapsed days and cannot end in the future. The
resolved UTC window is returned and can be reused; it anchors the interval,
not a snapshot of the data. Old timestamps imported later remain outside a
recent window. Explicit zero limits, null arguments and unknown properties
are invalid; only `source.recordedName` accepts null.

Inspection defaults to 200 stored rows, at most 1000, ordered by timestamp and
message ID descending. It is not a random sample or a deduplicated message or
business-order count. Copies with equal sorting fields can change which rows
fall at the sample edge. `sample.hasMoreRows` reports either the row cap or a
prefix shortened by the 8 MiB JSON budget.

The `roots` object describes `traits` and `properties`. For each root, `present`
counts non-SQL-NULL roots, including roots over 64 KiB that were not read.
`uninspected` counts those unread roots, and the sum of `kinds` plus
`uninspected` equals `present`. JSON null is present with kind `null`; SQL NULL
is absent. Roots can have any JSON kind, not just object.

`fields` contains descendant paths observed in inspected objects. Each field's
`present` and `kinds` cover the whole retained sample's inspected roots. Its
known absent rows equal `sample.rows - roots[root].uninspected - field.present`;
unread roots do not establish absence. Paths retain their exact segments.
Object keys may themselves be application data, such as identifiers in maps;
observed keys do not establish a stable schema.
`property` is supplied only for a path expressible in the current filter
grammar; it does not certify every operator or a governed reading.

The six kinds are null, boolean, number, string, object and array. Object keys
are visited in byte order, with at most eight segments below the root, 256
paths and an encoded-output budget. `sample.fieldsTruncated` reports these
path limits only, independently of `roots.*.uninspected`. Counts on retained
paths still cover the retained sample. Arrays are reported as arrays, without
element paths. No payload values or profile/message identifiers are returned.
An empty sample has zero root counters and an empty fields array, not NotFound.

Each JSON representation is limited to 128 KiB, excluding the MCP envelope.
The datastore bounds database work with a 30-second context. Technical read,
schema, invalid JSON and cancellation failures discard the result. Visible
limits do not justify partial results after a technical error.

Observed metadata does not establish a schema, required/nullable properties,
business meaning, units or allowed values. Ask the user for unknown meanings
or exact categorical values. A numeric path does not create a governed money
field, enable Sum/Average on custom events, or change raw JSON semantics.

## Connection and access

A valid workspace-scoped MCP key enables profile queries. The organization must
be enabled and requests consume rate capacity. No separate warehouse credentials
or additional workspace enablement switch is required.

These tools execute through Krenalis's **main warehouse connection**. Former MCP
credentials and their table, column and other database restrictions do not limit
which profile properties or supported event conditions a valid MCP key can read.
This includes existing keys whose old SQL access was restricted by warehouse
permissions. Revoke any key that should no longer have this workspace access.
Revocation is checked on subsequent calls, including in an initialized session;
it does not cancel an already admitted query.

On Snowflake the tools use the main virtual warehouse. Both PostgreSQL and
Snowflake support the Record, Count, Sum, Average and selected-experience forms
documented here. SQL execution is not provided.
Never bypass an unavailable capability through direct warehouse access.

## Arguments and results

`filter` is **JSON text in a String**, exactly as in the HTTP API query parameter.
This transports numeric literals without rounding or exponent conversion by the
MCP SDK. Do not send an object or SQL. Omit `filter` to list all profiles. An empty
String, JSON `null`, unknown arguments and explicit null arguments are invalid.

Other arguments: optional `properties` (top-level property names; omission requests
all), `order` (a sortable profile property), `orderDesc` (default false), `first`
(default 0), `limit` (default 20, range 1–1000), and `at` (behavioral filters only).
Arguments are limited to 500 KiB. The full HTTP POST, including its JSON-RPC
envelope, is limited to 1 MiB; oversize is HTTP 413 before the tool runs.
The complete result JSON is limited to 128 KiB.
An oversized result is an error: reduce `limit` or request fewer `properties`.
Rows are never silently dropped to fit that budget.

The returned text is JSON with `profiles`, `schema` and exact matching `total`.
With a behavioral filter, it also contains `evaluation: {at, match, noMatch, unknown}`. Totals
refer to the entire profile population before pagination, from the same
evaluation as the page. The page contains only MATCH profiles. UNKNOWN is not
NO_MATCH and is not an execution error. An empty page still returns totals.

On Snowflake, floating-point numbers in returned JSON can be rounded by the
server even though the filter compares their canonical values. A profile can
match `value > 100` while its returned JSON shows 100. Do not reclassify the
profile by reevaluating that displayed value; explain the precision limit if
it accounts for a visible discrepancy. Integer and decimal JSON digits remain
exact.

`at` is a nonfuture RFC3339 instant, exactly representable at microsecond
precision. Omission fixes T once. Reuse the returned `evaluation.at` for subsequent
pages to anchor relative periods. It does not freeze the data, identity mappings
or profile values between requests. Unified sorting uses NULLS LAST and KPID as a
deterministic tie-break; Record-only ordering retains the existing API behavior.

Invalid filters and unavailable capabilities return tool errors with actionable
messages/codes. Technical failures return an opaque operation error, never
invented totals, partial success, NO_MATCH or UNKNOWN. `ProfileResultTooLarge`
means reduce the page/projection; `ProfileProjectionTooWide` means request fewer
properties. `FilterCapabilityNotImplemented` and `UnifiedFilteringUnavailable`
must be reported to the user, not replaced with another query language.

The MCP adapter uses `InvalidArguments` for invalid requests and filters.
This is an MCP diagnostic prefix, not an additional core error code.

## Filter syntax and supported subset

The root is `{"operator":"and"|"or","rules":[...]}`, with optional
`defaultCalendar` as described below. Rules are Record
conditions, Count/Sum/Average/Experience/LatestN/LastVersusPrevious conditions or nested AND/OR groups. At most four main group
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

Record value comparisons on native String columns follow the warehouse's text
comparison rules, as in the existing SQL Record filters. Equality, inequality,
membership, ordering, intervals, containment, prefix and suffix can depend on
those rules. Case, accents and ordering may differ between warehouses or after
a column configuration change, even with the same Krenalis type. Krenalis does
not track or verify the column's collation.

For properties declared as `Array(String)`, `contains` and `does not contain`
compare whole String elements using the existing warehouse rules. PostgreSQL
uses the array column's collation; Snowflake compares the elements literally.
The same filter can therefore differ between warehouses or after a PostgreSQL
collation change. A typed `Array(String)` is distinct from a JSON property that
contains an array: JSON string comparisons remain literal. Missing/null values,
empty arrays and state operators retain their existing semantics.

Treat case sensitivity on native String properties as warehouse-dependent. If
the request requires a literal comparison independent of those rules, explain
that this is not guaranteed for native text; trying case variants does not
establish that guarantee. This applies to profile predicates and occurrence
where/verify bodies. A comparison error is a technical operation failure,
not an empty result or UNKNOWN.

Technical source selectors, internal message-identity comparisons used for
deduplication and selection, JSON string comparisons and governed currencies
retain their literal contracts. An ordinary Record on a native String column
such as `messageId`, `type`, `event` or `name` still follows the column's rules.

Count is `{"count":{"source":...,"period":...,"where":...,"minimum":...},
"operator":"...","values":[...]}`. It supports only:

- Source `{"kind":"recordedEvents","name":"EXACT EVENT NAME",
  "grain":"registered-message"}`. Optional `type` is `track`, `page` or
  `screen`; omission defaults to track without rewriting the document. Track
  requires `name`, matching the raw Event `event` value. For page/screen, `name`
  is optional and matches the Event `name` value. Omission selects all messages
  of that type. Explicit null or an empty string is invalid for either field.
  Names are literal and case-sensitive, independently of column collation.
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
- Optional `where` filters Event properties with Record predicates and one money
  predicate (see below), with at most two group levels and 100 descendants.
  Omission/null means no extra qualification. Count
  cannot be nested inside it. Main Record conditions refer to profile properties.
- Period `{"kind":"absoluteHalfOpen","start":"RFC3339","end":"RFC3339"}`
  with start < end <= T; or `{"kind":"elapsedLookback","duration":D}`; or
  `{"kind":"elapsedAgoBand","older":D1,"newer":D2}` with older > newer >= 0.
  A duration is `{"value":INTEGER,"unit":"UNIT"}`. Units: microsecond,
  millisecond, second, minute, hour, day (24 hours), week (7 × 24 hours).
  Periods are [start,end); events exactly at end/T are excluded. Elapsed units
  do not mean calendar days, weeks or months.

For three visits to `/pricing` in seven elapsed days, without requiring a page
name, use:

```json
{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents","type":"page","grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":7,"unit":"day"}},"where":{"operator":"and","rules":[{"property":"context.page.path","operator":"is","values":["/pricing"]}]}},"operator":"is greater than or equal to","values":[3]}]}
```

`source.name` is a source selection by the name actually recorded. A page/screen
with no name, SQL NULL name or an empty name is certainly excluded from a named
source such as Pricing. It does not make absence UNKNOWN. Omitting source.name
includes unnamed messages. In contrast, `where: name is "Pricing"` is a normal
Record value comparison: a missing name gives NULL and can make Count uncertain.
The two constructions are deliberately different; do not substitute one for the
other or generalize source selection to all optional properties.

Use `context.page.path` or `context.page.url` when those fields contain the
required information, or `properties.url` for the distinct raw JSON field.
There is no fallback between these paths. Native String comparisons use the
warehouse rules described above, so `/pricing` and `/PRICING` need not be
distinct; JSON string comparisons are literal. The filter does not normalize
URLs, decode percent escapes, strip query strings or perform prefix matching for `is`.
Use an explicit supported operator such as `starts with` when that is the request.
Missing decisive path/URL values retain Record UNKNOWN. A screen can likewise
use its recorded name or an appropriate Record qualification on its properties.
All existing Count periods, sample guards and explanations apply to these types.

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
The reference can remain on a Profiles filter without behavioral conditions; it is validated but
does not change Record results or add evaluation metadata. It is not allowed on
nested groups, occurrence `where` or in contexts restricted to Record filtering.
Use the timezone agreed with the user; ask if none has been established. Do not
infer it from the user's language, locale, server or event context. The calendar
does not reinterpret absolute instants, elapsed durations or raw JSON values.
It is optional for existing absolute/elapsed Count/Sum/Average filters. Per-condition and
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
reductions other than Count, Sum and Average, shares, other E forms and K/P/T are not available through this tool.

## Governed Count currency

For Count of messages in a governed native currency, supply both
`currencyField` and `nativeCurrency` inside `count`:

```json
{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}},"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is","values":[0]}]}
```

This asks for no recorded `Order Completed` messages in native EUR. Only that
currency definition/revision and track source are currently resolved. A page or
screen named Order Completed is an invalid monetary source. The fields are
optional as a pair, and neither accepts null. The literal requires three
uppercase ASCII letters, as for Sum. It does not convert currencies or deduplicate
business orders. Missing, JSON null and non-String currency values are unknown;
other Strings, including `"eur"`, are compared verbatim. There is no implicit USD
default. No relevant messages means Count zero and absence MATCH. One message
with an unknown business currency can make absence UNKNOWN. In particular, with
one message containing `"currency":null`, this governed absence is UNKNOWN while
the distinct raw JSON absence is MATCH. Never substitute one for the other.

An independent Record `where` can be combined with governed currency. Temporarily,
paths overlapping its reading (`properties`, `properties.currency` and descendants
of that currency path) return `FilterCapabilityNotImplemented`. This applies in
every nested group, with exact case-sensitive key segments. With a Record-only
where, `properties.Currency`, `properties.currency_code`, `properties.total` and
`properties.coupon` remain admissible. The governed currency requirement alone
does not read the amount; money adds that reading as described below. A false `where` excludes
the occurrence even with unknown currency. A missing coupon still follows raw
Record UNKNOWN; do not reinterpret it as a governed no-coupon value.
The sample minimum counts this qualified population and retains Kleene AND.
These are temporary execution limits, not changes to the full language.

Count explanations include the resolved `currencyField` and `nativeCurrency`
only when used. Existing raw Count explanations keep their original fields.

### Count messages satisfying a governed monetary condition

Use one money leaf in `count.where` to count individual messages whose native
currency and reported amount pass the test. For at least three registered orders
of at least 100 EUR each in the last 30 elapsed days:

```json
{"operator":"and","rules":[{"count":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}},"where":{"operator":"and","rules":[{"money":{"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is greater than or equal to","values":[100]}]}},"operator":"is greater than or equal to","values":[3]}]}
```

This is the preferred form for that request: money includes its own currency
requirement, so no outer currencyField/nativeCurrency pair is needed. Monetary
readings, exact threshold normalization, source compatibility and UNKNOWN rules
are the same as for money in experience.verify below. Money supports the four
ordered comparisons, is, is not and inclusive is between. Use one money is between
leaf for "between 50 and 100 EUR", with values [50,100]; do not split it into
two money leaves or raw amount/currency conditions. The Count terminal retains
all its existing comparisons independently of the monetary test.

| Requested measure | Construction |
|---|---|
| Number of individual native-EUR orders with total >= 100 | Count with money EUR >= 100 in where |
| Number of individual native-EUR orders with total between 50 and 100 | Count with money EUR is between [50,100] in where |
| Sum of native-EUR order totals >= 300 | Sum, field total, nativeCurrency EUR |
| Average native-EUR order total >= 100 | Average on that field/currency, with required minimum |
| Amounts converted from other currencies | Unavailable; no filter performs currency conversion |

Count counts registered messages, with message copies deduplicated, not business
orders. For "no order of at least 100 EUR", use the same where with Count is 0.
A potentially qualifying message with unreadable amount or unknown currency can
leave this absence UNKNOWN. A certainly different currency excludes it even if
the amount is unreadable.

The where can combine money with independent Record predicates using AND/OR.
Only one money leaf is admitted, including nested branches. Raw paths overlapping
the amount or currency reading (properties, properties.total, properties.currency
and descendants) return FilterCapabilityNotImplemented, with case-sensitive path
segments. properties.total_discount and properties.Total remain independent.
OccurredIn is invalid in Count.where. Money in Sum/Average.where or E base.where
remains unavailable. Invalid fields, operators and payloads remain errors.

The outer Count currency is an additional necessary requirement. If it reads the
same currency as money, the monetary test is evaluated under that requirement:
the same currency removes the redundant test; a different currency makes that
money leaf FALSE, including under OR. The outer requirement is retained.

Do not automatically remove an outer currency because money occurs somewhere in
the where. These requests differ for a USD message with coupon PROMO:

| Qualification | Is that USD PROMO message counted? |
|---|---|
| money EUR >= 100 OR coupon is PROMO, without outer currency | Yes |
| The same OR, with outer Count currency EUR | No |

Without OR alternatives, a required money EUR predicate alone or as a necessary
conjunct is equivalent with or without the additional outer EUR restriction.
Sample minimum still counts qualifying messages and uses the existing Kleene
guard; it is not a minimum on the money amount.

Explanations add `count.money:[{path,field,nativeCurrency,requested,normalized}]`.
Paths are zero-based rule indexes relative to count.where. The outer Count
currencyField/nativeCurrency remains separate from money[].nativeCurrency.
No event amount or witness is returned. Count explanations without money omit
this field, preserving their existing JSON.

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
`properties.currency` field. The source must be `recordedEvents`, type track
(explicit or omitted), exact name
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

This concerns the stored numeric value. Precision already lost during ingestion
cannot be recovered by the filter; the reading itself never rounds. In
particular, externally written scientific JSON numbers may have been changed
by Snowflake before registration. Do not infer a historical data guarantee
from the event catalog.

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
match. Count resolves governed currency through its explicit `currencyField`
reference; Sum through its monetary field, without a separate currency override.

Event names and arbitrary JSON contents are workspace-specific. The catalog
provides known definitions and schemas, not an inventory of all received events
or historical conformity. Use the recorded-event discovery tools for names and
field types absent from the catalog; ask the user about unknown meanings or
values. There is no SQL discovery or fallback tool. Report any
Unified capability that cannot execute instead of approximating the request.

## Monetary Average

Average is available for reported totals of track Order Completed messages in
one native currency. It uses the monetary field, periods and independent `where`
restrictions of Sum. For a reported average of at least 100 EUR with at least two
qualifying messages in thirty elapsed days:

```json
{"operator":"and","rules":[{"average":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}},"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},"nativeCurrency":"EUR","minimum":2},"operator":"is greater than or equal to","values":[100]}]}
```

`minimum` is required: omission, null and zero are invalid. It counts qualified
messages, including those with unknown amounts, and combines independently with
the value test using Kleene AND. An unreachable minimum is NO_MATCH even when
the average is unknown or undefined. Empty populations do not have a zero average.

The four ordered comparisons are supported. Thresholds are exact signed JSON
Numbers within the global 76-digit/37-scale limits and must be representable
with at most four fractional decimal places; trailing zeros do not matter.
Finer thresholds return FilterCapabilityNotImplemented, without rounding.
Equality, inequality and intervals remain unavailable in this first increment.
No division or rounded average is used to decide membership.
The scale limit applies to the threshold, not to the mathematical average.

The value test requires a determined population and known amounts for every
qualified message. Missing/null/incompatible amounts are not skipped or replaced
by zero. Uncertain qualification or unknown relevant amounts retain UNKNOWN,
unless the sample guard is certainly false. Values certainly in another currency
or excluded by where do not count. Even out-of-domain thresholds do not turn
unknown amounts into known values or certify their business domain.

This is the average of reported message totals in the selected native currency,
excluding other currencies without conversion. It is not an all-currency average
or a deduplicated business-order average. Keep coupon qualifications inside
`average.where`, not in sibling profile conditions. The same temporarily excluded
amount/currency paths as Sum apply.

Average explanations have kind `average`, with the usual period fields, field
reference and native currency, plus `sum`, `count`, `certain` and `uncertain`.
`sum` and `count` are exact JSON numbers, possibly using exponent notation;
neither is a rounded average. Count is null when membership is uncertain. Sum is
null unless membership and all amounts are known. Null never means zero. An
empty population reports sum=0 and count=0, while its required guard fails.

## Selected Experience: First/Last/Nth then verify

An `experience` rule selects the first, last or Nth recorded message from a declared base,
then verifies that message. It is a complete predicate, without an outer
`operator` or `values`. For profiles whose last Order Completed reports an
order ID starting with WEB-:

```json
{"operator":"and","rules":[{"experience":{"base":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}}},"select":{"kind":"last"},"verify":{"operator":"and","rules":[{"property":"properties.order_id","operator":"starts with","values":["WEB-"]}]}}}]}
```

`base.source` and `base.period` reuse the available Count sources and periods,
including named/unnamed page and screen sources, calendars and recorded history.
`base.where` optionally qualifies events **before** selection. `verify` is a
mandatory nonempty filter evaluated **after** selection. Both use the
Event schema and the 26 Record operators, at most two group levels and 100
descendant nodes. Verify additionally admits occurredIn and money leaves described below.
Neither admits behavioral rules or a local defaultCalendar. Omitted/null
base.where means no additional qualification. Null/omitted verify is invalid in
this public subset; the canonical NoAdditionalVerification form is not exposed.
An explicit select.kind is required: `{"kind":"first"}`, `{"kind":"last"}`
or `{"kind":"nth","position":3}`. Nth requires a JSON integer from 1 through
10 and selects one message, counting chronologically from the start of the base.
Position is invalid with First or Last. Latest N remains unimplemented.

Moving a predicate from verify into base.where changes the request. In the
example, verify tests the latest Order Completed. In base.where, the same test
would instead select the latest qualifying WEB- message, even if a later order
has another ID. Never fall back to an older message that passes verify or to the
latest non-null value. These are messages, not deduplicated business orders.

**The period restricts the base.** With Last and a period ending exactly at T,
this gives the same semantic outcome as a recency check on the last occurrence
of the same source and qualification, when both evaluations succeed. If the end
is before T, it means **last within that period**,
not that the overall latest occurrence lies there. Elapsed ago bands and complete
calendar periods normally have an earlier end; use the actual resolved bounds
(a calendar boundary can coincide with T).

For "the last order was 60–90 elapsed days ago", combine Count is 0 in
elapsedLookback(60 days) AND Count >= 1 in elapsedAgoBand(older=90 days,
newer=60 days), with identical sources and qualifications. The band is
[T-90 days,T-60 days); exactly T-60 belongs to the recent window. Do not use
Experience on that band alone: it would ignore more recent orders.

**First means first recorded, not first in real life.** Retention, deletions and
late imports of older events can change the first identified occurrence. First
on elapsedLookback(30 days) means first within those 30 days; it does not identify
customers whose first historical order is recent. Use First on fullRelevantHistory
with occurredIn in verify for the recent window. Do not present First in a recent
base as new customers.

### Select an ordinal occurrence

For "the third registered EUR order is recent and reports a total above 100",
select Nth on the full history of EUR orders, then verify time and amount:

```json
{"operator":"and","rules":[{"experience":{"base":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}},"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"},"select":{"kind":"nth","position":3},"verify":{"operator":"and","rules":[{"occurredIn":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}},{"money":{"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is greater than","values":[100]}]}}}]}
```

Do not silently choose between these readings:

| Request | Base | Select | Verify |
|---|---|---|---|
| Third historical EUR order is recent and above 100 | Full registered history, governed EUR | Nth 3 | occurredIn 30 days AND money EUR > 100 |
| Third EUR order within the last 30 days is above 100 | Last 30 days, governed EUR | Nth 3 | money EUR > 100 |
| Third historical order is in EUR, recent and above 100 | Full registered history, all currencies | Nth 3 | occurredIn 30 days AND money EUR > 100 |

Ask for clarification or state the chosen interpretation when "third EUR order
in the last 30 days" is ambiguous. The base period limits which messages occupy
positions; occurredIn tests the time only after selection. Positions refer to
the queryable registered history, so retention, deletions and late imports of
older messages can move them. "Third order" is not "at least three orders"
(Count >= 3) or "exactly three orders" (Count = 3). A fourth order does not cancel
the third. Verify never falls back to another position that passes its test.

### Verify when the selected event occurred

Only experience.verify admits `{"occurredIn":{...period...}}`, as a complete
leaf without operator or values. It can be combined with Record predicates in
AND/OR. It accepts elapsedLookback, elapsedAgoBand, lastCompleteCalendarPeriods
and absoluteHalfOpen. Calendar periods inherit the main filter's defaultCalendar;
a local calendar is invalid. Every window is [start,end), using the same T and
microsecond precision as the base. No history reference is allowed in those
four period forms. FullRelevantHistory here is valid but returns
FilterCapabilityNotImplemented; it is not a reason to change Record predicates.
The leaf is invalid in base.where, Count/Sum/Average.where, Profiles root or
Record-only pipelines.

For profiles whose **first recorded Order Completed** is within the last 30
elapsed days, select First over the entire queryable history, then check its time:

```json
{"operator":"and","rules":[{"experience":{"base":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}}},"select":{"kind":"first"},"verify":{"operator":"and","rules":[{"occurredIn":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}}]}}}]}
```

This covers the temporal part of the request, without asserting a real-world
new customer or applying governed monetary conditions. A customer with an older
recorded order and another recent order does not match; First over a recent base
would match instead.

For "the last recorded order was 60–90 elapsed days ago", use the same full
history base, select Last, and verify
`{"occurredIn":{"kind":"elapsedAgoBand","older":{"value":90,"unit":"day"},"newer":{"value":60,"unit":"day"}}}`.
This gives the same semantic outcome as the two-Count recipe above when both
evaluations succeed with the same source and qualification. Profiles without any
recorded order do not match either form.
Last on a base ending at T gives the same semantic outcome as Last on full history
with occurredIn for that base window when both evaluations succeed and their
additional qualification and verification are the same. The full-history form
checks more identities: faults in older messages can fail that form alone.
This does not imply identical execution failures or selection metadata.
A base ending before T does not express that recency test.

OccurredIn is TRUE or FALSE for each candidate's determined timestamp. It can
still leave the experience UNKNOWN: a possible winner inside the window and a
certain winner outside disagree, or the only in-window winner can be absent.
Explanations add `experience.occurredIn:[{path,start,end}]`. Path is the zero-based
sequence of rule indexes within verify; each leaf has its resolved half-open
bounds, without inventing a selected witness. Use these bounds to explain a date
failure and keep uncertainty about the selection separate from the time test.

The admitted evidence is local Record qualification with independent uncertainty
across determined message identities and known timestamps. With at least one
certainly qualified message, the candidate winners are all certain/possible
messages at or after the latest certain timestamp. Without any certain message,
all possible messages are candidates and the empty selection is possible.
The empty alternative is NO_MATCH. The rule is MATCH only when every candidate
passes verify and emptiness is impossible; NO_MATCH when every alternative fails;
otherwise UNKNOWN. A tie never uses message ID or reception time as a semantic
tie-break. All tied candidates passing can give MATCH; mixed results give UNKNOWN.
For First, candidates are all certain/possible messages at or before the earliest
certain timestamp. Without certain messages the same empty alternative applies.
Older optional messages cannot affect Last after a later certain message; later
optional messages cannot affect First before an earlier certain message.

For Nth, an identity is a candidate when N-1 lies between the number of certain
identities strictly earlier and the number of all non-excluded identities
earlier or tied, excluding itself. This uses determined identity/timestamps,
independent possible memberships and all orders within ties. Absence is certain
when fewer than N messages can qualify, impossible when at least N certainly
qualify, and possible otherwise. Absent Nth is NO_MATCH; possible absence leaves
UNKNOWN if any candidate can pass. Nth(1) has the same outcomes and candidates as
First, but its explanation retains `{"kind":"nth","position":1}`.

Copies of an identity must agree on attribution and period membership. For an
identity not certainly excluded from the base, they must also agree on the exact
stored timestamp, qualification and relevant verification. A mismatch is a
source-contract execution failure, including two timestamps inside the same
period. No partial listing or invented UNKNOWN is returned. Unknown timestamps
are outside this determined-time source contract, not silently excluded candidates.

Explanations have kind experience and an experience object with the standard
period fields, select and selection:{kind,absence,candidates}. Selection kind is
absent, unique or ambiguous. Absence is certain (zero candidates), possible
(candidates exist but the requested position may not exist) or impossible. A single candidate
with possible absence is still ambiguous. Candidate counts refer to distinct
possible first/last/Nth-message identities, not all events or completed business orders.
No arbitrary witness is selected or unbounded event list returned. Ambiguous
selection can still have a determined result when every alternative agrees.

Verify retains raw Record semantics: JSON null, missing paths and incompatible
kinds behave exactly as documented above. Raw currency is not governed
NativeCurrency. The money leaf below provides the governed monetary test.
Shared governed facts, cross-profile references, Latest N, relative scopes and
following events remain unimplemented. Do not approximate those requests with raw predicates, Count
or SQL unless they express the same explicitly requested meaning.

### Verify native currency and amount

Use one money leaf in experience.verify to test the selected message's native
currency and reported total together. The same field reference as Sum binds
properties.total and properties.currency; the caller cannot choose either path.
Only track Order Completed at registered-message grain is compatible with
commerce.order_completed.total@1. Another source is invalid configuration.

For the **last recorded Order Completed**, occurred within the last 30 elapsed
days, whose reported total is greater than 100 in native EUR:

```json
{"operator":"and","rules":[{"experience":{"base":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}}},"select":{"kind":"last"},"verify":{"operator":"and","rules":[{"occurredIn":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}},{"money":{"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is greater than","values":[100]}]}}}]}
```

First uses the same leaf after selecting the first recorded message. Neither
currency nor amount prefilters the base. A later USD message fails the EUR
requirement; never fall back to an earlier EUR message. Copies, ties, optional
candidates and possible absence retain the existing selected-experience rules.
This uses recorded messages, without conversion or business-order deduplication.

Money is Kleene AND of the native currency requirement and exact amount test.
The table describes the predicate on one message; the experience closes the
results over all compatible selected candidates and possible absence.

| Currency | Amount | Money outcome |
|---|---|---|
| Equal | Readable | Amount comparison outcome |
| Equal | Unreadable | UNKNOWN |
| Certainly different | Any | FALSE |
| Unknown | Readable, comparison false | FALSE |
| Unknown | Readable, comparison true | UNKNOWN |
| Unknown | Unreadable | UNKNOWN |

Currency uses the same verbatim String reader as Count and Sum. Missing, JSON
null and non-String values mean unknown currency; USD, eur and the empty String
are certainly different from EUR. There is no implicit USD default or case
normalization. NativeCurrency in the request requires three uppercase letters.
Amounts use Sum's exact Decimal(18,4) JSON Number reader. Missing, null, strings,
other kinds, numbers too fine or outside the reading domain are UNKNOWN, never
rounded or coerced. Trailing zeroes do not make an exactly representable number
unreadable. For EUR >=100, total 80 with unknown currency is FALSE: even if
EUR, it is insufficient. Total 120 with unknown currency is UNKNOWN.

Money supports is, is not, is less than, is less than or equal to,
is greater than, is greater than or equal to, and is between. Values contains
one exact signed JSON Number, or two for the inclusive interval, within global
precision 76/scale 37 and the monetary literal length limit, as for Sum.
Is not negates only the amount comparison: money EUR is not 100 means native
EUR with an amount different from 100. It never includes USD or another currency.
Finer ordered thresholds are normalized exactly to the readable grid:
>100 becomes >=100.0001; >=100.00001 becomes >=100.0001. This never rounds event
amounts. Comparisons outside +/-1e14 have a determined amount test only for
readable amounts: 1e20 in an event remains unreadable, even against an extreme threshold.
Equality never rounds the literal: is 100.00001 is FALSE for every readable
amount; is not 100.00001 is TRUE for every readable amount. Both retain UNKNOWN
for an unreadable amount. Numeric spellings 100, 100.0 and 1E2 are equal.
An interval is rounded inward to the readable grid, with inclusive endpoints:
[50.00001,100.00009] becomes [50.0001,100]. [99.99995,100.00004] remains an is
between test with [100,100]. A valid interval may contain no readable number,
such as [100.00001,100.00009]; this does not make the request invalid.
An inverted requested interval is invalid. Operators, arity and operand limits
are still validated even next to an unsupported rule. Sum and Average still
reject equality, difference and intervals as FilterCapabilityNotImplemented.

For "the last order in EUR between 100 and 200", disambiguate selection:
money in verify tests the last order overall; governed currency in the base
selects the last native-EUR order first. Never silently move the currency test.

This first subset admits at most one money leaf per experience verify, including
nested OR branches. It can be combined with occurredIn and Record leaves on
independent paths. When money occurs, raw Record paths overlapping its amount
or currency are temporarily unavailable in both base.where and verify. The check
covers properties, properties.total, properties.currency and their descendants,
with exact, case-sensitive path segments. The paths `properties.total_discount`
and `properties.Total` are different and remain allowed.

Multiple money leaves and these overlaps return FilterCapabilityNotImplemented;
they do not turn into UNKNOWN or silently drop conditions. Money qualification
in base.where or Sum/Average.where is likewise valid but not implemented.
Count.where admits one money as described above.
Money is invalid on the Profiles root and in Record-only pipelines. A raw
properties.currency is EUR prefilter is therefore unavailable together with
money in verify: it differs from governed currency on JSON null and must not
replace the governed base currency described below. Never split the tests
into independent profile conditions to approximate the selected-message request.
To select among native-EUR orders, restrict the base using the governed fields
described below. Raw predicates do not provide the governed currency and amount
reading and must not substitute for these fields.

Explanations add experience.money:[{path,field,nativeCurrency,requested,normalized}].
Path is zero-based within verify. Requested always contains operator and values,
with exact JSON numbers (possibly exponent notation). Normalized contains either
operator/values or readableAmounts:"none"|"all", never both. For is, is not and
is between, nondegenerate normalized tests retain the requested operator.
None means no readable amount satisfies the amount test; all means every
readable amount does. Both preserve UNKNOWN for unreadable amounts and retain
the separate currency requirement. For example, is 100.00001 normalizes to none,
is not 100.00001 to all, the narrow interval above to none, and [-1e20,1e20] to all.
These domains are not constant outcomes for the money leaf.

Existing ordered explanations retain their comparison format: >100 is still
normalized as >=100.0001. Clipped +/-1e14 bounds are sentinels outside every
readable amount: >=1e14 and <=-1e14 accept none; <=1e14 and >=-1e14 accept all.
They do not assert that an unreadable amount is known or bounded. This historical
format is intentionally retained instead of replacing it with readableAmounts.
No candidate amount or witness identity is exposed. An empty money list is omitted.

### Restrict First/Last/Nth to a native currency

`experience.base` accepts the optional `currencyField` and `nativeCurrency` pair
from Count. Supply both or neither; explicit null is invalid. The resolved field
`commerce.order_completed.currency` revision 1 is compatible only with track
Order Completed. The currency requires three uppercase ASCII letters. This is a
population restriction, without conversion or business-order deduplication.

For the last recorded **native-EUR** Order Completed, within the last 30 elapsed
days and reporting a total greater than 100:

```json
{"operator":"and","rules":[{"experience":{"base":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}},"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"},"select":{"kind":"last"},"verify":{"operator":"and","rules":[{"occurredIn":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}},{"money":{"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is greater than","values":[100]}]}}}]}
```

The money currency remains mandatory and explicit. The base fields select the
population first; money then verifies the selected message. First uses the same
rules with chronological order reversed. With EUR 150 followed by USD:

| Requested meaning | Currency placement | Outcome |
|---|---|---|
| The last order is in EUR and exceeds 100 | Verify only | NO_MATCH |
| Among EUR orders, the last exceeds 100 | Base and verify | MATCH |

For ambiguous wording such as "last order in EUR above 100", ask which meaning
is intended or explicitly state the interpretation before presenting the result.

Unknown currency means possible membership, not exclusion. A selected candidate
must satisfy the base currency. When money uses the same resolved reading, its
currency requirement is already TRUE for the same currency. A different currency
makes the whole leaf FALSE, even with an unreadable amount. This applies inside AND/OR
at every allowed verify depth. Other readings keep the complete monetary test.
It does not assert a currency for an event that is not selected or normalize raw
JSON. With EUR 150 followed by a possible EUR 150, both possible winners pass:
MATCH. With only a possible EUR 150, absence remains possible: UNKNOWN.
A USD money leaf under a EUR base is FALSE, not an invalid filter or capability
error; another OR branch can still pass. All field/operator/operand checks,
operand checks and the one-money limit remain in effect.

When the base fixes currency, raw paths overlapping that currency reading are
temporarily unavailable in both base.where and verify, even without money:
properties, properties.currency and descendants. Matching uses exact key
segments; properties.Currency and properties.currency_code are distinct. Raw
properties.total in verify remains allowed without money. With money, its
additional raw amount overlaps remain unavailable as before. Invalid parameters
still take precedence over capability errors.

Experience explanations add currencyField/nativeCurrency for the base only when
supplied. The money array retains each verify currency and requested/normalized
comparison. Neither metadata set replaces the other. Experiences without base
currency retain their prior JSON shape and meaning.

## Latest N: quantification on a complete selected block

A separate `latestN` rule selects the latest N recorded messages from its base
and tests one uniform `predicate` on the entire compatible block. `size` is a
JSON integer from 1 to 10. It requires N messages, not up to N, and has no
outer operator or values. It does not pick N passing messages: selection is
before the predicate. Do not replace it with Count of passing messages.

The base is exactly experience.base: same source, period, Record-only where,
and optional governed currencyField/nativeCurrency pair. Source identity,
deduplication, timestamp and currency conditioning rules are unchanged.
The predicate is an AND/OR group with the same two-group/100-descendant budget
as experience.verify, but accepts only Record and one money leaf, not occurredIn.
Money supports all seven current comparisons, including inclusive between.
The existing raw-overlap and shared-dependency restrictions still apply.

For "the latest three orders are all in EUR and between 50 and 100":

```json
{"operator":"and","rules":[{"latestN":{"base":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}}},"size":3,"predicate":{"operator":"and","rules":[{"money":{"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is between","values":[50,100]}]},"quantifier":{"kind":"all"}}}]}
```

Adding EUR currencyField/nativeCurrency to the base asks instead for the latest
three EUR orders. More recent USD orders remain outside that population.
Messages count at registered-message grain, not deduplicated business orders;
there is no currency conversion. Ask or state the interpretation when ambiguous.

Possible memberships and timestamp ties are resolved as compatible whole blocks,
never by combining independent winners for each position. With N=2, a possible
recent passing message, a certain middle failing message and a certain older
passing message give NO_MATCH: every complete block contains the failing one.
Three certain tied messages, two passing and one failing, give UNKNOWN for N=2.
No technical ID tie-break is used. A short base gives NO_MATCH; a possibly short
base prevents MATCH, but still gives NO_MATCH if no complete block can pass.

A base period ending before T means "latest N within that period", not "the
latest N of the history are within that period". For "the latest N are all in
the last 30 days", use Count >= N in that trailing window, with the same base
qualification. This equivalence requires a window ending at T; do not extend
it to ago bands or completed calendar periods. Retention, deletion and late
imports can change which registered messages make up a historical block.

Three explicit quantifiers execute: `{"kind":"all"}` requires the predicate on
every selected message; `{"kind":"some"}` requires it on at least one.
Cardinality tests how many messages within the complete block pass:

```json
{"kind":"cardinality","operator":"is greater than or equal to","values":[2]}
```

Cardinality has exactly four operators: `is` means exactly, `is greater than or
equal to` means at least, `is less than or equal to` means at most, and `is between`
is inclusive. `is not`, strict greater/less and other operators are invalid,
not silently rewritten. Values are JSON naturals (zero allowed, -0 reads as 0),
without fractions or exponents: one operand, two for between. Inverted intervals
and tests incompatible with [0,N] are invalid. Redundant upper bounds are valid.
All and Some have no operator/values; unknown fields, kinds and null are invalid.
Even Cardinality >= 0 requires a complete block: certain shortage gives NO_MATCH,
possible shortage prevents MATCH. Absence is not a block with zero passing
messages. "At least two of the latest three orders" uses latestN with size 3
and Cardinality >=2; "at least two orders in the last 30 days" uses Count.
Do not substitute one for the other.

Counts close over jointly compatible complete blocks. With N=2 and three
certain tied messages, two passing and one failing, Some is MATCH, All is
UNKNOWN, and Exactly 0 is NO_MATCH. With a possibly short base even a
predicate true on every full block can remain UNKNOWN. An unknown per-message
predicate makes the result UNKNOWN only when its completions can change the
requested test. This is neither independent position-by-position selection nor
K's N2 composition.

Explanations have kind latestN and a latestN object with the same period/history
and base currency metadata as experience, plus size, the requested quantifier,
selection and any money metadata (paths relative to predicate). Quantifier
operands remain plain natural digits, without truncating redundant upper bounds
to size; -0 is reported as 0. The internal reachable-count extrema, witnesses
and number of compatible blocks are not returned. selection.candidates counts
the union of identities in complete compatible blocks. With certain shortage it is zero;
absence is certain. Otherwise absence is possible or impossible. selection.kind
is unique exactly when absence is impossible and candidates equals size: it
describes the set, not its internal order. Three certain tied messages selected
with size=3 form a unique set. An unknown predicate can still make its result
UNKNOWN. Latest(1)/All and Latest(1)/Some have Last's outcome and selection
metadata, while retaining the requested latestN form.

## Last versus Previous: a reported monetary difference

Use `lastVersusPrevious` for a comparison of the last two messages, selected
**together** from the same base. For "the last EUR order decreased by at least
30 euros compared with the preceding EUR order":

```json
{"operator":"and","rules":[{"lastVersusPrevious":{"base":{"source":{"kind":"recordedEvents","name":"Order Completed","grain":"registered-message"},"period":{"kind":"fullRelevantHistory","history":{"definition":"registered-queryable-events","meaningRevision":"1"}},"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"},"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},"check":{"kind":"difference","operator":"is less than or equal to","values":[-30]}}}]}
```

Difference is always **Last minus Previous**. From 100 to 70 the difference is
70 - 100 = -30; "decreased by at least 30" means <= -30, not >= 30. The threshold
has the native currency of the base. There is no exchange-rate conversion.
These are registered messages, not deduplicated business orders: different
message IDs can represent the same business order and still occupy two positions.

The base is the same closed object as experience.base: source, period, optional
Record-only where and the currencyField/nativeCurrency pair. This execution
subset requires both currency fields and track Order Completed, with the bound
total field above. Omission of both currency fields is a valid unimplemented
capability; supplying only one, null or incompatible references/sources is invalid.
The currency of every selected message is fixed by membership, including a
possible member whose stored currency is unknown. A certainly different currency
is excluded. Missing/null/non-string currency remains possible, with the same
reading as Count and Sum.

"Last two orders in EUR" selects from EUR messages, ignoring later USD orders.
"Last two orders, both in EUR" selects from all orders and then verifies the
currency. The latter difference comparison is not implemented: do not silently
restrict the base to EUR or replace it with two independent selections. Ask or
state the interpretation when the user's wording is ambiguous. Raw where paths
overlapping the bound total or currency, their parents or descendants are
temporarily unavailable. Other independent Record conditions remain available.

Periods restrict the base before selection. A period ending before T asks for
the last two **within that period**, not the last two of history followed by a
recency check. Retention, deletion and late imports may change the registered
pair. Equal timestamps admit both orders: there is no ID tie-break. Two certain
messages at the same time give an ambiguous ordered pair even when they form
a unique set. Different possible pairs can nevertheless agree on the result.

The check is closed. `sameValue` and `differentValue` admit only kind.
`difference` and `ratio` require operator and values, using the seven monetary
operators: is, is not, the four ordered comparisons and inclusive is between
(two values). Inverted intervals, unknown fields, unknown kinds and null are
invalid. The public interval is inclusive only, a deliberate subset of the
normative interval forms. Currently only difference with <, <=, > or >= executes;
other valid checks return FilterCapabilityNotImplemented after complete
validation. Do not emulate them with raw conditions or SQL.

Requested thresholds are exact JSON numbers within 76 significant digits and
37 decimal places, including negatives and exponent notation. Both selected
amounts must be readable as JSON Numbers exactly representable as Decimal(18,4),
without conversion or rounding. Missing, null, strings, too-fine numbers and
out-of-domain values are unknown. Readable differences lie in the **inclusive**
range [-199999999999999.9998,199999999999999.9998].

Explanations have kind lastVersusPrevious and a lastVersusPrevious object with
start/end/unboundedStart/history under the same rules as Count, field, base
currencyField/nativeCurrency, check and selection. Check retains kind and the
requested comparison. Normalized contains either an inclusive operator/values
bound on the scale-four grid, or `{"readableDifferences":"none"}` /
`{"readableDifferences":"all"}`.
The two forms are exclusive. Numeric output may use exponent notation and can
be reused as exact input. None/all describe **only pairs with two readable
amounts**. A possible pair with an unreadable amount still yields UNKNOWN even
with all; all is not unconditional TRUE. Ordered money tests retain their older
clipped +/-1e14 sentinels for compatibility; difference uses its own wider
inclusive domain and explicit none/all. Do not interpret those money sentinels
as the domain of a difference.

Selection reports candidate identities, not the number of pairs. Shortage is
certain with fewer than two possible messages, possible with fewer than two
certain messages, otherwise impossible. Certain shortage is NO_MATCH; possible
shortage prevents MATCH but gives NO_MATCH if no compatible pair can pass.
Unique means one **ordered** identity pair and impossible absence. No amounts,
difference extrema or witnesses are returned. Technical faults fail the complete
operation.

## Examples

To understand a known profile's result, call `explain-profile-filter` with its
`kpid`, the same JSON-text `filter`, and optionally the listing's `evaluation.at`
as `at`. It returns `{kpid,at,rule}`. Each rule has `kind` (`and`, `or`, `record`,
`count`, `sum`, `average`, `experience`, `latestN`, `lastVersusPrevious`), `result` (`MATCH`, `NO_MATCH`, `UNKNOWN`), and ordered child `rules`
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
verified Unified backend, available on PostgreSQL and Snowflake.

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
