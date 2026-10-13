# Synthetic World v1 — Layer 2 Detailed Contract 5 v0

## Income

### 1. Status, authority and verified baseline

**Status:** proposed normative Contract 5 v0, ready for independent review. This document completes the public semantic Income fact and its dependency boundary. It is not freeze approval, empirical-fit evidence or a Layer-2 implementation.

**Requested model/configuration:** GPT-5.6 Sol xhigh. This records the requested configuration; it does not independently attest to the runtime model selected by Codex.

Direct repository inspection verified:

```text
Branch:  property-semantics
HEAD:    76fa0fff109ca6abad62f55c8761275455842dc5
Subject: Add Synthetic World Layer 1
```

All four authoritative frozen inputs were read completely and their exact SHA-256 values verified:

| Alias used below | Authoritative frozen file | Verified SHA-256 |
| --- | --- | --- |
| Contract 1 | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md` | `bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6` |
| Contract 2 | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md` | `e9af32a0bb9b48bf8972a408a1369901a3f3a9fe5ecd76d5762a367c055bc098` |
| Contract 3 | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md` | `df3ef0b52f37b587b8449cc71085dc9e5aca00f754bbd93e123c6fd5e3a4cb55` |
| Contract 4 | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md` | `a5ecfb735270ffba3e6afea23d85e548dd6dc3f07a8e9dfc5294a84ad3576418` |

`Contract N:line` citations refer only to the exact filename and verified content in this table. Earlier drafts are non-authoritative. Historical proposal or pending-review wording inside a supplied frozen file does not change its authority for this task.

**NORMATIVE:** frozen Layer 0/1 and Contracts 1–4 remain unchanged. Contract 5 uses the Income extension expressly deferred by Contract 1:458 and the downstream attachment expressly assigned to Contract 5 by Contract 4:732.

The stabilized Contract-5 input was read completely and its required checksum verified before review-artifact analysis:

```text
Inspected stabilized input: SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v0_STABILIZED.md
SHA-256: c33cdd2d65612bd158f5111f494ba6a85831cfe32bc9148cbf5bffdbb081b1a4
```

This is the inspected-input checksum, not a checksum of the complete corrected specification. Earlier original and review-ready Contract-5 artifacts were also present, inspected and preserved; their hashes are recorded in the repository-integrity proof as provenance only. Internal rules are resolved through this document's own sections, independently of an artifact filename or earlier line numbering.

This task completes independent-review support artifacts only: correlation mechanisms, the adversarial exercise, the expanded compatibility audit and independent-review questions. The Income design remains unchanged. The complete corrected specification is provided as a separate review artifact under the user's explicit instruction; any later replacement of an inspected source is outside this task.

The applicable root `AGENTS.md` was read. The unrelated tracked modification to `tools/types/semantic.go` and all pre-existing untracked artifacts were preserved. The modified semantic source supplies no design precedent. No web access or external research was used.

### 2. Scope and decision order

**NORMATIVE:** Contract 5 selects exactly one v1 design for economic meaning, annualization, currency, unit, integer domain, public representation, field placement, applicability, zero, activity-law selection, each branch's direct parents, employee Company access, self-employment without Company, non-working Income without occupational history, non-interference and identity obligations.

The decisions below are made in this order:

1. Economic concept.
2. Temporal meaning.
3. Currency, unit and numeric domain.
4. Public API representation.
5. All-branch applicability and zero.
6. PrimaryActivity branch selection.
7. Employee parents.
8. Self-employed parents.
9. Four non-working branch parents.
10. Non-interference.
11. D2 and Deterministic Protocol boundaries.

**DEFERRED:** empirical sources, numerical probabilities, distributions, zero frequencies, quantiles, tail parameters, empirical/model caps, fitted coefficients, model buckets, source crosswalk assets and exact deterministic sampling protocol. No empirical numerical distribution is selected here.

This contract requires no household, tax, wealth, business-turnover, employment-history or Income-source model. It adds no IncomeBand, source decomposition, history, runtime model configuration or public operation.

### 3. Decision 1 — economic concept

**SEMANTIC — NORMATIVE:** select **A: gross annual personal monetary income from all personal sources**, interpreted as the nonnegative personal Income inflow concept defined below and annualized under section 4.

> Income is the aggregate annualized rate of eligible monetary income attributable to the Person under the income conditions represented at PopulationAnchorDate, before personal income taxes and mandatory personal social contributions.

This is an individual synthetic economic measure. It is not a jurisdiction-specific taxable-income definition or a signed balance of all receipts, expenses, gains and losses. “Gross” refers to the stage before personal tax/contribution deductions; it does not turn Company or self-employed turnover into personal Income.

#### Concept evaluation

| Candidate | Decision | Reason |
| --- | --- | --- |
| A. Gross personal monetary income from all personal sources | Selected with the precise inflow and anchor-rate boundaries here | One comparable Person fact supports working and non-working states without a tax engine, household model or generated decomposition. |
| B. Employment/self-employment earnings only | Rejected | Coherent as a narrower earnings fact, but would omit pensions, benefits and other personal income and make most non-working observations structurally uninformative for the intended socioeconomic core. |
| C. Net/disposable personal income | Rejected | Would require unsupported deduction and, for disposable-resource interpretations, household rules. An unspecified net approximation would not expose a stable common meaning. |
| D. Household income | Rejected | No household entity, household membership or allocation rule exists; substituting household resources would change the unit of observation. |
| E. Branch-specific incomparable measures | Rejected | Salary, turnover, pension and available family resources cannot share one numeric field with a comparable economic meaning. |

#### Eligible personal monetary income

**SEMANTIC — NORMATIVE:** the aggregate includes the personal income rate attributable to the Person from all of these source classes when applicable at the anchor:

| Source class | Included meaning |
| --- | --- |
| Employment remuneration | Personal monetary wages, salary, ordinary monetary allowances, commissions and bonuses forming part of the current remuneration rate. Employer-only costs and nonmonetary remuneration are outside the amount. |
| Self-employment personal earnings | Monetary remuneration or positive earnings attributable to the Person after business operating costs and before personal taxes/contributions. It is the personal income portion, rather than customer receipts or business turnover. |
| Pension / retirement receipts | Personal monetary pension and retirement-income payments under the represented current receipt arrangement. |
| Public monetary benefits | Monetary benefits attributable to the Person, including income-support receipts; no benefit eligibility engine or legal entitlement claim is introduced. |
| Scholarship and similar receipts | Personal monetary scholarships, stipends and similar income-support payments under a current receipt arrangement. Fee waivers, tuition paid to an institution and other nonmonetary support are excluded. |
| Investment/property-derived personal income | Personal interest, dividends, income distributions, rents and royalties. For a property operation, operating costs are accounted for before identifying positive personal earnings; principal or capital returned is excluded. |
| Other personal monetary income | Other eligible personal monetary income under a current recurring income arrangement, subject to the same attribution, annualization and capital exclusions. This is not a missing-source category. |

Income does not require generating any source separately. These classes define the aggregate's meaning and future source-mapping boundary; they are not additional public or hidden sampled nodes.

The aggregate can include eligible secondary-source income regardless of PrimaryActivity. For example, employee Income may include dividends, retired Income may include rents, and student Income may include eligible secondary employment earnings. This does not introduce a secondary activity field, an occupational payload in a non-working branch or a latent secondary-role state. PrimaryActivity describes the represented primary activity, as already frozen by Contract 1:478 and Contract 3:586.

#### Explicit exclusions

**SEMANTIC — NORMATIVE:** exclude:

- Company revenue and Company Income of any kind.
- Business turnover or customer receipts before personal attribution.
- Business funds retained without personal income attribution; a transfer or drawing representing returned capital rather than earnings.
- Another household member's Income.
- Household transfers, allowances or shared resources merely available to the Person; inter-personal gifts, family support and inheritances.
- Wealth, asset value, capital stock and unrealized appreciation.
- Asset-sale proceeds, returned principal, capital transfers and capital gains; these are outside this v1 recurring personal-income inflow measure.
- Borrowing, credit availability, loan proceeds and debt balances.
- Customer value, predicted lifetime value and all Layer-3 facts.
- Expense reimbursements that merely restore money spent, nonmonetary benefits and imputed services.
- Personal consumption expenditure, personal debt payments and other personal outflows as deductions from Income.

A jointly reported household amount is not a personal amount merely because the Person can access it. D2 cannot divide such an amount by an invented household size or assign another person's income without an explicit compatible source mapping that establishes personal attribution independently of a generated household model.

#### Nonnegative attribution and losses

**SEMANTIC — NORMATIVE:** eligible income rates are nonnegative personal inflows. A business or property operation's loss is not negative personal monetary Income in this concept and does not offset salary, pension or another eligible inflow. Returned capital and financing used to cover that loss are not Income either.

This boundary is intentional: the measure records personal income inflows, rather than signed net economic results. It requires no personal-loss, wealth-change or tax-loss model.

A negative signed aggregate from an accounting or tax dataset is not a directly compatible observation. It must not be cast to unsigned, take its absolute value or be silently clamped to zero. A crosswalk may derive the eligible inflow aggregate only when source information supports the attribution and exclusion rules; otherwise it must reject the purported direct mapping or use the source only as documented evidence for a model assumption. Section 15 defines that distinction.

### 4. Decision 2 — temporal meaning

**SEMANTIC — NORMATIVE:** select **annualized current personal Income rate at PopulationAnchorDate**.

```text
PopulationAnchorDate = 2026-01-01
```

This is the frozen anchor, not a configurable date (`tools/fakedata/layer1.go:19`; Contract 1:269).

> “Annual” means the eligible personal monetary income rate over one complete annual payment cycle with the income-generating conditions represented at the anchor held constant.

It is a run rate, not the cash realized during 2025, 2026 or any other actual calendar year. It is not the amount realized in the preceding twelve months, a sum from a monthly history or a prediction that the Person's future conditions will remain unchanged.

| Temporal candidate | Decision |
| --- | --- |
| Historical realized calendar-year amount | Rejected: requires an unrepresented year's activity and receipt history. |
| Trailing-12-month amount | Rejected: requires an unrepresented interval and potentially different prior activity states. |
| Annualized current anchor-date rate | Selected: uses current anchor conditions and needs no timeline. |

**SEMANTIC — NORMATIVE:**

- A current monthly rate paid twelve times contributes twelve payments to its annual rate. If the represented remuneration arrangement includes thirteen or fourteen scheduled payments, its annual rate includes that complete schedule rather than applying twelve mechanically.
- Ordinary recurring bonuses, commissions, variable earnings and seasonal patterns belong to the represented annual run rate. Their rate is model-owned; v1 need not generate a payment schedule or realized monthly amounts.
- A receipt need not be paid on January 1 to contribute to the current rate. A current pension, periodic scholarship or annually paid income arrangement can contribute despite no payment on the anchor day.
- A newly current income arrangement is annualized without prorating by elapsed employment or benefit duration. No start/end date is inferred.
- Past one-off receipts and exceptional isolated capital transfers are not carried into the current rate. An expired scholarship or historical severance receipt is not current Income merely because it was paid recently.
- The annual cycle is an economic annualization convention, not a daily amount multiplied by 365 or 366. No leap-year rule or day-count convention is required.

The conceptual payment examples define annualization; they do not require generating those components or schedules.

