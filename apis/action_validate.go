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
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/events"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers"
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
func validateActionToSet(action ActionToSet, target state.Target, c *state.Connection, fileConnector *state.Connector, provider transformers.Provider) error {

	inSchema := action.InSchema
	outSchema := action.OutSchema

	importUserIdentitiesFromEvents := importsUserIdentitiesFromEvents(c.Connector().Type, c.Role, target)
	if importUserIdentitiesFromEvents {
		if inSchema.Valid() {
			return errors.BadRequest("input schema must be invalid for actions that import user identities from events")
		}
		// The input schema is the events schema without GID because this
		// actions imports user identities from incoming events, which, clearly,
		// still do not have any user associated.
		inSchema = events.Schema
	}

	dispatchEventsToApps := c.Role == state.Destination && target == state.Events && c.Connector().Type == state.AppType

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
		transformer := transformers.New(inSchema, outSchema, state.Transformation{Mapping: mapping}, 0, nil, nil)
		// Input property paths.
		inProps := transformer.Properties()
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
			if provider == nil || !provider.SupportLanguage(state.JavaScript) {
				return errors.Unprocessable(LanguageNotSupported, "JavaScript transformation language is not supported")
			}
		case "Python":
			if provider == nil || !provider.SupportLanguage(state.Python) {
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
		if containsNUL(action.Query) {
			return errors.BadRequest("query contains NUL rune")
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
	case state.MobileType, state.ServerType, state.WebsiteType:
		filtersAllowed = targetUsersOrGroups && c.Role == state.Source
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
			return errors.BadRequest("sheet cannot be empty because connector %s has sheets", fileConnector.Name)
		}
		if !fileConnector.HasSheets && action.Sheet != "" {
			return errors.BadRequest("connector %s does not have sheets", fileConnector.Name)
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
			if connector.Type == state.DatabaseType && action.LastChangeTimeFormat == "Excel" {
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
	exportUsersToFile := connector.Type == state.FileStorageType && c.Role == state.Destination && target == state.Users
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

	// Check if the transformation is mandatory, with at least one input
	// property.
	transformationMandatory := targetUsersOrGroups &&
		(connector.Type == state.AppType || connector.Type == state.DatabaseType ||
			(c.Role == state.Source && connector.Type == state.FileStorageType))
	if transformationMandatory && !haveTransformation {
		return errors.BadRequest("action must have a transformation")
	}

	// Transformations must have at least one property in the input schema,
	// except when importing identities from events and when dispatching events
	// to apps, where "constant" transformation functions are supported.
	//
	// TODO(Gianluca): there may be some inconsistencies in this part, regarding
	// UI, documentation and APIs. This still needs to be made consistent.
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
	} else if importUserIdentitiesFromEvents {
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
