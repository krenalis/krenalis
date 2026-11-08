# Internal Unified Record, Count, Sum, Average and Experience vertical

This package implements the first backend subset of the B architecture consolidated
in `filtering/14-confronto-architetture-interrogazione-unified-fase-4b.md`. The
normative baseline remains 06/07/08 version 1.2. Count, Sum, Average and Experience are exposed by the existing
Profiles HTTP endpoint and MCP `query-profiles` tool on the verified backend;
Admin and destination connections do not expose it yet. The prepared-filter
boundary refines the SQL transport described in report 14 §5.4: generated SQL is
now an internal detail of the warehouse implementation.

## Executable path

`Workspace.QueryProfiles` receives one `core.Filter`, decoded using the
public Record/Count/Sum/Average/Experience JSON contract below. The temporary private `unifiedRule` and
`unifiedCount` input trees have been removed. Profile predicates reuse
`prepareFilter` with the workspace Profile schema; Count qualification uses the
Event schema. Every prepared profile condition becomes one Record leaf in the
common canonical AND/OR tree. The temporary AND wrapper used to validate a
single Record condition does not enter that tree. Explicit groups, including
singletons, retain their four-level/100-descendant budget. T and pagination
remain operation parameters, separate from the filter document.
`datastore.Store.UnifiedProfiles` fixes the evaluation input T, validates the
supported canonical shape and calls `unified.Prepare` with resolved source
metadata and the existing `convertWhere` Record binders for the two subjects.

`Prepare` returns `warehouses.Filter`: profile Record leaves, Count sources,
mature periods with an optional lower bound, local qualifications and normalized accepted integer
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
- Track event-name equality retains its local nullable qualifier. Page/screen
  sources select their type and optionally the recorded `name`. Named sources
  require presence and equality, so an unnamed message is certainly excluded;
  sources without a name include all messages of that type. A raw `where` on
  `name` still retains NULL. The internal Scope carries the type; validation
  and preparation reject types outside track/page/screen, and both monetary
  consumers reject non-track sources even when named Order Completed.
- Explicit absolute `[Start,End)` periods with `Start < End <= T`, or elapsed
  periods resolved by core against that same T. Internal `Scope.Elapsed` uses
  `Older=d, Newer=0` for `ElapsedLookback(d)` and `Older=a, Newer=b` for
  `ElapsedAgoBand(a,b)`: both produce `[T-Older,T-Newer)`, with `Older > Newer >= 0`.
  Offsets are elapsed `time.Duration` values, not calendar days. Nanosecond
  precision and bounds outside the existing timestamp range are unsupported;
  no rounding or clipping is applied. This internal representation does not
  define public or persisted constructor payloads. The prepared warehouse
  filter receives resolved absolute UTC bounds, exact to microseconds. Event
  timestamps remain the values actually stored, including the current schema's
  millisecond precision; they are not reconstructed from the original input.
- `FullRelevantHistory` resolves the explicit built-in history reference
  `registered-queryable-events` at immutable meaning revision `1`. Its domain
  is all retained, queryable recorded events in the coherent query view, before
  T exclusively. It has no lower bound, relationship-start certificate or
  assertion about deleted or never-recorded events. Retention can therefore
  change counts and absence. This new definition adopts the 1.2 recorded-history
  meaning explicitly; no old meaning revision is reinterpreted. Core resolves
  the descriptor and passes `UnboundedStart` with a zero `Start` and resolved UTC
  `End=T` to the warehouse. The shared compiler omits the lower predicate and
  parameter; it does not replace it with year 1, the first event or retention.
  This descriptor is currently resolved for the checked `recordedEvents` source.
  Any additional source/repertoire must verify compatibility with the history
  definition before adopting it.
- Prepared Record qualification on Event properties, with the existing 26
  operators, literal payloads and schema-aware preparation. Local groups have
  the occurrence-body limit of two levels; this does not restrict existing root
  Record filters. The binder retains JSON key segments, typed JSON mode and
  flattened object-presence handling. No filter SQL comes from users.
