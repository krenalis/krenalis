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
	"errors"
	"io"
	"maps"
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

// Connector icon.
var icon = "<svg></svg>"

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name:       "Dummy",
		Categories: meergo.CategoryTest,
		AsSource: &meergo.AsAppSource{
			Targets:     meergo.TargetUser,
			HasSettings: true,
			Documentation: meergo.ConnectorRoleDocumentation{
				Summary:  "Import customers as users from Dummy",
				Overview: sourceOverview,
			},
		},
		AsDestination: &meergo.AsAppDestination{
			Targets:     meergo.TargetEvent | meergo.TargetUser,
			SendingMode: meergo.ClientAndServer,
			HasSettings: true,
			Documentation: meergo.ConnectorRoleDocumentation{
				Summary:  "Export users as customers and send events to Dummy",
				Overview: destinationOverview,
			},
		},
		Terms: meergo.AppTerms{
			User:  "customer",
			Users: "customers",
		},
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
	allCustomers             map[string]map[string]any
	customersLastChangeTimes map[string]time.Time
	customersLock            sync.Mutex
)

//go:embed customers.json
var jsonCustomers []byte

func newDummyId() string {
	b := make([]rune, 12)
	for i := range b {
		b[i] = rune(rand.IntN(20) + 'a')
	}
	return "dummy_" + string(b)
}

