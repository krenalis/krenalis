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

	"github.com/meergo/meergo/apis/connectors"
	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/apis/statistics"
	"github.com/meergo/meergo/apis/transformers"
	"github.com/meergo/meergo/types"
)

// exportUsers exports the users for the action.
// The action must have a store.
func (this *Action) exportUsers(ctx context.Context, stats *statistics.ActionCollector) error {

	action := this.action
	store := this.connection.store
	connector := action.Connection().Connector()

	if connector.Type == state.App {
		// Download the users from this connection to match the identities for the export.
		err := this.downloadUsersForExportMatch(ctx)
		if err != nil {
			if err, ok := err.(*connectors.SchemaError); ok {
				err.Msg = "in the app matching property, " + err.Msg + ". Please review and update the action before attempting to export the users."
			}
			return actionExecutionError{fmt.Errorf("cannot retrieve users information from app: %s", err)}
		}
		// If the export must be blocked in case of duplicated user on the
		// destination, check if there are duplicated users on the destination.
		if !*action.ExportOnDuplicatedUsers {
			u1, u2, ok, err := store.DuplicatedDestinationUsers(ctx, action.ID)
			if err != nil {
				if err == datastore.ErrMaintenanceMode {
					return actionExecutionError{err}
				}
				return actionExecutionError{fmt.Errorf("cannot look for duplicated destination users: %s", err)}
			}
			if ok {
				return actionExecutionError{fmt.Errorf("there are two users on the connection (%q and %q)"+
					" with the same value for the external matching property %q",
					u1, u2, action.MatchingProperties.External.Name)}
			}
		}
		// Check if there are duplicated users within Meergo.
		{
			u1, u2, ok, err := store.DuplicatedUsers(ctx, action.MatchingProperties.Internal)
			if err != nil {
				if err == datastore.ErrMaintenanceMode {
					return actionExecutionError{err}
				}
				return actionExecutionError{fmt.Errorf("cannot look for duplicated users on data warehouse: %s", err)}
			}
			if ok {
				return actionExecutionError{fmt.Errorf("there are two users (%s and %s)"+
					" with the same value for the internal matching property %q",
					u1, u2, action.MatchingProperties.Internal)}
			}
		}
	}

	// Get the transformer.
	var transformer *transformers.Transformer
	if t := this.action.Transformation; t.Mapping != nil || t.Function != nil {
		var err error
		transformer, err = transformers.New(action, this.apis.transformerProvider, &connector.TimeLayouts)
		if err != nil {
			return err
		}
	}

	// Determine the "order by" property.
	var orderBy string
	if action.Connection().Connector().Type == state.FileStorage {
		orderBy = action.FileOrderingPropertyPath
	} else {
		// For any other type of connector other than FileStorage, don't order
		// the results.
	}

	// Where condition.
	var where *datastore.Where
	if action.Filter != nil {
		where = &datastore.Where{
			Logical:    datastore.WhereLogical(action.Filter.Logical),
			Conditions: make([]datastore.WhereCondition, len(action.Filter.Conditions)),
		}
		for i, condition := range action.Filter.Conditions {
			where.Conditions[i] = (datastore.WhereCondition)(condition)
		}
	}

	io := "input"
	schema := action.InSchema
	if connector.Type == state.FileStorage {
		io = "output"
		schema = action.OutSchema
	}

	// Read the users.
	records, err := store.UserRecords(ctx, datastore.Query{
		Properties: types.PropertyNames(schema),
		Where:      where,
		OrderBy:    orderBy,
	}, action.Connection().Workspace().UserSchema)
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return actionExecutionError{err}
		}
		switch err := err.(type) {
		case *datastore.DataWarehouseError:
			// TODO(marco): log the error in a log specific of the workspace.
			ws := action.Connection().Workspace()
			slog.Error("cannot get users from the data warehouse", "workspace", ws.ID, "err", err)
			return err
		case *connectors.SchemaError:
			err.Msg = fmt.Sprintf("in the %s schema, %s. Please review and update the action before attempting to export the users.", io, err.Msg)
			return actionExecutionError{err}
		}
		return err
	}
	defer records.Close()

	var writer connectors.Writer

	ack := func(ids []string, err error) {
		for range ids {
			if err != nil {
				stats.Failed(statistics.Finalizing, err.Error())
				continue
			}
			stats.Passed(statistics.Finalizing)
		}
	}

	// Get the writer.
	switch connector.Type {
	case state.App:
		writer, err = this.app().Writer(state.Users, ack)
	case state.Database:
		writer, err = this.database().Writer(action.TableName, action.TableKeyProperty, action.OutSchema, ack)
	case state.FileStorage:
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

	// User represents a user to update or create.
	type User struct {
		ID     string           // External app identifier; is non-empty only for app users to update.
		Record datastore.Record // User record.
	}

	var (
		users  = make([]User, 0, 100)
		values = make([]map[string]any, 0, 100)
	)

	for record := range records.All(ctx) {

		if record.Err != nil {
			stats.Failed(statistics.Receiving, record.Err.Error())
			if connector.Type == state.FileStorage {
				return record.Err
			}
			goto Next
		}

		stats.Passed(statistics.Receiving)
		stats.Passed(statistics.InputValidation)

		if connector.Type == state.App {
			// Resolve the external identities.
			ids, err := this.resolveExternalIdentities(ctx, record)
			if err != nil {
				if err == datastore.ErrMaintenanceMode {
					return actionExecutionError{err}
				}
				return err
			}
			// Determine if this user must be exported or not.
			mode := *this.action.ExportMode
			existsOnApp := len(ids) > 0
			if (mode == state.CreateOnly && existsOnApp) || (mode == state.UpdateOnly && !existsOnApp) {
				goto Next
			}
			if existsOnApp {
				// Update the user(s).
				for _, id := range ids {
					users = append(users, User{ID: id, Record: record})
				}
			} else {
				// Create the user.
				users = append(users, User{Record: record})
			}
		} else {
			users = append(users, User{Record: record})
		}

	Next:

		// Does a bach processing of users.
		if len(users) == 100 || records.Last() {

			if transformer == nil {
				for _, user := range users {
					if ok := writer.Write(ctx, "", user.Record.Properties, user.Record.ID.(string)); !ok {
						return writer.Close(ctx)
					}
				}
				clear(users)
				users = users[0:0]
				continue
			}

			// Transform the users.
			clear(values)
			values = values[0:len(users)]
			for i, user := range users {
				values[i] = user.Record.Properties
			}
			results, err := transformer.Transform(ctx, values)
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
						stats.Passed(statistics.Transformation)
						stats.Failed(statistics.OutputValidation, result.Err.Error())
						continue
					}
					stats.Failed(statistics.Transformation, result.Err.Error())
					continue
				}
				stats.Passed(statistics.Transformation)
				stats.Passed(statistics.OutputValidation)
				if len(result.Value) == 0 {
					continue
				}
				if ok := writer.Write(ctx, user.ID, result.Value, user.Record.ID.(string)); !ok {
					return writer.Close(ctx)
				}
			}
			clear(users)
			users = users[0:0]

		}

	}
	if err = records.Err(); err != nil {
		return actionExecutionError{err}
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

	records, err := this.app().Users(ctx, schema, time.Time{})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
	}
	defer records.Close()

	// Importing users from a destination to match identities for the export.
	for user := range records.All(ctx) {

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
		err = this.setUserCursor(ctx, user.LastChangeTime)
		if err != nil {
			return actionExecutionError{err}
		}

	}
	if err = records.Err(); err != nil {
		return actionExecutionError{fmt.Errorf("an error occurred closing the database: %s", err)}
	}

	return nil
}

// resolveExternalIdentities resolves the external identities of the record and
// returns its external app identifiers, if resolved, or the empty slice if such
// user does not exist on the remote app.
//
// If the data warehouse is in maintenance mode, it returns the
// datastore.ErrMaintenanceMode error.
func (this *Action) resolveExternalIdentities(ctx context.Context, record datastore.Record) ([]string, error) {
	internalPropName := this.action.MatchingProperties.Internal
	property, ok := record.Properties[internalPropName]
	if !ok {
		property = nil
	}
	p, err := json.Marshal(property)
	if err != nil {
		return nil, err
	}
	c := this.connection
	ids, err := c.store.DestinationUsers(ctx, this.action.ID, string(p))
	if err != nil {
		return nil, err
	}
	return ids, nil
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
