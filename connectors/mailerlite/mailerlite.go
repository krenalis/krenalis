// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package mailerlite provides a connector for MailerLite.
// (https://developers.mailerlite.com/)
//
// MailerLite is a trademark of MailerLite Limited.
// This connector is not affiliated with or endorsed by MailerLite Limited.
package mailerlite

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

var apiBaseURL = "https://connect.mailerlite.com/api"

const (
	apiVersion            = "2026-05-22"
	legacyAPIKeyLength    = 32
	subscribersPageLimit  = "100"
	settingsAPIKeyMaxSize = 4096
)

const (
	// Parse MailerLite number fields whenever they fit in Krenalis' decimal representation.
	numberPrecision = types.MaxDecimalPrecision
	numberScale     = types.MaxDecimalScale
)

func init() {
	connectors.RegisterApplication(connectors.ApplicationSpec{
		Code:       "mailerlite",
		Label:      "MailerLite",
		Categories: connectors.CategorySaaS,
		AsSource: &connectors.AsApplicationSource{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Import subscribers as users from MailerLite",
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsApplicationDestination{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Export profiles as subscribers to MailerLite",
				Overview: destinationOverview,
			},
		},
		Terms: connectors.ApplicationTerms{
			User:   "Subscriber",
			Users:  "Subscribers",
			UserID: "Subscriber ID",
		},
		EndpointGroups: []connectors.EndpointGroup{{
			// MailerLite documents a global limit of 120 requests/minute.
			// https://developers.mailerlite.com/docs/#rate-limits
			RateLimit: connectors.RateLimit{RequestsPerSecond: 2, Burst: 2},
			RetryPolicy: connectors.RetryPolicy{
				"429":             connectors.RetryAfterStrategy(),
				"408 500 502 503": connectors.ExponentialStrategy(connectors.NetFailure, 100*time.Millisecond),
			},
		}},
		TimeLayouts: connectors.TimeLayouts{
			DateTime: time.DateTime,
		},
	}, New)
}

// New returns a new connector instance for MailerLite.
func New(env *connectors.ApplicationEnv) (*MailerLite, error) {
	return &MailerLite{env: env}, nil
}

type MailerLite struct {
	env *connectors.ApplicationEnv
}

type innerSettings struct {
	APIKey string `json:"apiKey"`
}

type field struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	Type string `json:"type"`
}

