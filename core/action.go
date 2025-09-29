//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package core

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/backoff"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/internal/connectors"
	"github.com/meergo/meergo/core/internal/datastore"
	"github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers/mappings"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"

	"github.com/google/uuid"
)

// Action represents an action of a connection.
type Action struct {
	core                 *Core
	action               *state.Action
	connection           *Connection
	ID                   int             `json:"id"`
	Connector            string          `json:"connector"`
	ConnectorType        ConnectorType   `json:"connectorType"`
	Connection           int             `json:"connection"`
	ConnectionRole       Role            `json:"connectionRole"`
	Target               Target          `json:"target"`
	Name                 string          `json:"name"`
	Enabled              bool            `json:"enabled"`
	EventType            *string         `json:"eventType"`
	Running              bool            `json:"running"`
	ScheduleStart        *int            `json:"scheduleStart"`
	SchedulePeriod       *SchedulePeriod `json:"schedulePeriod"`
	InSchema             types.Type      `json:"inSchema"`
	OutSchema            types.Type      `json:"outSchema"`
	Filter               *Filter         `json:"filter"`
	Transformation       *Transformation `json:"transformation"`
	Query                *string         `json:"query"`
	Format               string          `json:"format"`
	Path                 *string         `json:"path"`
	Sheet                *string         `json:"sheet"`
	Compression          Compression     `json:"compression"`
	OrderBy              *string         `json:"orderBy"`
	ExportMode           *ExportMode     `json:"exportMode"`
	Matching             *Matching       `json:"matching"`
	UpdateOnDuplicates   *bool           `json:"updateOnDuplicates"`
	TableName            *string         `json:"tableName"`
	TableKey             *string         `json:"tableKey"`
	IdentityColumn       *string         `json:"identityColumn"`
	LastChangeTimeColumn *string         `json:"lastChangeTimeColumn"`
	LastChangeTimeFormat *string         `json:"lastChangeTimeFormat"`
	Incremental          bool            `json:"incremental"`
}

// Matching establishes a relationship between a property in Meergo (input
// property) and a corresponding property in the app (output property) used
// during an export. This relationship determines whether a user or group in
// Meergo exists in the app and identifies the corresponding user or group in
// the app.
//
// The input property should be a property in the user schema, while the output
// property should be a property in the source schema of the connection.
// If the export mode includes "Create," the output property should also exist
// in the destination schema with the same type. However, the API does not
// check these conditions. It only requires that the input property is present
// in the input schema and the output property is present in the output schema.
//
// Note: The output property cannot be directly utilized in the action's
// transformation. During the export process, an implicit transformation maps
// the value of the input property to the output property. Only specific type
// conversions are permitted, which restrict the compatible types for these
// properties.
//
// Supported conversions:
//   - int to int, uint, text
//   - uint to int, uint, text
//   - text to int uint, uuid, text
//   - uuid to uuid, text
type Matching struct {
	In  string `json:"in"`  // name of the property in the input schema
	Out string `json:"out"` // name of the property in the output schema
}

// Language represents a transformation language. Valid values are "JavaScript"
// and "Python".
type Language string

// TransformationFunction represents a transformation function.
type TransformationFunction struct {
	Language     Language `json:"language"`
	Source       string   `json:"source"` // Source cannot be longer than MaxFunctionSourceSize runes.
	PreserveJSON bool     `json:"preserveJSON"`
	InPaths      []string `json:"inPaths"`
	OutPaths     []string `json:"outPaths"`
}

// Transformation represents a transformation.
type Transformation struct {
	Mapping  map[string]string       `json:"mapping,format:emitnull"`
	Function *TransformationFunction `json:"function"`
}

// ExportMode represents one of the three export modes.
type ExportMode string

const (
	CreateOnly     ExportMode = "CreateOnly"
	UpdateOnly     ExportMode = "UpdateOnly"
	CreateOrUpdate ExportMode = "CreateOrUpdate"
)

// Target represents a target.
type Target int

const (
	TargetEvent Target = iota + 1
	TargetUser
	TargetGroup
)

// MarshalJSON implements the json.Marshaler interface.
func (at Target) MarshalJSON() ([]byte, error) {
	return []byte(`"` + at.String() + `"`), nil
}

func (at Target) String() string {
	switch at {
	case TargetEvent:
		return "Event"
	case TargetUser:
		return "User"
	case TargetGroup:
		return "Group"
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
	case "Event":
		*at = TargetEvent
	case "User":
		*at = TargetUser
	case "Group":
		*at = TargetGroup
	default:
		return fmt.Errorf("json: invalid core.Target: %s", s)
	}
	return nil
}

// Delete deletes the action.
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
func (this *Action) Delete(ctx context.Context) error {
	this.core.mustBeOpen()
	c := this.action.Connection()
	n := state.DeleteAction{
		ID: this.action.ID,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		// Mark the action's function as discontinued.
		now := time.Now().UTC()
		_, err := tx.Exec(ctx, "INSERT INTO discontinued_functions (id, discontinued_at)\n"+
			"SELECT a.transformation_id, $1\n"+
			"FROM actions AS a\n"+
			"WHERE a.transformation_id != '' AND a.id = $2\n"+
			"ON CONFLICT (id) DO NOTHING", now, n.ID)
		if err != nil {
			return nil, err
		}
		// Delete the action.
		result, err := tx.Exec(ctx, "DELETE FROM actions WHERE id = $1", n.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("action %d does not exist", n.ID)
		}
		// Mark the action as deleted.
		if c.Role == state.Source && this.action.Target == state.TargetUser {
			_, err = tx.Exec(ctx, "UPDATE workspaces SET actions_to_purge = array_append(actions_to_purge, $1)"+
				" WHERE actions_to_purge IS NOT NULL", n.ID)
			if err != nil {
				return nil, err
			}
		}
		return n, nil
	})
	return err
}

