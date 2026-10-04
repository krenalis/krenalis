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
	simulatedSystemMaxUserCount  = 10_000_000
	simulatedSystemPolicyVersion = "generation-v3"
)

// SimulatedSystem represents the metadata of a simulated system.
type SimulatedSystem struct {
	ID                      string          `json:"id"`
	Workspace               string          `json:"workspace"`
	Name                    string          `json:"name"`
	Status                  string          `json:"status"`
	UserCount               int             `json:"userCount"`
	DuplicateRecordPercent  decimal.Decimal `json:"duplicateRecordPercent"`
	Countries               json.Value      `json:"countries"`
	GenerationPolicyVersion string          `json:"-"`
	GeneratedRecordCount    int             `json:"generatedRecordCount"`
	GenerationError         string          `json:"-"`
	SourceConnection        *string         `json:"sourceConnection"`
	DestinationConnection   *string         `json:"destinationConnection"`
	CreatedAt               time.Time       `json:"createdAt"`
	UpdatedAt               time.Time       `json:"updatedAt"`
}

// CreateSimulatedSystem creates a simulated system in the workspace.
//
// It returns an errors.BadRequestError for invalid configuration and an
// errors.UnavailableError when preparation storage is unavailable.
//
// It returns an errors.UnprocessableError error with code:
//
//   - SimulatedSystemsRequireDevelopment, if the workspace is not in
//     development.
//   - MaintenanceMode, if the warehouse is in maintenance mode.
func (this *Workspace) CreateSimulatedSystem(ctx context.Context, name string, userCount, duplicateRecordPercent, countries json.Value) (string, error) {
	this.core.mustBeOpen()
	if this.workspace.Environment != state.Development {
		return "", errors.Unprocessable(SimulatedSystemsRequireDevelopment, "simulated systems are only available in development workspaces")
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return "", errors.BadRequest("%s", err)
	}
	userCountValue, duplicateValue, countriesValue, err := validateSimulatedSystemConfiguration(userCount, duplicateRecordPercent, countries)
	if err != nil {
		return "", errors.BadRequest("%s", err)
	}
	status := state.SimulatedSystemAvailable
	policy := "empty-v1"
	var checkpoint json.Value
	var checkpointParameter any
	if userCountValue > 0 {
		if this.core.simulatedCatalog == nil {
			return "", errors.Unavailable("simulated system photo catalog is unavailable")
		}
		status = state.SimulatedSystemPreparing
		policy = simulatedSystemPolicyVersion
		checkpoint, err = initialSimulatedSystemCheckpoint(this.core.simulatedCatalog, this.core.simulatedPhotoBaseURL)
		if err != nil {
			return "", fmt.Errorf("cannot prepare simulated system checkpoint")
		}
		checkpointParameter = string(checkpoint)
	}
	_, err = this.store.CountSimulatedSystemRecords(ctx, "")
	if err != nil {
		return "", simulatedSystemWarehouseError(err)
	}
	n := state.CreateSimulatedSystem{
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
		n.ID = generateID(this.core.state.SimulatedSystem)
		err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			_, err := tx.Exec(ctx, "INSERT INTO simulated_systems (id, workspace, name, status, user_count,"+
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
			if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "simulated_systems_pkey" {
				continue
			}
			return "", err
		}
		if status == state.SimulatedSystemPreparing {
			this.core.wakeSimulatedSystemWorker()
		}
		return n.ID, nil
	}
}

// DeleteSimulatedSystem marks a simulated system as Deleting.
//
// It returns an errors.BadRequestError for an invalid identifier and an
// errors.NotFoundError error if the account does not exist. It returns an
// errors.UnprocessableError with code SimulatedSystemPreparing if the account
// is still being prepared or SimulatedSystemInUse if it is referenced by a
// connection. The status change is committed before this method returns;
// warehouse cleanup continues asynchronously.
func (this *Workspace) DeleteSimulatedSystem(ctx context.Context, id string) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("simulated system identifier %q is not valid", id)
	}
	n := state.DeleteSimulatedSystem{Workspace: this.workspace.ID, ID: id}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		var status state.SimulatedSystemStatus
		err := tx.QueryRow(ctx, "SELECT status FROM simulated_systems WHERE id = $1 AND workspace = $2 FOR UPDATE",
			n.ID, n.Workspace).Scan(&status)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errors.NotFound("simulated system %s does not exist", n.ID)
			}
			return nil, err
		}
		switch status {
		case state.SimulatedSystemPreparing:
			return nil, errors.Unprocessable(SimulatedSystemPreparing, "simulated system %s is still being prepared", n.ID)
		case state.SimulatedSystemAvailable, state.SimulatedSystemFailed:
		case state.SimulatedSystemDeleting:
			return nil, nil
		default:
			return nil, fmt.Errorf("simulated system %s has an invalid status", n.ID)
		}
		var connectionID string
		err = tx.QueryRow(ctx, "SELECT id FROM connections WHERE workspace = $1 AND simulated_system = $2 "+
			"ORDER BY id LIMIT 1",
			n.Workspace, n.ID).Scan(&connectionID)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if err == nil {
			return nil, errors.Unprocessable(SimulatedSystemInUse,
				"simulated system %s is referenced by connection %s", n.ID, connectionID)
		}
		now := time.Now().UTC()
		_, err = tx.Exec(ctx, "UPDATE simulated_systems SET status = $1, updated_at = $2 WHERE id = $3 AND workspace = $4",
			state.SimulatedSystemDeleting, now, n.ID, n.Workspace)
		if err != nil {
			return nil, err
		}
		return state.MarkSimulatedSystemDeleting{
			Workspace: n.Workspace,
			ID:        n.ID,
			UpdatedAt: now,
		}, nil
	})
	if err == nil {
		this.core.wakeSimulatedSystemWorker()
	}
	return err
}

