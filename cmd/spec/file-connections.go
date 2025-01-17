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

	pathParameter := types.Property{
		Name:           "path",
		Type:           types.Text(),
		CreateRequired: true,
		Placeholder:    `"users.csv"`,
		Description: "The file path relative to the root of the connection. " +
			"For details on the required format, refer to the file storage connector documentation.\n\n" +
			"The path must not be empty and cannot exceed 1024 characters in length.\n\n" +
			"Note that the path must be URL-encoded. For example, if the path is `docs/users/subscribers.csv`, you need to pass it as `docs%2Fusers%2Fsubscribers.csv`.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "file-connections",
		Name:        "File connections",
		Description: "A connection enables Meergo to retrieve customer and event data from an external source location or send them to an external destination location.",
		Endpoints: []*Endpoint{
			{
				Name:        "Retrieve a file",
				Description: "Returns schema and first rows of a file.",
				Method:      POST,
				URL:         "/v0/connections/:id/files/:path",
				Parameters: []types.Property{
					{
						Name:           "connection",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						CreateRequired: true,
						Description:    "The file storage source connection from which to read the file.",
					},
					pathParameter,
					{
						Name:           "format",
						Type:           types.Text().WithValues("CVS", "Excel", "Parquet", "JSON"),
						CreateRequired: true,
						Placeholder:    `"Excel"`,
						Description:    "The file format. It correspond to the name of a file connector.",
					},
					{
						Name:           "sheet",
						Type:           types.Text(),
						Placeholder:    `"Sheet1"`,
						UpdateRequired: true,
						Description: "The sheet name. It can only be used with the Excel format, where it is required.\n\n" +
							"When provided, it must have a length between 1 and 31 characters, not start or end with a single quote `'`, and cannot contain any of the following characters: `*`, `/`, `:`, `?`, `[`, `\\`, and `]`.",
					},
					{
						Name:        "compression",
						Type:        types.Text().WithValues("", "Zip", "Gzip", "Snappy"),
						Placeholder: `"Gzip"`,
						Description: "The method used to compress the file, if applicable.\n\n" +
							"**Note:** An Excel file is inherently compressed, so no compression method needs to be specified unless the file has been further compressed.",
					},
					{
						Name:        "formatSettings",
						Type:        types.Parameter("Settings"),
						Nullable:    true,
						Placeholder: `{ "HasColumnNames": true }`,
						Description: "The settings for the file format. Refer to the documentation for the [connector](/connectors/) related to the file format to understand the available settings and their corresponding values.\n\n" +
							"If the file format does not require any settings, the `formatSettings` field may be omitted or set to null.",
					},
					{
						Name:        "limit",
						Type:        types.Int(32).WithIntRange(0, 100),
						Placeholder: `50`,
						Description: "The maximum number of rows to return along with the schema. The maximum is 100; the default is 0, meaning no rows are returned.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "schema",
							Type:        types.Parameter("Schema"),
							Placeholder: `{ ... }`,
							Description: "The file's schema.",
						},
						{
							Name:        "rows",
							Type:        types.Array(types.Map(types.JSON())),
							Placeholder: `[ { ... } ]`,
							Description: "The rows.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{404, NotFound, "path does not exist"},
				},
			},
			{
				Name:        "Read the sheet names",
				Description: "Returns the list of sheets in a file, applicable to file formats that supports sheets.",
				Method:      POST,
				URL:         "/v0/connections/:id/files/:path/sheets",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						CreateRequired: true,
						Description:    "The file storage source connection from which to read the file.",
					},
					pathParameter,
					{
						Name:           "format",
						Type:           types.Text().WithValues("Excel"),
						CreateRequired: true,
						Placeholder:    `"Excel"`,
						Description:    "The file format. It correspond to the name of a file connector.",
					},
					{
						Name:        "compression",
						Type:        types.Text().WithValues("", "Zip", "Gzip", "Snappy"),
						Placeholder: `"Gzip"`,
						Description: "The method used to compress the file, if applicable.\n\n" +
							"**Note:** An Excel file is inherently compressed, so no compression method needs to be specified unless the file has been further compressed.",
					},
					{
						Name:        "formatSettings",
						Type:        types.Parameter("Settings"),
						Nullable:    true,
						Placeholder: `{ "HasColumnNames": true }`,
						Description: "The settings for the file format. Refer to the documentation for the [connector](/connectors/) related to the file format to understand the available settings and their corresponding values.\n\n" +
							"If the file format does not require any settings, the `formatSettings` field may be omitted or set to null.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "sheets",
							Type:        types.Array(types.Text()),
							Placeholder: `[ "Sheet1", "Sheet2" ]`,
							Description: "The file's sheets. They have a length between 1 and 31 characters, not start or end with a single quote `'`, and cannot contain any of the following characters: `*`, `/`, `:`, `?`, `[`, `\\`, and `]`.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{404, NotFound, "path does not exist"},
					{422, FormatNotExist, "format does not exist"},
					{422, InvalidSettings, "format settings are not valid"},
				},
			},
			{
				Name: "Retrieve the absolute path",
				Description: "Returns the absolute path of a file based on its path relative to the connection’s root.\n\n" +
					"While this complete path isn’t used directly by other API endpoints, it can help confirm that the relative path points to the correct file.",
				Method: GET,
				URL:    "/v0/connections/:id/files/:path/absolute",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Placeholder:    "1371036433",
						CreateRequired: true,
						Description:    "The ID of the file storage connection for which the path should be retrieved.",
					},
					pathParameter,
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "path",
							Type:        types.Text(),
							Placeholder: `"/data/users.csv"`,
							Description: "The complete path of the file.",
						},
					},
				},
				Errors: []Error{
					{404, NotFound, "workspace does not exist"},
					{404, NotFound, "connection does not exist"},
					{422, InvalidPath, "path is not valid"},
					{422, InvalidPlaceholder, "placeholder is not valid"},
				},
			},
		},
	})

}
