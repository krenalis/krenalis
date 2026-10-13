# Synthetic World v1 — Layer 2 Detailed Contract 2 v0

## Company Universe Protocol

### 1. Status and frozen baseline

**NORMATIVE:** This document is the proposed detailed normative Company-universe contract v0, ready for independent review. It is not an implementation or freeze approval.

Verified baseline:

```text
Branch: property-semantics
HEAD:   76fa0fff109ca6abad62f55c8761275455842dc5
Subject: Add Synthetic World Layer 1
```

Verified frozen Contract 1:

```text
File:   SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md
SHA-256:
bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6
```

**NORMATIVE:** Layer 0, Layer 1, and frozen Layer 2 Contract 1 remain unchanged. This contract extends Contract 1 by defining the reusable mechanical Company-universe and employer-selection-index protocol that it deferred.

**MODEL:** Missing empirical D2 data does not block this protocol. It prevents this contract from fixing the actual CompanyCount, EmployeeCount distribution, required semantic index set, conditioning dimensions, EmployerWeight derivations, and condition-to-index routing semantics.

### 2. Scope / non-scope

**NORMATIVE:** This contract freezes:

- Dense operational Company-universe semantics.
- Internal CompanyCount ownership, validation, and resource maximum.
- The EmployeeCount field, type, meaning, and safety range.
- Company existence versus index-specific employer eligibility.
- The private EmployerWeight abstraction.
- Canonical Company ordering.
- Exact logical selection-index representation and construction result.
- Exact weighted-selection interval mapping.
- Overflow and resource behavior for Company metadata and employer-selection indexes.
- Initialization, immutability, concurrency, and failure semantics within this contract’s scope.
- Compatibility and identity-coverage classifications.

**NORMATIVE:** Contract 2 adds no public `SocioeconomicWorld` method, public Company enumeration API, iterator, public CompanyCount field, or bulk Company API.

**DEFERRED:** This contract does not define:

- The actual numeric CompanyCount.
- An empirical Company-count derivation.
- EmployeeCount distributions.
- Which employer-selection indexes the completed v1 model requires.
- Index conditioning dimensions or semantic subsets.
- EmployerWeight derivations.
- Employee-assignment condition semantics.
- Condition-to-index routing semantics or representation.
- Company Name, Industry, geography, or SizeBand.
- Education, JobFamily, JobLevel, or Income.
- Person-to-Company causal conditioning.
- Stream paths or dependency Components.
- Exact SocioeconomicWorldID encoding.
- D2 source selection or empirical-fit requirements.

No runtime configuration is introduced.

### 3. Frozen Contract-1 constraints

**NORMATIVE:** The following Contract-1 decisions remain authoritative:

- `CompanyIndex` is a defined `uint64` type.
- Protocol coordinates are one-based; zero is invalid.
- Company identity uses entity kind `"company"` and the base World’s authoritative `IdentityNamespace`.
- Company identity does not depend on WorldSeed or CompanyCount.
- The operational universe is finite and dense.
- Protocol decoding does not prove operational membership.
- Company facts are independent of assigned Persons, materialization order, occupancy, worker count, and PersonRange.
- EmployeeCount means modeled workforce/headcount at `PopulationAnchorDate`.
- EmployeeCount is not realized Synthetic World occupancy.
- Assignment uses immutable Company-world state and deterministic random access.
- `NewSocioeconomicWorld(base *World)` remains the sole constructor.
- CompanyCount and model choices are not runtime constructor arguments.
- Every output-affecting Company-universe choice must eventually be covered by `SocioeconomicWorldID`.
- Company, Facts, and Person operations on an invalid wrapper return `ErrInvalidSocioeconomicWorld`; `ID` returns its empty invalid value.
- A generated employee’s Employer is a valid member of the generating socioeconomic World.

Frozen Contract 1 defines the complete conceptual public operation set:

```go
func NewSocioeconomicWorld(base *World) (*SocioeconomicWorld, error)

func (w *SocioeconomicWorld) Company(index CompanyIndex) (Company, error)
func (w *SocioeconomicWorld) Facts(index PersonIndex) (SocioeconomicFacts, error)
func (w *SocioeconomicWorld) ID() SocioeconomicWorldID
func (w *SocioeconomicWorld) Person(index PersonIndex) (SocioeconomicPerson, error)

func CompanyIDForIndex(namespace IdentityNamespace, index CompanyIndex) (string, error)
func CompanyIndexFromID(namespace IdentityNamespace, id string) (CompanyIndex, error)
```

**NORMATIVE:** Contract 2 does not add to or replace this operation set. Extending `Company` with EmployeeCount is permitted because Contract 1 explicitly deferred additional Company fields.

### 4. Source constraints

| Label | Frozen source evidence | Contract consequence |
| --- | --- | --- |
| **SOURCE FACT** | Generic identity encoding accepts a kind and any nonzero `uint64` coordinate: `tools/fakedata/identity.go:110-131`. Decoding returns the original nonzero coordinate after validating kind and namespace: `tools/fakedata/identity.go:134-165`. | Company identity needs no new codec and remains independent of the operational count. |
| **SOURCE FACT** | The frozen Company golden uses kind `"company"` and coordinate 1: `tools/fakedata/identity_test.go:314-327`. | This contract preserves that identity mapping. |
| **SOURCE FACT** | `streamFactory.entityStream` derives a fresh entity-scoped RNG from kind, coordinate, path, and dependencies: `tools/fakedata/stream.go:25-49`. | Company-scoped deterministic generation is feasible without Layer 0 changes. |
| **SOURCE FACT** | Entity streams reject coordinate zero and validate entity kind and path: `tools/fakedata/stream.go:63-82`. | Operational Company generation must validate membership before stream derivation. |
| **SOURCE FACT** | `Uint64n` uses rejection sampling and returns a value in `[0,n)`: `tools/fakedata/random.go:48-65`. | Employer selection must call this primitive instead of direct remainder reduction. |
| **SOURCE FACT** | `Uint64n(1)` returns zero without consuming a raw RNG output: `tools/fakedata/random.go:55-57`; its boundary and rejection behavior are golden-tested at `tools/fakedata/random_test.go:153-214`. | The protocol specifies one `Uint64n` invocation, not one raw `Uint64` invocation. |
| **SOURCE FACT** | `WeightedIndex` checks addition using `weight > math.MaxUint64-total`, rejects total zero, draws once, then scans cumulative intervals: `tools/fakedata/random.go:69-100`. | Checked totals and strict upper cumulative boundaries are existing deterministic precedents. Its linear scan is not the Company lookup algorithm. |
| **SOURCE FACT** | Layer 1 population-weighted place selection first chooses a bucket and then chooses within it: `tools/fakedata/layer1.go:479-493`. Buckets contain at most 128 place weights: `tools/fakedata/layer1_data.go:36`, `tools/fakedata/layer1_data.go:95-115`. | Layer 1’s hierarchical interval mapping is not reused. Each Company employer-selection index uses one canonical cumulative-prefix mapping. |
| **SOURCE FACT** | Layer 1 loads immutable data once with `sync.OnceValues`: `tools/fakedata/layer1_data.go:39`; it builds all tables before returning the data value: `tools/fakedata/layer1_data.go:237-328`. `NewWorld` similarly returns only after all state is established: `tools/fakedata/layer1.go:148-228`. | Required Company-universe and index state is immutable and constructed all-or-nothing. |
| **SOURCE FACT** | Repository resource precedents bound records, bytes, and integer allocation arithmetic before use: `tools/fakedata/layer1_photo.go:30`, `tools/fakedata/layer1_photo.go:517-537`, `tools/fakedata/layer1_photo.go:679-701`, and `tools/fakedata/component.go:139-148`. | Company limits must be explicit technical bounds with checked arithmetic. |
| **SOURCE FACT** | The package benchmarks entity-stream derivation, bounded sampling, and ordered weighted selection: `tools/fakedata/fakedata_bench_test.go:60-120`, `tools/fakedata/fakedata_bench_test.go:234-265`. | Future Company benchmarks should measure construction and logarithmic lookup without replacing correctness tests. |
| **SOURCE FACT** | Layer 0 provides `ErrAllZeroWeights`, `ErrTotalWeightOverflow`, and related foundation categories: `tools/fakedata/errors.go:7-28`. Layer 1 uses exported sentinel categories backed by an unexported string type: `tools/fakedata/layer1_errors.go:7-22`. | Constructor-visible invalid Company state uses frozen `ErrInvalidSocioeconomicWorld`; no new public error category is needed. |
| **SOURCE FACT** | Repository rules require exported argument validation and bounded execution for APIs without context: `AGENTS.md:68-72`, `AGENTS.md:152-168`, `AGENTS.md:261-269`. | The no-context constructor must have enforceable local work and memory bounds. |

### 5. Terminology

**NORMATIVE:**

- **CompanyCount** is the internal number of operational Companies in one `SocioeconomicWorld`.
- **Operational Company** is a Company whose coordinate lies in `1..CompanyCount`.
- **EmployeeCount** is that Company’s modeled workforce/headcount at `PopulationAnchorDate`.
- **EmployerWeight** is a private exact nonnegative integer interval length used by one employer-selection index.
- **Employer-selection index** is an immutable weighted index over a deterministic semantic subset of the operational Company universe.
- **Selection input entry** is a `(CompanyIndex, EmployerWeight)` pair supplied to the reusable index builder.
- **Positive entry** is a selection entry whose EmployerWeight is greater than zero.
- **Employee-assignment condition** is a future DAG/model state whose semantics determine which employer-selection index applies.
- **Condition-to-index resolution** is the future model-owned deterministic resolution of an assignment condition to exactly one required index.
- **Realized occupancy** is the number of generated Persons whose employer happens to be a Company. It is not EmployeeCount.
- **MiB** means exactly 1,048,576 bytes.

**MODEL — DEFERRED:** The completed v1 socioeconomic model may require zero, one, or more employer-selection indexes. The Socioeconomic DAG and Company/D2 model contracts own their exact set, conditioning dimensions, subsets, weight derivations, routing semantics, and routing representation.

