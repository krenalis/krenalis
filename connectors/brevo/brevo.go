// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package brevo provides a connector for Brevo.
// (https://developers.brevo.com/)
//
// Brevo is a trademark of Sendinblue SAS.
// This connector is not affiliated with or endorsed by Sendinblue SAS.
package brevo

import (
	"cmp"
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
	"sync"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterApplication(connectors.ApplicationSpec{
		Code:       "brevo",
		Label:      "Brevo",
		Categories: connectors.CategorySaaS,
		AsSource: &connectors.AsApplicationSource{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Import contacts as users from Brevo",
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsApplicationDestination{
			Targets:     connectors.TargetEvent | connectors.TargetUser,
			HasSettings: true,
			SendingMode: connectors.Server,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Export users as contacts and send events to Brevo",
				Overview: destinationOverview,
			},
		},
		Terms: connectors.ApplicationTerms{
			User:   "Contact",
			Users:  "Contacts",
			UserID: "Brevo ID",
		},
	}, New)
}

// New returns a new connector instance for Brevo.
func New(env *connectors.ApplicationEnv) (*Brevo, error) {
	b := Brevo{env: env}
	if len(env.Settings) > 0 {
		err := env.Settings.Unmarshal(&b.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for Brevo")
		}
	}
	return &b, nil
}

type Brevo struct {
	env      *connectors.ApplicationEnv
	settings *innerSettings

	mu                 sync.Mutex
	cachedSourceAttrs  types.Type
	cachedDestAttrs    types.Type
	cachedAttributesAt time.Time
}

type innerSettings struct {
	APIKey string `json:"apiKey"`
}

var apiBaseURL = "https://api.brevo.com/v3"

type brevoError struct {
	StatusCode int
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Metadata   map[string]any `json:"metadata"`
}

// Error returns the Brevo API error in a user-facing form.
func (err *brevoError) Error() string {
	if err == nil {
		return "<nil>"
	}
	switch {
	case err.Code != "" && err.Message != "":
		// TODO: please review whether this upstream API error may contain PII
		return fmt.Sprintf("Brevo error %q: %s", err.Code, err.Message)
	case err.Message != "":
		// TODO: please review whether this upstream API error may contain PII
		return "Brevo error: " + err.Message
	case err.Code != "":
		return "Brevo error: " + err.Code
	default:
		return fmt.Sprintf("Brevo returned HTTP %d", err.StatusCode)
	}
}

type batchEventsResponse struct {
	Status           string `json:"status"`
	TotalEvents      int    `json:"total_events"`
	SuccessfulEvents int    `json:"successful_events"`
	FailedEvents     int    `json:"failed_events"`
	Errors           []struct {
		EventIndex []int  `json:"eventIndex"`
		Message    string `json:"message"`
	} `json:"errors"`
}

type contactInfo struct {
	Attributes       map[string]any `json:"attributes"`
	CreatedAt        string         `json:"createdAt"`
	Email            string         `json:"email"`
	EmailBlacklisted bool           `json:"emailBlacklisted"`
	ID               int            `json:"id"`
	ListIDs          []any          `json:"listIds"`
	ListUnsubscribed []any          `json:"listUnsubscribed"`
	ModifiedAt       string         `json:"modifiedAt"`
	SMSBlacklisted   bool           `json:"smsBlacklisted"`
}

// EventTypes returns the event types.
func (brevo *Brevo) EventTypes(ctx context.Context) ([]*connectors.EventType, error) {
	return []*connectors.EventType{{
		ID:          "create_event",
		Name:        "Create event",
		Description: "Create a Brevo event",
	}}, nil
}

var eventNameRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// EventTypeSchema returns the schema of the specified event type.
func (brevo *Brevo) EventTypeSchema(ctx context.Context, eventType string) (types.Type, error) {
	if eventType != "create_event" {
		return types.Type{}, connectors.ErrEventTypeNotExist
	}
	return types.Object([]types.Property{
		{
			Name:           "event_name",
			Type:           types.String().WithMaxLength(255).WithPattern(eventNameRE),
			CreateRequired: true,
			Description:    "Event name",
		},
		{
			Name:        "event_properties",
			Type:        types.Map(types.JSON()),
			Prefilled:   "properties",
			Description: "Event properties",
		},
		{
			Name: "identifiers",
			Type: types.Object([]types.Property{
				{
					Name:        "contact_id",
					Type:        types.Int(64),
					Description: "Internal Brevo contact ID; it takes precedence over all other identifiers",
				},
				{
					Name:        "email_id",
					Type:        types.String(),
					Description: "Email address",
				},
				{
					Name:        "ext_id",
					Type:        types.String(),
					Description: "External identifier",
				},
				{
					Name:        "phone_id",
					Type:        types.String(),
					Description: "SMS identifier",
				},
				{
					Name:        "whatsapp_id",
					Type:        types.String(),
					Description: "WhatsApp identifier",
				},
				{
					Name:        "landline_number_id",
					Type:        types.String(),
					Description: "Landline identifier",
				},
			}),
			CreateRequired: true,
			Description:    "Contact identifiers associated with the event; at least one is required",
		},
		{
			Name:        "contact_properties",
			Type:        types.Map(types.JSON()),
			Prefilled:   "traits",
			Description: "Contact properties to update alongside the event",
		},
		{
			Name: "object",
			Type: types.Object([]types.Property{
				{
					Name:        "type",
					Type:        types.String(),
					Description: "Associated object type (e.g., subscription)",
				},
				{
					Name: "identifiers",
					Type: types.Object([]types.Property{
						{
							Name:        "ext_id",
							Type:        types.String(),
							Description: "External object ID",
						},
						{
							Name:        "id",
							Type:        types.String(),
							Description: "Internal object ID",
						},
					}),
					Description: "Associated object identifiers",
				},
			}),
			Description: "Optional object associated with the event",
		},
	}), nil
}

// PreviewSendEvents returns the HTTP request that would be used to send the
// events to the API, without actually sending it.
func (brevo *Brevo) PreviewSendEvents(ctx context.Context, events connectors.Events) (*http.Request, error) {
	return brevo.sendEvents(ctx, events, true)
}