- Optionally, one governed `NativeCurrency = literal` predicate. The Count
  contains an immutable meaning reference and the currency literal; its resolved
  descriptor specifies the JSON path and evidence contract. Missing, JSON null
  and non-String values are unknown business currency. Known Strings are compared
  verbatim: `"eur"` and
  `"not-a-code"` are known values different from `"EUR"`. No domain attestation,
  normalization or `KnownCanonicalStrings` flag is introduced. A future closed
  domain requires a distinct resolved reading contract. The public
  `currencyField` reference resolves `commerce.order_completed.currency`, revision
  1, for `Order Completed`, with an immutable path/evidence descriptor. Sum's
  monetary definition obtains its currency path from the same definition.
  Currently only internal tests supply `Bindings.Currencies` explicitly;
  production entry points pass empty bindings and use resolved descriptors.
  Missing or duplicate caller-supplied bindings fail before execution. Neither
  path infers a business meaning from raw JSON or automatically adopts meaning revisions.

The supported governed contract is `IndependentOccurrenceValues`: an unknown
value is local to one occurrence, admits both qualifying and non-qualifying
completions, and has no declared cross-occurrence constraint. Raw Record
predicates have their existing local ternary contract; SQL NULL alone creates
neither a governed fact nor a cross-occurrence dependency. A resolved currency
predicate can combine with a Record `where` whose paths do not overlap the
currency reading. The check visits all groups and compares exact key segments:
`properties`, `properties.currency` and its descendants are unavailable, while
`properties.currency_code`, `properties.Currency` and `properties.total` remain
admissible. Count and Sum share that admission check but supply their own reading
paths; Sum also excludes its amount path. Predicates using caller-supplied currency bindings
still reject mixed qualification, since their path is unavailable to validation.

For a determined deduplicated population, let S be the number of TRUE qualifiers
and U the number of NULL qualifiers. With the admitted independent memberships,
every integer in `[S,S+U]` is reachable: choose any k of the U occurrences for
each k between 0 and U. Core normalizes the test into at most two sorted,
disjoint, non-adjacent accepted integer ranges and retains the sample minimum
separately. The physical helper compares the reachable integer interval with
each requirement and composes their ternary results using Kleene AND. Thus
`is not 3` with minimum 3 over [1,3] is UNKNOWN, not NO_MATCH. This proof does
not apply to E, K, shares or arbitrary correlated bases. It does not treat an
arbitrary real interval as reachable.

`SharedBoolean` records a governed `EqualCurrentProfile` dependency explicitly
outside this implementation. With two known occurrence flags both true and one
shared unknown g, the counts are `{0,2}`; with flags true/false the same equality
produces count 1 in every completion (g / its typed complement). Both inputs are
rejected with `*UnsupportedError` before SQL execution. They are not declared
invalid under 1.2 and do not yield UNKNOWN.

The following also remain outside this vertical: AvailableHistory bindings,
future/unmatured windows (06 §17.4), other calendar constructors, personal
anchors, unknown timestamps/attribution/grain,
business deduplication, source unions, property-reference terminals, other
governed predicates, owned-member traversal, other E forms and K/P/T, reductions other than Count/Sum/Average and shares. Count
literal bounds above signed 64-bit range are unsupported. Structurally invalid
input remains an error, distinct from unsupported valid-language capability.

## Monetary Sum subset

The public `sum` subject reuses the source, periods and `operator`/`values` shape
of Count. It requires `field:{definition:"commerce.order_completed.total",
meaningRevision:"1"}` and `nativeCurrency`, three uppercase letters. The immutable
reading binds track `Order Completed`, `properties.total` and `properties.currency`
at registered-message grain. Other source names are invalid. This is a sum of
reported totals in the selected native currency, not all-currency spend, settled
payments or distinct business orders. The event catalog does not certify old data.

