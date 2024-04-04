//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package mailchimp implements the Mailchimp connector.
// (https://mailchimp.com/developer/)
package mailchimp

import (
	"bytes"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the App, AppRecords, UI, and Webhooks interfaces.
var _ interface {
	chichi.App
	chichi.AppRecords
	chichi.UI
	chichi.Webhooks
} = (*MailChimp)(nil)

func init() {
	chichi.RegisterApp(chichi.AppInfo{
		Name:                   "Mailchimp",
		Targets:                chichi.Users,
		SourceDescription:      "import contacts as users from Mailchimp",
		DestinationDescription: "export users as contacts to Mailchimp",
		TermForUsers:           "contacts",
		SuggestedDisplayedID:   "EmailAddress",
		Icon:                   icon,
		WebhooksPer:            chichi.WebhooksPerSource,
		OAuth: chichi.OAuth{
			AuthURL:   "https://login.mailchimp.com/oauth2/authorize?response_type=code",
			TokenURL:  "https://login.mailchimp.com/oauth2/token",
			ExpiresIn: math.MaxInt32,
		},
	}, New)
}

// New returns a new Mailchimp connector instance.
func New(conf *chichi.AppConfig) (*MailChimp, error) {
	c := MailChimp{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Mailchimp connector")
		}
		// TODO(marco): re-enable webhooks when a public IP is used.
		//err = c.initWebhooks()
		//if err != nil {
		//		return nil, err
		//}
	}
	return &c, nil
}

type MailChimp struct {
	conf     *chichi.AppConfig
	settings *settings
}

// Create creates a record for the specified target with the given properties.
func (mc *MailChimp) Create(_ context.Context, _ chichi.Targets, record map[string]any) error {
	panic("TODO: not implemented")
}

// ReceiveWebhook receives a webhook request and returns its payloads.
func (mc *MailChimp) ReceiveWebhook(r *http.Request) ([]chichi.WebhookPayload, error) {

	if mc.settings.WebhookSecret == "" {
		// Webhooks are not set up.
		if r.Method == "GET" && r.Header.Get("User-Agent") == "MailChimp.com WebHook Validator" {
			// Setup call from Mailchimp.
			return nil, nil
		}
		return nil, errors.New("unexpected webhook")
	}

	if r.URL.Query().Get("secret") != mc.settings.WebhookSecret {
		// The webhook is not authenticated.
		return nil, errors.New("unauthorized webhook")
	}

	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	timestamp, err := time.Parse(time.DateTime, r.Form.Get("fired_at"))
	if err != nil {
		return nil, err
	}
	user := r.Form.Get("data[id]")

	// TODO(carlo): subscribe and unsubscribe events are important and should be handled as separate event types.
	var events = make([]chichi.WebhookPayload, 1)
	switch r.Form.Get("type") {
	case "subscribe":
		// User subscribed.
		events[0] = chichi.UserCreateEvent{
			Timestamp: timestamp,
			User:      user,
		}
	case "unsubscribe", "profile", "upemail":
		// User profile updated.
		events[0] = chichi.UserChangeEvent{
			Timestamp: timestamp,
			User:      user,
		}
	case "cleaned":
		// User profile deleted.
		// TODO(carlo): couldn't trigger this webhook, so the effective content is unknown.
		events[0] = chichi.UserDeleteEvent{
			Timestamp: timestamp,
			User:      user,
		}
	}
	return events, nil
}

// Records returns the records of the specified target, starting from the given
// cursor.
func (mc *MailChimp) Records(ctx context.Context, _ chichi.Targets, properties []string, cursor chichi.Cursor) ([]chichi.Record, string, error) {

	path := "/lists/" + mc.settings.List + "/members"
	values := url.Values{
		"fields":     {serializeProperties(properties)},
		"sort_field": {"last_changed"},
		"sort_dir":   {"ASC"},
		"count":      {"1000"},
	}
	if !cursor.UpdatedAt.IsZero() {
		values.Set("since_last_changed", cursor.UpdatedAt.Format(time.RFC3339))
	}
	if cursor.Next != "" {
		values.Set("offset", cursor.Next)
	}

	var response struct {
		Members    []Member
		TotalItems int `json:"total_items"`
	}

	err := mc.call(ctx, "GET", path, values, nil, 200, &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Members) == 0 {
		return nil, "", io.EOF
	}

	users := make([]chichi.Record, len(response.Members))
	for i, member := range response.Members {
		users[i] = chichi.Record{
			ID:         member.ID,
			Properties: member.Properties(),
			UpdatedAt:  member.LastChanged.UTC(),
		}
	}

	offset, _ := strconv.Atoi(cursor.Next)
	eof := offset+len(response.Members) >= response.TotalItems
	if last := users[len(users)-1]; last.UpdatedAt.Equal(cursor.UpdatedAt) {
		offset += len(response.Members)
	} else {
		offset = 0
	}
	if eof {
		return users, strconv.Itoa(offset), io.EOF
	}

	return users, strconv.Itoa(offset), nil
}

