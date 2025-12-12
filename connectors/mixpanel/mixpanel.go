// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package mixpanel provides a connector for Mixpanel.
// (https://developer.mixpanel.com/reference/overview)
//
// Mixpanel is a trademark of Mixpanel, Inc.
// This connector is not affiliated with or endorsed by Mixpanel, Inc.
package mixpanel

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

// Mixpanel supports NoEncoding and Gzip for request bodies.
const contentEncoding = connectors.Gzip

//go:embed documentation/overview.md
var overview string

func init() {
	connectors.RegisterAPI(connectors.APISpec{
		Code:       "mixpanel",
		Label:      "Mixpanel",
		Categories: connectors.CategorySaaS,
		AsDestination: &connectors.AsAPIDestination{
			Targets:     connectors.TargetEvent,
			HasSettings: true,
			SendingMode: connectors.Server,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Send events to Mixpanel",
				Overview: overview,
			},
		},
		EndpointGroups: []connectors.EndpointGroup{{
			// https://developer.mixpanel.com/reference/import-events
			RateLimit: connectors.RateLimit{RequestsPerSecond: 15, Burst: 20, MaxConcurrentRequests: 20},
			// https://developer.mixpanel.com/reference/import-events#rate-limits
			RetryPolicy: connectors.RetryPolicy{
				"429":     connectors.ExponentialStrategy(connectors.Slowdown, 2*time.Second),
				"502 503": connectors.ExponentialStrategy(connectors.NetFailure, 2*time.Second),
			}},
		},
	}, New)
}

type Mixpanel struct {
	env      *connectors.APIEnv
	settings *innerSettings
}

type innerSettings struct {
	ProjectID     string
	ProjectToken  string
	DataResidency string
}

// New returns a new connector instance for Mixpanel.
func New(env *connectors.APIEnv) (*Mixpanel, error) {
	c := Mixpanel{env: env}
	if len(env.Settings) > 0 {
		err := env.Settings.Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for Mixpanel")
		}
	}
	return &c, nil
}

// EventTypes returns the event types.
func (mp *Mixpanel) EventTypes(ctx context.Context) ([]*connectors.EventType, error) {
	return []*connectors.EventType{
		{
			ID:          "order_completed",
			Name:        "Send order completed events",
			Description: "Send order completed events to Mixpanel",
			Filter:      "type is 'track' and event is 'Order Completed'",
		},
		{
			ID:          "product_purchased",
			Name:        "Send product purchased events",
			Description: "Send an event to Mixpanel for every product purchased",
			Filter:      "type is 'track' and event is 'Order Completed' and properties.products is not empty",
		},
		{
			ID:          "track",
			Name:        "Send track events",
			Description: "Send track events to Mixpanel",
			Filter:      "type is 'track' and event is not 'Order Completed'",
		},
		{
			ID:          "page",
			Name:        "Send page events",
			Description: "Send page events to Mixpanel",
			Filter:      "type is 'page'",
		},
		{
			ID:          "screen",
			Name:        "Send screen events",
			Description: "Send screen events to Mixpanel",
			Filter:      "type is 'screen'",
		},
	}, nil
}

// PreviewSendEvents returns the HTTP request that would be used to send the
// events to the API, without actually sending it.
func (mp *Mixpanel) PreviewSendEvents(ctx context.Context, events connectors.Events) (*http.Request, error) {
	return mp.sendEvents(ctx, events, true)
}

// SendEvents sends events to the API.
func (mp *Mixpanel) SendEvents(ctx context.Context, events connectors.Events) error {
	_, err := mp.sendEvents(ctx, events, false)
	return err
}

