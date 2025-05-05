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
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
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

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name: "Mixpanel",
		AsDestination: &meergo.AsAppDestination{
			Description: "Send events to Mixpanel",
			Targets:     meergo.EventsTarget,
			HasSettings: true,
			SendingMode: meergo.Cloud,
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

// EventRequest returns a request to dispatch an event to the app.
func (mp *Mixpanel) EventRequest(ctx context.Context, event meergo.Event, eventType string, schema types.Type, properties map[string]any, redacted bool) (*meergo.EventRequest, error) {

	if properties["event"].(string) == "" {
		return nil, errors.New("event cannot be empty")
	}

	req := &meergo.EventRequest{
		Endpoint: "api",
		Method:   "POST",
		URL:      "https://api.mixpanel.com/",
		Header:   http.Header{},
	}
	if mp.settings.UseEuropeanEndpoint {
		req.Endpoint = "api-eu"
		req.URL = "https://api-eu.mixpanel.com/"
	}
	req.URL += "import?strict=0&project_id=" + mp.settings.ProjectID
	req.Header.Set("Content-Type", "application/x-ndjson")
	authorization := base64.StdEncoding.EncodeToString([]byte(mp.settings.Username + ":" + mp.settings.Secret))
	if redacted {
		authorization = "[REDACTED]"
	}
	req.Header.Set("Authorization", authorization)

	body := properties["properties"].(map[string]any)
	body["$insert_id"] = event.MessageId
	body["time"] = formatTimestamp(event.Timestamp())
	distinctID := event.AnonymousId()
	if userId := event.UserId(); userId != "" {
		distinctID = userId
	}
	body["distinct_id"] = distinctID
	body["$device_id"] = event.AnonymousId
	context := event.Context()
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

	var err error
	req.Body, err = json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// EventTypes returns the event types of the connector's instance.
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

// Schema returns the schema of the specified target in the specified role.
func (mp *Mixpanel) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {
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

// formatTimestamp formats the timestamp t of an event as expected by Mixpanel.
func formatTimestamp(t time.Time) string {
	ms := strconv.FormatInt(t.UnixMilli(), 10)
	l := len(ms)
	if l <= 3 {
		return "0." + ms
	}
	return ms[:l-3] + "." + ms[l-3:]
}
