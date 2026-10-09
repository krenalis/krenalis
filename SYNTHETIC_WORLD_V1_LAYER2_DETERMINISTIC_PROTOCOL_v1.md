# Normative deterministic protocol

## Synthetic World v1 — Layer 2 deterministic protocol v1

**STATUS: FROZEN — independently reviewed and manually approved.**

This document defines the approved runtime compatibility choices. It does not approve implementation, change the prescribed socioeconomic model, or certify a generated production world. “SHALL” below is approved normative wording. Statements marked **INHERITED** restate existing authority; statements marked **NEW** select previously deferred protocol behavior.

## 1. Scope, authority and notation

**INHERITED.** Contracts 1–5 and D2-A, in their supplied frozen bytes, control public types, ownership, the dependency graph, applicability, anchor semantics, numerical architecture, resource limits and non-interference. Their historical proposal headings do not demote their supplied authority. Within D2-B-owned scope, B1 v3 explicit wording takes precedence over v2 Income generation, v1 quality policy and imported v0 generator wording. The publication contract owns publication and evidence, not generation streams.

**AMENDED LIFECYCLE (FROZEN).** The [external model loading amendment](SYNTHETIC_WORLD_V1_LAYER2_EXTERNAL_MODEL_LOADING_AMENDMENT_v1.md) §2 controls only the identified acquisition, ownership, construction-transfer and phase-accounting clauses. All other frozen requirements remain controlling. That amendment owns transport/acquisition policy and the explicit model boundary; this protocol consumes its successfully acquired immutable model.

This protocol completes only deterministic addressing, runtime fingerprints, bounded sampling outside the already-frozen Employer selector, and SocioeconomicWorldID bytes. The protocol itself introduces no public API, configuration argument, seed, entity, socioeconomic latent, distribution family, fitting operation, publication artifact requirement, or statistical acceptance gate. Source-observation / SourceWorld protocols and future CountryPack mechanisms are not Layer 2 runtime authority.

Controlling references: C1 §§6–16; C2 §§14–18; C4 §§5–20,22–29; C5 §§8–16; D2-A §§2,4–6,15–22; External Model Loading Amendment §§2–7; B0 §32; B3 §§2,4,5.1; Publication §18. The unchanged audit records the original input filenames and hashes.

Notation: `S(name,value)` is the existing StringComponent; `U(name,value)` is Uint64Component; `B(name,bytes)` is BytesComponent. `K(domain,fields)` is the existing canonical KFD1 record. `H(domain,fields) = SHA-256(K(domain,fields))`. Digests in B components are the raw 32 bytes, never the ASCII hexadecimal display. `C(x)` is the inherited canonical logical JSON serialization of logical value x, including exactly one terminal LF. These symbols are specification notation, not new public functions.

## 2. Stable protocol identity and versioning

**NEW.** The stable protocol identity is the exact ASCII string:

```text
socioeconomic-v1
```

The stochastic path prefix is exactly `socioeconomic/v1/`. A path identifies a semantic node, not a Go declaration, working-stage document, publication report or model download location. Its v1 semantics incorporate the frozen taxonomy meanings, SizeBand mapping, Income meaning, numerical addressing order, and the rules below.

A change to a byte domain, KFD1 field set/type/encoding, path, entity kind or coordinate meaning, dependency projection, model-fingerprint semantics, deterministic source, bounded-draw realization, sample boundary, semantic draw count, or world-ID representation requires a reviewed protocol version. Reusing a version with changed compatibility behavior is forbidden.

A change only to canonical numerical laws, factors, CompanyCount, local age maps or tuple assignments within the same approved architecture is a model revision, not inherently a protocol revision: the corresponding content identities change under this protocol. Such revisions are not caller-configurable in v1. Source/provenance/report/code changes that preserve logical model and protocol behavior do not change runtime identity or outputs.

A change to a node's outcome meaning or deterministic model semantics requires a reviewed change to that node's versioned path/domain, not only a changed global report label. A future local semantic/protocol revision must preserve unaffected node addresses and identities when the frozen non-interference contract applies. Changing a global revision label is not permission to replace every node's innovation. A future specification must identify affected nodes explicitly; this document neither defines nor authorizes that future model.

## 3. Inherited KFD1 byte contract

**INHERITED / ADOPTED.** Reuse `tools/fakedata/component.go:101–162`; no alternative framing format is introduced. All integers below are unsigned, big-endian, fixed width:

```text
ASCII "KFD1"                                      4 bytes
uint32 domain UTF-8 byte length                   4 bytes
domain bytes                                     stated length
uint32 number of components                      4 bytes
for each component, sorted by its ASCII name:
    uint32 component-name byte length            4 bytes
    component-name bytes                         stated length
    kind                                         1 byte
    uint64 value-byte length                     8 bytes
    value bytes                                  stated length
```

Kinds: string `0x01`, uint64 `0x02`, bytes `0x03`. A U value always has an eight-byte payload, including zero and enum ordinals; no varint or native-endian encoding is used. Strings are exact valid UTF-8, without BOM, case folding, normalization, trimming, termination or implicit length units other than bytes. Bytes are uninterpreted. Empty string and empty bytes have length zero but different kind tags. Nil/empty byte slices have the same byte representation. An omitted component is distinct from a present empty component. No omitted optional component is silently replaced by null, zero or empty bytes.

