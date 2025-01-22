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

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name: "Mailchimp",
		AsSource: &meergo.AsAppSource{
			Description: "Import contacts as users from Mailchimp",
			Targets:     meergo.Users,
			HasSettings: true,
		},
		AsDestination: &meergo.AsAppDestination{
			Description: "Export users as contacts to Mailchimp",
			Targets:     meergo.Users,
			HasSettings: true,
		},
		TermForUsers: "contacts",
		Icon:         icon,
		WebhooksPer:  meergo.WebhooksPerConnection,
		OAuth: meergo.OAuth{
			AuthURL:   "https://login.mailchimp.com/oauth2/authorize?response_type=code",
			TokenURL:  "https://login.mailchimp.com/oauth2/token",
			ExpiresIn: math.MaxInt32,
		},
		BackoffPolicy: meergo.BackoffPolicy{
			// https://mailchimp.com/developer/marketing/docs/fundamentals/#api-limits
			"403 429 500": meergo.ExponentialStrategy(50 * time.Millisecond),
		},
	}, New)
}

// New returns a new Mailchimp connector instance.
func New(conf *meergo.AppConfig) (*MailChimp, error) {
	c := MailChimp{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
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
	conf     *meergo.AppConfig
	settings *innerSettings
}

type innerSettings struct {
	List          string
	DataCenter    string
	WebhookSecret string
}

// OAuthAccount returns the app's account associated with the OAuth
// authorization.
func (mc *MailChimp) OAuthAccount(ctx context.Context) (string, error) {
	_, account, err := mc.metadata()
	return account, err
}

// ReceiveWebhook receives a webhook request and returns its payloads.
func (mc *MailChimp) ReceiveWebhook(r *http.Request, role meergo.Role) ([]meergo.WebhookPayload, error) {

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
	var events = make([]meergo.WebhookPayload, 1)
	switch r.Form.Get("type") {
	case "subscribe":
		// User subscribed.
		events[0] = meergo.UserCreateEvent{
			Timestamp: timestamp,
			User:      user,
		}
	case "unsubscribe", "profile", "upemail":
		// User profile updated.
		events[0] = meergo.UserChangeEvent{
			Timestamp: timestamp,
			User:      user,
		}
	case "cleaned":
		// User profile deleted.
		// TODO(carlo): couldn't trigger this webhook, so the effective content is unknown.
		events[0] = meergo.UserDeleteEvent{
			Timestamp: timestamp,
			User:      user,
		}
	}
	return events, nil
}

// Records returns the records of the specified target.
func (mc *MailChimp) Records(ctx context.Context, _ meergo.Targets, _ types.Type, lastChangeTime time.Time, _, properties []string, cursor string) ([]meergo.Record, string, error) {

	path := "/lists/" + mc.settings.List + "/members"
	values := url.Values{
		"fields":     {serializeProperties(properties)},
		"sort_field": {"timestamp_signup"},
		"sort_dir":   {"ASC"},
		"count":      {"1000"},
	}
	if !lastChangeTime.IsZero() {
		values.Set("since_last_changed", lastChangeTime.Format(time.RFC3339))
	}
	if cursor != "" {
		values.Set("offset", cursor)
	}

	var response struct {
		Members    []Member `json:"members"`
		TotalItems int      `json:"total_items"`
	}

	err := mc.call(ctx, "GET", path, values, nil, 200, &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Members) == 0 {
		return nil, "", io.EOF
	}

	users := make([]meergo.Record, len(response.Members))
	for i, member := range response.Members {
		users[i] = meergo.Record{
			ID:             member.ID,
			Properties:     member.Properties(),
			LastChangeTime: member.LastChanged.UTC(),
		}
	}

	offset, _ := strconv.Atoi(cursor)
	eof := offset+len(response.Members) >= response.TotalItems
	if last := users[len(users)-1]; last.LastChangeTime.Equal(lastChangeTime) {
		offset += len(response.Members)
	} else {
		offset = 0
	}
	if eof {
		return users, strconv.Itoa(offset), io.EOF
	}

	return users, strconv.Itoa(offset), nil
}

// Schema returns the schema of the specified target in the specified role.
func (mc *MailChimp) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {
	params := url.Values{
		"fields": []string{"merge_fields.options.choices,merge_fields.name,merge_fields.tag,merge_fields.type"},
	}
	var res struct {
		MergeFields []struct {
			Options struct {
				Choices []string `json:"choices"`
			} `json:"options"`
			Name string `json:"name"`
			Tag  string `json:"tag"`
			Type string `json:"type"`
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
			Name:        mf.Tag,
			Description: mf.Name,
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
			Name:        "ConsentsToOneToOneMessaging",
			Type:        types.Boolean(),
			Description: "Consents to OneToOne messaging",
		}, {
			Name:        "ContactID",
			Type:        types.Text(),
			Description: "Contact ID",
		}, {
			Name:        "EmailAddress",
			Type:        types.Text(),
			Description: "Email address",
		}, {
			Name:        "EmailClient",
			Type:        types.Text(),
			Description: "Email client",
		}, {
			Name:        "EmailType",
			Type:        types.Text(),
			Description: "Email type",
		}, {
			Name:        "FullName",
			Type:        types.Text(),
			Description: "Full name",
		}, {
			Name: "ID",
			Type: types.Text(),
		}, {
			Name: "Interests",
			Type: types.JSON(),
		}, {
			Name:        "IPOpt",
			Type:        types.Text(),
			Description: "Opt-in IP address",
		}, {
			Name:        "IPSignup",
			Type:        types.Text(),
			Description: "Sign up IP address",
		}, {
			Name:        "Language",
			Type:        types.Text(),
			Description: "Subscriber's language",
		}, {
			Name:        "LastChanged",
			Type:        types.DateTime(),
			Description: "Time of the last update",
		}, {
			Name: "LastNote",
			Type: types.Object([]types.Property{
				{
					Name: "note_id",
					Type: types.Int(32),
				}, {
					Name: "created_at",
					Type: types.DateTime(),
				}, {
					Name: "created_by",
					Type: types.Text(),
				}, {
					Name: "note",
					Type: types.Text(),
				},
			}),
			Description: "Last Note",
		}, {
			Name:        "ListID",
			Type:        types.Text(),
			Description: "List ID",
		}, {
			Name: "Location",
			Type: types.Object([]types.Property{
				{
					Name:        "latitude",
					Type:        types.Int(32),
					Description: "Latitude",
				}, {
					Name:        "longitude",
					Type:        types.Int(32),
					Description: "Longitude",
				}, {
					Name:        "gmtoff",
					Type:        types.Int(32),
					Description: "Time difference in hours from GMT",
				}, {
					Name:        "dstoff",
					Type:        types.Int(32),
					Description: "Daylight saving time offset",
				}, {
					Name:        "country_code",
					Type:        types.Text(),
					Description: "Country code",
				}, {
					Name:        "timezone",
					Type:        types.Text(),
					Description: "Time zone",
				}, {
					Name:        "region",
					Type:        types.Text(),
					Description: "Region",
				},
			}),
			Description: "Location",
		}, {
			Name:        "MarketingPermissions",
			Type:        types.JSON(),
			Description: "Marketing permissions",
		}, {
			Name:        "MemberRating",
			Type:        types.Int(32),
			Description: "Member rating",
		},
		{
			Name:        "MergeFields",
			Type:        types.Object(mergeFields),
			Description: "Merge fields",
		},
		{
			Name:        "Source",
			Type:        types.Text(),
			Description: "Source",
		}, {
			Name: "Stats",
			Type: types.Object([]types.Property{
				{
					Name:        "avg_open_rate",
					Type:        types.Int(32),
					Description: "Open rate",
				}, {
					Name:        "avg_click_rate",
					Type:        types.Int(32),
					Description: "Click rate",
				}, {
					Name:        "ecommerce_data",
					Type:        types.JSON(),
					Description: "Ecommerce data",
				},
			}),
			Description: "Stats",
		}, {
			Name:        "Status",
			Type:        types.Text(),
			Description: "Status",
		}, {
			Name:        "Tags",
			Type:        types.JSON(),
			Description: "Tags",
		}, {
			Name:        "TagsCount",
			Type:        types.Int(32),
			Description: "Tags count",
		}, {
			Name:        "TimestampOpt",
			Type:        types.DateTime(),
			Description: "Opt-in time",
		}, {
			Name:        "TimestampSignup",
			Type:        types.DateTime(),
			Description: "Sign up time",
		}, {
			Name:        "UniqueEmailID",
			Type:        types.Text(),
			Description: "Unique email ID",
		}, {
			Name:        "UnsubscribeReason",
			Type:        types.Text(),
			Description: "Unsubscribe reason",
		}, {
			Name:        "WebID",
			Type:        types.Int(32),
			Description: "Web ID",
		}, {
			Name:        "Vip",
			Type:        types.Boolean(),
			Description: "VIP status",
		},
	})
	if err != nil {
		return types.Type{}, fmt.Errorf("cannot create schema from properties: %s", err)
	}

	return schema, nil
}

