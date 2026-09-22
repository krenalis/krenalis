// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

const (
	simulatedAccountMaxUserCount  = 1_000_000
	simulatedAccountPolicyVersion = "generation-v1"
)

// SimulatedAccount represents the metadata of a simulated account.
type SimulatedAccount struct {
	ID                      string          `json:"id"`
	Workspace               string          `json:"workspace"`
	Name                    string          `json:"name"`
	Status                  string          `json:"status"`
	UserCount               int             `json:"userCount"`
	DuplicateRecordPercent  decimal.Decimal `json:"duplicateRecordPercent"`
	Countries               json.Value      `json:"countries"`
	GenerationPolicyVersion string          `json:"generationPolicyVersion"`
	GeneratedRecordCount    int             `json:"generatedRecordCount"`
	GenerationError         string          `json:"generationError"`
	CreatedAt               time.Time       `json:"createdAt"`
	UpdatedAt               time.Time       `json:"updatedAt"`
}

// CreateSimulatedAccount creates a simulated account in the workspace.
//
// It returns an errors.BadRequestError for invalid configuration and an
// errors.UnavailableError when preparation storage is unavailable.
//
// It returns an errors.UnprocessableError error with code:
//
//   - SimulatedAccountsRequireDevelopment, if the workspace is not in
//     development.
//   - MaintenanceMode, if the warehouse is in maintenance mode.
func (this *Workspace) CreateSimulatedAccount(ctx context.Context, name string, userCount, duplicateRecordPercent, countries json.Value) (string, error) {
	this.core.mustBeOpen()
	if this.workspace.Environment != state.Development {
		return "", errors.Unprocessable(SimulatedAccountsRequireDevelopment, "simulated accounts are only available in development workspaces")
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return "", errors.BadRequest("%s", err)
	}
	userCountValue, duplicateValue, countriesValue, err := validateSimulatedAccountConfiguration(userCount, duplicateRecordPercent, countries)
	if err != nil {
		return "", errors.BadRequest("%s", err)
	}
	status := state.SimulatedAccountReady
	policy := "empty-v1"
	var checkpoint json.Value
	var checkpointParameter any
	if userCountValue > 0 {
		if this.core.simulatedCatalog == nil {
			return "", errors.Unavailable("simulated account photo catalog is unavailable")
		}
		status = state.SimulatedAccountPreparing
		policy = simulatedAccountPolicyVersion
		checkpoint, err = initialSimulatedAccountCheckpoint(this.core.simulatedCatalog)
		if err != nil {
			return "", fmt.Errorf("cannot prepare simulated account checkpoint")
		}
		checkpointParameter = string(checkpoint)
	}
	_, err = this.store.CountSimulatedAccountRecords(ctx, "")
	if err != nil {
		return "", simulatedAccountWarehouseError(err)
	}
	n := state.CreateSimulatedAccount{
		Workspace:               this.workspace.ID,
		Name:                    name,
		UserCount:               userCountValue,
		DuplicateRecordPercent:  duplicateValue.String(),
		Countries:               countriesValue,
		Status:                  status,
		GenerationPolicyVersion: policy,
		GenerationCheckpoint:    checkpoint,
		GenerationError:         "",
		CreatedAt:               time.Now().UTC(),
	}
	n.UpdatedAt = n.CreatedAt
	for {
		n.ID = generateID(this.core.state.SimulatedAccount)
		err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			_, err := tx.Exec(ctx, "INSERT INTO simulated_accounts (id, workspace, name, status, user_count,"+
				" duplicate_record_percent, countries, generation_policy_version, generated_record_count,"+
				" generation_checkpoint, generation_error, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8,"+
				" $9, $10, $11, $12, $13)", n.ID, n.Workspace, n.Name, n.Status, n.UserCount,
				duplicateValue, string(n.Countries), n.GenerationPolicyVersion, n.GeneratedRecordCount, checkpointParameter,
				n.GenerationError, n.CreatedAt, n.UpdatedAt)
			if err != nil {
				return nil, err
			}
			return n, nil
		})
		if err != nil {
			if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "simulated_accounts_pkey" {
				continue
			}
			return "", err
		}
		if status == state.SimulatedAccountPreparing {
			this.core.wakeSimulatedAccountWorker()
		}
		return n.ID, nil
	}
}