### 6. Company universe

**PROTOCOL — NORMATIVE:**

For every successfully constructed socioeconomic World:

```text
CompanyCount > 0
Operational membership = 1 <= CompanyIndex <= CompanyCount
Canonical universe order = CompanyIndex ascending
```

The universe is dense. No hole, deleted coordinate, eligibility rule, future geography, Industry criterion, or index membership changes Company existence.

Every operational coordinate has a deterministic Company value and EmployeeCount. A Company may exist while being ineligible for one or every employer-selection index.

The full protocol coordinate domain remains `1..math.MaxUint64`. Coordinates above CompanyCount can still encode and decode as Company IDs, but `Company(index)` rejects them as outside this World.

### 7. CompanyCount ownership

**MODEL — NORMATIVE:** Decision A2 is selected.

The later **Company/D2 Model Contract** owns the actual numeric v1 CompanyCount. That contract must freeze one literal positive `uint64` value, supported by its empirical/model evidence, satisfying this contract’s hard maximum.

It may explain how evidence led to the number, but runtime construction must not derive a caller-specific count from WorldSeed, ReferenceDate, FaceCatalog, Person materialization, or occupancy.

**NORMATIVE:** CompanyCount is:

- Required private immutable `SocioeconomicWorld` state.
- Fixed model/data state.
- Positive and finite.
- The dense-universe cardinality.
- The same for every World using the same Company-universe model version.
- Not a constructor argument.
- Not mutable after construction.
- ReferenceDate-independent.
- FaceCatalog-independent.
- Used internally by `Company(index)` membership validation.
- Bounded by the Contract-2 hard maximum.
- Covered by `SocioeconomicWorldID`.

**DEFERRED:** No numeric MODEL value is selected here.

### 8. CompanyCount internal-state contract

**NORMATIVE:** CompanyCount remains private implementation state. Contract 2 introduces no public accessor for it.

In particular, Contract 2 does not add:

```go
func (w *SocioeconomicWorld) CompanyCount() uint64
```

and does not replace it with another public accessor, iterator, enumeration method, public field, or bulk Company API.

`Company(index)` validates membership against the private immutable count:

```text
index == 0              -> ErrInvalidIndex
index > CompanyCount    -> ErrCompanyIndexOutOfRange
otherwise               -> operational member
```

Internal CompanyCount is invariant under ReferenceDate and FaceCatalog changes. Same-package tests may inspect internal state where necessary, but no public enumeration contract is created.

### 9. CompanyCount hard resource limit

**RESOURCE — NORMATIVE:**

```text
Maximum CompanyCount = 1,048,576 = 2^20
```

This is a hard technical limit. It is not an estimate of the number of Italian enterprises.

Construction rejects:

```text
CompanyCount == 0
CompanyCount > 1,048,576
```

The limit:

- Fits signed 32-bit `int`, so required slice lengths and `int(index-1)` conversions are safe on supported 32-bit and 64-bit Go architectures.
- Permits an individual index containing every operational Company without requiring that such an index exist.
- Keeps linear Company metadata construction bounded.
- Is independent of `Layer1MaxPersonIndex`.

The limit is not exported as public API.

### 10. Company public model additions

**API — NORMATIVE:**

```go
// Company contains the generated facts for an operational Company.
type Company struct {
    ID            string
    EmployeeCount uint32
    // Additional fields belong to later contracts.
}
```

CompanyIndex is not duplicated in Company. The lookup coordinate is already known, and ID remains reversible under the frozen identity protocol.

This contract adds no Name, Industry, SizeBand, geography, or other Company field.

### 11. EmployeeCount semantics/type/range

**MODEL — NORMATIVE:** EmployeeCount means:

> The modeled Company workforce/headcount at `PopulationAnchorDate`.

It does not mean realized assigned Synthetic World Persons.

**API — NORMATIVE:** Its Go type is `uint32`.

**RESOURCE — NORMATIVE:** Its complete technical range is:

```text
0 <= EmployeeCount <= math.MaxUint32
```

The upper bound is the representational and model-safety limit of the public type. It is not an empirical statement that a real Company can or cannot reach a particular size.

Zero is valid. A zero value can represent an existing Company with no modeled workforce at the anchor. Every operational Company has exactly one EmployeeCount, including zero.

A fixed data value that is negative, fractional, unrepresentable, or greater than `math.MaxUint32` is malformed and must be rejected before conversion.

**MODEL — DEFERRED:** The distribution, correlations, and deterministic derivation of EmployeeCount belong to the Company/D2 Model Contract.

EmployeeCount is:

- Stable across ReferenceDate changes.
- Independent of FaceCatalog and PhotoID.
- Independent of Person generation and occupancy.
- Deterministic for the Company coordinate and fixed socioeconomic model.
- Covered by `SocioeconomicWorldID` through its model/data identity.

### 12. Existence versus eligibility

**PROTOCOL — NORMATIVE:**

Company existence is determined only by dense membership:

```text
1 <= CompanyIndex <= CompanyCount
```

Employer eligibility is index-specific.

For any employer-selection index:

- A positive EmployerWeight makes the Company eligible.
- A zero EmployerWeight makes the Company ineligible.
- A Company absent from the index’s semantic subset is ineligible for that index.
- Zero-weight entries consume no interval and are omitted from the compact index.
- Omission from an index does not remove the Company from the universe.
- A zero EmployeeCount Company must have zero EmployerWeight in every employee-employer index.
- A positive EmployeeCount does not require a positive weight in every index; deterministic semantic criteria may exclude it.

Thus:

```text
Company existence != eligibility in one employer-selection index
```

Contract 2 does not require any index to contain every operational Company.

A later model may specifically require a complete-universe input for one of its indexes. The generic builder may support that validation mode, but the mode is used only when the owning later contract requires it.

### 13. EmployerWeight

**PROTOCOL — NORMATIVE:** Employer selection uses a distinct private abstraction:

```go
type employerWeight uint64
```

EmployerWeight is:

> The exact nonnegative integer interval length assigned to a Company in one employer-selection index.

Its protocol range is the complete `uint64` range. Zero means ineligible. Positive values are relative weights; no normalization, floating-point representation, or probability rounding is allowed.

EmployerWeight is private. It is not added to Company output.

**MODEL — DEFERRED:** The Socioeconomic DAG and Company/D2 Model Contract own each required index’s semantic subset and EmployerWeight derivation.

This contract does not freeze:

```text
EmployerWeight == EmployeeCount
```

EmployeeCount is only a possible model input. A later derivation may transform it, provided the derivation is deterministic, integer-valued, occupancy-independent, identity-covered, and obeys the zero-EmployeeCount rule.

Malformed, negative, fractional, or greater-than-`math.MaxUint64` source values are invalid before conversion.

### 14. Canonical ordering

**PROTOCOL — NORMATIVE:**

The canonical Company-universe order is ascending numeric CompanyIndex.

Every universe-wide metadata array uses this order. EmployeeCount for CompanyIndex `i` is stored at position `i-1`.

Before cumulative intervals are constructed, every employer-selection input is canonicalized to:

```text
strictly ascending numeric CompanyIndex
```

The canonical result must satisfy:

- Every coordinate is operational.
- Every coordinate occurs at most once.
- Each weight remains attached to its original CompanyIndex.
- Input and map iteration order have no meaning.
- Identical semantic input produces identical ordered logical entries.
- No Go map iteration determines interval order.

The sorting or canonicalization algorithm is not part of output protocol identity when it:

1. Produces exactly the required ascending order.
2. Preserves exact `(CompanyIndex, EmployerWeight)` pairs.
3. Rejects duplicates.
4. Obeys Contract-2 construction and resource bounds.
5. Does not expose input iteration order to the resulting intervals.

A future contract may define how a semantic subset is chosen. It may not change the resulting ascending index order.

### 15. Selection-index representation

**PROTOCOL — NORMATIVE:** The logical representation is an immutable cumulative-prefix index:

```go
type employerSelectionEntry struct {
    company    CompanyIndex
    cumulative uint64
}

type employerSelectionIndex struct {
    entries []employerSelectionEntry
    total   uint64
}
```

Only positive-weight entries are retained.

For retained entry `i`:

```text
entries[i].company is strictly increasing
entries[i].cumulative = sum of retained weights through i
entries[i].cumulative is strictly increasing
```

The final cumulative value equals `total`.

The interval owned by entry `i` is:

```text
[previous cumulative, entries[i].cumulative)
```

where the previous cumulative value is zero for the first entry.

The index contains no per-employee slots, expanded weight array, mutable cursor, occupancy count, or map.

Raw EmployerWeights do not need to remain stored after construction. The cumulative difference between adjacent entries represents the original retained weight exactly.

### 16. Selection-index construction

**PROTOCOL — NORMATIVE:** Construction follows this exact semantic sequence:

1. Validate CompanyCount and all count/resource arithmetic.
2. Validate that input-entry count does not exceed CompanyCount.
3. Validate every coordinate is in the operational universe.
4. Validate every source weight is an exact `uint64`.
5. Count positive entries and compute total weight with checked addition.
6. Reject total zero or total overflow.
7. Verify projected aggregate input, retained-index, and construction-resource charges.
8. Obtain a private canonical representation without mutating model-owned input.
9. Canonicalize complete `(CompanyIndex, EmployerWeight)` pairs by ascending numeric CompanyIndex.
10. Reject duplicate CompanyIndex values.
11. If the later owning model requested complete-universe validation, require every coordinate in `1..CompanyCount` exactly once.
12. Scan in ascending order, omit zero weights, and construct exact cumulative totals.
13. Verify that the final cumulative value equals the preflight total.
14. Publish the completed immutable index only after every validation succeeds.

The canonicalization mechanism itself is not frozen. A comparison sort, radix sort, or another deterministic implementation may be used if it produces the exact required logical result and remains within the frozen work and memory bounds.

Zero-weight entries are valid but omitted. Empty input, nonempty all-zero input, and every other zero-total operational index are invalid.

