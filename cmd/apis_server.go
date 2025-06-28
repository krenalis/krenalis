//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/json"

	"github.com/gorilla/securecookie"
	"golang.org/x/text/unicode/norm"
)

// maxRequestSize is the maximum size bytes for an API request.
const maxRequestSize = 500 * 1024

//go:embed invite-member-email.html
var inviteMemberEmail string

//go:embed reset-password-email.html
var resetPasswordEmail string

// sessionMaxAge contains the max age property for the session cookie (6 hours).
const sessionMaxAge = 6 * 60 * 60

// LoginRequired is the error code returned by the API when login is required.
const LoginRequired errors.Code = "LoginRequired"

type sessionCookie struct {
	Organization int
	Member       int
}

const (
	sessionCookieName = "meergo_session"
	sessionCookiePath = "/api/"
)

type apisServer struct {
	core                        *core.Core
	secureCookie                *securecookie.SecureCookie // secureCookie contains keys to encrypt/decrypt/remove the session cookie.
	mux                         *http.ServeMux
	runsOnHTTPS                 bool
	javaScriptSDKURL            string
	eventURL                    string
	externalURL                 string
	skipMemberEmailVerification bool
	sentryTelemetry             struct {
		level       core.TelemetryLevel
		errorTunnel *sentryErrorTunnel
	}
}