// RecordSchema returns the schema of the specified target and role.
func (ml *MailerLite) RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {

	callURL := apiBaseURL + "/fields?limit=100&page="

	var fields []types.Property

	for page := 1; ; page++ {
		var response struct {
			Data []field `json:"data"`
			Meta struct {
				CurrentPage int `json:"current_page"`
				LastPage    int `json:"last_page"`
			} `json:"meta"`
		}
		err := ml.call(ctx, http.MethodGet, callURL+strconv.Itoa(page), nil, http.StatusOK, &response)
		if err != nil {
			return types.Type{}, err
		}
	FIELDS:
		for _, field := range response.Data {
			if !types.IsValidPropertyName(field.Key) {
				continue
			}
			for _, f := range fields {
				if f.Name == field.Key {
					continue FIELDS
				}
			}
			var typ types.Type
			switch field.Type {
			case "text":
				typ = types.String()
			case "number":
				typ = types.Decimal(numberPrecision, numberScale)
			case "date":
				typ = types.Date()
			default:
				continue
			}
			fields = append(fields, types.Property{
				Name:        field.Key,
				Type:        typ,
				Nullable:    true,
				Description: field.Name,
			})
		}
		if response.Meta.LastPage == 0 || response.Meta.CurrentPage >= response.Meta.LastPage {
			break
		}
	}

	var properties []types.Property
	if role == connectors.Source {
		properties = append(properties, types.Property{Name: "id", Type: types.String(), Description: "Subscriber ID"})
	}
	properties = append(properties,
		types.Property{Name: "email", Type: types.String(), CreateRequired: true, Description: "Email address, used only when creating a subscriber"},
		types.Property{Name: "groups", Type: types.Array(types.String()), ReadOptional: true, Description: "Group IDs"},
	)
	if len(fields) > 0 {
		properties = append(properties, types.Property{
			Name:        "fields",
			Type:        types.Object(fields),
			Description: "Subscriber fields",
		})
	}
	properties = append(properties,
		types.Property{Name: "status", Type: subscriberStatusType(), Description: "Subscriber status"},
		types.Property{Name: "source", Type: types.String(), ReadOptional: true, Description: "Subscriber source"},
		types.Property{Name: "sent", Type: types.Int(32), ReadOptional: true, Description: "Number of sent emails"},
		types.Property{Name: "opens_count", Type: types.Int(32), ReadOptional: true, Description: "Number of opens"},
		types.Property{Name: "clicks_count", Type: types.Int(32), ReadOptional: true, Description: "Number of clicks"},
		types.Property{Name: "open_rate", Type: types.Float(64), ReadOptional: true, Description: "Open rate"},
		types.Property{Name: "click_rate", Type: types.Float(64), ReadOptional: true, Description: "Click rate"},
		types.Property{Name: "ip_address", Type: types.IP(), Nullable: true, ReadOptional: true, Description: "Subscriber IP address"},
		types.Property{Name: "subscribed_at", Type: types.DateTime(), Nullable: true, ReadOptional: true, Description: "Subscription timestamp"},
		types.Property{Name: "unsubscribed_at", Type: types.DateTime(), Nullable: true, ReadOptional: true, Description: "Unsubscribe timestamp"},
		types.Property{Name: "created_at", Type: types.DateTime(), Description: "Creation timestamp"},
		types.Property{Name: "updated_at", Type: types.DateTime(), Description: "Last update timestamp"},
		types.Property{Name: "opted_in_at", Type: types.DateTime(), Nullable: true, ReadOptional: true, Description: "Opt-in timestamp"},
		types.Property{Name: "optin_ip", Type: types.IP(), Nullable: true, ReadOptional: true, Description: "Opt-in IP address"},
	)
	if role == connectors.Destination {
		properties = append(properties,
			types.Property{Name: "resubscribe", Type: types.Boolean(), Description: "Resubscribe a previously unsubscribed subscriber when allowed by MailerLite"},
		)
	}

	return types.AsRole(types.Object(properties), types.Role(role)), nil
}

// Records returns the records of the specified target.
func (ml *MailerLite) Records(ctx context.Context, target connectors.Targets, updatedAt time.Time, cursor string, schema types.Type) ([]connectors.Record, string, error) {

	requestURL := apiBaseURL + "/subscribers?limit=" + subscribersPageLimit + "&include=groups"
	if cursor != "" {
		requestURL += "&cursor=" + url.QueryEscape(cursor)
	}

	var response struct {
		Data []subscriber `json:"data"`
		Meta struct {
			NextCursor string `json:"next_cursor"`
		} `json:"meta"`
	}
	err := ml.call(ctx, http.MethodGet, requestURL, nil, http.StatusOK, &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Data) == 0 {
		return nil, "", io.EOF
	}

	records := make([]connectors.Record, 0, len(response.Data))
	for _, subscriber := range response.Data {
		record := ml.recordFromSubscriber(subscriber, schema)
		if record.Err != nil || updatedAt.IsZero() || !record.UpdatedAt.Before(updatedAt) {
			records = append(records, record)
		}
	}
	if response.Meta.NextCursor == "" {
		return records, "", io.EOF
	}

	return records, response.Meta.NextCursor, nil
}

// ServeUI serves the connector's user interface.
func (ml *MailerLite) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if err := ml.env.Settings.Load(ctx, &s); err != nil {
			return nil, err
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, ml.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	return &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{
				Name:      "apiKey",
				Label:     "API token",
				Rows:      6,
				MinLength: legacyAPIKeyLength + 1,
				MaxLength: settingsAPIKeyMaxSize,
			},
		},
		Settings: settings,
		Buttons:  []connectors.Button{connectors.SaveButton},
	}, nil
}

var datetimePropertyNames = []string{"subscribed_at", "unsubscribed_at", "created_at", "updated_at", "opted_in_at"}

