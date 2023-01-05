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
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	_ "time/tzdata" // workaround for clickhouse-go issue #162

	"github.com/ClickHouse/clickhouse-go/v2"
	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/go-chi/chi/v5"

	"chichi/apis/errors"
	"chichi/apis/postgres"
)

var InvalidDefinition errors.Code = "InvalidDefinition"

type APIs struct {
	db             *postgres.DB
	chDB           chDriver.Conn
	eventCollector *eventCollector
	eventProcessor *eventProcessor
	Accounts       *Accounts
	Connectors     *Connectors
	Users          *Users
}

var hasBeenCalled bool

type Config struct {
	PostgreSQL PostgreSQLConfig
	ClickHouse ClickHouseConfig
}

type PostgreSQLConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

type ClickHouseConfig struct {
	Address  string
	Username string
	Password string
	Database string
}

// New returns an API instance. It can only be called once.
func New(conf *Config) (*APIs, error) {

	if hasBeenCalled {
		return nil, errors.New("apis.New has already been called")
	}
	hasBeenCalled = true

	// Open connection to PostgreSQL.
	ps := conf.PostgreSQL
	db, err := postgres.Open(&postgres.Options{
		Host:     ps.Host,
		Port:     ps.Port,
		Username: ps.Username,
		Password: ps.Password,
		Database: ps.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to PostreSQL: %s", err)
	}
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("cannot ping PostreSQL: %s", err)
	}

	// Open connection to ClickHouse.
	ch := conf.ClickHouse
	chDB, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{conf.ClickHouse.Address},
		Auth: clickhouse.Auth{
			Database: ch.Database,
			Username: ch.Username,
			Password: ch.Password,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot connect to ClickHouse: %s", err)
	}
	err = chDB.Ping(context.Background())
	if err != nil {
		log.Printf("[warning] cannot ping ClickHouse server: %s", err)
	}

	apis := &APIs{db: db, chDB: chDB}
	apis.Users = &Users{apis}

	err = startStateKeeper(context.Background(), apis)
	if err != nil {
		log.Fatalf("[error] cannot load state: %s", err)
	}

	return apis, nil
}

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

	account, err := apis.Accounts.As(accountID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	workspace, err := account.Workspaces.As(workspaceID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	router := chi.NewRouter()
	router.Route("/api/connections", func(router chi.Router) {
		router.Get("/", func(w http.ResponseWriter, r *http.Request) {
			connections := workspace.Connections.List()
			_ = json.NewEncoder(w).Encode(connections)
		})
		router.Route("/{connectionID}", func(router chi.Router) {
			router.Get("/schema", func(w http.ResponseWriter, r *http.Request) {
				dsID, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if dsID <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				schema, err := workspace.Connections.Schema(dsID)
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
				dsID, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if dsID <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				err = workspace.Connections.Import(dsID, false)
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
				connection, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if connection <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				err = workspace.Connections.Export(connection)
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
				dsID, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if dsID <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				err = workspace.Connections.Import(dsID, true)
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
				connection, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if connection <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				mappings, err := workspace.Connections.Mappings(connection)
				if err != nil {
					if err, ok := err.(errors.ResponseWriterTo); ok {
						_ = err.WriteTo(w)
						return
					}
					log.Printf("[error] cannot list mappings: %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				mappingsInfos := make([]*MappingInfo, len(mappings))
				for i, m := range mappings {
					mappingsInfos[i] = &MappingInfo{
						ID:             m.id,
						In:             m.in,
						PredefinedFunc: m.predefinedFunc,
						SourceCode:     m.sourceCode,
						Out:            m.out,
					}
				}
				// TODO(Gianluca): this is a workaround that will be removed
				// when one-to-one mappings will be supported in the UI.
				for _, m := range mappingsInfos {
					if m.SourceCode == "" {
						m.SourceCode = "# one-to-one"
					}
				}
				_ = json.NewEncoder(w).Encode(mappingsInfos)
			})
			router.Put("/mappings", func(w http.ResponseWriter, r *http.Request) {
				connection, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if connection <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				var mappings []*MappingToCreate
				err := json.NewDecoder(r.Body).Decode(&mappings)
				if err != nil {
					http.Error(w, "Bad Request - invalid mappings", http.StatusBadRequest)
					return
				}
				// TODO(Gianluca): this is a workaround that will be removed
				// when one-to-one mappings will be supported in the UI.
				for _, m := range mappings {
					if m.SourceCode == "# one-to-one" {
						m.SourceCode = ""
					}
				}
				err = workspace.Connections.SetMappings(connection, mappings)
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
			id, err := workspace.EventListeners.Add(size, req.Source, req.Server, req.Stream)
			if err != nil {
				log.Printf("[error] %s", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
		})
		router.Delete("/{listenerID}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "listenerID")
			workspace.EventListeners.Remove(id)
		})
		router.Get("/{listenerID}/events", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "listenerID")
			events, discarded, err := workspace.EventListeners.Events(id)
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
			var setts PostgreSQLSettings
			err := json.NewDecoder(r.Body).Decode(&setts)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			err = workspace.ConnectWarehouse(&setts)
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
	router.Route("/api/workspace/reload-schema", func(router chi.Router) {
		router.Post("/", func(w http.ResponseWriter, r *http.Request) {
			err = workspace.ReloadSchema()
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

// DeprecatedProperty returns an instance of DeprecatedProperties which operates
// on the given property.
func (api *Account) DeprecatedProperty(property int) *DeprecatedProperties {
	properties := &DeprecatedProperties{
		Account: api,
		id:      property,
	}
	properties.SmartEvents = &SmartEvents{properties}
	properties.Visualization = &Visualization{properties}
	return properties
}
