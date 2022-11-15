//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"chichi/pkg/open2b/sql"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/go-chi/chi/v5"
)

var (
	ErrResourceNotFound              = errors.New("resource does not exist")
	ErrCannotGetConnectorAccessToken = errors.New("cannot get access token")
)

type APIs struct {
	myDB     *sql.DB
	chDB     chDriver.Conn
	Accounts *Accounts
	Users    *Users

	eventsQueue      []*Event
	eventsQueueMutex sync.Mutex
}

var hasBeenCalled bool

// New returns an API instance. It can only be called once.
func New(myDB *sql.DB, chDB chDriver.Conn) *APIs {
	if hasBeenCalled {
		panic("apis.New has already been called")
	}
	hasBeenCalled = true
	apis := &APIs{myDB: myDB, chDB: chDB}
	apis.Accounts = &Accounts{apis}
	apis.Users = &Users{apis}
	apis.initSchema()
	apis.startEventFlusher()
	return apis
}

type AccountAPI struct {
	account int
	apis    *APIs
	myDB    *sql.DB
	chDB    chDriver.Conn
}

// AsAccount returns an API restricted to the given account.
func (apis *APIs) AsAccount(account int) *AccountAPI {
	api := &AccountAPI{account: account, apis: apis, myDB: apis.myDB, chDB: apis.chDB}
	return api
}

// AsWorkspace returns an API restricted to the given workspace.
func (api *AccountAPI) AsWorkspace(workspace int) *WorkspaceAPI {
	ws := &WorkspaceAPI{workspace: workspace, api: api, myDB: api.myDB, chDB: api.chDB}
	ws.Connections = &Connections{ws}
	ws.Transformations = &Transformations{ws}
	return ws
}

