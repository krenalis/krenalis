# Synthetic World v1 — Layer 2 D2-B2 v3 publication contract

**STATUS: PROPOSED / UNFROZEN**

This is an offline, design-only publication contract. It closes the missing
publication/evidence contract identified before Run A. It produces no numerical
asset, producer implementation, source preflight or runtime protocol. Independent
review of this proposed contract precedes real D2-B2 production.

## 1. Scope and authority

This contract owns repository packaging, deterministic provenance/checksum
metadata, reproducibility evidence, and success/failure publication sets only.
Frozen Contracts 1–5/D2-A remain binding; within D2-B-owned scope the effective
precedence is v3 > v2 > v1 > v0. D2-B1 v3 §4 remains the sole acceptance
matrix. Neither an evidence representation nor a publication validation rule
adds a numerical acceptance threshold or executes superseded Income machinery.

References below use these exact root files; section/line references are to
these verified bytes, not historical reports or code introduced by this design.

| Authority ID | Exact filename | SHA-256 |
| --- | --- | --- |
| `Contract1` | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md` | `bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6` |
| `Contract2` | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md` | `e9af32a0bb9b48bf8972a408a1369901a3f3a9fe5ecd76d5762a367c055bc098` |
| `Contract3` | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md` | `df3ef0b52f37b587b8449cc71085dc9e5aca00f754bbd93e123c6fd5e3a4cb55` |
| `Contract4` | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md` | `a5ecfb735270ffba3e6afea23d85e548dd6dc3f07a8e9dfc5294a84ad3576418` |
| `Contract5` | `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md` | `8a75054f793ef47e7071e3eb9fe04c0b32abfdac9223da859f95a333a1526e67` |
| `D2A` | `SYNTHETIC_WORLD_V1_LAYER2_D2A_v1_FROZEN.md` | `33ca4c80599dddb0564ea4d27262e430b1931e0368c8d1bee6ca6c10d8078be4` |
| `D2B1.v0` | `SYNTHETIC_WORLD_V1_LAYER2_D2B1_v0.md` | `94ec7560518f0077681a9b65547832bf46c706c5e7885c7da332bbd476ccc006` |
| `D2B1.v1` | `SYNTHETIC_WORLD_V1_LAYER2_D2B1_v1.md` | `bf2e6a6dc9dbbed816190aef6705950567576c1a69833676bc4953319204236b` |
| `D2B1.v2` | `SYNTHETIC_WORLD_V1_LAYER2_D2B1_v2.md` | `36514d1c20a6bfab02b14feced6c2539fde28c5a60204a6d15629c8b9f82d6b9` |
| `D2B1.v3` | `SYNTHETIC_WORLD_V1_LAYER2_D2B1_v3.md` | `aa766ccdcae1f2c94d12f048e0fa6ba3384e801e5b5d4f6a09f0c49bee3e1a35` |

The eleventh authority ID is `PublicationContract`, with filename
`SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_PUBLICATION_CONTRACT.md` and the SHA of
its final exact reviewed bytes, calculated externally. Its SHA is not embedded
in this contract. A future production binds those reviewed bytes; proposed
status here is not freeze approval. All ten inherited authority hashes above
must match before reading/using their contents.

Supporting failure evidence is
`SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_FAILURE.md`, SHA-256
`17735b5c7c0503876f6331ab91026ff59282685261551c3a1c2f40abd244e1aa`.
It was read completely after hashing. It is neither generation authority nor
bundle input. The correct pre-Run-A stop is preserved without re-evaluation.

## 2. Evidence-obligation ledger

The following ledger was completed before selecting filenames/schemas. `Full`
means the evidence content named in that row must remain published, not that
external datasets must be copied. `Hash-only` permits external body identity
plus the required provenance, never a naked digest in place of required data.
`Distinct` describes inherited physical-file requirements before this contract
resolves undecided placement. `Bundle` is the chosen identity membership;
`Logical` means inherited logical identity material, not a universal RNG seed.
All ledger rows have future-Protocol ownership **NO**; later runtime portions
are identified separately in §§12, 15 and 18.

| ID / obligation | Exact authority location | Required content | Machine | Full | Hash-only | Distinct | Logical | Bundle | Future Protocol |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| O01 — Principal logical asset | v0 §32:1199–1229; v1 §6:222–223 | Complete inherited logical bytes | YES | YES | NO | YES | YES | YES | NO |
| O02 — Original source identities | v0 §5:98–132; §33:1239,1251 | 19 pinned external original identities and official provenance | YES | NO | YES | NOT YET DETERMINED | NO | YES | NO |
| O03 — ZIP member identities | v0 §33:1239 | Relevant member names, exact byte lengths and hashes | YES | NO | YES | NOT YET DETERMINED | NO | YES | NO |
| O04 — Normalized API identities | v0 §33:1240–1249; v3 §4 E01 | O1/O2/O3 exact normalization recipe, lengths and hashes | YES | NO | YES | NOT YET DETERMINED | NO | YES | NO |
| O05 — X4 metadata identity | v0 §5:135; §33:1241 | ISCED2011/UIS2012 inspected-text provenance, no required PDF | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |
| O06 — Crosswalk manifest | v0 §6:155; §32:1205; v1 §6:222–223 | Six named rule versions, scope, dependencies, hashes | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |
| O07 — Expanded crosswalk content/audit | v0 §§6–7:147–283; §32:1205,1227 | All nonzero coefficient entries; full X2/X3 audit; raw/canonical code provenance | YES | YES | NO | YES | NO | YES | NO |
| O08 — Generation/fit configuration | v0 §32:1205,1231; §33:1253; v3 §5.1:320–377 | Versioned effective-authority and arithmetic/profile configuration | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |
| O09 — Versioned fit/generation scripts | v0 §8:287–289; §32:1205,1231; §33:1253 | Exact complete deterministic producer source and dependency versions | YES | YES | NO | YES | NO | YES | NO |
| O10 — Fit/stage metrics | v0 §32:1205; v1 §§2–4,6; v3 §4:237–271 | Complete applicable HARD execution evidence; available fidelity metrics/coverage | YES | NO | NO | NOT YET DETERMINED | NO | YES | NO |
| O11 — Quality/acceptance evidence | v1 §6:222–228; v3 §4:237–271 | Effective 21-row matrix; separate warnings/diagnostics/deferred owner portions | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |
| O12 — Resource manifest/proof | v0 §31:1152–1195; D2-A §§21,23:547–665,698 | Actual cardinalities/charges, exact prospective ceilings, phases/guards | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |
| O13 — Whole logical checksum | v0 §32:1229 | SHA-256 of inherited canonical logical document | YES | YES | YES | NOT YET DETERMINED | YES | YES | NO |
| O14 — Per-model audit checksums | v0 §32:1229; D2-A §23:690,722 | Inherited subtree envelope checksums, node/branch separability | YES | YES | YES | NOT YET DETERMINED | YES | YES | NO |
| O15 — Independent/reordered reproduction | v0 §33:1253; v3 §4 E04; §5:307–311 | A/B byte/hash/object/count equality and reverse/permuted row checks | YES | NO | NO | NOT YET DETERMINED | NO | YES | NO |
| O16 — Human production/quality report | v0 §32:1205,1231; v1 §6:224–228; v3 §4 | Readable quality matrix, limitations, results and provenance | NO | YES | NO | YES | NO | NO | NO |
| O17 — Per-asset provenance | D2-A §23:690,701–722; v0 §29 cards | Source/crosswalk/transformation references for every model | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |
| O18 — D0/principal-role margins and flows | v0 §8:295; §14:467–469; §34:1265 | Modeled D0 margins, raw/principal margins, every transport flow | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |
| O19 — Employer fit/relationship/index proof | v0 §19:549–568; §32:1203; D2-A §23:692,716 | Cap trace, 40 analytic target tables/exact totals, pressure/support/prospective indexes | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |
| O20 — Income laws/zero/tail/repetition evidence | D2-A §23:693,699; v3 §§3–5 | Authored/canonical per-law counts, zeros, tails, atoms, spacing, occupancy/caps | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |
| O21 — Age/rare/support/sensitivity coverage | D2-A §23:694,697,699; v1 §3:144–149; v3 §4 W01,D04 | All eight maps and safe rare/support/seed-risk coverage; absent diagnostics explicit | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |
| O22 — Concept/missingness/source reconciliation | D2-A §23:695–697; v0 §§8,20; v3 §§3–4 | Versioned Income concept reconciliation; source flags/missingness/completion/limitations | YES | YES | NO | NOT YET DETERMINED | NO | YES | NO |

Frozen D2-A §18:453 says logical units are independent of eventual files;
§23:690–722 supplies per-asset provenance, counts, index/phase evidence and
separable checksum obligations, not filenames. v0 §32:1205 names evidence
families; :1231 requires versioning scripts/configuration/reports and explicitly
keeps datasets external. v1 §6:222–228 retains those families but supersedes
70-row/materiality report completeness with the effective quality table. v3
§4:237–271 requires applicable HARD evidence and disclosed available diagnostic
coverage, not a renewed high-fidelity or perturbation gate. These distinctions
control the full-versus-summary decisions in §12.

The phrase “crosswalk manifest/expanded files” requires published expanded
content, not just hashes. Neither v0 §§6–7/32 nor v1 §6 names a physical file
per crosswalk/version. A single distinct file containing all six expansions and
the complete required classification audit satisfies that plural content
family. Its manifest is an object in the metadata file. Configuration and
evidence manifests likewise have no mandated distinct physical files.

## 3. Artifact classes

- `CANONICAL_MODEL`: generated logical model content under the inherited
  logical schema; its sole file is the existing logical asset filename.
- `CANONICAL_PUBLICATION_METADATA`: canonical machine-readable provenance,
  evidence and file binding; it changes bundle identity, never model semantics.
- `VERSIONED_REPRODUCIBILITY_EVIDENCE`: exact versioned producer/crosswalk
  bytes required to reproduce/audit production; they are not logical laws.
- `HUMAN_REPORT`: readable audit narrative, noncanonical and excluded from all
  deterministic identity inputs. No prose is needed to reconstruct machine bytes.

A physical success file has exactly one class. Objects embedded in a metadata
file retain their evidence purpose without becoming additional physical files
or receiving a second artifact class.

## 4. Logical vs publication identity

Let C(x) be §7 canonical JSON bytes, including its terminal LF. Let L be the
unchanged canonical bytes of `D2B2-logical-assets-v0.json`.

**Logical model identity is SHA-256(L)**. Its separable per-model audit identities
are SHA-256(C({"schema":"SW-L2-D2B2-logical-v0","model":name,"asset":subtree}))
with exactly those three fields and the inherited subtree. The complete fixed
24-name audit inventory is:

- `ages.Education`
- `ages.Employee.JobLevel`
- `ages.Income[i]`
- `ages.Income[r]`
- `ages.Income[s]`
- `ages.Income[t]`
- `ages.Income[u]`
- `ages.PrimaryActivity`
- `companyCount`
- `companyIndustry`
- `education`
- `employeeCount`
- `employeeJobFamily`
- `employeeJobLevel`
- `employer`
- `income.e`
- `income.i`
- `income.r`
- `income.s`
- `income.t`
- `income.u`
- `primaryActivity`
- `selfEmployedIndustry`
- `selfEmployedJobFamily`

An age name references that exact mapping object; an Income name references
that exact branch object; other names reference their top-level value,
including the integer companyCount. Aggregate containers `ages`/`income` and
`schema` are not extra individual models. No new checksum domain is used.

**Publication-bundle identity is SHA-256(P)**, where P is final §7 canonical
bytes of `D2B2-publication-v3.json`. P binds every other machine file directly,
all required evidence/configuration by containment, and all authority/external
source identities. This single digest identifies a successful deterministic
bundle. It is not logical model identity or a runtime Protocol fingerprint.

A logical file SHA and whole-document logical SHA happen to equal because
that file is exactly L; they have different roles. Store the file hash once in
`artifacts`, and make `logicalAsset` a reference to that entry. Per-model audit
SHA values occupy `logicalAudits`. Temporary execution-detail digests occupy
`evidence.checks[].detail`; neither is a file hash or runtime fingerprint.

## 5. Exact success file set

All filenames are repository-root-relative. All five are unconditional and
required on success; no conditional success file exists. The contract/report
of this design, inherited authorities and historical failures are not newly
published production files. Authority bytes are bound as external repository
inputs. Historical failure files are retained, not deleted on later success.

| Exact filename | Class | Required | Schema/format and bytes | Obligations | Listed/hashed in publication artifacts |
| --- | --- | --- | --- | --- | --- |
| `D2B2-logical-assets-v0.json` | CANONICAL_MODEL | YES | `SW-L2-D2B2-logical-v0`; inherited v0 §32 exactly | O01, O13–14; generated content for O17/O20/O21 | YES |
| `D2B2-publication-v3.json` | CANONICAL_PUBLICATION_METADATA | YES | `SW-L2-D2B2-PUBLICATION-v3`; §7; embedded evidence/configuration schemas §§10–14 | O02–06, O08, O10–15, O17–22 | NO; final external SHA identifies it |
| `D2B2-crosswalks-v3.json` | VERSIONED_REPRODUCIBILITY_EVIDENCE | YES | `SW-L2-D2B2-CROSSWALKS-v3`; §§7,9 | O07; expanded bodies/audits for O06/O22 | YES |
| `D2B2-producer-v3.py` | VERSIONED_REPRODUCIBILITY_EVIDENCE | YES | Complete single CPython producer source; §10 byte/version policy | O09; executable reconstruction for O03–04/O08/O15 | YES |
| `SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_PRODUCTION_REPORT.md` | HUMAN_REPORT | YES | UTF-8 Markdown, no BOM, LF line endings, terminal LF; §17 | O16; readable quality narrative for O11 | NO |

