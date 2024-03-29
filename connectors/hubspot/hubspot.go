//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package hubspot implements the HubSpot connector.
// (https://developers.hubspot.com/docs/api/crm/understanding-the-crm)
package hubspot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the AppUsers and the AppGroups
// interfaces.
var _ interface {
	chichi.AppUsers
	chichi.AppGroups
	chichi.AppEvents
} = (*HubSpot)(nil)

func init() {
	chichi.RegisterApp(chichi.AppInfo{
		Name:                   "HubSpot",
		SourceDescription:      "import contacts as users and companies as groups from HubSpot",
		DestinationDescription: "export users as contacts and groups as companies to HubSpot",
		TermForUsers:           "contacts",
		TermForGroups:          "companies",
		ExternalIDLabel:        "HubSpot ID",
		Icon:                   icon,
		WebhooksPer:            chichi.WebhooksPerConnector,
		OAuth: chichi.OAuth{
			AuthURL:  "https://app-eu1.hubspot.com/oauth/authorize",
			TokenURL: "https://api.hubapi.com/oauth/v1/token",
			Scopes:   []string{"crm.objects.contacts.read", "crm.objects.contacts.write", "crm.schemas.contacts.read"},
		},
		SendingMode: chichi.Cloud,
	}, New)
}

// New returns a new HubSpot connector instance.
func New(conf *chichi.AppConfig) (*HubSpot, error) {
	c := HubSpot{
		setSettings: conf.SetSettings,
		httpClient:  conf.HTTPClient,
	}
	return &c, nil
}

type HubSpot struct {
	setSettings chichi.SetSettingsFunc
	httpClient  chichi.HTTPClient
	buf         bytes.Buffer
}

// CreateGroup creates a group with the given properties.
func (hs *HubSpot) CreateGroup(ctx context.Context, group map[string]any) error {
	// TODO(marco): implement
	return nil
}

// CreateUser creates a user with the given properties.
func (hs *HubSpot) CreateUser(ctx context.Context, user map[string]any) error {

	var body bytes.Buffer
	body.WriteString(`{"properties":`)
	err := json.NewEncoder(&body).Encode(user)
	if err != nil {
		return err
	}
	body.WriteString("}")

	return hs.call(ctx, "POST", "/crm/v3/objects/contacts", &body, 201, nil)
}

// EventTypes returns the event types of the connector's instance.
func (hs *HubSpot) EventTypes(ctx context.Context) ([]*chichi.EventType, error) {
	return nil, nil
}

// EventRequest returns an event request associated with the provided event
// type, event, and transformation data. If redacted is true, sensitive
// authentication data will be redacted in the returned request.
// This method is safe for concurrent use by multiple goroutines.
// If the specified event type does not exist, it returns the
// ErrEventTypeNotExist error.
func (hs *HubSpot) EventRequest(ctx context.Context, eventType *chichi.EventType, event *chichi.Event, data map[string]any, redacted bool) (*chichi.EventRequest, error) {
	panic("Not implemented")
}

// GroupSchema returns the group schema.
func (hs *HubSpot) GroupSchema(ctx context.Context) (types.Type, error) {
	// TODO(marco): implement
	return types.Type{}, nil
}

// Groups returns the groups starting from the given cursor.
func (hs *HubSpot) Groups(ctx context.Context, properties []string, cursor chichi.Cursor) ([]chichi.Record, string, error) {
	objects, after, err := hs.objects(ctx, "Company", properties, cursor)
	for _, object := range objects {
		contacts, err := hs.companyContacts(ctx, object.ID)
		if err != nil {
			return nil, "", err
		}
		object.Associations = contacts
	}
	return objects, after, err
}

