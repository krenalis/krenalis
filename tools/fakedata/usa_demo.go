// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/krenalis/krenalis/tools/externalassets"

	"golang.org/x/image/webp"
)

// USASourceVersion identifies observations of the internal USA demo.
const USASourceVersion = usaSourceUSDVersion

// USAWorldData owns the approved USD model and verified photo catalog.
type USAWorldData struct {
	country  *usaCountryData
	catalog  *FaceCatalog
	digest   [32]byte
	resolver *externalassets.Resolver
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

	if d == nil || d.catalog == nil || d.resolver == nil || ctx == nil {
		return PhotoAsset{}, nil, fmt.Errorf("initialized remote USA data and context are required")
	}
	asset, err := d.catalog.asset(id, PhotoSize256)
	if err != nil {
		return PhotoAsset{}, nil, err
	}

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