This is the smallest selected physical set: one frozen logical filename, one
required human report, one actual producer source, one distinct expanded
crosswalk body, and one machine metadata/checksum authority. Evidence and
configuration are complete schema-bearing objects embedded in the latter,
not separate convenience files. A separate source/resource/quality/checksum
manifest, separate configuration, per-version expansion files, duplicated
assets or report-hash artifact is forbidden. There is no `plausible.md` or
`demo-plausible.md` alongside the production report: v1 §6 makes that title
reporting metadata and replaces the old report requirements; this contract
selects the one required human production report as that quality report.

## 6. Exact failure file set

A failed attempt publishes only its deterministically derived failure report
below, class `HUMAN_REPORT`, **NONCANONICAL**, in UTF-8 Markdown without BOM,
with LF and terminal LF. Record blocker, phase, authority checks,
executed/unexecuted coverage, applicable HARD finding or OPEN ISSUE, and
integrity. Its SHA is external after final bytes.

### 6.1 Append-only failure ordinals

A D2-B2 v3 production attempt is a fresh invocation of the production assembly
process, including an invocation stopped at its authority/publication input
gates. Contract design, review and dummy exercises are not production attempts.
The namespace counts **failed production publications only**, not all invocations.
A SUCCESS consumes no failure ordinal and adds no attempt-history artifact or
field to the five-file success set. An invocation interrupted before publishing
a failure consumes no ordinal. There is no hidden reservation/session ledger.

The existing unsuffixed report is failure ordinal 1:
`SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_FAILURE.md`, exact SHA-256
`17735b5c7c0503876f6331ab91026ff59282685261551c3a1c2f40abd244e1aa`.
It predates this append-only namespace clarification and remains valid historical
evidence, byte-for-byte; it requires no added ordinal line. It is never rewritten.

For every integer n >= 2, the sole filename is
`SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_FAILURE_` + base-10(n) + `.md`.
Use uppercase `FAILURE`, one underscore, lowercase `.md`, no numeric width,
no leading zero, plus sign or padding. The first suffixed name is `_2.md`,
never `_1.md`. No timestamp, UUID, retry label or alternate spelling is allowed.
Each suffixed report starts with these exact lines (N replaced by base-10(n)):

```text
# Synthetic World v1 — Layer 2 D2-B2 v3 failure

Failure ordinal: N
```

That ordinal line occurs exactly once and equals the filename ordinal. The
remaining body contains the disclosures required above; it is noncanonical
human narrative, not checksum authority. A missing disclosure or malformed
UTF-8/LF/ordinal header makes the history invalid. This validates structural
history integrity; it is not authentication of arbitrary well-formed narrative.

Before an invocation, inspect the complete repository-root failure namespace:
every entry whose name starts with `SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_FAILURE`
under ASCII case-insensitive prefix comparison (aliases are rejected, not renamed).
Only the exact unsuffixed basename and the suffixed grammar above are permitted;
all must be regular files, not symlinks/directories. Require ordinal 1's pinned
bytes, unique ordinals and a contiguous sequence 1..m, with every suffixed
header/body valid. Missing ordinal 1, a gap, alias, duplicate ordinal, corrupt
report or unexpected future ordinal stops publication as an integrity error;
never infer missing history or search for a free filename. The expected next
failure identity is m+1, derived before repository publication from this history.
An apparent future ordinal is accepted as prior history only if the entire
contiguous history, including its report, is already valid before the invocation.

If the invocation fails, revalidate that the same history is unchanged and use
exactly the previously derived m+1 filename. Create it exclusively: an existing
entry at that expected identity, or a history change during the invocation, is
an integrity error, even if the existing bytes match. Do not overwrite, retry
with m+2, or choose a different basename. Concurrent invocations derived from
the same history cannot both publish a failure. History corruption may therefore
prevent even failure-report publication; keep that integrity diagnostic external.

Normative history fixture: the current pinned unsuffixed report alone gives
`SYNTHETIC_WORLD_V1_LAYER2_D2B2_v3_FAILURE_2.md`; after its valid failure
publication the next name is `_3.md`. A SUCCESS between these two failures
leaves `_3.md` as the next name. Failure reports remain outside successful
logical/publication identities and the checksum DAG.

No canonical model, partial metadata manifest, expanded crosswalk, producer,
configuration, temporary detail body, partial numerical asset or success report
is published by that failed attempt. Temporary computation stays external.
Previously published history is not cleaned or overwritten by this rule.
No inherited authority requires a partial publication exception.

## 7. Canonical byte rules

Every NEW JSON artifact or embedded checksum domain uses UTF-8, no BOM,
`ensure_ascii=False`, `sort_keys=True`, `separators=(',',':')`, and exactly
one terminal LF, equivalent to CPython json.dumps with those arguments plus LF.
Only objects, arrays, strings, exact JSON integers, booleans and explicitly
permitted nulls occur. No JSON float, NaN, Infinity, binary float conversion,
duplicate object key, surrogate code point, extra whitespace or CRLF is legal.
Reparse and recanonicalize to prove exact byte equality before publication.
No Unicode normalization is applied to raw audit text; parsing uses inherited
rules. JSON object keys sort by Unicode code point. No extension keys exist.

Shared types, used by all field tables below:

| Type / fields | Required | Allowed structure / semantics | Order / identity role |
| --- | --- | --- | --- |
| `N` | YES | JSON integer ≥0, bool rejected; resource/count/length integer | Exact machine value |
| `H` | YES | String matching `[0-9a-f]{64}` | SHA-256 over the specified exact byte domain |
| `Q.numerator`, `Q.denominator` | YES / YES | Canonical base-10 integer strings; numerator `0` or optional minus plus nonzero first digit; denominator positive, no leading zero; gcd=1; zero denominator representation is forbidden; zero is 0/1 | Exactly two keys; arbitrary-size rational avoids JSON float/uint narrowing |
| `D` | YES | Canonical finite-decimal string, nonexponential; optional minus; no leading zeros, trailing fractional zeros, trailing decimal point or negative zero; integer zero is `0` | Exact finite Decimal value; no rounded display string |
| `Power.base`, `Power.exponent` | YES / YES | `Q` base in [0,1], `N` exponent; exact mathematical base^exponent | Avoids publishing millions of exponentiated digits; exact comparison/reconstruction remains possible |
| `Ref.authorityId`, `Ref.section`, `Ref.locator` | YES / YES / YES | ID from §1; exact section number string; locator is exact table-row source label or declaration/paragraph prefix in that section; uniqueness checked against pinned bytes | No file-system path; keys fixed; arrays of Ref sort by the three strings, unique |
| `Subject.model`, `Subject.coordinates` | YES / YES | §4 model name or `all`; array of N in that model's inherited parent/state traversal, empty for a model-wide result | Compare model by §4 lexicographic list, then integer tuples; no arbitrary naming |
| `Detail.schema`, `Detail.rowCount`, `Detail.sha256` | YES / YES / YES | `SW-L2-D2B2-DETAIL-v3`, N, H; temporary full evidence byte-domain §12 | Digest only when full body publication is not required |

All non-grid ID/file/reference arrays are ascending lexicographic and unique
unless a table expressly freezes another order. A grid's outermost dimension
varies slowest; states, ages, buckets and canonical law IDs use inherited order.
An empty array means its defined complete collection is empty, never “unknown”.
All field-table rows are required unless an explicit null/empty condition is
stated. A combined path notation denotes every named required field, not a
choice between fields. Only the complete listed fields are accepted.

## 8. Source/normalized provenance

Original datasets and complete normalized snapshots stay external. Never add
source bodies simply because identities are required. These are the exact
19 `sourceId` entries, where `.method` distinguishes a companion from its
parent source. Filename, format, size, hash and selector text are copied from
v0 §5; source selection is unchanged.

| sourceId | Inherited use | Captured filename | Format | Expected byteLength | Expected SHA-256 | Selectors / members authority |
| --- | --- | --- | --- | ---: | --- | --- |
| A1 | A1 | `D08.zip` | ZIP/XLSX | 501818 | `cf1612975f5743e6f2ff144f382f4076ca6589e8f0d015fff06dbde3be1f4e8f` | D08/C08_Tavole.xlsx:8.1 segue2,8.2,8.3,8.9,8.11 |
| A1.method | A1 method | `N08.pdf` | PDF | 94453 | `fb113fd9ddaeaf7dc2a6a4c82b7d2bb7b3468a03d5b2fbca6c6f5fad33418eb1` | ASIA occupation and LFS metadata |
| B1 | B1 | `D14.zip` | ZIP/XLSX | 331793 | `4fbba671bdb2427cff5c58e87d027f55bdf8af2c4878d6d6f14e4df08dfc037a` | D14/C14_Tavole.xlsx:14.1,14.2,14.8 |
| B1.method | B1 method | `N14.pdf` | PDF | 94170 | `4a9cf70051b9970471f1396becb42d13fc3e4335420b831e153f23e7f069e2bc` | ASIA and FRAME survey/register definitions |
| E1 | E1 | `D07.zip` | ZIP/XLSX | 1662813 | `681ab9a6e7a2fffb4ac88c7c259a8188dd718171a38366cf3ad95409b0e23f7c` | D07/C07_Tavole.xlsx:7.13,7.14 segue |
| E1.method | E1 method | `N07.pdf` | PDF | 93982 | `4dd6b0253458b507ad846c8194e3526033595cb0e62378682174c5e6762b3d6f` | LFS attainment methodology |
| H1 | H1 | `ind22_ascii.zip` | ZIP/CSV | 3566985 | `fd31bb4263363b9098de6f0d0640a90484dc30d0d4fd0c5323706d61dc9fb675` | CSV/carcom22.csv,CSV/rper22.csv |
| H1.method | H1 method | `Legen22.pdf` | PDF | 455568 | `d30ec3ee46cae84bd5c05304ab0dd1bb8324b5ec6b9d2a64915e4465b206d6e7` | Table2 income aggregation and allocation footnotes |
| M1 | M1 | `silc-guidelines.pdf` | PDF | 6299128 | `c5d746f3d02025dd21f0309d3ad5259d7c0a7f32714a2519e41932de58a85adf` | DocSILC065 personal/household income and PL032 definitions |
| O1 | O1 | `lfsa_eegais.json` | JSON-stat | 7975 | `ef00067100bdd392a0755ac1bd8fb8683bf5ffa448d004935e05070885a21d5f` | Dimensions in O1 ledger |
| O2 | O2 | `lfsa_esgais.json` | JSON-stat | 13478 | `81f1a2cf5906acec6a4f523ecdbe7375301307e5969f592380f8aeca6dc0ece6` | Dimensions in O2 ledger |
| O3 | O3 | `lfsa_egised.json` | JSON-stat | 5886 | `3d782ef806dd9eb822aae0748f75053965cf3b4b5f6af6d6f92588eb98119d27` | Dimensions in O3 ledger |
| R1 | R1 | `inps-pensionati2024.pdf` | PDF | 692528 | `7dbeb6e13c14accc942c9526587458aa162198aba6f0fac80af84a8b217a8a58` | Tavola1 total2024, recipient income classes |
| W1 | W1 | `SES2022.pdf` | PDF | 993677 | `0be4c2e576a40c3fe5a376947cdfdfa9e82e610da1b9ccca3a0bdff2a81e9616` | Report tables; glossary pp9–10 and methodological note pp11–15 |
| W2 | W2 | `inps-wages2024.pdf` | PDF | 658673 | `d8cf1e1e6bbfe380a4cc978ae424242eb0c57cd11cceeeac53b23bafdca8ec7e` | Prospetto1; methodology footnotes |
| X1 | X1 | `isco-cp2021.xlsx` | XLSX | 49763 | `1bdd98bcc5de620225afe073b72495b5900e20d387c569820756e7b0df0b4e2b` | `III isco08- Vcp2021` |
| X2 | X2 | `ateco2022.xlsx` | XLSX | 115953 | `7c284756dcb972cca3f5c3842d007678566fb9aeb1db09a670ca0a889427cfaf` | Struttura Ateco 2007 agg 2022 |
| X3 | X3 | `ateco-bridge2026.xlsx` | XLSX | 615444 | `faf4530a869f1e3d6b68f464520dcddd691f4565d17e92e3988370cf3c808673` | ATECO 2025 vs ATECO 2022;LEGENDA |
| X3.method | X3 method | `ateco-bridge2026-note.pdf` | PDF | 631907 | `e1700811a4a1049603aca9f3823995503f20291178c5f169f467156e129e5f0b` | Bidirectional relation methodology |

`originalSources` is sorted/unique by sourceId and contains exactly this set.
Its source fields are:

| Field path | Type | Required | Allowed values / meaning | Array order | Identity role |
| --- | --- | --- | --- | --- | --- |
| `originalSources[].sourceId` | string | YES | Exact registry ID above | sourceId lexicographic | External original identity |
| `.role` | string | YES | `DATA` for A1/B1/E1/O1/O2/O3; `COMPONENT-CALIBRATION` for W1/W2/R1; `COMPONENT-CROSS-CHECK` for H1; `CLASSIFICATION-AUDIT` for X1/X2/X3; `METHODOLOGY` for M1 and all .method entries | — | Fixed source-use role, not DIRECT-equivalence claim |
| `.capturedFilename`, `.format` | string | YES | Exact registry values, no directories in captured filename | — | Original snapshot metadata |
| `.byteLength`, `.sha256` | N, H | YES | Exact matched registry length/SHA | — | Original file-byte identity; not published artifact identity |
| `.ledgerRef` | Ref | YES | D2B1.v0 §5, exact Source ID table row; companion refers to exact companion-link label and parent row | — | Binds institution/dataset/version/reference/publication/access/unit/coverage/classification/variables/geography/access/evidence-class/model uses verbatim in pinned authority |
| `.officialOriginUrl` | string | YES | Exact target URL of the corresponding §5 source/companion link; M1 uses its source-row target; no mirrors/temporary paths | — | Official origin, no current access timestamp |
| `.selectorsRef` | Ref | YES | D2B1.v0 §5 captured-file row keyed by inherited use; extraction sections are additionally bound by provenance recipeRefs | — | Normative selectors/units; no free producer selector |
| `.members` | array | YES | Exactly B1: `D14/C14_Tavole.xlsx`; E1: `D07/C07_Tavole.xlsx`; A1: `D08/C08_Tavole.xlsx`; H1: `CSV/carcom22.csv`,`CSV/rper22.csv`; all others empty | member filename lexicographic, unique | Relevant decompressed member identities required by v0 §33 |
| `.members[].filename`, `.byteLength`, `.sha256` | string, N, H | YES | Exact member name and exact decompressed bytes' measured length/SHA, before extraction | as above | Member-byte identity, distinct from ZIP-container identity |

The reference mechanism deliberately avoids recopying the entire source ledger
as a second authority: all provenance fields are reconstructible from the
published pointer plus the pinned repository authority, with no scraping,
network access, inferred dates or mutable landing-page dependency. Referenced
metadata includes the original 2026-09-16 capture date; it is a fixed source fact,
not the future production date. An exact-byte archived replica is evidenced
by the same registry identity and official-origin URL, with no archive-host
metadata in canonical bytes.

`metadataSources` contains exactly one object:
`sourceId="X4"`, `standardIdentity="ISCED2011/UIS2012"`,
`ledgerRef` to D2B1.v0 §5 X4, `inspectedTextRef` to its §5 X4 paragraph,
`downloadRequired=false`. Those five fields are required, no SHA/byte length
is fabricated for a PDF that was not captured. This identity is provenance only.

`normalizedSources` contains exactly three states, sorted O1/O2/O3:

| Field path | Type | Required | Allowed value / semantic meaning | Order | Identity role |
| --- | --- | --- | --- | --- | --- |
| `normalizedSources[].sourceId` | string | YES | O1, O2, O3 | sourceId | Original-source dependency, resolved in originalSources |
| `.normalizationId` | string | YES | `SW-OFFICIAL-API-SNAPSHOT-v0` | — | Inherited identifier/version |
| `.normalizationRef` | Ref | YES | D2B1.v0 §33, `Exact API normalization` paragraph | — | Exact transformation, including flags/Cartesian coverage |
| `.representationId` | string | YES | `SW-OFFICIAL-API-SNAPSHOT-v0` | — | Canonical normalized object representation, not runtime encoding |
| `.byteLength`, `.sha256` | N, H | YES | O1 40694 / 04fc8c132e6603f0963a7c405964f3fddb07a1052bf0b82305e9d007778f719a; O2 137178 / 722403ed096ac15e25ccc46ed95c91f450a0a46be5ac9b17ca6e23c9eb0c0342; O3 26761 / b638a147165cd213fcc9c8463391b61ddab4b1289d235e65ef001f4cbf7b1d5a | — | Exact materialized temporary normalized bytes |
| `.repositoryBodyPublished` | bool | YES | false | — | Explicit external-body rule |

Materialize and hash these three external normalized bodies during future
production; deterministic reconstruction from original bytes and the published
producer, plus matched normalized length/hash, satisfies inherited snapshot
evidence. No normalized body is published in the repository. Non-API extraction
states are fit inputs, not invented normalized-snapshot families; their rules,
member identities, selected raw/completed aggregate counts and execution
evidence are captured by configuration, crosswalks and §12 tables. There is no
choice to normalize all 19 files under a new representation.

### 8.1 Canonical source-coordinate keys

Every `canonicalRowKey` is the string K of exactly one JSON array
`[kind,value1,...,valueN]`, with kind and all values JSON strings. Coordinate
names and their positions are fixed by the closed catalog below; names are
not repeated inside K. Encode this array using §7's compact `ensure_ascii=False`
JSON rules **without a terminal LF**. K's byte representation is its UTF-8
encoding, without BOM. When K is a field in C(evidence), §7 escapes that string
once as the containing JSON string; decode the field before using K's bytes.
No raw concatenation of coordinate values, second serialization or display
label is a key. All coordinate values must be present and nonempty; missing,
null, numeric, boolean and empty-string coordinates are invalid for every kind.

JSON quotes/backslashes use exactly `\"`/`\\`; control characters U+0008,
U+0009, U+000A, U+000C, U+000D use `\b`, `\t`, `\n`, `\f`, `\r`;
other U+0000..U+001F use lowercase four-digit `\u00xx`. All other Unicode
scalar values, including U+002F, U+007F, U+2028 and non-ASCII text, are emitted
literally in UTF-8. Surrogates are invalid. No optional slash, Unicode or other
escaping, spaces between tokens, LF suffix or alternate framing is permitted.
A decoder checks arity/domains and exact re-encoding equality, not just JSON
parseability. Unicode code points, case and whitespace in string coordinates
are preserved exactly: no NFC/NFD, case folding or generic trimming occurs.
Existing source-specific parsing remains upstream; this codec adds no normalization.

| Kind | Coordinate names, in exact order | Value domain | Canonical example K |
| --- | --- | --- | --- |
| `SHEET-CELL` | `sheet`, `cell` | Exact worksheet title from the inherited selected workbook for A1/B1/E1/X1/X2/X3; cell is uppercase A1 notation, column A..XFD and row 1..1048576, no `$` or leading zero. Use every participating source cell's own address, not a merged-range label or row index. | `["SHEET-CELL","14.8","B35"]` |
| `O1` | `age`, `freq`, `geo`, `isco08`, `sex`, `time`, `unit` | Exact named codes from the pinned O1 JSON-stat category indexes, including every Cartesian metadata coordinate when testing row order; no presentation labels. | `["O1","Y20-64","A","IT","OC1","T","2024","THS_PER"]` |
| `O2` | `age`, `freq`, `geo`, `isco08`, `sex`, `time`, `unit`, `wstatus` | Exact named codes from pinned O2 category indexes; `wstatus` is last, not source dimension-index order. Selected production uses SELF; order tests also cover other captured categories. | `["O2","Y20-64","A","IT","OC1","T","2024","THS_PER","SELF"]` |
| `O3` | `age`, `freq`, `geo`, `isced11`, `isco08`, `sex`, `time`, `unit` | Exact named codes from pinned O3 category indexes, including missing/status-only Cartesian coordinates. | `["O3","Y20-64","A","IT","ED0-2","OC1","T","2024","THS_PER"]` |
| `SHIW-PERSON` | `NQUEST`, `NORD` | Exact positive-integer CSV identifier text matching `[1-9][0-9]*`; no numeric reformatting. CARCOM column `nord` maps upstream to coordinate name NORD under v0 §20; RPER uses NORD. The same pair denotes the same person in both members. | `["SHIW-PERSON","1001","2"]` |
| `PINNED-DATUM` | `section`, `datum` | W1/W2/R1 source-backed publication quantities explicitly named in v0 §§20/25; exact closed datum catalog below. They have no API/XLSX or microdata row identity. | `["PINNED-DATUM","25","recipient gross annualized pension mean"]` |

For `PINNED-DATUM`, section/datum pairs are exactly W1 section `20` with
`mean gross annual full-year/full-time-equivalent remuneration`, `hourly mean`,
`lower decile boundary`, `upper decile boundary`, `finance hourly mean`,
`hospitality hourly mean`; W2 section `20` with `workers with≥1 paid day`,
`mean historical annual gross remuneration`, `paid days`; and R1 section `25`
with `recipient gross annualized pension mean`, `benefit mean`. These labels
identify the respective quantities in the pinned paragraphs; they do not add
sources, measurements or source-selection rules. Methodology companions/M1/X4
have no numerical row key. Code-state/relation keys in §9 remain their existing
structured tuples, not additional source-row kinds; a source cell participating
in those audits uses `SHEET-CELL`. A modeled completion uses the full O1/O2/O3
coordinate where its missing/raw observation belongs, not a fabricated row kind.

Compare/sort K lexicographically by **unsigned UTF-8 bytes**, shorter prefix
first. Source-audit compares sourceId first (existing lexicographic rule), then
K by these bytes. This fixes evidence key order, not normalized snapshot order:
§8's inherited SW-OFFICIAL-API-SNAPSHOT-v0 bytes/pins remain unchanged. Structural
array orders expressly fixed in §9 likewise remain unchanged. All source-audit
hashes, row-order/permutation hashes and evidence ordering using source-row
coordinate identity use this one encoding. No pretty/display representation
participates in deterministic identity. HASH-PERMUTATION uses exactly
UTF-8(`D2B1-roworder-v0:` + inherited sourceID + `:`) followed by K's bytes;
compare the raw 32-byte SHA-256 digests, ties by K's unsigned UTF-8 bytes.
REVERSE reverses the sequence after this canonical source-key ordering. This
only fixes the mechanically dependent source/permutation representation; all
logical sorting, accumulation and Run A/B semantics remain inherited.

Normative codec fixtures (synthetic worksheet names exercise the codec only,
not authorization to select new worksheets):

| Semantic coordinate values | Exact K / required result |
| --- | --- |
| SHEET-CELL sheet=`14.8`, cell=`B35` | `["SHEET-CELL","14.8","B35"]` |
| SHEET-CELL sheet=the three characters `a`, U+0022, `b`, cell=`A1` | `["SHEET-CELL","a\"b","A1"]` |
| SHEET-CELL sheet=the three characters `a`, U+005C, `b`, cell=`A1` | `["SHEET-CELL","a\\b","A1"]` |
| SHEET-CELL sheet=`a,b:[]`, cell=`A1` | `["SHEET-CELL","a,b:[]","A1"]`; delimiters remain data |
| SHEET-CELL sheet=` Südtirol `, cell=`A1` | `["SHEET-CELL"," Südtirol ","A1"]`; edge U+0020 preserved |
| SHEET-CELL sheet=U+0065 U+0301, cell=`A1` | `["SHEET-CELL","é","A1"]`; UTF-8 hex `5b2253484545542d43454c4c222c2265cc81222c224131225d`, distinct from U+00E9 |
| SHEET-CELL sheet=U+0009, cell=`A1` | `["SHEET-CELL","\t","A1"]`; backslash+t, not literal TAB in K |
| Any missing/empty coordinate, e.g. SHIW NORD=` ` or empty | Reject: empty is invalid; space is not positive-integer identifier text |
| SHIW NQUEST=`1001`, NORD=`2` | `["SHIW-PERSON","1001","2"]`; fixed two-coordinate order |
| X3 sheet=`ATECO 2025 vs ATECO 2022`, cell=`B2`, rawCode=` 46.89 `, canonicalCode=`46.89` | `["SHEET-CELL","ATECO 2025 vs ATECO 2022","B2"]`; v0 §6 removes edge U+0020 from the code upstream, preserves rawCode, and never trims the sheet/key |

Injectivity: JSON decoding yields one ordered string array. Its first element
selects exactly one arity/named-coordinate order; the remaining elements recover
every coordinate without delimiter ambiguity. Canonical JSON string escaping
and UTF-8 scalar encoding each have one spelling, so one semantic tuple has
one K, and different kind/value tuples cannot share K. No source-generated
numerical state is needed for this proof. Full pinned metadata/key domains can
be scanned independently of numerical production; future fitted values add
no new source-key kind or alternate coordinate representation.

## 9. Crosswalk publication

Exactly `D2B2-crosswalks-v3.json` carries complete expansions for these six
inherited versions, sorted by version ID:
`SW-ACT-LFS24-v0`, `SW-EDU-QUAL24-v0`, `SW-IND-ATECO22-v0`,
`SW-JF-ISCO08-MAJOR-v0`, `SW-JL-RESP-v0`, `SW-REG-IT-v0`.
No new crosswalk version, CP vintage bridge, ATECO2025 production allocation,
Region pool or runtime crosswalk RNG is invented.

