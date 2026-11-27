// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package commerceready provides a customer connector for Commerce Ready.
// https://open2b.dev/commerce/customers
package commerceready

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"iter"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

const (
	maxAPIKeyBytes        = 4096    // connector policy; the provider specifies no key length
	customersPageMultiple = 100     // non-final pages contain a multiple of 100 customers
	maxResponseBytes      = 4 << 20 // connector policy, including successful responses
)

func init() {
	connectors.RegisterApplication(connectors.ApplicationSpec{
		Code:       "commerceready",
		Label:      "Commerce Ready",
		Categories: connectors.CategorySaaS,
		AsSource: &connectors.AsApplicationSource{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Import customers as users from Commerce Ready",
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsApplicationDestination{
			Targets:     connectors.TargetUser,
			HasSettings: true,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Create and update customers in Commerce Ready",
				Overview: destinationOverview,
			},
		},
		Terms: connectors.ApplicationTerms{User: "Customer", Users: "Customers", UserID: "Customer ID"},
		EndpointGroups: []connectors.EndpointGroup{{
			// https://open2b.dev/api: 1,000 calls per five minutes, shared by the shop.
			RateLimit: connectors.RateLimit{RequestsPerSecond: 1000.0 / 300, Burst: 1, MaxConcurrentRequests: 1},
			RetryPolicy: connectors.RetryPolicy{
				"429":         connectors.RetryAfterStrategy(),
				"500 502 504": connectors.ExponentialStrategy(connectors.NetFailure, time.Second),
				// https://open2b.dev/error-handling
				"503": connectors.HeaderStrategy(connectors.RateLimited, "X-Wait-Seconds", nil),
			},
		}},
	}, New)
}

var _ connectors.RecordUpserter = (*CommerceReady)(nil)

// CommerceReady imports, creates, and updates Commerce Ready customers.
type CommerceReady struct {
	env *connectors.ApplicationEnv
}

// New returns a Commerce Ready connector without making API requests.
func New(env *connectors.ApplicationEnv) (*CommerceReady, error) {
	return &CommerceReady{env: env}, nil
}

var emailPattern = regexp.MustCompile(`^[\w_\.\+\-\=\?\^\#]+\@(?:[a-zA-Z0-9\-]+\.)+\w+$`)

// billingAddressProperties defines fields exposed at the top level in Krenalis
// and nested under billingAddress in Commerce Ready requests and responses.
var billingAddressProperties = types.Object([]types.Property{
	{
		Name:        "companyName",
		Type:        types.String().WithMaxLength(100),
		DisplayName: "Company",
	},
	{
		Name:           "firstName",
		Type:           types.String().WithPattern(regexp.MustCompile(`(?s)^.{1,25}$`)),
		CreateRequired: true,
		DisplayName:    "First name",
	},
	{
		Name:           "lastName",
		Type:           types.String().WithPattern(regexp.MustCompile(`(?s)^.{1,25}$`)),
		CreateRequired: true,
		DisplayName:    "Last name",
	},
	{
		Name:           "street1",
		Type:           types.String().WithPattern(regexp.MustCompile(`(?s)^.{1,60}$`)),
		CreateRequired: true,
		DisplayName:    "Street",
	},
	{
		Name:        "street2",
		Type:        types.String().WithMaxLength(60),
		DisplayName: "Street, second line",
	},
	{
		Name:           "city",
		Type:           types.String().WithPattern(regexp.MustCompile(`(?s)^.{1,35}$`)),
		CreateRequired: true,
		DisplayName:    "City",
	},
	{
		Name:           "postalCode",
		Type:           types.String().WithPattern(regexp.MustCompile(`(?s)^.{1,20}$`)),
		CreateRequired: true,
		DisplayName:    "Postal code",
	},
	{
		Name:           "country",
		Type:           types.String().AsCountry(types.ISO3166Alpha2),
		CreateRequired: true,
		DisplayName:    "Country ISO code",
	},
	{
		Name:        "stateProv",
		Type:        types.String().WithMaxLength(3),
		DisplayName: "State or province",
	},
	{
		Name:        "personalCode",
		Type:        types.String().WithMaxLength(20),
		DisplayName: "Personal code",
	},
	{
		Name:        "companyCode",
		Type:        types.String().WithMaxLength(20),
		DisplayName: "VAT Number",
	},
	{
		Name:        "companyCode2",
		Type:        types.String().WithMaxLength(20),
		DisplayName: "REA",
	},
	{
		Name:        "phoneNumber",
		Type:        types.String().WithMaxLength(15),
		DisplayName: "Phone",
	},
	{
		Name:        "mobileNumber",
		Type:        types.String().WithMaxLength(15),
		DisplayName: "Mobile",
	},
	{
		Name:        "faxNumber",
		Type:        types.String().WithMaxLength(15),
		DisplayName: "Fax",
	},
	{
		Name:           "email",
		Type:           types.String().AsEmail().WithPattern(emailPattern), // TODO(marco): limit to 120 characters
		CreateRequired: true,
		DisplayName:    "Email address",
	},
}).Properties()

