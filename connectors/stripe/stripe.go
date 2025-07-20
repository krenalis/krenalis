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
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Connector icon.
var icon = "<svg></svg>"

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

var baseURL = "https://api.stripe.com"

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name:       "Stripe",
		Categories: meergo.CategoryEcommerce,
		AsSource: &meergo.AsAppSource{
			Targets:     meergo.TargetUser,
			HasSettings: true,
			Documentation: meergo.ConnectorRoleDocumentation{
				Summary:  "Import customers as users",
				Overview: sourceOverview,
			},
		},
		AsDestination: &meergo.AsAppDestination{
			Targets:     meergo.TargetUser,
			HasSettings: true,
			Documentation: meergo.ConnectorRoleDocumentation{
				Summary:  "Export users as customers",
				Overview: destinationOverview,
			},
		},
		Terms: meergo.AppTerms{
			User:  "customer",
			Users: "customers",
		},
		EndpointGroups: []meergo.EndpointGroup{{
			// https://docs.stripe.com/rate-limits
			RateLimit: meergo.RateLimit{RequestsPerSecond: 100, Burst: 200},
			// https://docs.stripe.com/api/errors
			RetryPolicy: meergo.RetryPolicy{
				"429":             meergo.ExponentialStrategy(meergo.Slowdown, 200*time.Millisecond),
				"500 502 503 504": meergo.ExponentialStrategy(meergo.NetFailure, 200*time.Millisecond),
			},
		}},
		Icon: icon,
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
	}
	return &c, nil
}

type Stripe struct {
	conf     *meergo.AppConfig
	settings *innerSettings
}

type innerSettings struct {
	APIKey string
}

// RecordSchema returns the schema of the specified target and role.
func (stripe *Stripe) RecordSchema(ctx context.Context, target meergo.Targets, role meergo.Role) (types.Type, error) {
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

// Records returns the records of the specified target.
func (stripe *Stripe) Records(ctx context.Context, _ meergo.Targets, _ time.Time, _, _ []string, cursor string, _ types.Type) ([]meergo.Record, string, error) {

	path := "/v1/customers"
	if cursor != "" {
		path += "?starting_after=" + url.QueryEscape(cursor)
	}

	var response struct {
		Data []map[string]any `json:"data"`
	}

	err := stripe.call(ctx, "GET", path, nil, 200, &response)
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
	return stripe.call(ctx, "POST", u, body.Bytes(), 200, nil)
}

func (stripe *Stripe) call(ctx context.Context, method, path string, body []byte, expectedStatus int, response any) error {
	var b io.Reader
	if body != nil {
		b = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, b)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+stripe.settings.APIKey)
	if req.Method == "POST" {
		// Mark the request as idempotent.
		req.Header.Set("Idempotency-Key", meergo.UUID())
		if body != nil {
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(body)), nil
			}
		}
	}
	res, err := stripe.conf.HTTPClient.Do(req)
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
	if n := len(s.APIKey); n < 1 || n > 100 {
		return meergo.NewInvalidSettingsError("API key length must be in [1,100]")
	}
	for i := 0; i < len(s.APIKey); i++ {
		c := s.APIKey[i]
		// ASCII characters with decimal codes from 33 (!) to 126 (~),
		// inclusive, are printable characters. The space character, having
		// decimal code 32, is therefore excluded from the range of accepted
		// characters, and this is intentional.
		if c < 33 || c > 126 {
			return meergo.NewInvalidSettingsError("API key must contain only valid characters")
		}
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