Known amounts are JSON Numbers exactly representable as Decimal(18,4); missing,
null, wrong kinds and non-representable readings are unknown. No coercion,
rounding, implicit USD, non-negative sign or finite domain for unknown historical
amounts is inferred. Currency uses the same verbatim known-String contract as
Count. An unknown amount certainly in another currency contributes zero.

The execution subset has four ordered comparisons and one exact signed literal
within the existing global precision 76/scale 37 limits, bounded lexically to
4096 bytes. Sum equality/difference/intervals are explicitly
unsupported, not invalid or UNKNOWN. Sum is admitted only
through Profiles APIs/MCP and the internal path; Record pipeline behavior and
`state.Where` persistence remain unchanged.

Optional `minimum` is a positive integer sample guard, currently bounded to
MaxInt64. It counts qualified events, including those with unknown amounts.
Count and Sum retain this guard separately and share the same SQL composition:
the guard and terminal comparison close independently, then combine with
Kleene AND. An unreachable minimum is NO_MATCH even with unbounded amounts.
For one certain 50 EUR order and two amounts of 60 with unknown currency,
minimum 3 and sum<=100 is UNKNOWN, although no joint completion satisfies both.
Sum bounds remain unconditioned.

Optional Sum `where` uses the same schema-aware Record preparation, two group
levels and 100 descendant nodes as Count. The resolved monetary definition owns
the amount/currency paths and the exact amount type. Both reading and dependency
admission use those paths. To retain independent contributions and reachable
extrema, overlapping paths are temporarily unsupported: the ancestor `properties`,
the exact amount/currency paths, and their descendants, in every AND/OR branch.
Segments are case-sensitive; `properties.Total` and `properties.total_discount`
remain separate from `properties.total`. Full validity checking precedes the
capability error. The compiler combines the source selector, Record `where` and
native currency with AND before reading amounts; false qualification makes even
an unknown amount irrelevant. Copies must agree on the combined qualification.

Core resolves the reading and normalizes the terminal exactly on the 1e-4 grid.
Greater becomes an inclusive threshold at floor(t*10000)+1, GreaterOrEqual at
ceil(t*10000), and the symmetric formulas apply to upper bounds. Signed rational
arithmetic is used, never Float. With at most MaxInt64 identities and known
Decimal(18,4) contributions, every finite total is strictly within +/-1e33.
Clipping an already normalized threshold to those outer sentinels is equivalent
for finite totals and for unbounded unknown contributions; no threshold is
rounded by a driver. The warehouse receives an inclusive half-line and a
representable Decimal(38,4) bound, never the original 76-digit operand as Float
or DECFLOAT. Occurrence-count overflow or arithmetic overflow is a technical error.

For independent local qualification, a known certain contribution is fixed, and
an optional known contribution contributes either zero or its signed value.
Summing their respective minima and maxima gives reachable extrema. Intermediate
totals can be sparse, so this is not a reachability algorithm for equality.
Reachability here is relative to the independent ternary qualification outcomes
of each occurrence, for Count as well as Sum. Record groups retain Kleene logic;
constraints between leaves over the same missing field are not solved jointly.
Thus a missing coupon leaves both `coupon is PROMO AND coupon is not PROMO` and
the corresponding OR UNKNOWN. The qualification model can keep that event
optional even when reasoning over joint raw-field assignments would decide it.
A relevant unknown amount has no certified sign/bounds; both sides are unbounded.
No governed shared fact is inferred from SQL NULL. Duplicate identities must agree
on attribution, period, qualification and relevant amount reading; a known/missing
mixture or two different known amounts fails the operation rather than becoming
an optional value. All-unknown readings remain unknown, as in Count qualifications.