// newAPIsServer returns an APIs server that handles requests for the given
// Core.
// runsOnHTTPs indicates if the server runs on HTTPS.
func newAPIsServer(core *core.Core, runsOnHTTPS bool,
	javaScriptSDKURL, eventURL string, externalURL string, skipMemberEmailVerification bool,
	sentryTelemetryLevel core.TelemetryLevel, sentryErrorTunnel *sentryErrorTunnel) *apisServer {

	s := &apisServer{
		core:                        core,
		runsOnHTTPS:                 runsOnHTTPS,
		javaScriptSDKURL:            javaScriptSDKURL,
		eventURL:                    eventURL,
		externalURL:                 externalURL,
		skipMemberEmailVerification: skipMemberEmailVerification,
	}
	s.sentryTelemetry.level = sentryTelemetryLevel
	s.sentryTelemetry.errorTunnel = sentryErrorTunnel

	encryptionKey := core.EncryptionKey()
	hashKey, blockKey := encryptionKey[:32], encryptionKey[32:]
	s.secureCookie = securecookie.New(hashKey, blockKey)
	s.secureCookie.MaxAge(sessionMaxAge)

	api := api{s}
	connector := connector{s}
	organization := organization{s}
	workspace := workspace{s}
	connection := connection{s}
	action := action{s}

	paths := map[string]func(w http.ResponseWriter, r *http.Request) (any, error){
		"DELETE /actions/{id}":                                   action.Delete,
		"DELETE /connections/{id}":                               connection.Delete,
		"DELETE /connections/{id}/event-write-keys/{key}":        connection.DeleteEventWriteKey,
		"DELETE /connections/{src}/links/{dst}":                  connection.UnlinkConnection,
		"DELETE /events/listeners/{id}":                          workspace.DeleteEventListener,
		"DELETE /keys/{key}":                                     organization.DeleteAPIKey, /* only admin */
		"DELETE /members/{id}":                                   organization.DeleteMember, /* only admin */
		"DELETE /workspaces/current":                             workspace.Delete,
		"GET    /actions/errors/{start}/{end}":                   workspace.ActionErrors,
		"GET    /actions/executions":                             workspace.Executions,
		"GET    /actions/executions/{id}":                        workspace.Execution,
		"GET    /actions/metrics/dates/{start}/{end}":            workspace.ActionMetricsPerDate,
		"GET    /actions/metrics/days/{days}":                    workspace.ActionMetricsPerDay,
		"GET    /actions/metrics/hours/{hours}":                  workspace.ActionMetricsPerHour,
		"GET    /actions/metrics/minutes/{minutes}":              workspace.ActionMetricsPerMinute,
		"GET    /actions/{id}":                                   workspace.Action,
		"GET    /connections":                                    workspace.Connections,
		"GET    /connections/auth-token":                         workspace.AuthToken,
		"GET    /connections/auth-url":                           connector.AuthCodeURL,
		"GET    /connections/{id}":                               workspace.Connection,
		"GET    /connections/{id}/action-types":                  connection.ActionTypes,   /* only admin */
		"GET    /connections/{id}/actions/schemas/Events/{type}": connection.ActionSchemas, /* only admin */
		"GET    /connections/{id}/actions/schemas/{target}":      connection.ActionSchemas, /* only admin */
		"GET    /connections/{id}/event-write-keys":              connection.EventWriteKeys,
		"GET    /connections/{id}/files/{path}":                  connection.File,
		"GET    /connections/{id}/files/{path}/absolute":         connection.AbsolutePath,
		"GET    /connections/{id}/files/{path}/sheets":           connection.Sheets,
		"GET    /connections/{id}/schemas/event/{type}":          connection.AppEventSchema,
		"GET    /connections/{id}/schemas/user":                  connection.AppUserSchemas,
		"GET    /connections/{id}/tables/{name}":                 connection.TableSchema,
		"GET    /connections/{id}/ui":                            connection.ServeUI, /* only admin */
		"GET    /connections/{id}/users":                         connection.AppUsers,
		"GET    /connectors":                                     api.Connectors,
		"GET    /connectors/{name}":                              api.Connector,
		"GET    /connectors/{name}/documentation":                api.ConnectorDocumentation,
		"GET    /event-url":                                      api.EventURL,
		"GET    /events":                                         workspace.Events,
		"GET    /events/listeners/{id}":                          workspace.ListenedEvents,
		"GET    /events/schema":                                  api.EventSchema,
		"GET    /events/settings/{write_key}":                    api.EventsSettings,
		"GET    /identity-resolution/latest":                     workspace.LatestIdentityResolution,
		"GET    /identity-resolution/settings":                   workspace.IdentityResolutionSettings,
		"GET    /installation-id":                                api.InstallationID,                   /* only admin */
		"GET    /javascript-sdk-url":                             api.JavaScriptSDKURL,                 /* only admin */
		"GET    /keys":                                           organization.APIKeys,                 /* only admin */
		"GET    /members":                                        organization.Members,                 /* only admin */
		"GET    /members/current":                                api.Member,                           /* only admin */
		"GET    /members/invitations/{token}":                    api.MemberInvitation,                 /* only admin */
		"GET    /members/reset-password/{token}":                 api.ValidateMemberPasswordResetToken, /* only admin */
		"GET    /skip-member-email-verification":                 api.SkipMemberEmailVerification,      /* only admin */
		"GET    /telemetry/level":                                api.SentryTelemetryLevel,             /* only admin */
		"GET    /transformation-languages":                       api.TransformationLanguages,
		"GET    /users":                                          workspace.Users,
		"GET    /users/schema":                                   workspace.UserSchema,
		"GET    /users/schema/latest-alter":                      workspace.LatestAlterUserSchema,
		"GET    /users/schema/suitable-as-identifiers":           workspace.UserPropertiesSuitableAsIdentifiers, /* only admin */
		"GET    /users/{id}/events":                              workspace.UserEvents,
		"GET    /users/{id}/identities":                          workspace.Identities,
		"GET    /users/{id}/traits":                              workspace.Traits,
		"GET    /warehouse":                                      workspace.Warehouse,
		"GET    /warehouse/types":                                api.WarehouseTypes,
		"GET    /workspaces":                                     organization.Workspaces,
		"GET    /workspaces/current":                             organization.Workspace,
		"POST   /actions":                                        connection.CreateAction,
		"POST   /actions/{id}/exec":                              action.Execute,
		"POST   /actions/{id}/ui-event":                          action.ServeUI, /* only admin */
		"POST   /connections":                                    workspace.CreateConnection,
		"POST   /connections/{id}/event-write-keys":              connection.CreateEventWriteKey,
		"POST   /connections/{id}/identities":                    connection.Identities,
		"POST   /connections/{id}/preview-send-event":            connection.PreviewSendEvent,
		"POST   /connections/{id}/query":                         connection.ExecQuery,
		"POST   /connections/{id}/ui-event":                      connection.ServeUI, /* only admin */
		"POST   /connections/{src}/links/{dst}":                  connection.LinkConnection,
		"POST   /events":                                         workspace.IngestEvents,
		"POST   /events/listeners":                               workspace.CreateEventListener,
		"POST   /events/{type}":                                  workspace.IngestEvents,
		"POST   /expressions-properties":                         api.ExpressionsProperties, /* only admin */
		"POST   /identity-resolution/start":                      workspace.StartIdentityResolution,
		"POST   /keys":                                           organization.CreateAPIKey, /* only admin */
		"POST   /members":                                        organization.AddMember,    /* only admin */
		"POST   /members/invitations":                            organization.InviteMember, /* only admin */
		"POST   /members/login":                                  s.login,
		"POST   /members/logout":                                 s.logout,               /* only admin */
		"POST   /sentry/errors":                                  s.forwardSentryError,   /* only admin */
		"POST   /transformations":                                api.TransformData,      /* only admin */
		"POST   /ui":                                             workspace.ServeUI,      /* only admin */
		"POST   /ui-event":                                       workspace.ServeUI,      /* only admin */
		"POST   /validate-expression":                            api.ValidateExpression, /* only admin */
		"POST   /warehouse/repair":                               workspace.RepairWarehouse,
		"POST   /workspaces":                                     organization.CreateWorkspace,
		"POST   /workspaces/test":                                organization.TestWorkspaceCreation,
		"PUT    /actions/{id}":                                   action.Update,
		"PUT    /actions/{id}/schedule":                          action.SetSchedulePeriod,
		"PUT    /actions/{id}/status":                            action.SetStatus,
		"PUT    /connections/{id}":                               connection.Update,
		"PUT    /identity-resolution/settings":                   workspace.UpdateIdentityResolutionSettings,
		"PUT    /keys/{key}":                                     organization.UpdateAPIKey,       /* only admin */
		"PUT    /members/current":                                organization.UpdateMember,       /* only admin */
		"PUT    /members/invitations/{token}":                    api.AcceptInvitation,            /* only admin */
		"PUT    /members/reset-password":                         api.SendMemberPasswordReset,     /* only admin */
		"PUT    /members/reset-password/{token}":                 api.ChangeMemberPasswordByToken, /* only admin */
		"PUT    /users/schema":                                   workspace.AlterUserSchema,
		"PUT    /users/schema/preview":                           workspace.PreviewAlterUserSchema,
		"PUT    /warehouse":                                      workspace.UpdateWarehouse,
		"PUT    /warehouse/mode":                                 workspace.UpdateWarehouseMode,
		"PUT    /warehouse/test":                                 workspace.TestWarehouseUpdate,
		"PUT    /workspaces/current":                             workspace.Update,
	}

	s.mux = http.NewServeMux()
	for path, serve := range paths {
		s.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store, max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			response, err := serve(w, r)
			if err != nil {
				select {
				case <-r.Context().Done():
					return
				default:
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
		})
	}

	return s
}