// RecordSchema returns the schema of the specified target and role.
func (cr *CommerceReady) RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {
	properties := billingAddressProperties.Slice()
	properties = append(properties,
		types.Property{
			Name:        "shippingAddress",
			Nullable:    true,
			DisplayName: "Shipping address",
			Type: types.Object([]types.Property{
				{
					Name:           "name",
					Type:           types.String().WithPattern(regexp.MustCompile(`(?s)^.{1,100}$`)),
					CreateRequired: true,
					DisplayName:    "Recipient name",
				},
				{
					Name:           "street1",
					Type:           types.String().WithPattern(regexp.MustCompile(`(?s)^.{1,60}$`)),
					CreateRequired: true,
					DisplayName:    "Street",
				},
				{
					Name:        "street2",
					Type:        types.String().WithMaxLength(60),
					DisplayName: "Street, second line",
				},
				{
					Name:           "city",
					Type:           types.String().WithPattern(regexp.MustCompile(`(?s)^.{1,35}$`)),
					CreateRequired: true,
					DisplayName:    "City",
				},
				{
					Name:           "postalCode",
					Type:           types.String().WithPattern(regexp.MustCompile(`(?s)^.{1,20}$`)),
					CreateRequired: true,
					DisplayName:    "Postal code",
				},
				{
					Name:           "country",
					Type:           types.String().AsCountry(types.ISO3166Alpha2),
					CreateRequired: true,
					DisplayName:    "Country ISO code",
				},
				{
					Name:        "stateProv",
					Type:        types.String().WithMaxLength(3),
					DisplayName: "State or province",
				},
			}),
		},
		types.Property{
			Name:        "code",
			Type:        types.String().WithMaxLength(32),
			DisplayName: "Code",
			Description: "Customer code",
		},
		types.Property{
			Name:        "group",
			Type:        types.Int(32).WithIntRange(1, 255),
			DisplayName: "Customer group ID",
		},
		types.Property{
			Name:        "collection",
			Type:        types.Int(32).WithIntRange(1, math.MaxInt32),
			Nullable:    true,
			DisplayName: "Catalog collection ID",
		},
		types.Property{
			Name:        "collections",
			Type:        types.Array(types.Int(32).WithIntRange(1, math.MaxInt32)).WithMaxElements(100),
			DisplayName: "Additional collection IDs",
		},
		types.Property{
			Name:        "invoiceRecipient",
			Type:        types.String().WithMaxLength(256),
			DisplayName: "Invoice recipient code or PEC email",
		},
		types.Property{
			Name:        "canLogin",
			Type:        types.Boolean(),
			DisplayName: "Customer can log in",
		},
		types.Property{
			Name:        "locale",
			Type:        types.String().WithPattern(regexp.MustCompile(`(?s)^.{2,5}$`)),
			DisplayName: "Locale",
		},
	)
	if role == connectors.Source {
		properties = append(properties,
			types.Property{
				Name:        "hasPassword",
				Type:        types.Boolean(),
				DisplayName: "Has a password",
			},
			types.Property{
				Name:        "creationTime",
				Type:        types.DateTime(),
				DisplayName: "Creation time",
			},
			types.Property{
				Name:        "loginTime",
				Type:        types.DateTime(),
				Nullable:    true,
				DisplayName: "Last login time",
			},
		)
	}
	return types.AsRole(types.Object(properties), types.Role(role)), nil
}

