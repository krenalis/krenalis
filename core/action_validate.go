//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package core

// This file contains the function "validateAction", as well as any support
// type, function and/or methods used exclusively by it.

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	"github.com/meergo/meergo/core/transformers/mappings"
	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

const (
	MaxFilePathSize             = 1024   // maximum allowed length for a file path.
	MaxFunctionSourceSize       = 50_000 // maximum allowed size for a transformation function source.
	MaxLastChangeTimeFormatSize = 64     // maximum allowed size for a last change time format.
	MaxQuerySize                = 1_000  // maximum allowed size for a database query.
	MaxTableNameSize            = 1024   // maximum allowed length for a database table name.
)

// validationState is a state for the validation of an action.
type validationState struct {

	// target is the action's target.
	target state.Target

	// connection is the action's connection.
	connection struct {
		role      state.Role
		connector struct {
			typ state.ConnectorType
		}
	}

	// format represents the action file format.
	//
	// If the action specify a format name, then this must be populated
	// according to that format, if exists, otherwise must be the empty
	// struct.
	format struct {
		typ         state.ConnectorType
		targets     state.ConnectorTargets
		hasSettings bool
		hasSheets   bool
	}

	// provider is the transformers.Provider instantiated on the Core.
	provider transformers.Provider
}

// validateActionToSet validates the given ActionToSet, in the context of the
// given validation state.
//
// It returns an errors.UnprocessableError error with code:
//
//   - FormatNotExist, if the action is on file and the specified format does
//     not exist.
//   - UnsupportedLanguage, if the transformation language is not supported.
func validateActionToSet(action ActionToSet, v validationState) error {

	inSchema := action.InSchema
	outSchema := action.OutSchema

	importEventsIntoWarehouse := isImportingEventsIntoWarehouse(v.connection.connector.typ, v.connection.role, v.target)
	dispatchEventsToApps := isDispatchingEventsToApps(v.connection.connector.typ, v.connection.role, v.target)
	importUserIdentitiesFromEvents := isImportingUserIdentitiesFromEvents(v.connection.connector.typ, v.connection.role, v.target)
	exportUsersToFile := isExportUsersToFile(v.connection.connector.typ, v.connection.role, v.target)

	allowConstantTransformation := importUserIdentitiesFromEvents || dispatchEventsToApps

	// In cases where the input schema refers to events, that is when:
	//
	//  - user identities are imported from events
	//  - events are imported into the data warehouse
	//  - events are dispatched to apps
	//
	// the input schema must be nil, which means the schema of the events.
	inSchemaIsEventSchema := importUserIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApps
	if inSchemaIsEventSchema {
		if inSchema.Valid() {
			switch {
			case importUserIdentitiesFromEvents:
				return errors.BadRequest("input schema must be invalid for actions that import user identities from events")
			case importEventsIntoWarehouse:
				return errors.BadRequest("input schema must be invalid for actions that import events into data warehouse")
			case dispatchEventsToApps:
				return errors.BadRequest("input schema must be invalid for actions that dispatch events to apps")
			}
		}
		inSchema = events.Schema
	}

	// Validate the action's connector.
	actionOnFile := v.connection.connector.typ == state.FileStorage
	if actionOnFile && action.Format == "" {
		return errors.BadRequest("actions on file storage connections must have a format")
	}
	if !actionOnFile && action.Format != "" {
		return errors.BadRequest("actions on %v connections cannot have a format", v.connection.connector.typ)
	}
	if action.Format != "" {
		if v.format.typ == 0 {
			return errors.Unprocessable(FormatNotExist, "format %q does not exist", action.Format)
		}
		if v.format.typ != state.File {
			return errors.BadRequest("format does not refer to a file connector")
		}
	}
	if actionOnFile && !v.format.targets.Contains(v.target) {
		return errors.BadRequest("target is not supported by the file format")
	}

	// First, do formal validations.

	// Validate the name.
	if err := util.ValidateStringField("name", action.Name, 60); err != nil {
		return errors.BadRequest("%s", err)
	}
	// Check that, if the schemas are valid, they have type Object.
	if inSchema.Valid() && inSchema.Kind() != types.ObjectKind {
		return errors.BadRequest("input schema, if provided, must be an object")
	}
	if outSchema.Valid() && outSchema.Kind() != types.ObjectKind {
		return errors.BadRequest("out schema, if provided, must be an object")
	}
	// Validate the filter.
	var usedInPaths []string
	if action.Filter != nil {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the filter")
		}
		properties, err := validateFilter(action.Filter, inSchema)
		if err != nil {
			return errors.BadRequest("filter is not valid: %w", err)
		}
		if !exportUsersToFile {
			usedInPaths = properties
		}
	}
	var usedOutPaths []string
	var mappingInPaths int
	if tr := action.Transformation; tr != nil {
		switch {
		case tr.Mapping != nil:
			if tr.Function != nil {
				return errors.BadRequest("action cannot have both transformation mapping and function")
			}
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
			if tr.Mapping != nil {
				return errors.BadRequest("action cannot have both transformation mapping and function")
			}
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
					return errors.Unprocessable(UnsupportedLanguage, "JavaScript transformation language is not supported")
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
			return errors.BadRequest("action cannot have a transformation without mapping and function.")
		}
	}
	// Validate the path.
	if action.Path != "" {
		if err := util.ValidateStringField("path", action.Path, MaxFilePathSize); err != nil {
			return errors.BadRequest("%s", err)
		}
		switch v.connection.role {
		case state.Source:
			_, err := connectors.ReplacePlaceholders(action.Path, func(_ string) (string, bool) {
				return "", false
			})
			if err != nil {
				return errors.BadRequest("placeholders syntax is not supported by source actions")
			}
		case state.Destination:
			_, err := connectors.ReplacePlaceholders(action.Path, func(name string) (string, bool) {
				name = strings.ToLower(name)
				return "", name == "today" || name == "now" || name == "unix"
			})
			if err != nil {
				return errors.BadRequest("path is not valid: %s", err)
			}
		}
	}
	// Validate the table name.
	if action.TableName != "" {
		if err := util.ValidateStringField("table name", action.TableName, MaxTableNameSize); err != nil {
			return errors.BadRequest("%s", err)
		}
	}
	// Validate the sheet.
	if action.Sheet != "" && !connectors.IsValidSheetName(action.Sheet) {
		return errors.BadRequest("sheet name is not valid")
	}
	// Validate the export mode.
	if action.ExportMode != "" {
		switch action.ExportMode {
		case CreateOnly, UpdateOnly, CreateOrUpdate:
		default:
			return errors.BadRequest("export mode %q is not valid", action.ExportMode)
		}
	}
	// Validate the matching properties.
	if action.Matching.In != "" || action.Matching.Out != "" {
		if action.Matching.In == "" {
			return errors.BadRequest("input matching property cannot be empty if output matching property is not empty")
		}
		if action.Matching.Out == "" {
			return errors.BadRequest("output matching property cannot be empty if input matching property is not empty")
		}
		if action.ExportMode == "" {
			return errors.BadRequest("export mode cannot be empty if there are matching properties")
		}
		// Validate the input matching property.
		if !types.IsValidPropertyName(action.Matching.In) {
			if types.IsValidPropertyPath(action.Matching.In) {
				return errors.BadRequest("matching properties cannot be property paths, can only be property names")
			}
			return errors.BadRequest("input matching property %q is not a valid property name", action.Matching.In)
		}
		if !inSchema.Valid() {
			return errors.BadRequest("input schema must be valid")
		}
		in, ok := inSchema.Property(action.Matching.In)
		if !ok {
			return errors.BadRequest("input matching property %q not found within the input schema", action.Matching.In)
		}
		if !canBeUsedAsMatchingProp(in.Type.Kind()) {
			return errors.BadRequest("type %s cannot be used as matching property", in.Type)
		}
		usedInPaths = append(usedInPaths, action.Matching.In)
		// Validate the output matching property.
		if !outSchema.Valid() {
			return errors.BadRequest("output schema must be valid")
		}
		if !types.IsValidPropertyName(action.Matching.Out) {
			if types.IsValidPropertyPath(action.Matching.Out) {
				return errors.BadRequest("matching properties cannot be property paths, can only be property names")
			}
			return errors.BadRequest("output matching property %q is not a valid property name", action.Matching.Out)
		}
		out, ok := outSchema.Property(action.Matching.Out)
		if !ok {
			return errors.BadRequest("output matching property %q not found within the output schema", action.Matching.Out)
		}
		if !canBeUsedAsMatchingProp(out.Type.Kind()) {
			return errors.BadRequest("type %s cannot be used as matching property", out.Type)
		}
		usedOutPaths = append(usedOutPaths, action.Matching.Out)
		// Check that the input property can be converted to the output property.
		switch in.Type.Kind() {
		case types.IntKind, types.UintKind:
			if k := out.Type.Kind(); k == types.UUIDKind {
				return errors.BadRequest("input matching property cannot be converted to the output matching property")
			}
		case types.UUIDKind:
			if k := out.Type.Kind(); k == types.IntKind || k == types.UintKind {
				return errors.BadRequest("input matching property cannot be converted to the output matching property")
			}
		}
		// Check that the output property has not been transformed.
		if tr := action.Transformation; tr != nil {
			if tr.Mapping != nil {
				if _, ok := tr.Mapping[action.Matching.Out]; ok {
					return errors.BadRequest("mapping cannot map over the output matching property")
				}
			} else {
				if slices.Contains(tr.Function.OutPaths, action.Matching.Out) {
					return errors.BadRequest("transformation function cannot transform over the output matching property")
				}
			}
		}
	}
	// Validate the compression.
	switch action.Compression {
	case NoCompression, ZipCompression, GzipCompression, SnappyCompression:
	default:
		return errors.BadRequest("compression %q is not valid", action.Compression)
	}
	// Validate the identity column.
	if action.IdentityColumn != "" {
		if !types.IsValidPropertyName(action.IdentityColumn) {
			return errors.BadRequest("identity column is not a valid property name")
		}
		if utf8.RuneCountInString(action.IdentityColumn) > 1024 {
			return errors.BadRequest("identity column is longer than 1024 runes")
		}
	}
	// Validate the last change time property.
	if action.LastChangeTimeProperty != "" {
		if !types.IsValidPropertyName(action.LastChangeTimeProperty) {
			return errors.BadRequest("last change time property is a not valid property name")
		}
		if utf8.RuneCountInString(action.LastChangeTimeProperty) > 1024 {
			return errors.BadRequest("last change time property is longer than 1024 runes")
		}
	}
	// Validate the last change time format.
	if action.LastChangeTimeFormat != "" {
		if err := validateLastChangeTimeFormat(action.LastChangeTimeFormat); err != nil {
			return errors.BadRequest("%s", err)
		}
	}
	// Validate the "order by" property path.
	if action.OrderBy != "" {
		if !types.IsValidPropertyPath(action.OrderBy) {
			return errors.BadRequest("the specified order by is not a valid property path")
		}
		if utf8.RuneCountInString(action.OrderBy) > 1024 {
			return errors.BadRequest("the specified order by is longer than 1024 runes")
		}
	}

	// Second, do validations based on the workspace and the connection.

	if importEventsIntoWarehouse && outSchema.Valid() {
		return errors.BadRequest("output schema must be invalid when importing events into data warehouse")
	}

	// Do some validations on the input and the output schemas.
	if inSchema.Valid() && !inSchemaIsEventSchema {
		if err := validateActionSchema("input", inSchema, v.connection.role, v.target, v.connection.connector.typ, action.TableKey); err != nil {
			return errors.BadRequest("%s", err)
		}
	}
	if outSchema.Valid() {
		if err := validateActionSchema("output", outSchema, v.connection.role, v.target, v.connection.connector.typ, action.TableKey); err != nil {
			return errors.BadRequest("%s", err)
		}
	}

	// Check if the settings are allowed and are a JSON Object.
	if v.connection.connector.typ == state.FileStorage {
		if action.FormatSettings == nil {
			if v.format.hasSettings {
				return errors.BadRequest("format settings must be provided because format %s has %s settings", action.Format, strings.ToLower(v.connection.role.String()))
			}
		} else {
			if !v.format.hasSettings {
				return errors.BadRequest("format settings cannot be provided because format %s has no %s settings", action.Format, strings.ToLower(v.connection.role.String()))
			}
			if !json.Valid(action.FormatSettings) || !action.FormatSettings.IsObject() {
				return errors.BadRequest("format settings are not a valid JSON Object")
			}
		}
	} else if action.FormatSettings != nil {
		return errors.BadRequest("%s actions cannot have %s format settings", strings.ToLower(v.connection.connector.typ.String()), strings.ToLower(v.connection.role.String()))
	}

	// Check if the compression is allowed.
	if action.Compression != NoCompression && v.connection.connector.typ != state.FileStorage {
		return errors.BadRequest("%s actions cannot have compression", strings.ToLower(v.connection.connector.typ.String()))
	}

	// Check if the query is allowed.
	if needsQuery := v.connection.connector.typ == state.Database && v.connection.role == state.Source; needsQuery {
		if err := util.ValidateStringField("query", action.Query, MaxQuerySize); err != nil {
			return errors.BadRequest("%s", err)
		}
	} else {
		if action.Query != "" {
			return errors.BadRequest("%s actions cannot have a query", v.connection.connector.typ)
		}
	}

	// Check if the filters are allowed.
	// Note that filters are always allowed except for actions that import users
	// from databases.
	filtersAllowed := !(v.connection.role == state.Source &&
		v.connection.connector.typ == state.Database && v.target == state.Users)
	if action.Filter != nil && !filtersAllowed {
		return errors.BadRequest("filters are not allowed")
	}

	// Check if the path and the sheet are allowed.
	if v.connection.connector.typ == state.FileStorage {
		if action.Path == "" {
			return errors.BadRequest("path cannot be empty for actions on storage connections")
		}
		if v.format.hasSheets && action.Sheet == "" {
			return errors.BadRequest("sheet cannot be empty because format %s has sheets", action.Format)
		}
		if !v.format.hasSheets && action.Sheet != "" {
			return errors.BadRequest("format %s does not have sheets", action.Format)
		}
	} else {
		if action.Path != "" {
			return errors.BadRequest("%s actions cannot have a path", v.connection.connector.typ)
		}
		if action.Sheet != "" {
			return errors.BadRequest("%s actions cannot have a sheet", v.connection.connector.typ)
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
		if action.IdentityColumn == "" {
			return errors.BadRequest("identity column is mandatory")
		}
		identityColumn, ok := inSchema.Property(action.IdentityColumn)
		if !ok {
			return errors.BadRequest("identity column %q not found within input schema", action.IdentityColumn)
		}
		switch k := identityColumn.Type.Kind(); k {
		case types.IntKind, types.UintKind, types.UUIDKind, types.JSONKind, types.TextKind:
		default:
			return errors.BadRequest("identity column %q has kind %s instead of Int, Uint, UUID, JSON, or Text", action.IdentityColumn, k)
		}
		if identityColumn.ReadOptional {
			return errors.BadRequest("identity column cannot be optional")
		}
		usedInPaths = append(usedInPaths, action.IdentityColumn)
		// Validate the last change time property and format.
		var requiresLastChangeTimeFormat bool
		if action.LastChangeTimeProperty != "" {
			lastChangeTime, ok := inSchema.Property(action.LastChangeTimeProperty)
			if !ok {
				return errors.BadRequest("last change time property %q not found within input schema", action.LastChangeTimeProperty)
			}
			switch k := lastChangeTime.Type.Kind(); k {
			case types.DateTimeKind, types.DateKind:
			case types.JSONKind, types.TextKind:
				requiresLastChangeTimeFormat = true
			default:
				return errors.BadRequest("last change time property %q has kind %s instead of DateTime, Date, JSON, or Text", action.LastChangeTimeProperty, k)
			}
			usedInPaths = append(usedInPaths, action.LastChangeTimeProperty)
		}
		if !requiresLastChangeTimeFormat && action.LastChangeTimeFormat != "" {
			return errors.BadRequest("action cannot specify a last change time format")
		} else if requiresLastChangeTimeFormat {
			if action.LastChangeTimeFormat == "" {
				return errors.BadRequest("last change time format is required")
			}
			if v.connection.connector.typ == state.Database && action.LastChangeTimeFormat == "Excel" {
				return errors.BadRequest("last change time format cannot be Excel for database actions")
			}
		}
	} else {
		if action.IdentityColumn != "" {
			return errors.BadRequest("action cannot specify an identity column")
		}
		if action.LastChangeTimeProperty != "" {
			return errors.BadRequest("action cannot specify a last change time property")
		}
		if action.LastChangeTimeFormat != "" {
			return errors.BadRequest("action cannot specify a last change time format")
		}
	}

	// Do some checks related to exporting users to files.
	if exportUsersToFile {
		// When exporting users to file, ensure that the input schema is valid,
		// as it contains the properties that will be exported to the file.
		if !inSchema.Valid() {
			return errors.BadRequest("input schema must be valid when exporting users to file")
		}
		if outSchema.Valid() {
			return errors.BadRequest("output schema must be invalid when exporting users to file")
		}
		// Check that OrderBy is defined and exists in the out
		// schema.
		if action.OrderBy == "" {
			return errors.BadRequest("order by property cannot be empty when exporting users to file")
		}
		p, err := types.PropertyByPath(inSchema, action.OrderBy)
		if err != nil {
			return errors.BadRequest("order by property cannot be found in action's input schema: %s", err)
		}
		// Check the allowed types.
		// Regarding the allowed types, we can use the same criterion used for
		// the allowed types of the workspace identifiers, so as to simplify the
		// specifications for the warehouse drivers.
		switch p.Type.Kind() {
		case types.IntKind, types.UintKind, types.UUIDKind, types.InetKind, types.TextKind:
			// Ok.
		case types.DecimalKind:
			if p.Type.Precision() != 0 {
				return errors.BadRequest("the Decimal type of the order by property cannot have a precision greater than 0")
			}
		default:
			return errors.BadRequest("order by property cannot have kind %s", p.Type.Kind())
		}
	} else {
		if action.OrderBy != "" {
			return errors.BadRequest("actions that do not export users to files cannot specify a order by property")
		}
	}

	// Do some checks related to exporting users to databases.
	exportUsersToDatabase := v.connection.connector.typ == state.Database && v.connection.role == state.Destination && v.target == state.Users
	if exportUsersToDatabase {
		if action.TableName == "" {
			return errors.BadRequest("table name cannot be empty for destination database actions")
		}
		if action.TableKey == "" {
			return errors.BadRequest("table key cannot be empty for destination database actions")
		}
		if !types.IsValidPropertyName(action.TableKey) {
			return errors.BadRequest("table key is not a valid property name")
		}
		if !outSchema.Valid() {
			return errors.BadRequest("out schema must be valid")
		}
		p, ok := outSchema.Property(action.TableKey)
		if !ok {
			return errors.BadRequest("table key %q not found within output schema", action.TableKey)
		}
		if !canBeUsedAsTableKey(p.Type.Kind()) {
			return errors.BadRequest("type %s cannot be used as table key", p.Type)
		}
		if tr := action.Transformation; tr != nil {
			if tr.Mapping != nil {
				if _, ok := tr.Mapping[action.TableKey]; !ok {
					return errors.BadRequest("an expression must be mapped to the table key")
				}
			} else {
				if !slices.Contains(tr.Function.OutPaths, action.TableKey) {
					return errors.BadRequest("the out properties of the transformation function must contain the table key")
				}
			}
		}
	} else {
		if action.TableName != "" {
			return errors.BadRequest("table name is not allowed")
		}
		if action.TableKey != "" {
			return errors.BadRequest("table key is not allowed")
		}
	}

	// Check if the export options are needed.
	needsExportOptions := v.connection.connector.typ == state.App &&
		v.connection.role == state.Destination && v.target == state.Users
	if needsExportOptions {
		if action.ExportMode == "" {
			return errors.BadRequest("export mode cannot be empty")
		}
		if action.Matching.In == "" {
			return errors.BadRequest("matching properties must be provided")
		}
	} else {
		if action.ExportMode != "" {
			return errors.BadRequest("export mode must be empty")
		}
		if action.Matching.In != "" {
			return errors.BadRequest("matching properties cannot be provided")
		}
	}

	eventBasedConn := v.connection.connector.typ == state.Mobile ||
		v.connection.connector.typ == state.Server ||
		v.connection.connector.typ == state.Website

	// Check the connections for which the transformation is prohibited.
	targetUsersOrGroups := v.target == state.Users || v.target == state.Groups
	transformationProhibited := (v.connection.role == state.Source && eventBasedConn && v.target == state.Events) ||
		(v.connection.role == state.Destination && v.connection.connector.typ == state.FileStorage && targetUsersOrGroups)
	if transformationProhibited && action.Transformation != nil {
		return errors.BadRequest("action cannot have a transformation")
	}

	// Check if the transformation is mandatory, with at least one input
	// property.
	transformationMandatory := targetUsersOrGroups &&
		(v.connection.connector.typ == state.App || v.connection.connector.typ == state.Database ||
			(v.connection.role == state.Source && v.connection.connector.typ == state.FileStorage))
	if transformationMandatory && action.Transformation == nil {
		return errors.BadRequest("action must have a transformation")
	}

	// If constant transformations are not allowed, there must be at least one
	// property used as input to the transformation, either in mappings or
	// functions.
	if tr := action.Transformation; tr != nil && !allowConstantTransformation {
		if tr.Mapping != nil && mappingInPaths == 0 {
			return errors.BadRequest("transformation must map at least one property")
		}
		if tr.Function != nil && len(action.Transformation.Function.InPaths) == 0 {
			return errors.BadRequest("transformation function must have at least one input property")
		}
	}

	// Ensure that every property in the input and output schemas have been used
	// (by the mappings, by the filters, etc...).
	if importUserIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApps {
		// In these cases the input schema is the full schema of the events,
		// both in case of mappings and transformation, so we cannot return the
		// error about unused properties in input schema because just a minor
		// part of them is generally used.
		if usedOutPaths != nil {
			if props := unusedProperties(outSchema, usedOutPaths); props != nil {
				return errors.BadRequest("output schema contains unused properties: %s", strings.Join(props, ", "))
			}
		}
	} else {
		if usedInPaths != nil {
			if props := unusedProperties(inSchema, usedInPaths); props != nil {
				return errors.BadRequest("input schema contains unused properties: %s", strings.Join(props, ", "))
			}
		}
		if usedOutPaths != nil {
			if props := unusedProperties(outSchema, usedOutPaths); props != nil {
				return errors.BadRequest("output schema contains unused properties: %s", strings.Join(props, ", "))
			}
		}
	}

	return nil
}