// RecordSchema returns the schema of the specified target and role.
func (brevo *Brevo) RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {

	var response struct {
		Attributes []struct {
			Category        string `json:"category"`
			Name            string `json:"name"`
			CalculatedValue string `json:"calculatedValue"`
			Type            string `json:"type"`
			Enumeration     []struct {
				Label string `json:"label"`
				Value int    `json:"value"`
			} `json:"enumeration"`
			MultiCategoryOptions []string `json:"multiCategoryOptions"`
		} `json:"attributes"`
	}
	err := brevo.call(ctx, http.MethodGet, apiBaseURL+"/contacts/attributes", nil, http.StatusOK, &response)
	if err != nil {
		return types.Type{}, err
	}

	var properties []types.Property

	if role == connectors.Source {
		properties = []types.Property{
			{
				Name:        "id",
				Type:        types.Int(64),
				Description: "Brevo contact ID",
			},
		}
	}

	attributes := []types.Property{
		{
			Name:        "EMAIL",
			Type:        types.String(),
			Nullable:    true,
			Description: "Email address",
		},
	}

	for _, attr := range response.Attributes {

		if !types.IsValidPropertyName(attr.Name) {
			continue
		}
		if role == connectors.Destination && attr.CalculatedValue != "" {
			continue
		}
		switch attr.Category {
		case "normal", "category", "calculated", "global":
		default:
			// Skip transactional attributes.
			continue
		}

		attribute := types.Property{
			Name:     attr.Name,
			Nullable: true,
		}
		switch attr.Name {
		case "EXT_ID":
			attribute.Description = "External identifier"
		case "FIRSTNAME":
			attribute.Description = "First name"
		case "LASTNAME":
			attribute.Description = "Last name"
		case "LANDLINE_NUMBER":
			attribute.Description = "Landline phone number"
		case "SMS":
			attribute.Description = "Phone number used for SMS, including country code"
		case "WHATSAPP":
			attribute.Description = "Phone number used for WhatsApp messages, including country code"
		default:
			attribute.Description = formatAttributeDescription(attr.Name)
		}

		switch attr.Type {
		case "text":
			attribute.Type = types.String().WithMaxLength(50000)
			attribute.Nullable = false
		case "date":
			attribute.Type = types.Date()
		case "float":
			attribute.Type = types.Decimal(15, 4)
		case "id", "user":
			attribute.Type = types.String()
		case "boolean":
			attribute.Type = types.Boolean()
		case "multiple-choice":
			if len(attr.MultiCategoryOptions) == 0 {
				return types.Type{}, fmt.Errorf("Brevo returned an empty options for attribute %q", attr.Name)
			}
			options := slices.Clone(attr.MultiCategoryOptions)
			slices.Sort(options)
			options = slices.Compact(options)
			attribute.Type = types.Array(types.String().WithValues(options...))
		default:
			if attr.Category != "category" {
				continue
			}
			if len(attr.Enumeration) == 0 {
				return types.Type{}, fmt.Errorf("Brevo returned an empty enumeration for attribute %q", attr.Name)
			}
			var description strings.Builder
			description.WriteString(attribute.Description)
			description.WriteString(`; allowed values: "`)
			values := make([]string, 0, len(attr.Enumeration))
			for i, option := range attr.Enumeration {
				value := strconv.Itoa(option.Value)
				if slices.Contains(values, value) {
					continue
				}
				values = append(values, value)
				if i > 0 {
					description.WriteString(`, "`)
				}
				description.WriteString(value)
				description.WriteString(`" = `)
				description.WriteString(strconv.Quote(option.Label))
			}
			attribute.Type = types.String().WithValues(values...)
			attribute.Description = description.String()
		}

		attributes = append(attributes, attribute)

	}

	slices.SortFunc(attributes, func(a, b types.Property) int {
		return cmp.Compare(a.Name, b.Name)
	})
	properties = append(properties, attributes...)

	if role == connectors.Source {
		properties = append(properties, []types.Property{
			{
				Name:        "listUnsubscribed",
				Type:        types.Array(types.Int(64)),
				Description: "IDs of the lists unsubscribed from",
			},
			{
				Name:        "createdAt",
				Type:        types.DateTime(),
				Description: "Creation timestamp",
			},
			{
				Name:        "modifiedAt",
				Type:        types.DateTime(),
				Description: "Last modification timestamp",
			},
		}...)
	}

	properties = append(properties, []types.Property{
		{
			Name:        "emailBlacklisted",
			Type:        types.Boolean(),
			Description: "Whether blacklisted from receiving emails",
		},
		{
			Name:        "smsBlacklisted",
			Type:        types.Boolean(),
			Description: "Whether blacklisted from receiving SMS messages",
		},
		{
			Name:        "listIds",
			Type:        types.Array(types.Int(64)),
			Description: "IDs of the lists it belongs to",
		},
	}...)

	return types.Object(properties), nil
}

const recordsPageLimit = 1000

// Records returns the records of the specified target.
func (brevo *Brevo) Records(ctx context.Context, target connectors.Targets, updatedAt time.Time, cursor string, schema types.Type) ([]connectors.Record, string, error) {
	offset := 0
	if cursor != "" {
		var err error
		offset, err = strconv.Atoi(cursor)
		if err != nil || offset < 0 {
			return nil, "", errors.New("invalid Brevo cursor")
		}
	}

	query := url.Values{
		"limit":  []string{strconv.Itoa(recordsPageLimit)},
		"offset": []string{strconv.Itoa(offset)},
		"sort":   []string{"asc"},
	}
	if !updatedAt.IsZero() {
		query.Set("modifiedSince", updatedAt.Format(time.RFC3339Nano))
	}

	var response struct {
		Contacts []contactInfo `json:"contacts"`
		Count    int           `json:"count"`
	}
	err := brevo.call(ctx, http.MethodGet, apiBaseURL+"/contacts?"+query.Encode(), nil, http.StatusOK, &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Contacts) == 0 {
		return nil, "", io.EOF
	}

	records := make([]connectors.Record, len(response.Contacts))

	for i, contact := range response.Contacts {
		record := connectors.Record{
			ID: strconv.Itoa(contact.ID),
		}
		if contact.ID <= 0 {
			record.Err = errors.New("Brevo has returned an invalid «id»")
			records[i] = record
			continue
		}
		record.Attributes = map[string]any{
			"id":               contact.ID,
			"EMAIL":            contact.Email,
			"emailBlacklisted": contact.EmailBlacklisted,
			"smsBlacklisted":   contact.SMSBlacklisted,
			"listIds":          contact.ListIDs,
			"createdAt":        contact.CreatedAt,
			"modifiedAt":       contact.ModifiedAt,
			"listUnsubscribed": contact.ListUnsubscribed,
		}
		if contact.Email == "" {
			record.Attributes["EMAIL"] = nil
		}
		if contact.ListUnsubscribed == nil {
			contact.ListUnsubscribed = []any{}
		}
		for _, p := range schema.Properties().All() {
			switch p.Name {
			case "id", "EMAIL", "listUnsubscribed", "createdAt", "modifiedAt", "emailBlacklisted", "smsBlacklisted", "listIds":
				continue
			}
			v, ok := contact.Attributes[p.Name]
			if !ok && p.Type.Kind() == types.StringKind {
				v = ""
			}
			record.Attributes[p.Name] = v
		}
		modifiedAt, err := time.Parse(time.RFC3339, contact.ModifiedAt)
		if err != nil {
			record.Err = errors.New("Brevo has returned an invalid «modifiedAt» value")
			records[i] = record
			continue
		}
		record.UpdatedAt = modifiedAt
		records[i] = record
	}

	nextOffset := offset + len(response.Contacts)
	if nextOffset >= response.Count {
		return records, "", io.EOF
	}

	return records, strconv.Itoa(nextOffset), nil
}

