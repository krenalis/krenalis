# Synthetic World v1 — Layer 2 D2-B1 v3

## Fast Demo Final Acceptance

**STATUS: PROPOSED / UNFROZEN**

**QUALITY: DEMO PLAUSIBLE**

**NON-INCOME GENERATOR: D2-B1 v0 @**
`94ec7560518f0077681a9b65547832bf46c706c5e7885c7da332bbd476ccc006`

**BASE QUALITY POLICY: D2-B1 v1 @**
`bf2e6a6dc9dbbed816190aef6705950567576c1a69833676bc4953319204236b`

**FAST INCOME GENERATOR: D2-B1 v2 @**
`36514d1c20a6bfab02b14feced6c2539fde28c5a60204a6d15629c8b9f82d6b9`

**v3 ACCEPTANCE OVERRIDES:**

1. Legacy branch model caps → **DIAGNOSTIC**.
2. V07 → **WARNING**.

**GENERATOR ARCHITECTURE CHANGE = NO. MODEL CHANGE = NO. QUALITY CHANGE = NO.**
**ACCEPTANCE CHANGE = NO. CANONICAL ARITHMETIC RECTIFICATION = YES.**

The fixed v2 candidate under §5.1's rectified arithmetic passes every effective
Income HARD check available in the temporary prototype. Six historical model-cap
findings and six V07 warnings remain visible. This proposed production basis is ready for focused
independent review, then D2-B2; it is not frozen or a D2-B2 success package.
The [supporting report](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v3_FAST_DEMO_FINAL_ACCEPTANCE_REPORT.md)
records the replay, witnesses, adversarial checks and integrity evidence.
The [arithmetic rectification report](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v3_CANONICAL_ARITHMETIC_RECTIFICATION_REPORT.md)
records the two arithmetic fixtures, exhaustive delta scan and updated hashes.

## 1. Authorities and frozen compatibility

The checksum-bound [v2 generator](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v2.md),
[v1 quality policy](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v1.md) and
[v0 non-Income recipe](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v0.md) retain their
owned scopes. The v2 supporting report was read completely and verified at
`6ad34b83e58d6df5106be00d0af0d01126b5d7bc334547c5ee6bf8d41d38a0be`;
it is supporting evidence, not acceptance authority. v1 and v2 were also read
completely. No authority hash mismatched.

| Frozen authority | Verified SHA-256 |
| --- | --- |
| Contract 1 v1 | `bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6` |
| Contract 2 v1 | `e9af32a0bb9b48bf8972a408a1369901a3f3a9fe5ecd76d5762a367c055bc098` |
| Contract 3 v1 | `df3ef0b52f37b587b8449cc71085dc9e5aca00f754bbd93e123c6fd5e3a4cb55` |
| Contract 4 v1 | `a5ecfb735270ffba3e6afea23d85e548dd6dc3f07a8e9dfc5294a84ad3576418` |
| Contract 5 v1 | `8a75054f793ef47e7071e3eb9fe04c0b32abfdac9223da859f95a333a1526e67` |
| D2-A v1 | `33ca4c80599dddb0564ea4d27262e430b1931e0368c8d1bee6ca6c10d8078be4` |

**Compatibility proof, established before either acceptance change:**

- Contract 5:63 defers empirical/model caps; :166–217 fixes the whole-EUR
  `0..4294967295` representation and rejects above-bound exact rates before
  ceiling. :217 expressly distinguishes a lower D2 model cap and states that
  Contract 5 freezes no empirical cap. Thus the legacy branch model caps
  **ARE NOT frozen Contract-5 representation bounds**.
- D2-A:71–83 requires uint32 supports/weights and checked uint64 totals;
  :388 leaves model caps and tail points to D2-B. Its :398–411 map/law-count
  limits and §21 resource charges/caps remain intact. Charges depend on
  bounded cardinalities and fixed widths, not the old branch EUR endpoints.
  Thus legacy branch model caps **ARE NOT required D2-A resource/integer
  bounds**.
