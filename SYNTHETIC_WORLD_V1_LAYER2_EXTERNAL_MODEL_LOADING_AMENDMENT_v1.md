# Synthetic World v1 — Layer 2 external model loading amendment v1

**STATUS: FROZEN — independently reviewed and manually approved.**

This approved normative amendment resolves only socioeconomic numerical-model
acquisition, ownership, transfer into construction, and transport lifetime.
SHALL/MUST wording is approved normative authority, not runtime implementation
approval.

## 1. Controlling conflict and repository basis

**RESOLVED CONFLICT.** [Contract 1](SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md)
§§5/7 prescribe `NewSocioeconomicWorld(base *World)`; §7 calls the base the
“sole caller-supplied dependency” and says deferred loading decisions “do not
expand its public arguments.” [D2-A](SYNTHETIC_WORLD_V1_LAYER2_D2A_v1_FROZEN.md)
§19 prescribes “embedded numerical inputs” and “No runtime external
files/network.” The fixed external production model therefore has no authorized
explicit transfer boundary into that constructor. Contract 1 §1 prohibits
changing World/WorldConfig; §7 already excludes acquisition from construction.

Source facts at `e3ca1442a49bd15e272472b43041de966d375580`, not authority
overriding those clauses:

- `tools/fakedata/socioeconomic_model.go:18–104` fixes
  `synthetic-world/it-2026a/model.json`, defines private model/source types,
  serializes acquisition, retains only a successful model, and retries failures.
  Its callers are package tests; no SocioeconomicWorld implementation exists.
- `tools/fakedata/socioeconomic_model_json.go:277–395` and
  `socioeconomic_model_compile.go:40–247` decode bounded typed logical-v0
  content, validate complete canonical laws/maps and compile immutable arrays.
  They do not generate Persons, Companies or random state. Raw JSON is not
  retained. Canonical acceptance is not producer fitting or content-hash pinning.
- `cmd/config.go:208–223` owns ExternalAssetsURLs configuration;
  `tools/externalassets/resolver.go:25–146` owns ordered HTTP resolution,
  content decoding, decoded-byte limits and consumer-aware fallback.
- `tools/fakedata/external_photo.go:29–96` provides the concrete source/Load
  precedent. `cmd/server.go:49–83`, `core/core.go:77–78,392–394` and
  `core/simulated_system_generation.go:406–410,498–500` show an application-owned
  reusable source and explicit acquired input to local World construction.
  This is an ownership analogy, not permission to change photo semantics or
  introduce a local socioeconomic-model path.

The protocol audit and independent review are historical evidence only. The
project guide and External Assets specification provide context; neither
silently supersedes frozen Layer 2 authority.

## 2. Exact, limited precedence

> Where this amendment conflicts with the Contract 1 constructor/public-operation
> clauses and D2-A lifecycle clauses identified below, this amendment controls
> only socioeconomic numerical-model acquisition, ownership, explicit transfer
> into SocioeconomicWorld construction, and the associated phase/error/resource
> accounting. Every other frozen requirement remains controlling.

The complete override is:

| Existing clause | Only the requirement superseded |
| --- | --- |
| Contract 1 §5, “complete conceptual public operation set,” constructor entry; §7 constructor signature, “sole caller-supplied dependency” rationale, and final “do not expand its public arguments” sentence | The base-only argument list and operation-set closure against an acquisition API. Add an explicit already-acquired immutable model argument and the acquisition boundary in §3. |
| D2-A §19 “Chosen lifecycle” paragraph | Replace embedded acquisition and the runtime-wide external-file/network prohibition with Phase A external acquisition. Keep bounded local construction and the prohibitions on fitting, mutable reload, lazy generation-time discovery and partial worlds. |
| D2-A §19 placement of steps 2–7 after base validation, and step 11's model-only identity/release work | Split model-only work into Phase A and base/Company/index work into Phase B as specified in §4. Acquisition can precede base validation; no validation requirement is removed. |
| D2-A §19 “Invalid prescribed model/data” paragraph | Acquisition returns no model on failure, rather than a nil world with ErrInvalidSocioeconomicWorld. Construction still returns a nil world and its frozen classification for invalid construction inputs/state. No new error taxonomy is introduced. |
| D2-A §21 “source-format bytes presented to the constructor,” constructor-only temporary lifetimes, and “joint Layer-2 constructor live/reserved peak” wording | Apply the same quantitative limits to the acquisition/construction lifetimes in §6. No numerical ceiling, charge formula or selected reservation increases. |

