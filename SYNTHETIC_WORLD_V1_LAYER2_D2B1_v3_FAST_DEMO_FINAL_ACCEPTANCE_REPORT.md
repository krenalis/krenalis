# D2-B1 v3 — Fast Demo Final Acceptance Report

**STATUS: PROPOSED / UNFROZEN. QUALITY: DEMO PLAUSIBLE.**

## 1. Decision

The unchanged fixed v2 Income candidate passes all effective HARD checks
available in the prototype. Exactly two acceptance changes finalize this
proposed production basis: historical branch monetary model caps become D02
diagnostics; V07 becomes a warning with unchanged computation/thresholds.
Six findings of each kind remain disclosed. Ready for focused independent
review, then D2-B2. No freezing or D2-B2 production is claimed.

## 2. Authorities

| Authority | Verified SHA-256 |
| --- | --- |
| D2-B1 v0, unchanged non-Income generator | `94ec7560518f0077681a9b65547832bf46c706c5e7885c7da332bbd476ccc006` |
| D2-B1 v1, base quality policy; read completely | `bf2e6a6dc9dbbed816190aef6705950567576c1a69833676bc4953319204236b` |
| D2-B1 v2, fixed Income generator; read completely | `36514d1c20a6bfab02b14feced6c2539fde28c5a60204a6d15629c8b9f82d6b9` |
| v2 report, supporting evidence; read completely | `6ad34b83e58d6df5106be00d0af0d01126b5d7bc334547c5ee6bf8d41d38a0be` |

Contracts 1–5 and D2-A match all six frozen hashes in the
[v3 specification](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v3.md) §1, before/after.
No authority mismatch occurred. The requested profile is GPT-5.6 Sol xhigh;
unavailable model/session metadata is not certified.

## 3. Frozen-authority compatibility proof

**PASS, proved before creating v3.** Contract 5:166–217 freezes whole-EUR
`0..4294967295`, checked exact pre-ceiling conversion and nonnegative personal
Income meaning. :63/:217 explicitly defer and distinguish empirical/model
caps: no branch-specific empirical cap is frozen. D2-A:71–83 freezes uint32
support/weight and checked uint64 totals; :388 assigns model caps/tail points
to D2-B. Its map/law-count/cardinality/resource limits constrain counts and
fixed-width storage, independently of historical branch EUR endpoints.

Therefore legacy branch model caps **ARE NOT frozen Contract-5 representation
bounds AND ARE NOT required D2-A resource/integer bounds**. Every replay
support satisfies the unchanged representation bound.

D2-A:409 says the capacity-witness values/spacings are not prescribed;
:699 assigns actual fit thresholds to D2-B while requiring spacing/body/tail
evidence. The V07 numerical rule originates in v0 §27/v1 §4, not frozen
Contracts/D2-A. **V07 spacing threshold IS NOT a frozen semantic or engineering
invariant.** Required evidence is still reported. All six frozen files were
searched; none imposes V07/REP01's numerical threshold. The proof capture is
`/tmp/krenalis-d2b1-v3-59408vmv/frozen-compatibility-before.txt`.

## 4. Exact two acceptance changes

**A:** S02 stays HARD for meaning/exclusions, annualization, whole EUR,
nonnegative domain, branch zeros/masses, positive preservation, authorized
parents and frozen integer/representation bounds. Only “support ≤ inherited
v0 branch model cap” leaves its HARD scope. Historical cap comparisons remain
unchanged D02 annotations, **DIAGNOSTIC — NOT A PRODUCTION SUCCESS CRITERION**.
No larger cap, clipping, rescaling, curve/bucket change or repair occurred.

**B:** V07 HARD→WARNING; same ordinary `max(500,.015×positive median)` and
top-decile `max(250,.01×positive median)` EUR limits. All computations,
membership/ties, counts and witnesses remain reportable. V06 still blocks
dominant positive atoms and V08 materially visible empty central 1000-EUR
ranges. With 384 positive authored points per law, local spacing imperfections
above the old high-fidelity limit are useful warnings and do not alone
invalidate a CDP demo.

