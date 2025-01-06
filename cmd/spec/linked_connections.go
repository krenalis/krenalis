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
		ID:          "linked-connections",
		Name:        "Linked connections",
		Description: "When a source connection is linked to a destination connection, events received from the source are forwarded to the destination.",
		Endpoints: []*Endpoint{
			{
				Name:        "Link connections",
				Description: "Links a source connection to a destination connection and vice versa.",
				Method:      POST,
				URL:         "/v0/connections/:src/links/:dst",
				Parameters: []types.Property{
					{
						Name:           "src",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "1371036433",
						Description:    "The ID of a source mobile, server, or website connection.",
					},
					{
						Name:           "dst",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "1554801134",
						Description:    "The ID of a destination app connection that handle events.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{422, LinkedConnectionNotExist, "linked connection does not exist"},
				},
			},
			{
				Name:        "Unlink connections",
				Description: "Unlink a source connection from a destination connection and vice versa.",
				Method:      DELETE,
				URL:         "/v0/connections/:src/links/:dst",
				Parameters: []types.Property{
					{
						Name:           "src",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "1371036433",
						Description:    "The ID of a source mobile, server, or website connection.",
					},
					{
						Name:           "dst",
						Type:           types.Int(32),
						CreateRequired: true,
						Placeholder:    "1554801134",
						Description:    "The ID of a destination app connection that handle events.",
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{422, LinkedConnectionNotExist, "linked connection does not exist"},
				},
			},
		},
	})

}
