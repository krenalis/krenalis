// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package klaviyo provides a connector for Klaviyo.
// (https://developers.klaviyo.com/)
//
// Klaviyo is a trademark of Klaviyo, Inc.
// This connector is not affiliated with or endorsed by Klaviyo, Inc.
package klaviyo

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/tools/validation"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterAPI(connectors.APISpec{
		Code:       "klaviyo",
		Label:      "Klaviyo",
		Categories: connectors.CategorySaaS,
		AsSource: &connectors.AsAPISource{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Import profiles as users from Klaviyo",
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsAPIDestination{
			Targets:     connectors.TargetEvent | connectors.TargetUser,
			HasSettings: true,
			SendingMode: connectors.Server,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Export users as profiles and send events to Klaviyo",
				Overview: destinationOverview,
			},
		},
		Terms: connectors.APITerms{
			User:  "Profile",
			Users: "Profiles",
		},
		EndpointGroups: []connectors.EndpointGroup{
			{
				Patterns:    []string{"/api/event-bulk-create-jobs"},
				RateLimit:   connectors.RateLimit{RequestsPerSecond: 2.5, Burst: 10},
				RetryPolicy: retryPolicy,
			},
			{
				Patterns:    []string{"/api/profiles/"},
				RateLimit:   connectors.RateLimit{RequestsPerSecond: 11.6, Burst: 75},
				RetryPolicy: retryPolicy,
			},
		},
	}, New)
}

var retryPolicy = connectors.RetryPolicy{
	// https://developers.klaviyo.com/en/docs/rate_limits_and_error_handling
	"429":     connectors.RetryAfterStrategy(),
	"500 503": connectors.ExponentialStrategy(connectors.NetFailure, 100*time.Millisecond),
}

// apiRevision is the API revision to use for calls to the Klaviyo API methods.
const apiRevision = "2024-07-15"

// New returns a new connector instance for Klaviyo.
func New(env *connectors.APIEnv) (*Klaviyo, error) {
	c := Klaviyo{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for Klaviyo")
		}
	}
	return &c, nil
}

type Klaviyo struct {
	env      *connectors.APIEnv
	settings *innerSettings
}

type innerSettings struct {
	PrivateAPIKey string
}

// EventTypeSchema returns the schema of the specified event type.
func (ky *Klaviyo) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {
	if eventType == "create_event" {
		return types.Object([]types.Property{
			{Name: "metric_name", Type: types.String().WithMaxLength(200), CreateRequired: true, Description: "Metric name"},
			{Name: "email", Type: types.String().WithMaxBytes(100), CreateRequired: true, Description: "Email"},
			{Name: "value", Type: types.Float(64).Real(), Description: "Value"},
			{Name: "value_currency", Type: types.String().WithMaxBytes(3), Description: "Currency (ISO code)"},
			{Name: "properties", Type: types.Map(types.JSON()), Description: "Properties"},
		}), nil
	}
	return types.Type{}, connectors.ErrUIEventNotExist
}

// EventTypes returns the event types.
func (ky *Klaviyo) EventTypes(ctx context.Context) ([]*connectors.EventType, error) {
	return []*connectors.EventType{
		{
			ID:          "create_event",
			Name:        "Create event",
			Description: "Create an event on Klaviyo",
		},
	}, nil
}

// PreviewSendEvents returns the HTTP request that would be used to send the
// events to the API, without actually sending it.
func (ky *Klaviyo) PreviewSendEvents(ctx context.Context, events connectors.Events) (*http.Request, error) {
	return ky.sendEvents(ctx, events, true)
}

