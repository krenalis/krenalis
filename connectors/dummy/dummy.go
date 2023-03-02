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
)

// exportOnly10Users, when true, makes Dummy export only 10 users instead of the
// entire data set.
const exportOnly10Users = true

// Make sure it implements the AppConnection interface.
var _ connector.AppConnection = &connection{}

func init() {
	connector.RegisterApp(connector.App{
		Name:      "Dummy",
		Endpoints: map[int]string{1: "Europe", 2: "America"},
		Open:      open,
	})
}

type connection struct {
	firehose connector.Firehose
}

// open opens a Dummy connection and returns it.
func open(ctx context.Context, conf *connector.AppConfig) (connector.AppConnection, error) {
	c := connection{
		firehose: conf.Firehose,
	}
	return &c, nil
}

// ActionTypes returns the connection's action types.
func (c *connection) ActionTypes() ([]*connector.ActionType, error) {
	actionTypes := []*connector.ActionType{
		{
			ID:          1,
			Name:        "Send Add to Cart",
			Description: "Send an Add to Cart event to Dummy",
			Endpoints:   []int{1, 2},
			Schema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "item_name", Type: types.Text()},
				{Name: "item_id", Type: types.Int()},
			}),
		},
		{
			ID:          2,
			Name:        "Send custom event",
			Description: "Send a custom event to Dummy",
			Endpoints:   []int{2},
			Schema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
			}),
		},
		{
			ID:          3,
			Name:        "Send Identity",
			Description: "Send an Identity to Dummy",
			Endpoints:   []int{1, 2},
			Schema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "traits", Type: types.Object([]types.Property{
					{Name: "address", Type: types.Object([]types.Property{
						{Name: "street1", Type: types.Text()},
						{Name: "street2", Type: types.Text()},
					})},
				})},
			}),
		},
	}
	return actionTypes, nil
}

func (c *connection) Groups(cursor string, properties []connector.PropertyPath) error {
	panic("not implemented")
}

func (c *connection) ReceiveWebhook(r *http.Request) ([]connector.Event, error) {
	panic("not implemented")
}

func (c *connection) Resource() (string, error) {
	return "", nil
}

func (c *connection) Schemas() (types.Type, types.Type, error) {
	userSchema := types.Object([]types.Property{
		{Name: "email", Type: types.Text()},
		{Name: "first_name", Type: types.Text()},
		{Name: "full_name", Type: types.Text()},
		{Name: "last_name", Type: types.Text()},
		{Name: "favourite_drink", Type: types.Text().WithEnum([]string{"tea", "beer", "wine", "water"})},
	})
	return userSchema, types.Type{}, nil
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
