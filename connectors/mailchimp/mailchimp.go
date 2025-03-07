//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package mailchimp implements the Mailchimp connector.
// (https://mailchimp.com/developer/marketing/)
package mailchimp

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/backoff"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/relvacode/iso8601"
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
	Audience      string
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
func (mc *MailChimp) Records(ctx context.Context, _ meergo.Targets, lastChangeTime time.Time, _, properties []string, cursor string, _ types.Type) ([]meergo.Record, string, error) {

	path := "/lists/" + url.PathEscape(mc.settings.Audience) + "/members"

	hasID := false
	hasLastChanged := false

	var fields strings.Builder
	for i, name := range properties {
		if i > 0 {
			fields.WriteByte(',')
		}
		fields.WriteString("members.")
		fields.WriteString(name)
		switch name {
		case "id":
			hasID = true
		case "last_changed":
			hasLastChanged = true
		}
	}
	if !hasID {
		fields.WriteString(",members.id")
	}
	if !hasLastChanged {
		fields.WriteString(",members.last_changed")
	}

	values := url.Values{
		"fields":     {fields.String()},
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
		Members    []map[string]any `json:"members"`
		TotalItems int              `json:"total_items"`
	}

	err := mc.call(ctx, "GET", path, values, nil, 200, &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Members) == 0 {
		return nil, "", io.EOF
	}

	records := make([]meergo.Record, len(response.Members))

	for i, properties := range response.Members {
		id, _ := properties["id"].(string)
		if id == "" {
			return nil, "", errors.New("server returned an invalid 'id' property for a member")
		}
		if !hasID {
			delete(properties, "id")
		}
		lastChanged, _ := properties["last_changed"].(string)
		lastChangeTime, err = iso8601.ParseString(lastChanged)
		if err != nil {
			return nil, "", errors.New("server returned an invalid 'last_changed' property for a member")
		}
		if !hasLastChanged {
			delete(properties, "last_changed")
		}
		records[i] = meergo.Record{
			ID:             id,
			Properties:     properties,
			LastChangeTime: lastChangeTime.UTC(),
		}
	}

	offset, _ := strconv.Atoi(cursor)
	eof := offset+len(response.Members) >= response.TotalItems
	if last := records[len(records)-1]; last.LastChangeTime.Equal(lastChangeTime) {
		offset += len(response.Members)
	} else {
		offset = 0
	}
	if eof {
		return records, strconv.Itoa(offset), io.EOF
	}

	return records, strconv.Itoa(offset), nil
}

// addressType is the types.Type corresponding to the Mailchimp "address" type.
var addressType = types.Object([]types.Property{
	{Name: "addr1", Type: types.Text(), UpdateRequired: true},
	{Name: "addr2", Type: types.Text()},
	{Name: "city", Type: types.Text(), UpdateRequired: true},
	{Name: "state", Type: types.Text(), UpdateRequired: true},
	{Name: "zip", Type: types.Text(), UpdateRequired: true},
	{Name: "country", Type: types.Text()},
})

