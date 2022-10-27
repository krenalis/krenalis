//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package mailchimp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"chichi/connectors"

	"github.com/open2b/nuts/capture"
)

var Debug = true

type mailchimpError struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
}

func (err *mailchimpError) Error() string {
	return fmt.Sprintf("unexpected error from Mailchimp: (%d) %s - %s", err.Status, err.Detail, err.Instance)
}

type connection struct {
	ClientSecret string
	accessToken  string
	ctx          context.Context
	firehose     connectors.Firehose
	settings     settings
}

type settings struct {
	List       string
	DataCenter string
}

func init() {
	connectors.RegisterAppConnector("Mailchimp", New)
}

// New returns a new Mailchimp connection.
func New(ctx context.Context, conf *connectors.AppConfig) (connectors.AppConnection, error) {
	c := connection{
		ctx:         ctx,
		firehose:    conf.Firehose,
		accessToken: conf.AccessToken,
	}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, err
		}
	}
	return &c, nil
}

// Groups returns the groups starting from the given cursor.
func (c *connection) Groups(cursor string, properties [][]string) error {
	return nil
}

// Properties returns all user properties.
func (c *connection) Properties() ([]connectors.Property, []connectors.Property, error) {
	params := url.Values{
		"fields": []string{"merge_fields.options.choices,merge_fields.name,merge_fields.tag,merge_fields.type"},
	}
	res := struct {
		MergeFields []struct {
			Options struct {
				Choices []string
			}
			Name string
			Tag  string
			Type string
		} `json:"merge_fields"`
	}{}
	err := c.call("GET", "/lists/"+c.settings.List+"/merge-fields", params, nil, 200, &res)
	if err != nil {
		return nil, nil, err
	}

	// Merge fields
	mergeFields := make([]connectors.Property, len(res.MergeFields))
	for i, mf := range res.MergeFields {
		mergeFields[i] = connectors.Property{
			Name:  mf.Tag,
			Label: mf.Name,
			Type:  "string",
		}
		switch mf.Type {
		case "address":
			mergeFields[i].Type = "JSON"
		case "radio", "dropdown":
			for _, choice := range mf.Options.Choices {
				mergeFields[i].Options = append(mergeFields[i].Options, connectors.PropertyOption{
					Label: choice,
					Value: choice,
				})
			}
		}
	}

	return []connectors.Property{
		{
			Name:  "ConsentsToOneToOneMessaging",
			Label: "Consents to OneToOne messaging",
			Type:  "bool",
		}, {
			Name:  "ContactID",
			Label: "Contact ID",
			Type:  "string",
		}, {
			Name:  "EmailAddress",
			Label: "Email address",
			Type:  "string",
		}, {
			Name:  "EmailClient",
			Label: "Email client",
			Type:  "string",
		}, {
			Name:  "EmailType",
			Label: "Email type",
			Type:  "string",
		}, {
			Name:  "FullName",
			Label: "Full name",
			Type:  "string",
		}, {
			Name:  "ID",
			Label: "ID",
			Type:  "string",
		}, {
			Name:  "Interests",
			Label: "Interests",
			Type:  "JSON",
		}, {
			Name:  "IPOpt",
			Label: "Opt-in IP address",
			Type:  "string",
		}, {
			Name:  "IPSignup",
			Label: "Sign up IP address",
			Type:  "string",
		}, {
			Name:  "Language",
			Label: "Subscriber's language",
			Type:  "string",
		}, {
			Name:  "LastChanged",
			Label: "Time of the last update",
			Type:  "dateTime",
		}, {
			Name:  "LastNote",
			Label: "Last Note",
			Type:  "JSON",
			Properties: []connectors.Property{
				{
					Name:  "note_id",
					Label: "ID",
					Type:  "int",
				},
				{
					Name:  "created_at",
					Label: "Created at",
					Type:  "dateTime",
				},
				{
					Name:  "created_by",
					Label: "Created by",
					Type:  "string",
				},
				{
					Name:  "note",
					Label: "Note content",
					Type:  "string",
				},
			},
		}, {
			Name:  "ListID",
			Label: "List ID",
			Type:  "string",
		}, {
			Name:  "Location",
			Label: "Location",
			Type:  "JSON",
			Properties: []connectors.Property{
				{
					Name:  "latitude",
					Label: "Latitude",
					Type:  "int",
				}, {
					Name:  "longitude",
					Label: "Longitude",
					Type:  "int",
				}, {
					Name:  "gmtoff",
					Label: "Time difference in hours from GMT",
					Type:  "int",
				}, {
					Name:  "dstoff",
					Label: "Daylight saving time offset",
					Type:  "int",
				}, {
					Name:  "country_code",
					Label: "Country code",
					Type:  "string",
				}, {
					Name:  "timezone",
					Label: "Time zone",
					Type:  "string",
				}, {
					Name:  "region",
					Label: "Region",
					Type:  "string",
				},
			},
		}, {
			Name:  "MarketingPermissions",
			Label: "Marketing permissions",
			Type:  "JSON",
		}, {
			Name:  "MemberRating",
			Label: "Member rating",
			Type:  "int",
		}, {
			Name:       "MergeFields",
			Label:      "Merge fields",
			Properties: mergeFields,
		}, {
			Name:  "Source",
			Label: "Source",
			Type:  "string",
		}, {
			Name:  "Stats",
			Label: "Stats",
			Type:  "JSON",
			Properties: []connectors.Property{
				{
					Name:  "avg_open_rate",
					Label: "Open rate",
					Type:  "int",
				},
				{
					Name:  "avg_click_rate",
					Label: "Click rate",
					Type:  "int",
				},
				{
					Name:  "ecommerce_data",
					Label: "Ecommerce data",
					Type:  "JSON",
				},
			},
		}, {
			Name:  "Status",
			Label: "Status",
			Type:  "string",
		}, {
			Name:  "Tags",
			Label: "Tags",
			Type:  "JSON",
		}, {
			Name:  "TagsCount",
			Label: "Tags count",
			Type:  "int",
		}, {
			Name:  "TimestampOpt",
			Label: "Opt-in time",
			Type:  "dateTime",
		}, {
			Name:  "TimestampSignup",
			Label: "Sign up time",
			Type:  "dateTime",
		}, {
			Name:  "UniqueEmailID",
			Label: "Unique email ID",
			Type:  "string",
		}, {
			Name:  "UnsubscribeReason",
			Label: "Unsubscribe reason",
			Type:  "string",
		}, {
			Name:  "WebID",
			Label: "Web ID",
			Type:  "int",
		}, {
			Name:  "Vip",
			Label: "VIP status",
			Type:  "bool",
		},
	}, nil, nil
}

