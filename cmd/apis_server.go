// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"bufio"
	"io"
	"log/slog"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/tools/errors"
	"github.com/meergo/meergo/tools/json"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"golang.org/x/text/unicode/norm"
)

// maxRequestSize is the maximum size bytes for an API request.
const maxRequestSize = 500 * 1024

var newline = []byte("\n")

var (
	// errMissingWorkspace is returned when the "Meergo-Workspace" header is required but not provided.
	errMissingWorkspace = errors.Forbidden("Meergo-Workspace header is missing; provide it in the format 'Meergo-Workspace: <WORKSPACE_ID>'")

	// errInvalidSessionCookie is returned when a session cookie has expired or is no lo longer valid.
	errInvalidSessionCookie = errors.Unauthorized("session cookie has expired or is no longer valid")
)

// sessionMaxAge contains the max age property for the session cookie (6 hours).
const sessionMaxAge = 6 * 60 * 60

type sessionCookie struct {
	Organization uuid.UUID
	Member       int
}

const (
	sessionCookieName = "meergo_session"
	sessionCookiePath = "/v1/"
)

type apisServer struct {
	core                            *core.Core
	secureCookie                    *securecookie.SecureCookie // secureCookie contains keys to encrypt/decrypt/remove the session cookie.
	mux                             *http.ServeMux
	runsOnHTTPS                     bool
	javaScriptSDKURL                string
	externalURL                     string
	externalEventURL                string
	externalAssetsURLs              []string
	potentialConnectorsURL          string // must be a valid URL or empty string (which means: do not load the JSON file).
	memberEmailVerificationRequired bool
	sentryTelemetry                 struct {
		level       core.TelemetryLevel
		errorTunnel *sentryErrorTunnel
	}
}

// newAPIsServer returns an APIs server that handles requests for the given
// Core.
// runsOnHTTPs indicates if the server runs on HTTPS.
func newAPIsServer(core *core.Core, runsOnHTTPS bool, javaScriptSDKURL, externalURL,
	externalEventURL string, externalAssetsURLs []string, potentialConnectorsURL string,
	memberEmailVerificationRequired bool, sentryTelemetryLevel core.TelemetryLevel,
	sentryErrorTunnel *sentryErrorTunnel,
) *apisServer {

	s := &apisServer{
		core:                            core,
		runsOnHTTPS:                     runsOnHTTPS,
		javaScriptSDKURL:                javaScriptSDKURL,
		externalURL:                     externalURL,
		externalEventURL:                externalEventURL,
		externalAssetsURLs:              externalAssetsURLs,
		potentialConnectorsURL:          potentialConnectorsURL,
		memberEmailVerificationRequired: memberEmailVerificationRequired,
	}
	s.sentryTelemetry.level = sentryTelemetryLevel
	s.sentryTelemetry.errorTunnel = sentryErrorTunnel

	encryptionKey := core.EncryptionKey()
	hashKey, blockKey := encryptionKey[:32], encryptionKey[32:]
	s.secureCookie = securecookie.New(hashKey, blockKey)
	s.secureCookie.MaxAge(sessionMaxAge)

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
					_, _ = s.logout(w, r)
				}
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				slog.Error("cmd: error occurred serving Core", "err", err)
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

	switch r.Method {
	case "POST", "PUT":
		if r.URL.Path == "/v1/sentry/errors" {
			// In this case, do not validate the content type, because when the
			// Sentry SDK sends user feedback with screenshots attached, the
			// content type is not JSON, and therefore this check would fail,
			// preventing such reports from being sent to Sentry.
		} else {
			// Validate the content type.
			mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || (mt != "application/json" && mt != "text/plain") || len(params) > 1 {
				err := errors.BadRequest("request's content type must be 'application/json'")
				_ = err.WriteTo(w)
				return
			}
			if charset, ok := params["charset"]; ok && strings.ToLower(charset) != "utf-8" {
				err := errors.BadRequest("request's content type charset must be 'utf-8'")
				_ = err.WriteTo(w)
				return
			}
		}
		// Validate the content length.
		if cl := r.Header.Get("Content-Length"); cl != "" {
			length, _ := strconv.Atoi(cl)
			if length < 0 || length > maxRequestSize {
				err := errors.BadRequest("request's content length must be in the range [1,%d]", maxRequestSize)
				_ = err.WriteTo(w)
				return
			}
		}
		lr := &io.LimitedReader{R: r.Body, N: maxRequestSize + 1}
		payload := norm.NFC.Reader(lr)
		r.Body = io.NopCloser(payload)
	}

	if !strings.HasPrefix(r.URL.Path, "/v1/") {
		http.NotFound(w, r)
		return
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
// The workspace is nil when the "Meergo-Workspace" header is absent.
//
// It returns errors.UnauthorizedError if authorization fails, or
// errInvalidSessionCookie if the session cookie is invalid.
func (s *apisServer) authenticateAdminRequest(r *http.Request) (org *core.Organization, ws *core.Workspace, userID int, err error) {

	// Get the session.
	cookie, _ := r.Cookie(sessionCookieName)
	if cookie == nil {
		return nil, nil, 0, errors.Unauthorized("Authorization header with the API key is not present in the request")
	}
	session := &sessionCookie{}
	err = s.secureCookie.Decode(sessionCookieName, cookie.Value, session)
	if err != nil {
		return nil, nil, 0, errInvalidSessionCookie
	}
	if id := session.Member; id < 1 || id > math.MaxInt32 {
		return nil, nil, 0, errInvalidSessionCookie
	}

	// Get the organization.
	org, err = s.core.Organization(session.Organization)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			return nil, nil, 0, errInvalidSessionCookie
		}
		return nil, nil, 0, err
	}
	// Verify that the member still exists.
	if !org.HasMember(session.Member) {
		return nil, nil, 0, errInvalidSessionCookie
	}
	// If the 'Meergo-Workspace' header is missing, return with a nil workspace.
	header, ok := r.Header["Meergo-Workspace"]
	if !ok {
		return org, nil, session.Member, nil
	}
	if len(header) > 1 {
		return nil, nil, 0, errors.BadRequest("request contains multiple Meergo-Workspace headers")
	}
	var workspaceID int64
	if header[0] != "" && header[0][0] != '+' {
		workspaceID, _ = strconv.ParseInt(header[0], 10, 32)
	}
	if workspaceID <= 0 {
		return nil, nil, 0, errors.BadRequest("Meergo-Workspace header is invalid; it should be in the format 'Meergo-Workspace: <WORKSPACE_ID>'")
	}
	ws, err = org.Workspace(int(workspaceID))
	if err != nil {
		return nil, nil, 0, err
	}

	return org, ws, session.Member, nil
}

