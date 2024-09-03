//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/apis/connectors"
	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/schemas"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/apis/statistics"
	"github.com/meergo/meergo/apis/transformers"
	"github.com/meergo/meergo/apis/transformers/mappings"
	"github.com/meergo/meergo/types"
)

// exportUsers exports the users for the action.
// The action must have a store.
func (this *Action) exportUsers(ctx context.Context, stats *statistics.Collector) error {

	action := this.action
	store := this.connection.store
	connector := action.Connection().Connector()

	var matching *datastore.Matching
	var internalMatchingProperty types.Property
	if connector.Type == state.App {
		// Synchronize destinations users with the app users.
		err := this.syncDestinationUsers(ctx)
		if err != nil {
			if err, ok := err.(*schemas.Error); ok {
				err.Msg = "in the app matching property, " + err.Msg + ". Please review and update the action before attempting to export the users."
			}
			return newActionError(statistics.OutputValidationStep, err)
		}
		internalMatchingProperty, _ = action.InSchema.Property(action.MatchingProperties.Internal)
		p := action.MatchingProperties.External
		matching = &datastore.Matching{
			Action:          action.ID,
			Property:        p.Name,
			ExportMode:      *this.action.ExportMode,
			AllowDuplicates: *action.ExportOnDuplicatedUsers,
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

	// Read the users.
	records, err := store.UserRecords(ctx, datastore.Query{
		Where:   where,
		OrderBy: orderBy,
	}, action.InSchema, matching)
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return newActionError(statistics.ReceivingStep, err)
		}
		switch err := err.(type) {
		case *datastore.DataWarehouseError:
			// TODO(marco): log the error in a log specific of the workspace.
			ws := action.Connection().Workspace()
			slog.Error("cannot get users from the data warehouse", "workspace", ws.ID, "err", err)
			return err
		case *schemas.Error:
			err.Msg = fmt.Sprintf("in the input schema, %s. Please review and update the action before attempting to export the users.", err.Msg)
			return newActionError(statistics.InputValidationStep, err)
		}
		return err
	}
	defer records.Close()

	var writer connectors.Writer

	// nonExistentUsers contains the destination users that no longer exist in the app.
	var nonExistentUsers []string

	ack := func(ids []string, err error) {
		for _, id := range ids {
			if err != nil && err != connectors.ErrRecordNotExist {
				stats.FailedFinalizing(1, err.Error())
				continue
			}
			if err == connectors.ErrRecordNotExist {
				nonExistentUsers = append(nonExistentUsers, id)
			}
			stats.PassedFinalizing(1)
		}
	}

	// Get the writer.
	switch connector.Type {
	case state.App:
		writer, err = this.app().Writer(ctx, action, ack)
	case state.Database:
		writer, err = this.database().Writer(ctx, action, ack)
	case state.FileStorage:
		replacer := newPathPlaceholderReplacer(time.Now().UTC())
		writer, err = this.file().Writer(ctx, replacer, ack)
		if err, ok := err.(connectors.PlaceholderError); ok {
			return fmt.Errorf("invalid file path: %s", err)
		}
	}
	if err != nil {
		if err, ok := err.(connectors.PlaceholderError); ok {
			return fmt.Errorf("invalid file path: %s", err)
		}
		if err, ok := err.(*schemas.Error); ok {
			err.Msg = "in the output schema, " + err.Msg + ". Please review and update the action before attempting to export the users."
		}
		return newActionError(statistics.OutputValidationStep, err)
	}
	defer writer.Close(ctx)

	// User represents a user to update or create.
	type User struct {
		ID            string           // External app identifier; is non-empty only for app users to update.
		Record        datastore.Record // User record.
		MatchingValue any              // External matching property value added to properties when creating an app user.
	}

	users := make([]User, 0, 100)
	transformationRecords := make([]transformers.Record, 0, 100)

	for record := range records.All(ctx) {

		if record.Err != nil {
			stats.FailedReceiving(1, record.Err.Error())
			if connector.Type == state.FileStorage {
				return record.Err
			}
			goto Next
		}

		stats.PassedReceiving(1)
		stats.PassedInputValidation(1)

		if connector.Type == state.App {
			if record.MatchingID == "" {
				// Create the user.
				value := record.Properties[action.MatchingProperties.Internal]
				value, err = internalToExternalMatchingProperty(value, internalMatchingProperty, action.MatchingProperties.External)
				if err != nil {
					return err
				}
				// The matching property and its value will be added to the properties after the transformation.
				users = append(users, User{Record: record, MatchingValue: value})
			} else {
				// Update the user.
				users = append(users, User{ID: record.MatchingID, Record: record})
			}
		} else {
			// Create the user.
			users = append(users, User{Record: record})
		}

	Next:

		// Does a bach processing of users.
		if len(users) == 100 || records.Last() {

			if transformer == nil {
				for _, user := range users {
					if user.MatchingValue != nil {
						record.Properties[action.MatchingProperties.External.Name] = user.MatchingValue
					}
					if ok := writer.Write(ctx, "", user.Record.Properties); !ok {
						return writer.Close(ctx)
					}
				}
				clear(users)
				users = users[0:0]
				continue
			}

			// Transform the users.
			clear(transformationRecords)
			transformationRecords = transformationRecords[0:len(users)]
			for i, user := range users {
				purpose := transformers.Create
				if user.ID != "" {
					purpose = transformers.Update
				}
				transformationRecords[i].Purpose = purpose
				transformationRecords[i].Properties = user.Record.Properties
			}
			err := transformer.Transform(ctx, transformationRecords)
			if err != nil {
				if err, ok := err.(transformers.FunctionExecutionError); ok {
					return newActionError(statistics.TransformationStep, err)
				}
				return err
			}
			for i, record := range transformationRecords {
				user := users[i]
				if record.Err != nil {
					if _, ok := record.Err.(ValidationError); ok {
						stats.PassedTransformation(1)
						stats.FailedOutputValidation(1, record.Err.Error())
						continue
					}
					stats.FailedTransformation(1, record.Err.Error())
					continue
				}
				stats.PassedTransformation(1)
				stats.PassedOutputValidation(1)
				if user.MatchingValue != nil {
					record.Properties[action.MatchingProperties.External.Name] = user.MatchingValue
				}
				if len(record.Properties) == 0 {
					stats.PassedFinalizing(1)
					continue
				}
				if ok := writer.Write(ctx, user.ID, record.Properties); !ok {
					return writer.Close(ctx)
				}
			}
			clear(users)
			users = users[0:0]

		}

	}
	if err = records.Err(); err != nil {
		return newActionError(statistics.ReceivingStep, err)
	}

	users = nil

	if writer2, ok := writer.(connectors.CommittableWriter); ok {
		err = writer2.Commit(ctx)
	} else {
		err = writer.Close(ctx)
	}
	if err != nil {
		return newActionError(statistics.FinalizingStep, err)
	}

	if nonExistentUsers != nil {
		err = this.connection.store.MergeDestinationUsers(ctx, this.action.ID, nil, nonExistentUsers)
	}

	return err
}

