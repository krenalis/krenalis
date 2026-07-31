// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"regexp"
	"sort"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/tools/errors"
)

const MaxRequiredConsentPurposes = 100 // maximum allowed number of required consent purposes.

// consentPurposeCodeFormat is the format of a consent purpose code.
var consentPurposeCodeFormat = regexp.MustCompile(`^[0-9A-Za-z._-]{1,100}$`)

// ConsentPurpose represents a purpose.
type ConsentPurpose struct {
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
	if !consentPurposeCodeFormat.MatchString(code) {
		return errors.BadRequest("code must be between 1 and 100 characters long and can only contain letters, digits, dots, hyphens and underscores")
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	n := state.AddConsentPurpose{
		Workspace: this.workspace.ID,
		Code:      code,
		Name:      name,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		_, err := tx.Exec(ctx, "INSERT INTO consent_purposes (workspace, code, name) VALUES ($1, $2, $3)",
			n.Workspace, n.Code, n.Name)
		if err != nil {
			return nil, err
		}
		return n, nil
	})
	if err != nil {
		if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "consent_purposes_pkey" {
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
		purposes[i] = &ConsentPurpose{Code: cp.Code, Name: cp.Name}
	}
	sort.Slice(purposes, func(i, j int) bool {
		a, b := purposes[i], purposes[j]
		return a.Name < b.Name || a.Name == b.Name && a.Code < b.Code
	})
	return purposes
}

// UpdateConsentPurpose updates the consent purpose with the given code, setting
// its code to code and its name to name.
//
// It returns an errors.NotFoundError error if the consent purpose does not
// exist.
//
// It returns an errors.UnprocessableError error with code
// ConsentPurposeCodeExists if another consent purpose with code newCode already
// exists in the workspace.
func (this *Workspace) UpdateConsentPurpose(ctx context.Context, purpose, code, name string) error {
	this.core.mustBeOpen()
	if !consentPurposeCodeFormat.MatchString(purpose) {
		return errors.BadRequest("code must be between 1 and 100 characters long and can only contain letters, digits, dots, hyphens and underscores")
	}
	if !consentPurposeCodeFormat.MatchString(code) {
		return errors.BadRequest("new code must be between 1 and 100 characters long and can only contain letters, digits, dots, hyphens and underscores")
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	current, ok := this.workspace.ConsentPurpose(purpose)
	if !ok {
		return errors.NotFound("consent purpose %q does not exist", purpose)
	}
	if code == current.Code && name == current.Name {
		return nil
	}
	n := state.UpdateConsentPurpose{
		Workspace: this.workspace.ID,
		Purpose:   purpose,
		Code:      code,
		Name:      name,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE consent_purposes SET code = $1, name = $2 WHERE workspace = $3 AND code = $4",
			n.Code, n.Name, n.Workspace, n.Purpose)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("consent purpose %q does not exist", n.Purpose)
		}
		return n, nil
	})
	if err != nil {
		if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "consent_purposes_pkey" {
			return errors.Unprocessable(ConsentPurposeCodeExists, "a consent purpose with code %q already exists", n.Code)
		}
		return err
	}
	return nil
}

// DeleteConsentPurpose deletes the consent purpose with the given code.
//
// It returns an errors.NotFoundError error if the consent purpose does not
// exist.
//
// It returns an errors.UnprocessableError error with code ConsentPurposeInUse
// if the consent purpose is currently required by one or more pipelines of the
// workspace.
func (this *Workspace) DeleteConsentPurpose(ctx context.Context, code string) error {
	this.core.mustBeOpen()
	if !consentPurposeCodeFormat.MatchString(code) {
		return errors.BadRequest("code must be between 1 and 100 characters long and can only contain letters, digits, dots, hyphens and underscores")
	}
	if _, ok := this.workspace.ConsentPurpose(code); !ok {
		return errors.NotFound("consent purpose %q does not exist", code)
	}
	n := state.DeleteConsentPurpose{
		Workspace: this.workspace.ID,
		Code:      code,
	}
	return this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		var inUse bool
		err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pipelines p JOIN connections c ON p.connection = c.id "+
			"WHERE c.workspace = $1 AND $2 = ANY(p.required_consents))", n.Workspace, n.Code).Scan(&inUse)
		if err != nil {
			return nil, err
		}
		if inUse {
			return nil, errors.Unprocessable(ConsentPurposeInUse, "consent purpose %q is required by one or more pipelines", n.Code)
		}
		result, err := tx.Exec(ctx, "DELETE FROM consent_purposes WHERE workspace = $1 AND code = $2", n.Workspace, n.Code)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("consent purpose %q does not exist", n.Code)
		}
		return n, nil
	})
}

// checkConsentPurposesExist checks that the workspace defines a consent purpose
// for each of the given codes.
//
// It returns an errors.UnprocessableError error with code
// ConsentPurposeNotExist, if a consent purpose does not exist.
func checkConsentPurposesExist(ws *state.Workspace, codes []string) error {
	for _, code := range codes {
		if _, ok := ws.ConsentPurpose(code); !ok {
			return errors.Unprocessable(ConsentPurposeNotExist, "consent purpose %q does not exist", code)
		}
	}
	return nil
}
