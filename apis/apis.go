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
	myDB       *sql.DB
	chDB       chDriver.Conn
	Connectors *Connectors
	Cursors    *Cursors
	Customers  *Customers
	Schemas    *Schemas
	Properties *Properties
	Users      *Users
}

var hasBeenCalled bool

// New returns an API instance. It can only be called once.
func New(myDB *sql.DB, chDB chDriver.Conn) *APIs {
	if hasBeenCalled {
		panic("apis.New has already been called")
	}
	hasBeenCalled = true
	apis := &APIs{myDB: myDB, chDB: chDB}
	apis.Connectors = &Connectors{apis}
	apis.Cursors = &Cursors{apis}
	apis.Customers = &Customers{apis}
	apis.Schemas = &Schemas{apis}
	apis.Properties = &Properties{apis}
	apis.Users = &Users{apis}
	apis.initSchema()
	return apis
}

type API struct {
	myDB     *sql.DB
	chDB     chDriver.Conn
	customer int
}

func (apis *APIs) API(customer int) *API {
	return &API{myDB: apis.myDB, chDB: apis.chDB, customer: customer}
}

var importRegexp = regexp.MustCompile(`/apis/connectors/(\d+)/((re)?import|properties|transformation)`)

// ServeHTTP servers the API methods from HTTP.
func (apis *APIs) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	m := importRegexp.FindStringSubmatch(r.URL.Path)
	if m != nil {
		id, _ := strconv.Atoi(m[1])
		if id <= 0 {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		var err error
		switch m[2] {
		case "properties":
			var properties []*ConnectorProperty
			properties, err = apis.Connectors.Properties(id)
			if err == nil {
				_ = json.NewEncoder(w).Encode(properties)
			}
		case "transformation":
			if r.Method == "GET" {
				var transformation string
				transformation, err = apis.Connectors.TransformationFunc(id)
				if err == nil {
					w.Header().Set("Content-Type", "text/plain")
					_, _ = io.WriteString(w, transformation)
					return
				}
			} else {
				var transformation []byte
				transformation, err = io.ReadAll(r.Body)
				err = apis.Connectors.SetTransformationFunc(id, string(transformation))
			}
		default:
			all := m[3] == "re"
			err = apis.Connectors.Import(id, all)
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

	http.Error(w, "Bad Request", http.StatusBadRequest)
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

func (apis *APIs) serveWebhook(r *http.Request) error {
	m := webhookPathReg.FindStringSubmatch(r.URL.Path)
	if m == nil {
		return errBadRequest
	}
	connID, _ := strconv.Atoi(m[1])
	if connID <= 0 {
		return errBadRequest
	}
	conn, err := apis.Connectors.Get(connID)
	if err != nil {
		return err
	}
	if conn == nil {
		return errNotFound
	}
	fh := apis.NewFirehose(connID, 1)
	connector := connectors.Connector(context.Background(), conn.Name, conn.ClientSecret, fh)
	return connector.ServeWebhook(r)
}

func (apis *APIs) initSchema() {

	apis.myDB.Scheme("Customers", "customers", struct {
		id          int
		name        string
		email       string
		password    string
		internalIPs string
	}{})

	apis.myDB.Scheme("ConnectorsProperties", "connectors_properties", struct {
		account   int
		connector int
		name      string
		typ       string `sql:"type"`
		label     string
		options   string
		position  int
	}{})

	apis.myDB.Scheme("ConnectorsRawUserData", "connectors_raw_users_data", struct {
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
		id       int
		code     string
		customer int
	}{})

	apis.myDB.Scheme("Schemas", "schemas", struct {
		account      int
		user_schema  string
		group_schema string
		event_schema string
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

	apis.myDB.Scheme("AccountConnectors", "account_connectors", struct {
		account                           int
		connector                         int
		access_token                      string
		refresh_token                     string
		access_token_expiration_timestamp string
		transformation                    string
		user_cursor                       string
	}{})

}

// DeprecatedProperty returns an instance of DeprecatedProperties which operates
// on the given property.
func (api *API) DeprecatedProperty(property int) *DeprecatedProperties {
	properties := &DeprecatedProperties{
		API: api,
		id:  property,
	}
	properties.SmartEvents = &SmartEvents{properties}
	properties.Visualization = &Visualization{properties}
	return properties
}
