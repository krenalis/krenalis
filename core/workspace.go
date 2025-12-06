// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/collector"
	"github.com/meergo/meergo/core/internal/connections"
	"github.com/meergo/meergo/core/internal/datastore"
	"github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/util"
	"github.com/meergo/meergo/tools/errors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"

	"github.com/jxskiss/base62"
)

const (
	maxEventsListenedTo = 1000 // maximum number of processed events listened to.

	// MaxEventListeners is the maximum number of event listeners.
	MaxEventListeners = collector.MaxEventListeners
)

// Workspace represents a workspace.
type Workspace struct {
	core                           *Core
	organization                   *Organization
	store                          *datastore.Store
	workspace                      *state.Workspace
	ID                             int            `json:"id"`
	Name                           string         `json:"name"`
	ProfileSchema                  types.Type     `json:"profileSchema"`
	PrimarySources                 map[string]int `json:"primarySources,format:emitnull"`
	ResolveIdentitiesOnBatchImport bool           `json:"resolveIdentitiesOnBatchImport"`
	Identifiers                    []string       `json:"identifiers,format:emitnull"`
	WarehouseMode                  WarehouseMode  `json:"warehouseMode"`
	UIPreferences                  UIPreferences  `json:"uiPreferences"`
}

type UIPreferences struct {
	Profile struct {
		Image     string `json:"image"`     // property path.
		FirstName string `json:"firstName"` // property path.
		LastName  string `json:"lastName"`  // property path.
		Extra     string `json:"extra"`     // property path.
	} `json:"profile"`
}

// PipelineStep represents a step of a pipeline.
type PipelineStep int

const (
	ReceiveStep          = PipelineStep(metrics.ReceiveStep)
	InputValidationStep  = PipelineStep(metrics.InputValidationStep)
	FilterStep           = PipelineStep(metrics.FilterStep)
	TransformationStep   = PipelineStep(metrics.TransformationStep)
	OutputValidationStep = PipelineStep(metrics.OutputValidationStep)
	FinalizeStep         = PipelineStep(metrics.FinalizeStep)
)

func (step PipelineStep) String() string {
	switch step {
	case ReceiveStep:
		return "Receive"
	case InputValidationStep:
		return "InputValidation"
	case FilterStep:
		return "Filter"
	case TransformationStep:
		return "Transformation"
	case OutputValidationStep:
		return "OutputValidation"
	case FinalizeStep:
		return "Finalize"
	}
	panic("core: invalid PipelineStep")
}

// ParsePipelineStep parses a pipeline step and returns it. If step is not a
// valid returns 0 and an error.
func ParsePipelineStep(step string) (PipelineStep, error) {
	switch step {
	case "Receive":
		return ReceiveStep, nil
	case "InputValidation":
		return InputValidationStep, nil
	case "Filter":
		return FilterStep, nil
	case "Transformation":
		return TransformationStep, nil
	case "OutputValidation":
		return OutputValidationStep, nil
	case "Finalize":
		return FinalizeStep, nil
	}
	return 0, fmt.Errorf("step is not valid")
}

// PipelineError represents a pipeline error.
type PipelineError struct {
	Pipeline     int          `json:"pipeline"`
	Step         PipelineStep `json:"step"`
	Count        int          `json:"count"`
	Message      string       `json:"message"`
	LastOccurred time.Time    `json:"lastOccurred"`
}

// Pipeline returns the pipeline with identifier id of the workspace.
// It returns an errors.NotFound error if the pipeline does not exist.
func (this *Workspace) Pipeline(id int) (*Pipeline, error) {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid pipeline identifier", id)
	}
	p, ok := this.core.state.Pipeline(id)
	if !ok || p.Connection().Workspace().ID != this.workspace.ID {
		return nil, errors.NotFound("pipeline %d does not exist", id)
	}
	var pipeline Pipeline
	pipeline.fromState(this.core, this.store, p)
	return &pipeline, nil
}

// PipelineErrors returns the errors for the provided pipelines within the time
// range [start,end). The end time must not precede the start time, and both
// must be within [metrics.MinTime,metrics.MaxTime]. pipelines must not be empty.
// Returned errors are limited to [first, first+limit), where first >= 0 and
// 0 < limit <= 100.
func (this *Workspace) PipelineErrors(ctx context.Context, start, end time.Time, pipelines []int, step *PipelineStep, first, limit int) ([]PipelineError, error) {

	this.core.mustBeOpen()

	start = start.UTC()
	end = end.UTC()

	// Validate start and end.
	if start.Before(metrics.MinTime) {
		return nil, errors.New("start date is too far in the past")
	}
	if end.After(metrics.MaxTime) {
		return nil, errors.New("end date is too far in the future")
	}
	if end.Before(start) {
		return nil, fmt.Errorf("end date cannot be earlier than start date")
	}

	// Validate pipelines.
	if len(pipelines) == 0 {
		return nil, errors.BadRequest("pipelines cannot be empty")
	}
	for _, pipeline := range pipelines {
		if pipeline < 1 || pipeline > maxInt32 {
			return nil, errors.BadRequest("pipeline %d is not valid", pipeline)
		}
	}

	// Validate step.
	var s *metrics.Step
	if step != nil {
		if *step < ReceiveStep || *step > FinalizeStep {
			return nil, errors.BadRequest("step %d is not valid", *step)
		}
		s = (*metrics.Step)(step)
	}

	// validate first and limit.
	if first < 0 || first > 9999 {
		return nil, errors.BadRequest("first must be in range [0,9999]")
	}
	if limit < 1 || limit > 100 {
		return nil, errors.BadRequest("limit must be in range [1,100]")
	}

	pipelines = filterWorkspacePipelines(this.workspace, pipelines)
	if len(pipelines) == 0 {
		return []PipelineError{}, nil
	}

	metricsErrors, err := this.core.metrics.Errors(ctx, start, end, pipelines, s, first, limit)
	if err != nil {
		return nil, err
	}

	errs := make([]PipelineError, len(metricsErrors))
	for i, e := range metricsErrors {
		errs[i] = PipelineError{
			Pipeline:     e.Pipeline,
			Step:         PipelineStep(e.Step),
			Count:        e.Count,
			Message:      e.Message,
			LastOccurred: e.LastOccurred,
		}
	}

	return errs, nil
}

// PipelineMetrics represents pipeline metrics for a time period.
type PipelineMetrics struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Passed [][6]int  `json:"passed"`
	Failed [][6]int  `json:"failed"`
}

// MetricUnit represents the unit of time used for aggregating metrics.
// It can be:
// - Minute: aggregates metrics by minute
// - Hour: aggregates metrics by hour
// - Day: aggregates metrics by day
type MetricUnit int

const (
	Minute = MetricUnit(metrics.Minute)
	Hour   = MetricUnit(metrics.Hour)
	Day    = MetricUnit(metrics.Day)
)

// Attributes returns the attributes of a profile, given its MPID.
//
// It returns an errors.NotFoundError error, if the profile does not exist.
// It returns an errors.UnprocessableError error with code MaintenanceMode if
// the data warehouse is in maintenance mode.
func (this *Workspace) Attributes(ctx context.Context, mpid string) (json.Value, error) {

	this.core.mustBeOpen()

	ws := this.workspace

	// Validate the MPID.
	if _, ok := types.ParseUUID(mpid); !ok {
		return nil, errors.BadRequest("profile %q is not a valid profile identifier", mpid)
	}

	properties := this.workspace.ProfileSchema.Properties().Names()
	where := &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
		Property: []string{"_mpid"},
		Operator: state.OpIs,
		Values:   []any{mpid},
	}}}

	// Retrieve the profile attributes.
	profiles, _, err := this.store.Profiles(ctx, datastore.Query{
		Properties: properties,
		Where:      where,
		Limit:      1,
	})
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if err, ok := err.(*datastore.UnavailableError); ok {
			return nil, errors.Unavailable("%s", err)
		}
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, errors.NotFound("profile %q does not exist", mpid)
	}

	return types.Marshal(profiles[0], ws.ProfileSchema)
}

// ColumnTypeDescription returns a description for the warehouse column type
// corresponding to the given types.Type.
// The description is not required to be a syntactically valid warehouse type,
// and may therefore include additional human-readable details (such as type
// information, maximum character count, enum values, etc...).
func (this Workspace) ColumnTypeDescription(t types.Type) (string, error) {
	this.core.mustBeOpen()
	return this.store.ColumnTypeDescription(t)
}

