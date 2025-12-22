// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

// This file contains the function "validatePipeline", as well as any support
// type, function and/or methods used exclusively by it.

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo/core/internal/connections"
	"github.com/meergo/meergo/core/internal/datastore"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	"github.com/meergo/meergo/core/internal/transformers/mappings"
	"github.com/meergo/meergo/core/internal/util"
	"github.com/meergo/meergo/tools/errors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

const (
	MaxFilePathSize        = 1024   // maximum allowed length for a file path.
	MaxFunctionSourceSize  = 50_000 // maximum allowed size for a transformation function source.
	MaxUpdatedAtFormatSize = 64     // maximum allowed size for an update time format.
	MaxQuerySize           = 1_000  // maximum allowed size for a database query.
	MaxTableNameSize       = 1024   // maximum allowed length for a database table name.
)

// validationState is a state for the validation of a pipeline.
type validationState struct {

	// target is the pipeline's target.
	target state.Target

	// connection is the pipeline's connection.
	connection struct {
		role      state.Role
		connector struct {
			typ state.ConnectorType
		}
	}

	// format represents the pipeline file format.
	//
	// If the pipeline specifies a format name, then this must be populated
	// according to that format, if exists, otherwise must be the empty
	// struct.
	format struct {
		typ         state.ConnectorType
		targets     state.ConnectorTargets
		hasSettings bool
		hasSheets   bool
	}

	// provider is the transformers.FunctionProvider instantiated on the Core.
	provider transformers.FunctionProvider
}

