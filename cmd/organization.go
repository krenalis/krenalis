// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

type organization struct {
	*apisServer
}

// AddMember adds a new member of an organization.
//
// It returns an errors.UnprocessableError with code WorkOSEnabled when WorkOS
// authentication is configured.
//
// If the ability to add new members without requiring email invitation has not
// been enabled, it returns an errors.UnprocessableError error with code
// EmailInvitationRequired.
func (organization organization) AddMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	if organization.workOS != nil {
		return nil, errors.Unprocessable(core.BuiltInAuthenticationDisabled, "members cannot be added because WorkOS authentication is enabled")
	}
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	org, _, _, err := organization.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	if organization.inviteMembersViaEmail {
		return nil, errors.Unprocessable(core.EmailInvitationRequired, "email invitation is required")
	}
	var body struct {
		MemberToSet struct {
			Name     string `json:"name"`
			Image    []byte `json:"image"`
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"memberToSet"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	memberToSet := core.MemberToSet{
		Name:     body.MemberToSet.Name,
		Email:    body.MemberToSet.Email,
		Password: body.MemberToSet.Password,
	}
	if body.MemberToSet.Image != nil {
		fileType := http.DetectContentType(body.MemberToSet.Image)
		avatar := &core.Avatar{
			Image:    body.MemberToSet.Image,
			MimeType: fileType,
		}
		memberToSet.Avatar = avatar
	}
	_, err = org.AddMember(r.Context(), memberToSet)
	return nil, err
}

// AccessKeys returns the access keys of an organization.
func (organization organization) AccessKeys(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, _, err := organization.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	keys, err := org.AccessKeys(r.Context())
	if err != nil {
		return nil, err
	}
	for i, key := range keys {
		prefix := "api_"
		if key.Type == core.AccessKeyTypeMCP {
			prefix = "mcp_"
		}
		keys[i].Token = prefix + key.Token
	}
	return map[string][]*core.AccessKey{"keys": keys}, nil
}

// CreateAccessKey creates a new access key for an organization.
func (organization organization) CreateAccessKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	org, _, _, err := organization.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name      string              `json:"name"`
		Workspace *string             `json:"workspace"`
		Type      *core.AccessKeyType `json:"type"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	var workspace string
	if body.Workspace != nil {
		if !core.IsValidID(*body.Workspace) {
			return nil, errors.BadRequest("workspace %q is not a valid workspace identifier", *body.Workspace)
		}
		workspace = *body.Workspace
	}
	if body.Type == nil {
		return nil, errors.BadRequest("type is required and cannot be null")
	}
	id, token, err := org.CreateAccessKey(r.Context(), body.Name, workspace, *body.Type)
	if err != nil {
		return nil, err
	}
	switch *body.Type {
	case core.AccessKeyTypeAPI:
		token = "api_" + token
	case core.AccessKeyTypeMCP:
		token = "mcp_" + token
	}
	return map[string]any{"id": id, "token": token}, nil
}

// CreateWorkspace creates a workspace for the organization.
func (organization organization) CreateWorkspace(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	org, err := organization.admitOrganizationRequest(r, x1)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name          string             `json:"name"`
		ProfileSchema types.Type         `json:"profileSchema"`
		Warehouse     core.Warehouse     `json:"warehouse"`
		UIPreferences core.UIPreferences `json:"uiPreferences"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	id, err := org.CreateWorkspace(r.Context(), body.Name, body.ProfileSchema, body.Warehouse, body.UIPreferences)
	if err != nil {
		if err2, ok := err.(*errors.UnprocessableError); ok && err2.Code == core.OrganizationNotExist {
			return nil, errors.Unauthorized("API key in the Authorization header of the request does not exist")
		}
		return nil, err
	}
	return map[string]string{"id": id}, nil
}

// Delete deletes the organization with the given identifier.
//
// Authentication is performed using the organizations API key.
func (organization organization) Delete(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := organization.authenticateOrganizationsRequest(r); err != nil {
		return nil, err
	}
	org, err := organization.core.Organization(r.PathValue("id"))
	if err != nil {
		return nil, err
	}
	err = org.Delete(r.Context())
	return nil, err
}

// DeleteAccessKey deletes an access key of an organization.
func (organization organization) DeleteAccessKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, _, err := organization.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	err = org.DeleteAccessKey(r.Context(), r.PathValue("key"))
	return nil, err
}

// DeleteMember deletes a member of an organization.
//
// It returns an errors.UnprocessableError with code WorkOSEnabled when WorkOS
// authentication is configured.
func (organization organization) DeleteMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	if organization.workOS != nil {
		return nil, errors.Unprocessable(core.BuiltInAuthenticationDisabled, "members cannot be deleted because WorkOS authentication is enabled")
	}
	org, _, _, err := organization.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	err = org.DeleteMember(r.Context(), r.PathValue("id"))
	return nil, err
}

