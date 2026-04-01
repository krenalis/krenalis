// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"github.com/krenalis/krenalis/tools/errors"
)

const (
	AlterSchemaInExecution        errors.Code = "AlterSchemaInExecution"
	AuthenticationFailed          errors.Code = "AuthenticationFailed"
	CannotRunIncrementally        errors.Code = "CannotRunIncrementally"
	ConnectionNotExist            errors.Code = "ConnectionNotExist"
	ConnectorNotExist             errors.Code = "ConnectorNotExist"
	DifferentWarehouse            errors.Code = "DifferentWarehouse"
	EmailSendFailed               errors.Code = "EmailSendFailed"
	EmailInvitationRequired       errors.Code = "EmailInvitationRequired" // Returned by apisServer.
	EventNotExist                 errors.Code = "EventNotExist"
	EventTypeNotExist             errors.Code = "EventTypeNotExist"
	FormatNotExist                errors.Code = "FormatNotExist"
	IdentityResolutionInExecution errors.Code = "IdentityResolutionInExecution"
	InspectionMode                errors.Code = "InspectionMode"
	InvalidAlterSchema            errors.Code = "InvalidAlterSchema"
	InvalidEvent                  errors.Code = "InvalidEvent"
	InvalidPath                   errors.Code = "InvalidPath"
	InvalidPlaceholder            errors.Code = "InvalidPlaceholder"
	InvalidSettings               errors.Code = "InvalidSettings"
	InvalidWarehouseSettings      errors.Code = "InvalidWarehouseSettings"
	InvitationTokenExpired        errors.Code = "InvitationTokenExpired"
	LinkedConnectionNotExist      errors.Code = "LinkedConnectionNotExist"
	MaintenanceMode               errors.Code = "MaintenanceMode"
	MemberEmailExists             errors.Code = "MemberEmailExists"
	NoColumnsFound                errors.Code = "NoColumnsFound"
	NotReadOnlyMCPSettings        errors.Code = "NotReadOnlyMCPSettings"
	OperationAlreadyExecuting     errors.Code = "OperationAlreadyExecuting"
	OrderNotExist                 errors.Code = "OrderNotExist"
	OrderTypeNotSortable          errors.Code = "OrderTypeNotSortable"
	OrganizationNotExist          errors.Code = "OrganizationNotExist"
	PipelineDisabled              errors.Code = "PipelineDisabled"
	PropertyNotExist              errors.Code = "PropertyNotExist"
	RunInProgress                 errors.Code = "RunInProgress"
	SchemaNotAligned              errors.Code = "SchemaNotAligned"
	SheetNotExist                 errors.Code = "SheetNotExist"
	SingleEventWriteKey           errors.Code = "SingleEventWriteKey"
	TargetExist                   errors.Code = "TargetExist"
	TooManyEventWriteKeys         errors.Code = "TooManyEventWriteKeys"
	TooManyListeners              errors.Code = "TooManyListeners"
	TransformationFailed          errors.Code = "TransformationFailed"
	TypeNotAllowed                errors.Code = "TypeNotAllowed"
	UnsupportedLanguage           errors.Code = "UnsupportedLanguage"
	WarehouseNotInitializable     errors.Code = "WarehouseNotInitializable"
	WarehousePlatformNotExist     errors.Code = "WarehousePlatformNotExist"
	WorkspaceNotExist             errors.Code = "WorkspaceNotExist"
)
