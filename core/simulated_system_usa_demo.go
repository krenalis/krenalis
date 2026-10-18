// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"net/url"
	"strings"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/simulated"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/validation"
)

const simulatedSystemUSADemoPolicyVersion = simulated.USADemoGenerationPolicy

// USADemoPreview is a bounded view of one stored internal demo record.
type USADemoPreview struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Photo           string `json:"photo"`
	Location        string `json:"location"`
	Education       string `json:"education"`
	PrimaryActivity string `json:"primaryActivity"`
	AnnualIncomeUSD uint32 `json:"annualIncomeUSD"`
}

// USADemoEnabled reports whether a USA World data source is configured.
func (core *Core) USADemoEnabled() bool {
	core.mustBeOpen()
	return core.usaWorldData != nil || core.usaWorldSource != nil
}

// SimulatedSystemUSAPhoto returns a verified internal demo image.
// It returns errors.NotFoundError for an unavailable image or workspace mode,
// and errors.UnavailableError if the configured asset fails integrity checks.
func (this *Workspace) SimulatedSystemUSAPhoto(ctx context.Context, id string) ([]byte, error) {

	this.core.mustBeOpen()

	if this.workspace.Environment != state.Development ||
		(this.core.usaWorldData == nil && this.core.usaWorldSource == nil) {
		return nil, errors.NotFound("USA demo photo is unavailable")
	}
	if len(id) != len("face-000001") || !strings.HasPrefix(id, "face-") {
		return nil, errors.NotFound("USA demo photo is unavailable")
	}
	worldData, err := this.core.usaData(ctx)
	if err != nil {
		return nil, errors.Unavailable("USA World data is unavailable")
	}
	_, data, err := worldData.Photo(ctx, id)
	if err != nil {
		if errors.Is(err, fakedata.ErrInvalidPhotoAsset) {
			return nil, errors.NotFound("USA demo photo is unavailable")
		}
		return nil, errors.Unavailable("USA demo photo failed integrity verification")
	}

	return data, nil
}

// SimulatedSystemUSAPreview returns up to eight stored generated USA records.
// Destination-created observations without canonical facts are omitted.
// It returns errors.BadRequestError for an invalid identifier,
// errors.NotFoundError for an unavailable system, and errors.UnavailableError
// when its stored records cannot be safely displayed.
func (this *Workspace) SimulatedSystemUSAPreview(ctx context.Context, id string) ([]USADemoPreview, error) {

	this.core.mustBeOpen()

	if !IsValidID(id) {
		return nil, errors.BadRequest("simulated system identifier is not valid")
	}
	account, exists := this.workspace.SimulatedSystem(id)
	if !exists || this.workspace.Environment != state.Development ||
		(account.GenerationPolicyVersion != simulatedSystemUSADemoPolicyVersion &&
			account.GenerationPolicyVersion != simulated.USAGenerationPolicy) ||
		(this.core.usaWorldData == nil && this.core.usaWorldSource == nil) {
		return nil, errors.NotFound("USA demo system is unavailable")
	}
	data, err := this.core.usaData(ctx)
	if err != nil {
		return nil, errors.Unavailable("USA World data is unavailable")
	}
	catalog, err := data.PhotoCatalog()
	if err != nil {
		return nil, errors.Unavailable("USA demo configuration is unavailable")
	}
	base, err := validation.ParseURL(this.core.usaPhotoBaseURL, validation.NoQuery)
	if err != nil {
		return nil, errors.Unavailable("USA photo configuration is unavailable")
	}
	photoURL, err := url.Parse(base)
	if err != nil {
		return nil, errors.Unavailable("USA photo configuration is unavailable")
	}
	if photoURL.Host == "" {
		return nil, errors.Unavailable("USA photo configuration is unavailable")
	}
	photoURL = photoURL.JoinPath(this.workspace.ID)
	var documents *simulated.USADocuments
	if account.GenerationPolicyVersion == simulated.USAGenerationPolicy {
		documents, err = this.core.usaDocuments(ctx, this.workspace.ID, account.ID)
		if err != nil {
			return nil, errors.Unavailable("USA application configuration is unavailable")
		}
	}
	page, err := this.store.SimulatedSystemRecords(ctx, id, datastore.SimulatedRecordQuery{Limit: 8})
	if err != nil {
		return nil, errors.Unavailable("USA demo records are unavailable")
	}
	result := make([]USADemoPreview, 0, len(page.Records))
	for _, record := range page.Records {

		if len(record.Attributes) > 32<<10 {
			return nil, errors.Unavailable("USA demo record exceeds display limits")
		}
		var document simulatedSystemUSADemoDocument
		encoded := record.Attributes
		if documents != nil {
			attributes, err := documents.Decode(ctx, record.ID, encoded)
			if err != nil {
				return nil, errors.Unavailable("USA application record is invalid")
			}
			if _, generated := attributes["annual_income_usd"]; !generated {
				continue
			}
			encoded, err = json.Marshal(attributes)
			if err != nil {
				return nil, errors.Unavailable("USA application record is invalid")
			}
		}
		err = json.Unmarshal(encoded, &document)
		if err != nil {
			return nil, errors.Unavailable("USA demo record is invalid")
		}
		if document.Country != "US" || document.Currency != "USD" || document.AnnualIncomeUSD == nil ||
			document.AnnualIncomeEUR != nil || document.Address == nil ||
			document.CustomerID != record.ID || len(record.ID) > 256 {
			return nil, errors.Unavailable("USA demo record is incompatible")
		}
		_, err = catalog.Asset(document.PhotoID, fakedata.PhotoSize256)
		if err != nil {
			return nil, errors.Unavailable("USA demo record photo is invalid")
		}
		name := ""
		for _, part := range []*string{document.FirstName, document.LastName} {
			if part != nil {
				name += *part + " "
			}
		}
		location := document.Address.Municipality + ", " + document.Address.StateProv
		for _, text := range []string{name, location, string(document.EducationLevel), string(document.PrimaryActivity)} {
			err = util.ValidateStringField("demo record text", text, 256)
			if err != nil {
				return nil, errors.Unavailable("USA demo record text is invalid")
			}
		}
		result = append(result, USADemoPreview{ID: record.ID, Name: strings.TrimSpace(name),
			Photo: photoURL.JoinPath(document.PhotoID + ".webp").String(), Location: location,
			Education: string(document.EducationLevel), PrimaryActivity: string(document.PrimaryActivity),
			AnnualIncomeUSD: *document.AnnualIncomeUSD})

	}

	return result, nil
}

// The USA projection adds USD without extending the frozen Italian document.
type simulatedSystemUSADemoDocument struct {
	simulatedSystemDocument
	Country         string  `json:"country"`
	Currency        string  `json:"currency"`
	PhotoID         string  `json:"photo_id"`
	AnnualIncomeUSD *uint32 `json:"annual_income_usd"`
}