// ServeHTTP servers the API methods from HTTP.
func (s *apisServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "POST", "PUT":
		if r.URL.Path == "/api/v1/sentry/errors" {
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

	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		http.NotFound(w, r)
		return
	}
	r.URL.Path = r.URL.Path[len("/api/v1"):]
	if r.URL.RawPath != "" {
		r.URL.RawPath = r.URL.RawPath[len("/api/v1"):]
	}

	s.mux.ServeHTTP(w, r)
}

// credentials validates the authorization of the request r, authenticates it,
// and returns the associated organization and workspace.
// The workspace will be nil unless one of the following conditions is met:
//
// - The API key in the request is restricted to a specific workspace.
// - The API key is present and the Meergo-Workspace header is provided.
// - A session cookie is included in the request.
//
// If the request is not authorized, an errors.UnauthorizedError is returned.
func (s *apisServer) credentials(r *http.Request) (*core.Organization, *core.Workspace, error) {

	if auth, ok := r.Header["Authorization"]; ok {
		// Attempt to read and process the Authorization header.
		if len(auth) > 1 {
			return nil, nil, errors.BadRequest("request contains multiple Authorization headers")
		}
		token, found := strings.CutPrefix(auth[0], "Bearer ")
		if !found || token == "" {
			return nil, nil, errors.BadRequest("Authorization header is invalid. It should be in the format 'Authorization: Bearer <YOUR_API_KEY>'.")
		}
		organizationID, workspaceID, found := s.core.APIKey(token)
		if !found {
			return nil, nil, errors.Unauthorized("API key in the Authorization header of the request does not exist")
		}
		org, err := s.core.Organization(r.Context(), organizationID)
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
			return nil, nil, errors.BadRequest("request contains multiple Meergo-Warehouse headers")
		}
		var id int64
		if header[0] != "" && header[0][0] != '+' {
			id, _ = strconv.ParseInt(header[0], 10, 32)
		}
		if id <= 0 {
			return nil, nil, errors.BadRequest("Meergo-Workspace header is invalid. It should be in the format 'Meergo-Workspace: <WORKSPACE_ID>'")
		}
		ws, err := org.Workspace(int(id))
		if err != nil {
			return nil, nil, err
		}
		return org, ws, nil
	}

	org, _, err := s.memberCredentials(r)
	if err != nil {
		return nil, nil, err
	}
	header, ok := r.Header["Meergo-Workspace"]
	if !ok {
		return org, nil, nil
	}
	if len(header) > 1 {
		return nil, nil, errors.BadRequest("request contains multiple Meergo-Warehouse headers")
	}
	var workspaceID int64
	if header[0] != "" && header[0][0] != '+' {
		workspaceID, _ = strconv.ParseInt(header[0], 10, 32)
	}
	if workspaceID <= 0 {
		return nil, nil, errors.BadRequest("Meergo-Workspace header is invalid. It should be in the format 'Meergo-Workspace: <WORKSPACE_ID>'")
	}
	ws, err := org.Workspace(int(workspaceID))
	if err != nil {
		return nil, nil, err
	}

	return org, ws, nil
}