// SendEvents sends events to the API.
func (brevo *Brevo) SendEvents(ctx context.Context, events connectors.Events) error {
	_, err := brevo.sendEvents(ctx, events, false)
	return err
}

// ServeUI serves the connector's user interface.
func (brevo *Brevo) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {
	switch event {
	case "load":
		var s innerSettings
		if brevo.settings != nil {
			s = *brevo.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, brevo.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	return &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{
				Name:        "apiKey",
				Label:       "Your API key",
				Type:        "password",
				MinLength:   9,
				MaxLength:   255,
				Placeholder: "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",
			},
		},
		Settings: settings,
	}, nil
}

// Upsert updates or creates records in the API for the specified target.
func (brevo *Brevo) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records, schema types.Type) error {

	r := records.First()
	if r.IsCreate() && !hasCreateIdentifiers(r.Attributes) {
		return connectors.RecordsError{0: errors.New("creating a Brevo contact requires one of: «EMAIL», «EXT_ID», «SMS», «LANDLINE_NUMBER», or «WHATSAPP»")}
	}

	var attributes map[string]any

	bb := brevo.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()

	bb.WriteByte('{')
	for name, value := range r.Attributes {
		if r.IsCreate() && value == nil {
			continue
		}
		switch name {
		case "emailBlacklisted", "smsBlacklisted", "listIds":
			_ = bb.EncodeKeyValue(name, value)
		case "EMAIL":
			if r.IsCreate() {
				_ = bb.EncodeKeyValue("email", value)
			}
			fallthrough
		default:
			if value == nil {
				value = ""
			}
			if attributes == nil {
				attributes = map[string]any{name: value}
			} else {
				attributes[name] = value
			}
		}
	}
	if attributes != nil {
		_ = bb.EncodeKeyValue("attributes", attributes)
	}
	bb.WriteByte('}')

	var err error
	if r.IsCreate() {
		err = brevo.call(ctx, http.MethodPost, apiBaseURL+"/contacts", bb, http.StatusCreated, nil)
	} else {
		err = brevo.call(ctx, http.MethodPut, apiBaseURL+"/contacts/"+url.PathEscape(r.ID)+"?identifierType=contact_id", bb, http.StatusNoContent, nil)
	}
	if err != nil {
		if err, ok := err.(*brevoError); ok {
			switch err.StatusCode {
			case http.StatusBadRequest, http.StatusNotFound, http.StatusTooEarly:
				return connectors.RecordsError{0: err}
			}
		}
		return err
	}

	return nil
}

