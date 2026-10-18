// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/krenalis/krenalis/tools/externalassets"

	"golang.org/x/image/webp"
)

// USASourceVersion identifies observations of the internal USA demo.
const USASourceVersion = usaSourceUSDVersion

var usaDemoPhotoPins = []usaCountrySourcePin{
	{"tools/fakefacegen/output/faces-v1/catalog.json", 241,
		"0dd88add15c6e6d31f1d473421c423ff0608bf123f5ac0c6e8a731d10a8ee914"},
	{"tools/fakefacegen/output/faces-v1/specs.json", 700712,
		"c777624adde88643d186c667f5c863d2ccee819c88e13554d58d76b7ed029876"},
	{"tools/fakefacegen/output/faces-v1/manifest.json", 693438,
		"6d2eb7e4d28c4ec83b617a6f9941a29f694de5bffa33e7d6b4b6433e00413a81"},
	{"tools/fakefacegen/output/faces-v1/batch.json", 26947,
		"1c5ee40407d09af6a90cfa5f1fa5c12abfae4348d554b4691e2adb2c1b948840"},
	{"tools/fakefacegen/output/faces-v1/requests.jsonl", 1645737,
		"2fe06a96a505eebb51fc8b53a92b8ba05bfc82d1dc787083588c67d9038fdc95"},
	{"synthetic-world-staging-ii-20261013/photos.json", 1256737,
		"b81061ea6754ecb68a8e5b5fcc25ef10321d738500557dc98ae7b99c14fa0db4"},
}

// USAWorldData owns the approved USD model and verified photo catalog.
type USAWorldData struct {
	country  *usaCountryData
	catalog  *FaceCatalog
	root     string
	digest   [32]byte
	packaged bool
	resolver *externalassets.Resolver
}

