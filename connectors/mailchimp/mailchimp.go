// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package mailchimp provides a connector for Mailchimp.
// (https://mailchimp.com/developer/marketing/)
//
// Mailchimp is a trademark of The Rocket Science Group LLC.
// This connector is not affiliated with or endorsed by The Rocket Science Group
// LLC.
package mailchimp

import (
	"archive/tar"
	"compress/gzip"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/relvacode/iso8601"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

func init() {
	connectors.RegisterApplication(connectors.ApplicationSpec{
		Code:       "mailchimp",
		Label:      "Mailchimp",
		Categories: connectors.CategorySaaS,
		AsSource: &connectors.AsApplicationSource{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Import contacts as users from Mailchimp",
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsApplicationDestination{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Export users as contacts to Mailchimp",
				Overview: destinationOverview,
			},
		},
		Terms: connectors.ApplicationTerms{
			User:  "Contact",
			Users: "Contacts",
		},
		OAuth: connectors.OAuth{
			AuthURL:           "https://login.mailchimp.com/oauth2/authorize?response_type=code",
			TokenURL:          "https://login.mailchimp.com/oauth2/token",
			ExpiresIn:         math.MaxInt32,
			DisallowLocalhost: true,
		},
		EndpointGroups: []connectors.EndpointGroup{
			// Endpoint group used for the Mailchimp API.
			{
				Patterns: []string{
					"GET  login.mailchimp.com/oauth2/metadata", // metadata endpoint
					"POST login.mailchimp.com/",                // OAuth token endpoint
					"GET  /3.0/lists",                          // audiences
					"GET  /3.0/lists/",                         // RecordSchema, Records, and webhooks
					"GET  /3.0/batches/",                       // Upsert
					"POST /3.0/batches",                        // Upsert
				},
				RequireOAuth: true,
				// https://mailchimp.com/developer/marketing/docs/fundamentals/#throttling
				RateLimit: connectors.RateLimit{RequestsPerSecond: 20, Burst: 20, MaxConcurrentRequests: 10},
				// https://mailchimp.com/developer/marketing/docs/fundamentals/#api-limits
				RetryPolicy: connectors.RetryPolicy{
					"403 429": connectors.ExponentialStrategy(connectors.Slowdown, 50*time.Millisecond),
					"500":     connectors.ExponentialStrategy(connectors.NetFailure, 50*time.Millisecond),
				},
			},
			// Generic endpoint group used by Upsert to retrieve results,
			// where the request domains and paths are not known in advance.
			{
				Patterns:  []string{"GET /"},
				RateLimit: connectors.RateLimit{RequestsPerSecond: 1, Burst: 1},
			},
		},
	}, New)
}

// New returns a new connector instance for Mailchimp.
func New(env *connectors.ApplicationEnv) (*Mailchimp, error) {
	c := Mailchimp{env: env}
	if len(env.Settings) > 0 {
		err := env.Settings.Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for Mailchimp")
		}
		// TODO(marco): re-enable webhooks when a public IP is used.
		//err = c.initWebhooks()
		//if err != nil {
		//		return nil, err
		//}
	}
	return &c, nil
}

type Mailchimp struct {
	env      *connectors.ApplicationEnv
	settings *innerSettings
}

type innerSettings struct {
	Audience      string `json:"audience"`
	DataCenter    string `json:"dataCenter"`
	WebhookSecret string `json:"webhookSecret"`
}

// OAuthAccount returns the API's account associated with the OAuth
// authorization.
func (mc *Mailchimp) OAuthAccount(ctx context.Context) (string, error) {
	_, account, err := mc.metadata(ctx)
	return account, err
}

