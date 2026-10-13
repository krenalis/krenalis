# Synthetic World v1 — Layer 2 D2-B1 v2

## Fast Demo Income Generator

**STATUS: PROPOSED / UNFROZEN**

**QUALITY TARGET: DEMO PLAUSIBLE**

**NON-INCOME GENERATOR BASIS: D2-B1 v0 @**
`94ec7560518f0077681a9b65547832bf46c706c5e7885c7da332bbd476ccc006`

**QUALITY BASIS: D2-B1 v1 @**
`bf2e6a6dc9dbbed816190aef6705950567576c1a69833676bc4953319204236b`

**INCOME GENERATOR: FAST DEMO v2**

**NON-INCOME GENERATOR CHANGE = NO**

**FIXED-DESIGN PROTOTYPE VERDICT: FAIL — S02 and V07. DESIGN STOPPED.**
This document records the one requested algorithm and its failed prototype;
it does not authorize production, freezing, tuning or a gate waiver.
The [supporting report](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v2_FAST_DEMO_INCOME_REPORT.md)
contains the measured results and exact failure witnesses.

## 1. Decision, authorities and boundaries

The high-fidelity Income generator is unnecessarily expensive for Synthetic
World CDP demos and is superseded. v2 preserves semantic correctness,
determinism, broad socioeconomic ordering, visible plausibility and engineering
safety as requirements. Its finite catalogue replaces high-fidelity transport,
pooling and compression; statistical fidelity remains nonblocking.

The checksum-bound [v0](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v0.md) supplies unchanged
non-Income generation, branch positive Q curves, branch zero masses, authorized
parent factors, taxonomies, age maps, numerical helpers and serialization
principles. The checksum-bound [v1](SYNTHETIC_WORLD_V1_LAYER2_D2B1_v1.md) supplies
Demo Plausible quality policy except for the explicit Income adaptations here.
All authored Income curves, factors, grids and buckets are modeled assumptions,
not newly researched or fitted population statistics. Synthetic demo data are
not validated for statistical inference, demographic estimation, socioeconomic
research, real-population fairness measurement or policy analysis.

Contracts 1–5 and D2-A are frozen and unmodified:

| Authority | SHA-256 |
| --- | --- |
| Contract 1 v1 | `bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6` |
| Contract 2 v1 | `e9af32a0bb9b48bf8972a408a1369901a3f3a9fe5ecd76d5762a367c055bc098` |
| Contract 3 v1 | `df3ef0b52f37b587b8449cc71085dc9e5aca00f754bbd93e123c6fd5e3a4cb55` |
| Contract 4 v1 | `a5ecfb735270ffba3e6afea23d85e548dd6dc3f07a8e9dfc5294a84ad3576418` |
| Contract 5 v1 | `8a75054f793ef47e7071e3eb9fe04c0b32abfdac9223da859f95a333a1526e67` |
| D2-A v1 | `33ca4c80599dddb0564ea4d27262e430b1931e0368c8d1bee6ca6c10d8078be4` |

The interrupted handoff was read completely; its SHA-256 is
`5b7ad3f7c0f6c059ce83dd8a9ad1b8f199dee391cd008e4e27d20701e2cf8170`.
It is supporting implementation/performance evidence only. Interrupted Income
objects are neither authoritative nor reusable production assets.

The requested profile is GPT-5.6 Sol xhigh; this document does not certify
model/session metadata unavailable from the execution environment.

## 2. Supersession table

| Disposition | Components |
| --- | --- |
| Inherited unchanged | Contracts/D2-A; complete non-Income generator; Income semantics and parents; branch zero masses; Q base shapes; parent factors; Demo Plausible quality intent; canonical engineering principles |
| Superseded for Income | National D0 quantile transport/F0; dense G production and LOCKS; H; adjacent Ward state; population-weighted pooling; original-function barycenters/member-envelope production; global support scoring/allocation; greedy splits/interval-loss optimization; old 512/256/64/64/64/64 catalogue; cluster-specific acceptance wording |
| Added | Fixed 384-point positive grid; equal-cardinality distinct-scalar buckets; reachable lower-median representative; direct scaled finite laws; direct tuple maps; Income-specific E02/V05 and mechanical E05 adaptations |