Changing ReferenceDate alone must leave Income unchanged. Composite Person output may have a different CurrentAge, but its Income remains the same anchor-rate fact. There is no socioeconomic aging, prior state reconstruction or Income projection to ReferenceDate (`Contract 1:283`).

PopulationAnchorDate defines meaning and eventual identity coverage. It is not a free date dimension in a branch law. ReferenceDate, current calendar time and Layer-1 SnapshotID never select a price basis, period or model version.

### 5. Decision 3 — currency, unit and numeric domain

**API / SEMANTIC — NORMATIVE:**

```text
Currency:              EUR
Unit:                  one whole nominal EUR per annualized year
Underlying Go type:    uint32
Valid public range:    0..4,294,967,295 inclusive
Representation maximum: math.MaxUint32 whole EUR per year
```

Amounts use the nominal EUR basis of the anchor model. They are not USD, purchasing-power units, inflation-adjusted generic units or EUR cents. Currency and unit are fixed semantics, not per-Person fields or runtime options.

Whole EUR is selected over EUR cents. It is adequate for CDP segmentation and avoids cent-level accounting precision unsupported by a synthetic aggregate run rate. Floating point is not the public representation and must not determine compatibility-sensitive conversion or sampling through implementation-dependent rounding.

Runtime Income values, annualization, unit conversion and sampling use checked integer or exact rational arithmetic, with no floating point.

`uint32` is sufficient for the v1 whole-EUR representation: its maximum is EUR 4,294,967,295 annually. In cents, the same type would represent only EUR 42,949,672.95 and imply finer precision. Neither number is a claim about plausible Italian personal Income.

The unsigned type follows the nonnegative inflow concept in section 3. It is not an instruction to repair a signed net-income value by removing its sign.

#### Whole-EUR quantization and strict zero preservation

**SEMANTIC CONVERSION — NORMATIVE:** a compatible nonnegative annualized rate `x`, expressed exactly in EUR before whole-EUR quantization, maps to:

```text
Q(x) = ceil(x)
```

Validate `0 <= x <= 4,294,967,295` before conversion. Integer-valued rates are unchanged. Any positive fractional-EUR rate rounds upward to the next whole EUR, preserving the strict distinction between no Income and positive Income. The upward error is less than EUR 1 annually.

| Compatible rate in exact EUR/year | Public whole-EUR value |
| ---: | ---: |
| `0` | `0` |
| `0.01` | `1` |
| `1` | `1` |
| `1.01` | `2` |
| `4,294,967,295` | `4,294,967,295` |
| `4,294,967,295.01` | Rejected before conversion |

These are representation fixtures, not distribution parameters or typical Income amounts. Ceiling is selected specifically so that quantization cannot turn positive Income into the no-Income ground truth. It is explicit unit conversion, not empirical saturation or a positive-income plausibility floor.

`x` and `Q` are specification notation. They introduce neither a public operation nor a requirement to generate a hidden fractional rate. A D2 model may generate valid whole-EUR values directly. Any source/model conversion from finer monetary precision must implement this same exact quantization, using checked integer or exact rational arithmetic.

**RESOURCE / CORRECTNESS — NORMATIVE:**

- Every value in `0..math.MaxUint32` is structurally representable, including the maximum.
- A negative, non-finite, unrepresentable or above-maximum purported compatible aggregate is invalid. Fractional exact monetary source rates are allowed only through the specified conversion; the public value is always an integer.
- Annualization products, rate sums, rational numerators/denominators and unit conversions must be checked before arithmetic, allocation or narrowing can overflow.
- No overflow, wraparound, silent saturation, sign reinterpretation or post-failure fallback is allowed.
- A purported compatible rate above the maximum is rejected even if some alternative rounding would fit.

**MODEL — DEFERRED:** a future empirical/model cap is distinct from the representation maximum. D2 owns any lower support limit or explicit bounded-tail law, its evidence and identity. Such a law may intentionally generate from a bounded distribution; it must not disguise clipping or repair of an invalid draw as successful generation. Contract 5 freezes no empirical cap or branch-specific Income minimum.

### 6. Decision 4 — minimal public API and placement

**API — NORMATIVE:** add one defined scalar type:

```go
// AnnualIncomeEUR is gross personal monetary Income annualized at the anchor
// in whole EUR.
type AnnualIncomeEUR uint32
```

Add exactly one field:

```go
Income AnnualIncomeEUR
```

Its authoritative location is `SocioeconomicFacts.Income`. It is a mandatory value field, not a pointer, optional wrapper or source-bearing struct. Preserve the existing four SocioeconomicFacts fields in their frozen relative order and append Income. Do not duplicate it in Person, either occupational payload or Company.

**Contract 5 adds no public function or method.**

The constructor and retrieval boundary remain exactly those frozen by Contract 1:153. Income is returned by Facts and in Person's contained Facts through the same unexported socioeconomic generation logic (`Contract 1:263`).

No parser, validator, rank API, IncomeBand, currency converter, history accessor, breakdown, new sentinel, public configuration, public Income version constant or new wrapper operation is introduced. The full `uint32` range supplies the scalar domain; no reserved missingness value exists.

#### Exact post-Contract-5 public type skeleton

The following repeats existing types for context and adds only AnnualIncomeEUR and the Income field. Existing taxonomy constants remain exactly as frozen in Contracts 1 and 3; none is added or changed. Private wrapper state is deliberately not an implementation layout contract.

```go
// SocioeconomicWorld provides deterministic socioeconomic random access.
type SocioeconomicWorld struct {
    base *World
    // Private immutable model, Company, index and identity state omitted.
}

// SocioeconomicWorldID identifies a stable socioeconomic configuration.
type SocioeconomicWorldID string

// SocioeconomicPerson combines a Layer 1 Person with anchor-date facts.
type SocioeconomicPerson struct {
    Person Person
    Facts  SocioeconomicFacts
}

// AnnualIncomeEUR is gross personal monetary Income annualized at the anchor
// in whole EUR.
type AnnualIncomeEUR uint32

// CompanyIndex is a one-based coordinate for a synthetic Company.
type CompanyIndex uint64

// CompanySizeBand identifies the modeled workforce interval at the anchor.
type CompanySizeBand string

// EducationLevel identifies highest completed attainment at the anchor.
type EducationLevel string

// Industry identifies primary economic activity at the anchor.
type Industry string

// JobFamily identifies the primary occupational function at the anchor.
type JobFamily string

// JobLevel identifies employee organizational responsibility at the anchor.
type JobLevel string

// PrimaryActivity identifies the primary activity at the anchor.
type PrimaryActivity string

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
    Income          AnnualIncomeEUR
}
```

The complete public operation set remains:

```go
func NewSocioeconomicWorld(base *World) (*SocioeconomicWorld, error)

func (w *SocioeconomicWorld) Company(index CompanyIndex) (Company, error)
func (w *SocioeconomicWorld) Facts(index PersonIndex) (SocioeconomicFacts, error)
func (w *SocioeconomicWorld) ID() SocioeconomicWorldID
func (w *SocioeconomicWorld) Person(index PersonIndex) (SocioeconomicPerson, error)

func CompanyIDForIndex(namespace IdentityNamespace, index CompanyIndex) (string, error)
func CompanyIndexFromID(namespace IdentityNamespace, id string) (CompanyIndex, error)
```

These are skeletons and signature notation, not implemented files. They imply no serialization tags, new JSON schema or new public constants. Future declarations retain the identifiable-error documentation and repository conventions already required by Contract 1.

### 7. Decision 5 — all-branch applicability and zero

**APPLICABILITY — NORMATIVE:** every successfully generated Person has exactly one mandatory Income value. The same economic concept, anchor annualization, currency, unit and domain apply across all six PrimaryActivity states.

**ZERO — NORMATIVE:**

> Income == 0 means no personal monetary Income under the frozen Income concept at the anchor.

More precisely, the eligible current annualized rate is zero. It does not mean that no money was paid on January 1 or that the Person has no wealth, household support or available credit. Those are different, unmodeled concepts.

Zero must never mean unknown, not measured, not applicable, not provided, source-mapping failure, failed generation or absent occupational data. The zero-preserving whole-EUR conversion in section 5 ensures that a positive compatible rate cannot become this no-Income state.

| PrimaryActivity | Income applicable? | Income mandatory? | Zero structurally valid? | Frozen occupational envelope |
| --- | --- | --- | --- | --- |
| `employee` | Yes | Yes | Yes | Employee present; SelfEmployed absent; valid eligible Employer, JobFamily and JobLevel |
| `self-employed` | Yes | Yes | Yes | SelfEmployed present; Employee absent; Industry and JobFamily; no Company or JobLevel |
| `unemployed` | Yes | Yes | Yes | Both working payloads absent |
| `student` | Yes | Yes | Yes | Both working payloads absent |
| `retired` | Yes | Yes | Yes | Both working payloads absent |
| `inactive` | Yes | Yes | Yes | Both working payloads absent |

An employee role does not structurally guarantee a positive eligible personal monetary rate; remuneration may be nonmonetary or the current monetary rate may be zero. Self-employment may have no positive personal earnings. Retired activity does not guarantee a positive pension receipt. These statuses do not justify introducing salary, pension or profit as separately generated prerequisites.

Likewise, non-working primary activity does not require zero or prohibit positive Income. PrimaryActivity is not an exhaustive Income-source classification. Unusual zero or high values are future model-plausibility concerns, not new structural invalidity.

No activity-specific positive minimum, pension entitlement, salary ordering, education-income gate or maximum shared with Company size is frozen. A model may assign zero probability to a structurally valid value in some condition; that does not redefine the public domain.

A failed Facts or Person operation still returns its frozen zero result with a non-nil error. Its zero Income field is not a successful ground-truth observation. A zero-value SocioeconomicFacts remains invalid because its Education/activity envelope is uninitialized (`Contract 1:590`; Contract 1:476).

### 8. Decision 6 — PrimaryActivity selects the Income law

**BRANCH SELECTION — NORMATIVE:** one mandatory public Income node is evaluated using exactly one of six branch-specific laws, selected by the already-generated PrimaryActivity.

```text
PrimaryActivity chooses the Income branch law.
PrimaryActivity is not an ordinary numerical conditioning dimension
inside the already-selected branch law.
```

Each branch has separate law/model ownership and the closed parent set below. The laws share the same outcome semantics; they need not share probabilities or shape. Law sharing or pooling is permitted only if it preserves branch-specific identity and non-interference obligations.

`Income[e]`, `Income[s]`, `Income[u]`, `Income[t]`, `Income[r]` and `Income[i]` below denote conditional-law templates for employee, self-employed, unemployed, student, retired and inactive. They are not six public fields, six simultaneous samples or hidden socioeconomic facts.

For a requested Person:

1. Validate the wrapper and operational PersonIndex under Contract 1.
2. Acquire the permitted frozen Layer-1 inputs and generate Education and PrimaryActivity under Contract 4.
3. Evaluate only the applicable Contract-4 occupational subsequence.
4. Select the corresponding Income law and supply only that branch's authorized information.
5. Generate one valid Income value and return the complete facts only on success.

Do not evaluate an absent branch's occupation, route a non-employee to an employer, sample six candidate incomes and select afterward, or select an activity from a sampled Income. Constructor loading of immutable model assets is not per-Person law evaluation.

Changing PrimaryActivity may change Income by changing the selected law and its applicable parents. Selection counts toward semantic descendant reasoning. It does not add occupational facts to non-working branches (`Contract 4:358`).