// validatePipelineToSet validates the given PipelineToSet, in the context of
// the given validation state.
//
// It returns an errors.UnprocessableError error with code:
//
//   - FormatNotExist, if the pipeline is on file and the specified format does
//     not exist.
//   - UnsupportedLanguage, if the transformation language is not supported.
func validatePipelineToSet(pipeline PipelineToSet, v validationState) error {

	inSchema := pipeline.InSchema
	outSchema := pipeline.OutSchema

	importEventsIntoWarehouse := isImportingEventsIntoWarehouse(v.connection.connector.typ, v.connection.role, v.target)
	dispatchEventsToAplications := isDispatchingEventsToApplications(v.connection.connector.typ, v.connection.role, v.target)
	importUserIdentitiesFromEvents := isImportingUserIdentitiesFromEvents(v.connection.connector.typ, v.connection.role, v.target)
	exportUsersToFile := isExportUsersToFile(v.connection.connector.typ, v.connection.role, v.target)

	allowConstantTransformation := importUserIdentitiesFromEvents || dispatchEventsToAplications

	// In cases where the input schema refers to events, that is when:
	//
	//  - identities are imported from events
	//  - events are imported into the data warehouse
	//  - events are dispatched to apps
	//
	// the input schema must be nil, which means the schema of the events.
	inSchemaIsEventSchema := importUserIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToAplications
	if inSchemaIsEventSchema {
		if inSchema.Valid() {
			switch {
			case importUserIdentitiesFromEvents:
				return errors.BadRequest("input schema must be invalid for pipelines that import identities from events")
			case importEventsIntoWarehouse:
				return errors.BadRequest("input schema must be invalid for pipelines that import events into data warehouse")
			case dispatchEventsToAplications:
				return errors.BadRequest("input schema must be invalid for pipelines that send events to applications")
			}
		}
		inSchema = eventPipelineSchema
	}

	// Validate the pipeline's connector.
	pipelineOnFile := v.connection.connector.typ == state.FileStorage
	if pipelineOnFile && pipeline.Format == "" {
		return errors.BadRequest("pipelines on file storage connections must have a format")
	}
	if !pipelineOnFile && pipeline.Format != "" {
		return errors.BadRequest("pipelines on %v connections cannot have a format", v.connection.connector.typ)
	}
	if pipeline.Format != "" {
		if v.format.typ == 0 {
			return errors.Unprocessable(FormatNotExist, "format %q does not exist", pipeline.Format)
		}
		if v.format.typ != state.File {
			return errors.BadRequest("format does not refer to a file connector")
		}
	}
	if pipelineOnFile && !v.format.targets.Contains(v.target) {
		return errors.BadRequest("target is not supported by the file format")
	}

	// First, perform formal validations.

	// Validate the name.
	if err := util.ValidateStringField("name", pipeline.Name, 60); err != nil {
		return errors.BadRequest("%s", err)
	}
	// Check that, if the schemas are valid, they have type object.
	var inProperties types.Properties
	if inSchema.Valid() {
		if inSchema.Kind() != types.ObjectKind {
			return errors.BadRequest("input schema, if provided, must be an object")
		}
		inProperties = inSchema.Properties()
	}
	var outProperties types.Properties
	if outSchema.Valid() {
		if outSchema.Kind() != types.ObjectKind {
			return errors.BadRequest("out schema, if provided, must be an object")
		}
		outProperties = outSchema.Properties()
	}
	// Validate the filter.
	var usedInPaths []string
	filtersAllowed := !(v.connection.role == state.Source &&
		v.connection.connector.typ == state.Database && v.target == state.TargetUser)
	if pipeline.Filter != nil {
		if !filtersAllowed {
			return errors.BadRequest("filters are not allowed")
		}
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the filter")
		}
		properties, err := validateFilter(pipeline.Filter, inSchema, v.connection.role, v.target)
		if err != nil {
			return errors.BadRequest("filter is not valid: %w", err)
		}
		if !exportUsersToFile {
			usedInPaths = properties
		}
	}
	// Validate the transformation.
	var usedOutPaths []string
	var mappingInPaths int
	if tr := pipeline.Transformation; tr != nil {
		if tr.Mapping != nil && tr.Function != nil {
			return errors.BadRequest("pipeline cannot have both transformation mapping and function")
		}
		switch {
		case tr.Mapping != nil:
			// Validate the transformation mapping.
			if len(tr.Mapping) == 0 {
				return errors.BadRequest("transformation mapping must have mapped properties")
			}
			if !inSchema.Valid() && !allowConstantTransformation {
				return errors.BadRequest("input schema is required by the mapping")
			}
			if !outSchema.Valid() {
				return errors.BadRequest("output schema is required by the mapping")
			}
			transformer, err := mappings.New(tr.Mapping, inSchema, outSchema, false, nil)
			if err != nil {
				return errors.BadRequest("invalid mapping: %s", err)
			}
			// Input property paths.
			inProps := transformer.InPaths()
			mappingInPaths = len(inProps)
			usedInPaths = append(usedInPaths, inProps...)
			// Output property paths.
			usedOutPaths = transformer.OutPaths()
		case tr.Function != nil:
			// Validate the transformation function.
			if !inSchema.Valid() && !allowConstantTransformation {
				return errors.BadRequest("input schema is required by the transformation function")
			}
			if !outSchema.Valid() {
				return errors.BadRequest("output schema is required by the transformation function")
			}
			if err := util.ValidateStringField("source of transformation function", tr.Function.Source, MaxFunctionSourceSize); err != nil {
				return errors.BadRequest("%s", err)
			}
			switch tr.Function.Language {
			case "JavaScript":
				if v.provider == nil || !v.provider.SupportLanguage(state.JavaScript) {
					return errors.Unprocessable(UnsupportedLanguage, "JavaScript function language is not supported")
				}
			case "Python":
				if v.provider == nil || !v.provider.SupportLanguage(state.Python) {
					return errors.Unprocessable(UnsupportedLanguage, "Python transformation language is not supported")
				}
			case "":
				return errors.BadRequest("transformation language is empty")
			default:
				return errors.BadRequest("transformation language %q is not valid", tr.Function.Language)
			}
			err := validateTransformationFunctionPaths("input", inSchema, tr.Function.InPaths, allowConstantTransformation)
			if err != nil {
				return errors.BadRequest("%s", err.Error())
			}
			err = validateTransformationFunctionPaths("output", outSchema, tr.Function.OutPaths, allowConstantTransformation)
			if err != nil {
				return errors.BadRequest("%s", err.Error())
			}
			usedInPaths = append(usedInPaths, tr.Function.InPaths...)
			usedOutPaths = tr.Function.OutPaths
		default:
			return errors.BadRequest("pipeline cannot have a transformation without mapping and function.")
		}
	}
	// Validate the path.
	if pipeline.Path != "" {
		if err := util.ValidateStringField("path", pipeline.Path, MaxFilePathSize); err != nil {
			return errors.BadRequest("%s", err)
		}
		switch v.connection.role {
		case state.Source:
			_, err := connections.ReplacePlaceholders(pipeline.Path, func(_ string) (string, bool) {
				return "", false
			})
			if err != nil {
				return errors.BadRequest("placeholders syntax is not supported by source pipelines")
			}
		case state.Destination:
			_, err := connections.ReplacePlaceholders(pipeline.Path, func(name string) (string, bool) {
				name = strings.ToLower(name)
				return "", name == "today" || name == "now" || name == "unix"
			})
			if err != nil {
				return errors.BadRequest("path is not valid: %s", err)
			}
		}
	}
	// Validate the table name.
	if pipeline.TableName != "" {
		if err := util.ValidateStringField("table name", pipeline.TableName, MaxTableNameSize); err != nil {
			return errors.BadRequest("%s", err)
		}
	}
	// Validate the sheet.
	if pipeline.Sheet != "" && !connections.IsValidSheetName(pipeline.Sheet) {
		return errors.BadRequest("sheet name is not valid")
	}
	// Validate the export mode.
	if pipeline.ExportMode != "" {
		switch pipeline.ExportMode {
		case CreateOnly, UpdateOnly, CreateOrUpdate:
		default:
			return errors.BadRequest("export mode %q is not valid", pipeline.ExportMode)
		}
	}
	// Validate the matching properties.
	if pipeline.Matching.In != "" || pipeline.Matching.Out != "" {
		if pipeline.Matching.In == "" {
			return errors.BadRequest("input matching property cannot be empty if output matching property is not empty")
		}
		if pipeline.Matching.Out == "" {
			return errors.BadRequest("output matching property cannot be empty if input matching property is not empty")
		}
		if pipeline.ExportMode == "" {
			return errors.BadRequest("export mode cannot be empty if there are matching properties")
		}
		// Validate the input matching property.
		if !inSchema.Valid() {
			return errors.BadRequest("input schema must be valid")
		}
		if !types.IsValidPropertyPath(pipeline.Matching.In) {
			return errors.BadRequest("input matching property %q is not a valid property path", pipeline.Matching.In)
		}
		in, err := inProperties.ByPath(pipeline.Matching.In)
		if err != nil {
			return errors.BadRequest("input matching property %q not found within the input schema", pipeline.Matching.In)
		}
		if !canBeUsedAsMatchingProp(in.Type.Kind()) {
			return errors.BadRequest("type %s cannot be used as matching property", in.Type)
		}
		// Validate the output matching property.
		if !outSchema.Valid() {
			return errors.BadRequest("output schema must be valid")
		}
		if !types.IsValidPropertyPath(pipeline.Matching.Out) {
			return errors.BadRequest("output matching property %q is not a valid property name", pipeline.Matching.Out)
		}
		out, err := outProperties.ByPath(pipeline.Matching.Out)
		if err != nil {
			return errors.BadRequest("output matching property %q not found within the output schema", pipeline.Matching.Out)
		}
		if !canBeUsedAsMatchingProp(out.Type.Kind()) {
			return errors.BadRequest("type %s cannot be used as matching property", out.Type)
		}
		// Check that the input property can be converted to the output property.
		switch in.Type.Kind() {
		case types.IntKind:
			if k := out.Type.Kind(); k == types.UUIDKind {
				return errors.BadRequest("input matching property cannot be converted to the output matching property")
			}
		case types.UUIDKind:
			if k := out.Type.Kind(); k == types.IntKind {
				return errors.BadRequest("input matching property cannot be converted to the output matching property")
			}
		}
		// Check that the output property has not been transformed.
		// This includes checks on the property itself and all its parent paths.
		if tr := pipeline.Transformation; tr != nil {
			path := pipeline.Matching.Out
			for {
				if tr.Mapping != nil {
					if _, ok := tr.Mapping[path]; ok {
						return errors.BadRequest("mapping cannot map over the output matching property")
					}
				} else {
					if slices.Contains(tr.Function.OutPaths, path) {
						return errors.BadRequest("transformation function cannot transform over the output matching property")
					}
				}
				i := strings.LastIndexByte(path, '.')
				if i == -1 {
					break
				}
				path = path[:i]
			}
		}
		usedInPaths = append(usedInPaths, pipeline.Matching.In)
		usedOutPaths = append(usedOutPaths, pipeline.Matching.Out)
	}
	// Validate the compression.
	switch pipeline.Compression {
	case NoCompression, ZipCompression, GzipCompression, SnappyCompression:
	default:
		return errors.BadRequest("compression %q is not valid", pipeline.Compression)
	}
	// Validate the identity column.
	if pipeline.IdentityColumn != "" {
		if !types.IsValidPropertyName(pipeline.IdentityColumn) {
			return errors.BadRequest("identity column is not a valid property name")
		}
		if utf8.RuneCountInString(pipeline.IdentityColumn) > 1024 {
			return errors.BadRequest("identity column is longer than 1024 runes")
		}
	}
	// Validate the update time column.
	if pipeline.UpdatedAtColumn != "" {
		if !types.IsValidPropertyName(pipeline.UpdatedAtColumn) {
			return errors.BadRequest("update time column is not a valid property name")
		}
		if utf8.RuneCountInString(pipeline.UpdatedAtColumn) > 1024 {
			return errors.BadRequest("update time column is longer than 1024 runes")
		}
	}
	// Validate the update time format.
	if pipeline.UpdatedAtFormat != "" {
		if err := validateUpdatedAtFormat(pipeline.UpdatedAtFormat); err != nil {
			return errors.BadRequest("%s", err)
		}
	}
	// Validate the "order by" property path.
	if pipeline.OrderBy != "" {
		if !types.IsValidPropertyPath(pipeline.OrderBy) {
			return errors.BadRequest("the specified order by is not a valid property path")
		}
		if utf8.RuneCountInString(pipeline.OrderBy) > 1024 {
			return errors.BadRequest("the specified order by is longer than 1024 runes")
		}
	}
	// Validate incremental.
	if pipeline.Incremental {
		if v.connection.role == state.Destination {
			return errors.BadRequest("incremental cannot be true for destination pipelines")
		}
		switch v.connection.connector.typ {
		case state.Application:
		case state.Database, state.FileStorage:
			if pipeline.UpdatedAtColumn == "" {
				return errors.BadRequest("incremental requires an update time column")
			}
		default:
			return errors.BadRequest("incremental cannot be true for %s pipelines", v.connection.connector.typ)
		}
	}

	// Second, do validations based on the workspace and the connection.

	if importEventsIntoWarehouse && outSchema.Valid() {
		return errors.BadRequest("output schema must be invalid when importing events into data warehouse")
	}

	// Do some validations on the input and the output schemas.
	if inSchema.Valid() && !inSchemaIsEventSchema {
		if err := validatePipelineSchema("input", inSchema, v.connection.role, v.target, v.connection.connector.typ, pipeline.TableKey); err != nil {
			return errors.BadRequest("%s", err)
		}
	}
	if outSchema.Valid() {
		if err := validatePipelineSchema("output", outSchema, v.connection.role, v.target, v.connection.connector.typ, pipeline.TableKey); err != nil {
			return errors.BadRequest("%s", err)
		}
	}

	// Check if the settings are allowed and are a JSON Object.
	if v.connection.connector.typ == state.FileStorage {
		if pipeline.FormatSettings == nil {
			if v.format.hasSettings {
				return errors.BadRequest("format settings must be provided because format %s has %s settings", pipeline.Format, strings.ToLower(v.connection.role.String()))
			}
		} else {
			if !v.format.hasSettings {
				return errors.BadRequest("format settings cannot be provided because format %s has no %s settings", pipeline.Format, strings.ToLower(v.connection.role.String()))
			}
			if !json.Valid(pipeline.FormatSettings) || !pipeline.FormatSettings.IsObject() {
				return errors.BadRequest("format settings are not a valid JSON Object")
			}
		}
	} else if pipeline.FormatSettings != nil {
		return errors.BadRequest("%s pipelines cannot have %s format settings", strings.ToLower(v.connection.connector.typ.String()), strings.ToLower(v.connection.role.String()))
	}

	// Check if the compression is allowed.
	if pipeline.Compression != NoCompression && v.connection.connector.typ != state.FileStorage {
		return errors.BadRequest("%s pipelines cannot have compression", strings.ToLower(v.connection.connector.typ.String()))
	}

	// Check if the query is allowed.
	if needsQuery := v.connection.connector.typ == state.Database && v.connection.role == state.Source; needsQuery {
		if err := util.ValidateStringField("query", pipeline.Query, MaxQuerySize); err != nil {
			return errors.BadRequest("%s", err)
		}
	} else {
		if pipeline.Query != "" {
			return errors.BadRequest("%s pipelines cannot have a query", v.connection.connector.typ)
		}
	}

	// Check if the path and the sheet are allowed.
	if v.connection.connector.typ == state.FileStorage {
		if pipeline.Path == "" {
			return errors.BadRequest("path cannot be empty for pipelines on storage connections")
		}
		if v.format.hasSheets && pipeline.Sheet == "" {
			return errors.BadRequest("sheet cannot be empty because format %s has sheets", pipeline.Format)
		}
		if !v.format.hasSheets && pipeline.Sheet != "" {
			return errors.BadRequest("format %s does not have sheets", pipeline.Format)
		}
	} else {
		if pipeline.Path != "" {
			return errors.BadRequest("%s pipelines cannot have a path", v.connection.connector.typ)
		}
		if pipeline.Sheet != "" {
			return errors.BadRequest("%s pipelines cannot have a sheet", v.connection.connector.typ)
		}
	}

	// Check the column for the identity column and for the timestamp.
	importFromColumns := v.connection.role == state.Source &&
		(v.connection.connector.typ == state.Database || v.connection.connector.typ == state.FileStorage)
	if importFromColumns {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema must be valid")
		}
		// Validate the identity column.
		if pipeline.IdentityColumn == "" {
			return errors.BadRequest("identity column is mandatory")
		}
		identityColumn, ok := inProperties.ByName(pipeline.IdentityColumn)
		if !ok {
			return errors.BadRequest("identity column %q not found within input schema", pipeline.IdentityColumn)
		}
		switch k := identityColumn.Type.Kind(); k {
		case types.StringKind, types.IntKind, types.UUIDKind, types.JSONKind:
		default:
			return errors.BadRequest("identity column %q has kind %s instead of int, uuid, json, or string", pipeline.IdentityColumn, k)
		}
		if identityColumn.ReadOptional {
			return errors.BadRequest("identity column cannot be optional")
		}
		usedInPaths = append(usedInPaths, pipeline.IdentityColumn)
		// Validate the update time column and format.
		var requiresUpdatedAtFormat bool
		if pipeline.UpdatedAtColumn != "" {
			updatedAt, ok := inProperties.ByName(pipeline.UpdatedAtColumn)
			if !ok {
				return errors.BadRequest("update time column %q not found within input schema", pipeline.UpdatedAtColumn)
			}
			switch k := updatedAt.Type.Kind(); k {
			case types.StringKind, types.JSONKind:
				requiresUpdatedAtFormat = true
			case types.DateTimeKind, types.DateKind:
			default:
				return errors.BadRequest("update time column %q has kind %s instead of datetime, date, json, or string", pipeline.UpdatedAtColumn, k)
			}
			usedInPaths = append(usedInPaths, pipeline.UpdatedAtColumn)
		}
		if !requiresUpdatedAtFormat && pipeline.UpdatedAtFormat != "" {
			return errors.BadRequest("pipeline cannot specify an update time format")
		} else if requiresUpdatedAtFormat {
			if pipeline.UpdatedAtFormat == "" {
				return errors.BadRequest("update time format is required")
			}
			if v.connection.connector.typ == state.Database && pipeline.UpdatedAtFormat == "Excel" {
				return errors.BadRequest("update time format cannot be Excel for database pipelines")
			}
		}
	} else {
		if pipeline.IdentityColumn != "" {
			return errors.BadRequest("pipeline cannot specify an identity column")
		}
		if pipeline.UpdatedAtColumn != "" {
			return errors.BadRequest("pipeline cannot specify an update time column")
		}
		if pipeline.UpdatedAtFormat != "" {
			return errors.BadRequest("pipeline cannot specify an update time format")
		}
	}

	// Do some checks related to exporting profiles to files.
	if exportUsersToFile {
		// When exporting profiles to file, ensure that the input schema is valid,
		// as it contains the properties that will be exported to the file.
		if !inSchema.Valid() {
			return errors.BadRequest("input schema must be valid when exporting profiles to file")
		}
		if outSchema.Valid() {
			return errors.BadRequest("output schema must be invalid when exporting profiles to file")
		}
		// Check that OrderBy is defined and exists in the input schema.
		if pipeline.OrderBy == "" {
			return errors.BadRequest("order by property cannot be empty when exporting profiles to file")
		}
		p, err := inSchema.Properties().ByPath(pipeline.OrderBy)
		if err != nil {
			return errors.BadRequest("order by property cannot be found in pipeline's input schema: %s", err)
		}
		// Check the allowed types.
		// We can use the same criteria as for the allowed types of workspace identifiers,
		// to simplify the specifications for warehouse platforms.
		switch p.Type.Kind() {
		case types.StringKind, types.IntKind, types.UUIDKind, types.IPKind:
			// Ok.
		case types.DecimalKind:
			if p.Type.Precision() != 0 {
				return errors.BadRequest("the decimal type of the order by property cannot have a precision greater than 0")
			}
		default:
			return errors.BadRequest("order by property cannot have kind %s", p.Type.Kind())
		}
	} else {
		if pipeline.OrderBy != "" {
			return errors.BadRequest("pipelines that do not export users to files cannot specify a order by property")
		}
	}

	// Do some checks related to exporting users to databases.
	exportUsersToDatabase := v.connection.connector.typ == state.Database && v.connection.role == state.Destination && v.target == state.TargetUser
	if exportUsersToDatabase {
		if pipeline.TableName == "" {
			return errors.BadRequest("table name cannot be empty for destination database pipelines")
		}
		if pipeline.TableKey == "" {
			return errors.BadRequest("table key cannot be empty for destination database pipelines")
		}
		if !types.IsValidPropertyName(pipeline.TableKey) {
			return errors.BadRequest("table key is not a valid property name")
		}
		if !outSchema.Valid() {
			return errors.BadRequest("out schema must be valid")
		}
		p, ok := outProperties.ByName(pipeline.TableKey)
		if !ok {
			return errors.BadRequest("table key %q not found within output schema", pipeline.TableKey)
		}
		if !canBeUsedAsTableKey(p.Type.Kind()) {
			return errors.BadRequest("type %s cannot be used as table key", p.Type)
		}
		if tr := pipeline.Transformation; tr != nil {
			if tr.Mapping != nil {
				if _, ok := tr.Mapping[pipeline.TableKey]; !ok {
					return errors.BadRequest("an expression must be mapped to the table key")
				}
				if len(tr.Mapping) == 1 {
					return errors.BadRequest("in addition to the table key, there must be at least one other mapped column")
				}
			} else {
				if !slices.Contains(tr.Function.OutPaths, pipeline.TableKey) {
					return errors.BadRequest("the out properties of the transformation function must contain the table key")
				}
				if len(tr.Function.OutPaths) == 1 {
					return errors.BadRequest("the out properties of the transformation function" +
						" must contain at least one other property in addition to the table key")
				}
			}
		}
	} else {
		if pipeline.TableName != "" {
			return errors.BadRequest("table name is not allowed")
		}
		if pipeline.TableKey != "" {
			return errors.BadRequest("table key is not allowed")
		}
	}

	// Check if the export options are needed.
	needsExportOptions := v.connection.connector.typ == state.Application &&
		v.connection.role == state.Destination && v.target == state.TargetUser
	if needsExportOptions {
		if pipeline.ExportMode == "" {
			return errors.BadRequest("export mode cannot be empty")
		}
		if pipeline.Matching.In == "" {
			return errors.BadRequest("matching properties must be provided")
		}
	} else {
		if pipeline.ExportMode != "" {
			return errors.BadRequest("export mode must be empty")
		}
		if pipeline.Matching.In != "" {
			return errors.BadRequest("matching properties cannot be provided")
		}
	}

	targetUsersOrGroups := v.target == state.TargetUser || v.target == state.TargetGroup

	// Check that UpdateOnDuplicates is allowed.
	updateOnDuplicatesAllowed := v.connection.connector.typ == state.Application &&
		v.connection.role == state.Destination && targetUsersOrGroups
	if !updateOnDuplicatesAllowed && pipeline.UpdateOnDuplicates {
		return errors.BadRequest("update on duplicates is not allowed")
	}

	// Check the connections for which the transformation is prohibited.
	transformationProhibited :=
		(v.connection.role == state.Source && (v.connection.connector.typ == state.SDK || v.connection.connector.typ == state.Webhook) && v.target == state.TargetEvent) ||
			(v.connection.role == state.Destination && v.connection.connector.typ == state.FileStorage && targetUsersOrGroups)
	if transformationProhibited && pipeline.Transformation != nil {
		return errors.BadRequest("pipeline cannot have a transformation")
	}

	// Check if the transformation is mandatory, with at least one input
	// property.
	transformationMandatory := targetUsersOrGroups &&
		(v.connection.connector.typ == state.Application || v.connection.connector.typ == state.Database ||
			(v.connection.role == state.Source && v.connection.connector.typ == state.FileStorage))
	if transformationMandatory && pipeline.Transformation == nil {
		return errors.BadRequest("pipeline must have a transformation")
	}

	// If constant transformations are not allowed, there must be at least one
	// property used as input to the transformation, either in mappings or
	// functions.
	if tr := pipeline.Transformation; tr != nil && !allowConstantTransformation {
		if tr.Mapping != nil && mappingInPaths == 0 {
			return errors.BadRequest("transformation must map at least one property")
		}
		if tr.Function != nil && len(pipeline.Transformation.Function.InPaths) == 0 {
			return errors.BadRequest("transformation function must have at least one input property")
		}
	}

	// It is not necessary to check that the properties of the output schema
	// marked as CreateRequired or UpdateRequired are actually transformed, as
	// this is already guaranteed. Because:
	//
	// - it is checked that every property passed in a schema is used, and since
	//   the output schema contains only transformed properties (with two
	//   exceptions, see below), this ensures that all properties of the output
	//   schema (whether Create/UpdateRequired or not) are certainly
	//   transformed;
	//
	// - as an exception to the first point, in the case of export to a
	//   database, the output schema also contains the table key property, but
	//   this is checked ad-hoc, including the fact that it must necessarily be
	//   transformed, so there is no check to be done.
	//
	// - as a second and final exception to the first point, in the case of
	//   exporting users to apps, the output schema contains the output matching
	//   property, but this property cannot be explicitly transformed, as it is
	//   managed by Meergo, so again, there is no check to be done.

	// Ensure that every property in the input and output schemas have been used
	// (by the mappings, by the filters, etc...).
	if usedInPaths != nil {
		if inSchemaIsEventSchema {
			// In these cases, the input schema is the full schema of the events,
			// both for mappings and transformations, so we cannot return an
			// error about unused properties in the input schema, because only a
			// small part of them is usually used.
		} else if path, ok := unusedPropertyPath(inSchema, usedInPaths); ok {
			return errors.BadRequest("input schema contains an unused property: %s", path)
		}
	}
	if usedOutPaths != nil {
		if path, ok := unusedPropertyPath(outSchema, usedOutPaths); ok {
			return errors.BadRequest("output schema contains an unused property: %s", path)
		}
	}

	return nil
}