// ServeHTTP servers the API methods from HTTP.
func (apis *APIs) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if strings.HasPrefix(r.URL.Path, "/api/v1/events") {
		err := apis.serveEvents(w, r)
		if err != nil {
			switch err {
			case errBadRequest:
				http.Error(w, "Bad Request", http.StatusBadRequest)
			case errUnauthorized:
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			default:
				log.Printf("[error] %s", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Read the workspace.
	workspace, _ := strconv.Atoi(r.Header.Get("X-Workspace"))
	if workspace <= 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	// Read the account.
	var account int
	err := apis.myDB.QueryRow("SELECT `account` FROM `workspaces` WHERE `id` = ?", workspace).Scan(&account)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		log.Printf("[error] %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	api := apis.AsAccount(account)
	ws := api.AsWorkspace(workspace)

	router := chi.NewRouter()
	router.Route("/api/connections", func(router chi.Router) {
		router.Get("/", func(w http.ResponseWriter, r *http.Request) {
			var connections []*Connection
			connections, err = ws.Connections.List()
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
				var properties []ConnectionProperty
				properties, _, err = ws.Connections.Properties(dsID)
				if err != nil {
					if _, ok := err.(ConnectionNotFoundError); ok {
						http.Error(w, "Not Found", http.StatusNotFound)
					} else {
						log.Printf("[error] %s", err)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
					return
				}
				_ = json.NewEncoder(w).Encode(properties)
			})
			router.Post("/import", func(w http.ResponseWriter, r *http.Request) {
				dsID, _ := strconv.Atoi(chi.URLParam(r, "connectionID"))
				if dsID <= 0 {
					http.Error(w, "Bad Request: invalid connection ID", http.StatusBadRequest)
					return
				}
				err = ws.Connections.Import(dsID, false)
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
				err = ws.Connections.Import(dsID, true)
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
				transformations, err := ws.Connections.Transformations.List(dsID)
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
	router.Route("/api/transformations", func(router chi.Router) {
		router.Put("/", func(w http.ResponseWriter, r *http.Request) {
			var req TransformationToCreate
			err := json.NewDecoder(r.Body).Decode(&req)
			if err != nil {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
			tID, err := ws.Connections.Transformations.Create(req)
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
			err = ws.Connections.Transformations.Update(tID, req)
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
	err := apis.myDB.QueryRow(
		"SELECT `c`.`oAuthClientID`, `c`.`oAuthClientSecret`, `c`.`oAuthTokenEndpoint`, `r`.`oAuthRefreshToken`\n"+
			"FROM `resources` AS `r`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `r`.`connector`\n"+
			"WHERE `r`.`id` = ?", resource).
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

	_, err = apis.myDB.Exec(
		"UPDATE `resources`\n"+
			"SET `oAuthAccessToken` = ?, `oAuthRefreshToken` = ?, `oAuthExpiresIn` = ?\n"+
			"WHERE `id` = ?",
		response.AccessToken, response.RefreshToken, expiresIn, resource)
	if err != nil {
		return "", err
	}

	return response.AccessToken, nil
}

func (apis *APIs) initSchema() {

	apis.myDB.Scheme("Accounts", "accounts", struct {
		id          int
		name        string
		email       string
		password    string
		internalIPs string
	}{})

	apis.myDB.Scheme("Connections", "connections", struct {
		id              int
		workspace       int
		typ             int `sql:"type"`
		role            int
		connector       int
		storage         int
		resource        string
		websiteHost     string
		userCursor      string
		identityColumn  string
		timestampColumn string
		settings        string
		properties      string
		usedProperties  string
		usersQuery      string
	}{})

	apis.myDB.Scheme("ConnectionsKeys", "connections_keys", struct {
		connection int
		position   int
		key        string
	}{})

	apis.myDB.Scheme("ConnectionsImports", "connections_imports", struct {
		id         int
		connection int
		storage    int
		startTime  time.Time
		endTime    time.Time
		error      string
	}{})

	apis.myDB.Scheme("ConnectionsStats", "connections_stats", struct {
		connection int
		timeSlot   int
		usersIn    int
	}{})

	apis.myDB.Scheme("ConnectionsUsers", "connections_users", struct {
		connection   int
		user         string
		data         string
		timestamps   string
		goldenRecord int
	}{})

	apis.myDB.Scheme("Connectors", "connectors", struct {
		id                    int
		name                  string
		typ                   int `sql:"type"`
		logoURL               string
		webhooksPer           WebhooksPer
		oAuthURL              string
		oAuthClientID         string
		oAuthClientSecret     string
		oAuthTokenEndpoint    string
		oAuthDefaultTokenType string
		oAuthDefaultExpiresIn int
		oAuthForcedExpiresIn  string
	}{})

	apis.myDB.Scheme("Devices", "devices", struct {
		property int
		id       string
		user     int
	}{})

	apis.myDB.Scheme("Domains", "domains", struct {
		property int
		name     string
	}{})

	apis.myDB.Scheme("Properties", "properties", struct {
		id      int
		code    string
		account int
	}{})

	apis.myDB.Scheme("SmartEvents", "smart_events", struct {
		property int
		id       int
		name     string
		event    string
		pages    string
		buttons  string
	}{})

	apis.myDB.Scheme("Transformations", "transformations", struct {
		id               int
		goldenRecordName string
		sourceCode       string
	}{})

	apis.myDB.Scheme("TransformationsConnections", "transformations_connections", struct {
		connection     int
		property       string
		transformation int
	}{})

	apis.myDB.Scheme("Users", "users", struct {
		property int
		id       int
		device   string
	}{})

	apis.myDB.Scheme("Workspaces", "workspaces", struct {
		id          int
		account     int
		name        string
		userSchema  string
		groupSchema string
		eventSchema string
	}{})

}

// DeprecatedProperty returns an instance of DeprecatedProperties which operates
// on the given property.
func (api *AccountAPI) DeprecatedProperty(property int) *DeprecatedProperties {
	properties := &DeprecatedProperties{
		AccountAPI: api,
		id:         property,
	}
	properties.SmartEvents = &SmartEvents{properties}
	properties.Visualization = &Visualization{properties}
	return properties
}
