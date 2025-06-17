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
	"bytes"
	"context"
	_ "embed"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

//go:embed documentation/overview.md
var overview string

// sendToDebugServer controls whether the events should be sent to the debug
// server instead of the production server.
//
// See
// https://developers.google.com/analytics/devguides/collection/protocol/ga4/validating-events?client_type=firebase
const sendToDebugServer = false

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name:       "Google Analytics",
		Categories: meergo.CategoryAnalytics,
		AsDestination: &meergo.AsAppDestination{
			Targets:     meergo.TargetEvent,
			HasSettings: true,
			SendingMode: meergo.Cloud,
			Documentation: meergo.ConnectorRoleDocumentation{
				Summary:  "Send events to Google Analytics",
				Overview: overview,
			},
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

// EventTypes returns the event types.
func (ga *Analytics) EventTypes(ctx context.Context) ([]*meergo.EventType, error) {
	return meergoEventTypes, nil
}

// EventTypeSchema returns the schema of the specified event type.
func (ga *Analytics) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {
	event, ok := eventTypeByID[eventType]
	if ok {
		return event.Schema, nil
	}
	return types.Type{}, meergo.ErrEventTypeNotExist
}

// PreviewSendEvents returns the HTTP request that would be used to send the
// events to the app, without actually sending it.
func (ga *Analytics) PreviewSendEvents(ctx context.Context, events meergo.Events) (*http.Request, error) {
	return ga.sendEvents(ctx, events, true)
}

// SendEvents sends events to the app.
func (ga *Analytics) SendEvents(ctx context.Context, events meergo.Events) error {
	_, err := ga.sendEvents(ctx, events, false)
	return err
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

const maxEventRequestSize = 130 * 1024 // from https://developers.google.com/analytics/devguides/collection/protocol/ga4/sending-events?client_type=gtag#limitations.

// sendEvents sends the given events to the app, returning the HTTP request and
// any error in sending the request or in the app server's response. If preview
// is true, the HTTP request is built but not sent, so it is only returned.
func (ga *Analytics) sendEvents(ctx context.Context, events meergo.Events, preview bool) (*http.Request, error) {

	u := "https://www.google-analytics.com/"
	if sendToDebugServer {
		u += "debug/"
	}

	secret := ga.settings.APISecret
	if preview {
		secret = "REDACTED"
	}
	u += "mp/collect?api_secret=" + url.QueryEscape(secret) + "&measurement_id=" + url.QueryEscape(ga.settings.MeasurementID)

	var eventsWriter json.Buffer

	eventsPerRequest := 0
	var userID, anonymousId string
	for i, event := range events.SameUser() {
		if i == 0 {
			// Note that 'userID' may also be empty after this assignment, which
			// means that only events with empty 'userID' will be sent in this
			// batch.
			userID = event.Raw.UserId()
			anonymousId = event.Raw.AnonymousId()
		} else {
			if event.Raw.UserId() != userID {
				events.Skip()
				continue
			}
			eventsWriter.WriteByte(',')
		}
		eventsWriter.WriteByte('{')
		eventsWriter.EncodeKeyValue("name", event.Type)
		if event.Schema.Valid() {
			params, err := types.Marshal(event.Properties, event.Schema)
			if err != nil {
				return nil, err
			}
			eventsWriter.EncodeKeyValue("params", params)
		}
		eventsWriter.EncodeKeyValue("timestamp_micros", event.Raw.Timestamp().UnixMicro())
		eventsWriter.WriteByte('}')
		eventsPerRequest++
		if eventsPerRequest == 25 {
			// From the Google Analytics documentation: «Requests can have a maximum of 25 events.»
			// https://developers.google.com/analytics/devguides/collection/protocol/ga4/sending-events?client_type=gtag#limitations.
			break
		}
		if eventsWriter.Len()+300 > maxEventRequestSize {
			// From the Google Analytics documentation: «The post body must be smaller than 130kB.»
			// https://developers.google.com/analytics/devguides/collection/protocol/ga4/sending-events?client_type=gtag#limitations.
			//
			// 300 is just a margin value that takes into account the rest of
			// the body, which will be built later, the JSON Object that
			// encloses "events" (so the various keys "client_id", "user_id",
			// etc.).
			break
		}
	}

	// Build the actual body.
	var body json.Buffer
	body.WriteByte('{')
	body.EncodeKeyValue("client_id", anonymousId)
	if userID != "" {
		body.EncodeKeyValue("user_id", userID)
	}
	body.WriteString(`,"events":[`)
	body.Write(eventsWriter.Bytes())
	body.WriteString("]}")

	req, err := http.NewRequest("POST", u, bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if preview {
		return req, nil
	}

	_, err = ga.conf.HTTPClient.DoIdempotent(req, true)
	if err != nil {
		return req, err
	}

	// TODO: handle errors

	return req, nil
}