// RecordSchema returns the schema of the specified target and role.
func (mc *Mailchimp) RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {

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
			field.Type = types.String().WithMaxLength(255)
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
			field.Type = types.String().WithValues(values...)
			if !slices.Contains(values, "") {
				field.Nullable = true
			}
		case "date":
			field.Type = types.Date()
			field.Nullable = true
		case "birthday":
			field.Type = types.String().WithMaxLength(5)
		case "address":
			field.Type = addressType
			// If ADDRESS is an empty string, it will be set to nil.
			if f.Tag == "ADDRESS" {
				field.Nullable = true
			}
		case "zip":
			field.Type = types.String().WithMaxLength(5)
		case "phone":
			field.Type = types.String()
		case "url":
			field.Type = types.String()
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

// Records returns the records of the specified target.
func (mc *Mailchimp) Records(ctx context.Context, target connectors.Targets, updatedAt time.Time, cursor string, schema types.Type) ([]connectors.Record, string, error) {

	path := "/lists/" + url.PathEscape(mc.settings.Audience) + "/members"

	hasID := false
	hasLastChanged := false
	hasMergeFields := false

	var fields strings.Builder
	for i, p := range schema.Properties().All() {
		if i > 0 {
			fields.WriteByte(',')
		}
		fields.WriteString("members.")
		fields.WriteString(p.Name)
		switch p.Name {
		case "id":
			hasID = true
		case "last_changed":
			hasLastChanged = true
		case "merge_fields":
			hasMergeFields = true
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
	if !updatedAt.IsZero() {
		values.Set("since_last_changed", updatedAt.Format(time.RFC3339))
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

	records := make([]connectors.Record, len(response.Members))

	for i, attributes := range response.Members {
		id, _ := attributes["id"].(string)
		if id == "" {
			return nil, "", errors.New("server returned an invalid 'id' property for a member")
		}
		if !hasID {
			delete(attributes, "id")
		}
		lastChanged, _ := attributes["last_changed"].(string)
		updatedAt, err = iso8601.ParseString(lastChanged)
		if err != nil {
			return nil, "", errors.New("server returned an invalid 'last_changed' property for a member")
		}
		if !hasLastChanged {
			delete(attributes, "last_changed")
		}
		if hasMergeFields {
			// merge_fields.ADDRESS is returned as an empty string when the contact has no address.
			if fields, ok := attributes["merge_fields"].(map[string]any); ok && fields["ADDRESS"] == "" {
				fields["ADDRESS"] = nil
			}
		}
		records[i] = connectors.Record{
			ID:         id,
			Attributes: attributes,
			UpdatedAt:  updatedAt.UTC(),
		}
	}

	offset, _ := strconv.Atoi(cursor)
	eof := offset+len(response.Members) >= response.TotalItems
	if last := records[len(records)-1]; last.UpdatedAt.Equal(updatedAt) {
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
	{Name: "addr1", Type: types.String(), UpdateRequired: true, Description: "Street Address"},
	{Name: "addr2", Type: types.String(), Description: "Address Line 2"},
	{Name: "city", Type: types.String(), UpdateRequired: true, Description: "City"},
	{Name: "state", Type: types.String(), UpdateRequired: true, Description: "State/Province/Region"},
	{Name: "zip", Type: types.String(), UpdateRequired: true, Description: "Postal/Zip Code"},
	{Name: "country", Type: types.String(), Description: "Country"},
})

// ServeUI serves the connector's user interface.
func (mc *Mailchimp) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

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
		return nil, connectors.ErrUIEventNotExist
	}

	// Get the audiences.
	audiences, err := mc.audiences(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]connectors.Option, len(audiences))
	for i, audience := range audiences {
		options[i] = connectors.Option{
			Text:  audience.Name,
			Value: audience.ID,
		}
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Select{Name: "audience", Label: "Audience", Options: options},
		},
		Settings: settings,
	}

	return ui, nil
}

const maxBodyRecordsBytes = 100 * 1024 * 1024
const maxBodyRecords = 5000

