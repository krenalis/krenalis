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
	"errors"
	"io"
	"log"
	"maps"
	"math/rand/v2"
	"net/http"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name: "Dummy",
		AsSource: &meergo.AsAppSource{
			Description: "Import users from Dummy",
			Targets:     meergo.Users,
		},
		AsDestination: &meergo.AsAppDestination{
			Description: "Export users and send events to Dummy",
			Targets:     meergo.Events | meergo.Users,
			SendingMode: meergo.Combined,
			HasSettings: true,
		},
		TermForUsers:    "users",
		IdentityIDLabel: "Dummy Unique ID",
		Icon:            icon,
	}, New)
}

// New returns a new Dummy connector instance.
func New(conf *meergo.AppConfig) (*Dummy, error) {
	c := Dummy{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Dummy connector")
		}
	}
	return &c, nil
}

type Dummy struct {
	conf     *meergo.AppConfig
	settings *innerSettings
}

var (
	allUsers             map[string]map[string]any
	usersLastChangeTimes map[string]time.Time
	usersLock            sync.Mutex
)

//go:embed users.json
var jsonUsers []byte

func newDummyId() string {
	b := make([]rune, 12)
	for i := range b {
		b[i] = rune(rand.IntN(20) + 'a')
	}
	return "dummy_" + string(b)
}

// EventRequest returns a request to dispatch an event to the app.
func (dummy *Dummy) EventRequest(ctx context.Context, event meergo.Event, eventType string, schema types.Type, properties map[string]any, redacted bool) (*meergo.EventRequest, error) {
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
func (dummy *Dummy) Records(ctx context.Context, _ meergo.Targets, lastChangeTime time.Time, ids, _ []string, _ string, _ types.Type) ([]meergo.Record, string, error) {
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	default:
	}
	usersLock.Lock()
	defer usersLock.Unlock()
	users := make([]meergo.Record, 0, len(allUsers))
	for id, props := range allUsers {
		if usersLastChangeTimes[id].Before(lastChangeTime) {
			continue
		}
		if ids != nil && !slices.Contains(ids, id) {
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
		usersLastChangeTimes[u.ID] = time.Now().UTC().Truncate(time.Microsecond)
	}
	usersLock.Unlock()
}

type innerSettings struct {
	URLForDispatchingEvents string
}

// Schema returns the schema of the specified target in the specified role.
func (dummy *Dummy) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {
	if target == meergo.Users {
		var properties []types.Property
		if role == meergo.Source {
			properties = append(properties, types.Property{Name: "dummyId", Type: types.Text()})
		}
		properties = append(properties, []types.Property{
			{Name: "email", Type: types.Text(), Nullable: true},
			{Name: "firstName", Type: types.Text(), Nullable: true},
			{Name: "fullName", Type: types.Text(), Nullable: true},
			{Name: "lastName", Type: types.Text(), Nullable: true},
			{Name: "favouriteDrink", Type: types.Text().WithValues("tea", "beer", "wine", "water"), Nullable: true},
			{Name: "favourite_movie", Type: types.Text(), ReadOptional: true},
		}...)
		if role == meergo.Destination {
			properties = append(properties, types.Property{Name: "additionalProperties", Type: types.Map(types.Text())})
		}
		properties = append(properties, []types.Property{
			{Name: "address", Type: types.Object([]types.Property{
				{Name: "street", Type: types.Text(), Nullable: true},
				{Name: "postal_code", Type: types.Text(), Nullable: true},
				{Name: "city", Type: types.Text(), Nullable: true},
			}), Nullable: true},
		}...)
		return types.Object(properties), nil
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
func (dummy *Dummy) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if dummy.settings != nil {
			s = *dummy.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, dummy.saveSettings(ctx, settings)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "URLForDispatchingEvents", Label: "URL for dispatching events", Role: meergo.Destination, Placeholder: "https://example.com"},
		},
		Settings: settings,
	}

	return ui, nil
}

// nonRequiredProperties contains the names of the properties that are both in
// the source and destination schema and are not requires for create.
var nonRequiredProperties = []string{"email", "firstName", "lastName", "fullName", "favouriteDrink", "address"}

// Upsert updates or creates records in the app for the specified target.
func (dummy *Dummy) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {

	usersLock.Lock()
	defer usersLock.Unlock()

	for _, record := range records.All() {

		// Prepare the properties to log.
		properties, err := json.Marshal(record.Properties)
		if err != nil {
			return err
		}

		var id string
		if record.ID == "" {
			// Add a new users into the in-memory users.
			log.Printf("[info] Dummy: CreateUser(%v)", string(properties))
			user := maps.Clone(record.Properties)
			id = newDummyId()
			user["dummyId"] = id
			for _, p := range nonRequiredProperties {
				if v, ok := user[p]; !ok {
					user[p] = nil
				} else if p == "address" {
					address := v.(map[string]any)
					if _, ok := address["street"]; !ok {
						address["street"] = nil
					}
					if _, ok := address["postal_code"]; !ok {
						address["postal_code"] = nil
					}
					if _, ok := address["city"]; !ok {
						address["city"] = nil
					}
				}
			}
			allUsers[id] = user
		} else {
			// Update the in-memory users.
			user, ok := allUsers[record.ID]
			if !ok {
				log.Printf("[info] Dummy: UpdateUser(%q, %v): user not found", record.ID, string(properties))
				continue
			}
			log.Printf("[info] Dummy: UpdateUser(%q, %v)", record.ID, string(properties))
			maps.Copy(user, record.Properties)
			id = record.ID
		}

		usersLastChangeTimes[id] = time.Now().UTC().Truncate(time.Microsecond)

	}

	return nil
}

// saveSettings saves the settings.
func (dummy *Dummy) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
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
