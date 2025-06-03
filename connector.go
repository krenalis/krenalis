//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergo

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
)

// Category represents a connector's category.
type Category int

const (

	// Note: when categories are changed, the 'Category.String' method (defined
	// below) must also be changed accordingly, as well as the various
	// references to the categories under 'doc'.

	CategoryAnalytics Category = 1 << iota
	CategoryAutomation
	CategoryCRM
	CategoryDatabase
	CategoryEcommerce
	CategoryEmail
	CategoryEventStreaming
	CategoryFile
	CategoryFileStorage
	CategoryMarketing
	CategoryMobile
	CategoryOLAP
	CategorySDK
	CategoryTest
	CategoryWebsite
)

// String returns the string representation of a Category.
func (c Category) String() string {
	switch c {
	case CategoryAnalytics:
		return "Analytics"
	case CategoryAutomation:
		return "Automation"
	case CategoryCRM:
		return "CRM"
	case CategoryDatabase:
		return "Database"
	case CategoryEcommerce:
		return "E-commerce"
	case CategoryEmail:
		return "Email"
	case CategoryEventStreaming:
		return "Event streaming"
	case CategoryFile:
		return "File"
	case CategoryFileStorage:
		return "File storage"
	case CategoryMarketing:
		return "Marketing"
	case CategoryMobile:
		return "Mobile"
	case CategoryOLAP:
		return "OLAP"
	case CategorySDK:
		return "SDK"
	case CategoryTest:
		return "Test"
	case CategoryWebsite:
		return "Website"
	default:
		return fmt.Sprintf("<unexpected category %d>", c)
	}
}

type ConnectorDocumentation struct {
	Source      ConnectorRoleDocumentation
	Destination ConnectorRoleDocumentation
}

type ConnectorRoleDocumentation struct {
	Summary  string
	Overview string
}

// ConnectorInfo is the interface implemented by connector infos.
type ConnectorInfo interface {
	ReflectType() reflect.Type
}

// A SetSettingsFunc value is a function used by connectors to set settings.
type SetSettingsFunc func(context.Context, []byte) error

// TimeLayouts represents the layouts for time values.
// If a layout is left empty, it is ISO 8601.
type TimeLayouts struct {
	DateTime string // if left empty, values are formatted with the layout "2006-01-02T15:04:05.999Z"
	Date     string // if left empty, values are formatted with the layout "2006-01-02"
	Time     string // if left empty, values are formatted with the layout "15:04:05.999Z"
}

// HTTPClient is the interface implemented by the HTTP client used by
// connectors.
type HTTPClient interface {

	// Do sends an HTTP request with an Authorization header if required. It returns
	// the response and ensures that the request body is closed, even in the case of
	// errors. Redirects are not followed.
	//
	// If an error occurs during GET, PUT, DELETE, or HEAD requests, it retries
	// using the client's backoff policy or a default policy if the client has no
	// policy.
	Do(req *http.Request) (res *http.Response, err error)

	// DoIdempotent behaves like Do, but unlike Do, which assumes GET, PUT, DELETE,
	// and HEAD requests are idempotent by default, it allows to explicitly specify
	// idempotency.
	//
	// If an error occurs during an idempotent request, it retries using the
	// client's backoff policy or a default policy if the client has no policy.
	DoIdempotent(req *http.Request, idempotent bool) (*http.Response, error)

	// ClientSecret returns the OAuth client secret of the HTTP client.
	ClientSecret() (string, error)

	// AccessToken returns an OAuth access token.
	AccessToken(ctx context.Context) (string, error)

	// UUID returns a random version 4 UUID, suitable for use as an idempotency key.
	UUID() string
}

// Role represents a role.
type Role int

const (
	Both        Role = iota // both
	Source                  // source
	Destination             // destination
)

// String returns the string representation of role.
// It panics if role is not a valid Role value.
func (role Role) String() string {
	switch role {
	case Both:
		return "Both"
	case Source:
		return "Source"
	case Destination:
		return "Destination"
	}
	panic("invalid role")
}