### 9. Decision 7 — employee Income parents

**DAG — NORMATIVE:** the exact employee Income **value-conditioning set** is:

```text
{
    Region,
    Employee.JobFamily,
    Employee.JobLevel,
    Employee.CurrentIndustry,
    Company.SizeBand[Employer]
}
```

Employer additionally supplies a **relational key only** to obtain authorized selected-Company information. It is not an Income class or additional probability-table dimension.

The complete direct information set, including that key dependency, is therefore:

```text
Income[e] direct parents = {
    Region,
    Employee.JobFamily,
    Employee.JobLevel,
    Employee.CurrentIndustry,
    Employer [relational key only],
    Company.SizeBand[Employer]
}
```

Employee.CurrentIndustry is exactly `Company.Industry[Employer]`, the already-frozen authoritative relational information (`Contract 4:311`). It is the **sole sector dimension**. Listing it does not authorize a second independent Company.Industry dimension or a copied EmployeeFacts.Industry field.

#### Independent candidate evaluation

| Candidate | Decision | Plausible+ mechanism and minimality reason |
| --- | --- | --- |
| AnchorAge | Omit as a direct employee Income parent | Lifecycle association can arise through age -> JobLevel, age -> Education -> occupation and activity selection. Residual age effects at a fixed complete employee Income value-parent vector are intentionally not modeled. Age never means tenure. |
| Region | Include directly | Allows broad personal Income-rate differences within the same function, responsibility, sector and employer size. Upstream occupational-composition paths alone cannot express that residual difference. It remains the Person's frozen Region, not employer location or commuting distance. |
| Education | Omit directly | Its Income relationship can arise through JobFamily, JobLevel and activity selection. No additional residual education premium at a fixed employee parent vector is necessary for this v1 core. |
| Employee.JobFamily | Include directly | Broad functions can have different monetary-rate profiles within the same responsibility, sector and size. Responsibility and employer matching do not replace function. |
| Employee.JobLevel | Include directly | Permits responsibility-related Income profiles within a broad function and employer context. It remains delegated authority, not a salary band. No mandatory monotonic salary ordering is introduced. |
| Employer coordinate | Relational key only | Locates the selected Company facts. Numeric coordinate, modulo, hash, identity token and Company-specific random intercept cannot classify employee Income. |
| Employee.CurrentIndustry | Include as the sole sector value | Selected employer sector can affect the Income law within fixed function/responsibility/size. Matching function to sector upstream does not express this residual sector effect. |
| Company.Industry[Employer] | Same information, not an additional dimension | Implementations may obtain the sector through this exact lookup, but must not condition twice or distinguish provenance from the same authoritative value. |
| Company.EmployeeCount[Employer] | Omit directly | Broad size is supplied by SizeBand. There is no authorized residual employee-count effect within a fixed band. Count can still affect Income upstream through Employer selection or a changed SizeBand. |
| Company.SizeBand[Employer] | Include directly | One coarse size dimension permits employer-context differences within function, responsibility and sector without workforce-sized detail. Its exact count-derived meaning remains frozen. |
| Company.ID | Forbid | Protocol identity has no personal Income meaning and is not an authorized socioeconomic classifier. |

Direct Region does not reopen Employer conditioning: Employer remains conditioned on JobFamily and JobLevel with its frozen Company-side information. Income must not revise Employer or occupations to fit regional pay.

Gross all-source Income does not require directly adding every correlate of a possible secondary source. D2 models the aggregate conditional law using this small set. Asset-income or age/education effects not expressible through the authorized mechanisms remain deliberate v1 limitations.

#### Selected Company isolation

**RELATIONAL — NORMATIVE:** given `Employer = j`, employee Income may inspect only these semantic facts of Company `j`:

```text
Company.Industry[j]   [only as Employee.CurrentIndustry]
Company.SizeBand[j]
```

Operational validation may establish that `j` is a member. If SizeBand is derived on demand, reading `Company.EmployeeCount[j]` solely to apply the exact frozen SizeBand mapping is permitted derivation machinery. The raw count must not reach the Income law, its innovation dependencies or an additional lookup key.

Retrieving a Company value that also contains ID and EmployeeCount does not authorize using those fields. No new public Company lookup operation or public projection API is required.

Changing Company `k != j` must not directly change Income when Employer, authorized selected-Company values, other Income parents, the employee Income law/model/data and Income addressing remain fixed. Income must not read an unselected Company, a universe summary, Industry composition, size histogram, employer-index weights, routing identity or CompanyCount as an extra dimension.

Two valid employers with the same Industry and SizeBand are indistinguishable to the employee Income law when its other parents, own model/data and Person-scoped innovation are fixed. Replacing one by the other must not change Income solely because their coordinates or IDs differ. Employer is not included in Income stochastic addressing as an arbitrary Company-specific seed.

Changing an unselected Company can affect family-wide Employer selection under Contract 4. If selection changes Employer, a newly selected Industry or SizeBand may change Income through the authorized relational path. If Employer and all Income value parents remain unchanged, Income remains unchanged despite upstream model/metadata changes. These indirect selection paths do not authorize direct universe inspection by Income.

Changing a selected positive EmployeeCount **within a fixed SizeBand**, with Employer and all other Income value parents fixed, must not change Income. Crossing a frozen SizeBand boundary may change the law through the band value. No inconsistent count/band fixture or independently sampled band is allowed.

### 10. Decision 8 — self-employed Income parents and meaning

**DAG — NORMATIVE:**

```text
Income[s] direct parents = {
    AnchorAge,
    Region,
    SelfEmployed.JobFamily,
    SelfEmployed.Industry
}
```

| Candidate | Decision | Reason |
| --- | --- | --- |
| AnchorAge | Include directly | Permits broad lifecycle aggregate-Income differences within current function and Industry. This branch has no JobLevel path to carry responsibility/lifecycle variation. It is not tenure, business age or years trading. |
| Region | Include directly | Permits broad personal Income differences within the same function and Industry. Upstream family composition alone cannot supply this residual regional effect. |
| Education | Omit directly | Association can arise through Education -> SelfEmployed.JobFamily and primary-activity selection. A residual premium at fixed age, Region, function and Industry is not another v1 dimension. |
| SelfEmployed.JobFamily | Include directly | Functional monetary-earning differences can remain within the same Industry. Industry's upstream dependence on JobFamily does not replace that functional distinction. |
| SelfEmployed.Industry | Include directly | Economic-activity differences can remain within the same function and age/Region. Its conditional generation under Contract 4 does not make the two facts interchangeable. |
| Employer, Company, CompanyIndex, Company.ID, EmployeeCount, SizeBand or JobLevel | Absent and forbidden | None is an applicable self-employed fact. Do not invent a business entity, workforce surrogate, employer or responsibility tier. |

**SEMANTIC — NORMATIVE:** self-employed Income uses exactly the same gross personal aggregate as every other branch. Self-employment personal earnings are attributed after business operating costs and before personal taxes/contributions; eligible pension, benefit or investment receipts may also contribute to the conceptual aggregate.

It is not turnover, gross invoices, business valuation, retained business funds, cash drawings regardless of origin or a proxy Company salary. Zero can represent no eligible personal monetary Income, including no positive personally attributable earnings. No profitability, capital, employee-count or business-history node is generated.

The aggregate law may represent uncertainty and heterogeneity as one Income outcome. It may not sample hidden business size, affluence, tenure, profit or component-source facts to create additional dependencies. The frozen self-employed occupational direction stays family -> Industry (`Contract 4:341`).

### 11. Decision 9 — the four non-working Income laws

**DAG — NORMATIVE:** select the following exact sets separately:

```text
Income[u] direct parents = {AnchorAge, Education, Region}
Income[t] direct parents = {AnchorAge, Education, Region}
Income[r] direct parents = {AnchorAge, Education, Region}
Income[i] direct parents = {AnchorAge, Education, Region}
```

Their dimensions match; their branch laws and model/data ownership remain distinct.

| Branch law | AnchorAge justification | Education justification | Region justification |
| --- | --- | --- | --- |
| Unemployed | Broad lifecycle differences in aggregate personal receipt rates without unemployment duration or a former job | Completed attainment can distinguish aggregate receipt profiles at fixed age and Region when no current occupational channel exists | Broad regional receipt-rate differences at fixed age and attainment |
| Student | Broad lifecycle differences among the frozen adult students, without study duration | Completed attainment can distinguish stipend/support and other eligible aggregate receipt profiles; it is not the program pursued or a scholarship entitlement | Broad regional receipt-rate differences without institution or finer geography |
| Retired | Broad lifecycle differences in current aggregate pension and other eligible income rates, without retirement date | Completed attainment permits broad aggregate receipt differences without inventing historical occupation, salary or contributions | Broad regional aggregate-rate differences without former employer location |
| Inactive | Broad lifecycle differences in eligible aggregate receipt rates without inferring the reason for inactivity | Completed attainment supplies a limited current common socioeconomic dimension when occupation is absent | Broad regional differences without a household, wealth or local-benefit engine |

Each direct edge supplies a residual distinction that primary-activity selection alone cannot express **within that already-selected branch**. Education is included here because no current JobFamily or JobLevel mediation exists. This does not assert a numerical sign, strength, entitlement or causal reconstruction of past earnings.

**SEMANTIC — NORMATIVE:**

- Unemployed does not mean no benefits, no investments or no eligible secondary income. It creates no prior employment, last salary, unemployment start date or benefit-contribution record.
- Student does not mean a stipend exists, and completed Education does not encode current course, institution, duration or eligibility. Shared household support remains excluded even when it supports study.
- Retired does not require a pension, former employer, former JobFamily, last JobLevel, retirement age, career duration or contribution history. Income is the current aggregate rate, not pension replacement of a generated former salary.
- Inactive creates no reason-for-inactivity, hidden household role, incapacity state, caring role, wealth state or occupational history. It can have zero or positive eligible personal Income.

Only AnchorAge, current completed Education and frozen Region may be read by these laws. Both working payloads remain nil, and their models, employer routing and Company lookups must not be evaluated for the Person. No current or former Industry, JobFamily, JobLevel or Employer is manufactured and then discarded.

D2 can fit a broad aggregate distribution from evidence without reproducing a person's benefit or pension calculation. Such empirical aggregation does not authorize generating the omitted histories or source states.

### 12. Authoritative Income parent table, edges and acyclicity

**NORMATIVE:** this table and the edge list are authoritative together. A parent set is closed authorization, not a demand that every dimension have different numerical effects in production. Addressing and a law's own model/data are separate from semantic facts, as defined by Contract 4:92.

| Law template | Class | Exact direct value parents | Additional direct relational key | Selection |
| --- | --- | --- | --- | --- |
| Income[e] | PERSON MODEL FACT | {Region, Employee.JobFamily, Employee.JobLevel, Employee.CurrentIndustry, Company.SizeBand[Employer]} | Employer; key only | employee |
| Income[s] | PERSON MODEL FACT | {AnchorAge, Region, SelfEmployed.JobFamily, SelfEmployed.Industry} | None | self-employed |
| Income[u] | PERSON MODEL FACT | {AnchorAge, Education, Region} | None | unemployed |
| Income[t] | PERSON MODEL FACT | {AnchorAge, Education, Region} | None | student |
| Income[r] | PERSON MODEL FACT | {AnchorAge, Education, Region} | None | retired |
| Income[i] | PERSON MODEL FACT | {AnchorAge, Education, Region} | None | inactive |

