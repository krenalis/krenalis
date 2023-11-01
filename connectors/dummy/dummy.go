//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package dummy implements the Dummy connector.
package dummy

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
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
	"chichi/connector/ui"
)

// Connector icon.
var icon = "<svg></svg>"

var (
	users           map[string]map[string]any
	usersTimestamps map[string]time.Time
	usersLock       sync.Mutex
)

//go:embed users.json
var jsonUsers []byte

// Make sure it implements the AppEventsConnection and the AppUsersConnection
// interfaces.
var _ interface {
	connector.AppEventsConnection
	connector.AppUsersConnection
	connector.UI
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
func open(conf *connector.AppConfig) (*connection, error) {
	c := connection{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of CSV connection")
		}
	}
	return &c, nil
}

type connection struct {
	conf     *connector.AppConfig
	settings *settings
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
func (c *connection) CreateUser(ctx context.Context, properties map[string]any) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
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
	u := map[string]any{}
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
func (c *connection) EventTypes(ctx context.Context) ([]*connector.EventType, error) {
	if c.conf.Role == connector.Source {
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

// PreviewSendEvent returns a preview of the event that would be sent when
// calling SendEvent with the same arguments.
func (c *connection) PreviewSendEvent(ctx context.Context, eventType string, event *connector.Event, data map[string]any) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("POST https://example.com/api\n")
	b.WriteString("Accept: application/json\n")
	b.WriteString("Content-Type: application/json\n\n")
	body, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return nil, err
	}
	b.Write(body)
	return b.Bytes(), nil
}

// ReceiveWebhook receives a webhook request and returns its payloads.
// It returns the ErrWebhookUnauthorized error is the request was not
// authorized. The context is the request's context.
func (c *connection) ReceiveWebhook(r *http.Request) ([]connector.WebhookPayload, error) {
	panic("not implemented")
}

func (c *connection) Resource(ctx context.Context) (string, error) {
	return "", nil
}

type settings struct {
	LargeDataset bool
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if c.settings != nil {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := c.ValidateSettings(ctx, values)
		if err != nil {
			return nil, nil, err
		}
		err = c.conf.SetSettings(ctx, s)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings for Dummy saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Checkbox{Name: "LargeDataset", Label: "Make available the large users dataset (1000 users) instead of just 10", Role: ui.Source},
		},
		Values:  values,
		Actions: []ui.Action{{Event: "save", Text: "Save", Variant: "primary"}},
	}

	return form, nil, nil
}

// SendEvent sends the event, along with the given mapped data.
// eventType specifies the event type corresponding to the event.
func (c *connection) SendEvent(ctx context.Context, eventType string, event *connector.Event, data map[string]any) error {
	log.Printf("dummy: sending event %#v, %#v", event, data)
	time.Sleep(50 * time.Millisecond)
	return nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var settings settings
	err := json.Unmarshal(values, &settings)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&settings)
}

// UpdateUser updates the user with identifier id setting the given properties.
func (c *connection) UpdateUser(ctx context.Context, id string, properties map[string]any) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
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
		u = map[string]any{}
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
	{Name: "favourite_drink", Type: types.Text().WithValues("tea", "beer", "wine", "water")},
})

// UserSchema returns the user schema.
func (c *connection) UserSchema(ctx context.Context) (types.Type, error) {
	return userSchema, nil
}

// Users returns the users starting from the given cursor.
func (c *connection) Users(ctx context.Context, properties []string, cursor connector.Cursor) ([]connector.User, string, error) {
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	default:
	}
	usersLock.Lock()
	defer usersLock.Unlock()
	objects := make([]connector.User, 0, len(users))
	for id, props := range users {
		objects = append(objects, connector.User{
			ID:         id,
			Properties: props,
			Timestamp:  usersTimestamps[id],
		})
	}
	sort.Slice(objects, func(i, j int) bool { return objects[i].ID < objects[j].ID })
	if !c.settings.LargeDataset {
		objects = objects[:10]
	}
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
	usersLock.Lock()
	users = make(map[string]map[string]any, len(rawUsers))
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