- D2-A:409 explicitly says its capacity-witness values and spacings are not
  prescribed. :699 assigns actual fit thresholds to D2-B and requires
  repetition, occupancy, spacing, zero/body/tail evidence without prescribing
  a universal empirical repetition or bin threshold. The old V07 rule comes
  from v0 §27 and v1 §4, not a frozen authority. Searching all six frozen
  documents found no V07/REP01 numerical requirement. Thus the **V07 spacing
  threshold IS NOT a frozen Contract/D2-A semantic or engineering invariant**.

All six frozen documents remain byte-identical. This proof does not demote
their required evidence, semantics, support architecture or engineering bounds.

The requested profile is GPT-5.6 Sol xhigh; unavailable model/session telemetry
is not certified by this document. Synthetic CDP demonstration data are not
calibrated or validated for statistical inference, demographic estimation,
socioeconomic research, real-population fairness measurement or policy analysis.

## 2. Inheritance and precedence

Within each document's owned scope, precedence is:

```text
v3 explicit wording
>
v2 Income-generator wording
>
v1 quality wording
>
v0 imported generator wording
```

Frozen Contracts 1–5 and D2-A remain binding above these proposed documents.
v3 owns acceptance, synchronized reporting/verdict wording and §5.1's canonical
Income arithmetic rectification. The latter pins interpolation order and exact
scaling/ceiling, superseding ambiguous imported arithmetic wording; resulting
support/hash deltas are authoritative. It cannot tune any modeled value,
factor, curve, grid, bucket, map, age boundary, integerization or resource
architecture. v2 owns
the fixed fast Income generator; v0 owns unchanged non-Income generation.
v1's Demo Plausible policy is inherited through the effective table in §4.

§4 is the **sole acceptance authority for future D2-B2 v3**. No inherited
acceptance sentence, decision card, migration table, HARD-family heading,
ledger, stopping rule, verdict or production-success recommendation can add
an independent blocker. In particular, this supersedes:

- v0 §20's transported-support cap restriction and §§27–30/36–39 acceptance
  wording, decision cards and verdicts insofar as they would make a legacy
  branch monetary model cap or V07 spacing exceedance HARD. The original
  Q endpoint values remain unchanged base-curve numbers and annotations.
- v1 §4 S02's branch-cap clause and V07 severity, plus §§2–6 wording saying
  all visible-plausibility rows or REP01 gaps are HARD, or that fine tails'
  model caps remain HARD. Their comparison moves to D02; frozen
  representation bounds remain HARD.
- v2 §4 step 5's candidate rejection for a retained model-cap comparison;
  §7's S02 model-cap clause and V07 inclusion among HARD computations;
  §§8/11's stopping/verdict wording and the opening rejected-candidate card,
  solely where they would restore either superseded acceptance blocker.
- Historical reports' S02/V07 FAILs and rejection recommendations as present
  production blockers. They remain true historical findings under v2 policy;
  neither their bytes nor their historical verdict is rewritten.

v2's other algorithm adaptations are retained, not additional v3 changes.
A failure of any effective HARD rule still stops production; missing applicable
production evidence remains OPEN ISSUE. No historical fine-fit deviation is
revived through a report or a required evidence record.

## 3. Exact acceptance decisions and computations

### Decision A: S02 HARD scope and D02 diagnostic scope

**S02 remains HARD.** It requires Contract-5 Income meaning: eligible personal
gross monetary Income, annualized under current PopulationAnchorDate conditions
before personal taxes/contributions, with business earnings personally
attributed after operating costs. Household resources, turnover, wealth,
capital gains/principal, gifts/transfers, noncash compensation and offsetting
business losses remain excluded. Branch ownership, Company/Employer relations
and closed Income parent sets remain frozen. No hidden state is permitted.

| Branch | Authorized value parents | Exact pre-integer zero mass |
| --- | --- | ---: |
| e employee | Region, JobFamily, JobLevel, CurrentIndustry, Company.SizeBand[Employer] | .002 |
| s self-employed | AnchorAge, Region, JobFamily, Industry | .080 |
| u unemployed | AnchorAge, Education, Region | .350 |
| t student | AnchorAge, Education, Region | .650 |
| r retired | AnchorAge, Education, Region | .010 |
| i inactive | AnchorAge, Education, Region | .550 |