**DAG — NORMATIVE:** Income adds exactly these conditional direct information edges:

```text
Region -> Income[e]
Employee.JobFamily -> Income[e]
Employee.JobLevel -> Income[e]
Employee.CurrentIndustry -> Income[e]
Employer -> Income[e]                         [relational key only]
Company.SizeBand[Employer] -> Income[e]        [selected payload only]
AnchorAge -> Income[s]
Region -> Income[s]
SelfEmployed.JobFamily -> Income[s]
SelfEmployed.Industry -> Income[s]
AnchorAge -> Income[u]
Education -> Income[u]
Region -> Income[u]
AnchorAge -> Income[t]
Education -> Income[t]
Region -> Income[t]
AnchorAge -> Income[r]
Education -> Income[r]
Region -> Income[r]
AnchorAge -> Income[i]
Education -> Income[i]
Region -> Income[i]
```

There are 22 conditional information-edge templates: 21 value edges and one relational-key edge. The sector payload reaches employee Income through the existing Employee.CurrentIndustry node; it is not listed again as an independent Company.Industry edge.

**LAW SELECTOR — NORMATIVE:** separately:

```text
PrimaryActivity -- employee law --> Income[e]
PrimaryActivity -- self-employed law --> Income[s]
PrimaryActivity -- unemployed law --> Income[u]
PrimaryActivity -- student law --> Income[t]
PrimaryActivity -- retired law --> Income[r]
PrimaryActivity -- inactive law --> Income[i]
```

These six selector relationships count for descendant reasoning. They select how the mandatory common Income exists as a realized value; they do not make its applicability optional or add six ordinary probability dimensions.

**ACYCLICITY PROOF:** preserve every Contract-4 node, its 23 direct semantic edge templates and six occupational gates unchanged. Append Income after the applicable Contract-4 subsequence. Every new value parent, selected payload and relational key already exists earlier; PrimaryActivity is already earlier. No Income edge returns upstream, and no branch evaluates another branch's occupation or Income law. Therefore the completed socioeconomic core remains acyclic, including law-selection relationships.

### 13. Correlation mechanisms

This explanatory table restates the closed Income graph in sections 9–12 and the relevant frozen Contract-4 paths. It describes permitted information routes, not empirical sign, strength, ordering or a guaranteed correlation. It adds no parent, hidden component, distribution or structural gate. NOT MODELED IN V1 denotes an omitted economic effect; FORBIDDEN denotes information or feedback that may not enter generation.

| Relationship | Mechanism | Exact explanation |
| --- | --- | --- |
| AnchorAge × employee Income | UPSTREAM CONTRACT-4 PATH | AnchorAge -> Employee.JobLevel -> Income[e]; also AnchorAge -> Education -> Employee.JobFamily -> Income[e] and AnchorAge -> Education -> Employee.JobLevel -> Income[e]. No direct employee age edge or residual age effect at fixed employee Income value parents; age is not tenure. |
| AnchorAge × self-employed Income | DIRECT INCOME EDGE | AnchorAge -> Income[s], allowing a lifecycle distinction within the authorized current function/Industry context without tenure or business history. |
| AnchorAge × non-working Income | DIRECT INCOME EDGE | AnchorAge -> each of Income[u], Income[t], Income[r] and Income[i]; no former occupation or receipt history is reconstructed. |
| Region × employee Income | DIRECT INCOME EDGE | Region -> Income[e], including distinctions within fixed function, responsibility, selected sector and SizeBand. Region is the frozen Person Region only. |
| Region × self-employed Income | DIRECT INCOME EDGE | Region -> Income[s]; no Company location or finer residential geography is introduced. |
| Region × non-working Income | DIRECT INCOME EDGE | Region -> each of the four separately owned non-working Income laws; no local household or benefit engine is introduced. |
| Education × employee Income | UPSTREAM CONTRACT-4 PATH | Education -> Employee.JobFamily -> Income[e] and Education -> Employee.JobLevel -> Income[e], also Education -> Employee.JobFamily -> Employee.JobLevel -> Income[e]. No residual direct education effect at fixed employee Income value parents. |
| Education × self-employed Income | UPSTREAM CONTRACT-4 PATH | Education -> SelfEmployed.JobFamily -> Income[s] and Education -> SelfEmployed.JobFamily -> SelfEmployed.Industry -> Income[s]. No direct self-employed education edge. |
| Education × non-working Income | DIRECT INCOME EDGE | Education -> each of Income[u], Income[t], Income[r] and Income[i]; it is completed attainment, not a reconstructed qualification, pension or eligibility history. |
| PrimaryActivity × Income | PRIMARY-ACTIVITY BRANCH EFFECT | The already-generated state selects exactly one of six common-outcome branch laws. It is not another numerical dimension inside that selected law. |
| Employee.JobFamily × employee Income | DIRECT INCOME EDGE | Employee.JobFamily -> Income[e]; the value means occupational function, not salary or employer Industry. |
| Employee.JobLevel × employee Income | DIRECT INCOME EDGE | Employee.JobLevel -> Income[e]; delegated responsibility may distinguish conditional laws without a mandatory Income ordering. |
| Company Industry × employee Income | SELECTED-EMPLOYER RELATIONAL INPUT | Company.Industry[Employer] -> Employee.CurrentIndustry -> Income[e]. CurrentIndustry is the sole direct sector value; the lookup authorizes no second sector dimension or unselected Company read. |
| Company SizeBand × employee Income | DIRECT INCOME EDGE | Company.SizeBand[Employer] -> Income[e] is a direct selected-employer value. Employer is only its relational key; the band retains the frozen count-derived meaning. |
| Company EmployeeCount × employee Income | UPSTREAM CONTRACT-4 PATH | For Employer=j, Company.EmployeeCount[j] -> Company.SizeBand[j] -> Income[e]. Company.EmployeeCount[*] can also affect Employer selection -> selected Employee.CurrentIndustry -> Income[e] or selection -> selected Company.SizeBand[Employer] -> Income[e]. At fixed Employer, SizeBand and other Income value parents, raw count has no residual Income effect. |
| Employer identity/coordinate × employee Income | NOT MODELED IN V1 | An economic coordinate/ID effect is absent. Employer supplies only lookup access to authorized selected values; equal payloads are Income-equivalent under fixed other Income parents, own law/model/data and Person innovation. Arbitrary employer-specific seeding, intercepts and classifiers are forbidden. |
| SelfEmployed.JobFamily × Income | DIRECT INCOME EDGE | SelfEmployed.JobFamily -> Income[s], as current primary function. It creates no employee responsibility tier or business entity. |
| SelfEmployed.Industry × Income | DIRECT INCOME EDGE | SelfEmployed.Industry -> Income[s], as current primary economic activity; it is distinct from function and from business turnover. |
| Gender × Income | FORBIDDEN | No direct or transitive Income dependency, proxy, model key or stochastic addressing dependency may use Gender. |
| CurrentAge × Income | FORBIDDEN | Only the authorized node-specific AnchorAge input is available. CurrentAge cannot condition Income or be used to reconstruct AnchorAge. |
| ReferenceDate × Income | FORBIDDEN | Changing only ReferenceDate leaves anchor Income unchanged; it cannot select a rate period, price basis, model or stream. |
| Layer-3 facts × Income | FORBIDDEN | Future Layer-3 facts may consume Income only under their own contracts; no Layer-3 fact, customer value or downstream calibration may feed back into Income. |

### 14. Decision 10 — Income non-interference

**NORMATIVE:**

```text
Contract-4 facts -> Income: permitted only through the exact sets above
Income -> Contract-4 fact: forbidden
```

No Income amount, law, source-data identity, stream identity, sampled proxy, calibration score or derived band may feed into:

```text
Company.Industry
Company.EmployeeCount
Company.SizeBand
Education
PrimaryActivity
Employee.JobFamily
Employee.JobLevel
Employer
Employee.CurrentIndustry
SelfEmployed.JobFamily
SelfEmployed.Industry
```

An Income-model/data/protocol change must leave all these facts unchanged with their own models, permitted inputs and addressing fixed, even when the full SocioeconomicWorldID changes. Income cannot cause upstream resampling, rejection followed by replacement, employer reweighting, career matching or post-generation reconciliation. Do not reopen Contract 4 to improve Income fit (`Contract 4:734`).

#### Model and branch isolation

| Changed model/data | Income that may change | Facts that must remain identical under fixed other models/inputs/addressing |
| --- | --- | --- |
| Employee Income law | Employee Income and any future expressly authorized downstream descendants | Every Contract-4 fact; Income for self-employed and all four non-working branches |
| Self-employed Income law | Self-employed Income and its authorized future descendants | Every Contract-4 fact; Income in all other branches |
| One non-working Income law | Income in that one activity branch and its authorized future descendants | Every Contract-4 fact; Income in the other five branches |
| Common output-affecting Income semantic/conversion protocol | Income in any branch affected by that versioned change | Every Contract-4 fact and frozen Layer-0/1 outputs |
| Layer-3 law/data | No Layer-2 Income change | All Layer-2 facts and Income addressing |

Permission to change a descendant is not a requirement that each sampled value differ. Independent law changes must nevertheless be demonstrably isolated; accidental sampled equality is insufficient evidence.

Contract-4 model changes may propagate into Income only by changing law selection or authorized parent values. In particular:

- Education can affect employee Income through occupation/responsibility or activity; it has no extra direct employee premium.
- AnchorAge can affect employee Income through frozen upstream paths or activity; its employee residual at fixed authorized values is absent.
- Company count/Industry/weight/routing changes may affect employee Income through selected Employer payloads, not through global metadata fingerprints.
- Self-employed Industry changes can affect self-employed Income but cannot alter its JobFamily or employee Income.
- Non-working Income cannot change directly because a Company or working-branch occupational model changed.

**FIXED-PARENT INVARIANCE — NORMATIVE:** if law selection, all applicable Income value parents, its own relevant law/model/data and permitted Person addressing/innovation are unchanged, Income must remain unchanged. Upstream model identities, storage representations, unused dimensions or global checksums cannot add information beyond those values. The Employer-key replacement equivalence in section 9 also applies.

#### Excluded inputs and hidden state

**NORMATIVE:** Income must not depend directly or transitively on:

```text
Gender
CurrentAge
ReferenceDate
Layer1 SnapshotID
FaceCatalog
PhotoID
```

The only authorized Layer-1 socioeconomic facts remain `{AnchorAge, Region}`, with node-specific uses in the table above. AnchorAge is the separate result of frozen birth logic, never reconstructed from CurrentAge (`tools/fakedata/layer1.go:355`; Contract 4:133).

Region is exactly the private Region of the canonical municipality selected by the frozen Layer-1 address municipality prefix (`Contract 4:139`; `tools/fakedata/layer1.go:267`; `tools/fakedata/layer1_data.go:396`). It does not authorize Province, Municipality, PostalCode, street, Place ID, population, birthplace or employer geography. Region pooling may coarsen this information; it cannot recover finer distinctions.

Gender-dependent names, birth year/date, photo metadata, household-resource proxies, customer value, generated histories, latent socioeconomic class, latent business size and affluence scores are not alternative routes into Income. The no-hidden-socioeconomic-state rule remains frozen (`Contract 4:377`).

