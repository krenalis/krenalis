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
	}, open)
}

type connection struct {
	ctx      context.Context
	conf     *connector.AppConfig
	settings *settings
}

type settings struct {
	ProjectID string
	Username  string
	Secret    string
}

// open opens a Mixpanel connection and returns it.
func open(ctx context.Context, conf *connector.AppConfig) (*connection, error) {
	c := connection{ctx: ctx, conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Mixpanel connection")
		}
	}
	return &c, nil
}

// EventTypes returns the connection's event types.
func (c *connection) EventTypes() ([]*connector.EventType, error) {
	if c.conf.Role != connector.DestinationRole {
		return nil, nil
	}
	schema := types.Object([]types.Property{
		{Name: "event", Label: "Event Name", Type: types.Text().WithCharLen(255), Required: true},
		{Name: "properties", Label: "Your Properties", Type: types.Map(types.JSON()), Required: true},
	})
	eventTypes := []*connector.EventType{
		{
			ID:          "track",
			Name:        "Send track events",
			Description: "Send track events to Mixpanel",
			Schema:      schema,
		},
		{
			ID:          "page",
			Name:        "Send page events",
			Description: "Send page events to Mixpanel",
			Schema:      schema,
		},
		{
			ID:          "screen",
			Name:        "Send screen events",
			Description: "Send screen events to Mixpanel",
			Schema:      schema,
		},
	}
	return eventTypes, nil
}

// Resource returns the resource.
func (c *connection) Resource() (string, error) {
	return "", nil
}

// SendEvent sends the event, along with the given mapped event.
// eventType specifies the event type corresponding to the event.
func (c *connection) SendEvent(event connector.Event, mappedEvent map[string]any, eventType string) error {

	if e := mappedEvent["event"].(string); e == "" {
		mappedEvent["event"] = "$mp_web_page_view"
	}

	p := mappedEvent["properties"].(map[string]any)

	p["$insert_id"] = event.MessageID
	p["time"] = formatTimestamp(event.Timestamp)
	distinctID := event.AnonymousID
	if event.UserID != "" {
		distinctID = event.UserID
	}
	p["distinct_id"] = distinctID
	p["$device_id"] = event.AnonymousID
	if event.IP == "" {
		if event.Location.Country != "" {
			p["mp_country_code"] = event.Location.Country
		}
		if event.Location.Region != "" {
			p["$region"] = event.Location.Region
		}
		if event.Location.City != "" {
			p["$city"] = event.Location.City
		}
	} else {
		p["ip"] = event.IP
		// Supplying the 'ip' property, Mixpanel automatically enriches the event with country, region, and city
		// if they are not supplied. Provide either all or none of these properties to ensure that Mixpanel's
		// enrichment occurs for all or none of them.
		if event.Location.Country != "" || event.Location.Region != "" || event.Location.City != "" {
			if event.Location.Country != "" {
				p["mp_country_code"] = event.Location.Country
			} else {
				p["mp_country_code"] = nil
			}
			if event.Location.Region != "" {
				p["$region"] = event.Location.Region
			} else {
				p["$region"] = nil
			}
			if event.Location.City != "" {
				p["$city"] = event.Location.City
			} else {
				p["$city"] = nil
			}
		}
	}
	if event.OS.Name != "" {
		p["$os"] = event.OS.Name
	}
	if event.Browser.Name != "" {
		p["$browser"] = event.Browser.Name
	} else if event.Browser.Other != "" {
		p["$browser"] = event.Browser.Other
	}
	if event.Browser.Version != "" {
		p["$browser_version"] = event.Browser.Version
	}
	if event.Page.Referrer != "" {
		u, err := url.Parse(event.Page.Referrer)
		if err == nil {
			p["$referrer"] = event.Page.Referrer
			p["$referring_domain"] = u.Hostname()
		}
	}
	if event.Page.URL != "" {
		p["$current_url"] = event.Page.URL
		p["current_page_title"] = event.Page.Title
		u, err := url.Parse(event.Page.URL)
		if err == nil {
			p["current_domain"] = u.Hostname()
			p["current_url_path"] = u.Path
			p["current_url_protocol"] = u.Scheme + ":"
		}
	}
	if event.Screen.Width != 0 {
		p["$screen_width"] = event.Screen.Width
	}
	if event.Screen.Height != 0 {
		p["$screen_height"] = event.Screen.Height
	}

	// Send the event.
	body, err := json.Marshal(mappedEvent)
	if err != nil {
		return err
	}
	err = c.call("POST", "/import", bytes.NewReader(body), 200, nil)

	return err
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

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
		s, err := c.ValidateSettings(values)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, c.conf.SetSettings(s)
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
func (c *connection) ValidateSettings(values []byte) ([]byte, error) {
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

func (c *connection) call(method, path string, body io.Reader, expectedStatus int, response any) error {

	u := "https://api.mixpanel.com"
	if c.conf.Region == connector.PrivacyRegionEurope {
		u = "https://api-eu.mixpanel.com"
	}
	u += path + "?strict=0&project_id=" + c.settings.ProjectID

	req, err := http.NewRequestWithContext(c.ctx, method, u, body)
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
