//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package spec

import (
	"github.com/meergo/meergo/core/types"
)

func init() {

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "others",
		Name: "Others",
		Description: "These endpoints provide general information about the server. " +
			"At the moment, they allow to retrieve the set of languages supported by transformation functions, as well as public information that is primarily consumed by the Admin Console.",
		Endpoints: []*Endpoint{
			{
				Name:        "Get public metadata",
				Description: "Retrieves public, non-sensitive metadata about the server. The endpoint is unauthenticated and safe for client-side use, exposing only data intended for discovery such as server capabilities and configuration hints.",
				Method:      GET,
				URL:         "/v1/public/metadata",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "installationID",
							Type:        types.UUID(),
							Prefilled:   `"9d16fb0b-adf1-4b84-b1b9-be1d017f0dfc"`,
							Description: "Uniquely identifies the Meergo installation.",
						},
						{
							Name:        "externalURL",
							Type:        types.Text(),
							Prefilled:   `"https://example.com/"`,
							Description: "Public address used to access the server from outside the internal network.",
						},
						{
							Name:        "externalEventURL",
							Type:        types.Text(),
							Prefilled:   `"https://example.com/api/v1/events"`,
							Description: "Public address for the event ingestion endpoint `/api/v1/events`, making it reachable externally.",
						},
						{
							Name:        "javascriptSDKURL",
							Type:        types.Text(),
							Prefilled:   `"https://cdn.jsdelivr.net/npm/@meergo/javascript-sdk/dist/meergo.min.js"`,
							Description: "Location where the JavaScript SDK is served.",
						},
						{
							Name:        "skipMemberEmailVerification",
							Type:        types.Boolean(),
							Prefilled:   `false`,
							Description: "Indicates whether email verification is required when a member registers.",
						},
						{
							Name:        "canSendMemberPasswordReset",
							Type:        types.Boolean(),
							Prefilled:   `true`,
							Description: "Indicates whether the system can send password reset emails to members.",
						},
						{
							Name:      "telemetryLevel",
							Type:      types.Text().WithValues("none", "errors", "stats", "all"),
							Prefilled: `"all"`,
							Description: "Required telemetry data reporting mode in Meergo:\n\n" +
								"- `\"none\"`: no telemetry data must be sent\n" +
								"- `\"errors\"`: only error-related telemetry data must be sent\n" +
								"- `\"stats\"`: only usage statistics must be sent\n" +
								"- `\"all\"`: both error-related data and usage statistics must be sent\n\n",
						},
					},
				},
			},
			{
				Name:        "List transformation languages",
				Description: "Returns a list of supported languages that can be used for transformation functions.",
				Method:      GET,
				URL:         "/v1/system/transformations/languages",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name: "languages",
							Type: types.Array(types.Object([]types.Property{
								{
									Name:        "name",
									Type:        types.Text(),
									Prefilled:   `"JavaScript"`,
									Description: "The name of the supported language.",
								},
								{
									Name:        "icon",
									Type:        types.Text(),
									Prefilled:   `"<svg icon>"`,
									Description: "The icon of the supported language.",
								},
							})),
							Prefilled: "...",
						},
					},
				},
			},
		},
	})

}
