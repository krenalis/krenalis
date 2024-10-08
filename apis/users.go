//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"log/slog"

	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

// User represents a user.
type User struct {
	apis      *APIs
	workspace *state.Workspace
	store     *datastore.Store
	id        uuid.UUID
}

// eventProperties contains the properties of the events as returned by the
// (*User).Events method.
var eventProperties []string

func init() {
	eventProperties = make([]string, types.NumProperties(datastore.EventSchema)-1)
	i := 0
	for _, p := range datastore.EventSchema.Properties() {
		if p.Name == "user" {
			continue
		}
		eventProperties[i] = p.Name
		i++
	}
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
func (this *User) Events(ctx context.Context, limit int) ([]byte, error) {

	this.apis.mustBeOpen()

	if limit < 1 || limit > 200 {
		return nil, errors.BadRequest("limit %d is not valid", limit)
	}

	// Retrieve the events records.
	evs, err := this.store.Events(ctx, datastore.Query{
		Properties: eventProperties,
		Where: &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
			Property: "user",
			Operator: state.OpIs,
			Values:   []any{this.id.String()},
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
			Where: &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
				Property: "__id__",
				Operator: state.OpIs,
				Values:   []any{this.id.String()},
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

	es := make([]any, len(evs))
	for i, e := range evs {
		es[i] = e
	}

	return json.MarshalBySchema(es, types.Array(datastore.EventSchema))
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
	this.apis.mustBeOpen()
	if first < 0 {
		return nil, 0, errors.BadRequest("first %d is not valid", limit)
	}
	if limit < 1 || limit > 1000 {
		return nil, 0, errors.BadRequest("limit %d is not valid", limit)
	}
	where := &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
		Property: "__gid__",
		Operator: state.OpIs,
		Values:   []any{this.id.String()},
	}}}
	ws := &Workspace{
		apis:      this.apis,
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

	this.apis.mustBeOpen()

	ws := this.workspace

	properties := types.PropertyNames(this.workspace.UserSchema)
	where := &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
		Property: "__id__",
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