The completed v1 socioeconomic model may require one or more immutable employer-selection indexes. Contract 2 does not require any particular semantic index to exist.

A later model that can generate employees must guarantee that the required index exists and has positive total weight for every reachable employee-assignment condition.

No runtime fallback to an unrelated index is allowed.

### 17. Employer-selection algorithm

**PROTOCOL — NORMATIVE:** Given a valid selection index and the fresh RNG selected by the later deterministic-stream contract:

```text
r = Uint64n(index.total)
```

Return the CompanyIndex at the first position satisfying:

```text
index.entries[position].cumulative > r
```

The search is deterministic lower-bound binary search over the strict predicate `cumulative > r`.

The interval convention is exact:

```text
weight w at cumulative interval [start, start+w)
```

The selector performs no CompanyCount scan.

A successfully constructed index guarantees that binary search finds one entry. If internal corruption violates this invariant, the operation must return a non-nil opaque internal error rather than report success with a zero or arbitrary CompanyIndex.

### 18. RNG compatibility semantics

**PROTOCOL — NORMATIVE:**

- The selector invokes frozen `Uint64n(total)` exactly once at the protocol level.
- It does not use `Uint64() % total`.
- It does not use floating-point thresholds.
- It does not invoke frozen `WeightedIndex`.
- It does not perform hierarchical bucket selection.
- It does not consume an RNG draw for each Company.
- Zero-weight entries do not consume intervals.
- Input or map order does not affect the result.
- The selected interval uses `cumulative > r`, not `cumulative >= r`.
- `total == 1` still requires one `Uint64n(1)` invocation. Frozen semantics mean that invocation consumes no raw `Uint64` output.
- Rejection sampling may internally consume more than one raw `Uint64` output.

Compatibility concerns the final CompanyIndex produced from the frozen RNG state and the exact canonical cumulative index. No promise is made about the number of raw `Uint64` calls.

**DEFERRED:** The Person-scoped stream paths, dependency Components, employee-assignment conditions, condition-to-index routing semantics, and routing representation belong to later contracts.

### 19. Overflow rules

**PROTOCOL — NORMATIVE:**

For every weight addition:

```text
if weight > math.MaxUint64 - total:
    reject
total += weight
```

A valid index requires:

```text
1 <= total <= math.MaxUint64
```

No wrapping, saturation, rescaling, normalization, truncation, or floating-point fallback is allowed.

Positive-entry counting, input-entry counting, index counting, byte-charge multiplication, and byte-charge addition also use checked arithmetic before conversion or allocation.

Cumulative construction repeats or preserves the checked-sum guarantee. A mismatch between preflight total and final cumulative total is invalid fixed state.

### 20. Resource and memory bounds

All values in this section are **RESOURCE** limits, not empirical MODEL claims.

#### Fixed limits

```text
Maximum CompanyCount:                  1,048,576
Maximum employer-selection indexes:   1,024
Aggregate retained index budget:      64 MiB
Aggregate selection-input budget:     64 MiB
Retained Contract-2 state budget:     72 MiB
Contract-2 construction peak budget: 128 MiB
```

These maxima do not require a model to construct any particular number of indexes. The index count may be zero when the completed model has no reachable employee assignment.

#### Normative logical charge model

**RESOURCE — NORMATIVE:** All formulas in this section are contractual logical resource charges. Constants and terms such as `64`, `32*K`, `16*P`, `16*M`, and `4*CompanyCount` are not claims about exact Go allocator RSS, GC metadata, malloc size classes, or total process memory.

Every conforming implementation must enforce these charges regardless of physical allocator details. Actual implementation storage may be smaller; that does not reduce the prescribed charges. Implementation-specific additional storage requires a conservative logical charge in the appropriate explicit term below. Counts, byte amounts, their derivations, and every sum or product must be validated with checked arithmetic before use.

Every allocation governed by Contract 2 must be accounted for exactly once in a construction phase. The retained-state terms are disjoint; `retainedBytes` then enters `peakBytes` once. A buffer already covered by a named charge must not also appear in an additional-state term. Conservative reservations may exceed current live storage, but do not authorize omitted allocations or duplicate charges.

#### Normative retained-index accounting

For `K` indexes and `P` total retained positive entries:

```text
retainedIndexBytes = 64 + 32*K + 16*P
retainedIndexBytes <= 64 MiB
K <= 1,024
```

The charge comprises root state, one slice-and-total descriptor per index, and one 16-byte entry per retained positive weight. Backing-array capacities equal lengths.

#### Normative selection-input accounting

For `K` index inputs and `M` total raw input entries, including zero weights:

```text
selectionInputBytes = 64 + 32*K + 16*M
selectionInputBytes <= 64 MiB
```

This budget bounds total semantic input volume and construction work even when many input entries have zero weight. It is not a measure or proof of simultaneously-live physical memory. Live raw inputs and model-loading buffers are accounted for separately in the construction-peak formula, including inputs preloaded for a later index.

Each individual index input satisfies:

```text
input entry count <= CompanyCount
```

#### Company metadata accounting

The Contract-2-owned Company metadata required after construction is one dense EmployeeCount array:

```text
companyMetadataBytes = 64 + 4*CompanyCount
```

At maximum CompanyCount:

```text
companyMetadataBytes = 4,194,368 bytes
```

The required representation does not retain separate EmployerWeight arrays after their cumulative indexes are built. An implementation choosing to retain such arrays or equivalent auxiliary state must include their conservative charge in `additionalRetainedBytes`.

#### Retained state

**RESOURCE — NORMATIVE:** `additionalRetainedBytes` is the total charged bytes of every Contract-2-owned retained structure that remains live after construction and is not already included in `companyMetadataBytes` or `retainedIndexBytes`.

This includes any implementation-chosen retained EmployerWeight array or equivalent auxiliary state. It excludes Company metadata already charged, cumulative entries/descriptors already charged, and future DAG routing state owned by another contract. No Contract-2-owned retained allocation may escape this accounting by being called an implementation detail.

Using checked arithmetic:

```text
retainedBytes =
    companyMetadataBytes
    + retainedIndexBytes
    + additionalRetainedBytes

retainedBytes <= 72 MiB
```

An implementation with no additional retained state uses `additionalRetainedBytes = 0`.

At maximum CompanyCount, the full 64 MiB retained-index budget, and zero additional retained bytes:

```text
71,303,232 bytes
```

are charged, below the 72 MiB limit of 75,497,472 bytes. Additional retained storage consumes the remaining allowance and must also satisfy the construction-peak budget.

#### Normative construction-peak accounting

Only one index input is canonicalized at a time.

**RESOURCE — NORMATIVE:** `otherLiveConstructionBytes` is the charged bytes of every simultaneously-live temporary or source/model-loading allocation in the current construction phase that is not already included in `retainedBytes`, `rawEmployeeCountStagingBytes`, `currentRawInputBytes`, or `canonicalizationWorkingBytes`.

Examples include decoded model-loading buffers, preloaded future selection inputs still live while another index is canonicalized, parsing/validation buffers, and other Contract-2 construction temporaries. No such object is required to exist. If none exist, `otherLiveConstructionBytes = 0`.

The fixed construction overhead is one 4 KiB logical reservation for fixed control bookkeeping. It cannot absorb otherwise uncharged variable storage: any implementation-specific excess or additional buffer enters the appropriate explicit term. Storage already covered by the fixed allowance is not charged again.

Every implementation must charge, using checked arithmetic:

```text
peakBytes =
    retainedBytes
    + rawEmployeeCountStagingBytes
    + currentRawInputBytes
    + canonicalizationWorkingBytes
    + otherLiveConstructionBytes
    + fixedConstructionOverhead
```

where:

```text
rawEmployeeCountStagingBytes = 8 * CompanyCount
currentRawInputBytes         = 16 * currentInputEntryCount
fixedConstructionOverhead   = 4 KiB
peakBytes                   <= 128 MiB
```

`canonicalizationWorkingBytes` includes every private copy, scratch array, auxiliary table, stack allowance, and other temporary storage used by the selected canonicalization implementation, except fixed control bookkeeping already charged once in `fixedConstructionOverhead`. The fixed allowance cannot be assigned to two live structures at once; any excess receives its own charge.

The raw EmployeeCount staging representation is released after checked conversion to the retained `uint32` array. The formulas above and the resource validator conservatively reserve its full eight-byte-per-Company charge even in subsequent phases. Current raw input similarly receives the prescribed 16-byte-per-entry charge. Model-loading or decoded representations beyond these charged buffers enter `otherLiveConstructionBytes`.

Another canonicalization implementation is conforming only if its own conservative peak charge stays within 128 MiB.

#### Resource lifetime and allocation validation

**RESOURCE — NORMATIVE:** Before any allocation whose presence could make a resource limit fail, construction must conservatively compute the charge for the phase in which that allocation will coexist with other live state. It must include the proposed allocation and validate all relevant counts, checked byte arithmetic, integer conversions, and budgets before allocating.

This applies to every Contract-2 allocation sized from CompanyCount, input-entry count, positive-entry count, retained-state amount, canonicalization work, or model-loading state. Allocation followed by discovery of a budget violation is nonconforming. Each new allocation or changed phase requires a still-valid preflight accounting; earlier validation is sufficient only if its conservative reservation covers the complete new phase.

A conforming implementation may reduce a phase charge by releasing storage before the next phase. It may claim a buffer is absent only after that buffer is no longer live or reachable for construction use. The conservative fixed reservations in the formulas may remain even after the corresponding buffer is released.

- If selection inputs are loaded one at a time and released, only the current live raw input needs raw-input peak charging.
- If multiple selection inputs remain simultaneously live, every additional live input is charged in `otherLiveConstructionBytes`.
- Model-loading buffers must either be released before the charged phase or included in `otherLiveConstructionBytes`.
- Additional Contract-2 retained state is charged in `additionalRetainedBytes`, and therefore in `retainedBytes` and the peak, without a second charge as temporary state.
- Copies and distinct backing storage are charged separately; aliases to storage already charged do not cause the same backing storage to be charged again.

