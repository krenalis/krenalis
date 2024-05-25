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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the App, AppEvents and UIHandler interfaces.
var _ interface {
	chichi.App
	chichi.AppEvents
	chichi.UIHandler
} = (*Mixpanel)(nil)

func init() {
	chichi.RegisterApp(chichi.AppInfo{
		Name:                   "Mixpanel",
		Targets:                chichi.Events,
		DestinationDescription: "send events to Mixpanel",
		Icon:                   icon,
		SendingMode:            chichi.Cloud,
	}, New)
}

type Mixpanel struct {
	conf     *chichi.AppConfig
	settings *Settings
}

type Settings struct {
	ProjectID string
	Username  string
	Secret    string
}

// New returns a new Mixpanel connector instance.
func New(conf *chichi.AppConfig) (*Mixpanel, error) {
	c := Mixpanel{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Mixpanel connector")
		}
	}
	return &c, nil
}

// EventRequest returns a request to dispatch an event to the app.
func (mp *Mixpanel) EventRequest(ctx context.Context, typ string, event *chichi.Event, extra map[string]any, schema types.Type, redacted bool) (*chichi.EventRequest, error) {

	if extra["event"].(string) == "" {
		return nil, errors.New("event cannot be empty")
	}

	req := &chichi.EventRequest{
		Endpoint: "api",
		Method:   "POST",
		URL:      "https://api.mixpanel.com/",
		Header:   http.Header{},
	}
	if mp.conf.Region == chichi.PrivacyRegionEurope {
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

	body := extra["properties"].(map[string]any)
	body["$insert_id"] = event.MessageId
	body["time"] = formatTimestamp(event.Timestamp)
	distinctID := event.AnonymousId
	if event.UserId != "" {
		distinctID = event.UserId
	}
	body["distinct_id"] = distinctID
	body["$device_id"] = event.AnonymousId
	if event.Context.IP == "" {
		if event.Context.Location.Country != "" {
			body["mp_country_code"] = event.Context.Location.Country
		}
		if event.Context.Location.City != "" {
			body["$city"] = event.Context.Location.City
		}
	} else {
		body["ip"] = event.Context.IP
		// Supplying the 'ip' property, Mixpanel automatically enriches the event with country, region, and city
		// if they are not supplied. Provide either all or none of these properties to ensure that Mixpanel's
		// enrichment occurs for all or none of them.
		if event.Context.Location.Country != "" || event.Context.Location.City != "" {
			if event.Context.Location.Country != "" {
				body["mp_country_code"] = event.Context.Location.Country
			} else {
				body["mp_country_code"] = nil
			}
			if event.Context.Location.City != "" {
				body["$city"] = event.Context.Location.City
			} else {
				body["$city"] = nil
			}
		}
	}
	if event.Context.OS.Name != "" {
		body["$os"] = event.Context.OS.Name
	}
	if event.Context.Browser.Name != "" {
		body["$browser"] = event.Context.Browser.Name
	} else if event.Context.Browser.Other != "" {
		body["$browser"] = event.Context.Browser.Other
	}
	if event.Context.Browser.Version != "" {
		body["$browser_version"] = event.Context.Browser.Version
	}
	if event.Context.Page.Referrer != "" {
		u, err := url.Parse(event.Context.Page.Referrer)
		if err == nil {
			body["$referrer"] = event.Context.Page.Referrer
			body["$referring_domain"] = u.Hostname()
		}
	}
	if event.Context.Page.URL != "" {
		body["$current_url"] = event.Context.Page.URL
		body["current_page_title"] = event.Context.Page.Title
		u, err := url.Parse(event.Context.Page.URL)
		if err == nil {
			body["current_domain"] = u.Hostname()
			body["current_url_path"] = u.Path
			body["current_url_protocol"] = u.Scheme + ":"
		}
	}
	if event.Context.Screen.Width != 0 {
		body["$screen_width"] = event.Context.Screen.Width
	}
	if event.Context.Screen.Height != 0 {
		body["$screen_height"] = event.Context.Screen.Height
	}

	var err error
	req.Body, err = json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// EventTypes returns the event types of the connector's instance.
func (mp *Mixpanel) EventTypes(ctx context.Context) ([]*chichi.EventType, error) {
	return []*chichi.EventType{
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

// Schema returns the schema of the specified target.
func (mp *Mixpanel) Schema(ctx context.Context, target chichi.Targets, role chichi.Role, eventType string) (types.Type, error) {
	schema := func(placeholder string) types.Type {
		return types.Object([]types.Property{
			{Name: "event", Label: "Event Name", Placeholder: placeholder, Type: types.Text().WithCharLen(255), Required: true},
			{Name: "properties", Label: "Your Properties", Type: types.Map(types.JSON()), Required: true},
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
	return types.Type{}, chichi.ErrEventTypeNotExist
}

// ServeUI serves the connector's user interface.
func (mp *Mixpanel) ServeUI(ctx context.Context, event string, values []byte, role chichi.Role) (*chichi.UI, error) {

	switch event {
	case "load":
		var s Settings
		if mp.settings != nil {
			s = *mp.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, mp.saveValues(ctx, values)
	default:
		return nil, chichi.ErrUIEventNotExist
	}

	ui := &chichi.UI{
		Fields: []chichi.Component{
			&chichi.Input{Name: "ProjectID", Label: "Project ID", Placeholder: "1234567", Type: "text", MinLength: 1, MaxLength: 20},
			&chichi.Input{Name: "Username", Label: "Service Account Username", Placeholder: "youraccount.82us7b.mp-service-account", Type: "text", MinLength: 20, MaxLength: 100},
			&chichi.Input{Name: "Secret", Label: "Service Account Secret", Placeholder: "OfCknZXmL1shKB7qhxdpvkwqQYwn4PQr", Type: "text", MinLength: 32, MaxLength: 100},
		},
		Values: values,
	}

	return ui, nil
}

func (mp *Mixpanel) call(ctx context.Context, method, path string, body io.Reader, expectedStatus int, response any) error {

	u := "https://api.mixpanel.com"
	if mp.conf.Region == chichi.PrivacyRegionEurope {
		u = "https://api-eu.mixpanel.com"
	}
	u += path + "?strict=0&project_id=" + mp.settings.ProjectID

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}

	req.SetBasicAuth(mp.settings.Username, mp.settings.Secret)
	req.Header.Set("Content-Type", "application/x-ndjson")

	res, err := mp.conf.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != expectedStatus {
		mpErr := &mixpanelError{}
		dec := json.NewDecoder(res.Body)
		_ = dec.Decode(mpErr)
		return mpErr
	}

	if response != nil {
		dec := json.NewDecoder(res.Body)
		return dec.Decode(response)
	}

	return nil
}

// saveValues saves the user-entered values as settings.
func (mp *Mixpanel) saveValues(ctx context.Context, values []byte) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	if n, err := strconv.Atoi(s.ProjectID); err != nil || n < 0 {
		return chichi.NewInvalidUIValuesError("project ID must be a positive number")
	}
	if n := len(s.Username); n < 20 || n > 100 {
		return chichi.NewInvalidUIValuesError("username length must be in range [20, 100]")
	}
	if n := len(s.Secret); n < 32 || n > 100 {
		return chichi.NewInvalidUIValuesError("secret length must be in range [32, 100]")
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

type mixpanelError struct {
	Code               int
	ErrorText          string `json:"error"`
	NumRecordsImported int    `json:"num_records_imported"`
	Status             string
	FailedRecords      []struct {
		Index    int
		InsertId string `json:"insert_id"`
		Field    string
		Message  string
	} `json:"failed_records"`
}

func (err *mixpanelError) Error() string {
	if err.ErrorText != "" {
		return fmt.Sprintf("unexpected error from Mixpanel (%s): %s", err.Status, err.ErrorText)
	}
	var msg strings.Builder
	for i, record := range err.FailedRecords {
		if i > 0 {
			msg.WriteString(", ")
		}
		_, _ = io.WriteString(&msg, record.Message)
	}
	return fmt.Sprintf("unexpected error from Mixpanel (%s): %s", err.Status, &msg)
}