// Execute executes the action, which must be an app, database, or file storage
// action with a target of User or Group. It starts an execution and returns its
// identifier. Both the action and its connection must be enabled and the action
// must not already be executing.
//
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
// It returns an errors.UnprocessableError error with code
//
//   - ActionDisabled, if the action is disabled.
//   - CannotExecuteIncrementally, if incremental requires a last change time
//     column.
//   - ExecutionInProgress, if the action is already in progress.
//   - InspectionMode, if the data warehouse is in inspection mode.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
func (this *Action) Execute(ctx context.Context, incremental *bool) (int, error) {
	this.core.mustBeOpen()
	c := this.action.Connection()
	if t := this.action.Target; t != state.TargetUser && t != state.TargetGroup {
		return 0, errors.BadRequest("action %d with target %s cannot be executed", this.action.ID, t)
	}
	typ := c.Connector().Type
	switch typ {
	case state.App, state.Database, state.FileStorage:
	default:
		return 0, errors.BadRequest("%s actions cannot be executed", strings.ToLower(typ.String()))
	}
	if incremental != nil {
		if c.Role == state.Destination {
			return 0, errors.BadRequest("incremental cannot be provided for destination actions")
		}
		if *incremental && typ != state.App && this.action.LastChangeTimeColumn == "" {
			return 0, errors.Unprocessable(CannotExecuteIncrementally, "incremental requires a last change time column")
		}
	}
	if !this.action.Enabled {
		return 0, errors.Unprocessable(ActionDisabled, "action %d is disabled", c.ID)
	}
	if _, ok := this.action.Execution(); ok {
		return 0, errors.Unprocessable(ExecutionInProgress, "action %d is already in progress", this.action.ID)
	}
	switch this.connection.store.Mode() {
	case state.Inspection:
		return 0, errors.Unprocessable(InspectionMode, "data warehouse is in inspection mode")
	case state.Maintenance:
		return 0, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
	}
	return this.createExecution(ctx, incremental)
}

