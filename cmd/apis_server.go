// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/krenalis/krenalis/cmd/internal/workos"
	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/validation"

	"github.com/gorilla/securecookie"
	"golang.org/x/text/unicode/norm"
)

// maxRequestSize is the maximum size bytes for an API request.
const maxRequestSize = 500 * 1024

// maxWorkOSPayloadSize is the maximum size in bytes for a WorkOS webhook or
// action payload.
const maxWorkOSPayloadSize = 64 * 1024

var newline = []byte("\n")

var (
	// errMissingWorkspace is returned when the "Krenalis-Workspace" header is required but not provided.
	errMissingWorkspace = errors.Forbidden("Krenalis-Workspace header is missing; provide it in the format 'Krenalis-Workspace: <WORKSPACE_ID>'")

	// errInvalidSessionCookie is returned when a session cookie has expired or is no lo longer valid.
	errInvalidSessionCookie = errors.Unauthorized("session cookie has expired or is no longer valid")
)

// sessionMaxAge contains the max age property for the session cookie (6 hours).
const sessionMaxAge = 6 * 60 * 60

type sessionCookie struct {
	Organization string
	Member       string
}

// cookieKeysFunc loads the hash key and block key used to sign and encrypt
// session cookies.
type cookieKeysFunc func(context.Context) ([]byte, []byte, error)

const (
	sessionCookieName = "krenalis_session"
	sessionCookiePath = "/v1/"
)

type apisServer struct {
	core    *core.Core
	cookies struct {
		sync.Mutex
		*securecookie.SecureCookie // secureCookie contains keys to encrypt/decrypt/remove the session cookie.
	}
	cookieKeys             cookieKeysFunc
	mux                    *http.ServeMux
	runsOnHTTPS            bool
	javaScriptSDKURL       string
	externalURL            string
	externalEventURL       string
	externalAssetsURLs     []string
	potentialConnectorsURL string // must be a valid URL or empty string (which means: do not load the JSON file).
	inviteMembersViaEmail  bool
	organizationsAPIKey    string         // can be empty (which means that organizations APIs cannot be used)
	workos                 *workos.Workos // nil when WorkOS authentication is not configured.
	sentryTelemetry        struct {
		level       core.TelemetryLevel
		errorTunnel *sentryErrorTunnel
	}
}

// newAPIsServer returns an APIs server that handles requests for the given
// Core.
// runsOnHTTPs indicates if the server runs on HTTPS.
func newAPIsServer(core *core.Core, runsOnHTTPS bool, javaScriptSDKURL, externalURL,
	externalEventURL string, externalAssetsURLs []string, potentialConnectorsURL string,
	inviteMembersViaEmail bool, organizationsAPIKey string, sentryTelemetryLevel core.TelemetryLevel,
	sentryErrorTunnel *sentryErrorTunnel, workosClientID, workosAPIKey, workosWebhookSecret,
	workosActionsSecret string, workosDevMode bool) *apisServer {

	s := &apisServer{
		core:                   core,
		cookieKeys:             core.CookieKeys,
		runsOnHTTPS:            runsOnHTTPS,
		javaScriptSDKURL:       javaScriptSDKURL,
		externalURL:            externalURL,
		externalEventURL:       externalEventURL,
		externalAssetsURLs:     externalAssetsURLs,
		potentialConnectorsURL: potentialConnectorsURL,
		inviteMembersViaEmail:  inviteMembersViaEmail,
		organizationsAPIKey:    organizationsAPIKey,
	}

	if workosClientID != "" {
		s.workos = workos.New(workosClientID, workosAPIKey, workosWebhookSecret, workosActionsSecret, workosDevMode)
	}

	s.sentryTelemetry.level = sentryTelemetryLevel
	s.sentryTelemetry.errorTunnel = sentryErrorTunnel

	s.mux = http.NewServeMux()
	for pattern, handler := range endpoints(s) {
		s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store, max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			response, err := handler(w, r)
			if err != nil {
				if r.Context().Err() != nil {
					return
				}
				if err == errInvalidSessionCookie {
					s.deleteSessionCookie(w, r)
				}
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				slog.Error("cmd: error occurred serving Core", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if response != nil {
				w.Header().Set("Content-Type", "application/json")
				_ = json.Encode(w, response)
			}
			// Append a newline to JSON responses without a Content-Length header.
			// This keeps terminal tools like curl from printing the prompt on the same line.
			if h := w.Header(); h.Get("Content-Type") == "application/json" {
				if _, ok := h["Content-Length"]; !ok {
					_, _ = w.Write(newline)
				}
			}
		})
	}

	return s
}

