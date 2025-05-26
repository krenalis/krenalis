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
	"errors"
	"net/http"
	_url "net/url"
	"strings"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

// sendToDebugServer controls whether the events should be sent to the debug
// server instead of the production server.
//
// See
// https://developers.google.com/analytics/devguides/collection/protocol/ga4/validating-events?client_type=firebase
const sendToDebugServer = false

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name: "Google Analytics",
		AsDestination: &meergo.AsAppDestination{
			Description: "Send events to Google Analytics",
			Targets:     meergo.EventsTarget,
			HasSettings: true,
			SendingMode: meergo.Cloud,
		},
		Icon: icon,
	}, New)
}

// New returns a new Google Analytics connector instance.
func New(conf *meergo.AppConfig) (*Analytics, error) {
	c := Analytics{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Google Analytics connector")
		}
	}
	return &c, nil
}

type Analytics struct {
	conf     *meergo.AppConfig
	settings *innerSettings
}

type innerSettings struct {
	MeasurementID string
	APISecret     string
}

// EventRequest returns a request to dispatch an event to the app.
func (ga *Analytics) EventRequest(ctx context.Context, event meergo.RawEvent, eventType string, schema types.Type, properties map[string]any, redacted bool) (*meergo.EventRequest, error) {

	req := &meergo.EventRequest{
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

	// Marshal the properties, if present.
	var propertiesJSON json.Value
	if schema.Valid() {
		var err error
		propertiesJSON, err = types.Marshal(properties, schema)
		if err != nil {
			return nil, err
		}
	}

	// Prepare the request's body.
	type requestEvent struct {
		Name   string     `json:"name"`
		Params json.Value `json:"params,omitempty"`
	}
	body := struct {
		ClientID        string         `json:"client_id"`
		UserID          string         `json:"user_id,omitempty"`
		TimestampMicros int64          `json:"timestamp_micros"`
		Events          []requestEvent `json:"events"`
	}{
		ClientID:        event.AnonymousId(),
		UserID:          event.UserId(),
		TimestampMicros: event.Timestamp().UnixMicro(),
		Events: []requestEvent{
			{Name: eventType, Params: propertiesJSON},
		},
	}
	var err error
	req.Body, err = json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// EventTypeSchema returns the schema of the specified event type.
func (ga *Analytics) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {
	event, ok := eventTypeByID[eventType]
	if ok {
		return event.Schema, nil
	}
	return types.Type{}, meergo.ErrEventTypeNotExist
}

// EventTypes returns the event types of the connector's instance.
func (ga *Analytics) EventTypes(ctx context.Context) ([]*meergo.EventType, error) {
	return meergoEventTypes, nil
}

// ServeUI serves the connector's user interface.
func (ga *Analytics) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if ga.settings != nil {
			s = *ga.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, ga.saveSettings(ctx, settings)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "MeasurementID", Label: "Measurement ID", Placeholder: "G-2XYZBEB6AB", Type: "text", MinLength: 2, MaxLength: 20, HelpText: "Follow these instructions to get your Measurement ID: https://support.google.com/analytics/answer/9539598#find-G-ID"},
			&meergo.Input{Name: "APISecret", Label: "API Secret", Placeholder: "ZuHCHFZbRBi8V7u8crWFUz", Type: "text", MinLength: 1, MaxLength: 40},
		},
		Settings: settings,
	}

	return ui, nil

}

// saveSettings saves the settings.
func (ga *Analytics) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	if n := len(s.MeasurementID); n < 2 || n > 20 {
		return meergo.NewInvalidSettingsError("Measurement ID length must be in [2,20]")
	}
	if !strings.HasPrefix(s.MeasurementID, "G-") && !strings.HasPrefix(s.MeasurementID, "AW-") {
		return meergo.NewInvalidSettingsError("Measurement ID must begin with 'G-' or 'AW-'")
	}
	if n := len(s.APISecret); n < 1 || n > 40 {
		return meergo.NewInvalidSettingsError("API secret length must be in [1,40]")
	}
	for i := 0; i < len(s.APISecret); i++ {
		c := s.APISecret[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9' || c == '-') {
			return meergo.NewInvalidSettingsError("API secret must contain only alphanumeric and '-' characters")
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
