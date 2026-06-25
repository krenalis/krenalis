// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http"
)

// endpointHandler represents an API endpoint handler.
type endpointHandler func(w http.ResponseWriter, r *http.Request) (any, error)

// endpoints returns the endpoints for the provided API server.
//
// Keep patterns in sync with the client scrub patterns in
// `admin/src/lib/telemetry/scrubURL.ts`.
func endpoints(s *apisServer) map[string]endpointHandler {
	api := api{s}
	connector := connector{s}
	organization := organization{s}
	workspace := workspace{s}
	connection := connection{s}
	pipeline := pipeline{s}
	return map[string]endpointHandler{
		"DELETE /connections/{id}":                            connection.Delete,
		"DELETE /connections/{id}/event-write-keys/{key}":     connection.DeleteEventWriteKey,
		"DELETE /connections/{src}/links/{dst}":               connection.UnlinkConnection,
		"DELETE /events/listeners/{id}":                       workspace.DeleteEventListener,
		"DELETE /keys/{key}":                                  organization.DeleteAccessKey, /* Admin console only */
		"DELETE /members/{id}":                                organization.DeleteMember,    /* Admin console only */
		"DELETE /organizations/{id}":                          organization.Delete,
		"DELETE /pipelines/{id}":                              pipeline.Delete,
		"DELETE /workspaces/current":                          workspace.Delete,
		"GET    /{$}":                                         api.Index,
		"GET    /connections":                                 workspace.Connections,
		"GET    /connections/auth-token":                      workspace.AuthToken,
		"GET    /connections/auth-url":                        connector.AuthURL,
		"GET    /connections/{id}":                            workspace.Connection,
		"GET    /connections/{id}/event-write-keys":           connection.EventWriteKeys,
		"GET    /connections/{id}/files":                      connection.File,
		"GET    /connections/{id}/files/absolute":             connection.AbsolutePath,
		"GET    /connections/{id}/files/sheets":               connection.Sheets,
		"GET    /connections/{id}/identities":                 connection.Identities,
		"GET    /connections/{id}/pipeline-types":             connection.PipelineTypes,   /* Admin console only */
		"GET    /connections/{id}/pipelines/schemas/Events":   connection.PipelineSchemas, /* Admin console only */
		"GET    /connections/{id}/pipelines/schemas/{target}": connection.PipelineSchemas, /* Admin console only */
		"GET    /connections/{id}/schemas/event":              connection.AppEventSchema,
		"GET    /connections/{id}/schemas/user":               connection.ApplicationUserSchemas,
		"GET    /connections/{id}/tables":                     connection.TableSchema,
		"GET    /connections/{id}/ui":                         connection.ServeUI, /* Admin console only */
		"GET    /connections/{id}/users":                      connection.ApplicationUsers,
		"GET    /connectors":                                  api.Connectors,
		"GET    /connectors/{code}":                           api.Connector,
		"GET    /connectors/{code}/documentation":             api.ConnectorDocumentation,
		"GET    /events":                                      workspace.Events,
		"GET    /events/listeners/{id}":                       workspace.ListenedEvents,
		"GET    /events/schema":                               api.EventSchema,
		"GET    /events/settings/{write_key}":                 api.EventsSettings,
		"GET    /identity-resolution/latest":                  workspace.LatestIdentityResolution,
		"GET    /identity-resolution/settings":                workspace.IdentityResolutionSettings,
		"GET    /keys":                                        organization.AccessKeys,              /* Admin console only */
		"GET    /members":                                     organization.Members,                 /* Admin console only */
		"GET    /members/current":                             api.Member,                           /* Admin console only */
		"GET    /members/invitations/{token}":                 api.MemberInvitation,                 /* Admin console only */
		"GET    /members/reset-password/{token}":              api.ValidateMemberPasswordResetToken, /* Admin console only */
		"GET    /organizations/{id}":                          api.Organization,
		"GET    /organizations":                               api.Organizations,
		"GET    /pipelines/errors/{start}/{end}":              workspace.PipelineErrors,
		"GET    /pipelines/metrics/dates/{start}/{end}":       workspace.PipelineMetricsPerDate,
		"GET    /pipelines/metrics/days/{days}":               workspace.PipelineMetricsPerDay,
		"GET    /pipelines/metrics/hours/{hours}":             workspace.PipelineMetricsPerHour,
		"GET    /pipelines/metrics/minutes/{minutes}":         workspace.PipelineMetricsPerMinute,
		"GET    /pipelines/runs":                              workspace.PipelineRuns,
		"GET    /pipelines/runs/{id}":                         workspace.PipelineRun,
		"GET    /pipelines/{id}":                              workspace.Pipeline,
		"GET    /profiles":                                    workspace.Profiles,
		"GET    /profiles/schema":                             workspace.ProfileSchema,
		"GET    /profiles/schema/latest-alter":                workspace.LatestAlterProfileSchema,
		"GET    /profiles/schema/suitable-as-identifiers":     workspace.ProfilePropertiesSuitableAsIdentifiers, /* Admin console only */
		"GET    /profiles/{kpid}/attributes":                  workspace.Attributes,
		"GET    /profiles/{kpid}/events":                      workspace.ProfileEvents,
		"GET    /profiles/{kpid}/identities":                  workspace.Identities,
		"GET    /public/metadata":                             api.PublicMetadata,
		"GET    /system/transformations/languages":            api.TransformationLanguages,
		"GET    /warehouse":                                   workspace.Warehouse,
		"GET    /warehouse/platforms":                         api.WarehousePlatforms,
		"GET    /workspaces":                                  organization.Workspaces,
		"GET    /workspaces/current":                          organization.Workspace,
		"POST   /connections":                                 workspace.CreateConnection,
		"POST   /connections/{id}/event-write-keys":           connection.CreateEventWriteKey,
		"POST   /connections/{id}/preview-send-event":         connection.PreviewSendEvent,
		"POST   /connections/{id}/query":                      connection.ExecQuery,
		"POST   /connections/{id}/ui-event":                   connection.ServeUI, /* Admin console only */
		"POST   /connections/{src}/links/{dst}":               connection.LinkConnection,
		"POST   /events":                                      workspace.IngestEvents,
		"POST   /events/listeners":                            workspace.CreateEventListener,
		"POST   /events/{type}":                               workspace.IngestEvents,
		"POST   /expressions-properties":                      api.ExpressionsProperties, /* Admin console only */
		"POST   /identity-resolution/start":                   workspace.StartIdentityResolution,
		"POST   /keys":                                        organization.CreateAccessKey, /* Admin console only */
		"POST   /members":                                     organization.AddMember,       /* Admin console only */
		"POST   /members/invitations":                         organization.InviteMember,    /* Admin console only */
		"POST   /members/login":                               s.login,                      /* Admin console only */
		"POST   /members/logout":                              s.logout,                     /* Admin console only */
		"POST   /organizations":                               api.CreateOrganization,
		"POST   /pipelines":                                   connection.CreatePipeline,
		"POST   /pipelines/{id}/runs":                         pipeline.Run,
		"POST   /pipelines/{id}/ui-event":                     pipeline.ServeUI,       /* Admin console only */
		"POST   /sentry/errors":                               s.forwardSentryError,   /* Admin console only */
		"POST   /transformations":                             api.TransformData,      /* Admin console only */
		"POST   /ui":                                          workspace.ServeUI,      /* Admin console only */
		"POST   /ui-event":                                    workspace.ServeUI,      /* Admin console only */
		"POST   /validate-expression":                         api.ValidateExpression, /* Admin console only */
		"POST   /warehouse/repair":                            workspace.RepairWarehouse,
		"POST   /workspaces":                                  organization.CreateWorkspace,
		"POST   /workspaces/test":                             organization.TestWorkspaceCreation,
		"PUT    /connections/{id}":                            connection.Update,
		"PUT    /identity-resolution/settings":                workspace.UpdateIdentityResolutionSettings,
		"PUT    /keys/{key}":                                  organization.UpdateAccessKey,    /* Admin console only */
		"PUT    /members/current":                             organization.UpdateMember,       /* Admin console only */
		"PUT    /members/invitations/{token}":                 api.AcceptInvitation,            /* Admin console only */
		"PUT    /members/reset-password":                      api.SendMemberPasswordReset,     /* Admin console only */
		"PUT    /members/reset-password/{token}":              api.ChangeMemberPasswordByToken, /* Admin console only */
		"PUT    /organizations/{id}":                          organization.Update,
		"PUT    /organizations/{id}/status":                   organization.SetStatus,
		"PUT    /pipelines/{id}":                              pipeline.Update,
		"PUT    /pipelines/{id}/schedule":                     pipeline.SetSchedulePeriod,
		"PUT    /pipelines/{id}/status":                       pipeline.SetStatus,
		"PUT    /profiles/schema":                             workspace.AlterProfileSchema,
		"PUT    /profiles/schema/preview":                     workspace.PreviewAlterProfileSchema,
		"PUT    /warehouse":                                   workspace.UpdateWarehouse,
		"PUT    /warehouse/mode":                              workspace.UpdateWarehouseMode,
		"PUT    /warehouse/test":                              workspace.TestWarehouseUpdate,
		"PUT    /workspaces/current":                          workspace.Update,
	}
}
