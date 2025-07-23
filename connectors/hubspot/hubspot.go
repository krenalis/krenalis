//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package hubspot implements the HubSpot connector.
// (https://developers.hubspot.com/docs/guides/api/crm/understanding-the-crm)
package hubspot

import (
	"context"
	_ "embed"
	"fmt"
	"io"
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

// TODO(Gianluca): Groups are partially supported by this connector. When they
// are fully supported by both the connector and Meergo, re-enable the
// descriptions that refer to the groups and add the target "Group" where
// needed.

func init() {
	meergo.RegisterApp(meergo.AppInfo{
		Name:       "HubSpot",
		Categories: meergo.CategoryCRM,
		AsSource: &meergo.AsAppSource{
			Targets: meergo.TargetUser,
			Documentation: meergo.ConnectorRoleDocumentation{
				Summary:  "Import contacts as users from HubSpot",
				Overview: sourceOverview,
			},
		},
		AsDestination: &meergo.AsAppDestination{
			Targets: meergo.TargetUser,
			Documentation: meergo.ConnectorRoleDocumentation{
				Summary:  "Export users as contacts to HubSpot",
				Overview: destinationOverview,
			},
		},
		Terms: meergo.AppTerms{
			User:  "contact",
			Users: "contacts",
		},
		IdentityIDLabel: "HubSpot ID",
		OAuth: meergo.OAuth{
			AuthURL:           "https://app-eu1.hubspot.com/oauth/authorize",
			TokenURL:          "https://api.hubapi.com/oauth/v1/token",
			SourceScopes:      []string{"crm.objects.contacts.read", "crm.schemas.contacts.read"},
			DestinationScopes: []string{"crm.objects.contacts.read", "crm.objects.contacts.write", "crm.schemas.contacts.read"},
		},
		EndpointGroups: []meergo.EndpointGroup{{
			// https://developers.hubspot.com/docs/guides/apps/api-usage/usage-details#public-apps
			RateLimit: meergo.RateLimit{RequestsPerSecond: 11, Burst: 110},
			// https://developers.hubspot.com/docs/reference/api/other-resources/error-handling
			RetryPolicy: meergo.RetryPolicy{
				"429":                         meergo.HeaderStrategy(meergo.RateLimited, "X-HubSpot-RateLimit-Interval-Milliseconds", parseMilliseconds),
				"477":                         meergo.RetryAfterStrategy(),
				"500 502 503 504 521 523 524": meergo.ExponentialStrategy(meergo.NetFailure, time.Second),
			},
		}},
		Icon: icon,
	}, New)
}

// New returns a new HubSpot connector instance.
func New(conf *meergo.AppConfig) (*HubSpot, error) {
	c := HubSpot{
		httpClient: conf.HTTPClient,
	}
	return &c, nil
}

type HubSpot struct {
	httpClient meergo.HTTPClient
}

// OAuthAccount returns the app's account associated with the OAuth
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
		return "", fmt.Errorf("connector HubSpot has returned an invalid account (portalId): %d", res.PortalId)
	}
	return strconv.Itoa(res.PortalId), nil
}

// RecordSchema returns the schema of the specified target and role.
func (hs *HubSpot) RecordSchema(ctx context.Context, target meergo.Targets, role meergo.Role) (types.Type, error) {

	var response struct {
		Results []struct {
			Hidden  bool   `json:"hidden"`
			Name    string `json:"name"`
			Options []struct {
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

	properties := make([]types.Property, 0, len(response.Results))
	for _, r := range response.Results {
		typ := propertyType(r.Type)
		if !typ.Valid() {
			continue
		}
		if role == meergo.Destination && r.ModificationMetadata.ReadOnlyValue {
			continue
		}
		property := types.Property{
			Name:        r.Name,
			Type:        typ,
			Nullable:    true,
			Description: r.Label + "\n\n" + r.Description,
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
		properties = append(properties, property)
	}

	schema, err := types.ObjectOf(properties)
	if err != nil {
		return types.Type{}, fmt.Errorf("cannot create schema from properties: %s", err)
	}

	return schema, nil
}

// Records returns the records of the specified target.
func (hs *HubSpot) Records(ctx context.Context, target meergo.Targets, lastChangeTime time.Time, ids, properties []string, cursor string, _ types.Type) ([]meergo.Record, string, error) {

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

	bb := hs.httpClient.GetBodyBuffer(meergo.NoEncoding) // It also supports Gzip.
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
	for i, p := range properties {
		if i > 0 {
			bb.WriteByte(',')
		}
		bb.WriteByte('"')
		bb.WriteString(p)
		bb.WriteByte('"')
	}
	bb.WriteString(`]}`)

	err := hs.call(ctx, "POST", path, bb, &response)
	if err != nil {
		return nil, "", err
	}
	if len(response.Results) == 0 {
		return nil, "", io.EOF
	}

	records := make([]meergo.Record, len(response.Results))
	for i, result := range response.Results {
		records[i] = meergo.Record{
			ID: result.ID,
		}
		updatedAt, err := time.Parse(time.RFC3339, result.UpdatedAt)
		if err != nil {
			records[i].Err = fmt.Errorf("HubSpot has returned an invalid value for updatedAt: %q", updatedAt)
			continue
		}
		records[i].Properties = result.Properties
		records[i].LastChangeTime = updatedAt.UTC()
	}

	cursor = response.Paging.Next.After
	if cursor == "" {
		err = io.EOF
	}

	return records, cursor, err
}

// Upsert updates or creates records in the app for the specified target.
func (hs *HubSpot) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {

	// Note that records.All() cannot be used because the HubSpot API's "upsert" method does not allow updating contacts using "hs_object_id".
	// See https://community.hubspot.com/t5/APIs-Integrations/Create-or-update-a-batch-of-contacts-by-unique-property-values/m-p/1047925.

	method := "update"
	if first, _ := records.Peek(); first.ID == "" {
		method = "create"
	}

	bb := hs.httpClient.GetBodyBuffer(meergo.Gzip) // It also supports NoEncoding.
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
		}
		_ = bb.EncodeKeyValue("properties", record.Properties)
		bb.WriteByte('}')
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

func (hs *HubSpot) call(ctx context.Context, method, path string, bb *meergo.BodyBuffer, response any) error {
	req, err := bb.NewRequest(ctx, method, "https://api.hubapi.com/"+path[1:])
	if err != nil {
		return err
	}
	res, err := hs.httpClient.Do(req)
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
// (https://developers.hubspot.com/docs/guides/api/crm/properties#property-type-and-fieldtype-values).
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