// ReceiveWebhook receives a webhook request and returns its events.
// It returns the ErrWebhookUnauthorized error is the request was not authorized.
func (c *connection) ReceiveWebhook(r *http.Request) ([]connectors.Event, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	timestamp, err := time.Parse("2006-01-02 15:04:05", r.Form.Get("fired_at"))
	if err != nil {
		return nil, err
	}
	user := r.Form.Get("data[id]")

	// TODO(carlo): subscribe and unsubscribe events are important and should be handled as separate event types.
	var events = make([]connectors.Event, 1)
	switch r.Form.Get("type") {
	case "subscribe":
		// User subscribed.
		events[0] = connectors.UserCreateEvent{
			Timestamp: timestamp,
			User:      user,
		}
	case "unsubscribe", "profile", "upemail":
		// User profile updated.
		events[0] = connectors.UserChangeEvent{
			Timestamp: timestamp,
			User:      user,
		}
	case "cleaned":
		// User profile deleted.
		// TODO(carlo): couldn't trigger this webhook, so the effective content is unknown.
		events[0] = connectors.UserDeleteEvent{
			Timestamp: timestamp,
			User:      user,
		}
	}
	return events, nil
}

// Resource returns the resource from a client token.
func (c *connection) Resource() (string, error) {
	_, resource, err := c.getMetadata()
	if err != nil {
		return "", err
	}
	return resource, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, form []byte) (*connectors.SettingsUI, error) {

	if c.settings.List != "" {
		// TODO: list has been chosen, and cannot be modified
	}
	var err error
	if c.settings.DataCenter == "" {
		c.settings.DataCenter, _, err = c.getMetadata()
		if err != nil {
			// TODO: handle error
		}
		s, err := json.Marshal(c.settings)
		if err != nil {
			// TODO: handle error
		}
		c.firehose.SetSettings(s)
	}

	// TODO(carlo): implement the interface to obtain the settings (datacenter and list to use)
	lists, err := c.getLists()
	if err != nil {
		// TODO: handle error
	}
	_ = lists
	// list := config["List"].(string)
	//
	// lists, err := c.getLists()
	// if err != nil {
	// 	return err
	// }
	//
	// foundList := false
	// for _, l := range lists {
	// 	if l.ID == list {
	// 		foundList = true
	// 		break
	// 	}
	// }
	// if !foundList {
	// 	return &connectors.ConfigError{"List": "list does not exist"}
	// }
	//
	// // Read webhooks for each list and remove the ones that are not for the current one.
	// hasWebhook := false
	// for _, l := range lists {
	// 	hooks, err := c.getWebhooks(l.ID)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	for _, wh := range hooks {
	// 		if l.ID == list {
	// 			// Check if the webhook is the correct one.
	// 			if strings.HasPrefix(wh.URL, c.WebhookBase) {
	// 				if hasWebhook {
	// 					// A webhook was already found, this must be removed.
	// 					err := c.deleteWebhook(l.ID, wh.ID)
	// 					if err != nil {
	// 						return err
	// 					}
	// 					continue
	// 				}
	// 				if wh.Events.Cleaned &&
	// 					wh.Events.Profile &&
	// 					wh.Events.Subscribe &&
	// 					wh.Events.Unsubscribe &&
	// 					wh.Events.Upemail &&
	// 					!wh.Events.Campaign {
	// 					// The correct webhook is already set.
	// 					hasWebhook = true
	// 					continue
	// 				}
	// 				// Update the webhook to the correct settings.
	// 				err := c.setWebhook(l.ID, wh.ID)
	// 				if err != nil {
	// 					return err
	// 				}
	// 				hasWebhook = true
	// 			}
	// 		} else {
	// 			// Delete the webhook if they were set from the connector.
	// 			if strings.HasPrefix(wh.URL, c.WebhookBase) {
	// 				err := c.deleteWebhook(l.ID, wh.ID)
	// 				if err != nil {
	// 					return err
	// 				}
	// 			}
	// 		}
	// 	}
	// }
	//
	// if !hasWebhook {
	// 	err := c.setWebhook(list, "")
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	//
	// connectors.ApplyConfig(config)

	return nil, nil
}

