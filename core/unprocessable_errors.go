// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"github.com/meergo/meergo/core/errors"
)

const (
	ActionDisabled                errors.Code = "ActionDisabled"
	AlterSchemaInExecution        errors.Code = "AlterSchemaInExecution"
	AuthenticationFailed          errors.Code = "AuthenticationFailed"
	CannotExecuteIncrementally    errors.Code = "CannotExecuteIncrementally"
	ConnectionNotExist            errors.Code = "ConnectionNotExist"
	ConnectorNotExist             errors.Code = "ConnectorNotExist"
	DifferentWarehouse            errors.Code = "DifferentWarehouse"
	EmailSendFailed               errors.Code = "EmailSendFailed"
	EmailVerificationRequired     errors.Code = "EmailVerificationRequired" // Returned by apisServer.
	EventNotExist                 errors.Code = "EventNotExist"
	EventTypeNotExist             errors.Code = "EventTypeNotExist"
	ExecutionInProgress           errors.Code = "ExecutionInProgress"
	FormatNotExist                errors.Code = "FormatNotExist"
	IdentityResolutionInExecution errors.Code = "IdentityResolutionInExecution"
	InspectionMode                errors.Code = "InspectionMode"
	InvalidEvent                  errors.Code = "InvalidEvent"
	InvalidPath                   errors.Code = "InvalidPath"
	InvalidPlaceholder            errors.Code = "InvalidPlaceholder"
	InvalidAlterSchema            errors.Code = "InvalidAlterSchema"
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
	PropertyNotExist              errors.Code = "PropertyNotExist"
	SchemaNotAligned              errors.Code = "SchemaNotAligned"
	SheetNotExist                 errors.Code = "SheetNotExist"
	SingleEventWriteKey           errors.Code = "SingleEventWriteKey"
	TargetExist                   errors.Code = "TargetExist"
	TooManyEventWriteKeys         errors.Code = "TooManyEventWriteKeys"
	TooManyListeners              errors.Code = "TooManyListeners"
	TransformationFailed          errors.Code = "TransformationFailed"
	TypeNotAllowed                errors.Code = "TypeNotAllowed"
	UnsupportedLanguage           errors.Code = "UnsupportedLanguage"
	WarehouseNonInitializable     errors.Code = "WarehouseNonInitializable"
	WarehouseDriverNotExist       errors.Code = "WarehouseDriverNotExist"
	WorkspaceNotExist             errors.Code = "WorkspaceNotExist"
)
