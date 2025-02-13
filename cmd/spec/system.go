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
		ID:          "system",
		Name:        "System",
		Description: "...",
		Endpoints: []*Endpoint{
			{
				Name:        "List supported languages",
				Description: "Returns a list of supported languages that can be used for transformation functions.",
				Method:      GET,
				URL:         "/v1/system/languages",
				Response: &Response{
					Parameters: []types.Property{
						{
							Name: "languages",
							Type: types.Object([]types.Property{
								{
									Name:        "name",
									Type:        types.Text(),
									Placeholder: `"JavaScript"`,
									Description: "The name of the supported language.",
								},
								{
									Name:        "icon",
									Type:        types.Text(),
									Placeholder: `"<svg icon>"`,
									Description: "The icon of the supported language.",
								},
							}),
							Placeholder: "...",
						},
					},
				},
			},
		},
	})

}
