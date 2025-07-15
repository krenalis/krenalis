//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package mixpanel implements the Mixpanel connector.
// (https://developer.mixpanel.com/reference/overview)
package mixpanel

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

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
		Name:       "Mixpanel",
		Categories: meergo.CategoryAnalytics,
		AsDestination: &meergo.AsAppDestination{
			Targets:     meergo.TargetEvent,
			HasSettings: true,
			SendingMode: meergo.Server,
			Documentation: meergo.ConnectorRoleDocumentation{
				Summary:  "Send events to Mixpanel",
				Overview: overview,
			},
		},
		EndpointGroups: []meergo.EndpointGroup{{
			// https://developer.mixpanel.com/reference/import-events
			RateLimit: meergo.RateLimit{RequestsPerSecond: 15, Burst: 20, MaxConcurrentRequests: 20},
			// https://mailchimp.com/developer/marketing/docs/fundamentals/#api-limits
			RetryPolicy: meergo.RetryPolicy{
				// https://developer.mixpanel.com/reference/import-events#rate-limits
				"429":     meergo.ExponentialStrategy(meergo.Slowdown, 2*time.Second),
				"502 503": meergo.ExponentialStrategy(meergo.NetFailure, 2*time.Second),
			}},
		},
		Icon: icon,
	}, New)
}

type Mixpanel struct {
	conf     *meergo.AppConfig
	settings *innerSettings
}

type innerSettings struct {
	ProjectID           string
	ProjectToken        string
	UseEuropeanEndpoint bool
}

// New returns a new Mixpanel connector instance.
func New(conf *meergo.AppConfig) (*Mixpanel, error) {
	c := Mixpanel{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Mixpanel connector")
		}
	}
	return &c, nil
}

// EventTypes returns the event types.
func (mp *Mixpanel) EventTypes(ctx context.Context) ([]*meergo.EventType, error) {
	return []*meergo.EventType{
		{
			ID:          "track",
			Name:        "Send track events",
			Description: "Send track events to Mixpanel",
		},
		{
			ID:          "page",
			Name:        "Send page events",
			Description: "Send page events to Mixpanel",
		},
		{
			ID:          "screen",
			Name:        "Send screen events",
			Description: "Send screen events to Mixpanel",
		},
	}, nil
}

// PreviewSendEvents returns the HTTP request that would be used to send the
// events to the app, without actually sending it.
func (mp *Mixpanel) PreviewSendEvents(ctx context.Context, events meergo.Events) (*http.Request, error) {
	return mp.sendEvents(ctx, events, true)
}

// SendEvents sends events to the app.
func (mp *Mixpanel) SendEvents(ctx context.Context, events meergo.Events) error {
	_, err := mp.sendEvents(ctx, events, false)
	return err
}

// EventTypeSchema returns the schema of the specified event type.
func (mp *Mixpanel) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {
	schema := func(placeholder string) types.Type {
		return types.Object([]types.Property{
			{Name: "event", Placeholder: placeholder, Type: types.Text().WithCharLen(255), CreateRequired: true, Description: "Event Name"},
			{Name: "properties", Type: types.Map(types.JSON()), CreateRequired: true, Description: "Your Properties"},
		})
	}
	switch eventType {
	case "track":
		return schema("event"), nil
	case "page":
		return schema(`"Page View"`), nil
	case "screen":
		return schema(`"Screen View"`), nil
	}
	return types.Type{}, meergo.ErrEventTypeNotExist
}

