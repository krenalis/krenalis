# Synthetic World v1 — Layer 2 Detailed Contract 1 v0

## API, Identity & Activity State

### 1. Status and frozen baseline

**Status:** revised proposed normative contract v0, ready for independent review. This document is not a freeze approval or an implementation.

Verified baseline:

```text
Branch: property-semantics
HEAD:   76fa0fff109ca6abad62f55c8761275455842dc5
Subject: Add Synthetic World Layer 1
```

Verified relevant history:

```text
76fa0fff1 Add Synthetic World Layer 1
1a107bf7f Add deterministic fakedata foundation
a2f031948 Add frozen Italian empirical data
b79b87734 Add curated Italian plausibility data
ae6de32d3 Add fakefacegen fixture support
2674a61e4 Normalize phone semantic values to E.164
```

**NORMATIVE:** Layer 2 shall be implementable through new same-package files in `tools/fakedata`. It shall not require changes to frozen `Person`, `World`, `WorldConfig`, `World.Person`, Layer 0 identity or stream protocols, Layer 1 WorldID/SnapshotID semantics, or Layer 1 goldens.

All source citations are repository-relative `file:line` references to the tracked baseline above. The unrelated working-tree modification to `tools/types/semantic.go` supplies no normative precedent.

This is a specification-only task. The revised specification is returned in this response and is not persisted into the repository. The specification file created earlier in this session has been removed; the actual integrity checks are reported in the appendix.

### 2. Scope / non-scope

This contract resolves exactly seven decisions:

1. Public domain names and primary types.
2. Wrapper ownership and constructor.
3. Composite and facts-only Person access.
4. Company coordinate, identity, and membership APIs.
5. Stable socioeconomic-world identity.
6. PrimaryActivity and branch payload representation.
7. Validation, errors, immutability, and API invariants.

It does not define distributions, empirical crosswalks, datasets, CompanyCount, resource maxima, EmployeeCount representation, employer weights, selection algorithms, occupational field applicability, income, Company names, geography conditioning, stream paths, or identity encoding.

**NORMATIVE:** “must” and “shall” denote requirements. SOURCE FACT identifies frozen evidence; RATIONALE explains a decision; DEFERRED identifies work owned by later contracts.

The skeletons define the public structural boundary. Referenced taxonomy types and explicitly omitted occupational, runtime, or Company fields remain subject to their named future contracts. The skeletons are not complete, compilable implementation files.

### 3. Source constraints

| SOURCE FACT | Frozen evidence | Contract consequence |
| --- | --- | --- |
| `Person` contains Layer 1 facts, CurrentAge, and PhotoID. | `tools/fakedata/layer1.go:104` | Use named containment; do not extend Person. |
| WorldConfig contains namespace, seed, reference date, and FaceCatalog; World retains private runtime state. | `tools/fakedata/layer1.go:119`, `tools/fakedata/layer1.go:127` | Accept an existing World as authoritative. |
| World.Person delegates to unexported `person`. | `tools/fakedata/layer1.go:234` | Composite retrieval can reuse the frozen implementation. |
| Person generation rejects zero and indices above Layer1MaxPersonIndex. | `tools/fakedata/layer1.go:496` | Both socioeconomic Person paths retain those bounds and errors. |
| `birth` returns AnchorAge separately and anchors birth generation to 2026-01-01. | `tools/fakedata/layer1.go:355` | Facts can obtain AnchorAge without full Person generation. |
| Full Person generation selects a photo; selection scans catalog entries. | `tools/fakedata/layer1.go:537`, `tools/fakedata/layer1_photo.go:276` | Facts-only access has a concrete computational and semantic purpose. |
| Generic entity identity helpers accept an entity kind and raw uint64 coordinate. | `tools/fakedata/identity.go:111`, `tools/fakedata/identity.go:135` | Company identity needs no new codec or PersonIndex conversion. |
| Frozen tests establish a Company forward/inverse golden. | `tools/fakedata/identity_test.go:314` | The public Company functions preserve that mapping. |
| Generic entity streams derive fresh RNG values from kind and raw coordinate. | `tools/fakedata/stream.go:36` | Company streams require no Layer 0 change. |
| Stable WorldID composition excludes reference date and catalog; SnapshotID composition includes them. | `tools/fakedata/layer1.go:188`, `tools/fakedata/layer1.go:195` | Preserve stable identity versus snapshot identity. |

Applicable repository rules require exported argument validation, prohibit direct exported-to-exported calls within the same package, require full parameter lists on one line, and require documentation of identifiable errors. See `AGENTS.md:36`, `AGENTS.md:64`, `AGENTS.md:168`, and `AGENTS.md:253`.

Generation APIs shall delegate through unexported implementation helpers where required. Frozen helpers are reused as they stand; this contract does not request retroactive cleanup of frozen code.

### 4. Public naming decisions

**NORMATIVE:** adopt these final v0 names:

| Concept | Public name |
| --- | --- |
| Socioeconomic world wrapper | `SocioeconomicWorld` |
| Composite Person observation | `SocioeconomicPerson` |
| Anchor-date socioeconomic facts | `SocioeconomicFacts` |
| Stable socioeconomic-world identity | `SocioeconomicWorldID` |
| Company value | `Company` |
| Company coordinate | `CompanyIndex` |
| Mutually exclusive primary activity | `PrimaryActivity` |
| Employee branch payload | `EmployeeFacts` |
| Self-employed branch payload | `SelfEmployedFacts` |