Names follow the existing lowercase-identifier grammar, maximum 64 bytes; duplicates are rejected. Domains obey the existing domain grammar and 128-byte maximum. Caller ordering never determines record ordering. All lengths, counts and allocation arithmetic are validated before use. There is no new native uint8/uint16/uint32 KFD kind: those model widths are widened exactly to U when they appear as scalar dependencies.

Nested model content is B containing exactly C(x); nested dependency content in the stream record is the raw SHA-256 digest of its KFD1 dependency record. Do not substitute JSON for a KFD1 record, encode a digest twice, or treat a hexadecimal string as raw digest bytes. The complete field sets below are closed.

## 4. Canonical model content and model fingerprints

### 4.1 Canonical logical content

**INHERITED.** The principal logical model has schema `SW-L2-D2B2-logical-v0`, with precisely the fields in B0 §32. Canonical category vectors retain their zero slots and are GCD-normalized. Numerical laws have ascending unique uint32 supports and aligned positive uint32 weights after prescribed sort/merge/zero omission/GCD. Income law IDs are branch-local consecutive IDs by first occurrence of distinct canonical law content in canonical tuple traversal. Law labels and duplicate authored definitions do not survive as independent identity material.

C(x) uses compact UTF-8 JSON with object keys in Unicode-code-point order, no BOM, no spaces between tokens, literal non-ASCII scalar characters and exactly one LF. It is the serialization equivalent of `json.dumps(x, sort_keys=True, ensure_ascii=False, separators=(',',':')) + '\n'`. The logical model's values are objects, arrays, strings and exact nonnegative integers, not floating point, booleans, nulls or surrogate code points. String escapes follow the inherited JSON convention: quote and backslash are escaped; standard short escapes represent backspace, tab, LF, form feed and CR; other U+0000..U+001F use lowercase `\u00xx`; other scalar values, including `/`, remain literal. There is no normalization of strings. For these logical subtrees, all field names and scalar strings are already prescribed.

This protocol consumes the canonical, validated, compiled immutable model acquired under the loading amendment §4A, which defines runtime input admission and its distinction from producer canonicalization. Identity is computed from the resulting canonical logical values, not the incoming field order, whitespace, compressed file, producer code or transport representation. Compiled cumulative differences recover the canonical individual weights exactly; age lower bounds are recoverable from a valid compiled age map. No producer arithmetic is rerun to reconstruct these logical values.

### 4.2 Node-local fingerprints

**NEW.** For a registry node n with exact full path P, let A be exactly its listed logical asset subtree. Define:

```text
fingerprint domain = "fakedata/" + P + "/model"
fields = { B("asset", C(A)) }
```

Only when that registry row names an age map, add exactly:

```text
B("age-map", C(L.ages[the named consumer map]))
```

The age object contains both `lowerBounds` and `compiled`; their equality under the frozen interval rule is validated, not independently chosen. The fields sort `age-map` before `asset` when both exist. The node fingerprint M_n is H of this record. Each path's domain qualifies its model identity; even identical content in two branches remains separately owned. No whole-model checksum, upstream model fingerprint, publication audit hash or unrelated consumer's age map is added.

The 15 fingerprints are computed once from immutable canonical model state and may be retained as 15 raw 32-byte values. The registry's local asset subtree includes every conditional row, or its full Income branch map and law pool, not merely the row chosen for one Person. Per-row/per-law fingerprint granularity is not required. Law IDs are internal table addresses, not additional random dependencies.

### 4.3 Whole logical identity

**NEW BINDING; EXISTING DIGEST DEFINITION.** Let L be the complete canonical logical model. Set:

```text
M_all = SHA-256(C(L))
```

This reuses the publication contract's existing logical-model digest definition, expressly for the world-ID binding in §5. It is not the publication bundle digest, a new release checksum or a stream seed. It includes CompanyCount, all numerical units, maps and factors. The protocol identity covers the common semantic specification and all runtime realization choices. Together they cover the output-affecting configuration required by C1 §15 and D2-A §20.

No separate release-name hash, source-manifest hash, audit envelope, whole-model runtime binary serialization or aggregate tree of the 15 fingerprints is needed. A release label alone is insufficient to distinguish changed canonical contents. Reusing the whole logical digest in each stream is forbidden. Publication's 24 per-model audit hashes remain audit identities, distinct from the 15 runtime identities above.

Identity equality here is hash-based compatibility, not a mathematical claim that a finite SHA-256 digest is collision-free. KFD1's typed record encoding is unambiguous; hash collision resistance is the inherited engineering assumption.

## 5. SocioeconomicWorldID

**INHERITED.** Bind the actual stored base World's stable WorldID. Do not reconstruct a new WorldID from copied configuration. SnapshotID, ReferenceDate, FaceCatalog and PhotoID are excluded. The empty socioeconomic ID is invalid for a constructed wrapper.

