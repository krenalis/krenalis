//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package server

import (
	_ "embed"
	"encoding/json"
	"html"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"chichi/apis"
	"chichi/apis/errors"
	"chichi/apis/events/eventschema"
	"chichi/telemetry"
	"chichi/types"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/securecookie"
)

//go:embed invite-member-email.html
var inviteMemberEmail string

// sessionMaxAge contains the max age property for the session cookie (6 hours).
const sessionMaxAge = 6 * 60 * 60

// LoginRequired is the error code returned by the API when login is required.
const LoginRequired errors.Code = "LoginRequired"

var workspacePathRegExp = regexp.MustCompile(`^/api/workspaces(/.*)?$`)

type apisServer struct {
	apis         *apis.APIs
	secureCookie *securecookie.SecureCookie // secureCookie contains keys to encrypt/decrypt/remove the session cookie.
}

// newAPIsServer returns an APIs server that handles requests for the given
// APIs. sessionKey is the key used to encrypt the session cookie.
// It panics if the session key is not at least 64 bytes long.
func newAPIsServer(apis *apis.APIs, sessionKey []byte) *apisServer {
	if len(sessionKey) != 64 {
		panic("sessionKey is not 64 bytes long")
	}
	hashKey, blockKey := sessionKey[:32], sessionKey[32:]
	sc := securecookie.New(hashKey, blockKey)
	sc.MaxAge(sessionMaxAge)
	return &apisServer{apis: apis, secureCookie: sc}
}

