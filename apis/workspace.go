//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/apis/connectors"
	"github.com/meergo/meergo/apis/datastore"
	"github.com/meergo/meergo/apis/encoding"
	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/events"
	"github.com/meergo/meergo/apis/events/collector"
	"github.com/meergo/meergo/apis/filters"
	"github.com/meergo/meergo/apis/postgres"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/apis/statistics"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/jxskiss/base62"
)

const (
	maxEventsListenedTo = 1000 // maximum number of processed events listened to.
)

// Workspace represents a workspace.
type Workspace struct {
	apis                           *APIs
	organization                   *Organization
	store                          *datastore.Store
	workspace                      *state.Workspace
	ID                             int
	Name                           string
	UserSchema                     types.Type
	UserPrimarySources             map[string]int
	ResolveIdentitiesOnBatchImport bool
	Identifiers                    []string
	WarehouseMode                  WarehouseMode
	PrivacyRegion                  PrivacyRegion
	DisplayedProperties            DisplayedProperties
}

// PrivacyRegion represents a privacy region.
type PrivacyRegion string

const (
	PrivacyRegionNotSpecified PrivacyRegion = ""
	PrivacyRegionEurope       PrivacyRegion = "Europe"
)

// DisplayedProperties represents the displayed properties.
type DisplayedProperties struct {
	Image       string
	FirstName   string
	LastName    string
	Information string
}

// ActionStep represents a step of an action.
type ActionStep int

const (
	ReceivingStep        = ActionStep(statistics.ReceivingStep)
	InputValidationStep  = ActionStep(statistics.InputValidationStep)
	FilteringStep        = ActionStep(statistics.FilteringStep)
	TransformationStep   = ActionStep(statistics.TransformationStep)
	OutputValidationStep = ActionStep(statistics.OutputValidationStep)
	FinalizingStep       = ActionStep(statistics.FinalizingStep)
)

func (step ActionStep) String() string {
	switch step {
	case ReceivingStep:
		return "Receiving"
	case InputValidationStep:
		return "InputValidation"
	case FilteringStep:
		return "Filtering"
	case TransformationStep:
		return "Transformation"
	case OutputValidationStep:
		return "OutputValidation"
	case FinalizingStep:
		return "Finalizing"
	}
	panic("apis: invalid ActionStep")
}

// ParseActionStep parses an action step and returns it. If step is not a valid
// returns 0 and an error.
func ParseActionStep(step string) (ActionStep, error) {
	switch step {
	case "Receiving":
		return ReceivingStep, nil
	case "InputValidation":
		return InputValidationStep, nil
	case "Filtering":
		return FilteringStep, nil
	case "Transformation":
		return TransformationStep, nil
	case "OutputValidation":
		return OutputValidationStep, nil
	case "Finalizing":
		return FinalizingStep, nil
	}
	return 0, fmt.Errorf("step is not valid")
}

// ActionError represents an action error.
type ActionError struct {
	Action       int
	Step         ActionStep
	Count        int
	Message      string
	LastOccurred time.Time
}

