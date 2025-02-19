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
	"maps"
	"math"
	"math/rand/v2"
	"net/http"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/metrics"
	"github.com/meergo/meergo/types"
)

// Constants for simulating the HTTP delay.
// The enabling of the delay is controlled by a connector setting.
const (
	httpDelayStdDev = 1.1
	httpDelayMean   = 0.02
	httpDelayMin    = 0.05 // seconds
	httpDelayMax    = 10   // seconds
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name: "Dummy",
		AsSource: &meergo.AsAppSource{
			Description: "Import users from Dummy",
			Targets:     meergo.Users,
			HasSettings: true,
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
	dummy.simulateHTTPDelay()
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
	metrics.Increment("Dummy.Records.calls", 1)
	dummy.simulateHTTPDelay()
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

	now := time.Now().UTC()

	usersLock.Lock()
	allUsers = make(map[string]map[string]any, len(rawUsers))
	usersLastChangeTimes = make(map[string]time.Time, len(rawUsers))
	for _, u := range rawUsers {
		u.Properties["dummyId"] = u.ID
		allUsers[u.ID] = u.Properties
		// Go back in time by a maximum of 100 milliseconds. This allows
		// timestamps to be spread over a time frame large enough to maintain
		// some randomness, but not so large that a timestamp is in the past
		// since the last import.
		nanosecDelta := rand.IntN(100e6)
		usersLastChangeTimes[u.ID] = now.Add(-time.Duration(nanosecDelta)).Truncate(time.Microsecond)
	}
	usersLock.Unlock()

}

type innerSettings struct {
	UserExportFailPercentage int // in [0, 100]
	URLForDispatchingEvents  string
	SimulateHTTPDelay        bool
}

// Schema returns the schema of the specified target in the specified role.
func (dummy *Dummy) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {
	dummy.simulateHTTPDelay()
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
			&meergo.Input{
				Name:            "UserExportFailPercentage",
				Type:            "number",
				Label:           "Percentage that the export of every single user may fail",
				Placeholder:     "10",
				HelpText:        "0 does not fail any user exports. 100 fails them all.",
				OnlyIntegerPart: true,
				Role:            meergo.Destination,
			},
			&meergo.Input{
				Name:        "URLForDispatchingEvents",
				Label:       "URL for dispatching events",
				Placeholder: "https://example.com",
				Role:        meergo.Destination,
			},
			&meergo.Checkbox{
				Name:  "SimulateHTTPDelay",
				Label: "Pretend that Dummy operates via HTTP calls, introducing fictitious delays",
				Role:  meergo.Both,
			},
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

	dummy.simulateHTTPDelay()

	recordsError := make(meergo.RecordsError, 0)

	usersLock.Lock()
	defer usersLock.Unlock()

	for i, record := range records.All() {

		metrics.Increment("Dummy.Upsert.records_read_from_iterator", 1)

		if dummy.userExportRandomlyFails() {
			metrics.Increment("Dummy.Upsert.export_failed", 1)
			recordsError[i] = errors.New("writing of user record failed (due to a causal failure probability configured in Dummy)")
			continue
		}

		var id string
		if record.ID == "" {
			// Add a new users into the in-memory users.
			metrics.Increment("Dummy.Upsert.users_created", 1)
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
				metrics.Increment("Dummy.Upsert.updated_users_not_found", 1)
				recordsError[i] = errors.New("the user to update does not exist in Dummy")
				continue
			}
			metrics.Increment("Dummy.Upsert.updated_users", 1)
			maps.Copy(user, record.Properties)
			id = record.ID
		}

		usersLastChangeTimes[id] = time.Now().UTC().Truncate(time.Microsecond)

	}

	if len(recordsError) > 0 {
		return recordsError
	}

	return nil
}

// simulateHTTPDelay simulates an HTTP delay. If the settings indicates not to
// simulate delay, this method does nothing.
func (dummy *Dummy) simulateHTTPDelay() {
	if !dummy.settings.SimulateHTTPDelay {
		return
	}
	// Determine the delay (in seconds).
	delay := rand.NormFloat64()*httpDelayStdDev + httpDelayMean
	delay = math.Max(httpDelayMin, delay)
	delay = math.Min(httpDelayMax, delay)
	// Sleep.
	time.Sleep(time.Duration(delay * 10e9))
	metrics.Increment("Dummy.simulateHTTPDelay.simulated_delays", 1)
}

// saveSettings saves the settings.
func (dummy *Dummy) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	if s.UserExportFailPercentage < 0 || s.UserExportFailPercentage > 100 {
		return meergo.NewInvalidsettingsError("percentage must be in range [0, 100]")
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

// userExportRandomlyFails determines whether exporting (i.e., writing to Dummy)
// a user should randomly fail, based on the settings.
func (dummy *Dummy) userExportRandomlyFails() bool {
	switch failPerc := dummy.settings.UserExportFailPercentage; failPerc {
	case 0:
		return false
	case 100:
		return true
	default:
		return rand.IntN(100) < failPerc
	}
}
