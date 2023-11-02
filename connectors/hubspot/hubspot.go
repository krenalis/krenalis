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

	"chichi/connector"
	"chichi/connector/types"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the AppUsersConnection and the AppGroupsConnection
// interfaces.
var _ interface {
	connector.AppUsersConnection
	connector.AppGroupsConnection
} = (*connection)(nil)

func init() {
	connector.RegisterApp(connector.App{
		Name:                   "HubSpot",
		SourceDescription:      "import contacts as users and companies as groups from HubSpot",
		DestinationDescription: "export users as contacts and groups as companies to HubSpot",
		TermForUsers:           "contacts",
		TermForGroups:          "companies",
		Icon:                   icon,
		WebhooksPer:            connector.WebhooksPerConnector,
		OAuth: connector.OAuth{
			AuthURL:  "https://app-eu1.hubspot.com/oauth/authorize",
			TokenURL: "https://api.hubapi.com/oauth/v1/token",
			Scopes:   []string{"crm.objects.contacts.read", "crm.objects.contacts.write", "crm.schemas.contacts.read"},
		},
	}, new)
}

// new returns a new HubSpot connection.
func new(conf *connector.AppConfig) (*connection, error) {
	c := connection{
		setSettings: conf.SetSettings,
		httpClient:  conf.HTTPClient,
	}
	return &c, nil
}

type connection struct {
	setSettings connector.SetSettingsFunc
	httpClient  connector.HTTPClient
	buf         bytes.Buffer
}

// CreateGroup creates a group with the given properties.
func (c *connection) CreateGroup(ctx context.Context, group map[string]any) error {
	// TODO(marco): implement
	return nil
}

// CreateUser creates a user with the given properties.
func (c *connection) CreateUser(ctx context.Context, user map[string]any) error {

	var body bytes.Buffer
	body.WriteString(`{"properties":`)
	err := json.NewEncoder(&body).Encode(user)
	if err != nil {
		return err
	}
	body.WriteString("}")

	return c.call(ctx, "POST", "/crm/v3/objects/contacts", &body, 201, nil)
}

// EventTypes returns the connection's event types.
func (c *connection) EventTypes(ctx context.Context) ([]*connector.EventType, error) {
	return nil, nil
}

// GroupSchema returns the group schema.
func (c *connection) GroupSchema(ctx context.Context) (types.Type, error) {
	// TODO(marco): implement
	return types.Type{}, nil
}

// Groups returns the groups starting from the given cursor.
func (c *connection) Groups(ctx context.Context, properties []string, cursor connector.Cursor) ([]connector.Group, string, error) {
	objects, after, err := c.objects(ctx, "Company", properties, cursor)
	for _, object := range objects {
		contacts, err := c.companyContacts(ctx, object.ID)
		if err != nil {
			return nil, "", err
		}
		object.Associations = contacts
	}
	return objects, after, err
}

