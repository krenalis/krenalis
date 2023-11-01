//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package stripe implements the Stripe connector.
package stripe

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"chichi/connector"
	"chichi/connector/types"
	"chichi/connector/ui"
)

const maxEventPayload = 1024 * 1024

// Connector icon.
var icon = "<svg></svg>"

var baseURL = "https://api.stripe.com"

type webhookSettings struct {
	id     string
	secret string
}

type settings struct {
	APIKey  string `json:"api_key"`
	webhook webhookSettings
}

type connection struct {
	conf     *connector.AppConfig
	settings *settings
}

// Make sure it implements the AppUsersConnection interface.
var _ interface {
	connector.AppUsersConnection
} = (*connection)(nil)

func init() {
	connector.RegisterApp(connector.App{
		Name:                   "Stripe",
		SourceDescription:      "import customers as users",
		DestinationDescription: "export users as customers",
		TermForUsers:           "customers",
		Icon:                   icon,
		WebhooksPer:            connector.WebhooksPerSource,
	}, open)
}

// open opens a Stripe connection.
func open(conf *connector.AppConfig) (*connection, error) {
	c := connection{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Stripe connection")
		}
	}
	err := c.setupWebhooksEndpoint()
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Resource returns the resource.
func (c *connection) Resource(ctx context.Context) (string, error) {
	return "", nil
}

// CreateUser creates a user with the given properties.
func (c *connection) CreateUser(ctx context.Context, properties connector.Properties) error {

	var body bytes.Buffer
	err := encodeRequest(&body, properties, nil)
	if err != nil {
		return fmt.Errorf("cannot compute form-encoded request body: %s", err)
	}

	return c.call(ctx, "POST", "/v1/customers", &body, 200, nil)
}

