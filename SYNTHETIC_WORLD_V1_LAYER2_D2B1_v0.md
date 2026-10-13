# Synthetic World v1 — Layer 2 D2-B1 v0

## Empirical Evidence, Crosswalks and Numerical Fitting Plan

## 1. Status and frozen authority

**Proposed evidence/fitting basis, revised in place after self-review and ready for independent review; not frozen.** D2-B1 fixes the numerical production recipe. D2-B2 computes and audits assets; it does not select evidence or redesign the recipe. Layer 2 is not implemented here.

Authority is Contracts 1–5 and D2-A, all read in full. The checksum gate passed before empirical research:

| Authority | SHA-256 verified |
| --- | --- |
| `SYNTHETIC_WORLD_V1_LAYER2_D2A_v1_FROZEN.md` | `33ca4c80599dddb0564ea4d27262e430b1931e0368c8d1bee6ca6c10d8078be4` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT1_v1_FROZEN.md` | `bc148f17d77adeb779d6c28cc5b87962ee1eb131f7e473444dd9b66ca13fa4f6` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT2_v1_FROZEN.md` | `e9af32a0bb9b48bf8972a408a1369901a3f3a9fe5ecd76d5762a367c055bc098` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT3_v1_FROZEN.md` | `df3ef0b52f37b587b8449cc71085dc9e5aca00f754bbd93e123c6fd5e3a4cb55` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT4_v1_FROZEN.md` | `a5ecfb735270ffba3e6afea23d85e548dd6dc3f07a8e9dfc5294a84ad3576418` |
| `SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md` | `8a75054f793ef47e7071e3eb9fe04c0b32abfdac9223da859f95a333a1526e67` |

The parent sets, taxonomies, DAG, Employer factorization and index architecture, Income semantics, finite-law family and maxima, support architecture, resource architecture, and identity/Protocol boundaries remain normative. Historical review wording inside checksum-authorized artifacts does not demote their authority.

The requested review profile is GPT-5.6 Sol xhigh. This document does not certify a model configuration that the execution environment did not expose.

Current quality target: **Plausible**, under the acceptance policy in §§27/28/30. This intentional product-requirement revision changes statistical acceptance, not the socioeconomic generator. Contracts 1–5 and D2-A remain frozen and byte-identical; their historical Plausible+ terminology does not impose a universal rare-tuple error threshold or a quality-report filename. D2-A §23 expressly leaves actual fit thresholds to D2-B. All required architectural evidence, including rare combinations, remains reported.

This redesign authorizes an in-place edit of this proposed/unfrozen specification and the new root report `SYNTHETIC_WORLD_V1_LAYER2_D2B1_PLAUSIBLE_REDESIGN_REPORT.md`. The user's subsequent instruction additionally authorizes `SYNTHETIC_WORLD_V1_LAYER2_D2B1_PLAUSIBLE_REDESIGN_FINAL_RESPONSE.md` to hold the final response. Historical integrity records below describe their original executions. Diagnostic scripts and calculations remain under `/tmp`; no numerical assets, implementation, staging or commit are produced.

Numerical-label convention: **SOURCE-BACKED [ID]** means an observation or an exactly specified arithmetic transform of it. **MODELED ASSUMPTION** means an authored target, extrapolation, split, regularization, lifecycle boundary or algorithmic constant. Every numerical constant in the fitting recipe is MODELED ASSUMPTION unless explicitly identified as SOURCE-BACKED or a normative frozen limit. Combining a source margin with an assumed split produces an assumed target, not a new official statistic. Resource arithmetic is a projection from normative D2-A formulas, not empirical evidence.

## 2. Research and anchor-date policy

Research/access date: **2026-09-16**. `PopulationAnchorDate = 2026-01-01`; public Income is nominal anchor-basis EUR.

Reference period, publication date, release/version and access date are separate fields in §5. Full-year 2025 observations published in 2026 are eligible. A later revision of 2024 observations is eligible; its snapshot is fixed. No January–August 2026 realized employment, pay, benefits or prices enters production targets. The July 2026 ATECO correspondence update is eligible methodology, not an economic outcome.

Selected demographic/labour observation vintage is 2024. Company size uses 2023 ASIA; detailed business composition uses 2022 FRAME-SBS because the inspected 2023 table has only four broad sectors. The resulting anchor stock is an explicitly assumed vintage carry-forward. SES 2022 supplies remuneration calibration; INPS 2024 provides a nearer level check and pension-component calibration. These vintages are retained without automatic inflation adjustment.

Monetary uses are fixed: W1 RELATIVE STRUCTURE and DISTRIBUTION SHAPE; W2 ABSOLUTE LEVEL calibration; R1 ABSOLUTE LEVEL pension-component calibration; H1 DISTRIBUTION SHAPE/TAIL CROSS-CHECK. No monetary index is applied. In particular, no CPI wage uplift, pension revaluation or historical net-to-gross conversion is inferred. All final all-source Income quantile targets are MODELED ASSUMPTION on the nominal anchor basis.

## 3. D2-A model requirements matrix

Here `B` is the selected bucket count, `r` categorical rows, `Q` categorical weights, `L` Income laws, `U` support points, and `T` tuple slots. Every authorized grid is completed offline. All states remain semantically valid; fitted support is not a new validity rule.

| # | Output fact/model | Authorized parents | Numerical shape and capacity | D2-B required decisions | Plausible relationships | Resource charge |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Company.Industry | None | One 14-state law | Enterprise-unit concept, shares, crosswalk, integer weights | Industry, Industry×size | Categorical r=1,Q=14 |
| 2 | Company.EmployeeCount | Industry | 14 finite laws; ≤128 authored/canonical points each | Headcount reconciliation, support, body/tail/zero/cap and fitting | Size bands, mean/quantiles, upper tail, zeros | Support family L=14,U≤1792 |
| 3 | EducationLevel | EducationAgeBucket, Region | B≤6 ×20, seven states | Age map, attainment split, regional completion | Education×age/Region | r=120,Q=840 selected |
| 4 | PrimaryActivity | ActivityAgeBucket, Education, Region | B≤8 ×7×20, six states | Source overlaps, age map, complete conditional laws | Activity×age/Education/Region | r=1120,Q=6720 selected |
| 5 | Employee.JobFamily | Education, Region | 7×20, ten states | Occupation crosswalk, employee fit | Family×Education/Region | r=140,Q=1400 |
| 6 | Employee.JobLevel | LevelAgeBucket, Education, JobFamily | B≤6 ×7×10, four states | Responsibility proxies, age map, pooling/shrinkage | Level×age/Education/family | r=420,Q=1680 selected |
| 7 | SelfEmployed.JobFamily | Education, Region | 7×20, ten states | Independently fitted SE occupation assets | Family×Education/Region | r=140,Q=1400 |
| 8 | SelfEmployed.Industry | JobFamily | Ten 14-state laws | SE-sector targets and independent crosswalk/fit | SE Industry×family | r=10,Q=140 |
| 9 | Employer | JobFamily, JobLevel, frozen Company domain/Industry/EmployeeCount/SizeBand | min(n,C)×I×S; zero n excluded; I 10×14, S 4×4 | C, all 156 factors, joint Company-model evaluation | Selected Industry×family, size×level, employee exposure, unusual valid support | 40 direct indexes; raw 40N; positive ≤40N |
| 10 | Income[e] | Region, JobFamily, JobLevel, CurrentIndustry, SizeBand | 56,000 map slots; 44,800 reachable; ≤512 laws | All-source targets, pooling, compression | Branch/Region/family/level/Industry/size | 11,200 zero-band slots sentinel 65535 |
| 11 | Income[s] | Income[s]AgeBucket, Region, JobFamily, Industry | ≤16,800 slots; ≤256 laws | Personal attributed SE earnings, age map, tails | Age/Region/family/Industry | T=16,800 selected |
| 12 | Income[u] | Income[u]AgeBucket, Education, Region | ≤840 slots; ≤64 laws | Benefits plus eligible personal inflows, age map | Age/Education/Region, zeros/tails | T=840 selected |
| 13 | Income[t] | Income[t]AgeBucket, Education, Region | ≤840 slots; ≤64 laws | Scholarship/personal inflows; no family support | Age/Education/Region, zeros/tails | T=840 selected |
| 14 | Income[r] | Income[r]AgeBucket, Education, Region | ≤840 slots; ≤64 laws | Pensions plus other eligible inflows | Age/Education/Region, zeros/tails | T=840 selected |
| 15 | Income[i] | Income[i]AgeBucket, Education, Region | ≤840 slots; ≤64 laws | Personal inflows without hidden household state | Age/Education/Region, zeros/tails | T=840 selected |

Every Income law has ≤512 authored pairs and ≤512 canonical positive points; aggregate selected support budget is 240,000, not 512 points everywhere. Aggregate law ceiling is 1024. D2-A resource limits remain: categorical/support/map charges; 64 MiB Employer input/index; 72 MiB R2; 8 MiB each outside retained, encoded input and decoded/work state; 24 MiB combined outside accounting; 80 MiB joint retained; 128 MiB constructor peak. §31 applies the exact formulas.

Canonical order is D2-A order, never alphabetical source order. Industry: agriculture; production-and-utilities; construction; commerce; transport-and-logistics; hospitality; information-and-communication; finance-and-real-estate; professional-and-business-services; public-administration-and-safety; education; health-and-social-care; arts-and-recreation; other-services. JobFamily: production-and-fieldwork; skilled-trades; transport-and-logistics; customer-and-personal-service; administration; sales-and-commercial; technical-and-scientific; professional-and-creative; care-and-education; protective-services. Education: below-primary, primary, lower-secondary, upper-secondary, post-secondary-non-tertiary, tertiary, doctoral. Activity: employee, self-employed, unemployed, student, retired, inactive. Level: individual-contributor, supervisor, manager, executive. Size: zero, micro (1–9), small (10–49), medium (50–249), large (250+).

## 4. Source-selection methodology

Execution gates are sequential: A requirements reconstruction → B candidate/methodology/unit discovery → C minimal source selection → D explicit reconciliation → E single fitting recipe → F actual proposals → G production contract. The Company/Employer numerical pilot used the reconciled proposals; it is a proposal check, not canonical asset production.

Candidate scoring uses integer 0–4 judgments on semantic compatibility, unit, primary ownership/quality, anchor relevance, coverage, dimensions, reproducibility, recency and inverse complexity. Compare lexicographically in that order; recency cannot override concept/unit incompatibility. Scores are analyst judgments, not official statistics.

| Intended use | Selected candidate and score vector | Rejected/downgraded candidate and decisive reason |
| --- | --- | --- |
| Company sector | B1 enterprise counts (3,4,4,2,2,4,4,2,4) | Workforce-weighted LFS sectors (1,1,4,4,4,2,4,4,4): wrong Company unit, duplicates Employer exposure |
| Company headcount shape | B1 ASIA 2023 (2,4,4,3,2,3,4,3,4) | SBS personnel costs/FTE (1,2,4,2,2,3,4,2,3): not stock headcount |
| Education | E1 LFS attainment tables (3,4,4,3,3,3,4,3,4) | Enrolment/graduates: flows or current participation, not completed attainment |
| Activity | A1 LFS age/Education/Region margins (2,4,4,3,3,3,4,3,3) | Administrative beneficiaries: eligibility is not principal activity; PL032 SUF unavailable as accessible production joint |
| Occupation | O1/O2 branch-specific LFS margins, O3 common education association (3,4,4,3,3,2,4,3,3) | Company sector shares: neither branch-specific occupations nor personal units |
| JobLevel | A1 administrative qualification calibration plus explicit responsibility prior (1,3,4,3,2,2,4,3,3) | ISCO major group equals authority tier: rejected semantic substitution |
| Employee monetary calibration | W2 2024 gross historical remuneration + W1 SES structural audit (2,4,4,3,2,3,4,3,3) | Salary-only as all-source Income: incompatible; restricted SES file is not indispensable |
| Pension component | R1 beneficiary annualized stock (3,4,4,3,3,3,4,3,3) | D05 pension counts: benefit unit, not recipient's aggregate |
| Personal all-source Income | Explicit prior constrained by W1/W2/R1, H1 component cross-check | Household totals and historical signed/net aggregates fail frozen concept; no selected accessible source identifies the current gross eligible all-source rate |

The sufficient set is three ASI chapter files, three small Eurostat responses, two remuneration publications, one pension publication, one public SHIW release, and classification/methodology documents. Extra collection stops here. LFS public-use and restricted research access, EU-SILC, SES access rules and SHIW documentation were investigated. Restricted data are never indispensable. No commercial or unofficial economic data are used. Gap-filling OECD/Commission datasets are not selected because they add no compatible indispensable dimension.

## 5. Source ledger

All access dates below are 2026-09-16. `pub not stated` is an honest missing metadata field, not an inferred date. ASI publication date is the edition release, not necessarily the original survey dissemination date. API `updated` is the snapshot version timestamp; the original first-publication date of each 2024 series is not exposed in the response. Production uses the captured version, not “latest”. A use has exactly one evidence class; monetary-use roles in §2 do not upgrade that class.

