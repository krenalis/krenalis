# Synthetic World v1 — Layer 2 Detailed Contract 4 v0

## Socioeconomic DAG

### 1. Status and frozen baseline

**Status:** proposed normative Contract 4 v0, corrected and pending independent Astra review. This document selects one socioeconomic semantic dependency graph. It is not freeze approval, empirical-fit evidence, or a Layer-2 implementation.

**Requested model/configuration:** GPT-5.6 Sol xhigh. This records the requested execution configuration; it does not independently attest to the runtime model selected by Codex.

**SOURCE FACT:** direct repository inspection verified:

```text
Branch:  property-semantics
HEAD:    76fa0fff109ca6abad62f55c8761275455842dc5
Subject: Add Synthetic World Layer 1
```

The authoritative Contract-4 source was read completely and its SHA-256 verified before rectification and again at completion:

```text
SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v0.md
SHA-256: 66eb1be43c1a03d02a389b6308c1312410515cef14ac1e2fb4a970ae9fe622ed
```

The following exact frozen files had their SHA-256 values verified before rectification and again at completion:

| Authoritative frozen input | Verified SHA-256 |
| --- | --- |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md` | `bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md` | `e9af32a0bb9b48bf8972a408a1369901a3f3a9fe5ecd76d5762a367c055bc098` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md` | `df3ef0b52f37b587b8449cc71085dc9e5aca00f754bbd93e123c6fd5e3a4cb55` |

Hereafter **Contract 1**, **Contract 2**, and **Contract 3** mean precisely these files. Their authoritative status comes from the supplied frozen-input instruction and verified hashes. Historical proposal or review-status wording inside those files does not substitute a draft for the supplied frozen input.

**NORMATIVE:** frozen Layer 0/1 and Contracts 1–3 remain unchanged. The pre-existing modification to `tools/types/semantic.go` supplies no normative precedent.

The specification text is the output of the read-only drafting/rectification task. Any subsequent placement or naming of that text as a repository-root review artifact is outside this task and does not alter the normative Contract-4 content. No Layer-2 source, data, fixtures, or tests are implemented.

### 2. Scope / non-scope

**NORMATIVE — DAG:** Contract 4 fixes:

- Semantic nodes, exact direct-parent sets, edge direction, and occupational branch gates.
- Company and Person generation separation.
- Deterministic SizeBand and relational employee Industry derivations.
- Employer selection's complete Person-side condition and permitted Company-side information.
- Forbidden dependencies, hidden-state policy, and semantic non-interference.
- Random-access and immutable-state architectural requirements.
- Downstream-only Income and Layer-3 boundaries.

**DEFERRED:** this contract does not fix category probabilities, empirical assets, fitted distributions, source crosswalks, exact CompanyCount, EmployerWeight formulas, particular employer subsets, index count, condition-to-index routing, or routing representation.

It does not define stream names, Component identifiers, seed or framing bytes, fingerprint decomposition, exact RNG sequences beyond already-frozen Contract-2 requirements, or SocioeconomicWorldID encoding.

There is one final DAG. Its parent table and edge list are authoritative together; no additional semantic dependency is permitted by an example, implementation convenience, or later empirical table.

### 3. Frozen constraints

| SOURCE FACT | Exact repository evidence | Binding consequence |
| --- | --- | --- |
| Layer 2 extends the same package without changing frozen Layer-1 APIs or goldens. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:28` | No new Layer-1 public API is needed or authorized. |
| Facts-only access obtains AnchorAge without full Person or photo generation. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:259` | Reuse the birth boundary after operational-index validation. |
| Socioeconomic facts describe `PopulationAnchorDate = 2026-01-01`. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:271`, `tools/fakedata/layer1.go:19` | No socioeconomic aging with ReferenceDate. |
| Gender and all listed snapshot/photo inputs are excluded directly and transitively. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:612` | Exclusions also cover Company generation, selection, proxies, and stable identity. |
| CompanyCount is one later-prescribed literal, private and Person-independent. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:160` | A fixed dense domain, not a population-dependent count. |
| EmployeeCount is modeled anchor workforce, a `uint32`, with zero valid. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:248` | It never measures realized assignments. |
| Zero EmployeeCount requires zero weight in every employee-employer index. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:297` | Generated employers have positive modeled workforce. |
| Selection uses canonical cumulative indexes and one `Uint64n(total)` invocation. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:442`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:470` | Contract 4 does not replace this selector. |
| Index and resource maxima are frozen separately from empirical values. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:516` | At most 1,024 indexes; all aggregate budgets still apply. |
| Education is highest completed attainment and applies in every activity. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:149`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:159` | Current study is not completed attainment. |
| SizeBand has one exact count-derived result, without RNG. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:280`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:298` | No sampled SizeBand variable. |
| JobLevel means delegated organizational responsibility, not salary, tenure, age, or count. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:417` | Correlation never redefines responsibility. |
| Employee Industry is resolved through Employer; it has no EmployeeFacts field. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:503` | One authoritative Company Industry. |
| Self-employment has Industry and JobFamily, without Employer, Company, or JobLevel. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:516`, `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:522` | A separate occupational branch without a business entity. |
| All four non-working branches have no current occupational facts. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:545` | Structural absence, not placeholder sampling. |
| Canonical combinations have no new hard correlation-validity matrix. | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:611` | The DAG authorizes model tendencies, not new taxonomy invalidity. |

**NORMATIVE:** the exclusions are exactly:

```text
Gender
CurrentAge
ReferenceDate
Layer1 SnapshotID
FaceCatalog
PhotoID
```

They apply to every Contract-4 generation node and to socioeconomic stable identity. Derived proxies cannot circumvent them. A composite may contain excluded Layer-1 observations, but they cannot enter the socioeconomic generation subgraph.

### 4. Terminology

**NORMATIVE:**

- **Semantic parent:** a fact whose value is permitted to affect a child's conditional distribution or deterministic semantic result.
- **Direct parent:** a permitted immediate information input, not merely an ancestor whose value is correlated with another parent.
- **Addressing input:** protocol information locating a deterministic result or random innovation. WorldSeed, PersonIndex, CompanyIndex, and future stream domains are not semantic parents merely because sampling uses them.
- **Node model/data:** the versioned conditional law or deterministic semantic mapping owned by that node. Its tables and thresholds implement the law; they are not additional socioeconomic facts.
- **Branch gate:** a discriminator controlling whether a node exists and is evaluated. It is distinct from an additional dimension of the node's conditional model.
- **Semantic descendant:** a node reachable through direct semantic edges or branch-gating relationships. Gating counts for non-interference and applicability changes.
- **Conditional innovation:** protocol sampling machinery for one model node. It does not authorize an extra shared latent fact.
- **OperationalCompanyUniverse:** the immutable dense domain `1..CompanyCount` already frozen by Contract 2. It is an explicit fixed Layer-2 model-domain input, not a new Layer-1 fact, public field, or sampled latent.

`Education` denotes `SocioeconomicFacts.Education`, whose type is EducationLevel. `Employer` denotes `EmployeeFacts.Employer`. `Employee.JobFamily`, `Employee.JobLevel`, `SelfEmployed.JobFamily`, and `SelfEmployed.Industry` denote their frozen branch fields.

Company nodes are templates instantiated once per operational CompanyIndex `j`. `Company.X[*]` means the finite keyed family of `X` values over OperationalCompanyUniverse. It introduces no aggregate statistic or additional semantic variable. A same-Company edge is instantiated only for matching `j`.

`Company.Industry[Employer]` means the Industry payload of exactly the operational Company selected by Employer. It is a selected relational lookup payload, not a stochastic family-wide parent, new public field, or sampled node. Employer supplies the key; only the selected Company's Industry supplies the value. This selected-payload notation does not narrow the family-wide Company information authorized for Employer selection.

`Employee.CurrentIndustry` is notation for a relationally available fact, not a new Person field. Future Income and future Layer-3 facts are boundary placeholders; their model nodes and incoming edges are not defined by this contract.

A parent set is a closed authorization set. A later model may pool categories or give a permitted dimension no numerical effect in some conditions. It may not add information outside that set or treat all ancestors as direct inputs.

### 5. Semantic-input boundary

**NORMATIVE:** the complete Layer-1 semantic input set authorized for Layer-2 generation is:

```text
{AnchorAge, Region}
```

Only the node-specific uses in section 18 are permitted. Authorization of Region does not authorize it as a direct parent of every Person node.

BirthDate, birth year, names, contact values, street, Municipality, Province/StateProv, PostalCode, canonical Place coordinate, Place population, and any other Layer-1 observation are not additional Layer-2 semantic inputs.

Private access, protocol addressing, operational validation, and binding the actual stable Layer-1 WorldID remain available under frozen contracts. None grants permission to use an otherwise unauthorized observation as a model dimension.

**NORMATIVE:** addressing may locate a random-access Person or Company and select its isolated innovation. It must not introduce an undeclared socioeconomic dependency, encode excluded inputs, or serve as a hidden socioeconomic classifier. Exact permitted addressing and fingerprint composition remain protocol-owned.

Changing WorldSeed or a requested coordinate can change generated facts; these are changes of deterministic realization or entity, not additions to the semantic-parent table.

### 6. Approved Layer-1 inputs and geography decision

**DAG — NORMATIVE:** AnchorAge is the integer age returned by frozen `World.birth` at PopulationAnchorDate. It is not derived from CurrentAge. The frozen representation has 55 ages, 21–75.

**SOURCE FACT:** age selection and the separate return value are at `tools/fakedata/layer1.go:355`, `tools/fakedata/layer1.go:365`, and `tools/fakedata/layer1.go:381`. The 55 consecutive ages are enforced at `tools/fakedata/layer1_data.go:135` and `tools/fakedata/layer1_data.go:146`.

**DAG — NORMATIVE:** the sole approved geography abstraction is **Region**, defined exactly as:

```text
Region(i) = placeRow.region for the canonical municipality selected
            by frozen Layer-1 address municipality selection for Person i
