//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	_ "time/tzdata" // workaround for clickhouse-go issue #162

	"chichi/apis/postgres"
	"chichi/apis/types"
	_connector "chichi/connector"

	"github.com/ClickHouse/clickhouse-go/v2"
	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/go-chi/chi/v5"
)

var ErrCannotGetConnectorAccessToken = errors.New("cannot get access token")

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

	// Read all connectors.
	connectors := map[int]*Connector{}
	err = db.QueryScan("SELECT id, name, type, logo_url, webhooks_per, oauth_url, oauth_client_id,"+
		" oauth_client_secret, oauth_token_endpoint, oauth_default_token_type, oauth_default_expires_in,"+
		" oauth_forced_expires_in FROM connectors", func(rows *postgres.Rows) error {
		for rows.Next() {
			c := Connector{state: connectorState{resources: map[int]*Resource{}}}
			oauth := ConnectorOAuth{}
			if err := rows.Scan(&c.id, &c.name, &c.typ, &c.logoURL, &c.webhooksPer, &oauth.URL, &oauth.ClientID, &oauth.ClientSecret,
				&oauth.TokenEndpoint, &oauth.DefaultTokenType, &oauth.DefaultExpiresIn, &oauth.ForcedExpiresIn); err != nil {
				return err
			}
			if oauth.URL != "" {
				c.oAuth = &oauth
			}
			connectors[c.id] = &c
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	apis.Connectors = newConnectors(apis, connectors)

	// Read all resources.
	resources := map[int]*Resource{}
	err = db.QueryScan("SELECT id, connector, code, oauth_access_token, oauth_refresh_token, oauth_expires_in\n"+
		"FROM resources", func(rows *postgres.Rows) error {
		for rows.Next() {
			r := Resource{}
			var connectorID int
			if err := rows.Scan(&r.id, &connectorID, &r.code, &r.oAuthAccessToken, &r.oAuthRefreshToken, &r.oAuthExpiresIn); err != nil {
				return err
			}
			connector := connectors[connectorID]
			connector.state.resources[r.id] = &r
			resources[r.id] = &r
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Read all accounts.
	accounts := map[int]*Account{}
	err = db.QueryScan("SELECT id, name, email, internal_ips FROM accounts", func(rows *postgres.Rows) error {
		var id int
		var name, email, ips string
		for rows.Next() {
			if err := rows.Scan(&id, &name, &email, &ips); err != nil {
				return err
			}
			account := &Account{
				apis:        apis,
				db:          apis.db,
				chDB:        apis.chDB,
				id:          id,
				name:        name,
				email:       email,
				internalIPs: strings.Fields(ips),
			}
			account.Workspaces = newWorkspaces(account)
			accounts[id] = account
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	apis.Accounts = newAccounts(apis, accounts)

	// Read all workspaces.
	workspaces := map[int]*Workspace{}
	err = db.QueryScan("SELECT id, account, user_schema, group_schema, event_schema FROM workspaces",
		func(rows *postgres.Rows) error {
			var id, accountID int
			var userSchema, groupSchema, eventSchema string
			for rows.Next() {
				if err := rows.Scan(&id, &accountID, &userSchema, &groupSchema, &eventSchema); err != nil {
					return err
				}
				account := accounts[accountID]
				workspace := &Workspace{
					db:          db,
					chDB:        chDB,
					id:          id,
					account:     account,
					userSchema:  userSchema,
					groupSchema: groupSchema,
					eventSchema: eventSchema,
				}
				workspace.Connections = newConnections(workspace)
				workspace.EventListeners = &EventListeners{workspace}
				workspace.Transformations = &Transformations{workspace}
				account.Workspaces.state.ids[id] = workspace
				workspaces[id] = workspace
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	// Read all connections.
	connections := map[int]*Connection{}
	err = db.QueryScan("SELECT id, workspace, name, role, enabled, connector, COALESCE(storage, 0),"+
		" COALESCE(stream, 0), resource, website_host, user_cursor, identity_column, timestamp_column, settings,"+
		" schema, users_query FROM connections", func(rows *postgres.Rows) error {
		for rows.Next() {
			var workspaceID, connector, storage, stream, resource int
			var rawSchema string
			c := Connection{}
			if err := rows.Scan(&c.id, &workspaceID, &c.name, &c.role, &c.enabled, &connector, &storage, &stream, &resource,
				&c.websiteHost, &c.userCursor, &c.identityColumn, &c.timestampColumn, &c.settings, &rawSchema,
				&c.usersQuery); err != nil {
				return err
			}
			workspace := workspaces[workspaceID]
			c.account = workspace.account
			c.workspace = workspace
			c.connector = connectors[connector]
			if storage > 0 {
				if s, ok := connections[storage]; ok {
					c.storage = s
				} else {
					c.storage = &Connection{}
					connections[storage] = c.storage
				}
			}
			if stream > 0 {
				if s, ok := connections[stream]; ok {
					c.stream = s
				} else {
					c.stream = &Connection{}
					connections[stream] = c.stream
				}
			}
			if resource > 0 {
				c.resource = resources[resource]
			}
			if len(rawSchema) > 0 {
				c.schema, err = types.ParseSchema(strings.NewReader(rawSchema), nil)
				if err != nil {
					// TODO(marco) disable the connection instead of returning an error
					return err
				}
			}
			connection, ok := connections[c.id]
			if ok {
				*connection = c
			} else {
				connection = &Connection{}
				*connection = c
			}

			workspace.Connections.state.ids[c.id] = connection
			connections[c.id] = connection
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Read the source event stream collectors and the source connections that
	// send the events into the stream with their keys.
	var streams []*eventCollectorStream
	err = db.QueryScan(
		"SELECT s.id, co.name AS connector, s.settings, ci.id AS event_collector_producer, ci.type, k.key\n"+
			"FROM connections AS s\n"+
			"INNER JOIN connectors AS co ON co.id = s.connector\n"+
			"INNER JOIN connections AS ci ON ci.stream = s.id\n"+
			"INNER JOIN connections_keys AS k ON k.connection = ci.id\n"+
			"WHERE s.type = 'EventStream' AND s.role = 'Source' AND s.settings <> '' AND s.enabled AND ci.enabled",
		func(rows *postgres.Rows) error {
		Rows:
			for rows.Next() {
				var stream eventCollectorStream
				var producerID int
				var producerType ConnectorType
				var producerKey string
				if err := rows.Scan(&stream.ID, &stream.Connector, &stream.Settings, &producerID, &producerType, &producerKey); err != nil {
					return err
				}
				for _, s := range streams {
					if s.ID == stream.ID {
						for _, p := range s.Producers {
							if p.ID == producerID {
								p.Keys = append(p.Keys, producerKey)
								continue Rows
							}
						}
						s.Producers = append(s.Producers, &eventCollectorProducer{
							ID:   producerID,
							Type: _connector.Type(producerType),
							Keys: []string{producerKey},
						})
						continue Rows
					}
				}
				stream.Producers = []*eventCollectorProducer{{
					ID:   producerID,
					Type: _connector.Type(producerType),
					Keys: []string{producerKey},
				}}
				streams = append(streams, &stream)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}

	apis.eventCollector, err = newEventCollector(context.Background(), streams)
	if err != nil {
		return nil, err
	}

	// Read the all the source event stream processors.
	var allStreams []*eventProcessorStream
	err = db.QueryScan(
		"SELECT s.id, co.name AS connector, s.settings\n"+
			"FROM connections AS s\n"+
			"INNER JOIN connectors AS co ON co.id = s.connector\n"+
			"WHERE s.type = 'EventStream' AND s.role = 'Source' AND s.settings <> '' AND s.enabled",
		func(rows *postgres.Rows) error {
			for rows.Next() {
				var stream eventProcessorStream
				if err := rows.Scan(&stream.ID, &stream.Connector, &stream.Settings); err != nil {
					return err
				}
				allStreams = append(allStreams, &stream)
			}
			return nil
		})
	if err != nil {
		log.Fatal(err)
	}

	apis.eventProcessor = newEventProcessor(apis.db, apis.chDB, allStreams)
	go apis.eventProcessor.Run(context.Background())

	go apis.keepState(context.Background())

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
		if err == postgres.ErrNoRows {
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
			router.Get("/properties", func(w http.ResponseWriter, r *http.Request) {
				dsID, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if dsID <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				schema, err := workspace.Connections.Schema(dsID)
				if err != nil {
					if _, ok := err.(ConnectionNotFoundError); ok {
						http.Error(w, "Not Found", http.StatusNotFound)
					} else {
						log.Printf("[error] %s", err)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
					return
				}
				var properties []types.Property
				if schema.Valid() {
					properties = schema.Properties()
				} else {
					properties = []types.Property{}
				}
				_ = json.NewEncoder(w).Encode(properties)
			})
			router.Post("/import", func(w http.ResponseWriter, r *http.Request) {
				dsID, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if dsID <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				err = workspace.Connections.Import(dsID, false)
				if err != nil {
					if _, ok := err.(ConnectionNotFoundError); ok {
						http.Error(w, "Not Found", http.StatusNotFound)
					} else {
						log.Printf("[error] %s", err)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
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
					if _, ok := err.(ConnectionNotFoundError); ok {
						http.Error(w, "Not Found", http.StatusNotFound)
					} else {
						log.Printf("[error] %s", err)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
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
					if _, ok := err.(ConnectionNotFoundError); ok {
						http.Error(w, "Not Found", http.StatusNotFound)
					} else {
						log.Printf("[error] %s", err)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
					return
				}
			})
			router.Get("/transformations", func(w http.ResponseWriter, r *http.Request) {
				connection, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if connection <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				transformations, err := workspace.Connections.Transformations.List(connection)
				if err != nil {
					if _, ok := err.(ConnectionNotFoundError); ok {
						http.Error(w, "Not Found", http.StatusNotFound)
					} else {
						log.Printf("[error] cannot list transformations: %s", err)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
					return
				}
				_ = json.NewEncoder(w).Encode(transformations)
			})
			router.Put("/transformations", func(w http.ResponseWriter, r *http.Request) {
				connection, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if connection <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				var req []Transformation
				err := json.NewDecoder(r.Body).Decode(&req)
				if err != nil {
					http.Error(w, "Bad Request - invalid transformations", http.StatusBadRequest)
					return
				}
				err = workspace.Connections.Transformations.SaveAll(connection, req)
				if err != nil {
					if _, ok := err.(ConnectionNotFoundError); ok {
						http.Error(w, "Not Found", http.StatusNotFound)
					} else {
						log.Printf("[error] cannot save transformations: %s", err)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
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
