//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"github.com/meergo/meergo/apis/errors"
)

const (
	AlterSchemaInProgress        errors.Code = "AlterSchemaInProgress"
	AuthenticationFailed         errors.Code = "AuthenticationFailed"
	CannotDeleteLastKey          errors.Code = "CannotDeleteLastKey"
	ConnectionDisabled           errors.Code = "ConnectionDisabled"
	ConnectionNotExist           errors.Code = "ConnectionNotExist"
	ConnectorNotExist            errors.Code = "ConnectorNotExist"
	DataWarehouseFailed          errors.Code = "DataWarehouseFailed"
	EmailSendFailed              errors.Code = "EmailSendFailed"
	EventNotExist                errors.Code = "EventNotExist"
	EventTypeNotExist            errors.Code = "EventTypeNotExist"
	ExecutionInProgress          errors.Code = "ExecutionInProgress"
	IdentityResolutionInProgress errors.Code = "IdentityResolutionInProgress"
	InspectionMode               errors.Code = "InspectionMode"
	InvalidPath                  errors.Code = "InvalidPath"
	InvalidPlaceholder           errors.Code = "InvalidPlaceholder"
	InvalidSchemaChange          errors.Code = "InvalidSchemaChange"
	InvalidTable                 errors.Code = "InvalidTable"
	InvalidUIValues              errors.Code = "InvalidUIValues"
	InvalidWarehouseSettings     errors.Code = "InvalidWarehouseSettings"
	InvalidWarehouseType         errors.Code = "InvalidWarehouseType"
	InvitationTokenExpired       errors.Code = "InvitationTokenExpired"
	KeyNotExist                  errors.Code = "KeyNotExist"
	LinkedConnectionNotExist     errors.Code = "LinkedConnectionNotExist"
	MaintenanceMode              errors.Code = "MaintenanceMode"
	MemberEmailExists            errors.Code = "MemberEmailExists"
	NoColumnsFound               errors.Code = "NoColumnsFound"
	OrderNotExist                errors.Code = "OrderNotExist"
	OrderTypeNotSortable         errors.Code = "OrderTypeNotSortable"
	PropertyNotExist             errors.Code = "PropertyNotExist"
	SchemaNotAligned             errors.Code = "SchemaNotAligned"
	SheetNotExist                errors.Code = "SheetNotExist"
	TargetExist                  errors.Code = "TargetExist"
	TooManyKeys                  errors.Code = "TooManyKeys"
	TooManyListeners             errors.Code = "TooManyListeners"
	TransformationFailed         errors.Code = "TransformationFailed"
	TypeNotAllowed               errors.Code = "TypeNotAllowed"
	UnsupportedLanguage          errors.Code = "UnsupportedLanguage"
	WarehouseNotEmpty            errors.Code = "WarehouseNotEmpty"
	WorkspaceNotExist            errors.Code = "WorkspaceNotExist"
)
