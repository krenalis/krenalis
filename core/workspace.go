//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package core

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/events/collector"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/telemetry"
	"github.com/meergo/meergo/types"

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
	UserSchema                     types.Type     `json:"userSchema"`
	UserPrimarySources             map[string]int `json:"userPrimarySources,format:emitnull"`
	ResolveIdentitiesOnBatchImport bool           `json:"resolveIdentitiesOnBatchImport"`
	Identifiers                    []string       `json:"identifiers,format:emitnull"`
	WarehouseMode                  WarehouseMode  `json:"warehouseMode"`
	UIPreferences                  UIPreferences  `json:"uiPreferences"`
}

type UIPreferences struct {
	UserProfile struct {
		Image     string `json:"image"`     // property path.
		FirstName string `json:"firstName"` // property path.
		LastName  string `json:"lastName"`  // property path.
		Extra     string `json:"extra"`     // property path.
	} `json:"userProfile"`
}

// ActionStep represents a step of an action.
type ActionStep int

const (
	ReceiveStep          = ActionStep(metrics.ReceiveStep)
	InputValidationStep  = ActionStep(metrics.InputValidationStep)
	FilterStep           = ActionStep(metrics.FilterStep)
	TransformationStep   = ActionStep(metrics.TransformationStep)
	OutputValidationStep = ActionStep(metrics.OutputValidationStep)
	FinalizeStep         = ActionStep(metrics.FinalizeStep)
)

func (step ActionStep) String() string {
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
	panic("core: invalid ActionStep")
}

// ParseActionStep parses an action step and returns it. If step is not a valid
// returns 0 and an error.
func ParseActionStep(step string) (ActionStep, error) {
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

// ActionError represents an action error.
type ActionError struct {
	Action       int        `json:"action"`
	Step         ActionStep `json:"step"`
	Count        int        `json:"count"`
	Message      string     `json:"message"`
	LastOccurred time.Time  `json:"lastOccurred"`
}

// Action returns the action with identifier id of the workspace.
// It returns an errors.NotFound error if the action does not exist.
func (this *Workspace) Action(id int) (*Action, error) {
	this.core.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("identifier %d is not a valid action identifier", id)
	}
	a, ok := this.core.state.Action(id)
	if !ok || a.Connection().Workspace().ID != this.workspace.ID {
		return nil, errors.NotFound("action %d does not exist", id)
	}
	var action Action
	action.fromState(this.core, this.store, a)
	return &action, nil
}