// ReceiveWebhook receives a webhook request and returns its events. It returns
// the ErrWebhookUnauthorized error is the request was not authorized. The
// context is the request's context.
func (c *connection) ReceiveWebhook(r *http.Request) ([]connector.WebhookPayload, error) {

	// Extract signature from Stripe-Signature header.
	var timestamp time.Time
	var signatures [][]byte
	{
		parts := strings.Split(r.Header.Get("Stripe-Signature"), ",")
		for _, part := range parts {
			if strings.HasPrefix(part, "t=") {
				ts, err := strconv.ParseInt(part[2:], 10, 64)
				if err != nil {
					return nil, connector.ErrWebhookUnauthorized
				}
				timestamp = time.Unix(ts, 0)
				continue
			}
			if strings.HasPrefix(part, "v1=") {
				sig, err := hex.DecodeString(part[3:])
				if err == nil {
					signatures = append(signatures, sig)
				}
			}
		}
		if len(signatures) == 0 {
			return nil, connector.ErrWebhookUnauthorized
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxEventPayload+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxEventPayload {
		return nil, errors.New("webhook payload is too large")
	}

	// Calculate the message signature and check if it matches one of the
	// signatures in the header.
	mac := hmac.New(sha256.New, []byte(c.settings.webhook.secret))
	mac.Write([]byte(fmt.Sprintf("%d", timestamp.Unix())))
	mac.Write([]byte("."))
	mac.Write(body)
	messageSignature := mac.Sum(nil)

	var isValidSignature bool
	for _, sig := range signatures {
		if hmac.Equal(messageSignature, sig) {
			isValidSignature = true
			break
		}
	}
	if !isValidSignature {
		return nil, connector.ErrWebhookUnauthorized
	}

	var message struct {
		Id   string
		Data struct {
			Object             map[string]any
			PreviousAttributes map[string]any `json:"previous_attributes"`
		}
		Type    string
		Created int64
	}

	err = json.Unmarshal(body, &message)
	if err != nil {
		return nil, errors.New("webhook message is malformed")
	}

	var events []connector.WebhookPayload
	tmp := time.UnixMilli(message.Created).UTC()
	switch message.Type {
	case "customer.created":
		event := connector.UserCreateEvent{
			Timestamp:  tmp,
			User:       message.Data.Object["id"].(string),
			Properties: message.Data.Object,
		}
		events = append(events, event)
	case "customer.deleted":
		event := connector.UserDeleteEvent{
			Timestamp: tmp,
			User:      message.Data.Object["id"].(string),
		}
		events = append(events, event)
	case "customer.updated":
		for modifiedAttributeName := range message.Data.PreviousAttributes {
			events = append(events, connector.UserPropertyChangeEvent{
				Timestamp: tmp,
				User:      message.Data.Object["id"].(string),
				Name:      modifiedAttributeName,
				Value:     message.Data.Object[modifiedAttributeName],
			})
		}
	}

	return events, nil

}

// UserSchema returns the user schema.
func (c *connection) UserSchema(ctx context.Context) (types.Type, error) {
	// docs: https://stripe.com/docs/api/customers/object
	//
	// currently the user schema is the standard schema of the user returned
	// when the api is called without specifying the "expand" field.
	//
	// Stripe gives the ability to use this additional "expand" field when
	// calling its APIs to retrieve additional information:
	// https://stripe.com/docs/api/expanding_objects

	return schema, nil
}

// UpdateUser updates the user with identifier id setting the given properties.
func (c *connection) UpdateUser(ctx context.Context, id string, properties connector.Properties) error {

	var body bytes.Buffer
	err := encodeRequest(&body, properties, nil)
	if err != nil {
		return fmt.Errorf("cannot compute form-encoded request body: %s", err)
	}

	return c.call(ctx, "POST", "/v1/customers/"+id, &body, 200, nil)
}

// Users returns the users starting from the given cursor.
func (c *connection) Users(ctx context.Context, properties []string, cursor connector.Cursor) ([]connector.Object, string, error) {

	var body io.Reader
	if cursor.ID != "" {
		form := url.Values{
			"starting_after": {cursor.ID},
		}
		body = strings.NewReader(form.Encode())
	}

	var response struct {
		Data []map[string]any
	}

	err := c.call(ctx, "GET", "/v1/customers", body, 200, &response)
	if err != nil {
		return nil, "", err
	}

	if len(response.Data) == 0 {
		return nil, "", io.EOF
	}

	objects := make([]connector.Object, len(response.Data))
	for i, customer := range response.Data {
		objects[i] = connector.Object{
			ID:         customer["id"].(string),
			Properties: customer,
			Timestamp:  time.Now(),
		}
	}

	return objects, "", nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if c.settings != nil {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := c.ValidateSettings(ctx, values)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, c.conf.SetSettings(ctx, s)
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "api_key", Label: "API Key", HelpText: "Your Stripe API key, which can be a live/test secret key or a restricted API key (see https://stripe.com/docs/keys)."},
		},
		Actions: []ui.Action{{Event: "save", Text: "Save", Variant: "primary"}},
		Values:  values,
	}

	return form, nil, nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(ctx context.Context, values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	if s.APIKey == "" {
		return nil, ui.Errorf("API key cannot be empty")
	}
	return json.Marshal(&s)
}

func (c *connection) setupWebhooksEndpoint() error {
	if c.conf.SetSettings == nil || c.settings.webhook.secret != "" {
		return nil
	}

	form := url.Values{
		"url":              {c.conf.WebhookURL},
		"enabled_events[]": {"customer.created", "customer.deleted", "customer.updated"},
	}
	body := strings.NewReader(form.Encode())

	response := struct {
		ID     string
		Secret string
	}{}
	err := c.call(context.TODO(), "POST", "/v1/webhook_endpoints", body, 200, &response)
	if err != nil {
		return err
	}

	settings := settings{
		APIKey: c.settings.APIKey,
		webhook: webhookSettings{
			id:     response.ID,
			secret: response.Secret,
		},
	}

	values, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	// Save the settings.
	s, err := c.ValidateSettings(context.TODO(), values)
	if err != nil {
		return err
	}

	return c.conf.SetSettings(context.TODO(), s)
}

func (c *connection) call(ctx context.Context, method, path string, body io.Reader, expectedStatus int, response any) error {
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+c.settings.APIKey)
	res, err := c.conf.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()
	if res.StatusCode != expectedStatus {
		var errorResponse stripeErrorResponse
		dec := json.NewDecoder(res.Body)
		_ = dec.Decode(&errorResponse)
		err := errorResponse.Error
		err.statusCode = res.StatusCode
		return &err
	}
	if response != nil {
		dec := json.NewDecoder(res.Body)
		return dec.Decode(response)
	}
	return nil
}

