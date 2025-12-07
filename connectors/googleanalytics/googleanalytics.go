// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package googleanalytics provides a connector for Google Analytics.
// (https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference)
//
// Google and Google Analytics are trademarks of Google LLC.
// This connector is not affiliated with or endorsed by Google LLC.
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

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

//go:embed documentation/overview.md
var overview string

func init() {
	connectors.RegisterAPI(connectors.APISpec{
		Code:       "google-analytics",
		Label:      "Google Analytics",
		Categories: connectors.CategorySaaS,
		AsDestination: &connectors.AsAPIDestination{
			Targets:     connectors.TargetEvent,
			HasSettings: true,
			SendingMode: connectors.Server,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Send events to Google Analytics",
				Overview: overview,
			},
		},
		EndpointGroups: []connectors.EndpointGroup{{
			// https://developers.google.com/analytics/devguides/limits-and-quotas
			RateLimit: connectors.RateLimit{RequestsPerSecond: 11, Burst: 110},
		}},
	}, New)
}

// New returns a new connector instance for Google Analytics.
func New(env *connectors.APIEnv) (*Analytics, error) {
	c := Analytics{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for Google Analytics")
		}
	}
	return &c, nil
}

type Analytics struct {
	env      *connectors.APIEnv
	settings *innerSettings
}

type innerSettings struct {
	MeasurementID      string
	APISecret          string
	CollectionEndpoint string
}

// EventTypes returns the event types.
func (ga *Analytics) EventTypes(ctx context.Context) ([]*connectors.EventType, error) {
	return meergoEventTypes, nil
}

// EventTypeSchema returns the schema of the specified event type.
func (ga *Analytics) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {
	event, ok := eventTypeByID[eventType]
	if ok {
		return event.Schema, nil
	}
	return types.Type{}, connectors.ErrEventTypeNotExist
}

// PreviewSendEvents returns the HTTP request that would be used to send the
// events to the API, without actually sending it.
func (ga *Analytics) PreviewSendEvents(ctx context.Context, events connectors.Events) (*http.Request, error) {
	return ga.sendEvents(ctx, events, true)
}

// SendEvents sends events to the API.
func (ga *Analytics) SendEvents(ctx context.Context, events connectors.Events) error {
	_, err := ga.sendEvents(ctx, events, false)
	return err
}

// ServeUI serves the connector's user interface.
func (ga *Analytics) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if ga.settings == nil {
			s.CollectionEndpoint = "Global"
		} else {
			s = *ga.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, ga.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "MeasurementID", Label: "Measurement ID", Placeholder: "G-2XYZBEB6AB", Type: "text", MinLength: 2, MaxLength: 20, HelpText: "Follow these instructions to get your Measurement ID: https://support.google.com/analytics/answer/9539598#find-G-ID"},
			&connectors.Input{Name: "APISecret", Label: "API secret", Placeholder: "ZuHCHFZbRBi8V7u8crWFUz", Type: "text", MinLength: 1, MaxLength: 40},
			&connectors.Select{Name: "CollectionEndpoint", Label: "Collection endpoint", Options: []connectors.Option{{Text: "Global", Value: "Global"}, {Text: "European Union", Value: "EU"}}},
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
	// Validate MeasurementID.
	if n := len(s.MeasurementID); n < 2 || n > 20 {
		return connectors.NewInvalidSettingsError("Measurement ID length must be in [2,20]")
	}
	if !strings.HasPrefix(s.MeasurementID, "G-") && !strings.HasPrefix(s.MeasurementID, "AW-") {
		return connectors.NewInvalidSettingsError("Measurement ID must begin with 'G-' or 'AW-'")
	}
	// Validate APISecret.
	if n := len(s.APISecret); n < 1 || n > 100 {
		return connectors.NewInvalidSettingsError("API secret length must be in [1,100]")
	}
	for i := 0; i < len(s.APISecret); i++ {
		c := s.APISecret[i]
		// ASCII characters with decimal codes from 33 (!) to 126 (~),
		// inclusive, are printable characters. The space character, having
		// decimal code 32, is therefore excluded from the range of accepted
		// characters, and this is intentional.
		if c < 33 || c > 126 {
			return connectors.NewInvalidSettingsError("API secret must contain only valid characters")
		}
	}
	// Validate CollectionEndpoint.
	switch s.CollectionEndpoint {
	case "Global", "EU":
	default:
		return connectors.NewInvalidSettingsError("collection endpoint must be set to Global or EU")
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ga.env.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	ga.settings = &s
	return nil
}