type batchOperation struct {
	Method string            `json:"method"`
	Path   string            `json:"path"`
	Params map[string]string `json:"params"`
	Body   string            `json:"body"`
}
type batchResponse struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	ErroredOperations int    `json:"errored_operations"`
	ResponseBodyURL   string `json:"response_body_url"`
}

// SetUsers sets the given users.
func (c *connection) SetUsers(users []connectors.User) error {

	var r struct {
		Operations []batchOperation `json:"operations"`
	}
	var basePath = "/lists/" + c.settings.List + "/members/"
	for _, u := range users {
		body, err := json.Marshal(u.Properties)
		if err != nil {
			return err
		}
		r.Operations = append(r.Operations, batchOperation{
			Method: "PUT",
			Path:   basePath + u.ID,
			Params: map[string]string{"skip_merge_validation": "true"},
			Body:   string(body),
		})
	}
	rq, err := json.Marshal(r)
	if err != nil {
		return err
	}

	var response batchResponse
	err = c.call("POST", "/batches", nil, bytes.NewReader(rq), 200, &response)
	if err != nil {
		return err
	}

	if response.Status != "finished" {
		// Retrieve the batch at one minute intervals until it's status is finished.
		path := "/batches/" + response.ID
		response := batchResponse{}
		for i := 0; i < 5; i++ {
			time.Sleep(time.Minute)
			err = c.call("GET", path, nil, bytes.NewReader(rq), 200, &response)
			if err != nil {
				return err
			}
			if response.Status != "finished" {
				continue
			}
			if response.ErroredOperations != 0 {
				return errors.New("could not update all users")
			}
		}
		return errors.New("could not complete batch operation")
	}

	return nil
}