```

This means the exact stored Region label from the selected row of frozen `data/it-2026a/places.csv`, column `region`. It is the administrative Region associated with the represented address municipality. It is not birthplace, Company location, or a time-varying migration fact.

**SOURCE FACT:** public Address exposes Municipality, StateProv, PostalCode and Country, but no Region, at `tools/fakedata/layer1.go:85`. Private `placeRow` already stores `istatCode`, `municipality`, `stateProv`, `region`, and `population`, at `tools/fakedata/layer1_data.go:87`. The CSV header and parser preserve the Region column at `tools/fakedata/layer1_data.go:253` and `tools/fakedata/layer1_data.go:396`.

The exact municipality is obtainable by reproducing only the frozen address municipality-selection prefix: its mixture decision followed by `World.municipality` with that decision. The relevant source is `tools/fakedata/layer1.go:267`, `tools/fakedata/layer1.go:271`, `tools/fakedata/layer1.go:275`, and `tools/fakedata/layer1.go:465`. Frozen tests independently recover the same municipality at `tools/fakedata/layer1_geography_test.go:38`.

**NORMATIVE:** Layer 2 must use the municipality chosen by those frozen semantics. It must not draw a second socioeconomic municipality, use only one of the frozen mixture's paths, or match a municipality by a potentially ambiguous display name. It needs no postal, street, name, Gender, CurrentAge, or photo work to obtain Region.

Reading the selected private row to project its Region does not authorize finer row fields as socioeconomic inputs. Geography model logic receives only Region. Temporary municipality-selection intermediates are frozen Layer-1 input-acquisition mechanics and cannot influence the downstream law except through that Region.

**SOURCE FACT:** address municipality selection has no ReferenceDate or photo input, and the base stream root is established from model version and seed at `tools/fakedata/layer1.go:211`. ReferenceDate-independent roots are tested at `tools/fakedata/layer1_test.go:487`. Region is consequently anchor-stable under the frozen World semantics.

**RATIONALE:** Region supports broad attainment, activity, and occupational-composition differences without authorizing finer residential detail. Province, Municipality, canonical Place coordinate and PostalCode are unnecessary for this v1 graph. There is no concrete v1 socioeconomic need for PostalCode.

Later D2 may coarsen Region into macro-regions or pool Region categories. It may not distinguish Persons within the same Region using finer geography, municipality population, postal patterns, or a Place identifier. No new Layer-1 public API is added.

### 7. Node classification

**NORMATIVE:** each active semantic node has exactly one class in section 17.

- AnchorAge and Region are BASE INPUT nodes imported across the Layer-1 boundary.
- OperationalCompanyUniverse is a BASE INPUT fixed model-domain node imported from Contract 2.
- Company Industry and EmployeeCount are COMPANY MODEL FACT nodes.
- Education and applicable occupational model facts are PERSON MODEL FACT nodes.
- PrimaryActivity is classified as BRANCH GATE. Its value is nevertheless generated by a categorical model with its own exact parent set.
- Employer is RELATIONAL SELECTION.
- Company SizeBand is DETERMINISTIC DERIVED FACT.
- Employee current Industry is RELATIONAL DERIVED FACT.
- Future Income and future Layer-3 facts are FUTURE DOWNSTREAM FACT boundary placeholders.

Company.ID is classified as an auxiliary DETERMINISTIC DERIVED FACT under the frozen identity protocol. It is outside the stochastic socioeconomic graph and has no semantic generation edges here. Namespace and CompanyIndex are its protocol identity inputs, not socioeconomic parents. Its frozen derivation is documented at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:318` and `tools/fakedata/identity.go:111`.

No hidden Company or Person socioeconomic node is introduced. Model lookup keys, immutable index representations, and routing transforms are implementation of existing nodes, subject to sections 15 and 26.

### 8. Company DAG

**DAG — NORMATIVE:**

```text
Company.Industry has no direct semantic parents.
Company.EmployeeCount has exactly {Company.Industry} as direct semantic parents.
Company.EmployeeCount -> Company.SizeBand
```

These are same-Company relationships. Company Industry is generated first, then EmployeeCount conditional on that Industry, then SizeBand through the frozen mapping.

**RATIONALE:** a conditional workforce model can express different Company-size profiles across economic activities while keeping Industry stable under EmployeeCount-model changes. It needs neither a joint hidden state nor a realized employee population. This is a generation factorization; it does not assert a complete causal theory of real enterprise formation.

Industry and EmployeeCount distributions remain Company/D2-owned. Industry-conditioned EmployeeCount may include zero, and every zero-count Company still has valid Industry and SizeBand.

**DERIVED — NORMATIVE:** preserve Contract 3 exactly:

| EmployeeCount | Company.SizeBand |
| --- | --- |
| `0` | `zero` |
| `1..9` | `micro` |
| `10..49` | `small` |
| `50..249` | `medium` |
| `250..math.MaxUint32` | `large` |

The exhaustive mapping is frozen at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:260`. It consumes no independent RNG.

No Company fact depends on AnchorAge, Person Region, generated Person facts, employer assignments, realized occupancy, another generated Company fact outside its listed same-Company parents, ReferenceDate, Income, or Layer 3. Model generation is random-access per Company coordinate; construction may evaluate it to establish immutable metadata and selection state.

### 9. Education and PrimaryActivity DAG

**DAG — NORMATIVE:**

```text
Education direct semantic parents = {AnchorAge, Region}
PrimaryActivity direct semantic parents = {AnchorAge, Education, Region}
```

Education is generated independently of occupational branches. Its model returns highest completed attainment at the anchor. It cannot inspect activity, occupation, Employer, Company, Income, or future Layer-3 state.

**RATIONALE:** AnchorAge permits attained-education cohort differences; Region permits broad attainment-composition differences. Neither requires enrollment or history.

PrimaryActivity's direct Education input is deliberate. At fixed AnchorAge and Region, completed attainment can affect the model's balance of working, studying, unemployment and inactivity. Shared age and Region alone would not permit that conditional distinction. Direct Region also permits activity differences at fixed age and attainment, which attainment mediation alone cannot express.

PrimaryActivity is generated as one categorical state over the six frozen values. The model does not generate hidden overlapping activity flags and resolve them by inspecting downstream occupational facts.

**NORMATIVE:** overlapping real-world or empirical roles are represented by one primary state. Later MODEL/D2 owns numerical allocation and source-overlap treatment, using only AnchorAge, Education and Region. No primary-state precedence rule, empirical probability, or new represented secondary activity is frozen here. This closes the DAG's overlap responsibility without adding a hidden role-state node or a downstream dependency; frozen secondary-activity semantics are at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:476` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:573`.

### 10. Employee DAG

**DAG — NORMATIVE:** employee generation proceeds in this direction:

```text
Employee.JobFamily -> Employee.JobLevel
Employee.JobFamily -> Employer
Employee.JobLevel -> Employer
Employer -> Employee.CurrentIndustry
```

The exact occupational parent sets are:

```text
Employee.JobFamily direct semantic parents = {Education, Region}
Employee.JobLevel direct semantic parents = {AnchorAge, Education, Employee.JobFamily}
```

**RATIONALE:** Education permits different occupational-function mixtures; Region permits occupational composition beyond attainment differences, including production, service and knowledge-work mixes. Direct AnchorAge is omitted from JobFamily: age-related composition can arise through Education and selection into the employee branch. Residual age effects on family at fixed Education and Region are not an additional v1 model dimension.

JobLevel's direct AnchorAge permits a soft age-related responsibility distribution without defining tenure. Direct Education permits responsibility differences within the same broad JobFamily and age; JobFamily permits different functional responsibility profiles. Broad categories do not encode qualifications so none becomes a hard educational prerequisite.

JobFamily and JobLevel are generated before Employer and do not inspect employer Industry, SizeBand, EmployeeCount or identity. Employer cannot cause either occupational value to be revised, resampled, rejected for a new taxonomy-validity rule, or regenerated.

**NORMATIVE:** JobLevel continues to mean delegated organizational responsibility/scope. It is not salary, tenure, a deterministic age label, or a deterministic Company-size label. Older or highly educated individual contributors and low-education executives remain structurally representable under frozen Contract 3.

Company Industry×JobFamily and SizeBand×JobLevel relationships are expressed by downstream employer-selection semantics, not reverse occupational edges. At fixed JobFamily and JobLevel, additional Company effects on generation of those already-selected occupational facts are not modeled in v1.

PrimaryActivity is the employee branch gate, not an additional conditional model dimension for these nodes.

### 11. Employer-selection semantic contract

**RELATIONAL / DAG — NORMATIVE:** freeze the complete Person-side employee-assignment condition as the semantic tuple:

```text
(Employee.JobFamily, Employee.JobLevel)
```

This notation fixes its information content, not its Go representation, serialization, lookup key, or routing structure.

```text
Employer Person-side direct semantic parents =
    {Employee.JobFamily, Employee.JobLevel}
```

Education, AnchorAge and Region may influence Employer only through permitted paths ending in JobFamily or JobLevel. They may not be read directly by routing, index criteria, EmployerWeight derivation, auxiliary employer randomness, or tie-breaking.

PrimaryActivity only establishes that employer selection applies. It contributes no additional assignment-condition dimension inside the employee branch.

**RATIONALE:** JobFamily carries the occupational compatibility information needed for Industry composition; JobLevel carries responsibility information needed for Company-size composition. Directly conditioning Employer again on attainment, age, or residential Region would add residual dependencies and model-change ripple beyond this minimal v1 selection condition. Company geography is not a frozen fact, so no Company-location or commuting model is authorized.

**NORMATIVE:** the exact Company-side socioeconomic information available to selection subsets and EmployerWeight derivation is:

```text
{Company.EmployeeCount, Company.Industry, Company.SizeBand}
```

The immutable OperationalCompanyUniverse is also a direct domain input to Employer. CompanyIndex identifies candidates and provides frozen canonical numeric ordering; it is not an additional socioeconomic classifier. Company.ID is not an authorized selection or weighting dimension.

EmployeeCount permits the mandatory zero-workforce exclusion and later modeled workforce-sensitive weighting. Industry permits functional matching. SizeBand permits responsibility-sensitive Company-size composition. SizeBand adds no independent information beyond EmployeeCount and must be precisely its frozen deterministic transform. Authorizing both explicitly does not create a new size latent or permit another band definition.

Thus the complete direct-parent set, including relational domain and Company information, is:

```text
Employer direct semantic parents = {
    Employee.JobFamily,
    Employee.JobLevel,
    OperationalCompanyUniverse,
    Company.EmployeeCount[*],
    Company.Industry[*],
    Company.SizeBand[*]
}
```

No future Company field is implicitly authorized. No aggregate realized occupancy, Company revenue, Company geography, owned-business fact, generated coworker, employee roster, or assignment history is available.

Immutable subsets, per-index weights, and routing may use deterministic transforms of only this authorized information and their own later-versioned model/data. Such transforms must add no Person or Company semantic input. EmployerWeight remains the private interval-weight abstraction of Contract 2, not a public Company fact or hidden socioeconomic state.

**PROTOCOL — NORMATIVE:** every reachable employee condition must resolve deterministically to exactly one existing positive-total Contract-2 index. Index sharing is permitted. Missing or invalid reachable-condition coverage fails construction validation; no unrelated runtime fallback is permitted. These requirements are frozen at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:844`.