// ReceiveWebhook receives a webhook request and returns its payloads. It
// returns the ErrWebhookUnauthorized error is the request was not authorized.
// The context is the request's context.
func (hs *HubSpot) ReceiveWebhook(r *http.Request) ([]chichi.WebhookPayload, error) {
	// See https://developers.hubspot.com/docs/api/webhooks.

	// Check if the webhook is valid.
	clientSecret, err := hs.httpClient.ClientSecret()
	if err != nil {
		return nil, err
	}
	if !isValidWebhook(clientSecret, r) {
		return nil, chichi.ErrWebhookUnauthorized
	}

	var events []chichi.WebhookPayload

	// Read the requests.
	var requests []struct {
		ObjectId         int
		OccurredAt       int64
		PortalId         int
		PropertyName     string
		PropertyValue    string
		SubscriptionType string
	}
	err = json.NewDecoder(r.Body).Decode(&requests)
	if err != nil {
		return nil, err
	}
	for _, req := range requests {
		var event chichi.WebhookPayload
		timestamp := time.UnixMilli(req.OccurredAt).UTC()
		resource := strconv.Itoa(req.PortalId)
		switch req.SubscriptionType {
		case "company.propertyChange":
			event = chichi.GroupPropertyChangeEvent{
				Timestamp: timestamp,
				Resource:  resource,
				Group:     strconv.Itoa(req.ObjectId),
				Name:      req.PropertyName,
				Value:     req.PropertyValue,
			}
		case "contact.propertyChange":
			event = chichi.UserPropertyChangeEvent{
				Timestamp: timestamp,
				Resource:  resource,
				User:      strconv.Itoa(req.ObjectId),
				Name:      req.PropertyName,
				Value:     req.PropertyValue,
			}
		case "company.creation":
			event = chichi.GroupCreateEvent{
				Timestamp: timestamp,
				Resource:  resource,
				Group:     strconv.Itoa(req.ObjectId),
			}
		case "contact.creation":
			event = chichi.UserCreateEvent{
				Timestamp: timestamp,
				Resource:  resource,
				User:      strconv.Itoa(req.ObjectId),
				Properties: map[string]any{
					req.PropertyName: req.PropertyValue,
				},
			}
		case "company.deletion":
			event = chichi.GroupDeleteEvent{
				Timestamp: timestamp,
				Resource:  resource,
				Group:     strconv.Itoa(req.ObjectId),
			}
		case "contact.deletion":
			event = chichi.UserDeleteEvent{
				Timestamp: timestamp,
				Resource:  resource,
				User:      strconv.Itoa(req.ObjectId),
			}
		}
		events = append(events, event)
	}

	return events, nil
}

// Resource returns the resource from a client token.
func (hs *HubSpot) Resource(ctx context.Context) (string, error) {
	var res struct {
		PortalId int
	}
	err := hs.call(ctx, "GET", "/account-info/v3/details", nil, 200, &res)
	if err != nil {
		return "", err
	}
	if res.PortalId <= 0 {
		return "", fmt.Errorf("connector HubSpot has returned an invalid resource (portalId): %d", res.PortalId)
	}
	return strconv.Itoa(res.PortalId), nil
}

// UpdateGroup updates the group with identifier id setting the given
// properties.
func (hs *HubSpot) UpdateGroup(ctx context.Context, id string, group map[string]any) error {
	// TODO(marco): implement
	return nil
}

// UpdateUser updates the user with identifier id setting the given properties.
// It requires the "crm.objects.contacts.write" scope.
func (hs *HubSpot) UpdateUser(ctx context.Context, id string, user map[string]any) error {

	var body bytes.Buffer
	body.WriteString(`{"inputs":[`)
	idJSON, _ := json.Marshal(id)
	body.WriteString(`{"id":`)
	body.Write(idJSON)
	body.WriteString(`,"properties":`)
	err := json.NewEncoder(&body).Encode(user)
	if err != nil {
		return err
	}
	body.WriteString(`}]}`)

	return hs.call(ctx, "POST", "/crm/v3/objects/contacts/batch/update", &body, 200, nil)
}