Zero is valid in every branch and means no eligible current personal Income,
never missingness/failure/inapplicability. Preserve exact branch zero masses
before integerization and final zero error ≤3×10^-7. Positive support stays
positive. Validate finite nonnegative exact rates ≤4294967295 before whole-EUR
ceiling; final supports are whole-EUR uint32, weights uint32, totals checked
uint64. Employee Employer is only the key for selected Industry/SizeBand;
employee age, Education, raw count except band derivation, employer identity
or random effect, Gender, ReferenceDate and finer geography remain excluded.

The sole removed HARD clause is **“support must remain ≤ inherited v0 branch
model cap.”** Its unchanged annotations are e1000000, s2000000, u150000,
t100000, r300000, i250000 EUR. Compare every final law maximum with its
historical cap, record every exceedance under D02 with branch, bucket/canonical
law ID, actual maximum, historical cap and frozen representation bound.

```text
frozen representation/semantic violation → HARD FAIL
historical branch model-cap exceedance → DIAGNOSTIC
DIAGNOSTIC — NOT A PRODUCTION SUCCESS CRITERION
```

Do not increase/replace the caps, clip/rescale laws or change curves/buckets.
Law-count, point-count, Employer C/I/S and resource caps retain their distinct
engineering meaning and HARD requirements.

### Decision B: V07 WARNING, unchanged metric and thresholds

Use final integer laws. Positive quantiles remove zero and renormalize; a
quantile is the first ascending value whose CDF reaches its fraction. Among
adjacent positive supports within the inclusive positive q10–q90 range, compute
the maximum gap and its support-pair witness. Ordinary limit remains
`max(500 EUR, .015 × positive median)`. The highest
`ceil(totalLawCount/10)` laws by pre-integer D0 population rho use the unchanged
`max(250 EUR, .01 × positive median)` limit; ties use branch order e,s,u,t,r,i
and earliest canonical tuple. Compute all laws and retain top-decile membership.

**Only severity changes: V07 HARD → WARNING.** Exceedances must be computed,
recorded and visible in demo-plausible reporting, including affected-law count,
branch, bucket/canonical ID, gap, exact historical limit and worst support-pair
witness. A V07 exceedance alone cannot block production. No threshold changes.

V06 remains HARD against dominant positive atoms; V08 remains HARD against
materially visible empty central 1000-EUR ranges. Each Fast Demo law has 384
positive authored points. Local spacing imperfections above the old
high-fidelity V07 threshold are useful warnings but are not sufficient by
themselves to invalidate a CDP demo. V07 remains useful disclosed evidence.

### Retained evaluation conventions

Form final integer-law branch mixtures under inherited pre-integer D0 before
computing quantiles; never average tuple medians. V04 positive-median reference
is original `Q_b(.5)`; unconditional-mean reference is
`(1-z_b) × integral(Q_b)`. Both candidate/reference ratios remain [.5,2],
inclusive; every unequal reference positive-median pair keeps its strict sign.
V05 uses all scalar-ordered v2 bucket-law pairs, including rare, at positive
q10/q25/q50/q75/q90/q95/q99; its larger uncompressed target quantile is direct
`m_k*Q_b(q)`. V06 unconditional positive atoms remain ≤.02 in every final law
and mixture, with zero exempt.

V08 uses the original Q_b positive q10–q90 range and fully internal half-open
1000-EUR bins `[1000k,1000(k+1))`; any bin with original positive-conditional
mass ≥.005 must have positive candidate branch mass. E05 support reservations
remain applicable. No substituted bin size, threshold or target is permitted.

Integerization TV ≤3×10^-7, maximum entry error ≤1.3×10^-7 and Income mean
change ≤1 EUR remain HARD; categorical entry error ≤10^-8 and EmployeeCount
mean change ≤.01 remain inherited. V03 retains the same ten standardized
contrasts and strict-sign computations in v1 §3. Engineering correctness,
successful construction, determinism, source integrity and frozen semantics
remain mandatory. Fast Demo reduces statistical precision, not correctness.

## 4. Effective v3 acceptance table

This complete table starts from v1's 21 rows, carries forward the already
authorized v2 algorithm interpretations, and applies only Decisions A/B.
ENGINEERING and SEMANTIC remain HARD; the principal Income visible-plausibility
blockers are V04, V05, V06 and V08. V07 is the disclosed spacing WARNING.
Every applicable HARD row needs full production evidence and PASS; missing
evidence is OPEN ISSUE. Warnings/diagnostics are nonblocking and never prompt
retuning. All six branches, including rare tuples, use the same policy.