// ServeHTTP servers the API methods from HTTP.
func (s *apisServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, span := telemetry.TraceSpan(r.Context(), "apis.ServeHTTP", "ip", r.RemoteAddr, "urlPath", r.URL.Path)
	defer span.End()

	telemetry.IncrementCounter(ctx, "apis.ServeHTTP", 1)

	switch r.URL.Path {
	case "/api/members/login":
		s.login(w, r)
		return
	case "/api/members/logout":
		s.logout(w, r)
		return
	}

	var session *sessionCookie
	var organization *apis.Organization
	var member *apis.Member
	var err error
	if requiresLogin(r.URL.Path, r.Method) {
		session = s.getSession(r)
		if session == nil {
			respond(w, errors.Unprocessable(LoginRequired, "login is required"))
			return
		}
		// Read the organization.
		organization, err = s.apis.Organization(ctx, session.Organization)
		if err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				s.removeSession(w, r)
				respond(w, errors.Unprocessable(LoginRequired, "login is required"))
				return
			}
			respond(w, err)
			return
		}
		if member, err = organization.Member(ctx, session.Member); err != nil {
			if _, ok := err.(*errors.NotFoundError); ok {
				s.removeSession(w, r)
				respond(w, errors.Unprocessable(LoginRequired, "login is required"))
				return
			}
			respond(w, err)
			return
		}
	}

	router := chi.NewRouter()

	if workspacePathRegExp.MatchString(r.URL.Path) {

		var workspace *apis.Workspace
		if r.URL.Path != "/api/workspaces" && r.URL.Path != "/api/workspaces/" {
			// The path must contain the id of the workspace.
			var workspaceID int
			workspaceID, err = strconv.Atoi(strings.Split(r.URL.Path, "/")[3])
			if err != nil || workspaceID < 1 || workspaceID > math.MaxInt32 {
				http.Error(w, "Bad Request (invalid workspace id)", http.StatusBadRequest)
				return
			}
			workspace, err = organization.Workspace(workspaceID)
			if err != nil {
				http.NotFound(w, r)
				return
			}
		}

		router.Route("/api/workspaces", func(router chi.Router) {
			router.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(organization.Workspaces())
			})
			router.Post("/", func(w http.ResponseWriter, r *http.Request) {
				req := struct {
					Name          string
					PrivacyRegion apis.PrivacyRegion
				}{}
				err := json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, errors.BadRequest("invalid JSON"))
					return
				}
				id, err := organization.AddWorkspace(ctx, req.Name, req.PrivacyRegion)
				w.Header().Add("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]int{"id": id})
				respond(w, err)
			})
			router.Route("/{workspaceID}", func(router chi.Router) {
				router.Get("/", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(workspace)
				})
				router.Put("/", func(w http.ResponseWriter, r *http.Request) {
					var req struct {
						Name                string
						PrivacyRegion       apis.PrivacyRegion
						DisplayedProperties apis.DisplayedProperties
					}
					err := json.NewDecoder(r.Body).Decode(&req)
					if err != nil {
						respond(w, errors.BadRequest("invalid JSON"))
						return
					}
					err = workspace.Set(ctx, req.Name, req.PrivacyRegion, req.DisplayedProperties)
					respond(w, err)
				})
				router.Delete("/", func(w http.ResponseWriter, r *http.Request) {
					err := workspace.Delete(ctx)
					respond(w, err)
				})
				router.Route("/change-users-schema", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						req := struct {
							Schema  types.Type
							RePaths map[string]any
						}{}
						err := json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						err = workspace.ChangeUsersSchema(ctx, req.Schema, req.RePaths)
						respond(w, err)
					})
				})
				router.Route("/change-users-schema-queries", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						req := struct {
							Schema  types.Type
							RePaths map[string]any
						}{}
						err := json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						queries, err := workspace.ChangeUsersSchemaQueries(ctx, req.Schema, req.RePaths)
						if err != nil {
							respond(w, err)
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]any{
							"Queries": queries,
						})
					})
				})
				router.Route("/identifiers", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						req := struct {
							Identifiers []string
						}{}
						err := json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						err = workspace.SetIdentifiers(ctx, req.Identifiers)
						respond(w, err)
					})
				})
				router.Route("/connect-warehouse", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						req := struct {
							Type     apis.WarehouseType
							Settings json.RawMessage
						}{}
						err := json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						err = workspace.ConnectWarehouse(ctx, req.Type, req.Settings)
						respond(w, err)
					})
				})
				router.Route("/disconnect-warehouse", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						err = workspace.DisconnectWarehouse(ctx)
						respond(w, err)
					})
				})
				router.Route("/init-warehouse", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						err = workspace.InitWarehouse(ctx)
						respond(w, err)
					})
				})
				router.Route("/ping-warehouse", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						req := struct {
							Type     apis.WarehouseType
							Settings json.RawMessage
						}{}
						err := json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						err = workspace.PingWarehouse(ctx, req.Type, req.Settings)
						respond(w, err)
					})
				})
				router.Route("/warehouse-settings", func(router chi.Router) {
					router.Get("/", func(w http.ResponseWriter, r *http.Request) {
						typ, settings, err := workspace.WarehouseSettings()
						if err != nil {
							respond(w, err)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]any{
							"type":     typ,
							"settings": json.RawMessage(settings),
						})
					})
					router.Put("/", func(w http.ResponseWriter, r *http.Request) {
						req := struct {
							Type     apis.WarehouseType
							Settings json.RawMessage
						}{}
						err := json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						err = workspace.ChangeWarehouseSettings(ctx, req.Type, req.Settings)
						respond(w, err)
					})
				})
				router.Route("/user-schema", func(router chi.Router) {
					router.Get("/", func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(workspace.UsersSchema)
					})
				})
				router.Route("/identifiers-schema", func(router chi.Router) {
					router.Get("/", func(w http.ResponseWriter, r *http.Request) {
						schema, err := workspace.IdentifiersSchema(ctx)
						if err != nil {
							respond(w, err)
							return
						}
						w.Header().Add("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(schema)
					})
				})
				router.Route("/run-identity-resolution", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						err := workspace.RunIdentityResolution(ctx)
						respond(w, err)
					})
				})
				router.Post("/ui", func(w http.ResponseWriter, r *http.Request) {
					var req struct {
						Connector  int
						Role       string
						OAuthToken string
					}
					err = json.NewDecoder(r.Body).Decode(&req)
					if err != nil {
						respond(w, err)
						return
					}
					var role apis.Role
					switch req.Role {
					case "Source":
						role = apis.Source
					case "Destination":
						role = apis.Destination
					default:
						respond(w, errors.BadRequest("unexpected connection role '%s'", req.Role))
						return
					}
					form, err := workspace.ServeUI(ctx, "load", nil, req.Connector, role, req.OAuthToken)
					if err != nil {
						respond(w, err)
						return
					}
					w.Header().Add("Content-Type", "application/json")
					_, _ = w.Write(form)
				})
				router.Post("/ui-event", func(w http.ResponseWriter, r *http.Request) {
					var req struct {
						Connector  int
						Event      string
						Values     json.RawMessage
						Role       string
						OAuthToken string
					}
					err = json.NewDecoder(r.Body).Decode(&req)
					if err != nil {
						respond(w, err)
						return
					}
					var role apis.Role
					switch req.Role {
					case "Source":
						role = apis.Source
					case "Destination":
						role = apis.Destination
					default:
						respond(w, errors.BadRequest("unexpected connection role '%s'", req.Role))
						return
					}
					form, err := workspace.ServeUI(ctx, req.Event, req.Values, req.Connector, role, req.OAuthToken)
					if err != nil {
						respond(w, err)
						return
					}
					w.Header().Add("Content-Type", "application/json")
					_, _ = w.Write(form)
				})
				router.Route("/oauth-token", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						var req struct {
							Connector   int
							OAuthCode   string
							RedirectURI string
						}
						err := json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						oauthToken, err := workspace.OAuthToken(ctx, req.OAuthCode, req.RedirectURI, req.Connector)
						if err != nil {
							respond(w, err)
							return
						}
						w.Header().Add("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(oauthToken)
					})
				})
				router.Route("/add-connection", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						var req struct {
							Connection apis.ConnectionToAdd
							OAuthToken string
						}
						err = json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						id, err := workspace.AddConnection(ctx, req.Connection, req.OAuthToken)
						if err != nil {
							respond(w, err)
							return
						}
						w.Header().Add("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(id)
					})
				})
				router.Route("/privacy-region", func(router chi.Router) {
					router.Get("/", func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(workspace.PrivacyRegion)
					})
				})
				router.Route("/connections", func(router chi.Router) {
					router.Get("/", func(w http.ResponseWriter, r *http.Request) {
						connections := workspace.Connections()
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(connections)
					})
					router.Route("/{connectionID}", func(router chi.Router) {
						router.Get("/", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(connection)
						})
						router.Post("/", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								Connection apis.ConnectionToSet
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, errors.BadRequest("invalid JSON"))
								return
							}
							err = connection.Set(ctx, req.Connection)
							respond(w, err)
						})
						router.Delete("/", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							err = connection.Delete(ctx)
							respond(w, err)
						})
						router.Post("/actions", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								Target    apis.Target
								EventType string
								Action    apis.ActionToSet
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, errors.BadRequest("invalid JSON"))
								return
							}
							actionID, err := connection.AddAction(ctx, req.Target, req.EventType, req.Action)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(actionID)
						})
						router.Get("/actions/{actionID}", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							actionID, _ := strconv.Atoi(chi.URLParam(r, "actionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							action, err := connection.Action(ctx, actionID)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(action)
						})
						router.Put("/actions/{actionID}", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							actionID, _ := strconv.Atoi(chi.URLParam(r, "actionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var req apis.ActionToSet
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, errors.BadRequest("invalid JSON"))
								return
							}
							action, err := connection.Action(ctx, actionID)
							if err != nil {
								respond(w, err)
								return
							}
							err = action.Set(ctx, req)
							respond(w, err)
						})
						router.Delete("/actions/{actionID}", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							actionID, _ := strconv.Atoi(chi.URLParam(r, "actionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							action, err := connection.Action(ctx, actionID)
							if err != nil {
								respond(w, err)
								return
							}
							err = action.Delete(ctx)
							respond(w, err)
						})
						router.Post("/actions/{actionID}/execute", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							actionID, _ := strconv.Atoi(chi.URLParam(r, "actionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							action, err := connection.Action(ctx, actionID)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								Reimport bool
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, errors.BadRequest("invalid JSON"))
								return
							}
							err = action.Execute(ctx, req.Reimport)
							respond(w, err)
						})
						router.Post("/actions/{actionID}/schedule-period", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							actionID, _ := strconv.Atoi(chi.URLParam(r, "actionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							action, err := connection.Action(ctx, actionID)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								SchedulePeriod apis.SchedulePeriod
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, errors.BadRequest("invalid JSON"))
								return
							}
							err = action.SetSchedulePeriod(ctx, req.SchedulePeriod)
							respond(w, err)
						})
						router.Post("/actions/{actionID}/status", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							actionID, _ := strconv.Atoi(chi.URLParam(r, "actionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							action, err := connection.Action(ctx, actionID)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								Enabled bool
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, errors.BadRequest("invalid JSON"))
								return
							}
							err = action.SetStatus(ctx, req.Enabled)
							respond(w, err)
						})
						router.Post("/actions/{actionID}/ui-event", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							actionID, _ := strconv.Atoi(chi.URLParam(r, "actionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							action, err := connection.Action(ctx, actionID)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								Event  string
								Values json.RawMessage
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, err)
								return
							}
							form, err := action.ServeUI(ctx, req.Event, req.Values)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Add("Content-Type", "application/json")
							_, _ = w.Write(form)
						})
						router.Route("/action-schemas", func(router chi.Router) {
							router.Get("/Users", func(w http.ResponseWriter, r *http.Request) {
								id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
								connection, err := workspace.Connection(ctx, id)
								if err != nil {
									respond(w, err)
									return
								}
								schemas, err := connection.ActionSchemas(ctx, apis.Users, "")
								if err != nil {
									respond(w, err)
									return
								}
								w.Header().Set("Content-Type", "application/json")
								_ = json.NewEncoder(w).Encode(schemas)
							})
							router.Get("/Groups", func(w http.ResponseWriter, r *http.Request) {
								id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
								connection, err := workspace.Connection(ctx, id)
								if err != nil {
									respond(w, err)
									return
								}
								schemas, err := connection.ActionSchemas(ctx, apis.Groups, "")
								if err != nil {
									respond(w, err)
									return
								}
								w.Header().Set("Content-Type", "application/json")
								_ = json.NewEncoder(w).Encode(schemas)
							})
							router.Get("/Events", func(w http.ResponseWriter, r *http.Request) {
								id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
								connection, err := workspace.Connection(ctx, id)
								if err != nil {
									respond(w, err)
									return
								}
								schemas, err := connection.ActionSchemas(ctx, apis.Events, "")
								if err != nil {
									respond(w, err)
									return
								}
								w.Header().Set("Content-Type", "application/json")
								_ = json.NewEncoder(w).Encode(schemas)
							})
							router.Get("/Events/{eventType}", func(w http.ResponseWriter, r *http.Request) {
								id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
								eventType := chi.URLParam(r, "eventType")
								// Workaround for the issue of Chi https://github.com/go-chi/chi/issues/642.
								eventType, err = url.PathUnescape(eventType)
								if err != nil {
									respond(w, errors.BadRequest("invalid event type"))
									return
								}
								connection, err := workspace.Connection(ctx, id)
								if err != nil {
									respond(w, err)
									return
								}
								schemas, err := connection.ActionSchemas(ctx, apis.Events, eventType)
								if err != nil {
									respond(w, err)
									return
								}
								w.Header().Set("Content-Type", "application/json")
								_ = json.NewEncoder(w).Encode(schemas)
							})
						})
						router.Post("/app-users", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								Schema types.Type
								Cursor string
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, errors.BadRequest("invalid JSON"))
								return
							}
							users, cursor, err := connection.AppUsers(ctx, req.Schema, req.Cursor)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(map[string]any{"users": json.RawMessage(users), "cursor": cursor})
						})
						router.Get("/complete-path/{path}", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							path := chi.URLParam(r, "path")
							// Workaround for the issue of Chi https://github.com/go-chi/chi/issues/642.
							path, err = url.PathUnescape(path)
							if err != nil {
								respond(w, errors.BadRequest("invalid path"))
								return
							}
							completePath, err := connection.CompletePath(ctx, path)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(map[string]any{"path": completePath})
						})
						router.Post("/records", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								FileConnector int
								Path          string
								Sheet         string
								Compression   apis.Compression
								Settings      json.RawMessage
								Limit         int
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, errors.BadRequest("invalid JSON"))
								return
							}
							records, schema, err := connection.Records(ctx, req.FileConnector, req.Path, req.Sheet, req.Compression, req.Settings, req.Limit)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(map[string]any{"records": json.RawMessage(records), "schema": schema})
						})
						router.Post("/sheets", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								FileConnector int
								Path          string
								Compression   apis.Compression
								Settings      json.RawMessage
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, errors.BadRequest("invalid JSON"))
								return
							}
							sheets, err := connection.Sheets(ctx, req.FileConnector, req.Path, req.Settings, req.Compression)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(map[string]any{"sheets": sheets})
						})
						router.Get("/executions", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							executions, err := connection.Executions(ctx)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(executions)
						})
						router.Post("/identities", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							var req struct {
								First int
								Limit int
							}
							err := json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, errors.BadRequest("invalid JSON"))
								return
							}
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							identities, count, err := connection.Identities(ctx, req.First, req.Limit)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(map[string]any{
								"identities": json.RawMessage(identities),
								"count":      count,
							})
						})
						router.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var stats *apis.ConnectionsStats
							stats, err = connection.Stats(ctx)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(stats)
						})
						router.Get("/ui", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							form, err := connection.ServeUI(ctx, "load", nil)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Add("Content-Type", "application/json")
							_, _ = w.Write(form)
						})
						router.Post("/ui-event", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								Event  string
								Values json.RawMessage
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, err)
								return
							}
							form, err := connection.ServeUI(ctx, req.Event, req.Values)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Add("Content-Type", "application/json")
							_, _ = w.Write(form)
						})
						router.Delete("/event-connections/{id}", func(w http.ResponseWriter, r *http.Request) {
							connectionID, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							idString, err := url.PathUnescape(chi.URLParam(r, "id"))
							if err != nil {
								respond(w, errors.BadRequest("invalid event connection id"))
								return
							}
							connection, err := workspace.Connection(ctx, connectionID)
							if err != nil {
								respond(w, err)
								return
							}
							id, _ := strconv.Atoi(idString)
							err = connection.RemoveEventConnection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
						})
						router.Post("/event-connections/{id}", func(w http.ResponseWriter, r *http.Request) {
							connectionID, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							idString, err := url.PathUnescape(chi.URLParam(r, "id"))
							if err != nil {
								respond(w, errors.BadRequest("invalid event connection id"))
								return
							}
							connection, err := workspace.Connection(ctx, connectionID)
							if err != nil {
								respond(w, err)
								return
							}
							id, _ := strconv.Atoi(idString)
							err = connection.AddEventConnection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
						})
						router.Post("/event-preview", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								EventType      string
								Event          *apis.ObservedEvent
								Transformation apis.Transformation
								OutSchema      types.Type
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, err)
								return
							}
							preview, err := connection.PreviewSendEvent(ctx, req.EventType, req.Event, req.Transformation, req.OutSchema)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Add("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(map[string]any{"preview": string(preview)})
						})
						router.Post("/exec-query", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var req struct {
								Query string
								Limit int
							}
							err = json.NewDecoder(r.Body).Decode(&req)
							if err != nil {
								respond(w, err)
								return
							}
							rows, schema, err := connection.ExecQuery(ctx, req.Query, req.Limit)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Add("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(map[string]any{"Rows": json.RawMessage(rows), "Schema": schema})
						})
						router.Get("/tables/{table}/schema", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							table := chi.URLParam(r, "table")
							// Workaround for the issue of Chi https://github.com/go-chi/chi/issues/642.
							table, err = url.PathUnescape(table)
							if err != nil {
								respond(w, errors.BadRequest("invalid table name"))
								return
							}
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							schema, err := connection.TableSchema(ctx, table)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(schema)
						})
						router.Get("/keys", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var keys []string
							keys, err = connection.Keys()
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(keys)
						})
						router.Post("/keys", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							var key string
							key, err = connection.GenerateKey(ctx)
							if err != nil {
								respond(w, err)
								return
							}
							w.Header().Set("Content-Type", "application/json")
							_ = json.NewEncoder(w).Encode(key)
						})
						router.Delete("/keys/{key}", func(w http.ResponseWriter, r *http.Request) {
							id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
							connection, err := workspace.Connection(ctx, id)
							if err != nil {
								respond(w, err)
								return
							}
							key := chi.URLParam(r, "key")
							err = connection.RevokeKey(ctx, key)
							respond(w, err)
						})
					})
				})
				router.Route("/event-listeners", func(router chi.Router) {
					router.Put("/", func(w http.ResponseWriter, r *http.Request) {
						var req struct {
							Size      *int
							Source    int
							OnlyValid bool
						}
						err := json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						var size = 10
						if req.Size != nil {
							size = *req.Size
						}
						id, err := workspace.AddEventListener(ctx, size, req.Source, req.OnlyValid)
						if err != nil {
							respond(w, err)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
					})
					router.Delete("/{listenerID}", func(w http.ResponseWriter, r *http.Request) {
						id := chi.URLParam(r, "listenerID")
						workspace.RemoveEventListener(id)
					})
					router.Get("/{listenerID}/events", func(w http.ResponseWriter, r *http.Request) {
						id := chi.URLParam(r, "listenerID")
						events, discarded, err := workspace.ListenedEvents(id)
						if err != nil {
							respond(w, err)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]any{
							"events":    events,
							"discarded": discarded,
						})
					})
				})
				router.Route("/users", func(router chi.Router) {
					router.Post("/", func(w http.ResponseWriter, r *http.Request) {
						var req struct {
							Properties []string
							Filter     *apis.Filter
							Order      string
							OrderDesc  bool
							First      int
							Limit      int
						}
						err := json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						users, schema, count, err := workspace.Users(ctx, req.Properties, req.Filter, req.Order, req.OrderDesc, req.First, req.Limit)
						if err != nil {
							respond(w, err)
							return
						}
						w.Header().Add("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]any{
							"users":  json.RawMessage(users),
							"schema": schema,
							"count":  count,
						})
					})
					router.Get("/{userID}/events", func(w http.ResponseWriter, r *http.Request) {
						id, _ := strconv.Atoi(chi.URLParam(r, "userID"))
						user, err := workspace.User(id)
						if err != nil {
							respond(w, err)
							return
						}
						events, err := user.Events(ctx, 10)
						if err != nil {
							respond(w, err)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]any{
							"events": json.RawMessage(events),
						})
					})
					router.Post("/{userID}/identities", func(w http.ResponseWriter, r *http.Request) {
						id, _ := strconv.Atoi(chi.URLParam(r, "userID"))
						var req struct {
							First int
							Limit int
						}
						err := json.NewDecoder(r.Body).Decode(&req)
						if err != nil {
							respond(w, errors.BadRequest("invalid JSON"))
							return
						}
						user, err := workspace.User(id)
						if err != nil {
							respond(w, err)
							return
						}
						identities, count, err := user.Identities(ctx, req.First, req.Limit)
						if err != nil {
							respond(w, err)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]any{
							"identities": json.RawMessage(identities),
							"count":      count,
						})
					})
					router.Get("/{userID}/traits", func(w http.ResponseWriter, r *http.Request) {
						id, _ := strconv.Atoi(chi.URLParam(r, "userID"))
						user, err := workspace.User(id)
						if err != nil {
							respond(w, err)
							return
						}
						traits, err := user.Traits(ctx)
						if err != nil {
							respond(w, err)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(map[string]any{
							"traits": json.RawMessage(traits),
						})
					})
				})
			})
		})
	}

	router.Route("/api/connectors", func(router chi.Router) {
		router.Get("/", func(w http.ResponseWriter, r *http.Request) {
			connectors := s.apis.Connectors(ctx)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(connectors)
		})
		router.Route("/{connectorID}", func(router chi.Router) {
			router.Get("/", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectorID"))
				connector, err := s.apis.Connector(ctx, id)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(connector)
			})
			router.Get("/auth-code-url", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectorID"))
				connector, err := s.apis.Connector(ctx, id)
				if err != nil {
					respond(w, err)
					return
				}
				redirectURI := r.URL.Query().Get("redirecturi")
				url, err := connector.AuthCodeURL(redirectURI)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Add("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"url": url})
			})
		})
	})

	router.Get("/api/events-schema", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(eventschema.SchemaWithoutGID)
	})
	router.Post("/api/validate-expression", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Expression string
			Properties []types.Property
			Type       types.Type
			Required   bool
			Nullable   bool
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respond(w, errors.BadRequest("invalid JSON"))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		message := s.apis.ValidateExpression(req.Expression, req.Properties, req.Type, req.Required, req.Nullable)
		_ = json.NewEncoder(w).Encode(message)
	})
	router.Post("/api/expressions-properties", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Expressions []apis.ExpressionToBeExtracted
			Schema      types.Type
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respond(w, errors.BadRequest("invalid JSON"))
			return
		}
		properties, err := s.apis.ExpressionsProperties(req.Expressions, req.Schema)
		if err != nil {
			respond(w, errors.BadRequest(err.Error()))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(properties)
	})
	router.Get("/api/transformation-languages", func(w http.ResponseWriter, r *http.Request) {
		languages := s.apis.TransformationLanguages()
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string][]string{"languages": languages})
	})
	router.Post("/api/transform-data", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Data           json.RawMessage
			InSchema       types.Type
			OutSchema      types.Type
			Transformation apis.Transformation
		}
		err = json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respond(w, err)
			return
		}
		data, err := s.apis.TransformData(ctx, req.Data, req.InSchema, req.OutSchema, req.Transformation)
		if err != nil {
			respond(w, err)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": json.RawMessage(data)})
	})
	router.Route("/api/members", func(router chi.Router) {
		router.Get("/", func(w http.ResponseWriter, r *http.Request) {
			members, err := organization.Members(ctx)
			if err != nil {
				respond(w, err)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(members)
		})
		router.Route("/invitations", func(router chi.Router) {
			router.Post("/", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Email string
				}
				err = json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, err)
					return
				}
				emailTemplate := strings.ReplaceAll(inviteMemberEmail, "${invitationFrom}", html.EscapeString(member.Email))
				emailTemplate = strings.ReplaceAll(emailTemplate, "${organization}", html.EscapeString(organization.Name))
				err = organization.InviteMember(ctx, req.Email, emailTemplate)
				respond(w, err)
			})
			router.Get("/{token}", func(w http.ResponseWriter, r *http.Request) {
				invitationToken := chi.URLParam(r, "token")
				organizationName, email, err := s.apis.MemberInvitation(r.Context(), invitationToken)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"email": email, "organization": organizationName})
			})
			router.Put("/{token}", func(w http.ResponseWriter, r *http.Request) {
				invitationToken := chi.URLParam(r, "token")
				var req struct {
					Name     string
					Password string
				}
				err = json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, err)
					return
				}
				err = s.apis.AcceptInvitation(r.Context(), invitationToken, req.Name, req.Password)
				respond(w, err)
			})
		})
		router.Route("/current", func(router chi.Router) {
			router.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(member)
			})
			router.Put("/", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					MemberToSet struct {
						Name     string
						Image    []byte
						Email    string
						Password string
					}
				}
				err = json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, err)
					return
				}
				memberToSet := apis.MemberToSet{
					Name:     req.MemberToSet.Name,
					Email:    req.MemberToSet.Email,
					Password: req.MemberToSet.Password,
				}
				if req.MemberToSet.Image != nil {
					fileType := http.DetectContentType(req.MemberToSet.Image)
					avatar := &apis.Avatar{
						Image:    req.MemberToSet.Image,
						MimeType: fileType,
					}
					memberToSet.Avatar = avatar
				}
				err := organization.SetMember(ctx, session.Member, memberToSet)
				respond(w, err)
			})
		})
		router.Route("/{id}", func(router chi.Router) {
			router.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "id"))
				err := organization.DeleteMember(ctx, id)
				respond(w, err)
			})
		})
	})

	router.ServeHTTP(w, r)

}

// respond responds to the HTTP client writing on w, in case of error, and also
// writes on the log if the error is an internal server error.
func respond(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	if err, ok := err.(errors.ResponseWriterTo); ok {
		_ = err.WriteTo(w)
		return
	}
	slog.Error("error occurred serving APIs:", "err", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

// requiresLogin checks if path requires login.
func requiresLogin(path string, method string) bool {
	if strings.HasPrefix(path, "/api/members/invitations/") && path != "/api/members/invitations/" && (method == http.MethodGet || method == http.MethodPut) {
		return false
	}
	return true
}