// canBeUsedAsMatchingProp reports whether a type with kind k can be used as a
// matching property when exporting users to an application.
func canBeUsedAsMatchingProp(k types.Kind) bool {
	// Only int, uint, uuid, and string types are allowed.
	return k == types.StringKind || k == types.IntKind || k == types.UUIDKind
}

// canBeUsedAsTableKey reports whether a type with kind k can be used as a
// table key when exporting users to databases.
func canBeUsedAsTableKey(k types.Kind) bool {
	// Only int, uint, uuid, and string types are allowed.
	return k == types.StringKind || k == types.IntKind || k == types.UUIDKind
}

// unusedPropertyPath returns the path of an unused property in the schema and
// true if at least one exists; otherwise, it returns "" and false.
// The schema must be valid.
func unusedPropertyPath(schema types.Type, usedPaths []string) (string, bool) {
walk:
	for schemaPath, property := range schema.Properties().WalkObjects() {
		if property.Type.Kind() == types.ObjectKind {
			// Do not report unused errors for object properties, only for their
			// sub-properties.
			continue
		}
		if slices.Contains(usedPaths, schemaPath) {
			// The schema property is used directly.
			continue
		}
		for _, usedPath := range usedPaths {
			if strings.HasPrefix(schemaPath, usedPath+".") {
				// The schema property is not used directly, but a higher level
				// property is used, which therefore implies that all
				// sub-properties are also indirectly used.
				continue walk
			}
		}
		return schemaPath, true
	}
	return "", false
}

