// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

const (
	faceCatalogEntryDomain                 = "fakedata/face-catalog-entry/v1"
	faceCatalogSHA256ComponentName         = "face-catalog-sha256"
	faceCatalogSnapshotDomain              = "fakedata/face-catalog-snapshot/v1"
	faceCatalogSnapshotModeComplete        = "complete"
	faceCatalogSnapshotModeSelectedFixture = "selected-fixture"
	faceCatalogVersionComponentName        = "face-catalog-version"
	maxFaceCatalogRecords                  = 100_000
)

var supportedPhotoSizes = [...]PhotoSize{PhotoSize64, PhotoSize128, PhotoSize256, PhotoSize512, PhotoSize1024}

// PhotoSize identifies one supported square photo size in pixels.
type PhotoSize uint16

// Supported public photo sizes.
const (
	PhotoSize64   PhotoSize = 64
	PhotoSize128  PhotoSize = 128
	PhotoSize256  PhotoSize = 256
	PhotoSize512  PhotoSize = 512
	PhotoSize1024 PhotoSize = 1024
)

// PhotoAsset describes one verified public photo asset.
type PhotoAsset struct {
	PhotoID        string
	Size           PhotoSize
	Width          uint32
	Height         uint32
	ContentSHA256  string
	ContentLength  int64
	CatalogVersion string
	CatalogSHA256  string
	Path           string
}

// FaceCatalog is an immutable verified fakefacegen v1 catalog snapshot.
type FaceCatalog struct {
	version        string
	sourceChecksum [sha256.Size]byte
	checksum       [sha256.Size]byte
	mode           string
	dependencies   [2]Component
	faces          []photoFace
	assets         map[photoAssetKey]photoAssetRecord
	paths          map[string]photoAssetRecord
}

// LoadFaceCatalog verifies and loads a complete fakefacegen v1 catalog
// snapshot. Every generated record must provide all supported photo sizes.
// It returns ctx.Err() when canceled or ErrInvalidFaceCatalog when the
// snapshot is invalid.
func LoadFaceCatalog(ctx context.Context, directory string) (*FaceCatalog, error) {
	return loadFaceCatalog(ctx, directory, nil, faceCatalogSnapshotModeComplete)
}

// loadSelectedFaceCatalogFixture loads only explicitly selected records from
// an intentionally incomplete fixture. Duplicate selected IDs are invalid.
func loadSelectedFaceCatalogFixture(ctx context.Context, directory string, selectedIDs []string) (*FaceCatalog, error) {
	return loadFaceCatalog(ctx, directory, selectedIDs, faceCatalogSnapshotModeSelectedFixture)
}

var (
	faceCatalogLoadStepForTest         func(string)
	faceCatalogMetadataCapturedForTest func()
)

