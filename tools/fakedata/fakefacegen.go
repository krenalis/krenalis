// Package fakedata provides deterministic synthetic data for tests and demos.
package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/webp"
)

const (
	fakefacegenCatalogVersion   = 1
	fakefacegenMaxImageBytes    = 32 << 20
	fakefacegenMaxMetadataBytes = 128 << 20
	fakefacegenPromptVersion    = "portrait-v1"
	fakefacegenSpecVersion      = "spec-v1"
	fakefacegenV1SourceVersion  = "fakefacegen-v1"
	fakefacegenGenderFemale     = fakefacegenGender("female")
	fakefacegenGenderMale       = fakefacegenGender("male")
	fakefacegenStatusFailed     = fakefacegenSourceStatus("failed")
	fakefacegenStatusGenerated  = fakefacegenSourceStatus("generated")
	fakefacegenStatusMissing    = fakefacegenSourceStatus("missing")
	fakefacegenPhotoSize64      = fakefacegenPhotoSize(64)
	fakefacegenPhotoSize128     = fakefacegenPhotoSize(128)
	fakefacegenPhotoSize256     = fakefacegenPhotoSize(256)
	fakefacegenPhotoSize512     = fakefacegenPhotoSize(512)
	fakefacegenPhotoSize1024    = fakefacegenPhotoSize(1024)
)

type fakefacegenGender string

type fakefacegenPhotoSize uint16

type fakefacegenSourceStatus string

type fakefacegenSourceAsset struct {
	PhotoID  string
	Size     fakefacegenPhotoSize
	MIMEType string
	SHA256   [32]byte
	ByteSize int64
	Width    uint32
	Height   uint32
	path     string
}

type fakefacegenSourceFace struct {
	SourceVersion string
	ID            string
	Gender        fakefacegenGender
	RequestedAge  int
	Status        fakefacegenSourceStatus
	Excluded      bool
	Diagnostic    string
	Assets        [5]fakefacegenSourceAsset
}

type fakefacegenCatalog struct {
	CatalogVersion int    `json:"catalog_version"`
	Seed           uint64 `json:"seed"`
	Count          int    `json:"count"`
	SpecVersion    string `json:"spec_version"`
	PromptVersion  string `json:"prompt_version"`
	Model          string `json:"model"`
	Quality        string `json:"quality"`
	Size           string `json:"size"`
	Format         string `json:"format"`
	Background     string `json:"background"`
}

type fakefacegenSpec struct {
	ID               string `json:"id"`
	Age              int    `json:"age"`
	Sex              string `json:"sex"`
	SkinTone         string `json:"skin_tone"`
	FaceShape        string `json:"face_shape"`
	HairColor        string `json:"hair_color"`
	HairLength       string `json:"hair_length"`
	HairTexture      string `json:"hair_texture"`
	RecedingHairline bool   `json:"receding_hairline"`
	FacialHair       string `json:"facial_hair,omitempty"`
	Glasses          bool   `json:"glasses"`
	NaturalDetail    string `json:"natural_detail,omitempty"`
	Clothing         string `json:"clothing"`
	Environment      string `json:"environment"`
	Expression       string `json:"expression"`
	Lighting         string `json:"lighting"`
	PhotoStyle       string `json:"photo_style"`
}

type fakefacegenManifest struct {
	Assets []fakefacegenManifestAsset `json:"assets"`
}

type fakefacegenManifestAsset struct {
	ID            string                    `json:"id"`
	File          string                    `json:"file,omitempty"`
	SpecID        string                    `json:"spec_id"`
	SpecVersion   string                    `json:"spec_version"`
	PromptVersion string                    `json:"prompt_version"`
	Model         string                    `json:"model"`
	Quality       string                    `json:"quality"`
	Size          string                    `json:"size"`
	Format        string                    `json:"format"`
	Synthetic     bool                      `json:"synthetic"`
	Status        fakefacegenSourceStatus   `json:"status"`
	SHA256        string                    `json:"sha256,omitempty"`
	Bytes         int64                     `json:"bytes,omitempty"`
	Width         int                       `json:"width,omitempty"`
	Height        int                       `json:"height,omitempty"`
	RequestID     string                    `json:"request_id,omitempty"`
	Error         *fakefacegenManifestError `json:"error,omitempty"`
}

type fakefacegenManifestError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// fakefacegenDiagnostic returns source diagnostic text for an exclusion.
func fakefacegenDiagnostic(asset fakefacegenManifestAsset) string {

	if asset.Error == nil {
		return fmt.Sprintf("fakefacegen source status %q", asset.Status)
	}
	if asset.Error.Code == "" {
		return asset.Error.Message
	}

	return asset.Error.Code + ": " + asset.Error.Message
}

