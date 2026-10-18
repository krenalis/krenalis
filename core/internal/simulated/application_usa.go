// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import (
	"bytes"
	"context"
	jsonv2 "encoding/json/v2"
	"fmt"
	"net/url"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// USAGenerationPolicy identifies private country-aware USA application data.
const USAGenerationPolicy = "generation-us-v2"

const usaApplicationDocumentSchema = "sw-application-us-usd-m2-v1"

var usaObservationSchema = func() types.Type {
	properties := append([]types.Property{{Name: "customer_id", Type: types.String().WithMaxLength(256)}},
		destinationSchema.Properties().Slice()...)
	for i := range properties {
		switch properties[i].Name {
		case "first_name", "last_name", "email", "phone":
			properties[i].ReadOptional = true
		}
	}
	return types.Object(properties)
}()

// USADocuments keeps the M2 candidate private while exposing the existing
// application schema; Destination edits cannot overwrite canonical USD facts.
type USADocuments struct {
	world        *fakedata.USAWorld
	photoBaseURL *url.URL
}

// NewUSADocuments binds private documents to a prepared World and photo URL.
func NewUSADocuments(world *fakedata.USAWorld, photoBaseURL *url.URL) (*USADocuments, error) {

	if world == nil || photoBaseURL == nil || photoBaseURL.Host == "" ||
		(photoBaseURL.Scheme != "http" && photoBaseURL.Scheme != "https") {
		return nil, fmt.Errorf("prepared USA World and photo URL are required")
	}
	_, err := world.DocumentBinding()
	if err != nil {
		return nil, err
	}
	base := *photoBaseURL

	return &USADocuments{world: world, photoBaseURL: &base}, nil
}

// Decode validates a bounded stored document and returns its Source projection.
// Cancellation returns ctx.Err().
func (d *USADocuments) Decode(ctx context.Context, id string, stored json.Value) (map[string]any, error) {
	return d.decodeSource(ctx, id, stored)
}

// Record samples once and persists the existing candidate USD envelope with
// independently writable application observations.
func (d *USADocuments) Record(index fakedata.PersonIndex, representation int) (datastore.SimulatedSystemRecord, error) {

	if d == nil || d.world == nil || d.photoBaseURL == nil {
		return datastore.SimulatedSystemRecord{}, fmt.Errorf("initialized USA documents are required")
	}
	records, err := d.world.Records(index)
	if err != nil {
		return datastore.SimulatedSystemRecord{}, err
	}
	if representation < 0 || representation >= len(records) {
		return datastore.SimulatedSystemRecord{}, fmt.Errorf("USA observation representation is out of range")
	}
	record := records[representation]
	o := record.Observation
	if o.PhotoID == nil {
		return datastore.SimulatedSystemRecord{}, fmt.Errorf("USA observation photo is missing")
	}
	source, err := d.world.EncodeDocument(record)
	if err != nil {
		return datastore.SimulatedSystemRecord{}, err
	}
	birthDate, err := time.Parse(time.DateOnly, o.BirthDate)
	if err != nil {
		return datastore.SimulatedSystemRecord{}, fmt.Errorf("invalid USA observation birth date")
	}
	attributes := map[string]any{
		"customer_id": o.ID, "gender": string(o.Gender), "birth_date": birthDate,
		"current_age": int(o.CurrentAge), "site_url": o.SiteURL,
		"photo": d.photoBaseURL.JoinPath(*o.PhotoID + ".webp").String(),
		"address": map[string]any{"street": o.Street, "municipality": o.Municipality,
			"state_prov": o.State, "postal_code": o.ZIP, "country": "US"},
	}
	for name, value := range map[string]*string{
		"first_name": o.FirstName, "last_name": o.LastName, "email": o.Email, "phone": o.Phone,
	} {
		if value != nil {
			attributes[name] = *value
		}
	}
	observation, err := types.MarshalValidate(attributes, usaObservationSchema)
	if err != nil {
		return datastore.SimulatedSystemRecord{}, err
	}
	encoded, err := d.encode(source, observation)
	if err != nil {
		return datastore.SimulatedSystemRecord{}, err
	}

	return datastore.SimulatedSystemRecord{ID: o.ID, Data: encoded}, nil
}

// create stores a Destination observation without inventing canonical facts.
func (d *USADocuments) create(id string, attributes map[string]any) (json.Value, error) {

	if len(id) == 0 || len(id) > 256 {
		return nil, fmt.Errorf("invalid USA observation identity")
	}
	err := validateDestinationAttributes(attributes, true)
	if err != nil {
		return nil, err
	}
	observation, err := newDestinationRecordDocument(id, attributes)
	if err != nil {
		return nil, err
	}

	return d.encode(nil, observation)
}