// Upsert updates or creates records in the API for the specified target.
func (mc *Mailchimp) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records, schema types.Type) error {

	basePath := "/lists/" + url.PathEscape(mc.settings.Audience) + "/members"

	bb := mc.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()

	bb.WriteString(`{"operations":[`)

	n := 0
	for record := range records.All() {
		size := bb.Len()
		if n > 0 {
			bb.WriteByte(',')
		}
		method := "PATCH"
		if record.IsCreate() {
			method = "POST"
		}
		bb.WriteString(`{"method":"`)
		bb.WriteString(method)
		bb.WriteString(`","path":"`)
		bb.WriteString(basePath)
		if record.IsUpdate() {
			bb.WriteByte('/')
			bb.WriteString(url.PathEscape(record.ID))
		}
		bb.WriteString(`","params":{"skip_merge_validation":true},"body":`)
		_ = bb.EncodeQuoted(record.Attributes)
		bb.WriteByte('}')
		if bb.Len()+len(`]}`) > maxBodyRecordsBytes {
			bb.Truncate(size)
			records.Postpone()
			break
		}
		if err := bb.Flush(); err != nil {
			return err
		}
		n++
		if n == maxBodyRecords {
			break
		}
	}
	bb.WriteString(`]}`)

	type batchResponse struct {
		ID                string `json:"id"`
		Status            string `json:"status"`
		ErroredOperations int    `json:"errored_operations"`
		ResponseBodyURL   string `json:"response_body_url"`
	}
	var batchRes batchResponse
	err := mc.call(ctx, "POST", "/batches", nil, bb, 200, &batchRes)
	if err != nil {
		return err
	}
	if batchRes.Status == "finished" && batchRes.ErroredOperations == 0 {
		return nil
	}

	// The batch operation is not finished or some operations are failed.
	if batchRes.Status != "finished" {
		bo := backoff.New(100)
		bo.SetCap(1 * time.Minute)
		batchID := batchRes.ID
		if batchID == "" {
			return errors.New("server does not returned the batch identifier")
		}
		statusPath := "/batches/" + url.PathEscape(batchID)
		for batchRes.Status != "finished" && bo.Next(ctx) {
			err = mc.call(ctx, "GET", statusPath, nil, nil, 200, &batchRes)
			if err != nil {
				return err
			}
		}
		if batchRes.Status != "finished" {
			return errors.New("server does not responded in time to batch operation")
		}
	}

	// The batch operation has completed; check the status of each operation if errors occurred.
	if batchRes.ErroredOperations == 0 {
		return nil
	}

	// At least one operation failed. Read the results for all operations.
	if _, err := url.Parse(batchRes.ResponseBodyURL); err != nil {
		return errors.New("server returned an invalid response body URL")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", batchRes.ResponseBodyURL, nil)
	if err != nil {
		return err
	}
	res, err := mc.env.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("Mailchimp returned a %d status code while retrieving the results file", res.StatusCode)
	}

	// recordsErr is the error that will be returned containing all the operation errors.
	recordsErr := connectors.RecordsError{}

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
	gzResults, err := gzip.NewReader(res.Body)
	if err != nil {
		return fmt.Errorf("could not read Mailchimp's results file: %q", err)
	}
	tarResults := tar.NewReader(gzResults)
	for {
		header, err := tarResults.Next()
		if err != nil {
			if err == io.EOF {
				return errors.New("Mailchimp's response does not contain any response file")
			}
			return fmt.Errorf("could not read Mailchimp's results file: %s", err)
		}
		if !header.FileInfo().IsDir() {
			break
		}
	}
	dec := json.NewDecoder(tarResults)
	tok, err := dec.ReadToken()
	if err != nil {
		return fmt.Errorf("could not parse Mailchimp's results file: %s", err)
	}
	if tok.Kind() != '[' {
		return fmt.Errorf("could not parse Mailchimp's results file: expecting Array, got %s", tok.Kind())
	}
	for i := 0; dec.PeekKind() == '{'; i++ {
		err = dec.Decode(&result)
		if err != nil {
			return err
		}
		if result.StatusCode != 200 {
			err = json.Unmarshal([]byte(result.Response), &response)
			if err != nil {
				return fmt.Errorf("could not parse Mailchimp's results file: %s", err)
			}
			if result.StatusCode == 400 {
				if strings.HasSuffix(response.Detail, " enter a real email address.") {
					recordsErr[i] = fmt.Errorf("email address looks fake or invalid")
					continue
				}
				recordsErr[i] = fmt.Errorf("Mailchimp returned a 400 %q error", response.Title)
				continue
			}
			if len(response.Errors) == 0 {
				recordsErr[i] = errors.New(response.Detail)
			} else {
				recordsErr[i] = fmt.Errorf("%s: %s", response.Errors[0].Field, response.Errors[0].Message)
			}
		}
	}

	return recordsErr
}

// saveSettings validates and saves the settings.
func (mc *Mailchimp) saveSettings(ctx context.Context, settings json.Value) error {
	var audience struct {
		Audience string `json:"audience"`
	}
	err := settings.Unmarshal(&audience)
	if err != nil {
		return err
	}
	if audience.Audience == "" || len(audience.Audience) > 100 {
		return connectors.NewInvalidSettingsError("audience length must be in range [1, 100]")
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
		return connectors.NewInvalidSettingsError("audience does not exist")
	}
	dataCenter, _, err := mc.metadata(ctx)
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
	err = mc.env.SetSettings(ctx, b)
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
	var s strings.Builder
	s.WriteString(err.Title)
	if len(err.Errors) == 0 {
		s.WriteString(" " + err.Detail)
	} else {
		for _, e := range err.Errors {
			s.WriteString("\n\t" + e.Field + ": " + e.Message)
		}
	}
	return s.String()
}

