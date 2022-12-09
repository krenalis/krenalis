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
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"chichi/apis/types"
	_connector "chichi/connector"
	"chichi/pkg/open2b/sql"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/go-chi/chi/v5"
)

var (
	ErrResourceNotFound              = errors.New("resource does not exist")
	ErrCannotGetConnectorAccessToken = errors.New("cannot get access token")
)

type APIs struct {
	db             *sql.DB
	chDB           chDriver.Conn
	eventCollector *eventCollector
	eventProcessor *eventProcessor
	Accounts       *Accounts
	Users          *Users
}

var hasBeenCalled bool

// New returns an API instance. It can only be called once.
func New(db *sql.DB, chDB chDriver.Conn) *APIs {

	if hasBeenCalled {
		panic("apis.New has already been called")
	}
	hasBeenCalled = true

	apis := &APIs{db: db, chDB: chDB}
	apis.Users = &Users{apis}

	// Read all accounts.
	accounts := map[int]*Account{}
	err := db.QueryScan("SELECT id FROM accounts", func(rows *sql.Rows) error {
		var id int
		for rows.Next() {
			if err := rows.Scan(&id); err != nil {
				return err
			}
			accounts[id] = &Account{
				id:         id,
				apis:       apis,
				db:         apis.db,
				chDB:       apis.chDB,
				Workspaces: &Workspaces{},
			}
		}
		return nil
	})

	// Read all workspaces.
	err = db.QueryScan("SELECT id, account FROM workspaces", func(rows *sql.Rows) error {
		var id, accountID int
		for rows.Next() {
			if err := rows.Scan(&id, &accountID); err != nil {
				return err
			}
			account := accounts[accountID]
			if account.Workspaces.workspaces == nil {
				account.Workspaces.workspaces = map[int]*Workspace{}
			}
			workspace := &Workspace{
				workspace: id,
				account:   account,
				db:        db,
				chDB:      chDB,
			}
			workspace.Connections = &Connections{Workspace: workspace}
			workspace.EventListeners = &EventListeners{workspace}
			workspace.Transformations = &Transformations{workspace}
			account.Workspaces.workspaces[id] = workspace
		}
		return nil
	})

	apis.Accounts = &Accounts{apis, accounts}

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
		func(rows *sql.Rows) error {
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
		panic(err)
	}

	apis.eventCollector, err = newEventCollector(context.Background(), streams)
	if err != nil {
		panic(err)
	}

	// Read the all the source event stream processors.
	var allStreams []*eventProcessorStream
	err = db.QueryScan(
		"SELECT s.id, co.name AS connector, s.settings\n"+
			"FROM connections AS s\n"+
			"INNER JOIN connectors AS co ON co.id = s.connector\n"+
			"WHERE s.type = 'EventStream' AND s.role = 'Source' AND s.settings <> '' AND s.enabled",
		func(rows *sql.Rows) error {
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
		panic(err)
	}

	apis.eventProcessor = newEventProcessor(apis.db, apis.chDB, allStreams)
	go apis.eventProcessor.Run(context.Background())

	return apis
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
			var connections []*Connection
			connections, err = workspace.Connections.List()
			if err != nil {
				log.Printf("[error] %s", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
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
				dsID, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if dsID <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				transformations, err := workspace.Connections.Transformations.List(dsID)
				if err != nil {
					if _, ok := err.(ConnectionNotFoundError); ok {
						http.Error(w, "Not Found", http.StatusNotFound)
					} else {
						log.Printf("[error] %s", err)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
					return
				}
				_ = json.NewEncoder(w).Encode(transformations)
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
	router.Route("/api/transformations", func(router chi.Router) {
		router.Put("/", func(w http.ResponseWriter, r *http.Request) {
			var req TransformationToCreate
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			tID, err := workspace.Connections.Transformations.Create(req)
			if err != nil {
				if _, ok := err.(ConnectionNotFoundError); ok {
					http.Error(w, "Not Found", http.StatusNotFound)
				} else {
					log.Printf("[error] %s", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
				return
			}
			_ = json.NewEncoder(w).Encode(tID)
		})
		router.Patch("/{transformationID}", func(w http.ResponseWriter, r *http.Request) {
			tID, _ := strconv.Atoi(chi.URLParam(r, "transformationID"))
			if tID <= 0 {
				http.Error(w, "Bad Request: invalid transformation ID", http.StatusBadRequest)
				return
			}
			var req TransformationToUpdate
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			err = workspace.Connections.Transformations.Update(tID, req)
			if err != nil {
				log.Printf("[error] %s", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		})
	})

	router.ServeHTTP(w, r)

}

// refreshOAuthToken refreshes the access token of the resource with identifier
// id.
// Returns the ErrResourceNotFound error if the resource does not exist.
func (apis *APIs) refreshOAuthToken(resource int) (string, error) {

	var clientID, clientSecret, tokenEndpoint, refreshToken string
	err := apis.db.QueryRow(
		"SELECT c.oauth_client_id, c.oauth_client_secret, c.oauth_token_endpoint, r.oauth_refresh_token\n"+
			"FROM resources AS r\n"+
			"INNER JOIN connectors AS c ON c.id = r.connector\n"+
			"WHERE r.id = $1", resource).
		Scan(&clientID, &clientSecret, &tokenEndpoint, &refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrResourceNotFound
		}
		return "", err
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", "https://localhost:9090/admin/oauth/authorize")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusBadRequest {
			errData := struct {
				status string
			}{}
			err = json.NewDecoder(res.Body).Decode(&errData)
			if err != nil {
				return "", err
			}
			// TODO(@Andrea): check the status returned by services different
			// from Hubspot.
			if errData.status == "BAD_REFRESH_TOKEN" {
				return "", ErrCannotGetConnectorAccessToken
			}
		}
		return "", fmt.Errorf("unexpected status %d returned by connector while trying to get a new access token via refresh token", res.StatusCode)
	}

	response := struct {
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
	}{}
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&response)
	if err != nil {
		return "", err
	}

	// Convert expires_in into a timestamp.
	expiresIn := time.Now().UTC().Add(time.Duration(response.ExpiresIn) * time.Second) // TODO(marco): ExpiresIn should be relative to response time?

	_, err = apis.db.Exec(
		"UPDATE resources\n"+
			"SET oauth_access_token = $1, oauth_refresh_token = $2, oauth_expires_in = $3\n"+
			"WHERE id = $4",
		response.AccessToken, response.RefreshToken, expiresIn, resource)
	if err != nil {
		return "", err
	}

	return response.AccessToken, nil
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