Income law sampling must have its own isolated Person-scoped innovation. Reusing an occupation draw, employer draw or another branch's sampled result to induce an undeclared dependency is forbidden. Private deterministic buckets may use only the selected law's parents and own model/data; they cannot smuggle unauthorized ancestor values into the law.

#### Random access and facts-only isolation

Income is bounded local generation from immutable model state. It does not inspect another Person, realized occupancy, request history, PersonRange, a shared advancing RNG or mutable registry. No per-Person Company-universe scan or allocation proportional to income amount or Company workforce is allowed.

Repeated, forward, reverse and concurrent retrieval agree by contents. Facts-only and composite Facts expose identical Income for the same socioeconomic World and PersonIndex. Mutating a returned Income or occupational payload cannot affect internal state or another independently returned result.

Facts-only generation must not call full frozen Person generation or select a photo. Full Person generation obtains Gender/names/photo (`tools/fakedata/layer1.go:509`, `tools/fakedata/layer1.go:517`, `tools/fakedata/layer1.go:537`); its contained snapshot fields cannot become Income inputs. Existing lower-level input acquisition and unexported generation boundaries suffice without modifying frozen source.

### 15. Decision 11 — D2/model boundary

**NORMATIVE:** Contract 5 owns the outcome meaning, period, currency/unit, representation, applicability, zero, exact branch parents and non-interference. Later **Income MODEL/D2 contracts** own numerical laws and evidence under those fixed boundaries.

| Subject | Frozen here | Later Income MODEL/D2 responsibility |
| --- | --- | --- |
| Common outcome | Gross personal monetary aggregate rate, as precisely bounded above | Fit a compatible aggregate law; do not substitute salary, turnover or household resources |
| Time | Annualized current anchor conditions | Explicit source-period reconciliation and annualization assumptions without generated history |
| Monetary basis | Whole nominal anchor-basis EUR; exact ceiling conversion | Versioned deterministic source currency/price-basis conversion where needed; no runtime converter |
| Numeric domain | `0..math.MaxUint32`; zero preserved | Valid bounded production support, any empirical/model cap and exact safe conversion |
| Six branch laws | PrimaryActivity selects one law; exact sets | Separate numerical laws, parameters and output-affecting identities; pooling only within the boundary |
| Zero | Valid no-Income ground truth in every branch | Zero frequencies and conditional support, with evidence rather than a structural positivity mandate |
| Model transforms | Parent-only, deterministic and information-preserving or coarsening | Exact age/Region/education/occupation buckets, coefficients and thresholds; no latent or extra dimension |
| Selected Company | Sector once and SizeBand only | No arbitrary employer intercept or count sensitivity within a fixed band |
| Source interpretation | Personal attribution and inclusion/exclusion requirements | Deterministic crosswalks, documented coverage, source provenance and checksums |
| Resources | Bounded immutable local generation; no new state mandated | Concrete finite input, retained-state, construction-peak and lookup-work bounds for the representation introduced |

D2 must document a deterministic source-to-concept mapping before labeling source values as directly compatible Income observations. The mapping must state personal versus household unit, source coverage, gross versus net treatment, business-to-person attribution, excluded capital flows, source period, annualization, currency/price basis, precision, zero treatment and valid-domain checks.

A historical annual amount, salary-only statistic, signed taxable aggregate or household total does not automatically map to this anchor-date all-source measure. A source may inform model calibration through explicit assumptions and limitations, but must not be described as a directly equivalent measurement when it is not. Missing decomposition cannot be relabeled as confirmed zero or complete all-source coverage.

Offline evidence may describe histories, components or finer demographic/geographic categories. Such evidence does not authorize those as generated Person parents. The frozen v1 law must use only its selected branch information, and any aggregation/marginalization or approximation used to form that law must be documented. No runtime record lookup keyed by Person/Company identity or excluded attribute is permitted.

D2 may pool parent values or give a dimension no numerical effect. It may not add Education to employee or self-employed Income, AnchorAge to employee Income, EmployeeCount within employee SizeBand, finer Region detail, Company information to non-employees, occupation history, source states or a shared hidden score. A numerical fit requiring one of those residual relationships remains a v1 limitation; it is not permission to widen this contract or Contract 4.

#### Initialization, validation and errors

**NORMATIVE:** the existing constructor establishes and validates the prescribed immutable Income law/model/data before a usable wrapper escapes. Every reachable activity-and-parent condition must have one deterministic valid law resolution. Pooling is allowed; missing or ambiguous coverage, malformed data and invalid arithmetic are not.

There is no unrelated runtime fallback, skipped Income, default zero for mapping failure, partial successful facts, reload API or caller-selectable Income model.

Invalid prescribed Income model/data or failure to establish its required identities returns no wrapper and the frozen invalid-world classification. Unexpected internal failures remain non-nil opaque errors under Contract 1's existing distinction. Generation cannot report success without producing the mandatory valid Income. No new public Income error identity is needed. Frozen generation-error identities and Contract-2 constructor weight-error hiding remain unchanged; neither rule is broadened or weakened here.

The scalar adds four bytes of value payload per returned facts value and requires no retained per-Person table. This is not an allocator/RSS guarantee or permission to allocate from the whole numeric domain.

Contract 5 does not freeze a dense Income lookup table, distribution representation, routing structure or new empirical-table budget. Before implementation, the later owner introducing Income assets must freeze enforceable finite resource bounds and account for source/loading, retained and simultaneous construction storage before potentially excessive allocation. It must preserve the no-context constructor's bounded local work and all frozen Contract-2 budgets. Model availability and resource completion remain later implementation prerequisites, not contradictions in this semantic contract.

### 16. Deterministic Protocol boundary

**PROTOCOL — DEFERRED:** the later Deterministic Protocol Contract owns exact stream paths, Component names, framing, seed bytes, canonical sampling order, inverse-CDF/table/interpolation mechanism, draw sequence, law routing representation and SocioeconomicWorldID encoding. Existing frozen Layer-0 machinery and Contract-2 employer-selection semantics remain unchanged.

**NORMATIVE:** the later protocol must realize:

- One mandatory Income outcome and exactly one selected law per Person.
- The exact branch value sets and restricted Employer key/payload access.
- Person-scoped fresh random access, independent of Employer coordinate as an arbitrary random effect.
- Separate branch/law/model identities sufficient for the isolation table.
- No excluded input in a stream root, dependency Component, law key or proxy.
- Fixed-parent invariance despite upstream data/model identity changes.
- Exact integer outcomes and the frozen whole-EUR conversion wherever conversion is required.
- Repeat-order, reverse-order and concurrent equivalence.

No stream string, Component identifier or concrete sampling algorithm is invented by this semantic contract. No universal one-raw-draw claim is made. Frozen bounded sampling can use rejection internally (`tools/fakedata/random.go:48`); whatever Income sampling the later protocol freezes must preserve the semantic boundaries here.

Frozen fresh entity streams derive RNG state from root, coordinate, path and dependency Components (`tools/fakedata/stream.go:36`, `tools/fakedata/stream.go:63`). Those extension points permit Income isolation without changing Layer 0. They do not supply that isolation automatically: using full SocioeconomicWorldID or broad whole-asset checksums in every stream would violate this contract.

Binding the actual stable Layer-1 WorldID in the full socioeconomic identity is required, but it must not turn unrelated Layer-1 dataset changes into Income realization changes when the approved input values and permitted addressing remain fixed (`Contract 4:763`). The protocol owns compatible selective decomposition, not additional semantic permission.

### 17. Identity and versioning obligations

**IDENTITY — NORMATIVE:** eventual SocioeconomicWorldID must bind the actual base WorldID and all output-affecting Layer-2 spec/model/data/protocol choices already required by Contracts 1–4. Income coverage includes:

- The common economic meaning and personal-attribution boundary.
- Eligible source interpretation and capital/household/loss exclusions.
- PopulationAnchorDate and current-rate annualization.
- EUR, nominal monetary basis, whole-EUR unit and exact quantization semantics.
- Public Income type/placement and all-branch mandatory applicability/zero meaning.
- The six-law selector semantics and exact branch parent sets.
- The Employer-key restriction, one authoritative sector dimension and selected-Company isolation.
- Each branch's output-affecting law/model/data identity, source crosswalk, numerical coefficients, buckets, tail/support choices and empirical/model cap if selected later.
- Output-affecting Income sampling and addressing protocol semantics.

Changes to any of these semantic or output-affecting choices require appropriate spec/model/data/protocol version review and eventual full identity change. Two Worlds exposing identical integers with a different currency, period or economic meaning do not represent the same Income specification.

Income-specific identity changes do not change Person IDs, Company IDs, the frozen base WorldID, Layer-1 SnapshotID or any Contract-4 fact. The broader SocioeconomicWorldID must change while selective generation isolation remains true. It must never be an RNG mega-seed (`Contract 1:405`).

Physically shared source files may have provenance checksums, but generation identities must isolate the output-affecting portion belonging to each branch/node. Changing one Income branch's data must not perturb another branch just because both reside in one file. The full identity may cover both portions without passing their combined identity to every innovation.

Pure wording changes, equivalent storage/routing, implementation scheduling or arithmetic implementations preserving all meanings and outcomes do not independently require identity drift. A resource-only guardrail change preserves the frozen distinction: no drift for a World valid under both versions when its semantic model and outputs remain unchanged. Selecting a different actual support/model is output-sensitive even if accompanied by a resource change.

The public underlying type and field shape are also Go API commitments. A future representation redesign requires API review even when the common numeric subset would be unchanged. No public version constant, identity composer, parser or new observation-key type is added.

The composite observation key remains the frozen tuple:

```text
(SocioeconomicWorldID, Layer1 SnapshotID, PersonIndex)
```

Income is a common anchor fact within that observation, not a new identity domain or Income timeline (`Contract 1:425`). Exact identity bytes and component naming remain protocol-owned.

### 18. Required future verification

These are obligations for future implementation, not tests implemented or run against Layer 2 by this specification task. Use intentional branch fixtures and independent call-graph/model-input audits; random samples and accidental equality cannot prove the contract.

