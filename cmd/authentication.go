// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	"net/http"
	"strings"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/validation"
)

// x1 is the cost of an API operation that consumes one unit of rate-limit
// capacity.
const x1 = 1

// admitOrganizationRequest authenticates an organization-only request,
// applies the organization's nonspecific rate-limit budget unless the request
// is from the Admin console, and returns the organization.
//
// See also [admitWorkspaceOptionalRequest] and [admitWorkspaceRequest].
func (s *apisServer) admitOrganizationRequest(r *http.Request, rateLimitCost int) (*core.Organization, error) {
	authenticated, err := s.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if authenticated.workspace != nil {
		return nil, errors.Unauthorized("organization request cannot specify a workspace")
	}
	if err := authenticated.applyRateLimitTo(r.Context(), authenticated.organization, rateLimitCost); err != nil {
		return nil, err
	}
	return authenticated.organization, nil
}

// admitWorkspaceOptionalRequest authenticates a request for which selecting a
// workspace is optional. Unless the request comes from the Admin console, it
// applies the selected workspace's rate-limit budget or, if no workspace is
// selected, the authenticated organization's nonspecific budget.
//
// See also [admitOrganizationRequest] and [admitWorkspaceRequest].
func (s *apisServer) admitWorkspaceOptionalRequest(r *http.Request, rateLimitCost int) (authenticatedRequest, error) {
	authenticated, err := s.authenticateRequest(r)
	if err != nil {
		return authenticatedRequest{}, err
	}
	if err := authenticated.applyRateLimitTo(r.Context(), authenticated.scopedRateLimitSubject(), rateLimitCost); err != nil {
		return authenticatedRequest{}, err
	}
	return authenticated, nil
}

// admitWorkspaceRequest authenticates a workspace-scoped request, applies the
// workspace rate limit unless the request is from the Admin console, and
// returns the required workspace.
//
// See also [admitWorkspaceOptionalRequest] and [admitOrganizationRequest].
func (s *apisServer) admitWorkspaceRequest(r *http.Request, rateLimitCost int) (*core.Workspace, error) {
	authenticated, err := s.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if authenticated.workspace == nil {
		return nil, errMissingWorkspace
	}
	if err := authenticated.applyRateLimitTo(r.Context(), authenticated.workspace, rateLimitCost); err != nil {
		return nil, err
	}
	return authenticated.workspace, nil
}

// authenticateAPIKeyRequest authenticates a request using an API key and
// resolves its organization and optional workspace. The caller has already
// confirmed that the Authorization header is present, so this function does
// not fall back to Admin console authentication.
func (s *apisServer) authenticateAPIKeyRequest(r *http.Request, auth []string) (authenticatedRequest, error) {
	if len(auth) > 1 {
		return authenticatedRequest{}, errors.BadRequest("request contains multiple Authorization headers")
	}
	token, found := validation.ParseBearer(auth[0])
	if !found {
		return authenticatedRequest{}, errors.BadRequest("Authorization header is invalid; it should be in the format 'Authorization: Bearer <YOUR_API_KEY>'")
	}
	token, found = strings.CutPrefix(token, "api_")
	if !found {
		return authenticatedRequest{}, errors.BadRequest("API key is not valid; API keys start with the api_ prefix")
	}
	organizationID, workspaceID, err := s.core.AccessKey(r.Context(), token, core.AccessKeyTypeAPI)
	if err != nil {
		switch err.(type) {
		case *errors.BadRequestError:
			err = errors.Unauthorized("API key in the Authorization header of the request is malformed")
		case *errors.NotFoundError:
			err = errors.Unauthorized("API key in the Authorization header of the request does not exist")
		}
		return authenticatedRequest{}, err
	}
	org, err := s.core.Organization(organizationID)
	if err != nil {
		return authenticatedRequest{}, err
	}
	if !org.Enabled {
		return authenticatedRequest{}, errors.Unprocessable(core.OrganizationDisabled, "organization %s is disabled", org.ID)
	}
	if workspaceID != "" {
		ws, err := org.Workspace(workspaceID)
		if err != nil {
			return authenticatedRequest{}, err
		}
		return authenticatedRequest{organization: org, workspace: ws}, nil
	}

	header, ok := r.Header["Krenalis-Workspace"]
	if !ok {
		return authenticatedRequest{organization: org}, nil
	}
	if len(header) > 1 {
		return authenticatedRequest{}, errors.BadRequest("request contains multiple Krenalis-Workspace headers")
	}
	id := header[0]
	if !core.IsValidID(id) {
		return authenticatedRequest{}, errors.BadRequest("Krenalis-Workspace header is invalid; it should be in the format 'Krenalis-Workspace: <WORKSPACE_ID>'")
	}
	ws, err := org.Workspace(id)
	if err != nil {
		return authenticatedRequest{}, err
	}
	return authenticatedRequest{organization: org, workspace: ws}, nil
}