**NEW.** Decode the actual canonical `world_v1_` WorldID using the frozen `worldIDDigest` semantics into its raw 32-byte digest W. This operation preserves the identity of the actual supplied World and does not recompute it. The approved exact world identity is:

```text
domain: fakedata/socioeconomic-world-id/v1
fields, in KFD1's resulting name order:
    B("model", M_all)
    S("population-anchor-date", "2026-01-01")
    S("protocol", "socioeconomic-v1")
    B("world-id", W)

SocioeconomicWorldID =
    "socioeconomic-world_v1_" + lowercase_hex(SHA-256(the KFD1 record))
```

The world-ID domain version labels the identity encoding; the `protocol` field identifies the complete generation contract. A future sampler revision can therefore change the protocol identity without inventing a new world-ID codec.

The suffix is exactly 64 lowercase hexadecimal characters, with no whitespace, braces, padding, separator beyond the literal prefix, or LF. No public composer/parser is added. A malformed base identity fails construction rather than generating an empty or repaired ID.

Equal actual base WorldID, canonical model and protocol produce the same socioeconomic ID. A ReferenceDate-only or photo-only snapshot variation leaves it unchanged. Changed output-affecting model content changes its covered input; a different HTTP origin, URL, compression, cache, restart state, report path, report prose or producer formatting does not. The world identity is never passed to any stochastic node.

## 6. Root derivation and exact entity addresses

**NEW LAYER-2 BINDING; REUSED FOUNDATION.** Every stochastic Layer 2 node uses the actual immutable `base.streams` factory unchanged. No socioeconomic stream factory, top-level seed, root salt, WorldID-derived seed or namespace-derived secondary seed is introduced. Node paths and dependency records supply domain separation.

The supplied Layer 1 constructs this factory from exactly:

```text
H("fakedata/stream-root/v1", {
    S("world-model-version", "world-v1"),
    U("world-seed", the base World's original seed)
})
```

That is an explanation of the frozen root, not authority to reconstruct a second base from independently supplied values. WorldID additionally binds identity/market/spec inputs; those extra inputs are not all present in the stream root. In particular, namespace-only changes may preserve stochastic facts while changing entity IDs and world identity, consistently with the existing foundation. Snapshot state is absent from this root.

| Entity | Exact stream scope/kind/coordinate | Operational validation | Identity |
|---|---|---|---|
| Socioeconomic Person | scope `entity`, kind `person`, uint64 of the requested PersonIndex, without decrement | 1..60,466,176; frozen Layer1MaxPersonIndex | Frozen Person ID from namespace and coordinate; the text ID is not a stochastic input |
| Company | scope `entity`, kind `company`, uint64 of the requested CompanyIndex, without decrement | 1..CompanyCount; current N=65,536 | Frozen Company ID from namespace and coordinate; no seed or model input |
| Socioeconomic World | No stochastic world-level draw in this protocol | Valid constructed base and complete prescribed model | §5 identity; not a company/person coordinate |

PersonIndex and CompanyIndex remain distinct semantic types. A protocol Company ID can encode any nonzero uint64 coordinate even above N, but such an ID does not prove operational membership. Generic foundation stream/ID fixtures above the Person operational maximum are not permission to request such a Person. Array offset i−1 is storage only, never the stream coordinate. PersonID, its encoded token, Company.ID, WorldID and SocioeconomicWorldID are not interchangeable with coordinates.

### Exact stream construction

For a node, derive a fresh stream using the foundation's existing `entityStream(kind, index, path, dependencies...)`. Its semantics, restated for independent implementations, are:

```text
D = H("fakedata/stream-dependencies/v1", dependencies)
T = H("fakedata/stream-seed/v1", {
    B("dependencies", D),
    U("entity-index", coordinate),
    S("entity-kind", kind),
    S("path", full node path),
    B("root", actual base stream root),
    S("scope", "entity")
})
initial SplitMix64 state = unsigned big-endian integer T[0:8]
```

There is no extra draw counter, PersonID, CompanyID, whole Person serialization or seed added by Layer 2. The dependency field set is exactly §8. A request derives a fresh value; it never advances a persistent node or global RNG. The foundation's empty-dependency digest is a hash of the empty KFD1 record, not 32 zero bytes; Layer 2 stochastic nodes always have at least their model component.

## 7. Semantic acquisition, exact enum ordinals and law addressing

**INHERITED.** Acquire AnchorAge from the separate age returned by frozen `World.birth`, after operational PersonIndex validation. Do not derive it from CurrentAge or forward BirthDate/birth year. Acquire Region by reproducing exactly the frozen Layer 1 address municipality-selection prefix, including its mixture decision and appropriate uniform/population municipality path, and project only the selected row's exact Region label. Do not draw a second municipality or use full Person/photo generation. The baseline municipality acquisition's own multi-draw mechanics remain unchanged; they are not Layer 2 socioeconomic innovations.

All socioeconomic facts and Company facts describe PopulationAnchorDate = 2026-01-01. Composite Layer 1 CurrentAge continues to use ReferenceDate. No timeline, age-forwarding or Income price-date selection is introduced.

