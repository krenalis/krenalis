// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

const (
	MaxRequiredConsentPurposes      = 100  // maximum allowed number of required consent purposes.
	maxConsentPurposeEventLocations = 5    // maximum number of event locations of a consent purpose
	maxConsentLocationStringLen     = 1024 // maximum length of a consent location string
)

// ConsentPurpose describes a consent purpose and where its consent value is
// represented in events and profiles.
type ConsentPurpose struct {
	ID                     string                  `json:"id"`
	Name                   string                  `json:"name"`
	EventConsentLocations  []EventConsentLocation  `json:"eventConsentLocations"`
	ProfileConsentLocation *ProfileConsentLocation `json:"profileConsentLocation"`
}

// ConsentPurposeToSet contains the fields used to create or update a consent purpose.
type ConsentPurposeToSet struct {
	Name                   string                  `json:"name"`
	EventConsentLocations  []EventConsentLocation  `json:"eventConsentLocations"`
	ProfileConsentLocation *ProfileConsentLocation `json:"profileConsentLocation"`
}

// EventConsentLocation identifies a property under context.consents that is checked
// for the consent value in incoming events.
type EventConsentLocation struct {
	PurposeCode string `json:"purposeCode"`
}

// ProfileConsentLocation identifies the profile schema property and optional JSON key
// used to represent the consent value for a purpose.
type ProfileConsentLocation struct {
	Property string `json:"property"`
	JSONKey  string `json:"jsonKey,omitempty"`
}

// AddConsentPurpose adds a consent purpose with the values of purpose.
func (this *Workspace) AddConsentPurpose(ctx context.Context, purpose ConsentPurposeToSet) error {
	this.core.mustBeOpen()
	err := validateConsentPurposeToSet(purpose)
	if err != nil {
		return errors.BadRequest("%s", err)
	}
	n := state.AddConsentPurpose{
		Workspace: this.workspace.ID,
		Name:      purpose.Name,
	}
	n.EventConsentLocations = make([]state.EventConsentLocation, len(purpose.EventConsentLocations))
	purposeCodes := make([]string, len(n.EventConsentLocations))
	for i, loc := range purpose.EventConsentLocations {
		n.EventConsentLocations[i] = state.EventConsentLocation{PurposeCode: loc.PurposeCode}
		purposeCodes[i] = loc.PurposeCode
	}
	var property, jsonKey string
	if loc := purpose.ProfileConsentLocation; loc != nil {
		n.ProfileConsentLocation = &state.ProfileConsentLocation{
			Property: loc.Property,
			JSONKey:  loc.JSONKey,
		}
		property, jsonKey = loc.Property, loc.JSONKey
	}
	n.ID = generateID(this.workspace.ConsentPurpose)
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		err := lockWorkspace(ctx, tx, n.Workspace)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(ctx, "INSERT INTO consent_purposes"+
			" (workspace, id, name, event_purpose_codes, profile_property, profile_json_key)"+
			" VALUES ($1, $2, $3, $4, $5, $6)", n.Workspace, n.ID, n.Name, purposeCodes, property, jsonKey)
		if err != nil {
			return nil, err
		}
		return n, nil
	})
	return err
}

// ConsentPurposes returns the consent purposes of the workspace, ordered by
// name.
func (this *Workspace) ConsentPurposes() []*ConsentPurpose {
	this.core.mustBeOpen()
	consentPurposes := this.workspace.ConsentPurposes()
	purposes := make([]*ConsentPurpose, len(consentPurposes))
	for i, p := range consentPurposes {
		purpose := &ConsentPurpose{
			ID:                    p.ID,
			Name:                  p.Name,
			EventConsentLocations: make([]EventConsentLocation, len(p.EventConsentLocations)),
		}
		for j, loc := range p.EventConsentLocations {
			purpose.EventConsentLocations[j] = EventConsentLocation(loc)
		}
		if loc := p.ProfileConsentLocation; loc != nil {
			purpose.ProfileConsentLocation = new(ProfileConsentLocation(*loc))
		}
		purposes[i] = purpose
	}
	sort.Slice(purposes, func(i, j int) bool {
		a, b := purposes[i], purposes[j]
		return a.Name < b.Name || a.Name == b.Name && a.ID < b.ID
	})
	return purposes
}

// DeleteConsentPurpose deletes the consent purpose with the given ID.
//
// It returns an errors.NotFoundError error if the consent purpose does not
// exist.
//
// It returns an errors.UnprocessableError error with code ConsentPurposeInUse
// if the consent purpose is currently required by one or more pipelines of the
// workspace.
func (this *Workspace) DeleteConsentPurpose(ctx context.Context, id string) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("identifier %q is not a valid consent purpose identifier", id)
	}
	if _, ok := this.workspace.ConsentPurpose(id); !ok {
		return errors.NotFound("consent purpose %s does not exist", id)
	}
	n := state.DeleteConsentPurpose{
		Workspace: this.workspace.ID,
		ID:        id,
	}
	return this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		err := lockWorkspace(ctx, tx, n.Workspace)
		if err != nil {
			return nil, err
		}
		var inUse bool
		err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pipelines p JOIN connections c ON p.connection = c.id "+
			"WHERE c.workspace = $1 AND $2 = ANY(p.required_consents))", n.Workspace, n.ID).Scan(&inUse)
		if err != nil {
			return nil, err
		}
		if inUse {
			return nil, errors.Unprocessable(ConsentPurposeInUse,
				"consent purpose %s is required by one or more pipelines", n.ID)
		}
		result, err := tx.Exec(ctx, "DELETE FROM consent_purposes WHERE workspace = $1 AND id = $2", n.Workspace, n.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("consent purpose %s does not exist", n.ID)
		}
		return n, nil
	})
}