// MarshalJSON encodes the Action as JSON.
func (this *Action) MarshalJSON() ([]byte, error) {
	type serializedAction struct {
		ID             int           `json:"id"`
		Name           string        `json:"name"`
		Connector      string        `json:"connector"`
		ConnectorType  ConnectorType `json:"connectorType"`
		Connection     int           `json:"connection"`
		ConnectionRole Role          `json:"connectionRole"`
		Target         Target        `json:"target"`
		Enabled        bool          `json:"enabled"`
	}
	a := serializedAction{
		ID:             this.ID,
		Name:           this.Name,
		Connector:      this.Connector,
		ConnectorType:  this.ConnectorType,
		Connection:     this.Connection,
		ConnectionRole: this.ConnectionRole,
		Target:         this.Target,
		Enabled:        this.Enabled,
	}
	var serialized any
	if a.ConnectionRole == Source {
		if a.Target == TargetUser {
			switch a.ConnectorType {
			case App:
				serialized = struct {
					serializedAction
					Filter         *Filter         `json:"filter"`
					Incremental    bool            `json:"incremental"`
					Transformation Transformation  `json:"transformation"`
					InSchema       types.Type      `json:"inSchema"`
					OutSchema      types.Type      `json:"outSchema"`
					Running        bool            `json:"running"`
					ScheduleStart  *int            `json:"scheduleStart"`
					SchedulePeriod *SchedulePeriod `json:"schedulePeriod"`
				}{
					serializedAction: a,
					Filter:           this.Filter,
					Incremental:      this.Incremental,
					Transformation:   *this.Transformation,
					InSchema:         this.InSchema,
					OutSchema:        this.OutSchema,
					Running:          this.Running,
					ScheduleStart:    this.ScheduleStart,
					SchedulePeriod:   this.SchedulePeriod,
				}
			case Database:
				serialized = struct {
					serializedAction
					Query                string          `json:"query"`
					IdentityColumn       string          `json:"identityColumn"`
					LastChangeTimeColumn *string         `json:"lastChangeTimeColumn"`
					LastChangeTimeFormat *string         `json:"lastChangeTimeFormat"`
					Incremental          bool            `json:"incremental"`
					Transformation       Transformation  `json:"transformation"`
					InSchema             types.Type      `json:"inSchema"`
					OutSchema            types.Type      `json:"outSchema"`
					Running              bool            `json:"running"`
					ScheduleStart        *int            `json:"scheduleStart"`
					SchedulePeriod       *SchedulePeriod `json:"schedulePeriod"`
				}{
					serializedAction:     a,
					Query:                *this.Query,
					IdentityColumn:       *this.IdentityColumn,
					LastChangeTimeColumn: this.LastChangeTimeColumn,
					LastChangeTimeFormat: this.LastChangeTimeFormat,
					Incremental:          this.Incremental,
					Transformation:       *this.Transformation,
					InSchema:             this.InSchema,
					OutSchema:            this.OutSchema,
					Running:              this.Running,
					ScheduleStart:        this.ScheduleStart,
					SchedulePeriod:       this.SchedulePeriod,
				}
			case FileStorage:
				serialized = struct {
					serializedAction
					Format               string          `json:"format"`
					Path                 string          `json:"path"`
					Sheet                *string         `json:"sheet"`
					Compression          Compression     `json:"compression"`
					Filter               *Filter         `json:"filter"`
					IdentityColumn       string          `json:"identityColumn"`
					LastChangeTimeColumn *string         `json:"lastChangeTimeColumn"`
					LastChangeTimeFormat *string         `json:"lastChangeTimeFormat"`
					Incremental          bool            `json:"incremental"`
					Transformation       Transformation  `json:"transformation"`
					InSchema             types.Type      `json:"inSchema"`
					OutSchema            types.Type      `json:"outSchema"`
					Running              bool            `json:"running"`
					ScheduleStart        *int            `json:"scheduleStart"`
					SchedulePeriod       *SchedulePeriod `json:"schedulePeriod"`
				}{
					serializedAction:     a,
					Format:               this.Format,
					Path:                 *this.Path,
					Sheet:                this.Sheet,
					Compression:          this.Compression,
					Filter:               this.Filter,
					IdentityColumn:       *this.IdentityColumn,
					LastChangeTimeColumn: this.LastChangeTimeColumn,
					LastChangeTimeFormat: this.LastChangeTimeFormat,
					Incremental:          this.Incremental,
					Transformation:       *this.Transformation,
					InSchema:             this.InSchema,
					OutSchema:            this.OutSchema,
					Running:              this.Running,
					ScheduleStart:        this.ScheduleStart,
					SchedulePeriod:       this.SchedulePeriod,
				}
			case SDK:
				serialized = struct {
					serializedAction
					Filter         *Filter         `json:"filter"`
					Transformation *Transformation `json:"transformation"`
					InSchema       types.Type      `json:"inSchema"`
					OutSchema      types.Type      `json:"outSchema"`
				}{
					serializedAction: a,
					Filter:           this.Filter,
					Transformation:   this.Transformation,
					InSchema:         this.InSchema,
					OutSchema:        this.OutSchema,
				}
			}
		}
		if a.Target == TargetEvent {
			serialized = struct {
				serializedAction
				Filter   *Filter    `json:"filter"`
				InSchema types.Type `json:"inSchema"`
			}{
				serializedAction: a,
				Filter:           this.Filter,
				InSchema:         this.InSchema,
			}
		}
	}
	if a.ConnectionRole == Destination {
		if a.Target == TargetUser {
			switch a.ConnectorType {
			case App:
				serialized = struct {
					serializedAction
					Filter             *Filter         `json:"filter"`
					Matching           Matching        `json:"matching"`
					ExportMode         ExportMode      `json:"exportMode"`
					UpdateOnDuplicates bool            `json:"updateOnDuplicates"`
					Transformation     Transformation  `json:"transformation"`
					InSchema           types.Type      `json:"inSchema"`
					OutSchema          types.Type      `json:"outSchema"`
					Running            bool            `json:"running"`
					ScheduleStart      *int            `json:"scheduleStart"`
					SchedulePeriod     *SchedulePeriod `json:"schedulePeriod"`
				}{
					serializedAction:   a,
					Filter:             this.Filter,
					Matching:           *this.Matching,
					ExportMode:         *this.ExportMode,
					UpdateOnDuplicates: *this.UpdateOnDuplicates,
					Transformation:     *this.Transformation,
					InSchema:           this.InSchema,
					OutSchema:          this.OutSchema,
					Running:            this.Running,
					ScheduleStart:      this.ScheduleStart,
					SchedulePeriod:     this.SchedulePeriod,
				}
			case Database:
				serialized = struct {
					serializedAction
					Filter         *Filter         `json:"filter"`
					TableName      string          `json:"tableName"`
					TableKey       string          `json:"tableKey"`
					Transformation Transformation  `json:"transformation"`
					InSchema       types.Type      `json:"inSchema"`
					OutSchema      types.Type      `json:"outSchema"`
					Running        bool            `json:"running"`
					ScheduleStart  *int            `json:"scheduleStart"`
					SchedulePeriod *SchedulePeriod `json:"schedulePeriod"`
				}{
					serializedAction: a,
					Filter:           this.Filter,
					TableName:        *this.TableName,
					TableKey:         *this.TableKey,
					Transformation:   *this.Transformation,
					InSchema:         this.InSchema,
					OutSchema:        this.OutSchema,
					Running:          this.Running,
					ScheduleStart:    this.ScheduleStart,
					SchedulePeriod:   this.SchedulePeriod,
				}
			case FileStorage:
				serialized = struct {
					serializedAction
					Format         string          `json:"format"`
					Path           string          `json:"path"`
					Sheet          *string         `json:"sheet"`
					Compression    Compression     `json:"compression"`
					OrderBy        string          `json:"orderBy"`
					Filter         *Filter         `json:"filter"`
					InSchema       types.Type      `json:"inSchema"`
					Running        bool            `json:"running"`
					ScheduleStart  *int            `json:"scheduleStart"`
					SchedulePeriod *SchedulePeriod `json:"schedulePeriod"`
				}{
					serializedAction: a,
					Format:           this.Format,
					Path:             *this.Path,
					Sheet:            this.Sheet,
					Compression:      this.Compression,
					OrderBy:          *this.OrderBy,
					Filter:           this.Filter,
					InSchema:         this.InSchema,
					Running:          this.Running,
					ScheduleStart:    this.ScheduleStart,
					SchedulePeriod:   this.SchedulePeriod,
				}
			}
		}
		if a.Target == TargetEvent {
			serialized = struct {
				serializedAction
				EventType      string          `json:"eventType"`
				Filter         *Filter         `json:"filter"`
				Transformation *Transformation `json:"transformation"`
				InSchema       types.Type      `json:"inSchema"`
				OutSchema      types.Type      `json:"outSchema"`
			}{
				serializedAction: a,
				EventType:        *this.EventType,
				Filter:           this.Filter,
				Transformation:   this.Transformation,
				InSchema:         this.InSchema,
				OutSchema:        this.OutSchema,
			}
		}
	}
	if serialized == nil {
		panic(fmt.Sprintf("unexpected role: %s, target: %s, type: %s", a.ConnectionRole, a.Target, a.ConnectorType))
	}
	return json.Marshal(serialized)
}

