//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package googleanalytics4 implements the Google Analytics 4 connector.
// (https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference)
package googleanalytics4

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	_url "net/url"
	"strings"

	"chichi/connector"
	"chichi/connector/types"
	"chichi/connector/ui"
)

// Connector icon.
var icon = "<svg></svg>"

// sendToDebugServer controls whether the events should be sent to the debug
// server instead of the production server; it also enables printing some debug
// information on the log.
//
// See
// https://developers.google.com/analytics/devguides/collection/protocol/ga4/validating-events?client_type=firebase
const sendToDebugServer = false

// Make sure it implements the UI and the AppEventsConnection interfaces.
var _ interface {
	connector.UI
	connector.AppEventsConnection
} = (*connection)(nil)

func init() {
	connector.RegisterApp(connector.App{
		Name:                   "Google Analytics 4",
		DestinationDescription: "send events to Google Analytics 4",
		Icon:                   icon,
	}, open)
}

// open opens a Google Analytics 4 connection and returns it.
func open(conf *connector.AppConfig) (*connection, error) {
	c := connection{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Google Analytics 4 connection")
		}
	}
	return &c, nil
}

type connection struct {
	conf     *connector.AppConfig
	settings *settings
}

type settings struct {
	MeasurementID string
	APISecret     string
}

// EventTypes returns the connection's event types.
func (c *connection) EventTypes(ctx context.Context) ([]*connector.EventType, error) {
	if c.conf.Role == connector.SourceRole {
		return nil, nil
	}
	eventTypes := []*connector.EventType{
		// https://developers.google.com/analytics/devguides/collection/ga4/views?client_type=gtag#manually_send_page_view_events
		{
			ID:          "page_view",
			Name:        "Page view",
			Description: "Send a Page view event to Google Analytics 4",
		},
		// https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference/events#share
		{
			ID:          "share",
			Name:        "Share",
			Description: "Send a Share event to Google Analytics 4",
			Schema: types.Object([]types.Property{
				{Name: "method", Type: types.Text()},
				{Name: "content_type", Type: types.Text()},
				{Name: "item_id", Type: types.Text()},
			}),
		},
	}
	return eventTypes, nil
}

// Resource returns the resource from a client token.
func (c *connection) Resource(ctx context.Context) (string, error) {
	return "", nil
}

// SendEvent sends the event, along with the given mapped data.
// eventType specifies the event type corresponding to the event.
func (c *connection) SendEvent(ctx context.Context, eventType string, event *connector.Event, data map[string]any) error {
	return c.collect(eventBody(eventType, event, data))
}

// SendEventPreview returns a preview of the event that would be sent when
// calling SendEvent with the same arguments.
func (c *connection) SendEventPreview(ctx context.Context, eventType string, event *connector.Event, data map[string]any) ([]byte, error) {
	return json.MarshalIndent(eventBody(eventType, event, data), "", "\t")
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
		return nil, nil, c.conf.SetSettings(ctx, s)
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "MeasurementID", Label: "Measurement ID", Placeholder: "G-2XYZBEB6AB", Type: "text", MinLength: 2, MaxLength: 20, HelpText: "Follow these instructions to get your Measurement ID: https://support.google.com/analytics/answer/9539598#find-G-ID"},
			&ui.Input{Name: "APISecret", Label: "API Secret", Placeholder: "ZuHCHFZbRBi8V7u8crWFUz", Type: "text", MinLength: 1, MaxLength: 40},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil

}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	if n := len(s.MeasurementID); n < 2 || n > 20 {
		return nil, ui.Errorf("Measurement ID length must be in [2,20]")
	}
	if !strings.HasPrefix(s.MeasurementID, "G-") && !strings.HasPrefix(s.MeasurementID, "AW-") {
		return nil, ui.Errorf("Measurement ID must begin with 'G-' or 'AW-'")
	}
	if n := len(s.APISecret); n < 1 || n > 40 {
		return nil, ui.Errorf("API Secret length must be in [1,40]")
	}
	for i := 0; i < len(s.APISecret); i++ {
		c := s.APISecret[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return nil, ui.Errorf("API secret must contain only alphanumeric characters")
		}
	}
	return json.Marshal(&s)
}

func (c *connection) collect(body any) error {

	// Build the URL.
	url := &_url.URL{
		Scheme: "https",
		Host:   "www.google-analytics.com",
		Path:   "/mp/collect",
	}
	if sendToDebugServer {
		url.Path = "/debug/mp/collect"
	}
	values := url.Query()
	values.Add("api_secret", c.settings.APISecret)
	values.Add("measurement_id", c.settings.MeasurementID)
	url.RawQuery = values.Encode()

	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(body)
	if err != nil {
		return err
	}

	// Issue the POST request to www.google-analytics.com.
	req, err := http.NewRequest("POST", url.String(), &b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.conf.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%q returned status code %d", url, resp.StatusCode)
	}

	// Print some information, if in test mode.
	if sendToDebugServer {
		response, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		log.Printf("%q returned status code %d and this content: %s", url, resp.StatusCode, response)
	}

	return nil
}

func eventBody(eventType string, event *connector.Event, data map[string]any) any {
	var ev map[string]any
	switch eventType {
	case "page_view":
		ev = map[string]any{
			"page_location": event.Context.Page.URL,
			"page_referrer": event.Context.Page.Referrer,
			"page_title":    event.Context.Page.Title,
		}
	case "share":
		ev = map[string]any{}
		if method, ok := data["method"].(string); ok {
			ev["method"] = method
		}
		if contentType, ok := data["content_type"].(string); ok {
			ev["content_type"] = contentType
		}
		if itemID, ok := data["item_id"].(string); ok {
			ev["item_id"] = itemID
		}
	default:
		panic(fmt.Sprintf("unsupported event type %q", eventType))
	}
	body := map[string]any{
		// TODO(Gianluca): consider sending the user ID as the client_id, if
		// defined, otherwise the anonymousID.
		"client_id": event.AnonymousId,
		"user_id":   event.UserId,
		"events":    []map[string]any{ev},
	}
	return body
}
