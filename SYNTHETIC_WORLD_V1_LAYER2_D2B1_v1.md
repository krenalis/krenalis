# Synthetic World v1 — Layer 2 D2-B1 v1

## Demo Plausible Production Basis

**STATUS: PROPOSED / UNFROZEN**

**QUALITY TARGET: DEMO PLAUSIBLE**

**GENERATOR BASIS: D2-B1 v0 SHA-256**
`94ec7560518f0077681a9b65547832bf46c706c5e7885c7da332bbd476ccc006`

**SUPERSEDES: v0 production-quality acceptance criteria for future D2-B2 production.**

**GENERATOR NUMERICS CHANGED = NO**

### 1. Product decision and historical authority

Synthetic data must be deterministic, internally coherent, heterogeneous and
credible when inspected or used in a CDP demo. It is not intended to reproduce
official population statistics precisely and must not be represented as suitable
for statistical inference about real populations.

Normatively, Demo Plausible assets are synthetic demonstration data. They are
not calibrated or validated for statistical inference, demographic estimation,
socioeconomic research, fairness measurement against real populations, or policy
analysis.

D2-B1 v0 is the historical high-fidelity lineage: Plausible+ and later Plausible
acceptance work. D2-B1 v1 is the proposed Demo Plausible production basis. v0 is
preserved byte-for-byte. Historical failures remain facts under their then-current
authority; this overlay neither asserts a v0 PASS nor revises old reviews.

The requested execution/review profile is GPT-5.6 Sol xhigh. This document does
not certify session metadata that the environment does not expose.

### 2. Normative inheritance and supersession boundary

> Except where D2-B1 v1 explicitly supersedes quality acceptance, reporting or
> production-success criteria, the numerical generator recipe is inherited
> unchanged from D2-B1 v0 SHA-256
> `94ec7560518f0077681a9b65547832bf46c706c5e7885c7da332bbd476ccc006`.

The reference is [D2-B1 v0](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v0.md).
The checksum, rather than a subsequently edited file with the same name, binds
the imported bytes. The following inventory identifies inheritance; it does not
restate or replace its numerical recipe.

| Imported generator portion | v0 location | v1 numerical change |
| --- | --- | --- |
| Source set/hashes, parsing, Trentino, ATECO, full X2/X3 audit, crosswalks | §§2–8, 33–35 and rectification history | NONE |
| CompanyCount, Company Industry, EmployeeCount | §§10–12 | NONE |
| Education, PrimaryActivity, employee JobFamily/JobLevel, SE JobFamily/Industry | §§13–18 | NONE |
| Employer C/I/S, cap fitting, selection and relational support | §19; frozen Contracts/D2-A | NONE |
| D0, fit sequencing, exact arithmetic and precision | §§8–9 | NONE |
| Income zero masses, Q curves, Region/tuple factors and target transport | §§20–26 | NONE |
| Exact-profile grouping, scalar order, H, adjacent Ward, original-function barycenters, branch law caps | §28 algorithm | NONE |
| Support allocation, compression, integerization and canonicalization | §§9, 27–28 algorithms; §32 schema/serialization | NONE |
| Resource formulas and prospective limits | §31; frozen D2-A | NONE |

D0 remains required offline fitting/production state, never runtime state.
No source, crosswalk, coefficient, age map, cap, grid, pooling decision or
allocator is selected again. In particular, the fixed greedy compression
sequence and budget stopping rule do not depend on loosening its quality gates.

The sole production acceptance authority of this overlay is the 21-row table
in §4, with §3 defining its computations. v0 acceptance sentences, matrix
severities, materiality/protected-segment machinery, report-completeness rules
for fine-fit metrics, verdicts and acceptance-related ledger wording are
superseded wherever they would impose an additional statistical-success gate.
This applies across v0 §§9–30, 32, 36–39, including decision cards and risks;
the algorithms and serialization in those sections remain imported.

Only acceptance/reporting/success changes. Semantic contracts, probability-law
validity, executability and convergence, integer correctness, canonicalization,
determinism, resource safety, referential integrity and source integrity cannot
be waived. Correctly constructing G is HARD; its small distance from a denser
analytic-reference polyline is DIAGNOSTIC. A failed calculation or invalid
reference is HARD; exceeding old N04 approximation tolerances alone is not.

