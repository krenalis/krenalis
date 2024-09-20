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
	AlreadyConnected             errors.Code = "AlreadyConnected"
	AlterSchemaInProgress        errors.Code = "AlterSchemaInProgress"
	AuthenticationFailed         errors.Code = "AuthenticationFailed"
	CannotSendEmails             errors.Code = "CannotSendEmails"
	ConnectionDisabled           errors.Code = "ConnectionDisabled"
	ConnectionNotExist           errors.Code = "ConnectionNotExist"
	ConnectorNotExist            errors.Code = "ConnectorNotExist"
	CurrentlyConnected           errors.Code = "CurrentlyConnected"
	DataWarehouseFailed          errors.Code = "DataWarehouseFailed"
	DataWarehouseNeedsRepair     errors.Code = "DataWarehouseNeedsRepair"
	DataWarehouseNotInitialized  errors.Code = "DataWarehouseNotInitialized"
	DatabaseFailed               errors.Code = "DatabaseFailed"
	EventNotExist                errors.Code = "EventNotExist"
	EventTypeNotExist            errors.Code = "EventTypeNotExist"
	ExecutionInProgress          errors.Code = "ExecutionInProgress"
	FetchSchemaFailed            errors.Code = "FetchSchemaFailed"
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
	NoWarehouse                  errors.Code = "NoWarehouse"
	NotAllowedType               errors.Code = "NotAllowedType"
	NotCompatibleSchema          errors.Code = "NotCompatibleSchema"
	NotConnected                 errors.Code = "NotConnected"
	OrderNotExist                errors.Code = "OrderNotExist"
	OrderTypeNotSortable         errors.Code = "OrderTypeNotSortable"
	PropertyNotExist             errors.Code = "PropertyNotExist"
	ReadFileFailed               errors.Code = "ReadFileFailed"
	SheetNotExist                errors.Code = "SheetNotExist"
	TargetAlreadyExist           errors.Code = "TargetAlreadyExist"
	TooManyKeys                  errors.Code = "TooManyKeys"
	TooManyListeners             errors.Code = "TooManyListeners"
	TransformationFailed         errors.Code = "TransformationFailed"
	UniqueKey                    errors.Code = "UniqueKey"
	WorkspaceNotExist            errors.Code = "WorkspaceNotExist"
)