func loadFaceCatalog(ctx context.Context, directory string, selectedIDs []string, mode string) (*FaceCatalog, error) {

	absoluteDirectory, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve directory: %v", ErrInvalidFaceCatalog, err)
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	switch mode {
	case faceCatalogSnapshotModeComplete:
	case faceCatalogSnapshotModeSelectedFixture:
		if len(selectedIDs) == 0 || len(selectedIDs) > maxFaceCatalogRecords {
			return nil, fmt.Errorf("%w: invalid selected photo count %d", ErrInvalidFaceCatalog, len(selectedIDs))
		}
	default:
		return nil, fmt.Errorf("%w: invalid snapshot mode %q", ErrInvalidFaceCatalog, mode)
	}

	snapshot, err := captureFaceCatalogMetadata(ctx, absoluteDirectory)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, contextErr
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidFaceCatalog, err)
	}
	if mode == faceCatalogSnapshotModeSelectedFixture && len(selectedIDs) > len(snapshot.specs) {
		return nil, fmt.Errorf("%w: invalid selected photo count %d", ErrInvalidFaceCatalog, len(selectedIDs))
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if faceCatalogMetadataCapturedForTest != nil {
		faceCatalogMetadataCapturedForTest()
	}
	faces, err := loadFaceCatalogFaces(ctx, absoluteDirectory, snapshot, selectedIDs, mode)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, contextErr
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidFaceCatalog, err)
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	sort.Slice(faces, func(i, j int) bool {
		return faces[i].ID < faces[j].ID
	})
	sourceChecksum := sha256.Sum256(snapshot.catalogBytes)
	checksum, err := semanticFaceCatalogChecksum(fakefacegenV1SourceVersion, sourceChecksum, mode, faces)
	if err != nil {
		return nil, fmt.Errorf("%w: semantic catalog identity: %v", ErrInvalidFaceCatalog, err)
	}
	versionDependency, err := StringComponent(faceCatalogVersionComponentName, fakefacegenV1SourceVersion)
	if err != nil {
		return nil, fmt.Errorf("%w: catalog version dependency: %v", ErrInvalidFaceCatalog, err)
	}
	checksumDependency, err := BytesComponent(faceCatalogSHA256ComponentName, checksum[:])
	if err != nil {
		return nil, fmt.Errorf("%w: catalog checksum dependency: %v", ErrInvalidFaceCatalog, err)
	}

	catalog := &FaceCatalog{
		version: fakefacegenV1SourceVersion, sourceChecksum: sourceChecksum, checksum: checksum, mode: mode,
		dependencies: [2]Component{versionDependency, checksumDependency},
		faces:        make([]photoFace, 0, len(faces)), assets: map[photoAssetKey]photoAssetRecord{},
		paths: map[string]photoAssetRecord{},
	}
	for _, face := range faces {
		current := photoFace{
			id: face.ID, gender: Gender(face.Gender), requestedAge: face.RequestedAge,
			status: face.Status, excluded: face.Excluded, diagnostic: face.Diagnostic,
		}
		catalog.faces = append(catalog.faces, current)
		if face.Excluded {
			continue
		}
		for _, source := range face.Assets {

			data, err := readBoundedPhotoFile(
				ctx,
				filepath.Join(absoluteDirectory, filepath.FromSlash(source.path)), fakefacegenMaxImageBytes,
			)
			if err != nil {
				if contextErr := ctx.Err(); contextErr != nil {
					return nil, contextErr
				}
				return nil, fmt.Errorf("%w: read verified asset %s/%d: %v", ErrInvalidFaceCatalog, face.ID, source.Size, err)
			}
			digest := sha256.Sum256(data)
			if int64(len(data)) != source.ByteSize || digest != source.SHA256 {
				return nil, fmt.Errorf(
					"%w: verified asset %s/%d changed while loading", ErrInvalidFaceCatalog, face.ID, source.Size,
				)
			}

			size := PhotoSize(source.Size)
			path := publicPhotoPath(checksum, size, face.ID)
			metadata := PhotoAsset{
				PhotoID: face.ID, Size: size, Width: source.Width, Height: source.Height,
				ContentSHA256: hex.EncodeToString(source.SHA256[:]), ContentLength: source.ByteSize,
				CatalogVersion: fakefacegenV1SourceVersion, CatalogSHA256: hex.EncodeToString(checksum[:]), Path: path,
			}
			record := photoAssetRecord{metadata: metadata, data: data}
			key := photoAssetKey{photoID: face.ID, size: size}
			catalog.assets[key] = record
			catalog.paths[path] = record
			if faceCatalogLoadStepForTest != nil {
				faceCatalogLoadStepForTest("asset")
			}

		}
	}

	err = verifyFaceCatalogMetadataStable(ctx, absoluteDirectory, snapshot)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, contextErr
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidFaceCatalog, err)
	}

	return catalog, nil
}

// Asset returns normalized public metadata for a verified photo asset.
// It returns ErrInvalidFaceCatalog for a nil catalog or ErrInvalidPhotoAsset
// when photoID or size is unavailable.
func (c *FaceCatalog) Asset(photoID string, size PhotoSize) (PhotoAsset, error) {
	return c.asset(photoID, size)
}