Contracts 1–5 and D2-A remain frozen under the six hashes in v0 §1. Their
CompanyIndex/dense finite universe, activity gating and ownership, Employer
relations, Contract-5 Income meaning, Gender/finer-geography exclusions, random
access, branch isolation, integer bounds and resource ceilings remain intact.
No new age/activity semantic exclusion is introduced: unusual adult students,
early retirees and low-education executives remain valid. If implementation
requires violating a frozen contract, stop and report the contradiction.

### 3. Mechanical evaluation conventions

Use inherited canonical state/tuple order and pre-integer D0 for analytic
population mixtures. A conditional quantile is the first value whose CDF reaches
the specified fraction; positive quantiles remove zero and renormalize. Form
mixtures of laws before computing their quantiles. Never average tuple medians
to obtain a branch median. Final checks use canonical integer laws; intermediate
checks use the representations actually available at that stage.

For the coarse categorical direction row V03, compute the following fixed
contrasts on both inherited pre-integer targets and canonical rows. Average
other parent states equally to standardize the comparison; average states
within each named group equally. Education and JobLevel scores are their
zero-based canonical ordinals. Low Education is the first three states; high
Education is tertiary plus doctoral. The functional group is technical-and-
scientific, professional-and-creative, and care-and-education.

- Last minus first age-bucket mean Education score.
- Last minus first Activity age-bucket probability, separately for employee,
  student and retired; high minus low Education working probability
  (employee plus self-employed).
- High minus low Education functional-group probability, separately for
  employee and SE JobFamily.
- Last minus first age-bucket, high minus low Education, and functional-group
  minus the first six JobFamilies: mean employee JobLevel score.

These ten contrasts preserve modeled directions, without a percentage-fit
requirement. A nonzero target contrast requires the same strict sign in the
candidate; a zero target contrast imposes no direction. This sign rule is also
used for pairwise branch positive-median ranks in V04. It introduces no
empirical tolerance or extra runtime input. Income socioeconomic directions
use inherited scalar order in V05, rather than dozens of conditional centres.

V04 has the only new numerical tolerance: final branch unconditional mean
and positive median must each be between one-half and twice their own original
§20 branch target. Bounds are inclusive and identical for all six branches.
Reference mean is `(1-z_b) × integral(Q_b)`; reference positive median is
`Q_b(.5)`. This checks scale, not precise calibration. Halving/doubling the
annual Income centre or average changes CDP spending/earning segments by an
entire factor of two; that is a demo-scale error rather than a microcell
distance excess. The limit is a round magnitude guard, not an adjustment of
old 2–5% fit thresholds. The focused six-branch check is in the supporting
report; finite-law evidence remains required in future production.

For V06/V07 retain the existing v0 §27 broad artifact computations unchanged.
Positive atom probabilities are unconditional law probabilities; zero is
exempt. The stricter top decile is the highest `ceil(totalLawCount/10)` laws
by D0 population mass rho, with inherited branch/earliest-tuple ties.
V08 uses the original branch target's positive q10–q90 range, 1000-EUR
half-open bins `[1000k,1000(k+1))`, and fully internal bins; fail an empty
candidate bin when its original positive-conditional
target mass is at least .005. This is branch occupancy, not .005 joint-segment
materiality. Tail gaps outside the central range remain diagnostic.