// EventTypeSchema returns the schema of the specified event type.
func (mp *Mixpanel) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {

	var schema types.Type

	switch eventType {
	case "order_completed":
		schema = types.Object([]types.Property{
			{Name: "event", Prefilled: `"Order Completed"`, Type: types.String().WithMaxLength(255), CreateRequired: true, Description: "Event name"},
			{
				Name: "properties",
				Prefilled: `map(` +
					`"order_id", properties.order_id,` +
					`"affiliation",properties.affiliation,` +
					`"currency",properties.currency,` +
					`"revenue",properties.revenue,` +
					`"coupon",properties.coupon,` +
					`"discount",properties.discount,` +
					`"shipping",properties.shipping,` +
					`"tax",properties.tax,` +
					`"value",properties.value,` +
					`"products",properties.products)`,
				Type:        types.Map(types.JSON()),
				Description: "Event properties",
			},
		})
	case "product_purchased":
		schema = types.Object([]types.Property{
			{Name: "event", Prefilled: `"Product Purchased"`, Type: types.String().WithMaxLength(255), CreateRequired: true, Description: "Event name"},
			{
				Name:           "products",
				Prefilled:      "properties.products",
				Type:           types.Array(types.Map(types.JSON())).WithMinElements(1),
				CreateRequired: true,
				Description:    "Purchased products",
			},
			{
				Name:        "properties",
				Type:        types.Map(types.JSON()),
				Description: "Event properties",
			},
		})
	default:
		var event string
		switch eventType {
		case "page":
			event = `"Viewed " name`
		case "screen":
			event = `"Viewed " name`
		case "track":
			event = "event"
		default:
			return types.Type{}, connectors.ErrEventTypeNotExist
		}
		schema = types.Object([]types.Property{
			{Name: "event", Prefilled: event, Type: types.String().WithMaxLength(255), CreateRequired: true, Description: "Event name"},
			{
				Name:        "properties",
				Type:        types.Map(types.JSON()),
				Description: "Event properties",
			},
		})
	}

	return schema, nil
}

// ServeUI serves the connector's user interface.
func (mp *Mixpanel) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if mp.settings != nil {
			s = *mp.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, mp.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "ProjectID", Label: "Project ID", Placeholder: "1234567", Type: "text", MinLength: 1, MaxLength: 20},
			&connectors.Input{Name: "ProjectToken", Label: "Project Token", Placeholder: "d8e8fca2dc0f896fd7cb4cb0031ba249", Type: "text", MinLength: 32, MaxLength: 32},
			&connectors.Select{Name: "DataResidency", Label: "Data Residency", Options: []connectors.Option{
				{Text: "United States", Value: "US"},
				{Text: "European Union", Value: "EU"},
				{Text: "India", Value: "IN"},
			}},
		},
		Settings: settings,
	}

	return ui, nil
}

// saveSettings validates and saves the settings.
func (mp *Mixpanel) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Validate ProjectID.
	if n, err := strconv.Atoi(s.ProjectID); err != nil || n < 0 {
		return connectors.NewInvalidSettingsError("Project ID must be a positive number")
	}
	// Validate ProjectToken.
	if n := len(s.ProjectToken); n < 1 || n > 100 {
		return connectors.NewInvalidSettingsError("Project Token length must be in range [1,100]")
	}
	for i := 0; i < len(s.ProjectToken); i++ {
		c := s.ProjectToken[i]
		// ASCII characters with decimal codes from 33 (!) to 126 (~),
		// inclusive, are printable characters. The space character, having
		// decimal code 32, is therefore excluded from the range of accepted
		// characters, and this is intentional.
		if c < 33 || c > 126 {
			return connectors.NewInvalidSettingsError("Project Token must contain only valid characters")
		}
	}
	// Validate DataResidency.
	switch s.DataResidency {
	case "US", "EU", "IN":
	default:
		return connectors.NewInvalidSettingsError("Data Residency must be set to US, EU, or IN")
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = mp.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	mp.settings = &s
	return nil
}

const (
	// For more details, see:
	// https://developer.mixpanel.com/reference/import-events.
	maxEventsPerRequest = 2_000            // 2000 events per request.
	maxBodyEventsBytes  = 10 * 1024 * 1024 // 10 MB (uncompressed) per request.
)

type contextKey byte

// sendBadRequestContextKey, when set to true in the context, signals sendEvents
// to send an intentionally invalid request to Mixpanel. This is used to
// simulate a "Bad Request" scenario, where Mixpanel is expected to return a Bad
// Request error.
const sendBadRequestContextKey contextKey = 0