// Records returns the records of the specified target.
func (ky *Klaviyo) Records(ctx context.Context, _ connectors.Targets, lastChangeTime time.Time, ids []string, cursor string, schema types.Type) ([]connectors.Record, string, error) {

	var hasID bool
	var hasUpdated bool

	u := cursor
	if u == "" {
		var b strings.Builder
		b.WriteString("https://a.klaviyo.com/api/profiles/?fields%5Bprofile%5D=")
		i := 0
		for _, p := range schema.Properties().All() {
			if p.Name == "id" {
				hasID = true
				continue
			}
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(p.Name)
			if p.Name == "updated" {
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

	users := make([]connectors.Record, len(response.Data))
	for i, data := range response.Data {
		users[i] = connectors.Record{
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
		if pp, ok := data.Attributes["properties"].(map[string]any); ok {
			for k, v := range pp {
				pp[k], _ = json.Marshal(v)
			}
		}
		users[i].Attributes = data.Attributes
		users[i].LastChangeTime = lastChangeTime.UTC()
	}

	if response.Links.Next == "" {
		return users, "", io.EOF
	}

	return users, response.Links.Next, nil
}

// RecordSchema returns the schema of the specified target and role.
func (ky *Klaviyo) RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {
	// The fields which are not marked as "required" in the documentation
	// (available here:
	// https://developers.klaviyo.com/en/reference/get_profiles) are declared as
	// nullable properties.
	schema := types.Object([]types.Property{
		{
			Name:        "id",
			Type:        types.String(),
			Description: "Unique ID",
		},
		{
			Name:        "email",
			Type:        types.String(),
			Nullable:    true,
			Description: "Email",
		},
		{
			Name:        "phone_number",
			Type:        types.String(),
			Nullable:    true,
			Description: "Phone",
		},
		{
			Name:        "external_id",
			Type:        types.String(),
			Nullable:    true,
			Description: "External Id",
		},
		{
			Name:        "anonymous_id",
			Type:        types.String(),
			Nullable:    true,
			Description: "Anonymous Id",
		},
		{
			Name:        "first_name",
			Type:        types.String(),
			Nullable:    true,
			Description: "First name",
		},
		{
			Name:        "last_name",
			Type:        types.String(),
			Nullable:    true,
			Description: "Last name",
		},
		{
			Name:        "organization",
			Type:        types.String(),
			Nullable:    true,
			Description: "Organization",
		},
		{
			Name:        "title",
			Type:        types.String(),
			Nullable:    true,
			Description: "Title",
		},
		{
			Name:        "image",
			Type:        types.String(),
			Nullable:    true,
			Description: "Image",
		},
		{
			Name:        "created",
			Type:        types.DateTime(),
			Nullable:    true,
			Description: "Profile created",
		},
		{
			Name:        "updated",
			Type:        types.DateTime(),
			Nullable:    true,
			Description: "Profile updated",
		},
		{
			Name:        "last_event_date",
			Type:        types.DateTime(),
			Nullable:    true,
			Description: "Last event date",
		},
		{
			Name: "location",
			Type: types.Object([]types.Property{
				{
					Name:        "address1",
					Type:        types.String(),
					Nullable:    true,
					Description: "Street",
				},
				{
					Name:        "address2",
					Type:        types.String(),
					Nullable:    true,
					Description: "Street (second line)",
				},
				{
					Name:        "city",
					Type:        types.String(),
					Nullable:    true,
					Description: "City",
				},
				{
					Name:        "country",
					Type:        types.String(),
					Nullable:    true,
					Description: "Country",
				},
				{
					Name:        "latitude",
					Type:        types.Float(64),
					Nullable:    true,
					Description: "Latitude",
				},
				{
					Name:        "longitude",
					Type:        types.Float(64),
					Nullable:    true,
					Description: "Longitude",
				},
				{
					Name:        "region",
					Type:        types.String(),
					Nullable:    true,
					Description: "Region",
				},
				{
					Name:        "zip",
					Type:        types.String(),
					Nullable:    true,
					Description: "ZIP code",
				},
				{
					Name:        "timezone",
					Type:        types.String(),
					Nullable:    true,
					Description: "Timezone",
				},
				{
					Name:        "ip",
					Type:        types.IP(),
					Nullable:    true,
					Description: "IP address",
				},
			}),
			Nullable:    true,
			Description: "Location",
		},
		{
			Name:        "properties",
			Type:        types.Map(types.JSON()),
			Nullable:    true,
			Description: "Custom properties",
		},
	})
	if role == connectors.Destination {
		sourceOnlyProperties := []string{"id", "anonymous_id", "created", "updated", "last_event_date"}
		schema = types.Filter(schema, func(p types.Property) bool {
			return !slices.Contains(sourceOnlyProperties, p.Name)
		})
	}
	return schema, nil
}

// SendEvents sends events to the API.
func (ky *Klaviyo) SendEvents(ctx context.Context, events connectors.Events) error {
	_, err := ky.sendEvents(ctx, events, false)
	return err
}

// ServeUI serves the connector's user interface.
func (ky *Klaviyo) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

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
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "PrivateAPIKey", Label: "Your Private Key", Placeholder: "pk_62a6ty4674c6bc5df7c252ea4ed2c7ef81", Type: "text", MinLength: 37, MaxLength: 255},
		},
		Settings: settings,
	}

	return ui, nil
}

