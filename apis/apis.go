//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"chichi/connectors"
	"chichi/pkg/open2b/sql"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var webhookPathReg = regexp.MustCompile(`^/webhook/(\d+)/`)

type APIs struct {
	myDB     *sql.DB
	chDB     chDriver.Conn
	Accounts *Accounts
	Users    *Users
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
	return apis
}

type AccountAPI struct {
	account     int
	apis        *APIs
	myDB        *sql.DB
	chDB        chDriver.Conn
	DataSources *DataSources
	Schemas     *Schemas
}

// AsAccount returns an API restricted to the given account.
func (apis *APIs) AsAccount(account int) *AccountAPI {
	api := &AccountAPI{account: account, apis: apis, myDB: apis.myDB, chDB: apis.chDB}
	api.DataSources = &DataSources{api}
	api.Schemas = &Schemas{api}
	return api
}

var importRegexp = regexp.MustCompile(`/apis/data-sources/((\d+)/((re)?import|properties|transformation))?`)

// ServeHTTP servers the API methods from HTTP.
func (apis *APIs) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	// Read the account.
	account, _ := strconv.Atoi(r.Header.Get("X-Account"))
	if account <= 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	api := apis.AsAccount(account)

	m := importRegexp.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var err error
	if m[1] == "" {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method Not Allowed", 405)
			return
		}
		var sources []*DataSource
		sources, err = api.DataSources.List()
		if err == nil {
			_ = json.NewEncoder(w).Encode(sources)
		}
	} else {
		id, _ := strconv.Atoi(m[2])
		if id <= 0 {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		switch m[3] {
		case "properties":
			var properties []*DataSourceProperty
			properties, err = api.DataSources.Properties(id)
			if err == nil {
				_ = json.NewEncoder(w).Encode(properties)
			}
		case "transformation":
			if r.Method == "GET" {
				var transformation string
				transformation, err = api.DataSources.TransformationFunc(id)
				if err == nil {
					w.Header().Set("Content-Type", "text/plain")
					_, _ = io.WriteString(w, transformation)
					return
				}
			} else {
				var transformation []byte
				transformation, err = io.ReadAll(r.Body)
				err = api.DataSources.SetTransformationFunc(id, string(transformation))
			}
		case "import":
			all := m[4] == "re"
			err = api.DataSources.Import(id, all)
		default:
			panic("unexpected path")
		}
	}
	if err != nil {
		if err == ErrConnectorNotFound {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		log.Printf("[error] call to %q failed: %s", r.URL.Path, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	return
}

// Errors returned to and handled by the ServeWebhook method.
var errBadRequest = errors.New("bad request")
var errNotFound = errors.New("not found")

// ServeWebhook serves a webhook request. The request path starts with
// "/webhook/{connector}/" where {connector} is a connector identifier.
func (apis *APIs) ServeWebhook(w http.ResponseWriter, r *http.Request) {
	err := apis.serveWebhook(r)
	if err != nil {
		switch err {
		case errBadRequest:
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		case errNotFound:
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		case connectors.ErrWebhookUnauthorized:
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Printf("cannot serve webhook: %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	return
}

// Connector represents a connector.
type Connector struct {
	ID            int
	Name          string
	OauthURL      string
	LogoURL       string
	ClientID      string
	ClientSecret  string
	TokenEndpoint string
}

// Connector returns the connector with the given identifier.
func (apis *APIs) Connector(id int) (*Connector, error) {
	connector := Connector{ID: id}
	err := apis.myDB.QueryRow("SELECT `name`, `oauthURL`, `logoURL`, `clientID`, `clientSecret`, `tokenEndpoint`\nFROM `connectors`\nWHERE `id` = ?", id).
		Scan(&connector.Name, &connector.OauthURL, &connector.LogoURL, &connector.ClientID, &connector.ClientSecret, &connector.TokenEndpoint)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &connector, nil
}

// Connectors returns all connectors.
func (apis *APIs) Connectors() ([]*Connector, error) {
	connectors := []*Connector{}
	err := apis.myDB.QueryScan("SELECT `id`, `name`, `oauthURL`, `logoURL`\nFROM `connectors`", func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var connector Connector
			if err = rows.Scan(&connector.ID, &connector.Name, &connector.OauthURL, &connector.LogoURL); err != nil {
				return err
			}
			connectors = append(connectors, &connector)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connectors, nil
}

// newFirehose returns a new firehose for the given connector.
// The returned firehouse does not have an assigned account. Use the
// api.newFirehose method to get a firehouse with an assigned account.
func (apis *APIs) newFirehose(connector int) *firehose {
	return &firehose{
		connector: connector,
		apis:      apis,
	}
}

func (apis *APIs) serveWebhook(r *http.Request) error {
	m := webhookPathReg.FindStringSubmatch(r.URL.Path)
	if m == nil {
		return errBadRequest
	}
	connID, _ := strconv.Atoi(m[1])
	if connID <= 0 {
		return errBadRequest
	}
	conn, err := apis.Connector(connID)
	if err != nil {
		return err
	}
	if conn == nil {
		return errNotFound
	}
	fh := apis.newFirehose(connID)
	connector := connectors.Connector(context.Background(), conn.Name, conn.ClientSecret, fh)
	return connector.ServeWebhook(r)
}

func (apis *APIs) initSchema() {

	apis.myDB.Scheme("Accounts", "accounts", struct {
		id          int
		name        string
		email       string
		password    string
		internalIPs string
	}{})

	apis.myDB.Scheme("Connectors", "connectors", struct {
		id            int
		oauthURL      string
		logoURL       string
		clientID      string
		clientSecret  string
		tokenEndpoint string
	}{})

	apis.myDB.Scheme("DataSources", "data_sources", struct {
		account                        int
		connector                      int
		accessToken                    string
		refreshToken                   string
		accessTokenExpirationTimestamp string
		transformation                 string
		userCursor                     string
	}{})

	apis.myDB.Scheme("DataSourcesProperties", "data_sources_properties", struct {
		account   int
		connector int
		name      string
		typ       string `sql:"type"`
		label     string
		options   string
		position  int
	}{})

	apis.myDB.Scheme("DataSourcesRawUserData", "data_sources_raw_users_data", struct {
		account   int
		connector int
		data      string
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

	apis.myDB.Scheme("Schemas", "schemas", struct {
		account     int
		userSchema  string
		groupSchema string
		eventSchema string
	}{})

	apis.myDB.Scheme("SmartEvents", "smart_events", struct {
		property int
		id       int
		name     string
		event    string
		pages    string
		buttons  string
	}{})

	apis.myDB.Scheme("Users", "users", struct {
		property int
		id       int
		device   string
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
