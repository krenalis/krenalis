# D2-B1 v3 — Canonical Income Arithmetic Rectification Report

**STATUS: PROPOSED / UNFROZEN. QUALITY: DEMO PLAUSIBLE.**

Offline surgical rectification, 2026-09-17. MODEL CHANGE = NO; QUALITY CHANGE = NO.
The two numerical rules are now normative in [v3 §5.1](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v3.md).
Six authored supports change by +1 EUR in one law. All required effective
Income HARD gates pass; acceptance and generator architecture remain unchanged.
The requested execution profile is GPT-5.6 Sol xhigh; unavailable model/session
telemetry is not certified. No web or D2-B2 production was used.

## 1. Independent findings

Astra independently showed that mathematically equivalent interpolation schedules
produce different finite Decimal96 Q values and whole-EUR supports. Employee
bucket/ID 11, j=170 yields 65164 for multiply-before-divide and 65163 for
ratio-first interpolation. Decimal context alone did not select an operation graph.

Astra also showed that context-rounded scaling and exact multiplication of the
cached finite operands yield different ceilings: 5106 versus 5107. This second
case was found mechanically by exhaustive reconstruction, then compared with the
preserved historical `ceiling_witness.py`; coordinates were not invented.
These are canonical determinism defects, not statistical quality, acceptance,
Fast Income architecture, resource or frozen semantic defects.

The historical [Astra review](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v3_ASTRA_FOCUSED_REVIEW.md),
Sol pre-review/evidence and earlier reports remain byte-for-byte historical
records against the old v3 SHA. This report does not rewrite their findings.

## 2. Frozen-authority interpretation

Before editing, current v3 SHA matched exactly `785a9d9bd575d85b5a91a82bf49d5ac9bb69a8b8f2aaa41891da61458b996ae8`.
v3, v2, v1 and v0 were read completely, as were the applicable AGENTS.md and
v2/v3 reports. Contract 5 and D2-A governing arithmetic, canonicalization,
representation and resource sections were read directly from frozen authority.

Contract 5 [§5](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md) lines 178–215
requires checked integer or exact rational arithmetic, an exact pre-quantization
EUR rate, `ceil(x)`, and exact inclusive `0..4294967295` validation before
conversion. Its lines 188–207 explicitly define exact conversion; its .01-above-
maximum fixture rejects before ceiling. D2-A [§§4–5/16/21](SYNTHETIC_WORLD_V1_LAYER2_D2A_v1_FROZEN.md)
lines 71–89/117–123/398–413/573–640 retains uint32 supports/weights, checked
uint64 totals, exact sort/merge/GCD, branch-local first-occurrence IDs and
bounded finite resources. Neither frozen authority authorizes context rounding
of an already-canonical finite scale product before the ceiling.

**Confirmed interpretation:** use canonical cached Q and the exact reachable
representative as finite operands, form their exact mathematical product, then
apply whole-EUR ceiling. This honors frozen semantics even when historical
prototype bytes change. Contracts 1–5 and D2-A were not modified.

## 3. Canonical interpolation operation graph

Canonical context: CPython 3.13.2; Decimal precision 96; ROUND_HALF_EVEN;
Emin=-999999999; Emax=999999999. Grid remains `(2j+1)/768`, j=0..383,
converted by canonical Decimal division of exact integer operands.

```text
d_u = u - u0
d_q = q1 - q0
p   = d_u * d_q
r   = p / (u1 - u0)
q   = q0 + r
```

Each operation executes sequentially under that context. The denominator
subtraction occurs after p and before division. The cached q is the finite
Decimal96 result, without extra 10^-40 quantization. The equivalent expression
is `q0 + (u - u0) * (q1 - q0) / (u1 - u0)` with this specified graph.
Mathematical equivalence is insufficient for canonical compatibility.
Ratio-first evaluation, reassociation, reordered/FMA or compiler substitutions
cannot replace this graph where output can change; binary float is forbidden.

## 4. Normative interpolation fixture — 65164

Employee e; bucket 11; canonical law ID 11; positive authored j=170;
u=Decimal96(341/768); segment u0=.25/u1=.50; q0=24000/q1=32000;
exact representative 2.15712000.

```text
canonical cached Q:
30208.3333333333333333333333333333333333333333333333333333333333333333333333333333333333333333334

exact scaled pre-ceiling rate:
65163.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000143808
= 65163 + 1.43808e-91

PASS -> 65164
```

Ratio-first interpolation with exact scaling yields 65163 and fails the normative
fixture. A full ratio-first/exact-scale attack differs at exactly this employee
cell; it does not redefine the historical-versus-rectified delta scan below.

## 5. Canonical exact scaling/ceiling rule

```text
base = canonical cached Decimal96 Q result from the pinned graph
m = exact canonical reachable bucket representative
scaled_exact = exact finite-decimal product(base, m)
support = mathematical ceil(scaled_exact)
```

Validate the exact product in `0..4294967295` before ceiling; every authored
positive cell remains strictly positive. `scaled_exact` is never first rounded
back to Decimal96. Exact ratios (`Decimal.as_integer_ratio()`), Fraction,
integer coefficient/exponent multiplication or another exact rational
representation are valid implementations. Context-rounded `m*base` followed
by ceiling is nonconforming when context rounding changes the ceiling.
Arbitrarily increasing precision is not the semantic rule.