| Area | Required evidence |
| --- | --- |
| Public shape | Exact `AnnualIncomeEUR uint32`, mandatory `SocioeconomicFacts.Income`, unchanged existing fields/types and operations; no additional Income API, field, band, config or missingness value |
| Economic mapping | Literal compatible/incompatible source fixtures distinguish personal Income from salary-only, turnover, household support, capital return, asset proceeds and signed net aggregates; check personal attribution and documented calibration assumptions |
| Annualization | Current periodic/annual schedules use their complete annual rate; no dependence on payment occurring January 1, elapsed employment, historical year or ReferenceDate |
| Domain and quantization | `0`, positive sub-EUR, exact whole EUR, fractional positive EUR, MaxUint32 and above-maximum fixtures; zero remains exact no Income; checked conversion and annualization reject overflow before narrowing |
| Loss boundary | Negative purported compatible aggregates reject rather than wrap, take absolute value or clamp; losses do not masquerade as negative inflows or offset unrelated eligible receipts |
| All-branch applicability | Exercise all six activities deliberately; each has exactly one Income and can structurally expose zero or positive Income while retaining the frozen occupational envelope |
| Employee parent set | Controlled models exercise Region, function, responsibility, sector and SizeBand independently; age/education changes at fixed applicable value parents cannot directly perturb employee Income |
| Company sector authority | Current employee Industry equals the selected Company's Industry; Income receives sector once, with no copied field or second sector dimension |
| Selected Company isolation | Fix Employer=j; any attempted unselected Company access observably fails. Change k!=j with all selected values/model/addressing fixed and require identical Income |
| Employer-key equivalence | Substitute an operational employer with the same Industry/SizeBand under fixed other values and Person innovation; Income is identical despite different coordinate/ID |
| EmployeeCount exclusion | Vary selected positive count within one correct SizeBand and require identical Income; cross a real band boundary to show permitted band propagation; never fabricate count/band inconsistency |
| Indirect selection path | A controlled unselected-Company change can alter Employer selection and selected payloads upstream; Income may then change only through those payloads, without direct universe reads |
| Self-employment | Exercise age, Region, function and Industry; fixed-parent Education changes have no direct effect. Company, Employer and JobLevel evaluation must observably fail if invoked |
| Non-working laws | Each of unemployed/student/retired/inactive uses only age, attained Education and Region; probes reject any occupational/history/Company evaluation, beyond merely checking nil final payloads |
| Law selection | Exactly the selected Income law is evaluated; inactive-law probes cannot fire in employee generation, and conversely. Income cannot select or revise PrimaryActivity |
| Upstream non-interference | Mutate each Income law/model/data/protocol identity separately; force a distinguishable Income outcome and require every named Contract-4 fact exactly invariant |
| Branch isolation | One branch-law identity/data mutation changes no other branch's Income under fixed other models/inputs/innovation, including when full world identity changes |
| Fixed-parent invariance | Upstream identity/metadata changes with the complete applicable Income value vector and own law/innovation fixed leave Income identical |
| Excluded inputs | Audit call graphs, raw/derived proxies, stream roots, Components and model keys for all six exclusions; controlled probes vary excluded state while permitted inputs remain fixed where feasible |
| Geography | Same Region with different finer municipality/Province/postal/population detail leaves Income law/output unchanged at fixed other values/addressing; no independent socioeconomic municipality draw |
| No hidden state | No sampled source decomposition, affluence/business/household score, prior role, source-eligibility flags or reused cross-node innovation couples Income to undeclared facts |
| Facts-only/composite | Equal Income by contents; facts-only performs no full Person/photo generation and preserves frozen input acquisition and index validation |
| Determinism/ownership | Repeated, forward, reverse and concurrent results agree; caller mutation cannot alter independent results/internal state; appropriate future race verification |
| Initialization/errors/resources | Complete reachable-law coverage; malformed/overflowing/oversized prescribed assets fail before invalid allocation or wrapper escape; no successful default zero or partial facts; all frozen error identities preserved |
| Identity | Income semantic/data/law changes perturb full identity as required while base identities and upstream facts remain unchanged; resource/equivalent implementation changes obey section 17; ReferenceDate/catalog variation preserves Income and socioeconomic identity |
| Frozen compatibility | Existing Layer-0/1 tests/goldens remain unchanged and pass once implementation is undertaken; Income adds no change to their source or public APIs |

Controlled internal version fixtures do not create production runtime configuration, public test hooks or permission to modify frozen source. Do not extract an operation solely to test it; verify actual generated outputs and meaningful production boundaries under the repository rules.

Empirical-fit tests belong to the later Income MODEL/D2 owner. No numeric fit target, percentile, zero proportion, salary ratio or minimum is frozen by these structural verification obligations.

### 19. Adversarial semantic exercise

This is a semantic review exercise, not an empirical plausibility test or a new validity matrix. Single-Person cases presume the existing canonical taxonomy, operational Employer and pointer-envelope requirements and an Income value within section 5's domain. “Low” and “high” define no thresholds. Role descriptions are explanatory context, not additional generated facts. Structural validity does not guarantee positive probability under a later D2 law.

The classification vocabulary is STRUCTURALLY VALID, STRUCTURALLY INVALID, or VALIDITY DEPENDS ON A SPECIFIC STATED SEMANTIC CONDITION. For comparison cases, the stated invariance conditions determine whether the paired outcomes comply; failure of a required condition is not repaired by resampling upstream facts.

| # | Case | Classification | Reason / exact condition |
| --- | --- | --- | --- |
| 1 | Employee with Income = 0 | STRUCTURALLY VALID | Zero is legitimate no-Income ground truth even with a valid current employee role; personal monetary positivity is not a role invariant. |
| 2 | Low-Income employee | STRUCTURALLY VALID | No structural wage floor is frozen. A valid value in the common domain is not invalid because it is low. |
| 3 | High-Income employee | STRUCTURALLY VALID | Any amount within the public domain remains structurally representable. Rarity and later model support do not create an empirical public cap. |
| 4 | Low-education executive | STRUCTURALLY VALID | Canonical low attainment and executive responsibility can coexist; neither Contract 3 nor Contract 4 imposes an educational prerequisite or Income ordering. |
| 5 | Expert individual contributor | STRUCTURALLY VALID | JobLevelIndividualContributor explicitly includes expert contributors. Expertise is explanatory role context, not a new public tier or compensation rule. |
| 6 | Finance employee | STRUCTURALLY VALID | Selected finance Industry is an authorized CurrentIndustry value; function is classified independently, with no finance-specific structural pay rule. |
| 7 | Hospitality employee | STRUCTURALLY VALID | Selected hospitality economic activity and a canonical function can coexist; Industry does not force function, responsibility or a particular Income. |
| 8 | Employee at a micro Company | STRUCTURALLY VALID | The Employer has positive modeled count and the exact derived micro band. No small-Company Income or JobLevel gate is added. |
| 9 | Employee at a large Company | STRUCTURALLY VALID | The Employer has positive modeled count and the exact derived large band. Large size does not require high Income or a senior JobLevel. |
| 10 | Self-employed lawyer | STRUCTURALLY VALID | A primary legal-work function is JobFamilyProfessionalAndCreative; direct self-employed Industry remains separately classified. Personal Income requires no Company or qualification gate. |
| 11 | Self-employed electrician | STRUCTURALLY VALID | Primary craft installation/repair is JobFamilySkilledTrades. Income is personal eligible earnings and other eligible inflows, not invoices or turnover. |
| 12 | Self-employed competitive athlete | STRUCTURALLY VALID | Primary competitive performance uses JobFamilyProfessionalAndCreative under frozen Contract 3. It adds no Company, Employer, JobLevel or sport-specific Income rule. |
| 13 | Self-employed Person with Income = 0 | STRUCTURALLY VALID | No positive personal earnings or other eligible current inflow is required by self-employment. |
| 14 | Unemployed Person with Income = 0 | STRUCTURALLY VALID | The aggregate eligible current rate can be zero; both occupational payloads remain absent. |
| 15 | Unemployed Person with positive Income | STRUCTURALLY VALID | Eligible benefits or other personal inflows can yield positive aggregate Income without a current or former occupational payload. |
| 16 | Student with Income = 0 | STRUCTURALLY VALID | Student status does not guarantee a scholarship, stipend or other eligible personal inflow. |
| 17 | Student with positive Income | STRUCTURALLY VALID | Eligible stipends or other personal income can be positive. Household support remains excluded and no occupational payload is required. |
| 18 | Retired Person with positive Income | STRUCTURALLY VALID | Current pension or other eligible personal inflows can contribute; no former salary, employer or contribution history is needed. |
| 19 | Retired Person with Income = 0 | STRUCTURALLY VALID | Retired status does not guarantee a pension or any positive eligible personal rate. |
| 20 | Inactive Person with Income = 0 | STRUCTURALLY VALID | Inactive primary activity can have no eligible current personal inflow; no reason-for-inactivity state is inferred. |
| 21 | Inactive Person with positive Income | STRUCTURALLY VALID | Eligible personal investment, benefit or other inflows can be positive without household, wealth or occupational-history facts. |
| 22 | Same complete authorized Income parents, different Gender | VALIDITY DEPENDS ON A SPECIFIC STATED SEMANTIC CONDITION | With branch selection, own Income law/model/data and legitimate Person innovation also fixed, Income must be identical. A Gender-induced difference would be structurally invalid dependency leakage. |
| 23 | Same anchor socioeconomic facts, different ReferenceDate | VALIDITY DEPENDS ON A SPECIFIC STATED SEMANTIC CONDITION | Changing only ReferenceDate leaves Income identical; contained Layer-1 CurrentAge may differ. ReferenceDate-induced Income drift would violate anchor invariance. |
| 24 | Employee fixed to Employer j; unselected Company k changes | VALIDITY DEPENDS ON A SPECIFIC STATED SEMANTIC CONDITION | With all authorized selected payloads, other Income value parents, own law/model/data and addressing fixed, Income must be identical. A direct unselected-Company effect is invalid; an upstream changed Employer is a different scenario. |
| 25 | Employee fixed to Employer j; selected Company Industry changes | VALIDITY DEPENDS ON A SPECIFIC STATED SEMANTIC CONDITION | CurrentIndustry must equal the changed Company.Industry[j]. Income may change through that authorized value or remain equal. Keeping stale CurrentIndustry, or feeding Income back to Company, would be invalid. |
| 26 | Income model/data changes while every Contract-4 model/input remains fixed | VALIDITY DEPENDS ON A SPECIFIC STATED SEMANTIC CONDITION | Under fixed Contract-4 addressing, every Contract-4 fact must remain identical. Income and output-affecting full socioeconomic identity may change; a branch-specific law change must also preserve the other Income branches. |

### 20. Decision ledger

“Resolved here” means selected normatively by this v0 proposal, not independently approved for freeze.

| Subject | Decision | Class | Resolved here? | Later owner if deferred |
| --- | --- | --- | --- | --- |
| Economic concept | A: nonnegative gross personal monetary Income inflow from all eligible personal sources | SEMANTIC | Yes | None |
| Personal attribution | Business costs before identifying personal earnings; personal taxes/contributions not deducted | SEMANTIC | Yes | D2 compatible source mappings |
| Non-income resources | Company turnover, household resources, wealth/capital, credit and customer value excluded | SEMANTIC | Yes | None |
| Period | Annualized current rate under anchor conditions, not realized calendar/trailing history | SEMANTIC | Yes | D2 source reconciliation |
| Currency/unit | Whole nominal anchor-basis EUR/year | SEMANTIC / API | Yes | D2 conversion assets if needed |
| Scalar | Defined `AnnualIncomeEUR uint32` | API | Yes | None |
| Field | One appended mandatory `SocioeconomicFacts.Income` | API / APPLICABILITY | Yes | None |
| Domain | `0..math.MaxUint32` inclusive; no reserved value | SEMANTIC / REPRESENTATION | Yes | None |
| Fractional conversion | Exact ceiling after valid-range check; positive remains positive | SEMANTIC CONVERSION | Yes | D2 implementation and protocol realization |
| Zero | No eligible personal monetary rate at anchor; valid in every activity | SEMANTIC | Yes | D2 frequencies |
| Negative values | Outside inflow concept; no silent signed-net clamp | SEMANTIC | Yes | D2 compatible mapping |
| Law selection | Existing PrimaryActivity selects exactly one of six laws; not a within-law numerical dimension | DAG / SELECTOR | Yes | Protocol realization |
| Employee values | Region, JobFamily, JobLevel, CurrentIndustry, selected SizeBand | DAG | Yes | D2 laws |
| Employer | Key only; same selected semantic payload means no coordinate-specific Income effect | RELATIONAL | Yes | Protocol realization |
| EmployeeCount | No residual count effect within a fixed SizeBand | DAG | Yes | None |
| Self-employed values | AnchorAge, Region, JobFamily, Industry; no Company or JobLevel | DAG | Yes | D2 law |
| Four non-working sets | Each exactly AnchorAge, Education, Region; separate laws | DAG | Yes | D2 laws |
| Occupational history | Absent and never evaluated/invented for Income | SEMANTIC / DAG | Yes | None |
| Income feedback | Forbidden into every Contract-4 fact and its addressing | DAG / NON-INTERFERENCE | Yes | Protocol realization |
| Excluded Layer-1 inputs | All six exclusions and proxies preserved; only age/Region authorized node-specifically | DAG | Yes | None |
| Hidden socioeconomic/source states | None introduced or authorized | DAG | Yes | None |
| New public operations | None | API | Yes | None |
| Probabilities/distributions/caps | No numerical law or empirical cap selected | MODEL / DATA | No | Income MODEL/D2 |
| Source crosswalks/buckets | Explicit deterministic compatible mappings and parent-only transforms required | MODEL / DATA | No assets selected | Income MODEL/D2 |
| Retained assets/resource bounds | No new table mandated; later representation must have finite complete accounting | RESOURCE | Requirement only | Owner introducing Income model assets |
| Streams/draws/encoding | Exact names, Components, sampling and identity bytes deferred | PROTOCOL | No | Deterministic Protocol |
| Identity coverage | All output-affecting Income choices; broad identity and selective generation isolation simultaneously | IDENTITY | Yes | Deterministic Protocol encoding |