// Checksum returns the raw semantic SHA-256 identity of the verified snapshot.
func (c *FaceCatalog) Checksum() [sha256.Size]byte {
	if c == nil {
		return [sha256.Size]byte{}
	}
	return c.checksum
}

// Handler returns an HTTP handler for verified catalog bytes.
func (c *FaceCatalog) Handler() http.Handler {
	return c.handler()
}

// SourceCatalogChecksum returns the raw SHA-256 of the source catalog.json.
func (c *FaceCatalog) SourceCatalogChecksum() [sha256.Size]byte {
	if c == nil {
		return [sha256.Size]byte{}
	}
	return c.sourceChecksum
}

// Version returns the catalog's stable source version.
func (c *FaceCatalog) Version() string {
	if c == nil {
		return ""
	}
	return c.version
}

func (c *FaceCatalog) asset(photoID string, size PhotoSize) (PhotoAsset, error) {
	if c == nil {
		return PhotoAsset{}, ErrInvalidFaceCatalog
	}
	record, exists := c.assets[photoAssetKey{photoID: photoID, size: size}]
	if !exists {
		return PhotoAsset{}, ErrInvalidPhotoAsset
	}
	return record.metadata, nil
}

func (c *FaceCatalog) handler() http.Handler {
	return photoHTTPHandler{catalog: c}
}

func (c *FaceCatalog) hasUsableGender(gender Gender) bool {
	for _, face := range c.faces {
		if face.status == fakefacegenStatusGenerated && !face.excluded && face.gender == gender {
			return true
		}
	}
	return false
}

func (c *FaceCatalog) selectPhoto(factory streamFactory, index PersonIndex, gender Gender, anchorAge int) (string, error) {

	bestDistance := int(^uint(0) >> 1)
	candidateCount := uint64(0)
	for _, face := range c.faces {
		if face.status != fakefacegenStatusGenerated || face.excluded || face.gender != gender {
			continue
		}
		distance := face.requestedAge - anchorAge
		if distance < 0 {
			distance = -distance
		}
		if distance < bestDistance {
			bestDistance = distance
			candidateCount = 1
			continue
		}
		if distance == bestDistance {
			candidateCount++
		}
	}
	if candidateCount == 0 {
		return "", fmt.Errorf("%w: gender %s", ErrNoCompatiblePhoto, gender)
	}

	rng, err := factory.entityStream(personEntityKind, uint64(index), streamPhotoSelect, c.dependencies[:]...)
	if err != nil {
		return "", err
	}
	choice, err := rng.Uint64n(candidateCount)
	if err != nil {
		return "", err
	}
	for _, face := range c.faces {
		if face.status != fakefacegenStatusGenerated || face.excluded || face.gender != gender {
			continue
		}
		distance := face.requestedAge - anchorAge
		if distance < 0 {
			distance = -distance
		}
		if distance != bestDistance {
			continue
		}
		if choice == 0 {
			return face.id, nil
		}
		choice--
	}

	return "", ErrInvalidFaceCatalog
}

