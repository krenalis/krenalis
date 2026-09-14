package fakedata

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const fakefacegenFixtureDirectory = "testdata/fakefacegen-v1-selected"

var (
	fakefacegenFixtureMetadataSHA256 = map[string]string{
		"catalog.json":  "1644a688a2a8729acf1335adf634842100d9cdaafb1394e098fc16b74a3c7c59",
		"manifest.json": "b79554fd1c1df6cf947a36f2761cf21ecc347b47b5a51ea3d9742ebfd4479bd6",
		"specs.json":    "3a085f51a8a70c8773faae2a4858c04378ab4797c7ef0c353d55be3a4ec0396c",
	}
	fakefacegenFixtureFiles = []string{
		"FIXTURE.md",
		"catalog.json",
		"derived/128/face-000013.webp",
		"derived/128/face-000037.webp",
		"derived/256/face-000013.webp",
		"derived/256/face-000037.webp",
		"derived/512/face-000013.webp",
		"derived/512/face-000037.webp",
		"derived/64/face-000013.webp",
		"derived/64/face-000037.webp",
		"manifest.json",
		"masters/face-000013.webp",
		"masters/face-000037.webp",
		"specs.json",
	}
	fakefacegenFixtureGoldens = []fakefacegenGoldenFace{
		{
			ID:           "face-000013",
			Gender:       fakefacegenGenderFemale,
			RequestedAge: 21,
			Assets: [5]fakefacegenGoldenAsset{
				{
					Size:     fakefacegenPhotoSize64,
					ByteSize: 1552,
					SHA256:   "3943d7c72110e8091609371dc6bab93a8a667530caca03822d5fed78c89ae9ea",
					Path:     "derived/64/face-000013.webp",
				},
				{
					Size:     fakefacegenPhotoSize128,
					ByteSize: 4326,
					SHA256:   "c10fbd0727b2f65252c4b0228750e1a4bf8637f5ab6570644ab7674819089f1a",
					Path:     "derived/128/face-000013.webp",
				},
				{
					Size:     fakefacegenPhotoSize256,
					ByteSize: 11394,
					SHA256:   "193edf01c48b1c06c36be7ced9b3f6a3182c87dcac1eb095802ad70090a41755",
					Path:     "derived/256/face-000013.webp",
				},
				{
					Size:     fakefacegenPhotoSize512,
					ByteSize: 32238,
					SHA256:   "702b297c53aa60b0409ebaba66587123dac7f0a1d9e6e72146bc7742b90e235b",
					Path:     "derived/512/face-000013.webp",
				},
				{
					Size:     fakefacegenPhotoSize1024,
					ByteSize: 993432,
					SHA256:   "e347b2e21cf547f4937aa196980ab8710c15038a7e95a68f20272ffaa6ea1529",
					Path:     "masters/face-000013.webp",
				},
			},
		},
		{
			ID:           "face-000037",
			Gender:       fakefacegenGenderMale,
			RequestedAge: 68,
			Assets: [5]fakefacegenGoldenAsset{
				{
					Size:     fakefacegenPhotoSize64,
					ByteSize: 1154,
					SHA256:   "da90d38171489eb26be67abba14bad8f560c8dac6c88da9587a1591836043dc1",
					Path:     "derived/64/face-000037.webp",
				},
				{
					Size:     fakefacegenPhotoSize128,
					ByteSize: 3028,
					SHA256:   "6a9176916d97394ae3bcf7e1d2593676138a0d980434076dc22a12d4cd060981",
					Path:     "derived/128/face-000037.webp",
				},
				{
					Size:     fakefacegenPhotoSize256,
					ByteSize: 9286,
					SHA256:   "30ea6ef94c5dbeea74c4a7b7b236428b20fee01578142be75c6f5238453b4e86",
					Path:     "derived/256/face-000037.webp",
				},
				{
					Size:     fakefacegenPhotoSize512,
					ByteSize: 34550,
					SHA256:   "5f26843865fc3010e55e529a9500c76dae6fb79b212ecdd98017a00a0479dfa7",
					Path:     "derived/512/face-000037.webp",
				},
				{
					Size:     fakefacegenPhotoSize1024,
					ByteSize: 1145688,
					SHA256:   "7fa4f0015a6d5111d48ff8182ed1cc4ab8a00458d20816eba72b3dbbf5ce61de",
					Path:     "masters/face-000037.webp",
				},
			},
		},
	}
)