// ServeHTTP servers the API methods from HTTP.
func (s *apisServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if !strings.HasPrefix(r.URL.Path, "/v1/") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case "GET", "DELETE":
		err := validateForbiddenBody(r)
		if err != nil {
			if err, ok := err.(errors.ResponseWriterTo); ok {
				_ = err.WriteTo(w)
				return
			}
			slog.Error("cmd: error occurred serving Core", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	case "POST", "PUT":
		// Validate the content length.
		if cl := r.Header.Get("Content-Length"); cl != "" {
			length, _ := strconv.Atoi(cl)
			if length < 0 || length > maxRequestSize {
				err := errors.BadRequest("request's content length must be in the range [1,%d]", maxRequestSize)
				_ = err.WriteTo(w)
				return
			}
		}

	}

	r.URL.Path = r.URL.Path[len("/v1"):]
	if r.URL.RawPath != "" {
		r.URL.RawPath = r.URL.RawPath[len("/v1"):]
	}

	s.mux.ServeHTTP(w, r)
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
	// Verify that the member still exists.
	if exists, err := org.HasMember(session.Member); err != nil || !exists {
		return nil, nil, "", errInvalidSessionCookie
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
// organization and optional workspace.
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
func (s *apisServer) authenticateRequest(r *http.Request) (*core.Organization, *core.Workspace, error) {

	if auth, ok := r.Header["Authorization"]; ok {
		// Attempt to read and process the Authorization header.
		if len(auth) > 1 {
			return nil, nil, errors.BadRequest("request contains multiple Authorization headers")
		}
		token, found := validation.ParseBearer(auth[0])
		if !found {
			return nil, nil, errors.BadRequest("Authorization header is invalid; it should be in the format 'Authorization: Bearer <YOUR_API_KEY>'")
		}
		token, found = strings.CutPrefix(token, "api_")
		if !found {
			return nil, nil, errors.BadRequest("API key is not valid; API keys start with the api_ prefix")
		}
		organizationID, workspaceID, err := s.core.AccessKey(r.Context(), token, core.AccessKeyTypeAPI)
		if err != nil {
			switch err.(type) {
			case *errors.BadRequestError:
				err = errors.Unauthorized("API key in the Authorization header of the request is malformed")
			case *errors.NotFoundError:
				err = errors.Unauthorized("API key in the Authorization header of the request does not exist")
			}
			return nil, nil, err
		}
		org, err := s.core.Organization(organizationID)
		if err != nil {
			return nil, nil, err
		}
		// If the key is restricted to a workspace, return the workspace as well.
		if workspaceID != "" {
			ws, err := org.Workspace(workspaceID)
			if err != nil {
				return nil, nil, err
			}
			return org, ws, nil
		}
		header, ok := r.Header["Krenalis-Workspace"]
		// If the Krenalis-Workspace header is present, return the workspace as well.
		if !ok {
			return org, nil, nil
		}
		if len(header) > 1 {
			return nil, nil, errors.BadRequest("request contains multiple Krenalis-Workspace headers")
		}
		id := header[0]
		if !core.IsValidID(id) {
			return nil, nil, errors.BadRequest("Krenalis-Workspace header is invalid; it should be in the format 'Krenalis-Workspace: <WORKSPACE_ID>'")
		}
		ws, err := org.Workspace(id)
		if err != nil {
			return nil, nil, err
		}
		return org, ws, nil
	}

	org, ws, _, err := s.authenticateAdminRequest(r)
	if err != nil {
		return nil, nil, err
	}

	return org, ws, nil
}

// deleteSessionCookie invalidates the session by removing the session cookie.
func (s *apisServer) deleteSessionCookie(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie(sessionCookieName)
	if cookie == nil {
		return
	}
	// Remove the session.
	c := &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		Secure:   s.runsOnHTTPS,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
	writeSessionCookie(w, c)
}

// forwardSentryError forwards a telemetry error from a client to Sentry.
// If not authorized, this method does nothing.
func (s *apisServer) forwardSentryError(w http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, _, err := s.authenticateAdminRequest(r); err != nil {
		return nil, err
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	s.sentryTelemetry.errorTunnel.ServeHTTP(w, r)
	return nil, nil
}

// login logs a user in.
//
// If WorkOS is configured, it verifies the WorkOS access token, retrieves the
// member by WorkOS user ID, auto-provisions them if they are not already
// registered in Krenalis, and sets the same encrypted session cookie as the
// native login.
//
// If workOS is not configured, it uses Krenalis native email and password
// login.
func (s *apisServer) login(w http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateRequiredBody(w, r, false); err != nil {
		return nil, err
	}

	var org *core.Organization
	var memberID string
	if s.workos == nil {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			IsUnique bool   `json:"isUnique"`
		}
		err := json.Decode(r.Body, &body)
		if err != nil {
			return nil, errors.BadRequest("")
		}

		// Retrieve the organization and the member.
		organizations, _ := s.core.Organizations(core.SortByName, 0, 2)
		if len(organizations) == 0 {
			return nil, errors.New("there are no organizations")
		}
		if len(organizations) > 1 {
			return nil, errors.New("there is more than one organization")
		}
		org = organizations[0]
		memberID, err = org.AuthenticateMember(r.Context(), body.Email, body.Password)
		if err != nil {
			if err, ok := err.(*errors.UnprocessableError); ok && err.Code == core.AuthenticationFailed {
				return []any{"", "AuthenticationFailed"}, nil
			}
			return nil, err
		}

		if body.IsUnique {
			members, err := org.Members(r.Context())
			if err != nil {
				return nil, err
			}
			if len(members) > 1 {
				return []any{"", "AuthenticationFailed"}, nil
			}
		}
	} else {
		var body struct {
			AccessToken string `json:"accessToken"`
		}
		err := json.Decode(r.Body, &body)
		if err != nil || body.AccessToken == "" {
			return nil, errors.BadRequest("")
		}

		workosUser, err := s.workos.Authenticate(r.Context(), body.AccessToken)
		if err != nil {
			if errors.Is(err, workos.ErrInvalidToken) {
				return nil, errors.Unauthorized("invalid WorkOS token")
			}
			return nil, err
		}

		email := strings.TrimSpace(norm.NFC.String(workosUser.Email))
		firstName := strings.TrimSpace(norm.NFC.String(workosUser.FirstName))
		lastName := strings.TrimSpace(norm.NFC.String(workosUser.LastName))

		org, err = s.core.Organization(workosUser.OrganizationExternalID)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				slog.Error("WorkOS user authenticated but no matching Krenalis organization found",
					"workos_user_id", workosUser.ID,
					"organization_id", workosUser.OrganizationExternalID,
				)
				return nil, errors.Unauthorized("invalid organization ID in WorkOS token")
			}
			return nil, err
		}

		memberID, err = org.MemberByWorkOSID(r.Context(), workosUser.ID)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); !ok {
				return nil, err
			}
			name := strings.TrimSpace(firstName + " " + lastName)
			memberID, err = org.ProvisionMemberFromWorkOS(r.Context(), name, email, workosUser.ID)
			if err != nil {
				if e, ok := err.(*errors.UnprocessableError); ok && e.Code == core.MemberEmailExists {
					memberID, err = org.MemberByWorkOSID(r.Context(), workosUser.ID)
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			}
		}
	}

	// Store the session.
	sc := &sessionCookie{Organization: org.ID, Member: memberID}
	se, err := s.secureCookie(r.Context())
	if err != nil {
		return nil, err
	}
	value, err := se.Encode(sessionCookieName, sc)
	if err != nil {
		return nil, err
	}

	c := &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     sessionCookiePath,
		Secure:   s.runsOnHTTPS,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	writeSessionCookie(w, c)

	return []any{memberID, nil}, nil
}