**DEFERRED:** Company/D2 owns actual index count, Company subsets, exact integer EmployerWeight formulas, empirical values and routing semantics consistent with this condition. The later routing owner fixes representation and any retained-state resource accounting. No one-index-per-condition rule is imposed.

### 12. Employee Industry derivation

**RELATIONAL DERIVED — NORMATIVE:** preserve exactly:

```text
Employer -> Employee.CurrentIndustry
Company.Industry[Employer] -> Employee.CurrentIndustry

Employee.CurrentIndustry = Company(Employee.Employer).Industry

Given Employer = j:
Employee.CurrentIndustry = Company.Industry[j]
```

Resolution uses the socioeconomic World that generated Employer. The exact direct relational parents are `{Employer, Company.Industry[Employer]}`. Employer supplies the key, and the Industry payload of exactly that selected operational Company supplies the value. `Company.Industry[Employer]` is a selected relational lookup payload, not the complete Company Industry family or another Person-side selection condition. Derivation must not read an unselected Company's Industry.

**NORMATIVE:**

- Holding Employer == j fixed, changing Company.Industry[k], k != j, MUST NOT directly change Employee.CurrentIndustry.
- Holding Employer == j fixed, changing Company.Industry[j] MAY change Employee.CurrentIndustry; the result must always equal the selected payload.
- Changing Employer MAY change Employee.CurrentIndustry; the result resolves the newly selected Company's Industry.

Employer selection separately retains its authorized family-wide `Company.Industry[*]` input. Changing an unselected Company's Industry may change selection semantics and, if Employer changes, may indirectly change current Industry. It does not become a direct parent of the fixed-employer derivation.

The derivation consumes no independent RNG, creates no copied `EmployeeFacts.Industry` field or other separate Person field, and never feeds back into Company Industry. Different employers may share Industry, so a changed Employer does not require a different Industry value.