type fakefacegenGoldenAsset struct {
	Size     fakefacegenPhotoSize
	ByteSize int64
	SHA256   string
	Path     string
}

type fakefacegenGoldenFace struct {
	ID           string
	Gender       fakefacegenGender
	RequestedAge int
	Assets       [5]fakefacegenGoldenAsset
}

// TestFakefacegenAuthenticFixture protects the verbatim metadata and images.
func TestFakefacegenAuthenticFixture(t *testing.T) {

	for name, expected := range fakefacegenFixtureMetadataSHA256 {
		actual, err := fakefacegenFileSHA256(filepath.Join(fakefacegenFixtureDirectory, name))
		if err != nil {
			t.Fatalf("expected metadata %s to be readable, got %v", name, err)
		}
		if actual != expected {
			t.Fatalf("expected metadata %s SHA-256 %s, got %s", name, expected, actual)
		}
	}

	var metadata fakefacegenCatalog
	err := readFakefacegenJSON(filepath.Join(fakefacegenFixtureDirectory, "catalog.json"), &metadata)
	if err != nil {
		t.Fatalf("expected authentic catalog metadata, got %v", err)
	}
	if metadata.Count != 50 {
		t.Fatalf("expected catalog count 50, got %d", metadata.Count)
	}
	var specs []fakefacegenSpec
	err = readFakefacegenJSON(filepath.Join(fakefacegenFixtureDirectory, "specs.json"), &specs)
	if err != nil {
		t.Fatalf("expected authentic specs metadata, got %v", err)
	}
	if len(specs) != 50 {
		t.Fatalf("expected 50 specs, got %d", len(specs))
	}
	var manifest fakefacegenManifest
	err = readFakefacegenJSON(filepath.Join(fakefacegenFixtureDirectory, "manifest.json"), &manifest)
	if err != nil {
		t.Fatalf("expected authentic manifest metadata, got %v", err)
	}
	if len(manifest.Assets) != 50 {
		t.Fatalf("expected 50 manifest assets, got %d", len(manifest.Assets))
	}

	files, err := fakefacegenFiles(fakefacegenFixtureDirectory)
	if err != nil {
		t.Fatalf("expected fixture inventory, got %v", err)
	}
	if !slices.Equal(files, fakefacegenFixtureFiles) {
		t.Fatalf("expected fixture files %v, got %v", fakefacegenFixtureFiles, files)
	}

	err = verifyFakefacegenFixture(fakefacegenFixtureDirectory)
	if err != nil {
		t.Fatalf("expected authentic fakefacegen fixture verification, got %v", err)
	}

}

// TestFakefacegenIntegrityMutations verifies required asset failure modes.
func TestFakefacegenIntegrityMutations(t *testing.T) {

	t.Run("missing derivative", func(t *testing.T) {
		directory := copyFakefacegenFixture(t)
		path := filepath.Join(directory, "derived", "64", "face-000013.webp")
		err := os.Remove(path)
		if err != nil {
			t.Fatalf("expected derivative removal, got %v", err)
		}
		_, err = loadFakefacegenV1Selected(directory, []string{"face-000013"})
		if err == nil || !strings.Contains(err.Error(), "open fakefacegen asset") {
			t.Fatalf("expected missing derivative error, got %v", err)
		}
	})

	t.Run("different bytes", func(t *testing.T) {
		directory := copyFakefacegenFixture(t)
		path := filepath.Join(directory, "derived", "64", "face-000013.webp")
		err := os.WriteFile(path, []byte("not a WebP image"), 0o644)
		if err != nil {
			t.Fatalf("expected asset replacement, got %v", err)
		}
		_, err = loadFakefacegenV1Selected(directory, []string{"face-000013"})
		if err == nil || !strings.Contains(err.Error(), "not image/webp") {
			t.Fatalf("expected WebP integrity error, got %v", err)
		}
	})

	t.Run("different bytes of same length", func(t *testing.T) {
		directory := copyFakefacegenFixture(t)
		path := filepath.Join(directory, "derived", "64", "face-000013.webp")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected authentic asset bytes, got %v", err)
		}
		originalLength := len(data)
		data[len(data)-1] ^= 1
		err = os.WriteFile(path, data, 0o644)
		if err != nil {
			t.Fatalf("expected same-length asset replacement, got %v", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected replaced asset metadata, got %v", err)
		}
		if info.Size() != int64(originalLength) {
			t.Fatalf("expected replacement size %d, got %d", originalLength, info.Size())
		}
		err = verifyFakefacegenFixture(directory)
		if err == nil {
			t.Fatalf("expected same-length integrity failure, got %v", err)
		}
	})

	t.Run("wrong dimensions", func(t *testing.T) {
		directory := copyFakefacegenFixture(t)
		source := filepath.Join(directory, "derived", "128", "face-000013.webp")
		destination := filepath.Join(directory, "derived", "64", "face-000013.webp")
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("expected replacement image bytes, got %v", err)
		}
		err = os.WriteFile(destination, data, 0o644)
		if err != nil {
			t.Fatalf("expected wrong-dimension replacement, got %v", err)
		}
		_, err = loadFakefacegenV1Selected(directory, []string{"face-000013"})
		if err == nil || !strings.Contains(err.Error(), "expected 64x64") {
			t.Fatalf("expected dimension integrity error, got %v", err)
		}
	})

	t.Run("manifest master SHA-256 mismatch", func(t *testing.T) {
		directory := copyFakefacegenFixture(t)
		mutateFakefacegenManifest(t, directory, "face-000013", func(asset *fakefacegenManifestAsset) {
			asset.SHA256 = strings.Repeat("0", sha256.Size*2)
		})
		_, err := loadFakefacegenV1Selected(directory, []string{"face-000013"})
		if err == nil || !strings.Contains(err.Error(), "SHA-256 does not match manifest") {
			t.Fatalf("expected manifest SHA-256 integrity error, got %v", err)
		}
	})

}