// Records returns the records of the specified target.
func (cr *CommerceReady) Records(ctx context.Context, target connectors.Targets, updatedAt time.Time, cursor string, schema types.Type) ([]connectors.Record, string, error) {

	var after int64
	if cursor != "" {
		var err error
		after, err = strconv.ParseInt(cursor, 10, 32)
		if err != nil {
			return nil, "", errors.New("connectors/commerceready: invalid customer cursor")
		}
		if after <= 0 {
			return nil, "", errors.New("connectors/commerceready: invalid customer cursor")
		}
	}
	var s innerSettings
	err := cr.env.Settings.Load(ctx, &s)
	if err != nil {
		return nil, "", err
	}
	location, err := time.LoadLocation(s.TimeZone)
	if err != nil {
		return nil, "", errors.New("Commerce Ready shop time zone is invalid")
	}
	fields := []string{"id", "updateTime"}
	var hasBillingAddress bool
	for _, p := range schema.Properties().All() {
		if billingAddressProperties.ContainsName(p.Name) {
			if !hasBillingAddress {
				fields = append(fields, "billingAddress")
				hasBillingAddress = true
			}
			continue
		}
		fields = append(fields, p.Name)
	}

	for {
		var records []connectors.Record
		count := 0
		payload := map[string]any{
			"conditions": map[string]any{"after": after},
			"fields":     fields,
			"order":      []string{"id"},
		}
		for item, err := range cr.find(ctx, s, payload) {
			if err != nil {
				return nil, "", err
			}
			count++
			if int64(item.ID) <= after {
				return nil, "", errors.New("Commerce Ready returned customer IDs out of order")
			}
			after = int64(item.ID)
			modified, err := time.ParseInLocation(time.DateTime, item.UpdateTime, location)
			if err != nil {
				return nil, "", errors.New("Commerce Ready returned an invalid customer update time")
			}
			if modified.Year() < 1 || modified.Year() > 9999 {
				return nil, "", errors.New("Commerce Ready returned an invalid customer update time")
			}
			if modified.Before(updatedAt) {
				continue
			}
			attributes, err := item.attributes(schema, location)
			if err != nil {
				return nil, "", err
			}
			records = append(records, connectors.Record{
				ID:         strconv.FormatInt(int64(item.ID), 10),
				UpdatedAt:  modified,
				Attributes: attributes,
			})
		}
		if count == 0 || count%customersPageMultiple != 0 {
			return records, "", io.EOF
		}
		if len(records) > 0 {
			return records, strconv.FormatInt(after, 10), nil
		}
		// Continue through filtered pages: the runtime rejects empty non-final pages.
	}

}

// timeZoneNames contains the public-domain IANA zone names from Go 1.27.3's
// lib/time/zoneinfo.zip.
//
//go:embed timezones.txt
var timeZoneNames string

