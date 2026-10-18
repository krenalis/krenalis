// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"fmt"
	"time"
)

const (
	usaM2USDSchema          = "sw-country-aware-usd-m2-candidate-v1"
	usaM2USDDocumentVersion = 1
	usaM2USDDocumentBytes   = usaSourceUSDRecordBytes + 4096
)

// usaM2USDCodec binds the owner-adopted M2 USD document to one prepared source
// and World; application persistence keeps its separate Source representation.
type usaM2USDCodec struct {
	adapter    *usaSourceUSDAdapter
	provenance usaM2USDProvenance
}

func newUSAM2USDCodec(adapter *usaSourceUSDAdapter) (*usaM2USDCodec, error) {

	if adapter == nil || adapter.world == nil || adapter.source == nil || adapter.world.data == nil ||
		adapter.world.data.base == nil || adapter.world.data.phones == nil ||
		adapter.world.data.socioeconomic == nil || adapter.world.data.socioeconomic.model == nil {
		return nil, fmt.Errorf("prepared USA World and source are required for the M2 candidate")
	}
	w := adapter.world
	d := w.data.socioeconomic
	if !d.model.initialized || d.document.SourceModel == nil || w.baseID == "" || w.id == "" {
		return nil, fmt.Errorf("complete USA model provenance is required for the M2 candidate")
	}
	digest, err := d.logicalDigest()
	if err != nil {
		return nil, err
	}
	provenance := usaM2USDProvenance{
		BaseWorldID: w.baseID, BaseDataDigest: fmt.Sprintf("%x", w.data.base.digest),
		PhoneRuleDigest: fmt.Sprintf("%x", w.data.phones.identity), ModelID: fmt.Sprintf("%x", d.model.digest),
		SocioeconomicDataDigest: fmt.Sprintf("%x", digest), SourceAssetsSHA256: d.document.SourceModel.ByteSHA256,
		AnchorDate: d.document.AnchorDate, GenerationProtocol: usaSourceUSDVersion,
	}

	return &usaM2USDCodec{adapter: adapter, provenance: provenance}, nil
}

// decode accepts only the closed candidate envelope, without an expected
// original record or a new Person, Employer or Income draw.
func (c *usaM2USDCodec) decode(ctx context.Context, encoded []byte) (usaM2USDDocument, error) {

	if len(encoded) > usaM2USDDocumentBytes {
		return usaM2USDDocument{}, fmt.Errorf("M2 candidate document exceeds byte limit")
	}
	var document usaM2USDDocument
	err := decodeInternationalJSON(ctx, bytes.NewReader(encoded), &document)
	if err != nil {
		return usaM2USDDocument{}, err
	}
	err = c.validate(document)
	if err != nil {
		return usaM2USDDocument{}, err
	}

	return document, nil
}

// encode reuses the existing observation DTO inside the owner-adopted M2
// envelope without creating a new Person, Employer or Income draw.
func (c *usaM2USDCodec) encode(record usaSourceUSDRecord) ([]byte, error) {

	if c == nil {
		return nil, fmt.Errorf("initialized M2 candidate codec is required")
	}
	document := usaM2USDDocument{
		Schema: usaM2USDSchema, DocumentVersion: usaM2USDDocumentVersion,
		Provenance: c.provenance, SourceRecord: record,
	}
	err := c.validate(document)
	if err != nil {
		return nil, err
	}
	encoded, err := internationalCanonical(document)
	if err != nil {
		return nil, err
	}
	if len(encoded) > usaM2USDDocumentBytes {
		return nil, fmt.Errorf("M2 candidate document exceeds byte limit")
	}

	return encoded, nil
}

func (c *usaM2USDCodec) validate(document usaM2USDDocument) error {

	if c == nil || c.adapter == nil || document.Schema != usaM2USDSchema ||
		document.DocumentVersion != usaM2USDDocumentVersion || document.Provenance != c.provenance {
		return fmt.Errorf("incompatible M2 candidate version or World and model provenance")
	}
	err := c.adapter.validate(document.SourceRecord)
	if err != nil {
		return err
	}
	_, err = time.Parse(time.DateOnly, document.SourceRecord.Observation.BirthDate)
	if err != nil {
		return fmt.Errorf("invalid M2 candidate birth date")
	}

	return nil
}

// usaM2USDDocument is the owner-adopted versioned US/USD envelope around the
// existing observation DTO, not a replacement for Italy V5 or application Source.
type usaM2USDDocument struct {
	Schema          string             `json:"schema"`
	DocumentVersion uint32             `json:"documentVersion"`
	Provenance      usaM2USDProvenance `json:"provenance"`
	SourceRecord    usaSourceUSDRecord `json:"sourceRecord"`
}

// usaM2USDProvenance identifies the prepared numerical and non-photo World
// inputs; none of these fields asserts an official SnapshotID or release.
type usaM2USDProvenance struct {
	BaseWorldID             WorldID `json:"baseWorldID"`
	BaseDataDigest          string  `json:"baseDataDigest"`
	PhoneRuleDigest         string  `json:"phoneRuleDigest"`
	ModelID                 string  `json:"modelID"`
	SocioeconomicDataDigest string  `json:"socioeconomicDataDigest"`
	SourceAssetsSHA256      string  `json:"sourceAssetsSHA256"`
	AnchorDate              string  `json:"anchorDate"`
	GenerationProtocol      string  `json:"generationProtocol"`
}