## 5. Effective table and inherited-blocker audit

The sole future D2-B2 v3 acceptance authority is specification §4's complete
21-row table: **14 HARD, 3 WARNING, 4 DIAGNOSTIC**. HARD: E01–E05, S01–S02,
V01–V06, V08. WARNING: V07/W01/W02. DIAGNOSTIC: D01–D04. The cap comparison
moves into existing D02; no row is added. Only V07 changes row severity.
Existing v2 E02/V05/E05 generator interpretations are retained.

The requested branch/model-cap, V07, spacing/gap, S02, verdict/card, HARD and
production-success search captured 207 inherited matches across v0/v1/v2 and
their v1/v2 reports. Specification §2 explicitly supersedes conflicting v0
transport/card/matrix prose, v1 S02/V07/HARD-family/migration prose, v2 step5/
§7/stopping/verdict prose and old rejection recommendations. Historical FAILs
remain historical facts. Scoped precedence is v3>v2>v1>v0; frozen authorities
remain binding. Acceptance cannot change generator numerics. No old blocker
leaks through; no fine-fit/materiality/tail metric becomes an implicit blocker.
Search and row-comparison evidence stays under the temporary audit directory.

## 6. One unchanged fixed-v2 replay

One temporary replay used the surviving corrected v2 `prototype.py`; an AST
comparison proved that only EVIDENCE/WORK path literals changed. The historical
script SHA is `060f69ea3d409b461eabe1058751ccaaff56d13553ebf7ec5cb14e837b1fd5cd`.
Its hash-bound producer and all 14 embedded/readable modules matched. CPython
3.13.2, Decimal96/ROUND_HALF_EVEN/exponent bounds, binary-float rejection,
exact probability arithmetic and inherited Hamilton rules were retained.

Replay at `/tmp/krenalis-d2b1-v3-59408vmv/replay` reproduced **2304 Q calls,
60 laws, 23040 authored/final positive points, 76160 map slots**. Exactly
64960 assignments are reachable; 11200 employee zero-band slots are sentinel
65535. All laws have 385 distinct points including zero and remain referenced.
SE bucket6→ID7/bucket7→ID6 and all canonical first-occurrence IDs are unchanged.

Law, map and bucket bytes match the v2 report's discarded prototype evidence
exactly. All generation/result fields except diagnostic timing match; the
complete resource result matches exactly. No unexpected discrepancy occurred.
Original v2 S02/V07 FAIL labels were retained in historical-policy replay logs;
the separate effective-policy check applies only the authorized classifications.

D0 entered after laws/maps were built, solely for mixtures/validation. Needed
non-Income/D0 state was reconstructed by unchanged inherited tooling, with no
new fitting basis, parameter selection or refit decision. No superseded Income
module or producer main/publication entry point was executed.

## 7. All branch-cap diagnostics

Bucket/canonical IDs are zero-based. All six findings satisfy the frozen bound.
Unchanged other caps are s2000000/u150000/r300000/i250000 EUR; no law in those
branches exceeds its historical cap. Overall catalogue maximum is 1468132 EUR.

| Branch | Bucket / canonical law ID | Actual maximum EUR | Historical model cap EUR | Frozen representation bound EUR | Frozen bound |
| --- | --- | ---: | ---: | ---: | --- |
| e employee | 13 / 13 | 1079361 | 1000000 | 4294967295 | PASS |
| e employee | 14 / 14 | 1229969 | 1000000 | 4294967295 | PASS |
| e employee | 15 / 15 | 1468132 | 1000000 | 4294967295 | PASS |
| t student | 5 / 5 | 103490 | 100000 | 4294967295 | PASS |
| t student | 6 / 6 | 128217 | 100000 | 4294967295 | PASS |
| t student | 7 / 7 | 163928 | 100000 | 4294967295 | PASS |