| Field path | Type | Required | Allowed values / structure / meaning | Array order | Identity role |
| --- | --- | --- | --- | --- | --- |
| `schema` | string | YES | `SW-L2-D2B2-CROSSWALKS-v3` | — | New expansion-file schema only |
| `crosswalks` | array | YES | Exactly six objects with `version`,`entries` | version lexicographic, unique | Full required rule expansions |
| `crosswalks[].version` | string | YES | Six exact IDs above | — | Existing rule identity |
| `.entries` | array | YES | Entry object below; all §7 table rows retained, all specified nonzero vector coefficients expanded; no omitted required rule | (sourceCode,targetState with null first,context.ageBucket with null first,context.education with null first,context.population,ruleRef locator), unique | Inherited version→code→state primary ordering; contexts break ties |
| `.entries[].sourceCode` | string | YES | Exact code or conceptual source-category cell from §7 or its expressly referenced §13/14/18 expansion; comma-listed independent codes expanded individually; no ad hoc recoding | as above | Canonical rule input label |
| `.targetState` | string/null | YES | Frozen machine state/Region label; null only for no-assignment/rejected/calibration-only rules with no frozen output | — | No unknown public state |
| `.context.ageBucket`, `.context.education`, `.context.population` | N/null, N/null, string | YES | Bucket/state only where rule depends on it, otherwise null; population `ALL`,`COMPANY`,`EMPLOYEE`,`SELF-EMPLOYED`,`AUDIT` according to §7 rule's stated scope | — | Precisely exposes contextual split inputs |
| `.mappingType` | string | YES | `DIRECT`,`AGGREGATED`,`PROBABILISTIC-OFFLINE`,`CALIBRATION-ONLY`,`REJECTED`, exactly inherited table classification | — | Audit classification, not acceptance severity |
| `.coefficient` | Q/null | YES | Exact nonzero coefficient for numeric allocation; 1 for direct/aggregation allocation; null for qualitative/rejected/audit set-valued rules without numeric allocation | — | Never invent equal coefficients for X3 |
| `.ruleRef` | Ref | YES | D2B1.v0 §7 exact row; §13/14/18 reference added through `expansionRef` when table refers to contextual numeric rules | — | Bound rule semantics/ambiguity/resolution |
| `.expansionRef` | Ref | YES | Same §7 row when no referenced numeric expansion; otherwise precise referenced §13/14/18 paragraph/table | — | No producer-selected split |
| `audit` | object | YES | Exactly `codeCells`,`codeStates`,`relations`,`unmatched`,`regionLabels`,`examples` | — | Full inherited audit, not digest-only |
| `audit.codeCells` | array | YES | Objects `sourceId` (X2/X3), `sheet` (exact §6 sheet), `cell` (A1 notation), `rawCode` (exact text), `canonicalCode` (rawCode with edge U+0020 only removed), `rawTitle` (X2 title; null for X3) | sourceId,sheet,numeric row,column; unique cell | All participating nonblank code cells; preserves compatible repetitions/collisions and raw text |
| `audit.codeStates` | array | YES | Objects `vintage` (2022/2025 JSON integer), `code` canonical string, `states` array of frozen Industry labels | vintage,code; states in frozen Industry order, unique | Every audited code, ancestor unions/partner unions; 2025 unmatched has empty states |
| `audit.relations` | array | YES | Objects `code2025`,`code2022`,`coverage2025`,`coverage2022`,`hierarchy`,`states2025`,`states2022`,`addsState`,`partial`,`provenance` | Inherited exact relation key (code2025,code2022,coverage2025,coverage2022,hierarchy), lexicographic, unique | Full exact deduplicated matched X3 relation enumeration |
| `.relations[].coverage2025`, `.coverage2022`, `.hierarchy` | string | YES | Coverage `TOTALE`/`PARZIALE`; hierarchy exact validated source-cell text under v0 §6; states arrays frozen Industry order, unique; addsState/partial bool per §6 | — | Flags preserved, no economic coefficients |
| `.relations[].provenance` | array | YES | Objects `row` N (2..6746), `m` JSON integer 1/2/4, `updateAnnotation` exact source text or null for blank; all raw rows contributing to that relation | row,m,annotation(null first); unique | Do not lose update annotation or M=4 when exact relations deduplicate |
| `audit.unmatched` | array | YES | Objects `code2025` canonical string, `states` empty array, `provenance` same record shape but m=9 | code2025; provenance as above | All three inherited M=9 records, outside matched counts |
| `audit.regionLabels` | array | YES | Objects `sourceId` B1/E1/A1, `member` exact ZIP member, `sheet`,`cell`,`rawLabel`,`normalizedLabel`,`canonicalRegion`,`disposition` | sourceId,member,sheet,numeric row,column; unique cell | Every selected geography label, NFC/exact aliases; canonicalRegion null only for excluded national/macroarea/header; disposition `USED-PARENT`,`USED-CHILD`,`EXCLUDED-CHILD`,`EXCLUDED-NONREGION` |
| `audit.examples` | array | YES | Objects `ruleRef` Ref, `result` PASS, `targetStates` frozen-state array | Ref order; targetStates in associated taxonomy order, unique | Every functional/attainment/sector example explicitly named in v0 §§6–7, including athletes/coaches/fitness, repairs/veterinary/security; no new example universe |

Expansion is mechanical: retain every §7 rule, expand explicit positive vectors
into one entry per target; expand age-specific qualification vectors in §13,
and referenced Activity splits/transport in §14 by their authorized age and
Education context. Region code/alias rows expand each explicitly listed code;
macroarea constituent sets remain qualitative calibration entries, not equal
allocation probabilities. The §18 independent-person sector splits belong to
Industry entries with SELF-EMPLOYED context and their §18 expansionRef; they
must never reuse Company enterprise coefficients. Rejected/concept-only rows
remain reviewable with null coefficient/target as permitted above. Identical
entries arising from the same rule/context are deduplicated; conflicting
entries or unrecognized source codes fail E01 rather than guessing a state.

`crosswalkManifest` in §11 contains six objects, each with exactly `version`,
`recipeRefs` (v0 §§6/7 plus §§13/14/18 only as used above), `artifactRef`
(fixed expansion filename), `entryCount` N, and `expansionSha256` H.
`expansionSha256` hashes C(the corresponding complete `crosswalks[]` object),
not the whole file. Full audit is bound by the single file hash in artifacts;
this is not duplicate file-checksum authority. recipeRefs use shared Ref order;
manifest entries sort by version. The recipeRefs section sets are exactly
ACT:6/7/14, EDU:6/7/13, IND:6/7/18, JF:6/7, JL:6/7, REG:6/7,
all under D2B1.v0 with the corresponding exact section heading as locator.
Version prefixes here denote the six full IDs above, not new identifiers.

## 10. Producer/config publication

Option A is fixed: success publishes one complete, self-contained
`D2B2-producer-v3.py`, covering all extraction, non-Income fitting, rectified
Income generation, validation, row-order reproduction and assembly. No auxiliary
script/module/notebook is a repository artifact. Source bytes are UTF-8 without
BOM, LF line endings, exactly one terminal LF, no embedded timestamp/session/
PID/host/OS or absolute temporary path. CLI-supplied external paths may be used
at execution but are never emitted into deterministic evidence. The exact
file-byte SHA/length is recorded only in publication `artifacts`.

Canonical execution uses CPython 3.13.2 and its standard library as prescribed
by v0 §8; no third-party dependency or exploratory NumPy input is permitted.
Imported standard-library modules must be declared by sorted unique names in
`configuration.software.standardLibraryModules`; this array equals the actual
complete source import set (including modules invoked through importlib), all
resolved to that CPython release. Unknown/external modules fail the source
policy. Thus dependencies are a mechanically determined source fact, not a
producer-selected provenance format or unpinned package dependency. The script
may use the allowed equivalent exact integer/rational scaling mechanisms in
v3 §5.1; this contract does not redesign that algorithm.

Configuration is **not a separate file**. The complete versioned configuration
artifact is the schema-bearing object `evidence.configuration` inside the
single publication file. Its own exact bytes are C(that object); its identity
is SHA-256 of those bytes, derived when needed rather than redundantly stored.
This satisfies v0 §32:1205/:1231 and §33:1253, which require versioned exact
configuration, not a prescribed physical filename. There is no future option
to select a separate configuration file or embed config inside logical assets.

| Configuration field path | Type | Required | Allowed value / meaning | Array order | Identity role |
| --- | --- | --- | --- | --- | --- |
| `schema` | string | YES | `SW-L2-D2B2-CONFIG-v3` | — | Exact embedded config schema |
| `mode` | string | YES | `FRESH` | — | No historical generated-asset reuse |
| `profile` | string | YES | `FAST-DEMO-v3-CANONICAL-ARITHMETIC` | — | Effective inherited generator profile |
| `authorityRefs` | array of string | YES | All eleven authority IDs of §1, resolving to their exact SHA references in evidence.authorities | lexicographic, unique | v0/v1/v2/v3 reviewed bytes included, not just version names |
| `sourceIds` | array of string | YES | Exactly nineteen IDs of §8 | lexicographic, unique | Inputs fixed by inherited registry |
| `producerRef`, `crosswalkArtifactRef` | string | YES | Fixed producer / expanded-crosswalk filenames of §5 | — | Machine artifact entry references, not host paths |
| `software.implementation`, `.version` | string | YES | `CPython`, `3.13.2` | — | Normative software requirement |
| `software.standardLibraryModules` | array of string | YES | Exact complete module names imported by producer; no duplicates, no third-party distribution | lexicographic, unique | Dependency declarations versioned with producer/config |
| `arithmeticRef` | Ref | YES | D2B1.v3 §5.1, `Canonical Income arithmetic` | — | Ordered Decimal96 Q then exact finite scaling/ceiling |
| `decimal.precision`, `.rounding`, `.Emin`, `.Emax` | integer,string,integer,integer | YES | 96, `ROUND_HALF_EVEN`, -999999999, 999999999 | — | Required environment facts, not new model constants |
| `generatorRefs` | array of Ref | YES | D2B1.v0 §§3,6–19,20–26,31–33, using section headings as locator; D2B1.v2 §§2–8; D2B1.v3 §§2–5.1 | Ref order, unique | Effective production rules with §1 precedence |

Actual model constants are inherited by reference, not copied into config as
editable parameters. Decimal context numbers/software version are repeated
only because they are explicitly required execution facts. No override, seed,
custom cap, source selector, arbitrary extension key or runtime setting exists.
Values appearing in producer code must implement/check the cited authority,
never silently override it. Configuration cannot authorize a model revision.

## 11. Evidence manifest schema

The **evidence manifest is embedded**, exactly at publication field `evidence`;
there is no physical evidence-manifest filename. Its independent canonical byte
domain is C(evidence); §14 binds that exact complete object with a digest.
No evidence/config object references/hashes the final publication manifest.
Filename references below identify already-final leaf artifacts; resolving them
against the containing artifacts list is not a hash of that containing list.

| Field path (relative to evidence) | Type | Required | Allowed structure / semantic meaning | Array ordering | Identity role |
| --- | --- | --- | --- | --- | --- |
| `schema` | string | YES | `SW-L2-D2B2-EVIDENCE-v3` | — | Embedded evidence schema |
| `authorities` | array | YES | Exactly eleven `{id,filename,sha256}` objects; IDs/filenames §1; H values; ten pinned hashes must match, contract SHA external | id lexicographic, unique | Sole authority identity namespace |
| `originalSources`, `normalizedSources`, `metadataSources` | arrays | YES | Complete §8 objects | §8 | Sole original/normalized source identity namespaces |
| `crosswalkManifest` | array | YES | Six exact §9 entries | version lexicographic | Rule expansion binding |
| `producerRef` | string | YES | `D2B2-producer-v3.py` | — | Resolves to sole file-checksum entry |
| `configuration` | object | YES | Complete §10 schema | §10 | Required versioned configuration bytes by containment |
| `provenance` | array | YES | One object per §4 audit model, `{model,sourceIds,crosswalkVersions,recipeRefs,evidenceClass}` | model lexicographic; sourceIds/crosswalkVersions/recipeRefs shared order | Per-model transformation/reconciliation references |
| `provenance[].evidenceClass` | string | YES | `MODELED-ASSUMPTION` for every generated asset; this never upgrades individual sources to direct all-source equivalence | — | Explicit modeling status |
| `checks` | array | YES | Exactly 21 CheckSet objects, §12 | effective matrix order E01–E05,S01–S02,V01–V08,W01–W02,D01–D04 | All applicable execution and coverage evidence |
| `acceptance` | array | YES | Exactly 21 acceptance-row objects, §12 | same effective matrix order | Sole current acceptance result representation |
| `resources` | object | YES | Complete §13 object | §13 | Resource manifest/prospective proof, not RSS |
| `reproducibility` | object | YES | Complete §15 object | §15 | Sole A/B/row-order evidence namespace |

Provenance source/crosswalk/recipe sets are mechanically the dependencies in
v0 §29's corresponding model card, with v2 §§2–8 replacing Income algorithms
and v3 §§2–5.1 overriding acceptance/arithmetic. Age entries use their defining
v0 model section; companyCount uses v0 §10; no Income G/H/Ward dependency is
retained. All provenance sourceIds resolve in originalSources or metadataSources (thus
X4 is allowed only through its metadata identity); unknown IDs are forbidden.
Source companion IDs accompany each used parent source. All assets
retain v0 §§3/8/9/32 references as applicable. Crosswalk dependencies are the
exact named versions used by that card, not every version simply because it
is in the same file. This preserves independent node/branch auditability.
Income concept reconciliation is additionally carried by each branch's v0
§20–26, Contract5 §§3/5/15 and v3 §3 recipeRefs. Those pinned complete rules
satisfy the required versioned unit/gross/business/capital/time/annualization/
nominal/zero/domain reconciliation without inventing conversion parameters.

## 12. Acceptance evidence

No number/result in this proposed contract is a production measurement.
The effective matrix is exactly 21 rows: 14 HARD, 3 WARNING, 4 DIAGNOSTIC.
HARD IDs E01–E05/S01–S02/V01–V06/V08; WARNING V07/W01/W02;
DIAGNOSTIC D01–D04. S02 remains HARD; historical monetary caps are D02 only.