The frozen authoritative location is `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:503` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:505`.

### 13. Self-employed DAG

**DAG — NORMATIVE:**

```text
SelfEmployed.JobFamily direct semantic parents = {Education, Region}
SelfEmployed.Industry direct semantic parents = {SelfEmployed.JobFamily}
```

Self-employed JobFamily is generated first. Industry is then generated conditional on that functional family, permitting Industry×JobFamily correlation without a Company relationship or shared hidden business state.

**RATIONALE:** the common occupational vocabulary supports the same broad education and regional composition dimensions, while the Industry conditional distinguishes the economic activity in which that self-employed function is performed. Direct age is omitted; age relationships can arise through Education and self-employed branch selection. Additional Region or Education effects on Industry at fixed self-employed JobFamily are not modeled in v1.

The employee and self-employed JobFamily models have separate model/data ownership and need not share probabilities merely because their taxonomy and parent dimensions match. Changing one branch's occupational model must not perturb the other branch's model realization.

Self-employed generation has no Employer, Company, CompanyIndex, Company.ID, Company SizeBand, EmployeeCount, JobLevel, or owned-business entity. PrimaryActivity gates the branch without adding a model dimension.

Industry concerns current primary economic activity and JobFamily concerns the work performed, preserving `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:531`. The chosen generation factorization does not redefine either taxonomy or create a hard Industry×JobFamily validity matrix.

### 14. Branch gating

**BRANCH GATE — NORMATIVE:** Education and PrimaryActivity are always evaluated for a valid requested Person. After PrimaryActivity is selected, evaluate only the applicable occupational branch:

| PrimaryActivity | Occupational evaluation | Structural result |
| --- | --- | --- |
| `employee` | Employee.JobFamily, Employee.JobLevel, Employer; current Industry available by relational derivation | Employee present; SelfEmployed absent |
| `self-employed` | SelfEmployed.JobFamily, SelfEmployed.Industry | SelfEmployed present; Employee absent |
| `unemployed` | None | Both working payloads absent |
| `student` | None | Both working payloads absent |
| `retired` | None | Both working payloads absent |
| `inactive` | None | Both working payloads absent |

Do not sample, score, route, or derive a current occupational value for an inapplicable branch and discard it afterward. Do not invent former occupations or employers for non-working states. This is a semantic evaluation requirement, beyond merely returning nil payloads.

An immutable constructor may load both branches' model assets and build all required indexes. Loading model state is not evaluating an inapplicable Person's occupational facts.

PrimaryActivity's gate relationships are listed separately in section 19. They count toward descendant and applicability reasoning but are not occupational probability-table dimensions. Exact stream-path realization remains protocol-owned.

### 15. Hidden-state policy

**DAG — NORMATIVE:** v1 contains no hidden socioeconomic latent variable. No hidden state may influence public Layer-2 facts unless its semantic meaning and complete edges are explicitly frozen by Contract 4. This contract freezes no such latent node.

Later D2 cannot introduce a socioeconomic class, status score, career tier, affluence latent, Company latent, household latent, overlapping-role state, or other sampled common cause to couple public outputs.

Likewise it cannot reuse one hidden random draw across model nodes to create an undeclared common cause. Protocol random innovations belong to existing nodes; a shared immutable stream root is addressing machinery and must preserve the graph's isolation.

Pure deterministic helper transforms are permitted when they use only the child's authorized parents and model/data, add no information, and introduce no new edge. Examples are a Region macro-group, an AnchorAge bucket, an Education lookup key, or an employer routing key derived solely from JobFamily and JobLevel.

Output-sensitive thresholds and lookup mappings are versioned MODEL/DATA semantics under their later owner. They cannot authorize a finer input, alter frozen public SizeBand, or pass unauthorized ancestor information into a child. No helper extraction or concrete representation is required by this policy.

### 16. Correlation mechanisms

**NORMATIVE:** correlation alone is insufficient reason for an additional edge. The selected graph's mechanisms are:

| Relationship | Mechanism | Exact path / explanation |
| --- | --- | --- |
| AnchorAge × Education | DIRECT EDGE | AnchorAge -> Education |
| AnchorAge × PrimaryActivity | DIRECT EDGE | AnchorAge -> PrimaryActivity; also mediated through Education |
| Education × PrimaryActivity | DIRECT EDGE | Education -> PrimaryActivity permits attainment effects at fixed age and Region |
| geography × Education | DIRECT EDGE | The sole geography input is Region: Region -> Education |
| geography × PrimaryActivity | DIRECT EDGE | Region -> PrimaryActivity permits residual regional activity differences |
| Industry × EmployeeCount | DIRECT EDGE | Company.Industry[j] -> Company.EmployeeCount[j] |
| Company Industry × employee JobFamily | EMPLOYER-SELECTION EFFECT | Employee.JobFamily -> Employer and Company.Industry[*] -> Employer; Company.Industry[Employer] supplies employee current Industry |
| Company SizeBand × employee JobLevel | EMPLOYER-SELECTION EFFECT | Employee.JobLevel -> Employer and Company.SizeBand[*] -> Employer; responsibility changes the selected-size composition |
| Employee JobFamily × JobLevel | DIRECT EDGE | Employee.JobFamily -> Employee.JobLevel |
| self-employed Industry × JobFamily | DIRECT EDGE | SelfEmployed.JobFamily -> SelfEmployed.Industry |
| Person socioeconomic facts × Employer | DIRECT EDGE | JobFamily and JobLevel are direct Person parents; Education, AnchorAge and Region act only through authorized paths to them; activity gates existence |

Additional relevant mechanisms clarify deliberate omissions:

| Relationship | Mechanism | Exact path / explanation |
| --- | --- | --- |
| Company EmployeeCount × SizeBand | DETERMINISTIC DERIVATION | Company.EmployeeCount -> Company.SizeBand through the exact frozen mapping |
| Education × employee JobLevel | DIRECT EDGE | Education -> Employee.JobLevel, including distinctions within a broad family |
| Employee and self-employed family composition across the population | SHARED CAUSE | Their branch-specific models use Education and Region; no Person evaluates both current branches |
| Residential Region × employer location / commuting | NOT MODELED IN V1 | No Company geography is frozen or authorized |
| Education × Employer at fixed JobFamily and JobLevel | NOT MODELED IN V1 | Education is not an additional employer condition |
| AnchorAge × Employer at fixed JobFamily and JobLevel | NOT MODELED IN V1 | AnchorAge is not an additional employer condition |
| Region × Employer at fixed JobFamily and JobLevel | NOT MODELED IN V1 | Region is not an additional employer condition |
| Self-employed Region × Industry at fixed JobFamily | NOT MODELED IN V1 | Region influences self-employed Industry only through JobFamily |

No numerical strength, sign, or empirical success is asserted by any mechanism label. Upstream age effects on occupational-family composition remain mediated through Education and branch selection; they introduce no direct age-family edge.

### 17. Authoritative node table

**NORMATIVE:** this table is complete for the current graph and explicitly identified future boundaries.

| Node | Class | Branch | Generated/derived |
| --- | --- | --- | --- |
| AnchorAge | BASE INPUT | Always available | Frozen Layer-1 input; integer age at anchor |
| Region | BASE INPUT | Always available | Frozen Layer-1 selected municipality's exact private Region label |
| OperationalCompanyUniverse | BASE INPUT | Company world; employee selection | Fixed dense model-domain input `1..CompanyCount` |
| Company.ID[j] | DETERMINISTIC DERIVED FACT | Every operational Company | Auxiliary protocol identity; outside stochastic socioeconomic graph |
| Company.Industry[j] | COMPANY MODEL FACT | Every operational Company | Model-generated |
| Company.EmployeeCount[j] | COMPANY MODEL FACT | Every operational Company | Model-generated conditional on same-Company Industry |
| Company.SizeBand[j] | DETERMINISTIC DERIVED FACT | Every operational Company | Exact frozen transform of EmployeeCount; no independent RNG |
| Education | PERSON MODEL FACT | All six activities | Model-generated completed attainment |
| PrimaryActivity | BRANCH GATE | All Persons | Model-generated categorical discriminator; gates occupation |
| Employee.JobFamily | PERSON MODEL FACT | Employee only | Model-generated |
| Employee.JobLevel | PERSON MODEL FACT | Employee only | Model-generated organizational responsibility |
| Employer | RELATIONAL SELECTION | Employee only | Selected operational eligible CompanyIndex |
| Employee.CurrentIndustry | RELATIONAL DERIVED FACT | Employee only | Company.Industry[Employer], the selected relational lookup payload; no separate Person field or RNG |
| SelfEmployed.JobFamily | PERSON MODEL FACT | Self-employed only | Model-generated |
| SelfEmployed.Industry | PERSON MODEL FACT | Self-employed only | Model-generated conditional on self-employed JobFamily |
| Future Income | FUTURE DOWNSTREAM FACT | Applicability deferred to Contract 5 | Boundary placeholder; not a current Contract-4 model node |
| Future Layer-3 facts | FUTURE DOWNSTREAM FACT | Applicability deferred to Layer 3 | Boundary placeholder; not current Contract-4 model nodes |

The keyed Company families used by Employer selection and the selected Company.Industry[Employer] payload used by current-Industry lookup refer to these same existing Company nodes, not separately sampled copies. No material current semantic node is implicit.

### 18. Authoritative parent-set table

**NORMATIVE:** sets are closed and exact. `∅` means no direct semantic parent. Protocol addressing, identity encoding, and a node's own model/data are deliberately not listed as semantic facts.

| Node | Branch | Exact direct semantic parents | Notes |
| --- | --- | --- | --- |
| AnchorAge | Base | ∅ within Layer 2 | Frozen Layer-1 acquisition boundary; no CurrentAge reconstruction |
| Region | Base | ∅ within Layer 2 | Projection of frozen municipality selection; finer geography not a Layer-2 parent |
| OperationalCompanyUniverse | Company world | ∅ | Prescribed Contract-2 domain; later CompanyCount value is model-owned |
| Company.ID[j] | Company | ∅ socioeconomic parents | Auxiliary protocol derivation from namespace and CompanyIndex; no stochastic DAG edges |
| Company.Industry[j] | Company | ∅ | No Person or Company-size parent |
| Company.EmployeeCount[j] | Company | {Company.Industry[j]} | Same Company only; no occupancy |
| Company.SizeBand[j] | Company | {Company.EmployeeCount[j]} | Exact Contract-3 deterministic mapping |
| Education | Always | {AnchorAge, Region} | Completed attainment, not enrollment |
| PrimaryActivity | Always | {AnchorAge, Education, Region} | Generated model value; also branch gate |
| Employee.JobFamily | Employee | {Education, Region} | PrimaryActivity gate listed separately |
| Employee.JobLevel | Employee | {AnchorAge, Education, Employee.JobFamily} | No direct geography or Company input |
| Employer | Employee | {Employee.JobFamily, Employee.JobLevel, OperationalCompanyUniverse, Company.EmployeeCount[*], Company.Industry[*], Company.SizeBand[*]} | Person condition contains exactly JobFamily and JobLevel |
| Employee.CurrentIndustry | Employee | {Employer, Company.Industry[Employer]} | Employer supplies the key; only the selected Company's Industry supplies the value; no unselected Industry parent or independent RNG |
| SelfEmployed.JobFamily | Self-employed | {Education, Region} | Separate branch model/data from employee JobFamily |
| SelfEmployed.Industry | Self-employed | {SelfEmployed.JobFamily} | No Company or additional direct base input |
| Future Income | Future | No incoming edges assigned by Contract 4; exact parents expressly owned by Contract 5 | This is a boundary deferral, not an unconditional Income model |
| Future Layer-3 facts | Future | No incoming edges assigned by Contract 4; exact parents expressly owned by future Layer-3 contracts | Only downstream attachment is authorized |

Operational membership validates whether Company[j] exists. It does not make CompanyCount a conditional Industry or EmployeeCount dimension. With Company model/data and addressing held fixed, changing only the domain does not change Company facts at coordinates operational under both versions.

No extra parent may be introduced through aggregate metadata, model lookup keys, routing, helper transforms, or a full-world fingerprint. Future-node rows do not defer any current node's parent decision.

### 19. Authoritative edge list

**DAG — NORMATIVE:** exactly one line per direct semantic edge, using Company templates as defined in section 4. The two same-Company edges instantiate independently for each `j`; `[*]` edges permit the finite family as Employer-selection input. `[Employer]` permits only the selected relational lookup payload for employee current Industry.

```text
Company.Industry[j] -> Company.EmployeeCount[j]
Company.EmployeeCount[j] -> Company.SizeBand[j]
AnchorAge -> Education
Region -> Education
AnchorAge -> PrimaryActivity
Education -> PrimaryActivity
Region -> PrimaryActivity
Education -> Employee.JobFamily
Region -> Employee.JobFamily
AnchorAge -> Employee.JobLevel
Education -> Employee.JobLevel
Employee.JobFamily -> Employee.JobLevel
Employee.JobFamily -> Employer
Employee.JobLevel -> Employer
OperationalCompanyUniverse -> Employer
Company.EmployeeCount[*] -> Employer
Company.Industry[*] -> Employer
Company.SizeBand[*] -> Employer
Employer -> Employee.CurrentIndustry
Company.Industry[Employer] -> Employee.CurrentIndustry
Education -> SelfEmployed.JobFamily
Region -> SelfEmployed.JobFamily
SelfEmployed.JobFamily -> SelfEmployed.Industry
```

There are **23 direct semantic edge templates**. The Company.Industry[Employer] payload edge to employee current Industry is selected relational derivation, not a stochastic family-wide parent or reverse Company-generation dependency.

**BRANCH GATE — NORMATIVE:** these six relationships are separate from model parents:

```text
PrimaryActivity -- employee gate --> Employee.JobFamily
PrimaryActivity -- employee gate --> Employee.JobLevel
PrimaryActivity -- employee gate --> Employer
PrimaryActivity -- employee gate --> Employee.CurrentIndustry
PrimaryActivity -- self-employed gate --> SelfEmployed.JobFamily
PrimaryActivity -- self-employed gate --> SelfEmployed.Industry
```

Company.ID has no socioeconomic edges. Income and Layer 3 have no concrete incoming edges frozen here; their downstream-only boundary permissions do not widen this current edge list.

### 20. Topological order and acyclicity proof

**DAG — NORMATIVE:** one valid complete ordering of the current node templates, including the auxiliary protocol identity, is:

```text
1.  AnchorAge
2.  Region
3.  OperationalCompanyUniverse
4.  Company.ID[*]                         auxiliary identity
5.  Company.Industry[*]
6.  Company.EmployeeCount[*]
7.  Company.SizeBand[*]
8.  Education
9.  PrimaryActivity
10. Employee.JobFamily                   when employee
11. Employee.JobLevel                    when employee
12. Employer                            when employee
13. Employee.CurrentIndustry            selected payload after Employer, when employee
14. SelfEmployed.JobFamily               when self-employed
15. SelfEmployed.Industry                when self-employed
```

Future Income may attach after this current graph under Contract 5; future Layer-3 facts may attach downstream under their contracts. Those placeholders follow the current ordering, without this document assigning their exact parents or relative internal scheduling.

**PROOF:** every listed direct edge and every gate goes from an earlier position to a later one. Company.Industry[Employer] references an already earlier Company.Industry[j] after Employer supplies j; it introduces no new sampled node or feedback path. Same-Company edges have no cross-coordinate return path. Company facts never consume Person facts; Employer never returns information to occupational generation; self-employed Industry never returns information to JobFamily. Therefore no directed cycle exists, including when gate relationships are counted.

The ordering lists both branch templates for completeness. A particular Person executes only its applicable subsequence. Constructor precomputation and equivalent immutable lookup scheduling may differ without changing this dependency graph.

### 21. Forbidden edges

**NORMATIVE:** any direct semantic edge absent from section 19 is forbidden. The following important prohibitions are explicit; exclusions cannot be circumvented transitively.

| Forbidden dependency | Reason / exact scope |
| --- | --- |
| Gender -> any Contract-4 generation node | Frozen direct/transitive exclusion; also socioeconomic identity |
| CurrentAge -> any Contract-4 generation node | Frozen exclusion; AnchorAge cannot be reconstructed from it |
| ReferenceDate -> any Contract-4 generation node | All facts are anchored; also excluded from stable identity |
| Layer1 SnapshotID -> any Contract-4 generation node | Snapshot identity is not socioeconomic generation state |
| FaceCatalog -> any Contract-4 generation node | No catalog-dependent generation or stable identity |
| PhotoID -> any Contract-4 generation node | No photo-derived input or proxy |
| Excluded-input-derived proxy -> any Contract-4 generation node | Includes gender-dependent names, snapshot-derived buckets and fingerprints |
| Unauthorized Layer-1 observation -> current model or selection node | Only AnchorAge and exact Region are authorized semantically |
| Person facts -> Company.Industry | Company generation is Person-independent |
| Person facts -> Company.EmployeeCount | Modeled workforce cannot depend on generated Persons |
| Person facts -> Company.SizeBand | Sole parent is modeled EmployeeCount |
| Person Region or AnchorAge -> Company model fact | These belong to a Person, not a frozen Company input |
| Realized occupancy -> Company facts | No assignment feedback or headcount reconciliation |
| Assignment history / Person request pattern -> Company or selection facts | No mutable population-dependent sampling state |
| Company.EmployeeCount -> Company.Industry | Reverse of the selected Company factorization |
| Company.SizeBand -> Company.EmployeeCount | Count determines band; no reverse resampling or target band |
| Company.SizeBand -> Company.Industry | No direct size-to-Industry model dimension |
| Company[k] fact -> Company[j] model fact for k != j | No cross-Company generation dependency is frozen |
| PrimaryActivity -> Education | Education is completed attainment before branch selection |
| Occupational fact / Employer / Company fact -> Education | No downstream attainment feedback |
| Occupational fact / Employer / Company fact -> PrimaryActivity | Primary state cannot inspect downstream occupations or selection |
| Employer -> Employee.JobFamily | Reverse of JobFamily -> Employer; forbidden as generation dependency |
| Employer -> Employee.JobLevel | Reverse of JobLevel -> Employer; no employer-caused responsibility generation |
| Employer Company Industry / SizeBand / EmployeeCount -> Employee.JobFamily or Employee.JobLevel | Company matching is downstream selection only |
| Employee.JobLevel -> Employee.JobFamily | Reverse of JobFamily -> JobLevel |
| Employee.CurrentIndustry -> employee occupation, Employer, or Company fact | Derived result cannot feed back into its inputs |
| Company.Industry[k], k != Employer -> Employee.CurrentIndustry directly | With Employer fixed, only the selected lookup payload is a direct Industry parent; family-wide Employer selection remains authorized |
| Education / AnchorAge / Region -> Employer directly | They influence selection only through JobFamily or JobLevel |
| PrimaryActivity -> Employer conditional model dimension | Employee gate only; no extra routing dimension |
| AnchorAge -> either branch's JobFamily directly | Age composition is mediated through authorized inputs and gates |
| Region -> Employee.JobLevel directly | Only AnchorAge, Education and employee JobFamily are direct parents |
| SelfEmployed.Industry -> SelfEmployed.JobFamily | Reverse of the chosen self-employed direction |
| Company / Employer / JobLevel / owned business -> self-employed model nodes | No such self-employed architecture exists in v1 |
| Employee occupational model node -> self-employed model node, or conversely | Branches have shared authorized base causes, not cross-branch edges |
| Income -> any Contract-4 node | Income is future downstream only |
| Layer-3 fact -> any Contract-4 node | No customer value, audience, preference, consent or other Layer-3 feedback |
| Hidden socioeconomic latent -> public Layer-2 fact | No latent node has been frozen |
| Global SocioeconomicWorldID -> every stochastic node as a universal dependency | Would violate model-component non-interference |
| Company.ID -> employer subset / EmployerWeight semantics | Protocol identity is not authorized socioeconomic Company information |

Coincidental correlation from common addressing or permitted causes is not permission to read a forbidden value. Access needed solely to produce the contained frozen Layer-1 Person does not enter this socioeconomic subgraph.

### 22. Semantic non-interference

**NORMATIVE:** changing model/data for one stochastic node may alter that node and its semantic descendants. It must not alter non-descendants because addressing, identity, table loading, construction order, or implementation state is shared.

Gates are included in descendant reachability: changing PrimaryActivity may change occupational applicability and thereby the current branch's presence and contents. It must not change Education or Company facts.

The following descendant sets summarize the selected graph, before future downstream attachment:

| Changed model/data family | Current semantic descendants that may change | Important non-descendants that must remain identical |
| --- | --- | --- |
| Company Industry | Company EmployeeCount, SizeBand, Employer, employee current Industry through the selected payload or a changed Employer | Education, PrimaryActivity, both branches' JobFamily, employee JobLevel, self-employed Industry |
| Company EmployeeCount | SizeBand, Employer, employee current Industry through a changed Employer | Company Industry, all Person model facts generated before selection, self-employed facts |
| Education | PrimaryActivity, employee JobFamily, employee JobLevel, Employer, employee current Industry, self-employed JobFamily, self-employed Industry | Company Industry, EmployeeCount, SizeBand, protocol Company IDs, base inputs |
| PrimaryActivity | All six gated occupational nodes, including their applicability | Education, Company facts, base inputs |
| Employee JobFamily | Employee JobLevel, Employer, employee current Industry | Education, PrimaryActivity, Company facts, self-employed model facts |
| Employee JobLevel | Employer, employee current Industry | Employee JobFamily, Education, PrimaryActivity, Company facts, self-employed facts |
| Employer subsets / weights / routing | Employee current Industry | Employee JobFamily and JobLevel, Education, PrimaryActivity, Company facts, self-employed facts |
| Self-employed JobFamily | Self-employed Industry | Employee branch facts, Education, PrimaryActivity, Company facts |
| Self-employed Industry | None within Contract 4 | Every other current node |

A row's changed node itself may also change. Permission for descendants to change is not a requirement that every value changes; a derived or sampled descendant may retain the same value. Controlled fixtures must demonstrate allowed propagation without relying on accidental equality.

For a Company-coordinate-specific change, only its explicit same-coordinate generation descendants can change directly. Selection and the selected relational output may also change. A global Company Industry model change changes the Industry node family; this does not authorize cross-Company fact reads during individual Company generation.

These family-level descendant permissions do not make every Company's Industry a direct parent of a fixed employee's current Industry. Given Employer = j, only Company.Industry[j] is the direct Industry payload. Changing Company.Industry[k], k != j, may affect family-wide Employer-selection semantics, but MUST NOT change the derived current Industry while Employer remains j. If Employer changes, the derivation uses only the newly selected payload.

If authorized parents retain the same values and a child's own model/data and permitted addressing dependencies remain unchanged, changing unrelated ancestor implementation or metadata must not perturb that child. Later protocol fingerprints must not treat an ancestor's entire implementation identity as additional child information beyond the frozen dependency boundary.

Model assets physically shared across nodes require isolation of their output-affecting identities. A change intended to one node is not allowed to become a change to every stream merely because a whole-file checksum or global version is reused.

**SOURCE FACT:** frozen Layer-1 dataset-isolation tests already distinguish age, name, place, postal and street dependencies at `tools/fakedata/layer1_data_test.go:15`. A broad stable WorldID is not a substitute for selective dependencies; Contract 1 expressly forbids using socioeconomic identity as an RNG mega-seed at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:405`.