Contract 1 §7 steps 2–3 remain binding: construction establishes prescribed D2
state by retaining the successfully acquired immutable model, checks its
supported initialization and performs applicable world-specific validation.
It need not repeat acquisition's decoding/compilation. Contract 1 §24 continues
to govern construction and generation; acquisition is a separate operation
with the failure boundary in §7 below. Contract 1 §6 already permits sharing
immutable tables; ownership requires retained access, not a private copy.

Restatements inherit only the exact exceptions in the table. No independent
requirement in Contracts 2–5 or D2-A is superseded merely because it concerns
models, validation, errors or resources, or appears beside an amended clause.

Contracts 1–5 and D2-A otherwise remain the highest authority. D2-B1 retains
its owned-scope precedence v3 > v2 > v1 > v0; the D2-B2 publication contract
retains its publication/evidence scope. This amendment changes no historical
document, publication file set, canonical numerical content or acceptance rule.

## 3. Required public capabilities and recommended Go shape

**REQUIRED NORMATIVE CAPABILITIES:** callers SHALL explicitly acquire a publicly
usable and nameable opaque immutable model of the prescribed it-2026a release, then
supply it with the authoritative base World to the sole SocioeconomicWorld
constructor. Acquisition SHALL be context-aware and reusable, retain successful
loads and permit retry after failure. Its state SHALL have an explicit owner,
never hidden mutable package-global state. No source, resolver, origin, path or
loader callback is passed to world construction. The acquired model is not a
selector for another model, release or CompanyCount.

**NON-NORMATIVE RECOMMENDED GO SHAPE:** promote the existing private concrete
source/model boundary, following FaceCatalogSource. The new model/acquisition
identifiers, acquisition signatures and origin-parameter representation below
are illustrative. The existing world constructor retains its frozen name and
gains the required explicit model argument.

```go
type SocioeconomicModel struct { /* unexported immutable state */ }
type SocioeconomicModelSource struct { /* unexported acquisition state */ }

func NewSocioeconomicModelSource(origins []string) (*SocioeconomicModelSource, error)
func (s *SocioeconomicModelSource) Load(ctx context.Context) (*SocioeconomicModel, error)
func NewSocioeconomicWorld(base *World, model *SocioeconomicModel) (*SocioeconomicWorld, error)
```

A long-lived application-owned source using the existing ExternalAssetsURLs is
recommended for reuse, following the cmd/core photo-source precedent. Its
placement and retention duration are implementation choices; process-lifetime
retention is not required. Model/world lifetime follows §5. No public decoder,
model setters, provider interface, registry, options framework, refresh or Close
API is needed. These declarations specify future work; none exists publicly at
this baseline.

## 4. Three lifecycle phases

### A — Model acquisition

Only the acquisition owner, through the existing External Assets resolver, may
access the network for the socioeconomic model. Acquisition resolves the fixed path
`synthetic-world/it-2026a/model.json` using the existing
`KRENALIS_EXTERNAL_ASSETS_URLS` configuration. Preserve its default and ordered
origins; introduce no second variable, caller-selected path/release, arbitrary
filesystem discovery, embedded production JSON or generated-Go model.
The large production model JSON SHALL remain outside Git.

Use the existing External Assets resolver's transport contract by reference
(`tools/externalassets/resolver.go` at the baseline above), including ordered
fallback, content decoding and final size/trailer checks. Acquisition succeeds
only after transport **and** model consumption succeed. The 8 MiB decompressed
numerical-input bound, typed closed-schema decoding, validation and bounded
immutable compilation apply here. A compiled candidate remains unpublished
until the resolver's final checks succeed; failed candidates are discarded.
No publication metadata, crosswalk or producer files are acquired for generation.
Private model-loader code is implementation evidence, not additional authority.

Reusable acquisition SHALL support safe concurrent use and reuse its successful
immutable model without fetching again through that owner. Failure remains
retryable and failed candidates SHALL NOT accumulate. Successful state SHALL NOT
be refreshed or mutated in place. Exact synchronization, waiter arbitration and
cache layout are implementation details, not public compatibility requirements.

**Runtime input admission:** the fixed External Asset SHALL carry the complete
canonical `SW-L2-D2B2-logical-v0` model prescribed by D2-B1 v0 §§9/32, v3 §4 E03
and the D2-B2 publication contract §§4–5/7/19. Acquisition SHALL validate its
closed schema, canonical numerical laws/maps and all applicable bounds; it
SHALL reject unknown/extension keys and noncanonical numerical content rather
than normalize an authored representation into an admissible runtime asset.
The published logical document obeys the canonical byte rules; acquisition
need not require byte-for-byte canonical JSON spelling, key order or whitespace.
Bounded accepted formatting variations do not change logical content or identity.