// validatePipelineSchema validates a pipeline schema, returning an error if it
// is not valid. It is not called if schema is the event schema.
//
// io specifies whether the validation relates to "input" or "output", schema is
// the schema of the input or output pipeline, role and target are the role and
// target of the pipeline, and typ is the pipeline's connection type.
func validatePipelineSchema(io string, schema types.Type, role state.Role, target state.Target, typ state.ConnectorType, tableKey string) error {

	isProfileSchema := target == state.TargetUser &&
		(io == "input" && role == state.Destination || io == "output" && role == state.Source)

	for path, p := range schema.Properties().WalkAll() {
		if p.Prefilled != "" {
			return fmt.Errorf("%s pipeline schema property %q has a prefilled value, but pipeline schema properties cannot have prefilled values", io, path)
		}
		if isProfileSchema {
			if datastore.IsMetaProperty(path) {
				return fmt.Errorf("%s pipeline schema property %q is a meta property", io, path)
			}
			if k := p.Type.Kind(); k == types.ArrayKind || k == types.MapKind {
				elemK := p.Type.Elem().Kind()
				if elemK == types.ArrayKind || elemK == types.ObjectKind || elemK == types.MapKind {
					return fmt.Errorf("%s pipeline schema property %q cannot have type %s(%s)", io, path, k, elemK)
				}
			}
			if p.CreateRequired {
				return fmt.Errorf("%s pipeline schema property %q cannot have CreateRequired set to true", io, path)
			}
			if p.UpdateRequired {
				return fmt.Errorf("%s pipeline schema property %q cannot have UpdateRequired set to true", io, path)
			}
			if !p.ReadOptional {
				return fmt.Errorf("%s pipeline schema property %q must have ReadOptional set to true", io, path)
			}
			if p.Nullable {
				return fmt.Errorf("%s pipeline schema property %q cannot have Nullable set to true", io, path)
			}
			continue
		}
		if role == state.Source && io == "input" {
			if p.CreateRequired {
				return fmt.Errorf("source pipeline schema property %q cannot have CreateRequired set to true", path)
			}
			if p.UpdateRequired {
				return fmt.Errorf("%s pipeline schema property %q cannot have UpdateRequired set to true", io, path)
			}
			if p.ReadOptional && typ == state.Database {
				return fmt.Errorf("%s pipeline schema property %q cannot have ReadOptional set to true", io, path)
			}
			continue
		}
		if role == state.Destination && io == "output" {
			switch {
			case typ == state.Application && target == state.TargetEvent:
				if p.UpdateRequired {
					return fmt.Errorf("output pipeline schema property %q cannot have UpdateRequired set to true", path)
				}
			case typ == state.Database:
				if path == tableKey {
					if !p.CreateRequired {
						return fmt.Errorf("table key property %q in output pipeline schema must have CreateRequired set to true", path)
					}
					if p.Nullable {
						return fmt.Errorf("table key property %q in output pipeline schema cannot have Nullable set to true", path)
					}
				} else {
					if p.CreateRequired {
						return fmt.Errorf("output pipeline schema property %q cannot have CreateRequired set to true", path)
					}
				}
				if p.UpdateRequired {
					return fmt.Errorf("output pipeline schema property %q cannot have UpdateRequired set to true", path)
				}
			}
			if p.ReadOptional {
				return fmt.Errorf("output pipeline schema property %q cannot have ReadOptional set to true", path)
			}
		}
	}

	if isProfileSchema {
		if err := datastore.CheckConflictingProperties(io, schema); err != nil {
			return err
		}
	}

	return nil
}