| family | scope | severity | mechanical rule | demo consequence |
| --- | --- | --- | --- | --- |
| E01 ENGINEERING — inputs | All sources, crosswalks, normalized inputs | HARD | Exact inherited hashes/bytes/selectors/units; Trentino and ATECO preflights and full X2/X3 audit pass; reject malformed or substituted input. | Assets have the declared source basis. |
| E02 ENGINEERING — construction | All fitted rows and references; unchanged non-Income construction and fixed v2 Income | HARD | Finite nonnegative normalized laws and valid intentional support; execute inherited non-Income algorithms/arithmetic/ties and convergence/feasibility conditions. Income executes the exact v2 fixed cache/bucket/direct-law/map algorithm; no G/H/Ward/barycenter/member-envelope/compression obligation remains for Income. No empirical-distance limit is included. | No invalid law, failed fit or silently skipped operation. |
| E03 ENGINEERING — integer/canonical output | All ordinary laws, maps, factors and serialization | HARD | Exact v0 §9 Hamilton/ceiling/checked uint32–uint64 rules and positive/zero preservation; retain integerization-only probability/mean bounds; exact sorting/merge/GCD/dedup/first-occurrence IDs/schema/serialization. Employer factors/intervals never rescaled. Required quantiles finite and nondecreasing. | Correct bounded integers and one reproducible representation. |
| E04 ENGINEERING — reproducibility/isolation | Logical assets; runtime-owner obligations in their phase | HARD | Prescribed independent/reordered-input canonical outputs and checksums equal; separable identity material remains branch-local. Frozen runtime random access, non-interference and selected-branch-only evaluation remain required in their owning phase. | Retrieval order or unrelated branches cannot change facts. |
| E05 ENGINEERING — resources/capacities | Every production/construction phase, under frozen ownership | HARD | All inherited prospective resource formulas/ceilings pass; N=65536; count ≤128 points; frozen Income law maxima 512/256/64/64/64/64 plus unchanged v2 generator maxima 16/12/8/8/8/8 (total ≤60); ≤512 authored/canonical points, ≤240000 authored points; unchanged ≥128-distinct and top-decile ≥384-authored reservations, met directly by v2's 384 positive cells. Report actual charges and exact prospective constructor/index bounds. | Production executes within the frozen budgets. |
| S01 SEMANTIC — facts/DAG/relations | Every structural tuple and successful applicable result | HARD | Contracts1–5/D2-A exact types/parents/age coverage/gates/ownership; employee facts and eligible in-world Employer mandatory; SE/non-working forbidden facts absent; CompanyIndex finite-domain semantics, SizeBand/count and selected Company Industry relations exact; no hidden/forbidden input. | CDP joins and activity facts remain coherent. |
| S02 SEMANTIC — Income meaning/zeros | Every tuple, intermediate and final law; all branches | HARD | Contract5 personal gross annualized anchor-rate whole-EUR meaning/exclusions and authorized parent sets; 0 means no Income, positive stays positive; inherited branch z unchanged exactly pre-integer, final zero error ≤3×10^-7; finite nonnegative domain, exact pre-ceiling representation check and frozen uint32/checked uint64 bounds. Historical branch monetary model-cap comparison belongs solely to D02. | No household/turnover substitution, changed zero semantics or representation violation. |
| V01 VISIBLE PLAUSIBILITY — Company breadth | Analytic Company root/count mixture | HARD | All 14 intended Industries positive; all five intended SizeBands positive; count support includes each positive band and more than one positive count. No forced rare-giant realization. | Company catalogue spans the intended sectors and size bands. |
| V02 VISIBLE PLAUSIBILITY — category breadth | Six categorical models and branch marginals | HARD | Each intended-positive state remains positive; rows with multiple target states do not become deterministic; a target varying across parents does not become one identical canonical row everywhere; all six Activity branches positive under D0. | Major states and conditional variety cannot disappear. |
| V03 VISIBLE PLAUSIBILITY — categorical directions | Ten fixed broad contrasts in v1 §3 | HARD | Each nonzero modeled contrast has the same strict sign after integerization; zero target contrasts exempt. No per-row TV or percentage matching. | Age, qualification, activity, function and responsibility trends are preserved. |
| V04 VISIBLE PLAUSIBILITY — Income branch scale/rank | Six final branch mixtures under D0 | HARD | Each unconditional mean and positive median in [.5,2] × its original branch reference; pairwise unequal target positive-median ranks keep their sign. | Annual Income scale and branch order cannot shift by a factor of two or invert. |
| V05 VISIBLE PLAUSIBILITY — Income profile order | Scalar-ordered v2 bucket-law pairs and final laws, including rare | HARD | Existing SAN01: opposite-order positive q10/q25/q50/q75/q90/q95/q99 difference ≤max(100 EUR,.05 × larger corresponding uncompressed direct-scaled target quantile). Equality allowed. | Modeled socioeconomic profiles cannot acquire the retained reversal artifact. |
| V06 VISIBLE PLAUSIBILITY — positive Income atoms | Every final law and branch mixture | HARD | Existing REP01 maximum unconditional positive atom ≤.02; zero exempt. Mixture atom mass also ≤.02 (follows from the per-law bound). | One accidental positive amount cannot dominate a branch. |
| V07 VISIBLE PLAUSIBILITY — central Income spacing | Final laws; population top decile stricter | WARNING | Existing REP01 positive q10–q90 adjacent gap ≤max(500 EUR,.015 × positive median); top decile ≤max(250 EUR,.01 × positive median). Compute and report all exceedances, count and worst law/value/limit/support-pair witness. NOT A PRODUCTION SUCCESS CRITERION. | Local spacing imperfections remain visible warnings and do not block a demo when HARD rules pass. |
| V08 VISIBLE PLAUSIBILITY — Income occupancy | Six branch central histograms | HARD | Existing REP02 with §3 bin convention: no empty internal 1000-EUR bin carrying original positive target mass ≥.005. E05 distinct-support reservations also apply. | A population-sized central interval cannot vanish through repetition/collapse. |
| W01 FIDELITY — source/boundary caveats | Cheap or reused source comparisons and age-boundary evidence | WARNING | Preserve existing >5pp unexplained coarse-source and eight-map boundary warning limits where evaluated; record coverage and unexecuted scenarios. NOT A D2-B2 SUCCESS CRITERION. | Caveats remain visible without precision fitting. |
| W02 FIDELITY — finite-universe/model risks | Company tails and Employer fitting pressure | WARNING | Preserve >.05 analytic tail-absence warning and existing factor/cap-pressure records; no forced samples/retries. Actual factor/resource violations fail E02/E05. NOT A D2-B2 SUCCESS CRITERION. | Rare giant employers are not falsely guaranteed. |
| D01 FIDELITY — non-Income residuals | Company, categorical and Employer target/source distances | DIAGNOSTIC | Retain available fitted-cell/TV/quantile/mean/tail/exposure/DAG residuals with old limits as comparison annotations only. NOT A D2-B2 SUCCESS CRITERION. | Narrow official/model percentage matching is unnecessary. |
| D02 FIDELITY — Income precision and historical model caps | G-reference; tuple/segment/branch; pooled/finite/final; every law's historical monetary cap comparison | DIAGNOSTIC | Available N04/P/F/A/M fine distances, centres, q99/q999 and tail errors remain diagnostic, including M00/M01/P02/P03/P04 and 10^-5/.005 materiality analyses; no required materiality enumeration or superseded machinery. Record every legacy branch model-cap exceedance with actual maximum, unchanged historical cap and satisfied frozen representation bound. NOT A PRODUCTION SUCCESS CRITERION. | Microsegments, rare cells and fine tails may be statistically rough; a historical model-cap exceedance alone cannot reject valid Income. |
| D03 FIDELITY — reuse and tails | Income laws/mixtures and Company analytic events | DIAGNOSTIC | Record available collision probability, support multiplicity/distinct counts, non-central tail gaps, finite-universe variance/absence and smaller V05 reversals. NOT A D2-B2 SUCCESS CRITERION. | Finite support reuse is acceptable within the HARD artifact guards. |
| D04 FIDELITY — statistical sensitivity | Existing fixed scenarios and eight age-map evidence | DIAGNOSTIC | Reuse available results; identify missing scenarios. No complete perturbed fitting suite or sensitivity-distance PASS required. Production constants cannot change in response. NOT A D2-B2 SUCCESS CRITERION. | Demo production does not promise statistical robustness. |

