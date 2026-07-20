// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/connections"
	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/transformers/mappings"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/google/uuid"
)

// eventPipelineSchema defines the event schema for pipelines.
// It excludes the kpid property.
var eventPipelineSchema types.Type

// eventPropertyNames lists all event property names.
// The first name is always "kpid".
var eventPropertyNames []string

func init() {
	properties := schemas.Event.Properties().Slice()
	eventPipelineSchema = types.Object(properties[1:])
	eventPropertyNames = make([]string, len(properties))
	for i, p := range properties {
		eventPropertyNames[i] = p.Name
	}
}

// Pipeline represents a pipeline of a connection.
type Pipeline struct {
	core               *Core
	pipeline           *state.Pipeline
	connection         *Connection
	ID                 string           `json:"id"`
	Connector          string           `json:"connector"`
	ConnectorType      ConnectorType    `json:"connectorType"`
	Connection         string           `json:"connection"`
	ConnectionRole     Role             `json:"connectionRole"`
	Target             Target           `json:"target"`
	Name               string           `json:"name"`
	Enabled            bool             `json:"enabled"`
	EventType          *string          `json:"eventType"`
	Running            bool             `json:"running"`
	ScheduleStart      *int             `json:"scheduleStart"`
	SchedulePeriod     *SchedulePeriod  `json:"schedulePeriod"`
	InSchema           types.Type       `json:"inSchema"`
	OutSchema          types.Type       `json:"outSchema"`
	Filter             *Filter          `json:"filter"`
	RequiredConsents   RequiredConsents `json:"requiredConsents"`
	Transformation     *Transformation  `json:"transformation"`
	Query              *string          `json:"query"`
	Format             string           `json:"format"`
	Path               *string          `json:"path"`
	Sheet              *string          `json:"sheet"`
	Compression        Compression      `json:"compression"`
	OrderBy            *string          `json:"orderBy"`
	ExportMode         *ExportMode      `json:"exportMode"`
	Matching           *Matching        `json:"matching"`
	UpdateOnDuplicates *bool            `json:"updateOnDuplicates"`
	TableName          *string          `json:"tableName"`
	TableKey           *string          `json:"tableKey"`
	UserIDColumn       *string          `json:"userIDColumn"`
	UpdatedAtColumn    *string          `json:"updatedAtColumn"`
	UpdatedAtFormat    *string          `json:"updatedAtFormat"`
	Incremental        bool             `json:"incremental"`
}

// Matching establishes a relationship between a property in Krenalis (input
// property) and a corresponding property in the application (output property)
// used during an export. This relationship determines whether a user or group
// in Krenalis exists in the application and identifies the corresponding user
// or group in the application.
//
// The input property should be a property in the profile schema, while the
// output property should be a property in the source schema of the connection.
// If the export mode includes "Create," the output property should also exist
// in the destination schema with the same type. However, the application does
// not check these conditions. It only requires that the input property is
// present in the input schema and the output property is present in the output
// schema.
//
// Note: The output property cannot be directly utilized in the pipeline's
// transformation. During the export process, an implicit transformation maps
// the value of the input property to the output property. Only specific type
// conversions are permitted, which restrict the compatible types for these
// properties.
//
// Supported conversions:
//   - int to int, string
//   - string to int, uuid, string
//   - uuid to uuid, string
type Matching struct {
	In  string `json:"in"`  // path of the property in the input schema
	Out string `json:"out"` // path of the property in the output schema
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

// RequiredConsents represents the consent purposes required by a pipeline
type RequiredConsents struct {
	Operator ConsentPurposesOperator `json:"operator"`
	Purposes []string                `json:"purposes"`
}

// ConsentPurposesOperator represents the logical operator applied to the
// consent purposes required by a pipeline.
type ConsentPurposesOperator string

const (
	PurposesNone ConsentPurposesOperator = ""
	PurposesAnd  ConsentPurposesOperator = "and"
	PurposesOr   ConsentPurposesOperator = "or"
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
	if bytes.Equal(data, null) {
		return errors.BadRequest("target cannot be null")
	}
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return errors.BadRequest(`target must be "Event", "User", or "Group"`)
	}
	switch s {
	case "Event":
		*at = TargetEvent
	case "User":
		*at = TargetUser
	case "Group":
		*at = TargetGroup
	default:
		return errors.BadRequest(`target must be "Event", "User", or "Group"`)
	}
	return nil
}

// Delete deletes the pipeline.
// It returns an errors.NotFoundError error if the pipeline does not exist
// anymore.
func (this *Pipeline) Delete(ctx context.Context) error {
	this.core.mustBeOpen()
	c := this.pipeline.Connection()
	n := state.DeletePipeline{
		ID: this.pipeline.ID,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		// Mark the pipeline's function as discontinued.
		now := time.Now().UTC()
		_, err := tx.Exec(ctx, "INSERT INTO discontinued_functions (id, discontinued_at)\n"+
			"SELECT p.transformation_id, $1\n"+
			"FROM pipelines AS p\n"+
			"WHERE p.transformation_id != '' AND p.id = $2\n"+
			"ON CONFLICT (id) DO NOTHING", now, n.ID)
		if err != nil {
			return nil, err
		}
		// Delete the pipeline.
		result, err := tx.Exec(ctx, "DELETE FROM pipelines WHERE id = $1", n.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("pipeline %s does not exist", n.ID)
		}
		// Mark the pipeline as deleted.
		if c.Role == state.Source && this.pipeline.Target == state.TargetUser {
			_, err = tx.Exec(ctx, "UPDATE workspaces SET pipelines_to_purge = array_append(pipelines_to_purge, $1)"+
				" WHERE id = $2", n.ID, c.Workspace().ID)
			if err != nil {
				return nil, err
			}
		}
		return n, nil
	})
	return err
}