// validateUpdatedAtFormat validates the given update time format for importing
// files and database rows, returning an error in case the format is not valid.
//
// Valid formats are
//
//   - "ISO8601": the ISO 8601 format
//   - "Excel": the Excel format, a float value stored in an Excel cell representing a date/datetime
//   - a string containing a '%' character: the strftime() function format
//
// NOTE: keep in sync with the function
// 'core/connectors.parseUpdatedAtColumnWithFormat'.
func validateUpdatedAtFormat(format string) error {
	switch format {
	case
		"ISO8601",
		"Excel":
		return nil
	}
	if err := util.ValidateStringField("update time format", format, MaxUpdatedAtFormatSize); err != nil {
		return err
	}
	if !strings.Contains(format, "%") {
		return fmt.Errorf("update time format %q is not valid", format)
	}
	return nil
}

// validateTransformationFunctionPaths validates the transformation function
// paths of a pipeline.
//
// io specifies whether the validation relates to "input" or "output", schema is
// the schema of the input or output pipeline, paths are the function paths for
// input or output.
//
// allowConstantTransformation indicates if the transformation functions allows
// constant transformations or not.
//
// In more detail:
//
//   - paths can never be nil;
//   - paths can be empty only in the case of the input transformation function
//     when dispatching events to apps;
//   - each path must exist in the schema;
//   - there can be no repeated paths, nor paths that are sub-paths of others
//     (such as "x.a" and "x");
//   - paths cannot "cross" array and map elements, but only object, so it is
//     possible to refer to array and map properties only as a whole, not to
//     their specific elements.
//
// It panics if the schema is valid and is not an object.
func validateTransformationFunctionPaths(io string, schema types.Type, paths []string, allowConstantTransformation bool) error {
	if len(paths) == 0 {
		if paths == nil {
			return fmt.Errorf("%s properties of transformation function cannot be null", io)
		}
		if allowConstantTransformation && io == "input" {
			return nil
		}
		return fmt.Errorf("there are no %s properties in transformation function", io)
	}
	for _, p := range paths {
		if !types.IsValidPropertyPath(p) {
			return fmt.Errorf("transformation function %s property path %q is not valid", io, p)
		}
	}
	for i, p := range paths {
		for j, p2 := range paths {
			if i == j || len(p2) < len(p) {
				continue
			}
			if len(p2) == len(p) {
				if p == p2 {
					return fmt.Errorf("transformation function %s property path %q is repeated", io, p)
				}
				continue
			}
			// Check that p is not sub-paths of p2.
			if p2[len(p)] == '.' && p2[:len(p)] == p {
				return fmt.Errorf("transformation function %s paths cannot contain both %q and its sub-property path %q", io, p, p2)
			}
		}
	}
	if schema.Valid() {
		properties := schema.Properties()
		for _, p := range paths {
			if _, err := properties.ByPath(p); err != nil {
				return fmt.Errorf("%s property %q of transformation function does not exist in schema", io, p)
			}
		}
	}
	return nil
}
