//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package apis

// This file contains the function "validateActionToSet", as well as any support
// type, function and/or methods used exclusively by it.

import (
	"fmt"
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

// validationState is a state for the validation of an action.
type validationState struct {

	// connection is the action's connection.
	connection struct {
		role      state.Role
		connector struct {
			typ state.ConnectorType
		}
	}

	// connector represents the action's connector.
	//
	// If the actions specifies a connector name, then this must be populated
	// according to that connector, if exists, otherwise must be the empty
	// struct.
	connector struct {
		typ       state.ConnectorType
		hasUI     bool
		hasSheets bool
	}

	// provider is the transformers.Provider instantiated on the APIs.
	provider transformers.Provider
}

// validateAction validates action and target, in the context of the given
// validation state.
//
// It returns an errors.UnprocessableError error with code:
//
//   - LanguageNotSupported, if the transformation language is not supported.
//   - ConnectorNotExist, if the action is on file and the specified file
//     connector does not exist.
func validateAction(action ActionToSet, target state.Target, v validationState) error {

	inSchema := action.InSchema
	outSchema := action.OutSchema

	// Check if the target is allowed.
	var targetIsAllowed bool
	switch v.connection.role {
	case state.Source:
		switch v.connection.connector.typ {
		case state.AppType, state.DatabaseType, state.FileStorageType:
			targetIsAllowed = target == state.Users || target == state.Groups
		case state.MobileType, state.ServerType, state.WebsiteType:
			targetIsAllowed = true
		}
	case state.Destination:
		switch v.connection.connector.typ {
		case state.AppType:
			targetIsAllowed = true
		case state.DatabaseType, state.FileStorageType:
			targetIsAllowed = target == state.Users || target == state.Groups
		}
	}
	if !targetIsAllowed {
		role := strings.ToLower(v.connection.role.String())
		typ := v.connection.connector.typ.String()
		return errors.BadRequest("action with target '%s' not allowed for %s %s connections", target, role, typ)
	}

	dispatchEventsToApps := dispatchesEventsToApps(v.connection.connector.typ, v.connection.role, target)
	importUserIdentitiesFromEvents := importsUserIdentitiesFromEvents(v.connection.connector.typ, v.connection.role, target)

	// When dispatching events to apps or when importing user identities from
	// events, the input schema must be the invalid schema.
	if dispatchEventsToApps || importUserIdentitiesFromEvents {
		if inSchema.Valid() {
			if importUserIdentitiesFromEvents {
				return errors.BadRequest("input schema must be invalid for actions that import user identities from events")
			}
			return errors.BadRequest("input schema must be invalid for actions that dispatch events to apps")
		}
		// The input schema is the events schema without the GID, because both
		// the actions that import user identities from events and the actions
		// that dispatch events to apps have in input an event without a GID, as
		// the GID is added to the event when it is already in the warehouse.
		inSchema = events.Schema
	}

	// Validate the action's connector.
	actionOnFile := v.connection.connector.typ == state.FileStorageType
	if actionOnFile && action.Connector == "" {
		return errors.BadRequest("actions on file storage connections must have a connector")
	}
	if !actionOnFile && action.Connector != "" {
		return errors.BadRequest("actions on %v connections cannot have a connector", v.connection.connector.typ)
	}
	if action.Connector != "" {
		if v.connector.typ == 0 {
			return errors.Unprocessable(ConnectorNotExist, "connector %q does not exist", action.Connector)
		}
		if v.connector.typ != state.FileType {
			return errors.BadRequest("type of the action's connector must be File, got %v", v.connector.typ)
		}
	}

	// First, do formal validations.

	// Validate the name.
	if action.Name == "" {
		return errors.BadRequest("name is empty")
	}
	if !utf8.ValidString(action.Name) {
		return errors.BadRequest("name is not UTF-8 encoded")
	}
	if containsNUL(action.Name) {
		return errors.BadRequest("name contains NUL rune")
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
	var usedInPaths []string
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
	var usedOutPaths []string
	var mappingInProperties int
	if mapping := action.Transformation.Mapping; mapping != nil {
		if len(mapping) == 0 {
			return errors.BadRequest("transformation mapping must have mapped properties")
		}
		if !inSchema.Valid() && !dispatchEventsToApps {
			return errors.BadRequest("input schema is required by the mapping")
		}
		if !outSchema.Valid() {
			return errors.BadRequest("output schema is required by the mapping")
		}
		transformer, err := mappings.New(mapping, inSchema, outSchema, nil)
		if err != nil {
			return errors.BadRequest("invalid mapping: %s", err)
		}
		// Input property paths.
		inProps := transformer.InProperties()
		mappingInProperties = len(inProps)
		usedInPaths = append(usedInPaths, inProps...)
		// Output property paths.
		usedOutPaths = maps.Keys(mapping)
	}
	// Validate the transformation.
	if function := action.Transformation.Function; function != nil {
		if !inSchema.Valid() && !dispatchEventsToApps {
			return errors.BadRequest("input schema is required by the transformation")
		}
		if !outSchema.Valid() {
			return errors.BadRequest("output schema is required by the transformation")
		}
		if function.Source == "" {
			return errors.BadRequest("function transformation source is empty")
		}
		if containsNUL(function.Source) {
			return errors.BadRequest("function transformation source contains NUL rune")
		}
		switch function.Language {
		case "JavaScript":
			if v.provider == nil || !v.provider.SupportLanguage(state.JavaScript) {
				return errors.Unprocessable(LanguageNotSupported, "JavaScript transformation language is not supported")
			}
		case "Python":
			if v.provider == nil || !v.provider.SupportLanguage(state.Python) {
				return errors.Unprocessable(LanguageNotSupported, "Python transformation language is not supported")
			}
		case "":
			return errors.BadRequest("transformation language is empty")
		default:
			return errors.BadRequest("transformation language %q is not valid", action.Transformation.Function.Language)
		}
		err := validateTransformationFunctionProperties("input", inSchema, function.InProperties, dispatchEventsToApps)
		if err != nil {
			return errors.BadRequest("%s", err.Error())
		}
		err = validateTransformationFunctionProperties("output", outSchema, function.OutProperties, dispatchEventsToApps)
		if err != nil {
			return errors.BadRequest("%s", err.Error())
		}
		usedInPaths = append(usedInPaths, function.InProperties...)
		usedOutPaths = function.OutProperties
	}
	// Validate the path.
	if action.Path != "" {
		if !utf8.ValidString(action.Path) {
			return errors.BadRequest("path is not UTF-8 encoded")
		}
		if containsNUL(action.Path) {
			return errors.BadRequest("path contains NUL rune")
		}
		if n := utf8.RuneCountInString(action.Path); n > 1024 {
			return errors.BadRequest("path is longer than 1024 runes")
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
		if !utf8.ValidString(action.TableName) {
			return errors.BadRequest("table name is not UTF-8 encoded")
		}
		if containsNUL(action.TableName) {
			return errors.BadRequest("table name contains NUL rune")
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
		usedInPaths = append(usedInPaths, props.Internal)
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
			return errors.BadRequest(err.Error())
		}
	}
	// Validate the file ordering property path.
	if action.FileOrderingPropertyPath != "" {
		if !types.IsValidPropertyPath(action.FileOrderingPropertyPath) {
			return errors.BadRequest("the specified file ordering is a not valid property path")
		}
		if utf8.RuneCountInString(action.FileOrderingPropertyPath) > 1024 {
			return errors.BadRequest("file ordering property path is longer than 1024 runes")
		}
	}

	// Second, do validations based on the workspace and the connection.

	eventBasedConn := v.connection.connector.typ == state.MobileType ||
		v.connection.connector.typ == state.ServerType ||
		v.connection.connector.typ == state.WebsiteType

	// Check that schemas that refer to users cannot contain "nullable" or
	// "required" properties.
	if target == state.Users {
		if v.connection.role == state.Source && outSchema.Valid() {
			for path, p := range types.Walk(outSchema) {
				if p.Nullable {
					return errors.BadRequest("property %q in output schema cannot be nullable", path)
				}
				if p.Required {
					return errors.BadRequest("property %q in output schema cannot be required", path)
				}
			}
		} else if v.connection.role == state.Destination && inSchema.Valid() {
			for path, p := range types.Walk(inSchema) {
				if p.Nullable {
					return errors.BadRequest("property %q in input schema cannot be nullable", path)
				}
				if p.Required {
					return errors.BadRequest("property %q in input schema cannot be required", path)
				}
			}
		}
	}

	// In case of a source connection, since its actions write on the data
	// warehouse, the output schema cannot contain meta properties because such
	// properties are not writable by user transformations.
	if v.connection.role == state.Source && outSchema.Valid() {
		for _, p := range outSchema.Properties() {
			if isMetaProperty(p.Name) {
				return errors.BadRequest("output schema cannot contain meta properties")
			}
		}
	}

	// Check if the UI values are allowed and are a JSON Object.
	if v.connection.connector.typ == state.FileStorageType {
		if action.UIValues == nil {
			if v.connector.hasUI {
				return errors.BadRequest("UI values must be provided because connector %s has a UI", action.Connector)
			}
		} else {
			if !v.connector.hasUI {
				return errors.BadRequest("UI values cannot be provided because connector %s has no UI", action.Connector)
			}
			if !isJSONObject(action.UIValues) {
				return errors.BadRequest("UI values are not a valid JSON Object")
			}
		}
	} else if action.UIValues != nil {
		return errors.BadRequest("%s actions cannot have UI values", strings.ToLower(v.connection.connector.typ.String()))
	}

	// Check if the compression is allowed.
	if action.Compression != NoCompression && v.connection.connector.typ != state.FileStorageType {
		return errors.BadRequest("%s actions cannot have compression", strings.ToLower(v.connection.connector.typ.String()))
	}

	// Check if the query is allowed.
	if needsQuery := v.connection.connector.typ == state.DatabaseType && v.connection.role == state.Source; needsQuery {
		if action.Query == "" {
			return errors.BadRequest("query cannot be empty for database actions")
		}
		if containsNUL(action.Query) {
			return errors.BadRequest("query contains NUL rune")
		}
	} else {
		if action.Query != "" {
			return errors.BadRequest("%s actions cannot have a query", v.connection.connector.typ)
		}
	}

	// Check if the filters are allowed.
	targetUsersOrGroups := target == state.Users || target == state.Groups
	var filtersAllowed bool
	switch v.connection.connector.typ {
	case state.AppType:
		filtersAllowed = v.connection.role == state.Destination
	case state.DatabaseType:
		filtersAllowed = v.connection.role == state.Destination
	case state.FileStorageType:
		filtersAllowed = targetUsersOrGroups && v.connection.role == state.Destination
	case state.MobileType, state.ServerType, state.WebsiteType:
		filtersAllowed = targetUsersOrGroups && v.connection.role == state.Source
	}
	if action.Filter != nil && !filtersAllowed {
		return errors.BadRequest("filters are not allowed")
	}

	// Check if the path and the sheet are allowed.
	if v.connection.connector.typ == state.FileStorageType {
		if action.Path == "" {
			return errors.BadRequest("path cannot be empty for actions on storage connections")
		}
		if v.connector.hasSheets && action.Sheet == "" {
			return errors.BadRequest("sheet cannot be empty because connector %s has sheets", action.Connector)
		}
		if !v.connector.hasSheets && action.Sheet != "" {
			return errors.BadRequest("connector %s does not have sheets", action.Connector)
		}
	} else {
		if action.Path != "" {
			return errors.BadRequest("%s actions cannot have a path", v.connection.connector.typ)
		}
		if action.Sheet != "" {
			return errors.BadRequest("%s actions cannot have a sheet", v.connection.connector.typ)
		}
	}

	// Check the column for the identity property and for the timestamp.
	importFromColumns := v.connection.role == state.Source &&
		(v.connection.connector.typ == state.DatabaseType || v.connection.connector.typ == state.FileStorageType)
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
		usedInPaths = append(usedInPaths, action.IdentityProperty)
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
			if v.connection.connector.typ == state.DatabaseType && action.LastChangeTimeFormat == "Excel" {
				return errors.BadRequest("last change time format cannot be Excel for database actions")
			}
		}
	} else {
		if action.IdentityProperty != "" {
			return errors.BadRequest("action cannot specify an identity property")
		}
		if action.LastChangeTimeProperty != "" {
			return errors.BadRequest("action cannot specify a last change time property")
		}
		if action.LastChangeTimeFormat != "" {
			return errors.BadRequest("action cannot specify a last change time format")
		}
	}

	// Do some checks related to exporting users to files.
	exportUsersToFile := v.connection.connector.typ == state.FileStorageType && v.connection.role == state.Destination && target == state.Users
	if exportUsersToFile {
		// When exporting users to file, ensure that the output schema is valid,
		// as it contains the properties that will be exported to the file.
		if !outSchema.Valid() {
			return errors.BadRequest("output schema cannot be empty when exporting users to file")
		}
		// Check that FileOrderingPropertyPath is defined and exists in the out
		// schema.
		if action.FileOrderingPropertyPath == "" {
			return errors.BadRequest("file ordering property path cannot be empty when exporting users to file")
		}
		p, err := outSchema.PropertyByPath(action.FileOrderingPropertyPath)
		if err != nil {
			return errors.BadRequest("file ordering property path cannot be found in action's output schema: %s", err)
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
				return errors.BadRequest("the Decimal type of the file ordering property cannot have a precision greater than 0")
			}
		default:
			return errors.BadRequest("file ordering property cannot have kind %s", p.Type.Kind())
		}
	} else {
		if action.FileOrderingPropertyPath != "" {
			return errors.BadRequest("actions that do not export users to files cannot specify a file ordering property path")
		}
	}

	// Check if the table name is allowed.
	needsTableName := v.connection.connector.typ == state.DatabaseType && v.connection.role == state.Destination
	if needsTableName && action.TableName == "" {
		return errors.BadRequest("table name cannot be empty for destination database actions")
	} else if !needsTableName && action.TableName != "" {
		return errors.BadRequest("table name is not allowed")
	}

	// Check if the export options are needed.
	needsExportOptions := v.connection.connector.typ == state.AppType &&
		v.connection.role == state.Destination && target == state.Users
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
	transformationProhibited := (v.connection.role == state.Source && eventBasedConn && target == state.Events) ||
		(v.connection.role == state.Destination && v.connection.connector.typ == state.FileStorageType && targetUsersOrGroups)
	haveTransformation := action.Transformation.Mapping != nil || action.Transformation.Function != nil
	if transformationProhibited && haveTransformation {
		return errors.BadRequest("action cannot have a transformation")
	}

	// Check if the transformation is mandatory, with at least one input
	// property.
	transformationMandatory := targetUsersOrGroups &&
		(v.connection.connector.typ == state.AppType || v.connection.connector.typ == state.DatabaseType ||
			(v.connection.role == state.Source && v.connection.connector.typ == state.FileStorageType))
	if transformationMandatory && !haveTransformation {
		return errors.BadRequest("action must have a transformation")
	}

	// Transformations must have at least one property in the input schema,
	// except when importing identities from events and when dispatching events
	// to apps, where "constant" transformation functions are supported.
	if !importUserIdentitiesFromEvents && !dispatchEventsToApps {
		if action.Transformation.Mapping != nil && mappingInProperties == 0 {
			return errors.BadRequest("transformation must map at least one property")
		}
		if action.Transformation.Function != nil && len(action.Transformation.Function.InProperties) == 0 {
			return errors.BadRequest("transformation function must have at least one input property")
		}
	}

	// Ensure that every property in the input and output schemas have been used
	// (by the mappings, by the filters, etc...).
	if action.Transformation.Function != nil {
		// The action has a transformation function, so we do not know which
		// properties are used; consequently, this check would always pass
		// because we would consider every property of the schema as used.
	} else if importUserIdentitiesFromEvents || dispatchEventsToApps {
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
// 'apis/connectors.parseLastChangeTimePropertyWithFormat'.
func validateLastChangeTimeFormat(format string) error {
	switch format {
	case
		"ISO8601",
		"Excel":
		return nil
	}
	if format == "" {
		return errors.New("last change time format is empty")
	}
	if utf8.RuneCountInString(format) > 64 {
		return errors.New("last change time format is longer than 64 runes")
	}
	if !utf8.ValidString(format) {
		return errors.New("last change time format contains invalid UTF-8 characters")
	}
	if !strings.Contains(format, "%") {
		return fmt.Errorf("last change time format %q is not valid", format)
	}
	if containsNUL(format) {
		return errors.New("last change time format contains the NUL rune")
	}
	return nil
}

// validateTransformationFunctionProperties validates the transformation
// function properties of an action.
//
// io specifies whether the validation relates to "input" or "output", schema is
// the schema of the input or output action, properties are the function
// properties for input or output, and dispatchEventsToApps indicates if the
// action dispatches events to apps.
//
// It panics if the schema is valid and is not an Object
func validateTransformationFunctionProperties(io string, schema types.Type, properties []string, dispatchEventsToApps bool) error {
	if len(properties) == 0 {
		if properties == nil {
			return fmt.Errorf("function transformation %s properties cannot be null", io)
		}
		if dispatchEventsToApps && io == "input" {
			return nil
		}
		return fmt.Errorf("there are no function transformation %s properties", io)
	}
	has := make(map[string]struct{}, len(properties))
	for _, name := range properties {
		if _, ok := has[name]; ok {
			return fmt.Errorf("function transformation %s property %q is repeated", io, name)
		}
		has[name] = struct{}{}
	}
	if schema.Valid() {
		for _, p := range schema.Properties() {
			delete(has, p.Name)
		}
	}
	if len(has) > 0 {
		for _, name := range properties {
			if _, ok := has[name]; ok {
				if !types.IsValidPropertyName(name) {
					return fmt.Errorf("function transformation %s property name %q is not valid", io, name)
				}
				return fmt.Errorf("function transformation %s property %q does not exist in schema", io, name)
			}
		}
	}
	return nil
}