// UserSchema returns the user schema.
func (hs *HubSpot) UserSchema(ctx context.Context) (types.Type, error) {

	var response struct {
		Results []struct {
			Hidden  bool
			Name    string
			Options []struct {
				Label  string
				Value  string
				Hidden bool
			}
			Label                string
			Description          string
			Type                 string
			ModificationMetadata struct {
				ReadOnlyValue bool
			}
		}
	}
	err := hs.call(ctx, "GET", "/crm/v3/properties/contact", nil, 200, &response)
	if err != nil {
		return types.Type{}, err
	}

	properties := make([]types.Property, 0, len(response.Results))
	for _, r := range response.Results {
		typ, err := propertyType(r.Name, r.Type)
		if err != nil {
			return types.Type{}, err
		}
		property := types.Property{
			Name:        r.Name,
			Label:       r.Label,
			Description: r.Description,
			Type:        typ,
			Nullable:    true,
		}
		if r.ModificationMetadata.ReadOnlyValue {
			property.Role = types.SourceRole
		}
		if typ.Kind() == types.TextKind {
			if len(r.Options) == 0 {
				property.Type.WithCharLen(65536)
			} else {
				var n int
				for _, option := range r.Options {
					if !option.Hidden {
						n++
					}
				}
				if n == 0 {
					continue // all options are hidden, skip the property
				}
				values := make([]string, 0, n)
				for _, option := range r.Options {
					if option.Hidden {
						continue
					}
					values = append(values, option.Value)
				}
				property.Type = typ.WithValues(values...)
			}
		}
		properties = append(properties, property)
	}

	schema, err := types.ObjectOf(properties)
	if err != nil {
		return types.Type{}, fmt.Errorf("cannot create schema from properties: %s", err)
	}

	return schema, nil
}

// Users returns the users starting from the given cursor.
func (hs *HubSpot) Users(ctx context.Context, properties []string, cursor chichi.Cursor) ([]chichi.Record, string, error) {
	return hs.objects(ctx, "Contact", properties, cursor)
}

// objects returns the contacts, if typ is "Contact", or the companies, if typ
// is "Company", starting from the given cursor.
func (hs *HubSpot) objects(ctx context.Context, typ string, properties []string, cursor chichi.Cursor) ([]chichi.Record, string, error) {

	path := "/crm/v3/objects/"
	var propertyName string
	if typ == "Contact" {
		path += "contacts/search"
		propertyName = "lastmodifieddate"
	} else {
		path += "companies/search"
		propertyName = "hs_lastmodifieddate"
	}

	hs.buf.Reset()
	hs.buf.WriteString(`{"filterGroups":[{"filters":[{"value":"`)
	if cursor.Timestamp.IsZero() {
		hs.buf.WriteByte('0')
	} else {
		hs.buf.WriteString(strconv.FormatInt(cursor.Timestamp.UnixMilli(), 10))
	}
	hs.buf.WriteString(`","propertyName":"` + propertyName + `","operator":"GTE"}` +
		`]}],"sorts":["` + propertyName + `"],"limit":100,"properties":[`)
	for i, p := range properties {
		if i > 0 {
			hs.buf.WriteByte(',')
		}
		hs.buf.WriteByte('"')
		hs.buf.WriteString(p)
		hs.buf.WriteByte('"')
	}
	hs.buf.WriteString(`]}`)

	var response struct {
		Results []struct {
			ID         string
			Properties map[string]any
			UpdatedAt  string
		}
		Paging struct {
			Next struct {
				After string
			}
		}
	}

	err := hs.call(ctx, "POST", path, &hs.buf, 200, &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Results) == 0 {
		return nil, "", io.EOF
	}

	objects := make([]chichi.Record, len(response.Results))
	for i, result := range response.Results {
		objects[i] = chichi.Record{
			ID: result.ID,
		}
		updatedAt, err := time.Parse(time.RFC3339, result.UpdatedAt)
		if err != nil {
			objects[i].Err = fmt.Errorf("HubSpot has returned an invalid value for updatedAt: %q", updatedAt)
			continue
		}
		objects[i].Properties = result.Properties
		objects[i].Timestamp = updatedAt.UTC()
	}

	if response.Paging.Next.After == "" {
		return objects, "", io.EOF
	}

	return objects, "", nil
}

