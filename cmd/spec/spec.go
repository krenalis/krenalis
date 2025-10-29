//go:generate go run github.com/meergo/meergo/cmd/spec/gen

// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package spec

import (
	"github.com/meergo/meergo/core/types"
)

var Specification = &Spec{Resources: []*Resource{}}

type Spec struct {
	Resources []*Resource `json:"resources"`
}

type Resource struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Endpoints   []*Endpoint `json:"endpoints"`
}

type Method string

const (
	GET    Method = "GET"
	POST   Method = "POST"
	PUT    Method = "PUT"
	DELETE Method = "DELETE"
)

type Endpoint struct {
	Name           string           `json:"name"`
	Description    string           `json:"description"`
	Method         Method           `json:"method"`
	WriteKeyAuth   bool             `json:"writeKeyAuth,omitzero"`
	URL            string           `json:"url"`
	Parameters     []types.Property `json:"parameters"`
	MoreParameters []types.Property `json:"moreParameters"`
	Response       *Response        `json:"response"`
	Errors         []Error          `json:"errors"`
}

type Response struct {
	Description string           `json:"description"`
	Parameters  []types.Property `json:"parameters"`
}

type Error struct {

	// HTTP status code of the error.
	Status int `json:"status"`

	// Code for unprocessable errors; empty for others.
	Code string `json:"code"`

	// Message.
	Message string `json:"message"`
}

type Property struct {
	Name string
	// Prefilled holds an example value for documentation purposes.
	// When the parameter is passed in the body, it contains JSON code.
	// When passed in the query string it contains the value directly or,
	// if the property is an array, the values separated by a comma.
	Prefilled      string
	Type           Type
	CreateRequired bool // true if the parameter is required.
	UpdateRequired bool // true if the parameter is conditionally required.
	ReadOptional   bool
	Nullable       bool
	Description    string
}

type Type struct {
	Kind string

	BitSize int
	Minimum int
	Maximum int
	Real    bool

	ElementType    *Type
	MinElements    int
	MaxElements    int
	UniqueElements bool

	Precision int
	Scale     int

	ByteLen int
	CharLen int
	Regexp  string
	Values  []string

	Properties []Property
}

const (
	NotFound = "NotFound"

	ActionDisabled                = "ActionDisabled"
	AlterSchemaInExecution        = "AlterSchemaInExecution"
	AuthenticationFailed          = "AuthenticationFailed"
	CannotExecuteIncrementally    = "CannotExecuteIncrementally"
	ConnectionNotExist            = "ConnectionNotExist"
	ConnectorNotExist             = "ConnectorNotExist"
	DifferentWarehouse            = "DifferentWarehouse"
	EmailSendFailed               = "EmailSendFailed"
	EventNotExist                 = "EventNotExist"
	EventTypeNotExist             = "EventTypeNotExist"
	ExecutionInProgress           = "ExecutionInProgress"
	FormatNotExist                = "FormatNotExist"
	IdentityResolutionInExecution = "IdentityResolutionInExecution"
	InspectionMode                = "InspectionMode"
	InvalidAlterSchema            = "InvalidAlterSchema"
	InvalidEvent                  = "InvalidEvent"
	InvalidPath                   = "InvalidPath"
	InvalidPlaceholder            = "InvalidPlaceholder"
	InvalidSettings               = "InvalidSettings"
	InvalidWarehouseSettings      = "InvalidWarehouseSettings"
	InvitationTokenExpired        = "InvitationTokenExpired"
	LinkedConnectionNotExist      = "LinkedConnectionNotExist"
	MaintenanceMode               = "MaintenanceMode"
	MemberEmailExists             = "MemberEmailExists"
	NoColumnsFound                = "NoColumnsFound"
	NotReadOnlyMCPSettings        = "NotReadOnlyMCPSettings"
	OperationAlreadyExecuting     = "OperationAlreadyExecuting"
	OrderNotExist                 = "OrderNotExist"
	OrderTypeNotSortable          = "OrderTypeNotSortable"
	PropertyNotExist              = "PropertyNotExist"
	SchemaNotAligned              = "SchemaNotAligned"
	SheetNotExist                 = "SheetNotExist"
	SingleEventWriteKey           = "SingleEventWriteKey"
	TargetExist                   = "TargetExist"
	TooManyEventWriteKeys         = "TooManyEventWriteKeys"
	TooManyListeners              = "TooManyListeners"
	TransformationFailed          = "TransformationFailed"
	TypeNotAllowed                = "TypeNotAllowed"
	UnsupportedColumnType         = "UnsupportedColumnType"
	UnsupportedLanguage           = "UnsupportedLanguage"
	WarehouseNonInitializable     = "WarehouseNonInitializable"
	WarehouseTypeNotExist         = "WarehouseTypeNotExist"
	WorkspaceNotExist             = "WorkspaceNotExist"
)

var filterOperators = []string{
	"is",
	"is not",
	"is less than",
	"is less than or equal to",
	"is greater than",
	"is greater than or equal to",
	"is between",
	"is not between",
	"contains",
	"does not contain",
	"is one of",
	"is not one of",
	"starts with",
	"ends with",
	"is before",
	"is on or before",
	"is after",
	"is on or after",
	"is true",
	"is false",
	"is empty",
	"is not empty",
	"is null",
	"is not null",
	"exists",
	"does not exist",
}

var filterType = types.Object([]types.Property{
	{
		Name:           "logical",
		Type:           types.Text().WithValues("and", "or"),
		CreateRequired: true,
		Prefilled:      `"and"`,
	},
	{
		Name: "conditions",
		Type: types.Array(types.Object([]types.Property{
			{
				Name:           "property",
				Type:           types.Text(),
				CreateRequired: true,
				Description:    "The name or path of the property. If the property has a `json` type, it can include a json path.",
			},
			{
				Name:           "operator",
				Type:           types.Text().WithValues(filterOperators...),
				CreateRequired: true,
				Description:    "The condition's operator. The allowed values depend on the property's type.",
			},
			{
				Name:        "values",
				Type:        types.Array(types.Text().WithCharLen(60)),
				Description: "The values the operator applies to, if any. These depend on both the operator and the property's type, including whether they're present and how many there are.",
			},
		})),
		CreateRequired: true,
		Prefilled:      `[ { "property": "name", "operator": "is", "values": [ "Mary" ] } ]`,
		Description:    "A filter's condition.",
	},
})