// canBeUsedAsMatchingProp reports whether a type with kind k can be used as a
// matching property when exporting users to an app.
func canBeUsedAsMatchingProp(k types.Kind) bool {
	// Only integers, UUIDs and texts are allowed.
	return k == types.IntKind || k == types.UintKind || k == types.UUIDKind || k == types.TextKind
}

// canBeUsedAsTableKey reports whether a type with kind k can be used as a
// table key when exporting users to databases.
func canBeUsedAsTableKey(k types.Kind) bool {
	// Only integers, UUIDs and texts are allowed.
	return k == types.IntKind || k == types.UintKind || k == types.UUIDKind || k == types.TextKind
}

// unusedProperties returns the names of the unused properties in schema, if
// there is at least one, otherwise returns nil. schema must be valid.
func unusedProperties(schema types.Type, used []string) []string {
	isUsed := make(map[string]bool, len(used))
	for _, p := range used {
		name, _, _ := strings.Cut(p, ".")
		isUsed[name] = true
	}
	var unused []string
	for _, p := range schema.Properties() {
		if isUsed[p.Name] {
			continue
		}
		if unused == nil {
			unused = []string{p.Name}
		} else {
			unused = append(unused, p.Name)
		}
	}
	slices.Sort(unused)
	return unused
}

// validateActionSchema validates an action schema, returning an error if it is
// not valid. It is not called if schema is the event schema.
//
// io specifies whether the validation relates to "input" or "output", schema is
// the schema of the input or output action, role and target are the role and
// target of the action, and typ is the action's connection type.
func validateActionSchema(io string, schema types.Type, role state.Role, target state.Target, typ state.ConnectorType, tableKey string) error {

	isUserSchema := target == state.Users &&
		(io == "input" && role == state.Destination || io == "output" && role == state.Source)

	for path, p := range types.WalkAll(schema) {
		if p.Placeholder != "" {
			return fmt.Errorf("%s action schema property %q has a placeholder, but action schema properties cannot have placeholders", io, path)
		}
		if isUserSchema {
			if isMetaProperty(path) {
				return fmt.Errorf("%s action schema property %q is a meta property", io, path)
			}
			if k := p.Type.Kind(); k == types.ArrayKind || k == types.MapKind {
				elemK := p.Type.Elem().Kind()
				if elemK == types.ArrayKind || elemK == types.ObjectKind || elemK == types.MapKind {
					return fmt.Errorf("%s action schema property %q cannot have type %s(%s)", io, path, k, elemK)
				}
			}
			if p.CreateRequired {
				return fmt.Errorf("%s action schema property %q cannot have CreateRequired set to true", io, path)
			}
			if p.UpdateRequired {
				return fmt.Errorf("%s action schema property %q cannot have UpdateRequired set to true", io, path)
			}
			if !p.ReadOptional {
				return fmt.Errorf("%s action schema property %q must have ReadOptional set to true", io, path)
			}
			if p.Nullable {
				return fmt.Errorf("%s action schema property %q cannot have Nullable set to true", io, path)
			}
			continue
		}
		if role == state.Source && io == "input" {
			if p.CreateRequired {
				return fmt.Errorf("source action schema property %q cannot have CreateRequired set to true", path)
			}
			if p.UpdateRequired {
				return fmt.Errorf("%s action schema property %q cannot have UpdateRequired set to true", io, path)
			}
			if p.ReadOptional && typ == state.Database {
				return fmt.Errorf("%s action schema property %q cannot have ReadOptional set to true", io, path)
			}
			continue
		}
		if role == state.Destination && io == "output" {
			switch {
			case typ == state.App && target == state.Events:
				if p.UpdateRequired {
					return fmt.Errorf("output action schema property %q cannot have UpdateRequired set to true", path)
				}
			case typ == state.Database:
				if p.CreateRequired {
					return fmt.Errorf("output action schema property %q cannot have CreateRequired set to true", path)
				}
				if p.UpdateRequired {
					return fmt.Errorf("output action schema property %q cannot have UpdateRequired set to true", path)
				}
				if p.Nullable && path == tableKey {
					return fmt.Errorf("output action schema property %q cannot have Nullable set to true", path)
				}
			}
			if p.ReadOptional {
				return fmt.Errorf("output action schema property %q cannot have ReadOptional set to true", path)
			}
		}
	}

	if isUserSchema {
		if err := datastore.CheckConflictingProperties(io, schema); err != nil {
			return err
		}
	}

	return nil
}