// ReceiveWebhook receives a webhook request and returns its payloads.
// It returns the ErrWebhookUnauthorized error is the request was not
// authorized. The context is the request's context.
func (c *connection) ReceiveWebhook(r *http.Request) ([]connector.WebhookPayload, error) {
	// See https://developers.hubspot.com/docs/api/webhooks.

	// Check if the webhook is valid.
	clientSecret, err := c.httpClient.ClientSecret()
	if err != nil {
		return nil, err
	}
	if !isValidWebhook(clientSecret, r) {
		return nil, connector.ErrWebhookUnauthorized
	}

	var events []connector.WebhookPayload

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
		var event connector.WebhookPayload
		timestamp := time.UnixMilli(req.OccurredAt).UTC()
		resource := strconv.Itoa(req.PortalId)
		switch req.SubscriptionType {
		case "company.propertyChange":
			event = connector.GroupPropertyChangeEvent{
				Timestamp: timestamp,
				Resource:  resource,
				Group:     strconv.Itoa(req.ObjectId),
				Name:      req.PropertyName,
				Value:     req.PropertyValue,
			}
		case "contact.propertyChange":
			event = connector.UserPropertyChangeEvent{
				Timestamp: timestamp,
				Resource:  resource,
				User:      strconv.Itoa(req.ObjectId),
				Name:      req.PropertyName,
				Value:     req.PropertyValue,
			}
		case "company.creation":
			event = connector.GroupCreateEvent{
				Timestamp: timestamp,
				Resource:  resource,
				Group:     strconv.Itoa(req.ObjectId),
			}
		case "contact.creation":
			event = connector.UserCreateEvent{
				Timestamp: timestamp,
				Resource:  resource,
				User:      strconv.Itoa(req.ObjectId),
				Properties: map[string]any{
					req.PropertyName: req.PropertyValue,
				},
			}
		case "company.deletion":
			event = connector.GroupDeleteEvent{
				Timestamp: timestamp,
				Resource:  resource,
				Group:     strconv.Itoa(req.ObjectId),
			}
		case "contact.deletion":
			event = connector.UserDeleteEvent{
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
func (c *connection) Resource(ctx context.Context) (string, error) {
	var res struct {
		PortalId int
	}
	err := c.call(ctx, "GET", "/account-info/v3/details", nil, 200, &res)
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
func (c *connection) UpdateGroup(ctx context.Context, id string, group map[string]any) error {
	// TODO(marco): implement
	return nil
}

// UpdateUser updates the user with identifier id setting the given properties.
// It requires the "crm.objects.contacts.write" scope.
func (c *connection) UpdateUser(ctx context.Context, id string, user map[string]any) error {

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

	return c.call(ctx, "POST", "/crm/v3/objects/contacts/batch/update", &body, 200, nil)
}

// UserSchema returns the user schema.
func (c *connection) UserSchema(ctx context.Context) (types.Type, error) {

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
	err := c.call(ctx, "GET", "/crm/v3/properties/contact", nil, 200, &response)
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
		if typ.PhysicalType() == types.PtText {
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
func (c *connection) Users(ctx context.Context, properties []string, cursor connector.Cursor) ([]connector.User, string, error) {
	return c.objects(ctx, "Contact", properties, cursor)
}

// objects returns the contacts, if typ is "Contact", or the companies, if typ
// is "Company", starting from the given cursor.
func (c *connection) objects(ctx context.Context, typ string, properties []string, cursor connector.Cursor) ([]connector.User, string, error) {

	path := "/crm/v3/objects/"
	var propertyName string
	if typ == "Contact" {
		path += "contacts/search"
		propertyName = "lastmodifieddate"
	} else {
		path += "companies/search"
		propertyName = "hs_lastmodifieddate"
	}

	c.buf.Reset()
	c.buf.WriteString(`{"filterGroups":[{"filters":[{"value":"`)
	if cursor.Timestamp.IsZero() {
		c.buf.WriteByte('0')
	} else {
		c.buf.WriteString(strconv.FormatInt(cursor.Timestamp.UnixMilli(), 10))
	}
	c.buf.WriteString(`","propertyName":"` + propertyName + `","operator":"GTE"}` +
		`]}],"sorts":["` + propertyName + `"],"limit":100,"properties":[`)
	for i, p := range properties {
		if i > 0 {
			c.buf.WriteByte(',')
		}
		c.buf.WriteByte('"')
		c.buf.WriteString(p)
		c.buf.WriteByte('"')
	}
	c.buf.WriteString(`]}`)

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

	err := c.call(ctx, "POST", path, &c.buf, 200, &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Results) == 0 {
		return nil, "", io.EOF
	}

	objects := make([]connector.User, len(response.Results))
	for i, result := range response.Results {
		updatedAt, err := time.Parse(time.RFC3339, result.UpdatedAt)
		if err != nil {
			return nil, "", fmt.Errorf("invalid updatedAt returned by HubSpot: %q", updatedAt)
		}
		objects[i] = connector.User{
			ID:         result.ID,
			Properties: result.Properties,
			Timestamp:  updatedAt,
		}
	}

	if response.Paging.Next.After == "" {
		return objects, "", io.EOF
	}

	return objects, "", nil
}

// companyContacts returns the contacts of the given company.
func (c *connection) companyContacts(ctx context.Context, company string) ([]string, error) {
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
		err := c.call(ctx, "GET", requestURL, nil, 200, &response)
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

func (c *connection) call(ctx context.Context, method, path string, body io.Reader, expectedStatus int, response any) error {
	req, err := http.NewRequestWithContext(ctx, method, "https://api.hubapi.com/"+path[1:], body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
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
		return types.Date().WithLayout(time.RFC3339), nil
	case "datetime":
		return types.DateTime().WithLayout(time.RFC3339), nil
	case "enumeration":
		return types.Text(), nil
	case "number":
		return types.Decimal(types.MaxDecimalPrecision-1, 1), nil
	case "string", "phone_number":
		return types.Text(), nil
	}
	return types.Type{}, connector.NewNotSupportedTypeError(c, t)
}