func (c *FaceCatalog) validate() error {
	if c == nil || c.version == "" || c.sourceChecksum == ([sha256.Size]byte{}) ||
		c.checksum == ([sha256.Size]byte{}) || len(c.faces) == 0 {
		return ErrInvalidFaceCatalog
	}
	if c.mode != faceCatalogSnapshotModeComplete && c.mode != faceCatalogSnapshotModeSelectedFixture {
		return ErrInvalidFaceCatalog
	}
	if c.dependencies[0].name != faceCatalogVersionComponentName ||
		c.dependencies[1].name != faceCatalogSHA256ComponentName {
		return ErrInvalidFaceCatalog
	}
	for _, face := range c.faces {
		if face.id == "" || face.gender != GenderFemale && face.gender != GenderMale ||
			face.requestedAge < 21 || face.requestedAge > 75 {
			return fmt.Errorf("%w: photo %s has invalid selection metadata", ErrInvalidFaceCatalog, face.id)
		}
		switch face.status {
		case fakefacegenStatusFailed, fakefacegenStatusMissing:
			if !face.excluded {
				return fmt.Errorf("%w: photo %s status %s is not excluded", ErrInvalidFaceCatalog, face.id, face.status)
			}
		case fakefacegenStatusGenerated:
			if face.excluded {
				return fmt.Errorf("%w: generated photo %s is excluded", ErrInvalidFaceCatalog, face.id)
			}
		default:
			return fmt.Errorf("%w: photo %s has unknown status", ErrInvalidFaceCatalog, face.id)
		}
		if face.status == fakefacegenStatusGenerated {
			for _, size := range supportedPhotoSizes {
				if _, exists := c.assets[photoAssetKey{photoID: face.id, size: size}]; !exists {
					return fmt.Errorf("%w: photo %s has no verified size %d", ErrInvalidFaceCatalog, face.id, size)
				}
			}
		}
	}
	return nil
}

type faceCatalogMetadataSnapshot struct {
	catalog       fakefacegenCatalog
	catalogBytes  []byte
	specs         []fakefacegenSpec
	specsBytes    []byte
	specsByID     map[string]fakefacegenSpec
	manifestBytes []byte
	assetsByID    map[string]fakefacegenManifestAsset
}

type photoAssetKey struct {
	photoID string
	size    PhotoSize
}

type photoAssetRecord struct {
	metadata PhotoAsset
	data     []byte
}

type photoFace struct {
	id           string
	gender       Gender
	requestedAge int
	status       fakefacegenSourceStatus
	excluded     bool
	diagnostic   string
}

type photoHTTPHandler struct {
	catalog *FaceCatalog
}

func (h photoHTTPHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	if h.catalog == nil {
		http.NotFound(writer, request)
		return
	}
	record, exists := h.catalog.paths[request.URL.Path]
	if !exists {
		http.NotFound(writer, request)
		return
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writer.Header().Set("Content-Type", "image/webp")
	writer.Header().Set("Content-Length", strconv.FormatInt(record.metadata.ContentLength, 10))
	writer.WriteHeader(http.StatusOK)
	if request.Method == http.MethodGet {
		_, _ = writer.Write(record.data)
	}

}

func captureFaceCatalogMetadata(ctx context.Context, directory string) (faceCatalogMetadataSnapshot, error) {

	catalogPath := filepath.Join(directory, "catalog.json")
	catalogBytes, err := readBoundedPhotoFile(ctx, catalogPath, fakefacegenMaxMetadataBytes)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, fmt.Errorf("read fakefacegen metadata %q: %w", catalogPath, err)
	}
	if faceCatalogLoadStepForTest != nil {
		faceCatalogLoadStepForTest("catalog.json")
	}
	var catalog fakefacegenCatalog
	err = decodeFaceCatalogJSON(catalogPath, catalogBytes, &catalog)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}
	err = validateFakefacegenCatalog(catalog)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}
	err = validateFaceCatalogRecordCount("catalog", catalog.Count)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}

	specsPath := filepath.Join(directory, "specs.json")
	specsBytes, err := readBoundedPhotoFile(ctx, specsPath, fakefacegenMaxMetadataBytes)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, fmt.Errorf("read fakefacegen metadata %q: %w", specsPath, err)
	}
	if faceCatalogLoadStepForTest != nil {
		faceCatalogLoadStepForTest("specs.json")
	}
	specCount, err := countFaceCatalogSpecs(specsPath, specsBytes)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}

	manifestPath := filepath.Join(directory, "manifest.json")
	manifestBytes, err := readBoundedPhotoFile(ctx, manifestPath, fakefacegenMaxMetadataBytes)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, fmt.Errorf("read fakefacegen metadata %q: %w", manifestPath, err)
	}
	if faceCatalogLoadStepForTest != nil {
		faceCatalogLoadStepForTest("manifest.json")
	}
	manifestCount, err := countFaceCatalogManifestAssets(manifestPath, manifestBytes)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}
	err = validateFaceCatalogRecordCounts(catalog.Count, specCount, manifestCount)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}

	var specs []fakefacegenSpec
	err = decodeFaceCatalogJSON(specsPath, specsBytes, &specs)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}
	var manifest fakefacegenManifest
	err = decodeFaceCatalogJSON(manifestPath, manifestBytes, &manifest)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}
	err = validateFaceCatalogRecordCounts(catalog.Count, len(specs), len(manifest.Assets))
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}

	specsByID, err := indexFakefacegenSpecs(specs)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}
	assetsByID, err := indexFakefacegenManifest(manifest)
	if err != nil {
		return faceCatalogMetadataSnapshot{}, err
	}
	for id := range specsByID {
		if _, exists := assetsByID[id]; !exists {
			return faceCatalogMetadataSnapshot{}, fmt.Errorf("fakefacegen manifest has no asset for spec %q", id)
		}
	}

	return faceCatalogMetadataSnapshot{
		catalog: catalog, catalogBytes: catalogBytes, specs: specs, specsBytes: specsBytes, specsByID: specsByID,
		manifestBytes: manifestBytes, assetsByID: assetsByID,
	}, nil
}

