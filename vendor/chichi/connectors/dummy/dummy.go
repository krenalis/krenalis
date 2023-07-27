//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package dummy implements the Dummy connector.
package dummy

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"chichi/apis/normalization"
	"chichi/connector"
	"chichi/connector/types"
)

// Connector icon.
var icon = "<svg></svg>"

var (
	users           map[string]connector.Properties
	usersTimestamps map[string]time.Time
	usersLock       sync.Mutex
)

// loadOnly10Users, when true, makes Dummy to load only 10 users instead of the
// entire data set.
const loadOnly10Users = true

//go:embed users.json
var jsonUsers []byte

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

// open opens a Dummy connection.
func open(ctx context.Context, conf *connector.AppConfig) (*connection, error) {
	c := connection{conf: conf}
	return &c, nil
}

type connection struct {
	conf *connector.AppConfig
}

var randGenerator = rand.New(rand.NewSource(time.Now().Unix()))

func newUserID() string {
	b := make([]rune, 12)
	for i := range b {
		b[i] = rune(randGenerator.Intn(20) + 'a')
	}
	return "dummy_" + string(b)
}

// CreateUser creates a user with the given properties.
func (c *connection) CreateUser(properties connector.Properties) error {

	// Normalize and validate the properties.
	properties, err := normalize(properties, userSchema)
	if err != nil {
		return err
	}

	// Write the user on the log.
	propsDump, err := json.Marshal(properties)
	if err != nil {
		return err
	}
	log.Printf("[info] Dummy: CreateUser(%v)", string(propsDump))

	// Update the in-memory users.
	usersLock.Lock()
	defer usersLock.Unlock()
	u := connector.Properties{}
	id := newUserID()
	u["dummy_id"] = id
	for name, value := range properties {
		u[name] = value
	}
	users[id] = u
	usersTimestamps[id] = time.Now().UTC()

	return nil
}

// EventTypes returns the connection's event types.
func (c *connection) EventTypes() ([]*connector.EventType, error) {
	if c.conf.Role == connector.SourceRole {
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
		{
			ID:          "send_event_with_no_schema",
			Name:        "Send event with no schema",
			Description: "Send an event which does not require mapping",
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

// UpdateUser updates the user with identifier id setting the given properties.
func (c *connection) UpdateUser(id string, properties connector.Properties) error {

	// Normalize and validate the properties.
	properties, err := normalize(properties, userSchema)
	if err != nil {
		return err
	}

	// Write the user on the log.
	propsDump, err := json.Marshal(properties)
	if err != nil {
		return err
	}
	log.Printf("[info] Dummy: UpdateUser(%q, %v)", id, string(propsDump))

	// Update the in-memory users.
	usersLock.Lock()
	defer usersLock.Unlock()
	u, ok := users[id]
	if !ok {
		u = connector.Properties{}
	}
	u["dummy_id"] = id
	for name, value := range properties {
		u[name] = value
	}
	users[id] = u
	usersTimestamps[id] = time.Now().UTC()

	return nil
}

var userSchema = types.Object([]types.Property{
	{Name: "dummy_id", Type: types.Text(), Role: types.SourceRole},
	{Name: "email", Type: types.Text()},
	{Name: "first_name", Type: types.Text()},
	{Name: "full_name", Type: types.Text()},
	{Name: "last_name", Type: types.Text()},
	{Name: "favourite_drink", Type: types.Text().WithEnum([]string{"tea", "beer", "wine", "water"})},
})

// UserSchema returns the user schema.
func (c *connection) UserSchema() (types.Type, error) {
	return userSchema, nil
}

// Users returns the users starting from the given cursor.
func (c *connection) Users(properties []types.Path, cursor connector.Cursor) ([]connector.Object, string, error) {
	usersLock.Lock()
	defer usersLock.Unlock()
	objects := make([]connector.Object, 0, len(users))
	for id, props := range users {
		objects = append(objects, connector.Object{
			ID:         id,
			Properties: props,
			Timestamp:  usersTimestamps[id],
		})
	}
	sort.Slice(objects, func(i, j int) bool { return objects[i].ID < objects[j].ID })
	return objects, "", io.EOF
}

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
	users = make(map[string]connector.Properties, len(rawUsers))
	usersTimestamps = make(map[string]time.Time, len(rawUsers))
	for _, u := range rawUsers {
		u.Properties["dummy_id"] = u.ID
		users[u.ID] = u.Properties
		usersTimestamps[u.ID] = time.Now().UTC()
	}
	usersLock.Unlock()
}

func normalize(values map[string]any, schema types.Type) (map[string]any, error) {
	out := make(map[string]any, len(values))
	for name, value := range values {
		prop, ok := schema.Property(name)
		if !ok {
			return nil, fmt.Errorf("property %q not found", name)
		}
		v, err := normalization.NormalizeAppProperty(name, prop.Type, value, prop.Nullable)
		if err != nil {
			return nil, err
		}
		out[name] = v
	}
	return out, nil
}
