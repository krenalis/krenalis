//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"net/http"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

type api struct {
	*apisServer
}

// AcceptInvitation accepts the invitation with a given invitation token.
//
// Login is not required to call AcceptInvitation.
func (api api) AcceptInvitation(_ http.ResponseWriter, r *http.Request) (any, error) {
	var body struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = api.core.AcceptInvitation(r.Context(), r.PathValue("token"), body.Name, body.Password)
	return nil, err
}

// Connector returns a connector.
func (api api) Connector(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	return api.core.Connector(api.name(r))
}

// Connectors returns the connectors.
func (api api) Connectors(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	return map[string]any{"connectors": api.core.Connectors()}, nil
}

// EventSchema returns the events schema.
func (api api) EventSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	return events.Schema, nil
}

// ExpressionsProperties returns all the unique properties contained inside a
// list of expressions.
func (api api) ExpressionsProperties(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	var body struct {
		Expressions []core.ExpressionToBeExtracted `json:"expressions"`
		Schema      types.Type                     `json:"schema"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	return api.core.ExpressionsProperties(body.Expressions, body.Schema)
}

// Member returns the current member.
func (api api) Member(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, member, err := api.memberCredentials(r)
	if err != nil {
		return nil, err
	}
	return member, nil
}

// MemberInvitation returns the organization's name and email of the member
// invited with a given invitation token.
//
// Login is not required to call MemberInvitation.
func (api api) MemberInvitation(_ http.ResponseWriter, r *http.Request) (any, error) {
	organization, email, err := api.core.MemberInvitation(r.Context(), r.PathValue("token"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"email": email, "organization": organization}, nil
}

// TransformData transforms data using a mapping or a function transformation
// and returns the transformed data.
func (api api) TransformData(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	var body struct {
		Data           json.Value              `json:"data"`
		InSchema       types.Type              `json:"inSchema"`
		OutSchema      types.Type              `json:"outSchema"`
		Transformation core.DataTransformation `json:"transformation"`
		Purpose        core.Purpose            `json:"purpose"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	data, err := api.core.TransformData(r.Context(), body.Data, body.InSchema, body.OutSchema, body.Transformation, body.Purpose)
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": data}, nil
}

// TransformationLanguages returns the supported transformation languages.
func (api api) TransformationLanguages(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	languages := api.core.TransformationLanguages()
	return map[string][]string{"languages": languages}, nil
}

// ValidateExpression validates an expression.
func (api api) ValidateExpression(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	var body struct {
		Expression string           `json:"expression"`
		Properties []types.Property `json:"properties"`
		Type       types.Type       `json:"type"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	return api.core.ValidateExpression(body.Expression, body.Properties, body.Type)
}

// WarehouseTypes returns the supported data warehouse types.
func (api api) WarehouseTypes(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	return map[string]any{"types": api.core.WarehouseTypes()}, nil
}

func (api api) name(r *http.Request) string {
	return r.PathValue("name")
}
