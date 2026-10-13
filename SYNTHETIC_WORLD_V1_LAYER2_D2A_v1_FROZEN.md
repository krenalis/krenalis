# Synthetic World v1 — Layer 2 D2-A v0

## Numerical Model Architecture

### 1. Status / frozen baseline

**Status:** proposed numerical-model architecture v0 candidate awaiting independent re-review. Independent GPT-6 Astra xhigh review reported exactly two HIGH findings: restrictive workforce-cap ceiling (F1) and insufficient per-law Income support capacity (F2). This candidate rectifies those findings; it is not frozen and no Astra approval of the revised bytes is claimed. “Frozen now” below means resolved by this proposal, not independently approved. Layer 2 is not implemented.

**Requested model/configuration:** GPT-5.6 Sol xhigh. This records the requested configuration, not an independent attestation of the runtime model.

Verified repository baseline:

```text
Branch:  property-semantics
HEAD:    76fa0fff109ca6abad62f55c8761275455842dc5
Subject: Add Synthetic World Layer 1
```

The following artifacts were read completely and SHA-256 verified. Their exact filenames and hashes determine authority, including where historical proposal wording remains inside them. Contract-5 drafts are not authority.

| Frozen contract | SHA-256 |
| --- | --- |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md` | `bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md` | `e9af32a0bb9b48bf8972a408a1369901a3f3a9fe5ecd76d5762a367c055bc098` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md` | `df3ef0b52f37b587b8449cc71085dc9e5aca00f754bbd93e123c6fd5e3a4cb55` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md` | `a5ecfb735270ffba3e6afea23d85e548dd6dc3f07a8e9dfc5294a84ad3576418` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md` | `8a75054f793ef47e7071e3eb9fe04c0b32abfdac9223da859f95a333a1526e67` |

All repository citations below are portable repository-relative `file:line` references. The root `AGENTS.md` was inspected. Pre-change code is the implementation precedent; the unrelated `tools/types/semantic.go` modification is not precedent (`AGENTS.md:11`).

The initial delivery was authorized as one new specification file in the repository root. The present surgical rectification authorizes edits only to this existing v0 candidate; the user's subsequent instruction additionally authorizes one new root-level rectification report containing the response, not another D2-A specification. No implementation, frozen-contract or unrelated-file edits, staging, commit, generators, web or external research are authorized. Analysis and verification artifacts use `/tmp` only.

### 2. Scope and ownership boundary

**NORMATIVE:** D2-A freezes one architecture for precisely the fifteen model templates in section 17. Six Income templates realize one selected public Income node. Company.SizeBand and Employee.CurrentIndustry remain deterministic derivations. No further socioeconomic stochastic node exists.

| Owner | Exclusive responsibility |
| --- | --- |
| D2-A | Representation families; logical schemas; category and dimension order; canonical weight semantics; parent addressing and pooling; bucket/law cardinality maxima; structural support; Employer factorization and indexes; Income architecture; asset decomposition; constructor compilation and validation; finite input, retained, peak and work bounds; D2-B obligations |
| D2-B | Actual probabilities and categorical weights; count and Income supports/weights; actual per-law Income cardinalities and aggregate capacity allocation; zero masses; Employer I/S magnitudes and fitted workforce saturation cap C within its safe envelope; empirical/model caps; CompanyCount; permitted age boundaries and Income tuple-to-law assignments; sources, crosswalk contents, provenance, fitting, transformations, checksums, canonical fitted artifacts, concrete byte usage and Plausible+ evidence |
| Deterministic Protocol | Stream paths, Component names, framing, seed construction, fingerprints, exact sampling call sequence, bounded-integer realization, CDF/search realization outside the already-frozen Employer selector, world-ID encoding and canonical binary identity serialization |

D2-A logical law IDs, canonical dimension ordinals and indexing requirements are numerical addressing semantics, not Component names or identity bytes. Protocol chooses their concrete realization subject to bounded lookup and canonical meaning. D2-B cannot choose another distribution family, a fallback hierarchy, additional parent dimensions, factorization, bucket family or index architecture.

The sole public constructor remains `NewSocioeconomicWorld(base *World)` with prescribed private model data, not runtime model arguments (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:195`).

### 3. Design principles

1. Preserve Contracts 1–5, exact parent sets, directions, gates, outcome meanings and exclusions.
2. Use finite, exact integer laws. Never allocate in proportion to workforce, income amount or weight sum.
3. Retain broad conditional correlations and allow uncommon valid profiles through numerical support rather than invented semantic restrictions.
4. Use small complete categorical tables; use bounded conditional-law sharing for the substantially larger Income grids.
5. Compile sparse source evidence into complete fitted assets offline in D2-B. Constructor compilation validates numerical assets; generation performs no fitting, inference or empirical fallback.
6. Distinguish semantic validity from positive model probability. A zero weight is model data, never an invalid taxonomy value or missing observation.
7. Give each model its own logical identity and innovation. Physical asset sharing must not couple unrelated nodes.
8. Keep only components required for frozen relationships, Plausible+, sourceability, determinism or resource safety.

The frozen foundation provides checked integer weighted sampling and unbiased bounded draws (`tools/fakedata/random.go:48`, `tools/fakedata/random.go:77`). Fresh entity streams are independently derivable without shared advancing RNG (`tools/fakedata/stream.go:36`). These establish feasibility; they do not authorize new socioeconomic information.

### 4. Common numerical primitives

**Decision:** unsigned integer weight vectors are the sole default primitive for small categorical laws and finite numerical support laws.

| Candidate | Assessment |
| --- | --- |
| Integer weights | Selected: exact zero, exact arithmetic, compact fixed widths, GCD equivalence and bounded-integer compatibility |
| Rational probabilities | Rejected as runtime assets: extra denominators and common-denominator growth add no benefit over canonical integer ratios |
| Fixed-point probabilities | Rejected: a prescribed scale and sum-to-scale requirement introduce unnecessary quantization and identity conventions |
| Floating-point probabilities | Rejected: avoid rounding, non-finite values and platform-sensitive numerical behavior |

A weight means a relative probability proportional to that exact integer. Authored weights are exact `uint32` values in `0..4,294,967,295`. Negative, fractional, non-finite or out-of-width values reject before narrowing. Empty laws and all-zero vectors reject. Sum weights using checked `uint64` addition before any sample. No wrapping, saturation or silent repair.

Small categorical laws are dense vectors with every frozen output category, including zero-weight categories. A row contains exactly the declared vector length. Its total is positive. GCD canonicalization in section 5 precedes identity and runtime sampling.

Finite numerical laws contain authored `(uint32 support, uint32 weight)` pairs. Both EmployeeCount and Income use this same support-point primitive. No intervals, interpolation, sampled segment selector, parametric runtime fitting or separate zero-mass draw is used.

**Numerical primitives and private widths:**

| Primitive | Representation | Width | Canonicalization | Invalid cases |
| --- | --- | --- | --- | --- |
| Small categorical weight | Dense integer vector in frozen category order | Authored/canonical `uint32`; total `uint64` | Per-row GCD division; zeros preserved | Wrong dimension, empty/all zero, noninteger/out of range, total overflow |
| Count support law | Finite support/weight pairs, at most 128 authored pairs and 128 canonical points | Support/weight `uint32`; merge/total `uint64` | Sort, merge equal supports, omit zero-mass points, GCD, checked narrowing | Invalid support/weight, excessive authored or canonical count, zero total, overflow, canonical merged weight above uint32 |
| Income support law | Same finite-point primitive, final whole-EUR supports only; ≤512 authored pairs and ≤512 canonical positive-mass points per law, subject to aggregate byte preflight | Support/weight `uint32`; merge/total `uint64` | Same exact point canonicalization as count law | Invalid support/weight, excessive per-law or aggregate resources, zero total, overflow, unrepresentable canonical merged weight, incompatible monetary conversion before asset production |
| Employer factor | Exact positive integer; workforce zero handled separately | `uint16`, permitted positive range `1..4,095` | Canonical table order only; no scale normalization | Zero positive factor, out of range, noninteger |
| Workforce cap | One D2-B fitted saturation parameter for the unchanged deterministic transform | `uint16`, `1..65,535` | Exact value | Zero/out of range/noninteger; reject before narrowing |
| Employer final weight | Frozen exact interval length | `uint64` | No normalization/rescaling | Ineligible positive weight, multiplication or total overflow |
| Age boundary / bucket ID | Sorted lower-bound ages / consecutive ordinal | `uint8` | Increasing lower bounds; IDs by interval order | Incomplete/overlapping/noncontiguous intervals, invalid age or ID |
| Region/category ordinal | Exact canonical dimension position | `uint8` | Fixed semantic-key order | Unknown/alias/duplicate category |
| Private pool/law ID | Branch-local canonical Income law ordinal | `uint16` | First occurrence in canonical tuple traversal after law canonicalization | Missing/out-of-range reference; unused law; invalid sentinel in reachable cell |
| Per-law support-point cardinality | Explicit bounded cardinality for one count or Income law | `uint16` | Exact count | Zero or above 128 for EmployeeCount / 512 for Income where a law is required |
| CompanyCount | Prescribed literal | `uint64` | Exact value | Zero or above D2-A maximum |

No independent per-axis nominal pool IDs exist. Canonical Income law IDs are also its private tuple-pool IDs. Optional sampling acceleration must fit the resource reservations in section 21; its exact CDF/search mechanics remain Protocol-owned.

### 5. Canonical category ordering and weight normalization

**Category order:** D2-A adopts the displayed taxonomy order of frozen Contract 3 for numerical vectors and model dimensions. For nominal taxonomies this is an addressing order, not a socioeconomic rank. Contract 3 left sampling order deferred (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:131`); D2-A now fixes numerical category order, while Protocol retains exact sampling realization.

| Domain | Canonical order, left to right |
| --- | --- |
| EducationLevel | `below-primary`, `primary`, `lower-secondary`, `upper-secondary`, `post-secondary-non-tertiary`, `tertiary`, `doctoral` |
| Industry | `agriculture`, `production-and-utilities`, `construction`, `commerce`, `transport-and-logistics`, `hospitality`, `information-and-communication`, `finance-and-real-estate`, `professional-and-business-services`, `public-administration-and-safety`, `education`, `health-and-social-care`, `arts-and-recreation`, `other-services` |
| JobFamily | `production-and-fieldwork`, `skilled-trades`, `transport-and-logistics`, `customer-and-personal-service`, `administration`, `sales-and-commercial`, `technical-and-scientific`, `professional-and-creative`, `care-and-education`, `protective-services` |
| JobLevel | `individual-contributor`, `supervisor`, `manager`, `executive` |
| PrimaryActivity | `employee`, `self-employed`, `unemployed`, `student`, `retired`, `inactive` |
| CompanySizeBand | `zero`, `micro`, `small`, `medium`, `large` |
| Region | `Abruzzo`, `Basilicata`, `Calabria`, `Campania`, `Emilia-Romagna`, `Friuli-Venezia Giulia`, `Lazio`, `Liguria`, `Lombardia`, `Marche`, `Molise`, `Piemonte`, `Puglia`, `Sardegna`, `Sicilia`, `Toscana`, `Trentino-Alto Adige/Südtirol`, `Umbria`, `Valle d'Aosta/Vallée d'Aoste`, `Veneto` |

Taxonomy evidence: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:137`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:193`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:334`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:408`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:563`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:260`.

The Region list is the verified distinct-label set of the frozen places asset, put in the explicit order above. Region means the selected municipality row's exact stored label; only that projection is a model input (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:139`; `tools/fakedata/layer1_data.go:396`). No locale collation, Unicode normalization, spelling alias, source row order or map iteration determines numerical order.