// Resource returns the resource.
func (mc *MailChimp) Resource(ctx context.Context) (string, error) {
	_, resource, err := mc.metadata()
	return resource, err
}

// Schema returns the schema of the specified target.
func (mc *MailChimp) Schema(ctx context.Context, target chichi.Targets, eventType string) (types.Type, error) {
	params := url.Values{
		"fields": []string{"merge_fields.options.choices,merge_fields.name,merge_fields.tag,merge_fields.type"},
	}
	var res struct {
		MergeFields []struct {
			Options struct {
				Choices []string
			}
			Name string
			Tag  string
			Type string
		} `json:"merge_fields"`
	}
	err := mc.call(ctx, "GET", "/lists/"+mc.settings.List+"/merge-fields", params, nil, 200, &res)
	if err != nil {
		return types.Type{}, err
	}

	// Merge fields.
	mergeFields := make([]types.Property, len(res.MergeFields))
	for i, mf := range res.MergeFields {
		field := types.Property{
			Name:  mf.Tag,
			Label: mf.Name,
		}
		switch mf.Type {
		case "address":
			field.Type = types.JSON()
		case "radio", "dropdown":
			field.Type = types.Text().WithValues(mf.Options.Choices...)
		default:
			field.Type = types.Text()
		}
		mergeFields[i] = field
	}

	schema, err := types.ObjectOf([]types.Property{
		{
			Name:  "ConsentsToOneToOneMessaging",
			Label: "Consents to OneToOne messaging",
			Type:  types.Boolean(),
		}, {
			Name:  "ContactID",
			Label: "Contact ID",
			Type:  types.Text(),
		}, {
			Name:  "EmailAddress",
			Label: "Email address",
			Type:  types.Text(),
		}, {
			Name:  "EmailClient",
			Label: "Email client",
			Type:  types.Text(),
		}, {
			Name:  "EmailType",
			Label: "Email type",
			Type:  types.Text(),
		}, {
			Name:  "FullName",
			Label: "Full name",
			Type:  types.Text(),
		}, {
			Name:  "ID",
			Label: "ID",
			Type:  types.Text(),
		}, {
			Name:  "Interests",
			Label: "Interests",
			Type:  types.JSON(),
		}, {
			Name:  "IPOpt",
			Label: "Opt-in IP address",
			Type:  types.Text(),
		}, {
			Name:  "IPSignup",
			Label: "Sign up IP address",
			Type:  types.Text(),
		}, {
			Name:  "Language",
			Label: "Subscriber's language",
			Type:  types.Text(),
		}, {
			Name:  "LastChanged",
			Label: "Time of the last update",
			Type:  types.DateTime(),
		}, {
			Name:  "LastNote",
			Label: "Last Note",
			Type: types.Object([]types.Property{
				{
					Name:  "note_id",
					Label: "ID",
					Type:  types.Int(32),
				}, {
					Name:  "created_at",
					Label: "Created at",
					Type:  types.DateTime(),
				}, {
					Name:  "created_by",
					Label: "Created by",
					Type:  types.Text(),
				}, {
					Name:  "note",
					Label: "Note content",
					Type:  types.Text(),
				},
			}),
		}, {
			Name:  "ListID",
			Label: "List ID",
			Type:  types.Text(),
		}, {
			Name:  "Location",
			Label: "Location",
			Type: types.Object([]types.Property{
				{
					Name:  "latitude",
					Label: "Latitude",
					Type:  types.Int(32),
				}, {
					Name:  "longitude",
					Label: "Longitude",
					Type:  types.Int(32),
				}, {
					Name:  "gmtoff",
					Label: "Time difference in hours from GMT",
					Type:  types.Int(32),
				}, {
					Name:  "dstoff",
					Label: "Daylight saving time offset",
					Type:  types.Int(32),
				}, {
					Name:  "country_code",
					Label: "Country code",
					Type:  types.Text(),
				}, {
					Name:  "timezone",
					Label: "Time zone",
					Type:  types.Text(),
				}, {
					Name:  "region",
					Label: "Region",
					Type:  types.Text(),
				},
			}),
		}, {
			Name:  "MarketingPermissions",
			Label: "Marketing permissions",
			Type:  types.JSON(),
		}, {
			Name:  "MemberRating",
			Label: "Member rating",
			Type:  types.Int(32),
		},
		{
			Name:  "MergeFields",
			Label: "Merge fields",
			Type:  types.Object(mergeFields),
		},
		{
			Name:  "Source",
			Label: "Source",
			Type:  types.Text(),
		}, {
			Name:  "Stats",
			Label: "Stats",
			Type: types.Object([]types.Property{
				{
					Name:  "avg_open_rate",
					Label: "Open rate",
					Type:  types.Int(32),
				}, {
					Name:  "avg_click_rate",
					Label: "Click rate",
					Type:  types.Int(32),
				}, {
					Name:  "ecommerce_data",
					Label: "Ecommerce data",
					Type:  types.JSON(),
				},
			}),
		}, {
			Name:  "Status",
			Label: "Status",
			Type:  types.Text(),
		}, {
			Name:  "Tags",
			Label: "Tags",
			Type:  types.JSON(),
		}, {
			Name:  "TagsCount",
			Label: "Tags count",
			Type:  types.Int(32),
		}, {
			Name:  "TimestampOpt",
			Label: "Opt-in time",
			Type:  types.DateTime(),
		}, {
			Name:  "TimestampSignup",
			Label: "Sign up time",
			Type:  types.DateTime(),
		}, {
			Name:  "UniqueEmailID",
			Label: "Unique email ID",
			Type:  types.Text(),
		}, {
			Name:  "UnsubscribeReason",
			Label: "Unsubscribe reason",
			Type:  types.Text(),
		}, {
			Name:  "WebID",
			Label: "Web ID",
			Type:  types.Int(32),
		}, {
			Name:  "Vip",
			Label: "VIP status",
			Type:  types.Boolean(),
		},
	})
	if err != nil {
		return types.Type{}, fmt.Errorf("cannot create schema from properties: %s", err)
	}

	return schema, nil
}

