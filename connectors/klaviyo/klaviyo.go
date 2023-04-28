//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package klaviyo

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"

	"github.com/open2b/nuts/capture"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the AppEventsConnection and the AppUsersConnection
// interfaces.
var _ interface {
	connector.AppEventsConnection
	connector.AppUsersConnection
} = (*connection)(nil)

var Debug = false

func init() {
	connector.RegisterApp(connector.App{
		Name:                   "Klaviyo",
		SourceDescription:      "import clients as users from Klaviyo",
		DestinationDescription: "export users as clients and send events to Klaviyo",
		TermForUsers:           "users",
		Icon:                   icon,
	}, open)
}

type connection struct {
	ctx      context.Context
	role     connector.Role
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	PrivateAPIKey string
}

// open opens a Klaviyo connection and returns it.
func open(ctx context.Context, conf *connector.AppConfig) (*connection, error) {
	c := connection{
		ctx:      ctx,
		role:     conf.Role,
		firehose: conf.Firehose,
	}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Klaviyo connection")
		}
	}
	return &c, nil
}

// EventTypes returns the connection's event types.
func (c *connection) EventTypes() ([]*connector.EventType, error) {
	if c.role == connector.SourceRole {
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

// SetUsers sets the users.
func (c *connection) SetUsers(users []connector.User) error {
	return errors.New("not implemented")
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
		s, err := c.SettingsUI(values)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, c.firehose.SetSettings(s)
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

// SettingsUI obtains the settings from UI values and returns them.
func (c *connection) SettingsUI(values []byte) ([]byte, error) {
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

// UserSchema returns the user schema.
func (c *connection) UserSchema() (types.Type, error) {
	schema := types.Object([]types.Property{
		{
			Name:  "email",
			Label: "Email",
			Type:  types.Text(),
		},
		{
			Name:  "phone_number",
			Label: "Phone",
			Type:  types.Text(),
		},
		{
			Name: "external_id",
			Type: types.Text(),
		},
		{
			Name: "anonymous_id",
			Type: types.Text(),
		},
		{
			Name:  "first_name",
			Label: "First name",
			Type:  types.Text(),
		},
		{
			Name:  "last_name",
			Label: "Last name",
			Type:  types.Text(),
		},
		{
			Name:  "organization",
			Label: "Organization",
			Type:  types.Text(),
		},
		{
			Name:  "title",
			Label: "Title",
			Type:  types.Text(),
		},
		{
			Name:  "image",
			Label: "Image",
			Type:  types.Text(),
		},
		{
			Name:  "created",
			Label: "Profile Created",
			Type:  types.DateTime().WithLayout(time.RFC3339),
		},
		{
			Name:  "updated",
			Label: "Profile Updated",
			Type:  types.DateTime().WithLayout(time.RFC3339),
		},
		{
			Name: "last_event_date",
			Type: types.DateTime().WithLayout(time.RFC3339),
		},
		{
			Name:  "location",
			Label: "Location",
			Type: types.Object([]types.Property{
				{
					Name:  "address1",
					Label: "Address1",
					Type:  types.Text(),
				},
				{
					Name:  "address2",
					Label: "Address2",
					Type:  types.Text(),
				},
				{
					Name:  "city",
					Label: "City",
					Type:  types.Text(),
				},
				{
					Name:  "country",
					Label: "Country",
					Type:  types.Text(),
				},
				{
					Name:  "latitude",
					Label: "Latitude",
					Type:  types.Text(),
				},
				{
					Name:  "longitude",
					Label: "Longitude",
					Type:  types.Text(),
				},
				{
					Name:  "region",
					Label: "Region",
					Type:  types.Text(),
				},
				{
					Name:  "zip",
					Label: "Zip",
					Type:  types.Text(),
				},
				{
					Name:  "timezone",
					Label: "Timezone",
					Type:  types.Text(),
				},
			}),
		}})
	return schema, nil
}

// Users returns the users starting from the given cursor.
func (c *connection) Users(cursor string, properties []connector.PropertyPath) error {

	it, err := c.newIterator(properties, 100)
	if err != nil {
		return err
	}
	for {
		profiles, err := it.next()
		if err != nil {
			return err
		}
		if len(profiles) == 0 {
			break
		}
		for _, profile := range profiles {
			c.firehose.SetUser(profile.ID, profile.Attributes, profile.timestamp, nil)
		}
	}

	return nil
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

	var dump *bufio.Writer
	if Debug {
		dump = bufio.NewWriter(os.Stdout)
		dump.WriteString("\nRequest:\n------\n")
		capture.Request(req, dump, true, true)
	}

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if Debug {
		dump.Reset(os.Stdout)
		dump.WriteString("\n\n\nResponse:\n------\n")
		capture.Response(res, dump, true, true)
	}

	if res.StatusCode != expectedStatus {
		hsErr := &klaviyoError{statusCode: res.StatusCode}
		dec := json.NewDecoder(res.Body)
		_ = dec.Decode(hsErr)
		return hsErr
	}

	if response != nil {
		dec := json.NewDecoder(res.Body)
		err = dec.Decode(response)
		if err != nil {
			return err
		}
	}

	return nil
}

// pageSize must be in [1, 100].
func (c *connection) newIterator(properties []connector.PropertyPath, pageSize int) (*iter, error) {

	if pageSize < 0 || pageSize > 100 {
		return nil, errors.New("invalid page size")
	}

	it := iter{
		connection: c,
	}

	// Encode the URL query parameters.
	var b strings.Builder
	b.WriteString("https://a.klaviyo.com/api/profiles/?")
	b.WriteString("fields%5Bprofile%5D=")
	for i, p := range properties {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(url.QueryEscape(p[0]))
		if p[0] == "updated" {
			it.HasUpdatedProperty = true
		}
	}
	if !it.HasUpdatedProperty {
		b.WriteString(",updated")
	}

	b.WriteString("&page%5Bsize%5D=")
	b.WriteString(strconv.Itoa(pageSize))
	b.WriteString("&sort=created")

	it.Next = b.String()

	return &it, nil
}

type iter struct {
	*connection
	HasUpdatedProperty bool
	Next               string
	Terminated         bool
}

type profile struct {
	ID         string
	Attributes map[string]any
	timestamp  time.Time
}

// next returns the next objects or nil if there are no objects.
func (it *iter) next() ([]profile, error) {

	if it.Terminated {
		return nil, nil
	}

	var response struct {
		Data  []profile
		Links struct {
			Next string
		}
	}

	err := it.call("GET", it.Next, nil, 200, &response)
	if err != nil {
		return nil, err
	}

	if response.Links.Next != "" && !strings.HasPrefix(response.Links.Next, "https://a.klaviyo.com/") {
		return nil, fmt.Errorf("unexpected links.next URL %q", response.Links.Next)
	}

	it.Next = response.Links.Next
	it.Terminated = it.Next == ""

	if len(response.Data) == 0 {
		return nil, nil
	}

	for i, p := range response.Data {
		updated, _ := p.Attributes["updated"].(string)
		t, err := time.Parse(time.RFC3339, updated)
		if err != nil {
			return nil, fmt.Errorf("Klaviyo has returned a missing or invalid \"updated\" attribute: %q", updated)
		}
		response.Data[i].timestamp = t.UTC()
		if !it.HasUpdatedProperty {
			delete(p.Attributes, "updated")
		}
	}

	return response.Data, nil
}
