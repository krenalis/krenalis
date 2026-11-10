# Profile queries through MCP

`query-profiles` calls `Workspace.QueryProfiles`, with the same Record/Count/Sum/Average
document and execution path as `GET /v1/profiles`. `profile-filter-doc` exposes
the supported syntax, examples and limitations to the model. The existing SQL
tool remains available for discovery and other analysis. Server instructions
direct profile/segment questions to Unified and prohibit SQL fallback for
unavailable filtering capabilities.

`explain-profile-filter` calls `Workspace.ExplainProfileFilter`, also available as
`GET /v1/profiles/{kpid}/filter-explanation`. It accepts `kpid`, JSON-text `filter`
and optional `at`, and returns the ordered rule tree with ternary outcomes and
resolved periods, occurrence counters and monetary bounds. It uses the same MCP
enablement and unrestricted main connection as `query-profiles`. No event payloads are returned. It is a new
evaluation, not a snapshot or an alternative segment-membership policy; all
explanations remain unavailable on the unverified Snowflake backend.

`lastVersusPrevious` compares the reported total of the last two recorded
Order Completed messages in one governed native currency. Its base is shared
with experience/latestN; only difference with four ordered comparisons executes.
Explanations describe the ordered pair's candidate union, absence and uniqueness,
and requested/normalized tests, without returning amounts or witnesses. The
guide distinguishes this population from the last two messages of all currencies.
The Snowflake gate remains closed for this form too.

## Event catalog

`list-event-definitions` returns metadata summaries, with optional literal
case-insensitive substring search (256 UTF-8 bytes). `get-event-definition`
accepts an exact ID and returns the native traits/properties schemas with
descriptions and units. Both call `core.Workspace`, not internal state or the
warehouse, and use the usual workspace MCP authentication and rate limit.
MCP warehouse settings are not required for catalog access.

Arguments must be an object of at most 2 KiB, with only string `query` (list)
or `id` (get); null and unknown arguments are rejected. Results are complete
JSON, published as structured content and equivalent JSON text fallback, with a
128 KiB JSON payload limit. Empty searches return `definitions: []`.
Unknown IDs produce `EventDefinitionNotExist`; malformed arguments use the MCP
`InvalidArguments` prefix. There is no new HTTP endpoint or filter syntax.

The documentation distinguishes schema metadata from historical conformity,
the catalog ID from the exact track event name, and native field meaning from
raw JSON predicate semantics. It explicitly excludes inferred capabilities for
arrays, sums, governed bindings and identity definitions. For page/screen Count,
the generic catalog definition supplies metadata; the source uses `type` and an
optional recorded `name`, never the catalog ID or display label.
The catalog does not change invalid-event handling or ingestion conversions.

## Access and connections

A workspace-scoped MCP key, enabled organization, request rate capacity and
configured MCP warehouse settings are required. Removing those settings disables
new profile requests. This does not terminate an already admitted query.

The settings are an **enablement condition** for `query-profiles`. Execution uses
the main Krenalis datastore, connection and warehouse mode checks. The separate
MCP database credentials are used only by `query-data-warehouse`: their table,
column, row or other database restrictions do not constrain profile queries or
the event properties used in their predicates. Administrators must account for
this when enabling MCP access. This change does not grant write tools or accept
user-supplied SQL through the profile path.

On Snowflake this also selects the main virtual warehouse instead of the one
possibly configured for MCP. Record-only queries use the existing Record path.
Count, Sum and Average preserve the unverified-backend gate and return an actionable capability
error. Local tests do not establish Snowflake runtime conformance.

## Transport and limits

The `filter` argument is JSON **text**, as in the HTTP query parameter, decoded
by `core.Filter.UnmarshalJSON`. This is the same filter language, with no new
syntax or persistence. Numeric operands never traverse an `any`/float64 map.

The vendored MCP SDK retains `RawArguments` but also unmarshals an untyped map
(`mcp/tools.go`, `CallToolParams.UnmarshalJSON`). An object argument with a
400-digit natural is rejected by that second decode before reaching a handler.
Transporting the document as text also preserves those valid-language operands,
so core can classify execution range limits and invalidity in the proper order.
No SDK change or global schema-validation setting is needed.

Wire arguments are decoded strictly from the original JSON, with exact Go
integer pagination and rejection of unknown/null arguments. The default limit
is 20; core retains the range 1–1000. Arguments are limited to 500 KiB. The full
result JSON, including schema and metadata, is limited to 128 KiB before any
content is returned. Oversize yields `ProfileResultTooLarge` and advice to reduce
limit/projection; no rows are truncated. The underlying HTTP/core/warehouse
budgets are unchanged. The response budget is not a new warehouse scan budget
for the legacy Record path.