// authenticateAdminRequest authenticates a request from the Admin console and
// returns its organization, optional workspace, and member ID.
//
// Authorization is provided through a session cookie.
//
// The workspace is nil when the "Krenalis-Workspace" header is absent.
//
// An invalid or no longer usable session returns errInvalidSessionCookie.
// Other request, organization, and workspace errors are returned unchanged.
func (s *apisServer) authenticateAdminRequest(r *http.Request) (org *core.Organization, ws *core.Workspace, userID string, err error) {

	// Get the session.
	cookie, _ := r.Cookie(sessionCookieName)
	if cookie == nil {
		return nil, nil, "", errors.Unauthorized("Authorization header with the API key is not present in the request")
	}
	session := &sessionCookie{}
	se, err := s.secureCookie(r.Context())
	if err != nil {
		return nil, nil, "", err
	}
	err = se.Decode(sessionCookieName, cookie.Value, session)
	if err != nil {
		return nil, nil, "", errInvalidSessionCookie
	}

	// Get the organization.
	org, err = s.core.Organization(session.Organization)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			return nil, nil, "", errInvalidSessionCookie
		}
		return nil, nil, "", err
	}
	// Verify that the member can still log in.
	if canLogin, err := org.CanMemberLogin(session.Member); err != nil || !canLogin {
		return nil, nil, "", errInvalidSessionCookie
	}
	// Verify that the organization is enabled.
	if !org.Enabled {
		return nil, nil, "", errors.Unprocessable(core.OrganizationDisabled, "organization %s is disabled", org.ID)
	}
	// If the 'Krenalis-Workspace' header is missing, return with a nil workspace.
	header, ok := r.Header["Krenalis-Workspace"]
	if !ok {
		return org, nil, session.Member, nil
	}
	if len(header) > 1 {
		return nil, nil, "", errors.BadRequest("request contains multiple Krenalis-Workspace headers")
	}
	workspaceID := header[0]
	if !core.IsValidID(workspaceID) {
		return nil, nil, "", errors.BadRequest("Krenalis-Workspace header is invalid; it should be in the format 'Krenalis-Workspace: <WORKSPACE_ID>'")
	}
	ws, err = org.Workspace(workspaceID)
	if err != nil {
		return nil, nil, "", err
	}

	return org, ws, session.Member, nil
}

// authenticateOrganizationsRequest authenticates a request to the organizations
// API using an organizations API key from the Authorization header.
func (s *apisServer) authenticateOrganizationsRequest(r *http.Request) error {
	auth, ok := r.Header["Authorization"]
	if !ok {
		return errors.Unauthorized("Authorization header with the organizations API key is not present in the request")
	}
	if len(auth) > 1 {
		return errors.BadRequest("request contains multiple Authorization headers")
	}
	token, found := validation.ParseBearer(auth[0])
	if !found {
		return errors.BadRequest("Authorization header is invalid; it should be in the format 'Authorization: Bearer <YOUR_ORGANIZATIONS_API_KEY>'")
	}
	if !strings.HasPrefix(token, "org_") {
		return errors.BadRequest("organizations APIs require specific keys for authentication (these are keys that begin with 'org_')")
	}
	if s.organizationsAPIKey == "" || token != s.organizationsAPIKey {
		return errors.Unauthorized("organizations API key in the Authorization header of the request is not valid")
	}
	return nil
}

// authenticateRequest authenticates a request and returns its organization and
// optional workspace. It does not consume rate-limit capacity.
//
// Authentication uses either:
//
// - an API key from the "Authorization" header;
// - a session cookie from the Admin console.
//
// When the Authorization header is present, API key authentication is used and
// Admin console authentication is not attempted.
//
// The workspace is nil when no workspace is selected by either the API key or
// the "Krenalis-Workspace" header.
//
// API key lookup errors for malformed or unknown keys are returned as
// unauthorized errors. Other validation, organization, and workspace errors
// are returned unchanged.
func (s *apisServer) authenticateRequest(r *http.Request) (authenticatedRequest, error) {

	if auth, ok := r.Header["Authorization"]; ok {
		return s.authenticateAPIKeyRequest(r, auth)
	}

	org, ws, _, err := s.authenticateAdminRequest(r)
	if err != nil {
		return authenticatedRequest{}, err
	}
	return authenticatedRequest{organization: org, workspace: ws, rateLimitExempt: true}, nil
}

// rateLimitCapacityConsumer is a subject that can consume API rate-limit
// capacity.
type rateLimitCapacityConsumer interface {
	ConsumeRateLimitCapacity(context.Context, int) error
}

// authenticatedRequest contains the organization and optional workspace
// identified by the request's credentials. Admin requests are exempt from API
// rate limiting.
type authenticatedRequest struct {
	organization    *core.Organization
	workspace       *core.Workspace
	rateLimitExempt bool
}

// applyRateLimitTo consumes capacity from subject unless the request is exempt.
func (authenticated authenticatedRequest) applyRateLimitTo(ctx context.Context, subject rateLimitCapacityConsumer, cost int) error {
	if authenticated.rateLimitExempt {
		return nil
	}
	err := subject.ConsumeRateLimitCapacity(ctx, cost)
	if errors.Is(err, core.ErrAPICapacityExceeded) {
		return errors.TooManyRequests("API rate limit exceeded")
	}
	return err
}

// scopedRateLimitSubject returns the subject whose budget applies to the
// request: the workspace for a workspace-scoped request, or the organization
// otherwise.
func (authenticated authenticatedRequest) scopedRateLimitSubject() rateLimitCapacityConsumer {
	if authenticated.workspace != nil {
		return authenticated.workspace
	}
	return authenticated.organization
}
