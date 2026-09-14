# Authentic selected-record fakefacegen fixture

This directory is test evidence and is not itself part of the fakefacegen
format.

```text
source:
    tools/fakefacegen/output/catalog-test

source fakefacegen revision:
    8af8b51e0077fc3570f18c17d1c7cf73c58a3ffa

catalog_version:
    1

spec_version:
    spec-v1

prompt_version:
    portrait-v1

selected IDs:
    face-000013
    face-000037

fixture type:
    authentic selected-record fixture
```

`catalog.json`, `specs.json`, and `manifest.json` are verbatim global files.
Only image assets for the selected IDs are included. This directory is not a
complete fakefacegen catalog.

The fixture represents an explicitly selected, quiescent local snapshot. Batch
state, requests, and remote result files are outside its source contract. A
future retry creates a new snapshot after its results have been materialized.

`PhotoID` is catalog-local and remains the source ID without a catalog prefix.
The same ID may identify different bytes in another catalog. Complete asset
identity will be the FaceCatalog checksum, `PhotoID`, and photo size; `PhotoID`
is not a person identity key.

Metadata SHA-256 digests:

```text
catalog.json   1644a688a2a8729acf1335adf634842100d9cdaafb1394e098fc16b74a3c7c59
specs.json     3a085f51a8a70c8773faae2a4858c04378ab4797c7ef0c353d55be3a4ec0396c
manifest.json  b79554fd1c1df6cf947a36f2761cf21ecc347b47b5a51ea3d9742ebfd4479bd6
```
