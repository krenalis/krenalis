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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

// TODO(Gianluca): Groups are partially supported by this connector. When they
// are fully supported by both the connector and Meergo, re-enable the
// descriptions that refer to the groups and add the target "Groups" where
// needed.

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name: "HubSpot",
		AsSource: &meergo.AsAppSource{
			// Description: "Import contacts as users and companies as groups from HubSpot",
			Description: "Import contacts as users from HubSpot",
			Targets:     meergo.Users,
		},
		AsDestination: &meergo.AsAppDestination{
			// Description: "Export users as contacts and groups as companies to HubSpot",
			Description: "Export users as contacts to HubSpot",
			Targets:     meergo.Users,
		},
		TermForUsers: "contacts",
		// TermForGroups:   "companies",
		IdentityIDLabel: "HubSpot ID",
		Icon:            icon,
		WebhooksPer:     meergo.WebhooksPerConnector,
		OAuth: meergo.OAuth{
			AuthURL:           "https://app-eu1.hubspot.com/oauth/authorize",
			TokenURL:          "https://api.hubapi.com/oauth/v1/token",
			SourceScopes:      []string{"crm.objects.contacts.read", "crm.schemas.contacts.read"},
			DestinationScopes: []string{"crm.objects.contacts.read", "crm.objects.contacts.write", "crm.schemas.contacts.read"},
		},
		BackoffPolicy: meergo.BackoffPolicy{
			// https://developers.hubspot.com/docs/api/error-handling
			"429":                         meergo.HeaderStrategy("X-HubSpot-RateLimit-Interval-Milliseconds", parseMilliseconds),
			"477":                         meergo.RetryAfterStrategy(),
			"500 502 503 504 521 523 524": meergo.ExponentialStrategy(time.Second),
		},
	}, New)
}

// New returns a new HubSpot connector instance.
func New(conf *meergo.AppConfig) (*HubSpot, error) {
	c := HubSpot{
		httpClient: conf.HTTPClient,
	}
	return &c, nil
}

type HubSpot struct {
	httpClient meergo.HTTPClient
	buf        bytes.Buffer
}

// OAuthAccount returns the app's account associated with the OAuth
// authorization.
func (hs *HubSpot) OAuthAccount(ctx context.Context) (string, error) {
	var res struct {
		PortalId int `json:"portalId"`
	}
	err := hs.call(ctx, "GET", "/account-info/v3/details", nil, &res)
	if err != nil {
		return "", err
	}
	if res.PortalId <= 0 {
		return "", fmt.Errorf("connector HubSpot has returned an invalid account (portalId): %d", res.PortalId)
	}
	return strconv.Itoa(res.PortalId), nil
}

// Records returns the records of the specified target.
func (hs *HubSpot) Records(ctx context.Context, target meergo.Targets, lastChangeTime time.Time, ids, properties []string, cursor string, _ types.Type) ([]meergo.Record, string, error) {

	path := "/crm/v3/objects/"
	if target == meergo.Users {
		path += "contacts/"
	} else {
		path += "companies/"
	}

	var response struct {
		Results []struct {
			ID         string         `json:"id"`
			Properties map[string]any `json:"properties"`
			UpdatedAt  string         `json:"updatedAt"`
		} `json:"results"`
		Paging struct {
			Next struct {
				After string `json:"after"`
			} `json:"next"`
		} `json:"paging"`
	}

	var err error

	hs.buf.Reset()
	hs.buf.WriteByte('{')

	if ids != nil {
		hs.buf.WriteString(`"inputs":[`)
		for i, id := range ids {
			if i > 0 {
				hs.buf.WriteByte(',')
			}
			hs.buf.WriteString(`{"id":"`)
			hs.buf.WriteString(id)
			hs.buf.WriteString(`"}`)
		}
		hs.buf.WriteString(`],`)
		path += "batch/read"
	} else {
		propertyName := "lastmodifieddate"
		if target == meergo.Groups {
			propertyName = "hs_lastmodifieddate"
		}
		unix := lastChangeTime.UnixMilli()
		if unix < 0 {
			unix = 0
		}
		hs.buf.WriteString(`"filterGroups":[{"filters":[{"value":"`)
		hs.buf.WriteString(strconv.FormatInt(unix, 10))
		hs.buf.WriteString(`","propertyName":"` + propertyName + `","operator":"GTE"}` +
			`]}],"sorts":["` + propertyName + `"],`)
		path += "search"
	}

	hs.buf.WriteString(`"after":"`)
	hs.buf.WriteString(cursor)
	hs.buf.WriteString(`","limit":100,"properties":[`)
	for i, p := range properties {
		if i > 0 {
			hs.buf.WriteByte(',')
		}
		hs.buf.WriteByte('"')
		hs.buf.WriteString(p)
		hs.buf.WriteByte('"')
	}
	hs.buf.WriteString(`]}`)

	err = hs.call(ctx, "POST", path, &hs.buf, &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Results) == 0 {
		return nil, "", io.EOF
	}

	records := make([]meergo.Record, len(response.Results))
	for i, result := range response.Results {
		records[i] = meergo.Record{
			ID: result.ID,
		}
		updatedAt, err := time.Parse(time.RFC3339, result.UpdatedAt)
		if err != nil {
			records[i].Err = fmt.Errorf("HubSpot has returned an invalid value for updatedAt: %q", updatedAt)
			continue
		}
		records[i].Properties = result.Properties
		records[i].LastChangeTime = updatedAt.UTC()
	}

	if target == meergo.Groups {
		for _, object := range records {
			contacts, err := hs.companyContacts(ctx, object.ID)
			if err != nil {
				return nil, "", err
			}
			object.Associations = contacts
		}
	}

	cursor = response.Paging.Next.After
	if cursor == "" {
		err = io.EOF
	}

	return records, cursor, err
}

