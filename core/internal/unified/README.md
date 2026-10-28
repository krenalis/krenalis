# Internal Unified Record and Count vertical

This package implements the first backend subset of the B architecture consolidated
in `filtering/14-confronto-architetture-interrogazione-unified-fase-4b.md`. The
normative baseline remains 06/07/08 version 1.2. This is an internal Go entry;
Admin, API and destination connections do not expose it yet. The prepared-filter
boundary refines the SQL transport described in report 14 §5.4: generated SQL is
now an internal detail of the warehouse implementation.

## Executable path

`Workspace.unifiedProfiles` prepares its internal recursive request using
`prepareFilter`: profile predicates use the workspace Profile schema; Count
qualification uses the Event schema. Every prepared profile condition becomes
one Record leaf in the common canonical AND/OR tree. Groups, including singleton
groups, remain visible to its four-level/100-descendant validation. The request
is a Go service input, not a second public filter or persistence format.
`datastore.Store.UnifiedProfiles` fixes the evaluation input T, validates the
supported canonical shape and calls `unified.Prepare` with resolved source
metadata and the existing `convertWhere` Record binders for the two subjects.

`Prepare` returns `warehouses.Filter`: profile Record leaves, Count sources,
mature absolute periods, local qualifications and normalized accepted integer
ranges. It produces no SQL and receives no dialect callbacks. T remains in core because maturity and all
periods have already been resolved before the warehouse call.

Both drivers implement `ProfileFilterWarehouse.ProfileOutcomes`. They use the
shared physical composer and decoder in `warehouses/internal/profilefilter`,
with their existing Record renderer, pool and normalized scanner. SQL, parameter
positions, CTEs and the metadata-row protocol stay below the warehouse boundary.
Adding a backend does not require backend-specific branches in core.

`RowQuery`, `Expr`, `core.Filter`, `FilterCondition`, `state.Where`, their JSON
representations and equality are unchanged. The prepared filter has no public
wire or persistence format, relational IR, optimizer or generic expression
language. `RecordFilter` reads only the profile population row;
`RecordQualification` reads only an occurrence row. Neither can express a
property-reference comparison or a governed current-profile dependency.

## Admitted subset and evidence contract

- Profile Record conditions and Count share the root AND/OR tree: at most four
  groups including the root and 100 descendant nodes excluding the root. A Record
  subtree is expanded into these same groups and leaves, not an opaque body with
  a separate root budget. Nil root is AllProfiles.
- Profile Record reuses the existing 26 operators, schema validation, typed
  literals and paths. SQL NULL becomes UNKNOWN before outer composition; the
  six explicit data-state operators keep their original results. Ordinary
  profile properties are not Nullable in the schema: `is null` remains invalid
  for them, unlike a JSON path. Missing ordinary values remain SQL NULL.
- For the current flattened SQL profile layout, object `exists` is TRUE when
  at least one descendant column is not SQL NULL; `does not exist` is its
  two-valued complement. A JSON null in a descendant column is present. The
  existing binder supplies this interpretation, also used for event objects.
  It cannot distinguish an unmaterialized empty object from an absent object;
  no equivalence with an original Go object representation is certified.
- `Direct(Count(scope), literal test)`: equality, inequality, ordered comparisons
  and inclusive integer intervals; optional occurrence sample minimum. Absence
  is `Count = 0`, existence is `Count >= 1`. There are no new public operators.
- The physical source is `krenalis_events`, with known action timestamp, determined
  `message_id` occurrence identity and determined `kpid` attribution. The grain
  is **registered-message**, not business Order identity. Different message IDs
  with identical payloads remain different occurrences. Duplicate source-view
  join rows with the same identity and relevant state are deduplicated before
  counting. Source conflicts are execution errors under this determined-identity
  contract, not a general policy for semantic identity uncertainty.
- The raw event-name equality is a local nullable qualifier. An unknown name is
  retained as possible evidence; a certainly different name is excluded.
- Explicit absolute `[Start,End)` periods with `Start < End <= T`. All bounds
  are unambiguous instants, represented exactly to microseconds in UTC. Event
  timestamps remain the values actually stored, including the current schema's
  millisecond precision; they are not reconstructed from the original input.
- Prepared Record qualification on Event properties, with the existing 26
  operators, literal payloads and schema-aware preparation. Local groups have
  the occurrence-body limit of two levels; this does not restrict existing root
  Record filters. The binder retains JSON key segments, typed JSON mode and
  flattened object-presence handling. No filter SQL comes from users.