// companyContacts returns the contacts of the given company.
func (hs *HubSpot) companyContacts(ctx context.Context, company string) ([]string, error) {
	contacts := []string{}
	path := "/crm/v3/objects/companies/" + url.PathEscape(company) + "/associations/Contact"
	after := ""
	for {
		var response struct {
			Results []struct {
				ID string
			}
			Paging struct {
				Next struct {
					After string
				}
			}
		}
		requestURL := path
		if after != "" {
			requestURL += "?after=" + url.QueryEscape(after)
		}
		err := hs.call(ctx, "GET", requestURL, nil, 200, &response)
		if err != nil {
			return nil, err
		}

		for _, result := range response.Results {
			contacts = append(contacts, result.ID)
		}
		after = response.Paging.Next.After
		if after == "" {
			break
		}
	}
	return contacts, nil
}

func (hs *HubSpot) call(ctx context.Context, method, path string, body io.Reader, expectedStatus int, response any) error {
	req, err := http.NewRequestWithContext(ctx, method, "https://api.hubapi.com/"+path[1:], body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := hs.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()
	if res.StatusCode != expectedStatus {
		hsErr := &hubspotError{statusCode: res.StatusCode}
		dec := json.NewDecoder(res.Body)
		_ = dec.Decode(hsErr)
		return hsErr
	}
	if response != nil {
		dec := json.NewDecoder(res.Body)
		return dec.Decode(response)
	}
	return nil
}

// isValidWebhook reports whether the webhook is valid.
// https://developers.hubspot.com/docs/api/webhooks/validating-requests.
func isValidWebhook(clientSecret string, r *http.Request) bool {
	// The HTTP method must be POST.
	if r.Method != "POST" {
		return false
	}
	// The timestamp cannot be older than 5 minutes.
	timestamp, _ := strconv.ParseInt(r.Header.Get("X-HubSpot-Request-Timestamp"), 10, 64)
	if timestamp < time.Now().UTC().Add(-5*time.Minute).UnixMilli() {
		return false
	}
	// Read the signature.
	signature, err := base64.StdEncoding.DecodeString(r.Header.Get("X-HubSpot-Signature-v3"))
	if err != nil {
		return false
	}
	// Read the body.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	_ = r.Body.Close()
	// The body must be UTF-8 encoded.
	if !utf8.Valid(body) {
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	// Compute the HMAC SHA-256 signature.
	mac := hmac.New(sha256.New, []byte(clientSecret))
	_, _ = io.WriteString(mac, "POST")
	_, _ = io.WriteString(mac, "https://")
	_, _ = io.WriteString(mac, r.Host)
	_, _ = io.WriteString(mac, r.RequestURI)
	_, _ = mac.Write(body)
	_, _ = io.WriteString(mac, r.Header.Get("X-HubSpot-Request-Timestamp"))
	// The signature of the request must be the same as the computed signature.
	return hmac.Equal(signature, mac.Sum(nil))
}

type hubspotError struct {
	statusCode int
	Status     string
	Message    string
	Errors     []struct {
		Message string
		In      string
	}
	Category      string
	CorrelationId string
}

func (err *hubspotError) Error() string {
	return fmt.Sprintf("unexpected error from HubSpot: (%d) %s", err.statusCode, err.Message)
}

// propertyType returns the property type of the HubSpot property type t with name c.
// (https://developers.hubspot.com/docs/api/crm/properties#property-type-and-fieldtype-values).
func propertyType(c, t string) (types.Type, error) {
	switch t {
	case "bool":
		return types.Boolean(), nil
	case "date":
		return types.Date(), nil
	case "datetime":
		return types.DateTime(), nil
	case "enumeration":
		return types.Text(), nil
	case "number":
		return types.Decimal(types.MaxDecimalPrecision-1, 1), nil
	case "string", "phone_number":
		return types.Text(), nil
	}
	return types.Type{}, chichi.NewNotSupportedTypeError(c, t)
}
