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

	"chichi/apis/datastore"
	"chichi/apis/datastore/expr"
	"chichi/apis/datastore/warehouses"
	"chichi/apis/encoding"
	"chichi/apis/errors"
	"chichi/apis/events/eventschema"
	"chichi/apis/state"
	"chichi/connector/types"
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
// It returns an errors.UnprocessableError error with code
//
//   - NoEventsSchema, if the data warehouse does not have events schema.
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
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
	schema := eventschema.SchemaWithGID
	properties := schema.Properties()

	// Retrieve the events records.
	propsPaths := make([]types.Path, len(properties))
	for i, p := range properties {
		propsPaths[i] = types.Path{p.Name}
	}
	records, err := this.store.Events(ctx, datastore.EventsQuery{
		Properties: propsPaths,
		Where:      expr.NewBaseExpr("gid", expr.OperatorEqual, this.id),
		Schema:     schema,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	evs := []map[string]any{}
	err = records.For(func(record warehouses.Record) error {
		if record.Err != nil {
			return err
		}
		evs = append(evs, record.Properties)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err = records.Err(); err != nil {
		return nil, err
	}

	return encoding.MarshalSlice(schema, evs)
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
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
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
	where := expr.NewBaseExpr("Gid", expr.OperatorEqual, this.id)
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
//   - NoUsersSchema, if the data warehouse does not have users schema.
//   - NoWarehouse, if the workspace does not have a data warehouse.
func (this *User) Traits(ctx context.Context) ([]byte, error) {

	this.apis.mustBeOpen()

	ws := this.workspace

	// Verify that the workspace has a data warehouse.
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Read the users schema and determine the properties to select.
	schemas, err := this.store.Schemas(ctx)
	if err != nil {
		return nil, err
	}
	usersSchema := schemas["users"]
	if !usersSchema.Valid() {
		return nil, errors.Unprocessable(NoUsersSchema, "missing users schema")
	}
	properties := []types.Path{}
	for _, p := range usersSchema.PropertiesNames() {
		properties = append(properties, types.Path{p})
	}

	// Retrieve the user traits as records.
	records, _, err := this.store.Users(ctx, datastore.UsersQuery{
		Schema:     usersSchema,
		Properties: properties,
		Where:      expr.NewBaseExpr("Id", expr.OperatorEqual, this.id),
		Limit:      1,
	})
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			slog.Error("cannot get users from the data store", "workspace", ws.ID, "err", err)
			return nil, errors.Unprocessable(DataWarehouseFailed, "store connection is failed: %w", err.Err)
		}
		return nil, err
	}
	var traits map[string]any
	err = records.For(func(user warehouses.Record) error {
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

	return encoding.Marshal(usersSchema, traits)
}
