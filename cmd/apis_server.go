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
	"net/http"
	"strings"

	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/json"

	"github.com/gorilla/securecookie"
)

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
// Core. sessionKey is the key used to encrypt the session cookie.
// runsOnHTTPs indicates if the server runs on HTTPS.
// It panics if the session key is not at least 64 bytes long.
func newAPIsServer(core *core.Core, sessionKey []byte, runsOnHTTPS bool) *apisServer {

	if len(sessionKey) != 64 {
		panic("sessionKey is not 64 bytes long")
	}

	s := &apisServer{core: core, runsOnHTTPS: runsOnHTTPS}

	hashKey, blockKey := sessionKey[:32], sessionKey[32:]
	s.secureCookie = securecookie.New(hashKey, blockKey)
	s.secureCookie.MaxAge(sessionMaxAge)

	api := api{s}
	connector := connector{s}
	organization := organization{s}
	workspace := workspace{s}
	connection := connection{s}
	action := action{s}

	paths := map[string]func(w http.ResponseWriter, r *http.Request) (any, error){
		"DELETE /api/keys/{key}":                                                                         organization.DeleteAPIKey, /* only UI */
		"DELETE /api/members/{member}":                                                                   organization.DeleteMember, /* only UI */
		"DELETE /api/workspaces/{workspace}":                                                             workspace.Delete,
		"DELETE /api/workspaces/{workspace}/connections/{connection}":                                    connection.Delete,
		"DELETE /api/workspaces/{workspace}/connections/{connection}/actions/{action}":                   action.Delete,
		"DELETE /api/workspaces/{workspace}/connections/{connection}/keys/{key}":                         connection.DeleteWriteKey,
		"DELETE /api/workspaces/{workspace}/connections/{connection}/linked-connections/{connection2}":   connection.UnlinkConnection,
		"DELETE /api/workspaces/{workspace}/event-listeners/{listener}":                                  workspace.DeleteEventListener,
		"GET    /api/connectors":                                                                         api.Connectors,
		"GET    /api/connectors/{connector}":                                                             api.Connector,
		"GET    /api/connectors/{connector}/auth-code-url":                                               connector.AuthCodeURL,
		"GET    /api/event-schema":                                                                       api.EventSchema,
		"GET    /api/keys":                                                                               organization.APIKeys, /* only UI */
		"GET    /api/members":                                                                            organization.Members, /* only UI */
		"GET    /api/members/current":                                                                    api.Member,           /* only UI */
		"GET    /api/members/invitations/{token}":                                                        api.MemberInvitation, /* only UI */
		"GET    /api/transformation-languages":                                                           api.TransformationLanguages,
		"GET    /api/warehouses":                                                                         api.WarehouseTypes,
		"GET    /api/workspaces":                                                                         organization.Workspaces,
		"GET    /api/workspaces/{workspace}":                                                             organization.Workspace,
		"GET    /api/workspaces/{workspace}/action-errors":                                               workspace.ActionErrors,
		"GET    /api/workspaces/{workspace}/action-metrics/dates":                                        workspace.ActionMetricsPerDate,
		"GET    /api/workspaces/{workspace}/action-metrics/days":                                         workspace.ActionMetricsPerDay,
		"GET    /api/workspaces/{workspace}/action-metrics/hours":                                        workspace.ActionMetricsPerHour,
		"GET    /api/workspaces/{workspace}/action-metrics/minutes":                                      workspace.ActionMetricsPerMinute,
		"GET    /api/workspaces/{workspace}/connections":                                                 workspace.Connections,
		"GET    /api/workspaces/{workspace}/connections/{connection}":                                    workspace.Connection,
		"GET    /api/workspaces/{workspace}/connections/{connection}/action-types":                       connection.ActionTypes,
		"GET    /api/workspaces/{workspace}/connections/{connection}/actions/schemas/Events/{eventType}": connection.ActionSchemas,
		"GET    /api/workspaces/{workspace}/connections/{connection}/actions/schemas/{target}":           connection.ActionSchemas,
		"GET    /api/workspaces/{workspace}/connections/{connection}/actions/{action}":                   connection.Action,
		"GET    /api/workspaces/{workspace}/connections/{connection}/complete-path/{path}":               connection.CompletePath,
		"GET    /api/workspaces/{workspace}/connections/{connection}/executions":                         connection.Executions,
		"GET    /api/workspaces/{workspace}/connections/{connection}/keys":                               connection.WriteKeys,
		"GET    /api/workspaces/{workspace}/connections/{connection}/tables/{table}/schema":              connection.TableSchema,
		"GET    /api/workspaces/{workspace}/connections/{connection}/ui":                                 connection.ServeUI, /* only UI */
		"GET    /api/workspaces/{workspace}/event-listeners/{listener}/events":                           workspace.ListenedEvents,
		"GET    /api/workspaces/{workspace}/identifiers-schema":                                          workspace.IdentifiersSchema,
		"GET    /api/workspaces/{workspace}/identity-resolution/execution":                               workspace.LastIdentityResolution,
		"GET    /api/workspaces/{workspace}/user-schema":                                                 workspace.UserSchema,
		"GET    /api/workspaces/{workspace}/users/{user}/identities":                                     workspace.Identities,
		"GET    /api/workspaces/{workspace}/users/{user}/traits":                                         workspace.Traits,
		"GET    /api/workspaces/{workspace}/warehouse/settings":                                          workspace.Warehouse,
		"POST   /api/can-initialize-warehouse":                                                           organization.TestWorkspaceCreation,
		"POST   /api/expressions-properties":                                                             api.ExpressionsProperties, /* only UI */
		"POST   /api/keys":                                                                               organization.CreateAPIKey, /* only UI */
		"POST   /api/members/invitations":                                                                organization.InviteMember, /* only UI */
		"POST   /api/members/login":                                                                      s.login,                   /* only UI */
		"POST   /api/members/logout":                                                                     s.logout,                  /* only UI */
		"POST   /api/transformations":                                                                    api.TransformData,         /* only UI */
		"POST   /api/validate-expression":                                                                api.ValidateExpression,    /* only UI */
		"POST   /api/workspaces":                                                                         organization.CreateWorkspace,
		"POST   /api/workspaces/{workspace}/change-user-schema-queries":                                  workspace.PreviewUserSchemaUpdate,
		"POST   /api/workspaces/{workspace}/connections":                                                 workspace.CreateConnection,
		"POST   /api/workspaces/{workspace}/connections/{connection}/actions":                            connection.CreateAction,
		"POST   /api/workspaces/{workspace}/connections/{connection}/actions/{action}/executions":        action.Execute,
		"POST   /api/workspaces/{workspace}/connections/{connection}/actions/{action}/ui-event":          action.ServeUI, /* only UI */
		"POST   /api/workspaces/{workspace}/connections/{connection}/app-users":                          connection.AppUsers,
		"POST   /api/workspaces/{workspace}/connections/{connection}/events/send-previews":               connection.PreviewSendEvent,
		"POST   /api/workspaces/{workspace}/connections/{connection}/identities":                         connection.Identities,
		"POST   /api/workspaces/{workspace}/connections/{connection}/keys":                               connection.CreateWriteKey,
		"POST   /api/workspaces/{workspace}/connections/{connection}/linked-connections/{connection2}":   connection.LinkConnection,
		"POST   /api/workspaces/{workspace}/connections/{connection}/query/executions":                   connection.ExecQuery,
		"POST   /api/workspaces/{workspace}/connections/{connection}/records":                            connection.Records,
		"POST   /api/workspaces/{workspace}/connections/{connection}/sheets":                             connection.Sheets,
		"POST   /api/workspaces/{workspace}/connections/{connection}/ui-event":                           connection.ServeUI, /* only UI */
		"POST   /api/workspaces/{workspace}/event-listeners":                                             workspace.CreateEventListener,
		"POST   /api/workspaces/{workspace}/events":                                                      workspace.Events,
		"POST   /api/workspaces/{workspace}/identity-resolutions":                                        workspace.StartIdentityResolution,
		"POST   /api/workspaces/{workspace}/oauth-token":                                                 workspace.OAuthToken,
		"POST   /api/workspaces/{workspace}/ui":                                                          workspace.ServeUI, /* only UI */
		"POST   /api/workspaces/{workspace}/ui-event":                                                    workspace.ServeUI, /* only UI */
		"POST   /api/workspaces/{workspace}/users":                                                       workspace.Users,
		"POST   /api/workspaces/{workspace}/warehouse/can-change-settings":                               workspace.TestWarehouseUpdate,
		"POST   /api/workspaces/{workspace}/warehouse/repair":                                            workspace.RepairWarehouse,
		"PUT    /api/keys/{key}":                                                                         organization.UpdateAPIKey, /* only UI */
		"PUT    /api/members/current":                                                                    organization.UpdateMember, /* only UI */
		"PUT    /api/members/invitations/{token}":                                                        api.AcceptInvitation,      /* only UI */
		"PUT    /api/workspaces/{workspace}":                                                             workspace.Update,
		"PUT    /api/workspaces/{workspace}/connections/{connection}":                                    connection.Update,
		"PUT    /api/workspaces/{workspace}/connections/{connection}/actions/{action}":                   action.Update,
		"PUT    /api/workspaces/{workspace}/connections/{connection}/actions/{action}/schedule-period":   action.SetSchedulePeriod,
		"PUT    /api/workspaces/{workspace}/connections/{connection}/actions/{action}/status":            action.SetStatus,
		"PUT    /api/workspaces/{workspace}/identity-resolution/settings":                                workspace.UpdateIdentityResolution,
		"PUT    /api/workspaces/{workspace}/user-schema":                                                 workspace.UpdateUserSchema,
		"PUT    /api/workspaces/{workspace}/warehouse/mode":                                              workspace.UpdateWarehouseMode,
		"PUT    /api/workspaces/{workspace}/warehouse/settings":                                          workspace.UpdateWarehouse,
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
	s.mux.ServeHTTP(w, r)
}

var loginRequiredError = errors.Unprocessable(LoginRequired, "login is required")

// credentials returns the logged member with its organization. If no member is
// logged, it returns an errors.Unprocessable error with code LoginRequired.
func (s *apisServer) credentials(r *http.Request) (*core.Member, *core.Organization, error) {

	// Get the session.
	cookie, _ := r.Cookie(sessionCookieName)
	if cookie == nil {
		return nil, nil, loginRequiredError
	}
	session := &sessionCookie{}
	err := s.secureCookie.Decode(sessionCookieName, cookie.Value, session)
	if err != nil {
		return nil, nil, loginRequiredError
	}

	// Get the organization.
	organization, err := s.core.Organization(r.Context(), session.Organization)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			return nil, nil, loginRequiredError
		}
		return nil, nil, err
	}

	// Get the member.
	member, err := organization.Member(r.Context(), session.Member)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			err = loginRequiredError
		}
		return nil, nil, err
	}

	return member, organization, nil
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