Unchanged non-Income scope includes sources/hashes, parsing, Trentino, ATECO,
full X2/X3 audit and crosswalks; CompanyCount, Company Industry and EmployeeCount;
Education and PrimaryActivity; employee JobFamily/JobLevel and SE
JobFamily/Industry; Employer C/I/S, fitting/structure/selection architecture;
all eight age maps; integer helpers and resource architecture. No source,
coefficient, non-Income fit decision, taxonomy or lifecycle boundary is selected
again. There is no feedback from Income to these components.

Historical algorithms may remain in historical code. Canonical v2 Income
construction must neither import nor execute them, including indirectly.

## 3. Exact semantics, parents and zeros

Income remains the nonnegative personal gross annualized current monetary
Income rate at PopulationAnchorDate, before personal taxes and mandatory
personal social contributions, publicly in whole nominal EUR under Contract 5.
Business/professional earnings are personally attributed after operating costs.
Losses do not offset other eligible inflows. It is neither turnover, salary-only,
pension-only, wealth nor household resources.

Eligible remuneration, positive personal business earnings, pensions, benefits,
scholarships, monetary interest/dividends/rents/royalties and other eligible
inflows retain their meaning. Family gifts/support, interpersonal transfers,
inheritance, capital/wealth, loans, returned principal, capital gains, sale
proceeds, imputed rent and noncash compensation remain excluded. There is no
hidden component, socioeconomic latent, history or eligibility state.

Every successful Person has one mandatory Income. PrimaryActivity selects
exactly one branch; absent occupational branches are not evaluated. Zero means
no eligible current personal Income and is valid in all branches; it never
means missingness, failure or inapplicability. ReferenceDate, Gender and finer
geography remain excluded. Existing ownership, identity isolation and Employer
relational semantics remain frozen.

| Branch, canonical order | Exact value parents | Exact branch zero mass | Maximum buckets |
| --- | --- | ---: | ---: |
| e employee | Region, JobFamily, JobLevel, CurrentIndustry, Company.SizeBand[Employer] | .002 | 16 |
| s self-employed | AnchorAge, Region, JobFamily, Industry | .080 | 12 |
| u unemployed | AnchorAge, Education, Region | .350 | 8 |
| t student | AnchorAge, Education, Region | .650 | 8 |
| r retired | AnchorAge, Education, Region | .010 | 8 |
| i inactive | AnchorAge, Education, Region | .550 | 8 |

Employee Employer is only the relational key for the selected Industry/SizeBand.
AnchorAge, Education, raw EmployeeCount, Employer random effect/identity and
Gender are not employee Income value parents. Raw count may only derive
SizeBand. Equal selected Industry/SizeBand and fixed other parents/innovation
cannot acquire an employer-specific Income effect. SE and nonworking branches
gain no Company/Employer facts. The inherited five Income age maps coarsen only
their authorized AnchorAge input and retain all 55 supported ages, 21–75.

## 4. Normative fast algorithm

Canonical Income production is exactly:

```text
inherited branch Q curve
→ evaluate once on fixed 384-point positive grid
→ inherited authorized-parent scalar
→ deterministic coarse scalar bucket
→ direct scaled finite law
→ whole-EUR merge/integerization
→ tuple→law map
```

Use CPython 3.13.2, Decimal precision 96, ROUND_HALF_EVEN and exponent bounds
±999999999, with binary-float operations rejected, as in v0 §8. Finite-decimal
factor products are exact at this precision. Exact rational probability
accumulation, inherited Hamilton rules and checked integer widths remain
normative. Never introduce rounding, averaging or fitting of scalar profiles.

1. **Cache the six base vectors.** For j=0..383, form the exact rational
   `(2j+1)/768` and convert canonically to Decimal96. Evaluate each inherited
   piecewise-linear Q_b once at each resulting u_j. Cache `base_b[j]` as its
   canonical Decimal96 value. The vector is not a dense G polyline and is not
   interpolated or adaptively refined. No additional value-quantization rule
   is imported from the superseded G/H machinery.