- Alternatively, one governed `NativeCurrency = literal` predicate. The Count
  contains an immutable meaning reference and the currency literal; separate
  `Bindings.Currencies` resolve the exact reference/revision to its JSON path
  and evidence contract. Missing, JSON null and non-String values are unknown
  business currency. Known Strings are compared verbatim: `"eur"` and
  `"not-a-code"` are known values different from `"EUR"`. No domain attestation,
  normalization or `KnownCanonicalStrings` flag is introduced. A future closed
  domain requires a distinct resolved reading contract. Bindings are supplied by an internal caller; this change does not
  implement a registry, infer a business meaning from raw JSON, or automatically
  adopt meaning revisions. Missing or duplicate bindings fail before execution.

The supported governed contract is `IndependentOccurrenceValues`: an unknown
value is local to one occurrence, admits both qualifying and non-qualifying
completions, and has no declared cross-occurrence constraint. Raw Record
predicates have their existing local ternary contract; SQL NULL alone creates
neither a governed fact nor a cross-occurrence dependency. Combining raw and
governed qualification in the same Count is currently unsupported.

For a determined deduplicated population, let S be the number of TRUE qualifiers
and U the number of NULL qualifiers. With the admitted independent memberships,
every integer in `[S,S+U]` is reachable: choose any k of the U occurrences for
each k between 0 and U. Core normalizes the test and sample minimum into at
most two sorted, disjoint, non-adjacent accepted integer ranges. The physical
helper compares the reachable integer interval with that accepted set. This proof does not apply to E, K, shares or arbitrary
correlated bases. It does not treat an arbitrary real interval as reachable.

`SharedBoolean` records a governed `EqualCurrentProfile` dependency explicitly
outside this implementation. With two known occurrence flags both true and one
shared unknown g, the counts are `{0,2}`; with flags true/false the same equality
produces count 1 in every completion (g / its typed complement). Both inputs are
rejected with `*UnsupportedError` before SQL execution. They are not declared
invalid under 1.2 and do not yield UNKNOWN.

The following also remain outside this vertical: governed FullRelevantHistory
and AvailableHistory bindings, future/unmatured windows (06 §17.4), elapsed and
calendar constructors, personal anchors, unknown timestamps/attribution/grain,
business deduplication, source unions, property-reference terminals, other
governed predicates, owned-member traversal, E/K/P/T, sums and shares. Count
literal bounds above signed 64-bit range are unsupported. Structurally invalid
input remains an error, distinct from unsupported valid-language capability.

## Results, resources and consistency

One statement computes the profile outcomes, all three profile totals and a
deterministically ordered UUID page from the same evaluation. The Profiles
platea is the left input; an empty event population yields Count zero. A join
placeholder never becomes an unknown occurrence.

`Page.MatchingOnly` selects the MATCH membership page. When false it requests an
internal outcome page with MATCH/NO_MATCH/UNKNOWN; this is not an include-UNKNOWN
segment policy or a public diagnostic API. The totals always refer to the full
profile population, independently of the page and its projection. Empty pages
have a metadata-only row, explicitly distinguished from a profile.

The population projection reads only its profile table, before any Count join.
Record leaves are projected without filtering or coalescing their NULL results.
Its internal identity and leaf aliases contain `$`, which profile property names
cannot contain, including on dialects supporting lateral SELECT aliases. The
selected identity is joined to Count results only after this projection.

Profile totals use checked signed 64-bit values. Pages contain at most 1,000
profiles; offset is bounded to signed 32-bit range. Output UUID text is bounded
at the query source and validated by the existing scanner and decoder. Generated
statements are limited to 1 MiB, 100 columns and 1,000 typed arguments. The entire
prepared filter shares a 16,384-node physical expression budget, distinct from
its 100 semantic descendants: object presence may expand into many columns. The
internal argument adapter admits text and UTC DateTime; it never uses float for
Count bounds or existing Record Decimal/JSON operands.

Any query, scan, final `Rows.Err`, cancellation or Close failure returns an error
and a nil result. There is no successful partial page/count and no error-to-
UNKNOWN conversion. Cancelled and failed operations close their cursor, and the
store's mode operation remains held until decoding and Close complete.

The caller supplies T once. The query assumes coherent Profiles, event facts
and profile attribution for that operation. Identity Resolution/publication,
asynchronous ingestion/KPID attribution and destination matching remain separate
deferred work. Fixture views are published before the tests and not concurrently
modified. Neither T nor UUID ordering freezes a view across later requests.

## Public/persisted format still to decide

The following existing public Record input retains its current round-trip:

```json
{"operator":"and","rules":[{"property":"properties.currency","operator":"is","values":["EUR"]}]}
```

Within `core`, a programmatic request can reuse it without defining a new wire tag:

```go
result, err := workspace.unifiedProfiles(ctx, unifiedQuery{
    Root: &unifiedRule{Operator: OpAnd, Rules: []unifiedRule{
        {Profile: profileFilter},
        {Count: &unifiedCount{Scope: scope, Where: recordFilter,
            Test: unified.CountTest{Operator: unified.Equal, Value: 0}}},
    }},
    At: at, Page: unified.Page{Limit: 50, MatchingOnly: true},
})
```