// Schema returns the schema of the specified target in the specified role.
func (mc *MailChimp) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {

	// Fetch the contact fields, also known as audience fields or merge fields.
	// Mailchimp allows for more than 1,000 fields per audience, but the connector reasonably reads only the first 1,000.
	// See https://mailchimp.com/developer/marketing/docs/merge-fields/.
	var res struct {
		MergeFields []struct {
			Tag          string `json:"tag"`
			Name         string `json:"name"`
			Type         string `json:"type"`
			Required     bool   `json:"required"`
			DisplayOrder int    `json:"display_order"`
			Options      struct {
				Choices []string `json:"choices"`
			} `json:"options"`
		} `json:"merge_fields"`
	}
	params := url.Values{
		"count":  []string{"1000"},
		"fields": []string{"merge_fields.tag,merge_fields.name,merge_fields.type,merge_fields.required,merge_fields.display_order,merge_fields.options.choices"},
	}
	err := mc.call(ctx, "GET", "/lists/"+url.PathEscape(mc.settings.Audience)+"/merge-fields", params, nil, 200, &res)
	if err != nil {
		return types.Type{}, err
	}
	fields := make([]types.Property, 0, len(res.MergeFields))
	for _, f := range res.MergeFields {
		if !types.IsValidPropertyName(f.Tag) {
			continue
		}
		var field types.Property
		switch f.Type {
		case "text":
			field.Type = types.Text().WithCharLen(255)
		case "number":
			field.Type = types.Decimal(14, 2)
			field.Nullable = true
		case "radio", "dropdown":
			var values []string
		Choices:
			// Remove duplicated values.
			for i, value := range f.Options.Choices {
				for _, value2 := range f.Options.Choices[i+1:] {
					if value == value2 {
						continue Choices
					}
				}
				values = append(values, value)
			}
			if values == nil {
				continue
			}
			field.Type = types.Text().WithValues(values...)
			if !slices.Contains(values, "") {
				field.Nullable = true
			}
		case "date":
			field.Type = types.Date()
			field.Nullable = true
		case "birthday":
			field.Type = types.Text().WithCharLen(5)
		case "address":
			field.Type = addressType
		case "zip":
			field.Type = types.Text().WithCharLen(5)
		case "phone":
			field.Type = types.Text()
		case "url":
			field.Type = types.Text()
		default:
			continue
		}
		field.Name = f.Tag
		field.UpdateRequired = f.Required
		field.Description = f.Name
		fields = append(fields, field)
	}

	// Build the schema.
	properties := make([]types.Property, len(staticProperties)+1)
	copy(properties[:2], staticProperties[:2])
	properties[2] = types.Property{
		Name:        "merge_fields",
		Type:        types.Object(fields),
		Description: "Audience fields",
	}
	copy(properties[3:], staticProperties[2:])

	return types.Object(properties), nil
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

	// Get the audiences.
	audiences, err := mc.audiences(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]meergo.Option, len(audiences))
	for i, audience := range audiences {
		options[i] = meergo.Option{
			Text:  audience.Name,
			Value: audience.ID,
		}
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Select{Name: "Audience", Label: "Audience", Options: options},
		},
		Settings: settings,
	}

	return ui, nil
}

const maxBodyRecordsBytes = 100 * 1024 * 1024
const maxBodyRecords = 5000

