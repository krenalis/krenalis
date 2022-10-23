//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package dummy

import (
	"context"
	"net/http"
	"time"

	"chichi/connectors"
)

// Make sure it implements the AppConnection interface.
var _ connectors.AppConnection = &connection{}

func init() {
	connectors.RegisterAppConnector("Dummy", New)
}

type connection struct {
	firehose     connectors.Firehose
	clientSecret string
}

// New returns a new Dummy connection.
func New(ctx context.Context, conf *connectors.AppConfig) (connectors.AppConnection, error) {
	c := connection{
		firehose:     conf.Firehose,
		clientSecret: conf.ClientSecret,
	}
	return &c, nil
}

func (c *connection) Groups(cursor string, properties [][]string) error {
	panic("not implemented")
}

func (c *connection) Properties() ([]connectors.Property, []connectors.Property, error) {
	userProps := []connectors.Property{
		{Name: "first_name", Type: "string"},
		{Name: "last_name", Type: "string"},
		{Name: "email", Type: "string"},
	}
	return userProps, nil, nil
}

func (c *connection) ReceiveWebhook(r *http.Request) ([]connectors.Event, error) {
	panic("not implemented")
}

func (c *connection) Resource() (string, error) {
	return "dummy-resource", nil
}

func (c *connection) ServeUserInterface(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
}

type user struct {
	ID         string
	Properties map[string]any
}

var now = time.Now()

var users = []user{
	{
		ID: "1",
		Properties: map[string]any{
			"first_name": "Mario",
			"last_name":  connectors.TimestampedValue{Value: "Rossi", Timestamp: now.Add(5 * time.Second)},
			"email":      connectors.TimestampedValue{Value: "mariorossi@example.com", Timestamp: now.Add(1 * time.Second)},
		},
	},
	{
		ID: "2",
		Properties: map[string]any{
			"first_name": "Luigi",
			"last_name":  connectors.TimestampedValue{Value: "Verdi", Timestamp: now.Add(7 * time.Second)},
			"email":      connectors.TimestampedValue{Value: "luigiverdi@example.com", Timestamp: now.Add(3 * time.Second)},
		},
	},
}

func (c *connection) SetUsers(users []connectors.User) error {
	panic("not implemented")
}

func (c *connection) Users(cursor string, properties [][]string) error {
	for _, user := range users {
		c.firehose.SetUser(user.ID, now, user.Properties)
	}
	return nil
}
