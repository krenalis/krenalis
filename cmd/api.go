//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"encoding/json"
	"net/http"

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/events"
	"github.com/open2b/chichi/types"
)

type api struct {
	*apisServer
}

// AcceptInvitation accepts the invitation with a given invitation token.
//
// Login is not required to call AcceptInvitation.
func (api api) AcceptInvitation(_ http.ResponseWriter, r *http.Request) (any, error) {
	body := struct {
		Name     string
		Password string
	}{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = api.apis.AcceptInvitation(r.Context(), r.PathValue("token"), body.Name, body.Password)
	return nil, err
}

// Connector returns a connector.
func (api api) Connector(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	connector := api.connector(r)
	return api.apis.Connector(r.Context(), connector)
}

// Connectors returns the connectors.
func (api api) Connectors(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	return api.apis.Connectors(r.Context()), nil
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
	body := struct {
		Expressions []apis.ExpressionToBeExtracted
		Schema      types.Type
	}{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	return api.apis.ExpressionsProperties(body.Expressions, body.Schema)
}

// Member returns the current member.
func (api api) Member(_ http.ResponseWriter, r *http.Request) (any, error) {
	member, _, err := api.credentials(r)
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
	organization, email, err := api.apis.MemberInvitation(r.Context(), r.PathValue("token"))
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
	body := struct {
		Data           rawJSON
		InSchema       types.Type
		OutSchema      types.Type
		Transformation apis.DataTransformation
	}{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	data, err := api.apis.TransformData(r.Context(), body.Data, body.InSchema, body.OutSchema, body.Transformation)
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": rawJSON(data)}, nil
}

// TransformationLanguages returns the supported transformation languages.
func (api api) TransformationLanguages(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	languages := api.apis.TransformationLanguages()
	return map[string][]string{"languages": languages}, nil
}

// ValidateExpression validates an expression.
func (api api) ValidateExpression(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	body := struct {
		Expression string
		Properties []types.Property
		Type       types.Type
		Required   bool
		Nullable   bool
	}{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	message := api.apis.ValidateExpression(body.Expression, body.Properties, body.Type, body.Required, body.Nullable)
	return message, nil
}

func (api api) connector(r *http.Request) string {
	return r.PathValue("connector")
}