D2-A §§4–6/16/19's sorting, support merging, zero-mass omission, GCD normalization
and law-ID canonicalization remain required for admissible authored input; they
are canonicalization, not repair. D2-B production completes them before publishing
the selected runtime representation (D2-A §18 leaves physical representation to
later owners). Selecting this representation does not create a public
authored-input compiler.
D2-A §19 steps 2–6 and the model-only part of step 7 apply in A by validating
their canonical postconditions and compiling runtime state under the unchanged
input/resource limits. No D2-B fitting or producer arithmetic is rerun. Complete
step 11's model-only identity material before publishing the immutable model,
using the separately reviewed deterministic protocol; no world may lazily mutate
shared identity state. Its byte formulas are unchanged.

### B — Socioeconomic World construction

NewSocioeconomicWorld SHALL consume the authoritative base World and a
successfully acquired immutable model. It SHALL perform bounded local work
only, with no HTTP, origin fallback, filesystem discovery, transport JSON
decode, fitting, model production or publication reconstruction.

Retain Contract 1 §7 base provenance/initialization checks without modifying
World or generating a Person/photo. Reject nil or uninitialized models; only
successfully acquired immutable state is supported. Validation need not decode
or duplicate an already validated model. No arbitrary unsafe-memory validation
is promised. Callers must treat both supplied pointers as immutable, including
not overwriting their structs, and no writable model fields/slices are exposed.

Perform D2-A §19 step 1, step 7's Company/index allocation preflight, steps
8–10 and step 11's world-specific identity binding/publication. Retain the
prescribed Company metadata, forty serial Employer indexes, complete routing,
checked arithmetic and positive-total checks. Reuse model identity material;
bind the actual stored base WorldID. Check the full model-plus-world resource
charges before allocation. Return a usable immutable world only after every
local invariant succeeds; otherwise return `nil, err`. A valid model does not
guarantee that every base's realized Company universe admits construction.

### C — Random-access generation

Company/Facts/Person/ID access after successful construction SHALL use only
retained immutable state and local deterministic computation. No HTTP, resolver,
model loading, transport decode or dependence on continued origin availability
is allowed. Preserve random access, query-order independence, concurrent-read
safety, node isolation and selected-branch-only evaluation. Loading/validating
every branch during acquisition is not evaluating every branch for a Person.

## 5. Ownership, sharing and identity

| Owner | Retained state and responsibility |
| --- | --- |
| Acquisition owner | Origin/resolver configuration, acquisition coordination and reusable success/retry state. Failed candidates are not retained. |
| Immutable socioeconomic model | Validated/compiled numerical laws, maps, factors and model-only canonical identity material. No base World, generated Company roster, RNG, origin or retry/cache state. |
| SocioeconomicWorld | Authoritative base reference, shared immutable model reference, bounded world-specific Company metadata, Employer indexes and deterministic identity/runtime state. No acquisition owner or resolver dependency. |

Multiple worlds, including worlds with different valid bases, MAY share the
same successfully acquired immutable model safely. No world or caller may
mutate it. Do not copy model-sized arrays into each world merely for ownership.
An acquired model retained by a caller or world, and a retained constructed
world, SHALL remain usable if the acquisition owner is discarded, independently
of network reachability or cache ownership. Construction failure neither
invalidates the acquired model nor triggers an acquisition retry. Sharing adds
no global state.

For the same accepted canonical logical model, base and protocol, chosen origin,
URL, origin order, compression, HTTP headers, resolver/cache history and process
restart SHALL NOT change socioeconomic values or identity. No transport term
enters the base root, node fingerprints, M_all, dependency/stream records,
entity coordinates or SocioeconomicWorldID. Source and model pointer identities
are likewise irrelevant. Origin ordering may select different content when
origins disagree; this is not a guarantee that different models are identical.
There is no new content pin, authentication or release-selection mechanism.

## 6. Resource lifetime: same budgets, explicit liveness

D2-A §21 and Contract 2 resource formulas, component bounds and conservative
reservations remain unchanged. In particular E≤8 MiB, D≤8 MiB, R_out≤8 MiB,
R_out+E_live+D_live≤24 MiB, Employer input/index≤64 MiB each, R2≤72 MiB,
joint retained≤80 MiB and joint live/reserved peak≤128 MiB still apply. Preserve
the selected D2-B reservations (including E≤6.5 MiB, D≤6 MiB, the 1 MiB outside
bookkeeping allowance and H2≤1 MiB), 8N count staging, 16m raw index input,
≤32m canonicalization work and 4,096-byte control charge.