// ServeUI serves the connector's user interface.
func (mp *Mixpanel) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

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
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "ProjectID", Label: "Project ID", Placeholder: "1234567", Type: "text", MinLength: 1, MaxLength: 20},
			&meergo.Input{Name: "ProjectToken", Label: "Project Token", Placeholder: "d8e8fca2dc0f896fd7cb4cb0031ba249", Type: "text", MinLength: 32, MaxLength: 32},
			&meergo.Switch{Name: "UseEuropeanEndpoint", Label: "Use the European Endpoint"},
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
	if n, err := strconv.Atoi(s.ProjectID); err != nil || n < 0 {
		return meergo.NewInvalidSettingsError("Project ID must be a positive number")
	}
	if n := len(s.ProjectToken); n < 1 || n > 100 {
		return meergo.NewInvalidSettingsError("Project Token length must be in range [1,100]")
	}
	for i := 0; i < len(s.ProjectToken); i++ {
		c := s.ProjectToken[i]
		// ASCII characters with decimal codes from 33 (!) to 126 (~),
		// inclusive, are printable characters. The space character, having
		// decimal code 32, is therefore excluded from the range of accepted
		// characters, and this is intentional.
		if c < 33 || c > 126 {
			return meergo.NewInvalidSettingsError("Project Token must contain only valid characters")
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = mp.conf.SetSettings(ctx, b)
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

// sendEvents sends the given events to the app and returns the sent HTTP
// request.
// If preview is true, the HTTP request is built but not sent, so it is
// only returned.
//
// If an error occurs while sending the events to the app, a nil *http.Request
// and the error are returned.
func (mp *Mixpanel) sendEvents(ctx context.Context, events meergo.Events, preview bool) (*http.Request, error) {

	// body is a bytes.Buffer that contains newline-delimited JSON objects
	// representing the events to send to Mixpanel.
	var body bytes.Buffer

	n := 0
	for event := range events.All() {

		size := body.Len()

		if event.Type.Values["event"].(string) == "" {
			return nil, errors.New("event cannot be empty")
		}

		properties := event.Type.Values["properties"].(map[string]any)
		properties["$insert_id"] = event.Received.MessageId()
		properties["time"] = event.Received.Timestamp().UnixMilli()
		if sendBadRequest, _ := ctx.Value(connectorTestString("sendBadRequest")).(bool); sendBadRequest {
			delete(properties, "time")
		}
		distinctID := event.Received.AnonymousId()
		if userId, ok := event.Received.UserId(); ok {
			distinctID = userId
		}
		properties["distinct_id"] = distinctID
		properties["$device_id"] = event.Received.AnonymousId()
		if context, ok := event.Received.Context(); ok {
			if ip, ok := context.IP(); ok {
				properties["ip"] = ip
				// Supplying the 'ip' property, Mixpanel automatically enriches the event with country, region, and city
				// if they are not supplied. Provide either all or none of these properties to ensure that Mixpanel's
				// enrichment occurs for all or none of them.
				if location, ok := context.Location(); ok {
					country, hasCountry := location.Country()
					city, hasCity := location.City()
					if hasCountry || hasCity {
						if hasCountry {
							properties["mp_country_code"] = country
						} else {
							properties["mp_country_code"] = nil
						}
						if hasCity {
							properties["$city"] = city
						} else {
							properties["$city"] = nil
						}
					}
				} else {
					if location, ok := context.Location(); ok {
						if country, ok := location.Country(); ok {
							properties["mp_country_code"] = country
						}
						if city, ok := location.City(); ok {
							properties["$city"] = city
						}
					}
				}
			}
			if os, ok := context.OS(); ok {
				if osName, ok := os.Name(); ok {
					properties["$os"] = osName
				}
			}
			if browser, ok := context.Browser(); ok {
				if name, ok := browser.Name(); ok {
					properties["$browser"] = name
				} else if other, ok := browser.Other(); ok {
					properties["$browser"] = other
				} else if version, ok := browser.Version(); ok {
					properties["$browser"] = version
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
			}
		}

		err := json.Encode(&body, map[string]any{
			"event":      event.Type.Values["event"].(string),
			"properties": properties,
		})
		if err != nil {
			return nil, err
		}

		body.WriteByte('\n')

		if body.Len() > maxBodyEventsBytes {
			body.Truncate(size)
			events.Postpone()
			break
		}

		n++
		if n == maxEventsPerRequest {
			break
		}
	}

	u := "https://api.mixpanel.com/"
	if mp.settings.UseEuropeanEndpoint {
		u = "https://api-eu.mixpanel.com/"
	}
	u += "import?strict=1&project_id=" + mp.settings.ProjectID

	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(body.Bytes()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")

	if preview {
		req.Header.Set("Authorization", "Basic [REDACTED]")
	} else {
		req.SetBasicAuth(mp.settings.ProjectToken, "")
	}

	// Mark the request as idempotent.
	req.Header["Idempotency-Key"] = nil
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body.Bytes())), nil
	}

	storeHTTPRequestWhenTesting(ctx, req)

	if preview {
		return req, nil
	}

	// Send the request.
	res, err := mp.conf.HTTPClient.Do(req)
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
		return nil, errors.New("unexpected status 400 with empty 'failed_records' in response from Mixpanel")
	}
	errors := make(meergo.EventsError, len(out.FailedRecords))
	for _, f := range out.FailedRecords {
		errors[f.Index] = fmt.Errorf("sending event %q: %s", f.Field, f.Message)
	}

	return nil, errors
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
