//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

type action struct {
	*apisServer
}

// Delete deletes an action.
func (action action) Delete(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
	if err != nil {
		return nil, err
	}
	err = a.Delete(r.Context())
	return nil, err
}

// Execute executes an action.
func (action action) Execute(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Reload bool `json:"reload"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	id, err := a.Execute(r.Context(), body.Reload)
	if err != nil {
		return nil, err
	}
	return map[string]int{"id": id}, nil
}

// ServeUI serves the UI of an action.
func (action action) ServeUI(w http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Event          string     `json:"event"`
		FormatSettings json.Value `json:"formatSettings"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	ui, err := a.ServeUI(r.Context(), body.Event, body.FormatSettings)
	if err != nil {
		return nil, err
	}
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(ui)
	return nil, nil
}

// SetSchedulePeriod sets the schedule period of an action.
func (action action) SetSchedulePeriod(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Period *core.SchedulePeriod `json:"period"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = a.SetSchedulePeriod(r.Context(), body.Period)
	return nil, err
}

// SetStatus sets the status of an action.
func (action action) SetStatus(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = a.SetStatus(r.Context(), body.Enabled)
	return nil, err
}

// Update updates an action.
func (action action) Update(_ http.ResponseWriter, r *http.Request) (any, error) {
	a, err := action.id(r)
	if err != nil {
		return nil, err
	}
	var body core.ActionToSet
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = a.Update(r.Context(), body)
	return nil, err
}

func (action action) id(r *http.Request) (*core.Action, error) {
	ws, err := workspace{action.apisServer}.workspace(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("id")
	if v[0] == '+' {
		return nil, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return nil, errors.NotFound("")
	}
	return ws.Action(id)
}

// toAPIAction converts a core.Action value into the format required by the API.
func toAPIAction(action *core.Action) any {
	type apiAction struct {
		ID             int                `json:"id"`
		Name           string             `json:"name"`
		Connector      string             `json:"connector"`
		ConnectorType  core.ConnectorType `json:"connectorType"`
		Connection     int                `json:"connection"`
		ConnectionRole core.Role          `json:"connectionRole"`
		Target         core.Target        `json:"target"`
		Enabled        bool               `json:"enabled"`
	}
	a := apiAction{
		ID:             action.ID,
		Name:           action.Name,
		Connector:      action.Connector,
		ConnectorType:  action.ConnectorType,
		Connection:     action.Connection,
		ConnectionRole: action.ConnectionRole,
		Target:         action.Target,
		Enabled:        action.Enabled,
	}
	if a.ConnectionRole == core.Source {
		if a.Target == core.Users {
			switch a.ConnectorType {
			case core.App:
				return struct {
					apiAction
					Filter         *core.Filter         `json:"filter"`
					InSchema       types.Type           `json:"inSchema"`
					OutSchema      types.Type           `json:"outSchema"`
					Transformation core.Transformation  `json:"transformation"`
					Running        bool                 `json:"running"`
					ScheduleStart  *int                 `json:"scheduleStart"`
					SchedulePeriod *core.SchedulePeriod `json:"schedulePeriod"`
				}{
					apiAction:      a,
					Filter:         action.Filter,
					InSchema:       action.InSchema,
					OutSchema:      action.OutSchema,
					Transformation: *action.Transformation,
					Running:        action.Running,
					ScheduleStart:  action.ScheduleStart,
					SchedulePeriod: action.SchedulePeriod,
				}
			case core.Database:
				return struct {
					apiAction
					Query                  string               `json:"query"`
					IdentityProperty       string               `json:"identityProperty"`
					LastChangeTimeProperty *string              `json:"lastChangeTimeProperty"`
					LastChangeTimeFormat   *string              `json:"lastChangeTimeFormat"`
					InSchema               types.Type           `json:"inSchema"`
					OutSchema              types.Type           `json:"outSchema"`
					Transformation         core.Transformation  `json:"transformation"`
					Running                bool                 `json:"running"`
					ScheduleStart          *int                 `json:"scheduleStart"`
					SchedulePeriod         *core.SchedulePeriod `json:"schedulePeriod"`
				}{
					apiAction:            a,
					Query:                *action.Query,
					IdentityProperty:     *action.IdentityProperty,
					LastChangeTimeFormat: action.LastChangeTimeFormat,
					InSchema:             action.InSchema,
					OutSchema:            action.OutSchema,
					Transformation:       *action.Transformation,
					Running:              action.Running,
					ScheduleStart:        action.ScheduleStart,
					SchedulePeriod:       action.SchedulePeriod,
				}
			case core.FileStorage:
				return struct {
					apiAction
					Format         string               `json:"format"`
					Path           string               `json:"path"`
					Sheet          *string              `json:"sheet"`
					Compression    core.Compression     `json:"compression"`
					Filter         *core.Filter         `json:"filter"`
					InSchema       types.Type           `json:"inSchema"`
					OutSchema      types.Type           `json:"outSchema"`
					Transformation core.Transformation  `json:"transformation"`
					Running        bool                 `json:"running"`
					ScheduleStart  *int                 `json:"scheduleStart"`
					SchedulePeriod *core.SchedulePeriod `json:"schedulePeriod"`
				}{
					apiAction:      a,
					Format:         action.Format,
					Path:           *action.Path,
					Sheet:          action.Sheet,
					Compression:    action.Compression,
					InSchema:       action.InSchema,
					OutSchema:      action.OutSchema,
					Transformation: *action.Transformation,
					Running:        action.Running,
					ScheduleStart:  action.ScheduleStart,
					SchedulePeriod: action.SchedulePeriod,
				}
			case core.Mobile, core.Server, core.Website:
				return struct {
					apiAction
					Filter         *core.Filter         `json:"filter"`
					InSchema       types.Type           `json:"inSchema"`
					OutSchema      types.Type           `json:"outSchema"`
					Transformation *core.Transformation `json:"transformation"`
				}{
					apiAction:      a,
					InSchema:       action.InSchema,
					OutSchema:      action.OutSchema,
					Transformation: action.Transformation,
				}
			}
		}
		if a.Target == core.Events {
			return struct {
				apiAction
				Filter   *core.Filter `json:"filter"`
				InSchema types.Type   `json:"inSchema"`
			}{
				apiAction: a,
				Filter:    action.Filter,
				InSchema:  action.InSchema,
			}
		}
	}
	if a.ConnectionRole == core.Destination {
		if a.Target == core.Users {
			switch a.ConnectorType {
			case core.App:
				return struct {
					apiAction
					Filter             *core.Filter         `json:"filter"`
					ExportMode         core.ExportMode      `json:"exportMode"`
					Matching           core.Matching        `json:"matching"`
					ExportOnDuplicates bool                 `json:"exportOnDuplicates"`
					InSchema           types.Type           `json:"inSchema"`
					OutSchema          types.Type           `json:"outSchema"`
					Transformation     core.Transformation  `json:"transformation"`
					Running            bool                 `json:"running"`
					ScheduleStart      *int                 `json:"scheduleStart"`
					SchedulePeriod     *core.SchedulePeriod `json:"schedulePeriod"`
				}{
					apiAction:          a,
					Filter:             action.Filter,
					ExportMode:         *action.ExportMode,
					Matching:           *action.Matching,
					ExportOnDuplicates: *action.ExportOnDuplicates,
					InSchema:           action.InSchema,
					OutSchema:          action.OutSchema,
					Transformation:     *action.Transformation,
					Running:            action.Running,
					ScheduleStart:      action.ScheduleStart,
					SchedulePeriod:     action.SchedulePeriod,
				}
			case core.Database:
				return struct {
					apiAction
					Filter         *core.Filter         `json:"filter"`
					TableName      string               `json:"tableName"`
					TableKey       string               `json:"tableKey"`
					InSchema       types.Type           `json:"inSchema"`
					OutSchema      types.Type           `json:"outSchema"`
					Transformation core.Transformation  `json:"transformation"`
					Running        bool                 `json:"running"`
					ScheduleStart  *int                 `json:"scheduleStart"`
					SchedulePeriod *core.SchedulePeriod `json:"schedulePeriod"`
				}{
					apiAction:      a,
					Filter:         action.Filter,
					TableName:      *action.TableName,
					TableKey:       *action.TableKey,
					InSchema:       action.InSchema,
					OutSchema:      action.OutSchema,
					Transformation: *action.Transformation,
					Running:        action.Running,
					ScheduleStart:  action.ScheduleStart,
					SchedulePeriod: action.SchedulePeriod,
				}
			case core.FileStorage:
				return struct {
					apiAction
					Format         string               `json:"format"`
					Path           string               `json:"path"`
					Sheet          *string              `json:"sheet"`
					Compression    core.Compression     `json:"compression"`
					OrderBy        string               `json:"orderBy"`
					Filter         *core.Filter         `json:"filter"`
					InSchema       types.Type           `json:"inSchema"`
					Running        bool                 `json:"running"`
					ScheduleStart  *int                 `json:"scheduleStart"`
					SchedulePeriod *core.SchedulePeriod `json:"schedulePeriod"`
				}{
					apiAction:      a,
					Format:         action.Format,
					Path:           *action.Path,
					Sheet:          action.Sheet,
					Compression:    action.Compression,
					OrderBy:        *action.OrderBy,
					Filter:         action.Filter,
					InSchema:       action.InSchema,
					Running:        action.Running,
					ScheduleStart:  action.ScheduleStart,
					SchedulePeriod: action.SchedulePeriod,
				}
			}
		}
		if a.Target == core.Events {
			return struct {
				apiAction
				EventType      string               `json:"eventType"`
				Filter         *core.Filter         `json:"filter"`
				InSchema       types.Type           `json:"inSchema"`
				OutSchema      types.Type           `json:"outSchema"`
				Transformation *core.Transformation `json:"transformation"`
			}{
				apiAction:      a,
				EventType:      *action.EventType,
				Filter:         action.Filter,
				InSchema:       action.InSchema,
				OutSchema:      action.OutSchema,
				Transformation: action.Transformation,
			}
		}
	}
	panic(fmt.Sprintf("unexpected role: %s, target: %s, type: %s", a.ConnectionRole, a.Target, a.ConnectorType))
}