func countFaceCatalogJSONArray(decoder *json.Decoder, subject string) (int, error) {

	token, err := decoder.Token()
	if err != nil {
		return 0, err
	}
	if token == nil {
		return 0, nil
	}
	if token != json.Delim('[') {
		return 0, fmt.Errorf("fakefacegen %s is not an array", subject)
	}

	count := 0
	for decoder.More() {
		if count == maxFaceCatalogRecords {
			return 0, fmt.Errorf("fakefacegen %s contains more than %d records", subject, maxFaceCatalogRecords)
		}
		var record json.RawMessage
		err = decoder.Decode(&record)
		if err != nil {
			return 0, err
		}
		count++
	}
	_, err = decoder.Token()
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countFaceCatalogManifestAssets(path string, data []byte) (int, error) {

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	token, err := decoder.Token()
	if err != nil {
		return 0, fmt.Errorf("decode fakefacegen metadata %q: %w", path, err)
	}
	if token == nil {
		err = finishFaceCatalogJSON(decoder, path)
		return 0, err
	}
	if token != json.Delim('{') {
		return 0, fmt.Errorf("decode fakefacegen metadata %q: manifest is not an object", path)
	}

	count := 0
	for decoder.More() {
		fieldToken, err := decoder.Token()
		if err != nil {
			return 0, fmt.Errorf("decode fakefacegen metadata %q: %w", path, err)
		}
		field, ok := fieldToken.(string)
		if !ok || field != "assets" {
			return 0, fmt.Errorf("decode fakefacegen metadata %q: unknown field %q", path, field)
		}
		count, err = countFaceCatalogJSONArray(decoder, "manifest")
		if err != nil {
			return 0, fmt.Errorf("decode fakefacegen metadata %q: %w", path, err)
		}
	}
	_, err = decoder.Token()
	if err != nil {
		return 0, fmt.Errorf("decode fakefacegen metadata %q: %w", path, err)
	}
	err = finishFaceCatalogJSON(decoder, path)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func countFaceCatalogSpecs(path string, data []byte) (int, error) {

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	count, err := countFaceCatalogJSONArray(decoder, "specs")
	if err != nil {
		return 0, fmt.Errorf("decode fakefacegen metadata %q: %w", path, err)
	}
	err = finishFaceCatalogJSON(decoder, path)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func decodeFaceCatalogJSON(path string, data []byte, value any) error {

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(value)
	if err != nil {
		return fmt.Errorf("decode fakefacegen metadata %q: %w", path, err)
	}

	return finishFaceCatalogJSON(decoder, path)
}

func finishFaceCatalogJSON(decoder *json.Decoder, path string) error {

	_, err := decoder.Token()
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("decode fakefacegen metadata %q: multiple JSON values", path)
	}

	return fmt.Errorf("decode fakefacegen metadata %q: %w", path, err)
}

func loadFaceCatalogFaces(ctx context.Context, directory string, snapshot faceCatalogMetadataSnapshot, selectedIDs []string, mode string) ([]fakefacegenSourceFace, error) {

	if mode == faceCatalogSnapshotModeComplete {
		faces := make([]fakefacegenSourceFace, len(snapshot.specs))
		for index, spec := range snapshot.specs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			face, err := mapFakefacegenFace(directory, snapshot.catalog, spec, snapshot.assetsByID[spec.ID])
			if err != nil {
				return nil, err
			}
			faces[index] = face
		}
		return faces, nil
	}

	faces := make([]fakefacegenSourceFace, len(selectedIDs))
	selected := make(map[string]struct{}, len(selectedIDs))
	for index, id := range selectedIDs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, exists := selected[id]; exists {
			return nil, fmt.Errorf("fakefacegen selected ID %q appears more than once", id)
		}
		spec, exists := snapshot.specsByID[id]
		if !exists {
			return nil, fmt.Errorf("fakefacegen selected ID %q has no spec", id)
		}
		face, err := mapFakefacegenFace(directory, snapshot.catalog, spec, snapshot.assetsByID[id])
		if err != nil {
			return nil, err
		}
		selected[id] = struct{}{}
		faces[index] = face
	}

	return faces, nil
}

