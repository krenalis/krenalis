// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package stripe provides a connector for Stripe.
// (https://docs.stripe.com/api)
//
// Stripe is a trademark of Stripe, Inc.
// This connector is not affiliated with or endorsed by Stripe, Inc.
package stripe

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

var baseURL = "https://api.stripe.com"

func init() {
	connectors.RegisterAPI(connectors.APISpec{
		Code:       "stripe",
		Label:      "Stripe",
		Categories: connectors.CategorySaaS,
		AsSource: &connectors.AsAPISource{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Import customers as users",
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsAPIDestination{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Export users as customers",
				Overview: destinationOverview,
			},
		},
		Terms: connectors.APITerms{
			User:  "Customer",
			Users: "Customers",
		},
		EndpointGroups: []connectors.EndpointGroup{{
			// https://docs.stripe.com/rate-limits
			RateLimit: connectors.RateLimit{RequestsPerSecond: 100, Burst: 200},
			// https://docs.stripe.com/api/errors
			RetryPolicy: connectors.RetryPolicy{
				"429":             connectors.ExponentialStrategy(connectors.Slowdown, 200*time.Millisecond),
				"500 502 503 504": connectors.ExponentialStrategy(connectors.NetFailure, 200*time.Millisecond),
			},
		}},
		TimeLayouts: connectors.TimeLayouts{
			DateTime: "unix",
		},
	}, New)
}

// New returns a new connector instance for Stripe.
func New(env *connectors.APIEnv) (*Stripe, error) {
	c := Stripe{env: env}
	if len(env.Settings) > 0 {
		err := env.Settings.Unmarshal(&c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of connector for Stripe")
		}
	}
	return &c, nil
}

type Stripe struct {
	env      *connectors.APIEnv
	settings *innerSettings
}

type innerSettings struct {
	APIKey string `json:"apiKey"`
}

// RecordSchema returns the schema of the specified target and role.
func (stripe *Stripe) RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {
	if role == connectors.Source {
		return sourceSchema, nil
	}
	return destinationSchema, nil
}

// Records returns the records of the specified target.
func (stripe *Stripe) Records(ctx context.Context, _ connectors.Targets, _ time.Time, _ []string, cursor string, _ types.Type) ([]connectors.Record, string, error) {

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

	users := make([]connectors.Record, len(response.Data))
	for i, attributes := range response.Data {
		id, _ := attributes["id"].(string)
		if id == "" {
			return nil, "", errors.New("unexpected customer identifier from Stripe")
		}
		users[i] = connectors.Record{
			ID:             id,
			Attributes:     attributes,
			LastChangeTime: time.Now().UTC(),
		}
	}
	cursor = users[len(users)-1].ID

	return users, cursor, nil
}

// ServeUI serves the connector's user interface.
func (stripe *Stripe) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {

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
		return nil, connectors.ErrUIEventNotExist
	}

	ui := &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Input{Name: "apiKey", Label: "API Key", HelpText: "Your Stripe API key, which can be a live/test secret key or a restricted API key (see https://stripe.com/docs/keys)."},
		},
		Settings: settings,
	}

	return ui, nil
}

var arrayIndex = regexp.MustCompile(`^\w+\[(\d+)\]`)
var localeCode = regexp.MustCompile(`^[a-z]{2}(?:-[A-Z]{2})?$`)