Parent dimensions traverse lexicographically in the dimension order declared for each model, first dimension outermost and last dimension innermost. Ages and age buckets traverse ascending. Numerical supports traverse ascending numeric value. Employer entries remain ascending CompanyIndex under Contract 2 (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:336`).

**Proportional categorical/support-law equivalence:** for a valid vector, compute the exact GCD of its nonzero weights and divide every weight by it, leaving zeros unchanged. No floating point is used. `[1,2,3]` and `[2,4,6]` become the identical canonical vector. All-zero input has no valid normalization and rejects. Canonicalization happens before both identity and sampling, so authored rescaling alone cannot change node realization or output-sensitive identity. These rules apply to new Layer-2 models only; frozen Layer-0/1 tables and outputs remain unchanged.

For numerical supports, constructor compilation first sorts complete support/weight pairs, merges equal supports by checked uint64 addition, removes zero-mass groups and checks positive total. It then divides merged weights by their GCD and verifies each result fits uint32. Reject an unrepresentable canonical merged weight; do not round it. The authored-pair cap applies before merging. Equal positive supports appear once. Zero is retained exactly when its mass is positive. Omitting a zero-mass support does not make that value semantically invalid. Different duplicate decompositions yielding the same canonical law are equivalent.

Canonical Income laws with identical support/weight content within one branch are merged. Before merging, enforce the branch's authored-definition and authored-pair caps in section 16; deduplication cannot make oversized input admissible. Traverse that branch's complete canonical parent map and assign consecutive law IDs by first occurrence of each distinct canonical law. Remap references accordingly. Within those input caps, authored law labels, permutations and duplicate identical law definitions have no model meaning. Require all authored laws to be referenced; canonical laws are all referenced. No merging across branches affects identity.

**Employer exception:** proportional final Employer weights are not canonical-equivalent. Contract 2 freezes exact interval lengths and forbids rescaling/normalization (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:323`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:494`). Common scaling changes total and exact bounded-draw mapping, so probability equality does not establish deterministic-output equivalence. Factor magnitudes are exact model data; neither factors nor constructed Employer weights undergo GCD division. The categorical equivalence rule does not alter this frozen protocol.

### 6. Parent exact/bucket/pool architecture

**Definitions:** EXACT preserves each semantic value as a distinct address coordinate; two addressed rows may contain equal laws. BUCKETED maps an ordered parent to one contiguous age interval. POOLED denotes the final bounded sharing of Income laws across full authorized parent tuples. Individual nominal Income dimensions remain EXACT when forming those tuples. Joint law sharing is not an additional per-axis pool mapping and does not add or remove a semantic parent.

All Region dimensions are EXACT over the 20 labels. All Education, Industry, JobFamily, JobLevel and applicable SizeBand dimensions are EXACT. There is no Region macro-pool, no education/family/industry coarsening table, and no mapping inferred from municipality population. Exact Region is sufficiently small for the categorical tables and only contributes compact addresses in Income maps.

AnchorAge uses eight deliberately model-local mappings:

| Mapping | Consumer | Maximum buckets | Maximum law-address rows affected |
| --- | --- | ---: | ---: |
| Education age | Education | 6 | 120 |
| Activity age | PrimaryActivity | 8 | 1,120 |
| Employee level age | Employee.JobLevel | 6 | 420 |
| Self-employed Income age | Income[s] | 6 | 16,800 |
| Unemployed Income age | Income[u] | 6 | 840 |
| Student Income age | Income[t] | 6 | 840 |
| Retired Income age | Income[r] | 6 | 840 |
| Inactive Income age | Income[i] | 6 | 840 |

D2-B supplies actual bucket counts in `1..maximum` and integer lower bounds. The first lower bound is 21; subsequent bounds strictly increase and are at most 75. An interval includes its lower bound and ends immediately before the next lower bound; the final interval ends at 75. Thus the domain 21–75 is covered once, contiguously, without overlap or gaps. A boundary age belongs to the interval starting there. Constructor compilation produces a 55-entry age-to-bucket address array for each consumer. No eager context or external acquisition is involved; the range is frozen (`tools/fakedata/layer1_data.go:135`, `tools/fakedata/layer1_data.go:146`).

No output-sensitive age mapping is shared between nodes. Equal mappings may be physically deduplicated, but identities remain consumer-local. This permits independent fitting and branch isolation without eight gratuitously different required threshold sets: equal thresholds are allowed, not mandated. Six buckets suffice for broad attainment, responsibility and Income lifecycle effects; activity receives eight because transitions between the six primary states need somewhat finer age resolution. These are complexity ceilings, not age thresholds or empirical fit claims.

Changing a consumer's boundaries or effective tuple-to-law assignment is output-sensitive for that consumer. Reordering equivalent source records or relabeling canonical-equivalent law IDs is not. No additional bucket family is permitted.

**Authoritative parent representation table:** the class describes parent address encoding before joint Income law sharing. Every frozen direct parent, including Employer relational information, appears below.

| Node | Parent | EXACT / BUCKETED / POOLED | D2-A architecture | D2-B freedom |
| --- | --- | --- | --- | --- |
| Company.Industry | None | — | Root vector | Weights only |
| Company.EmployeeCount | Same Company.Industry | EXACT | 14 distinct law addresses | Supports/weights |
| Education | AnchorAge | BUCKETED | Own contiguous map, at most 6 buckets | Count/boundaries |
| Education | Region | EXACT | 20-value axis | Row weights |
| PrimaryActivity | AnchorAge | BUCKETED | Own map, at most 8 buckets | Count/boundaries |
| PrimaryActivity | Education | EXACT | Seven-value axis | Row weights |
| PrimaryActivity | Region | EXACT | 20-value axis | Row weights |
| Employee.JobFamily | Education | EXACT | Seven-value axis | Row weights |
| Employee.JobFamily | Region | EXACT | 20-value axis | Row weights |
| Employee.JobLevel | AnchorAge | BUCKETED | Own map, at most 6 buckets | Count/boundaries |
| Employee.JobLevel | Education | EXACT | Seven-value axis | Row weights |
| Employee.JobLevel | Employee.JobFamily | EXACT | Ten-value axis | Row weights |
| SelfEmployed.JobFamily | Education | EXACT | Seven-value axis; separate law | Own row weights |
| SelfEmployed.JobFamily | Region | EXACT | 20-value axis; separate law | Own row weights |
| SelfEmployed.Industry | SelfEmployed.JobFamily | EXACT | Ten distinct law addresses | Industry weights |
| Employer | Employee.JobFamily | EXACT | Ten selector-condition coordinates; Industry factor row | Factor magnitudes |
| Employer | Employee.JobLevel | EXACT | Four selector-condition coordinates; Size factor row | Factor magnitudes |
| Employer | OperationalCompanyUniverse | EXACT | Fixed dense `1..N`; 40 complete raw inputs | Literal N within cap |
| Employer | Company.EmployeeCount[*] | EXACT | Zero exclusion; deterministic `min(count,C)` for positives | Workforce cap C |
| Employer | Company.Industry[*] | EXACT | 14-column compatibility factor | Factor magnitudes |
| Employer | Company.SizeBand[*] | EXACT | Four positive-band factor columns; zero handled by exclusion | Factor magnitudes |
| Income[e] | Region | EXACT | Full tuple coordinate before law pooling | Tuple-to-law assignment |
| Income[e] | Employee.JobFamily | EXACT | Full tuple coordinate before law pooling | Tuple-to-law assignment |
| Income[e] | Employee.JobLevel | EXACT | Full tuple coordinate before law pooling | Tuple-to-law assignment |
| Income[e] | Employee.CurrentIndustry | EXACT | Sole 14-value sector coordinate | Tuple-to-law assignment |
| Income[e] | Company.SizeBand[Employer] | EXACT | Full five-band axis; zero slice unreachable | Assignments in four positive bands |
| Income[e] | Employer | EXACT | Relational access key only; never a numeric law coordinate or innovation input | None |
| Income[s] | AnchorAge | BUCKETED | Own map, at most 6 buckets | Count/boundaries |
| Income[s] | Region | EXACT | Full tuple coordinate | Tuple-to-law assignment |
| Income[s] | SelfEmployed.JobFamily | EXACT | Full tuple coordinate | Tuple-to-law assignment |
| Income[s] | SelfEmployed.Industry | EXACT | Full tuple coordinate | Tuple-to-law assignment |
| Income[u] | AnchorAge | BUCKETED | Own map, at most 6 buckets | Count/boundaries |
| Income[u] | Education | EXACT | Seven-value tuple coordinate | Tuple-to-law assignment |
| Income[u] | Region | EXACT | 20-value tuple coordinate | Tuple-to-law assignment |
| Income[t] | AnchorAge | BUCKETED | Own map, at most 6 buckets | Count/boundaries |
| Income[t] | Education | EXACT | Seven-value tuple coordinate | Tuple-to-law assignment |
| Income[t] | Region | EXACT | 20-value tuple coordinate | Tuple-to-law assignment |
| Income[r] | AnchorAge | BUCKETED | Own map, at most 6 buckets | Count/boundaries |
| Income[r] | Education | EXACT | Seven-value tuple coordinate | Tuple-to-law assignment |
| Income[r] | Region | EXACT | 20-value tuple coordinate | Tuple-to-law assignment |
| Income[i] | AnchorAge | BUCKETED | Own map, at most 6 buckets | Count/boundaries |
| Income[i] | Education | EXACT | Seven-value tuple coordinate | Tuple-to-law assignment |
| Income[i] | Region | EXACT | 20-value tuple coordinate | Tuple-to-law assignment |

The final Income full-tuple-to-law operation is POOLED: at most 512 employee, 256 self-employed and 64 laws in each other branch. Pool membership can depend jointly on only the listed value parents. It cannot depend on the Employer key, raw employee count, an ancestor's identity or source metadata. Keeping exact address coordinates is compatible with equal-law pooling; EXACT does not require all addressed probabilities to differ. There is no runtime pool search or hierarchy.

### 7. Company Industry

**Chosen law:** one dense 14-entry canonical uint32 weight vector; no semantic parent. Validate dimension, integer range, positive total and GCD canonical form. Actual weights are D2-B data. The canonical vector and Industry semantics are separable Company.Industry identity material.

CompanyIndex provides entity addressing only. It is not a vector dimension, ordinal economic effect, trend, modulo class or tie-break preference. No geography, Person, count, SizeBand, CompanyCount or another Company enters the law. CompanyCount only establishes operational membership. Industry generation precedes EmployeeCount (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:184`).

### 8. Company EmployeeCount

**Chosen law:** one finite explicit uint32 support-point distribution per exact Industry, at most 128 authored pairs and 128 canonical positive-mass points per Industry.

| Candidate | Decision and reason |
| --- | --- |
| Explicit support points + integer weights | Selected: finite, directly uint32, sourceable from aggregate size profiles and capable of independently weighted right-tail points |
| Piecewise integer ranges + weights | Rejected: within-range laws and endpoint arithmetic add an extra modeling mechanism without a required v1 benefit |
| Empirical quantile/breakpoint law | Rejected as runtime architecture: quantile interpolation and discretization rules would add machinery; D2-B may use quantiles offline to choose final points |
| Fixed-point parametric law | Rejected: special-function, tail and rounding rules add fitting and deterministic burden |
| Mixture over count-support segments | Rejected: a sampled segment is unnecessary when one point law already represents all five bands and skew |

The 14 laws are complete even when the root assigns an Industry zero weight. Every source support is an exact whole count in `0..MaxUint32`; zero is valid. Sort and merge as section 5 prescribes. No dense uint32-domain table is permitted.

D2-B can place support at zero, inside `1..9`, `10..49`, `50..249`, and anywhere from `250..MaxUint32`, with independently chosen masses. Up to 128 points leave ample room for a dense low-count body and increasingly spaced tail. D2-A requires this capability and future intentional verification of all bands; it does not force every Industry to have positive mass in every band or choose a count cap. D2-B must explain band omissions and provide overall Plausible+ size/tail evidence.

Only realized EmployeeCount leaves the law. A private sampled support index is discarded and cannot feed another node. No SizeBand is sampled, no segment is retained, and no downstream model receives support-point position. SizeBand derives afterward under the exact frozen intervals (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:280`). Its zero/micro/small/medium/large thresholds are not D2-B parameters.

The Industry-conditioned canonical support laws are the EmployeeCount model identity. Changing them cannot perturb Company Industry. Actual finite tail truncation and the largest production support are D2-B-owned, explicitly documented bounded modeling choices, never clipping of an invalid runtime draw.

### 9. Education

**Chosen shape:** `EducationAgeBucket × Region -> 7-entry categorical vector`.

Dimension order is age bucket then Region. With `B_E <= 6`, there are exactly `20 B_E` rows, at most 120, and `140 B_E` weights, at most 840. All rows are required, including zero-probability source regions/ages; constructor compilation checks complete uniqueness and canonical dimensions.

Exact Region is simpler than another mapping asset and retains attainment differences within broad age cohorts. Broad aggregate evidence plus documented assumptions can fill 120 rows without a highly sparse occupation cross-product. D2-B may author equal vectors in multiple rows, but cannot introduce pooling rules, runtime national fallback, enrollment, Gender or reverse activity conditioning. Output remains highest completed attainment.

### 10. PrimaryActivity

**Chosen shape:** `ActivityAgeBucket × Education × Region -> 6-entry categorical vector`.

With `B_A <= 8`, require exactly `140 B_A` rows, at most 1,120, and `840 B_A` weights, at most 6,720. Dimension order is age bucket, Education, Region. All cells resolve before construction succeeds.

A larger activity age ceiling preserves broad study, working and retirement transitions without exact 55-age fitting. Seven attainment groups and 20 Regions are compact enough to retain obvious segment correlations. Sparse source cross-products are a D2-B modeling problem: offline smoothing or assumption-based fitting must produce all rows. No runtime backoff is introduced.

There is exactly one categorical outcome over six activities. No overlapping activity flags, workforce-status latent or precedence stage is sampled. D2-B reconciles overlapping source populations offline within this six-state law (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:225`).

### 11. Employee JobFamily

**Chosen shape:** `Education × Region -> 10-entry categorical vector`, exactly 140 rows and 1,400 weights.

Every canonical parent cell is required. No direct AnchorAge, Employer, Company Industry, EmployeeCount or SizeBand dimension enters. Education and Region preserve occupational composition; employment branch selection can provide mediated age association. D2-B supplies branch-specific weights and source-to-function mappings. Zero weights express numerical support only.

### 12. Employee JobLevel

**Chosen shape:** `EmployeeLevelAgeBucket × Education × Employee.JobFamily -> 4-entry categorical vector`.

With `B_L <= 6`, require exactly `70 B_L` rows, at most 420, and `280 B_L` weights, at most 1,680. Dimension order is age bucket, Education, JobFamily. No Region or Company input enters.

The law can retain older expert individual contributors, young supervisors and low-attainment executives without a hard responsibility gate. D2-B chooses magnitudes and documents zeroes that resemble exclusions. Responsibility remains delegated authority, not an age, salary, tenure or size category (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:417`).

### 13. Self-employed JobFamily

**Chosen shape:** its own `Education × Region -> 10-entry categorical vector`, exactly 140 rows and 1,400 weights.

The taxonomy and axis order match employee family for reviewability; its numerical contents and identity do not. Changing employee family data cannot change this branch's realization through a file checksum or shared row registry. No Company, Employer, JobLevel or direct age enters (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:352`).

### 14. Self-employed Industry

**Chosen shape:** `SelfEmployed.JobFamily -> 14-entry Industry vector`, exactly 10 rows and 140 weights.

Generate family first and then Industry. No Region, Education, Company model, Employer selector, owned-business entity or workforce surrogate is read directly. Equal taxonomy does not share Company Industry probabilities or identity. All family/Industry combinations remain semantically valid; numerical zeroes do not create a validity matrix.

### 15. Employer numerical architecture

#### Eligibility and support

**Chosen eligibility:** every operational Company with EmployeeCount greater than zero receives strictly positive weight in every one of the 40 `(JobFamily, JobLevel)` conditions. EmployeeCount zero always yields weight zero and is omitted from every retained index. There is no additional hard zero.

Thus every positive-workforce Company is structurally available to every employee condition. Unusual family/Industry pairs and executives at micro employers retain support. Plausibility comes from magnitudes. No family/Industry, size/level, qualification or age impossibility matrix is created. This is a model-specific strengthening of Contract 2's permissive positive-count eligibility, not a change to its generic builder (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:278`).

#### Exact factorization

**Chosen formula:** for count `n`, family `f`, level `l`, Industry `i` and its correctly derived band `b`:

```text
n == 0: W = 0
n > 0:  W = min(n, C) × I[f,i] × S[l,b]
```

`C` is a single global Employer workforce cap. `I` is a `10 × 14` Industry compatibility factor table. `S` is a `4 × 4` factor table over levels and the positive bands micro, small, medium, large. The zero-band column is not authored; zero-workforce exclusion determines zero weight before factor lookup.

I and S use uint16 storage and permitted values `1..4,095`; C uses uint16 storage and permitted values `1..65,535`. Their exact values are D2-B-owned. Reject malformed out-of-range source values before narrowing; cast/widen before multiplication and check each product and cumulative addition. Neither factor nor weight normalization is performed. These are exact model parameters, not categorical probability vectors.

The minimal candidate factorization is sufficient:

| Frozen effect | Representation |
| --- | --- |
| EmployeeCount prevalence | Count-sensitive `min(n,C)` |
| JobFamily × Industry | Full compatibility table I |
| JobLevel × SizeBand | Full positive-band preference table S |
| Unusual valid combinations | Every factor positive |
| No unauthorized Person information | Only f and l enter |
| No CompanyIndex economic effect | Weight depends only on n, i and derived b |

No family×size, level×Industry or three-way interaction table is added: no frozen relationship requires it. Joint family/level selection effects already arise by multiplication over the actual Company facts. A raw-count workforce factor could let an extreme uint32 Company dominate; count bucketing would add a mapping and discard within-band differences. The capped raw-count transform retains direct count sensitivity below C and gives D2-B one transparent calibration control. No piecewise slope, count-bin family or industry-specific cap is permitted.

**NORMATIVE:** C is a D2-B fitted Employer-prevalence saturation parameter. The D2-A upper bound 65,535 is only a safe architecture envelope: it is not a realistic-firm-size claim, public EmployeeCount cap, SizeBand threshold, required fitted value, or requirement that production workforce prevalence be proportional through 65,535. Counts above actual C remain valid and retain their actual public EmployeeCount. Count differences below C can change weights within one SizeBand; at/above C the workforce factor is saturated, so count differences need not change weights when other applicable facts are fixed. Neither case adds a residual employee Income count effect.

Arithmetic is safe structurally at the architecture maximum:

```text
0 <= W <= 65,535 × 4,095 × 4,095 = 1,098,958,053,375
CompanyCount <= 104,855
selector total <= 104,855 × 1,098,958,053,375
               = 115,231,246,686,635,625
               < 18,446,744,073,709,551,615 = MaxUint64
```

Valid prescribed parameter maxima therefore cannot overflow a final product or total. These products were independently recalculated with exact integer arithmetic in a /tmp verification script. Checked arithmetic is still required to reject malformed data, accounting bugs or an invalid implementation. No actual fitted parameter value is selected by this proof. Widening only C's accepted envelope changes no candidate membership, entry width, index count, routing or CompanyCount ceiling: every positive-count Company still has positive weight in all 40 indexes. Contract-2 entry counts and byte formulas below remain unchanged.

#### Routing and indexes

**Chosen architecture:** exactly 40 direct immutable Contract-2 indexes, one for each canonical JobFamily × JobLevel condition. No index sharing, routing pool, sampled Industry stage, alias index or per-Person candidate scan.

Canonical condition order is JobFamily outermost and JobLevel innermost. Routing is stateless direct Cartesian addressing to that condition's index, with no retained routing table. This fixes logical addressing and index count; it does not name Protocol Components or prescribe Go types or byte encoding.

For every index, raw input includes every CompanyIndex `1..N` exactly once, including zero-weight Companies. Use Contract-2 complete-universe validation. Retained membership is precisely the positive-workforce subset, common to all 40 indexes, with condition-specific weights. Canonical input and retained entry order are ascending CompanyIndex. Each index must have positive total, even if a condition currently has zero upstream production probability. This coverage choice is simpler and remains stable when D2-B changes upstream zero weights.

