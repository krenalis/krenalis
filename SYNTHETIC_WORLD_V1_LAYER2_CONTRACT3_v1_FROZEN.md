# Synthetic World v1 — Layer 2 Detailed Contract 3 v0

## Taxonomy + Applicability

### 1. Status and frozen baseline

**Status:** proposed normative Contract 3 v0, ready for independent review. This document resolves taxonomy and applicability decisions; it is not freeze approval or implementation evidence. The independent GPT-6 Astra xhigh freeze review identified one JobFamily semantic coverage defect for competitive athletic performance; all other reviewed areas passed. This proposed correction rectifies that defect. Independent closure review remains pending.

**Requested model/configuration:** GPT-5.6 Sol xhigh.

Verified repository baseline:

```text
Branch:  property-semantics
HEAD:    76fa0fff109ca6abad62f55c8761275455842dc5
Subject: Add Synthetic World Layer 1
```

Both frozen contracts were checksum-verified before the original analysis, read completely, and checksum-verified again before the read-only semantic rectification:

| Frozen file | Verified SHA-256 |
| --- | --- |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md` | `bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md` | `e9af32a0bb9b48bf8972a408a1369901a3f3a9fe5ecd76d5762a367c055bc098` |

**NORMATIVE:** Layer 0, Layer 1, frozen Contract 1 and frozen Contract 2 remain unchanged. Contract 3 completes only their expressly deferred taxonomy and structural fields.

The unrelated modification to `tools/types/semantic.go` was preserved byte-for-byte and supplies no design precedent. Unrelated untracked files were preserved.

The specification text is the output of the read-only drafting task. Any subsequent placement or naming of that text as a repository-root review artifact is outside the drafting task and does not alter the normative Contract-3 content. No Layer-2 code is implemented. No web access, external research or generators are required or used.

### 2. Scope / non-scope

**NORMATIVE:** Contract 3 defines:

- Public taxonomy types and exact canonical constants.
- Semantic meaning and zero behavior.
- Semantic ordering where meaningful.
- Company Industry placement.
- Public CompanySizeBand, its exact thresholds and pure deterministic derivation from EmployeeCount.
- Complete employee and self-employed occupational payloads.
- Exact applicability for all six frozen PrimaryActivity values.
- Generated-value structural invariants.
- Conceptual taxonomy identity coverage and future verification obligations.

Contract 3 answers what categorical values mean and where they apply.

**DEFERRED:** probabilities, statistical correlations, causal generation, empirical crosswalks, income relationships, Company Industry distributions, employer conditioning, stream paths, canonical sampling order and exact identity encoding.

Contract 3 introduces no public function or method, taxonomy registry, runtime configuration, enumeration service, lookup service or configurable constructor.

### 3. Frozen Contract-1/2 constraints

| SOURCE FACT | Exact evidence | Contract-3 consequence |
| --- | --- | --- |
| Education and both branch pointers already belong to SocioeconomicFacts. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:120` | Preserve those field locations and types; complete EducationLevel. |
| Occupational fields and applicability are expressly deferred. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:143` | Complete EmployeeFacts and SelfEmployedFacts here. |
| Contract 1 fixes the complete conceptual public operation set. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:153` | Add types/constants and deferred fields only. |
| PrimaryActivity has six exact machine IDs; zero and unrecognized values are invalid. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:460` | Preserve all six values without additions or renaming. |
| Employee Employer is a mandatory CompanyIndex in the generating World. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:486` | Preserve the reference without copied Company identity or Industry. |
| Self-employment has no employee employer reference or owned-business entity. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:502` | Store current Industry directly in SelfEmployedFacts. |
| Both working payloads are absent in all four non-working branches. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:519` | Preserve structural absence of current occupational facts. |
| Company contains ID and EmployeeCount; further fields remain deferred. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:229` | Add Industry and derived SizeBand without altering either frozen field. |
| EmployeeCount is modeled anchor-date workforce/headcount, with valid range `0..math.MaxUint32`. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:246` | Preserve zero-count Companies and occupancy independence. |
| Zero EmployeeCount requires zero EmployerWeight in every employee-employer index. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:278` | Every generated employee employer necessarily has positive EmployeeCount. |
| Company generation never uses assigned Persons, occupancy or assignment history. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:773` | Industry remains Company-scoped; SizeBand uses only modeled EmployeeCount. |
| SizeBand thresholds are expressly deferred to this owner. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:649` | Freeze thresholds here, completing the architecture commitment. |
| Industry and SizeBand are expressly deferred to Taxonomy & Applicability / Company Contract. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:963` | Complete both Company facts here. |
| Socioeconomic generation excludes Gender and all listed snapshot/photo inputs directly and transitively. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:610` | Preserve these exclusions throughout taxonomy and applicability. |
| Facts describe the population anchor rather than ReferenceDate. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:269` | All Contract-3 facts describe `2026-01-01`. |

**SOURCE FACT:** the package already uses a defined string type with explicit typed string constants for Gender, at `tools/fakedata/layer1.go:94`. Contract 3 follows that representation pattern.

Future source declarations must follow repository declaration and comment conventions, including `AGENTS.md:23` and `AGENTS.md:243`. This does not authorize cleanup of frozen declarations.

### 4. Design principles

**NORMATIVE:**

1. **Applicability is structural.** Mandatory fields, absent fields and semantically impossible states belong here.
2. **Plausibility is model-owned.** Rarity, correlations and empirical frequency do not establish invalidity.
3. **Layer 2 is ground truth.** An inapplicable concept is structurally absent; an applicable concept has a valid semantic value.
4. **Taxonomies are broad and fixed.** Categories describe stable concepts rather than detailed statistical classifications.
5. **Occupational function and organizational scope are separate.** JobFamily describes the work; JobLevel describes employee responsibility.
6. **Facts have one authoritative location.** Employee Industry comes from the employer Company.
7. **No timeline is introduced.** Attainment, activity and occupational facts all describe the anchor.
8. **No excluded dependency enters indirectly.** Names, photographs or other derived snapshot facts cannot influence taxonomy generation.
9. **Derived categories have one result.** SizeBand is determined solely by EmployeeCount; it is never sampled independently.

**RATIONALE:** these principles permit independent structural implementations before empirical D2 acquisition, while leaving later models enough freedom to generate plausible populations.

### 5. Canonical taxonomy conventions

**NORMATIVE:** all five new taxonomy types are defined Go types with underlying type `string`:

```go
type EducationLevel string
type Industry string
type JobFamily string
type JobLevel string
type CompanySizeBand string
```

For every new taxonomy, and the existing PrimaryActivity:

- The empty string is invalid.
- Successful generated values belong to the exact constant set specified by the owning contract.
- Machine IDs are exact, lowercase ASCII strings with hyphens where shown.
- Case variants, surrounding whitespace, alternative spellings and aliases are not canonical values.
- No `unknown`, `unspecified`, `missing`, `not-provided` or `n/a` value exists.
- No public parser, validator, enumeration method or conversion helper is introduced.
- Declaration order and lexical string order have no implicit semantic meaning.
- No numeric rank is exposed.
- No overall serialization schema or field tags are defined here.

A manually cast string is not valid merely because Go permits the cast. D2 source crosswalks may interpret source spellings; that does not make those spellings generated canonical values or add a public parsing API.

Machine-ID uniqueness is required **within each taxonomy**. The same string may occur in distinct taxonomy types, such as `transport-and-logistics` in Industry and JobFamily. Type and field context distinguish those semantic domains; global string uniqueness is not required.

| Taxonomy | Semantic class |
| --- | --- |
| EducationLevel | **ORDINAL** |
| Industry | **NOMINAL** |
| JobFamily | **NOMINAL** |
| JobLevel | **ORDINAL** |
| PrimaryActivity | **NOMINAL**; frozen vocabulary preserved |
| CompanySizeBand | **DERIVED-ORDINAL**; exact workforce intervals derived from EmployeeCount |

Employee current Industry is obtained deterministically through a relationship. That does not make the Industry vocabulary ordinal. EducationLevel and JobLevel have semantic order, but neither is derived deterministically from other public facts by this contract.

**DEFERRED:** any canonical selection order needed by deterministic sampling belongs to the Deterministic Protocol Contract.

### 6. EducationLevel

**API / TAXONOMY — NORMATIVE:** define `EducationLevel string` with exactly seven constants.

| Public constant | Exact machine ID | Semantic meaning |
| --- | --- | --- |
| `EducationLevelBelowPrimary` | `below-primary` | No completed primary educational stage or higher attained qualification. Includes no formal schooling and incomplete primary schooling. |
| `EducationLevelPrimary` | `primary` | Primary education is the highest completed educational stage. |
| `EducationLevelLowerSecondary` | `lower-secondary` | Lower secondary education is the highest completed educational stage. |
| `EducationLevelUpperSecondary` | `upper-secondary` | Upper secondary education, including a vocational qualification at that stage, is the highest completed attainment. |
| `EducationLevelPostSecondaryNonTertiary` | `post-secondary-non-tertiary` | A completed formal qualification above upper secondary education but outside tertiary higher education is the highest attainment. |
| `EducationLevelTertiary` | `tertiary` | Completed tertiary higher education below a research doctorate. Includes short-cycle higher education, bachelor's, master's and non-doctoral professional higher qualifications. |
| `EducationLevelDoctoral` | `doctoral` | A completed research doctorate or educationally equivalent research-doctoral qualification is the highest attainment. |

**SEMANTIC — NORMATIVE:**

> Education means the highest completed educational stage or attained educational qualification at PopulationAnchorDate.