// Upsert updates or creates records in the API for the specified target.
func (stripe *Stripe) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records) error {

	record := records.First()

	// Validate 'metadata' property.
	if metadata, ok := record.Attributes["metadata"].(map[string]any); ok {
		if len(metadata) > 50 {
			return errors.New("«metadata» contains more than 50 keys")
		}
		for k := range metadata {
			if k == "" {
				return errors.New("«metadata» contains an empty key")
			}
			if utf8.RuneCountInString(k) > 40 {
				return errors.New("«metadata» contains a key longer than 40 characters")
			}
			if strings.ContainsAny(k, "[]") {
				return errors.New("«metadata» contains a key with square brackets")
			}
		}
	}
	// Validate 'invoice_prefix'.
	if prefix, ok := record.Attributes["invoice_prefix"].(string); ok {
		for i := 0; i < len(prefix); i++ {
			if c := prefix[i]; '0' <= c && c <= '9' || 'A' <= c && c <= 'Z' {
				continue
			}
			return errors.New("«invoice_prefix» does not contain only uppercase letters and numbers")
		}
		if len(prefix) < 3 {
			return errors.New("«invoice_prefix» is shorter than 3 characters")
		}
	}
	// Validate 'preferred_locales'.
	if locales, ok := record.Attributes["preferred_locales"].([]any); ok {
		for _, lc := range locales {
			if !localeCode.MatchString(lc.(string)) {
				return errors.New("«preferred_locales» contains an invalid locale identifier")
			}
		}
	}

	// Drop create/update-only fields.
	if record.ID == "" {
		delete(record.Attributes, "default_source")
		if tax, ok := record.Attributes["tax"].(map[string]any); ok {
			if validate, ok := tax["validate_location"]; ok && validate == "auto" {
				delete(tax, "validate_location")
			}
		}
	} else {
		delete(record.Attributes, "payment_method")
		delete(record.Attributes, "tax_id_data")
	}

	// Stripe requires empty arrays and maps to be serialized as empty strings.
	// Apply this only when updating a customer.
	if record.ID != "" {
		if metadata, ok := record.Attributes["metadata"].(map[string]any); ok && len(metadata) == 0 {
			record.Attributes["metadata"] = nil
		}
		if settings, ok := record.Attributes["invoice_settings"].(map[string]any); ok {
			if fields, ok := settings["custom_fields"].([]any); ok && len(fields) == 0 {
				settings["custom_fields"] = nil
			}
		}
		if locales, ok := record.Attributes["preferred_locales"].([]any); ok && len(locales) == 0 {
			record.Attributes["preferred_locales"] = nil
		}
	}

	bb := stripe.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()
	encodeAttributes(bb, record.Attributes)

	u := "/v1/customers"
	if record.ID != "" {
		u += "/" + record.ID
	}

	err := stripe.call(ctx, "POST", u, bb, 200, nil)
	if err != nil {
		if sErr, ok := err.(*stripeError); ok {
			if sErr.Type == "invalid_request_error" {
				switch sErr.Param {
				case "email":
					return errors.New("«email» is not a valid email address")
				case "source":
					// Stripe returns "source" instead of "default_source" when the resource is missing.
					sErr.Param = "default_source"
					fallthrough
				case "default_source":
					if sErr.Code == "resource_missing" {
						return errors.New("«default_source» is not valid or does not match any resource")
					}
				case "invoice_settings[default_payment_method]":
					if sErr.Code == "resource_missing" {
						return errors.New("«invoice_settings.default_payment_method» does not match any payment method")
					}
				case "invoice_settings[rendering_options][template]":
					if sErr.Code == "resource_missing" {
						return errors.New("«invoice_settings.rendering_options.template» does not match any invoice rendering template")
					}
				case "tax[ip_address]":
					return errors.New("«tax.ip_address» is not a valid IP address")
				}
				if sErr.Code == "tax_id_invalid" {
					var typ string
					if matches := arrayIndex.FindStringSubmatch(sErr.Param); matches != nil {
						if i, err := strconv.Atoi(matches[1]); err == nil && i >= 0 {
							if data, ok := record.Attributes["tax_id_data"].([]any); ok && i < len(data) {
								typ, _ = data[i].(map[string]any)["type"].(string)
							}
						}
					}
					return fmt.Errorf("«tax_id_data» contains an invalid value for tax ID type «\"%s\"»", typ)
				}
				return fmt.Errorf("«%s» is invalid: %s", sErr.Param, sErr.Code)
			}
			return fmt.Errorf("Stripe API returned an error: %q", sErr.Message)
		}
		return err
	}

	return nil
}

