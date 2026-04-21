// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"html"
	"net/http"
	"strings"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/google/uuid"
)

type organization struct {
	*apisServer
}

// AddMember adds a new member of an organization.
//
// If the ability to add new members without requiring email invitation has not
// been enabled, it returns an errors.UnprocessableError error with code
// EmailInvitationRequired.
func (organization organization) AddMember(_ http.ResponseWriter, r *http.Request) (any, error) {
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
	err = org.AddMember(r.Context(), memberToSet)
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
		Workspace *int                `json:"workspace"`
		Type      *core.AccessKeyType `json:"type"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	var workspace int
	if body.Workspace != nil {
		if *body.Workspace == 0 {
			return nil, errors.BadRequest("workspace is not a valid workspace identifier")
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
	org, ws, err := organization.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if ws != nil {
		return nil, errors.Unauthorized("workspaces cannot be created with a workspace restricted API key")
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
	return map[string]int{"id": id}, nil
}

// Delete deletes the organization with the given identifier.
func (organization organization) Delete(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := organization.authenticateOrganizationsRequest(r); err != nil {
		return nil, err
	}
	if err := validateForbiddenBody(r); err != nil {
		return nil, err
	}
	id, ok := parseOrganizationUUID(r.PathValue("id"))
	if !ok {
		return nil, errors.BadRequest("identifier %q is not a valid organization identifier", r.PathValue("id"))
	}
	org, err := organization.core.Organization(id)
	if err != nil {
		return nil, err
	}
	err = org.Delete(r.Context())
	return nil, err
}

// DeleteAccessKey deletes an access key of an organization.
func (organization organization) DeleteAccessKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateForbiddenBody(r); err != nil {
		return nil, err
	}
	org, _, _, err := organization.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	id, err := organization.key(r)
	if err != nil {
		return nil, err
	}
	err = org.DeleteAccessKey(r.Context(), id)
	return nil, err
}

// DeleteMember deletes a member of an organization.
func (organization organization) DeleteMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateForbiddenBody(r); err != nil {
		return nil, err
	}
	org, _, _, err := organization.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	id, err := organization.id(r)
	if err != nil {
		return nil, err
	}
	err = org.DeleteMember(r.Context(), id)
	return nil, err
}

// InviteMember sends an invitation email.
func (organization organization) InviteMember(_ http.ResponseWriter, r *http.Request) (any, error) {
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

// TestWorkspaceCreation tests a workspace creation.
func (organization organization) TestWorkspaceCreation(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	org, ws, err := organization.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if ws != nil {
		return nil, errors.Unauthorized("workspace creation cannot be tested with a workspace restricted API key")
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
	key, err := organization.key(r) // ID of the access key
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
	err = org.UpdateAccessKey(r.Context(), key, body.Name)
	return nil, err
}

// UpdateMember updates the currently logged-in member of the organization.
func (organization organization) UpdateMember(_ http.ResponseWriter, r *http.Request) (any, error) {
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

// Update updates the name of the organization with the given identifier.
func (organization organization) Update(_ http.ResponseWriter, r *http.Request) (any, error) {
	if err := organization.authenticateOrganizationsRequest(r); err != nil {
		return nil, err
	}
	if err := validateRequiredBody(r, false); err != nil {
		return nil, err
	}
	id, ok := parseOrganizationUUID(r.PathValue("id"))
	if !ok {
		return nil, errors.BadRequest("identifier %q is not a valid organization identifier", r.PathValue("id"))
	}
	var body struct {
		Name string `json:"name"`
	}
	err := json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	org, err := organization.core.Organization(id)
	if err != nil {
		return nil, err
	}
	err = org.Update(r.Context(), body.Name)
	return nil, err
}

// Workspace returns the current workspace.
func (organization organization) Workspace(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, ws, err := organization.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errMissingWorkspace
	}
	return ws, nil
}

// Workspaces returns the workspaces of an organization.
func (organization organization) Workspaces(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, ws, err := organization.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if ws != nil {
		return nil, errors.Unauthorized("workspaces cannot be listed with a workspace restricted API key")
	}
	return map[string]any{"workspaces": org.Workspaces()}, nil
}

// id returns the value of the 'id' path parameter parsed as a member ID.
func (organization organization) id(r *http.Request) (int, error) {
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		return 0, errors.BadRequest("identifier %q is not a valid member identifier", r.PathValue("id"))
	}
	return id, nil
}

// key returns the value of the 'key' path parameter parsed as an access key ID.
func (organization organization) key(r *http.Request) (int, error) {
	key, ok := parseID(r.PathValue("key"))
	if !ok {
		return 0, errors.BadRequest("identifier %q is not a valid access key identifier", r.PathValue("key"))
	}
	return key, nil
}

// parseOrganizationUUID parses and returns a UUID representing the ID of an
// organization, returning true if it is valid, false otherwise.
func parseOrganizationUUID(s string) (uuid.UUID, bool) {
	if len(s) != 36 {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