// ActionErrors returns the errors for the provided actions within the time
// range [start,end). The end time must not precede the start time, and both
// must be within [metrics.MinTime,metrics.MaxTime]. actions must not be empty.
// Returned errors are limited to [first, first+limit), where first >= 0 and
// 0 < limit <= 100.
func (this *Workspace) ActionErrors(ctx context.Context, start, end time.Time, actions []int, step *ActionStep, first, limit int) ([]ActionError, error) {

	this.core.mustBeOpen()

	start = start.UTC()
	end = end.UTC()

	// Validate start and end.
	if start.Before(metrics.MinTime) {
		return nil, errors.New("start date is too far in the past")
	}
	if end.After(metrics.MaxTime) {
		return nil, errors.New("end date date is too far in the future")
	}
	if end.Before(start) {
		return nil, fmt.Errorf("end date cannot be earlier than start date")
	}

	// Validate actions.
	if len(actions) == 0 {
		return nil, errors.BadRequest("actions cannot be empty")
	}
	for _, action := range actions {
		if action < 1 || action > maxInt32 {
			return nil, errors.BadRequest("action %d is not valid", action)
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

	actions = filterWorkspaceActions(this.workspace, actions)
	if len(actions) == 0 {
		return []ActionError{}, nil
	}

	metricsErrors, err := this.core.metrics.Errors(ctx, start, end, actions, s, first, limit)
	if err != nil {
		return nil, err
	}

	errs := make([]ActionError, len(metricsErrors))
	for i, e := range metricsErrors {
		errs[i] = ActionError{
			Action:       e.Action,
			Step:         ActionStep(e.Step),
			Count:        e.Count,
			Message:      e.Message,
			LastOccurred: e.LastOccurred,
		}
	}

	return errs, nil
}

// ActionMetrics represents action metrics for a time period.
type ActionMetrics struct {
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

// ActionMetricsPerDate returns metrics aggregated by day for the time interval
// between the specified start and end dates. The dates must be no earlier than
// 1970-01-01 and no later than 2262-04-10. The day of the start date must be at
// least one day before the day of the end date. actions specifies the actions
// for which metrics are returned and must not be empty.
func (this *Workspace) ActionMetricsPerDate(ctx context.Context, start, end time.Time, actions []int) (ActionMetrics, error) {

	this.core.mustBeOpen()

	start = start.UTC().Truncate(24 * time.Hour)
	end = end.UTC().Truncate(24 * time.Hour)

	// Validate start and end.
	if start.Before(metrics.MinTime) {
		return ActionMetrics{}, errors.BadRequest("start date is too far in the past")
	}
	if end.After(metrics.MaxTime) {
		return ActionMetrics{}, errors.BadRequest("end date date is too far in the future")
	}
	if !end.After(start) {
		return ActionMetrics{}, errors.BadRequest("day of the end date must be after the day of the start date")
	}

	// Validate actions.
	if len(actions) == 0 {
		return ActionMetrics{}, errors.BadRequest("actions if non-nil, cannot be empty")
	}
	for _, action := range actions {
		if action < 1 || action > maxInt32 {
			return ActionMetrics{}, errors.BadRequest("action %d is not valid", action)
		}
	}

	actions = filterWorkspaceActions(this.workspace, actions)
	if len(actions) == 0 {
		number := int(end.Sub(start).Hours() / 24)
		return ActionMetrics{
			Start:  start,
			End:    end,
			Passed: make([][6]int, number),
			Failed: make([][6]int, number),
		}, nil
	}

	metrics, err := this.core.metrics.MetricsPerDate(ctx, start, end, actions)
	if err != nil {
		return ActionMetrics{}, err
	}

	return ActionMetrics{
		Start:  metrics.Start,
		End:    metrics.End,
		Passed: metrics.Passed,
		Failed: metrics.Failed,
	}, nil
}

// ActionMetricsPerTimeUnit returns metrics for the specified number of minutes,
// hours, or days based on the unit, which can be Minute, Hour, or Day, up to
// the current time. number must be in the following ranges: [1,60] for minutes,
// [1,48] for hours, and [1,30] for days. actions specifies the actions for
// which metrics are returned and must not be empty.
func (this *Workspace) ActionMetricsPerTimeUnit(ctx context.Context, number int, unit MetricUnit, actions []int) (ActionMetrics, error) {

	this.core.mustBeOpen()

	// Validate number and unit.
	switch unit {
	case Minute:
		if number < 1 || number > 60 {
			return ActionMetrics{}, errors.BadRequest("minutes must be in range [1,60]")
		}
	case Hour:
		if number < 1 || number > 48 {
			return ActionMetrics{}, errors.BadRequest("hours must be in range [1,48]")
		}
	case Day:
		if number < 1 || number > 30 {
			return ActionMetrics{}, errors.BadRequest("days must be in range [1,30]")
		}
	}

	// Validate actions.
	if len(actions) == 0 {
		return ActionMetrics{}, errors.BadRequest("actions if non-nil, cannot be empty")
	}
	for _, action := range actions {
		if action < 1 || action > maxInt32 {
			return ActionMetrics{}, errors.BadRequest("action %d is not valid", action)
		}
	}

	actions = filterWorkspaceActions(this.workspace, actions)
	if len(actions) == 0 {
		return ActionMetrics{
			Passed: make([][6]int, number),
			Failed: make([][6]int, number),
		}, nil
	}

	metrics, err := this.core.metrics.MetricsPerTimeUnit(ctx, number, time.Duration(unit), actions)
	if err != nil {
		return ActionMetrics{}, err
	}

	return ActionMetrics{
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
		return "", errors.BadRequest("connector name is empty")
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

	auth, err := this.core.connectors.GrantAuthorization(ctx, c, code, redirectionURI)
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
		Connector:         conn.Name,
		ConnectorType:     ConnectorType(conn.Type),
		Role:              Role(c.Role),
		Strategy:          (*Strategy)(c.Strategy),
		SendingMode:       (*SendingMode)(c.SendingMode),
		WebsiteHost:       c.WebsiteHost,
		LinkedConnections: slices.Clone(c.LinkedConnections),
		ActionsCount:      len(c.Actions()),
		Health:            Health(c.Health),
	}

	// Set the actions.
	actions := c.Actions()
	a := make([]Action, len(actions))
	connection.Actions = &a
	for i, a := range actions {
		(*connection.Actions)[i].fromState(this.core, this.store, a)
	}

	// Set the event types.
	if conn.Type == state.App && c.Role == state.Destination &&
		c.Connector().DestinationTargets.Contains(state.Events) {
		appEventTypes, err := connection.app().EventTypes(ctx)
		if err != nil {
			return nil, err
		}
		eventTypes := make([]EventType, len(appEventTypes))
		for i, et := range appEventTypes {
			eventTypes[i] = EventType{
				ID:          et.ID,
				Name:        et.Name,
				Description: et.Description,
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
			Connector:         conn.Name,
			ConnectorType:     ConnectorType(conn.Type),
			Role:              Role(c.Role),
			Strategy:          (*Strategy)(c.Strategy),
			SendingMode:       (*SendingMode)(c.SendingMode),
			WebsiteHost:       c.WebsiteHost,
			LinkedConnections: slices.Clone(c.LinkedConnections),
			ActionsCount:      len(c.Actions()),
			Health:            Health(c.Health),
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
		return 0, errors.BadRequest("connector name is empty")
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
	if host := connection.WebsiteHost; host != "" {
		if _, _, err := parseWebsiteHost(host); err != nil {
			return 0, errors.BadRequest("website host %q is not valid", host)
		}
	}

	c, ok := this.core.state.Connector(connection.Connector)
	if !ok {
		return 0, errors.Unprocessable(ConnectorNotExist, "connector %q does not exist", connection.Connector)
	}
	switch c.Type {
	case state.File:
		return 0, errors.BadRequest("connections cannot have type file")
	case state.Mobile, state.Server, state.Website:
		if connection.Role == Destination {
			return 0, errors.BadRequest("%s connections cannot be destinations", strings.ToLower(c.Type.String()))
		}
	}

	if connection.WebsiteHost != "" && c.Type != state.Website {
		return 0, errors.BadRequest("%s connections cannot have a website host", strings.ToLower(c.Type.String()))
	}

	// Validate linked connections.
	err := validateLinkedConnections(connection.LinkedConnections, c, this.workspace, state.Role(connection.Role))
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
		WebsiteHost:       connection.WebsiteHost,
		LinkedConnections: connection.LinkedConnections,
	}
	if n.Name == "" {
		n.Name = c.Name
	}
	slices.Sort(n.LinkedConnections)

	// Validate the strategy.
	if connection.Role == Source {
		switch c.Type {
		case state.Mobile, state.Website:
			if connection.Strategy == nil {
				return 0, errors.BadRequest("%s connections must have a strategy", strings.ToLower(c.Type.String()))
			}
		default:
			if connection.Strategy != nil {
				return 0, errors.BadRequest("%s connections cannot have a strategy", strings.ToLower(c.Type.String()))
			}
		}
	}

	// Validate the sending mode.
	if connection.Role == Destination {
		if c.SendingMode != nil {
			if connection.SendingMode == nil {
				return 0, errors.BadRequest("connector %s requires a sending mode", c.Name)
			}
			if !c.SendingMode.Contains(state.SendingMode(*connection.SendingMode)) {
				return 0, errors.BadRequest("connector %s does not support sending mode %s", c.Name, *c.SendingMode)
			}
		} else if connection.SendingMode != nil {
			return 0, errors.BadRequest("connector %s does not support sending modes", c.Name)
		}
	} else if connection.SendingMode != nil {
		return 0, errors.BadRequest("source connections cannot have a sending mode")
	}

	// Validate the website host.
	if n.WebsiteHost != "" {
		if c.Type != state.Website {
			return 0, errors.BadRequest("connector %s cannot have a website host, it's a %s",
				c.Name, strings.ToLower(c.Type.String()))
		}
		if h, p, found := strings.Cut(n.WebsiteHost, ":"); h == "" || len(n.WebsiteHost) > 255 {
			return 0, errors.BadRequest("website host %q is not valid", n.WebsiteHost)
		} else if found {
			if port, _ := strconv.Atoi(p); port < 1 || port > 65535 {
				return 0, errors.BadRequest("website host %q is not valid", n.WebsiteHost)
			}
		}
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
		if account.Workspace != this.workspace.ID || account.Connector != c.Name {
			return 0, errors.BadRequest("authorization token is not valid")
		}
		n.Account.Code = account.Code
		a, ok := this.workspace.AccountByCode(account.Code)
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
			return 0, errors.BadRequest("settings must be provided because connector %s has %s settings", c.Name, strings.ToLower(connection.Role.String()))
		}
	} else {
		if connection.Role == Source && !c.HasSourceSettings || connection.Role == Destination && !c.HasDestinationSettings {
			return 0, errors.BadRequest("settings cannot be provided because connector %s has no %s settings", c.Name, strings.ToLower(connection.Role.String()))
		}
		var clientSecret string
		if c.OAuth != nil {
			clientSecret = c.OAuth.ClientSecret
		}
		conf := &connectors.ConnectorConfig{
			Role: n.Role,
		}
		conf.OAuth.Account = n.Account.Code
		conf.OAuth.ClientSecret = clientSecret
		conf.OAuth.AccessToken = n.Account.AccessToken
		n.Settings, err = this.core.connectors.UpdatedSettings(ctx, c, conf, settings)
		if err != nil {
			switch err.(type) {
			case *meergo.InvalidSettingsError:
				err = errors.Unprocessable(InvalidSettings, "%s", err)
			case *connectors.UnavailableError:
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
	switch c.Type {
	case state.Mobile, state.Server, state.Website:
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

	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
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
				return err
			}
		}
		// Insert the connection.
		_, err = tx.Exec(ctx, "INSERT INTO connections "+
			"(id, workspace, name, connector, role, account,"+
			" strategy, sending_mode, website_host, linked_connections, settings)"+
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
			n.ID, n.Workspace, n.Name, n.Connector, n.Role, n.Account.ID, n.Strategy,
			n.SendingMode, n.WebsiteHost, n.LinkedConnections, string(n.Settings))
		if err != nil {
			if db.IsForeignKeyViolation(err) && db.ErrConstraintName(err) == "connections_workspace_fkey" {
				err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
			}
			return err
		}
		// Link connections.
		if n.LinkedConnections != nil {
			result, err := tx.Exec(ctx, add, n.ID)
			if err != nil {
				return err
			}
			if int(result.RowsAffected()) < len(n.LinkedConnections) {
				return errors.Unprocessable(LinkedConnectionNotExist, "a linked connection does not exist")
			}
		}
		if n.EventWriteKey != "" {
			// Insert the event write key.
			_, err = tx.Exec(ctx, "INSERT INTO event_write_keys (connection, key, creation_time) VALUES ($1, $2, $3)",
				n.ID, n.EventWriteKey, time.Now().UTC())
			if err != nil {
				return err
			}
		}
		return tx.Notify(ctx, n)
	})
	if err != nil {
		return 0, err
	}

	return n.ID, nil
}

// CreateEventListener creates an event listener for the workspace that listens
// to events and returns its identifier.
//
// size specifies the maximum number of observed events to be returned by a
// subsequent call to the ListenedEvents method and must be in the range
// [1, 1000].
//
// If filter is non-nil, only events that satisfy the filter will be observed.
//
// It returns an errors.UnprocessableError with code TooManyListeners, if there
// are already too many listeners.
func (this *Workspace) CreateEventListener(size int, filter *Filter) (string, error) {
	this.core.mustBeOpen()
	if size < 1 || size > maxEventsListenedTo {
		return "", errors.BadRequest("size %d is not valid", size)
	}
	var where *state.Where
	if filter != nil {
		_, err := validateFilter(filter, events.Schema)
		if err != nil {
			return "", errors.BadRequest("filter is not valid: %w", err)
		}
		where = convertFilterToWhere(filter, events.Schema)
	}
	id, err := this.core.events.observer.CreateListener(size, where)
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
	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "DELETE FROM workspaces WHERE id = $1", n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("workspace %d does not exist", n.ID)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// DeleteEventListener deletes the given event listener of the workspace. It
// does nothing if the listener does not exist.
func (this *Workspace) DeleteEventListener(listener string) {
	this.core.mustBeOpen()
	this.core.events.observer.DeleteListener(listener)
}

// Events returns events that match the provided filter, if not nil, and are
// within the range [first,first+limit], where first >= 0, 0 < limit <= 1000,
// and only includes the specified properties from the event schema. properties
// must contain at least one property.
//
// order specifies the property by which to sort the events. It cannot be of
// type json or object. If not provided, the events are sorted by their ID.
// orderDesc controls whether the events should be sorted in descending order,
// when true, or ascending order.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore. It returns an errors.UnprocessableError error with code
// MaintenanceMode if the data warehouse is in maintenance mode.
func (this *Workspace) Events(ctx context.Context, properties []string, filter *Filter, order string, orderDesc bool, first, limit int) ([]map[string]any, error) {

	this.core.mustBeOpen()

	// Validate the properties.
	if len(properties) == 0 {
		return nil, errors.BadRequest("properties is empty")
	}
	propertyByName := map[string]types.Property{}
	for _, p := range events.Schema.Properties() {
		propertyByName[p.Name] = p
	}
	for _, name := range properties {
		if _, ok := propertyByName[name]; !ok {
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
		_, err := validateFilter(filter, events.Schema)
		if err != nil {
			if err, ok := err.(types.PathNotExistError); ok {
				return nil, errors.BadRequest("filter's property %q does not exist", err.Path)
			}
			return nil, errors.BadRequest("filter is not valid: %w", err)
		}
		where = convertFilterToWhere(filter, events.Schema)
	}

	// Validate the order.
	if order != "" {
		if !types.IsValidPropertyName(order) {
			return nil, errors.BadRequest("order %q is not a valid property name", order)
		}
		orderProperty, ok := propertyByName[order]
		if !ok {
			return nil, errors.BadRequest("order property %q does not exist", order)
		}
		switch orderProperty.Type.Kind() {
		case types.JSONKind, types.ObjectKind:
			return nil, errors.BadRequest("cannot sort by %s: property has type %s", order, orderProperty.Type)
		}
	} else {
		order = "id"
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
			// TODO(marco): log the error in a log specific of the workspace.
			slog.Error("cannot get users from the data warehouse", "workspace", this.workspace.ID, "err", err)
			return nil, errors.Unavailable("%s", err)
		}
		return nil, err
	}

	return evts, nil
}

// Execution returns the execution with the specified identifier for an action
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
			Action:    exe.Action().ID,
			StartTime: exe.StartTime,
		}, nil
	}
	var exe Execution
	err := this.core.db.QueryRow(ctx,
		"SELECT e.id, e.action, e.start_time, e.end_time, e.passed_0, e.passed_1, e.passed_2, e.passed_3,"+
			" e.passed_4, e.passed_5, e.failed_0, e.failed_1, e.failed_2, e.failed_3, e.failed_4,"+
			" e.failed_5, e.error_message\n"+
			"FROM actions_executions e\n"+
			"INNER JOIN actions a ON a.id = e.action\n"+
			"INNER JOIN connections c ON c.id = a.connection\n"+
			"WHERE c.workspace = $1 AND e.id = $2", this.workspace.ID, id).Scan(
		&exe.ID, &exe.Action, &exe.StartTime, &exe.EndTime, &exe.Passed[0], &exe.Passed[1], &exe.Passed[2], &exe.Passed[3],
		&exe.Passed[4], &exe.Passed[5], &exe.Failed[0], &exe.Failed[1], &exe.Failed[2], &exe.Failed[3], &exe.Failed[4],
		&exe.Failed[5], &exe.Error)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NotFound("action execution %d does not exist", id)
		}
		return nil, err
	}
	if exe.EndTime == nil {
		exe.Passed = [6]int{}
		exe.Failed = [6]int{}
	}
	return &exe, nil
}

