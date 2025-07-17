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
	"fmt"
	"io"
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

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name:       "Google Analytics",
		Categories: meergo.CategoryAnalytics,
		AsDestination: &meergo.AsAppDestination{
			Targets:     meergo.TargetEvent,
			HasSettings: true,
			SendingMode: meergo.Server,
			Documentation: meergo.ConnectorRoleDocumentation{
				Summary:  "Send events to Google Analytics",
				Overview: overview,
			},
		},
		EndpointGroups: []meergo.EndpointGroup{{
			// https://developers.hubspot.com/docs/guides/apps/api-usage/usage-details#public-apps
			RateLimit: meergo.RateLimit{RequestsPerSecond: 11, Burst: 110},
		}},
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
	if n := len(s.APISecret); n < 1 || n > 100 {
		return meergo.NewInvalidSettingsError("API secret length must be in [1,100]")
	}
	for i := 0; i < len(s.APISecret); i++ {
		c := s.APISecret[i]
		// ASCII characters with decimal codes from 33 (!) to 126 (~),
		// inclusive, are printable characters. The space character, having
		// decimal code 32, is therefore excluded from the range of accepted
		// characters, and this is intentional.
		if c < 33 || c > 126 {
			return meergo.NewInvalidSettingsError("API secret must contain only valid characters")
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

// sendEvents sends the given events to the app and returns the sent HTTP
// request.
// If preview is true, the HTTP request is built but not sent, so it is
// only returned.
//
// If an error occurs while sending the events to the app, a nil *http.Request
// and the error are returned.
func (ga *Analytics) sendEvents(ctx context.Context, events meergo.Events, preview bool) (*http.Request, error) {

	var eventsWriter json.Buffer
	var userID, anonymousId string

	n := 0
	for event := range events.SameUser() {
		if n == 0 {
			// Note that 'userID' may also be empty after this assignment, which
			// means that only events with empty 'userID' will be sent in this
			// batch.
			userID, _ = event.Received.UserId()
			anonymousId = event.Received.AnonymousId()
		} else {
			if uId, _ := event.Received.UserId(); uId != userID {
				events.Postpone()
				continue
			}
			eventsWriter.WriteByte(',')
		}
		size := eventsWriter.Len()
		eventsWriter.WriteByte('{')
		eventsWriter.EncodeKeyValue("name", event.Type.ID)
		if event.Type.Values != nil {
			params, err := types.Marshal(event.Type.Values, event.Type.Schema)
			if err != nil {
				return nil, err
			}
			eventsWriter.EncodeKeyValue("params", params)
		}
		eventsWriter.EncodeKeyValue("timestamp_micros", event.Received.Timestamp().UnixMicro())
		eventsWriter.WriteByte('}')

		if eventsWriter.Len()+300 > maxEventRequestSize {
			// From the Google Analytics documentation: «The post body must be smaller than 130kB.»
			// https://developers.google.com/analytics/devguides/collection/protocol/ga4/sending-events?client_type=gtag#limitations.
			//
			// 300 is just a margin value that takes into account the rest of
			// the body, which will be built later, the JSON Object that
			// encloses "events" (so the various keys "client_id", "user_id",
			// etc.).
			eventsWriter.Truncate(size)
			events.Postpone()
			break
		}

		n++
		if n == 25 {
			// From the Google Analytics documentation: «Requests can have a maximum of 25 events.»
			// https://developers.google.com/analytics/devguides/collection/protocol/ga4/sending-events?client_type=gtag#limitations.
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

	if preview {
		// First, it performs an actual send to the Google Analytics debug
		// server to validate the request, returning an error in case of
		// validation issues.
		u := requestURL(ga.settings.APISecret, true, false, ga.settings.MeasurementID)
		req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(body.Bytes()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		// Mark the request as idempotent.
		req.Header["Idempotency-Key"] = nil
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body.Bytes())), nil
		}
		// Do the request.
		resp, err := ga.conf.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		var validationResponse struct {
			ValidationMessages []json.Value `json:"validationMessages"`
		}
		err = json.NewDecoder(resp.Body).Decode(&validationResponse)
		if err != nil {
			return nil, err
		}
		if len(validationResponse.ValidationMessages) > 0 {
			msg := "the Google Analytics debug server has returned validation messages:\n"
			for _, m := range validationResponse.ValidationMessages {
				msg += string(m)
			}
			return nil, errors.New(msg)
		}
		// Next, build a new request to be returned to Meergo, in which
		// sensitive information (such as the API secret) is redacted.
		u = requestURL(ga.settings.APISecret, true, true, ga.settings.MeasurementID)
		req, err = http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(body.Bytes()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}

	// Build the request to send to Google Analytics.
	u := requestURL(ga.settings.APISecret, false, false, ga.settings.MeasurementID)
	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	// Mark the request as idempotent.
	req.Header["Idempotency-Key"] = nil
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body.Bytes())), nil
	}

	storeHTTPRequestWhenTesting(ctx, req)

	res, err := ga.conf.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 204 {
		return nil, fmt.Errorf("Google Analytics responded with error code %d", res.StatusCode)
	}

	return req, nil
}

// requestURL builds and returns the request URL to which the events request
// should be sent, based on the given parameters.
func requestURL(apiSecret string, toDebugServer bool, redactSensitiveInfo bool, measurementID string) string {
	u := "https://www.google-analytics.com/"
	if toDebugServer {
		u += "debug/"
	}
	if redactSensitiveInfo {
		apiSecret = "REDACTED"
	}
	u += "mp/collect?api_secret=" + url.QueryEscape(apiSecret) + "&measurement_id=" + url.QueryEscape(measurementID)
	return u
}

// connectorTestString is a defined type used solely to pass a typed key to the
// context. This is to avoid any kind of collisions with other values that may
// be inserted into the context.
type connectorTestString string

// storeHTTPRequestWhenTesting stores the HTTP request if requested by tests.
//
// To enable saving the HTTP request, the context must include, under the key
// 'connectorTestString("storeSentHTTPRequest")', a pointer to an http.Request
// memory location where the sent request will be written.
func storeHTTPRequestWhenTesting(ctx context.Context, req *http.Request) {
	if stored := ctx.Value(connectorTestString("storeSentHTTPRequest")); stored != nil {
		clonedReq := req.Clone(req.Context())
		bodyBytes, _ := io.ReadAll(req.Body)
		clonedReq.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		*(stored.(*http.Request)) = *clonedReq
	}
}
