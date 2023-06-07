//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

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

// Make sure it implements the AppEventsConnection interface.
var _ connector.AppEventsConnection = (*connection)(nil)

func init() {
	connector.RegisterApp(connector.App{
		Name:                   "Google Analytics 4",
		DestinationDescription: "send events to Google Analytics 4",
		Icon:                   icon,
	}, open)
}

type connection struct {
	ctx         context.Context
	role        connector.Role
	settings    *settings
	setSettings connector.SetSettingsFunc
	httpClient  connector.HTTPClient
}

type settings struct {
	MeasurementID string
	APISecret     string
}

// open opens a Google Analytics 4 connection and returns it.
func open(ctx context.Context, conf *connector.AppConfig) (*connection, error) {
	c := connection{
		ctx:         ctx,
		role:        conf.Role,
		setSettings: conf.SetSettings,
		httpClient:  conf.HTTPClient,
	}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Google Analytics 4 connection")
		}
	}
	return &c, nil
}

// EventTypes returns the connection's event types.
func (c *connection) EventTypes() ([]*connector.EventType, error) {
	if c.role == connector.SourceRole {
		return nil, nil
	}
	eventTypes := []*connector.EventType{
		// https://developers.google.com/analytics/devguides/collection/ga4/views?client_type=gtag#manually_send_page_view_events
		{
			ID:          "event_page_view",
			Name:        "Page view",
			Description: "Send a Page view event to Google Analytics 4",
		},
		// https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference/events#share
		{
			ID:          "event_share",
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
func (c *connection) Resource() (string, error) {
	return "", nil
}

// SendEvent sends the event, along with the given mapped event.
// eventType specifies the event type corresponding to the event.
func (c *connection) SendEvent(event connector.Event, mappedEvent map[string]any, eventType string) error {
	var err error
	switch eventType {
	case "event_page_view":
		err = c.collect(event.AnonymousID, event.UserID, "page_view", map[string]any{
			"page_location": event.Page.URL,
			"page_referrer": event.Page.Referrer,
			"page_title":    event.Page.Title,
		})
	case "event_share":
		params := map[string]any{}
		if method, ok := mappedEvent["method"].(string); ok {
			params["method"] = method
		}
		if contentType, ok := mappedEvent["content_type"].(string); ok {
			params["content_type"] = contentType
		}
		if itemID, ok := mappedEvent["item_id"].(string); ok {
			params["item_id"] = itemID
		}
		err = c.collect(event.AnonymousID, event.UserID, "share", params)
	default:
		panic(fmt.Sprintf("unsupported event type %q", eventType))
	}
	return err
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

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
		s, err := c.ValidateSettings(values)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, c.setSettings(s)
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
func (c *connection) ValidateSettings(values []byte) ([]byte, error) {
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

func (c *connection) collect(anonymousID, userID, eventName string, eventParams map[string]any) error {
	// See https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference.

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

	// Build the request's data.
	data := map[string]any{
		// TODO(Gianluca): consider sending the user ID as the client_id, if
		// defined, otherwise the anonymousID.
		"client_id": anonymousID,
		"user_id":   userID,
		"events": []map[string]any{
			{
				"name":   eventName,
				"params": eventParams,
			},
		},
	}
	body := &bytes.Buffer{}
	err := json.NewEncoder(body).Encode(data)
	if err != nil {
		return err
	}

	// Issue the POST request to www.google-analytics.com.
	req, err := http.NewRequest("POST", url.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
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
