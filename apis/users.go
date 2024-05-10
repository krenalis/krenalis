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
	"strconv"

	"github.com/open2b/chichi/apis/datastore"
	"github.com/open2b/chichi/apis/encoding"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/events"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"
)

// User represents a user.
type User struct {
	apis      *APIs
	workspace *state.Workspace
	store     *datastore.Store
	id        int
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

	ws := this.workspace

	// Verify that the workspace has a data warehouse.
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Read the event schema's properties.
	schema := events.WarehouseSchema
	properties := schema.Properties()

	// Retrieve the events records.
	propsPaths := make([]types.Path, len(properties))
	for i, p := range properties {
		propsPaths[i] = types.Path{p.Name}
	}
	records, err := this.store.Events(ctx, datastore.EventsQuery{
		Properties: propsPaths,
		Filter: &state.Filter{Logical: "all", Conditions: []state.FilterCondition{{
			Property: "gid",
			Operator: "is",
			Value:    strconv.Itoa(this.id),
		}}},
		Limit: limit,
	})
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		return nil, err
	}

	evs := []map[string]any{}
	err = records.For(func(record datastore.Record) error {
		if record.Err != nil {
			return err
		}
		// Convert "snake_case" property names (used in the data warehouse) to
		// "camelCase" (used in the exposed events).
		convertEventPropertyCase(record.Properties)
		evs = append(evs, record.Properties)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err = records.Err(); err != nil {
		return nil, err
	}
	if len(evs) == 0 {
		// Verify that the user exists.
		ws := &Workspace{
			apis:      this.apis,
			store:     this.store,
			workspace: this.workspace,
		}
		filter := &state.Filter{Logical: "all", Conditions: []state.FilterCondition{{
			Property: "__gid__",
			Operator: "is",
			Value:    strconv.Itoa(this.id),
		}}}
		_, count, err := ws.userIdentities(ctx, filter, 0, 1)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, errors.NotFound("user %d does not exist", this.id)
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
		Value:    strconv.Itoa(this.id),
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
		return nil, 0, errors.NotFound("user %d does not exist", this.id)
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

	// Determine the properties to select.
	properties := []types.Path{}
	for _, p := range ws.UsersSchema.PropertiesNames() {
		properties = append(properties, types.Path{p})
	}

	// Retrieve the user traits as records.
	records, _, err := this.store.Users(ctx, datastore.UsersQuery{
		Properties: properties,
		Filter: &state.Filter{Logical: "all", Conditions: []state.FilterCondition{{
			Property: "__id__",
			Operator: "is",
			Value:    strconv.Itoa(this.id),
		}}},
		Limit: 1,
	}, ws.UsersSchema)
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			slog.Error("cannot get users from the data store", "workspace", ws.ID, "err", err)
			return nil, errors.Unprocessable(DataWarehouseFailed, "store connection is failed: %w", err.Err)
		}
		return nil, err
	}
	var traits map[string]any
	err = records.For(func(user datastore.Record) error {
		if user.Err != nil {
			return err
		}
		traits = user.Properties
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err = records.Err(); err != nil {
		return nil, err
	}
	if traits == nil {
		return nil, errors.NotFound("user %d does not exist", this.id)
	}

	return encoding.Marshal(ws.UsersSchema, traits)
}

// convertEventPropertyCase converts the case of the event property names from
// "snake_case" (used in the data warehouse) to "camelCase" (used in the event
// exposed via HTTP or in the action).
func convertEventPropertyCase(event map[string]any) {

	// Anonymous ID.
	event["anonymousId"] = event["anonymous_id"]
	delete(event, "anonymous_id")

	context := event["context"].(map[string]any)

	// Context > Device.
	device := context["device"].(map[string]any)
	device["advertisingId"] = device["advertising_id"]
	delete(device, "advertising_id")
	device["adTrackingEnabled"] = device["ad_tracking_enabled"]
	delete(device, "ad_tracking_enabled")

	// Context > User agent.
	context["userAgent"] = context["user_agent"]
	delete(context, "user_agent")

	// Group ID.
	event["groupId"] = event["group_id"]
	delete(event, "group_id")

	// Message ID.
	event["messageId"] = event["message_id"]
	delete(event, "message_id")

	// Received at.
	event["receivedAt"] = event["received_at"]
	delete(event, "received_at")

	// User ID.
	event["userId"] = event["user_id"]
	delete(event, "user_id")
}
