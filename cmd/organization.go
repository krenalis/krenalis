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
	_, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	keys, err := o.APIKeys(r.Context())
	if err != nil {
		return nil, err
	}
	return map[string][]*core.APIKey{"keys": keys}, nil
}

// CreateWorkspace creates a workspace for the organization.
func (organization organization) CreateWorkspace(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name                string                   `json:"name"`
		UserSchema          types.Type               `json:"userSchema"`
		DisplayedProperties core.DisplayedProperties `json:"displayedProperties"`
		PrivacyRegion       core.PrivacyRegion       `json:"privacyRegion"`
		Warehouse           struct {
			Name     string             `json:"name"`
			Mode     core.WarehouseMode `json:"mode"`
			Settings json.Value         `json:"settings"`
		} `json:"warehouse"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	id, err := o.CreateWorkspace(r.Context(), body.Name, body.PrivacyRegion, body.UserSchema,
		body.DisplayedProperties, body.Warehouse.Name, body.Warehouse.Settings, body.Warehouse.Mode)
	if err != nil {
		return nil, err
	}
	return map[string]int{"id": id}, nil
}

// TestWorkspaceCreation tests a workspace creation.
func (organization organization) TestWorkspaceCreation(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		Name     string             `json:"name"`
		Mode     core.WarehouseMode `json:"mode"`
		Settings json.Value         `json:"settings"`
	}
	err = json.Decode(r.Body, &body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = o.TestWorkspaceCreation(r.Context(), body.Name, body.Settings)
	return nil, err
}

// CreateAPIKey creates a new API key for an organization.
func (organization organization) CreateAPIKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
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
	id, token, err := o.CreateAPIKey(r.Context(), body.Name, workspace)
	if err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "token": token}, nil
}

// DeleteAPIKey deletes an API key of an organization.
func (organization organization) DeleteAPIKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	key, err := organization.key(r)
	if err != nil {
		return nil, err
	}
	err = o.DeleteAPIKey(r.Context(), key)
	return nil, err
}

// DeleteMember deletes a member of an organization.
func (organization organization) DeleteMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	member, err := organization.member(r)
	if err != nil {
		return nil, err
	}
	err = o.DeleteMember(r.Context(), member)
	return nil, err
}

// InviteMember sends an invitation email.
func (organization organization) InviteMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	member, o, err := organization.credentials(r)
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
	emailTemplate = strings.ReplaceAll(emailTemplate, "${organization}", html.EscapeString(o.Name))
	err = o.InviteMember(r.Context(), body.Email, emailTemplate)
	return nil, err
}

// Members returns the members of an organization.
func (organization organization) Members(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	return o.Members(r.Context())
}

// UpdateAPIKey updates the name of an API key for an organization.
func (organization organization) UpdateAPIKey(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
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
	err = o.UpdateAPIKey(r.Context(), key, body.Name)
	return nil, err
}

// UpdateMember updates a member of an organization.
func (organization organization) UpdateMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	member, o, err := organization.credentials(r)
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
	err = o.UpdateMember(r.Context(), member.ID, memberToSet)
	return nil, err
}

// Workspace returns the workspace of an organization.
func (organization organization) Workspace(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	v := r.PathValue("workspace")
	if v[0] == '+' {
		return nil, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return nil, errors.NotFound("")
	}
	return o.Workspace(id)
}

// Workspaces returns the workspaces of an organization.
func (organization organization) Workspaces(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	return o.Workspaces(), nil
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

func (organization organization) member(r *http.Request) (int, error) {
	v := r.PathValue("member")
	if v[0] == '+' {
		return 0, errors.NotFound("")
	}
	id, _ := strconv.Atoi(v)
	if id <= 0 {
		return 0, errors.NotFound("")
	}
	return id, nil
}