**TOTAL 21; HARD 14; WARNING 3; DIAGNOSTIC 4.** HARD IDs are E01–E05,
S01–S02, V01–V06 and V08. WARNING IDs are V07, W01, W02. DIAGNOSTIC IDs
remain D01–D04. S02 stays one HARD row; its removed clause moves into existing
D02, adding no row. Only V07 changes row severity. Income has eleven applicable
HARD families: E01–E05, S01–S02, V04–V06 and V08.

G-reference nW1, material/rare tuple nW1, microtuple medians/quantiles,
regional conditional fidelity, q99/q999 fit, fine-tail fidelity and the
materiality/protected-segment framework remain nonblocking diagnostics.
Historical limits are comparison annotations, not replaced or enlarged limits.
Disclose expensive diagnostics as unexecuted when absent; do not call them PASS.
Frozen D2-A's evidence dimensions retain a concise record or traceable reference,
without an additional statistical robustness or fine-fit success gate.

## 5. Fixed generator, resources and phase ownership

The fixed v2 construction retains its modeled values under §5.1: 384 positive
authored cells per law; exactly 2304 cached branch-Q evaluations; maximum laws
e16/s12/u8/t8/r8/i8, total60; equal-cardinality exact-scalar buckets;
reachable lower-median representatives; direct scaled laws; whole-EUR ceiling;
exact probability merging/Hamilton integerization; direct tuple→law maps and
frozen first-occurrence canonical IDs. Canonical SE bucket6→ID7 and bucket7→ID6
remain unchanged. Canonical identical-law merging stays within each branch.