// ServeUI serves the user interface for the format settings of a file action.
// event is the event to be served and settings are the updated settings.
//
// It returns an errors.UnprocessableError error with code:
//
//   - EventNotExist, if the event does not exist.
//   - InvalidSettings, if the settings are not valid.
func (this *Action) ServeUI(ctx context.Context, event string, settings json.Value) (json.Value, error) {
	this.core.mustBeOpen()
	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	connection := this.action.Connection()
	connector := connection.Connector()
	if connector.Type != state.FileStorage {
		return nil, errors.BadRequest("cannot serve the UI of an action on a %s connection", connector.Type)
	}
	formatConnector := this.action.Format()
	if connection.Role == state.Source && !formatConnector.HasSourceSettings {
		return nil, errors.BadRequest("connector %s does not have source settings", formatConnector.Name)
	}
	if connection.Role == state.Destination && !formatConnector.HasDestinationSettings {
		return nil, errors.BadRequest("connector %s does not have destination settings", formatConnector.Name)
	}
	ui, err := this.core.connectors.ServeActionUI(ctx, this.action, event, settings)
	if err != nil {
		if err == meergo.ErrUIEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s connector", event, connector.Name)
		} else {
			switch err.(type) {
			case *meergo.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
			case *connectors.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
		}
		return nil, err
	}
	return ui, nil
}

// SetSchedulePeriod sets the schedule period, in minutes, of the action. The
// action must be a User or Group action and period can be 0, 5, 15, 30, 60,
// 120, 180, 360, 480, 720, or 1440. The schedular is disabled if period is nil.
func (this *Action) SetSchedulePeriod(ctx context.Context, period *SchedulePeriod) error {
	this.core.mustBeOpen()
	switch this.action.Target {
	case state.TargetUser, state.TargetGroup:
	default:
		return errors.BadRequest("cannot set schedule period of a %s action", this.action.Target)
	}
	n := state.SetActionSchedulePeriod{
		ID: this.action.ID,
	}
	if period != nil {
		switch *period {
		case 5, 15, 30, 60, 120, 180, 360, 480, 720, 1440:
			n.SchedulePeriod = int16(*period)
		default:
			return errors.BadRequest("schedule period %d is not valid", period)
		}
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE actions SET schedule_period = $1 WHERE id = $2 AND schedule_period <> $1", n.SchedulePeriod, n.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, nil
		}
		return n, nil
	})
	return err
}

// SetStatus sets the status of the action.
func (this *Action) SetStatus(ctx context.Context, enabled bool) error {
	this.core.mustBeOpen()
	if enabled == this.action.Enabled {
		return nil
	}
	n := state.SetActionStatus{
		ID:      this.action.ID,
		Enabled: enabled,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE actions SET enabled = $1 WHERE id = $2 AND enabled <> $1", n.Enabled, n.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, nil
		}
		return n, nil
	})
	return err
}

