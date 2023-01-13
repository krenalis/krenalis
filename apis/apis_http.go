//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"chichi/apis/errors"

	"github.com/go-chi/chi/v5"
)

// ServeHTTP servers the API methods from HTTP.
func (apis *APIs) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if strings.HasPrefix(r.URL.Path, "/api/v1/events") {
		if apis.eventCollector == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		apis.eventCollector.ServeHTTP(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Read the workspace.
	workspaceID, _ := strconv.Atoi(r.Header.Get("X-Workspace"))
	if workspaceID <= 0 {
		http.Error(w, "Bad Request (missing 'X-Workspace' header)", http.StatusBadRequest)
		return
	}
	// Read the account.
	var accountID int
	err := apis.db.QueryRow("SELECT account FROM workspaces WHERE id = $1", workspaceID).Scan(&accountID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		log.Printf("[error] %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	account, err := apis.Account(accountID)
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
			_ = json.NewEncoder(w).Encode(connections)
		})
		router.Route("/{connectionID}", func(router chi.Router) {
			router.Get("/schema", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				schema, err := connection.Schema()
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				if schema.Valid() {
					_ = json.NewEncoder(w).Encode(schema)
				} else {
					_, _ = w.Write([]byte("null"))
				}
			})
			router.Post("/import", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				err = connection.Import(false)
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			})
			router.Post("/export", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				err = connection.Export()
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			})
			router.Post("/reimport", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				err = connection.Import(true)
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
			})
			router.Get("/mappings", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				_ = json.NewEncoder(w).Encode(connection.Mappings)
			})
			router.Put("/mappings", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				connection, err := workspace.Connection(id)
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				var mappings []*MappingToCreate
				err = json.NewDecoder(r.Body).Decode(&mappings)
				if err != nil {
					http.Error(w, "Bad Request - invalid mappings", http.StatusBadRequest)
					return
				}
				err = connection.SetMappings(mappings)
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] cannot save mappings: %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				_, _ = w.Write([]byte(`{"status":"ok"}`))
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
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			var size = 10
			if req.Size != nil {
				size = *req.Size
			}
			id, err := workspace.EventListeners().Add(size, req.Source, req.Server, req.Stream)
			if err != nil {
				log.Printf("[error] %s", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
		})
		router.Delete("/{listenerID}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "listenerID")
			workspace.EventListeners().Remove(id)
		})
		router.Get("/{listenerID}/events", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "listenerID")
			events, discarded, err := workspace.EventListeners().Events(id)
			if err != nil {
				log.Printf("[error] %s", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
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
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			schema, users, err := workspace.Users(req.Properties, "", 0, 1000)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %s", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
	})
	router.Route("/api/workspace/connect-warehouse", func(router chi.Router) {
		router.Post("/", func(w http.ResponseWriter, r *http.Request) {
			req := struct {
				Type     WarehouseType
				Settings json.RawMessage
			}{}
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			err = workspace.ConnectWarehouse(req.Type, req.Settings)
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		})
	})
	router.Route("/api/workspace/disconnect-warehouse", func(router chi.Router) {
		router.Post("/", func(w http.ResponseWriter, r *http.Request) {
			err = workspace.DisconnectWarehouse()
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		})
	})
	router.Route("/api/workspace/init-warehouse", func(router chi.Router) {
		router.Post("/", func(w http.ResponseWriter, r *http.Request) {
			err = workspace.InitWarehouse()
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		})
	})
	router.Route("/api/workspace/reload-schemas", func(router chi.Router) {
		router.Post("/", func(w http.ResponseWriter, r *http.Request) {
			err = workspace.ReloadSchemas()
			if err != nil {
				if err, ok := err.(errors.ResponseWriterTo); ok {
					_ = err.WriteTo(w)
					return
				}
				log.Printf("[error] %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		})
	})
	router.ServeHTTP(w, r)

}
