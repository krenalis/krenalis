//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/open2b/chichi/apis/datastore"
	"github.com/open2b/chichi/apis/encoding"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/events"
	"github.com/open2b/chichi/apis/state"
)

// User represents a user.
type User struct {
	apis      *APIs
	workspace *state.Workspace
	store     *datastore.Store
	id        uuid.UUID
}

// Events returns the events of the user, ordered from the most recent to the
// oldest. limit is the maximum number of events to return, it must be in range
// [1, 200].
//
// It returns an errors.NotFoundError error, if the user does not exist.
// It returns an errors.UnprocessableError error with code
//
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
//   - NoEventsSchema, if the data warehouse does not have events schema.
//   - NoWarehouse, if the workspace does not have a data warehouse.
func (this *User) Events(ctx context.Context, limit int) ([]byte, error) {

	this.apis.mustBeOpen()

	if limit < 1 || limit > 200 {
		return nil, errors.BadRequest("limit %d is not valid", limit)
	}

	// Verify that the workspace has a data warehouse.
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", this.workspace.ID)
	}

	// Retrieve the events records.
	evs, err := this.store.Events(ctx, datastore.Query{
		Properties: events.Schema.PropertiesNames(),
		Filter: &state.Filter{Logical: "all", Conditions: []state.FilterCondition{{
			Property: "user",
			Operator: "is",
			Value:    this.id.String(),
		}}},
		OrderBy:   "timestamp",
		OrderDesc: true,
		Limit:     limit,
	})
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		return nil, err
	}
	if len(evs) == 0 {
		// Verify that the user exists.
		_, count, err := this.store.Users(ctx, datastore.Query{
			Properties: []string{"__id__"},
			Filter: &state.Filter{Logical: "all", Conditions: []state.FilterCondition{{
				Property: "__id__",
				Operator: "is",
				Value:    this.id.String(),
			}}},
			Limit: 1,
		})
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, errors.NotFound("user %s does not exist", this.id)
		}
	}

	return encoding.MarshalSlice(events.Schema, evs)
}

// Identities returns the users identities of the user, and an estimate of their
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
//   - NoWarehouse, if the workspace does not have a data warehouse.
func (this *User) Identities(ctx context.Context, first, limit int) ([]byte, int, error) {
	this.apis.mustBeOpen()
	if first < 0 {
		return nil, 0, errors.BadRequest("first %d is not valid", limit)
	}
	if limit < 1 || limit > 1000 {
		return nil, 0, errors.BadRequest("limit %d is not valid", limit)
	}
	if this.store == nil {
		return nil, 0, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", this.workspace.ID)
	}
	filter := &state.Filter{Logical: "all", Conditions: []state.FilterCondition{{
		Property: "__gid__",
		Operator: "is",
		Value:    this.id.String(),
	}}}
	ws := &Workspace{
		apis:      this.apis,
		store:     this.store,
		workspace: this.workspace,
	}
	identities, count, err := ws.userIdentities(ctx, filter, first, limit)
	if err != nil {
		return nil, 0, err
	}
	if identities == nil {
		return nil, 0, errors.NotFound("user %s does not exist", this.id)
	}
	data, err := json.Marshal(identities)
	return data, count, err
}

// Traits returns the traits of the user.
//
// It returns an errors.NotFoundError error, if the user does not exist.
// It returns an errors.UnprocessableError error with code
//
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
//   - NoWarehouse, if the workspace does not have a data warehouse.
func (this *User) Traits(ctx context.Context) ([]byte, error) {

	this.apis.mustBeOpen()

	ws := this.workspace

	// Verify that the workspace has a data warehouse.
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	properties := this.workspace.UsersSchema.PropertiesNames()
	filter := &state.Filter{Logical: "all", Conditions: []state.FilterCondition{{
		Property: "__id__",
		Operator: "is",
		Value:    this.id.String(),
	}}}

	// Retrieve the user traits.
	records, _, err := this.store.Users(ctx, datastore.Query{
		Properties: properties,
		Filter:     filter,
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

	return encoding.Marshal(ws.UsersSchema, records[0])
}
