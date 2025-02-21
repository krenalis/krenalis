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

// sessionMaxAge contains the max age property for the session cookie (6 hours).
const sessionMaxAge = 6 * 60 * 60

// LoginRequired is the error code returned by the API when login is required.
const LoginRequired errors.Code = "LoginRequired"

type sessionCookie struct {
	Organization int
	Member       int
}

const (
	sessionCookieName = "api"
	sessionCookiePath = "/api/"
)

type apisServer struct {
	core         *core.Core
	secureCookie *securecookie.SecureCookie // secureCookie contains keys to encrypt/decrypt/remove the session cookie.
	mux          *http.ServeMux
	runsOnHTTPS  bool
}

// newAPIsServer returns an APIs server that handles requests for the given
// Core. encryptionKey is the key used to encrypt the session cookie.
// runsOnHTTPs indicates if the server runs on HTTPS.
// It panics if the session key is not at least 64 bytes long.
func newAPIsServer(core *core.Core, encryptionKey []byte, runsOnHTTPS bool) *apisServer {

	if len(encryptionKey) != 64 {
		panic("encryptionKey is not 64 bytes long")
	}

	s := &apisServer{core: core, runsOnHTTPS: runsOnHTTPS}

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
		"DELETE /keys/{key}":                                     organization.DeleteAPIKey, /* only UI */
		"DELETE /members/{id}":                                   organization.DeleteMember, /* only UI */
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
		"GET    /connections/{id}/action-types":                  connection.ActionTypes,   /* only UI */
		"GET    /connections/{id}/actions/schemas/Events/{type}": connection.ActionSchemas, /* only UI */
		"GET    /connections/{id}/actions/schemas/{target}":      connection.ActionSchemas, /* only UI */
		"GET    /connections/{id}/event-write-keys":              connection.EventWriteKeys,
		"GET    /connections/{id}/files/{path}/absolute":         connection.AbsolutePath,
		"GET    /connections/{id}/schemas/event/{type}":          connection.AppEventSchema,
		"GET    /connections/{id}/schemas/user":                  connection.AppUserSchemas,
		"GET    /connections/{id}/tables/{name}":                 connection.TableSchema,
		"GET    /connections/{id}/ui":                            connection.ServeUI, /* only UI */
		"GET    /connectors":                                     api.Connectors,
		"GET    /connectors/{name}":                              api.Connector,
		"GET    /events/listeners/{id}":                          workspace.ListenedEvents,
		"GET    /events/schema":                                  api.EventSchema,
		"GET    /events/settings/{write_key}":                    api.EventsSettings,
		"GET    /identifiers-schema":                             workspace.IdentifiersSchema,
		"GET    /identity-resolution/latest":                     workspace.LatestIdentityResolution,
		"GET    /identity-resolution/settings":                   workspace.IdentityResolutionSettings,
		"GET    /keys":                                           organization.APIKeys, /* only UI */
		"GET    /members":                                        organization.Members, /* only UI */
		"GET    /members/current":                                api.Member,           /* only UI */
		"GET    /members/invitations/{token}":                    api.MemberInvitation, /* only UI */
		"GET    /transformation-languages":                       api.TransformationLanguages,
		"GET    /users/schema":                                   workspace.UserSchema,
		"GET    /users/{id}/events":                              workspace.UserEvents,
		"GET    /users/{id}/identities":                          workspace.Identities,
		"GET    /users/{id}/traits":                              workspace.Traits,
		"GET    /warehouse":                                      workspace.Warehouse,
		"GET    /warehouse/types":                                api.WarehouseTypes,
		"GET    /workspaces":                                     organization.Workspaces,
		"GET    /workspaces/current":                             organization.Workspace,
		"POST   /actions":                                        connection.CreateAction,
		"POST   /actions/{id}/exec":                              action.Execute,
		"POST   /actions/{id}/ui-event":                          action.ServeUI, /* only UI */
		"POST   /connections":                                    workspace.CreateConnection,
		"POST   /connections/{id}/event-write-keys":              connection.CreateEventWriteKey,
		"POST   /connections/{id}/files/{path}":                  connection.File,
		"POST   /connections/{id}/files/{path}/sheets":           connection.Sheets,
		"POST   /connections/{id}/identities":                    connection.Identities,
		"POST   /connections/{id}/preview-send-event":            connection.PreviewSendEvent,
		"POST   /connections/{id}/query":                         connection.ExecQuery,
		"POST   /connections/{id}/ui-event":                      connection.ServeUI, /* only UI */
		"POST   /connections/{id}/users":                         connection.AppUsers,
		"POST   /connections/{src}/links/{dst}":                  connection.LinkConnection,
		"POST   /events":                                         workspace.IngestEvents,
		"POST   /events/retrive":                                 workspace.Events,
		"POST   /events/listeners":                               workspace.CreateEventListener,
		"POST   /events/{type}":                                  workspace.IngestEvents,
		"POST   /expressions-properties":                         api.ExpressionsProperties, /* only UI */
		"POST   /identity-resolution/start":                      workspace.StartIdentityResolution,
		"POST   /keys":                                           organization.CreateAPIKey, /* only UI */
		"POST   /members/invitations":                            organization.InviteMember, /* only UI */
		"POST   /members/login":                                  s.login,                   /* only UI */
		"POST   /members/logout":                                 s.logout,                  /* only UI */
		"POST   /transformations":                                api.TransformData,         /* only UI */
		"POST   /ui":                                             workspace.ServeUI,         /* only UI */
		"POST   /ui-event":                                       workspace.ServeUI,         /* only UI */
		"POST   /users":                                          workspace.Users,
		"POST   /validate-expression":                            api.ValidateExpression, /* only UI */
		"POST   /warehouse/repair":                               workspace.RepairWarehouse,
		"POST   /workspaces":                                     organization.CreateWorkspace,
		"POST   /workspaces/test":                                organization.TestWorkspaceCreation,
		"PUT    /actions/{id}":                                   action.Update,
		"PUT    /actions/{id}/schedule":                          action.SetSchedulePeriod,
		"PUT    /actions/{id}/status":                            action.SetStatus,
		"PUT    /connections/{id}":                               connection.Update,
		"PUT    /identity-resolution/settings":                   workspace.UpdateIdentityResolutionSettings,
		"PUT    /keys/{key}":                                     organization.UpdateAPIKey, /* only UI */
		"PUT    /members/current":                                organization.UpdateMember, /* only UI */
		"PUT    /members/invitations/{token}":                    api.AcceptInvitation,      /* only UI */
		"PUT    /users/schema":                                   workspace.UpdateUserSchema,
		"PUT    /users/schema/preview":                           workspace.PreviewUserSchemaUpdate,
		"PUT    /warehouse":                                      workspace.UpdateWarehouse,
		"PUT    /warehouse/mode":                                 workspace.UpdateWarehouseMode,
		"PUT    /warehouse/test":                                 workspace.TestWarehouseUpdate,
		"PUT    /workspaces/current":                             workspace.Update,
	}

	s.mux = http.NewServeMux()
	for path, serve := range paths {
		s.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			response, err := serve(w, r)
			if err != nil {
				select {
				case <-r.Context().Done():
					return
				default:
				}
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				slog.Error("error occurred serving Core", "err", err)
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

	// Store the session.
	sc := &sessionCookie{Organization: organization.ID, Member: memberID}
	value, err := s.secureCookie.Encode(sessionCookieName, sc)
	if err != nil {
		return nil, err
	}

	c := &http.Cookie{
		Name:  sessionCookieName,
		Value: value,
		Path:  sessionCookiePath,
		// TODO(Gianluca): disabling Secure for HTTP connections is necessary
		// because we currently have only a loing mecanism based on cookies,
		// that would not work on HTTP connections.
		// See the issue https://github.com/meergo/meergo/issues/1153.
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
		Name:  sessionCookieName,
		Value: "",
		Path:  sessionCookiePath,
		// TODO(Gianluca): disabling Secure for HTTP connections is necessary
		// because we currently have only a loing mecanism based on cookies,
		// that would not work on HTTP connections.
		// See the issue https://github.com/meergo/meergo/issues/1153.
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
