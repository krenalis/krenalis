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

// authenticatedRequest contains the organization and optional workspace
// resolved from a request's credentials. Admin requests are exempt from API
// rate limiting.
type authenticatedRequest struct {
	organization    *core.Organization
	workspace       *core.Workspace
	rateLimitExempt bool
}

// applyRateLimit consumes capacity from subject unless the request is exempt.
func (authenticated authenticatedRequest) applyRateLimit(ctx context.Context, subject rateLimitCapacityConsumer, cost int) error {
	if authenticated.rateLimitExempt {
		return nil
	}
	err := subject.ConsumeRateLimitCapacity(ctx, cost)
	if errors.Is(err, core.ErrAPICapacityExceeded) {
		return errors.TooManyRequests("API rate limit exceeded")
	}
	return err
}

// rateLimitCapacityConsumer is a subject with an API rate-limit bucket.
type rateLimitCapacityConsumer interface {
	ConsumeRateLimitCapacity(context.Context, int) error
}

// admitNonspecificRequest authenticates a request that is not in a specific
// API category and applies the organization's rate-limit policy.
func (s *apisServer) admitNonspecificRequest(r *http.Request, rateLimitCost int) (authenticatedRequest, error) {
	authenticated, err := s.authenticateRequest(r)
	if err != nil {
		return authenticatedRequest{}, err
	}
	if err := authenticated.applyRateLimit(r.Context(), authenticated.organization, rateLimitCost); err != nil {
		return authenticatedRequest{}, err
	}
	return authenticated, nil
}

// admitWorkspaceRequest admits a request and returns its required workspace.
func (s *apisServer) admitWorkspaceRequest(r *http.Request, rateLimitCost int) (*core.Workspace, error) {
	authenticated, err := s.authenticateRequest(r)
	if err != nil {
		return nil, err
	}
	if authenticated.workspace == nil {
		return nil, errMissingWorkspace
	}
	if err := authenticated.applyRateLimit(r.Context(), authenticated.workspace, rateLimitCost); err != nil {
		return nil, err
	}
	return authenticated.workspace, nil
}

// authenticateAdminRequest authenticates an Admin console request r and
// returns the associated organization, workspace, and member ID.
//
// Authorization is provided via a session cookie.
//
// The workspace is nil when the "Krenalis-Workspace" header is absent.
//
// It returns errors.UnauthorizedError if authorization fails, or
// errInvalidSessionCookie if the session cookie is invalid.
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

// authenticateAPIKeyRequest authenticates an API key request and resolves its
// organization and optional workspace. The Authorization header is supplied by
// authenticateRequest so this function cannot fall back to Admin credentials.
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

// authenticateOrganizationsRequest authenticates a request to the organizations
// API. Authorization is provided via the "Authorization: Bearer <key>" header.
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

// authenticateRequest authenticates the request r and returns the associated
// organization and optional workspace. It does not consume rate-limit
// capacity.
//
// Authorization sources:
//   - API key in the "Authorization" header
//   - Session cookie from the Admin console
//
// The workspace is nil when the "Krenalis-Workspace" header is absent and either:
//   - the API key is not bound to a workspace, or
//   - the session cookie is from the Admin console.
//
// If authorization fails, an errors.UnauthorizedError is returned.
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