**NEW ENCODING; INHERITED ORDERS.** All enum and Region dependencies are U components containing zero-based D2-A §5 ordinals. They are not declaration-order guesses, lexical ranks, iota values, source classification codes or string hashes. Exact canonical labels map to these lists, left to right:

| Domain | Canonical ordinal order |
|---|---|
| Education | below-primary; primary; lower-secondary; upper-secondary; post-secondary-non-tertiary; tertiary; doctoral |
| Industry | agriculture; production-and-utilities; construction; commerce; transport-and-logistics; hospitality; information-and-communication; finance-and-real-estate; professional-and-business-services; public-administration-and-safety; education; health-and-social-care; arts-and-recreation; other-services |
| JobFamily | production-and-fieldwork; skilled-trades; transport-and-logistics; customer-and-personal-service; administration; sales-and-commercial; technical-and-scientific; professional-and-creative; care-and-education; protective-services |
| JobLevel | individual-contributor; supervisor; manager; executive |
| PrimaryActivity | employee; self-employed; unemployed; student; retired; inactive |
| SizeBand | zero; micro; small; medium; large |
| Region | Abruzzo; Basilicata; Calabria; Campania; Emilia-Romagna; Friuli-Venezia Giulia; Lazio; Liguria; Lombardia; Marche; Molise; Piemonte; Puglia; Sardegna; Sicilia; Toscana; Trentino-Alto Adige/Südtirol; Umbria; Valle d'Aosta/Vallée d'Aoste; Veneto |

All unknown, empty, aliased or noncanonical labels are invalid. Source-crosswalk normalization is offline and must not create runtime Region aliases. Public enum values remain the frozen semantic strings. Widening an ordinal to U does not create a public numerical rank.

For each age-consuming node, use that consumer's own `compiled[AnchorAge-21]` bucket ordinal. **NEW:** frame this model-local bucket, not raw AnchorAge. This implements D2-A's BUCKETED addressing and avoids introducing within-bucket random-address distinctions beyond its numerical partition. The consumer's complete own age map is in M_n. The first lower bound is 21; boundaries belong to the interval starting there; the final interval ends at 75. No age map is inherited from another consumer merely because its contents currently match.

The current age lower bounds are inherited, not newly selected:

| Consumers | Lower bounds |
|---|---|
| Education; Employee.JobLevel; Income[s]; Income[u]; Income[i], each independently owned | 21,25,35,45,55,65 |
| PrimaryActivity | 21,25,35,45,55,60,65,70 |
| Income[t] | 21,25,30,40,55,65 |
| Income[r] | 21,45,55,60,65,70 |

For dimensions `(d0,...,dk)` of sizes `(s0,...,sk)`, the flat row/tuple offset is left-folded `offset = offset*sj + dj`, starting at zero. This implements the inherited first-dimension-outermost order. Every multiplication/addition and bound is established safely before indexing. A categorical row starts at `row*outputWidth`. Its cumulative weights are row-local; its last cumulative value is the row total. No total from a preceding row leaks in.

## 8. Single normative stochastic registry

**NEW paths, dependency bytes and non-Employer call realization; INHERITED parents, law families and traversal.** Every full path is `socioeconomic/v1/` plus the literal suffix below. Every stochastic row has exactly `B("model", M_n)` plus the listed U dependencies. The ordering shown for parent values is numerical tuple order; KFD1 itself sorts component names. No PrimaryActivity component is added inside a gated branch.

For every stochastic row, the primitive is the unchanged frozen SplitMix64 `Uint64n(total)` and the semantic call count is **ONE**. `CAT`, `POINT`, `EMP` denote §9 mappings; none introduces another sample. `A` means the current node's own age bucket, `E` Education, `R` Region, `F` applicable JobFamily, `J` employee JobLevel, `I` relevant Industry, `B` SizeBand. These abbreviations do not replace semantic parent definitions.