There are intentionally two regimes: ordered Decimal96 operations canonicalize
Q; exact multiplication of the resulting finite operands precedes the ceiling.
Do not replace the cache by an ideal real-number Q derived from the knots.
Do not infer the scaling rule from the interpolation context or vice versa.
All inherited probability quantization, Hamilton, merging and GCD rules remain.

## 6. Mechanically identified scaling fixture — 5107

Inactive i; bucket 7; canonical law ID 7; positive authored j=104;
u=Decimal96(209/768); segment u0=.25/u1=.50; q0=3500/q1=7500;
exact representative 1.324800.

```text
canonical cached Q:
3854.16666666666666666666666666666666666666666666666666666666666666666666666666666666666666666667

Decimal96-rounded product:
5106.00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000

exact finite-decimal product:
5106.000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004416
= 5106 + 4.416e-93

rounded-product ceiling: 5106 (nonconforming)
exact-product ceiling:   5107 (canonical)
PASS -> 5107
```

The exact product is formed from the cached finite Q, not from an unrounded
interpolation target. Both independent conforming paths pass this fixture.

## 7. Exhaustive 23040-support delta scan

The unchanged fixed candidate was parsed directly from checksum-bound v0/v2
recipe data. Factors, all 22506 distinct scalars, bucket boundaries and lower-
median representatives were reconstructed; bucket bytes match historical bytes.
Every one of 60 laws × 384 positive authored cells was compared.

Historical path: ordered Decimal96 interpolation, context-rounded Decimal96
representative multiplication, then ceiling. Rebuilt historical law/map/bucket
bytes exactly reproduce all three pinned historical hashes. Rectified path:
the same ordered interpolation, exact finite product, mathematical ceiling.

**Exactly six supports change; one branch (inactive), one law (bucket/ID 7);
maximum absolute delta 1 EUR.** These are all changed authored-cell locations:

| Branch | Bucket / ID | j | Exact grid fraction before Decimal96 | Historical EUR | Canonical EUR | Delta EUR |
| --- | --- | ---: | --- | ---: | ---: | ---: |
| i inactive | 7 / 7 | 104 | 209/768 | 5106 | 5107 | +1 |
| i inactive | 7 / 7 | 119 | 239/768 | 5934 | 5935 | +1 |
| i inactive | 7 / 7 | 134 | 269/768 | 6762 | 6763 | +1 |
| i inactive | 7 / 7 | 149 | 299/768 | 7590 | 7591 | +1 |
| i inactive | 7 / 7 | 164 | 329/768 | 8418 | 8419 | +1 |
| i inactive | 7 / 7 | 179 | 359/768 | 9246 | 9247 | +1 |

For each of the six witnesses, the canonical exact product equals the historical
integer plus 4.416e-93; Decimal96 multiplication erased that positive excess.
All other 23034 authored supports are identical. No tuning, fitting, clustering,
adaptive processing, optimization, clipping or rescaling was attempted.

## 8. Updated candidate hashes, deduplication, counts and IDs

The complete cheap Income prototype was rebuilt under both arithmetic paths.
Each reconstruction performs 2304 Q evaluations, produces 60 laws and 23040
positive cells. Laws e/s/u/t/r/i remain 16/12/8/8/8/8. Each final law has 385
strictly ascending distinct supports including zero; there are 23100 total
points. No positive supports merge and no within-branch identical laws deduplicate.
All 60 laws remain referenced. Exact weights, totals and GCD are unchanged.

First-occurrence canonical IDs and every tuple-map entry remain unchanged:
SE bucket6→ID7/bucket7→ID6, all other bucket labels equal local IDs. Maps retain
76160 slots: 64960 reachable and 11200 employee sentinel-65535 slots. A second
Fraction-based scalar/map reconstruction from reversed tuple enumeration
reproduced representatives, canonical law order and every map entry.

Prototype checksums use inherited `json.dumps(sort_keys=True,ensure_ascii=False,
separators=(',',':'))`, compact UTF-8 and one terminal LF. Hashes are temporary
reconstruction evidence, not published D2-B2 assets or Protocol identity bytes.

| Prototype material | Bytes old / new | Old SHA-256 | New SHA-256 |
| --- | ---: | --- | --- |
| All canonical law rows | 331095 / 331095 | `072a9b5e248013fcaf4a9175e305c99dd09570e36fea231faa3ba3f3f259dbc9` | `d1de8c861b2803c2394f004fa9789833d7c6cbf96bfef481eb9f5493118c6b53` |
| Complete maps | 212659 / 212659 | `1335b1d86a3c03b939b3e4a8f524d8f52daf8281fde8cbb5695de428a4261ddb` | `1335b1d86a3c03b939b3e4a8f524d8f52daf8281fde8cbb5695de428a4261ddb` |
| Exact buckets | 8829 / 8829 | `509eccd8b0933ede12e4d8beb11656693f44fb1b6fd034f7a780737414abbba5` | `509eccd8b0933ede12e4d8beb11656693f44fb1b6fd034f7a780737414abbba5` |

Only i7's compact canonical row changes:
old `ef65cff1ca90ed2983bc88ab21a51f0b36b5b2ca6536cc7d5d97f2a7062a2d0e`;
new `638b72359eef65104ff154184d76f68fd7d3a75e6a6d388ea02b1dc0b772c6ee`.
The other 59 row hashes are unchanged (all final hashes are in Appendix A).
Authored-support evidence checksum is
`505a9511cf4146d3b6f6d477549f7b8f5c97230bdfc7401b3d7d1fe58cc2e68d`.