Vocational and adult educational equivalents belong to the stage they complete. Attendance, partial completion, informal expertise, occupational rank, an occupational license or an honorary title does not by itself establish a higher educational stage.

`below-primary` is positive ground truth about attainment. It is not an unknown value.

A professional qualification whose title contains "doctor" is not automatically `doctoral`; the doctoral category requires research-doctoral attainment.

No prerequisite history is stored. The category identifies highest attainment without asserting that every lower credential is separately recorded.

**APPLICABILITY — NORMATIVE:** every generated Person has exactly one valid EducationLevel in every PrimaryActivity branch.

A student retains their highest completed attainment. For example, a student pursuing tertiary education whose highest completed qualification is upper secondary has `EducationLevelUpperSecondary`. Student activity does not encode the level currently being pursued.

No program, course, enrollment, institution or completion-date field is added.

**ORDERING — ORDINAL:**

```text
below-primary
  < primary
  < lower-secondary
  < upper-secondary
  < post-secondary-non-tertiary
  < tertiary
  < doctoral
```

This orders completed educational stage within this synthetic vocabulary. It does not order intelligence, skill, prestige, employability, subject difficulty or income. Distinctions inside `tertiary` are deliberately unordered and unrepresented.

**RATIONALE:** seven categories preserve low attainment, school stages, post-school non-tertiary qualifications, higher education and research-doctoral attainment without splitting individual degree levels or subjects.

**SOURCE FACT:** frozen age parsing accepts 55 consecutive anchor ages beginning at 21, establishing an anchor-age range of 21–75: `tools/fakedata/layer1_data.go:133`. This vocabulary covers attainment across that range without adding age gates.

**MODEL — DEFERRED:** education probabilities, AnchorAge correlations, occupation and income relationships, and qualification-source crosswalks belong to Socioeconomic DAG / D2 and Income where applicable.

No official ISCED equivalence is asserted.

### 7. Industry

**API / TAXONOMY — NORMATIVE:** define `Industry string` with exactly fourteen constants.

Industry describes the primary economic activity of an operational Company or the current primary self-employed activity.

| Public constant | Exact machine ID | Semantic meaning |
| --- | --- | --- |
| `IndustryAgriculture` | `agriculture` | Cultivation, livestock production, forestry, fishing and related primary agricultural production. |
| `IndustryProductionAndUtilities` | `production-and-utilities` | Extraction, manufacturing and industrial processing; industrial equipment servicing; operation of energy, water, waste and utility services. |
| `IndustryConstruction` | `construction` | Construction, installation, alteration and repair of buildings, civil infrastructure and fixed building systems. |
| `IndustryCommerce` | `commerce` | Wholesale, retail and intermediation of goods transactions, including goods commerce conducted online. |
| `IndustryTransportAndLogistics` | `transport-and-logistics` | Passenger or freight movement, delivery, storage, warehousing and logistics services. |
| `IndustryHospitality` | `hospitality` | Accommodation and food or beverage preparation and service for immediate consumption. |
| `IndustryInformationAndCommunication` | `information-and-communication` | Software, computing and data services, telecommunications, publishing, broadcasting and media dissemination. |
| `IndustryFinanceAndRealEstate` | `finance-and-real-estate` | Financial, insurance and investment services; property ownership services, leasing, brokerage and real-estate management. Construction of property remains construction. |
| `IndustryProfessionalAndBusinessServices` | `professional-and-business-services` | Advisory, legal, accounting, engineering, scientific, advertising and other services supplied to organizations; business administration and support services not covered by a more specific Industry. |
| `IndustryPublicAdministrationAndSafety` | `public-administration-and-safety` | Administration of public or statutory functions, public order, defense, emergency protection and protective security services, including private protective services. |
| `IndustryEducation` | `education` | Provision of teaching, formal education, vocational instruction and educational training services. |
| `IndustryHealthAndSocialCare` | `health-and-social-care` | Clinical and veterinary healthcare, therapeutic services, nursing, social care and support for care needs. |
| `IndustryArtsAndRecreation` | `arts-and-recreation` | Artistic creation and performance, cultural activities, entertainment, sport and recreational services. |
| `IndustryOtherServices` | `other-services` | Known service activity outside all preceding Industry definitions, including household and personal services, personal-item and vehicle repair, and membership or community services whose primary activity is not otherwise covered. |

**SEMANTIC — NORMATIVE:**

- Industry concerns the entity's primary economic output or service.
- Internal employee functions do not determine Company Industry.
- The customer's Industry does not automatically determine a supplier's Industry.
- Ownership, legal form and employee occupational function are not encoded.
- Platform-specific sales channels do not create additional Industries.
- A Company with zero EmployeeCount still has exactly one valid Industry.
- Each self-employed payload has exactly one valid Industry.

The explicit sector boundaries apply to the activity being classified. For example, engineering design services are professional and business services; physical construction is construction; operation of a utility is production and utilities.

A service covered by a named specific Industry stays in that Industry even when supplied to another organization. The broad business-services description does not absorb software services, education, healthcare, transport or protective services merely because their customer is a business. Information and communication covers publishing and dissemination; artistic creation and performance belong to arts and recreation.

`other-services` is a semantic residual for a **known service activity outside the named groups**. It must not mean "the source did not identify an Industry" or "the implementation could not decide."

Companies and self-employed activities may have unrepresented secondary economic activities. Selecting one primary activity from genuinely mixed activities belongs to the later Company/D2 and Socioeconomic DAG models. This does not change the meanings or singular field cardinality frozen here.

**ORDERING — NOMINAL:** no Industry is semantically above another.

**RATIONALE:** fourteen broad groups retain major differences across the production economy, construction, commerce, transport, consumer services, knowledge services and public-facing institutions. A bounded residual completes service coverage without creating a missingness state.

**MODEL — DEFERRED:** Company Industry generation, self-employed Industry probabilities, primary-activity selection for mixed economic activities, geography correlations, Industry→JobFamily probabilities and empirical source crosswalks.

These are synthetic groups. No official ISTAT or other statistical-classification equivalence is asserted.

### 8. CompanySizeBand decision

**API / TAXONOMY — NORMATIVE: Outcome A.** v1 includes public `CompanySizeBand string` and mandatory `Company.SizeBand CompanySizeBand`.

#### Frozen-architecture reconciliation

**SOURCE FACT — frozen architecture wording supplied in the reconciliation request:**

> EmployeeCount is modeled workforce/headcount at the anchor, not realized occupancy; SizeBand is derived later.

The independent architecture review supplied with that request also recorded:

> Deriving SizeBand from EmployeeCount is sound: once band semantics are fixed, independent sampling would permit contradictory values.

These quotations are user-supplied frozen-architecture evidence, not quotations attributed to an unidentified repository file. They establish the architecture commitment being reconciled.

**SOURCE FACT:** frozen Contract 1 explicitly assigns **SizeBand thresholds** to Taxonomy & Applicability / Company contract at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:649`. Frozen Contract 2 assigns **Industry and SizeBand** to Taxonomy & Applicability / Company Contract at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:963`.

**NORMATIVE:** “derived later” is completed here as a v1 Company fact. Neither supplied frozen quotation makes existence conditional, and neither frozen deferral supplies an unambiguous optional-exposure exception. Contract 3 therefore cannot close those deferrals by omitting SizeBand. No frozen contract is reopened or modified.

Redundancy with EmployeeCount does not override that commitment. A public derived fact is acceptable only with exact consistency. Outcome B is not selected.

#### Vocabulary and exact boundaries

| Public constant | Exact machine ID | Inclusive EmployeeCount interval | Semantic meaning |
| --- | --- | --- | --- |
| `CompanySizeBandZero` | `zero` | `0` | Existing Company with no modeled workforce at the anchor. |
| `CompanySizeBandMicro` | `micro` | `1..9` | Company with a positive, single-digit modeled workforce. |
| `CompanySizeBandSmall` | `small` | `10..49` | Company with modeled workforce from ten through forty-nine. |
| `CompanySizeBandMedium` | `medium` | `50..249` | Company with modeled workforce from fifty through two hundred forty-nine. |
| `CompanySizeBandLarge` | `large` | `250..math.MaxUint32` | Company with modeled workforce of at least two hundred fifty, within the frozen uint32 range. |

The five intervals are pairwise disjoint and exhaustive over `0..math.MaxUint32`. They admit exactly one result for every valid EmployeeCount.

The canonical nonempty string `"zero"` is valid ground truth about modeled workforce. The Go zero string `""` is invalid. Zero workforce does not remove an operational Company and does not imply unknown size.

**RATIONALE:** the thresholds are assessed and adopted as fixed synthetic segmentation. A separate zero band preserves the frozen distinction between operational existence and positive workforce. Four positive bands retain useful broad differences between single-digit organizations, organizations with tens of workers, larger operating organizations and the upper tail without introducing many narrow categories. The final band covers the complete remaining representational domain.

The `10`, `50` and `250` transitions are chosen coarse grouping boundaries. They assert no empirical discontinuity, legal status, official statistical equivalence or particular workforce distribution. Narrower bins would commit unnecessary detail; collapsing all positive counts would discard the intended segmentation. No official classification is claimed.

#### Deterministic derivation

**DERIVED SEMANTIC — NORMATIVE:**

```text
SizeBand = deterministic function of EmployeeCount

count == 0                 -> zero
1 <= count <= 9            -> micro
10 <= count <= 49          -> small
50 <= count <= 249         -> medium
250 <= count <= MaxUint32  -> large
```

Every successful `Company(index)` output must satisfy:

```text
Company.SizeBand == bandFor(Company.EmployeeCount)
```

`bandFor` is specification notation for the mapping above. It does not add a public helper, mandate a private helper or justify extraction solely for tests.

The derivation:

- Uses only the Company's frozen modeled EmployeeCount.
- Makes no RNG call and consumes no independent random draw.
- Never samples SizeBand independently.
- Uses no Person state, generated employee count, realized occupancy, assignment history or generation order.
- Uses no ReferenceDate, CurrentAge or external lookup.
- Does not alter, clamp or resample EmployeeCount to obtain a desired band.
- Requires no additional probabilistic DAG variable.

An implementation may materialize or cache the derived value, but no stored, sampled or cached value may disagree with the exact mapping on successful output. Storage mechanics do not weaken the invariant.

#### Ordering, time and identity

**ORDERING — DERIVED-ORDINAL:**

```text
zero < micro < small < medium < large
```

This orders disjoint modeled-workforce intervals. It does not order revenue, compensation, influence, prestige or JobLevel. No numeric rank is exposed.

SizeBand describes modeled workforce at `PopulationAnchorDate = 2026-01-01`, remains invariant under ReferenceDate changes and derives from modeled EmployeeCount rather than realized Synthetic World occupancy.

**IDENTITY — NORMATIVE:** changing a SizeBand machine ID, boundary or derivation meaning is output-sensitive and eventually perturbs SocioeconomicWorldID through its model/spec identity. Equivalent implementation mechanics preserving the exact mapping do not. Exact identity encoding remains deferred.

**MODEL — DEFERRED:** Company/D2 owns EmployeeCount generation and any separately justified private model bucketing. Private buckets are model concepts, not substitutes for or overrides of public Company.SizeBand. Their output-affecting semantics and resource accounting remain with their later owner.

The public SizeBand exposure, vocabulary, thresholds, ordering and derivation are resolved now, not deferred.

### 9. JobFamily

**API / TAXONOMY — NORMATIVE:** define `JobFamily string` with exactly ten constants.

JobFamily describes the primary occupational function performed in the represented current working role.

| Public constant | Exact machine ID | Semantic meaning |
| --- | --- | --- |
| `JobFamilyProductionAndFieldwork` | `production-and-fieldwork` | Cultivation, extraction, physical production, assembly, industrial process operation and general physical labor supporting production or works. Craft installation and repair belong to skilled trades. |
| `JobFamilySkilledTrades` | `skilled-trades` | Craft or trade work constructing, installing, maintaining or repairing physical structures, equipment and goods. The category describes the work method, not the worker's seniority or credentials. |
| `JobFamilyTransportAndLogistics` | `transport-and-logistics` | Driving, transporting, delivering, storing, handling and dispatching people or goods, including operational logistics coordination. |
| `JobFamilyCustomerAndPersonalService` | `customer-and-personal-service` | Direct customer assistance and practical personal or household services, including reception, hospitality preparation/service, cleaning and grooming. Excludes the separately defined sales, protective, care and educational functions. |
| `JobFamilyAdministration` | `administration` | Recordkeeping, clerical processing, scheduling, routine financial administration and organizational administrative support. |
| `JobFamilySalesAndCommercial` | `sales-and-commercial` | Selling, purchasing, marketing, commercial negotiation and development of customer or supplier business relationships. |
| `JobFamilyTechnicalAndScientific` | `technical-and-scientific` | Engineering, computing, technical design, laboratory work and scientific research or analysis. Physical trade execution and direct clinical care remain in their respective families. |
| `JobFamilyProfessionalAndCreative` | `professional-and-creative` | Legal, financial, advisory, policy, organizational strategy and other professional knowledge work; journalism, artistic creation, expressive or aesthetic design, and competitive athletic performance as a performance-based occupational function. Excludes functions explicitly covered by technical/scientific, care/education, administration or commercial work. |
| `JobFamilyCareAndEducation` | `care-and-education` | Direct clinical or veterinary care, nursing, therapy, social care, personal assistance for care needs, teaching and educational instruction. |
| `JobFamilyProtectiveServices` | `protective-services` | Security, public-order protection, defense, enforcement, firefighting and protective rescue work. Direct medical treatment belongs to care and education. |

**SEMANTIC — NORMATIVE:**

- JobFamily describes function, not Company Industry.
- It does not encode seniority, management rank, formal qualification or compensation.
- Apprentices and experienced workers performing the same trade can share JobFamily.
- Supervising a function does not automatically change that function's JobFamily.
- A role primarily performing general strategy or governance work may use professional and creative; its organizational authority remains JobLevel.
- Known duties covered by a specifically named function are not placed in a broader family merely because the occupation is also considered "professional."

The distinctions concern work performed. Scientific healthcare research is technical and scientific; direct clinical treatment is care and education. Engineering design is technical and scientific; craft installation is skilled trades. Routine financial record processing is administration; financial advisory work is professional and creative; commercial negotiation is sales and commercial.

Both working branches use this same vocabulary.

**SPORT-FUNCTION BOUNDARIES — NORMATIVE:** a working role whose primary occupational function is competitive athletic performance has `JobFamilyProfessionalAndCreative` (`professional-and-creative`). Sports coaching or athletic instruction whose primary function is teaching has `JobFamilyCareAndEducation` (`care-and-education`).

These classifications follow actual primary work. Payment, professional status or employment in a sport/recreation organization alone does not select JobFamilyProfessionalAndCreative. Other sport-sector work retains its existing functional classification.

The following semantic examples also record the required self-review attack. The stated primary function controls each result:

| Represented working role | Actual primary function | Required JobFamily |
| --- | --- | --- |
| Competitive swimmer | Competing in swimming, without coaching, customer-assistance or management duties | `JobFamilyProfessionalAndCreative` |
| Professional football player | Competitive football performance | `JobFamilyProfessionalAndCreative` |
| Self-employed professional golfer | Competitive golf performance | `JobFamilyProfessionalAndCreative` |
| Sports coach | Coaching through teaching and athletic instruction | `JobFamilyCareAndEducation` |
| Personal athletic instructor | Athletic instruction | `JobFamilyCareAndEducation` |
| Sports physician | Providing direct clinical care | `JobFamilyCareAndEducation` |
| Sports physiotherapist | Providing direct therapeutic care | `JobFamilyCareAndEducation` |
| Sports scientist | Technical/scientific research or analysis | `JobFamilyTechnicalAndScientific` |
| Sports journalist | Journalism | `JobFamilyProfessionalAndCreative` |
| Sports marketer or commercial salesperson | Marketing or commercial sales | `JobFamilySalesAndCommercial` |
| Sports administrator or clerical worker | Administrative records, scheduling or clerical processing | `JobFamilyAdministration` |
| Stadium receptionist or facility customer-service worker | Reception or direct customer assistance | `JobFamilyCustomerAndPersonalService` |
| Sports-event security worker | Protective security | `JobFamilyProtectiveServices` |

Company Industry does not substitute for employee JobFamily. A professional sports club may have `IndustryArtsAndRecreation`; its competitive athlete has `JobFamilyProfessionalAndCreative`, its clerical sports administrator or accountant performing routine financial administration has `JobFamilyAdministration`, and its security worker has `JobFamilyProtectiveServices`. These examples neither change IndustryArtsAndRecreation nor create an Industry×JobFamily validity matrix.

The same competitive-performance rule applies to self-employment: when athletic competition is the represented primary function, `SelfEmployed.JobFamily = JobFamilyProfessionalAndCreative`. Direct `SelfEmployedFacts.Industry` is classified independently under the existing Industry semantics. No Employer, CompanyID, JobLevel or owned-business entity is introduced.

This is a semantic extension before Contract 3 freezes. The exact ten constants and machine IDs remain unchanged. Eventual identity coverage already includes output-sensitive JobFamily semantic boundaries; no exact identity encoding or new model dependency is added.

All thirteen roles above have a deterministic classification under their stated primary functions. The attack exposes no further coverage gap. Selecting a primary function for genuinely mixed duties remains model-owned; D2 cannot substitute for missing taxonomy meaning once that function is known.

Every represented working role has one primary JobFamily. Secondary functions and multiple concurrent roles are not represented. Generation or source mapping that selects a primary function remains model-owned; this contract freezes the meaning of the selected category.

**ORDERING — NOMINAL:** no semantic ordering exists.

**RATIONALE:** ten functional groups cover physical production, crafts, movement of goods, practical services, administration, commerce, technical work, professional/creative work including competitive athletic performance, care/teaching and protection without introducing job titles or management categories.

**APPLICABILITY — NORMATIVE:** JobFamily is mandatory in EmployeeFacts and SelfEmployedFacts, and absent in all four non-working branches.

No Industry×JobFamily validity matrix is introduced. All canonical combinations remain structurally valid at this abstraction level.

**MODEL — DEFERRED:** Industry conditioning, occupation-source crosswalks, primary-function selection, correlations and probabilities.

No official ISCO or CP2011 equivalence is asserted.

### 10. JobLevel

**API / TAXONOMY — NORMATIVE:** define `JobLevel string` with exactly four constants.

| Public constant | Exact machine ID | Semantic meaning |
| --- | --- | --- |
| `JobLevelIndividualContributor` | `individual-contributor` | Performs assigned occupational work without delegated direction of a team/workstream or unit/function-level operating authority. Includes experienced and expert contributors. |
| `JobLevelSupervisor` | `supervisor` | Has delegated day-to-day coordination and oversight of a team or operational workstream, without unit/function-level authority over operating objectives and resources. |
| `JobLevelManager` | `manager` | Has operating accountability for an organizational unit or function, with delegated authority over its objectives and resource or process choices. Direct reports are not required. |
| `JobLevelExecutive` | `executive` | Has strategic direction or governance authority for the employing organization or a major division, beyond ordinary unit/function operating management. |

