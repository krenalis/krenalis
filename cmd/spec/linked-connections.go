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

	srcParameter := types.Property{
		Name:           "src",
		Type:           types.Int(32),
		CreateRequired: true,
		Prefilled:      "1371036433",
		Description:    "The ID of the source connection. It must be an SDK or webhook connection.",
	}
	dstParameter := types.Property{
		Name:           "dst",
		Type:           types.Int(32),
		CreateRequired: true,
		Prefilled:      "1554801134",
		Description:    "The ID of a destination connection. It must be an API that supports events.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "linked-connections",
		Name:        "Linked connections",
		Description: "SDK and webhook source connections can be linked to destination API connections. When linked, events received from the source are sent to the API.",
		Endpoints: []*Endpoint{
			{
				Name:        "Link connections",
				Description: "Links a source to a destination. It succeeds if the connections are already linked.",
				Method:      POST,
				URL:         "/v1/connections/:src/links/:dst",
				Parameters: []types.Property{
					srcParameter,
					dstParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
				},
			},
			{
				Name:        "Unlink connections",
				Description: "Unlinks a source from a destination. It succeeds if the connections are not linked.",
				Method:      DELETE,
				URL:         "/v1/connections/:src/links/:dst",
				Parameters: []types.Property{
					srcParameter,
					dstParameter,
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
				},
			},
		},
	})

}