// SimulatedSystem returns a simulated system of the workspace.
// It returns an errors.BadRequestError for an invalid identifier and an
// errors.NotFoundError if the account does not exist.
func (this *Workspace) SimulatedSystem(id string) (*SimulatedSystem, error) {

	this.core.mustBeOpen()

	if !IsValidID(id) {
		return nil, errors.BadRequest("simulated system identifier %q is not valid", id)
	}
	account, ok := this.workspace.SimulatedSystem(id)
	if !ok {
		return nil, errors.NotFound("simulated system %s does not exist", id)
	}

	result := simulatedSystemFromState(account)
	for _, connection := range this.workspace.Connections() {
		boundAccount, ok := connection.SimulatedSystem()
		if !ok || boundAccount.ID != id {
			continue
		}
		connectionID := connection.ID
		switch connection.Role {
		case state.Source:
			result.SourceConnection = &connectionID
		case state.Destination:
			result.DestinationConnection = &connectionID
		}
	}

	return result, nil
}

// SimulatedSystems returns the simulated systems of the workspace, ordered
// by name and then identifier.
func (this *Workspace) SimulatedSystems() []*SimulatedSystem {

	this.core.mustBeOpen()

	accounts := this.workspace.SimulatedSystems()
	result := make([]*SimulatedSystem, 0, len(accounts))
	byID := make(map[string]*SimulatedSystem, len(accounts))
	for _, account := range accounts {
		item := simulatedSystemFromState(account)
		result = append(result, item)
		byID[account.ID] = item
	}

	for _, connection := range this.workspace.Connections() {
		boundAccount, ok := connection.SimulatedSystem()
		if !ok {
			continue
		}
		item := byID[boundAccount.ID]
		if item == nil {
			continue
		}
		connectionID := connection.ID
		switch connection.Role {
		case state.Source:
			item.SourceConnection = &connectionID
		case state.Destination:
			item.DestinationConnection = &connectionID
		}
	}

	return result
}

// RenameSimulatedSystem changes the name of a simulated system.
// It returns an errors.BadRequestError for invalid input and an
// errors.NotFoundError if the account does not exist.
func (this *Workspace) RenameSimulatedSystem(ctx context.Context, id, name string) error {
	this.core.mustBeOpen()
	if !IsValidID(id) {
		return errors.BadRequest("simulated system identifier %q is not valid", id)
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	account, ok := this.workspace.SimulatedSystem(id)
	if !ok || account.Status == state.SimulatedSystemDeleting {
		return errors.NotFound("simulated system %s does not exist", id)
	}
	if account.Name == name {
		return nil
	}
	n := state.UpdateSimulatedSystem{Workspace: this.workspace.ID, ID: id, Name: name, UpdatedAt: time.Now().UTC()}
	return this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE simulated_systems SET name = $1, updated_at = $2 WHERE id = $3 AND workspace = $4"+
			" AND status <> 'Deleting'",
			n.Name, n.UpdatedAt, n.ID, n.Workspace)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("simulated system %s does not exist", n.ID)
		}
		return n, nil
	})
}

func simulatedSystemFromState(account *state.SimulatedSystem) *SimulatedSystem {
	workspace := account.Workspace()
	workspaceID := ""
	if workspace != nil {
		workspaceID = workspace.ID
	}
	return &SimulatedSystem{
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

func simulatedSystemWarehouseError(err error) error {
	if err == nil {
		return nil
	}
	if err == datastore.ErrInspectionMode {
		return errors.Unprocessable(InspectionMode, "data warehouse is in inspection mode")
	}
	if err == datastore.ErrMaintenanceMode {
		return errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
	}
	return errors.Unavailable("simulated system warehouse operation failed")
}

func validateSimulatedSystemConfiguration(userCount, duplicateRecordPercent, countries json.Value) (int, decimal.Decimal, json.Value, error) {
	if userCount == nil {
		return 0, decimal.Decimal{}, nil, fmt.Errorf("userCount is required")
	}
	count, err := userCount.Int()
	if err != nil {
		return 0, decimal.Decimal{}, nil, fmt.Errorf("userCount must be an integer")
	}
	if count < 0 || count > simulatedSystemMaxUserCount {
		return 0, decimal.Decimal{}, nil, fmt.Errorf("userCount must be between 0 and %d", simulatedSystemMaxUserCount)
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