| Acceptance field path | Type | Required | Allowed value / meaning | Array ordering | Identity role |
| --- | --- | --- | --- | --- | --- |
| `acceptance[].id` | string | YES | Exact effective ID above | fixed matrix order, unique | No additional gate row |
| `.severity` | string | YES | Exact severity of that ID | — | Inherited policy, immutable here |
| `.owner`, `.phase` | string | YES | `D2-B2`, `PRODUCTION-VALIDATION` | — | D2-B2-owned portion, not runtime certification |
| `.result` | string | YES | Vocabulary PASS/FAIL/WARNING/DIAGNOSTIC/DEFERRED-BY-NORMATIVE-OWNER; in successful artifact HARD=PASS, WARNING=WARNING, DIAGNOSTIC=DIAGNOSTIC | — | WARNING even with zero exceedances is never called PASS |
| `.summary.checkCount`, `.findingCount`, `.unexecutedCount` | N | YES | Counters from corresponding CheckSet; findingCount counts disclosed warning/diagnostic findings, not a disguised PASS | — | No prose-dependent summary bytes |
| `.evidenceRefs` | array of string | YES | Fixed JSON pointers: every row `/evidence/checks/` plus its zero-based fixed row index; E01 also originalSources/normalizedSources/crosswalkManifest; E04 also reproducibility; E05 also resources | lexicographic, unique | Traceable complete evidence |
| `.laterPhase` | array | YES | Exact LaterPortion objects below; empty for every other ID | owner,phase lexicographic, unique | DEFERRED is visibly separate from production PASS |
| `laterPhase[].owner`, `.phase`, `.result`, `.authorityRef` | string,string,string,Ref | YES | E04: owner `Protocol/Implementation`, phase `RUNTIME-ISOLATION`; E05: owner `Implementation`, phase `ACTUAL-CONSTRUCTOR-ALLOCATIONS`; S01: owner `Implementation`, phase `ACTUAL-PAYLOAD-AND-SELECTION`; all result `DEFERRED-BY-NORMATIVE-OWNER`, authorityRef D2B1.v3 §5 paragraph `This focused acceptance replay` | above | Only already-deferred runtime portions; no D2-B2 HARD portion moved |

Warnings/diagnostics with unavailable optional expensive work disclose coverage;
required computations, including all V07 law checks and every cap comparison,
still execute. A missing applicable HARD result cannot be encoded DEFERRED or
PASS and prevents success publication. An OPEN ISSUE goes in the failure report.

Exact bounded machine tables avoid arbitrary evidence-key conventions.
Every CheckSet has exactly `{id,tables,detail,checkCount,findingCount,unexecutedCount}`.
`id` is its effective row ID. `tables` contains exactly that row's catalog tables
below in table-ID lexicographic order. Each table has exactly `{id,coverage,rows}`;
coverage is `EXECUTED`,`REUSED`,`UNEXECUTED` or `SUPERSEDED`. Required full
production tables are EXECUTED; available diagnostic tables may be REUSED;
UNEXECUTED/SUPERSEDED has empty rows and a coverage row identifying the scope.
Each row object contains exactly the named catalog columns, with shared types.
No arbitrary extension columns/keys. Table unique keys and sort rules follow
its last column. Every required finite grid is complete, including zero cells.

| CheckSet/Table field path | Type | Required | Allowed values / semantic meaning | Array order | Identity role |
| --- | --- | --- | --- | --- | --- |
| `checks[].id` | string | YES | Exact effective21-row ID; must equal its corresponding acceptance ID | fixed matrix order | Evidence-family binding |
| `.tables` | array of Table | YES | Exactly that ID's complete closed catalog table set | table id lexicographic, unique | Required body/coverage placement |
| `.detail` | Detail/null | YES | Detail only for V05; null for every other ID | — | Full temporary pair-check digest, never required-versioned-body replacement |
| `.checkCount`, `.findingCount`, `.unexecutedCount` | N | YES | Exact counters from the mechanical rules below; must equal acceptance summary | — | Deterministic execution summary |
| `.tables[].id` | string | YES | Exact table ID in that CheckSet's catalog | above | No new table family choice |
| `.tables[].coverage` | string | YES | EXECUTED for mandatory full tables; optional available diagnostic table EXECUTED/REUSED; optional absent table UNEXECUTED/SUPERSEDED | — | Availability is separate from PASS |
| `.tables[].rows` | array of closed row objects | YES | Exact catalog fields/types/grid; empty only for explicitly optional UNEXECUTED/SUPERSEDED tables or an actually empty defined domain | each catalog's key order, unique | Full published required evidence |

`detail` is null except V05; V05 has the required Detail object hashing the
complete temporary pair-check table, not a published file. Each temporary
body is C({"schema":"SW-L2-D2B2-DETAIL-v3","id":"V05","records":records}),
where records have the exact V05-detail columns below and canonical row order.
The temporary basename is `D2B2-V05-detail-v3.json`; no directory is recorded.
V05 published summaries retain every branch/quantile worst witness and totals;
full pair enumeration is executed but not a repository artifact. No required
versioned table is replaced by a temporary digest.

Column notation: `B` bool; `S` string with its stated domain; `I` signed JSON
integer; `C` N-array of inherited coordinate indices; `M` §4 model name;
`R` Ref; `Q?`/`D?` mean null only under the catalog's explicit condition.
`status` S/null is a raw official quality flag, null only if absent, never dropped.
`law` N is branch-local canonical law ID; `branch` uses e,s,u,t,r,i.
Numeric quantities are exact, never formatted MiB/percentage display strings.

| Check owner / table ID | Exact row columns (type) and normative coverage/content | Unique key / row array order |
| --- | --- | --- |
| E01 / `source-audit` | sourceId S; canonicalRowKey S (exact §8.1 canonical source-coordinate key); raw Q?; completed Q?; status S/null; classification S (`OBSERVED`,`SOURCE-INTERNAL-COMPLETION`,`MODELED-RESIDUAL`,`ABSENT-MODELED-ZERO`,`EXCLUDED`); result S PASS. Full selected aggregate totals/majors/count anomalies/quality flags required by v0 §8, including raw/completed missing categories and rounding discrepancies; null raw only missing/excluded, null completed only excluded. No SHIW person rows or complete source body is published. | sourceId,canonicalRowKey; unique |
| E01 / `input-gates` | sourceId S; hashMatch B; lengthMatch B; selectorMatch B; unitsMatch B; parserPass B; result S PASS. Exactly 19 rows; complete Region/ATECO/X2/X3 audit remains in expanded crosswalk file. | sourceId |
| E01 / `h1-join-audit` | carcomUniqueKeyCount N; rperUniqueKeyCount N; matchedRperKeyCount N; duplicateCarcomKeyCount N; duplicateRperKeyCount N; missingRequiredKeyCount N; ageAndWeightValid B; joinRule S `RPER-KEY-INCLUSION-IN-CARCOM`; result S PASS. One complete input-audit summary; required RPER keys must be included in CARCOM, not equal its whole key domain. No person-key/microdata publication. | single row |
| E02 / `fit-stages` | model M; stage N; coordinates C; iterationCount N; maxAbsResidual D?; maxRelResidual D?; normalized B; finiteNonnegative B; feasible B; result S PASS. All production steps: 0 source/authority input, 1 constructed preinteger target/fit, 2 integer output, 3 canonical output; all four per weighted model, 0/1/3 per age map/companyCount/Employer factors. Per fitted row/mean-tilt band/cap candidate where applicable; residuals null only for closed-form or non-margin steps. No G/H/Ward stage. | model,stage,coordinates; compare C lexicographically numerically |
| E02 / `d0-margins` | projection S; coordinates C; mass Q; classification S MODELED-ASSUMPTION. Full D0 singleton margins for age/Region/Education/Activity/family/level/Industry/SizeBand and all two-axis parent/output margins named in v0 §§13–19; projection is exact ordered axis names joined by `>` (axis vocabulary below). No unauthorized runtime input follows from audit axes. | projection,coordinates |
| E02 / `activity-margins` | stage S RAW/PRINCIPAL; axis S AgeBucket/Education/Region; coordinates C (axis state,Activity state); mass Q; classification S MODELED-ASSUMPTION. Full raw and reconciled margins. | stage (RAW then PRINCIPAL),axis lexicographic,coordinates |
| E02 / `activity-flows` | ageBucket N; education N; region N; step N; fromActivity N; toActivity N; mass Q. Complete seven flow kinds: step0 s→i helper; step1 e/s/u→t student; step2 e/s/u→r from remainder. All 8×7×20 contexts and all seven kinds, including zero masses; transition bounds/precedence are inherited. | ageBucket,education,region,step,fromActivity,toActivity |
| E02 / `income-construction` | branch S; qEvaluations N; cachedPointCount N; distinctScalarCount N; bucketCount N; tupleSlots N; reachableSlots N; sentinelSlots N; noD0ConstructionInput B; orderedQArithmetic B; exactScaling B; result S PASS. Six complete branch rows; exactly384 cached/Q evaluations per branch and2304 in aggregate; direct construction only, not per-law Q reevaluation. | branch order |
| E02 / `income-buckets` | branch S; bucket N; startIndex N; endIndex N; representativeIndex N; representative Q; law N; authoredPositiveCells N; result S PASS. Every authored scalar bucket; exact equal-cardinality boundaries/lower-median/reachable representative, branch-local canonical-law mapping and384 positive authored cells. | branch order,bucket |
| E02 / `employer-cap-profile` | cap N; iterations N; objective D; unpenalizedObjective D; bandMultipliers D-array length4; maxResidual D; selected B; unpenalizedSelected B. Full fixed eleven-candidate grid, no new profile. | cap ascending |
| E02 / `employer-targets` | family N; level N; industry N; positiveSizeBand N (1..4); mass Q; classification S MODELED-ASSUMPTION. All40 normalized analytic relationship tables from prescribed listed factors/Company laws (2240 cells). | family,level,industry,positiveSizeBand |
| E03 / `integerization` | model M; coordinates C; tv Q; maxEntryError Q; meanChange Q?; positivePreserved B; uint32Checked B; uint64Checked B; canonicalPass B; result S PASS. Full categorical rows/count laws/authored Income bucket laws; meanChange null only categorical rows. Counts/sorting/GCD/dedup/IDs/schema/positive and zero preservation are checked, Employer factors never normalized. | model,coordinates |
| E04 / `branch-isolation` | model M; subtreeOnly B; foreignBranchInputsAbsent B; result S PASS. All24 audit models, asset identity material only. | model |
| E05 / `allocation-guards` | phase S (fixed §13 phases); componentsBounded B; sumsBounded B; beforeAllocation B; result S PASS. Prospective proof/producer guards, not runtime instrumentation. | phase in §13 order |
| S01 / `structure` | model M; structuralTupleCount N; coveredTupleCount N; authorizedParentsOnly B; frozenDomains B; gatesConsistent B; positiveEligibleSupport B; result S PASS. Each model, complete structural coverage; Employer all40 conditions; Income unreachable employee slice accounted separately. | model |
| S02 / `income-semantics` | branch S; law N; exactZero Q; finalZero Q; zeroError Q; minimumPositive N; maximum N; preCeilingChecked B; personalConceptRef R; result S PASS. Every canonical law, with exact zero-mass provenance and domain/parent/ceiling checks. | branch order,law |
| V01 / `company-breadth` | industryPositiveCount N; bandPositiveCount N; distinctPositiveCounts N; result S PASS. One analytic model-wide row, no sampled Company roster. | single row |
| V02 / `category-breadth` | model M; rowCount N; intendedPositiveCount N; preservedPositiveCount N; multiStateRowCount N; collapsedRowCount N; distinctCanonicalRows N; result S PASS. Six categorical models only. Activity branch positivity additionally in d0-margins. | model |
| V03 / `categorical-contrasts` | contrast N (0..9 in v1 §3 bullet expansion order); target Q; final Q; result S PASS. Exactly ten fixed standardized sign contrasts, zero target exempt. | contrast |
| V04 / `branch-scales` | branch S; referenceMean Q; finalMean Q; referencePositiveMedian Q; finalPositiveMedian N; meanRatio Q; medianRatio Q; result S PASS. Six rows, final mixtures formed before quantiles. | branch order |
| V04 / `branch-ranks` | firstBranch S; secondBranch S; referenceDifference Q; finalDifference I; result S PASS. All15 ordered branch pairs, equal reference exempt. | branch-pair order from e,s,u,t,r,i |
| V05 / `profile-order-summary` | branch S; quantile N (10,25,50,75,90,95,99); pairCount N; failureCount N; worstFirstBucket N; worstSecondBucket N; firstLaw N; secondLaw N; reversal I; targetLarger Q; limit Q; excess Q; result S PASS. One worst (largest reversal-minus-limit) per branch/quantile; tie by earliest ascending bucket pair. bucketCount=1 gives no pair, then witness columns are null, counts0; witness integer/Q null only then. | branch order,quantile ascending |
| V05-detail (temporary only) | branch S; firstBucket N; secondBucket N; quantile N; firstLaw N; secondLaw N; firstQuantile N; secondQuantile N; targetLarger Q; limit Q; reversal I; excess Q; result S PASS/FAIL. Every distinct scalar-ordered bucket pair and seven quantiles; equality allowed. | branch order,firstBucket,secondBucket,quantile |
| V06 / `positive-atoms` | branch S; law N/null (null is branch mixture); value N; mass Q; limit Q; result S PASS. Full law/mixture maxima, tie lowest positive support, zero exempt. | branch order,law with mixture after laws |
| V07 / `central-spacing` | branch S; law N; buckets N-array; rho Q; topDecile B; positiveMedian N; gap N; left N; right N; limit Q; exceeds B; result S WARNING. Full per-law maximum inclusive q10–q90 adjacent-gap check; tie lowest left support; canonical law mapped to all authored buckets. Retain every affected law and its worst support-pair witness; overall worst is largest gap-minus-limit, ties branch/earliest tuple. | branch order,law; buckets ascending unique |
| V08 / `central-occupancy` | branch S; binLower N; targetMass Q; candidateMass Q; required B; occupied B; result S PASS. Full internal half-open1000-EUR bins from original positive Q q10–q90; target threshold unchanged. | branch order,binLower |
| W01 / `source-boundary-coverage` | scope S (`COARSE-SOURCES` or exact eight mapping names); coverage S EXECUTED/REUSED/UNEXECUTED; findingCount N; diagnosticRefs S-array of corresponding diagnostic-row JSON pointers. Exactly9 rows, no missing-map omission. | scope lexicographic; diagnosticRefs lexicographic unique |
| W02 / `company-risks` | event S (`INDUSTRY`,`SIZE-BAND`,`INDUSTRY-SIZE-BAND`,`COUNT-GE1000`,`COUNT-GE10000`,`COUNT-GE65536`,`ALL-ZERO`); coordinates C; probability Q; expectedCount Q; variance Q; absence Power; warning B. Full14 Industry, five bands,70 intersections, three n≥1000/10000/65536 events and ALL-ZERO event. Tail warnings >.05 are unchanged. Probability is zero probability for ALL-ZERO, with exponent N; all others absence base1−probability. | event lexicographic,coordinates |
| W02 / `factor-pressure` | component S C/I/S; actualMaximum N; architecturalMaximum N; ratio Q; boundaryPressure B; result S WARNING. Three rows, cap-profile unpenalized preference retained in E02 table. | C,I,S |
| D01 / `nonincome-diagnostics` | diagnosticId S (inherited v0 §30 matrix ID); subject Subject; metric S (fixed diagnostic vocabulary below); observed Q/D; reference Q/D/null; oldLimit Q/D/null; coverage S EXECUTED/REUSED; finding B; result S DIAGNOSTIC. Retain all available fitted-cell/TV/quantile/mean/tail/exposure/DAG residuals; null reference/limit only when inherited metric has none. | diagnosticId,subject,metric; no duplicates |
| D02 / `historical-cap-comparisons` | branch S; law N; buckets N-array; actualMaximum N; historicalCap N; representationBound N; exceeds B; result S DIAGNOSTIC. Every law's unchanged cap comparison; all exceeded records remain versioned, no temporary digest substitute. | branch order,law; buckets ascending unique |
| D02 / `income-diagnostics` | Same diagnostic row schema as nonincome-diagnostics; available v0 N04/P/F/A/M dimensions only, no mandatory superseded computation. | same key |
| D02 / `h1-component-diagnostics` | component S YL1/YTP1/YM/YTA2; observedWeightShare Q; zeroShare Q; negativeShare Q; positiveShare Q; signedMean Q; positiveQ10 Q?; positiveQ50 Q?; positiveQ90 Q?; positiveQ99 Q?; result S DIAGNOSTIC. Preserve available exact v0 §8 historical component measurements, signed values never clamped/absoluted or relabeled personal all-source targets. Exactly four rows when available; positive quantiles null only when that component has no observed positive weighted value. Otherwise table is explicitly UNEXECUTED with empty rows under effective optional diagnostic coverage. | component lexicographic, unique |
| D03 / `reuse-tail-diagnostics` | Same diagnostic row schema; available collision/multiplicity/distinct/spacing/tail/finite-universe/delta-method and smaller V05 reversal evidence. | same key |
| D04 / `sensitivity-coverage` | scenario S (closed coverage inventory below); coverage S EXECUTED/REUSED/UNEXECUTED/SUPERSEDED; diagnosticRefs S-array of diagnostic-row JSON pointers. Every scenario family identified, no required perturbation/refit PASS. | scenario lexicographic; refs lexicographic unique |
| D04 / `sensitivity-diagnostics` | Same diagnostic row schema plus required scenario S (§12.1); available fixed-scenario/all-eight-map evidence only. | diagnosticId,subject,scenario,metric; no duplicates; scenario compared by unsigned UTF-8 bytes |

