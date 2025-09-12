//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
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

// ChangeMemberPasswordByToken changes the password of a member with the given
// reset password token.
//
// Login is not required to call ChangeMemberPasswordByToken.
func (api api) ChangeMemberPasswordByToken(_ http.ResponseWriter, r *http.Request) (any, error) {
	var body struct {
		Password string `json:"password"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = api.core.ChangeMemberPasswordByToken(r.Context(), r.PathValue("token"), body.Password)
	return nil, err
}

// Connector returns a connector.
func (api api) Connector(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	return api.core.Connector(api.name(r))
}

// ConnectorDocumentation returns the documentation of a connector.
func (api api) ConnectorDocumentation(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	return api.core.ConnectorDocumentation(api.name(r))
}

// Connectors returns the connectors.
func (api api) Connectors(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	return map[string]any{"connectors": api.core.Connectors()}, nil
}

// InstallationID returns the installation ID.
func (api api) InstallationID(w http.ResponseWriter, r *http.Request) (any, error) {
	return api.core.InstallationID(), nil
}

// EventSchema returns the event schema.
func (api api) EventSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.credentials(r); err != nil {
		return nil, err
	}
	return core.EventSchema(), nil
}

// EventsSettings returns the events settings.
func (api api) EventsSettings(w http.ResponseWriter, r *http.Request) (any, error) {
	// Removes the headers that were set earlier, as ServeEvents handles the response fully.
	w.Header().Del("Cache-Control")
	w.Header().Del("Pragma")
	w.Header().Del("Expires")
	api.core.ServeEvents(w, r)
	return nil, nil
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

// ExternalEventURL returns the external URL that receives the events, for
// example "https://my.meergo.example.com/api/v1/events".
func (api api) ExternalEventURL(w http.ResponseWriter, r *http.Request) (any, error) {
	return api.externalEventURL, nil
}

// Member returns the current member.
func (api api) Member(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, memberID, err := api.memberCredentials(r)
	if err != nil {
		return nil, err
	}
	// Get the member.
	member, err := org.Member(r.Context(), memberID)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			err = errInvalidSessionCookie
		}
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

// SendMemberPasswordReset sends a reset password email.
//
// Login is not required to call SendMemberPasswordReset.
func (api api) SendMemberPasswordReset(_ http.ResponseWriter, r *http.Request) (any, error) {
	var body struct {
		Email string `json:"email"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	organization, err := api.core.Organization(r.Context(), 1)
	if err != nil {
		return nil, fmt.Errorf("cannot read organization: %s", err)
	}
	emailTemplate := strings.ReplaceAll(resetPasswordEmail, "${externalURL}", html.EscapeString(api.externalURL))
	err = organization.SendMemberPasswordReset(r.Context(), body.Email, emailTemplate)
	return nil, err
}

// CanSendMemberPasswordReset returns whether it is possible to send the reset
// password email.
func (api api) CanSendMemberPasswordReset(_ http.ResponseWriter, r *http.Request) (any, error) {
	return api.core.CanSendMemberPasswordReset(), nil
}

// SentryTelemetryLevel returns the Sentry telemetry level set. Possible return
// values are: "none", "errors", "stats" or "all".
func (api api) SentryTelemetryLevel(w http.ResponseWriter, r *http.Request) (any, error) {
	return string(api.sentryTelemetry.level), nil
}

// SkipMemberEmailVerification returns whether to skip the verification
// of the email during the creation of a new member.
func (api api) SkipMemberEmailVerification(w http.ResponseWriter, r *http.Request) (any, error) {
	return api.skipMemberEmailVerification, nil
}

// ValidateMemberPasswordResetToken validates the given password reset token.
//
// Login is not required to call ValidateMemberPasswordResetToken.
func (api api) ValidateMemberPasswordResetToken(_ http.ResponseWriter, r *http.Request) (any, error) {
	err := api.core.ValidateMemberPasswordResetToken(r.Context(), r.PathValue("token"))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// JavaScriptSDKURL returns the URL that serves the JavaScript SDK.
func (api api) JavaScriptSDKURL(w http.ResponseWriter, r *http.Request) (any, error) {
	return api.javaScriptSDKURL, nil
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