Listing still uses its existing single-statement page/totals protocol. Explanations
use the unchanged six-column protocol for filters without Sum, or two additional
bounded text columns for Sum bounds. The decoder bounds text before decimal
parsing, checks representability within 76/37 and lower<=upper for finite bounds,
and discards all results on scan/iteration/cancellation/Close failure. Public Sum
explanations always include `certain` and `uncertain` qualified counts, even
without a minimum, plus field/currency and the same temporal metadata as Count.
Sum rows reuse the existing integer count columns; decimal bounds retain their
two bounded text columns. Count and Sum counts are nonnegative and ordered.
Null `bounds.lower`/`upper` means no finite bound, never zero. Finite values use
the existing exact JSON-number serializer. References stay in core, not drivers.

PostgreSQL runtime tests include signed, optional and unknown contributions,
exact precision, duplicate conflicts, profiles without events, empty pages,
half-open time borders, cancellation, and an independent enumeration of sums for
27 bases against all four comparisons. HTTP/MCP fixtures traverse the public
codec, service, datastore, compiler, transport and decoder. Snowflake has mirrored
local adapter tests but remains inactive pending actual runtime verification,
including VARIANT numeric kinds, precision and overflow. Its current reading
accepts only fixed-point VARIANT numbers; DOUBLE representations must be resolved
and verified before activation. No remote conformance
or infrastructure guarantee is inferred from these local tests.

## Monetary Average subset

`average` shares Sum's strict subject codec and monetary configuration, with an
obligatory positive `minimum`. The aliases describe the same payload fields;
validation of the distinct Average condition enforces the required guard.
It has the same registered-message grain, native-currency restriction, governed
Decimal(18,4) reading, periods and independent where paths as Sum.

Only the four ordered literal comparisons are activated. Exact thresholds must
fit the global 76/37 domain and have effective scale <=4; finer values are valid
language but unimplemented. Core predecides the value test at thresholds outside
the open (-1e14,1e14) range of determined averages, without attributing such a
business bound to missing amounts. Otherwise it passes the exact threshold,
direction and strictness. With n<=MaxInt64, t*n fits Decimal(38,4).

The shared warehouse composer reuses the monetary population, copy checks and
count/sum statistics. It compares S to t*n, without AVG or division, only when
population is determined, nonempty and all relevant readings are known. The
baseline otherwise returns UNKNOWN for the value test; the independent minimum
guard composes using Kleene AND. This does not use Sum's reachable extrema as
average extrema or infer a mean from non-null values alone.

Explanations reuse the bounded numeric transport, identified by Average node
metadata, and expose exact sum/count inputs with certain/uncertain counts. No
quotient is serialized. Unknown population means count=null and sum=null;
known population with an unknown amount has an exact count and sum=null.
An empty population has sum=0/count=0 with a false mandatory guard.

PostgreSQL has executable datastore and HTTP/MCP fixtures; both dialects have
local compilation tests. Snowflake remains gated and unverified on a real server.

## Results, resources and consistency

### Single-profile explanations

`Workspace.ExplainProfileFilter` exposes a separate read-only operation through
`GET /v1/profiles/{kpid}/filter-explanation?filter=JSON&at=RFC3339` and MCP
`explain-profile-filter`. The filter is the same public Record/Count document;
neither membership selection nor listing responses change. The operation returns
every main-tree rule outcome for one KPID, in input order, including non-matches
and UNKNOWNs. It is not a listing of UNKNOWN profiles or a new selection policy.

Each Count includes its resolved UTC `[start,end)` period, `certain` qualifying
occurrences and `uncertain` qualifications. For full history, `start` is omitted
and `unboundedStart:true` makes the absent lower bound explicit; `end` is T.
The `history` object reports the resolved `definition` and `meaningRevision`
from the semantic model in core; these references are not warehouse concerns.
Existing bounded-period responses remain unchanged. Under this subset's
independent occurrence contract, the reachable count is every integer from `certain` through
`certain + uncertain`. For example, 1 certain and 1 uncertain makes Count >= 2
UNKNOWN. A parent OR may still MATCH because another child matches. Record
qualifications are closed before counting: this explanation does not identify
which JSON path or operand produced a NULL, infer a governed fact, or report
individual event payloads. It does not explain unsupported E forms or K/P/T constructs.

