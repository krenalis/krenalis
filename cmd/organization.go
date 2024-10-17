//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/meergo/meergo/apis"
	"github.com/meergo/meergo/apis/errors"
)

type organization struct {
	*apisServer
}

// AddWorkspace adds a workspace to an organization.
func (organization organization) AddWorkspace(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Name          string
		PrivacyRegion apis.PrivacyRegion
		Warehouse     struct {
			Name     string
			Mode     apis.WarehouseMode
			Settings rawJSON
		}
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	id, err := o.AddWorkspace(r.Context(), body.Name, body.PrivacyRegion, body.Warehouse.Name, body.Warehouse.Settings, body.Warehouse.Mode)
	if err != nil {
		return nil, err
	}
	return map[string]int{"id": id}, nil
}

// CanInitializeWarehouse indicates whether a data warehouse can be initialized.
func (organization organization) CanInitializeWarehouse(_ http.ResponseWriter, r *http.Request) (any, error) {
	_, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	body := struct {
		Name     string
		Mode     apis.WarehouseMode
		Settings rawJSON
	}{}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	err = o.CanInitializeWarehouse(r.Context(), body.Name, body.Settings)
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
		Email string
	}
	err = json.NewDecoder(r.Body).Decode(&body)
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

// SetMember sets a member of an organization.
func (organization organization) SetMember(_ http.ResponseWriter, r *http.Request) (any, error) {
	member, o, err := organization.credentials(r)
	if err != nil {
		return nil, err
	}
	var body struct {
		MemberToSet struct {
			Name     string
			Image    []byte
			Email    string
			Password string
		}
	}
	err = json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("%s", err)
	}
	memberToSet := apis.MemberToSet{
		Name:     body.MemberToSet.Name,
		Email:    body.MemberToSet.Email,
		Password: body.MemberToSet.Password,
	}
	if body.MemberToSet.Image != nil {
		fileType := http.DetectContentType(body.MemberToSet.Image)
		avatar := &apis.Avatar{
			Image:    body.MemberToSet.Image,
			MimeType: fileType,
		}
		memberToSet.Avatar = avatar
	}
	err = o.SetMember(r.Context(), member.ID, memberToSet)
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