// ServeUI serves the connector's user interface.
func (mc *MailChimp) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if mc.settings != nil {
			s = *mc.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, mc.saveSettings(ctx, settings)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	// Get the lists.
	lists, err := mc.lists(ctx)
	if err != nil {
		return nil, err
	}
	listOptions := make([]meergo.Option, len(lists))
	for i, list := range lists {
		listOptions[i] = meergo.Option{
			Text:  list.Name,
			Value: list.ID,
		}
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Select{Name: "List", Label: "List", Options: listOptions},
		},
		Settings: settings,
	}

	return ui, nil
}

// Upsert updates or creates records in the app for the specified target.
func (mc *MailChimp) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {

	record := records.First()
	if record.ID == "" {
		panic("TODO: create not implemented")
	}

	var r struct {
		Operations []batchOperation `json:"operations"`
	}
	var basePath = "/lists/" + mc.settings.List + "/members/"
	body, err := json.Marshal(record.Properties)
	if err != nil {
		return err
	}
	r.Operations = append(r.Operations, batchOperation{
		Method: "PUT",
		Path:   basePath + record.ID,
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

// saveSettings validates and saves the settings.
func (mc *MailChimp) saveSettings(ctx context.Context, settings json.Value) error {
	var list struct {
		List string
	}
	err := settings.Unmarshal(&list)
	if err != nil {
		return err
	}
	if list.List == "" || len(list.List) > 100 {
		return meergo.NewInvalidsettingsError("list length must be in range [1, 100]")
	}
	// Check if the list exists.
	lists, err := mc.lists(ctx)
	if err != nil {
		return err
	}
	var found bool
	for _, l := range lists {
		if l.ID == list.List {
			found = true
			break
		}
	}
	if !found {
		return meergo.NewInvalidsettingsError("list does not exist")
	}
	dataCenter, _, err := mc.metadata()
	if err != nil {
		return err
	}
	s := innerSettings{
		List:       list.List,
		DataCenter: dataCenter,
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = mc.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	mc.settings = &s
	return nil
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

type mailchimpError struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
	Errors   []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"errors"`
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
	plist = append(plist, properties...)
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
		err := json.Decode(res.Body, mcErr)
		if err != nil {
			return err
		}
		return mcErr
	}

	if response != nil {
		return json.Decode(res.Body, response)
	}

	return nil
}

type list struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
			Lists []list `json:"lists"`
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
		Webhooks []webhook `json:"webhooks"`
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
		DC     string `json:"dc"`
		UserID int    `json:"user_id"`
	}{}
	err = json.Decode(res.Body, &r)
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