// EventTypeSchema returns the schema of the specified event type.
func (dummy *Dummy) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {
	dummy.simulateHTTPDelay()
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

// EventTypes returns the event types.
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

// PreviewSendEvents returns the HTTP request that would be used to send the
// events to the app, without actually sending it.
func (dummy *Dummy) PreviewSendEvents(ctx context.Context, events meergo.Events) (*http.Request, error) {
	return dummy.sendEvents(ctx, events, true)
}

// RecordSchema returns the schema of the specified target and role.
func (dummy *Dummy) RecordSchema(ctx context.Context, target meergo.Targets, role meergo.Role) (types.Type, error) {
	dummy.simulateHTTPDelay()
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

// Records returns the records of the specified target.
func (dummy *Dummy) Records(ctx context.Context, _ meergo.Targets, lastChangeTime time.Time, ids, _ []string, _ string, _ types.Type) ([]meergo.Record, string, error) {
	metrics.Increment("Dummy.Records.calls", 1)
	dummy.simulateHTTPDelay()
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	default:
	}
	customersLock.Lock()
	defer customersLock.Unlock()
	customers := make([]meergo.Record, 0, len(allCustomers))
	for id, props := range allCustomers {
		if customersLastChangeTimes[id].Before(lastChangeTime) {
			continue
		}
		if ids != nil && !slices.Contains(ids, id) {
			continue
		}
		customers = append(customers, meergo.Record{
			ID:             id,
			Properties:     deepClone(props),
			LastChangeTime: customersLastChangeTimes[id],
		})
	}
	sort.Slice(customers, func(i, j int) bool { return customers[i].ID < customers[j].ID })
	return customers, "", io.EOF
}

func init() {

	var rawCustomers []struct {
		ID         string
		Properties map[string]any
	}
	err := json.Unmarshal(jsonCustomers, &rawCustomers)
	if err != nil {
		panic(err)
	}

	// Only the first 10 customers are taken. The others, with the current
	// implementation of Dummy, remain defined in the JSON file but are not
	// used.
	rawCustomers = rawCustomers[:10]

	now := time.Now().UTC()

	customersLock.Lock()
	allCustomers = make(map[string]map[string]any, len(rawCustomers))
	customersLastChangeTimes = make(map[string]time.Time, len(rawCustomers))
	for _, u := range rawCustomers {
		u.Properties["dummyId"] = u.ID
		allCustomers[u.ID] = u.Properties
		// Go back in time by a maximum of 100 milliseconds. This allows
		// timestamps to be spread over a time frame large enough to maintain
		// some randomness, but not so large that a timestamp is in the past
		// since the last import.
		nanosecDelta := rand.IntN(100e6)
		customersLastChangeTimes[u.ID] = now.Add(-time.Duration(nanosecDelta)).Truncate(time.Microsecond)
	}
	customersLock.Unlock()

}

type innerSettings struct {
	CustomerExportFailPercentage int // in [0, 100]
	URLForDispatchingEvents      string
	SimulateHTTPDelay            bool
}

// SendEvents sends events to the app.
func (dummy *Dummy) SendEvents(ctx context.Context, events meergo.Events) error {
	_, err := dummy.sendEvents(ctx, events, false)
	return err
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
				Name:            "CustomerExportFailPercentage",
				Type:            "number",
				Label:           "Percentage that the export of every single customer may fail",
				Placeholder:     "10",
				HelpText:        "0 does not fail any customer exports. 100 fails them all.",
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

	recordsError := make(meergo.RecordsError)

	customersLock.Lock()
	defer customersLock.Unlock()

	n := 0
	for record := range records.All() {

		metrics.Increment("Dummy.Upsert.records_read_from_iterator", 1)

		if dummy.customerExportRandomlyFails() {
			metrics.Increment("Dummy.Upsert.export_failed", 1)
			recordsError[n] = errors.New("writing of customer record failed (due to a causal failure probability configured in Dummy)")
			n++
			continue
		}

		var id string
		if record.ID == "" {
			// Add a new customers into the in-memory customers.
			metrics.Increment("Dummy.Upsert.customers_created", 1)
			customer := maps.Clone(record.Properties)
			id = newDummyId()
			customer["dummyId"] = id
			for _, p := range nonRequiredProperties {
				if v, ok := customer[p]; !ok {
					customer[p] = nil
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
			allCustomers[id] = customer
		} else {
			// Update the in-memory customers.
			customer, ok := allCustomers[record.ID]
			if !ok {
				metrics.Increment("Dummy.Upsert.updated_customers_not_found", 1)
				recordsError[n] = errors.New("the customer to update does not exist in Dummy")
				n++
				continue
			}
			metrics.Increment("Dummy.Upsert.updated_customers", 1)
			maps.Copy(customer, record.Properties)
			id = record.ID
		}

		customersLastChangeTimes[id] = time.Now().UTC().Truncate(time.Microsecond)
		n++

	}

	if len(recordsError) > 0 {
		return recordsError
	}

	return nil
}

// customerExportRandomlyFails determines whether exporting (i.e., writing to
// Dummy) a customer should randomly fail, based on the settings.
func (dummy *Dummy) customerExportRandomlyFails() bool {
	switch failPerc := dummy.settings.CustomerExportFailPercentage; failPerc {
	case 0:
		return false
	case 100:
		return true
	default:
		return rand.IntN(100) < failPerc
	}
}

// saveSettings saves the settings.
func (dummy *Dummy) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	if s.CustomerExportFailPercentage < 0 || s.CustomerExportFailPercentage > 100 {
		return meergo.NewInvalidSettingsError("percentage must be in range [0, 100]")
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

// simulateHTTPDelay simulates an HTTP delay. If the settings indicate not to
// simulate delay, this method does nothing.
func (dummy *Dummy) simulateHTTPDelay() {
	if !dummy.settings.SimulateHTTPDelay {
		return
	}
	latency := rand.Float64()*1.3 + 1.5 // seconds.
	time.Sleep(time.Duration(latency * 1e9))
	metrics.Increment("Dummy.simulateHTTPDelay.simulated_delays", 1)
}

// sendEvents sends the given events to the app and returns the sent HTTP
// request.
// If preview is true, the HTTP request is built but not sent, so it is
// only returned.
//
// If an error occurs while sending the events to the app, a nil *http.Request
// and the error are returned.
func (dummy *Dummy) sendEvents(ctx context.Context, events meergo.Events, preview bool) (*http.Request, error) {
	event := events.First()
	var body []byte
	if event.Type.Schema.Valid() {
		var err error
		body, err = types.Marshal(event.Type.Values, event.Type.Schema)
		if err != nil {
			return nil, err
		}
	}
	u := "https://example.com/"
	if dummy.settings.URLForDispatchingEvents != "" {
		u = dummy.settings.URLForDispatchingEvents
	}
	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// Mark the request as idempotent.
	req.Header["Idempotency-Key"] = nil
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	if preview {
		return req, nil
	}
	_, err = dummy.conf.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	// TODO: handle errors
	return req, nil
}

// deepClone returns a deep clone of the provided properties.
func deepClone(properties map[string]interface{}) map[string]interface{} {
	bytes, _ := json.Marshal(properties)
	var clone map[string]any
	_ = json.Unmarshal(bytes, &clone)
	return clone
}
