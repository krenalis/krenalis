# Unified Filtering: Snowflake campaign

Run commands from the repository root. This is a test runbook, not an activation
of Unified Filtering on Snowflake. The production gate remains false. Local
compilation, PostgreSQL results and skipped Snowflake tests are not remote passes.
The complete acceptance checklist is in
[the Unified README](../../core/internal/unified/README.md#consolidated-snowflake-verification-campaign).

## Local preparation

These commands neither load credentials nor connect to Snowflake. PostgreSQL
integration uses a disposable local Docker container. Keep the Snowflake skip
variable set even if credentials happen to exist in the shell.

```sh
export GOPROXY=off GOSUMDB=off
export GOCACHE="$PWD/.gocache"
export KRENALIS_SKIP_SNOWFLAKE_TESTS=true

go test ./core/internal/unified ./warehouses/internal/profilefilter \
  ./warehouses/snowflake ./connectors/snowflake ./core/internal/connections \
  ./cmd/internal/mcp -short -count=1 -json > /tmp/unified-local.jsonl

go test ./core/internal/datastore \
  -run '^TestUnified(Conformance(PostgreSQL|Snowflake)|StoredNumbers(PostgreSQL|Snowflake)|ProfilesSnowflakeGate)$' \
  -count=1 -timeout=5m -json > /tmp/unified-fixtures-postgresql.jsonl

go test ./test -run '^TestEqualityOfMergeFiles$' -count=1

go test ./test/snowflaketester -race -count=1
python3 -B -m unittest discover -s test/snowflaketester -p 'test_*.py'
```

`TestUnifiedConformancePostgreSQL/Snowflake` shares fixture data, expected results
and assertions. Backend-specific SQL is confined to table setup and fixture
writes. Events go through the real Merge writer; copies are inserted into tables
without identity constraints so that a primary key cannot hide deduplication
defects. Normal fixtures assume coherent publication; the `sources` group
deliberately violates attribution and copy contracts. Asynchronous publication
is not tested.

The original `selections`, `periods` and `pages` groups have remote evidence below.
The added `count-money`, `guards`, `sources` and `protocol` groups passed on local
PostgreSQL with the race detector and on Snowflake in both session timezones on
2026-11-12. They run in a separate remote block before the baseline fixtures in
the sequence below.

| Fixture | Coverage |
| --- | --- |
| `selections` | 32 histories: 12 fixed counterexamples, 16 PCG(26,11011) samples of at most six events, four eleven-event borders. First, Last, Nth(1/2/3/10), Latest(1/2/10) All/Some/exactly-one/at-most-one, all four ordered Last-minus-Previous comparisons. Missing, null and incompatible currency, negative amounts, unreadable amounts, opposite Decimal(18,4) borders, optional membership, ties and shared-currency conditioning. |
| Selection assertions | Existing independent enumeration oracles supply outcomes and candidate/absence/pair metadata. Every profile's metadata is checked for Latest(2)/Some and difference <= -30; the other rules explain the first six profiles, plus all four eleven-event histories for position/size 10. All rules check exact ternary totals, profile ordering and empty matching pages. Relevant timestamp conflicts fail without partial results; excluded copies and unrelated-profile explanations remain usable. |
| `periods` | Absolute, elapsed and ago-band periods; Rome complete days across both DST changes, complete week and month. Start/end and +/- one microsecond are checked through both Count and First/occurredIn. History includes an event in year 0001 and excludes T. received_at/sent_at deliberately lie outside the window. |
| `pages` | Page and screen, named and unnamed, missing/empty/lowercase names, profile without events, literal dotted JSON key segments and count explanations. |
| `count-money` | Nineteen monetary observations plus an empty profile, two identities per populated profile and physical copies. Thirteen literal tests, external currency absent/EUR/USD, nested AND/OR, Count >= 1, = 0 and != 2 with minimum 2. Independent completion oracles supply outcomes; one complete configuration checks every profile's count bounds. |
| `guards` | Seventeen batched populations for eight Sum/Average tests: independent where/currency uncertainty, positive and negative amounts, unknown amounts counted toward minimum, certainly excluded unreadable amounts, Kleene counterexamples, exact sums beyond binary64 precision and physical-copy deduplication. Handwritten expectations check every profile's exact inputs/bounds and ternary outcome. |
| `sources` | Relevant qualification, attribution, period, exact timestamp, readability and amount conflicts across Count, Sum, Average, selected experiences, Latest N and LastVersusPrevious; equivalent unreadable values and excluded identities; missing identity; unrelated-profile explanation isolation. Readback verifies physical copies were retained. NULL timestamps test only selected readers' existing runtime guards: known timestamps remain a source precondition for reductions. |
| `protocol` | Six mixed Record/occurrence explanation layouts checked on all nine profiles; nonempty, partial and empty matching pages, exact totals/T, projected fields and deterministic identity tie-breaking in both sort directions. Pure Record membership/explanation works after dropping the event table, including missing profiles and an empty population. |
| `TestUnifiedStoredNumbers*` | Shared six-profile monetary fixture: Count, Sum with minimum, Average, selected money, LastVersusPrevious and composition, with all profile outcomes/explanations, canceled operation and empty pages. |

The bounded fixture complements the full PostgreSQL completion oracles; it does
not replace them. It avoids running the exhaustive suites remotely. Its selection
block has 22 main batched membership checks and 204 individual explanations, rather
than thousands of remote oracle queries. The named subtests can be selected
separately, for example `-run '^TestUnifiedConformancePostgreSQL$/periods'`.
The remote sequence isolates the four new groups in `unified-new-groups.jsonl`.
The subsequent `unified-snowflake.jsonl` block keeps the baseline stored-number,
selection, period and page/screen fixtures; its `-skip` excludes the four groups
already run. Both blocks use the same overlay, session/schema audits and timeout,
and together cover the complete bounded fixture without repeating the new groups.

## Preparing an isolated gate and session overlay

Prepare this only for testing. The helper copies the current sources, opens the
gate in the overlay and sets login parameters in all three connection factories:
warehouse, connector and schema-creation helper. Tracked production sources stay
unchanged. Regenerate after source changes; do not reuse old implementation copies.

```sh
session_timezone=UTC
campaign_dir=$(mktemp -d /tmp/krenalis-snowflake-campaign.XXXXXX)
python3 -B test/snowflaketester/prepare.py "$campaign_dir" --timezone "$session_timezone"

# Compile and prove that the bootstrap still skips before credential access.
KRENALIS_SKIP_SNOWFLAKE_TESTS=true go test -overlay "$campaign_dir/overlay.json" \
  ./core/internal/datastore -run '^TestUnified(Conformance|StoredNumbers)Snowflake$' \
  -count=1 -json > "$campaign_dir/compile-only.jsonl"
```

This last command must report SKIP for both tests. The database/sql cursor
failure regression uses an in-memory driver and can run with this overlay
without credentials or network access:

```sh
KRENALIS_SKIP_SNOWFLAKE_TESTS=true go test -overlay "$campaign_dir/overlay.json" \
  ./warehouses/snowflake -run '^TestProfileSchemaFailures$' -race -count=1
```

Test the production gate separately without an overlay (`TestUnifiedProfilesSnowflakeGate` above). Do not
run the whole datastore package with the gate overlaid: gate-contract assertions
intentionally expect it to be closed. There is no runtime bypass.
Run the remote blocks once with UTC, then in a fresh campaign directory prepared
after setting `session_timezone=America/Los_Angeles`. This DST zone differs from the Rome calendar
used by the fixtures. Both runs use the same expectations and cover half-open
microsecond borders and the spring/autumn calendar transitions.

## Remote execution, when authorized

The user authorised the live campaign on 2026-11-11. Use the existing
`KRENALIS_SNOWFLAKE_TESTER_*` environment variables,
an isolated test database/role and the temporary schemas created by
`snowflaketester`. Do not include credential files or values in commands, logs or
commits. Do not enable shell tracing. No application or demo data is needed.

Keep runs sequential (`-parallel=1 -p=1`), retain JSON test results, check cleanup
and redact server diagnostics before sharing them: driver errors can contain
account, database, user or schema identifiers. A skipped test is not a pass.
The overlay passes TIMEZONE, TIMESTAMP_TYPE_MAPPING=TIMESTAMP_NTZ,
TIMESTAMP_INPUT_FORMAT=AUTO and STATEMENT_TIMEOUT_IN_SECONDS=60 through
`Config.Params`. The vendored driver copies this configuration for each
[Connect](../../vendor/github.com/snowflakedb/gosnowflake/v2/connector.go#L34),
[initializes the session parameters](../../vendor/github.com/snowflakedb/gosnowflake/v2/connection.go#L680)
and [sends them at login](../../vendor/github.com/snowflakedb/gosnowflake/v2/auth.go#L299).
This includes replacement connections opened later.
The overlay's `SessionConnector` checks
[SHOW PARAMETERS IN SESSION](https://docs.snowflake.com/en/sql-reference/sql/show-parameters)
on each new physical connection before returning it to the pool. It logs the four
verified values, closes a connection that fails verification and leaves the
underlying driver connection unchanged. This diagnostic query runs only in the
test overlay; production queries and settings do not change.
The fixtures do not reset TIMEZONE or the timestamp formats. The numeric tests
retain their 60-second timeout on their single connection when run without the
overlay. Session changes after verification and other timestamp input formats
remain separate coverage.

The environment owner must verify server-side time limits and resource controls
cover every test connection. A session setting can override a user default;
warehouse and session timeouts interact as described in the
[Snowflake cost-control documentation](https://docs.snowflake.com/en/user-guide/cost-controlling-controls).
`go test -timeout` can terminate the test process without server cancellation or
`t.Cleanup`, so it is not a substitute for those controls.

For new blocks, `run.py` uses Python 3.11+ and its standard library. It saves a
source manifest before starting Go and again
after it exits: HEAD, working-tree status, test-file and modified source hashes,
overlay/replacement hashes, Go version, driver version, exact command and result.
It reports source changes during the run as a nonzero exit, even if Go passes.
Skipped blocks and selections that execute no tests also return a nonzero exit.
Use fresh output names outside the checkout; evidence is never overwritten.
The helper does not source `.env` files. It decodes Go JSON events and rejoins
adjacent `Output` fragments with the same package/import path, test and action
until a newline, stream change or EOF. It then masks Snowflake test environment
values case-insensitively in `Output` only and re-encodes the event, retaining
the first fragment's metadata. Package, test and action metadata remain unchanged;
non-event lines are masked as text. Values at most four characters long are
flagged by variable name in the manifest, without recording their values.
Reassembly is limited to 4 MiB of UTF-8 output per line. Oversized output is
replaced in full with `<OUTPUT_EXCEEDS_LIMIT>`, the manifest records the limit
violation and the block fails (exit 5 if otherwise successful). The helper keeps
draining Go output so test cleanup can finish. The schema audit also fails,
since omitted output could contain schema records.
Exit 2 also covers invalid arguments; a manifest distinguishes source changes
during a run from argument errors that prevent a run from starting.
Inspect redacted logs before sharing.
The manifest contains source and schema metadata; credentials belong in the
environment, never command arguments. The helper adds `-mod=vendor -parallel=1 -p=1 -failfast
-count=1 -json`, matching the flags used in the live runner.
For remote blocks, `--require-schemas` also requires nonempty, balanced schema
records; `KRENALIS_SKIP_SNOWFLAKE_TESTS=false` enables the same requirement.
The helper collects generated schema names before masking output and
updates the manifest when creation/drop records arrive, including before a run
completes. Generated schema names are not credentials.
The campaign commands also require `--require-session-timezone "$session_timezone"`.
Before masking, the runner records session verification counts by timezone in
`sessions.verified`, the requested zone in `sessions.required_timezone` and the
number of malformed records in `sessions.invalid_records`. Only the two campaign
timezone names are stored; unrecognized field values are discarded.
A block fails if verification records are absent, malformed, mixed or use a
different timezone (exit 6 if otherwise successful). This also detects an omitted
or wrong overlay. Without the option, session collection does not add a failure
condition. Inspect each block's exit and session evidence alongside the schema
cleanup audit.

Run the following blocks in order, stopping on a failed prerequisite.
Once the physical blocks pass, a failure in `unified-new-groups.jsonl` does not
block `unified-snowflake.jsonl`: run the baseline block anyway and retain both
results.

```sh
# Credentials must already have been supplied through the environment.
# These commands are remote, including tests that do not need the Unified gate.
KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone "$session_timezone" "$campaign_dir/physical-warehouse.jsonl" -- \
  -overlay "$campaign_dir/overlay.json" ./warehouses/snowflake \
  -run '^(TestJSONNumbers|TestProfileExactDecimal|TestJSONExpressions|Test_Merge)$' \
  -timeout=30m

KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone "$session_timezone" "$campaign_dir/physical-connector.jsonl" -- \
  -overlay "$campaign_dir/overlay.json" ./connectors/snowflake \
  -run '^Test_Merge_Query$' -timeout=15m

KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone "$session_timezone" "$campaign_dir/unified-new-groups.jsonl" -- \
  -overlay "$campaign_dir/overlay.json" ./core/internal/datastore \
  -run '^TestUnifiedConformanceSnowflake$/(count-money|guards|sources|protocol)$' \
  -timeout=45m

KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone "$session_timezone" "$campaign_dir/unified-snowflake.jsonl" -- \
  -overlay "$campaign_dir/overlay.json" \
  ./core/internal/datastore -run '^TestUnified(StoredNumbers|Conformance)Snowflake$' \
  -skip '^TestUnifiedConformanceSnowflake$/(count-money|guards|sources|protocol)$' \
  -timeout=45m

# The PostgreSQL arm is skipped: this block isolates real Snowflake Record reads.
KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone "$session_timezone" "$campaign_dir/record-floats.jsonl" -- \
  -overlay "$campaign_dir/overlay.json" ./core/internal/datastore \
  -run '^TestJSONFloatFilters$/Snowflake' -timeout=15m
```

Physical numeric expectations follow the approved
[JSON transport contract](../../warehouses/snowflake/README.md). New fixed-point
writes and historical DOUBLE writes are separate fixtures. Preserve exact
INTEGER/DECIMAL values; compare native DOUBLE transport separately from returned
JSON display text. A rounded display is an accepted output limit, not a reason to
weaken Record or governed monetary expectations. Do not add corrective queries,
rewrite connector SQL or restore the discarded result dictionary.

## Numeric transport: Arrow/JSON

`TestJSONNumericTransport` in the warehouse and connector uses shared fixtures:
27 rows for Arrow and a compact JSON subset. Each numeric value appears in
VARIANT, ARRAY and OBJECT, including nested arrays and literal dotted/bracketed
keys. Real Merge writes cover exact 38-digit integers, scale 37 and scientific
notation; explicit PARSE_JSON writes create historical DOUBLEs. Diagnostic reads
check TYPEOF on all seven value paths and exact fixed-point values. Application
reads must retain the server's numeric tokens, comparing JSON structure without
depending on key order or whitespace.
Type and numeric differences are collected across fixtures instead of stopping
at the first mismatch. Diagnostics identify the fixture and path or column,
report physical kinds from a closed set and quote the first unequal numeric
tokens with a 400-character limit per token. Structural precondition failures
still stop the read. Historical JSON uses bind parameters, not SQL string literals.

Both fixtures compare application tokens with a separate diagnostic read of the
same immutable data in the same requested format. Stability of the server's text
between those two reads is an assumption to verify remotely; there is no
corrective query in production. The connector fixture checks its raw query API
and metadata, while local tests pass the complete matrix through the actual core
normalizer. The connector's query stays unchanged.
DOUBLE tokens, native bits and round-trip results are observations, not a renewed
promise of binary64 identity for nested JSON. The fixture names identify common
cases across the full and compact populations. Negative-zero sign is not an
acceptance requirement.

Arrow checks the intended ordinary path. Explicit JSON runs are compatibility
probes, not certification of equivalent support. A separate
`connections.TestSnowflakeNativeTransport` reads FLOAT, BOOLEAN, integer and
decimal columns through both application readers. It uses the connector's actual
`queryScanValue` normalization and also observes the raw native Go types.
It collects all incompatibilities before one final failure: the warehouse's
current rejection of textual FLOAT/BOOLEAN is expected to surface in JSON mode.
Do not weaken this probe or change production behavior merely to obtain a pass.

`prepare.py --result-format ARROW|JSON` sets the request at login in all three
connection factories and requires its session value. Only these overlays add an
observation to a copy of the vendored driver's `populateChunkDownloader`. It emits
`Snowflake result format observed tag=... format=arrow|json|unknown` only for six
closed fixture tags. It does not alter rows or decoders. SHOW/DDL/COPY responses
are not evidence for application reads. Normal builds and the other campaign
profiles retain their existing behavior.

`run.py --require-result-format` and repeated `--require-result-tag` options are
paired. The `result_formats` manifest records the requested session value and
observed formats per tag before output masking. Every required tag needs at
least one observation, and all observed formats must match. Missing, malformed,
unknown or mixed records fail an otherwise successful block with exit 7.
`verified` records the verdict even when Go or another check fails; it is null
when no result format was required. Existing failures retain their original
code. Require both `verified: true` and `source_stable: true` before attributing
a native JSON failure to that transport. A session setting alone cannot certify
the result format. These records and the overlay hashes complement the existing
schema/session checks; they do not replace them.
Both entry scripts disable local bytecode writes before importing shared code.

Local checks use fake results, the real CSV encoders, connector normalization and
an isolated driver-overlay unit test. To compile the remote fixtures without
accessing credentials, prepare an overlay, then run with the skip variable true:

```sh
numeric_campaign=$(mktemp -d /tmp/krenalis-numeric-transport.XXXXXX)
python3 -B test/snowflaketester/prepare.py "$numeric_campaign/ARROW" \
  --timezone UTC --result-format ARROW
KRENALIS_SKIP_SNOWFLAKE_TESTS=true go test -overlay "$numeric_campaign/ARROW/overlay.json" \
  ./warehouses/snowflake ./connectors/snowflake ./core/internal/connections \
  -run '^Test(JSONNumericTransport|SnowflakeNativeTransport)$' -count=1 -json
```

The three remote tests must SKIP here. Local passes and compilation do not attest
either server format. These small fixtures do not deliberately force remote
chunk downloads; that remains separate coverage.

This is the next independent campaign; it does not require replaying the earlier
Unified blocks. When live access is authorized, use fresh evidence names and UTC
overlays. Run the four numeric blocks first, inspecting each result:

```sh
numeric_campaign=$(mktemp -d /tmp/krenalis-numeric-transport-live.XXXXXX)
for numeric_format in ARROW JSON; do
  numeric_dir="$numeric_campaign/$numeric_format"
  python3 -B test/snowflaketester/prepare.py "$numeric_dir" \
    --timezone UTC --result-format "$numeric_format"
  for numeric_path in warehouses connectors; do
    numeric_role=${numeric_path%s}
    KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py \
      --require-schemas --require-session-timezone UTC \
      --require-result-format "$numeric_format" \
      --require-result-tag "krenalis-transport/$numeric_role-values" \
      --require-result-tag "krenalis-transport/$numeric_role-reference" \
      "$numeric_dir/$numeric_role.jsonl" -- -overlay "$numeric_dir/overlay.json" \
      "./$numeric_path/snowflake" -run '^TestJSONNumericTransport$' -timeout=15m
  done
done
```

Then run native probes as separate blocks, so their failure cannot hide the
numeric checks behind `-failfast`. A JSON incompatibility is evidence to inspect,
not permission to change contracts or open the gate:

```sh
for numeric_format in ARROW JSON; do
  numeric_dir="$numeric_campaign/$numeric_format"
  KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py \
    --require-schemas --require-session-timezone UTC \
    --require-result-format "$numeric_format" \
    --require-result-tag krenalis-transport/warehouse-native \
    --require-result-tag krenalis-transport/connector-native \
    "$numeric_dir/native.jsonl" -- -overlay "$numeric_dir/overlay.json" \
    ./core/internal/connections -run '^TestSnowflakeNativeTransport$' -timeout=15m
  python3 test/snowflaketester/run.py --audit "$numeric_dir"
done
```

### Live transport results on 2026-11-12

All six blocks ran on `816c4cd6d`, using real writes and reads. Each manifest has
`result_formats.verified: true` and `source_stable: true`, including the failed
native JSON probe. No test was skipped.

| Block | Arrow | JSON |
| --- | --- | --- |
| Warehouse JSON numeric transport | PASS, 27 fixtures | PASS, 17 fixtures |
| Connector JSON numeric transport | PASS, 27 fixtures | PASS, 17 fixtures |
| Native types through both readers | PASS | FAIL, FLOAT/BOOLEAN incompatibilities |

The four numeric blocks checked 88 rows, seven physical-kind paths per row and
all three application cells. New fixed-point writes and historical DOUBLE
fixtures passed their distinct expectations. Bind parameters inside PARSE_JSON
worked on the server. The seven common DOUBLE observations had identical tokens
across the four blocks. The successor of 100 still displays as 100 when nested;
the maximum binary64 token still exceeds the finite ParseFloat range. These
observations preserve the approved server-text contract.

With JSON results, the warehouse rejects textual native FLOAT and BOOLEAN. The
connector receives Boolean strings `1`/`0` and rejects them; it reads the native
successor of 100 as 100. INTEGER and DECIMAL passed through both readers in both
formats. The native JSON probe remains failed; no production correction was made.
Native JSON FLOAT text was not logged. Its digit count and representation of the
maximum finite value, NaN and infinities remain unmeasured; nested DOUBLE text
does not establish these native-format properties. Native INTEGER/DECIMAL
coverage here is limited to BIGINT and DECIMAL(38,37) on three rows.

Both schema-ledger audits report three schemas created and three dropped.
Separate read-only server inventories found none of these six schemas remaining.
Evidence, per-query format records, masked logs and `summary.json` are under
`/private/tmp/krenalis-numeric-transport-live-yxltfce1/`. The initial sandbox DNS
failure occurred before schema creation and has separate evidence under
`/private/tmp/krenalis-numeric-transport-live-dlbs2lhs/`.

Account/user-level parameter changes, forced remote chunk downloads and
additional native types remain outside this block. The Unified gate is closed.

### Read-only parameter inspection on 2026-11-12

A separate probe, without a result-format override, executed SHOW PARAMETERS
LIKE 'GO_QUERY_RESULT_FORMAT' IN ACCOUNT, IN USER and IN SESSION. Each returned
one exact row with value/default ARROW and an empty level. This observes the
test environment's default; no ALTER was attempted to establish which scopes
allow changes. It does not attribute a format to the earlier conformance runs.
The probe created no schemas. Its bounded observations, temporary test overlay
and source-stable manifest are under
`/private/tmp/krenalis-snowflake-parameter-scope-t3a61rh2/`.

## Cleanup after completion or interruption

Audit schema records collected in manifests before redaction, including failed
runs and interrupted runs:

```sh
python3 test/snowflaketester/run.py --audit "$campaign_dir"
```

The audit reports manifest and created/dropped record totals, missing drops and
unmatched drops. No manifests, incomplete/invalid manifests or required schema
records missing from a block make it fail. It does not infer cleanup from a
zero count after searching redacted text. Older manifests without schema evidence
cannot pass this audit; use their original records and the server inventory.
A remote attempt that fails before any CREATE also fails the audit for missing
required records; use the server inventory to verify cleanup in that case.

A process can die between CREATE and its log record. After an interrupted run,
also inspect the supplied test database using
[SHOW SCHEMAS](https://docs.snowflake.com/en/sql-reference/sql/show-schemas):

```sql
SHOW SCHEMAS IN DATABASE <test_database>
  STARTS WITH 'KRENALIS_TEST_SCHEMA_' LIMIT 1000;
```

Match exact schema names with this run's logs and query history; do not drop a
prefix wholesale, since other test runs can still be using their schemas.
Confirm outstanding queries have stopped, then remove only this run's schemas.
If results reach the limit, finish the inventory before treating it as complete.

## Live results on 2026-11-11

The prepared blocks ran against the supplied test database with real writes and
reads. No production gate or implementation code changed. Successful test nodes
(including parent nodes) were:

| Block | PASS | SKIP |
| --- | ---: | ---: |
| `TestJSONExpressions` | 334 | 0 |
| `TestJSONNumbers`, `TestProfileExactDecimal`, `Test_Merge` | 3 | 0 |
| Connector `Test_Merge_Query` | 1 | 0 |
| `TestUnifiedConformanceSnowflake`, `TestUnifiedStoredNumbersSnowflake` | 44 | 0 |
| `TestJSONFloatFilters/Snowflake` | 482 | 0 |

These are 864 successful nodes, not 864 distinct SQL queries. The numeric batch
checked 21,216 exact comparisons; the last block checked 480 Float Record cases.
The new writer stored `1.1E0` as DECIMAL 1.1 and `1.00000000000000001E2` as
DECIMAL 100.000000000000001. Historical PARSE_JSON fixtures retained DOUBLE
storage and their separately approved reading. The Unified fixtures passed
outcomes, explanations, calendars, microsecond borders, copies and source-fault
isolation. All ten schemas logged as created across attempts were logged as
dropped; this count is not an independent server inventory after a hard timeout.

Two fixture defects were found and corrected before completing the blocks:
`TestJSONExpressions` assumed scientific inputs were always DOUBLE and derived
its oracle from a floating-point conversion even for new fixed-point writes;
it now asserts the physical type and uses exact fixed-point values. The loop
adding historical DOUBLE fixtures in `TestJSONNumbers` called JSON Kind on the
empty SQL NULL placeholder; it now skips that placeholder. Numeric expectations
were not weakened, and the corrected type assertions detect loss in new writes.

After `TestJSONExpressions` passed, its process encountered the second fixture
defect in `TestJSONNumbers`. The numeric/Merge tests were then rerun separately
with `-run '^(TestJSONNumbers|TestProfileExactDecimal|Test_Merge)$'` and the same
vendor, sequential and fail-fast flags;
the table combines successful test nodes from these runs. Initial failed attempts
are retained alongside successful runs, rather than counted as passes.
Logs were redacted before writing under
`/tmp/krenalis-snowflake-campaign-live-20261111/`; `evidence-summary.json`
identifies the relevant log for each block. No credentials are included here.
These historical logs did not capture commit/source manifests. Their source
provenance is therefore incomplete; the helper above applies to subsequent runs,
without retroactively certifying them. Historical subtest labels `latest2/1/0`
and `difference/3` correspond to the current `latest2/some` and `difference/le`.

## Live UTC/non-UTC results on 2026-11-11

All four blocks ran twice on `981927691abc1ea79766d2c5fd2e1c88ff6aa72e`, first
with UTC and then with America/Los_Angeles. Each run used its own freshly
generated overlay, required session evidence and retained source manifests.
All eight blocks reported stable sources, no malformed session records and
successful output-redaction checks.

| Block | UTC PASS | America/Los_Angeles PASS | SKIP |
| --- | ---: | ---: | ---: |
| Physical warehouse | 337 | 337 | 0 |
| Physical connector | 1 | 1 | 0 |
| Unified conformance and stored numbers | 44 | 44 | 0 |
| Float Record filters | 482 | 482 | 0 |
| Total, including parent nodes | 864 | 864 | 0 |

The sorted package/test/action records matched across the two configurations.
Each configuration recorded 19 session verifications: 8 for the warehouse
block, 5 for the connector, 4 for Unified and 2 for Float Record filters. Every
record confirmed the requested timezone, NTZ mapping, AUTO timestamp input and
the 60-second statement timeout. These results cover the fixture expectations,
including the Rome calendar's spring/autumn transitions and microsecond borders.
No statement timed out; timeout/cancellation failure behavior was not exercised.

Each manifest audit reported 8 schemas created and 8 dropped, with no missing
or unmatched records. A separate read-only server inventory checked all 16
campaign schema names and found none remaining. It listed 23 other test schemas,
which were left untouched. The inventory itself passed with stable sources.

Evidence is under `/tmp/krenalis-snowflake-p1-live-zsbbhh91/`: `summary.json`,
the `UTC/` and `America-Los_Angeles/` manifests and redacted JSON logs, and
`inventory/report.json` with the inventory's source overlay and manifest.
No implementation change was needed. This verifies P1's two session
configurations; resource-monitor policies and credit limits were not inspected.
The production gate remains closed and the remaining acceptance coverage below
still applies.

## Shared conformance results on 2026-11-12

The four added groups and the separate baseline block ran on `816c4cd6d`, with
unchanged expectations in UTC and America/Los_Angeles sessions:

| Block | UTC PASS | America/Los_Angeles PASS | SKIP / FAIL |
| --- | ---: | ---: | --- |
| Count money, guards, sources and protocol | 392 | 392 | 0 / 0 |
| Stored numbers, selections, periods and pages | 44 | 44 | 0 / 0 |

These are test/subtest nodes, not SQL-query counts. In each configuration the
blocks share only the parent `TestUnifiedConformanceSnowflake`; their union has
435 nodes. Every block has stable source hashes and verified session parameters.
These four conformance blocks used the session default without requesting or
recording GO_QUERY_RESULT_FORMAT or the actual response format. Their passes are
not attributed to Arrow; the per-query format observations above cover only the
six numeric-transport blocks. Record the required and actual format in the next
necessary acceptance run rather than repeating this suite only for attribution.
The runs took about 14 minutes for each new-group block and 8 minutes for each
baseline block, including setup and many individual explanations. This is not a
production latency benchmark.

Each configuration created and dropped three schemas. Independent read-only
inventories found zero remaining campaign schemas. Evidence is under
`/private/tmp/krenalis-unified-conformance-live-zgm78an6/`: `summary.json`, per-zone
manifests and masked logs, ledger audits, and the `inventory/` results. Output
checks found no occurrences of the six Snowflake configuration values in either
campaign's test output. No implementation or production gate changed.

## Public Profiles service

`TestUnifiedServicePostgreSQL` and `TestUnifiedServiceSnowflake` use the same
six profiles, thirteen recorded messages and handwritten public JSON oracles.
Both initialize their warehouse with production `Initialize` and publish the
fixtures through `Merge`. The application database remains a disposable local
PostgreSQL container in both tests. No publisher or pipeline runs concurrently.

The group checks public membership/totals, every Count money explanation,
Sum/Average exact decimal inputs, Nth/occurredIn and absence, Latest N, an
ambiguous ordered pair, projection/schema serialization, microsecond profile
timestamps, NULLS LAST/UUID pagination, a missing profile and oversized-page
rejection without a partial result. October in Europe/Rome crosses DST; events
on both half-open boundaries distinguish the intended calendar window.
The projected profile has a SQL NULL label: the schema retains the property,
while its attributes omit the key. JSON null inside the document remains null.
The Sum/Average input `100.125 + 70.0003` yields the exact sum `170.1253`;
binary64 addition would instead expose `170.12529999999998`.

Run locally first, with no Snowflake credentials:

```sh
KRENALIS_SKIP_SNOWFLAKE_TESTS=true go test ./core -race \
  -run '^TestUnified(ServicePostgreSQL|WorkspacePostgreSQL)$' -count=1 -timeout=10m
```

Run Snowflake separately from the existing PostgreSQL service test: that test
contains a deliberately unconfigured Snowflake workspace whose gate must stay
closed. A normal build of the new Snowflake test skips before loading credentials.
With authorized live access, execute the complete bounded group in each session
timezone, using a fresh directory and the existing credential environment:

```sh
service_campaign=$(mktemp -d /tmp/krenalis-unified-service-live.XXXXXX)
for service_timezone in UTC America/Los_Angeles; do
  service_dir="$service_campaign/$service_timezone"
  python3 -B test/snowflaketester/prepare.py "$service_dir" \
    --timezone "$service_timezone" --result-format ARROW
  KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py \
    --require-schemas --require-session-timezone "$service_timezone" \
    --require-result-format ARROW \
    --require-result-tag krenalis-transport/service-query \
    --require-result-tag krenalis-transport/service-explanation \
    --require-result-tag krenalis-transport/service-record \
    "$service_dir/service.jsonl" -- -overlay "$service_dir/overlay.json" ./core \
    -run '^TestUnifiedServiceSnowflake$' -race -timeout=15m
done
```

The closed service tags are set on public-method contexts, outside initialization
and writes. The driver observer records format and `statement=select|other` from
response metadata. The runner requires at least one SELECT for every service tag,
and the requested format on every observed response, including any `other`.
Historical manifests without statement metadata remain usable for transport
tags; new service evidence requires it. Check each result independently and use
the ledger audit and independent `TestCampaignInventory` described above.

This group verifies the public core facade with real state and datastore. Its
warehouse factory uses an alias returning a real test instance with no production
dialer or encrypted-settings loading. It does not run `Core.Open`, the HTTP
handler, authentication or MCP. Page JSON uses the same schema-directed attribute
encoder as the handler, not an HTTP request. The HTTP and MCP groups below cover
the separate transports; MCP uses the main connection and a workspace MCP key.
This group's format evidence does not retroactively label the 435 conformance
nodes, and does not open the production gate.

### Live public-service results on 2026-11-12

The initial runs below used commit `9b6bd07e8`, before the NULL-projection and
arithmetic-sensitive fixture refinements described above.

The complete group passed with the race detector in both UTC and
America/Los_Angeles: 11 PASS nodes per configuration, no FAIL or SKIP. The same
handwritten JSON oracles passed on local PostgreSQL; they were not adjusted for
Snowflake. In each remote configuration the observer recorded 17 Unified-query,
18 explanation and two legacy-Record SELECT responses, all Arrow. Both verified
connections per configuration had the requested session parameters. Every run
had stable source/overlay hashes.

Each configuration created and dropped one schema. Separate read-only inventories
confirmed zero remaining schemas from each run, and both ledger audits passed.
The four remote JSONL logs contain none of the six Snowflake configuration values.
The evidence directory is
`/private/tmp/krenalis-unified-service-live-8wbl_pqf/`: per-zone manifests/logs and
overlays, `inventory/`, `credential-output-check.json`, the PostgreSQL baseline,
and the two local mutation checks. Both mutations were detected: increasing the
public Count uncertainty and truncating public update timestamps to milliseconds.
The ordinary, unoverlaid Snowflake test still skipped at the closed gate without
credentials. HTTP was verified separately below; MCP and the other acceptance
gaps remain open.

After review, the refined fixtures were rerun in both session timezones with the
race detector: 11 PASS nodes each, no FAIL/SKIP. The new projection fixes SQL NULL
as an omitted attribute while retaining its schema property; Sum/Average expose
the exact sum `170.1253`. Column conversion for `Initialize` now uses production
`util.PropertiesToColumns`. Each session explicitly requested ARROW through the
overlay, unlike current production configuration. Per configuration the same
17/18/2 application SELECT responses were observed as Arrow, with stable sources,
two verified connections, one created/dropped schema and a clean independent
inventory. The four new logs contain no Snowflake configuration values.

New evidence is under
`/private/tmp/krenalis-unified-service-review-live-ekuykqab/`: summary, per-zone
logs/manifests/overlays/audits, independent inventories and credential-output check.
The local `mutants/` check confirms that retaining SQL NULL attributes fails the
new projection golden. Removing nil attributes again still passes: Unified already
omits them before building the public response, so that mutation is a no-op.
Both inventories also saw 23 other test schemas; none was modified.

## Profiles HTTP endpoints

`TestProfilesHTTPPostgreSQL` and `TestProfilesHTTPSnowflake` share four profiles,
five recorded messages and handwritten full-response JSON expectations. They run
a real server, log in, create the workspace through `POST /v1/workspaces`, disable
identity resolution and use a workspace-restricted API key. No pipeline runs.
The Snowflake case uses the registered production provider, settings validation,
`CanInitialize`, `Initialize`, organization dialer and encrypted-settings loader.
Only the fixture writer receives its settings directly.

The HTTP cases cover Count money and UNKNOWN, Sum with minimum and exact decimal
explanations, full/descending/empty pages, projection with SQL NULL omission,
Decimal(38,10) beyond 2^53, nested JSON null, objects, arrays and microsecond
timestamps. A pure Record page has its own complete golden and no `evaluation`.
Missing-profile and oversized-page responses must be 404 and 422 respectively,
with the complete error body and no partial page. Every response checks status
and JSON Content-Type; bodies are compared as compact JSON bytes, without a
float64 decoding step.

The tester's `SetWarehouse` option copies settings per instance and skips only
the warehouse PostgreSQL container. The application PostgreSQL and NATS remain
disposable local containers. The normal tester still starts an external server;
`UsesInternalServer` derives from its existing launch constant. Snowflake skips
before credentials unless both the internal-server and gate overlays are present.
The HTTP overlay changes neither SQL nor handler behavior. It adds the three
service tags at the public core delegation sites, covering all downstream reads,
and reuses the existing session verifier and driver response observer.

Prepare real Admin assets outside the checkout; UI behavior is not tested. The
HTTP overlay disables automatic asset generation and uses that filesystem,
avoiding the tester's pre-existing asset-generation wait problem. Do not use the
`freshadminassets` build tag or run `go generate` in the checkout for this group.

```sh
# First verify the ordinary external server locally, without Snowflake settings.
KRENALIS_SKIP_SNOWFLAKE_TESTS=true KRENALIS_TESTS_ADDR=127.0.0.1:2027 \
  go test ./test -run '^TestProfilesHTTPPostgreSQL$' -race -count=1 -timeout=10m

http_campaign=$(mktemp -d /tmp/krenalis-unified-http-live.XXXXXX)
go build -mod=vendor -o "$http_campaign/admin-gen" ./admin
mkdir "$http_campaign/asset-root"
(cd "$http_campaign/asset-root" && "$http_campaign/admin-gen")
for http_timezone in UTC America/Los_Angeles; do
  python3 -B test/snowflaketester/prepare.py "$http_campaign/$http_timezone" \
    --timezone "$http_timezone" --result-format ARROW --http \
    --http-assets "$http_campaign/asset-root/admin/assets"
done

# Verify the internal build on PostgreSQL before the remote runs.
KRENALIS_SKIP_SNOWFLAKE_TESTS=true KRENALIS_TESTS_ADDR=127.0.0.1:2027 \
  python3 test/snowflaketester/run.py "$http_campaign/postgresql.jsonl" -- \
    -overlay "$http_campaign/UTC/overlay.json" ./test \
    -run '^TestProfilesHTTPPostgreSQL$' -race -timeout=10m

# Authorized live access and the existing credential environment are required.
for http_timezone in UTC America/Los_Angeles; do
  http_dir="$http_campaign/$http_timezone"
  KRENALIS_SKIP_SNOWFLAKE_TESTS=false KRENALIS_TESTS_ADDR=127.0.0.1:2027 \
    python3 test/snowflaketester/run.py \
    --require-schemas --require-session-timezone "$http_timezone" \
    --require-result-format ARROW \
    --require-result-tag krenalis-transport/service-query \
    --require-result-tag krenalis-transport/service-explanation \
    --require-result-tag krenalis-transport/service-record \
    "$http_dir/http.jsonl" -- -overlay "$http_dir/overlay.json" ./test \
    -run '^TestProfilesHTTPSnowflake$' -race -timeout=15m
done
```

Select exactly one server test per process and use a free dedicated port, never
the demo. Expected application reads per Snowflake configuration: seven Unified
SELECTs (Count, Sum, three pages, projection, page budget), three explanation
SELECTs (Count, Sum, missing profile), and two legacy Record SELECTs. Compare these
exact counts with the manifest; the runner enforces presence and format, not
exact counts. Bootstrap, fixture writes and session preflight have no service tag.

Run the ledger audit and independent inventory described above for each timezone.
Cleanup closes the fixture writer, then the server, then drops the schema. A
pre-existing container-startup panic can bypass test cleanup: in that case use
the ledger and inventory to identify the exact campaign schema before removing
it. Do not remove other visible test schemas.

### Live HTTP results on 2026-11-12

Both configurations passed with the race detector: nine PASS nodes each, no
FAIL/SKIP, stable source and overlay hashes. Each recorded exactly 7/3/2 SELECT
responses for query/explanation/Record, all Arrow, and five verified sessions
with the requested timezone, NTZ/AUTO, 60-second timeout and ARROW setting.
The same golden passed on PostgreSQL with both internal and ordinary external
startup, nine nodes each. The external baseline exposed a pre-existing race in
the tester's completion channel; creating it before launching the goroutine
fixed the race in both modes, and both complete runs passed again.

Each remote configuration created and dropped one schema. Separate inventories
and ledger audits confirmed no remaining campaign schema; the 23 other visible
test schemas were left untouched. None of the six Snowflake settings appears in
the four remote logs, including case and JSON-escaping variants. A local handler
mutant replacing only Record attributes with `{}` fails the new complete golden.
The overlay tests and all 37 Python campaign tests passed, as did the tester's
race-enabled tests and `go vet` for both changed Go packages.

Evidence is under `/private/tmp/krenalis-unified-http-live-j6mwdzkj/`: per-zone
HTTP logs/manifests/overlays/audits, `inventory/`, credential-output check,
PostgreSQL internal/external final logs, and the Record handler mutant. The gate
remains closed in production. MCP transport, UI, production native Arrow
configuration and the remaining activation checklist are outside this group.

After HTTP review, the tester creates each completion channel immediately before
launching its server goroutine, after all startup validation/build failures.
This preserves the race fix while leaving `Stop` free to clean up an unstarted
server. Both PostgreSQL modes passed again with `-race`, nine nodes each; forced
failures before internal startup and external build terminated within two seconds
and removed their containers. New local evidence and per-setting credential
match counts are in `/private/tmp/krenalis-http-startup-review-zvi_k6wy/`.
Those counts cover literal and JSON-escaped forms, case-insensitively, in all four
remote logs: zero matches. Snowflake was not rerun for this startup-only fix;
the live source/overlay hashes above refer to commit `e3d501268`.

## Profiles MCP transport

`TestProfilesMCPPostgreSQL` and `TestProfilesMCPSnowflake` reuse the HTTP group's
four profiles and five messages through `newProfileTransportFixture`. They run
the same real server and registered provider. A workspace-scoped MCP key created
through the API authorizes every tool; no separate warehouse credentials or
read-only user are needed. The SQL tool and both `readonlysql` packages have
been removed. The main connection determines data access, including for existing
keys formerly restricted by separate warehouse grants.

Five bounded cases exercise catalog/detail/guide, Count and its UNKNOWN
explanation, Sum with minimum and exact `170.1253` bounds, projection, and Record.
Pages and explanations compare complete JSON text with handwritten expectations,
without decoding numeric operands or attributes through float64. The projection
retains Decimal(38,10) beyond 2^53, nested JSON null, omitted SQL NULL with its
schema property, objects, arrays with empty strings, and microsecond timestamps.
The pure Record response has no `evaluation`.
Catalog/detail/guide checks assert selected identifiers and metadata, not complete
response JSON.

Authentication errors, revocation within an existing session, disabled
organizations, workspace isolation, full pagination, unsupported capabilities
and the 128 KiB MCP budget remain in the larger PostgreSQL `TestProfilesMCP`.
Legacy configuration rejection is covered by `TestWarehouse`. Do not repeat
those backend-independent cases during the reduced Snowflake window.

Use the HTTP overlay and real generated assets described above. The Snowflake
entry point checks both internal startup and the gate before accessing
credentials. A normal build must SKIP, not report a remote pass. Prepare fresh
overlays from the current source; old HTTP evidence does not verify this group.

```sh
# Ordinary external server, no Snowflake settings.
KRENALIS_SKIP_SNOWFLAKE_TESTS=true KRENALIS_TESTS_ADDR=127.0.0.1:2038 \
  go test ./test -run '^TestProfilesMCPPostgreSQL$' -race -count=1 -timeout=10m

mcp_campaign=$(mktemp -d /tmp/krenalis-unified-mcp-live.XXXXXX)
go build -mod=vendor -o "$mcp_campaign/admin-gen" ./admin
mkdir "$mcp_campaign/asset-root"
(cd "$mcp_campaign/asset-root" && "$mcp_campaign/admin-gen")
for mcp_timezone in UTC America/Los_Angeles; do
  python3 -B test/snowflaketester/prepare.py "$mcp_campaign/$mcp_timezone" \
    --timezone "$mcp_timezone" --result-format ARROW --http \
    --http-assets "$mcp_campaign/asset-root/admin/assets"
done

# Verify internal startup on PostgreSQL before the remote window.
KRENALIS_SKIP_SNOWFLAKE_TESTS=true KRENALIS_TESTS_ADDR=127.0.0.1:2038 \
  python3 test/snowflaketester/run.py "$mcp_campaign/postgresql.jsonl" -- \
    -overlay "$mcp_campaign/UTC/overlay.json" ./test \
    -run '^TestProfilesMCPPostgreSQL$' -race -timeout=10m

# Run only in an authorized live window, with the existing primary credentials.
for mcp_timezone in UTC America/Los_Angeles; do
  mcp_dir="$mcp_campaign/$mcp_timezone"
  KRENALIS_SKIP_SNOWFLAKE_TESTS=false KRENALIS_TESTS_ADDR=127.0.0.1:2038 \
    python3 test/snowflaketester/run.py \
    --require-schemas --require-session-timezone "$mcp_timezone" \
    --require-result-format ARROW \
    --require-result-tag krenalis-transport/service-query \
    --require-result-tag krenalis-transport/service-explanation \
    --require-result-tag krenalis-transport/service-record \
    "$mcp_dir/mcp.jsonl" -- -overlay "$mcp_dir/overlay.json" ./test \
    -run '^TestProfilesMCPSnowflake$' -race -timeout=15m
done
```

Select one server test per process and a free port, never the demo. Expect six
PASS nodes (parent and five cases) and exactly 3/2/2 SELECT observations for
query/explanation/Record. Nine tool calls cover the five cases; initialization,
catalog reads and fixture writes have no service tag. Inspect exact counts in
the manifest in addition to the runner's format/presence checks. Require stable
source/overlay hashes, the requested session parameters, schema-ledger audit,
credential-output masking and independent inventory as in the HTTP campaign.
Both session runs passed on 2026-11-13, as recorded in the live results below.
The production gate remains closed.

## Recorded event discovery

`TestEventDiscoveryPostgreSQL` and `TestEventDiscoverySnowflake` share stored
track/page/screen fixtures and dedicated reads from `krenalis_events`, without
using the customizable `events` view or the Unified gate. The six cases check:

- Literal distinct names under case-insensitive Snowflake column collation:
  case, composed/decomposed Unicode, null/empty names, `%`, `_` and backslash as
  data, long-name omission and frequent versus rare sources.
  Exact inspection also distinguishes both case and Unicode pairs by row count
  and distinct observed property paths.
- Exact inspection of unnamed and empty-named pages.
- Nested object paths, arrays without traversal and an empty sample.
- SQL NULL, JSON null and all six observed JSON kinds.
- UTF-8 roots of exactly 64 KiB and 64 KiB + 1, the 8 MiB cumulative prefix,
  unread roots, the row cap and payload-free control rows.
- Cancellation after an observer sees an active source `DISTINCT`, no partial
  sources, server termination and recovery on the same warehouse. A volatile
  delay function bounds the fixture instead of requiring a large dataset.

The source case also checks the inclusive start, exclusive end and the preceding
microsecond. `TestEventDiscoveryMCP` checks both real tools, closed arguments,
text/structured equivalence and key revocation on PostgreSQL. The reduced
`TestEventDiscoveryMCPSnowflake` checks both tools on the five-message transport
fixture; it does not repeat the backend-independent authentication cases.

Run the following as one discovery campaign in an authorized window. Prepare
fresh overlays and assets as in the Profiles MCP section; reuse its
`mcp_campaign` directories only when their source hashes still match. The
standard UTC/ARROW HTTP overlay also enables the Unified gate for other tests,
but discovery does not depend on that change. The production gate stays closed.

```sh
# Local prerequisite, including the internal-server test on a separate port.
KRENALIS_SKIP_SNOWFLAKE_TESTS=true KRENALIS_TESTS_ADDR=127.0.0.1:2038 \
  python3 test/snowflaketester/run.py "$mcp_campaign/discovery-postgresql.jsonl" -- \
    -overlay "$mcp_campaign/UTC/overlay.json" -p=1 ./core/internal/datastore ./test \
    -run '^TestEventDiscovery(PostgreSQL|MCP)$' -race -timeout=10m

# Both new Snowflake entry points in the same bounded campaign.
KRENALIS_SKIP_SNOWFLAKE_TESTS=false KRENALIS_TESTS_ADDR=127.0.0.1:2038 \
  python3 test/snowflaketester/run.py \
    --require-schemas --require-session-timezone UTC \
    "$mcp_campaign/UTC/discovery.jsonl" -- \
    -overlay "$mcp_campaign/UTC/overlay.json" -p=1 ./core/internal/datastore ./test \
    -run '^TestEventDiscovery(Snowflake|MCPSnowflake)$' -race -timeout=10m
```

Expect eight PASS nodes remotely (the datastore parent, its six cases and the
MCP parent), with zero SKIP. Each fixture uses a disposable schema, ledger and
cleanup. Require stable source/overlay hashes, session/schema audits,
credential-output masking and independent inventory as in the transport
campaign. These reads project only text; this group does not add driver result
format observation tags. Requested ARROW settings alone do not prove the actual
format returned for discovery.

The cancellation case reuses the interruption observer. On PostgreSQL it
requires a sleeping target and its disappearance before the delay can complete;
on Snowflake it requires the tagged running query and terminal code 604.
The existing Unified interruption run is not evidence for this new SELECT.
Local virtual-time tests separately cover the 30-second datastore deadline and
discarding a result after late cancellation. Remote execution, including this
new cancellation case, must pass before merging discovery.

Discovery describes recorded values and does not certify equivalent Count
selection under nondefault collation. Record that comparison in the separate
collation group; do not alter discovery expectations to match Count. No merge
or gate-opening decision follows from PostgreSQL-only evidence.

### Live discovery and MCP results on 2026-11-13

Commit `2e74bfb81` passed the recorded-event discovery campaign and the pending
Profiles MCP group. No production code or expected values changed during these
runs. Fresh overlays and generated assets were kept outside the checkout.

| Group | Session | PASS | FAIL / SKIP |
| --- | --- | ---: | --- |
| Discovery datastore + MCP, PostgreSQL prerequisite | local | 14 | 0 / 0 |
| Profiles MCP, PostgreSQL prerequisite | local | 6 | 0 / 0 |
| Discovery datastore + MCP, Snowflake | UTC | 8 | 0 / 0 |
| Profiles MCP, Snowflake | UTC | 6 | 0 / 0 |
| Profiles MCP, Snowflake | America/Los_Angeles | 6 | 0 / 0 |

The discovery run verifies exact inspection of case and composed/decomposed
Unicode pairs under `en-ci`, distinct null/empty names, literal substring
searches, JSON kinds, multibyte and cumulative byte limits. Its running-query
cancellation ended with server code 604 after 4,916 ms, before the fixture's
30-second delay could finish, and the recovery read passed.

Both Profiles MCP runs compare exact page, explanation and projection JSON.
Count includes UNKNOWN; Sum preserves `170.1253`; projection preserves the
Decimal(38,10), JSON null, omitted SQL NULL and array empty-string cases.
Catalog/detail/guide checks remain selected metadata checks, not full JSON
comparisons. Each session observes exactly 3/2/2 SELECT responses for
service-query/service-explanation/service-record, all ARROW. Discovery has no
result observation tags, so its requested ARROW setting is not an attestation
of the actual result format for those reads.

All seven runs, including the two read-only inventories, report stable
source/overlay hashes. Session records verify UTC or America/Los_Angeles as
requested, NTZ/AUTO, the standard profile and its 60-second statement timeout.
The schema audit balances four creations and four drops; the independent final
inventory finds zero remaining campaign schemas. Other test schemas were left
alone. Credential-output audits for literal, JSON, HTML-escaped and account-dash
forms report zero matches in the selected logs, manifests and runner output.

The schema audit covers the five application test manifests at the campaign root;
the two read-only inventories have manifests in separate subdirectories.

Evidence: `/private/tmp/krenalis-discovery-snowflake-j78h_lan/`, including
`live.jsonl`, `mcp-utc.jsonl`, `mcp-la.jsonl`, their manifests, `summary.json`,
`schema-audit.json`, `inventory-final/inventory.jsonl` and per-run output audits.
The runtime was Go 1.27.3 with the vendored Krenalis-patched gosnowflake v2.3.0.

The pre-merge live condition for discovery is satisfied for this commit.
The broader Record/collation/Float-NaN and fault/resource checklist below remains
open; neither this campaign nor the test-only gate overlay enables the
production Unified gate.

## Record, collation and native Float campaign

The shared fixtures `TestUnifiedRecordConformancePostgreSQL/Snowflake`,
`TestUnifiedRecordSourcesPostgreSQL/Snowflake` and
`TestUnifiedRecordCollationPostgreSQL/Snowflake` extend the campaign without
opening the production gate. Run PostgreSQL first, then the full UTC block and
a reduced America/Los_Angeles block. Each remote block creates and tears down
its own test schemas; do not run it against an application or demo schema.

- The full Record matrix uses 117 native cases, including 34 SQL-specific Float
  cases, and all 118 JSON cases. Native cases share two rows and one column per
  case. Explanations check each leaf on both the present and absent row; their
  explicit SQL expectations do not use Go's boolean evaluator. JSON predicates
  alternate between a scalar column and a path with literal dot, quote and
  backslash keys, checking all 41 fixture outcomes and all ternary totals.
- The matrix covers all 26 original operators. Its successful application
  reads require 161 Unified SELECTs, 46 explanations and 20 legacy queries
  (two SELECTs each), for 247 SELECTs, excluding initialization and writes.
  It also checks empty pages and pure Record after dropping the event table.
- `TestUnifiedRecordStatesPostgreSQL/Snowflake` separately checks the six
  native state operators on SQL NULL, empty and populated String/Array values.
  Each query and all three explanations assert both leaves and their group:
  six queries and eighteen explanations, or 24 extra SELECTs. Native `is null`
  is exercised at the prepared physical boundary; public admission still
  requires a nullable property and is unchanged. JSON state coverage alone is
  not evidence for these native renderer branches.
- Source fixtures initialize the actual Krenalis tables. On Snowflake they set
  `DEFAULT_DDL_COLLATION='en-ci'` only on the generated temporary schema before
  initialization. PostgreSQL keeps its enum `type` and sets non-deterministic
  ICU collation only on event/name. They verify Count, Sum, Average,
  First/Last/Nth, Latest N and LastVersusPrevious, named/unnamed page/screen,
  unknown track names, Unicode spellings and unchanged raw `where` collation.
  They add 16 successful queries and 15 explanations.
- Collation fixtures compare explicit binary, English and case-insensitive
  columns, then the four decisive Record/Count Kleene compositions and invalid
  Real projections. PostgreSQL 18 supports POSITION with non-deterministic
  collations; this is not an expected error on the pinned PostgreSQL 18 image.
  Snowflake alone probes a CONTAINS-incompatible punctuation-insensitive
  collation as an execution failure, with no partial result.

The three new closed tags `krenalis-transport/record-query`,
`record-explanation` and `record-legacy` identify **datastore** reads. They do
not attest an HTTP/MCP or public core call. The overlay observes actual server
response formats for these tags; do not reuse the public service tags or infer
the actual format from the requested one.

Prepare fresh `--result-format ARROW` UTC and LA overlays using `prepare.py`,
then run the blocks below with the established credential environment. Keep
credentials outside logs and manifests. The runner never sources `.env` files.

```sh
python3 test/snowflaketester/run.py "$campaign_dir/record-local.jsonl" -- \
  ./core/internal/datastore -race \
  -run '^TestUnifiedRecord(Conformance|Sources|Collation|States)PostgreSQL$' -timeout=5m

python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone UTC --require-session-profile standard \
  --require-result-format arrow \
  --require-result-tag krenalis-transport/record-query \
  --require-result-tag krenalis-transport/record-explanation \
  --require-result-tag krenalis-transport/record-legacy \
  "$campaign_dir/record-utc.jsonl" -- \
  -overlay "$campaign_dir/record-utc-overlay/overlay.json" ./core/internal/datastore -race \
  -run '^TestUnifiedRecord(Conformance|Sources|Collation|States)Snowflake$' -timeout=20m

python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone America/Los_Angeles --require-session-profile standard \
  --require-result-format arrow \
  --require-result-tag krenalis-transport/record-query \
  --require-result-tag krenalis-transport/record-explanation \
  --require-result-tag krenalis-transport/record-legacy \
  "$campaign_dir/record-la.jsonl" -- \
  -overlay "$campaign_dir/record-la-overlay/overlay.json" ./core/internal/datastore -race \
  -run '^TestUnifiedRecord(Conformance|Collation)Snowflake$/^(native|false_AND_unknown|true_OR_unknown|unknown_AND_true|unknown_OR_false)$/^(0|6|8|9|10|14|15|16|17|18|19)$' \
  -timeout=10m
```

The reduced block selects temporal equality, interval, array containment and
before/after families, plus prefix/Boolean families and the four compositions.
The fixture still performs its bounded setup and final empty-page/event-table-independence checks. Filtered
subtests are excluded by `-run`, not skipped at runtime.

Before live execution, record the successful local counts and inspect the
query budget lines. Afterwards record observed formats, schema inventory,
source/overlay hashes, output audit and any differences. A green campaign alone
does not approve a native-text policy. The user's subsequent explicit exception
is recorded below; the Record renderer remains legacy-compatible.
NaN/infinites in native Float columns retain the
SQL legacy ordering; pipeline Go evaluation is a separate existing path.
Neither the JSON numeric contract nor the known native JSON transport failure
is changed by these tests.

### Live Record results (2026-11-13)

Code and fixture commit: `025a4347f7711ddf8025c52bc6088de4dd9f03f6`.
Evidence directory: `/private/tmp/krenalis-record-snowflake-h3l2qrte/`.

| Run | PASS | Result |
| --- | ---: | --- |
| `record-local` | 177 | complete PostgreSQL matrix, source and collation fixtures, with race detector |
| `record-la-local` | 17 | local check of the reduced selector |
| `record-utc` | 178 | complete Snowflake group, with the additional unsupported-collation failure case |
| `record-la` | 17 | reduced Snowflake group |
| `inventory-after/inventory-verified` | 1 | independent read-only inventory, five campaign schemas absent |

There were no FAIL or SKIP events in these five accepted runs. Each records
`source_stable=true`, the code commit above, and unchanged source/overlay hashes.
All 385 tracked source hashes match that commit. The foreign untracked root
`connectors.go` is unchanged and not part of the tested package dependency tree.

Actual SELECT response formats are ARROW throughout:

| Tag suffix | UTC SELECTs | America/Los_Angeles SELECTs |
| --- | ---: | ---: |
| `record-query` | 198 | 29 |
| `record-explanation` | 65 | 26 |
| `record-legacy` | 40 | 22 |

The UTC run therefore records 303 successful tagged application SELECTs, plus
two expected untagged execution failures (unsupported CONTAINS collation and
non-finite Real projection). The reduced run records 77 tagged SELECTs and the
Real projection failure. These are datastore/warehouse results; no new public
HTTP/MCP execution is claimed. The successful query/explanation formats and
requested ARROW session settings are independently observed, with no invalid
records. UTC opened six verified standard sessions, LA four.

The reduced command used a suffix expression that also matched native families
10, 18 and 19. They are prefix and Boolean families; all passed. The command
above now names the same eleven families explicitly with anchors, so future
runs repeat the actual covered set without relying on suffix matching.

The schema ledger has five creates and five drops (three UTC, two LA), no
orphans and no audit issues. A preliminary read-only baseline counted 23
pre-existing test schemas; the final independent inventory found those same
23 visible test schemas and zero remaining campaign schemas. Foreign schemas
were not altered. The baseline and first final-inventory wrapper incorrectly
set `KRENALIS_SKIP_SNOWFLAKE_TESTS=false`, making the runner demand schema
creation from read-only tests: their Go tests passed, but their manifests have
exit 4. The corrected final run leaves that variable unset and passes the
runner too. The failed wrappers are retained in the evidence subdirectories;
they are not accepted application runs or hidden failures.

All credential-value/form output audits are zero. Four isolated local mutants
were detected: removal of exact collation, removal of page/screen name
presence, a negative predicate accepting NULL, and native NaN replaced by NULL.
The Record renderer is unchanged, including native collation and SQL NaN
semantics. Source type/name selectors are now literal under inherited `en-ci` and the
PostgreSQL enum/ICU setup. Native-text policy was pending at the time of this
run; the approved exception is recorded below. The remaining activation checks
and the known native JSON transport issue remain separate. The production gate
is still false.

### Native state coverage follow-up (2026-11-13)

Review R1 identified that the six state operators had only JSON coverage.
Commit `f949bf02cf6731f8d858906c30212cd60a159a60` adds the native String/Array
matrix above, fixes source-selector block spacing and makes the named
UNKNOWN OR FALSE composition actually use a false Count on profile 1.
The explanation asserts that Count outcome, so restoring the old UNKNOWN
input is detected even though the root outcome remains UNKNOWN.

Evidence directory: `/private/tmp/krenalis-record-states-5oulagik/`.

| Run | PASS | Scope |
| --- | ---: | --- |
| `local` | 184 | all four PostgreSQL groups, race detector |
| `utc` | 12 | native states and four compositions on Snowflake |
| `la` | 5 | four compositions on Snowflake America/Los_Angeles |
| `inventory/inventory` | 1 | independent read-only inventory |

All four runs passed without FAIL or SKIP, with stable sources and overlays.
The 386 tracked source hashes in the three application manifests match the
code commit; the unchanged foreign `connectors.go` remains excluded from the
tested dependency tree. Actual ARROW SELECT counts are 10 query/22 explanation
in UTC, and 4/4 in LA. There are no legacy reads in these supplemental blocks.
UTC verifies four standard sessions and LA two, all NTZ/AUTO, requested ARROW
and 60-second statement limits, with no invalid records. The expected untagged
Real projection failure still executes in each collation fixture.

Two UTC schemas and one LA schema were created and dropped; the independent
inventory found none of these three and 23 other visible test schemas. The
schema audit reports no issues, and all three credential output audits are
zero. The inventory wrapper leaves the skip variable unset and passes without
schema-creation requirements. The full earlier matrix remains attested by the
original campaign; only the added coverage and changed composition were
repeated remotely.

Repeat the UTC supplement with the standard UTC ARROW overlay and both
`record-query`/`record-explanation` required tags, selecting:

```sh
-run '^TestUnifiedRecord(States|Collation)Snowflake$/^(empty|not_empty|null|not_null|exists|does_not_exist|false_AND_unknown|true_OR_unknown|unknown_AND_true|unknown_OR_false)$'
```

For the LA supplement, use the LA overlay, the same two tags and:

```sh
-run '^TestUnifiedRecordCollationSnowflake$/^(false_AND_unknown|true_OR_unknown|unknown_AND_true|unknown_OR_false)$'
```

Local isolated mutants replacing native array emptiness with `length=0`,
inverting native absence, and restoring the old composition input all fail
the new tests. The last is a fixture mutation, not a production-code mutation.
Message-identity collation (N1) was separate from this follow-up; its reader
correction and evidence are recorded below. No Record
renderer behavior or public filter admission changed; the gate stays false.

## Literal message identities (N1)

The shared compiler now fixes the collation of message identity in Count,
Sum/Average, First/Last/Nth, Latest N and LastVersusPrevious. Exact expressions
are used in identity projections and on both sides of each identity `IN`,
including the target-profile explanation seed. The derived grouping therefore
keeps byte-distinct IDs separate. NULL diagnostics, genuine-copy deduplication,
source fault rules, profile UUID comparisons and user Record rendering are
unchanged. Query SQL changes intentionally; timestamp arguments do not.

`TestUnifiedIdentityPostgreSQL/Snowflake` shares ten cases under PostgreSQL
non-deterministic ICU and Snowflake `en-ci-rtrim`. Controlled DML, rather than
Merge, preserves every physical fixture row, and a row count checks this before
each read. Cases cover case variants, trailing space, composed/decomposed
Unicode, determined order and ties, identical copies, distinct profile/period/
qualification, real copy conflicts and another profile's fault. Each complete
explanation checks all nine nodes and applicable counts, exact sums, Average
inputs, candidate/absence and pair metadata. Both profiles are explained in
non-fault cases. The isolated target explanation succeeds even when the full
population query fails for the other profile's genuine fault.

Live evidence is in `/private/tmp/krenalis-identity-verify-r6psxclh/`, using the
standard test-only gate/session/result overlays. The source base is `8b37e061f`
plus the four compiler changes and three new test files, hashed in the manifests.
No driver or session change is part of production code.

| Run | Result | Marked SELECTs |
| --- | --- | --- |
| `local-unit-complete` | 673 PASS with race detection | Local profile compiler, SQL and read tests; the two remote tests matched by the broad prefix are excluded explicitly. |
| `postgresql-regression` | 527 PASS with race detection | Local Identity, Conformance, RecordSources, Source, ProfileRecord and Pages groups. |
| `utc` | 11 PASS | 10 query and 17 explanation SELECTs, all observed Arrow. |
| `la` | 4 PASS | Two identity cases, their parent and a private DDL probe; 2 query and 4 explanation SELECTs, all observed Arrow. |
| `inventory/inventory` | 1 PASS | Three owned schemas created/dropped, none remaining; 23 other test schemas visible. |

UTC has two verified sessions; LA has four, including the private DDL probe.
All use NTZ/AUTO, 60-second statement limits and requested ARROW. The schema
audit also includes local mutants and plan probes, which create no Snowflake
objects. No resource-monitor or credit configuration changed.

Repeat the identity tests with fresh logs after loading the authorized test
environment, without printing its values:

```sh
python3 -B test/snowflaketester/prepare.py "$campaign_dir/identity-utc" --timezone UTC --result-format ARROW
KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone UTC --require-session-profile standard --require-result-format arrow \
  --require-result-tag krenalis-transport/record-query \
  --require-result-tag krenalis-transport/record-explanation "$campaign_dir/identity-utc.jsonl" -- \
  -overlay "$campaign_dir/identity-utc/overlay.json" ./core/internal/datastore -race \
  -run '^TestUnifiedIdentitySnowflake$' -timeout=8m
```

The reduced LA identity selection is
`^TestUnifiedIdentitySnowflake$/^(tied_identities|other_profile_fault)$`, with
the LA overlay and required timezone. Its repeat does not include the private
DDL probe, so it produces three passing nodes rather than four.

Four local mutants restore native identity independently in each compiler
constructor; the PostgreSQL fixture detects all four. A fifth restores only
the native left operand of the copies `IN`; the mirrored SQL contract test
detects that partial form. This last check is structural, not a claim that the
PostgreSQL integration fixture rejects every partially collated expression.
Claude's independent run of this specific partial mutant also fails in the
PostgreSQL fixture with SQLSTATE 42P22. Its Snowflake behavior is not verified.

A private PostgreSQL EXPLAIN/ANALYZE comparison used 128 profiles and 16,384
events with a deterministic non-C identity index. The parent explanation used
128 primary-key searches; the corrected explanation scanned the event table.
Observed server execution was 2.424 ms versus 3.826 ms. These are single smoke
observations, not production performance estimates or a reason to weaken
identity semantics.

The private DDL probe also confirmed an activation issue outside this reader
fix. Under schema default `en-ci-rtrim`, an unspecified column collation and
`COLLATE ''` both inherit the non-binary setting; explicit `COLLATE 'utf8'`
keeps case-distinct IDs separate. INFORMATION_SCHEMA.COLUMNS.COLLATION_NAME
reported those settings. Changing the inherited column to an uncollated
VARCHAR with ALTER COLUMN was rejected with code 040053, incompatible collations.
This does not establish a general migration procedure. Merge and identity
resolution SQL remain outside this correction and need a separate decision
before activation, as listed below. The production gate stays false.

## Failure and resource supplement

`TestUnifiedResourcesPostgreSQL/Snowflake` shares the physical budget, server
failure and bounded-load cases. It checks one oversized String, a cumulative
two-row overflow, exact 1 MiB success, UTF-8 overflow and exact UTF-8 success.
After each size rejection, a request without the oversized projection must
succeed on the same pool. A warehouse-owned view with a malformed profile key
must fail without returning a partial result.

For both membership and explanation, a temporary profile view raises a real
division-by-zero error. The datastore must preserve its opaque failure and no
result; the test checks the internal server SQLSTATE 22012. Restoring the view
must then produce the exact reference result on the same warehouse/pool. This
tests reuse after a server error, not reuse of one particular physical session.

The load smoke test uses 128 profiles and 16,384 messages with eight rules.
It checks a final matching page, an empty page with retained totals, all nine
explanation outcomes and exact Count/Sum/Average metadata. It is not a benchmark.

`TestUnifiedTimestampPrecisionPostgreSQL/Snowflake` checks Count, First and Last
on both sides of each half-open border and exactly on it, with one stored tick
of separation. Both warehouses run native precisions 3 and 6; only Snowflake
runs precision 9 because PostgreSQL cannot store nanosecond timestamps. This
does not extend the managed Snowflake NTZ schema to LTZ or TZ.

`TestModeTransitions` exercises the local coordinator with synctest: reads stay
live in inspection, maintenance waits or cancels according to the requested
policy, new admissions wait for the transition and then fail, and normal-mode
admission recovers. There is no network I/O in the virtual-time test.

Evidence is in `/private/tmp/krenalis-resources-j2ux3go2/`:

| Run | Result | Scope |
| --- | --- | --- |
| `postgresql` | 460 PASS with race detection | Initial resource/precision/mode tests and Identity/Conformance regressions. |
| `local-final` | 19 PASS with race detection | Final resource metadata checks, native precision 3/6 and four mode transitions. |
| `utc-final` | 15 PASS with race detection | Final shared Snowflake cases; 15 query and 5 explanation SELECTs, all Arrow. |
| `la` | 8 PASS with race detection | Cumulative limit, exact UTF-8 budget, server failures/recovery and precision 9; 5 query and 2 explanation SELECTs, all Arrow. |
| `failure` | 1 PASS with race detection | Private real-driver chunk fault and same-pool recovery, described below. |
| `inventory/inventory` | 1 PASS | Eight owned schemas created/dropped, none remaining; 23 other test schemas visible. |

UTC/LA each verified four standard sessions; the private fault probe verified
two UTC sessions. They use NTZ/AUTO, 60-second statement limits and requested
ARROW. The earlier `utc` also passed 15 nodes before the final scale-metadata
assertions. Four local mutants are detected: reject the exact byte boundary,
return success after a backend error, omit mode cancellation, and truncate
timestamp operands to milliseconds. No production source was changed.

Repeat the permanent tests with fresh logs and standard overlays:

```sh
KRENALIS_SKIP_SNOWFLAKE_TESTS=true go test ./core/internal/datastore -race \
  -run '^Test(ModeTransitions|Unified(Resources|TimestampPrecision)PostgreSQL)$' -count=1 -timeout=4m

KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone UTC --require-session-profile standard --require-result-format arrow \
  --require-result-tag krenalis-transport/record-query \
  --require-result-tag krenalis-transport/record-explanation "$campaign_dir/resources-utc.jsonl" -- \
  -overlay "$campaign_dir/utc-overlay/overlay.json" ./core/internal/datastore -race \
  -run '^TestUnified(Resources|TimestampPrecision)Snowflake$' -timeout=8m
```

The reduced LA selector is
`^TestUnified(Resources|TimestampPrecision)Snowflake$/^(cumulative_oversized|UTF8_exact_budget|server_error_and_recovery|precision-9)$`,
using the LA overlay and required timezone. The permanent suite does not enable
driver fault injection. The private observation overlay adds counts only and
does not change production, vendored sources, result formats or SQL.

### Remote chunks are reachable within the budget

Repetitive payloads at the exact byte budget arrived inline in these runs.
A separate private probe used 512 profiles with 2,048-byte SHA-256 digest
payloads each, exactly 1 MiB in total; it verified every byte and profile ID.
Snowflake returned 48 inline rows and three remote chunks. Their consumed
response-body bytes before the driver's chunk decompression were 309,536,
513,968 and 173,456, with EOF observed and no close errors. These are body
reads after the HTTP transport, not total or compressed network traffic;
the inline base64 length and advertised uncompressed sizes are
recorded separately. No URLs, headers or document content were logged.

The initial `entropy` test passed, but its wrapper exited 7 because that
query-only probe incorrectly required an explanation tag as well. It is
retained as diagnostic evidence, not counted as a fully verified run. The
subsequent `failure` run required only the query tag and exited zero.

The fault probe uses external overlays of the actual vendored driver and an
external Go test. It waits for the first delivered row, then fails download
of the second remote chunk. The driver's existing context-cancellation marker
stops retries; the caller context is not canceled. Only this test request has
the private context marker. Its counts show 48 delivered rows, three remote
chunks, one injected fault and a nil datastore result with an opaque error.
The same pool subsequently returns all 512 profiles and all 1 MiB of exact
payloads; that recovery used four remote chunks and no inline batch.

This is a driver-level fault with real data, not an actual network outage or
a caller cancellation. Its error logging is sanitized in the overlay to avoid
printing a signed chunk URL. The row counter excludes session-verification
queries and counts driver delivery, not completed application decoding.
The vendored call chain is `connection.go:446-484` (execution and downloader
startup), `connection_util.go:373-396` (inline data and chunk descriptors),
`chunk_downloader.go:82-145,200-234,410-446` (initial batch, prefetch and remote
body reads), and `rows.go:224-250` (driver delivery), all under
`vendor/github.com/snowflakedb/gosnowflake/v2/`.
No fault hook, private driver API or test bypass is compiled into production.
The probe does not test chunk faults in explanations or HTTP/MCP responses.
Permanent scan/iteration/Close error tests with controlled cursors remain
separate evidence. A repeatable repository workflow for the private chunk
probe is now provided below, with the retry-exhaustion path replacing the
private cancellation marker.

### Repeatable chunk checks

The shared `TestUnifiedResourcesPostgreSQL/Snowflake/chunk_payload` case now
loads the same 512 digest payloads and verifies every ID and byte at the exact
1 MiB budget. The normal observer generated by `prepare.py --result-format`
also prints the number of remote chunk descriptors and the statement kind for
the closed set of fixture tags. It prints no URL, header or document content.

`run.py --require-remote-chunks TAG` requires at least two descriptors for a
SELECT with that tag, alongside the matching format requirement. Missing,
malformed or insufficient evidence fails an otherwise successful block with
exit 8. The manifest records only bounded counts per known tag, observed before
redaction. These are response metadata, not a count of completed downloads:
the normal case's complete payload assertion establishes successful reading.
If Snowflake changes the layout to inline-only, the block must fail rather than
silently stop exercising remote chunks.

For the fault case, `prepare.py --chunk-fault` generates three driver patches
and a support file outside the checkout. The generator requires unique anchors
in the current vendor and fails if they disappear. The support and datastore
fault test require `-tags krenalis_chunkfault`; the tag and the matching overlay
are both necessary. Neither enters a normal build. ARROW and the standard
profile are required; the fault overlay cannot be combined with the HTTP overlay.

The marked request fails the second remote chunk after the first delivered
row, on every attempt, with a plain test sentinel. The driver exhausts its
existing retry limit, currently five retries after the initial attempt.
The caller context stays live. The test requires the sentinel inside the opaque
datastore failure, a nil result, positive row delivery, at least two remote
chunks, and exactly the permitted attempts. A subsequent unmarked query on the
same pool must return all 512 profiles and all payload bytes. The marked error
log omits the chunk URL. This remains a driver-level injection, not a real
network outage, HTTP/MCP test or explanation-chunk failure test.

Run the following only in an authorized live window, using fresh directories
outside the checkout and credentials already supplied in the environment:

```sh
python3 -B test/snowflaketester/prepare.py "$campaign_dir/chunks" \
  --timezone UTC --result-format ARROW
KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone UTC --require-session-profile standard --require-result-format arrow \
  --require-result-tag krenalis-transport/record-query \
  --require-remote-chunks krenalis-transport/record-query "$campaign_dir/chunks.jsonl" -- \
  -overlay "$campaign_dir/chunks/overlay.json" ./core/internal/datastore -race \
  -run '^TestUnifiedResourcesSnowflake$/^chunk_payload$' -timeout=5m

python3 -B test/snowflaketester/prepare.py "$campaign_dir/chunk-fault" \
  --timezone UTC --result-format ARROW --chunk-fault
KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone UTC --require-session-profile standard --require-result-format arrow \
  --require-result-tag krenalis-transport/record-query \
  --require-remote-chunks krenalis-transport/record-query "$campaign_dir/chunk-fault.jsonl" -- \
  -overlay "$campaign_dir/chunk-fault/overlay.json" -tags krenalis_chunkfault ./core/internal/datastore -race \
  -run '^TestUnifiedChunkFaultSnowflake$' -timeout=5m
```

The Python preparation tests run the actual driver's retry path locally with
synctest and fake rows, including the wait for first delivery and request-marker
isolation. Runner tests check fragment reassembly, malformed records, redaction
and preservation of an existing failure. These local checks use no credentials
or network and complement the two remote commands.

Local evidence on 2026-11-13 is in
`/private/tmp/krenalis-chunks-periodic-jjp6bcai/`: 44 Python tests pass,
`local-final` has 15 PostgreSQL PASS with race detection, and `driver-local`
passes the actual retry path with fake rows and synctest. The two driver
mutants, stopping at the first error and injecting before row delivery, each
fail the intended assertion. Normal and tagged-overlay vet checks pass; the
normal build excludes the tagged fault test.

The first `local` run was refused by the sandbox's Docker permissions and is
replaced by `local-final`. The live normal probe (`live-chunks`) was refused
at authentication with Snowflake error 390422, before schema creation or any
application SELECT. It is not a remote pass. The new normal and fault commands
remain pending until the next authorized live window; the fault command was
not attempted. The earlier private probe's remote success does not certify
the new retry-exhaustion path.

## Approved native-text policy (2026-11-13)

The user approved an explicit exception to executor invariance for ordinary
Record value comparisons on native String columns:
[07 §14.4.1](../../filtering/07-ast-canonico-riconciliato-v1.2.md#1441-deroga-approvata-per-il-testo-nativo).
The existing column comparison rules remain in use. Outcomes can differ between
warehouses or after those rules change, with the same Krenalis type; Krenalis
does not detect, store, verify or version the rules. This is a declared exception,
not evidence of invariant native-text results across warehouses.

Technical source selectors, internal message identity, JSON string comparisons
and governed currencies keep their literal contracts. Ordinary Record predicates
on native String identity/name columns still follow the column's rules. A
warehouse comparison error remains a technical failure. Renderer behavior,
null/missing semantics and Kleene composition are unchanged.

The bounded Record/collation/Float-NaN campaign above passed on both warehouses.
Source cases beyond the shared fixtures are not certified by that result. The
added groups provide remote evidence for Count money, Sum/Average where and
minimum, covered conflicts, mixed explanations, projections and pagination.
Approving the exception settles this policy choice; it does not open the gate
or replace the pending live chunk checks and other acceptance work below.

## Coverage still required before activation

Passing the commands above is necessary, but is not the complete acceptance
checklist. Record actual results against each block in the Unified README.
In particular, the following work remains explicit:

- Decide how native FLOAT/BOOLEAN compatibility with JSON results affects the
  supported transport contract. Nested numeric fixtures and the Arrow native
  path passed above; the failed JSON native probe does not certify equivalence.
- The failure/resource supplement above covers limits, a malformed identity,
  real server errors and recovery, a driver-level chunk fault, native timestamp
  precisions, local mode changes and a bounded load smoke test. Controlled
  cursors cover scan/iteration/Close failures. Keep their levels of evidence
  distinct; the repeatable chunk workflow above remains separate from real
  transport faults and explanation/HTTP/MCP chunk coverage.
  Running-query timeout/cancellation evidence is recorded separately below.
- Decide how managed identity columns retain literal identity in writes and
  attribution under non-binary collation, including existing warehouses.
  The N1 reader correction above prevents query-side identity merging, but
  does not repair Merge or identity-resolution comparisons, or recover rows
  already replaced. Both are pre-existing behavior outside Unified; a schema
  guarantee with detection or explicit local corrections remains a separate
  product/upgrade decision before activation. A detected copy conflict is a
  technical failure, not semantic UNKNOWN.
- Timestamp input formats beyond AUTO and environment-level resource controls.
  UTC and America/Los_Angeles with NTZ/AUTO and per-session 60-second statement
  limits are covered above; these runs do not verify resource-monitor policies
  or credit limits.

No gate change follows automatically from a green reduced suite. Resolve observed
differences, finish the acceptance checklist and keep a repeatable remote
regression subset before deciding activation. The deferred infrastructure work
and support for additional native numeric types remain separate.

## Running-query timeout and cancellation

The interruption fixtures replace the event source with a view over a single
Merge-written event. A volatile UDF delays reads without a large dataset.
Membership and explanation must fail without partial results, stop on the
server, and then succeed with exact reference results on the same warehouse
once the view is restored. This checks interruption, not query-plan performance.

Prepare separate UTC overlays with `prepare.py --profile timeout` and
`prepare.py --profile cancellation`. The timeout profile uses a 10-second
statement limit; cancellation retains 60 seconds. Both disable cached results.
Every new physical connection verifies these settings. Require
`--require-session-profile timeout` or `cancellation` alongside the timezone
option in `run.py`; missing, inconsistent or mixed profile evidence fails the
block. The standard profile remains the default and accepts legacy session
records without an explicit profile.

Run the two remote blocks sequentially in fresh overlay directories:

```sh
python3 -B test/snowflaketester/prepare.py "$campaign_dir/timeout-overlay" --timezone UTC --profile timeout
python3 -B test/snowflaketester/prepare.py "$campaign_dir/cancellation-overlay" --timezone UTC --profile cancellation

KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone UTC --require-session-profile timeout "$campaign_dir/timeout.jsonl" -- \
  -overlay "$campaign_dir/timeout-overlay/overlay.json" ./core/internal/datastore \
  -run '^TestUnifiedTimeoutSnowflake$' -timeout=4m

KRENALIS_SKIP_SNOWFLAKE_TESTS=false python3 test/snowflaketester/run.py --require-schemas \
  --require-session-timezone UTC --require-session-profile cancellation "$campaign_dir/cancellation.jsonl" -- \
  -overlay "$campaign_dir/cancellation-overlay/overlay.json" ./core/internal/datastore \
  -run '^TestUnifiedCancellationSnowflake$' -timeout=4m

# Local counterpart; no Snowflake credentials are used.
KRENALIS_SKIP_SNOWFLAKE_TESTS=true go test ./core/internal/datastore \
  -run '^TestUnified(Cancellation|Timeout)PostgreSQL$' -count=1 -timeout=3m
```

A distinct observer session reads query history by a unique context query tag.
Its own tag prevents accidental target-tag reuse. Session verification also
uses a distinct tag, including when a request opens a replacement connection. The observer alone raises its
pinned session limit to 60 seconds for metadata reads; the target pool is never
altered. RUNNING must be observed before interruption. A terminal Snowflake
record must have `FAILED_WITH_ERROR`, code 630 for timeout or 604 for cancellation,
and end time at or after start time. In-progress end times can be the Unix epoch,
so a non-NULL end time alone does not establish termination. A nonblocking check
also rejects an operation that already returned before cancel. Timeout client
durations must be at least the configured limit (two seconds on PostgreSQL, ten
on Snowflake); PostgreSQL must return before twenty seconds.

PostgreSQL uses a disposable database, a volatile `pg_sleep` function and
`pg_stat_activity`. Execution is established only when `wait_event='PgSleep'`,
while termination still requires no active target query. This avoids canceling
in the extended-protocol startup before the UDF begins. Its timeout profile
uses a two-second database default,
applied before the target pool opens. If result metadata is missing, both
adapters close the cursor and check its stored error before reporting a schema
mismatch. PostgreSQL tests require the original 57014 timeout code or the context
cancellation cause, without a textual schema-mismatch exception. The public
`UnavailableError` stays opaque and no partial result is returned. Cancellation
also requires the target to disappear before the 30-second UDF delay could
complete naturally.

The vendored Snowflake driver currently sends cancellation synchronously,
without a separate cleanup deadline. Measured cancellation latency includes
this step; the test's observation bounds do not establish a production latency
guarantee. The datastore retains its opaque `UnavailableError`, so cancellation
is not identifiable through `errors.Is` above that boundary. The later resource
supplement covers local mode transitions and a driver-level chunk fault after
query completion; neither is part of these running-query fixtures. Actual
network outages and credit/resource controls remain outside these fixtures.

The read-only inventory can be repeated against names from any campaign's
manifests, including incomplete runs. It bounds files, schema names and server
rows, and refuses an empty ledger or an inventory reaching its row limit.
It neither creates nor drops schemas, and leaves unrelated schemas alone:

```sh
# Use a standard UTC overlay. Credentials are already in the environment.
# Unset the skip flag: false would require schema creation in this read-only run.
env -u KRENALIS_SKIP_SNOWFLAKE_TESTS KRENALIS_SNOWFLAKE_CAMPAIGN_DIR="$campaign_dir" \
  python3 test/snowflaketester/run.py --require-session-timezone UTC \
  --require-session-profile standard "$inventory_dir/inventory.jsonl" -- \
  -overlay "$standard_overlay/overlay.json" ./test/snowflaketester \
  -run '^TestCampaignInventory$' -timeout=2m
```

### Interruption results on 2026-11-11

The UTC campaign on `e62f2d64a` passed all four running-query scenarios and
all four corresponding recovery requests. Each block reports three PASS nodes
(two scenarios and their parent), with no FAIL or SKIP:

| Profile | Path | Server code | Client elapsed |
| --- | --- | ---: | ---: |
| timeout | membership | 630 | 10,758 ms from call |
| timeout | explanation | 630 | 10,716 ms from call |
| cancellation | membership | 604 | 130 ms after cancel |
| cancellation | explanation | 604 | 141 ms after cancel |

Each block verified three physical connections under the requested profile;
the read-only inventory verified one standard connection. All three manifests
report stable sources, exit 0 and no invalid session records. The two fixture
schemas have matching CREATE/DROP records. A separate server inventory found
zero remaining campaign schemas and 23 other test schemas, which were untouched.

Evidence is outside the checkout at
`/private/var/folders/v8/wf803m8x1q9fvcwkvfps6qtr0000gn/T/krenalis-snowflake-interruption-hgaekg26/`:
`timeout.jsonl`, `cancellation.jsonl`, their manifests and
`inventory/inventory.jsonl` with its manifest. Logs contain masked Output;
manifest schema ledgers were extracted before masking. `run.py --audit` reports
two created and two dropped schemas, with no issues. The earlier raw feasibility
probes, including the corrected schema-name attempt, remain separate evidence;
they are not counted as Unified scenario passes.

Local checks passed: the datastore suite with `-short`, then the final targeted
PostgreSQL timeout/cancellation tests, 26 Python runner/overlay tests, test-helper
race tests, vet and build of the affected packages. The full datastore suite ran
before the cancellation test entry points were added; the final targeted run
covers that addition. Production code, session configuration and the closed
Snowflake gate did not change.

### Post-review verification on 2026-11-11

The corrections in `536845daa` and `70f413279` isolate the session-verification
tag, use a fresh scenario tag for each operation, check that the operation has
not returned before cancel and enforce the timeout duration bounds. Both
adapters now return a stored cursor failure before reporting a schema mismatch;
PostgreSQL tests require 57014 or the cancellation cause without a textual
exception. The datastore's public error identity and the gate are unchanged.

The live UTC campaign was repeated on `70f413279`, with stable sources and no
FAIL/SKIP in either block:

| Profile | Path | Server code | Client elapsed |
| --- | --- | ---: | ---: |
| timeout | membership | 630 | 10,619 ms from call |
| timeout | explanation | 630 | 11,137 ms from call |
| cancellation | membership | 604 | 1,517 ms after cancel |
| cancellation | explanation | 604 | 1,685 ms after cancel |

Each block again reports three PASS nodes and three verified connections.
The read-only inventory reports one PASS and one standard-profile connection,
zero remaining campaign schemas and the same 23 unrelated test schemas. Both
created schemas have drop records and the audit reports no issues. Evidence is
also in `/tmp/krenalis-snowflake-interruption-review-lz9eiq1v/`, using the same
log/manifest layout as the original run.

A durable local copy of both runs, outside the checkout and temporary folders,
is stored at
`/Users/marco/src/krenalis-test-evidence/snowflake-interruption-70f413279-bd5dda129471.zip`.
SHA-256: `bd5dda1294715324b4d311708fade07068b15fe445cdc0d363f863bec86fd73c`.
Its adjacent `.sha256` file verifies the archive. `INDEX.json` maps 42 archived
files to their original paths, sizes and hashes, so original manifests and
overlays remain unchanged. Contents are masked test output, manifests and test
overlays; launchers and credential files are excluded.

Local verification passed: four final PostgreSQL scenarios, test-helper race
checks, 26 Python tests, profile/reader/explanation tests in both adapters and
the shared compiler, vet and build. Three overlay mutants were detected: reuse
of the target query tag, and removal of error preservation in each PostgreSQL
adapter. Full datastore and full remote numeric/calendar campaigns were not
repeated for these changes; their prior evidence remains separate.