| Node / entity and gate | Authorized value parents / numerical tuple | Literal path suffix | Own logical asset; own age map | U dependency fields | Mapping → output |
|---|---|---|---|---|---|
| Company Industry / company j | None | company/industry | companyIndustry; none | none | CAT → Industry |
| Company EmployeeCount / same j | Same Company's Industry `(I)` | company/employee-count | employeeCount; none | industry=I | POINT → uint32 EmployeeCount |
| Education / person i, always | AnchorAge, Region `(A,R)` | person/education | education; Education | age-bucket=A, region=R | CAT → Education |
| PrimaryActivity / person i, always | AnchorAge, Education, Region `(A,E,R)` | person/primary-activity | primaryActivity; PrimaryActivity | age-bucket=A, education=E, region=R | CAT → PrimaryActivity |
| Employee JobFamily / person i, employee | Education, Region `(E,R)` | person/employee/job-family | employeeJobFamily; none | education=E, region=R | CAT → Employee.JobFamily |
| Employee JobLevel / same i, employee | AnchorAge, Education, Employee.JobFamily `(A,E,F)` | person/employee/job-level | employeeJobLevel; Employee.JobLevel | age-bucket=A, education=E, job-family=F | CAT → Employee.JobLevel |
| Employer / same i, employee | Person `(F,J)`; operational universe and Company count/Industry/SizeBand families via the immutable index | person/employee/employer | employer; none | job-family=F, job-level=J | EMP → CompanyIndex |
| SelfEmployed JobFamily / person i, self-employed | Education, Region `(E,R)` | person/self-employed/job-family | selfEmployedJobFamily; none | education=E, region=R | CAT → SelfEmployed.JobFamily |
| SelfEmployed Industry / same i, self-employed | SelfEmployed.JobFamily `(F)` | person/self-employed/industry | selfEmployedIndustry; none | job-family=F | CAT → SelfEmployed.Industry |
| Employee Income / person i, employee | Region, Employee.JobFamily, Employee.JobLevel, selected Industry, selected SizeBand `(R,F,J,I,B)`; Employer is lookup key only | person/income/employee | income.e; none | region=R, job-family=F, job-level=J, industry=I, size-band=B | POINT → Income |
| SelfEmployed Income / person i, self-employed | AnchorAge, Region, SelfEmployed.JobFamily, SelfEmployed.Industry `(A,R,F,I)` | person/income/self-employed | income.s; Income[s] | age-bucket=A, region=R, job-family=F, industry=I | POINT → Income |
| Unemployed Income / person i, unemployed | AnchorAge, Education, Region `(A,E,R)` | person/income/unemployed | income.u; Income[u] | age-bucket=A, education=E, region=R | POINT → Income |
| Student Income / person i, student | AnchorAge, Education, Region `(A,E,R)` | person/income/student | income.t; Income[t] | age-bucket=A, education=E, region=R | POINT → Income |
| Retired Income / person i, retired | AnchorAge, Education, Region `(A,E,R)` | person/income/retired | income.r; Income[r] | age-bucket=A, education=E, region=R | POINT → Income |
| Inactive Income / person i, inactive | AnchorAge, Education, Region `(A,E,R)` | person/income/inactive | income.i; Income[i] | age-bucket=A, education=E, region=R | POINT → Income |
| Company SizeBand / company j | Same Company's EmployeeCount | NO STREAM | Frozen deterministic mapping | NO DEPENDENCY RECORD | NO DRAW → SizeBand |
| Employee CurrentIndustry / person i, employee | Employer key; only Industry of selected Company | NO STREAM | Selected relational lookup | NO DEPENDENCY RECORD | NO DRAW → Company[Employer].Industry |

`income.e` etc. mean the exact existing `L.income["e"]` etc. subtrees, not new dotted JSON keys. These include full `laws` and `tupleMap`. Asset `employeeCount` includes all 14 Industry-addressed laws. Age-map names are exact existing logical names. The registry's model schema/dimensions determine row widths and tuple-map bounds; caller-supplied field sets are never accepted as an alternative registry.

Closed parent sets mean every unlisted fact is forbidden. Specifically: Company nodes take no Person Region/age/state; employee JobFamily takes no direct age; JobLevel takes no Region or Company; Employer takes no direct age/Education/Region/Company identity; self-employment takes no Company/Employer/JobLevel; employee Income takes no direct age/Education/raw count/Employer key or Company identity in its innovation. Gender, current age, ReferenceDate, SnapshotID, name, full address, finer geography, email, phone, photo and arbitrary Person state are excluded throughout. Node-local tables are model data, not additional public-fact parents.

Company-family data need not be rehashed into the Employer innovation: it already determines its exact cumulative index and total. The domain cardinality enters index construction and world identity. Adding Company-generation fingerprints or a roster digest to the Employer seed is unnecessary and is not part of this protocol. Similarly, selected Company identity is not needed to sample employee Income once authorized selected payload values are obtained.

## 9. Exact bounded sampling

### 9.1 Frozen bounded integer primitive

**INHERITED / ADOPTED FOR ALL ROWS.** Use exactly the source-defined SplitMix64 sequence and rejection-based Uint64n, not an arbitrary unbiased RNG. For independent compatibility, all state and products below wrap modulo 2^64:

```text
Uint64:
    state += 0x9e3779b97f4a7c15
    z = state
    z = (z XOR (z >> 30)) * 0xbf58476d1ce4e5b9
    z = (z XOR (z >> 27)) * 0x94d049bb133111eb
    return z XOR (z >> 31)

Uint64n(n):
    n == 0: error, consume nothing
    n == 1: return 0, consume no raw Uint64
    threshold = 2^64 mod n
    repeat x = Uint64 until x >= threshold
    return x mod n
```

The threshold can be computed with the frozen unsigned expression `(0-n)%n`. Direct `Uint64()%n`, floating thresholds, a different rejection orientation, a different hash-to-state width, or another PRNG is nonconforming. This restates existing semantics, not a new random primitive.

One semantic bounded invocation does not mean exactly one raw Uint64. Rejection may consume several; n=1 consumes none. No cross-node state is shared, so these internal details do not shift another node's draw. Implementations may optimize internally only when they reproduce the frozen primitive's values from the same initial state and bound. This specification adds no separate public promise about raw-call counts.

### 9.2 CAT, POINT and EMP

