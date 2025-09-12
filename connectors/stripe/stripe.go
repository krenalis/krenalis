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
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
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
		TimeLayouts: meergo.TimeLayouts{
			DateTime: "unix",
		},
		Icon: icon,
	}, New)
}

// New returns a new Stripe connector instance.
func New(env *meergo.AppEnv) (*Stripe, error) {
	c := Stripe{env: env}
	if len(env.Settings) > 0 {
		err := json.Value(env.Settings).Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Stripe connector")
		}
	}
	return &c, nil
}

type Stripe struct {
	env      *meergo.AppEnv
	settings *innerSettings
}

type innerSettings struct {
	APIKey string
}

// RecordSchema returns the schema of the specified target and role.
func (stripe *Stripe) RecordSchema(ctx context.Context, target meergo.Targets, role meergo.Role) (types.Type, error) {
	// Currently the user schema is the standard schema of the user returned
	// when the api is called without specifying the "expand" field.
	//
	// Stripe gives the ability to use this additional "expand" field when
	// calling its APIs to retrieve additional information:
	// https://stripe.com/docs/api/expanding_objects
	if role == meergo.Source {
		return sourceSchema, nil
	}
	return destinationSchema, nil
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
	bb := stripe.env.HTTPClient.GetBodyBuffer(meergo.NoEncoding)
	defer bb.Close()
	record := records.First()
	mode := modeCreate
	if record.ID != "" {
		mode = modeUpdate
	}
	err := encodeRequest(bb, record.Properties, nil, mode)
	if err != nil {
		return fmt.Errorf("cannot compute form-encoded request body: %s", err)
	}
	u := "/v1/customers"
	if record.ID != "" {
		u += "/" + record.ID
	}
	return stripe.call(ctx, "POST", u, bb, 200, nil)
}

func (stripe *Stripe) call(ctx context.Context, method, path string, bb *meergo.BodyBuffer, expectedStatus int, response any) error {
	req, err := bb.NewRequest(ctx, method, baseURL+path)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+stripe.settings.APIKey)
	if req.Method == "POST" {
		req.Header.Set("Idempotency-Key", meergo.UUID()) // mark the request as idempotent
	}
	res, err := stripe.env.HTTPClient.Do(req)
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
	err = stripe.env.SetSettings(ctx, b)
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

type upsertMode int

const (
	modeCreate upsertMode = iota + 1
	modeUpdate
)

func encodeRequest(bb *meergo.BodyBuffer, request map[string]interface{}, parents []string, mode upsertMode) error {

	if len(request) > 0 {

		for field, value := range request {

			// Ignore fields not matching with create/update mode.
			switch mode {
			// When creating, ignore fields specific for updating.
			case modeCreate:
				if field == "default_source" && len(parents) == 0 {
					continue
				}
			// When updating, ignore fields specific for creation.
			case modeUpdate:
				switch {
				case
					field == "payment_method" && len(parents) == 0,
					field == "tax_id_data" && len(parents) == 0,
					field == "test_clock" && len(parents) == 0:
					continue
				}
			}

			switch v := value.(type) {
			case bool, string, int, nil:
				if bb.Len() > 0 {
					bb.WriteByte('&')
				}
				writePath(bb, append(parents, field))
				bb.WriteByte('=')
			case map[string]interface{}:
				return encodeRequest(bb, v, append(parents, field), mode)
			default:
				return errors.New("unsupported type")
			}
			switch v := value.(type) {
			case bool:
				if v {
					bb.WriteString("true")
				} else {
					bb.WriteString("false")
				}
			case string:
				bb.WriteString(url.QueryEscape(v))
			case int:
				bb.WriteString(strconv.Itoa(v))
			case nil:
				bb.WriteString("")
			}

		}
	}

	return nil
}

func writePath(bb *meergo.BodyBuffer, path []string) {
	for i, v := range path {
		switch i {
		case 0:
			bb.WriteString(url.QueryEscape(v))
		default:
			bb.WriteByte('[')
			bb.WriteString(url.QueryEscape(v))
			bb.WriteByte(']')
		}
	}
}