The 64 MiB aggregate selection-input budget remains a semantic input-volume/work limit. It cannot substitute for phase-by-phase live-storage accounting. No simultaneously-live Contract-2 allocation may be omitted because it is implementation-specific.

#### Future routing-state resource ownership

**RESOURCE — NORMATIVE:** The budgets in this contract cover only the Company metadata and employer-selection-index machinery defined here.

Any future contract that introduces retained condition-routing state must define and prove its own bounded resource accounting and must preserve the bounded-local-work requirement of `NewSocioeconomicWorld(base)`. It must not retroactively weaken Contract-2 index budgets.

Contract 2 does not charge an undefined routing representation against the 72 MiB formula. If a future routing contract uses no retained state, no additional charge is required.

#### Reference working-space example

**RATIONALE:** One conforming implementation may use a private 16-byte-per-entry copy followed by an eight-pass stable byte-wise LSD radix sort with one 16-byte-per-entry scratch array.

This reference example uses:

```text
additionalRetainedBytes    = 0
otherLiveConstructionBytes = 0
```

It has no uncharged retained weight array, loading buffer, or future raw input. Fixed bookkeeping, including the reference sort's fixed-size counters, fits the charged allowances. For one maximum-size input, this example charges:

```text
raw EmployeeCount staging:  8 * 1,048,576 =  8,388,608 bytes
raw selection input:       16 * 1,048,576 = 16,777,216 bytes
private selection copy:    16 * 1,048,576 = 16,777,216 bytes
radix scratch:             16 * 1,048,576 = 16,777,216 bytes
```

The reference retained charge of 71,303,232 bytes plus those arrays is:

```text
130,023,488 bytes
```

After the 4 KiB fixed allowance, the reference charged peak is:

```text
130,027,584 bytes
```

which is below 128 MiB, or 134,217,728 bytes.

This arithmetic demonstrates feasibility. Radix sorting, eight passes, and this precise scratch layout are not PROTOCOL requirements or identity inputs.

#### Complexity

**RESOURCE — NORMATIVE:**

- Dense Company metadata construction is `O(CompanyCount)`.
- Index validation and cumulative construction are linear in input-entry count.
- Canonicalization must use `O(m log m)` work or better for `m` input entries.
- The reference radix strategy uses `O(m)` work because its pass count is fixed.
- One employer selection is `O(log positive-entry count)`, therefore `O(log CompanyCount)`.
- No Person lookup performs an `O(CompanyCount)` scan.
- No allocation is proportional to the sum of EmployeeCount.
- No one-slot-per-employee representation is permitted.
- Indexes are built serially during construction.

The sorting algorithm and its exact asymptotic choice within this bound are RESOURCE concerns. They are not output-protocol identity when the canonical logical entries remain identical.

### 21. Initialization/failure semantics

**NORMATIVE:** `NewSocioeconomicWorld(base)` establishes, before returning:

- The actual CompanyCount.
- Dense immutable EmployeeCount state.
- Every employer-selection index required by the completed fixed v1 model.
- Relevant Contract-2 model/data identities.
- Final `SocioeconomicWorldID` coverage.

Contract 2 does not require the constructor to precompute or retain condition-to-index routing state. The later Socioeconomic DAG and Company/D2 model contracts own establishment or validation of their deterministic routing mechanism.

Contract-2 construction rejects any required selection index that is malformed or violates this contract’s protocol or resource constraints. All Contract-2 retained storage, including `additionalRetainedBytes`, and all live construction phases, including `otherLiveConstructionBytes`, must pass the complete resource accounting before their potentially excessive allocations. No partially constructed wrapper escapes.

It returns `nil, err` and `errors.Is(err, ErrInvalidSocioeconomicWorld) == true` for:

- Zero CompanyCount.
- CompanyCount above 1,048,576.
- EmployeeCount outside `0..math.MaxUint32`.
- Malformed EmployeeCount source data.
- Malformed or unrepresentable EmployerWeight.
- Positive EmployerWeight for a zero-EmployeeCount Company.
- Duplicate selection entries.
- Zero or out-of-universe CompanyIndex in an index input.
- A complete-universe mode input that omits an operational coordinate.
- Empty or all-zero operationally selectable index.
- Weight-total overflow.
- Count or byte-accounting overflow.
- More than 1,024 required indexes.
- Retained-index, input, retained-state, or construction-peak budget violation.
- A required employer-selection index that is absent or malformed according to the completed model.
- An index-set, subset, or weight derivation inconsistent with the completed fixed model.
- Malformed fixed Company/D2 state.
- Failure to establish required Company-universe or index-model identities.

No new public error category is introduced. Constructor callers have one actionable classification: the fixed socioeconomic World could not be constructed.

**NORMATIVE:** For invalid fixed Company-universe, model, or selection-index configuration handled by `NewSocioeconomicWorld(base)`, the returned constructor error must satisfy all three predicates:

```go
errors.Is(err, ErrInvalidSocioeconomicWorld) == true
errors.Is(err, ErrAllZeroWeights) == false
errors.Is(err, ErrTotalWeightOverflow) == false
```

The lower-level Layer-0 weight-validation sentinels are implementation details in this constructor-validation context. Do not join, wrap, directly return, or otherwise preserve them in a way that makes `errors.Is` match them, including through custom identity matching. Contextual messages may explain an all-zero table or overflow; callers classify invalid fixed socioeconomic state through `ErrInvalidSocioeconomicWorld` only.

**NORMATIVE:** This rule applies only to Contract-2 constructor validation of fixed Company/model/index state. It is not a package-wide error-hiding rule and does not change propagation of identifiable errors from frozen generation operations. Those retain the identities required by frozen Contract 1 (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:568`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:584`).

Unexpected non-configuration failures remain non-nil opaque internal errors under frozen Contract 1.

There is no repair, clamping, dropping, deduplication, weight rescaling, partial wrapper, runtime fallback, or reload API.

### 22. Company generation versus index state

**NORMATIVE:** Contract-2 initialization precomputes:

- The actual CompanyCount.
- One dense immutable EmployeeCount array in CompanyIndex order.
- Every immutable employer-selection index required by the completed v1 model.
- Relevant Contract-2 model/data identities and final `SocioeconomicWorldID` coverage.

The required index set may contain zero, one, or multiple indexes. Contract 2 does not require a global or unconditioned index.

Contract 2 does not freeze whether future condition-to-index resolution is represented by:

- An array.
- A table.
- A map.
- Arithmetic dispatch.
- Hierarchical model state.
- Generated code.
- Another deterministic bounded representation.

No new public routing API is introduced. A later owning contract must establish or validate its routing mechanism independently of the Contract-2 index representation.

Initialization does not need to precompute every final Company value. `Company(index)` remains deterministic Company-scoped generation:

1. Validate the wrapper.
2. Validate operational membership against private CompanyCount.
3. Generate Company ID through the frozen generic identity helper using the base namespace and `"company"`.
4. Read EmployeeCount from immutable position `index-1`.
5. Generate future Company fields through their owning contracts.

The public Company method must use unexported helpers rather than call another exported same-package operation.

Company facts and index state never use:

- Assigned Persons.
- Person generation order.
- Realized occupancy.
- Assignment history.
- PersonRange.
- A mutable employer registry.

The following is explicitly forbidden:

```text
EmployeeCount = realized number of assigned Persons
```

### 23. Employer-index and routing extensibility

**PROTOCOL — NORMATIVE:** This section is the general rule for every employer-selection index, rather than an exception to a mandatory global index.

The completed v1 socioeconomic model may require one or more immutable indexes over deterministic semantic subsets. Every index that exists must:

- Reference only Companies from the same dense universe.
- Use unique CompanyIndex entries.
- Canonicalize to ascending numeric CompanyIndex.
- Preserve each exact CompanyIndex/EmployerWeight pair.
- Use private exact nonnegative integer EmployerWeights.
- Omit zero weights from compact intervals.
- Use the same cumulative-prefix representation.
- Use the same half-open intervals.
- Use the same strict `cumulative > r` search.
- Use one `Uint64n(total)` invocation.
- Have positive total.
- Obey the 1,024-index limit and aggregate input, retained-memory, work, and construction-peak budgets.
- Be fully constructed before the wrapper is returned.
- Have output-affecting subset, criteria, and weight semantics covered by `SocioeconomicWorldID`.

**MODEL — DEFERRED:** The exact index set, conditioning dimensions, semantic subsets, per-index EmployerWeight derivations, condition-to-index routing semantics, and routing representation are owned by the Socioeconomic DAG and Company/D2 model contracts.

**NORMATIVE:** The completed v1 model must define a deterministic resolution from every reachable employee-assignment condition to exactly one required employer-selection index.

Before implementation is considered valid, later owning contracts must ensure:

1. Every reachable employee-assignment condition resolves deterministically.
2. The resolved index exists.
3. The resolved index has positive total weight.
4. The index uses the exact Contract-2 employer-selection protocol.
5. No runtime fallback to an unrelated index exists.
6. Routing semantics that can change Person-to-Company output are covered by `SocioeconomicWorldID`.

Contract 2 does not freeze the routing representation and does not expose a public routing API.

### 24. Concurrency/immutability

**NORMATIVE:** A successfully constructed socioeconomic World and every employer-selection index it owns are immutable and safe for concurrent read-only calls.

There is no:

- Shared advancing RNG.
- Lazy mutable index construction.
- Occupancy counter.
- Assignment registry.
- Per-Company mutex.
- Cache whose population affects output.
- Mutation of cumulative arrays after construction.

Each Person selection receives a fresh deterministic stream under the later stream contract and uses the index resolved by the later model’s deterministic routing semantics.

Repeated, reversed-order, and concurrent generation agree by value. Mutating a returned Company value cannot affect internal EmployeeCount state or later results.

Any retained future routing state must separately satisfy its owning contract’s immutability, concurrency, and bounded-resource rules.

