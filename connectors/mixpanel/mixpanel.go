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
	"encoding/base64"
	"errors"
	"fmt"
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
			SendingMode: meergo.Cloud,
			Documentation: meergo.ConnectorRoleDocumentation{
				Summary:  "Send events to Mixpanel",
				Overview: overview,
			},
		},
		Icon: icon,
		BackoffPolicy: meergo.BackoffPolicy{
			// https://developer.mixpanel.com/reference/import-events#rate-limits
			"429 502 503": meergo.ExponentialStrategy(2 * time.Second),
		},
	}, New)
}

type Mixpanel struct {
	conf     *meergo.AppConfig
	settings *innerSettings
}

type innerSettings struct {
	ProjectID           string
	Username            string
	Secret              string
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
			&meergo.Input{Name: "Username", Label: "Service Account Username", Placeholder: "youraccount.82us7b.mp-service-account", Type: "text", MinLength: 20, MaxLength: 100},
			&meergo.Input{Name: "Secret", Label: "Service Account Secret", Placeholder: "OfCknZXmL1shKB7qhxdpvkwqQYwn4PQr", Type: "text", MinLength: 32, MaxLength: 100},
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
		return meergo.NewInvalidSettingsError("project ID must be a positive number")
	}
	if n := len(s.Username); n < 20 || n > 100 {
		return meergo.NewInvalidSettingsError("username length must be in range [20, 100]")
	}
	// TODO(Gianluca): this validation could be improved and/or standardized
	// with the others across connectors, but it's not worth the effort since
	// the project token will likely be deprecated and/or removed in future
	// commits in favor of more modern authentication methods.
	if n := len(s.Secret); n < 32 || n > 100 {
		return meergo.NewInvalidSettingsError("secret length must be in range [32, 100]")
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

// sendEvents sends the given events to the app, returning the HTTP request and
// any error in sending the request or in the app server's response. If preview
// is true, the HTTP request is built but not sent, so it is only returned.
func (mp *Mixpanel) sendEvents(ctx context.Context, events meergo.Events, preview bool) (*http.Request, error) {

	// TODO(Gianluca): handle the limits imposed by Mixpanel, see
	// https://developer.mixpanel.com/reference/import-events.

	// nlDelimitedEvents is a bytes.Buffer that contains newline-delimited JSON
	// objects representing the events to send to Mixpanel.
	var nlDelimitedEvents bytes.Buffer
	for _, event := range events.All() {

		if event.Properties["event"].(string) == "" {
			return nil, errors.New("event cannot be empty")
		}

		body := event.Properties["properties"].(map[string]any)
		body["$insert_id"] = event.Raw.MessageId()
		body["time"] = formatTimestamp(event.Raw.Timestamp())
		distinctID := event.Raw.AnonymousId()
		if userId := event.Raw.UserId(); userId != "" {
			distinctID = userId
		}
		body["distinct_id"] = distinctID
		body["$device_id"] = event.Raw.AnonymousId()
		context := event.Raw.Context()
		if ip := context.IP(); ip == "" {
			if location, ok := context.Location(); ok {
				if country := location.Country(); country != "" {
					body["mp_country_code"] = country
				}
				if city := location.City(); city != "" {
					body["$city"] = city
				}
			}
		} else {
			body["ip"] = context.IP()
			// Supplying the 'ip' property, Mixpanel automatically enriches the event with country, region, and city
			// if they are not supplied. Provide either all or none of these properties to ensure that Mixpanel's
			// enrichment occurs for all or none of them.
			if location, ok := context.Location(); ok {
				country := location.Country()
				city := location.City()
				if country != "" || city != "" {
					if country != "" {
						body["mp_country_code"] = country
					} else {
						body["mp_country_code"] = nil
					}
					if city != "" {
						body["$city"] = city
					} else {
						body["$city"] = nil
					}
				}
			}
		}
		if os, ok := context.OS(); ok && os.Name() != "" {
			body["$os"] = os.Name()
		}
		if browser, ok := context.Browser(); ok {
			if browser.Name() != "" {
				body["$browser"] = browser.Name()
			} else if browser.Other() != "" {
				body["$browser"] = browser.Other()
			}
			if browser.Version() != "" {
				body["$browser_version"] = browser.Version()
			}
		}
		if page, ok := context.Page(); ok {
			if referrer := page.Referrer(); referrer != "" {
				u, err := url.Parse(referrer)
				if err == nil {
					body["$referrer"] = referrer
					body["$referring_domain"] = u.Hostname()
				}
			}
			if pageURL := page.URL(); pageURL != "" {
				body["$current_url"] = pageURL
				body["current_page_title"] = page.Title()
				u, err := url.Parse(pageURL)
				if err == nil {
					body["current_domain"] = u.Hostname()
					body["current_url_path"] = u.Path
					body["current_url_protocol"] = u.Scheme + ":"
				}
			}
		}
		if screen, ok := context.Screen(); ok {
			if w := screen.Width(); w != 0 {
				body["$screen_width"] = w
			}
			if h := screen.Height(); h != 0 {
				body["$screen_height"] = h
			}
		}

		err := json.Encode(&nlDelimitedEvents, body)
		if err != nil {
			return nil, err
		}

		nlDelimitedEvents.WriteByte('\n')
	}

	u := "https://api.mixpanel.com/"
	if mp.settings.UseEuropeanEndpoint {
		u = "https://api-eu.mixpanel.com/"
	}
	u += "import?strict=1&project_id=" + mp.settings.ProjectID

	req, err := http.NewRequest("POST", u, &nlDelimitedEvents)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	authorization := base64.StdEncoding.EncodeToString([]byte(mp.settings.Username + ":" + mp.settings.Secret))
	if preview {
		authorization = "[REDACTED]"
	}
	req.Header.Set("Authorization", "Basic "+authorization)

	if preview {
		return req, nil
	}

	// Send the request.
	res, err := mp.conf.HTTPClient.DoIdempotent(req, true)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == 200 {
		return req, nil
	}

	// Handle the error.
	if res.StatusCode != 400 {
		return req, fmt.Errorf("Mixpanel server responded with %d error code", res.StatusCode)
	}
	var out struct {
		FailedRecords []struct {
			Index    int    `json:"index"`
			InsertId string `json:"insert_id"`
			Field    string `json:"field"`
			Message  string `json:"message"`
		} `json:"failed_records"`
	}
	err = json.Decode(res.Body, out)
	if err != nil {
		return req, fmt.Errorf("cannot decode Mixpanel response: %v", err)
	}
	if len(out.FailedRecords) == 0 {
		return req, errors.New("unexpected status 400 with empty 'failed_records' in response from Mixpanel")
	}
	errors := make(meergo.EventsError, len(out.FailedRecords))
	for _, f := range out.FailedRecords {
		errors[f.Index] = fmt.Errorf("sending event %q: %s", f.Field, f.Message)
	}

	return req, errors
}

// formatTimestamp formats the timestamp t of an event as expected by Mixpanel.
func formatTimestamp(t time.Time) string {
	ms := strconv.FormatInt(t.UnixMilli(), 10)
	l := len(ms)
	if l <= 3 {
		return "0." + ms
	}
	return ms[:l-3] + "." + ms[l-3:]
}