// ReceiveWebhook receives a webhook request and returns its payloads.
func (hs *HubSpot) ReceiveWebhook(r *http.Request, role meergo.Role) ([]meergo.WebhookPayload, error) {
	// See https://developers.hubspot.com/docs/api/webhooks.

	// Check if the webhook is valid.
	clientSecret, err := hs.httpClient.ClientSecret()
	if err != nil {
		return nil, err
	}
	if !isValidWebhook(clientSecret, r) {
		return nil, meergo.ErrWebhookUnauthorized
	}

	var events []meergo.WebhookPayload

	// Read the requests.
	var requests []struct {
		ObjectId         int    `json:"objectId"`
		OccurredAt       int64  `json:"occurredAt"`
		PortalId         int    `json:"portalId"`
		PropertyName     string `json:"propertyName"`
		PropertyValue    string `json:"propertyValue"`
		SubscriptionType string `json:"subscriptionType"`
	}
	err = json.Decode(r.Body, &requests)
	if err != nil {
		return nil, err
	}
	for _, req := range requests {
		var event meergo.WebhookPayload
		timestamp := time.UnixMilli(req.OccurredAt).UTC()
		account := strconv.Itoa(req.PortalId)
		switch req.SubscriptionType {
		case "company.propertyChange":
			event = meergo.GroupPropertyChangeEvent{
				Timestamp: timestamp,
				Account:   account,
				Group:     strconv.Itoa(req.ObjectId),
				Name:      req.PropertyName,
				Value:     req.PropertyValue,
			}
		case "contact.propertyChange":
			event = meergo.UserPropertyChangeEvent{
				Timestamp: timestamp,
				Account:   account,
				User:      strconv.Itoa(req.ObjectId),
				Name:      req.PropertyName,
				Value:     req.PropertyValue,
			}
		case "company.creation":
			event = meergo.GroupCreateEvent{
				Timestamp: timestamp,
				Account:   account,
				Group:     strconv.Itoa(req.ObjectId),
			}
		case "contact.creation":
			event = meergo.UserCreateEvent{
				Timestamp: timestamp,
				Account:   account,
				User:      strconv.Itoa(req.ObjectId),
				Properties: map[string]any{
					req.PropertyName: req.PropertyValue,
				},
			}
		case "company.deletion":
			event = meergo.GroupDeleteEvent{
				Timestamp: timestamp,
				Account:   account,
				Group:     strconv.Itoa(req.ObjectId),
			}
		case "contact.deletion":
			event = meergo.UserDeleteEvent{
				Timestamp: timestamp,
				Account:   account,
				User:      strconv.Itoa(req.ObjectId),
			}
		}
		events = append(events, event)
	}

	return events, nil
}

