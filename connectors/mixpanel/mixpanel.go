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

	"chichi/connector"
	"chichi/connector/types"
	"chichi/connector/ui"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the UI, the AppEventsConnection, and the
// AppUsersConnection interfaces.
var _ interface {
	connector.UI
	connector.AppEventsConnection
} = (*connection)(nil)

func init() {
	connector.RegisterApp(connector.App{
		Name:                   "Mixpanel",
		DestinationDescription: "send events to Mixpanel",
		Icon:                   icon,
	}, new)
}

type connection struct {
	conf     *connector.AppConfig
	settings *settings
}

type settings struct {
	ProjectID string
	Username  string
	Secret    string
}

// new returns a new Mixpanel connection.
func new(conf *connector.AppConfig) (*connection, error) {
	c := connection{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Mixpanel connection")
		}
	}
	return &c, nil
}

// EventRequest returns an event request associated with the provided event
// type, event, and transformation data. If redacted is true, sensitive
// authentication data will be redacted in the returned request.
// This method is safe for concurrent use by multiple goroutines.
// If the specified event type does not exist, it returns the
// ErrEventTypeNotExist error.
func (c *connection) EventRequest(ctx context.Context, eventType *connector.EventType, event *connector.Event, data map[string]any, redacted bool) (*connector.EventRequest, error) {

	if data["event"].(string) == "" {
		return nil, errors.New("event cannot be empty")
	}

	req := &connector.EventRequest{
		Method: "POST",
		URL:    "https://api.mixpanel.com/",
		Header: http.Header{},
	}
	if c.conf.Region == connector.PrivacyRegionEurope {
		req.URL = "https://api-eu.mixpanel.com/"
	}
	req.URL += "import?strict=0&project_id=" + c.settings.ProjectID
	req.Header.Set("Content-Type", "application/x-ndjson")
	authorization := base64.StdEncoding.EncodeToString([]byte(c.settings.Username + ":" + c.settings.Secret))
	if redacted {
		authorization = "[REDACTED]"
	}
	req.Header.Set("Authorization", authorization)

	body := data["properties"].(map[string]any)
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

// EventTypes returns the connection's event types.
func (c *connection) EventTypes(ctx context.Context) ([]*connector.EventType, error) {
	if c.conf.Role != connector.Destination {
		return nil, nil
	}
	schema := func(placeholder string) types.Type {
		return types.Object([]types.Property{
			{Name: "event", Label: "Event Name", Placeholder: placeholder, Type: types.Text().WithCharLen(255), Required: true},
			{Name: "properties", Label: "Your Properties", Type: types.Map(types.JSON()), Required: true},
		})
	}
	eventTypes := []*connector.EventType{
		{
			ID:          "track",
			Name:        "Send track events",
			Description: "Send track events to Mixpanel",
			Schema:      schema("event"),
		},
		{
			ID:          "page",
			Name:        "Send page events",
			Description: "Send page events to Mixpanel",
			Schema:      schema(`"Page View"`),
		},
		{
			ID:          "screen",
			Name:        "Send screen events",
			Description: "Send screen events to Mixpanel",
			Schema:      schema(`"Screen View"`),
		},
	}
	return eventTypes, nil
}

// Resource returns the resource.
func (c *connection) Resource(ctx context.Context) (string, error) {
	return "", nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if c.settings != nil {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := c.ValidateSettings(ctx, values)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, c.conf.SetSettings(ctx, s)
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "ProjectID", Label: "Project ID", Placeholder: "1234567", Type: "text", MinLength: 1, MaxLength: 20},
			&ui.Input{Name: "Username", Label: "Service Account Username", Placeholder: "youraccount.82us7b.mp-service-account", Type: "text", MinLength: 20, MaxLength: 100},
			&ui.Input{Name: "Secret", Label: "Service Account Secret", Placeholder: "OfCknZXmL1shKB7qhxdpvkwqQYwn4PQr", Type: "text", MinLength: 32, MaxLength: 100},
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
func (c *connection) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	if n, err := strconv.Atoi(s.ProjectID); err != nil || n < 0 {
		return nil, ui.Errorf("project ID must be a positive number")
	}
	if n := len(s.Username); n < 20 || n > 100 {
		return nil, ui.Errorf("username length must be in range [20, 100]")
	}
	if n := len(s.Secret); n < 32 || n > 100 {
		return nil, ui.Errorf("secret length must be in range [32, 100]")
	}
	return json.Marshal(&s)
}

func (c *connection) call(ctx context.Context, method, path string, body io.Reader, expectedStatus int, response any) error {

	u := "https://api.mixpanel.com"
	if c.conf.Region == connector.PrivacyRegionEurope {
		u = "https://api-eu.mixpanel.com"
	}
	u += path + "?strict=0&project_id=" + c.settings.ProjectID

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.settings.Username, c.settings.Secret)
	req.Header.Set("Content-Type", "application/x-ndjson")

	res, err := c.conf.HTTPClient.Do(req)
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