For branch model audit hashes, the in-memory envelope is exactly v0 §32:
`{"schema":"SW-L2-D2B2-logical-v0","model":"income.<branch>","asset":
{"laws":<canonical rows>,"tupleMap":<complete map>}}` under that serialization.
No full logical-assets document or canonical package was emitted.

| Branch | New law-pool array SHA-256 | New logical model audit SHA-256 | Changed |
| --- | --- | --- | --- |
| e | `29aa246b41994d2299e2937379d4109e825f39a788d0243894f0390ea36cceba` | `bd0320d9a402afe7a565a5e9eb0830536729f64bbf653f7c684b4236955f7ee0` | NO |
| s | `64685483b76d686232a894ca11290af4a85d13aec2bb999207d2b20d4f7a8fc3` | `35490c2b6176c2981648ede5a56b996250f1e3b3f33ed2aefb22008aee4a18bf` | NO |
| u | `45596a441d61b4d4eb36e30614865d4608f7fcc979886114a624860ef4c7d793` | `a510ec0c44112667bde0b486a2976b407df90c3d57255d7056c59c08353e328a` | NO |
| t | `d23d680097a0414d5e6ef2fed94ebb1776acdd101adc1349e410e063d52b2699` | `547f51c1515c1889c6c44b50aa3da64292b2871089717e8b9d749df7e829934b` | NO |
| r | `8f6bcbbd4cd656e0424d881dfc1273de3c86ff49abe236c5cbc3c8a444a5b928` | `b10bd0087f959ec6bf2ce645ddab30da3f8d193f3d3b1d6e4fad10e96ae5bbb4` | NO |
| i | `ef5ee6b92c8b857847bafdcbb907dc99fd6c37622941d9443cc03d7c7989469e` | `5f1800e7d8ee713ac0f9eb687860bb431d294b13991f8d514895b4fb4fb02741` | YES |

Inactive pool old hash: `083fb226d60e8dd7b98b49aa92170351547976f6653ee247b3cc7dfa23bf152b`.
Inactive logical model old hash: `65cf90e6cd8fdf0a15c299b712f6ba9a1033ddc213ab1b98c7ee313e56958b68`.
Thus branch output-sensitive identity material changes only for inactive Income;
a stable local law ID does not imply unchanged law content or fingerprint.

## 9. Effective HARD replay and unchanged acceptance

The effective §4 table was compared byte-for-byte with pre-rectification v3:
21 total, 14 HARD, 3 WARNING, 4 DIAGNOSTIC. Every table row, threshold and
severity is unchanged. Legacy caps stay DIAGNOSTIC; V07 stays WARNING.

The checksum-bound pre-integer validation D0 was reused only after complete
Income construction; its SHA is
`6842b138699db4e856be4e14d43801b22c0576d157e0f97ab40f88994645a7d5`.
It matches historical replay evidence exactly. Parsed profiles, zero masses,
Q knots and all eight age maps agree with that state. No non-Income fitting or
D0 reconstruction was executed. The retained exact-rational gate computations
were replayed on the rectified final integer laws and mixtures.

| Check | Result | Evidence |
| --- | --- | --- |
| S02 | PASS | Meaning/parent/exclusion inspection unchanged; all 23040 exact pre-ceiling products strictly positive and ≤MaxUint32; exact branch z unchanged before integerization; final zero error bounded |
| V04 | PASS | Six final-mixture mean/positive-median ratios in [.5,2], all 15 strict reference-rank pairs preserved |
| V05 | PASS | 2086 scalar-ordered pair/quantile checks; maximum opposite difference 0 EUR |
| V06 | PASS | Every law and all six mixtures below .02 unconditional positive atom |
| V08 | PASS | All 170 required internal 1000-EUR bins occupied: e44/s39/u21/t13/r30/i23 |
| Law validity / integerization / zeros | PASS | Checked uint32 supports/weights, positive checked uint64 totals, exact probability mass, support sort/GCD and unchanged weights |
| Support order | PASS | All final laws strictly ascending; 20736 adjacent-law corresponding-cell monotonicity checks |
| Tuple→law completeness / IDs | PASS | All 64960 reachable cells and 11200 employee sentinels checked; all laws referenced; two independently built maps equal |
| D2-A capacities / resources | PASS | Unchanged cardinalities and all prospective frozen bounds in §11 |

V04 values below are rounded displays only; all decisions used exact rational
comparisons on complete final-law mixtures (never averaged tuple medians).

| Branch | Positive median EUR | Median/reference | Unconditional mean EUR | Mean/reference | Result |
| --- | ---: | ---: | ---: | ---: | --- |
| e | 33897 | 1.059281 | 43600.596581803 | 1.118769960 | PASS |
| s | 26164 | 0.934429 | 40202.122640158 | 0.967732356 | PASS |
| u | 9278 | 0.927800 | 7799.507532945 | 0.941116679 | PASS |
| t | 4506 | 1.126500 | 2747.256530875 | 1.178661834 | PASS |
| r | 17222 | 0.861100 | 20380.151341454 | 0.865140035 | PASS |
| i | 6894 | 0.919200 | 5311.333067942 | 0.919126097 | PASS |

Only the inactive unconditional mean changes, from approximately 5311.332702
to 5311.333067942 EUR; medians, ranks and every PASS result remain unchanged.

