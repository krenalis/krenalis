//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/connector/types"
	"chichi/telemetry"

	"github.com/go-chi/chi/v5"
)

// ServeHTTP servers the API methods from HTTP.
func (apis *APIs) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, span := telemetry.TraceSpan(r.Context(), "apis.ServeHTTP", "ip", r.RemoteAddr, "urlPath", r.URL.Path)
	defer span.End()

	telemetry.IncrementCounter(ctx, "apis.ServeHTTP", 1)

	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		apis.events.ServeHTTP(w, r)
		return
	}

	// Read the workspace.
	workspaceID, _ := strconv.Atoi(r.Header.Get("X-Workspace"))
	if workspaceID < 1 || workspaceID > math.MaxInt32 {
		http.Error(w, "Bad Request (missing 'X-Workspace' header)", http.StatusBadRequest)
		return
	}
	// Read the account.
	var accountID int
	err := apis.db.QueryRow(context.Background(), "SELECT account FROM workspaces WHERE id = $1", workspaceID).Scan(&accountID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		log.Printf("[error] %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	account, err := apis.Account(ctx, accountID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	workspace, err := account.Workspace(workspaceID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	router := chi.NewRouter()
	router.Route("/api/connections", func(router chi.Router) {
		router.Get("/", func(w http.ResponseWriter, r *http.Request) {
			connections := workspace.Connections()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(connections)
		})
		router.Route("/{connectionID}", func(router chi.Router) {
			router.Get("/", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(connection)
			})
			router.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				err = connection.Delete()
				respond(w, err)
			})
			router.Post("/actions", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				var req struct {
					Target    ActionTarget
					EventType string
					Action    ActionToSet
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
				connection, err := workspace.Connection(id)
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
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				var req ActionToSet
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
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				action, err := connection.Action(ctx, actionID)
				if err != nil {
					respond(w, err)
					return
				}
				err = action.Delete()
				respond(w, err)
			})
			router.Post("/actions/{actionID}/execute", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				actionID, _ := strconv.Atoi(chi.URLParam(r, "actionID"))
				connection, err := workspace.Connection(id)
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
				connection, err := workspace.Connection(id)
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
					SchedulePeriod SchedulePeriod
				}
				err = json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, errors.BadRequest("invalid JSON"))
					return
				}
				err = action.SetSchedulePeriod(req.SchedulePeriod)
				respond(w, err)
			})
			router.Post("/actions/{actionID}/status", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				actionID, _ := strconv.Atoi(chi.URLParam(r, "actionID"))
				connection, err := workspace.Connection(id)
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
				err = action.SetStatus(req.Enabled)
				respond(w, err)
			})
			router.Route("/action-schemas", func(router chi.Router) {
				router.Get("/Users", func(w http.ResponseWriter, r *http.Request) {
					id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
					connection, err := workspace.Connection(id)
					if err != nil {
						respond(w, err)
						return
					}
					schemas, err := connection.ActionSchemas(ctx, UsersTarget, "")
					if err != nil {
						respond(w, err)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(schemas)
				})
				router.Get("/Groups", func(w http.ResponseWriter, r *http.Request) {
					id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
					connection, err := workspace.Connection(id)
					if err != nil {
						respond(w, err)
						return
					}
					schemas, err := connection.ActionSchemas(ctx, GroupsTarget, "")
					if err != nil {
						respond(w, err)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(schemas)
				})
				router.Get("/Events", func(w http.ResponseWriter, r *http.Request) {
					id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
					connection, err := workspace.Connection(id)
					if err != nil {
						respond(w, err)
						return
					}
					schemas, err := connection.ActionSchemas(ctx, EventsTarget, "")
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
					connection, err := workspace.Connection(id)
					if err != nil {
						respond(w, err)
						return
					}
					schemas, err := connection.ActionSchemas(ctx, EventsTarget, eventType)
					if err != nil {
						respond(w, err)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(schemas)
				})
			})
			router.Get("/complete-path/{path}", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
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
				completePath, err := connection.CompletePath(path)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"path": completePath})
			})
			router.Get("/records", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				path := r.URL.Query().Get("path")
				sheet := r.URL.Query().Get("sheet")
				limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
				if err != nil {
					respond(w, errors.BadRequest("limit is not valid"))
					return
				}
				records, schema, err := connection.Records(path, sheet, limit)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"records": records, "schema": schema})
			})
			router.Get("/sheets", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				path := r.URL.Query().Get("path")
				sheets, err := connection.Sheets(path)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"sheets": sheets})
			})
			router.Post("/status", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
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
				err = connection.SetStatus(req.Enabled)
				respond(w, err)
			})
			router.Get("/imports", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				executions, err := connection.Executions()
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(executions)

			})
			router.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				var stats *ConnectionsStats
				stats, err = connection.Stats()
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(stats)
			})
			router.Get("/ui", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				form, err := connection.ServeUI("load", nil)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Add("Content-Type", "application/json")
				_, _ = w.Write(form)
			})
			router.Post("/ui-event", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
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
				form, err := connection.ServeUI(req.Event, req.Values)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Add("Content-Type", "application/json")
				_, _ = w.Write(form)
			})
			router.Post("/exec-query", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
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
				rows, schema, err := connection.ExecQuery(req.Query, req.Limit)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Add("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"Rows": rows, "Schema": schema})
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
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				schema, err := connection.TableSchema(table)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(schema)
			})
			router.Get("/keys", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
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
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				var key string
				key, err = connection.GenerateKey()
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(key)
			})
			router.Delete("/keys/{key}", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				key := chi.URLParam(r, "key")
				err = connection.RevokeKey(key)
				respond(w, err)
			})
			router.Post("/storage", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					respond(w, err)
					return
				}
				var req struct {
					Storage     int
					Compression string
				}
				err = json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, err)
					return
				}
				err = connection.SetStorage(req.Storage, Compression(req.Compression))
				respond(w, err)
			})
		})
	})
	router.Route("/api/connectors", func(router chi.Router) {
		router.Get("/", func(w http.ResponseWriter, r *http.Request) {
			connectors := apis.Connectors(ctx)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(connectors)
		})
		router.Route("/{connectorID}", func(router chi.Router) {
			router.Get("/", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectorID"))
				connector, err := apis.Connector(ctx, id)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(connector)
			})
			router.Post("/ui", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectorID"))
				connector, err := apis.Connector(ctx, id)
				if err != nil {
					respond(w, err)
					return
				}
				var req struct {
					Role       string
					OAuthToken string
				}
				err = json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, err)
					return
				}
				var role ConnectionRole
				switch req.Role {
				case "Source":
					role = SourceRole
				case "Destination":
					role = DestinationRole
				default:
					respond(w, errors.BadRequest("unexpected connection role '%s'", req.Role))
					return
				}
				form, err := connector.ServeUI("load", nil, role, req.OAuthToken)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Add("Content-Type", "application/json")
				_, _ = w.Write(form)
			})
			router.Get("/auth-code-url", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectorID"))
				connector, err := apis.Connector(ctx, id)
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
			router.Post("/ui-event", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectorID"))
				connector, err := apis.Connector(ctx, id)
				if err != nil {
					respond(w, err)
					return
				}
				var req struct {
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
				var role ConnectionRole
				switch req.Role {
				case "Source":
					role = SourceRole
				case "Destination":
					role = DestinationRole
				default:
					respond(w, errors.BadRequest("unexpected connection role '%s'", req.Role))
					return
				}
				form, err := connector.ServeUI(req.Event, req.Values, role, req.OAuthToken)
				if err != nil {
					respond(w, err)
					return
				}
				w.Header().Add("Content-Type", "application/json")
				_, _ = w.Write(form)
			})
		})
	})
	router.Route("/api/event-listeners", func(router chi.Router) {
		router.Put("/", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Size   *int
				Source int
				Server int
				Stream int
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
			id, err := workspace.AddEventListener(size, req.Source, req.Server, req.Stream)
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
	router.Route("/api/users", func(router chi.Router) {
		router.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Properties []string
				Start      int
				End        int
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				respond(w, errors.BadRequest("invalid JSON"))
				return
			}
			schema, users, err := workspace.Users(req.Properties, "", 0, 1000)
			if err != nil {
				respond(w, err)
				return
			}
			var end int
			if len(users) < req.End {
				end = len(users)
			} else {
				end = req.End
			}
			w.Header().Add("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"count":  len(users),
				"users":  users[req.Start:end],
				"schema": schema,
			})
		})
		router.Get("/{userID}/events", func(w http.ResponseWriter, r *http.Request) {
			id, _ := strconv.Atoi(chi.URLParam(r, "userID"))
			user, err := workspace.User(id)
			if err != nil {
				respond(w, err)
				return
			}
			events, err := user.Events(10)
			if err != nil {
				respond(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"events": events,
			})
		})
		router.Get("/{userID}/traits", func(w http.ResponseWriter, r *http.Request) {
			id, _ := strconv.Atoi(chi.URLParam(r, "userID"))
			user, err := workspace.User(id)
			if err != nil {
				respond(w, err)
				return
			}
			traits, err := user.Traits()
			if err != nil {
				respond(w, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"traits": traits,
			})
		})
	})
	router.Route("/api/workspace", func(router chi.Router) {
		router.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(workspace)
		})
		router.Route("/anonymous-identifiers", func(router chi.Router) {
			router.Post("/", func(w http.ResponseWriter, r *http.Request) {
				req := struct {
					AnonymousIdentifiers AnonymousIdentifiers
				}{}
				err := json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, errors.BadRequest("invalid JSON"))
					return
				}
				err = workspace.SetAnonymousIdentifiers(req.AnonymousIdentifiers)
				respond(w, err)
			})
		})
		router.Route("/connect-redis", func(router chi.Router) {
			router.Post("/", func(w http.ResponseWriter, r *http.Request) {
				req := struct {
					Settings json.RawMessage
				}{}
				err := json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, errors.BadRequest("invalid JSON"))
					return
				}
				err = workspace.ConnectRedis(req.Settings)
				respond(w, err)
			})
		})
		router.Route("/connect-warehouse", func(router chi.Router) {
			router.Post("/", func(w http.ResponseWriter, r *http.Request) {
				req := struct {
					Type     WarehouseType
					Settings json.RawMessage
				}{}
				err := json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, errors.BadRequest("invalid JSON"))
					return
				}
				err = workspace.ConnectWarehouse(req.Type, req.Settings)
				respond(w, err)
			})
		})
		router.Route("/disconnect-redis", func(router chi.Router) {
			router.Post("/", func(w http.ResponseWriter, r *http.Request) {
				err = workspace.DisconnectRedis()
				respond(w, err)
			})
		})
		router.Route("/disconnect-warehouse", func(router chi.Router) {
			router.Post("/", func(w http.ResponseWriter, r *http.Request) {
				err = workspace.DisconnectWarehouse()
				respond(w, err)
			})
		})
		router.Route("/init-warehouse", func(router chi.Router) {
			router.Post("/", func(w http.ResponseWriter, r *http.Request) {
				err = workspace.InitWarehouse()
				respond(w, err)
			})
		})
		router.Route("/reload-schemas", func(router chi.Router) {
			router.Post("/", func(w http.ResponseWriter, r *http.Request) {
				err = workspace.ReloadSchemas()
				respond(w, err)
			})
		})
		router.Route("/user-schema", func(router chi.Router) {
			router.Get("/", func(w http.ResponseWriter, r *http.Request) {
				schema := workspace.Schema("users")
				w.Header().Add("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(schema)
			})
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
				oauthToken, err := workspace.OAuthToken(req.OAuthCode, req.RedirectURI, req.Connector)
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
					Connector int
					Role      string
					Settings  json.RawMessage
					Options   ConnectionOptions
				}
				err := json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, errors.BadRequest("invalid JSON"))
					return
				}
				var role ConnectionRole
				switch req.Role {
				case "Source":
					role = SourceRole
				case "Destination":
					role = DestinationRole
				default:
					respond(w, errors.BadRequest("unexpected connection role '%s'", req.Role))
					return
				}
				id, err := workspace.AddConnection(role, req.Connector, req.Settings, req.Options)
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
			router.Post("/", func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					PrivacyRegion PrivacyRegion
				}
				err := json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					respond(w, errors.BadRequest("invalid JSON"))
					return
				}
				err = workspace.SetPrivacyRegion(req.PrivacyRegion)
				respond(w, err)
			})
		})
	})
	router.Get("/api/events-schema", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(events.Schema.Unflatten())
	})
	router.Post("/api/validate-expression", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Expression                  string
			Schema                      types.Type
			DestinationPropertyType     types.Type
			DestinationPropertyNullable bool
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respond(w, errors.BadRequest("invalid JSON"))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		message := apis.validateExpression(req.Expression, req.Schema, req.DestinationPropertyType, req.DestinationPropertyNullable)
		_ = json.NewEncoder(w).Encode(message)
	})
	router.Post("/api/expressions-properties", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Expressions []ExpressionToBeExtracted
			Schema      types.Type
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respond(w, errors.BadRequest("invalid JSON"))
			return
		}
		properties, err := apis.expressionsProperties(req.Expressions, req.Schema)
		if err != nil {
			respond(w, errors.BadRequest(err.Error()))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(properties)
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
	log.Printf("[error] %s", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
