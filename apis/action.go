//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/datastore"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/events"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers"
	"github.com/open2b/chichi/telemetry"
	"github.com/open2b/chichi/types"
)

// Action represents an action of a connection.
type Action struct {
	apis                    *APIs
	action                  *state.Action
	connection              *Connection
	ID                      int
	Connection              int
	Target                  Target
	Name                    string
	Enabled                 bool
	EventType               *string
	Running                 bool
	ScheduleStart           *int
	SchedulePeriod          *SchedulePeriod
	InSchema                types.Type
	OutSchema               types.Type
	Filter                  *Filter
	Transformation          Transformation
	Query                   *string
	Connector               string
	Path                    *string
	Sheet                   *string
	Compression             Compression
	Table                   *string
	IdentityProperty        *string
	DisplayedProperty       string
	LastChangeTimeProperty  *string
	LastChangeTimeFormat    *string
	ExportMode              *ExportMode
	MatchingProperties      *MatchingProperties
	ExportOnDuplicatedUsers *bool
}

// Language represents a transformation language. Valid values are "JavaScript"
// and "Python".
type Language string

// TransformationFunction represents a transformation function.
type TransformationFunction struct {
	Source   string
	Language Language
}

// Transformation represents a transformation.
type Transformation struct {
	Mapping  map[string]string
	Function *TransformationFunction
}

// ExportMode represents one of the three export modes.
type ExportMode string

const (
	CreateOnly     ExportMode = "CreateOnly"
	UpdateOnly     ExportMode = "UpdateOnly"
	CreateOrUpdate ExportMode = "CreateOrUpdate"
)

// fromState serializes action into this.
func (this *Action) fromState(apis *APIs, store *datastore.Store, action *state.Action) {
	c := action.Connection()
	this.apis = apis
	this.action = action
	this.connection = &Connection{apis: apis, store: store, connection: c}
	this.ID = action.ID
	this.Connection = c.ID
	this.Target = Target(action.Target)
	this.Name = action.Name
	this.Enabled = action.Enabled
	if action.EventType != "" {
		et := action.EventType
		this.EventType = &et
	}
	_, this.Running = this.action.Execution()
	if action.Target == state.Users || action.Target == state.Groups {
		start := int(action.ScheduleStart)
		period := SchedulePeriod(action.SchedulePeriod)
		this.ScheduleStart = &start
		this.SchedulePeriod = &period
	}
	this.InSchema = action.InSchema
	this.OutSchema = action.OutSchema
	if action.Filter != nil {
		this.Filter = &Filter{
			Logical:    FilterLogical(action.Filter.Logical),
			Conditions: make([]FilterCondition, len(action.Filter.Conditions)),
		}
		for i, condition := range action.Filter.Conditions {
			this.Filter.Conditions[i] = FilterCondition(condition)
		}
	}
	if mapping := action.Transformation.Mapping; mapping != nil {
		this.Transformation.Mapping = make(map[string]string, len(mapping))
		for out, in := range mapping {
			this.Transformation.Mapping[out] = in
		}
	}
	if function := action.Transformation.Function; function != nil {
		this.Transformation.Function = &TransformationFunction{
			Source:   function.Source,
			Language: Language(function.Language.String()),
		}
	}
	if action.Query != "" {
		query := action.Query
		this.Query = &query
	}
	if c := action.Connector(); c != nil {
		this.Connector = c.Name
	}
	if action.Path != "" {
		path := action.Path
		this.Path = &path
	}
	if action.Sheet != "" {
		sheet := action.Sheet
		this.Sheet = &sheet
	}
	this.Compression = Compression(action.Compression)
	if action.TableName != "" {
		table := action.TableName
		this.Table = &table
	}
	if action.IdentityProperty != "" {
		p := action.IdentityProperty
		this.IdentityProperty = &p
	}
	this.DisplayedProperty = action.DisplayedProperty
	if action.LastChangeTimeProperty != "" {
		column := action.LastChangeTimeProperty
		this.LastChangeTimeProperty = &column
	}
	if action.LastChangeTimeFormat != "" {
		format := action.LastChangeTimeFormat
		this.LastChangeTimeFormat = &format
	}
	this.ExportMode = (*ExportMode)(action.ExportMode)
	if props := action.MatchingProperties; props != nil {
		this.MatchingProperties = &MatchingProperties{
			Internal: props.Internal,
			External: props.External,
		}
	}
	this.ExportOnDuplicatedUsers = action.ExportOnDuplicatedUsers
}