// InviteMember sends an invitation email.
//
// It returns an errors.UnprocessableError with code WorkOSEnabled when WorkOS
// authentication is configured.
func (organization organization) InviteMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	if organization.workOS != nil {
		return nil, errors.Unprocessable(core.BuiltInAuthenticationDisabled, "members cannot be invited because WorkOS authentication is enabled")
	}
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	org, _, memberID, err := organization.authenticateAdminRequest(r)
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
	var body struct {
		Email string `json:"email"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	inviteMemberEmail, err := static.ReadFile("static/invite_member_email.html")
	if err != nil {
		return nil, errors.New("embedded file 'static/invite_member_email.html' not found in executable")
	}
	emailTemplate := strings.ReplaceAll(string(inviteMemberEmail), "${invitationFrom}", html.EscapeString(member.Email))
	emailTemplate = strings.ReplaceAll(emailTemplate, "${organization}", html.EscapeString(org.Name))
	emailTemplate = strings.ReplaceAll(emailTemplate, "${externalURL}", html.EscapeString(organization.externalURL))
	err = org.InviteMember(r.Context(), body.Email, emailTemplate)
	return nil, err
}

// Members returns the members of an organization.
func (organization organization) Members(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, _, err := organization.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	return org.Members(r.Context())
}

// PipelineMetricsPerDate returns metrics by day for a time interval between
// specified start and end dates.
func (organization organization) PipelineMetricsPerDate(_ http.ResponseWriter, r *http.Request) (any, error) {

	authenticated, err := organization.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if err := authenticated.applyRateLimitTo(r.Context(), authenticated.scopedRateLimitSubject(), x1); err != nil {
		return nil, err
	}

	// Parse start.
	s := r.PathValue("start")
	start, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return nil, errors.NotFound("start is not valid")
	}

	// Parse end.
	e := r.PathValue("end")
	end, err := time.Parse(time.DateOnly, e)
	if err != nil {
		return nil, errors.NotFound("end is not valid")
	}

	// Set workspace.
	var workspace string
	if authenticated.workspace != nil {
		workspace = authenticated.workspace.ID
	}

	// Parse selection.
	selection, err := parseMetricsSelection(r)
	if err != nil {
		return nil, err
	}

	return authenticated.organization.PipelineMetricsPerDate(r.Context(), start, end, workspace, selection)
}

// PipelineMetricsPerDay returns the pipeline metrics for a specified number of
// days.
func (organization organization) PipelineMetricsPerDay(_ http.ResponseWriter, r *http.Request) (any, error) {

	authenticated, err := organization.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if err := authenticated.applyRateLimitTo(r.Context(), authenticated.scopedRateLimitSubject(), x1); err != nil {
		return nil, err
	}

	// Parse days.
	n := r.PathValue("days")
	days, err := strconv.Atoi(n)
	if err != nil {
		return nil, errors.NotFound("days is not valid")
	}

	// Set workspace.
	var workspace string
	if authenticated.workspace != nil {
		workspace = authenticated.workspace.ID
	}

	// Parse selection.
	selection, err := parseMetricsSelection(r)
	if err != nil {
		return nil, err
	}

	return authenticated.organization.PipelineMetricsPerTimeUnit(r.Context(), days, core.Day, workspace, selection)
}

// PipelineMetricsPerHour returns the pipeline metrics for a specified number of
// hours.
func (organization organization) PipelineMetricsPerHour(_ http.ResponseWriter, r *http.Request) (any, error) {

	authenticated, err := organization.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if err := authenticated.applyRateLimitTo(r.Context(), authenticated.scopedRateLimitSubject(), x1); err != nil {
		return nil, err
	}

	// Parse hours.
	n := r.PathValue("hours")
	hours, err := strconv.Atoi(n)
	if err != nil {
		return nil, errors.NotFound("hours is not valid")
	}

	// Set workspace.
	var workspace string
	if authenticated.workspace != nil {
		workspace = authenticated.workspace.ID
	}

	// Parse selection.
	selection, err := parseMetricsSelection(r)
	if err != nil {
		return nil, err
	}

	return authenticated.organization.PipelineMetricsPerTimeUnit(r.Context(), hours, core.Hour, workspace, selection)
}

// PipelineMetricsPerMinute returns the pipeline metrics for a specified number
// of minutes.
func (organization organization) PipelineMetricsPerMinute(_ http.ResponseWriter, r *http.Request) (any, error) {

	authenticated, err := organization.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if err := authenticated.applyRateLimitTo(r.Context(), authenticated.scopedRateLimitSubject(), x1); err != nil {
		return nil, err
	}

	// Parse minutes.
	n := r.PathValue("minutes")
	minutes, err := strconv.Atoi(n)
	if err != nil {
		return nil, errors.NotFound("minutes is not valid")
	}

	// Set workspace.
	var workspace string
	if authenticated.workspace != nil {
		workspace = authenticated.workspace.ID
	}

	// Parse selection.
	selection, err := parseMetricsSelection(r)
	if err != nil {
		return nil, err
	}

	return authenticated.organization.PipelineMetricsPerTimeUnit(r.Context(), minutes, core.Minute, workspace, selection)
}

// SetStatus sets the status of an organization.
//
// Authentication is performed using the organizations API key.
func (organization organization) SetStatus(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := organization.authenticateOrganizationsRequest(r); err != nil {
		return nil, err
	}
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	org, err := organization.core.Organization(r.PathValue("id"))
	if err != nil {
		return nil, err
	}
	err = org.SetStatus(r.Context(), body.Enabled)
	return nil, err
}

// TestWorkspaceCreation tests a workspace creation.
func (organization organization) TestWorkspaceCreation(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	org, err := organization.admitOrganizationRequest(r, x1)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name          string             `json:"name"`
		ProfileSchema types.Type         `json:"profileSchema"`
		Warehouse     core.Warehouse     `json:"warehouse"`
		UIPreferences core.UIPreferences `json:"uiPreferences"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = org.TestWorkspaceCreation(r.Context(), body.Name, body.ProfileSchema, body.Warehouse, body.UIPreferences)
	return nil, err
}