// sendEvents sends the given events to the API and returns the sent HTTP
// request. If preview is true, the HTTP request is built but not sent, so it is
// only returned.
//
// If an error occurs while sending the events to the API, a nil *http.Request
// and the error are returned.
func (mp *Mixpanel) sendEvents(ctx context.Context, events connectors.Events, preview bool) (*http.Request, error) {

	sendBadRequest, _ := ctx.Value(sendBadRequestContextKey).(bool)

	// bb contains newline-delimited JSON objects representing the events.
	bb := mp.env.HTTPClient.GetBodyBuffer(contentEncoding)
	defer bb.Close()

	n := 0
	for event := range events.All() {

		values := event.Type.Values

		if values["event"].(string) == "" {
			return nil, errors.New("event cannot be empty")
		}

		// Build a unique identifier for the event.
		insertId := "[PIPELINE]"
		if !preview {
			insertId = strconv.Itoa(event.DestinationPipeline)
		}
		insertId += "*" + event.Received.MessageId()

		properties := map[string]any{
			"$device_id": event.Received.AnonymousId(),
			"$insert_id": insertId,
			"$source":    "meergo",
			"time":       event.Received.Timestamp().UnixMilli(),
		}
		if sendBadRequest {
			delete(properties, "time")
		}
		if userID, ok := event.Received.UserID(); ok {
			properties["$user_id"] = userID
		}

		// Set distinct_id. This has no effect if the project uses the simplified ID Merge API.
		if userID, ok := event.Received.UserID(); ok {
			properties["distinct_id"] = userID
		} else {
			properties["distinct_id"] = event.Received.AnonymousId()
		}

		if context, ok := event.Received.Context(); ok {
			if app, ok := context.App(); ok {
				if name, ok := app.Name(); ok {
					properties["$app_name"] = name
				}
				if version, ok := app.Version(); ok {
					properties["$app_version_string"] = version
				}
				if build, ok := app.Build(); ok {
					properties["$app_build_number"] = build
				}
				if namespace, ok := app.Namespace(); ok {
					properties["$app_namespace"] = namespace
				}
			}
			if browser, ok := context.Browser(); ok {
				if name, ok := browser.Name(); ok {
					properties["$browser"] = name
				} else if other, ok := browser.Other(); ok {
					properties["$browser"] = other
				}
				if _, ok := properties["$browser"]; ok {
					if version, ok := browser.Version(); ok {
						properties["$browser_version"] = version
					}
				}
			}
			if campaign, ok := context.Campaign(); ok {
				if source, ok := campaign.Source(); ok {
					properties["utm_source"] = source
				}
				if name, ok := campaign.Name(); ok {
					properties["utm_campaign"] = name
				}
				if medium, ok := campaign.Medium(); ok {
					properties["utm_medium"] = medium
				}
				if term, ok := campaign.Term(); ok {
					properties["utm_term"] = term
				}
				if content, ok := campaign.Content(); ok {
					properties["utm_content"] = content
				}
			}
			if device, ok := context.Device(); ok {
				if id, ok := device.Id(); ok {
					properties["device_id"] = id
				}
				if advertisingId, ok := device.AdvertisingId(); ok {
					properties["$ios_ifa"] = advertisingId
				}
				if adTrackingEnabled, ok := device.AdTrackingEnabled(); ok {
					properties["ad_tracking_enabled"] = adTrackingEnabled
				}
				if manufacturer, ok := device.Manufacturer(); ok {
					properties["$manufacturer"] = manufacturer
				}
				if model, ok := device.Model(); ok {
					properties["$model"] = model
				}
				if name, ok := device.Name(); ok {
					properties["$device"] = name
					properties["$device_name"] = name
				}
				if typ, ok := device.Type(); ok {
					properties["$device_type"] = typ
				}
			}
			if ip, ok := context.IP(); ok {
				properties["ip"] = ip
			}
			if library, ok := context.Library(); ok {
				if name, ok := library.Name(); ok {
					if strings.HasPrefix(name, "meergo") || strings.HasPrefix(name, "Meergo") {
						properties["mp_lib"] = name
					} else {
						properties["mp_lib"] = "Meergo: " + name
					}
					if version, ok := library.Version(); ok {
						properties["$lib_version"] = version
					}
				}
			}
			if locale, ok := context.Locale(); ok {
				properties["$locale"] = locale
			}
			if location, ok := context.Location(); ok {
				country, okCountry := location.Country()
				city, okCity := location.City()
				if okCountry || okCity {
					if okCountry {
						properties["mp_country_code"] = country
					} else {
						properties["mp_country_code"] = nil
					}
					if okCity {
						properties["$city"] = city
					} else {
						properties["$city"] = nil
					}
				}
			}
			if network, ok := context.Network(); ok {
				if bluetooth, ok := network.Bluetooth(); ok {
					properties["$bluetooth_enabled"] = bluetooth
				}
				if carrier, ok := network.Carrier(); ok {
					properties["$carrier"] = carrier
				}
				if cellular, ok := network.Cellular(); ok {
					properties["$cellular_enabled"] = cellular
				}
				if wifi, ok := network.WiFi(); ok {
					properties["$wifi_enabled"] = wifi
				}
			}
			if os, ok := context.OS(); ok {
				if osName, ok := os.Name(); ok {
					properties["$os"] = osName
					if osVersion, ok := os.Version(); ok {
						properties["$os_version"] = osVersion
					}
				}
			}
			if page, ok := context.Page(); ok {
				if referrer, ok := page.Referrer(); ok {
					u, err := url.Parse(referrer)
					if err == nil {
						properties["$referrer"] = referrer
						properties["$referring_domain"] = u.Hostname()
					}
				}
				if pageURL, ok := page.URL(); ok {
					properties["$current_url"] = pageURL
					properties["current_page_title"], _ = page.Title()
					u, err := url.Parse(pageURL)
					if err == nil {
						properties["current_domain"] = u.Hostname()
						properties["current_url_path"] = u.Path
						properties["current_url_protocol"] = u.Scheme + ":"
					}
				}
			}
			if screen, ok := context.Screen(); ok {
				if w, ok := screen.Width(); ok {
					properties["$screen_width"] = w
				}
				if h, ok := screen.Height(); ok {
					properties["$screen_height"] = h
				}
				if d, ok := screen.Density(); ok {
					properties["$screen_density"] = d
				}
			}
			if session, ok := context.Session(); ok {
				if id, ok := session.Id(); ok {
					properties["session_id"] = id
				}
			}
			if timezone, ok := context.Timezone(); ok {
				properties["timezone"] = timezone
			}
		}

		if pp, ok := values["properties"].(map[string]any); ok {
			for name, value := range pp {
				properties[name] = value
			}
		}

		if event.Type.ID == "product_purchased" {
			// Generate an event for every product purchased.
			insertID := properties["$insert_id"].(string)
			delete(properties, "$insert_id")
			time := properties["time"].(int64)
			products := values["products"].([]any)
			for i, product := range products {
				pp := maps.Clone(properties)
				pp["$insert_id"] = strconv.Itoa(i+1) + "#" + insertID
				pp["time"] = time + 1 + int64(i)
				for name, value := range product.(map[string]any) {
					if _, ok := pp[name]; !ok {
						pp[name] = value
					}
				}
				err := bb.Encode(map[string]any{
					"event":      values["event"].(string),
					"properties": pp,
				})
				if err != nil {
					return nil, err
				}
				bb.WriteByte('\n')
			}
		} else {
			err := bb.Encode(map[string]any{
				"event":      values["event"].(string),
				"properties": properties,
			})
			if err != nil {
				return nil, err
			}
			bb.WriteByte('\n')
		}

		if bb.Len() > maxBodyEventsBytes {
			bb.Truncate(0)
			events.Postpone()
			break
		}

		if err := bb.Flush(); err != nil {
			return nil, err
		}

		n++
		if n == maxEventsPerRequest {
			break
		}
	}

	u := "https://api.mixpanel.com/"
	switch mp.settings.DataResidency {
	case "EU":
		u = "https://api-eu.mixpanel.com/"
	case "IN":
		u = "https://api-in.mixpanel.com/"
	}
	u += "import?strict=1&project_id=" + mp.settings.ProjectID

	req, err := bb.NewRequest(ctx, "POST", u)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	req.Header["Idempotency-Key"] = nil // mark the request as idempotent

	if preview {
		req.Header.Set("Authorization", "Basic [REDACTED]")
	} else {
		req.SetBasicAuth(mp.settings.ProjectToken, "")
	}

	if preview {
		return req, nil
	}

	// Send the request.
	res, err := mp.env.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == 200 {
		return req, nil
	}

	// Handle the error.
	if res.StatusCode != 400 {
		return nil, fmt.Errorf("Mixpanel server responded with %d error code", res.StatusCode)
	}
	var out struct {
		FailedRecords []struct {
			Index    int    `json:"index"`
			InsertId string `json:"insert_id"`
			Field    string `json:"field"`
			Message  string `json:"message"`
		} `json:"failed_records"`
	}
	err = json.Decode(res.Body, &out)
	if err != nil {
		return nil, fmt.Errorf("cannot decode Mixpanel response: %v", err)
	}
	if len(out.FailedRecords) == 0 {
		return nil, errors.New("Mixpanel responded with status 400, but 'failed_records' was empty")
	}
	eventErrors := make(connectors.EventsError, len(out.FailedRecords))
	for _, record := range out.FailedRecords {
		eventErrors[record.Index] = fmt.Errorf("%s: %s", record.Field, record.Message)
	}

	return nil, eventErrors
}