// ServeUI serves the connector's user interface.
func (cr *CommerceReady) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {
	switch event {
	case "load":
		var s innerSettings
		err := cr.env.Settings.Load(ctx, &s)
		if err != nil {
			return nil, err
		}
		// The Admin submits untouched settings unchanged and adds apiKey only on
		// input. Omission preserves a saved key; an explicit empty value is invalid.
		settings, err = json.Marshal(map[string]any{"apiURL": s.APIURL, "timeZone": s.TimeZone})
		if err != nil {
			return nil, err
		}
		keyHelp := "Secret key with customers.find permission."
		if role == connectors.Destination {
			keyHelp = "Secret key with customers.find, customers.create, and customers.update permissions."
		}
		if s.APIKey != "" {
			keyHelp = "A key is saved. Leave this field untouched to keep it, or enter a replacement."
		}
		names := strings.Fields(timeZoneNames)
		timeZones := make([]connectors.Option, len(names))
		for i, name := range names {
			timeZones[i] = connectors.Option{Text: name, Value: name}
		}
		return &connectors.UI{
			Settings: settings,
			Fields: []connectors.Component{
				&connectors.Input{
					Name:        "apiURL",
					Type:        "url",
					Label:       "API URL",
					Placeholder: "https://shop.example.com/api/v13/",
					HelpText:    "Base URL of the API, without a method name.",
					MaxLength:   2048,
				},
				&connectors.Input{
					Name:      "apiKey",
					Type:      "password",
					Label:     "Secret API key",
					HelpText:  keyHelp,
					MaxLength: maxAPIKeyBytes,
				},
				&connectors.Select{
					Name:        "timeZone",
					Label:       "Shop time zone",
					Placeholder: "Select a time zone",
					HelpText:    "Time zone used by the shop's API timestamps.",
					Options:     timeZones,
				},
			},
			Buttons: []connectors.Button{connectors.SaveButton},
		}, nil
	case "save":
		return nil, cr.saveSettings(ctx, settings)
	default:
		return nil, connectors.ErrUIEventNotExist
	}
}

// Upsert updates or creates records in the API for the specified target.
func (cr *CommerceReady) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records, schema types.Type) error {

	var s innerSettings
	err := cr.env.Settings.Load(ctx, &s)
	if err != nil {
		return err
	}

	record := records.First()

	var payload map[string]any
	{
		customer := make(map[string]any, len(record.Attributes))
		var billingAddress map[string]any
		for name, value := range record.Attributes {
			if billingAddressProperties.ContainsName(name) {
				if billingAddress == nil {
					billingAddress = map[string]any{name: value}
				} else {
					billingAddress[name] = value
				}
				continue
			}
			customer[name] = value
		}
		if billingAddress != nil {
			customer["billingAddress"] = billingAddress
		}
		payload = map[string]any{"customer": customer}
	}

	operation := "create"
	if record.IsUpdate() {
		operation = "update"
		id, err := strconv.ParseInt(record.ID, 10, 32)
		if err != nil || id <= 0 {
			return connectors.RecordsError{0: errors.New("Commerce Ready customer ID is not a positive int32")}
		}
		payload["id"] = id
	}

	response, err := cr.call(ctx, s, operation, payload)
	if err != nil {
		if apiErr, ok := errors.AsType[*apiError](err); ok {
			switch apiErr.Type {
			case "UnknownField", "Missing", "MissingCombination", "Malformed", "InvalidValue",
				"InvalidCombination", "AlreadyExists", "NotFound":
				return connectors.RecordsError{0: apiErr}
			default:
				// Permission and unknown failures apply to the request.
			}
		}
		return err
	}
	if record.IsCreate() && response.ID <= 0 {
		return errors.New("Commerce Ready did not return the created customer ID")
	}

	return nil
}

