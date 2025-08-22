//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package cmd

import (
	"net/http"
)

// endpointHandler represents an API endpoint handler.
type endpointHandler func(w http.ResponseWriter, r *http.Request) (any, error)

// endpoints returns the endpoints for the provided API server.
func endpoints(s *apisServer) map[string]endpointHandler {
	api := api{s}
	connector := connector{s}
	organization := organization{s}
	workspace := workspace{s}
	connection := connection{s}
	action := action{s}
	return map[string]endpointHandler{
		"DELETE /actions/{id}":                              action.Delete,
		"DELETE /connections/{id}":                          connection.Delete,
		"DELETE /connections/{id}/event-write-keys/{key}":   connection.DeleteEventWriteKey,
		"DELETE /connections/{src}/links/{dst}":             connection.UnlinkConnection,
		"DELETE /events/listeners/{id}":                     workspace.DeleteEventListener,
		"DELETE /keys/{key}":                                organization.DeleteAccessKey, /* only Admin */
		"DELETE /members/{id}":                              organization.DeleteMember,    /* only Admin */
		"DELETE /workspaces/current":                        workspace.Delete,
		"GET    /actions/errors/{start}/{end}":              workspace.ActionErrors,
		"GET    /actions/executions":                        workspace.Executions,
		"GET    /actions/executions/{id}":                   workspace.Execution,
		"GET    /actions/metrics/dates/{start}/{end}":       workspace.ActionMetricsPerDate,
		"GET    /actions/metrics/days/{days}":               workspace.ActionMetricsPerDay,
		"GET    /actions/metrics/hours/{hours}":             workspace.ActionMetricsPerHour,
		"GET    /actions/metrics/minutes/{minutes}":         workspace.ActionMetricsPerMinute,
		"GET    /actions/{id}":                              workspace.Action,
		"GET    /connections":                               workspace.Connections,
		"GET    /connections/auth-token":                    workspace.AuthToken,
		"GET    /connections/auth-url":                      connector.AuthCodeURL,
		"GET    /connections/{id}":                          workspace.Connection,
		"GET    /connections/{id}/action-types":             connection.ActionTypes,   /* only Admin */
		"GET    /connections/{id}/actions/schemas/Events":   connection.ActionSchemas, /* only Admin */
		"GET    /connections/{id}/actions/schemas/{target}": connection.ActionSchemas, /* only Admin */
		"GET    /connections/{id}/event-write-keys":         connection.EventWriteKeys,
		"GET    /connections/{id}/files":                    connection.File,
		"GET    /connections/{id}/files/absolute":           connection.AbsolutePath,
		"GET    /connections/{id}/files/sheets":             connection.Sheets,
		"GET    /connections/{id}/identities":               connection.Identities,
		"GET    /connections/{id}/schemas/event":            connection.AppEventSchema,
		"GET    /connections/{id}/schemas/user":             connection.AppUserSchemas,
		"GET    /connections/{id}/tables":                   connection.TableSchema,
		"GET    /connections/{id}/ui":                       connection.ServeUI, /* only Admin */
		"GET    /connections/{id}/users":                    connection.AppUsers,
		"GET    /connectors":                                api.Connectors,
		"GET    /connectors/{name}":                         api.Connector,
		"GET    /connectors/{name}/documentation":           api.ConnectorDocumentation,
		"GET    /event-url":                                 api.EventURL,
		"GET    /events":                                    workspace.Events,
		"GET    /events/listeners/{id}":                     workspace.ListenedEvents,
		"GET    /events/schema":                             api.EventSchema,
		"GET    /events/settings/{write_key}":               api.EventsSettings,
		"GET    /identity-resolution/latest":                workspace.LatestIdentityResolution,
		"GET    /identity-resolution/settings":              workspace.IdentityResolutionSettings,
		"GET    /installation-id":                           api.InstallationID,                   /* only Admin */
		"GET    /javascript-sdk-url":                        api.JavaScriptSDKURL,                 /* only Admin */
		"GET    /keys":                                      organization.AccessKeys,              /* only Admin */
		"GET    /members":                                   organization.Members,                 /* only Admin */
		"GET    /members/current":                           api.Member,                           /* only Admin */
		"GET    /members/invitations/{token}":               api.MemberInvitation,                 /* only Admin */
		"GET    /members/reset-password/{token}":            api.ValidateMemberPasswordResetToken, /* only Admin */
		"GET    /skip-member-email-verification":            api.SkipMemberEmailVerification,      /* only Admin */
		"GET    /system/transformations/languages":          api.TransformationLanguages,
		"GET    /telemetry/level":                           api.SentryTelemetryLevel, /* only Admin */
		"GET    /users":                                     workspace.Users,
		"GET    /users/schema":                              workspace.UserSchema,
		"GET    /users/schema/latest-alter":                 workspace.LatestAlterUserSchema,
		"GET    /users/schema/suitable-as-identifiers":      workspace.UserPropertiesSuitableAsIdentifiers, /* only Admin */
		"GET    /users/{id}/events":                         workspace.UserEvents,
		"GET    /users/{id}/identities":                     workspace.Identities,
		"GET    /users/{id}/traits":                         workspace.Traits,
		"GET    /warehouse":                                 workspace.Warehouse,
		"GET    /warehouse/types":                           api.WarehouseTypes,
		"GET    /workspaces":                                organization.Workspaces,
		"GET    /workspaces/current":                        organization.Workspace,
		"POST   /actions":                                   connection.CreateAction,
		"POST   /actions/{id}/exec":                         action.Execute,
		"POST   /actions/{id}/ui-event":                     action.ServeUI, /* only Admin */
		"POST   /connections":                               workspace.CreateConnection,
		"POST   /connections/{id}/event-write-keys":         connection.CreateEventWriteKey,
		"POST   /connections/{id}/preview-send-event":       connection.PreviewSendEvent,
		"POST   /connections/{id}/query":                    connection.ExecQuery,
		"POST   /connections/{id}/ui-event":                 connection.ServeUI, /* only Admin */
		"POST   /connections/{src}/links/{dst}":             connection.LinkConnection,
		"POST   /events":                                    workspace.IngestEvents,
		"POST   /events/listeners":                          workspace.CreateEventListener,
		"POST   /events/{type}":                             workspace.IngestEvents,
		"POST   /expressions-properties":                    api.ExpressionsProperties, /* only Admin */
		"POST   /identity-resolution/start":                 workspace.StartIdentityResolution,
		"POST   /keys":                                      organization.CreateAccessKey, /* only Admin */
		"POST   /members":                                   organization.AddMember,       /* only Admin */
		"POST   /members/invitations":                       organization.InviteMember,    /* only Admin */
		"POST   /members/login":                             s.login,                      /* only Admin */
		"POST   /members/logout":                            s.logout,                     /* only Admin */
		"POST   /sentry/errors":                             s.forwardSentryError,         /* only Admin */
		"POST   /transformations":                           api.TransformData,            /* only Admin */
		"POST   /ui":                                        workspace.ServeUI,            /* only Admin */
		"POST   /ui-event":                                  workspace.ServeUI,            /* only Admin */
		"POST   /validate-expression":                       api.ValidateExpression,       /* only Admin */
		"POST   /warehouse/repair":                          workspace.RepairWarehouse,
		"POST   /workspaces":                                organization.CreateWorkspace,
		"POST   /workspaces/test":                           organization.TestWorkspaceCreation,
		"PUT    /actions/{id}":                              action.Update,
		"PUT    /actions/{id}/schedule":                     action.SetSchedulePeriod,
		"PUT    /actions/{id}/status":                       action.SetStatus,
		"PUT    /connections/{id}":                          connection.Update,
		"PUT    /identity-resolution/settings":              workspace.UpdateIdentityResolutionSettings,
		"PUT    /keys/{key}":                                organization.UpdateAccessKey,    /* only Admin */
		"PUT    /members/current":                           organization.UpdateMember,       /* only Admin */
		"PUT    /members/invitations/{token}":               api.AcceptInvitation,            /* only Admin */
		"PUT    /members/reset-password":                    api.SendMemberPasswordReset,     /* only Admin */
		"PUT    /members/reset-password/{token}":            api.ChangeMemberPasswordByToken, /* only Admin */
		"PUT    /users/schema":                              workspace.AlterUserSchema,
		"PUT    /users/schema/preview":                      workspace.PreviewAlterUserSchema,
		"PUT    /warehouse":                                 workspace.UpdateWarehouse,
		"PUT    /warehouse/mode":                            workspace.UpdateWarehouseMode,
		"PUT    /warehouse/test":                            workspace.TestWarehouseUpdate,
		"PUT    /workspaces/current":                        workspace.Update,
	}
}