**RATIONALE:** these names describe the domain and distinguish the composite from frozen Person. No public name exposes the implementation layer number.

**DEFERRED:** supporting taxonomy definitions, completion of occupational payloads, and additional Company or income fields.

### 5. Core public type skeletons

**NORMATIVE structural skeleton:**

```go
// SocioeconomicWorld provides deterministic socioeconomic random access.
type SocioeconomicWorld struct {
    base *World
    // Private immutable D2, Company-universe/index, and identity state omitted.
}

// SocioeconomicPerson combines a Layer 1 Person with anchor-date facts.
type SocioeconomicPerson struct {
    Person Person
    Facts  SocioeconomicFacts
}

// SocioeconomicWorldID identifies a stable socioeconomic configuration.
type SocioeconomicWorldID string

// Company contains the generated facts for an operational Company.
type Company struct {
    ID string
    // Additional Company fields belong to later contracts.
}

// CompanyIndex is a one-based coordinate for a synthetic Company.
type CompanyIndex uint64

// SocioeconomicFacts contains socioeconomic state at PopulationAnchorDate.
type SocioeconomicFacts struct {
    Education       EducationLevel
    PrimaryActivity PrimaryActivity
    Employee        *EmployeeFacts
    SelfEmployed    *SelfEmployedFacts
}

// EmployeeFacts contains current employee-specific facts at the anchor.
type EmployeeFacts struct {
    Employer CompanyIndex
    // Occupational facts are completed by Taxonomy & Applicability.
}

// SelfEmployedFacts contains self-employment facts at the anchor.
type SelfEmployedFacts struct {
    // Occupational facts are completed by Taxonomy & Applicability.
}

// PrimaryActivity identifies the primary activity at the anchor.
type PrimaryActivity string
```

`EducationLevel` refers to a future taxonomy-owned type. Its structural location is fixed here; its underlying representation and values are not.

For occupational facts, Contract 1 freezes only the activity branch envelope and the mandatory employee employer reference. It does not freeze JobFamily fields as always-present values in either branch, nor `JobLevel *JobLevel` as the final optionality representation.

**NORMATIVE:** Taxonomy & Applicability shall extend and complete the occupational payload contract before implementation, including JobFamily/JobLevel fields, representations, and applicability.

The skeletal `SelfEmployedFacts` declaration expresses the branch envelope; it does not prescribe an empty final occupational payload. No empty placeholder taxonomy declarations shall be introduced to make this incomplete skeleton compile.

The Company skeleton fixes only ID. It does not define an otherwise empty final Company model. No CompanyIndex field is duplicated in Company: the lookup coordinate is already known, and the ID is reversible.

The complete conceptual public operation set is:

```go
func NewSocioeconomicWorld(base *World) (*SocioeconomicWorld, error)

func (w *SocioeconomicWorld) Company(index CompanyIndex) (Company, error)
func (w *SocioeconomicWorld) Facts(index PersonIndex) (SocioeconomicFacts, error)
func (w *SocioeconomicWorld) ID() SocioeconomicWorldID
func (w *SocioeconomicWorld) Person(index PersonIndex) (SocioeconomicPerson, error)

func CompanyIDForIndex(namespace IdentityNamespace, index CompanyIndex) (string, error)
func CompanyIndexFromID(namespace IdentityNamespace, id string) (CompanyIndex, error)
```

These signatures are contract notation. Future source declarations must carry the documentation and identifiable-error descriptions specified below and follow repository declaration ordering.

### 6. Socioeconomic-world ownership

**NORMATIVE:** each constructed wrapper retains one frozen base World and owns:

- Immutable D2 runtime state.
- Immutable finite Company-universe state and its selection index.
- One stable SocioeconomicWorldID.

The supplied World is authoritative for namespace, seed-derived runtime state, ReferenceDate, WorldID, and SnapshotID. No constructor accepts independent copies of those inputs.

Layer 2 may read private frozen state and call existing unexported helpers because it remains in `tools/fakedata`. World retains `namespace`, `worldID`, `snapshotID`, `streams`, and `data`; these provide the relevant extension points. SOURCE FACT: `tools/fakedata/layer1.go:127`.

Layer 2 shall not mutate the base. Callers shall likewise treat the supplied World as immutable, including not overwriting its struct through the retained pointer. Concurrent external mutation of the base is outside the API contract.

Immutable internal tables may be shared. No mutable runtime state may be exposed to callers.

**RATIONALE:** one authoritative base prevents mismatched namespace, seed, and snapshot inputs.

### 7. Constructor

**NORMATIVE:**

```go
func NewSocioeconomicWorld(base *World) (*SocioeconomicWorld, error)
```

Layer 2 v1 has one prescribed socioeconomic model. Output-affecting D2 versions, Company universe parameters, model versions, and other Layer 2 model choices are frozen contract/data inputs, not caller-configurable runtime constructor arguments.

Therefore v1 has:

- No exported socioeconomic config struct.
- No runtime CompanyCount constructor argument.
- No D2-selection constructor argument.
- No duplicated namespace, seed, ReferenceDate, or Layer 1 WorldID arguments.
- No separately named configurable socioeconomic constructor.

Future contracts shall complete the fixed v1 model behind this constructor. A future version of Synthetic World may introduce another API if needed; Contract 1 neither reserves nor promises one.

Construction shall:

1. Validate the supplied base under the provenance and initialization rules below.
2. Establish the prescribed D2 and Company runtime state.
3. Validate that state under the later owning contracts.
4. Establish the stable socioeconomic identity.
5. Return a usable immutable wrapper only after all steps succeed.