// call executes a write operation and never exposes provider error text.
func (cr *CommerceReady) call(ctx context.Context, settings innerSettings, operation string, payload any) (*apiResponse, error) {
	res, err := cr.request(ctx, settings, operation, payload)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	limited := &io.LimitedReader{R: res.Body, N: maxResponseBytes + 1}
	var response apiResponse
	err = json.Decode(limited, &response)
	if limited.N == 0 {
		return nil, errors.New("Commerce Ready response exceeds the connector size limit")
	}
	if err != nil {
		return nil, errors.New("Commerce Ready returned an invalid API response")
	}
	err = response.err()
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// find reads one page of customers lazily and returns each customer by value.
func (cr *CommerceReady) find(ctx context.Context, settings innerSettings, payload any) iter.Seq2[customer, error] {
	return func(yield func(customer, error) bool) {

		res, err := cr.request(ctx, settings, "find", payload)
		if err != nil {
			yield(customer{}, err)
			return
		}
		defer res.Body.Close()
		limited := &io.LimitedReader{R: res.Body, N: maxResponseBytes + 1}
		dec := json.NewDecoder(limited)
		fail := func(err error) {
			if limited.N == 0 {
				err = errors.New("Commerce Ready response exceeds the connector size limit")
			}
			yield(customer{}, err)
		}

		var response apiResponse
		if dec.PeekKind() != json.Object {
			fail(errors.New("Commerce Ready returned an invalid API response"))
			return
		}
		err = dec.SkipToken()
		if err != nil {
			fail(errors.New("Commerce Ready returned an invalid API response"))
			return
		}
		var item customer
		var hasCustomers bool
		for dec.PeekKind() != '}' {
			key, err := dec.ReadToken()
			if err != nil {
				fail(errors.New("Commerce Ready returned an invalid API response"))
				return
			}
			switch key.String() {
			case "status":
				err = dec.Decode(&response.Status)
			case "error":
				err = dec.Decode(&response.Error)
			case "id":
				err = dec.Decode(&response.ID)
			case "customers":
				if dec.PeekKind() != json.Array {
					fail(errors.New("Commerce Ready returned an invalid customer page"))
					return
				}
				err = dec.SkipToken()
				if err != nil {
					fail(errors.New("Commerce Ready returned an invalid customer page"))
					return
				}
				hasCustomers = true
				for dec.PeekKind() != ']' {
					// Reset nullable and omitted fields before reusing the customer.
					item = customer{}
					err = dec.Decode(&item)
					if err != nil {
						fail(errors.New("Commerce Ready returned invalid customer data"))
						return
					}
					if !yield(item, nil) {
						return
					}
				}
				err = dec.SkipToken()
			default:
				err = dec.SkipValue()
			}
			if err != nil {
				fail(errors.New("Commerce Ready returned an invalid API response"))
				return
			}
		}
		err = dec.SkipToken()
		if err != nil {
			fail(errors.New("Commerce Ready returned an invalid API response"))
			return
		}
		_, err = dec.ReadToken()
		if err != nil {
			if err != io.EOF {
				fail(errors.New("Commerce Ready returned an invalid API response"))
				return
			}
		}
		if err == nil {
			fail(errors.New("Commerce Ready returned an invalid API response"))
			return
		}
		if limited.N == 0 {
			yield(customer{}, errors.New("Commerce Ready response exceeds the connector size limit"))
			return
		}
		err = response.err()
		if err != nil {
			fail(err)
			return
		}
		if !hasCustomers {
			fail(errors.New("Commerce Ready returned an invalid customer page"))
		}

	}
}

// request executes an API request; the caller must close the response body.
func (cr *CommerceReady) request(ctx context.Context, settings innerSettings, operation string, payload any) (*http.Response, error) {
	bb := cr.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding)
	defer bb.Close()
	err := bb.Encode(payload)
	if err != nil {
		return nil, errors.New("connectors/commerceready: cannot encode API request")
	}
	u := settings.APIURL + "commerce.customers." + operation
	req, err := bb.NewRequest(ctx, http.MethodPost, u)
	if err != nil {
		return nil, errors.New("Commerce Ready API URL is invalid")
	}
	if operation == "find" {
		// find is read-only despite using POST; only mark reads as replayable.
		req.Header["Idempotency-Key"] = nil
	}
	req.Header.Set("X-Secret-Key", settings.APIKey)
	res, err := cr.env.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("Commerce Ready returned HTTP %d", res.StatusCode)
	}
	return res, nil
}