// Update updates the action.
//
// Refer to the specifications in the file "core/Actions.csv" for more details.
//
// It returns an errors.UnprocessableError error with code:
//
//   - FormatNotExist, if the format does not exist.
//   - InvalidSettings, if the settings are not valid.
//   - UnsupportedLanguage, if the transformation language is not supported.
func (this *Action) Update(ctx context.Context, action ActionToSet) error {

	this.core.mustBeOpen()

	// Retrieve the file format, if specified in the action.
	var format *state.Connector
	if action.Format != "" {
		format, _ = this.core.state.Connector(action.Format)
	}

	c := this.action.Connection()

	// Validate the action.
	v := validationState{}
	v.target = this.action.Target
	v.connection.role = c.Role
	v.connection.connector.typ = c.Connector().Type
	if format != nil {
		v.format.typ = format.Type
		if c.Role == state.Source {
			v.format.targets = format.SourceTargets
		} else {
			v.format.targets = format.DestinationTargets
		}
		v.format.hasSheets = format.HasSheets
		v.format.hasSettings = c.Role == state.Source && format.HasSourceSettings || c.Role == state.Destination && format.HasDestinationSettings
	}
	v.provider = this.core.functionProvider
	err := validateActionToSet(action, v)
	if err != nil {
		return err
	}

	// Determine the input schema.
	inSchema := action.InSchema
	importUserIdentitiesFromEvents := isImportingUserIdentitiesFromEvents(c.Connector().Type, c.Role, this.action.Target)
	importEventsIntoWarehouse := isImportingEventsIntoWarehouse(c.Connector().Type, c.Role, this.action.Target)
	dispatchEventsToApps := isDispatchingEventsToApps(c.Connector().Type, c.Role, this.action.Target)
	if importUserIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApps {
		inSchema = schemas.Event
	}

	n := state.UpdateAction{
		ID:                   this.action.ID,
		Name:                 action.Name,
		Enabled:              action.Enabled,
		InSchema:             inSchema,
		OutSchema:            action.OutSchema,
		Transformation:       toStateTransformation(action.Transformation, inSchema, action.OutSchema),
		Query:                action.Query,
		Format:               action.Format,
		Path:                 action.Path,
		Sheet:                action.Sheet,
		Compression:          state.Compression(action.Compression),
		OrderBy:              action.OrderBy,
		ExportMode:           state.ExportMode(action.ExportMode),
		Matching:             state.Matching(action.Matching),
		UpdateOnDuplicates:   action.UpdateOnDuplicates,
		TableName:            action.TableName,
		TableKey:             action.TableKey,
		IdentityColumn:       action.IdentityColumn,
		LastChangeTimeColumn: action.LastChangeTimeColumn,
		LastChangeTimeFormat: action.LastChangeTimeFormat,
		Incremental:          action.Incremental,
	}

	// Add the filter to the notification.
	if action.Filter != nil {
		n.Filter, _ = convertFilterToWhere(action.Filter, inSchema).MarshalJSON()
	}

	// Determine the format name, for file actions.
	var formatName *string
	if format != nil {
		name := format.Name
		formatName = &name
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
	if tr := action.Transformation; tr != nil && tr.Mapping != nil {
		mapping, err = json.Marshal(action.Transformation.Mapping)
		if err != nil {
			return err
		}
	}

	// Format settings.
	if format != nil && action.FormatSettings != nil {
		conf := &connectors.ConnectorConfig{
			Role: this.action.Connection().Role,
		}
		n.FormatSettings, err = this.core.connectors.UpdatedSettings(ctx, format, conf, action.FormatSettings)
		if err != nil {
			switch err.(type) {
			case *meergo.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
			case *connectors.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
			return err
		}
	}

	// Transformation.
	if fn := n.Transformation.Function; fn != nil {
		current := this.action.Transformation.Function
		if current == nil || fn.Language != current.Language {
			name := transformationFunctionName(n.ID)
			fn.ID, fn.Version, err = this.core.functionProvider.Create(ctx, name, fn.Language, fn.Source)
			if err != nil {
				return err
			}
		} else if fn.Source != current.Source {
			fn.ID = current.ID
			fn.Version, err = this.core.functionProvider.Update(ctx, fn.ID, fn.Source)
			if err != nil {
				return err
			}
		} else {
			// The function's language and source code should not be changed.
			// It will be verified during the transaction and assigned the current version.
		}
	}

	update := "UPDATE actions SET\n" +
		"name = $1, enabled = $2, in_schema = $3, out_schema = $4, filter = $5, " +
		"transformation_mapping = $6, transformation_id = $7, transformation_version = $8, transformation_language = $9, " +
		"transformation_source = $10, transformation_preserve_json = $11, transformation_in_paths = $12, " +
		"transformation_out_paths = $13, query = $14, format = $15, path = $16, sheet = $17, " +
		"compression = $18, order_by = $19, format_settings = $20, export_mode = $21, matching_in = $22, " +
		"matching_out = $23, update_on_duplicates = $24, table_name = $25, table_key = $26, " +
		"identity_column = $27, last_change_time_column = $28, last_change_time_format = $29, incremental = $30, " +
		"properties_to_unset = $31"
	if (c.Role == state.Source && !action.Incremental) || shouldReload(this.action, &n) {
		update += ", cursor = '0001-01-01 00:00:00+00'"
	}
	update += "\nWHERE id = $32"

	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		var function state.TransformationFunction
		if fn := n.Transformation.Function; fn != nil {
			var current state.TransformationFunction
			if fn.ID == "" {
				err := tx.QueryRow(ctx, "SELECT transformation_id, transformation_version, transformation_language, transformation_source "+
					"FROM actions WHERE id = $1", n.ID).Scan(&current.ID, &current.Version, &current.Language, &current.Source)
				if err != nil {
					return nil, err
				}
				if current.Language != fn.Language || current.Source != fn.Source {
					return nil, fmt.Errorf("abort update action %d: it was optimistically assumed that the transformation"+
						" had not changed, but it has indeed changed", n.ID)
				}
				fn.ID = current.ID
				fn.Version = current.Version
			}
			function = *fn
		}
		// Mark the action’s function as discontinued if its identifier changes.
		now := time.Now().UTC()
		_, err := tx.Exec(ctx, "INSERT INTO discontinued_functions (id, discontinued_at)\n"+
			"SELECT a.transformation_id, $1\n"+
			"FROM actions AS a\n"+
			"WHERE a.transformation_id != '' AND a.transformation_id != $2 AND a.id = $3\n"+
			"ON CONFLICT (id) DO NOTHING", now, function.ID, n.ID)
		if err != nil {
			return nil, err
		}
		// Determine properties that are no longer transformed.
		if c.Role == state.Source && this.action.Target == state.TargetUser {
			var prevOutPaths []string
			err := tx.QueryRow(ctx, "SELECT transformation_out_paths, properties_to_unset "+
				"FROM actions WHERE id = $1", n.ID).Scan(&prevOutPaths, &n.PropertiesToUnset)
			if err != nil {
				return nil, err
			}
			hasPath := make(map[string]struct{}, len(n.Transformation.OutPaths))
			for _, path := range n.Transformation.OutPaths {
				hasPath[path] = struct{}{}
			}
			for _, path := range prevOutPaths {
				if _, ok := hasPath[path]; !ok && !slices.Contains(n.PropertiesToUnset, path) {
					n.PropertiesToUnset = append(n.PropertiesToUnset, path)
				}
			}
		}
		// Update the action.
		result, err := tx.Exec(ctx, update,
			n.Name, n.Enabled, rawInSchema, rawOutSchema, n.Filter, mapping,
			function.ID, function.Version, function.Language, function.Source, function.PreserveJSON, n.Transformation.InPaths,
			n.Transformation.OutPaths, n.Query, formatName, n.Path, n.Sheet, n.Compression, n.OrderBy,
			string(n.FormatSettings), n.ExportMode, n.Matching.In, n.Matching.Out, n.UpdateOnDuplicates, n.TableName,
			n.TableKey, n.IdentityColumn, n.LastChangeTimeColumn, n.LastChangeTimeFormat, n.Incremental, n.PropertiesToUnset,
			n.ID,
		)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, nil
		}
		return n, nil
	})

	return err
}

// app returns the app of the action.
func (this *Action) app() *connectors.App {
	return this.core.connectors.App(this.action.Connection())
}