// TestFakefacegenPhotoIDScope verifies that source IDs remain catalog-local.
func TestFakefacegenPhotoIDScope(t *testing.T) {

	faces, err := loadFakefacegenV1Selected(fakefacegenFixtureDirectory, []string{"face-000013", "face-000037"})
	if err != nil {
		t.Fatalf("expected selected fixture faces, got %v", err)
	}
	for _, face := range faces {
		for _, asset := range face.Assets {
			if asset.PhotoID != face.ID {
				t.Fatalf("expected catalog-local PhotoID %q, got %q", face.ID, asset.PhotoID)
			}
			if strings.Contains(asset.PhotoID, fakefacegenV1SourceVersion) {
				t.Fatalf("expected unprefixed source PhotoID %q, got %q", face.ID, asset.PhotoID)
			}
		}
	}

}

// TestFakefacegenSourceStatuses verifies source-defined snapshot statuses.
func TestFakefacegenSourceStatuses(t *testing.T) {

	faces, err := loadFakefacegenV1Selected(fakefacegenFixtureDirectory, []string{"face-000013"})
	if err != nil {
		t.Fatalf("expected generated source status, got %v", err)
	}
	if faces[0].Status != fakefacegenStatusGenerated || faces[0].Excluded {
		t.Fatalf("expected recognized generated status, got status %q excluded=%t", faces[0].Status, faces[0].Excluded)
	}

	// failed and missing are fakefacegen-defined source statuses, but neither
	// was observed in the authentic local catalogs. These are mutation tests.
	for _, status := range []fakefacegenSourceStatus{fakefacegenStatusFailed, fakefacegenStatusMissing} {
		t.Run(string(status), func(t *testing.T) {
			directory := copyFakefacegenFixture(t)
			mutateFakefacegenManifest(t, directory, "face-000013", func(asset *fakefacegenManifestAsset) {
				asset.Status = status
			})
			faces, err := loadFakefacegenV1Selected(directory, []string{"face-000013"})
			if err != nil {
				t.Fatalf("expected recognized %q source status, got %v", status, err)
			}
			if faces[0].Status != status || !faces[0].Excluded || faces[0].Diagnostic == "" {
				t.Fatalf("expected excluded status %q with diagnostic, got %#v", status, faces[0])
			}
		})
	}

	t.Run("unknown", func(t *testing.T) {
		directory := copyFakefacegenFixture(t)
		mutateFakefacegenManifest(t, directory, "face-000013", func(asset *fakefacegenManifestAsset) {
			asset.Status = "unexpected"
		})
		_, err := loadFakefacegenV1Selected(directory, []string{"face-000013"})
		if err == nil || !strings.Contains(err.Error(), "unsupported status") {
			t.Fatalf("expected unknown source status error, got %v", err)
		}
	})

}