// saveSettings validates, normalizes, and stores the connector settings.
func (cr *CommerceReady) saveSettings(ctx context.Context, settings json.Value) error {

	var submitted struct {
		APIURL   string     `json:"apiURL"`
		APIKey   json.Value `json:"apiKey"`
		TimeZone string     `json:"timeZone"`
	}
	err := settings.Unmarshal(&submitted)
	if err != nil {
		return connectors.NewInvalidSettingsError("Commerce Ready settings is not a valid object")
	}
	s := innerSettings{
		APIURL:   submitted.APIURL,
		TimeZone: submitted.TimeZone,
	}
	if submitted.APIKey == nil {
		var previous innerSettings
		err = cr.env.Settings.Load(ctx, &previous)
		if err != nil {
			return err
		}
		s.APIKey = previous.APIKey
	} else {
		if !submitted.APIKey.IsString() {
			return connectors.NewInvalidSettingsError("Secret API key is not a string")
		}
		err = submitted.APIKey.Unmarshal(&s.APIKey)
		if err != nil {
			return connectors.NewInvalidSettingsError("Secret API key is not a string")
		}
	}

	// Validate the destination before storing a credential for it.
	s.APIURL = strings.TrimRight(s.APIURL, "/") + "/"
	if len(s.APIURL) > 2048 {
		return connectors.NewInvalidSettingsError("API URL exceeds 2048 bytes")
	}
	u, err := url.Parse(s.APIURL)
	if err != nil {
		return connectors.NewInvalidSettingsError("API URL is not a valid HTTPS URL")
	}
	if u.Scheme != "https" || u.Hostname() == "" || u.User != nil ||
		u.RawQuery != "" || u.ForceQuery || strings.Contains(s.APIURL, "#") {
		return connectors.NewInvalidSettingsError("API URL is invalid or contains credentials, query or fragment")
	}

	// Validate the key as a bounded HTTP header value.
	if s.APIKey == "" || len(s.APIKey) > maxAPIKeyBytes {
		return connectors.NewInvalidSettingsError("Secret API key is empty or exceeds 4096 bytes")
	}
	for i := range len(s.APIKey) {
		if s.APIKey[i] <= ' ' || s.APIKey[i] > '~' {
			return connectors.NewInvalidSettingsError("Secret API key contains spaces, control or non-ASCII characters")
		}
	}

	// Validate the time zone.
	if s.TimeZone == "" || len(s.TimeZone) > 100 || s.TimeZone == "Local" {
		return connectors.NewInvalidSettingsError("Shop time zone is not an explicit IANA time zone")
	}
	_, err = time.LoadLocation(s.TimeZone)
	if err != nil {
		return connectors.NewInvalidSettingsError("Shop time zone is not a valid IANA time zone")
	}

	return cr.env.Settings.Store(ctx, s)
}

type apiError struct {
	Type string `json:"type"`
}

// Error returns a sanitized description of a provider failure.
func (err *apiError) Error() string {
	switch err.Type {
	case "AccessDenied":
		return "Commerce Ready denied access to an API field"
	case "AlreadyExists":
		return "Commerce Ready customer code or email already exists"
	case "NotFound":
		return "Commerce Ready customer or referenced resource does not exist"
	case "Missing", "MissingCombination":
		return "Commerce Ready requires additional customer fields"
	case "Malformed", "InvalidValue", "InvalidCombination", "UnknownField":
		return "Commerce Ready rejected invalid customer fields"
	default:
		return "Commerce Ready rejected the API request"
	}
}

type apiResponse struct {
	Status string    `json:"status"`
	Error  *apiError `json:"error"`
	ID     int32     `json:"id"`
}

func (response *apiResponse) err() error {
	switch response.Status {
	case "ok":
		return nil
	case "error":
		if response.Error == nil {
			return errors.New("Commerce Ready returned an invalid API error")
		}
		return response.Error
	default:
		return errors.New("Commerce Ready returned an invalid API status")
	}
}

type innerSettings struct {
	APIURL   string `json:"apiURL"`
	APIKey   string `json:"apiKey"`
	TimeZone string `json:"timeZone"`
}