### 25. SocioeconomicWorldID coverage

**NORMATIVE:** Exact encoding remains deferred, but eventual `SocioeconomicWorldID` coverage must include:

- Actual CompanyCount.
- Company-universe model/version.
- EmployeeCount model and data identity.
- The exact output-affecting employer-selection index set.
- Each index’s semantic subset and EmployerWeight model identity.
- Output-affecting condition-to-index routing semantics.
- The Contract-2 cumulative-prefix selector protocol version.
- Ascending CompanyIndex canonical ordering.
- Zero-weight omission semantics.
- Half-open interval semantics.
- Strict binary-search convention.
- Every future output-affecting Company-model choice.

The sorting or canonicalization implementation is not an identity input when it produces the same ordered logical entries and cumulative intervals.

The physical representation of condition routing is not an identity input when it implements identical routing semantics and produces identical output.

Resource guardrails are distinguished from output semantics.

A change to a RESOURCE limit, canonicalization implementation, or routing representation that leaves a World valid under both versions and leaves every generated value, logical interval, and routing result unchanged does not, by itself, require SocioeconomicWorldID drift.

A resource change that enables a newly selected actual model, or accompanies an output-affecting model/protocol change, is covered through that model/protocol identity.

### 26. Compatibility/versioning classification

#### OUTPUT-PROTOCOL SENSITIVE

Changing any of these can change Company facts or the Company selected for a Person and therefore requires model/protocol version and identity consideration:

- Dense membership semantics.
- Actual CompanyCount.
- EmployeeCount model/data.
- The semantic employer-selection index set.
- Index subset membership.
- EmployerWeight derivation.
- Condition-to-index routing semantics.
- Zero-EmployeeCount eligibility rule.
- Zero-weight omission.
- Ascending CompanyIndex order.
- Logical cumulative-prefix representation.
- Half-open interval convention.
- Strict `cumulative > r` predicate.
- One-`Uint64n(total)` selector algorithm.
- Any switch to hierarchical, alias-table, floating-point, or direct-remainder selection.

#### RESOURCE-ONLY

Changing these alters the accepted construction envelope or implementation work but does not change output for Worlds valid under both versions:

- Hard maximum CompanyCount.
- EmployeeCount hard maximum.
- Maximum selection-index count.
- Aggregate input-entry budget.
- Aggregate retained-index budget.
- Retained-state budget.
- Construction-peak budget.
- Additional retained-state and other live construction-state accounting, without changing output for Worlds valid under both versions.
- Canonicalization algorithm, provided it produces identical canonical entries.
- Canonicalization working-space strategy.
- Canonicalization complexity within the frozen resource bound.

#### DEFERRED REPRESENTATION

The physical representation of condition-to-index routing is deferred to the owning DAG/model contract. It is not frozen by Contract 2. If its semantics are unchanged, changing only that representation is not output-protocol sensitive, though its owning resource and API compatibility rules still apply.

The `uint32` EmployeeCount field is also public API shape. Changing its Go type would require API compatibility review even if all common values stayed equal.

### 27. Deferred-design ledger

| Deferred subject | Exact later owner | Required resolution |
| --- | --- | --- |
| Actual numeric CompanyCount | Company/D2 Model Contract | One literal positive value no greater than 1,048,576, with evidence |
| EmployeeCount distribution | Company/D2 Model Contract | Deterministic anchor-date model and data identity |
| Required employer-index set | Socioeconomic DAG and Company/D2 Model Contract | Exact number of indexes and the conditions selecting each one |
| Index semantic subsets | Socioeconomic DAG and Company/D2 Model Contract | Exact deterministic Company membership for each index |
| EmployerWeight derivations | Company/D2 Model Contract | Exact integer derivation and identity for each required index |
| Complete-universe mode usage | Owning later model contract | Whether a particular semantic index must contain every Company |
| Condition-to-index mapping semantics | Socioeconomic DAG and Company/D2 Model Contract | Complete deterministic coverage of reachable assignment conditions; output-sensitive |
| Routing representation | Socioeconomic DAG and Company/D2 Model Contract | Bounded deterministic representation, if retained state is used |
| Routing resource budget | Contract owning retained routing state | Separate bounded accounting without weakening Contract-2 budgets |
| Company fields beyond ID and EmployeeCount | Company Contract | Field semantics, types, models, and resource bounds |
| Company Name | Company Contract | Model and data |
| Industry and SizeBand | Taxonomy & Applicability / Company Contract | Taxonomies and applicability |
| Person-to-Company causal conditioning | Socioeconomic DAG | Permitted dependencies |
| Stream paths and Components | Deterministic Protocol Contract | Exact stream domains, paths, and dependency sets |
| SocioeconomicWorldID encoding | Deterministic Protocol Contract | Exact domain, Components, and textual encoding |
| Empirical Italian enterprise data | D2 contract | Sources, transformations, checksums, and fit requirements |

These are fixed v1 model/data decisions, not future runtime constructor options.

### 28. Normative pseudocode

**NORMATIVE:** All count/byte arguments are exact nonnegative `uint64` values, validated before use or conversion. Their derivations, all additions, and all multiplications are checked. Overflow or an invalid value fails with `ErrInvalidSocioeconomicWorld`; constructor validation never exposes `ErrAllZeroWeights` or `ErrTotalWeightOverflow` through the error chain.

Model-loading callers must validate each prospective source-buffer allocation before creating the arrays passed below. Accepting an already-loaded input does not authorize allocate-then-check loading. Before each allocating phase, call `validateResourceState` with conservative counts and charges for everything that will coexist in that phase. Revalidate before any allocation not covered by that reservation. `additionalRetainedBytes` includes retained storage being allocated, and `otherLiveConstructionBytes` includes all other live or proposed loading/temporary storage.

#### Universe validation

```text
validateUniverse(companyCount, rawEmployeeCounts, additionalRetainedBytes, otherLiveConstructionBytes):

    if companyCount == 0:
        fail ErrInvalidSocioeconomicWorld

    if companyCount > 1,048,576:
        fail ErrInvalidSocioeconomicWorld

    if companyCount > MaxInt:
        fail ErrInvalidSocioeconomicWorld

    if uint64(length(rawEmployeeCounts)) != companyCount:
        fail ErrInvalidSocioeconomicWorld

    validateResourceState(
        companyCount,
        indexCount = 0,
        aggregateInputEntries = 0,
        aggregatePositiveEntries = 0,
        currentInputEntries = 0,
        additionalRetainedBytes = additionalRetainedBytes,
        canonicalizationWorkingBytes = 0,
        otherLiveConstructionBytes = otherLiveConstructionBytes,
    )

    employeeCounts = allocate uint32[integer(companyCount)]

    for position from 0 through integer(companyCount)-1:
        raw = rawEmployeeCounts[position]

        if raw is negative, fractional, not an exact integer,
           or raw > math.MaxUint32:
            fail ErrInvalidSocioeconomicWorld

        employeeCounts[position] = uint32(raw)

    return immutable employeeCounts
```

The later Company/D2 Model Contract supplies `companyCount` and `rawEmployeeCounts`; it does not change this validation. When no additional retained or other live storage exists, use `validateUniverse(companyCount, rawEmployeeCounts, 0, 0)`.

For the reference loading phase with only raw EmployeeCount staging and no index input or additional storage, preflight before allocating that staging buffer is explicitly:

```text
validateResourceState(
    companyCount,
    indexCount = 0,
    aggregateInputEntries = 0,
    aggregatePositiveEntries = 0,
    currentInputEntries = 0,
    additionalRetainedBytes = 0,
    canonicalizationWorkingBytes = 0,
    otherLiveConstructionBytes = 0,
)
```

Any loader needing extra buffers must include their prospective checked charge instead of using the zero-other-storage example.

#### Canonicalization postcondition

```text
canonicalizeSelectionEntries(inputEntries):

    obtain a private representation without mutating inputEntries

    order complete (CompanyIndex, EmployerWeight) pairs
    by ascending numeric CompanyIndex

    if any CompanyIndex occurs more than once:
        fail ErrInvalidSocioeconomicWorld

    verify:
        output length == input length
        every input pair occurs exactly once in output
        every output pair came from input unchanged
        output CompanyIndex values are strictly increasing

    return output
```

The mechanism used to establish this postcondition is deliberately not specified. It must satisfy the frozen resource accounting.

#### Selection-index build

```text
buildSelectionIndex(
    companyCount,
    inputEntries,
    requireCompleteUniverse,
    priorIndexCount,
    priorAggregateInputEntries,
    priorAggregatePositiveEntries,
    employeeCounts,
    additionalRetainedBytes,
    canonicalizationWorkingBytes,
    otherLiveConstructionBytes,
):

    inputCount = uint64(length(inputEntries))

    if companyCount == 0 or companyCount > 1,048,576:
        fail ErrInvalidSocioeconomicWorld

    if inputCount > companyCount:
        fail ErrInvalidSocioeconomicWorld

    if requireCompleteUniverse and inputCount != companyCount:
        fail ErrInvalidSocioeconomicWorld

    total = uint64(0)
    positiveCount = uint64(0)

    for entry in inputEntries:
        company = uint64(entry.company)

        if company == 0 or company > companyCount:
            fail ErrInvalidSocioeconomicWorld

        weight = parseExactUint64(entry.weight)
        if parsing fails:
            fail ErrInvalidSocioeconomicWorld

        if employeeCounts[integer(company-1)] == 0 and weight != 0:
            fail ErrInvalidSocioeconomicWorld

        if weight != 0:
            positiveCount = checkedAdd(positiveCount, 1)

        if weight > math.MaxUint64 - total:
            fail ErrInvalidSocioeconomicWorld

        total += weight

    if total == 0:
        fail ErrInvalidSocioeconomicWorld

    validateResourceState(
        companyCount,
        checkedAdd(priorIndexCount, 1),
        checkedAdd(priorAggregateInputEntries, inputCount),
        checkedAdd(priorAggregatePositiveEntries, positiveCount),
        inputCount,
        additionalRetainedBytes,
        canonicalizationWorkingBytes,
        otherLiveConstructionBytes,
    )

    ordered = canonicalizeSelectionEntries(inputEntries)

    if requireCompleteUniverse:
        for position from 0 through length(ordered)-1:
            if uint64(ordered[position].company) != uint64(position)+1:
                fail ErrInvalidSocioeconomicWorld

    output = allocate employerSelectionEntry[integer(positiveCount)]
    cumulative = uint64(0)
    outputPosition = 0

    for entry in ordered:
        weight = uint64(entry.weight)

        if weight == 0:
            continue

        if weight > math.MaxUint64 - cumulative:
            fail ErrInvalidSocioeconomicWorld

        cumulative += weight
        output[outputPosition] = {
            company: entry.company,
            cumulative: cumulative,
        }
        outputPosition += 1

    if outputPosition != integer(positiveCount) or cumulative != total:
        fail ErrInvalidSocioeconomicWorld

    return immutable employerSelectionIndex{
        entries: output with length and capacity equal to positiveCount,
        total: total,
    }
```

