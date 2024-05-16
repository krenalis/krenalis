//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package googleanalytics implements the Google Analytics connector.
// (https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference)
package googleanalytics

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
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the App, AppEvents, and UIHandler interfaces.
var _ interface {
	chichi.App
	chichi.AppEvents
	chichi.UIHandler
} = (*Analytics)(nil)

// sendToDebugServer controls whether the events should be sent to the debug
// server instead of the production server.
//
// See
// https://developers.google.com/analytics/devguides/collection/protocol/ga4/validating-events?client_type=firebase
const sendToDebugServer = false

func init() {
	chichi.RegisterApp(chichi.AppInfo{
		Name:                   "Google Analytics",
		Targets:                chichi.Events,
		DestinationDescription: "send events to Google Analytics",
		Icon:                   icon,
		SendingMode:            chichi.Cloud,
	}, New)
}

// New returns a new Google Analytics connector instance.
func New(conf *chichi.AppConfig) (*Analytics, error) {
	c := Analytics{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Google Analytics connector")
		}
	}
	return &c, nil
}

type Analytics struct {
	conf     *chichi.AppConfig
	settings *Settings
}

type Settings struct {
	MeasurementID string
	APISecret     string
}

// EventRequest returns a request to dispatch an event to the app.
func (ga *Analytics) EventRequest(ctx context.Context, typ string, event *chichi.Event, extra map[string]any, schema types.Type, redacted bool) (*chichi.EventRequest, error) {
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
	switch typ {
	case "page_view":
		ev = map[string]any{
			"page_location": event.Context.Page.URL,
			"page_referrer": event.Context.Page.Referrer,
			"page_title":    event.Context.Page.Title,
		}
	case "share":
		ev = map[string]any{}
		if method, ok := extra["method"].(string); ok {
			ev["method"] = method
		}
		if contentType, ok := extra["content_type"].(string); ok {
			ev["content_type"] = contentType
		}
		if itemID, ok := extra["item_id"].(string); ok {
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
func (ga *Analytics) EventTypes(ctx context.Context) ([]*chichi.EventType, error) {
	return []*chichi.EventType{
		// https://developers.google.com/analytics/devguides/collection/ga4/views?client_type=gtag#manually_send_page_view_events
		{
			ID:          "page_view",
			Name:        "Page view",
			Description: "Send a Page view event to Google Analytics",
		},
		// https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference/events#share
		{
			ID:          "share",
			Name:        "Share",
			Description: "Send a Share event to Google Analytics",
		},
	}, nil
}

// Schema returns the schema of the specified target.
func (ga *Analytics) Schema(ctx context.Context, target chichi.Targets, eventType string) (types.Type, error) {
	switch eventType {
	case "page_view":
		return types.Type{}, nil
	case "share":
		return types.Object([]types.Property{
			{Name: "method", Type: types.Text()},
			{Name: "content_type", Type: types.Text()},
			{Name: "item_id", Type: types.Text()},
		}), nil
	}
	return types.Type{}, chichi.ErrEventTypeNotExist
}

// ServeUI serves the connector's user interface.
func (ga *Analytics) ServeUI(ctx context.Context, event string, values []byte) (*chichi.UI, error) {

	switch event {
	case "load":
		var s Settings
		if ga.settings != nil {
			s = *ga.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, ga.saveValues(ctx, values)
	default:
		return nil, chichi.ErrUIEventNotExist
	}

	ui := &chichi.UI{
		Fields: []chichi.Component{
			&chichi.Input{Name: "MeasurementID", Label: "Measurement ID", Placeholder: "G-2XYZBEB6AB", Type: "text", MinLength: 2, MaxLength: 20, HelpText: "Follow these instructions to get your Measurement ID: https://support.google.com/analytics/answer/9539598#find-G-ID"},
			&chichi.Input{Name: "APISecret", Label: "API Secret", Placeholder: "ZuHCHFZbRBi8V7u8crWFUz", Type: "text", MinLength: 1, MaxLength: 40},
		},
		Values: values,
	}

	return ui, nil

}

// saveValues saves the user-entered values as settings.
func (ga *Analytics) saveValues(ctx context.Context, values []byte) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	if n := len(s.MeasurementID); n < 2 || n > 20 {
		return chichi.NewInvalidUIValuesError("Measurement ID length must be in [2,20]")
	}
	if !strings.HasPrefix(s.MeasurementID, "G-") && !strings.HasPrefix(s.MeasurementID, "AW-") {
		return chichi.NewInvalidUIValuesError("Measurement ID must begin with 'G-' or 'AW-'")
	}
	if n := len(s.APISecret); n < 1 || n > 40 {
		return chichi.NewInvalidUIValuesError("API Secret length must be in [1,40]")
	}
	for i := 0; i < len(s.APISecret); i++ {
		c := s.APISecret[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return chichi.NewInvalidUIValuesError("API secret must contain only alphanumeric characters")
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ga.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	ga.settings = &s
	return nil
}