**SEMANTIC — NORMATIVE:**

JobLevel describes organizational responsibility and delegated authority in the represented employee role.

It is not:

- An occupational function.
- An experience or tenure category.
- A qualification category.
- A compensation band.
- A label derived from age.
- A label derived from Company EmployeeCount.

Where a role includes several responsibility tiers, its JobLevel is the highest applicable tier under these definitions. This is a semantic classification rule, not a PrimaryActivity precedence rule or a generation probability rule.

JobLevel is **ORDINAL**, not **DERIVED-ORDINAL**: Contract 3 does not calculate it from Education, JobFamily, EmployeeCount or any other public fact. The responsibility definitions do not introduce additional public primitives or require generation of an organizational hierarchy.

These categories do not require an organizational chart, generated coworkers, generated direct reports or occupation history. A Company need not exhibit every tier. Delegated direction of a workstream means responsibility beyond performing one's own assigned tasks; individual task autonomy alone does not establish supervisor status.

An expert remains an individual contributor unless their role has the organizational authority described by a higher tier. Conversely, a low-education employee can have executive responsibility.

**ORDERING — ORDINAL:**

```text
individual-contributor < supervisor < manager < executive
```

The order describes responsibility tiers within the employing organization's delegation of authority. It does not compare absolute workforce, budget, prestige or economic influence across Companies.

**APPLICABILITY — NORMATIVE:** every employee has exactly one valid JobLevel. JobLevel is structurally absent for self-employed, unemployed, student, retired and inactive activity.

No `owner`, `unknown`, `not-applicable`, junior or senior value is introduced.

**RATIONALE:** four tiers distinguish assigned work, operational oversight, operating authority and strategic authority across shops, service organizations, factories and professional employers.

**MODEL — DEFERRED:** category probabilities and relationships with permitted experience proxies, AnchorAge, education, Company characteristics and income.

### 11. Company completion

**API / APPLICABILITY — NORMATIVE:** the exact Company shape after Contract 3 is:

```go
type Company struct {
    ID            string
    EmployeeCount uint32
    Industry      Industry
    SizeBand      CompanySizeBand
}
```

Every operational Company has exactly one valid primary Industry and exactly one valid SizeBand at the anchor, including Companies with zero EmployeeCount. SizeBand is the exact deterministic category of modeled EmployeeCount defined in section 8.

Company Industry is not determined by:

- Generated employees.
- Employees' JobFamily or JobLevel.
- Realized occupancy.
- Assignment history.
- Person request order or PersonRange.

Industry and SizeBand do not change dense operational membership. Zero-count Companies have `CompanySizeBandZero` and retain frozen zero employee-employer weight.

**SOURCE FACT:** Contract 2 allows additional Company fields and defines existence independently of future Industry criteria: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:150` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:229`.

**DEFERRED:** the Company/D2 model owns how Industry is generated. Other Company facts already deferred by the frozen contracts remain outside Contract 3.

No CompanyIndex field is added. SizeBand is Company-scoped and is not duplicated in Person or occupational payloads.

### 12. EmployeeFacts completion

**API / APPLICABILITY — NORMATIVE:**

```go
type EmployeeFacts struct {
    Employer  CompanyIndex
    JobFamily JobFamily
    JobLevel  JobLevel
}
```

For employee PrimaryActivity:

- Employee is non-nil.
- SelfEmployed is nil.
- Employer is mandatory, nonzero and operational in the generating World.
- Employer obeys frozen employee-employer eligibility.
- JobFamily and JobLevel are mandatory canonical values.

Current employee Industry is exactly the Industry of the Company resolved through Employee.Employer in the generating socioeconomic World.

EmployeeFacts contains no Industry field, copied Company, Company ID, past employer or historical occupation.

**SOURCE FACT:** the CompanyIndex-only employer relationship is frozen at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:494`. Occupational completion is delegated at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:500`.

Positive employer EmployeeCount is necessary because Contract 2 excludes zero-count Companies from every employee-employer index. It is not sufficient to establish eligibility in a particular later model index.

### 13. SelfEmployedFacts completion

**API / APPLICABILITY — NORMATIVE:**

```go
type SelfEmployedFacts struct {
    Industry  Industry
    JobFamily JobFamily
}
```

For self-employed PrimaryActivity:

- SelfEmployed is non-nil.
- Employee is nil.
- Industry and JobFamily are mandatory canonical values.
- There is no Employer or other Company reference.
- There is no JobLevel.
- No owned-business entity is manufactured.

Industry describes current self-employed economic activity. JobFamily describes the work performed in that activity.

**SOURCE FACT:** the frozen self-employment envelope expressly excludes an owned-business entity and employee employer relationship: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:504`.

**RATIONALE:** direct Industry supplies the applicable economic fact without introducing an entity relationship absent from v1.

Self-employment income remains owned by the Income Contract. No income field is added here.

### 14. PrimaryActivity applicability matrix

**APPLICABILITY — NORMATIVE:** the following matrix is exhaustive.

"Absent" means the applicable field does not exist in that branch or its containing branch pointer is nil. It does not mean an empty taxonomy value.

| PrimaryActivity | Education | Employee payload | SelfEmployed payload | Employer | Current JobFamily | Current JobLevel | Current Industry |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `employee` | Mandatory, valid | Present | Absent | Mandatory operational, eligible CompanyIndex | Mandatory in Employee | Mandatory in Employee | Employer Company.Industry; not duplicated |
| `self-employed` | Mandatory, valid | Absent | Present | Absent | Mandatory in SelfEmployed | Absent | Mandatory in SelfEmployed |
| `unemployed` | Mandatory, valid | Absent | Absent | Absent | Absent | Absent | Absent |
| `student` | Mandatory, valid | Absent | Absent | Absent | Absent | Absent | Absent |
| `retired` | Mandatory, valid | Absent | Absent | Absent | Absent | Absent | Absent |
| `inactive` | Mandatory, valid | Absent | Absent | Absent | Absent | Absent | Absent |

The frozen public discriminator constants remain exactly:

```text
PrimaryActivityEmployee     = "employee"
PrimaryActivitySelfEmployed = "self-employed"
PrimaryActivityUnemployed   = "unemployed"
PrimaryActivityStudent      = "student"
PrimaryActivityRetired      = "retired"
PrimaryActivityInactive     = "inactive"
```

**SOURCE FACT:** their exact definitions are at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:464`; the frozen branch envelope is at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:529`.

No Education value is restricted to a particular activity branch.

SizeBand is mandatory only on Company. The employee employer Company consequently has its own SizeBand, but no SizeBand field is added to EmployeeFacts, SelfEmployedFacts or SocioeconomicFacts. Self-employment does not acquire a Company entity for this purpose. The matrix remains unchanged.

### 15. PrimaryActivity overlap/model boundary

**NORMATIVE:** applicability is evaluated for the already-selected, mutually-exclusive PrimaryActivity.

Contract 3 introduces no precedence rule among employee, self-employed, unemployed, student, retired and inactive.

**SOURCE FACT:** frozen PrimaryActivity categories do not assert the absence of unmodeled secondary activities: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:476`.

**MODEL — DEFERRED:**

- The **Socioeconomic DAG Contract** owns how the synthetic model selects one primary state when generating overlapping roles.
- **D2 contracts**, under that DAG's semantics, own empirical source mappings and treatment of overlapping source categories.

Thus a working student or working retiree does not force a particular discriminator here. Once the model selects it, section 14 applies exactly.

If `student` or `retired` is selected, both occupational payloads are absent even if an unmodeled secondary working activity is conceivable. If `employee` is selected, the full employee payload is present.

This is an exact boundary on represented primary facts, not a claim that the Person has no secondary activity.

### 16. Structural validation rules

**NORMATIVE:** every successful generated result must satisfy all applicable invariants.

1. EducationLevel, Industry, JobFamily, JobLevel and CompanySizeBand values are members of their exact canonical sets.
2. Empty strings, aliases and unrecognized strings are invalid taxonomy values.
3. Every generated SocioeconomicFacts has one valid EducationLevel and one valid frozen PrimaryActivity.
4. Employee and SelfEmployed pointer presence matches section 14 exactly.
5. Employee Employer is nonzero and belongs to the generating World.
6. Employee Employer obeys frozen eligibility; consequently its Company has positive EmployeeCount.
7. Every employee has valid JobFamily and JobLevel.
8. Employee current Industry equals the referenced Company's valid Industry and is not stored independently.
9. Every self-employed payload has valid Industry and JobFamily.
10. SelfEmployedFacts has no JobLevel or Company employer relationship.
11. All four non-working branches have both working pointers nil and no represented current Employer, JobFamily, JobLevel or Industry.
12. Every operational Company has valid Industry and frozen EmployeeCount.
13. Zero EmployeeCount remains valid for an operational Company and maps exactly to CompanySizeBandZero.
14. Every Company has valid SizeBand and `Company.SizeBand == bandFor(Company.EmployeeCount)` under section 8.
15. SizeBand derivation consumes no RNG and uses no Person state, realized occupancy, assignment history, ReferenceDate, CurrentAge or external lookup.
16. No independently sampled, stored or cached SizeBand may disagree with EmployeeCount on successful output.
17. No inapplicable concept is represented by an allocated empty payload, zero placeholder or missingness constant.
18. No hard rule is added for Education×Activity, Education×JobFamily, Industry×JobFamily, JobFamily×JobLevel or Company size×JobLevel.