// call sends an authenticated request to the Brevo API and decodes the expected
// response.
func (brevo *Brevo) call(ctx context.Context, method, url string, body *connectors.BodyBuffer, expectedStatus int, response any) error {

	req, err := body.NewRequest(ctx, method, url)
	if err != nil {
		return err
	}

	req.Header.Set("Api-Key", brevo.settings.APIKey)

	res, err := brevo.env.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != expectedStatus {
		apiErr := &brevoError{StatusCode: res.StatusCode}
		err := json.Decode(res.Body, apiErr)
		if err != nil {
			return err
		}
		return apiErr
	}
	if response != nil && res.StatusCode != http.StatusNoContent {
		return json.Decode(res.Body, response)
	}

	return nil
}

// saveSettings validates and stores the connector settings.
func (brevo *Brevo) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	if n := len(s.APIKey); n < 1 || n > 255 {
		return connectors.NewInvalidSettingsError("«api_key» length must be in range [1,255]")
	}
	for i := 0; i < len(s.APIKey); i++ {
		c := s.APIKey[i]
		if c <= ' ' || c == 0x7f {
			return connectors.NewInvalidSettingsError("«api_key» must not contain spaces or control characters")
		}
	}
	// Check whether the configured API key can access the Brevo account endpoint.
	previous := brevo.settings
	brevo.settings = &s
	if err := brevo.call(ctx, http.MethodGet, apiBaseURL+"/account", nil, http.StatusOK, nil); err != nil {
		brevo.settings = previous
		if err, ok := err.(*brevoError); ok {
			if err.StatusCode == http.StatusUnauthorized || err.StatusCode == http.StatusForbidden {
				return connectors.NewInvalidSettingsError("«api_key» is not valid or cannot access the Brevo API")
			}
		}
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		brevo.settings = previous
		return err
	}
	if err := brevo.env.SetSettings(ctx, b); err != nil {
		brevo.settings = previous
		return err
	}
	return nil
}

const (
	maxEventsPerRequest    = 200
	maxRequestBodySize     = 512 * 1024 // 512 KB
	maxEventPropertiesSize = 50 * 1024  // 50 KB
)

