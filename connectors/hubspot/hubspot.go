// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package hubspot provides a connector for HubSpot.
// (https://developers.hubspot.com/docs/api-reference/overview)
//
// HubSpot is a trademark of HubSpot, Inc.
// This connector is not affiliated with or endorsed by HubSpot, Inc.
package hubspot

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

//go:embed documentation/source/overview.md
var sourceOverview string

//go:embed documentation/destination/overview.md
var destinationOverview string

// TODO(Gianluca): Groups are partially supported by this connector. When they
// are fully supported by both the connector and Meergo, re-enable the
// descriptions that refer to the groups and add the target "Group" where
// needed.

func init() {
	connectors.RegisterAPI(connectors.APISpec{
		Code:       "hubspot",
		Label:      "HubSpot",
		Categories: connectors.CategorySaaS,
		AsSource: &connectors.AsAPISource{
			Targets: connectors.TargetUser,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Import contacts as users from HubSpot",
				Overview: sourceOverview,
			},
		},
		AsDestination: &connectors.AsAPIDestination{
			Targets: connectors.TargetUser,
			Documentation: connectors.RoleDocumentation{
				Summary:  "Export users as contacts to HubSpot",
				Overview: destinationOverview,
			},
		},
		Terms: connectors.APITerms{
			User:  "contact",
			Users: "contacts",
		},
		IdentityIDLabel: "HubSpot ID",
		OAuth: connectors.OAuth{
			AuthURL:           "https://app-eu1.hubspot.com/oauth/authorize",
			TokenURL:          "https://api.hubapi.com/oauth/v1/token",
			SourceScopes:      []string{"oauth", "crm.objects.contacts.read", "crm.schemas.contacts.read"},
			DestinationScopes: []string{"oauth", "crm.objects.contacts.read", "crm.objects.contacts.write", "crm.schemas.contacts.read"},
			Disallow127_0_0_1: true,
		},
		EndpointGroups: []connectors.EndpointGroup{{
			RequireOAuth: true,
			// https://developers.hubspot.com/docs/developer-tooling/platform/usage-guidelines
			RateLimit: connectors.RateLimit{RequestsPerSecond: 11, Burst: 110},
			// https://developers.hubspot.com/docs/api-reference/error-handling
			RetryPolicy: connectors.RetryPolicy{
				"429":                         connectors.HeaderStrategy(connectors.RateLimited, "X-HubSpot-RateLimit-Interval-Milliseconds", parseMilliseconds),
				"477":                         connectors.RetryAfterStrategy(),
				"500 502 503 504 521 523 524": connectors.ExponentialStrategy(connectors.NetFailure, time.Second),
			},
		}},
	}, New)
}

// New returns a new connector instance for HubSpot.
func New(env *connectors.APIEnv) (*HubSpot, error) {
	c := HubSpot{env: env}
	return &c, nil
}

type HubSpot struct {
	env *connectors.APIEnv
}

// OAuthAccount returns the API's account associated with the OAuth
// authorization.
func (hs *HubSpot) OAuthAccount(ctx context.Context) (string, error) {
	var res struct {
		PortalId int `json:"portalId"`
	}
	err := hs.call(ctx, "GET", "/account-info/v3/details", nil, &res)
	if err != nil {
		return "", err
	}
	if res.PortalId <= 0 {
		return "", fmt.Errorf("HubSpot has returned an invalid account (portalId): %d", res.PortalId)
	}
	return strconv.Itoa(res.PortalId), nil
}

var propertyGroups = []struct {
	Name        string
	HSName      string
	Description string
}{
	{"contact", "contactinformation", "Contact Information"},
	{"emails", "emailinformation", "Email Information"},
	{"analytics", "analyticsinformation", "Web Analytics History"},
	{"activity", "contact_activity", "Contact activity"},
	{"contact_lifecycle", "contactlcs", "Contact Lifecycle Stage Properties"},
	{"facebook", "facebook_ads_properties", "Facebook Ads Properties"},
	{"conversion", "conversioninformation", "Conversion Onformation"},
	{"deals", "deal_information", "Deal Information"},
	{"sales", "sales_properties", "Sales Properties"},
	{"orders", "order_information", "Order Information"},
	{"cripted", "contactscripted", "Calculated Contact Information"},
	{"social", "socialmediainformation", "Social Media Information"},
	{"sms", "smsinformation", "SMS Information"},
	{"multi_account", "multiaccountmanagement", "Multi Account Management"},
}