These requirements apply equally to Facts output and the facts contained in composite Person output. The Company invariants apply to every successful Company output, including Companies that are not eligible employers.

**SOURCE FACT:** public structs may be manually constructed or mutated; the guarantee concerns successful package outputs at return. No public facts validator exists: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:544`.

Failure results may be the zero value required by the frozen error contract. They are not successful ground-truth states.

Generation must not report success when it cannot establish these invariants. Invalid prescribed taxonomy/model state is handled through the frozen construction error contract; unexpected generation failures remain non-nil internal errors. No taxonomy-specific public error category is added.

**SOURCE FACT:** zero results and non-nil failure errors are required at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:586`.

### 17. Soft-correlation exclusions

**MODEL — NOT CONTRACT-3 VALIDITY RULES:**

```text
young -> student
old -> retired
high education -> professional occupation
large Company -> high JobLevel
Industry -> particular JobFamily
geography -> particular Industry
education -> income
JobLevel -> income
```

The following are not automatically rejected:

- An older student.
- A young self-employed Person within the frozen population.
- A low-education manager or executive.
- An unusual JobFamily within an Industry.
- An expert employee with individual-contributor JobLevel.
- An executive in a small Company.

No rarity threshold, educational prerequisite, gender condition, age cutoff or Company-size gate is introduced. The frozen exclusion of zero-count Companies from employee-employer indexes remains mandatory; it is not a new JobLevel size rule.

The broad taxonomy categories do not encode detailed licensed occupations. Contract 3 therefore does not infer qualification requirements from JobFamily.

**DEFERRED:** later Socioeconomic DAG / D2 models own plausibility tendencies. Income relationships belong to the Income Contract / DAG.

Where later age correlations are permitted, they use frozen AnchorAge, never CurrentAge or an adjustment derived from CurrentAge.

### 18. Company independence and temporal semantics

**NORMATIVE:** every Contract-3 fact describes:

```text
PopulationAnchorDate = 2026-01-01
```

**SOURCE FACT:** the frozen constant is at `tools/fakedata/layer1.go:19`.

Changing Layer-1 ReferenceDate alone leaves Education, PrimaryActivity, occupational payloads, Company Industry, EmployeeCount and SizeBand unchanged. It may change contained Layer-1 CurrentAge.

**SOURCE FACT:** frozen composite temporal semantics require this separation at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:279`. Layer 1 computes CurrentAge from its reference date at `tools/fakedata/layer1.go:543`.

No education, employment, occupation or Company timeline is added.

Company Industry remains independent of generated Persons, occupancy, assignment history and generation order. SizeBand inherits that independence through its sole input, modeled EmployeeCount. Employee occupational facts do not feed back into Company facts.

EmployeeCount retains modeled workforce/headcount semantics. SizeBand derives solely from that count, never from realized Synthetic World occupancy, and requires no RNG.

Socioeconomic generation must not use directly or transitively:

```text
Gender
CurrentAge
ReferenceDate
Layer1 SnapshotID
FaceCatalog
PhotoID
```

This prohibition covers taxonomy selection, applicability, Company facts, employer selection and stable socioeconomic identity. It includes gender-dependent names and other derived proxies.

**SOURCE FACT:** the complete exclusion and AnchorAge boundary are frozen at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:612`.

**DEFERRED:** permitted relationships among remaining facts belong to the Socioeconomic DAG. Contract 3 does not define their direction or strength.

### 19. Taxonomy identity/versioning

**NORMATIVE:** eventual SocioeconomicWorldID coverage includes the output-affecting taxonomy and applicability specification.

Coverage must include, conceptually:

- Taxonomy semantic version.
- Canonical machine-ID sets.
- Semantic interpretation of categories.
- Applicability and branch-field placement rules.
- Normative semantic ordering when used in output-affecting model semantics.
- Output-affecting source crosswalks and model interpretations.
- SizeBand machine IDs, exact thresholds and deterministic derivation semantics.

A change capable of changing generated semantic facts must be represented by the appropriate model/spec/data identity and perturb SocioeconomicWorldID.

Machine-ID changes are compatibility-sensitive even when a display label appears similar.

Pure documentation or display wording that leaves semantic interpretation and generated output unchanged need not perturb identity.

Declaration order, implementation order and storage layout do not by themselves require identity drift when semantics and generated output are identical. If a sampling-order change changes output, it is an output-affecting protocol change.

SizeBand has exactly one result for each valid EmployeeCount. Changing its machine IDs, threshold boundaries or derivation semantics is output-sensitive and must eventually perturb SocioeconomicWorldID. Changing implementation mechanics while preserving the exact mapping does not. Output-affecting private size groupings remain covered by their later owning model identity and cannot redefine the public mapping.

**SOURCE FACT:** Contract 1 requires every output-affecting Layer-2 choice to be covered at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:390`. Contract 2 distinguishes semantics from equivalent implementation changes at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:877`.

**PROTOCOL — DEFERRED:** exact identity encoding, component names, textual format, domains and version-token representation belong to the Deterministic Protocol Contract. No public identity operation is added.

### 20. Resource boundary

**NORMATIVE:** the new vocabularies contain exactly:

```text
EducationLevel:   7 values
Industry:        14 values
JobFamily:       10 values
JobLevel:         4 values
CompanySizeBand:  5 values
Total:           40 values
```

PrimaryActivity retains its six frozen values.

Taxonomy constants require negligible fixed state. Contract 3 requires no network access, runtime taxonomy loading, caller dictionaries, dynamic categories or unbounded registries.

SizeBand derivation is a bounded fixed mapping requiring no retained lookup table or per-Person state. No new numerical resource budget is necessary for these fixed vocabularies.

**DEFERRED:** empirical crosswalk tables, model data and any retained Company Industry or SizeBand representation require bounded resource accounting under their Company/D2 owners. A cached derived band is not exempt from accounting merely because the vocabulary is small.

The fixed vocabulary does not make per-Company retained data free. Future storage must be accounted for by its owner and must not weaken frozen Contract-2 budgets.

**SOURCE FACT:** Contract-2 budgets cover its defined Company metadata and selection machinery, rather than all future state: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:660`.

### 21. Consolidated public type skeleton

**API / TAXONOMY / APPLICABILITY — NORMATIVE:** this is the exact Contract-3 structural result. The prose above defines semantics and validity.

CompanyIndex and PrimaryActivity below repeat frozen Contract-1 declarations for context; they are not additions.

```go
// CompanyIndex is a one-based coordinate for a synthetic Company.
type CompanyIndex uint64

// PrimaryActivity identifies the primary activity at the anchor.
type PrimaryActivity string

// Supported primary activities at PopulationAnchorDate.
const (
    PrimaryActivityEmployee     PrimaryActivity = "employee"
    PrimaryActivitySelfEmployed PrimaryActivity = "self-employed"
    PrimaryActivityUnemployed   PrimaryActivity = "unemployed"
    PrimaryActivityStudent      PrimaryActivity = "student"
    PrimaryActivityRetired      PrimaryActivity = "retired"
    PrimaryActivityInactive     PrimaryActivity = "inactive"
)

// EducationLevel identifies highest completed attainment at the anchor.
type EducationLevel string

// Supported completed educational attainment levels.
const (
    EducationLevelBelowPrimary            EducationLevel = "below-primary"
    EducationLevelPrimary                 EducationLevel = "primary"
    EducationLevelLowerSecondary          EducationLevel = "lower-secondary"
    EducationLevelUpperSecondary          EducationLevel = "upper-secondary"
    EducationLevelPostSecondaryNonTertiary EducationLevel = "post-secondary-non-tertiary"
    EducationLevelTertiary                 EducationLevel = "tertiary"
    EducationLevelDoctoral                 EducationLevel = "doctoral"
)

// Industry identifies primary economic activity at the anchor.
type Industry string

// Supported primary economic activity groups.
const (
    IndustryAgriculture                     Industry = "agriculture"
    IndustryProductionAndUtilities          Industry = "production-and-utilities"
    IndustryConstruction                    Industry = "construction"
    IndustryCommerce                        Industry = "commerce"
    IndustryTransportAndLogistics           Industry = "transport-and-logistics"
    IndustryHospitality                     Industry = "hospitality"
    IndustryInformationAndCommunication     Industry = "information-and-communication"
    IndustryFinanceAndRealEstate            Industry = "finance-and-real-estate"
    IndustryProfessionalAndBusinessServices Industry = "professional-and-business-services"
    IndustryPublicAdministrationAndSafety   Industry = "public-administration-and-safety"
    IndustryEducation                       Industry = "education"
    IndustryHealthAndSocialCare             Industry = "health-and-social-care"
    IndustryArtsAndRecreation               Industry = "arts-and-recreation"
    IndustryOtherServices                   Industry = "other-services"
)

// JobFamily identifies the primary occupational function at the anchor.
type JobFamily string

// Supported primary occupational functions.
const (
    JobFamilyProductionAndFieldwork     JobFamily = "production-and-fieldwork"
    JobFamilySkilledTrades              JobFamily = "skilled-trades"
    JobFamilyTransportAndLogistics      JobFamily = "transport-and-logistics"
    JobFamilyCustomerAndPersonalService JobFamily = "customer-and-personal-service"
    JobFamilyAdministration             JobFamily = "administration"
    JobFamilySalesAndCommercial         JobFamily = "sales-and-commercial"
    JobFamilyTechnicalAndScientific     JobFamily = "technical-and-scientific"
    JobFamilyProfessionalAndCreative    JobFamily = "professional-and-creative"
    JobFamilyCareAndEducation           JobFamily = "care-and-education"
    JobFamilyProtectiveServices         JobFamily = "protective-services"
)

