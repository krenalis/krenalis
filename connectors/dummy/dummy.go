// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package dummy provides a dummy connector for testing.
package dummy

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"maps"
	"math/rand/v2"
	"net/http"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/metrics"
	"github.com/meergo/meergo/tools/types"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterAPI(connectors.APISpec{
		Code:       "dummy",
		Label:      "Dummy",
		Categories: connectors.CategoryTesting,
		AsSource: &connectors.AsAPISource{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Import customers as users from Dummy",
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsAPIDestination{
			Targets:     connectors.TargetEvent | connectors.TargetUser,
			SendingMode: connectors.Server,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Export users as customers and send events to Dummy",
				Overview: destinationOverview,
			},
		},
		Terms: connectors.APITerms{
			User:   "Customer",
			Users:  "Customers",
			UserID: "Dummy Unique ID",
		},
		EndpointGroups: []connectors.EndpointGroup{
			{
				Patterns:  []string{"/"},
				RateLimit: connectors.RateLimit{RequestsPerSecond: 100, Burst: 100},
			},
		},
	}, New)
}

// New returns a new connector instance for testing.
func New(env *connectors.APIEnv) (*Dummy, error) {
	c := Dummy{env: env}
	if len(env.Settings) > 0 {
		err := env.Settings.Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Dummy connector")
		}
	}
	return &c, nil
}

type Dummy struct {
	env      *connectors.APIEnv
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
			{Name: "email", Type: types.String(), CreateRequired: true, Description: "Email"},
			{Name: "itemName", Type: types.String(), Description: "Item name"},
			{Name: "itemId", Type: types.Int(32), Description: "Item ID"},
		}), nil
	case "send_custom_event":
		return types.Object([]types.Property{
			{Name: "email", Type: types.String(), Description: "Email"},
		}), nil
	case "send_identity":
		return types.Object([]types.Property{
			{Name: "email", CreateRequired: true, Type: types.String(), Description: "Email"},
			{Name: "traits", Type: types.Object([]types.Property{
				{Name: "address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), Description: "Street"},
					{Name: "street2", Type: types.String(), Description: "Street (second line)"},
				}), Description: "Address"},
			}), Description: "Traits"},
		}), nil
	case "send_generic_event":
		return types.Object([]types.Property{
			{Name: "properties", Type: types.JSON(), Description: "Properties"},
		}), nil
	case "send_event_with_no_schema":
		return types.Type{}, nil
	}
	return types.Type{}, connectors.ErrEventTypeNotExist
}

