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

Both schema-ledger audits report three schemas created and three dropped.
Separate read-only server inventories found none of these six schemas remaining.
Evidence, per-query format records, masked logs and `summary.json` are under
`/private/tmp/krenalis-numeric-transport-live-yxltfce1/`. The initial sandbox DNS
failure occurred before schema creation and has separate evidence under
`/private/tmp/krenalis-numeric-transport-live-dlbs2lhs/`.

Account/user-level parameter inspection, forced remote chunk downloads and
additional native types remain outside this block. The Unified gate is closed.

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
The runs took about 14 minutes for each new-group block and 8 minutes for each
baseline block, including setup and many individual explanations. This is not a
production latency benchmark.

Each configuration created and dropped three schemas. Independent read-only
inventories found zero remaining campaign schemas. Evidence is under
`/private/tmp/krenalis-unified-conformance-live-zgm78an6/`: `summary.json`, per-zone
manifests and masked logs, ledger audits, and the `inventory/` results. Output
checks found no occurrences of the six Snowflake configuration values in either
campaign's test output. No implementation or production gate changed.

## Coverage still required before activation

Passing the commands above is necessary, but is not the complete acceptance
checklist. Record actual results against each block in the Unified README.
In particular, the following work remains explicit:

- Decide how native FLOAT/BOOLEAN compatibility with JSON results affects the
  supported transport contract. Nested numeric fixtures and the Arrow native
  path passed above; the failed JSON native probe does not certify equivalence.
- Remaining Record operator/collation/Float-NaN cases and source cases beyond the
  shared fixtures. The added groups now provide remote evidence for Count money,
  Sum/Average where and minimum, covered conflicts, mixed explanations,
  projections and pagination. Not every Record case has a Snowflake setup yet.
- Real core service plus HTTP/MCP on Snowflake. The current
  `TestUnifiedWorkspacePostgreSQL` and `TestProfilesMCP` hard-code PostgreSQL
  setup/DML. Running them unchanged verifies the local service, not Snowflake.
- Further scan/iteration/close failures, malformed/oversized results, additional
  timestamp precisions, session changes and a small scale/resource smoke test.
  Running-query timeout/cancellation evidence is recorded separately below.
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
is not identifiable through `errors.Is` above that boundary. Download
interruption after query completion, mode changes and credit/resource controls
remain outside these fixtures.

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
