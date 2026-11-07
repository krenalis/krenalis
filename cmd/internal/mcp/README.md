# Profile queries through MCP

`query-profiles` calls `Workspace.QueryProfiles`, with the same Record/Count/Sum
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
arrays, sums, governed bindings and generic page/screen or identity definitions.
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
Count and Sum preserve the unverified-backend gate and return an actionable capability
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
encoding. Count and Sum return only MATCH profiles, plus `evaluation` with all three
totals and T. Empty pages use `profiles: []` and retain totals. Reusing `at`
anchors relative periods; it does not freeze the data between calls. Count and Sum
retain their deterministic tie-break, not a new Record sorting contract.

`BadRequestError` and `UnprocessableError` become bounded actionable tool errors
with their classification/code. Technical execution errors are opaque operation
failures without profile data. The profile path does not reuse the old SQL tool's
partial-result handling.

No Admin filter editor, public Count currency
binding, E/K/P/T, persistence or destination execution is introduced. Coherent
source views remain an infrastructure prerequisite, not a guarantee established
by this MCP adapter.
