// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"sort"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

const (
	MaxRequiredConsentPurposes = 100 // maximum allowed number of required consent purposes.
	maxConsentPurposeCodeLen   = 100 // maximum length of a consent purpose code.
)

// ConsentPurpose represents a purpose.
type ConsentPurpose struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// AddConsentPurpose adds a consent purpose with the given code and name.
//
// code must be a valid consent purpose code and name must be between 1 and 100
// runes long.
//
// It returns an errors.UnprocessableError error with code
// ConsentPurposeCodeExists if a consent purpose with the same code already
// exists in the workspace.
func (this *Workspace) AddConsentPurpose(ctx context.Context, code, name string) error {
	this.core.mustBeOpen()
	if err := validateConsentPurposeCode(code); err != nil {
		return errors.BadRequest("%s", err)
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	n := state.AddConsentPurpose{
		Workspace: this.workspace.ID,
		Code:      code,
		Name:      name,
	}
	n.ID = generateID(this.workspace.ConsentPurpose)
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		_, err := tx.Exec(ctx, "INSERT INTO consent_purposes (workspace, id, code, name) VALUES ($1, $2, $3, $4)",
			n.Workspace, n.ID, n.Code, n.Name)
		if err != nil {
			return nil, err
		}
		return n, nil
	})
	if err != nil {
		if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "consent_purposes_workspace_code_key" {
			return errors.Unprocessable(ConsentPurposeCodeExists, "a consent purpose with code %q already exists", n.Code)
		}
		return err
	}
	return nil
}

// ConsentPurposes returns the consent purposes of the workspace, ordered by
// name.
func (this *Workspace) ConsentPurposes() []*ConsentPurpose {
	this.core.mustBeOpen()
	consentPurposes := this.workspace.ConsentPurposes()
	purposes := make([]*ConsentPurpose, len(consentPurposes))
	for i, cp := range consentPurposes {
		purposes[i] = &ConsentPurpose{ID: cp.ID, Code: cp.Code, Name: cp.Name}
	}
	sort.Slice(purposes, func(i, j int) bool {
		a, b := purposes[i], purposes[j]
		return a.Name < b.Name || a.Name == b.Name && a.Code < b.Code
	})
	return purposes
}

// DeleteConsentPurpose deletes the consent purpose with the given id.
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
		var inUse bool
		err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pipelines p JOIN connections c ON p.connection = c.id "+
			"WHERE c.workspace = $1 AND $2 = ANY(p.required_consents))", n.Workspace, n.ID).Scan(&inUse)
		if err != nil {
			return nil, err
		}
		if inUse {
			return nil, errors.Unprocessable(ConsentPurposeInUse, "consent purpose %s is required by one or more pipelines", n.ID)
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

// UpdateConsentPurpose updates the consent purpose with the given id, setting
// its code and name to those of purpose.
//
// It returns an errors.NotFoundError error if the consent purpose does not
// exist.
//
// It returns an errors.UnprocessableError error with code
// ConsentPurposeCodeExists if another consent purpose with the new code already
// exists in the workspace.
func (this *Workspace) UpdateConsentPurpose(ctx context.Context, id string, purpose ConsentPurpose) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("identifier %q is not a valid consent purpose identifier", id)
	}
	if err := validateConsentPurposeCode(purpose.Code); err != nil {
		return errors.BadRequest("%s", err)
	}
	if err := util.ValidateStringField("name", purpose.Name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	current, ok := this.workspace.ConsentPurpose(id)
	if !ok {
		return errors.NotFound("consent purpose %s does not exist", id)
	}
	if purpose.Code == current.Code && purpose.Name == current.Name {
		return nil
	}
	n := state.UpdateConsentPurpose{
		Workspace: this.workspace.ID,
		ID:        id,
		Code:      purpose.Code,
		Name:      purpose.Name,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE consent_purposes SET code = $1, name = $2 WHERE workspace = $3 AND id = $4",
			n.Code, n.Name, n.Workspace, n.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("consent purpose %s does not exist", n.ID)
		}
		return n, nil
	})
	if err != nil {
		if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "consent_purposes_workspace_code_key" {
			return errors.Unprocessable(ConsentPurposeCodeExists, "a consent purpose with code %q already exists", n.Code)
		}
		return err
	}
	return nil
}

// IsValidConsentPurposeCode returns whether the given consent purpose code is
// valid.
//
// A valid consent purpose code has the same format as a property name and is at
// most maxConsentPurposeCodeLen characters long.
func IsValidConsentPurposeCode(code string) bool {
	return validateConsentPurposeCode(code) == nil
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

// validateConsentPurposeCode validates the given consent purpose code.
func validateConsentPurposeCode(code string) error {
	if len(code) > maxConsentPurposeCodeLen || !types.IsValidPropertyName(code) {
		return fmt.Errorf("code %q is not a valid consent purpose code. Consent purpose codes must be from 1 to %d"+
			" characters long, must start with a letter or underscore [A-Za-z_] and subsequently contain only letters,"+
			" numbers, or underscores [A-Za-z0-9_]", code, maxConsentPurposeCodeLen)
	}
	return nil
}
