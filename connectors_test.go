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
	c := connectors.Connector("HubSpot", clientSecret)
	err := c.Users(context.Background(), token, "", []string{"email"})
	if err != nil {
		log.Fatal(err)
	}
	err = c.Groups(context.Background(), token, "", []string{"domain"})
	if err != nil {
		log.Fatal(err)
	}
}

func TestProperties(t *testing.T) {
	c := connectors.Connector("HubSpot", clientSecret)
	properties, err := c.Properties(context.Background(), token)
	if err != nil {
		log.Fatal(err)
	}
	v, _ := json.Marshal(properties)
	fmt.Printf("\n\nproperties:\n%s\n", v)
}
