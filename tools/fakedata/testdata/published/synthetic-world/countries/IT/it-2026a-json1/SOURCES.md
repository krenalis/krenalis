# it-2026a empirical data sources

This directory freezes the human-approved D1-A empirical assets for Synthetic
World v1. Acquisition metadata, physical bindings, audit results, and review
evidence are preserved under `provenance/`. Raw source archives are not stored
in Git.

## ISTAT POSAS

- Purpose: age distribution and municipality population.
- Reference date: 2026-01-01.
- Data status: 2026 estimate, as documented by ISTAT in the acquired product.
- License: CC BY 4.0.

`age_weights.csv` and the population column in `places.csv` come from POSAS.
They do not depend on postal sources or postal overrides.

## ISTAT SITUAS

- Purpose: territorial universe, region, Province/UTS, and `StateProv`.
- Territorial snapshot: 2026-01-01.
- License: CC BY 4.0.

`StateProv` is the canonical two-letter Province/UTS abbreviation associated
with the snapshot entity. It is not a claim of historical vehicle-registration
legal effectiveness at the exact snapshot date.

## ISTAT Sardinia crosswalk

The official ISTAT Sardinia crosswalk supplies the 377 municipality-code
recodings used to align POSAS population records with the approved SITUAS
territorial universe. The source locator, frozen hash, and physical row binding
are recorded in the D1-A provenance artifacts.

## GeoNames

- Purpose: primary automatic postal observations.
- Reference date: not declared.
- Retrieved-at timestamp: recorded in `provenance/source-lock-d1a.json`.
- License: CC BY 4.0 declaration preserved with the acquisition evidence.

GeoNames is not treated as an authoritative complete Italian postal inventory.

## IPA Enti

- Purpose: evidence for the 147 reviewed postal overrides.
- Reference date: not declared.
- License: CC BY 4.0.

An IPA row proves an observed CAP associated with the exact
`Codice_comune_ISTAT`; it does not prove that the resulting set is the
municipality's complete postal inventory.

## Attribution

Reuse and redistribution must credit ISTAT for POSAS, SITUAS, and the Sardinia
crosswalk; GeoNames for the postal export; and AgID/IndicePA for IPA Enti.
Retain the CC BY 4.0 notices and source links recorded in the source locks.

## D1-B curated plausibility assets

`first_names.csv`, `last_names.csv`, and `street_names.csv` are the frozen `it-2026a` D1-B curated assets. First names are evidence-informed curated; surnames and street fragments are original curated. Reviewer B approved the consolidated package, and curator A plus Reviewer B attested the final manifest hashes. `output_license_status = deferred_to_project_policy`.