// logout logs the user out.
func (s *apisServer) logout(w http.ResponseWriter, r *http.Request) (any, error) {
	if err := validateForbiddenBody(r); err != nil {
		return nil, err
	}
	s.deleteSessionCookie(w, r)
	return nil, nil
}

// handleWorkOSAction handles the WorkOS user-registration Action. It verifies
// the request signature and denies registration if the email the user is
// registering with does not match the email on the WorkOS invitation.
func (s *apisServer) handleWorkOSAction(w http.ResponseWriter, r *http.Request) (any, error) {
	if s.workos == nil {
		return nil, errors.Unauthorized("WorkOS is not configured")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxWorkOSPayloadSize)
	rawBody, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			return nil, errors.BadRequest("request body too large")
		}
		return nil, errors.BadRequest("failed to read request body")
	}

	sigHeader := r.Header.Get("WorkOS-Signature")
	if sigHeader == "" {
		return nil, errors.Unauthorized("WorkOS action is missing the signature header")
	}
	if err := s.workos.VerifyActionSignature(rawBody, sigHeader); err != nil {
		return nil, errors.Unauthorized("invalid WorkOS action signature")
	}

	var action struct {
		UserData struct {
			Email string `json:"email"`
		} `json:"user_data"`
		Invitation *struct {
			Email string `json:"email"`
		} `json:"invitation"`
	}
	normalizedBody := norm.NFC.Bytes(rawBody)
	if err := json.Unmarshal(normalizedBody, &action); err != nil {
		return nil, errors.BadRequest("invalid action payload")
	}

	verdict, message := "Deny", "Registration is by invitation only."

	if action.Invitation != nil {
		if strings.EqualFold(action.UserData.Email, action.Invitation.Email) {
			verdict, message = "Allow", ""
		} else {
			message = "You must register with the email address you were invited with."
		}
	}

	responseJSON, err := s.workos.BuildActionResponse(verdict, message)
	if err != nil {
		return nil, err
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(responseJSON)
	return nil, nil
}