// PipelineMetricsPerDate returns metrics aggregated by day for the time interval
// between the specified start and end dates. The dates must be no earlier than
// 1970-01-01 and no later than 2262-04-10. The day of the start date must be at
// least one day before the day of the end date. pipelines specifies the
// pipelines for which metrics are returned and must not be empty.
func (this *Workspace) PipelineMetricsPerDate(ctx context.Context, start, end time.Time, pipelines []int) (PipelineMetrics, error) {

	this.core.mustBeOpen()

	start = start.UTC().Truncate(24 * time.Hour)
	end = end.UTC().Truncate(24 * time.Hour)

	// Validate start and end.
	if start.Before(metrics.MinTime) {
		return PipelineMetrics{}, errors.BadRequest("start date is too far in the past")
	}
	if end.After(metrics.MaxTime) {
		return PipelineMetrics{}, errors.BadRequest("end date is too far in the future")
	}
	if !end.After(start) {
		return PipelineMetrics{}, errors.BadRequest("day of the end date must be after the day of the start date")
	}

	// Validate pipelines.
	if len(pipelines) == 0 {
		return PipelineMetrics{}, errors.BadRequest("pipelines if non-nil, cannot be empty")
	}
	for _, pipeline := range pipelines {
		if pipeline < 1 || pipeline > maxInt32 {
			return PipelineMetrics{}, errors.BadRequest("pipeline %d is not valid", pipeline)
		}
	}

	pipelines = filterWorkspacePipelines(this.workspace, pipelines)
	if len(pipelines) == 0 {
		number := int(end.Sub(start).Hours() / 24)
		return PipelineMetrics{
			Start:  start,
			End:    end,
			Passed: make([][6]int, number),
			Failed: make([][6]int, number),
		}, nil
	}

	metrics, err := this.core.metrics.MetricsPerDate(ctx, start, end, pipelines)
	if err != nil {
		return PipelineMetrics{}, err
	}

	return PipelineMetrics{
		Start:  metrics.Start,
		End:    metrics.End,
		Passed: metrics.Passed,
		Failed: metrics.Failed,
	}, nil
}

// PipelineMetricsPerTimeUnit returns metrics for the specified number of minutes,
// hours, or days based on the unit, which can be Minute, Hour, or Day, up to
// the current time. number must be in the following ranges: [1,60] for minutes,
// [1,48] for hours, and [1,30] for days. pipelines specifies the pipelines for
// which metrics are returned and must not be empty.
func (this *Workspace) PipelineMetricsPerTimeUnit(ctx context.Context, number int, unit MetricUnit, pipelines []int) (PipelineMetrics, error) {

	this.core.mustBeOpen()

	// Validate number and unit.
	switch unit {
	case Minute:
		if number < 1 || number > 60 {
			return PipelineMetrics{}, errors.BadRequest("minutes must be in range [1,60]")
		}
	case Hour:
		if number < 1 || number > 48 {
			return PipelineMetrics{}, errors.BadRequest("hours must be in range [1,48]")
		}
	case Day:
		if number < 1 || number > 30 {
			return PipelineMetrics{}, errors.BadRequest("days must be in range [1,30]")
		}
	}

	// Validate pipelines.
	if len(pipelines) == 0 {
		return PipelineMetrics{}, errors.BadRequest("pipelines if non-nil, cannot be empty")
	}
	for _, pipeline := range pipelines {
		if pipeline < 1 || pipeline > maxInt32 {
			return PipelineMetrics{}, errors.BadRequest("pipeline %d is not valid", pipeline)
		}
	}

	pipelines = filterWorkspacePipelines(this.workspace, pipelines)
	if len(pipelines) == 0 {
		return PipelineMetrics{
			Passed: make([][6]int, number),
			Failed: make([][6]int, number),
		}, nil
	}

	metrics, err := this.core.metrics.MetricsPerTimeUnit(ctx, number, time.Duration(unit), pipelines)
	if err != nil {
		return PipelineMetrics{}, err
	}

	return PipelineMetrics{
		Start:  metrics.Start,
		End:    metrics.End,
		Passed: metrics.Passed,
		Failed: metrics.Failed,
	}, nil
}

// authorizedOAuthAccount represents an authorized OAuth account that can be
// used to create a new connection.
type authorizedOAuthAccount struct {
	Workspace    int
	Connector    string
	Code         string
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Time
}

// AuthToken returns an authorization token, given an authorization code and
// the redirection URI used to obtain that code, that can be used to add a new
// connection to the workspace for the specified connector.
//
// It returns an errors.NotFound error if the workspace does not exist anymore.
// It returns an errors.UnprocessableError error with code ConnectorNotExist if
// the connector does not exist.
func (this *Workspace) AuthToken(ctx context.Context, connector, redirectionURI, code string) (string, error) {

	this.core.mustBeOpen()

	if connector == "" {
		return "", errors.BadRequest("connector code is empty")
	}
	if code == "" {
		return "", errors.BadRequest("authorization code is empty")
	}

	c, ok := this.core.state.Connector(connector)
	if !ok {
		return "", errors.Unprocessable(ConnectorNotExist, "connector %q does not exist", connector)
	}
	if c.OAuth == nil {
		return "", errors.BadRequest("connector %s does not support authorization", connector)
	}

	auth, err := this.core.connections.GrantAuthorization(ctx, c, code, redirectionURI)
	if err != nil {
		return "", err
	}

	account, err := json.Marshal(authorizedOAuthAccount{
		Workspace:    this.workspace.ID,
		Connector:    connector,
		Code:         auth.AccountCode,
		AccessToken:  auth.AccessToken,
		RefreshToken: auth.RefreshToken,
		ExpiresIn:    auth.ExpiresIn,
	})
	if err != nil {
		return "", err
	}

	// TODO(marco): Encrypt the token.

	return base62.EncodeToString(account), nil
}

// Connection returns the connection with identifier id of the workspace.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
func (this *Workspace) Connection(ctx context.Context, id int) (*Connection, error) {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("connection identifier %d is not valid", id)
	}
	c, ok := this.workspace.Connection(id)
	if !ok {
		return nil, errors.NotFound("connection %d does not exist", id)
	}
	conn := c.Connector()

	connection := Connection{
		core:              this.core,
		store:             this.store,
		connection:        c,
		ID:                c.ID,
		Name:              c.Name,
		Connector:         conn.Code,
		ConnectorType:     ConnectorType(conn.Type),
		Role:              Role(c.Role),
		Strategy:          (*Strategy)(c.Strategy),
		SendingMode:       (*SendingMode)(c.SendingMode),
		LinkedConnections: slices.Clone(c.LinkedConnections),
		Health:            Health(c.Health),
	}

	// Set the pipelines.
	pipelines := c.Pipelines()
	connection.Pipelines = make([]Pipeline, len(pipelines))
	for i, pipeline := range pipelines {
		connection.Pipelines[i].fromState(this.core, this.store, pipeline)
	}

	// Set the event types.
	if conn.Type == state.API && c.Role == state.Destination &&
		c.Connector().DestinationTargets.Contains(state.TargetEvent) {
		appEventTypes, err := connection.api().EventTypes(ctx)
		if err != nil {
			return nil, err
		}
		eventTypes := make([]EventType, len(appEventTypes))
		for i, et := range appEventTypes {
			eventTypes[i] = EventType{
				ID:          et.ID,
				Name:        et.Name,
				Description: et.Description,
				Filter:      et.Filter,
			}
		}
		connection.EventTypes = &eventTypes
	}

	return &connection, nil
}

// Connections returns the connections of the workspace.
func (this *Workspace) Connections() []*Connection {
	this.core.mustBeOpen()
	connections := this.workspace.Connections()
	infos := make([]*Connection, len(connections))
	for i, c := range connections {
		conn := c.Connector()
		connection := Connection{
			core:              this.core,
			store:             this.store,
			connection:        c,
			ID:                c.ID,
			Name:              c.Name,
			Connector:         conn.Code,
			ConnectorType:     ConnectorType(conn.Type),
			Role:              Role(c.Role),
			Strategy:          (*Strategy)(c.Strategy),
			SendingMode:       (*SendingMode)(c.SendingMode),
			LinkedConnections: slices.Clone(c.LinkedConnections),
			Health:            Health(c.Health),
		}

		// Set the pipelines.
		pipelines := c.Pipelines()
		connection.Pipelines = make([]Pipeline, len(pipelines))
		for i, pipeline := range pipelines {
			connection.Pipelines[i].fromState(this.core, this.store, pipeline)
		}

		infos[i] = &connection
	}
	sort.Slice(infos, func(i, j int) bool {
		a, b := infos[i], infos[j]
		return a.Name < b.Name || a.Name == b.Name && a.ID == b.ID
	})
	return infos
}

