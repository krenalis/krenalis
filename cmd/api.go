// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	_ "embed"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

type api struct {
	*apisServer
}

// AcceptInvitation accepts the invitation with a given invitation token.
//
// Authentication is not required to call AcceptInvitation.
func (api api) AcceptInvitation(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
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
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
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
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
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

// Index returns the index.
func (api api) Index(w http.ResponseWriter, r *http.Request) (any, error) {
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive, nosnippet, notranslate, noimageindex")
	accept := strings.ToLower(r.Header.Get("Accept"))
	wantsHTML := accept == "" || strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*") && !strings.Contains(accept, "application/json")
	if wantsHTML {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fi, err := static.Open("static/api_index.html")
		if err != nil {
			return nil, errors.New("embedded file 'static/api_index.html' not found in executable")
		}
		_, _ = io.Copy(w, fi)
		_ = fi.Close()
		return nil, nil
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"api":"Krenalis API","version":"v1","documentation":"https://www.krenalis.com/docs/ref/api"}`))
	return nil, nil
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
	InstallationID             string   `json:"installationID"`
	ExternalURL                string   `json:"externalURL"`
	ExternalEventURL           string   `json:"externalEventURL"`
	ExternalAssetsURLs         []string `json:"externalAssetsURLs"`
	PotentialConnectorsURL     *string  `json:"potentialConnectorsURL"`
	JavaScriptSDKURL           string   `json:"javascriptSDKURL"`
	InviteMembersViaEmail      bool     `json:"inviteMembersViaEmail"`
	CanSendMemberPasswordReset bool     `json:"canSendMemberPasswordReset"`
	TelemetryLevel             string   `json:"telemetryLevel"`
}

// PublicMetadata returns public information about the server installation:
//
//   - installationID: installation ID
//   - externalURL: canonical external URL - https://example.com/
//   - externalEventURL: external event URL - https://example.com/v1/events
//   - externalAssetsURLs: external assets URLs.
//   - potentialConnectorsURL: URL of JSON with potential connectors, or empty string.
//   - javaScriptSDKURL: URL of the JavaScript SDK - https://example.com/krenalis.min.js
//   - inviteMembersViaEmail: should new members be added by sending invitation emails??
//   - canSendMemberPasswordReset: can send the reset password email?
//   - telemetryLevel: telemetry level - none, errors, stats, or all
//
// Authentication is not required to call PublicMetadata.
func (api api) PublicMetadata(_ http.ResponseWriter, r *http.Request) (any, error) {
	metadata := publicMetadata{
		InstallationID:             api.core.InstallationID(),
		ExternalURL:                api.externalURL,
		ExternalEventURL:           api.externalEventURL,
		ExternalAssetsURLs:         api.externalAssetsURLs,
		JavaScriptSDKURL:           api.javaScriptSDKURL,
		InviteMembersViaEmail:      api.inviteMembersViaEmail,
		CanSendMemberPasswordReset: api.core.CanSendMemberPasswordReset(),
		TelemetryLevel:             string(api.sentryTelemetry.level),
	}
	if api.potentialConnectorsURL != "" {
		metadata.PotentialConnectorsURL = &api.potentialConnectorsURL
	}
	return metadata, nil
}

// SendMemberPasswordReset sends a reset password email.
//
// Authentication is not required to call SendMemberPasswordReset.
func (api api) SendMemberPasswordReset(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
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
	resetPasswordEmail, err := static.ReadFile("static/reset_password_email.html")
	if err != nil {
		return nil, errors.New("embedded file 'static/reset_password_email.html' not found in executable")
	}
	emailTemplate := strings.ReplaceAll(string(resetPasswordEmail), "${externalURL}", html.EscapeString(api.externalURL))
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
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
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
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
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

// WarehousePlatforms returns the supported data warehouse platforms.
func (api api) WarehousePlatforms(_ http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, err := api.authenticateRequest(r); err != nil {
		return nil, err
	}
	return map[string]any{"platforms": api.core.WarehousePlatforms()}, nil
}

func (api api) code(r *http.Request) string {
	return r.PathValue("code")
}

// splitQueryParameters expands comma-separated query parameter values.
// Each input string may contain one or more comma-delimited entries.
// All entries are split, trimmed, and returned individually.
// Empty or whitespace-only entries are discarded.
//
// For example, []string{"1,2,3"} becomes []string{"1", "2", "3"}.
//
// If no valid entries exist, it returns nil.
func splitQueryParameters(values []string) []string {
	var properties []string
	for _, v := range values {
		for p := range strings.SplitSeq(v, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if properties == nil {
				properties = make([]string, 0, len(values))
			}
			properties = append(properties, p)
		}
	}
	return properties
}