// call calls the Mailchimp API.
func (mc *Mailchimp) call(ctx context.Context, method, path string, params url.Values, bb *connectors.BodyBuffer, expectedStatus int, response any) error {

	var dataCenter string
	if mc.settings == nil {
		var err error
		dataCenter, _, err = mc.metadata(ctx)
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

	req, err := bb.NewRequest(ctx, method, u)
	if err != nil {
		return err
	}

	res, err := mc.env.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != expectedStatus {
		mcErr := &mailchimpError{Status: res.StatusCode}
		err := json.Decode(res.Body, mcErr)
		if err != nil {
			return errors.New("Mailchimp returned a 400 Bad Request")
		}
		return fmt.Errorf("Mailchimp returned a 400 Bad Request with error %q", mcErr)
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
func (mc *Mailchimp) audiences(ctx context.Context) ([]audience, error) {
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

var errAudienceNotExist = errors.New("audience does not exist")

// webhooks returns the webhooks for the provide audience.
// If audience does not exist, it returns the errAudienceNotExist error.
func (mc *Mailchimp) webhooks(ctx context.Context, audience string) ([]webhook, error) {
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

// metadata returns the datacenter and the account id.
func (mc *Mailchimp) metadata(ctx context.Context) (string, string, error) {
	// Retrieve the datacenter calling the Metadata endpoint.
	// https://mailchimp.com/developer/marketing/guides/access-user-data-oauth-2/#implement-the-oauth-2-workflow-on-your-server
	req, err := http.NewRequestWithContext(ctx, "GET", "https://login.mailchimp.com/oauth2/metadata", nil)
	if err != nil {
		return "", "", err
	}
	res, err := mc.env.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", "", fmt.Errorf("fetching metadata, Mailchimp returned a %d status code", res.StatusCode)
	}
	r := struct {
		DC     string `json:"dc"`
		UserID int    `json:"user_id"`
	}{}
	err = json.Decode(res.Body, &r)
	if err != nil {
		return "", "", err
	}
	if r.DC == "" {
		return "", "", errors.New("fetching metadata, Mailchimp returned an empty data center")
	}
	if r.UserID <= 0 {
		return "", "", fmt.Errorf("fetching metadata, Mailchimp returned an invalid user ID: %d", r.UserID)
	}
	return r.DC, strconv.Itoa(r.UserID), nil
}

// staticProperties contains the static properties of the user schema.
var staticProperties []types.Property

func init() {
	staticProperties = []types.Property{
		{
			Name:           "email_address",
			Type:           types.String(),
			CreateRequired: true,
			Description:    "Email address",
		},
		{
			Name:           "status",
			Type:           types.String().WithValues("subscribed", "unsubscribed", "cleaned", "pending", "transactional", "archived"),
			CreateRequired: true,
			Description:    "Status",
		},
		{
			Name:        "id",
			Type:        types.String(),
			Description: "ID",
		},
		{
			Name:        "unique_email_id",
			Type:        types.String(),
			Description: "Unique email ID",
		},
		{
			Name:        "contact_id",
			Type:        types.String(),
			Description: "Contact ID",
		},
		{
			Name:        "full_name",
			Type:        types.String(),
			Description: "Full name",
		},
		{
			Name:        "web_id",
			Type:        types.Int(32),
			Description: "Web ID",
		},
		{
			Name:        "email_type",
			Type:        types.String().WithValues("html", "text"),
			Description: "Email type",
		},
		{
			Name:         "unsubscribe_reason",
			Type:         types.String(),
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
					{Name: "currency_code", Type: types.String().WithMaxLength(3), Description: "Currency code"},
				}), ReadOptional: true, Nullable: true, Description: "Ecommerce"},
			}),
			Description: "Stats",
		},
		{
			Name:        "ip_signup",
			Type:        types.IP(),
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
			Type:        types.IP(),
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
			Type:        types.String().WithMaxLength(5),
			Description: "Language",
		},
		{
			Name:        "vip",
			Type:        types.Boolean(),
			Description: "VIP status",
		},
		{
			Name:        "email_client",
			Type:        types.String(),
			Description: "Email client",
		},
		{
			Name: "location",
			Type: types.Object([]types.Property{
				{Name: "latitude", Type: types.Int(32), Description: "Latitude"},
				{Name: "longitude", Type: types.Int(32), Description: "Longitude"},
				{Name: "gmtoff", Type: types.Int(32), Description: "GMT offset"},
				{Name: "dstoff", Type: types.Int(32), Description: "DST offset"},
				{Name: "country_code", Type: types.String().WithMaxLength(2), Description: "Country code"},
				{Name: "timezone", Type: types.String(), Description: "Location timezone"},
				{Name: "region", Type: types.String(), Description: "Location region"},
			}),
			Description: "Location",
		},
		{
			Name: "marketing_permissions",
			Type: types.Object([]types.Property{
				{Name: "marketing_permission_id", Type: types.String(), Description: "ID"},
				{Name: "text", Type: types.String(), Description: "Text"},
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
				{Name: "created_by", Type: types.String(), Description: "Author"},
				{Name: "note", Type: types.String(), Description: "Content"},
			}),
			ReadOptional: true,
			Description:  "Last note",
		},
		{
			Name:        "source",
			Type:        types.String(),
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
				{Name: "name", Type: types.String(), Description: "Name"},
			})),
			Description: "Tags",
		},
	}
}