// EventTypes returns the event types.
func (dummy *Dummy) EventTypes(ctx context.Context) ([]*connectors.EventType, error) {
	dummy.simulateHTTPDelay()
	return []*connectors.EventType{
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
// events to the API, without actually sending it.
func (dummy *Dummy) PreviewSendEvents(ctx context.Context, events connectors.Events) (*http.Request, error) {
	return dummy.sendEvents(ctx, events, true)
}

// RecordSchema returns the schema of the specified target and role.
func (dummy *Dummy) RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {
	dummy.simulateHTTPDelay()
	var properties []types.Property
	if role == connectors.Source {
		properties = append(properties, types.Property{Name: "dummyId", Type: types.String(), Description: "Dummy ID"})
	}
	properties = append(properties, []types.Property{
		{Name: "email", Type: types.String(), Nullable: true, Description: "Email"},
		{Name: "firstName", Type: types.String(), Nullable: true, Description: "First name"},
		{Name: "fullName", Type: types.String(), Nullable: true, Description: "Full name"},
		{Name: "lastName", Type: types.String(), Nullable: true, Description: "Last name"},
		{Name: "favouriteDrink", Type: types.String().WithValues("tea", "beer", "wine", "water"), Nullable: true, Description: "Favourite drink"},
		{Name: "favourite_movie", Type: types.String(), ReadOptional: true, Description: "Favourite movie"},
	}...)
	if role == connectors.Destination {
		properties = append(properties, types.Property{Name: "additionalProperties", Type: types.Map(types.String()), Description: "Additional properties"})
	}
	properties = append(properties, []types.Property{
		{Name: "address", Type: types.Object([]types.Property{
			{Name: "street", Type: types.String(), Nullable: true, Description: "Street"},
			{Name: "postal_code", Type: types.String(), Nullable: true, Description: "Postal code"},
			{Name: "city", Type: types.String(), Nullable: true, Description: "City"},
		}), Nullable: true, Description: "Address"},
	}...)
	return types.Object(properties), nil
}

// Records returns the records of the specified target.
func (dummy *Dummy) Records(ctx context.Context, _ connectors.Targets, lastChangeTime time.Time, ids []string, _ string, _ types.Type) ([]connectors.Record, string, error) {
	metrics.Increment("Dummy.Records.calls", 1)
	dummy.simulateHTTPDelay()
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	default:
	}
	customersLock.Lock()
	defer customersLock.Unlock()
	customers := make([]connectors.Record, 0, len(allCustomers))
	for id, attributes := range allCustomers {
		if customersLastChangeTimes[id].Before(lastChangeTime) {
			continue
		}
		if ids != nil && !slices.Contains(ids, id) {
			continue
		}
		customers = append(customers, connectors.Record{
			ID:             id,
			Attributes:     deepClone(attributes),
			LastChangeTime: customersLastChangeTimes[id],
		})
	}
	sort.Slice(customers, func(i, j int) bool { return customers[i].ID < customers[j].ID })
	return customers, "", io.EOF
}

func init() {

	var rawCustomers []struct {
		ID         string
		Attributes map[string]any
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
		u.Attributes["dummyId"] = u.ID
		allCustomers[u.ID] = u.Attributes
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
	CustomerExportFailPercentage int    `json:"customerExportFailPercentage"`
	URLForDispatchingEvents      string `json:"urlForDispatchingEvents"`
	SimulateHTTPDelay            bool   `json:"simulateHTTPDelay"`
}

// SendEvents sends events to the API.
func (dummy *Dummy) SendEvents(ctx context.Context, events connectors.Events) error {
	_, err := dummy.sendEvents(ctx, events, false)
	return err
}

// ServeUI serves the connector's user interface.
func (dummy *Dummy) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

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
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{
				Name:            "customerExportFailPercentage",
				Type:            "number",
				Label:           "Percentage that the export of every single customer may fail",
				Placeholder:     "10",
				HelpText:        "0 does not fail any customer exports. 100 fails them all.",
				OnlyIntegerPart: true,
				Role:            connectors.Destination,
			},
			&connectors.Input{
				Name:        "urlForDispatchingEvents",
				Label:       "URL for dispatching events",
				Placeholder: "https://example.com",
				Role:        connectors.Destination,
			},
			&connectors.Checkbox{
				Name:  "simulateHTTPDelay",
				Label: "Pretend that Dummy operates via HTTP calls, introducing fictitious delays",
				Role:  connectors.Both,
			},
		},
		Settings: settings,
	}

	return ui, nil
}

// nonRequiredProperties contains the names of the properties that are both in
// the source and destination schema and are not requires for create.
var nonRequiredProperties = []string{"email", "firstName", "lastName", "fullName", "favouriteDrink", "address"}

// Upsert updates or creates records in the API for the specified target.
func (dummy *Dummy) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records) error {

	dummy.simulateHTTPDelay()

	recordsError := make(connectors.RecordsError)

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
			customer := maps.Clone(record.Attributes)
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
			maps.Copy(customer, record.Attributes)
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
		return connectors.NewInvalidSettingsError("percentage must be in range [0, 100]")
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = dummy.env.SetSettings(ctx, b)
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

// sendEvents sends the given events to the API and returns the sent HTTP
// request. If preview is true, the HTTP request is built but not sent, so it is
// only returned.
//
// If an error occurs while sending the events to the API, a nil *http.Request
// and the error are returned.
func (dummy *Dummy) sendEvents(ctx context.Context, events connectors.Events, preview bool) (*http.Request, error) {
	event := events.First()
	var body []byte
	if event.Type.Values != nil {
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
	// Mark the request as idempotent.
	req.Header["Idempotency-Key"] = nil
	if preview {
		return req, nil
	}
	res, err := dummy.env.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	switch res.StatusCode {
	case 200, 201, 202, 204:
	default:
		return nil, fmt.Errorf("Dummy endpoint responded with error code %d", res.StatusCode)
	}
	return req, nil
}

// deepClone returns a deep clone of the provided properties.
func deepClone(properties map[string]interface{}) map[string]interface{} {
	bytes, _ := json.Marshal(properties)
	var clone map[string]any
	_ = json.Unmarshal(bytes, &clone)
	return clone
}