Failure returns `nil, err`. A partially initialized wrapper shall not escape.

The base-validation contract is:

- Nil base is invalid.
- Supported input provenance is a World successfully constructed by `NewWorld`, including an equivalent value copy of that valid World.
- Implementation must reject a zero-value or uninitialized World where this can be determined from existing frozen state.
- Layer 1 shall not be modified merely to add an initialization marker.
- Exact internal validation checks belong to implementation review unless a stronger source-proven rule is established.

Validation shall not generate a Person, select a photo, or rebuild the base from separately supplied configuration. The contract does not claim to validate arbitrary unsafe memory corruption.

**RATIONALE:** the constructor accepts the sole caller-supplied dependency. The remaining v1 model is prescribed, so runtime configuration placeholders are unnecessary.

The no-context signature commits this constructor to bounded local work over the prescribed model and bounded data. Generation has the same bounded-local-work requirement. Later D2 and Company contracts must supply enforceable resource bounds before implementation; this contract does not assert that those bounds have already been designed.

Network access, external acquisition, and operations whose duration cannot be bounded do not belong in these signatures. Any necessary loading boundary must respect `AGENTS.md:70` without introducing caller-selectable v1 model choices.

**DEFERRED:** the fixed D2 versions, internal loading/validation, Company universe parameters, and resource bounds. Their values and implementation complete this constructor; they do not expand its public arguments.

### 8. Composite Person API

**NORMATIVE:**

```go
func (w *SocioeconomicWorld) Person(index PersonIndex) (SocioeconomicPerson, error)
```

Success contains the same frozen Layer 1 Person returned by the base for the index and the same socioeconomic facts returned by Facts for that index.

Containment is named: `result.Person` and `result.Facts`. Neither field is anonymous embedding.

On a valid wrapper, zero returns ErrInvalidIndex; an index above Layer1MaxPersonIndex returns ErrLayer1IndexOutOfRange. Frozen generation errors retain their errors.Is behavior. Failure returns the zero composite value and a non-nil error.

The public method delegates to an unexported Layer 2 implementation. That implementation may reuse frozen `base.person` and the shared socioeconomic-facts implementation. The public method shall not directly call another exported same-package method.

### 9. Facts-only API

**NORMATIVE:** expose both retrieval paths:

```go
func (w *SocioeconomicWorld) Facts(index PersonIndex) (SocioeconomicFacts, error)
```

Facts enforces the same operational PersonIndex bounds and error identities as Person. Validation must occur before using lower-level helpers: frozen `birth` does not implement the complete Person operational-boundary check. SOURCE FACT: `tools/fakedata/layer1.go:355`, `tools/fakedata/layer1.go:498`.

Facts shall not invoke full Layer 1 Person generation, select a PhotoID, or require photo selection as a generation dependency. AnchorAge comes from frozen birth logic.

Both public retrieval paths shall use the same unexported socioeconomic generation logic, so semantic agreement is structural rather than maintained through two independent implementations.

**RATIONALE:** full Person generation performs name, contact, address, and photo work; photo selection scans the catalog. Facts-only access avoids unrelated work and makes the dependency boundary enforceable. SOURCE FACT: `tools/fakedata/layer1.go:505`, `tools/fakedata/layer1_photo.go:276`.

The caller still supplies a valid base World. Facts-only access does not introduce a catalog-free replacement for NewWorld or change frozen construction requirements.

### 10. Composite Person semantics

**NORMATIVE:** socioeconomic facts describe state at:

```text
PopulationAnchorDate = 2026-01-01
```

SOURCE FACT: this is the frozen constant at `tools/fakedata/layer1.go:19`.

The contained Layer 1 Person describes the base snapshot. Its CurrentAge remains calculated using that World’s ReferenceDate. SOURCE FACT: `tools/fakedata/layer1.go:543`.

A composite therefore contains two explicit temporal perspectives: socioeconomic facts at the population anchor and frozen Layer 1 snapshot-derived values at ReferenceDate.

Changing ReferenceDate alone shall leave socioeconomic facts unchanged. It may change the contained CurrentAge. No employment, education, income, or Company timeline exists in v1, and composite retrieval shall not age socioeconomic facts forward or backward.

Equality of facts means equality of contents, including pointed-to values. Pointer addresses are not part of observation identity, and Go `==` on these structs is not the prescribed content-equality operation.

### 11. CompanyIndex

**NORMATIVE:**

```go
type CompanyIndex uint64
```

Its protocol coordinate domain is `1..math.MaxUint64`. Zero is invalid.

CompanyIndex is a distinct defined type. Company identity, lookup, and streams convert directly to or from generic uint64 coordinates where necessary. They shall not convert through PersonIndex or apply Layer 1 Person bounds.

The operational Company universe is finite:

```text
1..CompanyCount
```

This contract defines neither CompanyCount nor its safety maximum. CompanyCount is a prescribed v1 contract/data input, not a runtime constructor argument.

**RATIONALE:** protocol coordinates and operational resource limits are separate concepts.

### 12. Company ID encode/decode

**NORMATIVE:**

```go
func CompanyIDForIndex(namespace IdentityNamespace, index CompanyIndex) (string, error)
func CompanyIndexFromID(namespace IdentityNamespace, id string) (CompanyIndex, error)
```

Both functions reuse the frozen generic entity identity helpers with entity kind exactly `"company"`.

