//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package stripe implements the Stripe connector.
// (https://docs.stripe.com/api)
package stripe

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

const maxEventPayload = 1024 * 1024

// Connector icon.
var icon = "<svg></svg>"

var baseURL = "https://api.stripe.com"

type webhookSettings struct {
	id     string
	secret string
}

type innerSettings struct {
	APIKey  string
	webhook webhookSettings
}

type Stripe struct {
	conf     *meergo.AppConfig
	settings *innerSettings
}

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name: "Stripe",
		AsSource: &meergo.AsAppSource{
			Description: "Import customers as users",
			Targets:     meergo.Users,
			HasSettings: true,
		},
		AsDestination: &meergo.AsAppDestination{
			Description: "Export users as customers",
			Targets:     meergo.Users,
			HasSettings: true,
		},
		TermForUsers: "customers",
		BackoffPolicy: meergo.BackoffPolicy{
			// https://docs.stripe.com/api/errors
			"429 500 502 503 504": meergo.ExponentialStrategy(200 * time.Millisecond),
		},
		Icon:        icon,
		WebhooksPer: meergo.WebhooksPerConnection,
	}, New)
}

// New returns a new Stripe connector instance.
func New(conf *meergo.AppConfig) (*Stripe, error) {
	c := Stripe{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Value(conf.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Stripe connector")
		}
		if c.settings.webhook.secret == "" && conf.SetSettings != nil {
			err = c.setupWebhooksEndpoint()
			if err != nil {
				return nil, err
			}
		}
	}
	return &c, nil
}

// ReceiveWebhook receives a webhook request and returns its payloads.
func (stripe *Stripe) ReceiveWebhook(r *http.Request, role meergo.Role) ([]meergo.WebhookPayload, error) {

	// Extract signature from Stripe-Signature header.
	var timestamp time.Time
	var signatures [][]byte
	{
		parts := strings.Split(r.Header.Get("Stripe-Signature"), ",")
		for _, part := range parts {
			if strings.HasPrefix(part, "t=") {
				ts, err := strconv.ParseInt(part[2:], 10, 64)
				if err != nil {
					return nil, meergo.ErrWebhookUnauthorized
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
			return nil, meergo.ErrWebhookUnauthorized
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
	mac := hmac.New(sha256.New, []byte(stripe.settings.webhook.secret))
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
		return nil, meergo.ErrWebhookUnauthorized
	}

	var message struct {
		ID   string `json:"id"`
		Data struct {
			Object             map[string]any `json:"object"`
			PreviousAttributes map[string]any `json:"previous_attributes"`
		} `json:"data"`
		Type    string `json:"type"`
		Created int64  `json:"created"`
	}

	err = json.Unmarshal(body, &message)
	if err != nil {
		return nil, errors.New("webhook message is malformed")
	}

	var events []meergo.WebhookPayload
	tmp := time.UnixMilli(message.Created).UTC()
	switch message.Type {
	case "customer.created":
		event := meergo.UserCreateEvent{
			Timestamp:  tmp,
			User:       message.Data.Object["id"].(string),
			Properties: message.Data.Object,
		}
		events = append(events, event)
	case "customer.deleted":
		event := meergo.UserDeleteEvent{
			Timestamp: tmp,
			User:      message.Data.Object["id"].(string),
		}
		events = append(events, event)
	case "customer.updated":
		for modifiedAttributeName := range message.Data.PreviousAttributes {
			events = append(events, meergo.UserPropertyChangeEvent{
				Timestamp: tmp,
				User:      message.Data.Object["id"].(string),
				Name:      modifiedAttributeName,
				Value:     message.Data.Object[modifiedAttributeName],
			})
		}
	}

	return events, nil

}

// Records returns the records of the specified target.
func (stripe *Stripe) Records(ctx context.Context, _ meergo.Targets, _ types.Type, _ time.Time, _, _ []string, cursor string) ([]meergo.Record, string, error) {

	var body io.Reader
	if cursor != "" {
		form := url.Values{
			"starting_after": {cursor},
		}
		body = strings.NewReader(form.Encode())
	}

	var response struct {
		Data []map[string]any `json:"data"`
	}

	err := stripe.call(ctx, "GET", "/v1/customers", body, 200, &response)
	if err != nil {
		return nil, "", err
	}

	if len(response.Data) == 0 {
		return nil, "", io.EOF
	}

	users := make([]meergo.Record, len(response.Data))
	for i, customer := range response.Data {
		users[i] = meergo.Record{
			ID:             customer["id"].(string),
			Properties:     customer,
			LastChangeTime: time.Now().UTC(),
		}
	}
	cursor = users[len(users)-1].ID

	return users, cursor, nil
}

// Schema returns the schema of the specified target in the specified role.
func (stripe *Stripe) Schema(ctx context.Context, target meergo.Targets, role meergo.Role, eventType string) (types.Type, error) {
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

// ServeUI serves the connector's user interface.
func (stripe *Stripe) ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s innerSettings
		if stripe.settings != nil {
			s = *stripe.settings
		}
		settings, _ = json.Marshal(s)
	case "save":
		return nil, stripe.saveSettings(ctx, settings)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "APIKey", Label: "API Key", HelpText: "Your Stripe API key, which can be a live/test secret key or a restricted API key (see https://stripe.com/docs/keys)."},
		},
		Settings: settings,
	}

	return ui, nil
}