2. **Enumerate authorized tuples.** Use the inherited reachable grids and
   parent-factor tables in v0 §§20–26. Compute `m(t)` as the exact product of
   only its authorized factors. No D0 mass enters this product or enumeration.
3. **Partition distinct scalars.** Deduplicate exact-equal values and sort
   ascending numerically. With n distinct scalars, use
   `K=min(n, branch maximum)`. Bucket k has indices
   `[floor(k*n/K), floor((k+1)*n/K))`, for k=0..K−1. Every bucket is nonempty.
   This partitions distinct values equally by cardinality, irrespective of
   tuple multiplicity or population mass.
4. **Select the reachable representative.** For `[start,end)`, use scalar
   index `start+(end-start-1)//2`. This is the lower median, never a mean,
   geometric mean, centroid or optimized value.
5. **Build each finite law.** Scale only the cached vector:
   `x_j=ceil(m_k*base_b[j])`. Check finite nonnegative rates and the uint32
   representation bound before conversion. Apply Contract-5 ceiling exactly;
   no positive point may become zero. Validate the retained branch model cap
   separately; cap failure rejects the candidate, without clipping or repair.
6. **Merge and integerize.** Each of the 384 authored positive cells has exact
   unconditional probability `(1-z_b)/384`. Sort supports, merge equal
   positive EUR values and sum their exact rational probabilities. Add the
   exact branch zero atom z_b. Apply inherited v0 §9 positive-reserving
   Hamilton apportionment at M=2^32−1, including its probability quantization
   and preservation rules, remainder ties by ascending support. Use checked
   uint64 additions/totals, omit intentional zero weights, GCD-canonicalize
   and validate uint32 supports/weights. Preserve the authored cell count
   independently of duplicate merging.
7. **Complete the maps and canonical IDs.** Map every exact scalar to its
   unique bucket and every reachable tuple to that bucket's canonical law.
   Preserve the employee zero-SizeBand sentinel slice. Apply frozen D2-A
   first-occurrence canonical IDs and within-branch exact-law deduplication
   as described in §6. Require all laws referenced and no missing cell.

Exactly 6×384=2304 branch-Q calls produce the base vectors. Bucket-law scaling,
support generation and map completion never call Q again. There are at most
60 laws and 23040 authored positive cells, plus at most 60 zero atoms: 385
authored pairs per law, below D2-A's 512-pair ceiling. There is no runtime
scaling, interpolation, adaptive grid or authored-support allocator.

## 5. D0 separation

D0 remains inherited offline state for non-Income fitting, Demo Plausible
validation, branch/population mixtures and diagnostics. It is **not an input**
to v2 Income law construction, scalar buckets, representatives, supports or
tuple→law mapping. No F0 or `Q_b(F0(m*Q_b(u)))` transport exists in this path.

The prototype constructed all laws/maps before reconstructing D0. Only the
validation phase consumed D0 tuple and activity masses, including population
top-decile membership. This is an execution boundary, not merely a renamed
transport calculation. D0 and source/provenance heaps are never runtime assets.

## 6. Deterministic order, identity and serialization

| Object | Exact order |
| --- | --- |
| Branches | e,s,u,t,r,i |
| Scalar values | Ascending exact numerical value, exact equality deduplicated |
| Buckets/authored catalogue labels | Ascending k; boundaries and lower-median indices from §4 |
| Positive grid/cache | Ascending j=0..383 |
| Supports and apportionment ties | Ascending whole-EUR value; zero first |
| Employee tuple traversal | Region → JobFamily → JobLevel → CurrentIndustry → SizeBand; zero,micro,small,medium,large |
| SE tuple traversal | Income[s]AgeBucket → Region → JobFamily → Industry |
| Nonworking tuple traversal | Own Income age bucket → Education → Region |
| Dimension states | Exact frozen D2-A §5 order; first dimension outermost, last innermost; ages/buckets ascending |
| Canonical law IDs | Branch-local consecutive IDs by first occurrence of canonical support/weight content in the complete parent map, as frozen by D2-A §5 |
| Tuple→law maps | Complete dense maps in the preceding tuple order; employee zero band is sentinel 65535 |