// indexFakefacegenManifest validates statuses and indexes assets by source ID.
func indexFakefacegenManifest(manifest fakefacegenManifest) (map[string]fakefacegenManifestAsset, error) {

	assets := make(map[string]fakefacegenManifestAsset, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		if asset.ID == "" {
			return nil, errors.New("fakefacegen manifest contains an empty asset ID")
		}
		if _, exists := assets[asset.ID]; exists {
			return nil, fmt.Errorf("fakefacegen manifest contains duplicate asset ID %q", asset.ID)
		}
		switch asset.Status {
		case fakefacegenStatusFailed, fakefacegenStatusGenerated, fakefacegenStatusMissing:
		default:
			return nil, fmt.Errorf("fakefacegen asset %q has unsupported status %q", asset.ID, asset.Status)
		}
		assets[asset.ID] = asset
	}

	return assets, nil
}

// indexFakefacegenSpecs validates source ID order and indexes specs by ID.
func indexFakefacegenSpecs(specs []fakefacegenSpec) (map[string]fakefacegenSpec, error) {

	byID := make(map[string]fakefacegenSpec, len(specs))
	for position, current := range specs {
		expectedID := fmt.Sprintf("face-%06d", position+1)
		if current.ID != expectedID {
			return nil, fmt.Errorf("fakefacegen spec at position %d has ID %q, expected %q", position+1, current.ID, expectedID)
		}
		if _, exists := byID[current.ID]; exists {
			return nil, fmt.Errorf("fakefacegen specs contain duplicate ID %q", current.ID)
		}
		byID[current.ID] = current
	}

	return byID, nil
}

// inspectFakefacegenAsset validates and describes one selected WebP asset.
func inspectFakefacegenAsset(root, photoID, relativePath string, size fakefacegenPhotoSize) (fakefacegenSourceAsset, error) {

	path := filepath.Join(root, filepath.FromSlash(relativePath))
	file, err := os.Open(path)
	if err != nil {
		return fakefacegenSourceAsset{}, fmt.Errorf("open fakefacegen asset %q: %w", relativePath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, fakefacegenMaxImageBytes+1))
	if err != nil {
		return fakefacegenSourceAsset{}, fmt.Errorf("read fakefacegen asset %q: %w", relativePath, err)
	}
	if len(data) == 0 {
		return fakefacegenSourceAsset{}, fmt.Errorf("fakefacegen asset %q is empty", relativePath)
	}
	if len(data) > fakefacegenMaxImageBytes {
		return fakefacegenSourceAsset{}, fmt.Errorf(
			"fakefacegen asset %q exceeds %d bytes", relativePath, fakefacegenMaxImageBytes,
		)
	}
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return fakefacegenSourceAsset{}, fmt.Errorf("fakefacegen asset %q is not image/webp", relativePath)
	}

	config, err := webp.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fakefacegenSourceAsset{}, fmt.Errorf("decode fakefacegen WebP configuration %q: %w", relativePath, err)
	}
	expectedSize := int(size)
	if config.Width != expectedSize || config.Height != expectedSize {
		return fakefacegenSourceAsset{}, fmt.Errorf(
			"fakefacegen asset %q has dimensions %dx%d, expected %dx%d",
			relativePath, config.Width, config.Height, expectedSize, expectedSize,
		)
	}
	decoded, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		return fakefacegenSourceAsset{}, fmt.Errorf("decode fakefacegen WebP %q: %w", relativePath, err)
	}
	if decoded.Bounds().Dx() != expectedSize || decoded.Bounds().Dy() != expectedSize {
		return fakefacegenSourceAsset{}, fmt.Errorf(
			"decoded fakefacegen asset %q has dimensions %dx%d, expected %dx%d",
			relativePath, decoded.Bounds().Dx(), decoded.Bounds().Dy(), expectedSize, expectedSize,
		)
	}
	digest := sha256.Sum256(data)

	return fakefacegenSourceAsset{
		PhotoID:  photoID,
		Size:     size,
		MIMEType: "image/webp",
		SHA256:   digest,
		ByteSize: int64(len(data)),
		Width:    uint32(config.Width),
		Height:   uint32(config.Height),
		path:     relativePath,
	}, nil
}