| Branch | Maximum unconditional positive law atom | Maximum mixture atom | V06 |
| --- | ---: | ---: | --- |
| e | 0.002598959 | 0.000905160 | PASS |
| s | 0.002395833 | 0.000571165 | PASS |
| u | 0.001692709 | 0.000901172 | PASS |
| t | 0.000911459 | 0.000650269 | PASS |
| r | 0.002578125 | 0.001226694 | PASS |
| i | 0.001171875 | 0.000561656 | PASS |

Maximum final zero/entry error and TV stay `199/3435973836` (<5.8e-8).
Maximum integerization-only mean change stays `1358454797/916259689600` EUR
(<.001483). All inherited integerization tolerances remain satisfied.
All 1454756 one-factor comparisons retain 1111486 strict bucket increases,
343270 ties and zero reversals; every factor axis has visible strict increases.

V07 has **six SE WARNINGs**, unchanged in count and all recorded metric fields;
none is top-decile. The same top-decile e0/e1/r4/e2/e3/i5 laws all pass the
stricter threshold. V08 candidate bin masses are also unchanged by these six
within-bin support moves.

| Branch | Bucket / ID | Positive median EUR | Gap EUR | Exact limit EUR | Worst pair EUR |
| --- | --- | ---: | ---: | ---: | --- |
| s | 6 / 7 | 24959 | 512 | 500.00 | 42245→42757 |
| s | 7 / 6 | 27428 | 563 | 500.00 | 46986→47549 |
| s | 8 / 8 | 30109 | 618 | 500.00 | 51578→52196 |
| s | 9 / 9 | 33149 | 680 | 500.00 | 56786→57466 |
| s | 10 / 10 | 37402 | 767 | 561.03 | 63305→64072 |
| s | 11 / 11 | 45060 | 925 | 675.90 | 93824→94749 |

All six historical-cap comparisons remain DIAGNOSTIC and satisfy the frozen
representation bound 4294967295; no other historical cap is exceeded:

| Branch | Bucket / ID | Maximum EUR | Historical cap EUR | Frozen representation |
| --- | --- | ---: | ---: | --- |
| e | 13 / 13 | 1079361 | 1000000 | PASS |
| e | 14 / 14 | 1229969 | 1000000 | PASS |
| e | 15 / 15 | 1468132 | 1000000 | PASS |
| t | 5 / 5 | 103490 | 100000 | PASS |
| t | 6 / 6 | 128217 | 100000 | PASS |
| t | 7 / 7 | 163928 | 100000 | PASS |

PASS claims cover the fixed Income prototype and frozen compatibility checks.
Complete independent D2-B2 production/serialization, full non-Income acceptance,
and Protocol/runtime-owner tests were not performed or newly certified.
Expensive statistical diagnostics/sensitivity were not executed or called PASS.

## 10. Independent arithmetic confirmation and determinism attack

**MATCH.** Path A uses individually ordered Decimal96 operations and
`Decimal.as_integer_ratio()` → Fraction exact product → integer ceiling.
Path B uses Fraction arithmetic and a separately implemented integer
significant-digit ROUND_HALF_EVEN routine after each interpolation operation;
its scaling multiplies exact finite-decimal coefficients/exponents and applies
integer division with remainder for ceiling. It does not use Decimal to
evaluate interpolation or scaling. The fixed candidate is far inside exponent
bounds, which Path B checks; no general Decimal-runtime replacement is claimed.

The implementations also independently implement probability quantization,
Hamilton ties/GCD, law construction and canonical map/ID reconstruction.
They share only parsed authored inputs and prescribed rules. They compare
2304 exact finite Q values (384 per branch), all 23040 supports and all 60
final canonical laws byte-for-byte, plus all complete maps and IDs.
Both normative fixtures pass. Ratio-first and prematurely rounded scaling
are explicitly rejected, rather than treated as alternate canonical paths.

Arithmetic/replay scripts and result evidence are retained outside the repository
under `/tmp/krenalis-d2b1-v3-arithmetic`; script checksums appear in §12.
A dense G, transport or historical Income fitter was never imported/executed.

## 11. Performance/resource non-regression and future compatibility

Exactly 2304 Q evaluations per independent reconstruction; 60 laws; 23040
authored positive cells; no per-law Q reevaluation. Exact finite multiplication
of all 23040 cells is mandatory and acceptable. There is no D0 Income transport,
dense G, H, Ward, clustering, support optimizer, fitting or adaptive processing.
Architecture changed = NO. No timing optimization or arbitrary-precision
workaround replaces the exact scaling semantics.

| Charge / reservation | Bytes | Frozen ceiling |
| --- | ---: | --- |
| Income input / retained | 185664 / 371904 | Aggregate budgets |
| Tuple maps | 152704 | Complete frozen grids |
| Compact whole-model encoded estimate / reservation | 685277 / 6815744 | 8388608 |
| Decoded/work estimate / reservation | 1629218 / 6291456 | 8388608 |
| Outside retained / combined | 1878264 / 14985464 | 8388608 / 25165824 |
| Employer input/index | 41944384 | 67108864 |
| R2 / joint retained | 43255546 / 45133810 | 75497472 / 83886080 |
| Serial-index constructor peak | 72095482 | 134217728 |
| Per-index canonicalization work maximum | 2097152 | 32N, N=65536 |

All values exactly equal historical resource evidence. Each law retains 385
points, meeting ≤512, ≥128 distinct and ≥384 top-decile authored reservations;
aggregate authored points 23100 stay below 240000. Authored/canonical storage,
scratch and live E/D reservations retain their separate charges and lifetimes.
Encoded bytes are an in-memory estimate, not a canonical D2-B2 publication or
runtime RSS claim; actual constructor allocation guards remain phase-owned.