## 8. All V07 warnings

**Six affected SE laws**, all outside the population top decile. Its six laws
e0/e1/r4/e2/e3/i5 pass the unchanged stricter rule. Threshold comparisons are
exact rational; finite decimals below are exact displays.

| Branch | Bucket / canonical law ID | Positive median EUR | Worst central gap EUR | Historical limit EUR | Support-pair witness EUR |
| --- | --- | ---: | ---: | ---: | --- |
| s SE | 6 / 7 | 24959 | 512 | 500 | 42245→42757 |
| s SE | 7 / 6 | 27428 | 563 | 500 | 46986→47549 |
| s SE | 8 / 8 | 30109 | 618 | 500 | 51578→52196 |
| s SE | 9 / 9 | 33149 | 680 | 500 | 56786→57466 |
| s SE | 10 / 10 | 37402 | 767 | 561.03 | 63305→64072 |
| s SE | 11 / 11 | 45060 | 925 | 675.9 | 93824→94749 |

Worst warning: **s bucket 11 / law 11, gap 925 EUR >675.9 EUR**, witness
93824→94749, reachable representative 1.612875. It is also the worst relative
and absolute threshold exceedance. All 60 law computations are retained.

## 9. Retained HARD results and evidence limits

| Check | Effective result | Evidence |
| --- | --- | --- |
| S02 | HARD / PASS | Frozen meaning/parent inspection; exact pre-integer zeros; finite nonnegative whole-EUR/positive-preserving rates and checked bounds in replay; cap comparisons separated |
| V04 | HARD / PASS | All six mean/positive-median ratios in [.5,2]; all 15 reference-rank pairs preserved |
| V05 | HARD / PASS | 2086 ordered bucket-pair/quantile checks; maximum opposite difference 0 EUR |
| V06 | HARD / PASS | Every law and all six mixtures ≤.02 after merging/integerization |
| V08 | HARD / PASS | Required central bins e44/s39/u21/t13/r30/i23, total170; none empty |
| Available engineering | HARD / PASS | Source/hash/preflight/full X2/X3 checks; successful exact construction; valid laws/integers/GCD/sorted support; total referenced maps; reversed-input bucket/map/canonical order; resource arithmetic |

Maximum final zero/entry error and TV remain `199/3435973836` (<5.8×10^-8);
maximum integerization mean change `1358454797/916259689600` EUR (<.001483).
All inherited integer guards pass. The 1454756 one-factor comparisons have
1111486 strict visible bucket increases, 343270 ties and zero reversals; every
inherited factor axis has visible increases. No successful v2 result changed.

**ALL EFFECTIVE HARD PASS is scoped to checks available in this prototype.**
Independent complete D2-B2 runs/serialization/checksums, full non-Income
V01–V03 review and runtime-owner tests are not newly certified here. Their
applicable HARD obligations remain required in their owning phase. This
acceptance finalization is not D2-B2 SUCCESS. Expensive fine-fit/sensitivity
diagnostics were not executed or called PASS.

## 10. Adversarial A–H policy results

Policy fixtures used exact comparisons against unchanged thresholds; they
were not alternate generated candidates or tuning. Inclusive boundaries were
also checked. A/G use actual fixed-candidate witnesses with its HARD set PASS.

| Case | Witness | Observed result |
| --- | --- | --- |
| A | e13 maximum 1079361>historical 1000000, valid frozen bound/semantics and retained HARD results | May PASS; D02 diagnostic recorded |
| B | Support 4294967296; also exact pre-ceiling 4294967295.01 | HARD FAIL S02/E03; MaxUint32 itself stays representable |
| C | Mean and positive median separately at 2.01×reference | HARD FAIL V04; exactly 2× retains inclusive validity |
| D | e median 28000/s33897 reverses reference rank while scale remains valid; ordered-profile opposite difference 251>250 EUR | HARD FAIL V04; V05 as applicable |
| E | Positive unconditional atom .020000001 | HARD FAIL V06; .02 boundary remains valid |
| F | Required employee [16000,17000) bin's candidate mass set to 0 | HARD FAIL V08; target .005 boundary also fails if empty |
| G | SE bucket 6/law 7, gap 512>500 EUR, other HARD rules pass | WARNING; production not blocked |
| H | Company-turnover/household meaning; unauthorized employee AnchorAge | HARD FAIL S02 |