There is no D0 Income transport, dense G, H, Ward, population-weighted Income
clustering, global support allocation, greedy compression, adaptive law
optimization or per-law Q reevaluation. D0 is validation/non-Income state only.
No tuning, refit, support clipping/rescaling, new threshold or design iteration
is authorized. Performance timing remains diagnostic.

Pre-rectification historical evidence: one /tmp replay reproduced exactly 60 laws,
23040 positive points, 23100 points
including zero, and 76160 map slots (64960 reachable, 11200 employee sentinels).
Law/map/bucket bytes match historical v2 prototype evidence; all non-timing
results match. S02 semantic/frozen-bound scope, V04/V05/V06/V08, probability/
integer validity, canonicalization, map completeness, broad factor ordering,
available source/construction checks and prospective D2-A resources PASS.
Six cap comparisons are D02 diagnostics; six SE spacing exceedances are V07
warnings. No other effective prototype HARD failure exists.

Income input/retained charges remain 185664/371904 bytes; encoded estimate
685277 and decoded/work estimate 1629218; maps 152704; outside retained 1878264;
outside combined 14985464; Employer input/index 41944384; R2 43255546; joint
retained 45133810; constructor peak 72095482 bytes. Employer architecture,
fixed widths, phase lifetimes, full E/D reservations and capacities are unchanged.
The encoding is an in-memory estimate, not canonical D2-B2 bytes. Actual
construction must still preflight allocation and account for all live state.

This focused acceptance replay does not certify independent complete D2-B2
reproduction/serialization or runtime-owned obligations. Future D2-B2 proves
numerical assets, canonical equality, prospective bounds and identity material;
Protocol/implementation proves random access, actual selection, payloads,
non-interference and selected-branch non-evaluation in its owning phase.
No canonical D2-B2 assets, Run A/Run B, logical-assets.json, canonical manifest
or package checksum file is produced here. Prototype material stays under /tmp.

### 5.1. Canonical Income arithmetic — normative rectification

Frozen Contract 5 §5 requires the exact pre-ceiling EUR rate and mathematical
ceiling, with checked integer or exact rational conversion. D2-A §§4–5/16/21
retain integer supports/weights, canonical IDs and fixed resource bounds.
Neither authority permits a new context rounding of the scaling product before
ceiling. MODEL CHANGE = NO; QUALITY CHANGE = NO; acceptance §4 is unchanged.