// loadFakefacegenV1Selected is limited to explicitly selected records because
// its authentic test fixture intentionally retains global metadata while
// including image assets only for those records. A future production loader
// must validate a complete snapshot instead of exposing partial selection.
func loadFakefacegenV1Selected(directory string, selectedIDs []string) ([]fakefacegenSourceFace, error) {

	if len(selectedIDs) == 0 {
		return nil, errors.New("fakefacegen selected IDs are empty")
	}

	var metadata fakefacegenCatalog
	err := readFakefacegenJSON(filepath.Join(directory, "catalog.json"), &metadata)
	if err != nil {
		return nil, err
	}
	err = validateFakefacegenCatalog(metadata)
	if err != nil {
		return nil, err
	}

	var specs []fakefacegenSpec
	err = readFakefacegenJSON(filepath.Join(directory, "specs.json"), &specs)
	if err != nil {
		return nil, err
	}
	if len(specs) != metadata.Count {
		return nil, fmt.Errorf("fakefacegen catalog count is %d but specs contains %d records", metadata.Count, len(specs))
	}
	byID, err := indexFakefacegenSpecs(specs)
	if err != nil {
		return nil, err
	}

	var manifest fakefacegenManifest
	err = readFakefacegenJSON(filepath.Join(directory, "manifest.json"), &manifest)
	if err != nil {
		return nil, err
	}
	if len(manifest.Assets) != metadata.Count {
		return nil, fmt.Errorf(
			"fakefacegen catalog count is %d but manifest contains %d assets", metadata.Count, len(manifest.Assets),
		)
	}
	assetsByID, err := indexFakefacegenManifest(manifest)
	if err != nil {
		return nil, err
	}
	for id := range byID {
		if _, exists := assetsByID[id]; !exists {
			return nil, fmt.Errorf("fakefacegen manifest has no asset for spec %q", id)
		}
	}

	faces := make([]fakefacegenSourceFace, len(selectedIDs))
	selected := map[string]struct{}{}
	for index, id := range selectedIDs {
		if _, exists := selected[id]; exists {
			return nil, fmt.Errorf("fakefacegen selected ID %q appears more than once", id)
		}
		currentSpec, exists := byID[id]
		if !exists {
			return nil, fmt.Errorf("fakefacegen selected ID %q has no spec", id)
		}
		asset := assetsByID[id]
		face, err := mapFakefacegenFace(directory, metadata, currentSpec, asset)
		if err != nil {
			return nil, err
		}
		selected[id] = struct{}{}
		faces[index] = face
	}

	return faces, nil
}

// mapFakefacegenFace maps one selected source spec and manifest record.
func mapFakefacegenFace(directory string, metadata fakefacegenCatalog, currentSpec fakefacegenSpec, manifestAsset fakefacegenManifestAsset) (fakefacegenSourceFace, error) {

	err := validateFakefacegenSpec(currentSpec)
	if err != nil {
		return fakefacegenSourceFace{}, err
	}
	err = validateFakefacegenManifestAsset(metadata, currentSpec, manifestAsset)
	if err != nil {
		return fakefacegenSourceFace{}, err
	}

	face := fakefacegenSourceFace{
		SourceVersion: fakefacegenV1SourceVersion,
		ID:            currentSpec.ID,
		Gender:        fakefacegenGender(currentSpec.Sex),
		RequestedAge:  currentSpec.Age,
		Status:        manifestAsset.Status,
	}
	if manifestAsset.Status != fakefacegenStatusGenerated {
		face.Excluded = true
		face.Diagnostic = fakefacegenDiagnostic(manifestAsset)
		return face, nil
	}

	variants := [...]struct {
		size fakefacegenPhotoSize
		path string
	}{
		{size: fakefacegenPhotoSize64, path: filepath.ToSlash(filepath.Join("derived", "64", currentSpec.ID+".webp"))},
		{size: fakefacegenPhotoSize128, path: filepath.ToSlash(filepath.Join("derived", "128", currentSpec.ID+".webp"))},
		{size: fakefacegenPhotoSize256, path: filepath.ToSlash(filepath.Join("derived", "256", currentSpec.ID+".webp"))},
		{size: fakefacegenPhotoSize512, path: filepath.ToSlash(filepath.Join("derived", "512", currentSpec.ID+".webp"))},
		{size: fakefacegenPhotoSize1024, path: manifestAsset.File},
	}
	for index, variant := range variants {
		current, err := inspectFakefacegenAsset(directory, currentSpec.ID, variant.path, variant.size)
		if err != nil {
			return fakefacegenSourceFace{}, err
		}
		face.Assets[index] = current
	}

	master := face.Assets[len(face.Assets)-1]
	if hex.EncodeToString(master.SHA256[:]) != manifestAsset.SHA256 {
		return fakefacegenSourceFace{}, fmt.Errorf("fakefacegen master %q SHA-256 does not match manifest", manifestAsset.ID)
	}
	if master.ByteSize != manifestAsset.Bytes {
		return fakefacegenSourceFace{}, fmt.Errorf(
			"fakefacegen master %q has %d bytes, expected %d from manifest",
			manifestAsset.ID, master.ByteSize, manifestAsset.Bytes,
		)
	}

	return face, nil
}

