{% extends "/layouts/doc.html" %}
{% macro Title string %}AI features{% end %}
{% Article %}

# AI features

You can interact with Meergo through AI. This feature allows you to query Meergo it for answers to questions like:

> *In the current workspace, analyze the users who interacted with the site over the past two weeks. What insights can you share about them?*

⚙️ Please note that, currently, using this feature requires some technical knowledge and it is highly experimental. It will be made more accessible, complete and stable in the future.

## Model compatibility

Meergo's AI features have been tested with **OpenAI's GPT-4.1 mini** model. Other models *might* work, but compatibility is not guaranteed as they haven't been tested. ⚠️ 🤖

## MCP server details

```plain
 Meergo instance     ←→     LLM Application     ←→      LLM 
(with MCP server)          (es. fast-agent)       (es. GPT, Gemini)   
                                 ↑
                                 ↓
                                User
```

Meergo exposes an **MCP (Model Context Protocol)** server to interact using your own LLM application.

Some details:

* The MCP server is exposed at the `/mcp` endpoint, e.g., `http://localhost:2022/mcp`. This endpoint is always exposed, and there's currently no configuration to enable, modify, or disable it.

* Authentication is done via an API key passed in the `Authorization` header, using the same format as Meergo’s HTTP APIs. Some LLM applications have a built-in support for this kind of MCP server authentication; for example, **fast-agent** implements this through the `--auth` CLI option.

* The API key must be associated to a workspace, which will be used for the interaction with Meergo. API keys not associated to a specific workspace are not allowed when interacting with the Meergo MCP server.

## Implemented Features

Meergo's exposed MCP server currently allows to:

* **Query the data warehouse** tables (events, users...) for analyzing data.
* Expose **user schema** and **event schema** detailed information, including information about the corresponding view and table on the warehouse, so the LLM can analyze them and provide information about them.
* Expose information about the **connections of the workspace**.
* Expose information about **user identities and Identity Resolution**, including its **last execution**.
* Expose **prompts** to suggest some **recommended uses cases**.

More will be implemented in the future.

## Create a read-only user for PostgreSQL

The MCP server requires a read-only user for accessing the data warehouse.

So, for example, to create an user called `foo_ro` to for the MCP server to access the warehouse called `warehouse`:

1. Open a PostgreSQL shell into `warehouse`, for example:
  
  ```bash
  psql warehouse
  ```

2. Create the `foo_ro` user:

  ```sql
  CREATE USER foo_ro WITH PASSWORD 'strong_password';
  ```

3. Grant the database `warehouse` connection to `foo_ro`:

  ```sql
  GRANT CONNECT ON DATABASE warehouse TO foo_ro;
  ```

4. Grant the access to the warehouse schema (eg. `public`):

  ```sql
  GRANT USAGE ON SCHEMA public TO foo_ro;
  ```

5. Let `foo_ro` run `SELECT` queries on every table in the schema:

  ```sql
  GRANT SELECT ON ALL TABLES IN SCHEMA public TO foo_ro;
  ```

6. Set default permissions for future tables created in the warehouse schema for `foo_ro`:

  ```sql
  ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO foo_ro;
  ```

## Example interaction with Meergo using **fast-agent**

This example assumes the use of **fast-agent** as the LLM application to connect to an **OpenAI** model and to Meergo's MCP.

Once you understand this example, you can adapt it to your needs (eg. another LL model, etc...) and use your preferred LLM application.

### Requirements

* A Open AI API key stored in the `OPENAI_API_KEY` environment variable.
* The [`fast-agent`](https://fast-agent.ai/) command installed locally.
* A Meergo MCP key associated to a specific workspace, stored in the `MEERGO_MCP_KEY` environment variable.
* A running instance of Meergo.
* Data warehouse MCP login settings must have been configured for the workspace.

### Starting the chat

```bash
fast-agent go --model openai --url=http://localhost:2022/mcp --auth=$MEERGO_MCP_KEY
```

By changing the command-line parameters and environment variables, you can configure **fast-agent** to connect to a different model or a different Meergo instance. See the [documentation for the `fast-agent go` command](https://fast-agent.ai/ref/go_command/) for more details.

## Known issues

### HTTPS certificate issue

When running locally, if the Meergo server uses HTTPS, the `fast-agent` command might be unable to verify certificates, depending on how they were installed. It's recommended to run Meergo over HTTP instead in case of problems.

### MCP resource errors

You might see errors like:

```plain
[mcp_agent.mcp.mcp_agent_client_session] send_request failed: resources not supported
[mcp_agent.mcp.mcp_aggregator.default] Failed to list_resources '' on server 'localhost_2022_mcp': resources not supported
[mcp_agent.mcp.mcp_aggregator.default] Error fetching resources from localhost_2022_mcp: resources not supported
```

when interacting via `fast-agent`. These errors don’t seem to affect usability and will be resolved in the future.

### Invalid function name error

You may encounter an error like this:

```plain
[mcp_agent.llm.augmented_llm] API Error: 400 - * 
GenerateContentRequest.tools[0].function_declarations[0].name: Invalid function name. Must start with a letter 
or an underscore. Must be alphameric (a-z, A-Z, 0-9), underscores (_), dots (.) or dashes (-), with a maximum 
length of 64.

  Finished       | FastAgent CLI     / Elapsed Time 00:00:08

Provider Configuration Error:
API Error: 400

Details:
* GenerateContentRequest.tools[0].function_declarations[0].name: Invalid function name. Must start with a 
letter or an underscore. Must be alphameric (a-z, A-Z, 0-9), underscores (_), dots (.) or dashes (-), with a 
maximum length of 64.
```

This depends on the fact that in the interaction between the server and the MCP client, a prefix is added to the tool names that makes those tools invalid (for example `127_0_0_1_2022_mcp-user-schema`). It's not clear why this happens.

This is resolved by having **fast-agent** connect to `localhost` instead of `127.0.0.1`. The interaction example shown above already takes this into account.