No protection gap was found in these attacks or the fixed candidate. Exact
E02 construction remains HARD, preventing arbitrary altered laws from passing
the complete set by satisfying only marginal visible metrics. This is not a
claim that the statistical diagnostics establish population authenticity.

## 11. Resources and unchanged performance architecture

**D2-A PASS; all resource results exactly equal v2.** Counts remain 60 laws,
23100 total points/23040 positive, 76160 map slots. Each law meets ≤512 and
the inherited ≥128-distinct/top-decile ≥384-authored reservations directly.

| Charge / prospective reservation | Bytes | Applicable ceiling |
| --- | ---: | --- |
| Income input / retained | 185664 / 371904 | Included in frozen aggregate bounds |
| Maps | 152704 | Complete frozen grids |
| Compact whole-model encoded estimate | 685277 | 6.5 MiB inherited reservation; frozen 8 MiB |
| Decoded/work estimate | 1629218 | 6 MiB inherited reservation; frozen 8 MiB |
| Outside retained / combined | 1878264 / 14985464 | 8 MiB /24 MiB |
| Employer input/index | 41944384 | 64 MiB each |
| R2 / joint retained | 43255546 /45133810 | 72 MiB /80 MiB |
| Conservative serial-index constructor peak | 72095482 | 128 MiB |
| Maximum per-index canonicalization work | 2097152 | 32N, N 65536 |

v2's decoded/work itemization, full E/D live reservations, 1 MiB bookkeeping/
Employer allowances and all phase charges remain identical. Encoding is an
in-memory prototype estimate, not emitted D2-B2 bytes. Actual implementation
still must honor before-allocation guards and backing/scratch lifetimes.

Exactly 2304 cached Q calls, 23040 authored positive points, 60 laws; no transport,
clustering, optimization or per-law Q reevaluation. Runtime tuple addressing
and bounded ≤385-point law sampling, Employer architecture/work are unchanged.
Single replay construction .873481125s, D0 validation reconstruction 14.339706417s,
offline validation 2.454999416s: **diagnostic timings only**, no benchmark or
speed threshold. No performance architecture change or tuning occurred.

## 12. Future CountryPack compatibility

**PASS. Current pack=IT; multi-country implemented=NO.** Future CountryPack,
PopulationMix and multinational Company workforce footprints remain notes.
No generic requirement for 20 Regions, ATECO, universal EUR, shared Person/
Employer country or single-country Company workforce was added.

## 13. Repository integrity and package sanity

**Package sanity PASS:**

```bash
GOCACHE=/tmp/krenalis-d2b1-v3-gocache GONOSUMDB='*' GOPROXY=off go test ./tools/fakedata
git diff --check
git diff --cached --quiet
```

The package passed in 13.265s. This tests the existing package, not an
implemented Layer-2 generator. Git checks and new-document whitespace/LF/link
checks passed. No staging or commit occurred.

**Repository integrity PASS.** Before/after branch `property-semantics`, HEAD
`76fa0fff109ca6abad62f55c8761275455842dc5`, subject `Add Synthetic World Layer 1`
are unchanged. Full unfiltered porcelain-v1 and human statuses, working/cached
binary diffs, raw index SHA, all pre-existing file hashes/symlink targets and
all root historical Synthetic World evidence hashes were captured under
`/tmp/krenalis-d2b1-v3-59408vmv`. Removing exactly the two added untracked
records reproduces the complete before porcelain status bytes. All 100136
pre-existing inventory entries are identical; only the two authorized files
are added. No logical-assets.json, D2-B2 package, canonical manifest/checksum
file or Run A/Run B was produced/executed.