// UpdateConsentPurpose updates the consent purpose with the given ID, setting
// its name and consent locations to those of purpose.
//
// It returns an errors.NotFoundError error if the consent purpose does not
// exist.
func (this *Workspace) UpdateConsentPurpose(ctx context.Context, id string, purpose ConsentPurposeToSet) error {

	this.core.mustBeOpen()

	if !IsValidID(id) {
		return errors.BadRequest("identifier %q is not a valid consent purpose identifier", id)
	}
	err := validateConsentPurposeToSet(purpose)
	if err != nil {
		return errors.BadRequest("%s", err)
	}

	n := state.UpdateConsentPurpose{
		Workspace: this.workspace.ID,
		ID:        id,
		Name:      purpose.Name,
	}
	n.EventConsentLocations = make([]state.EventConsentLocation, len(purpose.EventConsentLocations))
	purposeCodes := make([]string, len(n.EventConsentLocations))
	for i, loc := range purpose.EventConsentLocations {
		n.EventConsentLocations[i] = state.EventConsentLocation(loc)
		purposeCodes[i] = loc.PurposeCode
	}
	var property, jsonKey string
	if loc := purpose.ProfileConsentLocation; loc != nil {
		n.ProfileConsentLocation = new(state.ProfileConsentLocation(*loc))
		property, jsonKey = loc.Property, loc.JSONKey
	}

	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		err := lockWorkspace(ctx, tx, n.Workspace)
		if err != nil {
			return nil, err
		}
		result, err := tx.Exec(ctx, "UPDATE consent_purposes"+
			" SET name = $1, event_purpose_codes = $2, profile_property = $3, profile_json_key = $4"+
			" WHERE workspace = $5 AND id = $6", n.Name, purposeCodes, property, jsonKey, n.Workspace, n.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("consent purpose %s does not exist", n.ID)
		}
		return n, nil
	})

	return err
}

// knownConsentPurposeIDs returns the set of identifiers of the consent purposes
// defined in the workspace.
func knownConsentPurposeIDs(ws *state.Workspace) map[string]bool {
	ids := make(map[string]bool)
	for _, cp := range ws.ConsentPurposes() {
		ids[cp.ID] = true
	}
	return ids
}

// addRequiredConsentProperties adds to the given schema, when they are not
// already in it, the properties that hold the consents given for the purposes
// required by a pipeline, so that they are read together with the properties
// the pipeline declares. The properties are taken as they are from the given
// profile schema, so that the schema stays aligned with it.
//
// It returns the resulting schema and the set of the paths of the properties
// that have been added, which the caller must remove from what it reads before
// processing it, given that the pipeline does not declare them.
//
// A location with a JSON key adds the JSON property. A purpose whose property
// does not exist or has an incompatible type is skipped: the consent it
// requires is then never given.
//
// It returns an error if the schema has a non-object property along the path of
// the property that holds a consent.
func addRequiredConsentProperties(schema, profileSchema types.Type, purposes []*state.ConsentPurpose) (types.Type, map[string]bool, error) {
	if schema.Kind() != types.ObjectKind || profileSchema.Kind() != types.ObjectKind {
		return schema, nil, nil
	}
	var added map[string]bool
	for _, purpose := range purposes {
		if purpose == nil || purpose.ProfileConsentLocation == nil {
			continue
		}
		path := strings.Join(purpose.ProfilePropertyPath(), ".")
		if path == "" {
			continue
		}
		property, err := profileSchema.Properties().ByPath(path)
		if err != nil {
			// The property does not exist, so there is nothing to read and the
			// consent for the purpose is not given.
			continue
		}
		requiredKind := types.BooleanKind
		if purpose.ProfileConsentLocation.JSONKey != "" {
			requiredKind = types.JSONKind
		}
		if property.Type.Kind() != requiredKind {
			// The property type does not match the configured consent path.
			continue
		}
		s, isAdded, err := types.AddPropertyAtPath(schema, path, property)
		if err != nil {
			return schema, nil, fmt.Errorf("cannot read the consent for consent purpose %s: %s", purpose.ID, err)
		}
		if isAdded {
			if added == nil {
				added = map[string]bool{}
			}
			added[path] = true
			schema = s
		}
	}
	return schema, added, nil
}

// validateConsentPurposeToSet validates the name and consent locations of the
// given consent purpose.
func validateConsentPurposeToSet(purpose ConsentPurposeToSet) error {
	if err := util.ValidateStringField("name", purpose.Name, 100); err != nil {
		return err
	}
	// Validate event consent locations.
	if len(purpose.EventConsentLocations) > maxConsentPurposeEventLocations {
		return fmt.Errorf("consent purpose can have at most %d event consent locations",
			maxConsentPurposeEventLocations)
	}
	seenPurposeCodes := map[string]bool{}
	for _, loc := range purpose.EventConsentLocations {
		if err := util.ValidateStringField("purposeCode", loc.PurposeCode, maxConsentLocationStringLen); err != nil {
			return err
		}
		if seenPurposeCodes[loc.PurposeCode] {
			return fmt.Errorf("purpose code %q is duplicated", loc.PurposeCode)
		}
		seenPurposeCodes[loc.PurposeCode] = true
	}
	// Validate the profile consent location.
	if loc := purpose.ProfileConsentLocation; loc != nil {
		if err := util.ValidateStringField("property", loc.Property, maxConsentLocationStringLen); err != nil {
			return err
		}
		if !types.IsValidPropertyPath(loc.Property) {
			return fmt.Errorf("%q is not a valid property path", loc.Property)
		}
		if loc.JSONKey != "" {
			if err := util.ValidateStringField("JSON key", loc.JSONKey, maxConsentLocationStringLen); err != nil {
				return err
			}
		}
	}
	return nil
}