// readFakefacegenJSON reads exactly one bounded source JSON value.
func readFakefacegenJSON(path string, value any) error {

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open fakefacegen metadata %q: %w", path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, fakefacegenMaxMetadataBytes+1))
	if err != nil {
		return fmt.Errorf("read fakefacegen metadata %q: %w", path, err)
	}
	if len(data) > fakefacegenMaxMetadataBytes {
		return fmt.Errorf("fakefacegen metadata %q exceeds %d bytes", path, fakefacegenMaxMetadataBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(value)
	if err != nil {
		return fmt.Errorf("decode fakefacegen metadata %q: %w", path, err)
	}
	var extra any
	err = decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return fmt.Errorf("decode fakefacegen metadata %q: multiple JSON values", path)
	}

	return fmt.Errorf("decode fakefacegen metadata %q: %w", path, err)
}

// validateFakefacegenCatalog enforces the only supported source version.
func validateFakefacegenCatalog(metadata fakefacegenCatalog) error {

	if metadata.CatalogVersion != fakefacegenCatalogVersion {
		return fmt.Errorf("unsupported fakefacegen catalog version %d", metadata.CatalogVersion)
	}
	if metadata.Count < 1 {
		return errors.New("fakefacegen catalog count must be at least 1")
	}
	if metadata.SpecVersion != fakefacegenSpecVersion {
		return fmt.Errorf("unsupported fakefacegen spec version %q", metadata.SpecVersion)
	}
	if metadata.PromptVersion != fakefacegenPromptVersion {
		return fmt.Errorf("unsupported fakefacegen prompt version %q", metadata.PromptVersion)
	}
	if metadata.Model != "gpt-image-2" || metadata.Quality != "low" || metadata.Size != "1024x1024" ||
		metadata.Format != "webp" || metadata.Background != "opaque" {
		return errors.New("unsupported fakefacegen catalog image configuration")
	}

	return nil
}

// validateFakefacegenManifestAsset verifies a spec-to-master mapping.
func validateFakefacegenManifestAsset(metadata fakefacegenCatalog, currentSpec fakefacegenSpec, asset fakefacegenManifestAsset) error {

	if asset.ID != currentSpec.ID || asset.SpecID != currentSpec.ID {
		return fmt.Errorf(
			"fakefacegen spec %q does not match manifest asset ID %q and spec ID %q",
			currentSpec.ID, asset.ID, asset.SpecID,
		)
	}
	if asset.SpecVersion != metadata.SpecVersion || asset.PromptVersion != metadata.PromptVersion ||
		asset.Model != metadata.Model || asset.Quality != metadata.Quality || asset.Size != metadata.Size ||
		asset.Format != metadata.Format {
		return fmt.Errorf("fakefacegen manifest asset %q does not match catalog configuration", asset.ID)
	}
	if !asset.Synthetic {
		return fmt.Errorf("fakefacegen manifest asset %q is not marked synthetic", asset.ID)
	}
	if asset.Status != fakefacegenStatusGenerated {
		return nil
	}

	expectedFile := filepath.ToSlash(filepath.Join("masters", currentSpec.ID+".webp"))
	if asset.File != expectedFile {
		return fmt.Errorf("fakefacegen generated asset %q has file %q, expected %q", asset.ID, asset.File, expectedFile)
	}
	if len(asset.SHA256) != sha256.Size*2 || asset.SHA256 != strings.ToLower(asset.SHA256) {
		return fmt.Errorf("fakefacegen generated asset %q has invalid SHA-256 %q", asset.ID, asset.SHA256)
	}
	_, err := hex.DecodeString(asset.SHA256)
	if err != nil {
		return fmt.Errorf("fakefacegen generated asset %q has invalid SHA-256 %q", asset.ID, asset.SHA256)
	}
	if asset.Bytes < 1 {
		return fmt.Errorf("fakefacegen generated asset %q has invalid byte size %d", asset.ID, asset.Bytes)
	}
	if asset.Width != int(fakefacegenPhotoSize1024) || asset.Height != int(fakefacegenPhotoSize1024) {
		return fmt.Errorf(
			"fakefacegen generated asset %q has dimensions %dx%d, expected 1024x1024",
			asset.ID, asset.Width, asset.Height,
		)
	}

	return nil
}

// validateFakefacegenSpec verifies the normalized age and gender inputs.
func validateFakefacegenSpec(current fakefacegenSpec) error {

	if current.Age < 21 || current.Age > 75 {
		return fmt.Errorf("fakefacegen spec %q has age %d outside 21..75", current.ID, current.Age)
	}
	if current.Sex != string(fakefacegenGenderFemale) && current.Sex != string(fakefacegenGenderMale) {
		return fmt.Errorf("fakefacegen spec %q has unsupported sex %q", current.ID, current.Sex)
	}

	return nil
}