// handleWorkOSWebhook handles incoming WorkOS webhook events.
func (s *apisServer) handleWorkOSWebhook(w http.ResponseWriter, r *http.Request) (any, error) {
	if s.workos == nil {
		return nil, errors.Unauthorized("WorkOS is not configured")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxWorkOSPayloadSize)
	rawBody, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			return nil, errors.BadRequest("request body too large")
		}
		return nil, errors.BadRequest("failed to read request body")
	}

	sigHeader := r.Header.Get("WorkOS-Signature")
	if sigHeader == "" {
		return nil, errors.Unauthorized("WorkOS webhook is missing the signature header")
	}
	err = s.workos.VerifyWebhookSignature(rawBody, sigHeader)
	if err != nil {
		return nil, errors.Unauthorized("invalid WorkOS webhook signature")
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
			ID             string  `json:"id"`
			Email          string  `json:"email"`
			FirstName      string  `json:"first_name"`
			LastName       string  `json:"last_name"`
			Name           string  `json:"name"`
			ExternalID     *string `json:"external_id"`
			UserID         string  `json:"user_id"`
			OrganizationID string  `json:"organization_id"`
		} `json:"data"`
	}
	normalizedBody := norm.NFC.Bytes(rawBody)
	if err := json.Unmarshal(normalizedBody, &event); err != nil {
		return nil, errors.BadRequest("invalid webhook payload")
	}

	switch event.Event {
	case "user.updated":
		name := strings.TrimSpace(event.Data.FirstName + " " + event.Data.LastName)
		if event.Data.ID == "" || event.Data.Email == "" {
			return nil, nil
		}
		if runes := []rune(name); len(runes) > 255 {
			name = string(runes[:255])
		}
		if err := s.core.UpdateMembersByWorkOSID(r.Context(), event.Data.ID, name, event.Data.Email); err != nil {
			if e, ok := err.(*errors.UnprocessableError); ok && e.Code == core.MemberEmailExists {
				// Email already in use, skip the update without returning
				// errors to prevent webhook retries.
				return nil, nil
			}
			return nil, err
		}
	case "user.deleted":
		if event.Data.ID == "" {
			return nil, nil
		}
		if err := s.core.DeleteMembersByWorkOSID(r.Context(), event.Data.ID); err != nil {
			return nil, err
		}
	case "organization.updated":
		if event.Data.ExternalID == nil || *event.Data.ExternalID == "" {
			return nil, nil
		}
		orgID := *event.Data.ExternalID
		orgName := event.Data.Name
		if orgName == "" {
			return nil, nil
		}
		if runes := []rune(orgName); len(runes) > 255 {
			orgName = string(runes[:255])
		}
		org, err := s.core.Organization(orgID)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				return nil, nil
			}
			return nil, err
		}
		if err := org.Update(r.Context(), orgName); err != nil {
			return nil, err
		}
	case "organization_membership.created":
		if event.Data.UserID == "" || event.Data.OrganizationID == "" {
			return nil, nil
		}
		workosUser, err := s.workos.User(r.Context(), event.Data.UserID)
		if err != nil {
			return nil, err
		}
		orgID, err := s.workos.OrganizationExternalID(r.Context(), event.Data.OrganizationID)
		if err != nil {
			if errors.Is(err, workos.ErrOrganizationNotLinked) {
				return nil, nil
			}
			return nil, err
		}
		org, err := s.core.Organization(orgID)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				return nil, nil
			}
			return nil, err
		}
		email := strings.TrimSpace(norm.NFC.String(workosUser.Email))
		firstName := strings.TrimSpace(norm.NFC.String(workosUser.FirstName))
		lastName := strings.TrimSpace(norm.NFC.String(workosUser.LastName))
		name := strings.TrimSpace(firstName + " " + lastName)
		if runes := []rune(name); len(runes) > 255 {
			name = string(runes[:255])
		}
		if _, err = org.ProvisionMemberFromWorkOS(r.Context(), name, email, event.Data.UserID); err != nil {
			if e, ok := err.(*errors.UnprocessableError); ok && e.Code == core.MemberEmailExists {
				return nil, nil
			}
			return nil, err
		}
	case "organization_membership.deleted":
		if event.Data.UserID == "" || event.Data.OrganizationID == "" {
			return nil, nil
		}
		orgID, err := s.workos.OrganizationExternalID(r.Context(), event.Data.OrganizationID)
		if err != nil {
			if errors.Is(err, workos.ErrOrganizationNotLinked) {
				return nil, nil
			}
			return nil, err
		}
		org, err := s.core.Organization(orgID)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				return nil, nil
			}
			return nil, err
		}
		memberID, err := org.MemberByWorkOSID(r.Context(), event.Data.UserID)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				return nil, nil
			}
			return nil, err
		}
		if err := org.DeleteMember(r.Context(), memberID); err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				return nil, nil
			}
			return nil, err
		}
	}

	return nil, nil
}