// LoadUSAWorldData verifies portable or repository assets and referenced images.
// It performs no generation, conversion, publication or economic authoring.
// Cancellation returns ctx.Err().
func LoadUSAWorldData(ctx context.Context, root string) (*USAWorldData, error) {

	if ctx == nil || root == "" {
		return nil, fmt.Errorf("USA demo context and repository directory are required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(filepath.Join(root, usaReleasePath))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if err == nil {
		return loadUSAReleaseData(ctx, root)
	}
	assets, err := loadUSACountryAssets(ctx, root)
	if err != nil {
		return nil, err
	}
	model, err := readBoundedPhotoFile(ctx, filepath.Join(root,
		"synthetic-world-usa-m1-internal-country-diagnostic-v2-CANDIDATE/assets/socioeconomic.json"),
		usaSourceUSDModelBytes)
	if err != nil {
		return nil, err
	}
	if len(model) != usaSourceUSDModelBytes || fmt.Sprintf("%x", sha256.Sum256(model)) != usaSourceUSDModelSHA256 {
		return nil, fmt.Errorf("altered USA demo model")
	}
	country, err := prepareUSACountryData(ctx, assets, model)
	if err != nil {
		return nil, err
	}

	// Pin the original batch, requests and catalogs before decoding them.
	photoFiles := map[string][]byte{}
	for _, pin := range usaDemoPhotoPins {
		encoded, err := readBoundedPhotoFile(ctx, filepath.Join(root, pin.Path), pin.Bytes)
		if err != nil {
			return nil, err
		}
		if len(encoded) != pin.Bytes || fmt.Sprintf("%x", sha256.Sum256(encoded)) != pin.SHA256 {
			return nil, fmt.Errorf("altered USA demo photo evidence: %s", pin.Path)
		}
		photoFiles[pin.Path] = encoded
	}
	var batch struct {
		Current struct {
			ID         string   `json:"batch_id"`
			RequestIDs []string `json:"request_ids"`
		} `json:"current"`
	}
	err = json.Unmarshal(photoFiles[usaDemoPhotoPins[3].Path], &batch)
	if err != nil {
		return nil, err
	}
	if batch.Current.ID != "batch_6a99e1bbe12c8190be4982172f3f9f4a" || len(batch.Current.RequestIDs) != 1250 {
		return nil, fmt.Errorf("incompatible USA demo photo batch")
	}
	requestIDs := map[string]bool{}
	for _, id := range batch.Current.RequestIDs {
		if requestIDs[id] {
			return nil, fmt.Errorf("duplicate USA demo photo request")
		}
		requestIDs[id] = true
	}
	requests := bytes.Split(bytes.TrimSpace(photoFiles[usaDemoPhotoPins[4].Path]), []byte{'\n'})
	if len(requests) != 1250 {
		return nil, fmt.Errorf("incomplete USA demo photo requests")
	}
	for _, encoded := range requests {
		var request struct {
			ID string `json:"custom_id"`
		}
		err = json.Unmarshal(encoded, &request)
		if err != nil {
			return nil, err
		}
		if !requestIDs[request.ID] {
			return nil, fmt.Errorf("unmatched USA demo photo request")
		}
		requestIDs[request.ID] = false
	}

	photos, err := decodePhotoCatalogData(ctx, bytes.NewReader(photoFiles[usaDemoPhotoPins[5].Path]))
	if err != nil {
		return nil, err
	}
	corpus := filepath.Join(root, "tools/fakefacegen/output/faces-v1")
	metadata, err := captureFaceCatalogMetadata(ctx, corpus)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(metadata.catalogBytes, photoFiles[usaDemoPhotoPins[0].Path]) ||
		!bytes.Equal(metadata.specsBytes, photoFiles[usaDemoPhotoPins[1].Path]) ||
		!bytes.Equal(metadata.manifestBytes, photoFiles[usaDemoPhotoPins[2].Path]) || len(photos.document.Faces) != 1250 {
		return nil, fmt.Errorf("USA demo photo metadata changed during loading")
	}
	for _, face := range photos.document.Faces {

		spec, ok := metadata.specsByID[face.ID]
		asset, exists := metadata.assetsByID[face.ID]
		_, requested := requestIDs[face.ID]
		if !ok || !exists || !requested || spec.Sex != face.Gender || uint32(spec.Age) != face.RequestedAge ||
			len(face.Renditions) != len(supportedPhotoSizes) {
			return nil, fmt.Errorf("unmatched USA demo photo catalog entry")
		}
		for _, rendition := range face.Renditions {

			size := rendition.Width
			if size != rendition.Height || rendition.Path !=
				fmt.Sprintf("synthetic-world/photos/v1/%d/%s.webp", size, face.ID) || rendition.Integrity == nil ||
				rendition.Integrity.Bytes == 0 || rendition.Integrity.Bytes > 8<<20 {
				return nil, fmt.Errorf("incompatible USA demo photo rendition")
			}
			path := fmt.Sprintf("derived/%d/%s.webp", size, face.ID)
			if size == 1024 {
				path = "masters/" + face.ID + ".webp"
				if asset.File != path || asset.SHA256 != rendition.Integrity.SHA256 ||
					asset.Bytes != int64(rendition.Integrity.Bytes) {
					return nil, fmt.Errorf("USA demo master differs from batch manifest")
				}
			}
			encoded, err := readBoundedPhotoFile(ctx, filepath.Join(corpus, path), int(rendition.Integrity.Bytes))
			if err != nil {
				return nil, err
			}
			if uint64(len(encoded)) != rendition.Integrity.Bytes ||
				fmt.Sprintf("%x", sha256.Sum256(encoded)) != rendition.Integrity.SHA256 {
				return nil, fmt.Errorf("altered USA demo photo asset")
			}
			config, err := webp.DecodeConfig(bytes.NewReader(encoded))
			if err != nil {
				return nil, err
			}
			if config.Width != int(size) || config.Height != int(size) {
				return nil, fmt.Errorf("incompatible USA demo photo dimensions")
			}

		}

	}
	catalog, err := prepareItalianPhotoCatalog(photos,
		internationalTaxonomyFor("sw-l1-gender-v1", []Gender{GenderFemale, GenderMale}))
	if err != nil {
		return nil, err
	}
	digest, err := internationalDigest("fakedata/usa-internal-demo/v1", struct {
		Base   [32]byte
		Phones [32]byte
		Model  [32]byte
		Photos [32]byte
	}{country.base.digest, country.phones.identity, country.socioeconomic.model.digest, photos.digest})
	if err != nil {
		return nil, err
	}

	return &USAWorldData{country: country, catalog: catalog, root: root, digest: digest}, nil
}

// BaseRelease returns the verified USA base release used by the World.
func (d *USAWorldData) BaseRelease() (string, error) {
	if d == nil || d.country == nil {
		return "", fmt.Errorf("initialized USA demo data is required")
	}
	return d.country.baseDocument.Release, nil
}

// Digest identifies the exact base, phone, USD model and photo documents.
func (d *USAWorldData) Digest() ([32]byte, error) {
	if d == nil || d.country == nil || d.catalog == nil {
		return [32]byte{}, fmt.Errorf("initialized USA demo data is required")
	}
	return d.digest, nil
}

// Photo reads an existing 256 rendition and rechecks its declared integrity.
// It returns ErrInvalidPhotoAsset for an unknown photo identifier.
// Cancellation returns ctx.Err().
func (d *USAWorldData) Photo(ctx context.Context, id string) (PhotoAsset, []byte, error) {

	if d == nil || d.catalog == nil || ctx == nil {
		return PhotoAsset{}, nil, fmt.Errorf("initialized USA demo data and context are required")
	}
	asset, err := d.catalog.asset(id, PhotoSize256)
	if err != nil {
		return PhotoAsset{}, nil, err
	}
	if d.resolver != nil {

		var encoded []byte
		err = d.resolver.Resolve(ctx, asset.Path, asset.ContentLength, func(reader io.Reader) error {

			data, err := io.ReadAll(reader)
			if err != nil {
				return err
			}
			if int64(len(data)) != asset.ContentLength || fmt.Sprintf("%x", sha256.Sum256(data)) != asset.ContentSHA256 {
				return fmt.Errorf("altered USA remote photo asset")
			}
			config, err := webp.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				return err
			}
			if config.Width != int(asset.Width) || config.Height != int(asset.Height) {
				return fmt.Errorf("incompatible USA remote photo dimensions")
			}
			encoded = data

			return nil
		})
		if err != nil {
			if contextErr := ctx.Err(); contextErr != nil {
				return PhotoAsset{}, nil, contextErr
			}
			return PhotoAsset{}, nil, err
		}

		return asset, encoded, nil
	}
	path := filepath.Join(d.root, "tools/fakefacegen/output/faces-v1/derived/256", id+".webp")
	if d.packaged {
		path = filepath.Join(d.root, "synthetic-world/photos/v1/256", id+".webp")
	}
	encoded, err := readBoundedPhotoFile(ctx, path, int(asset.ContentLength))
	if err != nil {
		return PhotoAsset{}, nil, err
	}
	if int64(len(encoded)) != asset.ContentLength || fmt.Sprintf("%x", sha256.Sum256(encoded)) != asset.ContentSHA256 {
		return PhotoAsset{}, nil, fmt.Errorf("altered USA demo photo asset")
	}

	return asset, encoded, nil
}