**PROTOCOL — REQUIRED:** the future Deterministic Protocol must provide sufficient stream and fingerprint isolation to satisfy these guarantees. This document fixes the guarantee and graph, not the decomposition or bytes.

### 23. Random access and concurrency

**NORMATIVE:** the selected DAG preserves Person random access, Company random access, order independence, reverse-order equivalence, and concurrent read safety.

Generating Person `i` may inspect only its permitted frozen Layer-1 inputs and immutable model, Company-world, routing and index state. It may never inspect another generated Person. Generating Company `j` never inspects a generated Person or a generated Company's fact outside its own listed parents.

No population materialization, previous Person generation, assignment history, realized occupancy, mutable sampling cursor, shared advancing RNG, or output-sensitive cache population is permitted.

Company universe construction may evaluate Company-scoped facts to build immutable indexes. This finite model construction is not Person population materialization. It is bounded by frozen Contract 2 and later data owners.

Per-Person occupational generation remains bounded local work. Employer lookup uses the immutable routed Contract-2 index and logarithmic selector, with no per-Person Company-universe scan. These requirements are frozen at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:708` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:859`.

Repeated, forward, reverse and concurrent retrieval agree by contents. Independent returned payloads must not expose mutable internal state, preserving `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:604`.

### 24. Constructor vs lookup ownership

**NORMATIVE:** semantic ownership is:

| State / work | Ownership class | Requirement |
| --- | --- | --- |
| Validated prescribed CompanyCount and OperationalCompanyUniverse | Constructor-owned immutable state | Established before a usable wrapper escapes |
| Validated D2 assets and node models | Constructor-owned immutable state | All-or-nothing validation; no new public model configuration |
| Dense EmployeeCount metadata | Constructor-owned immutable state | Required by Contract 2; generated under the Company DAG |
| Employer-selection indexes | Constructor-owned immutable state | All required indexes exist and validate before return |
| Routing mechanism / retained routing state, if later required | Constructor-owned immutable state or validated bounded deterministic resolution | Complete reachable-condition coverage; representation deferred |
| Company Industry and EmployeeCount semantic generation | Per-Company model generation | Person-independent; may be evaluated during construction and consistently retrieved |
| Education, PrimaryActivity and applicable occupation models | Per-Person model generation | Exact parents and branch gates only |
| Employer | Relational selection | Immutable routed index; frozen weighted-selection protocol |
| Company SizeBand | Pure deterministic derivation | Exact count mapping; no independent RNG |
| Employee current Industry | Pure relational derivation | Employer key and exactly Company.Industry[Employer] payload; no unselected Industry read, independent RNG or copied field |
| Company.ID | Pure deterministic protocol derivation | Existing namespace/coordinate identity protocol |

No concrete struct layout is frozen. Construction cannot defer required indexes into lazy mutable generation. Immutable model assets may be shared when their semantic and fingerprint isolation is preserved.

Resource accounting for future retained Industry data, cached bands, D2 tables and routing belongs to the contract introducing that state. It cannot weaken or silently consume an uncharged exception to Contract-2 budgets. Separate routing ownership is frozen at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:662`; future Company-state ownership is explicit at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:739`.

### 25. Employer-index feasibility

**SOURCE FACT:** frozen category counts are EducationLevel 7, JobFamily 10, JobLevel 4, Industry 14 and SizeBand 5 at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:725`. AnchorAge has 55 possibilities as cited in section 6. The index maximum is 1,024 at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:518`.

**PROOF:** the entire Person-side employer condition has raw Cartesian cardinality:

```text
10 JobFamily values × 4 JobLevel values = 40 condition states
```

PrimaryActivity is constant inside the applicable employee branch. Education, AnchorAge and Region are not additional condition dimensions. Industry and SizeBand are Company candidate facts, not Person condition coordinates. OperationalCompanyUniverse is fixed for a constructed World.

There can therefore be at most 40 distinct employer-selection distributions required solely by this Person condition in one World. Routing may map several conditions to the same index. Contract 4 neither requires 40 indexes nor freezes one index per condition. It inherently requires no more than the 1,024-index limit.