Core preserves the prepared shape and periods. The optional warehouse
`ProfileFilterExplanationWarehouse` capability uses the same Record renderers,
Count reductions and terminals as listing. One statement restricts the profile
population and returns at most one row per node, with a fixed six-column schema.
Count identity seeds include copies attributed to the target profile outside
the period too; all copies of seeded identities are checked for attribution,
period and qualification conflicts. Other profiles' filters are not evaluated.
Consequently, a target explanation can succeed while population listing fails
because of source-contract faults confined to other profiles. Success means
completion of that target's evaluation, not validation of the whole source.

The decoder bounds node rows to 101, checks profile identity, preorder indices,
nonnegative count bounds and source diagnostics, and always closes the cursor.
Cancellation, scan, iteration, source or Close errors discard the entire result.
A missing profile is NotFound. PostgreSQL is verified through the full Go path;
Snowflake has the same adapter and local rendering tests but remains gated.

An explanation is a new evaluation. Reusing listing's `evaluation.at` anchors
the periods but does not freeze the data or the runtime tzdb. Source coherence
is assumed, as for listing; the deferred infrastructure is not implemented here.

### Profile pages

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

- Source kind is `recordedEvents`; optional `type` defaults to track. Track name
  is mandatory and reads Event `event`. Page/screen name is optional and reads
  Event `name`, with a presence guard only for named sources. Explicit empty or
  null name/type fields are invalid; omission remains distinct. Preparation
  resolves the default in the internal Scope without altering the public JSON.
  `grain` is mandatory and admits only the built-in `registered-message` projection.
  Other names are invalid input in both the public preparer and internal
  model, not unavailable capabilities. A bare string cannot identify a
  governed grain definition and revision. Future governed references must
  distinguish unresolved definitions from known unavailable capabilities.
  No source ontology or governed currency definition is inferred from raw
  names/paths.
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

For complete calendar periods, the root Count filter declares its calendar:

```json
{
  "operator": "and",
  "rules": [{
    "count": {
      "source": {"kind": "recordedEvents", "name": "Order Completed", "grain": "registered-message"},
      "period": {"kind": "lastCompleteCalendarPeriods", "unit": "month", "count": 1}
    },
    "operator": "is greater than or equal to",
    "values": [2]
  }],
  "defaultCalendar": {"definition": "gregorian-monday/Europe/Rome", "meaningRevision": "1"}
}
```

`unit` admits `day`, `week` and `month`; `count` is a positive JSON integer.
They select one contiguous interval of the last N complete periods, excluding
the current one. Count still counts occurrences, not qualified periods. Core
resolves both UTC bounds against the operation's T before calling the warehouse;
days can be 23 or 25 hours. Weeks run Monday through Sunday.

Built-in calendar definitions use `gregorian-monday/TIMEZONE`, at immutable
meaning revision `1`. TIMEZONE is an explicit IANA name, including `UTC`;
`Local`, unknown definitions/revisions and implicit workspace calendars are not
admitted. Names and aliases must exactly match `calendar_zones.txt`, copied from
Go 1.27.2's `lib/time/zoneinfo.zip`, excluding the `Factory` placeholder. This
keeps name validation independent of the host's filesystem and local aliases.
Timezone offsets follow the runtime IANA database, with Go's embedded
database available as a fallback; the meaning revision does not pin a tzdb build.
No configurable registry or workspace setting is introduced.

`defaultCalendar` is optional for existing Count forms and mandatory for this
calendar form. It is allowed only on the main root and applies to every calendar
Count, including inside nested groups. Condition overrides and profile timezones
are not implemented. Profiles queries accept and validate the reference even
without Count, retaining the existing Record execution and response. Contexts
restricted to Record filtering reject it rather than dropping it through
`state.Where`. Relative periods and the explicit reference
survive encode/decode and preparation unchanged. There is no Unified persistence
format or materialized UTC period added by this increment.

