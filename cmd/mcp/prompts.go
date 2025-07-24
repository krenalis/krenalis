//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var prompts = []server.ServerPrompt{

	// Prompt for getting a description of the user schema.
	{
		Prompt: mcp.NewPrompt("describe-user-schema", mcp.WithPromptDescription("Get a better understanding of the user schema")),
		Handler: func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			pr := mcp.NewGetPromptResult("Get a better understanding of the user schema", []mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent(
						"Retrieve the current user schema associated with the workspace and provide a high-level description of it,"+
							" indicating which parts of it could be improved."+
							" Also explain the relationship between the properties of the user schema and the columns of the corresponding 'users' view on the data warehouse.",
					),
				),
			})
			return pr, nil
		},
	},

	// Prompt for getting a description of the event schema.
	{
		Prompt: mcp.NewPrompt("describe-event-schema", mcp.WithPromptDescription("Get a better understanding of the event schema")),
		Handler: func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			pr := mcp.NewGetPromptResult("Get a better understanding of the event schema", []mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent("Retrieve the event schema and provide a high-level description of it."+
						" Also explain the relationship between the properties of the event schema and the columns of the corresponding 'events' table on the data warehouse.",
					),
				),
			})
			return pr, nil
		},
	},
}