Arithmetic semantics are generic over curves and exact bucket representatives.
They depend on neither ATECO nor twenty Regions nor Italian-specific bucket-count
logic beyond current asset data. Current frozen Contracts retain Italian nominal
EUR Income. No CountryPack, PopulationMix or multi-country implementation occurs.

## 12. Repository integrity and package sanity

Only v3 is modified and this report is created in the repository. No v4,
production assets, implementation, staging or commit. Before/after captures
include branch, HEAD, subject, full unfiltered human and porcelain-v1 statuses,
working/cached binary diffs, raw Git index bytes, root authority/history hashes,
semantic.go content/diff, every repository file/symlink outside .git, and
historical temporary review/evidence inventories.

**Repository integrity PASS.** Branch `property-semantics`, HEAD
`76fa0fff109ca6abad62f55c8761275455842dc5`, subject
`Add Synthetic World Layer 1` are identical before/after. No index mutation:
raw Git index SHA-256 before/after
`9bf5ddbc729f55523ac70792666243018b250ae81053a8e4d741071e003d81c3`.

The pre-existing working binary diff is exclusively `tools/types/semantic.go`;
its 1432 bytes are unchanged, SHA-256
`0311e8b4c5aa5eb470112a26e96e24b1e2946ac350b35851676992bbe61eb6f4`.
Cached binary diff remains empty, SHA-256
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
`semantic.go` itself retains the checksum in the manifest below. v3 was already
untracked at entry, so its explicit before/after document diff is captured as
`v3.diff` outside the repository; tracked Git diff alone does not describe it.

Full unfiltered statuses were captured without abbreviating untracked entries:

| Capture | Before bytes | After bytes | Before SHA-256 | After SHA-256 |
| --- | ---: | ---: | --- | --- |
| Human status | 4452225 | 4452305 | `30aef395831d71e64e50b048302b8b55e0ffc53d903a16935b2ab09004ec0922` | `7101978765c90261810fd4ae61d149a9cbf35377a0f99a4f8acc350966a6241e` |
| Porcelain v1 status | 4565588 | 4565670 | `cfc50fff19cabb246d4c536b137fbebe5b621e0e368b773149a9350df661bb41` | `499bce8a2050aa337f003d9426135cd2e2e92ff0bf954e3106581a8c9a4c86dd` |

Removing exactly this report's new untracked entry from either after-status
reproduces the corresponding before-status bytes. Before/after command outputs
are retained as `branch`, `head`, `subject`, `status`, `status_porcelain`,
`working_binary_diff`, `cached_binary_diff` with `.before`/`.after` suffixes,
plus `index.before`/`index.after`, in `/tmp/krenalis-d2b1-v3-arithmetic`.

The complete repository inventory outside `.git` grows from 100139 to 100140
file/symlink entries: exactly this report added, exactly v3 changed, nothing
removed; every other entry is identical. Before-inventory SHA-256:
`28f3e26ba94a26d8d56406141dc23bfc6e71769401a7d5c807934622d2d268c1`.
All 136 historical temporary review/evidence entries are unchanged across
`/tmp/astra_v3_independent_20260917`, `/tmp/krenalis-d2b1-v3-59408vmv`
and `/tmp/krenalis-d2b1-v2-evidence`; historical inventory SHA-256:
`7ffb255adfd21e979f9f8bccc1e0514d0eb2e31adc0915d8214ce276550e2581`.
The accidental Sol pre-review, independent Astra review and prior evidence
remain historical, with no rewritten verdict or checksum.

Root authority/history and semantic manifest: all following SHA-256 values
are unchanged before/after. v3's permitted difference is recorded in §13.