The two additional-state arguments are conservative charges for the builder phase, including storage retained outside this particular index and any future inputs or loading buffers still live. Pass `additionalRetainedBytes = 0` and `otherLiveConstructionBytes = 0` explicitly when neither kind exists. The validator reserves the output positive entries before allocating them, while raw input and all canonicalization working space remain charged.

`requireCompleteUniverse` is a generic validation mode. Contract 2 does not require any caller to set it to true. A later owning model may do so for a specifically defined semantic index.

#### Weighted selection

```text
selectEmployer(rng, index):

    # Construction guarantees index.total > 0 and at least one entry.
    r, err = rng.Uint64n(index.total)
    if err != nil:
        return 0, err

    low = 0
    high = length(index.entries)

    while low < high:
        middle = low + (high-low)/2

        if index.entries[middle].cumulative > r:
            high = middle
        else:
            low = middle + 1

    if low == length(index.entries):
        return 0, opaque internal invariant error

    return index.entries[low].company, nil
```

#### Company membership check

```text
validateCompanyMembership(world, index):

    if world is nil or detectably uninitialized:
        return ErrInvalidSocioeconomicWorld

    if index == 0:
        return ErrInvalidIndex

    if uint64(index) > world.companyCount:
        return ErrCompanyIndexOutOfRange

    return nil
```

#### Resource validation

```text
validateResourceState(
    companyCount,
    indexCount,
    aggregateInputEntries,
    aggregatePositiveEntries,
    currentInputEntries,
    additionalRetainedBytes,
    canonicalizationWorkingBytes,
    otherLiveConstructionBytes,
):

    validate every argument as an exact nonnegative uint64
    require every argument derivation to have used checked arithmetic
    otherwise fail ErrInvalidSocioeconomicWorld

    if companyCount == 0 or companyCount > 1,048,576:
        fail ErrInvalidSocioeconomicWorld

    if indexCount > 1,024:
        fail ErrInvalidSocioeconomicWorld

    if currentInputEntries > companyCount:
        fail ErrInvalidSocioeconomicWorld

    companyBytes =
        checkedAdd(64, checkedMultiply(4, companyCount))

    retainedIndexBytes =
        checkedAdd(
            64,
            checkedAdd(
                checkedMultiply(32, indexCount),
                checkedMultiply(16, aggregatePositiveEntries),
            ),
        )

    inputBytes =
        checkedAdd(
            64,
            checkedAdd(
                checkedMultiply(32, indexCount),
                checkedMultiply(16, aggregateInputEntries),
            ),
        )

    if retainedIndexBytes > 64 MiB:
        fail ErrInvalidSocioeconomicWorld

    if inputBytes > 64 MiB:
        fail ErrInvalidSocioeconomicWorld

    retainedBytes =
        checkedAdd(
            checkedAdd(companyBytes, retainedIndexBytes),
            additionalRetainedBytes,
        )

    if retainedBytes > 72 MiB:
        fail ErrInvalidSocioeconomicWorld

    rawEmployeeCountStagingBytes = checkedMultiply(8, companyCount)
    currentRawInputBytes = checkedMultiply(16, currentInputEntries)

    peakBytes =
        checkedAdd(
            retainedBytes,
            checkedAdd(
                rawEmployeeCountStagingBytes,
                checkedAdd(
                    currentRawInputBytes,
                    checkedAdd(
                        canonicalizationWorkingBytes,
                        checkedAdd(otherLiveConstructionBytes, 4 KiB),
                    ),
                ),
            ),
        )

    if peakBytes > 128 MiB:
        fail ErrInvalidSocioeconomicWorld

    succeed
```

Every `checkedAdd` and `checkedMultiply` fails on overflow before returning a value, producing the constructor-validation classification from section 21. All integer length/index conversions are validated before allocation. `companyBytes` is the pseudocode name for `companyMetadataBytes`; `inputBytes` is the name for `selectionInputBytes`.

`additionalRetainedBytes` and `otherLiveConstructionBytes` are checked totals of the disjoint structures defined in section 20. `canonicalizationWorkingBytes`, together with fixed control bookkeeping charged once in the 4 KiB allowance, conservatively covers the selected implementation's complete private working storage. The raw-staging charge is a conservative reservation, including after release; additional live buffers cannot be hidden in it. All phase charges include prospective allocations before they occur.

Condition-routing state is intentionally absent from this pseudocode. Its later owning contract must define separate resource validation if it retains such state.

### 29. Arithmetic/resource proof

1. **Slice and index conversions**

   CompanyCount is at most 1,048,576, below `math.MaxInt32`. CompanyCount, `index-1`, input lengths, positive-entry counts for one index, and slice positions therefore convert safely to `int` on 32-bit and 64-bit Go architectures.

2. **Selection-index size**

   One index retains at most CompanyCount positive entries. Its maximum entry payload is:

   ```text
   16 * 1,048,576 = 16,777,216 bytes
   ```

   All retained indexes together must satisfy the exact 64 MiB accounting formula.

3. **Weight summation**

   Every addition checks `weight > math.MaxUint64-total` first. A model whose sum would overflow is invalid and never reaches cumulative construction.

4. **Cumulative totals**

   A valid total lies in `1..math.MaxUint64`. Cumulative values are exact checked partial sums and therefore fit `uint64`. The final cumulative value is verified against the preflight total.

5. **No EmployeeCount-sum allocation**

   EmployeeCount uses one four-byte retained value per Company and, at construction, at most one bounded eight-byte raw staging value per Company. No operation sums EmployeeCounts to determine allocation length. Employer indexes retain one entry per positive-weight Company, never one entry per modeled employee.

6. **Lookup complexity**

   Selection performs one bounded draw and binary search over at most CompanyCount positive entries. It is `O(log CompanyCount)` and performs no linear Company scan.

7. **Aggregate retained memory**

   At maximum CompanyCount, dense EmployeeCount metadata costs 4,194,368 charged bytes. Adding the complete 64 MiB retained-index budget yields 71,303,232 bytes, below 72 MiB, when `additionalRetainedBytes = 0`.

   Every other retained Contract-2 structure contributes exactly once through `additionalRetainedBytes`; checked addition enforces `retainedBytes <= 75,497,472`. The baseline allowance remaining is 4,194,240 bytes. An additional retained charge of 4,194,241 bytes produces 75,497,473 bytes and therefore rejects before the additional allocation. It cannot bypass the check as implementation-specific auxiliary state.

8. **Construction memory**

   Every conforming implementation supplies conservative `additionalRetainedBytes`, `canonicalizationWorkingBytes`, and `otherLiveConstructionBytes` charges. The disjoint phase accounting covers every simultaneously-live Contract-2 allocation once, including proposed allocations. Checked arithmetic and rejection precede excessive allocation.

   The non-normative radix example uses 33,554,432 canonicalization working bytes for one maximum-size private copy and scratch array, with `additionalRetainedBytes = 0` and `otherLiveConstructionBytes = 0`. Raw EmployeeCount staging, raw input, that working space, and the reference retained charge total 130,023,488 bytes before overhead. Adding the fixed 4 KiB allowance gives 130,027,584 bytes, below 134,217,728 bytes.

   Keeping an otherwise-valid 8 MiB decoded loading buffer live during that reference phase sets `otherLiveConstructionBytes = 8,388,608`. The charged peak becomes 138,416,192 bytes, above 128 MiB, so the prospective phase rejects before an allocation would cross the limit. Releasing that buffer before the next phase removes its other-live charge only after it is no longer live or reachable for construction use. The same rule applies to live future raw inputs: their additional buffers enter this term even when their semantic entries fit the aggregate input budget.

   These are normative logical charges, not an RSS prediction. Smaller physical storage does not waive a prescribed formula; additional storage must receive an explicit charge.

9. **Index-set boundedness**

   The completed model may define at most 1,024 indexes. Both zero and positive input entries count against the 64 MiB aggregate selection-input budget, preventing an unbounded number of large all-zero semantic inputs. That aggregate semantic-volume bound remains separate from simultaneously-live storage; future inputs retained during a current build are also peak-charged without counting the same live buffer twice.

10. **Routing-state separation**

   No undefined routing representation is included in the 72 MiB Contract-2 formula. A future retained routing representation is valid only after its owner proves a separate finite bound and preserves bounded constructor work. A stateless deterministic resolution needs no retained-state charge.

### 30. Required future tests

No tests are implemented by this contract.

#### CompanyCount

Tests must cover through constructor validation, Company membership behavior, and same-package internal inspection where necessary:

- Zero rejected during construction.
- Exact hard maximum accepted by resource validation.
- Maximum plus one rejected before Company-sized allocation.
- Internal count equal to the later prescribed model value.
- Stability across ReferenceDate and FaceCatalog variations.
- Company membership accepted at 1 and CompanyCount.
- Zero and CompanyCount+1 rejected with frozen error identities.
- Protocol decoding outside the universe remaining distinct from membership.
- No public CompanyCount accessor, iterator, public field, or bulk enumeration method.