**NORMATIVE:** the aggregate input and retained-memory budgets remain independent obligations. A 40-state condition does not prove that 40 complete-universe indexes fit. At maximum CompanyCount, retaining every Company in each of 40 indexes would require 41,943,040 positive entries and exceed the frozen 64 MiB index budget. Such a later model is invalid, even though its index count is legal.

**RATIONALE — feasibility witness only:** a deliberately small immutable test-world domain can contain one positive-workforce Company in each of the 14 Industry × 4 positive SizeBand cells: 56 Companies. Even allowing 40 distinct indexes with all 56 Companies represented in each gives:

```text
K = 40
P = M = 40 × 56 = 2,240

retainedIndexBytes = 64 + 32×40 + 16×2,240 = 37,184
selectionInputBytes = 37,184
companyMetadataBytes = 64 + 4×56 = 288
Contract-2 retainedBytes = 37,472
```

The Contract-2 reference working-space charges, with no additional retained or other live state, give:

```text
37,472 retained
+ 448 raw EmployeeCount staging
+ 896 current raw input
+ 1,792 canonicalization working space
+ 4,096 fixed overhead
= 44,704 charged construction bytes
```

This witness permits simultaneous Industry-sensitive family selection and size-sensitive level selection within all index and Contract-2 resource limits. It freezes neither actual CompanyCount, actual Company values, index subsets, weights, index count, routing, nor implementation sorting. Its purpose is to prove that the DAG does not inherently demand an infeasible index family.

Later D2 must separately prove resource feasibility and positive-total coverage for its actual chosen World/model, including any future retained routing or Company data. Missing empirical evidence is not a DAG blocker.

### 26. D2 boundary

**NORMATIVE:** for every current stochastic/model node, Contract 4 owns exact semantic parents, branch applicability, generation direction and permitted information. Later owners supply the law:

| Node / model | Frozen by Contract 4 | Later MODEL/D2 owner supplies |
| --- | --- | --- |
| Company Industry | No semantic parents; Company-scoped and Person-independent | Industry distribution, source mappings and model/data identity |
| Company EmployeeCount | Same-Company Industry only | Industry-conditioned count distribution, source conversion and safe validated data |
| Education | AnchorAge and Region | Completed-attainment probabilities, age/Region pooling and educational crosswalks |
| PrimaryActivity | AnchorAge, Education and Region | Six-state probabilities, empirical overlap resolution and source mappings |
| Employee JobFamily | Education and Region | Employee functional-family probabilities and source crosswalks |
| Employee JobLevel | AnchorAge, Education and employee JobFamily | Responsibility-tier probabilities and permitted model buckets |
| Employer | Exact JobFamily/JobLevel condition; exact Company inputs and dense domain | Index count, semantic subsets, integer EmployerWeight formulas and deterministic routing semantics |
| Self-employed JobFamily | Education and Region; separate branch model | Self-employed functional-family probabilities and crosswalks |
| Self-employed Industry | Self-employed JobFamily only | Conditional current-economic-activity probabilities and crosswalks |

Company/D2 also selects one exact CompanyCount under Contract 2. Model-local bucket thresholds, tables, empirical source identities and fitted numerical relationships remain versioned MODEL/DATA choices. Frozen public taxonomy meanings and SizeBand thresholds remain unchanged.

D2 may pool authorized dimensions, transform them without extra information, and fit correlations by the selected mechanisms. It may not add a Person-side employer condition, finer geography, Company field, hidden latent, cross-node joint sampler, feedback path, or unauthorized direct ancestor input.

A numerical fit that needs an unauthorized residual relationship must remain within the v1 modeling limitation or explicitly request reopening Contract 4. It cannot silently widen the graph. No empirical probability or crosswalk is invented here.

### 27. Income boundary

**DAG — NORMATIVE:** Income is future downstream of Contract-4 socioeconomic facts. No Contract-4 node depends on Income, Income-model identity as stochastic addressing state, or an Income-derived proxy.

Contract 5 owns Income type, applicability, amount/band semantics, exact parent set and distributions. This document assigns none of those. Contract 5 may select available information only within the frozen semantic-input and exclusion boundaries and without feedback into the current graph.

An Income-model/data change may change Income and its later descendants and eventual full socioeconomic identity. It must not perturb Company Industry, EmployeeCount, SizeBand, Education, PrimaryActivity, either occupational branch, Employer, or employee current Industry.

### 28. Layer-3 boundary

**ARCHITECTURE — NORMATIVE:**

```text
Layer 2 -> future Layer 3: permitted downstream attachment
Layer 3 -> Layer 2: forbidden feedback
```

Customer value, audiences, preferences, consents and all other Layer-3 facts or model-derived proxies cannot become Layer-2 parents or generation fingerprints. This includes attempts to calibrate Layer-2 facts by evaluating a generated Layer-3 output.

No Layer-3 schema, model, applicability or exact parent set is designed here.

### 29. Deterministic Protocol boundary

**PROTOCOL — DEFERRED:** later protocol work owns stream path strings, Component identifiers, framing additions, seed bytes, fingerprints, precise sampling order and draw sequences not already frozen, routing representation where assigned to that owner, and exact SocioeconomicWorldID encoding.

Contract 4 does not prescribe map/slice layout, routing structures, public protocol APIs, or stream names.

**NORMATIVE:** the protocol must realize the exact parent and gating graph, excluded-input boundary, fresh random access, and semantic non-interference. It must provide sufficient node/branch/model isolation that a local model change cannot perturb a non-descendant through shared global state.

**World identity:** eventual SocioeconomicWorldID covers every output-affecting Layer-2 model/data/spec choice, including Income, and binds the actual stable Layer-1 WorldID as already required by `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md:390`.

**Generation dependency:** each node uses only its authorized semantic information, its own relevant model/data, and compatible deterministic addressing. The complete global SocioeconomicWorldID must not automatically be included in every stochastic dependency set. Its drift following a model change is not permission to regenerate unrelated facts differently.

The same distinction applies to using the broad Layer-1 WorldID as a universal stream input when unrelated underlying data changes would perturb otherwise fixed Layer-2 semantic inputs. Identity coverage is not a universal stochastic dependency. The future protocol must specify the exact compatible decomposition without changing frozen Layer-0/1 stream machinery.

### 30. Identity/versioning

**NORMATIVE:** changing any of the following after freeze is output-sensitive and requires model/protocol version review and eventual SocioeconomicWorldID coverage:

- Node set or node meaning.
- Direct edge or parent set, including authorized Layer-1 input abstraction.
- Occupational branch gating or semantic evaluation behavior.
- Derived-versus-sampled or relational-versus-stored classification.
- Company Industry/EmployeeCount direction.
- Employee JobFamily/JobLevel/Employer direction.
- Employer Person-side conditioning dimensions or Company-side authorized inputs.
- Self-employed causal direction.
- Hidden-state policy or semantic non-interference boundary.
- Output-affecting model probabilities, source mappings, bucket thresholds, selection weights, subsets or routing semantics under their later owners.

Equivalent implementation scheduling, immutable storage changes, and helper representation changes that preserve this graph, frozen protocol semantics and every generated value do not by themselves require identity drift.

Resource-only changes retain the Contract-2 distinction between acceptance envelope and output semantics for Worlds valid under both versions. They do not authorize output-sensitive model changes without version coverage.

SocioeconomicWorldID remains invariant under ReferenceDate, SnapshotID, FaceCatalog and PhotoID changes. It never incorporates Gender as a generation dependency. Broad identity coverage and selective generation isolation must hold simultaneously.

### 31. Required future tests

**NORMATIVE:** specify these tests for future implementation; none is implemented by this drafting task. Use intentional controlled fixtures, literal expected relationships and independent call-graph/model-input inspection. Coincidentally equal random results are insufficient isolation evidence.