For company-risks, coordinates are [industry] for INDUSTRY, [band] for
SIZE-BAND, [industry,band] for INDUSTRY-SIZE-BAND, and [] for the three
COUNT events/ALL-ZERO. No other event string is allowed.

The D0 axis vocabulary is `Age`, `Region`, `Education`, `PrimaryActivity`,
`Employee.JobFamily`, `Employee.JobLevel`, `SelfEmployed.JobFamily`,
`SelfEmployed.Industry`, `Company.Industry`, `Company.SizeBand`. Include every
singleton; two-axis inventory is exactly Education>Age,Education>Region,
PrimaryActivity>Age,PrimaryActivity>Education,PrimaryActivity>Region,
Employee.JobFamily>Education,Employee.JobFamily>Region,
Employee.JobLevel>Age,Employee.JobLevel>Education,Employee.JobLevel>Employee.JobFamily,
SelfEmployed.JobFamily>Education,SelfEmployed.JobFamily>Region,
SelfEmployed.Industry>SelfEmployed.JobFamily,Company.Industry>Company.SizeBand.
Coordinates use the listed axis order, actual integer age21..75 for Age,
and other frozen zero-based states. Occupational margins are branch-conditional;
Activity margins include branch masses, all labeling remains modeled.

Diagnostic metric vocabulary is exactly `TV`,`MEAN`,`Q10`,`Q25`,`Q50`,`Q75`,
`Q90`,`Q95`,`Q99`,`Q999`,`TAIL20K`,`TAIL50K`,`TAIL100K`,`TAIL250K`,`TAIL1M`,
`NW1`,`IQR`,`SD`,`COLLISION`,`SUPPORT-MULTIPLICITY`,`DISTINCT-VALUES`,
`TAIL-GAP`,`BOUNDARY-TV`,`MEDIAN-CHANGE`,`COARSE-MARGIN`,`EXPOSURE`,
`DAG-RESIDUAL`,`RATIO-VARIANCE`,`STANDARD-ERROR`,`INTERVAL-LOW`,`INTERVAL-HIGH`,
`SMALLER-REVERSAL`. DiagnosticId must be an exact row ID in the pinned v0 §30
matrix; no producer-invented family. Observed/reference/limit use Q when exact
rational, D when a final canonical Decimal96 value of the inherited computation;
representation is determined by that arithmetic, not an arbitrary type choice.
Unexecuted expensive scenarios have no metric value and are never called PASS.

Sensitivity coverage inventory is exactly the eight strings `AGE-MAP:` plus
each exact mapping name, plus `COMPANY-COUNT`,`COMPANY-AUGMENTATION`,
`HEADCOUNT-ZERO`,`STOCK-MEAN`,`HEADCOUNT-CAP`,`HEADCOUNT-TAIL`,
`EDUCATION-SPLIT`,`REGION-SHRINK`,`ACTIVITY-EDUCATION`,`ACTIVITY-OVERLAP`,
`ACTIVITY-HELPER`,`JOBLEVEL-PRIOR`,`OCCUPATION-SPLIT`,`SE-BORROWING`,
`SE-INDUSTRY-SPLIT`,`EMPLOYER-PENALTY`,`EMPLOYER-I`,`INCOME-LEVEL`,
`INCOME-ZERO`,`INCOME-TAIL`,`LAYER1-PARENTS`,`RARE-SUPPORT`,
`CONSTRUCTION-SEED-RISK`. Fine-fit/superseded Income scenarios may be explicitly
SUPERSEDED under v2; the required architectural record remains. Runtime sample
construction/selection remains deferred, never invented to fill this table.

### 12.1 Sensitivity diagnostic scenario identity

Only D04 `sensitivity-diagnostics` adds the required string column `scenario`.
Every other diagnostic table retains its exact existing schema/key and has
implicit baseline identity; scenario columns there are forbidden. All available
fixed-scenario or eight-map result rows belong in D04, with their originating
scenario retained. This adds no execution requirement, perturbation, acceptance
gate or new experiment. D04's `sensitivity-coverage.scenario` remains its existing
family token, not the row's detailed scenario identity.

A detailed scenario is the compact canonical JSON **string of an array of
strings**, no terminal LF, with precisely §8.1's string/UTF-8/escaping rules.
Only the tuples in this table are legal. Integer strings use `0` or `[1-9][0-9]*`. Magnitudes use exact §7 D
strings, including leading `0.`; a direction is encoded by the actual substituted
value/multiplier, not a producer-selected label. AGE-MAP instead uses exactly
`-1` or `+1`, including the plus sign. No omitted/null/empty scenario or open
arbitrary-string namespace is legal. A parsed but differently escaped/spaced
array is rejected by canonical re-encoding equality.

| Family / exact tuple shape | Complete allowed dimensions / values | Pinned obligation |
| --- | --- | --- |
| `["BASELINE"]` | Unperturbed production/reference result only | Existing available diagnostic baseline |
| `["AGE-MAP",map,boundary,delta]` | map is one of the exact eight §12 coverage mapping names; boundary is one internal lower bound of that map in v0 §20's eight-map table; delta `-1`/`+1` | v0 §§13/14/16/20/30 |
| `["COMPANY-COUNT",n]` | n `32768`/`98304` | v0 §30 |
| `["COMPANY-AUGMENTATION",sector,k]` | sector `A`/`K`/`O`/`P`/`Q`; k `0.75`/`1.25` | v0 §§11/30 |
| `["HEADCOUNT-ZERO",z]` | z `0.015`/`0.045` | v0 §30 |
| `["STOCK-MEAN",k]` | k `0.9`/`1.1` | v0 §30 |
| `["HEADCOUNT-CAP",cap]` | cap `65536`/`262144` | v0 §30 |
| `["HEADCOUNT-TAIL",k]` | k `0.5`/`1.5`; both upper strata changed together | v0 §30 |
| `["EDUCATION-SPLIT",bucket,state,k]` | bucket `0`..`5` in Education map order; state `below-primary`/`primary`/`post-secondary-non-tertiary`/`doctoral`; k `0.75`/`1.25` | v0 §§13/30 |
| `["REGION-SHRINK",model,exponent]` | model `education`/`primaryActivity`/`employeeJobFamily`/`selfEmployedJobFamily`; exponent `0.25`/`0.75` | v0 §30 |
| `["ACTIVITY-EDUCATION",exponent]` | exponent `0.25`/`0.75` | v0 §30 |
| `["ACTIVITY-OVERLAP",k]` | k `0.5`/`1.5`; all four vectors together | v0 §30 |
| `["ACTIVITY-HELPER",fraction]` | fraction `0.01`/`0.03` | v0 §30 |
| `["JOBLEVEL-PRIOR",tier,k]` | tier `1`/`2`/`3`, frozen non-IC state index; k `0.75`/`1.25` | v0 §30 |
| `["OCCUPATION-SPLIT",major,family,k]` | One ambiguous major OC1..OC9 and one strictly positive family coefficient of that exact v0 §7 vector; family frozen index `0`..`9`; k `0.8`/`1.2`. NRP's modeled uniform completion and direct OC0 are not ambiguous-major scenarios. The shared vector is one dimension, not separate O1/O2/O3 variants. | v0 §§7/30 |
| `["SE-BORROWING",exponent]` | exponent `0.25`/`0.75` | v0 §30 |
| `["SE-INDUSTRY-SPLIT",vector,entry,k]` | vector `COMMERCE-HOSPITALITY` (entries `0`/`1`), `G` (entries `0`/`1`), or `OTHER-SERVICES` (entries `0`..`8`); entry is the positive coefficient's index in v0 §18's stated vector order; k `0.8`/`1.2` | v0 §§18/30 |
| `["EMPLOYER-PENALTY",penalty]` | penalty `0.005`/`0.02` | v0 §30 |
| `["EMPLOYER-I",family,k]` | family frozen index `0`..`9`; k `0.75`/`1.25`; all maximum I cells of that family changed together | v0 §30 |
| `["EMPLOYER-I","DAG-VS-INDEPENDENCE"]` | The one prescribed actual-DAG versus independence-design comparison | v0 §§19/30 EMP05 |
| `["INCOME-LEVEL",branch,k]` | e/r k `0.9`/`1.1`; s/u k `0.85`/`1.15`; t/i k `0.8`/`1.2` | v0 §30; availability/supersession still v2/v3 |
| `["INCOME-ZERO",branch,z]` | e `0.001`/`0.004`; s `0.04`/`0.12`; u `0.25`/`0.45`; t `0.55`/`0.75`; r `0.005`/`0.02`; i `0.45`/`0.65` | v0 §30; availability/supersession still v2/v3 |
| `["INCOME-TAIL",branch,k]` | branch e/s/u/t/r/i; k `0.5`/`1.5`; q95 excess stretched together at .99/.999/1 | v0 §30; availability/supersession still v2/v3 |
| `["LAYER1-PARENTS"]`, `["RARE-SUPPORT"]`, `["CONSTRUCTION-SEED-RISK"]` | One existing fixed architectural comparison/scope per token, no newly defined perturbation | Existing §12 coverage inventory / v1 §3 |

D04 uniqueness is exactly `(diagnosticId, subject, scenario, metric)`.
Compare diagnosticId by the existing lexicographic rule, subject by model
string Unicode code-point lexicographic order (including `all`) then numeric
coordinate tuple order, scenario by unsigned UTF-8 bytes (shorter
prefix first), then metric by the existing lexicographic rule. Duplicate complete
tuples are rejected, never overwritten; distinct scenarios are never merged.
The old key is a projection of this key, so no previously distinct row can merge.
One available result for one subject/metric/scenario has exactly one row. Multiple
metrics/subjects from the same scenario retain their existing dimensions. Ref
pointers are built only after this sorting; coverage families point to every
available member scenario result. AGE-MAP tuples resolve to `AGE-MAP:`+map;
BASELINE tuples have no sensitivity-coverage family reference. Other
tuples resolve to the family equal to their first token. This changes no coverage
inventory, pointer format, counters or acceptance ownership.

Normative coexistence fixtures: for diagnosticId `SEN02`, subject
`{"model":"ages.Education","coordinates":[]}`, metric `BOUNDARY-TV`,
these three scenario strings identify three distinct representable rows:

```text
["BASELINE"]
["AGE-MAP","Education","25","-1"]
["AGE-MAP","Education","25","+1"]
```

Canonical row order for those rows is +1, -1, BASELINE (ASCII byte order).
Observed values are not specified here; no numerical evidence is fabricated.
Other required family fixtures are exactly each tuple shape instantiated with
its first listed dimension/value, plus its second perturbation value; enumerate
all legal finite tuples from the table for an identity-only collision scan.
Unavailable expensive scenarios remain disclosed coverage, not new fitting work.

For V05, firstBucket<secondBucket in ascending exact representative order.
At each specified positive quantile, reversal=max(0,firstQuantile−secondQuantile),
targetLarger=max(the two uncompressed direct-scaled target quantiles),
limit=max(100,.05×targetLarger), excess=reversal−limit. Summary worst selection
uses greatest excess, then earliest bucket pair; pairCount is all bucket pairs
for that branch/quantile. No alternative relative-error ranking is allowed.

Diagnostic references are canonical RFC6901 JSON pointers to the final sorted
`rows` element inside the appropriate evidence CheckSet/table. No free row-key
encoding is invented. Pointer indices are determined after canonical sorting;
all pointers resolve within E, never to P or a temporary file.

CheckSet counters are mechanical. checkCount is the sum of row counts in that
CheckSet's tables except E02 d0-margins/activity-margins/activity-flows/
employer-cap-profile/employer-targets (value-only provenance tables). V05 uses
complete temporary detail rowCount instead of summary count. W01/D04 coverage
rows count only EXECUTED/REUSED scopes; D04 diagnostic rows count separately.
Empty UNEXECUTED/SUPERSEDED tables contribute zero executed checks. findingCount
is0 for HARD successful rows; V07 counts exceeds=true; W01 sums findingCount;
W02 counts warning=true risk rows plus boundaryPressure=true factor rows;
D01/D03/D04 count diagnostic finding=true; D02 adds exceeds=true cap rows to
its diagnostic finding=true count. unexecutedCount counts UNEXECUTED/SUPERSEDED
row scopes in W01/D04 and each empty optional diagnostic table marked that way;
it is0 for HARD required scope. No later-phase DEFERRED portion enters a
production counter. These counters describe execution/evidence, not new gates.

Diagnostic coverage is deterministic from producer bytes and freshly generated
stage results. REUSED means reuse within that fresh pipeline or a value explicitly
bound by the pinned authorities; it never means opportunistic discovery of old
/tmp caches, interrupted prototypes or host-local evidence. Enabled optional
metrics/coverage follow the versioned producer's fixed logic, not a command-line
choice or available-time budget. Identical inputs/result therefore cannot acquire
different evidence merely because another host retains different temporary files.

Size decisions are closed: full small required tables, complete six crosswalk
expansions/X2-X3 audit, exact source missingness/flags/count records and H1 join audit, available four-component
H1 measurement summaries, every
activity flow, all40 Employer targets/cap trace, per-law counts/zeros/atoms/V07/
caps, all branch-scale/occupancy rows, and all coverage rows stay versioned.
V05 pair details alone use full execution + count + digest + required worst
witnesses. Fine-distance/reused diagnostics retain available records; optional
absent expensive suites get coverage, not newly required enumeration. There is
no complete 70-row materiality scan or Income fine-fit table production demand.
Independent audit uses pinned external originals plus published producer and
config; a temporary digest is not a claim of standalone proof of execution.

## 13. Resource evidence

The embedded `resources` object is the resource manifest. No separate file.
Machine counts/bytes are exact JSON integers. `actual` below means the charge
computed from freshly produced numerical counts under frozen formulas; runtime
index/constructor terms are explicitly prospective, never measured Go/RSS.
Actual canonical logical JSON length is measured; metadata/reports/offline
fitting heaps do not become runtime asset input or retained model state.

| Field path | Type | Required | Structure / meaning | Array order | Identity role |
| --- | --- | --- | --- | --- | --- |
| `basis` | string | YES | `PRODUCED-NUMERICAL-ASSETS-AND-PROSPECTIVE-FROZEN-BOUNDS` | — | Prevent false runtime-allocation certification |
| `counts` | array | YES | One per24 model: `model`, `rows`,`weights`,`authoredLaws`,`canonicalLaws`,`authoredPoints`,`canonicalPoints`,`tupleSlots`,`reachableSlots`,`sentinelSlots`,`ageBuckets` (all N except model); nonapplicable numeric fields0 | model lexicographic, unique | Actual counts, no formatted units |
| `lawCardinalities` | array | YES | Per Income canonical law: `branch`,`law`,`buckets` N-array, `authoredLawCount`,`authoredPoints`,`canonicalPoints`,`distinctPositivePoints` N; authored points aggregate its bucket definitions before dedup | branch order,law; buckets ascending unique | L_a/U_a distinct from L_c/U_c; per-authored-bucket count385 retained through buckets |
| `chargeComponents` | array | YES | `{id,inputBytes,retainedBytes,decodedWorkBytes}`; N values, exact frozen formulas and itemized decode/scratch charges | id lexicographic, unique | Complete disjoint numerical-charge proof |
| `encoded`, `decodedWork`, `outsideRetained`, `outsideCombined`, `employerInput`, `employerIndex`, `R2`, `jointRetained`, `constructorPeak`, `canonicalizationWork` | object each | YES | Each exactly `{actual,reservation,ceiling,result}`; N byte quantities and result PASS | — | All required actual/reservation/ceiling categories |
| `encoded.logicalJsonByteLength` | N | YES | Additional required field only in encoded; measured inherited logical file length; encoded.actual equals this exact audit-JSON byte length; selected reservation supplies bounded prospective headroom | — | Logical source format size, not a runtime format definition |
| `decodedWork.items` | array | YES | Additional required field only in decodedWork; objects `{id,bytes}` N; component inventory below | id lexicographic, unique | Itemized parsed/canonical/working live charges before dedup |
| `employerConditions` | array | YES | 40 objects `{family,level,rawCountCeiling,positiveCountCeiling,maximumWeight,totalCeiling,analyticExpectedTotal,inputBytes,indexBytes,canonicalizationBytes,result}`; all N except analyticExpectedTotal Q/result PASS | family then level, zero-based inherited order | Exact per-index prospective input/index/totals; no realized Company roster |
| `phases` | array | YES | Each `{id,R2Live,outsideRetainedLive,encodedLive,decodedWorkLive,countStaging,rawEmployerInput,canonicalizationWork,otherLive,control,total,result}`; N bytes/result PASS | fixed phase order below | Simultaneous live allocations/reservations; no timing |
| `result` | string | YES | PASS only with every inherited formula/reservation/cap/guard satisfied | — | E05 evidence, no acceptance change |

Charge component IDs are the24 §4 model names plus `companyIndustryMetadata`,
`companyEmployeeCountMetadata`,`universeDescriptor`,`outsideBookkeeping`,
`employerIndexes`,`employerH2`; all unique. Decode item IDs are exactly
`ageDefinitions`,`ageCompiled`,`categoricalParsed`,`categoricalCanonical`,
`countParsed`,`countCanonical`,`incomeParsed`,`incomeCanonical`,`tupleMapsParsed`,
`tupleMapsCanonical`,`remapScratch`,`mergeScratch`,`rowDescriptors`,
`identityHeaders`,`additionalWork`; zeros explicitly retained. AdditionalWork
is bounded/itemized by producer's fixed source layout, not an unaccounted
allocation category. Parsed and canonical copies are distinct live charges;
backing aliases count once. Author count, not deduplicated retained count,
budgets live authored input/scratch. Item sum equals decodedWork.actual.

Frozen formula references are D2-A §21, v0 §31 and effective v3 §5. Numeric
reservations: encoded6815744 bytes, decodedWork6291456 bytes, each
ceiling8388608; outsideBookkeeping1048576; Employer H2 reservation1048576;
otherLive reservation25165824; control4096. OutsideRetained ceiling8388608;
outsideCombined25165824; Employer input/index67108864 each; R2 ceiling75497472;
jointRetained83886080; constructorPeak134217728; canonicalization≤32m with
m≤N=65536, reservation/maximum2097152. Reserve full authorized state/lifetimes,
not historical estimated output or expected .97N support. Actual≤reservation
where a selected reservation exists; reservation≤frozen ceiling throughout.

For each condition, conservative m=q=N. input/index each uses64+32K+16A/P
in aggregate (K40,A/P40N); per-index attribution is32+16N, with root64 charged
once in employerIndexes. MaximumWeight is exact max over produced count
support/Industry/positive-band of min(count,C)×I×S for that condition;
totalCeiling=N×maximumWeight with checked uint64. analyticExpectedTotal is
N×the exact one-Company expected raw weight under the preinteger Company law.
Positive support follows from every factor>0 and every count>0 being eligible;
zero-count exclusion is proved structurally. These are exact prospective
index-manifest/totals for the numerical phase, not fabricated realized totals.
Actual indexes/selected Companies/realized interval totals are later runtime
owner work under v0 §32 item3 and v3 §5:307–311.

Phase inventory is exactly `NUMERICAL-LOAD`,`COMPANY-GENERATION`, then
`EMPLOYER-00` through `EMPLOYER-39` (family outer,level inner), then
`FINAL-RETAINED`. Each serial Employer phase charges prospective retained R2,
8N count staging, one16m raw input, ≤32m canonicalization, all simultaneous
outside/E/D buffers in otherLive and4096 control. `total` is the disjoint
frozen phase sum, with named outside buffers included in otherLive rather than
added twice. Numerical loading/outside compilation includes already-live R2
if any. Final retains Rout+R2, releases loading/raw/work temporaries. No buffer
release is assumed without that phase's proof. ConstructorPeak is max phase
actual/reservation sum under the same schedule. Every additional storage,
identity/header/descriptor or compiler excess is charged, never omitted.
No physical runtime encoding/layout is specified by these charge records.
Encoded actual bytes concern the inherited audit JSON only; future Protocol
binary bytes are not inputs to this proof or any deterministic publication
field. Runtime implementations separately prove their actual layout within
the same frozen bounds without changing an existing publication identity.

## 14. Publication/checksum manifest schema

The single file/checksum authority is `D2B2-publication-v3.json`.
Complete field schema:

| Field path | Type | Required | Allowed values / meaning | Array ordering | Identity role |
| --- | --- | --- | --- | --- | --- |
| `schema` | string | YES | `SW-L2-D2B2-PUBLICATION-v3` | — | Final publication schema |
| `artifacts` | array | YES | Exactly three file entries below | filename lexicographic, unique | Sole exact-file-set/checksum authority |
| `artifacts[].filename`, `.artifactClass` | string | YES | Logical filename→CANONICAL_MODEL; crosswalk and producer filenames→VERSIONED_REPRODUCIBILITY_EVIDENCE; no other entry | as above | Fixed machine leaf set |
| `.byteLength`, `.sha256` | N,H | YES | Exact final leaf file bytes, lowercase SHA-256 | — | No normalized/decoded/file-system-metadata hash |
| `logicalAsset.artifactRef`, `.schema` | string | YES | `D2B2-logical-assets-v0.json`, `SW-L2-D2B2-logical-v0` | — | Whole logical identity resolves to that file entry SHA, no duplicate hash field |
| `logicalAudits` | array | YES | Exactly24 `{model,sha256}` entries; H over §4 inherited envelope | model lexicographic, unique | Per-model audit namespace, separate from file hashes |
| `evidence` | object | YES | Complete §11 embedded evidence manifest | §§8–13/15 | Full evidence by containment |
| `evidenceIdentity.schema`, `.sha256` | string,H | YES | `SW-L2-D2B2-EVIDENCE-v3`, SHA-256(C(evidence)) | — | Embedded evidence byte-domain identity; not final publication self-hash |
| `d2b1V3AuthorityRef` | string | YES | `D2B1.v3`, resolves to evidence.authorities pinned hash | — | Required current recipe SHA reference |
| `frozenAuthorityRefs` | array of string | YES | Contract1,Contract2,Contract3,Contract4,Contract5,D2A | lexicographic, unique | Required frozen-authority SHA references |
| `reproducibilityRef` | string | YES | `/evidence/reproducibility` | — | Run A SHA, Run B SHA and equality recorded once there |

No root `sha256`, `manifestSha256`, self artifact entry, publication filename
reference, report filename/SHA/content, signature, time, session or host data.
The external final manifest SHA is SHA-256(P), calculated only after final
serialization. A file hash already authoritative in artifacts is referenced
rather than recopied into producer/config/logicalAsset fields. Authority hashes
are authoritative only in evidence.authorities. No redundant checksum manifest.

## 15. Run A/B representation

Exactly one logical file is published. A/B are semantic labels for independent
fresh reconstructions, not filesystem names. Source order permutations are
additional prescribed tests; they are not renamed A/B or waived by equal hashes.
No timing, temporary directory, machine identity or session enters this object.