// ServeUI serves the connector's user interface.
func (mc *MailChimp) ServeUI(ctx context.Context, event string, values []byte) (*chichi.Form, *chichi.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if mc.settings != nil {
			s = *mc.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := mc.ValidateSettings(ctx, values)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, mc.conf.SetSettings(ctx, s)
	default:
		return nil, nil, chichi.ErrEventNotExist
	}

	// Get the lists.
	lists, err := mc.lists(ctx)
	if err != nil {
		return nil, nil, err
	}
	options := make([]chichi.Option, len(lists))
	for i, list := range lists {
		options[i] = chichi.Option{
			Text:  list.Name,
			Value: list.ID,
		}
	}

	form := &chichi.Form{
		Fields: []chichi.Component{
			&chichi.Select{Name: "list", Label: "List", Options: options},
		},
		Actions: []chichi.Action{{Event: "save", Text: "Save", Variant: "primary"}},
	}

	return form, nil, nil
}

// Update updates the record of the specified target with the identifier id,
// setting the given properties.
func (mc *MailChimp) Update(ctx context.Context, _ chichi.Targets, id string, record map[string]any) error {

	var r struct {
		Operations []batchOperation `json:"operations"`
	}
	var basePath = "/lists/" + mc.settings.List + "/members/"
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}
	r.Operations = append(r.Operations, batchOperation{
		Method: "PUT",
		Path:   basePath + id,
		Params: map[string]string{"skip_merge_validation": "true"},
		Body:   string(body),
	})
	rq, err := json.Marshal(r)
	if err != nil {
		return err
	}

	var response batchResponse
	err = mc.call(ctx, "POST", "/batches", nil, bytes.NewReader(rq), 200, &response)
	if err != nil {
		return err
	}

	if response.Status != "finished" {
		// Retrieve the batch at one minute intervals until it's status is finished.
		path := "/batches/" + response.ID
		response := batchResponse{}
		for i := 0; i < 5; i++ {
			time.Sleep(time.Minute)
			err = mc.call(ctx, "GET", path, nil, bytes.NewReader(rq), 200, &response)
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

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (mc *MailChimp) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var s struct {
		List string
	}
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	if s.List == "" || len(s.List) > 100 {
		return nil, chichi.Errorf("list length must be in range [1, 100]")
	}
	// Check if the list exists.
	lists, err := mc.lists(ctx)
	if err != nil {
		return nil, err
	}
	var found bool
	for _, list := range lists {
		if list.ID == s.List {
			found = true
			break
		}
	}
	if !found {
		return nil, chichi.Errorf("list does not exist")
	}
	dataCenter, _, err := mc.metadata()
	if err != nil {
		return nil, err
	}
	settings := settings{
		List:       s.List,
		DataCenter: dataCenter,
	}
	return json.Marshal(&settings)
}

type batchOperation struct {
	Method string
	Path   string
	Params map[string]string
	Body   string
}

type batchResponse struct {
	ID                string
	Status            string
	ErroredOperations int    `json:"errored_operations"`
	ResponseBodyURL   string `json:"response_body_url"`
}

type mailchimpError struct {
	Type     string
	Title    string
	Status   int
	Detail   string
	Instance string
	Errors   []struct {
		Field   string
		Message string
	}
}

func (err *mailchimpError) Error() string {
	s := err.Title
	if len(err.Errors) == 0 {
		s += " " + err.Detail
	} else {
		for _, e := range err.Errors {
			s += "\n\t" + e.Field + ": " + e.Message
		}
	}
	return s
}

type settings struct {
	List          string
	DataCenter    string
	WebhookSecret string
}

// serializeProperties serializes the properties in the Mailchimp "fields"
// parameter format.
func serializeProperties(properties []string) string {
	var hasID, hasLastChange bool
	for i, p := range properties {
		var realName string
		switch p {
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
		properties[i] = "members." + realName
	}
	var plist []string
	if !hasID {
		plist = append(plist, "members.id")
	}
	if !hasLastChange {
		plist = append(plist, "members.last_changed")
	}
	for _, p := range properties {
		plist = append(plist, p)
	}
	return strings.Join(plist, ",")
}

// call calls the Mailchimp API.
func (mc *MailChimp) call(ctx context.Context, method, path string, params url.Values, body io.Reader, expectedStatus int, response any) error {

	var dataCenter string
	if mc.settings == nil {
		var err error
		dataCenter, _, err = mc.metadata()
		if err != nil {
			return err
		}
	} else {
		dataCenter = mc.settings.DataCenter
	}

	var u = "https://" + dataCenter + ".api.mailchimp.com/3.0/" + path[1:]
	if params != nil {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := mc.conf.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if res.StatusCode != expectedStatus {
		mcErr := &mailchimpError{Status: res.StatusCode}
		dec := json.NewDecoder(res.Body)
		_ = dec.Decode(mcErr)
		return mcErr
	}

	if response != nil {
		dec := json.NewDecoder(res.Body)
		return dec.Decode(response)
	}

	return nil
}

type list struct {
	ID, Name string
}

// lists returns the lists.
func (mc *MailChimp) lists(ctx context.Context) ([]list, error) {
	params := url.Values{
		"fields":     {"lists.name,lists.id"},
		"count":      {"1000"},
		"sort_field": {"date_created"},
		"sort_dir":   {"ASC"},
	}
	var lists []list
	for {
		if len(lists) > 0 {
			params.Set("offset", strconv.Itoa(len(lists)))
		}
		var response struct {
			Lists []list
		}
		err := mc.call(ctx, "GET", "/lists", params, nil, 200, &response)
		if err != nil {
			return nil, err
		}
		lists = append(lists, response.Lists...)
		if len(response.Lists) < 1000 {
			break
		}
	}
	return lists, nil
}

type webhook struct {
	Events struct {
		Campaign    bool
		Cleaned     bool
		Profile     bool
		Subscribe   bool
		Unsubscribe bool
		Upemail     bool
	}
	ID      string
	Sources struct {
		Admin bool
		API   bool
		User  bool
	}
	URL string
}

// initWebhooks initializes webhooks.
func (mc *MailChimp) initWebhooks(ctx context.Context) error {
	if mc.conf.SetSettings == nil || mc.settings.WebhookSecret != "" {
		return nil
	}
	baseURL := mc.conf.WebhookURL
	webhooks, err := mc.webhooks(ctx, mc.settings.List)
	if err != nil {
		return err
	}
	var secret string
	for _, webhook := range webhooks {
		u, err := url.Parse(webhook.URL)
		if err != nil {
			return fmt.Errorf("Mailchimp has returned an invalid webhook URL")
		}
		if strings.HasPrefix(u.String(), baseURL) {
			continue
		}
		if secret == "" {
			sec := u.Query().Get("secret")
			if len(sec) == 20 {
				secret = sec
				if !(webhook.Events.Cleaned &&
					webhook.Events.Profile &&
					webhook.Events.Subscribe &&
					webhook.Events.Unsubscribe &&
					webhook.Events.Upemail &&
					!webhook.Events.Campaign) {
					err = mc.updateWebhook(ctx, mc.settings.List, webhook.ID)
					if err != nil {
						return err
					}
				}
				continue
			}
		}
		_ = mc.deleteWebhook(ctx, mc.settings.List, webhook.ID)
	}
	if secret == "" {
		secret, err = mc.createWebhook(ctx, mc.settings.List)
		if err != nil {
			return fmt.Errorf("cannot create webhook: %s", err)
		}
	}
	mc.settings.WebhookSecret = secret
	b, err := json.Marshal(&mc.settings)
	if err != nil {
		return err
	}
	return mc.conf.SetSettings(ctx, b)
}

var errListNotExist = errors.New("list does not exist")

// webhooks returns the webhooks for list.
// If list does not exist, it returns the errListNotExist error.
func (mc *MailChimp) webhooks(ctx context.Context, list string) ([]webhook, error) {
	var response struct {
		Webhooks []webhook
	}
	err := mc.call(ctx, "GET", "/lists/"+url.PathEscape(list)+"/webhooks", nil, nil, 200, &response)
	if err != nil {
		if err, ok := err.(*mailchimpError); ok && err.Status == 404 {
			return nil, errListNotExist
		}
		return nil, err
	}
	return response.Webhooks, nil
}

// createWebhook creates a webhook for list and returns its secret.
func (mc *MailChimp) createWebhook(ctx context.Context, list string) (string, error) {
	path := "/lists/" + url.PathEscape(list) + "/webhooks"
	secret, err := generateRandomString(20)
	if err != nil {
		return "", err
	}
	webhookURL, _ := json.Marshal(mc.conf.WebhookURL + "?secret=" + url.QueryEscape(secret))
	body := `{"events":{"subscribe":true,"unsubscribe":true,"profile":true,"cleaned":true,"upemail":true,"campaign":false},` +
		`"sources":{"user":true,"admin":true,"api":true},"url":` + string(webhookURL) + `}`
	err = mc.call(ctx, "POST", path, nil, strings.NewReader(body), 200, nil)
	if err != nil {
		return "", err
	}
	return secret, nil
}

// deleteWebhook deletes webhook. It does nothing if the webhook does not exist.
func (mc *MailChimp) deleteWebhook(ctx context.Context, list, webhook string) error {
	err := mc.call(ctx, "DELETE", "/lists/"+url.PathEscape(list)+"/webhooks/"+url.PathEscape(webhook), nil, nil, 204, nil)
	if e, ok := err.(*mailchimpError); ok && e.Status == 404 {
		err = nil
	}
	return err
}

// updateWebhook updates the webhook for list.
func (mc *MailChimp) updateWebhook(ctx context.Context, list, webhook string) error {
	path := "/lists/" + url.PathEscape(list) + "/webhooks/" + url.PathEscape(webhook)
	body := `{"events":{"subscribe":true,"unsubscribe":true,"profile":true,"cleaned":true,"upemail":true,"campaign":false},` +
		`"sources":{"user":true,"admin":true,"api":true}`
	return mc.call(ctx, "PATCH", path, nil, strings.NewReader(body), 200, nil)
}

// parseCursor parses a cursor and returns the last modified datetime and offset.
func parseCursor(cursor string) (string, int) {
	if cursor == "" {
		return "", 0
	}
	parts := strings.SplitN(cursor, "/", 2)
	offset, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0
	}
	return parts[0], int(offset)
}

// serializeCursor serializes time and an offset as cursor.
func serializeCursor(time string, offset int) string {
	return time + "/" + strconv.Itoa(offset)
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

// metadata returns the datacenter and the account id.
func (mc *MailChimp) metadata() (string, string, error) {
	// Retrieve the datacenter calling the Metadata endpoint.
	// https://mailchimp.com/developer/marketing/guides/access-user-data-oauth-2/#implement-the-oauth-2-workflow-on-your-server
	req, err := http.NewRequest("GET", "https://login.mailchimp.com/oauth2/metadata", nil)
	if err != nil {
		return "", "", err
	}
	res, err := mc.conf.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()
	if res.StatusCode != 200 {
		return "", "", fmt.Errorf("fetching metadata, MailChimp returned a %d status code", res.StatusCode)
	}
	r := struct {
		DC     string
		UserID int `json:"user_id"`
	}{}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		return "", "", err
	}
	return r.DC, strconv.Itoa(r.UserID), nil
}

// generateRandomString generates a random string of length characters, composed
// of random letters and numbers.
func generateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	max := big.NewInt(int64(len(charset)))
	s := make([]byte, length)
	for i := range s {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		s[i] = charset[n.Int64()]
	}
	return string(s), nil
}