| File | Unchanged SHA-256 |
| --- | --- |
| AGENTS.md | `4552765ad6bff507d483f89c96768a00bdfc9e55ee79209d726686116a84126f` |
| SYNTHETIC_WORLD_L2_DISCOVERY.md | `b4607112c7573a4f3d7b3746d753c72823af2165f300b3b92c393e8011209a14` |
| SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v0.md | `0d1f5d2b55a50ab5b44b591dea896aeb3c11cd798acc7712620520ab8b8d37bb` |
| SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md | `bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6` |
| SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md | `e9af32a0bb9b48bf8972a408a1369901a3f3a9fe5ecd76d5762a367c055bc098` |
| SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md | `df3ef0b52f37b587b8449cc71085dc9e5aca00f754bbd93e123c6fd5e3a4cb55` |
| SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md | `a5ecfb735270ffba3e6afea23d85e548dd6dc3f07a8e9dfc5294a84ad3576418` |
| SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v0_REVIEW_READY.md | `b4a5fa1a1872cd2afb97ae99041dc9851d004595367457d92e3da5a8f5b5a757` |
| SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v0_STABILIZED.md | `c33cdd2d65612bd158f5111f494ba6a85831cfe32bc9148cbf5bffdbb081b1a4` |
| SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md | `8a75054f793ef47e7071e3eb9fe04c0b32abfdac9223da859f95a333a1526e67` |
| SYNTHETIC_WORLD_V1_LAYER2_D2A_v0_RECTIFICATION_REPORT.md | `2ff76063b7d886839895be9d3a0872f419ff84c1b9fb5721410242306ea0bec3` |
| SYNTHETIC_WORLD_V1_LAYER2_D2A_v1_FROZEN.md | `33ca4c80599dddb0564ea4d27262e430b1931e0368c8d1bee6ca6c10d8078be4` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_M00_SCOPE_RECTIFICATION_FINAL_RESPONSE.md | `b9f0d41c3689c76e41492cefa11d925093558a13f4daf90012ed0fe85d58c088` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_M00_SCOPE_RECTIFICATION_REPORT.md | `07f475aee74bafb02bb97db83ed5a0c2d9a7ae007854da915a58437d18762aae` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_PLAUSIBLE_REDESIGN_FINAL_RESPONSE.md | `bc1556108e6a5d0fd0eabb8cbae295830f5ba14c3135e0009c3d9c9da7746477` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_PLAUSIBLE_REDESIGN_REPORT.md | `9144b5d90f6d2fb265fda0e96219c8cea86c3a88deb430c33d6154fe52214c28` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_v0.md | `94ec7560518f0077681a9b65547832bf46c706c5e7885c7da332bbd476ccc006` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_v0_ASTRA_REVIEW.md | `adb764cd848404bf867905319e3467369e4603e4bfeafd088923edc96699cfe7` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_v0_ATECO_PARSING_RECTIFICATION_REPORT.md | `8f28687a135eabc18fb6d18dcb76a7e28c680c125e0549d0137a3a7b90e57266` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_v0_TRENTINO_RECTIFICATION_REPORT.md | `f8013fdf37aa8ddcb98540f102b18e8effd881739658b93cbabf996bf42dbab8` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_v1.md | `bf2e6a6dc9dbbed816190aef6705950567576c1a69833676bc4953319204236b` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_v1_DEMO_PLAUSIBLE_REPORT.md | `0a0152fbc738cf704c5d5b8be3275e70575bc523207882883e71c85ba652ca24` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_v2.md | `36514d1c20a6bfab02b14feced6c2539fde28c5a60204a6d15629c8b9f82d6b9` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_v2_FAST_DEMO_INCOME_REPORT.md | `6ad34b83e58d6df5106be00d0af0d01126b5d7bc334547c5ee6bf8d41d38a0be` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_v3_ASTRA_FOCUSED_REVIEW.md | `5b44a1a1772d51c5411fe7733e164ab634605750b772acd1c0ee4492b4bbdffc` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B1_v3_FAST_DEMO_FINAL_ACCEPTANCE_REPORT.md | `93c7ba67e678d809538e5aa112edbc049d889a6377cbfd8f7523f0c89bc9f40b` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B2_v0_FAILURE.md | `8f6ec46eab02678e901e4a4116ff553741e9945028409f2772e5e441c8ed5be5` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B2_v0_RETRY1_FAILURE.md | `abf72d30363f6728b7b28f1080610dcf362da126a18806e4e9c81793bf116a26` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B2_v0_RETRY2_BLOCKED.md | `5840ac21217c4f1f973add700358327f8421bdbe9032b57b3ea787429e5f314b` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B2_v0_RETRY2_FAILURE.md | `2ded62f05b81b959db31f2898dd3e81bec7204329ae3b087569c80f4374721eb` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B2_v1_FAILURE.md | `313876297c5f2dff1a12a5c42a7a740062afa4f5fec004f892fadbce58c43ef0` |
| SYNTHETIC_WORLD_V1_LAYER2_D2B2_v1_INTERRUPTED_HANDOFF.md | `5b7ad3f7c0f6c059ce83dd8a9ad1b8f199dee391cd008e4e27d20701e2cf8170` |
| tools/types/semantic.go | `06a7ff690b3e12cdaf623e3ffb36003d1675a6aec69b3ee59e9ead428cc17842` |

Temporary verification implementation checksums (no repository source changes):

| Script | SHA-256 |
| --- | --- |
| arithmetic.py | `0436d9c520d80bd9614b36f0806bcc6c414ecd38f7fa311987ff745d131c6349` |
| validate.py | `1ebe646a8ac4587423f17830574e5e6638be518de186116ff05bec72e944103a` |
| integrity.py | `7b5263c443a7040523938f7ad46f27f43faa89f6a7d04f84a0b32506e90cf6e2` |

The inventories, complete captures, document diff, arithmetic results,
rectified law/map/bucket bytes, gate/resource replay and integrity assertions
remain available in the same temporary evidence directory. No canonical
production asset or Git object was created.

Offline required commands:

```bash
GOCACHE=/tmp/krenalis-d2b1-v3-arithmetic-gocache \
GONOSUMDB='*' \
GOPROXY=off \
  go test ./tools/fakedata

git diff --check
git diff --cached --quiet
```

Package sanity **PASS**, exit 0:
`ok github.com/krenalis/krenalis/tools/fakedata 13.601s`.
Git checks and separate authorized-untracked-document whitespace/LF checks PASS.
This validates the existing package, not a Go Layer-2 implementation.
The validation harness initially hit a final-summary variable collision after
all gate/resource calculations completed; only temporary summary tooling was
corrected and rerun successfully, without candidate or model changes.

## 13. Old/new v3 SHA and final self-review

Old v3 SHA-256: `785a9d9bd575d85b5a91a82bf49d5ac9bb69a8b8f2aaa41891da61458b996ae8`.

New v3 SHA-256: `aa766ccdcae1f2c94d12f048e0fa6ba3384e801e5b5d4f6a09f0c49bee3e1a35`.

