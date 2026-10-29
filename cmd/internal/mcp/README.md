# Profile queries through MCP

`query-profiles` calls `Workspace.QueryProfiles`, with the same Record/Count
document and execution path as `GET /v1/profiles`. `profile-filter-doc` exposes
the supported syntax, examples and limitations to the model. The existing SQL
tool remains available for discovery and other analysis. Server instructions
direct profile/segment questions to Unified and prohibit SQL fallback for
unavailable filtering capabilities.

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
Count preserves the unverified-backend gate and returns an actionable capability
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

The argument limit is checked after the SDK has read and decoded the HTTP body;
it is not an HTTP ingress limit. Attribute byte checks follow each call to
`types.Marshal`, so the response budget does not bound the allocation used to
encode one profile's attributes. The legacy Record page is already loaded before
that step.

Record attributes use `types.Marshal`, preserving Decimal, Int64 and raw JSON
encoding. Count returns only MATCH profiles, plus `evaluation` with all three
totals and T. Empty pages use `profiles: []` and retain totals. Reusing `at`
anchors relative periods; it does not freeze the data between calls. Only Count
gets its existing deterministic tie-break, not a new Record sorting contract.

`BadRequestError` and `UnprocessableError` become bounded actionable tool errors
with their classification/code. Technical execution errors are opaque operation
failures without profile data. The profile path does not reuse the old SQL tool's
partial-result handling.

No Admin filter editor, event catalog, calendar constructor, governed public
binding, E/K/P/T, persistence or destination execution is introduced. Coherent
source views remain an infrastructure prerequisite, not a guarantee established
by this MCP adapter.