// authenticateRequest authenticates the request r and returns the associated
// organization and optional workspace.
//
// Authorization sources:
//   - API key in the "Authorization" header
//   - Session cookie from the Admin console
//
// The workspace is nil when the "Meergo-Workspace" header is absent and either:
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
		token, found := strings.CutPrefix(auth[0], "Bearer ")
		if !found || token == "" {
			return nil, nil, errors.BadRequest("Authorization header is invalid; it should be in the format 'Authorization: Bearer <YOUR_API_KEY>'")
		}
		organizationID, workspaceID, found := s.core.AccessKey(token, core.AccessKeyTypeAPI)
		if !found {
			return nil, nil, errors.Unauthorized("API key in the Authorization header of the request does not exist")
		}
		org, err := s.core.Organization(organizationID)
		if err != nil {
			return nil, nil, err
		}
		// If the key is restricted to a workspace, return the workspace as well.
		if workspaceID > 0 {
			ws, err := org.Workspace(workspaceID)
			if err != nil {
				return nil, nil, err
			}
			return org, ws, nil
		}
		header, ok := r.Header["Meergo-Workspace"]
		// If the Meergo-Workspace header is present, return the workspace as well.
		if !ok {
			return org, nil, nil
		}
		if len(header) > 1 {
			return nil, nil, errors.BadRequest("request contains multiple Meergo-Workspace headers")
		}
		var id int64
		if header[0] != "" && header[0][0] != '+' {
			id, _ = strconv.ParseInt(header[0], 10, 32)
		}
		if id <= 0 {
			return nil, nil, errors.BadRequest("Meergo-Workspace header is invalid; it should be in the format 'Meergo-Workspace: <WORKSPACE_ID>'")
		}
		ws, err := org.Workspace(int(id))
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

// forwardSentryError forwards a telemetry error from a client to Sentry.
// If not authorized, this method does nothing.
func (s *apisServer) forwardSentryError(w http.ResponseWriter, r *http.Request) (any, error) {
	if _, _, _, err := s.authenticateAdminRequest(r); err != nil {
		return nil, err
	}
	s.sentryTelemetry.errorTunnel.ServeHTTP(w, r)
	return nil, nil
}

// login logs a user in.
func (s *apisServer) login(w http.ResponseWriter, r *http.Request) (any, error) {

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
	organizations, _ := s.core.Organizations(core.SortByName, 0, 1)
	if len(organizations) == 0 {
		return nil, errors.New("there are no organizations")
	}
	org := organizations[0]
	memberID, err := org.AuthenticateMember(r.Context(), body.Email, body.Password)
	if err != nil {
		if err, ok := err.(*errors.UnprocessableError); ok && err.Code == core.AuthenticationFailed {
			return []any{0, "AuthenticationFailed"}, nil
		}
		return nil, err
	}

	if body.IsUnique {
		members, err := org.Members(r.Context())
		if err != nil {
			return nil, err
		}
		if len(members) > 1 {
			return []any{0, "AuthenticationFailed"}, nil
		}
	}

	// Store the session.
	sc := &sessionCookie{Organization: org.ID, Member: memberID}
	value, err := s.secureCookie.Encode(sessionCookieName, sc)
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
	cookie, _ := r.Cookie(sessionCookieName)
	if cookie == nil {
		return nil, nil
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
	return nil, nil
}

// parseID parses a decimal identifier in the form /^[1-9][0-9]*$/.
// The value must be in the range [1, math.MaxInt32].
// It returns (value, true) if valid, or (0, false) otherwise.
func parseID(s string) (int, bool) {
	if len(s) == 0 || s[0] == '0' {
		return 0, false
	}
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
		if n > math.MaxInt32 {
			return 0, false
		}
	}
	if n < 1 {
		return 0, false
	}
	return int(n), true
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