// validateLastChangeTimeFormat validates the given last change time format for
// importing files and database rows, returning an error in case the format is
// not valid.
//
// Valid formats are
//
//   - "ISO8601": the ISO 8601 format
//   - "Excel": the Excel format, a float value stored in an Excel cell representing a date/datetime
//   - a string containing a '%' character: the strftime() function format
//
// NOTE: keep in sync with the function
// 'core/connectors.parseLastChangeTimePropertyWithFormat'.
func validateLastChangeTimeFormat(format string) error {
	switch format {
	case
		"ISO8601",
		"Excel":
		return nil
	}
	if err := util.ValidateStringField("last change time format", format, MaxLastChangeTimeFormatSize); err != nil {
		return err
	}
	if !strings.Contains(format, "%") {
		return fmt.Errorf("last change time format %q is not valid", format)
	}
	return nil
}

// validateTransformationFunctionPaths validates the transformation function
// paths of an action.
//
// io specifies whether the validation relates to "input" or "output", schema is
// the schema of the input or output action, paths are the function paths for
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
//   - paths cannot "cross" Array and Map elements, but only Object, so it is
//     possible to refer to Array and Map properties only as a whole, not to
//     their specific elements.
//
// It panics if the schema is valid and is not an Object.
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
	has := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if !types.IsValidPropertyPath(path) {
			return fmt.Errorf("transformation function %s property path %q is not valid", io, path)
		}
		if _, ok := has[path]; ok {
			return fmt.Errorf("transformation function %s property path %q is repeated", io, path)
		}
		for _, path2 := range slices.Sorted(maps.Keys(has)) {
			if strings.HasPrefix(path, path2) || strings.HasPrefix(path2, path) {
				if len(path2) < len(path) {
					path, path2 = path2, path
				}
				return fmt.Errorf("transformation function %s paths cannot contain both %q and its sub-property path %q", io, path, path2)
			}
		}
		has[path] = struct{}{}
	}
	if schema.Valid() {
		for path := range types.WalkObjects(schema) {
			delete(has, path)
		}
	}
	if len(has) > 0 {
		for _, path := range paths {
			if _, ok := has[path]; ok {
				return fmt.Errorf("%s property %q of transformation function does not exist in schema", io, path)
			}
		}
	}
	return nil
}
