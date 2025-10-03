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
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/core/internal/connectors"
	"github.com/meergo/meergo/core/internal/datastore"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	meergoMetrics "github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/types"
)

// exportUsers exports the users for the action.
//
// Returns an error if execution does not reach its natural completion.
// If the error is caused by the schema, the connector, or the data warehouse,
// it returns an *actionError, which is expected to be logged as is.
func (this *Action) exportUsers(ctx context.Context) error {

	action := this.action
	store := this.connection.store
	connector := action.Connection().Connector()
	meergoMetrics.Increment("Action.exportUsers.calls", 1)

	// Synchronize destinations users with the app users.
	if connector.Type == state.App {
		err := this.syncDestinationUsers(ctx)
		if err != nil {
			if err, ok := err.(*schemas.Error); ok {
				err.Msg = "in the app matching property, " + err.Msg + ". Please review and update the action before attempting to export the users."
			}
			return newActionError(metrics.OutputValidationStep, err)
		}
	}

	// Get the matching properties.
	var matchingIn, matchingOut types.Property
	if connector.Type == state.App {
		matchingIn, _ = action.InSchema.Properties().ByPath(action.Matching.In)
		matchingOut, _ = action.OutSchema.Properties().ByPath(action.Matching.Out)
	}

	// Build the transformer.
	var transformer *transformers.Transformer
	if t := this.action.Transformation; t.Mapping != nil || t.Function != nil {
		var err error
		transformer, err = transformers.New(action, this.core.functionProvider, &connector.TimeLayouts)
		if err != nil {
			return err
		}
	}

	// Read the users.
	query := datastore.Query{Where: action.Filter}
	if connector.Type == state.FileStorage {
		query.OrderBy = action.OrderBy
	}
	var matching *datastore.Matching
	if connector.Type == state.App {
		matching = &datastore.Matching{
			Action:             action.ID,
			InProperty:         action.Matching.In,
			ExportMode:         this.action.ExportMode,
			UpdateOnDuplicates: action.UpdateOnDuplicates,
		}
	}
	records, err := store.UserRecords(ctx, query, action.InSchema, matching)
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return newActionError(metrics.ReceiveStep, err)
		}
		switch err := err.(type) {
		case *datastore.UnavailableError:
			return err
		case *schemas.Error:
			err.Msg = fmt.Sprintf("in the input schema, %s. Please review and update the action before attempting to export the users.", err.Msg)
			return newActionError(metrics.InputValidationStep, err)
		}
		return err
	}
	defer records.Close()

	var writer connectors.Writer

	var ack func([]string, error)
	if connector.Type != state.FileStorage {
		ack = func(ids []string, err error) {
			meergoMetrics.Increment("Action.exportUsers.ack.calls", 1)
			if err != nil {
				this.core.metrics.FinalizeFailed(action.ID, len(ids), err.Error())
				return
			}
			this.core.metrics.FinalizePassed(action.ID, len(ids))
		}
	}

	// alreadyExportedKeys keeps track of the keys of users exported to the
	// database during this export, indexed by their table key value (which can
	// have Go type int, uint o string).
	var alreadyExportedKeys map[any]struct{}

	// Get the writer.
	switch connector.Type {
	case state.App:
		// The value of the out matching property is written to the app only when
		// creating a new user or updating an existing user if the property is update-required.
		// When updating a user and the property is not update-required, it should not be written again
		// with the same value. In this case, alignment with the app schema does not need to be validated.
		// Therefore, the property must be removed from the schema passed to App.Writer
		// so that the alignment check is skipped.
		outSchema := action.OutSchema
		if action.ExportMode == state.UpdateOnly && !matchingOut.UpdateRequired {
			outSchema = types.Prune(outSchema, func(path string) bool {
				return path != action.Matching.Out
			})
		}
		writer, err = this.app().Writer(ctx, outSchema, action.ExportMode, action.Target, ack)
	case state.Database:
		writer, err = this.database().Writer(ctx, action, ack)
		alreadyExportedKeys = make(map[any]struct{})
	case state.FileStorage:
		replacer := newPathPlaceholderReplacer(time.Now().UTC())
		writer, err = this.file().Writer(ctx, replacer)
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

	var readCount int // Total number of records successfully read from the warehouse so far.

	if connector.Type == state.FileStorage {
		defer func() {
			if readCount > 0 {
				this.core.metrics.FinalizeFailed(action.ID, readCount, err.Error())
			}
		}()
	}

Records:
	for record := range records.All(ctx) {

		meergoMetrics.Increment("Action.exportUsers.iterations_over_records_All", 1)

		if record.Err != nil {
			this.core.metrics.ReceiveFailed(action.ID, 1, record.Err.Error())
			if connector.Type == state.FileStorage {
				return newActionError(metrics.ReceiveStep, record.Err)
			}
			goto Next
		}

		readCount++
		this.core.metrics.ReceivePassed(action.ID, 1)

		switch connector.Type {
		default:
			users = append(users, User{Record: record})
		case state.App:
			user := User{Record: record}
			// Update: use ExternalID as the user ID.
			if isUpdate := record.ExternalID != ""; isUpdate {
				user.ID = record.ExternalID
			}
			// Create or when the outgoing matching property must be updated: compute and preserve the matching value.
			if isCreate := record.ExternalID == ""; isCreate || matchingOut.UpdateRequired {
				value, _ := getPropertyValue(record.Properties, action.Matching.In)
				user.MatchingValue, err = convertToExternal(value, matchingIn.Type, matchingOut.Type, action.Matching.In, action.Matching.Out)
				if err != nil {
					this.core.metrics.InputValidationFailed(action.ID, 1, err.Error())
					goto Next
				}
			}
			users = append(users, user)
		}

		this.core.metrics.InputValidationPassed(action.ID, 1)

	Next:

		// Does a bach processing of users.
		if len(users) == 100 || records.Last() {

			if transformer == nil {
				for _, user := range users {
					if !writer.Write(ctx, "", user.Record.Properties) {
						break Records
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
				if _, ok := err.(transformers.FunctionExecError); ok {
					err = newActionError(metrics.TransformationStep, err)
				}
				return err
			}
			for i, record := range transformationRecords {
				user := users[i]
				if err := record.Err; err != nil {
					switch err.(type) {
					case transformers.RecordTransformationError:
						this.core.metrics.TransformationFailed(action.ID, 1, err.Error())
					case transformers.RecordValidationError:
						this.core.metrics.TransformationPassed(action.ID, 1)
						this.core.metrics.OutputValidationFailed(action.ID, 1, err.Error())
					}
					continue
				}
				this.core.metrics.TransformationPassed(action.ID, 1)
				this.core.metrics.OutputValidationPassed(action.ID, 1)
				if user.MatchingValue != nil {
					setPropertyValue(record.Properties, action.Matching.Out, user.MatchingValue)
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
				if !writer.Write(ctx, user.ID, record.Properties) {
					break Records
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

	err = writer.Close(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return err
		}
		return newActionError(metrics.FinalizeStep, err)
	}

	if connector.Type == state.FileStorage {
		this.core.metrics.FinalizePassed(action.ID, readCount)
		readCount = 0 // prevents them from being flagged as failed in the metrics
	}

	return nil
}

// syncDestinationUsers syncs the destination users of the action.
func (this *Action) syncDestinationUsers(ctx context.Context) error {

	store := this.connection.store

	// Delete the outdated destination users.
	err := store.DeleteDestinationUsers(ctx, this.action.ID)
	if err != nil {
		return err
	}

	// Build a schema containing only the hierarchy that leads to the matching output property.
	schema, _ := types.PruneAtPath(this.action.OutSchema, this.action.Matching.Out)

	records, err := this.app().Users(ctx, schema, nil, time.Time{})
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

		// Store the user only if the output matching property is not nil.
		v, ok := getPropertyValue(user.Properties, this.action.Matching.Out)
		if !ok {
			panic(fmt.Sprintf("out matching property value of action %d is missing", this.action.ID))
		}
		if v != nil {
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
func convertToExternal(v any, in, ex types.Type, inPath, outPath string) (any, error) {
	if v == nil {
		panic(fmt.Sprintf("core: unexpected value nil for internal kind %s", in.Kind()))
	}
	switch ex.Kind() {
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
			panic(fmt.Sprintf("core: unexpected value of type %T for internal kind %s ", v, in.Kind()))
		}
		if byteLen, ok := ex.ByteLen(); ok && len(s) > byteLen {
			return nil, errMatchingPropertyConversion(inPath, outPath)
		}
		if charLen, ok := ex.CharLen(); ok && utf8.RuneCountInString(s) > charLen {
			return nil, errMatchingPropertyConversion(inPath, outPath)
		}
		if values := ex.Values(); values != nil && !slices.Contains(values, s) {
			return nil, errMatchingPropertyConversion(inPath, outPath)
		}
		if re := ex.Regexp(); re != nil && !re.MatchString(s) {
			return nil, errMatchingPropertyConversion(inPath, outPath)
		}
		return s, nil
	case types.IntKind:
		var i int64
		switch v := v.(type) {
		case int:
			i = int64(v)
		case uint:
			i = int64(v)
			if i < 0 {
				return nil, errMatchingPropertyConversion(inPath, outPath)
			}
		case string:
			var err error
			i, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, errMatchingPropertyConversion(inPath, outPath)
			}
		default:
			panic(fmt.Sprintf("core: unexpected value of type %T for internal kind %s ", v, in.Kind()))
		}
		min, max := ex.IntRange()
		if i < min || i > max {
			return nil, errMatchingPropertyConversion(inPath, outPath)
		}
		return int(i), nil
	case types.UintKind:
		var i uint64
		switch v := v.(type) {
		case int:
			if v < 0 {
				return nil, errMatchingPropertyConversion(inPath, outPath)
			}
			i = uint64(v)
		case uint:
			i = uint64(v)
		case string:
			var err error
			i, err = strconv.ParseUint(v, 10, 64)
			if err != nil {
				return nil, errMatchingPropertyConversion(inPath, outPath)
			}
		default:
			panic(fmt.Sprintf("core: unexpected value of type %T for internal kind %s ", v, in.Kind()))
		}
		min, max := ex.UintRange()
		if i < min || i > max {
			return nil, errMatchingPropertyConversion(inPath, outPath)
		}
		return uint(i), nil
	case types.UUIDKind:
		switch in.Kind() {
		case types.TextKind:
			u, ok := types.ParseUUID(v.(string))
			if !ok {
				return nil, errMatchingPropertyConversion(inPath, outPath)
			}
			return u, nil
		case types.UUIDKind:
			return v, nil
		default:
			panic(fmt.Sprintf("core: unexpected value of type %T for internal kind %s ", v, in.Kind()))
		}
	}
	panic(fmt.Sprintf("core: unexpected external kind %s", ex.Kind()))
}

// getPropertyValue gets a nested property by path in properties. It returns the
// property's value and true if found, or nil and false if missing.
func getPropertyValue(properties map[string]any, path string) (any, bool) {
	for {
		name, rest, found := strings.Cut(path, ".")
		if !found {
			break
		}
		pp, ok := properties[name].(map[string]any)
		if !ok {
			return nil, false
		}
		properties = pp
		path = rest
	}
	value, ok := properties[path]
	return value, ok
}

// setPropertyValue sets the value for the given out property path in
// properties. If the property or a parent exist, setPropertyValue overwrite it.
func setPropertyValue(properties map[string]any, path string, value any) {
	for {
		name, rest, found := strings.Cut(path, ".")
		if !found {
			break
		}
		pp, ok := properties[name].(map[string]any)
		if !ok {
			pp = map[string]any{}
			properties[name] = pp
		}
		properties = pp
		path = rest
	}
	properties[path] = value
}

// stringifyMatchingValue returns the string representation of a value for a
// matching property. v cannot be nil.
func stringifyMatchingValue(v any) string {
	switch v := v.(type) {
	case string: // text and uuid
		return v
	case int: // int(n)
		return strconv.Itoa(v)
	case uint: // uint(n)
		return strconv.FormatUint(uint64(v), 10)
	default:
		panic(fmt.Sprintf("unexpected matching property value with type %T", v))
	}
}
