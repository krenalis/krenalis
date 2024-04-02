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
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	_url "net/url"
	"strings"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"
	"github.com/open2b/chichi/ui"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the AppEvents and UI interfaces.
var _ interface {
	chichi.AppEvents
	chichi.UI
} = (*GoogleAnalytics)(nil)

// sendToDebugServer controls whether the events should be sent to the debug
// server instead of the production server.
//
// See
// https://developers.google.com/analytics/devguides/collection/protocol/ga4/validating-events?client_type=firebase
const sendToDebugServer = false

func init() {
	chichi.RegisterApp(chichi.AppInfo{
		Name:                   "Google Analytics 4",
		Targets:                chichi.Events,
		DestinationDescription: "send events to Google Analytics 4",
		Icon:                   icon,
		SendingMode:            chichi.Cloud,
	}, New)
}

// New returns a new Google Analytics 4 connector instance.
func New(conf *chichi.AppConfig) (*GoogleAnalytics, error) {
	c := GoogleAnalytics{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Google Analytics 4 connector")
		}
	}
	return &c, nil
}

type GoogleAnalytics struct {
	conf     *chichi.AppConfig
	settings *settings
}

type settings struct {
	MeasurementID string
	APISecret     string
}

// EventRequest returns an event request associated with the provided event
// type, event, and transformation data. If redacted is true, sensitive
// authentication data will be redacted in the returned request.
// This method is safe for concurrent use by multiple goroutines.
// If the specified event type does not exist, it returns the
// ErrEventTypeNotExist error.
func (ga *GoogleAnalytics) EventRequest(ctx context.Context, eventType *chichi.EventType, event *chichi.Event, data map[string]any, redacted bool) (*chichi.EventRequest, error) {
	req := &chichi.EventRequest{
		Method: "POST",
		URL:    "https://www.google-analytics.com/",
		Header: http.Header{},
	}
	if sendToDebugServer {
		req.URL += "debug/"
	}
	secret := ga.settings.APISecret
	if redacted {
		secret = "REDACTED"
	}
	req.URL += "mp/collect?api_secret=" + _url.QueryEscape(secret) + "&measurement_id=" + _url.QueryEscape(ga.settings.MeasurementID)
	req.Header.Set("Content-Type", "application/json")
	var ev map[string]any
	switch eventType.ID {
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
	}
	body := map[string]any{
		// TODO(Gianluca): consider sending the user ID as the client_id, if
		// defined, otherwise the anonymousID.
		"client_id": event.AnonymousId,
		"user_id":   event.UserId,
		"events":    []map[string]any{ev},
	}
	var err error
	req.Body, err = json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// EventTypes returns the event types of the connector's instance.
func (ga *GoogleAnalytics) EventTypes(ctx context.Context) ([]*chichi.EventType, error) {
	if ga.conf.Role == chichi.Source {
		return nil, nil
	}
	eventTypes := []*chichi.EventType{
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
func (ga *GoogleAnalytics) Resource(ctx context.Context) (string, error) {
	return "", nil
}

// ServeUI serves the connector's user interface.
func (ga *GoogleAnalytics) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if ga.settings != nil {
			s = *ga.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := ga.ValidateSettings(ctx, values)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, ga.conf.SetSettings(ctx, s)
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
func (ga *GoogleAnalytics) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
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