// MarshalJSON encodes the Pipeline as JSON.
func (this *Pipeline) MarshalJSON() ([]byte, error) {
	type serializedPipeline struct {
		ID             string        `json:"id"`
		Name           string        `json:"name"`
		Connector      string        `json:"connector"`
		ConnectorType  ConnectorType `json:"connectorType"`
		Connection     string        `json:"connection"`
		ConnectionRole Role          `json:"connectionRole"`
		Target         Target        `json:"target"`
		Enabled        bool          `json:"enabled"`
	}
	p := serializedPipeline{
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
	if p.ConnectionRole == Source {
		if p.Target == TargetUser {
			switch p.ConnectorType {
			case Application:
				serialized = struct {
					serializedPipeline
					Filter         *Filter         `json:"filter"`
					Incremental    bool            `json:"incremental"`
					Transformation Transformation  `json:"transformation"`
					InSchema       types.Type      `json:"inSchema"`
					OutSchema      types.Type      `json:"outSchema"`
					Running        bool            `json:"running"`
					ScheduleStart  *int            `json:"scheduleStart"`
					SchedulePeriod *SchedulePeriod `json:"schedulePeriod"`
				}{
					serializedPipeline: p,
					Filter:             this.Filter,
					Incremental:        this.Incremental,
					Transformation:     *this.Transformation,
					InSchema:           this.InSchema,
					OutSchema:          this.OutSchema,
					Running:            this.Running,
					ScheduleStart:      this.ScheduleStart,
					SchedulePeriod:     this.SchedulePeriod,
				}
			case Database:
				serialized = struct {
					serializedPipeline
					Query           string          `json:"query"`
					UserIDColumn    string          `json:"userIDColumn"`
					UpdatedAtColumn *string         `json:"updatedAtColumn"`
					UpdatedAtFormat *string         `json:"updatedAtFormat"`
					Incremental     bool            `json:"incremental"`
					Transformation  Transformation  `json:"transformation"`
					InSchema        types.Type      `json:"inSchema"`
					OutSchema       types.Type      `json:"outSchema"`
					Running         bool            `json:"running"`
					ScheduleStart   *int            `json:"scheduleStart"`
					SchedulePeriod  *SchedulePeriod `json:"schedulePeriod"`
				}{
					serializedPipeline: p,
					Query:              *this.Query,
					UserIDColumn:       *this.UserIDColumn,
					UpdatedAtColumn:    this.UpdatedAtColumn,
					UpdatedAtFormat:    this.UpdatedAtFormat,
					Incremental:        this.Incremental,
					Transformation:     *this.Transformation,
					InSchema:           this.InSchema,
					OutSchema:          this.OutSchema,
					Running:            this.Running,
					ScheduleStart:      this.ScheduleStart,
					SchedulePeriod:     this.SchedulePeriod,
				}
			case FileStorage:
				serialized = struct {
					serializedPipeline
					Format          string          `json:"format"`
					Path            string          `json:"path"`
					Sheet           *string         `json:"sheet"`
					Compression     Compression     `json:"compression"`
					Filter          *Filter         `json:"filter"`
					UserIDColumn    string          `json:"userIDColumn"`
					UpdatedAtColumn *string         `json:"updatedAtColumn"`
					UpdatedAtFormat *string         `json:"updatedAtFormat"`
					Incremental     bool            `json:"incremental"`
					Transformation  Transformation  `json:"transformation"`
					InSchema        types.Type      `json:"inSchema"`
					OutSchema       types.Type      `json:"outSchema"`
					Running         bool            `json:"running"`
					ScheduleStart   *int            `json:"scheduleStart"`
					SchedulePeriod  *SchedulePeriod `json:"schedulePeriod"`
				}{
					serializedPipeline: p,
					Format:             this.Format,
					Path:               *this.Path,
					Sheet:              this.Sheet,
					Compression:        this.Compression,
					Filter:             this.Filter,
					UserIDColumn:       *this.UserIDColumn,
					UpdatedAtColumn:    this.UpdatedAtColumn,
					UpdatedAtFormat:    this.UpdatedAtFormat,
					Incremental:        this.Incremental,
					Transformation:     *this.Transformation,
					InSchema:           this.InSchema,
					OutSchema:          this.OutSchema,
					Running:            this.Running,
					ScheduleStart:      this.ScheduleStart,
					SchedulePeriod:     this.SchedulePeriod,
				}
			case SDK, Webhook:
				serialized = struct {
					serializedPipeline
					Filter           *Filter          `json:"filter"`
					RequiredConsents RequiredConsents `json:"requiredConsents"`
					Transformation   *Transformation  `json:"transformation"`
					InSchema         types.Type       `json:"inSchema"`
					OutSchema        types.Type       `json:"outSchema"`
				}{
					serializedPipeline: p,
					Filter:             this.Filter,
					RequiredConsents:   this.RequiredConsents,
					Transformation:     this.Transformation,
					InSchema:           this.InSchema,
					OutSchema:          this.OutSchema,
				}
			}
		}
		if p.Target == TargetEvent {
			serialized = struct {
				serializedPipeline
				Filter           *Filter          `json:"filter"`
				RequiredConsents RequiredConsents `json:"requiredConsents"`
				InSchema         types.Type       `json:"inSchema"`
			}{
				serializedPipeline: p,
				Filter:             this.Filter,
				RequiredConsents:   this.RequiredConsents,
				InSchema:           this.InSchema,
			}
		}
	}
	if p.ConnectionRole == Destination {
		if p.Target == TargetUser {
			switch p.ConnectorType {
			case Application:
				serialized = struct {
					serializedPipeline
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
					serializedPipeline: p,
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
					serializedPipeline
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
					serializedPipeline: p,
					Filter:             this.Filter,
					TableName:          *this.TableName,
					TableKey:           *this.TableKey,
					Transformation:     *this.Transformation,
					InSchema:           this.InSchema,
					OutSchema:          this.OutSchema,
					Running:            this.Running,
					ScheduleStart:      this.ScheduleStart,
					SchedulePeriod:     this.SchedulePeriod,
				}
			case FileStorage:
				serialized = struct {
					serializedPipeline
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
					serializedPipeline: p,
					Format:             this.Format,
					Path:               *this.Path,
					Sheet:              this.Sheet,
					Compression:        this.Compression,
					OrderBy:            *this.OrderBy,
					Filter:             this.Filter,
					InSchema:           this.InSchema,
					Running:            this.Running,
					ScheduleStart:      this.ScheduleStart,
					SchedulePeriod:     this.SchedulePeriod,
				}
			}
		}
		if p.Target == TargetEvent {
			serialized = struct {
				serializedPipeline
				EventType        string           `json:"eventType"`
				Filter           *Filter          `json:"filter"`
				RequiredConsents RequiredConsents `json:"requiredConsents"`
				Transformation   *Transformation  `json:"transformation"`
				InSchema         types.Type       `json:"inSchema"`
				OutSchema        types.Type       `json:"outSchema"`
			}{
				serializedPipeline: p,
				EventType:          *this.EventType,
				Filter:             this.Filter,
				RequiredConsents:   this.RequiredConsents,
				Transformation:     this.Transformation,
				InSchema:           this.InSchema,
				OutSchema:          this.OutSchema,
			}
		}
	}
	if serialized == nil {
		panic(fmt.Sprintf("unexpected role: %s, target: %s, type: %s", p.ConnectionRole, p.Target, p.ConnectorType))
	}
	return json.Marshal(serialized)
}

// Run starts a new run for the pipeline and returns its identifier. The
// pipeline must be an application, database, or file pipeline with a target of
// User or Group. It must be enabled and must not already have a run in
// progress.
//
// It returns an errors.NotFoundError if the pipeline no longer exists.
// It returns an errors.UnprocessableError with one of the following codes:
//
//   - CannotRunIncrementally, if incremental mode requires a last-change-time
//     column.
//   - InspectionMode, if the data warehouse is in inspection mode.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
//   - OrganizationDisabled, if the organization is disabled.
//   - PipelineAlreadyRunning, if the pipeline is already running.
//   - PipelineDisabled, if the pipeline is disabled.
func (this *Pipeline) Run(ctx context.Context, incremental *bool) (string, error) {
	this.core.mustBeOpen()
	c := this.pipeline.Connection()
	if t := this.pipeline.Target; t != state.TargetUser && t != state.TargetGroup {
		return "", errors.BadRequest("pipeline %s with target %s cannot be run", this.pipeline.ID, t)
	}
	typ := c.Connector().Type
	switch typ {
	case state.Application, state.Database, state.FileStorage:
	default:
		return "", errors.BadRequest("%s pipelines cannot be run", strings.ToLower(typ.String()))
	}
	if incremental != nil && c.Role == state.Destination {
		return "", errors.BadRequest("incremental cannot be provided for destination pipelines")
	}
	if org := c.Workspace().Organization(); !org.Enabled {
		return "", errors.Unprocessable(OrganizationDisabled, "organization %s is disabled", org.ID)
	}
	if incremental != nil && *incremental && typ != state.Application && this.pipeline.UpdatedAtColumn == "" {
		return "", errors.Unprocessable(CannotRunIncrementally, "incremental requires an update time column")
	}
	if !this.pipeline.Enabled {
		return "", errors.Unprocessable(PipelineDisabled, "pipeline %s is disabled", this.pipeline.ID)
	}
	if _, ok := this.pipeline.Run(); ok {
		return "", errors.Unprocessable(PipelineAlreadyRunning, "pipeline %s is already running", this.pipeline.ID)
	}
	switch this.connection.store.Mode() {
	case state.Inspection:
		return "", errors.Unprocessable(InspectionMode, "data warehouse is in inspection mode")
	case state.Maintenance:
		return "", errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
	}
	return this.createRun(ctx, incremental)
}

// ServeUI serves the user interface for the format settings of a file pipeline.
// event is the event to be served and settings are the updated settings.
//
// It returns an errors.UnprocessableError error with code:
//
//   - EventNotExist, if the event does not exist.
//   - InvalidSettings, if the settings are not valid.
func (this *Pipeline) ServeUI(ctx context.Context, event string, settings json.Value) (json.Value, error) {
	this.core.mustBeOpen()
	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	connection := this.pipeline.Connection()
	connector := connection.Connector()
	if connector.Type != state.FileStorage {
		return nil, errors.BadRequest("cannot serve the UI of a pipeline on a %s connection", connector.Type)
	}
	formatConnector := this.pipeline.Format()
	if connection.Role == state.Source && !formatConnector.HasSourceSettings {
		return nil, errors.BadRequest("connector %s does not have source settings", formatConnector.Code)
	}
	if connection.Role == state.Destination && !formatConnector.HasDestinationSettings {
		return nil, errors.BadRequest("connector %s does not have destination settings", formatConnector.Code)
	}
	ui, err := this.core.connections.ServePipelineUI(ctx, this.pipeline, event, settings)
	if err != nil {
		if err == connectors.ErrUIEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s connector", event, connector.Code)
		} else {
			switch err.(type) {
			case *connectors.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
			case *connections.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
		}
		return nil, err
	}
	return ui, nil
}

// SetSchedulePeriod sets the schedule period, in minutes, of the pipeline. The
// pipeline must be a User or Group pipeline and period can be 0, 5, 15, 30, 60,
// 120, 180, 360, 480, 720, or 1440. The schedular is disabled if period is nil.
func (this *Pipeline) SetSchedulePeriod(ctx context.Context, period *SchedulePeriod) error {
	this.core.mustBeOpen()
	switch this.pipeline.Target {
	case state.TargetUser, state.TargetGroup:
	default:
		return errors.BadRequest("cannot set schedule period of a %s pipeline", this.pipeline.Target)
	}
	n := state.SetPipelineSchedulePeriod{
		ID: this.pipeline.ID,
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
		result, err := tx.Exec(ctx, "UPDATE pipelines SET schedule_period = $1 WHERE id = $2 AND schedule_period <> $1", n.SchedulePeriod, n.ID)
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

// SetStatus sets the status of the pipeline.
func (this *Pipeline) SetStatus(ctx context.Context, enabled bool) error {
	this.core.mustBeOpen()
	if enabled == this.pipeline.Enabled {
		return nil
	}
	n := state.SetPipelineStatus{
		ID:      this.pipeline.ID,
		Enabled: enabled,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE pipelines SET enabled = $1 WHERE id = $2 AND enabled <> $1", n.Enabled, n.ID)
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

// Update updates the pipeline.
//
// Refer to the specifications in the file "core/Pipelines.csv" for more
// details.
//
// It returns an errors.UnprocessableError error with code:
//
//   - ConnectorsLimitReached, if the organization cannot have more connectors.
//   - ConsentPurposeNotExist, if a required consent purpose does not exist.
//   - FormatNotExist, if the format does not exist.
//   - InvalidSettings, if the settings are not valid.
//   - SchemaNotAligned, if the output schema is not aligned with the event type
//     schema.
//   - UnsupportedLanguage, if the transformation language is not supported.
func (this *Pipeline) Update(ctx context.Context, pipeline PipelineToSet) error {

	this.core.mustBeOpen()

	// Normalize the required consents.
	if pipeline.RequiredConsents.Purposes == nil {
		pipeline.RequiredConsents.Purposes = []string{}
	}

	// Retrieve the file format, if specified in the pipeline.
	var format *state.Connector
	if pipeline.Format != "" {
		format, _ = this.core.state.Connector(pipeline.Format)
	}

	c := this.pipeline.Connection()

	// Validate the pipeline.
	v := validationState{}
	v.target = this.pipeline.Target
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
	if len(pipeline.RequiredConsents.Purposes) > 0 {
		v.knownConsentPurposeIDs = knownConsentPurposeIDs(c.Workspace())
	}
	err := validatePipelineToSet(pipeline, v)
	if err != nil {
		return err
	}

	// Only for destination event pipeline checks that the out schema is aligned with the event type's schema.
	// See issue https://github.com/krenalis/krenalis/issues/2086.
	if this.pipeline.EventType != "" {
		app := this.application()
		eventTypeSchema, err := app.Schema(ctx, state.TargetEvent, this.pipeline.EventType)
		if err != nil {
			return err
		}
		err = schemas.CheckAlignment(pipeline.OutSchema, eventTypeSchema, new(state.CreateOnly))
		if err != nil {
			return errors.Unprocessable(SchemaNotAligned, "output schema is not aligned with the event type schema: %w", err)
		}
	}

	// Determine the input schema.
	inSchema := pipeline.InSchema
	importUserIdentitiesFromEvents := isImportingUserIdentitiesFromEvents(c.Connector().Type, c.Role, this.pipeline.Target)
	importEventsIntoWarehouse := isImportingEventsIntoWarehouse(c.Connector().Type, c.Role, this.pipeline.Target)
	dispatchEventsToApplications := isDispatchingEventsToApplications(c.Connector().Type, c.Role, this.pipeline.Target)
	if importUserIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApplications {
		inSchema = eventPipelineSchema
	}

	n := state.UpdatePipeline{
		ID:                 this.pipeline.ID,
		Name:               pipeline.Name,
		Enabled:            pipeline.Enabled,
		InSchema:           inSchema,
		OutSchema:          pipeline.OutSchema,
		RequiredConsents:   toStateRequiredConsents(pipeline.RequiredConsents),
		Transformation:     toStateTransformation(pipeline.Transformation, inSchema, pipeline.OutSchema),
		Query:              pipeline.Query,
		Format:             pipeline.Format,
		Path:               pipeline.Path,
		Sheet:              pipeline.Sheet,
		Compression:        state.Compression(pipeline.Compression),
		OrderBy:            pipeline.OrderBy,
		ExportMode:         state.ExportMode(pipeline.ExportMode),
		Matching:           state.Matching(pipeline.Matching),
		UpdateOnDuplicates: pipeline.UpdateOnDuplicates,
		TableName:          pipeline.TableName,
		TableKey:           pipeline.TableKey,
		UserIDColumn:       pipeline.UserIDColumn,
		UpdatedAtColumn:    pipeline.UpdatedAtColumn,
		UpdatedAtFormat:    pipeline.UpdatedAtFormat,
		Incremental:        pipeline.Incremental,
	}

	// Add the filter to the notification.
	if pipeline.Filter != nil {
		n.Filter, _ = convertFilterToWhere(pipeline.Filter, inSchema).MarshalJSON()
	}

	// Determine the format code, for file pipelines.
	var formatCode *string
	if format != nil {
		formatCode = new(format.Code)
	}

	// Marshal the input and the output schemas.
	rawInSchema, err := marshalSchema(inSchema)
	if err != nil {
		return err
	}
	rawOutSchema, err := marshalSchema(pipeline.OutSchema)
	if err != nil {
		return err
	}

	// Marshal the mapping.
	var mapping []byte
	if tr := pipeline.Transformation; tr != nil && tr.Mapping != nil {
		mapping, err = json.Marshal(pipeline.Transformation.Mapping)
		if err != nil {
			return err
		}
	}

	// Format settings.
	if format != nil && pipeline.FormatSettings != nil {
		conf := &connections.ConnectorConfig{
			Role: this.pipeline.Connection().Role,
		}
		n.FormatSettings, err = this.core.connections.UpdatedSettings(ctx, format, conf, pipeline.FormatSettings)
		if err != nil {
			switch err.(type) {
			case *connectors.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
			case *connections.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
			return err
		}
	}

	// Transformation.
	if fn := n.Transformation.Function; fn != nil {
		current := this.pipeline.Transformation.Function
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

	update := "UPDATE pipelines SET\n" +
		"name = $1, enabled = $2, in_schema = $3, out_schema = $4, filter = $5, required_consents = $6, required_consents_operator = $7, " +
		"transformation_mapping = $8, transformation_id = $9, transformation_version = $10, transformation_language = $11, " +
		"transformation_source = $12, transformation_preserve_json = $13, transformation_in_paths = $14, " +
		"transformation_out_paths = $15, query = $16, format = $17, path = $18, sheet = $19, " +
		"compression = $20, order_by = $21, format_settings = $22, export_mode = $23, matching_in = $24, " +
		"matching_out = $25, update_on_duplicates = $26, table_name = $27, table_key = $28, " +
		"user_id_column = $29, updated_at_column = $30, updated_at_format = $31, incremental = $32, " +
		"properties_to_unset = $33"
	if (c.Role == state.Source && !pipeline.Incremental) || shouldReload(this.pipeline, &n) {
		update += ", cursor = '0001-01-01 00:00:00+00'"
	}
	update += "\nWHERE id = $34"

	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		var function state.TransformationFunction
		if fn := n.Transformation.Function; fn != nil {
			var current state.TransformationFunction
			if fn.ID == "" {
				err := tx.QueryRow(ctx, "SELECT transformation_id, transformation_version, transformation_language, transformation_source "+
					"FROM pipelines WHERE id = $1", n.ID).Scan(&current.ID, &current.Version, &current.Language, &current.Source)
				if err != nil {
					return nil, err
				}
				if current.Language != fn.Language || current.Source != fn.Source {
					return nil, fmt.Errorf("abort update pipeline %s: it was optimistically assumed that the transformation"+
						" had not changed, but it has indeed changed", n.ID)
				}
				fn.ID = current.ID
				fn.Version = current.Version
			}
			function = *fn
		}
		if formatCode != nil {
			if err := checkUpdatePipelineConnectorLimit(ctx, tx, n.ID, *formatCode); err != nil {
				return nil, err
			}
		}
		// Mark the pipeline’s function as discontinued if its identifier changes.
		now := time.Now().UTC()
		_, err := tx.Exec(ctx, "INSERT INTO discontinued_functions (id, discontinued_at)\n"+
			"SELECT p.transformation_id, $1\n"+
			"FROM pipelines AS p\n"+
			"WHERE p.transformation_id != '' AND p.transformation_id != $2 AND p.id = $3\n"+
			"ON CONFLICT (id) DO NOTHING", now, function.ID, n.ID)
		if err != nil {
			return nil, err
		}
		// Determine properties that are no longer transformed.
		if c.Role == state.Source && this.pipeline.Target == state.TargetUser {
			var prevOutPaths []string
			err := tx.QueryRow(ctx, "SELECT transformation_out_paths, properties_to_unset "+
				"FROM pipelines WHERE id = $1", n.ID).Scan(&prevOutPaths, &n.PropertiesToUnset)
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
		// Update the pipeline.
		result, err := tx.Exec(ctx, update,
			n.Name, n.Enabled, rawInSchema, rawOutSchema, n.Filter, n.RequiredConsents.Purposes, n.RequiredConsents.Operator, mapping,
			function.ID, function.Version, function.Language, function.Source, function.PreserveJSON, n.Transformation.InPaths,
			n.Transformation.OutPaths, n.Query, formatCode, n.Path, n.Sheet, n.Compression, n.OrderBy,
			n.FormatSettings, n.ExportMode, n.Matching.In, n.Matching.Out, n.UpdateOnDuplicates, n.TableName,
			n.TableKey, n.UserIDColumn, n.UpdatedAtColumn, n.UpdatedAtFormat, n.Incremental, n.PropertiesToUnset,
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

// checkUpdatePipelineConnectorLimit checks whether the pipeline can be updated
// to use the given format without exceeding the organization's connector limit.
// The pipeline currently being updated is excluded from the usage count because
// its current format will be replaced by format.
func checkUpdatePipelineConnectorLimit(ctx context.Context, tx *db.Tx, pipeline, format string) error {

	var organization, currentFormat string
	var limit int

	err := tx.QueryRow(ctx, `
	SELECT o.id, o.connectors_limit, COALESCE(p.format, '') AS current_format
	FROM pipelines p
	JOIN connections c ON c.id = p.connection
	JOIN workspaces ws ON ws.id = c.workspace
	JOIN organizations o ON o.id = ws.organization
	WHERE p.id = $1
	FOR UPDATE OF o`, pipeline).Scan(&organization, &limit, &currentFormat)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	if format == "" || currentFormat == format {
		return nil
	}

	var count int
	var used bool

	err = tx.QueryRow(ctx, `
	SELECT
		COUNT(DISTINCT connector) AS connectors_count,
		COALESCE(BOOL_OR(connector = $3), false) AS connector_used
	FROM organization_connector_references
	WHERE organization = $1
	  AND NOT (resource_type = 'pipeline' AND resource = $2)`, organization, pipeline, format).Scan(&count, &used)
	if err != nil {
		return err
	}
	if !used && count >= limit {
		return errors.Unprocessable(ConnectorsLimitReached, "organization cannot use more than %d connectors", limit)
	}

	return nil
}

// application returns the application of the pipeline.
func (this *Pipeline) application() *connections.Application {
	return this.core.connections.Application(this.pipeline.Connection())
}

// createRun creates a new pipeline run and returns its identifier.
//
// It returns an errors.NotFoundError if the pipeline no longer exists.
// It returns an errors.UnprocessableError with one of the following codes:
//
//   - OrganizationDisabled, if the organization is disabled.
//   - PipelineAlreadyRunning, if the pipeline is already running.
//   - PipelineDisabled, if the pipeline is disabled.
func (this *Pipeline) createRun(ctx context.Context, incremental *bool) (string, error) {

	n := state.RunPipeline{
		Pipeline:  this.pipeline.ID,
		StartTime: time.Now().UTC(),
	}

	org := this.pipeline.Organization()

	for {
		n.ID = generateID[any](nil)
		err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			var function string
			var pipelineEnabled, organizationEnabled, inc bool
			var cursor time.Time
			// Verify that both the organization and the pipeline are enabled
			// and load the settings needed to initialize it.
			err := tx.QueryRow(ctx, "SELECT p.enabled, o.enabled, p.transformation_id, p.incremental, p.cursor\n"+
				"FROM pipelines AS p\n"+
				"INNER JOIN connections AS c ON p.connection = c.id\n"+
				"INNER JOIN workspaces AS w ON c.workspace = w.id\n"+
				"INNER JOIN organizations AS o ON w.organization = o.id\n"+
				"WHERE p.id = $1", n.Pipeline).Scan(&pipelineEnabled, &organizationEnabled, &function, &inc, &cursor)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil, errors.NotFound("pipeline %s does not exist", n.Pipeline)
				}
				return nil, err
			}
			if !organizationEnabled {
				return nil, errors.Unprocessable(OrganizationDisabled, "organization %s is disabled", org.ID)
			}
			if !pipelineEnabled {
				return nil, errors.Unprocessable(PipelineDisabled, "pipeline %s is disabled", this.pipeline.ID)
			}
			if incremental == nil {
				n.Incremental = inc
			} else {
				n.Incremental = *incremental
			}
			if n.Incremental {
				n.Cursor = cursor
			}
			// Create the pending run and initialize its ping time so the cleaner
			// does not consider it stale while it waits to be acquired by a node.
			_, err = tx.Exec(ctx, "INSERT INTO pipelines_runs (id, pipeline, function, cursor, incremental, start_time, ping_time)\n"+
				"VALUES ($1, $2, $3, $4, $5, $6, $6)", n.ID, n.Pipeline, function, n.Cursor, n.Incremental, n.StartTime)
			if err != nil {
				switch {
				case db.IsForeignKeyViolation(err):
					if db.ErrConstraintName(err) == "pipelines_runs_pipeline_fkey" {
						err = errors.NotFound("pipeline %s does not exit", n.Pipeline)
					}
				case db.IsUniqueViolation(err):
					if db.ErrConstraintName(err) == "pipelines_one_live_run_idx" {
						err = errors.Unprocessable(PipelineAlreadyRunning, "pipeline %s is already running", this.pipeline.ID)
					}
				}
				return nil, err
			}
			return n, nil
		})
		if err != nil {
			if db.IsUniqueViolation(err) && db.ErrConstraintName(err) == "pipelines_runs_pkey" {
				continue
			}
			return "", err
		}
		break
	}

	return n.ID, nil
}

// database returns the database of the pipeline.
// The caller must call the database's Close method when the database is no
// longer needed.
func (this *Pipeline) database() *connections.Database {
	p := this.pipeline
	return this.core.connections.Database(p.Connection())
}

// file returns the file of the pipeline.
func (this *Pipeline) file() *connections.File {
	return this.core.connections.File(this.pipeline)
}

// fromState serializes pipeline into this.
func (this *Pipeline) fromState(core *Core, store *datastore.Store, pipeline *state.Pipeline) {
	c := pipeline.Connection()
	connector := c.Connector()
	this.core = core
	this.pipeline = pipeline
	this.connection = &Connection{core: core, store: store, connection: c}
	this.ID = pipeline.ID
	this.Connector = connector.Code
	this.ConnectorType = ConnectorType(connector.Type)
	this.Connection = c.ID
	this.ConnectionRole = Role(c.Role)
	this.Target = Target(pipeline.Target)
	this.Name = pipeline.Name
	this.Enabled = pipeline.Enabled
	if pipeline.EventType != "" {
		this.EventType = new(pipeline.EventType)
	}
	_, this.Running = this.pipeline.Run()
	if pipeline.Target == state.TargetUser || pipeline.Target == state.TargetGroup {
		if pipeline.SchedulePeriod != 0 {
			this.ScheduleStart = new(int(pipeline.ScheduleStart))
			this.SchedulePeriod = new(SchedulePeriod(pipeline.SchedulePeriod))
		}
	}
	this.InSchema = pipeline.InSchema
	this.OutSchema = pipeline.OutSchema
	if pipeline.Filter != nil {
		this.Filter = convertWhereToFilter(pipeline.Filter, pipeline.InSchema)
	}
	this.RequiredConsents = RequiredConsents{
		Operator: ConsentPurposesOperator(pipeline.RequiredConsents.Operator),
		Purposes: pipeline.RequiredConsents.Purposes,
	}
	if pipeline.Transformation.Mapping != nil {
		this.Transformation = &Transformation{
			Mapping: maps.Clone(pipeline.Transformation.Mapping),
		}
	}
	if function := pipeline.Transformation.Function; function != nil {
		this.Transformation = &Transformation{
			Function: &TransformationFunction{
				Language:     Language(function.Language.String()),
				Source:       function.Source,
				PreserveJSON: function.PreserveJSON,
				InPaths:      slices.Clone(pipeline.Transformation.InPaths),
				OutPaths:     slices.Clone(pipeline.Transformation.OutPaths),
			},
		}
	}
	if pipeline.Query != "" {
		this.Query = new(pipeline.Query)
	}
	if f := pipeline.Format(); f != nil {
		this.Format = f.Code
	}
	if pipeline.Path != "" {
		this.Path = new(pipeline.Path)
	}
	if pipeline.Sheet != "" {
		this.Sheet = new(pipeline.Sheet)
	}
	this.Compression = Compression(pipeline.Compression)
	if pipeline.TableName != "" {
		this.TableName = new(pipeline.TableName)
	}
	if pipeline.ExportMode != "" {
		this.ExportMode = new(ExportMode(pipeline.ExportMode))
		this.Matching = new(Matching(pipeline.Matching))
		this.UpdateOnDuplicates = new(pipeline.UpdateOnDuplicates)
	}
	if pipeline.TableKey != "" {
		this.TableKey = new(pipeline.TableKey)
	}
	if pipeline.UserIDColumn != "" {
		this.UserIDColumn = new(pipeline.UserIDColumn)
	}
	if pipeline.UpdatedAtColumn != "" {
		this.UpdatedAtColumn = new(pipeline.UpdatedAtColumn)
	}
	if pipeline.UpdatedAtFormat != "" {
		this.UpdatedAtFormat = new(pipeline.UpdatedAtFormat)
	}
	this.Incremental = pipeline.Incremental
	if pipeline.OrderBy != "" {
		this.OrderBy = new(pipeline.OrderBy)
	}
}

// setRunCursor sets the cursor of the pipeline run.
func (this *Pipeline) setRunCursor(ctx context.Context, cursor time.Time) error {
	run, _ := this.pipeline.Run()
	_, err := this.core.db.Exec(ctx, "UPDATE pipelines_runs SET cursor = $1 WHERE id = $2", cursor, run.ID)
	return err
}

// PipelineToSet represents a pipeline to set in a connection, by creating a new
// pipeline (using the method Connection.CreatePipeline) or updating an
// existing one (using the method Pipeline.Update).
//
// Refer to the specifications in the file "core/Pipelines.csv" for more
// details.
type PipelineToSet struct {

	// Name must be a non-empty valid UTF-8 encoded string and cannot be longer
	// than 60 runes.
	Name string `json:"name"`

	// Enabled indicates whether the pipeline is enabled or not.
	Enabled bool `json:"enabled"`

	// Filter is the filter of the pipeline, if it has one, otherwise is nil.
	Filter *Filter `json:"filter"`

	// RequiredConsents is the set of consent purposes that must be present in
	// an event's consent for it to be delivered.
	RequiredConsents RequiredConsents `json:"requiredConsents"`

	// InSchema is the input schema of the pipeline.
	//
	// Please refer to the 'Pipelines.csv' file for a complete list of properties
	// that must be inside this schema, based on the connection and pipeline type.
	InSchema types.Type `json:"inSchema"`

	// OutSchema is the output schema of the pipeline.
	//
	// Please refer to the 'Pipelines.csv' file for a complete list of properties
	// that must be inside this schema, based on the connection and pipeline type.
	OutSchema types.Type `json:"outSchema"`

	// Transformation is the mapping or function transformation, if it has one.
	//
	// Every pipeline that supports transformations may have an associated mapping
	// or function, which are mutually exclusive.
	//
	// Please refer to the 'Pipelines.csv' file for details about this
	// transformation and the properties it eventually operates on, based on the
	// connection and the pipeline type.
	Transformation *Transformation `json:"transformation"`

	// Query is the query of the pipeline, if it has one, otherwise it is the
	// empty string. It cannot be longer than MaxQuerySize runes.
	Query string `json:"query"`

	// Format is the file format and corresponds to the name of a file connector.
	// For non-file pipelines, this must be empty.
	Format string `json:"format"`

	// Path is the path of the file. It cannot be longer than MaxFilePathSize
	// runes, and it is empty for non-file pipelines.
	Path string `json:"path"`

	// Sheet is the sheet name for multiple sheets file pipelines. It must be UTF-8
	// encoded, have a length in the range [1, 31], should not start or end with
	// "'", and cannot contain any of "*", "/", ":", "?", "[", "\", and "]". It is
	// empty for non-file and non-multipart sheets pipelines. Sheet names are
	// case-insensitive.
	Sheet string `json:"sheet"`

	// Compression is the compression of the pipeline on file storage connections.
	// In any other case, must be 0.
	Compression Compression `json:"compression"`

	// OrderBy is the property path for which to order profiles when they are
	// exported to a file, and must therefore refer to a property of the
	// pipeline's output schema (OutSchema). It cannot be longer than 1024 runes.
	// For pipelines that do not export profiles to file, this is the empty string.
	OrderBy string `json:"orderBy"`

	// FormatSettings represents the format settings of a file connector.
	// It must be nil if the connector does not have settings.
	FormatSettings json.Value `json:"formatSettings"`

	// Mode specifies, for apps, whether the export should create users or groups,
	// update them, or do both.
	ExportMode ExportMode `json:"exportMode"`

	// Matching defines a relationship between a property in Krenalis ("in") and
	// a corresponding property in the application ("out") used during an
	// export.
	Matching Matching `json:"matching"`

	// UpdateOnDuplicates indicates whether to proceed with the export even if
	// duplicate users or groups are found in the application.
	UpdateOnDuplicates bool `json:"updateOnDuplicates"`

	// TableName is the name of the table for the export and it is defined for
	// destination database-pipelines; in any other case, it is the empty string.
	// It cannot be longer than MaxTableNameSize runes.
	TableName string `json:"tableName"`

	// TableKey is the name of the property used as table key when exporting
	// profiles to databases.
	// It is the empty string for any other type of pipeline.
	TableKey string `json:"tableKey"`

	// UserIDColumn is the property name used as user ID when importing from a
	// file or from a database.
	// It cannot be longer than 1024 runes.
	UserIDColumn string `json:"userIDColumn"`

	// UpdatedAtColumn is the update time column when importing from a file or from
	// a database. May be empty to indicate that no properties should be used for
	// reading the update time. Also refer to the documentation of UpdatedAtFormat,
	// which is strictly related to this.
	// It cannot be longer than 1024 runes.
	UpdatedAtColumn string `json:"updatedAtColumn"`

	// UpdatedAtFormat indicates the update time value format for parsing the value
	// read from the update time column.
	//
	// Represents a format when a UpdatedAtColumn is provided and its corresponding
	// property kind is json or string, otherwise it is the empty string.
	//
	// In case it is provided, accepted values are:
	//
	//   - "ISO8601": the ISO 8601 format
	//   - "Excel": the Excel format, a float value stored in an Excel cell
	//     representing a date/datetime
	//   - a string containing a '%' character: the strftime() function format
	//
	// "Excel" format is only allowed for file pipelines.
	//
	// It cannot be longer than MaxUpdatedAtFormatSize runes.
	UpdatedAtFormat string `json:"updatedAtFormat"`

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
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return errors.BadRequest(`schedule period can be "5m", "15m", "30m", "1h", "2h", "3h", "6h", "8h", "12h", or "24h"`)
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
		return errors.BadRequest(`schedule period can be "5m", "15m", "30m", "1h", "2h", "3h", "6h", "8h", "12h", or "24h"`)
	}
	*period = p
	return nil
}

// isDispatchingEventsToApplications reports whether a connector of the given
// type, on a connection with the given role, and a pipeline with the given
// target, is dispatching events to applications.
func isDispatchingEventsToApplications(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	return role == state.Destination && target == state.TargetEvent && connectorType == state.Application
}

// isExportUsersToFile reports whether a connector of the given type, on a
// connection with the given role is exporting users into a file.
func isExportUsersToFile(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	return connectorType == state.FileStorage && role == state.Destination && target == state.TargetUser
}

// isImportingEventsIntoWarehouse reports whether a connector of the given type,
// on a connection with the given role, and a pipeline with the given target, is
// importing events into the data warehouse.
func isImportingEventsIntoWarehouse(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	return role == state.Source && target == state.TargetEvent && (connectorType == state.SDK || connectorType == state.Webhook)
}

// isImportingUserIdentitiesFromEvents reports whether a connector of the
// given type, on a connection with the given role, and a pipeline with the
// given target, is importing identities from events.
func isImportingUserIdentitiesFromEvents(connectorType state.ConnectorType, role state.Role, target state.Target) bool {
	return role == state.Source && target == state.TargetUser && (connectorType == state.SDK || connectorType == state.Webhook)
}

// onlyForMatching returns a schema which contains only the properties of schema
// which can be used for the apps export matching.
//
// Returns an invalid schema in case none of the properties of schema can be
// used.
func onlyForMatching(schema types.Type) types.Type {
	properties := schema.Properties()
	return types.Prune(schema, func(path string) bool {
		p, _ := properties.ByPath(path)
		return canBeUsedAsMatchingProp(p.Type.Kind())
	})
}

// shouldReload reports whether the next pipeline run requires reloading, based
// on whether the update notification applies to the pipeline.
func shouldReload(a *state.Pipeline, n *state.UpdatePipeline) bool {
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
	if f := a.Format(); f != nil && f.Code != n.Format {
		return true
	}
	if a.Path != n.Path || a.Sheet != n.Sheet {
		return true
	}
	if !bytes.Equal(a.FormatSettings, n.FormatSettings) {
		return true
	}
	if a.UserIDColumn != n.UserIDColumn {
		return true
	}
	if a.UpdatedAtColumn != n.UpdatedAtColumn {
		return true
	}
	if a.UpdatedAtFormat != n.UpdatedAtFormat {
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

// toStateRequiredConsents converts the required consents to a
// state.RequiredConsents value.
func toStateRequiredConsents(requiredConsents RequiredConsents) state.RequiredConsents {
	return state.RequiredConsents{
		Operator: state.ConsentPurposesOperator(requiredConsents.Operator),
		Purposes: requiredConsents.Purposes,
	}
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
// for a pipeline. If pipeline is 0, the returned name refers to a preview
// transformation function.
func transformationFunctionName(pipeline string) string {
	if pipeline == "" {
		return fmt.Sprintf("krenalis_preview_%s", uuid.NewString())
	}
	now := time.Now().UTC()
	return fmt.Sprintf("krenalis_pipeline_%s_%s-%09d", pipeline, now.Format("2006-01-02T15-04-05"), now.Nanosecond())
}