#### EmployeeCount

Tests must cover:

- Zero accepted.
- `math.MaxUint32` accepted.
- A source value above `math.MaxUint32` rejected before conversion.
- Every operational Company having a value.
- Stability at `PopulationAnchorDate`.
- ReferenceDate and FaceCatalog independence.
- Person-order, PersonRange, occupancy, and assignment-history independence.
- A zero EmployeeCount Company rejecting a positive EmployerWeight.

#### Index construction

Tests must cover:

- Arbitrary input permutations producing identical compact entries and totals.
- Canonical entries ordered by ascending CompanyIndex.
- Weights remaining attached to their original CompanyIndex.
- Duplicate rejection, including duplicate zero-weight entries.
- Zero and above-universe CompanyIndex rejection.
- Optional complete-universe mode accepting a complete input and rejecting an omission.
- Ordinary subset mode allowing omitted Companies.
- Zero weights omitted without consuming intervals.
- Empty and all-zero operational-index rejection.
- A sole weight of `math.MaxUint64` accepted.
- `{math.MaxUint64, 1}` rejected as overflow.
- Aggregate resource rejection before excessive allocation.
- Input and retained-entry accounting including zero-weight inputs.
- Index-count limit.
- Exact-capacity immutable output arrays.
- Equivalent canonical indexes selecting identically.
- Alternative conforming canonicalization implementations, if present, producing identical logical entries and cumulative totals.

No test may require radix sorting, eight passes, or a particular sorting implementation.

#### Additional retained state

Required tests must establish:

- Zero additional retained bytes preserves baseline behavior.
- Additional retained bytes enter the 72 MiB check and also the peak through `retainedBytes` exactly once.
- The retained-limit example in section 29 rejects before the additional allocation.
- Retained EmployerWeight or other auxiliary arrays cannot bypass accounting by being called implementation details.
- Checked addition rejects overflowing additional-state charges before allocation.

#### Other live construction state

Required tests must establish:

- Zero other-live bytes preserves baseline behavior.
- Decoded model-loading, parsing, and still-live future-input buffers contribute to peak.
- Simultaneous extra raw input buffers that push a phase above 128 MiB reject before the excessive allocation even when aggregate semantic input is within 64 MiB.
- The loading-buffer peak example in section 29 rejects.
- Releasing a buffer before a later phase removes its live charge only when it is no longer live or reachable for construction use.
- No buffer/backing storage is counted twice and no live Contract-2 construction buffer is uncharged.
- Semantic aggregate-input volume is not substituted for phase-specific live-memory accounting.
- All count/byte derivations and checked sums/products reject overflow before allocation.
- Allocation instrumentation or equivalent evidence proves that budget rejection precedes the allocation that would cross the limit, including source/model loading.

#### Constructor error identities

For representative invalid fixed Company/model/index states, require a nil wrapper and a non-nil error satisfying `errors.Is(err, ErrInvalidSocioeconomicWorld) == true`.

Test all-zero weights and weight-total overflow independently. In each case, assert both negative identities as well:

```go
errors.Is(err, ErrAllZeroWeights) == false
errors.Is(err, ErrTotalWeightOverflow) == false
```

Contextual constructor wrapping must preserve the invalid-world identity without exposing either lower-level weight sentinel. Generation-error propagation tests required by frozen Contract 1 remain unchanged and continue to prove the specified frozen identities.

Preserve independent Company membership tests: on a valid wrapper, `CompanyIndex == 0` returns `ErrInvalidIndex`, and `CompanyIndex > CompanyCount` returns `ErrCompanyIndexOutOfRange`.

#### Selector golden

Use this literal unordered index input with CompanyCount 4:

```text
(CompanyIndex 3, EmployerWeight 2)
(CompanyIndex 1, EmployerWeight 3)
(CompanyIndex 4, EmployerWeight 0)
(CompanyIndex 2, EmployerWeight 1)
```

This fixture happens to mention all four Companies but does not require a complete-universe index in the model.

The required compact index is:

| Position | CompanyIndex | Cumulative | Interval |
| ---: | ---: | ---: | --- |
| 0 | 1 | 3 | `[0,3)` |
| 1 | 2 | 4 | `[3,4)` |
| 2 | 3 | 6 | `[4,6)` |

Every draw boundary must produce:

| Draw `r` | Expected CompanyIndex |
| ---: | ---: |
| 0 | 1 |
| 1 | 1 |
| 2 | 1 |
| 3 | 2 |
| 4 | 3 |
| 5 | 3 |

Frozen SplitMix64 initial states that exercise each first draw under `Uint64n(6)` are:

| Desired draw | Initial state |
| ---: | --- |
| 0 | `0x0000000000000013` |
| 1 | `0x0000000000000000` |
| 2 | `0x0000000000000005` |
| 3 | `0x0000000000000003` |
| 4 | `0x0000000000000002` |
| 5 | `0x0000000000000001` |

Starting from state zero, ten repeated selections must produce:

```text
draws:     [1, 0, 1, 4, 1, 0, 5, 2, 5, 2]
companies: [1, 1, 1, 3, 1, 1, 3, 1, 3, 1]
```

Also test:

- `total == 1`.
- First and last interval.
- Every interval boundary.
- Reverse-order Person generation.
- Repeated fresh-stream equivalence.
- Concurrent equivalence.
- No promise of one raw `Uint64` call.

#### Complexity and invariants

Tests, benchmarks, or call-graph assertions must establish:

- No slot expansion by EmployeeCount or EmployerWeight.
- No `O(CompanyCount)` scan per selection.
- Binary-search behavior as index size grows.
- Canonicalization uses `O(m log m)` work or better.
- Actual construction remains within frozen resource limits.
- No mutable registry or occupancy state.
- Zero indexes are permitted only when the completed model has no reachable employee-assignment condition.

#### Future routing obligations

The later owning contracts must specify tests proving:

- Every reachable employee-assignment condition resolves deterministically.
- Every reachable condition resolves to exactly one index.
- The resolved index exists and has positive total weight.
- Every resolved index conforms to this Contract-2 selector protocol.
- Equivalent routing semantics produce equivalent Person-to-Company results.
- Missing, ambiguous, or zero-total routing fails validation.
- No runtime fallback to an unrelated index.
- Retained routing state, if any, obeys its separately frozen resource budget.

These tests must validate semantics and complete coverage without prescribing an array, table, map, arithmetic dispatch, hierarchy, generated-code layout, or other representation.

#### Compatibility

Tests must establish:

- A deliberately changed weight mapping changes deterministic assignments for a crafted draw.
- Input permutation does not change logical intervals or assignment.
- Changing the canonicalization implementation does not change logical intervals or assignment.
- Company generation order does not change Companies or indexes.
- Person materialization order does not change assignment.
- Zero entries do not shift positive intervals.
- Replacing strict `>` with `>=`, changing canonical order, or using hierarchical selection fails the golden.
- Missing-index conditions fail validation rather than falling back at runtime.

#### Identity coverage

Future identity tests must establish:

- Changing actual CompanyCount perturbs `SocioeconomicWorldID`.
- Changing EmployeeCount model/data identity perturbs it.
- Changing the semantic index set perturbs it when output-affecting.
- Changing index subset or EmployerWeight model identity perturbs it.
- Changing condition-to-index routing semantics perturbs it when output-affecting.
- Changing selector protocol semantics perturbs it.
- Changing only canonicalization implementation does not perturb it when logical entries are identical.
- Changing only routing representation does not perturb it when routing semantics and output are identical.
- A resource-only guardrail change leaves identity unchanged for a World valid under both versions.
- ReferenceDate and FaceCatalog changes leave it unchanged.

No empirical-fit test belongs to this contract.

### 31. Decision ledger

| Subject | Decision | Class | Frozen now? | Later owner if deferred |
| --- | --- | --- | ---: | --- |
| Dense Company universe | Operational coordinates are exactly `1..CompanyCount` | PROTOCOL | Yes | None |
| Actual CompanyCount | One literal positive v1 model value, not runtime-configurable | MODEL | No | Company/D2 Model Contract |
| CompanyCount hard max | `1,048,576` | RESOURCE | Yes | None |
| CompanyCount public accessor | None; frozen Contract-1 public operation set is unchanged | API | Yes | None |
| CompanyCount internal state | Private immutable dense-universe cardinality used by membership validation | PROTOCOL / MODEL | Yes, except numeric value | Company/D2 Model Contract for value |
| EmployeeCount type | Public `uint32` field | API | Yes | None |
| EmployeeCount hard range | `0..math.MaxUint32`; every operational Company has one | RESOURCE | Yes | None |
| EmployeeCount distribution | No distribution selected | MODEL | No | Company/D2 Model Contract |
| Required employer-index set | Contract 2 requires no specific semantic index; completed model may require zero, one, or more | MODEL / DAG | No | Socioeconomic DAG and Company/D2 Model Contract |
| EmployerWeight abstraction | Private exact nonnegative `uint64` interval weight | PROTOCOL | Yes | None |
| EmployerWeight derivations | Deterministic per-index fixed models; equality with EmployeeCount is not assumed | MODEL | No | Company/D2 Model Contract |
| Eligibility | Positive weight eligible; zero/absent ineligible; existence unchanged | PROTOCOL | Yes | None |
| Complete-universe mode | Optional generic validation mode; no such index is required here | PROTOCOL / MODEL | Builder semantics only | Owning later model contract |
| Canonical order | Strictly ascending numeric CompanyIndex | PROTOCOL | Yes | None |
| Canonicalization mechanism | Any conforming bounded implementation producing the exact canonical order | RESOURCE | Yes | None |
| Index representation | Immutable positive-entry cumulative-prefix array | PROTOCOL | Yes | None |
| Selection algorithm | One `Uint64n(total)` plus first cumulative `> r` binary search | PROTOCOL | Yes | None |
| Overflow semantics | Checked uint64 addition; valid total `1..math.MaxUint64`; no repair | PROTOCOL | Yes | None |
| Aggregate memory bound | 64 MiB indexes, 72 MiB retained Contract-2 state, 128 MiB construction peak | RESOURCE | Yes | None |
| Additional retained Contract-2 state | Explicit checked `additionalRetainedBytes` in retained and peak totals; no uncharged retained structures | RESOURCE | Yes | None |
| Other live construction/model-loading state | Explicit checked `otherLiveConstructionBytes` phase charge; no uncharged live buffers | RESOURCE | Yes | None |
| Allocation preflight | Complete conservative phase accounting before any potentially excessive allocation | RESOURCE | Yes | None |
| Constructor lower-level weight-error visibility | Hidden for Contract-2 constructor validation; only invalid-world classification; frozen generation propagation unchanged | ERROR CONTRACT | Yes | None |
| Condition-to-index mapping semantics | Deferred; deterministic complete reachable-condition resolution; output-sensitive | MODEL / DAG | No | Socioeconomic DAG and Company/D2 Model Contract |
| Routing representation | No representation frozen and no public API added | RESOURCE / IMPLEMENTATION | No | Owning Socioeconomic DAG/model contract |
| Routing resource accounting | Separate finite budget required only if later contract retains routing state | RESOURCE | No value frozen here | Contract owning retained routing state |
| Company extra fields | Only EmployeeCount is added here; every specific additional field is deferred | API / MODEL | EmployeeCount only | Company Contract |
| Stream paths | No path or dependency Component selected | PROTOCOL | No | Deterministic Protocol Contract |
| Identity encoding | Coverage fixed; exact encoding and component names deferred | PROTOCOL | No | Deterministic Protocol Contract |