// Upsert updates or creates records in the app for the specified target.
func (stripe *Stripe) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {
	record := records.First()
	var body bytes.Buffer
	err := encodeRequest(&body, record.Properties, nil)
	if err != nil {
		return fmt.Errorf("cannot compute form-encoded request body: %s", err)
	}
	u := "/v1/customers"
	if record.ID != "" {
		u += "/" + record.ID
	}
	return stripe.call(ctx, "POST", u, &body, 200, nil)
}

func (stripe *Stripe) call(ctx context.Context, method, path string, body io.Reader, expectedStatus int, response any) error {
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return err
	}
	client := stripe.conf.HTTPClient
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+stripe.settings.APIKey)
	if req.Method == "POST" {
		req.Header.Set("Idempotency-Key", client.UUID())
	}
	res, err := client.DoIdempotent(req, true)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()
	if res.StatusCode != expectedStatus {
		var errorResponse stripeErrorResponse
		err := json.Decode(res.Body, &errorResponse)
		if err != nil {
			return err
		}
		errResponse := errorResponse.Error
		errResponse.statusCode = res.StatusCode
		return &errResponse
	}
	if response != nil {
		return json.Decode(res.Body, response)
	}
	return nil
}

// saveSettings validates and saves the settings.
func (stripe *Stripe) saveSettings(ctx context.Context, settings json.Value) error {
	var s innerSettings
	err := settings.Unmarshal(&s)
	if err != nil {
		return err
	}
	if s.APIKey == "" {
		return meergo.NewInvalidsettingsError("API key cannot be empty")
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = stripe.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	stripe.settings = &s
	return nil
}

// setupWebhooksEndpoint sets up the endpoint for webhooks.
// It can be called if stripe.settings is not nil and stripe.conf.SetSettings is
// not nil.
func (stripe *Stripe) setupWebhooksEndpoint() error {

	form := url.Values{
		"url":              {stripe.conf.WebhookURL},
		"enabled_events[]": {"customer.created", "customer.deleted", "customer.updated"},
	}
	body := strings.NewReader(form.Encode())

	response := struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
	}{}
	err := stripe.call(context.TODO(), "POST", "/v1/webhook_endpoints", body, 200, &response)
	if err != nil {
		return err
	}

	settings := innerSettings{
		APIKey: stripe.settings.APIKey,
		webhook: webhookSettings{
			id:     response.ID,
			secret: response.Secret,
		},
	}

	values, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	return stripe.saveSettings(context.TODO(), values)
}

type stripeErrorResponse struct {
	Error stripeError `json:"error"`
}