Nonexistent or ambiguous midnight boundaries require an unavailable capability,
not a hidden DST choice or semantic UNKNOWN. For example, Havana's ambiguous
midnight on 2026-11-01 makes any window starting or ending at that boundary
unavailable, including October and November 2026 as single complete months.
Counts whose bounds exceed the timestamp domain also remain unavailable.
Missing decisive event values retain
the ordinary ternary rules. These changes add no SQL dialect features; Snowflake
retains its unverified-backend gate.

Required Count configuration members must be present and non-null. Omitting
`minimum` means no extra sample guard; `minimum:null` is invalid. Omitting
`where` or supplying `where:null` means no additional qualification, and the
encoder omits it. This is a filter-specific absence convention, not a general
default for optional members. Fields from another temporal variant are invalid
even when null. Count `values:null` and `values:[null]` are invalid; Record
literal JSON null and existing Record omission/null behavior stay unchanged.
The absence of a root filter keeps each endpoint's existing contract, including
the rejection of `filter=null` in query strings.

Encoding/decoding is distinct from semantic validation, as for existing Record
conditions. Unknown/duplicate members and ambiguous shapes are rejected;
operator/type/cardinality/source/period checks occur during preparation and
canonical admission. Invalidity takes precedence over unavailable capabilities
throughout the document, independently of rule order. Natural operands, sample
minimums and elapsed offsets are checked exactly before execution range limits,
including inverted intervals beyond uint64. Absolute instants with more than
nine fractional digits are rejected before decoding into time.Time; admitted
nanoseconds retain the existing microsecond execution capability check.
The lexical precision guard preserves the offsets admitted by time.Time,
including `-00:00`; it does not impose the Record DateTime parser's offset rules.
The numeric limits of the executable backend remain capability errors.
An older decoder rejects the unknown `count` member.

The existing Record-only preparer explicitly rejects behavioral subjects outside the admitted
Profiles path. Natural literals are validated and compared as decimal strings;
fixed elapsed-unit multiplication is linear in their digit count. This preserves
exact validation and invalid-before-unsupported precedence without large-integer
arithmetic on unbounded public values. There is no semantic digit limit. The
HTTP handler bounds Count filter documents to 500 KiB; valid literals beyond the
backend representation still receive a capability error.

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
Saving valid but unavailable filters or unresolved governed references as
drafts still requires a product decision when persistence is implemented.

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

## Profiles HTTP entry

`GET /v1/profiles` uses `Workspace.QueryProfiles`. A request without behavioral conditions delegates
to the existing Record query and writer: no evaluation object, added warehouse
query or changed response encoding. Requests containing Count, Sum, Average or Experience return only MATCH
profiles with the requested projection. `total` is the exact MATCH count. An
additional object describes that request's complete evaluation, for example:

```json
{"evaluation":{"at":"2026-02-01T00:00:00Z","match":2,"noMatch":1,"unknown":1}}
```

All three counts refer to the full population, before pagination, and share one
SQL evaluation with the page and its attributes. Count of occurrences remains
separate from these profile totals. An empty page still returns the totals.
UNKNOWN profiles are excluded from membership without being relabelled NO_MATCH.
The existing raw JSON contract remains: a present JSON null compared with the
String `"EUR"` is FALSE; a missing raw JSON path is NULL. No governed currency
binding or shared latent fact is inferred from either value. Governed
NativeCurrency is available publicly as the optional `count.currencyField` and
`count.nativeCurrency` pair. Only `commerce.order_completed.currency` revision
1, on `Order Completed`, is resolved. Explicit null, a missing half of the pair,
unknown reference/revision or incompatible source is invalid. The literal is
three uppercase ASCII letters, with no normalization or implicit default. The
same source, period, grain, terminals, sample minimum and limits still apply.
Count explanations add the resolved reference and literal only for governed
currency Counts; existing documents and raw Count explanation JSON are unchanged.