// createExecution creates an execution for the action and returns its
// identifier.
//
// It returns an errors.NotFoundError error if the action does not exist
// anymore.
// It returns an errors.UnprocessableError error with code ExecutionInProgress
// if the action is already in progress.
func (this *Action) createExecution(ctx context.Context, incremental *bool) (int, error) {

	n := state.ExecuteAction{
		Action:    this.action.ID,
		StartTime: time.Now().UTC(),
	}

	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		var function string
		var inc, executing bool
		var cursor time.Time
		err := tx.QueryRow(ctx, "SELECT a.transformation_id, a.incremental, a.cursor, e.id IS NOT NULL AND e.end_time IS NULL\n"+
			"FROM actions AS a\n"+
			"LEFT JOIN actions_executions AS e ON a.id = e.action\n"+
			"WHERE a.id = $1\n"+
			"ORDER BY e.id DESC\n"+
			"LIMIT 1", n.Action).Scan(&function, &inc, &cursor, &executing)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errors.NotFound("action %d does not exist", n.Action)
			}
			return nil, err
		}
		if executing {
			return nil, errors.Unprocessable(ExecutionInProgress, "execution of action %d is in progress", this.action.ID)
		}
		if incremental == nil {
			n.Incremental = inc
		} else {
			n.Incremental = *incremental
		}
		if n.Incremental {
			n.Cursor = cursor
		}
		err = tx.QueryRow(ctx, "INSERT INTO actions_executions (action, function, cursor, incremental, start_time, ping_time)\n"+
			"VALUES ($1, $2, $3, $4, $5, $5)\nRETURNING id", n.Action, function, n.Cursor, n.Incremental, n.StartTime).Scan(&n.ID)
		if err != nil {
			if db.IsForeignKeyViolation(err) {
				if db.ErrConstraintName(err) == "actions_executions_action_fkey" {
					err = errors.NotFound("action %d does not exit", n.Action)
				}
			}
			return nil, err
		}
		return n, nil
	})
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// database returns the database of the action.
// The caller must call the database's Close method when the database is no
// longer needed.
func (this *Action) database() *connectors.Database {
	a := this.action
	return this.core.connectors.Database(a.Connection())
}

// endExecution marks an action execution as completed, setting the specified
// error if any.
func (this *Action) endExecution(err error) {

	ctx := this.core.close.ctx

	execution, ok := this.action.Execution()
	if !ok {
		return
	}

	endTime := time.Now().UTC()

	// Handle the execution error, if any.
	var errorStep = metrics.ReceiveStep
	var errorMessage string
	if err != nil {
		if actionErr, ok := err.(*actionError); ok {
			errorStep = actionErr.step
			errorMessage = err.Error()
		} else {
			if ctx.Err() != nil {
				errorMessage = "execution has been cancelled"
			} else {
				errorMessage = "an internal error has occurred"
				slog.Error("core: cannot execute action", "action", this.action.ID, "execution", execution.ID, "err", err)
			}
		}
		this.core.metrics.Failed(errorStep, this.action.ID, 0, errorMessage)
	}

	// Waits for the metrics to be saved.
	this.core.metrics.WaitStore()

	n := state.EndActionExecution{
		ID:     execution.ID,
		Action: this.action.ID,
		Health: state.Healthy,
	}

	timeSlot := metrics.TimeSlotFromTime(execution.StartTime)

	// Mark the execution as completed, summarise the metrics, and send the end notification.
	bo := backoff.New(200)
	for bo.Next(ctx) {
		err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			res, err := tx.Exec(ctx,
				"WITH s AS (\n"+
					"\tSELECT COALESCE(SUM(passed_0), 0) as passed_0, COALESCE(SUM(passed_1), 0) as passed_1, COALESCE(SUM(passed_2), 0) as passed_2,"+
					" COALESCE(SUM(passed_3), 0) as passed_3, COALESCE(SUM(passed_4), 0) as passed_4, COALESCE(SUM(passed_5), 0) as passed_5,"+
					" COALESCE(SUM(failed_0), 0) as failed_0, COALESCE(SUM(failed_1), 0) as failed_1, COALESCE(SUM(failed_2), 0) as failed_2,"+
					" COALESCE(SUM(failed_3), 0) as failed_3, COALESCE(SUM(failed_4), 0) as failed_4, COALESCE(SUM(failed_5), 0) as failed_5\n"+
					" FROM actions_metrics\n"+
					"\tWHERE action = $2 AND timeslot >= $3\n"+
					")\n"+
					"UPDATE actions_executions AS e\n"+
					"SET function = '', end_time = $4,"+
					" passed_0 = e.passed_0 + s.passed_0, passed_1 = e.passed_1 + s.passed_1, passed_2 = e.passed_2 + s.passed_2,"+
					" passed_3 = e.passed_3 + s.passed_3, passed_4 = e.passed_4 + s.passed_4, passed_5 = e.passed_5 + s.passed_5,"+
					" failed_0 = e.failed_0 + s.failed_0, failed_1 = e.failed_1 + s.failed_1, failed_2 = e.failed_2 + s.failed_2,"+
					" failed_3 = e.failed_3 + s.failed_3, failed_4 = e.failed_4 + s.failed_4, failed_5 = e.failed_5 + s.failed_5,"+
					" error = $5\n"+
					"FROM s\n"+
					"WHERE id = $1 AND end_time IS NULL", execution.ID, this.action.ID, timeSlot, endTime, errorMessage)
			if err != nil {
				return nil, err
			}
			// Do nothing if the execution no longer exists or has already been closed.
			if res.RowsAffected() == 0 {
				return nil, nil
			}
			// Update the action's cursor.
			var exists bool
			err = tx.QueryRow(ctx, "UPDATE actions SET cursor = (SELECT cursor FROM actions_executions WHERE id = $1 LIMIT 1), health = $2 WHERE id = $3 RETURNING true",
				n.ID, n.Health, this.action.ID).Scan(&exists)
			if err != nil {
				if err == sql.ErrNoRows {
					// The action does not exist anymore.
					return nil, nil
				}
				return nil, err
			}
			return n, nil
		})
		if err != nil {
			if ctx.Err() != nil {
				// The context has been canceled.
				return
			}
			slog.Error(fmt.Sprintf("core: cannot end action execution, retrying after %s", bo.WaitTime()), "error", err)
			continue
		}
		break
	}

}

// file returns the file of the action.
func (this *Action) file() *connectors.File {
	return this.core.connectors.File(this.action)
}