func publicPhotoPath(checksum [sha256.Size]byte, size PhotoSize, photoID string) string {
	return "/photos/" + hex.EncodeToString(checksum[:]) + "/" + strconv.Itoa(int(size)) + "/" + photoID + ".webp"
}

func readBoundedPhotoFile(ctx context.Context, path string, maximum int) ([]byte, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if err = ctx.Err(); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if len(data) > maximum {
		return nil, fmt.Errorf("file exceeds %d bytes", maximum)
	}

	return data, nil
}

func semanticFaceCatalogChecksum(version string, sourceChecksum [sha256.Size]byte, mode string, faces []fakefacegenSourceFace) ([sha256.Size]byte, error) {

	if uint64(len(faces)) > math.MaxUint32 {
		return [sha256.Size]byte{}, ErrComponentCountOverflow
	}
	if len(faces) > (math.MaxInt-4)/(4+sha256.Size) {
		return [sha256.Size]byte{}, fmt.Errorf("canonical entry sequence is too large")
	}
	ordered := append([]fakefacegenSourceFace(nil), faces...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].ID < ordered[j].ID
	})
	entries := make([]byte, 0, 4+len(faces)*(4+sha256.Size))
	entries = appendUint32(entries, uint32(len(faces)))
	for _, face := range ordered {
		if uint64(len(face.ID)) > math.MaxUint32 {
			return [sha256.Size]byte{}, fmt.Errorf("photo ID is too long")
		}
		digest, err := semanticFaceCatalogEntryDigest(face)
		if err != nil {
			return [sha256.Size]byte{}, err
		}
		entries = appendUint32(entries, uint32(len(face.ID)))
		entries = append(entries, face.ID...)
		entries = append(entries, digest[:]...)
	}

	catalogVersion, err := StringComponent("catalog-version", version)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	sourceCatalogSHA256, err := BytesComponent("source-catalog-sha256", sourceChecksum[:])
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	snapshotMode, err := StringComponent("snapshot-mode", mode)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	canonicalEntries, err := BytesComponent("entries", entries)
	if err != nil {
		return [sha256.Size]byte{}, err
	}

	return canonicalDigest(
		faceCatalogSnapshotDomain,
		[]Component{catalogVersion, sourceCatalogSHA256, snapshotMode, canonicalEntries},
	)
}

