// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package spec

import (
	"github.com/meergo/meergo/core/types"
)

func init() {

	idProperty := types.Property{
		Name:           "id",
		Type:           types.Int(32),
		Prefilled:      "1371036433",
		CreateRequired: true,
		Description:    "The ID of the connection from which to read the users. It must be a source file storage.",
	}
	pathParameter := types.Property{
		Name:           "path",
		Type:           types.Text(),
		CreateRequired: true,
		Prefilled:      `users.csv`,
		Description: "The file path relative to the root of the file storage. " +
			"For details on the required format, refer to the file storage connector documentation.\n\n" +
			"The path must not be empty and cannot exceed 1,024 characters in length.\n\n" +
			"Note that the path must be URL-encoded. For example, if the path is `docs/users/subscribers.csv`, you need to pass it as `docs%2Fusers%2Fsubscribers.csv`.",
	}
	formatParameter := types.Property{
		Name:           "format",
		Type:           types.Text().WithValues("csv", "excel", "parquet", "json"),
		CreateRequired: true,
		Prefilled:      `excel`,
		Description:    "The file format. Note that it corresponds to the code of the file connector used to read the file.",
	}
	compressionParameter := types.Property{
		Name:      "compression",
		Type:      types.Text().WithValues("", "Zip", "Gzip", "Snappy"),
		Prefilled: `Gzip`,
		Description: "The method used to compress the file, if applicable.\n\n" +
			"**Note:** An Excel file is inherently compressed, so no compression method needs to be specified unless the file has been further compressed.",
	}
	formatSettingsParameter := types.Property{
		Name:      "formatSettings",
		Type:      types.Parameter("Settings"),
		Nullable:  true,
		Prefilled: `{"HasColumnNames":true}`,
		Description: "The settings for the file format. Refer to the documentation for the [connector](/connectors/) related to the file format to understand the available settings and their corresponding values.\n\n" +
			"If the file format does not require any settings, the `formatSettings` parameter must be either omitted or set to `null`.",
	}

	Specification.Resources = append(Specification.Resources, &Resource{
		ID:          "connections/file-storages",
		Name:        "File storages",
		Description: "These endpoints are specific to file storage connections.",
		Endpoints: []*Endpoint{
			{
				Name:        "Retrieve file",
				Description: "Returns schema and first rows of a file.",
				Method:      GET,
				URL:         "/v1/connections/:id/files",
				Parameters: []types.Property{
					idProperty,
					pathParameter,
					formatParameter,
					{
						Name:           "sheet",
						Type:           types.Text(),
						Prefilled:      `Sheet1`,
						UpdateRequired: true,
						Description: "The sheet name. It can only be used with the \"excel\" format, where it is required.\n\n" +
							"When provided, it must have a length between 1 and 31 characters, not start or end with a single quote `'`, and cannot contain any of the following characters: `*`, `/`, `:`, `?`, `[`, `\\`, and `]`.",
					},
					compressionParameter,
					formatSettingsParameter,
					{
						Name:        "limit",
						Type:        types.Int(32).WithIntRange(0, 100),
						Prefilled:   `50`,
						Description: "The maximum number of rows to return along with the schema. The maximum is 100; the default is 0, meaning no rows are returned.",
					},
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "schema",
							Type:        types.Parameter("schema"),
							Nullable:    true,
							Prefilled:   `{ ... }`,
							Description: "The file's schema. It will be null if there are no supported columns.",
						},
						{
							Name:        "rows",
							Type:        types.Array(types.Map(types.JSON())),
							Prefilled:   `[ { ... } ]`,
							Description: "The file's rows.",
						},
						{
							Name:      "issues",
							Type:      types.Array(types.Text()),
							Nullable:  true,
							Prefilled: `[ "Column \"score\" cannot be imported because its type \"INT128\" is not supported" ]`,
							Description: "The issues encountered while reading the file, such as unsupported columns, which did not prevent processing. " +
								"If it is not null, it contains at least one issue.",
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
				Name:        "Read sheets",
				Description: "Returns the list of sheets in a file, applicable to file formats that support sheets.",
				Method:      GET,
				URL:         "/v1/connections/:id/files/sheets",
				Parameters: []types.Property{
					idProperty,
					pathParameter,
					formatParameter,
					compressionParameter,
					formatSettingsParameter,
				},
				Response: &Response{
					Parameters: []types.Property{
						{
							Name:        "sheets",
							Type:        types.Array(types.Text()),
							Prefilled:   `[ "Sheet1", "Sheet2" ]`,
							Description: "The sheets of the file. They have a length between 1 and 31 characters, not start or end with a single quote `'`, and cannot contain any of the following characters: `*`, `/`, `:`, `?`, `[`, `\\`, and `]`.",
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
				Name: "Get absolute path",
				Description: "Returns the file path relative to the root of the file storage.\n\n" +
					"While this absolute path isn't used directly by other API endpoints, it can help confirm that the relative path points to the correct file.",
				Method: GET,
				URL:    "/v1/connections/:id/files/absolute",
				Parameters: []types.Property{
					{
						Name:           "id",
						Type:           types.Int(32),
						Prefilled:      "1371036433",
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
							Prefilled:   `"/data/users.csv"`,
							Description: "The absolute path of the file.",
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