// Upsert updates or creates records in the API for the specified target.
func (ky *Klaviyo) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records) error {

	record := records.First()

	properties, ok := record.Attributes["properties"]
	if ok {
		delete(record.Attributes, "properties")
	}
	bb := ky.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()
	bb.WriteString(`{"data":{"type":"profile","attributes":`)
	_ = bb.Encode(record.Attributes)
	if ok {
		bb.Truncate(bb.Len() - 1) // remove '}'.
		bb.WriteString(`,"properties":`)
		_ = bb.Encode(properties)
		bb.WriteByte('}') // add '}'.
	}
	if record.ID != "" {
		bb.WriteString(`,"id":`)
		_ = bb.Encode(record.ID)
	}
	bb.WriteString(`}}`)

	u := "https://a.klaviyo.com/api/profiles/"
	if record.ID == "" {
		return ky.call(ctx, "POST", u, bb, 201, nil)
	}

	return ky.call(ctx, "PATCH", u+url.PathEscape(record.ID)+"/", bb, 200, nil)
}

// saveSettings validates and saves the settings.
func (ky *Klaviyo) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	// Klaviyo private key specs are documented here:
	// https://help.klaviyo.com/hc/en-us/articles/360052448451.
	if n := len(s.PrivateAPIKey); n < 37 {
		return connectors.NewInvalidSettingsError("private API key must be at least 37 characters long")
	}
	if !strings.HasPrefix(s.PrivateAPIKey, "pk_") {
		return connectors.NewInvalidSettingsError("private API key must begin with 'pk_'")
	}
	for i := 3; i < len(s.PrivateAPIKey); i++ {
		c := s.PrivateAPIKey[i]
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return connectors.NewInvalidSettingsError("private API key after 'pk_' must contain only alphanumeric characters")
		}
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ky.env.SetSettings(ctx, b)
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