Encoding accepts every nonzero CompanyIndex in the protocol domain, without consulting CompanyCount. Decoding establishes valid frozen syntax/version, the expected entity kind, the expected namespace name and generation, and an assigned nonzero coordinate. Decoding does not establish operational membership.

The frozen helper checks kind and namespace separately and rejects the unassigned token instead of returning coordinate zero. SOURCE FACT: `tools/fakedata/identity.go:135`.

The frozen mapping shall remain:

```text
Namespace:    {Name: "krenalis-demo", Generation: 1}
CompanyIndex: 1
Company ID:   company_v1_krenalis-demo_00000001_ef434ynxgmwyh
```

The inverse returns CompanyIndex(1). SOURCE FACT: `tools/fakedata/identity_test.go:314`.

Company identity depends on namespace and coordinate under the frozen entity-kind protocol. WorldSeed does not affect it: the identity key includes namespace and kind, not seed. SOURCE FACT: `tools/fakedata/identity.go:169`.

No second Company namespace, new codec, or new CompanyID Go type is introduced. A Company ID is returned as a string, following the frozen Person ID API at `tools/fakedata/identity.go:274`.

### 13. Operational Company membership

**NORMATIVE:** a coordinate belongs to a socioeconomic World exactly when:

```text
1 <= CompanyIndex <= CompanyCount
```

Successful protocol decoding is insufficient evidence of membership. The protocol ID of a coordinate is independent of CompanyCount.

A coordinate’s membership depends on the operational universe, even though its protocol identity does not. This separation does not introduce caller-configurable universes in v1.

Every generated employee employer reference belongs to the same socioeconomic World that generated the facts. Later fixed-model validation must reject a model that can generate an employee while providing no eligible employer; the count and eligibility policy remain deferred.

**DEFERRED:** the prescribed CompanyCount, resource bounds, and index construction belong to Company Universe Protocol.

### 14. Company retrieval

**NORMATIVE:**

```go
func (w *SocioeconomicWorld) Company(index CompanyIndex) (Company, error)
```

The method validates membership before generating or accessing Company data.

- Zero returns ErrInvalidIndex.
- A nonzero coordinate above CompanyCount returns ErrCompanyIndexOutOfRange.
- Success returns the deterministic Company for the coordinate in this World.
- Company.ID uses the base World’s namespace and frozen Company identity protocol.
- Failure returns Company{} and a non-nil error.

Repeated and concurrent lookups agree by value. Company facts are anchor-stable and obey the socioeconomic dependency exclusions.

**Decision:** no CompanyByID API in v0. Decode with CompanyIndexFromID, then call Company.

**RATIONALE:** those two operations provide the complete use case and keep coordinate decoding visibly separate from membership. A forwarding method adds no required behavior.

### 15. Stable socioeconomic world identity

**NORMATIVE:**

```go
type SocioeconomicWorldID string

func (w *SocioeconomicWorld) ID() SocioeconomicWorldID
```

The value identifies the stable socioeconomic Layer 2 model/data/configuration associated with one underlying Layer 1 stable World.

It is a distinct defined type, not an alias for WorldID or SnapshotID. String-backed identity types follow frozen package practice at `tools/fakedata/composed_id.go:19`.

Its eventual computation shall bind:

- The actual underlying World.WorldID().
- Layer 2 model/spec identity.
- PopulationAnchorDate.
- Every output-affecting D2 identity.
- Company-universe and Company-model semantics.
- Employer-selection semantics.
- Income-model semantics.
- Every other output-affecting Layer 2 configuration.

“Output-affecting” here concerns Layer 2 facts and Company outputs; it does not pull snapshot-only Layer 1 state into this identity. In v1, Layer 2 model/configuration inputs are prescribed by the frozen contracts and data, not selected through constructor arguments.

The identity is independent of ReferenceDate, Layer 1 SnapshotID, FaceCatalog, and PhotoID. It is fixed at successful construction. The empty value is not a valid constructed-world identity.

It shall not serve as an RNG mega-seed. Identity coverage and random dependency selection are separate responsibilities.

**RATIONALE:** a distinct type and wrapper accessor separate socioeconomic model identity from existing frozen identities without duplicating their names.

**DEFERRED:** exact KFD1 encoding, textual format, domain, and component names belong to Deterministic Protocol Contract. No public identity composer/parser is introduced here.

### 16. Relationship to Layer 1 WorldID/SnapshotID

**NORMATIVE:** frozen World.WorldID() and World.SnapshotID() retain their meanings and implementations.

The wrapper exposes neither duplicate Layer 1 identity accessors nor an underlying-World accessor. Callers needing those identities retain the base supplied to construction.

**RATIONALE:** callers already possess the authoritative base, so duplication is unnecessary.

Layer 2 binds the actual stored Layer 1 WorldID rather than reconstructing an equivalent-looking value from independent inputs. Same-package implementation can read the authoritative field returned by the frozen accessor at `tools/fakedata/layer1.go:260`.

For equal Layer 2 configuration and equal base WorldID, different base ReferenceDates shall yield equal socioeconomic identities and facts. Frozen tests establish equal WorldID, different SnapshotID, and equal stream roots under ReferenceDate variation: `tools/fakedata/layer1_test.go:487`.

No socioeconomic SnapshotID is introduced.

### 17. Identity concept separation