| Source ID | Institution | Dataset/release | Official URL | Release/version | Reference period | Publication date | Access date | Statistical unit | Population/coverage | Classification | Variables | Geography | Access type | Evidence class | Models using it |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| B1 | ISTAT | ASI2025 D14; ASIA/FRAME-SBS | [D14.zip](https://www.istat.it/storage/ASI/2025/dati/D14.zip) | ASI2025 captured bytes | ASIA 2023; FRAME-SBS 2022 | 2025-12-17 edition | 2026-09-16 | Statistical enterprise; annual workforce | Active market enterprises; excludes/omits parts of A,K,O and non-market populations in detailed SBS table | ATECO2007 update2022 / NACE Rev.2 | 14.1,14.2,14.8: enterprises, addetti, dipendenti, band composition | Italy, Region; sector detail nationwide | Official XLSX in ZIP | CALIBRATION | 1,2,9; audit of 8 |
| E1 | ISTAT | ASI2025 D07 attainment | [D07.zip](https://www.istat.it/storage/ASI/2025/dati/D07.zip) | ASI2025 captured bytes | 2024 annual average | 2025-12-17 edition | 2026-09-16 | Resident person | LFS private-household population; table ages15–89 | Highest completed qualification, grouped ISCED2011-compatible levels | 7.13,7.14: total-person counts, age, qualification, Region | 20 Regions, age bands | Official XLSX in ZIP | TRANSFORMABLE | 3,4,5,6,7; audit masses 10–15 |
| A1 | ISTAT | ASI2025 D08 labour/ASIA employment | [D08.zip](https://www.istat.it/storage/ASI/2025/dati/D08.zip) | ASI2025 captured bytes | LFS2024; ASIA employment2023 | 2025-12-17 edition | 2026-09-16 | Person for LFS; annual employees for ASIA | Private-household LFS; market-enterprise employee subset for 8.9/8.11 | LFS professional status; qualification groups; ATECO2007 update2022 | 8.1,8.2,8.3,8.9,8.11 | Region/national, broad sector | Official XLSX in ZIP | CALIBRATION | 4,6,8,9; audit masses 10–15 |
| O1 | Eurostat/ISTAT EU-LFS | lfsa_eegais | [API O1](https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/lfsa_eegais?geo=IT&time=2024&sex=T&lang=EN) | updated2026-09-10T23:00:00+0200 | 2024 | First publication not exposed; version2026-09-10 | 2026-09-16 | Employee person, main occupation | LFS private households, age20–64 selected | ISCO-08 majors | freq=A,unit=THS_PER,sex=T,age=Y20-64,isco08 OC0–9,NRP | Italy | Official JSON-stat API | CALIBRATION | 5,6,9 |
| O2 | Eurostat/ISTAT EU-LFS | lfsa_esgais | [API O2](https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/lfsa_esgais?geo=IT&time=2024&sex=T&lang=EN) | updated2026-09-10T23:00:00+0200 | 2024 | First publication not exposed; version2026-09-10 | 2026-09-16 | Self-employed person | age20–64, wstatus=SELF, all own-account/employer self-employment | ISCO-08 majors | freq=A,unit=THS_PER,sex=T,age=Y20-64,wstatus=SELF,OC0–9,NRP | Italy | Official JSON-stat API | CALIBRATION | 7,8 |
| O3 | Eurostat/ISTAT EU-LFS | lfsa_egised | [API O3](https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/lfsa_egised?geo=IT&time=2024&sex=T&lang=EN) | updated2026-09-10T23:00:00+0200 | 2024 | First publication not exposed; version2026-09-10 | 2026-09-16 | Employed person, not employee-only | age20–64 all employed | ISCO-08; ISCED2011 | ED0-2,ED3_4,ED5-8 ×OC0–9; NRP | Italy | Official JSON-stat API | CALIBRATION | 5,7 |
| W1 | ISTAT | RCL-SES2022 report/methodology | [SES2022 PDF](https://www.istat.it/wp-content/uploads/2025/01/REPORT_STRUTTURA_RETRIBUZIONI_2022.pdf) | Report2025-01-20 | October2022 and calendar2022 | 2025-01-20 | 2026-09-16 | Paid employee/position in economic unit | Units≥10 employees, B–S; Italy includes O; A,T,U and micro units excluded | ATECO2007; CP2021 (report glossary); native completed-qualification groups, not consumed as source-code conditioning | Gross hourly earnings, annual full-year/full-time equivalent remuneration | Published NUTS1/national | Official publication; scientific microdata not required | CALIBRATION | 10 |
| W2 | INPS | Private employee observatory2024 | [Report id1043](https://servizi2.inps.it/servizi/osservatoristatistici/api/getAllegato/?idAllegato=1043) | Statistics in brief Nov2025 | Calendar2024 | 2025-11-18 announcement | 2026-09-16 | Worker with≥1 paid day in year | Private non-agricultural, domestic workers excluded | Administrative qualifica; geographic groups | Total workers, gross historical remuneration, paid days | Italy/Region and macroareas | Official publication | CALIBRATION | 6,10 |
| R1 | INPS | Casellario pensionati2024 | [Report id1007](https://servizi2.inps.it/servizi/osservatoristatistici/api/getAllegato/?idAllegato=1007) | July2025 administrative extraction | Stock2024-12-31 | 2025-10-23 | 2026-09-16 | Beneficiary and benefit; recipient totals distinguished | Italian pension system incl. assistential/indemnity recipients, not only retired primary status | Pension categories, recipient income classes | Recipient annualized gross pension total, age, macroarea | Italy/Region/macroareas as published | Official publication | CALIBRATION | 14; branch monetary audit |
| H1 | Banca d'Italia | SHIW2022 annual ASCII | [ind22_ascii.zip](https://www.bancaditalia.it/statistiche/tematiche/indagini-famiglie-imprese/bilanci-famiglie/distribuzione-microdati/documenti/ind22_ascii.zip) | Annual release listed2025-10-27; pinned bytes | Calendar2022 | Release2025-10-27; archive description2025-01-10 | 2026-09-16 | Person components; household resources separately | Survey private households; anonymized annual archive | Native SHIW2022 variable/category release; occupation, education and industry codes unused; no assumed CP/ISCO vintage conversion | CARCOM22 ETA,PESOFIT,NQUEST,NORD; RPER22 YL1,YTP1,YM,YTA2; excluded aggregate Y | Region/macroarea available; fine geography discarded | Public-use CSV in ZIP | CROSS-CHECK | 10–15 |
| X1 | ISTAT | ISCO08–CP2021 official correspondence | [ISCO–CP XLSX](https://www.istat.it/wp-content/uploads/2011/03/raccordo-ISCO08_III-Cp2021_V.xlsx) | CP2021, official release accompanying classification | In force2023-01-01 | Classification announcement2023-01-09; file publication not stated | 2026-09-16 | Classification code relation | ISCO08 minor/CP2021 detailed codes | ISCO-08,CP2021 | Code pairs | No geography | Official XLSX | CROSS-CHECK | 5,7 crosswalk audit |
| X2 | ISTAT | ATECO2007 update2022 structure | [ATECO2022 XLSX](https://www.istat.it/it/files/2022/03/Struttura-ATECO-2007-aggiornamento-2022.xlsx) | ATECO2007 update2022 | Classification in force2022 | 2022-03 classification release; exact file day not stated | 2026-09-16 | Code/title | Economic activities | ATECO2007 update2022 | All code/title rows | No geography | Official XLSX | TRANSFORMABLE | 1,2,8,9 |
| X3 | ISTAT | ATECO2025↔2022 correspondence update2026 | [Bridge2026 XLSX](https://www.istat.it/wp-content/uploads/2026/07/Aggiornamento-2026-Tavola-raccordo-bidirezionale-ATECO-2025-ATECO-2022-italiano.xlsx) | Bidirectional theoretical correspondence, update2026 | Classification methodology, no economic period | 2026-07-15 | 2026-09-16 | Many-to-many code relation | Both classifications | ATECO2025/NACE2.1 vs ATECO2022/NACE2 | Code relation, partial/total flags, highlighted changes | No geography | Official XLSX + note | CROSS-CHECK | Classification audit only; no production bridge |
| X4 | UNESCO UIS | ISCED2011 | [Official ISCED publication](https://uis.unesco.org/sites/default/files/documents/international-standard-classification-of-education-isced-2011-en.pdf) | ISCED2011 adopted2011; published2012 | Classification, no observation year | 2012 | 2026-09-16 web text; direct download404 | Qualification/program standard | International metadata | ISCED2011 attainment | Attainment level concepts | No geography | Official publication text; no production download dependency | CROSS-CHECK | 3 education semantics |
| M1 | Eurostat | EU-SILC DocSILC065 | [2025 updated guidelines](https://ec.europa.eu/eurostat/documents/d/microdata/methodological-guidelines-2025-operation_updated) | 2025 operation Updated | Methodology; no Italian income observation selected | Publication day not stated | 2026-09-16 | Personal P/R vs household H/D variables | Survey population described by operation; SUF access restricted | ISCED2011, labour status code version2021+ | PY010G,PY050G,PY080G–PY140G; HY040G,HY090G; PL032,RB050 | Methodology only | Official PDF; no production SUF dependency | CROSS-CHECK | 4,10–15 concept audit |

Official methodology companions: [N14](https://www.istat.it/storage/ASI/2025/note-metodologiche/N14.pdf), [N07](https://www.istat.it/storage/ASI/2025/note-metodologiche/N07.pdf), [N08](https://www.istat.it/storage/ASI/2025/note-metodologiche/N08.pdf); [LFS methodology](https://ec.europa.eu/eurostat/web/lfs/methodology); [SHIW archive description](https://www.bancaditalia.it/statistiche/tematiche/indagini-famiglie-imprese/bilanci-famiglie/documentazione/documenti/2022/Legen22.pdf); [ATECO bridge note](https://www.istat.it/wp-content/uploads/2026/07/Aggiornamento-2026-Tavola-raccordo-bidirezionale-ATECO-2025-ATECO-2022-Nota.pdf); [ISCO official structure](https://isco-ilo.netlify.app/en/isco-08/). Population coverage is not inferred from row totals alone.

Captured-file provenance (download/access2026-09-16; bytes are transport-decoded file bytes, not ZIP members). The exact official URLs above and companion links determine the locations.

| Source/use | Captured filename | Format | Byte size | SHA-256 | Relevant identifiers |
| --- | --- | --- | ---: | --- | --- |
| B1 | `D14.zip` | ZIP/XLSX | 331793 | `4fbba671bdb2427cff5c58e87d027f55bdf8af2c4878d6d6f14e4df08dfc037a` | D14/C14_Tavole.xlsx:14.1,14.2,14.8 |
| E1 | `D07.zip` | ZIP/XLSX | 1662813 | `681ab9a6e7a2fffb4ac88c7c259a8188dd718171a38366cf3ad95409b0e23f7c` | D07/C07_Tavole.xlsx:7.13,7.14 segue |
| A1 | `D08.zip` | ZIP/XLSX | 501818 | `cf1612975f5743e6f2ff144f382f4076ca6589e8f0d015fff06dbde3be1f4e8f` | D08/C08_Tavole.xlsx:8.1 segue2,8.2,8.3,8.9,8.11 |
| O1 | `lfsa_eegais.json` | JSON-stat | 7975 | `ef00067100bdd392a0755ac1bd8fb8683bf5ffa448d004935e05070885a21d5f` | Dimensions in O1 ledger |
| O2 | `lfsa_esgais.json` | JSON-stat | 13478 | `81f1a2cf5906acec6a4f523ecdbe7375301307e5969f592380f8aeca6dc0ece6` | Dimensions in O2 ledger |
| O3 | `lfsa_egised.json` | JSON-stat | 5886 | `3d782ef806dd9eb822aae0748f75053965cf3b4b5f6af6d6f92588eb98119d27` | Dimensions in O3 ledger |
| W1 | `SES2022.pdf` | PDF | 993677 | `0be4c2e576a40c3fe5a376947cdfdfa9e82e610da1b9ccca3a0bdff2a81e9616` | Report tables; glossary pp9–10 and methodological note pp11–15 |
| W2 | `inps-wages2024.pdf` | PDF | 658673 | `d8cf1e1e6bbfe380a4cc978ae424242eb0c57cd11cceeeac53b23bafdca8ec7e` | Prospetto1; methodology footnotes |
| R1 | `inps-pensionati2024.pdf` | PDF | 692528 | `7dbeb6e13c14accc942c9526587458aa162198aba6f0fac80af84a8b217a8a58` | Tavola1 total2024, recipient income classes |
| H1 | `ind22_ascii.zip` | ZIP/CSV | 3566985 | `fd31bb4263363b9098de6f0d0640a90484dc30d0d4fd0c5323706d61dc9fb675` | CSV/carcom22.csv,CSV/rper22.csv |
| X1 | `isco-cp2021.xlsx` | XLSX | 49763 | `1bdd98bcc5de620225afe073b72495b5900e20d387c569820756e7b0df0b4e2b` | `III isco08- Vcp2021` |
| X2 | `ateco2022.xlsx` | XLSX | 115953 | `7c284756dcb972cca3f5c3842d007678566fb9aeb1db09a670ca0a889427cfaf` | Struttura Ateco 2007 agg 2022 |
| X3 | `ateco-bridge2026.xlsx` | XLSX | 615444 | `faf4530a869f1e3d6b68f464520dcddd691f4565d17e92e3988370cf3c808673` | ATECO 2025 vs ATECO 2022;LEGENDA |
| X3 method | `ateco-bridge2026-note.pdf` | PDF | 631907 | `e1700811a4a1049603aca9f3823995503f20291178c5f169f467156e129e5f0b` | Bidirectional relation methodology |
| M1 | `silc-guidelines.pdf` | PDF | 6299128 | `c5d746f3d02025dd21f0309d3ad5259d7c0a7f32714a2519e41932de58a85adf` | DocSILC065 personal/household income and PL032 definitions |
| B1 method | `N14.pdf` | PDF | 94170 | `4a9cf70051b9970471f1396becb42d13fc3e4335420b831e153f23e7f069e2bc` | ASIA and FRAME survey/register definitions |
| E1 method | `N07.pdf` | PDF | 93982 | `4dd6b0253458b507ad846c8194e3526033595cb0e62378682174c5e6762b3d6f` | LFS attainment methodology |
| A1 method | `N08.pdf` | PDF | 94453 | `fb113fd9ddaeaf7dc2a6a4c82b7d2bb7b3468a03d5b2fbca6c6f5fad33418eb1` | ASIA occupation and LFS metadata |
| H1 method | `Legen22.pdf` | PDF | 455568 | `d30ec3ee46cae84bd5c05304ab0dd1bb8324b5ec6b9d2a64915e4465b206d6e7` | Table2 income aggregation and allocation footnotes |

ASI extraction is by worksheet title and cell coordinates specified below, never sheet order. Read numeric cells from their official cached XML `<v>` decimal text, including cached formula results; do not recalculate Excel formulas or round the values to the printed precision. Resolve shared strings/inline strings for labels, not as numeric observations. A missing numeric cache in a selected cell is an input failure. A hyphen, dot or blank is not an observed numeric zero unless the table legend establishes it; the selected input rectangles have numeric cells. JSON values carry status flags; missing/suppressed cells are treated by §8, not silently zeroed. O1/O2/O3 total-sex rows aggregate all genders; no gender-specific row is selected. SHIW checks use all records irrespective of SEX.

X4 is metadata only: the official release identity is ISCED2011 (UIS2012). Its downloadable URL returned404 while official text was readable through web access. No canonical numerical asset depends on reacquiring that PDF. D2-B2 must retain the recorded standard identity, not replace a numerical source with it.

## 6. Classification-version reconciliation

Versions are named independently of observation dates. B1 and A1 ASIA/SBS use ATECO2007 aggiornamento2022 (national extension of NACE Rev.2); O1–O3 use ISCO-08 and O3 uses ISCED2011. W1 reports ATECO2007 and CP2021 in its glossary; no unpublished ATECO update2022 equivalence is asserted. Its selected national/hourly-sector aggregates require no occupation-code conversion; no CP2021 count is combined with the ISCO08 production margins. E1/A1 Italian qualification group labels are mapped directly to the frozen attainment states, with explicit offline splits. H1 codes are used only for component checks, not silently interpreted as CP2021.

X3 is the **2026-07-15** updated theoretical bidirectional correspondence. It replaces the 2025-02-07 correspondence; the official notice says the classification structures are unchanged while relations are revised. The separate operational recoding table is not a population-allocation matrix. [Official update](https://www.istat.it/notizia/relazioni-corrispondenza-ateco-2025-ateco-2022-aggiornamento-2026/).

Production route: **ATECO2007 update2022 source → SW Industry directly**. No observation classified ATECO2025 or NACE Rev.2.1 is selected, so no production vintage bridge is necessary. X3 is a compatibility audit under the exact code-set procedure below; flag partial and cross-state relations without reweighting production. A many-to-many relation has no implicit equal split. An attempted replacement using a 2025-classified release is a source change requiring D2-B1 revision.

Occupation route: **ISCO-08 major aggregates → authored offline probabilistic SW JobFamily crosswalk**. CP2011 and CP2021 are not combined. X1 supports the audit of CP2021/ISCO minor categories but does not upgrade major-group splits to direct evidence. The frozen JobLevel has no ISCO skill-level correspondence. In particular, ISCO's ISCED1997 skill model is not used as an attainment vintage bridge.

**ATECO code identifier parsing (X2/X3 classification audit only):** for every code-cell text used as an identifier, including X2 code column A and X3 code columns B (ATECO2025) and F (ATECO2022), read the source cell text exactly and preserve that raw text separately with its source/cell provenance for audit reporting. Form the canonical code identifier by removing only U+0020 SPACE characters from the beginning and end of the string. Do not remove U+0009 TAB, U+000A LF, U+000D CR, U+00A0 NO-BREAK SPACE or any other Unicode whitespace. Preserve internal spaces, dots, digits, letters, slashes, hyphens, punctuation and case. No Unicode whitespace stripping, whitespace collapsing, regex cleanup, punctuation normalization, case folding, fuzzy matching or numeric parsing/reformatting is performed. Apply the identical rule symmetrically to both sources; compare canonical identifiers with canonical identifiers. This rule does not apply to Region labels, Industry labels, occupation labels, source titles, worksheet names, descriptive text or numeric cells.

Processing order for the audit is raw cell text → ATECO code identifier parsing → canonical code string → structural/code validation → hierarchy interpretation → state-set construction → partner recognition → relation construction → exact relation deduplication → compatibility audit. All code syntax recognition, hierarchy processing, descendant lookup, partner lookup and relation code fields use the canonical identifier; raw text remains auditable and cannot affect code identity. If edge U+0020 removal produces an empty identifier where a code is required, fail input validation under the existing missing/invalid-code semantics; do not interpret it as zero, invent a code, use a title as a code or default to another Industry. The existing M=9 unmatched-record semantics below remain unchanged.

Within each source code column, one canonical identifier denotes one semantic code identity. If multiple source rows canonicalize to that identifier, inspect their compatibility under the existing D2-B1 hierarchy/table semantics; never silently select a row by input order. Compatible repetition is permitted only where those existing semantics allow it. Conflicting code meaning, title, hierarchy or required audit role fails input validation. Repeated X3 codes with distinct correspondence partners remain governed by the existing many-to-many relation semantics. Retain the raw-row provenance of compatible repetitions. The three RETRY1 identity fixtures authorize no conflicting duplicate.

X3 audit procedure (classification only): use X3 sheet `ATECO 2025 vs ATECO 2022`, exact data rows2–6746, columns B/F for2025/2022 codes, D/H for their respective TOTALE/PARZIALE coverage, I for hierarchy, M for provenance, and N for the update annotation. Ignore styled rows outside the fixed rectangle. For matched rows (M=1,2,4), require both code strings, valid hierarchy and both coverage modalities; retain both directional flags and provenance. M=4 is an explicit European/national interpretation warning, not an allocation coefficient. For M=9, require a valid2025 code, blank2022 code and no coverage pair, then emit an UNMATCHED audit record with an empty state set; do not treat this officially unmatched code as malformed or assign other-services. The pinned snapshot has3 M=9 records (46.89,46.89.0,46.89.00) and32 M=4 matched records, SOURCE-BACKED [X3]. Read X2 sheet `Struttura Ateco 2007 agg 2022`, code column A and title column B; ignore the header and wholly blank rows. Build the2022 code-state sets from its leaf codes (six digits with dots): section membership uses official division ranges A01–03,B05–09,C10–33,D35,E36–39,F41–43,G45–47,H49–53,I55–56,J58–63,K64–66,L68,M69–75,N77–82,O84,P85,Q86–88,R90–93,S94–96,T97–98,U99. Apply the §7 ordinary section state to each leaf, with45.20 descendants→other-services,45.40 descendants→{commerce,other-services} because this audit conservatively treats motorcycle sale/repair as mixed,75 descendants→health-and-social-care and80 descendants→public-administration-and-safety. Other G/M/N leaves use commerce/business/business respectively. For each non-leaf2022 code, including section letters, take the union of its descendant leaf sets; do not discard a repair/veterinary/security exception at an ancestor. For a2025 endpoint its audit state set is the union of the2022 sets over **all** its X3 correspondence partners, never its new section letter. Deduplicate exact(code2025,code2022,coverage2025,coverage2022,hierarchy) relations using canonical code identifiers for both code fields, sort that tuple lexicographically, and report both sets, whether the new union adds another state, whether either coverage is PARZIALE and the update annotation. A missing2022 code outside an M=9 record, an unrecognized matched partner, invalid coverage/provenance or an empty state set for a matched record fails the audit. M=9 records are listed separately and never included in matched-pair counts or production input. This set-valued report determines no production allocation and never supplies many-to-many economic weights.

Crosswalk versions fixed here: `SW-IND-ATECO22-v0`, `SW-JF-ISCO08-MAJOR-v0`, `SW-JL-RESP-v0`, `SW-EDU-QUAL24-v0`, `SW-ACT-LFS24-v0`, `SW-REG-IT-v0`. Versions are authored D2-B1 identifiers, not official classification releases. D2-B2 expands and hashes these exact rules; it cannot edit a coefficient or precedence.

## 7. Crosswalk inventory

Probabilistic rows below specify vectors in the canonical state order in §3. D2-B2 expands each nonzero vector coefficient into a separate crosswalk entry. Omitted components are intentional crosswalk zeros, not semantic invalidity. Source statistical masses are split offline; no categorical source label, source Gender or crosswalk RNG survives at runtime.

| Source code/category | Source version | Frozen state | Mapping type | Rule | Ambiguity | Resolution | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A | ATECO2007 update2022 | agriculture | DIRECT | Agriculture/forestry/fishing | None at section level | Source coverage augmentation is separately assumed | X2; frozen Industry |
| B,C,D,E | Same | production-and-utilities | AGGREGATED | Sum four enterprise counts | None | Industrial equipment servicing stays here | X2/B1 |
| F | Same | construction | DIRECT | Fixed-structure construction/installation | Installation vs servicing | 43.2 fixed systems here; industrial machinery repair 33 in production | X2; frozen Industry |
| G | Same | commerce / other-services | PROBABILISTIC-OFFLINE | For selected whole-section enterprise count: .94 commerce,.06 other-services | Vehicle/motorcycle repair mixed with sales | .06 is MODELED ASSUMPTION; source code45.20 repair→other,45.40 mixed sale/repair→.5/.5 in classification audit only | X2/B1; modeled split |
| H | Same | transport-and-logistics | DIRECT | Transport/storage | None | Logistics Company function, not worker's JobFamily | X2 |
| I | Same | hospitality | DIRECT | Accommodation/food services | None | Hospitality preparation is still service JobFamily | X2 |
| J | Same | information-and-communication | AGGREGATED | Information, publishing, software, communications | Software publishing vs arts | J stays information; live creation/performance R stays arts | X2 |
| K,L | Same | finance-and-real-estate | AGGREGATED | Finance/insurance/real estate | K absent in selected detailed SBS | Preserve observed L; add assumed K mass §11 | X2/B1 |
| M | Same | professional-and-business-services / health-and-social-care | PROBABILISTIC-OFFLINE | Selected M enterprise count: .985 business,.015 health | Veterinary activity75 has specific care function | All identifiable75→health in classification audit; .015 aggregate split assumed | X2/B1 |
| N | Same | professional-and-business-services / public-administration-and-safety | PROBABILISTIC-OFFLINE | Selected N enterprise count: .98 business,.02 safety | Private security80, mixed support | Identifiable80→safety in audit; .02 aggregate split assumed | X2/B1 |
| O | Same | public-administration-and-safety | DIRECT | Public administration/defence/protection | Missing non-market universe | Explicit augmentation §11 | X2 |
| P | Same | education | DIRECT | Teaching institutions | Non-market omissions | Explicit augmentation §11 | X2 |
| Q | Same | health-and-social-care | DIRECT | Healthcare/social care | Non-market omissions | Explicit augmentation §11 | X2 |
| R | Same | arts-and-recreation | DIRECT | Arts/sports/recreation | Worker function differs | Athlete/coaching split is occupational, not Industry | X2 |
| S | Same | other-services | DIRECT | Selected known personal/repair/membership service count | Detailed activities remain known services | No residual or unknown-code allocation | X2/B1 |
| T | Same | other-services | CALIBRATION-ONLY | Household-employer classification audit only | Household unit differs from Company | No observed enterprise count or runtime household variable | X2 |
| U | Same | public-administration-and-safety | CALIBRATION-ONLY | Extraterritorial public institutions audit | Not selected business population | No production observed mass | X2; modeled organizational proxy |
| B–E / F / G–I / other | ASIA2023 broad groups | EmployeeCount industry-law group | CALIBRATION-ONLY | Assign each Industry to group §12 | No detailed Industry×size evidence | Fixed grouping; independent A,K,O/non-market headcounts assumed | B1 |
| ATECO2025↔2022 partial/many-to-many pairs | X3 update2026 | Classification audit states only | CALIBRATION-ONLY | Compare direct old-code rules; no production conversion | Cross-state and mixed-code relations | Enumerate all such pairs, never use operational recode as shares | X3 |
| X3 M=9 /2025 code without2022 counterpart | X3 update2026 | No frozen assignment | REJECTED | UNMATCHED classification-audit record only | Officially absent correspondence | Keep empty audit state set; no production mass and no input-failure substitution | X3 |
| Blank/unknown/invalid Industry code | Any | No state | REJECTED | Abort input audit | No generic residual concept | No default to other-services | Frozen taxonomy |

Occupation crosswalk, MODELED ASSUMPTION coefficients throughout ambiguous majors:

| Source code/category | Source version | Frozen state | Mapping type | Rule (ten-component vector) | Ambiguity | Resolution | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| OC1 managers | ISCO-08 | Function vector | PROBABILISTIC-OFFLINE | [0,0,0,0,.20,.20,.15,.45,0,0] | Strategic vs managed occupational function | Function distribution assumed; no automatic JobLevel assignment | O1–O3/X1; frozen function semantics |
| OC2 professionals | Same | Function vector | PROBABILISTIC-OFFLINE | [0,0,0,0,0,0,.45,.20,.35,0] | Scientific, creative/legal/advisory, clinical/teaching | Named functions supersede generic profession | O1–O3/X1 |
| OC3 technicians/associate professionals | Same | Function vector | PROBABILISTIC-OFFLINE | [.05,0,0,0,.15,.15,.43,.02,.20,0] | Clinical technicians, finance, coaching, athletics | .02 creative includes competitive athletes; care includes instruction | O1–O3/X1; modeled composition |
| OC4 clerical | Same | Function vector | PROBABILISTIC-OFFLINE | [0,0,0,.03,.97,0,0,0,0,0] | Reception belongs to service | .03 reception/service share assumed; code4226 audit→customer service | O1–O3; limitation explicit |
| OC5 service/sales | Same | Function vector | PROBABILISTIC-OFFLINE | [0,0,0,.50,0,.38,0,0,.08,.04] | Sales vs reception/personal care vs protection | Four-way assumed mixture | O1–O3 |
| OC6 agricultural skilled | Same | Function vector | PROBABILISTIC-OFFLINE | [.97,.03,0,0,0,0,0,0,0,0] | Craft servicing in agriculture | .03 servicing share assumed; servicing-specific duties→skilled trades | O1–O3 |
| OC7 craft/trades | Same | Function vector | PROBABILISTIC-OFFLINE | [.15,.85,0,0,0,0,0,0,0,0] | Food/craft production vs installation/repair | .15 production assumed | O1–O3 |
| OC8 operators/assemblers | Same | Function vector | PROBABILISTIC-OFFLINE | [.60,.10,.30,0,0,0,0,0,0,0] | Production operation vs driving vs trade work | Fixed assumed mixture | O1–O3 |
| OC9 elementary | Same | Function vector | PROBABILISTIC-OFFLINE | [.55,.05,.12,.25,.03,0,0,0,0,0] | Labour/logistics/cleaning/office support | Fixed assumed mixture | O1–O3 |
| OC0 armed-forces occupations | Same | protective-services | DIRECT | [0,0,0,0,0,0,0,0,0,1] | Medical occupational function is separate | Clinical code22 audit→care, not protection merely because military employer | O1–O3; frozen function |
| NRP/no response | Same | All ten | PROBABILISTIC-OFFLINE | Observed NRP, or expressly modeled O1/O2 residual, split uniformly .10 per family | NRP cells can be missing | §8 completion/reconciliation; distinguish observed, residual and absent-modeled-zero masses | O1–O3 |
| CP2011/CP2021 detailed categories | Official CP vintages | Audit only | CALIBRATION-ONLY | X1 relations inspected; no production counts converted | Many-to-many duties | Never copy CP label into ISCO-major distribution | X1 |

The classification audit uses function rules, without importing additional occupation counts: ISCO21/25 scientific/engineering/computing→technical;22/23 clinical/teaching→care; routine accounting/records→administration; financial/legal/advisory knowledge work→professional; selling/negotiation→sales; hospitality/reception/practical service→customer service;54 protection→protective. **3421 competitive athletes→professional-and-creative;3422 coaches/instructors→care-and-education;3423 fitness/recreation instruction→care-and-education when instruction is the principal function.** These audit examples do not claim the major mixtures are empirically identified. OC4/OC6 approximations are reviewed as aggregate calibration and cannot certify each individual detailed occupation.

Responsibility crosswalk:

| Source code/category | Source version | Frozen state | Mapping type | Rule | Ambiguity | Resolution | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| No delegated team/unit authority | Frozen responsibility definition | individual-contributor | DIRECT | Occupational function alone conveys no authority | None | Highest actual responsibility tier controls | Contract3 |
| Daily team/workstream supervision | Same | supervisor | DIRECT | Day-to-day delegated supervision | Manager may supervise too | Higher objective/resource authority→manager | Contract3 |
| Unit/function objectives/resources | Same | manager | DIRECT | Delegated unit authority | Major strategic division | Strategic organization/major-division governance→executive | Contract3 |
| Strategic organization/major-division governance | Same | executive | DIRECT | Highest responsibility tier | Titles misleading | Duties control | Contract3 |
| Operai/impiegati/apprentices | INPS/ASIA qualifica2023/24 | No direct tier | CALIBRATION-ONLY | Administrative grade, not responsibility | Can supervise/manage | Do not map deterministically | A1/W2 |
| Quadri/dirigenti | Same | No direct tier | CALIBRATION-ONLY | Senior grade proxy | Quadri can be IC; executive not every dirigente | Use only prior plausibility audit | A1/W2 |
| ISCO OC1, skill levels | ISCO-08 | No direct tier | REJECTED | Function/skill taxonomy is not authority | Managers heterogeneous | §16 modeled responsibility prior | O1/X1 |

Education crosswalk:

| Source code/category | Source version | Frozen state | Mapping type | Rule | Ambiguity | Resolution | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| No completed primary / ISCED-A below1 | ISCED2011/qualification2024 | below-primary | DIRECT | Highest completed attainment | Programme participation differs | Completion only | X4; frozen Education |
| Completed primary / level1 | Same | primary | DIRECT | Highest completed attainment | None | No enrolment substitution | X4 |
| Completed lower-secondary / level2 | Same | lower-secondary | DIRECT | Same | None | Completed vocational equivalents by source group | X4/E1 |
| Level3 | Same | upper-secondary | DIRECT | Same | Vocational qualifications differ | Source grouping respected | X4 |
| Level4 | Same | post-secondary-non-tertiary | DIRECT | Same | Italian grouping merges3/4 | Split §13, not assumed exact source separation | X4/E1 |
| Levels5–7 | Same | tertiary | AGGREGATED | Short-cycle, bachelor, master equivalents | None for selected group | No doctoral conflation | X4 |
| Level8 | Same | doctoral | DIRECT | Completed doctorate | Enrollment not completion | Separate modeled fraction when source5–8 merged | X4 |
| E1/A1 “al più secondario inferiore” / ED0-2 | 2024/ISCED2011 | First three states | PROBABILISTIC-OFFLINE | Six age-specific vectors §13 | Below-primary/primary unresolved | Authored splits explicitly assumed | E1/A1 |
| “secondario superiore” / ED3_4 | Same | upper-secondary/post-secondary | PROBABILISTIC-OFFLINE | Age-specific post-secondary fraction §13 | Level3/4 unresolved | Authored split | E1/O3 |
| “terziario” / ED5-8 | Same | tertiary/doctoral | PROBABILISTIC-OFFLINE | Age-specific doctoral fraction §13 | Level8 unresolved | Authored split | E1/O3 |
| Education missing | Selected response | No observed state | REJECTED | Selected O3 NRP handled statistically §15 | Unknown attainment | No runtime unknown state | Frozen seven states |

Activity crosswalk:

| Source code/category | Source version | Frozen state | Mapping type | Rule | Ambiguity | Resolution | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| LFS dipendente main job | LFS2024 | employee before principal-role reconciliation | CALIBRATION-ONLY | Paid dependent employment component | Student/retired workers may have other primary role | Explicit transitions §14 | A1 |
| LFS independent / O2 SELF | Same | self-employed before reconciliation | CALIBRATION-ONLY | Business/professional working role | Unpaid family helpers in broad independent count | .02 broad-independent fraction→inactive, assumed; not turnover | A1/O2 |
| ILO unemployed | Same | unemployed before reconciliation | CALIBRATION-ONLY | Seeking/available/non-employed definition | Principal student/retired status differs | Explicit transitions §14 | A1 |
| LFS inactive | Same | student/retired/inactive mixture | PROBABILISTIC-OFFLINE | Age splits §14 | Inactivity is not a principal-role taxonomy | Fixed split, then principal-role reconciliation | A1 |
| Principal current student | PL032 category5, operation2021+ | student | DIRECT | Principal self-description if observed | Can earn/search | Selected production aggregates do not observe it; modeled transition only | M1; frozen ownership |
| Principal retired | PL032 category3, operation2021+ | retired | DIRECT | Same | Can earn or be inactive | Same | M1 |
| PL032 employed1 | Same | employee/SE | CALIBRATION-ONLY | Cannot separate employee/SE alone | Position required | Not selected as production field | M1 |
| PL032 unemployed2; unable4/domestic6/other8 | Same | unemployed; inactive | AGGREGATED | Principal status concept | Military category7 may be employment | Category7 rejected as direct crosswalk; no production data from it | M1 |
| Old PL031 modalities | pre2021 EU-SILC | No production state | REJECTED | Different code/version | New PL032 has changed modalities | Never reinterpret old codes as new | M1 |

Geography, `SW-REG-IT-v0`: ISTAT Region codes below are authoritative input labels; NUTS aliases are an audited aggregation convenience. NUTS2021/2024 codes listed have the same selected Region aggregation. Source-label pipeline: raw label → Unicode NFC → replace curly apostrophes U+2019 with ASCII apostrophes → collapse whitespace → trim → apply the finite exact source-label alias table → exact frozen Region lookup. The table gains only the Trentino entry below; all existing explicitly approved aliases/rules remain unchanged. Accept exactly the frozen Region labels, the listed official codes, and Bolzano/Bozen or Trento as the two Trentino child aliases. No fuzzy label matching or guessed geography is permitted. If both the combined Trentino Region and its two autonomous-province children are present, use the combined row and exclude both children. If only children exist, sum their counts first, then form proportions. No equal average of provincial percentages.

Exact source-label alias addition, applied after generic normalization and before exact lookup:

| Exact normalized source label | Frozen Region label |
| --- | --- |
| `Trentino-Alto Adige/ Südtirol` | `Trentino-Alto Adige/Südtirol` |

This exact entry recognizes A1 `8.1 segue2!A24`, raw label `Trentino-Alto Adige/\nSüdtirol`. It is not a regular expression or a generic punctuation/whitespace rule: `Trentino-Alto Adige / Südtirol` remains rejected. Canonical spelling, Region order, numeric cells and parent/child precedence are unchanged.

| Source code/category | Source version | Frozen state | Mapping type | Rule | Ambiguity | Resolution | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 13 / ITF1 | ISTAT Region; NUTS2021/2024 | Abruzzo | DIRECT | Exact Region | None | NFC label | E1/A1/B1 |
| 17 / ITF5 | Same | Basilicata | DIRECT | Same | None | Same | Same |
| 18 / ITF6 | Same | Calabria | DIRECT | Same | None | Same | Same |
| 15 / ITF3 | Same | Campania | DIRECT | Same | None | Same | Same |
| 08 / ITH5 | Same | Emilia-Romagna | DIRECT | Same | None | Same | Same |
| 06 / ITH4 | Same | Friuli-Venezia Giulia | DIRECT | Same | None | Same | Same |
| 12 / ITI4 | Same | Lazio | DIRECT | Same | None | Same | Same |
| 07 / ITC3 | Same | Liguria | DIRECT | Same | None | Same | Same |
| 03 / ITC4 | Same | Lombardia | DIRECT | Same | None | Same | Same |
| 11 / ITI3 | Same | Marche | DIRECT | Same | None | Same | Same |
| 14 / ITF2 | Same | Molise | DIRECT | Same | None | Same | Same |
| 01 / ITC1 | Same | Piemonte | DIRECT | Same | None | Same | Same |
| 16 / ITF4 | Same | Puglia | DIRECT | Same | None | Same | Same |
| 20 / ITG2 | Same | Sardegna | DIRECT | Same | None | Same | Same |
| 19 / ITG1 | Same | Sicilia | DIRECT | Same | None | Same | Same |
| 09 / ITI1 | Same | Toscana | DIRECT | Same | None | Same | Same |
| 04 / ITH1+ITH2 | Same | Trentino-Alto Adige/Südtirol | AGGREGATED | Bolzano/Bozen + Trento | Parent/children duplicate | Parent precedence as above | Same |
| 10 / ITI2 | Same | Umbria | DIRECT | Same | None | Same | Same |
| 02 / ITC2 | Same | Valle d'Aosta/Vallée d'Aoste | DIRECT | Accept line-break/curly-apostrophe source spelling after exact alias normalization | Bilingual spellings | Preserve frozen exact label in assets | Same |
| 05 / ITH3 | Same | Veneto | DIRECT | Same | None | Same | Same |
| NUTS1 macroareas | Published SES/SHIW/INPS | Constituent Regions only | CALIBRATION-ONLY | NW={01,02,03,07};NE={04,05,06,08};Centre={09,10,11,12};South={13,14,15,16,17,18};Islands={19,20} | Within-area effects unavailable | Authored Region factors §20; no macroarea runtime parent | W1/W2/R1/H1 |
| Municipality/province/urban class/coordinates | Any | No retained state | REJECTED | May aggregate to listed Region offline | Hidden geography | Drop before fitting output; no proxy retained | Frozen geography |

## 8. Reproducible fitting framework

One deterministic pipeline is selected. Software for canonical fitting: **CPython3.13.2**, standard-library `decimal`, `fractions`, `csv`, `zipfile`, XML and `hashlib`; no stochastic optimization. Decimal precision96, ROUND_HALF_EVEN, exponent bounds±999999999, single deterministic process. Numeric input strings are parsed as Decimal, not binary float. Probabilities and algorithm outputs are quantized to 40 decimal places before conversion to exact rational weights. Source row order is replaced by canonical tuple/code order before every accumulation. Compensating a malformed source with an inferred number is forbidden.

The exploratory Company/Employer pilot ran CPython3.13.2 and NumPy2.2.3. It checked the proposals without producing canonical assets. Canonical D2-B2 uses the decimal recipe and checks the recorded proposal values, not floating point pilot bytes.

Published tabulation counts already contain official statistical weighting. Sum total-person numeric counts; do not reweight them as observations or treat expanded population counts as sample sizes. O1/O2/O3 `THS_PER` units cancel in proportions. H1 cross-checks use PESOFIT; never historical PESO, household counts or repeated household weighting as person weights. No restricted sample weight is fabricated.

**Audit parent distribution D0 (MODELED ASSUMPTION):** age uniformly weighted over the 55 integers21–75; Region weights proportional to E1 `7.14 segue` K for exactly20 Region rows (exclude geographic parents/children duplicates); conditional Education from §13; subsequent working branches and levels from the specified DAG laws. This is an offline fitting/validation design distribution, not a change to Layer1's generation. Publish D0 margins and label them modeled. Non-working tuple mass is its D0 joint with activity; employee Income tuple mass sums age/Education out and uses the analytic Employer distribution; SE Income sums Education out. Unauthorized evidence dimensions disappear into these final authorized tables. Report actual Layer1-parent sensitivity separately; no additional runtime parent is introduced.

**KL projection/IPF operator:** positive seed joint q indexed by complete authorized tuple and output state; minimize Σ p log(p/q), subject to exact prescribed parent-tuple masses and listed output margins. Initialize with the seed normalized to1. Each sweep scales parent rows to their fixed masses, then output margins in the declared order. Enumerate cells lexicographically. Stop when maximum absolute margin residual<10^-24 and maximum relative residual for margins≥10^-12<10^-20; maximum50,000 sweeps. Nonconvergence fails production and requires a D2-B1 revision; it never selects another solver or relaxes a margin. Cells of zero D0 mass remain completed by the same positive conditional seed and do not constrain fit. Here modeled D0 gives positive mass to every reachable tuple.

Closed-form normalized-product fits are the unique KL projection to a specified positive row target, not an unspecified regression. Seed-floor construction is exact: Activity uses its six raw age-margin×Education-association×Region-association products; employee/SE JobFamily use their own national family-margin×Education-association×Region-association products; JobLevel uses national-tier-prior×age×Education×family-factor products. In each case add10^-6 to each raw product, normalize that conditional row, then multiply by its prescribed D0 parent mass to form the IPF joint seed. For SE Industry first normalize each M row to a14-state conditional vector, add10^-6 to each state and renormalize, then multiply by the fixed SE family mass before joint raking. Company Industry is direct target normalization and receives no seed floor. Education uses its two expressly defined coarse/final row floors. The IPF constraints do not receive pseudocounts. Each categorical seed receives `10^-6` additive pseudoprobability per state at the location specified in its model section, before normalization; do not add the floor again after IPF, principal-role transport or integerization. Education has two expressly specified locations, the three-state coarse row and the final seven-state row; source effects are shrunk explicitly where prescribed. This prevents accidental fitted zeros. The pseudoprobability is a modeling choice, not a measured rare-category rate. There are no intentional zero final categorical states.

Sparse completion is fixed per model: Education normalized-product half-shrunk regional association; Activity KL/IPF with age margins and explicit overlap transport; employee/SE occupation half-shrunk all-worker education association plus independently fitted branch margins; JobLevel KL/IPF on responsibility priors; SE Industry KL/IPF on its own sector margins; Income ordered Ward pooling plus quantile compression. There is no runtime fallback.

API status treatment: filter the exact listed age/sex/status/Education slices; do not sum overlapping ranges or SELF subcategories. Numeric cells with reliability flags remain observations and flags remain in reports. For an O1/O2 missing selected major OC0–9, use that same source's Y15-74 same-major count divided by its Y15-74 TOTAL, times the selected Y20-64 TOTAL. If both ages lack that major, assign zero **modeled identification mass**, report the missing category and retain final public-family support through the specified positive seed floor; never label the absent count an observed zero. Let T be selected TOTAL and A the sum of completed major counts. If NRP is observed, preserve it; if NRP is missing, set modeled unresolved response mass r=max(0,T−A), report it as assumed residual, not official NRP. Rescale the completed major vector proportionally to T−r and allocate r uniformly to ten families; require T>0, A>0,0≤r<T. This explicitly reconciles published rounding. In the pinned O1 slice NRP is absent and the known-major sum18392.5 differs from TOTAL18392.6 by .1 thousand (SOURCE-BACKED [O1]); that .1 is modeled residual, not evidence of exactly100 nonrespondents. O2 also has missing NRP, with no residual, and OC0 is missing in both age slices.

For O3, retain exactly three Education groups. Selected missing major cells use the Y_GE15 within-Education major/TOTAL ratio, times the selected group's TOTAL. If both ages miss it, use that selected group's TOTAL times the major marginal from observed O3 Y20-64/isced11=TOTAL OC0–9 cells, normalized over observed majors; an unobserved major marginal has zero modeled identification mass with a missingness report. Missing O3 NRP is zero modeled identification mass, never an observed zero; observed NRP is allocated uniformly. Rescale completed majors proportionally to each group's TOTAL minus observed NRP before forming the family row. This is the precise proportional discrepancy treatment referenced in §15. Required TOTAL or an entirely empty major marginal fails input. The pinned O3 NRP cells are absent and are reported. Source-internal completion introduces no new source, Gender correction or runtime fallback.

**Precision and fit sequencing:** source count vectors are normalized by the sum of their retained cells, independently of any rounded published total. All numerical IPF margins must have the same total1 before fitting. Any source total discrepancy is reported; no incompatible rounded totals are imposed simultaneously. Decimal96 operations remain unquantized through bisection/IPF/profile optimization; quantize their final row probabilities to10^-40 exactly once, normalize that rational vector to1, then apply §9. G-polyline values and H log-quantile vectors are quantized to10^-40 after their final evaluation, not at every iterative step. The scalar tuple profiles are exact products of the authored finite-decimal factors; compare/group them as exact rationals. Use the fitted pre-integer categorical and Company distributions throughout D0, cap fitting, national transport and allocation; recompute a final DAG audit using canonical integer rows only after all assets are authored. Thus downstream fitting does not depend on an unspecified choice of pre/post-integer parents. Each preceding model fit is completed before a descendant consumes its rows. No post-fit floor is added.

**H1 diagnostic measurement:** join unique(NQUEST,NORD) keys and sort by that pair; fail on duplicate/missing join keys, invalid age or negative/missing PESOFIT. For each of YL1,YTP1,YM,YTA2 report observed-weight share, weighted zero/negative/positive rates, signed observed mean and positive-only q10/q50/q90/q99. Positive-subset weights are renormalized within that component, not within a branch. A weighted quantile is the first ascending observed value whose cumulative weight reaches the requested fraction; ties aggregate by exact value. These historical net component diagnostics are not fitted personal all-source margins. Use the component definitions/exclusions of §20.

## 9. Integerization

All ordinary categorical and finite-law rows use **positive-reserving Hamilton apportionment**. Parse normalized fitted p at 10^-40 precision; intentional support is declared before quantization. Let M=2^32−1 and k be the number of intentionally positive entries. Reserve1 for each; allocate the remaining M−k proportionally to p using exact rational arithmetic: floor quotas, then assign residual units to largest fractional remainders, ties by canonical state/support order. Intentionally positive probabilities smaller than10^-40 are represented for allocation by10^-40 and renormalized; they retain their reserved1. Intentional zeros get exactly0 and are never reserved.

Check all support values, each weight and aggregate additions before narrowing; individual weights≤M, law total≤uint64. Sort finite supports, merge equal values with checked uint64 addition, drop intentionally zero-weight pairs, divide all positive weights by their GCD, then verify each canonical weight fits uint32. Categorical laws preserve zero entries in their frozen positions and GCD all positive entries. Recompute exact probabilities after GCD. This does not change support or probabilities.

Acceptance: total variation≤3×10^-7 per law, maximum absolute entry error≤1.3×10^-7; categorical maximum entry error≤10^-8. Report mean and quantile change separately; EmployeeCount mean change≤.01 headcount, Income mean≤1 EUR from integerization. A point forced positive is not called an observed probability; fitted-zero, semantic-invalid and rounding-zero are three separate audit categories. Tiny mass cannot vanish by rounding.

Employer factors are an exception required by D2-A: §19's uint16 I/S and C are exact authored magnitudes; **no GCD or scale normalization of Employer factors or intervals** is performed. Whole-EUR support quantization uses Contract5 **ceiling**, not half-even rounding, after checking0≤x≤MaxUint32. Half-up in §19 is only factor fitting, not public Income rounding.

## 10. CompanyCount

**Selected CompanyCount=65,536 — MODELED ASSUMPTION.** It is synthetic support, not an estimate of Italian enterprise count and not mechanically derived from Person population.

| Candidate | Industry/size usefulness | Employer repetition and tail | Employer raw/index charge at all-positive support | Decision |
| --- | --- | --- | ---: | --- |
| 32,768 | Strong common Industry coverage; fewer large firms and niche combinations | More employer repetition, fewer convincing large entities | 20 MiB +1344 bytes | Rejected: avoidable loss of size diversity |
| **65,536** | All14 states well represented; smallest proposed Industry≈.98%, about644 companies in expectation | About66 large companies under selected mixed laws; substantial repetitions in large synthetic populations | **40 MiB +1344 bytes** | Selected; useful fake-source universe and ample resource headroom |
| 98,304 | More diversity, modest incremental utility | Less repetition; resource cost rises50% | 60 MiB +1344 bytes | Rejected: consumes most64 MiB index capacity |

Expected Company counts and tail counts are projections from assumed source-calibrated laws, not guaranteed census counts. At the selected mixture the expected250+ count is approximately66.20; expected n≥10000 is1.324 and expected n≥65536 is.03310 (MODELED ASSUMPTION projection). The corresponding absence risks are approximately26.6% and96.7%; extreme giants are legal rare support, not dependable fake-source examples in every universe. These risks must be reported rather than described as guaranteed upper-tail representation. Company sampling is not stratified, forced or retried. All-zero Company realization remains the frozen constructor failure, even though its probability under .03 zero mass is negligible. §31 reserves resources for N positive companies, not the expected .97N.

## 11. Company Industry

Target concept: **enterprise-count composition extended to a synthetic organization universe**. B1 detailed FRAME-SBS enterprise counts are the compatible counting unit within their covered market population. Employee-weighted sector shares are rejected for this law: Employer already multiplies by workforce and compatibility. Synthetic augmentation for excluded agriculture/finance/public/non-market activities is explicit.

SOURCE-BACKED [B1] input: `C14_Tavole.xlsx`, sheet14.8 B19:B34 (2022), total B35=4,472,966. Group B–E into production; F into construction; G,H,I,J into their states; L into finance/real estate; M+N into business; P,Q,R,S into their states. Apply §7 enterprise-count splits to G,M,N. Add masses relative to the **original B35 total**: A .050, K .020, O .010, additional non-market P .008, additional non-market Q .007 (all MODELED ASSUMPTION). Normalize the resulting14 masses. Do not apply the splits to employee counts or compensate absent sectors with employee exposure.

The exact recipe, rather than rounded display, is authoritative. Targets below are **MODELED ASSUMPTION, calibrated by B1/X2**:

| Industry in canonical order | Target percent |
| --- | ---: |
| agriculture | 4.56621005 |
| production-and-utilities | 7.75220345 |
| construction | 10.80784093 |
| commerce | 19.82994227 |
| transport-and-logistics | 2.38600325 |
| hospitality | 6.71935217 |
| information-and-communication | 2.38285904 |
| finance-and-real-estate | 6.44787488 |
| professional-and-business-services | 21.08185141 |
| public-administration-and-safety | .98283227 |
| education | 1.60846003 |
| health-and-social-care | 8.04052658 |
| arts-and-recreation | 1.65762953 |
| other-services | 5.73641414 |

Fit: direct normalized target, ordinary integerization §9. Source2023 four-sector ASIA count shares (8.091536%,11.543647%,31.529715%,48.835102%, SOURCE-BACKED B1 sheet14.1 C21:C24) are a calibration comparison for this B1 use, not a second production constraint: detailed SBS omits K and augmentation changes the population. Report the decomposed difference rather than rake incompatible totals together. The selected B1 production use remains CALIBRATION; its contextual check does not upgrade evidence.

Validation: authored target error≤.01 percentage point per Industry; realization count deviations≤max(1 percentage point,5 binomial standard errors) in the separate sample audit. Unknown code aborts; no catch-all residual.

## 12. Company EmployeeCount

Frozen meaning, Contract2: **“The modeled Company workforce/headcount at `PopulationAnchorDate`.”** It is independent of realized generated employee occupancy.

Terminology audit:

| Source term | Statistical meaning | EmployeeCount classification | Permitted determination |
| --- | --- | --- | --- |
| Anchor workforce/headcount of same Company domain | Stock persons under frozen Company workforce concept | DIRECT if actually observed; no such selected source | Could identify stock distribution, but unavailable here |
| dipendenti/employees | Dependent employment subset, often annual averages in ASIA | CALIBRATION [A1/B1] | Employee exposure/size association; not total workforce stock |
| addetti/persons employed | Employees plus independent persons; selected register measures annual average | CALIBRATION [B1] | Band frequencies, mean-size and tail-shape prior within covered enterprises |
| annual-average addetti | Average across year, fractional, active-enterprise unit | CALIBRATION [B1] | Cannot identify zero stock, anchor instant, hires/exits or exact stock quantiles |
| FTE | Hours-normalized labour volume | CALIBRATION only; not selected | Cannot become headcount without hours/person information |
| jobs/positions | Employment relationship counts; multiple jobs per person possible | CALIBRATION only | No one-position/one-person equality asserted |

MODELED ASSUMPTION reconciliation: carry annual-average workforce distributions forward as anchor headcount shape, preserve band-conditioned mean addetti as stock-mean targets, and introduce .03 synthetic dormant/zero mass in every Industry law. This does not assert addetti=dipendenti or redefine frozen headcount as employees only. The modeled universe extends active market enterprises.

SOURCE-BACKED [B1] input table14.2 rows14–17, columns C,D,E,F enterprise band percentages; I,J,K,L workforce percentages; O mean workforce. Source first band is0–9; the table's last-band label “oltre250” is assigned to frozen250+ as a MODELED ASSUMPTION at its displayed aggregation boundary.

| Group | Source enterprise proportions 0–9 /10–49 /50–249 /250+ | Source workforce proportions same bands | Source total mean | Derived source mean per band |
| --- | --- | --- | ---: | --- |
| B–E | .8073730449,.1605933984,.0272902240,.0047433327 | .1851647575,.2708281868,.2352902865,.3087167692 | 11.4022407696 | 2.6150156490,19.2289858986,98.3076027464,742.1075338600 |
| F | .9463033603,.0499533733,.0035012205,.0002420458 | .5692995074,.2800836190,.0995255136,.0510913600 | 3.0496618114 | 1.8346875217,17.0991518612,86.6895284030,643.7268217054 |
| G–I | .9400985511,.0541502283,.0049042001,.0008470204 | .4553409521,.2150668558,.1081373014,.2214548907 | 4.3314613678 | 2.0979627517,17.2031366554,95.5084479619,1132.4677453366 |
| Other ASIA services | .9777008606,.0184094194,.0031827401,.0007069799 | .4687034997,.1182101087,.1125358553,.3005505363 | 2.8626035400 | 1.3723137121,18.3812790614,101.2164130435,1216.9469510665 |

Use exact source cells and normalize enterprise/workforce proportions independently, avoiding printed-rounding drift. Derived band mean is total mean×workforce proportion/enterprise proportion. Final targets inherit modeled stock/coverage assumptions.

Fixed Industry group assignment (MODELED ASSUMPTION where detailed data absent): agriculture→other-services group; production→B–E; construction→F; commerce/transport/hospitality→G–I; all remaining Industries→other-services group. Agriculture and public/non-market unit sizes are uncertain, not observed through ASIA.

Support algorithm yields exactly128 authored candidate values per Industry: every integer0–49;32 values `floor(exp(ln50+j(ln249−ln50)/31)+.5)`,j=0..31;46 values `floor(exp(ln250+j(ln131072−ln250)/45)+.5)`,j=0..45. Sort/deduplicate; for this configuration cardinality128. Model/production cap=131,072 headcount — MODELED ASSUMPTION, below normative uint32 maximum; no source maxima are silently truncated into that value.

Final masses: zero .03; positive micro p0−.03; small p1; medium p2; large p3. Positive micro target mean is original0–9 mean×p0/(p0−.03); remaining band means unchanged. This preserves the source-calibrated unconditional mean while acknowledging the assumed zero split.

Within each positive band select one **finite maximum-entropy exponential tilt of the log-uniform support prior**: q(n)∝n^-1 exp(λn), constrain its mean to the stated target, no fitted-zero points. In the large band impose fixed conditional stratum masses .980 on250–9999,.0195 on10000–65535,.0005 on65536–131072 (MODELED ASSUMPTION). Normalize the tilted prior separately within each stratum, multiply by its fixed mass, and solve one common λ for the overall large-band mean. This fixes P(n≥10000|large)=.020 and P(n≥65536|large)=.0005, preserving explicit giant-employer support while retaining the source-calibrated large mean. The constrained mean must lie inside the mass-weighted stratum support hull. The same monotone bisection solves the unique entropy tilt. Initialize λ bracket[−64,64];256 deterministic bisection steps, subtract maximum exponent before normalization. Source target outside support hull is a configuration failure. Body has every small integer; medium/large use logarithmic spacing, with mass determined by the exact mean constraint. No SizeBand is sampled. Integerize the128-point law under §9, retaining intentionally positive tiny tail points. All14 Industry laws are emitted separately even when their numerical content matches.

Checks: band target error≤.01 percentage point, mean≤1%, zero mass≤.01 percentage point, q50/q90/q99 and band-conditioned q90 against fitted targets≤max(1 headcount,2%); tail P(n≥1000),P(n≥10000),P(n≥65536) against the specified fitted law≤.01 percentage point absolute. Unidentified anchor tail is reported as assumption sensitivity, not statistical representativeness.

## 13. Education

Age mapping `Education`: **[21,25,35,45,55,65]**. Buckets end immediately before the next lower bound; last ends75. First uses20–24 source proportions for21–24; last borrows65–89 proportions for65–75. These restrictions are MODELED ASSUMPTION, not observed single-year crops.

SOURCE-BACKED [E1], low/middle/high counts in thousands from sheet7.13 total-person rows44,45+46,47+48,49+50,51+52,53 respectively. The column order for low/middle/high is **H,E,B**, not B,E,H; K is the published total. Sum the indicated age rows within each of those three columns before normalizing:

| Source age | Low | Middle | High |
| --- | ---: | ---: | ---: |
| 20–24 | 395.047 | 2199.299 | 330.810 |
| 25–34 | 1193.275 | 3032.502 | 1950.255 |
| 35–44 | 1837.412 | 3232.752 | 1964.523 |
| 45–54 | 3244.761 | 4055.495 | 1866.582 |
| 55–64 | 4179.139 | 3623.189 | 1227.122 |
| 65–89 | 8983.868 | 3182.859 | 1157.479 |

Region inputs are E1 `7.14 segue` **H,E,B** for low/middle/high, with K as the published total, for exactly20 canonical Regions; aggregate Trentino precedence §7. The all-person Italy reference is row40, H40/E40/B40. Normalize each three-count vector by its own sum, rather than K, to avoid incompatible totals from published rounding. The tertiary share is B/(H+E+B). K is used only for D0 population weights and a rounding audit. Their different age coverage15–89 precludes asserting that these are direct21–75 Region margins.

Selected fit: let d_a be normalized age low/middle/high counts, d_r normalized regional counts, d_N normalized Italy row. Coarse conditional target `p(g|a,r) ∝ d_a,g × (d_r,g/d_N,g)^.5`, additive10^-6 floor then normalization. Half-shrinkage is MODELED ASSUMPTION, avoiding amplification of small-Region sampling noise and age-composition differences. This formula is the full sparse-cell completion; no unobserved age×Region interaction is fitted.

Split each conditional coarse law into seven states using the following fixed coefficients (MODELED ASSUMPTION):

| Education bucket lower | Low→below-primary/primary/lower-secondary | Middle→post-secondary fraction | High→doctoral fraction |
| --- | --- | ---: | ---: |
| 21 | .005,.035,.960 | .030 | .003 |
| 25 | .005,.045,.950 | .030 | .025 |
| 35 | .006,.070,.924 | .025 | .030 |
| 45 | .010,.120,.870 | .020 | .035 |
| 55 | .020,.200,.780 | .015 | .035 |
| 65 | .035,.350,.615 | .010 | .035 |

Remainders of middle/high go to upper-secondary/tertiary. Apply the categorical floor once to the final seven-state row and normalize. Source marginals are diagnostic coarse attainment constraints; final targets are the120 exact normalized-product/split rows, and their D0 Education×age and Education×Region margins. Integerize §9; no Gender.

Lifecycle: completed tertiary rises by mid-twenties; cohorts carry different attainment;55/65 preserve older cohort education. Boundaries25/35/45/55/65 are source-resolution aligned but the choice of aggregating five-year source groups into decades is MODELED ASSUMPTION. Shift each internal bound±1 independently in the sensitivity run; report weighted TV and source coarse-margin change. Acceptance of the production boundary is fixed; sensitivity exceeding2 percentage points weighted TV is a review warning, not a search for a better boundary.

## 14. PrimaryActivity

Age mapping: **[21,25,35,45,55,60,65,70]**. The55–59/60–64 and65–69/70–75 divisions represent the employment-to-retirement transition. These are MODELED ASSUMPTION boundaries;25/35/45/55 match source resolution,60/65/70 provide readable lifecycle transitions without creating statutory eligibility gates.

Raw labour-status targets before principal-role reconciliation use A1 sheet8.2: total employment/total population rows22/43 and unemployment/total row29/43, columns C,D,E for25–34/35–44/45–54. Working split θ=A1 sheet8.3 L26/D26=5,084.948/23,932.264 (SOURCE-BACKED broad independent share). Let e,u be raw employed/unemployed shares, inactive remainder1−e−u; split employment into(1−θ)e employees and θe independent workers.

| Raw age bucket | e | u | Student fraction of inactive | Retired fraction of inactive after student split | Label |
| --- | ---: | ---: | ---: | ---: | --- |
| 21–24 | .43 | .10 | .70 | .001 | MODELED ASSUMPTION; LFS15–24 is too broad |
| 25–34 | 4242.291/6176.032 | 425.477/6176.032 | .15 | .003 | e/u SOURCE-BACKED A1; splits MODELED ASSUMPTION |
| 35–44 | 5371.295/7034.687 | 357.051/7034.687 | .02 | .008 | Same |
| 45–54 | 7060.634/9166.839 | 364.774/9166.839 | .005 | .05 | Same |
| 55–59 | .70 | .030 | .005 | .35 | MODELED ASSUMPTION; paired with60–64 near source55–64 employment .59004573 |
| 60–64 | .48 | .020 | .002 | .70 | MODELED ASSUMPTION |
| 65–69 | .17 | .007 | .001 | .90 | MODELED ASSUMPTION |
| 70–75 | .05 | .003 | .001 | .94 | MODELED ASSUMPTION |

The table plus the exact working split fully defines six raw age margins. Education association seed: for each source age column C,D,E,F and coarse attainment index g=0,1,2 (low,middle,high), use employed row19+g, unemployed row26+g and inactive row33+g. Normalize these three status counts within g, divide by the all-attainment status share obtained by normalizing rows22,29,36, and raise the ratio to .5. Employee/SE inherit employed association; student/retired/inactive inherit inactive association. Ages55–64 use F;65+ borrow F with the additional .5 shrink, thus a total exponent .25 on the source ratio. Ages21–24 use the following modeled likelihood vectors directly, without an additional power: employment[.7,1,1.3], unemployment[1.2,1,.7], non-labour[1.1,1,.8]. Within each coarse Education group share the same association; no individual qualification effect is added.

Region seed: from A1 `8.1 segue2` all-person Region rows, employed E, unemployed F, age14-and-below J, resident M; proxy working-age denominator M−J. Use employed/unemployed/non-labour shares relative to the Italy shares, exponent .5. Its age mixing is CALIBRATION; this is not a direct Region activity margin. All source fine geography disappears. KL/IPF §8 seed includes raw age margins and these two half-shrunk associations. Constraints: fixed D0 age×Education×Region parent masses, and the six raw age margins. No additional incompatible population totals are raked.

Principal-role reconciliation is **offline transport**, not a runtime secondary-activity variable. If principal status were directly observed, that principal response would prevail over a weekly ILO overlap. Selected aggregate sources do not identify such responses, so the following fixed proportions approximate the missing reconciliation (MODELED ASSUMPTION). Row vectors follow the eight age buckets:

- Employee/SE→principal student: [.12,.03,.005,.002,.001,.001,.001,.001]. Multiply by Education group factors [.5 for first3 states,1 for next2,2 for tertiary/doctoral]; cap at .40.
- Remaining employee/SE→principal retired: [.001,.001,.003,.01,.03,.08,.25,.40].
- Unemployed→principal student: [.25,.05,.01,.005,.003,.002,.001,.001].
- Remaining unemployed→principal retired: [.001,.002,.003,.02,.05,.10,.30,.40].
- Before these transitions, .02 of broad independent-worker mass→inactive for unidentified unpaid helpers; this is a population reconciliation assumption, not an observed helper share.

Precedence when overlaps coincide: principal-student allocation first, principal-retired allocation from the remainder, then the original working/unemployed state. Original student and retired masses stay in their principal states. Original inactive mass remains inactive after the explicit raw student/retired split. Student+employee and student+unemployed therefore do not automatically become employee/unemployed. Retired+employee can remain retired; retired+inactive becomes retired only through the explicit principal-retired allocation. Multiple paid jobs use the main represented job's professional position; no second job is retained. Unidentified mixed categories use this exact transport, never a hidden tie-break on Income/Gender.

Final target margins are the transported joint's Activity×age, Activity×Education and Activity×Region; compute and print both raw ILO-compatible margins and reconciled principal-role margins. Every final row is complete and positive, integerized §9. Source class remains CALIBRATION. Shift each internal boundary±1 for sensitivity with source associations held fixed; report≤3 percentage points weighted TV as the warning threshold. No legal pension-age eligibility is encoded.

## 15. Employee JobFamily

Sources O1 employee occupational margins and O3 all-employed Education association; E1 regional attainment calibration. O3 is not employee-only and therefore cannot identify employee conditionals directly.

SOURCE-BACKED input selectors are §5. For O1 compute ten family masses by multiplying OC1–9,OC0 counts by `SW-JF-ISCO08-MAJOR-v0`; allocate NRP uniformly. Result is **MODELED ASSUMPTION calibrated by O1**, since major splits are authored. Proposed national family shares in canonical order are [0.1457916227,0.1216509901,0.0365689462,0.1189420745,0.1787030110,0.0943136370,0.1445715125,0.0399953786,0.1000281635,0.0194346639]; exact source counts/crosswalk rule prevail over display rounding.

O3 Education association: use only ED0-2,ED3_4,ED5-8 (not overlapping general/vocational subsets). For each of these groups, crosswalk its OC counts to ten families; allocate occupation NRP uniformly. Missing NRP and any discrepancy between that group's TOTAL and completed occupational counts are reconciled by the exact source-internal procedure in §8; no absent NRP is silently read as an observed zero. Education NRP is excluded from education-association fitting and its count reported; it is not a public eighth education state. Normalize rows. Seed association `A(g,f)=P_O3(f|g)/P_O3(f)` raised to .5, where P_O3(f) is the count-weighted family marginal over exactly the three retained Education groups after their stated occupation-total reconciliation, excluding Education NRP and all overlapping Education TOTAL rows. Map first3 Education states to low, next2 to middle, last2 to high. This borrowing of structural association is explicit CALIBRATION.

Regional seed factor `R(r,f)=(tertiary_share_r/tertiary_share_Italy)^(k_f/2)` from E1, with k=[−.6,−.4,−.2,−.2,.1,.1,.5,.6,.3,0] (MODELED ASSUMPTION). It uses Region only; no employer/geographic proxy becomes a parent.

Selected fit: KL/IPF §8 with positive seed branch family margin×A×R; fixed D0 employee Education×Region masses and independently identified O1-derived national ten-family margin. No Company or JobLevel enters these laws.140 rows complete; integerize §9. Validate family×Education/Region against the exact fitted joint≤.5 percentage point per cell and monotone broad qualification effect, not a claim of survey representativeness.

## 16. Employee JobLevel

Age mapping **[21,25,35,45,55,65]**. Age/Education/JobFamily only. No Region, Company, Employer, Income, size or Gender.

Direct responsibility observations adequate for these full grids were not found in accessible selected tables. A1 administrative qualifica and W2 senior grades are CALIBRATION proxies; OC1 managers is not JobLevel. The entire tier prior and conditional effect structure below are **MODELED ASSUMPTION**. Uncertainty is material and explicitly survives review.

National target [individual-contributor .830, supervisor .090, manager .065, executive .015]. Administrative quadri/dirigenti aggregate shares are a plausibility check only, never equal .080 managerial/executive authority.

Age seed factors (IC factor1 throughout):

| Lower bound | Supervisor | Manager | Executive |
| --- | ---: | ---: | ---: |
| 21 | .45 | .20 | .08 |
| 25 | .75 | .45 | .20 |
| 35 | 1.10 | .90 | .65 |
| 45 | 1.25 | 1.30 | 1.20 |
| 55 | 1.20 | 1.45 | 1.65 |
| 65 | 1.00 | 1.30 | 1.70 |

Education seed factors, canonical seven rows, IC1: supervisor[.6,.7,.8,1,1.1,1.2,1.2]; manager[.35,.45,.6,1,1.15,1.5,1.6]; executive[.25,.3,.45,1,1.1,1.8,2]. JobFamily seed factors, canonical ten rows, IC1: supervisor[1.2,1.2,1.1,1.1,1,1,1,1,1,1.2]; manager[.7,.7,.8,.7,1.1,1.2,1.2,1.4,1,1.1]; executive[.5,.5,.7,.5,1.2,1.2,1.4,1.8,.8,1.1].

Selected fit: multiply national prior by the three seed factors and categorical floor. KL/IPF fixed D0 employee age×Education×family masses (Region marginalized before this model), national four-tier target only. Complete420 rows; integerize §9. This is a regularized responsibility prior fit, not an ISCO regression. Validate exact margins, increasing managerial/executive odds through middle life and nonzero unusual valid tiers. Boundaries±1 sensitivity warning2 percentage points weighted TV; national supervisor/manager/executive priors±25% tested separately without changing production.

## 17. Self-employed JobFamily

Independently fitted sources: **O2**, not O1; common O3 Education associations are borrowed with exponent .5, and E1 regional factor uses the same explicit k vector as §15. O2 uses wstatus=SELF only; never sum SELF+SELF_S+SELF_NS.

National ten-family target: exact O2 age20–64 OC counts × the fixed major crosswalk plus uniform occupation NRP, normalized. This is a **MODELED ASSUMPTION calibrated by O2**. The proposed ten-family margin in frozen order is [.1302688756,.1400393808,.0115548615,.1004478997,.0682547981,.1214358139,.1934023628,.1020688485,.1253082564,.0072189028] (MODELED ASSUMPTION calibrated by O2; displayed rounding only). The exact selected O2 counts and crosswalk arithmetic are authoritative. These are independently fitted self-employed family weights.

Selected fit: positive seed O2-derived target×O3 half-shrunk Education association×E1 half-shrunk regional association; KL/IPF with fixed D0 self-employed Education×Region parent masses and the O2-derived national family margin.140 complete rows; integerize §9. Borrowing never copies an employee row, residual or correction. Differences from employee margins are reported explicitly. Source age20–64 extrapolation to older/younger represented workers is MODELED ASSUMPTION collapsed into Education×Region; no age parent is added. Validate branch family margin≤.5 percentage point and Education/Region fitted-joint cells≤.5 percentage point.

## 18. Self-employed Industry

Primary sector ownership is **A1 LFS independent persons**, not Company enterprise count. SOURCE-BACKED [A1] broad independent counts, sheet8.3 L19,L20,L21,L23,L24, normalized by the **sum of those five exact cells**: agriculture351.792; production441.425; construction537.860; commerce/hospitality1431.497; other services2322.375 thousand persons. They identify five broad margins, not a14-state JobFamily joint. L26 is an audit total, not the normalization denominator: the five displayed counts sum to5084.949, whereas L26 displays5084.948 thousand persons. Dividing by L26 would make the prescribed margin sum differ from1 by approximately1.9666×10^-7 and prevent the10^-24-residual IPF fit from converging. Record this source rounding discrepancy; do not change a count to force agreement.

Fixed crosswalk split (MODELED ASSUMPTION): commerce/hospitality mass .70 to the G component and .30 hospitality; G component .94 commerce,.06 repair→other-services. Other-services mass is split across H,J,K/L,M/N,O,P,Q,R,S with proportions [.06,.12,.10,.40,.005,.06,.19,.045,.02]. These sum1. Veterinary/security detailed cases are already absorbed into this explicitly authored broad functional mix; no incompatible enterprise-weighted M/N split is imposed on person counts.

This yields an exact14-state independent-person target from the five source counts. Selected fit: KL/IPF on ten-family×14-Industry joint with fixed independently fitted SE family masses and this national SE Industry margin; seed `M[f,i]` from §19 (positive compatibility profile), not Company Industry law. Report borrowing of relative functional compatibility as MODELED ASSUMPTION; no Employer factor fitted residual is borrowed.10 rows,140 weights; integerize §9. Validate five source broad margins (before documented repair/other-service allocation) and final14-state targets≤.5 percentage point, plus functional compatibility and positive unusual support.

## 19. Employer fitting

**Selected C=4096; I=64×M below; S below.** All factors are authored integer magnitudes, not normalized ratio-equivalent substitutes. C is jointly source-calibrated with the Company model; I's occupational affinities and JobLevel size-profile seed are MODELED ASSUMPTION because accessible evidence does not identify those full conditionals. This uncertainty is not hidden by calling occupational classification “direct employer evidence”.

M, in canonical family rows and14 Industry columns:

| JobFamily | A | Prod | Const | Comm | Trans | Hosp | Info | Fin/RE | Business | PA/safety | Edu | Health | Arts | Other |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| production-and-fieldwork | 16 | 32 | 12 | 2 | 2 | 1 | 1 | 1 | 2 | 2 | 1 | 1 | 2 | 3 |
| skilled-trades | 3 | 24 | 32 | 3 | 3 | 1 | 2 | 1 | 4 | 2 | 1 | 2 | 2 | 12 |
| transport-and-logistics | 2 | 3 | 2 | 4 | 32 | 2 | 2 | 1 | 3 | 2 | 1 | 2 | 2 | 3 |
| customer-and-personal-service | 1 | 1 | 1 | 8 | 3 | 32 | 2 | 2 | 4 | 2 | 2 | 8 | 8 | 16 |
| administration | 2 | 4 | 3 | 8 | 4 | 6 | 8 | 16 | 16 | 16 | 8 | 8 | 6 | 4 |
| sales-and-commercial | 2 | 4 | 3 | 32 | 4 | 8 | 12 | 16 | 12 | 2 | 2 | 3 | 4 | 4 |
| technical-and-scientific | 2 | 24 | 12 | 2 | 4 | 1 | 32 | 8 | 24 | 8 | 8 | 12 | 4 | 2 |
| professional-and-creative | 1 | 2 | 2 | 2 | 2 | 2 | 16 | 16 | 32 | 8 | 8 | 4 | 24 | 4 |
| care-and-education | 1 | 1 | 1 | 1 | 1 | 1 | 2 | 1 | 4 | 4 | 32 | 32 | 8 | 4 |
| protective-services | 1 | 2 | 2 | 2 | 3 | 2 | 2 | 2 | 4 | 32 | 2 | 2 | 4 | 2 |

Thus all140 actual I values are fully specified (64–2048). No cell is zero: protective workers outside safety institutions, care workers outside health, technicians in small service firms and athletes outside arts all remain unusual-valid support.

Actual16 S values, micro/small/medium/large:

| JobLevel | micro | small | medium | large |
| --- | ---: | ---: | ---: | ---: |
| individual-contributor | 37 | 78 | 71 | 82 |
| supervisor | 23 | 78 | 106 | 164 |
| manager | 9 | 49 | 106 | 246 |
| executive | 2 | 20 | 71 | 328 |

Target relationships: functional affinity profile M; increasing relative large-vs-micro exposure with responsibility; employee rather than all-addetti size exposure. SOURCE-BACKED [A1] employee proportions in sheet8.11 rows15–18,H:I:J:K differ materially from B1 addetti exposures. Transform them to the synthetic Industry mix with weights Company Industry p_i×B1 group mean μ_i (MODELED ASSUMPTION coverage bridge). Result target positive-band exposure≈[.2320320270,.2552196083,.1854141722,.3273341924]. It is **MODELED ASSUMPTION calibrated by A1/B1**, not the literal national employee distribution. A1 total employee shares[.2284752137,.2628261411,.1900595088,.3186391364] remain a source-unit check.

Single selected fitting procedure: **deterministic regularized cap-profile calibration**. CompanyCount and the complete Company laws are fixed first. Evaluate their analytic distribution, not empirical enterprise rows. Let `a_i,b(C)=Σ_n in band b p(n|i)min(n,C)`. For each40 condition(f,l), the implied Company state margin is proportional to `p_i a_i,b(C) I_fi S_lb`; normalize separately for each condition. N multiplies all expected weights and cancels from these expected margins, but its finite-universe variation is tested separately. This explicitly includes Company composition/headcount and cap in every evaluation.

Fitting audit condition masses are fixed `O1-derived family margin × [.83,.09,.065,.015]` (MODELED ASSUMPTION independence design for cap calibration). The subsequent DAG-joint sensitivity check uses actual fitted family×level masses; it does not change factors.

Candidate C grid=[64,128,256,512,1024,2048,4096,8192,16384,32768,65535]. I is constrained exactly to64M. Size seed S0=[[64,64,64,64],[40,64,96,128],[16,40,96,192],[4,16,64,256]]. For each C optimize four common band multipliers v with geometric mean1, initialized[1,1,1,1]. Compute overall implied exposure h from40 normalized conditions; update all four v_b simultaneously from that same h, v_b←v_b×target_b/h_b, then divide v by its geometric mean. Stop max|h−target|<10^-24, at most50,000 iterations. This is the unique profile correction for the fixed positive design; failed convergence fails the recipe.

Objective, including all requested margin roles: hard constraints I=64M (functional relative-affinity target), S=S0×v (responsibility relative size target), employee-size exposure matched by profile; among cap profiles minimize **J(C)=Σ_b ln(v_b)^2+.01 ln(C/64)^2**. The .01 regularization is MODELED ASSUMPTION: discourage very high caps used to avoid correcting the source-unit mismatch. Find the global minimum J first; among candidates with J≤minimum+10^-20 choose the smaller C. Initial factors, objective weights and cap grid cannot change in D2-B2. The .01 term is explicit; C is source-calibrated under assumptions, not purely statistically identified.

Pilot profile scores J≈.623 at2048,.588 at4096,.595 at8192: selected4096. The **unpenalized** profile preferred65535. This boundary pressure is a material review signal caused by non-identification of cap vs band correction; it is reported, not silently clipped. The regularized selected cap has16× upper-bound headroom. I maximum2048 is about50% of4095; S maximum328 is about8.0% of4095.

Factor quantization: half-up `floor(S0×v+.5)`, bounds1..4095 checked before acceptance. Baseline scales are exactly64 for I and the printed S0 for S; do not renormalize quantized factors. The listed actual S table is the frozen D2-B1 proposal output; D2-B2 recomputes the trace and checks it reproduces these integers. It may not choose a neighboring cap or repair a factor discrepancy. Integerization-induced exposure TV≤.005; per-band≤.5 percentage point. Exact target family×Industry and level×SizeBand margin/input tables are the40 analytic margins from the **listed C/I/S and Company laws**, with the source-calibrated overall exposure as the independent constraint. Report these modeled relationship targets separately from official margins; they are not claimed to be empirically observed joint tables.

No I×S interaction, Company-size×Industry interaction, geographic factor or Gender factor is added. Check all40 conditions, all Company-domain entries, positive support, zero exclusion, uint64 totals and unusually valid states. Finite N audit uses the mandatory multinomial, absence-risk and ratio-variance calculations in §30; no forced Company strata, SourceID-based identity seed or new runtime index architecture is introduced.

## 20. Income evidence hierarchy

Contract5 meaning: **“Income is the aggregate annualized rate of eligible monetary income attributable to the Person under the income conditions represented at PopulationAnchorDate, before personal income taxes and mandatory personal social contributions.”** It is personal, nonnegative, current annualized gross rate, expressed publicly by ceiling to whole EUR. Personal business earnings are after operating costs, before personal taxes/contributions. Negative business loss does not offset another eligible inflow.

Eligible-source scope includes remuneration, positive personally attributed business/professional earnings, pensions, personal benefits, scholarships, cash interest/dividends/rents/royalties and other eligible monetary income. Family gifts/support, interpersonal transfers, inheritance, capital/wealth, loans, principal returned, capital gains, sale proceeds, imputed rent and non-cash compensation are excluded. No household attribution model, wealth tier, benefit eligibility/history, secondary-job variable or latent component index survives into canonical laws.

The selected accessible official evidence does **not** identify current gross eligible all-source personal rates jointly over every frozen branch/parent grid. This is a limitation of the inspected evidence set, not a claim that every official release has been exhausted. The final targets therefore use an explicit **MODELED ASSUMPTION all-source distribution prior**, checked against compatible personal component evidence. Salary and pension component levels are subordinate calibration, not definitions of Income. Income uncertainty is the largest modeled gap in this v0.

SES audit [W1]:2022, employees paid for the entire October reference month in units≥10 employees, sectionsB–S. Italy includes public administrationO from2014; agricultureA, household/extraterritorial employersT/U and micro units are excluded. Published structural statistics do not cover the whole employee branch. SOURCE-BACKED [W1]: mean gross annual full-year/full-time-equivalent remuneration37,302 EUR; hourly mean16.4 EUR, lower/upper decile boundaries8.8/26.6 EUR; finance25.9 vs hospitality10.9 EUR/hour. The annual statistic includes cash and in-kind remuneration and is standardized to full-time/full-year; the hourly statistic excludes in-kind and nonregular components. No unpublished monetary-only annual subtraction is inferred. Both remain CALIBRATION; neither is a compatible all-source cash Income level. Age/Education/Gender SES effects are not carried into employee Income. [SES report/methodology](https://www.istat.it/wp-content/uploads/2025/01/REPORT_STRUTTURA_RETRIBUZIONI_2022.pdf).

INPS [W2]:2024 workers with≥1 paid day,17,731,002 persons, mean historical annual gross remuneration24,486 EUR and247 paid days (SOURCE-BACKED). Different duration/population explains why it is not equal to a current full-year remuneration rate. Do not annualize by365/247 or treat paid days as hours/FTE. [Official report](https://servizi2.inps.it/servizi/osservatoristatistici/api/getAllegato/?idAllegato=1043).

EU-SILC audit [M1], exact version DocSILC0652025 Updated: personal P/R files differ from household H/D. PY010G is gross employee cash/near-cash; PY050G is signed self-employment benefits/losses and includes conceptually ineligible nonmonetary items; PY080G,PY090G,PY100G,PY110G,PY120G,PY130G,PY140G identify private pensions, unemployment, old-age, survivor, sickness, disability and education benefits. HY040G rents and HY090G investment income are household-level; HY010/HY020 household totals are not Person Income. Gross/net conversion, imputation/top-bottom flags must be inspected before economic use. PL032 replaced PL031 in2021. Historical income reference year differs from survey year; no Italian SUF release is selected here. Restricted/anonymized SUF cannot supply indispensable production targets. [Guidelines](https://ec.europa.eu/eurostat/documents/d/microdata/methodological-guidelines-2025-operation_updated).

SHIW audit [H1]: use annual2022 personal RPER22 components joined to CARCOM22, not RFAM22 household totals. YL1 is net monetary remuneration, YTP1 pensions excluding YTP2 arrears, YM net self-employment, YTA2 scholarships. YL2 noncash, YTA31 gifts/transfers, YCA2 imputed rent, modeled wealth-derived capital aggregates and YTA6 assigned to NORD1 are excluded as direct Person evidence. Y is a net aggregate with incompatible components. Signed YM is CROSS-CHECK only: report negatives and positive-subset quantiles; do not clamp, absolute-value or add it to production Income. PESOFIT weights, age21–75, all genders. These are national personal-component diagnostics only: no household, occupation, qualification, geography or labour-status selection is used, and no branch-specific benefit or all-source Income rate is inferred from them. Join keys NQUEST+NORD (CARCOM `nord`, case-normalized). Missing “.” components are omitted from that component diagnostic with missing-weight share reported; no imputation. [Archive description](https://www.bancaditalia.it/statistiche/tematiche/indagini-famiglie-imprese/bilanci-famiglie/documentazione/documenti/2022/Legen22.pdf).

One numerical Income fitting recipe applies independently to all six branches:

1. Specify branch positive quantile curve Qb below and exact zero mass zb. Qb is piecewise linear in positive probability u between its listed knots, offline only. Values are nominal anchor EUR assumptions, not historical observations.
2. Compute a positive scalar profile m for each reachable authorized tuple from the factor tables below. No unauthorized parent enters m.
3. Under branch-normalized D0 tuple masses d, compute positive-mixture CDF `F0(x)=Σ_t d_t Fb(x/m_t)`; Fb is the exact inverse of Qb, clamped to0/1 outside support.
4. Apply deterministic national quantile transport: tuple positive target `Qt(u)=Qb(F0(m_t Qb(u)))`. Because the source mixture is continuous, its transported positive national distribution is Qb. Zero mass remains zb in every tuple. This freezes concrete branch margins while retaining monotone conditional rankings.
5. Pool Qt by §28; compress by §27; integerize by §9. No parameter, CDF, interpolation, jitter or mixture state survives at runtime.

Shared Region profile (all six branches), **MODELED ASSUMPTION**, calibrated only in direction by official macroarea evidence; fine regional magnitudes are not claimed observed:

| Region | Multiplier |
| --- | ---: |
| Abruzzo | .94 |
| Basilicata | .90 |
| Calabria | .85 |
| Campania | .88 |
| Emilia-Romagna | 1.05 |
| Friuli-Venezia Giulia | 1.02 |
| Lazio | 1.06 |
| Liguria | 1.01 |
| Lombardia | 1.10 |
| Marche | .97 |
| Molise | .90 |
| Piemonte | 1.04 |
| Puglia | .88 |
| Sardegna | .90 |
| Sicilia | .85 |
| Toscana | 1.00 |
| Trentino-Alto Adige/Südtirol | 1.05 |
| Umbria | .96 |
| Valle d'Aosta/Vallée d'Aoste | 1.01 |
| Veneto | 1.04 |

Concrete **positive-conditional** quantile input table, every value **MODELED ASSUMPTION**, supported in scale/context by W1/W2/R1 and component-only H1 cross-checks. Zero is separate; do not misreport positive median as unconditional median.

| u | Employee e | Self-employed s | Unemployed u | Student t | Retired r | Inactive i |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 0 | 1 | 1 | 1 | 1 | 1 | 1 |
| .01 | 4000 | 1000 | 500 | 150 | 4000 | 300 |
| .10 | 16000 | 8000 | 2000 | 600 | 9000 | 1200 |
| .25 | 24000 | 16000 | 5000 | 1500 | 14000 | 3500 |
| .50 | 32000 | 28000 | 10000 | 4000 | 20000 | 7500 |
| .75 | 43000 | 47000 | 16000 | 8000 | 28500 | 14000 |
| .90 | 60000 | 80000 | 23000 | 14000 | 39000 | 25000 |
| .95 | 80000 | 115000 | 32000 | 20000 | 49000 | 40000 |
| .99 | 140000 | 260000 | 60000 | 40000 | 80000 | 85000 |
| .999 | 400000 | 800000 | 100000 | 70000 | 160000 | 160000 |
| 1 | 1000000 | 2000000 | 150000 | 100000 | 300000 | 250000 |

The last row fixes model support caps for the all-source branch transport, not a claim that higher legal Income is invalid. Qt cannot exceed the branch cap. Positive-tail mass and mean are exactly determined by the quantile curve; no Pareto exponent or tail cutoff remains to choose.

All eight actual age mappings, including Income mappings:

| Mapping | Lower bounds | Source IDs / resolution | Lifecycle rationale | Boundary evidence class | Nearby-boundary sensitivity |
| --- | --- | --- | --- | --- | --- |
| Education | 21,25,35,45,55,65 | E1 five-year/65–89 attainment | Qualification completion/cohort differences | MODELED ASSUMPTION aggregation aligned to source edges | ±1 each; warning weightedTV>.02 |
| PrimaryActivity | 21,25,35,45,55,60,65,70 | A1 decade/55–64 status, R1 older-person context | Study entry, work, retirement transitions | MODELED ASSUMPTION; no legal gate | ±1 each; warning weightedTV>.03 |
| Employee.JobLevel | 21,25,35,45,55,65 | A1/W2 grade proxies; weak direct responsibility | Authority accumulates then plateaus | MODELED ASSUMPTION | ±1 each; warning weightedTV>.02 |
| Income[s] | 21,25,35,45,55,65 | H1 age/components CROSS-CHECK; A1 workforce context | Establishment, mature business earnings, taper | MODELED ASSUMPTION; source does not identify current-rate breaks | ±1 each; warning branch median change>3% |
| Income[u] | 21,25,35,45,55,65 | A1 lifecycle; M1 component metadata | Early/mature benefits and other personal inflows | MODELED ASSUMPTION | Same |
| Income[t] | 21,25,30,40,55,65 | H1 scholarship component, A1 study context | Typical study, postgraduate and adult study | MODELED ASSUMPTION;30/40 not source optimization | Same |
| Income[r] | 21,45,55,60,65,70 | R1 recipient ages/annualized pension context | Rare early retirement, transition, mature pension income | MODELED ASSUMPTION | Same |
| Income[i] | 21,25,35,45,55,65 | H1 personal components/A1 lifecycle | Personal benefits/property inflows can occur throughout adulthood | MODELED ASSUMPTION | Same |

Every mapping compiles to the55 supported integer ages. Lower21/upper75 are normative coverage, not fitted boundaries. Test each internal bound at±1 while holding the selected recipe fixed; sensitivity outputs are not new production choices. No mapping exceeds its frozen bucket maximum.

## 21. Employee Income

Parents exactly Region,JobFamily,JobLevel,CurrentIndustry,positive SizeBand. Age/Education/employer identity/raw EmployeeCount/Gender are absent. Zero SizeBand tuples use sentinel65535 and are unreachable.

m is the product of Region profile, family [.83,.94,.92,.73,.98,1,1.19,1.27,1.06,1.07], level[1,1.20,1.65,2.80], Industry[.85,1.03,.96,.90,1.01,.80,1.17,1.24,1.10,1.04,1,1.05,.92,.83], positive size[.90,1,1.08,1.15]. All are **MODELED ASSUMPTION**; W1 calibrates broad relative earnings structure but does not empirically identify the complete tensor. Level effects represent responsibility-associated all-source distributions, not a salary equation.

Zero mass **.002 MODELED ASSUMPTION**. Positive distribution and national all-source targets are §20. The wider right tail allows eligible secondary monetary inflows; component correlation is represented only by the finite all-source prior. No fixed salary+household residual, latent capital tier or hidden secondary-job parent is introduced. The level38–39k implied mean is checked against the meaning and duration of W1/W2, not forced equal to either remuneration mean.

Selected fit is national quantile transport → ordered Ward pool cap512 → budgeted finite quantile compression → integerization. Outputs:56,000 uint16 map entries, reachable44,800;≤512 laws. Report conditional family/level/Industry/size differences and salary-to-all-source uncertainty. No post-anchor2026 wage observations enter targets.

## 22. Self-employed Income

Parents exactly age bucket[21,25,35,45,55,65],Region,JobFamily,Industry. m: Region×age[.60,.85,1.05,1.15,1.10,.90]×family[.70,.90,.85,.72,1,1,1.25,1.30,.95,.95]×Industry[.70,1,.95,.85,.90,.80,1.20,1.35,1.15,.90,.90,1.05,.85,.78]. **MODELED ASSUMPTION** throughout; H1 signed net YM is cross-check only, not gross attributed earnings.

Zero mass **.080 MODELED ASSUMPTION**. Qs positive median28,000 and upper-tail table §20 are explicit assumed all-source rates. Signed business loss is not mapped to abs(profit), unsigned profit or a clamped all-source total. No turnover, receipts, household business resources or employer cost is used directly. Current eligible business earnings after operating costs plus other positive personal inflows are represented by the finite all-source prior; loss cases motivate zero/small values without negative support or offsetting other inflows.

Selected fit: national quantile transport, ordered Ward cap256, quantile compression, §9.16,800 map slots. Report separate source negative-YM rate, positive-YM quantiles and the modeled gap from net historical components to gross anchor rate. Do not infer a net-to-gross tax schedule.

## 23. Unemployed Income

Parents age[21,25,35,45,55,65],Education,Region. m=Region×age[.85,.95,1.05,1.10,1.05,.95]×Education[.70,.75,.85,1,1.05,1.15,1.20]. **MODELED ASSUMPTION**.

Zero mass **.350 MODELED ASSUMPTION**; positive median10,000 and complete Q table §20. Benefits alone are not all-source Income. Personal eligible benefit rates and other monetary inflows can be positive, but no eligibility state, unemployment duration, past wage, pension history or household support enters the final table. M1 supplies component concepts, not Italian benefit-rate estimates. H1 supplies personal-component scope context rather than an identified branch benefit-rate distribution; no new INPS benefit dataset is left for D2-B2 to select.

Fit: national quantile transport, ordered Ward cap64, compression, integerization.840 slots. Validate zero/median/positive quantiles/tails and Education/Region effects; record modeled uncertainty explicitly.

## 24. Student Income

Parents age[21,25,30,40,55,65],Education,Region. m=Region×age[1,1.25,1.35,1.25,1.15,1.05]×Education[.50,.60,.80,1,1.20,1.60,2]. **MODELED ASSUMPTION**.

Zero mass **.650 MODELED ASSUMPTION**; unconditional median is0, positive median4,000. H1 YTA2 offers scholarship-component context; YTA31 gifts/support is excluded, even if often economically important to students. Eligible personal scholarship/royalty/cash investment/secondary remuneration rates are represented through the all-source distribution without secondary employment or institution state. No family resources are allocated per household member. The older-student cells are complete assumptions, not age validity gates.

Fit: national quantile transport, ordered Ward cap64, compression, integerization.840 slots. Validate zero mass, positive central resolution and tails; family-support inclusion is a production failure.

## 25. Retired Income

Parents age[21,45,55,60,65,70],Education,Region. m=Region×age[.60,.80,.95,1.05,1,.95]×Education[.70,.75,.85,1,1.05,1.18,1.25]. **MODELED ASSUMPTION**.

Zero mass **.010 MODELED ASSUMPTION**; positive median20,000 and Qr table §20. SOURCE-BACKED [R1] recipient gross annualized pension mean22,331 EUR at2024-12-31; the benefit mean15,821 EUR is a different unit and is not used as Person level. R1 annualization is13×monthly pension and12×monthly accompanying allowance. It is pension-component evidence, not the whole retired branch or the2026 stock. [R1 report](https://servizi2.inps.it/servizi/osservatoristatistici/api/getAllegato/?idAllegato=1007).

The prior's eligible additional personal interest/rent/dividend/royalty/other monetary income explains a broader all-source distribution without an imposed pension-only law. No household capital is divided among retirees; no pension history or eligibility flag becomes a parent. H1 uses YTP1 rather than arrears-inclusive YTP, and excludes gifts/imputed rent. Very early retired cells are low-confidence modeled support, still semantically valid.

Fit: national quantile transport, ordered Ward cap64, compression, integerization.840 slots. Audit pension-component versus all-source mean, zero rate, positive tail and Region/Education relationships. No automatic pension revaluation is applied.

## 26. Inactive Income

Parents age[21,25,35,45,55,65],Education,Region. m=Region×age[.60,.80,1,1.10,1.15,1.10]×Education[.70,.75,.85,1,1.05,1.15,1.20]. **MODELED ASSUMPTION**.

Zero mass **.550 MODELED ASSUMPTION**; unconditional median0, positive median7,500 and Qi table §20. Positive personal Income is possible through eligible personal pensions/benefits or monetary property/royalty inflows even without working/student/retired principal activity. H1 household-resource/wealth-derived capital assignments do not identify that personal amount and are not used directly. No wealth class, partner, household size, institution, benefit eligibility or employment history becomes hidden conditioning.

Fit: national quantile transport, ordered Ward cap64, compression, integerization.840 slots. Audit all-source uncertainty and personal-only component exclusions; unexplained household-level leakage fails production.

## 27. Income compression

Selected algorithm: **locked-quantile contiguous partition with conditional-mean representatives and deterministic budgeted refinement**. Continuous/polyline operations are strictly offline target construction; final laws are finite whole-EUR pairs.

For each pooled positive law evaluate shared Q at master knots `G=sorted({j/8192:j=0..8192} ∪ {0,.01,.10,.25,.50,.75,.90,.95,.99,.999,1})`. Values are rounded to10^-40 Decimal and represented as exact rationals; shared Q is linearly interpolated between these master knots only during fitting. Zero mass is its separate atom. Initial positive partition boundaries are all11 published positive quantile knots. These boundaries remain locked; no final segment crosses a central/tail target knot.

For a positive quantile interval[a,b], representative is **ceil(∫a^b Q(u)du/(b−a))**, with exact piecewise-linear integration; its probability is(1−z)(b−a). Loss is `(1−z)∫a^b |Q(u)−representative|du`, exactly integrated on the polyline, so zero-heavy branches receive the stated positive-mass adjustment. Split candidates are internal master knots. Choose the split with largest loss reduction, ties by lower u. Refinement builds the complete greedy sequence through512 authored slots (including zero), no random initialization. Keep every partition interval in the authored-slot count, including intervals with equal whole-EUR representatives. At compression output, merge equal representatives by exact rational addition of their probabilities **before** §9 apportionment; Hamilton reserves one unit per distinct intentionally positive value, not per duplicate interval. Preserve the pre-merge authored count for the512/240000 checks and fit manifest. Integer-weight canonicalization and within-branch law deduplication occur afterward. Do not jitter equal representatives apart.

Cardinality allocation is a single fixed rule across all branches:

- Minimum128 authored slots/law (zero included); maximum512. Construct all baseline laws in branch order e,s,u,t,r,i and earliest-member tuple order. If duplicate representatives leave fewer than128 distinct points, continue that law's fixed greedy sequence one authored slot at a time until128 distinct points, charging every extra authored slot to the global budget. Reaching512 without128 distinct points is a validation failure. This baseline completion precedes the top-decile reservation; reservations count actual authored slots, and their sum must fit240000 before any discretionary allocation. No claim that a wide support range alone guarantees128 greedy distinct points is made.
- Law population mass ρ is D0 branch mass×pooled tuple mass. The highest ceil(totalLawCount/10) laws byρ receive at least384 authored slots; ties by branch order e,s,u,t,r,i then earliest canonical tuple.
- Remaining capacity is allocated in blocks of32 slots, selecting the law with greatest `evidenceQuality × (ρ+10^-8) × reduction in normalized W1` for its next32 greedy refinements. Evaluate this score as an exact rational from the G polyline, fixed probabilities and median; exact ties use branch order e,s,u,t,r,i then earliest member canonical tuple. Quality constants[e1,s.8,u.6,t.4,r.9,i.4] are MODELED ASSUMPTION. Normalization denominator=max(1000,positiveMedian). Tail complexity affects the exact W1 gain, zero mass affects positive loss, and population mass prevents high-volume coarse artifacts.
- Stop when no next full32-slot block fits the **240,000 aggregate authored-point budget**, or no law has32 unused slots under its512 authored-slot cap. A law whose duplicate-preserving baseline leaves fewer than32 slots can finish below512; no partial discretionary block is allocated. Law minimum/top-decile reservation is performed first. Without duplicate-induced extra baseline slots, the maximum1024-law baseline and top-decile reservation consume157440 authored slots; any such extras are accounted for before declaring remaining capacity. If required reservations exceed240000, the recipe fails rather than silently reducing a minimum. No law receives512 by default.

No HARD compression tolerance is relaxed to spend less memory. Final refinement can stop below the budget if all512 limits are reached. Budget exhaustion with a failed HARD metric triggers D2-B1 revision, not runtime fallback or interpolation.

Compression acceptance, comparing pooled target to finite law before integerization (HARD except the explicitly identified fine-tail warning; dimensions/severities are enumerated in §30): zero error exactly0; positive q10/q25/q50/q75/q90 relative error≤2%; q95≤3%,q99≤5% HARD; q999≤10% WARNING (fine-tail fidelity); mean relative error≤1% or absolute1 EUR whichever larger; positive IQR and standard deviation≤5%; tail mass at100k/250k/1m absolute error≤.002, evaluated only for thresholds within branch cap. Weighted normalized W1≤.01, worst law≤.03. A threshold above cap has intentionally zero target mass.

Resolution diagnostics: in each law positive q10–q90 maximum adjacent-support gap≤max(500 EUR,.015 positiveMedian); for population-highest decile laws≤max(250 EUR,.01 positiveMedian). Positive maximum atom mass≤.02 (zero atom exempt). National branch histograms use1000-EUR bins within positive q10–q90; any entirely empty internal bin with target mass≥.005 fails. Distinct-value repetition is measured by Σ_l ρ_l Σ_positive p_l(v)^2 and by aggregate support multiplicity across laws, not by demanding uniqueness from finite distributions. Tail gaps are reported separately and allowed to be larger. No tiny positive atom is rounded away.

## 28. Income law pooling

Selected method: **deterministic adjacent ordered Ward clustering in log-quantile space**, separately by branch. All targets are stochastically ordered by their scalar authorized profile m after common monotone national transport. This permits an efficient adjacent clustering rule without an unrecorded multidimensional heuristic.

Distance is positive conditional squared log-quantile Wasserstein distance, `d²(A,B)=∫0^1 [ln(1+QA(u))−ln(1+QB(u))]^2 du`. Use fixed master **distance** knots `H=sorted({j/128:j=0..128} ∪ {.01,.10,.25,.50,.75,.90,.95,.99,.995,.999,.9995,.9999})`; integrate by trapezoidal quadrature. Zero-mass difference is0 within a branch; if code finds unequal target zero masses, fail rather than silently change the metric. Log1p avoids a singularity at small positive values.

Initialize one cluster per distinct exact scalar m, with all equal-m tuples assigned together; weight is branch-normalized D0 tuple mass. Sort by m, ties by canonical tuple. Repeatedly merge adjacent clusters with minimum Ward increase `massA×massB/(massA+massB) × d²(centroidA,centroidB)`. Centroids are weighted means z(h)=ln(1+Q(h)) at each H knot. Compute their distance directly as the trapezoidal sum of (z_A−z_B)^2; do not apply ln(1+x) a second time to z. Find the global minimum Ward increase first; candidates with increase≤minimum+10^-30 are tied, then choose the pair with smallest member canonical-tuple ordinal, followed by the lower-profile cluster ordinal. No seed, random assignment, manual segment mapping or cross-branch pooling is used.

Stop when distinct clusters≤branch cap[e512,s256,u64,t64,r64,i64]. The selected cap is mandatory; fewer are used only if fewer distinct profiles exist or canonical laws coincide later. Refit each shared positive Q pointwise as `exp(weighted mean ln(1+Qt(u)))−1` over all member tuples using original target functions, not its short H summary; then create its G polyline for compression. This explicitly selects the pointwise weighted log-quantile barycenter between H knots as well; the H-grid quadrature alone would not determine values between its knots. Within-branch canonical identical finite laws merge after integerization. IDs are reassigned in first canonical-tuple occurrence order; clustering IDs never become asset identity.

Pooling acceptance before support compression, uniformly for e/s/u/t/r/i: branch-weighted normalized W1≤.015 and weighted positive median error≤2% remain HARD AGGREGATE gates. Each material full tuple (within-branch D0 conditional mass≥10^-5) must have normalized W1≤.04 and positive median error≤10% (HARD MATERIAL_SEGMENT). For rare full tuples, normalized W1>.08 or positive median error>10% is WARNING RARE_SEGMENT_SANITY, with both worst-case values retained. Rarity never exempts a tuple from the HARD sanity rules in §30.

Protected visible segments retain positive q10/q50/q90 error≤5% HARD: every authorized parent-state marginal (MAJOR_CORRELATION), plus each joint parent segment with within-branch D0 conditional mass≥.005 (MATERIAL_SEGMENT). Segment targets/candidates are conditional distribution mixtures, not averages of tuple quantiles or medians. Zero mass preserved; branch mean change≤2%, q99≤5%, tail mass absolute change≤.003 remain HARD. Publish material/rare worst cases, weighted errors and every protected segment, not just an average.

Total target→pooled→finite→integer checks additionally require branch national zero mass≤.001 absolute, q25/q50/q75/q90 relative error≤3% or100 EUR, q95≤5%,q99≤8%,q999≤15%, mean≤3%, IQR/standard deviation≤8%, normalized W1≤.025. Zero quantiles are compared in absolute EUR and must remain0 when target0. Protected parent segments are not hidden by branch-level recalibration. Failed HARD metrics require evidence/fitting review and D2-B1 revision; WARNING and DIAGNOSTIC findings are accepted and reported, without retuning; D2-B2 cannot increase caps or choose another clustering rule.

## 29. Model decision cards

Each card incorporates the explicitly referenced recipe, including selectors, sampling/statistical weights, objectives, initialization, stopping, smoothing, constraints and validation. References are binding, not choices left to D2-B2. Canonical sizes below are D2-A logical charges, not Go implementation measurements. Income branch U is the deterministic allocator output; only its numerical enumeration is deferred.

Concrete unconditional national Income targets derived from §20 (MODELED ASSUMPTION; displayed EUR rounded to nearest for review, exact Q/zero formula authoritative): Q(v)=0 for v≤z, otherwise Qpositive((v−z)/(1−z)); mean=(1−z)∫Qpositive. This provides branch zeros, central quantiles, spread and tails without substituting positive medians.

| Branch | Zero mass | Unconditional q10 | q25 | q50 | q75 | q90 | q99 | Mean | Upper model cap |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| e | 0.002 | 15760 | 23920 | 31968 | 42978 | 59977 | 139970 | 38971.90 | 1000000 |
| s | 0.08 | 1913 | 12522 | 25913 | 45348 | 78087 | 256848 | 41542.60 | 2000000 |
| u | 0.35 | 0 | 0 | 4615 | 12769 | 20487 | 56231 | 8287.50 | 150000 |
| t | 0.65 | 0 | 0 | 0 | 1857 | 7429 | 30714 | 2330.83 | 100000 |
| r | 0.01 | 8495 | 13747 | 19879 | 28414 | 38929 | 79922 | 23557.05 | 300000 |
| i | 0.55 | 0 | 0 | 0 | 6611 | 16037 | 71250 | 5778.68 | 250000 |

Exact spread/tail target inputs for each branch: IQR=Q(.75)−Q(.25); variance=(1−z)∫Qpositive²−mean², with piecewise-linear-square integration; threshold mass=(1−z)(1−Fpositive(h)). Report h∈{20k,50k,100k,250k,1m} EUR plus q95/q999 from the same curve. These formulas are exact target tables, not another distribution-family choice.

### Card 1 — Company.Industry

Model: Company.Industry.  
Frozen parents: None.  
Primary Source IDs: B1.  
Secondary Source IDs: X2; X3 classification audit.  
Evidence classification: B1 CALIBRATION; X2 TRANSFORMABLE.  
Source/reference vintage: FRAME-SBS2022; ASI2025 release.  
Classification/crosswalk version: SW-IND-ATECO22-v0.  
Selected fitting method: Direct normalized source-count extension §11.  
Actual age mapping: Not applicable.  
Concrete target margins/distribution: 14 exact shares §11; enterprise-count unit.  
Modeled assumptions: Missing-sector augmentation and G/M/N splits; no workforce weighting.  
Integerization: §9 Hamilton + GCD.  
Expected canonical asset size: 264 bytes retained;128 logical input.  
Plausible validation: Industry shares and B1 broad-sector decomposition §30.  
D2-B2 exact instructions: Extract specified B1 counts; apply frozen splits/additions; emit one14-weight vector.

### Card 2 — Company.EmployeeCount

Model: Company.EmployeeCount.  
Frozen parents: Industry.  
Primary Source IDs: B1.  
Secondary Source IDs: A1 exposure audit; X2.  
Evidence classification: CALIBRATION; annual-average addetti ≠ stock.  
Source/reference vintage: ASIA2023; ASI2025.  
Classification/crosswalk version: SW-IND-ATECO22-v0.  
Selected fitting method: Finite log-uniform maximum-entropy mean tilt §12.  
Actual age mapping: Not applicable.  
Concrete target margins/distribution: Four source band/mean input rows and14 exact group assignments §12.  
Modeled assumptions: Stock carry-forward,3% dormant mass,131072 cap,explicit .02/.0005 conditional upper-tail masses,weak non-market/agriculture size.  
Integerization: §9; all128 candidate values intentionally supported.  
Expected canonical asset size: 29184 bytes retained;14512 logical input;1792 points.  
Plausible validation: Band,zero,mean,quantiles/tails §30.  
D2-B2 exact instructions: Generate128 values by fixed formula; solve each band mean; form14 laws; never sample SizeBand.

### Card 3 — Education

Model: Education.  
Frozen parents: EducationAgeBucket×Region.  
Primary Source IDs: E1.  
Secondary Source IDs: X4 metadata.  
Evidence classification: E1 TRANSFORMABLE; fine splits modeled.  
Source/reference vintage: LFS2024; ASI2025.  
Classification/crosswalk version: SW-EDU-QUAL24-v0;SW-REG-IT-v0.  
Selected fitting method: Half-shrunk normalized product/split §13.  
Actual age mapping: [21,25,35,45,55,65].  
Concrete target margins/distribution: Exact coarse counts,20 Region cells and split vectors §13.  
Modeled assumptions: Within-coarse attainment, regional/age dependence,65–89 carry-down.  
Integerization: §9; seven states positive.  
Expected canonical asset size: 13984 retained;4384 input;120 rows840 weights.  
Plausible validation: Education×age/Region §30.  
D2-B2 exact instructions: Compute each of120 normalized seven-state laws; print exact D0 margins.

### Card 4 — PrimaryActivity

Model: PrimaryActivity.  
Frozen parents: ActivityAgeBucket×Education×Region.  
Primary Source IDs: A1.  
Secondary Source IDs: E1 D0; M1 concept audit.  
Evidence classification: CALIBRATION plus modeled overlap transport.  
Source/reference vintage: LFS2024; ASI2025.  
Classification/crosswalk version: SW-ACT-LFS24-v0;SW-EDU-QUAL24-v0;SW-REG-IT-v0.  
Selected fitting method: KL/IPF raw age margins then explicit principal-role transport §14.  
Actual age mapping: [21,25,35,45,55,60,65,70].  
Concrete target margins/distribution: Exact eight raw six-state margins, associations and overlap vectors §14.  
Modeled assumptions: Student/retired overlaps,unpaid helpers,65+ status and sparse cells.  
Integerization: §9; all six states positive.  
Expected canonical asset size: 116544 retained;35904 input;1120 rows6720 weights.  
Plausible validation: Activity×age/Education/Region; raw vs principal §30.  
D2-B2 exact instructions: Fit specified raw joint; apply transition in declared precedence; emit1120 laws.

### Card 5 — Employee.JobFamily

Model: Employee.JobFamily.  
Frozen parents: Education×Region.  
Primary Source IDs: O1.  
Secondary Source IDs: O3 structural association; E1; X1 audit.  
Evidence classification: O1/O3 CALIBRATION; splits modeled.  
Source/reference vintage: EU-LFS2024 snapshot updated2026-09-10.  
Classification/crosswalk version: SW-JF-ISCO08-MAJOR-v0;SW-EDU-QUAL24-v0;SW-REG-IT-v0.  
Selected fitting method: KL/IPF to independent employee family margin §15.  
Actual age mapping: Not applicable.  
Concrete target margins/distribution: O1-derived ten-family shares, O3 half association, E1 regional vector §15.  
Modeled assumptions: Major function splits,NRP,age20–64 extrapolation; no gender residual.  
Integerization: §9.  
Expected canonical asset size: 21344 retained;6784 input;140 rows1400 weights.  
Plausible validation: Family×Education/Region and sports-function audit §30.  
D2-B2 exact instructions: Use employee O1 only for branch marginal; crosswalk/refit140 rows; no SE copying.

### Card 6 — Employee.JobLevel

Model: Employee.JobLevel.  
Frozen parents: LevelAgeBucket×Education×JobFamily.  
Primary Source IDs: None: responsibility prior explicitly modeled.  
Secondary Source IDs: A1/W2 proxy calibration; O1 audit masses.  
Evidence classification: MODELED ASSUMPTION; proxies CALIBRATION.  
Source/reference vintage: ASIA2023/INPS2024 proxies; anchor-basis authored prior.  
Classification/crosswalk version: SW-JL-RESP-v0.  
Selected fitting method: KL/IPF responsibility prior §16.  
Actual age mapping: [21,25,35,45,55,65].  
Concrete target margins/distribution: Four-tier national[.83,.09,.065,.015]; exact age/Education/family factors §16.  
Modeled assumptions: Authority not identified by grade/ISCO; conditional effects modeled.  
Integerization: §9.  
Expected canonical asset size: 33664 retained;10144 input;420 rows1680 weights.  
Plausible validation: Level×age/Education/family, proxy uncertainty §30.  
D2-B2 exact instructions: Collapse Region out of D0 employee parent masses; fit four-tier prior; emit420 rows.

### Card 7 — SelfEmployed.JobFamily

Model: SelfEmployed.JobFamily.  
Frozen parents: Education×Region.  
Primary Source IDs: O2.  
Secondary Source IDs: O3; E1; X1 audit.  
Evidence classification: O2/O3 CALIBRATION; independently fitted.  
Source/reference vintage: EU-LFS2024 snapshot updated2026-09-10.  
Classification/crosswalk version: SW-JF-ISCO08-MAJOR-v0;SW-EDU-QUAL24-v0;SW-REG-IT-v0.  
Selected fitting method: KL/IPF independent O2 family margin §17.  
Actual age mapping: Not applicable.  
Concrete target margins/distribution: Exact O2 SELF selected slice×crosswalk; O3/E1 association rules.  
Modeled assumptions: All-worker structural borrowing and major-function splits.  
Integerization: §9.  
Expected canonical asset size: 21344 retained;6784 input;140 rows1400 weights.  
Plausible validation: SE family×Education/Region, difference from employee §30.  
D2-B2 exact instructions: Select SELF only; compute own marginal and140 rows; do not load employee laws as seed.

### Card 8 — SelfEmployed.Industry

Model: SelfEmployed.Industry.  
Frozen parents: JobFamily.  
Primary Source IDs: A1 independent-person sectors.  
Secondary Source IDs: O2-derived family masses; modeled M; X2.  
Evidence classification: CALIBRATION; finer14-state split modeled.  
Source/reference vintage: LFS2024 sector counts; ASI2025.  
Classification/crosswalk version: SW-IND-ATECO22-v0 SE-person split §18.  
Selected fitting method: KL/IPF family×Industry joint §18.  
Actual age mapping: Not applicable.  
Concrete target margins/distribution: Five exact source counts and deterministic14-state split §18.  
Modeled assumptions: Unobserved detailed-sector/occupation joint; no enterprise-distribution substitution.  
Integerization: §9.  
Expected canonical asset size: 2064 retained;704 input;10 rows140 weights.  
Plausible validation: SE Industry×family/broad sector §30.  
D2-B2 exact instructions: Rake independently fitted SE family and SE14-sector margins using fixed positive M.

### Card 9 — Employer

Model: Employer.  
Frozen parents: JobFamily×JobLevel and frozen Company Industry/count/band domain.  
Primary Source IDs: A1 employee-size exposure; B1 Company laws.  
Secondary Source IDs: O1 audit family; responsibility design prior.  
Evidence classification: CALIBRATION with modeled I/size affinities and regularization.  
Source/reference vintage: ASIA2023,FRAME2022,LFS2024.  
Classification/crosswalk version: SW-IND-ATECO22-v0;SW-JF-ISCO08-MAJOR-v0;SW-JL-RESP-v0.  
Selected fitting method: Joint Company-model regularized cap-profile calibration §19.  
Actual age mapping: Not applicable.  
Concrete target margins/distribution: C4096;I64M;S printed;employee exposure target[.232032027,.255219608,.185414172,.327334192].  
Modeled assumptions: Unidentified I/JobLevel×size;cap penalty; source coverage bridge.  
Integerization: §19 half-up factors; no factor GCD/renormalization.  
Expected canonical asset size: 378 factor bytes; Employer R2/index projection §31.  
Plausible validation: All40 conditions,selectedIndustry/family,size/level,employee exposure,unusual support.  
D2-B2 exact instructions: Recompute profile trace and verify actual C/I/S; evaluate expected Company distribution; emit156 factors, cap and fit report.

### Card 10 — Income[e]

Model: finite whole-EUR Income[e].  
Frozen parents: Region×JobFamily×JobLevel×CurrentIndustry×SizeBand.  
Primary Source IDs: W1/W2 calibration; all-source target modeled.  
Secondary Source IDs: H1; M1 concept audit.  
Evidence classification: modeled all-source target; W1/W2/R1 CALIBRATION where named, H1/M1 CROSS-CHECK; no DIRECT all-source observations asserted.  
Source/reference vintage: SES2022/INPS2024; all-source nominal anchor assumption; no post-anchor realized outcomes.  
Classification/crosswalk version: SW-REG-IT-v0; SW-JF-ISCO08-MAJOR-v0;SW-IND-ATECO22-v0;SW-JL-RESP-v0.  
Selected fitting method: exact national quantile transport §20 using profile §21, adjacent ordered Ward §28, contiguous quantile compression/allocator §27.  
Actual age mapping: None; zero-band sentinel65535; no age dependency.  
Concrete target margins/distribution: zero mass 0.002; complete positive/unconditional Q/mean/spread/tail input tables §§20/29 and every exact tuple profile §21.  
Modeled assumptions: current gross all-source level/composition, tail cap 1000000 EUR, authorized relative profiles, constant branch zero mass, D0 transport calibration; branch safeguards §21.  
Integerization: whole-EUR ceiling; positive-preserving Hamilton and law GCD §9; duplicate canonical laws merge within branch only.  
Expected canonical asset size: T=56000, map=112064 bytes; pool retained64+32L+16U with L≤512; aggregateU≤240000 across six branches, per-law≤512; predicted shared resource bound §31.  
Plausible validation: zero/positive and unconditional quantiles,mean,spread/tails,authorized parent differences,pooling/finite/integer error,repetition/occupancy/gaps §§27/28/30.  
D2-B2 exact instructions: enumerate authorized tuples in frozen order; calculate D0 mass/profile/national transport; pool to≤512, allocate supports by the frozen global rule; compress/ceil/integerize; emit complete map plus branch-owned law pool and all error stages. No runtime source components or unauthorized parents.

### Card 11 — Income[s]

Model: finite whole-EUR Income[s].  
Frozen parents: Income[s]AgeBucket×Region×JobFamily×Industry.  
Primary Source IDs: None: all-source/gross attributed-rate prior modeled.  
Secondary Source IDs: H1 signed net component CROSS-CHECK; M1.  
Evidence classification: modeled all-source target; W1/W2/R1 CALIBRATION where named, H1/M1 CROSS-CHECK; no DIRECT all-source observations asserted.  
Source/reference vintage: H1 calendar2022; modeled nominal anchor prior; no post-anchor realized outcomes.  
Classification/crosswalk version: SW-REG-IT-v0; SW-JF-ISCO08-MAJOR-v0;SW-IND-ATECO22-v0.  
Selected fitting method: exact national quantile transport §20 using profile §22, adjacent ordered Ward §28, contiguous quantile compression/allocator §27.  
Actual age mapping: [21,25,35,45,55,65]; six lower bounds, last upper75.  
Concrete target margins/distribution: zero mass 0.08; complete positive/unconditional Q/mean/spread/tail input tables §§20/29 and every exact tuple profile §22.  
Modeled assumptions: current gross all-source level/composition, tail cap 2000000 EUR, authorized relative profiles, constant branch zero mass, D0 transport calibration; branch safeguards §22.  
Integerization: whole-EUR ceiling; positive-preserving Hamilton and law GCD §9; duplicate canonical laws merge within branch only.  
Expected canonical asset size: T=16800, map=33664 bytes; pool retained64+32L+16U with L≤256; aggregateU≤240000 across six branches, per-law≤512; predicted shared resource bound §31.  
Plausible validation: zero/positive and unconditional quantiles,mean,spread/tails,authorized parent differences,pooling/finite/integer error,repetition/occupancy/gaps §§27/28/30.  
D2-B2 exact instructions: enumerate authorized tuples in frozen order; calculate D0 mass/profile/national transport; pool to≤256, allocate supports by the frozen global rule; compress/ceil/integerize; emit complete map plus branch-owned law pool and all error stages. No runtime source components or unauthorized parents.

### Card 12 — Income[u]

Model: finite whole-EUR Income[u].  
Frozen parents: Income[u]AgeBucket×Education×Region.  
Primary Source IDs: None: all-source rate prior modeled.  
Secondary Source IDs: H1 personal components; M1 concept audit.  
Evidence classification: modeled all-source target; W1/W2/R1 CALIBRATION where named, H1/M1 CROSS-CHECK; no DIRECT all-source observations asserted.  
Source/reference vintage: H1 calendar2022; modeled nominal anchor prior; no post-anchor realized outcomes.  
Classification/crosswalk version: SW-REG-IT-v0; SW-EDU-QUAL24-v0.  
Selected fitting method: exact national quantile transport §20 using profile §23, adjacent ordered Ward §28, contiguous quantile compression/allocator §27.  
Actual age mapping: [21,25,35,45,55,65]; six lower bounds, last upper75.  
Concrete target margins/distribution: zero mass 0.35; complete positive/unconditional Q/mean/spread/tail input tables §§20/29 and every exact tuple profile §23.  
Modeled assumptions: current gross all-source level/composition, tail cap 150000 EUR, authorized relative profiles, constant branch zero mass, D0 transport calibration; branch safeguards §23.  
Integerization: whole-EUR ceiling; positive-preserving Hamilton and law GCD §9; duplicate canonical laws merge within branch only.  
Expected canonical asset size: T=840, map=1744 bytes; pool retained64+32L+16U with L≤64; aggregateU≤240000 across six branches, per-law≤512; predicted shared resource bound §31.  
Plausible validation: zero/positive and unconditional quantiles,mean,spread/tails,authorized parent differences,pooling/finite/integer error,repetition/occupancy/gaps §§27/28/30.  
D2-B2 exact instructions: enumerate authorized tuples in frozen order; calculate D0 mass/profile/national transport; pool to≤64, allocate supports by the frozen global rule; compress/ceil/integerize; emit complete map plus branch-owned law pool and all error stages. No runtime source components or unauthorized parents.

### Card 13 — Income[t]

Model: finite whole-EUR Income[t].  
Frozen parents: Income[t]AgeBucket×Education×Region.  
Primary Source IDs: None: personal scholarship/all-source rate prior modeled.  
Secondary Source IDs: H1 YTA2 CROSS-CHECK; M1.  
Evidence classification: modeled all-source target; W1/W2/R1 CALIBRATION where named, H1/M1 CROSS-CHECK; no DIRECT all-source observations asserted.  
Source/reference vintage: H1 calendar2022; modeled nominal anchor prior; no post-anchor realized outcomes.  
Classification/crosswalk version: SW-REG-IT-v0; SW-EDU-QUAL24-v0.  
Selected fitting method: exact national quantile transport §20 using profile §24, adjacent ordered Ward §28, contiguous quantile compression/allocator §27.  
Actual age mapping: [21,25,30,40,55,65]; six lower bounds, last upper75.  
Concrete target margins/distribution: zero mass 0.65; complete positive/unconditional Q/mean/spread/tail input tables §§20/29 and every exact tuple profile §24.  
Modeled assumptions: current gross all-source level/composition, tail cap 100000 EUR, authorized relative profiles, constant branch zero mass, D0 transport calibration; branch safeguards §24.  
Integerization: whole-EUR ceiling; positive-preserving Hamilton and law GCD §9; duplicate canonical laws merge within branch only.  
Expected canonical asset size: T=840, map=1744 bytes; pool retained64+32L+16U with L≤64; aggregateU≤240000 across six branches, per-law≤512; predicted shared resource bound §31.  
Plausible validation: zero/positive and unconditional quantiles,mean,spread/tails,authorized parent differences,pooling/finite/integer error,repetition/occupancy/gaps §§27/28/30.  
D2-B2 exact instructions: enumerate authorized tuples in frozen order; calculate D0 mass/profile/national transport; pool to≤64, allocate supports by the frozen global rule; compress/ceil/integerize; emit complete map plus branch-owned law pool and all error stages. No runtime source components or unauthorized parents.

### Card 14 — Income[r]

Model: finite whole-EUR Income[r].  
Frozen parents: Income[r]AgeBucket×Education×Region.  
Primary Source IDs: R1 pension-component calibration; all-source prior modeled.  
Secondary Source IDs: H1 YTP1; M1 concept audit.  
Evidence classification: modeled all-source target; W1/W2/R1 CALIBRATION where named, H1/M1 CROSS-CHECK; no DIRECT all-source observations asserted.  
Source/reference vintage: Pension stock2024-12-31 extractionJuly2025; modeled2026 basis; no post-anchor realized outcomes.  
Classification/crosswalk version: SW-REG-IT-v0; SW-EDU-QUAL24-v0.  
Selected fitting method: exact national quantile transport §20 using profile §25, adjacent ordered Ward §28, contiguous quantile compression/allocator §27.  
Actual age mapping: [21,45,55,60,65,70]; six lower bounds, last upper75.  
Concrete target margins/distribution: zero mass 0.01; complete positive/unconditional Q/mean/spread/tail input tables §§20/29 and every exact tuple profile §25.  
Modeled assumptions: current gross all-source level/composition, tail cap 300000 EUR, authorized relative profiles, constant branch zero mass, D0 transport calibration; branch safeguards §25.  
Integerization: whole-EUR ceiling; positive-preserving Hamilton and law GCD §9; duplicate canonical laws merge within branch only.  
Expected canonical asset size: T=840, map=1744 bytes; pool retained64+32L+16U with L≤64; aggregateU≤240000 across six branches, per-law≤512; predicted shared resource bound §31.  
Plausible validation: zero/positive and unconditional quantiles,mean,spread/tails,authorized parent differences,pooling/finite/integer error,repetition/occupancy/gaps §§27/28/30.  
D2-B2 exact instructions: enumerate authorized tuples in frozen order; calculate D0 mass/profile/national transport; pool to≤64, allocate supports by the frozen global rule; compress/ceil/integerize; emit complete map plus branch-owned law pool and all error stages. No runtime source components or unauthorized parents.

### Card 15 — Income[i]

Model: finite whole-EUR Income[i].  
Frozen parents: Income[i]AgeBucket×Education×Region.  
Primary Source IDs: None: personal all-source rate prior modeled.  
Secondary Source IDs: H1 component CROSS-CHECK; M1.  
Evidence classification: modeled all-source target; W1/W2/R1 CALIBRATION where named, H1/M1 CROSS-CHECK; no DIRECT all-source observations asserted.  
Source/reference vintage: H1 calendar2022; modeled nominal anchor prior; no post-anchor realized outcomes.  
Classification/crosswalk version: SW-REG-IT-v0; SW-EDU-QUAL24-v0.  
Selected fitting method: exact national quantile transport §20 using profile §26, adjacent ordered Ward §28, contiguous quantile compression/allocator §27.  
Actual age mapping: [21,25,35,45,55,65]; six lower bounds, last upper75.  
Concrete target margins/distribution: zero mass 0.55; complete positive/unconditional Q/mean/spread/tail input tables §§20/29 and every exact tuple profile §26.  
Modeled assumptions: current gross all-source level/composition, tail cap 250000 EUR, authorized relative profiles, constant branch zero mass, D0 transport calibration; branch safeguards §26.  
Integerization: whole-EUR ceiling; positive-preserving Hamilton and law GCD §9; duplicate canonical laws merge within branch only.  
Expected canonical asset size: T=840, map=1744 bytes; pool retained64+32L+16U with L≤64; aggregateU≤240000 across six branches, per-law≤512; predicted shared resource bound §31.  
Plausible validation: zero/positive and unconditional quantiles,mean,spread/tails,authorized parent differences,pooling/finite/integer error,repetition/occupancy/gaps §§27/28/30.  
D2-B2 exact instructions: enumerate authorized tuples in frozen order; calculate D0 mass/profile/national transport; pool to≤64, allocate supports by the frozen global rule; compress/ceil/integerize; emit complete map plus branch-owned law pool and all error stages. No runtime source components or unauthorized parents.

## 30. Plausible validation

These are synthetic plausibility tolerances, not population-representativeness confidence statements. The exact fitted target tables and documented source transforms are separate from raw official margins. Metrics are computed analytically under D0 and the Company-model distribution; the finite-universe audit uses the deterministic analytic calculations specified below. Protocol sampling evidence is outside this numerical phase and cannot be required to execute D2-B2. No new identity seed or sampler is selected by D2-B1.

Categorical distance: TV=.5Σ|p−q|. Probability errors are absolute proportions; .005 means .5 percentage point. Conditional validation reports every row, weighted mean, worst row and parent-state marginal; Income additionally separates material/rare worst cases. Missing mandatory metrics are a HARD report failure, not a pass; recorded WARNING/DIAGNOSTIC deviations do not fail production.

Income metric convention, binding on §§27/28/30: v-quantile is inf{x:F(x)≥v}; positive quantiles renormalize after removing the zero atom. Income tail mass is P(X>h), while Company threshold mass is P(n≥h). W1=∫0^1|Qcandidate(v)−Qreference(v)|dv on the **unconditional** law, including its zero atom. Normalize per tuple/law by max(1000,reference positive median). With equal branch zero rates this is(1−z) times the positive-quantile integral; use that factor in compression gain. Integrate rational polyline/step-law comparisons exactly by their breakpoint union. An original Qt or refitted barycenter used as a numerical reference is its G-grid polyline; the discrepancy from the original analytic transport is separately bounded below. Branch-weighted tuple error means Σ_t d_t×normalizedW1_t, d conditional on that branch; global allocator ρ additionally includes the branch population mass.

### Plausible quality contract

Plausible requires structurally correct, deterministic synthetic facts, believable population aggregates and the intended major socioeconomic correlations. It protects material conditional behavior without requiring fine empirical fidelity for every rare high-dimensional tuple or fine tail. These six requirements are normative:

| Requirement | Mechanical acceptance role |
| --- | --- |
| Structural plausibility | HARD: no contradictory activity/payload, invalid employer/self-employed relation, forbidden occupational payload, Contract-5 violation or invalid probability/support law; S01/S02/N02/N07/N08/EMP06/SAN02 |
| Aggregate plausibility | HARD: Company, categorical, Employer and all six Income branch/population-weighted gates retain their existing thresholds; C01/C04/E01/PA01/JF01/SE01/EMP04/P01/P05/A01–A04 |
| Major-correlation plausibility | HARD: preserve age↔Education; age/Education/Region↔Activity; Education/Region↔JobFamily; age/Education/family↔JobLevel; occupation↔employer Industry; activity/occupation/Region↔Income through exact fitted targets and protected parent marginals; E01/PA01/JF01/JL01/SE01/EMP01/EMP02/EMP05/P04a/M01a/SAN01 |
| Material-segment plausibility | HARD: material full tuples and material joint visible segments must independently pass; good weighted fit cannot excuse a failure; P02/P04b/M00/M01b |
| Rare-microsegment plausibility | HARD sanity plus WARNING fidelity: validity, bounds, ordering, barycenter integrity, compression integrity and visible-artifact checks apply regardless of mass; P03/M02/SAN01–SAN03/F01/F02/F04/F06/F07/REP01/REP02 |
| Fine-tail fidelity | SOFT: per-law compression q999 error>10% is WARNING, not production failure; F03. Branch tail thresholds, valid ordered tails, caps and atom checks remain HARD; P05/A02/SAN01/SAN02/REP01 |

Each acceptance metric has exactly one dimension: STRUCTURAL, NUMERICAL_INTEGRITY, AGGREGATE, MAJOR_CORRELATION, MATERIAL_SEGMENT, RARE_SEGMENT_SANITY, TAIL_SANITY, REPETITION_SANITY, RESOURCE or DETERMINISM, as assigned in the matrix below. There is no blended score. HARD failure prevents D2-B2 success; WARNING is accepted but prominently surfaced in `plausible.md`; DIAGNOSTIC is recorded for analysis only. All mandatory metrics, including warnings/diagnostics, must be reported. A missing or unevaluated HARD gate cannot be called PASS. Protocol-owned runtime verification remains with its frozen owner; this phase requires numerical/asset determinism without selecting a runtime sampler.

### Mechanical materiality and visible segments

For each branch b, let `d_b(t)=D0(b,authorizedTuple=t)/D0(b)`, using the fitted **pre-integer** distributions and marginalization rules of §8; these masses sum to1 over reachable tuples. Material full tuple iff `d_b(t)≥10^-5`; rare full tuple iff `0<d_b(t)<10^-5`. Equality is material. Compare the modeled rational masses, not generated counts, rounded display values, positive-Income mass `(1−z_b)d_b(t)`, joint population mass `D0(b)d_b(t)`, exact-profile/cluster mass or post-pooling candidate probabilities. The existing important-tuple `10^-5` rule and this probability space predate RETRY2; neither is changed. Apply identically to all six branches.

For a parent-subset assignment S, `d_b(S)=Σ_{t in S}d_b(t)`. Material visible segment iff `d_b(S)≥.005`, retaining the existing joint-segment threshold. Independently, every single authorized parent-state marginal remains protected as a major correlation even when its mass is below .005. Protected segments are the union of those singletons and material joint segments, exactly the former important-segment set. Enumerate every nonempty subset/assignment canonically before evaluating candidate errors; employee zero-band cells are excluded. A small full tuple can belong to several protected segments; each is tested as its own mixture of laws conditional on S, with weights `d_b(t)/d_b(S)`. Never average member quantiles in place of the mixture quantile. Branch-wide mixtures are tested separately. Lower-dimensional age, Education, Region, JobFamily, JobLevel, Industry and SizeBand segments use only the value parents authorized for that branch; no employee age/Education parent is added.

End-to-end material full-tuple positive q10/q25/q50/q75/q90 retain the former 10% conditional limit HARD (M00). M00 applies to every material full tuple, regardless of whether that tuple is also a member of one or more protected coarser segments. Membership in a protected segment does not exempt a material full tuple from M00. Where M01 applies to one or more protected projections containing that tuple, it is an additional independent HARD check (M01a/M01b). Passing M01 does not imply, replace, subsume or waive M00. Other non-protected proper-subset joints and rare full tuples retain their positive-centre errors as WARNING at the former 10% limit (M02); full-tuple pooling median/W1 retain their distinct WARNING limits (P03). This does not convert every Cartesian combination into a HARD conditional gate. Repetition uses unconditional positive atom probabilities, zero omitted, and overall law massesρ.

### HARD sanity independent of materiality

Every reachable tuple resolves to one valid law with finite, nonnegative, ordered quantiles; whole-EUR positive supports lie in `1..branchCap`, zero has its prescribed semantics, and positive weights/totals/capacities are valid. Ordered-profile checks below apply to rare as well as material tuples and prevent gross socioeconomic reversal. Every uncompressed barycenter on G must lie between the minimum and maximum original member Qt values at that knot, with absolute numerical allowance10^-40; this is a mathematical property of the unchanged log-barycenter, not an empirical error tolerance. The existing per-law compression centre/mean/spread/W1 checks, central-gap/maximum-atom checks and branch histogram checks apply regardless of law/tuple mass. Branch tail and protected-segment gates remain HARD. These small, general rules bound rare output scale and artifacts without imposing an additional rare-tuple percentage-error ceiling. A 50% or larger rare-tuple target error is deterministically WARNING when these HARD gates pass; it is HARD failure whenever any sanity, aggregate or protected-segment gate fails.

Ordered-profile audit compares every pair of pooled clusters in scalar-m order at positive q10/q25/q50/q75/q90/q95/q99. Equalities are allowed. Report every finite-law reversal; it is material and fails if its opposite-order value difference exceeds max(100EUR,.05×the larger of the two corresponding uncompressed target quantiles). Smaller discretization reversals are recorded, not treated as exact semantic invalidity. Parent-marginal ranks are compared with their own target marginal mixture, not inferred from a single profile coefficient.

Numerical-reference audit: evaluate the analytic transported/refitted Q additionally at the fixed check grid sorted({j/16384:j=0..16384}∪G), form its check polyline and compare with the G reference by exact integration. Reference normalizedW1≤10^-4 and reference mean/positive q10/q50/q90/q99 relative difference≤.001 are mandatory. This controls the specified fitting discretization; it does not claim an exact continuous transport has been compressed without approximation. Weighted branch/segment final tolerances are checked against the higher-resolution check reference as well. Failure does not select a new G grid.

Finite-universe audit, deterministic and mandatory: for each Company state/event A report p_A, expected countNp_A, varianceNp_A(1−p_A), and absence probability(1−p_A)^N. Events are all14 Industries, five SizeBands, their70 intersections and n≥1000/10000/65536; all-positive absence risk is z^N. Compute powers by exact integer exponentiation of the fitted rational probability. For each Employer condition and selected state A define one-Company Y=W and X_A=W1_A under the specified Company distribution; q_A=E[X_A]/E[Y]. Report delta-method variance E[(X_A−q_AY)^2]/(N E[Y]^2), its square-root standard error and5-SE interval clamped to[0,1], clearly labeled an approximation. Report all Industry/band states and joint cells, with no new RNG, seed, sampler or Company realization. Tail-absence risk above .05 is a review warning, not a forced retry or a constructor change. Sample-based Protocol checks remain a later owner's responsibility.

### Normative acceptance matrix and change ledger

This matrix is exhaustive for the gate families in §§9/11–19/27/28/30/31/32/33/36. Existing production acceptance limits were HARD unless explicitly warnings or diagnostic reporting; the old-severity column records that prior policy. Scope includes all original conditional rows unless explicitly narrowed. Every metric within a row inherits its single dimension/severity; SAN01d separately classifies below-limit reversal records. Referenced recipes define exact computations, not discretion. A tolerance A% or B units means error≤max(A%×reference,B); zero references remain0 where specified. No threshold is calibrated to RETRY2.

| ID | Metric / family | Scope | Old Plausible+ severity | New dimension | New severity | Threshold / mechanical rule | Materiality interaction | Reason | Change |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| S01 | Frozen facts/DAG/relations | All models/reachable branches | HARD | STRUCTURAL | HARD | Contracts1–5/D2-A: exact payload gates, Company domain/CompanyIndex, valid employers/SE, Income scope, Gender/geography exclusions, random-access/non-interference boundaries | No exemption | Semantic correctness | UNCHANGED |
| S02 | Source concepts/feature allowlists | All extraction/config/assets | HARD | STRUCTURAL | HARD | §§2/5/6/7/20 exact units/vintages/crosswalks; no hidden parent, salary-only/household/transfer/loss substitution | No exemption | Preserve meanings and source basis | UNCHANGED |
| N01 | Snapshots and classification/input audits | All pinned sources; X2/X3; Trentino | HARD | NUMERICAL_INTEGRITY | HARD | §§5/6/7/8/33 exact SHA/size/schema/cells; edge-U+0020 only; M=9 separate; partner/hierarchy/collision checks; exact Region alias/precedence | No exemption | Do not weaken parsing/audit rectifications | UNCHANGED |
| N02 | Probability laws/fitted support validity | Every categorical/finite row | HARD | NUMERICAL_INTEGRITY | HARD | §§8/9: finite nonnegative probabilities, total1, valid intentional support; positive total and checked bounds before narrowing | No exemption | Invalid laws cannot be accepted | UNCHANGED |
| N03 | Convergence/feasibility/fit sequencing | Every production fit | HARD | NUMERICAL_INTEGRITY | HARD | §§8/12/19: exact constraints; IPF absolute<10^-24 and relative<10^-20 for margins≥10^-12 within50000 sweeps; specified256-step hull-safe tilt; fixed cap-profile convergence; prescribed floor/precision/pre-integer sequencing | No exemption | No fallback/nonconvergence | UNCHANGED |
| N04 | G numerical-reference accuracy | Every transported/refitted reference; branch/segments | HARD | NUMERICAL_INTEGRITY | HARD | §30 check-grid nW1≤10^-4; mean and positive q10/q50/q90/q99 relative difference≤.001; weighted branch/segment final checks also against check reference | No exemption | Numerical approximation integrity | UNCHANGED |
| N05 | Integer probability errors | Every canonical ordinary law | HARD | NUMERICAL_INTEGRITY | HARD | §9 TV≤3×10^-7; maximum entry≤1.3×10^-7; categorical maximum≤10^-8 | No exemption | Engineering accuracy | UNCHANGED |
| N06 | Integer mean errors | EmployeeCount/Income laws | HARD | NUMERICAL_INTEGRITY | HARD | §9 EmployeeCount mean≤.01 headcount; Income mean≤1EUR | No exemption | No integerization drift | UNCHANGED |
| N07 | Positive-support preservation/integer bounds | All laws/factors/totals | HARD | NUMERICAL_INTEGRITY | HARD | §9 intentional positive set unchanged; intentional zeros0; ceiling only after range checks; uint32 weights/support, checked uint64 sums; no wrap/clipping | No exemption | Rare mass cannot vanish | UNCHANGED |
| N08 | Canonicalization/IDs | All assets/maps/laws | HARD | NUMERICAL_INTEGRITY | HARD | §§9/28/32 exact ordering, duplicate merge, GCD ordinary laws, within-branch dedup/first-occurrence IDs; Employer factors/intervals never rescaled | No exemption | Canonical behavior | UNCHANGED |
| D01 | Numerical reproducibility | Source-order and independent asset reruns | HARD | DETERMINISM | HARD | §§8/28/32/33 pinned arithmetic/algorithms/ties; canonical logical bytes/checksums identical under prescribed reverse/permuted source order and independent rerun | No exemption | Reproducibility remains exact | UNCHANGED |
| R01 | Frozen resource limits | All input/retained/constructor phases | HARD | RESOURCE | HARD | §31/D2-A exact E/D/Rout≤8MiB each; combined≤24MiB; index/input≤64MiB; R2≤72MiB; joint≤80MiB; peak≤128MiB; Cwork≤32m; prospective checks | No exemption | No overflow/excess allocation | UNCHANGED |
| R02 | Selected capacities/reservations | Every authored/canonical law and phase | HARD | RESOURCE | HARD | §§3/27/31: CompanyCount65536; count≤128 points; Income e512/s256/u64/t64/r64/i64 laws,≤512 authored/canonical points per law,≤240000 authored points;128 distinct points including zero;384 authored slots for top population decile; E≤6.5MiB,D≤6MiB and selected phase reservations | No exemption | Caps/allocation stay fixed | UNCHANGED |
| C01 | Company Industry target | 14-state integer law | HARD | AGGREGATE | HARD | §11 target absolute error≤.0001 per state | All Industry states | Population composition | UNCHANGED |
| C02 | Company source decomposition | Four source-sector comparisons | WARNING | AGGREGATE | WARNING | §30 unexplained source difference>5pp warns; coverage/vintage/augmentation decomposition mandatory | No fit-based selection | Do not claim direct representativeness | UNCHANGED |
| C03 | Company realization count deviation | Separate later sample audit; not executed in D2-B2 | HARD (later owner) | AGGREGATE | HARD | §11 deviation≤max(1pp,5binomialSE); no forced sample/retry or new Protocol mechanism | Existing sample scope | Keep later audit; no new phase dependency | UNCHANGED |
| C04 | Company band/zero targets | Overall five bands/zero | HARD | AGGREGATE | HARD | §§12/30 band/zero error≤.0001; zero .03±.0001 | All bands | Visible size composition | UNCHANGED |
| C05 | Company Industry×band targets | Every Industry and band | HARD | MAJOR_CORRELATION | HARD | §§12/30 target absolute error≤.0001 per cell; band derived from count | All cells as before | Industry/size relationship | UNCHANGED |
| C06 | Headcount centre/mean | Overall/byIndustry q50/q90/q99;band q90;mean | HARD | AGGREGATE | HARD | §§12/30 quantile error≤max(1headcount,2%); mean≤1%; integer mean N06 | All original scopes | Count distributions unchanged | UNCHANGED |
| C07 | Company tails | P(n≥1000/10000/65536);support cap | HARD | TAIL_SANITY | HARD | §§12/30 mass absolute error≤.0001 versus fixed fit; support≤131072 | All original scopes | Existing assumed-tail protection | UNCHANGED |
| C08 | Finite-universe variance/risk | Industry/band/intersection/tail;40 Employer conditions | DIAGNOSTIC | AGGREGATE | DIAGNOSTIC | §30 exact p/Np/variance/absence and stated delta-method5-SE intervals; no sampler; all values recorded | All prescribed events | Construction-risk evidence | UNCHANGED |
| C09 | Tail absence risk | Finite-universe analytic events | WARNING | TAIL_SANITY | WARNING | §30 tail absence probability>.05 warns; never forces realization/retry | All tail events | Rare giants are not guaranteed | UNCHANGED |
| E01 | Education targets/correlation | Rows and D0 Education×age/Region | HARD | MAJOR_CORRELATION | HARD | §§13/30 target≤.005/cell and weightedTV≤.005; specified cohort/qualification structure retained | All existing scopes | Broad attainment relationships | UNCHANGED |
| E02 | Education source comparison | Raw coarse attainment | WARNING | AGGREGATE | WARNING | §30 unexplained raw coarse-source difference>5pp warns | No changed source target | Keep evidence caveats | UNCHANGED |
| PA01 | Activity targets/correlation | Rows;Activity×age/Education/Region | HARD | MAJOR_CORRELATION | HARD | §§14/30 transported exact target≤.005/cell and weightedTV≤.005 | All existing scopes | Broad lifecycle/qualification/geography | UNCHANGED |
| PA02 | Principal-role reconciliation | Every overlap flow | HARD | STRUCTURAL | HARD | §14 exact transport and precedence; all raw/final margins and overlap flows recorded; no raw=final assertion | No exemption | Branch meaning cannot change | UNCHANGED |
| JF01 | Employee/SE JobFamily | Branch rows/national/family×Education/Region | HARD | MAJOR_CORRELATION | HARD | §§15/17/30 target≤.005/cell and weightedTV≤.005; independent branch margins; prescribed qualification effects and sports functions | All existing scopes | Occupation relationships remain | UNCHANGED |
| JL01 | JobLevel targets/correlation | Rows;Level×age/Education/family;national | HARD | MAJOR_CORRELATION | HARD | §§16/30 cell≤.005;national≤.001; target authority odds/lifecycle trends; no grade/ISCO equivalence | All existing scopes | Responsibility relationships remain | UNCHANGED |
| SE01 | SE Industry targets/correlation | 14-state/five-broad margins;family×Industry | HARD | MAJOR_CORRELATION | HARD | §§18/30 target≤.005/cell and weightedTV≤.005; functional compatibility and positive unusual support | All existing scopes | Person sectors stay independent of Company | UNCHANGED |
| EMP01 | Employer Industry targets | 40 conditions;Industry×family | HARD | MAJOR_CORRELATION | HARD | §§19/30 analytic listed-factor target≤.001/cell | All conditions | Occupation/employer association | UNCHANGED |
| EMP02 | Employer size targets | 40 conditions;band×level | HARD | MAJOR_CORRELATION | HARD | §§19/30 analytic listed-factor target≤.001/cell | All conditions | Responsibility/size association | UNCHANGED |
| EMP03 | Employer factor reproduction | C/I/S;profile trace and selected cap | HARD | NUMERICAL_INTEGRITY | HARD | §19 C4096,I64M and16 listed S exact; factor1..4095; no normalization/clipping; selected global-min/tie rule exact | No exemption | No retuning | UNCHANGED |
| EMP04 | Employee exposure quantization | Overall calibrated positive bands | HARD | AGGREGATE | HARD | §19 TV≤.005 and per-band absolute error≤.005 after factor quantization | Whole design | Keep passing aggregate threshold | UNCHANGED |
| EMP05 | Employer DAG/design sensitivity | Actual fitted family×level versus independence design | HARD | MAJOR_CORRELATION | HARD | §30 overall band absolute difference≤.05/cell | Whole actual DAG | DAG cannot hide bad exposure | UNCHANGED |
| EMP06 | Employer support/arithmetic | All40 conditions and Company domain entries | HARD | STRUCTURAL | HARD | §19 every positive Company positive raw weight; zero Company0; valid domain;checked uint64 totals;no hidden exclusion | No exemption | Relational validity | UNCHANGED |
| EMP07 | Factor/cap boundary pressure | Profile diagnostics | DIAGNOSTIC | NUMERICAL_INTEGRITY | DIAGNOSTIC | §§19/36 report unpenalized cap,objective grid,factor≥90% bound pressure;bound violation still EMP03 HARD | All factors/caps | Non-identification remains visible | UNCHANGED |
| I01 | Income zero preservation | Every tuple/pool/finite law | HARD | NUMERICAL_INTEGRITY | HARD | §§20/27/28 zero unchanged through pooling/compression; before integerization error exactly0 | No exemption | Fixed branch zero recipe | UNCHANGED |
| I02 | Income branch zero | All six branch mixtures | HARD | AGGREGATE | HARD | §30 e.002/s.08/u.35/t.65/r.01/i.55 absolute error≤.001; integer≤3×10^-7 | All branches | Zero-heavy branches stay plausible | UNCHANGED |
| I03 | Income parent zero | Each authorized parent-state marginal | HARD | MAJOR_CORRELATION | HARD | §30 absolute error≤.002; integer≤3×10^-7 | All singletons | No visible zero-rate distortion | UNCHANGED |
| P01 | Pooling weighted errors | Each branch under d_b | HARD | AGGREGATE | HARD | §28 weighted nW1≤.015;weighted positive median relative error≤.02 | All tuple masses, including rare | Many rare errors cannot hide collectively | UNCHANGED |
| P02 | Pooling material tuple errors | Each material full tuple | HARD | MATERIAL_SEGMENT | HARD | §28 nW1≤.04;positive median relative error≤.10 (former universal .08 bound is weaker than .04) | d_b(t)≥10^-5 | Material worst-case retained | UNCHANGED |
| P03 | Pooling rare tuple worst cases | Every rare full tuple | HARD | RARE_SEGMENT_SANITY | WARNING | §28 nW1>.08 or positive median relative error>.10 warns;record worst errors and offending tuples; no fidelity-only HARD ceiling | 0<d_b(t)<10^-5 | Universal rare-cell fidelity exceeded Plausible need;sanity remains HARD | RECLASSIFIED |
| P04a | Pooling parent-state centres | Every authorized singleton | HARD | MAJOR_CORRELATION | HARD | §28 mixture positive q10/q50/q90 relative error≤.05 | Protected even below .005 | Protect major visible relationships | UNCHANGED |
| P04b | Pooling material segment centres | Every qualifying joint parent assignment | HARD | MATERIAL_SEGMENT | HARD | §28 mixture positive q10/q50/q90 relative error≤.05 | d_b(S)≥.005 | A good branch average is insufficient | UNCHANGED |
| P05 | Pooling branch moments/tails | Each branch mixture | HARD | AGGREGATE | HARD | §28 mean≤2%;q99≤5%;P(X>h) absolute change≤.003 at §30 thresholds | Whole branch | Passing aggregate thresholds unchanged | UNCHANGED |
| F01 | Finite zero error | Every pooled→finite law | HARD | NUMERICAL_INTEGRITY | HARD | §27 exact zero error0 | No exemption | Preserve fixed zero mass | UNCHANGED |
| F02 | Finite central accuracy | Every pooled→finite law | HARD | NUMERICAL_INTEGRITY | HARD | §27 positive q10/q25/q50/q75/q90 relative error≤2% | No exemption | Faithful discretization of selected shared law | UNCHANGED |
| F03 | Finite fine-tail fidelity | Every pooled→finite law positive q999 | HARD | TAIL_SANITY | WARNING | §27 relative error>10% warns; retain every value and worst case | All laws equally;not tuple-specific | Fine local tail resolution is soft;caps/atoms/branch tails remain HARD | RECLASSIFIED |
| F04 | Finite moments/spread | Every pooled→finite law | HARD | NUMERICAL_INTEGRITY | HARD | §27 mean error≤max(1EUR,1%);positive IQR/SD≤5% | No exemption | Prevent coarse-scale/tail distortion | UNCHANGED |
| F05 | Finite weighted distance | Branch/population-weighted pooled→finite errors | HARD | AGGREGATE | HARD | §27 weighted normalizedW1≤.01 under prescribed D0 weights | All laws including rare | Population accuracy retained | UNCHANGED |
| F06 | Finite worst distance | Every pooled→finite law | HARD | NUMERICAL_INTEGRITY | HARD | §27 nW1≤.03 | No exemption | A shared law cannot be grossly misrepresented | UNCHANGED |
| F07 | Finite visible tails | Every pooled→finite law | HARD | TAIL_SANITY | HARD | §27 positive q95≤3%,q99≤5%;tail absolute error≤.002 at100k/250k/1m within branch cap;above cap target0 | No exemption | Visible tail sanity retained | UNCHANGED |
| A01 | Final branch centre | Unconditional/positive q10/q25/q50/q75/q90 | HARD | AGGREGATE | HARD | §30 error≤max(100EUR,3%);target-zero quantiles remain exactly0 | All six branches | Believable branch aggregates | UNCHANGED |
| A02 | Final branch upper/tail | q95/q99/q999;P>20k/50k/100k/250k/1m | HARD | AGGREGATE | HARD | §§28/30 quantile relative error≤5%/8%/15%;tail absolute≤.005 | All six branches | No relaxation of branch tails | UNCHANGED |
| A03 | Final branch moments/spread | Mean;positive/unconditional IQR/SD | HARD | AGGREGATE | HARD | §§28/30 mean≤3%;IQR/SD≤8%;zero target quantities compared absolutely and retained0 | All six branches | Visible scale/spread retained | UNCHANGED |
| A04 | Final branch distance | Target→pooled→finite→integer branch mixture | HARD | AGGREGATE | HARD | §28 normalizedW1≤.025 | Whole branch;stage gates also apply | End-to-end aggregate accuracy | UNCHANGED |
| M00 | Final material full-tuple centres | Every material full tuple, regardless of membership in protected segments | HARD | MATERIAL_SEGMENT | HARD | §30 positive q10/q25/q50/q75/q90 relative error≤10% against original target; applicable M01a/M01b protected-projection HARD checks are cumulative and independent; passing M01 does not imply, replace, subsume or waive M00 | d_b(t)≥10^-5; no exemption from P02 | Retain former conditional centre protection for material full tuples | UNCHANGED |
| M01a | Final parent-state centres | Every authorized singleton | HARD | MAJOR_CORRELATION | HARD | §30 mixture positive q10/q25/q50/q75/q90 relative error≤5%;unconditional centre error≤max(100EUR,5%);target-zero quantiles0 | Protected even below .005 | Protect Region/age/Education/occupation marginals | UNCHANGED |
| M01b | Final material segment centres | Every qualifying joint parent assignment | HARD | MATERIAL_SEGMENT | HARD | §30 same centre rule as M01a against original target | d_b(S)≥.005 | Protect meaningful combinations without every Cartesian HARD gate | UNCHANGED |
| M02 | Other rare joint-segment centres | Every non-protected proper-subset joint and rare full tuple | HARD | RARE_SEGMENT_SANITY | WARNING | §30 positive q10/q25/q50/q75/q90 relative error>10% warns; all values retained | d_b(S)<.005;singletons remain M01a;material full tuples M00;rare full tuples P03 | Fine conditional fidelity no longer universally HARD | RECLASSIFIED |
| SAN01 | Ordered-profile reversals | Every scalar-ordered pooled-cluster pair;finite laws | HARD | MAJOR_CORRELATION | HARD | §30 positive q10/q25/q50/q75/q90/q95/q99 opposite difference≤max(100EUR,.05×larger corresponding uncompressed target quantile);smaller reversals DIAGNOSTIC | No exemption for rare | Gross socioeconomic inversion still fails | UNCHANGED |
| SAN01d | Smaller ordered-profile reversal records | Every finite-law reversal below SAN01 failure limit | DIAGNOSTIC | MAJOR_CORRELATION | DIAGNOSTIC | §30 record every opposite-order difference that does not fail SAN01; equality allowed | No exemption for rare | Preserve discretization diagnostics | UNCHANGED |
| SAN02 | Income law/quantile/support sanity | Every reachable law,including rare | HARD | RARE_SEGMENT_SANITY | HARD | §§9/20/27/28: finite nondecreasing quantiles;checked nonnegative valid law;positive support1..branchCap;zero semantics/capacities valid;REP01/REP02 also mandatory | No exemption | No invalid or absurd branch-scale output | UNCHANGED |
| SAN03 | Barycenter member envelope | Every shared uncompressed law at every G knot | — | RARE_SEGMENT_SANITY | HARD | min member Qt−10^-40≤sharedQ≤max member Qt+10^-40 | No exemption;original functions/member masses | General mathematical consequence of preserved barycenter;no fitted threshold | ADDED |
| REP01 | Central gaps/maximum atom | Every finite law;top population decile stricter | HARD | REPETITION_SANITY | HARD | §27 q10–q90 adjacent gap≤max(500EUR,.015positiveMedian);top decile≤max(250EUR,.01positiveMedian);max unconditional positive atom≤.02;zero exempt | All laws;ρ defines top decile | Visible catalogue/pathology safety net | UNCHANGED |
| REP02 | Branch histogram occupancy | Positive q10–q90;1000EUR bins | HARD | REPETITION_SANITY | HARD | §27 no entirely empty internal bin with target mass≥.005 | Branch-visible target bins | Prevent systematic gaps/artifacts | UNCHANGED |
| REP03 | Collisions/support reuse/tail gaps | Every law and aggregate support multiplicity | DIAGNOSTIC | REPETITION_SANITY | DIAGNOSTIC | §27 record ΣρΣpositive p(v)^2,support multiplicity,distinct-value counts,tail gaps;no invented uniqueness limit | All laws/rare collisions recorded | Keep useful diagnostics without false uniqueness promise | UNCHANGED |
| SEN01 | Fixed sensitivity suite | Every §30 prescribed one-at-a-time perturbation | DIAGNOSTIC | MAJOR_CORRELATION | DIAGNOSTIC | §30 exact scenarios/metrics;report fit/resource scenario failures;never retune production;scenario failure is not a production pass | Production severity not changed by diagnostic scenario | No overfitting/search | UNCHANGED |
| SEN02 | Boundary sensitivity warnings | Eight age maps | WARNING | MAJOR_CORRELATION | WARNING | §§13/14/16/20 Education/Level weightedTV>.02;Activity>.03;Income branch median change>3% warns | Prescribed ±1 scenarios | Keep original review thresholds | UNCHANGED |
| REP04 | Required evidence completeness | All mandatory metrics/manifests/reports | HARD | NUMERICAL_INTEGRITY | HARD | §§30/32 report every HARD/WARNING/DIAGNOSTIC value,scope,reference,mass,class,worst/weighted/segment result;missing mandatory evidence fails report | No missing rare diagnostics | A warning is not permission to omit evidence | UNCHANGED |

Change ledger totals are obtained from the final column: UNCHANGED preserves severity/threshold/model; RECLASSIFIED changes only the role of the existing rare-tuple, rare-joint-centre and per-law q999 diagnostics; ADDED SAN03 states a threshold-free mathematical invariant with the existing10^-40 numerical precision. RELAXED: none. REMOVED: none. Aggregate thresholds unchanged. All source/crosswalk/model coefficients, exact-profile grouping, scalar order, H, adjacent Ward/D0 weighting/original-function refit/ties, branch caps, compression/allocator/integerization/canonicalization and resource formulas are unchanged. There is no general exemption from validity, ordering, atom/gap, aggregate or protected-segment gates.

Student/inactive zero-heavy unconditional spread is reported alongside positive spread. Reliability, missing-weight share, vintage/coverage bridges and uncertainty remain report evidence, not fit residuals. Non-Income models retain their existing coefficients, acceptance thresholds and severity; lifecycle/odds trends are checked against their prescribed fitted targets, with only their existing integerization tolerances, not a newly fitted rank criterion.

Fixed sensitivity execution contract — every perturbation is **MODELED ASSUMPTION**, is one-at-a-time, and produces diagnostics only. Rebuild D0 and refit descendants in DAG order; keep the pinned sources, production fitting methods, numerical precision, law caps, resource limits and tolerances. Report reference and perturbed parent/branch masses and all acceptance metrics. A warning prompts independent review; a fit/resource failure remains a failed scenario, not permission to alter production.

- Age: move exactly one internal lower bound by−1 or+1, keeping its bucket's source inputs and parameter vectors fixed; then reassign ages and refit the affected node/descendants. Production bounds remain unchanged.
- CompanyCount: evaluate32768 and98304 with the same laws/factors; expected normalized Employer margins remain the same, but variance, absence risk and resource charges change.
- Company augmentation: multiply exactly one of the five missing-sector additions by .75 or1.25 and renormalize the14-state law. Headcount zero: substitute .015 or.045 everywhere, applying the same positive-micro mean correction. Stock mean: multiply all source band-mean targets by .9 or1.1. Cap diagnostics65536/262144 use the same support formula and fixed tail strata intersected with that cap; infeasible hull constraints are reported, not repaired.
- Headcount tail: multiply both large-band upper-stratum masses .0195/.0005 by .5 or1.5; assign the remainder to250–9999 and solve the same mean constraint.
- Education fine split: separately multiply one below-primary, primary, post-secondary or doctoral fraction in one age bucket by .75 or1.25; the low/middle/high residual goes to lower-secondary/upper-secondary/tertiary respectively. Other fractions remain fixed. Apply the original floor locations. Region shrink: for one model at a time use .25 or.75 instead of .5 in Education, Activity Region association, or occupational Region exponent k_f×shrink.
- Activity Education association: change the adult .5 exponent to .25 or.75, keeping the additional65+ half-shrink and the direct21–24 modeled vectors. Overlap: multiply all four student/retired transport vectors, excluding helper allocation, by .5 or1.5 before the original student cap/precedence. Helper diagnostic: substitute .01/.03 for .02.
- JobLevel: multiply exactly one non-IC national tier mass by .75 or1.25, assigning the residual to IC; all responsibility seed factors remain fixed. Occupation: multiply one positive coefficient of one ambiguous major vector by .8 or1.2, then normalize that row; zeros stay zero. Direct singleton rows need no split perturbation. SE borrowing: use .25/.75 instead of .5 for the SE O3 Education association only. SE Industry: multiply one positive entry of one within-broad-sector split vector by .8/1.2 and normalize that vector.
- Employer: substitute penalty .005/.02 for .01 and rerun the entire fixed cap grid/profile process. In separate scenarios, for one JobFamily at a time multiply its maximum I cells by .75/1.25, half-up to integers, keep all other I cells fixed and rerun the profile. Do not clip a factor violating4095; report failure. A changed diagnostic C/S is not a changed production proposal.
- Income level: in one branch use Q'(u)=1+k(Q(u)−1), k=.9/1.1 for e/r, .85/1.15 for s/u, .8/1.2 for t/i. Income zero: use the exact branch pairs e .001/.004,s .04/.12,u .25/.45,t .55/.75,r .005/.02,i .45/.65. Positive profiles remain fixed; recompute national transport, pooling and global allocation.
- Income tail: for one branch retain knots through .95; at .99,.999,1 set Q'(u)=Q(.95)+k(Q(u)−Q(.95)), k=.5/1.5. This stretches excess above q95 and preserves strictly increasing knots; mechanically multiplying q99 by .5 could put it below q95. The perturbed endpoint is the diagnostic cap, checked against uint32. Level, zero and tail experiments are separate.

Production constants do not change in response to sensitivity outputs. The suite is fully determined; D2-B2 does not select perturbation cells, rebalancing rules or tail repairs.

## 31. Resource projection

Projection follows frozen D2-A formulas; MiB=1,048,576 bytes. All headroom is deliberate. No persistent source microdata, fitting script, dataframe or notebook becomes runtime state.

Selected numerical envelope: N=65,536;1951 categorical rows,12,194 categorical weights;14 EmployeeCount laws,1792 points;8 maps of55 ages; Income law ceilings512+256+4×64=1024; authored Income supports≤240,000; tuple slots56,000+16,800+4×840=76,160. Zero employee-band11,200 slots are retained sentinel entries. Maximum positive Company count q=N is reserved.

| Object | D2-A formula | Projected charge |
| --- | --- | ---: |
| Categorical logical input | Σ(64+8r+4Q), seven model families |64,832 bytes |
| Categorical outside retained | Σ(64+32r+12Q) |209,208 bytes |
| EmployeeCount logical input |64+8×14+8×1792 |14,512 bytes |
| EmployeeCount outside retained |64+32×14+16×1792 |29,184 bytes |
| Eight age mappings |8×72 input;8×127 retained |576 input;1016 retained |
| Income tuple maps |Σbranches(64+2T) |152,704 bytes |
| Income law pool logical input |6×64+8×1024+8×240000 |1,928,576 bytes |
| Income law pools outside retained |6×64+32×1024+16×240000 |3,873,152 bytes |
| Universe descriptor |72 outsideR2 |72 bytes |
| Generated Industry array |64+N outsideR2 |65,600 bytes |
| Generated EmployeeCount metadata |64+4N inR2 |262,208 bytes |
| Raw Company count staging reservation |8N |524,288 bytes |
| Employer factors |F=64+2×157 |378 bytes |
| Employer indexes/raw semantic input |64+32K+16P;K=40,P=40N |41,944,384 bytes ≈40.0013MiB |
| H2 retained reservation |≤1MiB |1,048,576 bytes |
| R2 total |count metadata+indexes+F+H2 |43,255,546 bytes ≈41.2527MiB |
| Outside retained calculated payload |categorical+count+age+maps+Income+descriptor+Industry |4,330,936 bytes ≈4.1303MiB |
| Outside retained additional headroom reservation |1MiB descriptor/bookkeeping allowance |Rout reservation≈5.1303MiB |
| Encoded numerical input E_live |Canonical audit JSON payload plus bounded encoding allowance |Reservation6.5MiB;logical formula payload≈2.1MiB |
| Decoded/work D_live |All decoded numerical assets/work outsideR2;checked descriptors |Reservation6MiB |
| Outside combined |Rout+E_live+D_live |≈17.6303MiB<24MiB |
| Joint retained |R2+Rout |≈46.383MiB<80MiB |
| Canonical-working constructor allowance |D2-A Cwork≤32m, m≤N |At most2,097,152 bytes (2MiB); per phase recompute32m |
| Constructor peak bound |R2+8N+16m+32m+otherLive+4096; m≤N, otherLive reserved24MiB |≤72,095,482 bytes ≈68.7556MiB<128MiB |

Index/raw-input headroom≈24MiB;R2≈30.75MiB;outside combined≈6.37MiB;joint retained≈33.62MiB;peak≈59.24MiB. E_live6.5MiB leaves1.5MiB, D_live6MiB leaves2MiB and Rout leaves≈2.87MiB under their separate8MiB limits. Only one Employer raw index input is live at a time; its prescribed charge is16m, at most1MiB at the selected N. No fixed16MiB raw buffer is required or reserved. Private canonicalization copy, scratch, variable stack and auxiliary storage collectively fit32m, not a fixed32MiB allowance. All other live buffers enter the separate otherLive charge; they cannot be moved there to evade32m. No buffer's release is assumed without its frozen lifetime charge.

The1MiB outside-bookkeeping allowance is a selected **MODELED ASSUMPTION production reservation**, within frozen architecture. The canonical-working allowance is the normative32m per-index bound; at N=65536 its maximum is2MiB. D2-B2 proves the specified logical asset/temporary layout fits them; it may reduce actual use, not create an unaccounted resource category. Final runtime implementation mechanics and identity encoding remain with their frozen owners.

Selected construction-phase evidence layout: source microdata, Python fitting objects and report-generation heaps are offline production machinery, never constructor input or retained model state. Numerical loading reserves E≤6.5MiB and D≤6MiB together with the full Rout reservation; the decoded reservation uses the frozen primitive-width arrays/row descriptors, bounded remapping and one-law merge scratch, not an unbounded Python JSON object graph. Emit an itemized D charge using authored counts before deduplication, canonicalization scratch and descriptors; parsed/canonical copies are distinct live charges. Outside numerical compilation charges any already-live R2 as well. Company generation charges64+N Industry metadata in Rout,64+4N EmployeeCount inR2 and8N raw count staging. Each of the40 serial Employer phases reserves its prospective retained output inR2, exactly one16m raw input and at most32m total canonicalization work; all outside E/D/Rout and other live loading state are conservatively covered by otherLive≤24MiB. Final retained state keeps Rout+R2 and releases numerical source/decoded temporaries. These are fixed resource-accounting phases, not a new index, sampler, Protocol encoding or fitting method. D2-B2 emits the exact charge table from asset counts and encoding size; later implementation must honor the frozen primitives/lifetimes and these reservations rather than treating the Python fitting process as a runtime constructor.

The compact audit schema §32 bounds text roughly by240,000 value/weight pairs (each value≤2,000,000, each weight≤uint32),76,160 uint16 map entries and categorical rows: below6.5MiB with bounded descriptors. D2-B2 measures exact bytes rather than relying on the estimate. No pretty-printed duplicate target tables are charged as runtime numerical input. Reports are offline evidence artifacts, not retained model input.

Per-condition maximum raw Employer weight is≤4096×2048×328; combined totals atN65,536 fit uint64, but checked arithmetic remains mandatory. Literal worst frozen-factor bounds are also audited. Law/point deduplication only reduces these projections. No scenario uses512 points in every law. If any reservation or normative limit fails, revise fitting/support allocation in D2-B1; D2-A remains intact.

## 32. D2-B2 production contract

Mechanically compute and emit:

1. CompanyCount65,536;one14-state Company Industry vector;14 EmployeeCount laws;all8 lower-bound and55-entry compiled age mappings.
2. Education120 rows;Activity1120 rows;Employee family140 rows;Employee level420 rows;SE family140 rows;SE Industry10 rows;canonical uint32 categorical weights.
3. Employer C4096;I10×14=64M;S4×4 listed in §19;cap-profile trace,40 analytic relationship target tables,employee exposure/error and boundary-pressure report. No new Employer index implementation is produced in this numerical phase.
4. Each Income branch's complete tuple map and branch-owned finite law pool;employee zero-band sentinel65535;law counts within fixed caps;supports/weights within fixed per-law and aggregate envelopes;canonical within-branch deduplication and ID order.
5. Source manifest,all original/normalized snapshot hashes,crosswalk manifest/expanded files,fit configuration and scripts with exact versions,fit metrics at every stage,resource manifest,`plausible.md` quality report andlogical asset SHA-256 checksums. The quality report must enumerate the §30 matrix IDs, dimensions, severities, modeled masses, material/rare/segment membership, values/limits and findings; prominently list every WARNING separately from HARD failures and DIAGNOSTIC records. No frozen artifact prescribes `plausible-plus.md`; historical reports keep their original names/requirements.

Canonical logical audit schema is one compact UTF-8 JSON document `D2B2-logical-assets-v0.json` with these top-level keys (keys serialized lexicographically):

```
{
  "schema":"SW-L2-D2B2-logical-v0",
  "companyCount":65536,
  "ages":{mappingName:{"lowerBounds":[...],"compiled":[...55 bucket indices...]}},
  "companyIndustry":{"weights":[...14...]},
  "employeeCount":{"laws":[{"values":[...],"weights":[...]},...14...]},
  "education":{"weights":[...flat canonical rows...]},
  "primaryActivity":{"weights":[...]},
  "employeeJobFamily":{"weights":[...]},
  "employeeJobLevel":{"weights":[...]},
  "selfEmployedJobFamily":{"weights":[...]},
  "selfEmployedIndustry":{"weights":[...]},
  "employer":{"cap":4096,"industryFactors":[...140...],"sizeFactors":[...16...]},
  "income":{branch:{"tupleMap":[...],"laws":[{"values":[...],"weights":[...]},...]}}
}
```

The schema examples' ellipses are array content generated by the fully specified recipes, not unspecified field names. Mapping names exactly Education,PrimaryActivity,Employee.JobLevel,Income[s],Income[u],Income[t],Income[r],Income[i]. Branch keys e,s,u,t,r,i. All numbers are final integers; laws have sorted strictly increasing values and aligned weights. State arrays follow §3/D2-A. Tuple traversal: employee Region→family→level→Industry→SizeBand;SE age bucket→Region→family→Industry;other Income age bucket→Education→Region;categorical traversal follows its parent order in §3. Law IDs arise from first tuple occurrence after within-branch deduplication. Bucket indices are0..B−1; law IDs are0..L−1. The only unreachable employee reference is65535; no index is one-based. Crosswalk entries sorted by version,source code,frozen state.

Serialization for audit checksum: Python `json.dumps(sort_keys=True,ensure_ascii=False,separators=(',',':'))` followed by one LF, UTF-8, no BOM, integers only. Individual model logical checksums hash exactly `{"schema":"SW-L2-D2B2-logical-v0","model":modelName,"asset":subtree}` under that serialization. `modelName` is the exact top-level asset key for a top-level model, `ages.` followed by the exact mapping name for an age mapping, or `income.` followed by the branch key for an Income model; `subtree` is exactly the referenced JSON value, including `companyCount` as an integer. There are no other fields or whitespace. The whole-document checksum hashes the entire logical audit document under the same serialization. This is a reproducibility/audit encoding, **not SocioeconomicWorldID or Protocol-owned runtime encoding**. Physical implementation does not inherit JSON as a required API representation.

Fit scripts/configuration and report artifacts are versioned D2-B2 evidence; source datasets remain outside the repository unless separately authorized later. This D2-B1 creates no script or data file in the repository. D2-B2 script decisions are implementation of the exact algorithm, not a new modeling choice.

Forbidden without revising D2-B1: new source/release,classification bridge or coefficient,new age boundary,CompanyCount,new fit method/constraint/penalty,Employer cap/factor redesign,new pooling/distance/compression/allocation method,integerization,tail cap,validation tolerance/severity/materiality rule or resource architecture. Failed numerical acceptance is not permission to retune. Final results and reports undergo independent review before any D2-B freeze.

## 33. Reproducibility / source snapshots

Production snapshot policy is fixed now; this revision retains every recorded source URL, release and snapshot hash:

- Official downloaded ZIP/XLSX/PDF: retain exact captured bytes outside the repository with URL,release/reference/publication/access fields,byte count andSHA-256 in §5. Verify before extraction. ZIP container is the primary snapshot; record relevant member bytes/SHA additionally in D2-B2. Never silently reacquire a newer same-URL file.
- Official API: retain the captured raw response plus a normalized official response. Hash both;data/version must match §5/this section. Normalize observations by named coordinate codes,not response row/index order. Do not delete quality flags or use metadata update date as observation period.
- Official publication metadata without indispensable machine bytes (X4):record official standard identity and inspected text provenance. It supplies no production numerical datum. No restricted source is indispensable.

Exact API normalization `SW-OFFICIAL-API-SNAPSHOT-v0`: parse JSON with Decimal;for every Cartesian coordinate in the response, emit {coordinates:{axis:code},value:canonical decimal string ornull,status:official flag ornull}. Numeric string uses non-exponential notation with trailing fractional zeros removed;zero="0". Sort coordinate dictionaries by axis name and observation rows lexicographically over sorted axis names. Output object keys dataset,query={geo:IT,time:2024,sex:T,lang:EN},release=official updated,observations. Exclude presentation labels/unused extension metadata from this normalized payload; preserve all raw bytes separately. Serialize sorted keys,compact UTF-8 JSON plusLF. This prevents API index/row ordering from altering normalized content.

| API Source | Normalized byte count | Normalized SHA-256 |
| --- | ---: | --- |
| O1 |40694 | `04fc8c132e6603f0963a7c405964f3fddb07a1052bf0b82305e9d007778f719a` |
| O2 |137178 | `722403ed096ac15e25ccc46ed95c91f450a0a46be5ac9b17ca6e23c9eb0c0342` |
| O3 |26761 | `b638a147165cd213fcc9c8463391b61ddab4b1289d235e65ef001f4cbf7b1d5a` |

Snapshots captured in `/tmp/krenalis-d2b1-p049clg8` are temporary research inputs, not repository artifacts. D2-B2 must acquire the **same hashes** or use externally archived exact-byte replicas whose manifest records the official origin. If upstream bytes/version change and the pinned snapshot is unavailable,stop numerical acceptance and revise the evidence version through review. A mutable landing page is never the sole provenance.

Reproduction inputs are source snapshots/release identities,expanded crosswalks,fit scripts,configuration and CPython3.13.2/Decimal settings. The D2-B2 offline configuration must record the SHA-256 of the exact independently reviewed D2-B1 document bytes; this binds the in-place v0 revision as well as its named crosswalk recipes. It is provenance, not a Protocol runtime identity seed. No random seeds are required. Re-run from source observations in reverse row order and a deterministic permutation sorted by SHA-256 of UTF-8("D2B1-roworder-v0:"+sourceID+":"+canonicalRowKey), followed by canonical sorting before accumulation;logical asset checksums must match byte-for-byte. A tabulation/API canonicalRowKey is its sheet/cell or full named-code coordinate tuple; a SHIW key is NQUEST/NORD. This is an offline row-order test, not a runtime identity seed. A software change affecting output requires reproducibility verification and D2-B1 recipe revision if results differ,not a silent new canonical checksum. No source microdata or unauthorized evidence dimension becomes runtime state.

## 34. Assumption ledger

Every entry below is **MODELED ASSUMPTION**; cited official evidence supplies context or explicit calibration inputs, not proof of the assumed result. Sensitivity scenarios preserve the frozen parents and taxonomies.

| Decision | Evidence | Assumption | Why needed | Sensitivity | Risk if wrong | D2-B2 mitigation |
| --- | --- | --- | --- | --- | --- | --- |
| CompanyCount | B1 count/size diversity; D2-A resources |65,536 synthetic organizations | Support size is not census or Person scaling |32,768/98,304 comparison | Too much repetition or weak large support | Report expected diversity and finite-N variance; fixed N |
| EmployeeCount source reconciliation | B1 annual addetti; A1 employees | Annual average→anchor workforce shape/mean,3% dormant | No compatible full stock headcount release selected | Zero .015/.045;means±10% | Misstated stock/employee exposure | Preserve terminology;source-unit and cap-profile reports |
| Headcount body/tail | B1 band means,not within-band quantiles | Log-uniform entropy body;tail-stratum constraints;131072 cap | Coarse official distribution | Tail proportions×.5/1.5;cap65536/262144 diagnostic | Too few/too many giant employers | Fixed finite support;tail sensitivity;no runtime clamp/retry |
| Classification-version bridges | X1/X2/X3 | Direct old classification→SW;modeled aggregate G/M/N splits | Detailed duties unavailable in selected numeric rows | Split coefficients±20%;cross-state X3 audit | Repair/security/veterinary mass misplaced | Versioned entries,exact source codes,no implicit many-to-many allocation |
| PrimaryActivity overlap | A1,M1 principal-status concepts | Student-first then retired transport vectors,helper2% | Weekly LFS status does not identify principal-role overlap | Flows×.5/1.5 | Student/retired workers misclassified | Publish raw/principal margins and every flow |
| JobLevel proxying | A1/W2 qualifica;frozen responsibility meanings |[.83,.09,.065,.015] prior plus lifecycle/qualification/function factors | Authority tiers not observed in full accessible grids | Non-IC masses±25% | Inflated responsibility or education stereotypes | Mark all proxies CALIBRATION;report conditional targets and uncertainty |
| Occupation source missingness | O1/O2/O3 pinned cells and TOTALs | Explicit residual/modeled-zero identification treatment and source-internal age completion §8 | Absent NRP is not an observed zero; rounded source counts differ | Missingness/quality report; occupation split diagnostics | A rounding residual could be misread as real nonresponse | Keep raw counts; label completed/residual masses modeled; never DIRECT |
| SE occupational borrowing | O2 independent margins;O3 all-employed association | Half-strength O3 effects;independent O2 fit | Missing SE occupation×Education×Region joint | Association exponent .25/.75 | Employee patterns over-borrowed | Never copy employee law;compare branch differences |
| SE Industry split | A1 independent-person sectors |14-state split §18,positive M affinities | Only five source sectors selected | Broad within-sector proportions±20% normalized | Misplaced self-employment function/sector | Preserve independent-person margins;full crosswalk audit |
| Employer C | A1 employee-size exposure+B1 Company model | Regularized profile selected4096;penalty .01 | C vs band-factor correction not fully identified | .005/.02 penalty;source/stock means±10% | Cap chosen by prior rather than data | Report unpenalized65535 pressure and all profile objectives |
| Employer I/S | O1 family margins;frozen functions;A1 employee exposure | I64M and responsibility size profile,source-calibrated S | Required joint affinities missing | Affinities±25%;DAG vs independent design | Implausible sector/size sorting | All40 margins and unusual-valid support;factor headroom |
| Employee all-source beyond wages | W1/W2 remuneration;H1 personal components | Zero .002,Qe and all-source tails/profile | Salary is only one eligible component | Level stretch about1 EUR .9/1.1;tail excess above q95×.5/1.5;zero .001/.004 | Salary masquerades as Income or extras too large | Personal-source exclusions;salary-vs-all-source concept/level audit |
| Self-employed Income | H1 signed net YM,M1 component definitions | Zero .08,Qs,gross positive earnings/all-source shape | Historical net signed profit is incompatible | Zero .04/.12;Qs±15%;tails×.5/1.5 | Losses miscast as positive gross or turnover | Report negatives;no abs/clamp/turnover substitution |
| Unemployed Income | M1 personal benefits concepts;A1 principal activity | Zero .35,Qu,other personal inflows | Benefit eligibility/history forbidden;all-source rate unidentified | Zero .25/.45;positiveQ±15% | Benefit-only proxy or household leakage | No hidden eligibility;personal-only target audit |
| Student Income | H1 scholarship YTA2,M1 | Zero .65,Qt;no family transfers | Scholarship+other eligible inflows incomplete | Zero .55/.75;positiveQ±20% | Family support wrongly included or positive incomes too common | YTA31 excluded;older student complete assumptions |
| Retired Income | R1 recipient pension totals;H1 YTP1 | Zero .01,Qr,personal non-pension inflows;nominal carry-forward | Pension component is not whole Income or all principal retirees | Zero .005/.02;positiveQ±10%;tail×.5/1.5 | Pension-only law or unsupported extras | Recipient/benefit distinction;arrears/gifts/wealth exclusions |
| Inactive Income | H1 personal component concepts,M1 | Zero .55,Qi,positive personal benefits/property flows | Household resources cannot identify a Person rate | Zero .45/.65;positiveQ±20% | Household/partner/wealth state leaks in | No household allocation or latent tier;report gap |
| Income tails | W1/R1 scale;H1 component cross-check,not compatible top-tail observation | Fixed Q99/Q999/endpoints per branch | Rare all-source tail not identified | Excess above q95×.5/1.5 at last three knots;uint32-checked diagnostic caps | Too heavy/light tails or obvious gaps | Locked tail mass/quantile partitions;stage errors/cap diagnostics |
| Age boundaries | E1/A1 source resolution;R1/H1 lifecycle context | Eight actual maps §§13/14/16/20 | Runtime grids need finite lifecycle buckets | Each internal bound±1 | Artificial discontinuities or unstable marginal fit | Fixed production bounds;quantified sensitivity;no boundary search |
| Small-Region sparse cells | E1/A1 Region margins;O3 national association | Half-shrunk effects,no unseen age×Region interaction | Published marginals do not identify full joints/sample n | Regional exponent .25/.75;Valle d'Aosta/Molise diagnostics | Overfit regional noise or flatten real differences | Complete offline laws;report worst-region metrics,not runtime fallback |
| Income pooling/compression | Frozen finite-law caps;modeled Q targets | Ordered Ward/log metric,128-min/384-high-volume,240k budget | Finite resources and repeated-value realism | Metrics at all stages,512/budget pressure | Visible value gaps or law sharing harms rare segments | Mandatory gap/segment checks;no jitter or new pooling in D2-B2 |
| Nominal vintage carry | W1/W2/R1/H1 different historical bases | No CPI or net-to-gross transform;anchor targets authored | Current annualized concept unidentified | PositiveQ level perturbations above | False precision about2026 price/pay level | Explicit modeled labels;reference/publication dates separated |

## 35. Rejected / downgraded evidence ledger

Rows describe intended uses, so a rejected use can coexist with a selected calibration use of the same publication. Each intended use has exactly one final evidence class.

| Source | Intended use | Final classification | Reason | Replacement / treatment |
| --- | --- | --- | --- | --- |
| B1 enterprise counts | Whole synthetic organization census | CALIBRATION | Covered market enterprises exclude synthetic/non-market sectors | Explicit enterprise-unit extension §11 |
| B1 annual-average addetti | Direct anchor EmployeeCount stock | CALIBRATION | Annual mean,independent+dependent workforce,not instant stock | Stock carry-forward/zero/tail assumptions §12 |
| B1/A1 dipendenti | Total workforce count | CALIBRATION | Employee subset and annual measurement | Use A1 only for employee exposure;do not redefine Company workforce |
| B1 personnel costs/FTE | Personal gross Income/headcount | REJECTED | Employer costs and hours volumes are different concepts | W1/W2 remuneration calibration;modeled all-source prior |
| A1 ILO labour status | Exact principal PrimaryActivity | CALIBRATION | Student/retired overlaps and principal self-description differ | Explicit offline precedence/transport §14 |
| A1/W2 quadri/dirigenti | Exact manager/executive tiers | CALIBRATION | Administrative grade does not imply responsibility | Modeled responsibility prior §16 |
| ISCO OC1/skill ranks | JobLevel law | REJECTED | Occupational function/skill is not organizational authority | Frozen responsibility definitions and explicit prior |
| O1 employee occupations | SE JobFamily law | REJECTED | Wrong working branch | Independently fitted O2,with explicit O3 borrowing |
| Company Industry law | SE Industry law | REJECTED | Enterprise unit ≠ independent-person sector | A1 independent-person margins §18 |
| W1 SES micro-firm population | All employee sector/size coverage | REJECTED | Units<10,A,T,U uncovered;publicO included in Italy | Modeled micro/excluded-sector profile and W2 coverage context |
| W1/W2 salary-only amounts | Direct all-source Person Income | REJECTED | Remuneration omits eligible personal non-salary inflows and current-rate reconciliation | Explicit all-source target prior/transport §§20–21 |
| W1 annual FTFY remuneration | Direct monetary-only personal Income level | REJECTED | Includes in-kind pay and historical full-time/full-year standardization | Retain W1 CALIBRATION only; no inferred cash subtraction |
| W1 SES structural effects | Relative employee monetary plausibility | CALIBRATION | Covered employee remuneration structure only | Shrunk modeled authorized profiles;annual cash/in-kind mix retained as calibration caveat;Age/Education/Gender excluded from e |
| M1 EU-SILC HY010/HY020,HY040G,HY090G | Person gross all-source level/capital allocation | REJECTED | Household aggregates/ownership;no individual attribution | No household division;personal-all-source gap modeled |
| EU-SILC personal gross components in restricted SUF | Sole production canonical evidence | REJECTED | Access/release and current-rate/component exclusions not reproducible from public evidence alone | M1 methodology CROSS-CHECK;no SUF dependency |
| EU-SILC PY050G signed profit/loss | Direct nonnegative attributed business earnings | REJECTED | Signed historical aggregate includes noncash and cannot offset other inflows | Metadata audit only;positive gross-rate assumption,not clamp |
| H1 RFAM/Y household/net aggregate | Public Person Income | REJECTED | Net,household/imputed/noncash/transfer components incompatible | Selected personal component cross-check only |
| H1 RPER YM signed net self-employment | Current gross nonnegative SE Income | CROSS-CHECK | Personal but historical/net/signed;no positive-gross transformation identified | Negative rate and positive-only quantiles reported;no abs/clamp |
| H1 YTA31 gifts/family support | Student or inactive eligible Income | REJECTED | Transfers excluded by Contract5 | Qt/Qi personal-only assumed rates;YTA2 scholarship check |
| H1 YCA2/YCF wealth-derived/imputed resources,YTA6 assigned to NORD1 | Personal eligible capital/benefit Income | REJECTED | Imputation/household assignment not compatible personal ownership/inflow | No wealth-based allocation or principal-recipient fiction |
| R1 benefit mean / D05 ASI pension count tables | Retired Person all-source level | REJECTED | Benefit rather than person;recipient may not have retired primary activity | R1 recipient annualized component calibration;Qr broader prior |
| R1 total pension recipient Income | Entire retired Income concept | CALIBRATION | Pension component/recipient population only;older anchor stock | Explicit other eligible personal-inflow prior |
| ISTAT LFS/SES MFR and Eurostat scientific microdata | Indispensable high-dimensional fit | REJECTED | Restricted scientific access cannot be sole reproducible basis | Accessible published/API margins plus assumptions |
| 2026 January–August wages/employment/benefit outcomes | Anchor2026-01-01 production levels | REJECTED | Post-anchor realized economic leakage | Pin eligible2022–24 observations/2025 publications;2026 methodology only |
| ATECO2025/NACE2.1 release substituted into old code table | Seamless classification refresh | REJECTED | Code relations are many-to-many and2026 correspondence changed | Direct old-source→SW;X3 audit only |
| CP2011 data silently relabeled CP2021 | Occupation crosswalk | REJECTED | Code-vintage mismatch | O1–O3 ISCO08 production route;X1 audit |
| Official Gender-stratified cells | Gender-conditioned canonical model | REJECTED | Frozen dependency forbidden | Total-person tabulations/all weighted micro-records;no carried Gender correction |
| Province/city/urban/rural/coordinates | Runtime socioeconomic geography | REJECTED | Geography beyond authorized Region | Aggregate/discard offline;retain Region only |
| CPI uplift of every monetary source | Automatic nominal anchor conversion | REJECTED | Price index need not measure wage/pension/current-rate change | No index;authored nominal levels and explicit sensitivity |
| Newer isolated2025 occupation margins | Mixed2025 branch counts with2024 attainment/status joint design | REJECTED | Deliberate common2024 demographic reference portfolio;no newer numeric response selected | Fixed2024 snapshot and explicit12-month carry-forward;not a claim2025 data are inherently invalid |

## 36. Adversarial fit-plan review

Required response is deterministic: no runtime fallback,no silent evidence upgrade,no clipping,no new method in D2-B2. “Revise B1” means fail numerical acceptance,document the problem and obtain a reviewed recipe revision;it does not reopen D2-A.

| Attack / risk | Detection | Required response | D2-A feasible? |
| --- | --- | --- | --- |
| Rounded source margins do not sum to1 | Exact retained-cell sum versus published total and IPF constraint total | Normalize retained counts; audit total discrepancy; no solver/tolerance substitution | Yes |
| Missing NRP is silently converted to an observed zero | Raw API null/status and modeled completion provenance | Apply explicit §8 residual/modeled-zero rules; report missing source categories | Yes |
| X3 official unmatched codes are treated as malformed | M=9 plus blank2022 code; both-direction flag validation | Emit UNMATCHED audit records; no allocation or false input rejection | Yes |
| Fixed MiB work budget evades per-index32m bound | Per-phase m and all private/scratch/stack storage | Enforce32m exactly; excess fails even if global peak fits | Yes |
| Small Region sparse cell amplifies noise | Worst Valle d'Aosta/Molise categorical/Income metrics;regional sensitivity | Use fixed half-shrink completed cells;report uncertainty;failure→reviseB1,never runtime fallback | Yes |
| Classification-vintage mismatch | Source version/schema and code-set checks;X3 cross-state audit | Reject replacement/mixed version;pin specified source | Yes |
| Ambiguous Industry repair/veterinary/security mapping | Explicit G/M/N split audit;X2 detailed-code exceptions | Apply authored coefficients and functional exceptions;no residual other-services | Yes |
| Addetti mistaken for EmployeeCount | Unit/time metadata and B1/A1 exposure discrepancy | Keep CALIBRATION;stock/zero assumptions explicit | Yes |
| Student+employee copied as employee | Raw→principal flow report | Apply student allocation first,then retired/remainder | Yes |
| Retired+employee copied as employee | Same,older-age flow audit | Apply retired from residual;no earnings-based status override | Yes |
| Weak JobLevel evidence mislabeled direct | Every grade/ISCO evidence tag and tier-marginal check | Preserve modeled prior/proxy distinction;review uncertainty | Yes |
| SES missing micro firms | W1 coverage versus positive micro tuples | Use modeled micro profile;never claim SES whole-branch direct | Yes |
| SES sector exclusions/publicO error | W1 methodological sectionB–S/ItalyO check | A/T/U modeled;O included;no generic Eurostat exclusion imported | Yes |
| Salary-only Income | Source-component scope and all-source target ledger | Reject salary equality;use explicit all-source prior/transport | Yes |
| Household Income misuse | Variable/unit allowlist and H/D/RPER join audit | Reject household totals/ownership allocation | Yes |
| Negative self-employed profit | Negative source counts;signed-type checks | No abs/clamp/signed offset;cross-check only,modeled positive rates | Yes |
| Student family support | H1 YTA31/gifts exclusion allowlist | Exclude;any inclusion fails acceptance | Yes |
| Pension-only retired law | Eligible-source scope,source recipient vs branch audit | Preserve broader all-source Qr;never equate pension mean | Yes |
| Inactive household-resource leakage | Nohousehold/wealth features infitconfig/maps;component audit | Reject latent household allocation/state | Yes |
| Gender-stratified evidence survives | Final parent/feature allowlist;sex=T/all-record weighting checks | Marginalize first;abort any Gender-conditioned asset | Yes |
| Fine geography survives | Asset/config field scans against Region-only allowlist | Remove before canonical fit;any surviving proxy fails | Yes |
| Old nominal monetary level falsely CPI-updated | Transformation ledger and series allowlist (none selected) | Apply no index;keep assumptions/level sensitivity explicit | Yes |
| Integerization kills rare support | Positive-support set equality before/after;Hamilton reservations | Reserve1,exact remainder/GCD;any loss fails | Yes |
| Employer I/S wants4095 or clipping | Factor bound checks,unpenalized profile and≥90% bound diagnostics | Report pressure;do not clip;selected Imax2048,Smax328;failure→reviseB1 | Yes |
| Employer C wants65535 | Profile objective table and boundary flag | Report observed unpenalized pressure;regularized4096 fixed;no silent cap substitution | Yes |
| Income law wants>512 points | Full greedy refinement sequence,capacity/metric check | Stop at512;failed HARD resolution→reviseB1;fine-tail WARNING reported,not interpolation | Yes;recipe quality may need revision |
| Income compression visible gaps | Central gap,max atom,histogram target occupancy checks | Fixed allocation/refinement;failure→review/reviseB1;no jitter | Yes |
| Resource headroom too small | §31 exact input/retained/peak preflight and reservations | Fail acceptance;revise support allocation/fit inB1 | Yes |
| Restricted source unavailable | Manifest indispensable-source audit | No selected canonical asset needs restricted data | Yes |
| Upstream release changes | Raw/normalized SHA/version mismatch | Use pinned exact snapshot;unavailable→evidence-version revision | Yes |
| APIs reorder observations | Reverse/shuffle source-order reproduction | Canonical sorting;logical checksums must match | Yes |
| Fitted Company law makes target mean infeasible | Support-hull mean check,tail-stratum feasibility | Fail configuration;no support cap/zero change byB2 | Yes;reviseB1 if detected |
| Branch zero quantiles mistaken for positive quantiles | Dual unconditional/positive report | Use exact zb+Q formula;zero target median stays0 | Yes |
| Pooling hides rare tuple error | Material/rare worst cases,protected-segment errors,branch caps and sanity | HARD aggregate/material/sanity failure→reviseB1;rare fidelity WARNING retained,never hidden | Yes |
| Audit-parent mix differs from eventual Layer1 mix | D0 vs documented Layer1 sensitivity,Employer DAG-joint comparison | Report difference;no hidden age/Region proxy added;reviewbasis if tolerance fails | Yes |
| Software or floating point changes integer result | Pinned Decimal settings,second reproducibility run,integer checksum comparison | Reject divergent canonical output;no arbitrary tie/rounding patch | Yes |

## 37. Decision ledger

All authored target/algorithm choices carry the MODELED ASSUMPTION convention in §1. Source-backed inputs are named,not invented. “Compute” below means numerical execution of the fixed recipe;it is not methodological discretion.

| Subject | Selected D2-B1 decision | Source IDs | Modeled assumption? | D2-B2 freedom remaining |
| --- | --- | --- | ---: | --- |
| Source vintages | ASIA2023,FRAME2022,LFS2024,SES2022,INPS2024,SHIW2022;fixed releases/snapshots | B1,E1,A1,O1–3,W1–2,R1,H1 | Yes,vintage carry | Acquire same hashes/extract exact cells |
| Classification bridges | Direct ATECO22/ISCO08/qualification2024→SW;no production ATECO25/CP bridge | X1–4 | Yes,coarse splits | Expand/hash fixed crosswalks |
| CompanyCount |65,536 | B1;D2-A | Yes | Emit literal |
| Education age |[21,25,35,45,55,65] | E1 | Yes | Compile55 entries |
| Activity age |[21,25,35,45,55,60,65,70] | A1,R1 context | Yes | Same |
| JobLevel age |[21,25,35,45,55,65] | A1,W2 | Yes | Same |
| Income[s] age |[21,25,35,45,55,65] | H1,A1 | Yes | Same |
| Income[u] age |[21,25,35,45,55,65] | A1,M1 | Yes | Same |
| Income[t] age |[21,25,30,40,55,65] | H1,A1 | Yes | Same |
| Income[r] age |[21,45,55,60,65,70] | R1,H1 | Yes | Same |
| Income[i] age |[21,25,35,45,55,65] | H1,A1 | Yes | Same |
| Company Industry target | Enterprise-count source extension;14 shares §11 | B1,X2 | Yes,extension/splits | Exact arithmetic/integerization |
| EmployeeCount fit |128-point entropy tilt,source band means,.03 zero,131072 cap,locked tail strata | B1,A1 | Yes,stock/body/tail | Enumerate support/solve λ |
| Education fit | Half-shrunk normalized product plus7-state split | E1 | Yes | Generate120 rows |
| Activity fit | Raw-age KL/IPF then principal-role transport | A1,E1,M1 | Yes | Fit/transport1120 rows |
| Employee JobFamily fit | O1 marginal,O3/E1 half effects,KL/IPF | O1,O3,E1 | Yes,crosswalk/effects | Fit140 rows |
| Employee JobLevel fit | Responsibility prior KL/IPF,[.83,.09,.065,.015] | A1,W2 proxy | Yes | Fit420 rows |
| SE JobFamily fit | Independently O2 marginal,O3/E1 half effects | O2,O3,E1 | Yes | Fit140 rows |
| SE Industry fit | Independent-person five-sector→14 target;KL/IPF withM | A1,O2,X2 | Yes,splits/joint | Fit10 rows |
| Employer C |4096,source-calibrated regularized profile | A1,B1,O1 | Yes,penalty/design | Reproduce trace/verify literal |
| Employer factor fit | I64M;listed16 S;profile correction/half-up,no normalization | A1,B1,O1 | Yes,affinities/tier profile | Reproduce156 integers/margins |
| Income source hierarchy | Personal all-source concept;component calibration subordinate;explicit Q prior | W1–2,R1,H1,M1 | Yes,current all-source gap | Compute target transport/maps/laws |
| Monetary-vintage treatment | No CPI/index/net-gross conversion;nominal anchor Q authored | W1–2,R1,H1 | Yes | Apply no monetary adjustment |
| Income compression | Locked quantile intervals,conditional-mean ceiling,greedy exactW1 refinement | Modeled Q;D2-A | Yes | Compute selected finite points |
| Income support allocation |128-min,384-top population decile,32-slotblocks,max512,global240k | D0/modeltargets | Yes | Execute deterministic priority allocation |
| Income tuple pooling | Adjacent ordered Ward/logQ;fixed branch caps,barycenter refit,ties | D0/modeltargets | Yes | Numerical clustering only |
| Seed-floor/fit precision | Exact model-specific floor locations; Decimal96 iterative work, final40-place rational rows; pre-integer D0 | §§8/9 | Yes | Execute prescribed sequencing only |
| Missing occupation cells | Exact source-internal age completion; stated NRP residual/modeled-zero treatment | O1–3 | Yes,completion | Reproduce cells/targets; no new missingness choice |
| Integerization | Positive-reserving Hamilton M=2^32−1;exact remainders/GCD;Employer exception | D2-A,Contract5 | Yes,ordinary approximation rule | Exact allocations/checks |
| Source snapshot policy | Pinned official files+SHA;raw and normalized API+SHA | §5/33 | No,newsource forbidden | Reacquire identical bytes/normalize |
| Validation tolerances/severity | §§9,12,27,28,30,31 fixed;Plausible HARD/WARNING/DIAGNOSTIC matrix and mechanical materiality | D2-A/modeltargets | Yes,plausibility policy | Evaluate/report;no retuning |
| Resource envelope |N65536,Uincome≤240k,Rout≈5.13MiB,E6.5,D6,Cwork≤32m≤2MiB,peak≈68.76MiB | D2-A formulas | Yes,reservations;normativecaps no | Exact byte/charge preflight |
| Logical audit format | Compact sorted integer-only UTF-8 JSON schema §32 | D2-A logical order | Yes,audit serialization | Encode/checksum;no WorldID encoding decision |

## 38. Independent-review questions

1. Are selected source concepts genuinely compatible with frozen facts?
2. Is anchor-date discipline correct?
3. Are classification vintages reconciled explicitly?
4. Is EmployeeCount correctly distinguished from addetti/persons employed/FTE?
5. Is Gender completely absent from final model conditioning?
6. Does finer geography disappear before canonical assets?
7. Are all crosswalks versioned and reviewable?
8. Is PrimaryActivity precedence semantically correct?
9. Is JobLevel evidence/proxying honestly characterized?
10. Are employee and self-employed occupational assets independently fitted?
11. Is Company Industry fitted on the correct unit?
12. Is CompanyCount a defensible synthetic support size?
13. Are Employer factors jointly fitted with the Company model?
14. Does Employer fit have adequate factor headroom?
15. Is C determined empirically rather than arbitrarily, with its regularization/non-identification honestly disclosed?
16. Is salary evidence correctly subordinate to all-source Income semantics?
17. Is household Income excluded as direct Person Income?
18. Are self-employment losses reconciled correctly?
19. Is student family support excluded?
20. Is retired Income broader than pension alone?
21. Is inactive Income fit without hidden household state?
22. Is Income compression sufficiently high-resolution?
23. Is law pooling deterministic and reviewable?
24. Are integer weights faithful to fitted probabilities?
25. Are modeled assumptions explicit and sensitivity-tested?
26. Are source snapshots reproducible?
27. Does final fit leave resource headroom?
28. Does D2-B2 require no new architecture or fitting decision?
29. Can another team reproduce the same logical numerical assets from the recorded source snapshots/configuration?
30. Is D2-B1 ready to become the fixed empirical/fitting basis for D2-B2?

> Can D2-B2 now materialize every canonical numerical asset without making any new semantic, evidence-selection, crosswalk, bucketing, fitting, compression, pooling, integerization or resource-architecture decision?

## 39. D2-B1 verdict

**Complete proposed specification, ready for evidence/fit review; not frozen.** All15 model cards,8 actual age maps,CompanyCount,classification crosswalks,target inputs,fitting/compression/pooling/integerization rules,validation tolerances and numerical-resource reservations are selected. D2-B2 has numerical execution work and mandatory HARD/WARNING/DIAGNOSTIC reporting,not a menu of modeling choices.

Source coverage is strongest for enterprise-count/band statistics,completed attainment,labour-status margins and independently observed employee/SE occupational marginals. JobLevel responsibility is an explicit proxy-informed modeled prior. All-source current gross personal Income is not directly identified by the accessible selected evidence: all-source quantiles,zero rates,relative profiles,nonworking rates and tails are explicit assumptions with component calibration/cross-checks. Company headcount's annual-average→stock bridge,missing-sector organization sizes,principal-activity overlaps and Employer affinity/cap identification also need substantive independent review.

The Employer pilot detected unpenalized cap pressure at65535;the selected regularized proposal uses4096 with Imax2048,Smax328. Income pooling/compression has fixed high-volume resolution and240,000-point envelope. The corrected resource projection respects the per-index32m bound and all aggregate frozen bounds with substantial retained/peak headroom;canonical numerical metrics are not falsely claimed passed before D2-B2 computes them.

The focused offline quality-policy dry run preserves the numerical generator and reports unresolved HARD findings: Income[e]/[s] numerical-reference G accuracy and Income[u]/[r] material-tuple normalized W1. The confirmed RETRY2 student median deviation is a rare-tuple WARNING under the generic policy. Compression, finite-law sanity and canonical D2-B2 success are not asserted; see the redesign report for measured scopes and exact confirmations. No generator change is made to improve these findings.

Repository frozen authority and pre-existing changes remain untouched. The authorized repository changes for this revision are this specification, the redesign report and the subsequently requested final-response file named in §1. The offline package sanity command passed;it validates the pre-existing package,not an unimplemented Layer2 model. Reproducibility manifests contain official URLs,reference/release/publication/access distinctions,download byte counts andSHA-256 values,plus normalized API hashes. No downloaded dataset or research script is persisted in the repository.

## Deferred strictly to D2-B2

Mechanical extraction of pinned inputs;exact decimal fits and overlap transport;bulk support/weight enumeration;compiled55-entry age maps;Income quantile transport,deterministic cluster membership,barycenters and budget allocation;canonical integerization/deduplication andlaw IDs;source/crosswalk/fit/resource manifests;compact logical numerical audit assets;exact logical checksums;stage/segment/overall metrics andPlausible reports;reproducibility rerun andresource evidence. These are computations under this recipe.

No source,classification/bucketing/fitting/compression/pooling/integerization/resource decision is deferred. No Layer2 implementation,Protocol identity encoding,architecture amendment,D2-B freeze,staging orcommit is performed. An evidence/numerical failure requires revising D2-B1 through review rather than making a new decision inside D2-B2.

## Repository-integrity proof

The later user instruction authorizes this one new root Markdown document. Repository-integrity comparison therefore requires all pre-existing content and Git metadata to remain identical, with exactly this document added as an untracked file. Full before/after evidence and binary diffs were captured under `/tmp/krenalis-d2b1-p049clg8`; unrelated files were neither removed nor cleaned. Git inspection used `GIT_OPTIONAL_LOCKS=0`.

| Evidence | Before | After | Result |
| --- | --- | --- | --- |
| Branch | `property-semantics` | `property-semantics` | Identical |
| HEAD | `76fa0fff109ca6abad62f55c8761275455842dc5` | `76fa0fff109ca6abad62f55c8761275455842dc5` | Identical |
| Raw Git index SHA-256 | `9bf5ddbc729f55523ac70792666243018b250ae81053a8e4d741071e003d81c3` | `9bf5ddbc729f55523ac70792666243018b250ae81053a8e4d741071e003d81c3` | Identical; no staging |
| Full porcelain-v2 branch/status | 56899 lines; SHA-256 `3282b920f29312968e118c4952f26110d086a46fa1a43a7913e162108509be39` | 56900 lines; SHA-256 `510707f01398478426ae57d1988c1c08499b614032e940b6dee4405ea8f15fef` | Exactly one new untracked document |
| After status with only `? SYNTHETIC_WORLD_V1_LAYER2_D2B1_v0.md` removed | — | SHA-256 `3282b920f29312968e118c4952f26110d086a46fa1a43a7913e162108509be39` | Byte-identical to full before status |
| Working-tree binary diff | 1432 bytes; SHA-256 `0311e8b4c5aa5eb470112a26e96e24b1e2946ac350b35851676992bbe61eb6f4` | 1432 bytes; SHA-256 `0311e8b4c5aa5eb470112a26e96e24b1e2946ac350b35851676992bbe61eb6f4` | Pre-existing diff unchanged |
| Cached binary diff | 0 bytes; SHA-256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 bytes; SHA-256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | Identical |
| `tools/types/semantic.go` SHA-256 | `06a7ff690b3e12cdaf623e3ffb36003d1675a6aec69b3ee59e9ead428cc17842` | `06a7ff690b3e12cdaf623e3ffb36003d1675a6aec69b3ee59e9ead428cc17842` | Identical |
| `semantic.go` binary diff against HEAD | 1432 bytes; SHA-256 `0311e8b4c5aa5eb470112a26e96e24b1e2946ac350b35851676992bbe61eb6f4` | 1432 bytes; SHA-256 `0311e8b4c5aa5eb470112a26e96e24b1e2946ac350b35851676992bbe61eb6f4` | Pre-existing semantic diff unchanged |
| Contracts 1–5 and D2-A | All six SHA-256 values recorded in §1 | All six rehashed to exactly the §1 values | Identical |

The complete status captures, not a filtered summary alone, were compared. All captured authority file hashes, raw index bytes, branch/HEAD, cached and working-tree diffs, and `semantic.go` content/diff remained identical. The new document is untracked; it is not included in the pre-existing working-tree binary diff. No commit or frozen-specification edit was made. Downloads, API responses, public-use data, extraction scripts, pilot calculations, caches and research artifacts remain under `/tmp`; none was added to the repository.

Offline repository sanity command:

```bash
GOCACHE=/tmp/krenalis-d2b1-gocache \
GONOSUMDB='*' \
GOPROXY=off \
  go test ./tools/fakedata
```

Exit status **0**; result `ok github.com/krenalis/krenalis/tools/fakedata 13.321s`. Dependency resolution remained offline. This is package sanity evidence for the existing repository, not a numerical fit test or an implementation of Layer 2.

**Authorized in-place review/correction:** the user then requested direct corrections to this specification. Review-start document SHA-256 was `b39e2c8fbf3e075404b786adf57a07753874a09d5ab38777af718c5ddd75784a`. Only this already-untracked document was edited; its v0 status remains proposed and unfrozen. The revision corrected the per-index32m canonicalization bound and constructor peak, explicit H/E/B attainment column order, retained-cell normalization instead of rounded totals, occupation missing/NRP treatment, CP2021 and cash/in-kind SES metadata, set-valued X3 hierarchy/unmatched-code audit, precise floor/rounding/fit sequencing, Ward-distance and tie rules, duplicate-probability merging before apportionment, support reservation accounting, quantitative finite-universe risks, Income metrics and monotonic sensitivity scenarios. Source releases/hashes, frozen taxonomies/parents, CompanyCount and eight production age maps remain unchanged. The Employer proposal still reproduces C4096 and the sixteen listed S values. These are recipe corrections, not Layer-2 implementation or D2-B2 asset production.

Review verification: all39 numbered sections and15 complete cards passed structural checks; all19 recorded downloaded file sizes/SHA-256 values and three normalized API hashes matched the original captures. The X2 hierarchy covered every matched X3 partner, with3 official M=9 unmatched records handled separately; all coverage flags were checked. Exact count reconciliation and strictly increasing Income level/tail diagnostics were checked. A Decimal96 Company-group/cap-profile proposal check using the revised source-derived family margin reproduced the selected cap/factors; neighbouring objective values were approximately .622882 at2048,.587778 at4096,.595463 at8192. Exact resource arithmetic gives a72,095,482-byte projected peak (68.7556MiB), with canonicalization work bounded by32m≤2MiB. Full canonical pooling/compression metrics remain numerical D2-B2 work and are not claimed passed. No new Go code changed, so the successful offline package test above remains historical and was not rerun for this document-only review.

| Review integrity evidence | Before review | After corrections | Result |
| --- | --- | --- | --- |
| Branch / HEAD | `property-semantics` / `76fa0fff109ca6abad62f55c8761275455842dc5` | Same | Identical |
| Full porcelain-v2 status | 56900 lines; SHA-256 `510707f01398478426ae57d1988c1c08499b614032e940b6dee4405ea8f15fef` | Same full bytes and hash | Identical; document already untracked |
| Raw Git index SHA-256 | `9bf5ddbc729f55523ac70792666243018b250ae81053a8e4d741071e003d81c3` | Same | Identical; no staging |
| Working-tree binary diff | 1432 bytes; SHA-256 `0311e8b4c5aa5eb470112a26e96e24b1e2946ac350b35851676992bbe61eb6f4` | Same | Pre-existing diff unchanged |
| Cached binary diff | 0 bytes; SHA-256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | Same | Identical |
| Contracts1–5 / D2-A / semantic.go | Exact captured hashes above | Each rehashed identically | Frozen authority and existing semantic changes untouched |
| semantic.go binary diff | 1432 bytes; SHA-256 `0311e8b4c5aa5eb470112a26e96e24b1e2946ac350b35851676992bbe61eb6f4` | Same | Identical |

A before-content copy and complete document diff, together with full before/after Git snapshots, were retained temporarily under `/tmp/krenalis-d2b1-review-rt8_i3bp`. Every diff hunk was checked against an identified specification defect or its mechanical ledger/proof synchronization. No unrelated file was cleaned, modified, staged or committed; all review scripts and checks remain outside the repository. D2-B2 must bind the exact subsequently reviewed D2-B1 document hash in its offline configuration.

**Authorized surgical source-label rectification:** pre-rectification SHA-256 was `5a3141fb0716c0108af2ac92d5cc5895f9443a2969767b4b9a46d2f396dd5c8b`. The first D2-B2 execution stopped on A1's combined Trentino label because generic normalization left one space after `/`. Section7 now adds only the exact normalized-label alias `Trentino-Alto Adige/ Südtirol` → `Trentino-Alto Adige/Südtirol`, after generic normalization and before exact lookup. Existing aliases, parent/child precedence and all numerical/source/model/resource/tolerance decisions remain unchanged. D2-B1 v0 and SW-REG-IT-v0 remain proposed/unfrozen and retain their identifiers; the rectified document bytes require a new SHA-256 binding and delta review. The Astra review and first D2-B2 failure report remain historical evidence and are not edited.

**Authorized surgical ATECO code-parsing rectification:** pre-ATECO-rectification SHA-256 was `77b15639955b4b2b9e7181c65699b5e6df7bcae49475ad80e7d020c7fe58633e`. D2-B2 RETRY1 stopped before fitting because X2 code-cell text retained terminal U+0020 padding absent from the corresponding X3 code text. Section6 now explicitly removes only leading/trailing U+0020 from X2/X3 ATECO code identifiers, preserves raw text for audit, uses canonical identity throughout the classification audit and rejects conflicting canonical-code collisions under the existing table semantics. Source snapshots/hashes, classification mappings/coefficient splits, numerical targets, fitting/compression/pooling/integerization rules, tolerances, resources, Region normalization and the Trentino rectification remain unchanged. D2-B1 v0 and SW-IND-ATECO22-v0 retain their identifiers and proposed/unfrozen status; the new document SHA binds this in-place rectification and requires delta review. The historical reviews, Trentino report and both D2-B2 failure reports remain unchanged. Closure fixtures cover only the three recorded code pairs; the complete compatibility audit and numerical production remain for the next D2-B2 run.

## Revision history — intentional Plausible quality-policy redesign

2026-09-17: previous quality target **Plausible+**; new quality target **Plausible**. Pre-redesign SHA-256: `518506745dac7fd017683f41dc19cabdb12ac2735776315d88822d79d387c645`. The product requirement changes rare-cell/fine-tail acceptance severity while preserving semantic authority, all numerical model recipes and all aggregate thresholds. **This is an intentional product-requirement revision. It does not retroactively make the former Plausible+ specification successful.** RETRY2 remains a historical failure under that specification. D2-B1 v0 remains proposed/unfrozen; no D2-B2 SUCCESS or freeze is asserted. Current scope, dry-run availability/results, old/new hashes and repository-integrity proof are recorded in `SYNTHETIC_WORLD_V1_LAYER2_D2B1_PLAUSIBLE_REDESIGN_REPORT.md`; historical integrity records above remain unchanged.

L2 D2-B1 PLAUSIBLE QUALITY REDESIGN COMPLETE — READY FOR INDEPENDENT DELTA REVIEW

## Revision history — surgical M00 scope clarification

2026-09-17: independent Plausible quality delta review identified the M00 scope ambiguity. Pre-rectification SHA-256: `4ea846d16b987c18aa1565028b17985ebfe17d424c72be6a235cbd5b3472641b`. M00 explicitly applies to every material full tuple regardless of protected-segment membership; M01 remains additional and independent. This is a clarification/correction of the already intended HARD scope, retaining the M00 change-ledger classification UNCHANGED. No numerical threshold, materiality rule or generator behavior changed.
