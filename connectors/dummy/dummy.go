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
	"errors"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"
	"github.com/open2b/chichi/ui"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the AppEvents, AppUsers, and UI interfaces.
var _ interface {
	chichi.AppEvents
	chichi.AppUsers
	chichi.UI
} = (*Dummy)(nil)

func init() {
	chichi.RegisterApp(chichi.AppInfo{
		Name:                   "Dummy",
		SourceDescription:      "import users from Dummy",
		DestinationDescription: "export users and send events to Dummy",
		TermForUsers:           "users",
		ExternalIDLabel:        "Dummy Unique ID",
		Icon:                   icon,
		SendingMode:            chichi.Combined,
	}, New)
}

// New returns a new Dummy connector instance.
func New(conf *chichi.AppConfig) (*Dummy, error) {
	c := Dummy{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of CSV connector")
		}
	}
	return &c, nil
}

type Dummy struct {
	conf     *chichi.AppConfig
	settings *settings
}

var (
	allUsers        map[string]map[string]any
	usersTimestamps map[string]time.Time
	usersLock       sync.Mutex
)

//go:embed users.json
var jsonUsers []byte

var randGenerator = rand.New(rand.NewSource(time.Now().Unix()))

func newUserID() string {
	b := make([]rune, 12)
	for i := range b {
		b[i] = rune(randGenerator.Intn(20) + 'a')
	}
	return "dummy_" + string(b)
}

// CreateUser creates a user with the given properties.
func (dummy *Dummy) CreateUser(ctx context.Context, user map[string]any) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Write the user on the log.
	propsDump, err := json.Marshal(user)
	if err != nil {
		return err
	}
	log.Printf("[info] Dummy: CreateUser(%v)", string(propsDump))

	// Update the in-memory users.
	usersLock.Lock()
	defer usersLock.Unlock()
	u := map[string]any{}
	id := newUserID()
	u["dummyId"] = id
	for name, value := range user {
		u[name] = value
	}
	allUsers[id] = u
	usersTimestamps[id] = time.Now().UTC()

	return nil
}

// EventRequest returns an event request associated with the provided event
// type, event, and transformation data. If redacted is true, sensitive
// authentication data will be redacted in the returned request.
// This method is safe for concurrent use by multiple goroutines.
// If the specified event type does not exist, it returns the
// ErrEventTypeNotExist error.
func (dummy *Dummy) EventRequest(ctx context.Context, eventType *chichi.EventType, event *chichi.Event, data map[string]any, redacted bool) (*chichi.EventRequest, error) {
	req := &chichi.EventRequest{
		Method: "POST",
		URL:    "https://example.com/",
		Header: http.Header{},
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	var err error
	req.Body, err = json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// EventTypes returns the event types of the connector's instance.
func (dummy *Dummy) EventTypes(ctx context.Context) ([]*chichi.EventType, error) {
	if dummy.conf.Role == chichi.Source {
		return nil, nil
	}
	eventTypes := []*chichi.EventType{
		{
			ID:          "send_add_to_cart",
			Name:        "Send Add to Cart",
			Description: "Send an Add to Cart event to Dummy",
			Schema: types.Object([]types.Property{
				{Name: "email", Type: types.Text()},
				{Name: "itemName", Type: types.Text()},
				{Name: "itemId", Type: types.Int(32)},
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

// ReceiveWebhook receives a webhook request and returns its payloads. It
// returns the ErrWebhookUnauthorized error is the request was not authorized.
// The context is the request's context.
func (dummy *Dummy) ReceiveWebhook(r *http.Request) ([]chichi.WebhookPayload, error) {
	panic("not implemented")
}

func (dummy *Dummy) Resource(ctx context.Context) (string, error) {
	return "", nil
}

type settings struct {
	LargeDataset bool
}

// ServeUI serves the connector's user interface.
func (dummy *Dummy) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if dummy.settings != nil {
			s = *dummy.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := dummy.ValidateSettings(ctx, values)
		if err != nil {
			return nil, nil, err
		}
		err = dummy.conf.SetSettings(ctx, s)
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

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (dummy *Dummy) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var settings settings
	err := json.Unmarshal(values, &settings)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&settings)
}

// UpdateUser updates the user with identifier id setting the given properties.
func (dummy *Dummy) UpdateUser(ctx context.Context, id string, user map[string]any) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Write the user on the log.
	propsDump, err := json.Marshal(user)
	if err != nil {
		return err
	}
	log.Printf("[info] Dummy: UpdateUser(%q, %v)", id, string(propsDump))

	// Update the in-memory users.
	usersLock.Lock()
	defer usersLock.Unlock()
	u, ok := allUsers[id]
	if !ok {
		u = map[string]any{}
	}
	u["dummyId"] = id
	for name, value := range user {
		u[name] = value
	}
	allUsers[id] = u
	usersTimestamps[id] = time.Now().UTC()

	return nil
}

var userSchema = types.Object([]types.Property{
	{Name: "dummyId", Type: types.Text(), Role: types.SourceRole},
	{Name: "email", Type: types.Text()},
	{Name: "firstName", Type: types.Text()},
	{Name: "fullName", Type: types.Text()},
	{Name: "lastName", Type: types.Text()},
	{Name: "favouriteDrink", Type: types.Text().WithValues("tea", "beer", "wine", "water")},
})

// UserSchema returns the user schema.
func (dummy *Dummy) UserSchema(ctx context.Context) (types.Type, error) {
	return userSchema, nil
}

// Users returns the users starting from the given cursor.
func (dummy *Dummy) Users(ctx context.Context, properties []string, cursor chichi.Cursor) ([]chichi.Record, string, error) {
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	default:
	}
	usersLock.Lock()
	defer usersLock.Unlock()
	users := make([]chichi.Record, 0, len(allUsers))
	for id, props := range allUsers {
		users = append(users, chichi.Record{
			ID:         id,
			Properties: props,
			Timestamp:  usersTimestamps[id],
		})
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	if !dummy.settings.LargeDataset {
		users = users[:10]
	}
	return users, "", io.EOF
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
	allUsers = make(map[string]map[string]any, len(rawUsers))
	usersTimestamps = make(map[string]time.Time, len(rawUsers))
	for _, u := range rawUsers {
		u.Properties["dummyId"] = u.ID
		allUsers[u.ID] = u.Properties
		usersTimestamps[u.ID] = time.Now().UTC()
	}
	usersLock.Unlock()
}