| Area | Required controlled verification |
| --- | --- |
| Parent-set completeness | Inspect each current node's model, helper, table keys, routing and stream dependencies against the exact table; no extra fact or direct ancestor input |
| Parent isolation | Mutate one node's model/data with all other models and allowed inputs fixed; force a distinguishable output and verify every non-descendant exactly invariant |
| Descendant propagation | Deliberate fixtures show permitted paths can transmit a change, including Education -> activity/occupation -> Employer and Company Industry -> count -> band -> selection; do not require every descendant to change in every fixture |
| Company factorization | Industry-model changes may affect same-Company EmployeeCount; count-model changes do not alter Industry; no cross-Company fact input or hidden joint latent |
| Forbidden dependency probes | Inspect call graphs, semantic proxies, identity inputs and fingerprints for Gender, CurrentAge, ReferenceDate, SnapshotID, FaceCatalog and PhotoID; vary excluded state while permitted inputs remain fixed where feasible |
| AnchorAge | Use frozen birth's anchor-age result; probes show no reconstruction from CurrentAge, birth-year dimension, or ReferenceDate |
| Exact geography | Recover the canonical frozen municipality through both frozen mixture paths and project its exact Region; assert no independent municipality draw, ambiguous display-name lookup, postal or street dependency |
| Geography abstraction isolation | Hold Region fixed while varying finer municipality, Province, PostalCode and Place population in deliberate test-local inputs; current model laws and outputs remain unchanged with addressing fixed |
| Company independence | Vary Person generation order, number of requests, requested coordinates, materialization and count/request pattern; Company facts and immutable indexes remain identical |
| Branch gating | Intentionally exercise all six PrimaryActivity branches; non-applicable occupational model evaluation, employer routing and derived occupation lookup must observably fail if accidentally invoked |
| Branch separation | Employee-model changes leave self-employed branch realization unchanged and conversely, with shared base inputs and common models fixed |
| Employer conditioning | Independent fixtures for JobFamily and JobLevel show each permitted dimension can deliberately change selection semantics; direct Education, AnchorAge or Region variation at fixed JobFamily/JobLevel cannot change routing or weights |
| Company-side employer information | Fixtures exercise Industry, EmployeeCount and exact count-derived SizeBand in subset/weight semantics; zero count always gives zero employee-employer weight |
| Count versus band selection | Vary positive counts within one SizeBand to exercise authorized count sensitivity; vary across exact band boundaries to exercise permitted band sensitivity; never fabricate an inconsistent count/band pair |
| Unauthorized Company inputs | No Company.ID semantic weighting, Company geography, revenue, future field, occupancy or assignment-history input |
| Employer correlation mechanism | Crafted Company pools and controlled draw boundaries show Industry×JobFamily and SizeBand×JobLevel selection effects while occupation is held fixed after its generation |
| Relational Industry | Fix Employer = Company 10: changing Company 20 Industry leaves current Industry identical; changing Company 10 Industry may change it. Changing Employer to Company 20 resolves Company 20 Industry, including a fixture where both Companies share Industry. Prove only Company.Industry[Employer] is read; no independent draw, copied field or reverse edge |
| Selection versus derivation | Changing an unselected Company's Industry may deliberately change family-wide Employer-selection semantics while the fixed-employer derivation remains invariant; a controlled lookup probe must observably reject any unselected Industry read during derivation |
| Self-employed direction | JobFamily changes may alter Industry; self-employed Industry-model changes cannot alter JobFamily; no Employer, Company, JobLevel or business entity evaluation |
| Hidden-state exclusion | Inspect all random innovations and model intermediates; no shared socioeconomic score, latent, sampled overlapping state or reused cross-node draw |
| Global identity isolation | Change one node's output-affecting model identity; full SocioeconomicWorldID changes when required, while unrelated node facts remain identical despite global identity drift |
| Random access | Forward, reverse, repeated and concurrent Person and Company retrieval agree by contents; appropriate race verification once implemented |
| ReferenceDate | Valid Worlds differing only in ReferenceDate have identical socioeconomic facts, Company facts and SocioeconomicWorldID; a deliberate composite fixture changes CurrentAge |
| Catalog/photo invariance | Valid alternate catalogs preserve socioeconomic facts, Company facts and identity; facts-only must not perform photo selection |
| Domain change isolation | With Company models and addressing fixed, common operational Company coordinates retain facts when only the model-domain count changes in an internal version fixture |
| Employer routing feasibility | Exhaust every reachable condition; exactly one existing positive-total index per condition, no fallback; at most 1,024 indexes and all Contract-2 aggregate resource checks before excessive allocation |
| Future Income | Income-model/data changes do not perturb any current Contract-4 fact, even when eventual full socioeconomic identity changes |
| Future Layer 3 | Layer-3 data/model changes do not perturb Layer-2 facts or enter their generation dependency sets |
| Frozen compatibility | Existing Layer-0/1 tests and goldens remain unchanged and pass; facts-only and composite facts agree by contents |

Derived SizeBand and relational Industry tests must exercise actual successful output, not a duplicate helper formula. Gate probes require proof of non-evaluation, not merely nil final payloads. Test-only fixtures do not create public configuration, require frozen-source changes, or justify extraction solely for unit testing.

Authorized employer-condition dimensions may be pooled by production D2. The conditioning tests demonstrate the legal capability with controlled models; they do not require 40 distinct production indexes or distinct production probabilities for every category pair.

### 32. Decision ledger

"Frozen now?" means resolved normatively in this proposal, not independently approved for freeze.

| Subject | Decision | Class | Frozen now? | Later owner |
| --- | --- | --- | ---: | --- |
| Approved Layer-1 semantic inputs | Exactly {AnchorAge, Region}; node-specific uses only | DAG | Yes | None |
| Geography decision | Exact private placeRow.region of the frozen selected address municipality; no finer geography | DAG | Yes | None |
| Company Industry parents | ∅ | DAG | Yes | D2 values |
| Company EmployeeCount parents | {same-Company Industry} | DAG | Yes | D2 values |
| SizeBand derivation | Frozen Contract 3 exact EmployeeCount mapping | DERIVED | Yes | None |
| Education parents | {AnchorAge, Region} | DAG | Yes | D2 probabilities |
| PrimaryActivity parents | {AnchorAge, Education, Region} | DAG | Yes | D2 probabilities |
| Employee JobFamily parents | {Education, Region} | DAG | Yes | D2 probabilities |
| Employee JobLevel parents | {AnchorAge, Education, Employee.JobFamily} | DAG | Yes | D2 probabilities |
| Employer Person-side condition | Exactly {Employee.JobFamily, Employee.JobLevel} | DAG | Yes | Company/D2 weights/subsets |
| Employer Company-side inputs | Exactly {EmployeeCount, Industry, SizeBand}; fixed operational domain also explicit | DAG | Yes | Company/D2 weights/subsets |
| Employee Industry | {Employer, Company.Industry[Employer]}; selected relational key and payload only | RELATIONAL DERIVED | Yes | None |
| Self-employed JobFamily parents | {Education, Region} | DAG | Yes | D2 probabilities |
| Self-employed Industry parents | {SelfEmployed.JobFamily} | DAG | Yes | D2 probabilities |
| Hidden socioeconomic nodes | None; no new latent or sampled shared state permitted | DAG | Yes | None |
| Branch gating | Education/activity always; only applicable occupation branch; no sample-and-discard | DAG | Yes | None |
| Semantic non-interference | Node changes restricted to node and semantic/gated descendants | DAG | Yes | Deterministic Protocol realization |
| Income direction | Downstream only; no current incoming edge assigned | DAG | Yes | Income Contract |
| Layer-3 feedback | Forbidden | ARCHITECTURE | Yes | None |
| Exact probabilities | Deferred | MODEL | No | D2 |
| Empirical crosswalks | Deferred | MODEL / DATA | No | D2 |
| Exact stream paths/Components | Deferred | PROTOCOL | No | Deterministic Protocol |
| Exact SocioeconomicWorldID encoding | Deferred | PROTOCOL | No | Deterministic Protocol |
| Actual CompanyCount | Deferred within frozen Contract-2 bounds | MODEL | No | Company/D2 Model Contract |
| Exact index count, subsets and EmployerWeight formulas | Deferred within closed input sets and all Contract-2 limits | MODEL | No | Company/D2 Model Contract |
| Routing representation and retained resources | Deferred; complete deterministic coverage required | PROTOCOL / RESOURCE | No | Later contract introducing routing state |

### 33. Cross-contract contradiction audit

| Audit case | Classification | Explanation |
| --- | --- | --- |
| 1. Selected base inputs vs frozen exclusions | NO ISSUE | Only AnchorAge and Region are admitted; all six exclusions and proxy prohibitions remain binding, as frozen in Contract 1:612. |
| 2. Geography vs frozen Layer-1 availability | NO ISSUE | Private placeRow.region exists and the canonical municipality prefix is independently callable; section 6 gives exact source evidence. No public API change. |
| 3. Company DAG vs Person independence | NO ISSUE | Company Industry is a root; EmployeeCount uses only its own Company's Industry; band derives from count. Contract 2:806 forbids Person and occupancy inputs. |
| 4. Industry/EmployeeCount direction vs Company semantics | NO ISSUE | Industry-conditioned modeled workforce does not alter what either public fact means; Contract 2:268 and Contract 3:479 deferred generation. |
| 5. SizeBand vs Contract-3 mapping | NO ISSUE | Exact zero/micro/small/medium/large thresholds and no-RNG derivation are preserved from Contract 3:280. |
| 6. Education parents vs completed attainment | NO ISSUE | Age and Region condition highest completed attainment; activity, enrollment and occupation cannot define it. Contract 3:149 remains unchanged. |
| 7. PrimaryActivity parents vs applicability | NO ISSUE | One six-state model using only age, attainment and Region precedes exact gates; Contract 3:545 remains exhaustive. Numerical overlap treatment is later MODEL/D2 work. |
| 8. JobFamily/Employer direction vs employee Industry | NO ISSUE | JobFamily precedes Employer; current Industry has exactly Employer and Company.Industry[Employer] as relational parents. Family-wide Employer-selection inputs remain unchanged. No occupation or Company feedback; Contract 3:503 is preserved. |
| 9. JobLevel parents vs responsibility meaning | NO ISSUE | Age, attainment and family condition a responsibility model; they do not replace the authority semantics at Contract 3:417. |
| 10. Person-side employer condition vs Contract 2 | NO ISSUE | JobFamily/JobLevel complete the deferred assignment condition without altering the selector, which Contract 2:484 expressly left to later contracts. |
| 11. Company-side selection inputs vs frozen fields | NO ISSUE | EmployeeCount, Industry and exact derived SizeBand are all present in Contract 3:457; no Company geography or future field is invented. |
| 12. Routing vs 1,024-index maximum | DEFERRED CLEANLY | The condition has only 40 raw states and an explicit resource-feasible witness; actual indexes, weights, routing and aggregate budget proof remain Company/D2-owned under Contract 2:516. |
| 13. Self-employed DAG vs no-Company architecture | NO ISSUE | Family precedes direct Industry without Employer, Company, JobLevel or business entity, preserving Contract 3:522. |
| 14. Hidden-state policy vs D2 freedom | NO ISSUE | D2 retains probabilities and deterministic parent transforms; it has no authority to change the graph through a latent. Contract 3:690 deferred permitted relationships to this owner. |
| 15. Branch gating vs non-working absence | NO ISSUE | Inapplicable occupational facts are not evaluated; output absence remains exactly Contract 3:545. |
| 16. Non-interference vs random access | NO ISSUE | Explicit isolated dependencies, immutable indexes and no request-history inputs preserve random access. Fixed-employer current Industry reads only its selected payload; unselected Industry can affect selection separately. Contract 1:598 remains unchanged. |
| 17. Broad world identity vs per-node isolation | DEFERRED CLEANLY | Full identity covers all output-affecting choices while selective fingerprints must preserve non-descendants; Contract 1:405 already forbids an RNG mega-seed. Exact decomposition is protocol-owned. |
| 18. ReferenceDate exclusion vs anchor facts | NO ISSUE | Frozen birth age and municipality selection have no date input; socioeconomic facts remain anchored as required at Contract 1:283. |
| 19. DAG vs D2 ownership | DEFERRED CLEANLY | All current parent sets and directions are fixed; section 26 assigns all numerical laws and source mappings without authorizing extra dimensions. |
| 20. DAG vs Deterministic Protocol ownership | DEFERRED CLEANLY | Graph and isolation are semantic requirements; exact streams, Components, draw scheduling, fingerprints and bytes remain deferred as in Contract 3:900. |
| 21. Income downstream attachment | DEFERRED CLEANLY | No feedback exists; Contract 5 can assign available downstream parents without altering current nodes. No Income type or applicability is fixed. |
| 22. Layer-3 feedback prohibition | NO ISSUE | Only downstream consumption is permitted; no customer or preference value is an upstream dependency. |