type stripeErrorResponse struct {
	Error stripeError
}

type stripeError struct {
	statusCode int
	Type       string
	Code       string
	Message    string
	Param      string
}

func (err *stripeError) Error() string {
	return fmt.Sprintf("unexpected error from Stripe: (%d) %s", err.statusCode, err.Message)
}

func encodeRequest(body *bytes.Buffer, request map[string]interface{}, parents []string) error {
	if len(request) > 0 {
		for field, value := range request {

			switch v := value.(type) {
			case bool, string, int, nil:
				if body.Len() > 0 {
					body.WriteByte('&')
				}
				writePath(body, append(parents, field))
				body.WriteByte('=')
			case map[string]interface{}:
				return encodeRequest(body, v, append(parents, field))
			default:
				return errors.New("unsupported type")
			}

			switch v := value.(type) {
			case bool:
				if v {
					body.WriteString("true")
				} else {
					body.WriteString("false")
				}
			case string:
				body.WriteString(url.QueryEscape(v))
			case int:
				body.WriteString(strconv.Itoa(v))
			case nil:
				body.WriteString("")
			}
		}
	}
	return nil
}

func writePath(body *bytes.Buffer, path []string) {
	for i, v := range path {
		switch i {
		case 0:
			body.WriteString(url.QueryEscape(v))
		default:
			body.WriteByte('[')
			body.WriteString(url.QueryEscape(v))
			body.WriteByte(']')
		}
	}
}

