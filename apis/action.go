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
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/apis/connectors"
	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/events"
	"github.com/meergo/meergo/apis/filters"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/apis/transformers"
	"github.com/meergo/meergo/apis/transformers/mappings"
	"github.com/meergo/meergo/telemetry"
	"github.com/meergo/meergo/types"
)

// Action represents an action of a connection.
type Action struct {
	apis                     *APIs
	action                   *state.Action
	connection               *Connection
	ID                       int
	Connection               int
	Target                   Target
	Name                     string
	Enabled                  bool
	EventType                *string
	Running                  bool
	ScheduleStart            *int
	SchedulePeriod           *SchedulePeriod
	InSchema                 types.Type
	OutSchema                types.Type
	Filter                   *filters.Filter
	Transformation           Transformation
	Query                    *string
	Connector                string
	Path                     *string
	Sheet                    *string
	Compression              Compression
	Table                    *string
	TableKeyProperty         *string
	IdentityProperty         *string
	LastChangeTimeProperty   *string
	LastChangeTimeFormat     *string
	FileOrderingPropertyPath *string
	ExportMode               *ExportMode
	MatchingProperties       *MatchingProperties
	ExportOnDuplicatedUsers  *bool
}

// Language represents a transformation language. Valid values are "JavaScript"
// and "Python".
type Language string

