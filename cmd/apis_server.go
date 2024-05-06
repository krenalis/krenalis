//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/apis/errors"

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
	apis         *apis.APIs
	secureCookie *securecookie.SecureCookie // secureCookie contains keys to encrypt/decrypt/remove the session cookie.
	mux          *http.ServeMux
}

// newAPIsServer returns an APIs server that handles requests for the given
// APIs. sessionKey is the key used to encrypt the session cookie.
// It panics if the session key is not at least 64 bytes long.
func newAPIsServer(apis *apis.APIs, sessionKey []byte) *apisServer {

	if len(sessionKey) != 64 {
		panic("sessionKey is not 64 bytes long")
	}

	s := &apisServer{apis: apis}

	hashKey, blockKey := sessionKey[:32], sessionKey[32:]
	s.secureCookie = securecookie.New(hashKey, blockKey)
	s.secureCookie.MaxAge(sessionMaxAge)

	api := api{s}
	connector := connector{s}
	organization := organization{s}
	workspace := workspace{s}
	connection := connection{s}
	action := action{s}
	user := user{s}

	paths := map[string]func(w http.ResponseWriter, r *http.Request) (any, error){
		"DELETE /api/members/{member}":                                                                  organization.DeleteMember,
		"DELETE /api/workspaces/{workspace}":                                                            workspace.Delete,
		"DELETE /api/workspaces/{workspace}/connections/{connection}":                                   connection.Delete,
		"DELETE /api/workspaces/{workspace}/connections/{connection}/actions/{action}":                  action.Delete,
		"DELETE /api/workspaces/{workspace}/connections/{connection}/event-connections/{connection2}":   connection.RemoveEventConnection,
		"DELETE /api/workspaces/{workspace}/connections/{connection}/keys/{key}":                        connection.RevokeKey,
		"DELETE /api/workspaces/{workspace}/event-listeners/{listener}":                                 workspace.RemoveEventListener,
		"GET    /api/connectors":                                                                        api.Connectors,
		"GET    /api/connectors/{connector}":                                                            api.Connector,
		"GET    /api/connectors/{connector}/auth-code-url":                                              connector.AuthCodeURL,
		"GET    /api/events-schema":                                                                     api.EventSchema,
		"GET    /api/members":                                                                           organization.Members,
		"GET    /api/members/current":                                                                   api.Member,
		"GET    /api/members/invitations/{token}":                                                       api.MemberInvitation,
		"GET    /api/transformation-languages":                                                          api.TransformationLanguages,
		"GET    /api/workspaces":                                                                        organization.Workspaces,
		"GET    /api/workspaces/{workspace}":                                                            organization.Workspace,
		"GET    /api/workspaces/{workspace}/connections":                                                workspace.Connections,
		"GET    /api/workspaces/{workspace}/connections/{connection}":                                   workspace.Connection,
		"GET    /api/workspaces/{workspace}/connections/{connection}/action-schemas/Events/{eventType}": connection.ActionSchemas,
		"GET    /api/workspaces/{workspace}/connections/{connection}/action-schemas/{target}":           connection.ActionSchemas,
		"GET    /api/workspaces/{workspace}/connections/{connection}/actions/{action}":                  connection.Action,
		"GET    /api/workspaces/{workspace}/connections/{connection}/complete-path/{path}":              connection.CompletePath,
		"GET    /api/workspaces/{workspace}/connections/{connection}/executions":                        connection.Executions,
		"GET    /api/workspaces/{workspace}/connections/{connection}/keys":                              connection.Keys,
		"GET    /api/workspaces/{workspace}/connections/{connection}/stats":                             connection.Stats,
		"GET    /api/workspaces/{workspace}/connections/{connection}/tables/{table}/schema":             connection.TableSchema,
		"GET    /api/workspaces/{workspace}/connections/{connection}/ui":                                connection.ServeUI,
		"GET    /api/workspaces/{workspace}/event-listeners/{listener}/events":                          workspace.ListenedEvents,
		"GET    /api/workspaces/{workspace}/identifiers-schema":                                         workspace.IdentifiersSchema,
		"GET    /api/workspaces/{workspace}/privacy-region":                                             workspace.PrivacyRegion,
		"GET    /api/workspaces/{workspace}/user-schema":                                                workspace.UsersSchema,
		"GET    /api/workspaces/{workspace}/users/{user}/events":                                        user.Events,
		"GET    /api/workspaces/{workspace}/users/{user}/identities":                                    user.Identities,
		"GET    /api/workspaces/{workspace}/users/{user}/traits":                                        user.Traits,
		"GET    /api/workspaces/{workspace}/warehouse-settings":                                         workspace.WarehouseSettings,
		"POST   /api/expressions-properties":                                                            api.ExpressionsProperties,
		"POST   /api/members/invitations":                                                               organization.InviteMember,
		"POST   /api/members/login":                                                                     s.login,
		"POST   /api/members/logout":                                                                    s.logout,
		"POST   /api/transform-data":                                                                    api.TransformData,
		"POST   /api/validate-expression":                                                               api.ValidateExpression,
		"POST   /api/workspaces":                                                                        organization.AddWorkspace,
		"POST   /api/workspaces/{workspace}/add-connection":                                             workspace.AddConnection,
		"POST   /api/workspaces/{workspace}/change-users-schema":                                        workspace.ChangeUsersSchema,
		"POST   /api/workspaces/{workspace}/change-users-schema-queries":                                workspace.ChangeUsersSchemaQueries,
		"POST   /api/workspaces/{workspace}/connect-warehouse":                                          workspace.ConnectWarehouse,
		"POST   /api/workspaces/{workspace}/connections/{connection}":                                   connection.Set,
		"POST   /api/workspaces/{workspace}/connections/{connection}/actions":                           connection.AddAction,
		"POST   /api/workspaces/{workspace}/connections/{connection}/actions/{action}/execute":          action.Execute,
		"POST   /api/workspaces/{workspace}/connections/{connection}/actions/{action}/schedule-period":  action.SetSchedulePeriod,
		"POST   /api/workspaces/{workspace}/connections/{connection}/actions/{action}/status":           action.SetStatus,
		"POST   /api/workspaces/{workspace}/connections/{connection}/actions/{action}/ui-event":         action.ServeUI,
		"POST   /api/workspaces/{workspace}/connections/{connection}/app-users":                         connection.AppUsers,
		"POST   /api/workspaces/{workspace}/connections/{connection}/event-connections/{connection2}":   connection.AddEventConnection,
		"POST   /api/workspaces/{workspace}/connections/{connection}/event-preview":                     connection.PreviewSendEvent,
		"POST   /api/workspaces/{workspace}/connections/{connection}/exec-query":                        connection.ExecQuery,
		"POST   /api/workspaces/{workspace}/connections/{connection}/identities":                        connection.Identities,
		"POST   /api/workspaces/{workspace}/connections/{connection}/keys":                              connection.GenerateKey,
		"POST   /api/workspaces/{workspace}/connections/{connection}/records":                           connection.Records,
		"POST   /api/workspaces/{workspace}/connections/{connection}/sheets":                            connection.Sheets,
		"POST   /api/workspaces/{workspace}/connections/{connection}/ui-event":                          connection.ServeUI,
		"POST   /api/workspaces/{workspace}/disconnect-warehouse":                                       workspace.DisconnectWarehouse,
		"POST   /api/workspaces/{workspace}/identifiers":                                                workspace.SetIdentifiers,
		"POST   /api/workspaces/{workspace}/init-warehouse":                                             workspace.InitWarehouse,
		"POST   /api/workspaces/{workspace}/oauth-token":                                                workspace.OAuthToken,
		"POST   /api/workspaces/{workspace}/ping-warehouse":                                             workspace.PingWarehouse,
		"POST   /api/workspaces/{workspace}/run-identity-resolution":                                    workspace.RunIdentityResolution,
		"POST   /api/workspaces/{workspace}/ui":                                                         workspace.ServeUI,
		"POST   /api/workspaces/{workspace}/ui-event":                                                   workspace.ServeUI,
		"POST   /api/workspaces/{workspace}/users":                                                      workspace.Users,
		"PUT    /api/members/current":                                                                   organization.SetMember,
		"PUT    /api/members/invitations/{token}":                                                       api.AcceptInvitation,
		"PUT    /api/workspaces/{workspace}":                                                            workspace.Set,
		"PUT    /api/workspaces/{workspace}/connections/{connection}/actions/{action}":                  action.Set,
		"PUT    /api/workspaces/{workspace}/event-listeners/":                                           workspace.AddEventListener,
		"PUT    /api/workspaces/{workspace}/warehouse-mode":                                             workspace.ChangeWarehouseMode,
		"PUT    /api/workspaces/{workspace}/warehouse-settings":                                         workspace.ChangeWarehouseSettings,
	}

	s.mux = http.NewServeMux()
	for path, serve := range paths {
		s.mux.HandleFunc(rewrittenPath(path), func(w http.ResponseWriter, r *http.Request) {
			response, err := serve(w, r)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				slog.Error("error occurred serving APIs", "err", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if response != nil {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
			}
		})
	}

	return s
}

// rewrittenPath returns a path rewritten with only one space between method and
// path.
func rewrittenPath(path string) string {
	s := strings.IndexByte(path, ' ')
	if s == -1 {
		panic(fmt.Sprintf("parsing %q: it does not contain spaces", path))
	}
	var i int
	for i = s + 1; i < len(path); i++ {
		if path[i] != ' ' {
			break
		}
	}
	if i == s+1 {
		return path
	}
	return path[0:s+1] + path[i:]
}

// ServeHTTP servers the API methods from HTTP.
func (s *apisServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

var loginRequiredError = errors.Unprocessable(LoginRequired, "login is required")

// credentials returns the logged member with its organization. If no member is
// logged, it returns an errors.Unprocessable error with code LoginRequired.
func (s *apisServer) credentials(r *http.Request) (*apis.Member, *apis.Organization, error) {

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
	organization, err := s.apis.Organization(r.Context(), session.Organization)
	if err != nil {
		if _, ok := err.(*errors.NotFoundError); ok {
			return nil, nil, loginRequiredError
		}
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

	body := struct {
		Email    string
		Password string
	}{}
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&body)
	if err != nil {
		return nil, errors.BadRequest("")
	}

	// Retrieve the organization and the member.
	organization, err := s.apis.Organization(r.Context(), 1)
	if err != nil {
		return nil, fmt.Errorf("cannot read organization: %s", err)
	}
	memberID, err := organization.AuthenticateMember(r.Context(), body.Email, body.Password)
	if err != nil {
		if err, ok := err.(*errors.UnprocessableError); ok && err.Code == apis.AuthenticationFailed {
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
		Name:     sessionCookieName,
		Value:    value,
		Path:     sessionCookiePath,
		Secure:   true,
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
		Secure:   true,
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

var _ json.Marshaler = (*rawJSON)(nil)
var _ json.Unmarshaler = (*rawJSON)(nil)

// rawJSON is a raw encoded JSON value.
// It implements the json.Marshaler and json.Unmarshaler interfaces.
type rawJSON []byte

// MarshalJSON returns the JSON encoding form of raw.
func (raw rawJSON) MarshalJSON() ([]byte, error) {
	if raw == nil {
		return []byte("null"), nil
	}
	return raw, nil
}

var null = []byte("null")

// UnmarshalJSON sets *raw to a copy of data.
// Unlike the UnmarshalJSON method of json.RawMessage, it unmarshal a "null"
// JSON value to []byte(nil) instead of []byte("null").
func (raw *rawJSON) UnmarshalJSON(data []byte) error {
	if raw == nil {
		return errors.New("rawJSON.UnmarshalJSON: raw cannot be a nil pointer")
	}
	if bytes.Equal(data, null) {
		*raw = nil
		return nil
	}
	*raw = append((*raw)[:0], data...)
	return nil
}