// Target represents a target.
type Target int

const (
	Events Target = iota + 1
	Users
	Groups
)

// MarshalJSON implements the json.Marshaler interface.
func (at Target) MarshalJSON() ([]byte, error) {
	return []byte(`"` + at.String() + `"`), nil
}

func (at Target) String() string {
	switch at {
	case Events:
		return "Events"
	case Users:
		return "Users"
	case Groups:
		return "Groups"
	default:
		panic("invalid target")
	}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (at *Target) UnmarshalJSON(data []byte) error {
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("json: cannot scan a %T value into an api.Target value", v)
	}
	switch s {
	case "Events":
		*at = Events
	case "Users":
		*at = Users
	case "Groups":
		*at = Groups
	default:
		return fmt.Errorf("json: invalid apis.Target: %s", s)
	}
	return nil
}

// Delete deletes the action.
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
func (this *Action) Delete(ctx context.Context) error {
	this.apis.mustBeOpen()
	n := state.DeleteAction{
		ID: this.action.ID,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "DELETE FROM actions WHERE id = $1", n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("action %d does not exist", n.ID)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// ServeUI serves the user interface for the file action (on a file storage
// connection). event is the event to be served and values are the user-entered
// values in JSON format.
//
// It returns an errors.UnprocessableError error with code:
//
//   - EventNotExist, if the event does not exist.
//   - InvalidUIValues, if the user-entered values are not valid.
func (this *Action) ServeUI(ctx context.Context, event string, values []byte) ([]byte, error) {
	this.apis.mustBeOpen()
	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	connector := this.action.Connection().Connector()
	if connector.Type != state.FileStorageType {
		return nil, errors.BadRequest("cannot serve the UI of an action on a %s connection", connector.Type)
	}
	if !connector.HasUI {
		return nil, errors.BadRequest("connector %s does not have a UI", connector.Name)
	}
	ui, err := this.apis.connectors.ServeActionUI(ctx, this.action, event, values)
	if err != nil {
		if err == connectors.ErrUIEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s connector", event, connector.Name)
		} else if err2, ok := err.(connectors.InvalidUIValuesError); ok {
			err = errors.Unprocessable(InvalidUIValues, "%w", err2)
		}
		return nil, err
	}
	return ui, nil
}

// Execute executes the action, which must be an app, database, or file storage
// action with a target of Users or Groups.
//
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
// It returns an errors.UnprocessableError error with code
//
//   - ConnectionDisabled, if the connection is disabled.
//   - ExecutionInProgress, if the action is already in progress.
//   - NoWarehouse, if the workspace does not have a data warehouse.
func (this *Action) Execute(ctx context.Context, reimport bool) error {
	this.apis.mustBeOpen()
	ctx, span := telemetry.TraceSpan(ctx, "Action.Execute", "id", this.action.ID, "reimport", reimport)
	defer span.End()
	c := this.action.Connection()
	if !c.Enabled {
		return errors.Unprocessable(ConnectionDisabled, "connection %d is disabled", c.ID)
	}
	if _, ok := this.action.Execution(); ok {
		return errors.Unprocessable(ExecutionInProgress, "action %d is already in progress", this.action.ID)
	}
	if t := this.action.Target; t != state.Users && t != state.Groups {
		return errors.BadRequest("action %d with target %s cannot be executed", this.action.ID, t)
	}
	switch typ := c.Connector().Type; typ {
	case state.AppType, state.DatabaseType, state.FileStorageType:
	default:
		return errors.BadRequest("%s actions cannot be executed", strings.ToLower(typ.String()))
	}
	if this.connection.store == nil {
		ws := c.Workspace()
		return errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}
	return this.addExecution(ctx, reimport)
}

// Set sets the action.
//
// Refer to the specifications in the file "apis/Actions.md" for more details.
//
// It returns an errors.UnprocessableError error with code:
//
//   - ConnectorNotExist, if the connector does not exist.
//   - InvalidUIValues, if the user-entered values are not valid.
//   - LanguageNotSupported, if the transformation language is not supported.
func (this *Action) Set(ctx context.Context, action ActionToSet) error {

	this.apis.mustBeOpen()

	ctx, span := telemetry.TraceSpan(ctx, "Action.Set", "action", this.action.ID)
	defer span.End()

	// Validate the connector.
	actionOnFile := this.action.Connection().Connector().Type == state.FileStorageType
	if actionOnFile && action.Connector == "" {
		return errors.BadRequest("actions on file storage connections must have a connector")
	}
	if !actionOnFile && action.Connector != "" {
		return errors.BadRequest("actions on %v connections cannot have a connector", this.action.Connection().Connector().Type)
	}
	var fileConnector *state.Connector
	if action.Connector != "" {
		var ok bool
		fileConnector, ok = this.apis.state.Connector(action.Connector)
		if !ok {
			return errors.Unprocessable(ConnectorNotExist, "connector %q does not exist", action.Connector)
		}
		if fileConnector.Type != state.FileType {
			return errors.BadRequest("type of the action's connector must be File, got %v", fileConnector.Type)
		}
	}

	c := this.action.Connection()

	// Validate the action.
	err := validateActionToSet(action, this.action.Target, c, fileConnector, this.apis.functionTransformer)
	if err != nil {
		return err
	}

	inSchema := action.InSchema
	if importsUsersIdentitiesFromEvents(c.Connector().Type, c.Role, this.action.Target) {
		// Use the schema without GID because incoming events do not have a GID.
		inSchema = events.Schema
	}

	span.Log("action validated successfully")

	n := state.SetAction{
		ID:        this.action.ID,
		Name:      action.Name,
		Enabled:   action.Enabled,
		InSchema:  inSchema,
		OutSchema: action.OutSchema,
		Transformation: state.Transformation{
			Mapping: action.Transformation.Mapping,
		},
		Query:                   action.Query,
		Connector:               action.Connector,
		Path:                    action.Path,
		Sheet:                   action.Sheet,
		Compression:             state.Compression(action.Compression),
		TableName:               action.TableName,
		IdentityProperty:        action.IdentityProperty,
		DisplayedProperty:       action.DisplayedProperty,
		LastChangeTimeProperty:  action.LastChangeTimeProperty,
		LastChangeTimeFormat:    action.LastChangeTimeFormat,
		ExportMode:              (*state.ExportMode)(action.ExportMode),
		ExportOnDuplicatedUsers: action.ExportOnDuplicatedUsers,
	}
	if function := action.Transformation.Function; function != nil {
		n.Transformation.Function = &state.TransformationFunction{Source: function.Source}
		switch function.Language {
		case "JavaScript":
			n.Transformation.Function.Language = state.JavaScript
		case "Python":
			n.Transformation.Function.Language = state.Python
		}
	}

	// Add the filter to the notification and marshal it.
	var filter []byte
	if action.Filter != nil {
		n.Filter = &state.Filter{
			Logical:    state.FilterLogical(action.Filter.Logical),
			Conditions: make([]state.FilterCondition, len(action.Filter.Conditions)),
		}
		for i, condition := range action.Filter.Conditions {
			n.Filter.Conditions[i] = (state.FilterCondition)(condition)
		}
		filter, err = json.Marshal(action.Filter)
		if err != nil {
			return err
		}
	}

	// Determine the connector name, for file actions.
	var connectorName *string
	if fileConnector != nil {
		name := fileConnector.Name
		connectorName = &name
	}

	if props := action.MatchingProperties; props != nil {
		n.MatchingProperties = &state.MatchingProperties{
			Internal: props.Internal,
			External: props.External,
		}
	}

	// Marshal the input and the output schemas.
	rawInSchema, err := marshalSchema(inSchema)
	if err != nil {
		return err
	}
	rawOutSchema, err := marshalSchema(action.OutSchema)
	if err != nil {
		return err
	}

	// Marshal the mapping.
	var mapping []byte
	if action.Transformation.Mapping != nil {
		mapping, err = json.Marshal(action.Transformation.Mapping)
		if err != nil {
			return err
		}
	}

	// Matching properties.
	var matchPropInternal, matchPropExternal []byte
	if n.MatchingProperties != nil {
		var err error
		matchPropInternal, err = json.Marshal(n.MatchingProperties.Internal)
		if err != nil {
			return err
		}
		matchPropExternal, err = json.Marshal(n.MatchingProperties.External)
		if err != nil {
			return err
		}
	}

	// Settings.
	if fileConnector != nil && fileConnector.HasUI {
		conf := &connectors.ConnectorConfig{
			Role:   this.action.Connection().Role,
			Region: this.action.Connection().Workspace().PrivacyRegion,
		}
		n.Settings, err = this.apis.connectors.UpdatedSettings(ctx, fileConnector, conf, action.UIValues)
		if err != nil {
			if err2, ok := err.(connectors.InvalidUIValuesError); ok {
				err = errors.Unprocessable(InvalidUIValues, "%w", err2)
			}
			return err
		}
	}

	// Transformation.
	if fn := n.Transformation.Function; fn != nil {
		if this.action.Transformation.Function == nil {
			name := transformationFunctionName(n.ID, fn.Language)
			version, err := this.apis.functionTransformer.Create(ctx, name, fn.Source)
			if err == transformers.ErrFunctionExist {
				version, err = this.apis.functionTransformer.Update(ctx, name, fn.Source)
			}
			if err != nil {
				return err
			}
			n.Transformation.Function.Version = version
		} else if this.action.Transformation.Function.Source != fn.Source || this.action.Transformation.Function.Language != fn.Language {
			name := transformationFunctionName(n.ID, fn.Language)
			version, err := this.apis.functionTransformer.Update(ctx, name, fn.Source)
			if err == transformers.ErrFunctionNotExist {
				version, err = this.apis.functionTransformer.Create(ctx, name, fn.Source)
			}
			if err != nil {
				return err
			}
			n.Transformation.Function.Version = version
		} else {
			// The function's source code and language should not be changed.
			// It will be verified during the transaction and assigned the current version.
		}
	}

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		var function state.TransformationFunction
		if n.Transformation.Function != nil {
			var current state.TransformationFunction
			if n.Transformation.Function.Version == "" {
				err := tx.QueryRow(ctx, "SELECT transformation_source, transformation_language, transformation_version "+
					"FROM actions WHERE id = $1", n.ID).Scan(&current.Source, &current.Language, &current.Version)
				if err != nil {
					return err
				}
				if current.Source != n.Transformation.Function.Source || current.Language != n.Transformation.Function.Language {
					return fmt.Errorf("abort update action %d: it was optimistically assumed that the transformation"+
						" had not changed, but it has indeed changed", n.ID)
				}
				n.Transformation.Function.Version = current.Version
			}
			function = *n.Transformation.Function
		}
		result, err := tx.Exec(ctx, "UPDATE actions SET\n"+
			"name = $1, enabled = $2, in_schema = $3, out_schema = $4, filter = $5, "+
			"transformation_mapping = $6, transformation_source = $7, transformation_language = $8, "+
			"transformation_version = $9, query = $10, connector = $11, path = $12, "+
			"sheet = $13, compression = $14, settings = $15, table_name = $16,  identity_property = $17, "+
			"displayed_property = $18, last_change_time_property = $19, last_change_time_format = $20, "+
			"export_mode = $21, matching_properties_internal = $22, matching_properties_external = $23, "+
			"export_on_duplicated_users = $24\nWHERE id = $25",
			n.Name, n.Enabled, rawInSchema, rawOutSchema, string(filter), mapping,
			function.Source, function.Language, function.Version, n.Query, connectorName,
			n.Path, n.Sheet, n.Compression, string(n.Settings), n.TableName,
			n.IdentityProperty, n.DisplayedProperty, n.LastChangeTimeProperty, n.LastChangeTimeFormat,
			n.ExportMode, string(matchPropInternal),
			string(matchPropExternal), n.ExportOnDuplicatedUsers, n.ID,
		)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		return tx.Notify(ctx, n)
	})
	span.Log("action set successfully", "id", this.action.ID)

	return err
}