Raw Git index SHA before=after:
`9bf5ddbc729f55523ac70792666243018b250ae81053a8e4d741071e003d81c3`.
Working binary diff and semantic.go diff before=after,1432 bytes:
`0311e8b4c5aa5eb470112a26e96e24b1e2946ac350b35851676992bbe61eb6f4`.
Cached binary diff stays empty. semantic.go content before=after:
`06a7ff690b3e12cdaf623e3ffb36003d1675a6aec69b3ee59e9ead428cc17842`.
v0/v1/v2/v2-report and Contracts/D2-A retain the hashes in §2/specification §1.
Complete historical evidence hashes are in `before-hashes.json` and their
matching after capture; inventories do not embed a self-dependent report hash.

## 14. v3 SHA and final self-review

New D2-B1 v3 SHA-256:
`785a9d9bd575d85b5a91a82bf49d5ac9bb69a8b8f2aaa41891da61458b996ae8`.
The report's own final SHA-256 is supplied separately after its final bytes.

**Self-review PASS:** GENERATOR CHANGE=NO; FAST INCOME CHANGE=NO; frozen
Contracts/D2-A unchanged and compatibility proved; S02 HARD/PASS with model
caps diagnostic; V04/V05/V06/V08 HARD/PASS; V07 WARNING with unchanged limits;
engineering/semantic HARD intact; 21/14/3/4 table; no inherited blocker leak;
unchanged fixed replay all available effective HARD PASS; all cap/spacing
witnesses recorded; no tuning/new thresholds/generator changes; resources
unchanged/PASS; future CountryPack preserved; package/integrity PASS.

## 15. Independent-review recommendation and requested result

Proceed directly to focused independent review of the two acceptance decisions,
frozen-bound distinction, table/precedence, replay identity and complete witness
records. Then D2-B2 may produce/evidence canonical assets under the effective
v3 table, retaining all phase-owned HARD obligations. This basis is proposed,
unfrozen; no production run was performed here.

```text
D2-B1 v0 SHA: 94ec7560518f0077681a9b65547832bf46c706c5e7885c7da332bbd476ccc006
D2-B1 v1 SHA: bf2e6a6dc9dbbed816190aef6705950567576c1a69833676bc4953319204236b
D2-B1 v2 SHA: 36514d1c20a6bfab02b14feced6c2539fde28c5a60204a6d15629c8b9f82d6b9
New D2-B1 v3 SHA: 785a9d9bd575d85b5a91a82bf49d5ac9bb69a8b8f2aaa41891da61458b996ae8
Generator changed: NO
Fast Income changed: NO
Acceptance changes: 2
Frozen-authority compatibility: PASS
Legacy branch model caps: DIAGNOSTIC; 6 findings
S02: HARD / PASS
V04: HARD / PASS
V05: HARD / PASS
V06: HARD / PASS
V07: WARNING; 6 warnings
Worst warning: s bucket 11/law 11 gap 925 EUR, historical limit 675.9 EUR,
  witness 93824→94749 EUR
V08: HARD / PASS
Acceptance rows: 21
HARD: 14
WARNING: 3
DIAGNOSTIC: 4
Fixed v2 replay: ALL EFFECTIVE HARD PASS (available prototype scope, §9)
Q evaluations: 2304
Income laws: 60
Positive points: 23040
D2-A: PASS
Contracts: PASS
Performance architecture changed: NO
Multi-country implemented: NO
Future CountryPack compatibility: PASS
Package sanity: PASS
Repository integrity: PASS
v3 report SHA-256: supplied separately after final bytes; not self-embedded
```

L2 D2-B1 v3 FAST DEMO FINAL ACCEPTANCE COMPLETE — READY FOR FOCUSED INDEPENDENT REVIEW