// Upsert updates or creates records in the API for the specified target.
func (ml *MailerLite) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records, schema types.Type) error {

	record := records.First()
	if record.IsCreate() {
		if _, ok := record.Attributes["email"]; !ok {
			return connectors.RecordsError{0: errors.New("creating a MailerLite subscriber requires email")}
		}
	} else {
		delete(record.Attributes, "email")
	}

	// Convert time.Time values to strings.
	for _, name := range datetimePropertyNames {
		if t, ok := record.Attributes[name]; ok {
			if t, ok := t.(time.Time); ok {
				record.Attributes[name] = t.Format(time.DateTime)
			}
		}
	}
	if fields, ok := record.Attributes["fields"].(map[string]any); ok {
		for k, v := range fields {
			if t, ok := v.(time.Time); ok {
				fields[k] = t.Format(time.DateTime)
			}
		}
	}

	bb := ml.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()

	err := bb.Encode(record.Attributes)
	if err != nil {
		return err
	}

	var requestMethod string
	var requestURL string
	var expectedStatus int
	if record.IsCreate() {
		requestMethod = http.MethodPost
		requestURL = apiBaseURL + "/subscribers"
		expectedStatus = http.StatusCreated
	} else {
		requestMethod = http.MethodPut
		requestURL = apiBaseURL + "/subscribers/" + url.PathEscape(record.ID)
		expectedStatus = http.StatusOK
	}

	err = ml.call(ctx, requestMethod, requestURL, bb, expectedStatus, nil)
	if err != nil {
		if err, ok := err.(*apiError); ok {
			switch err.StatusCode {
			case http.StatusBadRequest, http.StatusNotFound, http.StatusUnprocessableEntity:
				return connectors.RecordsError{0: err}
			}
		}
		return err
	}

	return nil
}

func (ml *MailerLite) call(ctx context.Context, method, u string, body *connectors.BodyBuffer, expectedStatus int, response any) error {
	var s innerSettings
	if err := ml.env.Settings.Load(ctx, &s); err != nil {
		return err
	}
	return ml.callWithSettings(ctx, s, method, u, body, expectedStatus, response)
}

func (ml *MailerLite) callWithSettings(ctx context.Context, settings innerSettings, method, u string, body *connectors.BodyBuffer, expectedStatus int, response any) error {

	req, err := body.NewRequest(ctx, method, u)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+settings.APIKey)
	req.Header.Set("X-Version", apiVersion)

	res, err := ml.env.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != expectedStatus {
		var apiErr apiError
		err := json.Decode(res.Body, &apiErr)
		if err != nil {
			return err
		}
		apiErr.StatusCode = res.StatusCode
		return &apiErr
	}
	if response != nil {
		return json.Decode(res.Body, response)
	}

	return nil
}

func (ml *MailerLite) recordFromSubscriber(s subscriber, schema types.Type) connectors.Record {
	if s.ID == "" {
		return connectors.Record{Err: errors.New("MailerLite returned a subscriber without an id")}
	}
	record := connectors.Record{
		ID: s.ID,
		Attributes: map[string]any{
			"id":              s.ID,
			"email":           s.Email,
			"groups":          subscriberGroupsIDs(s.Groups),
			"status":          s.Status,
			"source":          s.Source,
			"sent":            s.Sent,
			"opens_count":     s.OpensCount,
			"clicks_count":    s.ClicksCount,
			"open_rate":       s.OpenRate,
			"click_rate":      s.ClickRate,
			"ip_address":      s.IPAddress,
			"subscribed_at":   s.SubscribedAt,
			"unsubscribed_at": s.UnsubscribedAt,
			"created_at":      s.CreatedAt,
			"updated_at":      s.UpdatedAt,
			"opted_in_at":     s.OptedInAt,
			"optin_ip":        s.OptinIP,
		},
	}
	var err error
	record.UpdatedAt, err = time.Parse(time.DateTime, s.UpdatedAt)
	if err != nil {
		record.Err = err
		return record
	}
	p, ok := schema.Properties().ByName("fields")
	if !ok {
		return record
	}
	fields := make(map[string]any, p.Type.Properties().Len())
	for _, f := range p.Type.Properties().All() {
		if value, ok := s.Fields[f.Name]; ok {
			if f.Type.Kind() == types.DecimalKind {
				fields[f.Name] = parseNumber(value)
			} else {
				fields[f.Name] = value
			}
		} else {
			fields[f.Name] = nil
		}
	}
	record.Attributes["fields"] = fields
	return record
}