// setUserCursor sets the user cursor of the action.
func (this *Action) setUserCursor(ctx context.Context, cursor state.Cursor) error {
	n := state.SetActionUserCursor{
		ID:         this.action.ID,
		UserCursor: cursor,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions\n"+
			"SET user_cursor.id = $1, user_cursor.last_change_time = $2 WHERE id = $3",
			n.UserCursor.ID, n.UserCursor.LastChangeTime, n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// SetSchedulePeriod sets the schedule period, in minutes, of the action. The
// action must be a Users or Groups action and period can be 5, 15, 30, 60, 120,
// 180, 360, 480, 720, or 1440.
func (this *Action) SetSchedulePeriod(ctx context.Context, period SchedulePeriod) error {
	this.apis.mustBeOpen()
	switch this.action.Target {
	case state.Users, state.Groups:
	default:
		return errors.BadRequest("cannot set schedule period of a %s action", this.action.Target)
	}
	switch period {
	case 5, 15, 30, 60, 120, 180, 360, 480, 720, 1440:
	default:
		return errors.BadRequest("schedule period %d is not valid", period)
	}
	n := state.SetActionSchedulePeriod{
		ID:             this.action.ID,
		SchedulePeriod: int16(period),
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions SET schedule_period = $1 WHERE id = $2 AND schedule_period <> $1", n.SchedulePeriod, n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// SetStatus sets the status of the action.
func (this *Action) SetStatus(ctx context.Context, enabled bool) error {
	this.apis.mustBeOpen()
	if enabled == this.action.Enabled {
		return nil
	}
	n := state.SetActionStatus{
		ID:      this.action.ID,
		Enabled: enabled,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE actions SET enabled = $1 WHERE id = $2 AND enabled <> $1", n.Enabled, n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return nil
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// app returns the app of the action.
func (this *Action) app() *connectors.App {
	return this.apis.connectors.App(this.action.Connection())
}

// database returns the database of the action.
// The caller must call the database's Close method when the database is no
// longer needed.
func (this *Action) database() *connectors.Database {
	a := this.action
	return this.apis.connectors.Database(a.Connection())
}

// isLanguageSupported reports whether the transformation language of the action
// is supported. If the action does not have a transformation, it returns true.
func (this *Action) isLanguageSupported() bool {
	transformation := this.action.Transformation.Function
	if transformation == nil {
		return true
	}
	if this.apis.functionTransformer != nil && this.apis.functionTransformer.SupportLanguage(transformation.Language) {
		return true
	}
	return false
}

// file returns the file of the action.
func (this *Action) file() *connectors.File {
	return this.apis.connectors.File(this.action, this.connection.connection.Role)
}

// ActionToSet represents an action to set in a connection, by adding a new
// action (using the method Connection.AddAction) or updating an existing one
// (using the method Action.Set).
//
// Refer to the specifications in the file "apis/Actions.md" for more details.
type ActionToSet struct {

	// Name must be a non-empty valid UTF-8 encoded string and cannot be longer
	// than 60 runes.
	Name string

	// Enabled indicates whether the action is enabled or not.
	Enabled bool

	// Filter is the filter of the action, if it has one, otherwise is nil.
	Filter *Filter

	// InSchema is the input schema of the action.
	//
	// It must contain exclusively:
	//
	// - the input properties used in the transformation, if this action has a
	//   transformation.
	// - the internal matching property, if this action has matching
	//   properties.
	// - the properties referred in the filters, if this action has filters.
	// - the identity and the last change time properties, if this action
	//   specifies them.
	InSchema types.Type

	// OutSchema is the output schema of the action.
	//
	// It must contain exclusively:
	//
	// - the output properties used in the transformation, if this action has a
	//   transformation.
	// - the schema of the users that will be exported to a file, it this action
	//   exports users to a file.
	OutSchema types.Type

	// Transformation is the mapping or function transformation, if it has one.
	//
	// Every action that supports transformations may have an associated
	// mapping or function, which are mutually exclusive.
	//
	// If it has a mapping, the names of the properties in which the values are
	// mapped (the keys of the map) must be present in the output schema of the
	// action, while the values of the map must be valid mapping expressions.
	Transformation Transformation

	// Query is the query of the action, if it has one, otherwise it is the
	// empty string.
	Query string

	// Connector is the connector of the action on file storage connections.
	// In any other case, must be empty.
	Connector string

	// Path is the path of the file. It cannot be longer than 1024 runes,
	// and it is empty for non-file actions.
	Path string

	// Sheet is the sheet name for multiple sheets file actions. It must be UTF-8
	// encoded, have a length in the range [1, 31], should not start or end with
	// "'", and cannot contain any of "*", "/", ":", "?", "[", "\", and "]". It is
	// empty for non-file and non-multipart sheets actions. Sheet names are
	// case-insensitive.
	Sheet string

	// Compression is the compression of the action on file storage connections.
	// In any other case, must be 0.
	Compression Compression

	// UIValues represents the user-entered values of the connector user interface
	// in JSON format.
	// It must be nil if the connector does not have a user interface.
	UIValues json.RawMessage

	// TableName is the name of the table for the export and it is defined for
	// destination database-actions; in any other case, it is the empty string.
	// It cannot be longer than 1024 runes.
	TableName string

	// IdentityProperty is the property name used as identity when importing
	// from a file or from a database.
	// It cannot be longer than 1024 runes.
	IdentityProperty string

	// DisplayedProperty, if not empty, is the property that holds the
	// identifier displayed in the UI for the imported user or group.
	//
	// In particular, for apps actions it is an app property, for file and
	// database actions it is a column name, while for event-based actions it is
	// a "traits" property.
	DisplayedProperty string

	// LastChangeTimeProperty is the last change time property when importing
	// from a file or from a database. May be empty to indicate that no
	// properties should be used for reading the last change times. Also refer
	// to the documentation of LastChangeTimeFormat, which is strictly related
	// to this.
	// It cannot be longer than 1024 runes.
	LastChangeTimeProperty string

	// LastChangeTimeFormat indicates the last change time value format for
	// parsing the value read from the last change time property.
	//
	// Represents a format when a LastChangeTimeProperty is provided and its
	// corresponding property kind is JSON or Text, otherwise it is the empty
	// string.
	//
	// In case it is provided, accepted values are:
	//
	//   - "DateTime", to parse timestamps in the format "2006-01-02 15:04:05"
	//   - "DateOnly", to parse date-only timestamps in the format "2006-01-02"
	//   - "ISO8601", to parse timestamps as a ISO 8601 timestamps.
	//   - "Excel", to parse timestamps as strings representing a float value
	//     stored in a Excel cell representing a date / datetime.
	//   - a strptime format, enclosed by single quote characters, compatible
	//     with the standard C89 functions strptime/strftime.
	//
	// It cannot be longer than 64 runes.
	LastChangeTimeFormat string

	// ExportMode is the export mode, if it has one.
	ExportMode *ExportMode

	// MatchingProperties are the internal and external properties used for matching
	// users during export to apps.
	MatchingProperties *MatchingProperties

	// ExportOnDuplicatedUsers indicates if the export to app connections should
	// be executed even in the case of duplicated users on the app.
	ExportOnDuplicatedUsers *bool
}

// MatchingProperties contains an internal property (belonging to the Golden
// Record) and an external property (belonging to the app) which are used to
// match identities of users in the data warehouse with users on the external
// app, during export.
type MatchingProperties struct {
	Internal string // the corresponding property is stored within the action's input schema.
	External types.Property
}

// SchedulePeriod represents a scheduler period in minutes.
// Valid values are 5, 15, 30, 60, 120, 180, 360, 480, 720, and 1440.
type SchedulePeriod int

// MarshalJSON implements the json.Marshaler interface.
// It panics if period is not a valid SchedulePeriod value.
func (period SchedulePeriod) MarshalJSON() ([]byte, error) {
	return []byte(`"` + period.String() + `"`), nil
}

// String returns the string representation of period.
// It panics if period is not a valid SchedulePeriod value.
func (period SchedulePeriod) String() string {
	switch period {
	case 5:
		return "5m"
	case 15:
		return "15m"
	case 30:
		return "30m"
	case 60:
		return "1h"
	case 120:
		return "2h"
	case 180:
		return "3h"
	case 360:
		return "6h"
	case 480:
		return "8h"
	case 720:
		return "12h"
	case 1440:
		return "24h"
	}
	panic("invalid schedule period")
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (period *SchedulePeriod) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("json: cannot scan a %T value into an api.SchedulePeriod value", v)
	}
	var p SchedulePeriod
	switch s {
	case "5m":
		p = 5
	case "15m":
		p = 15
	case "30m":
		p = 30
	case "1h":
		p = 60
	case "2h":
		p = 120
	case "3h":
		p = 180
	case "6h":
		p = 360
	case "8h":
		p = 480
	case "12h":
		p = 720
	case "24h":
		p = 1440
	default:
		return fmt.Errorf("json: invalid apis.SchedulePeriod: %s", s)
	}
	*period = p
	return nil
}

// canBeUsedAsAsMatchingProp reports whether a type with kind k can be used as a
// matching property for the export.
func canBeUsedAsAsMatchingProp(k types.Kind) bool {
	// Only integers, UUIDs and texts are allowed.
	return k == types.IntKind || k == types.UintKind || k == types.UUIDKind || k == types.TextKind
}

// importsUsersIdentitiesFromEvents reports whether a connector with the given
// type, on a connection with the given role, with an action with the given
// target, imports users identities from events.
func importsUsersIdentitiesFromEvents(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	if role == state.Source && target == state.Users {
		switch connectorType {
		case state.MobileType, state.ServerType, state.WebsiteType:
			return true
		}
	}
	return false
}

// onlyForMatching returns a schema which contains only the properties of schema
// which can be used for the apps export matching.
//
// Returns an invalid schema in case none of the properties of schema can be
// used.
func onlyForMatching(schema types.Type) types.Type {
	props := []types.Property{}
	for _, p := range schema.Properties() {
		if canBeUsedAsAsMatchingProp(p.Type.Kind()) {
			props = append(props, p)
		}
	}
	if len(props) == 0 {
		return types.Type{}
	}
	return types.Object(props)
}

// transformationFunctionName returns the name the transformation function for
// an action in the specified language.
//
// Keep in sync with the function having the same name in the mappings package.
func transformationFunctionName(action int, language state.Language) string {
	var ext string
	switch language {
	case state.JavaScript:
		ext = ".js"
	case state.Python:
		ext = ".py"
	default:
		panic("unexpected language")
	}
	return "action-" + strconv.Itoa(action) + ext
}

// validateTimestampFormat validates the given timestamp format for importing
// files, returning an error in case the format is not valid.
//
// NOTE: keep in sync with the function 'apis/connectors.parseTimestamp'.
func validateTimestampFormat(format string) error {
	switch format {
	case
		"DateTime",
		"DateOnly",
		"ISO8601",
		"Excel":
		return nil
	}
	if format == "" {
		return errors.New("timestamp format cannot be empty")
	}
	if !utf8.ValidString(format) {
		return errors.New("timestamp format must be UTF-8 valid")
	}
	if utf8.RuneCountInString(format) > 64 {
		return errors.New("timestamp format is longer than 64 runes")
	}
	if !strings.Contains(format, "%") {
		return fmt.Errorf("timestamp format %q is not a valid timestamp format", format)
	}
	if format[0] != '\'' || format[len(format)-1] != '\'' {
		return fmt.Errorf("timestamp strptime format must be enclosed between \"'\" characters")
	}
	return nil
}