| Concept | Meaning | Does not establish |
| --- | --- | --- |
| Person coordinate identity | PersonIndex / Person ID under the frozen namespace protocol | Socioeconomic configuration or snapshot |
| Company coordinate identity | CompanyIndex / Company ID in the namespace’s company domain | Operational Company membership |
| Operational Company membership | Coordinate is within this World’s finite Company universe | A new identity codec or namespace |
| SocioeconomicWorldID | Stable Layer 2 configuration associated with the actual stable Layer 1 World | ReferenceDate/photo snapshot state |
| Layer 1 SnapshotID | Frozen World’s ReferenceDate and photo-catalog snapshot configuration | A socioeconomic timeline or Layer 2 model identity |
| Composite observation key | SocioeconomicWorldID + Layer 1 SnapshotID + PersonIndex | A new combined hash |

**NORMATIVE:** the complete composite observation is identified by the semantic tuple:

```text
(SocioeconomicWorldID, Layer1 SnapshotID, PersonIndex)
```

SnapshotID must come from the base associated with the socioeconomic World. Mixing identities from unrelated Worlds does not name a valid generated observation.

No concatenation format, combined hash, or new key type is specified. Entity IDs alone do not identify the model-dependent facts attached to those entities.

### 18. SocioeconomicFacts

**NORMATIVE:** SocioeconomicFacts contains common socioeconomic facts and exactly one semantic PrimaryActivity branch.

Education is the structural location for education state. Its representation and values belong to Taxonomy & Applicability.

Activity-specific data resides in branch payloads. Empty strings shall not encode whether the employee or self-employed branch applies.

The facts contain no duplicate Person ID, PersonIndex, world identity, or snapshot identity. Their generation context and composite observation key provide those associations.

**RATIONALE:** branch grouping makes applicability explicit while retaining ordinary Go value structs.

**DEFERRED:** education taxonomy, additional common facts, income, and completion of the occupational payloads. Taxonomy & Applicability shall define occupational fields and applicability before implementation.

### 19. PrimaryActivity

**NORMATIVE:** freeze the six structural Go/string constants now:

```go
// Supported primary activities at PopulationAnchorDate.
const (
    PrimaryActivityEmployee     PrimaryActivity = "employee"
    PrimaryActivitySelfEmployed PrimaryActivity = "self-employed"
    PrimaryActivityUnemployed   PrimaryActivity = "unemployed"
    PrimaryActivityStudent      PrimaryActivity = "student"
    PrimaryActivityRetired      PrimaryActivity = "retired"
    PrimaryActivityInactive     PrimaryActivity = "inactive"
)
```

The zero value and every unrecognized value are invalid generated states. Exactly one branch applies.

These are primary-activity categories. They do not assert absence of secondary activities, which are not modeled by this contract.

The literal strings are the canonical machine IDs when the discriminator is serialized. No overall JSON schema, field tags, or wire format is defined here.

**RATIONALE:** the six branches are already architectural commitments. Their machine IDs require no empirical crosswalk.

**DEFERRED:** source crosswalks, probabilities, assignment rules, and detailed applicability. The six structural machine IDs are not deferred.

### 20. Employee payload

**NORMATIVE:** for PrimaryActivityEmployee:

- Employee is non-nil.
- SelfEmployed is nil.
- Employee.Employer is mandatory, nonzero, and belongs to the generating World.

The employer reference contains only CompanyIndex. It contains no independently stored Company ID, namespace, copied Company, or dedicated reference wrapper. Resolve it with the generating World’s Company method; that result supplies its ID.

**RATIONALE:** CompanyIndex is sufficient in this context and avoids inconsistent index/ID pairs. The reference is meaningful together with the generating World; facts detached from that context are not a self-contained cross-world reference format.

Contract 1 does not define an always-present employee JobFamily field or a final JobLevel optionality representation.

**DEFERRED:** Taxonomy & Applicability shall extend and complete EmployeeFacts before implementation, including JobFamily/JobLevel fields, representation, and applicability. This contract freezes neither universal JobLevel applicability nor the meaning or representation of occupational absence.

### 21. Self-employed payload

**NORMATIVE:** for PrimaryActivitySelfEmployed:

- SelfEmployed is non-nil.
- Employee is nil.
- There is no employee-employer Company reference.
- There is no owned-business entity in v1.

The distinct payload provides the branch envelope in which future occupational facts will be represented. Contract 1 does not define an always-present self-employed JobFamily field.

No Company shall be manufactured merely to give self-employment the same shape as employment.

**RATIONALE:** self-employment has different entity relationships and requires its own payload.

**DEFERRED:** Taxonomy & Applicability shall extend and complete the occupational payload before implementation, including fields, representations, and applicability. Self-employment income semantics remain owned by Income Contract.

### 22. Non-working states

**NORMATIVE:** for unemployed, student, retired, and inactive, both Employee and SelfEmployed are nil.

These branches have no current employer reference and no branch-specific payload in Contract 1.

Absence of current employment payload establishes no employment history. Retired status shall not cause a former employer to be invented; unemployed status shall not imply a previous job.

**RATIONALE:** this contract specifies no additional current facts requiring a separate non-working payload.

### 23. Impossible-state invariants

Every successful generated facts value shall satisfy:

| PrimaryActivity | Employee payload | Self-employed payload | Employee employer |
| --- | --- | --- | --- |
| employee | Present | Absent | Valid member coordinate |
| self-employed | Absent | Present | Absent |
| unemployed | Absent | Absent | Absent |
| student | Absent | Absent | Absent |
| retired | Absent | Absent | Absent |
| inactive | Absent | Absent | Absent |

Zero and unknown discriminators shall never be generated. These guarantees apply to facts-only output and facts contained in composite output.

