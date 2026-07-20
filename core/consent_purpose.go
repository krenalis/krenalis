// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"sort"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/tools/errors"
)

// ConsentPurpose represents a purpose.
type ConsentPurpose struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// AddConsentPurpose adds a consent purpose with the given name and code, and
// returns its identifier.
//
// name and code must be between 1 and 100 runes long.
//
// It returns an errors.UnprocessableError error with code
// ConsentPurposeCodeExists if a consent purpose with the same code already
// exists in the workspace.
func (this *Workspace) AddConsentPurpose(ctx context.Context, name, code string) (string, error) {
	this.core.mustBeOpen()
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return "", errors.BadRequest("%s", err)
	}
	if err := util.ValidateStringField("code", code, 100); err != nil {
		return "", errors.BadRequest("%s", err)
	}
	n := state.AddConsentPurpose{
		Workspace: this.workspace.ID,
		Name:      name,
		Code:      code,
	}
	for {
		n.ID = generateID[any](nil)
		err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			_, err := tx.Exec(ctx, "INSERT INTO consent_purposes (id, workspace, name, code) VALUES ($1, $2, $3, $4)",
				n.ID, n.Workspace, n.Name, n.Code)
			if err != nil {
				return nil, err
			}
			return n, nil
		})
		if err != nil {
			if db.IsUniqueViolation(err) {
				switch db.ErrConstraintName(err) {
				case "consent_purposes_pkey":
					continue
				case "consent_purposes_workspace_code_key":
					return "", errors.Unprocessable(ConsentPurposeCodeExists, "a consent purpose with code %q already exists", n.Code)
				}
			}
			return "", err
		}
		break
	}
	return n.ID, nil
}

// ConsentPurposes returns the consent purposes of the workspace, ordered by
// name.
func (this *Workspace) ConsentPurposes() []*ConsentPurpose {
	this.core.mustBeOpen()
	consentPurposes := this.workspace.ConsentPurposes()
	purposes := make([]*ConsentPurpose, len(consentPurposes))
	for i, cp := range consentPurposes {
		purposes[i] = &ConsentPurpose{ID: cp.ID, Name: cp.Name, Code: cp.Code}
	}
	sort.Slice(purposes, func(i, j int) bool {
		a, b := purposes[i], purposes[j]
		return a.Name < b.Name || a.Name == b.Name && a.ID < b.ID
	})
	return purposes
}

// UpdateConsentPurpose updates the name and code of the consent purpose with
// identifier id.
//
// It returns an errors.NotFoundError error if the consent purpose does not
// exist.
//
// It returns an errors.UnprocessableError error with code
// ConsentPurposeCodeExists if another consent purpose with the same code
// already exists in the workspace.
func (this *Workspace) UpdateConsentPurpose(ctx context.Context, id, name, code string) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("identifier %q is not a valid consent purpose identifier", id)
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	if err := util.ValidateStringField("code", code, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	n := state.UpdateConsentPurpose{
		ID:        id,
		Workspace: this.workspace.ID,
		Name:      name,
		Code:      code,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE consent_purposes SET name = $1, code = $2 WHERE id = $3 AND workspace = $4",
			n.Name, n.Code, n.ID, n.Workspace)
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

// DeleteConsentPurpose deletes the consent purpose with identifier id.
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
	n := state.DeleteConsentPurpose{
		ID:        id,
		Workspace: this.workspace.ID,
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
		result, err := tx.Exec(ctx, "DELETE FROM consent_purposes WHERE id = $1 AND workspace = $2", n.ID, n.Workspace)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("consent purpose %s does not exist", n.ID)
		}
		return n, nil
	})
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
