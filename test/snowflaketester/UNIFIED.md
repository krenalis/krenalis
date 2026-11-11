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
and assertions. Only table definitions and timestamp-copy SQL differ. Events go
through the real Merge writer; identical copies are then inserted directly so
that a primary key cannot hide deduplication defects. The fixture supplies a
coherent, fully attributed view; it does not test asynchronous publication.

| Fixture | Coverage |
| --- | --- |
| `selections` | 32 histories: 12 fixed counterexamples, 16 PCG(26,11011) samples of at most six events, four eleven-event borders. First, Last, Nth(1/2/3/10), Latest(1/2/10) All/Some/exactly-one/at-most-one, all four ordered Last-minus-Previous comparisons. Missing, null and incompatible currency, negative amounts, unreadable amounts, opposite Decimal(18,4) borders, optional membership, ties and shared-currency conditioning. |
| Selection assertions | Existing independent enumeration oracles supply outcomes and candidate/absence/pair metadata. Every profile's metadata is checked for Latest(2)/Some and difference <= -30; the other rules explain the first six profiles, plus all four eleven-event histories for position/size 10. All rules check exact ternary totals, profile ordering and empty matching pages. Relevant timestamp conflicts fail without partial results; excluded copies and unrelated-profile explanations remain usable. |
| `periods` | Absolute, elapsed and ago-band periods; Rome complete days across both DST changes, complete week and month. Start/end and +/- one microsecond are checked through both Count and First/occurredIn. History includes an event in year 0001 and excludes T. received_at/sent_at deliberately lie outside the window. |
| `pages` | Page and screen, named and unnamed, missing/empty/lowercase names, profile without events, literal dotted JSON key segments and count explanations. |
| `TestUnifiedStoredNumbers*` | Shared six-profile monetary fixture: Count, Sum with minimum, Average, selected money, LastVersusPrevious and composition, with all profile outcomes/explanations, canceled operation and empty pages. |

The bounded fixture complements the full PostgreSQL completion oracles; it does
not replace them. It avoids running the exhaustive suites remotely. Its selection
block has 22 main batched membership checks and 204 individual explanations, rather
than thousands of remote oracle queries. The named subtests can be selected
separately, for example `-run '^TestUnifiedConformancePostgreSQL$/periods'`.

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

This last command must report SKIP for both tests. Test the production gate
separately without an overlay (`TestUnifiedProfilesSnowflakeGate` above). Do not
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

Run the following blocks in order, stopping on a failed prerequisite:

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
  --require-session-timezone "$session_timezone" "$campaign_dir/unified-snowflake.jsonl" -- \
  -overlay "$campaign_dir/overlay.json" \
  ./core/internal/datastore -run '^TestUnified(StoredNumbers|Conformance)Snowflake$' \
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

## Coverage still required before activation

Passing the commands above is necessary, but is not the complete acceptance
checklist. Record actual results against each block in the Unified README.
In particular, the following work remains explicit:

- Additional nested Array/Map numeric boundaries and native FLOAT extraction
  versus nested output in VARIANT/OBJECT/ARRAY, explicitly verifying both Arrow
  and JSON result transport. The successful default-driver runs above do not
  certify both transport formats.
- Remaining Record operator/collation/Float-NaN cases and all source-conflict
  classes, Sum/Average independent where and minimum counterexamples, Count money
  predicates, mixed explanation layouts, projections and nonempty pagination.
  Existing PostgreSQL fixtures name these cases; not every one has a Snowflake
  setup yet. The bounded conformance test does not claim that coverage.
- Real core service plus HTTP/MCP on Snowflake. The current
  `TestUnifiedWorkspacePostgreSQL` and `TestProfilesMCP` hard-code PostgreSQL
  setup/DML. Running them unchanged verifies the local service, not Snowflake.
- Statement timeout/cancellation during execution, scan/iteration/close errors,
  malformed/oversized results, additional timestamp precisions, session changes
  and a small scale/resource smoke test.
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
Its own tag prevents accidental target-tag reuse. The observer alone raises its
pinned session limit to 60 seconds for metadata reads; the target pool is never
altered. RUNNING must be observed before interruption. A terminal Snowflake
record must have `FAILED_WITH_ERROR`, code 630 for timeout or 604 for cancellation,
and end time at or after start time. In-progress end times can be the Unix epoch,
so a non-NULL end time alone does not establish termination.

PostgreSQL uses a disposable database, a volatile `pg_sleep` function and
`pg_stat_activity`. Its timeout profile uses a two-second database default,
applied before the target pool opens. A timeout currently can lose the original
57014 diagnostic: pgx supplies a cursor without field metadata and the adapter
returns `profile outcome schema mismatch`. The tests retain this existing
behavior, including `UnavailableError` and no partial result; they do not change
production error handling. Its cancellation check also requires the target
to disappear before the 30-second UDF delay could complete naturally.

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