| Field path (relative to reproducibility) | Type | Required | Allowed structure / semantic meaning | Array ordering | Identity role |
| --- | --- | --- | --- | --- | --- |
| `runs` | array | YES | Exactly two `{label,logicalSha256,logicalByteLength,logicalAudits,counts}` objects | A then B | Independent output observations |
| `.label` | string | YES | A or B | fixed | No run timestamp |
| `.logicalSha256`, `.logicalByteLength` | H,N | YES | SHA/length of each exact canonical logical output; both match published logical artifact entry | — | Whole canonical-byte identity |
| `.logicalAudits` | array | YES | Exactly24 `{model,sha256}` using inherited envelopes; equal publication logicalAudits | model lexicographic, unique | Full canonical output SHA set per run |
| `.counts` | array | YES | Same24 counts object schema as resources.counts, freshly counted each run | model lexicographic, unique | Required exact object/cardinality equality |
| `byteEqual`, `objectEqual`, `countEqual`, `auditEqual` | bool | YES | true on success; actual byte/object/count/audit comparison performed, not only digest comparison | — | Complete A/B equality result |
| `rowOrderTests` | array | YES | Exactly `{order,logicalSha256,logicalAudits,byteEqual,objectEqual,countEqual}` objects for REVERSE and HASH-PERMUTATION | REVERSE then HASH-PERMUTATION | v0 §33 invariance after canonical sorting |
| `.order` | string | YES | REVERSE or HASH-PERMUTATION; exact v0 §33 reverse and SHA("D2B1-roworder-v0:"+sourceID+":"+canonicalRowKey) rule with §8.1 encoding/order/ties | fixed | Original inherited source IDs in permutation seed, .method IDs are provenance aliases only |
| `.logicalSha256`, `.logicalAudits` | H,array | YES | Equal Run A and published logical/audit values | audit order above | No reordered-source asset copy publication |
| `.byteEqual`, `.objectEqual`, `.countEqual` | bool | YES | true, compared against Run A full output | — | Evidence, no replay timing |
| `result` | string | YES | PASS only when all equalities hold | — | E04 production portion |

Duplicated output observations in A/B/row-order records are required independent
reproducibility evidence, not a competing file-checksum authority. The canonical
file entry remains authoritative for published file bytes. No numerical run is
performed by this design task; dummy labels/counts never enter repository text
as plausible production results.

## 16. Dependency DAG

Arrows are dependency→consumer. The complete graph is:

```text
Authorities ──────────────┬──────────────→ Configuration object C
External original sources│               ↑
       ↓                 │               │
External normalized states/member identities   Producer source S
       ↓                 │               │
Crosswalk artifact X ←───┴───────────────┘
       ↓
Generated logical asset L + generated validation evidence V
       ↓                         ↓
Evidence object E ← Authorities, originals/normalized/member identities,
                    X, S, C, V and run observations of L
       ↓
Publication manifest P ← E, leaf file identities of L/X/S, logical audit hashes
       ↓
Human report H ← final P/E/configuration/leaf digests and results
```

Exact edges (the drawing abbreviates multiple incoming inputs):

| Dependency → consumer | Edge meaning |
| --- | --- |
| Authorities → producer execution/configuration/X/L/V | uses as generation/provenance input (producer implementation conforms to authorities; static source bytes do not depend on a serialized manifest) |
| Producer source → configuration | uses as provenance input for exact dependency declaration and producerRef |
| Original sources → normalized states/member identities | uses as normalization/extraction input; original/member/normalized hashes are separate domains |
| Authorities + original/member/normalized identities + producer execution → X | uses as generation/provenance input for exact expansions/audits |
| Authorities + sources + X + configuration + producer execution → L/V | uses as generation/provenance input; Income construction excludes D0, validation consumes D0 |
| Authorities + external identities + X/S/C/V + L run observations → E | hashes/binds via identity fields, expansion subhashes, exact config containment and deterministic validation/run evidence; E refers only to leaf filenames, never P bytes |
| E → P | hashes/binds C(E) in evidenceIdentity and contains full E |
| L/X/S → P | hashes/binds exact leaf bytes once in artifacts; also binds L's inherited audit envelopes |
| P/E/C and leaf files → H | uses already-computed digests/results as narrative input; H is never hashed upstream |

Topological computation order: verified authorities/external originals and final
producer source; configuration/dependency declarations; normalized/member
identities; expanded crosswalk; logical/validation runs; evidence object; final
publication bytes; external publication SHA; human report; external report SHA.
Configuration can be built before X since its crosswalkArtifactRef is a fixed
name, not a crosswalk-byte digest. E's leaf refs are resolvable without P bytes.
Physical containment of E/C in P does not change this checksum dependency order.
E's own hash is stored in P, not E. P's hash is never stored in P. H's hash is
never stored in H. No self-reference and no A↔B checksum cycle exists.

## 17. Human-report boundary

The exact required production report is the HUMAN_REPORT filename in §5.
It summarizes authority verification, source/preflight result, generation,
21-row matrix, all warnings/diagnostics/coverage, exact resources, A/B/row-order
identity, artifact/config/evidence/publication hashes and limits. Prose may
include OS/CPU/elapsed time as diagnostic-only observations, but none enters
machine metadata. No prose reconstructs a missing model/config/evidence field.

The report is not an input to logical identity, publication manifest bytes,
evidence/configuration bytes or bundle identity. It has no self-hash. Calculate
its exact-byte SHA after final write and supply it externally in the final
response/audit capture; create no extra artifact solely to hash it. v0 §32's
required quality report does not mandate checksumming it into model identity,
and v1 §6 expressly makes its title reporting metadata. No contrary inherited
report-in-identity requirement exists.

## 18. Protocol boundary

WorldID/SnapshotID, runtime asset encoding/fingerprints, PRNG streams, component
stream paths, sampling/draw protocol, Go runtime serialization/wire format and
connector identities remain future Deterministic Protocol/implementation work.
Neither publication/config/source hashes nor the full bundle digest is a stream
seed. Per-model audit hashes supply inherited separable evidence, not new
runtime identities or Component framing. Changing only a future binary encoding
does not change L/P. No Go code or Protocol mechanism is designed/implemented.
Normative CPython/Decimal facts affect production; hostname/OS/CPU/PID/path/time/
session are diagnostic only and excluded from all machine checksum domains.

## 19. Assembly algorithm

For a future independently reviewed production with successful numerical output:

Validate and derive the §6.1 failure identity before the invocation; any failed
publication uses only that append-only path. This adds no successful artifact.

1. Verify ten inherited authority hashes and reviewed publication-contract bytes;
   verify exactly19 pinned original sizes/hashes/selectors and relevant ZIP
   members externally. Reconstruct/hash the three normalized API states exactly.
   Stop at any input blocker. Do not publish partial files.
2. Finalize complete single producer source and exact configuration object;
   source dependency/version declarations must match actual imports. Build the
   full six-version expanded crosswalk artifact and required audit; canonicalize
   to its fixed filename/bytes. Recheck required crosswalk gates.
3. Independently produce A/B and prescribed reversed/permuted-source outputs
   using unchanged authorities/arithmetic. Evaluate every applicable effective
   HARD scope, disclosed warning/diagnostic and prospective resource proof.
   Missing HARD evidence remains OPEN ISSUE. Divergence or any HARD failure
   permits only §6 failure publication, without retuning.
4. Canonicalize L under inherited filename/schema. Compute leaf S/X/L byte
   lengths/SHAs,24 inherited audits, config canonical bytes, full required tables,
   the temporary V05-detail digest/count and all run/count/equality records.
   No second logical asset copy or temporary evidence body is a repository file.
5. Assemble complete E from finalized upstream facts using the fixed tables,
   closed schemas/arrays and no diagnostic environment facts. Canonicalize E
   independently and compute SHA-256(C(E)). It neither contains nor hashes P.
6. Assemble P exactly per §14 with full E, its digest, sorted three leaf entries
   and audit/authority references. Canonicalize once; check parse/recanonicalize
   equality, file set, hashes, schemas, counts, result ownership, source pins,
   all required arrays and absence of cycles. Compute final SHA externally.
7. Write the one human report after P/S/X/L are final, quoting computed hashes.
   Compute report SHA externally after final bytes. Validate full five-file
   successful set; make all five reviewable together. Do not stage/commit here.

For identical authorities, original bytes, producer bytes and deterministic
generation result, configuration, expanded crosswalk, logical asset, versioned
machine evidence and final publication bytes MUST be identical across successful
productions. Source dependency modules follow producer bytes; count/witness ties
are fixed above. No production time/temporary-path/host fact enters this rule.
Human report wording may differ without changing either model or bundle digest.

## 20. Validation/negative rules

Reject unknown/missing keys, missing/unknown artifact filename, duplicate file
entry, wrong class/schema/SHA format, bool/float/string byteLength, unsorted or
duplicate unique-ID arrays, duplicate object keys, invalid result vocabulary,
WARNING/DIAGNOSTIC/later DEFERRED encoded PASS, missing required evidence, wrong
source/member/version sets, incorrect pin, unresolved refs or incorrect digest.
Publication success forbids false equality, HARD FAIL/OPEN ISSUE and any
production HARD scope mislabeled later-owner DEFERRED.

No deterministic string designated a filename/path/ref may be absolute or
contain a temporary directory, drive prefix, parent-directory traversal or URI
for a local file. Only fixed repository basenames, fixed official URLs and
ZIP-member/source-coordinate names are allowed. Raw source text is preserved
under inherited parsing semantics, never silently edited by this path rule.
A producer literal embedding a host temporary path is forbidden. Public manifest
cannot include itself in artifacts, its own final SHA under any field, or the
human report. E/C cannot include a final P digest or publication filename ref.
Containment/leaf references are allowed as precisely specified, not arbitrary
checksum edges. Reject self-reference attempts before serialization.

The three rectified identities also reject alternate/missing coordinate order,
unknown source-key kind, invalid arity/domain, delimiter-joined or differently
escaped/framed source keys, and any secondary key used in source/permutation
identity. Framing-sensitive input is encoded as data; ambiguous unframed input
is rejected. Reject missing/unknown/noncanonical sensitivity scenarios, invalid
map/boundary/sign/magnitude/dimension tuples, duplicate complete diagnostic keys,
and any producer that projects away scenario and merges the -1/+1 witnesses.
Reject historical failure overwrite, skipped ordinal, missing/corrupt/duplicate
history, arbitrary first-free naming and timestamp/UUID failure basenames under
§6.1. All previous schema/identity/acceptance negative rules remain binding.

Adversarial expected results:

| Case | Mechanically required result |
| --- | --- |
| A different /tmp directory | Only external execution paths change; S/config/X/L/E/P bytes unchanged. |
| B different wall-clock time | No time field in canonical domains; machine bytes unchanged. |
| C different hostname/OS diagnostic data | Ignored by machine assembly; unchanged bytes; CPython3.13.2/Decimal facts still required. |
| D original source bytes change | Pinned length/hash gate fails; no successful bundle. |
| E normalized result changes | Three exact normalized pins fail; success forbidden. Other derived extraction changes alter evidence and/or fail their applicable input/construction gate. |
| F producer bytes change | File entry SHA/length and dependency declaration change as applicable; P bundle identity changes even if L is identical. |
| G config bytes change | C(config) and contained E/P change; unknown/override changes are rejected. A legal dependency-declaration change must match changed producer imports. |
| H A/B canonical bytes differ | byteEqual=false and/or unequal output identity; success publication forbidden, regardless of any accidental equal digest. |
| I report wording only changes | H external SHA changes; L and P identities unchanged. |
| J future Protocol binary encoding changes | No P/L input edge; D2-B2 logical/publication identities unchanged. |

A tiny /tmp-only schema/serialization exercise must instantiate every new
schema-bearing domain (crosswalk, config, evidence, publication, temporary
V05 detail) with explicitly dummy data, twice from identical inputs. Require
byte-identical assembly, identical manifest digest, acyclic order, all required
fields satisfiable and all filenames predetermined. These are schema tests,
not source preflight, numerical production, arithmetic fixture execution,
Run A/B or produced-asset resource accounting. Supporting report records actual
dummy results and negative cases; dummy SHA/counts never stand for real output.

## 21. Fresh-production checklist

If a fresh producer reaches successful numerical output, must it make ANY
new repository-publication decision? **NO**.

- Filename choice: none; exactly five success basenames and one derived append-only failure basename (§6.1).
- Manifest/schema/class choice: none; one publication file, complete embedded
  evidence/config, fixed expanded crosswalk/source/report classes and schemas.
- Checksum-domain choice: none; exact file hashes, inherited logical/audit
  envelopes, exact embedded evidence/config domains and fixed temporary detail.
- Source/normalized provenance choice: none; exact19 originals, relevant members,
  three external normalized states and one metadata-only X4 identity.
- Crosswalk choice: none; all six full expansions/audits in one fixed JSON file.
- Producer/config publication choice: none; one complete source file, one embedded
  complete config object with exact authority/software/arithmetic references.
- Acceptance/resource/Run representation choice: none; exact21 rows/owner portions,
  closed table catalog, exact integer charge object and A/B/row-order evidence.
- Report/manifest dependency choice: none; human report downstream/excluded,
  external final hashes and complete acyclic DAG.

No numerical production, source fitting/generation, 19-source fresh preflight,
Company/categorical/Employer/Income output, production arithmetic fixtures,
real Run A/B or produced-asset D2-A accounting occurs in this design. Principal
logical filename/schema/structure/semantics/dimensions/orders/IDs/integers/
serialization/whole and per-model checksum rules remain inherited unchanged.
This proposed publication contract is ready only for micro independent review;
it neither certifies D2-B2 production success nor freezes any D2-B authority.

## 22. Rectification history

Three-finding in-place v3 rectification; status remains **PROPOSED / UNFROZEN**.
Pre-rectification SHA-256:
`6234d4e574f817b58202be5a92614a759244a6b5877362df503e3e3dee6d20ba`.
Reason: independent Astra review found three HIGH publication-contract defects:
source coordinate encoding, diagnostic scenario identity, and append-only failure
history. Only §§6.1/8.1/12.1 and their mechanically dependent schema/order/path
references are rectified. No v4, numerical production, acceptance/model change,
identity/checksum architecture change or Deterministic Protocol design occurs.
The original publication-contract report remains unchanged supporting evidence;
its old key/failure-path descriptions are superseded only by this normative delta.