// fromState serializes action into this.
func (this *Action) fromState(core *Core, store *datastore.Store, action *state.Action) {
	c := action.Connection()
	connector := c.Connector()
	this.core = core
	this.action = action
	this.connection = &Connection{core: core, store: store, connection: c}
	this.ID = action.ID
	this.Connector = connector.Name
	this.ConnectorType = ConnectorType(connector.Type)
	this.Connection = c.ID
	this.ConnectionRole = Role(c.Role)
	this.Target = Target(action.Target)
	this.Name = action.Name
	this.Enabled = action.Enabled
	if action.EventType != "" {
		et := action.EventType
		this.EventType = &et
	}
	_, this.Running = this.action.Execution()
	if action.Target == state.TargetUser || action.Target == state.TargetGroup {
		if action.SchedulePeriod != 0 {
			start := int(action.ScheduleStart)
			period := SchedulePeriod(action.SchedulePeriod)
			this.ScheduleStart = &start
			this.SchedulePeriod = &period
		}
	}
	this.InSchema = action.InSchema
	this.OutSchema = action.OutSchema
	if action.Filter != nil {
		this.Filter = convertWhereToFilter(action.Filter, action.InSchema)
	}
	if action.Transformation.Mapping != nil {
		this.Transformation = &Transformation{
			Mapping: maps.Clone(action.Transformation.Mapping),
		}
	}
	if function := action.Transformation.Function; function != nil {
		this.Transformation = &Transformation{
			Function: &TransformationFunction{
				Language:     Language(function.Language.String()),
				Source:       function.Source,
				PreserveJSON: function.PreserveJSON,
				InPaths:      slices.Clone(action.Transformation.InPaths),
				OutPaths:     slices.Clone(action.Transformation.OutPaths),
			},
		}
	}
	if action.Query != "" {
		query := action.Query
		this.Query = &query
	}
	if f := action.Format(); f != nil {
		this.Format = f.Name
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
		this.TableName = &table
	}
	if action.ExportMode != "" {
		mode := action.ExportMode
		matching := action.Matching
		updateOnDuplicates := action.UpdateOnDuplicates
		this.ExportMode = (*ExportMode)(&mode)
		this.Matching = (*Matching)(&matching)
		this.UpdateOnDuplicates = &updateOnDuplicates
	}
	if action.TableKey != "" {
		key := action.TableKey
		this.TableKey = &key
	}
	if action.IdentityColumn != "" {
		p := action.IdentityColumn
		this.IdentityColumn = &p
	}
	if action.LastChangeTimeColumn != "" {
		column := action.LastChangeTimeColumn
		this.LastChangeTimeColumn = &column
	}
	if action.LastChangeTimeFormat != "" {
		format := action.LastChangeTimeFormat
		this.LastChangeTimeFormat = &format
	}
	this.Incremental = action.Incremental
	if action.OrderBy != "" {
		p := action.OrderBy
		this.OrderBy = &p
	}
}

// setExecutionCursor sets the cursor of the action execution.
func (this *Action) setExecutionCursor(ctx context.Context, cursor time.Time) error {
	execution, _ := this.action.Execution()
	_, err := this.core.db.Exec(ctx, "UPDATE actions_executions SET cursor = $1 WHERE id = $2", cursor, execution.ID)
	return err
}

// ActionToSet represents an action to set in a connection, by creating a new
// action (using the method Connection.CreateAction) or updating an existing one
// (using the method Action.Update).
//
// Refer to the specifications in the file "core/Actions.csv" for more details.
type ActionToSet struct {

	// Name must be a non-empty valid UTF-8 encoded string and cannot be longer
	// than 60 runes.
	Name string `json:"name"`

	// Enabled indicates whether the action is enabled or not.
	Enabled bool `json:"enabled"`

	// Filter is the filter of the action, if it has one, otherwise is nil.
	Filter *Filter `json:"filter"`

	// InSchema is the input schema of the action.
	//
	// Please refer to the 'Actions.csv' file for a complete list of properties
	// that must be inside this schema, based on the connection and action type.
	InSchema types.Type `json:"inSchema"`

	// OutSchema is the output schema of the action.
	//
	// Please refer to the 'Actions.csv' file for a complete list of properties
	// that must be inside this schema, based on the connection and action type.
	OutSchema types.Type `json:"outSchema"`

	// Transformation is the mapping or function transformation, if it has one.
	//
	// Every action that supports transformations may have an associated mapping
	// or function, which are mutually exclusive.
	//
	// Please refer to the 'Actions.csv' file for details about this
	// transformation and the properties it eventually operates on, based on the
	// connection and the action type.
	Transformation *Transformation `json:"transformation"`

	// Query is the query of the action, if it has one, otherwise it is the
	// empty string. It cannot be longer than MaxQuerySize runes.
	Query string `json:"query"`

	// Format is the file format and corresponds to the name of a file connector.
	// For non-file actions, this must be empty.
	Format string `json:"format"`

	// Path is the path of the file. It cannot be longer than MaxFilePathSize
	// runes, and it is empty for non-file actions.
	Path string `json:"path"`

	// Sheet is the sheet name for multiple sheets file actions. It must be UTF-8
	// encoded, have a length in the range [1, 31], should not start or end with
	// "'", and cannot contain any of "*", "/", ":", "?", "[", "\", and "]". It is
	// empty for non-file and non-multipart sheets actions. Sheet names are
	// case-insensitive.
	Sheet string `json:"sheet"`

	// Compression is the compression of the action on file storage connections.
	// In any other case, must be 0.
	Compression Compression `json:"compression"`

	// OrderBy is the property path for which to order users when they are
	// exported to a file, and must therefore refer to a property of the
	// action's output schema (OutSchema). It cannot be longer than 1024 runes.
	// For actions that do not export users to file, this is the empty string.
	OrderBy string `json:"orderBy"`

	// FormatSettings represents the format settings of a file connector.
	// It must be nil if the connector does not have settings.
	FormatSettings json.Value `json:"formatSettings"`

	// Mode specifies, for apps, whether the export should create users or groups,
	// update them, or do both.
	ExportMode ExportMode `json:"exportMode"`

	// Matching defines a relationship between a property in Meergo ("in") and
	// a corresponding property in the app ("out") used during an export.
	Matching Matching `json:"matching"`

	// UpdateOnDuplicates indicates whether to proceed with the export even if
	// duplicate users or groups are found in the app.
	UpdateOnDuplicates bool `json:"updateOnDuplicates"`

	// TableName is the name of the table for the export and it is defined for
	// destination database-actions; in any other case, it is the empty string.
	// It cannot be longer than MaxTableNameSize runes.
	TableName string `json:"tableName"`

	// TableKey is the name of the property used as table key when exporting
	// users to databases.
	// It is the empty string for any other type of action.
	TableKey string `json:"tableKey"`

	// IdentityColumn is the property name used as identity when importing
	// from a file or from a database.
	// It cannot be longer than 1024 runes.
	IdentityColumn string `json:"identityColumn"`

	// LastChangeTimeColumn is the last change time column when importing
	// from a file or from a database. May be empty to indicate that no
	// properties should be used for reading the last change times. Also refer
	// to the documentation of LastChangeTimeFormat, which is strictly related
	// to this.
	// It cannot be longer than 1024 runes.
	LastChangeTimeColumn string `json:"lastChangeTimeColumn"`

	// LastChangeTimeFormat indicates the last change time value format for
	// parsing the value read from the last change time column.
	//
	// Represents a format when a LastChangeTimeColumn is provided and its
	// corresponding property kind is json or text, otherwise it is the empty
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
	// It cannot be longer than MaxLastChangeTimeFormatSize runes.
	LastChangeTimeFormat string `json:"lastChangeTimeFormat"`

	// Incremental determine whether users should be imported incrementally.
	// If false, users will be re-imported from scratch.
	Incremental bool `json:"incremental"`
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
		return fmt.Errorf("json: invalid core.SchedulePeriod: %s", s)
	}
	*period = p
	return nil
}

