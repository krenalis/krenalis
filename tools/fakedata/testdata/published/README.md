# World runtime fixtures

Existing US release documents, IT base/socioeconomic documents from staging-i and the compatible IT descriptor from staging-256, one shared catalog and its 1,250 original 256px WebP renditions (13,030,260 bytes). These are isolated runtime/E2E test inputs, not an upload directory. No production model, catalog or image was regenerated. The IT descriptor was taken from the compatible existing staging; this does not claim a new verification of the live origin.

manifest.json records paths within the production repository, byte sizes and physical SHA-256 values. Both countries use the same catalog and images. Preserve the USA NOTICE.txt and the Italian SOURCES.md with these fixtures. The full 256px set is needed because E2E workspace identities vary and may select any catalog entry; masters and other renditions are omitted.

## Italian data attribution

The Italian base.json is Krenalis's JSON conversion of the frozen it-2026a CSVs.
Credit ISTAT for POSAS, SITUAS and the Sardinia crosswalk; GeoNames for the postal
export; and AgID/IndicePA for IPA Enti, used for the reviewed postal overrides.
The original SOURCES.md is preserved byte for byte beside the Italian JSONs.
Its CSV and provenance paths describe the original source tree, now kept in
krenalis-simulated-world, rather than files required by the runtime loader.

Source and licence links below reproduce the frozen D1-A and D1-A2 source locks;
they are not a new licence assessment. Their recorded licence is
[CC BY 4.0](https://creativecommons.org/licenses/by/4.0/):

- ISTAT [POSAS](https://demo.istat.it/data/posas/POSAS_2026_it_Tutti_i_file.zip),
  SITUAS [municipalities](https://situas-servizi.istat.it/publish/reportspooljson?pfun=61&pdata=01%2F01%2F2026)
  and [Province/UTS](https://situas-servizi.istat.it/publish/reportspooljson?pfun=64&pdata=01%2F01%2F2026),
  [Sardinia crosswalk](https://www.istat.it/wp-content/uploads/2024/09/Codici-statistici-e-denominazioni-delle-unita-amministrative-della-Sardegna.zip);
  [legal notice](https://www.istat.it/note-legali/).
- GeoNames [Italian postal export](https://download.geonames.org/export/zip/IT.zip),
  [postal README](https://download.geonames.org/export/zip/readme.txt) and
  [licence notice](https://www.geonames.org/about.html).
- AgID/IndicePA [IPA Enti data and licence](https://www.indicepa.it/ipa-dati/dataset/enti/resource/d09adf99-dc10-4349-8c53-27b1e5aa97b6).

Krenalis adapted and curated these inputs before converting them to JSON; these
fixtures do not imply endorsement by the original publishers. The original
D1-B curated-data policy in SOURCES.md is retained without relicensing it.
