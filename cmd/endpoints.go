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
		"DELETE /keys/{key}":                                  organization.DeleteAccessKey, /* only Admin */
		"DELETE /members/{id}":                                organization.DeleteMember,    /* only Admin */
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
		"GET    /connections/{id}/pipeline-types":             connection.PipelineTypes,   /* only Admin */
		"GET    /connections/{id}/pipelines/schemas/Events":   connection.PipelineSchemas, /* only Admin */
		"GET    /connections/{id}/pipelines/schemas/{target}": connection.PipelineSchemas, /* only Admin */
		"GET    /connections/{id}/schemas/event":              connection.AppEventSchema,
		"GET    /connections/{id}/schemas/user":               connection.ApplicationUserSchemas,
		"GET    /connections/{id}/tables":                     connection.TableSchema,
		"GET    /connections/{id}/ui":                         connection.ServeUI, /* only Admin */
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
		"GET    /keys":                                        organization.AccessKeys,              /* only Admin */
		"GET    /members":                                     organization.Members,                 /* only Admin */
		"GET    /members/current":                             api.Member,                           /* only Admin */
		"GET    /members/invitations/{token}":                 api.MemberInvitation,                 /* only Admin */
		"GET    /members/reset-password/{token}":              api.ValidateMemberPasswordResetToken, /* only Admin */
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
		"GET    /profiles/schema/suitable-as-identifiers":     workspace.ProfilePropertiesSuitableAsIdentifiers, /* only Admin */
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
		"POST   /connections/{id}/ui-event":                   connection.ServeUI, /* only Admin */
		"POST   /connections/{src}/links/{dst}":               connection.LinkConnection,
		"POST   /events":                                      workspace.IngestEvents,
		"POST   /events/listeners":                            workspace.CreateEventListener,
		"POST   /events/{type}":                               workspace.IngestEvents,
		"POST   /expressions-properties":                      api.ExpressionsProperties, /* only Admin */
		"POST   /identity-resolution/start":                   workspace.StartIdentityResolution,
		"POST   /keys":                                        organization.CreateAccessKey, /* only Admin */
		"POST   /members":                                     organization.AddMember,       /* only Admin */
		"POST   /members/invitations":                         organization.InviteMember,    /* only Admin */
		"POST   /members/login":                               s.login,                      /* only Admin */
		"POST   /members/logout":                              s.logout,                     /* only Admin */
		"POST   /members/workos-login":                        s.workosLogin,                /* only Admin */
		"POST   /organizations":                               api.CreateOrganization,
		"POST   /pipelines":                                   connection.CreatePipeline,
		"POST   /pipelines/{id}/runs":                         pipeline.Run,
		"POST   /pipelines/{id}/ui-event":                     pipeline.ServeUI,       /* only Admin */
		"POST   /sentry/errors":                               s.forwardSentryError,   /* only Admin */
		"POST   /transformations":                             api.TransformData,      /* only Admin */
		"POST   /ui":                                          workspace.ServeUI,      /* only Admin */
		"POST   /ui-event":                                    workspace.ServeUI,      /* only Admin */
		"POST   /validate-expression":                         api.ValidateExpression, /* only Admin */
		"POST   /warehouse/repair":                            workspace.RepairWarehouse,
		"POST   /workos/actions/user-registration":            s.handleWorkosAction,
		"POST   /workspaces":                                  organization.CreateWorkspace,
		"POST   /workspaces/test":                             organization.TestWorkspaceCreation,
		"PUT    /connections/{id}":                            connection.Update,
		"PUT    /identity-resolution/settings":                workspace.UpdateIdentityResolutionSettings,
		"PUT    /keys/{key}":                                  organization.UpdateAccessKey,    /* only Admin */
		"PUT    /members/current":                             organization.UpdateMember,       /* only Admin */
		"PUT    /members/invitations/{token}":                 api.AcceptInvitation,            /* only Admin */
		"PUT    /members/reset-password":                      api.SendMemberPasswordReset,     /* only Admin */
		"PUT    /members/reset-password/{token}":              api.ChangeMemberPasswordByToken, /* only Admin */
		"PUT    /organizations/{id}":                          organization.Update,
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