// ActionErrors returns the errors for the provided actions within the time
// range [start,end). The end time must not precede the start time, and both
// must be within [statistics.MinTime,statistics.MaxTime]. actions must not be
// empty. Returned errors are limited to [first, first+limit), where first >= 0
// and 0 < limit <= 100.
func (this *Workspace) ActionErrors(ctx context.Context, start, end time.Time, actions []int, step *ActionStep, first, limit int) ([]ActionError, error) {

	this.apis.mustBeOpen()

	start = start.UTC()
	end = end.UTC()

	// Validate start and end.
	if start.Before(statistics.MinTime) {
		return nil, errors.New("start date is too far in the past")
	}
	if end.After(statistics.MaxTime) {
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
	var s *statistics.Step
	if step != nil {
		if *step < ReceivingStep || *step > FinalizingStep {
			return nil, errors.BadRequest("step %d is not valid", *step)
		}
		s = (*statistics.Step)(step)
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

	statisticsErrors, err := this.apis.statistics.Errors(ctx, start, end, actions, s, first, limit)
	if err != nil {
		return nil, err
	}

	errs := make([]ActionError, len(statisticsErrors))
	for i, e := range statisticsErrors {
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

// ActionStats represents action statistics for a time period.
type ActionStats struct {
	Start, End time.Time
	Passed     [][6]int
	Failed     [][6]int
}

// StatsUnit represents the unit of time used for aggregating statistics.
// It can be:
// - Minute: aggregates statistics by minute
// - Hour: aggregates statistics by hour
// - Day: aggregates statistics by day
type StatsUnit int

const (
	Minute = StatsUnit(statistics.Minute)
	Hour   = StatsUnit(statistics.Hour)
	Day    = StatsUnit(statistics.Day)
)

// ActionStatsPerDate returns statistics aggregated by day for the time interval
// between the specified start and end dates. The years in the dates must be
// no earlier than 1970 and no later than the next year. The day of the start
// date must be at least one day before the day of the end date. actions
// specifies the actions for which statistics are returned and must not be
// empty.
func (this *Workspace) ActionStatsPerDate(ctx context.Context, start, end time.Time, actions []int) (ActionStats, error) {

	this.apis.mustBeOpen()

	start = start.UTC().Truncate(24 * time.Hour)
	end = end.UTC().Truncate(24 * time.Hour)

	// Validate start and end.
	if start.Before(statistics.MinTime) {
		return ActionStats{}, errors.BadRequest("start date is too far in the past")
	}
	if end.After(statistics.MaxTime) {
		return ActionStats{}, errors.BadRequest("end date date is too far in the future")
	}
	if !end.After(start) {
		return ActionStats{}, errors.BadRequest("day of the end date must be after the day of the start date")
	}

	// Validate actions.
	if len(actions) == 0 {
		return ActionStats{}, errors.BadRequest("actions if non-nil, cannot be empty")
	}
	for _, action := range actions {
		if action < 1 || action > maxInt32 {
			return ActionStats{}, errors.BadRequest("action %d is not valid", action)
		}
	}

	actions = filterWorkspaceActions(this.workspace, actions)
	if len(actions) == 0 {
		number := int(end.Sub(start).Hours() / 24)
		return ActionStats{
			Start:  start,
			End:    end,
			Passed: make([][6]int, number),
			Failed: make([][6]int, number),
		}, nil
	}

	stats, err := this.apis.statistics.StatsPerDate(ctx, start, end, actions)
	if err != nil {
		return ActionStats{}, err
	}

	return ActionStats{
		Start:  stats.Start,
		End:    stats.End,
		Passed: stats.Passed,
		Failed: stats.Failed,
	}, nil
}

// ActionStatsPerTimeUnit returns statistics for the specified number of
// minutes, hours, or days based on the unit, which can be Minute, Hour, or Day,
// up to the current time. number must be in the following ranges: [1,60] for
// minutes, [1,48] for hours, and [1,30] for days. actions specifies the actions
// for which statistics are returned and must not be empty.
func (this *Workspace) ActionStatsPerTimeUnit(ctx context.Context, number int, unit StatsUnit, actions []int) (ActionStats, error) {

	this.apis.mustBeOpen()

	// Validate number and unit.
	switch unit {
	case Minute:
		if number < 1 || number > 60 {
			return ActionStats{}, errors.BadRequest("minutes must be in range [1,60]")
		}
	case Hour:
		if number < 1 || number > 48 {
			return ActionStats{}, errors.BadRequest("hours must be in range [1,48]")
		}
	case Day:
		if number < 1 || number > 30 {
			return ActionStats{}, errors.BadRequest("days must be in range [1,30]")
		}
	}

	// Validate actions.
	if len(actions) == 0 {
		return ActionStats{}, errors.BadRequest("actions if non-nil, cannot be empty")
	}
	for _, action := range actions {
		if action < 1 || action > maxInt32 {
			return ActionStats{}, errors.BadRequest("action %d is not valid", action)
		}
	}

	actions = filterWorkspaceActions(this.workspace, actions)
	if len(actions) == 0 {
		return ActionStats{
			Passed: make([][6]int, number),
			Failed: make([][6]int, number),
		}, nil
	}

	stats, err := this.apis.statistics.StatsPerTimeUnit(ctx, number, time.Duration(unit), actions)
	if err != nil {
		return ActionStats{}, err
	}

	return ActionStats{
		Start:  stats.Start,
		End:    stats.End,
		Passed: stats.Passed,
		Failed: stats.Failed,
	}, nil
}

// AddConnection adds a new connection. oAuthToken is an OAuth token returned by
// the OAuthToken method and must be empty if the connector does not support
// OAuth authentication.
//
// It returns an errors.UnprocessableError error with code
//
//   - ConnectorNotExist, if the connector does not exist.
//   - LinkedConnectionNotExist, if a linked connection does not exist.
//   - InvalidUIValues, if the user-entered values are not valid.
func (this *Workspace) AddConnection(ctx context.Context, connection ConnectionToAdd, oAuthToken string) (int, error) {

	this.apis.mustBeOpen()

	if connection.Role != Source && connection.Role != Destination {
		return 0, errors.BadRequest("role %d is not valid", int(connection.Role))
	}
	if connection.Connector == "" {
		return 0, errors.BadRequest("connector name is empty")
	}
	if containsNUL(connection.Name) || utf8.RuneCountInString(connection.Name) > 100 {
		return 0, errors.BadRequest("name %q is not valid", connection.Name)
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

	c, ok := this.apis.state.Connector(connection.Connector)
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

	n := state.AddConnection{
		Workspace:         this.workspace.ID,
		Name:              connection.Name,
		Role:              state.Role(connection.Role),
		Enabled:           connection.Enabled,
		Connector:         connection.Connector,
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

	// Validate OAuth.
	if (oAuthToken == "") != (c.OAuth == nil) {
		if oAuthToken == "" {
			return 0, errors.BadRequest("OAuth is required by connector %s", n.Connector)
		}
		return 0, errors.BadRequest("connector %s does not support OAuth", n.Connector)
	}

	// Set the OAuth account. It can be an existing account or an account that needs to be created.
	if oAuthToken != "" {
		data, err := base62.DecodeString(oAuthToken)
		if err != nil {
			return 0, errors.BadRequest("OAuth is not valid")
		}
		var account authorizedOAuthAccount
		err = json.Unmarshal(data, &account)
		if err != nil {
			return 0, errors.BadRequest("OAuth is not valid")
		}
		if account.Workspace != this.workspace.ID || account.Connector != c.Name {
			return 0, errors.BadRequest("OAuth is not valid")
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

	// Validate the UI values.
	if c.HasUI {
		values := connection.UIValues
		if values == nil {
			values = json.RawMessage("{}")
		}
		var clientSecret string
		if c.OAuth != nil {
			clientSecret = c.OAuth.ClientSecret
		}
		conf := &connectors.ConnectorConfig{
			Role:   n.Role,
			Region: state.PrivacyRegion(this.PrivacyRegion),
		}
		conf.OAuth.Account = n.Account.Code
		conf.OAuth.ClientSecret = clientSecret
		conf.OAuth.AccessToken = n.Account.AccessToken
		n.Settings, err = this.apis.connectors.UpdatedSettings(ctx, c, conf, values)
		if err != nil {
			switch err.(type) {
			case *connectors.InvalidUIValuesError:
				err = errors.Unprocessable(InvalidUIValues, "%s", err)
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

	// Generate a write key.
	switch c.Type {
	case state.Mobile, state.Server, state.Website:
		n.Key, err = generateWriteKey()
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

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
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
				if postgres.IsForeignKeyViolation(err) && postgres.ErrConstraintName(err) == "accounts_workspace_fkey" {
					err = errors.Unprocessable(WorkspaceNotExist, "workspace %d does not exist", n.Workspace)
				}
				return err
			}
		}
		// Insert the connection.
		_, err = tx.Exec(ctx, "INSERT INTO connections "+
			"(id, workspace, name, type, role, enabled, connector, account,"+
			" strategy, sending_mode, website_host, linked_connections, settings)"+
			" VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)",
			n.ID, n.Workspace, n.Name, c.Type, n.Role, n.Enabled, n.Connector, n.Account.ID,
			n.Strategy, n.SendingMode, n.WebsiteHost, n.LinkedConnections, string(n.Settings))
		if err != nil {
			if postgres.IsForeignKeyViolation(err) && postgres.ErrConstraintName(err) == "connections_workspace_fkey" {
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
		if n.Key != "" {
			// Insert the server key.
			_, err = tx.Exec(ctx, "INSERT INTO connections_keys (connection, value, creation_time) VALUES ($1, $2, $3)",
				n.ID, n.Key, time.Now().UTC())
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

// AddCollectedEventListener adds an event listener to the workspace that
// listens to collected events and returns its identifier.
//
// size specifies the maximum number of observed events to be returned by a
// subsequent call to the ListenedEvents method and must be in the range
// [1, 1000].
//
// sources contains the identifiers of the sources, whether they are mobile,
// server, or website connections. If sources is non-nil, only events
// originating from these sources will be observed.
//
// onlyValid determines whether only valid events should be observed.
//
// It returns an errors.UnprocessableError error with code:
//
//   - ConnectionNotExist, if a source connection does not exist.
//   - TooManyListeners, if there are already too many listeners.
func (this *Workspace) AddCollectedEventListener(size int, sources []int, onlyValid bool) (string, error) {
	this.apis.mustBeOpen()
	if size < 1 || size > maxEventsListenedTo {
		return "", errors.BadRequest("size %d is not valid", size)
	}
	err := this.validateEventListenerSources(sources)
	if err != nil {
		return "", errors.BadRequest("%s", err)
	}
	id, err := this.apis.events.observer.AddCollectedListener(size, sources, onlyValid)
	if err != nil {
		if err == collector.ErrTooManyListeners {
			err = errors.Unprocessable(TooManyListeners, "there are already %d listeners", collector.MaxEventListeners)
		}
		return "", err
	}
	return id, nil
}

// AddEnrichedEventListener adds an event listener to the workspace that listens
// to enriched events and returns its identifier.
//
// size specifies the maximum number of observed events to be returned by a
// subsequent call to the ListenedEvents method and must be in the range
// [1, 1000].
//
// sources contains the identifiers of the sources, whether they are mobile,
// server, or website connections. If sources is non-nil, only events
// originating from these sources will be observed.
//
// If hasUserTraits is true, only events with user traits will be observed,
// such as "identify" events and events with a non-nil traits in context.
//
// If filter is non-nil, only events that satisfy the filter will be observed.
//
// It returns an errors.UnprocessableError with code:
//
//   - ConnectionNotExist, if a source connection does not exist.
//   - TooManyListeners, if there are already too many listeners.
func (this *Workspace) AddEnrichedEventListener(size int, sources []int, hasUserTraits bool, filter *filters.Filter) (string, error) {
	this.apis.mustBeOpen()
	if size < 1 || size > maxEventsListenedTo {
		return "", errors.BadRequest("size %d is not valid", size)
	}
	err := this.validateEventListenerSources(sources)
	if err != nil {
		return "", errors.BadRequest("%s", err)
	}
	if filter != nil {
		_, err := filters.Validate(filter, events.Schema)
		if err != nil {
			return "", errors.BadRequest("filter is not valid: %w", err)
		}
	}
	id, err := this.apis.events.observer.AddEnrichedListener(size, sources, hasUserTraits, filter)
	if err != nil {
		if err == collector.ErrTooManyListeners {
			err = errors.Unprocessable(TooManyListeners, "there are already %d listeners", collector.MaxEventListeners)
		}
		return "", err
	}
	return id, nil
}

// ChangeIdentityResolutionSettings changes the settings of the Identity
// Resolution of the workspace.
//
// runOnBatchImport determines whether the identities should be resolved
// automatically every time a batch import is completed.
//
// identifiers specify the Identity Resolution identifiers in the specified
// order. An identifier must be a property in the user schema with a type of
// Int, Uint, UUID, Inet, Text, or Decimal with zero scale. Identifiers cannot
// be repeated.
//
// It returns an errors.UnprocessableError error with code:
//
//   - PropertyNotExist, if an identifier path does not exist in the user
//     schema.
//   - TypeNotAllowed, if an identifier path's type, as defined in the user
//     schema, is not allowed for identifiers.
func (this *Workspace) ChangeIdentityResolutionSettings(ctx context.Context, runOnBatchImport bool, identifiers []string) error {

	this.apis.mustBeOpen()

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
	n := state.SetIdentityResolutionSettings{
		Workspace:                      ws.ID,
		ResolveIdentitiesOnBatchImport: runOnBatchImport,
		Identifiers:                    identifiers,
	}

	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
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

// ChangeWarehouseMode changes the mode of the data warehouse for the workspace.
//
// If cancelIncompatibleOperations is true, the operations currently in progress
// on the warehouse that are incompatible with mode are cancelled.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore.
func (this *Workspace) ChangeWarehouseMode(ctx context.Context, mode WarehouseMode, cancelIncompatibleOperations bool) error {
	this.apis.mustBeOpen()

	switch mode {
	case Normal, Inspection, Maintenance:
	default:
		return errors.BadRequest("mode %d is not valid", mode)
	}

	ws := this.workspace

	n := state.SetWarehouseMode{
		Workspace:                    ws.ID,
		Mode:                         state.WarehouseMode(mode),
		CancelIncompatibleOperations: cancelIncompatibleOperations,
	}

	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_mode = $1 WHERE id = $2 AND warehouse_type IS NOT NULL",
			n.Mode, n.Workspace)
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

// LastIdentityResolution returns information about the last Identity
// Resolution of the workspace.
//
// In particular:
//
//   - if the Identity Resolution has been started and completed, returns its
//     start time and end time;
//   - if it is in progress, returns its start time and nil for the end time;
//   - if no Identity Resolution has ever been executed, returns nil and nil.
//
// It returns an errors.UnprocessableError error with code:
//
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
func (this *Workspace) LastIdentityResolution(ctx context.Context) (startTime, endTime *time.Time, err error) {
	this.apis.mustBeOpen()
	startTime, endTime, err = this.store.LastIdentityResolution(ctx)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return nil, nil, errors.Unprocessable(DataWarehouseFailed, "data warehouse failed: %s", err.Err)
		}
		if err == datastore.ErrMaintenanceMode {
			return nil, nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		return nil, nil, err
	}
	return startTime, endTime, nil
}

// ChangeWarehouseSettings changes the mode and the settings of the data
// warehouse for the workspace.
//
// If cancelIncompatibleOperations is true, the operations currently in progress
// on the warehouse that are incompatible with mode are cancelled.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore, and it returns an errors.UnprocessableError error with code
//
//   - InvalidWarehouseSettings, if the settings are not valid.
//   - InvalidWarehouseType, if the workspace is connected to a data warehouse
//     of a different type,
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *Workspace) ChangeWarehouseSettings(ctx context.Context, typ WarehouseType, mode WarehouseMode, settings []byte, cancelIncompatibleOperations bool) error {
	this.apis.mustBeOpen()

	switch mode {
	case Normal, Inspection, Maintenance:
	default:
		return errors.BadRequest("mode %d is not valid", mode)
	}

	ws := this.workspace
	if ws.Warehouse.Type != state.WarehouseType(typ) {
		return errors.Unprocessable(InvalidWarehouseType, "workspace %d is connected with a %s data warehouse, not %s", ws.ID, ws.Warehouse.Type, typ)
	}

	settings, err := this.apis.datastore.NormalizeWarehouseSettings(ws.Warehouse.Type, settings)
	if err != nil {
		if err, ok := err.(*datastore.SettingsError); ok {
			return errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
		}
		return err
	}

	err = this.apis.datastore.PingWarehouse(ctx, ws.Warehouse.Type, settings)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return errors.Unprocessable(DataWarehouseFailed, "cannot connect to the data warehouse: %w", err.Err)
		}
		return err
	}

	n := state.SetWarehouse{
		Workspace: ws.ID,
		Warehouse: state.Warehouse{
			Type:     ws.Warehouse.Type,
			Mode:     state.WarehouseMode(mode),
			Settings: settings,
		},
		CancelIncompatibleOperations: cancelIncompatibleOperations,
	}

	err = this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "UPDATE workspaces SET warehouse_mode = $1, warehouse_settings = $2 WHERE id = $3 AND warehouse_type = $4",
			n.Warehouse.Mode, string(n.Warehouse.Settings), n.Workspace, n.Warehouse.Type)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			var warehouseType *state.WarehouseType
			err = tx.QueryRow(ctx, "SELECT warehouse_type FROM workspaces WHERE id = $1", n.Workspace).Scan(&warehouseType)
			if err != nil {
				if err == sql.ErrNoRows {
					err = errors.NotFound("workspace %d does not exist", n.Workspace)
				}
				return err
			}
			return errors.Unprocessable(InvalidWarehouseType, "workspace %d is connected with a %s data warehouse, not %s",
				ws.ID, *warehouseType, n.Warehouse.Type)
		}
		return tx.Notify(ctx, n)
	})

	return err
}

// Connection returns the connection with identifier id of the workspace.
//
// If the connection does not exist, it returns an errors.NotFoundError error.
func (this *Workspace) Connection(id int) (*Connection, error) {
	this.apis.mustBeOpen()
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("connection identifier %d is not valid", id)
	}
	c, ok := this.workspace.Connection(id)
	if !ok {
		return nil, errors.NotFound("connection %d does not exist", id)
	}
	conn := c.Connector()

	connection := Connection{
		apis:              this.apis,
		store:             this.store,
		connection:        c,
		ID:                c.ID,
		Name:              c.Name,
		Type:              ConnectorType(conn.Type),
		Role:              Role(c.Role),
		Enabled:           c.Enabled,
		Connector:         conn.Name,
		Strategy:          (*Strategy)(c.Strategy),
		SendingMode:       (*SendingMode)(c.SendingMode),
		WebsiteHost:       c.WebsiteHost,
		LinkedConnections: slices.Clone(c.LinkedConnections),
		HasUI:             conn.HasUI,
		ActionsCount:      len(c.Actions()),
		Health:            Health(c.Health),
	}

	// Set the actions.
	actions := c.Actions()
	a := make([]Action, len(actions))
	connection.Actions = &a
	for i, a := range actions {
		(*connection.Actions)[i].fromState(this.apis, this.store, a)
	}
	return &connection, nil
}

// Connections returns the connections of the workspace.
func (this *Workspace) Connections() []*Connection {
	this.apis.mustBeOpen()
	connections := this.workspace.Connections()
	infos := make([]*Connection, len(connections))
	for i, c := range connections {
		conn := c.Connector()
		connection := Connection{
			apis:              this.apis,
			store:             this.store,
			connection:        c,
			ID:                c.ID,
			Name:              c.Name,
			Type:              ConnectorType(conn.Type),
			Role:              Role(c.Role),
			Enabled:           c.Enabled,
			Connector:         conn.Name,
			Strategy:          (*Strategy)(c.Strategy),
			SendingMode:       (*SendingMode)(c.SendingMode),
			WebsiteHost:       c.WebsiteHost,
			LinkedConnections: slices.Clone(c.LinkedConnections),
			HasUI:             conn.HasUI,
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

// Delete deletes the workspace with all its connections.
//
// If the workspace does not exist anymore, it returns an errors.NotFound error.
func (this *Workspace) Delete(ctx context.Context) error {
	this.apis.mustBeOpen()
	n := state.DeleteWorkspace{
		ID: this.workspace.ID,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		result, err := tx.Exec(ctx, "DELETE FROM workspaces WHERE id = $1 AND warehouse_type IS NULL", n.ID)
		if err != nil {
			return err
		}
		if result.RowsAffected() == 0 {
			var warehouseType state.WarehouseType
			err := tx.QueryRow(ctx, "SELECT warehouse_type FROM workspaces WHERE id = $1", n.ID).Scan(&warehouseType)
			if err != nil {
				if err == sql.ErrNoRows {
					return errors.NotFound("workspace %d does not exist", n.ID)
				}
				return err
			}
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// IdentifiersSchema returns the properties of the "users" schema that can be
// used as identifiers in the Identity Resolution.
// If none of the properties can be an identifier, this method returns the
// invalid schema.
func (this *Workspace) IdentifiersSchema() types.Type {
	this.apis.mustBeOpen()
	return types.SubsetFunc(this.workspace.UserSchema, func(p types.Property) bool {
		return canBeIdentifier(p.Type)
	})
}

// ObservedEvent represents an observed event.
type ObservedEvent struct {

	// Source, if not zero, it is the source mobile, server or website
	// connection for which the event was sent.
	Source int

	// Header is the message header. It is nil if a validation error occurred
	// processing the entire message.
	Header *ObservedEventHeader

	// Data contains the data, encoded in JSON, of a single event in the message,
	// if Header is not nil, or the data of the entire message, if Header is nil.
	Data []byte

	// Err, if not empty, is a validation error occurred processing the message.
	// It refers to a single event, if Header is not nil, or to the entire message
	// if Header is nil.
	Err string
}

type ObservedEventHeader struct {
	ReceivedAt time.Time   `json:"receivedAt"`
	RemoteAddr string      `json:"remoteAddr"`
	Method     string      `json:"method"`
	Proto      string      `json:"proto"`
	URL        string      `json:"url"`
	Headers    http.Header `json:"headers"`
}

// ListenedEvents returns the events listened to the specified listener and the
// number of discarded events. It returns collected events if the listener
// listens to collected events, and returns enriched events if the listener
// listens to enriched events.
//
// If the listener does not exist, the function returns an errors.NotFoundError.
func (this *Workspace) ListenedEvents(listener string) ([]ObservedEvent, int, error) {
	this.apis.mustBeOpen()
	observedEvents, discarded, err := this.apis.events.observer.Events(listener)
	if err != nil {
		if err == collector.ErrEventListenerNotFound {
			return nil, 0, errors.NotFound("event listener %q does not exist", listener)
		}
		return nil, 0, err
	}
	evs := make([]ObservedEvent, len(observedEvents))
	for i := range len(evs) {
		ov := observedEvents[i]
		var header *ObservedEventHeader
		if ov.Header != nil {
			header = &ObservedEventHeader{
				ReceivedAt: ov.Header.ReceivedAt,
				RemoteAddr: ov.Header.RemoteAddr,
				Method:     ov.Header.Method,
				Proto:      ov.Header.Proto,
				URL:        ov.Header.URL,
				Headers:    maps.Clone(ov.Header.Headers),
			}
		}
		evs[i] = ObservedEvent{
			Source: ov.Source,
			Header: header,
			Data:   slices.Clone(ov.Data),
			Err:    ov.Err,
		}
	}
	return evs, discarded, nil
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

// OAuthToken returns an OAuth token, given an OAuth authorization code and the
// redirection URI used to obtain that code, that can be used to add a new
// connection to the workspace for the specified connector.
//
// It returns an errors.NotFound error if the workspace does not exist anymore.
// It returns an errors.UnprocessableError error with code ConnectorNotExist if
// the connector does not exist.
func (this *Workspace) OAuthToken(ctx context.Context, code, redirectionURI string, connector string) (string, error) {

	this.apis.mustBeOpen()

	if code == "" {
		return "", errors.BadRequest("authorization code is empty")
	}
	if connector == "" {
		return "", errors.BadRequest("connector name is empty")
	}

	c, ok := this.apis.state.Connector(connector)
	if !ok {
		return "", errors.Unprocessable(ConnectorNotExist, "connector %q does not exist", connector)
	}
	if c.OAuth == nil {
		return "", errors.BadRequest("connector %s does not support OAuth", connector)
	}

	region := state.PrivacyRegion(this.PrivacyRegion)
	auth, err := this.apis.connectors.GrantAuthorization(ctx, c, code, redirectionURI, region)
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

// RemoveEventListener removes the given event listener from the workspace. It
// does nothing if the listener does not exist.
func (this *Workspace) RemoveEventListener(listener string) {
	this.apis.mustBeOpen()
	this.apis.events.observer.RemoveListener(listener)
}

// Rename renames the workspace with the given new name.
// name must be between 1 and 100 runes long.
//
// It returns an errors.NotFoundError error if the workspace does not exist
// anymore.
func (this *Workspace) Rename(ctx context.Context, name string) error {
	this.apis.mustBeOpen()
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return errors.BadRequest("name %q is not valid", name)
	}
	if name == this.workspace.Name {
		return nil
	}
	n := state.RenameWorkspace{
		Workspace: this.workspace.ID,
		Name:      name,
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
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
//
// It returns an errors.UnprocessableError error with code DataWarehouseFailed,
// if an error occurred during the repairing of the data warehouse.
func (this *Workspace) RepairWarehouse(ctx context.Context) error {
	this.apis.mustBeOpen()
	err := this.store.Repair(ctx)
	if err != nil {
		if err, ok := (err).(*datastore.DataWarehouseError); ok {
			return errors.Unprocessable(DataWarehouseFailed, "data warehouse failed: %s", err.Err)
		}
		return err
	}
	return nil
}

// ResolveIdentities resolves the identities of the workspace by creating and
// starting an Identity Resolution operation.
//
// It returns an errors.UnprocessableError error with code:
//
//   - AlterSchemaInProgress, if an alter schema operation is currently in
//     in progresso on the data warehouse.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - InspectionMode, if the data warehouse is in inspection mode.
//   - IdentityResolutionInProgress, if an Identity Resolution is already in
//     progress on the warehouse.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
func (this *Workspace) ResolveIdentities(ctx context.Context) error {
	this.apis.mustBeOpen()
	err := this.store.ResolveIdentities(ctx)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			return errors.Unprocessable(DataWarehouseFailed, "an error occurred on the data warehouse: %w", err.Err)
		}
		if err == datastore.ErrAlterSchemaInProgress {
			return errors.Unprocessable(AlterSchemaInProgress, "an alter schema operation is in progress on the data warehouse")
		}
		if err == datastore.ErrInspectionMode {
			return errors.Unprocessable(InspectionMode, "data warehouse is in inspection mode")
		}
		if err == datastore.ErrMaintenanceMode {
			return errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if err == datastore.ErrIdentityResolutionInProgress {
			return errors.Unprocessable(IdentityResolutionInProgress, "the Identity Resolution is already running on the data warehouse")
		}
		return err
	}
	return nil
}

// ServeUI serves the user interface for the given connector, with the given
// role. event is the event and values are the user-entered values in JSON
// format. oAuth is the OAuth token returned by the (*Workspace).OAuth method,
// it is required if the connector requires OAuth.
//
// It returns an errors.UnprocessableError error with code:
//
//   - ConnectorNotExist, if the connector does not exist.
//   - EventNotExist, if the event does not exist.
//   - InvalidUIValues, if the user-entered values are not valid.
func (this *Workspace) ServeUI(ctx context.Context, event string, values []byte, connector string, role Role, oAuth string) ([]byte, error) {

	this.apis.mustBeOpen()

	if connector == "" {
		return nil, errors.BadRequest("connector name is empty")
	}
	if role != Source && role != Destination {
		return nil, errors.BadRequest("role %d is not valid", role)
	}
	c, ok := this.apis.state.Connector(connector)
	if !ok {
		return nil, errors.Unprocessable(ConnectorNotExist, "connector %q does not exist", connector)
	}

	if !c.HasUI {
		return nil, errors.BadRequest("connector %s does not have a UI", connector)
	}

	if (oAuth == "") != (c.OAuth == nil) {
		if oAuth == "" {
			return nil, errors.BadRequest("OAuth is required by connector %s", c.Name)
		}
		return nil, errors.BadRequest("connector %s does not support OAuth", c.Name)
	}

	// Decode oAuth.
	var a authorizedOAuthAccount
	if oAuth != "" {
		data, err := base62.DecodeString(oAuth)
		if err != nil {
			return nil, errors.BadRequest("oAuth is not valid")
		}
		err = json.Unmarshal(data, &a)
		if err != nil {
			return nil, errors.BadRequest("oAuth is not valid")
		}
	}

	var clientSecret string
	if oAuth != "" {
		clientSecret = c.OAuth.ClientSecret
	}
	conf := &connectors.ConnectorConfig{
		Role:   state.Role(role),
		Region: this.workspace.PrivacyRegion,
	}
	conf.OAuth.Account = a.Code
	conf.OAuth.ClientSecret = clientSecret
	conf.OAuth.AccessToken = a.AccessToken

	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	ui, err := this.apis.connectors.ServeConnectorUI(ctx, c, conf, event, values)
	if err != nil {
		if err == connectors.ErrUIEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for connector %s", event, c.Name)
		} else {
			switch err.(type) {
			case *connectors.InvalidUIValuesError:
				err = errors.Unprocessable(InvalidUIValues, "%s", err)
			case *connectors.UnavailableError:
				err = errors.Unavailable("%s", err)
			}
		}
		return nil, err
	}

	return ui, nil
}

// Set sets the name, the privacy region and the displayed properties of the
// workspace. name must be between 1 and 100 runes long. displayedProperties
// must contain valid displayed property names. A valid displayed property name
// is an empty string, or alternatively a valid property name between 1 and 100
// runes long.
func (this *Workspace) Set(ctx context.Context, name string, region PrivacyRegion, displayedProperties DisplayedProperties) error {
	this.apis.mustBeOpen()
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return errors.BadRequest("name %q is not valid", name)
	}
	switch region {
	case PrivacyRegionNotSpecified,
		PrivacyRegionEurope:
	default:
		return errors.BadRequest("invalid privacy region %q", string(region))
	}
	if !isValidDisplayedPropertyName(displayedProperties.Image) {
		return errors.BadRequest("invalid displayed image %q", displayedProperties.Image)
	}
	if !isValidDisplayedPropertyName(displayedProperties.FirstName) {
		return errors.BadRequest("invalid displayed first name %q", displayedProperties.FirstName)
	}
	if !isValidDisplayedPropertyName(displayedProperties.LastName) {
		return errors.BadRequest("invalid displayed last name %q", displayedProperties.LastName)
	}
	if !isValidDisplayedPropertyName(displayedProperties.Information) {
		return errors.BadRequest("invalid displayed information %q", displayedProperties.Information)
	}
	ws := this.workspace
	n := state.SetWorkspace{
		Workspace:           ws.ID,
		Name:                name,
		PrivacyRegion:       state.PrivacyRegion(region),
		DisplayedProperties: state.DisplayedProperties(displayedProperties),
	}
	err := this.apis.state.Transaction(ctx, func(tx *state.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE workspaces SET name = $1, privacy_region = $2, displayed_image = $3, "+
			"displayed_first_name = $4, displayed_last_name = $5, displayed_information = $6 "+
			"WHERE id = $7",
			n.Name, n.PrivacyRegion, n.DisplayedProperties.Image, n.DisplayedProperties.FirstName,
			n.DisplayedProperties.LastName, n.DisplayedProperties.Information, n.Workspace)
		if err != nil {
			return err
		}
		return tx.Notify(ctx, n)
	})
	return err
}

// PingWarehouse pings the data warehouse with the given settings, verifying
// that the settings are valid and a connection can be established.
//
// It returns an errors.UnprocessableError error with code
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - InvalidWarehouseSettings, if the settings are not valid.
func (this *Workspace) PingWarehouse(ctx context.Context, typ WarehouseType, settings []byte) error {
	this.apis.mustBeOpen()
	err := this.apis.datastore.PingWarehouse(ctx, state.WarehouseType(typ), settings)
	switch err := err.(type) {
	case *datastore.SettingsError:
		return errors.Unprocessable(InvalidWarehouseSettings, "data warehouse settings are not valid: %w", err.Err)
	case *datastore.DataWarehouseError:
		return errors.Unprocessable(DataWarehouseFailed, "cannot connect to the data warehouse: %w", err.Err)
	}
	return err
}

// User returns the user with identifier id of the workspace. If the user does
// not exist, the error is deferred until methods of *User are called.
func (this *Workspace) User(id uuid.UUID) (*User, error) {
	this.apis.mustBeOpen()
	return &User{
		apis:      this.apis,
		workspace: this.workspace,
		store:     this.store,
		id:        id,
	}, nil
}

// Users returns the users, the user schema of the workspace, and an estimate of
// their count without applying first and limit. It returns the users that
// satisfies the filter, if not nil, and in range [first,first+limit] with first
// >= 0 and 0 < limit <= 1000 and only the given properties. properties cannot
// be empty and cannot contain meta properties.
//
// order is the property by which to sort the returned users and cannot have
// type JSON, Array, Object, or Map; when not provided, the users are ordered by
// their last change time.
//
// orderDesc control whether the returned users should be ordered in descending
// order instead of ascending, which is the default.
//
// It returns an errors.NotFoundError error, if the workspace does not exist
// anymore.
// It returns an errors.UnprocessableError error with code
//
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
//   - OrderNotExist, if order does not exist in schema.
//   - OrderTypeNotSortable, if the type of the order property is not sortable.
//   - PropertyNotExist, if a property does not exist.
func (this *Workspace) Users(ctx context.Context, properties []string, filter *filters.Filter, order string, orderDesc bool, first, limit int) ([]byte, types.Type, int, error) {

	this.apis.mustBeOpen()

	ws := this.workspace

	// Validate the properties.
	if len(properties) == 0 {
		return nil, types.Type{}, 0, errors.BadRequest("properties is empty")
	}
	for _, p := range properties {
		if isMetaProperty(p) {
			return nil, types.Type{}, 0, errors.BadRequest("properties cannot contain meta properties")
		}
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
	var where *datastore.Where
	if filter != nil {
		_, err := filters.Validate(filter, ws.UserSchema)
		if err != nil {
			if err, ok := err.(types.PathNotExistError); ok {
				return nil, types.Type{}, 0, errors.Unprocessable(PropertyNotExist, "filter's property %s does not exist", err.Path)
			}
			return nil, types.Type{}, 0, errors.BadRequest("filter is not valid: %w", err)
		}
		where = &datastore.Where{
			Logical:    datastore.WhereLogical(filter.Logical),
			Conditions: make([]datastore.WhereCondition, len(filter.Conditions)),
		}
		for i, condition := range filter.Conditions {
			where.Conditions[i] = (datastore.WhereCondition)(condition)
		}
	}
	if order != "" {
		if !types.IsValidPropertyName(order) {
			return nil, types.Type{}, 0, errors.BadRequest("order %q is not a valid property name", order)
		}
		if isMetaProperty(order) {
			return nil, types.Type{}, 0, errors.BadRequest("order %q cannot be a meta property", order)
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
	if first < 0 || first > maxInt32 {
		return nil, types.Type{}, 0, errors.BadRequest("first %d in not valid", first)
	}
	if limit < 1 || limit > 1000 {
		return nil, types.Type{}, 0, errors.BadRequest("limit %d is not valid", limit)
	}

	// Read the users.
	users, count, err := this.store.Users(ctx, datastore.Query{
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
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			slog.Error("cannot get users from the data warehouse", "workspace", ws.ID, "err", err)
			return nil, types.Type{}, 0, errors.Unprocessable(DataWarehouseFailed, "data warehouse connection is failed: %w", err.Err)
		}
		return nil, types.Type{}, 0, err
	}

	// Create the schema to return, with only the requested properties.
	requestedProperties := make([]types.Property, len(properties)+2)
	requestedProperties[0] = types.Property{Name: "__id__", Type: types.UUID()}
	requestedProperties[1] = types.Property{Name: "__last_change_time__", Type: types.DateTime()}
	for i, name := range properties {
		requestedProperties[i+2] = propertyByName[name]
	}
	schema := types.Object(requestedProperties)

	// Marshal the users into a JSON array like this:
	//
	//  [
	//  	{
	//  		"id": "f88893fb-fc04-4868-8ab7-041c225c79b4",
	//          "lastChangeTime": "2000-01-03T12:00:00Z",
	//  		"properties": {
	//  			"email": "a@example.com"
	//  		}
	//  	},
	//  	{
	//  		"id": "e0bb8a23-d1ee-4fe4-8264-5892499d21e5",
	//          "lastChangeTime": "2000-01-03T12:00:00Z",
	//  		"properties": {
	//  			"email": "c@example.com"
	//  		}
	//  	}
	//  ]
	var marshaledUsers bytes.Buffer
	marshaledUsers.WriteRune('[')
	for i, user := range users {
		id := user["__id__"].(string)
		delete(user, "__id__")
		lastChangeTime := user["__last_change_time__"].(time.Time)
		delete(user, "__last_change_time__")
		marshaledUser, err := encoding.Marshal(schema, user)
		if err != nil {
			return nil, types.Type{}, 0, err
		}
		if i > 0 {
			marshaledUsers.WriteByte(',')
		}
		marshaledUsers.WriteString(`{"id":"`)
		marshaledUsers.WriteString(id)
		marshaledUsers.WriteString(`","lastChangeTime":`)
		err = json.NewEncoder(&marshaledUsers).Encode(lastChangeTime)
		if err != nil {
			return nil, types.Type{}, 0, err
		}
		marshaledUsers.WriteString(`,"properties":`)
		marshaledUsers.Write(marshaledUser)
		marshaledUsers.WriteRune('}')
	}
	marshaledUsers.WriteRune(']')

	return marshaledUsers.Bytes(), schema, count, nil
}

// WarehouseSettings returns the type and settings of the data warehouse for the
// workspace.
func (this *Workspace) WarehouseSettings() (WarehouseType, []byte) {
	this.apis.mustBeOpen()
	ws := this.workspace
	return WarehouseType(ws.Warehouse.Type), slices.Clone(ws.Warehouse.Settings)
}

// userIdentities returns the user identities matching the provided where
// condition and an estimate of their count without applying first and limit.
//
// It returns the user identities in range [first,first+limit] with first >= 0
// and 0 < limit <= 1000.
//
// If there are no identities, a nil slice is returned.
//
// It returns an errors.UnprocessableError error with code
//
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
//   - MaintenanceMode, if the data warehouse is in maintenance mode.
func (this *Workspace) userIdentities(ctx context.Context, where *datastore.Where, first, limit int) ([]UserIdentity, int, error) {

	// Retrieve the identities from the data warehouse.
	records, count, err := this.store.UserIdentities(ctx, datastore.Query{
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
		conn, ok := this.apis.state.Connection(connID)
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

	// Since the count is an estimate, being counted separately from the actual
	// number of identities returned, ensure to not return a value lower than
	// the actually returned number of identities.
	count = max(len(identities), count)

	return identities, count, nil
}

// validateEventListenerSources validates the sources from which events are
// listened to.
func (this *Workspace) validateEventListenerSources(sources []int) error {
	if sources == nil {
		return nil
	}
	if len(sources) == 0 {
		return fmt.Errorf("sources, if not nil, cannot be empty")
	}
	for i, s := range sources {
		if s < 1 || s > math.MaxInt32 {
			return fmt.Errorf("source %d is not valid", s)
		}
		c, ok := this.workspace.Connection(s)
		if !ok {
			return errors.Unprocessable(ConnectionNotExist, "connection %d does not exist", sources)
		}
		switch c.Connector().Type {
		case state.Mobile, state.Server, state.Website:
		default:
			return errors.BadRequest("connection %d is not a mobile, server or website", sources)
		}
		if c.Role != state.Source {
			return errors.BadRequest("connection %d is not a source", sources)
		}
		for _, s2 := range sources[i+1:] {
			if s == s2 {
				return fmt.Errorf("sources contains duplicated values")
			}
		}
	}
	return nil
}

// ConnectionToAdd represents a connection to add to a workspace.
type ConnectionToAdd struct {

	// Name is the name of the connection. It cannot be longer than 100 runes.
	// If empty, the connection name will be the name of its connector.
	Name string

	// Role is the role.
	Role Role

	// Enable reports whether the connection is enabled or disabled when added.
	Enabled bool

	// Connector is the name of the connector.
	Connector string

	// Strategy is the strategy that determines how to merge anonymous and
	// non-anonymous users. It must be nil for destination connections and
	// non-event source connections.
	Strategy *Strategy

	// SendingMode is the mode used for sending events. It must be nil for
	// source connections and connections that does not support events.
	SendingMode *SendingMode

	// WebsiteHost is the host, in the form "host:port", of a website
	// connection. It must be empty if the connection is not a website. It
	// cannot be longer than 261 runes.
	WebsiteHost string

	// LinkedConnections, for connections supporting events, indicate the
	// connections to which events can be sent or received. It is nil if there
	// are no linked connections or if the connection do not support events.
	LinkedConnections []int

	// UIValues represents the user-entered values of the connector user interface
	// in JSON format.
	// It must be nil if the connector does not have a user interface.
	UIValues json.RawMessage
}

// WarehouseType represents a data warehouse type.
type WarehouseType int

const (
	PostgreSQL WarehouseType = iota + 1
	Snowflake
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if typ is not a valid WarehouseType value.
func (typ WarehouseType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + typ.String() + `"`), nil
}

// String returns the string representation of typ.
// It panics if typ is not a valid WarehouseType value.
func (typ WarehouseType) String() string {
	switch typ {
	case PostgreSQL:
		return "PostgreSQL"
	case Snowflake:
		return "Snowflake"
	}
	panic("invalid store type")
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (typ *WarehouseType) UnmarshalJSON(data []byte) error {
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
		return fmt.Errorf("json: cannot scan a %T value into an WarehouseType value", v)
	}
	var t WarehouseType
	switch s {
	case "PostgreSQL":
		t = PostgreSQL
	case "Snowflake":
		t = Snowflake
	default:
		return fmt.Errorf("json: invalid WarehouseType: %s", s)
	}
	*typ = t
	return nil
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

// isValidDisplayedPropertyName reports whether property is a valid displayed
// property name. A valid displayed property name is an empty string, or
// alternatively a valid property name between 1 and 100 runes long.
func isValidDisplayedPropertyName(property string) bool {
	if property != "" && (utf8.RuneCountInString(property) > 100 || !types.IsValidPropertyName(property)) {
		return false
	}
	return true
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
	ID             string    `json:"id"`           // empty string for identities imported from anonymous events.
	AnonymousIds   []string  `json:"anonymousIds"` // nil for identities not imported from events.
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