func (ky *Klaviyo) call(ctx context.Context, method, url string, bb *connectors.BodyBuffer, expectedStatus int, response any) error {

	req, err := bb.NewRequest(ctx, method, url)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Klaviyo-API-Key "+ky.settings.PrivateAPIKey)
	req.Header.Set("Revision", apiRevision)

	res, err := ky.env.HTTPClient.Do(req)
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

const maxBodyEventsBytes = 5 * 1024 * 1024
const maxBodyEvents = 1000

var emailRegex = regexp.MustCompile(`(?i)^(?:[a-z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+)*|"(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21\x23-\x5b\x5d-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])*")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21-\x5a\x53-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])+)])$`)

// sendEvents sends the given events to the API and returns the HTTP request
// that was sent. If preview is true, the HTTP request is constructed but not
// sent, and is only returned. If all events are discarded due to validation
// failures, it returns nil for the request.
//
// When preview is true and a non-nil request is returned, the caller is
// responsible for eventually closing the request body.
//
// If an error occurs while sending the events to the API, a nil *http.Request
// and the error are returned.
//
// An event is discarded if it does not satisfy these validations:
//   - The timestamp cannot be before the year 2000 or more than one year in the
//     future.
//   - values["email"] cannot be longer than 100 characters.
//   - values["email"] must match the emailRegex regular expression.
//   - values["currency_code"] must be a valid currency code.
//   - values["properties"] cannot contain more than 400 properties.
//   - A property cannot contain integers outside the 64-bit range.
//   - A property cannot have a depth greater than 9 (10 if including
//     values["properties"]).
//   - A property cannot contain a string longer than 100K characters.
//   - A property cannot contain an array with more than 4000 elements.
func (ky *Klaviyo) sendEvents(ctx context.Context, events connectors.Events, preview bool) (*http.Request, error) {

	now := time.Now().UTC()
	maxTimestamp := time.Date(now.Year()+1, now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.UTC)

	bb := ky.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()

	bb.WriteString(`{"data":{"type":"event-bulk-create-job","attributes":{"events-bulk-create":{"data":[`)

	n := 0
	for event := range events.All() {

		// Validate the event.
		timestamp := event.Received.Timestamp()
		if timestamp.Year() < 2_000 || timestamp.After(maxTimestamp) {
			events.Discard(errors.New("timezone is before 2000 or more than one year in the future"))
			continue
		}
		email := event.Type.Values["email"].(string)
		if utf8.RuneCountInString(email) > 100 {
			events.Discard(errors.New("email is longer than 100 characters"))
			continue
		}
		if !emailRegex.MatchString(email) {
			events.Discard(errors.New("email is not valid"))
			continue
		}
		if currency, ok := event.Type.Values["value_currency"].(string); ok && !validation.IsValidCurrencyCode(currency) {
			events.Discard(errors.New("value_currency is not a valid currency code"))
			continue
		}
		properties, ok := event.Type.Values["properties"].(map[string]any)
		if ok && len(properties) > 400 {
			events.Discard(errors.New("there are more than 400 properties"))
			continue
		}
		for _, v := range properties {
			if err := validateProperty(v.(json.Value)); err != nil {
				events.Discard(err)
				continue
			}
		}

		// Build a unique identifier for the event.
		uniqueId := "[PIPELINE]"
		if !preview {
			uniqueId = strconv.Itoa(event.DestinationPipeline)
		}
		uniqueId += "/" + event.Received.MessageId()

		if n > 0 {
			bb.WriteByte(',')
		}

		bb.WriteString(`{"type":"event-bulk-create","attributes":{"profile":{"data":{"type":"profile","attributes":{`)
		_ = bb.EncodeKeyValue("email", email) // email
		bb.WriteString(`}}},"events":{"data":[`)
		bb.WriteString(`{"type":"event","attributes":{`)
		if properties == nil { // properties
			bb.WriteString(`"properties":{},`)
		} else {
			_ = bb.EncodeKeyValue("properties", properties)
		}
		_ = bb.EncodeKeyValue("time", timestamp) // time
		if value, ok := event.Type.Values["value"]; ok {
			_ = bb.EncodeKeyValue("value", value) // value
		}
		if currency, ok := event.Type.Values["value_currency"]; ok {
			_ = bb.EncodeKeyValue("value_currency", currency) // value_currency
		}
		_ = bb.EncodeKeyValue("unique_id", uniqueId) // unique_id
		bb.WriteString(`,"metric":{"data":{"type":"metric","attributes":{`)
		_ = bb.EncodeKeyValue("name", event.Type.Values["metric_name"]) // metric_name
		bb.WriteString(`}}}}}]}}}`)
		if bb.Len()+len(`]}}}}`) > maxBodyEventsBytes {
			bb.Truncate(0)
			events.Postpone()
			break
		}

		if err := bb.Flush(); err != nil {
			return nil, err
		}

		n++
		if n == maxBodyEvents {
			break
		}

	}
	bb.WriteString(`]}}}}`)

	// Return if all events has been discarded.
	if n == 0 {
		return nil, nil
	}

	req, err := bb.NewRequest(ctx, "POST", "https://a.klaviyo.com/api/event-bulk-create-jobs")
	if err != nil {
		return nil, err
	}

	key := ky.settings.PrivateAPIKey
	if preview {
		key = "[REDACTED]"
	}
	req.Header.Set("Authorization", "Klaviyo-API-Key "+key)
	req.Header.Set("Revision", apiRevision)
	req.Header["Idempotency-Key"] = nil // mark the request as idempotent

	if err = storeHTTPRequestWhenTesting(ctx, req); err != nil {
		return nil, err
	}

	if preview {
		return req, nil
	}

	res, err := ky.env.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 202 {
		return nil, fmt.Errorf("Klaviyo's server returned a %d error", res.StatusCode)
	}

	return req, nil
}

const (
	maxArrayLen  = 4000       // 4,000 elements per array
	maxDepth     = 10 - 1     // max nesting: 9
	maxStringLen = 100 * 1024 // 100K characters
)

var (
	errIntOutOfRange   = errors.New("properties contains an integer that is not within the 64-bit range")
	errMaxDepthReached = errors.New("properties has a depth greater than 10")
	errStringTooLong   = errors.New("properties contains a string longer than 100K characters")
	errTooManyElements = fmt.Errorf("properties contains an array with more than 4000 elements")
)

// validateProperty validates a property and return an error is the property is
// not valid.
func validateProperty(v json.Value) error {

	// Fast path.
	switch v.Kind() {
	case 'n', 'f', 't':
		return nil
	case '0': // Number token
		s := v.String()
		if !strings.Contains(s, ".") && !strings.ContainsAny(s, "eE") {
			if _, err := strconv.ParseInt(s, 10, 64); err != nil {
				return errIntOutOfRange
			}
		}
		return nil
	case '"': // String token
		if utf8.RuneCountInString(v.String()) > maxStringLen {
			return errStringTooLong
		}
		return nil
	}

	var numElems [maxDepth + 1]uint16
	var arrayBitmap uint16
	depth := 0

	isInArray := func(depth int) bool {
		return depth > 0 && (arrayBitmap&(1<<depth) != 0)
	}
	incrementArrayElem := func() error {
		numElems[depth]++
		if numElems[depth] > maxArrayLen {
			return errTooManyElements
		}
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(v))

	for {
		tok, err := dec.ReadToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch tok.Kind() {
		case '[':
			if depth++; depth > maxDepth {
				return errMaxDepthReached
			}
			numElems[depth] = 0
			arrayBitmap |= 1 << depth
		case '{':
			if depth++; depth > maxDepth {
				return errMaxDepthReached
			}
			arrayBitmap &^= 1 << depth
		case ']':
			if depth > 0 {
				depth--
			}
		case '}':
			if depth > 0 {
				depth--
			}
		case '0': // Number token
			s := tok.String()
			if !strings.Contains(s, ".") && !strings.ContainsAny(s, "eE") {
				if _, err := strconv.ParseInt(s, 10, 64); err != nil {
					return errIntOutOfRange
				}
			}
			if isInArray(depth) {
				if err := incrementArrayElem(); err != nil {
					return err
				}
			}
		case '"': // String token
			if utf8.RuneCountInString(tok.String()) > maxStringLen {
				return errStringTooLong
			}
			if isInArray(depth) {
				if err := incrementArrayElem(); err != nil {
					return err
				}
			}
		default:
			if isInArray(depth) {
				if err := incrementArrayElem(); err != nil {
					return err
				}
			}
		}
	}

	return nil
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
func storeHTTPRequestWhenTesting(ctx context.Context, req *http.Request) error {
	stored := ctx.Value(connectorTestString("storeSentHTTPRequest"))
	if stored == nil {
		return nil
	}
	storedReq := req.Clone(req.Context())
	body, err := req.GetBody()
	if err != nil {
		return err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	storedReq.Body = io.NopCloser(bytes.NewReader(data))
	*(stored.(*http.Request)) = *storedReq
	return nil
}