### 32. Independent-review questions

1. Did Contract 2 invent any unsupported empirical MODEL value?
2. Is actual CompanyCount ownership unambiguous while CompanyCount remains private internal state?
3. Are RESOURCE limits safe and justified by arithmetic?
4. Are EmployeeCount and EmployerWeight properly separated?
5. Is existence distinct from eligibility?
6. Is the selection algorithm fully deterministic and sufficiently specified?
7. Can two implementations build identical intervals?
8. Are zero-weight/all-zero semantics precise?
9. Can any valid input overflow uint64?
10. Can any valid maximum cause unacceptable allocation?
11. Is per-Person employer lookup free of O(CompanyCount) scanning?
12. Does the protocol avoid occupancy coupling?
13. Can later contracts define complete deterministic condition-to-index resolution without Contract 2 freezing its representation?
14. Is SocioeconomicWorldID coverage correctly classified for routing semantics versus representation?
15. Does Contract 2 preserve the frozen Contract-1 public operation set and avoid taxonomy/DAG/income scope?
16. Does `retainedBytes` charge every retained Contract-2 structure exactly once?
17. Does `peakBytes` charge every simultaneously-live Contract-2 construction allocation exactly once?
18. Can any buffer evade the peak by being called “model loading” or “future input”?
19. Are semantic aggregate-input volume and live-memory accounting clearly distinct?
20. Can all resource checks happen before excessive allocation?
21. Are `ErrAllZeroWeights` and `ErrTotalWeightOverflow` definitely hidden from Contract-2 constructor error chains?
22. Are frozen generation errors unaffected?

### 33. Contract v0 verdict

**NORMATIVE:** This contract selects one coherent reusable Company-universe and employer-selection-index protocol.

It freezes a dense one-based universe, private immutable CompanyCount state, a `uint32` anchor-date EmployeeCount, a distinct private `uint64` EmployerWeight, ascending Company order, immutable cumulative-prefix indexes, and selection by one frozen `Uint64n(total)` invocation followed by strict-upper-bound binary search.

It adds no public `SocioeconomicWorld` method. Frozen Contract 1’s complete conceptual public operation set remains unchanged. The only public structural extension is `Company.EmployeeCount`, which Contract 1 explicitly allowed later contracts to define.

It does not require an unconditioned or complete-universe employer index. The completed socioeconomic model owns the exact index set, conditioning dimensions, semantic subsets, EmployerWeight derivations, condition-to-index routing semantics, and routing representation. Every index that model creates must use this contract’s exact logical build and selection protocol.

Contract 2 does not require precomputed routing state. Later owning contracts must establish or validate complete deterministic resolution for every reachable employee-assignment condition. Any retained routing representation must receive its own bounded resource contract without weakening these Company/index budgets.

Canonical ordering is compatibility-sensitive. The sorting implementation used to obtain it is not, provided it preserves exact pairs, rejects duplicates, produces identical logical entries, and obeys the frozen resource limits.

The actual CompanyCount, EmployeeCount distribution, required index set, EmployerWeight derivations, and routing semantics remain with later contracts because selecting them requires empirical or DAG/model evidence. No empirical value was invented.

Self-review found:

- No new public `SocioeconomicWorld` method.
- No public CompanyCount accessor, replacement iterator, public field, or bulk API.
- No mandatory global or unconditioned employer index.
- No universally required complete-universe weight table.
- No frozen routing representation or mandatory precomputed routing state.
- No undefined future routing state silently charged as free or unbounded.
- No radix or eight-pass compatibility requirement.
- No sorting implementation included in protocol identity.
- No RESOURCE limit presented as Italian enterprise realism.
- No EmployeeCount/EmployerWeight conflation.
- No existence/eligibility conflation.
- No map-order dependence.
- No unchecked valid-model weight sum.
- No uncharged additional retained state or live construction/model-loading buffer.
- No conflation of semantic aggregate input volume with live-memory accounting.
- No excessive allocation before the relevant checked resource validation.
- No visible lower-level weight sentinel in Contract-2 constructor validation errors.
- No change to frozen generation-error propagation.
- No per-Person Company scan.
- No allocation proportional to modeled workforce.
- No occupancy or assignment-history coupling.
- No taxonomy, DAG, geography, income, or stream-path decision.
- No contradiction with frozen Contract 1.

No Layer 2 source, data, or tests are implemented by this specification.

## Source-backed risks

- Frozen `WeightedIndex` performs a linear interval scan after drawing (`tools/fakedata/random.go:92-100`). Calling it directly with CompanyCount weights would violate the logarithmic employer-lookup requirement.
- Layer 1’s place selector uses two hierarchical weighted selections (`tools/fakedata/layer1.go:479-493`). Reusing that shape would change RNG consumption and interval mapping.
- `Uint64n` rejection sampling may consume multiple raw RNG outputs (`tools/fakedata/random.go:59-65`). Tests must assert final selection behavior rather than a universal one-raw-draw assumption.
- Protocol Company decoding has no operational CompanyCount (`tools/fakedata/identity.go:134-165`). Membership must remain a wrapper-level check using private socioeconomic-world state.
- The no-context constructor is coherent only while later Company/D2 loading and the chosen canonicalization implementation respect complete resource accounting and bounded work. Repository precedent validates allocation arithmetic before use (`tools/fakedata/component.go:139-148`). Additional retained structures must enter `additionalRetainedBytes`; decoded loading buffers and simultaneous future inputs must enter `otherLiveConstructionBytes`. The aggregate semantic-input budget alone proves no live-memory bound.
- Layer 0 exposes all-zero and overflow weight identities (`tools/fakedata/random.go:79-85`, `tools/fakedata/errors.go:9`, `tools/fakedata/errors.go:22`). Reusing validation must not leak those identities through a Contract-2 constructor error chain. Frozen generation operations still preserve their specified identities (`SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:568`).
- A later model that can generate employees must supply complete deterministic coverage of its reachable assignment conditions. This contract intentionally supplies neither a fallback index nor a routing representation.
- A future retained routing representation requires its own bounded resource proof; omission from the current 72 MiB calculation is not permission for unbounded state.
- Future semantic indexes remain compatible only if they use the exact canonical cumulative protocol and include their output-affecting subset, weight, and routing semantics in `SocioeconomicWorldID`.

## Deferred decisions

The Company/D2 Model Contract must provide the actual CompanyCount and EmployeeCount model/data.

The Socioeconomic DAG and Company/D2 model contracts must define the exact employer-index set, conditioning dimensions, semantic subsets, EmployerWeight derivations, condition-to-index routing semantics, and routing representation.

If those contracts introduce retained routing state, they must define a separate finite resource budget and preserve bounded local construction. A stateless deterministic routing mechanism requires no retained-state charge.

The Company Contract owns fields beyond ID and EmployeeCount. Taxonomy, income, geography, stream paths, dependency Components, and exact identity encoding remain with their already named later contracts.

Missing D2 evidence is not a blocker for this mechanical protocol.

## Repository-integrity proof

The rectification preserved the frozen repository baseline and did not modify Layer 0, Layer 1, frozen Contract 1, the source Contract 2, or the pre-existing `tools/types/semantic.go` working-tree change.

The specification text above is the output of the read-only rectification. Any subsequent placement or naming of that text as a repository-root review artifact is outside the drafting task and does not alter the normative Contract-2 content.

Checks performed for this rectification establish:

* Branch before/after comparison passed: `property-semantics`.
* HEAD before/after comparison passed: `76fa0fff109ca6abad62f55c8761275455842dc5`.
* Frozen Contract 1 SHA-256 before/after comparison passed and matched `bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6`.
* Frozen Contract 1 before/after status comparison passed.
* The pre-existing `tools/types/semantic.go` before/after SHA-256 and working-tree diff comparisons passed. Its tracked modification remains present.
* The source Contract 2 before/after SHA-256 comparison passed.
* Index before/after comparison and `git diff --cached --quiet` passed.
* `git diff --check` passed.
* `GOCACHE=/tmp/krenalis-l2-c2-astra-fix-gocache go test ./tools/fakedata` passed.
* Selector semantics, selector pseudocode, interval goldens, SplitMix64 states, and repeated-draw vectors were preserved against the source contract.
* Task-created temporary Go cache and integrity artifacts were removed.
* Nothing was staged or committed.

L2 DETAILED CONTRACT 2 v0 COMPLETE — READY FOR INDEPENDENT REVIEW
