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

	"github.com/google/uuid"
)

// Categories represents connector categories.
type Categories int

const (

	// Note: when categories are changed, the 'Categories.String' method
	// (defined below) must also be changed accordingly, as well as the various
	// references to the categories under 'doc'.

	CategoryAnalytics Categories = 1 << iota
	CategorySDKAndAPI
	CategoryCRM
	CategoryDatabase
	CategoryEcommerce
	CategoryEventStreaming
	CategoryFile
	CategoryFileStorage
	CategoryMarketing
	CategoryTest
)

// String returns the string representation of a single category.
func (c Categories) String() string {
	switch c {
	case CategoryAnalytics:
		return "Analytics"
	case CategorySDKAndAPI:
		return "SDK & API"
	case CategoryCRM:
		return "CRM"
	case CategoryDatabase:
		return "Database"
	case CategoryEcommerce:
		return "E-commerce"
	case CategoryEventStreaming:
		return "Event streaming"
	case CategoryFile:
		return "File"
	case CategoryFileStorage:
		return "File storage"
	case CategoryMarketing:
		return "Marketing"
	case CategoryTest:
		return "Test"
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

	// Do sends an HTTP request and returns the corresponding HTTP response.
	//
	// If the connector supports OAuth, it adds the Authorization header
	// automatically.
	//
	// It retries the request on network errors or when the connector's retry
	// policy applies. A request is retried only if it is idempotent
	// (see http.Transport for details), which is defined as:
	//
	//   - method is GET, HEAD, OPTIONS, or TRACE and Request.GetBody is set, or
	//   - Request.Header contains an Idempotency-Key or X-Idempotency-Key key.
	//
	// An empty header value is considered idempotent but is not sent.
	//
	// It always closes the request body, even if an error occurs.
	// It does not follow redirects.
	Do(req *http.Request) (res *http.Response, err error)

	// ClientSecret returns the OAuth client secret of the HTTP client.
	ClientSecret() (string, error)

	// AccessToken returns an OAuth access token.
	AccessToken(ctx context.Context) (string, error)
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

// UUID returns a random version 4 UUID. For example, it can be used as an
// idempotency key.
func UUID() string {
	return uuid.NewString()
}
