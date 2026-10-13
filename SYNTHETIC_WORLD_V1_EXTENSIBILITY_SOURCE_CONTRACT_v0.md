# Synthetic World v1 — Extensibility and Source Simulation

**Status: DRAFT · NOT FROZEN · ARCHITECTURE CONTRACT · NO IMPLEMENTATION**

This draft selects one architecture for a canonical population, imperfect
source observations, and hidden evaluation truth. It does not alter Layer 0–2
semantics. Here, **World** means the proposed country-aware generation owner,
which need not be the Italy-only Go `fakedata.World` type. No Go API is specified.

## Verified current state

The following classifications distinguish executable code from specifications
and files available in the working tree.

| Classification | Verified finding |
| --- | --- |
| **IMPLEMENTED** | `tools/fakedata/layer1.go:104–139, 145–257, 496–547` defines the typed Layer 1 `Person`, `WorldConfig`, Italy-only `World`, and random-access `World.Person(index)`. `Person` has ID, gender, birth date/current age, names, email, phone, site URL, address, and photo ID. `Address.Country` is set to `IT` by the Italian generator (`layer1.go:349–352`); it is an address value, not a general nationality declaration. Construction requires a valid face catalog. |
| **IMPLEMENTED** | `tools/fakedata/identity.go:25–35, 274–295` supplies namespace-scoped, index-based Person IDs; IDs do not incorporate the World seed. `tools/fakedata/stream.go:22–65` supplies fresh world/entity-scoped deterministic streams; Layer 1 addresses person draws by index and path (`layer1.go:549–551`). `tools/fakedata/range.go:9–57` supplies bounded index ranges. These are usable primitives, not a source simulator. |
| **AVAILABLE AS DATA/ASSET** | Layer 1 embeds the Italian CSV inputs through `tools/fakedata/layer1_data.go:39–42`, and `NewWorld` loads them (`layer1.go:154`). `D2B2-logical-assets-v0.json` and `D2B2-publication-v3.json` are present as canonical Italian Layer 2 model/publication files in the working tree. Presence of these JSON files does not imply a Go runtime. |
| **DEFINED / FROZEN BUT NOT IMPLEMENTED** | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:90–165` defines a named `SocioeconomicPerson` containing the frozen Layer 1 `Person` plus typed facts, and a `SocioeconomicWorld` wrapping the base World. Its own text calls the skeletons noncompilable (`:50`). Other frozen Layer 2 contracts define Italian semantics and numerical/runtime obligations; the D2-B2 publication contract leaves deterministic protocol and Go runtime work to later implementation. |
| **TO IMPLEMENT** | There is no Go `SocioeconomicWorld`, Layer 2 asset consumer, country selection boundary, `SourceInstance`, `SourceRecord`, Oracle, or source transport in the inspected runtime (`tools/fakedata` and repository Go search). Italian Layer 2 runtime integration and its prescribed deterministic protocol are prerequisites; they are not changes to the frozen model. |

## Scope, terms, and country boundary

A **canonical Person** is one generated individual's ground truth, identified by
PersonID. For the Italian path, that means the existing typed Layer 1 Person and,
once implemented, the separately defined typed socioeconomic facts. A
**SourceInstance** is one logical simulated source. A **SourceRecord** is one
local observation emitted by that source. The **Oracle** is hidden evaluation
truth relating a record to its canonical Person.

The country-aware World owns the population coordinate/PersonID, output-affecting
configuration, and choice of country implementation. A single simple boundary
passes the chosen country and stable Person coordinate to country-dependent
generation, which returns canonical Person data plus an unambiguous associated
country context. The context means the geography/model that generated the Person,
not citizenship or nationality. It may live beside the Person; this draft does
not require a new field on the frozen Layer 1 `Person`. The generic source
membership, record-count, identity, and Oracle machinery receives canonical
values and that context, never private Italian model tables or classifications.
Country-specific generation and any needed country-specific observation logic
remain local to the country implementation or local record mapping.

An Italy-only selection delegates to the existing Italian Layer 1 runtime and,
after its implementation, the frozen Italian Layer 2 runtime. This preserves
their semantics and does not require changing frozen `Person`, `World`, or
`WorldID`. A mixed-country World can be an enclosing runtime whose identity
reflects its output-affecting country composition. Its shape is deferred.

World configuration selects a nonempty set of requested countries and relative
composition, initially `IT` alone and later such sets as `IT+FR` or `IT+DE`.
Each PersonID is assigned exactly one country by a deterministic choice from
that configuration and World seed/version. With the same configuration, seed,
version, and PersonID, the choice is unaffected by request order, concurrency,
other generated Persons, map order, or the materialized population subset.
Population size does not enter the assignment, preserving the existing
index/identity model's population-size independence. Relative composition is a
selection policy, not an exact finite-sample quota. A configured country without
an implementation causes an explicit error before generation; no fallback
country is permitted. The choice algorithm remains open.

## Canonical fields and source observations

Canonical Person data remains typed. Adding a field may require a local type,
country generator, or record/transport mapping change. It must not change how
membership, multiplicity, SourceRecord identity, or Oracle association works:
those operations address Persons by PersonID and source identity, not by a fixed
list of Person fields. The existing Italian address and frozen Italian
socioeconomic classifications may remain inside the Italian implementation.
They become architectural coupling only if the shared source engine requires
their private categories to decide membership, count, or truth mapping. No
universal property bag or localization framework is implied.

For each `(Person, SourceInstance)` pair, source-record generation yields **zero
to N SourceRecords**. There is no semantic maximum N. A first implementation
may commonly yield zero or one and occasionally two. Each SourceInstance makes
its own deterministic membership decision for a Person; these decisions need
not be statistically independent. Simple per-source coverage suffices. A
finite run is not promised an exact overlap percentage. Absence from every
source, presence in any subset of sources, and intra-source duplicates all fit
the same rule. Future correlation policy can change membership choices without
changing records or Oracle meaning.

Each emitted SourceRecord has a source-local identity and its own observation
values. An observation may omit a field, represent it differently, or disagree
with canonical truth; two records for one Person can make different choices.
The absence of a record is distinct from an existing record whose field is
absent. The record is never another canonical Person. This draft assigns no
cause or history to duplicates or discrepancies, and no taxonomy of noise.

## Determinism, record identity, and Oracle

Stable semantic inputs determine source membership, record count, each local
record ID, and each record's observations. Those inputs include the World
identity/seed and model version, SourceInstance identity and configuration,
PersonID, a stable discriminator for each record of that Person in the source,
and a field discriminator where independent field observations require one.
Implementation should reuse existing deterministic identity/stream primitives
when suitable. It must not advance shared mutable randomness across Persons,
records, or SourceInstances: generation order, goroutine scheduling, Go map
iteration, a requested subset, or batch boundaries cannot change logical
results. Changed World/source configuration or later model versions need not
preserve outputs. No hash, PRNG, addressing encoding, or probability is frozen.

SourceRecordID is stable within deterministic generation and unique within its
SourceInstance. Its ordinary representation must not visibly encode PersonID
or act as a common hidden join key across sources. Ordinary source output also
excludes canonical PersonID, Oracle mappings, and duplicate-group IDs. Shared
ordinary observations, such as matching contact details, remain possible; the
simulator does not manufacture a universal identity-resolution shortcut.

For v1, every emitted record has exactly one true Person. The hidden Oracle
conceptually maps `(SourceInstanceID, SourceRecordID) -> PersonID`; many keys,
including two in one source, may map to the same PersonID. Evaluation can use
that mapping for both intra-source deduplication and cross-source resolution.
The Oracle is kept outside normal source output and ingestion interfaces. No
orphan/NO_MATCH records or evaluation platform are specified.

## Record generation and transport

The pipeline is `canonical Person + country context → source-record generation
→ logical SourceRecords → transport/serialization`. Source-record generation
owns membership, multiplicity, observations, local IDs, and Oracle truth.
Transport only represents those already chosen logical records; it does not
redraw membership or truth. Thus the same SourceRecords and Oracle association
can be represented through CSV, XLSX, Parquet, or fake SaaS adapters without
redesigning this boundary. Adapter protocols and schemas are deferred. Fake
database servers are outside scope.

Each Person can be generated and projected to a source independently, or
processed in bounded batches. Decisions cannot require a whole-population
roster, global shuffle, or simultaneous materialization of all SourceRecords.
This is a memory/semantic requirement, not a streaming API or distributed
processing design.

## Decision boundary

| Category | Content |
| --- | --- |
| **PROPOSED SEMANTIC DECISION** | Country-dependent Person generation has one boundary; every Person has an unambiguous generation-country context; configured country assignment is deterministic and population-size independent; `(Person, SourceInstance)` yields `0..N` observations; each record is a local observation, not canonical truth; source generation is deterministic and order independent; a hidden per-record Oracle maps to PersonID; transport follows logical record generation; bounded-memory generation is possible. |
| **DEFERRED IMPLEMENTATION DETAIL** | Go types/interfaces and names (including `CountryPack`); where country context is stored; method names; configuration/serialization formats; country-choice, hash, PRNG and record-ID implementations; coverage probabilities, duplicate rates and any v1 count limit; degradation algorithms; adapter schemas/protocols. |

## Six conceptual sufficiency checks

These are architecture checks, not executed tests.

1. **A — Existing Italy.** An IT-only country implementation delegates to
   frozen Layer 1 generation and the future conforming Italian Layer 2 runtime;
   no Italian model semantics change.
2. **B — Second country.** A tiny test-only country implementation can accept
   its assigned Person coordinate and return canonical data with its country
   context. Source and Oracle semantics stay the same; exposed values may need
   local adapter representation changes.
3. **C — New Person field.** A typed field and its generation/observation/adapter
   mappings can be added locally. Membership, record count, identity, and
   Oracle continue to use PersonID and SourceInstance identity.
4. **D — Partial overlap.** Two source instances make separate per-Person
   coverage decisions and can have overlapping, non-identical memberships,
   without an independence assumption.
5. **E — Intra-source duplicate.** One source emits two distinct local IDs and
   independently imperfect observations for one Person; both hidden mappings
   resolve to that PersonID.
6. **F — Order independence.** The same semantic inputs and stable addressing
   yield the same logical records when Persons or sources are visited in a
   different order or in bounded batches.

## Implementation prerequisites and blockers

The Italian Layer 1 runtime and deterministic primitives already exist, but
constructing it still requires valid frozen data and a face catalog. Before the
first source slice can expose full canonical Italian Persons, implementation
must consume and validate the frozen Layer 2 assets, satisfy the already-defined
Layer 2 runtime/deterministic-protocol obligations, and connect the resulting
typed facts to Italian Person generation. Then implementation must add the
minimal country-aware owner, source-record generation, hidden Oracle storage or
emission, and a transport adapter. None of these are claimed to exist today.

**Architectural blockers: 0.** The missing work is implementation, not a
conflict requiring a change to frozen Layer 0–2 semantics.

**NON-NORMATIVE first increment:** conforming Italian Person runtime → minimal
country boundary → two SourceInstances with partial overlap → occasional
imperfect duplicate → hidden Oracle mapping → CSV output. A test-only second
country can exercise the boundary later.
