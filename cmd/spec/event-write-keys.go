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
		ID:   "event-write-keys",
		Name: "Event write keys",
		Description: "Event write keys are used for authentication when sending events from websites, mobile apps, and servers " +
			"through the [Ingest event](events#ingest-event) and [Ingest batch events](events#ingest-batch-events) endpoints.\n\n" +
			"Keys for website and mobile connections are usually public, as they can be exposed in a website’s source code or on a mobile device. " +
			"In contrast, keys for server connections should always remain private.",
		Endpoints: []*Endpoint{
			{
				Name: "Create event write key",
				Description: "Creates an event write key for a website, mobile, or server connection. " +
					"Returns an error if the connection already has the maximum limit of 20 keys.",
				Method: POST,
				URL:    "/v1/connections/:id/event-write-keys",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "1371036433",
						Description:    "The ID of the connection for which to create the key. It must be a website, mobile, or server.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "key",
							Type:        types.Text(),
							Placeholder: `"aC7B37Bug92OI2JSnl9eKrfGeecZT5hA"`,
							Description: "The new created key.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{422, TooManyEventWriteKeys, "connection has already 20 keys"},
				},
			},
			{
				Name:        "List all event write keys",
				Description: "Returns all event write keys for a website, mobile, or server connection.",
				Method:      GET,
				URL:         "/v1/connections/:id/event-write-keys",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "1371036433",
						Description:    "The ID of the connection for which to return the keys. It must be a website, mobile, or server.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "keys",
							Type:        types.Array(types.Text()),
							Placeholder: `[ "aC7B37Bug92OI2JSnl9eKrfGeecZT5hA", "9HnSIbfreXvzD8tCb0L04xSseUNOavEp" ]`,
							Description: "The keys of the connection. At least one key is guaranteed to be present.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
				},
			},
			{
				Name:        "Delete write key",
				Description: "Deletes a write key from a website, mobile, or server connection. If the connection has only one key, it cannot be deleted.",
				Method:      DELETE,
				URL:         "/v1/connections/:id/event-write-keys/:key",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "1371036433",
						Description:    "The ID of the connection for which to delete the key. It must be a website, mobile, or server.",
					},
					{
						Name:           "key",
						Type:           types.Text(),
						CreateRequired: true,
						Placeholder:    `"aC7B37Bug92OI2JSnl9eKrfGeecZT5hA"`,
						Description:    "The key to delete.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{404, NotFound, "key does not exist"},
					{422, SingleEventWriteKey, "key cannot be deleted as it is the connection’s only key"},
				},
			},
		},
	})

}