Construct indexes serially. Retain cumulative intervals only, not 40 raw-weight arrays. The selector is already frozen: one `Uint64n(total)` invocation and first cumulative value strictly greater than the draw (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:442`). D2-A does not replace that sequence with a newly designed Protocol.

If the realized universe contains no positive-workforce Company, construction fails. There is no forced-positive Company, retry, Person-driven repair, fallback employer or change to root/count laws. D2-B must quantify this construction risk across supported seeds and choose credible count support and CompanyCount; lack of empirical values is not an architecture blocker.

#### Exact CompanyCount resource ceiling

Let:

```text
N = CompanyCount
K = 40
q = number of positive-workforce Companies
P = total retained entries = 40q
A = aggregate raw input entries = 40N
M = maximum entries in one raw input = N
```

Contract 2 names aggregate raw entries M in its input formula; this document uses A for that quantity to preserve the requested meaning of M as the per-index maximum. They must not be confused.

Frozen charges are:

```text
retainedIndexBytes = 64 + 32K + 16P
selectionInputBytes = 64 + 32K + 16A
companyMetadataBytes = 64 + 4N
```

Both index and semantic-input budgets are 64 MiB. Since inputs include all N Companies in every condition, the exact maximum is:

```text
Nmax = floor((67,108,864 - 64 - 32×40) / (16×40))
     = 104,855
```

At Nmax with q=N:

```text
K = 40
P = A = 4,194,200
M = 104,855
index/input charge = 67,108,544 bytes
remaining index/input allowance = 320 bytes
```

One additional Company would require another 640 aggregate bytes and exceed the budget. A high fraction of zero-count Companies cannot relax Nmax because zeros still count in A. Contract 2's protocol domain and 1,048,576 generic technical maximum remain unchanged; D2-A chooses this narrower fixed-model resource envelope. CompanyCount is still an actual D2-B literal, not this maximum.

A universe of up to 104,855 Companies provides ample architectural room for a broad synthetic population, repeated employers and fourteen industries across four positive size regions. No required relationship demands one million operational Companies or one company per employee. A compact hierarchical selector would add output-sensitive mechanics contrary to the simplicity target. Forty direct indexes are retained as the single architecture; D2-B cannot switch to shared or hierarchical indexes.

### 16. Income numerical architecture

#### One chosen finite law

**Chosen architecture:** every authored Income law definition contains at most 512 support/weight pairs; every retained canonical law contains at most 512 positive-mass points. Each law is a finite whole-EUR support-point distribution using the integer/GCD primitive of sections 4–5. Zero is an ordinary support value in that same law. There is no separate zero flag or zero draw. Existing law-count maxima are unchanged; not all laws may simultaneously use 512 points. Aggregate authored input/decode and canonical retained acceptance is governed by the existing byte formulas and whole-layer preflight in section 21.

| Candidate | Assessment |
| --- | --- |
| Whole-EUR support points + integer weights | Selected: direct valid outputs, explicit zero, arbitrary skew and increasingly spaced weighted tail points |
| Empirical quantile table | Useful offline in D2-B; not a second runtime family |
| Piecewise quantile curve | Rejected: interpolation and rounding rules add runtime machinery |
| Fixed-point parametric distribution | Rejected: unnecessary fitting, numerical-function and bounded-tail mechanics |
| Finite sampled mixture | Rejected: no required relationship needs a source/affluence segment when one aggregate law suffices |

D2-B converts source-derived monetary modeling into final nominal anchor-basis whole-EUR supports before runtime asset delivery. No per-Person currency conversion, annualization or fractional-income draw occurs. Finer source precision is permitted offline, using Contract 5's exact `ceil(x)` conversion after checking the compatible exact rate against the public range (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:188`). Positive source rates cannot become zero. Directly authored whole-EUR supports need no additional conversion.

Every support lies in `0..MaxUint32`. Unique canonical points sort ascending; duplicate source points merge exactly under section 5, including points made equal by whole-EUR conversion. Model caps and largest tail points are D2-B empirical/model choices. No silent runtime clipping or resampling is allowed. A finite law can represent skew and a heavy-tail-shaped bounded distribution; it does not assert an unbounded mathematical tail.

Income retains the frozen common meaning: nonnegative gross personal monetary aggregate from eligible personal sources, annualized under current anchor conditions. It is not salary-only, turnover, signed net loss, household resources or wealth (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:69`). A zero outcome is no eligible personal current income, not missingness (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:339`).

#### Branch-local complete mapping and law pools

**Chosen addressing:** a complete immutable canonical authorized-parent tuple map points to one branch-local canonical law ID. D2-B supplies complete assignments and law contents. Multiple tuples may share a law; no empirical fallback hierarchy exists. The map is direct bounded lookup after constructor compilation.

| Branch | Canonical tuple dimension order | Maximum map slots | Required reachable assignments | Maximum authored law definitions | Maximum unique canonical laws |
| --- | --- | ---: | ---: | ---: | ---: |
| Income[e] | Region, JobFamily, JobLevel, CurrentIndustry, SizeBand | `20×10×4×14×5 = 56,000` | `20×10×4×14×4 = 44,800` | 512 | 512 |
| Income[s] | Own age bucket, Region, JobFamily, Industry | `6×20×10×14 = 16,800` | `2,800 B_s` | 256 | 256 |
| Income[u] | Own age bucket, Education, Region | `6×7×20 = 840` | `140 B_u` | 64 | 64 |
| Income[t] | Own age bucket, Education, Region | 840 | `140 B_t` | 64 | 64 |
| Income[r] | Own age bucket, Education, Region | 840 | `140 B_r` | 64 | 64 |
| Income[i] | Own age bucket, Education, Region | 840 | `140 B_i` | 64 | 64 |

Maximum aggregate address slots are 76,160. Before canonicalization, there are at most 1,024 authored law definitions across the six branches, subject to each branch's own unchanged definition cap; afterward there are at most 1,024 unique laws. Per-law 512-point limits give a structural upper bound of 524,288 authored pairs/canonical points, not permission to retain that simultaneous volume. The old unconditional aggregate 131,072-point limits are replaced by resource preflight: actual authored bytes must fit encoded/decode caps, and actual canonical law/point charges together with all other outside state must fit R_out≤8 MiB. No arbitrary second global point budget or cross-branch pool is introduced.

**NORMATIVE input limits:** authored definitions count separately even when their contents are identical; authored pairs count before duplicate-support merging or zero-mass omission. Validate declared and observed definition counts, per-law authored-pair caps and aggregate actual/projected encoded/decode charges before allocating storage that would exceed the relevant cap. Then independently validate canonical per-law cardinalities and complete retained/phase charges before excessive canonical allocation. Duplicate merging cannot rescue an authored encoding that exceeds its definition, per-law pair or input/decode byte limits. A larger authored volume that satisfies those limits may canonicalize to a smaller valid retained volume. For Income[e], 512 referenced identical definitions of 512 pairs each satisfy the structural input cardinalities and may canonicalize to one law, subject to all byte checks. A 513th definition rejects before deduplication. Equivalent encodings preserve canonical identity and realization only when each encoding satisfies the authored input limits. A 128-point law remains permitted; 512 is capacity, not a required D2-B cardinality.

**Non-empirical Plausible+ capacity witness, not production values:** one heavily used parent tuple can resolve to a single 512-point law containing one zero point; 400 body points at €100 increments from €100 to €40,000; 50 further points from €41,000 to €90,000 at €1,000 increments; 50 upper-tail points from €100,000 to €590,000 at €10,000 increments; and 11 high-tail points from €1,000,000 to €11,000,000. This is exactly 512 unique ascending uint32 whole-EUR values. Independent integer weights in the same vector can allocate zero mass, concentrate the body and thin the upper/high tail to express broad skew. These descriptive portions are not sampled segments, mixtures or retained states. The rejected 128-point ceiling cannot cover the 400-bin illustrative body even before zero/tails; 512 materially supplies that resolution and 112 further slots. Neither these monetary values, spacings nor any masses are prescribed for D2-B. Actual high-volume-law discretization evidence is required in section 23.

Employee's five-band address grid uses the frozen SizeBand order. The 11,200 zero-band cells have one prescribed private invalid-law sentinel, `uint16(65,535)`, and are not distributions. Constructor compilation fills and checks that slice; D2-B cannot assign a law there. This is the only architecture-level unreachable Income slice, because a selected employee employer necessarily has positive count. All four positive bands and all canonical combinations of the other values require assignments regardless of upstream zero weights. The sentinel is private addressing only, not a public missingness category or zero-income outcome.

Maps for age-conditioned branches use their actual bucket counts, not padded hypothetical buckets. All defined buckets × canonical categories resolve once. The maximum slot formulas are reservations, not requirements to use maximum buckets or maximum laws.

The pool mechanism addresses empirical sparsity without requiring a fitted five-way distribution for each employee tuple. D2-B can group tuples using broad aggregate effects and documented assumptions, preserving whichever Region/function/level/sector/size differences matter for Plausible+. It must provide the complete compiled map; no distinct probability is required for every parent value. No additional per-axis nominal mapping or additive income-coefficient architecture is permitted. Law IDs have no socioeconomic meaning and never leave generation.

Changes to effective branch assignments, supports, relative weights or age boundaries are branch-model identity-sensitive. Mere authored law-ID relabeling, source reordering, proportional weights or equivalent duplicate decompositions are canonical-equivalent.

#### Relational and hidden-state exclusions