// JobLevel identifies employee organizational responsibility at the anchor.
type JobLevel string

// Supported employee organizational responsibility tiers.
const (
    JobLevelIndividualContributor JobLevel = "individual-contributor"
    JobLevelSupervisor            JobLevel = "supervisor"
    JobLevelManager               JobLevel = "manager"
    JobLevelExecutive             JobLevel = "executive"
)

// CompanySizeBand identifies the modeled workforce interval at the anchor.
type CompanySizeBand string

// Supported workforce bands derived solely from EmployeeCount.
const (
    CompanySizeBandZero   CompanySizeBand = "zero"
    CompanySizeBandMicro  CompanySizeBand = "micro"
    CompanySizeBandSmall  CompanySizeBand = "small"
    CompanySizeBandMedium CompanySizeBand = "medium"
    CompanySizeBandLarge  CompanySizeBand = "large"
)

// Company contains the generated facts for an operational Company.
type Company struct {
    ID            string
    EmployeeCount uint32
    Industry      Industry
    SizeBand      CompanySizeBand
}

// EmployeeFacts contains current employee-specific facts at the anchor.
type EmployeeFacts struct {
    Employer  CompanyIndex
    JobFamily JobFamily
    JobLevel  JobLevel
}

// SelfEmployedFacts contains self-employment facts at the anchor.
type SelfEmployedFacts struct {
    Industry  Industry
    JobFamily JobFamily
}

// SocioeconomicFacts contains socioeconomic state at PopulationAnchorDate.
type SocioeconomicFacts struct {
    Education       EducationLevel
    PrimaryActivity PrimaryActivity
    Employee        *EmployeeFacts
    SelfEmployed    *SelfEmployedFacts
}
```

Company.SizeBand is mandatory and exactly derived under section 8. It introduces no independent sampled variable or public derivation helper.

**NORMATIVE:** Contract 3 adds **no public function or method**. No public validation, parsing, ranking, enumeration or derivation operation is implied by this skeleton.

This is the exact shape at completion of Contract 3. Other fields expressly deferred by frozen Contract 1/2 remain with their owners; they are not implicitly added here.

### 22. Deferred-design ledger

| Deferred subject | Exact later owner | Required resolution |
| --- | --- | --- |
| Education probabilities and AnchorAge relationships | Socioeconomic DAG / D2 | Fixed generation semantics and evidence without redefining attainment |
| PrimaryActivity probabilities | Socioeconomic DAG / D2 | Selection model over the frozen six states |
| Overlapping activity generation | Socioeconomic DAG Contract | Selection of one mutually-exclusive represented primary state |
| Overlapping empirical activity categories | D2 contracts under Socioeconomic DAG semantics | Explicit source mapping and model treatment |
| Company Industry generation | Company/D2 Model Contract, coordinated with Socioeconomic DAG | Fixed deterministic model and permitted dependencies |
| Self-employed Industry generation | Socioeconomic DAG / D2 | Fixed model and permitted dependencies |
| Primary Industry selection for mixed activities | Company/D2 Model Contract for Companies; Socioeconomic DAG / D2 for self-employment | Select one primary economic activity without changing category meanings |
| JobFamily probabilities and primary-function selection | Socioeconomic DAG / D2 | Fixed function-selection semantics |
| Industry→JobFamily relationships | Socioeconomic DAG / D2 | Model relationships without retroactive hard validity matrices |
| JobLevel probabilities and correlations | Socioeconomic DAG / D2 | Responsibility-tier model without salary or age definitions |
| External educational, industry and occupational crosswalks | D2 contracts | Deterministic mappings, evidence, versions and resource accounting |
| CompanyCount and EmployeeCount distribution | Company/D2 Model Contract | Resolve outstanding frozen Contract-2 model inputs |
| Employer-index subsets, weights and routing | Socioeconomic DAG / Company/D2 Model Contract | Complete deterministic routing using frozen Contract-2 mechanics |
| Income facts and relationships | Income Contract / Socioeconomic DAG | Income semantics, representation and dependencies |
| Other deferred Company facts | Company Contract / Company/D2 Model Contract | Their fields, semantics, generation and resource bounds |
| Retained Industry or SizeBand data and empirical tables | Contract owning that data | Explicit bounded storage and construction accounting |
| Private EmployeeCount model bucketing, if needed | Company/D2 Model Contract | Model semantics, evidence, identity and resources; cannot redefine public SizeBand |
| Stream paths, dependency Components and sampling order | Deterministic Protocol Contract | Exact deterministic protocol |
| Exact SocioeconomicWorldID encoding | Deterministic Protocol Contract | Exact coverage representation, domains and format |

Public CompanySizeBand exposure, machine IDs, boundaries, DERIVED-ORDINAL ordering and exact deterministic derivation are **resolved as present**, not deferred. EmployeeCount generation remains model-owned.

Enrollment, historical occupation, former employer and owned-business identity are not introduced or implicitly reserved.

Missing empirical D2 evidence does not block this contract.

### 23. Required future tests

These tests are specified, not implemented.

**NORMATIVE:** use intentional fixtures. Random samples that may omit a category or activity branch are insufficient. Internal fixtures create no public runtime configuration or public test hook, and do not justify modifying frozen source.

**Taxonomy vocabulary**

For each taxonomy, require:

- Exact defined Go type and underlying representation.
- Every public constant name and exact machine ID.
- Exact closed constant set, including detection of unexpected additions.
- Uniqueness within that taxonomy; equal strings in distinct taxonomies remain permitted.
- Zero and noncanonical variants excluded from valid generated states.
- No missingness constants.
- Education and JobLevel semantic order as specified.
- CompanySizeBand DERIVED-ORDINAL order and exact EmployeeCount mapping as specified.
- No assumed Industry, JobFamily or PrimaryActivity semantic order.
- No lexical or declaration-order substitute for semantic order.

**Education**

Intentional fixtures must cover every EducationLevel and all six activity branches.

Require:

- Valid attained education in each branch.
- Student activity with completed attainment distinct from a higher qualification being pursued.
- Partial study not automatically treated as completed attainment.
- `below-primary` treated as legitimate low attainment, not unknown.
- No enrollment or current-program API introduced.
- No occupation-based or age-based attainment validity rule.

The qualification being pursued in the student fixture is explanatory fixture context, not an additional public field.

**Industry**

Require fixtures for every Industry constant and:

- Valid Industry for every operational Company in the test universe.
- Valid Industry for zero-EmployeeCount Companies.
- Valid direct self-employed Industry.
- Employee Industry resolved through the employer Company.
- Exact EmployeeFacts shape with no duplicated Industry.
- No current Industry in any non-working branch.
- `other-services` used for a known residual service activity, not missing source information.
- Specific economic activities do not change Industry merely because their customer is a business.

**CompanySizeBand boundaries and derivation**

Require exact public type, constants and Company field, with intentional boundary fixtures:

| EmployeeCount fixture | Expected SizeBand |
| --- | --- |
| `0` | `CompanySizeBandZero` (`zero`) |
| `1` | `CompanySizeBandMicro` (`micro`) |
| `9` | `CompanySizeBandMicro` (`micro`) |
| `10` | `CompanySizeBandSmall` (`small`) |
| `49` | `CompanySizeBandSmall` (`small`) |
| `50` | `CompanySizeBandMedium` (`medium`) |
| `249` | `CompanySizeBandMedium` (`medium`) |
| `250` | `CompanySizeBandLarge` (`large`) |
| `math.MaxUint32` | `CompanySizeBandLarge` (`large`) |

These fixtures test every lower/upper interval boundary and the complete domain's upper limit. Test the actual successful Company output and literal expected band, not merely an isolated duplicate of the implementation's mapping.

Also require:

- The canonical `zero` band is valid, while the empty string is invalid.
- Every successful Company output has exactly the band of its EmployeeCount.
- Repeated, reverse-order and concurrent Company generation preserve count and band.
- SizeBand is invariant under ReferenceDate changes, Person materialization order, realized occupancy and assignment history.
- Call-graph evidence and controlled internal fixtures establish that the derivation itself consumes no RNG or independent draw. Other Company generation steps may use their later frozen deterministic model; this test does not prohibit their RNG use.
- No independently sampled or inconsistent stored/cached band can escape as successful output.
- Exact API-shape checks establish no public derivation helper, enumeration method or runtime configuration.

Fixtures and probes do not authorize public test hooks, frozen-source changes or helper extraction solely for tests.

**Activity applicability**

Exercise each frozen discriminator intentionally:

```text
employee
self-employed
unemployed
student
retired
inactive
```

For each, assert every presence/absence entry in section 14.

Employee fixtures must establish operational employer membership, positive EmployeeCount and eligibility under their prescribed test model. A positive EmployeeCount alone must not be treated as universal index eligibility.

Non-working fixtures must contain no historical employer or occupation invented from their activity.

**Occupational taxonomy**

Require every JobFamily and every JobLevel constant.

Verify:

- Mandatory employee JobFamily and JobLevel.
- Mandatory self-employed JobFamily.
- Structural absence of self-employed JobLevel.
- Structural absence of both occupational taxonomies in non-working branches.
- JobFamily independent of responsibility tier.
- Cross-industry application of each JobLevel.
- Experienced individual contributors remain representable.
- Unusual but structurally valid Industry×JobFamily combinations are accepted.
- No salary, age, education or Company-size gate is introduced.
- Ordinary individual task autonomy alone is not supervisor responsibility.

**Competitive athletic performance classification fixtures**

Require intentional deterministic semantic-classification fixtures:

| Fixture | Required result |
| --- | --- |
| Employee competitive swimmer or equivalent salaried athlete whose primary work is competing, without coaching, customer-assistance or management duties | `Employee.JobFamily == JobFamilyProfessionalAndCreative` |
| Self-employed competitive athlete, such as a professional golfer, whose primary work is competing | `SelfEmployed.JobFamily == JobFamilyProfessionalAndCreative` |
| Sports coach or personal athletic instructor whose primary work is instruction | `JobFamilyCareAndEducation` in the applicable working payload |
| Two employees in one `IndustryArtsAndRecreation` Company: a competitive athlete and a clerical sports administrator | Athlete: `JobFamilyProfessionalAndCreative`; administrator: `JobFamilyAdministration` |

Also require each of the thirteen role/function examples in section 9, including direct-care, research, journalism, commercial, reception and security boundaries. Assert literal expected constants against the semantic fixtures, not merely a duplicated classification rule.

These are classification tests, not probability tests, empirical crosswalks or hard Industry mappings. Role and duty descriptions are explanatory fixture context, not new public fields. Employee fixtures preserve valid Employer and mandatory JobLevel. Self-employed fixtures preserve direct Industry and the absence of Employer, CompanyID, JobLevel and owned-business entity.

Verify exactly ten JobFamily constants and unchanged names/IDs, including `JobFamilyProfessionalAndCreative JobFamily = "professional-and-creative"`. No additional athletic, sport or performance category is introduced. No new parser, classifier, public helper or test hook is required by these fixtures.

**Invalid prescribed state and output validity**

Require evidence that malformed prescribed taxonomy state cannot escape as successful output, and that generation failure returns a non-nil error and the frozen zero result.

Caller-created invalid structs are not grounds for adding a public validator.

**Determinism and ownership**

With deliberate branch and category coverage, require:

- Repeated retrieval equivalence.
- Forward versus reverse retrieval equivalence.
- Concurrent retrieval equivalence.
- Facts versus composite-facts content equality.
- Company independence from Person materialization and assignment order.
- Returned payload mutation cannot change independently returned results or subsequent calls.

Use content equality for pointed-to payloads, not pointer-address equality.

**SOURCE FACT:** these ownership and concurrency guarantees are frozen at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:594`.

