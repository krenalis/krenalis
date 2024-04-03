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
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the App, AppEvents, AppRecords, and UI interfaces.
var _ interface {
	chichi.App
	chichi.AppEvents
	chichi.AppRecords
	chichi.UI
} = (*Dummy)(nil)

func init() {
	chichi.RegisterApp(chichi.AppInfo{
		Name:                   "Dummy",
		Targets:                chichi.Events | chichi.Users,
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

// Create creates a record for the specified target with the given properties.
func (dummy *Dummy) Create(ctx context.Context, _ chichi.Targets, user map[string]any) error {

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

// EventRequest returns a request to dispatch an event to the app. typ specifies
// the type of event to send, event is the received event, extra contains the
// extra information, schema is the schema of the extra information (that is the
// schema for the event type), and redacted indicates whether authentication
// data must be redacted in the returned request.
//
// schema is the invalid schema if extra is nil and vice versa.
//
// This method is safe for concurrent use by multiple goroutines. If the
// specified event type does not exist, it returns the ErrEventTypeNotExist
// error.
func (dummy *Dummy) EventRequest(ctx context.Context, typ string, event *chichi.Event, extra map[string]any, schema types.Type, redacted bool) (*chichi.EventRequest, error) {
	req := &chichi.EventRequest{
		Method: "POST",
		URL:    "https://example.com/",
		Header: http.Header{},
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	var err error
	req.Body, err = json.Marshal(extra)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// EventTypes returns the event types of the connector's instance.
func (dummy *Dummy) EventTypes(ctx context.Context) ([]*chichi.EventType, error) {
	return []*chichi.EventType{
		{
			ID:          "send_add_to_cart",
			Name:        "Send Add to Cart",
			Description: "Send an Add to Cart event to Dummy",
		},
		{
			ID:          "send_custom_event",
			Name:        "Send custom event",
			Description: "Send a custom event to Dummy",
		},
		{
			ID:          "send_identity",
			Name:        "Send Identity",
			Description: "Send an Identity to Dummy",
		},
		{
			ID:          "send_generic_event",
			Name:        "Send generic event",
			Description: "Send a generic event, useful for testing",
		},
		{
			ID:          "send_event_with_no_schema",
			Name:        "Send event with no schema",
			Description: "Send an event which does not require mapping",
		},
	}, nil
}

// Records returns the records of the specified target, starting from the given
// cursor.
func (dummy *Dummy) Records(ctx context.Context, _ chichi.Targets, properties []string, cursor chichi.Cursor) ([]chichi.Record, string, error) {
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

func (dummy *Dummy) Resource(ctx context.Context) (string, error) {
	return "", nil
}

type settings struct {
	LargeDataset bool
}

// Schema returns the schema of the specified target. For Users or Groups, it
// returns their respective schemas. For Events, it returns the schema of the
// specified event type.
func (dummy *Dummy) Schema(ctx context.Context, target chichi.Targets, eventType string) (types.Type, error) {
	if target == chichi.Users {
		return types.Object([]types.Property{
			{Name: "dummyId", Type: types.Text(), Role: types.SourceRole},
			{Name: "email", Type: types.Text()},
			{Name: "firstName", Type: types.Text()},
			{Name: "fullName", Type: types.Text()},
			{Name: "lastName", Type: types.Text()},
			{Name: "favouriteDrink", Type: types.Text().WithValues("tea", "beer", "wine", "water")},
		}), nil
	}
	switch eventType {
	case "send_add_to_cart":
		return types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
			{Name: "itemName", Type: types.Text()},
			{Name: "itemId", Type: types.Int(32)},
		}), nil
	case "send_custom_event":
		return types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
		}), nil
	case "send_identity":
		return types.Object([]types.Property{
			{Name: "email", Required: true, Type: types.Text()},
			{Name: "traits", Type: types.Object([]types.Property{
				{Name: "address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text()},
					{Name: "street2", Type: types.Text()},
				})},
			})},
		}), nil
	case "send_generic_event":
		return types.Object([]types.Property{
			{Name: "properties", Type: types.JSON()},
		}), nil
	case "send_event_with_no_schema":
		return types.Type{}, nil
	}
	return types.Type{}, chichi.ErrEventTypeNotExist
}

// ServeUI serves the connector's user interface.
func (dummy *Dummy) ServeUI(ctx context.Context, event string, values []byte) (*chichi.Form, *chichi.Alert, error) {

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
		return nil, chichi.SuccessAlert("Settings for Dummy saved"), nil
	default:
		return nil, nil, chichi.ErrEventNotExist
	}

	form := &chichi.Form{
		Fields: []chichi.Component{
			&chichi.Checkbox{Name: "LargeDataset", Label: "Make available the large users dataset (1000 users) instead of just 10", Role: chichi.Source},
		},
		Values:  values,
		Actions: []chichi.Action{{Event: "save", Text: "Save", Variant: "primary"}},
	}

	return form, nil, nil
}

// Update updates the record of the specified target with the identifier id,
// setting the given properties.
func (dummy *Dummy) Update(ctx context.Context, _ chichi.Targets, id string, user map[string]any) error {

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