// Upsert updates or creates records in the app for the specified target.
func (mc *MailChimp) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {

	basePath := "/lists/" + url.PathEscape(mc.settings.Audience) + "/members"

	var body json.Buffer
	body.WriteString(`{"operations":[`)

	for i, record := range records.All() {
		n := body.Len()
		if i > 0 {
			body.WriteByte(',')
		}
		method := "PUT"
		if record.ID == "" {
			method = "PATCH"
		}
		body.WriteString(`{"method":"`)
		body.WriteString(method)
		body.WriteString(`","path":"`)
		body.WriteString(basePath)
		if record.ID != "" {
			body.WriteByte('/')
			body.WriteString(url.PathEscape(record.ID))
		}
		body.WriteString(`","params":{"skip_merge_validation":true},"body":`)
		_ = body.EncodeQuoted(record.Properties)
		body.WriteByte('}')
		if body.Len()+len(`]}`) > maxBodyRecordsBytes {
			body.Truncate(n)
			records.Skip()
			break
		}
		if i+1 == maxBodyRecords {
			break
		}
	}
	body.WriteString(`]}`)

	type batchResponse struct {
		ID                string `json:"id"`
		Status            string `json:"status"`
		ErroredOperations int    `json:"errored_operations"`
		ResponseBodyURL   string `json:"response_body_url"`
	}
	var res batchResponse
	err := mc.call(ctx, "POST", "/batches", nil, &body, 200, &res)
	if err != nil {
		return err
	}
	if res.Status == "finished" && res.ErroredOperations == 0 {
		return nil
	}

	// The batch operation is not finished or some operations are failed.
	if res.Status != "finished" {
		bo := backoff.New(100)
		bo.SetCap(1 * time.Minute)
		batchID := res.ID
		if batchID == "" {
			return errors.New("server does not returned the batch identifier")
		}
		statusPath := "/batches/" + url.PathEscape(batchID)
		for res.Status != "finished" && bo.Next(ctx) {
			err = mc.call(ctx, "GET", statusPath, nil, nil, 200, &res)
			if err != nil {
				return err
			}
		}
		if res.Status != "finished" {
			return errors.New("server does not responded in time to batch operation")
		}
	}

	// The batch operation has completed; check the status of each operation if errors occurred.
	if res.ErroredOperations == 0 {
		return nil
	}

	// At least one operation failed. Read the results for all operations.
	if _, err := url.Parse(res.ResponseBodyURL); err != nil {
		return errors.New("server returned an invalid response body URL")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", res.ResponseBodyURL, nil)
	if err != nil {
		return err
	}
	r, err := mc.conf.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()

	// recordsErr is the error that will be returned containing all the operation errors.
	recordsErr := meergo.RecordsError{}

	// The 'tar.gz' JSON file, returned from Mailchimp, will be deserialized into the 'result' struct.
	var result struct {
		Response   string `json:"response"`
		StatusCode int    `json:"status_code"`
	}
	// The JSON code in 'result.Response' will be deserialized into the 'response' struct.
	var response struct {
		Detail string `json:"detail"`
		Errors []struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		} `json:"errors"`
		Title string `json:"title"`
	}

	// Parse the response.
	gzResults, err := gzip.NewReader(r.Body)
	if err != nil {
		return fmt.Errorf("cannot read the gzip file response from Mailchimp: %s", err)
	}
	tarResults := tar.NewReader(gzResults)
	for {
		header, err := tarResults.Next()
		if err != nil {
			if err == io.EOF {
				return errors.New("gzip file response from Mailchimp does not contain any files")
			}
			return fmt.Errorf("cannot read gzip file response from Mailchimp: %s", err)
		}
		if !header.FileInfo().IsDir() {
			break
		}
	}
	dec := json.NewDecoder(tarResults)
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("cannot parse the JSON response from Mailchimp: %s", err)
	}
	if tok.Kind() != '[' {
		return fmt.Errorf("cannot parse the JSON response from Mailchimp; expecting Array, got %s", tok.Kind())
	}
	for i := 0; dec.PeekKind() == '{'; i++ {
		err = dec.Decode(&result)
		if err != nil {
			return err
		}
		if result.StatusCode != 200 {
			err = json.Unmarshal([]byte(result.Response), &response)
			if err != nil {
				return fmt.Errorf("cannot parse the JSON response from Mailchimp: %s", err)
			}
			if result.StatusCode == 400 {
				slog.Error("connectors/mailchimp: server has returned a 400 error", "details", response.Detail)
				recordsErr[i] = fmt.Errorf("mailchimp has returned a 400 %s error to the connector", response.Title)
				continue
			}
			recordsErr[i] = fmt.Errorf("%s: %s", response.Errors[0].Field, response.Errors[0].Message)
		}
	}

	return recordsErr
}