Interpolation order fully specified; ratio-first rejected; 65164 PASS.
Cached-value scaling exact before ceiling; no Decimal96 product rounding;
5107 PASS. Two implementations match all 23040 supports and final laws.
No model tuning or acceptance change; S02/V04/V05/V06/V08 PASS; V07 WARNING (6).
Contracts/D2-A PASS; 2304 Q evaluations and 60-law Fast Demo architecture
preserved; repository integrity and package sanity PASS. No canonical
arithmetic interpretation remains intentionally open. v3 is PROPOSED / UNFROZEN.

## 14. Micro-review recommendation and requested result

Proceed to micro delta review of v3 §5.1: ordered operation graph, exact finite
product boundary, both normative fixtures, all six support deltas, changed i7
law/model hashes and unchanged table/maps/IDs/resources. Prior acceptance
choices remain in force. No freeze or D2-B2 production is declared here.

```text
Old v3 SHA: 785a9d9bd575d85b5a91a82bf49d5ac9bb69a8b8f2aaa41891da61458b996ae8
New v3 SHA: aa766ccdcae1f2c94d12f048e0fa6ba3384e801e5b5d4f6a09f0c49bee3e1a35
Generator architecture changed: NO
Acceptance changed: NO
Canonical arithmetic changed/clarified: YES
Interpolation operation order: PINNED
65164 fixture: PASS
Scaling product semantics: EXACT FINITE PRODUCT BEFORE CEILING
5107 fixture: PASS
Authored supports scanned: 23040
Supports changed vs historical prototype: 6
Maximum EUR change: 1
Laws affected: 1 (inactive bucket/canonical ID 7)
Law count: 60
Canonical IDs changed: NO
  SE bucket6→ID7/bucket7→ID6; all other local IDs and complete maps unchanged
S02: PASS
V04: PASS
V05: PASS
V06: PASS
V07: WARNING; 6
V08: PASS
Independent arithmetic implementations: MATCH
Contracts: PASS
D2-A: PASS
Performance architecture changed: NO
Package sanity: PASS
Repository integrity: PASS
Rectification report SHA-256: supplied separately after final bytes
```

## Appendix A. All final canonical row hashes

Each checksum hashes exactly one compact sorted `{"values":[...],"weights":[...]}`
row plus LF. Old and new are identical except inactive ID 7, whose old hash
is explicitly recorded in §8. All 60 were recomputed, not assumed unchanged.