**Q interpolation regime.** Parse authored finite-decimal knots exactly. For
j=0..383, convert `(2j+1)/768` by dividing its exact integer numerator by its
exact integer denominator under CPython 3.13.2's canonical Decimal context:
precision 96, ROUND_HALF_EVEN, Emin=-999999999, Emax=999999999. Select the
piecewise-linear segment by the inherited knots (an internal knot belongs to
the segment starting there; the final endpoint has its authored Q value).
Execute precisely this ordered operation graph under that same context:

```text
d_u = u - u0
d_q = q1 - q0
p   = d_u * d_q
r   = p / (u1 - u0)
q   = q0 + r
```

Each subtraction, multiplication, division and addition rounds sequentially
under Decimal96/ROUND_HALF_EVEN. In the fourth step evaluate `u1-u0` after
`p`, then divide `p` by that result. Cache the resulting finite `q` directly;
no additional 10^-40 quantization applies to this cache. The equivalent source
expression is `q0 + (u - u0) * (q1 - q0) / (u1 - u0)` with this operation
graph. Ratio-first `q0 + (u-u0)/(u1-u0)*(q1-q0)` is not canonical.

**Scaling/ceiling regime.** Treat the canonical cached `base=q` and the exact
reachable bucket representative `m` as already-canonical finite operands:

```text
base         = cached canonical Decimal96 Q result
m            = exact canonical bucket representative
scaled_exact = exact finite-decimal mathematical product(base, m)
support      = mathematical ceil(scaled_exact)
```

Check `0 <= scaled_exact <= 4294967295` exactly before conversion; all authored
positive cells require `scaled_exact > 0`. Keep checked arithmetic before
narrowing and the inherited probability merge/Hamilton/GCD rules unchanged.
`scaled_exact` MUST NOT first be rounded back to Decimal96. A conforming
implementation may use `Decimal.as_integer_ratio()`, exact integer
coefficient/exponent multiplication, Fraction or another exact rational
representation. For example, multiply the two integer ratios exactly, then
apply integer ceiling. Decimal96 `scaled=m*base; support=ceil(scaled)` is
forbidden when its context rounding changes the mathematical ceiling.
Arbitrarily increasing Decimal precision is not the rule. Do not recompute
an exact real-number Q from the original knots in place of the cached operand.

The two regimes are intentionally different: ordered Decimal96 interpolation
canonicalizes Q; exact multiplication of its finite cached value precedes
whole-EUR ceiling. A producer must not infer either rule from the other.
For compatibility-sensitive canonical arithmetic, mathematical equivalence is
insufficient. Follow the specified operation graph; expression reassociation,
compiler/FMA substitutions, ratio-first interpolation, premature context
rounding of the exact scale product and binary float cannot substitute where
they can affect canonical output.

**Normative fixture A — interpolation: PASS → 65164.** Employee branch e,
bucket/canonical law 11, positive authored cell j=170, `u=Decimal96(341/768)`,
segment u0=.25/u1=.50, q0=24000/q1=32000, representative `2.15712000`:

```text
cached Q = 30208.3333333333333333333333333333333333333333333333333333333333333333333333333333333333333333334
scaled_exact = 65163 + 1.43808e-91
support = 65164
```

Ratio-first interpolation yields support 65163 with exact scaling and MUST
FAIL this fixture. No fixture tolerance or approximate comparison is allowed.

**Normative fixture B — scaling: PASS → 5107.** Mechanically identified from
the fixed candidate: inactive branch i, bucket/canonical law 7, j=104,
`u=Decimal96(209/768)`, segment u0=.25/u1=.50, q0=3500/q1=7500,
representative `1.324800`:

```text
cached Q = 3854.16666666666666666666666666666666666666666666666666666666666666666666666666666666666666666667
Decimal96-rounded product = 5106
exact finite-decimal product = 5106 + 4.416e-93
rounded-product ceiling = 5106  (nonconforming)
exact-product ceiling = 5107    (canonical)
```

**Rectified replay.** Exhaustively comparing all 60×384=23040 authored positive
supports with historical ordered-interpolation/Decimal96-rounded-scaling
semantics changes exactly six supports, all inactive bucket/canonical ID 7:

| j | Historical EUR | Canonical EUR |
| ---: | ---: | ---: |
| 104 | 5106 | 5107 |
| 119 | 5934 | 5935 |
| 134 | 6762 | 6763 |
| 149 | 7590 | 7591 |
| 164 | 8418 | 8419 |
| 179 | 9246 | 9247 |

Maximum absolute delta is 1 EUR; one branch and one law are affected. No model
tuning follows. Two independent implementations (ordered Decimal plus exact
rational scaling; Fraction-based per-operation half-even rounding plus integer
coefficient/exponent scaling) match all 384 base Q values per branch, all
23040 supports and all 60 canonical laws. No duplicate supports or within-branch
identical laws occur. Counts stay 60 laws, 23040 positive/23100 total points;
first-occurrence IDs and tuple maps stay unchanged, including SE 6→7/7→6.

Using inherited compact sorted UTF-8 JSON plus LF for temporary prototype
evidence, updated laws SHA-256 is
`d1de8c861b2803c2394f004fa9789833d7c6cbf96bfef481eb9f5493118c6b53`.
Maps/buckets retain respectively
`1335b1d86a3c03b939b3e4a8f524d8f52daf8281fde8cbb5695de428a4261ddb` /
`509eccd8b0933ede12e4d8beb11656693f44fb1b6fd034f7a780737414abbba5`.
Law content changes only for i7; its identity material must reflect those
supports even though its ID stays 7. These are prototype checksums, not a
D2-B2 package. S02/V04/V05/V06/V08 and law/integer/zero/order/map/resource
checks PASS in the replay scope. V07 retains six SE WARNINGs; all six historical
cap comparisons remain DIAGNOSTIC. The §4 table stays 21/14/3/4.

Exactly 2304 Q evaluations per reconstruction, 60 laws and 23040 authored
positive cells are preserved. Exact finite multiplication of those cells is
required; no D0 transport, dense G, H, Ward, clustering or optimization returns.
All §5 resource charges/estimates remain unchanged. The arithmetic rules are
generic over asset curves and representatives: they depend on neither ATECO
nor twenty Regions nor Italian bucket-count logic; current Contracts still
give the IT Income values EUR meaning. No multi-country implementation occurs.

## 6. Future compatibility and review

Current pack = **IT**. Retain future core **CountryPack**, **PopulationMix**
and **multinational Company workforce footprints** as compatibility notes only.
No multi-country mechanism is implemented. Twenty Regions, ATECO and nominal
EUR remain IT/current-contract properties, not generic runtime requirements.
No Person-country==Employer-country or single-country workforce assumption
is introduced.

This rectification modifies only this document and adds its arithmetic report.
Prior reports/reviews and all other pre-existing files,
Contracts/D2-A, historical evidence, semantic.go and raw Git index are unchanged;
nothing is staged or committed. Micro delta review should verify §5.1's two
arithmetic rules/fixtures, exhaustive six-support delta, updated law hashes,
unchanged acceptance/maps/IDs and retained HARD results. This basis remains
PROPOSED / UNFROZEN; D2-B2 production is not run by this rectification.

## Revision history — canonical Income arithmetic surgical rectification

2026-09-17: pre-rectification v3 SHA-256:
`785a9d9bd575d85b5a91a82bf49d5ac9bb69a8b8f2aaa41891da61458b996ae8`.
Independent Astra review identified two canonical-arithmetic underspecifications:
Q interpolation operation order and exact scaling/ceiling semantics. Section
5.1 pins ordered Decimal96 Q evaluation, then exact finite-operand multiplication
before mathematical whole-EUR ceiling, consistent with frozen Contract 5.
Six i7 supports change by +1 EUR; all modeled values, acceptance thresholds/
severities, Fast Income architecture, maps/IDs and resources remain unchanged.
Historical Sol/Astra reviews and reports retain their original bytes and
verdicts against the old v3 SHA. v3 remains PROPOSED / UNFROZEN; no v4 is created.

L2 D2-B1 v3 CANONICAL INCOME ARITHMETIC RECTIFICATION COMPLETE — READY FOR MICRO DELTA REVIEW
