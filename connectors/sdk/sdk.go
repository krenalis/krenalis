//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package sdk implements connectors for .Net, Android, Go, Java, JavaScript,
// Node.js, and Python.
//
// .NET is a trademark of Microsoft Corporation.
// This connector is not affiliated with or endorsed by Microsoft Corporation.
//
// Android and Go are trademarks of Google LLC.
// This connector is not affiliated with or endorsed by Google LLC.
//
// Java and JavaScript are trademarks of Oracle Corporation.
// This connector is not affiliated with or endorsed by Oracle Corporation.
//
// Node.js is a trademark of the OpenJS Foundation.
// This connector is not affiliated with or endorsed by the OpenJS Foundation.
//
// Python is a trademark of the Python Software Foundation.
// This connector is not affiliated with or endorsed by the Python Software Foundation.
package sdk

import (
	_ "embed"

	"github.com/meergo/meergo"
)

//go:embed documentation/dotnet/overview.md
var dotnetOverview string

//go:embed documentation/android/overview.md
var androidOverview string

//go:embed documentation/go/overview.md
var goOverview string

//go:embed documentation/java/overview.md
var javaOverview string

//go:embed documentation/javascript/overview.md
var javaScriptOverview string

//go:embed documentation/node/overview.md
var nodeOverview string

//go:embed documentation/python/overview.md
var pythonOverview string

func init() {
	sdks := []meergo.SDKSpec{
		{
			Code:       "dotnet",
			Label:      ".NET",
			Categories: meergo.CategorySDK,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users using .NET",
					Overview: dotnetOverview,
				},
			},
		},
		{
			Code:                "android",
			Label:               "Android",
			Categories:          meergo.CategorySDK,
			Strategies:          true,
			FallbackToRequestIP: true,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from an Android mobile device",
					Overview: androidOverview,
				},
			},
		},
		{
			Code:       "go",
			Label:      "Go",
			Categories: meergo.CategorySDK,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users using Go",
					Overview: goOverview,
				},
			},
		},
		{
			Code:       "java",
			Label:      "Java",
			Categories: meergo.CategorySDK,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users using Java",
					Overview: javaOverview,
				},
			},
		},
		{
			Code:                "javascript",
			Label:               "JavaScript",
			Categories:          meergo.CategorySDK | meergo.CategoryWebsite,
			Strategies:          true,
			FallbackToRequestIP: true,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users from a website using JavaScript",
					Overview: javaScriptOverview,
				},
			},
		},
		{
			Code:       "nodejs",
			Label:      "Node.js",
			Categories: meergo.CategorySDK,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users using Node.js",
					Overview: nodeOverview,
				},
			},
		},
		{
			Code:       "python",
			Label:      "Python",
			Categories: meergo.CategorySDK,
			Documentation: meergo.ConnectorDocumentation{
				Source: meergo.ConnectorRoleDocumentation{
					Summary:  "Import events and users using Python",
					Overview: pythonOverview,
				},
			},
		},
	}
	for _, sdk := range sdks {
		meergo.RegisterSDK(sdk, New)
	}
}

// New returns a new SDK connector instance.
func New(env *meergo.SDKEnv) (*SDK, error) {
	return &SDK{}, nil
}

type SDK struct{}