// syncDestinationUsers syncs the destination users of the action.
func (this *Action) syncDestinationUsers(ctx context.Context) error {

	execution, _ := this.action.Execution()
	cursor := execution.Cursor

	// Delete the outdated destination users.
	if execution.Reload {
		store := this.connection.store
		err := store.DeleteDestinationUsers(ctx, this.action.ID)
		if err != nil {
			return err
		}
	}

	// Create a schema with only the matching property.
	externalProp := this.action.MatchingProperties.External
	schema := types.Object([]types.Property{externalProp})

	records, err := this.app().Users(ctx, schema, cursor)
	if err != nil {
		return err
	}
	defer records.Close()

	var users []datastore.DestinationUser

	// Importing users from a destination to match identities for the export.
	for user := range records.All(ctx) {

		if user.Err != nil {
			return user.Err
		}

		// Store the user only if it has a matching external property, and it is not nil.
		v, ok := user.Properties[externalProp.Name]
		if ok && v != nil {
			users = append(users, datastore.DestinationUser{
				User:     user.ID,
				Property: matchingPropertyToString(v),
			})
		}

		cursor = user.LastChangeTime

		if len(users) > 0 && (len(users) == 10000 || records.Last()) {
			// Merge destination users.
			err = this.connection.store.MergeDestinationUsers(ctx, this.action.ID, users, nil)
			if err != nil {
				return err
			}
			// Set the user cursor.
			err = this.setExecutionCursor(ctx, cursor)
			if err != nil {
				return err
			}
			users = users[0:0]
		}

	}
	if err = records.Err(); err != nil {
		return err
	}

	return nil
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

// matchingPropertyToString returns the string representation of a value for a
// matching property.
// v cannot be nil.
func matchingPropertyToString(v any) string {
	switch v := v.(type) {
	case int: // Int(n)
		return strconv.Itoa(v)
	case uint: // Uint(n)
		return strconv.FormatUint(uint64(v), 10)
	case string: // Text and UUID
		return v
	default:
		panic(fmt.Sprintf("unexpected matching property value with type %T", v))
	}
}

// internalToExternalMatchingProperty returns the value to be written to the
// external matching property of an app during users export, in case of
// creation.
//
// internal is the value of the internal matching property, while internalProp
// and externalProp are, respectively, the properties of the internal and the
// external matching property.
//
// Any returned error is an internal error.
func internalToExternalMatchingProperty(internal any, internalProp, externalProp types.Property) (any, error) {
	// TODO(Gianluca): this implementation requires to instantiate every time a
	// new 'mappings.Mapping', this is because we preferred to implement a
	// solution that requires less changes, at the moment, since the issue
	// https://github.com/meergo/meergo/issues/935 will require a deeper review
	// of the code structure of the export to app.
	expressions := map[string]string{externalProp.Name: internalProp.Name}
	inSchema := types.Object([]types.Property{internalProp})
	outSchema := types.Object([]types.Property{externalProp})
	m, err := mappings.New(expressions, inSchema, outSchema, nil)
	if err != nil {
		return nil, err
	}
	out, err := m.Transform(map[string]any{internalProp.Name: internal}, mappings.Create)
	if err != nil {
		return nil, err
	}
	return out[externalProp.Name], nil
}
