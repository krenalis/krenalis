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

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the App, AppEvents, AppRecords, and UIHandler interfaces.
var _ interface {
	meergo.App
	meergo.AppEvents
	meergo.UIHandler
	meergo.AppRecords
} = (*Dummy)(nil)

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name:                   "Dummy",
		Targets:                meergo.Events | meergo.Users,
		SourceDescription:      "import users from Dummy",
		DestinationDescription: "export users and send events to Dummy",
		TermForUsers:           "users",
		IdentityIDLabel:        "Dummy Unique ID",
		Icon:                   icon,
		SendingMode:            meergo.Combined,
	}, New)
}

// New returns a new Dummy connector instance.
func New(conf *meergo.AppConfig) (*Dummy, error) {
	c := Dummy{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Dummy connector")
		}
	}
	return &c, nil
}

type Dummy struct {
	conf     *meergo.AppConfig
	settings *Settings
}

var (
	allUsers             map[string]map[string]any
	usersLastChangeTimes map[string]time.Time
	usersLock            sync.Mutex
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
func (dummy *Dummy) Create(ctx context.Context, target meergo.Targets, properties map[string]any) error {

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
	u["dummyId"] = id
	for name, value := range properties {
		u[name] = value
	}
	allUsers[id] = u
	usersLastChangeTimes[id] = time.Now().UTC()

	return nil
}

// EventRequest returns a request to dispatch an event to the app.
func (dummy *Dummy) EventRequest(ctx context.Context, event *meergo.Event, eventType string, schema types.Type, properties map[string]any, redacted bool) (*meergo.EventRequest, error) {
	url := "https://example.com/"
	if dummy.settings.URLForDispatchingEvents != "" {
		url = dummy.settings.URLForDispatchingEvents
	}
	req := &meergo.EventRequest{
		Method: "POST",
		URL:    url,
		Header: http.Header{},
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if properties != nil {
		var err error
		req.Body, err = json.Marshal(properties)
		if err != nil {
			return nil, err
		}
	}
	return req, nil
}

// EventTypes returns the event types of the connector's instance.
func (dummy *Dummy) EventTypes(ctx context.Context) ([]*meergo.EventType, error) {
	return []*meergo.EventType{
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

// Records returns the records of the specified target.
func (dummy *Dummy) Records(ctx context.Context, target meergo.Targets, properties []string, cursor meergo.Cursor) ([]meergo.Record, string, error) {
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	default:
	}
	usersLock.Lock()
	defer usersLock.Unlock()
	users := make([]meergo.Record, 0, len(allUsers))
	for id, props := range allUsers {
		if usersLastChangeTimes[id].Before(cursor.LastChangeTime) {
			continue
		}
		users = append(users, meergo.Record{
			ID:             id,
			Properties:     props,
			LastChangeTime: usersLastChangeTimes[id],
		})
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
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
	// Only the first 10 users are taken. The others, with the current
	// implementation of Dummy, remain defined in the JSON file but are not
	// used.
	rawUsers = rawUsers[:10]
	usersLock.Lock()
	allUsers = make(map[string]map[string]any, len(rawUsers))
	usersLastChangeTimes = make(map[string]time.Time, len(rawUsers))
	for _, u := range rawUsers {
		u.Properties["dummyId"] = u.ID
		allUsers[u.ID] = u.Properties
		usersLastChangeTimes[u.ID] = time.Now().UTC()
	}
	usersLock.Unlock()
}

type Settings struct {
	URLForDispatchingEvents string
}

// Schema returns the schema of the specified target.
func (dummy *Dummy) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {
	if target == meergo.Users {
		return types.Object([]types.Property{
			{Name: "dummyId", Type: types.Text(), Role: types.SourceRole},
			{Name: "email", Type: types.Text()}, // TODO(Gianluca): removed 'CreateRequired' until UI is updated in https://github.com/meergo/meergo/issues/934.
			{Name: "firstName", Type: types.Text()},
			{Name: "fullName", Type: types.Text()},
			{Name: "lastName", Type: types.Text()},
			{Name: "favouriteDrink", Type: types.Text().WithValues("tea", "beer", "wine", "water")},
			{Name: "favourite_movie", Type: types.Text(), ReadOptional: true},
			{Name: "additionalProperties", Type: types.Map(types.Text()), Role: types.DestinationRole},
		}), nil
	}
	switch eventType {
	case "send_add_to_cart":
		return types.Object([]types.Property{
			{Name: "email", Type: types.Text(), CreateRequired: true},
			{Name: "itemName", Type: types.Text()},
			{Name: "itemId", Type: types.Int(32)},
		}), nil
	case "send_custom_event":
		return types.Object([]types.Property{
			{Name: "email", Type: types.Text()},
		}), nil
	case "send_identity":
		return types.Object([]types.Property{
			{Name: "email", CreateRequired: true, Type: types.Text()},
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
	return types.Type{}, meergo.ErrEventTypeNotExist
}

// ServeUI serves the connector's user interface.
func (dummy *Dummy) ServeUI(ctx context.Context, event string, values []byte, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s Settings
		if dummy.settings != nil {
			s = *dummy.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, dummy.saveValues(ctx, values)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "URLForDispatchingEvents", Label: "URL for dispatching events", Role: meergo.Destination, Placeholder: "https://example.com"},
		},
		Values: values,
	}

	return ui, nil
}

// Update updates a record of the specified target.
func (dummy *Dummy) Update(ctx context.Context, target meergo.Targets, id string, properties map[string]any) error {

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
	u, ok := allUsers[id]
	if !ok {
		u = map[string]any{}
	}
	u["dummyId"] = id
	for name, value := range properties {
		u[name] = value
	}
	allUsers[id] = u
	usersLastChangeTimes[id] = time.Now().UTC()

	return nil
}

// saveValues validates the user-entered values and returns the settings.
func (dummy *Dummy) saveValues(ctx context.Context, values []byte) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = dummy.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	dummy.settings = &s
	return nil
}
