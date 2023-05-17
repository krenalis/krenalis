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
	"log"
	"net/http"
	"sync"
	"time"

	"chichi/connector"
	"chichi/connector/types"
)

// Connector icon.
var icon = "<svg></svg>"

var (
	users     map[string]properties
	usersLock sync.Mutex
)

type properties map[string]any

// loadOnly10Users, when true, makes Dummy to load only 10 users instead of the
// entire data set.
const loadOnly10Users = true

//go:embed users.json
var jsonUsers []byte

func init() {
	var rawUsers []struct {
		ID         string
		Properties map[string]any
	}
	err := json.Unmarshal(jsonUsers, &rawUsers)
	if err != nil {
		panic(err)
	}
	if loadOnly10Users {
		rawUsers = rawUsers[:10]
	}
	usersLock.Lock()
	users = make(map[string]properties, len(rawUsers))
	for _, u := range rawUsers {
		u.Properties["dummy_id"] = u.ID
		users[u.ID] = u.Properties
	}
	usersLock.Unlock()
}

// Make sure it implements the AppEventsConnection and the AppUsersConnection
// interfaces.
var _ interface {
	connector.AppEventsConnection
	connector.AppUsersConnection
} = (*connection)(nil)

func init() {
	connector.RegisterApp(connector.App{
		Name:                   "Dummy",
		SourceDescription:      "import users from Dummy",
		DestinationDescription: "export users and send events to Dummy",
		TermForUsers:           "users",
		Icon:                   icon,
	}, open)
}

type connection struct {
	role     connector.Role
	firehose connector.Firehose
}

// open opens a Dummy connection.
func open(ctx context.Context, conf *connector.AppConfig) (*connection, error) {
	c := connection{role: conf.Role, firehose: conf.Firehose}
	return &c, nil
}

// EventTypes returns the connection's event types.
func (c *connection) EventTypes() ([]*connector.EventType, error) {
	if c.role == connector.SourceRole {
		return nil, nil
	}
	eventTypes := []*connector.EventType{
		{
			ID:          "send_add_to_cart",
			Name:        "Send Add to Cart",
			Description: "Send an Add to Cart event to Dummy",
			Schema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "item_name", Type: types.Text()},
				{Name: "item_id", Type: types.Int()},
			}),
		},
		{
			ID:          "send_custom_event",
			Name:        "Send custom event",
			Description: "Send a custom event to Dummy",
			Schema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
			}),
		},
		{
			ID:          "send_identity",
			Name:        "Send Identity",
			Description: "Send an Identity to Dummy",
			Schema: types.Object([]types.Property{
				{Name: "email", Required: true, Type: types.Text()},
				{Name: "traits", Type: types.Object([]types.Property{
					{Name: "address", Type: types.Object([]types.Property{
						{Name: "street1", Type: types.Text()},
						{Name: "street2", Type: types.Text()},
					})},
				})},
			}),
		},
		{
			ID:          "send_generic_event",
			Name:        "Send generic event",
			Description: "Send a generic event, useful for testing",
			Schema: types.Object([]types.Property{
				{Name: "properties", Type: types.JSON()},
			}),
		},
	}
	return eventTypes, nil
}

func (c *connection) ReceiveWebhook(r *http.Request) ([]connector.WebhookEvent, error) {
	panic("not implemented")
}

func (c *connection) Resource() (string, error) {
	return "", nil
}

// SendEvent sends the event, along with the given mapped event.
// eventType specifies the event type corresponding to the event.
func (c *connection) SendEvent(event connector.Event, mappedEvent map[string]any, eventType string) error {
	log.Printf("dummy: sending event %#v, %#v", event, mappedEvent)
	time.Sleep(50 * time.Millisecond)
	return nil
}

func (c *connection) SetUser(user connector.User) error {
	panic("not implemented")
}

// UserSchema returns the user schema.
func (c *connection) UserSchema() (types.Type, error) {
	schema := types.Object([]types.Property{
		{Name: "dummy_id", Type: types.Text()},
		{Name: "email", Type: types.Text()},
		{Name: "first_name", Type: types.Text()},
		{Name: "full_name", Type: types.Text()},
		{Name: "last_name", Type: types.Text()},
		{Name: "favourite_drink", Type: types.Text().WithEnum([]string{"tea", "beer", "wine", "water"})},
	})
	return schema, nil
}

var now = time.Now()

func (c *connection) Users(cursor string, properties []connector.PropertyPath) error {
	usersLock.Lock()
	defer usersLock.Unlock()
	for id, props := range users {
		c.firehose.SetUser(id, props, now, nil)
	}
	return nil
}
