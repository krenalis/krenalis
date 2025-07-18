{% extends "/layouts/doc.html" %}
{% macro Title string %}AI features{% end %}
{% Article %}

# AI features

You can interact with Meergo through AI.

> ⚙️ Currently, using this feature requires some technical knowledge and it is highly experimental. It will be made more accessible, complete and stable in the future.

```plain
 Meergo instance     ←→     LLM Application     ←→      LLM 
(with MCP server)          (es. fast-agent)       (es. GPT, Gemini)   
                                 ↑
                                 ↓
                                User
```

## MCP server details

Meergo exposes an MCP (Model Context Protocol) server to interact using your own LLM application.

Some details:

* The MCP server is exposed at the `/mcp` endpoint, e.g., `http://localhost:9090/mcp`. This endpoint is always exposed, and there's currently no configuration to enable, modify, or disable it.

* Authentication is done via an API key passed in the `Authorization` header, using the same format as Meergo’s HTTP APIs. Some LLM applications have a built-in support for this kind of MCP server authentication; for example, **fast-agent** implements this through the `--auth` CLI option.

* The API key must be associated to a workspace, which will be used for the interaction with Meergo. API keys not associated to a specific workspace are not allowed when interacting with the Meergo MCP server.

## Implemented Features

Meergo's exposed MCP server currently allows to:

* **Query the data warehouse** tables (events, users...) for analyzing data.
* Expose the **user schema**, so the LLM can analyze it and provide information about it.
* Expose the **event schema**, so the LLM can analyze it and provide information about it.

More will be implemented in the future.

## Example interaction with Meergo using **fast-agent**

This example assumes the use of **fast-agent** as the LLM application to connect to a **Gemini** model and to Meergo's MCP.

Once you understand this example, you can adapt it to your needs (eg. another LL model, etc...) and use your preferred LLM application.

### Requirements

* A Google API key stored in the `GOOGLE_API_KEY` environment variable.
* The [`fast-agent`](https://fast-agent.ai/) command installed locally.
* A Meergo API key associated to a specific workspace, stored in the `MEERGO_API_KEY` environment variable.
* A running instance of Meergo.

### Starting the chat

```bash
fast-agent go --model google --url=http://localhost:9090/mcp --auth=$MEERGO_API_KEY
```

By changing the command-line parameters and environment variables, you can configure **fast-agent** to connect to a different model or a different Meergo instance. See the [documentation for the `fast-agent go` command](https://fast-agent.ai/ref/go_command/) for more details.

## Known issues

### HTTPS certificate issue

When running locally, if the Meergo server uses HTTPS, the `fast-agent` command might be unable to verify certificates, depending on how they were installed. It's recommended to run Meergo over HTTP instead in case of problems.

### MCP resource errors

You might see errors like:

```plain
[mcp_agent.mcp.mcp_agent_client_session] send_request failed: resources not supported
[mcp_agent.mcp.mcp_aggregator.default] Failed to list_resources '' on server 'localhost_9090_mcp': resources not supported
[mcp_agent.mcp.mcp_aggregator.default] Error fetching resources from localhost_9090_mcp: resources not supported
```

when interacting via `fast-agent`. These errors don’t seem to affect usability and will be resolved in the future.

### Invalid function name error

You may encounter an error like this:

```plain
[mcp_agent.llm.augmented_llm] Google API Error: 400 - * 
GenerateContentRequest.tools[0].function_declarations[0].name: Invalid function name. Must start with a letter 
or an underscore. Must be alphameric (a-z, A-Z, 0-9), underscores (_), dots (.) or dashes (-), with a maximum 
length of 64.

  Finished       | FastAgent CLI     / Elapsed Time 00:00:08

Provider Configuration Error:
Google API Error: 400

Details:
* GenerateContentRequest.tools[0].function_declarations[0].name: Invalid function name. Must start with a 
letter or an underscore. Must be alphameric (a-z, A-Z, 0-9), underscores (_), dots (.) or dashes (-), with a 
maximum length of 64.
```

This depends on the fact that in the interaction between the server and the MCP client, a prefix is added to the tool names that makes those tools invalid (for example `127_0_0_1_9090_mcp-user-schema`). It's not clear why this happens.

This is resolved by having **fast-agent** connect to `localhost` instead of `127.0.0.1`. The interaction example shown above already takes this into account.
