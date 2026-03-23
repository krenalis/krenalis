// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var prompts = []server.ServerPrompt{
	simplePrompt(
		"describe-profile-schema",
		"Get a better understanding of the profile schema",
		"Retrieve the current profile schema associated with the workspace and provide a high-level description of it,"+
			" indicating which parts of it could be improved."+
			" Also explain the relationship between the properties of the profile schema and the columns of the corresponding 'profiles' view on the data warehouse.",
	),
	simplePrompt(
		"describe-event-schema",
		"Get a better understanding of the event schema",
		"Retrieve the event schema and provide a high-level description of it."+
			" Also explain the relationship between the properties of the event schema and the columns of the corresponding 'events' view on the data warehouse.",
	),
	simplePrompt(
		"workspace-connections",
		"Retrieve information about workspace connections",
		"List the connections currently present in the workspace."+
			" For each, display a brief description summarizing the connection type and the number of pipelines currently present."+
			" Also indicate which of them are currently configured and which are not, depending on the number of pipelines.",
	),
	simplePrompt(
		"how-pipelines-connections-identity-resolutions-work",
		"Explain how connections, pipelines and Identity Resolution work",
		"Explain to me how pipelines, connections, and identity resolution work. Provide concrete examples using data from my workspace, if possible.",
	),
	simplePrompt(
		"i-am-new-to-krenalis",
		"Get an overview on how Krenalis works",
		"I'm new to Krenalis and don't know how it works or what it does. Can you explain how it works, what it allows me to do, and what are the key concepts I need to work with?",
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