// UpdateAccessKey updates the name of an access key for an organization.
func (organization organization) UpdateAccessKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	org, _, _, err := organization.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name string `json:"name"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = org.UpdateAccessKey(r.Context(), r.PathValue("key"), body.Name)
	return nil, err
}

// UpdateMember updates the currently logged-in member of the organization.
//
// It returns an errors.UnprocessableError with code WorkOSEnabled when WorkOS
// authentication is configured.
func (organization organization) UpdateMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	if organization.workOS != nil {
		return nil, errors.Unprocessable(core.BuiltInAuthenticationDisabled, "members cannot be updated because WorkOS authentication is enabled")
	}
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	org, _, memberID, err := organization.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		MemberToSet struct {
			Name     string `json:"name"`
			Image    []byte `json:"image"`
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"memberToSet"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	memberToSet := core.MemberToSet{
		Name:     body.MemberToSet.Name,
		Email:    body.MemberToSet.Email,
		Password: body.MemberToSet.Password,
	}
	if body.MemberToSet.Image != nil {
		fileType := http.DetectContentType(body.MemberToSet.Image)
		avatar := &core.Avatar{
			Image:    body.MemberToSet.Image,
			MimeType: fileType,
		}
		memberToSet.Avatar = avatar
	}
	err = org.UpdateMember(r.Context(), memberID, memberToSet)
	if _, ok := err.(*errors.NotFoundError); ok {
		err = errInvalidSessionCookie
	}
	return nil, err
}

// Update updates the name and the limits of the organization with the given
// identifier.
//
// Authentication is performed using the organizations API key.
func (organization organization) Update(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := organization.authenticateOrganizationsRequest(r); err != nil {
		return nil, err
	}
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	var body struct {
		Name   string              `json:"name"`
		Limits *organizationLimits `json:"limits"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	limits, err := parseOrganizationLimits(body.Limits)
	if err != nil {
		return nil, err
	}
	org, err := organization.core.Organization(r.PathValue("id"))
	if err != nil {
		return nil, err
	}
	err = org.Update(r.Context(), body.Name, &limits)
	return nil, err
}

// Workspace returns the current workspace.
func (organization organization) Workspace(_ http.ResponseWriter, r *http.Request) (any, error) {
	return organization.admitWorkspaceRequest(r, x1)
}

// Workspaces returns the workspaces of an organization.
func (organization organization) Workspaces(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, err := organization.admitOrganizationRequest(r, x1)
	if err != nil {
		return nil, err
	}
	workspaces := org.Workspaces()
	return map[string]any{"workspaces": workspaces}, nil
}

// parseMetricsSelection parses the pipeline metrics query parameters.
func parseMetricsSelection(r *http.Request) (core.MetricSelection, error) {

	q := r.URL.Query()

	var selection core.MetricSelection

	// Parse workspaces, connections, and pipelines parameters.
	if values, ok := q["workspaces"]; ok {
		selection.Workspaces = splitQueryParameters(values)
		if selection.Workspaces == nil {
			selection.Workspaces = []string{}
		}
	}
	if values, ok := q["connections"]; ok {
		selection.Connections = splitQueryParameters(values)
		if selection.Connections == nil {
			selection.Connections = []string{}
		}
	}
	if values, ok := q["pipelines"]; ok {
		selection.Pipelines = splitQueryParameters(values)
		if selection.Pipelines == nil {
			selection.Pipelines = []string{}
		}
	}

	// Parse the target parameter.
	if values, ok := q["target"]; ok {
		if len(values) != 1 {
			return core.MetricSelection{}, errors.BadRequest("'target' parameter cannot be specified multiple times")
		}
		switch t := strings.TrimSpace(values[0]); t {
		case "Event":
			selection.Target = core.TargetEvent
		case "User":
			selection.Target = core.TargetUser
		case "":
			return core.MetricSelection{}, errors.BadRequest("'target' parameter cannot be empty")
		default:
			return core.MetricSelection{}, errors.BadRequest("'target' parameter is not valid")
		}
	}

	return selection, nil
}