// TransformationFunction represents a transformation function.
type TransformationFunction struct {
	Source        string
	Language      Language
	PreserveJSON  bool
	InProperties  []string
	OutProperties []string
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
		this.Filter = &filters.Filter{
			Logical:    filters.Logical(action.Filter.Logical),
			Conditions: make([]filters.Condition, len(action.Filter.Conditions)),
		}
		for i, condition := range action.Filter.Conditions {
			this.Filter.Conditions[i] = filters.Condition(condition)
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
			Source:        function.Source,
			Language:      Language(function.Language.String()),
			PreserveJSON:  function.PreserveJSON,
			InProperties:  slices.Clone(action.Transformation.InProperties),
			OutProperties: slices.Clone(action.Transformation.OutProperties),
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
	if action.TableKeyProperty != "" {
		key := action.TableKeyProperty
		this.TableKeyProperty = &key
	}
	if action.IdentityProperty != "" {
		p := action.IdentityProperty
		this.IdentityProperty = &p
	}
	if action.LastChangeTimeProperty != "" {
		column := action.LastChangeTimeProperty
		this.LastChangeTimeProperty = &column
	}
	if action.LastChangeTimeFormat != "" {
		format := action.LastChangeTimeFormat
		this.LastChangeTimeFormat = &format
	}
	if action.FileOrderingPropertyPath != "" {
		p := action.FileOrderingPropertyPath
		this.FileOrderingPropertyPath = &p
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
	c := this.action.Connection()
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
		if c.Role == state.Source && this.action.Target == state.Users {
			_, err = tx.Exec(ctx, "UPDATE workspaces SET actions_to_purge = array_append(actions_to_purge, $1)"+
				" WHERE actions_to_purge IS NOT NULL", n.ID)
			if err != nil {
				return err
			}
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
	if connector.Type != state.FileStorage {
		return nil, errors.BadRequest("cannot serve the UI of an action on a %s connection", connector.Type)
	}
	if !connector.HasUI {
		return nil, errors.BadRequest("connector %s does not have a UI", connector.Name)
	}
	ui, err := this.apis.connectors.ServeActionUI(ctx, this.action, event, values)
	if err != nil {
		if err == connectors.ErrUIEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s connector", event, connector.Name)
		} else {
			switch err.(type) {
			case connectors.InvalidUIValuesError:
				err = errors.Unprocessable(InvalidUIValues, "%s", err)
			case *connectors.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
		}
		return nil, err
	}
	return ui, nil
}

// Execute executes the action, which must be an app, database, or file storage
// action with a target of Users or Groups, creating an execution and returning
// its identifier.
//
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
// It returns an errors.UnprocessableError error with code
//
//   - ConnectionDisabled, if the connection is disabled.
//   - ExecutionInProgress, if the action is already in progress.
//   - InspectionMode, if the data warehouse is in inspection mode.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
func (this *Action) Execute(ctx context.Context, reload bool) (int, error) {
	this.apis.mustBeOpen()
	ctx, span := telemetry.TraceSpan(ctx, "Action.Execute", "id", this.action.ID, "reload", reload)
	defer span.End()
	c := this.action.Connection()
	if !c.Enabled {
		return 0, errors.Unprocessable(ConnectionDisabled, "connection %d is disabled", c.ID)
	}
	if _, ok := this.action.Execution(); ok {
		return 0, errors.Unprocessable(ExecutionInProgress, "action %d is already in progress", this.action.ID)
	}
	if t := this.action.Target; t != state.Users && t != state.Groups {
		return 0, errors.BadRequest("action %d with target %s cannot be executed", this.action.ID, t)
	}
	switch typ := c.Connector().Type; typ {
	case state.App, state.Database, state.FileStorage:
	default:
		return 0, errors.BadRequest("%s actions cannot be executed", strings.ToLower(typ.String()))
	}
	switch this.connection.store.Mode() {
	case state.Inspection:
		return 0, errors.Unprocessable(InspectionMode, "data warehouse is in inspection mode")
	case state.Maintenance:
		return 0, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
	}
	return this.addExecution(ctx, reload)
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

	// Retrieve the file connector, if specified in the action.
	var fileConnector *state.Connector
	if action.Connector != "" {
		fileConnector, _ = this.apis.state.Connector(action.Connector)
	}

	c := this.action.Connection()

	// Validate the action.
	v := validationState{}
	v.connection.role = c.Role
	v.connection.connector.typ = c.Connector().Type
	if fileConnector != nil {
		v.connector.typ = fileConnector.Type
		v.connector.hasSheets = fileConnector.HasSheets
		v.connector.hasUI = fileConnector.HasUI
	}
	v.provider = this.apis.transformerProvider
	err := validateAction(action, this.action.Target, v)
	if err != nil {
		return err
	}

	// Determine the input schema.
	inSchema := action.InSchema
	dispatchEventsToApps := isDispatchingEventsToApps(c.Connector().Type, c.Role, this.action.Target)
	importUserIdentitiesFromEvents := isImportingUserIdentitiesFromEvents(c.Connector().Type, c.Role, this.action.Target)
	if dispatchEventsToApps || importUserIdentitiesFromEvents {
		inSchema = events.Schema
	}

	span.Log("action validated successfully")

	n := state.SetAction{
		ID:                       this.action.ID,
		Name:                     action.Name,
		Enabled:                  action.Enabled,
		InSchema:                 inSchema,
		OutSchema:                action.OutSchema,
		Transformation:           toStateTransformation(action.Transformation),
		Query:                    action.Query,
		Connector:                action.Connector,
		Path:                     action.Path,
		Sheet:                    action.Sheet,
		Compression:              state.Compression(action.Compression),
		TableName:                action.TableName,
		TableKeyProperty:         action.TableKeyProperty,
		IdentityProperty:         action.IdentityProperty,
		LastChangeTimeProperty:   action.LastChangeTimeProperty,
		LastChangeTimeFormat:     action.LastChangeTimeFormat,
		FileOrderingPropertyPath: action.FileOrderingPropertyPath,
		ExportMode:               (*state.ExportMode)(action.ExportMode),
		ExportOnDuplicatedUsers:  action.ExportOnDuplicatedUsers,
	}
	if m := action.Transformation.Mapping; m != nil {
		m, _ := mappings.New(n.Transformation.Mapping, n.InSchema, n.OutSchema, nil)
		n.Transformation.InProperties = m.InProperties()
		n.Transformation.OutProperties = m.OutProperties()
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
			switch err.(type) {
			case connectors.InvalidUIValuesError:
				err = errors.Unprocessable(InvalidUIValues, "%s", err)
			case *connectors.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
			return err
		}
	}

	// Transformation.
	if fn := n.Transformation.Function; fn != nil {
		if this.action.Transformation.Function == nil {
			name := transformationFunctionName(n.ID, fn.Language)
			version, err := this.apis.transformerProvider.Create(ctx, name, fn.Source)
			if err == transformers.ErrFunctionExist {
				version, err = this.apis.transformerProvider.Update(ctx, name, fn.Source)
			}
			if err != nil {
				return err
			}
			n.Transformation.Function.Version = version
		} else if this.action.Transformation.Function.Source != fn.Source || this.action.Transformation.Function.Language != fn.Language {
			name := transformationFunctionName(n.ID, fn.Language)
			version, err := this.apis.transformerProvider.Update(ctx, name, fn.Source)
			if err == transformers.ErrFunctionNotExist {
				version, err = this.apis.transformerProvider.Create(ctx, name, fn.Source)
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

	// Check if the next execution of the action requires reloading.
	reload := shouldReload(this.action, &n)

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
			"transformation_version = $9, transformation_preserve_json = $10, transformation_in_properties = $11, "+
			"transformation_out_properties = $12, query = $13, connector = $14, path = $15, sheet = $16, "+
			"compression = $17, settings = $18, table_name = $19, table_key_property = $20, identity_property = $21, "+
			"reload = reload OR $22, last_change_time_property = $23, last_change_time_format = $24, "+
			"file_ordering_property_path = $25, export_mode = $26, matching_properties_internal = $27, "+
			"matching_properties_external = $28, export_on_duplicated_users = $29\nWHERE id = $30",
			n.Name, n.Enabled, rawInSchema, rawOutSchema, string(filter), mapping,
			function.Source, function.Language, function.Version, function.PreserveJSON, n.Transformation.InProperties,
			n.Transformation.OutProperties, n.Query, connectorName, n.Path, n.Sheet, n.Compression, string(n.Settings), n.TableName,
			n.TableKeyProperty, n.IdentityProperty, reload, n.LastChangeTimeProperty, n.LastChangeTimeFormat,
			n.FileOrderingPropertyPath, n.ExportMode, string(matchPropInternal), string(matchPropExternal),
			n.ExportOnDuplicatedUsers, n.ID,
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

// setExecutionCursor sets the cursor of the action execution.
func (this *Action) setExecutionCursor(ctx context.Context, cursor time.Time) error {
	execution, _ := this.action.Execution()
	_, err := this.apis.db.Exec(ctx, "UPDATE actions_executions SET cursor = $1 WHERE id = $2", cursor, execution.ID)
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

// file returns the file of the action.
func (this *Action) file() *connectors.File {
	return this.apis.connectors.File(this.action, this.connection.connection.Role)
}

// isLanguageSupported reports whether the transformation language of the action
// is supported. If the action does not have a transformation, it returns true.
func (this *Action) isLanguageSupported() bool {
	transformation := this.action.Transformation.Function
	if transformation == nil {
		return true
	}
	if this.apis.transformerProvider != nil && this.apis.transformerProvider.SupportLanguage(transformation.Language) {
		return true
	}
	return false
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
	//
	// Depending on the type of the action, this flag controls the enabling of
	// scheduling of action execution - for batch actions - or the enabling of
	// events receive/dispatching - for event based actions.
	Enabled bool

	// Filter is the filter of the action, if it has one, otherwise is nil.
	Filter *filters.Filter

	// InSchema is the input schema of the action.
	//
	// Please refer to the 'Actions.csv' file for a complete list of properties
	// that must be inside this schema, based on the connection and action type.
	InSchema types.Type

	// OutSchema is the output schema of the action.
	//
	// Please refer to the 'Actions.csv' file for a complete list of properties
	// that must be inside this schema, based on the connection and action type.
	OutSchema types.Type

	// Transformation is the mapping or function transformation, if it has one.
	//
	// Every action that supports transformations may have an associated mapping
	// or function, which are mutually exclusive.
	//
	// Please refer to the 'Actions.csv' file for details about this
	// transformation and the properties it eventually operates on, based on the
	// connection and the action type.
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

	// TableKeyProperty is the name of the property used as table key when
	// exporting users to databases.
	// It is the empty string for any other type of action.
	TableKeyProperty string

	// IdentityProperty is the property name used as identity when importing
	// from a file or from a database.
	// It cannot be longer than 1024 runes.
	IdentityProperty string

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
	//   - "ISO8601": the ISO 8601 format
	//   - "Excel": the Excel format, a float value stored in an Excel cell
	//     representing a date/datetime
	//   - a string containing a '%' character: the strftime() function format
	//
	// "Excel" format is only allowed for file actions.
	//
	// It cannot be longer than 64 runes.
	LastChangeTimeFormat string

	// FileOrderingPropertyPath is the property path for which to order users
	// when they are exported to a file, and must therefore refer to a property
	// of the action's output schema (OutSchema).
	// It cannot be longer than 1024 runes.
	// For actions that do not export users to file, this is the empty string.
	FileOrderingPropertyPath string

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

// isDispatchingEventsToApps reports whether a connector of the given type,
// on a connection with the given role, and an action with the given target,
// is dispatching events to apps.
func isDispatchingEventsToApps(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	return role == state.Destination && target == state.Events && connectorType == state.App
}

// isImportingUserIdentitiesFromEvents reports whether a connector of the
// given type, on a connection with the given role, and an action with the
// given target, is importing user identities from events.
func isImportingUserIdentitiesFromEvents(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	if role == state.Source && target == state.Users {
		switch connectorType {
		case state.Mobile, state.Server, state.Website:
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
	return types.SubsetFunc(schema, func(p types.Property) bool {
		return canBeUsedAsMatchingProp(p.Type.Kind())
	})
}

// shouldReload determines if the next execution of the action requires
// reloading, based on whether the notification n is used to modify the action.
func shouldReload(a *state.Action, n *state.SetAction) bool {
	if a.Target != state.Users {
		return false
	}
	c := a.Connection()
	if c.Role == state.Destination {
		if c.Connector().Type != state.App {
			return false
		}
		p1 := a.MatchingProperties.External
		p2 := n.MatchingProperties.External
		return p1.Name != p2.Name || !types.Equal(p1.Type, p2.Type)
	}
	if a.Query != n.Query {
		return true
	}
	if c := a.Connector(); c != nil && c.Name != n.Connector {
		return true
	}
	if a.Path != n.Path || a.Sheet != n.Sheet {
		return true
	}
	if !bytes.Equal(a.Settings, n.Settings) {
		return true
	}
	if a.IdentityProperty != n.IdentityProperty {
		return true
	}
	if a.LastChangeTimeProperty != n.LastChangeTimeProperty {
		return true
	}
	if a.LastChangeTimeFormat != n.LastChangeTimeFormat {
		return true
	}
	// Check the filters.
	if a.Filter != nil || n.Filter != nil {
		if a.Filter == nil || n.Filter == nil {
			return true
		}
		if a.Filter.Logical != n.Filter.Logical {
			return true
		}
		if !slices.Equal(a.Filter.Conditions, n.Filter.Conditions) {
			return true
		}
	}
	// Check the transformations.
	t1 := a.Transformation
	t2 := n.Transformation
	if !maps.Equal(t1.Mapping, t2.Mapping) {
		return true
	}
	if f1, f2 := t1.Function, t2.Function; f1 != nil || f2 != nil {
		if f1 == nil || f2 == nil {
			return true
		}
		if f1.Source != f2.Source {
			return true
		}
		if f1.Language != f2.Language {
			return true
		}
	}
	if !slices.Equal(t1.InProperties, t2.InProperties) {
		return true
	}
	if !slices.Equal(t1.OutProperties, t2.OutProperties) {
		return true
	}
	// Check the schemas.
	if !types.Equal(a.InSchema, n.InSchema) {
		return true
	}
	if !types.Equal(a.OutSchema, n.OutSchema) {
		return true
	}
	return false
}

// toStateTransformation converts a transformation to a state.Transformation
// value. It does not populate the input and output properties in case of
// mapping. It does not perform a deep copy and may modify the passed
// transformation.
func toStateTransformation(transformation Transformation) state.Transformation {
	var tr state.Transformation
	if function := transformation.Function; function != nil {
		slices.Sort(function.InProperties)
		slices.Sort(function.OutProperties)
		tr.Function = &state.TransformationFunction{
			Source: function.Source,
		}
		switch function.Language {
		case "JavaScript":
			tr.Function.Language = state.JavaScript
		case "Python":
			tr.Function.Language = state.Python
		}
		tr.Function.PreserveJSON = function.PreserveJSON
		tr.InProperties = function.InProperties
		tr.OutProperties = function.OutProperties
	} else {
		tr.Mapping = transformation.Mapping
	}
	return tr
}

// transformationFunctionName returns the name of the transformation function
// for an action in the specified language.
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
