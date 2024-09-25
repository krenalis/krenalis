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
	CannotSendEmails             errors.Code = "CannotSendEmails"
	ConnectionDisabled           errors.Code = "ConnectionDisabled"
	ConnectionNotExist           errors.Code = "ConnectionNotExist"
	ConnectorNotExist            errors.Code = "ConnectorNotExist"
	DataWarehouseFailed          errors.Code = "DataWarehouseFailed"
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
	LanguageNotSupported         errors.Code = "LanguageNotSupported"
	LinkedConnectionNotExist     errors.Code = "LinkedConnectionNotExist"
	MaintenanceMode              errors.Code = "MaintenanceMode"
	MemberEmailAlreadyExists     errors.Code = "MemberEmailAlreadyExists"
	NoColumns                    errors.Code = "NoColumns"
	NotAllowedType               errors.Code = "NotAllowedType"
	NotCompatibleSchema          errors.Code = "NotCompatibleSchema"
	OrderNotExist                errors.Code = "OrderNotExist"
	OrderTypeNotSortable         errors.Code = "OrderTypeNotSortable"
	PropertyNotExist             errors.Code = "PropertyNotExist"
	SheetNotExist                errors.Code = "SheetNotExist"
	TargetAlreadyExist           errors.Code = "TargetAlreadyExist"
	TooManyKeys                  errors.Code = "TooManyKeys"
	TooManyListeners             errors.Code = "TooManyListeners"
	TransformationFailed         errors.Code = "TransformationFailed"
	UniqueKey                    errors.Code = "UniqueKey"
	WarehouseIsNotEmpty          errors.Code = "WarehouseIsNotEmpty"
	WorkspaceNotExist            errors.Code = "WorkspaceNotExist"
)