func (ml *MailerLite) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	if err := settings.Unmarshal(&s); err != nil {
		return err
	}
	if s.APIKey == "" {
		return connectors.NewInvalidSettingsError("apiKey is required")
	}
	if len(s.APIKey) > settingsAPIKeyMaxSize {
		return connectors.NewInvalidSettingsError("apiKey is too long")
	}
	for i := 0; i < len(s.APIKey); i++ {
		if c := s.APIKey[i]; c <= ' ' || c == 0x7f {
			return connectors.NewInvalidSettingsError("apiKey must not contain spaces or control characters")
		}
	}
	err := ml.callWithSettings(ctx, s, http.MethodGet, apiBaseURL+"/fields?limit=1", nil, http.StatusOK, nil)
	if err != nil {
		if err, ok := err.(*apiError); ok && (err.StatusCode == http.StatusUnauthorized || err.StatusCode == http.StatusForbidden) {
			return connectors.NewInvalidSettingsError("apiKey is not valid or cannot access the MailerLite API")
		}
		return err
	}
	return ml.env.Settings.Store(ctx, s)
}

type apiError struct {
	StatusCode int
	Message    string              `json:"message"`
	Errors     map[string][]string `json:"errors"`
}

func (err *apiError) Error() string {
	if err == nil {
		return "<nil>"
	}
	if err.Message != "" {
		return fmt.Sprintf("MailerLite error: %s", err.Message)
	}
	return fmt.Sprintf("MailerLite returned HTTP %d", err.StatusCode)
}

// canonicalNumber removes leading zeros from the integer part of a MailerLite
// number string.
func canonicalNumber(s string) string {
	if s == "" {
		return s
	}
	var sign, integer, fraction, exp string
	switch s[0] {
	case '-':
		sign, s = "-", s[1:]
	case '+':
		s = s[1:]
	}
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		s, exp = s[:i], s[i:]
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		integer, fraction = s[:i], s[i:]
	} else {
		integer = s
	}
	integer = strings.TrimLeft(integer, "0")
	if integer == "" {
		integer = "0"
	}
	return sign + integer + fraction + exp
}

// parseNumber converts a MailerLite number field to a Krenalis decimal when
// possible.
func parseNumber(value any) any {
	if value == nil {
		return nil
	}
	s, ok := value.(string)
	if !ok || s == "" {
		return nil
	}
	n, err := decimal.Parse(s, numberPrecision, numberScale)
	if err != nil {
		n, err = decimal.Parse(canonicalNumber(s), numberPrecision, numberScale)
		if err != nil {
			return nil
		}
	}
	return n
}

func subscriberGroupsIDs(groups []subscriberGroup) []string {
	if len(groups) == 0 {
		return nil
	}
	ids := make([]string, 0, len(groups))
	for _, group := range groups {
		if group.ID != "" {
			ids = append(ids, group.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

type subscriber struct {
	ID             string            `json:"id"`
	Email          string            `json:"email"`
	Status         string            `json:"status"`
	Source         string            `json:"source"`
	Sent           int32             `json:"sent"`
	OpensCount     int32             `json:"opens_count"`
	ClicksCount    int32             `json:"clicks_count"`
	OpenRate       float64           `json:"open_rate"`
	ClickRate      float64           `json:"click_rate"`
	IPAddress      any               `json:"ip_address"`
	SubscribedAt   any               `json:"subscribed_at"`
	UnsubscribedAt any               `json:"unsubscribed_at"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
	Fields         map[string]any    `json:"fields"`
	Groups         []subscriberGroup `json:"groups"`
	OptedInAt      any               `json:"opted_in_at"`
	OptinIP        any               `json:"optin_ip"`
}

type subscriberGroup struct {
	ID string `json:"id"`
}

func subscriberStatusType() types.Type {
	return types.String().WithValues("active", "unsubscribed", "unconfirmed", "bounced", "junk")
}