var errInvalidSessionCookie = errors.Unauthorized("session cookie has expired or is no longer valid")

// forwardSentryError forwards a telemetry error from a client to Sentry.
// If the user is not logged in, this method does nothing.
func (s *apisServer) forwardSentryError(w http.ResponseWriter, r *http.Request) (any, error) {
	// Check if the user is logged. If not, discard the reported errors.
	organization := organization{s}
	_, _, err := organization.memberCredentials(r)
	if err != nil {
		if _, ok := err.(*errors.UnauthorizedError); ok {
			return nil, nil
		}
		return nil, err
	}
	s.sentryTelemetry.errorTunnel.ServeHTTP(w, r)
	return nil, nil
}

// memberCredentials is like credentials but only accepts a session cookie.
// It returns the associated organization and member.
//
// If the request is not authorized, an errors.UnauthorizedError is returned.
func (s *apisServer) memberCredentials(r *http.Request) (*core.Organization, *core.Member, error) {

	// Get the session.
	cookie, _ := r.Cookie(sessionCookieName)
	if cookie == nil {
		return nil, nil, errors.Unauthorized("the Authorization header with the API key is not present in the request")
	}
	session := &sessionCookie{}
	err := s.secureCookie.Decode(sessionCookieName, cookie.Value, session)
	if err != nil {
		return nil, nil, errInvalidSessionCookie
	}

	// Get the organization.
	organization, err := s.core.Organization(r.Context(), session.Organization)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			return nil, nil, errInvalidSessionCookie
		}
		return nil, nil, err
	}

	// Get the member.
	member, err := organization.Member(r.Context(), session.Member)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			err = errInvalidSessionCookie
		}
		return nil, nil, err
	}

	return organization, member, nil
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
	organization, err := s.core.Organization(r.Context(), 1)
	if err != nil {
		return nil, fmt.Errorf("cannot read organization: %s", err)
	}
	memberID, err := organization.AuthenticateMember(r.Context(), body.Email, body.Password)
	if err != nil {
		if err, ok := err.(*errors.UnprocessableError); ok && err.Code == core.AuthenticationFailed {
			return []any{0, "AuthenticationFailed"}, nil
		}
		return nil, err
	}

	if body.IsUnique {
		members, err := organization.Members(r.Context())
		if err != nil {
			return nil, err
		}
		if len(members) > 1 {
			return []any{0, "AuthenticationFailed"}, nil
		}
	}

	// Store the session.
	sc := &sessionCookie{Organization: organization.ID, Member: memberID}
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
	header := w.Header()
	if v := c.String(); v != "" {
		if cookies, ok := header["Set-Cookie"]; ok {
			prefix := sessionCookieName + "="
			for i, cookie := range cookies {
				if strings.HasPrefix(cookie, prefix) {
					cookies[i] = v + "; Priority=High"
					return nil, nil
				}
			}
		}
		header.Add("Set-Cookie", v+"; Priority=High")
	}

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
	header := w.Header()
	if v := c.String(); v != "" {
		if cookies, ok := header["Set-Cookie"]; ok {
			prefix := sessionCookieName + "="
			for i, cookie := range cookies {
				if strings.HasPrefix(cookie, prefix) {
					cookies[i] = v + "; Priority=High"
					return nil, nil
				}
			}
		}
		header.Add("Set-Cookie", v+"; Priority=High")
	}
	return nil, nil
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
