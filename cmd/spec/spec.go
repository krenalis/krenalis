//go:generate go run github.com/meergo/meergo/cmd/spec/gen

//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package spec

import (
	"github.com/meergo/meergo/types"
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
	Name           string
	Placeholder    string
	Type           Type
	CreateRequired bool // true is the parameter is required.
	UpdateRequired bool // true if the parameter is conditionally required.
	ReadOptional   bool
	Nullable       bool
	Description    string
}

type Type struct {
	Name string

	BitSize int
	Minimum int
	Maximum int
	Real    bool

	ElementType    []Type
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

	ActionDisabled               = "ActionDisabled"
	AlterSchemaInProgress        = "AlterSchemaInProgress"
	AuthenticationFailed         = "AuthenticationFailed"
	CannotExecuteIncrementally   = "CannotExecuteIncrementally"
	ConnectionNotExist           = "ConnectionNotExist"
	ConnectorNotExist            = "ConnectorNotExist"
	DifferentWarehouse           = "DifferentWarehouse"
	EmailSendFailed              = "EmailSendFailed"
	EventNotExist                = "EventNotExist"
	EventTypeNotExist            = "EventTypeNotExist"
	EventTypeNotExists           = "EventTypeNotExists"
	ExecutionInProgress          = "ExecutionInProgress"
	FormatNotExist               = "FormatNotExist"
	IdentityResolutionInProgress = "IdentityResolutionInProgress"
	InspectionMode               = "InspectionMode"
	InvalidPath                  = "InvalidPath"
	InvalidPlaceholder           = "InvalidPlaceholder"
	InvalidSchemaUpdate          = "InvalidSchemaUpdate"
	InvalidSettings              = "InvalidSettings"
	InvalidWarehouseSettings     = "InvalidWarehouseSettings"
	InvitationTokenExpired       = "InvitationTokenExpired"
	LinkedConnectionNotExist     = "LinkedConnectionNotExist"
	MaintenanceMode              = "MaintenanceMode"
	MemberEmailExists            = "MemberEmailExists"
	NoColumnsFound               = "NoColumnsFound"
	OrderNotExist                = "OrderNotExist"
	OrderTypeNotSortable         = "OrderTypeNotSortable"
	PropertyNotExist             = "PropertyNotExist"
	SchemaNotAligned             = "SchemaNotAligned"
	SheetNotExist                = "SheetNotExist"
	SingleEventWriteKey          = "SingleEventWriteKey"
	TargetExist                  = "TargetExist"
	TooManyEventWriteKeys        = "TooManyEventWriteKeys"
	TooManyListeners             = "TooManyListeners"
	TransformationFailed         = "TransformationFailed"
	TypeNotAllowed               = "TypeNotAllowed"
	UnsupportedColumnType        = "UnsupportedColumnType"
	UnsupportedLanguage          = "UnsupportedLanguage"
	WarehouseNonInitializable    = "WarehouseNonInitializable"
	WarehouseTypeNotExist        = "WarehouseTypeNotExist"
	WorkspaceNotExist            = "WorkspaceNotExist"
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
		Placeholder:    "and",
	},
	{
		Name: "conditions",
		Type: types.Array(types.Object([]types.Property{
			{
				Name:           "property",
				Type:           types.Text(),
				CreateRequired: true,
				Description:    "The name or path of the property. If the property has a JSON type, it can include a JSON path.",
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
		Placeholder:    "and",
		Description:    "A filter's condition.",
	},
})
