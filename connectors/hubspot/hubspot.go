//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package hubspot

// This package is the HubSpot connector.
// (https://developers.hubspot.com/docs/api/crm/understanding-the-crm)

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
	"unicode/utf8"

	"chichi/connector"
	"chichi/connector/types"

	"github.com/open2b/nuts/capture"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the AppUsersConnection and the AppGroupsConnection
// interfaces.
var _ interface {
	connector.AppUsersConnection
	connector.AppGroupsConnection
} = (*connection)(nil)

var Debug = false

func init() {
	connector.RegisterApp(connector.App{
		Name:                   "HubSpot",
		SourceDescription:      "import contacts as users and companies as groups from HubSpot",
		DestinationDescription: "export users as contacts and groups as companies to HubSpot",
		TermForUsers:           "contacts",
		TermForGroups:          "companies",
		Icon:                   icon,
		OAuth: connector.OAuth{
			URL:   "https://app-eu1.hubspot.com/oauth/authorize",
			Scope: "crm.objects.contacts.read crm.objects.contacts.write crm.schemas.contacts.read",
		},
		WebhooksPer: connector.WebhooksPerConnector,
	}, open)
}

type connection struct {
	ctx          context.Context
	clientSecret string
	firehose     connector.Firehose
	resource     string
	accessToken  string
}

// open opens a HubSpot connection and returns it.
func open(ctx context.Context, conf *connector.AppConfig) (*connection, error) {
	c := connection{
		ctx:          ctx,
		firehose:     conf.Firehose,
		clientSecret: conf.ClientSecret,
		resource:     conf.Resource,
		accessToken:  conf.AccessToken,
	}
	return &c, nil
}

// EventTypes returns the connection's event types.
func (c *connection) EventTypes() ([]*connector.EventType, error) {
	return nil, nil
}

// GroupSchema returns the group schema.
func (c *connection) GroupSchema() (types.Type, error) {
	// TODO(marco): implement
	return types.Type{}, nil
}

// Groups returns the groups starting from the given cursor.
func (c *connection) Groups(cursor string, properties []connector.PropertyPath) error {

	fromDate, err := parseCursor(cursor)
	if err != nil {
		return err
	}

	it, err := c.newIterator("Company", properties, fromDate, 100)
	if err != nil {
		return err
	}
	for {
		objects, err := it.next()
		if err != nil {
			return err
		}
		if objects == nil {
			break
		}
		for _, obj := range objects {
			c.firehose.SetGroup(obj.ID, obj.Properties, time.UnixMilli(obj.LastModifiedDate).UTC(), nil)
			contacts, err := c.companyContacts(obj.ID)
			if err != nil {
				return err
			}
			c.firehose.SetGroupUsers(obj.ID, contacts)
		}
		fromDate = objects[len(objects)-1].LastModifiedDate
		c.firehose.SetCursor(strconv.FormatInt(fromDate, 10))
	}

	return nil
}