Public structs allow callers to construct or mutate inconsistent values manually. The package guarantees validity of its successful outputs at return, not validity of arbitrary caller-created values.

No public API here accepts SocioeconomicFacts as input. No exported facts validator or activity-validation sentinel is added. Generation must establish the invariants and return a non-nil error if it cannot do so.

**RATIONALE:** a discriminator with branch pointers is sufficient for generated output. A sealed-interface hierarchy would not remove the need to establish employer membership and other relational invariants.

### 24. Error contract

**NORMATIVE:** reuse frozen categories where their meaning fits. Introduce only these two public error categories:

```text
ErrCompanyIndexOutOfRange
ErrInvalidSocioeconomicWorld
```

Follow existing exported sentinel constants backed by an unexported string error type. SOURCE FACT: `tools/fakedata/errors.go:7`, `tools/fakedata/layer1_errors.go:7`.

| Condition | Required identifiable error |
| --- | --- |
| Zero CompanyIndex with otherwise valid inputs | ErrInvalidIndex |
| Nonzero CompanyIndex above CompanyCount | ErrCompanyIndexOutOfRange |
| Nil base, detected uninitialized base, or invalid socioeconomic construction configuration | ErrInvalidSocioeconomicWorld |
| Zero PersonIndex on a valid wrapper | ErrInvalidIndex |
| PersonIndex above the frozen maximum | ErrLayer1IndexOutOfRange |
| A frozen generation operation fails | Preserve the returned error’s identity |
| Invalid namespace supplied to a Company identity function | ErrInvalidIdentityNamespace |

Company decoding preserves all applicable frozen decoder categories:

```text
ErrInvalidIdentityNamespace
ErrMalformedSyntheticID
ErrUnsupportedSyntheticIDVersion
ErrWrongEntityKind
ErrWrongIdentityNamespace
ErrUnassignedSyntheticID
```

Protocol validation precedence follows the frozen helpers, including namespace validation before coordinate encoding checks. SOURCE FACT: `tools/fakedata/identity.go:111`, `tools/fakedata/identity.go:135`.

errors.Is shall recognize specified categories through contextual wrapping. A positive out-of-universe coordinate is not classified as ErrInvalidIndex.

ErrInvalidSocioeconomicWorld identifies unusable base/configuration, including invalid prescribed D2 or Company model inputs. These are not caller-selectable v1 constructor arguments. An unexpected internal construction failure must still return a non-nil error, but shall not be misclassified as invalid caller configuration merely to fit this sentinel. Later model contracts may define additional identifiable failures only where callers have a concrete handling need.

Company, Facts, and Person on a nil or uninitialized socioeconomic wrapper return ErrInvalidSocioeconomicWorld before validating their coordinates. ID returns the invalid empty identity for such a wrapper. Successful construction always establishes a nonempty ID.

Every error-returning API returns its zero result on failure. No partial result is reported as success. Unexpected internal generation failures require an error but no new exported category here. There is no error per activity branch.

Future declarations shall document all reachable identifiable errors; error messages are not an API for classification. No HTTP-response error types are introduced.

### 25. Immutability/concurrency/random-access

**NORMATIVE:** a successfully constructed socioeconomic World is immutable for generation and safe for concurrent calls.

There shall be no shared advancing RNG, generated-Person registry, occupancy state, employer-assignment history, or behavior dependent on prior requests.

Repeated calls, reversed traversal, and concurrent retrieval agree by value. Company lookup is unaffected by which people have been generated. The immutable Company index is model state, not a record of generated assignments.

Fresh deterministic streams reuse frozen generic stream machinery. It returns a fresh RNG value for each derivation. SOURCE FACT: `tools/fakedata/stream.go:36`.

Returned payload pointers shall not expose mutable internal state. Mutating a returned employee or self-employed payload shall not affect the World, another independently returned result, or a later call. Future occupational and other reference-bearing taxonomy fields must preserve this ownership guarantee.

The concurrency guarantee does not make concurrent caller mutation of the same returned object safe. Ordinary caller copies of a returned struct may share its pointers; the guarantee concerns independent API results and internal state.

**DEFERRED:** stream paths, dependency components, and the Company selection-index algorithm.

### 26. Dependency exclusions

**NORMATIVE:** socioeconomic generation shall not use these as direct or transitive inputs:

```text
Gender
CurrentAge
ReferenceDate
Layer1 SnapshotID
FaceCatalog
PhotoID
```

This applies to socioeconomic Person facts, Company facts, employer selection, and stable socioeconomic identity. It also prohibits indirect use through derived values, such as gender-dependent names. Statistical correlation through an otherwise permitted shared cause is not itself a dependency on an excluded field.

Age logic uses frozen AnchorAge from base birth generation. It shall not reconstruct AnchorAge by adjusting CurrentAge.

The composite may contain frozen Layer 1 values derived from excluded inputs. Producing that Person is permitted; feeding excluded information from it into socioeconomic generation is not.

Validation that the supplied base is usable does not make its catalog or ReferenceDate socioeconomic generation inputs.

**DEFERRED:** permitted causal relationships among remaining inputs and outputs belong to Socioeconomic DAG.

### 27. Deferred-design ledger

