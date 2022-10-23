//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
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

	"chichi/connectors"

	"github.com/open2b/nuts/capture"
)

// Make sure it implements the AppConnection interface.
var _ connectors.AppConnection = &Connection{}

var Debug = false

func init() {
	connectors.RegisterAppConnector("HubSpot", New)
}

type Connection struct {
	ctx          context.Context
	clientSecret string
	firehose     connectors.Firehose
	resource     string
	accessToken  string
	settings     []byte
}

// New returns a new Hubspot connection.
func New(ctx context.Context, conf *connectors.AppConfig) (connectors.AppConnection, error) {
	c := Connection{
		ctx:          ctx,
		firehose:     conf.Firehose,
		clientSecret: conf.ClientSecret,
		resource:     conf.Resource,
		accessToken:  conf.AccessToken,
	}
	return &c, nil
}

// Groups returns the groups starting from the given cursor.
func (c *Connection) Groups(cursor string, properties [][]string) error {

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
			c.firehose.SetGroup(obj.ID, time.UnixMilli(obj.LastModifiedDate).UTC(), obj.Properties)
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

// Properties returns all user and group properties.
func (c *Connection) Properties() ([]connectors.Property, []connectors.Property, error) {

	var response struct {
		Results []struct {
			Hidden  bool
			Name    string
			Options []struct {
				Label  string
				Value  string
				Hidden bool
			}
			Label string
			Type  string
		}
	}
	err := c.call("GET", "/crm/v3/properties/contact", nil, 200, &response)
	if err != nil {
		return nil, nil, err
	}

	properties := make([]connectors.Property, 0)
	for _, r := range response.Results {
		switch r.Name {
		case "createdate", "lastmodifieddate", "hs_object_id":
			continue
		}
		property := connectors.Property{
			Name:  r.Name,
			Label: r.Label,
			Type:  r.Type,
		}
		var n int
		for _, option := range r.Options {
			if !option.Hidden {
				n++
			}
		}
		if n > 0 {
			property.Options = make([]connectors.PropertyOption, 0, n)
			for _, option := range r.Options {
				if option.Hidden {
					continue
				}
				property.Options = append(property.Options, connectors.PropertyOption{
					Label: option.Label,
					Value: option.Value,
				})
			}
		}
		properties = append(properties, property)
	}

	return properties, nil, nil
}

// ReceiveWebhook receives a webhook request and returns its events.
// It returns the ErrWebhookUnauthorized error is the request was not
// authorized.
// See https://developers.hubspot.com/docs/api/webhooks.
func (c *Connection) ReceiveWebhook(r *http.Request) ([]connectors.Event, error) {

	// Check if the webhook is valid.
	if !isValidWebhook(c.clientSecret, r) {
		return nil, connectors.ErrWebhookUnauthorized
	}

	var events []connectors.Event

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
		var event connectors.Event
		timestamp := time.UnixMilli(req.OccurredAt).UTC()
		resource := strconv.Itoa(req.PortalId)
		switch req.SubscriptionType {
		case "company.propertyChange":
			event = connectors.GroupPropertyChangeEvent{
				Timestamp: timestamp,
				Resource:  resource,
				Group:     strconv.Itoa(req.ObjectId),
				Name:      req.PropertyName,
				Value:     req.PropertyValue,
			}
		case "contact.propertyChange":
			event = connectors.UserPropertyChangeEvent{
				Timestamp: timestamp,
				Resource:  resource,
				User:      strconv.Itoa(req.ObjectId),
				Name:      req.PropertyName,
				Value:     req.PropertyValue,
			}
		case "company.creation":
			event = connectors.GroupCreateEvent{
				Timestamp: timestamp,
				Resource:  resource,
				Group:     strconv.Itoa(req.ObjectId),
			}
		case "contact.creation":
			event = connectors.UserCreateEvent{
				Timestamp: timestamp,
				Resource:  resource,
				User:      strconv.Itoa(req.ObjectId),
				Properties: connectors.Properties{
					req.PropertyName: req.PropertyValue,
				},
			}
		case "company.deletion":
			event = connectors.GroupDeleteEvent{
				Timestamp: timestamp,
				Resource:  resource,
				Group:     strconv.Itoa(req.ObjectId),
			}
		case "contact.deletion":
			event = connectors.UserDeleteEvent{
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
func (c *Connection) Resource() (string, error) {
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

// ServeUserInterface serves the connector's user interface.
func (c *Connection) ServeUserInterface(w http.ResponseWriter, r *http.Request) {}

// SetUsers sets the users.
// It requires the "crm.objects.contacts.write" scope.
func (c *Connection) SetUsers(users []connectors.User) error {

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

// Users returns the users starting from the given cursor.
func (c *Connection) Users(cursor string, properties [][]string) error {

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
			c.firehose.SetUser(obj.ID, time.UnixMilli(obj.LastModifiedDate).UTC(), obj.Properties)
		}
		fromDate = objects[len(objects)-1].LastModifiedDate
		c.firehose.SetCursor(serializeCursor(fromDate))
	}

	return nil
}

// companyContacts returns the contacts of the given company.
func (c *Connection) companyContacts(company string) ([]string, error) {
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
	*Connection
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
func (c *Connection) newIterator(typ string, properties [][]string, fromDate int64, limit int) (*iter, error) {

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
		Connection: c,
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

func (c *Connection) call(method, path string, body io.Reader, expectedStatus int, response any) error {

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
