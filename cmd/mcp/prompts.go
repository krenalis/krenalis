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
	simplePrompt(
		"describe-user-schema",
		"Get a better understanding of the user schema",
		"Retrieve the current user schema associated with the workspace and provide a high-level description of it,"+
			" indicating which parts of it could be improved."+
			" Also explain the relationship between the properties of the user schema and the columns of the corresponding 'users' view on the data warehouse.",
	),
	simplePrompt(
		"describe-event-schema",
		"Get a better understanding of the event schema",
		"Retrieve the event schema and provide a high-level description of it."+
			" Also explain the relationship between the properties of the event schema and the columns of the corresponding 'events' table on the data warehouse.",
	),
	simplePrompt(
		"workspace-connections",
		"Retrieve information about workspace connections",
		"List the connections currently present in the workspace."+
			" For each, display a brief description summarizing the connection type and the number of actions currently present."+
			" Also indicate which of them are currently configured and which are not, depending on the number of actions.",
	),
	simplePrompt(
		"how-actions-connections-identity-resolutions-work",
		"Explain how connections, actions and Identity Resolution work",
		"Explain to me how actions, connections, and identity resolution work. Provide concrete examples using data from my workspace, if possible.",
	),
	simplePrompt(
		"i-am-new-to-meergo",
		"Get an overview on how Meergo works",
		"I'm new to Meergo and don't know how it works or what it does. Can you explain how it works, what it allows me to do, and what are the key concepts I need to work with?",
	),
}

// simplePrompt returns a simple MCP prompt which is based only on a name,
// description and the prompt itself.
func simplePrompt(name, description, prompt string) server.ServerPrompt {
	return server.ServerPrompt{
		Prompt: mcp.NewPrompt(name, mcp.WithPromptDescription(description)),
		Handler: func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			pr := mcp.NewGetPromptResult(description, []mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(prompt)),
			})
			return pr, nil
		},
	}
}