Ascending bucket labels are deterministic authored labels, not permission to
change frozen canonical ID semantics. In this prototype SE buckets 6 and 7
receive canonical IDs 7 and 6 respectively; other bucket numbers coincide
with branch-local IDs. V05 sorts by representative, never by canonical ID.
This preserves D2-A while providing branch→ascending-bucket catalogue order.

Exact-law equivalence is support/weight content after sorting/merging/GCD.
D2-A's mandatory within-branch identical-law canonicalization remains; no
cross-branch deduplication changes identity. All 60 prototype laws are distinct
within their branches. Source/physical checksums remain evidence, not universal
generation seeds. Each Income branch owns separable output-sensitive model
material; other branches and Contract-4 generation remain isolated.

Future serialization inherits v0 §32 canonical compact UTF-8 JSON principles,
integer numerical laws/maps, prescribed field ordering and terminal LF.
Concrete stochastic paths, Components, binary identity framing and sampling
remain Protocol-owned. This design task emits no canonical package or manifest.

## 7. Explicit acceptance adaptations

v1's 21-row table remains the quality basis: ENGINEERING, SEMANTIC and VISIBLE
PLAUSIBILITY are HARD; FIDELITY is WARNING/DIAGNOSTIC and nonblocking. Only
Income-specific algorithm references are superseded as follows.

| Row/surface | v2 Income interpretation | Unchanged requirement |
| --- | --- | --- |
| E02 | Execute §4's exact fast algorithm, complete fixed caches/buckets/maps and valid finite laws; no G/H/Ward/barycenter/compression execution or member-envelope requirement | Executability, exact arithmetic/ties, finite nonnegative normalized laws and intentional support; non-Income E02 unchanged |
| V05 | Ordered v2 scalar-bucket law pairs, including rare; compare all ordered pairs at positive q10/q25/q50/q75/q90/q95/q99; opposite difference ≤max(100 EUR,.05×larger corresponding uncompressed direct-scaled target quantile) | Original SAN01 guard, equality permitted, no socioeconomic reversal artifact |
| E05 | Preserve frozen D2-A maxima, primitive widths/charges and phase caps; use v2's smaller 16/12/8/8/8/8 catalogue and fixed 384 positive cells directly | ≤512 authored/canonical points, ≤240000 authored points, inherited ≥128 distinct points including zero and ≥384 authored slots for highest population decile |
| Other imported Income algorithm wording | v0/v1 fit sequencing, cards, acceptance/reporting/migration/ledger prose requiring national transport, G/LOCKS/H, Ward, pooled barycenters, global allocation or greedy compression is explicitly superseded by §§2/4 | Quality intent, semantic/integer/resource/source validity and applicable non-Income algorithms |

The old law-count ceilings remain architectural upper bounds in D2-A; v2's
smaller ceilings are an additional generator restriction. Old cluster target
quantiles in V05 become the direct uncompressed `m_k*Q_b(q)` quantiles; these
reference checks do not re-evaluate the cached grid per law. Diagnostic rows
do not require superseded machinery to run merely to generate precision metrics.
No inherited non-Income acceptance wording changes.

Retain these HARD computations without loosening:

- **S02:** unchanged Contract-5 meaning, parents and z_b, exact pre-integer
  zeros and final zero error ≤3×10^-7; positive support remains positive,
  nonnegative and within the inherited branch model caps. Those caps remain
  e=1000000,s=2000000,u=150000,t=100000,r=300000,i=250000 EUR. They are D2-B
  modeled caps, not Contract-5 structural maxima.
- **V04:** mix final integer laws under inherited pre-integer D0, then compute
  positive median and unconditional mean. Each ratio to the original branch
  reference is in [.5,2], inclusive. Reference median is Q_b(.5); mean is
  `(1-z_b)*integral(Q_b)`. Preserve every unequal branch-median rank sign.
- **V06:** every final law and branch mixture has maximum unconditional
  positive atom ≤.02; zero is exempt. Check after EUR merging/integerization.