// RecordSchema returns the schema of the specified target and role.
func (hs *HubSpot) RecordSchema(ctx context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {

	var response struct {
		Results []struct {
			Hidden      bool   `json:"hidden"`
			Name        string `json:"name"`
			GroupHSName string `json:"groupName"`
			Options     []struct {
				Label  string `json:"label"`
				Value  string `json:"value"`
				Hidden bool   `json:"hidden"`
			} `json:"options"`
			Label                string `json:"label"`
			Description          string `json:"description"`
			Type                 string `json:"type"`
			ModificationMetadata struct {
				ReadOnlyValue bool `json:"readOnlyValue"`
			} `json:"modificationMetadata"`
		} `json:"results"`
	}
	err := hs.call(ctx, "GET", "/crm/v3/properties/contact", nil, &response)
	if err != nil {
		return types.Type{}, err
	}

	groups := map[string][]types.Property{}

	for _, r := range response.Results {
		typ := propertyType(r.Type)
		if !typ.Valid() {
			continue
		}
		if role == connectors.Destination && r.ModificationMetadata.ReadOnlyValue {
			continue
		}
		property := types.Property{
			Name:        r.Name,
			Type:        typ,
			Nullable:    true,
			Description: r.Label,
		}
		if typ.Kind() == types.TextKind {
			if len(r.Options) == 0 {
				property.Type.WithCharLen(65536)
			} else {
				var n int
				for _, option := range r.Options {
					if !option.Hidden {
						n++
					}
				}
				if n == 0 {
					continue // all options are hidden, skip the property
				}
				values := make([]string, 0, n)
				for _, option := range r.Options {
					if option.Hidden {
						continue
					}
					values = append(values, option.Value)
				}
				property.Type = typ.WithValues(values...)
			}
		}
		if properties, ok := groups[r.GroupHSName]; ok {
			groups[r.GroupHSName] = append(properties, property)
		} else {
			groups[r.GroupHSName] = []types.Property{property}
		}
	}

	// Group properties.
	properties := make([]types.Property, 0, len(groups))
	for _, group := range propertyGroups {
		pp, ok := groups[group.HSName]
		if !ok {
			continue
		}
		slices.SortStableFunc(pp, func(a, b types.Property) int {
			hsa := strings.HasPrefix(a.Name, "hs_")
			hsb := strings.HasPrefix(b.Name, "hs_")
			switch {
			case hsa == hsb:
				return 0
			case hsa:
				return 1
			default:
				return -1
			}
		})
		properties = append(properties, types.Property{
			Name:        group.Name,
			Type:        types.Object(pp),
			Description: group.Description,
		})
		delete(groups, group.HSName)
	}
	if len(groups) > 0 {
		names := make([]string, 0, len(groups))
		for name := range groups {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			properties = append(properties, types.Property{
				Name:         name,
				Type:         types.Object(groups[name]),
				ReadOptional: true,
			})
		}
	}

	schema, err := types.ObjectOf(properties)
	if err != nil {
		return types.Type{}, fmt.Errorf("cannot create schema from properties: %s", err)
	}

	return schema, nil
}

// Records returns the records of the specified target.
func (hs *HubSpot) Records(ctx context.Context, target connectors.Targets, lastChangeTime time.Time, ids []string, cursor string, schema types.Type) ([]connectors.Record, string, error) {

	path := "/crm/v3/objects/contacts/"

	var response struct {
		Results []struct {
			ID         string         `json:"id"`
			Properties map[string]any `json:"properties"`
			UpdatedAt  string         `json:"updatedAt"`
		} `json:"results"`
		Paging struct {
			Next struct {
				After string `json:"after"`
			} `json:"next"`
		} `json:"paging"`
	}

	bb := hs.env.HTTPClient.GetBodyBuffer(connectors.NoEncoding) // It also supports Gzip.
	defer bb.Close()

	bb.WriteByte('{')

	if ids != nil {
		bb.WriteString(`"inputs":[`)
		for i, id := range ids {
			if i > 0 {
				bb.WriteByte(',')
			}
			bb.WriteString(`{"id":"`)
			bb.WriteString(id)
			bb.WriteString(`"}`)
		}
		bb.WriteString(`],`)
		path += "batch/read"
	} else {
		propertyName := "lastmodifieddate"
		unix := lastChangeTime.UnixMilli()
		if unix < 0 {
			unix = 0
		}
		bb.WriteString(`"filterGroups":[{"filters":[{"value":"`)
		bb.WriteString(strconv.FormatInt(unix, 10))
		bb.WriteString(`","propertyName":"` + propertyName + `","operator":"GTE"}` +
			`]}],"sorts":["` + propertyName + `"],`)
		path += "search"
	}

	bb.WriteString(`"after":"`)
	bb.WriteString(cursor)
	bb.WriteString(`","limit":100,"properties":[`)
	i := 0
	for _, group := range schema.Properties().All() {
		for _, p := range group.Type.Properties().All() {
			if i > 0 {
				bb.WriteByte(',')
			}
			bb.WriteByte('"')
			bb.WriteString(p.Name)
			bb.WriteByte('"')
			i++
		}
	}
	bb.WriteString(`]}`)

	err := hs.call(ctx, "POST", path, bb, &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Results) == 0 {
		return nil, "", io.EOF
	}

	records := make([]connectors.Record, len(response.Results))
	for i, result := range response.Results {
		records[i] = connectors.Record{
			ID:         result.ID,
			Attributes: map[string]any{},
		}
		updatedAt, err := time.Parse(time.RFC3339, result.UpdatedAt)
		if err != nil {
			records[i].Err = fmt.Errorf("HubSpot has returned an invalid value for updatedAt: %q", updatedAt)
			continue
		}
		// Group properties.
		for _, group := range schema.Properties().All() {
			properties := group.Type.Properties()
			pp := make(map[string]any, properties.Len())
			for _, p := range properties.All() {
				if v, ok := result.Properties[p.Name]; ok {
					pp[p.Name] = v
				}
			}
			records[i].Attributes[group.Name] = pp
		}
		records[i].LastChangeTime = updatedAt.UTC()
	}

	cursor = response.Paging.Next.After
	if cursor == "" {
		err = io.EOF
	}

	return records, cursor, err
}

// Upsert updates or creates records in the API for the specified target.
func (hs *HubSpot) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records) error {

	// Note that records.All() cannot be used because the HubSpot API's "upsert" method does not allow updating contacts using "hs_object_id".
	// See https://community.hubspot.com/t5/APIs-Integrations/Create-or-update-a-batch-of-contacts-by-unique-property-values/m-p/1047925.

	method := "update"
	if first, _ := records.Peek(); first.ID == "" {
		method = "create"
	}

	bb := hs.env.HTTPClient.GetBodyBuffer(connectors.Gzip) // It also supports NoEncoding.
	defer bb.Close()

	bb.WriteString(`{"inputs":[`)

	n := 0
	for record := range records.Same() {
		if n > 0 {
			bb.WriteByte(',')
		}
		bb.WriteByte('{')
		if method == "update" {
			_ = bb.EncodeKeyValue("id", record.ID)
			_ = bb.WriteByte(',')
		}
		bb.WriteString(`"properties":{`)
		for _, properties := range record.Attributes {
			for name, value := range properties.(map[string]any) {
				_ = bb.EncodeKeyValue(name, value)
			}
		}
		bb.WriteString("}}")
		if err := bb.Flush(); err != nil {
			return err
		}
		if err := bb.Flush(); err != nil {
			return err
		}
		n++
		if n == 100 {
			break
		}
	}

	bb.WriteString(`]}`)

	return hs.call(ctx, "POST", "/crm/v3/objects/contacts/batch/"+method, bb, nil)
}

