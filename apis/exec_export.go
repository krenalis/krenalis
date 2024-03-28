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
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/datastore"
	"github.com/open2b/chichi/apis/datastore/expr"
	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/statistics"
	"github.com/open2b/chichi/apis/transformers"
	"github.com/open2b/chichi/types"
)

// exportUsers exports the users for the action.
// The action must have a store.
func (this *Action) exportUsers(ctx context.Context) error {

	action := this.action
	store := this.connection.store
	stats := this.apis.statistics.Action(action.ID)
	connector := action.Connection().Connector()

	if connector.Type == state.AppType {
		// Download the users from this connection to match the identities for the export.
		err := this.downloadUsersForExportMatch(ctx)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot retrieve users information from app: %s", err)}
		}
		// If the export must be blocked in case of duplicated user on the
		// destination, check if there are duplicated users on the destination.
		if !*action.ExportOnDuplicatedUsers {
			u1, u2, ok, err := store.DuplicatedDestinationUsers(ctx, action.ID)
			if err != nil {
				return actionExecutionError{fmt.Errorf("cannot look for duplicated destination users: %s", err)}
			}
			if ok {
				return actionExecutionError{fmt.Errorf("there are two users on the connection (%q and %q)"+
					" with the same value for the external matching property %q",
					u1, u2, action.MatchingProperties.External.Name)}
			}
		}
		// Check if there are duplicated users within Chichi.
		{
			u1, u2, ok, err := store.DuplicatedUsers(ctx, action.MatchingProperties.Internal)
			if err != nil {
				return actionExecutionError{fmt.Errorf("cannot look for duplicated users on data warehouse: %s", err)}
			}
			if ok {
				return actionExecutionError{fmt.Errorf("there are two users (%d and %d)"+
					" with the same value for the internal matching property %q",
					u1, u2, action.MatchingProperties.Internal)}
			}
		}
	}

	// Get the transformer.
	var err error
	var transformer *transformers.Transformer
	if tr := this.action.Transformation; tr.Mapping != nil || tr.Function != nil {
		transformer, err = transformers.New(action.InSchema, action.OutSchema, tr, action.ID, this.apis.functionTransformer, &connector.Layouts)
		if err != nil {
			return err
		}
	}

	schema := action.InSchema
	if connector.Type == state.FileStorageType {
		schema = action.OutSchema
	}

	// Determine the properties to select from the data warehouse.
	var properties []types.Path
	if action.Transformation.Mapping != nil {
		properties = transformer.Properties()
		if action.MatchingProperties != nil {
			internal := action.MatchingProperties.Internal
			var found bool
			for _, path := range properties {
				if len(path) == 1 && path[0] == internal {
					found = true
					break
				}
			}
			if !found {
				properties = append(properties, types.Path{internal})
			}
		}
	} else {
		for _, p := range schema.PropertiesNames() {
			properties = append(properties, types.Path{p})
		}
	}

	// Build the where from the filter, if any.
	var where expr.Expr
	if action.Filter != nil {
		where, err = convertActionFilterToExpr(action.Filter, schema)
		if err != nil {
			return err
		}
	}

	// Read the users.
	records, _, err := store.Users(ctx, datastore.UsersQuery{
		Properties: properties,
		Where:      where,
		OrderBy:    types.Property{Name: "Id", Type: types.Int(32)},
		Schema:     schema,
	})
	if err != nil {
		switch err := err.(type) {
		case *datastore.DataWarehouseError:
			// TODO(marco): log the error in a log specific of the workspace.
			ws := action.Connection().Workspace()
			slog.Error("cannot get users from the data warehouse", "workspace", ws.ID, "err", err)
			return errors.Unprocessable(DataWarehouseFailed, "warehouse connection is failed: %w", err.Err)
		case *datastore.SchemaError:
			err.Msg += ". Please review and update the action before attempting to export the users."
			return err
		}
		return err
	}
	defer records.Close()

	var writer connectors.Writer

	ack := func(err error, gids []int) {
		for _, gid := range gids {
			if err != nil {
				stats.Failed(statistics.ExportStep, gid, err)
				continue
			}
			stats.Passed(statistics.ExportStep)
		}
	}

	// Get the writer.
	switch connector.Type {
	case state.AppType:
		writer, err = this.app().Writer(state.Users, ack)
	case state.DatabaseType:
		writer, err = this.database().Writer(action.TableName, action.OutSchema, ack)
	case state.FileStorageType:
		replacer := newPathPlaceholderReplacer(time.Now().UTC())
		writer, err = this.file().Writer(ctx, schema, ack, replacer)
		if err, ok := err.(connectors.PlaceholderError); ok {
			return fmt.Errorf("invalid file path: %s", err)
		}
	}
	if err != nil {
		if err, ok := err.(connectors.PlaceholderError); ok {
			return fmt.Errorf("invalid file path: %s", err)
		}
		return actionExecutionError{err}
	}
	defer writer.Close(ctx)

	type userToProcess struct {
		GID        int
		ID         string
		Properties map[string]any
	}

	var (
		users  = make([]userToProcess, 0, 100)
		values = make([]map[string]any, 0, 100)
	)

	// processUsers does a bach processing of users.
	processUsers := func(users []userToProcess) error {

		if transformer == nil {
			for _, user := range users {
				record := connectors.Record{
					Properties: user.Properties,
				}
				if ok := writer.Write(ctx, user.GID, record); !ok {
					return writer.Close(ctx)
				}
			}
			return nil
		}

		// Transform the users.
		clear(values)
		values = values[0:len(users)]
		for i, user := range users {
			values[i] = user.Properties
		}
		results, err := transformer.TransformValues(ctx, values)
		if err != nil {
			if err, ok := err.(transformers.FunctionExecutionError); ok {
				return actionExecutionError{err}
			}
			return err
		}
		for i, result := range results {
			user := users[i]
			if result.Err != nil {
				if _, ok := result.Err.(ValidationError); ok {
					stats.Passed(statistics.TransformedStep)
					stats.Failed(statistics.OutputValidatedStep, user.GID, err)
					continue
				}
				stats.Failed(statistics.TransformedStep, user.GID, err)
				continue
			}
			user.Properties = result.Value
			stats.Passed(statistics.TransformedStep)
			stats.Passed(statistics.OutputValidatedStep)
			record := connectors.Record{
				ID:         user.ID,
				Properties: user.Properties,
			}
			if ok := writer.Write(ctx, user.GID, record); !ok {
				return writer.Close(ctx)
			}
		}

		return nil
	}

	err = records.For(func(user warehouses.Record) error {
		if user.Err != nil {
			stats.Failed(statistics.ReceivedStep, user.ID, user.Err)
			if connector.Type == state.FileStorageType {
				return err
			}
			return nil
		}
		stats.Passed(statistics.ReceivedStep)
		stats.Passed(statistics.InputValidatedStep)
		var id string
		if connector.Type == state.AppType {
			// Resolve the external identities.
			ids, err := this.resolveExternalIdentities(ctx, user)
			if err != nil {
				return err
			}
			// Determine if this user must be exported or not.
			mode := *this.action.ExportMode
			existsOnApp := len(ids) > 0
			if (mode == state.CreateOnly && existsOnApp) || (mode == state.UpdateOnly && !existsOnApp) {
				return nil
			}
			for _, id := range ids {
				users = append(users, userToProcess{
					GID:        user.ID,
					ID:         id,
					Properties: user.Properties,
				})
			}
		} else {
			users = append(users, userToProcess{
				GID:        user.ID,
				ID:         id,
				Properties: user.Properties,
			})
		}
		if len(users) == 100 {
			err := processUsers(users)
			if err != nil {
				return err
			}
			clear(users)
			users = users[0:0]
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err = records.Err(); err != nil {
		return actionExecutionError{err}
	}

	// Process the remaining users.
	if len(users) > 0 {
		err = processUsers(users)
		if err != nil {
			return err
		}
	}
	users = nil

	if writer2, ok := writer.(connectors.CommittableWriter); ok {
		err = writer2.Commit(ctx)
	} else {
		err = writer.Close(ctx)
	}
	if err != nil {
		return actionExecutionError{err}
	}

	return nil
}

// downloadUsersForExportMatch downloads the users of the external app for the
// matching of the export.
func (this *Action) downloadUsersForExportMatch(ctx context.Context) error {

	// Create a schema with only the matching property.
	externalProp := this.action.MatchingProperties.External
	schema := types.Object([]types.Property{externalProp})

	// TODO(Gianluca): here cursor.Next is set to "" as a workaround. See the
	// issue https://github.com/open2b/chichi/issues/183.
	var cursor state.Cursor

	records, err := this.app().Users(ctx, schema, "", cursor)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
	}
	defer records.Close()

	// Importing users from a destination to match identities for the export.
	err = records.For(func(user connectors.Record) error {

		if user.Err != nil {
			return actionExecutionError{user.Err}
		}

		p, err := json.Marshal(user.Properties[externalProp.Name])
		if err != nil {
			return actionExecutionError{err}
		}
		err = this.connection.store.SetDestinationUser(ctx, this.action.ID, user.ID, string(p))
		if err != nil {
			return actionExecutionError{err}
		}

		// Set the user cursor.
		err = this.setUserCursor(ctx, state.Cursor{ID: user.ID, Timestamp: user.Timestamp})
		if err != nil {
			return actionExecutionError{err}
		}

		return nil
	})
	if err != nil {
		return err
	}
	if err = records.Err(); err != nil {
		return actionExecutionError{fmt.Errorf("an error occurred closing the database: %s", err)}
	}

	return nil
}

// resolveExternalIdentities resolves the external identities of user and
// returns its external IDs, if resolved, or the empty slice if such user does
// not exist on the remote app.
func (this *Action) resolveExternalIdentities(ctx context.Context, user warehouses.Record) ([]string, error) {
	internalPropName := this.action.MatchingProperties.Internal
	property, ok := user.Properties[internalPropName]
	if !ok {
		return nil, fmt.Errorf("property %q not found", internalPropName)
	}
	p, err := json.Marshal(property)
	if err != nil {
		return nil, err
	}
	c := this.connection
	externalIDs, err := c.store.DestinationUsers(ctx, this.action.ID, string(p))
	if err != nil {
		return nil, err
	}
	return externalIDs, nil
}

// newPathPlaceholderReplacer returns a placeholder replacer that replaces the
// following placeholders using time.Now().UTC() as current time.
//
//	${today}  which renders to something like:  2035-10-30
//	${now}    which renders to something like:  2035-10-30-16-33-25
//	${unix}   which renders to something like:  2077374805
//
// These placeholders are case-insensitive, so ${TODAY} is handled like
// ${today}.
func newPathPlaceholderReplacer(t time.Time) func(string) (string, bool) {
	return func(name string) (string, bool) {
		switch strings.ToLower(name) {
		case "today":
			return t.Format(time.DateOnly), true
		case "now":
			return t.Format("2006-01-02-15-04-05"), true
		case "unix":
			return strconv.FormatInt(t.Unix(), 10), true
		}
		return "", false
	}
}