// CreateConnection creates a new connection. authToken is an authorization
// token returned by the AuthToken method and must be empty if the connector
// does not support authorization.
//
// Message broker connectors are not currently supported.
//
// It returns an errors.UnprocessableError error with code
//
//   - ConnectorNotExist, if the connector does not exist.
//   - LinkedConnectionNotExist, if a linked connection does not exist.
//   - InvalidSettings, if the settings are not valid.
func (this *Workspace) CreateConnection(ctx context.Context, connection ConnectionToAdd, authToken string) (int, error) {

	this.core.mustBeOpen()

	if connection.Role != Source && connection.Role != Destination {
		return 0, errors.BadRequest("role %d is not valid", int(connection.Role))
	}
	if connection.Connector == "" {
		return 0, errors.BadRequest("connector code is empty")
	}
	if err := util.ValidateStringField("name", connection.Name, 100); err != nil {
		return 0, errors.BadRequest("%s", err)
	}
	if s := connection.Strategy; s != nil {
		if !isValidStrategy(*s) {
			return 0, errors.BadRequest("strategy %q is not valid", *s)
		}
		if connection.Role == Destination {
			return 0, errors.BadRequest("destination connections cannot have a strategy")
		}
	}
	if sm := connection.SendingMode; sm != nil && !isValidSendingMode(*sm) {
		return 0, errors.BadRequest("sending mode %q is not valid", *sm)
	}

	c, ok := this.core.state.Connector(connection.Connector)
	if !ok {
		return 0, errors.Unprocessable(ConnectorNotExist, "connector %q does not exist", connection.Connector)
	}
	switch c.Type {
	case state.File:
		return 0, errors.BadRequest("connections cannot have type file")
	case state.MessageBroker:
		return 0, errors.BadRequest("message broker connectors are not currently supported")
	case state.SDK:
		if connection.Role == Destination {
			return 0, errors.BadRequest("SDK connections cannot be destinations")
		}
	case state.Webhook:
		if connection.Role == Destination {
			return 0, errors.BadRequest("webhook connections cannot be destinations")
		}
	}

	// Validate and normalize linked connections.
	var err error
	connection.LinkedConnections, err = validateLinkedConnections(connection.LinkedConnections, c, this.workspace, state.Role(connection.Role))
	if err != nil {
		return 0, err
	}

	n := state.CreateConnection{
		Workspace:         this.workspace.ID,
		Name:              connection.Name,
		Connector:         connection.Connector,
		Role:              state.Role(connection.Role),
		Strategy:          (*state.Strategy)(connection.Strategy),
		SendingMode:       (*state.SendingMode)(connection.SendingMode),
		LinkedConnections: connection.LinkedConnections,
	}
	if n.Name == "" {
		n.Name = c.Label
	}

	// Validate the strategy.
	if connection.Role == Source {
		if c.Strategies {
			if connection.Strategy == nil {
				return 0, errors.BadRequest("%s connections must have a strategy", strings.ToLower(c.Type.String()))
			}
		} else {
			if connection.Strategy != nil {
				return 0, errors.BadRequest("%s connections cannot have a strategy", strings.ToLower(c.Type.String()))
			}
		}
	}

	// Validate the sending mode.
	if connection.Role == Destination {
		if c.SendingMode != nil {
			if connection.SendingMode == nil {
				return 0, errors.BadRequest("connector %s requires a sending mode", c.Code)
			}
			if !c.SendingMode.Contains(state.SendingMode(*connection.SendingMode)) {
				return 0, errors.BadRequest("connector %s does not support sending mode %s", c.Code, *c.SendingMode)
			}
		} else if connection.SendingMode != nil {
			return 0, errors.BadRequest("connector %s does not support sending modes", c.Code)
		}
	} else if connection.SendingMode != nil {
		return 0, errors.BadRequest("source connections cannot have a sending mode")
	}

	// Validate the authorization token.
	if (authToken == "") != (c.OAuth == nil) {
		if authToken == "" {
			return 0, errors.BadRequest("authorization token is required by connector %s", n.Connector)
		}
		return 0, errors.BadRequest("connector %s does not support authorization", n.Connector)
	}

	// Set the OAuth account. It can be an existing account or an account that needs to be created.
	if authToken != "" {
		data, err := base62.DecodeString(authToken)
		if err != nil {
			return 0, errors.BadRequest("authorization token is not valid")
		}
		var account authorizedOAuthAccount
		err = json.Unmarshal(data, &account)
		if err != nil {
			return 0, errors.BadRequest("authorization token is not valid")
		}
		if account.Workspace != this.workspace.ID || account.Connector != c.Code {
			return 0, errors.BadRequest("authorization token is not valid")
		}
		n.Account.Code = account.Code
		a, ok := this.workspace.AccountByCode(account.Connector, account.Code)
		if ok {
			n.Account.ID = a.ID
		}
		if !ok || account.AccessToken != a.AccessToken || account.RefreshToken != a.RefreshToken ||
			account.ExpiresIn != a.ExpiresIn {
			n.Account.AccessToken = account.AccessToken
			n.Account.RefreshToken = account.RefreshToken
			n.Account.ExpiresIn = account.ExpiresIn
		}
	}

	// Validate the settings.
	if settings := connection.Settings; settings == nil {
		if connection.Role == Source && c.HasSourceSettings || connection.Role == Destination && c.HasDestinationSettings {
			return 0, errors.BadRequest("settings must be provided because connector %s has %s settings", c.Code, strings.ToLower(connection.Role.String()))
		}
	} else {
		if connection.Role == Source && !c.HasSourceSettings || connection.Role == Destination && !c.HasDestinationSettings {
			return 0, errors.BadRequest("settings cannot be provided because connector %s has no %s settings", c.Code, strings.ToLower(connection.Role.String()))
		}
		var clientSecret string
		if c.OAuth != nil {
			clientSecret = c.OAuth.ClientSecret
		}
		conf := &connections.ConnectorConfig{
			Role: n.Role,
		}
		conf.OAuth.Account = n.Account.Code
		conf.OAuth.ClientSecret = clientSecret
		conf.OAuth.AccessToken = n.Account.AccessToken
		n.Settings, err = this.core.connections.UpdatedSettings(ctx, c, conf, settings)
		if err != nil {
			switch err.(type) {
			case *connectors.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
			case *connections.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
			return 0, err
		}
	}

	// Generate the identifier.
	n.ID, err = generateRandomID()
	if err != nil {
		return 0, err
	}

	// Generate an event write key.
	if c.Type == state.SDK || c.Type == state.Webhook {
		n.EventWriteKey, err = generateEventWriteKey()
		if err != nil {
			return 0, err
		}
	}

	// Build the query to link connections.
	var add string
	if n.LinkedConnections != nil {
		var b strings.Builder
		for i, id := range n.LinkedConnections {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Itoa(id))
		}
		add = "UPDATE connections\n" +
			"SET linked_connections = (SELECT ARRAY(SELECT unnest(array_append(linked_connections, $1)) ORDER BY 1))\n" +
			"WHERE id IN (" + b.String() + ")"
	}

	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		if n.Account.Code != "" {
			if n.Account.ID == 0 {
				// Insert a new account.
				err = tx.QueryRow(ctx, "INSERT INTO accounts (workspace, connector, code, access_token,"+
					" refresh_token, expires_in) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
					n.Workspace, n.Connector, n.Account.Code, n.Account.AccessToken, n.Account.RefreshToken, n.Account.ExpiresIn).
					Scan(&n.Account.ID)
			} else if n.Account.AccessToken != "" {
				// Update the current account.
				_, err = tx.Exec(ctx, "UPDATE accounts "+
					"SET access_token = $1, refresh_token = $2, expires_in = $3 WHERE id = $4",
					n.Account.AccessToken, n.Account.RefreshToken, n.Account.ExpiresIn, n.Account.ID)
			}
			if err != nil {
				if db.IsForeignKeyViolation(err) && db.ErrConstraintName(err) == "accounts_workspace_fkey" {
					err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
				}
				return nil, err
			}
		}
		// Insert the connection.
		_, err = tx.Exec(ctx, "INSERT INTO connections "+
			"(id, workspace, name, connector, role, account,"+
			" strategy, sending_mode, linked_connections, settings)"+
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
			n.ID, n.Workspace, n.Name, n.Connector, n.Role, n.Account.ID, n.Strategy,
			n.SendingMode, n.LinkedConnections, string(n.Settings))
		if err != nil {
			if db.IsForeignKeyViolation(err) && db.ErrConstraintName(err) == "connections_workspace_fkey" {
				err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
			}
			return nil, err
		}
		// Link connections.
		if len(n.LinkedConnections) > 0 {
			result, err := tx.Exec(ctx, add, n.ID)
			if err != nil {
				return nil, err
			}
			if int(result.RowsAffected()) < len(n.LinkedConnections) {
				return nil, errors.Unprocessable(LinkedConnectionNotExist, "a linked connection does not exist")
			}
		}
		if n.EventWriteKey != "" {
			// Insert the event write key.
			_, err = tx.Exec(ctx, "INSERT INTO event_write_keys (connection, key, created_at) VALUES ($1, $2, $3)",
				n.ID, n.EventWriteKey, time.Now().UTC())
			if err != nil {
				return nil, err
			}
		}
		return n, nil
	})
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// CreateEventListener creates an event listener for the workspace that listens
// to events and returns its identifier.
//
// If connection is not 0, it must be the identifier of a connection on events.
//   - If it is a source, only events received from that source will be
//     returned.
//   - If it is a destination, only events received from the linked connections
//     present at the time of the call will be returned.
//
// size specifies the maximum number of observed events to be returned by a
// subsequent call to the ListenedEvents method and must be in the range
// [1, 1000].
//
// If filter is non-nil, only events that satisfy the filter will be observed.
//
// It returns an errors.UnprocessableError with code TooManyListeners, if there
// are already too many listeners.
func (this *Workspace) CreateEventListener(connection, size int, filter *Filter) (string, error) {
	this.core.mustBeOpen()
	if connection < 0 || connection > maxInt32 {
		return "", errors.BadRequest("identifier %d is not a valid connection identifier", connection)
	}
	if size < 1 || size > maxEventsListenedTo {
		return "", errors.BadRequest("size %d is not valid", size)
	}
	var connections []int
	if connection > 0 {
		c, ok := this.workspace.Connection(connection)
		if !ok {
			return "", errors.Unprocessable(ConnectionNotExist, "connection %d does not exist", connection)
		}
		if c.LinkedConnections == nil {
			return "", errors.BadRequest("connection %d does not support events", c.ID)
		}
		switch c.Role {
		case state.Source:
			connections = []int{c.ID}
		case state.Destination:
			connections = slices.Clone(c.LinkedConnections)
		}
	}
	var where *state.Where
	if filter != nil {
		_, err := validateFilter(filter, schemas.Event, state.Destination, state.TargetEvent)
		if err != nil {
			return "", errors.BadRequest("filter is not valid: %w", err)
		}
		where = convertFilterToWhere(filter, schemas.Event)
	}
	observer, ok := this.core.collector.Observer(this.workspace.ID)
	if !ok {
		return "", errors.New("observer either has not been created yet or has already been removed")
	}
	id, err := observer.CreateListener(connections, size, where)
	if err != nil {
		if err == collector.ErrTooManyListeners {
			err = errors.Unprocessable(TooManyListeners, "there are already %d listeners", MaxEventListeners)
		}
		return "", err
	}
	return id, nil
}

