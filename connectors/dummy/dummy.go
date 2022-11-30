//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package dummy

import (
	"context"
	"net/http"
	"time"

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"
)

// Make sure it implements the AppConnection interface.
var _ connector.AppConnection = &connection{}

func init() {
	connector.RegisterApp("Dummy", newConnection)
}

type connection struct {
	firehose     connector.Firehose
	clientSecret string
}

// newConnection returns a new Dummy connection.
func newConnection(ctx context.Context, conf *connector.AppConfig) (connector.AppConnection, error) {
	c := connection{
		firehose:     conf.Firehose,
		clientSecret: conf.ClientSecret,
	}
	return &c, nil
}

// Connector returns information about the connector.
func (c *connection) Connector() *connector.Connector {
	return &connector.Connector{
		Name: "Dummy",
		Type: connector.AppType,
	}
}

func (c *connection) Groups(cursor string, properties []connector.PropertyPath) error {
	panic("not implemented")
}

func (c *connection) ReceiveWebhook(r *http.Request) ([]connector.Event, error) {
	panic("not implemented")
}

func (c *connection) Resource() (string, error) {
	return "dummy-resource", nil
}

func (c *connection) Schemas() (types.Schema, types.Schema, error) {
	userSchema := types.MustSchemaOf([]types.Property{
		{Name: "first_name", Type: types.Text()},
		{Name: "last_name", Type: types.Text()},
		{Name: "email", Type: types.Text()},
	})
	return userSchema, types.Schema{}, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {
	return nil, nil, ui.ErrEventNotExist
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
			"last_name":  "Rossi",
			"email":      "mariorossi@example.com",
		},
	},
	{
		ID: "2",
		Properties: map[string]any{
			"first_name": "Luigi",
			"last_name":  "Verdi",
			"email":      "luigiverdi@example.com",
		},
	},
}

var timestamps = map[string]map[string]time.Time{
	"1": {
		"last_name": now.Add(5 * time.Second),
		"email":     now.Add(1 * time.Second),
	},
	"2": {
		"last_name": now.Add(7 * time.Second),
		"email":     now.Add(3 * time.Second),
	},
}

func (c *connection) SetUsers(users []connector.User) error {
	panic("not implemented")
}

func (c *connection) Users(cursor string, properties []connector.PropertyPath) error {
	for _, user := range users {
		c.firehose.SetUser(user.ID, user.Properties, now, timestamps[user.ID])
	}
	return nil
}