func (d *USADocuments) decode(ctx context.Context, id string, stored json.Value) (usaApplicationDocument, map[string]any, error) {

	if d == nil || d.world == nil || ctx == nil || len(id) == 0 || len(id) > 256 || len(stored) > 32<<10 {
		return usaApplicationDocument{}, nil, fmt.Errorf("invalid USA application document context or size")
	}
	var document usaApplicationDocument
	err := jsonv2.Unmarshal(stored, &document, jsonv2.RejectUnknownMembers(true))
	if err != nil {
		return usaApplicationDocument{}, nil, fmt.Errorf("invalid USA application document")
	}
	if document.Schema != usaApplicationDocumentSchema || len(document.Observation) > 12<<10 {
		return usaApplicationDocument{}, nil, fmt.Errorf("invalid USA application document")
	}
	attributes, err := types.Decode[map[string]any](bytes.NewReader(document.Observation), usaObservationSchema)
	if err != nil {
		return usaApplicationDocument{}, nil, fmt.Errorf("invalid USA application observation")
	}
	if attributes["customer_id"] != id {
		return usaApplicationDocument{}, nil, fmt.Errorf("USA observation identity differs from external ID")
	}
	address, ok := attributes["address"].(map[string]any)
	if !ok || address["country"] != "US" {
		return usaApplicationDocument{}, nil, fmt.Errorf("USA observation requires a US address")
	}
	attributes["country"], attributes["currency"] = "US", "USD"
	if document.Source == nil {
		return document, attributes, nil
	}
	record, err := d.world.DecodeDocument(ctx, document.Source)
	if err != nil {
		return usaApplicationDocument{}, nil, fmt.Errorf("invalid USA candidate document")
	}
	if record.Observation.ID != id || record.Observation.PhotoID == nil {
		return usaApplicationDocument{}, nil, fmt.Errorf("USA candidate identity or photo differs from observation")
	}
	attributes["photo_id"] = *record.Observation.PhotoID
	attributes["education_level"] = string(record.Education)
	attributes["primary_activity"] = string(record.PrimaryActivity)
	attributes["annual_income_usd"] = uint(*record.AnnualIncomeUSD)
	if record.Employee != nil {
		attributes["job_family"] = string(record.Employee.JobFamily)
		attributes["industry"] = string(record.Employee.Industry)
		attributes["job_level"] = string(record.Employee.JobLevel)
	}
	if record.SelfEmployed != nil {
		attributes["job_family"] = string(record.SelfEmployed.JobFamily)
		attributes["industry"] = string(record.SelfEmployed.Industry)
	}

	return document, attributes, nil
}

func (d *USADocuments) decodeSource(ctx context.Context, id string, stored json.Value) (map[string]any, error) {
	_, attributes, err := d.decode(ctx, id, stored)
	return attributes, err
}

func (d *USADocuments) encode(source, observation json.Value) (json.Value, error) {

	if d == nil || d.world == nil || len(observation) == 0 || len(observation) > 12<<10 || len(source) > 20<<10 {
		return nil, fmt.Errorf("invalid USA application document size")
	}
	var attributes map[string]json.Value
	err := json.Unmarshal(observation, &attributes)
	if err != nil {
		return nil, fmt.Errorf("invalid USA observation")
	}
	var address struct {
		Country string `json:"country"`
	}
	err = json.Unmarshal(attributes["address"], &address)
	if err != nil {
		return nil, fmt.Errorf("invalid USA observation address")
	}
	if address.Country != "US" {
		return nil, fmt.Errorf("USA observation requires a US address")
	}

	return json.Marshal(usaApplicationDocument{
		Schema: usaApplicationDocumentSchema, Source: source, Observation: observation,
	})
}

func (d *USADocuments) update(ctx context.Context, id string, stored json.Value, patch map[string]any) (json.Value, error) {

	document, _, err := d.decode(ctx, id, stored)
	if err != nil {
		return nil, err
	}
	observation, err := updateDestinationRecordDocument(id, document.Observation, patch, usaObservationSchema)
	if err != nil {
		return nil, err
	}

	return d.encode(document.Source, observation)
}

func (app *Application) countryDocuments(ctx context.Context) (*USADocuments, error) {

	if app.store == nil || app.store.account == "" {
		return nil, nil
	}
	workspace, ok := app.store.state.Workspace(app.store.workspace)
	if !ok {
		return nil, fmt.Errorf("simulated application workspace is unavailable")
	}
	account, ok := workspace.SimulatedSystem(app.store.account)
	if !ok {
		return nil, fmt.Errorf("simulated application account is unavailable")
	}
	if account.GenerationPolicyVersion != USAGenerationPolicy {
		return nil, nil
	}
	if app.store.usaDocuments == nil {
		return nil, fmt.Errorf("USA application document provider is unavailable")
	}

	return app.store.usaDocuments(ctx)
}

type usaApplicationDocument struct {
	Schema      string     `json:"schema"`
	Source      json.Value `json:"source,omitempty"`
	Observation json.Value `json:"observation"`
}
