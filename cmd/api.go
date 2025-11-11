// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
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
// Authentication is not required to call AcceptInvitation.
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
// Authentication is not required to call ChangeMemberPasswordByToken.
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
	if _, _, err := api.authenticateRequest(r); err != nil {
		return nil, err
	}
	return api.core.Connector(api.code(r))
}

// ConnectorDocumentation returns the documentation of a connector.
func (api api) ConnectorDocumentation(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.authenticateRequest(r); err != nil {
		return nil, err
	}
	return api.core.ConnectorDocumentation(api.code(r))
}

// Connectors returns the connectors.
func (api api) Connectors(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.authenticateRequest(r); err != nil {
		return nil, err
	}
	return map[string]any{"connectors": api.core.Connectors()}, nil
}

// EventSchema returns the event schema.
func (api api) EventSchema(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.authenticateRequest(r); err != nil {
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
	if _, _, _, err := api.authenticateAdminRequest(r); err != nil {
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
	org, _, memberID, err := api.authenticateAdminRequest(r)
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
// Authentication is not required to call MemberInvitation.
func (api api) MemberInvitation(_ http.ResponseWriter, r *http.Request) (any, error) {
	organization, email, err := api.core.MemberInvitation(r.Context(), r.PathValue("token"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"email": email, "organization": organization}, nil
}

type publicMetadata struct {
	InstallationID                  string   `json:"installationID"`
	ExternalURL                     string   `json:"externalURL"`
	ExternalEventURL                string   `json:"externalEventURL"`
	ExternalAssetsURLs              []string `json:"externalAssetsURLs"`
	JavaScriptSDKURL                string   `json:"javascriptSDKURL"`
	MemberEmailVerificationRequired bool     `json:"memberEmailVerificationRequired"`
	CanSendMemberPasswordReset      bool     `json:"canSendMemberPasswordReset"`
	TelemetryLevel                  string   `json:"telemetryLevel"`
}

// PublicMetadata returns public information about the server installation:
//
//   - installationID: installation ID
//   - externalURL: canonical external URL - https://example.com/
//   - externalEventURL: external event URL - https://example.com/api/v1/events
//   - externalAssetsURLs: external assets URLs.
//   - canSendMemberPasswordReset: can send the reset password email?
//   - telemetryLevel: telemetry level - none, errors, stats, or all
//
// Authentication is not required to call PublicMetadata.
func (api api) PublicMetadata(_ http.ResponseWriter, r *http.Request) (any, error) {
	metadata := publicMetadata{
		InstallationID:                  api.core.InstallationID(),
		ExternalURL:                     api.externalURL,
		ExternalEventURL:                api.externalEventURL,
		ExternalAssetsURLs:              api.externalAssetsURLs,
		JavaScriptSDKURL:                api.javaScriptSDKURL,
		MemberEmailVerificationRequired: api.memberEmailVerificationRequired,
		CanSendMemberPasswordReset:      api.core.CanSendMemberPasswordReset(),
		TelemetryLevel:                  string(api.sentryTelemetry.level),
	}
	return metadata, nil
}

// SendMemberPasswordReset sends a reset password email.
//
// Authentication is not required to call SendMemberPasswordReset.
func (api api) SendMemberPasswordReset(_ http.ResponseWriter, r *http.Request) (any, error) {
	var body struct {
		Email string `json:"email"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	organizations, _ := api.core.Organizations(core.SortByName, 0, 1)
	if len(organizations) == 0 {
		return nil, errors.New("there are no organizations")
	}
	org := organizations[0]
	emailTemplate := strings.ReplaceAll(resetPasswordEmail, "${externalURL}", html.EscapeString(api.externalURL))
	err = org.SendMemberPasswordReset(r.Context(), body.Email, emailTemplate)
	return nil, err
}

// ValidateMemberPasswordResetToken validates the given password reset token.
//
// Authentication is not required to call ValidateMemberPasswordResetToken.
func (api api) ValidateMemberPasswordResetToken(_ http.ResponseWriter, r *http.Request) (any, error) {
	err := api.core.ValidateMemberPasswordResetToken(r.Context(), r.PathValue("token"))
	return nil, err
}

// TransformData transforms data using a mapping or a function transformation
// and returns the transformed data.
func (api api) TransformData(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, _, err := api.authenticateAdminRequest(r); err != nil {
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
	if _, _, err := api.authenticateRequest(r); err != nil {
		return nil, err
	}
	languages := api.core.TransformationLanguages()
	return map[string][]string{"languages": languages}, nil
}

// ValidateExpression validates an expression.
func (api api) ValidateExpression(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, _, err := api.authenticateAdminRequest(r); err != nil {
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

// WarehouseDrivers returns the supported data warehouse drivers.
func (api api) WarehouseDrivers(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.authenticateRequest(r); err != nil {
		return nil, err
	}
	return map[string]any{"drivers": api.core.WarehouseDrivers()}, nil
}

func (api api) code(r *http.Request) string {
	return r.PathValue("code")
}
