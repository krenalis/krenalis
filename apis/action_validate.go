//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package apis

// This file contains the method "validateActionToSet", as well as any support
// function/method used exclusively by it.

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/events"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers"
	"github.com/open2b/chichi/apis/transformers/mappings"
	"github.com/open2b/chichi/types"

	"golang.org/x/exp/maps"
)

// validateActionToSet validates the action to set (when adding or setting an
// action) for the given target, on the connection c.
//
// fileConnector must be passed exclusively and necessarily when the connector
// of the storage has type FileStorage, otherwise it must be nil.
//
// tr is the transformers.Function instantiated on the APIs.
//
// Refer to the specifications in the file "apis/Actions.md" for more details.
//
// It returns an errors.UnprocessableError error with code LanguageNotSupported,
// if the transformation language is not supported.
func validateActionToSet(action ActionToSet, target state.Target, c *state.Connection, fileConnector *state.Connector, tr transformers.Function) error {

	inSchema := action.InSchema
	outSchema := action.OutSchema

	importUsersIdentitiesFromEvents := importsUsersIdentitiesFromEvents(c.Connector().Type, c.Role, target)
	if importUsersIdentitiesFromEvents {
		if inSchema.Valid() {
			return errors.BadRequest("input schema must be invalid for actions that import users identities from events")
		}
		// The input schema is the events schema without GID because this
		// actions imports users identities from incoming events, which,
		// clearly, still do not have any user associated.
		inSchema = events.Schema
	}

	// First, do formal validations.

	// Validate the name.
	if action.Name == "" {
		return errors.BadRequest("name is empty")
	}
	if !utf8.ValidString(action.Name) {
		return errors.BadRequest("name is not UTF-8 encoded")
	}
	if n := utf8.RuneCountInString(action.Name); n > 60 {
		return errors.BadRequest("name is longer than 60 runes")
	}
	// Validate the schemas.
	if inSchema.Valid() && inSchema.Kind() != types.ObjectKind {
		return errors.BadRequest("input schema, if provided, must be an object")
	}
	if outSchema.Valid() && outSchema.Kind() != types.ObjectKind {
		return errors.BadRequest("out schema, if provided, must be an object")
	}
	// Validate the filter.
	var usedInPaths []types.Path
	if action.Filter != nil {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the filter")
		}
		properties, err := validateFilter(action.Filter, inSchema)
		if err != nil {
			return errors.BadRequest("filter is not valid: %w", err)
		}
		usedInPaths = properties
	}
	// An action cannot have both mappings and transformations.
	if action.Transformation.Mapping != nil && action.Transformation.Function != nil {
		return errors.BadRequest("action cannot have both mappings and transformation")
	}
	// Validate the mapping.
	var usedOutPaths []types.Path
	var mappingInProperties int
	if mapping := action.Transformation.Mapping; mapping != nil {
		if len(mapping) == 0 {
			return errors.BadRequest("transformation mapping must have mapped properties")
		}
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the mapping")
		}
		if !outSchema.Valid() {
			return errors.BadRequest("output schema is required by the mapping")
		}
		transformer, err := mappings.New(mapping, inSchema, outSchema, nil)
		if err != nil {
			return errors.BadRequest("invalid mapping: %s", err)
		}
		// Input properties.
		inProps := transformer.Properties()
		mappingInProperties = len(inProps)
		usedInPaths = append(usedInPaths, inProps...)
		// Output properties.
		for m := range mapping {
			path, err := types.ParsePropertyPath(m)
			if err != nil {
				return errors.BadRequest("invalid property path %q", m)
			}
			usedOutPaths = append(usedOutPaths, path)
		}
	}
	// Validate the transformation.
	if function := action.Transformation.Function; function != nil {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema is required by the transformation")
		}
		if !outSchema.Valid() {
			return errors.BadRequest("output schema is required by the transformation")
		}
		if function.Source == "" {
			return errors.BadRequest("function transformation source is empty")
		}
		switch function.Language {
		case "JavaScript":
			if tr == nil || !tr.SupportLanguage(state.JavaScript) {
				return errors.Unprocessable(LanguageNotSupported, "JavaScript transformation language is not supported")
			}
		case "Python":
			if tr == nil || !tr.SupportLanguage(state.Python) {
				return errors.Unprocessable(LanguageNotSupported, "Python transformation language is not supported")
			}
		case "":
			return errors.BadRequest("transformation language is empty")
		default:
			return errors.BadRequest("transformation language %q is not valid", action.Transformation.Function.Language)
		}
	}
	// Validate the path.
	if action.Path != "" {
		if !utf8.ValidString(action.Path) {
			return errors.BadRequest("path is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(action.Path); n > 1024 {
			return errors.BadRequest("path is longer than 1024 runes")
		}
		switch c.Role {
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
		if !utf8.ValidString(action.TableName) {
			return errors.BadRequest("table name is not UTF-8 encoded")
		}
		if n := utf8.RuneCountInString(action.TableName); n > 1024 {
			return errors.BadRequest("table name is longer than 1024 runes")
		}
	}
	// Validate the sheet.
	if action.Sheet != "" && !connectors.IsValidSheetName(action.Sheet) {
		return errors.BadRequest("sheet name is not valid")
	}
	// Validate the export options.
	if action.ExportMode != nil {
		switch *action.ExportMode {
		case CreateOnly, UpdateOnly, CreateOrUpdate:
		default:
			return errors.BadRequest("export mode %q is not valid", *action.ExportMode)
		}
	}
	if action.MatchingProperties != nil {
		props := *action.MatchingProperties
		// Validate the internal matching property.
		if !types.IsValidPropertyName(props.Internal) {
			return errors.BadRequest("internal matching property %q is not a valid property name", props.Internal)
		}
		if !inSchema.Valid() {
			return errors.BadRequest("input schema must be valid")
		}
		internal, ok := inSchema.Property(props.Internal)
		if !ok {
			return errors.BadRequest("internal matching property %q not found within the input schema", props.Internal)
		}
		if !canBeUsedAsAsMatchingProp(internal.Type.Kind()) {
			return errors.BadRequest("type %s cannot be used as matching property", internal.Type)
		}
		usedInPaths = append(usedInPaths, types.Path{props.Internal})
		// Validate the external matching property.
		if !types.IsValidPropertyName(props.External.Name) {
			return errors.BadRequest("external matching property %q is not a valid property name", props.External.Name)
		}
		if !props.External.Type.Valid() {
			return errors.BadRequest("external matching property type is not valid")
		}
		if !canBeUsedAsAsMatchingProp(props.External.Type.Kind()) {
			return errors.BadRequest("type %s cannot be used as matching property", props.External.Type)
		}
	}
	// Validate the compression.
	switch action.Compression {
	case NoCompression, ZipCompression, GzipCompression, SnappyCompression:
	default:
		return errors.BadRequest("compression %q is not valid", action.Compression)
	}
	// Validate the identity property.
	if action.IdentityProperty != "" {
		if !types.IsValidPropertyName(action.IdentityProperty) {
			return errors.BadRequest("identity property is not a valid property name")
		}
		if utf8.RuneCountInString(action.IdentityProperty) > 1024 {
			return errors.BadRequest("identity property is longer than 1024 runes")
		}
	}
	// Validate the displayed property.
	if action.DisplayedProperty != "" {
		if !types.IsValidPropertyName(action.DisplayedProperty) {
			return errors.BadRequest("displayed property is not a valid property name")
		}
		if n := utf8.RuneCountInString(action.DisplayedProperty); n > 1024 {
			return errors.BadRequest("displayed property is longer than 1024 runes")
		}
	}
	// Validate the 'updated at' column.
	if action.UpdatedAtColumn != "" {
		if !types.IsValidPropertyName(action.UpdatedAtColumn) {
			return errors.BadRequest("'updated at' column is a not valid property name")
		}
		if utf8.RuneCountInString(action.UpdatedAtColumn) > 1024 {
			return errors.BadRequest("'updated at' column is longer than 1024 runes")
		}
	}
	// Validate the 'updated at' format.
	if action.UpdatedAtFormat != "" {
		if err := validateTimestampFormat(action.UpdatedAtFormat); err != nil {
			return errors.BadRequest(err.Error())
		}
	}

	// Second, do validations based on the workspace and the connection.

	connector := c.Connector()
	eventBasedConn := connector.Type == state.MobileType ||
		connector.Type == state.ServerType ||
		connector.Type == state.WebsiteType

	// In case of a source connection, since its actions write on the data
	// warehouse, the output schema cannot contain meta properties because such
	// properties are not writable by user transformations.
	if c.Role == state.Source && outSchema.Valid() {
		for _, p := range outSchema.Properties() {
			if isMetaProperty(p.Name) {
				return errors.BadRequest("output schema cannot contain meta properties")
			}
		}
	}

	// Validate the UI values.
	if fileConnector == nil {
		if action.UIValues != nil {
			return errors.BadRequest("UI values cannot be provided because %s actions have no UI", strings.ToLower(connector.Type.String()))
		}
	} else {
		if fileConnector.HasUI {
			if action.UIValues == nil {
				return errors.BadRequest("UI values must be provided because connector %s has a UI", fileConnector.Name)
			}
			if !isJSONObject(action.UIValues) {
				return errors.BadRequest("UI values are not a valid JSON Object")
			}
		} else if action.UIValues != nil {
			return errors.BadRequest("UI values cannot be provided because connector %s has no UI", fileConnector.Name)
		}
	}

	// Check if the UI values and the compression are allowed.
	if connector.Type == state.FileStorageType {
		if !fileConnector.HasUI {
			return errors.BadRequest("UI values cannot be provided because connector %s has no UI", fileConnector.Name)
		}
	} else {
		if action.UIValues != nil {
			return errors.BadRequest("UI values cannot be provided because %s actions has no UI", strings.ToLower(connector.Type.String()))
		}
		if action.Compression != NoCompression {
			return errors.BadRequest("actions on %s connections cannot have a compression", strings.ToLower(connector.Type.String()))
		}
	}

	// Check if the query is allowed.
	if needsQuery := connector.Type == state.DatabaseType && c.Role == state.Source; needsQuery {
		if action.Query == "" {
			return errors.BadRequest("query cannot be empty for database actions")
		}
	} else {
		if action.Query != "" {
			return errors.BadRequest("%s actions cannot have a query", connector.Type)
		}
	}

	// Check if the filters are allowed.
	targetUsersOrGroups := target == state.Users || target == state.Groups
	var filtersAllowed bool
	switch connector.Type {
	case state.AppType:
		filtersAllowed = c.Role == state.Destination
	case state.DatabaseType:
		filtersAllowed = c.Role == state.Destination
	case state.FileStorageType:
		filtersAllowed = targetUsersOrGroups && c.Role == state.Destination
	}
	if action.Filter != nil && !filtersAllowed {
		return errors.BadRequest("filters are not allowed")
	}

	// Check if the path and the sheet are allowed.
	if connector.Type == state.FileStorageType {
		if action.Path == "" {
			return errors.BadRequest("path cannot be empty for actions on storage connections")
		}
		if fileConnector.HasSheets && action.Sheet == "" {
			return errors.BadRequest("sheet cannot be empty because connector %d has sheets", fileConnector.ID)
		}
		if !fileConnector.HasSheets && action.Sheet != "" {
			return errors.BadRequest("connector %d does not have sheets", fileConnector.ID)
		}
	} else {
		if action.Path != "" {
			return errors.BadRequest("%s actions cannot have a path", connector.Type)
		}
		if action.Sheet != "" {
			return errors.BadRequest("%s actions cannot have a sheet", connector.Type)
		}
	}

	// Check the column for the identity property and for the timestamp.
	importFromColumns := c.Role == state.Source &&
		(connector.Type == state.DatabaseType || connector.Type == state.FileStorageType)
	if importFromColumns {
		if !inSchema.Valid() {
			return errors.BadRequest("input schema must be valid")
		}
		// Validate the identity property.
		if action.IdentityProperty == "" {
			return errors.BadRequest("identity property is mandatory")
		}
		identityProperty, ok := inSchema.Property(action.IdentityProperty)
		if !ok {
			return errors.BadRequest("identity property %q not found within input schema", action.IdentityProperty)
		}
		switch k := identityProperty.Type.Kind(); k {
		case types.IntKind, types.UintKind, types.UUIDKind, types.JSONKind, types.TextKind:
		default:
			return errors.BadRequest("identity property %q has kind %s instead of Int, Uint, UUID, JSON, or Text", action.IdentityProperty, k)
		}
		usedInPaths = append(usedInPaths, types.Path{action.IdentityProperty})
		// Validate the 'updated at' column and format.
		var requiresUpdatedAtFormat bool
		if action.UpdatedAtColumn != "" {
			updatedAtColumn, ok := inSchema.Property(action.UpdatedAtColumn)
			if !ok {
				return errors.BadRequest("'updated at' column %q not found within input schema", action.UpdatedAtColumn)
			}
			switch k := updatedAtColumn.Type.Kind(); k {
			case types.DateTimeKind, types.DateKind:
			case types.JSONKind, types.TextKind:
				requiresUpdatedAtFormat = true
			default:
				return errors.BadRequest("'updated at' column %q has kind %s instead of DateTime, Date, JSON, or Text", action.UpdatedAtColumn, k)
			}
			usedInPaths = append(usedInPaths, types.Path{action.UpdatedAtColumn})
		}
		if !requiresUpdatedAtFormat && action.UpdatedAtFormat != "" {
			return errors.BadRequest("action cannot specify an 'updated at' format")
		} else if requiresUpdatedAtFormat && action.UpdatedAtFormat == "" {
			return errors.BadRequest("'updated at' format is required")
		}
	} else {
		if action.IdentityProperty != "" {
			return errors.BadRequest("action cannot specify an identity property")
		}
		if action.UpdatedAtColumn != "" {
			return errors.BadRequest("action cannot specify an 'updated at' column")
		}
		if action.UpdatedAtFormat != "" {
			return errors.BadRequest("action cannot specify an 'updated at' format")
		}
	}

	// Validate the displayed property.
	if action.DisplayedProperty != "" {
		if c.Role != state.Source {
			return errors.BadRequest("destination actions cannot have a displayed property")
		}
		if t := connector.Type; t == state.StreamType {
			return errors.BadRequest("%s actions cannot have a displayed property", strings.ToLower(t.String()))
		} else if eventBasedConn && target == state.Events {
			return errors.BadRequest("%s actions importing events cannot have a displayed property", strings.ToLower(target.String()))
		}
	}

	// When exporting users to file, ensure that the output schema is valid, as
	// it contains the properties that will be exported to the file.
	if connector.Type == state.FileStorageType && c.Role == state.Destination && target == state.Users {
		if !outSchema.Valid() {
			return errors.BadRequest("output schema cannot be empty when exporting users to file")
		}
	}

	// Check if the table name is allowed.
	needsTableName := connector.Type == state.DatabaseType && c.Role == state.Destination
	if needsTableName && action.TableName == "" {
		return errors.BadRequest("table name cannot be empty for destination database actions")
	} else if !needsTableName && action.TableName != "" {
		return errors.BadRequest("table name is not allowed")
	}

	// Check if the export options are needed.
	needsExportOptions := connector.Type == state.AppType &&
		c.Role == state.Destination && target == state.Users
	if needsExportOptions {
		if action.ExportMode == nil {
			return errors.BadRequest("export mode cannot be nil")
		}
		if action.MatchingProperties == nil {
			return errors.BadRequest("matching properties cannot be nil")
		}
		if action.ExportOnDuplicatedUsers == nil {
			return errors.BadRequest("export on duplicated users setting cannot be nil")
		}
	} else {
		if action.ExportMode != nil {
			return errors.BadRequest("export mode must be nil")
		}
		if action.MatchingProperties != nil {
			return errors.BadRequest("matching properties must be nil")
		}
		if action.ExportOnDuplicatedUsers != nil {
			return errors.BadRequest("export on duplicated users setting must be nil")
		}
	}

	// Check the connections for which the transformation is prohibited.
	transformationProhibited := (c.Role == state.Source && eventBasedConn && target == state.Events) ||
		(c.Role == state.Destination && connector.Type == state.FileStorageType && targetUsersOrGroups)
	haveTransformation := action.Transformation.Mapping != nil || action.Transformation.Function != nil
	if transformationProhibited && haveTransformation {
		return errors.BadRequest("action cannot have a transformation")
	}

	// Check the connections for which the transformation function is
	// prohibited.
	if haveTransformation {
		funcForbidden := c.Role == state.Source && eventBasedConn && targetUsersOrGroups
		if funcForbidden && action.Transformation.Function != nil {
			return errors.BadRequest("action cannot have a transformation function")
		}
	}

	// Check if the transformation is mandatory, with at least one input
	// property.
	//
	// For mappings, at least one property path must appear in the input
	// expressions.
	//
	// For transformation functions, since every property of the input schema is
	// passed to the function, the input schema must be valid (thus it must
	// contain at least one property).
	transformationMandatory := targetUsersOrGroups &&
		(connector.Type == state.AppType || connector.Type == state.DatabaseType ||
			(c.Role == state.Source && connector.Type == state.FileStorageType))
	if transformationMandatory && !haveTransformation {
		return errors.BadRequest("action must have a transformation")
	}
	if action.Transformation.Mapping != nil && mappingInProperties == 0 {
		return errors.BadRequest("transformation must map at least one property")
	}
	if action.Transformation.Function != nil && !inSchema.Valid() {
		return errors.BadRequest("transformation function must have at least one input property")
	}

	// Ensure that every property in the input and output schemas have been used
	// (by the mappings, by the filters, etc...).
	if action.Transformation.Function != nil {
		// The action has a transformation function, so we do not know which
		// properties are used; consequently, this check would always pass
		// because we would consider every property of the schema as used.
	} else if importUsersIdentitiesFromEvents {
		// In this case the input schema is the full schema of the events, both
		// in case of mappings and transformation, so we cannot return the error
		// about unused properties in input schema because just a minor part of
		// them is generally used.
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

// unusedProperties returns the names of the unused properties in schema, if
// there is at least one, otherwise returns nil. schema must be valid.
func unusedProperties(schema types.Type, used []types.Path) []string {
	schemaProps := schema.PropertiesNames()
	notUsed := make(map[string]struct{}, len(schemaProps))
	for _, p := range schemaProps {
		notUsed[p] = struct{}{}
	}
	for _, path := range used {
		delete(notUsed, path[0])
	}
	if len(notUsed) == 0 {
		return nil
	}
	props := maps.Keys(notUsed)
	slices.Sort(props)
	return props
}