**NEW outside Employer; INHERITED for Employer.** Let c[k] be the checked cumulative sum through position k in canonical order. Let T = c[last], with 1 <= T <= MaxUint64. Invoke exactly one fresh node-specific Uint64n(T), obtaining r in `[0,T)`. Select:

```text
k = first position satisfying c[k] > r
```

This is lower-bound search on the monotone predicate `c[k] > r` (upper_bound by value), never `>=`. Its half-open interval is `[previous cumulative,c[k])`. A deterministic binary search is prescribed for Employer; an equivalent bounded scan or binary search is permitted for small categorical/support laws only if it returns exactly this k. No alternative probability representation is necessary: existing compiled cumulative integers are sufficient.

CAT returns the canonical output-category ordinal k mapped to its frozen semantic string. Zero-weight categories keep their slots and repeated cumulative values but can never be selected. POINT returns the aligned final uint32 support at k. Canonical numerical supports have positive weights; an eligible support of value zero is ordinary ground truth, not a separate coin flip. EMP returns the CompanyIndex associated with position k in the frozen compact index.

Validation precedes sampling: malformed/empty/all-zero laws, invalid supports, bad references and overflow reject. For each sum, `weight > MaxUint64-total` is rejected before addition, or a previously established bounded-width proof must establish impossibility. No normalization of Employer weights, wraparound, saturation, silent fallback, support clipping, jitter, zero repair or resampling is allowed. A valid positive-total cumulative law guarantees selection succeeds. Internal corruption returns a non-nil opaque internal error, never a zero/arbitrary success.

## 10. Company universe, identities and Employer

**INHERITED.** CompanyCount is exactly 65,536 for the current prescribed model; dense membership is 1..N. Company facts use two independently addressed draws at CompanyIndex j: Industry, then EmployeeCount conditional on its Industry. SizeBand is solely:

```text
0             → zero
1..9          → micro
10..49        → small
50..249       → medium
250..MaxUint32→ large
```

EmployeeCount is modeled anchor-date workforce/headcount, never occupancy, a mutable assignment counter, a roster length or a number of materialized Persons. Industry and count can be generated at j without generating j−1 or another Company. Constructor materialization of bounded immutable Company metadata remains permitted/required for indexes, not a sequential random dependency. No separate SizeBand array or independently sampled size exists in the D2-A retained architecture.

Company ID is exactly the frozen generic identity result for entity kind `company`, actual base namespace and raw CompanyIndex. No seed, N, model digest, socioeconomic ID or new codec enters it. The existing golden remains `company_v1_krenalis-demo_00000001_ef434ynxgmwyh` for namespace `krenalis-demo`, generation 1, index 1.

Construct exactly 40 required Employer indexes, with condition slot `4*F+J`. Every raw input includes each operational Company exactly once. For n=EmployeeCount[j]:

```text
n == 0: W_j = 0
n > 0: W_j = min(n,C) * I[F, Industry[j]] * S[J, positive SizeBand[j]]
```

The current values remain C=4096, I=64 times the exact B0 §19 matrix, and its exact S table. No factors are re-estimated here. I and S are strictly positive, so every positive-count Company is eligible in all 40 indexes; zero-count Companies exist but are ineligible. The positive SizeBand column order is micro,small,medium,large, i.e. full band ordinal minus one. All products/totals are checked uint64; neither factors nor final intervals are GCD-rescaled.

Sort by ascending CompanyIndex, reject duplicates, omit zero weights, build exact cumulative intervals and positive totals. Resolve `(F,J)` to its single existing index with no fallback. Use §8's Employee-scoped Employer stream, then exactly the frozen §9 EMP selector. CompanyCount is not scanned per employee. Assignments of other Persons never affect a weight or interval. An all-zero realized universe fails construction; no seed retry, forced employer, population-dependent count or occupancy repair is permitted.

The selected employee Industry is read only from the selected employer's authoritative Company Industry. Holding Employer j fixed, a change to an unselected Company's Industry cannot directly alter the relational Industry result or employee Income.

## 11. Income: six independent branch laws

**INHERITED.** Income is gross personal monetary Income from all eligible personal sources, annualized under conditions represented at the anchor, before personal taxes/contributions; business earnings are personally attributed after operating costs. It is not turnover, household resources, wealth, salary-only or a signed net-loss aggregate. Every successful facts value has one mandatory AnnualIncomeEUR, whole nominal EUR, uint32. Zero means no eligible personal Income, never missingness, non-applicability or generation failure.

Select exactly one Income branch using the already-generated PrimaryActivity. For each registry branch, flatten its authorized tuple in the stated order; use its own complete tupleMap to obtain its canonical law ID; read exactly that branch's law; sample its final supports with §9 POINT and that branch's independent stream. Law lookup has no draw; law ID and support position are not forwarded to any downstream node or added to the innovation.

Employee map shape is 20×10×4×14×5, including 11,200 zero-band entries with sentinel 65535. Those entries are unreachable with a valid eligible employer and must never be sampled; reaching one is an error. Positive bands yield the 44,800 reachable assignments. Self-employed map shape is own age-bucket count×20×10×14. Each non-working branch is its own age-bucket count×7×20. The actual maps remain the prescribed B3/B2 output; none is recomputed from socioeconomic scalars at runtime.

