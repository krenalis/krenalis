# Internal Unified Record and Count vertical

This package implements the first backend subset of the B architecture consolidated
in `filtering/14-confronto-architetture-interrogazione-unified-fase-4b.md`. The
normative baseline remains 06/07/08 version 1.2. This is an internal Go entry;
Admin, API and destination connections do not expose it yet. The prepared-filter
boundary refines the SQL transport described in report 14 §5.4: generated SQL is
now an internal detail of the warehouse implementation.

## Executable path

`Workspace.unifiedProfiles` now receives one `core.Filter`, decoded using the
public Record/Count JSON contract below. The temporary private `unifiedRule` and
`unifiedCount` input trees have been removed. Profile predicates reuse
`prepareFilter` with the workspace Profile schema; Count qualification uses the
Event schema. Every prepared profile condition becomes one Record leaf in the
common canonical AND/OR tree. The temporary AND wrapper used to validate a
single Record condition does not enter that tree. Explicit groups, including
singletons, retain their four-level/100-descendant budget. T and pagination
remain an internal operation request, separate from the filter document.
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

`RowQuery`, `Expr`, `FilterCondition`, `state.Where`, the Record JSON shapes
and existing equality remain unchanged. `core.FilterRule` additionally accepts
`FilterCountCondition`; the shared root `Filter` shape is preserved. The prepared filter has no public
wire or persistence format, relational IR, optimizer or generic expression
language. `RecordFilter` reads only the profile population row;
`RecordQualification` reads only an occurrence row. Neither can express a
property-reference comparison or a governed current-profile dependency.

## Admitted subset and evidence contract

- Profile Record conditions and Count share the root AND/OR tree: at most four
  groups including the root and 100 descendant nodes excluding the root. A Record
  subtree is expanded into these same groups and leaves, not an opaque body with
  a separate root budget. Record conditions and Count can be siblings in the
  deepest fourth group, or 99 Record leaves plus one Count in the root group.
  Passing a complete `Filter` deliberately retains its own explicit root group;
  it contributes a node and level like any nested input group. Nil root is
  AllProfiles.
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
- Explicit absolute `[Start,End)` periods with `Start < End <= T`, or elapsed
  periods resolved by core against that same T. Internal `Scope.Elapsed` uses
  `Older=d, Newer=0` for `ElapsedLookback(d)` and `Older=a, Newer=b` for
  `ElapsedAgoBand(a,b)`: both produce `[T-Older,T-Newer)`, with `Older > Newer >= 0`.
  Offsets are elapsed `time.Duration` values, not calendar days. Nanosecond
  precision and bounds outside the existing timestamp range are unsupported;
  no rounding or clipping is applied. This internal representation does not
  define public or persisted constructor payloads. The prepared warehouse
  filter still receives only absolute UTC bounds, exact to microseconds. Event
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
and AvailableHistory bindings, future/unmatured windows (06 §17.4), calendar
constructors, personal anchors, unknown timestamps/attribution/grain,
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

The datastore validates the prepared filter and request before calling the
backend, and returns these backend-independent admission errors directly.
Exceeding the physical-node budget is not a warehouse availability failure or a
semantic UNKNOWN. Drivers still validate their own arguments. SQL rendering,
transport and execution limits remain operation errors; no public error format
is defined by this internal entry.

Any query, scan, final `Rows.Err`, cancellation or Close failure returns an error
and a nil result. There is no successful partial page/count and no error-to-
UNKNOWN conversion. Cancelled and failed operations close their cursor, and the
store's mode operation remains held until decoding and Close complete.

The caller supplies T once. The query assumes coherent Profiles, event facts
and profile attribution for that operation. Identity Resolution/publication,
asynchronous ingestion/KPID attribution and destination matching remain separate
deferred work. Fixture views are published before the tests and not concurrently
modified. Neither T nor UUID ordering freezes a view across later requests.

## Public Count contract and persistence boundary

`FilterCountCondition` adds a `count` subject alongside the existing
`property` conditions and `rules` groups. Each rule has exactly one of these
three discriminators. A filter root remains a group. For example, profiles in
Italy with no recorded Purchase in the previous 30 elapsed days:

```json
{
  "operator": "and",
  "rules": [
    {"property": "country", "operator": "is", "values": ["IT"]},
    {
      "count": {
        "source": {"kind": "recordedEvents", "name": "Purchase", "grain": "registered-message"},
        "period": {"kind": "elapsedLookback", "duration": {"value": 30, "unit": "day"}}
      },
      "operator": "is",
      "values": [0]
    }
  ]
}
```

- Source kind is `recordedEvents`; `name` is a raw event name. `grain` is
  mandatory: this increment executes only `registered-message`. No source
  ontology or governed currency definition is inferred from raw names/paths.
- `operator` admits exactly `is`, `is not`, `is less than`,
  `is less than or equal to`, `is greater than`, `is greater than or equal to`,
  and `is between`. There are no Count membership, data-state or generic NOT
  operators. `is between` is inclusive and takes two values; the others take one.
- `values` contains nonnegative base-10 integer JSON numbers, following the
  Record Int spelling: no strings, decimals or exponent notation. Negative zero
  denotes zero. Numbers above 2^53 retain their exact digits. Out-of-range
  natural literals are unsupported execution capabilities, not a permanent
  int64 definition of the language; no conversion through Float is used.
- Optional `count.minimum` is a positive integer sample guard, independent of
  the comparison values. Omission means no extra sample minimum.
- Optional `count.where` reuses Record predicates against the Event schema.
  Its budget is two group levels and 100 descendants, independent of the main
  filter's four levels and 100 descendants. Count inside this body is rejected
  during decoding as well as programmatic Record preparation.
- `period.kind` is `absoluteHalfOpen` with `start`/`end` RFC 3339 instants,
  `elapsedLookback` with `duration`, or `elapsedAgoBand` with `older`/`newer`.
  Fields of another variant and null temporal fields are rejected. A duration
  contains an integer `value` and a fixed `unit`: `microsecond`, `millisecond`,
  `second`, `minute`, `hour`, `day` (24 hours), or `week` (7 days).
  Values are multiplied with checked integer arithmetic; months/years are not
  elapsed units. Relative periods retain their kind, offsets and units on the
  wire instead of becoming bounds resolved against one T.

Encoding/decoding is distinct from semantic validation, as for existing Record
conditions. Unknown/duplicate members and ambiguous shapes are rejected;
operator/type/cardinality/source/period checks occur during preparation and
canonical admission. Invalidity takes precedence over unavailable capabilities
throughout the document, independently of rule order. Natural operands, sample
minimums and elapsed offsets are checked exactly before execution range limits,
including inverted intervals beyond uint64. Absolute instants with more than
nine fractional digits are rejected before decoding into time.Time; admitted
nanoseconds retain the existing microsecond execution capability check.
The numeric limits of the executable backend remain capability errors.
An older decoder rejects the unknown `count` member.

The existing Record-only preparer explicitly rejects Count. This increment
connects the public document to the internal Unified service; it does not
activate Unified in the existing Profiles HTTP handler, Admin editor or
export pipeline. No new public result/UNKNOWN diagnostic contract is implied.

The application currently persists pipeline Record filters as `state.Where`,
using the existing schema-aware codec and equality. That format, its database
rows, notifications and reload/change detection are unchanged. There is no
saved Profiles segment store in the current schema. This increment provides
the public document's encode/decode round-trip, not a new application storage
format or migration for mixed filters. Such persistence must preserve explicit
grain, relative periods, applicable future meaning revisions and public grouping;
it must not save resolved T bounds or `warehouses.Filter`. No format/version for
that internal storage is chosen here. The canonical/internal `Rule` and
`Bindings` remain execution inputs rather than serialization contracts.
JSONB retains natural numeric values but normalizes member order and some
number spellings, including `-0` and exponent notation. Syntax and semantic
validation must precede any future save; byte-identical storage is not implied.
Unknown nonempty grain names and the treatment of null members still need an
explicit storage contract before persisting valid-but-unavailable documents;
this change preserves their existing admission behavior.

## Verification and activation