The only lifecycle reinterpretation is that numerical loading occurs in A
before local construction B. E's source-format limit applies to decompressed
JSON presented to acquisition, not compressed wire length. E_live counts actual
live encoded storage; streaming does not imply a second retained full JSON
buffer. D includes simultaneous decode/compile/identity scratch and attributable
transport/decompression working storage. Persistent acquisition bookkeeping also
needs its applicable additional retained charge. These costs are not free
because they precede construction or live outside the model object.

Model-only retained charges are part of the eventual world accounting even
when allocated in A. Add B's world-specific retained charges to them; never give
the model and world separate 8 MiB allowances. Preserve category attribution:
Employer factors F=378 belong to R2 even when stored in the shared model;
Company Industry metadata belongs to R_out and EmployeeCount metadata to R2.
Model identity digests/descriptors use the existing additional storage charges.

Preflight each phase's simultaneously live backing storage and reservations.
The joint 128 MiB limit covers A as well as B; the peak is the maximum phase
sum, not the sum of disjoint phase peaks. Copies count separately; aliases and
shared model backing count once within a live-state sum. Acquisition scratch
released before B is not charged as live in B; any overlap or retained buffer
must be charged, including in otherLiveConstructionBytes during index builds.
This does not waive a frozen conservative reservation merely because an actual
buffer was released. Failed-origin candidates/temporaries must not accumulate
across fallback or retries.

Each world's admission check includes its full applicable model contribution;
do not divide it by the number of sharing worlds to admit an oversized world.
Aggregate implementation accounting counts shared physical backing once and
all simultaneously live distinct world/acquisition state, without treating sharing
as new headroom. The frozen per-world limits do not become a newly invented
process-wide world-count limit. Actual buffer/capacity/lifetime proof remains an
implementation obligation: a decompressed-byte bound alone does not prove a
decompressor's working-memory bound, and publication's prospective numerical
charges are not an actual runtime allocation certificate.

## 7. Phase-local failures

Acquisition owns configuration, network/HTTP, content encoding/decompression,
byte-limit, JSON/schema, semantic/canonical and compilation/resource failures.
It returns no model unless the complete resolver attempt succeeds, retaining
existing bounded diagnostics and retry semantics. Use ordinary repository
errors as the existing source does; do not classify an acquisition failure as
ErrInvalidSocioeconomicWorld or invent a new public taxonomy. Public acquisition
entry points SHALL validate their documented arguments and initialization
requirements rather than panic. Their declarations SHALL document identifiable
errors actually exposed under repository and External Assets conventions;
no particular source receiver or waiter implementation is prescribed.

Construction owns invalid base/model arguments, local identity preparation,
Company/index construction, local resource bounds and retained-invariant
failures. Invalid inputs/prescribed world state retain
ErrInvalidSocioeconomicWorld classification and a nil wrapper; unexpected
internal failures remain opaque. Contract 2's hidden all-zero/overflow causes
and frozen generation-error propagation are unchanged. Generation cannot fail
because an origin has become unavailable. It still returns the frozen zero
result and non-nil error for a genuine runtime invariant failure, never partial
success, a default law or skipped Income.

## 8. Preserved behavior and vector invariance

This amendment preserves the socioeconomic public semantic types and all
unrelated frozen behavior: World, WorldConfig, Layer 0 identity/streams, Layer 1
behavior/goldens, PopulationAnchorDate, the dependency DAG, forbidden-input
isolation, Company universe and identity, modeled EmployeeCount and derived
SizeBand, Employer architecture, Income meaning and branch semantics, numerical
laws, taxonomy meanings, random access, order independence, branch
non-evaluation, immutable runtime behavior, validation requirements and all
resource ceilings subject only to §6's lifetime interpretation.

The deterministic protocol consumes the successfully acquired, validated,
compiled immutable model and owns its existing byte/sampling choices. This
amendment changes neither `socioeconomic-v1`, paths, KFD1, fingerprint formulas,
M_all, SocioeconomicWorldID, coordinates, Components, enum ordinals, age-bucket
addressing, SplitMix64, Uint64n, cumulative boundaries, Employer selection nor
Income branches. No transport state is a vector input; no vector input may
legitimately change because of this amendment.

`SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_VECTORS_v1.json` SHALL remain
byte-identical, SHA-256
`131b0de96484f61900f27daa3e182b9a526a1d1200f4b2819c084ceed2f2b218`.
Its synthetic fixture status is unchanged; it is not a production model or a
certificate of runtime implementation. Independent technical protocol review
and manual approval are complete.
