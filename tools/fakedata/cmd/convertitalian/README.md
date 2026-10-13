# Italian international JSON v1 conversion

Run from the repository root, with the original local canonical model available:

```sh
go run ./tools/fakedata/cmd/convertitalian \
  -csv tools/fakedata/data/it-2026a \
  -model D2B2-logical-assets-v0.json \
  -out /tmp/italian-json-v1-staging
```

The output directory must not exist. The command checks the original six CSV
hashes and canonical model hash, converts only their values, validates the
resulting bytes with the new typed decoders, and writes `base.json` and
`socioeconomic.json`. Both files use C1 and release `it-2026a-json1`. It reports
byte sizes, physical hashes, logical digests and the six base fingerprints.
It does not write a release descriptor or publish, embed or stage assets in Git.

The socioeconomic conversion compares every projected typed value with the old
decoder's input and every reconstructed frozen canonical byte with the original.
The new document decoders are deliberately Italian consumers of the international
schemas: other profiles are rejected. Documents are bounded to 8 MiB, Text values
to 4096 UTF-8 bytes, and collections to Italian consumer cardinalities. The
existing compiler checks numerical canonicality and retained numerical bounds.
These checks are not a memory or acquisition certification.

Run the complete equivalence proof with the real external model required:

```sh
SYNTHETIC_WORLD_ITALIAN_MODEL="$PWD/D2B2-logical-assets-v0.json" \
  go test ./tools/fakedata -run '^TestInternational' -count=1 -v
```

Without that environment variable the real-model test looks for the same local
file in the repository root and explicitly skips if it is absent. The base
comparison always uses all six real CSV files. Other socioeconomic validation
tests construct artificial models in memory; production JSON is never a fixture.

The six original CSV files remain offline converter inputs and provenance
material. The operational Italian runtime uses international JSON through
External Assets; CSV files are no longer embedded in the binary or used as a
runtime fallback. `generation-v5` is the only non-empty generation path.