Every MCP POST has a separate **1 MiB HTTP body limit**, including the JSON-RPC
envelope. A declared length above that limit is rejected before reading; actual
bytes are bounded with `http.MaxBytesReader`, including unknown/chunked lengths.
Oversize returns HTTP 413 before SDK parsing or tool execution. Other body read
failures return opaque HTTP 400; malformed JSON within the limit is still handled
by the SDK. GET/DELETE/HEAD handling is unchanged. The admitted body is buffered
and replayed to the SDK, so both reads are bounded; this is not a zero-copy path.

The tool's 500 KiB argument limit is still checked after SDK decoding. Attribute
byte checks follow each call to `types.Marshal`, so the response budget does not
bound the allocation used to encode one profile's attributes. The legacy Record
page is already loaded before that step.

Record attributes use `types.Marshal`, preserving Decimal, Int64 and raw JSON
encoding. Count, Sum and Average return only MATCH profiles, plus `evaluation` with all three
totals and T. Empty pages use `profiles: []` and retain totals. Reusing `at`
anchors relative periods; it does not freeze the data between calls. Count, Sum and Average
retain their deterministic tie-break, not a new Record sorting contract.

`BadRequestError` and `UnprocessableError` become bounded actionable tool errors
with their classification/code. Technical execution errors are opaque operation
failures without profile data. The profile path does not reuse the old SQL tool's
partial-result handling.

Count accepts an explicit `currencyField` meaning reference paired with
`nativeCurrency`; only `commerce.order_completed.currency` revision 1 for
`Order Completed` is resolved. Its governed reading is the same as Sum's currency
reading, including UNKNOWN for missing/null/non-String values. Independent Record
`where` is allowed except on paths overlapping that currency reading; dependent
forms return `FilterCapabilityNotImplemented`. Explanations include the reference
and currency only when supplied; existing raw Count documents/results are unchanged.

Count.where also accepts one governed money leaf, with the same reading and seven
comparisons as experience.verify. Independent Record predicates may be
combined with AND/OR; overlapping raw paths and repeated money leaves remain
unavailable. Explanations add count.money with paths relative to where and the
requested/normalized tests. An outer Count currency remains an additional
requirement: under OR it cannot be dropped merely because money also reads a
currency. The guide uses money alone for the simple individual-order threshold.
Sum/Average.where and experience.base.where remain Record-only.
Money equality and difference are exact; is not negates only the amount test.
Intervals are inclusive and normalized inward. Explanations preserve existing
ordered comparisons, while new degenerate tests use readableAmounts none/all;
both retain UNKNOWN for unreadable amounts and the separate currency requirement.
Sum and Average retain their four ordered terminals.

First/Last/Nth accept the same optional pair in `experience.base`, restricting the
population before selection. A selected candidate satisfies that base currency
even when its stored currency is unknown. Money verification on the same resolved
reading tests only the amount for the same currency and is FALSE for a different
currency. Other readings keep their full tests. Explanations distinguish base
`currencyField`/`nativeCurrency` from `money[].nativeCurrency` in the verify.
Nth selects one message at position 1–10 from the start of the base. Its period
does not test recency of the historical Nth; use full history plus occurredIn
for that request. Explanations retain kind nth and position, including Nth(1).
Raw overlaps with the base currency are unavailable in both base.where and verify;
the amount remains independent unless money reads it. Old experiences omit the
new fields. Snowflake retains its unverified-backend gate.

Count supports track (the default), page and screen. Page/screen may omit the
name to count all messages of that type. A named source certainly excludes
unnamed messages; a raw `where` on the optional name retains Record NULL.
Path/URL comparisons retain their literal Record semantics without normalization
or implicit fallback between context and properties. Sum and governed currency
remain restricted to track Order Completed, including through internal entry points.

No Admin filter editor, E/K/P/T, persistence or destination execution is introduced. Coherent
source views remain an infrastructure prerequisite, not a guarantee established
by this MCP adapter.

Average uses the same monetary source/reading as Sum, with mandatory minimum,
four ordered comparisons and effective threshold scale <=4. It compares the
exact sum to threshold times count and returns exact input numbers in its
explanations; it never divides or reports a rounded average. Unknown population
or amounts retain UNKNOWN, subject to the independent Kleene sample guard.
It excludes other native currencies without conversion or business deduplication.

Latest N uses its own `latestN` rule with the same experience base, size 1–10,
a uniform Record/money predicate and an explicit All, Some or canonical
Cardinality quantifier. Cardinality uses is (exactly), >=, <= and inclusive
between, with natural JSON operands and complete static validation.
Compatible whole blocks, including uncertain membership and timestamp ties,
are closed together. A short block never satisfies the rule. Explanations
retain the requested size/quantifier, candidate union and possible shortage;
natural operands remain plain digits (-0 becomes 0), including redundant upper
bounds. Internal reachable-count extrema are not exposed. Uniqueness concerns
the selected set, not the internal order of tied messages.
There is no occurredIn in this block predicate; trailing-window existence can
use Count with the same population. See profile-filter-doc for the distinction
from Count of passing messages and the two meanings of currency in the base
versus in the predicate. The Snowflake gate remains closed.