The runtime makes **one bounded support draw**, including the zero atom, for the selected branch. It does not generate a separate zero indicator, positive amount, segment, bucket, income source, interpolation parameter or scale. It does not execute Q curves, Decimal96, exact-scaling production, Hamilton apportionment, national transport, Ward pooling, fitting, support allocation or ceiling on a newly produced rate. Published supports are already final whole-EUR outcomes. B3 §5.1's production arithmetic remains authoritative for producing those values and is not reopened by this protocol.

For employee Income, Employer only locates the selected Industry and SizeBand. Replacing Employer j by k with the same selected values, holding Region/JobFamily/JobLevel and Person addressing/model fixed, gives identical dependency bytes, identical law and identical Income. Raw EmployeeCount changes within the same SizeBand likewise have no direct Income effect. No unselected Company contributes. Self-employed and non-working branches perform no employer lookup and use no Company model fingerprint.

## 12. Evaluation order, non-interference and random access

**INHERITED GUARANTEES; NEW CONCRETE REALIZATION.** For requested facts: validate wrapper/index; acquire allowed base inputs; generate Education; generate PrimaryActivity; evaluate only the selected occupational subsequence; evaluate only the selected Income branch; return a complete valid facts value or a non-nil error with the frozen zero result. Employee subsequence is JobFamily → JobLevel → Employer. Self-employed is JobFamily → Industry. Four non-working branches evaluate no occupation.

Do not sample, score, route or derive an inapplicable Person branch and discard it. Acquisition/validation of all branch model assets is not per-Person branch evaluation. The common facts routine is shared by facts-only and composite retrieval. No full Person/photo generation is a prerequisite of facts-only access.

Independent requests may calculate prerequisites on demand in topological order, or reuse immutable results. Requesting Income first cannot mean sampling it before its permitted parents are known. It means obtaining the same prerequisite values through their independent addresses; it does not add an Income-only public API. No ordering of requests, unrelated fields, Persons or Companies changes an address, law or result.

For each stochastic node, the result is a function only of the actual base realization root, its entity kind/coordinate, fixed node path, own canonical model identity, permitted parent projections, and selected cumulative law/index. Therefore repeated, forward, reverse, interleaved and concurrent requests agree. No global mutable RNG, previous-Person/Company dependency, materialized population, mutable employer occupancy or request counter exists.

A node model revision changes its own fingerprint, not non-descendant fingerprints. Descendants receive only their actual authorized parent values, not the ancestor's fingerprint. A full model/world identity change does not propagate through seeds. In particular: Income model changes cannot affect Company/occupation/Education/activity/Employer; one Income branch cannot affect another; EmployeeCount model changes cannot affect Company Industry; changing N alone preserves common Company coordinates' facts. Allowed descendant propagation is permission, not a guarantee every discrete value changes.

Returned branch payloads must not expose mutable retained state or cause later calls to depend on caller mutation. No semantic exemption is created for caches, workers or concurrent query execution.

## 13. Construction, validation, resources and failures

**AMENDED LIFECYCLE (FROZEN).** The sole NewSocioeconomicWorld constructor receives the authoritative base World and an explicit successfully acquired immutable model. Loading lifecycle, ownership, phase-local errors and resource lifetimes follow the loading amendment §§2–7; construction and generation remain bounded local operations. Contract 1 §7 steps 2–3 and §24 remain binding under that lifecycle.

Acquisition failures expose no model. Nil/uninitialized base or model, invalid local identity/state, incomplete reachable routing, invalid Company/index state or over-budget construction fail without exposing a partial wrapper. **INHERITED.** Company/Facts/Person on invalid wrappers return ErrInvalidSocioeconomicWorld; ID on an invalid wrapper returns the empty invalid identity. Zero indices use ErrInvalidIndex on a valid wrapper; operational upper bounds retain their owning errors. Unsupported primitive/component/internal failures must be reported through the frozen public error contract, not new success sentinels or invented public error categories.

All C2/D2-A resource limits remain in force, with only the loading amendment §6's phase-accounting interpretation. N is fixed at 65,536; D2-A's effective maximum 104,855 is tighter than C2's generic 1,048,576. Forty indexes use the existing serial construction and disjoint accounting. Model compilation, canonical serialization and fingerprint work must be included in the existing encoded, decoded/work, retained, additional-identity/bookkeeping and peak charges; cached fingerprints are not free memory. Retaining 15 fingerprints plus one logical digest is 512 payload bytes, before any implementation metadata. Static literal strings are not population-sized state.

Hashing canonical model projections is acquisition-phase model-compilation work completed before the immutable model is published; binding the actual base WorldID is local world-construction work. Process one bounded projection at a time and release temporary representations, or stream byte-identical serialization into SHA-256 with validated lengths; neither choice changes KFD1 or the output. No JSON serialization, provenance parsing or table-wide fingerprint reconstruction occurs per draw. This imposes no second runtime probability table or expanded support domain. Existing cumulative arrays may be reused unchanged.

The implementation must prove actual live allocations fit all frozen bounds, including any temporary canonical-record copies. Publication's prospective accounting is not a measured runtime allocation certificate. No implementation may silently exceed a sub-budget merely because total process memory is available.

