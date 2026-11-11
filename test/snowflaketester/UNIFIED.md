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
block has 22 batched membership checks and 204 individual explanations, rather
than thousands of remote oracle queries. The named subtests can be selected
separately, for example `-run '^TestUnifiedConformancePostgreSQL$/periods'`.

## Preparing an isolated gate overlay

Prepare this only for testing. It copies the current file, changes exactly the
gate constant, and leaves the tracked source untouched. Regenerate after changes
to that source file; do not reuse an old copy of its implementation.

```sh
campaign_dir=$(mktemp -d /tmp/krenalis-snowflake-campaign.XXXXXX)
python3 - "$PWD" "$campaign_dir" <<'PY'
import json
import pathlib
import sys

root, output = map(pathlib.Path, sys.argv[1:])
source = root / "warehouses/snowflake/profile_filter.go"
text = source.read_text()
old = "const profileFilterVerified = false"
if text.count(old) != 1:
    raise SystemExit("expected one closed Snowflake gate")
target = output / "profile_filter.go"
target.write_text(text.replace(old, "const profileFilterVerified = true"))
(output / "overlay.json").write_text(json.dumps({"Replace": {str(source): str(target)}}))
PY

# Compile and prove that the bootstrap still skips before credential access.
KRENALIS_SKIP_SNOWFLAKE_TESTS=true go test -overlay "$campaign_dir/overlay.json" \
  ./core/internal/datastore -run '^TestUnified(Conformance|StoredNumbers)Snowflake$' \
  -count=1 -json > "$campaign_dir/compile-only.jsonl"
```

This last command must report SKIP for both tests. Test the production gate
separately without an overlay (`TestUnifiedProfilesSnowflakeGate` above). Do not
run the whole datastore package with the gate overlaid: gate-contract assertions
intentionally expect it to be closed. There is no runtime bypass.

## Remote execution, when authorized

Remote testing is currently deferred. Do not execute this section until the user
re-enables it. Use the existing `KRENALIS_SNOWFLAKE_TESTER_*` environment variables,
an isolated test database/role and the temporary schemas created by
`snowflaketester`. Do not include credential files or values in commands, logs or
commits. Do not enable shell tracing. No application or demo data is needed.

Keep runs sequential (`-parallel=1 -p=1`), retain JSON test results, check cleanup
and redact server diagnostics before sharing them: driver errors can contain
account, database, user or schema identifiers. A skipped test is not a pass.
The fixture sets UTC, TIMESTAMP_NTZ mapping and a 60-second statement timeout;
remote checks must also confirm the active session and physical timestamp types.
It does not configure warehouses or resource monitors.

Run the following blocks in order, stopping on a failed prerequisite:

```sh
# Credentials must already have been supplied through the environment.
# These commands are remote, including tests that do not need the Unified gate.
KRENALIS_SKIP_SNOWFLAKE_TESTS=false go test ./warehouses/snowflake \
  -run '^(TestJSONNumbers|TestProfileExactDecimal|TestJSONExpressions|Test_Merge)$' \
  -parallel=1 -p=1 -count=1 -timeout=30m -json > "$campaign_dir/physical-warehouse.jsonl"

KRENALIS_SKIP_SNOWFLAKE_TESTS=false go test ./connectors/snowflake \
  -run '^Test_Merge_Query$' -parallel=1 -p=1 -count=1 -timeout=15m \
  -json > "$campaign_dir/physical-connector.jsonl"

KRENALIS_SKIP_SNOWFLAKE_TESTS=false go test -overlay "$campaign_dir/overlay.json" \
  ./core/internal/datastore -run '^TestUnified(StoredNumbers|Conformance)Snowflake$' \
  -parallel=1 -p=1 -count=1 -timeout=45m -json > "$campaign_dir/unified-snowflake.jsonl"

# The PostgreSQL arm is skipped: this block isolates real Snowflake Record reads.
KRENALIS_SKIP_SNOWFLAKE_TESTS=false go test ./core/internal/datastore \
  -run '^TestJSONFloatFilters$/Snowflake' -parallel=1 -p=1 -count=1 -timeout=15m \
  -json > "$campaign_dir/record-floats.jsonl"
```

Physical numeric expectations follow the approved
[JSON transport contract](../../warehouses/snowflake/README.md). New fixed-point
writes and historical DOUBLE writes are separate fixtures. Preserve exact
INTEGER/DECIMAL values; compare native DOUBLE transport separately from returned
JSON display text. A rounded display is an accepted output limit, not a reason to
weaken Record or governed monetary expectations. Do not add corrective queries,
rewrite connector SQL or restore the discarded result dictionary.

## Coverage still required before activation

Passing the commands above is necessary, but is not the complete acceptance
checklist. Record actual results against each block in the Unified README.
In particular, the following work remains explicit:

- Remote physical results for the new writer, midpoint fixtures, NUMBER borders,
  nested Array/Map and server JSON output; native FLOAT extraction versus nested
  output in VARIANT/OBJECT/ARRAY, both Arrow and JSON result transport.
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

No gate change follows automatically from a green reduced suite. Resolve observed
differences, finish the acceptance checklist and keep a repeatable remote
regression subset before deciding activation. The deferred infrastructure work
and support for additional native numeric types remain separate.
