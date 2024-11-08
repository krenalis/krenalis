//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package core

import (
	"context"
	"log/slog"

	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

// User represents a user.
type User struct {
	core      *Core
	workspace *state.Workspace
	store     *datastore.Store
	id        uuid.UUID
}

// Identities returns the user identities of the user, and an estimate of their
// count without applying first and limit.
//
// It returns the user identities in range [first,first+limit] with first >= 0
// and 0 < limit <= 1000.
//
// It returns an errors.NotFoundError error, if the user does not exist.
// It returns an errors.UnprocessableError error with code
//
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
func (this *User) Identities(ctx context.Context, first, limit int) ([]UserIdentity, int, error) {
	this.core.mustBeOpen()
	if first < 0 {
		return nil, 0, errors.BadRequest("first %d is not valid", limit)
	}
	if limit < 1 || limit > 1000 {
		return nil, 0, errors.BadRequest("limit %d is not valid", limit)
	}
	where := &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
		Property: []string{"__gid__"},
		Operator: state.OpIs,
		Values:   []any{this.id.String()},
	}}}
	ws := &Workspace{
		core:      this.core,
		store:     this.store,
		workspace: this.workspace,
	}
	identities, count, err := ws.userIdentities(ctx, where, first, limit)
	if err != nil {
		return nil, 0, err
	}
	if identities == nil {
		return nil, 0, errors.NotFound("user %s does not exist", this.id)
	}
	return identities, count, nil
}

// Traits returns the traits of the user.
//
// It returns an errors.NotFoundError error, if the user does not exist.
// It returns an errors.UnprocessableError error with code
//
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
func (this *User) Traits(ctx context.Context) ([]byte, error) {

	this.core.mustBeOpen()

	ws := this.workspace

	properties := types.PropertyNames(this.workspace.UserSchema)
	where := &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
		Property: []string{"__id__"},
		Operator: state.OpIs,
		Values:   []any{this.id.String()},
	}}}

	// Retrieve the user traits.
	records, _, err := this.store.Users(ctx, datastore.Query{
		Properties: properties,
		Where:      where,
		Limit:      1,
	})
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			slog.Error("cannot get users from the data warehouse", "workspace", ws.ID, "err", err)
			return nil, errors.Unprocessable(DataWarehouseFailed, "data warehouse connection is failed: %w", err.Err)
		}
		return nil, err
	}
	if len(records) == 0 {
		return nil, errors.NotFound("user %s does not exist", this.id)
	}

	return json.MarshalBySchema(records[0], ws.UserSchema)
}