## 14. Mechanical reference vectors

**NEW NORMATIVE FIXTURES, COMPUTED AND CROSS-CHECKED; NOT PRODUCTION GOLDENS.** Companion `SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_VECTORS_v1.json` contains the exact synthetic fixture input model, complete KFD1 records as hex, hashes, initial states, bounded draws and selected outputs. It is an approved protocol attachment, not an additional D2-B2 publication asset.

The fixture uses small full structural grids and synthetic laws to exercise addressing. It is not the prescribed D2-B generator output and is not authorized as a replacement production model. Component rows provide explicit parent vectors independently; evaluating fixtures for all six Income templates does not represent evaluating six branches for one runtime Person. The Employer row uses the frozen C2 selector-only candidate index, not a fabricated full generated universe. No actual SocioeconomicWorld or Layer 1 Person was constructed for these fixtures.

The JSON includes: 15 complete node fingerprint/dependency/stream examples; six coordinate and entity-ID examples including the largest operational Person and generic non-operational high coordinates; four sampling-family boundary sets; exact n=1 and rejection primitive fixtures; a UTF-8/empty-bytes/uint64 framing example; and an inactive-Income-only model perturbation. Hex is a display of bytes, not the bytes' framing at runtime. The b/s/u example exercises generic KFD1 framing only; it is not an alternative permitted field set for the §5 world identity.

The coordinate stream examples reuse the Education node example's path, model and parent components for `person`, and the Company Industry node example's path, model and parent components for `company`, changing only the requested coordinate. The inactive-only mutation starts from the fixture `model` and changes only `income.i.laws[0].values[1]` from 100 to 101; all other inputs remain unchanged.

Computed constants:

```text
baseRoot: 30248f6f4dfbdb86e37a6b68aba18e4d79c99a933af7211358577db734fdfdda
baseWorldID: world_v1_e51e7922e0763f75ac47a1f3973253bd20ea3a1772158d5963f38d546304ef9b
logicalModelDigest: a9165b7ffd3ec54692e94a9d21a963c6ad8c94267f5067b3cee7ee2e6428efae
socioeconomicWorldID: socioeconomic-world_v1_2246438b4107e2c8399837c5d51b677ac702234fdc998d23237f73de9eccbccc
```

The frozen primitive rejection witness is initial state 3, bound 9,223,372,036,854,775,809. Raw candidate `1d0b14e4db018fed` is rejected; `b3466f8a7b81a989` is accepted; r=3,694,763,184,872,335,752. This is one semantic invocation, not two. The n=1 fixture returns zero without advancing state.

Boundary fixture: cumulative `[3,4,6]`, r=0,2,3,5 selects positions 0,0,1,2. For CompanyIndex values `[1,2,3]`, results are 1,1,2,3. For Income supports `[0,100,1000]`, results are 0,0,100,1000. Categorical weights `[0,3,0,1,2,0,0]` select ordinals 1,1,3,4 for those draws: leading/interior/trailing zero weights are never selected.

Verification performed: Python independently encoded KFD1, SHA-256 composition, coordinates, identity mixing, SplitMix64 and sampling. A separate temporary Go verifier used unchanged supplied foundation files, independently canonicalized logical JSON, and matched model/dependency/stream bytes, digests, states, selections, identity inverses, boundaries and the model-mutation world ID. Original component/composed-ID/identity/random/stream test files passed together offline. No existing file or golden was changed.

Mandatory Layer 2 fixture comparisons concern frame bytes, digests, initial deterministic addresses, bounded results and selected outputs. Recorded `rawCandidates` and `finalStateHex` are explanatory traces of the existing primitive, not an additional Layer 2 API promise about private raw-call counts or unused post-selection state.

This verifies the approved byte/sampling fixtures, not a production runtime, full-model publication authenticity, payload generation, real World construction, Go race safety or complete Layer 1 regression. Those implementation obligations are not represented as PASS here.

## 15. Implementation obligations and explicit non-goals

Implementation review must check the public API/error contracts subject only to the loading amendment; actual base root and WorldID reuse; complete fixed registry; literal field types; validated canonical model reconstruction; no raw/transport checksums in generation; all categorical/count/Income boundaries; exact Employer indexes; facts-only/composite agreement; selected-branch non-evaluation; fixed-parent and branch-local perturbations; maximum operational random access; shuffled/concurrent retrieval; caller-mutation safety; resource guards and offline constructor behavior. Run the production model against the same protocol vectors, then add independently computed real-world integration goldens without selecting new protocol behavior.

Not selected here: numerical fitting, canonical production contents, quality thresholds, publication schema/file set, transport/resolver changes, source-observation integration, CountryPack/PopulationMix, Simulated System documents, admin/schema/database/API integration, public Income-only or Company enumeration methods, new sentinels or a socioeconomic snapshot ID. These are not hidden compatibility choices left in this protocol; they remain outside its scope under their existing owners.

**Approved status:** no additional value-affecting protocol choice is intentionally left to implementation. Independent technical review and manual approval are complete; this freeze is not proof that a future implementation is correct.
