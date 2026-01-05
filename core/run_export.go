// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/core/internal/connections"
	"github.com/meergo/meergo/core/internal/datastore"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	meergoMetrics "github.com/meergo/meergo/tools/metrics"
	"github.com/meergo/meergo/tools/types"
)

// exportProfiles exports the profiles for the pipeline.
//
// Returns an error if run does not reach its natural completion.
// If the error is caused by the schema, the connector, or the data warehouse,
// it returns an *pipelineError, which is expected to be logged as is.
func (this *Pipeline) exportProfiles(ctx context.Context) error {

	pipeline := this.pipeline
	store := this.connection.store
	connector := pipeline.Connection().Connector()
	meergoMetrics.Increment("Pipeline.exportUsers.calls", 1)

	// Synchronize destinations users with the application's users.
	if connector.Type == state.Application {
		err := this.syncDestinationProfiles(ctx)
		if err != nil {
			if err, ok := err.(*schemas.Error); ok {
				err.Msg = "in the destination matching property, " + err.Msg + ". Please review and update the pipeline before attempting to export the users."
			}
			return newPipelineError(metrics.OutputValidationStep, err)
		}
	}

	// Get the matching properties.
	var matchingIn, matchingOut types.Property
	if connector.Type == state.Application {
		matchingIn, _ = pipeline.InSchema.Properties().ByPath(pipeline.Matching.In)
		matchingOut, _ = pipeline.OutSchema.Properties().ByPath(pipeline.Matching.Out)
	}

	// Build the transformer.
	var transformer *transformers.Transformer
	if t := this.pipeline.Transformation; t.Mapping != nil || t.Function != nil {
		var err error
		transformer, err = transformers.New(pipeline, this.core.functionProvider, &connector.TimeLayouts)
		if err != nil {
			return err
		}
	}

	// Read the users.
	query := datastore.Query{Where: pipeline.Filter}
	if connector.Type == state.FileStorage {
		query.OrderBy = pipeline.OrderBy
	}
	var matching *datastore.Matching
	if connector.Type == state.Application {
		matching = &datastore.Matching{
			Pipeline:           pipeline.ID,
			InProperty:         pipeline.Matching.In,
			ExportMode:         this.pipeline.ExportMode,
			UpdateOnDuplicates: pipeline.UpdateOnDuplicates,
		}
	}
	records, err := store.ProfileRecords(ctx, query, pipeline.InSchema, matching)
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return newPipelineError(metrics.ReceiveStep, err)
		}
		switch err := err.(type) {
		case *datastore.UnavailableError:
			return err
		case *schemas.Error:
			err.Msg = fmt.Sprintf("in the input schema, %s. Please review and update the pipeline before attempting to export the profiles.", err.Msg)
			return newPipelineError(metrics.InputValidationStep, err)
		}
		return err
	}
	defer records.Close()

	var writer connections.Writer

	var ack func([]string, error)
	if connector.Type != state.FileStorage {
		ack = func(ids []string, err error) {
			meergoMetrics.Increment("Pipeline.exportProfiles.ack.calls", 1)
			if err != nil {
				this.core.metrics.FinalizeFailed(pipeline.ID, len(ids), err.Error())
				return
			}
			this.core.metrics.FinalizePassed(pipeline.ID, len(ids))
		}
	}

	// alreadyExportedKeys keeps track of the keys of users exported to the
	// database during this export, indexed by their table key value (which can
	// have Go type int or string).
	var alreadyExportedKeys map[any]struct{}

	// Get the writer.
	switch connector.Type {
	case state.Application:
		// The value of the out matching property is written to the application only when
		// creating a new user or updating an existing user if the property is update-required.
		// When updating a user and the property is not update-required, it should not be written again
		// with the same value. In this case, alignment with the application schema does not need to be validated.
		// Therefore, the property must be removed from the schema passed to Application.Writer
		// so that the alignment check is skipped.
		outSchema := pipeline.OutSchema
		if pipeline.ExportMode == state.UpdateOnly && !matchingOut.UpdateRequired {
			outSchema = types.Prune(outSchema, func(path string) bool {
				return path != pipeline.Matching.Out
			})
		}
		writer, err = this.application().Writer(ctx, outSchema, pipeline.ExportMode, pipeline.Target, ack)
	case state.Database:
		writer, err = this.database().Writer(ctx, pipeline, ack)
		alreadyExportedKeys = make(map[any]struct{})
	case state.FileStorage:
		replacer := newPathPlaceholderReplacer(time.Now().UTC())
		writer, err = this.file().Writer(ctx, replacer)
		if err, ok := err.(*connections.PlaceholderError); ok {
			return fmt.Errorf("invalid file path: %s", err)
		}
	}
	if err != nil {
		if err, ok := err.(*connections.PlaceholderError); ok {
			return fmt.Errorf("invalid file path: %s", err)
		}
		if err, ok := err.(*schemas.Error); ok {
			err.Msg = "in the output schema, " + err.Msg + ". Please review and update the pipeline before attempting to export the users."
		}
		return newPipelineError(metrics.OutputValidationStep, err)
	}
	defer writer.Close(ctx)

	// Profile represents a profile to update or create.
	type Profile struct {
		ID            string           // External application identifier; is non-empty only for application's profiles to update.
		Record        datastore.Record // Profile record.
		MatchingValue any              // External matching property value added to properties when creating an application profile.
	}

	profiles := make([]Profile, 0, 100)
	transformationRecords := make([]transformers.Record, 0, 100)

	var readCount int // Total number of records successfully read from the warehouse so far.

	if connector.Type == state.FileStorage {
		defer func() {
			if readCount > 0 {
				this.core.metrics.FinalizeFailed(pipeline.ID, readCount, err.Error())
			}
		}()
	}

