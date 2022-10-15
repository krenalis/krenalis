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

var Debug = false

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

type Connector struct {
	Firehose     connectors.Firehose
	ClientSecret string
	Context      context.Context
}

func init() {
	connectors.RegisterConnector("HubSpot", (*Connector)(nil))
}

// ApplyConfig applies the configuration config.
func (c *Connector) ApplyConfig(account string, config map[string]any) error {
	return c.ApplyConfig(account, config)
}

// ServeWebhook serves a webhook request.
// It returns the ErrWebhookUnauthorized error is the request was not
// authorized.
// See https://developers.hubspot.com/docs/api/webhooks.
func (c *Connector) ServeWebhook(r *http.Request) error {

	// Check if the webhook is valid.
	if !isValidWebhook(c.ClientSecret, r) {
		return connectors.ErrWebhookUnauthorized
	}

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
		return err
	}
	for _, req := range requests {
		ident := connectors.Identity{Account: strconv.Itoa(req.PortalId)}
		switch req.SubscriptionType {
		case "company.propertyChange":
			ident.Group = strconv.Itoa(req.ObjectId)
			c.Firehose.UpdateGroup(ident, req.OccurredAt, connectors.Properties{
				req.PropertyName: req.PropertyValue,
			}, nil)
		case "contact.propertyChange":
			ident.User = strconv.Itoa(req.ObjectId)
			c.Firehose.UpdateUser(ident, req.OccurredAt, connectors.Properties{
				req.PropertyName: req.PropertyValue,
			}, nil)
		case "company.creation":
			ident.Group = strconv.Itoa(req.ObjectId)
			c.Firehose.CreateGroup(ident, req.OccurredAt, connectors.Properties{
				req.PropertyName: req.PropertyValue,
			})
		case "contact.creation":
			ident.User = strconv.Itoa(req.ObjectId)
			c.Firehose.CreateUser(ident, req.OccurredAt, connectors.Properties{
				req.PropertyName: req.PropertyValue,
			})
		case "company.deletion":
			ident.Group = strconv.Itoa(req.ObjectId)
			c.Firehose.DeleteGroup(ident)
		case "contact.deletion":
			ident.User = strconv.Itoa(req.ObjectId)
			c.Firehose.DeleteUser(ident)
		}
	}

	return nil
}

// Properties returns all user and group properties.
func (c *Connector) Properties(token string) ([]connectors.Property, []connectors.Property, error) {

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
	err := c.call(token, "GET", "/crm/v3/properties/contact", nil, 200, &response)
	if err != nil {
		return nil, nil, err
	}

	properties := make([]connectors.Property, 0)
	for _, r := range response.Results {
		if r.Hidden {
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

// Resource returns the resource from a client token.
func (c *Connector) Resource(token string) (string, error) {
	var res struct {
		PortalId int
	}
	err := c.call(token, "GET", "/account-info/v3/details", nil, 200, &res)
	if err != nil {
		return "", err
	}
	if res.PortalId <= 0 {
		return "", fmt.Errorf("connector HubSpot has returned an invalid resource (portalId): %d", res.PortalId)
	}
	return strconv.Itoa(res.PortalId), nil
}

// SetUsers sets the users.
// It requires the "crm.objects.contacts.write" scope.
func (c *Connector) SetUsers(token string, users []connectors.User) error {

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

	return c.call(token, "POST", "/crm/v3/objects/contacts/batch/update", &body, 200, nil)
}

// Users returns the users starting from the given cursor.
func (c *Connector) Users(token, cursor string) error {

	fromDate, err := parseCursor(cursor)
	if err != nil {
		return err
	}

	it, err := c.newIterator(token, "Contact", fromDate, 100)
	if err != nil {
		return err
	}
	for {
		objects, err := it.Next()
		if err != nil {
			return err
		}
		if objects == nil {
			break
		}
		for _, obj := range objects {
			ident := connectors.Identity{User: obj.ID}
			c.Firehose.UpdateUser(ident, obj.LastModifiedDate, obj.Properties, nil)
		}
		fromDate = objects[len(objects)-1].LastModifiedDate
		c.Firehose.SetCursor(serializeCursor(fromDate))
	}

	return nil
}

// Groups returns the groups starting from the given cursor.
func (c *Connector) Groups(token, cursor string) error {

	fromDate, err := parseCursor(cursor)
	if err != nil {
		return err
	}

	it, err := c.newIterator(token, "Company", fromDate, 100)
	if err != nil {
		return err
	}
	for {
		objects, err := it.Next()
		if err != nil {
			return err
		}
		if objects == nil {
			break
		}
		for _, obj := range objects {
			contacts, err := c.companyContacts(token, obj.ID)
			if err != nil {
				return err
			}
			ident := connectors.Identity{Group: obj.ID}
			c.Firehose.UpdateGroup(ident, obj.LastModifiedDate, obj.Properties, contacts)
		}
		fromDate = objects[len(objects)-1].LastModifiedDate
		c.Firehose.SetCursor(strconv.FormatInt(fromDate, 10))
	}

	return nil
}

// companyContacts returns the contacts of the given company.
func (c *Connector) companyContacts(token, company string) ([]string, error) {
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
		err := c.call(token, "GET", requestURL, nil, 200, &response)
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
	*Connector
	Token      string
	Type       string
	Path       string
	FromDate   int64
	Limit      int
	Body       bytes.Buffer
	Terminated bool
}

// newIterator returns an iterator to iterates on objects of type typ. typ can
// be "Company" or "Contact".
// Requires the "crm.objects.contacts.read" scope for contacts and the
// "crm.objects.companies.read" for companies.
func (c *Connector) newIterator(token, typ string, fromDate int64, limit int) (*iter, error) {

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
		Connector: c,
		Token:     token,
		Type:      typ,
		Path:      path,
		FromDate:  fromDate,
		Limit:     limit,
	}

	return &it, nil
}

type object struct {
	ID               string
	Properties       map[string]any
	LastModifiedDate int64
}

// Next returns the next objects or nil if there are no objects.
func (it *iter) Next() ([]object, error) {

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
	it.Body.WriteString("}")

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

	err := it.call(it.Token, "POST", it.Path, &it.Body, 200, &response)
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
		objects[i] = object{
			ID:               result.ID,
			Properties:       result.Properties,
			LastModifiedDate: date.UnixMilli(),
		}
	}

	it.FromDate = objects[len(objects)-1].LastModifiedDate + 1

	return objects, nil
}

func (c *Connector) call(token, method, path string, body io.Reader, expectedStatus int, response any) error {

	req, err := http.NewRequestWithContext(c.Context, method, "https://api.hubapi.com/"+path[1:], body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
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