// TestFakefacegenUnsupportedCatalogVersion rejects future source versions.
func TestFakefacegenUnsupportedCatalogVersion(t *testing.T) {

	directory := copyFakefacegenFixture(t)
	path := filepath.Join(directory, "catalog.json")
	var metadata fakefacegenCatalog
	err := readFakefacegenJSON(path, &metadata)
	if err != nil {
		t.Fatalf("expected copied catalog metadata, got %v", err)
	}
	metadata.CatalogVersion = 2
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("expected mutated catalog encoding, got %v", err)
	}
	err = os.WriteFile(path, append(data, '\n'), 0o644)
	if err != nil {
		t.Fatalf("expected mutated catalog write, got %v", err)
	}
	_, err = loadFakefacegenV1Selected(directory, []string{"face-000013"})
	if err == nil || !strings.Contains(err.Error(), "unsupported fakefacegen catalog version 2") {
		t.Fatalf("expected unsupported fakefacegen catalog version error, got %v", err)
	}

}

func copyFakefacegenFixture(t *testing.T) string {

	t.Helper()
	directory := t.TempDir()
	err := os.CopyFS(directory, os.DirFS(fakefacegenFixtureDirectory))
	if err != nil {
		t.Fatalf("expected temporary fixture copy, got %v", err)
	}

	return directory
}

func fakefacegenFileSHA256(path string) (string, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)

	return hex.EncodeToString(digest[:]), nil
}

func fakefacegenFiles(directory string) ([]string, error) {

	files := []string{}
	err := fs.WalkDir(os.DirFS(directory), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)

	return files, nil
}

func mutateFakefacegenManifest(t *testing.T, directory, id string, mutate func(*fakefacegenManifestAsset)) {

	t.Helper()
	path := filepath.Join(directory, "manifest.json")
	var manifest fakefacegenManifest
	err := readFakefacegenJSON(path, &manifest)
	if err != nil {
		t.Fatalf("expected copied manifest metadata, got %v", err)
	}
	found := false
	for index := range manifest.Assets {
		if manifest.Assets[index].ID == id {
			mutate(&manifest.Assets[index])
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected manifest asset %q, got none", id)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("expected mutated manifest encoding, got %v", err)
	}
	err = os.WriteFile(path, append(data, '\n'), 0o644)
	if err != nil {
		t.Fatalf("expected mutated manifest write, got %v", err)
	}

}

func verifyFakefacegenFixture(directory string) error {

	for name, expected := range fakefacegenFixtureMetadataSHA256 {
		actual, err := fakefacegenFileSHA256(filepath.Join(directory, name))
		if err != nil {
			return err
		}
		if actual != expected {
			return fmt.Errorf("metadata %s has SHA-256 %s, expected %s", name, actual, expected)
		}
	}

	ids := make([]string, len(fakefacegenFixtureGoldens))
	for index, golden := range fakefacegenFixtureGoldens {
		ids[index] = golden.ID
	}
	faces, err := loadFakefacegenV1Selected(directory, ids)
	if err != nil {
		return err
	}
	for index, face := range faces {
		golden := fakefacegenFixtureGoldens[index]
		if face.SourceVersion != fakefacegenV1SourceVersion || face.ID != golden.ID || face.Gender != golden.Gender ||
			face.RequestedAge != golden.RequestedAge || face.Status != fakefacegenStatusGenerated || face.Excluded {
			return fmt.Errorf("face %q maps to unexpected evidence: %#v", golden.ID, face)
		}
		for assetIndex, asset := range face.Assets {
			expected := golden.Assets[assetIndex]
			if asset.PhotoID != golden.ID || asset.Size != expected.Size || asset.MIMEType != "image/webp" ||
				asset.ByteSize != expected.ByteSize || asset.Width != uint32(expected.Size) ||
				asset.Height != uint32(expected.Size) || asset.path != expected.Path {
				return fmt.Errorf("face %q size %d maps to unexpected asset evidence: %#v", golden.ID, expected.Size, asset)
			}
			actualSHA256 := hex.EncodeToString(asset.SHA256[:])
			if actualSHA256 != expected.SHA256 {
				return fmt.Errorf(
					"face %q size %d has SHA-256 %s, expected %s",
					golden.ID, expected.Size, actualSHA256, expected.SHA256,
				)
			}
		}
	}

	return nil
}