| Deferred subject | Owning future contract |
| --- | --- |
| Prescribed CompanyCount and Company resource bounds | Company Universe Protocol |
| EmployeeCount type/range | Company Universe Protocol |
| Employer-selection weight | Company Universe Protocol / D2 model |
| Selection-index algorithm | Company Universe Protocol |
| Fixed Company model parameters and index initialization | Company Universe Protocol |
| Company fields beyond ID and Company name model | Company contract |
| EducationLevel representation and values | Taxonomy & Applicability |
| PrimaryActivity empirical crosswalks and applicability | Taxonomy & Applicability |
| Industry | Taxonomy & Applicability |
| JobFamily fields, representation, values, and applicability | Taxonomy & Applicability |
| JobLevel fields, representation, semantics, and applicability | Taxonomy & Applicability |
| Completion of employee and self-employed occupational payloads before implementation | Taxonomy & Applicability |
| SizeBand thresholds | Taxonomy & Applicability / Company contract |
| Exact causal dependencies | Socioeconomic DAG |
| Geography conditioning | Socioeconomic DAG / D2-A |
| Income semantics, type, and bands | Income Contract |
| D2-A data | D2-A Contract |
| D2-B data | D2-B Contract |
| Prescribed D2 versions, internal loading/validation, and resource bounds | D2-A / D2-B contracts, with Company Universe Protocol |
| Stream paths and dependency components | Deterministic Protocol Contract |
| Exact SocioeconomicWorldID encoding, domain, and component names | Deterministic Protocol Contract |
| Overall serialization schema, if needed | A future serialization/API contract |

PrimaryActivity’s six serialized machine IDs are frozen by section 19; no deferred machine-ID decision remains.

This ledger does not authorize changes to frozen Layer 0/1. Later contracts must complete the fixed v1 model and occupational payload contract without treating omitted values or models as implicit defaults. They shall not introduce caller-configurable v1 model arguments.

### 28. API usage examples

These are caller-side fragments, not package implementations. Error handling surrounds each operation. `base` is the retained valid World, `socioeconomic` its successfully constructed wrapper, and `namespace` the same namespace originally used to construct base.

**A — Composite Person**

```go
result, err := socioeconomic.Person(personIndex)
if err != nil {
    return err
}
```

Read result.Person and result.Facts. No second generation call is required.

**B — Facts only**

```go
facts, err := socioeconomic.Facts(personIndex)
if err != nil {
    return err
}
```

This path generates no PhotoID and performs no photo selection.

**C — Company coordinate**

```go
company, err := socioeconomic.Company(companyIndex)
if err != nil {
    return err
}
```

Success establishes membership and returns the deterministic Company.

**D — Protocol Company ID decode**

```go
companyIndex, err := CompanyIndexFromID(namespace, companyID)
if err != nil {
    return err
}
```

Success establishes a coordinate only. No operational World is needed for protocol decoding.

**E — World membership after decoding**

```go
company, err := socioeconomic.Company(companyIndex)
if err != nil {
    return err
}
```

The successfully decoded coordinate may return ErrCompanyIndexOutOfRange. For generated employee facts, lookup instead uses facts.Employee.Employer in the World that produced the facts, after selecting the employee branch.

**F — ReferenceDate variation**

Construct two valid base Worlds differing only in ReferenceDate and wrap each with `NewSocioeconomicWorld(base)`. Both use the prescribed v1 socioeconomic model. For the same PersonIndex:

```text
socioeconomicA.ID() == socioeconomicB.ID()
factsA and factsB have equal contents
baseA.WorldID() == baseB.WorldID()
baseA.SnapshotID() != baseB.SnapshotID()
```

Contained Layer 1 CurrentAge may differ. No socioeconomic aging or derivation from CurrentAge occurs.

**G — Wrong namespace/kind**

A valid Person ID supplied to Company decoding returns ErrWrongEntityKind. A syntactically valid Company ID from another namespace name or generation returns ErrWrongIdentityNamespace, according to frozen decoder precedence.

All seven scenarios are implementable through new same-package files. None requires changing World, WorldConfig, or Person.

### 29. Required future tests

This specification does not implement tests. The future implementation shall provide:

| Area | Required coverage |
| --- | --- |
| Company identity | Zero rejected; frozen forward/inverse golden; wrong kind; wrong namespace name/generation; protocol boundary coordinates; seed independence |
| Decode versus membership | A coordinate outside a smaller operational universe decodes successfully, then Company lookup rejects it |
| Constructor | Nil rejected; zero-value/uninitialized base rejected using existing detectable state; valid NewWorld-derived base accepted; invalid prescribed D2/Company initialization returns no wrapper; validation performs no Person/photo generation |
| Person bounds | Both paths reject zero and above-maximum values with frozen sentinel identities |
| Composite consistency | Contained Person equals base Person; contained facts equal facts-only output by contents |
| Facts isolation | Facts-only does not call full Person generation or photo selection |
| Determinism | Repeat-order, forward-order, reverse-order, and concurrent equivalence |
| ReferenceDate | Equal socioeconomic identity and facts across date variation; different SnapshotIDs; a suitable fixture demonstrates changed CurrentAge |
| Catalog independence | Valid alternative catalogs preserve socioeconomic identity and facts |
| Activity state | No zero/unknown discriminator; exact payload consistency; employees have valid member employer references |
| Self-employment | Required payload present; employee payload and employee Company reference absent |
| Non-working states | Both working payloads absent; no historical employer inferred |
| Output ownership | Mutating one result does not affect independently returned results or later calls |
| Errors | errors.Is categories and wrapping; out-of-universe versus zero; invalid-wrapper behavior; zero result on error |
| Identity ownership | Actual base WorldID and all later-required output-affecting model/data inputs are bound |
| Dependency audit | No direct or derived Gender, CurrentAge, ReferenceDate, SnapshotID, catalog, or photo inputs to socioeconomic generation |
| Frozen compatibility | Existing Layer 0/1 tests and goldens remain unchanged and pass |