Records:
	for record := range records.All(ctx) {

		meergoMetrics.Increment("Pipeline.exportProfiles.iterations_over_records_All", 1)

		if record.Err != nil {
			this.core.metrics.ReceiveFailed(pipeline.ID, 1, record.Err.Error())
			if connector.Type == state.FileStorage {
				return newPipelineError(metrics.ReceiveStep, record.Err)
			}
			goto Next
		}

		readCount++
		this.core.metrics.ReceivePassed(pipeline.ID, 1)

		switch connector.Type {
		default:
			profiles = append(profiles, Profile{Record: record})
		case state.Application:
			profile := Profile{Record: record}
			// Update: use ExternalID as the profile ID.
			if isUpdate := record.ExternalID != ""; isUpdate {
				profile.ID = record.ExternalID
			}
			// Create or when the outgoing matching property must be updated: compute and preserve the matching value.
			if isCreate := record.ExternalID == ""; isCreate || matchingOut.UpdateRequired {
				value, _ := getAttribute(record.Attributes, pipeline.Matching.In)
				profile.MatchingValue, err = convertToExternal(value, matchingIn.Type, matchingOut.Type, pipeline.Matching.In, pipeline.Matching.Out)
				if err != nil {
					this.core.metrics.InputValidationFailed(pipeline.ID, 1, err.Error())
					goto Next
				}
			}
			profiles = append(profiles, profile)
		}

		this.core.metrics.InputValidationPassed(pipeline.ID, 1)

	Next:

		// Does a bach processing of profiles.
		if len(profiles) == 100 || records.Last() {

			if transformer == nil {
				for _, profile := range profiles {
					if !writer.Write(ctx, "", profile.Record.Attributes) {
						break Records
					}
				}
				clear(profiles)
				profiles = profiles[0:0]
				continue
			}

			// Transform the profiles.
			clear(transformationRecords)
			transformationRecords = transformationRecords[0:len(profiles)]
			for i, profile := range profiles {
				purpose := transformers.Create
				if profile.ID != "" {
					purpose = transformers.Update
				}
				transformationRecords[i].Purpose = purpose
				transformationRecords[i].Attributes = profile.Record.Attributes
			}
			err := transformer.Transform(ctx, transformationRecords)
			if err != nil {
				if _, ok := err.(transformers.FunctionExecError); ok {
					err = newPipelineError(metrics.TransformationStep, err)
				}
				return err
			}
			for i, record := range transformationRecords {
				user := profiles[i]
				if err := record.Err; err != nil {
					switch err.(type) {
					case transformers.RecordTransformationError:
						this.core.metrics.TransformationFailed(pipeline.ID, 1, err.Error())
					case transformers.RecordValidationError:
						this.core.metrics.TransformationPassed(pipeline.ID, 1)
						this.core.metrics.OutputValidationFailed(pipeline.ID, 1, err.Error())
					}
					continue
				}
				this.core.metrics.TransformationPassed(pipeline.ID, 1)
				this.core.metrics.OutputValidationPassed(pipeline.ID, 1)
				if user.MatchingValue != nil {
					setAttribute(record.Attributes, pipeline.Matching.Out, user.MatchingValue)
				}
				if connector.Type == state.Application && len(record.Attributes) == 0 {
					this.core.metrics.FinalizePassed(pipeline.ID, 1)
					continue
				}
				// In the case of exporting to the database, make sure that
				// profiles with the same value for the table key have not already
				// been exported.
				if connector.Type == state.Database {
					key := record.Attributes[pipeline.TableKey]
					if _, ok := alreadyExportedKeys[key]; ok {
						this.core.metrics.FinalizeFailed(pipeline.ID, 1,
							fmt.Sprintf("cannot export multiple profiles having the same value for %q, which is used as export table key", pipeline.TableKey))
						continue
					}
					alreadyExportedKeys[key] = struct{}{}
				}
				if !writer.Write(ctx, user.ID, record.Attributes) {
					break Records
				}
			}
			clear(profiles)
			profiles = profiles[0:0]

		}

	}
	if err = records.Err(); err != nil {
		return newPipelineError(metrics.ReceiveStep, err)
	}

	profiles = nil

	err = writer.Close(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return err
		}
		return newPipelineError(metrics.FinalizeStep, err)
	}

	if connector.Type == state.FileStorage {
		this.core.metrics.FinalizePassed(pipeline.ID, readCount)
		readCount = 0 // prevents them from being flagged as failed in the metrics
	}

	return nil
}