`at` optionally anchors Count periods. It is an RFC 3339 instant, not in the
future, exactly representable at microsecond precision; omission fixes T once
for the request. Empty, malformed, over-precise or future values are HTTP 400.
`at` is rejected without Count. Reusing the returned instant on later pages
avoids drift of relative periods, but does not freeze data or select a historical
Profiles/identity snapshot. Profile sorting uses the requested scalar column
(default `_updated_at`), NULLS LAST, then UUID ascending as a deterministic
tie-break. Only the Count path gains this explicit tie-break.

Invalid filters are HTTP 400. Missing properties and invalid sort properties
retain their existing 422 codes. Valid-language capabilities not implemented
return 422 `FilterCapabilityNotImplemented`; an unverified workspace backend
returns 422 `UnifiedFilteringUnavailable`. Maintenance and execution failure use
the existing MaintenanceMode/Unavailable responses; warehouse-supplied failure
text is not exposed by this new API path. Request-dependent page limits have
their own 422 codes: `ProfileProjectionTooWide` for the physical column budget
and `ProfileResultTooLarge` for SQL page, raw transport or HTTP encoding budgets.
These responses advise reducing `properties` or `limit` rather than retrying an
unavailable warehouse. Source-contract violations and execution/scan failures
remain technical failures and are not reclassified as page-size limits.
There is no Record fallback,
error-to-UNKNOWN conversion or successful partial page. All Count attributes are
encoded before the HTTP response is started.

The physical projection is limited to 90 flattened columns, including the
`_updated_at` metadata column: at most 89 user columns fit. An object may occupy
multiple columns. Omitting `properties` still requests every profile property;
if that projection exceeds the budget the request fails with
`ProfileProjectionTooWide`, before warehouse execution. It never silently drops
properties or switches to a Record query. Select a smaller explicit projection
to query a wider schema. Its SQL text-size budget remains 1 MiB per page. The
normalized cursor additionally bounds cumulative
raw text/binary input to 8 MiB before variable-size decoding, allowing for
physical encoding overhead; the handler bounds encoded attributes to 10 MiB.
Exceeding a resource limit fails the operation. The legacy Record path is not
subject to these new transport limits. Both drivers implement this projection
and ordering, but Snowflake remains gated until the runtime probes pass.

This increment does not define mixed-filter persistence, per-profile explanations,
Admin editing or destination execution. Aggregate ternary counts are not a
replacement for the per-profile evidence/diagnostics required by the complete
1.2 contract. Source-meaning registries and the deferred coherent-source release
guarantees remain separate work. This is the admitted Record/Count subset, not
all Unified capabilities.

## MCP entry

`query-profiles` reuses `Workspace.QueryProfiles` and the same JSON filter text
as the HTTP query parameter. MCP warehouse settings must be present, but the
query runs on the main datastore connection. The separate MCP user's database
permissions do not limit the profile or event properties read by this path.
Snowflake uses the main virtual warehouse; its Count gate remains closed.
The SQL tool stays available for discovery/other analysis, with instructions
prohibiting fallback to SQL for unsupported Unified features. See
[`cmd/internal/mcp/README.md`](../../../cmd/internal/mcp/README.md) and the
model-facing `profile-filter-doc` tool for transport, limits and examples.

## Selected Experience subset

`FilterExperienceCondition` extends the same public `core.Filter` rule union.
Its `experience` has `base:{source,period,where?}`, `select:{kind:"first"|"last"}` and a
mandatory nonempty `verify` of Record and selected timestamp predicates. There
is no outer operator/values, and no implicit selector or empty-body constructor. Where and verify use independent
local budgets (two groups, 100 descendants), not the main four-level/100-node
budget. Source and period binding reuse the existing recorded-event Count path;
no dialect or SQL enters core. The canonical owner retains base and verification
separately. `ExperienceVerification` is a closed selected-owner body, not a
relational plan or another public filter format. Nth is explicitly unimplemented,
not UNKNOWN. Record-only pipeline validation still rejects the new rule and state.Where is unchanged.