Both `profileFilter` and `recordFilter` use existing public Record JSON and are
prepared against their respective schemas inside the service. That JSON does
not express the Count, mixed tree, period or result request. A concrete unresolved
exposure is how to embed this Count-of-Purchase, with that exact Record predicate and Count=0, as a new
`core.FilterRule`, and how to load/save it without changing existing
`FilterCondition` payloads or `state.Where.Equal`. No Count JSON tag, marshaler,
database format or migration is chosen here. `TestPrepareUnifiedRecordCount`
checks public JSON encode/decode and the prepared Record equality independently
of that decision. Internal Rule/Bindings structs are not a serialization contract.

## Verification and activation

`TestUnifiedWorkspacePostgreSQL` decodes existing public Record JSON and
executes the complete service/datastore/warehouse path against a disposable
application database and warehouse schema, for Count, profile Record and mixed
AND/OR. Core tests preserve Record round-trip/equality and exercise the combined
semantic depth/node boundaries after expanding the Record tree.

`TestUnifiedProfilesPostgreSQL` exercises the real Go datastore/compiler/Record
renderer/compiled transport/scanner/decoder against PostgreSQL 18.6. It covers
raw and governed currency, empty population/history within the selected period,
multiple profiles, duplicate joins, identical-payload distinct occurrences,
Kleene composition, name uncertainty, microsecond bounds against timestamp(3),
membership and empty pages, a separate exhaustive completion oracle for 27
three-occurrence histories, reused typed JSON fixtures including literal dotted
keys, shared-fact rejection, SQL/scan errors and in-flight cancellation.

`TestUnifiedProfileRecordPostgreSQL` checks all nine Record/Count Kleene pairs,
MATCH pages and empty-page totals, and compares pure Record membership with the
existing `store.Profiles` listing. It covers JSON null/missing/negative/state
operators, exact Decimal and JSON numbers, literal JSON key segments, profile
properties named `profile_id` and `event`, and deepest object-presence leaves.
An all-FALSE Record does not mask a Count source error. Pure Record still runs
without an event table; empty populations preserve the totals protocol.

`TestUnifiedSourcePostgreSQL` uses a separate source without a primary key to
exercise duplicate identities, conflicting qualification, attribution (including
NULL/non-NULL), copies across the period boundary and irrelevant differences.
It also checks NativeCurrency Boolean/Number/Object/Array and verbatim Strings.
`TestNumericPostgreSQL` executes the real decision SQL at int64 boundaries and
checks that the current aggregate expressions have bigint result metadata.
These probes do not demonstrate Snowflake runtime behavior or production scale.

`warehouses/internal/profilefilter.TestRead` separately injects Scan, final Err and Close failures, malformed
metadata, duplicate/oversize pages, bad output types, overflow and cancellation.
Adapter tests on both packages reuse `warehouseexpr.ValueCases` and `JSONCases`
and verify coverage of all 26 operators against each original renderer. They
also check typed binds, schema errors and malformed prepared expressions, and
isolate profile Record rendering from event qualifications with collision-free
aliases. The source binders and population request share one ProfileSource.

The service schema and datastore column map follow the existing Profiles
binding path. Concurrent schema publication is not newly coordinated here;
coherence between those views remains an integration assumption to verify.

PostgreSQL is enabled for this internal subset. Snowflake has the mirrored
physical adapter and transport implementation, shared-compiler/local tests and
an explicit **inactive Unified gate**. No Snowflake credentials were configured
for this work; no remote query, scanner, precision/overflow, timestamp, NULL,
subquery or count/page conformance is claimed. Real Snowflake probes and the
common runtime fixtures must pass before changing that gate.

Example targeted commands, with Docker available and dependencies vendored:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off KRENALIS_SKIP_SNOWFLAKE_TESTS=true \
go test -mod=vendor ./core ./core/internal/datastore \
  -run '^(TestUnifiedWorkspacePostgreSQL|TestUnifiedProfiles.*|TestUnifiedSourcePostgreSQL|TestUnifiedProfileRecordPostgreSQL)$' -count=1

GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
go test -mod=vendor ./core ./core/internal/unified ./core/internal/datastore \
  ./warehouses ./warehouses/internal/profilefilter ./warehouses/postgresql ./warehouses/snowflake \
  -run '^(TestPrepareUnified.*|TestRead|TestValidate.*|TestUnifiedPrepare.*|TestAcceptedCountRanges|TestValidateProfileFilter|TestProfileFilterDialect|TestProfileFilterSQL|TestProfileRecordSQL|TestStatementValues)$' -count=1
```

API/Admin/destination activation additionally requires a decided and verified
public/persisted representation, real source-meaning bindings and admission,
required warning/diagnostic consumption, and the coherent-source guarantees of
the respective release. This change implements a backend vertical, not all 1.2.
