//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package core

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/schemas"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	"github.com/meergo/meergo/core/util"
	meergoMetrics "github.com/meergo/meergo/metrics"
	"github.com/meergo/meergo/types"
)

// exportUsers exports the users for the action.
// The action must have a store.
func (this *Action) exportUsers(ctx context.Context) error {

	action := this.action
	store := this.connection.store
	connector := action.Connection().Connector()
	meergoMetrics.Increment("Action.exportUsers.calls", 1)

	// alreadyExportedKeys keeps track of the keys of users exported to the
	// database during this export, indexed by their table key value (which can
	// have Go type int, uint o string).
	var alreadyExportedKeys map[any]struct{}

	var matching *datastore.Matching
	var matchingIn types.Property
	var matchingOut types.Property
	if connector.Type == state.App {
		// Get the matching properties.
		matchingIn, _ = action.InSchema.Property(action.Matching.In)
		matchingOut, _ = action.OutSchema.Property(action.Matching.Out)
		matching = &datastore.Matching{
			Action:             action.ID,
			InProperty:         matchingIn.Name,
			ExportMode:         this.action.ExportMode,
			UpdateOnDuplicates: action.UpdateOnDuplicates,
		}
		// Synchronize destinations users with the app users.
		err := this.syncDestinationUsers(ctx)
		if err != nil {
			if err, ok := err.(*schemas.Error); ok {
				err.Msg = "in the app matching property, " + err.Msg + ". Please review and update the action before attempting to export the users."
			}
			return newActionError(metrics.OutputValidationStep, err)
		}
	}

	// Get the transformer.
	var transformer *transformers.Transformer
	if t := this.action.Transformation; t.Mapping != nil || t.Function != nil {
		var err error
		transformer, err = transformers.New(action, this.core.transformerProvider, &connector.TimeLayouts)
		if err != nil {
			return err
		}
	}

	// Read the users.
	query := datastore.Query{Where: action.Filter}
	if connector.Type == state.FileStorage {
		query.OrderBy = action.OrderBy
	}
	records, err := store.UserRecords(ctx, query, action.InSchema, matching)
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return newActionError(metrics.ReceiveStep, err)
		}
		switch err := err.(type) {
		case *datastore.UnavailableError:
			// TODO(marco): log the error in a log specific of the workspace.
			ws := action.Connection().Workspace()
			slog.Error("cannot get users from the data warehouse", "workspace", ws.ID, "err", err)
			return err
		case *schemas.Error:
			err.Msg = fmt.Sprintf("in the input schema, %s. Please review and update the action before attempting to export the users.", err.Msg)
			return newActionError(metrics.InputValidationStep, err)
		}
		return err
	}
	defer records.Close()

	var writer connectors.Writer

	ack := func(ids []string, err error) {
		meergoMetrics.Increment("Action.exportUsers.ack.calls", 1)
		if err != nil {
			this.core.metrics.FinalizeFailed(action.ID, len(ids), err.Error())
			return
		}
		this.core.metrics.FinalizePassed(action.ID, len(ids))
	}

	// Get the writer.
	switch connector.Type {
	case state.App:
		outSchema := action.OutSchema
		if action.ExportMode == state.UpdateOnly && !matchingOut.UpdateRequired {
			outSchema = types.SubsetFunc(outSchema, func(p types.Property) bool {
				return p.Name != action.Matching.Out
			})
		}
		writer, err = this.app().Writer(ctx, outSchema, action.ExportMode, action.Target, ack)
	case state.Database:
		writer, err = this.database().Writer(ctx, action, ack)
		alreadyExportedKeys = make(map[any]struct{})
	case state.FileStorage:
		replacer := newPathPlaceholderReplacer(time.Now().UTC())
		writer, err = this.file().Writer(ctx, replacer, ack)
		if err, ok := err.(*connectors.PlaceholderError); ok {
			return fmt.Errorf("invalid file path: %s", err)
		}
	}
	if err != nil {
		if err, ok := err.(*connectors.PlaceholderError); ok {
			return fmt.Errorf("invalid file path: %s", err)
		}
		if err, ok := err.(*schemas.Error); ok {
			err.Msg = "in the output schema, " + err.Msg + ". Please review and update the action before attempting to export the users."
		}
		return newActionError(metrics.OutputValidationStep, err)
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

		meergoMetrics.Increment("Action.exportUsers.iterations_over_records_All", 1)

		if record.Err != nil {
			this.core.metrics.ReceiveFailed(action.ID, 1, record.Err.Error())
			if connector.Type == state.FileStorage {
				return record.Err
			}
			goto Next
		}

		this.core.metrics.ReceivePassed(action.ID, 1)

		if connector.Type == state.App {
			user := User{Record: record}
			isCreate := record.ExternalID == ""
			if !isCreate {
				user.ID = record.ExternalID
			}
			if isCreate || matchingOut.UpdateRequired {
				value := record.Properties[matchingIn.Name]
				user.MatchingValue, err = convertToExternal(value, matchingIn.Type, matchingOut.Type, matchingIn.Name, matchingOut.Name)
				if err != nil {
					this.core.metrics.InputValidationFailed(action.ID, 1, err.Error())
					goto Next
				}
			}
			users = append(users, user)
		} else {
			users = append(users, User{Record: record})
		}

		this.core.metrics.InputValidationPassed(action.ID, 1)

	Next:

		// Does a bach processing of users.
		if len(users) == 100 || records.Last() {

			if transformer == nil {
				for _, user := range users {
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
					return newActionError(metrics.TransformationStep, err)
				}
				return err
			}
			for i, record := range transformationRecords {
				user := users[i]
				if record.Err != nil {
					if _, ok := record.Err.(validationError); ok {
						this.core.metrics.TransformationPassed(action.ID, 1)
						this.core.metrics.OutputValidationFailed(action.ID, 1, record.Err.Error())
						continue
					}
					this.core.metrics.TransformationFailed(action.ID, 1, record.Err.Error())
					continue
				}
				this.core.metrics.TransformationPassed(action.ID, 1)
				this.core.metrics.OutputValidationPassed(action.ID, 1)
				if user.MatchingValue != nil {
					record.Properties[matchingOut.Name] = user.MatchingValue
				}
				if connector.Type == state.App && len(record.Properties) == 0 {
					this.core.metrics.FinalizePassed(action.ID, 1)
					continue
				}
				// In the case of exporting to the database, make sure that
				// users with the same value for the table key have not already
				// been exported.
				if connector.Type == state.Database {
					key := record.Properties[action.TableKey]
					if _, ok := alreadyExportedKeys[key]; ok {
						this.core.metrics.FinalizeFailed(action.ID, 1,
							fmt.Sprintf("cannot export multiple users having the same value for %q, which is used as export table key", action.TableKey))
						continue
					}
					alreadyExportedKeys[key] = struct{}{}
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
		return newActionError(metrics.ReceiveStep, err)
	}

	users = nil

	if writer2, ok := writer.(connectors.CommittableWriter); ok {
		err = writer2.Commit(ctx)
	} else {
		err = writer.Close(ctx)
	}
	if err != nil {
		return newActionError(metrics.FinalizeStep, err)
	}

	return err
}

// syncDestinationUsers syncs the destination users of the action.
func (this *Action) syncDestinationUsers(ctx context.Context) error {

	store := this.connection.store

	// Delete the outdated destination users.
	err := store.DeleteDestinationUsers(ctx, this.action.ID)
	if err != nil {
		return err
	}

	// Create a schema with only the out matching property.
	matchingOut, _ := this.action.OutSchema.Property(this.action.Matching.Out)
	schema := types.Object([]types.Property{matchingOut})

	records, err := this.app().Users(ctx, schema, time.Time{})
	if err != nil {
		return err
	}
	defer records.Close()

	var users []datastore.DestinationUser

	for user := range records.All(ctx) {

		// Return if a normalization error occurred.
		if user.Err != nil {
			return user.Err
		}

		// Store the user only if it has a value for the out matching property, and it is not nil.
		v, ok := user.Properties[matchingOut.Name]
		if ok && v != nil {
			users = append(users, datastore.DestinationUser{
				ExternalID:       user.ID,
				OutMatchingValue: stringifyMatchingValue(v),
			})
		}

		if len(users) > 0 && (len(users) == 10000 || records.Last()) {
			// Merge destination users.
			err = this.connection.store.MergeDestinationUsers(ctx, this.action.ID, users, nil)
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

func errMatchingPropertyConversion(in, ex string) error {
	return fmt.Errorf("%s property value cannot be converted to the app's %s property", in, ex)
}

// convertToExternal converts the value of an internal property to a type
// conforming to the external property. v is the value to convert, and in and ex
// are the types of the internal and external properties, respectively.
//
// Supported conversions are:
//   - int to int, uint, and text
//   - uint to int, uint, and text
//   - text to int, uint, uuid, and text
//   - uuid to uuid and text
//
// It panics if v is nil or the types in and ex are not conforming to these
// supported conversions. It returns an error if the converted value does not
// satisfy the constraints of the ex type.
func convertToExternal(v any, in, ex types.Type, inName, exName string) (any, error) {
	if v == nil {
		panic(fmt.Sprintf("core: unexpected value nil for internal kind %s ", in.Kind()))
	}
	switch ex.Kind() {
	case types.IntKind:
		var i int64
		switch v := v.(type) {
		case int:
			i = int64(v)
		case uint:
			i = int64(v)
			if i < 0 {
				return nil, errMatchingPropertyConversion(inName, exName)
			}
		case string:
			var err error
			i, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, errMatchingPropertyConversion(inName, exName)
			}
		default:
			panic(fmt.Sprintf("core: unexpected value %#v (type %T) for internal kind %s ", v, v, in.Kind()))
		}
		min, max := ex.IntRange()
		if i < min || i > max {
			return nil, errMatchingPropertyConversion(inName, exName)
		}
		return int(i), nil
	case types.UintKind:
		var i uint64
		switch v := v.(type) {
		case int:
			if v < 0 {
				return nil, errMatchingPropertyConversion(inName, exName)
			}
			i = uint64(v)
		case uint:
			i = uint64(v)
		case string:
			var err error
			i, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				return nil, errMatchingPropertyConversion(inName, exName)
			}
		default:
			panic(fmt.Sprintf("core: unexpected value %#v (type %T) for internal kind %s ", v, v, in.Kind()))
		}
		min, max := ex.UintRange()
		if i < min || i > max {
			return nil, errMatchingPropertyConversion(inName, exName)
		}
		return uint(i), nil
	case types.TextKind:
		var s string
		switch v := v.(type) {
		case int:
			s = strconv.FormatInt(int64(v), 10)
		case uint:
			s = strconv.FormatUint(uint64(v), 10)
		case string:
			s = v
		default:
			panic(fmt.Sprintf("core: unexpected value %#v (type %T) for internal kind %s ", v, v, in.Kind()))
		}
		if byteLen, ok := ex.ByteLen(); ok && len(s) > byteLen {
			return nil, errMatchingPropertyConversion(inName, exName)
		}
		if charLen, ok := ex.CharLen(); ok && utf8.RuneCountInString(s) > charLen {
			return nil, errMatchingPropertyConversion(inName, exName)
		}
		if values := ex.Values(); values != nil && !slices.Contains(values, s) {
			return nil, errMatchingPropertyConversion(inName, exName)
		}
		if re := ex.Regexp(); re != nil && !re.MatchString(s) {
			return nil, errMatchingPropertyConversion(inName, exName)
		}
		return s, nil
	case types.UUIDKind:
		switch in.Kind() {
		case types.UUIDKind:
			return v, nil
		case types.TextKind:
			u, ok := util.ParseUUID(v.(string))
			if !ok {
				return nil, errMatchingPropertyConversion(inName, exName)
			}
			return u, nil
		default:
			panic(fmt.Sprintf("core: unexpected value %#v (type %T) for internal kind %s ", v, v, in.Kind()))
		}
	}
	panic(fmt.Sprintf("core: unexpected external kind %s", ex.Kind()))
}

// stringifyMatchingValue returns the string representation of a value for a
// matching property. v cannot be nil.
func stringifyMatchingValue(v any) string {
	switch v := v.(type) {
	case int: // int(n)
		return strconv.Itoa(v)
	case uint: // uint(n)
		return strconv.FormatUint(uint64(v), 10)
	case string: // text and uuid
		return v
	default:
		panic(fmt.Sprintf("unexpected matching property value with type %T", v))
	}
}