// isDispatchingEventsToApps reports whether a connector of the given type,
// on a connection with the given role, and an action with the given target,
// is dispatching events to apps.
func isDispatchingEventsToApps(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	return role == state.Destination && target == state.TargetEvent && connectorType == state.App
}

// isExportUsersToFile reports whether a connector of the given type, on a
// connection with the given role is exporting users into a file.
func isExportUsersToFile(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	return connectorType == state.FileStorage && role == state.Destination && target == state.TargetUser
}

// isImportingEventsIntoWarehouse reports whether a connector of the given type,
// on a connection with the given role, and an action with the given target, is
// importing events into the data warehouse.
func isImportingEventsIntoWarehouse(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	return role == state.Source && target == state.TargetEvent && connectorType == state.SDK
}

// isImportingUserIdentitiesFromEvents reports whether a connector of the
// given type, on a connection with the given role, and an action with the
// given target, is importing user identities from events.
func isImportingUserIdentitiesFromEvents(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	return role == state.Source && target == state.TargetUser && connectorType == state.SDK
}

// onlyForMatching returns a schema which contains only the properties of schema
// which can be used for the apps export matching.
//
// Returns an invalid schema in case none of the properties of schema can be
// used.
func onlyForMatching(schema types.Type) types.Type {
	return types.Filter(schema, func(p types.Property) bool {
		return canBeUsedAsMatchingProp(p.Type.Kind())
	})
}

// shouldReload determines if the next execution of the action requires
// reloading, based on whether the notification n is used to update the action.
func shouldReload(a *state.Action, n *state.UpdateAction) bool {
	if a.Target != state.TargetUser && a.Target != state.TargetGroup {
		return false
	}
	if a.ExportMode != n.ExportMode {
		return true
	}
	if a.Matching.In != n.Matching.In {
		return true
	}
	if a.Matching.Out != n.Matching.Out {
		return true
	}
	if a.Query != n.Query {
		return true
	}
	if f := a.Format(); f != nil && f.Name != n.Format {
		return true
	}
	if a.Path != n.Path || a.Sheet != n.Sheet {
		return true
	}
	if !bytes.Equal(a.FormatSettings, n.FormatSettings) {
		return true
	}
	if a.IdentityColumn != n.IdentityColumn {
		return true
	}
	if a.LastChangeTimeColumn != n.LastChangeTimeColumn {
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
		if filter, _ := a.Filter.MarshalJSON(); !bytes.Equal(filter, n.Filter) {
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
	if !slices.Equal(t1.InPaths, t2.InPaths) {
		return true
	}
	if !slices.Equal(t1.OutPaths, t2.OutPaths) {
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
// value. It does not perform a deep copy and may modify the passed
// transformation.
func toStateTransformation(transformation *Transformation, inSchema, outSchema types.Type) state.Transformation {
	var tr state.Transformation
	if transformation == nil {
		return tr
	}
	if m := transformation.Mapping; m != nil {
		m, _ := mappings.New(transformation.Mapping, inSchema, outSchema, false, nil)
		return state.Transformation{
			Mapping:  transformation.Mapping,
			InPaths:  m.InPaths(),
			OutPaths: m.OutPaths(),
		}
	}
	fn := transformation.Function
	slices.Sort(fn.InPaths)
	slices.Sort(fn.OutPaths)
	language := state.JavaScript
	if fn.Language == "Python" {
		language = state.Python
	}
	return state.Transformation{
		Function: &state.TransformationFunction{
			Language:     language,
			Source:       fn.Source,
			PreserveJSON: fn.PreserveJSON,
		},
		InPaths:  fn.InPaths,
		OutPaths: fn.OutPaths,
	}
}

// transformationFunctionName returns the name of the transformation function
// for an action in the specified language.
// If action is 0, the returned name refers to a preview transformation
// function.
func transformationFunctionName(action int) string {
	if action == 0 {
		return fmt.Sprintf("meergo_preview_%s", uuid.NewString())
	}
	now := time.Now().UTC()
	return fmt.Sprintf("meergo_action%d_%s-%09d", action, now.Format("2006-01-02T15-04-05"), now.Nanosecond())
}