**ReferenceDate invariance**

Construct valid base Worlds differing only in ReferenceDate and assert identical socioeconomic identities, Company facts and socioeconomic facts. Include a fixture whose contained Layer-1 CurrentAge changes.

Frozen Layer-1 tests already establish WorldID/stream-root stability and SnapshotID change under ReferenceDate variation: `tools/fakedata/layer1_test.go:487`.

**Dependency exclusion**

Prove no direct or transitive dependency on:

```text
Gender
CurrentAge
ReferenceDate
SnapshotID
FaceCatalog
PhotoID
```

Require both:

- Inspection of call graphs, model inputs, derived proxies, identity inputs and stream dependency Components.
- Controlled dependency probes varying excluded state while permitted inputs remain fixed where feasible.

Equal sampled outputs alone are insufficient proof. Facts-only isolation probes must not require frozen-source changes or a new public hook.

**SOURCE FACT:** frozen Contract 1 requires call-graph evidence and a photo-dependency probe at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:768`.

**Identity**

Once encoding is specified, intentional fixtures must prove SocioeconomicWorldID perturbation for output-affecting changes to taxonomy version, machine IDs, semantic interpretation, applicability and crosswalk/model semantics. Include explicit mutations of SizeBand machine IDs, threshold boundaries and derivation semantics.

Also require no drift from equivalent declaration ordering, implementation changes, display wording, ReferenceDate or FaceCatalog changes. An equivalent SizeBand implementation preserving the exact count-to-band mapping must not cause identity drift.

These identity mutation fixtures test future versioning obligations; they are not production runtime configuration.

**Frozen compatibility**

Existing Layer 0/1 tests and goldens remain unchanged and pass. Future concurrent verification should include appropriate race checking.

No empirical-fit or probability test belongs to Contract 3.

### 24. Decision ledger

"Frozen now?" identifies decisions resolved normatively by this v0 proposal. It does not assert that independent freeze approval has occurred.

| Subject | Decision | Class | Frozen now? | Later owner if deferred |
| --- | --- | --- | ---: | --- |
| EducationLevel type | Defined `string` type | API / TAXONOMY | Yes | None |
| EducationLevel values | Exact seven constants in section 6 | TAXONOMY | Yes | None |
| Education semantics | Highest completed/attained education at anchor; mandatory in every branch | SEMANTIC | Yes | None |
| Education ordering | ORDINAL by completed educational stage; no numeric rank | SEMANTIC | Yes | None |
| Industry type | Defined `string` type | API / TAXONOMY | Yes | None |
| Industry values | Exact fourteen constants in section 7 | TAXONOMY | Yes | None |
| Industry ordering | NOMINAL; no semantic order | SEMANTIC | Yes | None |
| Company Industry | Mandatory `Company.Industry` for every operational Company | API / APPLICABILITY | Yes | None |
| Self-employed Industry | Mandatory `SelfEmployedFacts.Industry` | API / APPLICABILITY | Yes | None |
| Employee Industry duplication | Forbidden; resolve through Employer Company | APPLICABILITY | Yes | None |
| CompanySizeBand presence | Mandatory public `CompanySizeBand string` and `Company.SizeBand` | API / TAXONOMY | Yes | None |
| SizeBand values/thresholds | Exact five constants and inclusive intervals in section 8, exhaustive over uint32 | DERIVED SEMANTIC | Yes | None |
| SizeBand ordering/derivation | DERIVED-ORDINAL; sole input EmployeeCount; exactly one result; no RNG | DERIVED SEMANTIC | Yes | None |
| JobFamily type | Defined `string` type | API / TAXONOMY | Yes | None |
| JobFamily values | Exact ten constants in section 9 | TAXONOMY | Yes | None |
| JobFamily ordering | NOMINAL; no semantic order | SEMANTIC | Yes | None |
| JobLevel type | Defined `string` type | API / TAXONOMY | Yes | None |
| JobLevel values | Exact four constants in section 10 | TAXONOMY | Yes | None |
| JobLevel ordering | ORDINAL by delegated responsibility tier; not derived from other public facts | SEMANTIC | Yes | None |
| EmployeeFacts completion | Employer, mandatory JobFamily, mandatory JobLevel | API / APPLICABILITY | Yes | None |
| SelfEmployedFacts completion | Mandatory Industry and JobFamily; no JobLevel or Employer | API / APPLICABILITY | Yes | None |
| Non-working applicability | Education mandatory; both occupational payloads nil | APPLICABILITY | Yes | None |
| PrimaryActivity overlap resolution | Deferred; applicability exact after primary state selection | MODEL / SEMANTIC | No | Socioeconomic DAG for generation; D2 for source mapping |
| Industry→JobFamily probabilities | Deferred | MODEL | No | Socioeconomic DAG / D2 |
| Education probabilities | Deferred | MODEL | No | Socioeconomic DAG / D2 |
| JobLevel probabilities | Deferred | MODEL | No | Socioeconomic DAG / D2 |
| Age correlations | Deferred; permitted age input is frozen AnchorAge | MODEL | No | Socioeconomic DAG / D2 |
| Income relationships | Deferred | MODEL | No | Income Contract / DAG |
| External empirical crosswalks | Deferred; no official equivalence asserted | MODEL / DATA | No | D2 contracts |
| Stream paths | Deferred | PROTOCOL | No | Deterministic Protocol Contract |
| Exact identity encoding | Deferred | PROTOCOL | No | Deterministic Protocol Contract |

### 25. Cross-contract contradiction audit

| Case | Classification | Explanation and exact evidence |
| --- | --- | --- |
| 1. Frozen Education field vs selected EducationLevel taxonomy | **NO ISSUE** | The existing field retains its location; its deferred type is completed. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:120` |
| 2. Frozen Employee Employer vs occupational completion | **NO ISSUE** | Employer remains CompanyIndex; mandatory JobFamily and JobLevel complete the expressly deferred payload. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:486` |
| 3. Frozen SelfEmployed branch envelope vs completed payload | **NO ISSUE** | Industry and JobFamily fill the envelope without adding a Company relationship. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:502` |
| 4. Contract-2 Company fields vs Industry/SizeBand additions | **NO ISSUE** | ID and EmployeeCount are preserved; mandatory Industry and deterministically derived SizeBand complete the expressly deferred fields. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:229` |
| 5. Frozen public operation set vs new types/fields | **NO ISSUE** | Only taxonomy types/constants and deferred fields are completed; no operation is added. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:153` |
| 6. Company independence vs Industry | **NO ISSUE** | Industry is Company-scoped and does not use Persons or occupancy. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:806` |
| 7. Company independence vs derived SizeBand | **NO ISSUE** | SizeBand derives solely from frozen modeled EmployeeCount, which is independent of generated Persons, realized occupancy and assignment history. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:270` |
| 8. Employee Industry via Employer vs duplication | **NO ISSUE** | The frozen CompanyIndex-only relationship supplies authoritative Company facts; Industry is not copied. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:494` |
| 9. Self-employed Industry without CompanyID | **NO ISSUE** | Direct Industry respects the exclusion of an owned-business entity. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:508` |
| 10. JobFamily shared across employee/self-employed branches | **NO ISSUE** | Both deferred occupational payloads are completed with the same functional vocabulary. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:147` |
| 11. JobLevel employee-only applicability | **NO ISSUE** | Contract 1 expressly left JobLevel representation and applicability open; this contract resolves them. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:498` |
| 12. Non-working branches vs occupational absence | **NO ISSUE** | Both working pointers remain nil; no historical facts are inferred. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:519` |
| 13. Student activity vs completed educational attainment | **NO ISSUE** | Activity and Education are separate facts; current study need not be completed attainment. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:120` |
| 14. Mutually-exclusive PrimaryActivity vs real-world overlaps | **DEFERRED CLEANLY** | Secondary activities are unmodeled; DAG/D2 own primary-state selection and source mapping. Payload behavior is exact once selected. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:476` |
| 15. Gender exclusion vs taxonomy/applicability | **NO ISSUE** | No gender-specific value, field presence rule or dependency is introduced. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:612` |
| 16. ReferenceDate exclusion vs anchor-date facts | **NO ISSUE** | All facts remain anchor-stable and no timeline is introduced. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:271` |
| 17. Taxonomy vocabulary vs later empirical crosswalks | **DEFERRED CLEANLY** | Semantic groups are fixed here; D2 supplies external mappings without redefining them. Empirical crosswalks were deferred by Contract 1. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:484` |
| 18. Structural applicability vs later DAG correlations | **DEFERRED CLEANLY** | Presence and canonical validity are fixed; permitted causal relationships and probabilities remain DAG/model-owned. SizeBand is a deterministic category, not an independently sampled DAG variable. `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:631` |
| 19. Frozen architecture “SizeBand derived later” vs Contract-3 SizeBand decision | **NO ISSUE** | Outcome A completes the user-supplied frozen architecture commitment with a public deterministic fact and exact thresholds. The explicit frozen deferrals assign that completion here: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:649` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:963`. No frozen architecture or contract is reopened. |