const maxEventRequestSize = 130 * 1024 // from https://developers.google.com/analytics/devguides/collection/protocol/ga4/sending-events?client_type=gtag#limitations.

// sendEvents sends the given events to the API and returns the sent HTTP
// request.
// If preview is true, the HTTP request is built but not sent, so it is
// only returned.
//
// If an error occurs while sending the events to the API, a nil *http.Request
// and the error are returned.
func (ga *Analytics) sendEvents(ctx context.Context, events connectors.Events, preview bool) (*http.Request, error) {

	bb := ga.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()

	var userID string

	n := 0
	for event := range events.SameUser() {
		if n == 0 {
			// Note that 'userID' may also be empty after this assignment, which
			// means that only events with empty 'userID' will be sent in this
			// batch.
			userID, _ = event.Received.UserID()
			bb.WriteByte('{')
			bb.EncodeKeyValue("client_id", event.Received.AnonymousId())
			if userID != "" {
				bb.EncodeKeyValue("user_id", userID)
			}
			bb.WriteString(`,"events":[`)
		} else {
			if uId, _ := event.Received.UserID(); uId != userID {
				events.Postpone()
				continue
			}
			bb.WriteByte(',')
		}
		bb.WriteByte('{')
		bb.EncodeKeyValue("name", event.Type.ID)
		if event.Type.Values != nil {
			params, err := types.Marshal(event.Type.Values, event.Type.Schema)
			if err != nil {
				return nil, err
			}
			bb.EncodeKeyValue("params", params)
		}
		bb.EncodeKeyValue("timestamp_micros", event.Received.Timestamp().UnixMicro())
		bb.WriteByte('}')

		if bb.Len()+len("]}") > maxEventRequestSize {
			// From the Google Analytics documentation: «The post body must be smaller than 130kB.»
			// https://developers.google.com/analytics/devguides/collection/protocol/ga4/sending-events?client_type=gtag#limitations.
			bb.Truncate(0)
			events.Postpone()
			break
		}

		if err := bb.Flush(); err != nil {
			return nil, err
		}

		n++
		if n == 25 {
			// From the Google Analytics documentation: «Requests can have a maximum of 25 events.»
			// https://developers.google.com/analytics/devguides/collection/protocol/ga4/sending-events?client_type=gtag#limitations.
			break
		}
	}
	bb.WriteString("]}")

	if preview {

		// First, it performs an actual send to the Google Analytics debug
		// server to validate the request, returning an error in case of
		// validation issues.
		u := requestURL(ga.settings.CollectionEndpoint, ga.settings.APISecret, true, false, ga.settings.MeasurementID)
		req, err := bb.NewRequest(ctx, "POST", u)
		if err != nil {
			return nil, err
		}
		req.Header["Idempotency-Key"] = nil // mark the request as idempotent
		// Copy the body to reuse it.
		body, _ := io.ReadAll(req.Body)
		req.Body, _ = req.GetBody()
		// Do the request.
		resp, err := ga.env.HTTPClient.Do(req)
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
		u = requestURL(ga.settings.CollectionEndpoint, ga.settings.APISecret, true, true, ga.settings.MeasurementID)
		req, err = http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		return req, nil
	}

	// Build the request to send to Google Analytics.
	u := requestURL(ga.settings.CollectionEndpoint, ga.settings.APISecret, false, false, ga.settings.MeasurementID)
	req, err := bb.NewRequest(ctx, "POST", u)
	if err != nil {
		return nil, err
	}
	req.Header["Idempotency-Key"] = nil // mark the request as idempotent

	if err = storeHTTPRequestWhenTesting(ctx, req); err != nil {
		return nil, err
	}

	res, err := ga.env.HTTPClient.Do(req)
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
func requestURL(collectionEndpoint, apiSecret string, toDebugServer bool, redactSensitiveInfo bool, measurementID string) string {
	u := "https://"
	switch collectionEndpoint {
	case "Global":
		u += "www"
	case "EU":
		u += "region1"
	}
	u += ".google-analytics.com/"
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
func storeHTTPRequestWhenTesting(ctx context.Context, req *http.Request) error {
	stored := ctx.Value(connectorTestString("storeSentHTTPRequest"))
	if stored == nil {
		return nil
	}
	storedReq := req.Clone(req.Context())
	body, err := req.GetBody()
	if err != nil {
		return err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	storedReq.Body = io.NopCloser(bytes.NewReader(data))
	*(stored.(*http.Request)) = *storedReq
	return nil
}