// ReceiveWebhook receives a webhook request and returns its events.
// It returns the ErrWebhookUnauthorized error is the request was not
// authorized.
// See https://developers.hubspot.com/docs/api/webhooks.
func (c *connection) ReceiveWebhook(r *http.Request) ([]connector.WebhookEvent, error) {

	// Check if the webhook is valid.
	if !isValidWebhook(c.clientSecret, r) {
		return nil, connector.ErrWebhookUnauthorized
	}

	var events []connector.WebhookEvent

	// Read the requests.
	var requests []struct {
		ObjectId         int
		OccurredAt       int64
		PortalId         int
		PropertyName     string
		PropertyValue    string
		SubscriptionType string
	}
	err := json.NewDecoder(r.Body).Decode(&requests)
	if err != nil {
		return nil, err
	}
	for _, req := range requests {
		var event connector.WebhookEvent
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
				Properties: connector.Properties{
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
func (c *connection) Resource() (string, error) {
	var res struct {
		PortalId int
	}
	err := c.call("GET", "/account-info/v3/details", nil, 200, &res)
	if err != nil {
		return "", err
	}
	if res.PortalId <= 0 {
		return "", fmt.Errorf("connector HubSpot has returned an invalid resource (portalId): %d", res.PortalId)
	}
	return strconv.Itoa(res.PortalId), nil
}

// SetGroups sets the given groups.
func (c *connection) SetGroups(groups []connector.Group) error {
	// TODO(marco): implement
	return nil
}

// SetUsers sets the users.
// It requires the "crm.objects.contacts.write" scope.
func (c *connection) SetUsers(users []connector.User) error {

	var body bytes.Buffer
	body.WriteString(`{"inputs":[`)

	for i, user := range users {
		if i > 0 {
			body.WriteString(`,`)
		}
		id, _ := json.Marshal(user.ID)
		body.WriteString(`{"id":`)
		body.Write(id)
		body.WriteString(`,"properties":`)
		err := json.NewEncoder(&body).Encode(user.Properties)
		if err != nil {
			return err
		}
		body.WriteString(`}`)
	}

	body.WriteString(`]}`)

	return c.call("POST", "/crm/v3/objects/contacts/batch/update", &body, 200, nil)
}

// UserSchema returns the user schema.
func (c *connection) UserSchema() (types.Type, error) {

	var response struct {
		Results []struct {
			Hidden  bool
			Name    string
			Options []struct {
				Label  string
				Value  string
				Hidden bool
			}
			Label       string
			Description string
			Calculated  bool
			Type        string
		}
	}
	err := c.call("GET", "/crm/v3/properties/contact", nil, 200, &response)
	if err != nil {
		return types.Type{}, err
	}

	properties := []types.Property{}
	for _, r := range response.Results {
		switch r.Name {
		case "createdate", "lastmodifieddate", "hs_object_id":
			continue
		}
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
		if r.Calculated {
			property.Role = types.SourceRole
		}
		if typ.PhysicalType() == types.PtText {
			var n int
			for _, option := range r.Options {
				if !option.Hidden {
					n++
				}
			}
			if n > 0 {
				values := make([]string, 0, n)
				for _, option := range r.Options {
					if option.Hidden {
						continue
					}
					values = append(values, option.Value)
				}

				property.Type = property.Type.WithEnum(values)
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
func (c *connection) Users(cursor string, properties []connector.PropertyPath) error {

	fromDate, err := parseCursor(cursor)
	if err != nil {
		return err
	}

	it, err := c.newIterator("Contact", properties, fromDate, 100)
	if err != nil {
		return err
	}
	for {
		objects, err := it.next()
		if err != nil {
			return err
		}
		if len(objects) == 0 {
			break
		}
		for _, obj := range objects {
			c.firehose.SetUser(obj.ID, obj.Properties, time.UnixMilli(obj.LastModifiedDate).UTC(), nil)
		}
		fromDate = objects[len(objects)-1].LastModifiedDate
		c.firehose.SetCursor(serializeCursor(fromDate))
	}

	return nil
}

// companyContacts returns the contacts of the given company.
func (c *connection) companyContacts(company string) ([]string, error) {
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
		err := c.call("GET", requestURL, nil, 200, &response)
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

type iter struct {
	*connection
	Type       string
	Path       string
	Properties []byte
	FromDate   int64
	Limit      int
	Body       bytes.Buffer
	Terminated bool
}

// newIterator returns an iterator to iterates on objects of type typ. typ can
// be "Company" or "Contact".
// Requires the "crm.objects.contacts.read" scope for contacts and the
// "crm.objects.companies.read" for companies.
func (c *connection) newIterator(typ string, properties []connector.PropertyPath, fromDate int64, limit int) (*iter, error) {

	path := "/crm/v3/"
	switch typ {
	case "Company":
		path += "objects/companies/search"
	case "Contact":
		path += "objects/contacts/search"
	default:
		return nil, errors.New("invalid type")
	}
	if limit < 0 || limit > math.MaxInt32 {
		return nil, errors.New("invalid limit")
	}

	it := iter{
		connection: c,
		Type:       typ,
		Path:       path,
		FromDate:   fromDate,
		Limit:      limit,
	}

	// Marshal the properties.
	props := make([]string, len(properties))
	for i, p := range properties {
		props[i] = p[0]
	}
	var err error
	it.Properties, err = json.Marshal(props)
	if err != nil {
		return nil, err
	}

	return &it, nil
}

type object struct {
	ID               string
	Properties       map[string]any
	LastModifiedDate int64
}

// next returns the next objects or nil if there are no objects.
func (it *iter) next() ([]object, error) {

	if it.Terminated {
		return nil, nil
	}

	propertyName := "hs_lastmodifieddate"
	if it.Type == "Contact" {
		propertyName = propertyName[3:] // "lastmodifieddate"
	}

	it.Body.Reset()
	it.Body.WriteString(`{"filterGroups":[{"filters":[{"value":"`)
	it.Body.WriteString(strconv.FormatInt(it.FromDate, 10))
	it.Body.WriteString(`","propertyName":"` + propertyName + `","operator":"GTE"}` +
		`]}],"sorts":["` + propertyName + `"]`)
	if it.Limit != 0 {
		it.Body.WriteString(`,"limit":`)
		it.Body.WriteString(strconv.Itoa(it.Limit))
	}
	it.Body.WriteString(`,"properties":`)
	it.Body.Write(it.Properties)
	it.Body.WriteString(`}`)

	var response struct {
		Results []struct {
			ID         string
			Properties map[string]any
			UpdatedAt  string
			Archived   bool
		}
		Paging struct {
			Next struct {
				After string
			}
		}
	}

	err := it.call("POST", it.Path, &it.Body, 200, &response)
	if err != nil {
		return nil, err
	}
	it.Terminated = response.Paging.Next.After == ""
	if len(response.Results) == 0 {
		return nil, nil
	}

	objects := make([]object, len(response.Results))
	for i, result := range response.Results {
		date, err := time.Parse(time.RFC3339, result.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid updateAt returned by HubSpot: %q", date)
		}
		delete(result.Properties, "createdate")
		delete(result.Properties, "lastmodifieddate")
		delete(result.Properties, "hs_object_id")
		objects[i] = object{
			ID:               result.ID,
			Properties:       result.Properties,
			LastModifiedDate: date.UnixMilli(),
		}
	}

	it.FromDate = objects[len(objects)-1].LastModifiedDate + 1

	return objects, nil
}

func (c *connection) call(method, path string, body io.Reader, expectedStatus int, response any) error {

	req, err := http.NewRequestWithContext(c.ctx, method, "https://api.hubapi.com/"+path[1:], body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

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
		hsErr := &hubspotError{statusCode: res.StatusCode}
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

// parseCursor parses a cursor and returns the last modified date.
func parseCursor(cursor string) (int64, error) {
	if cursor == "" {
		return 0, nil
	}
	fromDate, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || fromDate < 0 {
		return 0, fmt.Errorf("invalid cursor: %q", cursor)
	}
	return fromDate, nil
}

// serializeCursor serialize a cursor.
func serializeCursor(fromDate int64) string {
	return strconv.FormatInt(fromDate, 10)
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
		return types.Text().WithCharLen(65536), nil
	}
	return types.Type{}, connector.NewNotSupportedTypeError(c, t)
}
