//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package dummy

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"
	"time"

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"
)

// exportOnly10Users, when true, makes Dummy export only 10 users instead of the
// entire data set.
const exportOnly10Users = true

// Make sure it implements the AppConnection interface.
var _ connector.AppConnection = &connection{}

func init() {
	connector.RegisterApp(connector.App{
		Name: "Dummy",
		Open: open,
	})
}

type connection struct {
	firehose     connector.Firehose
	clientSecret string
}

// open opens a Dummy connection and returns it.
func open(ctx context.Context, conf *connector.AppConfig) (connector.AppConnection, error) {
	c := connection{
		firehose:     conf.Firehose,
		clientSecret: conf.ClientSecret,
	}
	return &c, nil
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

func (c *connection) Schemas() (types.Type, types.Type, error) {
	userSchema := types.Object([]types.Property{
		{Name: "email", Type: types.Text()},
		{Name: "first_name", Type: types.Text()},
		{Name: "full_name", Type: types.Text()},
		{Name: "last_name", Type: types.Text()},
	})
	return userSchema, types.Type{}, nil
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

func (c *connection) SetUsers(users []connector.User) error {
	panic("not implemented")
}

//go:embed users.json
var jsonUsers []byte

func (c *connection) Users(cursor string, properties []connector.PropertyPath) error {
	var users []user
	err := json.Unmarshal(jsonUsers, &users)
	if err != nil {
		panic(err)
	}
	if exportOnly10Users {
		users = users[:10]
	}
	for _, user := range users {
		c.firehose.SetUser(user.ID, user.Properties, now, nil)
	}
	return nil
}
