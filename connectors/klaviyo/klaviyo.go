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
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the App, AppEvents, AppRecords, and UIHandler interfaces.
var _ interface {
	meergo.App
	meergo.AppEvents
	meergo.AppRecords
	meergo.UIHandler
} = (*Klavyio)(nil)

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name:                   "Klaviyo",
		Targets:                meergo.Events | meergo.Users,
		SourceDescription:      "import clients as users from Klaviyo",
		DestinationDescription: "export users as clients and send events to Klaviyo",
		TermForUsers:           "clients",
		Icon:                   icon,
		BackoffPolicy: meergo.BackoffPolicy{
			// https://developers.klaviyo.com/en/docs/rate_limits_and_error_handling
			"429":     meergo.RetryAfterStrategy(),
			"500 503": meergo.ExponentialStrategy(100 * time.Millisecond),
		},
		SendingMode: meergo.Cloud,
	}, New)
}

// New returns a new Klaviyo connector instance.
func New(conf *meergo.AppConfig) (*Klavyio, error) {
	c := Klavyio{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Klaviyo connector")
		}
	}
	return &c, nil
}

type Klavyio struct {
	conf     *meergo.AppConfig
	settings *Settings
}

type Settings struct {
	PrivateAPIKey string
}

// Create creates a record for the specified target with the given properties.
func (ky *Klavyio) Create(ctx context.Context, target meergo.Targets, properties map[string]any) error {
	panic("TODO: not implemented")
}

// EventRequest returns a request to dispatch an event to the app.
func (ky *Klavyio) EventRequest(ctx context.Context, event *meergo.Event, eventType string, schema types.Type, properties map[string]any, redacted bool) (*meergo.EventRequest, error) {
	req := &meergo.EventRequest{
		Method: "POST",
		URL:    "https://a.klaviyo.com/api/events/",
		Header: http.Header{},
	}
	key := ky.settings.PrivateAPIKey
	if redacted {
		key = "[REDACTED]"
	}
	req.Header.Set("Authorization", "Klaviyo-API-Key "+key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Revision", "2023-01-24")
	var body struct {
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
	body.Data.Type = "event"
	body.Data.Attributes.Profile.Email = properties["email"].(string)
	body.Data.Attributes.Metric.Name = properties["metric_name"].(string)
	body.Data.Attributes.Properties = properties
	body.Data.Attributes.Time = event.Timestamp.Format(time.RFC3339)
	var err error
	req.Body, err = json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// EventTypes returns the event types of the connector's instance.
func (ky *Klavyio) EventTypes(ctx context.Context) ([]*meergo.EventType, error) {
	return []*meergo.EventType{
		{
			ID:          "create_event",
			Name:        "Create Event",
			Description: "Create an Event on Klaviyo",
		},
	}, nil
}

// Records returns the records of the specified target.
func (ky *Klavyio) Records(ctx context.Context, _ meergo.Targets, properties []string, cursor meergo.Cursor) ([]meergo.Record, string, error) {

	var hasUpdatedProperty bool

	url := cursor.Next
	if url == "" {
		var b strings.Builder
		b.WriteString("https://a.klaviyo.com/api/profiles/?fields%5Bprofile%5D=")
		for i, p := range properties {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(p)
			if p == "updated" {
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

	err := ky.call(ctx, "GET", url, nil, 200, &response)
	if err != nil {
		return nil, "", err
	}
	if response.Links.Next != "" && !strings.HasPrefix(response.Links.Next, "https://a.klaviyo.com/") {
		return nil, "", fmt.Errorf("unexpected links.next URL %q", response.Links.Next)
	}
	if len(response.Data) == 0 {
		return nil, "", io.EOF
	}

	users := make([]meergo.Record, len(response.Data))
	for i, data := range response.Data {
		users[i] = meergo.Record{
			ID: data.ID,
		}
		updated, _ := data.Attributes["updated"].(string)
		lastChangeTime, err := time.Parse(time.RFC3339, updated)
		if err != nil {
			users[i].Err = fmt.Errorf("Klaviyo has returned an invalid value for the 'updated' attribute: %q", updated)
			continue
		}
		if !hasUpdatedProperty {
			delete(data.Attributes, "updated")
		}
		users[i].Properties = data.Attributes
		users[i].LastChangeTime = lastChangeTime.UTC()
	}

	if response.Links.Next == "" {
		return users, "", io.EOF
	}

	return users, response.Links.Next, nil
}

// Schema returns the schema of the specified target.
func (ky *Klavyio) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {

	if target == meergo.Events {
		if eventType != "create_event" {
			return types.Type{}, meergo.ErrUIEventNotExist
		}
		return types.Object([]types.Property{
			{Name: "email", Type: types.Text(), CreateRequired: true},
			{Name: "metric_name", Type: types.Text(), CreateRequired: true},
		}), nil
	}

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
			Type:     types.DateTime(),
			Nullable: true,
		},
		{
			Name:     "updated",
			Label:    "Profile Updated",
			Type:     types.DateTime(),
			Nullable: true,
		},
		{
			Name:     "last_event_date",
			Type:     types.DateTime(),
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

// ServeUI serves the connector's user interface.
func (ky *Klavyio) ServeUI(ctx context.Context, event string, values []byte, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s Settings
		if ky.settings != nil {
			s = *ky.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, ky.saveValues(ctx, values)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "PrivateAPIKey", Label: "Your Private Key", Placeholder: "pk_62a6ty4674c6bc5df7c252ea4ed2c7ef81", Type: "text", MinLength: 37, MaxLength: 255},
		},
		Values: values,
	}

	return ui, nil
}

// Update updates a record of the specified target.
func (ky *Klavyio) Update(ctx context.Context, target meergo.Targets, id string, properties map[string]any) error {
	panic("TODO: not implemented")
}

// saveValues saves the user-entered values as settings.
func (ky *Klavyio) saveValues(ctx context.Context, values []byte) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	if n := len(s.PrivateAPIKey); n < 37 {
		return meergo.NewInvalidUIValuesError("private API key must be at least 37 characters long")
	}
	if !strings.HasPrefix(s.PrivateAPIKey, "pk_") {
		return meergo.NewInvalidUIValuesError("private API key must begin with 'pk_'")
	}
	for i := 3; i < len(s.PrivateAPIKey); i++ {
		c := s.PrivateAPIKey[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return meergo.NewInvalidUIValuesError("private API key after 'pk_' must contain only alphanumeric characters")
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ky.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	ky.settings = &s
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

func (ky *Klavyio) call(ctx context.Context, method, url string, body io.Reader, expectedStatus int, response any) error {

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Klaviyo-API-Key "+ky.settings.PrivateAPIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Revision", "2023-01-24")

	res, err := ky.conf.HTTPClient.Do(req)
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