`TestUnifiedWorkspacePostgreSQL` decodes the new public Count document and
executes the complete service/datastore/warehouse path against a disposable
application database and warehouse schema, for all seven Count comparisons,
sample minimum, absolute/lookback/ago-band periods, profile Record and mixed
AND/OR. Its JSONB reload checks execution after normalization, not byte identity.
The `TestFilterCount*` tests check strict syntax, exact numeric literals,
Record-context rejection and independent main/occurrence budgets. Core tests preserve Record round-trip/equality and exercise the combined
semantic depth/node boundaries after expanding the Record tree, including
Record/Count siblings at four levels and 100 descendants. Tests for a complete
Profile filter retain the extra explicit group; their 98-leaf boundary is not
the limit for a flat mixed tree.

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

`TestUnifiedWritersPostgreSQL` starts from schema-decoded inputs and uses the
real batch/event identity writers and EventWriter, their flattening and
asynchronous flushers, PostgreSQL COPY/merge, and a completed Identity Resolution
publication/attribution before querying Unified. Eighteen profiles from two
connections cover no events, missing JSON, JSON null, empty objects, nested
values, exact JSON integers above 2^53, ordinary String values, raw/governed
currency and mixed Record/Count conditions. Missing and JSON null remain
distinct on profiles and events; an empty flattened object has no present
descendant, as in the existing Profiles layout. Separate event queries require
one Purchase per event fixture, distinguish root `properties` JSON null from
missing, and compare the exact JSON integer. This fixture exercises `user_id`
attribution only; anonymous-ID attribution remains untested. Additional elapsed
queries use Rome instants around both DST changes, fresh events, drained writes
and completed attribution before reading. They exercise both half-open borders,
exclusion of events at/after T, real timestamp(3) rounding from submillisecond
input, and microsecond start/end parameters. They also check governed UNKNOWN,
profiles without matching events, one T across composed periods, UTC result T
and empty membership pages with totals. The fixture has no concurrent writes
or Identity Resolution; it does not establish a concurrent read guarantee.

`TestUnifiedPrepareElapsedPeriods` checks backend-independent UTC bounds through
the real Record binder, including spring/autumn DST, microsecond windows,
the largest microsecond `time.Duration`, dates outside the Unix-nanosecond range,
and unchanged input. `TestValidateElapsedPeriod` distinguishes invalid widths,
offset order and conflicting period forms from unsupported precision/ranges.
The internal Workspace fixture also executes both relative forms through the
service with existing public Record qualification and the same ternary outcomes.

`TestUnifiedPreparedBudget` builds a semantically valid 99-condition tree on a
170-field object. Its deterministic physical-budget error is returned before
warehouse execution and is not classified as `UnavailableError`.

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
common runtime fixtures must pass before changing that gate. PostgreSQL writer
round-trips are verified by the fixture above; Snowflake writer/CSV round-trips
remain unverified. Snowflake activation also requires
resolving the declared ordinary-text collation and Float/NaN questions.

Example targeted commands, with Docker available and dependencies vendored:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off KRENALIS_SKIP_SNOWFLAKE_TESTS=true \
go test -mod=vendor ./core ./core/internal/datastore \
  -run '^(TestUnifiedWorkspacePostgreSQL|TestUnifiedProfiles.*|TestUnifiedSourcePostgreSQL|TestUnifiedProfileRecordPostgreSQL|TestUnifiedWritersPostgreSQL)$' -count=1

GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
go test -mod=vendor ./core ./core/internal/unified ./core/internal/datastore \
  ./warehouses ./warehouses/internal/profilefilter ./warehouses/postgresql ./warehouses/snowflake \
  -run '^(TestPrepareUnified.*|TestRead|TestValidate.*|TestUnifiedPrepare.*|TestAcceptedCountRanges|TestValidateProfileFilter|TestProfileFilterDialect|TestProfileFilterSQL|TestProfileRecordSQL|TestStatementValues)$' -count=1
```

API/Admin/destination activation additionally requires a decided and verified
public/persisted representation, real source-meaning bindings and admission,
required warning/diagnostic consumption, and the coherent-source guarantees of
the respective release. This change implements a backend vertical, not all 1.2.