// Delete deletes the workspace with all its connections.
//
// If the workspace does not exist anymore, it returns an errors.NotFound error.
func (this *Workspace) Delete(ctx context.Context) error {
	this.core.mustBeOpen()
	n := state.DeleteWorkspace{
		ID: this.workspace.ID,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		// Mark the pipeline functions as discontinued.
		now := time.Now().UTC()
		_, err := tx.Exec(ctx, "INSERT INTO discontinued_functions (id, discontinued_at)\n"+
			"SELECT p.transformation_id, $1\n"+
			"FROM pipelines AS p\n"+
			"INNER JOIN connections AS c ON p.connection = c.id\n"+
			"WHERE p.transformation_id != '' AND c.workspace = $2\n"+
			"ON CONFLICT (id) DO NOTHING", now, n.ID)
		if err != nil {
			return nil, err
		}
		// Delete the workspace.
		result, err := tx.Exec(ctx, "DELETE FROM workspaces WHERE id = $1", n.ID)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("workspace %d does not exist", n.ID)
		}
		return n, nil
	})
	return err
}

// DeleteEventListener deletes the given event listener of the workspace. It
// does nothing if the listener does not exist.
func (this *Workspace) DeleteEventListener(listener string) {
	this.core.mustBeOpen()
	observer, ok := this.core.collector.Observer(this.workspace.ID)
	if ok {
		observer.DeleteListener(listener)
	}
}

// Events returns events that match the provided filter, if not nil, and are
// within the range [first,first+limit], where first >= 0, 0 < limit <= 1000,
// and only includes the specified properties from the event schema. properties
// must contain at least one property.
//
// order specifies the property by which to sort the events. It cannot be of
// type json or object. If not provided, the events are sorted by messageId.
// orderDesc controls whether the events should be sorted in descending order,
// when true, or ascending order.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore. It returns an errors.UnprocessableError error with code
// MaintenanceMode if the data warehouse is in maintenance mode.
func (this *Workspace) Events(ctx context.Context, properties []string, filter *Filter, order string, orderDesc bool, first, limit int) ([]map[string]any, error) {

	this.core.mustBeOpen()

	eventProperties := schemas.Event.Properties()

	// Validate the properties.
	if len(properties) == 0 {
		return nil, errors.BadRequest("properties is empty")
	}
	for _, name := range properties {
		if _, ok := eventProperties.ByName(name); !ok {
			if name == "" {
				return nil, errors.BadRequest("a property name is empty")
			}
			if !types.IsValidPropertyName(name) {
				return nil, errors.BadRequest("property name %q is not valid", name)
			}
			return nil, errors.BadRequest("property %q does not exist", name)
		}
	}

	// Validate the filter.
	var where *state.Where
	if filter != nil {
		_, err := validateFilter(filter, schemas.Event, state.Destination, state.TargetEvent)
		if err != nil {
			if err, ok := err.(types.PathNotExistError); ok {
				return nil, errors.BadRequest("filter's property %q does not exist", err.Path)
			}
			return nil, errors.BadRequest("filter is not valid: %w", err)
		}
		where = convertFilterToWhere(filter, schemas.Event)
	}

	// Validate the order.
	if order != "" {
		orderProperty, ok := eventProperties.ByName(order)
		if !ok {
			if !types.IsValidPropertyName(order) {
				return nil, errors.BadRequest("order %q is not a valid property name", order)
			}
			return nil, errors.BadRequest("order property %q does not exist", order)
		}
		switch orderProperty.Type.Kind() {
		case types.JSONKind, types.ObjectKind:
			return nil, errors.BadRequest("cannot sort by %s: property has type %s", order, orderProperty.Type)
		}
	} else {
		order = "messageId"
	}

	// Validate first and limit.
	if first < 0 || first > maxInt32 {
		return nil, errors.BadRequest("first %d in not valid", first)
	}
	if limit < 1 || limit > 1000 {
		return nil, errors.BadRequest("limit %d is not valid", limit)
	}

	// Read the events.
	evts, err := this.store.Events(ctx, datastore.Query{
		Properties: properties,
		Where:      where,
		OrderBy:    order,
		OrderDesc:  orderDesc,
		First:      first,
		Limit:      limit,
	})
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if err, ok := err.(*datastore.UnavailableError); ok {
			return nil, errors.Unavailable("%s", err)
		}
		return nil, err
	}

	return evts, nil
}

// Execution returns the execution with the specified identifier for a pipeline
// in the workspace.
//
// If the execution does not exist, it returns an errors.NotFound error.
func (this *Workspace) Execution(ctx context.Context, id int) (*Execution, error) {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid execution identifier", id)
	}
	// Check if the execution is running.
	if exe, ok := this.workspace.Execution(id); ok {
		return &Execution{
			ID:        exe.ID,
			Pipeline:  exe.Pipeline().ID,
			StartTime: exe.StartTime,
		}, nil
	}
	var exe Execution
	err := this.core.db.QueryRow(ctx,
		"SELECT e.id, e.pipeline, e.start_time, e.end_time, e.passed_0, e.passed_1, e.passed_2, e.passed_3,"+
			" e.passed_4, e.passed_5, e.failed_0, e.failed_1, e.failed_2, e.failed_3, e.failed_4,"+
			" e.failed_5, e.error\n"+
			"FROM pipelines_executions e\n"+
			"INNER JOIN pipelines a ON a.id = e.pipeline\n"+
			"INNER JOIN connections c ON c.id = a.connection\n"+
			"WHERE c.workspace = $1 AND e.id = $2", this.workspace.ID, id).Scan(
		&exe.ID, &exe.Pipeline, &exe.StartTime, &exe.EndTime, &exe.Passed[0], &exe.Passed[1], &exe.Passed[2], &exe.Passed[3],
		&exe.Passed[4], &exe.Passed[5], &exe.Failed[0], &exe.Failed[1], &exe.Failed[2], &exe.Failed[3], &exe.Failed[4],
		&exe.Failed[5], &exe.Error)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("pipeline execution %d does not exist", id)
		}
		return nil, err
	}
	if exe.EndTime == nil {
		exe.Passed = [6]int{}
		exe.Failed = [6]int{}
	}
	return &exe, nil
}