Each HARD row requires evidence for its full applicable production scope;
missing evidence is OPEN ISSUE and cannot support D2-B2 SUCCESS. Frozen D2-A
evidence dimensions (including rare support, tails, repetition, construction
risk and all eight age maps' sensitivity) still need a concise record or a
traceable evidence reference. Fine-fit diagnostics may reuse verified evidence;
unexecuted expensive diagnostics must be labeled unexecuted, never PASS.
The old exhaustive perturbation/refitting suite is not a v1 success prerequisite.
There is no separate statistical robustness gate hidden in evidence reporting.

Phase ownership stays frozen: D2-B2 proves numerical assets, prospective
constructor/index bounds and separable identity material. Runtime random access,
actual index selection, payload generation and non-evaluation tests remain
Protocol/implementation obligations, not a demand to implement them in D2-B2.

### 4. Exhaustive normative acceptance table

There are exactly four top-level quality families. ENGINEERING, SEMANTIC and
VISIBLE PLAUSIBILITY are HARD. FIDELITY is WARNING/DIAGNOSTIC. Production
succeeds only with every applicable HARD row evidenced and PASS; warnings
are disclosed without retuning. All six Income branches use the same policy,
including rare tuples. No other v0 fidelity gate remains implicitly HARD.

| family | scope | severity | mechanical rule | demo consequence |
| --- | --- | --- | --- | --- |
| E01 ENGINEERING — inputs | All sources, crosswalks, normalized inputs | HARD | Exact inherited hashes/bytes/selectors/units; Trentino and ATECO preflights and full X2/X3 audit pass; reject malformed or substituted input. | Assets have the declared source basis. |
| E02 ENGINEERING — construction | All fitted rows, references, pooling and compression | HARD | Finite nonnegative normalized laws and valid intentional support; execute inherited algorithms/arithmetic/ties and convergence/feasibility conditions; G/H correctly constructed; each barycenter G knot stays within member envelope ±10^-40. No empirical-distance limit is included. | No invalid law, failed fit or silently skipped operation. |
| E03 ENGINEERING — integer/canonical output | All ordinary laws, maps, factors and serialization | HARD | Exact §9 Hamilton/ceiling/checked uint32–uint64 rules and positive/zero preservation; retain §9 integerization-only probability/mean bounds; exact sorting/merge/GCD/dedup/IDs/schema/serialization. Employer factors/intervals never rescaled. Required quantiles finite and nondecreasing. | Correct bounded integers and one reproducible representation. |
| E04 ENGINEERING — reproducibility/isolation | Logical assets; runtime-owner obligations in their phase | HARD | Prescribed independent/reordered-input canonical outputs and checksums equal; separable identity material remains branch-local. Frozen runtime random access, non-interference and selected-branch-only evaluation remain required in their owning phase. | Retrieval order or unrelated branches cannot change facts. |
| E05 ENGINEERING — resources/capacities | Every production/construction phase, under frozen ownership | HARD | All inherited prospective resource formulas/ceilings pass; N=65536; count ≤128 points; Income law caps 512/256/64/64/64/64, ≤512 authored/canonical points, ≤240000 authored points; unchanged 128-distinct and top-decile 384-authored reservations. Report actual asset charges plus exact prospective constructor/index bounds. | Production executes within the frozen budgets. |
| S01 SEMANTIC — facts/DAG/relations | Every structural tuple and successful applicable result | HARD | Contracts1–5/D2-A exact types/parents/age coverage/gates/ownership; employee facts and eligible in-world Employer mandatory; SE/non-working forbidden facts absent; CompanyIndex finite-domain semantics, SizeBand/count and selected Company Industry relations exact; no hidden/forbidden input. | CDP joins and activity facts remain coherent. |
| S02 SEMANTIC — Income meaning/zeros | Every tuple, intermediate and final law; all branches | HARD | Contract5 personal gross annualized anchor-rate whole-EUR meaning/exclusions; 0 means no Income, positive stays positive; inherited branch z unchanged exactly pre-integer, integer error ≤3×10^-7; valid nonnegative domain and each inherited branch cap. | No household/turnover substitution or changed zero semantics. |
| V01 VISIBLE PLAUSIBILITY — Company breadth | Analytic Company root/count mixture | HARD | All 14 intended Industries positive; all five intended SizeBands positive; count support includes each positive band and more than one positive count. No forced rare-giant realization. | Company catalogue spans the intended sectors and size bands. |
| V02 VISIBLE PLAUSIBILITY — category breadth | Six categorical models and branch marginals | HARD | Each intended-positive state remains positive; rows with multiple target states do not become deterministic; a target varying across parents does not become one identical canonical row everywhere; all six Activity branches positive under D0. | Major states and conditional variety cannot disappear. |
| V03 VISIBLE PLAUSIBILITY — categorical directions | Ten fixed broad contrasts in §3 | HARD | Each nonzero modeled contrast has the same strict sign after integerization; zero target contrasts exempt. No per-row TV or percentage matching. | Age, qualification, activity, function and responsibility trends are preserved. |
| V04 VISIBLE PLAUSIBILITY — Income branch scale/rank | Six final branch mixtures under D0 | HARD | Each unconditional mean and positive median in [.5,2] × its original branch reference; pairwise unequal target positive-median ranks keep their sign. | Annual Income scale and branch order cannot shift by a factor of two or invert. |
| V05 VISIBLE PLAUSIBILITY — Income profile order | Scalar-ordered cluster pairs and final laws, including rare | HARD | Existing SAN01: opposite-order positive q10/q25/q50/q75/q90/q95/q99 difference ≤max(100 EUR,.05 × larger corresponding uncompressed target quantile). Equality allowed. | Modeled socioeconomic profiles cannot acquire the retained reversal artifact. |
| V06 VISIBLE PLAUSIBILITY — positive Income atoms | Every final law and branch mixture | HARD | Existing REP01 maximum unconditional positive atom ≤.02; zero exempt. Mixture atom mass also ≤.02 (follows from the per-law bound). | One accidental positive amount cannot dominate a branch. |
| V07 VISIBLE PLAUSIBILITY — central Income spacing | Final laws; population top decile stricter | HARD | Existing REP01 positive q10–q90 adjacent gap ≤max(500 EUR,.015 × positive median); top decile ≤max(250 EUR,.01 × positive median). | Central values cannot present the retained sparse-catalogue gap artifact. |
| V08 VISIBLE PLAUSIBILITY — Income occupancy | Six branch central histograms | HARD | Existing REP02 with §3 bin convention: no empty internal 1000-EUR bin carrying original positive target mass ≥.005. E05 distinct-support reservations also apply. | A population-sized central interval cannot vanish through repetition/collapse. |
| W01 FIDELITY — source/boundary caveats | Cheap or reused source comparisons and age-boundary evidence | WARNING | Preserve existing >5pp unexplained coarse-source and eight-map boundary warning limits where evaluated; record coverage and unexecuted scenarios. NOT A D2-B2 SUCCESS CRITERION. | Caveats remain visible without precision fitting. |
| W02 FIDELITY — finite-universe/model risks | Company tails and Employer fitting pressure | WARNING | Preserve >.05 analytic tail-absence warning and existing factor/cap-pressure records; no forced samples/retries. Actual factor/resource violations fail E02/E05. NOT A D2-B2 SUCCESS CRITERION. | Rare giant employers are not falsely guaranteed. |
| D01 FIDELITY — non-Income residuals | Company, categorical and Employer target/source distances | DIAGNOSTIC | Retain available fitted-cell/TV/quantile/mean/tail/exposure/DAG residuals with old limits as comparison annotations only. NOT A D2-B2 SUCCESS CRITERION. | Narrow official/model percentage matching is unnecessary. |
| D02 FIDELITY — Income precision | G-reference; tuple/segment/branch; pooled/finite/final | DIAGNOSTIC | Available N04/P/F/A/M fine distances, centres, q99/q999 and tail errors remain diagnostic, including M00/M01/P02/P03/P04 and 10^-5/.005 materiality analyses; no required materiality enumeration. NOT A D2-B2 SUCCESS CRITERION. | Microsegments, rare cells and fine tails may be statistically rough. |
| D03 FIDELITY — reuse and tails | Income laws/mixtures and Company analytic events | DIAGNOSTIC | Record available collision probability, support multiplicity/distinct counts, non-central tail gaps, finite-universe variance/absence and smaller V05 reversals. NOT A D2-B2 SUCCESS CRITERION. | Finite support reuse is acceptable within the HARD artifact guards. |
| D04 FIDELITY — statistical sensitivity | Existing fixed scenarios and eight age-map evidence | DIAGNOSTIC | Reuse available results; identify missing scenarios. No complete perturbed fitting suite or sensitivity-distance PASS required. Production constants cannot change in response. NOT A D2-B2 SUCCESS CRITERION. | Demo production does not promise statistical robustness. |

Counts: **21 rows; 15 HARD, 2 WARNING, 4 DIAGNOSTIC**. Income has **12 HARD
families**, counting every applicable shared family: E01–E05, S01–S02 and
V04–V08. Required support/quantile/integer correctness is included, not omitted
from that count. V01–V03 cover non-Income components. A diagnostic deviation
cannot fail production unless independent evidence fails one of the HARD rows.

### 5. Compact v0 → v1 migration

| v0 family | v1 disposition | reason |
| --- | --- | --- |
| Frozen meanings, DAG, payloads, source-concept allowlists, Company/Employer relations | HARD | S01/S02 preserve exact semantics. |
| Source/hash/parsing/crosswalk audits; mathematical law construction and algorithm convergence | HARD | E01/E02 preserve input validity and executability. |
| Integerization/canonicalization/serialization, reproducibility, resources/reservations | HARD | E03–E05 preserve engineering guarantees. |
| N04 G-reference approximation distance/centre tolerances | DIAGNOSTIC | E02 requires correct G construction; D02 removes fine-grid proximity as success. |
| Company/categorical/Employer narrow target and sample-fit tolerances | DIAGNOSTIC | D01; broad coverage and directional checks are V01–V03. Later sampling does not inherit a v0 precision blocker. |
| Income zero/support/quantile validity and SAN03 mathematical envelope | HARD | S02/E02/E03; no rare-tuple exemption. |
| SAN01 ordered profiles; REP01 atom/gaps; REP02 branch occupancy | HARD | V05–V08 retain existing artifact guards. |
| P01–P05 pooling fidelity, F01–F07 finite fidelity, A01–A04 final fidelity | DIAGNOSTIC | D02 for distances/fit; exact zeros remain S02 and final scale V04. |
| M00/M01/M02 and material/rare/protected-segment acceptance framework | SUPERSEDED | No materiality-based production blocker or mandatory combinatorial scan; available metrics are D02. |
| Fine q99/q999 and tail reproduction | DIAGNOSTIC | D02/D03; ordering, caps, atoms and central occupancy remain HARD. |
| Source/boundary warnings and tail-absence/cap-pressure caveats | WARNING | W01/W02 retain disclosed limitations, never generator retuning. |
| REP03 reuse, smaller reversals, statistical sensitivity | DIAGNOSTIC | D03/D04; preserve cheap evidence without a robustness gate. |
| REP04 exhaustive fine-fit reporting and old Plausible/Plausible+ verdict/report terminology | SUPERSEDED | §4 needs complete HARD evidence and disclosed optional diagnostic coverage, not 70-row success reasoning. |

### 6. Legacy findings and reporting

The historical e/s G-reference nW1 values approximately
.000104228592495243 and .000273082651426828, u/r material-tuple nW1
.046571817168149 and .059686796310062, and t rare-tuple median error
11.007988420963658% are **DIAGNOSTIC — NOT A D2-B2 SUCCESS CRITERION**.
Their old .0001/.04/10% limits remain historical comparison annotations;
none is increased. No Ward/cap/Income change follows from reclassification.
The same assets still fail if they independently violate a §4 HARD rule.

For future D2-B2, retain the inherited logical asset schema and serialization,
source/crosswalk/fit configuration/checksum/resource manifests, and complete
parent grids. Replace the quality report's 70-row/materiality verdict with
this table's IDs, PASS/OPEN ISSUE evidence by component and the separate
WARNING/DIAGNOSTIC record. A report title may be `demo-plausible.md`; it is
reporting metadata, not a new asset or runtime schema. Cite available historical
diagnostics and explicitly disclose missing finite/canonical evidence.

This task performs focused evidence reuse only. It neither produces canonical
D2-B2 assets nor freezes v1. If a future run fails an engineering, semantic or
retained artifact gate, report the blocker before any separately authorized
generator reconsideration; never retune within production.

### 7. Non-blocking future compatibility

The current numerical basis represents the Italian country pack.
Future Synthetic World core architecture is expected to support:

- CountryPack.
- PopulationMix.
- Multinational Company workforce footprints.

D2-B1 v1 does not define those runtime mechanisms. Italian Regions, ATECO and
EUR remain properties of the imported Italian assets/current contracts. This
overlay adds no generic-runtime requirement for twenty Regions, ATECO, shared
Person/Employer country, permanently single-country workforce, or universal EUR.

L2 D2-B1 v1 DEMO PLAUSIBLE COMPLETE — READY FOR FOCUSED INDEPENDENT REVIEW
