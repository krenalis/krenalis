//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"chichi/connectors"
	"chichi/connectors/hubspot"
)

const token = "*****"
const clientSecret = "*****"

func init() {
	hubspot.Debug = true
}

func TestSync(t *testing.T) {
	c := connectors.Connector(context.Background(), "HubSpot", clientSecret)
	err := c.Users(token, "", []string{"email"})
	if err != nil {
		log.Fatal(err)
	}
	err = c.Groups(token, "", []string{"domain"})
	if err != nil {
		log.Fatal(err)
	}
}

func TestSetUsers(t *testing.T) {
	c := connectors.Connector(context.Background(), "HubSpot", clientSecret)
	user := connectors.User{
		ID:         "1",
		Properties: connectors.Properties{"email": "info@open2b.com"},
	}
	err := c.SetUsers(token, []connectors.User{user})
	if err != nil {
		log.Fatal(err)
	}
}

func TestProperties(t *testing.T) {
	c := connectors.Connector(context.Background(), "HubSpot", clientSecret)
	properties, err := c.Properties(token)
	if err != nil {
		log.Fatal(err)
	}
	v, _ := json.Marshal(properties)
	fmt.Printf("\n\nproperties:\n%s\n", v)
}