func semanticFaceCatalogEntryDigest(face fakefacegenSourceFace) ([sha256.Size]byte, error) {

	if face.ID == "" || face.Gender != fakefacegenGenderFemale && face.Gender != fakefacegenGenderMale ||
		face.RequestedAge < 21 || face.RequestedAge > 75 {
		return [sha256.Size]byte{}, fmt.Errorf("photo %s has invalid selection metadata", face.ID)
	}
	switch face.Status {
	case fakefacegenStatusFailed, fakefacegenStatusGenerated, fakefacegenStatusMissing:
	default:
		return [sha256.Size]byte{}, fmt.Errorf("photo %s has unknown status %q", face.ID, face.Status)
	}

	photoID, err := StringComponent("photo-id", face.ID)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	status, err := StringComponent("status", string(face.Status))
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	gender, err := StringComponent("gender", string(Gender(face.Gender)))
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	requestedAge, err := Uint64Component("requested-age", uint64(face.RequestedAge))
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	components := []Component{photoID, status, gender, requestedAge}
	if face.Status == fakefacegenStatusGenerated {
		for _, size := range supportedPhotoSizes {
			var checksum [sha256.Size]byte
			found := false
			for _, asset := range face.Assets {
				if PhotoSize(asset.Size) != size {
					continue
				}
				if found {
					return [sha256.Size]byte{}, fmt.Errorf("photo %s has duplicate size %d", face.ID, size)
				}
				checksum = asset.SHA256
				found = true
			}
			if !found {
				return [sha256.Size]byte{}, fmt.Errorf("photo %s has no size %d", face.ID, size)
			}
			component, err := BytesComponent("asset-"+strconv.Itoa(int(size))+"-sha256", checksum[:])
			if err != nil {
				return [sha256.Size]byte{}, err
			}
			components = append(components, component)
		}
	}

	return canonicalDigest(faceCatalogEntryDomain, components)
}

func validateFaceCatalogRecordCount(subject string, count int) error {
	if count < 0 {
		return fmt.Errorf("fakefacegen %s record count %d is invalid", subject, count)
	}
	if count > maxFaceCatalogRecords {
		return fmt.Errorf("fakefacegen %s record count %d exceeds limit %d", subject, count, maxFaceCatalogRecords)
	}
	return nil
}

func validateFaceCatalogRecordCounts(declared, specs, manifest int) error {

	err := validateFaceCatalogRecordCount("catalog", declared)
	if err != nil {
		return err
	}
	err = validateFaceCatalogRecordCount("specs", specs)
	if err != nil {
		return err
	}
	err = validateFaceCatalogRecordCount("manifest", manifest)
	if err != nil {
		return err
	}
	if specs != declared {
		return fmt.Errorf("fakefacegen catalog count is %d but specs contains %d records", declared, specs)
	}
	if manifest != declared {
		return fmt.Errorf("fakefacegen catalog count is %d but manifest contains %d assets", declared, manifest)
	}

	return nil
}

func verifyFaceCatalogMetadataStable(ctx context.Context, directory string, captured faceCatalogMetadataSnapshot) error {

	for _, metadata := range []struct {
		name     string
		captured []byte
	}{
		{name: "catalog.json", captured: captured.catalogBytes},
		{name: "specs.json", captured: captured.specsBytes},
		{name: "manifest.json", captured: captured.manifestBytes},
	} {
		path := filepath.Join(directory, metadata.name)
		current, err := readBoundedPhotoFile(ctx, path, fakefacegenMaxMetadataBytes)
		if err != nil {
			return fmt.Errorf("face catalog metadata changed during load: %s: %w", metadata.name, err)
		}
		if !bytes.Equal(current, metadata.captured) {
			return fmt.Errorf("face catalog metadata changed during load: %s", metadata.name)
		}
	}

	return nil
}