// Executions returns the executions of the actions of the workspace.
func (this *Workspace) Executions(ctx context.Context) ([]*Execution, error) {

	this.core.mustBeOpen()

	executions := []*Execution{}
	err := this.core.db.QueryScan(ctx,
		"SELECT e.id, e.action, e.start_time, e.end_time, e.passed_0, e.passed_1, e.passed_2, e.passed_3,"+
			" e.passed_4, e.passed_5, e.failed_0, e.failed_1, e.failed_2, e.failed_3, e.failed_4, e.failed_5, e.error_message\n"+
			"FROM actions_executions e\n"+
			"INNER JOIN actions a ON a.id = e.action\n"+
			"INNER JOIN connections c ON c.id = a.connection\n"+
			"WHERE c.workspace = $1\n"+
			"ORDER BY id DESC", this.workspace.ID, func(rows *db.Rows) error {
			var err error
			for rows.Next() {
				var exe Execution
				if err = rows.Scan(&exe.ID, &exe.Action, &exe.StartTime, &exe.EndTime, &exe.Passed[0], &exe.Passed[1], &exe.Passed[2], &exe.Passed[3],
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

// IdentifiersSchema returns the properties of the "users" schema that can be
// used as identifiers in the Identity Resolution.
// If none of the properties can be an identifier, this method returns the
// invalid schema.
func (this *Workspace) IdentifiersSchema() types.Type {
	this.core.mustBeOpen()
	return types.SubsetFunc(this.workspace.UserSchema, func(p types.Property) bool {
		return canBeIdentifier(p.Type)
	})
}

// Identities returns the identities of the provider user, and an estimate of
// their total number without applying first and limit.
//
// It returns the user identities in range [first,first+limit] with first >= 0
// and 0 < limit <= 1000.
//
// It returns an errors.NotFoundError error, if the user does not exist.
// It returns an errors.UnprocessableError error with code MaintenanceMode if
// the data warehouse is in maintenance mode.
func (this *Workspace) Identities(ctx context.Context, user string, first, limit int) ([]UserIdentity, int, error) {
	this.core.mustBeOpen()
	if _, ok := util.ParseUUID(user); !ok {
		return nil, 0, errors.BadRequest("user %q is not a valid user identifier", user)
	}
	if first < 0 {
		return nil, 0, errors.BadRequest("first %d is not valid", limit)
	}
	if limit < 1 || limit > 1000 {
		return nil, 0, errors.BadRequest("limit %d is not valid", limit)
	}
	where := &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
		Property: []string{"__gid__"},
		Operator: state.OpIs,
		Values:   []any{user},
	}}}
	ws := &Workspace{
		core:      this.core,
		store:     this.store,
		workspace: this.workspace,
	}
	identities, total, err := ws.userIdentities(ctx, where, first, limit)
	if err != nil {
		return nil, 0, err
	}
	if identities == nil {
		return nil, 0, errors.NotFound("user %q does not exist", user)
	}
	return identities, total, nil
}

// LatestIdentityResolution returns information about the latest Identity
// Resolution of the workspace.
//
// In particular:
//
//   - if the Identity Resolution has been started and completed, returns its
//     start time and end time;
//   - if it is in progress, returns its start time and nil for the end time;
//   - if no Identity Resolution has ever been executed, returns nil and nil.
//
// It returns an errors.UnprocessableError error with code MaintenanceMode if
// the data warehouse is in maintenance mode.
func (this *Workspace) LatestIdentityResolution(ctx context.Context) (startTime, endTime *time.Time, err error) {
	this.core.mustBeOpen()
	startTime, endTime, err = this.store.LatestIdentityResolution(ctx)
	if err != nil {
		if err, ok := err.(*datastore.UnavailableError); ok {
			return nil, nil, errors.Unavailable("%s", err)
		}
		if err == datastore.ErrMaintenanceMode {
			return nil, nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		return nil, nil, err
	}
	return startTime, endTime, nil
}

// ListenedEvents returns the events listened to the specified listener and the
// number of omitted events. If the listener does not exist, it returns an
// errors.NotFoundError.
func (this *Workspace) ListenedEvents(listener string) ([]json.Value, int, error) {
	this.core.mustBeOpen()
	observedEvents, omitted, err := this.core.events.observer.Events(listener)
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
	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET name = $1 WHERE id = $2", n.Name, n.Workspace)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			return errors.NotFound("workspace %d does not exist", n.Workspace)
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// RepairWarehouse repairs the database objects needed by Meergo on the
// workspace's data warehouse.
func (this *Workspace) RepairWarehouse(ctx context.Context) error {
	this.core.mustBeOpen()
	err := this.store.Repair(ctx, this.workspace.UserSchema)
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
func (this *Workspace) ServeUI(ctx context.Context, event string, settings json.Value, connector string, role Role, authToken string) ([]byte, error) {

	this.core.mustBeOpen()

	if connector == "" {
		return nil, errors.BadRequest("connector name is empty")
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
			return nil, errors.BadRequest("authorization token is required by connector %s", c.Name)
		}
		return nil, errors.BadRequest("connector %s does not support authorization", c.Name)
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
	conf := &connectors.ConnectorConfig{
		Role: state.Role(role),
	}
	conf.OAuth.Account = a.Code
	conf.OAuth.ClientSecret = clientSecret
	conf.OAuth.AccessToken = a.AccessToken

	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	ui, err := this.core.connectors.ServeConnectorUI(ctx, c, conf, event, settings)
	if err != nil {
		if err == meergo.ErrUIEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for connector %s", event, c.Name)
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

// StartIdentityResolution starts an Identity Resolution operation that resolves
// the identities of the workspace.
//
// It returns an errors.UnprocessableError error with code:
//
//   - InspectionMode, if the data warehouse is in inspection mode.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
func (this *Workspace) StartIdentityResolution(ctx context.Context) error {
	this.core.mustBeOpen()
	ctx, span := telemetry.TraceSpan(ctx, "Workspace.StartIdentityResolution", "workspace_id", this.workspace.ID)
	defer span.End()
	telemetry.IncrementCounter(ctx, "IdentityResolutionExecutions", 1)
	err := this.store.StartIdentityResolution(ctx)
	if err != nil {
		if err == datastore.ErrInspectionMode {
			return errors.Unprocessable(InspectionMode, "data warehouse is in inspection mode")
		}
		if err == datastore.ErrMaintenanceMode {
			return errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		return err
	}
	return nil
}

// TestWarehouseUpdate tests the update of the workspace's warehouse.
//
// It returns an errors.UnprocessableError with code:
//
//   - DifferentWarehouse, if the settings connect to a different
//     data warehouse.
//   - InvalidWarehouseSettings, if the settings are not valid.
func (this *Workspace) TestWarehouseUpdate(ctx context.Context, settings []byte) error {
	this.core.mustBeOpen()
	ws := this.workspace
	settings, err := this.core.datastore.NormalizeWarehouseSettings(ws.Warehouse.Type, settings)
	if err != nil {
		if err, ok := err.(*meergo.WarehouseSettingsError); ok {
			return errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return err
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

// Traits returns the traits of a user.
//
// It returns an errors.NotFoundError error, if the user does not exist.
// It returns an errors.UnprocessableError error with code MaintenanceMode if
// the data warehouse is in maintenance mode.
func (this *Workspace) Traits(ctx context.Context, user string) (json.Value, error) {

	this.core.mustBeOpen()

	ws := this.workspace

	// Validate the user.
	if _, ok := util.ParseUUID(user); !ok {
		return nil, errors.BadRequest("user %q is not a valid user identifier", user)
	}

	properties := types.PropertyNames(this.workspace.UserSchema)
	where := &state.Where{Logical: state.OpAnd, Conditions: []state.WhereCondition{{
		Property: []string{"__id__"},
		Operator: state.OpIs,
		Values:   []any{user},
	}}}

	// Retrieve the user traits.
	records, _, err := this.store.Users(ctx, datastore.Query{
		Properties: properties,
		Where:      where,
		Limit:      1,
	})
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if err, ok := err.(*datastore.UnavailableError); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			slog.Error("cannot get users from the data warehouse", "workspace", ws.ID, "err", err)
			return nil, errors.Unavailable("%s", err)
		}
		return nil, err
	}
	if len(records) == 0 {
		return nil, errors.NotFound("user %q does not exist", user)
	}

	return types.Marshal(records[0], ws.UserSchema)
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
	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE workspaces SET name = $1, ui_user_profile_image = $2, "+
			"ui_user_profile_first_name = $3, ui_user_profile_last_name = $4, "+
			"ui_user_profile_extra = $5 WHERE id = $6",
			n.Name, n.UIPreferences.UserProfile.Image, n.UIPreferences.UserProfile.FirstName,
			n.UIPreferences.UserProfile.LastName, n.UIPreferences.UserProfile.Extra, n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
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
// order. An identifier must be a property in the user schema with a type of
// int, uint, uuid, inet, text, or decimal with zero scale. Identifiers cannot
// be repeated.
//
// It returns an errors.UnprocessableError error with code:
//
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

	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		if len(identifiers) > 0 {
			var s []byte
			err := tx.QueryRow(ctx, "SELECT user_schema FROM workspaces WHERE id = $1", n.Workspace).Scan(&s)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return err
			}
			var schema types.Type
			err = json.Unmarshal(s, &schema)
			if err != nil {
				return err
			}
			for _, path := range identifiers {
				p, err := types.PropertyByPath(schema, path)
				if err != nil {
					return errors.Unprocessable(PropertyNotExist, "property %q does not exist in the user schema", path)
				}
				if !canBeIdentifier(p.Type) {
					return errors.Unprocessable(TypeNotAllowed, "property %q has a type %s, which is not allowed for identifiers", path, p.Type)
				}
			}
		}
		_, err := tx.Exec(ctx, "UPDATE workspaces SET resolve_identities_on_batch_import = $1,\n"+
			"identifiers = $2 WHERE id = $3", n.ResolveIdentitiesOnBatchImport, n.Identifiers, n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// UpdateWarehouse updates the mode and settings of the warehouse associated
// with the workspace.
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
func (this *Workspace) UpdateWarehouse(ctx context.Context, mode WarehouseMode, settings []byte, cancelIncompatibleOperations bool) error {
	this.core.mustBeOpen()

	switch mode {
	case Normal, Inspection, Maintenance:
	default:
		return errors.BadRequest("mode %d is not valid", mode)
	}

	ws := this.workspace

	settings, err := this.core.datastore.NormalizeWarehouseSettings(ws.Warehouse.Type, settings)
	if err != nil {
		if err, ok := err.(*meergo.WarehouseSettingsError); ok {
			return errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return err
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
		CancelIncompatibleOperations: cancelIncompatibleOperations,
	}

	err = this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_mode = $1, warehouse_settings = $2 WHERE id = $3",
			n.Mode, string(n.Settings), n.Workspace)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			var warehouseName string
			err = tx.QueryRow(ctx, "SELECT warehouse_type FROM workspaces WHERE id = $1", n.Workspace).Scan(&warehouseName)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return err
			}
			return err
		}
		return tx.Notify(ctx, n)
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

	err := this.core.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_mode = $1 WHERE id = $2", n.Mode, n.Workspace)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			err = tx.QueryVoid(ctx, "SELECT FROM workspaces WHERE id = $1", n.Workspace)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return err
			}
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// User represents a user.
type User struct {
	ID             string         `json:"id"`
	Traits         map[string]any `json:"traits"`
	LastChangeTime time.Time      `json:"lastChangeTime"`
}

// Users returns the users, the user schema, and an estimate of their total
// number without applying first and limit. It returns the users that satisfies
// the filter, if not nil, and in range [first,first+limit] with first >= 0 and
// 0 < limit <= 1000 and only the given properties. properties cannot be empty.
//
// order is the name of the property by which to sort the returned users and
// cannot have type json, array, object, or map; when not provided, the users
// are ordered by their last change time.
//
// orderDesc control whether the returned users should be ordered in descending
// order instead of ascending, which is the default.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore. It returns an errors.UnprocessableError error with code
//
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
//   - OrderNotExist, if order does not exist in schema.
//   - OrderTypeNotSortable, if the type of the order property is not sortable.
//   - PropertyNotExist, if a property does not exist.
func (this *Workspace) Users(ctx context.Context, properties []string, filter *Filter, order string, orderDesc bool, first, limit int) ([]User, types.Type, int, error) {

	this.core.mustBeOpen()

	ws := this.workspace

	// Validate the properties.
	if len(properties) == 0 {
		return nil, types.Type{}, 0, errors.BadRequest("properties is empty")
	}
	propertyByName := map[string]types.Property{}
	for _, p := range ws.UserSchema.Properties() {
		propertyByName[p.Name] = p
	}
	for _, name := range properties {
		if _, ok := propertyByName[name]; !ok {
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
		_, err := validateFilter(filter, ws.UserSchema)
		if err != nil {
			if err, ok := err.(types.PathNotExistError); ok {
				return nil, types.Type{}, 0, errors.Unprocessable(PropertyNotExist, "filter's property %s does not exist", err.Path)
			}
			return nil, types.Type{}, 0, errors.BadRequest("filter is not valid: %w", err)
		}
		where = convertFilterToWhere(filter, ws.UserSchema)
	}

	// Validate the order.
	if order != "" {
		if !types.IsValidPropertyName(order) {
			return nil, types.Type{}, 0, errors.BadRequest("order %q is not a valid property name", order)
		}
		orderProperty, ok := propertyByName[order]
		if !ok {
			return nil, types.Type{}, 0, errors.Unprocessable(OrderNotExist, "order %s does not exist in schema", order)
		}
		switch orderProperty.Type.Kind() {
		case types.JSONKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return nil, types.Type{}, 0, errors.Unprocessable(OrderTypeNotSortable,
				"cannot sort by %s: property has type %s", order, orderProperty.Type)
		}
	} else {
		order = "__last_change_time__"
	}

	// Validate first and limit.
	if first < 0 || first > maxInt32 {
		return nil, types.Type{}, 0, errors.BadRequest("first %d in not valid", first)
	}
	if limit < 1 || limit > 1000 {
		return nil, types.Type{}, 0, errors.BadRequest("limit %d is not valid", limit)
	}

	// Read the users.
	rows, total, err := this.store.Users(ctx, datastore.Query{
		Properties: append([]string{"__id__", "__last_change_time__"}, properties...),
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
			// TODO(marco): log the error in a log specific of the workspace.
			slog.Error("cannot get users from the data warehouse", "workspace", ws.ID, "err", err)
			return nil, types.Type{}, 0, errors.Unavailable("%s", err)
		}
		return nil, types.Type{}, 0, err
	}

	// Create the schema to return, with only the requested properties.
	props := make([]types.Property, len(properties))
	for i, name := range properties {
		props[i] = propertyByName[name]
	}
	schema := types.Object(props)

	users := make([]User, len(rows))
	for i, row := range rows {
		users[i].ID = row["__id__"].(string)
		users[i].Traits = row
		users[i].LastChangeTime = row["__last_change_time__"].(time.Time)
		delete(row, "__id__")
		delete(row, "__last_change_time__")
	}

	return users, schema, total, nil
}

// Warehouse returns type and settings of the data warehouse for the workspace.
func (this *Workspace) Warehouse() (string, json.Value) {
	this.core.mustBeOpen()
	ws := this.workspace
	return ws.Warehouse.Type, json.Value(slices.Clone(ws.Warehouse.Settings))
}

// userIdentities returns the user identities matching the provided where
// condition and an estimate of their total number without applying first and
// limit.
//
// It returns the user identities in range [first,first+limit] with first >= 0
// and 0 < limit <= 1000.
//
// If there are no identities, a nil slice is returned.
//
// It returns an errors.UnprocessableError error with code MaintenanceMode if
// the data warehouse is in maintenance mode.
func (this *Workspace) userIdentities(ctx context.Context, where *state.Where, first, limit int) ([]UserIdentity, int, error) {

	// Retrieve the identities from the data warehouse.
	records, total, err := this.store.UserIdentities(ctx, datastore.Query{
		Properties: []string{
			"__action__",
			"__is_anonymous__",
			"__identity_id__",
			"__connection__",
			"__anonymous_ids__",
			"__last_change_time__",
		},
		Where:   where,
		OrderBy: "__pk__",
		First:   first,
		Limit:   limit,
	})
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, 0, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		return nil, 0, err
	}

	// Create the identities from the records returned by the datastore.
	var identities []UserIdentity

	for _, record := range records {

		// Retrieve the connection.
		connID := record["__connection__"].(int)
		conn, ok := this.core.state.Connection(connID)
		if !ok {
			// The connection for this user identity no longer exists, so skip
			// this identity.
			continue
		}

		// Retrieve the action.
		actionID := record["__action__"].(int)
		_, ok = conn.Action(actionID)
		if !ok {
			// The action for this user identity no longer exists, so skip this
			// identity.
			continue
		}

		// Determine the value for the identity ID.
		identityID := record["__identity_id__"].(string)

		// Determine the anonymous IDs.
		var anonIDs []string
		if ids, ok := record["__anonymous_ids__"].([]any); ok {
			anonIDs = make([]string, len(ids))
			for i := range ids {
				anonIDs[i] = ids[i].(string)
			}
		}

		// In the case of anonymous identities, the anonymous ID is inside the
		// identity ID, so there is the need to populate the anonymous IDs by
		// taking that value, then reset the identity ID.
		if record["__is_anonymous__"].(bool) {
			anonIDs = append(anonIDs, identityID)
			identityID = ""
		}

		// Determine the last change time.
		lastChangeTime := record["__last_change_time__"].(time.Time)

		identities = append(identities, UserIdentity{
			Connection:     connID,
			Action:         actionID,
			ID:             identityID,
			AnonymousIds:   anonIDs,
			LastChangeTime: lastChangeTime,
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
	// non-anonymous users. It can only be provided for source Mobile and Website
	// connections.
	Strategy *Strategy `json:"strategy"`

	// SendingMode is the mode used for sending events. It can only be provided for
	// destination app connections that support it. In this case, it must be one of
	// the sending modes supported by the app.
	SendingMode *SendingMode `json:"sendingMode"`

	// WebsiteHost is the host, in the form "host:port", of a website
	// connection. It must be empty if the connection is not a website. It
	// cannot be longer than 261 runes.
	WebsiteHost string `json:"websiteHost"`

	// LinkedConnections, for connections supporting events, indicate the
	// connections to which events can be sent or received. It is nil if there
	// are no linked connections or if the connection do not support events.
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
		return nil
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

// canBeIdentifier reports whether a property with type t can be used as
// identifier.
func canBeIdentifier(t types.Type) bool {
	switch t.Kind() {
	case types.IntKind,
		types.UintKind,
		types.UUIDKind,
		types.InetKind,
		types.TextKind:
		return true
	case types.DecimalKind:
		return t.Scale() == 0
	default:
		return false
	}
}

// UserIdentity represents a user identity.
type UserIdentity struct {
	// TODO(Gianluca): the Connection field is kept here redundantly (the action
	// is already there) because the UI does not currently have the Action =>
	// Connection mapping available, and it would be very inconvenient to
	// retrieve this information where it is needed. When it will have it in the
	// future, we will remove this field.
	Connection     int       `json:"connection"`
	Action         int       `json:"action"`
	ID             string    `json:"id"`                           // empty string for identities imported from anonymous events.
	AnonymousIds   []string  `json:"anonymousIds,format:emitnull"` // nil for identities not imported from events.
	LastChangeTime time.Time `json:"lastChangeTime"`
}

// filterWorkspaceActions returns from actions, only the actions of the provided
// workspace. It does not change actions.
func filterWorkspaceActions(ws *state.Workspace, actions []int) []int {
	notExists := map[int]struct{}{}
	for _, action := range actions {
		notExists[action] = struct{}{}
	}
	for _, c := range ws.Connections() {
		for _, a := range c.Actions() {
			delete(notExists, a.ID)
		}
	}
	if len(notExists) == 0 {
		return actions
	}
	actions = slices.DeleteFunc(slices.Clone(actions), func(id int) bool {
		_, ok := notExists[id]
		return ok
	})
	return actions
}

// validateUIPreferences validates whether the given UI preferences are valid or
// not, returning an error if they are not.
func validateUIPreferences(preferences UIPreferences) error {
	if n := preferences.UserProfile.Image; n != "" && (len(n) > 1024 || !types.IsValidPropertyPath(n)) {
		return fmt.Errorf("invalid user profile 'image' %q", n)
	}
	if n := preferences.UserProfile.FirstName; n != "" && (len(n) > 1024 || !types.IsValidPropertyPath(n)) {
		return fmt.Errorf("invalid user profile 'firstName' %q", n)
	}
	if n := preferences.UserProfile.LastName; n != "" && (len(n) > 1024 || !types.IsValidPropertyPath(n)) {
		return fmt.Errorf("invalid user profile 'lastName' %q", n)
	}
	if n := preferences.UserProfile.Extra; n != "" && (len(n) > 1024 || !types.IsValidPropertyPath(n)) {
		return fmt.Errorf("invalid user profile 'extra' %q", n)
	}
	return nil
}