type customer struct {
	ID               int32            `json:"id"`
	Code             string           `json:"code"`
	CanLogin         bool             `json:"canLogin"`
	HasPassword      bool             `json:"hasPassword"`
	Locale           string           `json:"locale"`
	CreationTime     string           `json:"creationTime"`
	LoginTime        *string          `json:"loginTime"`
	UpdateTime       string           `json:"updateTime"`
	Group            int32            `json:"group"`
	Collection       *int32           `json:"collection"`
	Collections      []int32          `json:"collections"`
	InvoiceRecipient string           `json:"invoiceRecipient"`
	BillingAddress   billingAddress   `json:"billingAddress"`
	ShippingAddress  *shippingAddress `json:"shippingAddress"`
}

// attributes converts the requested customer fields to record attributes.
func (c *customer) attributes(schema types.Type, location *time.Location) (map[string]any, error) {
	attributes := make(map[string]any, schema.Properties().Len())
	for _, property := range schema.Properties().All() {
		var value any
		var err error
		switch property.Name {
		case "code":
			value = c.Code
		case "canLogin":
			value = c.CanLogin
		case "hasPassword":
			value = c.HasPassword
		case "locale":
			value = c.Locale
		case "group":
			value = int(c.Group)
		case "collection":
			if c.Collection != nil {
				value = int(*c.Collection)
			}
		case "collections":
			values := make([]any, len(c.Collections))
			for i, n := range c.Collections {
				values[i] = int(n)
			}
			value = values
		case "invoiceRecipient":
			value = c.InvoiceRecipient
		case "creationTime":
			value, err = time.ParseInLocation(time.DateTime, c.CreationTime, location)
		case "loginTime":
			if c.LoginTime != nil {
				value, err = time.ParseInLocation(time.DateTime, *c.LoginTime, location)
			}
		case "shippingAddress":
			if s := c.ShippingAddress; s != nil {
				value = map[string]any{
					"name": s.Name, "street1": s.Street1, "street2": s.Street2, "city": s.City,
					"postalCode": s.PostalCode, "stateProv": s.StateProv, "country": s.Country,
				}
			}
		case "firstName":
			value = c.BillingAddress.FirstName
		case "lastName":
			value = c.BillingAddress.LastName
		case "personalCode":
			value = c.BillingAddress.PersonalCode
		case "companyName":
			value = c.BillingAddress.CompanyName
		case "companyCode":
			value = c.BillingAddress.CompanyCode
		case "companyCode2":
			value = c.BillingAddress.CompanyCode2
		case "street1":
			value = c.BillingAddress.Street1
		case "street2":
			value = c.BillingAddress.Street2
		case "city":
			value = c.BillingAddress.City
		case "postalCode":
			value = c.BillingAddress.PostalCode
		case "stateProv":
			value = c.BillingAddress.StateProv
		case "country":
			value = c.BillingAddress.Country
		case "phoneNumber":
			value = c.BillingAddress.PhoneNumber
		case "mobileNumber":
			value = c.BillingAddress.MobileNumber
		case "faxNumber":
			value = c.BillingAddress.FaxNumber
		case "email":
			value = c.BillingAddress.Email
		default:
			return nil, errors.New("connectors/commerceready: unsupported customer property")
		}
		if err != nil {
			return nil, fmt.Errorf("Commerce Ready returned invalid field %s", connectors.QuoteErrorTerm(property.Name))
		}
		attributes[property.Name] = value
	}
	return attributes, nil
}

type billingAddress struct {
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	PersonalCode string `json:"personalCode"`
	CompanyName  string `json:"companyName"`
	CompanyCode  string `json:"companyCode"`
	CompanyCode2 string `json:"companyCode2"`
	Street1      string `json:"street1"`
	Street2      string `json:"street2"`
	City         string `json:"city"`
	PostalCode   string `json:"postalCode"`
	StateProv    string `json:"stateProv"`
	Country      string `json:"country"`
	PhoneNumber  string `json:"phoneNumber"`
	MobileNumber string `json:"mobileNumber"`
	FaxNumber    string `json:"faxNumber"`
	Email        string `json:"email"`
}

type shippingAddress struct {
	Name       string `json:"name"`
	Street1    string `json:"street1"`
	Street2    string `json:"street2"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode"`
	StateProv  string `json:"stateProv"`
	Country    string `json:"country"`
}