### 21. Cross-contract contradiction audit

The verdict classes are NO ISSUE, DEFERRED CLEANLY and REQUIRES CONTRACT FIX. A deferred numerical or protocol realization is not a semantic contradiction. This audit checks compatibility, not whether another Income design could be chosen.

| # | Audit case | Verdict | Reconciliation |
| --- | --- | --- | --- |
| 1 | Income field addition vs Contract-1 public operation boundary | NO ISSUE | Contract 1:458 expressly defers Income completion. One scalar and field add no operation; all seven frozen constructor/retrieval/identity signatures remain unchanged. |
| 2 | Income field placement vs SocioeconomicFacts | NO ISSUE | Income is appended to the existing common facts value; Person, Company and working payloads receive no duplicated Income field. The Contract-1 extension location is preserved. |
| 3 | Mandatory all-branch Income vs ground truth | NO ISSUE | The common personal inflow rate applies to every Person. No branch needs unknown, optional or inapplicable Income; successful facts are complete ground truth. |
| 4 | PrimaryActivity law selection vs six-state structure | NO ISSUE | The frozen six states select exactly one common-outcome law. Secondary eligible income sources do not create another primary state or payload (Contract 1:478). |
| 5 | Income parent sets vs Contract-4 downstream boundary | NO ISSUE | Contract 4:469 leaves Income incoming edges to Contract 5. The 22 conditional information edges append only downstream of the unchanged Contract-4 graph and gates. |
| 6 | AnchorAge use vs CurrentAge exclusion | NO ISSUE | Node-specific AnchorAge is the frozen birth result, never reconstructed from CurrentAge. Employees have no direct age parent; self-employed and non-working laws use only their authorized anchor input. |
| 7 | Region use vs finer geography | NO ISSUE | Only the exact frozen Region projection is authorized (Contract 4:115). Private municipality-selection intermediates add no finer Income dimension or new geography draw. |
| 8 | Employee parents vs JobFamily/JobLevel meanings | NO ISSUE | Function and delegated responsibility are distinct frozen facts, not compensation labels. Conditional Income edges imply no hard attainment, responsibility or pay ordering. |
| 9 | Employee Income vs Employer generation direction | NO ISSUE | Occupation/responsibility generate before Employer; Income follows selection and cannot revise, reject or reweight it. Employer retains its broader Company/count information authorization (Contract 4:259). |
| 10 | Selected Company facts vs Industry relational isolation | NO ISSUE | Employee.CurrentIndustry is exactly Company.Industry[Employer], once, with no copied employee Industry field or unselected-Company Income lookup (Contract 4:311). |
| 11 | Employee Income vs Company Person-independence | NO ISSUE | Reading selected Company payloads downstream introduces no Person or Income parent into Company generation. Company facts remain independent of requesting Person and Income laws. |
| 12 | Employer key-only vs arbitrary employer-specific effect | NO ISSUE | Employer accesses authorized selected values only. Equal Industry/SizeBand payloads are Income-equivalent under fixed other Income parents, own law/model/data and Person innovation; coordinate/ID seeding or an intercept is forbidden. |
| 13 | SizeBand parent vs raw EmployeeCount exclusion | NO ISSUE | The exact count-derived band is the direct value. Raw selected count can derive it but has no residual effect inside a fixed band; upstream count-driven selection remains unchanged. |
| 14 | Self-employed Income vs no Company | NO ISSUE | Personal attribution and the four-parent aggregate law need no Company, Employer, JobLevel or owned-business entity (Contract 3:516). Education remains indirect through function. |
| 15 | Non-working Income vs absent occupational payloads | NO ISSUE | Each separate law uses AnchorAge, Education and Region only. Both working pointers remain nil and no history/Company/occupational law is evaluated (Contract 4:358). |
| 16 | Zero Income vs no missingness | NO ISSUE | Zero means an exact absent eligible current inflow, not an absent measurement or payload. No frozen activity invariant promises positive personal receipts (Contract 3:625). |
| 17 | Common Income vs Company revenue/turnover | NO ISSUE | The aggregate is attributed to the Person. Company revenue and unattributed business turnover are excluded; personal earnings are identified after operating costs and before personal tax/contributions. |
| 18 | Common Income vs household resources | NO ISSUE | Another member’s income, gifts/family support and merely available household resources are excluded. No household membership, size or allocation mechanism is required. |
| 19 | Common Income vs wealth/customer value | NO ISSUE | Wealth, asset values, capital flows excluded in section 3, credit and customer value are not Income. Eligible investment/property income does not require generating those states. |
| 20 | Gross meaning vs no tax engine | NO ISSUE | Gross fixes the stage before personal taxes/contributions; v1 does not calculate net or disposable income, deductions or benefit eligibility. |
| 21 | uint32 signedness vs inflow concept | NO ISSUE | The chosen inflow aggregate is nonnegative. Signed accounting losses are incompatible direct observations, not values to cast, convert by absolute value or clamp. |
| 22 | Whole-EUR currency/unit vs D2 source conversion | DEFERRED CLEANLY | Output EUR/year, nominal anchor basis and checked ceiling conversion are fixed in section 5. Income D2 later owns explicit compatible raw-currency/period reconciliation and crosswalk assets. |
| 23 | Public domain vs Plausible+ and empirical caps | DEFERRED CLEANLY | The full representation domain is structurally valid. Income MODEL/D2 later owns production support, zero mass, dispersion/tails and any evidence-backed model cap without redefining public validity. |
| 24 | Gender exclusion | NO ISSUE | Gender and its proxies are prohibited directly and transitively in semantic inputs, model keys and stochastic addressing. Facts-only acquisition needs no Gender/name/photo work. |
| 25 | ReferenceDate invariance | NO ISSUE | Income remains the current annualized PopulationAnchorDate rate while composite CurrentAge is snapshot-relative (Contract 1:269). ReferenceDate and SnapshotID never condition Income. |
| 26 | Income non-interference vs Contract-4 facts | NO ISSUE | Income/model/data/protocol cannot alter any Contract-4 fact or its realization with its own inputs/models/addressing fixed (Contract 4:738); no upstream reconciliation is permitted. |
| 27 | Broad world identity vs isolated generation | DEFERRED CLEANLY | Complete output-affecting identity coverage and selective fixed-parent realization must coexist. Exact encoding/selective fingerprints remain Deterministic Protocol-owned (Contract 4:753); the full ID is not a universal seed. |
| 28 | Layer-3 downstream use vs feedback prohibition | NO ISSUE | Future Layer-3 contracts can authorize downstream consumption of Income. Their facts, data/models and calibration cannot feed back into Income or other Layer-2 facts. |

No frozen Layer-0/1 or Contract-1–4 contradiction makes Income impossible. No frozen contract must be reopened to implement this semantic extension. No case requires a contract fix. Missing empirical D2 evidence remains a cleanly deferred implementation prerequisite.

### 22. Independent-review questions

These are prompts for Astra’s independent review, not answers or additional normative choices. Review the selected design and its evidence, including its intentional v1 limitations.

1. Is Income one comparable economic quantity across all six PrimaryActivity states?
2. Is gross personal Income distinguished cleanly from salary, Company revenue, business turnover, household resources and wealth?
3. Is the annualized current anchor-rate meaning exact and free of hidden history?
4. Are EUR, whole-EUR unit, uint32 domain and signedness semantically coherent?
5. Is Q(x) = ceil(x) a stable representation semantic rather than premature model policy?
6. Is zero unambiguous and never missingness?
7. Is mandatory Income coherent in every branch?
8. Is the public API minimal?
9. Is PrimaryActivity’s role as branch-law selector exact?
10. Does every branch have a closed exact Income parent set?
11. Are employee Income parents minimal but sufficient for Plausible+?
12. Is omitting direct employee AnchorAge and Education a coherent v1 limitation?
13. Is Employer genuinely key-only rather than a hidden Company-specific pay effect?
14. Are CurrentIndustry and SizeBand the only selected Company Income values?
15. Is raw EmployeeCount correctly excluded within a fixed SizeBand?
16. Are unselected Companies isolated from employee Income?
17. Is self-employed Income truly personal Income rather than turnover/profit/accounting result?
18. Are non-working Income laws coherent without occupational history?
19. Is Gender excluded directly and transitively?
20. Is ReferenceDate fully excluded?
21. Can Income model/data changes leave every Contract-4 fact invariant?
22. Can D2 fit zero mass, skew and heavy tails without adding a semantic parent?
23. Has Contract 5 avoided freezing a probability-family choice?
24. Can D2 map source data into the personal/all-source/annualized/gross concept without semantic substitution?
25. Can the later Deterministic Protocol isolate Income branch/model changes from unrelated facts?
26. Can full SocioeconomicWorldID coverage coexist with fixed-parent realization isolation?
27. Can future Layer 3 consume Income without feedback?
28. Can two independent implementations using identical later D2/protocol contracts produce materially identical Income semantics?

### 23. Contract v0 verdict

Contract 5 selects one comparable personal Income fact across all six activities: a nonnegative gross personal monetary aggregate, annualized under current anchor conditions, in whole nominal EUR, exposed as `AnnualIncomeEUR uint32` through mandatory `SocioeconomicFacts.Income`.

Zero is legitimate no-Income ground truth in every branch. Exact whole-EUR conversion preserves that meaning. Personal earnings remain distinct from business turnover; non-working Income requires no occupational history. PrimaryActivity selects one closed branch law, selected Company access is restricted, and Income cannot perturb any Contract-4 fact.

The semantic and dependency design is complete for independent review. Empirical numerical laws, source assets, concrete model-resource bounds and exact deterministic protocol remain explicitly owned by later contracts. Their absence is not a contradiction or a reason to implement an implicit default.

## Source-backed risks

These are descriptive implementation and review risks supported by the inspected source and frozen contracts. They introduce no new Income meaning, parent or protocol commitment.