// secureCookie returns the *securecookie.SecureCookie instance.
func (s *apisServer) secureCookie(ctx context.Context) (*securecookie.SecureCookie, error) {
	s.cookies.Lock()
	defer s.cookies.Unlock()
	if s.cookies.SecureCookie != nil {
		return s.cookies.SecureCookie, nil
	}
	hashKey, blockKey, err := s.cookieKeys(ctx)
	if err != nil {
		return nil, err
	}
	s.cookies.SecureCookie = securecookie.New(hashKey, blockKey)
	s.cookies.SecureCookie.MaxAge(sessionMaxAge)
	return s.cookies.SecureCookie, nil
}

// validateForbiddenBody rejects requests that contain a request body.
func validateForbiddenBody(r *http.Request) error {
	if r.ContentLength == 0 {
		return nil
	}
	if r.ContentLength > 0 {
		return errors.BadRequest("request body not allowed")
	}
	var b [1]byte
	n, err := r.Body.Read(b[:])
	if err == io.EOF && n == 0 {
		return nil
	}
	if err != nil && !(err == io.EOF && n > 0) {
		return err
	}
	if n > 0 {
		// Put back the consumed byte to keep the body stream intact for downstream logging/handlers.
		r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(b[:n]), r.Body))
		return errors.BadRequest("request body not allowed")
	}
	return nil
}

// validateRequiredBody validates that the request body is present and is JSON,
// then applies size limiting and NFC normalization.
// If allowPlainText is true, it also allows "text/plain" as a content type.
func validateRequiredBody(w http.ResponseWriter, r *http.Request, allowPlainText bool) error {
	if r.ContentLength == 0 {
		return errors.BadRequest("request's body is missing")
	}
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" && (!allowPlainText || mt != "text/plain") {
		return errors.BadRequest("request's content type must be 'application/json'")
	}
	for k := range params {
		if strings.ToLower(k) != "charset" {
			return errors.BadRequest("request's content type must be 'application/json'")
		}
	}
	if charset, ok := params["charset"]; ok && strings.ToLower(charset) != "utf-8" {
		return errors.BadRequest("request's content type charset must be 'utf-8'")
	}
	b := http.MaxBytesReader(w, r.Body, maxRequestSize)
	r.Body = struct {
		io.Reader
		io.Closer
	}{
		Reader: norm.NFC.Reader(b),
		Closer: b,
	}
	return nil
}

// writeSessionCookie writes a session cookie on w. If one already exists,
// it replaces its value with c's value.
func writeSessionCookie(w http.ResponseWriter, c *http.Cookie) {
	v := c.String()
	if v == "" {
		return
	}
	header := w.Header()
	if cookies, ok := header["Set-Cookie"]; ok {
		const prefix = sessionCookieName + "="
		for i, cookie := range cookies {
			if strings.HasPrefix(cookie, prefix) {
				cookies[i] = v + "; Priority=High"
				return
			}
		}
	}
	header.Add("Set-Cookie", v+"; Priority=High")
}

type bodyWriter struct {
	w *bufio.Writer
}

func newBodyWriter(w io.Writer) bodyWriter {
	return bodyWriter{w: bufio.NewWriter(w)}
}

func (bw bodyWriter) availableBuffer() []byte {
	return bw.w.AvailableBuffer()
}

func (bw bodyWriter) flush() {
	_ = bw.w.Flush()
}

func (bw bodyWriter) write(p []byte) {
	_, _ = bw.w.Write(p)
}

func (bw bodyWriter) writeByte(c byte) {
	_ = bw.w.WriteByte(c)
}

func (bw bodyWriter) writeString(s string) {
	_, _ = bw.w.WriteString(s)
}