// Executions returns the executions of the pipelines of the workspace.
func (this *Workspace) Executions(ctx context.Context) ([]*Execution, error) {

	this.core.mustBeOpen()

	executions := []*Execution{}
	err := this.core.db.QueryScan(ctx,
		"SELECT e.id, e.pipeline, e.start_time, e.end_time, e.passed_0, e.passed_1, e.passed_2, e.passed_3,"+
			" e.passed_4, e.passed_5, e.failed_0, e.failed_1, e.failed_2, e.failed_3, e.failed_4, e.failed_5, e.error\n"+
			"FROM pipelines_executions e\n"+
			"INNER JOIN pipelines a ON a.id = e.pipeline\n"+
			"INNER JOIN connections c ON c.id = a.connection\n"+
			"WHERE c.workspace = $1\n"+
			"ORDER BY id DESC", this.workspace.ID, func(rows *db.Rows) error {
			var err error
			for rows.Next() {
				var exe Execution
				if err = rows.Scan(&exe.ID, &exe.Pipeline, &exe.StartTime, &exe.EndTime, &exe.Passed[0], &exe.Passed[1], &exe.Passed[2], &exe.Passed[3],
					&exe.Passed[4], &exe.Passed[5], &exe.Failed[0], &exe.Failed[1], &exe.Failed[2], &exe.Failed[3], &exe.Failed[4],
					&exe.Failed[5], &exe.Error); err != nil {
					return err
				}
				executions = append(executions, &exe)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	for _, exe := range executions {
		if exe.EndTime == nil {
			exe.Passed = [6]int{}
			exe.Failed = [6]int{}
		}
	}

	return executions, nil
}

// Identities returns the identities for the provided MPID, and an estimate of
// their total number without applying first and limit.
//
// It returns the identities in range [first,first+limit] with first >= 0
// and 0 < limit <= 1000.
//
// Identities are sorted by updated-at time in descending order, so the most
// recently updated identities come first.
//
// If the MPID does not exist, still return an empty slice instead of an error.
//
// It returns an errors.UnprocessableError error with code MaintenanceMode if
// the data warehouse is in maintenance mode.
func (this *Workspace) Identities(ctx context.Context, mpid string, first, limit int) ([]Identity, int, error) {
	this.core.mustBeOpen()
	if _, ok := types.ParseUUID(mpid); !ok {
		return nil, 0, errors.BadRequest("profile %q is not a valid MPID", mpid)
	}
	if first < 0 {
		return nil, 0, errors.BadRequest("first %d is not valid", first)
	}
	if limit < 1 || limit > 1000 {
		return nil, 0, errors.BadRequest("limit %d is not valid", limit)
	}
	where := &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
		Property: []string{"_mpid"},
		Operator: state.OpIs,
		Values:   []any{mpid},
	}}}
	ws := &Workspace{
		core:      this.core,
		store:     this.store,
		workspace: this.workspace,
	}
	identities, total, err := ws.identities(ctx, where, first, limit)
	if err != nil {
		return nil, 0, err
	}
	if identities == nil {
		identities = []Identity{}
	}
	return identities, total, nil
}

// LatestIdentityResolution return information about the latest identity
// resolution.
//
// In particular:
//
//   - if the Identity Resolution has been started and completed, returns its
//     start time and end time;
//   - if it is in progress, returns its start time and nil for the end time;
//   - if no Identity Resolution has never been executed, returns nil and nil.
//
// It returns an errors.NotFoundError error if the workspace does not exist
// anymore.
func (this *Workspace) LatestIdentityResolution() (startTime, endTime *time.Time, err error) {
	this.core.mustBeOpen()
	ws, ok := this.core.state.Workspace(this.workspace.ID)
	if !ok {
		return nil, nil, errors.NotFound("workspace %d does not exist", this.workspace.ID)
	}
	return ws.IR.StartTime, ws.IR.EndTime, nil
}

// LatestAlterProfileSchema return information about the latest altering of the
// profile schema.
//
// In particular:
//
//   - startTime is the start timestamp (UTC) of the latest altering of the
//     profile schema, either running or completed; if null, no profile schema
//     update has never been started for the workspace.
//   - endTime is the end timestamp (UTC) for the latest altering of the profile
//     schema; if null, it means that the profile schema altering is still in
//     progress, or that no schema altering has never been performed for the
//     workspace.
//   - updateErr is a possible error in the execution of the latest altering
//     of the profile schema; if null, it means that no altering of the profile
//     schema has never been executed, or that one is in progress, or that the
//     last one executed completed without errors.
//
// It returns an errors.NotFoundError error if the workspace does not exist
// anymore.
func (this *Workspace) LatestAlterProfileSchema() (startTime, endTime *time.Time, alterError string, err error) {
	this.core.mustBeOpen()
	ws, ok := this.core.state.Workspace(this.workspace.ID)
	if !ok {
		return nil, nil, "", errors.NotFound("workspace %d does not exist", this.workspace.ID)
	}
	if ws.AlterProfileSchema.Err != nil {
		alterError = *ws.AlterProfileSchema.Err
	}
	return ws.AlterProfileSchema.StartTime, ws.AlterProfileSchema.EndTime, alterError, nil
}

// ListenedEvents returns the events listened to the specified listener and the
// number of omitted events. If the listener does not exist, it returns an
// errors.NotFoundError.
func (this *Workspace) ListenedEvents(listener string) ([]json.Value, int, error) {
	this.core.mustBeOpen()
	observer, ok := this.core.collector.Observer(this.workspace.ID)
	if !ok {
		return nil, 0, errors.NotFound("workspace %d does not exist", this.workspace.ID)
	}
	observedEvents, omitted, err := observer.Events(listener)
	if err != nil {
		if err == collector.ErrEventListenerNotFound {
			return nil, 0, errors.NotFound("event listener %q does not exist", listener)
		}
		return nil, 0, err
	}
	for i, event := range observedEvents {
		observedEvents[i] = slices.Clone(event)
	}
	return observedEvents, omitted, nil
}

// ProfilePropertiesSuitableAsIdentifiers returns the properties of the profile
// schema that can be used as identifiers in the Identity Resolution.
// If none of the properties can be an identifier, this method returns the
// invalid schema.
func (this *Workspace) ProfilePropertiesSuitableAsIdentifiers() types.Type {
	this.core.mustBeOpen()
	return types.Filter(this.workspace.ProfileSchema, func(p types.Property) bool {
		return suitableAsIdentifier(p.Type)
	})
}

