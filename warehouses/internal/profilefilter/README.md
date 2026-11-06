# Prepared Profiles filter execution

This MIT package is shared by the warehouse drivers. `core` supplies a prepared
`warehouses.Filter`; it does not supply SQL or dialect functions. The initial
subset is profile Record, independent occurrence Count and outer AND/OR. The
source, typed Record operands, known-JSON-String reads, mature period and accepted cardinalities
are already determined. Each Count closes its own evidence before composition.

`Build` emits one statement for a projected, ordered profile page and exact
MATCH, NO_MATCH and UNKNOWN totals. Empty pages still carry totals in a marked metadata row.
`Statement` and `Dialect` are internal physical implementation details. Existing
Record renderers own the 26 operations; this package does not reinterpret them.

`BuildExplanation` uses the same compiler for a single determined profile,
retaining all main-tree outcomes and Count bounds. A fixed six-column protocol
returns at most 101 node rows in preorder. `ReadExplanation` validates the node
sequence, target identity and nonnegative bounds and closes the cursor on every
path. Source, cancellation, scan, iteration or Close failures return no partial
explanation. Unlike population listing, identity seeds are restricted to those
having any copy attributed to the target profile, including outside the period;
all copies of each seeded identity still participate in conflict checks.

Each profile Record leaf is projected only in the population-table SELECT,
without joins, a prefilter or NULL coalescing. The resulting nullable columns
are combined with closed Count outcomes in the shared AND/OR tree. Aliases for
this projection contain `$`, outside the property namespace, to avoid collisions
also on backends with lateral SELECT aliases. Only these internal columns are
carried into the Count joins, never `SELECT *` from the profile source.
Requested attribute columns and the optional scalar sort key use separate
internal aliases. Sorting is NULLS LAST with UUID ascending as a tie-break;
without a requested key the page retains UUID order. Empty pages have nullable
projection slots on the metadata row, not a fabricated profile.

Projection is bounded to 90 physical columns and 1 MiB of SQL text-sized page
values before transport. Dialects supply only the value-size operation; the
common composer guards the projected fields and returns a separate boolean
`payload_exceeded` diagnostic. `Read` maps it to
`warehouses.ErrProfileResultTooLarge`, independently of source violations. The
driver scanner additionally limits cumulative raw
text/binary input to 8 MiB before normalizing JSON, arrays or maps, returning the
same size-limit identity on overflow. Oversized projections return
`warehouses.ErrProfileProjectionTooWide` before execution. Existing
Record queries do not enable these new scanner limits.

Semantic groups and leaves share the four-level/100-descendant contract.
Expanded Record expressions have a separate, whole-filter 16,384-node budget,
so many object-presence expansions cannot each reset their validation budget.
A root Record leaf has only its one condition's binder expansion, not a hidden
user boolean tree. ProfileSource supplies both the binding context and the
outcome request population. Pure Record requires no occurrence source.

The Count procedure groups by identity, not by `(profile,identity,qualification)`.
For every identity with a copy in the period, it inspects all copies of that
identity, including those outside the period. The relevant tuple is attribution,
period membership and qualification within the period. Copies with the same
relevant state count once. Payload differences and timestamps with the same
membership do not add occurrences. Contradictions fail the operation under the
admitted determined-identity source contract. This is not an implementation of
semantic uncertain identity or deduplication.

All-unattributed identities are outside the Profiles view and contribute zero.
A mixture of attributed and unattributed copies is a conflict. Attribution
conflicts outside the requested population are conservatively rejected when an
identity has a copy in the period. Copies all outside the period are ignored.
A known-identity source with an attributed NULL identity in the period also
fails. Source schema guarantees and the coherent view remain caller/binding
preconditions; this is not a warehouse authenticity or general quality audit.

The helper produces bounded diagnostic fields with the totals, rather than
exceptions or error text dependent on a database dialect. A nonzero
`source_failures` field causes `Read` to discard the result with a technical
source-contract failure; it takes precedence over a concurrent size limit.
`payload_exceeded` instead identifies a page-size limit. Both return a nil result.
The decoder preserves the old checks for
count type, sign and overflow, UUIDs, ordering, page limits, cancellation,
`Rows.Err` and `Close`. Every failure returns a nil result.

Runtime counters have a signed-64 technical domain. PostgreSQL integer SUM
already errors on overflow; the helper additionally marks out-of-domain counters
on platforms whose SUM may use a wider number, so normalization near MaxInt64
cannot silently change a result. These checks are error handling, not a semantic
upper bound. Count literals remain exact integer SQL literals. Endpoints are
UTC timestamp parameters with exactly six fractional digits; no session clock
or timezone determines filter meaning.

The source scan and semijoin strategy is an initial physical implementation,
not a production performance claim. The same prepared contract permits another
physical strategy within a driver. There is no relational plan or optimizer.

PostgreSQL has real service/datastore/driver and numeric probes. The mirrored
Snowflake adapter is locally tested but remains gated until real conformance,
number/type/NULL, timestamp and resource probes pass. No current Snowflake
credentials or runtime measurements are claimed.