| Branch | Canonical ID | Final row SHA-256 | Changed vs historical |
| --- | ---: | --- | --- |
| e | 0 | `d7251314c713a177c81bc0ff409b55929563b4be2815add279d12f90700b468d` | NO |
| e | 1 | `1fbe13ff4437c13a3c3297fb153c8edde226155c3368be2118f3be913d4e6dc3` | NO |
| e | 2 | `43cc64cadbe199bb1790f0ec75f2f319e6178c930ec56d58c30108f238f6b690` | NO |
| e | 3 | `64eec46e9a400f14a5c8b848d05e63d87ed368120e77f36ba29cce86f01bb971` | NO |
| e | 4 | `26db0324534b2d3d63a0224ed1033dd3036b56b8515ee9badc5143ab894c8b2d` | NO |
| e | 5 | `ffe9479151a6103795345ffd944cdf282d990f490635a3974511f1ad6dc12a71` | NO |
| e | 6 | `095bf16ebf5672c975aa28a4f0fd9e330ccab49c91532e53bc16f3b7b77334a6` | NO |
| e | 7 | `41a560ebae879b8f3b7f1e273b35a83327db51b1bb4dcaab8d7711ca56e2932e` | NO |
| e | 8 | `c8d72a732bae995d307f99b632bdbfffba84aef1210d8e8cdb09709c8b67856a` | NO |
| e | 9 | `6a5b247e7b0d476e77ecae5917f43d50281245058ae5976b4198591a60d0c212` | NO |
| e | 10 | `8d71c73f72294c001104beebb89b60e0c30490f6a4d0d72963c0e38238e94db3` | NO |
| e | 11 | `f83ebcd1607792b94628ce6437b695f8383dfd3b5f047ba1db48fc1444d5c0c8` | NO |
| e | 12 | `066d578c5fe47f169a90659461a93b1d3a6a578383db47f6447dde5ac97c9896` | NO |
| e | 13 | `0b31fd439c2387c1fc8d4c02230808b04c3b091116243666843a78c417bf41cb` | NO |
| e | 14 | `044d4a4977943ba9217e8aa05079c2da107f32f4a4756098364439e8b8681846` | NO |
| e | 15 | `08593e046d4e69c9a1ed4d3d8ceb0a51746f7abb3a87c15aa91ce85bb09fb6aa` | NO |
| s | 0 | `4ff92fd66bdee173b35cc02475a4c12a31dcac62c836d605a0e198acb3017d4b` | NO |
| s | 1 | `18f76b9be30e776c9dcb81d868dfa264300ce8ec359681c237428886ba2ff33e` | NO |
| s | 2 | `aac1db0c74b1eb7531172752a28168feee2b1e71393a197399e8340a2a86ac38` | NO |
| s | 3 | `9af711649d03aa51344d3a62dd0803a5e241f6a2f2080929b78b846cbea00a0f` | NO |
| s | 4 | `79f504f835d9fc3cbe228ac54eb50f453fa4417558f0d8e656eb9c7ab72fac4b` | NO |
| s | 5 | `b9c0c6cd858ff71c8f94a53bdad9a53ad7d2b68fd7518ad837f1849d4d6f9112` | NO |
| s | 6 | `5c5979924d0dfb9f9288c14afb8b951a8830bb30e6832e8a08ead94abbcec953` | NO |
| s | 7 | `6b22f3f69ec1bd28e93edd1af9503a10ec04727a33fad81f1e1e3bbce0da3524` | NO |
| s | 8 | `5c9af41eda4869bda8eb6a4590be37c8172ed01f00323f6ca0f85d38da8a4901` | NO |
| s | 9 | `46654fdd742f0feac7605af47d6c5b2e120717365054a1efba7a9c4ef77d9ef4` | NO |
| s | 10 | `f9ee68d18661c623b008bfcd5d7ecb6b0d0c7796ca3b7752c677672339ae7e89` | NO |
| s | 11 | `8ab9783eb7426766b0292671b3df064a0df6795efa0a4e848cd654ecb87d1f3c` | NO |
| u | 0 | `64fe11c7e655e1d84df80c1ff7ca22b5453d3f61dd1f8c511065a902930797c8` | NO |
| u | 1 | `42b8ecd709d57072208521e1952251b4838a8c4065ee2b6273fe7974ec6fcecb` | NO |
| u | 2 | `473d61bcb63521d9d99d1fe8c0c97d55dd30ea472de3e1dcd8c1242eebe59bc5` | NO |
| u | 3 | `eb0afbe4dc666b66bdf186aec78058fe24c961399a4138511c932764c0603664` | NO |
| u | 4 | `21fccaeb52d7e7aec88880e96b22b35414ec247c6334daad9c823db83c70363d` | NO |
| u | 5 | `e61d407c0f9c52b5babf63c2319c281f58c501205421166e7d9a10488ab98da5` | NO |
| u | 6 | `4f2b9277a837f6f00ef575de62ca97a1267ecbdcfdf417d9321de17ae42c1469` | NO |
| u | 7 | `51c4beb71a1207afcd9716f92e5d735b88f8ae2a948e8b8622959436b038b7ec` | NO |
| t | 0 | `f432d91982eb52b32a4675332eafdf7eaa000d8788d8e1011b525c1db99151d2` | NO |
| t | 1 | `660948085616bf2eaabd01a2b2489cd10fea885ded5a3b9c9eedf50a03a1a5b8` | NO |
| t | 2 | `44fa18451fb3effadd4107296e2113a36bed8e93eef18332d017f798f3847625` | NO |
| t | 3 | `9c6a8be60a71bdd098525414cb9176dd8ff45aff6d0bccea554d08b0070367a5` | NO |
| t | 4 | `d49bd35dfbf5b9af96066901832ee5a4b9c9e976716ef867a7821777aa8ced44` | NO |
| t | 5 | `44e6f42aa0a70ebec3d0b170f01194853292ddfe061b6a85c10deba40abdaa56` | NO |
| t | 6 | `624edbe3d68366e5d3d146dd307b55823f76ec63bf4f319b8f5c16e5957f4c27` | NO |
| t | 7 | `5d3b0d1df510b65162280dbc32a71c4ad7100899443c3204659ae39eeadefef1` | NO |
| r | 0 | `7bf30dd2d6f2cf3037d1c9adfef7ea2784617af717f4555747a6e9d234f9a56e` | NO |
| r | 1 | `25fd8be566f2e189047645a2387ac80bbc9ac70d8e7fc91a46a2899346c2bf85` | NO |
| r | 2 | `2ee84ac7450de4417e636fb1a9ee6ab2e16dfeddbed824d99e91ea751615d9e1` | NO |
| r | 3 | `d64a6a231f7a0818da42bba142bf2d10dceabd117a2ac23a87537c0ba2e0e44c` | NO |
| r | 4 | `63c680e91921dff1c9da6138e719b120820bf2a2b33835019db4053fd76ebeb5` | NO |
| r | 5 | `112cfc0d024d9aa00e1a09e6a57f13ef0e7224bb14b6f6b052aa18241a11116d` | NO |
| r | 6 | `bc2976b6e9449abf5c13ef6f8e0b07f44834f655dcd9fb9bf853e3b5cd6c647a` | NO |
| r | 7 | `f9e4b7f6549297d7df4fe742c1a69ec186c5872868c2fde263cba86fc57ef92d` | NO |
| i | 0 | `b72151c699279f964f1acccd874e5920101ac2a0b115cbf14b82b3351de2ef61` | NO |
| i | 1 | `3df6379d117b485f3f0c8e4f9d111b47114f3e3b8ab1fd6d4ecf3dbe362928fa` | NO |
| i | 2 | `9166d0640872778e786f9d97bebc8049a63c73be8fb6850171efb62a9e07733c` | NO |
| i | 3 | `ef5568b6362f3b836f1fd5500886fb4a914f853f16389b4bc7d7b40a86ca0eeb` | NO |
| i | 4 | `3f92c74ea221fd4f1544e27e68413ed86d3e4b2fba264c8dda66932dc4a53fec` | NO |
| i | 5 | `d5e5ed2e8d499bd6d52b76a544df686df4e8cb958b1d99196cac94b3003b0637` | NO |
| i | 6 | `29b39e4cd333191f47731b19a3f4d2746daa6fcefebe07f721f608f1b261e73c` | NO |
| i | 7 | `638b72359eef65104ff154184d76f68fd7d3a75e6a6d388ea02b1dc0b772c6ee` | YES |

L2 D2-B1 v3 CANONICAL INCOME ARITHMETIC RECTIFICATION COMPLETE — READY FOR MICRO DELTA REVIEW