// DeleteSimulatedAccount deletes a simulated account and its warehouse records.
//
// It returns an errors.BadRequestError for an invalid identifier and an
// errors.UnavailableError when warehouse cleanup fails.
//
// It returns an errors.NotFoundError error if the account does not exist. It
// returns an errors.UnprocessableError with code SimulatedAccountPreparing if
// the account is still being prepared, SimulatedAccountInUse if the account is
// referenced by a connection, or InspectionMode or MaintenanceMode if warehouse
// cleanup is unavailable in the current mode.
func (this *Workspace) DeleteSimulatedAccount(ctx context.Context, id string) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("simulated account identifier %q is not valid", id)
	}
	n := state.DeleteSimulatedAccount{Workspace: this.workspace.ID, ID: id}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		var status state.SimulatedAccountStatus
		err := tx.QueryRow(ctx, "SELECT status FROM simulated_accounts WHERE id = $1 AND workspace = $2 FOR UPDATE",
			n.ID, n.Workspace).Scan(&status)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errors.NotFound("simulated account %s does not exist", n.ID)
			}
			return nil, err
		}
		switch status {
		case state.SimulatedAccountPreparing:
			return nil, errors.Unprocessable(SimulatedAccountPreparing, "simulated account %s is still being prepared", n.ID)
		case state.SimulatedAccountReady, state.SimulatedAccountFailed:
		default:
			return nil, fmt.Errorf("simulated account %s has an invalid status", n.ID)
		}
		var connectionID string
		err = tx.QueryRow(ctx, "SELECT id FROM connections WHERE workspace = $1 AND simulated_account = $2 "+
			"ORDER BY id LIMIT 1",
			n.Workspace, n.ID).Scan(&connectionID)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if err == nil {
			return nil, errors.Unprocessable(SimulatedAccountInUse,
				"simulated account %s is referenced by connection %s", n.ID, connectionID)
		}
		err = this.store.DeleteSimulatedAccountRecords(ctx, n.ID)
		if err != nil {
			return nil, simulatedAccountWarehouseError(err)
		}
		result, err := tx.Exec(ctx, "DELETE FROM simulated_accounts WHERE id = $1 AND workspace = $2", n.ID, n.Workspace)
		if err != nil {
			if db.IsForeignKeyViolation(err) && db.ErrConstraintName(err) == "connections_workspace_simulated_account_fkey" {
				return nil, errors.Unprocessable(SimulatedAccountInUse,
					"simulated account %s is referenced by one or more connections", n.ID)
			}
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("simulated account %s does not exist", n.ID)
		}
		return n, nil
	})
	return err
}

// SimulatedAccount returns a simulated account of the workspace.
// It returns an errors.BadRequestError for an invalid identifier and an
// errors.NotFoundError if the account does not exist.
func (this *Workspace) SimulatedAccount(id string) (*SimulatedAccount, error) {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return nil, errors.BadRequest("simulated account identifier %q is not valid", id)
	}
	account, ok := this.workspace.SimulatedAccount(id)
	if !ok {
		return nil, errors.NotFound("simulated account %s does not exist", id)
	}
	return simulatedAccountFromState(account), nil
}

// SimulatedAccounts returns the simulated accounts of the workspace, ordered
// by name and then identifier.
func (this *Workspace) SimulatedAccounts() []*SimulatedAccount {
	this.core.mustBeOpen()
	accounts := this.workspace.SimulatedAccounts()
	result := make([]*SimulatedAccount, len(accounts))
	for i, account := range accounts {
		result[i] = simulatedAccountFromState(account)
	}
	return result
}