No case requires a frozen-contract fix. The citation aliases `Contract 1:line`, `Contract 2:line` and `Contract 3:line` in this audit refer to the exact full filenames and verified hashes in section 1. Earlier draft files are not evidence.

### 34. Independent-review questions

1. Is the final graph acyclic?
2. Does every current model node have exactly one explicit parent set?
3. Are branch gates distinguished from model parents?
4. Are semantic parents separated from deterministic addressing inputs?
5. Is the geography abstraction exact and available in frozen Layer 1?
6. Does the Company DAG permit plausible Industry×size correlation?
7. Can Education and PrimaryActivity express plausible age/geography relationships?
8. Is the JobFamily↔Employer direction coherent and cycle-free?
9. Can Company Industry×JobFamily correlation arise under the selected architecture?
10. Can Company SizeBand×JobLevel correlation arise without a reverse edge?
11. Is Employer conditioned on the minimal sufficient Person information?
12. Are Company-side EmployerWeight inputs precisely bounded?
13. Can the employer design be implemented within frozen Contract-2 index/resource limits?
14. Is self-employment acyclic and capable of Industry×JobFamily correlation?
15. Are Company facts completely independent of generated Persons?
16. Are non-working occupational nodes genuinely gated off?
17. Are excluded Layer-1 values absent directly and transitively?
18. Does the hidden-state policy prevent D2 from silently changing the DAG?
19. Is semantic non-interference sufficiently strong, including selected-payload isolation for current Industry with Employer fixed?
20. Can the later deterministic protocol isolate nodes without using global World identity as a universal stochastic dependency?
21. Can later D2 fit plausible distributions without adding a semantic parent?
22. Can Income attach downstream without reopening Contract 4?
23. Is Layer-3→Layer-2 feedback impossible by contract?
24. Can two independent implementations follow this DAG plus identical later D2/protocol contracts without materially different dependency semantics?

### 35. Contract v0 verdict

**NORMATIVE:** Contract 4 preserves one coherent, acyclic socioeconomic graph, with exact Region and AnchorAge inputs, Industry-before-count Company generation, attained Education before activity, occupational family and responsibility before Employer, employee Industry derived from exactly Employer and Company.Industry[Employer], and family-before-Industry self-employment.

The closed employer Person condition has 40 possible states and requires no more than 1,024 distinct indexes. Aggregate index and data feasibility remains a separate later proof; a bounded witness shows no inherent contradiction.

The self-review checked every current parent, reverse direction, branch gate, Company/Person boundary, unauthorized geography dimension, hidden-state route, global-identity coupling, occupancy feedback and future downstream boundary. No cycle or contradiction with frozen Layer 0/1 or Contracts 1–3 was identified.

The deliberate residual relationships marked NOT MODELED IN V1 are modeling limitations, not missing facts to be invented. Their omission does not prevent the selected broad correlation mechanisms from expressing Plausible+ tendencies. Empirical fit remains unproven until D2 supplies evidence; missing evidence is not a blocker.

This corrected proposal is pending independent Astra semantic review. Layer 2 is not implemented, and no distributions or empirical weights are frozen by this specification.

## Source-backed risks

- **Facts-only dependency leakage.** Full Person generation calls Gender, gender-dependent names, address and photo selection at `tools/fakedata/layer1.go:509`, `tools/fakedata/layer1.go:517`, and `tools/fakedata/layer1.go:537`. It cannot be used as the socioeconomic input-acquisition path.
- **Incomplete operational validation.** Frozen `birth` begins with stream derivation, while full Person validates operational bounds at `tools/fakedata/layer1.go:355` and `tools/fakedata/layer1.go:498`. New facts generation must validate before invoking lower-level helpers.
- **Wrong Region or finer geography leakage.** Public Address omits Region, private rows contain finer fields, and municipality selection has two paths: `tools/fakedata/layer1.go:85`, `tools/fakedata/layer1_data.go:87`, `tools/fakedata/layer1.go:465`. A fresh municipality draw, display-name reconstruction, or downstream use of another row field would violate the selected abstraction.
- **Existence confused with employer eligibility.** Zero-count Companies remain operational but have zero employee-employer weight at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:264` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:297`. Later models must validate positive-total coverage for every reachable condition without repairing Companies from Person assignments.
- **Index-count feasibility mistaken for memory feasibility.** Contract 2 freezes both K and aggregate entry charges at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:537` and `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:549`. Forty full-universe indexes can exceed the entry budgets despite legal K.
- **Uncharged future state.** Contract-2 budget ownership and future routing accounting are limited at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md:662`; future Company retained-data accounting remains explicit at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:739`. Region-authorized models and routing tables do not make their future retained representations free.
- **Relational identity context lost.** Generic Company ID decoding knows no operational CompanyCount at `tools/fakedata/identity.go:135`. Current employee Industry must be resolved through the generating socioeconomic World and its valid Employer membership, reading exactly Company.Industry[Employer]; family-wide Industry inspection belongs to Employer selection, not this derivation.
- **Universal fingerprint ripple.** Fresh stream derivation hashes the supplied root and dependencies at `tools/fakedata/stream.go:44` and `tools/fakedata/stream.go:84`. Supplying a broad output-sensitive identity everywhere would perturb non-descendants; selective protocol isolation must be designed explicitly.
- **Residual model freedom overstated.** Frozen JobFamily is broad occupational function, not Company Industry, at `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md:349`. The selected two-dimension employer condition can fit family/level-related composition, but it cannot independently fit employer differences by age, attainment or Region within a fixed family/level pair. Later evidence must be assessed against this intentional v1 limit.

## Deferred decisions

| Deferred decision | Exact owner | Binding boundary |
| --- | --- | --- |
| Actual CompanyCount | Company/D2 Model Contract | One prescribed positive literal within Contract 2; not population-derived |
| Industry and Industry-conditioned EmployeeCount distributions | Company/D2 Model Contract | Selected Company direction and Person independence |
| Education, activity and occupational probabilities | D2/MODEL contracts | Exact section-18 parent sets and branch applicability |
| Educational, occupational and economic source crosswalks | D2 contracts | Frozen taxonomy meanings; no missingness or implicit input dimensions |
| Empirical overlapping activity treatment | PrimaryActivity MODEL/D2 | One represented primary category using only its authorized parents |
| Model-local buckets and thresholds | Contract owning that model/data | Deterministic parent transforms only; exact public SizeBand unchanged |
| Employer index count, subsets and integer weights | Company/D2 Model Contract | Exact JobFamily/JobLevel condition; exact Company information; all Contract-2 budgets |
| Condition-to-index routing semantics | Company/D2 Model Contract | Every reachable condition resolves to exactly one positive-total index; sharing permitted |
| Routing representation and retained resource proof | Later contract introducing routing state | Bounded immutable resolution; no weakening of existing budgets |
| Future retained Industry, band or empirical-table storage | Contract introducing that data | Explicit safe construction and retained-state accounting |
| Income type, applicability, semantics, exact parents and distributions | Contract 5 / Income Contract | Downstream only; current facts isolated from Income changes |
| Layer-3 models and exact parents | Future Layer-3 contracts | No feedback into Layer 2 |
| Streams, Components, selective fingerprints and deterministic sampling details | Deterministic Protocol Contract | Exact semantic DAG, branch gates, random access and non-interference |
| SocioeconomicWorldID encoding | Deterministic Protocol Contract | Complete identity coverage without universal stochastic coupling |

No current semantic parent set, geography abstraction, occupational causal direction, branch gate or hidden-state decision remains deferred.

## Repository-integrity proof

The authoritative Contract-4 source was read completely and its SHA-256 matched:

```text
SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v0.md
66eb1be43c1a03d02a389b6308c1312410515cef14ac1e2fb4a970ae9fe622ed
```

Frozen Contract-1/2/3 checksums matched the values in section 1 before and after this read-only rectification.

Before/after verification captured full `git status --porcelain=v1 -uall`, branch, HEAD, the cached diff and actual Git index SHA-256, SHA-256 and status for all four authoritative contract sources, and `tools/types/semantic.go` SHA-256 and working-tree diff. All 15 corresponding capture pairs compared byte-for-byte using `cmp`.

The authoritative Contract-4 source was not replaced or modified. Its checksum and status remained identical. All pre-existing tracked and untracked status entries were preserved, including the unrelated tracked modification:

```text
 M tools/types/semantic.go
```

The preserved semantic.go SHA-256 was:

```text
06a7ff690b3e12cdaf623e3ffb36003d1675a6aec69b3ee59e9ead428cc17842
```

The required existing-package sanity check passed during this rectification:

```text
GOCACHE=/tmp/krenalis-l2-c4-relational-fix-gocache go test ./tools/fakedata
ok github.com/krenalis/krenalis/tools/fakedata 13.241s
```

This is repository sanity evidence only. It tests no Layer-2 implementation. No generator, web access or external research was used.

`git diff --check` and `git diff --cached --quiet` passed before and after rectification. Final branch and HEAD remained:

```text
property-semantics
76fa0fff109ca6abad62f55c8761275455842dc5
```

Consistency verification preserved all unaffected current parent sets, all six branch gates, the 40-state employer condition and 23 direct semantic edge templates. It checked the selected-payload parent and edge, fixed-employer lookup isolation, absence of machine-local absolute repository citations, and preservation of every cited filename and line number.

All task-created temporary integrity captures and the temporary Go cache were removed from `/tmp`. No repository file was created, modified, renamed, formatted, deleted, staged or committed by this read-only rectification.

The specification text above is the output of the read-only drafting/rectification task. Any subsequent placement or naming of that text as a repository-root review artifact is outside this task and does not alter the normative Contract-4 content.

L2 DETAILED CONTRACT 4 v0 COMPLETE — READY FOR INDEPENDENT REVIEW