`warehouses.SelectedOccurrence` declares a determined message identity, attribution
and timestamp, row-local qualification and verification. The shared SQL keeps
all compatible winners: MAX(certain timestamp) for Last excludes older optional
rows; MIN(certain timestamp) for First excludes later optional rows; if no certain row exists, every possible row and the empty
alternative remain. Equal timestamps retain all identities. It returns TRUE
only with certain nonemptiness and all candidate verifications TRUE, FALSE when
all alternatives are FALSE (including certain emptiness), NULL otherwise.
NULL Record outcomes do not create governed shared facts or value completions.
No shared-fact or cross-profile predicate is admitted by this subset.

Copies are inspected before selection, including outside-period copies of a
seeded identity. Attribution and period membership must agree. Timestamp
disagreement inside the same period is ignored only for identities certainly
excluded from the base. Qualification and relevant verification conflicts remain
source faults, as do missing relevant timestamps. These checks apply only to
the E source contract; Count/Sum/Average behavior is unchanged.
Excluded rows do not need a known verify result. Source faults are complete
operation errors; explanations remain scoped to the requested profile.

Listing/count reuse the single-statement page/totals path. Explanation rows reuse
its two integer slots for candidate count and an empty-alternative flag, marked
as Experience by internal statement metadata, never as Count bounds. The decoder
checks domains and the guaranteed nonempty/empty relationship before projection.
Public explanations add period metadata, select and selection:{kind,absence,
candidates}. Absence is certain, possible or impossible; one candidate with
possible absence is ambiguous. No witness identity is invented or event payload
list transported. The result alone does not identify the missing Record operand.

First is the first recorded occurrence in the queryable history, not a claim
about the start of a real customer relationship. Retention, deletions and late
imports of older events can change it. First over a recent period means first
in that period, not first in the entire history.

The period always restricts the base. A window ending at T can express recency
of Last; a window ending earlier selects Last inside that window, ignoring later
messages. "Last order 60–90 days ago" uses absence in the last 60 days and
existence in the 60–90-day band, with the same source and qualification.

PostgreSQL tests exercise the real Go/Record/warehouse path, duplicated identities,
exact timestamp conflicts, missing timestamps, named and unnamed page/screen,
empty and uncertain bases, ties, no fallback, scoping, full history, independent
completion enumeration and HTTP/MCP. Snowflake has the same shared composer and
local adapter tests; its existing verification gate remains closed. No remote
Snowflake execution or infrastructure coherence guarantee is claimed.

### Selected timestamp membership

Only E verify admits a complete `{"occurredIn":{...period...}}` leaf, without
operator or values. It accepts elapsedLookback, elapsedAgoBand,
lastCompleteCalendarPeriods and absoluteHalfOpen, inheriting the root calendar.
The same period resolver and evaluation T are used for the base and each leaf.
Core binds the resolved [start,end) window to existing Record datetime >= and <
predicates on the selected occurrence; drivers retain their existing renderers.
A selected timestamp is determined, so the leaf is two-valued, while alternative
winners or possible absence can still leave the experience UNKNOWN.
FullRelevantHistory in this slot is valid but unimplemented; history in another
period is invalid. Invalid siblings win over missing capabilities in either
order. Every local group still observes the two-level/100-descendant budget.
No temporal leaf is admitted in base.where, reduction where, Profiles root or
Record pipelines. state.Where and saved Record filters are unchanged.

Experience explanations add occurredIn:[{path,start,end}], with zero-based rule
indexes relative to verify. Each leaf retains its own resolved window, even in
OR groups; candidate identities and payloads are not exposed. Empty metadata is
omitted, preserving the response for experiences with only Record verification.
First full history + recent occurredIn asks for a recent first recorded event;
First recent base instead asks for the first within that period. Retention,
deletions and older late imports can change a historical First.