// RenameSimulatedAccount changes the name of a simulated account.
// It returns an errors.BadRequestError for invalid input and an
// errors.NotFoundError if the account does not exist.
func (this *Workspace) RenameSimulatedAccount(ctx context.Context, id, name string) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("simulated account identifier %q is not valid", id)
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	account, ok := this.workspace.SimulatedAccount(id)
	if !ok {
		return errors.NotFound("simulated account %s does not exist", id)
	}
	if account.Name == name {
		return nil
	}
	n := state.UpdateSimulatedAccount{Workspace: this.workspace.ID, ID: id, Name: name, UpdatedAt: time.Now().UTC()}
	return this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE simulated_accounts SET name = $1, updated_at = $2 WHERE id = $3 AND workspace = $4",
			n.Name, n.UpdatedAt, n.ID, n.Workspace)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("simulated account %s does not exist", n.ID)
		}
		return n, nil
	})
}

func simulatedAccountFromState(account *state.SimulatedAccount) *SimulatedAccount {
	workspace := account.Workspace()
	workspaceID := ""
	if workspace != nil {
		workspaceID = workspace.ID
	}
	return &SimulatedAccount{
		ID:                      account.ID,
		Workspace:               workspaceID,
		Name:                    account.Name,
		Status:                  account.Status.String(),
		UserCount:               account.UserCount,
		DuplicateRecordPercent:  account.DuplicateRecordPercent,
		Countries:               json.Value(bytes.Clone(account.Countries)),
		GenerationPolicyVersion: account.GenerationPolicyVersion,
		GeneratedRecordCount:    account.GeneratedRecordCount,
		GenerationError:         account.GenerationError,
		CreatedAt:               account.CreatedAt,
		UpdatedAt:               account.UpdatedAt,
	}
}

func simulatedAccountWarehouseError(err error) error {
	if err == nil {
		return nil
	}
	if err == datastore.ErrInspectionMode {
		return errors.Unprocessable(InspectionMode, "data warehouse is in inspection mode")
	}
	if err == datastore.ErrMaintenanceMode {
		return errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
	}
	return errors.Unavailable("simulated account warehouse operation failed")
}

func validateSimulatedAccountConfiguration(userCount, duplicateRecordPercent, countries json.Value) (int, decimal.Decimal, json.Value, error) {
	if userCount == nil {
		return 0, decimal.Decimal{}, nil, fmt.Errorf("userCount is required")
	}
	count, err := userCount.Int()
	if err != nil {
		return 0, decimal.Decimal{}, nil, fmt.Errorf("userCount must be an integer")
	}
	if count < 0 || count > simulatedAccountMaxUserCount {
		return 0, decimal.Decimal{}, nil, fmt.Errorf("userCount must be between 0 and %d", simulatedAccountMaxUserCount)
	}

	duplicate := decimal.Decimal{}
	if duplicateRecordPercent != nil {
		if duplicateRecordPercent.IsNull() {
			return 0, decimal.Decimal{}, nil, fmt.Errorf("duplicateRecordPercent must be a number")
		}
		duplicate, err = duplicateRecordPercent.Decimal(5, 2)
		if err != nil {
			return 0, decimal.Decimal{}, nil, fmt.Errorf("duplicateRecordPercent is not valid")
		}
	}
	if duplicate.Sign() < 0 || duplicate.Greater(decimal.MustInt(50)) {
		return 0, decimal.Decimal{}, nil, fmt.Errorf("duplicateRecordPercent must be between 0 and 50")
	}
	if count == 0 && duplicate.Sign() != 0 {
		return 0, decimal.Decimal{}, nil, fmt.Errorf("duplicateRecordPercent must be 0 for an empty account")
	}

	if countries == nil {
		countries = json.Value("{}")
	}
	if !countries.IsObject() {
		return 0, decimal.Decimal{}, nil, fmt.Errorf("countries must be an object")
	}
	hasCountry := false
	for country, weight := range countries.Properties() {
		hasCountry = true
		if country != "IT" {
			return 0, decimal.Decimal{}, nil, fmt.Errorf("country %q is not supported", country)
		}
		value, err := weight.Int()
		if err != nil || value < 1 || value > 100 {
			return 0, decimal.Decimal{}, nil, fmt.Errorf("country %q weight must be an integer between 1 and 100", country)
		}
	}
	if count > 0 && !hasCountry {
		return 0, decimal.Decimal{}, nil, fmt.Errorf("countries must contain IT for a nonempty account")
	}
	return count, duplicate, json.Value(bytes.Clone(countries)), nil
}