// syncDestinationProfiles syncs the destination profiles of the pipeline.
func (this *Pipeline) syncDestinationProfiles(ctx context.Context) error {

	store := this.connection.store

	// Delete the outdated destination profiles.
	err := store.DeleteDestinationProfiles(ctx, this.pipeline.ID)
	if err != nil {
		return err
	}

	// Build a schema containing only the hierarchy that leads to the matching output property.
	schema, _ := types.PruneAtPath(this.pipeline.OutSchema, this.pipeline.Matching.Out)

	records, err := this.application().Users(ctx, schema, nil, time.Time{})
	if err != nil {
		return err
	}
	defer records.Close()

	var profiles []datastore.DestinationProfile

	for profile := range records.All(ctx) {

		// Return if a normalization error occurred.
		if profile.Err != nil {
			return profile.Err
		}

		// Store the profile only if the output matching property is not nil.
		v, ok := getAttribute(profile.Attributes, this.pipeline.Matching.Out)
		if !ok {
			panic(fmt.Sprintf("out matching property value of pipeline %d is missing", this.pipeline.ID))
		}
		if v != nil {
			profiles = append(profiles, datastore.DestinationProfile{
				ExternalID:       profile.ID,
				OutMatchingValue: stringifyMatchingValue(v),
			})
		}

		if len(profiles) > 0 && (len(profiles) == 10000 || records.Last()) {
			// Merge destination profiles.
			err = this.connection.store.MergeDestinationUsers(ctx, this.pipeline.ID, profiles, nil)
			if err != nil {
				return err
			}
			profiles = profiles[0:0]
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
	return fmt.Errorf("%s property value cannot be converted to the application's %s property", in, ex)
}

// convertToExternal converts the value of an internal property to a type
// conforming to the external property. v is the value to convert, and in and ex
// are the types of the internal and external properties, respectively.
//
// Supported conversions are:
//   - int to int and string
//   - string to int, uuid, and string
//   - uuid to uuid and string
//
// It panics if v is nil or the types in and ex are not conforming to these
// supported conversions. It returns an error if the converted value does not
// satisfy the constraints of the ex type.
func convertToExternal(v any, in, ex types.Type, inPath, outPath string) (any, error) {
	if v == nil {
		panic(fmt.Sprintf("core: unexpected value nil for internal kind %s", in.Kind()))
	}
	switch ex.Kind() {
	case types.StringKind:
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
		if n, ok := ex.MaxBytes(); ok && len(s) > n {
			return nil, errMatchingPropertyConversion(inPath, outPath)
		}
		if n, ok := ex.MaxLength(); ok && utf8.RuneCountInString(s) > n {
			return nil, errMatchingPropertyConversion(inPath, outPath)
		}
		if values := ex.Values(); values != nil && !slices.Contains(values, s) {
			return nil, errMatchingPropertyConversion(inPath, outPath)
		}
		if re := ex.Pattern(); re != nil && !re.MatchString(s) {
			return nil, errMatchingPropertyConversion(inPath, outPath)
		}
		return s, nil
	case types.IntKind:
		if ex.IsUnsigned() {
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
			min, max := ex.UnsignedRange()
			if i < min || i > max {
				return nil, errMatchingPropertyConversion(inPath, outPath)
			}
			return uint(i), nil
		}
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
	case types.UUIDKind:
		switch in.Kind() {
		case types.StringKind:
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

// getAttribute gets the attribute for a nested property by path in attributes.
// It returns the attribute and true if found, or nil and false if missing.
func getAttribute(attributes map[string]any, path string) (any, bool) {
	for {
		name, rest, found := strings.Cut(path, ".")
		if !found {
			break
		}
		attrs, ok := attributes[name].(map[string]any)
		if !ok {
			return nil, false
		}
		attributes = attrs
		path = rest
	}
	value, ok := attributes[path]
	return value, ok
}

// setAttribute sets the attribute for the given out property path in
// attributes. If the property or a parent exist, setAttribute overwrite it.
func setAttribute(attributes map[string]any, path string, value any) {
	for {
		name, rest, found := strings.Cut(path, ".")
		if !found {
			break
		}
		attrs, ok := attributes[name].(map[string]any)
		if !ok {
			attrs = map[string]any{}
			attributes[name] = attrs
		}
		attributes = attrs
		path = rest
	}
	attributes[path] = value
}

// stringifyMatchingValue returns the string representation of a value for a
// matching property. v cannot be nil.
func stringifyMatchingValue(v any) string {
	switch v := v.(type) {
	case string: // string and uuid
		return v
	case int: // int(n)
		return strconv.Itoa(v)
	case uint: // unsigned int(n)
		return strconv.FormatUint(uint64(v), 10)
	default:
		panic(fmt.Sprintf("unexpected matching property value with type %T", v))
	}
}
