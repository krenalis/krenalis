// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http"
	"sort"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// CreateSimulatedAccount creates a simulated account for the current
// workspace.
func (workspace workspace) CreateSimulatedAccount(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	ws, err := workspace.authenticateSimulatedAccountRequest(r)
	if err != nil {
		return nil, err
	}
	body, err := decodeSimulatedAccountBody(r, "name", "userCount", "duplicateRecordPercent", "countries")
	if err != nil {
		return nil, err
	}
	name, err := simulatedAccountStringField(body, "name")
	if err != nil {
		return nil, err
	}
	id, err := ws.CreateSimulatedAccount(r.Context(), name, body["userCount"], body["duplicateRecordPercent"], body["countries"])
	if err != nil {
		return nil, err
	}
	return map[string]string{"id": id}, nil
}

// DeleteSimulatedAccount deletes a simulated account of the current workspace.
func (workspace workspace) DeleteSimulatedAccount(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.authenticateSimulatedAccountRequest(r)
	if err != nil {
		return nil, err
	}
	err = ws.DeleteSimulatedAccount(r.Context(), r.PathValue("id"))
	return nil, err
}

// SimulatedAccount returns a simulated account of the current workspace.
func (workspace workspace) SimulatedAccount(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.authenticateSimulatedAccountRequest(r)
	if err != nil {
		return nil, err
	}
	return ws.SimulatedAccount(r.PathValue("id"))
}

// SimulatedAccounts returns the simulated accounts of the current workspace.
func (workspace workspace) SimulatedAccounts(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.authenticateSimulatedAccountRequest(r)
	if err != nil {
		return nil, err
	}
	return map[string]any{"simulatedAccounts": ws.SimulatedAccounts()}, nil
}

// RenameSimulatedAccount renames a simulated account of the current workspace.
func (workspace workspace) RenameSimulatedAccount(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	ws, err := workspace.authenticateSimulatedAccountRequest(r)
	if err != nil {
		return nil, err
	}
	body, err := decodeSimulatedAccountBody(r, "name")
	if err != nil {
		return nil, err
	}
	name, err := simulatedAccountStringField(body, "name")
	if err != nil {
		return nil, err
	}
	err = ws.RenameSimulatedAccount(r.Context(), r.PathValue("id"), name)
	return nil, err
}

func (workspace workspace) authenticateSimulatedAccountRequest(r *http.Request) (*core.Workspace, error) {
	if _, ok := r.Header["Authorization"]; ok {
		return nil, errors.Unauthorized("simulated account endpoints require Admin session authentication")
	}
	_, ws, _, err := workspace.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errMissingWorkspace
	}
	return ws, nil
}

func decodeSimulatedAccountBody(r *http.Request, allowed ...string) (map[string]json.Value, error) {
	var body map[string]json.Value
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	allowedFields := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		allowedFields[field] = struct{}{}
	}
	unknown := make([]string, 0)
	for field := range body {
		if _, ok := allowedFields[field]; !ok {
			unknown = append(unknown, field)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, errors.BadRequest("field %q cannot be set", unknown[0])
	}
	return body, nil
}

func simulatedAccountStringField(body map[string]json.Value, field string) (string, error) {
	value, ok := body[field]
	if !ok || !value.IsString() {
		return "", errors.BadRequest("%s must be a string", field)
	}
	return value.String(), nil
}
