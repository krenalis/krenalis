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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name: "Klaviyo",
		AsSource: &meergo.AsAppSource{
			Description: "Import profiles as users from Klaviyo",
			Targets:     meergo.Users,
			HasSettings: true,
		},
		AsDestination: &meergo.AsAppDestination{
			Description: "Export users as profiles and send events to Klaviyo",
			Targets:     meergo.Events | meergo.Users,
			HasSettings: true,
			SendingMode: meergo.Cloud,
		},
		Terms: meergo.AppTerms{
			User:  "client",
			Users: "clients",
		},
		Icon: icon,
		BackoffPolicy: meergo.BackoffPolicy{
			// https://developers.klaviyo.com/en/docs/rate_limits_and_error_handling
			"429":     meergo.RetryAfterStrategy(),
			"500 503": meergo.ExponentialStrategy(100 * time.Millisecond),
		},
	}, New)
}

// apiRevision is the API revision to use for calls to the Klaviyo API methods.
const apiRevision = "2024-07-15"

// New returns a new Klaviyo connector instance.
func New(conf *meergo.AppConfig) (*Klaviyo, error) {
	c := Klaviyo{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Klaviyo connector")
		}
	}
	return &c, nil
}

type Klaviyo struct {
	conf     *meergo.AppConfig
	settings *innerSettings
}

type innerSettings struct {
	PrivateAPIKey string
}

// EventRequest returns a request to dispatch an event to the app.
func (ky *Klaviyo) EventRequest(ctx context.Context, event meergo.Event, eventType string, schema types.Type, properties map[string]any, redacted bool) (*meergo.EventRequest, error) {
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
	req.Header.Set("Revision", apiRevision)
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
	body.Data.Attributes.Time = event.Timestamp().Format(time.RFC3339)
	var err error
	req.Body, err = json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// EventTypes returns the event types of the connector's instance.
func (ky *Klaviyo) EventTypes(ctx context.Context) ([]*meergo.EventType, error) {
	return []*meergo.EventType{
		{
			ID:          "create_event",
			Name:        "Create Event",
			Description: "Create an Event on Klaviyo",
		},
	}, nil
}

// Records returns the records of the specified target.
func (ky *Klaviyo) Records(ctx context.Context, _ meergo.Targets, lastChangeTime time.Time, ids, properties []string, cursor string, _ types.Type) ([]meergo.Record, string, error) {

	var hasID bool
	var hasUpdated bool

	u := cursor
	if u == "" {
		var b strings.Builder
		b.WriteString("https://a.klaviyo.com/api/profiles/?fields%5Bprofile%5D=")
		i := 0
		for _, p := range properties {
			if p == "id" {
				hasID = true
				continue
			}
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(p)
			if p == "updated" {
				hasUpdated = true
			}
			i++
		}
		if !hasUpdated {
			b.WriteString(",updated")
		}
		b.WriteString("&page%5Bsize%5D=100&sort=updated")
		if !lastChangeTime.IsZero() {
			b.WriteString("&filter=greater-than%28updated%2C")
			b.WriteString(url.QueryEscape(lastChangeTime.Add(-time.Second).Format(time.RFC3339)))
			b.WriteString("%29")
		}
		if ids != nil {
			b.WriteString("&filter=any%28id%2C%5B")
			for i, id := range ids {
				if i > 0 {
					b.WriteString("%2C")
				}
				b.WriteString(`%22`)
				b.WriteString(url.QueryEscape(id))
				b.WriteString(`%22`)
			}
			b.WriteString("%5D%29")
		}
		u = b.String()
	}

	var response struct {
		Data []struct {
			ID         string         `json:"id"`
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
		Links struct {
			Next string `json:"next"`
		} `json:"links"`
	}

	err := ky.call(ctx, "GET", u, nil, 200, &response)
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
		if hasID {
			data.Attributes["id"] = users[i].ID
		}
		if !hasUpdated {
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

// Schema returns the schema of the specified target in the specified role.
func (ky *Klaviyo) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {

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
			Name:        "id",
			Type:        types.Text(),
			Description: "Unique ID",
		},
		{
			Name:        "email",
			Type:        types.Text(),
			Nullable:    true,
			Description: "Email",
		},
		{
			Name:        "phone_number",
			Type:        types.Text(),
			Nullable:    true,
			Description: "Phone",
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
			Name:        "first_name",
			Type:        types.Text(),
			Nullable:    true,
			Description: "First name",
		},
		{
			Name:        "last_name",
			Type:        types.Text(),
			Nullable:    true,
			Description: "Last name",
		},
		{
			Name:     "organization",
			Type:     types.Text(),
			Nullable: true,
		},
		{
			Name:        "title",
			Type:        types.Text(),
			Nullable:    true,
			Description: "Title",
		},
		{
			Name:        "image",
			Type:        types.Text(),
			Nullable:    true,
			Description: "Image",
		},
		{
			Name:        "created",
			Type:        types.DateTime(),
			Nullable:    true,
			Description: "Profile Created",
		},
		{
			Name:        "updated",
			Type:        types.DateTime(),
			Nullable:    true,
			Description: "Profile Updated",
		},
		{
			Name:     "last_event_date",
			Type:     types.DateTime(),
			Nullable: true,
		},
		{
			Name: "location",
			Type: types.Object([]types.Property{
				{
					Name:     "address1",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "address2",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "city",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "country",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "latitude",
					Type:     types.Float(64),
					Nullable: true,
				},
				{
					Name:     "longitude",
					Type:     types.Float(64),
					Nullable: true,
				},
				{
					Name:     "region",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "zip",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "timezone",
					Type:     types.Text(),
					Nullable: true,
				},
				{
					Name:     "ip",
					Type:     types.Inet(),
					Nullable: true,
				},
			}),
			Nullable:    true,
			Description: "Location",
		},
		{
			Name:        "properties",
			Type:        types.Map(types.JSON()),
			Nullable:    true,
			Description: "Custom Properties",
		},
	})
	if role == meergo.Destination {
		sourceOnlyProperties := []string{"id", "anonymous_id", "created", "updated", "last_event_date"}
		schema = types.SubsetFunc(schema, func(p types.Property) bool {
			return !slices.Contains(sourceOnlyProperties, p.Name)
		})
	}
	return schema, nil
}

// ServeUI serves the connector's user interface.
func (ky *Klaviyo) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if ky.settings != nil {
			s = *ky.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, ky.saveSettings(ctx, settings)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "PrivateAPIKey", Label: "Your Private Key", Placeholder: "pk_62a6ty4674c6bc5df7c252ea4ed2c7ef81", Type: "text", MinLength: 37, MaxLength: 255},
		},
		Settings: settings,
	}

	return ui, nil
}

