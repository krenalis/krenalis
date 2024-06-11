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

// Make sure it implements the App, AppEvents, AppRecords, and UIHandler interfaces.
var _ interface {
	chichi.App
	chichi.AppEvents
	chichi.AppRecords
} = (*Dummy)(nil)

func init() {
	chichi.RegisterApp(chichi.AppInfo{
		Name:                   "Dummy",
		Targets:                chichi.Events | chichi.Users,
		SourceDescription:      "import users from Dummy",
		DestinationDescription: "export users and send events to Dummy",
		TermForUsers:           "users",
		IdentityIDLabel:        "Dummy Unique ID",
		Icon:                   icon,
		SendingMode:            chichi.Combined,
	}, New)
}

// New returns a new Dummy connector instance.
func New(conf *chichi.AppConfig) (*Dummy, error) {
	c := Dummy{conf: conf}
	return &c, nil
}

type Dummy struct {
	conf *chichi.AppConfig
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
func (dummy *Dummy) Create(ctx context.Context, target chichi.Targets, properties map[string]any) error {

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

// Records returns the records of the specified target.
func (dummy *Dummy) Records(ctx context.Context, target chichi.Targets, properties []string, cursor chichi.Cursor) ([]chichi.Record, string, error) {
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	default:
	}
	usersLock.Lock()
	defer usersLock.Unlock()
	users := make([]chichi.Record, 0, len(allUsers))
	for id, props := range allUsers {
		if usersLastChangeTimes[id].Before(cursor.LastChangeTime) {
			continue
		}
		users = append(users, chichi.Record{
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

type Settings struct{}

// Schema returns the schema of the specified target.
func (dummy *Dummy) Schema(ctx context.Context, target chichi.Targets, role chichi.Role, eventType string) (types.Type, error) {
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

// Update updates a record of the specified target.
func (dummy *Dummy) Update(ctx context.Context, target chichi.Targets, id string, properties map[string]any) error {

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
