//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package klaviyo implements the Klaviyo connector.
// (https://developers.klaviyo.com/)
package klaviyo

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	connector.AppUsersConnection
} = (*connection)(nil)

func init() {
	connector.RegisterApp(connector.App{
		Name:                   "Klaviyo",
		SourceDescription:      "import clients as users from Klaviyo",
		DestinationDescription: "export users as clients and send events to Klaviyo",
		TermForUsers:           "users",
		Icon:                   icon,
	}, open)
}

// open opens a Klaviyo connection and returns it.
func open(ctx context.Context, conf *connector.AppConfig) (*connection, error) {
	c := connection{ctx: ctx, conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Klaviyo connection")
		}
	}
	return &c, nil
}

type connection struct {
	ctx      context.Context
	conf     *connector.AppConfig
	settings *settings
}

type settings struct {
	PrivateAPIKey string
}

// CreateUser creates a user with the given properties.
func (c *connection) CreateUser(properties connector.Properties) error {
	panic("TODO: not implemented")
}

// EventTypes returns the connection's event types.
func (c *connection) EventTypes() ([]*connector.EventType, error) {
	if c.conf.Role == connector.SourceRole {
		return nil, nil
	}
	eventTypes := []*connector.EventType{
		{
			ID:          "create_event",
			Name:        "Create Event",
			Description: "Create an Event on Klaviyo",
			Schema: types.Object([]types.Property{
				{Name: "email", Type: types.Text(), Required: true},
				{Name: "metric_name", Type: types.Text(), Required: true},
			}),
		},
	}
	return eventTypes, nil
}

// ReceiveWebhook receives a webhook request and returns its events.
// It returns the ErrWebhookUnauthorized error is the request was not authorized.
func (c *connection) ReceiveWebhook(r *http.Request) ([]connector.WebhookEvent, error) {
	return nil, connector.ErrWebhookUnauthorized
}

// Resource returns the resource from a client token.
func (c *connection) Resource() (string, error) {
	return "", nil
}

// SendEvent sends the event, along with the given mapped event.
// eventType specifies the event type corresponding to the event.
func (c *connection) SendEvent(event connector.Event, mappedEvent map[string]any, eventType string) error {
	switch eventType {
	case "create_event":
		var msg struct {
			Data struct {
				Type       string `json:"type"`
				Attributes struct {
					Profile struct {
						Email string `json:"$email"`
					} `json:"profile"`
					Metric struct {
						Name string `json:"name"`
					} `json:"metric"`
					Properties map[string]any `json:"properties"`
					Time       string         `json:"time"`
					Value      any            `json:"value"`
				} `json:"attributes"`
			} `json:"data"`
		}
		msg.Data.Type = "event"
		msg.Data.Attributes.Profile.Email = mappedEvent["email"].(string)
		msg.Data.Attributes.Metric.Name = mappedEvent["metric_name"].(string)
		msg.Data.Attributes.Properties = mappedEvent
		msg.Data.Attributes.Time = event.Timestamp.Format(time.RFC3339)
		body, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		err = c.call("POST", "https://a.klaviyo.com/api/events/", bytes.NewReader(body), 202, nil)
		return err
	default:
		panic(fmt.Sprintf("unsupported event type %q", eventType))
	}
}