Employee Income uses only Region, family, level, CurrentIndustry and selected Company.SizeBand. Employer is a key solely to acquire the selected sector/band values. It is not a map coordinate, Company-specific intercept, seed, tie-break or law-pool classifier. Raw EmployeeCount can be read only to derive SizeBand if necessary. Two employers with equal Industry and SizeBand are Income-equivalent under fixed other parents, own law and Person innovation (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:443`). Unselected Company information, selector weights, CompanyCount and Employer routing identities cannot reach Income.

Self-employed Income uses only its own age bucket, Region, family and Industry. Each non-working law uses only its own age bucket, completed Education and Region. No Company, JobLevel, prior occupation, pension history, loss state, household entity or benefit eligibility engine is introduced.

Exactly one selected branch law is evaluated. A sampled support index is an ephemeral realization detail, not retained Person state; it is never shared across nodes, exposed to Layer 3, or used as affluence, income-source, business-size or career class. No hidden source mixture or extra socioeconomic fact is generated. Constructor loading of all branches is not per-Person sampling of them (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:366`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:677`).

### 17. Complete model inventory

This is the authoritative inventory. Numerical law representations describe canonical runtime state; Income authored input limits are separately frozen in section 16. Logical Model IDs name specification units only; they are not Protocol Component names.

| Model ID | Output node | Frozen semantic parents | Parent runtime representation | Numerical law representation | D2-B contents |
| --- | --- | --- | --- | --- | --- |
| Company.Industry | Company.Industry | ∅ | Root | One 14-entry uint32 vector | Industry weights |
| Company.EmployeeCount | Company.EmployeeCount | Same Company.Industry | Exact Industry address | 14 finite count-point laws, each ≤128 points | Count supports/weights and model cap evidence |
| Education | Education | AnchorAge, Region | Own ≤6 age buckets × exact 20 Regions | ≤120 seven-entry vectors | Boundaries and weights |
| PrimaryActivity | PrimaryActivity | AnchorAge, Education, Region | Own ≤8 age buckets × exact Education × exact Region | ≤1,120 six-entry vectors | Boundaries, six-state weights and source-overlap treatment |
| Employee.JobFamily | Employee.JobFamily | Education, Region | Exact 7×20 grid | 140 ten-entry vectors | Employee family weights |
| Employee.JobLevel | Employee.JobLevel | AnchorAge, Education, Employee.JobFamily | Own ≤6 age buckets × exact 7×10 grid | ≤420 four-entry vectors | Boundaries and level weights |
| Employer | Employer | Employee.JobFamily, Employee.JobLevel, OperationalCompanyUniverse, Company.EmployeeCount[*], Company.Industry[*], Company.SizeBand[*] | Exact f/l direct condition; full dense Company candidates | `min(count,C) × I[f,industry] × S[level,band]`; 40 Contract-2 indexes | C, 140 Industry factors, 16 Size factors, CompanyCount, derived index evidence |
| SelfEmployed.JobFamily | SelfEmployed.JobFamily | Education, Region | Exact 7×20 grid; separate model | 140 ten-entry vectors | Self-employed family weights |
| SelfEmployed.Industry | SelfEmployed.Industry | SelfEmployed.JobFamily | Exact ten-value axis | Ten 14-entry vectors | Self-employed Industry weights |
| Income[e] | Income selected for employee | Region, Employee.JobFamily, Employee.JobLevel, Employee.CurrentIndustry, Company.SizeBand[Employer]; Employer key only | Exact value tuple → branch law pool; key excluded from tuple | ≤512 whole-EUR point laws, each ≤512 points; aggregate byte preflight | Complete assignments, supports/weights, actual cardinalities, zero masses, cap/resolution evidence |
| Income[s] | Income selected for self-employed | AnchorAge, Region, SelfEmployed.JobFamily, SelfEmployed.Industry | Own ≤6 buckets and exact nominal tuple → branch law pool | ≤256 whole-EUR point laws, each ≤512 points; aggregate byte preflight | Boundaries, assignments, supports/weights, actual cardinalities, zero masses, cap/resolution evidence |
| Income[u] | Income selected for unemployed | AnchorAge, Education, Region | Own ≤6 buckets and exact Education/Region tuple → branch law pool | ≤64 whole-EUR point laws, each ≤512 points; aggregate byte preflight | Own boundaries, assignments, supports/weights, actual cardinalities, zero masses and cap/resolution evidence |
| Income[t] | Income selected for student | AnchorAge, Education, Region | Own ≤6 buckets and exact Education/Region tuple → branch law pool | ≤64 whole-EUR point laws, each ≤512 points; aggregate byte preflight | Own boundaries, assignments, supports/weights, actual cardinalities, zero masses and cap/resolution evidence |
| Income[r] | Income selected for retired | AnchorAge, Education, Region | Own ≤6 buckets and exact Education/Region tuple → branch law pool | ≤64 whole-EUR point laws, each ≤512 points; aggregate byte preflight | Own boundaries, assignments, supports/weights, actual cardinalities, zero masses and cap/resolution evidence |
| Income[i] | Income selected for inactive | AnchorAge, Education, Region | Own ≤6 buckets and exact Education/Region tuple → branch law pool | ≤64 whole-EUR point laws, each ≤512 points; aggregate byte preflight | Own boundaries, assignments, supports/weights, actual cardinalities, zero masses and cap/resolution evidence |

Frozen parent evidence: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:452` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:527`. Gates are not extra table dimensions. Company.ID, Company.SizeBand and Employee.CurrentIndustry are not additional stochastic models.

### 18. Logical asset inventory

All assets are private logical units independent of eventual files, Go declarations or serialization. Numerical contents are output-sensitive after canonicalization, except implementation-only metadata and source-label aliases that disappear during compilation. D2-B supplies provenance and concrete checksums for each unit; Protocol owns fingerprint bytes. Income authored-definition/pair limits apply before canonicalization; unique-law/point limits apply afterward, as stated in section 16.

| Asset | Consumers | Logical schema | Max cardinality | Validation | D2-B contents |
| --- | --- | --- | --- | --- | --- |
| Frozen canonical dimensions | All relevant models | Exact ordered taxonomy/Region keys | Education 7; Industry 14; family 10; level 4; activity 6; band 5; Region 20 | Exact sets/order, unique within domain, exact Region projection | Version compatibility and source crosswalks; no freedom to change keys/order |
| Company-universe descriptor | Company membership; Employer | One prescribed positive uint64 CompanyCount and model revision | One count; N≤104,855 | Literal, bounds, identity coverage; no seed-derived count | Actual N and evidence |
| Company Industry law | Company.Industry | Dense ordered Industry weight vector | 14 weights | uint32, dimension, positive total, GCD | Actual weights |
| EmployeeCount laws | Company.EmployeeCount | Exact Industry → ordered count/support law | 14 laws; ≤1,792 authored/canonical points | Complete Industry rows, support range, point cap, canonical merging/GCD | Supports, weights, any model cap and fit evidence |
| Education age mapping | Education only | Count and increasing age lower bounds | ≤6 intervals; compiled 55 addresses | Full contiguous 21–75 coverage, unique membership | Actual count/boundaries |
| Education vectors | Education only | Education age bucket × Region → vector | 120 rows / 840 weights | Complete unique grid, dimensions, weight validation | Actual seven-state weights |
| Activity age mapping | PrimaryActivity only | Same age schema | ≤8 intervals; compiled 55 addresses | Same age validation | Actual count/boundaries |
| Activity vectors | PrimaryActivity only | Activity bucket × Education × Region → vector | 1,120 rows / 6,720 weights | Complete unique grid, six states, weight validation | Weights and offline overlap mapping |
| Employee family vectors | Employee.JobFamily only | Education × Region → vector | 140 rows / 1,400 weights | Complete grid, ten states, weight validation | Own weights/crosswalks |
| Employee level age mapping | Employee.JobLevel only | Same age schema | ≤6 intervals; compiled 55 addresses | Same age validation | Actual count/boundaries |
| Employee level vectors | Employee.JobLevel only | Level bucket × Education × family → vector | 420 rows / 1,680 weights | Complete grid, four states, weight validation | Own responsibility weights |
| Self-employed family vectors | SelfEmployed.JobFamily only | Education × Region → vector | 140 rows / 1,400 weights | Complete grid, ten states, separate identity | Own weights/crosswalks |
| Self-employed Industry vectors | SelfEmployed.Industry only | Family → Industry vector | 10 rows / 140 weights | Complete family axis, fourteen states | Own weights/crosswalks |
| Employer factor model | Employer only | C; exact family×Industry factors I; level×positive-band factors S | 1+140+16 uint16 values | I/S positive 1..4,095; C positive 1..65,535; canonical order; checked products/totals | Actual I/S magnitudes and fitted C |
| Employer runtime indexes | Employer only | Exactly 40 complete raw inputs compiled into Contract-2 cumulative indexes | Raw A=40N; positive P=40q; per input M=N | Full universe, unique sorted keys, zero-count exclusion, all positive totals, resource preflight | Derived index manifests/totals and model evidence, not arbitrary authored candidate weights |
| Income[e] assignments | Income[e] only | Exact authorized value tuple → canonical law | 56,000 slots; 44,800 valid references; zero-band sentinel slice | Complete unique reachable assignments; zero slice invalid; branch-local references | Actual assignments; no key/count/universe dimension |
| Income[e] laws | Income[e] only | Whole-EUR support/weight laws | ≤512 authored definitions / unique canonical laws; ≤512 pairs/points per law; branch structural volume ≤262,144, subject to aggregate byte caps | Pre-merge authored and aggregate-byte caps; uint32 supports/weights; canonical point laws; referenced laws; within-branch law merging; retained preflight | Actual per-law cardinalities, supports, weights, zero mass, tails/caps and resolution evidence |
| Income[s] age mapping | Income[s] only | Same age schema | ≤6 intervals; compiled 55 addresses | Same age validation; branch identity | Actual count/boundaries |
| Income[s] assignments | Income[s] only | Bucket × Region × family × Industry → law | 16,800 slots | Complete unique grid, own law references | Actual assignments |
| Income[s] laws | Income[s] only | Whole-EUR point laws | ≤256 authored definitions / unique canonical laws; ≤512 pairs/points per law; branch structural volume ≤131,072, subject to aggregate byte caps | Same pre-merge, canonical and aggregate-resource validation | Own cardinalities/supports/weights/zeros/tails/caps and resolution evidence |
| Income[u] age/map/laws | Income[u] only | Own age definition; bucket × Education × Region → own point law | ≤6 buckets, 55 compiled ages, 840 map slots, 64 authored definitions / unique canonical laws, ≤512 pairs/points per law, branch structural volume ≤32,768 subject to aggregate byte caps | Age, complete map, own references, pre-merge and canonical point-law validation; aggregate-resource preflight | Own boundaries, assignments, cardinalities and laws/resolution evidence |
| Income[t] age/map/laws | Income[t] only | Same schema, separate branch | Same maxima | Same validation, separate identity | Own boundaries, assignments and laws |
| Income[r] age/map/laws | Income[r] only | Same schema, separate branch | Same maxima | Same validation, separate identity | Own boundaries, assignments and laws |
| Income[i] age/map/laws | Income[i] only | Same schema, separate branch | Same maxima | Same validation, separate identity | Own boundaries, assignments and laws |

Runtime Company Industry metadata is an additional derived dense N-entry array of canonical uint8 Industry ordinals. EmployeeCount metadata remains the Contract-2 uint32 array. No SizeBand array is retained; derive it from count. No per-Person table exists. These arrays are compiler products, not D2-B authored Company rosters. Company ID derives under the frozen namespace/coordinate codec, independently of model magnitudes (`tools/fakedata/identity.go:111`, `tools/fakedata/identity.go:169`).

Shared output-sensitive assets are limited to the intentionally common frozen outcome and dimension semantics. Age thresholds, Region mappings, Income pools and numerical tables are not shared semantically. A physical file or deduplicated backing array may hold several logical units only if selective identities and immutability preserve their ownership. No shared empirical latent, national fallback, source mixture or global row model is introduced.

### 19. Constructor compilation and validation

**Chosen lifecycle:** D2-B acquires sources and fits numerical assets offline; the prescribed embedded numerical inputs then undergo bounded local constructor validation, canonicalization and compilation into complete immutable runtime state. Only after identities, Company metadata, all indexes and all law maps succeed may the constructor expose SocioeconomicWorld. No runtime external files/network, fitting, mutable reload, lazy discovery or partially initialized world is allowed.

Source-oriented numerical records may differ in order and contain duplicate numerical supports as expressly allowed. They cannot contain unbounded raw empirical datasets, provenance documents, source histories or fitting engines for runtime processing. Monetary reconciliation is already completed in D2-B. Constructor work is bounded by section 21, not by source dataset size.

Required compilation order and checks:

1. Validate base provenance/initialization using frozen state, without full Person or photo generation. Retain frozen error and membership boundaries.
2. Check numerical schema/version compatibility, dimension vocabulary/order and all declared sizes. Validate encoded and projected decoded bytes before potentially excessive allocation. Check all count/byte products, sums and integer conversions.
3. Validate the eight local age mappings and compile their 55-entry address arrays. Require exact 21–75 coverage and deterministic membership.
4. Validate every categorical row key and vector, reject duplicate/missing rows, check uint32 weights and positive checked totals, and perform GCD canonicalization. Require the full structural grid even when an upstream category currently has zero weight.
5. Validate count-law keys and supports, pre-merge authored-pair caps, exact support/weight widths, checked duplicate merging, zero-mass omission and GCD normalization. Check canonical support uniqueness, ascending order and representability. Require all 14 Industry laws.
6. Validate all six Income branches: preflight unchanged authored-definition counts, ≤512 authored pairs per law and aggregate encoded/decode/phase charges before potentially excessive allocation, then validate own age maps, exact tuple dimensions, all reachable assignments once, employee zero-band sentinel slice and law reference ranges. Perform within-branch support/law canonicalization, independently check ≤512 canonical points per law, complete R_out≤8 MiB and joint phase charges before excessive canonical allocation, and require no unused authored or canonical law. Canonicalize IDs by first encounter in parent tuple order. Reject duplicate parent cells even when their content agrees. Deduplication never admits input exceeding authored cardinality or input/decode byte caps; source volume fitting those caps need not equal retained volume.
7. Validate the CompanyCount literal, Employer C/I/S factor dimensions and permitted positive ranges. Preflight projected A=40N and worst-case P≤40N before Company/index-sized allocation.
8. Generate each operational Company's Industry followed by its count under isolated Company laws. Retain canonical Industry ordinal and checked uint32 count arrays; derive SizeBand without storing another sampled fact. Verify generated metadata agrees with the same Company-scoped laws used for retrieval.
9. Build all 40 complete-universe Employer inputs/indexes serially. Validate factors, every exact weight/product, zero-count exclusion, unique operational keys, canonical order, totals, full input volume, all phase resources and Contract-2 cumulative construction. Do not retain raw weight arrays.
10. Validate direct routing covers all 40 conditions exactly once and every index has positive total. No conditional fallback exists, including currently unreachable upstream combinations.
11. Establish complete separable canonical identity material, bind actual base WorldID in full identity, and expose only completed immutable state. Release constructor-only input and scratch storage.

No asset repair by smoothing, clamping, resampling, dropping parent cells, selecting a national row or substituting zero is performed here. Sorting, specified support merging, zero-mass support omission and GCD reduction are deliberate canonicalization, not repair. Employer input duplicates remain errors and are never merged, as Contract 2 requires.

Invalid prescribed model/data returns nil wrapper and ErrInvalidSocioeconomicWorld classification. Contract-2 all-zero and overflow validation must not expose ErrAllZeroWeights or ErrTotalWeightOverflow through errors.Is (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:741`). Do not broaden this into a change to frozen generation-error propagation. Unexpected internal failures remain non-nil opaque errors; successful generation always produces the promised complete result (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:586`). No public validation API or new error category is added.

A valid compiled lookup cannot encounter a missing reachable cell. An internal-corruption or implementation-invariant failure must return a non-nil opaque error and the frozen zero result, never an arbitrary employer, default law, skipped Income or partial successful facts.

### 20. Model non-interference / identity decomposition

**NORMATIVE:** retain separable logical identity material for Company Industry, EmployeeCount, Education, Activity, employee family, employee level, Employer, self-employed family, self-employed Industry and each of the six Income branches. Each model owns its numerical tables, own age mapping where applicable, effective parent map and relevant outcome semantics. Employer owns C/I/S, eligibility/factorization/direct routing semantics and the exact selector law. Company universe has separate domain identity material.

Full SocioeconomicWorldID binds all canonical output-affecting units, common semantic specification and actual base WorldID. It is not a universal stochastic seed. Protocol must use only a node's relevant own identity and authorized semantic values, plus compatible addressing, rather than ancestor-wide fingerprints or physical file checksums. The frozen stream factory hashes supplied roots and dependencies; isolation is possible but not automatic (`tools/fakedata/stream.go:44`, `tools/fakedata/stream.go:84`).

| Changed logical model | Permitted propagation | Required isolation |
| --- | --- | --- |
| Company Industry | Same-Company count/band; Employer selection; selected employee sector/Income | Other Company coordinates do not become semantic inputs to a Company's own generation; all upstream Person and self-employed models unchanged |
| EmployeeCount | Band, Employer, selected employee sector/band and Income | Industry, attainment, activity, occupations and all self-employed/non-working Income unchanged at fixed own parents |
| Education | Activity/gates, employee family/level/Employer and Income paths; self-employed family/Industry/Income; non-working Income via Education | Company facts and base inputs unchanged |
| Activity | Selected occupational branch and selected Income law/applicability | Education and Company facts unchanged |
| Employee family | Employee level, Employer, selected sector and employee Income | Self-employed branch, Education, activity and Company facts unchanged |
| Employee level | Employer and employee Income | Employee family and self-employed branch unchanged |
| Employer factors/index/routing model | Employer and selected payload-dependent employee Income | All already-generated occupations and Company facts unchanged |
| Self-employed family | Its Industry and Income | Employee branch and Company facts unchanged |
| Self-employed Industry | Its Income | Its family and employee branch unchanged |
| One Income branch | That branch's Income and future explicitly authorized descendants | Every Contract-4 fact and other five Income branches unchanged |
| CompanyCount alone | Membership/domain and Employer selection, with possible selected payload changes | Common operational Company coordinates retain facts at fixed Company models/addressing |

These are permissions, not a requirement that every descendant integer change. Stronger fixed-parent invariance also applies: if a child's own canonical law/model, applicable authorized parent values and permitted innovation remain fixed, unrelated ancestor-identity changes cannot change it. In particular, unchanged employee sector/band means changing Employer key or count within the band cannot perturb Income. A selected Company's direct sector lookup never reads another Company.

Evidence: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:596`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:623`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:656`.

Source provenance checksums and physical file checksums are audit evidence. They are not blindly reused as generation identities. Canonical-equivalent row order, weight rescaling, support decomposition and law relabeling leave output-sensitive numerical identity unchanged. An output-sensitive crosswalk/semantic interpretation change needs appropriate logical model/spec identity; changing only a source wrapper's formatting does not. Exact hashing, domains, framing and binary serialization are Protocol-owned.

Common frozen category/Income meanings are intentionally shared and changes to them may affect every actual consumer. This does not make a branch's empirical section a shared asset. Equivalent duplicated age definitions remain separate consumer identities. Physically shared data cannot force a single mega-model fingerprint.

### 21. Resource and performance architecture

All bounds below are enforceable architectural logical charges, not measured RSS or Go allocator size predictions. D2-B supplies concrete used bytes later. Backing-array capacities equal logical lengths. Additional buffers, metadata, allocator-independent implementation storage and identity material must be conservatively charged; no allocation can evade a budget through a different name.

#### Numerical payload accounting

For a small categorical family with r rows and Q weights:

```text
input numerical charge    = 64 + 8r + 4Q
retained numerical charge = 64 + 32r + 12Q
```

The retained per-weight charge reserves uint32 canonical weights plus up to eight bytes of sampling acceleration. It does not mandate a CDF layout or sampling algorithm. The 32-byte row charge covers a bounded descriptor and total. Actual excess enters explicit additional storage below.

For a support-law family, let L_a be authored law definitions and U_a their authored pairs before any merging/omission; let L_c be retained canonical laws and U_c their canonical positive-mass points. Input caps apply to L_a/U_a and retained caps to L_c/U_c:

```text
input numerical charge    = 64 + 8L_a + 8U_a
retained numerical charge = 64 + 32L_c + 16U_c
```

The retained per-point reservation covers uint32 support, uint32 weight and up to eight bytes of acceleration. For EmployeeCount, L_a=L_c=14, each law has ≤128 authored pairs/canonical points, and family U_a/U_c≤1,792; each Industry keeps its own law address. For each Income branch, 1≤L_c≤L_a≤its unchanged section-16 law-count limit, with ≤512 authored pairs/canonical points per law, U_a≤512L_a and U_c≤512L_c. Actual aggregate Income volume also requires the byte acceptance condition below. Canonicalization can reduce used cardinalities but cannot relax authored cardinality or input/decode byte caps. Simultaneously live input, canonical output and duplicate-merge scratch are charged separately by their backing storage in construction phases; do not substitute a reduced canonical count for live authored input.

For a tuple map with T slots, input/retained numerical charge is `64 + 2T`. For one age definition, reserve input `64+8=72` bytes and retained `64+8+55=127` bytes, covering at most eight uint8 lower bounds and 55 compiled uint8 addresses. Smaller actual mappings do not authorize exceeding the fixed reservation without an additional charge.

| Component | Input bound | Retained bound | Peak formula/bound | Runtime work |
| --- | --- | --- | --- | --- |
| Company Industry categorical law | 128 bytes: r=1, Q=14 | 264 bytes | Retained + live source/decoded/scratch within outside-model phase cap | Root address + fixed 14-category law |
| EmployeeCount laws | ≤14,512 bytes: L_a=14, U_a≤1,792 | ≤29,184 bytes: L_c=14, U_c≤1,792 | Retained + source/merge/sort storage within outside-model phase cap | Exact Industry address + ≤128-point sample |
| Education vectors | ≤4,384 bytes: r≤120, Q≤840 | ≤13,984 bytes | Same phase cap | Age array/Region address + seven-category law |
| Activity vectors | ≤35,904 bytes: r≤1,120, Q≤6,720 | ≤116,544 bytes | Same phase cap | Three coordinates + six-category law |
| Employee family vectors | 6,784 bytes: r=140, Q=1,400 | 21,344 bytes | Same phase cap | Two coordinates + ten-category law |
| Employee level vectors | ≤10,144 bytes: r≤420, Q≤1,680 | ≤33,664 bytes | Same phase cap | Three coordinates + four-category law |
| Self-employed family vectors | 6,784 bytes | 21,344 bytes | Same phase cap | Two coordinates + ten-category law |
| Self-employed Industry vectors | 704 bytes: r=10, Q=140 | 2,064 bytes | Same phase cap | Family address + fourteen-category law |
| Eight local age mappings | 576 bytes aggregate | 1,016 bytes aggregate | Retained + bounded compiler scratch | One 55-entry array address |
| Company-universe descriptor | Numeric descriptor reservation 72 bytes | 72 bytes outside Contract 2 | Included in outside-model phase cap | Membership comparison |
| Company Industry metadata | Generated from laws; no authored roster | `64+N` bytes | Retained plus Company generation staging under joint cap | One Company-index address |
| EmployeeCount metadata | Frozen staging reservation `8N` | Frozen `64+4N` | Enters Contract-2 phase formula below | One Company-index address |
| Employer factors | F=378 bytes: root 64 plus 2×157 values | F=378 bytes, charged in Contract-2 additional state | R2 plus factor/source/scratch live charges | Constructor-only factor arithmetic; no per-Person factor evaluation |
| Employer condition routing | No empirical input or retained map | Zero routing bytes; fixed stateless dispatch | Fixed control allowance; no variable structure | Constant direct condition address |
| Employer selector indexes | `64+32K+16A`, ≤64 MiB | `64+32K+16P`, ≤64 MiB | Exact Contract-2 formula below | Prebuilt binary search O(log q), no Company scan |
| Six Income tuple maps | ≤152,704 bytes aggregate; T≤76,160 | Same ≤152,704 bytes | Retained + map validation/remapping scratch under outside cap | Fixed-size tuple address + one law reference |
| Income[e] laws | Structural numerical ceiling 2,101,312 bytes: L_a≤512, U_a≤262,144; actual input/decode preflight | Individual numerical ceiling 4,210,752 bytes: L_c≤512, U_c≤262,144; aggregate R_out preflight | Retained + support/law canonicalization under unchanged outside/joint caps | ≤512-point law sample |
| Income[s] laws | Structural numerical ceiling 1,050,688 bytes: L_a≤256, U_a≤131,072; actual input/decode preflight | Individual numerical ceiling 2,105,408 bytes: L_c≤256, U_c≤131,072; aggregate R_out preflight | Same phase cap | ≤512-point law sample |
| Each Income[u/t/r/i] law family | Structural numerical ceiling 262,720 bytes: L_a≤64, U_a≤32,768; actual input/decode preflight | Individual numerical ceiling 526,400 bytes: L_c≤64, U_c≤32,768; aggregate R_out preflight | Same phase cap | ≤512-point law sample |
| All outside-Contract-2 numerical assets, identity/metadata and excess | Encoded inputs E≤8 MiB; decoded input/working state D≤8 MiB | R_out≤8 MiB | `R_out+E_live+D_live ≤24 MiB`; joint phase preflight also required | No model-sized work beyond listed fixed domains |

The seven small categorical families are Company Industry, Education, activity, employee family, employee level, self-employed family and self-employed Industry. Their aggregate maxima are r=1,951 and Q=12,194, input 64,832 bytes and retained 209,208 bytes. EmployeeCount and Employer are accounted separately.

All six Income families retain the aggregate law-count ceilings L_a/L_c≤1,024. Per-law 512-point limits imply a structural U_a/U_c ceiling of 524,288; the corresponding authored numerical charge is at most 4,202,880 bytes. These are structural input-size bounds, not exemptions from actual encoded/decode/work caps. Retained individual ceilings in the table are not simultaneously admissible: 1,024 canonical laws each with 512 points would charge 8,421,760 bytes for Income laws alone, before other outside state.

**NORMATIVE aggregate byte acceptance, using the existing resource system:** let R_other be every outside-Contract-2 retained charge except canonical Income law families, including categorical/count assets, all tuple/age maps, Industry metadata, the universe descriptor and all applicable additional identity/header/storage charges. With branch-local actual counts, require:

```text
IncomeInputCharge = Σ_b (64 + 8L_a,b + 8U_a,b)
IncomeRetainedCharge = Σ_b (64 + 32L_c,b + 16U_c,b)
R_out = R_other + IncomeRetainedCharge <= 8,388,608 bytes
```

IncomeInputCharge is part of decoded numerical input accounting; actual encoded input E≤8 MiB and all simultaneously live decoded/work state D≤8 MiB still govern it and source-format/auxiliary storage. Validate authored counts and projected input/decode charges before excessive allocation. Establish actual canonical counts within bounded workspace or validated streaming/in-place compilation, then preflight complete R_out and joint peak before allocation that would cross a cap; a worst-case retained estimate is not permission to allocate oversized output, nor a reason to reject smaller canonical output when all authored limits are met. No aggregate point-count exemption or new resource budget exists. More points in one law consume the same existing per-point bytes and therefore leave fewer bytes for other laws or outside state.

#### Non-empirical Income capacity stress and allocation witnesses

Use maximum existing categorical/count/map/age reservations and Nmax Industry metadata. Their main outside charge excluding Income laws is 497,031 bytes; the existing universe descriptor adds 72 bytes. For illustration only, reserve a further 65,536 bytes of existing R_other for identity/header/other excess. This is neither a new cap nor a required D2-B allocation. At all unchanged law-count maxima, L_c=1,024 and Income family roots/descriptors charge 384+32×1,024=33,152 bytes.

| Hypothetical points in every canonical Income law | U_c | Income law retained charge | Main outside retained before universe/excess | R_out including universe + illustrative 64 KiB excess | Retained-cap handling |
| --- | ---: | ---: | ---: | ---: | --- |
| 128: historical baseline | 131,072 | 2,130,304 | 2,627,335 | 2,692,943 | Fits this reservation; old capacity is still permitted |
| 256: stress only, not chosen per-law ceiling | 262,144 | 4,227,456 | 4,724,487 | 4,790,095 | Fits this reservation; not the F2 capacity choice |
| 384: stress only, not chosen per-law ceiling | 393,216 | 6,324,608 | 6,821,639 | 6,887,247 | Fits this reservation; not the F2 capacity choice |
| 512: simultaneous maxima | 524,288 | 8,421,760 | 8,918,791 | 8,984,399 | Reject complete retained allocation before crossing R_out≤8 MiB |

**Usable high-resolution allocation:** with all 1,024 law slots occupied by distinct canonical laws, one law has 512 points and the other 1,023 have 128. U_c=131,456 and R_out=2,699,087 bytes under the same reservation, so a heavily used law actually can receive 512 points. If authored and canonical representations agree, the numerical authored charge is 1,060,224 bytes. E/D/phase preflight still applies to actual encoding and scratch.

**Near-limit allocation and rejection:** 900 distinct canonical laws with 512 points, 87 with 212 and 37 with 211 give L_c=1,024 and U_c=487,051. Branch law caps are satisfied, for example by assigning the 900 high-resolution laws to all 512 employee, all 256 self-employed, all 64 unemployed, all 64 student and four retired laws; the remaining 60 retired plus 27 inactive laws have 212 points and the remaining 37 inactive laws have 211. R_out=8,388,607 bytes, one byte below the existing cap; raising one 211-point law to 212 gives U_c=487,052 and R_out=8,388,623, 15 bytes above it, so that allocation rejects before excess retained storage is allocated. Matching authored representations charge 3,904,984 bytes of numerical input at the accepted boundary; actual E/D and simultaneous scratch must also fit their existing caps. These U values are conditional arithmetic witnesses, not normative global point limits: actual R_other and law counts determine available capacity. The existing 24 MiB outside-phase reservation and joint retained/peak policy are unchanged.

Encoded input limits include actual source-format bytes presented to the constructor. Generated Employer candidate inputs are separately bounded by Contract 2's semantic-input-volume charge; they are not an encoded external numerical asset and are not substituted for that 64 MiB volume bound. Numerical charge formulas describe bounded decoded content, not a serialization format or permission to disregard long strings. Every remaining header, dimension representation, version key, decoder object, identity digest/material, sort/hash auxiliary allocation and control excess is charged in R_out or D. Raw empirical data and provenance documents are not constructor inputs. E and D are distinct live storage categories; aliases count backing storage once, copies separately. All prospective allocations must pass both their component cap and phase sum before allocation.

#### Contract-2 and joint construction accounting

Let H2 be any implementation-specific Contract-2 retained excess beyond its frozen descriptors/entries/count array and the factor charge F. D2-A permits at most 1 MiB of such excess; it does not authorize retained raw weight arrays or another index architecture. Charge it through Contract 2's additionalRetainedBytes, not through routing or outside-model state:

```text
additionalRetainedBytes = F + H2
0 <= H2 <= 1 MiB
R2 = (64+4N) + (64+32K+16P) + F + H2
R2 <= 72 MiB
```

Keep the frozen semantic-input budget `64+32K+16A≤64 MiB`. In each serial index phase, current input count m≤M=N. D2-A bounds total canonicalization working storage by `32m` bytes, exclusive of control bookkeeping already in Contract 2's 4 KiB allowance. A private copy plus scratch is a feasibility witness, not a prescribed sorting algorithm. Any chosen algorithm must fit this bound and Contract 2's O(m log m)-or-better work rule.

Frozen phase accounting remains:

```text
C2peak = R2 + 8N + 16m + canonicalizationWorkingBytes
             + otherLiveConstructionBytes + 4,096
C2peak <= 128 MiB
```

Every source/model-loading or other simultaneously live buffer not already in a named term enters otherLiveConstructionBytes during that phase. Conservatively include all live outside-model state as well if it is needed or retained during the build; never treat it as free. Under the outside-model caps, a reservation of at most 24 MiB covers it. Additional Contract-2 construction temporaries must fit the remaining phase allowance, and D2-A's joint world-construction cap below; they cannot displace or conceal outside-model charges.

At Nmax, q=N, H2=0:

```text
companyMetadataBytes = 419,484
retainedIndexBytes = 67,108,544
R2 = 67,528,406
reference C2 peak before other-live state =
    R2 + 8N + 16N + 32N + 4,096 = 73,404,382
with full 24 MiB outside-model reservation = 98,570,206
with maximum H2 as well = 99,618,782 bytes
```

These are below the frozen 128 MiB limit of 134,217,728 bytes. At maximum H2 plus full 8 MiB R_out, joint retained state is at most 76,965,590 bytes, below D2-A's 80 MiB joint retained cap. This proof leaves roughly 33 MiB of construction allowance before the 128 MiB cap; actual extra temporaries still require preflight and concrete accounting, not automatic permission.

**D2-A whole-layer caps:** prescribed encoded numerical inputs ≤8 MiB; outside decoded/work state ≤8 MiB; outside retained state ≤8 MiB; joint Layer-2 retained charge ≤80 MiB; joint Layer-2 constructor live/reserved peak ≤128 MiB. Existing supplied Layer-1 World/assets are excluded, but every new Layer-2 allocation is included. These do not weaken Contract-2's own 64/64/72/128 MiB limits. Check each applicable cap separately.

Joint peak counts the disjoint union of allocations/reservations once: where outside state is already included in Contract-2 otherLiveConstructionBytes, do not add it again. Outside compilation phases include any already-live R2 state. The conservative example above fully covers both index construction and outside compilation with their simultaneous state; successful implementation must present phase-by-phase evidence for its actual schedule. No RSS assertion or double-charge is used to conceal an omitted allocation.

#### Work bounds

- Company model address and generation use fixed domains: 14-category root and at most 128 support points. Company metadata retrieval and SizeBand derivation are O(1).
- Person node model address lookup is O(1) via fixed-dimensional compiled addressing, with at most 14 categorical entries or 512 Income support points sampled. With these architectural caps, bounded linear scans or logarithmic support search are O(1) with respect to population and CompanyCount; exact choice is Protocol-owned.
- Employer direct routing is O(1); selection is the frozen prebuilt O(log q) binary search. No per-Person O(N) scan, reweighting, occupancy check or allocation occurs.
- Income uses bounded tuple lookup plus bounded support sampling. No history, Company-universe or source scan occurs.
- Numerical compilation is linear in bounded input/map volume plus bounded support/law canonicalization. A bounded law-equivalence scan is permitted; maximum Income support comparison work is fixed by 1,024 laws and 512 points, with actual aggregate volume resource-preflighted. Company generation is O(N); 40 index builds are O(40N log N) or better.
- Sampling works on exact positive totals, never expanded weights. Model lookup and storage have finite bounds. Frozen rejection sampling does not promise a universal maximum raw-draw count (`tools/fakedata/random.go:59`); D2-A does not add a rejection cutoff or alter its deterministic semantics.

Limits and working-space implementations alone are resource-sensitive. For models valid under both versions, a resource-only change preserving every law, interval and output does not force numerical identity drift (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:898`). Actual supports, assignments, factors, bucket boundaries or CompanyCount remain output-sensitive.

### 22. Deterministic Protocol boundary

Protocol shall specify exact streams, paths, Component names, framing, compatible seed decomposition, fingerprint computation, node sampling call sequences, support/CDF/search realization, canonical binary identity serialization and SocioeconomicWorldID bytes. None is invented here. Logical table order, canonical laws, tuple addressing, finite storage ceilings and 40 direct indexes must be preserved by that realization.

Contract 2 already fixes Employer cumulative intervals, ordering, zero omission, one Uint64n(total) invocation and strict binary search. Protocol may complete Person stream dependencies and Component identities, but cannot reopen those mechanics. No alias, hierarchical Employer selection or direct modulo draw is allowed.

Protocol must realize independent node/branch innovations, fixed-parent invariance, random access, excluded-input boundaries and canonical-equivalent-law invariance. It must not include Employer coordinate in employee Income innovation, broad upstream model identities where only parent values are authorized, or full SocioeconomicWorldID/base WorldID in every stream. Full world identity coverage and selective realization isolation are separate requirements (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:753`).

Exact whole-EUR conversion is already a Contract-5 semantic rule and is applied offline to D2-B supports. Protocol samples only canonical whole-EUR outcomes. It cannot reintroduce a fractional latent or a new source-state draw. No new public protocol API, Company API or caller configuration is authorized.

The Income capacity ceiling is now 512 points per law; EmployeeCount remains 128. This changes only bounded worst-case work and capacity acceptance, not the sampling algorithm boundary. Protocol still owns bounded-integer realization, CDF/search representation, permitted linear versus binary support lookup, exact draw sequence, paths, Components and framing. No interpolation or jitter is introduced.

### 23. D2-B required deliverables

D2-B must deliver all of the following, using this architecture without redesign:

1. One actual CompanyCount literal in `1..104,855`, with rationale and model revision.
2. Every logical asset in section 18, populated with actual permitted numerical values, mappings and boundaries, plus canonical compiled equivalents.
3. For each asset: source/provenance, source-to-semantic crosswalk, fitting/transformation method, complete structural parent coverage, canonical asset, separable identity/checksum input, concrete input/retained/peak accounting and Plausible+ validation evidence.
4. Actual EmployeeCount supports/weights for all fourteen Industry laws, band coverage evidence, skew/tail design and documented production/model caps; no sampled band or generated occupancy evidence as a source of count.
5. Actual fitted C in 1..65,535 and I/S magnitudes in 1..4,095, zero-count exclusion and positive support proof for all 40 conditions, index manifests and exact per-index/aggregate totals. Explain saturation at the chosen C without treating its architecture maximum as empirical evidence. Derived weights must match the unchanged formula; D2-B cannot supply arbitrary per-Company weights.
6. Actual support/weights, per-law authored/canonical cardinalities and tuple assignments for each Income branch, actual zero mass (including explicit zero mass where absent), branch-specific tail and cap evidence, and whole-EUR canonicalization. Report L_a/U_a before merging and L_c/U_c afterward for every branch; demonstrate unchanged law-count caps, ≤512 pairs/points per law and complete aggregate input/decode/retained/peak acceptance. Allocate higher resolution where fitted high-volume laws need it; no universal empirical cardinality is required.
7. Actual counts and boundaries for all eight permitted age maps, with full coverage and sensitivity evidence. No Region pool/per-axis nominal map is a deliverable because this architecture authorizes none.
8. Deterministic, versioned source-to-Income-concept reconciliation: personal versus household unit, all-source coverage, gross/net boundary, business attribution, capital exclusions, source period, annualization, currency/nominal anchor basis, exact precision/ceiling, zero interpretation and domain checks.
9. Distinguish directly compatible observations from calibration evidence and assumption-based modeling. Salary-only, historical realized, signed taxable or household statistics must not be relabeled as directly equivalent all-source current personal Income (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:704`).
10. Document every zero weight resembling a semantic exclusion, every unsupported/sparse source cell and how offline fitting completed it. Identify limitations of the frozen DAG rather than add hidden state or unauthorized residual parents.
11. Concrete phase-by-phase resource evidence, including encoded/decode buffers, canonicalization, identity material, Company arrays, input volume, indexes, scratch, excess terms and prospective-allocation guards.
12. Reproducible canonical artifacts and checksums, reviewable transformations and Plausible+ evidence over individual profiles, filter-visible relationships, aggregates, rare combinations and construction seed risks. For high-volume conditional Income laws, demonstrate that fitted discretization does not produce an obvious small-lookup-table appearance using appropriate distinct-value repetition, histogram occupancy, visible spacing/gaps, zero mass, central-body resolution and upper/high-tail coverage checks. D2-B owns actual fit thresholds and evidence; D2-A prescribes no universal empirical repetition or bin threshold. No scientific population-representativeness claim is required.

**D2-B deliverable matrix:** “provenance” includes crosswalk and fitting/transformation assumptions even for a predominantly modeled asset. Every Income law-count limit applies separately to authored definitions and unique canonical laws; contents include actual per-law cardinalities, fit evidence includes high-volume-law discretization checks, and resource evidence includes both L_a/U_a and L_c/U_c under aggregate byte preflight.

| Model/asset | Numerical contents | Provenance required | Fit evidence | Resource evidence |
| --- | --- | --- | --- | --- |
| Canonical dimensions/crosswalks | Exact frozen keys; source mappings only | Classification/version and mapping boundaries | Known functional/attainment/sector examples; no taxonomy reinterpretation | Bounded runtime dimension metadata; offline documents excluded |
| Company universe | Actual literal N | Enterprise-universe/model rationale | Broad usable Company variety; construction seed-risk evidence | N, 40N input projection, Company arrays and joint caps |
| Company Industry | 14 weights | Sector mapping and Company-mix assumptions | Major sector balance, rare support and Company independence | One-vector bytes |
| EmployeeCount | 14 point laws and actual cap choices | Industry/headcount mapping and tail transformation | Zero, every band, skew/tail and Industry differences | Point counts, bytes, staging and uint32 safety |
| Education age/vectors | ≤6 boundaries and full seven-state rows | Attainment/cohort/Region reconciliation | Broad age and Region attainment differences | Actual B_E, rows, weights, map bytes and compilation peak |
| Activity age/vectors | ≤8 boundaries and full six-state rows | Activity overlaps, age/attainment/Region mapping | Visible lifecycle/attainment/Region effects without rigid gates | Actual B_A, rows, bytes and compilation peak |
| Employee family | 140 ten-state rows | Employee occupation-to-function mapping | Education/Region composition and rare functions | Rows/weights/input/retained bytes |
| Employee level age/vectors | ≤6 boundaries and four-state rows | Responsibility interpretation; modeling assumptions | Age/attainment/function tendencies, expert IC and low-education executive support | Actual B_L, row/byte totals and peak |
| Self-employed family | Own 140 ten-state rows | Self-employed function mapping | Branch-distinct composition and rare functions | Own row/byte totals |
| Self-employed Industry | Ten fourteen-state rows | Self-employed activity/function mapping | Family×Industry association, unusual support | Own row/byte totals |
| Employer factor/index assets | C; 140 I; 16 S; all derived 40 indexes | Workforce prevalence and compatibility/size assumptions | Industry×family and size×level matching; all positive combinations | Exact W/totals, A/P/M/K, F/H2, every Contract-2/joint phase |
| Income[e] map/laws | ≤512 laws, assignments, support/weights, zeros/caps | Personal all-source annualized gross whole-EUR mapping | Function/level/Region/sector/size segments, zeros and tail | Actual maps/laws/points and branch/aggregate bytes |
| Income[s] age/map/laws | ≤6 buckets, ≤256 laws and assignments/supports/weights/zeros/caps | Personal attribution after business costs; compatible aggregate source assumptions | Age/Region/function/Industry segments and wide skew/tail | Actual B_s, map/law/point bytes and compilation peak |
| Income[u] age/map/laws | Own ≤6 buckets, ≤64 laws and assignments/supports/weights/zeros/caps | Unemployment aggregate receipt assumptions; no history | Age/attainment/Region receipt segments and positive/zero support | Own B_u/map/law/point/peak totals |
| Income[t] age/map/laws | Own ≤6 buckets, ≤64 laws and assignments/supports/weights/zeros/caps | Student aggregate receipts; exclude household support | Adult student receipt variation without enrollment latent | Own B_t/map/law/point/peak totals |
| Income[r] age/map/laws | Own ≤6 buckets, ≤64 laws and assignments/supports/weights/zeros/caps | Current retirement aggregate receipts; no former salary model | Age/attainment/Region variation, zero and tail | Own B_r/map/law/point/peak totals |
| Income[i] age/map/laws | Own ≤6 buckets, ≤64 laws and assignments/supports/weights/zeros/caps | Inactive aggregate receipt assumptions; no reason/household latent | Age/attainment/Region variation, zero and tail | Own B_i/map/law/point/peak totals |
| Canonical identity/checksum manifests | Separable canonical model material; no identity bytes chosen here | Sources, transformations, revisions and audit checksums | Canonical equivalence and node/branch isolation evidence | Bounded manifest/identity material and compiler storage |

### 24. Required future verification

Specify and later implement meaningful tests of actual compiled/generated behavior; this task implements none. Internal fixtures do not create public model configuration or justify changes to frozen source.

| Area | Required verification |
| --- | --- |
| Weight canonicalization | Proportional vectors, zeros, all-zero rejection, GCD exactness and support duplicate decompositions; identity and realization use the same canonical law |
| Category ordering | Source row permutations, map orders and authored law labels cannot change canonical tables, identity or output; exact taxonomy/Region dimension sets |
| Parent lookup | Exhaust every structural categorical and reachable Income tuple; exactly one valid resolution; duplicate/missing cells reject even where current upstream probability is zero |
| Age/Region | Every age 21–75, both sides and exact point of every boundary, full contiguous coverage; all 20 exact Region labels; finer geography cannot enter or alter mapping |
| EmployeeCount | Intentional zero, all five band regions and high-tail/MaxUint32 fixtures; exact integer output, duplicate merging, overflow rejection, no segment/band state and no count-to-Industry feedback |
| Employer conditions/support | Exhaust 40 conditions; every positive-count candidate positive, zeros excluded, unusual Industry/function and micro executive combinations retained; no occupation revision |
| Employer arithmetic/routing | C full accepted envelope 1..65,535 including 4,095/4,096/65,535, malformed C=65,536 rejection before uint16 narrowing; I/S maximum 4,095 unchanged; exact Wmax=1,098,958,053,375 and selector-total maximum 115,231,246,686,635,625; counts 4,095/4,096/20,000/100,000 distinguish below actual C and saturate only at fitted C when other applicable facts are fixed; complete direct routing, canonical intervals and checked malformed factor/sum rejection |
| Employer identity exclusion | Equal candidate facts imply equal W regardless of coordinate/ID; no identity economic factor; ascending coordinate order still supplies frozen deterministic intervals |
| Employer resources/work | Exact Nmax and Nmax+1, A/P/M/K, positive/zero input accounting, prospective allocation guards, identical membership/index counts/entry widths/resource bytes independent of accepted C magnitude, no raw-weight retention or O(N) per-Person scan; all phase/extra-state charges |
| Income supports | Zero, positive/high/MaxUint32, whole-EUR range, source ceiling fixtures, duplicate whole-EUR merging, sorted unique canonical support, point caps and positive totals |
| Income input cardinalities | For every branch, unchanged authored law-definition limit and limit+1; 512 authored pairs accepted when all byte caps fit, 513 rejected before support merging; independently validate 512/513 canonical positive points; declared/observed counts and reject-before-excess-allocation ordering; referenced duplicate definitions within limits merge canonically but cannot rescue excessive authored counts/bytes |
| Income capacity/resources | Historical 128-point law still valid; one 512-point law with lower-resolution laws; 256/384/512 simultaneous-density stress; complete authored encoded/decode and canonical R_out/phase preflight; below/above actual aggregate retained boundary including section-21 one-byte-below and 15-byte-above witnesses; high-volume segment repetition, histogram occupancy, spacing/gaps, zero/body/tail evidence; no interpolation/jitter and lookup/sampling work bounded by 512 points |
| Income mapping/branches | Complete mappings and within-branch law canonicalization; one selected branch evaluation only; all other branch probes must fail if accidentally called |
| Income relational isolation | Substitute equal-sector/equal-band employers at fixed other parents/Person innovation; equal Income. Vary count inside band; unchanged Income. Unselected Company access must observably fail |
| Hidden state | Call-graph/input/innovation audits exclude support indices as retained state, source classes, affluence scores, sampled SizeBand, cross-node reused draws and Layer-3 access to internal sample positions |
| Non-interference | Change each logical model separately, force distinguishable local outcomes and verify every non-descendant; branch changes preserve all other Income branches despite full world identity drift |
| Shared physical assets | One logical section edit and source-file reorder preserve other node identities/results; duplicated equal mappings remain independently owned |
| Construction | Malformed/incompatible/incomplete/oversized assets fail before excessive allocation or usable-world escape; nil wrapper/zero results and frozen error identities; no successful fallback |
| Random access/compatibility | Repeated/reverse/concurrent retrieval and ownership; Facts/composite equality; ReferenceDate/catalog invariance; frozen Layer-0/1 goldens unchanged; appropriate future race checks |
| D2-B Plausible+ | Source-backed assumptions, broad filter-visible correlations, aggregate coherence, unusual positive support, zero/tail choices and seed construction risk; no population representativeness requirement |

Tests must distinguish semantic validity from numerical support and prove branch non-evaluation, not merely nil final payloads. Allocation instrumentation or equivalent evidence must establish reject-before-allocation ordering. Benchmarks support complexity claims; they do not replace invariance or arithmetic tests. Exact new sampling/identity goldens wait for Protocol.

### 25. Adversarial architecture review

| # | Attack | Prescribed handling |
| --- | --- | --- |
| 1 | All-zero categorical vector | Reject fixed asset at construction; no default category or successful zero outcome |
| 2 | Common categorical scaling | Exact GCD reduction before identity/sampling; identical canonical law and realization |
| 3 | Reordered categorical source rows | Resolve semantic keys into frozen vector/grid order; no output or identity drift |
| 4 | Semantically valid category given zero weight | Accept if row total positive; preserve category validity and explicit zero weight; D2-B documents exclusion-looking zeroes |
| 5 | Missing parent cell | Reject construction, including structurally valid cells with current upstream zero probability |
| 6 | Duplicate parent cell | Reject even if values match; only numerical support duplicate merging is permitted |
| 7 | Sparse empirical source | D2-B completes the fixed model offline using documented assumptions/smoothing and bounded Income sharing; constructor/generation do not infer missing cells |
| 8 | Age exactly on a boundary | Belongs uniquely to interval whose lower bound equals that age; compiled 55-entry address proves coverage |
| 9 | Missing Region pool member | No Region pool is authorized; missing exact Region coverage or a proposed Region pool rejects |
| 10 | EmployeeCount zero | Valid operational Company, derived zero band, W=0 in all 40 indexes; existence unchanged |
| 11 | Count support in every SizeBand | Express directly through count points; derive band only after count; D2-B documents actual positive support/omissions |
| 12 | Very large EmployeeCount | Accept uint32 support; derived large band; Employer min(count,C) bounds prevalence; no workforce-sized allocation or clipping of count |
| 13 | Unusual family×Industry employer pair | Strictly positive I and W for every positive-count candidate; magnitude carries rarity |
| 14 | Executive at micro employer | Strictly positive S and W; no size/level validity gate or resampling |
| 15 | All 40 Employer conditions | Every condition has its direct complete-universe input and positive-total Contract-2 index; exhaust coverage before world escape |
| 16 | Maximum permitted N | N=104,855 fits input/index and demonstrated peak envelopes; N+1 rejects before Company/index allocation even with many zero counts |
| 17 | Employer multiplication overflow | Valid factors cannot overflow by the proof; invalid ranges reject before multiplication; checked widened products and totals still fail closed |
| 18 | Selected count changes within same band | Employer W may change below C; occupation remains fixed; Income cannot change when Employer and all Income value parents remain fixed; above C W may be identical |
| 19 | Income zero | Ordinary zero support in selected law, legitimate no-Income outcome; no hidden zero state or missingness meaning |
| 20 | Very high Income | Accept whole-EUR uint32 support within D2-B's documented finite law; above public range rejects before conversion; no saturation |
| 21 | Duplicate Income supports | Bound authored definitions and pairs before any deduplication, checked uint64 support merge, zero-mass omission, GCD and uint32 canonical-weight check; equivalent admissible decompositions yield same law; oversized duplicate definitions reject even if canonical output would fit |
| 22 | One Income branch model changes | Only that branch's Income and authorized descendants may change; all Contract-4 facts and other branches stay identical |
| 23 | Shared asset changes | Only intentionally shared frozen semantics can affect all actual consumers; local age/law data are not shared semantics; physical deduplication preserves separate identities |
| 24 | Shared file changes unrelated section | Whole-file provenance checksum may change; unrelated logical output identities and realizations remain identical |
| 25 | Malformed asset | Reject schema, dimensions, references, ranges, totals, coverage or identity incompleteness; no usable wrapper |
| 26 | Construction exceeds a cap | Checked complete phase preflight rejects before allocation that would cross component or joint cap |
| 27 | Runtime uncovered condition | Impossible for validated structural lookup; detected corruption/bug returns opaque error and zero result; no fallback or partial success |
| 28 | Proportional authored weights differ | Categorical/count/Income canonical laws and output-sensitive identities agree; Employer final intervals deliberately remain magnitude-sensitive under Contract 2 |
| 29 | C=4,095 / 4,096 / 65,535; malformed C=65,536 | Accept all three positive in-envelope fitted values; reject 65,536 before uint16 narrowing, and reject zero/noninteger values; I/S remain 1..4,095 |
| 30 | Counts 4,095 / 4,096 / 20,000 / 100,000 or count>C | With C=65,535 and fixed Industry/large band, workforce factors are 4,095 / 4,096 / 20,000 / 65,535 respectively; with C=4,095 all four saturate at 4,095. Saturation depends only on actual fitted C; retain actual public counts, positive support, fixed occupation and restricted Income parents |
| 31 | Maximum I/S/C product and selector total | At C=65,535 and I=S=4,095, Wmax=1,098,958,053,375; at Nmax total≤115,231,246,686,635,625<MaxUint64. Require checked arithmetic; the same positive membership, 40 indexes, uint64 entries and resource formulas apply independently of C magnitude |
| 32 | Old 128-point Income law / new 512-point law | Both admissible capacities when complete maps, positive totals and all byte caps fit; 512 is available resolution, not a requirement to pad every law |
| 33 | 513 authored Income pairs in one definition | Reject before duplicate-support merging or zero-mass omission, even if canonical content would shrink below 512 |
| 34 | 513 canonical positive-mass Income points | Reject malformed canonical model data before usable-world escape; canonical cardinality is independently bounded at 512 and cannot be produced by valid ≤512-pair source canonicalization |
| 35 | High-resolution law with many lower-resolution laws | Accept a 512-point law with 1,023 128-point laws under the section-21 reservation: U_c=131,456, R_out=2,699,087; require actual authored/decode/phase caps too |
| 36 | Aggregate retained boundary below / above limit | Under section-21 actual-count reservation, U_c=487,051 charges R_out=8,388,607 and fits; one additional canonical point charges 8,388,623 and rejects before excessive allocation. Recompute with actual R_other/L_c, not this illustrative U as a global cap |
| 37 | Large segment resolves to one high-resolution law | Use the same authorized tuple→law mapping and one whole-EUR finite law; up to 512 outcomes including a dense body/zero/tails are available. D2-B must show actual repetition/occupancy/spacing/body/tail Plausible+ evidence; no jitter, interpolation, latent or added parent repairs are allowed |
| 38 | Authored input exceeds encoded/decode cap but deduplicates small | Reject before the excessive encoded/decode allocation. Exact duplicate support/law merging and proportional GCD equivalence apply only to admissible authored inputs and cannot evade byte caps |

**Minimality closure:** retain only the common integer/support primitives, seven categorical families, eight model-local age maps, forty direct Employer indexes with three minimal factors, and six bounded Income tuple maps/law pools. Each age map represents an authorized age relationship; separate ownership preserves required non-interference. Exact Region removes a pool asset. Count/support points remove sampled segments. Employer I/S express frozen matching relationships and the cap bounds count prevalence. Income sharing bounds an otherwise large conditional-law cross-product and supports sparse-source fitting. No interaction table, mixture component, regional pool, shared empirical mapping or optional distribution alternative remains.

### 26. Decision ledger

| Subject | D2-A decision | Frozen now? | D2-B freedom | Protocol freedom |
| --- | --- | ---: | --- | --- |
| categorical weights | Exact dense uint32 integer weights; positive uint64 totals | Yes | Values within range | Exact sampling realization |
| proportional canonicalization | Exact per-law GCD before identity and sampling; Employer excluded | Yes | Canonical ratios/values | Identity encoding, preserving equivalence |
| category order | Explicit Contract-3 displayed order and fixed Region list | Yes | None | Serialization and search realization |
| age architecture | Eight local contiguous maps; Education/level/all age-Income ≤6, activity ≤8; 21–75 complete | Yes | Actual counts/boundaries only | Concrete addressing realization |
| Region architecture | Exact 20-value axis everywhere authorized; no Region pooling | Yes | Numerical rows/tuple-law assignments, no Region map | Concrete addressing realization |
| Industry model | One 14-weight root vector | Yes | Weights | Sampling |
| EmployeeCount model | Industry-indexed finite count points; ≤128 authored/canonical points per law | Yes | Supports/weights and empirical/model caps | Sampling |
| Education model | Own age bucket × Region → seven-weight vector | Yes | Boundaries/weights | Sampling/addressing |
| Activity model | Own age bucket × Education × Region → six-weight vector | Yes | Boundaries/weights/offline overlap fitting | Sampling/addressing |
| Employee JobFamily | Exact Education × Region → ten-weight vector | Yes | Weights | Sampling |
| Employee JobLevel | Own age bucket × Education × family → four-weight vector | Yes | Boundaries/weights | Sampling/addressing |
| SE JobFamily | Own exact Education × Region law, independently identified | Yes | Own weights | Isolated sampling |
| SE Industry | Family → fourteen-weight vector | Yes | Own weights | Sampling |
| Employer eligibility | Zero count ineligible; every positive count positive in all 40 conditions; no extra hard zeros | Yes | None | Frozen Contract-2 selector and isolated stream |
| Employer factorization | Unchanged min(count,C) × family/Industry factor × level/positive-band factor; uint16 I/S in 1..4,095, C in 1..65,535 | Yes | Fitted C and I/S magnitudes within separate ranges | Selector/identity realization; no factor redesign |
| Employer index architecture | 40 direct complete-universe inputs; common positive-count retained subset; stateless routing | Yes | N/factors within limits, derived intervals | Component identity and compliant construction realization |
| Income representation | Whole-EUR finite point laws; separately ≤512 authored pairs/canonical positive points per law; aggregate byte preflight | Yes | Supports/weights, actual cardinalities, resolution allocation and model caps within unchanged budgets | Sampling, no new algorithm |
| Income zero representation | Ordinary same-law support point; no separate draw | Yes | Actual zero mass | Sampling |
| Income law pooling/mapping | Complete branch tuple maps; separately ≤512/256/64/64/64/64 authored definitions and unique canonical laws; pre-merge input caps; local canonical IDs | Yes | Complete assignments and law contents within both caps | Bounded concrete addressing and identity encoding |
| constructor compilation | Validate/canonicalize/compile complete immutable assets and Company/index state before return | Yes | Prescribed concrete inputs | Compliant deterministic realization, not lifecycle redesign |
| runtime fallback | None | Yes | None | None |
| resource maxima | N≤104,855; EmployeeCount points≤128, Income points≤512 per law plus aggregate byte preflight; unchanged law-count caps; outside 8/8/8 MiB, joint retained80/peak128 MiB; Contract-2/H2 limits unchanged | Yes | Actual per-law/aggregate allocation and concrete usage below every cap | No relaxation; compliant storage/work realization |
| empirical values | Deferred | No | Exclusively D2-B | None |
| source provenance | Deferred | No | Exclusively D2-B | Canonical identity serialization only |
| stream/framing | Deferred | No | None | Exclusively Protocol |
| exact identity bytes | Deferred | No | Canonical logical/checksum inputs, no encoding choice | Exclusively Protocol |

“SE” means self-employed. Resource ceilings are architectural values, not empirical observations. Protocol freedom is constrained by frozen Contract-2 mechanics and this numerical architecture.

### 27. Cross-contract audit

Evidence anchors for this audit are the exact parent tables (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:452`; `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:527`), hidden-state policy (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:379`), Employer arithmetic/resource protocol (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:494`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:514`), and Income semantics/isolation (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:188`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:656`). Classifications concern the finalized architecture, after its self-review corrections.

| # | Audit subject | Classification | Explanation |
| --- | --- | --- | --- |
| 1 | Integer categorical weights vs RNG foundation | NO ISSUE | Exact positive uint64 totals are compatible with checked bounded sampling; no frozen RNG change |
| 2 | Proportional canonicalization vs identity | NO ISSUE | Canonical categorical/support law precedes both identity and sampling; Employer exact intervals remain separately magnitude-sensitive |
| 3 | Bucketing vs semantic parents | NO ISSUE | Every bucket uses only its node's authorized ordered AnchorAge; no new information |
| 4 | Region pooling vs finer geography | NO ISSUE | No Region pool is selected; exact stored Region only, no municipality/population proxy |
| 5 | Age bucketing vs AnchorAge | NO ISSUE | Complete contiguous 21–75 mappings use frozen anchor age, never CurrentAge or ReferenceDate |
| 6 | Industry root vs Company independence | NO ISSUE | One root law has no geography, Person, count or domain-size dimension |
| 7 | Count architecture vs Industry→Count | NO ISSUE | Same-Company Industry indexes count points; no reverse or joint hidden state |
| 8 | Count internal segmentation vs hidden state | NO ISSUE | No segmentation selected; ephemeral point index discarded |
| 9 | Count zero vs Employer eligibility | NO ISSUE | Existing zero-count Company remains operational and excluded from all employee selectors |
| 10 | SizeBand derived-only | NO ISSUE | Exact frozen mapping follows count without RNG, feedback or another model |
| 11 | Education architecture vs parents | NO ISSUE | Own age bucket and exact Region only; no activity/enrollment input |
| 12 | Activity architecture vs six states | NO ISSUE | One six-entry categorical law, no overlapping flags/latent |
| 13 | Employee family vs no direct age | NO ISSUE | Exact attainment/Region grid only; mediated age associations remain possible |
| 14 | Employee level vs no Employer input | NO ISSUE | Own age/attainment/family grid generated before selection |
| 15 | SE family vs branch isolation | NO ISSUE | Separate data/identity despite identical axis shape |
| 16 | SE Industry vs family→Industry | NO ISSUE | Exact family-conditioned law; no direct Region/Company dimension |
| 17 | Employer factorization vs frozen information | NO ISSUE | Only family/level, count, Industry, exact band and dense candidate domain enter |
| 18 | Employer hard/soft support vs Plausible+ | NO ISSUE | Only mandatory zero-workforce exclusion; all positive candidates retain unusual combinations |
| 19 | Workforce factor vs Count authorization | NO ISSUE | Unchanged authorized min(count,C), fitted C in safe 1..65,535 envelope; old 4,095 ceiling removed without any parent, factorization or index change |
| 20 | Index count vs 1,024 limit | NO ISSUE | Exactly 40 indexes, well below frozen maximum |
| 21 | Employer memory vs Contract-2 budgets | NO ISSUE | Exact Nmax from input/index budgets, serial phase proof and all additional-state terms retained |
| 22 | CompanyIndex economic exclusion | NO ISSUE | Coordinate supplies candidate key/order only; equal facts imply equal weight |
| 23 | Income vs whole EUR | NO ISSUE | D2-B delivers valid whole-EUR supports after exact required ceiling; no fractional runtime law |
| 24 | Income zero vs zero semantics | NO ISSUE | Same-law zero point means no Income, not absent branch/source failure |
| 25 | Income pooling vs parent sets | NO ISSUE | Complete exact/bucketed authorized tuple address maps to law; no added or finer input; equal laws do not erase semantic authorization |
| 26 | Income sampling state vs hidden latent | NO ISSUE | Ephemeral support position cannot be retained/shared or used as a source/affluence class |
| 27 | Income branch isolation | NO ISSUE | Six branch-local maps/laws/age identities; no cross-branch law registry identity |
| 28 | Source compilation vs public semantics | DEFERRED CLEANLY | Architecture requires final compatible supports; actual source reconciliation, caps and fit are D2-B evidence |
| 29 | Constructor compilation vs random access | NO ISSUE | Complete immutable models/indexes precede calls; no history or shared advancing RNG |
| 30 | No fallback vs error semantics | NO ISSUE | Invalid fixed coverage rejects constructor; detected runtime invariant failure returns error, never partial success |
| 31 | Shared assets vs selective identity | NO ISSUE | Only intentional common frozen semantics shared; physical storage keeps local numerical identities |
| 32 | Resource maxima vs implementation | NO ISSUE | Income ≤512 pairs/points per law with separate authored input/decode and complete canonical R_out/phase preflight; all existing byte caps unchanged; deduplication cannot bypass authored limits and simultaneous 512-everywhere canonical volume rejects |
| 33 | D2-B freedom vs architecture | NO ISSUE | Fitted C may reach 65,535; high-volume Income laws may use 512 points with lower-resolution laws under unchanged aggregate byte policy. Actual values/cardinalities/fit remain D2-B-owned, including discretization evidence; approved architecture unchanged |
| 34 | Protocol freedom vs D2-A | DEFERRED CLEANLY | Logical order/laws/indexes fixed; byte encoding, streams and exact new sampling remain Protocol-owned; Contract-2 selector unchanged |
| 35 | Layer-3 exclusion | NO ISSUE | Downstream attachment only; no Layer-3 parent, fit feedback or direct access to sample-internal states |

All 35 cases were re-evaluated after F1/F2 rectification; no finalized case is REQUIRES D2-A FIX. Workforce Count authorization and D2-B architectural freedom now classify NO ISSUE based on the wider fitted-C envelope, exact arithmetic, usable 512-point law capacity and byte-governed aggregate evidence. Other classifications and approved semantics are unchanged; only directly affected capacity explanations were synchronized. Missing empirical contents remain D2-B work. None requires changing Contracts 1–5, and revised bytes still require independent re-review.

### 28. Independent-review questions

1. Is D2-A architectural rather than empirical?
2. Are categorical weights exact and deterministic?
3. Is proportional canonicalization correct, including the Contract-2 Employer exception?
4. Are category orders canonical and independent of source/map order?
5. Are bucket/pool rules closed enough for D2-B?
6. Does any bucket add unauthorized information?
7. Is exact Region treatment safe and adequately sourceable?
8. Is EmployeeCount heavy-tail-capable without sampled SizeBand?
9. Are the unchanged EmployeeCount 128-point and uint32 support bounds practical?
10. Are Education/activity/family/level table dimensions sourceable through broad evidence and documented assumptions?
11. Is the chosen Employer factorization sufficient for the frozen effects?
12. Are Employer combinations soft enough to retain unusual valid profiles?
13. Is Employer weighting integer-safe at C=65,535 and I/S=4,095, with exact magnitude semantics preserved and saturation left to fitted C?
14. Are forty direct indexes and the N=104,855 resource ceiling feasible?
15. Is CompanyIndex economically inert while retaining frozen canonical interval order?
16. Can one heavily populated Income law use 512 points for zero, central-body resolution, skew and tails without interpolation/jitter, while actual aggregate bytes remain within approved caps?
17. Does whole-EUR runtime representation preserve Contract-5 ceiling and strict zero semantics?
18. Does full-tuple Income pooling preserve exact semantic-parent boundaries?
19. Are all six Income branch models independently identifiable?
20. Is hidden socioeconomic mixture/source state excluded?
21. Is runtime empirical fallback eliminated?
22. Is constructor validation complete, including structurally reachable zero-probability cells?
23. Are model assets logically decomposed for non-interference?
24. Are per-law authored/canonical caps and aggregate input/decode/retained/live-phase charges enforceable before excessive allocation, including the demonstrated near-boundary rejection?
25. Is D2-B fitted-C and per-law resolution-allocation freedom sufficient for fitting while preventing architectural redesign and requiring actual discretization evidence?
26. Have exact new deterministic sampling and identity mechanics remained Protocol-owned?
27. Can D2-B populate the architecture with plausible data without impossible sparse cross-products?
28. Can independent D2-B implementations produce canonically reviewable assets under these rules?
29. Is every retained component necessary for frozen semantics, Plausible+, sourceability, determinism or safety?
30. Have both HIGH findings been closed sufficiently for independent re-review without reopening Contracts 1–5, with v0 remaining unfrozen until that review succeeds?

### 29. D2-A v0 verdict

This specification selects exactly one architecture for each frozen model: canonical integer categorical vectors; Industry-conditioned finite count points; complete bounded age-conditioned categorical grids; forty direct Employer indexes with a positive three-factor count/Industry/size law; and six separately identified bounded maps to finite whole-EUR Income point laws.

Architecture is closed while empirical contents remain D2-B-owned. No unauthorized semantic parent, sampled SizeBand, hidden shared socioeconomic state, extra Income source state, runtime empirical fallback or premature Protocol byte/sampling choice remains. Employer feasibility is demonstrated structurally within frozen budgets; Income supports zero, skew and bounded tails. Required constructor failure, random access, identity isolation and before-allocation resource checks are explicit.

The two independent-review HIGH findings are rectified by only their capacity-envelope amendments and direct mechanical synchronization: F1 widens fitted C to 1..65,535 while I/S remain 1..4,095; F2 permits ≤512 authored pairs/canonical points per Income law under unchanged law-count and whole-layer byte caps. Exact arithmetic, the 400-bin body/zero/tail capacity witness, usable high-resolution allocation and near-limit resource rejection supply closure evidence. The following finding-closure and Astra-approval regression tables record the mechanical audit. No other approved architecture was reopened. D2-A remains an unfrozen v0 candidate ready for independent GPT-6 Astra xhigh re-review; no approval of revised bytes is claimed. D2-B fitting and Deterministic Protocol completion remain prerequisites to implementation.

**Finding closure after mechanical consistency verification; independent re-review pending:**

| Finding | Original problem | Exact amendment | Why it closes finding | New regression? |
| --- | --- | --- | --- | --- |
| F1 | `C<=4095` prematurely forces workforce saturation | uint16 fitted C in 1..65,535; unchanged I/S in 1..4,095 and min(count,C) | Counts above 4,095 remain distinguishable when actual C permits; Wmax=1,098,958,053,375 and total≤115,231,246,686,635,625 fit uint64; membership/index bytes unchanged | NO |
| F2 | Income support `<=128` limits a fixed conditional law to 128 values | Separately ≤512 authored pairs/canonical positive points per law; unchanged law-count maxima; aggregate authored input/decode and canonical retained byte preflight replace universal-128 volume limits | One law can cover the illustrative 400-bin body plus zero/upper/high tails; a 512-point law with lower-resolution laws fits R_out, while simultaneous 512-everywhere canonical volume and boundary excess reject under unchanged caps | NO |

**Regression audit against every Astra-approved area supplied for this rectification:** YES records preservation of the approved architecture, not Astra approval of revised bytes.

| Astra-approved area | Preserved? | Evidence |
| --- | ---: | --- |
| 15-model inventory | YES | §17 retains exactly the same fifteen IDs, outputs and semantic-parent/address columns |
| uint32 categorical weights | YES | §4 categorical primitive unchanged |
| ordinary-law GCD canonicalization | YES | §5 exact canonicalization unchanged; admissible support capacity differs only for Income |
| Employer exact-weight / no-normalization exception | YES | §5 unchanged exact interval semantics; §15 retains checked products/totals |
| canonical taxonomy ordering | YES | §5 unchanged displayed Contract-3 order and explicit Region labels |
| exact Region architecture | YES | §6 unchanged exact twenty-value dimensions; no new pool/geography |
| age-map architecture | YES | §6 unchanged eight consumer-local contiguous maps |
| age maxima 6/8 | YES | §6 unchanged ceilings and complete 21–75 coverage |
| Company Industry architecture | YES | §7 unchanged root fourteen-entry law |
| EmployeeCount finite-support architecture | YES | §8 unchanged exact Industry→point laws |
| EmployeeCount 128-point cap | YES | §8 unchanged authored/canonical 128 per Industry; §21 unchanged 1,792-point family reservations |
| Education architecture | YES | §9 unchanged bounded age-bucket×Region seven-state grid |
| PrimaryActivity architecture | YES | §10 unchanged single six-state age/education/Region law |
| Employee JobFamily architecture | YES | §11 unchanged Education×Region ten-state grid |
| Employee JobLevel architecture | YES | §12 unchanged age/education/family four-state grid |
| SelfEmployed JobFamily architecture | YES | §13 unchanged separate branch law |
| SelfEmployed Industry architecture | YES | §14 unchanged family→Industry law |
| Employer positive-support policy | YES | §15 eligibility unchanged: zero-count excluded, every positive count positive in all forty conditions |
| Employer separable factorization | YES | §15 unchanged workforce×family/Industry×level/band product; no interactions added |
| Employer I/S factor maximum 4095 | YES | §§4/15/18/26 keep I/S uint16 in 1..4,095; only C envelope changes |
| min(n,C) functional form | YES | §15 formula unchanged; actual fitted C now has a wider safe envelope |
| Employer dynamic range apart from C ceiling | YES | §15 exact positive I/S factors unchanged; no extra factor/parameter/transform |
| 40 direct Employer indexes | YES | §15 routing/index subsection unchanged |
| CompanyCount <= 104,855 | YES | §15 exact CompanyCount ceiling subsection unchanged |
| Employer resource formulas/index mathematics | YES | §§15/21 retain K=40, P=40q, A=40N, M=N, 16-byte entries and all frozen charges; C still occupies two bytes |
| all-zero-workforce constructor behavior | YES | §15 unchanged failure without retry, repair, forced-positive Company or occupancy state |
| Income finite-support distribution family | YES | §§4/16 retain one finite whole-EUR support/weight law; only capacity changes |
| Income law-count caps 512/256/64/64/64/64 | YES | §16 unchanged authored-definition and canonical-law maxima |
| Income tuple→law pooling | YES | §16 unchanged complete branch-local authorized tuples and zero-band sentinel slice |
| Income whole-EUR compilation | YES | §16 retains offline exact monetary reconciliation; no fractional runtime law |
| Income zero as ordinary support point | YES | §16 same-law zero; no zero flag/draw/source state |
| hidden-state exclusion | YES | §§8/16/20 retain ephemeral unshared sample positions and no socioeconomic latent |
| runtime no-fallback | YES | §§19/26 retain complete construction and failure on missing/malformed assets |
| constructor lifecycle | YES | §19 retains validate/canonicalize/compile before usable immutable world; only capacity/resource checks synchronized |
| model non-interference | YES | §20 unchanged selective node/branch identity and fixed-parent invariance |
| logical identity decomposition | YES | §20 unchanged logical units and physical-file isolation; §21 resource-only invariance retained |
| whole-layer resource caps | YES | §21 unchanged E/D/R_out 8 MiB each, joint retained80/peak128 MiB, H2≤1 MiB and Contract-2 budgets |
| runtime complexity | YES | §§21/22 same bounded lookup/support sampling and prebuilt O(log q) Employer; support bound increases to finite 512 only for Income |
| D2-B sourceability | YES | Existing broad-source/offline fitting approach and tuple-law sharing unchanged; §23 adds capacity-allocation/discretization evidence only |
| Deterministic Protocol boundary | YES | §22 retains all existing text and owns streams/bytes/draw/CDF/search realization; no new sampler, interpolation or jitter |

## Source-backed risks

These are risks of fitting or implementation under the inspected frozen contracts, not claims from newly researched empirical sources. Actual source-backed numerical conclusions belong to D2-B.

| Risk | Architectural response and evidence |
| --- | --- |
| Empirical sparsity | Complete offline fitting and documented assumptions are required; no constructor/runtime inference. Closed parent sets cannot be expanded to improve sparse fits (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:113`). |
| Huge conditional cross-products | Small categorical grids remain explicit; Income unique laws bounded at 512/256/64 per branch with compact complete tuple maps. Actual law-resource evidence remains necessary (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:722`). |
| Overfitted age boundaries | Local contiguous maps have small fixed ceilings; D2-B must justify actual boundaries and sensitivity, not optimize arbitrary per-age cuts. Frozen AnchorAge has only the prescribed 55 values (`tools/fakedata/layer1_data.go:135`). |
| Region pooling leakage | No Region pooling or finer geography is authorized. Acquisition projects exact Region only; finer private place fields must not reach numerical addresses (`tools/fakedata/layer1_data.go:87`; `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:154`). |
| Zero weight mistaken for semantic invalidity | Zeroes remain model support decisions; D2-B documents apparent exclusions and future positivity needs no taxonomy reopening (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:611`; `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:360`). |
| EmployeeCount heavy-tail truncation | Finite uint32 points allow tail shape but impose deliberate finite production support. D2-B documents actual cap/tail effects; public representability is not an empirical claim (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:253`). |
| Hidden SizeBand sampling | Only count is drawn; public band is the exact frozen consequence. Support indices cannot escape to another law (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:300`). |
| Employer factor overflow | Resource ranges prove valid products/totals fit; checked widening and range validation still required; no normalization/overflow repair (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:494`). |
| Employer index memory | Forty legal conditions do not alone establish memory feasibility. Nmax and raw zero-entry volume are explicitly proven; every buffer phase must be accounted (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:681`; `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:631`). |
| Employer hard-zero overuse | Positive C/I/S and every positive-count candidate preserve uncommon function/sector and micro executive profiles; no additional hard zero exists (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:632`). |
| CompanyIndex leakage | Candidate order is canonical protocol, never economic weighting. IDs and coordinate-derived intercepts are excluded (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:286`). |
| Income tail truncation | Finite point support is deliberately bounded; D2-B must document caps and high-segment consequences, not silently clip invalid draws (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:217`). |
| Income conditional discretization / aggregate capacity | A heavily used conditional law can now use 512 whole-EUR points, with actual cardinality/body/zero/tail and repetition/occupancy/spacing evidence owned by D2-B. Individual law maxima do not authorize simultaneous 512-everywhere canonical storage: complete input/decode/R_out preflight enforces unchanged bytes (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:722`). |
| Hidden Income source mixture | One aggregate law samples one Income; no source, benefit, wealth or business-size state is generated or retained (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:675`). |
| Branch identity coupling | Six independent maps/laws and five separate Income age definitions preserve branch isolation despite equal schemas (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:638`). |
| Shared-file fingerprint coupling | Canonical logical section identities replace whole-file stochastic dependencies; physical provenance checksums are audit-only (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:625`). |
| Source-format leakage | Constructor sees bounded fitted numerical inputs. Source spelling, histories, salary-only/household measures and fractional conventions cannot silently redefine runtime semantics (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:704`). |
| Runtime fallback creep | Missing coverage fails before world escape; no national fallback, zero substitution or inapplicable branch evaluation (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md:714`). |
| Incomplete resource accounting | Encoded, decoded, retained, identity, raw input and scratch lifetimes all have caps; allocations preflight phase coexistence. Contractual charges are not RSS and do not excuse uncharged extras (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:525`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:631`). |

Additional fitting limitations remain explicit: capped workforce prevalence cannot fit arbitrary within-tail company effects, but its fitted saturation is no longer forced by 4,095; 512 employee Income laws cannot reproduce every five-way population distribution; zero-workforce realization can fail construction; finite support points still create repeated numerical values, so D2-B must justify actual high-volume-law resolution up to 512 using the approved aggregate bytes. D2-B evaluates these against Plausible+, not an unsupported representativeness standard. None authorizes a new parent, latent, index architecture or runtime distribution family.

## Deferred decisions

| Deferred matter | Owner | Closed boundary |
| --- | --- | --- |
| Actual categorical weights/probabilities | D2-B | Prescribed canonical grids, uint32/GCD laws |
| Actual CompanyCount | D2-B | One literal positive N≤104,855; no runtime config or seed-derived count |
| Actual EmployeeCount support/weights/model cap | D2-B | Fourteen finite count laws, ≤128 authored/canonical points each, uint32 domain |
| Actual Employer C/I/S values | D2-B | Unchanged single capped raw-count transform and fixed two tables; uint16 I/S in 1..4,095, fitted C in 1..65,535; no final normalization |
| Actual Income supports/weights/cardinalities/zeros/tails/model caps | D2-B | Whole-EUR finite laws; unchanged law-count maxima, separately ≤512 authored pairs/canonical points per law and aggregate input/decode/retained byte preflight; actual high-volume discretization evidence; no source mixture or clipping |
| Eight actual age bucket counts/boundaries | D2-B | Consumer-local contiguous complete 21–75 intervals within fixed ceilings |
| Actual Income tuple-pool assignments | D2-B | Complete authorized tuple maps, local canonical IDs and fixed law maxima; no per-axis/finer Region map |
| Datasets, crosswalk contents, provenance, fitted parameters/transformation artifacts and Plausible+ fit | D2-B | Offline evidence produces this fixed numerical architecture and frozen meanings |
| Concrete byte totals, phase schedule evidence and derived Employer index manifests | D2-B / future implementation evidence | All fixed component/Contract-2/joint budgets and work bounds |
| Streams, Components, framing, seeds and selective model fingerprints | Deterministic Protocol | Closed parents, separate innovations, canonical equivalence and non-interference |
| Exact categorical/support sampling, bounded-draw and CDF/search realization | Deterministic Protocol | Canonical finite integer laws; Employer mechanics already frozen by Contract 2 |
| Canonical binary identity serialization and SocioeconomicWorldID encoding | Deterministic Protocol | Complete logical identity coverage without global stochastic coupling |
| Layer-3 facts and consumption | Future Layer-3 contracts | Downstream only; no Layer-2 feedback or direct sample-internal state |

No numerical family, category order, parent boundary, bucket family, nominal per-axis pooling rule, Employer factorization/routing/index count, Income pooling architecture, constructor lifecycle, fallback policy or resource maximum remains open. Missing sources or fitted values are D2-B work, not blockers.

## Repository-integrity proof

The following initial-delivery and earlier correction records are historical. The current F1/F2 rectification evidence is recorded separately at the end of this section; earlier arithmetic/count claims do not describe the revised capacity maxima.

The baseline and all five frozen-contract checksums in section 1 matched before analysis. Complete before captures under `/tmp` included branch, HEAD, full unfiltered `git status --porcelain=v1 -uall`, cached binary diff, raw `.git/index` SHA-256, the five contract SHA-256 values, and `tools/types/semantic.go` SHA-256 and working-tree diff. Initial `git diff --check` and `git diff --cached --quiet` passed.

The required offline repository sanity command passed with exit status zero:

```bash
GOCACHE=/tmp/krenalis-l2-d2a-gocache \
GONOSUMDB='*' \
GOPROXY=off \
  go test ./tools/fakedata
```

Actual result:

```text
ok  github.com/krenalis/krenalis/tools/fakedata  13.775s
```

This tests the existing package only. It supplies no Layer-2 implementation or numerical-fit evidence. No generator, network, web or external research was used.

Before delivery, independent arithmetic checks recalculated every categorical/support resource-table charge, the exact Employer N ceiling, maximum product/total, retained charge and peak witness. Mechanical consistency checks confirmed 29 numbered sections, exactly 15 model-inventory rows, 28 adversarial cases, 35 cross-contract audit cases, 30 independent-review questions, all 67 distinct repository citations resolving to existing lines, and the exact final marker. The complete draft was reviewed for agreement between tables and ledger, parent boundaries, canonical equivalence, resource ownership and deferred responsibilities.

Under the user's subsequent explicit instruction, the completed specification was placed in one new repository-root file:

```text
SYNTHETIC_WORLD_V1_LAYER2_D2A_v0.md
```

This is the only authorized repository addition. It is a specification artifact, not Layer-2 implementation. No pre-existing repository file was edited, renamed, formatted, deleted, staged or committed.

After-state equivalents of every before capture were made and compared. Branch, HEAD, cached binary diff, raw Git index hash, all five contract hashes and semantic.go hash/diff compared byte-for-byte unchanged. Full worktree status differs by exactly the one new untracked specification entry; removing that exact entry from the after capture reproduces the complete before status byte-for-byte. All pre-existing tracked and untracked entries, including the large pre-existing untracked cache/output trees, remain present and unchanged in status. No unrelated files were cleaned.

The before-status SHA-256, also obtained from the after status after removing only the authorized new entry, is:

```text
4a13d0fb9df02d2c6798a6d17e06651c2533e69c68fbfa0f41f64748475e6f3a
```

Preserved raw Git index SHA-256:

```text
9bf5ddbc729f55523ac70792666243018b250ae81053a8e4d741071e003d81c3
```

Preserved semantic.go SHA-256:

```text
06a7ff690b3e12cdaf623e3ffb36003d1675a6aec69b3ee59e9ead428cc17842
```

Final branch remains `property-semantics`; HEAD remains `76fa0fff109ca6abad62f55c8761275455842dc5`, subject `Add Synthetic World Layer 1`. Final `git diff --check` and `git diff --cached --quiet` passed. Every task-created `/tmp` analysis/integrity artifact and the temporary Go cache was removed. Nothing was staged or committed.

The original demand for a byte-identical full worktree status is superseded solely by the later authorization to create this document. All remaining integrity requirements are preserved; the single-file status delta is reported explicitly rather than described as an unchanged worktree.

**Subsequent authorized correction and completed review:** only this specification was revised. Income limits now separately constrain authored definitions/pairs before deduplication and canonical laws/points afterward, with unchanged numerical byte ceilings. The asset inventory, constructor rules, resource formulas, D2-B obligations, future verification and decision ledger were aligned. The uint16 support-point cardinality is explicitly per law. Structural/table/citation checks and independent resource arithmetic passed; the review found no additional substantive frozen-contract conflict. These are specification checks, not implemented Layer-2 tests. During this revision, before/after branch, HEAD, full worktree status, tracked working-tree binary diff, cached binary diff, raw Git index hash, Contracts 1–5 hashes and semantic.go hash were unchanged. The specification was already untracked at revision start, so its authorized content change does not add a status entry; a temporary before snapshot and content diff established the document change. Final whitespace checks passed, including the changed untracked document. All revision-created `/tmp` snapshots and verification artifacts were removed. No implementation, generator, network, staging or commit occurred; the initial offline Go-test result above remains historical and was not rerun for this document-only correction.

**Current F1/F2 surgical rectification evidence:** the starting target SHA-256 was ba3b3101aa678723781f6a6dbc268992391cd5a2dc5933436fa5ea7f4b3af209 and all five frozen-authority hashes matched section 1. Before editing, the target contents were explicitly copied to /tmp/krenalis-l2-d2a-rectification.before.md; target hash, branch, HEAD, full status, tracked/cached binary diffs, raw index, semantic.go and Contracts 1–5 captures were preserved under /tmp. Exact integer scripts independently recalculated the wider-C weight/total, unchanged index/peak formulas, every density stress charge, both allocation witnesses and the 512 unique whole-EUR witness points. The old-vs-new content diff and occurrence classification distinguished unchanged I/S/EmployeeCount/128 MiB limits from corrected or historical Income/C references. Checks confirmed 29 numbered sections, 15 model rows with unchanged semantic-parent/address columns, 38 adversarial cases, all 35 cross-contract cases, 30 review questions, consistent tables/ledger, 40 preserved Astra-approved areas and 67 resolving repository-relative citations. Approved §§5–14/20, Employer eligibility/routing/N-ceiling and Contract-2 construction-accounting text remained byte-identical where no synchronization was needed.

The required offline repository-sanity command passed with exit status zero using GOCACHE=/tmp/krenalis-l2-d2a-rectification-gocache, GONOSUMDB='*' and GOPROXY=off; GOTMPDIR/TMPDIR also pointed to the task's /tmp directory. Actual result: `ok github.com/krenalis/krenalis/tools/fakedata 13.419s`. No Layer-2 implementation or fit test is claimed. Final git diff --check, cached --quiet and untracked target/report whitespace checks passed. Before/after branch, HEAD, raw Git index, cached diff, semantic.go, Contracts 1–5 and all pre-existing tracked changes matched. The target's existing untracked status entry need not change when its contents change; the explicit before copy proves the intentional target delta. The user's later instruction additionally authorized one new root response report, SYNTHETIC_WORLD_V1_LAYER2_D2A_v0_RECTIFICATION_REPORT.md; removing only its new untracked entry from after-status reproduces before-status exactly. No unrelated file was edited or cleaned, and nothing was staged or committed. All current task-created /tmp snapshots, scripts, verification artifacts, Go cache and Go temporary directory were removed after evidence collection. The target's final checksum is reported in that separate response report, avoiding self-referential checksum content. Both HIGH findings are mechanically rectified; the candidate remains unfrozen pending independent re-review.

L2 D2-A v0 COMPLETE — READY FOR INDEPENDENT RE-REVIEW