// Users returns the users starting from the given cursor.
func (c *connection) Users(cursor string, properties [][]string) error {

	path := "/lists/" + c.settings.List + "/members"
	params := url.Values{
		"sort_field": []string{"last_changed"},
		"sort_dir":   []string{"ASC"},
		"count":      []string{"1000"},
	}
	if len(properties) > 0 {
		params.Set("fields", serializeProperties(properties))
	}
	sinceLastChange, offset, err := parseCursor(cursor)
	if err != nil {
		return err
	}
	for {
		if sinceLastChange != "" {
			params.Set("since_last_changed", sinceLastChange)
		}
		if offset > 0 {
			params.Set("offset", strconv.Itoa(offset))
		} else {
			params.Del("offset")
		}
		var response struct {
			Members    []Member
			TotalItems int `json:"total_items"`
		}
		err := c.call("GET", path, params, nil, 200, &response)
		if err != nil {
			return err
		}
		for _, m := range response.Members {
			c.firehose.SetUser(m.ID, m.LastChanged, m.Properties())
		}
		done := offset+len(response.Members) >= response.TotalItems
		if len(response.Members) > 0 {
			if slc := response.Members[len(response.Members)-1].LastChanged.Format(time.RFC3339); slc != sinceLastChange {
				sinceLastChange = slc
				offset = 0
			} else {
				offset += len(response.Members)
			}
			c.firehose.SetCursor(serializeCursor(sinceLastChange, offset))
		}
		if done {
			break
		}
	}

	return nil
}

// serializeProperties serializes the properties in the Mailchimp "fields"
// parameter format
func serializeProperties(properties [][]string) string {
	var hasID, hasLastChange bool
	for _, p := range properties {
		realName := ""
		switch p[0] {
		case "ConsentsToOneToOneMessaging":
			realName = "consents_to_one_to_one_messaging"
		case "ContactID":
			realName = "contact_id"
		case "EmailAddress":
			realName = "email_address"
		case "EmailClient":
			realName = "email_client"
		case "EmailType":
			realName = "email_type"
		case "FullName":
			realName = "full_name"
		case "ID":
			realName = "id"
			hasID = true
		case "Interests":
			realName = "interests"
		case "IPOpt":
			realName = "ip_opt"
		case "IPSignup":
			realName = "ip_signup"
		case "Language":
			realName = "language"
		case "LastChanged":
			realName = "last_changed"
			hasLastChange = true
		case "LastNote":
			realName = "last_note"
		case "ListID":
			realName = "list_id"
		case "Location":
			realName = "location"
		case "MarketingPermissions":
			realName = "marketing_permissions"
		case "MemberRating":
			realName = "member_rating"
		case "MergeFields":
			realName = "merge_fields"
		case "Source":
			realName = "source"
		case "Stats":
			realName = "stats"
		case "Status":
			realName = "status"
		case "Tags":
			realName = "tags"
		case "TagsCount":
			realName = "tags_count"
		case "TimestampOpt":
			realName = "timestamp_opt"
		case "TimestampSignup":
			realName = "timestamp_signup"
		case "UniqueEmailID":
			realName = "unique_email_id"
		case "UnsubscribeReason":
			realName = "unsubscribe_reason"
		case "WebID":
			realName = "web_id"
		case "Vip":
			realName = "vip"
		}
		p[0] = "members." + realName
	}
	plist := []string{}
	if !hasID {
		plist = append(plist, "members.id")
	}
	if !hasLastChange {
		plist = append(plist, "members.last_changed")
	}
	for _, ps := range properties {
		plist = append(plist, strings.Join(ps, "."))
	}
	return strings.Join(plist, ",")
}