- **V07:** among adjacent positive supports in final positive q10–q90, gap
  ≤max(500 EUR,.015×positive median). Highest `ceil(totalLawCount/10)` laws
  by D0 population rho use ≤max(250 EUR,.01×positive median); ties follow
  inherited branch and earliest-tuple order.
- **V08:** use original Q_b's positive q10–q90 and fully internal half-open
  1000-EUR bins. No bin with original positive-conditional mass ≥.005 may
  have zero candidate branch mass. Never substitute a different bin size.

Inherited integerization TV ≤3×10^-7, maximum entry error ≤1.3×10^-7 and
Income mean change ≤1 EUR remain HARD. Required quantiles must be finite and
nondecreasing; maps total, integers checked, and all applicable capacities met.
Missing production evidence remains OPEN ISSUE; this offline prototype is not
a D2-B2 success package or proof of runtime obligations.

## 8. Ordering proof and fixed-design stopping rule

For sorted exact scalars, contiguous nonempty buckets and reachable lower
medians imply `m(t1)<m(t2) ⇒ representative(t1)≤representative(t2)`.
For each positive cached base value, multiplication by a higher positive
representative and whole-EUR ceiling are monotone. Thus every corresponding
authored positive point is non-lower. Exact merging cannot change the authored
distribution. Final integer laws must still pass V05 mechanically.

Changing one inherited factor toward a larger value, with other parents fixed,
cannot decrease the representative. Coarsening may tie two tuples. Mechanical
evidence must also show visible strict increases for inherited factor directions;
no conditional percentage-fit requirement is added.

If this exact fixed design fails any retained HARD rule, stop the design.
Record gate, branch/law, exact result, required rule and smallest likely cause.
Do not change support/bucket counts, tune curves/factors, optimize, cluster,
compress adaptively, clip or weaken visible-plausibility rules. Any later design
decision is separate. The current prototype fails S02 and V07; no alternate
design was attempted and no production run is authorized by this document.

## 9. Prospective D2-A resources

The prototype has 60 authored/canonical laws, 23040 positive points and 60
zero points; no whole-EUR duplicate survives the initial generation because
none occurs. All laws have 385 distinct points and meet both support
reservations directly. Maps have 76160 slots, 64960 reachable assignments and
11200 employee sentinel slots. There is no global support allocator.

Using unchanged D2-A §21 formulas, Income numerical input is 185664 bytes,
Income retained charge 371904 bytes. The compact whole-model numeric encoding
estimate is 685277 bytes, computed in memory; it is not a published D2-B2
encoding. Bounded primitive decoded/work estimate is 1629218 bytes. Preserve
inherited E≤6.5 MiB/D≤6 MiB reservations, the 1 MiB outside bookkeeping
allowance and Employer H2≤1 MiB allowance. The report itemizes simultaneous
live phases: R_out=1878264, outside combined=14985464, R2=43255546,
joint retained=45133810 and constructor peak bound=72095482 bytes.
Every frozen capacity passes prospectively; implementation must still honor
actual array/descriptor widths, scratch lifetimes and preallocation checks.
Python fitting/report heaps are offline machinery, not a runtime RSS claim.

## 10. CountryPack compatibility

Conceptually, the fast Italian country asset is:

```text
IncomePack
├── branch base curves
├── inherited parent factors
├── scale buckets
├── finite laws
└── tuple→law maps
```

Only the Italian pack is defined. ATECO, twenty Italian Regions and nominal
EUR are current IT asset/Contract-5 properties, not universal assumptions for
future packs. No multi-country runtime, currency abstraction, PopulationMix,
multinational workforce mechanism or new public configuration is implemented.

## 11. Delivery and review scope

Only this specification and its supporting root report are added. Pre-existing
files, Contracts/D2-A, v0/v1, historical evidence, semantic.go and Git index
remain unchanged. Nothing is staged or committed. One fixed temporary prototype
was built under /tmp; prototype assets are removed after evidence capture.
No logical-assets.json, D2-B2 manifest, canonical package or package checksums
are emitted. Existing-package tests are sanity evidence only.

Focused independent review should first assess the exact S02/V07 witnesses
and the preservation of frozen canonical ID order. **The fixed generator is
not accepted for production: retained HARD failures remain unresolved.**