// Profile represents a profile.
type Profile struct {
	MPID       string         `json:"mpid"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	Attributes map[string]any `json:"attributes"`
}

// Profiles returns the profiles, the profile schema, and an estimate of their
// total number without applying first and limit. It returns the profiles that
// satisfies the filter, if not nil, and in range [first,first+limit] with
// first >= 0 and 0 < limit <= 1000 and only the given properties. properties
// cannot be empty.
//
// order is the name of the property by which to sort the returned profiles and
// cannot have type json, array, object, or map; when not provided, the profiles
// are ordered by their last change time.
//
// orderDesc control whether the returned profiles should be ordered in+
// descending order instead of ascending, which is the default.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore. It returns an errors.UnprocessableError error with code
//
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
//   - OrderNotExist, if order does not exist in schema.
//   - OrderTypeNotSortable, if the type of the order property is not sortable.
//   - PropertyNotExist, if a property does not exist.
func (this *Workspace) Profiles(ctx context.Context, properties []string, filter *Filter, order string, orderDesc bool, first, limit int) ([]Profile, types.Type, int, error) {

	this.core.mustBeOpen()

	ws := this.workspace

	profileProperties := ws.ProfileSchema.Properties()

	// Validate the properties.
	if len(properties) == 0 {
		return nil, types.Type{}, 0, errors.BadRequest("properties is empty")
	}
	for _, name := range properties {
		if _, ok := profileProperties.ByName(name); !ok {
			if name == "" {
				return nil, types.Type{}, 0, errors.BadRequest("a property name is empty")
			}
			if !types.IsValidPropertyName(name) {
				return nil, types.Type{}, 0, errors.BadRequest("property name %q is not valid", name)
			}
			return nil, types.Type{}, 0, errors.Unprocessable(PropertyNotExist, "property name %s does not exist", name)
		}
	}

	// Validate the filter.
	var where *state.Where
	if filter != nil {
		_, err := validateFilter(filter, ws.ProfileSchema, state.Destination, state.TargetUser)
		if err != nil {
			if err, ok := err.(types.PathNotExistError); ok {
				return nil, types.Type{}, 0, errors.Unprocessable(PropertyNotExist, "filter's property %s does not exist", err.Path)
			}
			return nil, types.Type{}, 0, errors.BadRequest("filter is not valid: %w", err)
		}
		where = convertFilterToWhere(filter, ws.ProfileSchema)
	}

	// Validate the order.
	if order != "" {
		orderProperty, ok := profileProperties.ByName(order)
		if !ok {
			if !types.IsValidPropertyName(order) {
				return nil, types.Type{}, 0, errors.BadRequest("order %q is not a valid property name", order)
			}
			return nil, types.Type{}, 0, errors.Unprocessable(OrderNotExist, "order %s does not exist in schema", order)
		}
		switch orderProperty.Type.Kind() {
		case types.JSONKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return nil, types.Type{}, 0, errors.Unprocessable(OrderTypeNotSortable,
				"cannot sort by %s: property has type %s", order, orderProperty.Type)
		}
	} else {
		order = "_updated_at"
	}

	// Validate first and limit.
	if first < 0 || first > maxInt32 {
		return nil, types.Type{}, 0, errors.BadRequest("first %d in not valid", first)
	}
	if limit < 1 || limit > 1000 {
		return nil, types.Type{}, 0, errors.BadRequest("limit %d is not valid", limit)
	}

	// Read the profiles.
	rows, total, err := this.store.Profiles(ctx, datastore.Query{
		Properties: append([]string{"_mpid", "_updated_at"}, properties...),
		Where:      where,
		OrderBy:    order,
		OrderDesc:  orderDesc,
		First:      first,
		Limit:      limit,
	})
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, types.Type{}, 0, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if err, ok := err.(*datastore.UnavailableError); ok {
			return nil, types.Type{}, 0, errors.Unavailable("%s", err)
		}
		return nil, types.Type{}, 0, err
	}

	// Create the schema to return, with only the requested properties.
	props := make([]types.Property, len(properties))
	for i, name := range properties {
		props[i], _ = profileProperties.ByName(name)
	}
	schema := types.Object(props)

	profiles := make([]Profile, len(rows))
	for i, row := range rows {
		profiles[i].MPID = row["_mpid"].(string)
		profiles[i].UpdatedAt = row["_updated_at"].(time.Time)
		profiles[i].Attributes = row
		delete(row, "_mpid")
		delete(row, "_updated_at")
	}

	return profiles, schema, total, nil
}

// Rename renames the workspace with the given new name.
// name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the workspace does not exist
// anymore.
func (this *Workspace) Rename(ctx context.Context, name string) error {
	this.core.mustBeOpen()
	if name == this.workspace.Name {
		return nil
	}
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	n := state.RenameWorkspace{
		Workspace: this.workspace.ID,
		Name:      name,
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET name = $1 WHERE id = $2", n.Name, n.Workspace)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, errors.NotFound("workspace %d does not exist", n.Workspace)
		}
		return n, err
	})
	return err
}

// RepairWarehouse repairs the database objects needed by Meergo on the
// workspace's data warehouse.
func (this *Workspace) RepairWarehouse(ctx context.Context) error {
	this.core.mustBeOpen()
	err := this.store.Repair(ctx, this.workspace.ProfileSchema)
	if err != nil {
		if err, ok := (err).(*datastore.UnavailableError); ok {
			return errors.Unavailable("%s", err)
		}
		return err
	}
	return nil
}

// ServeUI serves the user interface for the given connector, with the given
// role. event is the event and settings are connector's settings. oAuth is the
// OAuth token returned by the (*Workspace).OAuth method, it is required if the
// connector requires OAuth.
//
// It returns an errors.UnprocessableError error with code:
//
//   - ConnectorNotExist, if the connector does not exist.
//   - EventNotExist, if the event does not exist.
//   - InvalidSettings, if the settings are not valid.
func (this *Workspace) ServeUI(ctx context.Context, event string, settings json.Value, connector string, role Role, authToken string) (json.Value, error) {

	this.core.mustBeOpen()

	if connector == "" {
		return nil, errors.BadRequest("connector code is empty")
	}
	if role != Source && role != Destination {
		return nil, errors.BadRequest("role %d is not valid", role)
	}
	c, ok := this.core.state.Connector(connector)
	if !ok {
		return nil, errors.Unprocessable(ConnectorNotExist, "connector %q does not exist", connector)
	}

	if role == Source && !c.HasSourceSettings || role == Destination && !c.HasDestinationSettings {
		return nil, errors.BadRequest("connector %s does not have %s settings", connector, strings.ToLower(role.String()))
	}

	if (authToken == "") != (c.OAuth == nil) {
		if authToken == "" {
			return nil, errors.BadRequest("authorization token is required by connector %s", c.Code)
		}
		return nil, errors.BadRequest("connector %s does not support authorization", c.Code)
	}

	// Decode oAuth.
	var a authorizedOAuthAccount
	if authToken != "" {
		data, err := base62.DecodeString(authToken)
		if err != nil {
			return nil, errors.BadRequest("authorization token is not valid")
		}
		err = json.Unmarshal(data, &a)
		if err != nil {
			return nil, errors.BadRequest("authorization token is not valid")
		}
	}

	var clientSecret string
	if authToken != "" {
		clientSecret = c.OAuth.ClientSecret
	}
	conf := &connections.ConnectorConfig{
		Role: state.Role(role),
	}
	conf.OAuth.Account = a.Code
	conf.OAuth.ClientSecret = clientSecret
	conf.OAuth.AccessToken = a.AccessToken

	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	ui, err := this.core.connections.ServeConnectorUI(ctx, c, conf, event, settings)
	if err != nil {
		if err == connectors.ErrUIEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for connector %s", event, c.Code)
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

// StartIdentityResolution starts an Identity Resolution operation that resolves
// the identities of the workspace.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InspectionMode, if the data warehouse is in inspection mode.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
//   - OperationAlreadyExecuting, if another operation (identity resolution or
//     profile schema update) is already running.
func (this *Workspace) StartIdentityResolution(ctx context.Context) error {
	switch this.store.Mode() {
	case state.Inspection:
		return errors.Unprocessable(InspectionMode, "data warehouse is in inspection mode")
	case state.Maintenance:
		return errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
	}
	return this.core.startIdentityResolution(ctx, this.workspace.ID)
}

// TestWarehouseUpdate tests the update of the workspace's warehouse.
//
// It returns an errors.UnprocessableError with code:
//
//   - DifferentWarehouse, if the settings connect to a different
//     data warehouse.
//   - InvalidWarehouseSettings, if the settings are not valid.
//   - NotReadOnlyMCPSettings, if the MCP settings do not grant access to a
//     read-only user on the data warehouse.
func (this *Workspace) TestWarehouseUpdate(ctx context.Context, settings, mcpSettings []byte) error {
	this.core.mustBeOpen()
	ws := this.workspace
	settings, err := this.core.datastore.NormalizeWarehouseSettings(ws.Warehouse.Platform, settings)
	if err != nil {
		if err, ok := err.(*warehouses.SettingsError); ok {
			return errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return err
	}
	if mcpSettings != nil {
		// TODO(Gianluca): for https://github.com/meergo/meergo/issues/1833.
		if this.workspace.Warehouse.Platform == "Snowflake" {
			return errors.BadRequest("MCP feature data is currently not supported for workspaces connected to a Snowflake warehouse")
		}
		mcpSettings, err = this.core.datastore.NormalizeWarehouseSettings(ws.Warehouse.Platform, mcpSettings)
		if err != nil {
			if err, ok := err.(*warehouses.SettingsError); ok {
				return errors.Unprocessable(InvalidWarehouseSettings, "data warehouse MCP settings are not valid: %w", err.Err)
			}
			return err
		}
		if bytes.Equal(settings, mcpSettings) {
			return errors.Unprocessable(InvalidWarehouseSettings, "the MCP settings must be different from the data warehouse settings")
		}
		err = this.core.datastore.CheckMCPSettings(ctx, ws.Warehouse.Platform, mcpSettings)
		if err != nil {
			if err, ok := err.(*warehouses.SettingsNotReadOnly); ok {
				return errors.Unprocessable(NotReadOnlyMCPSettings, "invalid MCP settings: %s", err)
			}
			if err, ok := err.(*datastore.UnavailableError); ok {
				return errors.Unavailable("%s", err)
			}
			return err
		}
	}
	err = this.store.TestWarehouseUpdate(ctx, settings)
	if err != nil {
		if err, ok := err.(*datastore.UnavailableError); ok {
			return errors.Unavailable("%s", err)
		}
		if err == datastore.ErrDifferentWarehouse {
			return errors.Unprocessable(DifferentWarehouse, "the data warehouse is a different data warehouse")
		}
		return err
	}
	return nil
}

// Update updates the name and the displayed properties of the workspace. name
// must be between 1 and 100 runes long. displayedProperties must contain valid
// displayed property names. A valid displayed property name is an empty string,
// or alternatively a valid property name between 1 and 100 runes long.
func (this *Workspace) Update(ctx context.Context, name string, uiPreferences UIPreferences) error {
	this.core.mustBeOpen()
	if err := util.ValidateStringField("name", name, 100); err != nil {
		return errors.BadRequest("%s", err)
	}
	if err := validateUIPreferences(uiPreferences); err != nil {
		return errors.BadRequest("%s", err)
	}
	ws := this.workspace
	n := state.UpdateWorkspace{
		Workspace:     ws.ID,
		Name:          name,
		UIPreferences: state.UIPreferences(uiPreferences),
	}
	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		_, err := tx.Exec(ctx, "UPDATE workspaces SET name = $1, ui_profile_image = $2, "+
			"ui_profile_first_name = $3, ui_profile_last_name = $4, "+
			"ui_profile_extra = $5 WHERE id = $6",
			n.Name, n.UIPreferences.Profile.Image, n.UIPreferences.Profile.FirstName,
			n.UIPreferences.Profile.LastName, n.UIPreferences.Profile.Extra, n.Workspace)
		if err != nil {
			return nil, err
		}
		return n, nil
	})
	return err
}

// UpdateIdentityResolutionSettings updates the identity resolution settings of
// the workspace.
//
// runOnBatchImport determines whether the identities should be resolved
// automatically every time a batch import is completed.
//
// identifiers specify the identity resolution identifiers in the specified
// order. An identifier must be a property in the profile schema with a type of
// int, uuid, ip, string, or decimal with zero scale. Identifiers cannot be
// repeated.
//
// It returns an errors.UnprocessableError error with code:
//
//   - AlterSchemaInExecution, if an alter schema operation is currently running
//     on the workspace.
//   - IdentityResolutionInExecution, if an identity resolution operation is
//     currently running on the workspace.
//   - PropertyNotExist, if an identifier path does not exist in the user
//     schema.
//   - TypeNotAllowed, if an identifier path's type, as defined in the user
//     schema, is not allowed for identifiers.
func (this *Workspace) UpdateIdentityResolutionSettings(ctx context.Context, runOnBatchImport bool, identifiers []string) error {

	this.core.mustBeOpen()

	for i, id := range identifiers {
		if !types.IsValidPropertyPath(id) {
			return errors.BadRequest("identifier %q is not a valid property path", id)
		}
		if slices.Contains(identifiers[i+1:], id) {
			return errors.BadRequest("identifier %q is repeated", id)
		}
	}

	if identifiers == nil {
		identifiers = []string{}
	}
	ws := this.workspace
	n := state.UpdateIdentityResolutionSettings{
		Workspace:                      ws.ID,
		ResolveIdentitiesOnBatchImport: runOnBatchImport,
		Identifiers:                    identifiers,
	}

	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		var irOpID, alterSchemaOpID *string
		err := tx.QueryRow(ctx, "SELECT alter_profile_schema_id, ir_id FROM workspaces WHERE id = $1",
			n.Workspace).Scan(&alterSchemaOpID, &irOpID)
		if err != nil {
			return nil, err
		}
		if alterSchemaOpID != nil {
			return nil, errors.Unprocessable(AlterSchemaInExecution, "alter schema is in execution so the identifiers cannot be updated")
		}
		if irOpID != nil {
			return nil, errors.Unprocessable(IdentityResolutionInExecution, "identity resolution is in execution so the identifiers cannot be updated")
		}
		if len(identifiers) > 0 {
			var s []byte
			err := tx.QueryRow(ctx, "SELECT profile_schema FROM workspaces WHERE id = $1", n.Workspace).Scan(&s)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return nil, err
			}
			var schema types.Type
			err = json.Unmarshal(s, &schema)
			if err != nil {
				return nil, err
			}
			properties := schema.Properties()
			for _, path := range identifiers {
				p, err := properties.ByPath(path)
				if err != nil {
					return nil, errors.Unprocessable(PropertyNotExist, "property %q does not exist in the profile schema", path)
				}
				if !suitableAsIdentifier(p.Type) {
					return nil, errors.Unprocessable(TypeNotAllowed, "property %q has a type %s, which is not allowed for identifiers", path, p.Type)
				}
			}
		}
		_, err = tx.Exec(ctx, "UPDATE workspaces SET resolve_identities_on_batch_import = $1,\n"+
			"identifiers = $2 WHERE id = $3", n.ResolveIdentitiesOnBatchImport, n.Identifiers, n.Workspace)
		if err != nil {
			return nil, err
		}
		return n, nil
	})

	return err
}

// UpdateWarehouse updates the mode, settings and MCP settings (which can be
// nil) of the warehouse associated with the workspace.
//
// If cancelIncompatibleOperations is true, the operations currently in progress
// on the warehouse that are incompatible with mode are cancelled.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore, and it returns an errors.UnprocessableError error with code
//
//   - DifferentWarehouse, if the settings connect to a different
//     data warehouse.
//   - InvalidWarehouseSettings, if the settings are not valid.
//   - NotReadOnlyMCPSettings, if the MCP settings do not grant access to a
//     read-only user on the data warehouse.
func (this *Workspace) UpdateWarehouse(ctx context.Context, mode WarehouseMode, settings, mcpSettings []byte, cancelIncompatibleOperations bool) error {
	this.core.mustBeOpen()

	switch mode {
	case Normal, Inspection, Maintenance:
	default:
		return errors.BadRequest("mode %d is not valid", mode)
	}

	ws := this.workspace

	settings, err := this.core.datastore.NormalizeWarehouseSettings(ws.Warehouse.Platform, settings)
	if err != nil {
		if err, ok := err.(*warehouses.SettingsError); ok {
			return errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return err
	}

	if mcpSettings != nil {
		// TODO(Gianluca): for https://github.com/meergo/meergo/issues/1833.
		if this.workspace.Warehouse.Platform == "Snowflake" {
			return errors.BadRequest("MCP feature data is currently not supported for workspaces connected to a Snowflake warehouse")
		}
		var err error
		mcpSettings, err = this.core.datastore.NormalizeWarehouseSettings(ws.Warehouse.Platform, mcpSettings)
		if err != nil {
			if err, ok := err.(*warehouses.SettingsError); ok {
				return errors.Unprocessable(InvalidWarehouseSettings, "data warehouse MCP settings are not valid: %w", err.Err)
			}
			return err
		}
		if bytes.Equal(settings, mcpSettings) {
			return errors.Unprocessable(InvalidWarehouseSettings, "the MCP settings must be different from the data warehouse settings")
		}
		err = this.core.datastore.CheckMCPSettings(ctx, ws.Warehouse.Platform, mcpSettings)
		if err != nil {
			if err, ok := err.(*warehouses.SettingsNotReadOnly); ok {
				return errors.Unprocessable(NotReadOnlyMCPSettings, "invalid MCP settings: %s", err)
			}
			if err, ok := err.(*datastore.UnavailableError); ok {
				return errors.Unavailable("%s", err)
			}
			return err
		}
	}

	err = this.store.TestWarehouseUpdate(ctx, settings)
	if err != nil {
		if err, ok := err.(*datastore.UnavailableError); ok {
			return errors.Unavailable("%s", err)
		}
		if err == datastore.ErrDifferentWarehouse {
			return errors.Unprocessable(DifferentWarehouse, "the data warehouse is a different data warehouse")
		}
		return nil
	}

	n := state.UpdateWarehouse{
		Workspace:                    ws.ID,
		Mode:                         state.WarehouseMode(mode),
		Settings:                     settings,
		MCPSettings:                  mcpSettings,
		CancelIncompatibleOperations: cancelIncompatibleOperations,
	}

	var mcp string
	if n.MCPSettings != nil {
		mcp = string(n.MCPSettings)
	} else {
		mcp = "null"
	}
	err = this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_mode = $1, warehouse_settings = $2, warehouse_mcp_settings = $3 WHERE id = $4",
			n.Mode, string(n.Settings), mcp, n.Workspace)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			var warehouseName string
			err = tx.QueryRow(ctx, "SELECT warehouse_name FROM workspaces WHERE id = $1", n.Workspace).Scan(&warehouseName)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return nil, err
			}
			return nil, err
		}
		return n, nil
	})

	return err
}

// UpdateWarehouseMode updates the mode of the data warehouse for the workspace.
//
// If cancelIncompatibleOperations is true, the operations currently in progress
// on the warehouse that are incompatible with mode are cancelled.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore.
func (this *Workspace) UpdateWarehouseMode(ctx context.Context, mode WarehouseMode, cancelIncompatibleOperations bool) error {
	this.core.mustBeOpen()

	switch mode {
	case Normal, Inspection, Maintenance:
	default:
		return errors.BadRequest("mode %d is not valid", mode)
	}

	ws := this.workspace

	n := state.UpdateWarehouseMode{
		Workspace:                    ws.ID,
		Mode:                         state.WarehouseMode(mode),
		CancelIncompatibleOperations: cancelIncompatibleOperations,
	}

	err := this.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_mode = $1 WHERE id = $2 AND warehouse_mode != $1", n.Mode, n.Workspace)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			exists, err := tx.QueryExists(ctx, "SELECT FROM workspaces WHERE id = $1", n.Workspace)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, errors.NotFound("workspace %d does not exist", n.Workspace)
			}
			return nil, nil
		}
		return n, nil
	})

	return err
}

// Warehouse returns platform, settings and MCP settings of the data warehouse
// for the workspace.
func (this *Workspace) Warehouse() (string, json.Value, json.Value) {
	this.core.mustBeOpen()
	ws := this.workspace
	settings := json.Value(slices.Clone(ws.Warehouse.Settings))
	var mcpSettings json.Value
	if ws.Warehouse.MCPSettings != nil {
		mcpSettings = json.Value(slices.Clone(ws.Warehouse.MCPSettings))
	} else {
		mcpSettings = json.Value("null")
	}
	return ws.Warehouse.Platform, settings, mcpSettings
}

// identities returns the identities matching the provided where condition and
// an estimate of their total number without applying first and limit.
//
// It returns the identities in range [first,first+limit] with first >= 0
// and 0 < limit <= 1000.
//
// Identities are sorted by updated-at time in descending order, so the most
// recently updated identities come first.
//
// If there are no identities, a nil slice is returned.
//
// It returns an errors.UnprocessableError error with code MaintenanceMode if
// the data warehouse is in maintenance mode.
func (this *Workspace) identities(ctx context.Context, where *state.Where, first, limit int) ([]Identity, int, error) {

	// Retrieve the identities from the data warehouse.
	records, total, err := this.store.Identities(ctx, datastore.Query{
		Properties: []string{
			"_pipeline",
			"_is_anonymous",
			"_identity_id",
			"_connection",
			"_anonymous_ids",
			"_updated_at",
		},
		Where:     where,
		OrderBy:   "_updated_at",
		OrderDesc: true,
		First:     first,
		Limit:     limit,
	})
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, 0, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		return nil, 0, err
	}

	// Create the identities from the records returned by the datastore.
	var identities []Identity

	for _, record := range records {

		// Retrieve the connection.
		connID := record["_connection"].(int)
		conn, ok := this.workspace.Connection(connID)
		if !ok {
			// The connection for this identity no longer exists, so skip this identity.
			continue
		}

		// Retrieve the pipeline.
		pipelineID := record["_pipeline"].(int)
		_, ok = conn.Pipeline(pipelineID)
		if !ok {
			// The pipeline for this identity no longer exists, so skip this identity.
			continue
		}

		// Determine the value for the identity ID.
		identityID := record["_identity_id"].(string)

		// Determine the anonymous IDs.
		var anonIDs []string
		if ids, ok := record["_anonymous_ids"].([]any); ok {
			anonIDs = make([]string, len(ids))
			for i := range ids {
				anonIDs[i] = ids[i].(string)
			}
		}

		// In the case of anonymous identities, the anonymous ID is inside the
		// identity ID, so there is the need to populate the anonymous IDs by
		// taking that value, then reset the identity ID.
		if record["_is_anonymous"].(bool) {
			anonIDs = append(anonIDs, identityID)
			identityID = ""
		}

		// Get the updated-at time.
		updatedAt := record["_updated_at"].(time.Time)

		identities = append(identities, Identity{
			UserId:       identityID,
			AnonymousIds: anonIDs,
			UpdatedAt:    updatedAt,
			Connection:   connID,
			Pipeline:     pipelineID,
		})

	}

	// Since the total is an estimate, being counted separately from the actual
	// number of identities returned, ensure to not return a value lower than
	// the actually returned number of identities.
	total = max(len(identities), total)

	return identities, total, nil
}

// ConnectionToAdd represents a connection to add to a workspace.
type ConnectionToAdd struct {

	// Name is the name of the connection. It cannot be longer than 100 runes.
	// If empty, the connection name will be the name of its connector.
	Name string `json:"name"`

	// Role is the role.
	Role Role `json:"role"`

	// Connector is the name of the connector.
	Connector string `json:"connector"`

	// Strategy is the strategy that determines how to merge anonymous and
	// non-anonymous users. It can only be provided for source SDK connections
	// whose connector supports the strategies.
	Strategy *Strategy `json:"strategy"`

	// SendingMode is the mode used for sending events. It can only be provided for
	// destination API connections that support it. In this case, it must be one of
	// the sending modes supported by the API.
	SendingMode *SendingMode `json:"sendingMode"`

	// LinkedConnections, for connections supporting events, indicate the
	// connections to which events can be sent or received. It must be nil if the
	// connection do not support events.
	LinkedConnections []int `json:"linkedConnections"`

	// Settings represents the settings of the connector.
	// It must be nil if the connector does not have settings.
	Settings json.Value `json:"settings"`
}

// WarehouseMode represents a data warehouse mode.
type WarehouseMode int

const (
	Normal WarehouseMode = iota
	Inspection
	Maintenance
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if mode is not a valid WarehouseMode value.
func (mode WarehouseMode) MarshalJSON() ([]byte, error) {
	return []byte(`"` + mode.String() + `"`), nil
}

// String returns the string representation of mode.
// It panics if mode is not a valid WarehouseMode value.
func (mode WarehouseMode) String() string {
	switch mode {
	case Normal:
		return "Normal"
	case Inspection:
		return "Inspection"
	case Maintenance:
		return "Maintenance"
	}
	panic("invalid warehouse mode")
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (mode *WarehouseMode) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return errors.BadRequest("mode cannot be null")
	}
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	m, ok := v.(string)
	if !ok {
		return fmt.Errorf("json: cannot scan a %T value into an WarehouseMode value", v)
	}
	var mo WarehouseMode
	switch m {
	case "Normal":
		mo = Normal
	case "Inspection":
		mo = Inspection
	case "Maintenance":
		mo = Maintenance
	default:
		return fmt.Errorf("json: invalid WarehouseMode: %s", m)
	}
	*mode = mo
	return nil
}

// suitableAsIdentifier reports whether a property with type t can be used as
// identifier.
func suitableAsIdentifier(t types.Type) bool {
	switch t.Kind() {
	case types.StringKind,
		types.IntKind,
		types.UUIDKind,
		types.IPKind:
		return true
	case types.DecimalKind:
		return t.Scale() == 0
	default:
		return false
	}
}

// Identity represents an identity.
type Identity struct {
	UserId       string    `json:"userId"`                       // empty string for identities imported from anonymous events.
	AnonymousIds []string  `json:"anonymousIds,format:emitnull"` // nil for identities not imported from events.
	Connection   int       `json:"connection"`
	Pipeline     int       `json:"pipeline"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// filterWorkspacePipelines returns from pipelines, only the pipelines of the
// provided workspace. It does not change pipelines.
func filterWorkspacePipelines(ws *state.Workspace, pipelines []int) []int {
	notExists := map[int]struct{}{}
	for _, pipeline := range pipelines {
		notExists[pipeline] = struct{}{}
	}
	for _, c := range ws.Connections() {
		for _, p := range c.Pipelines() {
			delete(notExists, p.ID)
		}
	}
	if len(notExists) == 0 {
		return pipelines
	}
	pipelines = slices.DeleteFunc(slices.Clone(pipelines), func(id int) bool {
		_, ok := notExists[id]
		return ok
	})
	return pipelines
}

// validateUIPreferences validates whether the given UI preferences are valid or
// not, returning an error if they are not.
func validateUIPreferences(preferences UIPreferences) error {
	if n := preferences.Profile.Image; n != "" && (len(n) > 1024 || !types.IsValidPropertyPath(n)) {
		return fmt.Errorf("invalid profile 'image' %q", n)
	}
	if n := preferences.Profile.FirstName; n != "" && (len(n) > 1024 || !types.IsValidPropertyPath(n)) {
		return fmt.Errorf("invalid profile 'firstName' %q", n)
	}
	if n := preferences.Profile.LastName; n != "" && (len(n) > 1024 || !types.IsValidPropertyPath(n)) {
		return fmt.Errorf("invalid profile 'lastName' %q", n)
	}
	if n := preferences.Profile.Extra; n != "" && (len(n) > 1024 || !types.IsValidPropertyPath(n)) {
		return fmt.Errorf("invalid profile 'extra' %q", n)
	}
	return nil
}

const maxRawQuerySize = 10 * 1024 * 1024 // 10 MiB.

// RawQueryWarehouse executes a query on the warehouse, returning the result as
// a json.Value representing a JSON Array (representing the rows) of JSON Arrays
// (representing the values for each column).
//
// If the JSON size exceeds the allowed maximum, this method returns a valid
// JSON array of arrays containing only the rows within the limit, and
// simultaneously returns an error indicating the issue.
//
// If the workspace has no MCP settings configured, this method returns an
// error.
//
// TODO(Gianluca): the error handling is currently minimal. See the issue
// https://github.com/meergo/meergo/issues/1667.
func (this *Workspace) RawQueryWarehouse(ctx context.Context, query string) (json.Value, error) {

	this.core.mustBeOpen()

	// TODO(Gianluca): here the warehouse mode is not checked. The reason is
	// that the mode is currently stored in the store. We should review all
	// this. This is discussed in the issue https://github.com/meergo/meergo/issues/1224.

	// Retrieve the warehouse instance for the MCP.
	this.core.mcpMu.Lock()
	mcp, ok := this.core.mcp[this.workspace.ID]
	this.core.mcpMu.Unlock()
	if !ok {
		return nil, errors.New("workspace not found")
	}
	if mcp == nil {
		return nil, errors.New("the workspace lacks the MCP (Model Context Protocol) user configuration required to access the data warehouse")
	}

	// Execute the query on the data warehouse.
	rows, columnCount, err := mcp.RawQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	// Build the JSON response.
	b := json.NewBuffer()
	defer rows.Close()
	comma := false
	b.WriteByte('[')
	for rows.Next() {
		row := make([]any, columnCount)
		for i := range row {
			var v any
			row[i] = &v
		}
		err := rows.Scan(row...)
		if err != nil {
			return nil, err
		}
		size := b.Len()
		if comma {
			b.WriteByte(',')
		}
		err = b.Encode(row)
		if err != nil {
			return nil, err
		}
		// Truncate the response if it exceeds the limit, simultaneously
		// returning the truncated response and an error.
		if b.Len()+len("]") >= maxRawQuerySize {
			b.Truncate(size)
			b.WriteByte(']')
			value, err := b.Value()
			if err != nil {
				return nil, err
			}
			return value, fmt.Errorf("only a subset of rows was returned because the total size exceeded the %d-byte limit", maxRawQuerySize)
		}
		comma = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	b.WriteByte(']')

	value, err := b.Value()
	if err != nil {
		return nil, err
	}

	return value, nil
}