func (stripe *Stripe) call(ctx context.Context, method, path string, bb *connectors.BodyBuffer, expectedStatus int, response any) error {
	req, err := bb.NewRequest(ctx, method, baseURL+path)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+stripe.settings.APIKey)
	req.Header.Set("Stripe-Version", "2025-08-27.basil")

	if req.Method == "POST" {
		req.Header.Set("Idempotency-Key", connectors.UUID()) // mark the request as idempotent
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
	if n := len(s.APIKey); n < 1 || n > 200 {
		return connectors.NewInvalidSettingsError("API key length must be in [1,200]")
	}
	for i := 0; i < len(s.APIKey); i++ {
		c := s.APIKey[i]
		// ASCII characters with decimal codes from 33 (!) to 126 (~),
		// inclusive, are printable characters. The space character, having
		// decimal code 32, is therefore excluded from the range of accepted
		// characters, and this is intentional.
		if c < 33 || c > 126 {
			return connectors.NewInvalidSettingsError("API key must contain only valid characters")
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

const maxDestinationSchemaDepth = 4 // maximum nesting depth of the destination schema

// encodeAttributes encodes attributes as application/x-www-form-urlencoded,
// compatible with Stripe, and writes the result to dst.
//
// Only destination-schema types are serialized: text, bool, array, object, and
// map. Accordingly, property values must be nil or one of: string, int,
// map[string]any, or []any. Map keys must be non-empty.
func encodeAttributes(dst *connectors.BodyBuffer, attributes map[string]any) {

	type seg struct {
		name  string
		index int // >=0 for numbered array entries, -1 for []
	}
	var path [maxDestinationSchemaDepth]seg
	var depth int
	var wrote bool

	const noIndex = -1

	var buf [64]byte // numeric scratch

	// Emits the path like a[b][] and a[b][0][c].
	writePath := func() {
		if wrote {
			_ = dst.WriteByte('&')
		} else {
			wrote = true
		}
		// First segment (property name) does not require escaping by contract.
		dst.WriteString(path[0].name)
		// Subsequent segments.
		for i := 1; i < depth; i++ {
			_ = dst.WriteByte('[')
			seg := path[i]
			if seg.name != "" {
				dst.QueryEscapeString(seg.name)
			} else if seg.index != noIndex {
				n := strconv.AppendInt(buf[:0], int64(seg.index), 10)
				_, _ = dst.Write(n)
			} // else: []
			_ = dst.WriteByte(']')
		}
	}

	var walkValue func(v any)
	walkValue = func(v any) {
		switch v := v.(type) {
		case nil:
			writePath()
			_ = dst.WriteByte('=')
		case string:
			writePath()
			_ = dst.WriteByte('=')
			dst.QueryEscapeString(v)
		case int:
			writePath()
			_ = dst.WriteByte('=')
			n := strconv.AppendInt(buf[:0], int64(v), 10)
			dst.QueryEscape(n)
		case []any:
			for i := 0; i < len(v); i++ {
				if depth == len(path) {
					panic("stripe: maximum nesting exceeded")
				}
				segVal := seg{index: noIndex}
				if _, ok := v[i].(map[string]any); ok {
					segVal.index = i
				}
				path[depth] = segVal
				depth++
				walkValue(v[i])
				depth--
			}
		case map[string]any:
			for name, v := range v {
				if depth == len(path) {
					panic("stripe: maximum nesting exceeded")
				}
				path[depth] = seg{name: name}
				depth++
				walkValue(v)
				depth--
			}
		default:
			panic(fmt.Sprintf("stripe: unexpected type: %T", v))
		}
	}

	for name, value := range attributes {
		path[0] = seg{name: name}
		depth = 1
		walkValue(value)
	}
}