// Upsert updates or creates records in the app for the specified target.
func (ky *Klaviyo) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {

	record := records.First()

	customProperties, ok := record.Properties["properties"]
	if ok {
		delete(record.Properties, "properties")
	}
	var body json.Buffer
	body.WriteString(`{"data":{"type":"profile","attributes":`)
	_ = body.Encode(record.Properties)
	if ok {
		body.Truncate(body.Len() - 1) // remove '}'.
		body.WriteString(`,"properties":`)
		_ = body.Encode(customProperties)
		body.WriteByte('}') // add '}'.
	}
	if record.ID != "" {
		body.WriteString(`,"id":`)
		_ = body.Encode(record.ID)
	}
	body.WriteString(`}}`)

	u := "https://a.klaviyo.com/api/profiles/"
	if record.ID == "" {
		return ky.call(ctx, "POST", u, &body, 201, nil)
	}

	return ky.call(ctx, "PATCH", u+url.PathEscape(record.ID)+"/", &body, 200, nil)
}

// saveSettings validates and saves the settings.
func (ky *Klaviyo) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	if n := len(s.PrivateAPIKey); n < 37 {
		return meergo.NewInvalidSettingsError("private API key must be at least 37 characters long")
	}
	if !strings.HasPrefix(s.PrivateAPIKey, "pk_") {
		return meergo.NewInvalidSettingsError("private API key must begin with 'pk_'")
	}
	for i := 3; i < len(s.PrivateAPIKey); i++ {
		c := s.PrivateAPIKey[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return meergo.NewInvalidSettingsError("private API key after 'pk_' must contain only alphanumeric characters")
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
		ID     string `json:"id"`
		Code   string `json:"code"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
		Source struct {
			Pointer   string `json:"pointer"`
			Parameter string `json:"parameter"`
		} `json:"source"`
	} `json:"errors"`
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

func (ky *Klaviyo) call(ctx context.Context, method, url string, body io.Reader, expectedStatus int, response any) error {

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Klaviyo-API-Key "+ky.settings.PrivateAPIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Revision", apiRevision)

	res, err := ky.conf.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != expectedStatus {
		kErr := &klaviyoError{statusCode: res.StatusCode}
		err := json.Decode(res.Body, kErr)
		if err != nil {
			return err
		}
		return kErr
	}

	if response != nil {
		return json.Decode(res.Body, response)
	}

	return nil
}