// Schema returns the schema of the specified target in the specified role.
func (hs *HubSpot) Schema(ctx context.Context, _ meergo.Targets, role meergo.Role, _ string) (types.Type, error) {

	var response struct {
		Results []struct {
			Hidden  bool   `json:"hidden"`
			Name    string `json:"name"`
			Options []struct {
				Label  string `json:"label"`
				Value  string `json:"value"`
				Hidden bool   `json:"hidden"`
			} `json:"options"`
			Label                string `json:"label"`
			Description          string `json:"description"`
			Type                 string `json:"type"`
			ModificationMetadata struct {
				ReadOnlyValue bool `json:"readOnlyValue"`
			} `json:"modificationMetadata"`
		} `json:"results"`
	}
	err := hs.call(ctx, "GET", "/crm/v3/properties/contact", nil, &response)
	if err != nil {
		return types.Type{}, err
	}

	properties := make([]types.Property, 0, len(response.Results))
	for _, r := range response.Results {
		typ := propertyType(r.Type)
		if !typ.Valid() {
			continue
		}
		if role == meergo.Destination && r.ModificationMetadata.ReadOnlyValue {
			continue
		}
		property := types.Property{
			Name:        r.Name,
			Type:        typ,
			Nullable:    true,
			Description: r.Label + "\n\n" + r.Description,
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

// Upsert updates or creates records in the app for the specified target.
func (hs *HubSpot) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {

	if target == meergo.Groups {
		return errors.New("groups are not supported")
	}

	// Note that records.All() cannot be used because the HubSpot API's "upsert" method does not allow updating contacts using "hs_object_id".
	// See https://community.hubspot.com/t5/APIs-Integrations/Create-or-update-a-batch-of-contacts-by-unique-property-values/m-p/1047925.

	method := "update"
	if first, _ := records.Peek(); first.ID == "" {
		method = "create"
	}

	var body json.Buffer
	body.WriteString(`{"inputs":[`)

	for i, record := range records.Same() {
		if i > 0 {
			body.WriteByte(',')
		}
		body.WriteByte('{')
		if method == "update" {
			_ = body.EncodeKeyValue("id", record.ID)
		}
		_ = body.EncodeKeyValue("properties", record.Properties)
		body.WriteByte('}')
		if i+1 == 100 {
			break
		}
	}

	body.WriteString(`]}`)

	return hs.call(ctx, "POST", "/crm/v3/objects/contacts/batch/"+method, &body, nil)
}

// companyContacts returns the contacts of the given company.
func (hs *HubSpot) companyContacts(ctx context.Context, company string) ([]string, error) {
	contacts := []string{}
	path := "/crm/v3/objects/companies/" + url.PathEscape(company) + "/associations/Contact"
	after := ""
	for {
		var response struct {
			Results []struct {
				ID string `json:"id"`
			} `json:"results"`
			Paging struct {
				Next struct {
					After string `json:"after"`
				} `json:"next"`
			} `json:"paging"`
		}
		requestURL := path
		if after != "" {
			requestURL += "?after=" + url.QueryEscape(after)
		}
		err := hs.call(ctx, "GET", requestURL, nil, &response)
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

func (hs *HubSpot) call(ctx context.Context, method, path string, body io.Reader, response any) error {
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
	switch res.StatusCode {
	case 200, 201, 207:
	default:
		hsErr := &hubspotError{statusCode: res.StatusCode}
		err := json.Decode(res.Body, hsErr)
		if err != nil {
			return err
		}
		return hsErr
	}
	if response != nil {
		return json.Decode(res.Body, response)
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
	Status     string `json:"status"`
	Message    string `json:"message"`
	Errors     []struct {
		Message string `json:"message"`
		In      string `json:"in"`
	} `json:"errors"`
	Category      string `json:"category"`
	CorrelationId string `json:"correlationId"`
}

func (err *hubspotError) Error() string {
	return fmt.Sprintf("unexpected error from HubSpot: (%d) %s", err.statusCode, err.Message)
}

// parseMilliseconds parses the value of the
// "X-HubSpot-RateLimit-Interval-Milliseconds" response header.
func parseMilliseconds(s string) (time.Duration, error) {
	d, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(d) * time.Millisecond, nil
}

// propertyType returns the type of the HubSpot property with type t.
// If the property type is not supported, it returns an invalid type.
// (https://developers.hubspot.com/docs/api/crm/properties#property-type-and-fieldtype-values).
func propertyType(t string) types.Type {
	switch t {
	case "bool":
		return types.Boolean()
	case "date":
		return types.Date()
	case "datetime":
		return types.DateTime()
	case "enumeration":
		return types.Text()
	case "number":
		// HubSpot has no limitations on precision and scale.
		return types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)
	case "object_coordinates", "json":
		// These types are for internal use and are not visible in HubSpot.
		return types.Type{}
	case "string", "phone_number":
		return types.Text()
	}
	return types.Type{}
}