func (hs *HubSpot) call(ctx context.Context, method, path string, bb *connectors.BodyBuffer, response any) error {
	req, err := bb.NewRequest(ctx, method, "https://api.hubapi.com/"+path[1:])
	if err != nil {
		return err
	}
	res, err := hs.env.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()
	switch res.StatusCode {
	case 200, 201, 207:
	default:
		hsErr := &hubspotError{statusCode: res.StatusCode}
		err := json.Decode(res.Body, hsErr)
		if err != nil {
			return err
		}
		return hsErr
	}
	if response != nil {
		return json.Decode(res.Body, response)
	}
	return nil
}

type hubspotError struct {
	statusCode int
	Status     string `json:"status"`
	Message    string `json:"message"`
	Errors     []struct {
		Message string `json:"message"`
		In      string `json:"in"`
	} `json:"errors"`
	Category      string `json:"category"`
	CorrelationId string `json:"correlationId"`
}

func (err *hubspotError) Error() string {
	return fmt.Sprintf("unexpected error from HubSpot: (%d) %s", err.statusCode, err.Message)
}

// parseMilliseconds parses the value of the
// "X-HubSpot-RateLimit-Interval-Milliseconds" response header.
func parseMilliseconds(s string) (time.Duration, error) {
	d, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(d) * time.Millisecond, nil
}

// propertyType returns the type of the HubSpot property with type t.
// If the property type is not supported, it returns an invalid type.
// (https://developers.hubspot.com/docs/api-reference/crm-properties-v3/guide).
func propertyType(t string) types.Type {
	switch t {
	case "bool":
		return types.Boolean()
	case "date":
		return types.Date()
	case "datetime":
		return types.DateTime()
	case "enumeration":
		return types.Text()
	case "number":
		// HubSpot has no limitations on precision and scale.
		return types.Decimal(types.MaxDecimalPrecision, types.MaxDecimalScale)
	case "object_coordinates", "json":
		// These types are for internal use and are not visible in HubSpot.
		return types.Type{}
	case "string", "phone_number":
		return types.Text()
	}
	return types.Type{}
}