- **Facts-only dependency leakage.** Full frozen Person generation obtains Gender, gender-dependent names and PhotoID. Using it merely to obtain AnchorAge or Region would cross the excluded-input boundary; the existing birth and municipality-selection boundaries already provide the required inputs. Evidence: `tools/fakedata/layer1.go:509`, `tools/fakedata/layer1.go:517`, `tools/fakedata/layer1.go:537`, `tools/fakedata/layer1.go:355`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:259`.
- **Region and finer-geography leakage.** Private placeRow contains municipality, StateProv, Region, population and an Istat coordinate. Projection to Region can accidentally retain those finer fields, or full address work can add PostalCode/street. Only the frozen Region projection is authorized where the Income parent set includes it. Evidence: `tools/fakedata/layer1_data.go:87`, `tools/fakedata/layer1.go:86`, `tools/fakedata/layer1.go:265`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:152`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:154`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:160`.
- **Selected-Company leakage.** A convenient whole-Company retrieval exposes more than employee Income's selected CurrentIndustry and SizeBand payloads. ID, raw EmployeeCount, unselected Company information and other metadata can accidentally become lookup dimensions. Employer remains only the relational key. Evidence: Contract 5 section 9, “Selected Company isolation” (authorized payload, whole-Company retrieval and unselected-Company exclusion), `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:322`.
- **Employer coordinate as a random effect.** Entity stream derivation accepts a coordinate and path, so an implementation can accidentally create an employer-specific Income seed or intercept. That would distinguish employers with equal authorized semantic payload despite fixed Person innovation and other parents. Evidence: `tools/fakedata/stream.go:37`, `tools/fakedata/stream.go:63`, Contract 5 section 9, “Selected Company isolation” (Employer-key equivalence and Person-scoped innovation).
- **Raw EmployeeCount leakage.** Deriving SizeBand may require reading the selected count. Passing that raw value onward would add the expressly omitted residual count effect within a fixed band. Evidence: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:281`, Contract 5 section 9, “Selected Company isolation” (count-only SizeBand derivation and fixed-band count invariance).
- **Global identity ripple.** Frozen stream derivation hashes supplied roots and dependency Components. A broad socioeconomic or upstream identity reused as a universal seed can perturb Contract-4 facts after an Income change, or perturb Income despite unchanged law, parents and legitimate addressing. Evidence: `tools/fakedata/stream.go:84`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:405`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:623`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:738`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:763`.
- **Signed/net source mismatch.** Signed accounting/tax aggregates, business losses, salary-only statistics and household amounts are not automatically this gross personal all-source inflow measure. Treating them as direct observations, taking absolute values, clamping losses or substituting zero would conceal a mapping failure. Evidence: Contract 5 section 3, “Nonnegative attribution and losses”, and section 15, source-to-concept mapping and direct-measurement versus calibration distinction.
- **Whole-EUR conversion drift.** Truncation, round-to-nearest, floating point, saturation or unchecked arithmetic can change the frozen ceiling result or its strict zero distinction. Exact integer/rational conversion and pre-conversion maximum checking are already specified, not deferred alternatives. Evidence: Contract 5 section 5, “Whole-EUR quantization and strict zero preservation”, and its checked integer/rational arithmetic and overflow-rejection requirements.
- **Branch-law identity coupling.** Shared physical assets and whole-file checksums can turn a one-branch law/data edit into realization changes in other branches. The existing node/branch isolation requires output-affecting identities to remain selective despite full-world identity coverage. Evidence: `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:625`, Contract 5 section 14, “Model and branch isolation”, and section 17, shared-asset output-affecting identity isolation.
- **Unbudgeted future Income state.** A four-byte returned scalar does not budget retained model tables, source buffers or simultaneous construction storage. The later asset owner still needs finite input, retained and peak accounting before implementation and allocation. Evidence: Contract 5 section 15, “Initialization, validation and errors” (returned scalar payload and future asset-resource accounting), `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md:665`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:664`.

## Deferred decisions

This table makes the existing model/protocol deferrals explicit; it selects no new numerical law or semantic dependency.

| Deferred decision | Exact owner | Binding boundary |
| --- | --- | --- |
| Numerical Income laws and conditional tables/parameters | Income MODEL/D2 | One common frozen Income semantic; exactly the selected branch parent sets and selector behavior |
| Zero probabilities / zero mass | Income MODEL/D2 | Zero remains structurally valid no-Income ground truth in every branch; no mandatory-positive rule |
| Distribution shape | Income MODEL/D2 | No lognormal, gamma, Pareto, empirical CDF or other family is selected here; outcomes retain the frozen meaning and domain |
| Dispersion and tails | Income MODEL/D2 | Versioned numerical choices within authorized branch information and checked valid-domain arithmetic |
| Empirical/model caps | Income MODEL/D2 | Distinct from the unchanged MaxUint32 representational maximum; no implicit public cap or silent repair |
| Source assets and crosswalks | Income D2 | Explicit mapping into the frozen personal, gross, annualized anchor-rate, whole-EUR inflow semantic; incompatible sources are not relabeled as direct measurements |
| Raw source currency and period reconciliation | Income D2 | Preserve the frozen nominal anchor-basis EUR/year output concept, exact conversion and zero meaning; no runtime currency/period choice |
| Parent-only model buckets, pooling and thresholds | Income MODEL/D2 | May coarsen only the selected law's authorized parents; no finer geography, latent or additional dimension |
| Income input/retained/construction resource bounds | contract introducing Income assets | Finite complete source/loading, retained and peak accounting before implementation or potentially excessive allocation; frozen Contract-2 budgets remain intact |
| Income stream paths and Component names | Deterministic Protocol Contract | Realize the frozen parent, branch, selected-payload and non-interference boundaries; no identifiers selected here |
| Sampling algorithm, seed/framing bytes and draw protocol | Deterministic Protocol Contract | Exact deterministic integer/rational realization, fresh random access and fixed-parent invariance; no new sampler or draw sequence selected here |
| Income law routing representation | Deterministic Protocol Contract | Exactly one valid selected-law resolution per reachable condition; no unrelated fallback or inapplicable-law evaluation |
| SocioeconomicWorldID exact encoding | Deterministic Protocol Contract | Complete output-affecting identity coverage with selective generation isolation; no universal identity reseeding |
| Layer-3 use of Income | future Layer-3 contracts | Expressly authorized downstream use only; no feedback into Income or other Layer-2 facts |

None of these deferrals permits later owners to change the Income concept, public type/unit, applicability, zero semantics, branch selection, direct semantic parent sets, selected-Company isolation or excluded inputs without reopening Contract 5. Existing annualization, ceiling quantization, domain and non-interference commitments also remain unchanged.

Missing empirical D2 evidence is not a blocker. Exact numerical models, source assets, enforceable model-resource bounds and deterministic protocol completion remain prerequisites for future implementation rather than implicit defaults.

## Repository-integrity proof

The complete stabilized input was read and its required SHA-256 verified before substantive analysis:

```text
SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v0_STABILIZED.md
SHA-256: c33cdd2d65612bd158f5111f494ba6a85831cfe32bc9148cbf5bffdbb081b1a4
```

The earlier original and review-ready inputs were present and preserved:

| Inspected repository input | Preserved SHA-256 |
| --- | --- |
| SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v0.md | cefb7fc58a59adc54d484ea4bd00130d84132375122cd80fd9400d812ed94f8f |
| SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v0_REVIEW_READY.md | b4a5fa1a1872cd2afb97ae99041dc9851d004595367457d92e3da5a8f5b5a757 |

All three inspected Contract-5 files remained unchanged. These are repository-input checksums, not a checksum of this corrected specification. They are provenance only, not normative self-citation dependencies. Any later replacement under the original root v0 filename is outside this task.

Frozen Contracts 1–4 matched their exact section-1 hashes before and after this task. The applicable root AGENTS.md and relevant frozen semantic/dependency source were inspected. No alternate draft or unrelated modified source supplied normative precedent.

The required offline existing-package sanity command passed:

```bash
GOCACHE=/tmp/krenalis-l2-c5-final-artifact-gocache \
GONOSUMDB='*' \
GOPROXY=off \
  go test ./tools/fakedata
```

Actual result, exit status zero:

```text
ok  github.com/krenalis/krenalis/tools/fakedata  13.288s
```

This validates the existing repository package only. It is not Layer-2 Income implementation evidence or empirical-fit evidence. No Layer-2 implementation source or test was created. No generator, network access or external research was used.

Before analysis, 21 captures were made under /tmp: branch, HEAD, full unfiltered git status --porcelain=v1 -uall, cached binary diff, raw .git/index SHA-256, SHA/status for each of frozen Contracts 1–4, SHA/status for each of the three inspected Contract-5 inputs, and semantic.go SHA/diff. After analysis, complete consistency verification, the package test and cache cleanup, every corresponding before/after capture compared byte-for-byte using cmp, including full worktree status. Both git diff --check and git diff --cached --quiet passed before and after the read-only analysis.

The complete corrected specification was reread after integration. Existing economic, annualization, representation, quantization, applicability, branch, parent, non-interference, D2, protocol and identity text was preserved apart from heading numbers and required internal-reference updates. The decision ledger, all ten risk subjects/descriptions/external citations and the entire deferred-decision substance and ownership were preserved. The existing short audit was replaced by the required complete review table; the other three requested support sections were added. No new semantic parent, numerical distribution, hard ordering or probability was introduced.

Mechanical checks confirmed one unique public AnnualIncomeEUR scalar and one Income field addition, with identical focused/consolidated illustrations and unchanged seven public operations; no IncomeBand; six mandatory and zero-valid Income branches; five employee value parents plus Employer as key only; four self-employed parents; three exact parents for each separate non-working law; 22 conditional information edges, comprising 21 value edges and one relational-key edge; and six PrimaryActivity law selectors. Parent table and edge list remain equivalent. Income feedback, selected-Company isolation and all excluded-input/fixed-parent invariance rules remain unchanged.

The review supports contain all 22 requested correlation relationships, all 26 adversarial cases, all 28 contradiction-audit cases using only the prescribed verdict classes, and all 28 independent-review questions. Internal section references were checked against the new numbering and exact rules. No stale Contract-5 self-line citation or machine-local absolute repository path remains. No actual semantic/API/DAG contradiction was found; no controversial intentional Income choice was redesigned.

Under the user's explicit instruction, the complete corrected text was placed in one new repository-root review artifact. That was the only repository addition. No inspected source or other pre-existing repository file was created, replaced, modified, renamed, formatted or deleted. Final captures preserved every initial capture except full status; removing exactly the one authorized new untracked artifact entry from a temporary status comparison reproduced the complete initial status byte-for-byte using cmp. No pre-existing tracked or untracked entry changed.

The unrelated tools/types/semantic.go modification remained byte-identical, including its diff. Its preserved SHA-256 was:

```text
06a7ff690b3e12cdaf623e3ffb36003d1675a6aec69b3ee59e9ead428cc17842
```

Branch and HEAD remained property-semantics and 76fa0fff109ca6abad62f55c8761275455842dc5. All inspected source hashes/statuses, cached binary diff and raw Git index remained unchanged. Final git diff --check and git diff --cached --quiet passed. Nothing was staged or committed, and no Layer-2 implementation was performed. All task-created /tmp analysis and verification artifacts, including the Go cache, were removed.

L2 DETAILED CONTRACT 5 v0 COMPLETE — READY FOR INDEPENDENT REVIEW