// sendEvents sends or previews a batch of Brevo events.
func (brevo *Brevo) sendEvents(ctx context.Context, events connectors.Events, preview bool) (*http.Request, error) {

	// See https://developers.brevo.com/docs/event-endpoints and https://developers.brevo.com/docs/event-endpoints#create-events-in-batch.

	bb := brevo.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()

	bb.WriteString(`{"events":[`)

	n := 0
Events:
	for event := range events.All() {

		values := event.Type.Values

		// Validate identifiers.
		identifiers, _ := values["identifiers"].(map[string]any)
		if !hasSendEventsIdentifiers(identifiers) {
			events.Discard(errors.New("«identifiers» does not contain any contact identifier"))
			continue
		}

		// Validate contact properties.
		if pp, ok := values["contact_properties"].(map[string]any); ok {
			for name, value := range pp {
				if name == "" {
					events.Discard(errors.New("a key in «contact_properties» is empty"))
				}
				// Ensure the value is a string, number, or boolean.
				// Brevo documentation incorrectly states string or integer.
				v, _ := value.(json.Value)
				switch v.Kind() {
				case json.String, json.Number, json.True, json.False:
				default:
					events.Discard(errors.New("a value in «event_properties» is not a string, number, or boolean"))
					continue Events
				}
			}
		}

		// Validate event properties.
		if pp, ok := values["event_properties"].(map[string]any); ok {
			for key, _ := range pp {
				if key == "" {
					events.Discard(errors.New("a key in «event_properties» is empty"))
					continue Events
				}
				if utf8.RuneCountInString(key) > 255 {
					events.Discard(errors.New("a key in «event_properties» exceeds 255 characters"))
					continue Events
				}
				// Brevo documentation states that the value can be a string, number, object, or array,
				// but it also accepts booleans (converted to "0" or "1") and ignores null values.
				// Given these inconsistencies, send any value to Brevo.
			}
		}

		// Track size before adding the event.
		size := bb.Len()

		if n > 0 {
			bb.WriteByte(',')
		}
		bb.WriteByte('{')
		_ = bb.EncodeKeyValue("event_name", values["event_name"])
		_ = bb.EncodeKeyValue("identifiers", identifiers)
		if pp, ok := values["contact_properties"]; ok {
			_ = bb.EncodeKeyValue("contact_properties", pp)
		}
		_ = bb.EncodeKeyValue("event_date", event.Received.Timestamp().Format(time.RFC3339Nano))
		if pp, ok := values["event_properties"]; ok {
			previousSize := bb.Len()
			_ = bb.EncodeKeyValue("event_properties", pp)
			if bb.Len()-previousSize > maxEventPropertiesSize {
				bb.Truncate(size)
				events.Discard(errors.New("event properties exceed the 50 KB size limit"))
				continue Events
			}
		}
		if object, ok := values["object"].(map[string]any); ok && len(object) > 0 {
			_ = bb.EncodeKeyValue("object", object)
		}
		bb.WriteByte('}')

		// Stop if body exceeds 512 KB.
		if bb.Len()+len(`]}`) > maxRequestBodySize {
			bb.Truncate(size)
			events.Postpone()
			break
		}

		if err := bb.Flush(); err != nil {
			return nil, err
		}

		n++
		if n == maxEventsPerRequest {
			break
		}

	}

	if n == 0 {
		return nil, nil
	}

	bb.WriteString(`]}`)

	req, err := bb.NewRequest(ctx, http.MethodPost, apiBaseURL+"/events/batch")
	if err != nil {
		return nil, err
	}

	key := brevo.settings.APIKey
	if preview {
		key = "[REDACTED]"
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Api-Key", key)
	req.Header["Idempotency-Key"] = nil // mark the request as idempotent

	if preview {
		return req, nil
	}

	res, err := brevo.env.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {

		// Read the response body, capped at 50 KB.
		// This is more than enough to include around 200 failed events.
		body, err := io.ReadAll(io.LimitReader(res.Body, 50*1024))
		if err != nil {
			return nil, err
		}

		if res.StatusCode == http.StatusMultiStatus || res.StatusCode == http.StatusBadRequest {
			var response batchEventsResponse
			err = json.Unmarshal(body, &response)
			if err == nil && response.Status != "" {
				eventsErr := connectors.EventsError{}
				for _, e := range response.Errors {
					message := e.Message
					if message == "" {
						message = "Brevo rejected the event"
					}
					// TODO: please review whether this upstream API error may contain PII
					err = errors.New(message)
					for _, i := range e.EventIndex {
						eventsErr[i] = err
					}
				}
				if len(eventsErr) > 0 {
					return nil, eventsErr
				}
			}
		}

		var apiErr brevoError
		err = json.Unmarshal(body, &apiErr)
		if err == nil && apiErr.Code != "" {
			return nil, fmt.Errorf("%s: %s", apiErr.Code, apiErr.Message)
		}

		return nil, fmt.Errorf("unexpected HTTP status code returned by Brevo: %d", res.StatusCode)
	}

	return req, nil
}

// hasCreateIdentifiers checks whether a contact create payload has a supported
// identifier.
func hasCreateIdentifiers(attributes map[string]any) bool {
	if email, ok := attributes["EMAIL"]; ok && email != nil {
		return true
	}
	if _, ok := attributes["EXT_ID"]; ok {
		return true
	}
	if _, ok := attributes["SMS"]; ok {
		return true
	}
	if _, ok := attributes["LANDLINE_NUMBER"]; ok {
		return true
	}
	if _, ok := attributes["WHATSAPP"]; ok {
		return true
	}
	return false
}

// hasSendEventsIdentifiers checks whether a send events payload has a supported
// identifier.
func hasSendEventsIdentifiers(identifiers map[string]any) bool {
	if _, ok := identifiers["contact_id"]; ok {
		return true
	}
	if _, ok := identifiers["email_id"]; ok {
		return true
	}
	if _, ok := identifiers["ext_id"]; ok {
		return true
	}
	if _, ok := identifiers["landline_number_id"]; ok {
		return true
	}
	if _, ok := identifiers["phone_id"]; ok {
		return true
	}
	if _, ok := identifiers["whatsapp_id"]; ok {
		return true
	}
	return false
}

// formatAttributeDescription formats an attribute name in UPPER_SNAKE_CASE
// (e.g. "EXT_ID") into a human-readable description such as "Ext id".
func formatAttributeDescription(name string) string {
	b := make([]byte, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '_' {
			b[i] = ' '
			continue
		}
		if i == 0 {
			b[i] = c
		} else {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