var schema = types.Object([]types.Property{
	{
		Name:  "id",
		Label: "ID",
		Type:  types.Text(),
	},
	{
		Name:  "address",
		Label: "Address",
		Type: types.Object([]types.Property{
			{
				Name:     "city",
				Label:    "Address City",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "country",
				Label:    "Address Country",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "line1",
				Label:    "Address Line1",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "line2",
				Label:    "Address Line2",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "postal_code",
				Label:    "Address Postal Code",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "state",
				Label:    "Address State",
				Type:     types.Text(),
				Nullable: true,
			},
		}),
		Nullable: true,
	},
	{
		Name:     "description",
		Label:    "Description",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "email",
		Label:    "Email",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:  "metadata",
		Label: "Metadata",
		Type:  types.Map(types.Text()),
	},
	{
		Name:     "name",
		Label:    "Name",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "phone",
		Label:    "Phone",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:  "shipping",
		Label: "Shipping",
		Type: types.Object([]types.Property{
			{
				Name:  "address",
				Label: "Shipping Address",
				Type: types.Object([]types.Property{
					{
						Name:     "city",
						Label:    "Shipping Address City",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "country",
						Label:    "Shipping Address Country",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "line1",
						Label:    "Shipping Address Line1",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "line2",
						Label:    "Shipping Address Line2",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "postal_code",
						Label:    "Shipping Address Postal Code",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "state",
						Label:    "Shipping Address State",
						Type:     types.Text(),
						Nullable: true,
					},
				}),
				Nullable: true,
			},
			{
				Name:     "name",
				Label:    "Shipping Name",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "phone",
				Label:    "Shipping Phone",
				Type:     types.Text(),
				Nullable: true,
			},
		}),
		Nullable: true,
	},
	{
		Name:  "object",
		Label: "Object",
		Type:  types.Text(),
	},
	{
		Name:  "balance",
		Label: "Balance",
		Type:  types.Int(),
	},
	{
		Name:  "created",
		Label: "Created",
		Type:  types.Int64(),
	},
	{
		Name:  "currency",
		Label: "Currency",
		Type:  types.Text(),
	},
	{
		Name:     "default_source",
		Label:    "Default Source",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:  "delinquent",
		Label: "Delinquent",
		Type:  types.Boolean(),
	},
	{
		Name:  "discount",
		Label: "Discount",
		Type: types.Object([]types.Property{
			{
				Name:  "id",
				Label: "Discount ID",
				Type:  types.Text(),
			},
			{
				Name:  "coupon",
				Label: "Coupon",
				Type: types.Object([]types.Property{
					{
						Name:  "id",
						Label: "Coupon ID",
						Type:  types.Text(),
					},
					{
						Name:     "amount_off",
						Label:    "Coupon Amount Off",
						Type:     types.Int(),
						Nullable: true,
					},
					{
						Name:  "currency",
						Label: "Coupon Currency",
						Type:  types.Text(),
					},
					{
						Name:  "duration",
						Label: "Coupon Duration",
						Type:  types.Text(),
					},
					{
						Name:  "duration_in_months",
						Label: "Coupon Duration In Months",
						Type:  types.Int(),
					},
					{
						Name:  "metadata",
						Label: "Coupon Metadata",
						Type:  types.Map(types.Text()),
					},
					{
						Name:  "name",
						Label: "Coupon Name",
						Type:  types.Text(),
					},
					{
						Name:  "percent_off",
						Label: "Coupon Percent Off",
						Type:  types.Float(),
					},
					{
						Name:  "object",
						Label: "Coupon Object",
						Type:  types.Text(),
					},
					{
						Name:  "created",
						Label: "Coupon Created",
						Type:  types.Int64(),
					},
					{
						Name:  "livemode",
						Label: "Coupon Live Mode",
						Type:  types.Boolean(),
					},
					{
						Name:     "max_redemptions",
						Label:    "Coupon Max Redemptions",
						Type:     types.Int(),
						Nullable: true,
					},
					{
						Name:     "redeem_by",
						Label:    "Coupon Redeem By",
						Type:     types.Int64(),
						Nullable: true,
					},
					{
						Name:  "times_redeemed",
						Label: "Coupon Times Redeemed",
						Type:  types.Int(),
					},
					{
						Name:  "valid",
						Label: "Coupon Valid",
						Type:  types.Boolean(),
					},
				}),
			},
			{
				Name:  "customer",
				Label: "Discount Customer",
				Type:  types.Text(),
			},
			{
				Name:  "end",
				Label: "Discount End",
				Type:  types.Int64(),
			},
			{
				Name:  "start",
				Label: "Discount Start",
				Type:  types.Int64(),
			},
			{
				Name:     "subscription",
				Label:    "Discount Subscription",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:  "object",
				Label: "Discount Object",
				Type:  types.Text(),
			},
			{
				Name:     "checkout_session",
				Label:    "Discount Checkout Session",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "invoice",
				Label:    "Discount Invoice",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "invoice_item",
				Label:    "Discount Invoice Item",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "promotion_code",
				Label:    "Discount Invoice Item",
				Type:     types.Text(),
				Nullable: true,
			},
		}),
		Nullable: true,
	},
	{
		Name:  "invoice_prefix",
		Label: "Invoice Prefix",
		Type:  types.Text(),
	},
	{
		Name:  "invoice_settings",
		Label: "Invoice Settings",
		Type: types.Object([]types.Property{
			{
				Name:  "custom_fields",
				Label: "Custom Fields",
				Type: types.Array(types.Object([]types.Property{
					{
						Name:     "name",
						Label:    "Name",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "value",
						Label:    "Value",
						Type:     types.Text(),
						Nullable: true,
					},
				})),
				Nullable: true,
			},
			{
				Name:     "default_payment_method",
				Label:    "Default Payment Method",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "footer",
				Label:    "Footer",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:  "rendering_options",
				Label: "Rendering Options",
				Type: types.Object([]types.Property{
					{
						Name:     "amount_tax_display",
						Label:    "Amount Tax Display",
						Type:     types.Text(),
						Nullable: true,
					},
				}),
				Nullable: true,
			},
		}),
	},
	{
		Name:  "livemode",
		Label: "Live mode",
		Type:  types.Boolean(),
	},
	{
		Name:  "next_invoice_sequence",
		Label: "Next Invoice Sequence",
		Type:  types.Int(),
	},
	{
		Name:  "preferred_locales",
		Label: "Preferred Locales",
		Type:  types.Array(types.Text()),
	},
	{
		Name:  "tax_exempt",
		Label: "Tax Exempt",
		Type:  types.Text(),
	},
	{
		Name:     "test_clock",
		Label:    "Test Clock",
		Type:     types.Text(),
		Nullable: true,
	},
})
