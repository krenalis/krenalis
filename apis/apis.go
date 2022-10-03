//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"chichi/pkg/open2b/sql"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type APIs struct {
	myDB            *sql.DB
	chDB            chDriver.Conn
	Connectors      *Connectors
	Customers       *Customers
	Schemas         *Schemas
	Transformations *Transformations
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
	apis.Customers = &Customers{apis}
	apis.Schemas = &Schemas{apis}
	apis.Transformations = &Transformations{apis}
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

func (apis *APIs) initSchema() {

	apis.myDB.Scheme("Customers", "customers", struct {
		id          int
		name        string
		email       string
		password    string
		internalIPs string
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
	}{})

}

// Property returns an instance of Properties which operates on the given
// property.
func (api *API) Property(property int) *Properties {
	properties := &Properties{
		API: api,
		id:  property,
	}
	properties.SmartEvents = &SmartEvents{properties}
	properties.Visualization = &Visualization{properties}
	return properties
}
