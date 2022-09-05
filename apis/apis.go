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
	Customers *Customers
	myDB      *sql.DB
	chDB      chDriver.Conn
}

var hasBeenCalled bool

// New returns an API instance. It can only be called once.
func New(myDB *sql.DB, chDB chDriver.Conn) *APIs {
	if hasBeenCalled {
		panic("apis.New has already been called")
	}
	hasBeenCalled = true
	apis := &APIs{myDB: myDB, chDB: chDB}
	apis.Customers = &Customers{apis}
	return apis
}

type API struct {
	Properties *Properties
	myDB       *sql.DB
	chDB       chDriver.Conn
	customer   int
}

func (apis *APIs) API(customer int) *API {
	api := &API{myDB: apis.myDB, chDB: apis.chDB, customer: customer}
	api.Properties = &Properties{API: api}
	api.Properties.SmartEvents = &SmartEvents{api.Properties}
	api.Properties.Domains = &Domains{api.Properties}
	return api
}