type stripeError struct {
	statusCode int
	Type       string `json:"type"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Param      string `json:"param"`
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
		Name: "id",
		Type: types.Text(),
	},
	{
		Name: "address",
		Type: types.Object([]types.Property{
			{
				Name:     "city",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "country",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "line1",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "line2",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "postal_code",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "state",
				Type:     types.Text(),
				Nullable: true,
			},
		}),
		Nullable: true,
	},
	{
		Name:     "description",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "email",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name: "metadata",
		Type: types.Map(types.Text()),
	},
	{
		Name:     "name",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name:     "phone",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name: "shipping",
		Type: types.Object([]types.Property{
			{
				Name: "address",
				Type: types.Object([]types.Property{
					{
						Name:     "city",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "country",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "line1",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "line2",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "postal_code",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "state",
						Type:     types.Text(),
						Nullable: true,
					},
				}),
				Nullable: true,
			},
			{
				Name:     "name",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "phone",
				Type:     types.Text(),
				Nullable: true,
			},
		}),
		Nullable: true,
	},
	{
		Name: "object",
		Type: types.Text(),
	},
	{
		Name: "balance",
		Type: types.Int(32),
	},
	{
		Name: "created",
		Type: types.Int(64),
	},
	{
		Name: "currency",
		Type: types.Text(),
	},
	{
		Name:     "default_source",
		Type:     types.Text(),
		Nullable: true,
	},
	{
		Name: "delinquent",
		Type: types.Boolean(),
	},
	{
		Name: "discount",
		Type: types.Object([]types.Property{
			{
				Name: "id",
				Type: types.Text(),
			},
			{
				Name: "coupon",
				Type: types.Object([]types.Property{
					{
						Name: "id",
						Type: types.Text(),
					},
					{
						Name:     "amount_off",
						Type:     types.Int(32),
						Nullable: true,
					},
					{
						Name: "currency",
						Type: types.Text(),
					},
					{
						Name: "duration",
						Type: types.Text(),
					},
					{
						Name: "duration_in_months",
						Type: types.Int(32),
					},
					{
						Name: "metadata",
						Type: types.Map(types.Text()),
					},
					{
						Name: "name",
						Type: types.Text(),
					},
					{
						Name: "percent_off",
						Type: types.Float(64),
					},
					{
						Name: "object",
						Type: types.Text(),
					},
					{
						Name: "created",
						Type: types.Int(64),
					},
					{
						Name: "livemode",
						Type: types.Boolean(),
					},
					{
						Name:     "max_redemptions",
						Type:     types.Int(32),
						Nullable: true,
					},
					{
						Name:     "redeem_by",
						Type:     types.Int(64),
						Nullable: true,
					},
					{
						Name: "times_redeemed",
						Type: types.Int(32),
					},
					{
						Name: "valid",
						Type: types.Boolean(),
					},
				}),
			},
			{
				Name: "customer",
				Type: types.Text(),
			},
			{
				Name: "end",
				Type: types.Int(64),
			},
			{
				Name: "start",
				Type: types.Int(64),
			},
			{
				Name:     "subscription",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name: "object",
				Type: types.Text(),
			},
			{
				Name:     "checkout_session",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "invoice",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "invoice_item",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "promotion_code",
				Type:     types.Text(),
				Nullable: true,
			},
		}),
		Nullable: true,
	},
	{
		Name: "invoice_prefix",
		Type: types.Text(),
	},
	{
		Name: "invoice_settings",
		Type: types.Object([]types.Property{
			{
				Name: "custom_fields",
				Type: types.Array(types.Object([]types.Property{
					{
						Name:     "name",
						Type:     types.Text(),
						Nullable: true,
					},
					{
						Name:     "value",
						Type:     types.Text(),
						Nullable: true,
					},
				})),
				Nullable: true,
			},
			{
				Name:     "default_payment_method",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name:     "footer",
				Type:     types.Text(),
				Nullable: true,
			},
			{
				Name: "rendering_options",
				Type: types.Object([]types.Property{
					{
						Name:     "amount_tax_display",
						Type:     types.Text(),
						Nullable: true,
					},
				}),
				Nullable: true,
			},
		}),
	},
	{
		Name: "livemode",
		Type: types.Boolean(),
	},
	{
		Name: "next_invoice_sequence",
		Type: types.Int(32),
	},
	{
		Name: "preferred_locales",
		Type: types.Array(types.Text()),
	},
	{
		Name: "tax_exempt",
		Type: types.Text(),
	},
	{
		Name:     "test_clock",
		Type:     types.Text(),
		Nullable: true,
	},
})