// UpdateUser updates the user with identifier id setting the given properties.
func (c *connection) UpdateUser(id string, properties connector.Properties) error {
	panic("TODO: not implemented")
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
			&ui.Input{Name: "PrivateAPIKey", Label: "Your Private Key", Placeholder: "pk_62a6ty4674c6bc5df7c252ea4ed2c7ef81", Type: "text", MinLength: 37, MaxLength: 255},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// UserSchema returns the user schema.
func (c *connection) UserSchema() (types.Type, error) {
	// The fields which are not marked as "required" in the documentation
	// (available here:
	// https://developers.klaviyo.com/en/reference/get_profiles) are declared as
	// nullable properties.
	schema := types.Object([]types.Property{
		{
			Name:  "id",
			Label: "ID",
			Type:  types.Text(),
		},
		{
			Name:     "email",
			Label:    "Email",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name:     "phone_number",
			Label:    "Phone",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name:     "external_id",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name:     "anonymous_id",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name:     "first_name",
			Label:    "First name",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name:     "last_name",
			Label:    "Last name",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name:     "organization",
			Label:    "Organization",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name:     "title",
			Label:    "Title",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name:     "image",
			Label:    "Image",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name:     "created",
			Label:    "Profile Created",
			Type:     types.DateTime().WithLayout(time.RFC3339),
			Nullable: true,
		},
		{
			Name:     "updated",
			Label:    "Profile Updated",
			Type:     types.DateTime().WithLayout(time.RFC3339),
			Nullable: true,
		},
		{
			Name:     "last_event_date",
			Type:     types.DateTime().WithLayout(time.RFC3339),
			Nullable: true,
		},
		{
			Name:  "location",
			Label: "Location",
			Type: types.Object([]types.Property{
				{
					Name:     "address1",
					Label:    "Address1",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "address2",
					Label:    "Address2",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "city",
					Label:    "City",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "country",
					Label:    "Country",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "latitude",
					Label:    "Latitude",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "longitude",
					Label:    "Longitude",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "region",
					Label:    "Region",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "zip",
					Label:    "Zip",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "timezone",
					Label:    "Timezone",
					Type:     types.Text(),
					Nullable: true,
				},
			}),
			Nullable: true,
		},
	},
	)
	return schema, nil
}

// Users returns the users starting from the given cursor.
func (c *connection) Users(properties []types.Path, cursor connector.Cursor) ([]connector.Object, string, error) {

	var hasUpdatedProperty bool

	url := cursor.Next
	if url == "" {
		var b strings.Builder
		b.WriteString("https://a.klaviyo.com/api/profiles/?fields%5Bprofile%5D=")
		for i, p := range properties {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(p[0])
			if p[0] == "updated" {
				hasUpdatedProperty = true
			}
		}
		if !hasUpdatedProperty {
			b.WriteString(",updated")
		}
		b.WriteString("&page%5Bsize%5D=100&sort=created")
		url = b.String()
	}

	var response struct {
		Data []struct {
			ID         string
			Attributes map[string]any
		}
		Links struct {
			Next string
		}
	}

	err := c.call("GET", url, nil, 200, &response)
	if err != nil {
		return nil, "", err
	}
	if response.Links.Next != "" && !strings.HasPrefix(response.Links.Next, "https://a.klaviyo.com/") {
		return nil, "", fmt.Errorf("unexpected links.next URL %q", response.Links.Next)
	}
	if len(response.Data) == 0 {
		return nil, "", io.EOF
	}

	objects := make([]connector.Object, len(response.Data))
	for i, data := range response.Data {
		updated, _ := data.Attributes["updated"].(string)
		timestamp, err := time.Parse(time.RFC3339, updated)
		if err != nil {
			return nil, "", fmt.Errorf("Klaviyo has returned a missing or invalid \"updated\" attribute: %q", updated)
		}
		if !hasUpdatedProperty {
			delete(data.Attributes, "updated")
		}
		objects[i] = connector.Object{
			ID:         data.ID,
			Properties: data.Attributes,
			Timestamp:  timestamp,
		}
	}

	if response.Links.Next == "" {
		return objects, "", io.EOF
	}

	return objects, response.Links.Next, nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	if n := len(s.PrivateAPIKey); n < 37 {
		return nil, ui.Errorf("private API key must be at least 37 characters long")
	}
	if !strings.HasPrefix(s.PrivateAPIKey, "pk_") {
		return nil, ui.Errorf("private API key must begin with 'pk_'")
	}
	for i := 3; i < len(s.PrivateAPIKey); i++ {
		c := s.PrivateAPIKey[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return nil, ui.Errorf("private API key after 'pk_' must contain only alphanumeric characters")
		}
	}
	return json.Marshal(&s)
}

type klaviyoError struct {
	statusCode int
	Errors     []struct {
		ID     string
		Code   string
		Title  string
		Detail string
		Source struct {
			Pointer   string
			Parameter string
		}
	}
}

func (err *klaviyoError) Error() string {
	var msg strings.Builder
	for i, e := range err.Errors {
		if i > 0 {
			msg.WriteString(", ")
		}
		_, _ = fmt.Fprintf(&msg, "%s: %s (error code is %q)", e.Title, e.Detail, e.Code)
	}
	return fmt.Sprintf("unexpected error from Klaviyo (%d): %s", err.statusCode, &msg)
}

func (c *connection) call(method, url string, body io.Reader, expectedStatus int, response any) error {

	req, err := http.NewRequestWithContext(c.ctx, method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Klaviyo-API-Key "+c.settings.PrivateAPIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Revision", "2023-01-24")

	res, err := c.conf.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != expectedStatus {
		kErr := &klaviyoError{statusCode: res.StatusCode}
		dec := json.NewDecoder(res.Body)
		_ = dec.Decode(kErr)
		return kErr
	}

	if response != nil {
		dec := json.NewDecoder(res.Body)
		return dec.Decode(response)
	}

	return nil
}