// saveSettings validates and saves the settings.
func (mc *MailChimp) saveSettings(ctx context.Context, settings json.Value) error {
	var audience struct {
		Audience string
	}
	err := settings.Unmarshal(&audience)
	if err != nil {
		return err
	}
	if audience.Audience == "" || len(audience.Audience) > 100 {
		return meergo.NewInvalidsettingsError("audience length must be in range [1, 100]")
	}
	// Check if the audience exists.
	audiences, err := mc.audiences(ctx)
	if err != nil {
		return err
	}
	var found bool
	for _, l := range audiences {
		if l.ID == audience.Audience {
			found = true
			break
		}
	}
	if !found {
		return meergo.NewInvalidsettingsError("audience does not exist")
	}
	dataCenter, _, err := mc.metadata()
	if err != nil {
		return err
	}
	s := innerSettings{
		Audience:   audience.Audience,
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

type audience struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// audiences returns the audiences.
func (mc *MailChimp) audiences(ctx context.Context) ([]audience, error) {
	params := url.Values{
		"fields":     {"lists.name,lists.id"},
		"count":      {"1000"},
		"sort_field": {"date_created"},
		"sort_dir":   {"ASC"},
	}
	var audiences []audience
	var response struct {
		Audiences []audience `json:"lists"`
	}
	for {
		if len(audiences) > 0 {
			params.Set("offset", strconv.Itoa(len(audiences)))
		}
		err := mc.call(ctx, "GET", "/lists", params, nil, 200, &response)
		if err != nil {
			return nil, err
		}
		audiences = append(audiences, response.Audiences...)
		if len(response.Audiences) < 1000 {
			break
		}
		response.Audiences = response.Audiences[:0]
	}
	return audiences, nil
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
	webhooks, err := mc.webhooks(ctx, mc.settings.Audience)
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
					err = mc.updateWebhook(ctx, mc.settings.Audience, webhook.ID)
					if err != nil {
						return err
					}
				}
				continue
			}
		}
		_ = mc.deleteWebhook(ctx, mc.settings.Audience, webhook.ID)
	}
	if secret == "" {
		secret, err = mc.createWebhook(ctx, mc.settings.Audience)
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

var errAudienceNotExist = errors.New("audience does not exist")

// webhooks returns the webhooks for the provide audience.
// If audience does not exist, it returns the errAudienceNotExist error.
func (mc *MailChimp) webhooks(ctx context.Context, audience string) ([]webhook, error) {
	var response struct {
		Webhooks []webhook `json:"webhooks"`
	}
	err := mc.call(ctx, "GET", "/lists/"+url.PathEscape(audience)+"/webhooks", nil, nil, 200, &response)
	if err != nil {
		if err, ok := err.(*mailchimpError); ok && err.Status == 404 {
			return nil, errAudienceNotExist
		}
		return nil, err
	}
	return response.Webhooks, nil
}

// createWebhook creates a webhook for the provided audience and returns its
// secret.
func (mc *MailChimp) createWebhook(ctx context.Context, audience string) (string, error) {
	path := "/lists/" + url.PathEscape(audience) + "/webhooks"
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
func (mc *MailChimp) deleteWebhook(ctx context.Context, audience, webhook string) error {
	err := mc.call(ctx, "DELETE", "/lists/"+url.PathEscape(audience)+"/webhooks/"+url.PathEscape(webhook), nil, nil, 204, nil)
	if e, ok := err.(*mailchimpError); ok && e.Status == 404 {
		err = nil
	}
	return err
}

// updateWebhook updates the webhook for the provided audience.
func (mc *MailChimp) updateWebhook(ctx context.Context, audience, webhook string) error {
	path := "/lists/" + url.PathEscape(audience) + "/webhooks/" + url.PathEscape(webhook)
	body := `{"events":{"subscribe":true,"unsubscribe":true,"profile":true,"cleaned":true,"upemail":true,"campaign":false},` +
		`"sources":{"user":true,"admin":true,"api":true}`
	return mc.call(ctx, "PATCH", path, nil, strings.NewReader(body), 200, nil)
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

// staticProperties contains the static properties of the user schema.
var staticProperties []types.Property

func init() {
	staticProperties = []types.Property{
		{
			Name:           "email_address",
			Type:           types.Text(),
			CreateRequired: true,
			Description:    "Email address",
		},
		{
			Name:           "status",
			Type:           types.Text().WithValues("subscribed", "unsubscribed", "cleaned", "pending", "transactional", "archived"),
			CreateRequired: true,
			Description:    "Status",
		},
		{
			Name:        "id",
			Type:        types.Text(),
			Description: "ID",
		},
		{
			Name:        "unique_email_id",
			Type:        types.Text(),
			Description: "Unique email ID",
		},
		{
			Name:        "contact_id",
			Type:        types.Text(),
			Description: "Contact ID",
		},
		{
			Name:        "full_name",
			Type:        types.Text(),
			Description: "Full name",
		},
		{
			Name:        "web_id",
			Type:        types.Int(32),
			Description: "Web ID",
		},
		{
			Name:        "email_type",
			Type:        types.Text().WithValues("html", "text"),
			Description: "Email type",
		},
		{
			Name:         "unsubscribe_reason",
			Type:         types.Text(),
			ReadOptional: true,
			Description:  "Unsubscribe reason",
		},
		{
			Name:        "consents_to_one_to_one_messaging",
			Type:        types.Boolean(),
			Description: "Consents to 1:1 messaging",
		},
		{
			Name:        "interests",
			Type:        types.Map(types.Boolean()),
			Nullable:    true,
			Description: "Interests",
		},
		{
			Name: "stats",
			Type: types.Object([]types.Property{
				{Name: "avg_open_rate", Type: types.Decimal(14, 2), Description: "Average open rate"},
				{Name: "avg_click_rate", Type: types.Decimal(14, 2), Description: "Average click-through rate"},
				{Name: "ecommerce_data", Type: types.Object([]types.Property{
					{Name: "total_revenue", Type: types.Decimal(14, 2), Description: "Total revenue"},
					{Name: "number_of_orders", Type: types.Decimal(14, 2), Description: "Number of orders"},
					{Name: "currency_code", Type: types.Text().WithCharLen(3), Description: "Currency code"},
				}), ReadOptional: true, Nullable: true, Description: "Ecommerce"},
			}),
			Description: "Stats",
		},
		{
			Name:        "ip_signup",
			Type:        types.Inet(),
			Nullable:    true,
			Description: "Sign-up IP address",
		},
		{
			Name:        "timestamp_signup",
			Type:        types.DateTime(),
			Nullable:    true,
			Description: "Sign-up date",
		},
		{
			Name:        "ip_opt",
			Type:        types.Inet(),
			Description: "Opt-in IP address",
		},
		{
			Name:        "timestamp_opt",
			Type:        types.DateTime(),
			Description: "Opt-in date",
		},
		{
			Name:        "member_rating",
			Type:        types.Int(8).WithIntRange(1, 5),
			Description: "Star rating",
		},
		{
			Name:        "last_changed",
			Type:        types.DateTime(),
			Description: "Last info update",
		},
		{
			Name:        "language",
			Type:        types.Text().WithCharLen(5),
			Description: "Language",
		},
		{
			Name:        "vip",
			Type:        types.Boolean(),
			Description: "VIP status",
		},
		{
			Name:        "email_client",
			Type:        types.Text(),
			Description: "Email client",
		},
		{
			Name: "location",
			Type: types.Object([]types.Property{
				{Name: "latitude", Type: types.Int(32), Description: "Latitude"},
				{Name: "longitude", Type: types.Int(32), Description: "Longitude"},
				{Name: "gmtoff", Type: types.Int(32), Description: "GMT offset"},
				{Name: "dstoff", Type: types.Int(32), Description: "DST offset"},
				{Name: "country_code", Type: types.Text().WithCharLen(2), Description: "Country code"},
				{Name: "timezone", Type: types.Text(), Description: "Location timezone"},
				{Name: "region", Type: types.Text(), Description: "Location region"},
			}),
			Description: "Location",
		},
		{
			Name: "marketing_permissions",
			Type: types.Object([]types.Property{
				{Name: "marketing_permission_id", Type: types.Text(), Description: "ID"},
				{Name: "text", Type: types.Text(), Description: "Text"},
				{Name: "enabled", Type: types.Boolean(), Description: "Opt-in"},
			}),
			ReadOptional: true,
			Nullable:     true,
			Description:  "Marketing permissions",
		},
		{
			Name: "last_note",
			Type: types.Object([]types.Property{
				{Name: "note_id", Type: types.Int(32), Description: "ID"},
				{Name: "created_at", Type: types.DateTime(), Description: "Creation"},
				{Name: "created_by", Type: types.Text(), Description: "Author"},
				{Name: "note", Type: types.Text(), Description: "Content"},
			}),
			Description: "Last note",
		},
		{
			Name:        "source",
			Type:        types.Text(),
			Description: "Subscriber source",
		},
		{
			Name:        "tags_count",
			Type:        types.Int(32),
			Description: "Tag count",
		},
		{
			Name: "tags",
			Type: types.Array(types.Object([]types.Property{
				{Name: "id", Type: types.Int(32), Description: "ID"},
				{Name: "name", Type: types.Text(), Description: "Name"},
			})),
			Description: "Tags",
		},
	}
}