// PhotoCatalog returns the verified demo selection and rendition metadata.
func (d *USAWorldData) PhotoCatalog() (*FaceCatalog, error) {
	if d == nil || d.catalog == nil {
		return nil, fmt.Errorf("initialized USA demo data is required")
	}
	return d.catalog, nil
}

// USASourceObservation is a typed USD observation of the prepared USA World.
// Its transport remains private and does not extend the frozen SourceRecord.
type USASourceObservation = usaSourceUSDRecord

// USAWorld binds USA observations, photographs and the private M2 codec.
type USAWorld struct {
	adapter *usaSourceUSDAdapter
	codec   *usaM2USDCodec
}

// NewUSAWorld binds approved USA data and photos to a workspace and source.
// Invalid coordinates return ErrInvalidIdentityNamespace.
func NewUSAWorld(data *USAWorldData, namespace IdentityNamespace, seed uint64, source SourceInstanceConfig) (*USAWorld, error) {

	if data == nil || data.country == nil || data.catalog == nil {
		return nil, fmt.Errorf("complete USA demo data is required")
	}
	world, err := newUSACountryWorld(data.country, namespace, seed)
	if err != nil {
		return nil, err
	}
	world.catalog = data.catalog
	adapter, err := newUSASourceUSDAdapter(world, source)
	if err != nil {
		return nil, err
	}

	codec, err := newUSAM2USDCodec(adapter)
	if err != nil {
		return nil, err
	}

	return &USAWorld{adapter: adapter, codec: codec}, nil
}

// DecodeDocument validates the closed M2 candidate against this World and
// source without sampling a Person, Employer or Income.
// Cancellation returns ctx.Err().
func (w *USAWorld) DecodeDocument(ctx context.Context, encoded []byte) (USASourceObservation, error) {
	if w == nil || w.codec == nil || ctx == nil {
		return USASourceObservation{}, fmt.Errorf("initialized USA World and context are required")
	}
	document, err := w.codec.decode(ctx, encoded)
	if err != nil {
		return USASourceObservation{}, err
	}
	return document.SourceRecord, nil
}

// DocumentBinding identifies the adopted M2 USD schema, prepared provenance,
// World and source; it is not a country release descriptor or SnapshotID.
func (w *USAWorld) DocumentBinding() ([32]byte, error) {
	if w == nil || w.codec == nil || w.adapter == nil || w.adapter.source == nil {
		return [32]byte{}, fmt.Errorf("initialized USA World is required")
	}
	return internationalDigest("fakedata/usa-application-document/v1", struct {
		Schema     string
		Version    uint32
		Provenance usaM2USDProvenance
		World      SocioeconomicWorldID
		Source     SourceInstanceConfig
	}{usaM2USDSchema, usaM2USDDocumentVersion, w.codec.provenance, w.adapter.world.id, w.adapter.source.config})
}

// EncodeDocument encodes an existing observation using the adopted M2 USD
// schema; it does not publish country assets.
func (w *USAWorld) EncodeDocument(record USASourceObservation) ([]byte, error) {
	if w == nil || w.codec == nil {
		return nil, fmt.Errorf("initialized USA World is required")
	}
	return w.codec.encode(record)
}

// Records returns observations with PhotoID and the original typed USD facts.
// Invalid indices return ErrInvalidIndex or ErrLayer1IndexOutOfRange.
func (w *USAWorld) Records(index PersonIndex) ([]USASourceObservation, error) {

	if w == nil || w.adapter == nil {
		return nil, fmt.Errorf("initialized USA demo World is required")
	}
	person, err := w.adapter.world.person(index)
	if err != nil {
		return nil, err
	}
	records, _, err := w.adapter.bind(person)

	return records, err
}
