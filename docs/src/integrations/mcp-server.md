{% extends "/layouts/doc.html" %}
{% macro Title string %}MCP server{% end %}
{% Article %}

# MCP Server

Meergo includes an experimental **Model Context Protocol (MCP)** server that allows you to query the unified customer data that Meergo stored in your data warehouse using natural language.

Once enabled, you can ask Meergo questions like:

    In the current workspace, analyze the users who interacted with the site over the past two weeks. What insights can you share about them?

## Enabling MCP access

The Meergo MCP server uses a dedicated **read-only** account to access your data warehouse. This account is **separate** from the one used by other Meergo features and must be **explicitly enabled**.
Currently, only the PostgreSQL data warehouse is supported; Snowflake is coming soon. 

## Before you begin

To use the Meergo MCP server, make sure you have:

* A Meergo workspace linked to a PostgreSQL data warehouse
* An **MCP client** (for example, [fast-agent](https://fast-agent.ai/))
* Access to an **AI model** (for example, an OpenAI API key)

### 1. Create a read-only warehouse user

Create a PostgreSQL read-only user for the MCP server.
In this example, we'll create a user named `meergo_ro` to connect to a warehouse named `customer_dw`.

1. Open a PostgreSQL shell:

   ```sh
   $ psql customer_dw
   ```

2. Create the `meergo_ro` user:

   ```sql
   CREATE USER meergo_ro WITH PASSWORD 'strong_password';
   ```

3. Allow `meergo_ro` to connect to the `customer_dw` database:

   ```sql
   GRANT CONNECT ON DATABASE customer_dw TO meergo_ro;
   ```

4. Grant access to the schema (for example, `public`):

   ```sql
   GRANT USAGE ON SCHEMA public TO meergo_ro;
   ```

5. Allow `SELECT` on all existing tables in the schema:

   ```sql
   GRANT SELECT ON ALL TABLES IN SCHEMA public TO meergo_ro;
   ```

6. Set default permissions for any future tables:

   ```sql
   ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO meergo_ro;
   ```

### 2. Enable MCP in the Meergo Admin console

1. Log in to the **Meergo Admin Console**.
2. From the left menu, go to **Customization → Data Warehouse**.
3. Click **Modify...**
4. Under **MCP credentials**, select **Grant read-only access to the MCP server**.
5. Enter the read-only database credentials you created.
6. Click **Save**.

Once validated, proceed to create your access key.

### 3. Create an MCP access key

1. Click your **account icon** (top-right corner).
2. Select **API and MCP Server**.
3. Click **Add new MCP key**.
4. In the **Create a new MCP key** window:

    * Enter a **name** for your key to identify it later.
    * Choose the **workspace** where you want to enable MCP access.
5. Click **Add**.
6. In the **Your new MCP key** window, **copy the key immediately** — it won't be shown again.

### 4. Start a chat with fast-agent

1. Set your MCP key as an environment variable:

   ```sh
   $ export MEERGO_MCP_KEY="your_mcp_key_here"
   ```

2. Start `fast-agent`, replacing host and port with your Meergo instance details:

   ```sh
   $ fast-agent go --model openai --url=http://127.0.0.1:2022/mcp --auth=$MEERGO_MCP_KEY
   ```

3. Enter a prompt and start chatting with your Meergo instance.

## Example prompt

Once connected, try asking:

    Show me a summary of new users who joined last week and how many converted to paying customers
The AI model will query the MCP server and return insights directly from your warehouse.

## Known issues

### HTTPS certificate issue

When running locally, if the Meergo server uses HTTPS, the `fast-agent` command might be unable to verify certificates, depending on how they were installed. It's recommended to run Meergo over HTTP instead in case of problems.

### MCP resource errors

You might see errors like:

```
[mcp_agent.mcp.mcp_agent_client_session] send_request failed: resources not supported
[mcp_agent.mcp.mcp_aggregator.default] Failed to list_resources '' on server 'localhost_2022_mcp': resources not supported
[mcp_agent.mcp.mcp_aggregator.default] Error fetching resources from localhost_2022_mcp: resources not supported
```

when interacting via `fast-agent`. These errors don't seem to affect usability and will be resolved in the future.

### Invalid function name error

You may encounter an error like this:

```
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