An internal smaller-universe test fixture does not create a public runtime CompanyCount option.

Facts-only isolation needs evidence beyond equal output: inspect the new call graph and use a same-package test fixture that makes accidental photo selection observably fail. This must not require frozen-source changes, a public test hook, or a helper extracted solely for testing. A deliberately unusable photo dependency in a test-local copy is a dependency probe, not an additional supported public base configuration.

Activity coverage must intentionally exercise all branches. A random sample that happens to omit a branch is insufficient.

Occupational field and applicability tests belong to the completed Taxonomy & Applicability contract. Contract 1 shall not assert an always-present JobFamily value or a particular JobLevel optionality representation.

Encoding-dependent identity tests and operational-universe fixtures become concrete under their owning contracts. No distribution, empirical-fit, or weighting-policy tests belong to Contract 1.

### 30. Independent-review questions

1. Are the public names coherent and free of implementation-layer terminology?
2. Is the single constructor definitive for the prescribed v1 model, without runtime D2 or Company configuration, and can later bounds satisfy its no-context contract?
3. Is facts-only access justified and demonstrably isolated from full Person/photo generation?
4. Is protocol Company decoding fully separated from operational-universe membership?
5. Does the CompanyIndex-only employer reference avoid redundant, inconsistent state while retaining sufficient World context?
6. Is socioeconomic identity distinct from Layer 1 WorldID and SnapshotID?
7. Do discriminator/payload guarantees prevent invalid generated states without an elaborate hierarchy?
8. Are occupational payload fields and applicability fully left to Taxonomy & Applicability, without accidental JobFamily, JobLevel, income, or Company-weighting commitments?
9. Can every API and validation rule be implemented through new same-package files, without adding a Layer 1 initialization marker?
10. Does any direct or indirect socioeconomic dependency on Gender or ReferenceDate remain?

### 31. Contract v0 verdict

This contract selects one coherent API and activity-state model.

Frozen source provides the necessary extension points: private base state, independent birth generation, generic Company identity, and generic entity streams. No contradiction requiring frozen Layer 0/1 to be reopened was identified.

Layer 2 v1 has one prescribed socioeconomic model behind `NewSocioeconomicWorld(base *World)`. Future contracts complete that fixed model rather than introducing runtime constructor choices.

Taxonomy & Applicability must complete the occupational payload fields and applicability before implementation. Contract 1 freezes the working-branch envelopes and mandatory employee employer reference only.

The remaining deferred models, data, resource bounds, taxonomies, and deterministic protocol must be completed before implementation can be complete. Those dependencies do not block independent review of Contract 1.

No Layer 2 code, fixtures, or tests are implemented by this specification.

## Source-backed risks

- Frozen birth does not enforce the complete Person operational bound; Facts must validate first. Sources: `tools/fakedata/layer1.go:355`, `tools/fakedata/layer1.go:498`.
- Reusing full person generation for Facts would introduce Gender and photo work. Sources: `tools/fakedata/layer1.go:509`, `tools/fakedata/layer1.go:537`.
- Protocol decoding has no Company-universe knowledge; membership needs a World-level check. Source: `tools/fakedata/identity.go:135`.
- Base validation must use existing frozen constructor-established state. Exact checks require implementation review; no new initialization marker may be added. Sources: `tools/fakedata/layer1.go:127`, `tools/fakedata/layer1.go:224`.
- The no-context constructor requires later contracts to bound its local work. Source rule: `AGENTS.md:70`.

## Deferred decisions

Section 27 assigns future owners. The prescribed Company universe and bounds, occupational payload completion, taxonomy applicability, socioeconomic dependencies, income, D2 assets, stream definitions, and socioeconomic identity encoding remain deliberately unresolved.

These are fixed v1 contract/data design decisions, not deferred caller-configurable constructor arguments. No external datasets were researched and no distributions were designed.

## Repository-integrity proof

Only the session-created file was removed:

```text
SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v0.md
```

Its previous text remains available in the conversation. No pre-existing repository artifact was removed or modified.

Fresh before/after checks were run for this rectification:

- Captured full `git status --porcelain=v1 -uall` before removal and after validation.
- The raw status difference contains exactly one removed entry: the session-created specification file.
- Constructed the expected restored status by removing only that exact entry from the before snapshot. `cmp` against the final full status passed.
- All pre-existing untracked status entries remain unchanged.
- Before/after SHA-256 comparison of `tools/types/semantic.go` passed.
- Before/after comparison of its working-tree diff passed.
- `git diff --check` passed.
- `git diff --cached --quiet` passed.
- Confirmed that the removed specification path no longer exists.
- Re-ran `go test ./tools/fakedata` with a newly allocated Go cache under `/tmp`; it passed.
- Removed the temporary cache and temporary comparison files.

The original drafting-pass status snapshots had already been deleted during that pass’s cleanup. Consequently, the fresh comparison proves the exact single-file rollback against this rectification’s before snapshot; it is not represented as a new byte-for-byte comparison against an original snapshot that no longer exists.

Final branch and HEAD remain:

```text
property-semantics
76fa0fff109ca6abad62f55c8761275455842dc5
```

Final tracked working-tree status remains:

```text
 M tools/types/semantic.go
```

The pre-existing untracked entries remain present. Nothing was staged or committed. The revised specification is contained only in this response.

L2 DETAILED CONTRACT 1 v1 — FROZEN
