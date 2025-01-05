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

func init() {

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:   "connection-keys",
		Name: "Connection keys",
		Description: "The connection keys are used for authentication when sending events to Meergo.\n\n" +
			"Mobile and server keys are private, while website keys are usually public, as they can be accessed from the site’s source code.\n" +
			"A connection can have at most 20 keys.",
		Endpoints: []*Endpoint{
			{
				Name: "Generate a connection key",
				Description: "Generate a key for a mobile, server, or website source connection. " +
					"Returns an error if the connection already has the maximum limit of 20 keys.",
				Method: POST,
				URL:    "/v0/connections/:id/keys",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "1371036433",
						Description:    "The ID of a mobile, server, or website connection.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "key",
							Type:        types.Text(),
							Placeholder: `"aC7B37Bug92OI2JSnl9eKrfGeecZT5hA"`,
							Description: "The new key of the connection.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "connection does not exist"},
					{422, TooManyKeys, "connection has already 20 keys"},
				},
			},
			{
				Name:        "List all connection keys",
				Description: "Returns all keys for mobile, server, and website source connections.",
				Method:      GET,
				URL:         "/v0/connections/:id/keys",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "1371036433",
						Description:    "The ID of a mobile, server, or website connection.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "keys",
							Type:        types.Array(types.Text()),
							Placeholder: `[ "aC7B37Bug92OI2JSnl9eKrfGeecZT5hA", "9HnSIbfreXvzD8tCb0L04xSseUNOavEp" ]`,
							Description: "The keys of the connection.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "connection does not exist"},
				},
			},
			{
				Name:        "Delete a connection key",
				Description: "Delete a key of a mobile, server, or website source connection.",
				Method:      DELETE,
				URL:         "/v0/connections/:id/keys/:key",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "1371036433",
						Description:    "The ID of a mobile, server, or website connection.",
					},
					{
						Name:        "key",
						Type:        types.Text(),
						Placeholder: `"aC7B37Bug92OI2JSnl9eKrfGeecZT5hA"`,
						Description: "The key to delete.",
					},
				},
				Errors: []Error{
					{404, NotFound, "connection does not exist"},
					{404, NotFound, "key does not exist"},
					{422, ConnectionUniqueKey, "key cannot be revoked as it is the connection’s only key"},
				},
			},
		},
	})

}
