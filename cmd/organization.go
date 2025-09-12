//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/types"
)

type organization struct {
	*apisServer
}

// AddMember adds a new member of an organization.
//
// If the ability to add new members without requiring email
// verification has not been enabled, it returns an
// errors.UnprocessableError error with code EmailVerificationRequired.
func (organization organization) AddMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	if !organization.skipMemberEmailVerification {
		return nil, errors.Unprocessable(core.EmailVerificationRequired, "Email verification is required")
	}
	org, _, err := organization.memberCredentials(r)
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
	err = org.AddMember(r.Context(), memberToSet)
	return nil, err
}

// AccessKeys returns the access keys of an organization.
func (organization organization) AccessKeys(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, err := organization.memberCredentials(r)
	if err != nil {
		return nil, err
	}
	keys, err := org.AccessKeys(r.Context())
	if err != nil {
		return nil, err
	}
	return map[string][]*core.AccessKey{"keys": keys}, nil
}

// CreateAccessKey creates a new access key for an organization.
func (organization organization) CreateAccessKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, err := organization.memberCredentials(r)
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
		return nil, errors.BadRequest("access key type is required")
	}
	id, token, err := org.CreateAccessKey(r.Context(), body.Name, workspace, *body.Type)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "token": token}, nil
}

// CreateWorkspace creates a workspace for the organization.
func (organization organization) CreateWorkspace(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, err := organization.organization(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name       string     `json:"name"`
		UserSchema types.Type `json:"userSchema"`
		Warehouse  struct {
			Type        string             `json:"type"`
			Mode        core.WarehouseMode `json:"mode"`
			Settings    json.Value         `json:"settings"`
			MCPSettings json.Value         `json:"mcpSettings"`
		} `json:"warehouse"`
		UIPreferences core.UIPreferences `json:"uiPreferences"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if body.Warehouse.MCPSettings != nil && body.Warehouse.MCPSettings.IsNull() {
		body.Warehouse.MCPSettings = nil
	}
	id, err := org.CreateWorkspace(r.Context(), body.Name, body.UserSchema,
		body.UIPreferences, body.Warehouse.Type, body.Warehouse.Settings,
		body.Warehouse.MCPSettings, body.Warehouse.Mode)
	if err != nil {
		if err2, ok := err.(*errors.UnprocessableError); ok && err2.Code == core.OrganizationNotExist {
			return nil, errors.Unauthorized("API key in the Authorization header of the request does not exist")
		}
		return nil, err
	}
	return map[string]int{"id": id}, nil
}

// DeleteAccessKey deletes an access key of an organization.
func (organization organization) DeleteAccessKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, err := organization.memberCredentials(r)
	if err != nil {
		return nil, err
	}
	key, err := organization.key(r)
	if err != nil {
		return nil, err
	}
	err = org.DeleteAccessKey(r.Context(), key)
	return nil, err
}

// DeleteMember deletes a member of an organization.
func (organization organization) DeleteMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, err := organization.memberCredentials(r)
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
	org, memberID, err := organization.memberCredentials(r)
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
	emailTemplate := strings.ReplaceAll(inviteMemberEmail, "${invitationFrom}", html.EscapeString(member.Email))
	emailTemplate = strings.ReplaceAll(emailTemplate, "${organization}", html.EscapeString(org.Name))
	emailTemplate = strings.ReplaceAll(emailTemplate, "${externalURL}", html.EscapeString(organization.externalURL))
	err = org.InviteMember(r.Context(), body.Email, emailTemplate)
	return nil, err
}

// Members returns the members of an organization.
func (organization organization) Members(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, err := organization.memberCredentials(r)
	if err != nil {
		return nil, err
	}
	return org.Members(r.Context())
}

// TestWorkspaceCreation tests a workspace creation.
func (organization organization) TestWorkspaceCreation(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, err := organization.organization(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name       string     `json:"name"`
		UserSchema types.Type `json:"userSchema"`
		Warehouse  struct {
			Type        string             `json:"type"`
			Mode        core.WarehouseMode `json:"mode"`
			Settings    json.Value         `json:"settings"`
			MCPSettings json.Value         `json:"mcpSettings"`
		} `json:"warehouse"`
		UIPreferences core.UIPreferences `json:"uiPreferences"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	if body.Warehouse.MCPSettings != nil && body.Warehouse.MCPSettings.IsNull() {
		body.Warehouse.MCPSettings = nil
	}
	err = org.TestWorkspaceCreation(r.Context(), body.Name, body.UserSchema,
		body.UIPreferences, body.Warehouse.Type, body.Warehouse.Settings,
		body.Warehouse.MCPSettings, body.Warehouse.Mode)
	return nil, err
}

// UpdateAccessKey updates the name of an access key for an organization.
func (organization organization) UpdateAccessKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, err := organization.memberCredentials(r)
	if err != nil {
		return nil, err
	}
	key, err := organization.key(r)
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

// UpdateMember updates a member of an organization.
func (organization organization) UpdateMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, memberID, err := organization.memberCredentials(r)
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

// Workspace returns the current workspace.
func (organization organization) Workspace(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, ws, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, errors.Forbidden("provided key ")
	}
	return ws, nil
}

// Workspaces returns the workspaces of an organization.
func (organization organization) Workspaces(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, err := organization.organization(r)
	if err != nil {
		return nil, err
	}
	return map[string]any{"workspaces": org.Workspaces()}, nil
}

func (organization organization) id(r *http.Request) (int, error) {
	v := r.PathValue("id")
	if v[0] == '+' {
		return 0, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return 0, errors.NotFound("")
	}
	return id, nil
}

func (organization organization) key(r *http.Request) (int, error) {
	v := r.PathValue("key")
	if v[0] == '+' {
		return 0, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return 0, errors.NotFound("")
	}
	return id, nil
}

// organization returns the organization.
func (organization organization) organization(r *http.Request) (*core.Organization, error) {
	org, _, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	return org, nil
}
