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
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

type organization struct {
	*apisServer
}

// APIKeys returns the API keys of an organization.
func (organization organization) APIKeys(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, err := organization.memberCredentials(r)
	if err != nil {
		return nil, err
	}
	keys, err := org.APIKeys(r.Context())
	if err != nil {
		return nil, err
	}
	return map[string][]*core.APIKey{"keys": keys}, nil
}

// CreateAPIKey creates a new API key for an organization.
func (organization organization) CreateAPIKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, err := organization.memberCredentials(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name      string `json:"name"`
		Workspace *int   `json:"workspace"`
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
	id, token, err := org.CreateAPIKey(r.Context(), body.Name, workspace)
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
			Type     string             `json:"type"`
			Mode     core.WarehouseMode `json:"mode"`
			Settings json.Value         `json:"settings"`
		} `json:"warehouse"`
		UIPreferences core.UIPreferences `json:"uiPreferences"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	id, err := org.CreateWorkspace(r.Context(), body.Name, body.UserSchema,
		body.UIPreferences, body.Warehouse.Type, body.Warehouse.Settings, body.Warehouse.Mode)
	if err != nil {
		if err2, ok := err.(*errors.UnprocessableError); ok && err2.Code == core.OrganizationNotExist {
			return nil, errors.Unauthorized("API key in the Authorization header of the request does not exist")
		}
		return nil, err
	}
	return map[string]int{"id": id}, nil
}

// DeleteAPIKey deletes an API key of an organization.
func (organization organization) DeleteAPIKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, _, err := organization.memberCredentials(r)
	if err != nil {
		return nil, err
	}
	key, err := organization.key(r)
	if err != nil {
		return nil, err
	}
	err = org.DeleteAPIKey(r.Context(), key)
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
	org, member, err := organization.memberCredentials(r)
	if err != nil {
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
			Type     string             `json:"type"`
			Mode     core.WarehouseMode `json:"mode"`
			Settings json.Value         `json:"settings"`
		} `json:"warehouse"`
		UIPreferences core.UIPreferences `json:"uiPreferences"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = org.TestWorkspaceCreation(r.Context(), body.Name, body.UserSchema,
		body.UIPreferences, body.Warehouse.Type, body.Warehouse.Settings, body.Warehouse.Mode)
	return nil, err
}

// UpdateAPIKey updates the name of an API key for an organization.
func (organization organization) UpdateAPIKey(_ http.ResponseWriter, r *http.Request) (any, error) {
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
	err = org.UpdateAPIKey(r.Context(), key, body.Name)
	return nil, err
}

// UpdateMember updates a member of an organization.
func (organization organization) UpdateMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	org, member, err := organization.memberCredentials(r)
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
	err = org.UpdateMember(r.Context(), member.ID, memberToSet)
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