No case is classified **REQUIRES CONTRACT FIX**. No frozen-contract contradiction prevents a coherent design.

### 26. Independent-review questions

1. Is each taxonomy minimal but sufficient for Plausible+?
2. Are exact machine IDs stable and semantically non-overlapping?
3. Is ground-truth missingness avoided?
4. Is Education clearly completed attainment?
5. Is Industry broad enough for the wider economy without pretending to be an official classification?
6. Is employee Industry correctly derived rather than duplicated?
7. Is self-employed Industry represented correctly without a Company entity?
8. Does mandatory CompanySizeBand correctly complete the frozen architecture, and are the selected coarse thresholds justified rather than merely conventional?
9. Are all adopted SizeBand boundaries exact, disjoint and exhaustive over `0..math.MaxUint32`, with one deterministic result and no independent sampling?
10. Are JobFamily and JobLevel orthogonal?
11. Does JobFamily cover manual/service/logistics, knowledge work and competitive athletic performance, while keeping coaching, direct care and other sport-sector functions distinct?
12. Is JobLevel cross-industry and not a salary/age proxy?
13. Are employee and self-employed payloads now structurally complete?
14. Is applicability exact for all six PrimaryActivity values?
15. Is PrimaryActivity overlap handling correctly owned without premature model rules?
16. Did any statistical tendency become a hard structural rule?
17. Is semantic ordering frozen only where genuinely meaningful?
18. Did any excluded dependency enter taxonomy/applicability?
19. Did Contract 3 invade DAG, D2, Income or deterministic-protocol scope?
20. Can Contract 3 freeze before empirical D2 acquisition?

### 27. Contract v0 verdict

Contract 3 selects one complete semantic vocabulary and applicability design:

- Seven ordinal completed-attainment categories.
- Fourteen nominal economic activity groups.
- Ten nominal occupational functions.
- Four ordinal employee responsibility tiers.
- Five derived-ordinal Company workforce bands, with exhaustive exact thresholds and no RNG.
- Company-scoped Industry, employer-derived employee Industry and direct self-employed Industry.
- Complete working payloads and exact absence in all four non-working branches.
- Mandatory Company.SizeBand derived solely from modeled EmployeeCount, and no new public operations.

Self-review found no source-style missingness value, official-classification claim, historical occupational field, arbitrary age gate, gender dependency, duplicated employee Industry, missing self-employed Industry, income definition, empirical probability commitment or runtime taxonomy configuration.

The SizeBand reconciliation completes the frozen architecture commitment without changing the approved EducationLevel, Industry, JobFamily, JobLevel, occupational payloads or activity matrix. ID uniqueness remains per taxonomy, named specific functions retain their boundaries, and ordinal JobLevel remains distinct from deterministic derivation. The subsequent athletic-performance rectification changes only the JobFamilyProfessionalAndCreative semantic coverage and its mechanically affected examples, tests and review wording. All ten JobFamily constants and machine IDs remain unchanged. The independent review reported all other areas passed; independent closure review of this correction remains pending.

The thirteen sport-role self-review fixtures in section 9 each have a deterministic classification under the stated primary function. Competitive performance is explicitly covered, coaching and direct care remain separate, and Company Industry does not determine JobFamily. No further coverage gap was found.

The remaining mixed-role and empirical mapping decisions have explicit later owners. They do not leave branch structure ambiguous.

Contract 3 can be reviewed and frozen before empirical D2 acquisition. Implementing the complete Layer-2 model still requires the remaining DAG, Company/D2, Income and deterministic-protocol contracts.

## Source-backed risks

- **Dependency leakage through full Person generation.** Frozen Person generation obtains Gender, gender-dependent names and PhotoID: `tools/fakedata/layer1.go:509`, `tools/fakedata/layer1.go:517` and `tools/fakedata/layer1.go:537`. Socioeconomic logic must preserve the facts-only dependency boundary.
- **Membership confused with employer eligibility.** Zero-count Companies remain operational but have zero employee-employer weight: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:264` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:297`. Industry must exist for them without making them eligible employers.
- **Protocol identity confused with membership.** Frozen decoding has no operational CompanyCount: `tools/fakedata/identity.go:135`. Employee Industry must be resolved in the generating socioeconomic World.
- **Future Industry or cached SizeBand storage treated as already budgeted.** Contract 2 defines dense EmployeeCount metadata at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:564` and limits its budget ownership at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:660`. Later retained Industry or cached SizeBand data needs its own owner's accounting.
- **Broad categories reinterpreted during empirical acquisition.** Source mappings remain deferred, while all output-affecting model choices require identity coverage: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:390` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:484`. D2 must map into these meanings rather than silently redefine them.
- **Modeled workforce confused with occupancy or band sampled independently.** EmployeeCount is modeled anchor workforce, not realized assigned Persons: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:246`. Its Person/occupancy independence is frozen at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:270`; the SizeBand threshold and field deferrals are explicit at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:649` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:963`. Successful SizeBand must follow the exact mapping without independent sampling.

## Deferred decisions

Probabilities, permitted causal dependencies, overlapping primary-role selection, primary Industry/function selection for mixed activities, empirical crosswalks, Company Industry generation, CompanyCount, EmployeeCount distributions, employer-index model semantics, income, retained empirical-data resources, stream paths and exact identity encoding remain with the owners listed in section 22.

No taxonomy value, occupational field presence, semantic ordering or public SizeBand exposure/threshold/derivation decision remains unresolved. SizeBand is not a probabilistic DAG variable. Private model bucketing, if needed, remains model-owned without changing public bands.

## Repository-integrity proof

The current authoritative Contract-3 source was read completely and verified before rectification:

```text
SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v0.md
SHA-256: aa0585ad6d369efe051581c078d97b146086dfb3702398d342964ebc3f703678
```

Both frozen Contract-1/2 checksums matched the expected values in section 1 before and after analysis.

The requested existing-package sanity check passed:

```text
GOCACHE=/tmp/krenalis-l2-c3-athlete-fix-gocache go test ./tools/fakedata
ok github.com/krenalis/krenalis/tools/fakedata 13.109s
```

This verifies the existing package only and supplies no Layer-2 implementation evidence. No generators were run.

During the preceding read-only rectification task:

- Before/after branch, HEAD, full `git status --porcelain=v1 -uall` and cached-index captures compared byte-for-byte using `cmp`.
- Frozen Contract-1 SHA-256 and status captures compared byte-for-byte, and its checksum matched the expected value.
- Frozen Contract-2 SHA-256 and status captures compared byte-for-byte, and its checksum matched the expected value.
- The authoritative Contract-3 source SHA-256 and status captures compared byte-for-byte; its checksum remained exactly the value above.
- `tools/types/semantic.go` SHA-256, status and working-tree diff captures compared byte-for-byte.
- `git diff --check` and `git diff --cached --quiet` passed before and after analysis.
- All thirteen before/after capture comparisons passed, including the complete unfiltered status comparison.
- All task-created temporary verification artifacts and the temporary Go cache were removed.

The preserved `semantic.go` SHA-256 was:

```text
06a7ff690b3e12cdaf623e3ffb36003d1675a6aec69b3ee59e9ead428cc17842
```

Final branch and HEAD for that task remained:

```text
property-semantics
76fa0fff109ca6abad62f55c8761275455842dc5
```

No repository file was created, replaced, modified, renamed, formatted, deleted, staged or committed by the read-only rectification task. The known tracked modification and all pre-existing untracked entries were preserved. No web access or external research was used.

The complete corrected specification is the output of the read-only drafting task. Any subsequent placement or naming of that text as a repository-root review artifact is outside the drafting task and does not alter the normative Contract-3 content.

L2 DETAILED CONTRACT 3 v0 COMPLETE — READY FOR INDEPENDENT REVIEW
