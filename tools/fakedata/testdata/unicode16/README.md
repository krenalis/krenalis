# Unicode 16 conformance data for D1-B

`repertoire.json` is a small conformance fixture derived from these normative
Unicode 16.0.0 files, with their SHA-256 values pinned by `generate.py`:

- https://www.unicode.org/Public/16.0.0/ucd/UnicodeData.txt
- https://www.unicode.org/Public/16.0.0/ucd/CaseFolding.txt
- https://www.unicode.org/Public/16.0.0/ucd/DerivedNormalizationProps.txt

Regenerate offline after saving the three originals into a directory:

```sh
python3 tools/fakedata/testdata/unicode16/generate.py /path/to/ucd16
gofmt -w tools/fakedata/international_unicode.go
```

The generator uses the pinned files, not Python's Unicode database. The fixture
contains the 656 D1-B-admitted scalars, their 677-scalar normalization/folding
closure, all 505 canonical compositions in that closure, and long combining
sequences. Tests compare the existing vendored library with these Unicode-16
expectations. Production validation restricts the display repertoire before
normalization. No atom is regenerated and no stored display is normalized.

Ordinary NFC is required. The local bounded display helper uses the verified
scalar and pair operations without x/text's whole-string Stream-Safe CGJ
insertion at 30 nonstarters. This restriction applies to D1-B display keys,
not to international JSON strings or C1.

Unicode data © Unicode, Inc.; see https://www.unicode.org/license.txt .
This fixture is conformance data, separate from the Italian production assets.