// call calls the Mailchimp API.
func (c *connection) call(method, path string, params url.Values, body io.Reader, expectedStatus int, response any) error {

	if c.settings.DataCenter == "" {
		return errors.New("invalid datacenter")
	}

	var callPath = "https://" + c.settings.DataCenter + ".api.mailchimp.com/3.0/" + path[1:]
	if params != nil {
		callPath += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(c.ctx, method, callPath, body)
	if err != nil {
		return err
	}

	req.SetBasicAuth("anystring", c.accessToken)

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
		hsErr := &mailchimpError{Status: res.StatusCode}
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

type list struct {
	ID, Name string
}

// getLists returns all the lists.
func (c *connection) getLists() ([]list, error) {
	var params = url.Values{
		"fields":     []string{"lists.name", "lists.id"},
		"count":      []string{"1000"},
		"sort_field": []string{"date_created"},
		"sort_dir":   []string{"ASC"},
	}
	var lists []list
	for {
		if len(lists) > 0 {
			params.Set("offset", strconv.Itoa(len(lists)))
		}
		var response struct {
			Lists []list
		}
		err := c.call("GET", "/lists", params, nil, 200, &response)
		if err != nil {
			return nil, err
		}
		lists = append(lists, response.Lists...)
		if len(response.Lists) < 1000 {
			return lists, nil
		}
	}
}

type webhook struct {
	Events struct {
		Campaign    bool `json:"campaign"`
		Cleaned     bool `json:"cleaned"`
		Profile     bool `json:"profile"`
		Subscribe   bool `json:"subscribe"`
		Unsubscribe bool `json:"unsubscribe"`
		Upemail     bool `json:"upemail"`
	} `json:"events"`
	ID      string `json:"id"`
	Sources struct {
		Admin bool `json:"admin"`
		API   bool `json:"api"`
		User  bool `json:"user"`
	} `json:"sources"`
	URL string `json:"url"`
}

// deleteWebhook removes a webhook.
func (c *connection) deleteWebhook(webhook string) error {
	err := c.call("DELETE", "/lists/"+c.settings.List+"/webhooks/"+webhook, nil, nil, 204, nil)
	if err != nil {
		if e, ok := err.(*mailchimpError); ok {
			if e.Status == 404 {
				return nil
			}
		}
		return err
	}
	return nil
}

// getWebhooks returns all the webhooks.
func (c *connection) getWebhooks() ([]webhook, error) {
	var response struct {
		Webhooks []webhook
	}
	err := c.call("GET", "/lists/"+c.settings.List+"/webhooks", nil, nil, 200, &response)
	if err != nil {
		return nil, err
	}
	return response.Webhooks, nil
}

// setWebhook creates or updates a webhook.
func (c *connection) setWebhook(webhook string) error {
	webhookURL := c.firehose.WebhookURL()

	method := "POST"
	path := "/lists/" + c.settings.List + "/webhooks"
	bodyContent := `{"events":{"subscribe":true,"unsubscribe":true,"profile":true,"cleaned":true,"upemail":true,"campaign":false},` +
		`"sources":{"user":true,"admin":true,"api":true}`
	if webhook == "" {
		bodyContent += `,"url":"` + webhookURL + `"}`
	} else {
		method = "PATCH"
		path += "/" + webhook
		bodyContent += `}`
	}
	err := c.call(method, path, nil, bytes.NewBuffer([]byte(bodyContent)), 200, nil)
	if err != nil {
		return err
	}

	return nil
}

// parseCursor parses a cursor and returns the last modified datetime and offset.
func parseCursor(cursor string) (string, int, error) {
	if cursor == "" {
		return "", 0, nil
	}
	parts := strings.SplitN(cursor, "/", 2)
	offset, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid cursor: %q", cursor)
	}
	return parts[0], int(offset), nil
}

// serializeCursor serializes a time and an offset as cursor.
func serializeCursor(time string, offset int) string {
	return time + "/" + strconv.Itoa(offset)
}

// getMember returns a list member.
func (c *connection) getMember(id string) (*Member, error) {
	path := "/lists/" + c.settings.List + "/members/" + id
	var m Member
	err := c.call("GET", path, nil, nil, 200, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

type Member struct {
	ConsentsToOneToOneMessaging bool            `json:"consents_to_one_to_one_messaging"`
	ContactID                   string          `json:"contact_id"`
	EmailAddress                string          `json:"email_address"`
	EmailClient                 string          `json:"email_client"`
	EmailType                   string          `json:"email_type"`
	FullName                    string          `json:"full_name"`
	ID                          string          `json:"id"`
	Interests                   map[string]bool `json:"interests"`
	IPOpt                       string          `json:"ip_opt"`
	IPSignup                    string          `json:"ip_signup"`
	Language                    string          `json:"language"`
	LastChanged                 time.Time       `json:"last_changed"`
	LastNote                    struct {
		NoteID    int       `json:"note_id"`
		CreatedAt time.Time `json:"created_at"`
		CreatedBy string    `json:"created_by"`
		Note      string    `json:"note"`
	} `json:"last_note"`
	ListID   string `json:"list_id"`
	Location struct {
		Latitude    int    `json:"latitude"`
		Longitude   int    `json:"longitude"`
		Gmtoff      int    `json:"gmtoff"`
		Dstoff      int    `json:"dstoff"`
		CountryCode string `json:"country_code"`
		Timezone    string `json:"timezone"`
		Region      string `json:"region"`
	} `json:"location"`
	MarketingPermissions []struct {
		MarketingPermissionID string `json:"marketing_permission_id"`
		Text                  string `json:"text"`
		Enabled               bool   `json:"enabled"`
	} `json:"marketing_permissions"`
	MemberRating int            `json:"member_rating"`
	MergeFields  map[string]any `json:"merge_fields"`
	Source       string         `json:"source"`
	Stats        struct {
		AvgOpenRate   int `json:"avg_open_rate"`
		AvgClickRate  int `json:"avg_click_rate"`
		EcommerceData struct {
			TotalRevenue   int    `json:"total_revenue"`
			NumberOfOrders int    `json:"number_of_orders"`
			CurrencyCode   string `json:"currency_code"`
		} `json:"ecommerce_data"`
	} `json:"stats"`
	Status string `json:"status"`
	Tags   []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"tags"`
	TagsCount         int    `json:"tags_count"`
	TimestampOpt      string `json:"timestamp_opt"`
	TimestampSignup   string `json:"timestamp_signup"`
	UniqueEmailID     string `json:"unique_email_id"`
	UnsubscribeReason string `json:"unsubscribe_reason"`
	Vip               bool   `json:"vip"`
	WebID             int    `json:"web_id"`
}

func (m *Member) Properties() map[string]any {
	return map[string]any{
		"ConsentsToOneToOneMessaging": m.ConsentsToOneToOneMessaging,
		"ContactID":                   m.ContactID,
		"EmailAddress":                m.EmailAddress,
		"EmailClient":                 m.EmailClient,
		"EmailType":                   m.EmailType,
		"FullName":                    m.FullName,
		"ID":                          m.ID,
		"Interests":                   m.Interests,
		"IPOpt":                       m.IPOpt,
		"IPSignup":                    m.IPSignup,
		"Language":                    m.Language,
		"LastChanged":                 m.LastChanged,
		"LastNote":                    m.LastNote,
		"ListID":                      m.ListID,
		"Location":                    m.Location,
		"MarketingPermissions":        m.MarketingPermissions,
		"MemberRating":                m.MemberRating,
		"MergeFields":                 m.MergeFields,
		"Source":                      m.Source,
		"Stats":                       m.Stats,
		"Status":                      m.Status,
		"Tags":                        m.Tags,
		"TagsCount":                   m.TagsCount,
		"TimestampOpt":                m.TimestampOpt,
		"TimestampSignup":             m.TimestampSignup,
		"UniqueEmailID":               m.UniqueEmailID,
		"UnsubscribeReason":           m.UnsubscribeReason,
		"WebID":                       m.WebID,
		"Vip":                         m.Vip,
	}
}

// // setContext sets ctx as the context for c.
// func (c *connection) setContext(ctx context.Context) error {
// 	c.ctx = ctx
// 	c.accessToken, _ = ctx.Value(connectors.AccessTokenContextKey{}).(string)
// 	c.firehose, _ = ctx.Value(connectors.FirehoseContextKey{}).(connectors.Firehose)
// 	if s, ok := ctx.Value(connectors.SettingsContextKey{}).([]byte); ok && len(s) > 0 {
// 		return json.Unmarshal(s, &c.settings)
// 	}
// 	return nil
// }

// getMetadata returns the datacenter and the account id.
func (c *connection) getMetadata() (string, string, error) {
	// Retrieve the datacenter calling the Metadata endpoint.
	// https://mailchimp.com/developer/marketing/guides/access-user-data-oauth-2/#implement-the-oauth-2-workflow-on-your-server
	req, err := http.NewRequest("GET", "https://login.mailchimp.com/oauth2/metadata", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "OAuth "+c.accessToken)
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()
	r := struct {
		DC     string `json:"dc"`
		UserID int    `json:"user_id"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return "", "", err
	}
	return r.DC, strconv.Itoa(r.UserID), nil
}
